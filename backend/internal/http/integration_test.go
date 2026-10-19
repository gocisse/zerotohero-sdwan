package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/db"
	"velocloud-academy/backend/internal/store"
	"velocloud-academy/backend/migrations"
)

// These tests need a real Postgres. Set TEST_DATABASE_URL (e.g. the docker-compose database);
// `make test` does this. Each test gets its own freshly migrated, throwaway database.
// If the variable is set but the database is unreachable the tests FAIL; if it is unset they skip.

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "academy_test_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	cfg, _ := pgxpool.ParseConfig(adminURL)
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	if err := db.Migrate(ctx, pool, migrations.FS, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		admin.Close()
	})
	return pool
}

const secretMark = "SECRET-EXPLANATION"

// writeCourse writes a small course. full=false removes section s2 and question q-single,
// simulating content that changed after progress was recorded.
func writeCourse(t *testing.T, dir string, full bool) {
	t.Helper()
	w := func(rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sections := "[s1, s2]"
	if !full {
		sections = "[s1]"
	}
	w("course.yaml", "title: T\ndescription: D\nsections: "+sections+"\n")
	w("sections/01-a/section.yaml", "id: s1\ntitle: S1\nsummary: sum\nestimated_minutes: 5\nlessons: [l1, l2]\n")
	lesson := func(id string) string {
		return "---\nid: " + id + "\ntitle: Lesson " + id + "\nsummary: s\nestimated_minutes: 2\ntopics: [alpha]\n---\n\nBody " + id + "\n"
	}
	w("sections/01-a/lessons/01-l1.mdx", lesson("l1"))
	w("sections/01-a/lessons/02-l2.mdx", lesson("l2"))
	quiz := ""
	if full {
		quiz += `- id: q-single
  topic: alpha
  difficulty: 1
  type: single_choice
  prompt: Pick a.
  options: [{id: a, text: A}, {id: b, text: B, explanation: "` + secretMark + ` b"}]
  correct: a
  explanation: ` + secretMark + ` single
  lesson_ref: l1
`
	}
	quiz += `- id: q-multi
  topic: beta
  difficulty: 2
  type: multi_choice
  prompt: Pick a and b.
  options: [{id: a, text: A}, {id: b, text: B}, {id: c, text: C}]
  correct: [a, b]
  explanation: ` + secretMark + ` multi
  lesson_ref: l2
- id: q-tf
  topic: gamma
  difficulty: 1
  type: true_false
  prompt: True?
  correct: true
  explanation: ` + secretMark + ` tf
- id: q-order
  topic: alpha
  difficulty: 3
  type: ordering
  prompt: Order.
  options: [{id: a, text: A}, {id: b, text: B}, {id: c, text: C}]
  correct: [b, c, a]
  explanation: ` + secretMark + ` order
`
	w("sections/01-a/quiz.yaml", quiz)
	if full {
		w("sections/02-b/section.yaml", "id: s2\ntitle: S2\nsummary: sum\nestimated_minutes: 5\nlessons: [l3]\n")
		w("sections/02-b/lessons/01-l3.mdx", lesson("l3"))
		w("sections/02-b/quiz.yaml", `- id: q-s2
  topic: delta
  difficulty: 1
  type: true_false
  prompt: S2?
  correct: false
  explanation: `+secretMark+` s2
`)
	}
}

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	dir  string
	srv  http.Handler
	now  time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, pool: newTestPool(t), dir: t.TempDir(), now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	writeCourse(t, e.dir, true)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := store.New(e.pool)
	e.srv = NewRouter(Deps{
		Health: st, Content: content.NewProvider(e.dir, log), Progress: st, Log: log,
		Now: func() time.Time { return e.now },
	})
	return e
}

func (e *env) call(method, path string, body any) (int, []byte) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, httptest.NewRequest(method, path, rd))
	return rec.Code, rec.Body.Bytes()
}

func (e *env) must(method, path string, body any, wantCode int) []byte {
	e.t.Helper()
	code, b := e.call(method, path, body)
	if code != wantCode {
		e.t.Fatalf("%s %s = %d, want %d; body %s", method, path, code, wantCode, b)
	}
	return b
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", query, err)
	}
	return n
}

type answers map[string]any

func body(a answers) map[string]any {
	list := []map[string]any{}
	for id, ans := range a {
		list = append(list, map[string]any{"question_id": id, "answer": ans})
	}
	return map[string]any{"answers": list}
}

