package sim

// Routing extension of the topology schema (step 13). Every key here is OPTIONAL, so topology files written for
// steps 11 and 12 stay valid under strict decoding. See docs/SIMULATOR-MODEL.md sections 10, 11 and 18.

// ProfileRouting is the Routing & NAT area of a Profile. Only OSPF lives here: static routes are an Edge setting
// and BGP is authored per Edge (assumption 78).
type ProfileRouting struct {
	OSPF *OSPFProfile `yaml:"ospf" json:"ospf,omitempty"`
}

// OSPFProfile is the Profile's OSPF area table and redistribution settings. OSPF must be activated at the Profile
// first; the area table fields cannot be overridden at the Edge level.
type OSPFProfile struct {
	Enabled      bool              `yaml:"enabled" json:"enabled"`
	Areas        []OSPFArea        `yaml:"areas" json:"areas,omitempty"`
	Redistribute *OSPFRedistribute `yaml:"redistribute" json:"redistribute,omitempty"`
	Summaries    []OSPFSummary     `yaml:"summaries" json:"summaries,omitempty"`
}

// OSPFArea is one area. Only the Normal type exists.
type OSPFArea struct {
	ID string `yaml:"id" json:"id"`
}

// OSPF metric types of redistributed (external) routes and the default route.
const (
	MetricE1 = "e1"
	MetricE2 = "e2"
)

// OSPFRedistribute is the redistribution block of the OSPF settings. A nil pointer means "not set in the
// topology": the Assumptions object supplies the value (the defaults are VERIFY, assumption 22).
type OSPFRedistribute struct {
	OverlayPrefixes *bool `yaml:"overlay_prefixes" json:"overlay_prefixes,omitempty"`
	BGP             *bool `yaml:"bgp" json:"bgp,omitempty"`
	// DefaultRoute is none (documented default), e1 or e2.
	DefaultRoute string `yaml:"default_route" json:"default_route,omitempty"`
	// DefaultConditional: the default route is redistributed only when the Edge learns one (Advertise = Conditional).
	DefaultConditional bool   `yaml:"default_conditional" json:"default_conditional,omitempty"`
	Metric             *int   `yaml:"metric" json:"metric,omitempty"`
	MetricType         string `yaml:"metric_type" json:"metric_type,omitempty"`
}

// OSPFSummary is one row of the Route Summarization table (external routes only).
type OSPFSummary struct {
	Subnet      string `yaml:"subnet" json:"subnet"`
	NoAdvertise bool   `yaml:"no_advertise" json:"no_advertise,omitempty"`
	MetricType  string `yaml:"metric_type" json:"metric_type,omitempty"`
	Metric      *int   `yaml:"metric" json:"metric,omitempty"`
	Tag         *int   `yaml:"tag" json:"tag,omitempty"`
}

// EdgeRouting is the routing input of one Edge: static routes, what the LAN routers offer, OSPF interfaces and BGP.
type EdgeRouting struct {
	StaticRoutes []StaticRoute `yaml:"static_routes" json:"static_routes,omitempty"`
	// LANRoutes stands in for "what the LAN router would advertise to this Edge via OSPF or BGP".
	LANRoutes []LANRoute `yaml:"lan_routes" json:"lan_routes,omitempty"`
	OSPF      *EdgeOSPF  `yaml:"ospf" json:"ospf,omitempty"`
	BGP       *BGPConfig `yaml:"bgp" json:"bgp,omitempty"`
}

// StaticRoute is one row of Static Route Settings. Interface names a WAN interface (a WAN static route); without
// it the route is a LAN static route and the next hop must be inside a connected subnet.
type StaticRoute struct {
	ID          string `yaml:"id" json:"id"`
	Segment     string `yaml:"segment" json:"segment,omitempty"`
	Subnet      string `yaml:"subnet" json:"subnet"`
	NextHop     string `yaml:"next_hop" json:"next_hop,omitempty"`
	Interface   string `yaml:"interface" json:"interface,omitempty"`
	Cost        *int   `yaml:"cost" json:"cost,omitempty"`
	Preferred   bool   `yaml:"preferred" json:"preferred,omitempty"`
	Advertise   bool   `yaml:"advertise" json:"advertise,omitempty"`
	ICMPProbe   bool   `yaml:"icmp_probe" json:"icmp_probe,omitempty"`
	Description string `yaml:"description" json:"description,omitempty"`
}

