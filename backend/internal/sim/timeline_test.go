package sim

import "testing"

func frameAt(t *testing.T, r *TimelineResult, i int) Frame {
	t.Helper()
	if i >= len(r.Frames) {
		t.Fatalf("only %d frames", len(r.Frames))
	}
	return r.Frames[i]
}

func stateOf(f Frame, id string) string {
	for _, s := range f.States {
		if s.ID == id {
			return s.State
		}
	}
	return ""
}

func TestTimelineDynamicTunnelLifecycle(t *testing.T) {
	tp := demo(t)
	r, err := Timeline(tp, Conditions{}, []Event{
		{AtS: 10, Type: EvTrafficStart, A: "mesh-1-edge", B: "mesh-2-edge"},
		{AtS: 40, Type: EvTrafficStop, A: "mesh-1-edge", B: "mesh-2-edge"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := "dynamic:mesh-1-edge:mesh-2-edge"
	want := []struct {
		at    float64
		state string
	}{{0, StNotEstablished}, {10, StUp}, {40, StUp}, {220, StTornDown}}
	if len(r.Frames) != len(want) {
		t.Fatalf("frames: %d, want %d", len(r.Frames), len(want))
	}
	for i, w := range want {
		f := frameAt(t, r, i)
		if f.AtS != w.at || stateOf(f, id) != w.state {
			t.Errorf("frame %d at %.0f s: %s, want %.0f s %s", i, f.AtS, stateOf(f, id), w.at, w.state)
		}
	}
	// The first packet goes via the standing path (the Gateway) before the direct tunnel is used.
	if fp := frameAt(t, r, 1).FirstPacket; fp == nil || fp.Kind != OptViaGateway {
		t.Errorf("first packet: %+v", fp)
	}
	// Changes name the transition.
	ch := frameAt(t, r, 1).Changes
	found := false
	for _, c := range ch {
		found = found || (c.ID == id && c.From == StNotEstablished && c.To == StUp)
	}
	if !found {
		t.Errorf("changes: %+v", ch)
	}
	if frameAt(t, r, 3).Cause != "derived" || frameAt(t, r, 3).Basis[0].Rule != RDynTeardown {
		t.Errorf("teardown frame: %+v", frameAt(t, r, 3))
	}
}

func TestTimelineNewTrafficBeforeTeardownKeepsTheTunnel(t *testing.T) {
	r, err := Timeline(demo(t), Conditions{}, []Event{
		{AtS: 10, Type: EvTrafficStart, A: "mesh-1-edge", B: "mesh-2-edge"},
		{AtS: 20, Type: EvTrafficStop, A: "mesh-1-edge", B: "mesh-2-edge"},
		{AtS: 150, Type: EvTrafficStart, A: "mesh-2-edge", B: "mesh-1-edge"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := "dynamic:mesh-1-edge:mesh-2-edge"
	for _, f := range r.Frames {
		if f.AtS >= 10 && stateOf(f, id) != StUp {
			t.Errorf("at %.0f s the tunnel should still be up: %s", f.AtS, stateOf(f, id))
		}
	}
}

func TestTimelineRebuildAfterTeardown(t *testing.T) {
	r, err := Timeline(demo(t), Conditions{}, []Event{
		{AtS: 0, Type: EvTrafficStart, A: "mesh-1-edge", B: "mesh-2-edge"},
		{AtS: 5, Type: EvTrafficStop, A: "mesh-1-edge", B: "mesh-2-edge"},
		{AtS: 300, Type: EvTrafficStart, A: "mesh-2-edge", B: "mesh-1-edge"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := "dynamic:mesh-1-edge:mesh-2-edge"
	last := r.Frames[len(r.Frames)-1]
	if last.AtS != 300 || stateOf(last, id) != StUp || last.FirstPacket == nil {
		t.Errorf("a new cycle starts: first packets via the standing path, then a rebuild (assumption 53): %+v", last)
	}
	torn := false
	for _, f := range r.Frames {
		torn = torn || stateOf(f, id) == StTornDown
	}
	if !torn {
		t.Error("the tunnel should have been torn down in between")
	}
}

func TestTimelineHubFailoverRows(t *testing.T) {
	r, err := Timeline(demo(t), Conditions{}, []Event{
		{AtS: 5, Type: EvHubDown, Target: "hub-p-edge"},
		{AtS: 20, Type: EvHubUp, Target: "hub-p-edge"},
		{AtS: 30, Type: EvFlushFlows},
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func(f Frame, spoke string) FailoverState {
		for _, fs := range f.Failovers {
			if fs.Spoke == spoke {
				return fs
			}
		}
		t.Fatalf("no failover state for %s", spoke)
		return FailoverState{}
	}
	start, down, up, flushed := get(frameAt(t, r, 0), "spoke-1-edge"), get(frameAt(t, r, 1), "spoke-1-edge"), get(frameAt(t, r, 2), "spoke-1-edge"), get(frameAt(t, r, 3), "spoke-1-edge")
	if start.LocalHub != "hub-p-edge" || start.PeerHub != "hub-p-edge" {
		t.Errorf("start: %+v", start)
	}
	if down.LocalHub != "hub-s-edge" || down.PeerHub != "hub-s-edge" {
		t.Errorf("primary down: local flows move to the secondary Hub: %+v", down)
	}
	if up.PeerHub != "hub-s-edge" || up.PeerNote == "" {
		t.Errorf("peer flows are NOT moved back when the primary returns: %+v", up)
	}
	if up.LocalHub != "hub-s-edge" || len(up.NotModelled) == 0 || up.NotModelled[0].Rule != RNMHubReturnLocal {
		t.Errorf("moving local flows back is not documented, so it must be a not-modelled marker, not a guess: %+v", up)
	}
	if flushed.PeerHub != "hub-p-edge" {
		t.Errorf("after the flush peer flows follow the primary again: %+v", flushed)
	}
	// The tunnel to the primary Hub goes down and comes back; the secondary is never affected.
	id := "hub:spoke-1-edge:hub-p-edge"
	if stateOf(frameAt(t, r, 1), id) != StDown || stateOf(frameAt(t, r, 2), id) != StUp {
		t.Errorf("tunnel states: %s then %s", stateOf(frameAt(t, r, 1), id), stateOf(frameAt(t, r, 2), id))
	}
}

func TestTimelineLinkAndGatewayEvents(t *testing.T) {
	r, err := Timeline(demo(t), Conditions{}, []Event{
		{AtS: 1, Type: EvLinkDegraded, Target: "s1-inet"},
		{AtS: 2, Type: EvLinkDown, Target: "s1-inet"},
		{AtS: 3, Type: EvLinkUp, Target: "s1-inet"},
		{AtS: 4, Type: EvGatewayDown, Target: "gw-1"},
		{AtS: 5, Type: EvGatewayUp, Target: "gw-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := "gateway:spoke-3-edge:gw-1"
	if stateOf(frameAt(t, r, 4), id) != StDown || stateOf(frameAt(t, r, 5), id) != StUp {
		t.Errorf("gateway events: %s %s", stateOf(frameAt(t, r, 4), id), stateOf(frameAt(t, r, 5), id))
	}
	hub := "hub:spoke-1-edge:hub-p-edge"
	_ = hub
	if len(r.Frames) != 6 {
		t.Errorf("frames: %d", len(r.Frames))
	}
}

func TestTimelineRejectsBadEvents(t *testing.T) {
	bad := [][]Event{
		{{AtS: -1, Type: EvLinkDown, Target: "s1-inet"}},
		{{AtS: 1, Type: "explode"}},
		{{AtS: 1, Type: EvLinkDown, Target: "nope"}},
		{{AtS: 1, Type: EvHubDown, Target: "nope"}},
		{{AtS: 1, Type: EvGatewayDown, Target: "nope"}},
		{{AtS: 1, Type: EvTrafficStart, A: "mesh-1-edge", B: "mesh-1-edge"}},
		{{AtS: 1, Type: EvFlushFlows, Target: "nope"}},
	}
	for _, ev := range bad {
		_, err := Timeline(demo(t), Conditions{}, ev)
		if _, ok := err.(*TimelineError); !ok {
			t.Errorf("%+v: want TimelineError, got %v", ev, err)
		}
	}
}

func TestTimelineIsDeterministic(t *testing.T) {
	ev := []Event{{AtS: 10, Type: EvTrafficStart, A: "mesh-1-edge", B: "mesh-2-edge"}, {AtS: 40, Type: EvTrafficStop, A: "mesh-1-edge", B: "mesh-2-edge"}}
	a, _ := Timeline(demo(t), Conditions{}, ev)
	b, _ := Timeline(demo(t), Conditions{}, ev)
	if len(a.Frames) != len(b.Frames) {
		t.Fatal("frame counts differ")
	}
	for i := range a.Frames {
		if a.Frames[i].Label != b.Frames[i].Label || len(a.Frames[i].States) != len(b.Frames[i].States) {
			t.Fatalf("frame %d differs", i)
		}
	}
}
