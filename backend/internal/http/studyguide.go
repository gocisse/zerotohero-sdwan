package http

import (
	"net/http"
	"time"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/grading"
	"velocloud-academy/backend/internal/mastery"
)

// ---- GET /api/study-guide ----

type guideLessonDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SectionID string `json:"section_id"`
	Status    string `json:"status"`
}

type guideQuizTopicDTO struct {
	Topic       string   `json:"topic"`
	Answers     int      `json:"answers"`
	AccuracyPct *float64 `json:"accuracy_pct"` // nil until the topic has been answered
	Weak        bool     `json:"weak"`
	Due         int      `json:"due"`
}

type guideDiagramDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type guideLabDTO struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type guideProgressDTO struct {
	LessonsCompleted int `json:"lessons_completed"`
	LessonsTotal     int `json:"lessons_total"`
	// BestSectionScorePct is the best quiz attempt among the sections the topic's lessons belong to; TopicAccuracyPct is
	// the mastery accuracy over its quiz topic tags. Either is nil when there is nothing to base it on.
	BestSectionScorePct *float64 `json:"best_section_score_pct"`
	TopicAccuracyPct    *float64 `json:"topic_accuracy_pct"`
	QuizAnswers         int      `json:"quiz_answers"`
	LabsCompleted       int      `json:"labs_completed"`
	LabsTotal           int      `json:"labs_total"`
	ReviewDue           int      `json:"review_due"`
}

type guideTopicDTO struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	Kind       string              `json:"kind"`
	Coverage   string              `json:"coverage"`
	Note       string              `json:"note"`
	Lessons    []guideLessonDTO    `json:"lessons"`
	QuizTopics []guideQuizTopicDTO `json:"quiz_topics"`
	Diagrams   []guideDiagramDTO   `json:"diagrams"`
	Labs       []guideLabDTO       `json:"labs"`
	SimScreens []content.SimScreen `json:"sim_screens"`
	Progress   guideProgressDTO    `json:"progress"`
}

type guideSectionDTO struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	AddedByCourse bool            `json:"added_by_course"`
	Topics        []guideTopicDTO `json:"topics"`
}

type guideDTO struct {
	Sections        []guideSectionDTO   `json:"sections"`
	Beyond          content.GuideBeyond `json:"beyond"`
	UnmappedLessons []guideLessonDTO    `json:"unmapped_lessons"`
	Warnings        []content.Error     `json:"warnings"`
	SimScreens      []content.SimScreen `json:"sim_screens"`
}

