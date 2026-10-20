// Package obsreport builds docs/OBSERVATION-PRIORITIES.md: the course owner's lab checklist. It lists every VERIFY
// row of the fact sheets and every numbered assumption of docs/SIMULATOR-MODEL.md, says which observation files are
// attached, which engine rules, labs, lessons and quiz questions depend on each, and ranks the items by how much
// content they block.
package obsreport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/sim"
)

// SheetSections says which course sections a fact sheet backs. Lessons are not tagged with the rows they use, so a
// VERIFY row's lesson reach is section-level (coarse); assumptions have exact reach through labs and the engine.
// TestEverySheetIsMapped fails when a new fact sheet is missing here.
var SheetSections = map[string][]string{
	"wan-background": {"introduction"}, "sdwan-concepts": {"introduction"}, "branding-history": {"introduction"}, "vco-ui-map": {"introduction"}, "release-target": {"introduction"},
	"components": {"introduction", "architecture"}, "underlay-overlay": {"introduction", "architecture"},
	"edge-details": {"architecture"}, "orchestrator-details": {"architecture"}, "gateway-details": {"architecture"}, "tunnel-types": {"architecture"},
	"ports-protocols": {"architecture"}, "vcmp-tunnels": {"architecture"}, "mtu-overhead": {"architecture"},
	"dmpo": {"dmpo"}, "dmpo-monitoring": {"dmpo"}, "dmpo-steering": {"dmpo"}, "dmpo-remediation": {"dmpo"}, "dmpo-load-balancing": {"dmpo"}, "dmpo-vco-ui": {"dmpo"},
	"segments": {"configuration"}, "profiles": {"configuration"}, "edge-overrides": {"configuration"}, "business-policy": {"configuration"}, "firewall": {"configuration"},
	"applications-qos": {"configuration"}, "vco-config-ui": {"configuration"},
	"cloud-vpn": {"overlay"}, "hubs-and-branches": {"overlay"}, "non-sdwan-destinations": {"overlay"}, "network-services-other": {"overlay"}, "overlay-paths": {"overlay"}, "vco-overlay-ui": {"overlay"},
	"routing-overview": {"routing"}, "static-routes": {"routing"}, "ospf": {"routing"}, "bgp": {"routing"}, "route-redistribution-and-precedence": {"routing"},
	"overlay-flow-control": {"routing"}, "vco-routing-ui": {"routing"},
	"config-nsd-gateway": {"network-services-config"}, "config-nsd-edge": {"network-services-config"}, "config-api-credentials": {"network-services-config"}, "config-clusters-hubs": {"network-services-config"},
	"config-netflow": {"network-services-config"}, "config-dns": {"network-services-config"}, "config-private-network-names": {"network-services-config"}, "config-prefix-delegation": {"network-services-config"},
	"config-authentication": {"network-services-config"}, "config-vnf": {"network-services-config"},
	"config-segments": {"profile-policy-config"}, "config-profiles": {"profile-policy-config"}, "config-business-policy-rules": {"profile-policy-config"},
	"config-firewall-rules": {"profile-policy-config"}, "config-application-match": {"profile-policy-config"},
	"edge-models": {"topologies"}, "site-topologies": {"topologies"}, "bronze-silver-gold": {"topologies"}, "edge-ha": {"topologies"}, "vco-edge-models-ui": {"topologies"},
}

// SectionCandidate is an observation candidate with the section of its lesson.
type SectionCandidate struct {
	Section string
	content.ObservationCandidate
}

// candidateGroups says which sections have a candidates part in the report, in order, and its title.
var candidateGroups = []struct{ Section, Title string }{
	{"topologies", "Candidates from topology lessons"},
	{"network-services-config", "Candidates from network services config lessons"},
	{"profile-policy-config", "Candidates from profile and policy config lessons"},
}

// Item is one open question: a VERIFY row or an assumption.
type Item struct {
	Kind        string // "VERIFY row" or "assumption"
	ID          string // "bgp#34" or "assumption 22"
	Sheet       string
	Line        int
	Number      int
	Text        string
	Observation []string // observation files attached (block resolves the item)
	Rules       []string // engine rule ids
	Sites       int      // places in the engine that cite those rules
	Labs        []string
	Lessons     []string // exact (assumptions) or section-level (VERIFY rows)
	LessonsNote string   // "exact" or "section-level"
	Questions   int
	Score       int
}

