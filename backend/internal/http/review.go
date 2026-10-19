package http

import (
	"net/http"
	"time"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/grading"
	"velocloud-academy/backend/internal/review"
)

// reviewSessionSize caps how many questions one review session serves.
const reviewSessionSize = 15

type reviewQueueDTO struct {
	DueCount      int           `json:"due_count"`
	UpcomingCount int           `json:"upcoming_count"`
	NextDueAt     *time.Time    `json:"next_due_at"`
	SessionSize   int           `json:"session_size"`
	Questions     []questionDTO `json:"questions"` // without answers or explanations
}

// queueFor selects the questions due at now. Question states whose ids no longer exist in
// /content are ignored (never deleted).
func (h *progressHandlers) queueFor(r *http.Request, c *content.Course, now time.Time) (review.Queue, map[string]content.Question, error) {
	states, err := h.d.Progress.QuestionStates(r.Context())
	if err != nil {
		return review.Queue{}, nil, err
	}
	idx := questionIndex(c)
	var entries []review.Entry
	for _, st := range states {
		if _, ok := idx[st.QuestionID]; ok {
			entries = append(entries, review.Entry{ID: st.QuestionID, NextDueAt: st.NextDueAt})
		}
	}
	return review.SelectDue(entries, now, reviewSessionSize), idx, nil
}

func (h *progressHandlers) reviewQueue(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		q, idx, err := h.queueFor(r, c, h.d.Now())
		if err != nil {
			h.fail(w, "load review queue", err)
			return
		}
		out := reviewQueueDTO{
			DueCount: q.DueCount, UpcomingCount: q.Upcoming, NextDueAt: q.NextDue,
			SessionSize: reviewSessionSize, Questions: []questionDTO{},
		}
		for _, id := range q.IDs {
			out.Questions = append(out.Questions, toQuestionDTO(idx[id]))
		}
		writeJSON(w, http.StatusOK, out)
	})(w, r)
}

// reviewAnswers grades review answers with the same code as quizzes. It updates Leitner state but
// never creates quiz attempts.
func (h *progressHandlers) reviewAnswers(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		var req submitRequest
		if !decodeBody(w, r, &req) {
			return
		}
		idx := questionIndex(c)
		graded, err := grading.GradeSelected(func(id string) (content.Question, bool) { q, ok := idx[id]; return q, ok }, req.Answers)
		if err != nil {
			h.answersFailure(w, err)
			return
		}
		now := h.d.Now()
		if err := h.d.Progress.RecordReview(r.Context(), toRecords(graded), now); err != nil {
			h.fail(w, "record review", err)
			return
		}
		q, _, err := h.queueFor(r, c, now)
		if err != nil {
			h.fail(w, "load review queue", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results": toResults(graded), "due_count": q.DueCount, "upcoming_count": q.Upcoming, "next_due_at": q.NextDue,
		})
	})(w, r)
}
