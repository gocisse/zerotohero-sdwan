package sim

import (
	"fmt"
	"net/netip"
	"strings"
)

// Trace stage ids, in presentation order (assumption 91).
const (
	TrSegment     = "segment"
	TrPolicy      = "policy"
	TrFirewall    = "firewall"
	TrRoute       = "route"
	TrPath        = "path"
	TrRemediation = "remediation"
	TrMTU         = "mtu"
	TrReturn      = "return"
)

// Trace verdicts.
const (
	VerdictDelivered   = "delivered"
	VerdictBlocked     = "blocked"
	VerdictUnreachable = "unreachable"
	VerdictNotModelled = "not-modelled"
)

// Trace stage statuses.
const (
	TsOK         = "ok"
	TsBlocked    = "blocked"
	TsFailed     = "failed"
	TsInfo       = "info"
	TsNotModeled = "not-modelled"
	TsSkipped    = "skipped"
	TsNotReached = "not-reached"
)

// FlowSpec describes ONE flow to follow end to end. Destination is an IPv4 address or prefix, "edge:<id>" or
// "nsd:<id>". With neither SourceIP nor SourceInterface the flow is LAN traffic passing through the Edge (assumption 87).
type FlowSpec struct {
	Source          string `yaml:"source" json:"source"`
	SourceIP        string `yaml:"source_ip" json:"source_ip,omitempty"`
	SourceInterface string `yaml:"source_interface" json:"source_interface,omitempty"`
	Destination     string `yaml:"destination" json:"destination"`
	Segment         string `yaml:"segment" json:"segment,omitempty"`
	Application     string `yaml:"application" json:"application,omitempty"`
	Protocol        string `yaml:"protocol" json:"protocol,omitempty"`
	Port            int    `yaml:"port" json:"port,omitempty"`
	DSCP            string `yaml:"dscp" json:"dscp,omitempty"`
	TrafficClass    string `yaml:"traffic_class" json:"traffic_class,omitempty"`
	PacketSize      int    `yaml:"packet_size" json:"packet_size,omitempty"`
}

// FwVerdict is one firewall rule's evaluation for the flow.
type FwVerdict struct {
	RuleID   string  `json:"rule_id"`
	Group    string  `json:"group"`
	Owner    string  `json:"owner"`
	Action   string  `json:"action"`
	Matched  bool    `json:"matched"`
	Decisive bool    `json:"decisive"`
	Reason   string  `json:"reason"`
	Basis    Basis   `json:"basis"`
	Also     []Basis `json:"also,omitempty"`
}

// RemedMech is one remediation mechanism with its documented conditions (no active/inactive claim).
type RemedMech struct {
	Name       string  `json:"name"`
	Candidate  bool    `json:"candidate"`
	Conditions string  `json:"conditions"`
	Basis      Basis   `json:"basis"`
	Also       []Basis `json:"also,omitempty"`
}

// TraceStage is one step of the trace.
type TraceStage struct {
	Number      int           `json:"number"`
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Status      string        `json:"status"`
	Input       []string      `json:"input"`
	Decision    string        `json:"decision"`
	Reason      string        `json:"reason"`
	Details     []string      `json:"details,omitempty"`
	Basis       Basis         `json:"basis"`
	Also        []Basis       `json:"also,omitempty"`
	NotModelled []NotModelled `json:"not_modelled,omitempty"`
	// Heuristic is set when the stage applied a rule of thumb that is not documented (assumption 88); the UI shows it.
	Heuristic string `json:"heuristic,omitempty"`
	LessonRef string `json:"lesson_ref"`
	Screen    string `json:"screen"`
	// Structured evidence for the UI.
	Rules       []RuleVerdict `json:"rules,omitempty"`
	Firewall    []FwVerdict   `json:"firewall,omitempty"`
	Route       *Route        `json:"route,omitempty"`
	Candidates  []Route       `json:"candidates,omitempty"`
	Path        *PathDecision `json:"path,omitempty"`
	Remediation []RemedMech   `json:"remediation,omitempty"`
	MTU         *MTUResult    `json:"mtu,omitempty"`
	Reach       *ReachResult  `json:"reachability,omitempty"`
}

