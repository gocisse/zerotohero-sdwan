package sim

import (
	"fmt"
	"strconv"
	"strings"
)

// Lab task types.
const (
	TaskInspect  = "inspect"
	TaskDiagnose = "diagnose"
	TaskFix      = "fix"
	TaskPredict  = "predict"
)

// Lab is one practice scenario (content/labs/*.yaml). The fault is an overlay on a teaching topology: conditions,
// assumption values and config patches applied to a copy; the authored topology file is never changed.
type Lab struct {
	ID               string      `yaml:"id" json:"id"`
	Title            string      `yaml:"title" json:"title"`
	Summary          string      `yaml:"summary" json:"summary"`
	Difficulty       int         `yaml:"difficulty" json:"difficulty"`
	EstimatedMinutes int         `yaml:"estimated_minutes" json:"estimated_minutes"`
	Topology         string      `yaml:"topology" json:"topology"`
	Goals            []string    `yaml:"goals" json:"goals"`
	Screens          []LabScreen `yaml:"screens" json:"screens"`
	Fault            LabFault    `yaml:"fault" json:"fault"`
	Symptom          string      `yaml:"symptom" json:"symptom"`
	// Flow is the flow in question; probes and the runner's recomputed trace use it unless a probe names its own.
	Flow *FlowSpec `yaml:"flow" json:"flow,omitempty"`
	// RelyOn lists the assumption numbers the correct answers rely on; the symptom text must say so.
	RelyOn []int     `yaml:"relies_on_assumptions" json:"relies_on_assumptions,omitempty"`
	Tasks  []LabTask `yaml:"tasks" json:"tasks"`
	// Debrief maps each finding to where it would be found in the real VCO. It names the root causes, so the API serves it
	// only after the lab is completed or skipped.
	Debrief []LabDebrief `yaml:"debrief" json:"-"`
	Hints   []LabHint    `yaml:"hints" json:"hints"`
	Fixes   []LabFix     `yaml:"fixes" json:"fixes,omitempty"`
}

// LabScreen names a mock VCO screen to inspect; Path is relative to /sim/<topology>/.
type LabScreen struct {
	Label string `yaml:"label" json:"label"`
	Path  string `yaml:"path" json:"path"`
}

// LabFault is the injected fault.
type LabFault struct {
	Conditions  Conditions     `yaml:"conditions" json:"conditions"`
	Assumptions map[string]any `yaml:"assumptions" json:"assumptions,omitempty"`
	Patches     []Patch        `yaml:"patches" json:"patches,omitempty"`
}

// LabTask is one step. inspect, diagnose and predict are multiple choice: the correct choice is the one whose Value
// equals what the engine computes for the probe. fix is checked by recomputing the state with the applied fixes and
// comparing the probe with Equals.
type LabTask struct {
	ID          string      `yaml:"id" json:"id"`
	Type        string      `yaml:"type" json:"type"`
	Prompt      string      `yaml:"prompt" json:"prompt"`
	Probe       *Probe      `yaml:"probe" json:"-"`
	Choices     []LabChoice `yaml:"choices" json:"-"`
	Equals      string      `yaml:"equals" json:"-"`
	Explanation string      `yaml:"explanation" json:"-"`
}

// LabChoice is one answer; Value is the engine value it stands for (never sent before the task is checked).
type LabChoice struct {
	ID    string `yaml:"id" json:"id"`
	Text  string `yaml:"text" json:"text"`
	Value string `yaml:"value" json:"-"`
}

// LabDebrief is one line of the debrief: a finding and the VCO location the guides name for it. Where must be wording
// that appears in a VERIFIED or OBSERVED fact-sheet row (the content validator checks); Unverified marks a location the
// guides do not confirm, and then Note says so.
type LabDebrief struct {
	Finding    string `yaml:"finding" json:"finding"`
	Where      string `yaml:"where" json:"where"`
	Unverified bool   `yaml:"unverified" json:"unverified"`
	Note       string `yaml:"note" json:"note,omitempty"`
}

// LabHint is revealed on request, in order.
type LabHint struct {
	Task string `yaml:"task" json:"task,omitempty"`
	Text string `yaml:"text" json:"text"`
}

