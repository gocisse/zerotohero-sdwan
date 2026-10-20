package sim

import (
	"fmt"
	"sort"
)

// Tunnel types.
const (
	TunGateway    = "gateway"
	TunHub        = "hub"
	TunDynamic    = "dynamic"
	TunNSDGateway = "nsd-gateway"
	TunNSDEdge    = "nsd-edge"
)

// Tunnel (and path) states. Paths only use up, degraded and down.
const (
	StUp             = "up"
	StDegraded       = "degraded"
	StDown           = "down"
	StNotEstablished = "not-established"
	StTornDown       = "torn-down"
	// StUnknown is used for every tunnel when the Orchestrator is unreachable: what Edges do then is not documented.
	StUnknown = "unknown"
)

// Reason codes: the engine's own reason set (the challenge mode picks from these).
const (
	CodeOK               = "ok"
	CodeCloudVPNOff      = "cloud_vpn_disabled"
	CodeB2HOff           = "branch_to_hub_disabled"
	CodeB2BOff           = "branch_to_branch_disabled"
	CodeDynamicOff       = "dynamic_disabled"
	CodeNoTraffic        = "no_traffic_yet"
	CodeTornDown         = "torn_down_after_silence"
	CodeNATBlocks        = "nat_blocks_direct"
	CodeBuildNever       = "build_decision_never"
	CodeLinkDown         = "link_down"
	CodeLinkDegraded     = "link_degraded"
	CodePeerUnreachable  = "peer_unreachable"
	CodeNoLinkPair       = "no_matching_link_pair"
	CodeNSDNotEnabled    = "nsd_not_enabled"
	CodeOrchestratorDown = "orchestrator_unreachable"
	CodeOneSiteOnly      = "branch_to_branch_one_site"
	CodeNoGateway        = "no_gateway"
	CodeStandby          = "standby"
	CodeClusterMember    = "cluster_member"
)

