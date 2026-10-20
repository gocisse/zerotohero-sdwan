package sim

import (
	"fmt"
	"sort"
)

// Path option kinds.
const (
	OptDirect     = "direct-dynamic"
	OptHubTunnel  = "hub-tunnel"
	OptViaHub     = "via-hub"
	OptViaGateway = "via-gateway"
	OptNSDGateway = "nsd-via-gateway"
	OptNSDEdge    = "nsd-via-edge"
)

// FlowDest is the destination of a flow: an Edge or a Non SD-WAN destination. No prefixes: this step has no routing.
type FlowDest struct {
	Kind string `json:"kind"` // edge or nsd
	ID   string `json:"id"`
}

// Flow describes traffic only by source Edge, destination, application and traffic class.
type Flow struct {
	Source       string   `json:"source"`
	Destination  FlowDest `json:"destination"`
	Application  string   `json:"application,omitempty"`
	TrafficClass string   `json:"traffic_class,omitempty"` // real_time, transactional or bulk (informational)
	Segment      string   `json:"segment,omitempty"`       // default: the Global Segment
	// Matched lets a caller that already evaluated business policy (the flow tracer) hand the result over, so the
	// rules are evaluated in one place only.
	Matched *MatchedPolicy `json:"-"`
}

// MatchedPolicy is a business-policy result handed to SelectPath. Rule is nil when no rule matched.
type MatchedPolicy struct {
	Rule  *BusinessRule
	Owner string
}

// FlowError is returned for a flow that names things the topology does not have.
type FlowError struct{ Message string }

func (e *FlowError) Error() string { return e.Message }

// PathOption is one way the flow could travel, with its verdict.
type PathOption struct {
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	Via        string   `json:"via,omitempty"`
	TunnelIDs  []string `json:"tunnel_ids"`
	Usable     bool     `json:"usable"`
	Chosen     bool     `json:"chosen"`
	ReasonCode string   `json:"reason_code"`
	Reason     string   `json:"reason"`
	Basis      Basis    `json:"basis"`
	Also       []Basis  `json:"also,omitempty"`
}

// LinkChoice is the outcome of link steering on the source Edge's first leg.
type LinkChoice struct {
	Mode        string        `json:"mode"`
	Strictness  string        `json:"strictness,omitempty"`
	Configured  Steering      `json:"configured"`
	Action      string        `json:"action"` // use, move, drop, undetermined, none
	Chosen      *Path         `json:"chosen,omitempty"`
	Reason      string        `json:"reason"`
	Basis       Basis         `json:"basis"`
	Also        []Basis       `json:"also,omitempty"`
	NotModelled []NotModelled `json:"not_modelled,omitempty"`
}

// MTUResult is the effective packet MTU for VPN (IPsec) traffic along the chosen path.
type MTUResult struct {
	PathMTU      int     `json:"path_mtu"`
	Limiting     string  `json:"limiting"`
	OverheadMin  int     `json:"overhead_min"`
	OverheadMax  int     `json:"overhead_max"`
	EffectiveMin int     `json:"effective_min"`
	EffectiveMax int     `json:"effective_max"`
	Basis        Basis   `json:"basis"`
	Also         []Basis `json:"also,omitempty"`
}

// AssumedReachability states, loudly, that SelectPath does not use routes: reachability between sites is assumed
// there; the Reachability function answers the routing question.
type AssumedReachability struct {
	Assumed     bool        `json:"assumed"`
	Text        string      `json:"text"`
	NotModelled NotModelled `json:"not_modelled"`
}

// PathDecision is the answer of SelectPath.
type PathDecision struct {
	Flow           Flow                `json:"flow"`
	Reachability   AssumedReachability `json:"reachability"`
	RuleID         string              `json:"rule_id,omitempty"`
	RuleOwner      string              `json:"rule_owner,omitempty"`
	NetworkService string              `json:"network_service"`
	Options        []PathOption        `json:"options"`
	Chosen         *PathOption         `json:"chosen,omitempty"`
	Why            string              `json:"why"`
	Steering       *LinkChoice         `json:"steering,omitempty"`
	MTU            *MTUResult          `json:"mtu,omitempty"`
	Notes          []Note              `json:"notes"`
	NotModelled    []NotModelled       `json:"not_modelled"`
	Tunnels        []Tunnel            `json:"tunnels"`
}

