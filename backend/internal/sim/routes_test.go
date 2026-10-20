package sim

import (
	"encoding/json"
	"math/rand"
	"net/netip"
	"reflect"
	"testing"
)

// mini is a small valid topology for selection tests: x and y run OSPF and BGP toward LAN routers, z only observes.
func mini() *Topology {
	edge := func(id string, lid int, lan bool) Edge {
		e := Edge{ID: id, Name: id, Role: RoleBranch, Profile: "p", LogicalID: &lid,
			Interfaces: []Interface{{ID: "ge1", Kind: IfWan, Link: "l-" + id}}}
		if lan {
			e.Interfaces = append(e.Interfaces, Interface{ID: "ge3", Kind: IfRouted, Address: "10.1.1.1/30"})
			e.Routing = &EdgeRouting{
				OSPF: &EdgeOSPF{Interfaces: []OSPFInterface{{ID: "ge3", Mode: OSPFActive}}},
				BGP: &BGPConfig{LocalASN: 65000, Neighbors: []BGPNeighbor{
					{ID: "n1", IP: "10.1.1.2", ASN: 65002}, {ID: "n2", IP: "10.1.1.3", ASN: 65000}}},
			}
		}
		return e
	}
	return &Topology{
		Version: 1, Meta: Meta{ID: "mini", Title: "m", Summary: "s", Teaching: true}, Customer: Customer{ID: "c", Name: "C"},
		Segments: []Segment{{ID: "global", Name: "Global", Type: SegRegular}},
		Gateways: []Gateway{{ID: "g1", Region: "r1"}},
		Links: []WanLink{{ID: "l-x", Type: LinkPublic, Medium: MediumWired, UpMbps: 10, DownMbps: 10}, {ID: "l-y", Type: LinkPublic, Medium: MediumWired, UpMbps: 10, DownMbps: 10},
			{ID: "l-z", Type: LinkPublic, Medium: MediumWired, UpMbps: 10, DownMbps: 10}},
		Profiles: []Profile{{ID: "p", Name: "P", Segments: []string{"global"},
			CloudVPN: &CloudVPN{Segments: []string{"global"}, BranchToBranch: &BranchToBranch{Enabled: true, Via: B2BCloudGateways}},
			Routing:  &ProfileRouting{OSPF: &OSPFProfile{Enabled: true, Areas: []OSPFArea{{ID: "0"}}}}}},
		Sites: []Site{{ID: "s", Name: "S", Region: "r1", Edges: []Edge{edge("x", 1, true), edge("y", 2, true), edge("z", 3, false)}}},
	}
}

func extFlag(tp *Topology) {
	t := true
	tp.OverlayFlowControl = &OverlayFlowControl{AdvertiseFlags: &AdvertiseFlags{ExternalOSPF: &t}}
}

func addLAN(tp *Topology, edge string, lr ...LANRoute) {
	e := edgeOf(tp, edge)
	e.Routing.LANRoutes = append(e.Routing.LANRoutes, lr...)
}

func ospfR(id, typ string, metric int) LANRoute {
	return LANRoute{ID: id, Prefix: "10.9.0.0/24", Protocol: ProtoOSPF, Via: "ge3", OSPFType: typ, Metric: &metric}
}

func bgpR(id, via string, lp, med *int, path ...int) LANRoute {
	return LANRoute{ID: id, Prefix: "10.9.0.0/24", Protocol: ProtoBGP, Via: via, BGP: &BGPAttrs{ASPath: path, LocalPreference: lp, MED: med}}
}

func routesFor(t *testing.T, rs *RouteSet, edge, prefix string) []Route {
	t.Helper()
	for _, e := range rs.Edges {
		if e.Edge == edge {
			for _, st := range e.Segments {
				var out []Route
				for _, r := range st.Routes {
					if r.Prefix == prefix {
						out = append(out, r)
					}
				}
				return out
			}
		}
	}
	return nil
}