// Endpoint is one end of a tunnel.
type Endpoint struct {
	Kind string `json:"kind"` // edge, gateway or nsd
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Path is one WAN link pair a tunnel can use (a local link to a peer, as List Paths shows).
type Path struct {
	LocalIface string `json:"local_interface"`
	LocalLink  string `json:"local_link"`
	Remote     string `json:"remote"` // remote link id, or "Gateway public address" / "Gateway private address"
	State      string `json:"state"`
	Reason     string `json:"reason,omitempty"`
}

// Tunnel is one tunnel the engine computed, with its evidence.
type Tunnel struct {
	ID          string        `json:"id"`
	Type        string        `json:"type"`
	Mode        string        `json:"mode"` // static or dynamic (List Paths)
	A           Endpoint      `json:"a"`
	B           Endpoint      `json:"b"`
	Role        string        `json:"role,omitempty"` // active or standby (NSD), primary or secondary (Hub)
	HubIndex    int           `json:"hub_index,omitempty"`
	Paths       []Path        `json:"paths"`
	State       string        `json:"state"`
	ReasonCode  string        `json:"reason_code"`
	Reason      string        `json:"reason"`
	Basis       Basis         `json:"basis"`
	Also        []Basis       `json:"also,omitempty"`
	Notes       []Note        `json:"notes,omitempty"`
	NotModelled []NotModelled `json:"not_modelled,omitempty"`
	// Initiator is the Edge that started a dynamic tunnel.
	Initiator string `json:"initiator,omitempty"`
}

// GatewayAssignmentInfo says which Gateways an Edge uses and how that was decided.
type GatewayAssignmentInfo struct {
	Edge      string `json:"edge"`
	Primary   string `json:"primary"`
	Secondary string `json:"secondary,omitempty"`
	Derived   bool   `json:"derived"`
	Basis     Basis  `json:"basis"`
}

// ClusterAssignment says which cluster member a spoke tunnels to.
type ClusterAssignment struct {
	Spoke     string `json:"spoke"`
	Cluster   string `json:"cluster"`
	Member    string `json:"member"`
	LogicalID int    `json:"logical_id"`
	Members   int    `json:"members"`
	Basis     Basis  `json:"basis"`
}

// HubOrder is the ordered Hub list of one Edge.
type HubOrder struct {
	Edge  string   `json:"edge"`
	Hubs  []string `json:"hubs"`
	Basis Basis    `json:"basis"`
}

// TunnelSet is everything ComputeTunnels returns.
type TunnelSet struct {
	Topology           string                  `json:"topology"`
	Conditions         Conditions              `json:"conditions"`
	Tunnels            []Tunnel                `json:"tunnels"`
	GatewayAssignments []GatewayAssignmentInfo `json:"gateway_assignments"`
	ClusterAssignments []ClusterAssignment     `json:"cluster_assignments"`
	HubOrders          []HubOrder              `json:"hub_orders"`
	NotModelled        []NotModelled           `json:"not_modelled"`
}

func (ts *TunnelSet) addNM(rule, item string) NotModelled {
	nm := notModelled(rule, item)
	for _, x := range ts.NotModelled {
		if x.Rule == rule {
			return nm
		}
	}
	ts.NotModelled = append(ts.NotModelled, nm)
	return nm
}

func bases(rules ...string) []Basis {
	out := make([]Basis, 0, len(rules))
	for _, r := range rules {
		out = append(out, basisOf(r))
	}
	return out
}

func edgeEnd(e *edgeInfo) Endpoint { return Endpoint{Kind: "edge", ID: e.Edge.ID, Name: e.Edge.Name} }

func (c *overlayCtx) gatewayEnd(id string) Endpoint {
	return Endpoint{Kind: "gateway", ID: id, Name: id}
}

// aggregate reduces paths to one tunnel state: up if any path is up, else degraded if any is, else down.
func aggregate(paths []Path) (state, code, reason string) {
	if len(paths) == 0 {
		return StDown, CodeNoLinkPair, "no WAN link pair: a public link pairs only with a public link and a private link with a private link"
	}
	var up, deg int
	for _, p := range paths {
		switch p.State {
		case StUp:
			up++
		case StDegraded:
			deg++
		}
	}
	switch {
	case up > 0:
		return StUp, CodeOK, fmt.Sprintf("%d of %d path(s) up", up, len(paths))
	case deg > 0:
		return StDegraded, CodeLinkDegraded, "every usable path misses its SLA (a brownout)"
	}
	code = CodeLinkDown
	reason = "every path is down"
	for _, p := range paths {
		if p.Reason == "peer unreachable" {
			code, reason = CodePeerUnreachable, "the peer is unreachable, so every path is down"
		}
	}
	return StDown, code, reason
}

// edgePaths pairs the WAN links of two Edges: same type with same type (assumption 46).
func (c *overlayCtx) edgePaths(a, b *edgeInfo) []Path {
	out := []Path{}
	bothUnreachable := c.hubsDown[b.Edge.ID] || c.hubsDown[a.Edge.ID]
	for _, la := range a.Links {
		for _, lb := range b.Links {
			if la.Link.Type != lb.Link.Type { // rule: assume-46 // rule: assume-13
				continue
			}
			st := worse(c.linkState(la.Link.ID), c.linkState(lb.Link.ID))
			p := Path{LocalIface: la.IfaceID, LocalLink: la.Link.ID, Remote: lb.Link.ID, State: st}
			switch {
			case bothUnreachable: // rule: peer-unreachable
				p.State, p.Reason = StDown, "peer unreachable"
			case st == StDown:
				p.Reason = "link " + firstDown(c, la.Link.ID, lb.Link.ID) + " is down"
			case st == StDegraded:
				p.Reason = "link " + firstDegraded(c, la.Link.ID, lb.Link.ID) + " is degraded"
			}
			out = append(out, p)
		}
	}
	return out
}

func firstDown(c *overlayCtx, ids ...string) string {
	for _, id := range ids {
		if c.linksDown[id] {
			return id
		}
	}
	return ids[0]
}

func firstDegraded(c *overlayCtx, ids ...string) string {
	for _, id := range ids {
		if c.linksDegraded[id] {
			return id
		}
	}
	return ids[0]
}

// gatewayPaths: each WAN link reaches the Gateway's public (public links) or private (private links) address.
func (c *overlayCtx) gatewayPaths(a *edgeInfo, gw string) []Path {
	out := []Path{}
	for _, la := range a.Links {
		remote := "Gateway public address" // rule: assume-47 // rule: gateway-address-by-link
		if la.Link.Type == LinkPrivate {
			remote = "Gateway private address"
		}
		st := c.linkState(la.Link.ID)
		p := Path{LocalIface: la.IfaceID, LocalLink: la.Link.ID, Remote: remote, State: st}
		switch {
		case c.gwDown[gw]: // rule: peer-unreachable
			p.State, p.Reason = StDown, "peer unreachable"
		case st == StDown:
			p.Reason = "link " + la.Link.ID + " is down"
		case st == StDegraded:
			p.Reason = "link " + la.Link.ID + " is degraded"
		}
		out = append(out, p)
	}
	return out
}

// capPaths enforces the documented cap of 16 tunnels between an Edge and another Edge or Gateway.
func capPaths(t *Tunnel) {
	if len(t.Paths) > MaxEdgeTunnels { // rule: tunnel-cap
		n := len(t.Paths)
		t.Paths = t.Paths[:MaxEdgeTunnels]
		t.Notes = append(t.Notes, Note{Text: fmt.Sprintf("%d link pairs exist but at most %d tunnels are built between an Edge and another Edge or Gateway.", n, MaxEdgeTunnels), Basis: basisOf(RTunnelCap)})
	}
}

// ComputeTunnels returns every tunnel of the topology under the conditions, with state, reason and basis. The
// topology is never modified. The clock is used only for dynamic tunnels (see Conditions.Traffic).
func ComputeTunnels(t *Topology, cond Conditions, clock Clock) (*TunnelSet, error) {
	c, err := newOverlayCtx(t, cond)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		clock = SystemClock()
	}
	now := clock.Now()
	ts := &TunnelSet{Topology: t.Meta.ID, Conditions: cond, Tunnels: []Tunnel{}, GatewayAssignments: []GatewayAssignmentInfo{},
		ClusterAssignments: []ClusterAssignment{}, HubOrders: []HubOrder{}, NotModelled: []NotModelled{}}

	c.gatewayTunnels(ts)
	c.hubTunnels(ts)
	c.dynamicTunnels(ts, now)
	c.nsdTunnels(ts)

	if cond.OrchestratorUnreachable { // rule: nm-orchestrator-down
		nm := ts.addNM(RNMOrchestratorDown, "Edge behaviour with the Orchestrator unreachable")
		for i := range ts.Tunnels {
			tn := &ts.Tunnels[i]
			tn.State, tn.ReasonCode = StUnknown, CodeOrchestratorDown
			tn.Reason = "what an Edge does when the Orchestrator is unreachable is not documented"
			tn.NotModelled = append(tn.NotModelled, nm)
		}
	}
	order := map[string]int{TunGateway: 0, TunHub: 1, TunDynamic: 2, TunNSDGateway: 3, TunNSDEdge: 4}
	sort.SliceStable(ts.Tunnels, func(i, j int) bool {
		a, b := ts.Tunnels[i], ts.Tunnels[j]
		if order[a.Type] != order[b.Type] {
			return order[a.Type] < order[b.Type]
		}
		return a.ID < b.ID
	})
	return ts, nil
}

