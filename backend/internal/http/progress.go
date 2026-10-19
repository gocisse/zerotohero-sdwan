package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/grading"
	"velocloud-academy/backend/internal/mastery"
)

// ProgressStore is the persistence the progress, quiz and review handlers use.
type ProgressStore interface {
	RecordAttempt(ctx context.Context, in domain.AttemptInput) (domain.Attempt, error)
	RecordReview(ctx context.Context, answers []domain.AnswerRecord, now time.Time) error
	ListAttempts(ctx context.Context, sectionID string) ([]domain.Attempt, error)
	AllAttempts(ctx context.Context) ([]domain.Attempt, error)
	SkipQuiz(ctx context.Context, sectionID string, now time.Time) (time.Time, error)
	SetLessonProgress(ctx context.Context, lessonID, sectionID, status string, now time.Time) (domain.LessonProgress, error)
	LessonProgressAll(ctx context.Context) (map[string]domain.LessonProgress, error)
	SectionStates(ctx context.Context) (map[string]domain.SectionState, error)
	QuestionStates(ctx context.Context) ([]domain.QuestionState, error)
	AnswerHistory(ctx context.Context) ([]domain.AnswerEvent, error)
}

const maxBody = 1 << 20

type progressHandlers struct{ d Deps }

// withCourse resolves the current course (503 with every content error if invalid) for a handler.
func (h *progressHandlers) withCourse(fn func(w http.ResponseWriter, r *http.Request, c *content.Course)) http.HandlerFunc {
	return withCourse(h.d.Content, h.d.Log, fn)
}

