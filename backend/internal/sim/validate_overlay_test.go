package sim

import (
	"strings"
	"testing"
)

func TestShippedDemoTopologyHasNoIssues(t *testing.T) {
	if issues := Validate(demo(t)); len(issues) != 0 {
		t.Fatalf("hub-and-spoke-demo should validate cleanly: %v", issues)
	}
}

func TestExistingTopologiesStillValidWithoutOverlayKeys(t *testing.T) {
	for name, tp := range loadContentTopologies(t) {
		if name == "hub-and-spoke-demo.yaml" {
			continue
		}
		if tp.Clusters != nil || tp.NSDs != nil {
			t.Errorf("%s: step 11 topologies must not need overlay keys", name)
		}
		if errs := Errors(Validate(tp)); len(errs) != 0 {
			t.Errorf("%s: %v", name, errs)
		}
	}
}

func TestOverlayValidationRules(t *testing.T) {
	spoke := func(x *Topology) *CloudVPN { return x.Profiles[2].CloudVPN }
	cases := []struct {
		name   string
		mutate func(*Topology)
		level  string
		path   string
		msg    string
	}{
		{"gateway role", func(x *Topology) { x.Gateways[0].Roles = []string{"wizard"} }, LevelError, "gateways[gw-1].roles", "unknown role"},
		{"mtu above 1500", func(x *Topology) { m := 9000; x.Links[0].MTU = &m }, LevelError, "links[p1-inet].mtu", "jumbo"},
		{"mtu zero", func(x *Topology) { m := 0; x.Links[0].MTU = &m }, LevelError, "links[p1-inet].mtu", "positive"},
		{"mtu below 1300", func(x *Topology) { m := 1200; x.Links[0].MTU = &m }, LevelWarning, "links[p1-inet].mtu", "below 1300"},
		{"negative logical id", func(x *Topology) { n := -1; x.Sites[0].Edges[0].LogicalID = &n }, LevelError, "logical_id", "negative"},
		{"unknown primary gateway", func(x *Topology) { x.Sites[0].Edges[0].Gateways = &GatewayAssignment{Primary: "nope"} }, LevelError, "gateways.primary", "unknown Gateway"},
		{"unknown secondary gateway", func(x *Topology) {
			x.Sites[0].Edges[0].Gateways = &GatewayAssignment{Primary: "gw-1", Secondary: "nope"}
		}, LevelError, "gateways.secondary", "unknown Gateway"},
		{"same primary and secondary", func(x *Topology) {
			x.Sites[0].Edges[0].Gateways = &GatewayAssignment{Primary: "gw-1", Secondary: "gw-1"}
		}, LevelError, "gateways", "must differ"},

		{"duplicate cluster", func(x *Topology) { x.Clusters = append(x.Clusters, x.Clusters[0]) }, LevelError, "clusters[hub-cluster]", "duplicate cluster id"},
		{"cluster id equals edge id", func(x *Topology) { x.Clusters[0].ID = "hub-p-edge" }, LevelError, "clusters[hub-p-edge]", "also an Edge id"},
		{"cluster member not an Edge", func(x *Topology) { x.Clusters[0].Members = []string{"hub-c1", "a-link"} }, LevelError, "clusters[hub-cluster].members", "not an Edge"},
		{"cluster member twice", func(x *Topology) { x.Clusters[0].Members = []string{"hub-c1", "hub-c1"} }, LevelError, "members", "listed twice"},
		{"cluster of one", func(x *Topology) { x.Clusters[0].Members = []string{"hub-c1"} }, LevelWarning, "members", "groups several Edges"},
		{"cluster mixed models", func(x *Topology) { x.Sites[8].Edges[0].Model = "other-model" }, LevelWarning, "members", "same model"},
		{"edge in two clusters", func(x *Topology) {
			x.Clusters = append(x.Clusters, Cluster{ID: "second", Members: []string{"hub-c1", "hub-p-edge"}})
		}, LevelWarning, "clusters[second].members", "already a member"},

		{"unknown hub", func(x *Topology) { spoke(x).BranchToHub.Hubs = []string{"hub-p-edge", "ghost"} }, LevelError, "branch_to_hub.hubs", "neither an Edge nor a cluster"},
		{"hub order duplicated", func(x *Topology) { spoke(x).BranchToHub.Hubs = []string{"hub-p-edge", "hub-p-edge"} }, LevelError, "branch_to_hub.hubs", "Hub order must be unambiguous"},
		{"enabled without hubs", func(x *Topology) { spoke(x).BranchToHub.Hubs = nil }, LevelError, "branch_to_hub", "at least one Hub"},
		{"hubs while disabled", func(x *Topology) { spoke(x).BranchToHub.Enabled = false }, LevelWarning, "branch_to_hub", "Enable Branch to Hubs is off"},
		{"unknown backhaul hub", func(x *Topology) { spoke(x).BranchToHub.BackhaulHubs = []string{"ghost"} }, LevelError, "backhaul_hubs", "neither an Edge nor a cluster"},
		{"b2b via missing", func(x *Topology) { spoke(x).BranchToBranch.Via = "" }, LevelError, "branch_to_branch.via", "must be cloud_gateways or hubs_for_vpn"},
		{"b2b via invalid", func(x *Topology) { spoke(x).BranchToBranch.Via = "mesh" }, LevelError, "branch_to_branch.via", "must be cloud_gateways"},
		{"hubs for vpn needs hubs", func(x *Topology) { spoke(x).BranchToHub = nil }, LevelError, "branch_to_branch", "Hubs for VPN needs Branch to Hub"},
		{"dynamic while b2b off", func(x *Topology) {
			f := true
			x.Profiles[1].CloudVPN.BranchToBranch = &BranchToBranch{Enabled: false, Dynamic: &f}
		}, LevelWarning, "branch_to_branch.dynamic", "not enabled"},
		{"cloud vpn unknown segment", func(x *Topology) { spoke(x).Segments = []string{"nope"} }, LevelError, "cloud_vpn.segments", "unknown segment"},
		{"cloud vpn segment not in profile", func(x *Topology) {
			x.Segments = append(x.Segments, Segment{ID: "extra", Name: "Extra", Type: SegRegular})
			spoke(x).Segments = []string{"extra"}
		}, LevelError, "cloud_vpn.segments", "not part of the Profile"},
		{"cloud vpn segment twice", func(x *Topology) { spoke(x).Segments = []string{"global", "global"} }, LevelError, "cloud_vpn.segments", "listed twice"},

		{"duplicate nsd", func(x *Topology) { x.NSDs = append(x.NSDs, x.NSDs[0]) }, LevelError, "non_sdwan_destinations[azure-vhub]", "duplicate destination id"},
		{"nsd bad via", func(x *Topology) { x.NSDs[0].Via = "cloud" }, LevelError, "non_sdwan_destinations[azure-vhub].via", "must be"},
		{"nsd gateway bad type", func(x *Topology) { x.NSDs[0].Type = "toaster" }, LevelError, ".type", "not a documented via-Gateway type"},
		{"nsd gateway bad mode", func(x *Topology) { x.NSDs[0].Mode = "turbo" }, LevelError, ".mode", "active_active"},
		{"nsd gateway zero tunnels", func(x *Topology) { x.NSDs[0].Tunnels = 0 }, LevelError, ".tunnels", "between 1 and 2"},
		{"azure three tunnels", func(x *Topology) { x.NSDs[0].Tunnels = 3 }, LevelError, ".tunnels", "between 1 and 2"},
		{"cisco asa two tunnels", func(x *Topology) { x.NSDs[0].Type = "cisco_asa"; x.NSDs[0].Tunnels = 2 }, LevelError, ".tunnels", "between 1 and 1"},
		{"active active five tunnels", func(x *Topology) {
			x.NSDs[0].Type = "palo_alto"
			x.NSDs[0].Mode = ModeActiveActive
			x.NSDs[0].Tunnels = 5
		}, LevelError, ".tunnels", "between 1 and 4"},
		{"hot standby three tunnels", func(x *Topology) { x.NSDs[0].Type = "palo_alto"; x.NSDs[0].Tunnels = 3 }, LevelError, ".tunnels", "between 1 and 2"},
		{"nsd gateway missing", func(x *Topology) { x.NSDs[0].Gateway = "" }, LevelError, ".gateway", "required"},
		{"nsd unknown gateway", func(x *Topology) { x.NSDs[0].Gateway = "gw-9" }, LevelError, ".gateway", "unknown Gateway"},
		{"nsd gateway lacks role", func(x *Topology) { x.NSDs[0].Gateway = "gw-2" }, LevelError, ".gateway", "lacks the secure_vpn role"},
		{"nsd gateway with edge field", func(x *Topology) { x.NSDs[0].Edge = "spoke-1-edge" }, LevelError, ".edge", "only to a via-Edge"},
		{"nsd edge bad type", func(x *Topology) { x.NSDs[1].Type = "azure_virtual_hub" }, LevelError, ".type", "not a documented via-Edge type"},
		{"nsd edge two tunnels", func(x *Topology) { x.NSDs[1].Tunnels = 2 }, LevelError, ".tunnels", "one tunnel"},
		{"nsd edge with mode", func(x *Topology) { x.NSDs[1].Mode = ModeActiveActive }, LevelError, ".mode", "only to a via-Gateway"},
		{"nsd edge with gateway", func(x *Topology) { x.NSDs[1].Gateway = "gw-1" }, LevelError, ".gateway", "only to a via-Gateway"},
		{"nsd edge missing edge", func(x *Topology) { x.NSDs[1].Edge = "" }, LevelError, ".edge", "required"},
		{"nsd edge unknown edge", func(x *Topology) { x.NSDs[1].Edge = "ghost" }, LevelError, ".edge", "unknown Edge"},
		{"nsd edge unknown segment", func(x *Topology) { x.NSDs[1].Segment = "nope" }, LevelError, ".segment", "unknown segment"},
		{"one nsd service per edge and segment", func(x *Topology) {
			x.NSDs = append(x.NSDs, NSD{ID: "second", Via: NSDViaEdge, Type: "generic_ikev1", Edge: "spoke-2-edge"})
		}, LevelError, "non_sdwan_destinations[second]", "one service per segment"},

		{"nsd via gateway unknown", func(x *Topology) { spoke(x).NSDViaGateway = []string{"ghost"} }, LevelError, "nsd_via_gateway", "unknown Non SD-WAN destination"},
		{"nsd via gateway names an edge destination", func(x *Topology) { spoke(x).NSDViaGateway = []string{"aws-vpn"} }, LevelError, "nsd_via_gateway", "not a via-Gateway destination"},
		{"nsd via edge unknown", func(x *Topology) { spoke(x).NSDViaEdge = []string{"ghost"} }, LevelError, "nsd_via_edge", "unknown Non SD-WAN destination"},
		{"nsd via edge names a gateway destination", func(x *Topology) { spoke(x).NSDViaEdge = []string{"azure-vhub"} }, LevelError, "nsd_via_edge", "not a via-Edge destination"},
		{"nsd via gateway in an Edge override", func(x *Topology) {
			x.Sites[3].Edges[0].Overrides.CloudVPN = &CloudVPN{NSDViaGateway: []string{"azure-vhub"}}
		}, LevelError, "overrides.cloud_vpn.nsd_via_gateway", "Profile level only"},
		{"override hub list unknown entry", func(x *Topology) {
			x.Sites[3].Edges[0].Overrides.CloudVPN = &CloudVPN{BranchToHub: &BranchToHub{Enabled: true, Hubs: []string{"ghost"}}}
		}, LevelError, "overrides.cloud_vpn.branch_to_hub.hubs", "neither an Edge nor a cluster"},
		{"override segment outside profile", func(x *Topology) { x.Sites[3].Edges[0].Overrides.CloudVPN = &CloudVPN{Segments: []string{"nope"}} }, LevelError, "overrides.cloud_vpn.segments", "unknown segment"},
		{"via gateway without cloud vpn", func(x *Topology) { x.Profiles[3].CloudVPN = &CloudVPN{NSDViaGateway: []string{"azure-vhub"}} }, LevelWarning, "nsd_via_gateway", "needs Cloud VPN on"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := demo(t)
			c.mutate(tp)
			found := false
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

// The three breakages named in the step 12 brief, together: all must be reported at once.
func TestOverlayErrorsAreCollectedTogether(t *testing.T) {
	tp := demo(t)
	tp.Profiles[2].CloudVPN.BranchToHub.Hubs = []string{"hub-p-edge", "hub-p-edge"} // hub order duplicated
	tp.Clusters[0].Members = []string{"hub-c1", "azure-vhub"}                       // member that is not an Edge
	tp.NSDs[0].Gateway = "gw-missing"                                               // NSD pointing at a missing Gateway
	errs := Errors(Validate(tp))
	for _, want := range []string{"Hub order must be unambiguous", "is not an Edge of this topology", `unknown Gateway "gw-missing"`} {
		ok := false
		for _, e := range errs {
			ok = ok || strings.Contains(e.Message, want)
		}
		if !ok {
			t.Errorf("missing %q in %v", want, errs)
		}
	}
	if len(errs) != 3 {
		t.Errorf("exactly three errors expected, got %d: %v", len(errs), errs)
	}
}
