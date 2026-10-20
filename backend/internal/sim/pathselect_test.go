package sim

import (
	"encoding/json"
	"strings"
	"testing"
)

func edgeFlow(src, dst, app string) Flow {
	return Flow{Source: src, Destination: FlowDest{Kind: "edge", ID: dst}, Application: app}
}

func decide(t *testing.T, tp *Topology, c Conditions, f Flow) *PathDecision {
	t.Helper()
	d, err := SelectPath(tp, c, f, fakeClock())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func optionByKind(d *PathDecision, kind string) []PathOption {
	var out []PathOption
	for _, o := range d.Options {
		if o.Kind == kind {
			out = append(out, o)
		}
	}
	return out
}

func hasNM(d *PathDecision, rule string) bool {
	for _, n := range d.NotModelled {
		if n.Rule == rule {
			return true
		}
	}
	return false
}

func TestEveryDecisionStatesThatReachabilityIsAssumed(t *testing.T) {
	d := decide(t, demo(t), Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if !d.Reachability.Assumed || d.Reachability.NotModelled.Basis.Status != StatusNotModelled || !hasNM(d, RNMRoutingPrecedence) {
		t.Errorf("reachability: %+v", d.Reachability)
	}
}

func TestHubsForVPNStandingPathUsesHubsInOrder(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptViaHub || d.Chosen.Via != "hub-p-edge" {
		t.Fatalf("chosen: %+v", d.Chosen)
	}
	hubs := optionByKind(d, OptViaHub)
	if len(hubs) != 2 || !hubs[0].Chosen || hubs[1].Chosen || !hubs[1].Usable {
		t.Errorf("the secondary Hub is usable but not chosen while the primary is: %+v", hubs)
	}
	if hasOption(d, OptViaGateway) {
		t.Error("Hubs for VPN does not use the Gateway path (assumption 61)")
	}
	// The direct tunnel option is listed first and rejected because no traffic has flowed.
	if d.Options[0].Kind != OptDirect || d.Options[0].Usable || d.Options[0].ReasonCode != CodeNoTraffic {
		t.Errorf("direct option: %+v", d.Options[0])
	}
}

func hasOption(d *PathDecision, kind string) bool { return len(optionByKind(d, kind)) > 0 }

func TestHubFailoverToSecondaryAndNoPath(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{HubsUnreachable: []string{"hub-p-edge"}}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Via != "hub-s-edge" {
		t.Fatalf("the Spoke moves to the secondary Hub: %+v", d.Chosen)
	}
	primary := optionByKind(d, OptViaHub)[0]
	if primary.Usable || primary.ReasonCode != CodePeerUnreachable || !strings.Contains(primary.Reason, "unreachable") {
		t.Errorf("rejected primary must say why: %+v", primary)
	}
	if !hasBasisRule(d.Chosen.Also, RHubFailover) || !hasNM(d, RNMHubReturnLocal) {
		t.Errorf("failover is verified; moving back is not modelled: %+v", d.Chosen.Also)
	}
	d = decide(t, tp, Conditions{HubsUnreachable: []string{"hub-p-edge", "hub-s-edge"}}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if d.Chosen != nil || d.Why == "" {
		t.Errorf("no Hub is reachable, so no path: %+v", d.Chosen)
	}
}

func hasBasisRule(list []Basis, rule string) bool {
	for _, b := range list {
		if b.Rule == rule {
			return true
		}
	}
	return false
}

func TestDirectTunnelPreferredWhenItExists(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{Traffic: []TrafficSpec{{A: "spoke-1-edge", B: "spoke-2-edge", FirstPacketAgoS: 30, LastPacketAgoS: 1}}}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptDirect || !hasBasisRule(d.Chosen.Also, RA14DirectFirst) || !hasNM(d, RNMDirectVsHub) {
		t.Fatalf("direct tunnel: %+v", d.Chosen)
	}
	// After 180 s of silence the tunnel is gone and the flow is back on the Hub path.
	d = decide(t, tp, Conditions{Traffic: []TrafficSpec{{A: "spoke-1-edge", B: "spoke-2-edge", FirstPacketAgoS: 300, LastPacketAgoS: 180}}}, edgeFlow("spoke-1-edge", "spoke-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptViaHub || d.Options[0].ReasonCode != CodeTornDown {
		t.Errorf("torn down: %+v", d.Options[0])
	}
	// NAT keeps the direct tunnel from forming: the Hub path remains, with the reason.
	d = decide(t, tp, Conditions{Traffic: []TrafficSpec{{A: "spoke-1-edge", B: "spoke-3-edge", FirstPacketAgoS: 30, LastPacketAgoS: 1}}}, edgeFlow("spoke-1-edge", "spoke-3-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptViaHub || d.Options[0].ReasonCode != CodeNATBlocks {
		t.Errorf("NAT: %+v / %+v", d.Chosen, d.Options[0])
	}
}

func TestStandingPathViaCloudGateways(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("mesh-1-edge", "mesh-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptViaGateway || d.Chosen.Via != "gw-1" || hasOption(d, OptViaHub) {
		t.Fatalf("%+v", d.Chosen)
	}
	d = decide(t, tp, Conditions{GatewaysUnreachable: []string{"gw-1"}}, edgeFlow("mesh-1-edge", "mesh-2-edge", ""))
	if d.Chosen == nil || d.Chosen.Via != "gw-2" {
		t.Errorf("the secondary Gateway carries the flow: %+v", d.Chosen)
	}
	d = decide(t, tp, Conditions{GatewaysUnreachable: []string{"gw-1", "gw-2"}}, edgeFlow("mesh-1-edge", "mesh-2-edge", ""))
	if d.Chosen != nil {
		t.Errorf("no Gateway, no path: %+v", d.Chosen)
	}
}

func TestNoCloudVPNMeansNoBranchToBranchPath(t *testing.T) {
	d := decide(t, demo(t), Conditions{}, edgeFlow("plain-1-edge", "mesh-1-edge", ""))
	if d.Chosen != nil {
		t.Fatalf("no Cloud VPN: %+v", d.Chosen)
	}
	for _, o := range d.Options {
		if o.Usable || (o.ReasonCode != CodeCloudVPNOff && o.ReasonCode != CodeNoLinkPair && o.ReasonCode != CodeNoTraffic) {
			t.Errorf("option %s: %s", o.Label, o.ReasonCode)
		}
	}
}

func TestHubTunnelToTheHubItselfAndCluster(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "hub-p-edge", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptHubTunnel || !hasBasisRule(d.Chosen.Also, RA60HubTunnelDirect) {
		t.Errorf("a flow to the Hub uses the static Hub tunnel: %+v", d.Chosen)
	}
	d = decide(t, tp, Conditions{}, edgeFlow("cs-1-edge", "hub-c2", ""))
	if d.Chosen == nil || d.Chosen.Kind != OptHubTunnel {
		t.Errorf("assigned cluster member: %+v", d.Chosen)
	}
	d = decide(t, tp, Conditions{}, edgeFlow("cs-1-edge", "hub-c1", ""))
	if d.Chosen != nil && d.Chosen.Kind == OptHubTunnel {
		t.Error("cs-1 is assigned to hub-c2, not hub-c1")
	}
}

func TestNSDPaths(t *testing.T) {
	tp := demo(t)
	d, err := SelectPath(tp, Conditions{}, Flow{Source: "spoke-1-edge", Destination: FlowDest{Kind: "nsd", ID: "azure-vhub"}}, fakeClock())
	if err != nil || d.Chosen == nil || d.Chosen.Kind != OptNSDGateway || !hasBasisRule(d.Chosen.Also, RA15NSDByPolicy) || !hasNM(d, RNMNSDMTU) || d.MTU != nil {
		t.Fatalf("%v %+v", err, d.Chosen)
	}
	// Via Edge: only the Edge that builds the tunnel can use it without routing.
	d, _ = SelectPath(tp, Conditions{}, Flow{Source: "spoke-2-edge", Destination: FlowDest{Kind: "nsd", ID: "aws-vpn"}}, fakeClock())
	if d.Chosen == nil || d.Chosen.Kind != OptNSDEdge {
		t.Errorf("builder Edge: %+v", d.Chosen)
	}
	d, _ = SelectPath(tp, Conditions{}, Flow{Source: "spoke-1-edge", Destination: FlowDest{Kind: "nsd", ID: "aws-vpn"}}, fakeClock())
	if d.Chosen != nil || !strings.Contains(d.Options[0].Reason, "routing") {
		t.Errorf("another Edge: %+v", d.Options)
	}
	// Not enabled in the Profile.
	d, _ = SelectPath(tp, Conditions{}, Flow{Source: "mesh-1-edge", Destination: FlowDest{Kind: "nsd", ID: "azure-vhub"}}, fakeClock())
	if d.Chosen != nil || d.Options[0].ReasonCode != CodeNSDNotEnabled {
		t.Errorf("not enabled: %+v", d.Options[0])
	}
	// Destination down.
	d, _ = SelectPath(tp, Conditions{NSDsUnreachable: []string{"azure-vhub"}}, Flow{Source: "spoke-1-edge", Destination: FlowDest{Kind: "nsd", ID: "azure-vhub"}}, fakeClock())
	if d.Chosen != nil {
		t.Errorf("destination unreachable: %+v", d.Chosen)
	}
}

func TestDirectNetworkServiceIsNotAnOverlayFlow(t *testing.T) {
	tp := demo(t)
	tp.Profiles[2].BusinessRules = append(tp.Profiles[2].BusinessRules, BusinessRule{ID: "spoke-direct", Segment: "global", Match: RuleMatch{Application: "web"}, Action: RuleAction{NetworkService: NetDirect}})
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "web"))
	if d.Chosen != nil || len(d.Options) != 0 || !hasBasisRule(noteBases(d), RA63DirectNotOverlay) {
		t.Errorf("Direct: %+v", d)
	}
}

func noteBases(d *PathDecision) []Basis {
	var out []Basis
	for _, n := range d.Notes {
		out = append(out, n.Basis)
	}
	return out
}

func TestRuleMatchingFollowsAssumptions(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "backup"))
	if d.RuleID != "spoke-1-backup" || d.Steering == nil || d.Steering.Mode != SteerWanLink {
		t.Fatalf("Edge rule: %+v", d)
	}
	if !hasNM(d, RNMBizOrder) || !hasBasisRule(noteBases(d), RA1FirstMatch) {
		t.Error("rule order is not documented: assumption 1 and a not-modelled marker must show")
	}
	// An application with no rule: Smart Defaults treated as Multi-Path (assumption 2).
	d = decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "chat"))
	if d.RuleID != "" || d.NetworkService != NetMultipath || !hasBasisRule(noteBases(d), RA2NoRuleDefault) {
		t.Errorf("no rule: %+v", d)
	}
}

