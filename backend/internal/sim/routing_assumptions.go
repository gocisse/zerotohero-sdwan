package sim

import (
	"errors"
	"fmt"
)

// Filter evaluation orders and Logical ID directions the reader can choose between.
const (
	FilterOrderFirst = "first_match"
	FilterOrderLast  = "last_match"
	TieLower         = "lower"
	TieHigher        = "higher"
)

// RoutingAssumptions are the unverified defaults of the routing model. They arrive in every request, are never
// stored, and are echoed in every output next to the assumption number that explains them. A topology that sets
// a value itself (for example OSPF redistribute.overlay_prefixes) wins over the value here.
type RoutingAssumptions struct {
	OSPFOverlayPrefixes bool   `json:"ospf_overlay_prefixes"`
	OSPFRedistributeBGP bool   `json:"ospf_redistribute_bgp"`
	BGPOverlayPrefix    bool   `json:"bgp_overlay_prefix"`
	BGPConnected        bool   `json:"bgp_connected"`
	BGPRedistributeOSPF bool   `json:"bgp_redistribute_ospf"`
	FilterDefaultAction string `json:"filter_default_action"`
	FilterOrder         string `json:"filter_order"`
	AdvertiseStatic     bool   `json:"advertise_static"`
	AdvertiseConnected  bool   `json:"advertise_connected"`
	LogicalIDTieBreak   string `json:"logical_id_tiebreak"`
}

// DefaultRoutingAssumptions returns the documented defaults of the model (each one is an assumption).
func DefaultRoutingAssumptions() RoutingAssumptions {
	return RoutingAssumptions{
		// assumption 22: all five redistribution switches are off
		FilterDefaultAction: FilterPermit, // assumption 23
		FilterOrder:         FilterOrderFirst,
		AdvertiseStatic:     true, // assumption 69
		AdvertiseConnected:  true,
		LogicalIDTieBreak:   TieLower, // assumption 74
	}
}

// AssumptionError is returned for assumption values outside the choices.
type AssumptionError struct{ Message string }

func (e *AssumptionError) Error() string { return e.Message }

// IsAssumptionError reports whether err is an AssumptionError.
func IsAssumptionError(err error) bool {
	var ae *AssumptionError
	return errors.As(err, &ae)
}

// Validate rejects values outside the choices.
func (a RoutingAssumptions) Validate() error {
	if a.FilterDefaultAction != FilterPermit && a.FilterDefaultAction != FilterDeny {
		return &AssumptionError{fmt.Sprintf("filter_default_action must be %q or %q", FilterPermit, FilterDeny)}
	}
	if a.FilterOrder != FilterOrderFirst && a.FilterOrder != FilterOrderLast {
		return &AssumptionError{fmt.Sprintf("filter_order must be %q or %q", FilterOrderFirst, FilterOrderLast)}
	}
	if a.LogicalIDTieBreak != TieLower && a.LogicalIDTieBreak != TieHigher {
		return &AssumptionError{fmt.Sprintf("logical_id_tiebreak must be %q or %q", TieLower, TieHigher)}
	}
	return nil
}

// AssumptionSwitch describes one selectable assumption for the UI and for every output.
type AssumptionSwitch struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Kind       string   `json:"kind"` // bool or choice
	Choices    []string `json:"choices,omitempty"`
	Value      any      `json:"value"`
	Default    any      `json:"default"`
	Changed    bool     `json:"changed"`
	Assumption int      `json:"assumption"`
	Text       string   `json:"text"`
	Basis      Basis    `json:"basis"`
	// Used is true when a result actually took this value (the topology left it unset).
	Used bool `json:"used"`
}

type switchDef struct {
	key, label, kind string
	choices          []string
	rule             string
	get              func(RoutingAssumptions) any
	flip             func(*RoutingAssumptions)
}