// gatewayTunnels: every Edge has tunnels to its primary and secondary Gateway.
func (c *overlayCtx) gatewayTunnels(ts *TunnelSet) {
	for _, e := range c.edges {
		info := GatewayAssignmentInfo{Edge: e.Edge.ID, Primary: e.Gateways[0], Secondary: e.Gateways[1], Derived: e.Derived, Basis: basisOf(RGatewayAssign)}
		if e.Derived {
			info.Basis = basisOf(RA44GatewayDerive)
		}
		ts.GatewayAssignments = append(ts.GatewayAssignments, info)
		for i, g := range e.Gateways {
			if g == "" || c.gwByID[g] == nil {
				continue
			}
			tn := Tunnel{ID: fmt.Sprintf("gateway:%s:%s", e.Edge.ID, g), Type: TunGateway, Mode: "static", A: edgeEnd(e), B: c.gatewayEnd(g), // rule: gateway-tunnels
				Paths: c.gatewayPaths(e, g), Basis: basisOf(RGatewayTunnels), Also: bases(RGatewayAssign, RA45AlwaysGateway, RPathsMode, RPathsLocalLinks)} // rule: paths-mode // rule: paths-local-links // rule: assume-45
			if i == 0 {
				tn.Role = "primary"
			} else {
				tn.Role = "secondary"
			}
			if e.Derived {
				tn.Also = append(tn.Also, basisOf(RA44GatewayDerive))
			}
			capPaths(&tn)
			tn.State, tn.ReasonCode, tn.Reason = aggregate(tn.Paths)
			ts.Tunnels = append(ts.Tunnels, tn)
		}
	}
}