// studyGuide returns the study guide with, per topic, progress derived from stored data. Rows for ids that no longer
// exist in /content are ignored.
func (h *progressHandlers) studyGuide(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		g := c.StudyGuide()
		if g == nil {
			notFound(w, "study guide", "study-guide.yaml")
			return
		}
		ctx := r.Context()
		lessons, err := h.d.Progress.LessonProgressAll(ctx)
		if err != nil {
			h.fail(w, "load lesson progress", err)
			return
		}
		attempts, err := h.d.Progress.AllAttempts(ctx)
		if err != nil {
			h.fail(w, "load attempts", err)
			return
		}
		history, err := h.d.Progress.AnswerHistory(ctx)
		if err != nil {
			h.fail(w, "load answer history", err)
			return
		}
		states, err := h.d.Progress.QuestionStates(ctx)
		if err != nil {
			h.fail(w, "load question state", err)
			return
		}
		labProgress, err := h.d.Progress.LabProgressAll(ctx)
		if err != nil {
			h.fail(w, "load lab progress", err)
			return
		}

		bestBySection := map[string]float64{}
		for _, a := range attempts {
			if a.Total > 0 {
				if p := grading.Percentage(a.Score, a.Total); p > bestBySection[a.SectionID] || !hasKey(bestBySection, a.SectionID) {
					bestBySection[a.SectionID] = p
				}
			}
		}
		idx := questionIndex(c)
		var events []domain.AnswerEvent
		for _, e := range history {
			if q, ok := idx[e.QuestionID]; ok {
				e.Topic = q.Topic
				events = append(events, e)
			}
		}
		mastered := map[string]mastery.Topic{}
		for _, t := range mastery.Compute(events) {
			mastered[t.Topic] = t
		}
		now := h.d.Now()
		dueByTopic := map[string]int{}
		for _, st := range states {
			if q, ok := idx[st.QuestionID]; ok && !st.NextDueAt.After(now) {
				dueByTopic[q.Topic]++
			}
		}

		lessonDTO := func(id string) guideLessonDTO {
			l, _ := c.Lesson(id)
			status := domain.LessonNotStarted
			if lp, ok := lessons[id]; ok && (lp.Status == domain.LessonVisited || lp.Status == domain.LessonCompleted) {
				status = lp.Status
			}
			return guideLessonDTO{ID: id, Title: l.Meta.Title, SectionID: l.SectionID, Status: status}
		}
		screens := map[string]content.SimScreen{}
		for _, s := range content.SimScreens {
			screens[s.ID] = s
		}

		out := guideDTO{Sections: []guideSectionDTO{}, Beyond: g.Beyond, UnmappedLessons: []guideLessonDTO{}, Warnings: []content.Error{}, SimScreens: content.SimScreens}
		for _, w := range c.GuideWarnings() {
			out.Warnings = append(out.Warnings, w)
		}
		for _, id := range g.UnmappedLessons {
			out.UnmappedLessons = append(out.UnmappedLessons, lessonDTO(id))
		}
		for _, s := range g.Sections {
			sd := guideSectionDTO{ID: s.ID, Title: s.Title, AddedByCourse: s.AddedByCourse, Topics: []guideTopicDTO{}}
			for _, t := range s.Topics {
				td := guideTopicDTO{
					ID: t.ID, Title: t.Title, Kind: t.Kind, Coverage: t.Coverage, Note: t.Note,
					Lessons: []guideLessonDTO{}, QuizTopics: []guideQuizTopicDTO{}, Diagrams: []guideDiagramDTO{},
					Labs: []guideLabDTO{}, SimScreens: []content.SimScreen{},
				}
				seenSection := map[string]bool{}
				for _, id := range t.Lessons {
					ld := lessonDTO(id)
					td.Lessons = append(td.Lessons, ld)
					td.Progress.LessonsTotal++
					if ld.Status == domain.LessonCompleted {
						td.Progress.LessonsCompleted++
					}
					if p, ok := bestBySection[ld.SectionID]; ok && !seenSection[ld.SectionID] {
						seenSection[ld.SectionID] = true
						if td.Progress.BestSectionScorePct == nil || p > *td.Progress.BestSectionScorePct {
							best := p
							td.Progress.BestSectionScorePct = &best
						}
					}
				}
				answers, weighted := 0, 0.0
				for _, qt := range t.QuizTopics {
					qd := guideQuizTopicDTO{Topic: qt, Due: dueByTopic[qt]}
					if m, ok := mastered[qt]; ok {
						acc := round1(m.AccuracyPct)
						qd.Answers, qd.AccuracyPct, qd.Weak = m.Answers, &acc, m.Weak
						answers += m.Answers
						weighted += m.AccuracyPct * float64(m.Answers)
					}
					td.Progress.ReviewDue += qd.Due
					td.QuizTopics = append(td.QuizTopics, qd)
				}
				td.Progress.QuizAnswers = answers
				if answers > 0 {
					acc := round1(weighted / float64(answers))
					td.Progress.TopicAccuracyPct = &acc
				}
				for _, id := range t.Diagrams {
					if d, ok := c.Diagram(id); ok {
						td.Diagrams = append(td.Diagrams, guideDiagramDTO{ID: id, Title: d.Title})
					}
				}
				for _, id := range t.Labs {
					l, ok := c.Lab(id)
					if !ok {
						continue
					}
					status := domain.LabNotStarted
					if lp, ok := labProgress[id]; ok {
						status = lp.Status
					}
					td.Labs = append(td.Labs, guideLabDTO{ID: id, Title: l.Lab.Title, Status: status})
					td.Progress.LabsTotal++
					if status == domain.LabCompleted {
						td.Progress.LabsCompleted++
					}
				}
				for _, id := range t.SimScreens {
					td.SimScreens = append(td.SimScreens, screens[id])
				}
				sd.Topics = append(sd.Topics, td)
			}
			out.Sections = append(out.Sections, sd)
		}
		writeJSON(w, http.StatusOK, out)
	})(w, r)
}