type attemptResp struct {
	AttemptID  int64   `json:"attempt_id"`
	Score      int     `json:"score"`
	Total      int     `json:"total"`
	Percentage float64 `json:"percentage"`
	Passed     bool    `json:"passed"`
	Results    []struct {
		QuestionID         string            `json:"question_id"`
		Topic              string            `json:"topic"`
		Correct            bool              `json:"correct"`
		PartialScore       float64           `json:"partial_score"`
		CorrectAnswer      any               `json:"correct_answer"`
		Explanation        string            `json:"explanation"`
		OptionExplanations map[string]string `json:"option_explanations"`
		LessonRef          string            `json:"lesson_ref"`
	} `json:"results"`
}

func decode[T any](t *testing.T, b []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return v
}

var allRight = answers{"q-single": "a", "q-multi": []string{"a", "b"}, "q-tf": "true", "q-order": []string{"b", "c", "a"}}

func TestSubmitAttemptPersistsRowsAndUpdatesState(t *testing.T) {
	e := newEnv(t)
	// q-single right, q-multi partial (a only => 1/2), q-tf wrong, q-order unanswered.
	b := e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{"q-single": "a", "q-multi": []string{"a"}, "q-tf": "false"}), 200)
	r := decode[attemptResp](t, b)

	if r.Score != 1 || r.Total != 4 || r.Percentage != 25 || r.Passed {
		t.Fatalf("score=%d total=%d pct=%v passed=%v", r.Score, r.Total, r.Percentage, r.Passed)
	}
	if len(r.Results) != 4 || r.Results[0].QuestionID != "q-single" || r.Results[3].QuestionID != "q-order" {
		t.Fatalf("results not in quiz order: %+v", r.Results)
	}
	multi := r.Results[1]
	if multi.Correct || multi.PartialScore != 0.5 || multi.LessonRef != "l2" || !strings.Contains(multi.Explanation, secretMark) {
		t.Errorf("multi result = %+v", multi)
	}

	// Persisted rows.
	if n := e.count(`SELECT count(*) FROM quiz_attempts WHERE section_id='s1' AND score=1 AND total=4`); n != 1 {
		t.Errorf("quiz_attempts rows = %d", n)
	}
	if n := e.count(`SELECT count(*) FROM quiz_attempt_answers`); n != 4 {
		t.Errorf("quiz_attempt_answers rows = %d", n)
	}
	if n := e.count(`SELECT count(*) FROM quiz_attempt_answers WHERE question_id='q-multi' AND NOT correct AND partial_score=0.5`); n != 1 {
		t.Error("partial credit row missing")
	}

	// Leitner: the correct answer is promoted to box 2 (+3d); wrong/unanswered are box 1 and due now.
	if n := e.count(`SELECT count(*) FROM question_state WHERE question_id='q-single' AND box=2 AND next_due_at=$1`, e.now.Add(72*time.Hour)); n != 1 {
		t.Error("q-single should be box 2, due in 3 days")
	}
	for _, id := range []string{"q-multi", "q-tf", "q-order"} {
		if n := e.count(`SELECT count(*) FROM question_state WHERE question_id=$1 AND box=1 AND last_result='wrong' AND next_due_at=$2`, id, e.now); n != 1 {
			t.Errorf("%s should be box 1 and due immediately", id)
		}
	}

	// History endpoint.
	h := decode[struct {
		Attempts []struct {
			ID           int64
			Score, Total int
		}
	}](t, e.must("GET", "/api/sections/s1/quiz/attempts", nil, 200))
	if len(h.Attempts) != 1 || h.Attempts[0].Score != 1 {
		t.Errorf("history = %+v", h)
	}
}

func TestBadSubmissionsAreRejectedAndPersistNothing(t *testing.T) {
	e := newEnv(t)
	cases := map[string]any{
		"unknown question id": body(answers{"nope": "a"}),
		"wrong shape":         body(answers{"q-single": []string{"a"}}),
		"invalid json":        "not an object",
	}
	for name, b := range cases {
		e.must("POST", "/api/sections/s1/quiz/attempts", b, 400)
		_ = name
	}
	dup := map[string]any{"answers": []map[string]any{{"question_id": "q-tf", "answer": "true"}, {"question_id": "q-tf", "answer": "false"}}}
	e.must("POST", "/api/sections/s1/quiz/attempts", dup, 400)
	e.must("POST", "/api/sections/nope/quiz/attempts", body(answers{}), 404)
	if n := e.count(`SELECT count(*) FROM quiz_attempts`) + e.count(`SELECT count(*) FROM question_state`); n != 0 {
		t.Errorf("rejected submissions persisted %d rows", n)
	}
}

