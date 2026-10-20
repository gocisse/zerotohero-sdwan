package sim

import (
	"strings"
	"testing"
)

func TestApplyPatches(t *testing.T) {
	tp := rdemo(t)
	before := jsonOf(t, tp)
	cases := []struct {
		name    string
		patches []Patch
		check   func(*testing.T, *Topology)
		err     string
	}{
		{"set a nested bool, creating missing objects", []Patch{{Op: "set", Path: "edges[b-edge]/routing/bgp/redistribute/overlay_prefix", Value: true}}, func(t *testing.T, n *Topology) {
			rd := edgeOf(n, "b-edge").Routing.BGP.Redistribute
			if rd == nil || rd.OverlayPrefix == nil || !*rd.OverlayPrefix {
				t.Errorf("%+v", rd)
			}
		}, ""},
		{"remove an array element by id", []Patch{{Op: "remove", Path: "edges[c-edge]/routing/static_routes[printers]"}}, func(t *testing.T, n *Topology) {
			if len(edgeOf(n, "c-edge").Routing.StaticRoutes) != 0 {
				t.Error("static route survived")
			}
		}, ""},
		{"set a field of an element picked by id", []Patch{{Op: "set", Path: "edges[c-edge]/routing/static_routes[printers]/advertise", Value: false}}, func(t *testing.T, n *Topology) {
			if edgeOf(n, "c-edge").Routing.StaticRoutes[0].Advertise {
				t.Error("advertise still on")
			}
		}, ""},
		{"append to a list", []Patch{{Op: "append", Path: "edges[a-edge]/routing/lan_routes", Value: map[string]any{"id": "x", "prefix": "10.88.0.0/24", "protocol": "ospf", "via": "ge3", "ospf_type": "intra"}}}, func(t *testing.T, n *Topology) {
			if len(edgeOf(n, "a-edge").Routing.LANRoutes) != 4 {
				t.Error("not appended")
			}
		}, ""},
		{"a root list by id", []Patch{{Op: "set", Path: "profiles[branch-ospf]/routing/ospf/enabled", Value: false}}, func(t *testing.T, n *Topology) {
			if profileOf(n, "branch-ospf").Routing.OSPF.Enabled {
				t.Error("still enabled")
			}
		}, ""},
		{"unknown field", []Patch{{Op: "set", Path: "edges[b-edge]/routing/bgp/nonsense", Value: 1}}, nil, "not a valid document"},
		{"wrong type", []Patch{{Op: "set", Path: "edges[b-edge]/routing/bgp/local_asn", Value: "x"}}, nil, "not a valid document"},
		{"unknown id", []Patch{{Op: "set", Path: "edges[ghost]/name", Value: "x"}}, nil, "no Edge"},
		{"unknown element", []Patch{{Op: "remove", Path: "edges[c-edge]/routing/static_routes[ghost]"}}, nil, "no element"},
		{"unknown op", []Patch{{Op: "explode", Path: "edges[c-edge]/name"}}, nil, "unknown op"},
		{"remove a missing key", []Patch{{Op: "remove", Path: "edges[c-edge]/overrides/cloud_vpn"}}, nil, "does not exist"},
		{"bad path", []Patch{{Op: "set", Path: "edges[c-edge]//name", Value: "x"}}, nil, "empty path segment"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := ApplyPatches(tp, c.patches)
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) {
					t.Fatalf("want error %q, got %v", c.err, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c.check(t, n)
			if jsonOf(t, tp) != before {
				t.Error("the original topology was mutated")
			}
		})
	}
}
