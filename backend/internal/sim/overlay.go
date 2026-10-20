package sim

// Overlay extension of the topology schema (step 12). Every key here is OPTIONAL, so topology files written for
// step 11 stay valid under strict decoding. See docs/SIMULATOR-MODEL.md sections 8 and 17.

// Branch to Branch options.
const (
	B2BCloudGateways = "cloud_gateways"
	B2BHubsForVPN    = "hubs_for_vpn"
)

// BranchToHub is the "Branch to Hub Site (Permanent VPN)" section: Enable Branch to Hubs and Edit Hubs.
// Hubs is the ordered Hub list: the order is the Hub order. Entries are Edge ids or cluster ids.
type BranchToHub struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	Hubs         []string `yaml:"hubs" json:"hubs,omitempty"`
	BackhaulHubs []string `yaml:"backhaul_hubs" json:"backhaul_hubs,omitempty"`
}

// BranchToBranch is "Branch to Branch VPN (Transit & Dynamic)". Dynamic nil means "the documented default":
// the Orchestrator configures Dynamic Branch to Branch for all Edges.
type BranchToBranch struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Via     string `yaml:"via" json:"via,omitempty"`
	Dynamic *bool  `yaml:"dynamic" json:"dynamic,omitempty"`
}

// CloudVPN is the VPN Services > Cloud VPN area of a Profile, or of an Edge override. A nil sub-section means
// "not set here": in a Profile it is off, in an Edge override it is inherited (assumptions 42 and 43).
type CloudVPN struct {
	// Segments Cloud VPN is activated for; Cloud VPN is configured per segment. Empty means off.
	Segments       []string        `yaml:"segments" json:"segments,omitempty"`
	BranchToHub    *BranchToHub    `yaml:"branch_to_hub" json:"branch_to_hub,omitempty"`
	BranchToBranch *BranchToBranch `yaml:"branch_to_branch" json:"branch_to_branch,omitempty"`
	// NSDViaGateway lists Non SD-WAN destinations via Gateway ("Enable Edge to Non SD-WAN via Gateway"). Profile only.
	NSDViaGateway []string `yaml:"nsd_via_gateway" json:"nsd_via_gateway,omitempty"`
	// NSDViaEdge lists Non SD-WAN destinations via Edge enabled for these Edges.
	NSDViaEdge []string `yaml:"nsd_via_edge" json:"nsd_via_edge,omitempty"`
}

// Cluster is a Hub cluster: other Edges treat it as one Hub.
type Cluster struct {
	ID            string   `yaml:"id" json:"id"`
	Name          string   `yaml:"name" json:"name,omitempty"`
	Members       []string `yaml:"members" json:"members"`
	AutoRebalance bool     `yaml:"auto_rebalance" json:"auto_rebalance,omitempty"`
}

// NSD via and mode values.
const (
	NSDViaGateway    = "gateway"
	NSDViaEdge       = "edge"
	ModeActiveActive = "active_active"
	ModeHotStandby   = "active_hot_standby"
)

// NSD is a Non SD-WAN destination reached by an IPsec tunnel set built by a Gateway or by an Edge.
type NSD struct {
	ID   string `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name,omitempty"`
	Via  string `yaml:"via" json:"via"`
	Type string `yaml:"type" json:"type"`
	// Mode and Tunnels apply to via gateway: Tunnels is the number of tunnels (up to the documented cap).
	Mode    string `yaml:"mode" json:"mode,omitempty"`
	Tunnels int    `yaml:"tunnels" json:"tunnels,omitempty"`
	// Gateway builds the tunnels (via gateway); Edge builds the tunnel (via edge).
	Gateway string `yaml:"gateway" json:"gateway,omitempty"`
	Edge    string `yaml:"edge" json:"edge,omitempty"`
	// Segment is the segment of a via-edge service (one service per segment). Default: the Global Segment.
	Segment string `yaml:"segment" json:"segment,omitempty"`
	// Prefixes are the destination's static routes (step 13).
	Prefixes []NSDPrefix `yaml:"prefixes" json:"prefixes,omitempty"`
}

// GatewayAssignment pins an Edge's primary and secondary Gateway instead of deriving them from the region.
type GatewayAssignment struct {
	Primary   string `yaml:"primary" json:"primary"`
	Secondary string `yaml:"secondary" json:"secondary,omitempty"`
}

// Gateway roles (the roles documented for a Gateway).
const (
	RoleDataPlane = "data_plane"
	RoleSecureVPN = "secure_vpn"
)

// NSD types accepted per kind (the lists in the Network Services pages).
var (
	nsdGatewayTypes = []string{"aws_vpn_gateway", "check_point", "cisco_asa", "cisco_isr", "generic_ikev2", "azure_virtual_hub", "palo_alto", "sonicwall", "zscaler", "generic_ikev1", "generic_firewall"}
	nsdEdgeTypes    = []string{"generic_ikev1", "generic_ikev2", "azure_virtual_wan"}
)

// NSDTunnelCap returns the documented maximum tunnels of a via-gateway destination.
func NSDTunnelCap(typ, mode string) int {
	switch typ { // rule: nsd-tunnel-counts
	case "cisco_asa", "generic_firewall":
		return 1
	case "azure_virtual_hub":
		return 2
	}
	if mode == ModeHotStandby {
		return 2
	}
	return 4
}

// MaxEdgeTunnels is the documented cap on tunnels between an Edge and another Edge or Gateway.
const MaxEdgeTunnels = 16 // rule: tunnel-cap
