package sim

import (
	"fmt"
	"strings"
)

// overlay validates the optional overlay keys (step 12): Gateway roles, link MTU, Edge overlay attributes,
// clusters, Non SD-WAN destinations and the Cloud VPN settings of Profiles and Edge overrides.
func (v *validator) overlay() {
	t := v.t
	edges := map[string]*Edge{}
	edgeSite := map[string]string{}
	var order []string
	for si := range t.Sites {
		for ei := range t.Sites[si].Edges {
			e := &t.Sites[si].Edges[ei]
			if _, dup := edges[e.ID]; !dup {
				edges[e.ID] = e
				edgeSite[e.ID] = t.Sites[si].ID
				order = append(order, e.ID)
			}
		}
	}
	gws := map[string]*Gateway{}
	for i := range t.Gateways {
		g := &t.Gateways[i]
		gws[g.ID] = g
		for _, r := range g.Roles { // rule: gateway-roles
			if r != RoleDataPlane && r != RoleSecureVPN {
				v.errf(fmt.Sprintf("gateways[%s].roles", g.ID), "unknown role %q; documented roles used here are %s and %s", r, RoleDataPlane, RoleSecureVPN)
			}
		}
	}
	for i := range t.Links {
		l := &t.Links[i]
		if l.MTU == nil {
			continue
		}
		p := fmt.Sprintf("links[%s].mtu", l.ID)
		switch {
		case *l.MTU < 1:
			v.errf(p, "must be positive")
		case *l.MTU > 1500: // rule: mtu-no-jumbo
			v.errf(p, "%d is above 1500: jumbo frames are not supported", *l.MTU)
		case *l.MTU < 1300: // rule: mtu-1300-floor
			v.warnf(p, "%d is below 1300: path MTU discovery measures only down to 1300, so such an MTU must be configured manually", *l.MTU)
		}
	}
	for _, id := range order {
		e := edges[id]
		p := fmt.Sprintf("edges[%s]", e.ID)
		if e.LogicalID != nil && *e.LogicalID < 0 {
			v.errf(p+".logical_id", "must not be negative")
		}
		if a := e.Gateways; a != nil { // rule: gateway-assignment
			if _, ok := gws[a.Primary]; !ok {
				v.errf(p+".gateways.primary", "unknown Gateway %q", a.Primary)
			}
			if a.Secondary != "" {
				if _, ok := gws[a.Secondary]; !ok {
					v.errf(p+".gateways.secondary", "unknown Gateway %q", a.Secondary)
				}
				if a.Secondary == a.Primary {
					v.errf(p+".gateways", "primary and secondary Gateway must differ")
				}
			}
		}
	}
	v.clusters(edges)
	nsds := v.nsds(edges, gws)

	hubTargets := map[string]bool{}
	for id := range edges {
		hubTargets[id] = true
	}
	for _, c := range t.Clusters {
		hubTargets[c.ID] = true
	}
	for i := range t.Profiles {
		pr := &t.Profiles[i]
		if pr.CloudVPN != nil {
			v.cloudVPN(fmt.Sprintf("profiles[%s].cloud_vpn", pr.ID), pr.CloudVPN, false, pr.Segments, nil, hubTargets, nsds)
		}
	}
	for _, id := range order {
		e := edges[id]
		cv := e.Overrides.CloudVPN
		if cv == nil {
			continue
		}
		var segs []string
		var profB2H *BranchToHub
		if pr := v.profiles[e.Profile]; pr != nil {
			segs = pr.Segments
			if pr.CloudVPN != nil {
				profB2H = pr.CloudVPN.BranchToHub
			}
		}
		v.cloudVPN(fmt.Sprintf("edges[%s].overrides.cloud_vpn", e.ID), cv, true, segs, profB2H, hubTargets, nsds) // rule: cloud-vpn-override
	}
	v.tunnelCapWarnings(edges, order)
}

func (v *validator) clusters(edges map[string]*Edge) {
	seen := map[string]bool{}
	member := map[string]string{}
	for i := range v.t.Clusters {
		c := &v.t.Clusters[i]
		p := fmt.Sprintf("clusters[%d]", i)
		if !slugRe.MatchString(c.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", c.ID)
			continue
		}
		p = fmt.Sprintf("clusters[%s]", c.ID)
		if seen[c.ID] {
			v.errf(p, "duplicate cluster id %q", c.ID)
		}
		seen[c.ID] = true
		if _, clash := edges[c.ID]; clash {
			v.errf(p, "cluster id %q is also an Edge id: Hub lists name either an Edge or a cluster", c.ID)
		}
		if len(c.Members) < 2 { // rule: cluster-single-hub
			v.warnf(p+".members", "a Hub cluster groups several Edges; this one has %d", len(c.Members))
		}
		ms := map[string]bool{}
		models := map[string]bool{}
		for _, m := range c.Members {
			e, ok := edges[m]
			switch {
			case !ok:
				v.errf(p+".members", "%q is not an Edge of this topology", m)
				continue
			case ms[m]:
				v.errf(p+".members", "member %q listed twice", m)
			}
			ms[m] = true
			if other, dup := member[m]; dup && other != c.ID {
				v.warnf(p+".members", "Edge %q is already a member of cluster %q", m, other)
			}
			member[m] = c.ID
			models[e.Model] = true
		}
		if len(models) > 1 { // rule: cluster-same-model
			v.warnf(p+".members", "members should be the same model; found %d different models", len(models))
		}
	}
}