func TestSkipIsIdempotent(t *testing.T) {
	e := newEnv(t)
	first := decode[struct {
		QuizSkippedAt time.Time `json:"quiz_skipped_at"`
	}](t, e.must("POST", "/api/sections/s1/quiz/skip", nil, 200))
	e.now = e.now.Add(48 * time.Hour)
	second := decode[struct {
		QuizSkippedAt time.Time `json:"quiz_skipped_at"`
	}](t, e.must("POST", "/api/sections/s1/quiz/skip", nil, 200))
	if !first.QuizSkippedAt.Equal(second.QuizSkippedAt) {
		t.Errorf("second skip changed the timestamp: %v -> %v", first.QuizSkippedAt, second.QuizSkippedAt)
	}
	if n := e.count(`SELECT count(*) FROM section_state WHERE section_id='s1'`); n != 1 {
		t.Errorf("section_state rows = %d", n)
	}
	e.must("POST", "/api/sections/nope/quiz/skip", nil, 404)

	p := decode[struct {
		Sections []struct {
			SectionID   string `json:"section_id"`
			QuizSkipped bool   `json:"quiz_skipped"`
		}
	}](t, e.must("GET", "/api/progress", nil, 200))
	if !p.Sections[0].QuizSkipped || p.Sections[1].QuizSkipped {
		t.Errorf("sections = %+v", p.Sections)
	}
}

func TestLessonProgressNeverDowngrades(t *testing.T) {
	e := newEnv(t)
	put := func(status string) (string, *time.Time) {
		r := decode[struct {
			Status      string     `json:"status"`
			CompletedAt *time.Time `json:"completed_at"`
		}](t, e.must("PUT", "/api/lessons/l1/progress", map[string]string{"status": status}, 200))
		return r.Status, r.CompletedAt
	}
	if s, c := put("visited"); s != "visited" || c != nil {
		t.Fatalf("visited -> %s %v", s, c)
	}
	s, c1 := put("completed")
	if s != "completed" || c1 == nil {
		t.Fatalf("completed -> %s %v", s, c1)
	}
	e.now = e.now.Add(time.Hour)
	s, c2 := put("visited") // must not downgrade
	if s != "completed" || c2 == nil || !c1.Equal(*c2) {
		t.Fatalf("visited after completed -> %s %v (completed_at was %v)", s, c2, c1)
	}
	if s, _ := put("completed"); s != "completed" { // idempotent
		t.Fatalf("repeat completed -> %s", s)
	}
	if n := e.count(`SELECT count(*) FROM lesson_progress WHERE lesson_id='l1' AND status='completed'`); n != 1 {
		t.Errorf("rows = %d", n)
	}
	e.must("PUT", "/api/lessons/nope/progress", map[string]string{"status": "visited"}, 404)
	e.must("PUT", "/api/lessons/l1/progress", map[string]string{"status": "in_progress"}, 400)
	e.must("PUT", "/api/lessons/l1/progress", map[string]string{"status": "not_started"}, 400)
}

