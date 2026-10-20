// Package sim is the teaching-model simulator: a validated topology format and a pure effective-configuration
// engine. It is NOT an emulator. Every behaviour cites a VERIFIED or OBSERVED row of a fact sheet in
// content/reference/, or a numbered assumption in docs/SIMULATOR-MODEL.md (see rules.go and the
// "// rule: <id>" convention checked by rules_test.go). The package has no database, HTTP or clock dependency.
package sim

// SchemaVersion is the topology file version. The schema evolves additively: later steps add OPTIONAL fields
// (tunnels, routing, ...) so files written now stay valid under strict decoding.
const SchemaVersion = 1

// Topology is one teaching topology file (content/topologies/<id>.yaml).
type Topology struct {
	Version  int       `yaml:"version" json:"version"`
	Meta     Meta      `yaml:"meta" json:"meta"`
	Customer Customer  `yaml:"customer" json:"customer"`
	Segments []Segment `yaml:"segments" json:"segments"`
	Gateways []Gateway `yaml:"gateways" json:"gateways"`
	Links    []WanLink `yaml:"links" json:"links"`
	Profiles []Profile `yaml:"profiles" json:"profiles"`
	Sites    []Site    `yaml:"sites" json:"sites"`
	// Optional overlay keys (step 12).
	Clusters []Cluster `yaml:"clusters" json:"clusters,omitempty"`
	NSDs     []NSD     `yaml:"non_sdwan_destinations" json:"non_sdwan_destinations,omitempty"`
	// Optional routing keys (step 13).
	OverlayFlowControl *OverlayFlowControl `yaml:"overlay_flow_control" json:"overlay_flow_control,omitempty"`
}

// Meta labels the file. Teaching must be true: every topology here is an invented teaching network.
type Meta struct {
	ID       string `yaml:"id" json:"id"`
	Title    string `yaml:"title" json:"title"`
	Summary  string `yaml:"summary" json:"summary"`
	Teaching bool   `yaml:"teaching" json:"teaching"`
	Notes    string `yaml:"notes" json:"notes,omitempty"`
}

// Customer is the enterprise. Nil pointers mean "use the documented default".
type Customer struct {
	ID             string `yaml:"id" json:"id"`
	Name           string `yaml:"name" json:"name"`
	MaxSegments    *int   `yaml:"max_segments" json:"max_segments,omitempty"`
	PremiumService *bool  `yaml:"premium_service" json:"premium_service,omitempty"`
}

// Segment types.
const (
	SegRegular = "regular"
	SegPrivate = "private"
)

// Segment is a network segment. The Global Segment (id "global") always exists; listing it only sets its type.
type Segment struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	ServiceVLAN *int   `yaml:"service_vlan" json:"service_vlan,omitempty"`
}

// GlobalSegmentID is the id of the Global Segment.
const GlobalSegmentID = "global"

// Gateway is a Gateway in a region (used only for region sanity checks in this step).
type Gateway struct {
	ID     string   `yaml:"id" json:"id"`
	Region string   `yaml:"region" json:"region"`
	Roles  []string `yaml:"roles" json:"roles,omitempty"`
}

// WanLink is a WAN link: type public or private, medium wired or wireless, and bandwidth.
type WanLink struct {
	ID       string `yaml:"id" json:"id"`
	Type     string `yaml:"type" json:"type"`
	Medium   string `yaml:"medium" json:"medium"`
	UpMbps   int    `yaml:"up_mbps" json:"up_mbps"`
	DownMbps int    `yaml:"down_mbps" json:"down_mbps"`
	// MTU is the link MTU; nil means the documented path MTU discovery default (1500).
	MTU *int `yaml:"mtu" json:"mtu,omitempty"`
}

// Link types and media.
const (
	LinkPublic     = "public"
	LinkPrivate    = "private"
	MediumWired    = "wired"
	MediumWireless = "wireless"
)