// Steering: every mode with every link state, against the documented semantics.
func TestSteeringModesAgainstLinkStates(t *testing.T) {
	// spoke-1: public link s1-inet and private link s1-mpls; first leg is the Hub tunnel to the primary Hub.
	type want struct {
		action string
		link   string // chosen local link ("" for none)
		nm     string // expected not-modelled rule, if any
	}
	cases := []struct {
		name string
		app  string
		cond Conditions
		want want
	}{
		// Auto (no rule): best usable link.
		{"auto, both up", "chat", Conditions{}, want{"use", "s1-inet", ""}},
		{"auto, public down", "chat", Conditions{LinksDown: []string{"s1-inet"}}, want{"use", "s1-mpls", ""}},
		{"auto, public degraded and private up", "chat", Conditions{LinksDegraded: []string{"s1-inet"}}, want{"use", "s1-mpls", ""}},
		{"auto, both degraded", "chat", Conditions{LinksDegraded: []string{"s1-inet", "s1-mpls"}}, want{"use", "s1-inet", ""}},
		{"auto, both down", "chat", Conditions{LinksDown: []string{"s1-inet", "s1-mpls"}}, want{"", "", ""}},
		// Mandatory on the public link (bulk: public_wired mandatory).
		{"mandatory, up", "bulk", Conditions{}, want{"use", "s1-inet", ""}},
		{"mandatory, degraded stays", "bulk", Conditions{LinksDegraded: []string{"s1-inet"}}, want{"use", "s1-inet", ""}},
		{"mandatory, down drops", "bulk", Conditions{LinksDown: []string{"s1-inet"}}, want{"drop", "", ""}},
		// Mandatory WAN link (Edge rule backup).
		{"mandatory wan link, down drops", "backup", Conditions{LinksDown: []string{"s1-inet"}}, want{"drop", "", ""}},
		// Available on the public group (video).
		{"available, up", "video", Conditions{}, want{"use", "s1-inet", ""}},
		{"available, degraded stays", "video", Conditions{LinksDegraded: []string{"s1-inet"}}, want{"use", "s1-inet", ""}},
		{"available, down moves", "video", Conditions{LinksDown: []string{"s1-inet"}}, want{"move", "s1-mpls", ""}},
		// Preferred on the private group (voice): a private preferred link is not modelled when it is not up.
		{"preferred private, up", "voice", Conditions{}, want{"use", "s1-mpls", ""}},
		{"preferred private, degraded", "voice", Conditions{LinksDegraded: []string{"s1-mpls"}}, want{"undetermined", "", RNMPreferredPrivate}},
		{"preferred private, down", "voice", Conditions{LinksDown: []string{"s1-mpls"}}, want{"undetermined", "", RNMPreferredPrivate}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := decide(t, demo(t), c.cond, edgeFlow("spoke-1-edge", "spoke-2-edge", c.app))
			if d.Chosen == nil {
				// The only way there is no chosen option here is both links down on the first leg.
				if c.want.action != "" {
					t.Fatalf("no option chosen: %+v", d.Options)
				}
				return
			}
			s := d.Steering
			if s == nil {
				t.Fatalf("no steering decision")
			}
			link := ""
			if s.Chosen != nil {
				link = s.Chosen.LocalLink
			}
			if s.Action != c.want.action || link != c.want.link {
				t.Errorf("action/link = %s/%s, want %s/%s: %s", s.Action, link, c.want.action, c.want.link, s.Reason)
			}
			if c.want.nm != "" && !hasNM(d, c.want.nm) {
				t.Errorf("missing not-modelled marker %s", c.want.nm)
			}
			if s.Basis.Status == "" || s.Reason == "" {
				t.Errorf("a steering decision carries a basis and a reason: %+v", s)
			}
		})
	}
}