// LAN route protocols and OSPF route types.
const (
	ProtoOSPF = "ospf"
	ProtoBGP  = "bgp"

	OSPFIntra = "intra"
	OSPFInter = "inter"
	OSPFE1    = "e1"
	OSPFE2    = "e2"
)

// LANRoute is one prefix a LAN router offers. Via is an OSPF interface id (protocol ospf) or a BGP neighbor id.
type LANRoute struct {
	ID       string `yaml:"id" json:"id"`
	Prefix   string `yaml:"prefix" json:"prefix"`
	Segment  string `yaml:"segment" json:"segment,omitempty"`
	Protocol string `yaml:"protocol" json:"protocol"`
	Via      string `yaml:"via" json:"via"`
	// OSPFType (protocol ospf): intra, inter, e1 or e2. Metric is the OSPF cost.
	OSPFType string `yaml:"ospf_type" json:"ospf_type,omitempty"`
	Metric   *int   `yaml:"metric" json:"metric,omitempty"`
	// BGP (protocol bgp): the attributes the neighbor sends.
	BGP *BGPAttrs `yaml:"bgp" json:"bgp,omitempty"`
}

// BGPAttrs are the BGP attributes of an authored LAN route.
type BGPAttrs struct {
	ASPath          []int `yaml:"as_path" json:"as_path,omitempty"`
	LocalPreference *int  `yaml:"local_preference" json:"local_preference,omitempty"`
	MED             *int  `yaml:"med" json:"med,omitempty"`
}

// OSPF interface modes.
const (
	OSPFActive  = "active"
	OSPFPassive = "passive"
)

// EdgeOSPF lists the Edge interfaces OSPF is enabled on.
type EdgeOSPF struct {
	Interfaces []OSPFInterface `yaml:"interfaces" json:"interfaces,omitempty"`
}

// OSPFInterface enables OSPF on an interface: active (full OSPF, routed interfaces only) or passive.
type OSPFInterface struct {
	ID   string `yaml:"id" json:"id"`
	Mode string `yaml:"mode" json:"mode"`
	Area string `yaml:"area" json:"area,omitempty"`
}

// BGPConfig is the BGP area of an Edge (authored per Edge, assumption 78).
type BGPConfig struct {
	LocalASN     int              `yaml:"local_asn" json:"local_asn"`
	Neighbors    []BGPNeighbor    `yaml:"neighbors" json:"neighbors,omitempty"`
	Redistribute *BGPRedistribute `yaml:"redistribute" json:"redistribute,omitempty"`
	Filters      []BGPFilter      `yaml:"filters" json:"filters,omitempty"`
}

// BGP neighbor types (decided by comparing ASNs).
const (
	BGPInternal = "ibgp"
	BGPExternal = "ebgp"
)

// BGPNeighbor is one neighbor. Type is optional: it is derived from the ASNs and, if authored, must agree.
type BGPNeighbor struct {
	ID             string `yaml:"id" json:"id"`
	IP             string `yaml:"ip" json:"ip"`
	ASN            int    `yaml:"asn" json:"asn"`
	Type           string `yaml:"type" json:"type,omitempty"`
	Segment        string `yaml:"segment" json:"segment,omitempty"`
	InboundFilter  string `yaml:"inbound_filter" json:"inbound_filter,omitempty"`
	OutboundFilter string `yaml:"outbound_filter" json:"outbound_filter,omitempty"`
	// NSD marks a neighbor that belongs to a Non SD-WAN destination (only used for the outbound-filter rule).
	NSD string `yaml:"nsd" json:"nsd,omitempty"`
}

// BGP default-route options.
const (
	DefaultNo          = "no"
	DefaultYes         = "yes"
	DefaultConditional = "conditional"
)

