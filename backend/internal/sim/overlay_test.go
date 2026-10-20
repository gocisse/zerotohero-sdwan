package sim

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// demo loads a fresh copy of the shipped hub-and-spoke-demo topology.
func demo(t *testing.T) *Topology {
	t.Helper()
	tp := loadContentTopologies(t)["hub-and-spoke-demo.yaml"]
	if tp == nil {
		t.Fatal("hub-and-spoke-demo.yaml missing")
	}
	cp, err := clone(tp)
	if err != nil {
		t.Fatal(err)
	}
	return cp
}

func fakeClock() *FakeClock { return &FakeClock{T: time.Unix(1_000_000, 0).UTC()} }

func compute(t *testing.T, tp *Topology, c Conditions) *TunnelSet {
	t.Helper()
	ts, err := ComputeTunnels(tp, c, fakeClock())
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func find(ts *TunnelSet, id string) *Tunnel {
	for i := range ts.Tunnels {
		if ts.Tunnels[i].ID == id {
			return &ts.Tunnels[i]
		}
	}
	return nil
}

func mustFind(t *testing.T, ts *TunnelSet, id string) *Tunnel {
	t.Helper()
	tn := find(ts, id)
	if tn == nil {
		t.Fatalf("tunnel %s not found", id)
	}
	return tn
}

func TestGatewayTunnelsFormForEveryEdge(t *testing.T) {
	tp := demo(t)
	ts := compute(t, tp, Conditions{})
	for _, s := range tp.Sites {
		for _, e := range s.Edges {
			for _, g := range []string{"gw-1", "gw-2"} {
				tn := mustFind(t, ts, "gateway:"+e.ID+":"+g)
				if tn.State != StUp || tn.Mode != "static" || tn.Basis.Status != StatusVerified || tn.Basis.Ref != "tunnel-types" {
					t.Errorf("%s: %+v", tn.ID, tn)
				}
				wan := 0
				for _, in := range e.Interfaces {
					if in.Kind == IfWan {
						wan++
					}
				}
				if len(tn.Paths) != wan {
					t.Errorf("%s: %d paths for %d WAN links", tn.ID, len(tn.Paths), wan)
				}
			}
		}
	}
	// A private link reaches the Gateway's private address, a public link its public address.
	tn := mustFind(t, ts, "gateway:spoke-1-edge:gw-1")
	if tn.Paths[0].Remote != "Gateway public address" || tn.Paths[1].Remote != "Gateway private address" {
		t.Errorf("gateway addresses: %+v", tn.Paths)
	}
	if !hasAssumption(tn.Also, 44) {
		t.Errorf("derived Gateway assignment must show assumption 44: %+v", tn.Also)
	}
}

func hasAssumption(list []Basis, n int) bool {
	for _, b := range list {
		if b.Status == StatusAssumption && b.Ref == "assumption "+itoa(n) {
			return true
		}
	}
	return false
}

func TestExplicitGatewayAssignment(t *testing.T) {
	tp := demo(t)
	tp.Sites[0].Edges[0].Gateways = &GatewayAssignment{Primary: "gw-2", Secondary: "gw-1"}
	ts := compute(t, tp, Conditions{})
	tn := mustFind(t, ts, "gateway:plain-1-edge:gw-2")
	if tn.Role != "primary" || hasAssumption(tn.Also, 44) {
		t.Errorf("an explicit assignment is verified geolocation assignment, not a derivation: %+v", tn)
	}
}

func TestNoCloudVPNMeansNoHubAndNoDynamicTunnel(t *testing.T) {
	ts := compute(t, demo(t), Conditions{})
	for _, tn := range ts.Tunnels {
		if tn.Type == TunHub && tn.A.ID == "plain-1-edge" {
			t.Errorf("a Profile without Cloud VPN gets no Hub tunnel: %s", tn.ID)
		}
	}
	d := mustFind(t, ts, "dynamic:plain-1-edge:mesh-1-edge")
	if d.State != StNotEstablished || d.ReasonCode != CodeCloudVPNOff {
		t.Errorf("%+v", d)
	}
}

func TestHubTunnelsFormInHubOrder(t *testing.T) {
	ts := compute(t, demo(t), Conditions{})
	p := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge")
	s := mustFind(t, ts, "hub:spoke-1-edge:hub-s-edge")
	if p.HubIndex != 1 || p.Role != "primary" || s.HubIndex != 2 || s.Role != "secondary" {
		t.Errorf("Hub order: %+v %+v", p, s)
	}
	if p.State != StUp || p.Mode != "static" || p.Basis.Rule != RB2HPermanent {
		t.Errorf("%+v", p)
	}
	// Public pairs with public, private with private: 2 WAN links each side, one of each type.
	if len(p.Paths) != 2 || p.Paths[0].Remote != "hp-inet" || p.Paths[1].Remote != "hp-mpls" {
		t.Errorf("paths: %+v", p.Paths)
	}
	if find(ts, "hub:spoke-3-edge:hub-p-edge") == nil {
		t.Error("every spoke of the Profile builds a tunnel to every Hub")
	}
	if find(ts, "hub:hub-p-edge:hub-s-edge") != nil {
		t.Error("a Hub Edge is in the Hub Profile and does not tunnel to the other Hub")
	}
}

func TestHubTunnelPrerequisites(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Topology)
		code   string
	}{
		{"Cloud VPN has no segment", func(x *Topology) { x.Profiles[2].CloudVPN.Segments = nil }, CodeCloudVPNOff},
		{"Enable Branch to Hubs is off", func(x *Topology) { x.Profiles[2].CloudVPN.BranchToHub.Enabled = false }, CodeB2HOff},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := demo(t)
			c.mutate(tp)
			tn := mustFind(t, compute(t, tp, Conditions{}), "hub:spoke-1-edge:hub-p-edge")
			if tn.State != StNotEstablished || tn.ReasonCode != c.code {
				t.Errorf("want not-established/%s, got %s/%s: %s", c.code, tn.State, tn.ReasonCode, tn.Reason)
			}
		})
	}
	tp := demo(t)
	tp.Profiles[2].CloudVPN.Segments = nil
	tn := mustFind(t, compute(t, tp, Conditions{}), "hub:spoke-1-edge:hub-p-edge")
	if !hasAssumption(tn.Also, 42) || len(tn.NotModelled) == 0 {
		t.Errorf("the Cloud VPN default is not documented: assumption 42 and a not-modelled marker are required: %+v", tn)
	}
}