func TestPreferredOnAPublicLink(t *testing.T) {
	// Mesh Edge 1 has two public links (wired and wireless): Preferred on the wired group.
	tp := demo(t)
	tp.Profiles[1].BusinessRules = []BusinessRule{{ID: "mesh-pref", Segment: "global", Match: RuleMatch{Application: "app"},
		Action: RuleAction{NetworkService: NetMultipath, Steering: Steering{Mode: SteerGroup, Group: "public_wired", Strictness: "preferred"}}}}
	flow := edgeFlow("mesh-1-edge", "mesh-2-edge", "app")
	pick := func(c Conditions) (string, string) {
		d := decide(t, tp, c, flow)
		if d.Steering == nil || d.Steering.Chosen == nil {
			return d.Steering.Action, ""
		}
		return d.Steering.Action, d.Steering.Chosen.LocalLink
	}
	if a, l := pick(Conditions{}); a != "use" || l != "m1-inet" {
		t.Errorf("up: %s %s", a, l)
	}
	if a, l := pick(Conditions{LinksDegraded: []string{"m1-inet"}}); a != "move" || l != "m1-lte" {
		t.Errorf("SLA missed moves to another public link: %s %s", a, l)
	}
	if a, l := pick(Conditions{LinksDown: []string{"m1-inet"}}); a != "move" || l != "m1-lte" {
		t.Errorf("down moves to another public link: %s %s", a, l)
	}
	// Nothing else up: stay on the degraded preferred link.
	if a, l := pick(Conditions{LinksDegraded: []string{"m1-inet"}, LinksDown: []string{"m1-lte"}}); a != "use" || l != "m1-inet" {
		t.Errorf("nothing better: %s %s", a, l)
	}
}

