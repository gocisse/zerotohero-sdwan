package sim

import (
	"strings"
	"testing"
)

func TestBaseTopologyIsValid(t *testing.T) {
	if errs := Errors(Validate(baseTopology())); len(errs) != 0 {
		t.Fatalf("base topology should be valid: %v", errs)
	}
}

func TestShippedTopologiesAreValid(t *testing.T) {
	for name, tp := range loadContentTopologies(t) {
		for _, is := range Validate(tp) {
			t.Errorf("%s: %s %s", name, is.Level, is)
		}
		if !tp.Meta.Teaching {
			t.Errorf("%s: must be labelled teaching", name)
		}
	}
}

// Each case mutates the base topology and expects an issue at a path with a message fragment.
func TestValidationRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Topology)
		level  string
		path   string // substring of the issue path
		msg    string // substring of the message
	}{
		{"wrong version", func(x *Topology) { x.Version = 2 }, LevelError, "version", "must be 1"},
		{"bad meta id", func(x *Topology) { x.Meta.ID = "Bad ID" }, LevelError, "meta.id", "lowercase slug"},
		{"missing title", func(x *Topology) { x.Meta.Title = "" }, LevelError, "meta.title", "required"},
		{"not teaching", func(x *Topology) { x.Meta.Teaching = false }, LevelError, "meta.teaching", "teaching topology"},
		{"customer id", func(x *Topology) { x.Customer.ID = "" }, LevelError, "customer.id", "slug"},
		{"max segments zero", func(x *Topology) { x.Customer.MaxSegments = ip(0) }, LevelError, "customer.max_segments", "at least 1"},
		{"max segments above 128", func(x *Topology) { x.Customer.MaxSegments = ip(200) }, LevelWarning, "customer.max_segments", "128"},
		{"too many segments", func(x *Topology) {
			x.Customer.MaxSegments = ip(2)
			x.Segments = append(x.Segments, Segment{ID: "x", Name: "X", Type: SegRegular})
		}, LevelError, "segments", "exceed"},
		{"duplicate segment", func(x *Topology) { x.Segments = append(x.Segments, Segment{ID: "guest", Name: "G2", Type: SegRegular}) }, LevelError, "segments[guest]", "duplicate"},
		{"bad segment type", func(x *Topology) { x.Segments[1].Type = "weird" }, LevelError, "type", "regular"},
		{"gateway region", func(x *Topology) { x.Gateways[0].Region = "" }, LevelError, "region", "required"},
		{"duplicate gateway", func(x *Topology) { x.Gateways = append(x.Gateways, Gateway{ID: "gw", Region: "r"}) }, LevelError, "gateways[gw]", "duplicate"},
		{"duplicate link", func(x *Topology) { x.Links = append(x.Links, x.Links[0]) }, LevelError, "links[l-a1]", "duplicate"},
		{"link type", func(x *Topology) { x.Links[0].Type = "x" }, LevelError, "type", "public"},
		{"link medium", func(x *Topology) { x.Links[0].Medium = "x" }, LevelError, "medium", "wired"},
		{"link bandwidth", func(x *Topology) { x.Links[0].UpMbps = 0 }, LevelError, "links[l-a1]", "positive"},
		{"private wireless", func(x *Topology) { x.Links[1].Medium = MediumWireless }, LevelWarning, "links[l-a2]", "transport groups"},
		{"unused link", func(x *Topology) {
			x.Links = append(x.Links, WanLink{ID: "spare", Type: LinkPublic, Medium: MediumWired, UpMbps: 1, DownMbps: 1})
		}, LevelWarning, "links[spare]", "not used"},
		{"duplicate profile", func(x *Topology) { x.Profiles = append(x.Profiles, x.Profiles[0]) }, LevelError, "profiles[p1]", "duplicate"},
		{"profile no segments", func(x *Topology) { x.Profiles[0].Segments = nil }, LevelError, "profiles[p1].segments", "at least one"},
		{"profile unknown segment", func(x *Topology) { x.Profiles[0].Segments = append(x.Profiles[0].Segments, "nope") }, LevelError, "profiles[p1].segments", "unknown segment"},
		{"profile segment twice", func(x *Topology) { x.Profiles[0].Segments = []string{"global", "global"} }, LevelError, "profiles[p1].segments", "twice"},
		{"unknown device setting", func(x *Topology) { x.Profiles[0].Device["no.such"] = 1 }, LevelError, "profiles[p1].device.no.such", "unknown setting"},
		{"device value type", func(x *Topology) { x.Profiles[0].Device[StatefulSetting] = "yes" }, LevelError, StatefulSetting, "true or false"},
		{"device value list", func(x *Topology) { x.Profiles[0].Device["routing.bfd"] = []any{1} }, LevelError, "routing.bfd", "single value"},
		{"duplicate vlan", func(x *Topology) { x.Profiles[0].VLANs = append(x.Profiles[0].VLANs, VLAN{ID: 10, Name: "x"}) }, LevelError, "profiles[p1].vlans", "duplicate VLAN id 10"},
		{"vlan range", func(x *Topology) { x.Profiles[0].VLANs[0].ID = 5000 }, LevelError, "vlans[0].id", "between 1 and 4094"},
		{"rule duplicate id", func(x *Topology) { x.Profiles[0].BusinessRules[1].ID = "p1-voice" }, LevelError, "business_rules", "duplicate rule id"},
		{"rule unknown segment", func(x *Topology) { x.Profiles[0].BusinessRules[0].Segment = "nope" }, LevelError, "segment", "unknown segment"},
		{"rule segment not in profile", func(x *Topology) {
			x.Profiles[1].Segments = []string{"global"}
			x.Profiles[1].BusinessRules = []BusinessRule{{ID: "p2-r", Segment: "guest"}}
		}, LevelError, "p2-r", "not part of the Profile"},
		{"backhaul not internet", func(x *Topology) { x.Profiles[0].BusinessRules[0].Action.NetworkService = NetBackhaul }, LevelError, "network_service", "internet_backhaul applies only"},
		{"service class needs custom", func(x *Topology) { x.Profiles[0].BusinessRules[0].Action.ServiceClass = "bulk" }, LevelError, "service_class", "custom application"},
		{"service class value", func(x *Topology) {
			x.Profiles[0].BusinessRules[0].Match.CustomApplication = true
			x.Profiles[0].BusinessRules[0].Action.ServiceClass = "x"
		}, LevelError, "service_class", "real_time"},
		{"steering wan_link on profile", func(x *Topology) {
			x.Profiles[0].BusinessRules[0].Action.Steering = Steering{Mode: SteerWanLink, Link: "l-a2"}
		}, LevelError, "steering.mode", "only on Edge-level rules"},
		{"steering interface on profile", func(x *Topology) {
			x.Profiles[0].BusinessRules[0].Action.Steering = Steering{Mode: SteerInterface, Interface: "ge1"}
		}, LevelError, "steering.mode", "only on Edge-level rules"},
		{"steering unknown interface", func(x *Topology) {
			x.Sites[0].Edges[0].Overrides.BusinessRules[0].Action.Steering = Steering{Mode: SteerInterface, Interface: "zz"}
		}, LevelError, "steering.interface", "no interface"},
		{"steering unknown link", func(x *Topology) { x.Sites[0].Edges[0].Overrides.BusinessRules[0].Action.Steering.Link = "l-b1" }, LevelError, "steering.link", "no WAN interface"},
		{"steering group value", func(x *Topology) {
			x.Profiles[0].BusinessRules[0].Action.Steering = Steering{Mode: SteerGroup, Group: "x"}
		}, LevelError, "steering.group", "public_wired"},
		{"steering auto with group", func(x *Topology) {
			x.Profiles[0].BusinessRules[0].Action.Steering = Steering{Mode: SteerAuto, Group: "public_wired"}
		}, LevelError, "steering", "only to transport_group"},
		{"strictness with auto", func(x *Topology) { x.Profiles[0].BusinessRules[0].Action.Steering = Steering{Strictness: "mandatory"} }, LevelError, "strictness", "only with"},
		{"strictness value", func(x *Topology) { x.Sites[0].Edges[0].Overrides.BusinessRules[0].Action.Steering.Strictness = "x" }, LevelError, "strictness", "mandatory, preferred"},
		{"steering mode value", func(x *Topology) { x.Profiles[0].BusinessRules[0].Action.Steering.Mode = "x" }, LevelError, "steering.mode", "auto, transport_group"},
		{"rule protocol", func(x *Topology) { x.Profiles[0].BusinessRules[0].Match.Protocol = "sctp" }, LevelError, "match.protocol", "gre, icmp, tcp, udp"},
		{"rule priority", func(x *Topology) { x.Profiles[0].BusinessRules[0].Action.Priority = "urgent" }, LevelError, "priority", "high, normal or low"},
		{"firewall protocol", func(x *Topology) { x.Profiles[0].FirewallRules[0].Match.Protocol = "sctp" }, LevelError, "firewall_rules", "gre, icmp, tcp, udp"},
		{"firewall action", func(x *Topology) { x.Profiles[0].FirewallRules[0].Action = "x" }, LevelError, "action", "allow, drop, reject or skip"},
		{"reject needs stateful (profile)", func(x *Topology) { x.Profiles[0].Device[StatefulSetting] = false }, LevelError, "profiles[p1].firewall_rules", "needs firewall.stateful_firewall"},
		{"reject needs stateful (edge off)", func(x *Topology) {
			x.Sites[0].Edges[0].Overrides.Device[StatefulSetting] = false
			x.Sites[0].Edges[0].Overrides.Firewall.Rules[0].Action = FwSkip
		}, LevelError, "edges[e1]", "needs firewall.stateful_firewall"},
		{"edge firewall rules need override", func(x *Topology) { x.Sites[0].Edges[0].Overrides.Firewall.Override = false }, LevelError, "edges[e1].overrides.firewall", "override: true"},
		{"duplicate edge", func(x *Topology) { x.Sites[0].Edges = append(x.Sites[0].Edges, x.Sites[0].Edges[1]) }, LevelError, "edges[e2]", "duplicate edge id"},
		{"edge unknown profile", func(x *Topology) { x.Sites[0].Edges[0].Profile = "nope" }, LevelError, "edges[e1].profile", "unknown profile"},
		{"edge role", func(x *Topology) { x.Sites[0].Edges[0].Role = "spoke" }, LevelError, "role", "branch"},
		{"edge without wan", func(x *Topology) { x.Sites[0].Edges[1].Interfaces = x.Sites[0].Edges[1].Interfaces[1:] }, LevelError, "edges[e2].interfaces", "at least one WAN"},
		{"site without edges", func(x *Topology) { x.Sites = append(x.Sites, Site{ID: "s2", Name: "S2", Region: "r1"}) }, LevelError, "sites[s2].edges", "at least one Edge"},
		{"duplicate site", func(x *Topology) {
			x.Sites = append(x.Sites, Site{ID: "s1", Name: "S", Region: "r1", Edges: []Edge{{ID: "e9", Name: "E9", Role: RoleBranch, Profile: "p1", Interfaces: []Interface{{ID: "ge1", Kind: IfWan, Link: "l-b1"}}}}})
		}, LevelError, "sites[s1]", "duplicate site"},
		{"gateway region warning", func(x *Topology) { x.Sites[0].Region = "elsewhere" }, LevelWarning, "sites[s1].region", "no Gateway"},
		{"duplicate interface", func(x *Topology) {
			x.Sites[0].Edges[1].Interfaces = append(x.Sites[0].Edges[1].Interfaces, Interface{ID: "ge3", Kind: IfRouted})
		}, LevelError, "interfaces[ge3]", "duplicate interface"},
		{"wan without link", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[0].Link = "" }, LevelError, "interfaces[ge1].link", "needs a link"},
		{"wan unknown link", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[0].Link = "zz" }, LevelError, "interfaces[ge1].link", "unknown link"},
		{"link on lan", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].Link = "l-b1" }, LevelError, "interfaces[ge3].link", "only WAN"},
		{"link shared", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[0].Link = "l-a1" }, LevelError, "interfaces[ge1].link", "already used"},
		{"interface kind", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].Kind = "x" }, LevelError, "kind", "wan, lan, routed or loopback"},
		{"lan needs vlan", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].VLAN = nil }, LevelError, "interfaces[ge3].vlan", "needs a VLAN"},
		{"lan unknown vlan", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].VLAN = ip(99) }, LevelError, "interfaces[ge3].vlan", "neither inherited"},
		{"vlan on routed", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].Kind = IfRouted }, LevelError, "interfaces[ge3].vlan", "only LAN interfaces"},
		{"interface segment unknown", func(x *Topology) { x.Sites[0].Edges[1].Interfaces[1].Segments = []string{"nope"} }, LevelError, "interfaces[ge3].segments", "unknown segment"},
		{"interface segment not in profile", func(x *Topology) {
			x.Profiles[0].Segments = []string{"global"}
			x.Profiles[0].BusinessRules = x.Profiles[0].BusinessRules[:1]
			x.Sites[0].Edges[0].Overrides.BusinessRules = x.Sites[0].Edges[0].Overrides.BusinessRules[:1]
		}, LevelError, "interfaces[ge4].segments", "assumption 12"},
		{"loopback repeated", func(x *Topology) {
			e := &x.Sites[0].Edges[1]
			e.Interfaces = append(e.Interfaces, Interface{ID: "lo1", Kind: IfLoopback}, Interface{ID: "lo1", Kind: IfLoopback})
		}, LevelError, "interfaces[lo1]", "duplicate interface"},
		{"edge vlan duplicates profile vlan", func(x *Topology) { x.Sites[0].Edges[1].VLANs = []VLAN{{ID: 10, Name: "again"}} }, LevelError, "edges[e2].vlans", "duplicate VLAN id 10"},
		{"edge vlan limit", func(x *Topology) {
			var vs []VLAN
			for i := 100; i < 131; i++ {
				vs = append(vs, VLAN{ID: i, Name: "v"})
			}
			x.Sites[0].Edges[1].VLANs = vs
		}, LevelError, "edges[e2].vlans", "limit of 32"},
		{"override unknown setting", func(x *Topology) { x.Sites[0].Edges[1].Overrides.Device = map[string]any{"made.up": 1} }, LevelError, "edges[e2].overrides.device.made.up", "unknown setting"},
		{"override value type", func(x *Topology) {
			x.Sites[0].Edges[1].Overrides.Device = map[string]any{"routing.bfd": map[string]any{"a": 1}}
		}, LevelError, "routing.bfd", "single value"},
		{"override vlan not in profile", func(x *Topology) { x.Sites[0].Edges[1].Overrides.VLANs = []VLANOverride{{ID: 30}} }, LevelError, "overrides.vlans[0].id", "not a VLAN of Profile"},
		{"override vlan name", func(x *Topology) {
			x.Sites[0].Edges[1].Overrides.VLANs = []VLANOverride{{ID: 10, Settings: map[string]any{"name": "x"}}}
		}, LevelError, "settings.name", "cannot be overridden"},
		{"override vlan id", func(x *Topology) {
			x.Sites[0].Edges[1].Overrides.VLANs = []VLANOverride{{ID: 10, Settings: map[string]any{"id": 11}}}
		}, LevelError, "settings.id", "cannot be overridden"},
		{"override vlan twice", func(x *Topology) { x.Sites[0].Edges[1].Overrides.VLANs = []VLANOverride{{ID: 10}, {ID: 10}} }, LevelError, "overrides.vlans[1]", "twice"},
		{"edge rule segment not in profile", func(x *Topology) {
			x.Profiles[1].Segments = []string{"global"}
			x.Sites[0].Edges[1].Profile = "p2"
			x.Sites[0].Edges[1].Overrides.BusinessRules = []BusinessRule{{ID: "e2-r", Segment: "guest"}}
		}, LevelError, "e2-r", "not part of the Profile"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := baseTopology()
			c.mutate(tp)
			var found bool
			for _, is := range Validate(tp) {
				if is.Level == c.level && strings.Contains(is.Path, c.path) && strings.Contains(is.Message, c.msg) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected %s at %q containing %q; got:\n%v", c.level, c.path, c.msg, Validate(tp))
			}
		})
	}
}