// Report is the computed checklist.
type Report struct {
	Items         []Item
	VerifyRows    int
	Assumptions   int
	Observations  []string
	ResolvedCount int
	// Candidates are the [observation candidate] lines of the lessons, with the id of the section each lesson is in.
	Candidates []SectionCandidate
}

var citeRe = regexp.MustCompile(`//\s*rule:\s*([a-z0-9-]+)`)

// Build computes the report. contentDir is the content directory, simDir the engine package directory (for citation
// counts) and model the text of docs/SIMULATOR-MODEL.md.
func Build(contentDir, simDir, model string) (*Report, error) {
	fsys := os.DirFS(contentDir)
	course, err := content.Load(fsys)
	if err != nil {
		return nil, fmt.Errorf("content is invalid, fix it first: %w", err)
	}
	refs := content.ReadReferences(fsys)
	rows := content.SheetRows(refs)
	blocks, _ := content.ParseObservations(refs)
	sites, err := citationSites(simDir)
	if err != nil {
		return nil, err
	}
	assumptions := content.ParseAssumptions(model)

	lessonSection := map[string]string{}
	lessonText := map[string]string{}
	for _, s := range course.Sections {
		for _, l := range s.Lessons {
			lessonSection[l.Meta.ID] = s.Meta.ID
			lessonText[l.Meta.ID] = l.Body
		}
	}
	questionsByLesson := map[string]int{}
	questionsBySection := map[string]int{}
	for _, s := range course.Sections {
		for _, q := range s.Quiz {
			questionsByLesson[q.LessonRef]++
			questionsBySection[s.Meta.ID]++
		}
	}
	lessonsBySection := map[string][]string{}
	for _, s := range course.Sections {
		for _, l := range s.Lessons {
			lessonsBySection[s.Meta.ID] = append(lessonsBySection[s.Meta.ID], l.Meta.ID)
		}
	}

	rep := &Report{}
	for _, s := range course.Sections {
		for _, l := range s.Lessons {
			for _, c := range content.ParseCandidates(l.Meta.ID, l.Body) {
				rep.Candidates = append(rep.Candidates, SectionCandidate{Section: s.Meta.ID, ObservationCandidate: c})
			}
		}
	}
	obsFiles := map[string]bool{}
	for _, b := range blocks {
		obsFiles[b.File] = true
	}
	for f := range obsFiles {
		rep.Observations = append(rep.Observations, f)
	}
	sort.Strings(rep.Observations)

	attached := func(match func(content.ObservationRef) bool) []string {
		set := map[string]bool{}
		for _, b := range blocks {
			for _, r := range b.Resolves {
				if match(r) {
					set[b.File] = true
				}
			}
		}
		var out []string
		for f := range set {
			out = append(out, f)
		}
		sort.Strings(out)
		return out
	}

	// VERIFY rows.
	for _, r := range rows {
		if r.Status != "VERIFY" {
			continue
		}
		rep.VerifyRows++
		it := Item{Kind: "VERIFY row", ID: fmt.Sprintf("%s#%d", r.Sheet, r.Line), Sheet: r.Sheet, Line: r.Line, Text: r.Claim, LessonsNote: "section-level"}
		for _, rule := range sim.Rules {
			if rule.Status == sim.StatusNotModelled && rule.Sheet == r.Sheet && strings.Contains(r.Claim, rule.Claim) {
				it.Rules = append(it.Rules, rule.ID)
				it.Sites += sites[rule.ID]
			}
		}
		secs := SheetSections[r.Sheet]
		secSet := map[string]bool{}
		for _, sid := range secs {
			secSet[sid] = true
			it.Lessons = append(it.Lessons, lessonsBySection[sid]...)
			it.Questions += questionsBySection[sid]
		}
		for _, lab := range course.Labs() {
			for _, g := range lab.Lab.Goals {
				if secSet[lessonSection[g]] {
					it.Labs = append(it.Labs, lab.Lab.ID)
					break
				}
			}
		}
		sheet, line := r.Sheet, r.Line
		it.Observation = attached(func(ref content.ObservationRef) bool {
			if ref.Sheet != sheet || ref.Substring == "" {
				return false
			}
			for _, x := range rows {
				if x.Sheet == sheet && x.Line == line && strings.Contains(x.Claim, ref.Substring) {
					return true
				}
			}
			return false
		})
		// A VERIFY row is blocking content in proportion to the engine behaviour that is withheld because of it
		// (exact), plus a small weight for the section-level reach.
		it.Score = 6*len(it.Rules) + 2*it.Sites + len(it.Lessons) + len(it.Labs)
		rep.Items = append(rep.Items, it)
	}

	// Assumptions.
	nums := make([]int, 0, len(assumptions))
	for n := range assumptions {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		rep.Assumptions++
		it := Item{Kind: "assumption", ID: fmt.Sprintf("assumption %d", n), Number: n, Text: assumptions[n], LessonsNote: "exact"}
		for _, rule := range sim.Rules {
			if rule.Status == sim.StatusAssumption && rule.Assumption == n {
				it.Rules = append(it.Rules, rule.ID)
				it.Sites += sites[rule.ID]
			}
		}
		lessons := map[string]bool{}
		for _, lab := range course.Labs() {
			for _, a := range lab.Lab.RelyOn {
				if a == n {
					it.Labs = append(it.Labs, lab.Lab.ID)
					for _, g := range lab.Lab.Goals {
						lessons[g] = true
					}
				}
			}
		}
		re := regexp.MustCompile(fmt.Sprintf(`(?i)assumption\s+%d\b`, n))
		for id, body := range lessonText {
			if re.MatchString(body) {
				lessons[id] = true
			}
		}
		for id := range lessons {
			it.Lessons = append(it.Lessons, id)
			it.Questions += questionsByLesson[id]
		}
		sort.Strings(it.Lessons)
		num := n
		it.Observation = attached(func(ref content.ObservationRef) bool { return ref.Assumption == num })
		it.Score = 10*len(it.Labs) + 6*len(it.Rules) + 2*it.Sites + 3*len(it.Lessons) + it.Questions
		rep.Items = append(rep.Items, it)
	}

	for i := range rep.Items {
		sort.Strings(rep.Items[i].Rules)
		sort.Strings(rep.Items[i].Labs)
		if len(rep.Items[i].Observation) > 0 {
			rep.ResolvedCount++
		}
	}
	sort.SliceStable(rep.Items, func(i, j int) bool {
		a, b := rep.Items[i], rep.Items[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Kind != b.Kind {
			return a.Kind > b.Kind // assumptions before rows on a tie: they feed answers the learner sees
		}
		return a.ID < b.ID
	})
	return rep, nil
}

func citationSites(simDir string) (map[string]int, error) {
	out := map[string]int{}
	files, err := filepath.Glob(filepath.Join(simDir, "*.go"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no Go files in %s", simDir)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		for _, m := range citeRe.FindAllStringSubmatch(string(data), -1) {
			out[m[1]]++
		}
	}
	return out, nil
}

func excerpt(s string, n int) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}