func TestClusterAssignmentByLogicalID(t *testing.T) {
	cases := []struct {
		logical int
		member  string
	}{{7, "hub-c2"}, {6, "hub-c1"}, {0, "hub-c1"}, {1, "hub-c2"}}
	for _, c := range cases {
		tp := demo(t)
		for si := range tp.Sites {
			for ei := range tp.Sites[si].Edges {
				if tp.Sites[si].Edges[ei].ID == "cs-1-edge" {
					l := c.logical
					tp.Sites[si].Edges[ei].LogicalID = &l
				}
			}
		}
		ts := compute(t, tp, Conditions{})
		if find(ts, "hub:cs-1-edge:"+c.member) == nil {
			t.Errorf("logical id %d should map to %s", c.logical, c.member)
		}
		other := "hub-c1"
		if c.member == "hub-c1" {
			other = "hub-c2"
		}
		if find(ts, "hub:cs-1-edge:"+other) != nil {
			t.Errorf("a spoke tunnels to exactly one member of the cluster (logical id %d)", c.logical)
		}
		if len(ts.ClusterAssignments) != 1 || ts.ClusterAssignments[0].Member != c.member || ts.ClusterAssignments[0].Basis.Rule != RClusterAssign {
			t.Errorf("assignment: %+v", ts.ClusterAssignments)
		}
	}
}

