package sim

import (
	"fmt"
	"net/netip"
)

const maxASN = 4294967295 // 4-byte ASNs are supported

// routing validates the optional routing keys (step 13): Profile OSPF settings, Edge static routes, LAN routes,
// OSPF interfaces, BGP, Overlay Flow Control and Non SD-WAN destination prefixes. Every problem is reported.
func (v *validator) routing() {
	t := v.t
	edges := map[string]*Edge{}
	var order []string
	for si := range t.Sites {
		for ei := range t.Sites[si].Edges {
			e := &t.Sites[si].Edges[ei]
			if _, dup := edges[e.ID]; !dup {
				edges[e.ID] = e
				order = append(order, e.ID)
			}
		}
	}
	for i := range t.Profiles {
		if r := t.Profiles[i].Routing; r != nil && r.OSPF != nil {
			v.profileOSPF(fmt.Sprintf("profiles[%s].routing.ospf", t.Profiles[i].ID), r.OSPF)
		}
	}
	nsds := map[string]bool{}
	for _, n := range t.NSDs {
		nsds[n.ID] = true
	}
	for _, id := range order {
		if e := edges[id]; e.Routing != nil {
			v.edgeRouting(fmt.Sprintf("edges[%s].routing", e.ID), e, nsds)
		}
	}
	v.interfaceAddresses(edges, order)
	v.ofc(edges)
	v.nsdPrefixes()
}

// interfaceAddresses checks the optional interface addresses that give an Edge its connected routes.
func (v *validator) interfaceAddresses(edges map[string]*Edge, order []string) {
	for _, id := range order {
		e := edges[id]
		var seen []netip.Prefix
		for i := range e.Interfaces {
			in := &e.Interfaces[i]
			if in.Address == "" {
				continue
			}
			p := fmt.Sprintf("edges[%s].interfaces[%s].address", e.ID, in.ID)
			pf, err := netip.ParsePrefix(in.Address)
			if err != nil || !pf.Addr().Is4() {
				v.errf(p, "%q is not an IPv4 address with a prefix length like 10.20.0.1/24", in.Address)
				continue
			}
			if in.Kind == IfWan || in.Kind == IfLoopback {
				v.warnf(p, "only LAN and routed interface subnets become connected routes; this address is ignored")
				continue
			}
			for _, o := range seen {
				if overlap(o, pf.Masked()) { // rule: assume-79
					v.warnf(p, "%s overlaps the subnet %s of another interface of this Edge", pf.Masked(), o)
				}
			}
			seen = append(seen, pf.Masked())
		}
	}
}

// prefixOf parses a canonical IPv4 prefix and reports problems at path.
func (v *validator) prefixOf(path, s string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(s)
	switch {
	case err != nil:
		v.errf(path, "%q is not a prefix like 10.20.0.0/24", s)
		return p, false
	case !p.Addr().Is4():
		v.errf(path, "%q is not IPv4: IPv6 routing is not modelled", s)
		return p, false
	case p.Masked() != p:
		v.errf(path, "%q has host bits set; write the network address, %s", s, p.Masked())
		return p, false
	}
	return p, true
}

func (v *validator) rangeInt(path string, p *int, lo, hi int) {
	if p != nil && (*p < lo || *p > hi) {
		v.errf(path, "%d is outside %d to %d", *p, lo, hi)
	}
}