func list(xs []string, max int) string {
	if len(xs) == 0 {
		return "-"
	}
	if len(xs) > max {
		return strings.Join(xs[:max], ", ") + fmt.Sprintf(" (+%d)", len(xs)-max)
	}
	return strings.Join(xs, ", ")
}

// Markdown renders the report. The output is deterministic (no timestamp) so a stale file is easy to spot with git diff.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# Observation priorities (your lab checklist)\n\n")
	b.WriteString("_Generated by `make observations-report`; do not edit by hand._\n\n")
	b.WriteString("Every item below is something the course could not confirm from an official source: a fact-sheet row marked VERIFY, or a numbered assumption of the simulator (`docs/SIMULATOR-MODEL.md`). Settling one with a real-VCO observation is the most valuable thing you can do in a lab session. Follow `docs/HOW-TO-ADD-AN-OBSERVATION.md`.\n\n")
	open := 0
	for _, it := range r.Items {
		if len(it.Observation) == 0 {
			open++
		}
	}
	fmt.Fprintf(&b, "- VERIFY rows: **%d**; assumptions: **%d**; items with an observation attached: **%d**; still open: **%d**.\n", r.VerifyRows, r.Assumptions, r.ResolvedCount, open)
	fmt.Fprintf(&b, "- Observation files: %s.\n\n", list(r.Observations, 20))
	for _, g := range candidateGroups {
		b.WriteString(r.candidatesMarkdown(g.Section, g.Title))
	}
	b.WriteString("## How the ranking works\n\n")
	b.WriteString("Items that block the most content come first. `score` = for an **assumption**: 10 per lab that relies on it + 6 per engine rule + 2 per place in the engine that cites it + 3 per lesson (a lab's goal lessons, or a lesson that names the assumption) + 1 per quiz question tied to those lessons. For a **VERIFY row**: 6 per not-modelled engine rule built on it + 2 per engine citation + 1 per lesson of the section(s) the sheet backs (section-level, because lessons are not tagged with rows) + 1 per lab in those sections. Labs never depend on a VERIFY row (the lab validator rejects an answer that does), so a VERIFY row ranks by what the engine withholds.\n\n")
	b.WriteString("Quiz questions never rest on a VERIFY row or an assumption (their answers use VERIFIED rows only); the count shows how many questions sit in the affected lessons and should be re-read if an observation corrects something.\n\n")

	b.WriteString("## Top 25\n\n")
	b.WriteString("| rank | item | score | what it is | engine rules | labs | lessons | quiz questions | observation |\n|---|---|---|---|---|---|---|---|---|\n")
	for i, it := range r.Items {
		if i >= 25 {
			break
		}
		fmt.Fprintf(&b, "| %d | %s | %d | %s | %s (%d sites) | %s | %d (%s) | %d | %s |\n", i+1, it.ID, it.Score, excerpt(it.Text, 110), list(it.Rules, 3), it.Sites,
			list(it.Labs, 4), len(it.Lessons), it.LessonsNote, it.Questions, list(it.Observation, 2))
	}

	b.WriteString("\n## All assumptions\n\n| assumption | score | text | engine rules (sites) | labs | lessons (exact) | quiz questions | observation |\n|---|---|---|---|---|---|---|---|\n")
	var as, vr []Item
	for _, it := range r.Items {
		if it.Kind == "assumption" {
			as = append(as, it)
		} else {
			vr = append(vr, it)
		}
	}
	sort.SliceStable(as, func(i, j int) bool { return as[i].Number < as[j].Number })
	for _, it := range as {
		fmt.Fprintf(&b, "| %d | %d | %s | %s (%d) | %s | %s | %d | %s |\n", it.Number, it.Score, excerpt(it.Text, 140), list(it.Rules, 2), it.Sites, list(it.Labs, 4), list(it.Lessons, 4), it.Questions, list(it.Observation, 2))
	}
	b.WriteString("\n## All VERIFY rows (by fact sheet)\n\n| row | score | claim | not-modelled engine rules (sites) | section-level lessons | observation |\n|---|---|---|---|---|---|\n")
	sort.SliceStable(vr, func(i, j int) bool {
		if vr[i].Sheet != vr[j].Sheet {
			return vr[i].Sheet < vr[j].Sheet
		}
		return vr[i].Line < vr[j].Line
	})
	for _, it := range vr {
		fmt.Fprintf(&b, "| %s | %d | %s | %s (%d) | %d | %s |\n", it.ID, it.Score, excerpt(it.Text, 140), list(it.Rules, 2), it.Sites, len(it.Lessons), list(it.Observation, 2))
	}
	return b.String()
}

