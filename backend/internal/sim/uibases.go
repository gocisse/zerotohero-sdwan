package sim

// Evidence for claims the UI makes about screens, so the UI never invents a label. Each key is used by the
// simulator screens; every entry cites a registered rule.
const (
	RMonitorEdgeTabs = "monitor-edge-tabs"
	RMonitorLinksTab = "monitor-links-tab"
	RMonitorPathsTab = "monitor-paths-tab"
	RListPathsStates = "list-paths-states"

	RMonitorRoutingTabs = "monitor-routing-tabs"
	RRouteTableDump     = "diag-route-table-dump"
	RRouteTableColumns  = "route-table-dump-columns"
	RNoEdgeRoutesTab    = "no-edge-routes-tab"
	ROFCRoutesList      = "ofc-routes-list"
	RNMGatewayRouteTab  = "nm-gateway-route-table"
)

var uiRules = []Rule{
	{ID: RMonitorEdgeTabs, Status: StatusVerified, Sheet: "vco-overlay-ui", Claim: "Monitor > Edges > Edge tabs: QoE, Links, Paths, Flows, System", Text: "Monitor > Edges > (Edge) has the tabs QoE, Links, Paths, Flows and System."},
	{ID: RMonitorLinksTab, Status: StatusVerified, Sheet: "dmpo-vco-ui", Claim: "The Links tab lists per-link metrics and offers a Transport Groups view", Text: "The Links tab lists links and offers a Transport Groups view (Public Wired, Public Wireless, Private Wired)."},
	{ID: RMonitorPathsTab, Status: StatusVerified, Sheet: "dmpo-vco-ui", Claim: "The Paths tab shows paths to peers", Text: "The Paths tab shows paths to peers."},
	{ID: RMonitorRoutingTabs, Status: StatusVerified, Sheet: "vco-routing-ui", Claim: "Monitor > Routing has the tabs Multicast Groups, PIM Neighbors, BGP Edge Neighbor State, BFD, BGP Gateway Neighbor State and Gateway Route Table", Text: "Monitor > Routing has the tabs Multicast Groups, PIM Neighbors, BGP Edge Neighbor State, BFD, BGP Gateway Neighbor State and Gateway Route Table."},
	{ID: RRouteTableDump, Status: StatusVerified, Sheet: "vco-routing-ui", Claim: "Remote Diagnostics tests for routing on an Edge: Route Table Dump (p80)", Text: "A per-Edge route table is the Remote Diagnostics test Route Table Dump (Diagnostics tab, choose the Edge, pick the test, Run)."},
	{ID: RRouteTableColumns, Status: StatusVerified, Sheet: "routing-overview", Claim: "The Route Table Dump output columns are Address, Segment, Netmask, Type", Text: "Route Table Dump columns: Address, Segment, Netmask, Type, Cost, Preference, Order, Reachable, Next Hop, Next Hop Name, Destination Name, Lost Reason and (Not) Reachable Reason."},
	{ID: RNoEdgeRoutesTab, Status: StatusVerified, Sheet: "vco-routing-ui", Claim: "no per-Edge routes tab was described in the part read", Text: "Monitor > Edges shows no per-Edge routes tab in the documentation read."},
	{ID: ROFCRoutesList, Status: StatusVerified, Sheet: "overlay-flow-control", Claim: "The OFC Routes List table has the columns Segment, Subnet, Preferred VPN Exits, Route Type, Last Update, Created On, IPv4 Subnet and Eligible VPN Exits", Text: "Configure > Overlay Flow Control lists routes with the columns Segment, Subnet, Preferred VPN Exits, Route Type, Last Update, Created On, IPv4 Subnet and Eligible VPN Exits."},
	{ID: RNMGatewayRouteTab, Status: StatusNotModelled, Sheet: "vco-routing-ui", Claim: "The Gateway Route Table tab and BGP Gateway Neighbor State tab exist under Monitor > Routing (names as read; their columns were not read).", Text: "not modelled: the contents of Monitor > Routing > Gateway Route Table (its columns were not read)."},
	{ID: RListPathsStates, Status: StatusVerified, Sheet: "overlay-paths", Claim: "The packets of a flow can take any tunnel", Text: "Tunnels carry flows based on their real-time health."},
}

// UIBases returns the evidence behind screen-level claims, keyed for the frontend.
func UIBases() map[string]Basis {
	return map[string]Basis{
		"edge_tabs":            basisOf(RMonitorEdgeTabs),   // rule: monitor-edge-tabs
		"links_tab":            basisOf(RMonitorLinksTab),   // rule: monitor-links-tab
		"paths_tab":            basisOf(RMonitorPathsTab),   // rule: monitor-paths-tab
		"path_health":          basisOf(RListPathsStates),   // rule: list-paths-states
		"paths_mode":           basisOf(RPathsMode),         // rule: paths-mode
		"transport_groups":     basisOf(RA10TransportGroup), // rule: assume-10
		"degraded":             basisOf(RA55DegradedState),  // rule: assume-55
		"sla_thresholds":       notModelled(RNMSLAThresholds, "What makes a link degraded").Basis,
		"build_stub":           basisOf(RA16BuildNAT), // rule: assume-16
		"secure_tunnel":        notModelled(RNMSecureTunnel, "The secure tunnel decision").Basis,
		"routing_monitor_tabs": basisOf(RMonitorRoutingTabs),                                 // rule: monitor-routing-tabs
		"route_table_dump":     basisOf(RRouteTableDump),                                     // rule: diag-route-table-dump
		"route_table_columns":  basisOf(RRouteTableColumns),                                  // rule: route-table-dump-columns
		"no_edge_routes_tab":   basisOf(RNoEdgeRoutesTab),                                    // rule: no-edge-routes-tab
		"ofc_routes_list":      basisOf(ROFCRoutesList),                                      // rule: ofc-routes-list
		"gateway_route_table":  notModelled(RNMGatewayRouteTab, "Gateway Route Table").Basis, // rule: nm-gateway-route-table
	}
}
