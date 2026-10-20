package sim

import (
	"fmt"
	"net/netip"
	"strings"
)

func (tc *traceCtx) skipFrom(i int, why string) {
	for j := i; j < len(tc.res.Stages); j++ {
		if tc.res.Stages[j].Status == TsNotReached {
			tc.res.Stages[j].Status, tc.res.Stages[j].Decision, tc.res.Stages[j].Reason = TsSkipped, "Not needed.", why
		}
	}
}

// segment is stage 1: the flow's segment and the interface it enters the Edge on.
func (tc *traceCtx) segment(srcIP *netip.Addr, srcIface **Interface) bool {
	st := tc.stage(0)
	e := tc.src
	st.Input = []string{"source Edge " + e.Edge.Name, "segment " + tc.seg}
	st.Basis, st.Also = basisOf(RProfileSegments), bases(RGlobalSegment) // rule: profile-segments // rule: global-segment
	if !e.inProfile(tc.seg) {
		st.Status, st.Decision = TsFailed, "Blocked: the segment is not part of the Edge's Profile."
		st.Reason = fmt.Sprintf("Profile %s does not include segment %s, so the Edge carries no traffic in it.", e.Profile.Name, tc.seg)
		tc.stop(0, VerdictBlocked)
		return false
	}
	if ip := strings.TrimSpace(tc.f.SourceIP); ip != "" {
		a, err := netip.ParseAddr(ip)
		if err != nil || !a.Is4() {
			st.Status, st.Decision, st.Reason = TsFailed, "Blocked: the source IP is not an IPv4 address.", fmt.Sprintf("%q is not an IPv4 address.", ip)
			tc.stop(0, VerdictBlocked)
			return false
		}
		*srcIP = a
		st.Input = append(st.Input, "source IP "+ip)
	}
	inSeg := func(in *Interface) bool {
		segs := in.Segments
		if len(segs) == 0 {
			segs = e.Profile.Segments
		}
		return hasString(segs, tc.seg)
	}
	hasAddr := false
	for i := range e.Edge.Interfaces {
		hasAddr = hasAddr || e.Edge.Interfaces[i].Address != ""
	}
	if id := tc.f.SourceInterface; id != "" {
		in := e.ifaceByID(id)
		switch {
		case in == nil || in.Kind == IfWan:
			st.Status, st.Decision, st.Reason = TsFailed, "Blocked: the source interface is not a LAN interface of this Edge.", fmt.Sprintf("%q is not a LAN or routed interface of %s.", id, e.Edge.Name)
			tc.stop(0, VerdictBlocked)
			return false
		case !inSeg(in):
			st.Status, st.Decision, st.Reason = TsFailed, "Blocked: the interface is not in this segment.", fmt.Sprintf("Interface %s is assigned to other segments.", id)
			tc.stop(0, VerdictBlocked)
			return false
		}
		*srcIface = in
	} else if srcIP.IsValid() && hasAddr {
		for i := range e.Edge.Interfaces {
			in := &e.Edge.Interfaces[i]
			if (in.Kind != IfLan && in.Kind != IfRouted) || in.Address == "" || !inSeg(in) {
				continue
			}
			if p, err := netip.ParsePrefix(in.Address); err == nil && p.Masked().Contains(*srcIP) {
				*srcIface = in
				break
			}
		}
		if *srcIface == nil {
			st.Status, st.Decision, st.Reason = TsFailed, "Blocked: no LAN interface of this Edge has a subnet containing the source IP.", fmt.Sprintf("%s is outside every LAN subnet of %s in segment %s.", *srcIP, e.Edge.Name, tc.seg)
			tc.stop(0, VerdictBlocked)
			return false
		}
	}
	st.Status = TsOK
	switch {
	case *srcIface != nil:
		in := *srcIface
		vlan := "no VLAN"
		if in.VLAN != nil {
			vlan = fmt.Sprintf("VLAN %d", *in.VLAN)
		}
		st.Decision = fmt.Sprintf("The flow enters %s on interface %s (%s, %s).", e.Edge.Name, in.ID, in.Kind, vlan)
		st.Reason = "The interface is in the flow's segment" + map[bool]string{true: " (it has no segment list, so it is in every Profile segment).", false: "."}[len(in.Segments) == 0]
		st.Also = append(st.Also, basisOf(RInterfaceAllSegs)) // rule: interface-all-segments-default
	default:
		st.Decision = "The flow is LAN traffic passing through " + e.Edge.Name + "; no source interface is named."
		st.Reason = "Without a source IP or interface the simulator assumes LAN traffic passing through the Edge, so policy and firewall apply." // rule: assume-87
		st.Also = append(st.Also, basisOf(RA87UnspecHost))
	}
	return true
}

