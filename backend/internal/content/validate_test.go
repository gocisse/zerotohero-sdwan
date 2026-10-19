package content

import (
	"strings"
	"testing"
)

func validBundle() *Bundle {
	return &Bundle{
		CoursePath: "course.yaml",
		Course:     CourseMeta{Title: "T", Description: "D", Sections: []string{"s1", "s2"}},
		Sections: []SectionBundle{
			{
				Dir: "sections/01-a", MetaPath: "sections/01-a/section.yaml",
				Meta: SectionMeta{ID: "s1", Title: "S1", Summary: "sum", EstimatedMinutes: 5, Lessons: []string{"l1", "l2"}},
				Lessons: []LessonDoc{
					{Path: "sections/01-a/lessons/01-l1.mdx", Body: "x", Meta: LessonMeta{ID: "l1", Title: "L1", Summary: "s", EstimatedMinutes: 2, Topics: []string{"t"}}},
					{Path: "sections/01-a/lessons/02-l2.mdx", Body: "x", Meta: LessonMeta{ID: "l2", Title: "L2", Summary: "s", EstimatedMinutes: 2, Topics: []string{"t"}, Prerequisites: []string{"l1"}}},
				},
				QuizPath: "sections/01-a/quiz.yaml", HasQuiz: true,
				Quiz: []Question{
					{ID: "q-single", Topic: "t", Difficulty: 1, Type: SingleChoice, Prompt: "p", Explanation: "e", LessonRef: "l1",
						Options: []Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, Correct: StringList{"a"}},
					{ID: "q-multi", Topic: "t", Difficulty: 2, Type: MultiChoice, Prompt: "p", Explanation: "e",
						Options: []Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}}, Correct: StringList{"a", "b"}},
					{ID: "q-tf", Topic: "t", Difficulty: 3, Type: TrueFalse, Prompt: "p", Explanation: "e", Correct: StringList{"false"}},
					{ID: "q-order", Topic: "t", Difficulty: 2, Type: Ordering, Prompt: "p", Explanation: "e",
						Options: []Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}}, Correct: StringList{"c", "a", "b"}},
				},
			},
			{
				Dir: "sections/02-b", MetaPath: "sections/02-b/section.yaml",
				Meta: SectionMeta{ID: "s2", Title: "S2", Summary: "sum", EstimatedMinutes: 5, Lessons: []string{"l3"}},
				Lessons: []LessonDoc{
					{Path: "sections/02-b/lessons/01-l3.mdx", Body: "x", Meta: LessonMeta{ID: "l3", Title: "L3", Summary: "s", EstimatedMinutes: 2, Topics: []string{"t"}}},
				},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	q := func(b *Bundle, id string) *Question {
		for i := range b.Sections[0].Quiz {
			if b.Sections[0].Quiz[i].ID == id {
				return &b.Sections[0].Quiz[i]
			}
		}
		t.Fatalf("no question %s", id)
		return nil
	}
	lesson := func(b *Bundle, id string) *LessonDoc {
		for si := range b.Sections {
			for li := range b.Sections[si].Lessons {
				if b.Sections[si].Lessons[li].Meta.ID == id {
					return &b.Sections[si].Lessons[li]
				}
			}
		}
		t.Fatalf("no lesson %s", id)
		return nil
	}

	tests := []struct {
		name     string
		mutate   func(b *Bundle)
		wantFile string // empty means the bundle must be valid
		wantMsg  string
	}{
		{"valid bundle", func(b *Bundle) {}, "", ""},

		// course.yaml
		{"course title missing", func(b *Bundle) { b.Course.Title = "" }, "course.yaml", "title is required"},
		{"course lists unknown section", func(b *Bundle) { b.Course.Sections = append(b.Course.Sections, "zzz") }, "course.yaml", `"zzz" is listed but no section.yaml defines it`},
		{"course lists section twice", func(b *Bundle) { b.Course.Sections = []string{"s1", "s1", "s2"} }, "course.yaml", "listed more than once"},
		{"orphan section not in course", func(b *Bundle) { b.Course.Sections = []string{"s1"} }, "sections/02-b/section.yaml", "orphan section"},

		// section.yaml
		{"duplicate section id", func(b *Bundle) { b.Sections[1].Meta.ID = "s1" }, "sections/02-b/section.yaml", "duplicate section id"},
		{"section bad id format", func(b *Bundle) { b.Sections[0].Meta.ID = "Bad_Id"; b.Course.Sections[0] = "Bad_Id" }, "sections/01-a/section.yaml", "must be lowercase"},
		{"section minutes zero", func(b *Bundle) { b.Sections[0].Meta.EstimatedMinutes = 0 }, "sections/01-a/section.yaml", "estimated_minutes"},
		{"section lists unknown lesson", func(b *Bundle) { b.Sections[0].Meta.Lessons = append(b.Sections[0].Meta.Lessons, "nope") }, "sections/01-a/section.yaml", `"nope" is listed but no lessons`},
		{"section lists lesson from another section", func(b *Bundle) { b.Sections[0].Meta.Lessons = append(b.Sections[0].Meta.Lessons, "l3") }, "sections/01-a/section.yaml", "outside this section"},
		{"orphan lesson file", func(b *Bundle) { b.Sections[0].Meta.Lessons = []string{"l1"} }, "sections/01-a/lessons/02-l2.mdx", "orphan lesson"},
		{"orphan extra file", func(b *Bundle) { b.Extra = []string{"sections/01-a/notes.txt"} }, "sections/01-a/notes.txt", "orphan"},

		// lessons
		{"lesson title missing", func(b *Bundle) { lesson(b, "l1").Meta.Title = "" }, "sections/01-a/lessons/01-l1.mdx", "title is required"},
		{"lesson summary missing", func(b *Bundle) { lesson(b, "l1").Meta.Summary = "" }, "sections/01-a/lessons/01-l1.mdx", "summary is required"},
		{"lesson minutes missing", func(b *Bundle) { lesson(b, "l1").Meta.EstimatedMinutes = 0 }, "sections/01-a/lessons/01-l1.mdx", "estimated_minutes"},
		{"lesson topics missing", func(b *Bundle) { lesson(b, "l1").Meta.Topics = nil }, "sections/01-a/lessons/01-l1.mdx", "topics must list"},
		{"lesson body empty", func(b *Bundle) { lesson(b, "l1").Body = " \n" }, "sections/01-a/lessons/01-l1.mdx", "body is empty"},
		{"duplicate lesson id", func(b *Bundle) { lesson(b, "l3").Meta.ID = "l1"; b.Sections[1].Meta.Lessons = []string{"l1"} }, "sections/02-b/lessons/01-l3.mdx", "duplicate lesson id"},
		{"unknown prerequisite", func(b *Bundle) { lesson(b, "l2").Meta.Prerequisites = []string{"ghost"} }, "sections/01-a/lessons/02-l2.mdx", `prerequisite "ghost" does not exist`},
		{"self prerequisite", func(b *Bundle) { lesson(b, "l1").Meta.Prerequisites = []string{"l1"} }, "sections/01-a/lessons/01-l1.mdx", "itself"},
		{"prerequisite cycle", func(b *Bundle) { lesson(b, "l1").Meta.Prerequisites = []string{"l2"} }, "sections/01-a/lessons/", "prerequisite cycle"},
		{"longer prerequisite cycle across sections", func(b *Bundle) {
			lesson(b, "l1").Meta.Prerequisites = []string{"l3"}
			lesson(b, "l3").Meta.Prerequisites = []string{"l2"}
		}, "lessons/", "prerequisite cycle"},
		{"broken lesson suppresses unknown-reference noise", func(b *Bundle) {
			b.Sections[0].Lessons = append(b.Sections[0].Lessons, LessonDoc{Path: "sections/01-a/lessons/03.mdx", Broken: true})
			lesson(b, "l2").Meta.Prerequisites = []string{"maybe-the-broken-one"}
		}, "", ""},

		// quiz: common
		{"quiz empty", func(b *Bundle) { b.Sections[0].Quiz = nil }, "sections/01-a/quiz.yaml", "no questions"},
		{"duplicate question id in quiz", func(b *Bundle) { q(b, "q-tf").ID = "q-single" }, "sections/01-a/quiz.yaml", "duplicate question id"},
		{"duplicate question id across sections", func(b *Bundle) {
			b.Sections[1].HasQuiz = true
			b.Sections[1].QuizPath = "sections/02-b/quiz.yaml"
			b.Sections[1].Quiz = []Question{{ID: "q-single", Topic: "t", Difficulty: 1, Type: TrueFalse, Prompt: "p", Explanation: "e", Correct: StringList{"true"}}}
		}, "sections/02-b/quiz.yaml", "duplicate question id"},
		{"bad lesson_ref", func(b *Bundle) { q(b, "q-single").LessonRef = "nope" }, "sections/01-a/quiz.yaml", `lesson_ref "nope"`},
		{"difficulty out of range", func(b *Bundle) { q(b, "q-single").Difficulty = 4 }, "sections/01-a/quiz.yaml", "difficulty must be 1, 2 or 3"},
		{"topic missing", func(b *Bundle) { q(b, "q-single").Topic = "" }, "sections/01-a/quiz.yaml", "topic is required"},
		{"explanation missing", func(b *Bundle) { q(b, "q-single").Explanation = "" }, "sections/01-a/quiz.yaml", "explanation is required"},
		{"unknown type", func(b *Bundle) { q(b, "q-single").Type = "essay" }, "sections/01-a/quiz.yaml", `unknown type "essay"`},
		{"duplicate option id", func(b *Bundle) { q(b, "q-single").Options[1].ID = "a" }, "sections/01-a/quiz.yaml", `duplicate option id "a"`},
		{"option text missing", func(b *Bundle) { q(b, "q-single").Options[1].Text = "" }, "sections/01-a/quiz.yaml", "has no text"},

		// quiz: single_choice
		{"single_choice two correct", func(b *Bundle) { q(b, "q-single").Correct = StringList{"a", "b"} }, "sections/01-a/quiz.yaml", "exactly one correct option"},
		{"single_choice no correct", func(b *Bundle) { q(b, "q-single").Correct = nil }, "sections/01-a/quiz.yaml", "exactly one correct option"},
		{"single_choice correct not an option", func(b *Bundle) { q(b, "q-single").Correct = StringList{"z"} }, "sections/01-a/quiz.yaml", `correct id "z" is not one of the option ids`},
		{"single_choice one option", func(b *Bundle) { q(b, "q-single").Options = q(b, "q-single").Options[:1] }, "sections/01-a/quiz.yaml", "at least two options"},

		// quiz: multi_choice
		{"multi_choice one correct", func(b *Bundle) { q(b, "q-multi").Correct = StringList{"a"} }, "sections/01-a/quiz.yaml", "at least two correct"},
		{"multi_choice all correct", func(b *Bundle) { q(b, "q-multi").Correct = StringList{"a", "b", "c"} }, "sections/01-a/quiz.yaml", "every option correct"},
		{"multi_choice repeated correct", func(b *Bundle) { q(b, "q-multi").Correct = StringList{"a", "a"} }, "sections/01-a/quiz.yaml", "more than once"},

		// quiz: true_false
		{"true_false bad correct", func(b *Bundle) { q(b, "q-tf").Correct = StringList{"maybe"} }, "sections/01-a/quiz.yaml", "correct: true or correct: false"},
		{"true_false custom options", func(b *Bundle) { q(b, "q-tf").Options = []Option{{ID: "yes", Text: "Y"}, {ID: "no", Text: "N"}} }, "sections/01-a/quiz.yaml", "exactly ids"},
		{"true_false explicit standard options ok", func(b *Bundle) {
			q(b, "q-tf").Options = []Option{{ID: "true", Text: "True"}, {ID: "false", Text: "False"}}
		}, "", ""},

		// quiz: ordering
		{"ordering missing an item", func(b *Bundle) { q(b, "q-order").Correct = StringList{"c", "a"} }, "sections/01-a/quiz.yaml", "every option exactly once"},
		{"ordering repeated item", func(b *Bundle) { q(b, "q-order").Correct = StringList{"c", "a", "a"} }, "sections/01-a/quiz.yaml", "more than once"},
		{"ordering unknown item", func(b *Bundle) { q(b, "q-order").Correct = StringList{"c", "a", "zzz"} }, "sections/01-a/quiz.yaml", `"zzz" is not one of the option ids`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := validBundle()
			tt.mutate(b)
			errs := Validate(b)

			if tt.wantFile == "" {
				if len(errs) != 0 {
					t.Fatalf("expected valid, got:\n%v", errs)
				}
				return
			}
			for _, e := range errs {
				if strings.Contains(e.File, tt.wantFile) && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("no error for %q containing %q; got:\n%v", tt.wantFile, tt.wantMsg, errs)
		})
	}
}

func TestValidateCollectsAllErrors(t *testing.T) {
	b := validBundle()
	b.Course.Title = ""
	b.Sections[0].Quiz[0].LessonRef = "nope"
	b.Sections[0].Quiz[0].Correct = StringList{"a", "b"}
	b.Sections[1].Meta.EstimatedMinutes = 0
	if got := len(Validate(b)); got != 4 {
		t.Fatalf("want 4 errors reported together, got %d: %v", got, Validate(b))
	}
}
