package sim

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type testLabCtx struct {
	topos   map[string]*Topology
	lessons map[string]bool
}

func (c testLabCtx) Topology(id string) *Topology { return c.topos[id] }
func (c testLabCtx) HasLesson(id string) bool     { return c.lessons[id] }

func labCtx(t *testing.T) testLabCtx {
	c := testLabCtx{topos: map[string]*Topology{}, lessons: map[string]bool{}}
	for _, tp := range loadContentTopologies(t) {
		c.topos[tp.Meta.ID] = tp
	}
	for _, id := range []string{"ov-dynamic", "ov-cloud-vpn", "ov-hubs", "ov-diagnosis", "rt-diagnosis", "rt-redistribution", "rt-bgp", "rt-ofc", "cfg-business-policy", "cfg-overrides", "dmpo-steering", "arch-mtu-overhead", "cfg-profiles", "cfg-model", "cfg-diagnosis"} {
		c.lessons[id] = true
	}
	return c
}

func loadLab(t *testing.T, name string) *Lab {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../../content/labs", name+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var l Lab
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&l); err != nil {
		t.Fatal(err)
	}
	return &l
}

func shippedLabs(t *testing.T) []*Lab {
	files, _ := filepath.Glob("../../../content/labs/*.yaml")
	if len(files) < 6 {
		t.Fatalf("want at least 6 labs, found %d", len(files))
	}
	var out []*Lab
	for _, f := range files {
		out = append(out, loadLab(t, strings.TrimSuffix(filepath.Base(f), ".yaml")))
	}
	return out
}

func TestShippedLabsValidate(t *testing.T) {
	ctx := labCtx(t)
	for _, l := range shippedLabs(t) {
		if errs := ValidateLab(l, ctx); len(errs) != 0 {
			t.Errorf("%s: %v", l.ID, errs)
		}
	}
}

// Every lab can be solved: the engine's own answers pass every choice task, and some allowed fix passes every fix task.
func TestShippedLabsCanBeSolvedAndWrongAnswersFail(t *testing.T) {
	ctx := labCtx(t)
	for _, l := range shippedLabs(t) {
		base := ctx.Topology(l.Topology)
		var fixed []string
		for _, tk := range l.Tasks {
			if tk.Type == TaskFix {
				for _, f := range l.Fixes {
					if r, err := l.Check(base, tk.ID, "", []string{f.ID}); err == nil && r.Correct {
						fixed = append(fixed, f.ID)
						break
					}
				}
				if r, err := l.Check(base, tk.ID, "", nil); err != nil || r.Correct {
					t.Errorf("%s/%s: the check must fail before any fix", l.ID, tk.ID)
				}
				continue
			}
			right, wrong := 0, 0
			for _, c := range tk.Choices {
				r, err := l.Check(base, tk.ID, c.ID, fixed)
				if err != nil {
					t.Fatal(err)
				}
				if r.Correct {
					right++
				} else {
					wrong++
				}
				if r.Expected == "" {
					t.Errorf("%s/%s: the result must name the correct answer", l.ID, tk.ID)
				}
			}
			if right != 1 || wrong != len(tk.Choices)-1 {
				t.Errorf("%s/%s: %d right, %d wrong", l.ID, tk.ID, right, wrong)
			}
		}
		if hasFixTask(l) && len(fixed) == 0 {
			t.Errorf("%s: no fix solves it", l.ID)
		}
	}
}

func hasFixTask(l *Lab) bool {
	for _, tk := range l.Tasks {
		if tk.Type == TaskFix {
			return true
		}
	}
	return false
}