func linkGroup(l *WanLink) string { // rule: assume-10
	switch {
	case l.Type == LinkPublic && l.Medium == MediumWired:
		return "public_wired"
	case l.Type == LinkPublic && l.Medium == MediumWireless:
		return "public_wireless"
	case l.Type == LinkPrivate && l.Medium == MediumWired:
		return "private_wired"
	}
	return ""
}

type tunnelIndex map[string]*Tunnel

func indexTunnels(ts *TunnelSet) tunnelIndex {
	m := tunnelIndex{}
	for i := range ts.Tunnels {
		m[ts.Tunnels[i].ID] = &ts.Tunnels[i]
	}
	return m
}

func usableState(s string) bool { return s == StUp || s == StDegraded }

// SelectPath decides which path a flow takes: the ordered options, the chosen one, every rejected one with its
// reason, link steering on the first leg, and the effective MTU. Reachability between sites is ASSUMED (routing is
// not modelled yet).
func SelectPath(t *Topology, cond Conditions, flow Flow, clock Clock) (*PathDecision, error) {
	c, err := newOverlayCtx(t, cond)
	if err != nil {
		return nil, err
	}
	src := c.edgeByID[flow.Source]
	if src == nil {
		return nil, &FlowError{fmt.Sprintf("unknown source Edge %q", flow.Source)}
	}
	switch flow.Destination.Kind {
	case "edge":
		if c.edgeByID[flow.Destination.ID] == nil {
			return nil, &FlowError{fmt.Sprintf("unknown destination Edge %q", flow.Destination.ID)}
		}
		if flow.Destination.ID == flow.Source {
			return nil, &FlowError{"the destination must differ from the source"}
		}
	case "nsd":
		if c.nsdByID[flow.Destination.ID] == nil {
			return nil, &FlowError{fmt.Sprintf("unknown Non SD-WAN destination %q", flow.Destination.ID)}
		}
	default:
		return nil, &FlowError{`destination kind must be "edge" or "nsd"`}
	}
	if flow.Segment == "" {
		flow.Segment = GlobalSegmentID
	}
	ts, err := ComputeTunnels(t, cond, clock)
	if err != nil {
		return nil, err
	}
	tix := indexTunnels(ts)

	d := &PathDecision{Flow: flow, Options: []PathOption{}, Notes: []Note{}, NotModelled: []NotModelled{}, Tunnels: ts.Tunnels}
	d.Reachability = AssumedReachability{Assumed: true, Text: "Reachability between the two sites is assumed here: this decision uses tunnels and paths only. Ask the reachability explainer whether a route exists.",
		NotModelled: notModelled(RNMRoutingPrecedence, "Route and prefix reachability")} // rule: nm-routing-precedence
	addNM := func(rule, item string) NotModelled {
		nm := notModelled(rule, item)
		for _, x := range d.NotModelled {
			if x.Rule == rule {
				return nm
			}
		}
		d.NotModelled = append(d.NotModelled, nm)
		return nm
	}
	addNM(RNMRoutingPrecedence, "Route and prefix reachability")

	if cond.OrchestratorUnreachable { // rule: nm-orchestrator-down
		nm := addNM(RNMOrchestratorDown, "Edge behaviour with the Orchestrator unreachable")
		d.Why = "not modelled: " + nm.Reason
		d.NetworkService = "unknown"
		return d, nil
	}

	// 1. Business policy: which rule applies and what network service and steering it asks for.
	ec, err := EffectiveConfig(t, src.Edge.ID)
	if err != nil {
		return nil, err
	}
	rule, owner := matchRule(ec, flow)
	if flow.Matched != nil {
		rule, owner = flow.Matched.Rule, flow.Matched.Owner
	}
	st := Steering{}
	d.NetworkService = NetMultipath // rule: assume-2
	if rule != nil {
		d.RuleID, d.RuleOwner = rule.ID, owner
		st = rule.Action.Steering
		if rule.Action.NetworkService != "" {
			d.NetworkService = rule.Action.NetworkService
		}
		d.Notes = append(d.Notes, Note{Text: fmt.Sprintf("Rule %s (%s) matched. The order in which rules are evaluated is not documented; the simulator takes the first match with Edge rules above Profile rules.", rule.ID, owner), Basis: basisOf(RA1FirstMatch)})
		addNM(RNMBizOrder, "Business rule evaluation order")
	} else {
		d.Notes = append(d.Notes, Note{Text: "No business rule matched: Smart Defaults would apply; the simulator treats it as Multi-Path with Auto steering.", Basis: basisOf(RA2NoRuleDefault)})
	}
	if d.NetworkService == NetDirect { // rule: assume-63
		d.Why = "The matched rule's network service is Direct: the flow leaves outside the overlay, so no overlay path is chosen."
		d.Notes = append(d.Notes, Note{Text: d.Why, Basis: basisOf(RA63DirectNotOverlay)})
		return d, nil
	}

	// 2. Options.
	switch flow.Destination.Kind {
	case "edge":
		c.edgeOptions(d, src, c.edgeByID[flow.Destination.ID], tix, addNM)
	case "nsd":
		c.nsdOptions(d, src, c.nsdByID[flow.Destination.ID], tix, addNM)
	}

	// 3. Chosen: the first usable option in order.
	for i := range d.Options {
		if d.Options[i].Usable {
			d.Options[i].Chosen = true
			d.Chosen = &d.Options[i]
			break
		}
	}
	if d.Chosen == nil {
		d.Why = "No option is usable, so the flow has no path."
		return d, nil
	}
	d.Why = fmt.Sprintf("%s is the first usable option in the order direct tunnel, Hub tunnel, standing path.", d.Chosen.Label)
	if d.Chosen.Kind == OptDirect { // rule: assume-14
		d.Why = "A direct dynamic tunnel exists and is usable, so the flow uses it (assumed to be preferred over the standing path)."
		d.Notes = append(d.Notes, Note{Text: "Whether a flow prefers an existing direct tunnel is not documented.", Basis: basisOf(RA14DirectFirst)})
		addNM(RNMDirectVsHub, "Direct tunnel versus Hub or Gateway path") // rule: nm-direct-vs-hub
	}

	// 4. Link steering on the source Edge's first leg.
	if first := tix[d.Chosen.TunnelIDs[0]]; first != nil {
		lc := c.steer(first.Paths, st, src)
		d.Steering = &lc
		for _, nm := range lc.NotModelled {
			addNM(nm.Rule, nm.Item)
		}
		if lc.Action == "drop" || lc.Action == "undetermined" {
			d.Notes = append(d.Notes, Note{Text: "Link steering left the flow without a path: " + lc.Reason, Basis: lc.Basis})
		}
		// 5. MTU.
		if flow.Destination.Kind == "edge" && lc.Chosen != nil {
			d.MTU = c.mtuFor(first, lc.Chosen)
		} else if flow.Destination.Kind == "nsd" {
			addNM(RNMNSDMTU, "MTU along the Non SD-WAN tunnel") // rule: nm-nsd-mtu
		}
	}
	d.Notes = append(d.Notes, Note{Text: "Load balancing across links is not modelled: one path is chosen.", Basis: basisOf(RNMLoadBalancing)}) // rule: nm-load-balancing
	addNM(RNMLoadBalancing, "Load balancing across links")
	return d, nil
}