func TestClusterMemberDoesNotTunnelToItsOwnCluster(t *testing.T) {
	tp := demo(t)
	// Make a cluster member a spoke of its own cluster: it must be skipped.
	for i := range tp.Profiles {
		if tp.Profiles[i].ID == "hub" {
			tp.Profiles[i].CloudVPN = &CloudVPN{Segments: []string{"global"}, BranchToHub: &BranchToHub{Enabled: true, Hubs: []string{"hub-cluster"}}}
		}
	}
	ts := compute(t, tp, Conditions{})
	for _, tn := range ts.Tunnels {
		if tn.Type == TunHub && (tn.A.ID == "hub-c1" || tn.A.ID == "hub-c2") && strings.HasPrefix(tn.B.ID, "hub-c") {
			t.Errorf("%s: a cluster member does not tunnel to its own cluster", tn.ID)
		}
	}
}

func TestDynamicTunnelLifecycleBoundary(t *testing.T) {
	tp := demo(t)
	cases := []struct {
		name  string
		last  float64
		state string
		code  string
	}{
		{"no traffic", -1, StNotEstablished, CodeNoTraffic},
		{"traffic now", 0, StUp, CodeOK},
		{"179 s of silence keeps the tunnel", 179, StUp, CodeOK},
		{"180 s of silence tears it down", 180, StTornDown, CodeTornDown},
		{"600 s of silence", 600, StTornDown, CodeTornDown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cond := Conditions{}
			if c.last >= 0 {
				cond.Traffic = []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: c.last + 100, LastPacketAgoS: c.last}}
			}
			tn := mustFind(t, compute(t, tp, cond), "dynamic:mesh-1-edge:mesh-2-edge")
			if tn.State != c.state || tn.ReasonCode != c.code {
				t.Errorf("got %s/%s: %s", tn.State, tn.ReasonCode, tn.Reason)
			}
			if tn.Basis.Rule != RDynFirstPacket || !hasAssumption(tn.Also, 54) {
				t.Errorf("basis: %+v %+v", tn.Basis, tn.Also)
			}
		})
	}
}

func TestDynamicTunnelBuildConditions(t *testing.T) {
	tp := demo(t)
	traffic := func(a, b string) []TrafficSpec {
		return []TrafficSpec{{A: a, B: b, FirstPacketAgoS: 10, LastPacketAgoS: 1}}
	}
	// NAT in front of the responder blocks the direct tunnel.
	tn := mustFind(t, compute(t, tp, Conditions{Traffic: traffic("spoke-1-edge", "spoke-3-edge")}), "dynamic:spoke-1-edge:spoke-3-edge")
	if tn.State != StNotEstablished || tn.ReasonCode != CodeNATBlocks || !hasAssumption(tn.Also, 16) {
		t.Errorf("NAT: %s/%s %+v", tn.State, tn.ReasonCode, tn.Also)
	}
	hasNM := false
	for _, nm := range tn.NotModelled {
		hasNM = hasNM || nm.Rule == RNMSecureTunnel
	}
	if !hasNM {
		t.Error("the secure-tunnel decision must carry its not-modelled marker")
	}
	// The stub can be forced either way (a teaching switch).
	tn = mustFind(t, compute(t, tp, Conditions{DynamicBuild: BuildAlways, Traffic: traffic("spoke-1-edge", "spoke-3-edge")}), "dynamic:spoke-1-edge:spoke-3-edge")
	if tn.State != StUp {
		t.Errorf("always: %s", tn.State)
	}
	tn = mustFind(t, compute(t, tp, Conditions{DynamicBuild: BuildNever, Traffic: traffic("mesh-1-edge", "mesh-2-edge")}), "dynamic:mesh-1-edge:mesh-2-edge")
	if tn.State != StNotEstablished || tn.ReasonCode != CodeBuildNever {
		t.Errorf("never: %s/%s", tn.State, tn.ReasonCode)
	}
	// The responder decides: if the NAT Edge is the initiator the direct tunnel can still be built.
	tn = mustFind(t, compute(t, tp, Conditions{Traffic: traffic("spoke-3-edge", "spoke-1-edge")}), "dynamic:spoke-1-edge:spoke-3-edge")
	if tn.State != StUp {
		t.Errorf("initiator behind NAT: %s/%s", tn.State, tn.ReasonCode)
	}
}