// BGPRedistribute is the BGP Advanced Settings block. Nil switches take the Assumptions value (assumption 22).
type BGPRedistribute struct {
	OverlayPrefix *bool  `yaml:"overlay_prefix" json:"overlay_prefix,omitempty"`
	Connected     *bool  `yaml:"connected" json:"connected,omitempty"`
	OSPF          *bool  `yaml:"ospf" json:"ospf,omitempty"`
	OSPFMetric    *int   `yaml:"ospf_metric" json:"ospf_metric,omitempty"`
	DefaultRoute  string `yaml:"default_route" json:"default_route,omitempty"`
}

// Filter actions.
const (
	FilterPermit = "permit"
	FilterDeny   = "deny"
)

// BGPFilter is one entry of the Filter List.
type BGPFilter struct {
	ID    string          `yaml:"id" json:"id"`
	Rules []BGPFilterRule `yaml:"rules" json:"rules"`
}

// BGPFilterRule is one Match/Set rule. An empty match matches every route.
type BGPFilterRule struct {
	ID     string         `yaml:"id" json:"id,omitempty"`
	Match  BGPFilterMatch `yaml:"match" json:"match"`
	Action string         `yaml:"action" json:"action"`
	Set    BGPFilterSet   `yaml:"set" json:"set"`
}

// BGPFilterMatch: prefix (exact by default), MED and local preference are the match types the model supports.
type BGPFilterMatch struct {
	Prefix          string `yaml:"prefix" json:"prefix,omitempty"`
	ExactMatch      *bool  `yaml:"exact_match" json:"exact_match,omitempty"`
	Metric          *int   `yaml:"metric" json:"metric,omitempty"`
	LocalPreference *int   `yaml:"local_preference" json:"local_preference,omitempty"`
}

// BGPFilterSet: the set actions the model supports (community and AS-path exclude are not modelled).
type BGPFilterSet struct {
	LocalPreference *int  `yaml:"local_preference" json:"local_preference,omitempty"`
	Metric          *int  `yaml:"metric" json:"metric,omitempty"`
	ASPathPrepend   []int `yaml:"as_path_prepend" json:"as_path_prepend,omitempty"`
}

// OverlayFlowControl is Configure > Overlay Flow Control: Global Advertise Flags, Preferred VPN Exits and pins.
type OverlayFlowControl struct {
	AdvertiseFlags *AdvertiseFlags  `yaml:"advertise_flags" json:"advertise_flags,omitempty"`
	PreferredExits []PreferredExits `yaml:"preferred_exits" json:"preferred_exits,omitempty"`
	Pins           []RoutePin       `yaml:"pins" json:"pins,omitempty"`
}

// AdvertiseFlags are the Global Advertise Flags the model supports. Nil means "not set": External OSPF and NSD
// iBGP default to off (documented), Static and Connected take the Assumptions value (assumption 69).
type AdvertiseFlags struct {
	Static       *bool `yaml:"static" json:"static,omitempty"`
	Connected    *bool `yaml:"connected" json:"connected,omitempty"`
	ExternalOSPF *bool `yaml:"external_ospf" json:"external_ospf,omitempty"`
}

// PreferredExits is the edited Preferred VPN Exits order of one subnet: exits are Edge ids, best first.
type PreferredExits struct {
	Segment string   `yaml:"segment" json:"segment,omitempty"`
	Prefix  string   `yaml:"prefix" json:"prefix"`
	Exits   []string `yaml:"exits" json:"exits"`
}

// RoutePin is "Pin Learned Route Preference" on one route: the exit's route gets a preference starting from 1.
type RoutePin struct {
	Segment string `yaml:"segment" json:"segment,omitempty"`
	Prefix  string `yaml:"prefix" json:"prefix"`
	Exit    string `yaml:"exit" json:"exit"`
}

// NSDPrefix is a static route of a Non SD-WAN destination (it appears in the Edge's static route table).
type NSDPrefix struct {
	Prefix string `yaml:"prefix" json:"prefix"`
	Cost   *int   `yaml:"cost" json:"cost,omitempty"`
	// Advertise is the Advertise box (not editable for a destination via Gateway).
	Advertise bool `yaml:"advertise" json:"advertise,omitempty"`
}