// matchRule finds the business rule the flow matches: Edge rules first, then Profile rules that are not
// superseded, first match wins (assumptions 1 and 3). Rules that match on fields a flow does not carry never match.
func matchRule(ec *Effective, flow Flow) (*BusinessRule, string) {
	ev := evaluateRules(ec, RuleCtx{Segment: flow.Segment, Application: flow.Application, DestKind: flow.Destination.Kind}, true)
	return ev.Matched, ev.Owner
}

func optionBasis(kind string) (Basis, []Basis) {
	switch kind {
	case OptDirect:
		return basisOf(RDynFirstPacket), bases(RA14DirectFirst, RPathsMode)
	case OptHubTunnel:
		return basisOf(RB2HPermanent), bases(RA60HubTunnelDirect) // rule: assume-60
	case OptViaHub:
		return basisOf(RB2BOptions), bases(RB2BHubs64, RHubOrder, RA59HubAttrsEqual, RA61HubsOnly) // rule: b2b-hubs-6-4 // rule: assume-59
	case OptViaGateway:
		return basisOf(RB2BGateway), bases(RB2BOptions, RA61HubsOnly) // rule: b2b-via-gateway
	case OptNSDGateway:
		return basisOf(RNSDGatewayEnable), bases(RA15NSDByPolicy, RNetworkService) // rule: network-service-decides // rule: assume-15
	default:
		return basisOf(RNSDEdgeSegment), bases(RA15NSDByPolicy, RNetworkService)
	}
}

