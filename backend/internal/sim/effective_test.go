package sim

import (
	"errors"
	"testing"
)

func mustEffective(t *testing.T, tp *Topology, edge string) *Effective {
	t.Helper()
	ec, err := EffectiveConfig(tp, edge)
	if err != nil {
		t.Fatal(err)
	}
	return ec
}

func setting(ec *Effective, id string) EffectiveSetting {
	for _, s := range ec.Device {
		if s.ID == id {
			return s
		}
	}
	panic("no setting " + id)
}

func TestOverrideWinsOverProfile(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e1")
	s := setting(ec, "connectivity.arp_timeouts")
	if s.Value != 300 || s.Source.Kind != SrcEdgeOverride || s.Source.ID != "e1" {
		t.Fatalf("override should win: %+v", s)
	}
	if s.ProfileValue != 120 {
		t.Errorf("profile value should still be shown: %v", s.ProfileValue)
	}
	if s.Basis.Status != StatusVerified || s.Basis.Ref != "edge-overrides" {
		t.Errorf("layer merge is verified in edge-overrides: %+v", s.Basis)
	}
}

func TestInheritanceWhenNoOverride(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e2")
	s := setting(ec, "connectivity.arp_timeouts")
	if s.Value != 120 || s.Source.Kind != SrcProfile || s.Source.ID != "p1" {
		t.Fatalf("e2 should inherit from p1: %+v", s)
	}
}

func TestDocumentedDefaultAndUnset(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[1].Profile = "p2" // p2 sets no stateful firewall and no BFD
	ec := mustEffective(t, tp, "e2")
	st := setting(ec, StatefulSetting)
	if st.Value != false || st.Source.Kind != SrcDefault || st.Basis.Rule != RFwStatefulDefault || st.Basis.Status != StatusVerified {
		t.Fatalf("stateful firewall has a documented default (off): %+v", st)
	}
	bfd := setting(ec, "routing.bfd")
	if bfd.Value != nil || bfd.Source.Kind != SrcUnset || bfd.NotModelled == nil || bfd.Basis.Status != StatusNotModelled {
		t.Fatalf("BFD has no documented default: it must come back unset with a not-modelled marker: %+v", bfd)
	}
}

func TestEveryValueCarriesBasis(t *testing.T) {
	for name, tp := range loadContentTopologies(t) {
		for _, s := range tp.Sites {
			for _, e := range s.Edges {
				ec := mustEffective(t, tp, e.ID)
				for _, d := range ec.Device {
					if d.Basis.Rule == "" || d.Basis.Ref == "" || d.Basis.Status == "" {
						t.Errorf("%s %s %s: missing basis %+v", name, e.ID, d.ID, d.Basis)
					}
				}
				for _, v := range ec.VLANs {
					if v.Basis.Rule == "" {
						t.Errorf("%s %s vlan %d: missing basis", name, e.ID, v.ID)
					}
				}
				for _, in := range ec.Interfaces {
					if in.Basis.Rule == "" {
						t.Errorf("%s %s interface %s: missing basis", name, e.ID, in.ID)
					}
				}
				if len(ec.NotModelled) == 0 {
					t.Errorf("%s %s: not-modelled markers missing", name, e.ID)
				}
			}
		}
	}
}

func TestBusinessRuleSupersession(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e1")
	var global SegmentPolicy
	for _, sp := range ec.BusinessPolicy {
		if sp.Segment.ID == "global" {
			global = sp
		}
	}
	if len(global.EdgeOverrides) != 1 || global.EdgeOverrides[0].Rule.ID != "e1-voice" {
		t.Fatalf("edge rule missing: %+v", global.EdgeOverrides)
	}
	if got := global.EdgeOverrides[0].Supersedes; len(got) != 1 || got[0] != "p1-voice" {
		t.Errorf("e1-voice supersedes p1-voice (same match): %v", got)
	}
	if global.FromProfile[0].SupersededBy != "e1-voice" {
		t.Errorf("profile rule should be marked superseded: %+v", global.FromProfile[0])
	}
	hasA41 := false
	for _, b := range global.EdgeOverrides[0].Also {
		hasA41 = hasA41 || (b.Rule == RA41SameMatch && b.Status == StatusAssumption)
	}
	if !hasA41 {
		t.Errorf("matching by same match is assumption 41 and must be visible: %+v", global.EdgeOverrides[0].Also)
	}
	if global.Precedence.Basis.Status != StatusNotModelled || global.Precedence.Rule != RNMBizOrder {
		t.Errorf("rule evaluation order must be returned as not modelled: %+v", global.Precedence)
	}
}