func TestDynamicTunnelPrerequisites(t *testing.T) {
	traffic := []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 10, LastPacketAgoS: 1}}
	cases := []struct {
		name   string
		mutate func(*Topology)
		code   string
	}{
		{"Cloud VPN off", func(x *Topology) { x.Profiles[1].CloudVPN.Segments = nil }, CodeCloudVPNOff},
		{"Branch to Branch off", func(x *Topology) { x.Profiles[1].CloudVPN.BranchToBranch.Enabled = false }, CodeB2BOff},
		{"Dynamic off", func(x *Topology) { f := false; x.Profiles[1].CloudVPN.BranchToBranch.Dynamic = &f }, CodeDynamicOff},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := demo(t)
			c.mutate(tp)
			tn := mustFind(t, compute(t, tp, Conditions{Traffic: traffic}), "dynamic:mesh-1-edge:mesh-2-edge")
			if tn.State != StNotEstablished || tn.ReasonCode != c.code {
				t.Errorf("got %s/%s", tn.State, tn.ReasonCode)
			}
		})
	}
	// Enabled at only one of two sites: documented as unknown.
	tp := demo(t)
	tp.Sites[2].Edges[0].Overrides.CloudVPN = &CloudVPN{BranchToBranch: &BranchToBranch{Enabled: false}}
	tn := mustFind(t, compute(t, tp, Conditions{Traffic: traffic}), "dynamic:mesh-1-edge:mesh-2-edge")
	if tn.ReasonCode != CodeOneSiteOnly || len(tn.NotModelled) == 0 {
		t.Errorf("one site only must be not modelled: %+v", tn)
	}
}

func TestDynamicDefaultIsOn(t *testing.T) {
	tp := demo(t)
	if tp.Profiles[1].CloudVPN.BranchToBranch.Dynamic != nil {
		t.Fatal("the mesh Profile leaves Dynamic Branch to Branch unset to use the documented default")
	}
	tn := mustFind(t, compute(t, tp, Conditions{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 5, LastPacketAgoS: 1}}}), "dynamic:mesh-1-edge:mesh-2-edge")
	if tn.State != StUp {
		t.Errorf("default on: %s", tn.State)
	}
}

func TestNSDTunnelsAndCounts(t *testing.T) {
	ts := compute(t, demo(t), Conditions{})
	a1 := mustFind(t, ts, "nsd-gateway:azure-vhub:1")
	a2 := mustFind(t, ts, "nsd-gateway:azure-vhub:2")
	if a1.Role != "active" || a2.Role != "standby" || a2.ReasonCode != CodeStandby || !hasAssumption(a2.Also, 49) {
		t.Errorf("hot standby: %+v %+v", a1, a2)
	}
	if find(ts, "nsd-gateway:azure-vhub:3") != nil {
		t.Error("Azure Virtual Hub allows at most 2 tunnels")
	}
	e := mustFind(t, ts, "nsd-edge:spoke-2-edge:aws-vpn")
	if e.State != StUp || e.Type != TunNSDEdge || len(e.Paths) != 1 || e.Paths[0].LocalLink != "s2-inet" {
		t.Errorf("%+v", e)
	}
	caps := []struct {
		typ, mode string
		want      int
	}{{"generic_ikev2", ModeActiveActive, 4}, {"generic_ikev2", ModeHotStandby, 2}, {"cisco_asa", ModeActiveActive, 1}, {"generic_firewall", ModeHotStandby, 1}, {"azure_virtual_hub", ModeActiveActive, 2}}
	for _, c := range caps {
		if got := NSDTunnelCap(c.typ, c.mode); got != c.want {
			t.Errorf("%s/%s cap = %d, want %d", c.typ, c.mode, got, c.want)
		}
	}
	tp := demo(t)
	tp.NSDs[0].Tunnels = 9 // an invalid authored count is clamped by the engine to the documented cap
	if n := countType(compute(t, tp, Conditions{}), TunNSDGateway); n != 2 {
		t.Errorf("clamped to the cap, got %d tunnels", n)
	}
}