// nsds validates the Non SD-WAN destinations and returns them by id.
func (v *validator) nsds(edges map[string]*Edge, gws map[string]*Gateway) map[string]*NSD {
	out := map[string]*NSD{}
	viaEdge := map[string]string{}
	for i := range v.t.NSDs {
		n := &v.t.NSDs[i]
		p := fmt.Sprintf("non_sdwan_destinations[%d]", i)
		if !slugRe.MatchString(n.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", n.ID)
			continue
		}
		p = fmt.Sprintf("non_sdwan_destinations[%s]", n.ID)
		if _, dup := out[n.ID]; dup {
			v.errf(p, "duplicate destination id %q", n.ID)
			continue
		}
		out[n.ID] = n
		switch n.Via {
		case NSDViaGateway: // rule: nsd-gateway-types
			if !hasString(nsdGatewayTypes, n.Type) {
				v.errf(p+".type", "%q is not a documented via-Gateway type; allowed: %s", n.Type, strings.Join(nsdGatewayTypes, ", "))
			}
			if n.Mode != ModeActiveActive && n.Mode != ModeHotStandby {
				v.errf(p+".mode", "must be %s or %s for a via-Gateway destination, got %q", ModeActiveActive, ModeHotStandby, n.Mode)
			}
			cp := NSDTunnelCap(n.Type, n.Mode) // rule: nsd-tunnel-counts
			if n.Tunnels < 1 || n.Tunnels > cp {
				v.errf(p+".tunnels", "must be between 1 and %d for type %s in mode %s, got %d", cp, n.Type, n.Mode, n.Tunnels)
			}
			g, ok := gws[n.Gateway]
			switch {
			case n.Gateway == "":
				v.errf(p+".gateway", "the Gateway that builds the tunnels is required for a via-Gateway destination")
			case !ok:
				v.errf(p+".gateway", "unknown Gateway %q", n.Gateway)
			case len(g.Roles) > 0 && !hasString(g.Roles, RoleSecureVPN):
				v.errf(p+".gateway", "Gateway %q lacks the %s role that builds IPsec tunnels to Non SD-WAN destinations", n.Gateway, RoleSecureVPN) // rule: gateway-roles
			}
			if n.Edge != "" {
				v.errf(p+".edge", "applies only to a via-Edge destination")
			}
		case NSDViaEdge: // rule: nsd-edge-types
			if !hasString(nsdEdgeTypes, n.Type) {
				v.errf(p+".type", "%q is not a documented via-Edge type; allowed: %s", n.Type, strings.Join(nsdEdgeTypes, ", "))
			}
			if n.Tunnels > 1 {
				v.errf(p+".tunnels", "a via-Edge destination has one tunnel, got %d", n.Tunnels)
			}
			if n.Mode != "" {
				v.errf(p+".mode", "applies only to a via-Gateway destination")
			}
			if n.Gateway != "" {
				v.errf(p+".gateway", "applies only to a via-Gateway destination")
			}
			seg := n.Segment
			if seg == "" {
				seg = GlobalSegmentID
			}
			if _, ok := v.segs[seg]; !ok {
				v.errf(p+".segment", "unknown segment %q", seg)
			}
			switch {
			case n.Edge == "":
				v.errf(p+".edge", "the Edge that builds the tunnel is required for a via-Edge destination")
			case edges[n.Edge] == nil:
				v.errf(p+".edge", "unknown Edge %q", n.Edge)
			default:
				key := n.Edge + "/" + seg
				if other, dup := viaEdge[key]; dup { // rule: nsd-edge-per-segment
					v.errf(p, "Edge %q already has a via-Edge destination (%s) in segment %q: one service per segment", n.Edge, other, seg)
				}
				viaEdge[key] = n.ID
			}
		default:
			v.errf(p+".via", "must be %q or %q, got %q", NSDViaGateway, NSDViaEdge, n.Via)
		}
	}
	return out
}

