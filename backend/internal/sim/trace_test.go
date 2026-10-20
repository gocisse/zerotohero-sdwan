package sim

import (
	"strings"
	"testing"
)

func hdemo(t *testing.T) *Topology { return demo(t) }

func trace(t *testing.T, tp *Topology, c Conditions, as TraceAssumptions, f FlowSpec) *TraceResult {
	t.Helper()
	r, err := TraceFlow(tp, c, as, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func stg(r *TraceResult, id string) *TraceStage {
	for i := range r.Stages {
		if r.Stages[i].ID == id {
			return &r.Stages[i]
		}
	}
	return nil
}

func hasNMRule(r *TraceResult, rule string) bool { return hasRule(r.NotModelled, rule) }

// A flow delivered over each kind of path.
func TestTraceDeliveredOverEachPathType(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Topology)
		cond   Conditions
		flow   FlowSpec
		kind   string
	}{
		{"direct dynamic tunnel", nil, Conditions{Traffic: []TrafficSpec{{A: "mesh-1-edge", B: "mesh-2-edge", FirstPacketAgoS: 60, LastPacketAgoS: 2}}},
			FlowSpec{Source: "mesh-1-edge", Destination: "edge:mesh-2-edge"}, OptDirect},
		{"via the Gateway", nil, Conditions{}, FlowSpec{Source: "mesh-1-edge", Destination: "edge:mesh-2-edge"}, OptViaGateway},
		{"via a Hub", nil, Conditions{}, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge"}, OptViaHub},
		{"static Hub tunnel", nil, Conditions{}, FlowSpec{Source: "spoke-1-edge", Destination: "edge:hub-p-edge"}, OptHubTunnel},
		{"Non SD-WAN destination via Gateway", nil, Conditions{}, FlowSpec{Source: "spoke-1-edge", Destination: "nsd:azure-vhub"}, OptNSDGateway},
		{"Non SD-WAN destination via Edge", nil, Conditions{}, FlowSpec{Source: "spoke-2-edge", Destination: "nsd:aws-vpn"}, OptNSDEdge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := hdemo(t)
			if c.mutate != nil {
				c.mutate(tp)
			}
			r := trace(t, tp, c.cond, DefaultTraceAssumptions(), c.flow)
			if r.Verdict != VerdictDelivered {
				t.Fatalf("verdict %s at %s: %s", r.Verdict, r.BlockedStage, r.Summary)
			}
			p := stg(r, TrPath)
			if p.Path == nil || p.Path.Chosen == nil || p.Path.Chosen.Kind != c.kind {
				t.Errorf("chosen = %+v, want %s", p.Path.Chosen, c.kind)
			}
			if len(r.Stages) != 8 {
				t.Errorf("want 8 stages, got %d", len(r.Stages))
			}
			for _, s := range r.Stages {
				if s.Basis.Status == "" || s.LessonRef == "" || s.Screen == "" {
					t.Errorf("stage %s lacks basis, lesson or screen", s.ID)
				}
			}
			if !strings.Contains(r.Summary, "delivered") {
				t.Errorf("summary = %q", r.Summary)
			}
		})
	}
}

func TestTraceBlockedByPolicyThroughMandatorySteering(t *testing.T) {
	tp := hdemo(t)
	f := FlowSpec{Source: "spoke-1-edge", Destination: "edge:hub-p-edge", Application: "bulk"}
	ok := trace(t, tp, Conditions{}, DefaultTraceAssumptions(), f)
	if v := chosenRule(ok); ok.Verdict != VerdictDelivered || v != "spoke-bulk" {
		t.Fatalf("baseline: %s %s", ok.Verdict, ok.Summary)
	}
	r := trace(t, tp, Conditions{LinksDown: []string{"s1-inet"}}, DefaultTraceAssumptions(), f)
	if r.Verdict != VerdictBlocked || r.BlockedStage != TrPath {
		t.Fatalf("want blocked at path, got %s at %s: %s", r.Verdict, r.BlockedStage, r.Summary)
	}
	if !strings.Contains(stg(r, TrPath).Decision, "link steering") {
		t.Errorf("decision = %q", stg(r, TrPath).Decision)
	}
	// The same flow without the Mandatory rule is not blocked.
	f.Application = "voice"
	if r2 := trace(t, tp, Conditions{LinksDown: []string{"s1-inet"}}, DefaultTraceAssumptions(), f); r2.Verdict != VerdictDelivered {
		t.Errorf("voice: %s", r2.Summary)
	}
}