func (tc *traceCtx) destKind(kind string, pfx netip.Prefix) string {
	switch kind {
	case "edge", "nsd":
		return kind
	}
	for _, o := range tc.s.origins {
		if o.segment == tc.seg && o.pfx.Bits() > 0 && pfxContains(o.pfx, pfx) {
			return "ip"
		}
	}
	return DestInternet
}

// policy is stage 2: which business rule matches, which were skipped and why.
func (tc *traceCtx) policy(srcIP netip.Addr, kind, id string, dpfx netip.Prefix) (*MatchedPolicy, *Effective) {
	st := tc.stage(1)
	f := tc.f
	ec, err := EffectiveConfig(tc.t, tc.src.Edge.ID)
	if err != nil {
		st.Status, st.Decision, st.Reason = TsFailed, "The Edge's configuration could not be computed.", err.Error()
		return &MatchedPolicy{}, nil
	}
	dk := tc.destKind(kind, dpfx)
	ctx := RuleCtx{Segment: tc.seg, Application: f.Application, DestKind: dk, SrcIP: srcIP, Protocol: f.Protocol, Port: f.Port, DSCP: f.DSCP}
	if kind == "ip" {
		ctx.DestIP = dpfx.Addr()
	}
	st.Input = []string{"segment " + tc.seg, "application " + orDash(f.Application), "destination " + dk, "protocol " + orDash(f.Protocol), fmt.Sprintf("port %v", orDashInt(f.Port))}
	ev := evaluateRules(ec, ctx, tc.as.PolicyEdgeFirst)
	for _, sp := range ec.BusinessPolicy {
		if sp.Segment.ID == tc.seg && len(sp.EdgeOverrides) > 0 && len(sp.FromProfile) > 0 {
			tc.s.use("policy_edge_first")
		}
	}
	st.Rules = ev.Verdicts
	for _, v := range ev.Verdicts {
		mark := "no match"
		if v.Chosen {
			mark = "MATCHED"
		}
		st.Details = append(st.Details, fmt.Sprintf("%s (%s): %s: %s", v.RuleID, v.Owner, mark, v.Reason))
	}
	st.Basis, st.Also = basisOf(RBizHighestPrec), bases(RBizSegmentAware, RA1FirstMatch, RA3EdgeAbove, RA90RuleFields) // rule: biz-highest-precedence // rule: biz-segment-aware // rule: assume-1 // rule: assume-3 // rule: assume-90
	st.NotModelled = append(st.NotModelled, notModelled(RNMBizOrder, "Business rule evaluation order"))                // rule: nm-biz-order
	if ev.Matched == nil {
		st.Status = TsInfo
		st.Decision = "No business rule matches this flow."
		st.Reason = "When no rule matches a default action applies, but its definition is not given; the trace continues as Multi-Path only because of assumption 2."
		st.NotModelled = append(st.NotModelled, notModelled(RNMBizDefault, "The default action when no rule matches")) // rule: nm-biz-default
		st.Also = append(st.Also, basisOf(RA2NoRuleDefault))
		return &MatchedPolicy{}, ec
	}
	r := ev.Matched
	svc := ev.Effect.EffectiveService
	if svc == "" {
		svc = NetMultipath
	}
	steer := ev.Effect.SteeringMode
	if r.Action.Steering.Strictness != "" {
		steer += ", " + r.Action.Steering.Strictness
	}
	st.Status = TsOK
	st.Decision = fmt.Sprintf("Rule %s (%s) matches: network service %s, link steering %s.", r.ID, ev.Owner, strings.ReplaceAll(svc, "_", " "), strings.ReplaceAll(steer, "_", " "))
	st.Reason = "The first rule in precedence order whose every field matches the flow."
	if ev.Effect.Rewritten {
		st.Reason += " The segment is private, so Internet Multipath is treated as Direct."
		st.Also = append(st.Also, basisOf(RPrivateRewrite)) // rule: private-segment-rewrite
	}
	return &MatchedPolicy{Rule: r, Owner: ev.Owner}, ec
}

