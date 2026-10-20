package content

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func fstestFS(files map[string]string) fs.FS {
	m := fstest.MapFS{}
	for k, v := range files {
		m[k] = &fstest.MapFile{Data: []byte(v)}
	}
	return m
}

func TestLoadValidFixture(t *testing.T) {
	c, err := Load(os.DirFS("testdata/valid"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Meta.Title != "Test Course" || len(c.Sections) != 1 {
		t.Fatalf("unexpected course: %+v", c.Meta)
	}
	s := c.Sections[0]
	if len(s.Lessons) != 2 || s.Lessons[0].Meta.ID != "first" || s.Lessons[1].Meta.ID != "second" {
		t.Fatalf("lessons out of order: %+v", s.Lessons)
	}
	second, ok := c.Lesson("second")
	if !ok {
		t.Fatal("lesson second not indexed")
	}
	if !strings.Contains(second.Body, "horizontal rule above") || strings.Contains(second.Body, "prerequisites") {
		t.Errorf("body should exclude frontmatter but keep the rest: %q", second.Body)
	}
	if got := second.Meta.Prerequisites; len(got) != 1 || got[0] != "first" {
		t.Errorf("prerequisites = %v", got)
	}
	d, ok := c.Diagram("net")
	if !ok || len(d.Steps) != 1 || d.Steps[0].Packets[0].Route == nil || len(d.Steps[0].Packets[0].Route.Links) != 2 {
		t.Fatalf("diagram not loaded with resolved routes: %+v", d)
	}
	if len(c.Glossary) != 2 || c.Glossary[0].ID != "widget" || c.Glossary[1].Aliases == nil {
		t.Fatalf("glossary = %+v", c.Glossary)
	}
	if ts := c.Topologies(); len(ts) != 1 || ts[0].T.Meta.ID != "tiny" {
		t.Fatalf("topologies = %+v", ts)
	}
	if _, ok := c.Topology("tiny"); !ok {
		t.Error("topology not indexed by id")
	}
	tf := s.Quiz[1]
	if tf.Type != TrueFalse || len(tf.Options) != 2 || tf.Correct[0] != "true" {
		t.Errorf("true_false not normalized: %+v", tf)
	}
}

func TestLoadInvalidFixtureReportsEverything(t *testing.T) {
	_, err := Load(os.DirFS("testdata/invalid"))
	var list ErrorList
	if !errors.As(err, &list) {
		t.Fatalf("want ErrorList, got %T: %v", err, err)
	}
	// Reference errors (unknown prerequisite, unlisted section) are deliberately
	// suppressed here because 02-bad.mdx failed to parse and could be the target;
	// validate_test.go covers those rules.
	want := []struct{ file, contains string }{
		{"sections/01-intro/lessons/02-bad.mdx", "must start with a '---'"},
		{"sections/01-intro/quiz.yaml", "exactly one correct option"},
		{"sections/02-ghost/section.yaml", "cannot read file"},
		{"diagrams/broken.yaml", "yaml"},
		{"diagrams/bad-link.yaml", `to "ghost" is not a node`},
		{"topologies/dangling.yaml", `unknown profile "ghost"`},
		{"topologies/dangling.yaml", "duplicate edge id"},
		{"topologies/dangling.yaml", "unknown setting"},
		{"topologies/syntax.yaml", "surprise"},
		{"stray/", "orphan"},
		{"sections/02-ghost/note.txt", "orphan"},
	}
	for _, w := range want {
		found := false
		for _, e := range list {
			if e.File == w.file && strings.Contains(e.Message, w.contains) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing error %s: %q in:\n%v", w.file, w.contains, list)
		}
	}
}

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name, in, wantBody string
		wantErr            bool
	}{
		{"basic", "---\na: 1\n---\nbody\n", "body\n", false},
		{"crlf", "---\r\na: 1\r\n---\r\nbody\r\n", "body\n", false},
		{"no frontmatter", "body", "", true},
		{"unclosed", "---\na: 1\nbody", "", true},
		{"closing with trailing text", "---\na: 1\n---x\nbody", "", true},
		{"empty body", "---\na: 1\n---", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, body, _, err := splitFrontmatter([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestTopologyFileNameMustMatchID(t *testing.T) {
	fsys := fstestFS(map[string]string{
		"course.yaml":           "title: T\ndescription: d\nsections: []\n",
		"topologies/other.yaml": "version: 1\nmeta: { id: tiny, title: T, summary: s, teaching: true }\ncustomer: { id: c, name: C }\nprofiles: []\nsites: []\n",
	})
	_, err := Load(fsys)
	var list ErrorList
	if !errors.As(err, &list) {
		t.Fatalf("want ErrorList, got %v", err)
	}
	found := false
	for _, e := range list {
		found = found || (e.File == "topologies/other.yaml" && strings.Contains(e.Message, "must equal the file name"))
	}
	if !found {
		t.Errorf("missing file-name error in %v", list)
	}
}
