package sim

import "fmt"

// wanLink is a WAN interface of an Edge with its link.
type wanLink struct {
	IfaceID string
	Link    *WanLink
}

// edgeInfo is an Edge with everything the overlay engine needs, resolved once.
type edgeInfo struct {
	Edge    *Edge
	Site    *Site
	Profile *Profile
	Index   int
	Links   []wanLink
	CVPN    effCloudVPN
	// Gateways are the primary and secondary Gateway ids; Derived is true when they came from the region.
	Gateways [2]string
	Derived  bool
}

// effCloudVPN is the Cloud VPN configuration an Edge ends up with (Profile merged with its override).
type effCloudVPN struct {
	Segments      []string
	Enabled       bool // Cloud VPN is on for at least one segment
	B2H           BranchToHub
	B2B           BranchToBranch
	DynamicOn     bool
	NSDGateway    []string
	NSDEdge       []string
	FromOverride  bool
	ProfileHasCfg bool
}

type overlayCtx struct {
	t        *Topology
	cond     Conditions
	edges    []*edgeInfo
	edgeByID map[string]*edgeInfo
	gwByID   map[string]*Gateway
	linkByID map[string]*WanLink
	clusters map[string]*Cluster
	nsdByID  map[string]*NSD

	linksDown, linksDegraded, hubsDown, gwDown, nsdDown map[string]bool
}

func newOverlayCtx(t *Topology, cond Conditions) (*overlayCtx, error) {
	if bad := t.validateConditions(cond); len(bad) > 0 {
		return nil, &ConditionError{Problems: bad}
	}
	c := &overlayCtx{t: t, cond: cond, edgeByID: map[string]*edgeInfo{}, gwByID: map[string]*Gateway{},
		linkByID: map[string]*WanLink{}, clusters: map[string]*Cluster{}, nsdByID: map[string]*NSD{},
		linksDown: setOf(cond.LinksDown), linksDegraded: setOf(cond.LinksDegraded), hubsDown: setOf(cond.HubsUnreachable),
		gwDown: setOf(cond.GatewaysUnreachable), nsdDown: setOf(cond.NSDsUnreachable)}
	for i := range t.Gateways {
		c.gwByID[t.Gateways[i].ID] = &t.Gateways[i]
	}
	for i := range t.Links {
		c.linkByID[t.Links[i].ID] = &t.Links[i]
	}
	for i := range t.Clusters {
		c.clusters[t.Clusters[i].ID] = &t.Clusters[i]
	}
	for i := range t.NSDs {
		c.nsdByID[t.NSDs[i].ID] = &t.NSDs[i]
	}
	idx := 0
	for si := range t.Sites {
		for ei := range t.Sites[si].Edges {
			e := &t.Sites[si].Edges[ei]
			p := t.profileByID(e.Profile)
			if p == nil {
				return nil, fmt.Errorf("edge %s references unknown profile %q", e.ID, e.Profile)
			}
			info := &edgeInfo{Edge: e, Site: &t.Sites[si], Profile: p, Index: idx}
			idx++
			for _, in := range e.Interfaces {
				if in.Kind == IfWan {
					if l, ok := c.linkByID[in.Link]; ok {
						info.Links = append(info.Links, wanLink{IfaceID: in.ID, Link: l})
					}
				}
			}
			info.CVPN = effectiveCloudVPN(p, e)
			c.assignGateways(info)
			c.edges = append(c.edges, info)
			c.edgeByID[e.ID] = info
		}
	}
	return c, nil
}

// effectiveCloudVPN merges the Profile's Cloud VPN settings with the Edge override.
func effectiveCloudVPN(p *Profile, e *Edge) effCloudVPN {
	var out effCloudVPN
	apply := func(c *CloudVPN, isOverride bool) {
		if c == nil {
			return
		}
		if isOverride {
			out.FromOverride = true
		} else {
			out.ProfileHasCfg = true
		}
		if c.Segments != nil || !isOverride {
			out.Segments = c.Segments
		}
		if c.BranchToHub != nil {
			out.B2H = *c.BranchToHub
		}
		if c.BranchToBranch != nil {
			out.B2B = *c.BranchToBranch
			out.DynamicOn = c.BranchToBranch.Dynamic == nil || *c.BranchToBranch.Dynamic // rule: dynamic-default-on
		}
		if !isOverride && c.BranchToBranch == nil {
			out.DynamicOn = true // rule: dynamic-default-on
		}
		if !isOverride && c.NSDViaGateway != nil { // rule: nsd-gateway-profile-level
			out.NSDGateway = c.NSDViaGateway
		}
		if c.NSDViaEdge != nil {
			out.NSDEdge = c.NSDViaEdge
		}
	}
	out.DynamicOn = true // rule: dynamic-default-on
	apply(p.CloudVPN, false)
	apply(e.Overrides.CloudVPN, true)   // rule: cloud-vpn-override
	out.Enabled = len(out.Segments) > 0 // rule: cloud-vpn-per-segment
	return out
}

// assignGateways resolves an Edge's primary and secondary Gateway.
func (c *overlayCtx) assignGateways(info *edgeInfo) {
	e := info.Edge
	if e.Gateways != nil { // rule: gateway-assignment
		info.Gateways[0], info.Gateways[1] = e.Gateways.Primary, e.Gateways.Secondary
		return
	}
	info.Derived = true // rule: assume-44
	var same, other []string
	for _, g := range c.t.Gateways {
		if g.Region == info.Site.Region {
			same = append(same, g.ID)
		} else {
			other = append(other, g.ID)
		}
	}
	all := append(append([]string{}, same...), other...)
	if len(all) > 0 {
		info.Gateways[0] = all[0]
	}
	if len(all) > 1 {
		info.Gateways[1] = all[1]
	}
}

// linkState is the state of one WAN link under the conditions: down, degraded or up.
func (c *overlayCtx) linkState(id string) string {
	switch {
	case c.linksDown[id]: // rule: link-outage-brownout
		return "down"
	case c.linksDegraded[id]: // rule: assume-55
		return "degraded"
	}
	return "up"
}

// worse returns the worse of two states (down, degraded, up).
func worse(a, b string) string {
	rank := map[string]int{"up": 0, "degraded": 1, "down": 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// logicalID is an Edge's logical id: authored, else its position in the file.
func (e *edgeInfo) logicalID() int {
	if e.Edge.LogicalID != nil { // rule: assume-48
		return *e.Edge.LogicalID
	}
	return e.Index
}

// resolveHub turns a Hub list entry (an Edge or a cluster) into the Edge a spoke tunnels to.
func (c *overlayCtx) resolveHub(entry string, spoke *edgeInfo) (target *edgeInfo, cl *Cluster, why string) {
	if cluster, ok := c.clusters[entry]; ok { // rule: cluster-single-hub
		cl = cluster
		var members []*edgeInfo
		for _, m := range cluster.Members {
			if mi := c.edgeByID[m]; mi != nil {
				members = append(members, mi)
			}
		}
		if len(members) == 0 {
			return nil, cl, "the cluster has no members"
		}
		for _, m := range members {
			if m.Edge.ID == spoke.Edge.ID {
				return nil, cl, "this Edge is a member of the cluster"
			}
		}
		return members[spoke.logicalID()%len(members)], cl, "" // rule: cluster-assignment
	}
	if h := c.edgeByID[entry]; h != nil {
		if h.Edge.ID == spoke.Edge.ID {
			return nil, nil, "this Edge is the Hub"
		}
		return h, nil, ""
	}
	return nil, nil, "unknown Hub " + entry
}