func orDashInt(n int) any {
	if n == 0 {
		return "-"
	}
	return n
}

type fwCtx struct {
	src, dst netip.Addr
	proto    string
	port     int
	vlan     string
	app      string
}

func fwRuleMatch(m FirewallMatch, c fwCtx) (bool, string) {
	addr := func(spec string, a netip.Addr, what string) (bool, string) {
		if spec == "" || spec == "any" {
			return true, ""
		}
		p, ok := prefixOrAddr(spec)
		switch {
		case !ok:
			return false, "the " + what + " match " + spec + " is not an address or prefix"
		case !a.IsValid():
			return false, "the rule matches " + what + " " + spec + " and the flow carries none"
		case !p.Contains(a):
			return false, fmt.Sprintf("%s %s is outside %s", what, a, p)
		}
		return true, ""
	}
	if ok, why := addr(m.SourceIP, c.src, "source"); !ok {
		return false, why
	}
	if ok, why := addr(m.DestinationIP, c.dst, "destination"); !ok {
		return false, why
	}
	if m.Protocol != "" && !strings.EqualFold(m.Protocol, c.proto) {
		return false, fmt.Sprintf("protocol %q does not match %s", c.proto, m.Protocol)
	}
	if m.DestinationPort != "" {
		if c.port == 0 {
			return false, "the rule matches port " + m.DestinationPort + " and the flow carries none"
		}
		if ok, valid := portMatches(m.DestinationPort, c.port); !valid || !ok {
			return false, fmt.Sprintf("port %d does not match %s", c.port, m.DestinationPort)
		}
	}
	if m.VLAN != "" && m.VLAN != c.vlan {
		return false, fmt.Sprintf("VLAN %q does not match %s", c.vlan, m.VLAN)
	}
	if m.Application != "" && m.Application != c.app {
		return false, fmt.Sprintf("application %q does not match %s", c.app, m.Application)
	}
	return true, "every field of the rule matches"
}

