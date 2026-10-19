package content

import (
	"strings"
	"testing"
)

func validDiagram() Diagram {
	return Diagram{
		ID: "dg", Title: "T", Alt: "alt text", Width: 800, Height: 400,
		Nodes: []DiagramNode{
			{ID: "a", Label: "A", Kind: "host", X: 50, Y: 50, Group: "site"},
			{ID: "b", Label: "B", Kind: "vce", X: 200, Y: 50, Group: "site"},
			{ID: "net", Label: "Net", Kind: "internet", X: 400, Y: 50},
			{ID: "c", Label: "C", Kind: "datacenter", X: 600, Y: 200, Sublabel: "sub"},
		},
		Groups: []DiagramGroup{{ID: "site", Label: "Site"}, {ID: "boxed", Label: "Boxed", Box: &DiagramBox{X: 500, Y: 100, Width: 200, Height: 200}}},
		Links: []DiagramLink{
			{ID: "l-ab", From: "a", To: "b", Style: "lan"},
			{ID: "l-bn", From: "b", To: "net", Style: "underlay-internet", Label: "wan"},
			{ID: "l-nc", From: "net", To: "c", Style: "overlay-tunnel", Curve: 30},
		},
		Steps: []DiagramStep{
			{Title: "One", Narration: "n", Highlight: []string{"a", "l-ab"}, Show: []string{"c"}},
			{Title: "Two", Narration: "n", Packets: []DiagramPacket{{Path: []string{"a", "b", "net", "c"}, Label: "p", ColorStyle: "overlay-tunnel", DelayMS: 200}},
				Callouts: []DiagramCallout{{Target: "net", Text: "hi"}}},
		},
	}
}

func TestValidateDiagram(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(d *Diagram)
		wantMsg string // empty = valid
	}{
		{"valid", func(d *Diagram) {}, ""},
		{"valid static diagram (zero steps)", func(d *Diagram) { d.Steps = nil }, ""},
		{"valid path mixing link ids", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"a", "l-ab", "l-bn", "net", "l-nc"} }, ""},
		{"valid path traveling a link backwards", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"c", "net", "b"} }, ""},

		{"bad id format", func(d *Diagram) { d.ID = "Bad_Id" }, "id \"Bad_Id\" must be lowercase"},
		{"missing title", func(d *Diagram) { d.Title = "" }, "title is required"},
		{"missing alt", func(d *Diagram) { d.Alt = "  " }, "alt is required"},
		{"canvas size", func(d *Diagram) { d.Width = 0 }, "width and height must be positive"},
		{"no nodes", func(d *Diagram) { d.Nodes, d.Links, d.Steps, d.Groups = nil, nil, nil, nil }, "at least one node"},

		{"duplicate node id", func(d *Diagram) { d.Nodes[1].ID = "a" }, "duplicate node id"},
		{"node missing label", func(d *Diagram) { d.Nodes[0].Label = "" }, "label is required"},
		{"unknown node kind", func(d *Diagram) { d.Nodes[0].Kind = "toaster" }, `unknown kind "toaster"`},
		{"node outside canvas (right)", func(d *Diagram) { d.Nodes[0].X = 900 }, "outside the 800x400 canvas"},
		{"node outside canvas (negative)", func(d *Diagram) { d.Nodes[0].Y = -1 }, "outside the 800x400 canvas"},
		{"nodes at the same position", func(d *Diagram) { d.Nodes[1].X, d.Nodes[1].Y = 50, 50 }, "exactly the same position"},
		{"node in unknown group", func(d *Diagram) { d.Nodes[0].Group = "nope" }, `group "nope" does not exist`},

		{"group member missing", func(d *Diagram) { d.Groups[0].Nodes = []string{"ghost"} }, `member node "ghost" does not exist`},
		{"group without members or box", func(d *Diagram) { d.Groups = append(d.Groups, DiagramGroup{ID: "empty", Label: "E"}) }, "needs member nodes or an explicit box"},
		{"duplicate group id", func(d *Diagram) { d.Groups[1].ID = "site" }, "duplicate group id"},
		{"group box size", func(d *Diagram) { d.Groups[1].Box.Width = 0 }, "box width and height"},

		{"duplicate link id", func(d *Diagram) { d.Links[1].ID = "l-ab" }, "duplicate link id"},
		{"link id clashes with node id", func(d *Diagram) { d.Links[0].ID = "a" }, "also used by a node"},
		{"link to missing node (from)", func(d *Diagram) { d.Links[0].From = "ghost" }, `from "ghost" is not a node`},
		{"link to missing node (to)", func(d *Diagram) { d.Links[0].To = "ghost" }, `to "ghost" is not a node`},
		{"link to itself", func(d *Diagram) { d.Links[0].To = "a" }, "same node"},
		{"unknown link style", func(d *Diagram) { d.Links[0].Style = "wifi" }, `unknown style "wifi"`},

		{"step without title", func(d *Diagram) { d.Steps[0].Title = "" }, "step 1: title is required"},
		{"step without narration", func(d *Diagram) { d.Steps[0].Narration = "" }, "step 1: narration is required"},
		{"step references nothing", func(d *Diagram) { d.Steps[0] = DiagramStep{Title: "x", Narration: "y"} }, "references nothing"},
		{"highlight unknown id", func(d *Diagram) { d.Steps[0].Highlight = []string{"ghost"} }, `highlight refers to unknown node or link "ghost"`},
		{"show unknown id", func(d *Diagram) { d.Steps[0].Show = []string{"ghost"} }, `show refers to unknown`},
		{"hide unknown id", func(d *Diagram) { d.Steps[0].Hide = []string{"ghost"} }, `hide refers to unknown`},
		{"callout unknown target", func(d *Diagram) { d.Steps[1].Callouts[0].Target = "ghost" }, `callout 1 targets unknown node or link "ghost"`},
		{"callout without text", func(d *Diagram) { d.Steps[1].Callouts[0].Text = "" }, "callout 1 has no text"},
		{"packet path skips a link", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"a", "net"} }, `no link connects "a" and "net"`},
		{"packet path unknown id", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"a", "ghost"} }, `unknown node or link id "ghost"`},
		{"packet path starts with a link", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"l-ab", "b"} }, "must start with a node id"},
		{"packet link does not touch current node", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"a", "l-nc"} }, `link "l-nc" does not touch "a"`},
		{"packet path empty", func(d *Diagram) { d.Steps[1].Packets[0].Path = nil }, "path is empty"},
		{"packet path single node", func(d *Diagram) { d.Steps[1].Packets[0].Path = []string{"a"} }, "at least one link"},
		{"packet unknown color_style", func(d *Diagram) { d.Steps[1].Packets[0].ColorStyle = "rainbow" }, `unknown color_style "rainbow"`},
		{"packet negative delay", func(d *Diagram) { d.Steps[1].Packets[0].DelayMS = -5 }, "delay_ms must not be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDiagram()
			tt.mutate(&d)
			var errs ErrorList
			validateDiagram(&errs, "diagrams/dg.yaml", d)
			if tt.wantMsg == "" {
				if len(errs) != 0 {
					t.Fatalf("expected valid, got:\n%v", errs)
				}
				return
			}
			for _, e := range errs {
				if e.File == "diagrams/dg.yaml" && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("no error containing %q; got:\n%v", tt.wantMsg, errs)
		})
	}
}

