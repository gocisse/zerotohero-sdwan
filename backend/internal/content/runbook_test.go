package content

import (
	"os"
	"strings"
	"testing"
)

func realBundle(t *testing.T) *Bundle {
	t.Helper()
	b, errs := ReadBundle(os.DirFS("../../../content"))
	if len(errs) > 0 {
		t.Fatalf("content has read errors: %v", errs)
	}
	return b
}

func TestRunbooksValidateAndAreReported(t *testing.T) {
	b := realBundle(t)
	if len(b.Runbooks) < 4 {
		t.Fatalf("got %d runbooks, want at least 4", len(b.Runbooks))
	}
	if errs := Validate(b); len(errs) > 0 {
		t.Fatalf("the real runbooks do not validate: %v", errs)
	}
}

// Every problem of every runbook is reported together, with the file name.
func TestRunbookProblemsAreAllReported(t *testing.T) {
	b := realBundle(t)
	rs := b.Runbooks
	rs[0].Meta.Summary = ""
	rs[1].Meta.Kind = "poster"
	rs[2].Meta.ID = rs[3].Meta.ID
	rs[3].Body += "\n<Term id=\"no-such-term\" />\n<Diagram id=\"no-such-diagram\" />\n"
	got := Validate(b).Error()
	for _, want := range []string{"summary is required", "kind must be one of", "duplicate runbook id", `<Term id="no-such-term"`, `<Diagram id="no-such-diagram"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A study guide topic rated "full" with no lesson, in a section the course added, is an error like in any other section.
func TestAddedSectionFullTopicNeedsALesson(t *testing.T) {
	b := realBundle(t)
	var g6 *GuideSection
	for i := range b.Guide.Sections {
		if b.Guide.Sections[i].ID == "G6" {
			g6 = &b.Guide.Sections[i]
		}
	}
	if g6 == nil || !g6.AddedByCourse {
		t.Fatal("G6 must exist and be marked added_by_course")
	}
	g6.Topics[0].Coverage = CoverageFull
	g6.Topics[0].Lessons = nil
	if got := Validate(b).Error(); !strings.Contains(got, "ts-method") {
		t.Errorf("a full topic without a lesson was not reported:\n%s", got)
	}
}
