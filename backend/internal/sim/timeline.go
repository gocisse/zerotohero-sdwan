package sim

import (
	"fmt"
	"sort"
	"time"
)

// Event types for Timeline.
const (
	EvTrafficStart = "traffic_start"
	EvTrafficStop  = "traffic_stop"
	EvLinkDown     = "link_down"
	EvLinkUp       = "link_up"
	EvLinkDegraded = "link_degraded"
	EvLinkRestored = "link_restored"
	EvHubDown      = "hub_down"
	EvHubUp        = "hub_up"
	EvGatewayDown  = "gateway_down"
	EvGatewayUp    = "gateway_up"
	EvFlushFlows   = "flush_flows"
)

// Event is something that happens at a time offset (seconds from the start of the timeline).
type Event struct {
	AtS    float64 `json:"at_s"`
	Type   string  `json:"type"`
	A      string  `json:"a,omitempty"` // traffic: the initiating Edge
	B      string  `json:"b,omitempty"` // traffic: the peer
	Target string  `json:"target,omitempty"`
}

// TimelineError is returned for events the topology cannot take.
type TimelineError struct{ Message string }

func (e *TimelineError) Error() string { return e.Message }

// TunnelStateEntry is one tunnel's state in a frame.
type TunnelStateEntry struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	ReasonCode string `json:"reason_code"`
}

// TunnelChange is a tunnel whose state differs from the previous frame.
type TunnelChange struct {
	ID         string `json:"id"`
	From       string `json:"from"`
	To         string `json:"to"`
	ReasonCode string `json:"reason_code"`
	Reason     string `json:"reason"`
}

// FailoverState tracks, per Spoke, where local flows and peer flows go after Hub events (documented rows only).
type FailoverState struct {
	Spoke       string        `json:"spoke"`
	Hubs        []string      `json:"hubs"`
	LocalHub    string        `json:"local_hub"`
	LocalNote   string        `json:"local_note,omitempty"`
	PeerHub     string        `json:"peer_hub"`
	PeerNote    string        `json:"peer_note,omitempty"`
	Basis       Basis         `json:"basis"`
	NotModelled []NotModelled `json:"not_modelled,omitempty"`
}

// FirstPacket is the path the first packets of a new dynamic tunnel take: the standing path.
type FirstPacket struct {
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Tunnels []string `json:"tunnel_ids"`
	Reason  string   `json:"reason"`
	Basis   Basis    `json:"basis"`
}

// Frame is the state of the overlay at one moment of the timeline.
type Frame struct {
	AtS         float64            `json:"at_s"`
	Label       string             `json:"label"`
	Cause       string             `json:"cause"` // event or derived
	Event       *Event             `json:"event,omitempty"`
	States      []TunnelStateEntry `json:"states"`
	Changes     []TunnelChange     `json:"changes"`
	Failovers   []FailoverState    `json:"failovers"`
	FirstPacket *FirstPacket       `json:"first_packet,omitempty"`
	Basis       []Basis            `json:"basis,omitempty"`
}

// TimelineResult is the tunnel states over time. Tunnels holds the definitions (from the first frame).
type TimelineResult struct {
	Topology    string        `json:"topology"`
	Tunnels     []Tunnel      `json:"tunnels"`
	Frames      []Frame       `json:"frames"`
	NotModelled []NotModelled `json:"not_modelled"`
}

type pairTraffic struct {
	A, B           string
	First, Last    float64
	Active, Exists bool
}

