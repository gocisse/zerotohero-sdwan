package content

import (
	"strings"
	"testing"
	"testing/fstest"

	"velocloud-academy/backend/internal/sim"
)

// guideBundle is validBundle plus a valid study guide, a diagram, a lab and a glossary that uses guide topics. Only the
// guide rules are exercised (checkGuide), so the diagram and lab are bare stubs.
func guideBundle() *Bundle {
	b := validBundle()
	b.Diagrams = []DiagramDoc{{Path: "diagrams/d1.yaml", D: Diagram{ID: "d1"}}}
	b.Labs = []LabDoc{{Path: "labs/lab1.yaml", Lab: &sim.Lab{ID: "lab1"}}}
	b.GlossaryPath, b.HasGlossary = "glossary.yaml", true
	b.Glossary = []GlossaryTerm{
		{ID: "x", Term: "X", Short: "s", GuideTopics: []string{"t-one"}},
		{ID: "y", Term: "Y", Short: "s", BeyondGuide: true},
	}
	b.GuidePath, b.HasGuide = "study-guide.yaml", true
	b.Guide = StudyGuide{
		Beyond:          GuideBeyond{Title: "Beyond", Reason: "because"},
		UnmappedLessons: []string{"l3"},
		Sections: []GuideSection{{ID: "G1", Title: "One", Topics: []GuideTopic{
			{ID: "t-one", Title: "T1", Kind: "concept", Coverage: CoverageFull, Note: "n", Lessons: []string{"l1"}, QuizTopics: []string{"t"}, Diagrams: []string{"d1"}, Labs: []string{"lab1"}, SimScreens: []string{"map"}},
			{ID: "t-two", Title: "T2", Kind: "config", Coverage: CoveragePartial, Note: "n", Lessons: []string{"l2"}},
			{ID: "t-three", Title: "T3", Kind: "topology", Coverage: CoverageNone, Note: "n"},
		}}},
	}
	return b
}

func TestGuideValidation(t *testing.T) {
	topic := func(b *Bundle, id string) *GuideTopic {
		for si := range b.Guide.Sections {
			for ti := range b.Guide.Sections[si].Topics {
				if b.Guide.Sections[si].Topics[ti].ID == id {
					return &b.Guide.Sections[si].Topics[ti]
				}
			}
		}
		t.Fatalf("no topic %s", id)
		return nil
	}
	tests := []struct {
		name     string
		mutate   func(b *Bundle)
		wantFile string // empty means valid
		wantMsg  string
	}{
		{"valid guide", func(b *Bundle) {}, "", ""},
		{"no guide and no glossary use is valid", func(b *Bundle) {
			b.HasGuide = false
			b.Glossary = []GlossaryTerm{{ID: "x", Term: "X", Short: "s"}}
		}, "", ""},
		{"glossary uses guide topics without a guide", func(b *Bundle) { b.HasGuide = false }, "glossary.yaml", "no study-guide.yaml"},
		{"beyond reason missing", func(b *Bundle) { b.Guide.Beyond.Reason = " " }, "study-guide.yaml", "beyond.reason is required"},
		{"no sections", func(b *Bundle) { b.Guide.Sections = nil }, "study-guide.yaml", "sections must list at least one"},
		{"duplicate section id", func(b *Bundle) {
			b.Guide.Sections = append(b.Guide.Sections, GuideSection{ID: "G1", Title: "Again", Topics: []GuideTopic{{ID: "t-four", Title: "T4", Kind: "concept", Coverage: CoverageNone, Note: "n"}}})
		}, "study-guide.yaml", "duplicate section id"},
		{"duplicate topic id", func(b *Bundle) { topic(b, "t-two").ID = "t-one" }, "study-guide.yaml", "duplicate topic id"},
		{"topic bad id", func(b *Bundle) { topic(b, "t-two").ID = "Bad_Id" }, "study-guide.yaml", "must be lowercase"},
		{"topic bad kind", func(b *Bundle) { topic(b, "t-two").Kind = "lab" }, "study-guide.yaml", "kind must be concept, config or topology"},
		{"topic note missing", func(b *Bundle) { topic(b, "t-two").Note = "" }, "study-guide.yaml", "note is required"},
		{"topic bad coverage", func(b *Bundle) { topic(b, "t-two").Coverage = "most" }, "study-guide.yaml", "coverage must be full, partial or none"},
		{"full without a lesson", func(b *Bundle) { topic(b, "t-one").Lessons = nil }, "study-guide.yaml", "coverage full needs at least one lesson"},
		{"partial without a lesson", func(b *Bundle) { topic(b, "t-two").Lessons = nil }, "study-guide.yaml", "coverage partial needs at least one lesson"},
		{"none with a lesson", func(b *Bundle) { topic(b, "t-three").Lessons = []string{"l2"} }, "study-guide.yaml", "coverage none must have no lessons"},
		{"missing lesson", func(b *Bundle) { topic(b, "t-two").Lessons = []string{"nope"} }, "study-guide.yaml", `lessons "nope" does not exist`},
		{"missing quiz topic", func(b *Bundle) { topic(b, "t-one").QuizTopics = []string{"nope"} }, "study-guide.yaml", `quiz_topics "nope" does not exist`},
		{"missing diagram", func(b *Bundle) { topic(b, "t-one").Diagrams = []string{"nope"} }, "study-guide.yaml", `diagrams "nope" does not exist`},
		{"missing lab", func(b *Bundle) { topic(b, "t-one").Labs = []string{"nope"} }, "study-guide.yaml", `labs "nope" does not exist`},
		{"unknown simulator screen", func(b *Bundle) { topic(b, "t-one").SimScreens = []string{"nope"} }, "study-guide.yaml", `sim_screens "nope" does not exist`},
		{"duplicate reference", func(b *Bundle) { topic(b, "t-one").Diagrams = []string{"d1", "d1"} }, "study-guide.yaml", `diagrams lists "d1" more than once`},
		{"unmapped lesson that does not exist", func(b *Bundle) { b.Guide.UnmappedLessons = []string{"l3", "nope"} }, "study-guide.yaml", `unmapped_lessons "nope" does not exist`},
		{"unmapped lesson that a topic maps", func(b *Bundle) { b.Guide.UnmappedLessons = []string{"l3", "l1"} }, "study-guide.yaml", `unmapped_lessons "l1" is referenced by a topic`},
		{"term without topics or beyond flag", func(b *Bundle) { b.Glossary[1].BeyondGuide = false }, "glossary.yaml", "needs guide_topics or beyond_guide"},
		{"term with both", func(b *Bundle) { b.Glossary[0].BeyondGuide = true }, "glossary.yaml", "has both guide_topics and beyond_guide"},
		{"term with unknown topic", func(b *Bundle) { b.Glossary[0].GuideTopics = []string{"t-one", "nope"} }, "glossary.yaml", `guide_topics "nope" is not a topic`},
		{"term lists a topic twice", func(b *Bundle) { b.Glossary[0].GuideTopics = []string{"t-one", "t-one"} }, "glossary.yaml", "more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := guideBundle()
			tt.mutate(b)
			errs, _ := checkGuide(b)
			if tt.wantFile == "" {
				if len(errs) != 0 {
					t.Fatalf("expected a valid bundle, got: %v", errs)
				}
				return
			}
			for _, e := range errs {
				if e.File == tt.wantFile && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("want error in %s containing %q, got: %v", tt.wantFile, tt.wantMsg, errs)
		})
	}
}

