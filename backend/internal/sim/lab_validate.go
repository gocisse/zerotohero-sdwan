package sim

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// LabContext gives the validator what a lab refers to.
type LabContext interface {
	Topology(id string) *Topology
	HasLesson(id string) bool
}

var labIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var probeKinds = map[string]bool{PrTraceVerdict: true, PrTraceBlocked: true, PrTracePathKind: true, PrTracePathVia: true, PrTraceLink: true, PrTraceRule: true,
	PrTraceRuleOwner: true, PrTraceMTUMin: true, PrTraceMTULimit: true, PrTraceMTUFit: true, PrRejectedCode: true, PrReachFailing: true, PrReachOutcome: true,
	PrRouteField: true, PrTunnelState: true, PrTunnelReason: true, PrTimelinePeer: true, PrTimelineLocal: true, PrEffectiveValue: true, PrEffectiveSource: true}

// ValidateLab checks one lab and returns EVERY problem as "path: message". It actually applies the fault and every
// allowed fix and evaluates every check, so a lab that cannot be solved, has two correct root causes, or depends on a
// not-modelled result is an error.
func ValidateLab(l *Lab, ctx LabContext) []string {
	var errs []string
	add := func(path, format string, args ...any) { errs = append(errs, path+": "+fmt.Sprintf(format, args...)) }
	if !labIDRe.MatchString(l.ID) {
		add("id", "must be lowercase letters, digits and single hyphens, got %q", l.ID)
	}
	if strings.TrimSpace(l.Title) == "" {
		add("title", "must not be empty")
	}
	if strings.TrimSpace(l.Summary) == "" {
		add("summary", "must not be empty")
	}
	if l.Difficulty < 1 || l.Difficulty > 3 {
		add("difficulty", "must be 1 to 3, got %d", l.Difficulty)
	}
	if l.EstimatedMinutes <= 0 {
		add("estimated_minutes", "must be greater than 0")
	}
	if strings.TrimSpace(l.Symptom) == "" {
		add("symptom", "must not be empty")
	}
	if len(l.Goals) == 0 {
		add("goals", "name at least one lesson the lab reinforces")
	}
	for _, g := range l.Goals {
		if !ctx.HasLesson(g) {
			add("goals", "unknown lesson %q", g)
		}
	}
	if len(l.Screens) == 0 {
		add("screens", "name at least one mock VCO screen to inspect")
	}
	for i, sc := range l.Screens {
		if strings.TrimSpace(sc.Label) == "" || strings.TrimSpace(sc.Path) == "" || strings.HasPrefix(sc.Path, "/") {
			add(fmt.Sprintf("screens[%d]", i), "needs a label and a path relative to /sim/<topology>/ (no leading slash)")
		}
	}
	base := ctx.Topology(l.Topology)
	if base == nil {
		add("topology", "unknown topology %q", l.Topology)
		return errs
	}
	// The fault.
	st, err := l.State(base, nil)
	if err != nil {
		add("fault", "%v", err)
		return errs
	}
	for _, is := range Errors(Validate(st.Topo)) {
		add("fault.patches", "the patched topology is invalid: %s: %s", is.Path, is.Message)
	}
	if _, err := ComputeTunnels(st.Topo, st.Cond, nil); err != nil {
		add("fault.conditions", "%v", err)
	}
	if bad := st.Topo.validateRoutingConditions(st.Cond); len(bad) > 0 {
		add("fault.conditions", "%s", strings.Join(bad, "; "))
	}
	if l.Flow != nil {
		if _, err := TraceFlow(st.Topo, st.Cond, st.As, *l.Flow, nil); err != nil {
			add("flow", "%v", err)
		}
	}
	// Fixes.
	fixIDs := map[string]bool{}
	for i, f := range l.Fixes {
		p := fmt.Sprintf("fixes[%d]", i)
		switch {
		case !labIDRe.MatchString(f.ID):
			add(p+".id", "must be lowercase letters, digits and single hyphens, got %q", f.ID)
		case fixIDs[f.ID]:
			add(p+".id", "duplicate fix id %q", f.ID)
		}
		fixIDs[f.ID] = true
		if strings.TrimSpace(f.Title) == "" || strings.TrimSpace(f.Description) == "" {
			add(p, "a fix needs a title and a description")
		}
		if len(f.Patches) == 0 {
			add(p+".patches", "a fix needs at least one patch")
			continue
		}
		fst, err := l.State(base, []string{f.ID})
		if err != nil {
			add(p+".patches", "%v", err)
			continue
		}
		for _, is := range Errors(Validate(fst.Topo)) {
			add(p+".patches", "the patched topology is invalid: %s: %s", is.Path, is.Message)
		}
	}
	// Tasks.
	if len(l.Tasks) == 0 {
		add("tasks", "a lab needs at least one task")
	}
	taskIDs := map[string]bool{}
	hasFix := false
	relied := map[int]bool{}
	for _, n := range l.RelyOn {
		relied[n] = true
	}
	found := map[int]bool{}
	for i, tk := range l.Tasks {
		p := fmt.Sprintf("tasks[%d]", i)
		switch {
		case !labIDRe.MatchString(tk.ID):
			add(p+".id", "must be lowercase letters, digits and single hyphens, got %q", tk.ID)
		case taskIDs[tk.ID]:
			add(p+".id", "duplicate task id %q", tk.ID)
		}
		taskIDs[tk.ID] = true
		if strings.TrimSpace(tk.Prompt) == "" {
			add(p+".prompt", "must not be empty")
		}
		if strings.TrimSpace(tk.Explanation) == "" {
			add(p+".explanation", "must not be empty: it is shown after the task is checked")
		}
		if tk.Probe == nil {
			add(p+".probe", "every task reads its answer from the engine through a probe")
			continue
		}
		if !probeKinds[tk.Probe.Kind] {
			add(p+".probe.kind", "unknown probe kind %q", tk.Probe.Kind)
			continue
		}
		pr, err := st.Eval(tk.Probe)
		if err != nil {
			add(p+".probe", "%v", err)
			continue
		}
		if pr.NotModelled {
			add(p+".probe", "the answer depends on a not-modelled result; a lab may not (%s)", pr.Detail)
		}
		for _, n := range pr.Assumptions {
			found[n] = true
			if !relied[n] {
				add(p+".probe", "the answer relies on assumption %d; list it under relies_on_assumptions and say so in the symptom text", n)
			}
		}
		// A flip of a visible switch that changes the answer is a dependence on that assumption.
		for _, key := range st.As.switchKeys() {
			flipped, ok := st.As.Flip(key)
			if !ok {
				continue
			}
			other := &LabState{Topo: st.Topo, Cond: st.Cond, As: flipped, Flow: st.Flow}
			if o, err := other.Eval(tk.Probe); err == nil && o.Value != pr.Value {
				for _, sw := range st.As.Describe(nil) {
					if sw.Key == key && !relied[sw.Assumption] {
						add(p+".probe", "the answer changes with the %q switch (assumption %d); list it under relies_on_assumptions and say so in the symptom text", key, sw.Assumption)
					}
					if sw.Key == key {
						found[sw.Assumption] = true
					}
				}
			}
		}
		switch tk.Type {
		case TaskInspect, TaskDiagnose, TaskPredict:
			validateChoices(tk, pr, p, add)
		case TaskFix:
			hasFix = true
			if tk.Equals == "" {
				add(p+".equals", "a fix task states the value the probe must have afterwards")
			}
			if pr.Value == tk.Equals {
				add(p, "the check already passes in the faulty state, so there is nothing to fix")
			}
			passing := 0
			for _, f := range l.Fixes {
				fst, err := l.State(base, []string{f.ID})
				if err != nil {
					continue
				}
				v, err := fst.Eval(tk.Probe)
				if err != nil {
					add(p+".probe", "after fix %q: %v", f.ID, err)
					continue
				}
				if v.NotModelled {
					add(p+".probe", "after fix %q the answer depends on a not-modelled result", f.ID)
				}
				if v.Value == tk.Equals {
					passing++
				}
			}
			if passing == 0 {
				add(p, "no allowed fix makes the check pass (it expects %q)", tk.Equals)
			}
		default:
			add(p+".type", "must be inspect, diagnose, fix or predict, got %q", tk.Type)
		}
	}
	if len(l.Fixes) > 0 && !hasFix {
		add("fixes", "fixes are listed but no task is of type fix")
	}
	if hasFix && len(l.Fixes) == 0 {
		add("fixes", "a fix task needs allowed fixes")
	}
	// Hints.
	for i, h := range l.Hints {
		p := fmt.Sprintf("hints[%d]", i)
		if strings.TrimSpace(h.Text) == "" {
			add(p+".text", "must not be empty")
		}
		if h.Task != "" && !taskIDs[h.Task] {
			add(p+".task", "unknown task %q", h.Task)
		}
	}
	// Debrief.
	for i, d := range l.Debrief {
		p := fmt.Sprintf("debrief[%d]", i)
		if strings.TrimSpace(d.Finding) == "" || strings.TrimSpace(d.Where) == "" {
			add(p, "a debrief line needs a finding and where it is found in the VCO")
		}
		if d.Unverified && strings.TrimSpace(d.Note) == "" {
			add(p+".note", "an unverified location needs a note saying what the guides do not confirm")
		}
		if !d.Unverified && strings.TrimSpace(d.Note) != "" {
			add(p+".note", "a note is only for an unverified location")
		}
	}
	// Assumptions the lab relies on.
	nums := map[int]bool{}
	for _, r := range Rules {
		if r.Status == StatusAssumption {
			nums[r.Assumption] = true
		}
	}
	for _, n := range l.RelyOn {
		if !nums[n] {
			add("relies_on_assumptions", "assumption %d is not a registered assumption", n)
		}
	}
	if len(l.RelyOn) > 0 && !strings.Contains(strings.ToLower(l.Symptom), "assumption") {
		add("symptom", "the lab relies on assumptions %v and its symptom text must say so (use the word \"assumption\")", sortedInts(l.RelyOn))
	}
	var unused []int
	for n := range relied {
		if !found[n] && nums[n] {
			unused = append(unused, n)
		}
	}
	if len(unused) > 0 {
		add("relies_on_assumptions", "assumptions %v are listed but no answer relies on them", sortedInts(unused))
	}
	return errs
}

func sortedInts(in []int) []int {
	out := append([]int{}, in...)
	sort.Ints(out)
	return out
}

func validateChoices(tk LabTask, pr ProbeResult, p string, add func(string, string, ...any)) {
	if len(tk.Choices) < 2 {
		add(p+".choices", "needs at least two choices")
	}
	ids := map[string]bool{}
	correct := 0
	for j, c := range tk.Choices {
		cp := fmt.Sprintf("%s.choices[%d]", p, j)
		switch {
		case !labIDRe.MatchString(c.ID):
			add(cp+".id", "must be lowercase letters, digits and single hyphens, got %q", c.ID)
		case ids[c.ID]:
			add(cp+".id", "duplicate choice id %q", c.ID)
		}
		ids[c.ID] = true
		if strings.TrimSpace(c.Text) == "" || strings.TrimSpace(c.Value) == "" {
			add(cp, "a choice needs text and the engine value it stands for")
		}
		if c.Value == pr.Value {
			correct++
		}
	}
	switch {
	case correct == 0:
		add(p+".choices", "no choice is correct: the engine says %q", pr.Value)
	case correct > 1:
		add(p+".choices", "%d choices are correct (value %q): a task has exactly one correct answer", correct, pr.Value)
	}
}