func countType(ts *TunnelSet, typ string) int {
	n := 0
	for _, t := range ts.Tunnels {
		if t.Type == typ {
			n++
		}
	}
	return n
}

func TestNSDPrerequisites(t *testing.T) {
	tp := demo(t)
	tp.Profiles[2].CloudVPN.NSDViaGateway = nil
	ts := compute(t, tp, Conditions{})
	if tn := mustFind(t, ts, "nsd-gateway:azure-vhub:1"); tn.State != StNotEstablished || tn.ReasonCode != CodeNSDNotEnabled {
		t.Errorf("not enabled: %+v", tn)
	}
	tp = demo(t)
	tp.Profiles[2].CloudVPN.NSDViaEdge = nil
	if tn := mustFind(t, compute(t, tp, Conditions{}), "nsd-edge:spoke-2-edge:aws-vpn"); tn.State != StNotEstablished {
		t.Errorf("via Edge not enabled: %+v", tn)
	}
	tp = demo(t)
	tp.Profiles[2].CloudVPN.Segments = nil
	if tn := mustFind(t, compute(t, tp, Conditions{}), "nsd-gateway:azure-vhub:1"); tn.State != StNotEstablished {
		t.Errorf("via Gateway needs Cloud VPN: %+v", tn)
	}
}

func TestTunnelCapOf16(t *testing.T) {
	tp := demo(t)
	// Give one Edge five public links and its peer four: 20 pairs, 16 allowed.
	addLinks := func(edge string, prefix string, n int) {
		for si := range tp.Sites {
			for ei := range tp.Sites[si].Edges {
				e := &tp.Sites[si].Edges[ei]
				if e.ID != edge {
					continue
				}
				e.Interfaces = nil
				for k := 0; k < n; k++ {
					id := prefix + itoa(k)
					tp.Links = append(tp.Links, WanLink{ID: id, Type: LinkPublic, Medium: MediumWired, UpMbps: 1, DownMbps: 1})
					e.Interfaces = append(e.Interfaces, Interface{ID: "ge" + itoa(k), Kind: IfWan, Link: id})
				}
			}
		}
	}
	addLinks("mesh-1-edge", "x", 5)
	addLinks("mesh-2-edge", "y", 4)
	ts := compute(t, tp, Conditions{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 5, LastPacketAgoS: 1}}})
	tn := mustFind(t, ts, "dynamic:mesh-1-edge:mesh-2-edge")
	if len(tn.Paths) != MaxEdgeTunnels || len(tn.Notes) == 0 || tn.Notes[0].Basis.Rule != RTunnelCap {
		t.Errorf("cap: %d paths, notes %+v", len(tn.Paths), tn.Notes)
	}
	if w := warningsContaining(Validate(tp), "exceed the cap"); len(w) == 0 {
		t.Error("the validator should warn about the cap")
	}
}

func warningsContaining(issues []Issue, s string) []Issue {
	var out []Issue
	for _, i := range issues {
		if i.Level == LevelWarning && strings.Contains(i.Message, s) {
			out = append(out, i)
		}
	}
	return out
}

func TestLinkConditionsChangePathStates(t *testing.T) {
	tp := demo(t)
	ts := compute(t, tp, Conditions{LinksDown: []string{"s1-inet"}})
	tn := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge")
	if tn.State != StUp || tn.Paths[0].State != StDown || tn.Paths[1].State != StUp {
		t.Errorf("one path down, tunnel still up: %+v", tn)
	}
	ts = compute(t, tp, Conditions{LinksDegraded: []string{"s1-inet", "s1-mpls"}})
	if tn := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge"); tn.State != StDegraded {
		t.Errorf("all paths degraded: %s", tn.State)
	}
	ts = compute(t, tp, Conditions{LinksDown: []string{"s1-inet", "s1-mpls"}})
	if tn := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge"); tn.State != StDown || tn.ReasonCode != CodeLinkDown {
		t.Errorf("all paths down: %+v", tn)
	}
	// A remote link down also takes its pair down.
	ts = compute(t, tp, Conditions{LinksDown: []string{"hp-inet"}})
	if tn := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge"); tn.Paths[0].State != StDown {
		t.Errorf("the pair is as healthy as the worse link: %+v", tn.Paths)
	}
}

