package sim

// Rules added in step 12 (tunnel formation, dynamic lifecycle, hub failover, path selection, steering, MTU).
// Same convention as rules.go: code cites "// rule: <id>"; rules_test.go checks the evidence.
const (
	RGatewayTunnels   = "gateway-tunnels"
	RGatewayAssign    = "gateway-assignment"
	RGatewayAddr      = "gateway-address-by-link"
	RGatewayRoles     = "gateway-roles"
	RCloudVPNSegment  = "cloud-vpn-per-segment"
	RCloudVPNTypes    = "cloud-vpn-tunnel-types"
	RB2HNeedsCloudVPN = "b2h-needs-cloud-vpn"
	RB2HPermanent     = "b2h-permanent"
	RHubList          = "hub-designation"
	RHubOrder         = "hub-order-tiebreak"
	RHubFailover      = "hub-failover"
	RClusterOneHub    = "cluster-single-hub"
	RClusterAssign    = "cluster-assignment"
	RClusterModel     = "cluster-same-model"
	RB2BOptions       = "b2b-options"
	RB2BHubs64        = "b2b-hubs-6-4"
	RB2BGateway       = "b2b-via-gateway"
	RDynFirstPacket   = "dynamic-first-packet"
	RDynDefault       = "dynamic-default-on"
	RDynTeardown      = "dynamic-teardown-180"
	RDynNAT           = "dynamic-nat"
	RCloudVPNOverride = "cloud-vpn-override"
	RTunnelCap        = "tunnel-cap"
	RNSDGatewayProf   = "nsd-gateway-profile-level"
	RNSDCounts        = "nsd-tunnel-counts"
	RNSDEdgeSegment   = "nsd-edge-per-segment"
	RNSDGatewayTypes  = "nsd-gateway-types"
	RNSDEdgeTypes     = "nsd-edge-types"
	RNSDGatewayEnable = "nsd-gateway-enable"
	RPathsMode        = "paths-mode"
	RPathsLocalLinks  = "paths-local-links"
	RSteerModes       = "steering-modes"
	RSteerStrictness  = "steering-strictness"
	RSteerAutoBest    = "steering-auto-best"
	RSteerMandatory   = "steering-mandatory"
	RSteerPreferred   = "steering-preferred"
	RSteerAvailable   = "steering-available"
	RNetworkService   = "network-service-decides"
	RMTUVPN           = "mtu-vpn-overhead"
	RMTUDerived       = "mtu-derived"
	RMTUDefault       = "mtu-default-1500"
	RMTUNoJumbo       = "mtu-no-jumbo"
	RMTUFloor         = "mtu-1300-floor"
	RLinkStates       = "link-outage-brownout"
	RPeerUnreachable  = "peer-unreachable"

	RNMOrchestratorDown  = "nm-orchestrator-down"
	RNMDirectVsHub       = "nm-direct-vs-hub"
	RNMInflight          = "nm-inflight-teardown"
	RNMSecureTunnel      = "nm-secure-tunnel-decision"
	RNMCloudVPNDefault   = "nm-cloud-vpn-default"
	RNMCloudVPNEnable    = "nm-cloud-vpn-override-enable"
	RNMCloudVPNGranular  = "nm-cloud-vpn-override-granularity"
	RNMHubReturnLocal    = "nm-hub-return-local"
	RNMHubNAT            = "nm-hub-nat"
	RNMNSDMTU            = "nm-nsd-mtu"
	RNMPreferredPrivate  = "nm-preferred-private"
	RNMSLAThresholds     = "nm-sla-thresholds"
	RNMB2BOneSite        = "nm-b2b-one-site"
	RNMNSDGatewayChoice  = "nm-nsd-gateway-choice"
	RNMLoadBalancing     = "nm-load-balancing"
	RNMRoutingPrecedence = "nm-routing-precedence"

	RA42CloudVPNOff       = "assume-42"
	RA43OverrideWhole     = "assume-43"
	RA44GatewayDerive     = "assume-44"
	RA45AlwaysGateway     = "assume-45"
	RA46PathPairs         = "assume-46"
	RA47GatewayAddr       = "assume-47"
	RA48ClusterOrder      = "assume-48"
	RA49HotStandby        = "assume-49"
	RA50NSDBuilder        = "assume-50"
	RA51NoBuildDelay      = "assume-51"
	RA52Boundary          = "assume-52"
	RA53Restart           = "assume-53"
	RA54BothDynamic       = "assume-54"
	RA55DegradedState     = "assume-55"
	RA56AutoDeterministic = "assume-56"
	RA57GroupStrictness   = "assume-57"
	RA58AvailableMoves    = "assume-58"
	RA59HubAttrsEqual     = "assume-59"
	RA60HubTunnelDirect   = "assume-60"
	RA61HubsOnly          = "assume-61"
	RA62FirstLeg          = "assume-62"
	RA63DirectNotOverlay  = "assume-63"
	RA64LinkMTU           = "assume-64"
	RA65BuildSwitch       = "assume-65"
	RA66VPNOverhead       = "assume-66"
	RA67HoldNotApplied    = "assume-67"
	RA68NoRemediation     = "assume-68"
	RSteerHold            = "steering-hold-time"
	RA14DirectFirst       = "assume-14"
	RA15NSDByPolicy       = "assume-15"
	RA16BuildNAT          = "assume-16"
	RA13PrivateNeedsHub   = "assume-13"
	RA9DynamicAfterFlow   = "assume-9"
	RA1FirstMatch         = "assume-1"
	RA2NoRuleDefault      = "assume-2"
	RA3EdgeAbove          = "assume-3"
	RA10TransportGroup    = "assume-10"
)

