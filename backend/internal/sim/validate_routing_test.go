package sim

import (
	"strings"
	"testing"
)

func rdemo(t *testing.T) *Topology {
	t.Helper()
	tp := loadContentTopologies(t)["routing-demo.yaml"]
	if tp == nil {
		t.Fatal("routing-demo.yaml missing")
	}
	return tp
}

func edgeOf(tp *Topology, id string) *Edge {
	for si := range tp.Sites {
		for ei := range tp.Sites[si].Edges {
			if tp.Sites[si].Edges[ei].ID == id {
				return &tp.Sites[si].Edges[ei]
			}
		}
	}
	return nil
}

func profileOf(tp *Topology, id string) *Profile {
	for i := range tp.Profiles {
		if tp.Profiles[i].ID == id {
			return &tp.Profiles[i]
		}
	}
	return nil
}

func TestShippedRoutingDemoHasNoIssues(t *testing.T) {
	if issues := Validate(rdemo(t)); len(issues) != 0 {
		t.Fatalf("routing-demo should validate cleanly: %v", issues)
	}
}

func TestEarlierTopologiesNeedNoRoutingKeys(t *testing.T) {
	for name, tp := range loadContentTopologies(t) {
		if name == "routing-demo.yaml" {
			continue
		}
		if tp.OverlayFlowControl != nil {
			t.Errorf("%s: older topologies must not need routing keys", name)
		}
		for _, p := range tp.Profiles {
			if p.Routing != nil && name != "hub-and-spoke-demo.yaml" {
				t.Errorf("%s: profile %s has routing keys", name, p.ID)
			}
		}
		if errs := Errors(Validate(tp)); len(errs) != 0 {
			t.Errorf("%s: %v", name, errs)
		}
	}
}

func TestRoutingValidationRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Topology)
		level  string
		path   string
		msg    string
	}{
		{"valid is silent", func(x *Topology) {}, "", "", ""},
		{"static prefix syntax", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Subnet = "172.16.50/24" }, LevelError, "static_routes[0].subnet", "not a prefix"},
		{"static host bits", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Subnet = "172.16.50.7/24" }, LevelError, "static_routes[0].subnet", "host bits"},
		{"static ipv6", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Subnet = "2001:db8::/32" }, LevelError, "static_routes[0].subnet", "IPv6"},
		{"static duplicate id", func(x *Topology) {
			r := edgeOf(x, "c-edge").Routing
			r.StaticRoutes = append(r.StaticRoutes, r.StaticRoutes[0])
		}, LevelError, "static_routes[1].id", "duplicate"},
		{"static cost range", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Cost = ip(300) }, LevelError, "static_routes[0].cost", "outside 0 to 255"},
		{"static next hop outside connected", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].NextHop = "192.0.2.9" }, LevelError, "static_routes[0].next_hop", "not inside a connected subnet"},
		{"static LAN route without next hop", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].NextHop = "" }, LevelError, "static_routes[0].next_hop", "needs a next hop"},
		{"static interface not WAN", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Interface = "ge3" }, LevelError, "static_routes[0].interface", "not a WAN interface"},
		{"static unknown segment", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Segment = "nope" }, LevelError, "static_routes[0].segment", "unknown segment"},
		{"lan route unknown via (ospf)", func(x *Topology) { edgeOf(x, "a-edge").Routing.LANRoutes[0].Via = "ge9" }, LevelError, "lan_routes[0].via", "not enabled"},
		{"lan route unknown neighbor (bgp)", func(x *Topology) { edgeOf(x, "b-edge").Routing.LANRoutes[0].Via = "ghost" }, LevelError, "lan_routes[0].via", "does not exist"},
		{"lan route bad ospf type", func(x *Topology) { edgeOf(x, "a-edge").Routing.LANRoutes[0].OSPFType = "n1" }, LevelError, "lan_routes[0].ospf_type", "intra, inter, e1 or e2"},
		{"lan route bgp attrs on ospf", func(x *Topology) { edgeOf(x, "a-edge").Routing.LANRoutes[0].BGP = &BGPAttrs{} }, LevelError, "lan_routes[0].bgp", "do not apply"},
		{"lan route bad protocol", func(x *Topology) { edgeOf(x, "a-edge").Routing.LANRoutes[0].Protocol = "rip" }, LevelError, "lan_routes[0].protocol", "ospf or bgp"},
		{"lan route duplicate id", func(x *Topology) { edgeOf(x, "a-edge").Routing.LANRoutes[1].ID = "a-lan" }, LevelError, "lan_routes[1].id", "duplicate"},
		{"ospf full on switched interface", func(x *Topology) {
			e := edgeOf(x, "a-edge")
			e.Interfaces[1].Kind = IfLan
			e.Interfaces[1].VLAN = ip(10)
		}, LevelError, "ospf.interfaces[0].mode", "supports OSPF only as passive"},
		{"ospf on WAN interface", func(x *Topology) { edgeOf(x, "a-edge").Routing.OSPF.Interfaces[0].ID = "ge1" }, LevelError, "ospf.interfaces[0].id", "not modelled"},
		{"ospf interface unknown", func(x *Topology) { edgeOf(x, "a-edge").Routing.OSPF.Interfaces[0].ID = "ge8" }, LevelError, "ospf.interfaces[0].id", "unknown interface"},
		{"ospf not activated in the profile", func(x *Topology) { profileOf(x, "branch-ospf").Routing.OSPF.Enabled = false }, LevelError, "edges[a-edge].routing.ospf", "activated in the Profile first"},
		{"ospf area unknown", func(x *Topology) { edgeOf(x, "a-edge").Routing.OSPF.Interfaces[0].Area = "9" }, LevelError, "ospf.interfaces[0].area", "not in the Profile"},
		{"ospf passive route", func(x *Topology) { edgeOf(x, "a-edge").Routing.OSPF.Interfaces[0].Mode = OSPFPassive }, LevelWarning, "lan_routes[0].via", "passive"},
		{"ospf non-global segment", func(x *Topology) {
			x.Segments = append(x.Segments, Segment{ID: "guest", Name: "Guest", Type: SegRegular})
			profileOf(x, "branch-ospf").Segments = []string{"global", "guest"}
			edgeOf(x, "a-edge").Routing.LANRoutes[0].Segment = "guest"
		}, LevelError, "lan_routes[0].segment", "Global Segment"},
		{"ospf no areas", func(x *Topology) { profileOf(x, "branch-ospf").Routing.OSPF.Areas = nil }, LevelError, "ospf.areas", "no area"},
		{"ospf duplicate area", func(x *Topology) {
			o := profileOf(x, "branch-ospf").Routing.OSPF
			o.Areas = append(o.Areas, o.Areas[0])
		}, LevelError, "areas[1].id", "duplicate area"},
		{"ospf summary syntax", func(x *Topology) { profileOf(x, "branch-ospf").Routing.OSPF.Summaries[0].Subnet = "10.60.0.0/33" }, LevelError, "summaries[0].subnet", "not a prefix"},
		{"ospf summary tag", func(x *Topology) { profileOf(x, "branch-ospf").Routing.OSPF.Summaries[0].Tag = ip(0) }, LevelError, "summaries[0].tag", "outside 1 to 4294967295"},
		{"ospf default route value", func(x *Topology) { profileOf(x, "branch-ospf").Routing.OSPF.Redistribute.DefaultRoute = "e3" }, LevelError, "redistribute.default_route", "none, e1 or e2"},
		{"bgp duplicate neighbor", func(x *Topology) {
			b := edgeOf(x, "b-edge").Routing.BGP
			n := b.Neighbors[0]
			n.ID = "lan-r2"
			b.Neighbors = append(b.Neighbors, n)
		}, LevelError, "neighbors[1].ip", "duplicate neighbor"},
		{"bgp duplicate neighbor id", func(x *Topology) {
			b := edgeOf(x, "b-edge").Routing.BGP
			n := b.Neighbors[0]
			n.IP = "10.30.255.3"
			b.Neighbors = append(b.Neighbors, n)
		}, LevelError, "neighbors[1].id", "duplicate neighbor id"},
		{"bgp unknown filter", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Neighbors[0].OutboundFilter = "ghost" }, LevelError, "neighbors[0].outbound_filter", "unknown filter"},
		{"bgp type contradicts asn", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Neighbors[0].Type = BGPInternal }, LevelError, "neighbors[0].type", "contradicts the ASNs"},
		{"bgp asn zero", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.LocalASN = 0 }, LevelError, "bgp.local_asn", "ASN"},
		{"bgp neighbor ip", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Neighbors[0].IP = "10.30.255" }, LevelError, "neighbors[0].ip", "not an IPv4"},
		{"bgp filter action", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Filters[0].Rules[0].Action = "reject" }, LevelError, "rules[0].action", "permit or deny"},
		{"bgp filter prefix", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Filters[0].Rules[0].Match.Prefix = "10.99.0.0" }, LevelError, "rules[0].match.prefix", "not a prefix"},
		{"bgp filter without rules", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Filters[0].Rules = nil }, LevelError, "filters[0].rules", "at least one rule"},
		{"bgp unused filter", func(x *Topology) { edgeOf(x, "b-edge").Routing.BGP.Neighbors[0].InboundFilter = "" }, LevelWarning, "filters[from-lan]", "no neighbor uses"},
		{"bgp default route value", func(x *Topology) {
			edgeOf(x, "b-edge").Routing.BGP.Redistribute = &BGPRedistribute{DefaultRoute: "maybe"}
		}, LevelError, "redistribute.default_route", "no, yes or conditional"},
		{"bgp iBGP to NSD sets local preference outbound", func(x *Topology) {
			x.NSDs = append(x.NSDs, NSD{ID: "nsd1", Via: NSDViaEdge, Type: "generic_ikev2", Edge: "b-edge"})
			b := edgeOf(x, "b-edge").Routing.BGP
			b.Neighbors[0].ASN = b.LocalASN
			b.Neighbors[0].NSD = "nsd1"
			b.Neighbors[0].OutboundFilter = "from-lan"
		}, LevelError, "neighbors[0].outbound_filter", "cannot apply local preference"},
		{"bgp lan route segment differs from neighbor", func(x *Topology) {
			x.Segments = append(x.Segments, Segment{ID: "guest", Name: "Guest", Type: SegRegular})
			profileOf(x, "branch-bgp").Segments = []string{"global", "guest"}
			edgeOf(x, "b-edge").Routing.LANRoutes[0].Segment = "guest"
		}, LevelError, "lan_routes[0].segment", "neighbor"},
		{"ofc exit not an edge", func(x *Topology) {
			x.OverlayFlowControl = &OverlayFlowControl{PreferredExits: []PreferredExits{{Prefix: "10.40.0.0/24", Exits: []string{"a-edge", "ghost"}}}}
		}, LevelError, "preferred_exits[0].exits", "not an Edge"},
		{"ofc exits listed twice", func(x *Topology) {
			x.OverlayFlowControl = &OverlayFlowControl{PreferredExits: []PreferredExits{{Prefix: "10.40.0.0/24", Exits: []string{"a-edge", "a-edge"}}}}
		}, LevelError, "preferred_exits[0].exits", "listed twice"},
		{"ofc prefix listed twice", func(x *Topology) {
			pe := PreferredExits{Prefix: "10.40.0.0/24", Exits: []string{"a-edge"}}
			x.OverlayFlowControl = &OverlayFlowControl{PreferredExits: []PreferredExits{pe, pe}}
		}, LevelError, "preferred_exits[1].prefix", "listed twice"},
		{"ofc pin exit", func(x *Topology) {
			x.OverlayFlowControl = &OverlayFlowControl{Pins: []RoutePin{{Prefix: "10.40.0.0/24", Exit: "ghost"}}}
		}, LevelError, "pins[0].exit", "not an Edge"},
		{"nsd prefix duplicate", func(x *Topology) {
			x.NSDs = append(x.NSDs, NSD{ID: "nsd1", Via: NSDViaEdge, Type: "generic_ikev2", Edge: "b-edge", Prefixes: []NSDPrefix{{Prefix: "10.9.0.0/24"}, {Prefix: "10.9.0.0/24"}}})
		}, LevelError, "prefixes[1].prefix", "duplicate prefix"},
		{"interface address syntax", func(x *Topology) { edgeOf(x, "a-edge").Interfaces[1].Address = "10.20.255.1" }, LevelError, "interfaces[ge3].address", "prefix length"},
		{"interface subnets overlap", func(x *Topology) { edgeOf(x, "c-edge").Interfaces[3].Address = "10.60.0.9/23" }, LevelWarning, "interfaces[ge5].address", "overlaps"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := rdemo(t)
			c.mutate(tp)
			issues := Validate(tp)
			if c.level == "" {
				if len(issues) != 0 {
					t.Fatalf("unexpected issues: %v", issues)
				}
				return
			}
			for _, is := range issues {
				if is.Level == c.level && strings.Contains(is.Path, c.path) && strings.Contains(is.Message, c.msg) {
					return
				}
			}
			t.Errorf("expected %s at %q containing %q; got:\n%v", c.level, c.path, c.msg, issues)
		})
	}
}