func TestHubAndGatewayUnreachable(t *testing.T) {
	tp := demo(t)
	ts := compute(t, tp, Conditions{HubsUnreachable: []string{"hub-p-edge"}})
	if tn := mustFind(t, ts, "hub:spoke-1-edge:hub-p-edge"); tn.State != StDown || tn.ReasonCode != CodePeerUnreachable {
		t.Errorf("%+v", tn)
	}
	if tn := mustFind(t, ts, "hub:spoke-1-edge:hub-s-edge"); tn.State != StUp {
		t.Errorf("the secondary Hub is unaffected: %+v", tn)
	}
	ts = compute(t, tp, Conditions{GatewaysUnreachable: []string{"gw-1"}})
	if tn := mustFind(t, ts, "gateway:spoke-1-edge:gw-1"); tn.State != StDown || tn.ReasonCode != CodePeerUnreachable {
		t.Errorf("%+v", tn)
	}
	if tn := mustFind(t, ts, "nsd-gateway:azure-vhub:1"); tn.State != StDown {
		t.Errorf("the Gateway that builds the NSD tunnels is down: %+v", tn)
	}
	ts = compute(t, tp, Conditions{NSDsUnreachable: []string{"aws-vpn"}})
	if tn := mustFind(t, ts, "nsd-edge:spoke-2-edge:aws-vpn"); tn.State != StDown {
		t.Errorf("%+v", tn)
	}
}

func TestOrchestratorUnreachableIsNotModelled(t *testing.T) {
	ts := compute(t, demo(t), Conditions{OrchestratorUnreachable: true})
	for _, tn := range ts.Tunnels {
		if tn.State != StUnknown || tn.ReasonCode != CodeOrchestratorDown || len(tn.NotModelled) == 0 {
			t.Fatalf("%s: %+v", tn.ID, tn)
		}
	}
	found := false
	for _, nm := range ts.NotModelled {
		found = found || nm.Rule == RNMOrchestratorDown
	}
	if !found {
		t.Error("the result must carry the not-modelled marker")
	}
	d, err := SelectPath(demo(t), Conditions{OrchestratorUnreachable: true}, Flow{Source: "spoke-1-edge", Destination: FlowDest{Kind: "edge", ID: "spoke-2-edge"}}, fakeClock())
	if err != nil {
		t.Fatal(err)
	}
	if d.Chosen != nil || !strings.Contains(d.Why, "not modelled") {
		t.Errorf("no path may be invented: %+v", d)
	}
}

func TestConditionsLeaveTheTopologyUnchanged(t *testing.T) {
	tp := demo(t)
	before, _ := json.Marshal(tp)
	_, _ = ComputeTunnels(tp, Conditions{LinksDown: []string{"s1-inet"}, HubsUnreachable: []string{"hub-p-edge"}, DynamicBuild: BuildNever,
		Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 5, LastPacketAgoS: 1}}}, fakeClock())
	_, _ = SelectPath(tp, Conditions{LinksDown: []string{"s1-inet"}}, Flow{Source: "spoke-1-edge", Destination: FlowDest{Kind: "edge", ID: "spoke-2-edge"}}, fakeClock())
	_, _ = Timeline(tp, Conditions{}, []Event{{AtS: 1, Type: EvLinkDown, Target: "s1-inet"}, {AtS: 2, Type: EvHubDown, Target: "hub-p-edge"}})
	after, _ := json.Marshal(tp)
	if string(before) != string(after) {
		t.Fatal("conditions, paths and timelines must never modify the topology")
	}
}

