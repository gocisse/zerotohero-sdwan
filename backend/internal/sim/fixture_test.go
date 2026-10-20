package sim

import (
	"os"
	"path/filepath"
	"testing"
)

func ip(i int) *int { return &i }

// baseTopology is a small valid teaching topology used as the starting point of table tests.
func baseTopology() *Topology {
	return &Topology{
		Version:  1,
		Meta:     Meta{ID: "t", Title: "T", Summary: "s", Teaching: true},
		Customer: Customer{ID: "c", Name: "C"},
		Segments: []Segment{{ID: "global", Name: "Global Segment", Type: SegRegular}, {ID: "guest", Name: "Guest", Type: SegPrivate}},
		Gateways: []Gateway{{ID: "gw", Region: "r1"}},
		Links: []WanLink{
			{ID: "l-a1", Type: LinkPublic, Medium: MediumWired, UpMbps: 100, DownMbps: 100},
			{ID: "l-a2", Type: LinkPrivate, Medium: MediumWired, UpMbps: 50, DownMbps: 50},
			{ID: "l-b1", Type: LinkPublic, Medium: MediumWired, UpMbps: 100, DownMbps: 100},
		},
		Profiles: []Profile{
			{
				ID: "p1", Name: "P1", Segments: []string{"global", "guest"},
				Device: map[string]any{"connectivity.arp_timeouts": 120, "routing.bfd": false, StatefulSetting: true},
				VLANs:  []VLAN{{ID: 10, Name: "Corp"}, {ID: 20, Name: "Guest"}},
				BusinessRules: []BusinessRule{
					{ID: "p1-voice", Segment: "global", Match: RuleMatch{Application: "voice"},
						Action: RuleAction{Priority: "high", NetworkService: NetMultipath}},
					{ID: "p1-guest", Segment: "guest", Match: RuleMatch{Destination: DestInternet},
						Action: RuleAction{NetworkService: NetMultipath}},
				},
				FirewallRules: []FirewallRule{
					{ID: "p1-allow", Match: FirewallMatch{VLAN: "10"}, Action: FwAllow},
					{ID: "p1-reject", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "23"}, Action: FwReject},
				},
			},
			{
				ID: "p2", Name: "P2", Segments: []string{"global", "guest"},
				Device: map[string]any{"connectivity.arp_timeouts": 60},
				VLANs:  []VLAN{{ID: 10, Name: "Corp"}, {ID: 30, Name: "Servers"}},
			},
		},
		Sites: []Site{
			{ID: "s1", Name: "S1", Region: "r1", Edges: []Edge{
				{
					ID: "e1", Name: "E1", Role: RoleBranch, Profile: "p1",
					Interfaces: []Interface{
						{ID: "ge1", Kind: IfWan, Link: "l-a1"},
						{ID: "ge2", Kind: IfWan, Link: "l-a2"},
						{ID: "ge3", Kind: IfLan, VLAN: ip(10), Segments: []string{"global"}},
						{ID: "ge4", Kind: IfLan, VLAN: ip(20), Segments: []string{"guest"}, Override: true},
					},
					Overrides: EdgeOverrides{
						Device: map[string]any{"connectivity.arp_timeouts": 300},
						BusinessRules: []BusinessRule{
							{ID: "e1-voice", Segment: "global", Match: RuleMatch{Application: "voice"},
								Action: RuleAction{Priority: "high", NetworkService: NetMultipath,
									Steering: Steering{Mode: SteerWanLink, Link: "l-a2", Strictness: "mandatory"}}},
						},
						Firewall: FirewallOverride{Override: true, Rules: []FirewallRule{
							{ID: "e1-ssh", Match: FirewallMatch{Protocol: "tcp", DestinationPort: "22"}, Action: FwAllow},
						}},
					},
				},
				{
					ID: "e2", Name: "E2", Role: RoleBranch, Profile: "p1",
					Interfaces: []Interface{
						{ID: "ge1", Kind: IfWan, Link: "l-b1"},
						{ID: "ge3", Kind: IfLan, VLAN: ip(10)},
					},
				},
			}},
		},
	}
}

// loadContentTopologies decodes every file in content/topologies for tests that check the shipped topologies.
func loadContentTopologies(t *testing.T) map[string]*Topology {
	t.Helper()
	files, err := filepath.Glob("../../../content/topologies/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no topologies found: %v", err)
	}
	out := map[string]*Topology{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		tp, err := Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = tp
	}
	return out
}