// hubTunnels: with Cloud VPN and Branch to Hub on, every Edge builds a permanent tunnel to each Hub in the list.
func (c *overlayCtx) hubTunnels(ts *TunnelSet) {
	for _, e := range c.edges {
		hubs := e.CVPN.B2H.Hubs
		if len(hubs) == 0 {
			continue
		}
		ts.HubOrders = append(ts.HubOrders, HubOrder{Edge: e.Edge.ID, Hubs: append([]string{}, hubs...), Basis: basisOf(RHubOrder)})
		for idx, entry := range hubs { // rule: hub-designation
			target, cl, _ := c.resolveHub(entry, e)
			if target == nil {
				continue
			}
			tn := Tunnel{ID: fmt.Sprintf("hub:%s:%s", e.Edge.ID, target.Edge.ID), Type: TunHub, Mode: "static", A: edgeEnd(e), B: edgeEnd(target), HubIndex: idx + 1,
				Basis: basisOf(RB2HPermanent), Also: bases(RHubList, RCloudVPNSegment, RCloudVPNTypes, RPathsMode, RA46PathPairs)} // rule: cloud-vpn-tunnel-types // rule: b2h-permanent
			switch idx {
			case 0:
				tn.Role = "primary"
			case 1:
				tn.Role = "secondary"
			default:
				tn.Role = fmt.Sprintf("hub %d", idx+1)
			}
			if cl != nil { // rule: cluster-assignment
				tn.Also = append(tn.Also, basisOf(RClusterAssign), basisOf(RA48ClusterOrder))
				tn.Notes = append(tn.Notes, Note{Text: fmt.Sprintf("Hub entry %s is a cluster: %s is assigned by logical ID %d modulo %d.", entry, target.Edge.Name, e.logicalID(), len(cl.Members)), Basis: basisOf(RClusterAssign)})
				ts.ClusterAssignments = append(ts.ClusterAssignments, ClusterAssignment{Spoke: e.Edge.ID, Cluster: cl.ID, Member: target.Edge.ID, LogicalID: e.logicalID(), Members: len(cl.Members), Basis: basisOf(RClusterAssign)})
			}
			if idx > 0 {
				tn.Also = append(tn.Also, basisOf(RHubOrder))
			}
			switch {
			case !e.CVPN.Enabled: // rule: b2h-needs-cloud-vpn
				tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, CodeCloudVPNOff, "Cloud VPN is not enabled for this Edge (no segment), so no Branch to Hub tunnel is built"
				tn.Also = append(tn.Also, basisOf(RA42CloudVPNOff))                                              // rule: assume-42
				tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMCloudVPNDefault, "Cloud VPN default state")) // rule: nm-cloud-vpn-default
			case !e.CVPN.B2H.Enabled:
				tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, CodeB2HOff, "Enable Branch to Hubs is off for this Edge, so its Hubs get no tunnel"
			default:
				tn.Paths = c.edgePaths(e, target)
				capPaths(&tn)
				tn.State, tn.ReasonCode, tn.Reason = aggregate(tn.Paths)
				if target.Edge.NATBlocksIncomingVCMP { // rule: nm-hub-nat
					tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMHubNAT, "Hub Edge behind a NAT that blocks incoming UDP 2426"))
				}
			}
			if tn.Paths == nil {
				tn.Paths = []Path{}
			}
			if e.CVPN.FromOverride { // rule: nm-cloud-vpn-override-enable
				tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMCloudVPNGranular, "Edge Cloud VPN override")) // rule: nm-cloud-vpn-override-granularity
				tn.Also = append(tn.Also, basisOf(RA43OverrideWhole))                                             // rule: assume-43
			}
			ts.Tunnels = append(ts.Tunnels, tn)
		}
	}
}