func (h *progressHandlers) fail(w http.ResponseWriter, what string, err error) {
	h.d.Log.Error(what, "err", err)
	writeJSON(w, http.StatusInternalServerError, apiError{Error: "internal", Message: what + " failed"})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, apiError{Error: "bad_request", Message: msg})
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(v); err != nil {
		badRequest(w, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// answersFailure maps a grading error to a response: InputError -> 400, anything else -> 500.
func (h *progressHandlers) answersFailure(w http.ResponseWriter, err error) {
	var ie *grading.InputError
	if errors.As(err, &ie) {
		badRequest(w, ie.Message)
		return
	}
	h.fail(w, "grade answers", err)
}

// ---- grading response shape ----

type resultDTO struct {
	QuestionID         string            `json:"question_id"`
	Topic              string            `json:"topic"`
	Answered           bool              `json:"answered"`
	Correct            bool              `json:"correct"`
	PartialScore       float64           `json:"partial_score"`
	Given              any               `json:"given"`
	CorrectAnswer      any               `json:"correct_answer"`
	Explanation        string            `json:"explanation"`
	OptionExplanations map[string]string `json:"option_explanations"`
	LessonRef          string            `json:"lesson_ref,omitempty"`
}

func toResults(graded []grading.Graded) []resultDTO {
	out := make([]resultDTO, len(graded))
	for i, g := range graded {
		fb := grading.FeedbackFor(g)
		out[i] = resultDTO{
			QuestionID: g.Question.ID, Topic: fb.Topic, Answered: g.Answered, Correct: g.Correct,
			PartialScore: round3(g.Partial), Given: g.Given, CorrectAnswer: fb.CorrectAnswer,
			Explanation: fb.Explanation, OptionExplanations: fb.OptionExplanations, LessonRef: fb.LessonRef,
		}
	}
	return out
}

func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }

func toRecords(graded []grading.Graded) []domain.AnswerRecord {
	out := make([]domain.AnswerRecord, len(graded))
	for i, g := range graded {
		out[i] = domain.AnswerRecord{
			QuestionID: g.Question.ID, Topic: g.Question.Topic, Difficulty: g.Question.Difficulty,
			Correct: g.Correct, Partial: round3(g.Partial), Given: g.Given,
		}
	}
	return out
}

type submitRequest struct {
	Answers []grading.Submission `json:"answers"`
}

// ---- quiz attempts ----

type attemptDTO struct {
	ID        int64     `json:"id"`
	Score     int       `json:"score"`
	Total     int       `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

type attemptResponse struct {
	AttemptID  int64       `json:"attempt_id"`
	SectionID  string      `json:"section_id"`
	Score      int         `json:"score"`
	Total      int         `json:"total"`
	Percentage float64     `json:"percentage"`
	Passed     bool        `json:"passed"`
	CreatedAt  time.Time   `json:"created_at"`
	Results    []resultDTO `json:"results"`
}

func (h *progressHandlers) submitAttempt(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		s, ok := c.Section(id)
		if !ok {
			notFound(w, "section", id)
			return
		}
		if len(s.Quiz) == 0 {
			writeJSON(w, http.StatusNotFound, apiError{Error: "not_found", Message: "section " + id + " has no quiz"})
			return
		}
		var req submitRequest
		if !decodeBody(w, r, &req) {
			return
		}
		graded, err := grading.GradeQuiz(s.Quiz, req.Answers)
		if err != nil {
			h.answersFailure(w, err)
			return
		}

		score, total := grading.Score(graded), len(graded)
		pct := grading.Percentage(score, total)
		att, err := h.d.Progress.RecordAttempt(r.Context(), domain.AttemptInput{
			SectionID: id, Score: score, Total: total, Answers: toRecords(graded), Now: h.d.Now(),
		})
		if err != nil {
			h.fail(w, "record attempt", err)
			return
		}
		writeJSON(w, http.StatusOK, attemptResponse{
			AttemptID: att.ID, SectionID: id, Score: score, Total: total, Percentage: pct,
			// INFORMATIONAL ONLY. `passed` must never be used to lock, hide, gate or otherwise
			// restrict anything (lessons, sections, retries, navigation). Soft gating is a core rule.
			Passed:    pct >= grading.PassThresholdPct,
			CreatedAt: att.CreatedAt, Results: toResults(graded),
		})
	})(w, r)
}

func (h *progressHandlers) listAttempts(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		if _, ok := c.Section(id); !ok {
			notFound(w, "section", id)
			return
		}
		attempts, err := h.d.Progress.ListAttempts(r.Context(), id)
		if err != nil {
			h.fail(w, "list attempts", err)
			return
		}
		out := make([]attemptDTO, len(attempts))
		for i, a := range attempts {
			out[i] = attemptDTO{ID: a.ID, Score: a.Score, Total: a.Total, CreatedAt: a.CreatedAt}
		}
		writeJSON(w, http.StatusOK, map[string]any{"section_id": id, "attempts": out})
	})(w, r)
}

func (h *progressHandlers) skipQuiz(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		if _, ok := c.Section(id); !ok {
			notFound(w, "section", id)
			return
		}
		at, err := h.d.Progress.SkipQuiz(r.Context(), id, h.d.Now())
		if err != nil {
			h.fail(w, "skip quiz", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"section_id": id, "quiz_skipped_at": at})
	})(w, r)
}

// ---- lesson progress ----

func (h *progressHandlers) putLessonProgress(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		l, ok := c.Lesson(id)
		if !ok {
			notFound(w, "lesson", id)
			return
		}
		var req struct {
			Status string `json:"status"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		if req.Status != domain.LessonVisited && req.Status != domain.LessonCompleted {
			badRequest(w, `status must be "visited" or "completed"`)
			return
		}
		lp, err := h.d.Progress.SetLessonProgress(r.Context(), id, l.SectionID, req.Status, h.d.Now())
		if err != nil {
			h.fail(w, "save lesson progress", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lesson_id": id, "status": lp.Status, "completed_at": lp.CompletedAt})
	})(w, r)
}

// ---- progress summary ----

type lessonProgressDTO struct {
	LessonID  string `json:"lesson_id"`
	SectionID string `json:"section_id"`
	Status    string `json:"status"`
}

type sectionProgressDTO struct {
	SectionID        string     `json:"section_id"`
	LessonsCompleted int        `json:"lessons_completed"`
	LessonsTotal     int        `json:"lessons_total"`
	QuizAttempts     int        `json:"quiz_attempts"`
	BestScorePct     *float64   `json:"best_score_pct"`
	LatestScorePct   *float64   `json:"latest_score_pct"`
	QuizSkipped      bool       `json:"quiz_skipped"`
	QuizSkippedAt    *time.Time `json:"quiz_skipped_at"`
	LastVisitedAt    *time.Time `json:"last_visited_at"`
}

type progressDTO struct {
	OverallPercentage float64              `json:"overall_percentage"`
	LessonsCompleted  int                  `json:"lessons_completed"`
	LessonsTotal      int                  `json:"lessons_total"`
	Lessons           []lessonProgressDTO  `json:"lessons"`
	Sections          []sectionProgressDTO `json:"sections"`
}

// progress summarizes the current content only: rows for ids no longer in /content are ignored.
func (h *progressHandlers) progress(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		ctx := r.Context()
		lessons, err := h.d.Progress.LessonProgressAll(ctx)
		if err != nil {
			h.fail(w, "load lesson progress", err)
			return
		}
		states, err := h.d.Progress.SectionStates(ctx)
		if err != nil {
			h.fail(w, "load section state", err)
			return
		}
		attempts, err := h.d.Progress.AllAttempts(ctx) // oldest first
		if err != nil {
			h.fail(w, "load attempts", err)
			return
		}
		bySection := map[string][]domain.Attempt{}
		for _, a := range attempts {
			bySection[a.SectionID] = append(bySection[a.SectionID], a)
		}

		out := progressDTO{Lessons: []lessonProgressDTO{}, Sections: []sectionProgressDTO{}}
		for _, s := range c.Sections {
			sp := sectionProgressDTO{SectionID: s.Meta.ID, LessonsTotal: len(s.Lessons)}
			for _, l := range s.Lessons {
				status := domain.LessonNotStarted
				if lp, ok := lessons[l.Meta.ID]; ok && (lp.Status == domain.LessonVisited || lp.Status == domain.LessonCompleted) {
					status = lp.Status
				}
				if status == domain.LessonCompleted {
					sp.LessonsCompleted++
				}
				out.Lessons = append(out.Lessons, lessonProgressDTO{LessonID: l.Meta.ID, SectionID: s.Meta.ID, Status: status})
			}
			atts := bySection[s.Meta.ID]
			sp.QuizAttempts = len(atts)
			for _, a := range atts {
				if a.Total <= 0 {
					continue
				}
				p := grading.Percentage(a.Score, a.Total)
				if sp.BestScorePct == nil || p > *sp.BestScorePct {
					best := p
					sp.BestScorePct = &best
				}
				latest := p
				sp.LatestScorePct = &latest
			}
			if st, ok := states[s.Meta.ID]; ok {
				sp.QuizSkipped = st.QuizSkippedAt != nil
				sp.QuizSkippedAt, sp.LastVisitedAt = st.QuizSkippedAt, st.LastVisitedAt
			}
			out.LessonsCompleted += sp.LessonsCompleted
			out.LessonsTotal += sp.LessonsTotal
			out.Sections = append(out.Sections, sp)
		}
		if out.LessonsTotal > 0 {
			out.OverallPercentage = grading.Percentage(out.LessonsCompleted, out.LessonsTotal)
		}
		writeJSON(w, http.StatusOK, out)
	})(w, r)
}