// Soft gating: a failing score must change nothing except the attempt/answer/question-state rows,
// and every other action must remain available afterwards.
func TestFailingAttemptDoesNotRestrictAnything(t *testing.T) {
	e := newEnv(t)
	e.must("PUT", "/api/lessons/l1/progress", map[string]string{"status": "completed"}, 200)
	before := func() [3]int {
		return [3]int{
			e.count(`SELECT count(*) FROM lesson_progress`),
			e.count(`SELECT count(*) FROM section_state WHERE quiz_skipped_at IS NOT NULL`),
			e.count(`SELECT count(*) FROM lesson_progress WHERE status='completed'`),
		}
	}
	snap := before()

	r := decode[attemptResp](t, e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{}), 200))
	if r.Score != 0 || r.Passed {
		t.Fatalf("score=%d passed=%v", r.Score, r.Passed)
	}
	if after := before(); after != snap {
		t.Errorf("failing attempt changed lesson/section state: %v -> %v", snap, after)
	}

	// Everything keeps working: navigation data, other lessons, skipping, retrying, the next section.
	e.must("GET", "/api/course", nil, 200)
	e.must("GET", "/api/lessons/l2", nil, 200)
	e.must("GET", "/api/lessons/l3", nil, 200)
	e.must("GET", "/api/sections/s2/quiz", nil, 200)
	e.must("PUT", "/api/lessons/l3/progress", map[string]string{"status": "visited"}, 200)
	e.must("POST", "/api/sections/s1/quiz/skip", nil, 200)
	retry := decode[attemptResp](t, e.must("POST", "/api/sections/s1/quiz/attempts", body(allRight), 200))
	if retry.Score != 4 || !retry.Passed {
		t.Errorf("retry = %+v", retry)
	}
	if n := e.count(`SELECT count(*) FROM quiz_attempts WHERE section_id='s1'`); n != 2 {
		t.Errorf("attempts = %d, want 2", n)
	}
	// The earlier wrong answers moved up a box on the correct retry, not stuck.
	if n := e.count(`SELECT count(*) FROM question_state WHERE question_id='q-tf' AND box=2`); n != 1 {
		t.Error("q-tf should have been promoted on the correct retry")
	}
}

