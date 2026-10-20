package content

import (
	"os"
	"strings"
	"testing"
)

const sheet = "# bgp\n\n| claim | source URL | doc version | status |\n|---|---|---|---|\n| The default of the Overlay Prefix checkbox is unknown. | none | 7.0 | VERIFY |\n| BGP uses local preference. | https://example.com | 7.0 | VERIFIED |\n"

func obsFile(date, release, resolves, effect string) string {
	return "# t observations\n\n## Observation 1: title\n\n" +
		"- Date: " + date + "\n- Release and build: " + release + "\n- Screen / menu path: A > B\n- What was done: x\n- What was seen: y\n- Resolves: " + resolves + "\n- Effect: " + effect + "\n\n" +
		"| claim | source URL | doc version | status |\n|---|---|---|---|\n| z | user observation 2026-10-20 | 7.0 (lab) | OBSERVED |\n"
}

func problems(t *testing.T, docs ...ReferenceDoc) string {
	t.Helper()
	var sb strings.Builder
	for _, p := range CheckObservations(docs, map[int]bool{22: true}) {
		sb.WriteString(p.File + ": " + p.Message + "\n")
	}
	return sb.String()
}

func TestObservationValidator(t *testing.T) {
	bgp := ReferenceDoc{Path: "reference/bgp.md", Content: sheet}
	cases := []struct {
		name, file, want string
	}{
		{"valid row reference", obsFile("2026-10-20", "7.0.0.2 build X", "bgp: Overlay Prefix checkbox", "confirms"), ""},
		{"valid assumption reference", obsFile("2026-10-20", "7.0.0.2", "assumption 22", "corrects"), ""},
		{"missing date", obsFile("", "7.0.0.2", "assumption 22", "confirms"), "missing Date"},
		{"bad date", obsFile("20/10/2026", "7.0.0.2", "assumption 22", "confirms"), "not YYYY-MM-DD"},
		{"missing release", obsFile("2026-10-20", "", "assumption 22", "confirms"), "missing Release"},
		{"unknown row", obsFile("2026-10-20", "7.0", "bgp: no such text anywhere", "confirms"), "no row of reference/bgp.md"},
		{"unknown sheet", obsFile("2026-10-20", "7.0", "nosheet: Overlay Prefix", "confirms"), "no row of reference/nosheet.md"},
		{"unknown assumption", obsFile("2026-10-20", "7.0", "assumption 999", "confirms"), "assumption 999 is not listed"},
		{"bad effect", obsFile("2026-10-20", "7.0", "assumption 22", "maybe"), "must be one of confirms"},
		{"ambiguous row", obsFile("2026-10-20", "7.0", "bgp: .", "confirms"), "rows of reference/bgp.md contain that text"},
	}
	for _, c := range cases {
		got := problems(t, bgp, ReferenceDoc{Path: "reference/observations/t.md", Content: c.file})
		if c.want == "" && got != "" {
			t.Errorf("%s: unexpected problems:\n%s", c.name, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: want %q in:\n%s", c.name, c.want, got)
		}
	}
}

func TestObservationFileWithoutABlockAndUnsupportedObservedRows(t *testing.T) {
	bare := ReferenceDoc{Path: "reference/observations/t.md", Content: "| claim | source URL | doc version | status |\n|---|---|---|---|\n| z | user observation 2026-10-20 | 7.0 | OBSERVED |\n"}
	got := problems(t, bare)
	if !strings.Contains(got, "no observation block") || !strings.Contains(got, "OBSERVED row needs an observation block") {
		t.Errorf("got:\n%s", got)
	}
	// An OBSERVED row in a fact sheet with no observation file at all is flagged.
	rogue := ReferenceDoc{Path: "reference/bgp.md", Content: sheet + "| Something seen. | user observation 2026-10-20 | 7.0 | OBSERVED |\n"}
	if got := problems(t, rogue); !strings.Contains(got, "OBSERVED row needs an observation block dated 2026-10-20") {
		t.Errorf("got:\n%s", got)
	}
	// ...and is accepted when an observation of that date names it.
	if got := problems(t, rogue, ReferenceDoc{Path: "reference/observations/t.md", Content: obsFile("2026-10-20", "7.0", "bgp: Something seen", "confirms")}); got != "" {
		t.Errorf("supported row flagged:\n%s", got)
	}
	// An observation of another date does not support it.
	if got := problems(t, rogue, ReferenceDoc{Path: "reference/observations/t.md", Content: obsFile("2026-10-19", "7.0", "bgp: Something seen", "confirms")}); !strings.Contains(got, "OBSERVED row needs") {
		t.Errorf("a differently dated observation must not support the row:\n%s", got)
	}
}

func TestTheSampleObservationIsWellFormed(t *testing.T) {
	data, err := os.ReadFile("../../../docs/SAMPLE-OBSERVATION.md")
	if err != nil {
		t.Fatal(err)
	}
	got := problems(t, ReferenceDoc{Path: "reference/observations/sample.md", Content: string(data)})
	if got != "" {
		t.Errorf("the sample must pass the validator:\n%s", got)
	}
}

func TestSimulatorModelListsEveryAssumption(t *testing.T) {
	model, err := os.ReadFile("../../../docs/SIMULATOR-MODEL.md")
	if err != nil {
		t.Fatal(err)
	}
	m := ParseAssumptions(string(model))
	if len(m) < 91 {
		t.Fatalf("found %d assumptions, want at least 91", len(m))
	}
	for i := 1; i <= len(m); i++ {
		if m[i] == "" {
			t.Errorf("assumption %d is missing from the numbered list", i)
		}
	}
}

func TestTheRealObservationFolderIsValid(t *testing.T) {
	refs := ReadReferences(os.DirFS("../../../content"))
	model, _ := os.ReadFile("../../../docs/SIMULATOR-MODEL.md")
	for _, p := range CheckObservations(refs, ModelAssumptions(string(model))) {
		t.Errorf("%s: %s", p.File, p.Message)
	}
}

// Every lesson ends with the "what would you look for in the VCO" bridge (the last ## heading of the file).
func TestEveryLessonEndsWithTheVCOBridge(t *testing.T) {
	c, err := Load(os.DirFS("../../../content"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range c.Sections {
		for _, l := range s.Lessons {
			n++
			last := ""
			for _, line := range strings.Split(l.Body, "\n") {
				if strings.HasPrefix(line, "## ") {
					last = strings.TrimSpace(strings.TrimPrefix(line, "## "))
				}
			}
			if last != "What would you look for in the VCO" {
				t.Errorf("%s ends with %q, want the \"What would you look for in the VCO\" bridge", l.Path, last)
			}
		}
	}
	if n < 42 {
		t.Errorf("only %d lessons checked", n)
	}
}

// Lessons build on earlier ones: every prerequisite comes earlier in the course, and every lesson_ref of a quiz question
// points at a lesson of the same section.
func TestLessonOrderBuildsOnPrerequisites(t *testing.T) {
	c, err := Load(os.DirFS("../../../content"))
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	n := 0
	for _, s := range c.Sections {
		for _, l := range s.Lessons {
			pos[l.Meta.ID] = n
			n++
		}
	}
	for _, s := range c.Sections {
		for _, l := range s.Lessons {
			for _, p := range l.Meta.Prerequisites {
				if pos[p] >= pos[l.Meta.ID] {
					t.Errorf("%s has prerequisite %s, which comes later in the course", l.Meta.ID, p)
				}
			}
		}
		for _, q := range s.Quiz {
			if lesson, ok := c.Lesson(q.LessonRef); q.LessonRef != "" && (!ok || lesson.SectionID != s.Meta.ID) {
				t.Errorf("question %s: lesson_ref %q is not a lesson of section %s", q.ID, q.LessonRef, s.Meta.ID)
			}
		}
	}
}