func fwTopology(t *testing.T, stateful bool) *Topology {
	tp := hdemo(t)
	p := profileOf(tp, "spoke")
	p.FirewallRules = []FirewallRule{
		{ID: "skip-web", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "80"}, Action: FwSkip},
		{ID: "drop-telnet", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "23"}, Action: FwDrop},
		{ID: "reject-ssh", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "22"}, Action: FwReject},
		{ID: "allow-web", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "80,443"}, Action: FwAllow},
	}
	if p.Device == nil {
		p.Device = map[string]any{}
	}
	p.Device[StatefulSetting] = stateful
	return tp
}

func TestTraceBlockedByTheFirewallAsDocumented(t *testing.T) {
	flow := func(port int) FlowSpec {
		return FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge", Protocol: "tcp", Port: port}
	}
	as := DefaultTraceAssumptions()
	t.Run("Drop is silent", func(t *testing.T) {
		r := trace(t, fwTopology(t, false), Conditions{}, as, flow(23))
		if r.Verdict != VerdictBlocked || r.BlockedStage != TrFirewall || !strings.Contains(stg(r, TrFirewall).Decision, "silently") {
			t.Fatalf("%s %s", r.Verdict, r.Summary)
		}
		if stg(r, TrRoute).Status != TsNotReached {
			t.Error("later stages must not be reached")
		}
	})
	t.Run("Reject needs Stateful Firewall: with it off the rule is ignored", func(t *testing.T) {
		tp := fwTopology(t, false)
		if errs := Errors(Validate(tp)); len(errs) == 0 {
			t.Fatal("the fixture should be invalid (Reject and Skip without Stateful Firewall)")
		}
		r := trace(t, tp, Conditions{}, as, flow(22))
		if r.Verdict != VerdictDelivered {
			t.Errorf("an ignored Reject rule must not block: %s", r.Summary)
		}
	})
	t.Run("Reject with Stateful Firewall on", func(t *testing.T) {
		r := trace(t, fwTopology(t, true), Conditions{}, as, flow(22))
		if r.Verdict != VerdictBlocked || !strings.Contains(stg(r, TrFirewall).Decision, "rejects") {
			t.Fatalf("%s %s", r.Verdict, r.Summary)
		}
	})
	t.Run("Skip goes to the next rule", func(t *testing.T) {
		r := trace(t, fwTopology(t, true), Conditions{}, as, flow(80))
		fw := stg(r, TrFirewall)
		if r.Verdict != VerdictDelivered || !fw.Firewall[3].Decisive || fw.Firewall[0].Decisive {
			t.Errorf("verdicts = %+v", fw.Firewall)
		}
	})
	t.Run("the no-match action and the order are switches and visible", func(t *testing.T) {
		tp := fwTopology(t, true)
		other := flow(8080)
		r := trace(t, tp, Conditions{}, as, other)
		if r.Verdict != VerdictDelivered || !hasNMRule(r, RNMFwOrder) {
			t.Fatalf("default allow: %s", r.Summary)
		}
		deny := as
		deny.FirewallDefaultAction = FwDrop
		r = trace(t, tp, Conditions{}, deny, other)
		if r.Verdict != VerdictBlocked || r.BlockedStage != TrFirewall {
			t.Fatalf("default drop: %s", r.Summary)
		}
		if len(r.Assumptions) != 13 {
			t.Errorf("want 13 switches, got %d", len(r.Assumptions))
		}
		// With the default allow, the answer depends on the default-action switch.
		r = trace(t, tp, Conditions{}, as, other)
		found := false
		for _, d := range r.DependsOn {
			found = found || d.Switch.Key == "fw_default_action"
		}
		if !found {
			t.Errorf("depends_on = %+v", r.DependsOn)
		}
		// Last match: with two matching rules the last decides.
		last := as
		last.FirewallEvaluation = FwLastMatch
		r = trace(t, tp, Conditions{}, last, flow(80))
		if !stg(r, TrFirewall).Firewall[3].Decisive {
			t.Errorf("last match: %+v", stg(r, TrFirewall).Firewall)
		}
	})
	t.Run("the firewall decides independently of the business policy", func(t *testing.T) {
		r := trace(t, fwTopology(t, false), Conditions{}, as, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge", Application: "voice", Protocol: "tcp", Port: 23})
		if stg(r, TrPolicy).Status != TsOK || stg(r, TrFirewall).Status != TsBlocked {
			t.Errorf("policy %s, firewall %s", stg(r, TrPolicy).Status, stg(r, TrFirewall).Status)
		}
	})
}

func TestTraceNoRouteAndRouteLocalAndPathNotBuilt(t *testing.T) {
	tp := rdemo(t)
	as := DefaultTraceAssumptions()
	t.Run("no route", func(t *testing.T) {
		noDefault(tp)
		r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "c-edge", Destination: "10.250.0.1"})
		if r.Verdict != VerdictUnreachable || r.BlockedStage != TrRoute {
			t.Fatalf("%s %s", r.Verdict, r.Summary)
		}
		if stg(r, TrRoute).Reach == nil || stg(r, TrRoute).Reach.FirstFailing != StageKnown {
			t.Errorf("the route stage must carry the reachability explanation: %+v", stg(r, TrRoute).Reach)
		}
	})
	tp = rdemo(t)
	t.Run("delivered by a route and a path", func(t *testing.T) {
		r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "c-edge", Destination: "10.20.0.5", Application: "voice", Protocol: "udp", Port: 5060, TrafficClass: "real_time"})
		if r.Verdict != VerdictDelivered {
			t.Fatalf("%s", r.Summary)
		}
		if rt := stg(r, TrRoute).Route; rt == nil || rt.Origin != "a-edge" {
			t.Errorf("route = %+v", rt)
		}
		rem := stg(r, TrRemediation)
		if rem.Status != TsNotModeled || !hasNMRule(r, RNMRemedThresh) || len(rem.Remediation) != 3 || !rem.Remediation[0].Candidate {
			t.Errorf("remediation = %+v", rem)
		}
		if stg(r, TrMTU).MTU == nil || stg(r, TrMTU).MTU.EffectiveMin != 1388 {
			t.Errorf("mtu = %+v", stg(r, TrMTU).MTU)
		}
	})
	t.Run("a local destination never crosses the overlay", func(t *testing.T) {
		r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "c-edge", Destination: "10.60.1.5"})
		if r.Verdict != VerdictDelivered || stg(r, TrPath).Status != TsSkipped || !strings.Contains(r.Summary, "without crossing") {
			t.Errorf("%s %s", r.Verdict, r.Summary)
		}
	})
	t.Run("path not built: the owner's link is down", func(t *testing.T) {
		noDefault(tp)
		r := trace(t, tp, Conditions{LinksDown: []string{"a-inet"}}, as, FlowSpec{Source: "c-edge", Destination: "10.20.0.5"})
		if r.Verdict != VerdictUnreachable || r.BlockedStage != TrPath {
			t.Fatalf("%s at %s: %s", r.Verdict, r.BlockedStage, r.Summary)
		}
		if stg(r, TrRoute).Status != TsOK {
			t.Errorf("the route exists: %s", stg(r, TrRoute).Status)
		}
	})
	t.Run("a Hub down changes a hub flow", func(t *testing.T) {
		hs := hdemo(t)
		f := FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge"}
		base := trace(t, hs, Conditions{}, as, f)
		down := trace(t, hs, Conditions{HubsUnreachable: []string{"hub-p-edge"}}, as, f)
		if base.Verdict != VerdictDelivered || down.Verdict != VerdictDelivered || stg(base, TrPath).Path.Chosen.Label == stg(down, TrPath).Path.Chosen.Label {
			t.Errorf("%q vs %q", stg(base, TrPath).Path.Chosen.Label, stg(down, TrPath).Path.Chosen.Label)
		}
	})
}

