package http

import (
	"strings"
	"testing"
)

type selfCheckBody struct {
	Questions []struct {
		ID           string `json:"id"`
		Topic        string `json:"topic"`
		SectionID    string `json:"section_id"`
		Type         string `json:"type"`
		Options      []struct{ ID, Text string }
		Explanation  string `json:"explanation"`
		CorrectField any    `json:"correct"`
	} `json:"questions"`
	Screens []struct {
		ID      string `json:"id"`
		Prompt  string `json:"prompt"`
		Choices []struct{ ID, Text string }
		Screen  struct{ Label, Path string }
	} `json:"screens"`
}

// The self-check draws 12 questions across the sections, grades them with the quiz code and stores NOTHING.
func TestSelfCheckIsUngradedAndStoresNothing(t *testing.T) {
	e := newLabEnv(t)
	tables := []string{"quiz_attempts", "quiz_attempt_answers", "question_state", "review_answers", "lesson_progress", "section_state", "lab_progress"}
	counts := func() []int {
		var out []int
		for _, tb := range tables {
			out = append(out, e.count(`SELECT count(*) FROM `+tb))
		}
		return out
	}
	before := counts()

	body := e.must("GET", "/api/self-check", nil, 200)
	sc := decode[selfCheckBody](t, body)
	if len(sc.Questions) != 12 {
		t.Fatalf("want 12 questions, got %d", len(sc.Questions))
	}
	sections, ids := map[string]int{}, map[string]bool{}
	for _, q := range sc.Questions {
		sections[q.SectionID]++
		if ids[q.ID] {
			t.Errorf("duplicate question %s", q.ID)
		}
		ids[q.ID] = true
		if q.Explanation != "" {
			t.Error("a self-check question carries its explanation before grading")
		}
	}
	if len(sections) < 5 {
		t.Errorf("the 12 questions should span the sections, got %v", sections)
	}
	for sid, n := range sections {
		if n > 3 {
			t.Errorf("section %s supplies %d questions", sid, n)
		}
	}
	if len(sc.Screens) != 3 {
		t.Fatalf("want 3 screen tasks, got %d", len(sc.Screens))
	}
	s := string(body)
	for _, k := range []string{`"probe"`, `"explanation"`, `"equals"`, `"value"`, `"correct"`, `"patches"`} {
		if strings.Contains(s, k) {
			t.Errorf("GET /api/self-check carries the key %s", k)
		}
	}

	// Grade a partial set of answers: nothing is stored.
	var answers []map[string]any
	for _, q := range sc.Questions[:3] {
		answers = append(answers, map[string]any{"question_id": q.ID, "answer": nil})
	}
	graded := decode[struct {
		Score, Total int
		Results      []struct {
			QuestionID string `json:"question_id"`
			Correct    bool
		}
	}](t, e.must("POST", "/api/self-check/grade", map[string]any{"answers": answers}, 200))
	if graded.Total != 3 || len(graded.Results) != 3 {
		t.Errorf("%+v", graded)
	}
	e.must("POST", "/api/self-check/grade", map[string]any{"answers": []any{}}, 400)
	e.must("POST", "/api/self-check/grade", map[string]any{"answers": []map[string]any{{"question_id": "no-such-question", "answer": "x"}}}, 400)

	// Screen tasks: wrong answers give feedback, nothing is stored and nothing is locked.
	for _, sc := range sc.Screens {
		for _, c := range sc.Choices {
			r := decode[struct {
				Correct     bool
				Expected    string
				Explanation string
			}](t, e.must("POST", "/api/self-check/screens/"+sc.ID+"/check", map[string]any{"answer": c.ID}, 200))
			if r.Explanation == "" || r.Expected == "" {
				t.Errorf("%s/%s: %+v", sc.ID, c.ID, r)
			}
		}
	}
	e.must("POST", "/api/self-check/screens/no-such-task/check", map[string]any{"answer": "x"}, 404)

	after := counts()
	for i := range tables {
		if before[i] != after[i] {
			t.Errorf("%s changed from %d to %d rows: the self-check must store nothing", tables[i], before[i], after[i])
		}
	}

	// The screen tasks are not labs.
	e.must("GET", "/api/labs/self-check-screens", nil, 404)
	e.must("PUT", "/api/labs/self-check-screens/status", map[string]string{"status": "completed"}, 404)
	list := decode[struct {
		Labs []struct{ ID string } `json:"labs"`
	}](t, e.must("GET", "/api/labs", nil, 200))
	for _, l := range list.Labs {
		if !listedLab(l.ID) {
			t.Errorf("the lab list contains %s", l.ID)
		}
	}
}

// Two self-checks differ (fresh questions), and the screen tasks' answers come from the engine.
func TestSelfCheckIsDrawnAfreshAndEveryAnswerIsUnique(t *testing.T) {
	e := newLabEnv(t)
	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		for _, q := range decode[selfCheckBody](t, e.must("GET", "/api/self-check", nil, 200)).Questions {
			seen[q.ID] = true
		}
	}
	if len(seen) <= 12 {
		t.Errorf("six self-checks used only %d distinct questions", len(seen))
	}
	sc := decode[selfCheckBody](t, e.must("GET", "/api/self-check", nil, 200))
	for _, s := range sc.Screens {
		correct := 0
		for _, c := range s.Choices {
			if decode[struct{ Correct bool }](t, e.must("POST", "/api/self-check/screens/"+s.ID+"/check", map[string]any{"answer": c.ID}, 200)).Correct {
				correct++
			}
		}
		if correct != 1 {
			t.Errorf("screen task %s has %d correct choices", s.ID, correct)
		}
	}
}