func TestLabSixLabsCoverTheBrief(t *testing.T) {
	want := map[string]bool{"no-direct-tunnel": false, "hub-down": false, "branch-b-cannot-reach": false, "which-route-wins": false, "video-wrong-link": false, "mtu-ipsec": false}
	for _, l := range shippedLabs(t) {
		delete(want, l.ID)
		if len(l.Goals) == 0 || len(l.Screens) == 0 {
			t.Errorf("%s names no lessons or screens", l.ID)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing labs: %v", want)
	}
}

func TestLabFaultNeverMutatesTheAuthoredTopology(t *testing.T) {
	ctx := labCtx(t)
	for _, l := range shippedLabs(t) {
		base := ctx.Topology(l.Topology)
		before := jsonOf(t, base)
		if _, err := l.State(base, []string{}); err != nil {
			t.Fatal(err)
		}
		for _, f := range l.Fixes {
			if _, err := l.State(base, []string{f.ID}); err != nil {
				t.Fatal(err)
			}
		}
		if jsonOf(t, base) != before {
			t.Errorf("%s changed the authored topology", l.ID)
		}
	}
}

func TestLabValidationRules(t *testing.T) {
	ctx := labCtx(t)
	cases := []struct {
		name   string
		mutate func(*Lab)
		want   string
	}{
		{"a fix that cannot pass its check", func(l *Lab) {
			l.Fixes = []LabFix{{ID: "noop", Title: "t", Description: "d", Patches: []Patch{{Op: "set", Path: "edges[mesh-2-edge]/name", Value: "Renamed"}}}}
		}, "no allowed fix makes the check pass"},
		{"a diagnose task with two correct causes", func(l *Lab) {
			tk := &l.Tasks[1]
			tk.Choices[1].Value = tk.Choices[0].Value
		}, "choices are correct"},
		{"a diagnose task with no correct cause", func(l *Lab) { l.Tasks[1].Choices[0].Value = "nothing" }, "no choice is correct"},
		{"an unknown lesson", func(l *Lab) { l.Goals = append(l.Goals, "no-such-lesson") }, `unknown lesson "no-such-lesson"`},
		{"an unknown topology", func(l *Lab) { l.Topology = "nope" }, `unknown topology "nope"`},
		{"a fault patch that breaks the topology", func(l *Lab) {
			l.Fault.Patches = append(l.Fault.Patches, Patch{Op: "set", Path: "edges[mesh-1-edge]/profile", Value: "ghost"})
		}, "patched topology is invalid"},
		{"a fault patch that does not apply", func(l *Lab) {
			l.Fault.Patches = append(l.Fault.Patches, Patch{Op: "set", Path: "edges[ghost]/name", Value: "x"})
		}, "no Edge"},
		{"a check that already passes", func(l *Lab) { l.Tasks[2].Equals = "via-gateway" }, "already passes"},
		{"an unregistered assumption", func(l *Lab) { l.RelyOn = append(l.RelyOn, 9999) }, "not a registered assumption"},
		{"an answer that relies on an unlisted assumption", func(l *Lab) { l.RelyOn = []int{54} }, "list it under relies_on_assumptions"},
		{"a symptom that hides the assumptions", func(l *Lab) { l.Symptom = "Two branches cannot reach each other." }, `must say so`},
		{"a duplicate task id", func(l *Lab) { l.Tasks[1].ID = l.Tasks[0].ID }, "duplicate task id"},
		{"a hint for an unknown task", func(l *Lab) { l.Hints = append(l.Hints, LabHint{Task: "ghost", Text: "x"}) }, `unknown task "ghost"`},
		{"a bad difficulty", func(l *Lab) { l.Difficulty = 7 }, "must be 1 to 3"},
		{"a not-modelled answer", func(l *Lab) { l.Fault.Conditions.OrchestratorUnreachable = true }, "not-modelled"},
		{"an unknown probe kind", func(l *Lab) { l.Tasks[0].Probe.Kind = "telepathy" }, "unknown probe kind"},
		{"fixes without a fix task", func(l *Lab) { l.Tasks = l.Tasks[:2] }, "no task is of type fix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := loadLab(t, "no-direct-tunnel")
			c.mutate(l)
			errs := ValidateLab(l, ctx)
			for _, e := range errs {
				if strings.Contains(e, c.want) {
					return
				}
			}
			t.Errorf("want an error containing %q, got %v", c.want, errs)
		})
	}
}

func TestLabErrorsAreCollectedTogether(t *testing.T) {
	l := loadLab(t, "no-direct-tunnel")
	l.Fixes = []LabFix{{ID: "noop", Title: "t", Description: "d", Patches: []Patch{{Op: "set", Path: "edges[mesh-2-edge]/name", Value: "Renamed"}}}} // a fix that cannot pass
	l.Tasks[1].Choices[1].Value = l.Tasks[1].Choices[0].Value                                                                                        // two correct causes
	l.Goals = append(l.Goals, "no-such-lesson")                                                                                                      // an unknown lesson
	errs := ValidateLab(l, labCtx(t))
	for _, want := range []string{"no allowed fix makes the check pass", "choices are correct", `unknown lesson "no-such-lesson"`} {
		ok := false
		for _, e := range errs {
			ok = ok || strings.Contains(e, want)
		}
		if !ok {
			t.Errorf("missing %q in %v", want, errs)
		}
	}
}
