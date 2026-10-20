package sim

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
)

// Reachability stage ids, in the order of the Section 6 diagnostic (rt-diagnosis).
const (
	StageKnown        = "known"
	StageAdvertised   = "advertised"
	StageFiltered     = "filtered"
	StageRedistribute = "redistributed"
	StageWinner       = "winner"
	StagePath         = "path"
)

// Stage statuses.
const (
	StageOK         = "ok"
	StageFailed     = "failed"
	StageSkipped    = "skipped"
	StageNotReached = "not-reached"
	StageNotModeled = "not-modelled"
)

// Outcomes of a reachability query.
const (
	OutcomeReachable   = "reachable"
	OutcomeUnreachable = "unreachable"
	OutcomeNotModelled = "not-modelled"
)

// ReachQuery asks: can the source Edge reach this prefix or address, in this segment?
type ReachQuery struct {
	Source             string `json:"source"`
	Destination        string `json:"destination"`
	Segment            string `json:"segment,omitempty"`
	DestinationSegment string `json:"destination_segment,omitempty"`
}

// QueryError is returned for a query that names things the topology does not have.
type QueryError struct{ Message string }

func (e *QueryError) Error() string { return e.Message }

// Stage is one question of the diagnostic, answered.
type Stage struct {
	Number    int      `json:"number"`
	ID        string   `json:"id"`
	Question  string   `json:"question"`
	Status    string   `json:"status"`
	Detail    string   `json:"detail"`
	Evidence  []string `json:"evidence,omitempty"`
	Basis     Basis    `json:"basis"`
	Also      []Basis  `json:"also,omitempty"`
	LessonRef string   `json:"lesson_ref"`
	Where     string   `json:"where"`
}

// OriginView is one Edge that owns (or would own) the prefix and what happened to its route at each stage.
type OriginView struct {
	Edge     string  `json:"edge"`
	Name     string  `json:"name"`
	Prefix   string  `json:"prefix"`
	Source   string  `json:"source"`
	Via      string  `json:"via"`
	Stage    string  `json:"stage"`  // the stage it stops at, or "arrived"
	Status   string  `json:"status"` // ok or failed
	Reason   string  `json:"reason"`
	Basis    Basis   `json:"basis"`
	Also     []Basis `json:"also,omitempty"`
	Selected bool    `json:"selected"`
}

// ReachResult is the answer of Reachability.
type ReachResult struct {
	Query         ReachQuery         `json:"query"`
	Prefix        string             `json:"prefix"`
	Outcome       string             `json:"outcome"`
	FirstFailing  string             `json:"first_failing"`
	Reason        string             `json:"reason"`
	LessonRef     string             `json:"lesson_ref"`
	Stages        []Stage            `json:"stages"`
	Origins       []OriginView       `json:"origins"`
	Selected      *Route             `json:"selected,omitempty"`
	Candidates    []Route            `json:"candidates"`
	Path          *PathOption        `json:"path,omitempty"`
	Assumptions   []AssumptionSwitch `json:"assumptions"`
	DependsOn     []DependsOn        `json:"depends_on"`
	NotModelled   []NotModelled      `json:"not_modelled"`
	ReturnTraffic string             `json:"return_traffic"`
}

// DependsOn says that flipping one assumption switch changes the answer.
type DependsOn struct {
	Switch     AssumptionSwitch `json:"switch"`
	Flipped    any              `json:"flipped_value"`
	NewOutcome string           `json:"new_outcome"`
	NewFailing string           `json:"new_first_failing"`
}

// Reachability answers "why does Edge X have (or not have) a route to this prefix": the six stages of the route's
// life, the first failing stage and a reason. Unverified defaults it relied on are listed, and so are the
// assumption switches whose flip would change the answer.
func Reachability(t *Topology, cond Conditions, as RoutingAssumptions, q ReachQuery, clock Clock) (*ReachResult, error) {
	res, err := reachability(t, cond, as, q, clock)
	if err != nil {
		return nil, err
	}
	res.DependsOn = []DependsOn{}
	for _, d := range switchDefs {
		flipped, _ := as.Flip(d.key)
		other, err := reachability(t, cond, flipped, q, clock)
		if err != nil {
			continue
		}
		if other.Outcome != res.Outcome || other.FirstFailing != res.FirstFailing {
			sw := as.Describe(map[string]bool{d.key: true})
			for _, x := range sw {
				if x.Key == d.key {
					res.DependsOn = append(res.DependsOn, DependsOn{Switch: x, Flipped: d.get(flipped), NewOutcome: other.Outcome, NewFailing: other.FirstFailing})
				}
			}
		}
	}
	return res, nil
}