func TestOrphanedRowsAreIgnoredNotDeleted(t *testing.T) {
	e := newEnv(t)
	e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{"q-tf": "false"}), 200) // all wrong: everything due
	e.must("POST", "/api/sections/s2/quiz/attempts", body(answers{"q-s2": "true"}), 200)
	e.must("PUT", "/api/lessons/l3/progress", map[string]string{"status": "completed"}, 200)
	e.must("POST", "/api/sections/s2/quiz/skip", nil, 200)
	// A row for ids that never existed at all.
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO lesson_progress (lesson_id, status) VALUES ('ghost-lesson','completed');
		 INSERT INTO section_state (section_id, quiz_skipped_at) VALUES ('ghost-section', now());
		 INSERT INTO question_state (question_id, topic, box, last_result, last_answered_at, next_due_at) VALUES ('ghost-q','zeta',1,'wrong',now(),now())`); err != nil {
		t.Fatal(err)
	}
	counts := func() [5]int {
		return [5]int{
			e.count(`SELECT count(*) FROM lesson_progress`), e.count(`SELECT count(*) FROM section_state`),
			e.count(`SELECT count(*) FROM question_state`), e.count(`SELECT count(*) FROM quiz_attempts`),
			e.count(`SELECT count(*) FROM quiz_attempt_answers`),
		}
	}
	snap := counts()

	// Content changes: section s2 (with lesson l3 and q-s2) and question q-single disappear.
	time.Sleep(10 * time.Millisecond)
	if err := os.RemoveAll(filepath.Join(e.dir, "sections/02-b")); err != nil {
		t.Fatal(err)
	}
	writeCourse(t, e.dir, false)

	type progResp struct {
		LessonsTotal int `json:"lessons_total"`
		Lessons      []struct {
			LessonID string `json:"lesson_id"`
		}
		Sections []struct {
			SectionID    string `json:"section_id"`
			QuizAttempts int    `json:"quiz_attempts"`
		}
	}
	prog := decode[progResp](t, e.must("GET", "/api/progress", nil, 200))
	if prog.LessonsTotal != 2 || len(prog.Sections) != 1 || prog.Sections[0].SectionID != "s1" || prog.Sections[0].QuizAttempts != 1 {
		t.Errorf("progress = %+v", prog)
	}
	for _, l := range prog.Lessons {
		if strings.HasPrefix(l.LessonID, "ghost") || l.LessonID == "l3" {
			t.Errorf("orphan lesson leaked into progress: %s", l.LessonID)
		}
	}

	topics := decode[struct{ Topics []struct{ Topic string } }](t, e.must("GET", "/api/progress/topics", nil, 200))
	for _, tp := range topics.Topics {
		if tp.Topic == "delta" || tp.Topic == "zeta" {
			t.Errorf("orphan topic leaked: %s", tp.Topic)
		}
	}
	q := decode[struct {
		DueCount  int `json:"due_count"`
		Questions []struct{ ID string }
	}](t, e.must("GET", "/api/review/queue", nil, 200))
	for _, qq := range q.Questions {
		if qq.ID == "q-s2" || qq.ID == "ghost-q" || qq.ID == "q-single" {
			t.Errorf("orphan question in queue: %s", qq.ID)
		}
	}
	// Orphan question ids are unknown to review now.
	e.must("POST", "/api/review/answers", body(answers{"q-s2": "true"}), 400)
	// Skipping / attempts on the removed section are 404, not 500.
	e.must("POST", "/api/sections/s2/quiz/skip", nil, 404)
	e.must("GET", "/api/sections/s2/quiz/attempts", nil, 404)

	if after := counts(); after != snap {
		t.Errorf("rows were deleted or added: %v -> %v", snap, after)
	}
}

func TestReviewQueueAndAnswers(t *testing.T) {
	e := newEnv(t)
	// Fail the quiz on purpose except q-single: three wrong answers become due immediately.
	e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{"q-single": "a", "q-multi": []string{"c"}, "q-tf": "false", "q-order": []string{"a", "b", "c"}}), 200)

	raw := e.must("GET", "/api/review/queue", nil, 200)
	type queueResp struct {
		DueCount      int        `json:"due_count"`
		UpcomingCount int        `json:"upcoming_count"`
		NextDueAt     *time.Time `json:"next_due_at"`
		Questions     []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	qq := decode[queueResp](t, raw)
	if qq.DueCount != 3 || qq.UpcomingCount != 1 || len(qq.Questions) != 3 {
		t.Fatalf("queue = %+v", qq)
	}
	if qq.NextDueAt == nil || !qq.NextDueAt.Equal(e.now.Add(72*time.Hour)) {
		t.Errorf("next_due_at = %v", qq.NextDueAt)
	}
	assertNoAnswers(t, raw)

	// Answer two of them: tf right (promoted), multi wrong again (stays box 1).
	attemptsBefore := e.count(`SELECT count(*) FROM quiz_attempts`)
	e.now = e.now.Add(time.Hour)
	rb := e.must("POST", "/api/review/answers", body(answers{"q-tf": "true", "q-multi": []string{"c"}}), 200)
	rr := decode[struct {
		Results []struct {
			QuestionID  string `json:"question_id"`
			Correct     bool
			Explanation string
		} `json:"results"`
		DueCount int `json:"due_count"`
	}](t, rb)
	// Submission order comes from a map, so look results up by question id.
	byID := map[string]bool{}
	for _, r := range rr.Results {
		byID[r.QuestionID] = r.Correct
		if r.Explanation == "" {
			t.Fatalf("result for %s has no explanation: %+v", r.QuestionID, rr)
		}
	}
	if len(rr.Results) != 2 || !byID["q-tf"] || byID["q-multi"] {
		t.Fatalf("results = %+v", rr)
	}
	if rr.DueCount != 2 { // q-multi (wrong again) and q-order still due; q-tf moved out
		t.Errorf("due_count after review = %d", rr.DueCount)
	}
	if n := e.count(`SELECT count(*) FROM quiz_attempts`); n != attemptsBefore {
		t.Error("review created a quiz attempt")
	}
	if n := e.count(`SELECT count(*) FROM review_answers`); n != 2 {
		t.Errorf("review_answers rows = %d", n)
	}
	if n := e.count(`SELECT count(*) FROM question_state WHERE question_id='q-tf' AND box=2 AND times_correct=1 AND times_wrong=1`); n != 1 {
		t.Error("q-tf should be promoted to box 2")
	}
	if n := e.count(`SELECT count(*) FROM question_state WHERE question_id='q-multi' AND box=1 AND times_wrong=2`); n != 1 {
		t.Error("q-multi should stay in box 1 with 2 wrong answers")
	}

	e.must("POST", "/api/review/answers", body(answers{"nope": "a"}), 400)
	e.must("POST", "/api/review/answers", map[string]any{"answers": []any{}}, 400)

	// 15-question session cap is enforced by SelectDue (unit-tested); here: time passes, all become due.
	e.now = e.now.Add(100 * 24 * time.Hour)
	all := decode[struct {
		DueCount int `json:"due_count"`
	}](t, e.must("GET", "/api/review/queue", nil, 200))
	if all.DueCount != 4 {
		t.Errorf("after 100 days due = %d, want 4", all.DueCount)
	}
}

func TestTopicsAndProgressSummary(t *testing.T) {
	e := newEnv(t)
	// Fail topic alpha (q-single + q-order) twice, pass the rest: alpha has 4 wrong answers.
	for i := 0; i < 2; i++ {
		e.now = e.now.Add(time.Hour)
		e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{"q-multi": []string{"a", "b"}, "q-tf": "true"}), 200)
	}
	e.now = e.now.Add(time.Hour)
	e.must("POST", "/api/sections/s1/quiz/attempts", body(allRight), 200)

	tp := decode[struct {
		Topics []struct {
			Topic       string  `json:"topic"`
			Answers     int     `json:"answers"`
			AccuracyPct float64 `json:"accuracy_pct"`
			Weak        bool    `json:"weak"`
			Lessons     []struct {
				ID string `json:"id"`
			} `json:"lessons"`
		} `json:"topics"`
	}](t, e.must("GET", "/api/progress/topics", nil, 200))
	if len(tp.Topics) != 3 || tp.Topics[0].Topic != "alpha" {
		t.Fatalf("topics = %+v", tp.Topics)
	}
	a := tp.Topics[0]
	// 6 alpha answers oldest..newest: 0,0,0,0,1,1 with weights 1..6 -> (5+6)/21.
	if a.Answers != 6 || a.AccuracyPct != 52.4 || !a.Weak {
		t.Errorf("alpha = %+v", a)
	}
	if len(a.Lessons) == 0 {
		t.Error("alpha should list lessons to review")
	}

	e.must("PUT", "/api/lessons/l1/progress", map[string]string{"status": "completed"}, 200)
	p := decode[struct {
		OverallPercentage float64 `json:"overall_percentage"`
		LessonsCompleted  int     `json:"lessons_completed"`
		LessonsTotal      int     `json:"lessons_total"`
		Sections          []struct {
			SectionID      string   `json:"section_id"`
			QuizAttempts   int      `json:"quiz_attempts"`
			BestScorePct   *float64 `json:"best_score_pct"`
			LatestScorePct *float64 `json:"latest_score_pct"`
		} `json:"sections"`
	}](t, e.must("GET", "/api/progress", nil, 200))
	s1, s2 := p.Sections[0], p.Sections[1]
	if p.LessonsTotal != 3 || p.LessonsCompleted != 1 || p.OverallPercentage != 33.3 {
		t.Errorf("overall = %+v", p)
	}
	if s1.QuizAttempts != 3 || *s1.BestScorePct != 100 || *s1.LatestScorePct != 100 {
		t.Errorf("s1 = %+v", s1)
	}
	if s2.QuizAttempts != 0 || s2.BestScorePct != nil || s2.LatestScorePct != nil {
		t.Errorf("s2 = %+v", s2)
	}
}

func TestWeakTopicFlag(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 3; i++ {
		e.now = e.now.Add(time.Hour)
		e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{}), 200)
	}
	tp := decode[struct {
		Topics []struct {
			Topic string `json:"topic"`
			Weak  bool   `json:"weak"`
		} `json:"topics"`
	}](t, e.must("GET", "/api/progress/topics", nil, 200))
	for _, x := range tp.Topics {
		if !x.Weak {
			t.Errorf("topic %s should be weak after 3 failed attempts", x.Topic)
		}
	}
}

// assertNoAnswers fails if a pre-grading payload contains anything that reveals answers.
func assertNoAnswers(t *testing.T, raw []byte) {
	t.Helper()
	for _, banned := range []string{`"correct`, `"explanation`, `option_explanations`, `lesson_ref`, secretMark} {
		if strings.Contains(string(raw), banned) {
			t.Errorf("payload leaks %q: %s", banned, raw)
		}
	}
}

func TestQuizAndReviewQueueNeverRevealAnswers(t *testing.T) {
	e := newEnv(t)
	assertNoAnswers(t, e.must("GET", "/api/sections/s1/quiz", nil, 200))
	assertNoAnswers(t, e.must("GET", "/api/sections/s2/quiz", nil, 200))

	e.must("POST", "/api/sections/s1/quiz/attempts", body(answers{}), 200)
	raw := e.must("GET", "/api/review/queue", nil, 200)
	if !strings.Contains(string(raw), `"q-order"`) {
		t.Fatalf("queue should contain questions: %s", raw)
	}
	assertNoAnswers(t, raw)
}