func TestResolveRoute(t *testing.T) {
	d := validDiagram()
	tests := []struct {
		path      []string
		wantNodes string
		wantLinks string
	}{
		{[]string{"a", "b", "net", "c"}, "a,b,net,c", "l-ab,l-bn,l-nc"},
		{[]string{"a", "l-ab", "b"}, "a,b", "l-ab"},
		{[]string{"c", "net"}, "c,net", "l-nc"},
		{[]string{"a", "l-ab", "l-bn"}, "a,b,net", "l-ab,l-bn"},
	}
	for _, tt := range tests {
		r, err := d.resolveRoute(tt.path)
		if err != nil {
			t.Fatalf("%v: %v", tt.path, err)
		}
		if strings.Join(r.Nodes, ",") != tt.wantNodes || strings.Join(r.Links, ",") != tt.wantLinks {
			t.Errorf("%v -> nodes=%v links=%v", tt.path, r.Nodes, r.Links)
		}
	}
}

func TestNormalizedFillsRoutesAndEmptyLists(t *testing.T) {
	d := validDiagram().normalized()
	if d.Steps[1].Packets[0].Route == nil || len(d.Steps[1].Packets[0].Route.Links) != 3 {
		t.Fatalf("route not resolved: %+v", d.Steps[1].Packets[0])
	}
	if d.Steps[0].Hide == nil || d.Steps[0].Callouts == nil || d.Steps[0].Packets == nil {
		t.Error("nil lists would serialize as null")
	}
}