var switchDefs = []switchDef{
	{"ospf_overlay_prefixes", "OSPF: Overlay Prefixes checkbox (overlay routes into OSPF)", "bool", nil, RA22Redistribution, // rule: assume-22
		func(a RoutingAssumptions) any { return a.OSPFOverlayPrefixes }, func(a *RoutingAssumptions) { a.OSPFOverlayPrefixes = !a.OSPFOverlayPrefixes }},
	{"ospf_redistribute_bgp", "OSPF: BGP option (BGP routes into OSPF)", "bool", nil, RA22Redistribution, // rule: assume-22
		func(a RoutingAssumptions) any { return a.OSPFRedistributeBGP }, func(a *RoutingAssumptions) { a.OSPFRedistributeBGP = !a.OSPFRedistributeBGP }},
	{"bgp_overlay_prefix", "BGP: Overlay Prefix checkbox (overlay routes into BGP)", "bool", nil, RA22Redistribution, // rule: assume-22
		func(a RoutingAssumptions) any { return a.BGPOverlayPrefix }, func(a *RoutingAssumptions) { a.BGPOverlayPrefix = !a.BGPOverlayPrefix }},
	{"bgp_connected", "BGP: Connected Routes checkbox", "bool", nil, RA22Redistribution, // rule: assume-22
		func(a RoutingAssumptions) any { return a.BGPConnected }, func(a *RoutingAssumptions) { a.BGPConnected = !a.BGPConnected }},
	{"bgp_redistribute_ospf", "BGP: OSPF checkbox (OSPF routes into BGP)", "bool", nil, RA22Redistribution, // rule: assume-22
		func(a RoutingAssumptions) any { return a.BGPRedistributeOSPF }, func(a *RoutingAssumptions) { a.BGPRedistributeOSPF = !a.BGPRedistributeOSPF }},
	{"filter_default_action", "Filter default action when no rule matches", "choice", []string{FilterPermit, FilterDeny}, RA23FilterDefault, // rule: assume-23
		func(a RoutingAssumptions) any { return a.FilterDefaultAction }, func(a *RoutingAssumptions) {
			if a.FilterDefaultAction == FilterPermit {
				a.FilterDefaultAction = FilterDeny
			} else {
				a.FilterDefaultAction = FilterPermit
			}
		}},
	{"filter_order", "Filter rule evaluation order", "choice", []string{FilterOrderFirst, FilterOrderLast}, RA23FilterDefault, // rule: assume-23
		func(a RoutingAssumptions) any { return a.FilterOrder }, func(a *RoutingAssumptions) {
			if a.FilterOrder == FilterOrderFirst {
				a.FilterOrder = FilterOrderLast
			} else {
				a.FilterOrder = FilterOrderFirst
			}
		}},
	{"advertise_static", "Global Advertise Flag: static routes", "bool", nil, RA69AdvertiseDefaults, // rule: assume-69
		func(a RoutingAssumptions) any { return a.AdvertiseStatic }, func(a *RoutingAssumptions) { a.AdvertiseStatic = !a.AdvertiseStatic }},
	{"advertise_connected", "Global Advertise Flag: connected routes", "bool", nil, RA69AdvertiseDefaults, // rule: assume-69
		func(a RoutingAssumptions) any { return a.AdvertiseConnected }, func(a *RoutingAssumptions) { a.AdvertiseConnected = !a.AdvertiseConnected }},
	{"logical_id_tiebreak", "Logical ID tie-break direction", "choice", []string{TieLower, TieHigher}, RA74LogicalIDDir, // rule: assume-74
		func(a RoutingAssumptions) any { return a.LogicalIDTieBreak }, func(a *RoutingAssumptions) {
			if a.LogicalIDTieBreak == TieLower {
				a.LogicalIDTieBreak = TieHigher
			} else {
				a.LogicalIDTieBreak = TieLower
			}
		}},
}

// Describe lists every selectable assumption with its value, default and evidence. used says which ones a result
// actually relied on.
func (a RoutingAssumptions) Describe(used map[string]bool) []AssumptionSwitch {
	def := DefaultRoutingAssumptions()
	out := make([]AssumptionSwitch, 0, len(switchDefs))
	for _, d := range switchDefs {
		r := RuleByID(d.rule)
		out = append(out, AssumptionSwitch{Key: d.key, Label: d.label, Kind: d.kind, Choices: d.choices, Value: d.get(a), Default: d.get(def),
			Changed: d.get(a) != d.get(def), Assumption: r.Assumption, Text: r.Text, Basis: r.Basis(), Used: used[d.key]})
	}
	return out
}

// Flip returns a copy with one switch toggled (or cycled); ok is false for an unknown key.
func (a RoutingAssumptions) Flip(key string) (RoutingAssumptions, bool) {
	for _, d := range switchDefs {
		if d.key == key {
			b := a
			d.flip(&b)
			return b, true
		}
	}
	return a, false
}
