package http

import (
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/grading"
	"velocloud-academy/backend/internal/sim"
)

// The self-check (/sim/self-check) is ungraded and stores nothing: its questions are drawn from the quiz bank by topic and
// graded with the quiz grading code, but no attempt, no Leitner state and no mastery row is written. Its three "read this
// screen" tasks are the tasks of the lab file self-check-screens, which the lab endpoints never list or serve.

const (
	selfCheckQuestions = 12
	selfCheckLabID     = "self-check-screens"
	selfCheckPerSect   = 2
)

// listedLab reports whether a lab id is a lab learners see (the self-check screen tasks are not).
func listedLab(id string) bool { return !strings.HasPrefix(id, "self-check-") }

type selfQuestionDTO struct {
	questionDTO
	SectionID    string `json:"section_id"`
	SectionTitle string `json:"section_title"`
}

type selfScreenDTO struct {
	ID       string          `json:"id"`
	Prompt   string          `json:"prompt"`
	Choices  []sim.LabChoice `json:"choices"`
	Screen   labScreenDTO    `json:"screen"`
	Topology string          `json:"topology"`
}

type selfCheckDTO struct {
	Questions []selfQuestionDTO `json:"questions"`
	Screens   []selfScreenDTO   `json:"screens"`
	Lab       string            `json:"lab"`
	Symptom   string            `json:"symptom"`
}

// pickSelfCheck chooses up to n questions: round-robin over the sections, each section giving questions of topics not yet
// used, then the rest of the bank if some section has fewer topics than its share.
func pickSelfCheck(c *content.Course, n int, rng *rand.Rand) []selfQuestionDTO {
	type cand struct {
		q    content.Question
		sect *content.Section
	}
	pools := [][]cand{}
	for _, s := range c.Sections {
		var p []cand
		for _, q := range s.Quiz {
			p = append(p, cand{q, s})
		}
		rng.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
		if len(p) > 0 {
			pools = append(pools, p)
		}
	}
	used := map[string]bool{}
	topicUsed := map[string]bool{}
	var picked []cand
	take := func(sect int, distinctTopic bool) bool {
		for _, cd := range pools[sect] {
			if used[cd.q.ID] || (distinctTopic && topicUsed[cd.q.Topic]) {
				continue
			}
			used[cd.q.ID], topicUsed[cd.q.Topic] = true, true
			picked = append(picked, cd)
			return true
		}
		return false
	}
	perSect := make([]int, len(pools))
	for round := 0; round < selfCheckPerSect && len(picked) < n; round++ {
		for i := range pools {
			if len(picked) < n && take(i, true) {
				perSect[i]++
			}
		}
	}
	for len(picked) < n { // top up from the whole bank
		added := false
		for i := range pools {
			if len(picked) < n && take(i, false) {
				added = true
			}
		}
		if !added {
			break
		}
	}
	rng.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	out := make([]selfQuestionDTO, len(picked))
	for i, cd := range picked {
		out[i] = selfQuestionDTO{questionDTO: toQuestionDTO(cd.q), SectionID: cd.sect.Meta.ID, SectionTitle: cd.sect.Meta.Title}
	}
	return out
}

// SelfCheck builds a self-check (GET /api/self-check): fresh questions every time.
func SelfCheck(src ContentSource, d Deps) http.HandlerFunc {
	return withCourse(src, d.Log, func(w http.ResponseWriter, _ *http.Request, c *content.Course) {
		out := selfCheckDTO{Questions: pickSelfCheck(c, selfCheckQuestions, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))), Screens: []selfScreenDTO{}, Lab: selfCheckLabID}
		if lab, ok := c.Lab(selfCheckLabID); ok {
			out.Symptom = strings.TrimSpace(lab.Lab.Symptom)
			for i, t := range lab.Lab.Tasks {
				sc := labScreenDTO{}
				if i < len(lab.Lab.Screens) {
					sc = labScreenDTO{Label: lab.Lab.Screens[i].Label, Path: lab.Lab.Screens[i].Path}
				}
				out.Screens = append(out.Screens, selfScreenDTO{ID: t.ID, Prompt: t.Prompt, Choices: t.Choices, Screen: sc, Topology: lab.Lab.Topology})
			}
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// SelfCheckGrade grades answers with the quiz grading code and stores nothing (POST /api/self-check/grade).
func SelfCheckGrade(src ContentSource, d Deps) http.HandlerFunc {
	return withCourse(src, d.Log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		var req submitRequest
		if !decodeBody(w, r, &req) {
			return
		}
		idx := questionIndex(c)
		graded, err := grading.GradeSelected(func(id string) (content.Question, bool) { q, ok := idx[id]; return q, ok }, req.Answers)
		if err != nil {
			h := &progressHandlers{d: d}
			h.answersFailure(w, err)
			return
		}
		score := grading.Score(graded)
		writeJSON(w, http.StatusOK, map[string]any{"score": score, "total": len(graded), "results": toResults(graded)})
	})
}

// SelfCheckScreen checks one screen task without recording anything (POST /api/self-check/screens/{taskId}/check).
func SelfCheckScreen(src ContentSource, d Deps) http.HandlerFunc {
	return withCourse(src, d.Log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		taskID := chi.URLParam(r, "taskId")
		lab, ok := c.Lab(selfCheckLabID)
		if !ok {
			notFound(w, "lab", selfCheckLabID)
			return
		}
		if lab.Lab.TaskByID(taskID) == nil {
			notFound(w, "task", taskID)
			return
		}
		var req checkRequest
		if !decodeBody(w, r, &req) {
			return
		}
		topo, ok := c.Topology(lab.Lab.Topology)
		if !ok {
			notFound(w, "topology", lab.Lab.Topology)
			return
		}
		res, err := lab.Lab.Check(topo.T, taskID, req.Answer, nil)
		if err != nil {
			var pe *sim.PatchError
			if asPatchError(err, &pe) {
				badRequest(w, pe.Message)
				return
			}
			(&progressHandlers{d: d}).fail(w, "check self-check task", err)
			return
		}
		writeJSON(w, http.StatusOK, checkResponse{TaskID: taskID, Correct: res.Correct, Given: res.Given, Expected: res.Expected, Explanation: res.Explanation,
			Detail: res.Detail, Assumptions: assumptionNotes(res.Assumptions), Progress: toLabProgress(domain.LabProgress{}, false)})
	})
}

// listedLabOf finds a lab that learners can open; the self-check screen tasks are not one.
func listedLabOf(c *content.Course, id string) (*content.Lab, bool) {
	if !listedLab(id) {
		return nil, false
	}
	return c.Lab(id)
}