// Timeline runs a sequence of timed events against the topology. It uses a virtual clock (the offsets), never the
// real one, and never changes the topology: link, Hub and Gateway events overlay the base conditions.
func Timeline(t *Topology, base Conditions, events []Event) (*TimelineResult, error) {
	ev := append([]Event{}, events...)
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].AtS < ev[j].AtS })
	c0, err := newOverlayCtx(t, Conditions{DynamicBuild: base.DynamicBuild})
	if err != nil {
		return nil, err
	}
	for _, e := range ev {
		if e.AtS < 0 {
			return nil, &TimelineError{"event times must not be negative"}
		}
		switch e.Type {
		case EvTrafficStart, EvTrafficStop:
			if c0.edgeByID[e.A] == nil || c0.edgeByID[e.B] == nil || e.A == e.B {
				return nil, &TimelineError{fmt.Sprintf("%s needs two different Edges of the topology", e.Type)}
			}
		case EvLinkDown, EvLinkUp, EvLinkDegraded, EvLinkRestored:
			if c0.linkByID[e.Target] == nil {
				return nil, &TimelineError{fmt.Sprintf("%s: unknown link %q", e.Type, e.Target)}
			}
		case EvHubDown, EvHubUp:
			if c0.edgeByID[e.Target] == nil {
				return nil, &TimelineError{fmt.Sprintf("%s: unknown Edge %q", e.Type, e.Target)}
			}
		case EvGatewayDown, EvGatewayUp:
			if c0.gwByID[e.Target] == nil {
				return nil, &TimelineError{fmt.Sprintf("%s: unknown Gateway %q", e.Type, e.Target)}
			}
		case EvFlushFlows:
			if e.Target != "" && c0.edgeByID[e.Target] == nil {
				return nil, &TimelineError{fmt.Sprintf("flush_flows: unknown Edge %q", e.Target)}
			}
		default:
			return nil, &TimelineError{fmt.Sprintf("unknown event type %q", e.Type)}
		}
	}

	// Derived moments: 180 s after a traffic_stop with no new traffic in between.
	type moment struct {
		at      float64
		ev      *Event
		derived string
		pair    [2]string
	}
	var moments []moment
	for i := range ev {
		e := &ev[i]
		moments = append(moments, moment{at: e.AtS, ev: e})
		if e.Type == EvTrafficStop {
			restart := false
			for j := i + 1; j < len(ev); j++ {
				if ev[j].Type == EvTrafficStart && samePair(ev[j].A, ev[j].B, e.A, e.B) && ev[j].AtS < e.AtS+TeardownAfter.Seconds() {
					restart = true
				}
			}
			if !restart {
				moments = append(moments, moment{at: e.AtS + TeardownAfter.Seconds(), derived: "silence", pair: [2]string{e.A, e.B}})
			}
		}
	}
	sort.SliceStable(moments, func(i, j int) bool { return moments[i].at < moments[j].at })

	cond := Conditions{LinksDown: append([]string{}, base.LinksDown...), LinksDegraded: append([]string{}, base.LinksDegraded...),
		HubsUnreachable: append([]string{}, base.HubsUnreachable...), GatewaysUnreachable: append([]string{}, base.GatewaysUnreachable...),
		NSDsUnreachable: base.NSDsUnreachable, OrchestratorUnreachable: base.OrchestratorUnreachable, DynamicBuild: base.DynamicBuild}
	traffic := map[string]*pairTraffic{}
	clock := &FakeClock{T: time.Unix(0, 0).UTC()}
	out := &TimelineResult{Topology: t.Meta.ID, Frames: []Frame{}, NotModelled: []NotModelled{}}
	var prev map[string]TunnelStateEntry
	fo := newFailoverTracker(c0)

	build := func(at float64, label, cause string, e *Event, first *FirstPacket, basis []Basis) error {
		cc := cond
		cc.Traffic = nil
		keys := make([]string, 0, len(traffic))
		for k := range traffic {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := traffic[k]
			if !p.Exists {
				continue
			}
			last := 0.0
			if !p.Active {
				last = at - p.Last
			}
			cc.Traffic = append(cc.Traffic, TrafficSpec{A: p.A, B: p.B, FirstPacketAgoS: at - p.First, LastPacketAgoS: last})
		}
		clock.T = time.Unix(0, 0).UTC().Add(time.Duration(at * float64(time.Second)))
		ts, err := ComputeTunnels(t, cc, clock)
		if err != nil {
			return err
		}
		if out.Tunnels == nil {
			out.Tunnels = ts.Tunnels
		}
		for _, nm := range ts.NotModelled {
			addUnique(&out.NotModelled, nm)
		}
		f := Frame{AtS: at, Label: label, Cause: cause, Event: e, States: []TunnelStateEntry{}, Changes: []TunnelChange{}, FirstPacket: first, Basis: basis}
		cur := map[string]TunnelStateEntry{}
		for _, tn := range ts.Tunnels {
			ent := TunnelStateEntry{ID: tn.ID, State: tn.State, ReasonCode: tn.ReasonCode}
			f.States = append(f.States, ent)
			cur[tn.ID] = ent
			if p, ok := prev[tn.ID]; ok && p.State != tn.State {
				f.Changes = append(f.Changes, TunnelChange{ID: tn.ID, From: p.State, To: tn.State, ReasonCode: tn.ReasonCode, Reason: tn.Reason})
			}
		}
		f.Failovers = fo.states()
		prev = cur
		out.Frames = append(out.Frames, f)
		return nil
	}

	if err := build(0, "Start: no traffic yet", "derived", nil, nil, nil); err != nil {
		return nil, err
	}
	for _, m := range moments {
		if m.ev == nil {
			label := fmt.Sprintf("180 s of silence between %s and %s: the initiating Edge tears the direct tunnel down", m.pair[0], m.pair[1])
			if err := build(m.at, label, "derived", nil, nil, bases(RDynTeardown, RA52Boundary)); err != nil {
				return nil, err
			}
			continue
		}
		e := m.ev
		var first *FirstPacket
		label := ""
		var b []Basis
		switch e.Type {
		case EvTrafficStart:
			key := pairKey(e.A, e.B)
			p := traffic[key]
			if p == nil {
				p = &pairTraffic{A: e.A, B: e.B}
				traffic[key] = p
			}
			// A new cycle starts when there was no traffic or the tunnel was torn down (assumption 53).
			if !p.Exists || (!p.Active && e.AtS-p.Last >= TeardownAfter.Seconds()) {
				p.First, p.A, p.B = e.AtS, e.A, e.B
				first = firstPacketVia(t, cond, e.A, e.B, clock)
			}
			p.Exists, p.Active = true, true
			label = fmt.Sprintf("Traffic starts from %s to %s", e.A, e.B)
			b = bases(RDynFirstPacket, RA51NoBuildDelay) // rule: assume-51
		case EvTrafficStop:
			if p := traffic[pairKey(e.A, e.B)]; p != nil {
				p.Active, p.Last = false, e.AtS
			}
			label = fmt.Sprintf("Traffic between %s and %s stops: the 180 s silence clock starts", e.A, e.B)
			b = bases(RDynTeardown)
		case EvLinkDown:
			cond.LinksDown = addUniqueStr(cond.LinksDown, e.Target)
			label, b = "Link "+e.Target+" goes down (an outage)", bases(RLinkStates)
		case EvLinkUp:
			cond.LinksDown = removeStr(cond.LinksDown, e.Target)
			label = "Link " + e.Target + " comes back up"
		case EvLinkDegraded:
			cond.LinksDegraded = addUniqueStr(cond.LinksDegraded, e.Target)
			label, b = "Link "+e.Target+" degrades (a brownout)", bases(RA55DegradedState)
		case EvLinkRestored:
			cond.LinksDegraded = removeStr(cond.LinksDegraded, e.Target)
			label = "Link " + e.Target + " recovers"
		case EvHubDown:
			cond.HubsUnreachable = addUniqueStr(cond.HubsUnreachable, e.Target)
			fo.hubDown(e.Target, cond)
			label, b = "Hub "+e.Target+" becomes unreachable", bases(RHubFailover)
		case EvHubUp:
			cond.HubsUnreachable = removeStr(cond.HubsUnreachable, e.Target)
			fo.hubUp(e.Target, cond)
			label, b = "Hub "+e.Target+" is reachable again", bases(RHubFailover, RNMHubReturnLocal)
		case EvGatewayDown:
			cond.GatewaysUnreachable = addUniqueStr(cond.GatewaysUnreachable, e.Target)
			label = "Gateway " + e.Target + " becomes unreachable"
		case EvGatewayUp:
			cond.GatewaysUnreachable = removeStr(cond.GatewaysUnreachable, e.Target)
			label = "Gateway " + e.Target + " is reachable again"
		case EvFlushFlows:
			fo.flush(e.Target, cond)
			label, b = "Affected flows are flushed", bases(RHubFailover)
		}
		ee := *e
		if err := build(e.AtS, label, "event", &ee, first, b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// firstPacketVia is the path the first packets take: the standing path, because no direct tunnel exists yet.
func firstPacketVia(t *Topology, cond Conditions, a, b string, clock Clock) *FirstPacket {
	cc := cond
	cc.Traffic = nil
	d, err := SelectPath(t, cc, Flow{Source: a, Destination: FlowDest{Kind: "edge", ID: b}}, clock)
	if err != nil || d.Chosen == nil {
		return &FirstPacket{Label: "no standing path", Reason: "no standing path is usable, so the first packets cannot be forwarded by the overlay", Basis: basisOf(RDynFirstPacket), Tunnels: []string{}}
	}
	return &FirstPacket{Label: d.Chosen.Label, Kind: d.Chosen.Kind, Tunnels: d.Chosen.TunnelIDs, Reason: "the first packets use the standing path (via the Gateway or Hub) while the initiating Edge decides about a direct tunnel", Basis: basisOf(RDynFirstPacket)}
}

func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func samePair(a, b, c, d string) bool { return pairKey(a, b) == pairKey(c, d) }

func addUniqueStr(list []string, s string) []string {
	if hasString(list, s) {
		return list
	}
	return append(list, s)
}

func removeStr(list []string, s string) []string {
	out := list[:0:0]
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

func addUnique(list *[]NotModelled, nm NotModelled) {
	for _, x := range *list {
		if x.Rule == nm.Rule {
			return
		}
	}
	*list = append(*list, nm)
}

// failoverTracker follows the documented Hub failover rows only: local flows move to the secondary Hub when the
// primary goes down; peer flows are not moved back when it returns, until flushed.
type failoverTracker struct {
	c      *overlayCtx
	spokes []string
	hubs   map[string][]string // spoke -> resolved Hub Edge ids in Hub order
	local  map[string]string
	peer   map[string]string
	lnote  map[string]string
	pnote  map[string]string
}

func newFailoverTracker(c *overlayCtx) *failoverTracker {
	f := &failoverTracker{c: c, hubs: map[string][]string{}, local: map[string]string{}, peer: map[string]string{}, lnote: map[string]string{}, pnote: map[string]string{}}
	for _, e := range c.edges {
		var list []string
		if e.CVPN.Enabled && e.CVPN.B2H.Enabled {
			for _, entry := range e.CVPN.B2H.Hubs {
				if target, _, _ := c.resolveHub(entry, e); target != nil {
					list = append(list, target.Edge.ID)
				}
			}
		}
		if len(list) >= 2 {
			f.spokes = append(f.spokes, e.Edge.ID)
			f.hubs[e.Edge.ID] = list
			f.local[e.Edge.ID], f.peer[e.Edge.ID] = list[0], list[0]
		}
	}
	return f
}

func (f *failoverTracker) firstReachable(spoke string, cond Conditions) string {
	for _, h := range f.hubs[spoke] {
		if !hasString(cond.HubsUnreachable, h) {
			return h
		}
	}
	return f.hubs[spoke][0]
}

func (f *failoverTracker) hubDown(hub string, cond Conditions) {
	for _, s := range f.spokes {
		if f.local[s] == hub { // rule: hub-failover
			f.local[s] = f.firstReachable(s, cond)
			f.lnote[s] = "moved to the secondary Hub because the primary is unreachable"
		}
		if f.peer[s] == hub {
			f.peer[s] = f.firstReachable(s, cond)
			f.pnote[s] = "peers' flows use the secondary Hub while the primary is unreachable"
		}
	}
}

func (f *failoverTracker) hubUp(hub string, cond Conditions) {
	for _, s := range f.spokes {
		if f.hubs[s][0] == hub && f.local[s] != hub { // rule: nm-hub-return-local
			f.lnote[s] = "the primary Hub is back; whether local flows move back is not documented"
		}
		if f.hubs[s][0] == hub && f.peer[s] != hub {
			f.pnote[s] = "peer flows are not moved back when the primary returns; flush the affected flows" // rule: hub-failover
		}
	}
}

func (f *failoverTracker) flush(target string, cond Conditions) {
	for _, s := range f.spokes {
		if target == "" || target == s {
			if f.peer[s] != f.firstReachable(s, cond) {
				f.peer[s] = f.firstReachable(s, cond)
				f.pnote[s] = "flows flushed: peer flows follow the best reachable Hub again"
			}
		}
	}
}

func (f *failoverTracker) states() []FailoverState {
	out := []FailoverState{}
	for _, s := range f.spokes {
		fs := FailoverState{Spoke: s, Hubs: f.hubs[s], LocalHub: f.local[s], LocalNote: f.lnote[s], PeerHub: f.peer[s], PeerNote: f.pnote[s], Basis: basisOf(RHubFailover)}
		if f.lnote[s] != "" && f.local[s] != f.hubs[s][0] && f.lnote[s] != "moved to the secondary Hub because the primary is unreachable" {
			fs.NotModelled = append(fs.NotModelled, notModelled(RNMHubReturnLocal, "Moving local flows back to the primary Hub"))
		}
		out = append(out, fs)
	}
	return out
}
