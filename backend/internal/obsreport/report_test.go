package obsreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEverySheetIsMapped(t *testing.T) {
	files, _ := filepath.Glob("../../../content/reference/*.md")
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		if name == "README" {
			continue
		}
		if len(SheetSections[name]) == 0 {
			t.Errorf("fact sheet %s is not in SheetSections (obsreport/report.go): add the sections it backs", name)
		}
	}
}

func TestReportBuildsAndRanksBlockingItemsFirst(t *testing.T) {
	model, err := os.ReadFile("../../../docs/SIMULATOR-MODEL.md")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build("../../../content", "../sim", string(model))
	if err != nil {
		t.Fatal(err)
	}
	if r.Assumptions < 91 || r.VerifyRows == 0 {
		t.Fatalf("assumptions %d, verify rows %d", r.Assumptions, r.VerifyRows)
	}
	for i := 1; i < len(r.Items); i++ {
		if r.Items[i-1].Score < r.Items[i].Score {
			t.Fatalf("items %d and %d are out of order", i-1, i)
		}
	}
	md := r.Markdown()
	for _, want := range []string{"## Top 25", "## All assumptions", "## All VERIFY rows"} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q", want)
		}
	}
}

// The Go copy of the goals must match the frontend's goals.ts (sections and labs), so the course map and the dashboard agree.
func TestGoalsMatchTheFrontend(t *testing.T) {
	data, err := os.ReadFile("../../../frontend/src/progress/goals.ts")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, g := range Goals {
		i := strings.Index(src, "id: '"+g.ID+"'")
		if i < 0 {
			t.Errorf("goal %s is not in goals.ts", g.ID)
			continue
		}
		block := src[i:]
		if j := strings.Index(block[10:], "id: '"); j > 0 {
			if k := strings.Index(block[10+j:], "title:"); k >= 0 && k < 40 {
				block = block[:10+j]
			}
		}
		for _, s := range g.Sections {
			if !strings.Contains(block, "'"+s+"'") {
				t.Errorf("goal %s: section %s is missing in goals.ts", g.ID, s)
			}
		}
		for _, l := range g.Labs {
			if !strings.Contains(block, "'"+l+"'") {
				t.Errorf("goal %s: lab %s is missing in goals.ts", g.ID, l)
			}
		}
	}
}

func TestCourseMapBuilds(t *testing.T) {
	md, err := CourseMapMarkdown("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Goals and their coverage", "## Weak coverage and what would fix it", "Introduction", "capstone-2-hub-incident"} {
		if !strings.Contains(md, want) {
			t.Errorf("course map lacks %q", want)
		}
	}
}