// LabFix is a patch the learner may apply. Allowed fixes include wrong ones: choosing is the exercise.
type LabFix struct {
	ID          string  `yaml:"id" json:"id"`
	Title       string  `yaml:"title" json:"title"`
	Description string  `yaml:"description" json:"description"`
	Patches     []Patch `yaml:"patches" json:"patches"`
}

// Probe kinds. Each reads one value from the engine's own output.
const (
	PrTraceVerdict   = "trace_verdict"
	PrTraceBlocked   = "trace_blocked_stage"
	PrTracePathKind  = "trace_path_kind"
	PrTracePathVia   = "trace_path_via"
	PrTraceLink      = "trace_steered_link"
	PrTraceRule      = "trace_matched_rule"
	PrTraceRuleOwner = "trace_matched_owner"
	PrTraceMTUMin    = "trace_mtu_min"
	PrTraceMTULimit  = "trace_mtu_limiting"
	PrTraceMTUFit    = "trace_mtu_fit"
	PrRejectedCode   = "path_rejected_code"
	PrReachFailing   = "reach_first_failing"
	PrReachOutcome   = "reach_outcome"
	PrRouteField     = "route_field"
	PrTunnelState    = "tunnel_state"
	PrTunnelReason   = "tunnel_reason"
	PrTimelinePeer   = "timeline_peer_hub"
	PrTimelineLocal  = "timeline_local_hub"
	// PrEffectiveValue and PrEffectiveSource read one device setting of one Edge from EffectiveConfig (the value, or the
	// layer it comes from: profile, edge-override, default...). They add no behaviour: the probe only reads existing output.
	PrEffectiveValue  = "effective_value"
	PrEffectiveSource = "effective_source"
)

// Probe names a value in the engine's output.
type Probe struct {
	Kind    string    `yaml:"kind" json:"kind"`
	Flow    *FlowSpec `yaml:"flow" json:"flow,omitempty"`
	Edge    string    `yaml:"edge" json:"edge,omitempty"`
	Prefix  string    `yaml:"prefix" json:"prefix,omitempty"`
	Rank    int       `yaml:"rank" json:"rank,omitempty"`
	Field   string    `yaml:"field" json:"field,omitempty"`
	Tunnel  string    `yaml:"tunnel" json:"tunnel,omitempty"`
	Size    int       `yaml:"size" json:"size,omitempty"`
	Segment string    `yaml:"segment" json:"segment,omitempty"`
	// Events script a timeline (timeline_* probes): it always starts from a healthy network, so the lab's fault
	// conditions are not applied and the events are the fault.
	Events []Event `yaml:"events" json:"events,omitempty"`
}

// ProbeResult is a probe's value with the assumptions its evidence cites.
type ProbeResult struct {
	Value       string `json:"value"`
	Assumptions []int  `json:"assumptions"`
	NotModelled bool   `json:"not_modelled"`
	Detail      string `json:"detail"`
}

// LabState is a lab's topology with the fault and the applied fixes, plus the conditions and assumptions.
type LabState struct {
	Topo *Topology
	Cond Conditions
	As   TraceAssumptions
	Flow *FlowSpec
}

func (l *Lab) fix(id string) *LabFix {
	for i := range l.Fixes {
		if l.Fixes[i].ID == id {
			return &l.Fixes[i]
		}
	}
	return nil
}