// ---- topic mastery ----

type lessonRefDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SectionID string `json:"section_id"`
}

type topicDTO struct {
	Topic       string         `json:"topic"`
	Answers     int            `json:"answers"`
	AccuracyPct float64        `json:"accuracy_pct"`
	Weak        bool           `json:"weak"`
	Lessons     []lessonRefDTO `json:"lessons"`
}

// questionIndex maps question id -> question across all sections.
func questionIndex(c *content.Course) map[string]content.Question {
	idx := map[string]content.Question{}
	for _, s := range c.Sections {
		for _, q := range s.Quiz {
			idx[q.ID] = q
		}
	}
	return idx
}

func (h *progressHandlers) topics(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		history, err := h.d.Progress.AnswerHistory(r.Context())
		if err != nil {
			h.fail(w, "load answer history", err)
			return
		}
		idx := questionIndex(c)
		// Ignore answers to questions that no longer exist; use the current topic of those that do.
		var events []domain.AnswerEvent
		for _, e := range history {
			if q, ok := idx[e.QuestionID]; ok {
				e.Topic = q.Topic
				events = append(events, e)
			}
		}

		refs := map[string]map[string]bool{} // topic -> lesson ids referenced by its questions
		for _, q := range idx {
			if q.LessonRef != "" {
				if refs[q.Topic] == nil {
					refs[q.Topic] = map[string]bool{}
				}
				refs[q.Topic][q.LessonRef] = true
			}
		}

		out := []topicDTO{}
		for _, t := range mastery.Compute(events) {
			td := topicDTO{Topic: t.Topic, Answers: t.Answers, AccuracyPct: round1(t.AccuracyPct), Weak: t.Weak, Lessons: []lessonRefDTO{}}
			for _, s := range c.Sections {
				for _, l := range s.Lessons {
					if refs[t.Topic][l.Meta.ID] || hasTopic(l.Meta.Topics, t.Topic) {
						td.Lessons = append(td.Lessons, lessonRefDTO{ID: l.Meta.ID, Title: l.Meta.Title, SectionID: s.Meta.ID})
					}
				}
			}
			out = append(out, td)
		}
		writeJSON(w, http.StatusOK, map[string]any{"topics": out})
	})(w, r)
}

func hasTopic(topics []string, t string) bool {
	for _, x := range topics {
		if x == t {
			return true
		}
	}
	return false
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