func TestSteeringStrictnessDefaultsToAvailableWithAssumption(t *testing.T) {
	tp := demo(t)
	tp.Profiles[2].BusinessRules[1].Action.Steering.Strictness = ""
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "video"))
	if d.Steering.Strictness != "available" || !hasBasisRule(d.Steering.Also, RA57GroupStrictness) {
		t.Errorf("%+v", d.Steering)
	}
}

func TestMTU(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "chat"))
	m := d.MTU
	if m == nil || m.PathMTU != 1500 || m.EffectiveMin != 1388 || m.EffectiveMax != 1403 || m.OverheadMin != 97 || m.OverheadMax != 112 {
		t.Fatalf("default MTU: %+v", m)
	}
	if m.Basis.Status != StatusVerified || !hasBasisRule(m.Also, RMTUDerived) {
		t.Errorf("basis: %+v", m.Basis)
	}
	// spoke-3 has a 1400-byte link: the path MTU is the smaller of the two links of the first leg.
	d = decide(t, tp, Conditions{}, edgeFlow("spoke-3-edge", "spoke-2-edge", "chat"))
	if d.MTU == nil || d.MTU.PathMTU != 1400 || d.MTU.EffectiveMin != 1288 || d.MTU.EffectiveMax != 1303 || d.MTU.Limiting != "link s3-inet" {
		t.Errorf("1400 link: %+v", d.MTU)
	}
	for _, mtu := range []int{1300, 1500} {
		tp := demo(t)
		for i := range tp.Links {
			if tp.Links[i].ID == "s3-inet" {
				m := mtu
				tp.Links[i].MTU = &m
			}
		}
		d := decide(t, tp, Conditions{}, edgeFlow("spoke-3-edge", "spoke-2-edge", "chat"))
		if d.MTU.PathMTU != mtu || d.MTU.EffectiveMax != mtu-97 {
			t.Errorf("mtu %d: %+v", mtu, d.MTU)
		}
	}
}

