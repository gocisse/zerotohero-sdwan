package sim

import (
	"strings"
	"testing"
)

func noDefault(x *Topology) { edgeOf(x, "hub-edge").Routing.StaticRoutes = nil }

func reach(t *testing.T, tp *Topology, c Conditions, as RoutingAssumptions, src, dst string) *ReachResult {
	t.Helper()
	r, err := Reachability(tp, c, as, ReachQuery{Source: src, Destination: dst}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func stageOf(r *ReachResult, id string) Stage {
	for _, s := range r.Stages {
		if s.ID == id {
			return s
		}
	}
	return Stage{}
}

// Each way a prefix can fail to arrive gives its own first failing stage.
func TestFirstFailingStage(t *testing.T) {
	as := DefaultRoutingAssumptions()
	cases := []struct {
		name    string
		mutate  func(*Topology)
		cond    Conditions
		as      func(*RoutingAssumptions)
		src     string
		dst     string
		outcome string
		stage   string
		reason  string
		lesson  string
	}{
		{"reachable: A to C's static route", nil, Conditions{}, nil, "a-edge", "172.16.50.9", OutcomeReachable, "", "reaches", "rt-diagnosis"},
		{"nobody originates the prefix", noDefault, Conditions{}, nil, "c-edge", "10.250.0.0/24", OutcomeUnreachable, StageKnown, "nothing originates", "rt-diagnosis"},
		{"the neighbor is down", nil, Conditions{NeighborsDown: []string{"a-edge/ge3"}}, nil, "c-edge", "10.20.0.0/24", OutcomeUnreachable, StageKnown, "not Full", "rt-diagnosis"},
		{"external OSPF is held out", nil, Conditions{}, nil, "c-edge", "10.70.0.0/24", OutcomeUnreachable, StageAdvertised, "External OSPF flag", "rt-redistribution"},
		{"a static route without Advertise", func(x *Topology) { edgeOf(x, "c-edge").Routing.StaticRoutes[0].Advertise = false }, Conditions{}, nil, "a-edge", "172.16.50.0/24", OutcomeUnreachable, StageAdvertised, "Advertise box", "rt-redistribution"},
		{"denied by the inbound filter", nil, Conditions{}, nil, "a-edge", "10.99.0.0/24", OutcomeUnreachable, StageFiltered, "denies", "rt-filters"},
		{"not redistributed into the source LAN (assumption 22 default)", nil, Conditions{}, nil, "b-edge", "10.20.0.0/24", OutcomeUnreachable, StageRedistribute, "does not send it to its LAN", "rt-redistribution"},
		{"redistributed once the assumption is on", nil, Conditions{}, func(a *RoutingAssumptions) { a.BGPOverlayPrefix = true }, "b-edge", "10.20.0.0/24", OutcomeReachable, "", "reaches", "rt-diagnosis"},
		{"no route: Cloud VPN off at the source", func(x *Topology) { profileOf(x, "branch-static").CloudVPN = nil }, Conditions{}, nil, "c-edge", "10.20.0.0/24", OutcomeUnreachable, StageWinner, "Cloud VPN is not enabled", "rt-ofc"},
		{"path not built: the owner's link is down", noDefault, Conditions{LinksDown: []string{"a-inet"}}, nil, "c-edge", "10.20.0.0/24", OutcomeUnreachable, StagePath, "every candidate is unreachable", "rt-diagnosis"},
		{"the second exit keeps it reachable", nil, Conditions{LinksDown: []string{"a-inet"}}, nil, "c-edge", "10.40.0.0/24", OutcomeReachable, "", "reaches", "rt-diagnosis"},
		{"the static probe is down", noDefault, Conditions{ProbesDown: []string{"c-edge/printers"}}, nil, "a-edge", "172.16.50.0/24", OutcomeUnreachable, StagePath, "unreachable", "rt-diagnosis"},
		{"a local route needs no overlay", nil, Conditions{}, nil, "c-edge", "10.60.1.5", OutcomeReachable, "", "reaches", "rt-diagnosis"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp := rdemo(t)
			if c.mutate != nil {
				c.mutate(tp)
			}
			a := as
			if c.as != nil {
				c.as(&a)
			}
			r := reach(t, tp, c.cond, a, c.src, c.dst)
			if r.Outcome != c.outcome || r.FirstFailing != c.stage {
				t.Fatalf("outcome=%s failing=%q; want %s %q\n%s", r.Outcome, r.FirstFailing, c.outcome, c.stage, r.Reason)
			}
			if !strings.Contains(r.Reason, c.reason) && !strings.Contains(strings.Join(stageOf(r, c.stage).Evidence, " "), c.reason) {
				t.Errorf("reason %q lacks %q", r.Reason, c.reason)
			}
			if r.LessonRef != c.lesson {
				t.Errorf("lesson = %s, want %s", r.LessonRef, c.lesson)
			}
			for _, s := range r.Stages {
				if s.Basis.Status == "" || s.LessonRef == "" || s.Where == "" {
					t.Errorf("stage %s lacks basis, lesson or where", s.ID)
				}
			}
		})
	}
}

func TestStagesAfterTheFirstFailureAreNotReached(t *testing.T) {
	r := reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "c-edge", "10.70.0.0/24")
	seen := false
	for _, s := range r.Stages {
		if seen && s.Status != StageNotReached && s.Status != StageSkipped {
			t.Errorf("stage %s is %s after the first failure", s.ID, s.Status)
		}
		if s.ID == r.FirstFailing {
			seen = true
		}
	}
}