// TraceResult is the answer of TraceFlow.
type TraceResult struct {
	Flow         FlowSpec           `json:"flow"`
	Verdict      string             `json:"verdict"`
	BlockedAt    int                `json:"blocked_at"`
	BlockedStage string             `json:"blocked_stage"`
	Summary      string             `json:"summary"`
	Stages       []TraceStage       `json:"stages"`
	Assumptions  []AssumptionSwitch `json:"assumptions"`
	DependsOn    []DependsOn        `json:"depends_on"`
	NotModelled  []NotModelled      `json:"not_modelled"`
}

var traceMeta = []struct{ id, title, lesson, screen string }{
	{TrSegment, "Segment and source interface", "cfg-segments", "edges/{edge}?tab=device"},
	{TrPolicy, "Business policy at the source Edge", "cfg-business-policy", "edges/{edge}?tab=business-policy"},
	{TrFirewall, "Firewall at the source Edge", "cfg-firewall", "edges/{edge}?tab=firewall"},
	{TrRoute, "Route lookup", "rt-diagnosis", "routes"},
	{TrPath, "Path selection and link steering", "ov-diagnosis", "paths"},
	{TrRemediation, "Remediation", "dmpo-remediation", "monitor?edge={edge}&tab=links"},
	{TrMTU, "Overhead and MTU", "arch-mtu-overhead", "paths"},
	{TrReturn, "Return path", "rt-diagnosis", "reachability"},
}

// TraceFlow follows one flow end to end: segment and source interface, business policy, firewall, route lookup,
// path selection, remediation, overhead and MTU, return path. It reuses EffectiveConfig, ComputeRoutes, Reachability
// and SelectPath; every stage carries its evidence. The result also lists every assumption switch whose flip would
// change the verdict.
func TraceFlow(t *Topology, cond Conditions, as TraceAssumptions, f FlowSpec, clock Clock) (*TraceResult, error) {
	res, err := traceFlow(t, cond, as, f, clock)
	if err != nil {
		return nil, err
	}
	res.DependsOn = []DependsOn{}
	for _, key := range as.switchKeys() {
		flipped, ok := as.Flip(key)
		if !ok {
			continue
		}
		other, err := traceFlow(t, cond, flipped, f, clock)
		if err != nil {
			continue
		}
		if other.Verdict != res.Verdict || other.BlockedStage != res.BlockedStage {
			for _, sw := range as.Describe(map[string]bool{key: true}) {
				if sw.Key == key {
					res.DependsOn = append(res.DependsOn, DependsOn{Switch: sw, Flipped: flipped.value(key), NewOutcome: other.Verdict, NewFailing: other.BlockedStage})
				}
			}
		}
	}
	return res, nil
}

type traceCtx struct {
	t    *Topology
	cond Conditions
	as   TraceAssumptions
	f    FlowSpec
	s    *rtState
	src  *edgeInfo
	seg  string
	res  *TraceResult
	// ignoreFw lets later stages run after a firewall block (lab probes only).
	ignoreFw bool
}

func (tc *traceCtx) stage(i int) *TraceStage { return &tc.res.Stages[i] }

func (tc *traceCtx) stop(i int, verdict string) {
	tc.res.Verdict, tc.res.BlockedAt, tc.res.BlockedStage = verdict, i+1, traceMeta[i].id
}

func parseDest(d string) (kind, id string, pfx netip.Prefix, err error) {
	d = strings.TrimSpace(d)
	switch {
	case strings.HasPrefix(d, "edge:"):
		return "edge", strings.TrimPrefix(d, "edge:"), pfx, nil
	case strings.HasPrefix(d, "nsd:"):
		return "nsd", strings.TrimPrefix(d, "nsd:"), pfx, nil
	}
	p, err := parseQuery(d)
	return "ip", "", p, err
}

func traceFlow(t *Topology, cond Conditions, as TraceAssumptions, f FlowSpec, clock Clock) (*TraceResult, error) {
	return traceFlowOpts(t, cond, as, f, clock, false)
}