func (v *validator) profileOSPF(p string, o *OSPFProfile) {
	if !o.Enabled {
		if len(o.Areas) > 0 || o.Redistribute != nil || len(o.Summaries) > 0 {
			v.warnf(p, "OSPF is not enabled in this Profile, so its areas, redistribution and summaries have no effect")
		}
		return
	}
	if len(o.Areas) == 0 {
		v.errf(p+".areas", "OSPF is enabled but has no area: an Area ID is entered per area")
	}
	seen := map[string]bool{}
	for i, a := range o.Areas {
		ap := fmt.Sprintf("%s.areas[%d]", p, i)
		switch {
		case a.ID == "":
			v.errf(ap+".id", "must not be empty")
		case seen[a.ID]:
			v.errf(ap+".id", "duplicate area %q", a.ID)
		}
		seen[a.ID] = true
	}
	if r := o.Redistribute; r != nil {
		rp := p + ".redistribute"
		switch r.DefaultRoute {
		case "", "none", MetricE1, MetricE2:
		default:
			v.errf(rp+".default_route", "must be none, e1 or e2 (the OSPF route type of the default route)")
		}
		if r.DefaultConditional && r.DefaultRoute != MetricE1 && r.DefaultRoute != MetricE2 {
			v.warnf(rp+".default_conditional", "has no effect: the default route is None")
		}
		if r.MetricType != "" && r.MetricType != MetricE1 && r.MetricType != MetricE2 {
			v.errf(rp+".metric_type", "must be e1 or e2")
		}
		v.rangeInt(rp+".metric", r.Metric, 0, 16777215)
	}
	seenSum := map[string]bool{}
	for i, sm := range o.Summaries {
		sp := fmt.Sprintf("%s.summaries[%d]", p, i)
		if pf, ok := v.prefixOf(sp+".subnet", sm.Subnet); ok {
			if seenSum[pf.String()] {
				v.errf(sp+".subnet", "duplicate summary %s", pf)
			}
			seenSum[pf.String()] = true
		}
		if sm.MetricType != "" && sm.MetricType != MetricE1 && sm.MetricType != MetricE2 {
			v.errf(sp+".metric_type", "must be e1 or e2")
		}
		v.rangeInt(sp+".metric", sm.Metric, 0, 16777215) // rule: rt-ospf-summary-external
		v.rangeInt(sp+".tag", sm.Tag, 1, 4294967295)
	}
}

