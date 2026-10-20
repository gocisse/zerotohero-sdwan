package sim

// Rules added in step 13 (route sources, selection, advertisement, redistribution, filters, Overlay Flow Control,
// reachability). Same convention as rules.go: code cites "// rule: <id>"; rules_test.go checks the evidence.
const (
	RRtLocalRemote     = "rt-local-remote"
	RRtOrderOfOps      = "rt-order-of-operations"
	RRtLongestPrefix   = "rt-longest-prefix"
	RRtLowerWins       = "rt-lower-preference-wins"
	RRtDefaultPrefs    = "rt-default-preferences"
	RRtTieBreakers     = "rt-tie-breakers"
	RRtOSPFvsBGP       = "rt-ospf-beats-bgp"
	RRtBGPvsExternal   = "rt-bgp-beats-external-ospf"
	RRtOSPFTypeOrder   = "rt-ospf-type-order"
	RRtBGPAttrs        = "rt-bgp-attributes"
	RRtConnFlag        = "rt-connected-flag"
	RRtLastResort      = "rt-static-last-resort"
	RRtSwanVsVelo      = "rt-static-wan-vs-overlay"
	RRtRemoteOrder     = "rt-remote-order"
	RRtOneStatic       = "rt-one-static-advertised"
	RRtStaticAdvertise = "rt-static-advertise"
	RRtStaticCloudVPN  = "rt-static-needs-cloud-vpn"
	RRtProbe           = "rt-static-icmp-probe"
	RRtSendAll         = "rt-edge-sends-all"
	RRtReflect         = "rt-gateway-reflects"
	RRtOSPFAuto        = "rt-ospf-to-overlay-auto"
	RRtExternalHeld    = "rt-external-ospf-held-out"
	RRtBGPAdvertise    = "rt-bgp-advertise-reachable"
	RRtOverlapHeld     = "rt-overlap-not-in-ofc"
	RRtOverlapBGP      = "rt-overlap-not-in-bgp"
	RRtBestOnly        = "rt-redistribute-best-only"
	RRtOSPFExtDefaults = "rt-ospf-external-defaults"
	RRtOverlaySwitches = "rt-overlay-redistribution-switches"
	RRtOSPFPassive     = "rt-ospf-switched-passive"
	RRtOSPFRouted      = "rt-ospf-full-needs-routed"
	RRtOSPFProfile     = "rt-ospf-profile-first"
	RRtOSPFGlobal      = "rt-ospf-global-segment"
	RRtOSPFSummary     = "rt-ospf-summary-external"
	RRtOSPFDefault     = "rt-ospf-default-route"
	RRtBGPDefault      = "rt-bgp-default-route"
	RRtBGPType         = "rt-bgp-ibgp-ebgp"
	RRtBGPSwitches     = "rt-bgp-redistribute-switches"
	RRtBGPFromOSPF     = "rt-ospf-into-bgp"
	RRtFilterActions   = "rt-filter-set-actions"
	RRtFilterMatch     = "rt-filter-match"
	RRtNSDiBGPLP       = "rt-nsd-ibgp-no-lp"
	RRtOFCExits        = "rt-ofc-preferred-exits"
	RRtPin             = "rt-ofc-pin"
	RRtGlobalFlags     = "rt-global-advertise-flags"
	RRtUnreachable     = "rt-unreachable-code"
	RRtLostNext        = "rt-lost-reason-next"
	RRtSegmentScoped   = "rt-segment-scoped"
	RRtConnectedSubnet = "rt-connected-from-interfaces"
	RRtLogicalID       = "rt-logical-id-tiebreak"

	RNMRtCrossSegment = "nm-rt-cross-segment"
	RNMRtOEPref       = "nm-rt-oe-preference"
	RNMRtOSPFConnStat = "nm-rt-connected-static-into-ospf"
	RNMRtRoutedVsVelo = "nm-rt-routed-vs-overlay"
	RNMRtTransit      = "nm-rt-transit-metric"
	RNMRtTieAge       = "nm-rt-logical-id-and-age"
	RNMRtOSPFFilters  = "nm-rt-ospf-filters"
	RNMRtOFCRest      = "nm-rt-ofc-rest"
	RNMRtLoops        = "nm-rt-redistribution-loops"
	RNMRtPassive      = "nm-rt-passive-ospf"
	RNMRtASPath       = "nm-rt-overlay-attributes"

	RA69AdvertiseDefaults = "assume-69"
	RA70LastResort        = "assume-70"
	RA71StaticCost        = "assume-71"
	RA72ProbeCode         = "assume-72"
	RA73PassiveNoAdj      = "assume-73"
	RA74LogicalIDDir      = "assume-74"
	RA75DefaultLP         = "assume-75"
	RA76NSDSlots          = "assume-76"
	RA77OSPFDefaultAlways = "assume-77"
	RA78BGPPerEdge        = "assume-78"
	RA79Overlap           = "assume-79"
	RA80SummaryMetric     = "assume-80"
	RA81RemoteAttrs       = "assume-81"
	RA82OriginCloudVPN    = "assume-82"
	RA83StageOrder        = "assume-83"
	RA84DestinationMatch  = "assume-84"
	RA22Redistribution    = "assume-22"
	RA23FilterDefault     = "assume-23"
	RA24BGPOrder          = "assume-24"
	RA28StaticSegment     = "assume-28"
	RA29SummaryOnly       = "assume-29"
	RA32CloudVPNReflect   = "assume-32"
	RA30NeighborDown      = "assume-30"
)

