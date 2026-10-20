package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/store"
)

// realContentEnv serves the repository's real /content (read only) over a fresh throwaway database.
func realContentEnv(t *testing.T) (*env, *store.Store, *content.Course) {
	t.Helper()
	dir := "../../../content"
	if _, err := os.Stat(dir); err != nil {
		t.Skip("content directory not found")
	}
	e := &env{t: t, pool: newTestPool(t), dir: dir, now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := store.New(e.pool)
	prov := content.NewProvider(dir, log)
	course, err := prov.Current()
	if err != nil {
		t.Fatalf("real content invalid: %v", err)
	}
	e.srv = NewRouter(Deps{Health: st, Content: prov, Progress: st, Log: log, Now: func() time.Time { return e.now }})
	return e, st, course
}

func getGuide(t *testing.T, e *env) guideDTO {
	t.Helper()
	var g guideDTO
	if err := json.Unmarshal(e.must("GET", "/api/study-guide", nil, 200), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func findTopic(t *testing.T, g guideDTO, id string) guideTopicDTO {
	t.Helper()
	for _, s := range g.Sections {
		for _, tp := range s.Topics {
			if tp.ID == id {
				return tp
			}
		}
	}
	t.Fatalf("topic %s not in the response", id)
	return guideTopicDTO{}
}

func TestStudyGuideNotFoundWithoutAGuide(t *testing.T) {
	e := newEnv(t)
	e.must("GET", "/api/study-guide", nil, 404)
}

func TestStudyGuideStructureAndEmptyProgress(t *testing.T) {
	e, _, course := realContentEnv(t)
	g := getGuide(t, e)
	want := course.StudyGuide()
	if len(g.Sections) != len(want.Sections) {
		t.Fatalf("sections = %d, want %d", len(g.Sections), len(want.Sections))
	}
	for si, s := range want.Sections {
		if g.Sections[si].ID != s.ID || len(g.Sections[si].Topics) != len(s.Topics) {
			t.Fatalf("section %s does not match the file", s.ID)
		}
		for ti, tp := range s.Topics {
			got := g.Sections[si].Topics[ti]
			if got.ID != tp.ID || got.Coverage != tp.Coverage || len(got.Lessons) != len(tp.Lessons) {
				t.Errorf("topic %s does not match the file", tp.ID)
			}
			if tp.Coverage == content.CoverageNone && len(got.Lessons) != 0 {
				t.Errorf("topic %s: coverage none must carry no lessons", tp.ID)
			}
			p := got.Progress
			if p.LessonsCompleted != 0 || p.LabsCompleted != 0 || p.ReviewDue != 0 || p.BestSectionScorePct != nil || p.TopicAccuracyPct != nil {
				t.Errorf("topic %s: a fresh database must show no progress, got %+v", tp.ID, p)
			}
			if p.LessonsTotal != len(tp.Lessons) {
				t.Errorf("topic %s: lessons_total = %d, want %d", tp.ID, p.LessonsTotal, len(tp.Lessons))
			}
		}
	}
	if len(g.UnmappedLessons) != len(want.UnmappedLessons) {
		t.Errorf("unmapped lessons = %d, want %d", len(g.UnmappedLessons), len(want.UnmappedLessons))
	}
	if len(g.SimScreens) == 0 {
		t.Error("the sim screen list must be served")
	}
}

func TestStudyGuideDerivesProgress(t *testing.T) {
	e, st, course := realContentEnv(t)
	ctx := context.Background()

	// A lesson that one topic maps (arch-edge -> vce) and that another lesson shares a section with.
	e.must("PUT", "/api/lessons/arch-edge/progress", map[string]string{"status": "completed"}, 200)
	e.must("PUT", "/api/lessons/arch-gateway/progress", map[string]string{"status": "visited"}, 200)

	// A quiz attempt on the architecture section: answer one single-choice question correctly.
	sec, _ := course.Section("architecture")
	var q content.Question
	for _, c := range sec.Quiz {
		if c.Type == content.SingleChoice {
			q = c
			break
		}
	}
	e.must("POST", "/api/sections/architecture/quiz/attempts", map[string]any{"answers": []map[string]any{{"question_id": q.ID, "answer": q.Correct[0]}}}, 200)

	// A lab, and rows for ids that do not exist (they must be ignored everywhere).
	if _, err := st.SetLabStatus(ctx, "video-wrong-link", domain.LabCompleted, e.now); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetLabStatus(ctx, "deleted-lab", domain.LabCompleted, e.now); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetLessonProgress(ctx, "deleted-lesson", "architecture", domain.LessonCompleted, e.now); err != nil {
		t.Fatal(err)
	}

	g := getGuide(t, e)
	cases := []struct {
		topic                     string
		lessonsDone, lessonsTotal int
		hasBest, hasAccuracy      bool
		labsDone, labsTotal       int
		statuses                  map[string]string
	}{
		{"vce", 1, 1, true, false, 0, 0, map[string]string{"arch-edge": "completed"}},
		{"vcg", 0, 1, true, false, 0, 0, map[string]string{"arch-gateway": "visited"}},
		{"dmpo-3-link-steering", 0, 2, false, false, 1, 1, map[string]string{"dmpo-steering": "not_started"}},
		// A lessonsTotal of -1 means "however many lessons the topic maps": later steps add lessons to a topic, and the point of
		// these cases is that nothing is done, not how many lessons there are.
		{"policy-1", 0, -1, false, false, 1, 1, map[string]string{"cfg-business-policy": "not_started", "cfg-firewall": "not_started", "cfg-apps-qos": "not_started"}},
		{"gold-site", 0, -1, false, false, 0, 0, map[string]string{"top-gold": "not_started"}},
	}
	for _, c := range cases {
		tp := findTopic(t, g, c.topic)
		if c.lessonsTotal < 0 {
			c.lessonsTotal = len(tp.Lessons)
		}
		p := tp.Progress
		if p.LessonsCompleted != c.lessonsDone || p.LessonsTotal != c.lessonsTotal {
			t.Errorf("%s: lessons %d/%d, want %d/%d", c.topic, p.LessonsCompleted, p.LessonsTotal, c.lessonsDone, c.lessonsTotal)
		}
		if (p.BestSectionScorePct != nil) != c.hasBest {
			t.Errorf("%s: best section score present = %v, want %v", c.topic, p.BestSectionScorePct != nil, c.hasBest)
		}
		if p.LabsCompleted != c.labsDone || p.LabsTotal != c.labsTotal {
			t.Errorf("%s: labs %d/%d, want %d/%d", c.topic, p.LabsCompleted, p.LabsTotal, c.labsDone, c.labsTotal)
		}
		for _, l := range tp.Lessons {
			if want, ok := c.statuses[l.ID]; ok && l.Status != want {
				t.Errorf("%s: lesson %s status %s, want %s", c.topic, l.ID, l.Status, want)
			}
		}
	}

	// Each quiz topic's accuracy equals what /api/progress/topics reports (unanswered questions count as wrong).
	var topics struct {
		Topics []topicDTO `json:"topics"`
	}
	if err := json.Unmarshal(e.must("GET", "/api/progress/topics", nil, 200), &topics); err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{}
	for _, tp := range topics.Topics {
		want[tp.Topic] = tp.AccuracyPct
	}
	checked := 0
	for _, s := range g.Sections {
		for _, tp := range s.Topics {
			for _, qt := range tp.QuizTopics {
				w, answered := want[qt.Topic]
				if !answered {
					if qt.AccuracyPct != nil || qt.Answers != 0 {
						t.Errorf("%s / %s: unanswered topic must report no accuracy", tp.ID, qt.Topic)
					}
					continue
				}
				checked++
				if qt.AccuracyPct == nil || *qt.AccuracyPct != w {
					t.Errorf("%s / %s: accuracy %v, want %v", tp.ID, qt.Topic, qt.AccuracyPct, w)
				}
			}
		}
	}
	if checked == 0 {
		t.Error("no guide topic reported an answered quiz topic")
	}
}

func TestStudyGuideCountsReviewItemsDue(t *testing.T) {
	e, _, course := realContentEnv(t)
	sec, _ := course.Section("architecture")
	q := sec.Quiz[0]
	// A wrong answer sends the question back to box 1, due immediately.
	e.must("POST", "/api/sections/architecture/quiz/attempts", map[string]any{"answers": []map[string]any{{"question_id": q.ID, "answer": "definitely-not-an-option"}}}, 200)
	g := getGuide(t, e)
	due := 0
	for _, s := range g.Sections {
		for _, tp := range s.Topics {
			for _, qt := range tp.QuizTopics {
				if qt.Topic == q.Topic {
					due += qt.Due
					if tp.Progress.ReviewDue != qt.Due && len(tp.QuizTopics) == 1 {
						t.Errorf("%s: review_due %d != quiz topic due %d", tp.ID, tp.Progress.ReviewDue, qt.Due)
					}
				}
			}
		}
	}
	if due == 0 {
		t.Errorf("the missed question's topic %q reports nothing due", q.Topic)
	}
}

func TestActivityListsEventsAndIgnoresOrphans(t *testing.T) {
	e, st, _ := realContentEnv(t)
	ctx := context.Background()
	e.must("PUT", "/api/lessons/arch-edge/progress", map[string]string{"status": "completed"}, 200)
	e.now = e.now.Add(time.Hour)
	e.must("PUT", "/api/lessons/arch-gateway/progress", map[string]string{"status": "visited"}, 200)
	if _, err := st.SetLessonProgress(ctx, "deleted-lesson", "architecture", domain.LessonCompleted, e.now); err != nil {
		t.Fatal(err)
	}
	var a activityDTO
	if err := json.Unmarshal(e.must("GET", "/api/activity", nil, 200), &a); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, ev := range a.Events {
		kinds[ev.Kind]++
	}
	if kinds[domain.ActLessonCompleted] != 1 || kinds[domain.ActLessonVisited] != 1 || len(a.Events) != 2 {
		t.Errorf("events = %v, want exactly one completed and one visited lesson (the deleted lesson is ignored)", kinds)
	}
	if a.LastLesson == nil || a.LastLesson.LessonID != "arch-gateway" {
		t.Errorf("last lesson = %+v, want arch-gateway (the most recent touch)", a.LastLesson)
	}
	if len(a.Limits) == 0 {
		t.Error("the response must state what the timestamps cannot show")
	}
	if len(a.Quiz) == 0 {
		t.Error("quiz history must list every section, even with no attempts")
	}
}