// firewall is stage 3. Every rule is evaluated on its own; the verdict follows assumptions 85 and 86.
func (tc *traceCtx) firewall(ec *Effective, srcIP netip.Addr, iface *Interface, kind string, dpfx netip.Prefix) bool {
	st := tc.stage(2)
	f := tc.f
	st.Basis, st.Also = basisOf(RFwActions), bases(RFwPassThrough, RFwEdgePriority, RA85FwEvaluation, RA86FwDefault, RA90RuleFields) // rule: fw-actions // rule: fw-pass-through // rule: fw-edge-priority // rule: assume-85 // rule: assume-86 // rule: assume-90
	st.NotModelled = append(st.NotModelled, notModelled(RNMFwOrder, "Firewall rule order and the no-match action"))                  // rule: nm-fw-vs-bp-order
	if ec == nil {
		st.Status, st.Decision = TsFailed, "The Edge's firewall could not be computed."
		return true
	}
	stateful, _ := ec.Firewall.Stateful.Value.(bool)
	c := fwCtx{src: srcIP, proto: f.Protocol, port: f.Port, app: f.Application}
	if kind == "ip" {
		c.dst = dpfx.Addr()
	}
	if iface != nil && iface.VLAN != nil {
		c.vlan = fmt.Sprint(*iface.VLAN)
	}
	st.Input = []string{"protocol " + orDash(f.Protocol), fmt.Sprintf("port %v", orDashInt(f.Port)), "VLAN " + orDash(c.vlan), "Stateful Firewall " + map[bool]string{true: "on", false: "off"}[stateful]}
	rules := ec.Firewall.Rules
	order := make([]int, len(rules))
	for i := range order {
		order[i] = i
	}
	if tc.as.FirewallEvaluation == FwLastMatch {
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	verdicts := make([]FwVerdict, len(rules))
	for i, r := range rules {
		ok, why := fwRuleMatch(r.Rule.Match, c)
		verdicts[i] = FwVerdict{RuleID: r.Rule.ID, Group: r.Group, Owner: r.Owner.Name, Action: r.Rule.Action, Matched: ok, Reason: why, Basis: basisOf(RFwActions), Also: bases(RFwEdgePriority)}
	}
	decision, decisive := "", -1
	for _, i := range order {
		v := &verdicts[i]
		if !v.Matched {
			continue
		}
		switch v.Action {
		case FwSkip, FwReject:
			if !stateful {
				v.Reason += "; the action needs Stateful Firewall, which is off, so the rule is ignored" // rule: fw-reject-skip-stateful
				v.Also = append(v.Also, basisOf(RFwActionsStateful))
				continue
			}
			if v.Action == FwSkip {
				v.Reason += "; Skip goes to the next rule"
				continue
			}
		}
		decision, decisive = v.Action, i
		break
	}
	if decisive >= 0 {
		verdicts[decisive].Decisive = true
	}
	st.Firewall = verdicts
	if len(rules) > 0 && len(rules) > 1 {
		matches := 0
		for _, v := range verdicts {
			if v.Matched {
				matches++
			}
		}
		if matches > 1 {
			tc.s.use("fw_evaluation")
		}
	}
	for _, v := range verdicts {
		mark := "no match"
		if v.Decisive {
			mark = "DECIDES (" + v.Action + ")"
		} else if v.Matched {
			mark = "matches (" + v.Action + ")"
		}
		st.Details = append(st.Details, fmt.Sprintf("%s (%s group, %s): %s: %s", v.RuleID, v.Group, v.Owner, mark, v.Reason))
	}
	if stateful {
		st.Details = append(st.Details, "Stateful Firewall is on: matching is directional and asymmetric routing is not supported (checked at stage 8).") // rule: fw-stateful-symmetric
		st.Also = append(st.Also, basisOf(RFwSymmetric))
	}
	if decision == "" {
		tc.s.use("fw_default_action")
		decision = tc.as.FirewallDefaultAction
		st.Reason = fmt.Sprintf("No rule decides, so the assumed default action applies: %s.", decision)
	} else {
		st.Reason = fmt.Sprintf("Rule %s decides: %s.", verdicts[decisive].RuleID, decision)
	}
	switch decision {
	case FwDrop, FwReject:
		st.Status = TsBlocked
		st.Decision = map[string]string{FwDrop: "Blocked: the firewall drops the flow silently.", FwReject: "Blocked: the firewall rejects the flow (drop and send a reset)."}[decision]
		if tc.ignoreFw {
			return true
		}
		tc.stop(2, VerdictBlocked)
		return false
	}
	st.Status = TsOK
	st.Decision = "The firewall lets the flow through."
	return true
}

// route is stage 4: the route lookup (a destination Edge or Non SD-WAN destination named directly needs none).
func (tc *traceCtx) route(kind, id string, dpfx netip.Prefix, destEdge **edgeInfo, destNSD *string) bool {
	st := tc.stage(3)
	st.Basis, st.Also = basisOf(RRtOrderOfOps), bases(RRtLongestPrefix, RRtLocalRemote) // rule: rt-order-of-operations // rule: rt-longest-prefix // rule: rt-local-remote
	switch kind {
	case "edge":
		*destEdge = tc.s.c.edgeByID[id]
		st.Status, st.Decision, st.Reason = TsInfo, "The destination is the Edge "+(*destEdge).Edge.Name+"; no prefix lookup is needed.", "The flow names its destination Edge directly."
		return true
	case "nsd":
		*destNSD = id
		st.Status, st.Decision, st.Reason = TsInfo, "The destination is the Non SD-WAN destination "+id+"; no prefix lookup is needed.", "The flow names its destination directly."
		return true
	}
	rs := tc.s.routeSet()
	sel, cands := rs.Lookup(tc.src.Edge.ID, tc.seg, dpfx)
	st.Input = []string{"destination " + dpfx.String(), "segment " + tc.seg}
	st.Candidates = nonNilRoutes(cands)
	if len(cands) == 0 {
		q := ReachQuery{Source: tc.src.Edge.ID, Destination: dpfx.String(), Segment: tc.seg}
		rr, err := reachability(tc.t, tc.cond, tc.as.RoutingAssumptions, q, nil)
		st.Status, st.Decision = TsFailed, "No route: the Edge has no route for "+dpfx.String()+"."
		if err == nil {
			st.Reach, st.Reason = rr, rr.Reason
			st.LessonRef = rr.LessonRef
			for _, rs := range rr.Stages {
				if rs.ID == rr.FirstFailing {
					st.Basis, st.Also = rs.Basis, rs.Also
				}
			}
			for _, nm := range rr.NotModelled {
				st.NotModelled = append(st.NotModelled, nm)
			}
		}
		tc.stop(3, VerdictUnreachable)
		return false
	}
	r := sel
	if r == nil {
		c0 := cands[0]
		r = &c0
	}
	st.Route = r
	st.Basis, st.Also = r.Basis, r.Also
	st.Decision = fmt.Sprintf("The selected route is %s %s for %s (%s).", sourceTextOf(r), r.Prefix, dpfx, r.NextHop)
	st.Reason = r.Reason
	st.NotModelled = append(st.NotModelled, r.NotModelled...)
	switch {
	case !r.Reachable && r.Origin == "":
		st.Status = TsFailed
		st.Decision = fmt.Sprintf("The route %s is unreachable (%s).", r.Prefix, r.ReachableCode)
		st.Reason = r.ReachableReason
		tc.stop(3, VerdictUnreachable)
		return false
	case r.Origin == "" && r.Source != SrcNSDLocal:
		st.Status = TsOK
		st.Decision += " The destination is local to the source Edge."
		tc.skipFrom(4, "The route is local to the source Edge, so the flow never crosses the overlay.")
		return false
	case r.Source == SrcNSDLocal:
		*destNSD = r.Via
	default:
		*destEdge = tc.s.c.edgeByID[r.Origin]
	}
	st.Status = TsOK
	if !r.Reachable {
		st.Details = append(st.Details, "The route is marked unreachable ("+r.ReachableCode+"); stage 5 shows why no path is available.")
	}
	// LAN hosts only send the flow to the Edge when their routers learned the prefix: an Edge that runs OSPF or BGP toward
	// its LAN must redistribute the overlay route (lesson 8, check 4; assumption 83).
	if tc.src.runsLANRouting(tc.seg) && !r.isLocal() { // rule: assume-83
		okAds, all := tc.s.advertsCovering(tc.src.Edge.ID, r, dpfx)
		if len(okAds) == 0 {
			st.Status = TsFailed
			st.Decision = fmt.Sprintf("The route is in %s's table, but %s does not hand it to its LAN routers, so the LAN never sends the flow to the Edge.", tc.src.Edge.Name, tc.src.Edge.Name)
			st.Reason = "Overlay routes reach the LAN only through the Overlay Prefix (BGP) or Overlay Prefixes (OSPF) check box."
			st.Details = append(st.Details, advLines(all)...)
			st.Also = append(st.Also, bases(RRtOverlaySwitches, RA83StageOrder, RA22Redistribution)...) // rule: rt-overlay-redistribution-switches // rule: assume-83 // rule: assume-22
			tc.stop(3, VerdictUnreachable)
			return false
		}
		st.Details = append(st.Details, "The Edge sends the prefix to its LAN routers: "+okAds[0].Reason)
	}
	return true
}

// isLocal says whether the route is learned on the source Edge itself.
func (r *Route) isLocal() bool { return r.Origin == "" }

func sourceTextOf(r *Route) string { return sourceText(*r) }

// path is stage 5: SelectPath with the policy decision already made.
func (tc *traceCtx) path(pol *MatchedPolicy, destEdge *edgeInfo, destNSD string, out **PathDecision) bool {
	st := tc.stage(4)
	fl := Flow{Source: tc.src.Edge.ID, Application: tc.f.Application, TrafficClass: tc.f.TrafficClass, Segment: tc.seg, Matched: pol}
	if destNSD != "" {
		fl.Destination = FlowDest{Kind: "nsd", ID: destNSD}
	} else {
		fl.Destination = FlowDest{Kind: "edge", ID: destEdge.Edge.ID}
		if destEdge.Edge.ID == tc.src.Edge.ID {
			st.Status, st.Decision, st.Reason = TsSkipped, "Not needed.", "The destination is the source Edge itself."
			return false
		}
	}
	d, err := SelectPath(tc.t, tc.cond, fl, nil)
	if err != nil {
		st.Status, st.Decision, st.Reason = TsFailed, "The path could not be computed.", err.Error()
		tc.stop(4, VerdictUnreachable)
		return false
	}
	*out = d
	st.Path = d
	st.Input = []string{"from " + tc.src.Edge.Name, "to " + fl.Destination.Kind + " " + fl.Destination.ID, "network service " + strings.ReplaceAll(d.NetworkService, "_", " ")}
	st.Basis, st.Also = basisOf(RPathsMode), bases(RNetworkService)
	for _, nm := range d.NotModelled {
		if nm.Rule != RNMRoutingPrecedence { // the trace does its own route lookup at stage 4
			st.NotModelled = addNMUnique(st.NotModelled, nm)
		}
	}
	for _, o := range d.Options {
		mark := "rejected"
		if o.Chosen {
			mark = "CHOSEN"
		} else if o.Usable {
			mark = "usable, not chosen"
		}
		st.Details = append(st.Details, fmt.Sprintf("%s: %s: %s", o.Label, mark, o.Reason))
	}
	switch {
	case tc.cond.OrchestratorUnreachable:
		st.Status, st.Decision, st.Reason = TsNotModeled, "Not modelled: Edge behaviour with the Orchestrator unreachable.", d.Why
		tc.stop(4, VerdictNotModelled)
		return false
	case d.NetworkService == NetDirect:
		st.Status = TsOK
		st.Decision = "The matched rule's network service is Direct: the flow leaves the Edge outside the overlay."
		st.Reason = d.Why
		st.Basis = basisOf(RA63DirectNotOverlay) // rule: assume-63
		tc.skipFrom(5, "A Direct flow does not cross the overlay.")
		return false
	case d.Chosen == nil:
		st.Status = TsFailed
		st.Decision = "No usable path: every option is rejected."
		st.Reason = d.Why
		if len(d.Options) > 0 {
			st.Reason += " First rejected option: " + d.Options[0].Reason
		}
		tc.stop(4, VerdictUnreachable)
		return false
	}
	st.Basis, st.Also = d.Chosen.Basis, d.Chosen.Also
	if lc := d.Steering; lc != nil {
		st.Details = append(st.Details, "Link steering: "+lc.Reason)
		st.Also = append(st.Also, lc.Basis)
		st.Also = append(st.Also, lc.Also...)
		for _, nm := range lc.NotModelled {
			st.NotModelled = addNMUnique(st.NotModelled, nm)
		}
		switch lc.Action {
		case "drop":
			st.Status = TsBlocked
			st.Decision = "Blocked by the matched rule's link steering: " + d.Chosen.Label + " is the path, but the steered link is not available."
			st.Reason = lc.Reason
			tc.stop(4, VerdictBlocked)
			return false
		case "undetermined":
			st.Status = TsNotModeled
			st.Decision = "Not modelled: the steering outcome is not documented."
			st.Reason = lc.Reason
			tc.stop(4, VerdictNotModelled)
			return false
		}
	}
	st.Status = TsOK
	st.Decision = fmt.Sprintf("%s is chosen.", d.Chosen.Label)
	st.Reason = d.Why
	if d.Steering != nil && d.Steering.Chosen != nil {
		st.Decision += fmt.Sprintf(" The first leg uses link %s.", d.Steering.Chosen.LocalLink)
	}
	return true
}

// remediation is stage 6: documented conditions only, never a claim that a mechanism is active.
func (tc *traceCtx) remediation(pd *PathDecision) {
	st := tc.stage(5)
	f := tc.f
	st.Status = TsNotModeled
	st.Basis, st.Also = basisOf(RRemOnDemand), bases(RA67HoldNotApplied, RA68NoRemediation, RA89RemedClass) // rule: remed-on-demand // rule: assume-67 // rule: assume-68 // rule: assume-89
	st.NotModelled = append(st.NotModelled, notModelled(RNMRemedThresh, "Whether remediation is active"))   // rule: nm-remed-thresholds
	realTime := f.TrafficClass == "real_time"
	mechs := []RemedMech{
		{Name: "FEC", Candidate: f.Protocol == "udp" && realTime, Conditions: "Per flow, in real time, when loss crosses a threshold (the value is not given); applies to real-time traffic; switched off when loss no longer exceeds it. With Premium Service on the Gateway applies FEC to impaired traffic.", Basis: basisOf(RRemFEC), Also: bases(RRemPremium)}, // rule: remed-fec // rule: remed-premium
		{Name: "Jitter buffer", Candidate: f.Protocol == "udp" && realTime, Conditions: "Enabled for real-time applications when jitter exceeds 7 ms (voice) or 5 ms (video); the receiver buffers up to 10 packets or 200 ms; disabled if the flow is not constant.", Basis: basisOf(RRemJitter)},                                                                  // rule: remed-jitter-buffer
		{Name: "NACK", Candidate: f.Protocol == "tcp", Conditions: "For TCP applications such as file transfer: the receiver asks for a missing packet again when loss exceeds a per-application-type threshold (the value is not given).", Basis: basisOf(RRemNACK)},                                                                                               // rule: remed-nack
	}
	st.Remediation = mechs
	var cand []string
	for _, m := range mechs {
		if m.Candidate {
			cand = append(cand, m.Name)
		}
	}
	if len(cand) == 0 {
		st.Decision = "No remediation mechanism is a documented candidate for this flow's protocol and class."
	} else {
		st.Decision = "Documented candidates for this flow: " + strings.Join(cand, ", ") + ". Whether any is active right now is not modelled."
	}
	st.Reason = "On-demand remediation is used when steering cannot fix the problem (a single link, or several links degraded at once)."
	if pd != nil && pd.Chosen != nil && len(pd.Chosen.TunnelIDs) > 0 {
		for _, tn := range pd.Tunnels {
			if tn.ID != pd.Chosen.TunnelIDs[0] {
				continue
			}
			healthy := 0
			for _, p := range tn.Paths {
				if p.State == StUp {
					healthy++
				}
			}
			switch {
			case len(tn.Paths) <= 1:
				st.Details = append(st.Details, "The first leg has a single path: if it is degraded, steering cannot fix it and remediation is the documented answer.")
			case healthy == 0:
				st.Details = append(st.Details, "No path of the first leg is healthy: steering cannot fix it, which is the documented condition for remediation.")
			default:
				st.Details = append(st.Details, fmt.Sprintf("%d of %d paths of the first leg are healthy, so steering can move the flow first.", healthy, len(tn.Paths)))
			}
		}
	}
}

// mtu is stage 7: verified overhead numbers only.
func (tc *traceCtx) mtu(pd *PathDecision, nsd bool) {
	st := tc.stage(6)
	st.Basis = basisOf(RMTUVPN) // rule: mtu-vpn-overhead
	if pd == nil || pd.MTU == nil {
		st.Status = TsNotModeled
		st.Decision, st.Reason = "Not modelled: the effective MTU of this path.", "The overhead of tunnels to Non SD-WAN destinations was not found in the pages read."
		if nsd {
			st.NotModelled = append(st.NotModelled, notModelled(RNMNSDMTU, "MTU along the Non SD-WAN tunnel")) // rule: nm-nsd-mtu
		}
		return
	}
	m := pd.MTU
	st.MTU, st.Also = m, m.Also
	st.Status = TsOK
	st.Input = []string{fmt.Sprintf("path MTU %d (%s)", m.PathMTU, m.Limiting), fmt.Sprintf("VPN overhead %d to %d bytes", m.OverheadMin, m.OverheadMax)}
	st.Decision = fmt.Sprintf("Effective packet MTU %d to %d bytes (path MTU %d minus %d to %d bytes of IPsec overhead).", m.EffectiveMin, m.EffectiveMax, m.PathMTU, m.OverheadMin, m.OverheadMax)
	st.Reason = "VPN (IPsec) traffic adds 97 to 112 bytes; the exact figure depends on padding, so the range is shown."
	if n := tc.f.PacketSize; n > 0 {
		switch {
		case n <= m.EffectiveMin:
			st.Details = append(st.Details, fmt.Sprintf("A %d-byte packet fits.", n))
		case n > m.EffectiveMax:
			st.Details = append(st.Details, fmt.Sprintf("A %d-byte packet is larger than the effective MTU, so the Edge fragments it (IP fragmentation, RFC 791).", n))
			st.Also = append(st.Also, basisOf(RMTUFragment)) // rule: mtu-fragmentation
		default:
			st.Details = append(st.Details, fmt.Sprintf("A %d-byte packet is inside the overhead range (%d to %d): whether it fits depends on the padding of that packet.", n, m.EffectiveMin, m.EffectiveMax))
		}
	}
}

// returnPath is stage 8: only as far as the sheets support.
func (tc *traceCtx) returnPath(pd *PathDecision, srcIP netip.Addr, iface *Interface, destEdge *edgeInfo, destNSD string) {
	st := tc.stage(7)
	st.Basis = basisOf(RFwSymmetric) // rule: fw-stateful-symmetric
	if destNSD != "" || destEdge == nil || pd == nil || pd.Chosen == nil {
		st.Status = TsNotModeled
		st.Decision, st.Reason = "Not modelled: the return path.", "Return traffic from a destination outside the SD-WAN is not described."
		st.NotModelled = append(st.NotModelled, notModelled(RNMTraceReturn, "Return traffic")) // rule: nm-trace-return
		return
	}
	rev, err := SelectPath(tc.t, tc.cond, Flow{Source: destEdge.Edge.ID, Destination: FlowDest{Kind: "edge", ID: tc.src.Edge.ID}, Application: tc.f.Application, Segment: tc.seg}, nil)
	if err != nil || rev.Chosen == nil {
		st.Status = TsInfo
		st.Decision = "The reverse direction has no usable path."
		st.Reason = "Asymmetric reachability is a normal result; the model reports it and does not fail the forward flow."
		return
	}
	asym := rev.Chosen.Kind != pd.Chosen.Kind // rule: assume-88
	st.Status = TsOK
	st.Decision = fmt.Sprintf("The reverse direction uses %s; the forward direction uses %s.", rev.Chosen.Label, pd.Chosen.Label)
	st.Reason = "The reverse path is computed from the destination Edge with the same conditions."
	st.Also = append(st.Also, basisOf(RA88Asymmetry))
	st.Heuristic = HeuristicAsymmetry
	// Stateful Firewall: asymmetric routing is not supported (verified); what counts as asymmetric is assumption 88.
	ec, _ := EffectiveConfig(tc.t, tc.src.Edge.ID)
	if ec != nil {
		if on, _ := ec.Firewall.Stateful.Value.(bool); on && asym {
			st.Status = TsBlocked
			st.Decision = "Blocked (heuristic, may be wrong): Stateful Firewall is on and the return path differs from the forward path."
			st.Reason = "A stateful firewall enforces symmetric routing; asymmetric routing is not supported on an Edge with Stateful Firewall activated."
			tc.stop(7, VerdictBlocked)
		}
	}
}

var sourceLabels = map[string]string{SrcConnected: "connected", SrcLocalStatic: "static", SrcNSDLocal: "Non SD-WAN static", SrcNSDRemote: "remote Non SD-WAN static",
	SrcRemoteConnected: "remote connected", SrcRemoteStatic: "remote static", SrcOSPF: "OSPF", SrcBGP: "BGP", SrcRemoteDynamic: "remote dynamic"}

func sourceText(r Route) string {
	l := sourceLabels[r.Source]
	if l == "" {
		l = r.Source
	}
	if r.RouteType != "" {
		l += " " + r.RouteType
	}
	return l
}

// HeuristicAsymmetry is the visible label wherever assumption 88 (what counts as asymmetric) is applied.
const HeuristicAsymmetry = "Heuristic, may be wrong: the model calls a return path asymmetric when its kind of path differs from the forward path (assumption 88). Arista's documentation does not define the test."