func TestTraceInterfacesSegmentsAndPolicyDetails(t *testing.T) {
	tp := rdemo(t)
	as := DefaultTraceAssumptions()
	r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "a-edge", SourceIP: "10.20.255.2", Destination: "10.30.0.5"})
	if s := stg(r, TrSegment); s.Status != TsOK || !strings.Contains(s.Decision, "ge3") {
		t.Errorf("%+v", s)
	}
	if r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "a-edge", SourceIP: "192.0.2.5", Destination: "10.30.0.5"}); r.Verdict != VerdictBlocked || r.BlockedStage != TrSegment {
		t.Errorf("a source IP outside every LAN subnet: %s", r.Summary)
	}
	if r := trace(t, tp, Conditions{}, as, FlowSpec{Source: "a-edge", Destination: "10.30.0.5", Segment: "nope"}); r.BlockedStage != TrSegment {
		t.Errorf("unknown segment: %s", r.Summary)
	}
	// No rule matches: the default is marked not modelled, and the trace continues by assumption 2.
	r = trace(t, tp, Conditions{}, as, FlowSpec{Source: "a-edge", Destination: "10.30.0.5", Application: "voice"})
	if p := stg(r, TrPolicy); p.Status != TsInfo || !hasRule(p.NotModelled, RNMBizDefault) || r.Verdict != VerdictDelivered {
		t.Errorf("%+v", p)
	}
	// Rule matching carries the protocol, port and DSCP the flow brings.
	hs := hdemo(t)
	pr := profileOf(hs, "spoke")
	pr.BusinessRules = append([]BusinessRule{{ID: "ssh-first", Segment: "global", Match: RuleMatch{Protocol: "tcp", Port: "20-25"}, Action: RuleAction{NetworkService: NetMultipath}}}, pr.BusinessRules...)
	m := trace(t, hs, Conditions{}, as, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge", Protocol: "tcp", Port: 22})
	if v := chosenRule(m); v != "ssh-first" {
		t.Errorf("chosen = %q", v)
	}
	m = trace(t, hs, Conditions{}, as, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge", Protocol: "tcp", Port: 80})
	for _, v := range stg(m, TrPolicy).Rules {
		if v.RuleID == "ssh-first" && v.Matched {
			t.Error("port 80 is outside 20-25")
		}
	}
	// Orchestrator unreachable: not modelled at the path stage.
	o := trace(t, hs, Conditions{OrchestratorUnreachable: true}, as, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge"})
	if o.Verdict != VerdictNotModelled || o.BlockedStage != TrPath {
		t.Errorf("%s at %s", o.Verdict, o.BlockedStage)
	}
	// NSD: the MTU and the return path are marked not modelled.
	n := trace(t, hs, Conditions{}, as, FlowSpec{Source: "spoke-1-edge", Destination: "nsd:azure-vhub"})
	if !hasNMRule(n, RNMNSDMTU) || !hasNMRule(n, RNMTraceReturn) {
		t.Errorf("not modelled: %+v", n.NotModelled)
	}
}

func TestTraceMTUPacketSizeAndStatefulAsymmetry(t *testing.T) {
	hs := hdemo(t)
	as := DefaultTraceAssumptions()
	f := FlowSpec{Source: "spoke-3-edge", Destination: "edge:spoke-2-edge", PacketSize: 1450}
	r := trace(t, hs, Conditions{}, as, f)
	mtu := stg(r, TrMTU)
	if mtu.MTU == nil || !strings.Contains(strings.Join(mtu.Details, " "), "fragments") {
		t.Errorf("%+v", mtu)
	}
	f.PacketSize = 1000
	if !strings.Contains(strings.Join(stg(trace(t, hs, Conditions{}, as, f), TrMTU).Details, " "), "fits") {
		t.Error("a small packet fits")
	}
	// Stateful Firewall on and the return path differs: spoke-1 reaches spoke-2 via the primary Hub; with the primary
	// Hub unreachable from one side only the kinds can differ, which is covered by the unit below.
	rs := stg(trace(t, hs, Conditions{}, as, FlowSpec{Source: "spoke-1-edge", Destination: "edge:spoke-2-edge"}), TrReturn)
	if rs.Status != TsOK || !strings.Contains(rs.Decision, "reverse") {
		t.Errorf("%+v", rs)
	}
}

func TestTraceIsDeterministicAndDoesNotMutate(t *testing.T) {
	tp := rdemo(t)
	before := jsonOf(t, tp)
	c := Conditions{LinksDown: []string{"a-inet"}}
	f := FlowSpec{Source: "c-edge", Destination: "10.20.0.5", Application: "voice"}
	a := jsonOf(t, trace(t, tp, c, DefaultTraceAssumptions(), f))
	b := jsonOf(t, trace(t, tp, c, DefaultTraceAssumptions(), f))
	if a != b {
		t.Error("TraceFlow is not deterministic")
	}
	if jsonOf(t, tp) != before {
		t.Error("the topology changed")
	}
}

func TestTraceQueryErrors(t *testing.T) {
	tp := rdemo(t)
	for _, f := range []FlowSpec{{Source: "ghost", Destination: "10.0.0.1"}, {Source: "a-edge", Destination: "nonsense"}, {Source: "a-edge", Destination: "edge:ghost"}, {Source: "a-edge", Destination: "nsd:ghost"}} {
		if _, err := TraceFlow(tp, Conditions{}, DefaultTraceAssumptions(), f, nil); err == nil {
			t.Errorf("%+v should be rejected", f)
		}
	}
	bad := DefaultTraceAssumptions()
	bad.FirewallEvaluation = "random"
	if _, err := TraceFlow(tp, Conditions{}, bad, FlowSpec{Source: "a-edge", Destination: "10.0.0.1"}, nil); err == nil || !IsAssumptionError(err) {
		t.Errorf("bad assumptions: %v", err)
	}
}

func chosenRule(r *TraceResult) string {
	for _, v := range stg(r, TrPolicy).Rules {
		if v.Chosen {
			return v.RuleID
		}
	}
	return ""
}

// The lesson 8 check 4 appears in the trace: an Edge that runs BGP toward its LAN must hand overlay routes to it.
func TestTraceBlocksWhenTheLANNeverLearnsTheRoute(t *testing.T) {
	tp := rdemo(t)
	f := FlowSpec{Source: "b-edge", Destination: "10.20.0.5"}
	r := trace(t, tp, Conditions{}, DefaultTraceAssumptions(), f)
	if r.Verdict != VerdictUnreachable || r.BlockedStage != TrRoute || stg(r, TrRoute).Route == nil {
		t.Fatalf("%s at %s: %s", r.Verdict, r.BlockedStage, r.Summary)
	}
	if !strings.Contains(stg(r, TrRoute).Decision, "LAN routers") {
		t.Errorf("decision = %q", stg(r, TrRoute).Decision)
	}
	on := DefaultTraceAssumptions()
	on.BGPOverlayPrefix = true
	if r := trace(t, tp, Conditions{}, on, f); r.Verdict != VerdictDelivered {
		t.Errorf("with Overlay Prefix on: %s", r.Summary)
	}
	found := false
	for _, d := range r.DependsOn {
		found = found || d.Switch.Key == "bgp_overlay_prefix"
	}
	if !found {
		t.Errorf("depends_on = %+v", r.DependsOn)
	}
}

func TestTraceStageNotModelledMarkersAreNotRepeated(t *testing.T) {
	r := trace(t, hdemo(t), Conditions{}, DefaultTraceAssumptions(), FlowSpec{Source: "mesh-1-edge", Destination: "edge:mesh-2-edge"})
	for _, st := range r.Stages {
		seen := map[string]bool{}
		for _, nm := range st.NotModelled {
			if seen[nm.Rule] {
				t.Errorf("stage %s repeats %s", st.ID, nm.Rule)
			}
			seen[nm.Rule] = true
		}
	}
}