// candidatesMarkdown renders one "Candidates from ..." section: every [observation candidate] line of the lessons of a course
// section, counted per lesson, with the screen to open, the documented expectation and where the documentation may differ.
func (r *Report) candidatesMarkdown(section, title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", title)
	var mine []SectionCandidate
	for _, c := range r.Candidates {
		if c.Section == section {
			mine = append(mine, c)
		}
	}
	if len(mine) == 0 {
		b.WriteString("No lesson lists an observation candidate.\n\n")
		return b.String()
	}
	var order []string
	per := map[string]int{}
	for _, c := range mine {
		if per[c.Lesson] == 0 {
			order = append(order, c.Lesson)
		}
		per[c.Lesson]++
	}
	fmt.Fprintf(&b, "Lessons that end with a \"Check this in your own VCO\" box mark each suggested check `[observation candidate]`. They are **not** observations and settle nothing: they are things to look at in a real VCO or site. **%d candidate(s) in %d lesson(s).** When you have seen one, follow `docs/HOW-TO-ADD-AN-OBSERVATION.md`.\n\n", len(mine), len(order))
	b.WriteString("| lesson | candidates |\n|---|---|\n")
	for _, id := range order {
		fmt.Fprintf(&b, "| %s | %d |\n", id, per[id])
	}
	b.WriteString("\n| lesson | screen to open | documented expectation | where the documentation may differ |\n|---|---|---|---|\n")
	for _, c := range mine {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", c.Lesson, cell(c.Screen), cell(c.Expect), cell(c.Differ))
	}
	b.WriteString("\n")
	return b.String()
}

func cell(s string) string { return strings.ReplaceAll(s, "|", "/") }