func hasKey(m map[string]float64, k string) bool { _, ok := m[k]; return ok }

// ---- GET /api/activity ----

// activityWeeks is how far back the dashboard heatmap looks.
const activityWeeks = 12

type activityEventDTO struct {
	Kind string    `json:"kind"`
	At   time.Time `json:"at"`
}

type lastLessonDTO struct {
	LessonID  string    `json:"lesson_id"`
	SectionID string    `json:"section_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	At        time.Time `json:"at"`
}

type quizScoreDTO struct {
	Score     int       `json:"score"`
	Total     int       `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

type quizHistoryDTO struct {
	SectionID string         `json:"section_id"`
	Attempts  []quizScoreDTO `json:"attempts"` // the last 10, oldest first
}

type activityDTO struct {
	Since      time.Time          `json:"since"`
	Events     []activityEventDTO `json:"events"`
	LastLesson *lastLessonDTO     `json:"last_lesson"`
	Quiz       []quizHistoryDTO   `json:"quiz"`
	// Limits say what the stored timestamps cannot show, so the screen can say so instead of faking it.
	Limits []string `json:"limits"`
}

const sparkAttempts = 10

func (h *progressHandlers) activity(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		ctx := r.Context()
		now := h.d.Now()
		since := now.AddDate(0, 0, -7*activityWeeks)
		events, err := h.d.Progress.ActivityEvents(ctx, since)
		if err != nil {
			h.fail(w, "load activity", err)
			return
		}
		lessons, err := h.d.Progress.LessonProgressAll(ctx)
		if err != nil {
			h.fail(w, "load lesson progress", err)
			return
		}
		attempts, err := h.d.Progress.AllAttempts(ctx)
		if err != nil {
			h.fail(w, "load attempts", err)
			return
		}
		out := activityDTO{
			Since: since, Events: []activityEventDTO{}, Quiz: []quizHistoryDTO{},
			Limits: []string{
				"A lesson is recorded once: on the day it was completed, or on the day it was last opened if it is not completed yet. Earlier visits to the same lesson are not stored.",
				"A lab is recorded by its latest attempt and by its completion, not by every attempt.",
				"Time spent is not recorded, so the screen shows days with activity, never minutes.",
			},
		}
		idx := questionIndex(c)
		for _, e := range events {
			known := false
			switch e.Kind {
			case domain.ActLessonVisited, domain.ActLessonCompleted:
				_, known = c.Lesson(e.Ref)
			case domain.ActQuizAttempt:
				_, known = c.Section(e.Ref)
			case domain.ActReviewAnswer:
				_, known = idx[e.Ref]
			case domain.ActLabAttempt, domain.ActLabCompleted:
				_, known = listedLabOf(c, e.Ref)
			}
			if known {
				out.Events = append(out.Events, activityEventDTO{Kind: e.Kind, At: e.At})
			}
		}

		for _, lp := range lessons {
			l, ok := c.Lesson(lp.LessonID)
			if !ok || (lp.Status != domain.LessonVisited && lp.Status != domain.LessonCompleted) {
				continue
			}
			if out.LastLesson == nil || lp.UpdatedAt.After(out.LastLesson.At) {
				out.LastLesson = &lastLessonDTO{LessonID: l.Meta.ID, SectionID: l.SectionID, Title: l.Meta.Title, Status: lp.Status, At: lp.UpdatedAt}
			}
		}

		bySection := map[string][]quizScoreDTO{}
		for _, a := range attempts { // oldest first
			bySection[a.SectionID] = append(bySection[a.SectionID], quizScoreDTO{Score: a.Score, Total: a.Total, CreatedAt: a.CreatedAt})
		}
		for _, s := range c.Sections {
			list := bySection[s.Meta.ID]
			if len(list) > sparkAttempts {
				list = list[len(list)-sparkAttempts:]
			}
			out.Quiz = append(out.Quiz, quizHistoryDTO{SectionID: s.Meta.ID, Attempts: append([]quizScoreDTO{}, list...)})
		}
		writeJSON(w, http.StatusOK, out)
	})(w, r)
}