func TestValidateGlossary(t *testing.T) {
	lessons := map[string]string{"les": "p.mdx"}
	valid := func() []GlossaryTerm {
		return []GlossaryTerm{
			{ID: "alpha", Term: "Alpha", Aliases: []string{"A1"}, Short: "s", Related: []string{"beta"}, LessonRef: "les"},
			{ID: "beta", Term: "Beta", Short: "s"},
		}
	}
	tests := []struct {
		name    string
		mutate  func(g []GlossaryTerm) []GlossaryTerm
		wantMsg string
	}{
		{"valid", func(g []GlossaryTerm) []GlossaryTerm { return g }, ""},
		{"duplicate id", func(g []GlossaryTerm) []GlossaryTerm { g[1].ID = "alpha"; return g }, "duplicate term id"},
		{"bad id format", func(g []GlossaryTerm) []GlossaryTerm { g[1].ID = "Beta Term"; return g }, "must be lowercase"},
		{"missing id", func(g []GlossaryTerm) []GlossaryTerm { g[1].ID = ""; return g }, "id is required"},
		{"missing term", func(g []GlossaryTerm) []GlossaryTerm { g[1].Term = ""; return g }, "term is required"},
		{"missing short", func(g []GlossaryTerm) []GlossaryTerm { g[1].Short = " "; return g }, "short"},
		{"unknown related", func(g []GlossaryTerm) []GlossaryTerm { g[0].Related = []string{"ghost"}; return g }, `related term "ghost" does not exist`},
		{"related to itself", func(g []GlossaryTerm) []GlossaryTerm { g[0].Related = []string{"alpha"}; return g }, "lists itself as related"},
		{"unknown lesson_ref", func(g []GlossaryTerm) []GlossaryTerm { g[0].LessonRef = "ghost"; return g }, `lesson_ref "ghost"`},
		{"alias collides with another term", func(g []GlossaryTerm) []GlossaryTerm { g[1].Aliases = []string{"a1"}; return g }, "also the name or alias"},
		{"empty alias", func(g []GlossaryTerm) []GlossaryTerm { g[0].Aliases = []string{" "}; return g }, "empty alias"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errs ErrorList
			validateGlossary(&errs, "glossary.yaml", tt.mutate(valid()), lessons, false)
			if tt.wantMsg == "" {
				if len(errs) != 0 {
					t.Fatalf("expected valid, got %v", errs)
				}
				return
			}
			for _, e := range errs {
				if e.File == "glossary.yaml" && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("no error containing %q; got %v", tt.wantMsg, errs)
		})
	}
}

func TestScanTags(t *testing.T) {
	body := "intro\n<Diagram id=\"a\" />\n\n```mdx\n<Diagram id=\"in-fence\" />\n```\n\nUse `<Term id=\"inline\"/>` here and <Term id='b'>text</Term> and <Term id={\"c\"}/>.\n<Diagram title=\"no id\" />\n<DiagramX id=\"not-a-diagram\" />\n"
	d := scanTags(body, "Diagram")
	if len(d) != 2 || d[0].ID != "a" || d[0].Line != 2 || d[1].HasID || d[1].Line != 9 {
		t.Errorf("diagrams = %+v", d)
	}
	tm := scanTags(body, "Term")
	if len(tm) != 2 || tm[0].ID != "b" || tm[1].ID != "c" {
		t.Errorf("terms = %+v", tm)
	}
}

func TestValidateComponentReferencesInLessons(t *testing.T) {
	setup := func(body string) *Bundle {
		b := validBundle()
		d := validDiagram()
		b.Diagrams = []DiagramDoc{{Path: "diagrams/dg.yaml", D: d}}
		b.HasGlossary, b.GlossaryPath = true, "glossary.yaml"
		b.Glossary = []GlossaryTerm{{ID: "known", Term: "Known", Short: "s"}}
		b.Sections[0].Lessons[0].Body = body
		b.Sections[0].Lessons[0].BodyLine = 6
		return b
	}
	tests := []struct {
		name, body, wantMsg string
	}{
		{"valid references", "<Diagram id=\"dg\" />\n<Term id=\"known\" />", ""},
		{"unknown diagram id", "x\n<Diagram id=\"ghost\" />", `line 8: <Diagram id="ghost"`},
		{"diagram without id", "<Diagram title=\"x\" />", "<Diagram> needs an id"},
		{"unknown term id", "see <Term id=\"ghost\">x</Term>", `<Term id="ghost"`},
		{"term without id", "<Term>x</Term>", "<Term> needs an id"},
		{"references inside code are ignored", "```\n<Diagram id=\"ghost\" />\n```\n`<Term id=\"ghost\"/>`", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(setup(tt.body))
			if tt.wantMsg == "" {
				if len(errs) != 0 {
					t.Fatalf("expected valid, got %v", errs)
				}
				return
			}
			for _, e := range errs {
				if e.File == "sections/01-a/lessons/01-l1.mdx" && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("no error containing %q; got %v", tt.wantMsg, errs)
		})
	}
}

func TestValidateDiagramFiles(t *testing.T) {
	b := validBundle()
	d1, d2 := validDiagram(), validDiagram()
	b.Diagrams = []DiagramDoc{{Path: "diagrams/one.yaml", D: d1}, {Path: "diagrams/two.yaml", D: d2}}
	errs := Validate(b)
	found := false
	for _, e := range errs {
		if e.File == "diagrams/two.yaml" && strings.Contains(e.Message, `duplicate diagram id "dg"`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate diagram id not reported: %v", errs)
	}
}