// legVerdict checks that every tunnel of an option is usable and returns the first failure.
func legVerdict(ids []string, tix tunnelIndex) (bool, string, string) {
	for _, id := range ids {
		t := tix[id]
		if t == nil {
			return false, CodeNoLinkPair, "tunnel " + id + " does not exist"
		}
		if !usableState(t.State) {
			return false, t.ReasonCode, fmt.Sprintf("%s is %s: %s", describeTunnel(t), t.State, t.Reason)
		}
	}
	return true, CodeOK, "every tunnel on the path is up or degraded"
}

func describeTunnel(t *Tunnel) string {
	return fmt.Sprintf("the %s tunnel %s to %s", t.Type, t.A.Name, t.B.Name)
}

func (d *PathDecision) add(kind, label, via string, ids []string, usable bool, code, reason string) {
	b, also := optionBasis(kind)
	if ids == nil {
		ids = []string{}
	}
	d.Options = append(d.Options, PathOption{Kind: kind, Label: label, Via: via, TunnelIDs: ids, Usable: usable, ReasonCode: code, Reason: reason, Basis: b, Also: also})
}

func (c *overlayCtx) edgeOptions(d *PathDecision, src, dst *edgeInfo, tix tunnelIndex, addNM func(string, string) NotModelled) {
	a, b := src.Edge.ID, dst.Edge.ID
	// Direct dynamic tunnel.
	dyn := tix["dynamic:"+a+":"+b]
	if dyn == nil {
		dyn = tix["dynamic:"+b+":"+a]
	}
	if dyn != nil {
		ok := usableState(dyn.State)
		reason := dyn.Reason
		if ok {
			reason = "a direct dynamic tunnel exists: " + dyn.Reason
		}
		d.add(OptDirect, "Direct dynamic tunnel", "", []string{dyn.ID}, ok, dyn.ReasonCode, reason)
	}
	// Static Hub tunnel to the destination, when it is one of the source's Hubs.
	for _, h := range d.hubTunnelsFrom(src, tix) {
		if h.B.ID == b {
			ok := usableState(h.State)
			d.add(OptHubTunnel, "Static Hub tunnel to "+dst.Edge.Name, h.B.ID, []string{h.ID}, ok, h.ReasonCode, h.Reason)
		}
	}
	// Standing branch-to-branch path.
	switch {
	case !src.CVPN.Enabled || !dst.CVPN.Enabled:
		d.add(OptViaGateway, "Standing path", "", nil, false, CodeCloudVPNOff, "Cloud VPN is not enabled on both Edges, so there is no Branch to Branch path")
	case !src.CVPN.B2B.Enabled || !dst.CVPN.B2B.Enabled:
		d.add(OptViaGateway, "Standing path", "", nil, false, CodeB2BOff, "Branch to Branch VPN is not enabled on both Edges, so there is no standing path")
	case src.CVPN.B2B.Via == B2BHubsForVPN: // rule: assume-61
		hubs := d.hubTunnelsFrom(src, tix)
		if len(hubs) == 0 {
			d.add(OptViaHub, "Via a Hub", "", nil, false, CodeB2HOff, "Hubs for VPN is chosen but the source Edge has no Hub tunnel")
		}
		for _, h := range hubs {
			leg2 := tix[fmt.Sprintf("hub:%s:%s", b, h.B.ID)] // the destination's tunnel to the same Hub
			label := fmt.Sprintf("Via Hub %s (%s)", h.B.Name, h.Role)
			if leg2 == nil {
				d.add(OptViaHub, label, h.B.ID, []string{h.ID}, false, CodeB2HOff, dst.Edge.Name+" has no tunnel to this Hub")
				continue
			}
			ok, code, reason := legVerdict([]string{h.ID, leg2.ID}, tix)
			d.add(OptViaHub, label, h.B.ID, []string{h.ID, leg2.ID}, ok, code, reason)
		}
		if len(hubs) > 1 {
			d.Notes = append(d.Notes, Note{Text: "Hub order decides between Hubs when their routing attributes are equal; routing attributes are not modelled.", Basis: basisOf(RHubOrder)})
		}
	default: // Cloud Gateways (rule: b2b-via-gateway)
		var common []string
		for _, ga := range src.Gateways[:] {
			for _, gb := range dst.Gateways[:] {
				if ga != "" && ga == gb && !hasString(common, ga) {
					common = append(common, ga)
				}
			}
		}
		if len(common) == 0 {
			d.add(OptViaGateway, "Via the Gateway", "", nil, false, CodeNoGateway, "the two Edges share no Gateway and Gateway-to-Gateway forwarding is not described")
			break
		}
		for _, g := range common {
			ids := []string{"gateway:" + a + ":" + g, "gateway:" + b + ":" + g}
			ok, code, reason := legVerdict(ids, tix)
			d.add(OptViaGateway, "Via Gateway "+g, g, ids, ok, code, reason)
		}
	}
	// If a primary Hub is unreachable and the secondary is used, say which rule that is.
	for i := range d.Options {
		if d.Options[i].Kind == OptViaHub && i > 0 && d.Options[i].Usable && !d.Options[i-1].Usable && d.Options[i-1].Kind == OptViaHub {
			d.Options[i].Also = append(d.Options[i].Also, basisOf(RHubFailover)) // rule: hub-failover
			d.Notes = append(d.Notes, Note{Text: "The primary Hub is not usable, so the Spoke uses the secondary Hub for its local flows.", Basis: basisOf(RHubFailover)})
			addNM(RNMHubReturnLocal, "Moving local flows back when the primary Hub returns")
		}
	}
}