func TestPathSelectionRejectsUnknownThings(t *testing.T) {
	tp := demo(t)
	bad := []Flow{
		{Source: "nope", Destination: FlowDest{Kind: "edge", ID: "spoke-2-edge"}},
		{Source: "spoke-1-edge", Destination: FlowDest{Kind: "edge", ID: "nope"}},
		{Source: "spoke-1-edge", Destination: FlowDest{Kind: "edge", ID: "spoke-1-edge"}},
		{Source: "spoke-1-edge", Destination: FlowDest{Kind: "nsd", ID: "nope"}},
		{Source: "spoke-1-edge", Destination: FlowDest{Kind: "prefix", ID: "10.0.0.0/24"}},
	}
	for _, f := range bad {
		_, err := SelectPath(tp, Conditions{}, f, fakeClock())
		if _, ok := err.(*FlowError); !ok {
			t.Errorf("%+v: want FlowError, got %v", f, err)
		}
	}
}

func TestPathDecisionIsDeterministic(t *testing.T) {
	a, _ := json.Marshal(decide(t, demo(t), Conditions{LinksDegraded: []string{"s1-inet"}}, edgeFlow("spoke-1-edge", "spoke-2-edge", "video")))
	b, _ := json.Marshal(decide(t, demo(t), Conditions{LinksDegraded: []string{"s1-inet"}}, edgeFlow("spoke-1-edge", "spoke-2-edge", "video")))
	if string(a) != string(b) {
		t.Fatal("SelectPath must be deterministic")
	}
}

func TestSteeringShowsTheHoldAndRemediationAssumptions(t *testing.T) {
	tp := demo(t)
	d := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "video"))
	if !hasBasisRule(d.Steering.Also, RSteerHold) || !hasBasisRule(d.Steering.Also, RA67HoldNotApplied) {
		t.Errorf("a non-Auto decision must show that the one-minute hold is not applied: %+v", d.Steering.Also)
	}
	d = decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "chat"))
	if hasBasisRule(d.Steering.Also, RA67HoldNotApplied) {
		t.Error("the hold applies to a preferred link the policy names, not to Auto")
	}
	d = decide(t, tp, Conditions{LinksDegraded: []string{"s1-inet"}}, edgeFlow("spoke-1-edge", "spoke-2-edge", "video"))
	if !hasBasisRule(d.Steering.Also, RA68NoRemediation) {
		t.Errorf("a degraded outcome must say remediation is not modelled: %+v", d.Steering.Also)
	}
	m := decide(t, tp, Conditions{}, edgeFlow("spoke-1-edge", "spoke-2-edge", "chat")).MTU
	if !hasBasisRule(m.Also, RA66VPNOverhead) {
		t.Errorf("the MTU result assumes VPN overhead for site-to-site traffic: %+v", m.Also)
	}
}
