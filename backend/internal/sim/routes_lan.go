package sim

import (
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

// LANAdvertisement is one thing an Edge redistributes (or would redistribute) to its LAN routers, with the
// verdict and the evidence. It is how the model answers "is the route redistributed where it needs to be".
type LANAdvertisement struct {
	Edge       string   `json:"edge"`
	Protocol   string   `json:"protocol"` // ospf or bgp
	Neighbor   string   `json:"neighbor,omitempty"`
	Segment    string   `json:"segment"`
	Prefix     string   `json:"prefix"`
	FromRoute  string   `json:"from_route,omitempty"`
	Kind       string   `json:"kind"` // overlay prefix, BGP route, OSPF route, connected, default route, summary
	Advertised bool     `json:"advertised"`
	Reason     string   `json:"reason"`
	OSPFType   string   `json:"ospf_type,omitempty"`
	Metric     *int     `json:"metric,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
	SummaryOf  string   `json:"summarized_into,omitempty"`
	// Switch is the assumption key that supplied the deciding value ("" when the topology set it).
	Switch string  `json:"switch,omitempty"`
	Basis  Basis   `json:"basis"`
	Also   []Basis `json:"also,omitempty"`

	pfx netip.Prefix
}

type filterResult struct {
	permit bool
	attrs  attrs
	notes  []string
	why    string
	basis  Basis
	also   []Basis
}

func ruleMatches(r BGPFilterRule, p netip.Prefix, at attrs) bool {
	m := r.Match
	if m.Prefix != "" {
		rp, ok := pfxOf(m.Prefix)
		if !ok {
			return false
		}
		exact := m.ExactMatch == nil || *m.ExactMatch // rule: rt-filter-match
		if exact && rp != p {
			return false
		}
		if !exact && !pfxContains(rp, p) {
			return false
		}
	}
	if m.Metric != nil && (at.med == nil || *at.med != *m.Metric) {
		return false
	}
	if m.LocalPreference != nil {
		v := 100 // rule: assume-75
		if at.localPref != nil {
			v = *at.localPref
		}
		if v != *m.LocalPreference {
			return false
		}
	}
	return true
}

// evalFilter evaluates a filter against one route. The default action and the evaluation order are selectable
// assumptions (23): first_match walks the rules top to bottom, last_match bottom to top.
func (s *rtState) evalFilter(f *BGPFilter, p netip.Prefix, at attrs) filterResult {
	order := make([]int, len(f.Rules))
	for i := range order {
		order[i] = i
	}
	if s.as.FilterOrder == FilterOrderLast { // rule: assume-23
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	hit := -1
	matches := 0
	for _, i := range order {
		if ruleMatches(f.Rules[i], p, at) {
			matches++
			if hit < 0 {
				hit = i
			}
		}
	}
	if matches > 1 {
		s.use("filter_order")
	}
	if hit < 0 {
		s.use("filter_default_action")
		res := filterResult{permit: s.as.FilterDefaultAction == FilterPermit, attrs: at, basis: basisOf(RA23FilterDefault)} // rule: assume-23
		res.why = fmt.Sprintf("no rule of %s matched, so the default action %s applies", f.ID, s.as.FilterDefaultAction)
		return res
	}
	r := f.Rules[hit]
	label := r.ID
	if label == "" {
		label = "rule " + strconv.Itoa(hit+1)
	}
	res := filterResult{permit: r.Action == FilterPermit, attrs: at, basis: basisOf(RRtFilterMatch), also: bases(RRtFilterActions)} // rule: rt-filter-set-actions // rule: rt-filter-match
	res.why = fmt.Sprintf("%s of %s matched (%s)", label, f.ID, r.Action)
	if !res.permit {
		return res
	}
	res.attrs.asPath = append([]int(nil), at.asPath...)
	if v := r.Set.LocalPreference; v != nil { // rule: rt-filter-set-actions
		res.attrs.localPref = v
		res.notes = append(res.notes, fmt.Sprintf("the filter %s set local preference to %d", f.ID, *v))
	}
	if v := r.Set.Metric; v != nil {
		res.attrs.med = v
		res.notes = append(res.notes, fmt.Sprintf("the filter %s set the metric (MED) to %d", f.ID, *v))
	}
	if len(r.Set.ASPathPrepend) > 0 {
		res.attrs.asPath = append(append([]int(nil), r.Set.ASPathPrepend...), res.attrs.asPath...)
		res.notes = append(res.notes, fmt.Sprintf("the filter %s prepended %s to the AS path", f.ID, joinInts(r.Set.ASPathPrepend)))
	}
	return res
}

func joinInts(v []int) string {
	var p []string
	for _, x := range v {
		p = append(p, strconv.Itoa(x))
	}
	return strings.Join(p, " ")
}

func (s *rtState) pickBool(set *bool, def bool, key string) bool {
	if set != nil {
		return *set
	}
	s.use(key)
	return def
}

func (s *rtState) selectedRoutes(e *edgeInfo, seg string) []*Route {
	var out []*Route
	for _, r := range s.tables[e.Edge.ID+"|"+seg] {
		if r.Selected {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].pfx.Addr() != out[j].pfx.Addr() {
			return out[i].pfx.Addr().Less(out[j].pfx.Addr())
		}
		return out[i].pfx.Bits() < out[j].pfx.Bits()
	})
	return out
}

func isOverlayRoute(r *Route) bool {
	switch r.Source {
	case SrcRemoteConnected, SrcRemoteStatic, SrcRemoteDynamic, SrcNSDRemote:
		return true
	}
	return false
}

func hasDefault(rs []*Route) bool {
	for _, r := range rs {
		if r.pfx.Bits() == 0 {
			return true
		}
	}
	return false
}

// redistribute computes what every Edge sends to its LAN routers: OSPF external routes and BGP advertisements.
func (s *rtState) redistribute() {
	for _, e := range s.c.edges {
		s.redistributeOSPF(e)
		s.redistributeBGP(e)
	}
}

func (s *rtState) redistributeOSPF(e *edgeInfo) {
	prof := e.ospfProfile()
	r := e.routing()
	if prof == nil || r.OSPF == nil || len(r.OSPF.Interfaces) == 0 {
		return
	}
	rd := prof.Redistribute
	if rd == nil {
		rd = &OSPFRedistribute{}
	}
	adjacent := false
	for _, in := range r.OSPF.Interfaces {
		if in.Mode == OSPFActive && !s.nbDown[e.Edge.ID+"/"+in.ID] {
			adjacent = true
		}
	}
	overlay := s.pickBool(rd.OverlayPrefixes, s.as.OSPFOverlayPrefixes, "ospf_overlay_prefixes")
	fromBGP := s.pickBool(rd.BGP, s.as.OSPFRedistributeBGP, "ospf_redistribute_bgp")
	metric := 20 // rule: rt-ospf-external-defaults
	if rd.Metric != nil {
		metric = *rd.Metric
	}
	mtype := MetricE2
	if rd.MetricType != "" {
		mtype = rd.MetricType
	}
	s.nm(RNMRtOSPFConnStat, "Connected and static routes into OSPF") // rule: nm-rt-connected-static-into-ospf
	s.nm(RNMRtOSPFFilters, "OSPF route filters")                     // rule: nm-rt-ospf-filters
	var list []LANAdvertisement
	add := func(a LANAdvertisement) {
		a.Edge, a.Protocol, a.Segment = e.Edge.ID, ProtoOSPF, GlobalSegmentID
		list = append(list, a)
	}
	external := func(r *Route, kind string, on bool, key string, rule string) LANAdvertisement {
		a := LANAdvertisement{Prefix: r.Prefix, pfx: r.pfx, FromRoute: r.ID, Kind: kind, OSPFType: mtype, Metric: intp(metric), Basis: basisOf(rule), Also: bases(RRtBestOnly, RRtOSPFExtDefaults)} // rule: rt-redistribute-best-only // rule: rt-ospf-external-defaults
		switch {
		case !on:
			a.Reason = "the " + map[string]string{"overlay prefix": "Overlay Prefixes", "BGP route": "BGP"}[kind] + " option of the OSPF settings is off, so this route is not redistributed into OSPF"
			a.Switch = key
			if rd.OverlayPrefixes != nil && kind == "overlay prefix" || rd.BGP != nil && kind == "BGP route" {
				a.Switch = ""
			}
		case !adjacent:
			a.Reason = "no OSPF neighbor is Full on an active interface, so there is nothing to send it to"
			a.Basis = basisOf(RRtOSPFRouted) // rule: rt-ospf-full-needs-routed
		default:
			a.Advertised = true
			a.Reason = fmt.Sprintf("redistributed as an external %s route with metric %d", strings.ToUpper(mtype), metric)
		}
		return a
	}
	for _, r := range s.selectedRoutes(e, GlobalSegmentID) {
		switch {
		case isOverlayRoute(r):
			a := external(r, "overlay prefix", overlay, "ospf_overlay_prefixes", RRtOverlaySwitches) // rule: rt-overlay-redistribution-switches
			if overlay && a.Switch == "" && rd.OverlayPrefixes == nil {
				a.Switch = "ospf_overlay_prefixes"
			}
			add(a)
		case r.Source == SrcBGP:
			a := external(r, "BGP route", fromBGP, "ospf_redistribute_bgp", RRtBGPFromOSPF) // rule: rt-ospf-into-bgp
			if fromBGP && rd.BGP == nil {
				a.Switch = "ospf_redistribute_bgp"
			}
			add(a)
		}
	}
	// Default route into OSPF: off (None) unless a type is chosen; Conditional needs a default route in the table.
	if rd.DefaultRoute == MetricE1 || rd.DefaultRoute == MetricE2 { // rule: rt-ospf-default-route
		a := LANAdvertisement{Prefix: "0.0.0.0/0", Kind: "default route", OSPFType: rd.DefaultRoute, Metric: intp(metric), Basis: basisOf(RRtOSPFDefault), pfx: netip.MustParsePrefix("0.0.0.0/0")} // rule: rt-ospf-default-route
		switch {
		case !adjacent:
			a.Reason = "no OSPF neighbor is Full on an active interface"
		case rd.DefaultConditional && !hasDefault(s.selectedRoutes(e, GlobalSegmentID)):
			a.Reason = "Conditional: the Edge has not learned a default route through the overlay or underlay"
		default:
			a.Advertised = true
			a.Reason = "default route advertised into OSPF as " + strings.ToUpper(rd.DefaultRoute)
			if !rd.DefaultConditional { // rule: assume-77
				a.Also = bases(RA77OSPFDefaultAlways) // rule: assume-77
			}
		}
		add(a)
	}
	// Summarization table (external routes only).
	var summaries []LANAdvertisement
	for _, sm := range prof.Summaries {
		sp, ok := pfxOf(sm.Subnet)
		if !ok {
			continue
		}
		members := 0
		for i := range list {
			a := &list[i]
			if a.Kind == "default route" || !pfxContains(sp, a.pfx) || a.SummaryOf != "" {
				continue
			}
			if !a.Advertised {
				continue
			}
			members++
			a.Advertised = false
			a.SummaryOf = sp.String()
			if sm.NoAdvertise { // rule: rt-ospf-summary-external
				a.Reason = "the summary " + sp.String() + " has No Advertise, so the whole supernet is blocked"
				a.Basis = basisOf(RRtOSPFSummary) // rule: rt-ospf-summary-external
			} else {
				a.Reason = "summarised into " + sp.String() + "; only the summary is advertised"
				a.Basis, a.Also = basisOf(RRtOSPFSummary), bases(RA29SummaryOnly) // rule: rt-ospf-summary-external // rule: assume-29
			}
		}
		if members > 0 && !sm.NoAdvertise {
			t, m := mtype, metric // rule: assume-80
			if sm.MetricType != "" {
				t = sm.MetricType
			}
			if sm.Metric != nil {
				m = *sm.Metric
			}
			summaries = append(summaries, LANAdvertisement{Prefix: sp.String(), pfx: sp, Kind: "summary", Advertised: true, OSPFType: t, Metric: intp(m),
				Reason: fmt.Sprintf("summary of %d redistributed external routes as external %s with metric %d", members, strings.ToUpper(t), m),
				Basis:  basisOf(RRtOSPFSummary), Also: bases(RA80SummaryMetric, RA29SummaryOnly)}) // rule: rt-ospf-summary-external // rule: assume-80 // rule: assume-29
		}
	}
	for _, a := range summaries {
		add(a)
	}
	s.adverts[e.Edge.ID] = append(s.adverts[e.Edge.ID], list...)
}

func (s *rtState) redistributeBGP(e *edgeInfo) {
	bgp := e.routing().BGP
	if bgp == nil || len(bgp.Neighbors) == 0 {
		return
	}
	rd := bgp.Redistribute
	if rd == nil {
		rd = &BGPRedistribute{}
	}
	overlay := s.pickBool(rd.OverlayPrefix, s.as.BGPOverlayPrefix, "bgp_overlay_prefix")
	connected := s.pickBool(rd.Connected, s.as.BGPConnected, "bgp_connected")
	fromOSPF := s.pickBool(rd.OSPF, s.as.BGPRedistributeOSPF, "bgp_redistribute_ospf")
	ospfMetric := 20 // rule: rt-ospf-into-bgp
	if rd.OSPFMetric != nil {
		ospfMetric = *rd.OSPFMetric
	}
	sw := func(set bool, key string) string {
		if set {
			return ""
		}
		return key
	}
	_ = sw
	for _, nb := range bgp.Neighbors {
		seg := segOr(nb.Segment)
		if !e.inProfile(seg) {
			continue
		}
		flt := e.bgpFilter(nb.OutboundFilter)
		established := !s.nbDown[e.Edge.ID+"/"+nb.ID]
		table := s.selectedRoutes(e, seg)
		push := func(a LANAdvertisement) {
			a.Edge, a.Protocol, a.Neighbor, a.Segment = e.Edge.ID, ProtoBGP, nb.ID, seg
			if a.Advertised && !established {
				a.Advertised, a.Reason, a.Basis = false, "the BGP session with "+nb.IP+" is not Established", basisOf(RA30NeighborDown) // rule: assume-30
			}
			if a.Advertised && flt != nil {
				at := attrs{}
				if a.Metric != nil {
					at.med = a.Metric
				}
				res := s.evalFilter(flt, a.pfx, at)
				if !res.permit {
					a.Advertised = false
					a.Reason = fmt.Sprintf("the outbound filter %s on neighbor %s denies it (%s)", flt.ID, nb.IP, res.why)
					a.Basis, a.Also = res.basis, res.also
				} else {
					a.Attributes = append(a.Attributes, res.notes...)
					if res.attrs.med != nil {
						a.Metric = res.attrs.med
					}
				}
			}
			s.adverts[e.Edge.ID] = append(s.adverts[e.Edge.ID], a)
		}
		for _, r := range table {
			switch {
			case isOverlayRoute(r):
				a := LANAdvertisement{Prefix: r.Prefix, pfx: r.pfx, FromRoute: r.ID, Kind: "overlay prefix", Basis: basisOf(RRtOverlaySwitches), Also: bases(RRtBestOnly)} // rule: rt-redistribute-best-only // rule: rt-overlay-redistribution-switches
				switch {
				case !overlay:
					a.Reason = "the Overlay Prefix option of the BGP settings is off, so overlay routes are not redistributed into BGP"
					if rd.OverlayPrefix == nil {
						a.Switch = "bgp_overlay_prefix"
					}
				case s.overlapsConnected(e, seg, r.pfx): // rule: rt-overlap-not-in-bgp
					a.Reason = "it overlaps a connected subnet, so it is not redistributed into BGP even with Overlay Prefix on"
					a.Basis, a.Also = basisOf(RRtOverlapBGP), bases(RA79Overlap) // rule: rt-overlap-not-in-bgp // rule: assume-79
				default:
					a.Advertised = true
					a.Reason = "redistributed into BGP (Overlay Prefix on)"
					if rd.OverlayPrefix == nil {
						a.Switch = "bgp_overlay_prefix"
					}
				}
				push(a)
			case r.Source == SrcConnected:
				a := LANAdvertisement{Prefix: r.Prefix, pfx: r.pfx, FromRoute: r.ID, Kind: "connected", Basis: basisOf(RRtBGPSwitches), Also: bases(RRtConnectedSubnet)} // rule: rt-bgp-redistribute-switches // rule: rt-connected-from-interfaces
				if connected {
					a.Advertised, a.Reason = true, "redistributed into BGP (Connected Routes on)"
				} else {
					a.Reason = "the Connected Routes option of the BGP settings is off"
				}
				if rd.Connected == nil {
					a.Switch = "bgp_connected"
				}
				push(a)
			case r.Source == SrcOSPF:
				a := LANAdvertisement{Prefix: r.Prefix, pfx: r.pfx, FromRoute: r.ID, Kind: "OSPF route", Metric: intp(ospfMetric), Basis: basisOf(RRtBGPFromOSPF), Also: bases(RRtBestOnly)} // rule: rt-redistribute-best-only // rule: rt-ospf-into-bgp
				if fromOSPF {
					a.Advertised, a.Reason = true, fmt.Sprintf("redistributed into BGP (OSPF option on, metric %d)", ospfMetric)
				} else {
					a.Reason = "the OSPF option of the BGP settings is off"
				}
				if rd.OSPF == nil {
					a.Switch = "bgp_redistribute_ospf"
				}
				push(a)
			}
		}
		if dr := rd.DefaultRoute; dr == DefaultYes || dr == DefaultConditional { // rule: rt-bgp-default-route
			a := LANAdvertisement{Prefix: "0.0.0.0/0", pfx: netip.MustParsePrefix("0.0.0.0/0"), Kind: "default route", Basis: basisOf(RRtBGPDefault)} // rule: rt-bgp-default-route
			if dr == DefaultConditional && !hasDefault(table) {
				a.Reason = "Conditional: the Edge has not learned a default route through the overlay or underlay"
			} else {
				a.Advertised, a.Reason = true, "default route advertised ("+dr+")"
			}
			push(a)
		}
	}
}

func (s *rtState) overlapsConnected(e *edgeInfo, seg string, p netip.Prefix) bool {
	for _, o := range s.byEdge[e.Edge.ID] {
		if o.kind == "connected" && o.segment == seg && overlap(o.pfx, p) {
			return true
		}
	}
	return false
}

// advertsCovering lists what an Edge sends to its LAN that covers the prefix (the prefix itself, a summary or a
// default route), and all entries about the selected route (so a refusal can be explained).
func (s *rtState) advertsCovering(edge string, route *Route, p netip.Prefix) (ok []LANAdvertisement, all []LANAdvertisement) {
	for _, a := range s.adverts[edge] {
		covers := pfxContains(a.pfx, p)
		about := route != nil && a.FromRoute == route.ID
		if covers && a.Advertised {
			ok = append(ok, a)
		}
		if about || (covers && a.Kind != "overlay prefix") {
			all = append(all, a)
		}
	}
	return
}