func computeR(t *testing.T, tp *Topology, c Conditions, as RoutingAssumptions) *RouteSet {
	t.Helper()
	if errs := Errors(Validate(tp)); len(errs) != 0 {
		t.Fatalf("fixture invalid: %v", errs)
	}
	rs, err := ComputeRoutes(tp, c, as, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

// Each documented ordering between two dynamic routes learned at different Edges, seen from z.
func TestDynamicRouteOrdering(t *testing.T) {
	i := func(n int) *int { return &n }
	cases := []struct {
		name    string
		x, y    LANRoute
		ext     bool
		winner  string // origin Edge of the selected route
		winType string
		code    string // Lost Reason code of the loser ("" = none documented)
		mutate  func(*Topology)
		assume  func(*RoutingAssumptions)
	}{
		{"O beats O IA", ospfR("r", OSPFIntra, 10), ospfR("r", OSPFInter, 1), false, "x", TypeO, LRExtMetricType, nil, nil},
		{"O IA beats eBGP (1002 vs 1003)", ospfR("r", OSPFInter, 10), bgpR("r", "n1", nil, nil, 65002), false, "x", TypeOIA, LRExtOSPFInter, nil, nil},
		{"O beats eBGP even with a high local preference", bgpR("r", "n1", i(900), nil, 65002), ospfR("r", OSPFIntra, 99), false, "y", TypeO, LRExtOSPFInter, nil, nil},
		{"eBGP beats iBGP (1003 vs 1004)", bgpR("r", "n2", nil, nil), bgpR("r", "n1", nil, nil, 65002), false, "y", TypeEB, "", nil, nil},
		{"iBGP beats OSPF E1", ospfR("r", OSPFE1, 1), bgpR("r", "n2", nil, nil), true, "y", TypeIB, LRExtBGPRoute, nil, nil},
		{"E1 beats E2", ospfR("r", OSPFE2, 1), ospfR("r", OSPFE1, 50), true, "y", TypeOE1, LRExtMetricType, nil, nil},
		{"lower metric among E2", ospfR("r", OSPFE2, 30), ospfR("r", OSPFE2, 20), true, "y", TypeOE2, LRExtMetricVal, nil, nil},
		{"lower cost among O", ospfR("r", OSPFIntra, 30), ospfR("r", OSPFIntra, 20), false, "y", TypeO, LRExtMetricVal, nil, nil},
		{"higher local preference", bgpR("r", "n1", i(100), nil, 65002), bgpR("r", "n1", i(200), nil, 65002, 65003), false, "y", TypeEB, LRBGPLocalPref, nil, nil},
		{"unset local preference compares as 100", bgpR("r", "n1", nil, nil, 65002), bgpR("r", "n1", i(150), nil, 65002), false, "y", TypeEB, LRBGPLocalPref, nil, nil},
		{"shorter AS path", bgpR("r", "n1", nil, nil, 65002, 65003), bgpR("r", "n1", nil, nil, 65002), false, "y", TypeEB, LRBGPASPathLen, nil, nil},
		{"lower MED", bgpR("r", "n1", nil, i(50), 65002), bgpR("r", "n1", nil, i(10), 65002), false, "y", TypeEB, LRBGPMetric, nil, nil},
		{"logical id, lower wins by default", bgpR("r", "n1", nil, nil, 65002), bgpR("r", "n1", nil, nil, 65002), false, "x", TypeEB, LRLogicalID, nil, nil},
		{"logical id, higher wins when selected", bgpR("r", "n1", nil, nil, 65002), bgpR("r", "n1", nil, nil, 65002), false, "y", TypeEB, LRLogicalID, nil, func(a *RoutingAssumptions) { a.LogicalIDTieBreak = TieHigher }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := mini()
			if c.ext {
				extFlag(tp)
			}
			addLAN(tp, "x", c.x)
			addLAN(tp, "y", c.y)
			as := DefaultRoutingAssumptions()
			if c.assume != nil {
				c.assume(&as)
			}
			rs := computeR(t, tp, Conditions{}, as)
			rows := routesFor(t, rs, "z", "10.9.0.0/24")
			if len(rows) != 2 {
				t.Fatalf("want 2 candidates, got %+v", rows)
			}
			if !rows[0].Selected || rows[0].Origin != c.winner || rows[0].RouteType != c.winType {
				t.Errorf("winner = %s %s selected=%v; want %s %s: %s", rows[0].Origin, rows[0].RouteType, rows[0].Selected, c.winner, c.winType, rows[0].Reason)
			}
			if rows[1].Selected || rows[1].LostReasonCode != c.code {
				t.Errorf("loser selected=%v code=%q, want code %q (%s)", rows[1].Selected, rows[1].LostReasonCode, c.code, rows[1].Reason)
			}
			if rows[1].Basis.Status == "" || rows[0].Basis.Status == "" {
				t.Error("every route carries a basis")
			}
		})
	}
}

func TestPreferenceNumbersAndOEPreferenceIsNotModelled(t *testing.T) {
	tp := mini()
	extFlag(tp)
	addLAN(tp, "x", ospfR("a", OSPFIntra, 1), ospfR("b", OSPFE2, 1))
	tp.Sites[0].Edges[0].Routing.LANRoutes[1].Prefix = "10.8.0.0/24"
	rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
	o := routesFor(t, rs, "x", "10.9.0.0/24")[0]
	if o.Preference == nil || *o.Preference != 1001 {
		t.Errorf("O preference = %v, want 1001", o.Preference)
	}
	e2 := routesFor(t, rs, "x", "10.8.0.0/24")[0]
	if e2.Preference != nil || e2.PreferenceNote == "" || !hasRule(rs.NotModelled, RNMRtOEPref) {
		t.Errorf("E2 preference must be 'not modelled', got %v %q", e2.Preference, e2.PreferenceNote)
	}
}

func hasRule(nms []NotModelled, rule string) bool {
	for _, n := range nms {
		if n.Rule == rule {
			return true
		}
	}
	return false
}

// Equal in every documented key and no Logical IDs: the tie is flagged, not silently decided.
func TestUnresolvedTieIsFlagged(t *testing.T) {
	tp := mini()
	for si := range tp.Sites {
		for ei := range tp.Sites[si].Edges {
			tp.Sites[si].Edges[ei].LogicalID = nil
		}
	}
	addLAN(tp, "x", bgpR("r", "n1", nil, nil, 65002))
	addLAN(tp, "y", bgpR("r", "n1", nil, nil, 65002))
	rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
	rows := routesFor(t, rs, "z", "10.9.0.0/24")
	if !hasRule(rs.NotModelled, RNMRtTieAge) || rows[1].Basis.Rule != RNMRtTieAge && len(rows[1].Also) == 0 {
		t.Errorf("tie should be flagged not modelled: %+v", rs.NotModelled)
	}
}

func TestOrderOfOperations(t *testing.T) {
	t.Run("connected beats a static route for the same prefix", func(t *testing.T) {
		tp := mini()
		z := edgeOf(tp, "z")
		z.Interfaces = append(z.Interfaces, Interface{ID: "ge3", Kind: IfRouted, Address: "10.5.0.1/24"})
		z.Routing = &EdgeRouting{StaticRoutes: []StaticRoute{{ID: "s", Subnet: "10.5.0.0/24", Interface: "ge1", Preferred: true}}}
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		rows := routesFor(t, rs, "z", "10.5.0.0/24")
		if len(rows) != 2 || rows[0].Source != SrcConnected || rows[1].LostReasonCode != LRConnFlag {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("a preferred local static route beats a remote static route", func(t *testing.T) {
		tp := mini()
		addStatic(tp, "x", StaticRoute{ID: "sx", Subnet: "10.7.0.0/24", Interface: "ge1", Preferred: true, Advertise: true})
		addStatic(tp, "z", StaticRoute{ID: "sz", Subnet: "10.7.0.0/24", Interface: "ge1", Preferred: true})
		rows := routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "z", "10.7.0.0/24")
		if len(rows) != 2 || rows[0].Source != SrcLocalStatic || !rows[0].Selected || rows[1].Source != SrcRemoteStatic {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("a static route without Preferred is a last resort, and a WAN one loses with LR_NP_SWAN_VS_VELO", func(t *testing.T) {
		tp := mini()
		addStatic(tp, "x", StaticRoute{ID: "sx", Subnet: "10.7.0.0/24", Interface: "ge1", Preferred: true, Advertise: true})
		addStatic(tp, "z", StaticRoute{ID: "sz", Subnet: "10.7.0.0/24", Interface: "ge1"})
		rows := routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "z", "10.7.0.0/24")
		if rows[0].Source != SrcRemoteStatic || rows[1].Source != SrcLocalStatic || rows[1].LostReasonCode != LRSwanVsVelo {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("a remote connected route comes before a remote static route", func(t *testing.T) {
		tp := mini()
		addStatic(tp, "x", StaticRoute{ID: "sx", Subnet: "10.6.0.0/24", Interface: "ge1", Preferred: true, Advertise: true})
		y := edgeOf(tp, "y")
		y.Interfaces = append(y.Interfaces, Interface{ID: "ge4", Kind: IfRouted, Address: "10.6.0.1/24"})
		rows := routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "z", "10.6.0.0/24")
		if rows[0].Source != SrcRemoteConnected || rows[1].Source != SrcRemoteStatic {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("a remote Edge route beats a dynamic route and the conflicting rows are flagged", func(t *testing.T) {
		tp := mini()
		addStatic(tp, "z", StaticRoute{ID: "sz", Subnet: "10.9.0.0/24", Interface: "ge1", Preferred: true, Advertise: true})
		addLAN(tp, "x", ospfR("r", OSPFIntra, 1))
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		rows := routesFor(t, rs, "x", "10.9.0.0/24")
		if rows[0].Source != SrcRemoteStatic || !hasRule(rs.NotModelled, RNMRtRoutedVsVelo) {
			t.Errorf("%+v / %+v", rows[0], rs.NotModelled)
		}
	})
	t.Run("same-type static routes: the lower cost wins", func(t *testing.T) {
		tp := mini()
		addStatic(tp, "z", StaticRoute{ID: "s1", Subnet: "10.4.0.0/24", Interface: "ge1", Preferred: true, Cost: ip(10)},
			StaticRoute{ID: "s2", Subnet: "10.4.0.0/24", Interface: "ge1", Preferred: true, Cost: ip(5)})
		rows := routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "z", "10.4.0.0/24")
		if rows[0].Via != "s2" {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("the longest prefix wins before the order of operations", func(t *testing.T) {
		tp := mini()
		addLAN(tp, "x", ospfR("r", OSPFIntra, 1))
		addStatic(tp, "y", StaticRoute{ID: "d", Subnet: "10.0.0.0/8", Interface: "ge1", Preferred: true, Advertise: true})
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		sel, _ := rs.Lookup("z", "global", netip.MustParsePrefix("10.9.0.7/32"))
		if sel == nil || sel.Prefix != "10.9.0.0/24" || sel.Source != SrcRemoteDynamic {
			t.Errorf("lookup = %+v", sel)
		}
		sel, _ = rs.Lookup("z", "global", netip.MustParsePrefix("10.200.0.1/32"))
		if sel == nil || sel.Prefix != "10.0.0.0/8" {
			t.Errorf("lookup = %+v", sel)
		}
	})
}

func addStatic(tp *Topology, edge string, sr ...StaticRoute) {
	e := edgeOf(tp, edge)
	if e.Routing == nil {
		e.Routing = &EdgeRouting{}
	}
	e.Routing.StaticRoutes = append(e.Routing.StaticRoutes, sr...)
}

func TestAdvertisementRules(t *testing.T) {
	run := func(mut func(*Topology), as RoutingAssumptions) *RouteSet {
		tp := mini()
		addLAN(tp, "x", ospfR("r", OSPFE2, 5))
		mut(tp)
		return computeR(t, tp, Conditions{}, as)
	}
	t.Run("external OSPF is held out by default", func(t *testing.T) {
		rs := run(func(*Topology) {}, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "z", "10.9.0.0/24")) != 0 || len(routesFor(t, rs, "x", "10.9.0.0/24")) != 1 {
			t.Error("E2 route must stay local")
		}
	})
	t.Run("the External OSPF flag lets it in", func(t *testing.T) {
		rs := run(extFlag, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "z", "10.9.0.0/24")) != 1 {
			t.Error("E2 route must reach z")
		}
	})
	t.Run("a static route needs Advertise", func(t *testing.T) {
		rs := run(func(tp *Topology) { addStatic(tp, "x", StaticRoute{ID: "s", Subnet: "10.3.0.0/24", Interface: "ge1"}) }, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "z", "10.3.0.0/24")) != 0 {
			t.Error("not advertised")
		}
	})
	t.Run("the static flag assumption switches static advertisement off", func(t *testing.T) {
		as := DefaultRoutingAssumptions()
		as.AdvertiseStatic = false
		rs := run(func(tp *Topology) {
			addStatic(tp, "x", StaticRoute{ID: "s", Subnet: "10.3.0.0/24", Interface: "ge1", Advertise: true})
		}, as)
		if len(routesFor(t, rs, "z", "10.3.0.0/24")) != 0 {
			t.Error("flag off")
		}
		if !usedSwitch(rs, "advertise_static") {
			t.Error("the result should say it used advertise_static")
		}
	})
	t.Run("only one of several static routes is advertised", func(t *testing.T) {
		rs := run(func(tp *Topology) {
			addStatic(tp, "x", StaticRoute{ID: "s1", Subnet: "10.3.0.0/24", Interface: "ge1", Advertise: true, Cost: ip(9)},
				StaticRoute{ID: "s2", Subnet: "10.3.0.0/24", Interface: "ge1", Advertise: true, Cost: ip(1)})
		}, DefaultRoutingAssumptions())
		if got := routesFor(t, rs, "z", "10.3.0.0/24"); len(got) != 1 {
			t.Errorf("want one remote route, got %d", len(got))
		}
	})
	t.Run("Cloud VPN off on the owner stops advertisement (assumption 82)", func(t *testing.T) {
		rs := run(func(tp *Topology) {
			extFlag(tp)
			edgeOf(tp, "x").Overrides.CloudVPN = &CloudVPN{Segments: []string{}}
			edgeOf(tp, "x").Overrides.CloudVPN.Segments = nil
			profileOf(tp, "p").CloudVPN = nil
			profileOf(tp, "p").CloudVPN = &CloudVPN{Segments: []string{"global"}, BranchToBranch: &BranchToBranch{Enabled: true, Via: B2BCloudGateways}}
			edgeOf(tp, "x").Overrides.CloudVPN = &CloudVPN{Segments: []string{"global"}}
			edgeOf(tp, "x").Overrides.CloudVPN = nil
			// x gets its own Profile without Cloud VPN
			tp.Profiles = append(tp.Profiles, Profile{ID: "off", Name: "Off", Segments: []string{"global"}, Routing: profileOf(tp, "p").Routing})
			edgeOf(tp, "x").Profile = "off"
		}, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "z", "10.9.0.0/24")) != 0 {
			t.Error("x has no Cloud VPN: nothing is advertised")
		}
	})
	t.Run("a received prefix overlapping a connected subnet is held back", func(t *testing.T) {
		rs := run(func(tp *Topology) {
			extFlag(tp)
			x := edgeOf(tp, "x")
			x.Interfaces = append(x.Interfaces, Interface{ID: "ge4", Kind: IfRouted, Address: "10.9.0.1/16"})
		}, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "z", "10.9.0.0/24")) != 0 {
			t.Error("overlap suppression")
		}
	})
}

func usedSwitch(rs *RouteSet, key string) bool {
	for _, a := range rs.Assumptions {
		if a.Key == key && a.Used {
			return true
		}
	}
	return false
}

func TestRedistributionIntoTheLAN(t *testing.T) {
	adverts := func(rs *RouteSet, edge, proto string) []LANAdvertisement {
		for _, e := range rs.Edges {
			if e.Edge == edge {
				var out []LANAdvertisement
				for _, a := range e.LANAdvertisements {
					if a.Protocol == proto {
						out = append(out, a)
					}
				}
				return out
			}
		}
		return nil
	}
	build := func() *Topology {
		tp := mini()
		addLAN(tp, "y", ospfR("r", OSPFIntra, 1))
		return tp
	}
	t.Run("BGP overlay prefix: the assumption decides, and the result shows it", func(t *testing.T) {
		off := computeR(t, build(), Conditions{}, DefaultRoutingAssumptions())
		as := DefaultRoutingAssumptions()
		as.BGPOverlayPrefix = true
		on := computeR(t, build(), Conditions{}, as)
		a0, a1 := ofKind(adverts(off, "x", ProtoBGP), "overlay prefix"), ofKind(adverts(on, "x", ProtoBGP), "overlay prefix")
		if len(a0) == 0 || a0[0].Advertised || a0[0].Switch != "bgp_overlay_prefix" {
			t.Errorf("off: %+v", a0)
		}
		if len(a1) == 0 || !a1[0].Advertised {
			t.Errorf("on: %+v", a1)
		}
		if !usedSwitch(off, "bgp_overlay_prefix") || !usedSwitch(on, "bgp_overlay_prefix") {
			t.Error("outputs must list the assumption as used")
		}
	})
	t.Run("a value written in the topology wins over the assumption", func(t *testing.T) {
		tp := build()
		yes := true
		edgeOf(tp, "x").Routing.BGP.Redistribute = &BGPRedistribute{OverlayPrefix: &yes}
		as := DefaultRoutingAssumptions()
		rs := computeR(t, tp, Conditions{}, as)
		if a := ofKind(adverts(rs, "x", ProtoBGP), "overlay prefix"); len(a) == 0 || !a[0].Advertised || a[0].Switch != "" {
			t.Errorf("%+v", a)
		}
	})
	t.Run("OSPF externals default to E2 with metric 20", func(t *testing.T) {
		tp := build()
		yes := true
		profileOf(tp, "p").Routing.OSPF.Redistribute = &OSPFRedistribute{OverlayPrefixes: &yes}
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		a := adverts(rs, "x", ProtoOSPF)
		if len(a) == 0 || !a[0].Advertised || a[0].OSPFType != MetricE2 || a[0].Metric == nil || *a[0].Metric != 20 {
			t.Errorf("%+v", a)
		}
		if !hasRule(rs.NotModelled, RNMRtOSPFConnStat) {
			t.Error("connected and static into OSPF must be marked not modelled")
		}
	})
	t.Run("OSPF summaries replace the members, and No Advertise blocks the supernet", func(t *testing.T) {
		tp := build()
		yes := true
		profileOf(tp, "p").Routing.OSPF.Redistribute = &OSPFRedistribute{OverlayPrefixes: &yes}
		addStatic(tp, "y", StaticRoute{ID: "a", Subnet: "10.60.0.0/24", Interface: "ge1", Advertise: true}, StaticRoute{ID: "b", Subnet: "10.60.1.0/24", Interface: "ge1", Advertise: true})
		profileOf(tp, "p").Routing.OSPF.Summaries = []OSPFSummary{{Subnet: "10.60.0.0/22"}}
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		var sum, members int
		for _, a := range adverts(rs, "x", ProtoOSPF) {
			if a.Kind == "summary" && a.Advertised {
				sum++
			}
			if a.SummaryOf == "10.60.0.0/22" && !a.Advertised {
				members++
			}
		}
		if sum != 1 || members != 2 {
			t.Errorf("summary=%d members=%d", sum, members)
		}
		profileOf(tp, "p").Routing.OSPF.Summaries[0].NoAdvertise = true
		rs = computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		for _, a := range adverts(rs, "x", ProtoOSPF) {
			if (a.Kind == "summary" || a.SummaryOf != "") && a.Advertised {
				t.Errorf("No Advertise must block: %+v", a)
			}
		}
	})
	t.Run("only the best route of a prefix is redistributed", func(t *testing.T) {
		tp := build() // y offers an OSPF route; x also learns the same prefix from BGP, but loses to the OSPF route
		yes := true
		addLAN(tp, "x", bgpR("r", "n1", nil, nil, 65002))
		profileOf(tp, "p").Routing.OSPF.Redistribute = &OSPFRedistribute{BGP: &yes}
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		for _, a := range adverts(rs, "x", ProtoOSPF) {
			if a.Kind == "BGP route" {
				t.Errorf("the losing BGP route must not be redistributed: %+v", a)
			}
		}
	})
	t.Run("OSPF into BGP and BGP into OSPF follow their switches", func(t *testing.T) {
		tp := build()
		addLAN(tp, "x", ospfR("r", OSPFIntra, 0))
		as := DefaultRoutingAssumptions()
		as.BGPRedistributeOSPF = true
		rs := computeR(t, tp, Conditions{}, as)
		found := false
		for _, a := range adverts(rs, "x", ProtoBGP) {
			if a.Kind == "OSPF route" && a.Advertised && *a.Metric == 20 {
				found = true
			}
		}
		if !found {
			t.Error("OSPF route should be redistributed into BGP with metric 20")
		}
	})
	t.Run("default routes: OSPF None by default, conditional needs a default route", func(t *testing.T) {
		tp := build()
		yes := true
		profileOf(tp, "p").Routing.OSPF.Redistribute = &OSPFRedistribute{OverlayPrefixes: &yes, DefaultRoute: MetricE1, DefaultConditional: true}
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		for _, a := range adverts(rs, "x", ProtoOSPF) {
			if a.Kind == "default route" && a.Advertised {
				t.Error("conditional default without a learned default route")
			}
		}
		addStatic(tp, "z", StaticRoute{ID: "d", Subnet: "0.0.0.0/0", Interface: "ge1", Preferred: true, Advertise: true})
		rs = computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		ok := false
		for _, a := range adverts(rs, "x", ProtoOSPF) {
			ok = ok || (a.Kind == "default route" && a.Advertised && a.OSPFType == MetricE1)
		}
		if !ok {
			t.Error("conditional default should now be advertised")
		}
	})
}

func ofKind(list []LANAdvertisement, kind string) []LANAdvertisement {
	var out []LANAdvertisement
	for _, a := range list {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

func TestFilters(t *testing.T) {
	i := func(n int) *int { return &n }
	build := func(rules ...BGPFilterRule) *Topology {
		tp := mini()
		b := edgeOf(tp, "x").Routing.BGP
		b.Filters = []BGPFilter{{ID: "f", Rules: rules}}
		b.Neighbors[0].InboundFilter = "f"
		addLAN(tp, "x", bgpR("r", "n1", nil, nil, 65002))
		return tp
	}
	permitAll := BGPFilterRule{Action: FilterPermit}
	t.Run("a filter sets local preference and MED and prepends", func(t *testing.T) {
		tp := build(BGPFilterRule{Match: BGPFilterMatch{Prefix: "10.9.0.0/24"}, Action: FilterPermit, Set: BGPFilterSet{LocalPreference: i(250), Metric: i(7), ASPathPrepend: []int{65000, 65000}}})
		r := routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "x", "10.9.0.0/24")[0]
		if r.LocalPreference == nil || *r.LocalPreference != 250 || r.Metric == nil || *r.Metric != 7 || len(r.ASPath) != 3 || len(r.Notes) != 3 {
			t.Errorf("%+v", r)
		}
	})
	t.Run("deny removes the route and says so", func(t *testing.T) {
		tp := build(BGPFilterRule{Match: BGPFilterMatch{Prefix: "10.9.0.0/24"}, Action: FilterDeny})
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		if len(routesFor(t, rs, "x", "10.9.0.0/24")) != 0 {
			t.Error("denied route installed")
		}
		w := rs.Edges[0].Withheld
		if len(w) != 1 || w[0].Stage != "filtered" {
			t.Errorf("%+v", w)
		}
	})
	t.Run("default action when nothing matches is selectable", func(t *testing.T) {
		tp := build(BGPFilterRule{Match: BGPFilterMatch{Prefix: "10.1.0.0/24"}, Action: FilterPermit})
		permit := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		as := DefaultRoutingAssumptions()
		as.FilterDefaultAction = FilterDeny
		deny := computeR(t, tp, Conditions{}, as)
		if len(routesFor(t, permit, "x", "10.9.0.0/24")) != 1 || len(routesFor(t, deny, "x", "10.9.0.0/24")) != 0 || !usedSwitch(deny, "filter_default_action") {
			t.Error("default action switch has no effect")
		}
	})
	t.Run("evaluation order is selectable", func(t *testing.T) {
		tp := build(BGPFilterRule{Action: FilterDeny}, permitAll)
		first := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		as := DefaultRoutingAssumptions()
		as.FilterOrder = FilterOrderLast
		last := computeR(t, tp, Conditions{}, as)
		if len(routesFor(t, first, "x", "10.9.0.0/24")) != 0 || len(routesFor(t, last, "x", "10.9.0.0/24")) != 1 || !usedSwitch(last, "filter_order") {
			t.Error("order switch has no effect")
		}
	})
	t.Run("a non-exact match covers more specific prefixes", func(t *testing.T) {
		no := false
		tp := build(BGPFilterRule{Match: BGPFilterMatch{Prefix: "10.0.0.0/8", ExactMatch: &no}, Action: FilterDeny})
		if len(routesFor(t, computeR(t, tp, Conditions{}, DefaultRoutingAssumptions()), "x", "10.9.0.0/24")) != 0 {
			t.Error("non-exact deny")
		}
	})
	t.Run("an outbound filter stops an advertisement", func(t *testing.T) {
		tp := build(BGPFilterRule{Match: BGPFilterMatch{Prefix: "10.9.0.0/24"}, Action: FilterDeny})
		b := edgeOf(tp, "x").Routing.BGP
		b.Neighbors[0].InboundFilter = ""
		b.Neighbors[0].OutboundFilter = "f"
		yes := true
		b.Redistribute = &BGPRedistribute{OverlayPrefix: &yes}
		addLAN(tp, "y", LANRoute{ID: "q", Prefix: "10.9.0.0/24", Protocol: ProtoOSPF, Via: "ge3", OSPFType: OSPFIntra})
		rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
		for _, a := range rs.Edges[0].LANAdvertisements {
			if a.Protocol == ProtoBGP && a.Neighbor == "n1" && a.Prefix == "10.9.0.0/24" && a.Advertised {
				t.Errorf("outbound deny ignored: %+v", a)
			}
		}
	})
}

func TestConditionsChangeRoutes(t *testing.T) {
	tp := rdemo(t)
	as := DefaultRoutingAssumptions()
	base := computeR(t, tp, Conditions{}, as)
	if rows := routesFor(t, base, "c-edge", "10.40.0.0/24"); rows[0].Origin != "a-edge" {
		t.Fatalf("A (OSPF) is the preferred exit: %+v", rows)
	}
	t.Run("a link down makes the first exit unreachable and the second takes over", func(t *testing.T) {
		rs := computeR(t, tp, Conditions{LinksDown: []string{"a-inet"}}, as)
		rows := routesFor(t, rs, "c-edge", "10.40.0.0/24")
		if rows[0].Origin != "b-edge" || !rows[0].Selected || rows[1].Reachable || rows[1].ReachableCode != CodePRUnreachable {
			t.Errorf("%+v", rows)
		}
	})
	t.Run("a neighbor down removes what was learned through it", func(t *testing.T) {
		rs := computeR(t, tp, Conditions{NeighborsDown: []string{"a-edge/ge3"}}, as)
		if len(routesFor(t, rs, "c-edge", "10.20.0.0/24")) != 0 {
			t.Error("route survived the neighbor")
		}
	})
	t.Run("a failed ICMP probe makes the static route unreachable", func(t *testing.T) {
		rs := computeR(t, tp, Conditions{ProbesDown: []string{"c-edge/printers"}}, as)
		r := routesFor(t, rs, "c-edge", "172.16.50.0/24")[0]
		if r.Reachable || r.ReachableCode != CodeSLAStateDown || r.Selected {
			t.Errorf("%+v", r)
		}
		if r2 := routesFor(t, rs, "a-edge", "172.16.50.0/24")[0]; r2.Reachable {
			t.Errorf("remote copy should be unreachable too: %+v", r2)
		}
	})
}

func TestNSDRoutes(t *testing.T) {
	tp := loadContentTopologies(t)["hub-and-spoke-demo.yaml"]
	for i := range tp.NSDs {
		if tp.NSDs[i].ID == "aws-vpn" {
			tp.NSDs[i].Prefixes = []NSDPrefix{{Prefix: "10.77.0.0/24", Advertise: true}}
		}
	}
	rs := computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())
	local := routesFor(t, rs, "spoke-2-edge", "10.77.0.0/24")
	if len(local) != 1 || local[0].Source != SrcNSDLocal {
		t.Errorf("hosting Edge: %+v", local)
	}
	remote := routesFor(t, rs, "spoke-1-edge", "10.77.0.0/24")
	if len(remote) != 1 || remote[0].Source != SrcNSDRemote {
		t.Errorf("other Edge: %+v", remote)
	}
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDeterminismAndNoMutation(t *testing.T) {
	tp := rdemo(t)
	before := jsonOf(t, tp)
	as := DefaultRoutingAssumptions()
	as.BGPOverlayPrefix = true
	c := Conditions{LinksDown: []string{"a-inet"}, NeighborsDown: []string{"b-edge/lan-r1"}}
	a := jsonOf(t, computeR(t, tp, c, as))
	b := jsonOf(t, computeR(t, tp, c, as))
	if a != b {
		t.Error("ComputeRoutes is not deterministic")
	}
	q := ReachQuery{Source: "c-edge", Destination: "10.20.0.5"}
	r1, _ := Reachability(tp, c, as, q, nil)
	r2, _ := Reachability(tp, c, as, q, nil)
	if jsonOf(t, r1) != jsonOf(t, r2) {
		t.Error("Reachability is not deterministic")
	}
	if jsonOf(t, tp) != before {
		t.Error("conditions or assumptions changed the topology")
	}
}

// The result must not depend on the order routes are listed in.
func TestSelectionIndependentOfLearningOrder(t *testing.T) {
	selected := func(rs *RouteSet) map[string]string {
		out := map[string]string{}
		for _, e := range rs.Edges {
			for _, st := range e.Segments {
				for _, r := range st.Routes {
					if r.Selected {
						out[e.Edge+"|"+r.Prefix] = r.Source + "|" + r.RouteType + "|" + r.Origin + "|" + r.Via
					}
				}
			}
		}
		return out
	}
	want := selected(computeR(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions()))
	for seed := int64(1); seed <= 8; seed++ {
		tp := rdemo(t)
		rng := rand.New(rand.NewSource(seed))
		for si := range tp.Sites {
			for ei := range tp.Sites[si].Edges {
				if r := tp.Sites[si].Edges[ei].Routing; r != nil {
					rng.Shuffle(len(r.LANRoutes), func(i, j int) { r.LANRoutes[i], r.LANRoutes[j] = r.LANRoutes[j], r.LANRoutes[i] })
					rng.Shuffle(len(r.StaticRoutes), func(i, j int) { r.StaticRoutes[i], r.StaticRoutes[j] = r.StaticRoutes[j], r.StaticRoutes[i] })
				}
			}
		}
		if got := selected(computeR(t, tp, Conditions{}, DefaultRoutingAssumptions())); !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d changed the selection:\n%v\n%v", seed, got, want)
		}
	}
}

// Route ids are unique within an Edge (the UI uses them as keys) and a remote route shows no cost.
func TestRouteIDsAreUniqueAndRemoteRoutesShowNoCost(t *testing.T) {
	rs := computeR(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions())
	for _, e := range rs.Edges {
		seen := map[string]bool{}
		for _, st := range e.Segments {
			for _, r := range st.Routes {
				if r.ID == "" || seen[r.ID] {
					t.Errorf("%s: bad or duplicate route id %q", e.Edge, r.ID)
				}
				seen[r.ID] = true
				if r.Origin != "" && r.Cost != nil {
					t.Errorf("%s: remote route %s shows cost %d", e.Edge, r.Prefix, *r.Cost)
				}
			}
		}
	}
}
