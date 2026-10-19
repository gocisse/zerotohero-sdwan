package http

import (
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
)

// ContentSource provides the current (auto-reloaded) course.
type ContentSource interface {
	Current() (*content.Course, error)
}

type lessonSummaryDTO struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Summary          string `json:"summary"`
	EstimatedMinutes int    `json:"estimated_minutes"`
}

type sectionDTO struct {
	ID               string             `json:"id"`
	Title            string             `json:"title"`
	Summary          string             `json:"summary"`
	EstimatedMinutes int                `json:"estimated_minutes"`
	Placeholder      bool               `json:"placeholder"`
	QuestionCount    int                `json:"question_count"`
	Lessons          []lessonSummaryDTO `json:"lessons"`
}

type courseDTO struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Sections    []sectionDTO `json:"sections"`
}

type lessonDTO struct {
	ID               string   `json:"id"`
	SectionID        string   `json:"section_id"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Topics           []string `json:"topics"`
	Prerequisites    []string `json:"prerequisites"`
	Source           string   `json:"source"`
}

type optionDTO struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// questionDTO deliberately omits correct answers, explanations and lesson_ref;
// those are revealed by server-side grading.
type questionDTO struct {
	ID         string      `json:"id"`
	Topic      string      `json:"topic"`
	Difficulty int         `json:"difficulty"`
	Type       string      `json:"type"`
	Prompt     string      `json:"prompt"`
	Options    []optionDTO `json:"options"`
}

type quizDTO struct {
	SectionID string        `json:"section_id"`
	Questions []questionDTO `json:"questions"`
}

type apiError struct {
	Error   string          `json:"error"`
	Message string          `json:"message,omitempty"`
	Details []content.Error `json:"details,omitempty"`
}

// withCourse resolves the current course, or answers 503 listing every content error.
func withCourse(src ContentSource, log *slog.Logger, h func(http.ResponseWriter, *http.Request, *content.Course)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := src.Current()
		if err != nil {
			var list content.ErrorList
			if errors.As(err, &list) {
				writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "content_invalid", Message: "course content failed validation", Details: list})
				return
			}
			log.Error("load content", "err", err)
			writeJSON(w, http.StatusInternalServerError, apiError{Error: "content_unavailable", Message: err.Error()})
			return
		}
		h(w, r, c)
	}
}

func notFound(w http.ResponseWriter, what, id string) {
	writeJSON(w, http.StatusNotFound, apiError{Error: "not_found", Message: what + " " + id + " not found"})
}

func CourseHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, _ *http.Request, c *content.Course) {
		out := courseDTO{Title: c.Meta.Title, Description: c.Meta.Description, Sections: []sectionDTO{}}
		for _, s := range c.Sections {
			sd := sectionDTO{
				ID: s.Meta.ID, Title: s.Meta.Title, Summary: s.Meta.Summary,
				EstimatedMinutes: s.Meta.EstimatedMinutes, Placeholder: s.Meta.Placeholder,
				QuestionCount: len(s.Quiz), Lessons: []lessonSummaryDTO{},
			}
			for _, l := range s.Lessons {
				sd.Lessons = append(sd.Lessons, lessonSummaryDTO{
					ID: l.Meta.ID, Title: l.Meta.Title, Summary: l.Meta.Summary, EstimatedMinutes: l.Meta.EstimatedMinutes,
				})
			}
			out.Sections = append(out.Sections, sd)
		}
		writeJSON(w, http.StatusOK, out)
	})
}

func LessonHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		l, ok := c.Lesson(id)
		if !ok {
			notFound(w, "lesson", id)
			return
		}
		writeJSON(w, http.StatusOK, lessonDTO{
			ID: l.Meta.ID, SectionID: l.SectionID, Title: l.Meta.Title, Summary: l.Meta.Summary,
			EstimatedMinutes: l.Meta.EstimatedMinutes, Topics: nonNil(l.Meta.Topics),
			Prerequisites: nonNil(l.Meta.Prerequisites), Source: l.Body,
		})
	})
}

func QuizHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		s, ok := c.Section(id)
		if !ok {
			notFound(w, "section", id)
			return
		}
		out := quizDTO{SectionID: id, Questions: []questionDTO{}}
		for _, q := range s.Quiz {
			out.Questions = append(out.Questions, toQuestionDTO(q))
		}
		writeJSON(w, http.StatusOK, out)
	})
}

func DiagramHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		d, ok := c.Diagram(id)
		if !ok {
			notFound(w, "diagram", id)
			return
		}
		writeJSON(w, http.StatusOK, d)
	})
}

func GlossaryHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, _ *http.Request, c *content.Course) {
		terms := c.Glossary
		if terms == nil {
			terms = []content.GlossaryTerm{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"terms": terms})
	})
}

// toQuestionDTO strips answers from a question. The authored option order of an ordering
// question is its answer, so those options are shuffled on every call.
func toQuestionDTO(q content.Question) questionDTO {
	opts := make([]optionDTO, len(q.Options))
	for i, o := range q.Options {
		opts[i] = optionDTO{ID: o.ID, Text: o.Text}
	}
	if q.Type == content.Ordering {
		rand.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	}
	return questionDTO{ID: q.ID, Topic: q.Topic, Difficulty: q.Difficulty, Type: string(q.Type), Prompt: q.Prompt, Options: opts}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