// Several problems in one file must all be reported together, never only the first.
func TestAllErrorsAreCollected(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[0].Profile = "ghost"                               // dangling profile
	tp.Sites[0].Edges = append(tp.Sites[0].Edges, tp.Sites[0].Edges[1])  // duplicate edge id
	tp.Sites[0].Edges[1].Overrides.Device = map[string]any{"made.up": 1} // unknown setting
	errs := Errors(Validate(tp))
	want := []string{"unknown profile", "duplicate edge id", "unknown setting"}
	for _, w := range want {
		ok := false
		for _, e := range errs {
			ok = ok || strings.Contains(e.Message, w)
		}
		if !ok {
			t.Errorf("missing %q in %v", w, errs)
		}
	}
}

func TestValidateDoesNotPanicOnEmpty(t *testing.T) {
	if errs := Errors(Validate(&Topology{})); len(errs) == 0 {
		t.Fatal("an empty topology must be reported as invalid")
	}
}

func TestDecodeStrict(t *testing.T) {
	if _, err := Decode([]byte("version: 1\nsurprise: 1\n")); err == nil || !strings.Contains(err.Error(), "surprise") {
		t.Fatalf("unknown field must be an error, got %v", err)
	}
	if _, err := Decode([]byte("")); err == nil {
		t.Fatal("empty file must be an error")
	}
}