func TestGuideReportsEveryProblemTogether(t *testing.T) {
	b := guideBundle()
	b.Guide.Sections[0].Topics[1].Lessons = []string{"missing-lesson"}
	b.Guide.Sections[0].Topics[2].Lessons = []string{"l2"}
	b.Glossary[0].GuideTopics = []string{"unknown-topic"}
	errs, _ := checkGuide(b)
	for _, want := range []string{`lessons "missing-lesson" does not exist`, "coverage none must have no lessons", `guide_topics "unknown-topic" is not a topic`} {
		found := false
		for _, e := range errs {
			found = found || strings.Contains(e.Message, want)
		}
		if !found {
			t.Errorf("missing error %q in %v", want, errs)
		}
	}
}

func TestGuideWarnsAboutUnmappedLessons(t *testing.T) {
	b := guideBundle()
	b.Guide.UnmappedLessons = nil // l3 is now neither mapped nor listed
	if errs, _ := checkGuide(b); len(errs) != 0 {
		t.Fatalf("an unmapped lesson must not be an error: %v", errs)
	}
	_, warns := checkGuide(b)
	if len(warns) != 1 || !strings.Contains(warns[0].Message, `"l3"`) {
		t.Fatalf("want one warning naming l3, got %v", warns)
	}
	if _, warns := checkGuide(guideBundle()); len(warns) != 0 {
		t.Fatalf("a listed unmapped lesson is not a warning: %v", warns)
	}
}

func TestGuideLoadsFromFilesAndRejectsTypos(t *testing.T) {
	fsys := fstest.MapFS{"study-guide.yaml": {Data: []byte("beyond: {title: B, reason: r}\nsections: []\nsurprise: 1\n")}}
	b, errs := ReadBundle(fsys)
	found := false
	for _, e := range errs {
		found = found || (e.File == "study-guide.yaml" && strings.Contains(e.Message, "surprise"))
	}
	if !found || !b.GuideBroken {
		t.Fatalf("an unknown key must be an error: %v", errs)
	}
}