var overlayRules = []Rule{
	{ID: RGatewayTunnels, Status: StatusVerified, Sheet: "tunnel-types", Claim: "Each Edge creates DMPO tunnels with one or more Gateways", Text: "Each Edge creates DMPO tunnels with Gateways."},
	{ID: RGatewayAssign, Status: StatusVerified, Sheet: "gateway-details", Claim: "assigns each Edge a primary and a secondary Gateway by geolocation", Text: "Each Edge has a primary and a secondary Gateway, assigned by geolocation."},
	{ID: RGatewayAddr, Status: StatusVerified, Sheet: "gateway-details", Claim: "Internet-only Edges tunnel to the Gateway's public address", Text: "Internet-only Edges tunnel to the Gateway's public address and MPLS-only Edges to its private address."},
	{ID: RGatewayRoles, Status: StatusVerified, Sheet: "gateway-details", Claim: "A Gateway with the data-plane role forwards", Text: "Gateways have roles: data plane, control plane, secure VPN."},
	{ID: RCloudVPNSegment, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "Users must configure Cloud VPN per Segment", Text: "Cloud VPN is configured per segment."},
	{ID: RCloudVPNTypes, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "Enabling Cloud VPN lets the Edge initiate and respond", Text: "Cloud VPN offers Branch to Hub, Branch to Branch and Edge to Non SD-WAN via Gateway."},
	{ID: RB2HNeedsCloudVPN, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "In 6.4 the branch-to-hub tunnel comes up only after Cloud VPN is enabled", Text: "The Branch to Hub tunnel exists only after Cloud VPN is enabled."},
	{ID: RB2HPermanent, Status: StatusVerified, Sheet: "tunnel-types", Claim: "Branch to Hub VPN is permanent", Text: "Branch to Hub tunnels are permanent: every Edge builds one to each Hub."},
	{ID: RHubList, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "A Hub is an Edge designated in the Profile", Text: "A Hub is an Edge or cluster designated in the Profile's Hub list."},
	{ID: RHubOrder, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "Hub order is a tie-breaker", Text: "Hub order is the tie-breaker after routing attributes."},
	{ID: RHubFailover, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "If the primary Hub goes down, the Spoke updates the local flow's route", Text: "If the primary Hub goes down a Spoke moves local flows to the secondary Hub; peer flows are not moved back when it returns."},
	{ID: RClusterOneHub, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "Edge Clustering addresses the Hub scale issue", Text: "Other Edges treat a Hub cluster as a single Hub."},
	{ID: RClusterAssign, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "The Gateway assigns an Edge to a cluster member", Text: "An Edge is assigned to a cluster member by its logical ID modulo the number of Hubs."},
	{ID: RClusterModel, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "members should be the same model and capacity", Text: "Cluster members should be the same model."},
	{ID: RB2BOptions, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "Branch to Branch is set under", Text: "Branch to Branch VPN goes via Cloud Gateways or via Hubs for VPN."},
	{ID: RB2BHubs64, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "6.4: enabling Hubs for VPN deactivates the data plane", Text: "In 6.4, Hubs for VPN sends branch-to-branch traffic through the Hubs in order and not through the Gateway."},
	{ID: RB2BGateway, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "Gateway involvement: with Cloud Gateways, tunnels run from each Edge", Text: "With Cloud Gateways, Edge to Edge traffic goes through the Gateway."},
	{ID: RDynFirstPacket, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "Dynamic Branch to Branch: on activation", Text: "The first packet goes via the Gateway or Hub; the initiating Edge then builds a direct tunnel."},
	{ID: RDynDefault, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "By default, the Orchestrator configures it for all Edges", Text: "Dynamic Branch to Branch is on for all Edges by default."},
	{ID: RDynTeardown, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "after 180 seconds of traffic silence", Text: "The direct tunnel is torn down after 180 seconds of traffic silence."},
	{ID: RDynNAT, Status: StatusVerified, Sheet: "tunnel-types", Claim: "If branch Edges sit behind unconfigured NAT devices", Text: "Behind unconfigured NAT the direct tunnel will likely not come up."},
	{ID: RCloudVPNOverride, Status: StatusVerified, Sheet: "cloud-vpn", Claim: "An Edge's Device tab inherits the Cloud VPN settings", Text: "An Edge inherits Cloud VPN settings from its Profile and can override them."},
	{ID: RTunnelCap, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "The maximum number of tunnels an Edge can establish", Text: "At most 16 tunnels between an Edge and another Edge or Gateway."},
	{ID: RNSDGatewayProf, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "NSD via Gateway is defined exclusively at the Profile level", Text: "NSD via Gateway is Profile-level only."},
	{ID: RNSDCounts, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "Active/Active allows up to 4 tunnels", Text: "Tunnel caps: Active/Active 4, Hot-Standby 2, Cisco ASA and Generic Firewall 1, Azure Virtual Hub 2."},
	{ID: RNSDEdgeSegment, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "NSD via Edge is added under Configure > Network Services", Text: "NSD via Edge: one service per segment, enabled in the Profile."},
	{ID: RNSDGatewayTypes, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "NSD via Gateway supported types", Text: "The documented NSD via Gateway types."},
	{ID: RNSDEdgeTypes, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "NSD via Edge: an Edge builds an IPsec tunnel", Text: "The documented NSD via Edge types."},
	{ID: RNSDGatewayEnable, Status: StatusVerified, Sheet: "non-sdwan-destinations", Claim: "For NSD via Gateway the Profile Device tab has Cloud VPN on", Text: "NSD via Gateway needs Cloud VPN on and Edge to Non SD-WAN via Gateway enabled."},
	{ID: RPathsMode, Status: StatusVerified, Sheet: "hubs-and-branches", Claim: "List Paths shows each tunnel's mode", Text: "List Paths shows each tunnel's mode: Static (always up) or Dynamic (on demand)."},
	{ID: RPathsLocalLinks, Status: StatusVerified, Sheet: "tunnel-types", Claim: "The List Paths remote diagnostic checks the active tunnel paths between local WAN links and each peer", Text: "A path is a local WAN link to a peer."},
	{ID: RSteerModes, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "A business-policy rule has a Link Steering choice with four modes", Text: "Link steering modes: Auto, Transport Group, Interface, WAN Link."},
	{ID: RSteerStrictness, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "For Interface and WAN Link steering you also pick Mandatory", Text: "Mandatory, Preferred or Available."},
	{ID: RSteerAutoBest, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "Auto: DMPO picks the best links", Text: "Auto: DMPO picks the best links for the application."},
	{ID: RSteerMandatory, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "Mandatory, business-policy Table 6", Text: "Mandatory: packets are dropped if the chosen link or group is inactive; it never steers away."},
	{ID: RSteerPreferred, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "Preferred: stay on the chosen link while it meets the application SLA", Text: "Preferred: stay while the SLA holds, then move to another public link; no fallback to a private link on an SLA failure."},
	{ID: RSteerAvailable, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "Available: use the chosen link while it is up", Text: "Available: use the chosen link while it is up, whatever the SLA; flows move only if it goes down."},
	{ID: RNetworkService, Status: StatusVerified, Sheet: "overlay-paths", Claim: "Business policy network service (Direct, Multi-Path, Internet Backhaul) decides", Text: "The business policy network service decides how traffic is carried."},
	{ID: RMTUVPN, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "VPN (IPsec) traffic overhead totals 97 to 112 bytes", Text: "VPN (IPsec) traffic overhead is 97 to 112 bytes."},
	{ID: RMTUDerived, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "Derived arithmetic (not printed in the guide)", Text: "On a 1500-byte path VPN traffic leaves 1388 to 1403 bytes."},
	{ID: RMTUDefault, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "Path MTU discovery starts from a default link MTU of 1500 bytes", Text: "Path MTU discovery starts at 1500; 1500 is the maximum."},
	{ID: RMTUNoJumbo, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "VeloCloud SD-WAN does not support jumbo frames", Text: "No jumbo frames."},
	{ID: RMTUFloor, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "RFC 4821 probing measures MTU only down to 1300 bytes", Text: "Below 1300 the MTU must be configured manually."},
	{ID: RLinkStates, Status: StatusVerified, Sheet: "dmpo-monitoring", Claim: "A brownout or outage is detected within 300 to 500 ms", Text: "DMPO distinguishes an outage from a brownout."},
	{ID: RPeerUnreachable, Status: StatusVerified, Sheet: "overlay-paths", Claim: "The Edge route table's reason codes include PR_UNREACHABLE", Text: "An unreachable remote Gateway or Edge peer makes overlay paths unreachable."},

	{ID: RNMOrchestratorDown, Status: StatusNotModelled, Sheet: "hubs-and-branches", Claim: "What an Edge does when the Orchestrator is unreachable", Text: "not modelled: what an Edge does when the Orchestrator is unreachable (existing tunnels, forwarding, building new tunnels) is not stated."},
	{ID: RNMDirectVsHub, Status: StatusNotModelled, Sheet: "overlay-paths", Claim: "Whether a flow prefers a direct dynamic tunnel over the Hub or Gateway path", Text: "not modelled: whether a flow prefers an existing direct tunnel over the Hub or Gateway path, and how a flow is moved, is not stated."},
	{ID: RNMInflight, Status: StatusNotModelled, Sheet: "overlay-paths", Claim: "What happens to in-flight flows when a direct tunnel is torn down", Text: "not modelled: what happens to in-flight flows when a direct tunnel is torn down is not stated."},
	{ID: RNMSecureTunnel, Status: StatusNotModelled, Sheet: "hubs-and-branches", Claim: "Further conditions for the \"secure overlay multi-path tunnel\" decision were not found", Text: "not modelled: the exact conditions under which the initiating Edge decides a secure tunnel is possible are not stated. The simulator uses a visible, configurable stub."},
	{ID: RNMCloudVPNDefault, Status: StatusNotModelled, Sheet: "cloud-vpn", Claim: "The Cloud VPN default state", Text: "not modelled: the default state of Cloud VPN is not stated."},
	{ID: RNMCloudVPNEnable, Status: StatusNotModelled, Sheet: "cloud-vpn", Claim: "Whether per-Edge Cloud VPN overrides can enable a tunnel type", Text: "not modelled: whether an Edge override can enable a tunnel type its Profile disables is not stated."},
	{ID: RNMCloudVPNGranular, Status: StatusNotModelled, Sheet: "cloud-vpn", Claim: "Whether an Edge's Cloud VPN override replaces a whole sub-section", Text: "not modelled: whether an Edge's Cloud VPN override replaces a whole sub-section or single fields is not stated."},
	{ID: RNMHubReturnLocal, Status: StatusNotModelled, Sheet: "hubs-and-branches", Claim: "Whether a Spoke moves its local flows back to the primary Hub", Text: "not modelled: whether a Spoke moves local flows back to the primary Hub when it returns, and how long it takes to treat a Hub as down, are not stated."},
	{ID: RNMHubNAT, Status: StatusNotModelled, Sheet: "hubs-and-branches", Claim: "Whether a Hub Edge behind a NAT that blocks incoming UDP 2426", Text: "not modelled: whether a Hub Edge behind a NAT that blocks incoming UDP 2426 accepts tunnels is not stated."},
	{ID: RNMNSDMTU, Status: StatusNotModelled, Sheet: "mtu-overhead", Claim: "The IPsec overhead of tunnels to Non SD-WAN Destinations", Text: "not modelled: the effective MTU of tunnels to Non SD-WAN destinations is not stated."},
	{ID: RNMPreferredPrivate, Status: StatusNotModelled, Sheet: "dmpo-steering", Claim: "What Preferred steering does when the chosen link is a private link", Text: "not modelled: what Preferred steering does when the chosen link is a private link that is down or misses its SLA is not stated."},
	{ID: RNMSLAThresholds, Status: StatusNotModelled, Sheet: "dmpo-steering", Claim: "How many measured loss, latency or jitter values make a link degraded", Text: "not modelled: the loss, latency and jitter thresholds that make a link degraded are not available; degraded is only a state the reader sets."},
	{ID: RNMB2BOneSite, Status: StatusNotModelled, Sheet: "tunnel-types", Claim: "The full behaviour of Branch to Branch when only one of two sites has it enabled", Text: "not modelled: Branch to Branch behaviour when only one of two sites has it enabled."},
	{ID: RNMNSDGatewayChoice, Status: StatusNotModelled, Sheet: "non-sdwan-destinations", Claim: "Whether the Gateway tunnels from the nearest or a chosen Gateway", Text: "not modelled: which Gateway builds a Non SD-WAN tunnel is not stated; the topology names it."},
	{ID: RNMLoadBalancing, Status: StatusNotModelled, Sheet: "dmpo-load-balancing", Claim: "Whether current releases still load-balance Direct traffic per flow", Text: "not modelled: per-flow or per-packet load balancing across links."},
	{ID: RNMRoutingPrecedence, Status: StatusNotModelled, Sheet: "overlay-paths", Claim: "The documented precedence between route types", Text: "not modelled in a path decision: routes and prefixes. Reachability between sites is assumed here; the routing screens compute it."},

	{ID: RA42CloudVPNOff, Status: StatusAssumption, Assumption: 42, Text: "Cloud VPN is off unless the Profile lists segments under cloud_vpn (its default state is not documented)."},
	{ID: RA43OverrideWhole, Status: StatusAssumption, Assumption: 43, Text: "An Edge's cloud_vpn override replaces whole sub-sections (segments, branch_to_hub, branch_to_branch, NSD lists), not single fields."},
	{ID: RA44GatewayDerive, Status: StatusAssumption, Assumption: 44, Text: "Without an explicit assignment, the primary Gateway is the first Gateway in the site's region, the secondary the next one in that region, else the first Gateway of another region."},
	{ID: RA45AlwaysGateway, Status: StatusAssumption, Assumption: 45, Text: "Every Edge builds tunnels to both its primary and secondary Gateway, whatever Cloud VPN says."},
	{ID: RA46PathPairs, Status: StatusAssumption, Assumption: 46, Text: "A tunnel's paths are the pairs of same-type WAN links (public with public, private with private); a path is as healthy as the worse of its two links."},
	{ID: RA47GatewayAddr, Status: StatusAssumption, Assumption: 47, Text: "A public WAN link reaches a Gateway's public address and a private link its private address."},
	{ID: RA48ClusterOrder, Status: StatusAssumption, Assumption: 48, Text: "Cluster members keep their authored order; an Edge's logical ID is authored (default: its position in the file)."},
	{ID: RA49HotStandby, Status: StatusAssumption, Assumption: 49, Text: "In Active/Hot-Standby the first tunnel is active and the others are established but standby."},
	{ID: RA50NSDBuilder, Status: StatusAssumption, Assumption: 50, Text: "A via-Gateway destination is built by the Gateway the topology names; a via-Edge destination by one tunnel on the Edge's first public WAN link."},
	{ID: RA51NoBuildDelay, Status: StatusAssumption, Assumption: 51, Text: "The initiating Edge builds the direct tunnel at the moment of the first packet if the build decision allows it (no build delay)."},
	{ID: RA52Boundary, Status: StatusAssumption, Assumption: 52, Text: "Exactly 180 seconds of silence tears the tunnel down; 179 seconds keeps it."},
	{ID: RA53Restart, Status: StatusAssumption, Assumption: 53, Text: "After a teardown, new traffic starts the cycle again: first packets via the standing path, then a rebuild."},
	{ID: RA54BothDynamic, Status: StatusAssumption, Assumption: 54, Text: "A dynamic tunnel needs Branch to Branch VPN enabled and Dynamic Branch to Branch on for both Edges."},
	{ID: RA55DegradedState, Status: StatusAssumption, Assumption: 55, Text: "A degraded link is up but misses its SLA (a brownout); a down link is an outage. Which numbers cause either is not modelled."},
	{ID: RA56AutoDeterministic, Status: StatusAssumption, Assumption: 56, Text: "When several paths are equally healthy, Auto picks the first in a fixed order; DMPO's measurement-based choice is not modelled."},
	{ID: RA57GroupStrictness, Status: StatusAssumption, Assumption: 57, Text: "Transport Group steering follows the same Mandatory, Preferred and Available rules as Interface and WAN Link steering."},
	{ID: RA58AvailableMoves, Status: StatusAssumption, Assumption: 58, Text: "With Available, when the chosen link is down the flow moves to the path Auto would choose."},
	{ID: RA59HubAttrsEqual, Status: StatusAssumption, Assumption: 59, Text: "Routing attributes of Hub routes are assumed equal, so Hub order decides between Hubs."},
	{ID: RA60HubTunnelDirect, Status: StatusAssumption, Assumption: 60, Text: "For a flow whose destination is the Hub Edge itself, the static Hub tunnel is preferred over the Gateway path."},
	{ID: RA61HubsOnly, Status: StatusAssumption, Assumption: 61, Text: "With Hubs for VPN the standing branch-to-branch path uses only Hubs (the 6.4 behaviour) in 7.0 too; with Cloud Gateways only the Gateway."},
	{ID: RA62FirstLeg, Status: StatusAssumption, Assumption: 62, Text: "Link steering applies to the source Edge's first leg; the other legs are chosen as Auto."},
	{ID: RA63DirectNotOverlay, Status: StatusAssumption, Assumption: 63, Text: "A matched rule with network service Direct sends the flow outside the overlay, so no overlay path is chosen."},
	{ID: RA64LinkMTU, Status: StatusAssumption, Assumption: 64, Text: "A link's MTU is its authored value (default 1500); the path MTU is the smaller of the two links of the first leg."},
	{ID: RA65BuildSwitch, Status: StatusAssumption, Assumption: 65, Text: "The build decision can be forced to always or never by the reader (a teaching switch)."},
	{ID: RSteerHold, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "A flow may be moved onto another tunnel mid-session; after an outage or brownout clears there is a one-minute hold", Text: "After an outage or brownout clears there is a one-minute hold before steering back to a preferred link."},
	{ID: RA66VPNOverhead, Status: StatusAssumption, Assumption: 66, Text: "Flows between two Edges over Cloud VPN are VPN (IPsec) traffic for the MTU calculation."},
	{ID: RA67HoldNotApplied, Status: StatusAssumption, Assumption: 67, Text: "A decision reflects the current link states, as if the one-minute hold before moving back to a preferred link had already elapsed."},
	{ID: RA68NoRemediation, Status: StatusAssumption, Assumption: 68, Text: "On-demand remediation (FEC, jitter buffer, NACK) is not modelled: with degraded links the decision only names the link the flow would use."},
	{ID: RA14DirectFirst, Status: StatusAssumption, Assumption: 14, Text: "A flow uses an existing direct dynamic tunnel in preference to the standing Gateway or Hub path."},
	{ID: RA15NSDByPolicy, Status: StatusAssumption, Assumption: 15, Text: "When a flow could use an NSD tunnel or an overlay path, business policy decides; no further precedence is modelled."},
	{ID: RA16BuildNAT, Status: StatusAssumption, Assumption: 16, Text: "The decision 'a secure overlay tunnel is possible' is true unless a NAT blocks incoming UDP 2426 to a peer."},
	{ID: RA13PrivateNeedsHub, Status: StatusAssumption, Assumption: 13, Text: "A private link is only useful with a private-capable peer such as a Hub."},
	{ID: RA9DynamicAfterFlow, Status: StatusAssumption, Assumption: 9, Text: "Dynamic tunnels appear after the first flow between two branches."},
	{ID: RA1FirstMatch, Status: StatusAssumption, Assumption: 1, Text: "Business rules are evaluated top-down, first match wins."},
	{ID: RA2NoRuleDefault, Status: StatusAssumption, Assumption: 2, Text: "When no business rule matches, Smart Defaults applies; the simulator treats it as Multi-Path."},
	{ID: RA3EdgeAbove, Status: StatusAssumption, Assumption: 3, Text: "An Edge rule sits above Profile rules in precedence when both match."},
	{ID: RA10TransportGroup, Status: StatusAssumption, Assumption: 10, Text: "A link's transport group is derived from its type and medium."},
}

// Rules is the full registry: step 11 rules plus the overlay rules.
var Rules = append(append(append(append([]Rule{}, baseRules...), overlayRules...), uiRules...), routingRules...)