func parseQuery(q string) (netip.Prefix, error) {
	q = strings.TrimSpace(q)
	if strings.Contains(q, "/") {
		p, err := netip.ParsePrefix(q)
		if err != nil || !p.Addr().Is4() {
			return netip.Prefix{}, &QueryError{"the destination is not an IPv4 prefix like 10.20.0.0/24"}
		}
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(q)
	if err != nil || !a.Is4() {
		return netip.Prefix{}, &QueryError{"the destination is neither an IPv4 address nor a prefix like 10.20.0.0/24"}
	}
	return netip.PrefixFrom(a, 32), nil
}

var stageMeta = []struct{ id, q, lesson, where string }{
	{StageKnown, "Does the owning Edge know the prefix?", "rt-diagnosis", "Configure > Edges > (Edge) > Device > Routing & NAT: Static Route Settings, OSPF, BGP"},
	{StageAdvertised, "Is it advertised into the overlay?", "rt-redistribution", "the static route's Advertise box, or Configure > Overlay Flow Control > Global Advertise Flags"},
	{StageFiltered, "Is it filtered or held back?", "rt-filters", "the neighbor's Inbound Filter in Routing & NAT > BGP (Filter List)"},
	{StageRedistribute, "Is it redistributed where it needs to be (the source Edge's LAN)?", "rt-redistribution", "Routing & NAT > OSPF (Overlay Prefixes) or BGP (Overlay Prefix) on the source Edge or its Profile"},
	{StageWinner, "Which route wins at the source Edge?", "rt-ofc", "Configure > Overlay Flow Control (Preferred VPN Exits, pinning) or the filters"},
	{StagePath, "Is the path built and reachable?", "rt-diagnosis", "Monitor > Edges (Links, Paths) and Remote Diagnostics: List Paths, VPN Test (Section 5)"},
}

func reachability(t *Topology, cond Conditions, as RoutingAssumptions, q ReachQuery, clock Clock) (*ReachResult, error) {
	dst, err := parseQuery(q.Destination)
	if err != nil {
		return nil, err
	}
	s, err := newRtState(t, cond, as, clock)
	if err != nil {
		return nil, err
	}
	src := s.c.edgeByID[q.Source]
	if src == nil {
		return nil, &QueryError{fmt.Sprintf("unknown source Edge %q", q.Source)}
	}
	seg := segOr(q.Segment)
	known := false
	for _, sg := range t.Segments {
		known = known || sg.ID == seg
	}
	if !known && seg != GlobalSegmentID {
		return nil, &QueryError{fmt.Sprintf("unknown segment %q", seg)}
	}
	rs := s.routeSet()
	res := &ReachResult{Query: q, Prefix: dst.String(), Stages: []Stage{}, Origins: []OriginView{}, Candidates: []Route{}, NotModelled: rs.NotModelled}
	res.Query.Segment = seg
	res.ReturnTraffic = "Return traffic is a separate query from the other Edge: asymmetric reachability is a normal result and is not computed here."
	finish := func() *ReachResult { res.Assumptions = as.Describe(s.used); res.NotModelled = s.nms; return res }

	// Cross-segment queries are not answered.
	if q.DestinationSegment != "" && q.DestinationSegment != seg {
		nm := s.nm(RNMRtCrossSegment, "Routes crossing segments") // rule: nm-rt-cross-segment
		res.Outcome, res.FirstFailing = OutcomeNotModelled, StageKnown
		res.Reason = "not modelled: " + nm.Reason + fmt.Sprintf(" The question asks for segment %s from segment %s.", q.DestinationSegment, seg)
		res.LessonRef = "rt-big-picture"
		st := s.stage(0, StageNotModeled, res.Reason, nm.Basis)
		res.Stages = append(res.Stages, st)
		for i := 1; i < len(stageMeta); i++ {
			res.Stages = append(res.Stages, s.stage(i, StageNotReached, "Not evaluated: the question crosses segments.", nm.Basis))
		}
		return finish(), nil
	}

	// Origins: routes for exactly this prefix, else the most specific routes that cover it (assumption 84).
	var origins []*originRoute
	best := -1
	for _, o := range s.origins {
		if o.segment == seg && pfxContains(o.pfx, dst) {
			if o.pfx == dst {
				best = 99
			}
			if best < 99 && o.pfx.Bits() > best {
				best = o.pfx.Bits()
			}
		}
	}
	for _, o := range s.origins { // rule: assume-84
		if o.segment == seg && pfxContains(o.pfx, dst) && (best == 99 && o.pfx == dst || best != 99 && o.pfx.Bits() == best) {
			origins = append(origins, o)
		}
	}
	// A prefix that exists only in another segment: the model does not follow routes across segments.
	if len(origins) == 0 {
		for _, o := range s.origins {
			if o.segment != seg && pfxContains(o.pfx, dst) {
				nm := s.nm(RNMRtCrossSegment, "Routes crossing segments") // rule: nm-rt-cross-segment
				res.Outcome, res.FirstFailing, res.LessonRef = OutcomeNotModelled, StageKnown, "rt-big-picture"
				res.Reason = fmt.Sprintf("not modelled: the prefix exists only in segment %s and the query is for segment %s. %s", o.segment, seg, nm.Reason)
				res.Stages = append(res.Stages, s.stage(0, StageNotModeled, res.Reason, nm.Basis))
				for i := 1; i < len(stageMeta); i++ {
					res.Stages = append(res.Stages, s.stage(i, StageNotReached, "Not evaluated.", nm.Basis))
				}
				return finish(), nil
			}
		}
	}

	sel, cands := rs.Lookup(src.Edge.ID, seg, dst)
	res.Candidates = nonNilRoutes(cands)
	res.Selected = sel
	var remote, local []*originRoute
	for _, o := range origins {
		if o.edge == src {
			local = append(local, o)
		} else {
			remote = append(remote, o)
		}
	}
	localWins := sel != nil && sel.Origin == ""
	var stages [6]Stage
	for i := range stageMeta {
		stages[i] = s.stage(i, StageOK, "", basisOf(RA83StageOrder)) // rule: assume-83
	}

	// 1. Known at the owning Edge.
	{
		st := &stages[0]
		var yes, no []string
		for _, o := range origins {
			if o.known {
				yes = append(yes, fmt.Sprintf("%s knows %s (%s): %s", o.edge.Edge.Name, o.pfx, s.originLabel(o), o.knownWhy))
				if st.Basis.Rule == RA83StageOrder { // rule: assume-83
					st.Basis, st.Also = o.knownBasis, o.knownAlso
				}
			} else {
				no = append(no, fmt.Sprintf("%s does not learn %s (%s): %s", o.edge.Edge.Name, o.pfx, s.originLabel(o), o.knownWhy))
			}
		}
		switch {
		case len(origins) == 0:
			st.Status = StageFailed
			st.Detail = fmt.Sprintf("No Edge in the topology has a connected subnet, static route, OSPF route, BGP route or Non SD-WAN route covering %s in segment %s, so nothing originates it.", dst, seg)
			st.Basis = basisOf(RRtLocalRemote) // rule: rt-local-remote
		case len(yes) == 0:
			st.Status, st.Detail, st.Evidence = StageFailed, "No Edge learns the prefix.", no
			st.Basis, st.Also = origins[0].knownBasis, origins[0].knownAlso
		default:
			st.Detail, st.Evidence = fmt.Sprintf("%d of %d owning Edge(s) know the prefix.", len(yes), len(origins)), append(yes, no...)
		}
	}
	// 2 and 3: only the remote owners matter, and only when the source does not use a local route.
	if localWins {
		for _, i := range []int{1, 2, 3} {
			stages[i].Status = StageSkipped
			stages[i].Detail = "The selected route is local to the source Edge: the prefix does not need to travel through the overlay."
		}
	} else {
		var adv, notAdv, ok3, held []string
		var advOrigins []*originRoute
		for _, o := range remote {
			if !o.known {
				continue
			}
			if o.advOK {
				advOrigins = append(advOrigins, o)
				adv = append(adv, fmt.Sprintf("%s advertises %s: %s", o.edge.Edge.Name, o.pfx, o.advWhy))
			} else {
				notAdv = append(notAdv, fmt.Sprintf("%s does not advertise %s: %s", o.edge.Edge.Name, o.pfx, o.advWhy))
			}
		}
		st := &stages[1]
		switch {
		case len(adv) == 0 && len(notAdv) == 0:
			st.Status, st.Detail = StageSkipped, "No other Edge knows the prefix, so there is nothing to advertise."
		case len(adv) == 0:
			st.Status, st.Detail, st.Evidence = StageFailed, "No owning Edge advertises the prefix into the overlay.", notAdv
			for _, o := range remote {
				if o.known && !o.advOK {
					st.Basis, st.Also = o.advBasis, o.advAlso
					break
				}
			}
		default:
			st.Detail, st.Evidence = fmt.Sprintf("%d owning Edge(s) advertise the prefix into the overlay.", len(adv)), append(adv, notAdv...)
			st.Basis = advOrigins[0].advBasis
			st.Also = advOrigins[0].advAlso
		}
		st = &stages[2]
		for _, o := range advOrigins {
			if o.held {
				held = append(held, fmt.Sprintf("%s: %s", o.edge.Edge.Name, o.heldWhy))
			} else {
				ok3 = append(ok3, fmt.Sprintf("%s: not filtered or held back", o.edge.Edge.Name))
			}
		}
		switch {
		case len(advOrigins) == 0:
			st.Status, st.Detail = StageNotReached, "No route was advertised, so there is nothing to filter."
			if stages[1].Status == StageSkipped {
				st.Status, st.Detail = StageSkipped, "Nothing to filter."
			}
		case len(ok3) == 0:
			st.Status, st.Detail, st.Evidence = StageFailed, "Every advertised route is filtered or held back.", held
			st.Basis, st.Also = advOrigins[0].heldBasis, advOrigins[0].heldAlso
		default:
			st.Detail, st.Evidence = fmt.Sprintf("%d route(s) pass the filters.", len(ok3)), append(ok3, held...)
			st.Basis = basisOf(RRtFilterMatch) // rule: rt-filter-match
		}
		// 4. Redistributed into the source Edge's LAN.
		st = &stages[3]
		runsLAN := src.runsLANRouting(seg) // rule: assume-83
		switch {
		case !runsLAN:
			st.Status, st.Detail = StageSkipped, "The source Edge runs no OSPF or BGP toward its LAN in this topology; its LAN hosts are assumed to send traffic to the Edge."
			st.Basis = basisOf(RA83StageOrder) // rule: assume-83
		case sel == nil:
			st.Status, st.Detail = StageSkipped, "The source Edge has no selected route to redistribute (see stages 5 and 6)."
		default:
			okA, all := s.advertsCovering(src.Edge.ID, sel, dst)
			if len(okA) > 0 {
				st.Detail = fmt.Sprintf("%s sends %s to its LAN routers (%s).", src.Edge.Name, okA[0].Prefix, okA[0].Reason)
				st.Basis, st.Also = okA[0].Basis, okA[0].Also
				st.Evidence = advLines(okA)
			} else {
				st.Status = StageFailed
				st.Detail = fmt.Sprintf("%s selected the route but does not send it to its LAN routers.", src.Edge.Name)
				st.Evidence = advLines(all)
				if len(all) > 0 {
					st.Basis, st.Also = all[0].Basis, all[0].Also
				} else {
					st.Basis = basisOf(RRtOverlaySwitches) // rule: rt-overlay-redistribution-switches
				}
			}
		}
	}
	// 5. Winner.
	{
		st := &stages[4]
		if len(cands) == 0 {
			st.Status = StageFailed
			st.Basis = basisOf(RRtLocalRemote) // rule: rt-local-remote
			switch {
			case s.cond.OrchestratorUnreachable:
				st.Detail = "The source Edge has no route. With the Orchestrator unreachable the model does not compute remote routes."
				st.Basis = notModelled(RNMOrchestratorDown, "").Basis
			case len(remote) > 0 && !src.cvpnOn(seg):
				st.Detail = fmt.Sprintf("%s has no route: Cloud VPN is not enabled for segment %s on it, so the Gateway does not reflect other Edges' routes to it.", src.Edge.Name, seg)
				st.Basis, st.Also = basisOf(RA32CloudVPNReflect), bases(RRtReflect) // rule: rt-gateway-reflects // rule: assume-32
			default:
				st.Detail = fmt.Sprintf("%s has no route for %s: no candidate reached its table.", src.Edge.Name, dst)
			}
		} else {
			st.Evidence = routeLines(cands)
			w := cands[0]
			for i := range cands {
				if cands[i].Selected {
					w = cands[i]
					break
				}
			}
			st.Detail = fmt.Sprintf("%s has %d candidate(s) for %s; the winner is %s. %s", src.Edge.Name, len(cands), cands[0].Prefix, s.label(&w), w.Reason)
			st.Basis, st.Also = w.Basis, w.Also
			if sel != nil {
				for _, er := range rs.Edges {
					if er.Edge != src.Edge.ID {
						continue
					}
					for _, tb := range er.Segments {
						for _, r := range tb.Routes {
							if tb.Segment == seg && pfxContains(r.pfx, dst) && r.pfx.Bits() > sel.pfx.Bits() && !r.Reachable {
								st.Evidence = append(st.Evidence, fmt.Sprintf("The more specific route %s (%s) is unreachable (%s), so the less specific %s is used.", r.Prefix, s.label(&r), r.ReachableCode, sel.Prefix)) // rule: rt-longest-prefix
							}
						}
					}
				}
			}
			if w.Selected {
				st.Detail = fmt.Sprintf("%s has %d candidate(s) for %s; the winner is %s. %s", src.Edge.Name, len(cands), w.Prefix, s.label(&w), w.Reason)
			}
		}
	}
	// 6. Path.
	{
		st := &stages[5]
		switch {
		case len(cands) == 0:
			st.Status, st.Detail = StageNotReached, "There is no route to follow."
		case sel == nil:
			st.Status = StageFailed
			var parts []string
			for _, c := range cands {
				parts = append(parts, fmt.Sprintf("%s: %s (%s)", s.label(&c), c.ReachableReason, c.ReachableCode))
			}
			st.Detail = "A route exists but every candidate is unreachable, so nothing can be selected."
			st.Evidence = parts
			st.Basis = basisOf(RRtUnreachable) // rule: rt-unreachable-code
			for _, c := range cands {
				if c.Source != SrcLocalStatic {
					st.Also = bases(RRtLostNext) // rule: rt-lost-reason-next
				}
			}
		case sel.Origin == "":
			st.Status, st.Detail = StageSkipped, "The selected route is local to the source Edge, so no overlay path is needed."
		default:
			pv := s.pathTo(src, s.c.edgeByID[sel.Origin])
			st.Detail = fmt.Sprintf("The path to %s is built: %s.", sel.OriginName, pv.label)
			if pv.option != nil {
				res.Path = pv.option
				st.Basis, st.Also = pv.option.Basis, pv.option.Also
				st.Evidence = []string{pv.option.Reason}
			}
			if sel.Source == SrcNSDRemote || sel.Source == SrcNSDLocal {
				st.Detail += " The Non SD-WAN tunnel itself is up."
			}
		}
	}

	// Per-origin verdicts (several exits for one prefix).
	for _, o := range origins {
		ov := OriginView{Edge: o.edge.Edge.ID, Name: o.edge.Edge.Name, Prefix: o.pfx.String(), Source: s.originLabel(o), Via: o.via, Status: StageOK, Stage: "arrived"}
		switch {
		case !o.known:
			ov.Stage, ov.Status, ov.Reason, ov.Basis, ov.Also = StageKnown, StageFailed, o.knownWhy, o.knownBasis, o.knownAlso
		case o.edge == src:
			ov.Reason, ov.Basis = "this is the source Edge's own route", o.knownBasis
		case !o.advOK:
			ov.Stage, ov.Status, ov.Reason, ov.Basis, ov.Also = StageAdvertised, StageFailed, o.advWhy, o.advBasis, o.advAlso
		case o.held:
			ov.Stage, ov.Status, ov.Reason, ov.Basis, ov.Also = StageFiltered, StageFailed, o.heldWhy, o.heldBasis, o.heldAlso
		default:
			ov.Reason, ov.Basis = o.advWhy, o.advBasis
			if sel != nil && sel.Origin == o.edge.Edge.ID {
				ov.Selected = true
				ov.Reason = ""
			} else if rt := findRemote(cands, o.edge.Edge.ID); rt == nil {
				ov.Stage, ov.Status, ov.Reason = StageWinner, StageFailed, "its route did not reach the source Edge's table (Cloud VPN on the source or the Orchestrator)"
			} else {
				ov.Reason = rt.Reason
				if !rt.Reachable {
					ov.Stage, ov.Status = StagePath, StageFailed
				} else {
					ov.Stage = StageWinner
				}
			}
		}
		res.Origins = append(res.Origins, ov)
	}
	sort.SliceStable(res.Origins, func(i, j int) bool { return res.Origins[i].Name < res.Origins[j].Name })

	// Assemble: statuses after the first failure are "not reached".
	first := ""
	for i := range stages {
		if first != "" && stages[i].Status != StageSkipped {
			if stages[i].Status == StageFailed || stages[i].Status == StageOK {
				stages[i].Status = StageNotReached
			}
		}
		if first == "" && stages[i].Status == StageFailed {
			first = stages[i].ID
		}
	}
	res.Stages = stages[:]
	switch {
	case first == "":
		res.Outcome = OutcomeReachable
		res.Reason = fmt.Sprintf("%s has a selected, reachable route for %s.", src.Edge.Name, dst)
		if sel != nil {
			res.Reason = fmt.Sprintf("%s reaches %s using %s. %s", src.Edge.Name, dst, s.label(sel), sel.Reason)
			if sel.Prefix != dst.String() {
				res.Reason = fmt.Sprintf("%s reaches %s through the less specific prefix %s (%s). %s", src.Edge.Name, dst, sel.Prefix, s.label(sel), sel.Reason)
			}
		}
		res.LessonRef = "rt-diagnosis"
	default:
		res.Outcome, res.FirstFailing = OutcomeUnreachable, first
		for _, st := range stages {
			if st.ID == first {
				res.Reason = fmt.Sprintf("First failing stage %d (%s) %s", st.Number, st.Question, st.Detail)
				res.LessonRef = st.LessonRef
			}
		}
	}
	return finish(), nil
}

func (b *BGPConfig) neighborsOrNil() []BGPNeighbor {
	if b == nil {
		return nil
	}
	return b.Neighbors
}

func findRemote(c []Route, origin string) *Route {
	for i := range c {
		if c[i].Origin == origin {
			return &c[i]
		}
	}
	return nil
}

func nonNilRoutes(r []Route) []Route {
	if r == nil {
		return []Route{}
	}
	return r
}

func (s *rtState) stage(i int, status, detail string, b Basis) Stage {
	m := stageMeta[i]
	return Stage{Number: i + 1, ID: m.id, Question: m.q, Status: status, Detail: detail, Basis: b, LessonRef: m.lesson, Where: m.where}
}

func (s *rtState) originLabel(o *originRoute) string {
	switch o.kind {
	case "connected":
		return "connected"
	case "static":
		return "static route"
	case "ospf", "bgp":
		return o.routeType
	}
	return "Non SD-WAN"
}

func advLines(list []LANAdvertisement) []string {
	var out []string
	for _, a := range list {
		v := "sent"
		if !a.Advertised {
			v = "not sent"
		}
		out = append(out, fmt.Sprintf("%s %s (%s %s): %s", strings.ToUpper(a.Protocol), a.Prefix, a.Kind, v, a.Reason))
	}
	return out
}

func routeLines(list []Route) []string {
	var out []string
	for _, r := range list {
		mark := "candidate"
		if r.Selected {
			mark = "selected"
		}
		out = append(out, fmt.Sprintf("#%d %s %s (%s): %s", r.Rank, r.Prefix, r.Source, mark, r.Reason))
	}
	return out
}

// runsLANRouting says whether the Edge runs OSPF (Global Segment only) or BGP (a neighbor in the segment) toward its LAN.
func (e *edgeInfo) runsLANRouting(seg string) bool {
	if e.ospfProfile() != nil && e.routing().OSPF != nil && len(e.routing().OSPF.Interfaces) > 0 && seg == GlobalSegmentID {
		return true
	}
	for _, n := range e.routing().BGP.neighborsOrNil() {
		if segOr(n.Segment) == seg {
			return true
		}
	}
	return false
}