// traceFlowOpts with ignoreFirewall evaluates the firewall stage but does not let it stop the trace: lab probes about
// the path or the MTU read those stages independently of the firewall verdict (the order is not documented).
func traceFlowOpts(t *Topology, cond Conditions, as TraceAssumptions, f FlowSpec, clock Clock, ignoreFirewall bool) (*TraceResult, error) {
	if err := as.Validate(); err != nil {
		return nil, err
	}
	s, err := newRtState(t, cond, as.RoutingAssumptions, clock)
	if err != nil {
		return nil, err
	}
	src := s.c.edgeByID[f.Source]
	if src == nil {
		return nil, &QueryError{fmt.Sprintf("unknown source Edge %q", f.Source)}
	}
	kind, id, dpfx, err := parseDest(f.Destination)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "edge":
		if s.c.edgeByID[id] == nil {
			return nil, &QueryError{fmt.Sprintf("unknown destination Edge %q", id)}
		}
	case "nsd":
		if s.c.nsdByID[id] == nil {
			return nil, &QueryError{fmt.Sprintf("unknown Non SD-WAN destination %q", id)}
		}
	}
	f.Segment = segOr(f.Segment)
	f.Protocol = strings.ToLower(f.Protocol)
	tc := &traceCtx{t: t, cond: cond, as: as, f: f, s: s, src: src, seg: f.Segment, ignoreFw: ignoreFirewall,
		res: &TraceResult{Flow: f, Verdict: VerdictDelivered, Stages: []TraceStage{}, NotModelled: []NotModelled{}}}
	for i, m := range traceMeta {
		tc.res.Stages = append(tc.res.Stages, TraceStage{Number: i + 1, ID: m.id, Title: m.title, Status: TsNotReached, Input: []string{}, LessonRef: m.lesson,
			Screen: strings.ReplaceAll(m.screen, "{edge}", src.Edge.ID), Basis: basisOf(RA91StageOrder)}) // rule: assume-91
	}
	tc.res.NotModelled = append(tc.res.NotModelled, notModelled(RNMFwOrder, "Firewall versus business policy versus routing order")) // rule: nm-fw-vs-bp-order

	var srcIP netip.Addr
	var srcIface *Interface
	if !tc.segment(&srcIP, &srcIface) {
		return tc.finish(s), nil
	}
	pol, ec := tc.policy(srcIP, kind, id, dpfx)
	if !tc.firewall(ec, srcIP, srcIface, kind, dpfx) {
		return tc.finish(s), nil
	}
	var destEdge *edgeInfo
	var destNSD string
	if !tc.route(kind, id, dpfx, &destEdge, &destNSD) {
		return tc.finish(s), nil
	}
	var pd *PathDecision
	if !tc.path(pol, destEdge, destNSD, &pd) {
		return tc.finish(s), nil
	}
	tc.remediation(pd)
	tc.mtu(pd, destNSD != "")
	tc.returnPath(pd, srcIP, srcIface, destEdge, destNSD)
	return tc.finish(s), nil
}

func (tc *traceCtx) finish(s *rtState) *TraceResult {
	r := tc.res
	for i := range r.Stages {
		if r.Stages[i].Status == TsNotReached && r.Stages[i].Decision == "" {
			r.Stages[i].Decision = "Not reached: the trace ended earlier."
		}
	}
	r.NotModelled = append(r.NotModelled, s.nms...)
	for i := range r.Stages {
		for _, nm := range r.Stages[i].NotModelled {
			r.NotModelled = addNMUnique(r.NotModelled, nm)
		}
	}
	r.Assumptions = tc.as.Describe(s.used)
	r.Summary = tc.summary()
	return r
}

func addNMUnique(list []NotModelled, nm NotModelled) []NotModelled {
	for _, x := range list {
		if x.Rule == nm.Rule {
			return list
		}
	}
	return append(list, nm)
}

func (tc *traceCtx) summary() string {
	f := tc.f
	what := fmt.Sprintf("The flow from %s to %s", tc.src.Edge.Name, f.Destination)
	if f.Application != "" {
		what = fmt.Sprintf("The %s flow from %s to %s", f.Application, tc.src.Edge.Name, f.Destination)
	}
	r := tc.res
	switch r.Verdict {
	case VerdictDelivered:
		for _, st := range r.Stages {
			if st.ID == TrPath && st.Status == TsOK && st.Path != nil && st.Path.Chosen != nil {
				return fmt.Sprintf("%s is delivered over %s.", what, st.Path.Chosen.Label)
			}
		}
		return what + " is delivered without crossing the overlay."
	case VerdictNotModelled:
		return fmt.Sprintf("%s cannot be followed to the end: stage %d (%s) needs something the documentation does not say.", what, r.BlockedAt, strings.ToLower(traceMeta[r.BlockedAt-1].title))
	}
	st := r.Stages[r.BlockedAt-1]
	verb := "is blocked"
	if r.Verdict == VerdictUnreachable {
		verb = "cannot reach its destination"
	}
	return fmt.Sprintf("%s %s at stage %d (%s): %s", what, verb, st.Number, strings.ToLower(st.Title), st.Reason)
}