func TestDifferentMatchIsNotSuperseded(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[0].Overrides.BusinessRules[0].Match.Application = "video"
	ec := mustEffective(t, tp, "e1")
	for _, sp := range ec.BusinessPolicy {
		for _, r := range sp.FromProfile {
			if r.SupersededBy != "" {
				t.Errorf("rule %s must not be superseded by a rule with a different match", r.Rule.ID)
			}
		}
	}
}

func TestPrivateSegmentRewritesMultipathToDirect(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e1")
	for _, sp := range ec.BusinessPolicy {
		for _, r := range sp.FromProfile {
			switch {
			case sp.Segment.ID == "guest" && (!r.Rewritten || r.EffectiveService != NetDirect):
				t.Errorf("private segment multipath rule must read direct: %+v", r)
			case sp.Segment.ID == "global" && r.Rewritten:
				t.Errorf("regular segment must not be rewritten: %+v", r)
			}
		}
	}
}

func TestFirewallEdgeRulesComeFirst(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e1")
	got := []string{}
	for _, r := range ec.Firewall.Rules {
		got = append(got, r.Group+":"+r.Rule.ID)
	}
	want := []string{"edge-override:e1-ssh", "profile:p1-allow", "profile:p1-reject"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if ec.Firewall.Priority.Status != StatusVerified || ec.Firewall.Order.Basis.Status != StatusNotModelled {
		t.Errorf("group priority is verified, evaluation order is not modelled: %+v %+v", ec.Firewall.Priority, ec.Firewall.Order)
	}
}

func TestFirewallWithoutOverrideIsInheritedOnly(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e2")
	for _, r := range ec.Firewall.Rules {
		if r.Group != "profile" {
			t.Errorf("e2 has no override, got %s rule %s", r.Group, r.Rule.ID)
		}
	}
	if ec.Firewall.Override {
		t.Error("override tick should be off")
	}
}

func TestInterfaceSegmentsDefaultToAll(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e2")
	for _, in := range ec.Interfaces {
		if in.ID == "ge3" && (in.SegmentsSource != "profile-default" || len(in.Segments) != 2) {
			t.Errorf("an interface with no segments applies to all of the Profile's: %+v", in)
		}
	}
}

func TestLoopbackIsEdgeOnly(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[1].Interfaces = append(tp.Sites[0].Edges[1].Interfaces, Interface{ID: "lo1", Kind: IfLoopback})
	ec := mustEffective(t, tp, "e2")
	for _, in := range ec.Interfaces {
		if in.ID == "lo1" && (in.Source.Kind != SrcEdgeOnly || in.Basis.Rule != RLoopbackEdgeOnly) {
			t.Errorf("loopbacks exist only on Edges: %+v", in)
		}
	}
}

func TestUnknownEdge(t *testing.T) {
	if _, err := EffectiveConfig(baseTopology(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCustomerDefaults(t *testing.T) {
	ec := mustEffective(t, baseTopology(), "e1")
	for _, c := range ec.Customer {
		if c.Source.Kind != SrcDefault {
			t.Errorf("%s should come from the documented default: %+v", c.ID, c)
		}
	}
	if ec.Customer[0].Value != DefaultMaxSegments || ec.Customer[1].Value != true {
		t.Errorf("defaults: %+v", ec.Customer)
	}
}

func TestVlanOverrideKeepsNameAndID(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[1].Overrides.VLANs = []VLANOverride{{ID: 10, Settings: map[string]any{"x": 1}}}
	ec := mustEffective(t, tp, "e2")
	for _, v := range ec.VLANs {
		if v.ID == 10 && (v.Name != "Corp" || !v.Locked || v.Source.Kind != SrcEdgeOverride) {
			t.Errorf("a VLAN override keeps the Profile name and id: %+v", v)
		}
	}
}