// The three breakages named in the step 13 brief, together: all must be reported at once.
func TestRoutingErrorsAreCollectedTogether(t *testing.T) {
	tp := rdemo(t)
	b := edgeOf(tp, "b-edge").Routing.BGP
	n := b.Neighbors[0]
	n.ID = "lan-dup"
	b.Neighbors = append(b.Neighbors, n) // a duplicated neighbor
	a := edgeOf(tp, "a-edge")            // OSPF full mode on a switched interface
	a.Interfaces[1].Kind, a.Interfaces[1].VLAN = IfLan, ip(10)
	b.Neighbors[0].InboundFilter = "ghost-filter" // a filter that does not exist
	errs := Errors(Validate(tp))
	for _, want := range []string{"duplicate neighbor", "supports OSPF only as passive", `unknown filter "ghost-filter"`} {
		ok := false
		for _, e := range errs {
			ok = ok || strings.Contains(e.Message, want)
		}
		if !ok {
			t.Errorf("missing %q in %v", want, errs)
		}
	}
}

func TestRoutingConditionsAreValidated(t *testing.T) {
	tp := rdemo(t)
	for _, c := range []Conditions{{NeighborsDown: []string{"a-edge/ghost"}}, {ProbesDown: []string{"c-edge/ghost"}}, {ProbesDown: []string{"a-edge/a-lan"}}} {
		if _, err := ComputeRoutes(tp, c, DefaultRoutingAssumptions(), nil); err == nil {
			t.Errorf("conditions %+v should be rejected", c)
		}
	}
	bad := DefaultRoutingAssumptions()
	bad.FilterOrder = "random"
	if _, err := ComputeRoutes(tp, Conditions{}, bad, nil); err == nil {
		t.Error("invalid assumptions should be rejected")
	}
}
