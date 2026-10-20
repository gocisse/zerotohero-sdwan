package sim

// validateRoutingConditions reports routing conditions that name things the topology does not have.
func (t *Topology) validateRoutingConditions(c Conditions) []string {
	var bad []string
	nb := map[string]bool{}
	probe := map[string]bool{}
	for _, s := range t.Sites {
		for _, e := range s.Edges {
			if e.Routing == nil {
				continue
			}
			if e.Routing.OSPF != nil {
				for _, in := range e.Routing.OSPF.Interfaces {
					nb[e.ID+"/"+in.ID] = true
				}
			}
			if e.Routing.BGP != nil {
				for _, n := range e.Routing.BGP.Neighbors {
					nb[e.ID+"/"+n.ID] = true
				}
			}
			for _, r := range e.Routing.StaticRoutes {
				if r.ICMPProbe {
					probe[e.ID+"/"+r.ID] = true
				}
			}
		}
	}
	for _, id := range c.NeighborsDown {
		if !nb[id] {
			bad = append(bad, "unknown neighbor "+id+" (use <edge>/<OSPF interface or BGP neighbor id>)")
		}
	}
	for _, id := range c.ProbesDown {
		if !probe[id] {
			bad = append(bad, "no static route with an ICMP probe "+id+" (use <edge>/<route id>)")
		}
	}
	return bad
}