func TestReachabilityShowsAssumptionsAndSensitivity(t *testing.T) {
	r := reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "b-edge", "10.20.0.0/24")
	if len(r.Assumptions) != 10 {
		t.Fatalf("every selectable assumption must be listed, got %d", len(r.Assumptions))
	}
	used := false
	for _, a := range r.Assumptions {
		used = used || (a.Key == "bgp_overlay_prefix" && a.Used && a.Assumption == 22)
	}
	if !used {
		t.Error("the answer relies on assumption 22 and must say so")
	}
	if len(r.DependsOn) != 1 || r.DependsOn[0].Switch.Key != "bgp_overlay_prefix" || r.DependsOn[0].NewOutcome != OutcomeReachable {
		t.Errorf("depends_on = %+v", r.DependsOn)
	}
	// The static route's answer depends on the Global Advertise Flag assumption (assumption 69).
	r = reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "a-edge", "172.16.50.0/24")
	if len(r.DependsOn) != 1 || r.DependsOn[0].Switch.Key != "advertise_static" {
		t.Errorf("depends_on = %+v", r.DependsOn)
	}
	// An answer that no switch changes says so.
	r = reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "c-edge", "10.60.1.5")
	if len(r.DependsOn) != 0 {
		t.Errorf("depends_on = %+v", r.DependsOn)
	}
}

func TestMultipleExitsAreReportedPerOrigin(t *testing.T) {
	r := reach(t, rdemo(t), Conditions{LinksDown: []string{"a-inet"}}, DefaultRoutingAssumptions(), "c-edge", "10.40.0.0/24")
	if len(r.Origins) != 2 {
		t.Fatalf("origins = %+v", r.Origins)
	}
	for _, o := range r.Origins {
		switch o.Edge {
		case "a-edge":
			if o.Status != StageFailed || o.Stage != StagePath {
				t.Errorf("A: %+v", o)
			}
		case "b-edge":
			if !o.Selected {
				t.Errorf("B: %+v", o)
			}
		}
	}
	if r.Selected == nil || r.Selected.Origin != "b-edge" {
		t.Errorf("selected = %+v", r.Selected)
	}
}

func TestCrossSegmentIsNotModelled(t *testing.T) {
	tp := rdemo(t)
	r, err := Reachability(tp, Conditions{}, DefaultRoutingAssumptions(), ReachQuery{Source: "c-edge", Destination: "10.20.0.0/24", DestinationSegment: "guest"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Outcome != OutcomeNotModelled || !hasRule(r.NotModelled, RNMRtCrossSegment) {
		t.Errorf("%+v", r)
	}
	// A prefix that exists only in another segment.
	tp.Segments = append(tp.Segments, Segment{ID: "guest", Name: "Guest", Type: SegRegular})
	for i := range tp.Profiles {
		tp.Profiles[i].Segments = []string{"global", "guest"}
		if tp.Profiles[i].CloudVPN != nil {
			tp.Profiles[i].CloudVPN.Segments = []string{"global", "guest"}
		}
	}
	edgeOf(tp, "c-edge").Routing.StaticRoutes[0].Segment = "guest"
	noDefault(tp)
	r = reach(t, tp, Conditions{}, DefaultRoutingAssumptions(), "a-edge", "172.16.50.0/24")
	if r.Outcome != OutcomeNotModelled {
		t.Errorf("want not modelled, got %s: %s", r.Outcome, r.Reason)
	}
	r2, err := Reachability(tp, Conditions{}, DefaultRoutingAssumptions(), ReachQuery{Source: "a-edge", Destination: "172.16.50.0/24", Segment: "guest"}, nil)
	if err != nil || r2.Outcome != OutcomeReachable {
		t.Errorf("same-segment query should work: %v %+v", err, r2)
	}
}

func TestReachabilityQueryErrors(t *testing.T) {
	tp := rdemo(t)
	for _, q := range []ReachQuery{{Source: "ghost", Destination: "10.0.0.0/24"}, {Source: "a-edge", Destination: "nonsense"}, {Source: "a-edge", Destination: "2001:db8::/32"}, {Source: "a-edge", Destination: "10.0.0.0/24", Segment: "nope"}} {
		if _, err := Reachability(tp, Conditions{}, DefaultRoutingAssumptions(), q, nil); err == nil {
			t.Errorf("%+v should be rejected", q)
		} else if _, ok := err.(*QueryError); !ok {
			t.Errorf("%+v: want QueryError, got %T", q, err)
		}
	}
}

// Section 6 lesson 8 names six checks in this order; the stage list must follow it.
func TestStagesFollowTheLessonDiagnosticOrder(t *testing.T) {
	r := reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "a-edge", "172.16.50.0/24")
	want := []string{StageKnown, StageAdvertised, StageFiltered, StageRedistribute, StageWinner, StagePath}
	for i, s := range r.Stages {
		if s.ID != want[i] || s.Number != i+1 {
			t.Errorf("stage %d = %s", i, s.ID)
		}
	}
}

func TestDefaultRouteCoversAndShadowedRoutesAreExplained(t *testing.T) {
	r := reach(t, rdemo(t), Conditions{}, DefaultRoutingAssumptions(), "c-edge", "10.250.0.9")
	if r.Outcome != OutcomeReachable || !strings.Contains(r.Reason, "less specific prefix 0.0.0.0/0") {
		t.Errorf("%s", r.Reason)
	}
	r = reach(t, rdemo(t), Conditions{LinksDown: []string{"a-inet"}}, DefaultRoutingAssumptions(), "c-edge", "10.20.0.0/24")
	if r.Outcome != OutcomeReachable || !strings.Contains(strings.Join(stageOf(r, StageWinner).Evidence, " "), "more specific route 10.20.0.0/24") {
		t.Errorf("%+v", stageOf(r, StageWinner))
	}
}