func (v *validator) edgeRouting(p string, e *Edge, nsds map[string]bool) {
	pr := v.profiles[e.Profile]
	inProfile := func(seg string) bool {
		if pr == nil {
			return true
		}
		return hasString(pr.Segments, seg)
	}
	segOK := func(path, seg string) bool {
		if _, ok := v.segs[seg]; !ok && seg != GlobalSegmentID {
			v.errf(path, "unknown segment %q", seg)
			return false
		}
		if !inProfile(seg) {
			v.errf(path, "segment %q is not a segment of Profile %q", seg, e.Profile)
			return false
		}
		return true
	}
	// Connected subnets (LAN and routed interfaces with an address), for next-hop checks.
	var connected []netip.Prefix
	ifaces := map[string]*Interface{}
	for i := range e.Interfaces {
		in := &e.Interfaces[i]
		ifaces[in.ID] = in
		if (in.Kind == IfLan || in.Kind == IfRouted) && in.Address != "" {
			if pf, err := netip.ParsePrefix(in.Address); err == nil && pf.Addr().Is4() {
				connected = append(connected, pf.Masked())
			}
		}
	}
	r := e.Routing
	var ospfProfile *OSPFProfile
	if pr != nil && pr.Routing != nil {
		ospfProfile = pr.Routing.OSPF
	}

	// Static routes.
	seen := map[string]bool{}
	for i, sr := range r.StaticRoutes {
		sp := fmt.Sprintf("%s.static_routes[%d]", p, i)
		switch {
		case sr.ID == "":
			v.errf(sp+".id", "must not be empty")
		case seen[sr.ID]:
			v.errf(sp+".id", "duplicate static route id %q", sr.ID)
		}
		seen[sr.ID] = true
		segOK(sp+".segment", segOr(sr.Segment))
		v.prefixOf(sp+".subnet", sr.Subnet)
		v.rangeInt(sp+".cost", sr.Cost, 0, 255)
		var nh netip.Addr
		if sr.NextHop != "" {
			a, err := netip.ParseAddr(sr.NextHop)
			if err != nil || !a.Is4() {
				v.errf(sp+".next_hop", "%q is not an IPv4 address", sr.NextHop)
			} else {
				nh = a
			}
		}
		if sr.Interface != "" {
			if in := ifaces[sr.Interface]; in == nil || in.Kind != IfWan {
				v.errf(sp+".interface", "%q is not a WAN interface of this Edge (a WAN static route names its WAN interface)", sr.Interface)
			}
		} else if sr.NextHop == "" {
			v.errf(sp+".next_hop", "a LAN static route needs a next hop (or name a WAN interface)")
		} else if nh.IsValid() {
			inside := false
			for _, c := range connected {
				inside = inside || c.Contains(nh)
			}
			if !inside {
				v.errf(sp+".next_hop", "%s is not inside a connected subnet of this Edge and no WAN interface is given", nh)
			}
		}
		if sr.ICMPProbe && sr.NextHop == "" {
			v.warnf(sp+".icmp_probe", "the ICMP Probe checks the next hop, and this route has none")
		}
	}

	// OSPF interfaces.
	ospfIfaces := map[string]string{}
	if r.OSPF != nil {
		op := p + ".ospf"
		if ospfProfile == nil || !ospfProfile.Enabled { // rule: rt-ospf-profile-first
			v.errf(op, "OSPF must be activated in the Profile first (Routing & NAT > OSPF on Profile %q)", e.Profile)
		}
		for i, in := range r.OSPF.Interfaces {
			ip := fmt.Sprintf("%s.interfaces[%d]", op, i)
			if _, dup := ospfIfaces[in.ID]; dup {
				v.errf(ip+".id", "duplicate OSPF interface %q", in.ID)
			}
			ospfIfaces[in.ID] = in.Mode
			ifc := ifaces[in.ID]
			switch {
			case ifc == nil:
				v.errf(ip+".id", "unknown interface %q", in.ID)
			case ifc.Kind == IfWan || ifc.Kind == IfLoopback:
				v.errf(ip+".id", "OSPF on %s interface %q is not modelled (LAN interfaces only)", ifc.Kind, in.ID)
			}
			switch in.Mode {
			case OSPFActive:
				if ifc != nil && ifc.Kind == IfLan { // rule: rt-ospf-switched-passive
					v.errf(ip+".mode", "full OSPF needs a routed interface; the switched interface %q supports OSPF only as passive", in.ID) // rule: rt-ospf-full-needs-routed
				}
			case OSPFPassive:
			default:
				v.errf(ip+".mode", "must be %s or %s", OSPFActive, OSPFPassive)
			}
			if in.Area != "" && ospfProfile != nil {
				found := false
				for _, a := range ospfProfile.Areas {
					found = found || a.ID == in.Area
				}
				if !found {
					v.errf(ip+".area", "area %q is not in the Profile's area table", in.Area)
				}
			}
		}
	}

	// BGP.
	nbrs := map[string]*BGPNeighbor{}
	if b := r.BGP; b != nil {
		bp := p + ".bgp"
		if b.LocalASN < 1 || b.LocalASN > maxASN {
			v.errf(bp+".local_asn", "must be set to an ASN from 1 to %d", maxASN)
		}
		filters := map[string]*BGPFilter{}
		for i := range b.Filters {
			f := &b.Filters[i]
			fp := fmt.Sprintf("%s.filters[%d]", bp, i)
			switch {
			case f.ID == "":
				v.errf(fp+".id", "must not be empty")
			case filters[f.ID] != nil:
				v.errf(fp+".id", "duplicate filter id %q", f.ID)
			}
			filters[f.ID] = f
			if len(f.Rules) == 0 {
				v.errf(fp+".rules", "a filter needs at least one rule")
			}
			for j, rl := range f.Rules {
				rp := fmt.Sprintf("%s.rules[%d]", fp, j)
				if rl.Action != FilterPermit && rl.Action != FilterDeny { // rule: rt-filter-match
					v.errf(rp+".action", "must be %s or %s", FilterPermit, FilterDeny)
				}
				if rl.Match.Prefix != "" {
					v.prefixOf(rp+".match.prefix", rl.Match.Prefix)
				}
				v.rangeInt(rp+".match.metric", rl.Match.Metric, 0, maxASN)
				v.rangeInt(rp+".match.local_preference", rl.Match.LocalPreference, 0, maxASN)
				v.rangeInt(rp+".set.local_preference", rl.Set.LocalPreference, 0, maxASN) // rule: rt-filter-set-actions
				v.rangeInt(rp+".set.metric", rl.Set.Metric, 0, maxASN)
				for _, a := range rl.Set.ASPathPrepend {
					if a < 1 || a > maxASN {
						v.errf(rp+".set.as_path_prepend", "%d is not an ASN", a)
					}
				}
				if rl.Action == FilterDeny && (rl.Set.LocalPreference != nil || rl.Set.Metric != nil || len(rl.Set.ASPathPrepend) > 0) {
					v.warnf(rp+".set", "a deny rule never applies its set actions")
				}
			}
		}
		used := map[string]bool{}
		dup := map[string]bool{}
		for i := range b.Neighbors {
			n := &b.Neighbors[i]
			np := fmt.Sprintf("%s.neighbors[%d]", bp, i)
			switch {
			case n.ID == "":
				v.errf(np+".id", "must not be empty")
			case nbrs[n.ID] != nil:
				v.errf(np+".id", "duplicate neighbor id %q", n.ID)
			}
			nbrs[n.ID] = n
			seg := segOr(n.Segment)
			segOK(np+".segment", seg)
			if a, err := netip.ParseAddr(n.IP); err != nil || !a.Is4() {
				v.errf(np+".ip", "%q is not an IPv4 address", n.IP)
			} else if k := seg + "|" + n.IP; dup[k] {
				v.errf(np+".ip", "duplicate neighbor: %s is already a neighbor in segment %s", n.IP, seg)
			} else {
				dup[k] = true
			}
			if n.ASN < 1 || n.ASN > maxASN {
				v.errf(np+".asn", "must be an ASN from 1 to %d", maxASN)
			}
			derived := neighborType(b.LocalASN, n.ASN) // rule: rt-bgp-ibgp-ebgp
			if n.Type != "" && n.Type != BGPInternal && n.Type != BGPExternal {
				v.errf(np+".type", "must be %s or %s", BGPInternal, BGPExternal)
			} else if n.Type != "" && n.Type != derived {
				v.errf(np+".type", "%s contradicts the ASNs: local %d and neighbor %d make it %s", n.Type, b.LocalASN, n.ASN, derived)
			}
			for _, ref := range []struct{ field, id string }{{"inbound_filter", n.InboundFilter}, {"outbound_filter", n.OutboundFilter}} {
				if ref.id == "" {
					continue
				}
				used[ref.id] = true
				if filters[ref.id] == nil {
					v.errf(np+"."+ref.field, "unknown filter %q (filters are created in the Filter List first)", ref.id)
				}
			}
			if n.NSD != "" {
				if !nsds[n.NSD] {
					v.errf(np+".nsd", "unknown Non SD-WAN destination %q", n.NSD)
				}
				if f := filters[n.OutboundFilter]; f != nil && derived == BGPInternal { // rule: rt-nsd-ibgp-no-lp
					for _, rl := range f.Rules {
						if rl.Set.LocalPreference != nil {
							v.errf(np+".outbound_filter", "with iBGP to a Non SD-WAN destination the system cannot apply local preference via an outbound filter (filter %q sets it); use Metric or AS-path prepend", f.ID)
							break
						}
					}
				}
			}
		}
		for _, f := range b.Filters {
			if !used[f.ID] {
				v.warnf(fmt.Sprintf("%s.filters[%s]", bp, f.ID), "no neighbor uses this filter")
			}
		}
		if rd := b.Redistribute; rd != nil {
			switch rd.DefaultRoute {
			case "", DefaultNo, DefaultYes, DefaultConditional:
			default:
				v.errf(bp+".redistribute.default_route", "must be %s, %s or %s", DefaultNo, DefaultYes, DefaultConditional) // rule: rt-bgp-default-route
			}
			v.rangeInt(bp+".redistribute.ospf_metric", rd.OSPFMetric, 0, maxASN)
		}
	}

	// LAN routes.
	seenLR := map[string]bool{}
	for i, lr := range r.LANRoutes {
		lp := fmt.Sprintf("%s.lan_routes[%d]", p, i)
		switch {
		case lr.ID == "":
			v.errf(lp+".id", "must not be empty")
		case seenLR[lr.ID]:
			v.errf(lp+".id", "duplicate LAN route id %q", lr.ID)
		}
		seenLR[lr.ID] = true
		v.prefixOf(lp+".prefix", lr.Prefix)
		seg := segOr(lr.Segment)
		segOK(lp+".segment", seg)
		switch lr.Protocol {
		case ProtoOSPF:
			mode, ok := ospfIfaces[lr.Via]
			switch {
			case !ok:
				v.errf(lp+".via", "OSPF interface %q is not enabled under this Edge's ospf.interfaces", lr.Via)
			case mode == OSPFPassive:
				v.warnf(lp+".via", "interface %q is passive: it forms no adjacency, so this route is never learned (assumption 73)", lr.Via) // rule: assume-73
			}
			if seg != GlobalSegmentID { // rule: rt-ospf-global-segment
				v.errf(lp+".segment", "OSPF exists only on the Global Segment")
			}
			switch lr.OSPFType {
			case OSPFIntra, OSPFInter, OSPFE1, OSPFE2:
			default:
				v.errf(lp+".ospf_type", "must be intra, inter, e1 or e2")
			}
			if lr.BGP != nil {
				v.errf(lp+".bgp", "BGP attributes do not apply to an OSPF route")
			}
			v.rangeInt(lp+".metric", lr.Metric, 0, 16777215)
		case ProtoBGP:
			n := nbrs[lr.Via]
			if n == nil {
				v.errf(lp+".via", "BGP neighbor %q does not exist under this Edge's bgp.neighbors", lr.Via)
			} else if segOr(n.Segment) != seg {
				v.errf(lp+".segment", "neighbor %q is in segment %s", lr.Via, segOr(n.Segment))
			}
			if lr.OSPFType != "" || lr.Metric != nil {
				v.errf(lp+".ospf_type", "OSPF fields do not apply to a BGP route")
			}
			if lr.BGP != nil {
				for _, a := range lr.BGP.ASPath {
					if a < 1 || a > maxASN {
						v.errf(lp+".bgp.as_path", "%d is not an ASN", a)
					}
				}
				v.rangeInt(lp+".bgp.local_preference", lr.BGP.LocalPreference, 0, maxASN)
				v.rangeInt(lp+".bgp.med", lr.BGP.MED, 0, maxASN)
			}
		default:
			v.errf(lp+".protocol", "must be ospf or bgp")
		}
	}
}