// State computes the lab's state from the authored topology: fault patches, then the applied fixes in order.
func (l *Lab) State(base *Topology, applied []string) (*LabState, error) {
	patches := append([]Patch{}, l.Fault.Patches...)
	seen := map[string]bool{}
	for _, id := range applied {
		f := l.fix(id)
		if f == nil {
			return nil, &PatchError{fmt.Sprintf("unknown fix %q", id)}
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		patches = append(patches, f.Patches...)
	}
	topo, err := ApplyPatches(base, patches)
	if err != nil {
		return nil, err
	}
	as := DefaultTraceAssumptions()
	for k, v := range l.Fault.Assumptions {
		if as, err = as.Set(k, v); err != nil {
			return nil, err
		}
	}
	return &LabState{Topo: topo, Cond: l.Fault.Conditions, As: as, Flow: l.Flow}, nil
}

func assumptionNumbers(bs ...Basis) []int {
	var out []int
	seen := map[int]bool{}
	for _, b := range bs {
		if b.Status != StatusAssumption || !strings.HasPrefix(b.Rule, "assume-") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(b.Rule, "assume-"))
		if err == nil && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func (s *LabState) flowFor(p *Probe) (FlowSpec, error) {
	switch {
	case p.Flow != nil:
		return *p.Flow, nil
	case s.Flow != nil:
		return *s.Flow, nil
	}
	return FlowSpec{}, fmt.Errorf("the probe %s needs a flow and neither the probe nor the lab has one", p.Kind)
}

// Eval reads the probe's value from the state.
func (s *LabState) Eval(p *Probe) (ProbeResult, error) {
	switch p.Kind {
	case PrRouteField:
		return s.evalRoute(p)
	case PrTunnelState, PrTunnelReason:
		return s.evalTunnel(p)
	case PrEffectiveValue, PrEffectiveSource:
		return s.evalEffective(p)
	case PrTimelinePeer, PrTimelineLocal:
		return s.evalTimeline(p)
	case PrReachFailing, PrReachOutcome:
		f, err := s.flowFor(p)
		if err != nil {
			return ProbeResult{}, err
		}
		rr, err := Reachability(s.Topo, s.Cond, s.As.RoutingAssumptions, ReachQuery{Source: f.Source, Destination: f.Destination, Segment: f.Segment}, nil)
		if err != nil {
			return ProbeResult{}, err
		}
		res := ProbeResult{Value: rr.FirstFailing, Detail: rr.Reason, NotModelled: rr.Outcome == OutcomeNotModelled}
		if p.Kind == PrReachOutcome {
			res.Value = rr.Outcome
		}
		if res.Value == "" {
			res.Value = "none"
		}
		for _, st := range rr.Stages {
			if st.ID == rr.FirstFailing || rr.FirstFailing == "" {
				res.Assumptions = append(res.Assumptions, assumptionNumbers(append([]Basis{st.Basis}, st.Also...)...)...)
			}
		}
		return res, nil
	}
	f, err := s.flowFor(p)
	if err != nil {
		return ProbeResult{}, err
	}
	tr, err := traceFlowOpts(s.Topo, s.Cond, s.As, f, nil, p.Kind != PrTraceVerdict && p.Kind != PrTraceBlocked)
	if err != nil {
		return ProbeResult{}, err
	}
	return evalTrace(p, tr)
}

func stageByID(tr *TraceResult, id string) *TraceStage {
	for i := range tr.Stages {
		if tr.Stages[i].ID == id {
			return &tr.Stages[i]
		}
	}
	return nil
}

func evalTrace(p *Probe, tr *TraceResult) (ProbeResult, error) {
	res := ProbeResult{NotModelled: tr.Verdict == VerdictNotModelled, Detail: tr.Summary}
	cite := func(st *TraceStage) {
		if st != nil {
			res.Assumptions = append(res.Assumptions, assumptionNumbers(append([]Basis{st.Basis}, st.Also...)...)...)
		}
	}
	path, policy, mtu := stageByID(tr, TrPath), stageByID(tr, TrPolicy), stageByID(tr, TrMTU)
	switch p.Kind {
	case PrTraceVerdict:
		res.Value = tr.Verdict
	case PrTraceBlocked:
		res.Value = tr.BlockedStage
		if res.Value == "" {
			res.Value = "none"
		}
		cite(stageByID(tr, tr.BlockedStage))
	case PrTracePathKind, PrTracePathVia:
		res.Value = "none"
		if path.Path != nil && path.Path.Chosen != nil {
			res.Value = path.Path.Chosen.Kind
			if p.Kind == PrTracePathVia {
				res.Value = path.Path.Chosen.Via
			}
			res.Assumptions = append(res.Assumptions, assumptionNumbers(append([]Basis{path.Path.Chosen.Basis}, path.Path.Chosen.Also...)...)...)
		}
		res.NotModelled = res.NotModelled || path.Status == TsNotModeled
	case PrTraceLink:
		res.Value = "none"
		if path.Path != nil && path.Path.Steering != nil && path.Path.Steering.Chosen != nil {
			res.Value = path.Path.Steering.Chosen.LocalLink
			res.Assumptions = append(res.Assumptions, assumptionNumbers(append([]Basis{path.Path.Steering.Basis}, path.Path.Steering.Also...)...)...)
		}
	case PrTraceRule, PrTraceRuleOwner:
		res.Value = "none"
		for _, v := range policy.Rules {
			if v.Chosen {
				res.Value = v.RuleID
				if p.Kind == PrTraceRuleOwner {
					res.Value = v.Owner
				}
			}
		}
		cite(policy)
	case PrTraceMTUMin, PrTraceMTULimit, PrTraceMTUFit:
		if mtu.MTU == nil {
			res.Value, res.NotModelled = "none", true
			break
		}
		cite(mtu)
		switch p.Kind {
		case PrTraceMTUMin:
			res.Value = strconv.Itoa(mtu.MTU.EffectiveMin)
		case PrTraceMTULimit:
			res.Value = mtu.MTU.Limiting
		default:
			switch {
			case p.Size <= mtu.MTU.EffectiveMin:
				res.Value = "fits"
			case p.Size > mtu.MTU.EffectiveMax:
				res.Value = "fragments"
			default:
				res.Value = "depends_on_padding"
			}
		}
	case PrRejectedCode:
		res.Value = "none"
		if path.Path != nil {
			for _, o := range path.Path.Options {
				if !o.Usable && (p.Field == "" || o.Kind == p.Field) {
					res.Value = o.ReasonCode
					res.Assumptions = append(res.Assumptions, assumptionNumbers(append([]Basis{o.Basis}, o.Also...)...)...)
					break
				}
			}
		}
	default:
		return res, fmt.Errorf("unknown probe kind %q", p.Kind)
	}
	return res, nil
}

func (s *LabState) evalRoute(p *Probe) (ProbeResult, error) {
	pfx, ok := pfxOf(p.Prefix)
	if !ok {
		return ProbeResult{}, fmt.Errorf("probe prefix %q is not a prefix", p.Prefix)
	}
	rs, err := ComputeRoutes(s.Topo, s.Cond, s.As.RoutingAssumptions, nil)
	if err != nil {
		return ProbeResult{}, err
	}
	seg := segOr(p.Segment)
	_, cands := rs.Lookup(p.Edge, seg, pfx)
	rank := p.Rank
	if rank == 0 {
		rank = 1
	}
	res := ProbeResult{Value: "none"}
	if len(cands) < rank {
		return res, nil
	}
	r := cands[rank-1]
	switch p.Field {
	case "source":
		res.Value = r.Source
	case "route_type":
		res.Value = r.RouteType
	case "origin":
		res.Value = r.Origin
	case "via":
		res.Value = r.Via
	case "lost_reason_code":
		res.Value = r.LostReasonCode
	case "reachable":
		res.Value = strconv.FormatBool(r.Reachable)
	default:
		return res, fmt.Errorf("unknown route field %q", p.Field)
	}
	if res.Value == "" {
		res.Value = "none"
	}
	res.Detail = r.Reason
	res.Assumptions = assumptionNumbers(append([]Basis{r.Basis}, r.Also...)...)
	res.NotModelled = len(r.NotModelled) > 0
	return res, nil
}

func (s *LabState) evalTunnel(p *Probe) (ProbeResult, error) {
	ts, err := ComputeTunnels(s.Topo, s.Cond, nil)
	if err != nil {
		return ProbeResult{}, err
	}
	id := p.Tunnel
	find := func(id string) *Tunnel {
		for i := range ts.Tunnels {
			if ts.Tunnels[i].ID == id {
				return &ts.Tunnels[i]
			}
		}
		return nil
	}
	tn := find(id)
	if tn == nil {
		if parts := strings.Split(id, ":"); len(parts) == 3 {
			tn = find(parts[0] + ":" + parts[2] + ":" + parts[1])
		}
	}
	if tn == nil {
		return ProbeResult{Value: "none"}, nil
	}
	res := ProbeResult{Value: tn.State, Detail: tn.Reason, Assumptions: assumptionNumbers(append([]Basis{tn.Basis}, tn.Also...)...), NotModelled: tn.State == StUnknown}
	if p.Kind == PrTunnelReason {
		res.Value = tn.ReasonCode
	}
	return res, nil
}

// CheckResult is the outcome of checking one task.
type CheckResult struct {
	Correct     bool   `json:"correct"`
	Given       string `json:"given"`
	Expected    string `json:"expected"`
	Explanation string `json:"explanation"`
	Detail      string `json:"detail"`
	Assumptions []int  `json:"assumptions"`
}

// TaskByID finds a task.
func (l *Lab) TaskByID(id string) *LabTask {
	for i := range l.Tasks {
		if l.Tasks[i].ID == id {
			return &l.Tasks[i]
		}
	}
	return nil
}

// Check grades one task against the recomputed state (with the fixes applied so far). The correct answer is the
// engine's value, never a stored string.
func (l *Lab) Check(base *Topology, taskID, answer string, applied []string) (*CheckResult, error) {
	task := l.TaskByID(taskID)
	if task == nil {
		return nil, &PatchError{fmt.Sprintf("unknown task %q", taskID)}
	}
	st, err := l.State(base, applied)
	if err != nil {
		return nil, err
	}
	pr, err := st.Eval(task.Probe)
	if err != nil {
		return nil, err
	}
	res := &CheckResult{Explanation: task.Explanation, Detail: pr.Detail, Assumptions: pr.Assumptions}
	if res.Assumptions == nil {
		res.Assumptions = []int{}
	}
	if task.Type == TaskFix {
		res.Given = pr.Value
		res.Expected = task.Equals
		res.Correct = pr.Value == task.Equals
		return res, nil
	}
	var chosen *LabChoice
	for i := range task.Choices {
		if task.Choices[i].ID == answer {
			chosen = &task.Choices[i]
		}
	}
	if chosen == nil {
		return nil, &PatchError{fmt.Sprintf("unknown choice %q", answer)}
	}
	res.Given = chosen.Text
	for _, c := range task.Choices {
		if c.Value == pr.Value {
			res.Expected = c.Text
		}
	}
	res.Correct = chosen.Value == pr.Value
	return res, nil
}

// HasFix reports whether the lab allows a fix with this id.
func (l *Lab) HasFix(id string) bool { return l.fix(id) != nil }

func (s *LabState) evalTimeline(p *Probe) (ProbeResult, error) {
	tr, err := Timeline(s.Topo, Conditions{}, p.Events)
	if err != nil {
		return ProbeResult{}, err
	}
	res := ProbeResult{Value: "none"}
	if len(tr.Frames) == 0 {
		return res, nil
	}
	for _, f := range tr.Frames[len(tr.Frames)-1].Failovers {
		if f.Spoke != p.Edge {
			continue
		}
		res.Assumptions = assumptionNumbers(f.Basis)
		if p.Kind == PrTimelinePeer {
			res.Value, res.Detail = f.PeerHub, f.PeerNote
		} else {
			res.Value, res.Detail = f.LocalHub, f.LocalNote
			res.NotModelled = len(f.NotModelled) > 0
		}
	}
	return res, nil
}

func (s *LabState) evalEffective(p *Probe) (ProbeResult, error) {
	ec, err := EffectiveConfig(s.Topo, p.Edge)
	if err != nil {
		return ProbeResult{}, err
	}
	for _, d := range ec.Device {
		if d.ID != p.Field {
			continue
		}
		res := ProbeResult{Value: fmt.Sprint(d.Value), Detail: d.Why, NotModelled: d.NotModelled != nil,
			Assumptions: assumptionNumbers(append([]Basis{d.Basis}, d.Also...)...)}
		if p.Kind == PrEffectiveSource {
			res.Value = d.Source.Kind
		}
		if res.Value == "" || res.Value == "<nil>" {
			res.Value = "unset"
		}
		return res, nil
	}
	return ProbeResult{}, fmt.Errorf("the probe %s: edge %q has no device setting %q", p.Kind, p.Edge, p.Field)
}