// hubTunnelsFrom lists a source Edge's Hub tunnels in Hub order.
func (d *PathDecision) hubTunnelsFrom(src *edgeInfo, tix tunnelIndex) []*Tunnel {
	var out []*Tunnel
	for _, t := range tix {
		if t.Type == TunHub && t.A.ID == src.Edge.ID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HubIndex < out[j].HubIndex })
	return out
}

func (c *overlayCtx) nsdOptions(d *PathDecision, src *edgeInfo, n *NSD, tix tunnelIndex, addNM func(string, string) NotModelled) {
	if n.Via == NSDViaGateway {
		var first *Tunnel
		for _, t := range tix {
			if t.Type == TunNSDGateway && t.B.ID == n.ID && usableState(t.State) && t.Role != "standby" {
				if first == nil || t.ID < first.ID {
					first = t
				}
			}
		}
		switch {
		case !src.CVPN.Enabled || !hasString(src.CVPN.NSDGateway, n.ID):
			d.add(OptNSDGateway, "NSD via Gateway "+n.ID, n.Gateway, nil, false, CodeNSDNotEnabled, "the source Edge's Profile does not enable this destination (Cloud VPN and Edge to Non SD-WAN via Gateway)")
		case src.Gateways[0] != n.Gateway && src.Gateways[1] != n.Gateway:
			d.add(OptNSDGateway, "NSD via Gateway "+n.ID, n.Gateway, nil, false, CodeNoGateway, "the destination is built by Gateway "+n.Gateway+", which is not one of this Edge's Gateways; Gateway-to-Gateway forwarding is not described")
		case first == nil:
			d.add(OptNSDGateway, "NSD via Gateway "+n.ID, n.Gateway, nil, false, CodePeerUnreachable, "no active tunnel to the destination is up")
		default:
			ids := []string{"gateway:" + src.Edge.ID + ":" + n.Gateway, first.ID}
			ok, code, reason := legVerdict(ids, tix)
			d.add(OptNSDGateway, "NSD via Gateway "+n.ID, n.Gateway, ids, ok, code, reason)
		}
		return
	}
	id := fmt.Sprintf("nsd-edge:%s:%s", src.Edge.ID, n.ID)
	switch {
	case n.Edge != src.Edge.ID:
		d.add(OptNSDEdge, "NSD via Edge "+n.ID, n.Edge, nil, false, CodeNSDNotEnabled, "the destination's tunnel is built by Edge "+n.Edge+"; reaching it from another Edge needs routing, which is not modelled yet")
	case tix[id] == nil:
		d.add(OptNSDEdge, "NSD via Edge "+n.ID, n.Edge, nil, false, CodeNSDNotEnabled, "no tunnel is built")
	default:
		ok, code, reason := legVerdict([]string{id}, tix)
		d.add(OptNSDEdge, "NSD via Edge "+n.ID, n.Edge, []string{id}, ok, code, reason)
	}
}