// VLAN is a Profile or Edge VLAN. Name and ID of a Profile VLAN cannot be overridden on an Edge.
type VLAN struct {
	ID   int    `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`
}

// Profile is a configuration template: segments, device settings, VLANs, business rules and firewall rules.
type Profile struct {
	ID            string         `yaml:"id" json:"id"`
	Name          string         `yaml:"name" json:"name"`
	Description   string         `yaml:"description" json:"description,omitempty"`
	Segments      []string       `yaml:"segments" json:"segments"`
	CloudVPN      *CloudVPN      `yaml:"cloud_vpn" json:"cloud_vpn,omitempty"`
	Device        map[string]any `yaml:"device" json:"device,omitempty"`
	VLANs         []VLAN         `yaml:"vlans" json:"vlans,omitempty"`
	BusinessRules []BusinessRule `yaml:"business_rules" json:"business_rules,omitempty"`
	FirewallRules []FirewallRule `yaml:"firewall_rules" json:"firewall_rules,omitempty"`
	// Routing is the optional Routing & NAT area (step 13).
	Routing *ProfileRouting `yaml:"routing" json:"routing,omitempty"`
}

// Site groups Edges; Region drives Gateway assignment in a later step.
type Site struct {
	ID     string `yaml:"id" json:"id"`
	Name   string `yaml:"name" json:"name"`
	Region string `yaml:"region" json:"region"`
	Edges  []Edge `yaml:"edges" json:"edges"`
}

// Edge roles.
const (
	RoleBranch = "branch"
	RoleHub    = "hub"
)

// Edge is one Edge: exactly one Profile, interfaces, its own VLANs, and its overrides.
type Edge struct {
	ID         string        `yaml:"id" json:"id"`
	Name       string        `yaml:"name" json:"name"`
	Model      string        `yaml:"model" json:"model,omitempty"`
	Role       string        `yaml:"role" json:"role"`
	Profile    string        `yaml:"profile" json:"profile"`
	Interfaces []Interface   `yaml:"interfaces" json:"interfaces"`
	VLANs      []VLAN        `yaml:"vlans" json:"vlans,omitempty"`
	Overrides  EdgeOverrides `yaml:"overrides" json:"overrides"`
	// Optional overlay attributes (step 12).
	LogicalID             *int               `yaml:"logical_id" json:"logical_id,omitempty"`
	Gateways              *GatewayAssignment `yaml:"gateways" json:"gateways,omitempty"`
	NATBlocksIncomingVCMP bool               `yaml:"nat_blocks_incoming_vcmp" json:"nat_blocks_incoming_vcmp,omitempty"`
	// Routing is the optional routing input of the Edge (step 13).
	Routing *EdgeRouting `yaml:"routing" json:"routing,omitempty"`
}

// Interface kinds.
const (
	IfWan      = "wan"
	IfLan      = "lan"
	IfRouted   = "routed"
	IfLoopback = "loopback"
)

// Interface is an Edge interface. Segments empty means "all segments of the Profile" (documented default).
// Override is the interface-configuration override tick (it matters on a Profile switch).
type Interface struct {
	ID       string   `yaml:"id" json:"id"`
	Kind     string   `yaml:"kind" json:"kind"`
	VLAN     *int     `yaml:"vlan" json:"vlan,omitempty"`
	Segments []string `yaml:"segments" json:"segments,omitempty"`
	Link     string   `yaml:"link" json:"link,omitempty"`
	Address  string   `yaml:"address" json:"address,omitempty"`
	Override bool     `yaml:"override" json:"override,omitempty"`
}

// EdgeOverrides is everything an Edge changes relative to its Profile.
type EdgeOverrides struct {
	Device        map[string]any   `yaml:"device" json:"device,omitempty"`
	VLANs         []VLANOverride   `yaml:"vlans" json:"vlans,omitempty"`
	BusinessRules []BusinessRule   `yaml:"business_rules" json:"business_rules,omitempty"`
	Firewall      FirewallOverride `yaml:"firewall" json:"firewall"`
	CloudVPN      *CloudVPN        `yaml:"cloud_vpn" json:"cloud_vpn,omitempty"`
}