func TestInvalidConditionsAreReported(t *testing.T) {
	bad := []Conditions{
		{LinksDown: []string{"nope"}}, {HubsUnreachable: []string{"nope"}}, {GatewaysUnreachable: []string{"nope"}},
		{NSDsUnreachable: []string{"nope"}}, {DynamicBuild: "maybe"},
		{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-1-edge"}}},
		{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 1, LastPacketAgoS: 5}}},
	}
	for _, c := range bad {
		_, err := ComputeTunnels(demo(t), c, fakeClock())
		var ce *ConditionError
		if err == nil || !asCondErr(err, &ce) {
			t.Errorf("%+v should be a ConditionError, got %v", c, err)
		}
	}
}

func asCondErr(err error, target **ConditionError) bool {
	ce, ok := err.(*ConditionError)
	if ok {
		*target = ce
	}
	return ok
}

// Property-style checks.
func TestComputeTunnelsIsDeterministicAndSymmetric(t *testing.T) {
	tp := demo(t)
	cond := Conditions{LinksDegraded: []string{"m1-lte"}, Traffic: []TrafficSpec{{A: "mesh-2-edge", B: "mesh-1-edge", FirstPacketAgoS: 9, LastPacketAgoS: 2}}}
	a, _ := json.Marshal(compute(t, tp, cond))
	b, _ := json.Marshal(compute(t, tp, cond))
	if string(a) != string(b) {
		t.Fatal("ComputeTunnels must be deterministic")
	}
	ts := compute(t, tp, cond)
	seen := map[string]string{}
	for _, tn := range ts.Tunnels {
		if tn.Type != TunDynamic {
			continue
		}
		key := pairKey(tn.A.ID, tn.B.ID)
		if prev, dup := seen[key]; dup {
			t.Errorf("the pair %s has two dynamic tunnels: %s and %s (endpoints must be symmetric)", key, prev, tn.ID)
		}
		seen[key] = tn.ID
		if tn.A.ID > tn.B.ID && false {
			t.Error("unreachable")
		}
	}
	// A pair's tunnel is the same whichever Edge is named first in the traffic evidence.
	x := mustFind(t, compute(t, tp, Conditions{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 9, LastPacketAgoS: 2}}}), "dynamic:mesh-1-edge:mesh-2-edge")
	y := mustFind(t, compute(t, tp, Conditions{Traffic: []TrafficSpec{{A: "mesh-2-edge", B: "mesh-1-edge", FirstPacketAgoS: 9, LastPacketAgoS: 2}}}), "dynamic:mesh-1-edge:mesh-2-edge")
	if x.State != y.State {
		t.Errorf("states differ with the initiator: %s vs %s", x.State, y.State)
	}
}

func TestOverrideCloudVPNReplacesSubsections(t *testing.T) {
	tp := demo(t)
	// Spoke 2 overrides its Hub list: only the secondary Hub.
	for si := range tp.Sites {
		for ei := range tp.Sites[si].Edges {
			if tp.Sites[si].Edges[ei].ID == "spoke-2-edge" {
				tp.Sites[si].Edges[ei].Overrides.CloudVPN = &CloudVPN{BranchToHub: &BranchToHub{Enabled: true, Hubs: []string{"hub-s-edge"}}}
			}
		}
	}
	ts := compute(t, tp, Conditions{})
	if find(ts, "hub:spoke-2-edge:hub-p-edge") != nil {
		t.Error("the override replaces the Hub list")
	}
	tn := mustFind(t, ts, "hub:spoke-2-edge:hub-s-edge")
	if !hasAssumption(tn.Also, 43) || len(tn.NotModelled) == 0 {
		t.Errorf("override granularity is an assumption plus a not-modelled marker: %+v", tn)
	}
	if find(ts, "hub:spoke-1-edge:hub-p-edge") == nil {
		t.Error("other Edges still inherit the Profile")
	}
}