func (v *validator) ofc(edges map[string]*Edge) {
	o := v.t.OverlayFlowControl
	if o == nil {
		return
	}
	segOK := func(path, seg string) {
		if _, ok := v.segs[seg]; !ok && seg != GlobalSegmentID {
			v.errf(path, "unknown segment %q", seg)
		}
	}
	seen := map[string]bool{}
	for i, pe := range o.PreferredExits {
		p := fmt.Sprintf("overlay_flow_control.preferred_exits[%d]", i)
		seg := segOr(pe.Segment)
		segOK(p+".segment", seg)
		if _, ok := v.prefixOf(p+".prefix", pe.Prefix); ok {
			if k := seg + "|" + pe.Prefix; seen[k] {
				v.errf(p+".prefix", "the Preferred VPN Exits of %s in segment %s are listed twice", pe.Prefix, seg)
			} else {
				seen[k] = true
			}
		}
		if len(pe.Exits) == 0 {
			v.errf(p+".exits", "list at least one exit")
		}
		dup := map[string]bool{}
		for _, x := range pe.Exits { // rule: rt-ofc-preferred-exits
			switch {
			case edges[x] == nil:
				v.errf(p+".exits", "%q is not an Edge", x)
			case dup[x]:
				v.errf(p+".exits", "exit %q is listed twice", x)
			}
			dup[x] = true
		}
	}
	seenPin := map[string]bool{}
	for i, pn := range o.Pins {
		p := fmt.Sprintf("overlay_flow_control.pins[%d]", i)
		seg := segOr(pn.Segment)
		segOK(p+".segment", seg)
		v.prefixOf(p+".prefix", pn.Prefix)
		if edges[pn.Exit] == nil { // rule: rt-ofc-pin
			v.errf(p+".exit", "%q is not an Edge", pn.Exit)
		}
		if k := seg + "|" + pn.Prefix + "|" + pn.Exit; seenPin[k] {
			v.errf(p, "duplicate pin of %s through %s", pn.Prefix, pn.Exit)
		} else {
			seenPin[k] = true
		}
	}
}

func (v *validator) nsdPrefixes() {
	for i := range v.t.NSDs {
		n := &v.t.NSDs[i]
		seen := map[string]bool{}
		for j, np := range n.Prefixes {
			p := fmt.Sprintf("non_sdwan_destinations[%s].prefixes[%d]", n.ID, j)
			if pf, ok := v.prefixOf(p+".prefix", np.Prefix); ok {
				if seen[pf.String()] {
					v.errf(p+".prefix", "duplicate prefix %s", pf)
				}
				seen[pf.String()] = true
			}
			v.rangeInt(p+".cost", np.Cost, 0, 255)
			if np.Advertise && n.Via == NSDViaGateway {
				v.warnf(p+".advertise", "Advertise cannot be edited for a destination via Gateway; the setting is ignored")
			}
		}
	}
}
