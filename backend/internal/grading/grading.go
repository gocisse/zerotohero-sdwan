// Package grading grades quiz and review answers against validated content.
// It is pure: no database, no HTTP, no clock.
package grading

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"velocloud-academy/backend/internal/content"
)

// PassThresholdPct is the informational "passed" threshold for a section quiz.
const PassThresholdPct = 70.0

// InputError marks a malformed submission (wrong answer shape, unknown or duplicate
// question id). Callers should answer 400. A merely *wrong* answer is not an InputError.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

func inputErr(format string, args ...any) error {
	return &InputError{Message: fmt.Sprintf(format, args...)}
}

// Submission is one answer as sent by the client. Answer is a JSON string (single_choice,
// true_false), or an array of option ids (multi_choice, ordering). Absent or null = unanswered.
type Submission struct {
	QuestionID string          `json:"question_id"`
	Answer     json.RawMessage `json:"answer"`
}

// Graded is the outcome for one question.
type Graded struct {
	Question content.Question
	Answered bool
	Given    any     // string, []string or nil; what the client picked, normalized
	Correct  bool    // exact match only
	Partial  float64 // 0..1; equals 1 or 0 except for multi_choice partial credit
}

// Grade grades one answer. It returns an *InputError if the answer has the wrong JSON shape
// for the question type. Option ids that do not exist are simply wrong.
func Grade(q content.Question, raw json.RawMessage) (Graded, error) {
	g := Graded{Question: q}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return g, nil
	}

	switch q.Type {
	case content.SingleChoice, content.TrueFalse:
		id, err := decodeSingle(q, raw)
		if err != nil {
			return g, err
		}
		if id == "" {
			return g, nil
		}
		g.Answered, g.Given = true, id
		g.Correct = len(q.Correct) == 1 && id == q.Correct[0]

	case content.MultiChoice:
		var picks []string
		if err := json.Unmarshal(raw, &picks); err != nil {
			return g, inputErr("answer for question %q must be an array of option ids", q.ID)
		}
		picks = dedupe(picks)
		if len(picks) == 0 {
			return g, nil
		}
		g.Answered, g.Given = true, picks
		good, bad := 0, 0
		for _, p := range picks {
			if slices.Contains(q.Correct, p) {
				good++
			} else {
				bad++
			}
		}
		g.Correct = bad == 0 && good == len(q.Correct)
		if total := len(q.Correct); total > 0 {
			g.Partial = max(0, float64(good-bad)/float64(total))
		}
		if g.Correct {
			g.Partial = 1
		}
		return g, nil

	case content.Ordering:
		var order []string
		if err := json.Unmarshal(raw, &order); err != nil {
			return g, inputErr("answer for question %q must be an array of option ids in order", q.ID)
		}
		if len(order) == 0 {
			return g, nil
		}
		g.Answered, g.Given = true, order
		g.Correct = slices.Equal(order, []string(q.Correct))

	default:
		return g, inputErr("question %q has unsupported type %q", q.ID, q.Type)
	}

	if g.Correct {
		g.Partial = 1
	}
	return g, nil
}

// decodeSingle accepts an option id string; for true_false also a JSON boolean.
func decodeSingle(q content.Question, raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var b bool
	if q.Type == content.TrueFalse && json.Unmarshal(raw, &b) == nil {
		return fmt.Sprint(b), nil
	}
	return "", inputErr("answer for question %q must be an option id string", q.ID)
}

func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// GradeQuiz grades a full section quiz. Every question is graded in quiz order; questions with
// no submission count as unanswered (wrong). Unknown or duplicate question ids are InputErrors.
func GradeQuiz(questions []content.Question, subs []Submission) ([]Graded, error) {
	byID := make(map[string]json.RawMessage, len(subs))
	known := make(map[string]bool, len(questions))
	for _, q := range questions {
		known[q.ID] = true
	}
	for _, s := range subs {
		if !known[s.QuestionID] {
			return nil, inputErr("unknown question id %q", s.QuestionID)
		}
		if _, dup := byID[s.QuestionID]; dup {
			return nil, inputErr("duplicate answer for question %q", s.QuestionID)
		}
		byID[s.QuestionID] = s.Answer
	}
	out := make([]Graded, 0, len(questions))
	for _, q := range questions {
		g, err := Grade(q, byID[q.ID])
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// GradeSelected grades only the submitted questions (review sessions), in submission order.
func GradeSelected(lookup func(id string) (content.Question, bool), subs []Submission) ([]Graded, error) {
	if len(subs) == 0 {
		return nil, inputErr("no answers submitted")
	}
	seen := map[string]bool{}
	out := make([]Graded, 0, len(subs))
	for _, s := range subs {
		q, ok := lookup(s.QuestionID)
		if !ok {
			return nil, inputErr("unknown question id %q", s.QuestionID)
		}
		if seen[s.QuestionID] {
			return nil, inputErr("duplicate answer for question %q", s.QuestionID)
		}
		seen[s.QuestionID] = true
		g, err := Grade(q, s.Answer)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// Score counts exactly-correct questions.
func Score(graded []Graded) int {
	n := 0
	for _, g := range graded {
		if g.Correct {
			n++
		}
	}
	return n
}

// Percentage returns score/total as a percentage rounded to one decimal.
func Percentage(score, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(int(float64(score)/float64(total)*1000+0.5)) / 10
}

// Feedback is what is revealed after grading.
type Feedback struct {
	CorrectAnswer      any // string (single_choice, true_false) or []string (multi_choice, ordering)
	Explanation        string
	OptionExplanations map[string]string // explanations for wrongly picked options only
	LessonRef          string
	Topic              string
}

// FeedbackFor builds the post-grading feedback for one question.
func FeedbackFor(g Graded) Feedback {
	q := g.Question
	fb := Feedback{
		Explanation:        q.Explanation,
		OptionExplanations: map[string]string{},
		LessonRef:          q.LessonRef,
		Topic:              q.Topic,
	}
	switch q.Type {
	case content.SingleChoice, content.TrueFalse:
		if len(q.Correct) > 0 {
			fb.CorrectAnswer = q.Correct[0]
		}
	default:
		fb.CorrectAnswer = []string(q.Correct)
	}

	if q.Type == content.SingleChoice || q.Type == content.MultiChoice {
		var picked []string
		switch v := g.Given.(type) {
		case string:
			picked = []string{v}
		case []string:
			picked = v
		}
		for _, id := range picked {
			if slices.Contains(q.Correct, id) {
				continue
			}
			for _, o := range q.Options {
				if o.ID == id && o.Explanation != "" {
					fb.OptionExplanations[id] = o.Explanation
				}
			}
		}
	}
	return fb
}