// VLANOverride overrides settings of an inherited Profile VLAN. Settings may not contain id or name.
type VLANOverride struct {
	ID       int            `yaml:"id" json:"id"`
	Settings map[string]any `yaml:"settings" json:"settings"`
}

// FirewallOverride is the Edge Firewall tab: the Override tick and the Edge's own rules.
type FirewallOverride struct {
	Override bool           `yaml:"override" json:"override"`
	Rules    []FirewallRule `yaml:"rules" json:"rules,omitempty"`
}

// Business rule enums.
const (
	NetDirect      = "direct"
	NetMultipath   = "multipath"
	NetBackhaul    = "internet_backhaul"
	SteerAuto      = "auto"
	SteerGroup     = "transport_group"
	SteerInterface = "interface"
	SteerWanLink   = "wan_link"
	DestInternet   = "internet"
)

// RuleMatch is the match of a business rule. All fields are strings so two matches compare with ==.
type RuleMatch struct {
	Application       string `yaml:"application" json:"application,omitempty"`
	CustomApplication bool   `yaml:"custom_application" json:"custom_application,omitempty"`
	Source            string `yaml:"source" json:"source,omitempty"`
	Destination       string `yaml:"destination" json:"destination,omitempty"`
	Protocol          string `yaml:"protocol" json:"protocol,omitempty"`
	Port              string `yaml:"port" json:"port,omitempty"`
	DSCP              string `yaml:"dscp" json:"dscp,omitempty"`
}

// Steering is the link-steering action of a rule.
type Steering struct {
	Mode       string `yaml:"mode" json:"mode"`
	Group      string `yaml:"group" json:"group,omitempty"`
	Interface  string `yaml:"interface" json:"interface,omitempty"`
	Link       string `yaml:"link" json:"link,omitempty"`
	Strictness string `yaml:"strictness" json:"strictness,omitempty"`
}

// RuleAction is the action of a business rule.
type RuleAction struct {
	Priority       string   `yaml:"priority" json:"priority,omitempty"`
	NetworkService string   `yaml:"network_service" json:"network_service,omitempty"`
	Steering       Steering `yaml:"steering" json:"steering"`
	ServiceClass   string   `yaml:"service_class" json:"service_class,omitempty"`
	RateLimit      bool     `yaml:"rate_limit" json:"rate_limit,omitempty"`
	NAT            bool     `yaml:"nat" json:"nat,omitempty"`
}

// BusinessRule is one business policy rule, per segment.
type BusinessRule struct {
	ID      string     `yaml:"id" json:"id"`
	Name    string     `yaml:"name" json:"name,omitempty"`
	Segment string     `yaml:"segment" json:"segment"`
	Match   RuleMatch  `yaml:"match" json:"match"`
	Action  RuleAction `yaml:"action" json:"action"`
}

// Firewall actions.
const (
	FwAllow  = "allow"
	FwDrop   = "drop"
	FwReject = "reject"
	FwSkip   = "skip"
)

// FirewallMatch is the match of a firewall rule (all strings, comparable).
type FirewallMatch struct {
	SourceIP        string `yaml:"source_ip" json:"source_ip,omitempty"`
	DestinationIP   string `yaml:"destination_ip" json:"destination_ip,omitempty"`
	Protocol        string `yaml:"protocol" json:"protocol,omitempty"`
	DestinationPort string `yaml:"destination_port" json:"destination_port,omitempty"`
	VLAN            string `yaml:"vlan" json:"vlan,omitempty"`
	Application     string `yaml:"application" json:"application,omitempty"`
}

// FirewallRule is one firewall rule.
type FirewallRule struct {
	ID     string        `yaml:"id" json:"id"`
	Name   string        `yaml:"name" json:"name,omitempty"`
	Match  FirewallMatch `yaml:"match" json:"match"`
	Action string        `yaml:"action" json:"action"`
}