// steer applies link steering to the paths of the first leg, following the Section 3 semantics.
func (c *overlayCtx) steer(paths []Path, st Steering, e *edgeInfo) LinkChoice {
	mode := st.Mode
	if mode == "" {
		mode = SteerAuto // rule: steering-auto-default
	}
	lc := LinkChoice{Mode: mode, Strictness: st.Strictness, Configured: st, Basis: basisOf(RSteerModes), Also: bases(RA62FirstLeg, RSteerStrictness)} // rule: steering-modes // rule: steering-strictness // rule: assume-62
	linkOf := func(p Path) *WanLink { return c.linkByID[p.LocalLink] }
	firstState := func(list []Path, state string) *Path {
		for i := range list {
			if list[i].State == state {
				return &list[i]
			}
		}
		return nil
	}
	auto := func(list []Path) *Path { // rule: assume-56
		if p := firstState(list, StUp); p != nil {
			return p
		}
		return firstState(list, StDegraded)
	}
	if mode == SteerAuto { // rule: steering-auto-best
		lc.Basis, lc.Also = basisOf(RSteerAutoBest), bases(RA56AutoDeterministic, RA55DegradedState)
		if p := auto(paths); p != nil {
			lc.Action, lc.Chosen = "use", p
			lc.Reason = fmt.Sprintf("Auto picks the best usable link; %s is %s. DMPO's measurement-based choice among equal links is not modelled.", p.LocalLink, p.State)
		} else {
			lc.Action, lc.Reason = "drop", "no path of the tunnel is up or degraded"
		}
		lc.NotModelled = append(lc.NotModelled, notModelled(RNMSLAThresholds, "What makes a link degraded")) // rule: nm-sla-thresholds
		return lc
	}
	// The set of links the rule points at.
	set := map[string]bool{}
	switch mode {
	case SteerInterface:
		for _, in := range e.Edge.Interfaces {
			if in.ID == st.Interface && in.Kind == IfWan {
				set[in.Link] = true
			}
		}
	case SteerWanLink:
		set[st.Link] = true
	case SteerGroup:
		for _, l := range e.Links {
			if linkGroup(l.Link) == st.Group {
				set[l.Link.ID] = true
			}
		}
		lc.Also = append(lc.Also, basisOf(RA57GroupStrictness), basisOf(RA10TransportGroup))
	}
	var pref, others []Path
	for _, p := range paths {
		if set[p.LocalLink] {
			pref = append(pref, p)
		} else {
			others = append(others, p)
		}
	}
	strict := st.Strictness
	if strict == "" {
		strict = "available" // rule: assume-57
	}
	lc.Strictness = strict
	if len(pref) == 0 {
		lc.Action, lc.Reason = "undetermined", "none of the tunnel's paths uses the link the rule points at"
		return lc
	}
	switch strict {
	case "mandatory": // rule: steering-mandatory
		lc.Basis = basisOf(RSteerMandatory)
		if p := auto(pref); p != nil {
			lc.Action, lc.Chosen = "use", p
			lc.Reason = fmt.Sprintf("Mandatory pins the flow to %s (%s). It never steers away, even if the SLA is missed.", p.LocalLink, p.State)
		} else {
			lc.Action = "drop"
			lc.Reason = "Mandatory: the chosen link is down, so the packets are dropped; the flow never moves to another link."
		}
	case "preferred": // rule: steering-preferred
		lc.Basis = basisOf(RSteerPreferred)
		if p := firstState(pref, StUp); p != nil {
			lc.Action, lc.Chosen = "use", p
			lc.Reason = fmt.Sprintf("Preferred: %s is up and meets its SLA, so the flow stays.", p.LocalLink)
			break
		}
		prefPrivate := false
		for _, p := range pref {
			if l := linkOf(p); l != nil && l.Type == LinkPrivate {
				prefPrivate = true
			}
		}
		if prefPrivate { // rule: nm-preferred-private
			nm := notModelled(RNMPreferredPrivate, "Preferred steering on a private link")
			lc.Action, lc.NotModelled = "undetermined", append(lc.NotModelled, nm)
			lc.Reason = nm.Reason
			break
		}
		var pubUp, privUp *Path
		for i := range others {
			l := linkOf(others[i])
			if others[i].State == StUp && l != nil {
				if l.Type == LinkPublic && pubUp == nil {
					pubUp = &others[i]
				}
				if l.Type == LinkPrivate && privUp == nil {
					privUp = &others[i]
				}
			}
		}
		switch {
		case pubUp != nil:
			lc.Action, lc.Chosen = "move", pubUp
			lc.Reason = fmt.Sprintf("Preferred: the chosen link is %s, so the flow moves to the public link %s.", pref[0].State, pubUp.LocalLink)
		case privUp != nil:
			lc.Action, lc.Chosen = "move", privUp
			lc.Reason = fmt.Sprintf("Preferred: the chosen link and the other public links are down or degraded, so the flow may use the private link %s.", privUp.LocalLink)
		case firstState(pref, StDegraded) != nil:
			p := firstState(pref, StDegraded)
			lc.Action, lc.Chosen = "use", p
			lc.Reason = "Preferred: nothing better is up, so the flow stays on the degraded chosen link."
		default:
			lc.Action, lc.Reason = "drop", "Preferred: no link of the tunnel is up or degraded."
		}
	default: // available (rule: steering-available)
		lc.Basis = basisOf(RSteerAvailable) // rule: steering-available
		if p := auto(pref); p != nil {
			lc.Action, lc.Chosen = "use", p
			lc.Reason = fmt.Sprintf("Available: %s is up (%s), so the flow uses it whatever the SLA.", p.LocalLink, p.State)
		} else if p := auto(others); p != nil { // rule: assume-58
			lc.Action, lc.Chosen = "move", p
			lc.Reason = fmt.Sprintf("Available: the chosen link is down, so the flow moves to %s.", p.LocalLink)
			lc.Also = append(lc.Also, basisOf(RA58AvailableMoves))
		} else {
			lc.Action, lc.Reason = "drop", "Available: no link of the tunnel is up or degraded."
		}
	}
	// Documented one-minute hold and remediation are not applied by a single stateless decision.
	if lc.Chosen != nil && mode != SteerAuto && lc.Action == "use" { // rule: steering-hold-time
		lc.Also = append(lc.Also, basisOf(RSteerHold), basisOf(RA67HoldNotApplied)) // rule: assume-67
	}
	if lc.Chosen != nil && lc.Chosen.State == StDegraded { // rule: assume-68
		lc.Also = append(lc.Also, basisOf(RA68NoRemediation))
	}
	return lc
}

