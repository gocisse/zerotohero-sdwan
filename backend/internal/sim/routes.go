package sim

import (
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

// Route sources shown in a route table.
const (
	SrcConnected       = "connected"
	SrcLocalStatic     = "local-static"
	SrcNSDLocal        = "nsd-local"
	SrcNSDRemote       = "nsd-remote"
	SrcRemoteConnected = "remote-connected"
	SrcRemoteStatic    = "remote-static"
	SrcOSPF            = "ospf"
	SrcBGP             = "bgp"
	SrcRemoteDynamic   = "remote-dynamic"
)

// Route types of dynamic routes (the labels the guides use).
const (
	TypeO   = "O"
	TypeOIA = "O IA"
	TypeOE1 = "O E1"
	TypeOE2 = "O E2"
	TypeEB  = "eBGP"
	TypeIB  = "iBGP"
)

// Reachability reason codes the model uses; each is a code the guides name.
const (
	CodePRUnreachable = "PR_UNREACHABLE"
	CodeSLAStateDown  = "SLA_STATE_DOWN"
)

// Route is one candidate route in one Edge's table for one segment, with everything the Route Table Dump would
// show that the model can honestly fill, plus the model's own reason and evidence.
type Route struct {
	ID        string `json:"id"`
	Edge      string `json:"edge"`
	Segment   string `json:"segment"`
	Prefix    string `json:"prefix"`
	Source    string `json:"source"`
	RouteType string `json:"route_type,omitempty"`
	// Origin is the Edge that owns the route (empty for a route learned on this Edge).
	Origin          string `json:"origin,omitempty"`
	OriginName      string `json:"origin_name,omitempty"`
	Via             string `json:"via,omitempty"` // interface, neighbor, static route or NSD id
	NextHop         string `json:"next_hop"`
	Cost            *int   `json:"cost,omitempty"`
	Preference      *int   `json:"preference,omitempty"`
	PreferenceNote  string `json:"preference_note,omitempty"`
	LocalPreference *int   `json:"local_preference,omitempty"`
	Metric          *int   `json:"metric,omitempty"` // BGP MED
	ASPath          []int  `json:"as_path,omitempty"`

	Reachable       bool   `json:"reachable"`
	ReachableCode   string `json:"reachable_code,omitempty"`
	ReachableReason string `json:"reachable_reason,omitempty"`
	Selected        bool   `json:"selected"`
	Rank            int    `json:"rank"`
	LostReasonCode  string `json:"lost_reason_code,omitempty"`
	Reason          string `json:"reason"`

	Basis       Basis         `json:"basis"`
	Also        []Basis       `json:"also,omitempty"`
	NotModelled []NotModelled `json:"not_modelled,omitempty"`
	// Notes record filter actions and other things done to the route's attributes.
	Notes []string `json:"notes,omitempty"`

	pfx    netip.Prefix
	owner  *originRoute
	exitID string
	// cmpCost is the cost used to compare routes (the owner's static cost or OSPF metric); Cost is what the table shows.
	cmpCost *int
}

// SegmentTable is one segment's table on one Edge.
type SegmentTable struct {
	Segment string  `json:"segment"`
	Routes  []Route `json:"routes"`
}

// Withheld is a route a LAN router offers (or a static route) that never reached the Edge's table, with the stage
// and the reason.
type Withheld struct {
	Edge    string  `json:"edge"`
	Segment string  `json:"segment"`
	Prefix  string  `json:"prefix"`
	Source  string  `json:"source"`
	Via     string  `json:"via"`
	Stage   string  `json:"stage"` // known or filtered
	Reason  string  `json:"reason"`
	Basis   Basis   `json:"basis"`
	Also    []Basis `json:"also,omitempty"`
}

// EdgeRoutes is everything computed for one Edge.
type EdgeRoutes struct {
	Edge     string         `json:"edge"`
	Name     string         `json:"name"`
	Segments []SegmentTable `json:"segments"`
	// LANAdvertisements lists what the Edge redistributes to its LAN routers and why each entry is or is not sent.
	LANAdvertisements []LANAdvertisement `json:"lan_advertisements"`
	Withheld          []Withheld         `json:"withheld"`
}

// RouteSet is the answer of ComputeRoutes.
type RouteSet struct {
	Topology    string             `json:"topology"`
	Conditions  Conditions         `json:"conditions"`
	Assumptions []AssumptionSwitch `json:"assumptions"`
	Edges       []EdgeRoutes       `json:"edges"`
	NotModelled []NotModelled      `json:"not_modelled"`
}

// originRoute is a route as its owning Edge knows it, before it is installed or advertised. Reachability reads it
// to explain each stage of the route's life.
type originRoute struct {
	edge    *edgeInfo
	segment string
	pfx     netip.Prefix
	kind    string // connected, static, ospf, bgp, nsd
	id      string
	via     string

	static *StaticRoute
	lan    *LANRoute
	nsd    *NSD
	nsdPfx *NSDPrefix
	nbr    *BGPNeighbor

	routeType string
	cost      *int
	attrs     attrs

	known       bool
	knownWhy    string
	knownBasis  Basis
	knownAlso   []Basis
	installed   bool
	advOK       bool
	advWhy      string
	advBasis    Basis
	advAlso     []Basis
	held        bool
	heldStage   string // filtered
	heldWhy     string
	heldBasis   Basis
	heldAlso    []Basis
	reachable   bool
	reachCode   string
	reachReason string
	notes       []string
	advertised  bool // finally enters the overlay
}

type attrs struct {
	localPref *int
	med       *int
	asPath    []int
}

type pathVerdict struct {
	ok     bool
	label  string
	reason string
	option *PathOption
}

type rtState struct {
	t    *Topology
	c    *overlayCtx
	ts   *TunnelSet
	tix  tunnelIndex
	cond Conditions
	as   RoutingAssumptions

	used   map[string]bool
	nms    []NotModelled
	nmSeen map[string]bool

	flagStatic, flagConnected, flagExtOSPF bool
	origins                                []*originRoute
	byEdge                                 map[string][]*originRoute
	paths                                  map[string]pathVerdict
	exits                                  map[string][]string
	pins                                   map[string]map[string]int
	nbDown, probeDown                      map[string]bool
	tables                                 map[string][]*Route // edge|segment -> candidates
	adverts                                map[string][]LANAdvertisement
	withheld                               map[string][]Withheld
}

func (s *rtState) nm(rule, item string) NotModelled {
	n := notModelled(rule, item)
	if !s.nmSeen[rule] {
		s.nmSeen[rule] = true
		s.nms = append(s.nms, n)
	}
	return n
}

func (s *rtState) use(key string) { s.used[key] = true }

func pfxOf(str string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(str)
	if err != nil || !p.Addr().Is4() {
		return netip.Prefix{}, false
	}
	return p.Masked(), true
}

func pfxContains(outer, inner netip.Prefix) bool {
	return outer.Bits() <= inner.Bits() && outer.Contains(inner.Addr())
}

// overlap: two prefixes overlap when one contains the other (assumption 79).
func overlap(a, b netip.Prefix) bool { // rule: assume-79
	return pfxContains(a, b) || pfxContains(b, a)
}

func segOr(seg string) string {
	if seg == "" {
		return GlobalSegmentID
	}
	return seg
}

func intp(i int) *int { return &i }

// cvpnOn says whether Cloud VPN is enabled on the Edge for the segment.
func (e *edgeInfo) cvpnOn(seg string) bool {
	return e.CVPN.Enabled && hasString(e.CVPN.Segments, seg)
}

func (e *edgeInfo) inProfile(seg string) bool { return hasString(e.Profile.Segments, seg) }

// ospfProfile returns the Profile's OSPF settings when OSPF is activated.
func (e *edgeInfo) ospfProfile() *OSPFProfile {
	if e.Profile.Routing != nil && e.Profile.Routing.OSPF != nil && e.Profile.Routing.OSPF.Enabled { // rule: rt-ospf-profile-first
		return e.Profile.Routing.OSPF
	}
	return nil
}

func (e *edgeInfo) routing() *EdgeRouting {
	if e.Edge.Routing == nil {
		return &EdgeRouting{}
	}
	return e.Edge.Routing
}

func (e *edgeInfo) ifaceByID(id string) *Interface {
	for i := range e.Edge.Interfaces {
		if e.Edge.Interfaces[i].ID == id {
			return &e.Edge.Interfaces[i]
		}
	}
	return nil
}

func (e *edgeInfo) ospfIface(id string) *OSPFInterface {
	r := e.routing()
	if r.OSPF == nil {
		return nil
	}
	for i := range r.OSPF.Interfaces {
		if r.OSPF.Interfaces[i].ID == id {
			return &r.OSPF.Interfaces[i]
		}
	}
	return nil
}

func (e *edgeInfo) bgpNeighbor(id string) *BGPNeighbor {
	r := e.routing()
	if r.BGP == nil {
		return nil
	}
	for i := range r.BGP.Neighbors {
		if r.BGP.Neighbors[i].ID == id {
			return &r.BGP.Neighbors[i]
		}
	}
	return nil
}

func (e *edgeInfo) bgpFilter(id string) *BGPFilter {
	r := e.routing()
	if r.BGP == nil || id == "" {
		return nil
	}
	for i := range r.BGP.Filters {
		if r.BGP.Filters[i].ID == id {
			return &r.BGP.Filters[i]
		}
	}
	return nil
}

func neighborType(local, asn int) string {
	if local == asn { // rule: rt-bgp-ibgp-ebgp
		return BGPInternal
	}
	return BGPExternal
}

// ComputeRoutes computes every Edge's route tables for every segment, from the authored topology, the conditions
// and the assumptions. It never changes the topology. A nil clock means the system clock.
func ComputeRoutes(t *Topology, cond Conditions, as RoutingAssumptions, clock Clock) (*RouteSet, error) {
	s, err := newRtState(t, cond, as, clock)
	if err != nil {
		return nil, err
	}
	return s.routeSet(), nil
}

func newRtState(t *Topology, cond Conditions, as RoutingAssumptions, clock Clock) (*rtState, error) {
	if err := as.Validate(); err != nil {
		return nil, err
	}
	if clock == nil {
		clock = SystemClock()
	}
	c, err := newOverlayCtx(t, cond)
	if err != nil {
		return nil, err
	}
	if bad := t.validateRoutingConditions(cond); len(bad) > 0 {
		return nil, &ConditionError{Problems: bad}
	}
	ts, err := ComputeTunnels(t, cond, clock)
	if err != nil {
		return nil, err
	}
	s := &rtState{t: t, c: c, ts: ts, tix: indexTunnels(ts), cond: cond, as: as, used: map[string]bool{}, nmSeen: map[string]bool{},
		byEdge: map[string][]*originRoute{}, paths: map[string]pathVerdict{}, exits: map[string][]string{}, pins: map[string]map[string]int{},
		nbDown: setOf(cond.NeighborsDown), probeDown: setOf(cond.ProbesDown), tables: map[string][]*Route{},
		adverts: map[string][]LANAdvertisement{}, withheld: map[string][]Withheld{}}
	s.nms = []NotModelled{}
	s.resolveFlags()
	s.resolveOFC()
	for _, e := range c.edges {
		s.discover(e)
	}
	s.finishOrigins()
	s.buildTables()
	s.selectAll()
	s.redistribute()
	return s, nil
}

func (s *rtState) resolveFlags() {
	flags := &AdvertiseFlags{}
	if s.t.OverlayFlowControl != nil && s.t.OverlayFlowControl.AdvertiseFlags != nil {
		flags = s.t.OverlayFlowControl.AdvertiseFlags
	}
	s.flagStatic = s.as.AdvertiseStatic // rule: assume-69
	if flags.Static != nil {
		s.flagStatic = *flags.Static
	} else {
		s.use("advertise_static")
	}
	s.flagConnected = s.as.AdvertiseConnected
	if flags.Connected != nil {
		s.flagConnected = *flags.Connected
	} else {
		s.use("advertise_connected")
	}
	s.flagExtOSPF = false // rule: rt-global-advertise-flags
	if flags.ExternalOSPF != nil {
		s.flagExtOSPF = *flags.ExternalOSPF
	}
	s.nm(RNMRtOFCRest, "Internal, external and uplink Global Advertise Flags") // rule: nm-rt-ofc-rest
}

func (s *rtState) resolveOFC() {
	if s.t.OverlayFlowControl == nil {
		return
	}
	for _, pe := range s.t.OverlayFlowControl.PreferredExits {
		s.exits[segOr(pe.Segment)+"|"+pe.Prefix] = pe.Exits // rule: rt-ofc-preferred-exits
	}
	for i, p := range s.t.OverlayFlowControl.Pins {
		k := segOr(p.Segment) + "|" + p.Prefix
		if s.pins[k] == nil {
			s.pins[k] = map[string]int{}
		}
		s.pins[k][p.Exit] = i + 1 // rule: rt-ofc-pin
	}
}

func (s *rtState) add(o *originRoute) *originRoute {
	s.origins = append(s.origins, o)
	s.byEdge[o.edge.Edge.ID] = append(s.byEdge[o.edge.Edge.ID], o)
	return o
}

// discover finds every route the Edge knows locally: connected subnets, static routes, what its LAN routers offer
// and Non SD-WAN destination routes.
func (s *rtState) discover(e *edgeInfo) {
	// Connected routes: one per LAN or routed interface subnet, in each segment of the interface.
	for _, in := range e.Edge.Interfaces {
		if (in.Kind != IfLan && in.Kind != IfRouted) || in.Address == "" {
			continue
		}
		p, ok := pfxOf(in.Address)
		if !ok {
			continue
		}
		segs := in.Segments
		if len(segs) == 0 {
			segs = e.Profile.Segments
		}
		for _, seg := range segs {
			s.add(&originRoute{edge: e, segment: seg, pfx: p, kind: "connected", id: in.ID, via: in.ID, known: true, installed: true, // rule: rt-connected-from-interfaces
				knownWhy: "the interface " + in.ID + " has this subnet", knownBasis: basisOf(RRtConnectedSubnet), reachable: true}) // rule: rt-connected-from-interfaces
		}
	}
	// Static routes.
	r := e.routing()
	for i := range r.StaticRoutes {
		sr := &r.StaticRoutes[i]
		p, ok := pfxOf(sr.Subnet)
		if !ok {
			continue
		}
		o := &originRoute{edge: e, segment: segOr(sr.Segment), pfx: p, kind: "static", id: sr.ID, via: sr.ID, static: sr, known: true, installed: true, // rule: assume-28
			knownWhy: "configured under Static Route Settings", knownBasis: basisOf(RA28StaticSegment), reachable: true} // rule: assume-28
		cost := 0 // rule: assume-71
		if sr.Cost != nil {
			cost = *sr.Cost
		}
		o.cost = &cost
		if sr.ICMPProbe && s.probeDown[e.Edge.ID+"/"+sr.ID] { // rule: assume-72
			o.reachable, o.reachCode = false, CodeSLAStateDown
			o.reachReason = "the ICMP probe of this static route is down, so the next hop is treated as unreachable"
		}
		s.add(o)
	}
	// Routes the LAN routers offer.
	for i := range r.LANRoutes {
		lr := &r.LANRoutes[i]
		p, ok := pfxOf(lr.Prefix)
		if !ok {
			continue
		}
		o := &originRoute{edge: e, segment: segOr(lr.Segment), pfx: p, id: lr.ID, via: lr.Via, lan: lr, reachable: true}
		switch lr.Protocol {
		case ProtoOSPF:
			s.learnOSPF(o)
		case ProtoBGP:
			s.learnBGP(o)
		default:
			continue
		}
		s.add(o)
	}
	// Non SD-WAN destination routes.
	for i := range s.t.NSDs {
		n := &s.t.NSDs[i]
		if len(n.Prefixes) == 0 {
			continue
		}
		use := false
		if n.Via == NSDViaEdge {
			use = n.Edge == e.Edge.ID && hasString(e.CVPN.NSDEdge, n.ID) // rule: assume-76
		} else {
			use = e.CVPN.Enabled && hasString(e.CVPN.NSDGateway, n.ID)
		}
		if !use {
			continue
		}
		for pi := range n.Prefixes {
			np := &n.Prefixes[pi]
			p, ok := pfxOf(np.Prefix)
			if !ok {
				continue
			}
			cost := 0
			if np.Cost != nil {
				cost = *np.Cost
			}
			o := &originRoute{edge: e, segment: segOr(n.Segment), pfx: p, kind: "nsd", id: n.ID + ":" + np.Prefix, via: n.ID, nsd: n, nsdPfx: np, cost: &cost, known: true, installed: true,
				knownWhy: "static route of the Non SD-WAN destination " + n.ID, knownBasis: basisOf(RA76NSDSlots)} // rule: assume-76
			o.reachable, o.reachReason = s.nsdUsable(e, n)
			if !o.reachable {
				o.reachCode = CodePRUnreachable
			}
			s.add(o)
		}
	}
}

// nsdUsable says whether the Edge has a working path to the destination's tunnel.
func (s *rtState) nsdUsable(e *edgeInfo, n *NSD) (bool, string) {
	d := &PathDecision{Options: []PathOption{}}
	s.c.nsdOptions(d, e, n, s.tix, func(rule, item string) NotModelled { return s.nm(rule, item) })
	for _, o := range d.Options {
		if o.Usable {
			return true, ""
		}
	}
	if len(d.Options) > 0 {
		return false, d.Options[0].Reason
	}
	return false, "no tunnel to the destination"
}

func (s *rtState) learnOSPF(o *originRoute) {
	e, lr := o.edge, o.lan
	o.kind = "ospf"
	o.routeType = map[string]string{OSPFIntra: TypeO, OSPFInter: TypeOIA, OSPFE1: TypeOE1, OSPFE2: TypeOE2}[lr.OSPFType]
	if lr.Metric != nil {
		o.cost = lr.Metric
	}
	o.knownBasis = basisOf(RRtOSPFProfile) // rule: rt-ospf-profile-first
	switch {
	case e.ospfProfile() == nil:
		o.knownWhy = "OSPF is not activated in the Profile, so the Edge runs no OSPF"
	case o.segment != GlobalSegmentID: // rule: rt-ospf-global-segment
		o.knownWhy = "OSPF exists only on the Global Segment"
		o.knownBasis = basisOf(RRtOSPFGlobal) // rule: rt-ospf-global-segment
	case e.ospfIface(lr.Via) == nil:
		o.knownWhy = "OSPF is not enabled on interface " + lr.Via
	case e.ospfIface(lr.Via).Mode != OSPFActive: // rule: assume-73
		o.knownWhy = "interface " + lr.Via + " is passive, so no adjacency forms and no route is learned through it"
		o.knownBasis, o.knownAlso = basisOf(RA73PassiveNoAdj), bases(RRtOSPFPassive) // rule: rt-ospf-switched-passive // rule: assume-73
		s.nm(RNMRtPassive, "What a passive OSPF interface does")                     // rule: nm-rt-passive-ospf
	case s.nbDown[e.Edge.ID+"/"+lr.Via]: // rule: assume-30
		o.knownWhy = "the OSPF neighbor on " + lr.Via + " is not Full, so its routes are not learned"
		o.knownBasis = basisOf(RA30NeighborDown) // rule: assume-30
	default:
		o.known, o.installed = true, true
		o.knownWhy = "learned from the OSPF neighbor on " + lr.Via + " (state Full)"
		o.knownBasis = basisOf(RRtLocalRemote) // rule: rt-local-remote
	}
	if !o.known {
		s.withhold(o, "known")
	}
}

func (s *rtState) learnBGP(o *originRoute) {
	e, lr := o.edge, o.lan
	o.kind = "bgp"
	nb := e.bgpNeighbor(lr.Via)
	o.nbr = nb
	if lr.BGP != nil {
		o.attrs = attrs{localPref: lr.BGP.LocalPreference, med: lr.BGP.MED, asPath: append([]int(nil), lr.BGP.ASPath...)}
	}
	o.knownBasis = basisOf(RRtBGPType) // rule: rt-bgp-ibgp-ebgp
	bgp := e.routing().BGP             // rule: assume-78
	if nb == nil || bgp == nil {
		o.knownWhy = "BGP is not configured with neighbor " + lr.Via
		s.withhold(o, "known")
		return
	}
	o.routeType = TypeEB
	if neighborType(bgp.LocalASN, nb.ASN) == BGPInternal { // rule: rt-bgp-ibgp-ebgp
		o.routeType = TypeIB
	}
	if s.nbDown[e.Edge.ID+"/"+lr.Via] { // rule: assume-30
		o.knownWhy = "the BGP session with " + nb.IP + " is not Established, so its routes are not learned"
		o.knownBasis = basisOf(RA30NeighborDown) // rule: assume-30
		s.withhold(o, "known")
		return
	}
	o.known = true
	o.knownWhy = "received from BGP neighbor " + nb.IP + " (Established)"
	o.knownBasis = basisOf(RRtLocalRemote) // rule: rt-local-remote
	// Inbound filter.
	if f := e.bgpFilter(nb.InboundFilter); f != nil {
		res := s.evalFilter(f, o.pfx, o.attrs)
		o.attrs = res.attrs
		o.notes = append(o.notes, res.notes...)
		if !res.permit {
			o.held, o.heldStage = true, "filtered"
			o.heldWhy = fmt.Sprintf("the inbound filter %s on neighbor %s denies it (%s)", f.ID, nb.IP, res.why)
			o.heldBasis, o.heldAlso = res.basis, res.also
			s.withhold(o, "filtered")
			return
		}
	}
	o.installed = true
}

func (s *rtState) withhold(o *originRoute, stage string) {
	why, b, also := o.knownWhy, o.knownBasis, o.knownAlso
	if stage == "filtered" {
		why, b, also = o.heldWhy, o.heldBasis, o.heldAlso
	}
	src := "ospf"
	if o.kind == "bgp" {
		src = "bgp"
	}
	s.withheld[o.edge.Edge.ID] = append(s.withheld[o.edge.Edge.ID], Withheld{Edge: o.edge.Edge.ID, Segment: o.segment, Prefix: o.pfx.String(), Source: src, Via: o.via,
		Stage: stage, Reason: why, Basis: b, Also: also})
}

// finishOrigins decides, for every origin route, whether it enters the overlay and whether it is held back.
func (s *rtState) finishOrigins() {
	// Only one of several static routes for the same network is advertised: the best (lowest cost, LAN first).
	best := map[string]*originRoute{}
	for _, o := range s.origins {
		if o.kind != "static" {
			continue
		}
		k := o.edge.Edge.ID + "|" + o.segment + "|" + o.pfx.String()
		if cur, ok := best[k]; !ok || staticBetter(o, cur) {
			best[k] = o
		}
	}
	for _, o := range s.origins {
		if !o.known {
			continue
		}
		s.decideAdvertise(o, best)
		s.decideHeld(o)
		o.advertised = o.known && o.advOK && !o.held && o.installed
	}
}

func staticBetter(a, b *originRoute) bool {
	if *a.cost != *b.cost {
		return *a.cost < *b.cost
	}
	aw, bw := a.static.Interface != "", b.static.Interface != ""
	if aw != bw {
		return !aw
	}
	return a.id < b.id
}

func (s *rtState) decideAdvertise(o *originRoute, best map[string]*originRoute) {
	e := o.edge
	no := func(why string, b Basis, also ...Basis) {
		o.advOK, o.advWhy, o.advBasis, o.advAlso = false, why, b, also
	}
	yes := func(why string, b Basis, also ...Basis) {
		o.advOK, o.advWhy, o.advBasis, o.advAlso = true, why, b, also
	}
	if !e.cvpnOn(o.segment) { // rule: assume-82
		b := basisOf(RA82OriginCloudVPN) // rule: assume-82
		if o.kind == "static" {
			b = basisOf(RRtStaticCloudVPN) // rule: rt-static-needs-cloud-vpn
		}
		no("Cloud VPN is not enabled for segment "+o.segment+" on "+e.Edge.Name+", so nothing is advertised into the overlay", b, bases(RA82OriginCloudVPN)...) // rule: assume-82
		return
	}
	switch o.kind {
	case "connected":
		if !s.flagConnected {
			no("the Global Advertise Flag for connected routes is off", basisOf(RRtGlobalFlags)) // rule: rt-global-advertise-flags
			return
		}
		yes("connected routes are advertised (Global Advertise Flag on)", basisOf(RRtSendAll), bases(RA69AdvertiseDefaults)...) // rule: rt-edge-sends-all // rule: assume-69
	case "static":
		switch {
		case !o.static.Advertise:
			no("the Advertise box of this static route is not ticked", basisOf(RRtStaticAdvertise)) // rule: rt-static-advertise
		case !s.flagStatic:
			no("the Global Advertise Flag for static routes is off", basisOf(RRtGlobalFlags)) // rule: rt-global-advertise-flags
		case best[e.Edge.ID+"|"+o.segment+"|"+o.pfx.String()] != o:
			no("only one static route per network is advertised to the overlay, and another one (lower cost) is", basisOf(RRtOneStatic)) // rule: rt-one-static-advertised
		default:
			yes("the Advertise box is ticked and the flag is on", basisOf(RRtStaticAdvertise), bases(RRtOneStatic)...) // rule: rt-one-static-advertised // rule: rt-static-advertise
		}
	case "ospf":
		if o.routeType == TypeOE1 || o.routeType == TypeOE2 {
			if !s.flagExtOSPF {
				no("external OSPF routes (E1 and E2) are held out of the overlay until the External OSPF flag is enabled", basisOf(RRtExternalHeld), bases(RRtGlobalFlags)...) // rule: rt-external-ospf-held-out // rule: rt-global-advertise-flags
				return
			}
			yes("External OSPF flag is on", basisOf(RRtGlobalFlags)) // rule: rt-global-advertise-flags
			return
		}
		yes("routes learned through OSPF go to the Gateway automatically", basisOf(RRtOSPFAuto)) // rule: rt-ospf-to-overlay-auto
	case "bgp":
		yes("the route is received from an Established neighbor: advertise True, reachable Yes", basisOf(RRtBGPAdvertise)) // rule: rt-bgp-advertise-reachable
	case "nsd":
		if o.nsd.Via == NSDViaEdge {
			if o.nsdPfx.Advertise {
				yes("the Advertise box of this Non SD-WAN static route is ticked", basisOf(RRtStaticAdvertise)) // rule: rt-static-advertise
				return
			}
			no("the Advertise box of this Non SD-WAN static route is not ticked", basisOf(RRtStaticAdvertise)) // rule: rt-static-advertise
			return
		}
		no("Advertise cannot be edited for a destination via Gateway; its routes are used by the Edges that enable it, not advertised", basisOf(RA76NSDSlots)) // rule: assume-76
	}
}

// decideHeld applies the overlap rule: a received prefix that overlaps a locally connected subnet is not advertised.
func (s *rtState) decideHeld(o *originRoute) {
	if o.held || (o.kind != "ospf" && o.kind != "bgp") {
		return
	}
	for _, c := range s.byEdge[o.edge.Edge.ID] { // rule: rt-overlap-not-in-ofc
		if c.kind == "connected" && c.segment == o.segment && overlap(c.pfx, o.pfx) {
			o.held, o.heldStage = true, "filtered"
			o.heldWhy = fmt.Sprintf("it overlaps the connected subnet %s, so it is not advertised into the Overlay Flow Control table", c.pfx)
			o.heldBasis, o.heldAlso = basisOf(RRtOverlapHeld), bases(RA79Overlap) // rule: rt-overlap-not-in-ofc // rule: assume-79
			return
		}
	}
}

// pathTo says whether src has a usable overlay path to the Edge that owns a route.
func (s *rtState) pathTo(src, dst *edgeInfo) pathVerdict {
	k := src.Edge.ID + ">" + dst.Edge.ID
	if v, ok := s.paths[k]; ok {
		return v
	}
	d := &PathDecision{Options: []PathOption{}}
	s.c.edgeOptions(d, src, dst, s.tix, func(rule, item string) NotModelled { return s.nm(rule, item) })
	v := pathVerdict{reason: "no overlay option"}
	for i := range d.Options {
		if d.Options[i].Usable {
			o := d.Options[i]
			v = pathVerdict{ok: true, label: o.Label, option: &o}
			break
		}
	}
	if !v.ok && len(d.Options) > 0 {
		var parts []string
		for _, o := range d.Options {
			parts = append(parts, o.Label+": "+o.Reason)
		}
		v.reason = strings.Join(parts, "; ")
	}
	if s.cond.OrchestratorUnreachable { // rule: nm-orchestrator-down
		s.nm(RNMOrchestratorDown, "Edge behaviour with the Orchestrator unreachable")
	}
	s.paths[k] = v
	return v
}

func (s *rtState) prefValue(typ string) (*int, string) {
	switch typ {
	case TypeO:
		return intp(1001), "" // rule: rt-default-preferences
	case TypeOIA:
		return intp(1002), ""
	case TypeEB:
		return intp(1003), ""
	case TypeIB:
		return intp(1004), ""
	case TypeOE1, TypeOE2:
		s.nm(RNMRtOEPref, "Preference of OSPF E1 and E2 routes") // rule: nm-rt-oe-preference
		return nil, "not modelled: the printed value is VERIFY; ordered after BGP"
	}
	return nil, ""
}

// buildTables creates every Edge's candidate routes: its own installed routes and the remote routes the overlay
// delivers.
func (s *rtState) buildTables() {
	for _, e := range s.c.edges {
		for _, seg := range e.Profile.Segments {
			var cands []*Route
			for _, o := range s.byEdge[e.Edge.ID] {
				if o.segment == seg && o.installed {
					cands = append(cands, s.localRoute(o))
				}
			}
			if s.cond.OrchestratorUnreachable { // rule: nm-orchestrator-down
				s.nm(RNMOrchestratorDown, "Edge behaviour with the Orchestrator unreachable")
			} else if e.cvpnOn(seg) { // rule: assume-32
				for _, o := range s.origins {
					if o.edge == e || o.segment != seg || !o.advertised {
						continue
					}
					cands = append(cands, s.remoteRoute(e, o))
				}
			}
			s.tables[e.Edge.ID+"|"+seg] = cands
		}
	}
}

func (s *rtState) newRoute(e *edgeInfo, o *originRoute) *Route {
	r := &Route{Edge: e.Edge.ID, Segment: o.segment, Prefix: o.pfx.String(), pfx: o.pfx, owner: o, Via: o.via, Reachable: true, RouteType: o.routeType, Cost: o.cost,
		LocalPreference: o.attrs.localPref, Metric: o.attrs.med, ASPath: o.attrs.asPath, exitID: o.edge.Edge.ID, cmpCost: o.cost, Notes: append([]string(nil), o.notes...)}
	return r
}

func (s *rtState) localRoute(o *originRoute) *Route {
	r := s.newRoute(o.edge, o)
	r.Reachable, r.ReachableCode, r.ReachableReason = o.reachable, o.reachCode, o.reachReason
	r.Basis = basisOf(RRtLocalRemote) // rule: rt-local-remote
	switch o.kind {
	case "connected":
		r.Source, r.NextHop, r.Basis = SrcConnected, "connected on "+o.via, basisOf(RRtConnectedSubnet) // rule: rt-connected-from-interfaces
	case "static":
		r.Source = SrcLocalStatic
		r.Basis = basisOf(RRtOrderOfOps) // rule: rt-order-of-operations
		if o.static.Interface != "" {
			r.NextHop = fmt.Sprintf("%s via WAN interface %s", orDash(o.static.NextHop), o.static.Interface)
		} else {
			r.NextHop = fmt.Sprintf("%s (LAN)", orDash(o.static.NextHop))
		}
		if o.static.ICMPProbe {
			r.Also = append(r.Also, basisOf(RRtProbe)) // rule: rt-static-icmp-probe
		}
		if !o.reachable {
			r.Also = append(r.Also, basisOf(RA72ProbeCode)) // rule: assume-72
		}
	case "ospf":
		r.Source = SrcOSPF
		r.NextHop = "OSPF neighbor on " + o.via
		r.Basis = basisOf(RRtOSPFTypeOrder) // rule: rt-ospf-type-order
		r.Preference, r.PreferenceNote = s.prefValue(o.routeType)
	case "bgp":
		r.Source = SrcBGP
		r.NextHop = "BGP neighbor " + o.nbr.IP
		r.Basis = basisOf(RRtBGPAttrs) // rule: rt-bgp-attributes
		r.Preference, r.PreferenceNote = s.prefValue(o.routeType)
	case "nsd":
		r.Source = SrcNSDLocal
		r.NextHop = "Non SD-WAN destination " + o.via
		r.Basis, r.Also = basisOf(RA76NSDSlots), bases(RRtOrderOfOps) // rule: rt-order-of-operations // rule: assume-76
	}
	if o.kind == "ospf" || o.kind == "bgp" {
		r.Also = append(r.Also, bases(RRtDefaultPrefs)...) // rule: rt-default-preferences
	}
	r.ID = routeID(r)
	return r
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (s *rtState) remoteRoute(e *edgeInfo, o *originRoute) *Route {
	r := s.newRoute(e, o)
	r.Origin, r.OriginName = o.edge.Edge.ID, o.edge.Edge.Name
	r.Via = ""
	r.NextHop = "overlay route (next-hop type not modelled)"                                           // rule: assume-81
	r.Basis, r.Also = basisOf(RRtReflect), bases(RRtLocalRemote, RA81RemoteAttrs, RA32CloudVPNReflect) // rule: rt-local-remote // rule: rt-gateway-reflects // rule: assume-81 // rule: assume-32
	switch o.kind {
	case "connected":
		r.Source = SrcRemoteConnected
	case "static":
		r.Source = SrcRemoteStatic
	case "nsd":
		r.Source = SrcNSDRemote
	default:
		r.Source = SrcRemoteDynamic
		r.Preference, r.PreferenceNote = s.prefValue(o.routeType)
	}
	pv := s.pathTo(e, o.edge)
	r.Reachable = pv.ok
	if !pv.ok {
		r.ReachableCode = CodePRUnreachable
		r.ReachableReason = "the remote peer " + o.edge.Edge.Name + " is unreachable: " + pv.reason
		r.Also = append(r.Also, basisOf(RRtUnreachable)) // rule: rt-unreachable-code
	} else if o.kind == "static" && !o.reachable {
		r.Reachable, r.ReachableCode, r.ReachableReason = false, o.reachCode, "the owning Edge's probe of this static route is down"
	} else if o.kind == "nsd" && !o.reachable {
		r.Reachable, r.ReachableCode, r.ReachableReason = false, CodePRUnreachable, "the Non SD-WAN destination is unreachable: "+o.reachReason
	}
	r.Cost = nil                                          // the transit metric is not modelled, so the table shows no cost for a remote route
	s.nm(RNMRtTransit, "Transit metric of remote routes") // rule: nm-rt-transit-metric
	if o.kind == "bgp" || o.kind == "ospf" {
		s.nm(RNMRtASPath, "How the overlay changes route attributes") // rule: nm-rt-overlay-attributes
	}
	r.ID = routeID(r)
	return r
}

func routeID(r *Route) string {
	return strings.Join([]string{r.Edge, r.Segment, r.Prefix, r.Source, r.RouteType, r.exitID, r.Via}, "|")
}

// routeSet assembles the public result.
func (s *rtState) routeSet() *RouteSet {
	rs := &RouteSet{Topology: s.t.Meta.ID, Conditions: s.cond, Assumptions: s.as.Describe(s.used), Edges: []EdgeRoutes{}}
	for _, e := range s.c.edges {
		er := EdgeRoutes{Edge: e.Edge.ID, Name: e.Edge.Name, Segments: []SegmentTable{}, LANAdvertisements: s.adverts[e.Edge.ID], Withheld: s.withheld[e.Edge.ID]}
		if er.LANAdvertisements == nil {
			er.LANAdvertisements = []LANAdvertisement{}
		}
		if er.Withheld == nil {
			er.Withheld = []Withheld{}
		}
		for _, seg := range e.Profile.Segments {
			st := SegmentTable{Segment: seg, Routes: []Route{}}
			for _, r := range s.tables[e.Edge.ID+"|"+seg] {
				st.Routes = append(st.Routes, *r)
			}
			sort.SliceStable(st.Routes, func(i, j int) bool {
				a, b := st.Routes[i], st.Routes[j]
				if a.pfx.Addr() != b.pfx.Addr() {
					return a.pfx.Addr().Less(b.pfx.Addr())
				}
				if a.pfx.Bits() != b.pfx.Bits() {
					return a.pfx.Bits() < b.pfx.Bits()
				}
				return a.Rank < b.Rank
			})
			er.Segments = append(er.Segments, st)
		}
		rs.Edges = append(rs.Edges, er)
	}
	s.nm(RNMRtCrossSegment, "Routes crossing segments") // rule: nm-rt-cross-segment
	s.nm(RNMRtLoops, "Redistribution loops")            // rule: nm-rt-redistribution-loops
	s.nm(RNMRtReflector, "Gateway as route reflector")  // rule: nm-rt-gateway-reflector
	rs.NotModelled = s.nms
	rs.Assumptions = s.as.Describe(s.used)
	return rs
}

// Lookup finds the route an Edge would use for a destination address: the longest prefix that has a reachable
// candidate, then the selected route of that prefix. Cross-segment lookups are not answered.
func (rs *RouteSet) Lookup(edge, segment string, dst netip.Prefix) (*Route, []Route) {
	segment = segOr(segment) // rule: rt-segment-scoped
	var best *netip.Prefix
	for _, er := range rs.Edges {
		if er.Edge != edge {
			continue
		}
		for _, st := range er.Segments {
			if st.Segment != segment {
				continue
			}
			for i := range st.Routes {
				r := st.Routes[i]
				if !pfxContains(r.pfx, dst) {
					continue
				}
				if r.Selected && (best == nil || r.pfx.Bits() > best.Bits()) { // rule: rt-longest-prefix
					p := r.pfx
					best = &p
				}
			}
			if best == nil {
				// a prefix whose candidates are all unreachable still answers (as unreachable), longest first
				for i := range st.Routes {
					r := st.Routes[i]
					if pfxContains(r.pfx, dst) && (best == nil || r.pfx.Bits() > best.Bits()) {
						p := r.pfx
						best = &p
					}
				}
			}
			if best == nil {
				return nil, nil
			}
			var cands []Route
			var sel *Route
			for i := range st.Routes {
				if st.Routes[i].pfx == *best {
					cands = append(cands, st.Routes[i])
					if st.Routes[i].Selected {
						c := st.Routes[i]
						sel = &c
					}
				}
			}
			return sel, cands
		}
	}
	return nil, nil
}

func itoaS(i int) string { return strconv.Itoa(i) }
