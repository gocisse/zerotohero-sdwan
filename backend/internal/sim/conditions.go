package sim

import "time"

// Clock supplies "now" to the dynamic tunnel lifecycle. Tests inject a FakeClock; the API uses SystemClock.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// SystemClock returns the real clock.
func SystemClock() Clock { return systemClock{} }

// FakeClock is a controllable clock for tests and for the timeline player.
type FakeClock struct{ T time.Time }

// Now returns the fake time.
func (c *FakeClock) Now() time.Time { return c.T }

// Advance moves the fake time forward.
func (c *FakeClock) Advance(d time.Duration) { c.T = c.T.Add(d) }

// Build decision modes for the "secure tunnel possible" stub.
const (
	BuildAuto   = "auto"
	BuildAlways = "always"
	BuildNever  = "never"
)

// TrafficSpec says that Edge A has been sending to Edge B: the first packet and the latest packet, as seconds
// before "now". It is how a stateless request tells the engine about a dynamic tunnel's history.
type TrafficSpec struct {
	A               string  `yaml:"a" json:"a"`
	B               string  `yaml:"b" json:"b"`
	FirstPacketAgoS float64 `yaml:"first_packet_ago_s" json:"first_packet_ago_s"`
	LastPacketAgoS  float64 `yaml:"last_packet_ago_s" json:"last_packet_ago_s"`
}

// Conditions is an overlay on the authored topology: nothing in it ever modifies the topology. It arrives in the
// request body, so the API stays stateless.
type Conditions struct {
	// LinksDown and LinksDegraded name WAN links (ids from the topology's links list).
	LinksDown     []string `yaml:"links_down" json:"links_down,omitempty"`
	LinksDegraded []string `yaml:"links_degraded" json:"links_degraded,omitempty"`
	// HubsUnreachable names Edges (typically Hubs) that peers cannot reach.
	HubsUnreachable     []string `yaml:"hubs_unreachable" json:"hubs_unreachable,omitempty"`
	GatewaysUnreachable []string `yaml:"gateways_unreachable" json:"gateways_unreachable,omitempty"`
	NSDsUnreachable     []string `yaml:"nsds_unreachable" json:"nsds_unreachable,omitempty"`
	// OrchestratorUnreachable: what Edges do then is not documented, so tunnel states become "unknown".
	OrchestratorUnreachable bool `yaml:"orchestrator_unreachable" json:"orchestrator_unreachable,omitempty"`
	// DynamicBuild forces the build decision stub: "auto" (default), "always" or "never" (assumption 65).
	DynamicBuild string        `yaml:"dynamic_build" json:"dynamic_build,omitempty"`
	Traffic      []TrafficSpec `yaml:"traffic" json:"traffic,omitempty"`
	// NeighborsDown names LAN neighbors that are not Full (OSPF) or Established (BGP), as "<edge>/<interface or
	// neighbor id>" (step 13). ProbesDown names static routes whose ICMP probe fails, as "<edge>/<route id>".
	NeighborsDown []string `yaml:"neighbors_down" json:"neighbors_down,omitempty"`
	ProbesDown    []string `yaml:"probes_down" json:"probes_down,omitempty"`
}

func setOf(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// validateConditions reports ids in the conditions that the topology does not have.
func (t *Topology) validateConditions(c Conditions) []string {
	var bad []string
	links := map[string]bool{}
	for _, l := range t.Links {
		links[l.ID] = true
	}
	edges := map[string]bool{}
	for _, s := range t.Sites {
		for _, e := range s.Edges {
			edges[e.ID] = true
		}
	}
	gws := map[string]bool{}
	for _, g := range t.Gateways {
		gws[g.ID] = true
	}
	nsds := map[string]bool{}
	for _, n := range t.NSDs {
		nsds[n.ID] = true
	}
	for _, id := range c.LinksDown {
		if !links[id] {
			bad = append(bad, "unknown link "+id)
		}
	}
	for _, id := range c.LinksDegraded {
		if !links[id] {
			bad = append(bad, "unknown link "+id)
		}
	}
	for _, id := range c.HubsUnreachable {
		if !edges[id] {
			bad = append(bad, "unknown Edge "+id)
		}
	}
	for _, id := range c.GatewaysUnreachable {
		if !gws[id] {
			bad = append(bad, "unknown Gateway "+id)
		}
	}
	for _, id := range c.NSDsUnreachable {
		if !nsds[id] {
			bad = append(bad, "unknown Non SD-WAN destination "+id)
		}
	}
	switch c.DynamicBuild {
	case "", BuildAuto, BuildAlways, BuildNever:
	default:
		bad = append(bad, "dynamic_build must be auto, always or never")
	}
	for _, tr := range c.Traffic {
		if !edges[tr.A] || !edges[tr.B] || tr.A == tr.B {
			bad = append(bad, "traffic needs two different Edges of the topology")
		}
		if tr.FirstPacketAgoS < tr.LastPacketAgoS || tr.LastPacketAgoS < 0 {
			bad = append(bad, "traffic: first_packet_ago_s must be at least last_packet_ago_s, and both non-negative")
		}
	}
	return bad
}

// ConditionError is returned for conditions that name things the topology does not have.
type ConditionError struct{ Problems []string }

func (e *ConditionError) Error() string {
	s := "invalid conditions:"
	for _, p := range e.Problems {
		s += " " + p + ";"
	}
	return s
}