// cloudVPN validates a Cloud VPN block of a Profile or of an Edge override.
func (v *validator) cloudVPN(path string, cv *CloudVPN, isOverride bool, profileSegs []string, profileB2H *BranchToHub, hubTargets map[string]bool, nsds map[string]*NSD) {
	seen := map[string]bool{}
	for _, s := range cv.Segments { // rule: cloud-vpn-per-segment
		if _, ok := v.segs[s]; !ok {
			v.errf(path+".segments", "unknown segment %q", s)
		} else if !hasString(profileSegs, s) {
			v.errf(path+".segments", "segment %q is not part of the Profile's segments", s)
		}
		if seen[s] {
			v.errf(path+".segments", "segment %q listed twice", s)
		}
		seen[s] = true
	}
	b2h := profileB2H
	if cv.BranchToHub != nil {
		b2h = cv.BranchToHub
		h := cv.BranchToHub
		hs := map[string]bool{}
		for _, id := range h.Hubs { // rule: hub-designation
			if !hubTargets[id] {
				v.errf(path+".branch_to_hub.hubs", "%q is neither an Edge nor a cluster of this topology", id)
			}
			if hs[id] {
				v.errf(path+".branch_to_hub.hubs", "Hub %q listed twice: Hub order must be unambiguous", id) // rule: hub-order-tiebreak
			}
			hs[id] = true
		}
		for _, id := range h.BackhaulHubs {
			if !hubTargets[id] {
				v.errf(path+".branch_to_hub.backhaul_hubs", "%q is neither an Edge nor a cluster of this topology", id)
			}
		}
		if h.Enabled && len(h.Hubs) == 0 {
			v.errf(path+".branch_to_hub", "Enable Branch to Hubs needs at least one Hub")
		}
		if !h.Enabled && len(h.Hubs) > 0 {
			v.warnf(path+".branch_to_hub", "Hubs are listed but Enable Branch to Hubs is off")
		}
	}
	if b := cv.BranchToBranch; b != nil { // rule: b2b-options
		switch b.Via {
		case B2BCloudGateways, B2BHubsForVPN:
		case "":
			if b.Enabled {
				v.errf(path+".branch_to_branch.via", "must be %s or %s when Branch to Branch VPN is enabled", B2BCloudGateways, B2BHubsForVPN)
			}
		default:
			v.errf(path+".branch_to_branch.via", "must be %s or %s, got %q", B2BCloudGateways, B2BHubsForVPN, b.Via)
		}
		if b.Enabled && b.Via == B2BHubsForVPN && (b2h == nil || !b2h.Enabled || len(b2h.Hubs) == 0) {
			v.errf(path+".branch_to_branch", "Hubs for VPN needs Branch to Hub enabled with at least one Hub")
		}
		if b.Dynamic != nil && !b.Enabled {
			v.warnf(path+".branch_to_branch.dynamic", "Dynamic Branch to Branch is set but Branch to Branch VPN is not enabled")
		}
	}
	for _, id := range cv.NSDViaGateway {
		n := nsds[id]
		switch {
		case isOverride: // rule: nsd-gateway-profile-level
			v.errf(path+".nsd_via_gateway", "Non SD-WAN destinations via Gateway are defined at Profile level only and cannot be set on an Edge")
		case n == nil:
			v.errf(path+".nsd_via_gateway", "unknown Non SD-WAN destination %q", id)
		case n.Via != NSDViaGateway:
			v.errf(path+".nsd_via_gateway", "destination %q is not a via-Gateway destination", id)
		}
	}
	for _, id := range cv.NSDViaEdge {
		n := nsds[id]
		switch {
		case n == nil:
			v.errf(path+".nsd_via_edge", "unknown Non SD-WAN destination %q", id)
		case n.Via != NSDViaEdge:
			v.errf(path+".nsd_via_edge", "destination %q is not a via-Edge destination", id)
		}
	}
	if len(cv.Segments) == 0 && len(cv.NSDViaGateway) > 0 && !isOverride { // rule: nsd-gateway-enable
		v.warnf(path+".nsd_via_gateway", "Edge to Non SD-WAN via Gateway needs Cloud VPN on (no segments are listed)")
	}
}

// tunnelCapWarnings warns when the link counts of two Edges would exceed the documented 16-tunnel cap (the engine
// then builds only 16).
func (v *validator) tunnelCapWarnings(edges map[string]*Edge, order []string) {
	links := map[string]*WanLink{}
	for i := range v.t.Links {
		links[v.t.Links[i].ID] = &v.t.Links[i]
	}
	count := func(e *Edge) (pub, priv int) {
		for _, in := range e.Interfaces {
			if in.Kind != IfWan {
				continue
			}
			if l := links[in.Link]; l != nil {
				if l.Type == LinkPublic {
					pub++
				} else {
					priv++
				}
			}
		}
		return
	}
	for i, a := range order {
		pa, va := count(edges[a])
		if pa+va > MaxEdgeTunnels { // rule: tunnel-cap
			v.warnf(fmt.Sprintf("edges[%s].interfaces", a), "%d WAN links would exceed the cap of %d tunnels to a Gateway; only %d are built", pa+va, MaxEdgeTunnels, MaxEdgeTunnels)
		}
		for _, b := range order[i+1:] {
			pb, vb := count(edges[b])
			if n := pa*pb + va*vb; n > MaxEdgeTunnels {
				v.warnf(fmt.Sprintf("edges[%s].interfaces", a), "%d same-type link pairs with Edge %s exceed the cap of %d tunnels; only %d are built", n, b, MaxEdgeTunnels, MaxEdgeTunnels)
			}
		}
	}
}
