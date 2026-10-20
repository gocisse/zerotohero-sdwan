package sim

import "fmt"

// Firewall evaluation options the reader can choose between.
const (
	FwFirstMatch = "first_match"
	FwLastMatch  = "last_match"
)

// TraceAssumptions are the unverified defaults of the routing model plus the three the flow tracer adds. They arrive
// in every request, are never stored, and are echoed in every output with their assumption number.
type TraceAssumptions struct {
	RoutingAssumptions
	PolicyEdgeFirst       bool   `json:"policy_edge_first"`
	FirewallEvaluation    string `json:"fw_evaluation"`
	FirewallDefaultAction string `json:"fw_default_action"`
}

// DefaultTraceAssumptions returns the model's defaults (each one an assumption).
func DefaultTraceAssumptions() TraceAssumptions {
	return TraceAssumptions{RoutingAssumptions: DefaultRoutingAssumptions(), PolicyEdgeFirst: true, // assumption 3
		FirewallEvaluation: FwFirstMatch, FirewallDefaultAction: FwAllow} // assumptions 85 and 86
}

// Validate rejects values outside the choices.
func (a TraceAssumptions) Validate() error {
	if err := a.RoutingAssumptions.Validate(); err != nil {
		return err
	}
	if a.FirewallEvaluation != FwFirstMatch && a.FirewallEvaluation != FwLastMatch {
		return &AssumptionError{fmt.Sprintf("fw_evaluation must be %q or %q", FwFirstMatch, FwLastMatch)}
	}
	if a.FirewallDefaultAction != FwAllow && a.FirewallDefaultAction != FwDrop {
		return &AssumptionError{fmt.Sprintf("fw_default_action must be %q or %q", FwAllow, FwDrop)}
	}
	return nil
}

// Describe lists every selectable assumption (routing and trace), with the ones the result used marked.
func (a TraceAssumptions) Describe(used map[string]bool) []AssumptionSwitch {
	out := a.RoutingAssumptions.Describe(used)
	def := DefaultTraceAssumptions()
	add := func(key, label, kind string, choices []string, rule string, v, d any) {
		r := RuleByID(rule)
		out = append(out, AssumptionSwitch{Key: key, Label: label, Kind: kind, Choices: choices, Value: v, Default: d, Changed: v != d,
			Assumption: r.Assumption, Text: r.Text, Basis: r.Basis(), Used: used[key]})
	}
	add("policy_edge_first", "Business policy: Edge rules above Profile rules", "bool", nil, RA3EdgeAbove, a.PolicyEdgeFirst, def.PolicyEdgeFirst)
	add("fw_evaluation", "Firewall rule evaluation order", "choice", []string{FwFirstMatch, FwLastMatch}, RA85FwEvaluation, a.FirewallEvaluation, def.FirewallEvaluation)
	add("fw_default_action", "Firewall action when no rule decides", "choice", []string{FwAllow, FwDrop}, RA86FwDefault, a.FirewallDefaultAction, def.FirewallDefaultAction)
	return out
}

// Flip returns a copy with one switch toggled or cycled; ok is false for an unknown key.
func (a TraceAssumptions) Flip(key string) (TraceAssumptions, bool) {
	b := a
	switch key {
	case "policy_edge_first":
		b.PolicyEdgeFirst = !b.PolicyEdgeFirst
		return b, true
	case "fw_evaluation":
		if b.FirewallEvaluation == FwFirstMatch {
			b.FirewallEvaluation = FwLastMatch
		} else {
			b.FirewallEvaluation = FwFirstMatch
		}
		return b, true
	case "fw_default_action":
		if b.FirewallDefaultAction == FwAllow {
			b.FirewallDefaultAction = FwDrop
		} else {
			b.FirewallDefaultAction = FwAllow
		}
		return b, true
	}
	r, ok := a.RoutingAssumptions.Flip(key)
	b.RoutingAssumptions = r
	return b, ok
}

// switchKeys lists every switch key (routing first).
func (a TraceAssumptions) switchKeys() []string {
	var keys []string
	for _, d := range switchDefs {
		keys = append(keys, d.key)
	}
	return append(keys, "policy_edge_first", "fw_evaluation", "fw_default_action")
}

func (a TraceAssumptions) value(key string) any {
	switch key {
	case "policy_edge_first":
		return a.PolicyEdgeFirst
	case "fw_evaluation":
		return a.FirewallEvaluation
	case "fw_default_action":
		return a.FirewallDefaultAction
	}
	for _, d := range switchDefs {
		if d.key == key {
			return d.get(a.RoutingAssumptions)
		}
	}
	return nil
}

// Set sets one switch to a value (a bool or a string), for labs that start from a different assumption.
func (a TraceAssumptions) Set(key string, v any) (TraceAssumptions, error) {
	cur := a.value(key)
	if cur == nil {
		return a, fmt.Errorf("unknown assumption switch %q", key)
	}
	b := a
	for i := 0; i < 3 && b.value(key) != v; i++ {
		b, _ = b.Flip(key)
	}
	if b.value(key) != v {
		return a, fmt.Errorf("switch %q cannot be set to %v", key, v)
	}
	return b, b.Validate()
}
