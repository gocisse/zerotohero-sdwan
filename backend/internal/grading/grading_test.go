package grading

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"velocloud-academy/backend/internal/content"
)

func opts(ids ...string) []content.Option {
	out := make([]content.Option, len(ids))
	for i, id := range ids {
		out[i] = content.Option{ID: id, Text: id}
	}
	return out
}

var (
	single = content.Question{ID: "q-single", Topic: "t1", Type: content.SingleChoice, Options: opts("a", "b", "c"), Correct: content.StringList{"a"}}
	multi  = content.Question{ID: "q-multi", Topic: "t2", Type: content.MultiChoice, Options: opts("a", "b", "c", "d"), Correct: content.StringList{"a", "b", "c"}}
	tf     = content.Question{ID: "q-tf", Topic: "t3", Type: content.TrueFalse, Options: opts("true", "false"), Correct: content.StringList{"true"}}
	order  = content.Question{ID: "q-order", Topic: "t1", Type: content.Ordering, Options: opts("a", "b", "c"), Correct: content.StringList{"b", "c", "a"}}
)

func TestGrade(t *testing.T) {
	tests := []struct {
		name        string
		q           content.Question
		answer      string // raw JSON; "" means absent
		wantErr     bool
		wantAnswer  bool
		wantCorrect bool
		wantPartial float64
	}{
		// single_choice
		{"single all-correct", single, `"a"`, false, true, true, 1},
		{"single wrong", single, `"b"`, false, true, false, 0},
		{"single unknown option is just wrong", single, `"zzz"`, false, true, false, 0},
		{"single unanswered (absent)", single, ``, false, false, false, 0},
		{"single unanswered (null)", single, `null`, false, false, false, 0},
		{"single unanswered (empty string)", single, `""`, false, false, false, 0},
		{"single wrong shape: array", single, `["a"]`, true, false, false, 0},
		{"single wrong shape: number", single, `3`, true, false, false, 0},
		{"single wrong shape: bool", single, `true`, true, false, false, 0},

		// true_false
		{"tf correct string", tf, `"true"`, false, true, true, 1},
		{"tf correct bool", tf, `true`, false, true, true, 1},
		{"tf wrong string", tf, `"false"`, false, true, false, 0},
		{"tf wrong bool", tf, `false`, false, true, false, 0},
		{"tf garbage string is wrong", tf, `"maybe"`, false, true, false, 0},
		{"tf unanswered", tf, `null`, false, false, false, 0},
		{"tf wrong shape: array", tf, `["true"]`, true, false, false, 0},

		// multi_choice
		{"multi all-correct", multi, `["c","a","b"]`, false, true, true, 1},
		{"multi all-wrong", multi, `["d"]`, false, true, false, 0},
		{"multi partial: 2 of 3 correct", multi, `["a","b"]`, false, true, false, 2.0 / 3},
		{"multi partial: 2 correct 1 wrong", multi, `["a","b","d"]`, false, true, false, 1.0 / 3},
		{"multi wrong outweighs right: floor at 0", multi, `["a","d"]`, false, true, false, 0},
		{"multi all correct plus one wrong is not exact", multi, `["a","b","c","d"]`, false, true, false, 2.0 / 3},
		{"multi duplicates in picks count once", multi, `["a","a","a"]`, false, true, false, 1.0 / 3},
		{"multi unknown option is a wrong pick", multi, `["a","b","c","zzz"]`, false, true, false, 2.0 / 3},
		{"multi unanswered (absent)", multi, ``, false, false, false, 0},
		{"multi unanswered (empty array)", multi, `[]`, false, false, false, 0},
		{"multi wrong shape: string", multi, `"a"`, true, false, false, 0},

		// ordering
		{"ordering all-correct", order, `["b","c","a"]`, false, true, true, 1},
		{"ordering wrong order", order, `["a","b","c"]`, false, true, false, 0},
		{"ordering incomplete", order, `["b","c"]`, false, true, false, 0},
		{"ordering repeats", order, `["b","b","b"]`, false, true, false, 0},
		{"ordering unanswered", order, `[]`, false, false, false, 0},
		{"ordering wrong shape: string", order, `"b"`, true, false, false, 0},
		{"ordering wrong shape: array of numbers", order, `[1,2,3]`, true, false, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := Grade(tt.q, json.RawMessage(tt.answer))
			if tt.wantErr {
				var ie *InputError
				if !errors.As(err, &ie) {
					t.Fatalf("want InputError, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if g.Answered != tt.wantAnswer || g.Correct != tt.wantCorrect {
				t.Errorf("answered=%v correct=%v, want %v/%v", g.Answered, g.Correct, tt.wantAnswer, tt.wantCorrect)
			}
			if d := g.Partial - tt.wantPartial; d > 1e-9 || d < -1e-9 {
				t.Errorf("partial = %v, want %v", g.Partial, tt.wantPartial)
			}
		})
	}
}

func TestGradeQuiz(t *testing.T) {
	qs := []content.Question{single, multi, tf, order}
	sub := func(id, ans string) Submission { return Submission{QuestionID: id, Answer: json.RawMessage(ans)} }

	t.Run("all correct", func(t *testing.T) {
		got, err := GradeQuiz(qs, []Submission{sub("q-single", `"a"`), sub("q-multi", `["a","b","c"]`), sub("q-tf", `"true"`), sub("q-order", `["b","c","a"]`)})
		if err != nil || Score(got) != 4 {
			t.Fatalf("score=%d err=%v", Score(got), err)
		}
	})
	t.Run("all wrong", func(t *testing.T) {
		got, err := GradeQuiz(qs, []Submission{sub("q-single", `"c"`), sub("q-multi", `["d"]`), sub("q-tf", `"false"`), sub("q-order", `["a","b","c"]`)})
		if err != nil || Score(got) != 0 {
			t.Fatalf("score=%d err=%v", Score(got), err)
		}
	})
	t.Run("missing answers count as wrong and every question is graded in quiz order", func(t *testing.T) {
		got, err := GradeQuiz(qs, []Submission{sub("q-tf", `"true"`)})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 4 || Score(got) != 1 || got[0].Question.ID != "q-single" || got[0].Answered {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("empty submission is all unanswered", func(t *testing.T) {
		got, err := GradeQuiz(qs, nil)
		if err != nil || Score(got) != 0 || len(got) != 4 {
			t.Fatalf("score=%d err=%v", Score(got), err)
		}
	})
	for _, tt := range []struct {
		name string
		subs []Submission
	}{
		{"unknown id", []Submission{sub("nope", `"a"`)}},
		{"duplicate ids", []Submission{sub("q-single", `"a"`), sub("q-single", `"b"`)}},
		{"wrong shape", []Submission{sub("q-single", `["a"]`)}},
	} {
		t.Run(tt.name+" is an input error", func(t *testing.T) {
			_, err := GradeQuiz(qs, tt.subs)
			var ie *InputError
			if !errors.As(err, &ie) {
				t.Fatalf("want InputError, got %v", err)
			}
		})
	}
}

func TestGradeSelected(t *testing.T) {
	lookup := func(id string) (content.Question, bool) {
		for _, q := range []content.Question{single, tf} {
			if q.ID == id {
				return q, true
			}
		}
		return content.Question{}, false
	}
	sub := func(id, ans string) Submission { return Submission{QuestionID: id, Answer: json.RawMessage(ans)} }

	got, err := GradeSelected(lookup, []Submission{sub("q-tf", `"true"`), sub("q-single", `"b"`)})
	if err != nil || len(got) != 2 || got[0].Question.ID != "q-tf" || !got[0].Correct || got[1].Correct {
		t.Fatalf("got %+v err=%v", got, err)
	}
	for name, subs := range map[string][]Submission{
		"empty":     nil,
		"unknown":   {sub("zzz", `"a"`)},
		"duplicate": {sub("q-tf", `"true"`), sub("q-tf", `"true"`)},
	} {
		if _, err := GradeSelected(lookup, subs); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestFeedback(t *testing.T) {
	q := single
	q.Explanation = "because"
	q.LessonRef = "lesson-1"
	q.Options = []content.Option{{ID: "a", Text: "a"}, {ID: "b", Text: "b", Explanation: "b is wrong"}, {ID: "c", Text: "c"}}

	g, _ := Grade(q, json.RawMessage(`"b"`))
	fb := FeedbackFor(g)
	if fb.CorrectAnswer != "a" || fb.Explanation != "because" || fb.LessonRef != "lesson-1" || fb.Topic != "t1" {
		t.Errorf("feedback = %+v", fb)
	}
	if !reflect.DeepEqual(fb.OptionExplanations, map[string]string{"b": "b is wrong"}) {
		t.Errorf("option explanations = %v", fb.OptionExplanations)
	}

	// A correct pick reveals no per-option explanation, and unpicked wrong options stay hidden.
	q.Options[0].Explanation = "a is the answer"
	g, _ = Grade(q, json.RawMessage(`"a"`))
	if got := FeedbackFor(g).OptionExplanations; len(got) != 0 {
		t.Errorf("option explanations for correct pick = %v", got)
	}

	if mf := FeedbackFor(Graded{Question: multi}); !reflect.DeepEqual(mf.CorrectAnswer, []string{"a", "b", "c"}) {
		t.Errorf("multi correct answer = %v", mf.CorrectAnswer)
	}
}

func TestPercentage(t *testing.T) {
	for _, tt := range []struct {
		score, total int
		want         float64
	}{{0, 4, 0}, {1, 3, 33.3}, {2, 3, 66.7}, {3, 4, 75}, {4, 4, 100}, {0, 0, 0}} {
		if got := Percentage(tt.score, tt.total); got != tt.want {
			t.Errorf("Percentage(%d,%d) = %v, want %v", tt.score, tt.total, got, tt.want)
		}
	}
}