// nsdTunnels builds NSD via Gateway (Gateway to destination) and NSD via Edge (Edge to destination) tunnels.
func (c *overlayCtx) nsdTunnels(ts *TunnelSet) {
	for i := range c.t.NSDs {
		n := &c.t.NSDs[i]
		dest := Endpoint{Kind: "nsd", ID: n.ID, Name: nameOr(n.Name, n.ID)}
		if n.Via == NSDViaGateway {
			enabled := false
			for _, e := range c.edges {
				if e.CVPN.Enabled && hasString(e.CVPN.NSDGateway, n.ID) { // rule: nsd-gateway-enable
					enabled = true
				}
			}
			count := n.Tunnels
			if count == 0 {
				count = NSDTunnelCap(n.Type, n.Mode) // rule: nsd-tunnel-counts
			}
			if cp := NSDTunnelCap(n.Type, n.Mode); count > cp {
				count = cp
			}
			for k := 1; k <= count; k++ {
				tn := Tunnel{ID: fmt.Sprintf("nsd-gateway:%s:%d", n.ID, k), Type: TunNSDGateway, Mode: "static", A: c.gatewayEnd(n.Gateway), B: dest, // rule: nsd-gateway-profile-level
					Basis: basisOf(RNSDCounts), Also: bases(RNSDGatewayProf, RA50NSDBuilder, RNSDGatewayTypes)} // rule: assume-50
				tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMNSDGatewayChoice, "Which Gateway builds the destination tunnels")) // rule: nm-nsd-gateway-choice
				if n.Mode == ModeHotStandby {                                                                                          // rule: assume-49
					tn.Role = "active"
					if k > 1 {
						tn.Role = "standby"
					}
					tn.Also = append(tn.Also, basisOf(RA49HotStandby))
				} else {
					tn.Role = "active"
				}
				switch {
				case !enabled:
					tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, CodeNSDNotEnabled, "no Profile with Cloud VPN lists this destination under Edge to Non SD-WAN via Gateway"
				case c.gwByID[n.Gateway] == nil:
					tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, CodeNoGateway, "the destination's Gateway is not defined"
				default:
					st, reason := StUp, ""
					switch {
					case c.gwDown[n.Gateway]: // rule: peer-unreachable
						st, reason = StDown, "the Gateway is unreachable"
					case c.nsdDown[n.ID]:
						st, reason = StDown, "the destination is unreachable"
					}
					tn.Paths = []Path{{LocalIface: "-", LocalLink: n.Gateway, Remote: n.ID, State: st, Reason: reason}}
					tn.State = st
					if st == StUp {
						tn.ReasonCode, tn.Reason = CodeOK, "IPsec tunnel built by the Gateway"
						if tn.Role == "standby" {
							tn.ReasonCode, tn.Reason = CodeStandby, "established but standby (Active/Hot-Standby)"
						}
					} else {
						tn.ReasonCode, tn.Reason = CodePeerUnreachable, reason
					}
				}
				if tn.Paths == nil {
					tn.Paths = []Path{}
				}
				ts.Tunnels = append(ts.Tunnels, tn)
			}
			continue
		}
		// NSD via Edge: one tunnel from the builder Edge's first public WAN link.
		b := c.edgeByID[n.Edge]
		tn := Tunnel{ID: fmt.Sprintf("nsd-edge:%s:%s", n.Edge, n.ID), Type: TunNSDEdge, Mode: "static", B: dest, Role: "active", // rule: nsd-edge-per-segment
			Basis: basisOf(RNSDEdgeSegment), Also: bases(RNSDEdgeTypes, RA50NSDBuilder)}
		if b == nil {
			continue
		}
		tn.A = edgeEnd(b)
		if !hasString(b.CVPN.NSDEdge, n.ID) {
			tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, CodeNSDNotEnabled, "the destination is not enabled for this Edge (Profile or Edge VPN Services)"
		} else {
			var first *wanLink
			for k := range b.Links {
				if b.Links[k].Link.Type == LinkPublic {
					first = &b.Links[k]
					break
				}
			}
			if first == nil {
				tn.State, tn.ReasonCode, tn.Reason = StDown, CodeNoLinkPair, "the Edge has no public WAN link to build the tunnel on"
			} else {
				st := c.linkState(first.Link.ID)
				reason := ""
				switch {
				case c.nsdDown[n.ID]:
					st, reason = StDown, "the destination is unreachable"
				case st != StUp:
					reason = "link " + first.Link.ID + " is " + st
				}
				tn.Paths = []Path{{LocalIface: first.IfaceID, LocalLink: first.Link.ID, Remote: n.ID, State: st, Reason: reason}}
				tn.State, tn.ReasonCode, tn.Reason = aggregate(tn.Paths)
				if c.nsdDown[n.ID] {
					tn.ReasonCode = CodePeerUnreachable
				}
			}
		}
		if tn.Paths == nil {
			tn.Paths = []Path{}
		}
		ts.Tunnels = append(ts.Tunnels, tn)
	}
}

func nameOr(name, id string) string {
	if name != "" {
		return name
	}
	return id
}