// mtuFor computes the effective packet MTU for VPN (IPsec) traffic on the chosen first-leg path.
func (c *overlayCtx) mtuFor(t *Tunnel, p *Path) *MTUResult {
	mtu := func(id string) int {
		if l := c.linkByID[id]; l != nil && l.MTU != nil {
			return *l.MTU // rule: assume-64
		}
		return 1500 // rule: mtu-default-1500
	}
	local := mtu(p.LocalLink)
	limit := "link " + p.LocalLink
	path := local
	if t.B.Kind == "edge" {
		if r := mtu(p.Remote); r < path {
			path, limit = r, "link "+p.Remote
		}
	}
	if path > 1500 { // rule: mtu-no-jumbo
		path, limit = 1500, "the 1500-byte maximum"
	}
	const oMin, oMax = 97, 112                                                                                                                  // rule: mtu-vpn-overhead
	r := &MTUResult{PathMTU: path, Limiting: limit, OverheadMin: oMin, OverheadMax: oMax, EffectiveMin: path - oMax, EffectiveMax: path - oMin, // rule: mtu-derived
		Basis: basisOf(RMTUVPN), Also: bases(RMTUDerived, RMTUDefault, RMTUNoJumbo, RA64LinkMTU, RA66VPNOverhead)} // rule: assume-66
	if path < 1300 { // rule: mtu-1300-floor
		r.Also = append(r.Also, basisOf(RMTUFloor))
	}
	return r
}