const (
	sheetPrec = "route-redistribution-and-precedence"
	sheetOSPF = "ospf"
	sheetBGP  = "bgp"
	sheetOFC  = "overlay-flow-control"
	sheetStat = "static-routes"
	sheetRO   = "routing-overview"
)

var routingRules = []Rule{
	{ID: RRtLocalRemote, Status: StatusVerified, Sheet: sheetRO, Claim: "Routes learned locally by an Edge come from a direct connection, static configuration, or dynamic protocols (BGP or OSPF) and are called local routes", Text: "Local routes come from connected subnets, static routes and OSPF or BGP; routes learned through the overlay from other Edges are remote routes."},
	{ID: RRtOrderOfOps, Status: StatusVerified, Sheet: sheetPrec, Claim: "2 local connected, 3 local static (Preferred enabled; LAN static over WAN static), 4 local NSD static (IPsec over GRE), 5 remote NSD static, 6 remote Edge routes", Text: "Fixed order of operations after the longest prefix: local connected, local static (Preferred), local NSD static, remote NSD static, remote Edge routes, Partner Gateway static, then dynamic routes."},
	{ID: RRtLongestPrefix, Status: StatusVerified, Sheet: sheetPrec, Claim: "The order is: 1 longest prefix length", Text: "The longest prefix is compared first, before the order of operations."},
	{ID: RRtLowerWins, Status: StatusVerified, Sheet: sheetPrec, Claim: "the lower the preference value, the higher the priority", Text: "Among dynamic routes the lower route preference value wins."},
	{ID: RRtDefaultPrefs, Status: StatusVerified, Sheet: sheetPrec, Claim: "OSPF O (intra) 1001; OSPF IA (inter) 1002; eBGP 1003; iBGP 1004", Text: "Default route preferences: OSPF O 1001, OSPF O IA 1002, eBGP 1003, iBGP 1004."},
	{ID: RRtTieBreakers, Status: StatusVerified, Sheet: sheetPrec, Claim: "Tie-breakers when preference is equal: first the protocol attributes (BGP attributes or OSPF metrics), then the Logical ID (LID) of the next-hop device", Text: "Equal preference is broken by protocol attributes, then the Logical ID of the next-hop device, then route age."},
	{ID: RRtOSPFvsBGP, Status: StatusVerified, Sheet: sheetPrec, Claim: "an OSPF route with an intra- or inter-area metric beats a BGP route (LR_EXT_OSPF_INTER)", Text: "An OSPF intra-area or inter-area route beats a BGP route (LR_EXT_OSPF_INTER)."},
	{ID: RRtBGPvsExternal, Status: StatusVerified, Sheet: sheetPrec, Claim: "a BGP route beats an OSPF route of type OE1 or OE2 (LR_EXT_BGP_RT)", Text: "A BGP route beats an OSPF external route of type OE1 or OE2 (LR_EXT_BGP_RT)."},
	{ID: RRtOSPFTypeOrder, Status: StatusVerified, Sheet: sheetPrec, Claim: "between two OSPF routes the metric type order is intra, inter, OE1, OE2 (LR_EXT_METRIC_TYPE) and then lower metric (LR_EXT_METRIC_VAL)", Text: "Between OSPF routes the type order is O, O IA, O E1, O E2 (LR_EXT_METRIC_TYPE), then lower metric (LR_EXT_METRIC_VAL)."},
	{ID: RRtBGPAttrs, Status: StatusVerified, Sheet: sheetBGP, Claim: "Lost Reason codes LR_BGP_LOCAL_PREF (higher local preference wins), LR_BGP_ASPATH_LEN (shorter AS path wins) and LR_BGP_METRIC (lower metric wins)", Text: "BGP routes are compared on local preference (higher), AS-path length (shorter) and metric (lower): LR_BGP_LOCAL_PREF, LR_BGP_ASPATH_LEN, LR_BGP_METRIC."},
	{ID: RRtConnFlag, Status: StatusVerified, Sheet: sheetPrec, Claim: "a connected route beats a non-connected one (LR_CONN_FLAG)", Text: "A connected route beats a non-connected one (LR_CONN_FLAG)."},
	{ID: RRtLastResort, Status: StatusVerified, Sheet: sheetStat, Claim: "with Preferred disabled overlay routes take precedence and the static route is a last resort", Text: "A static route without Preferred is a last resort: overlay routes take precedence."},
	{ID: RRtSwanVsVelo, Status: StatusVerified, Sheet: sheetPrec, Claim: "a non-preferred static WAN route loses to a via-VeloCloud route (LR_NP_SWAN_VS_VELO)", Text: "A non-preferred static WAN route loses to a via-VeloCloud route (LR_NP_SWAN_VS_VELO)."},
	{ID: RRtRemoteOrder, Status: StatusVerified, Sheet: sheetPrec, Claim: "6 remote Edge routes (remote connected, then remote LAN or WAN static)", Text: "Remote Edge routes are ordered remote connected first, then remote static."},
	{ID: RRtOneStatic, Status: StatusVerified, Sheet: sheetStat, Claim: "the Edge advertises only one of them to the SD-WAN overlay", Text: "With several static routes for the same network only one is advertised to the overlay."},
	{ID: RRtStaticAdvertise, Status: StatusVerified, Sheet: sheetStat, Claim: "Advertise: advertises the route over the VPN so other Edges can reach it", Text: "A static route enters the overlay only when Advertise is ticked."},
	{ID: RRtStaticCloudVPN, Status: StatusVerified, Sheet: sheetStat, Claim: "Static routes need Cloud VPN to be activated and the Advertise setting to be visible in the overlay", Text: "Static routes need Cloud VPN to be advertised into the overlay."},
	{ID: RRtProbe, Status: StatusVerified, Sheet: sheetStat, Claim: "ICMP Probe: checks reachability of the next hop and triggers failover", Text: "The ICMP Probe checks the next hop of a static route and triggers failover."},
	{ID: RRtSendAll, Status: StatusVerified, Sheet: sheetRO, Claim: "An Edge sends all locally learned routes to the Gateway based on its configuration", Text: "An Edge sends all locally learned routes to the Gateway, subject to its configuration."},
	{ID: RRtReflect, Status: StatusVerified, Sheet: sheetRO, Claim: "The Gateway acts as a BGP route reflector, reflecting routes from one Edge to the others in the enterprise based on Profile settings", Text: "The Gateway reflects routes from one Edge to the others based on Profile settings."},
	{ID: RRtOSPFAuto, Status: StatusVerified, Sheet: sheetOSPF, Claim: "Routes the Edge learns through OSPF are automatically redistributed to the controller (Gateway)", Text: "Routes learned through OSPF go to the Gateway automatically."},
	{ID: RRtExternalHeld, Status: StatusVerified, Sheet: sheetOSPF, Claim: "External OSPF routes and NSD iBGP routes must be enabled by their flags", Text: "External OSPF routes stay out of the overlay until the External OSPF flag is enabled."},
	{ID: RRtBGPAdvertise, Status: StatusVerified, Sheet: sheetBGP, Claim: "List BGP Routes (prefix received; advertise flag must be True and reachable Yes to enter the overlay route table)", Text: "A BGP route enters the overlay only with advertise True and reachable Yes."},
	{ID: RRtOverlapHeld, Status: StatusVerified, Sheet: sheetOFC, Claim: "A prefix that overlaps a locally connected interface subnet is not added to the OFC table (route-overlap suppression).", Text: "A received prefix that overlaps a locally connected subnet is not advertised into the Overlay Flow Control table."},
	{ID: RRtOverlapBGP, Status: StatusVerified, Sheet: sheetPrec, Claim: "an overlapping prefix is not redistributed into BGP even when overlay prefix redistribution is on", Text: "An overlay prefix that overlaps a connected subnet is not redistributed into BGP."},
	{ID: RRtBestOnly, Status: StatusVerified, Sheet: sheetPrec, Claim: "only redistributes the best route from the system RIB to the destination protocol", Text: "Only the best route of a prefix is redistributed to a protocol."},
	{ID: RRtOSPFExtDefaults, Status: StatusVerified, Sheet: sheetOSPF, Claim: "is the metric OSPF puts in external LSAs generated from the redistributed routes, default 20", Text: "Routes redistributed into OSPF are external routes with metric 20 and type E2 unless set otherwise."},
	{ID: RRtOverlaySwitches, Status: StatusVerified, Sheet: sheetPrec, Claim: "Redistribution out of the overlay: BGP has the Overlay Prefix checkbox", Text: "Overlay routes reach the LAN only through the Overlay Prefix (BGP) or Overlay Prefixes (OSPF) checkbox."},
	{ID: RRtOSPFPassive, Status: StatusVerified, Sheet: sheetOSPF, Claim: "the Edge only supports OSPF on a switched interface as a passive interface", Text: "On a switched interface the Edge supports OSPF only as a passive interface."},
	{ID: RRtOSPFRouted, Status: StatusVerified, Sheet: sheetOSPF, Claim: "To get full OSPF functionality, users must use it on routed interfaces", Text: "Full OSPF functionality needs a routed interface."},
	{ID: RRtOSPFProfile, Status: StatusVerified, Sheet: sheetOSPF, Claim: "OSPF must be activated at the Profile level first", Text: "OSPF must be activated in the Profile before it is enabled on an Edge interface."},
	{ID: RRtOSPFGlobal, Status: StatusVerified, Sheet: sheetOSPF, Claim: "OSPF is configured only on the Global Segment", Text: "OSPF exists only on the Global Segment."},
	{ID: RRtOSPFSummary, Status: StatusVerified, Sheet: sheetOSPF, Claim: "When setting No Advertise, all the external routes (Type-5) under this supernet summarize and do not advertise it", Text: "The OSPF summarization table acts on external (Type-5) routes under a supernet; No Advertise blocks the supernet."},
	{ID: RRtOSPFDefault, Status: StatusVerified, Sheet: sheetOSPF, Claim: "The configuration uses None by default", Text: "The default route into OSPF is off (None) unless an OSPF route type is chosen."},
	{ID: RRtBGPDefault, Status: StatusVerified, Sheet: sheetBGP, Claim: "Default route advertisement: per-neighbor and global settings combine", Text: "A BGP default route is advertised by the Default Route setting; Conditional means only when the Edge learns one."},
	{ID: RRtBGPType, Status: StatusVerified, Sheet: sheetBGP, Claim: "iBGP versus eBGP is decided by whether the local and neighbor ASNs are the same", Text: "A neighbor is iBGP when its ASN equals the local ASN, otherwise eBGP."},
	{ID: RRtBGPSwitches, Status: StatusVerified, Sheet: sheetBGP, Claim: "Advanced Settings (global to IPv4 neighbors; the Edge shares them with NSD neighbors): Overlay Prefix", Text: "BGP redistribution switches: Overlay Prefix, Connected Routes, OSPF and Default Route."},
	{ID: RRtBGPFromOSPF, Status: StatusVerified, Sheet: sheetPrec, Claim: "BGP can take OSPF routes (OSPF checkbox, metric 20) and OSPF can take BGP routes", Text: "BGP can take OSPF routes (metric 20 by default) and OSPF can take BGP routes."},
	{ID: RRtFilterActions, Status: StatusVerified, Sheet: sheetBGP, Claim: "Filter Action Set options: None, Local Preference, Community (with Additive), Metric, As-Path-Prepend and As-Path-Exclude", Text: "A filter can set local preference, metric and AS-path prepend (community and exclude are not modelled)."},
	{ID: RRtFilterMatch, Status: StatusVerified, Sheet: sheetBGP, Claim: "Exact Match is enabled by default; Action Type is Permit or Deny", Text: "A filter rule matches a prefix (Exact Match on by default) and permits or denies it."},
	{ID: RRtNSDiBGPLP, Status: StatusVerified, Sheet: sheetBGP, Claim: "the system cannot apply local preference via an outbound filter", Text: "With iBGP to a Non SD-WAN destination an outbound filter cannot set local preference."},
	{ID: RRtOFCExits, Status: StatusVerified, Sheet: sheetOFC, Claim: "the Orchestrator then sends the change to the relevant Edges, which override their previous preferences", Text: "Preferred VPN Exits for a subnet override the default order of the exits."},
	{ID: RRtPin, Status: StatusVerified, Sheet: sheetOFC, Claim: "pinned routes get preference values starting from 1 (highest priority)", Text: "Pinned routes get a preference starting from 1, ahead of every default value."},
	{ID: RRtGlobalFlags, Status: StatusVerified, Sheet: sheetOFC, Claim: "External OSPF routes and NSD iBGP routes stay out until their flags are enabled", Text: "Global Advertise Flags: External OSPF and NSD iBGP routes stay out of the overlay until enabled."},
	{ID: RRtUnreachable, Status: StatusVerified, Sheet: "overlay-paths", Claim: "PR_UNREACHABLE (for overlay routes, the remote Gateway or Edge peer is unreachable)", Text: "An overlay route is unreachable (PR_UNREACHABLE) when the remote peer is unreachable."},
	{ID: RRtLostNext, Status: StatusVerified, Sheet: sheetRO, Claim: "Lost Reason gives the code for why a route lost the route-preference calculation to the next preferred route", Text: "A route's Lost Reason says why it lost to the next preferred route."},
	{ID: RRtSegmentScoped, Status: StatusVerified, Sheet: sheetRO, Claim: "Routes are scoped to a segment in the diagnostics", Text: "Routes belong to a segment."},
	{ID: RRtConnectedSubnet, Status: StatusVerified, Sheet: sheetBGP, Claim: "redistribute all the connected Interface subnets", Text: "An Edge's connected routes are its interface subnets."},
	{ID: RRtLogicalID, Status: StatusVerified, Sheet: "overlay-paths", Claim: "the tie-breakers LR_NETADDR and LR_LOGICAL_ID", Text: "After attributes, the Logical ID breaks a tie (LR_LOGICAL_ID)."},

	{ID: RNMRtCrossSegment, Status: StatusNotModelled, Sheet: sheetRO, Claim: "how routes cross segments, is not described in the pages read", Text: "not modelled: how routes cross segments. A query or route that needs another segment is not answered."},
	{ID: RNMRtOEPref, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "The same table prints OSPF OE1 as 1001006, OE2 as 1001007 and BGP Uplink as 1001008.", Text: "not modelled: the preference numbers of OSPF E1 and E2 routes (the printed values look inconsistent); only their order after BGP is used."},
	{ID: RNMRtOSPFConnStat, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "whether connected routes and static routes are redistributed into OSPF by default, are not stated", Text: "not modelled: redistribution of connected and static routes into OSPF."},
	{ID: RNMRtRoutedVsVelo, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "Whether a route learned from a routing protocol at this Edge beats a remote Edge connected or static route", Text: "not modelled: whether an OSPF or BGP route learned at this Edge beats a remote Edge connected or static route. The order of operations is applied and flagged."},
	{ID: RNMRtTransit, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "How the transit metric (0, 32 plus the Hub's order, 90) enters route selection", Text: "not modelled: the transit metric (0, 32 plus Hub order, 90) and its use in selection."},
	{ID: RNMRtTieAge, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "The direction of the Logical ID tie-break between Edges", Text: "not modelled: route age as the last tie-breaker and the direction of the Logical ID tie-break are not documented."},
	{ID: RNMRtOSPFFilters, Status: StatusNotModelled, Sheet: sheetOSPF, Claim: "whether inbound OSPF filters default to accept are not stated in the text read", Text: "not modelled: OSPF route filters (inbound and outbound)."},
	{ID: RNMRtOFCRest, Status: StatusNotModelled, Sheet: sheetOFC, Claim: "The default state of each Global Advertise Flag other than the two named exclusions", Text: "not modelled: the internal, external and uplink Global Advertise Flags, Eligible VPN Exits and Use NSD policy."},
	{ID: RNMRtLoops, Status: StatusNotModelled, Sheet: sheetPrec, Claim: "Whether OSPF routes redistributed into BGP and back can loop", Text: "not modelled: redistribution loops. Redistribution is computed in one pass; routes the LAN would learn back are not fed in again."},
	{ID: RNMRtPassive, Status: StatusNotModelled, Sheet: sheetOSPF, Claim: "What a passive OSPF interface on a switched LAN interface does", Text: "not modelled: what a passive OSPF interface does beyond the documented support statement."},
	{ID: RNMRtASPath, Status: StatusNotModelled, Sheet: sheetBGP, Claim: "what happens to the AS path and attributes of a route carried through the overlay from one Edge to another", Text: "not modelled: how the overlay changes the AS path and attributes of a route carried between Edges."},

	{ID: RA69AdvertiseDefaults, Status: StatusAssumption, Assumption: 69, Text: "The Global Advertise Flags for static and connected routes default to on when the topology does not set them (only the External OSPF and NSD iBGP defaults are documented)."},
	{ID: RA70LastResort, Status: StatusAssumption, Assumption: 70, Text: "A static route without Preferred is a last resort behind every other candidate for the same prefix, including routes learned from OSPF or BGP on the Edge (the sheet states it against overlay routes)."},
	{ID: RA71StaticCost, Status: StatusAssumption, Assumption: 71, Text: "A static route without a Cost has cost 0; among static routes of the same type for one prefix the lower cost wins."},
	{ID: RA72ProbeCode, Status: StatusAssumption, Assumption: 72, Text: "A static route whose ICMP probe is down is unreachable with the code SLA_STATE_DOWN."},
	{ID: RA73PassiveNoAdj, Status: StatusAssumption, Assumption: 73, Text: "A passive OSPF interface forms no adjacency, so no route is learned through it."},
	{ID: RA74LogicalIDDir, Status: StatusAssumption, Assumption: 74, Text: "The Logical ID tie-break between Edges prefers the lower Logical ID by default; the direction is a selectable assumption."},
	{ID: RA75DefaultLP, Status: StatusAssumption, Assumption: 75, Text: "A BGP route without a local preference compares as 100 (the common BGP default)."},
	{ID: RA76NSDSlots, Status: StatusAssumption, Assumption: 76, Text: "A Non SD-WAN destination built by an Edge is a local NSD static route on that Edge and a remote NSD static route elsewhere; one built by a Gateway is a local NSD static route on every Edge that enables it."},
	{ID: RA77OSPFDefaultAlways, Status: StatusAssumption, Assumption: 77, Text: "An OSPF default route that is not Conditional is advertised whether or not the Edge has a default route."},
	{ID: RA78BGPPerEdge, Status: StatusAssumption, Assumption: 78, Text: "BGP is authored per Edge (as if overridden); Profile-level BGP inheritance is not modelled."},
	{ID: RA79Overlap, Status: StatusAssumption, Assumption: 79, Text: "Two prefixes overlap when one contains the other (or they are equal)."},
	{ID: RA80SummaryMetric, Status: StatusAssumption, Assumption: 80, Text: "A summary without No Advertise is advertised as one external route with the summary's Metric Type and Metric (default E2 and 20)."},
	{ID: RA81RemoteAttrs, Status: StatusAssumption, Assumption: 81, Text: "A remote route keeps the attributes of the owning Edge's route and its next hop is that Edge."},
	{ID: RA82OriginCloudVPN, Status: StatusAssumption, Assumption: 82, Text: "An Edge advertises a route into the overlay only when Cloud VPN is enabled for the route's segment (stated for static routes, assumed for every other route)."},
	{ID: RA83StageOrder, Status: StatusAssumption, Assumption: 83, Text: "The reachability stages follow the Section 6 diagnostic order (our teaching method): known, advertised, filtered, redistributed, winner, path. The redistribution stage applies only to a source Edge that runs OSPF or BGP."},
	{ID: RA84DestinationMatch, Status: StatusAssumption, Assumption: 84, Text: "An origin route matches a queried prefix when it is the same or a less specific prefix; the most specific one is used."},
	{ID: RA22Redistribution, Status: StatusAssumption, Assumption: 22, Text: "Default states of Overlay Prefix (BGP), Overlay Prefixes (OSPF), Connected Routes (BGP), OSPF into BGP and BGP into OSPF are all off, until observed."},
	{ID: RA23FilterDefault, Status: StatusAssumption, Assumption: 23, Text: "A route filter permits a route that matches no rule, and rules are evaluated top to bottom with the first match winning; both are selectable assumptions."},
	{ID: RA24BGPOrder, Status: StatusAssumption, Assumption: 24, Text: "Between two BGP routes the Edge applies local preference, then AS-path length, then metric."},
	{ID: RA28StaticSegment, Status: StatusAssumption, Assumption: 28, Text: "Static routes apply to the Global Segment unless the topology says otherwise; routes do not cross segments."},
	{ID: RA29SummaryOnly, Status: StatusAssumption, Assumption: 29, Text: "The summarization table affects only OSPF external routes; summarizing OSPF routes into the overlay is not modelled."},
	{ID: RA32CloudVPNReflect, Status: StatusAssumption, Assumption: 32, Text: "The Gateway reflects a route only to Edges whose Profile and segment have Cloud VPN enabled."},
	{ID: RA30NeighborDown, Status: StatusAssumption, Assumption: 30, Text: "A neighbor that is not Full or Established removes the routes learned through it immediately; BFD timing is not modelled."},
}
