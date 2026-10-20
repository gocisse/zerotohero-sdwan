package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/sim"
)

// Labs are practice scenarios. Nothing here ever locks anything: a failed check only returns feedback. The answers
// (the correct choice, the expected values, the explanations, the hints and the fix patches) never leave the server
// before the learner asks for them: tasks are sent without them, a check returns the explanation, hints are revealed
// one at a time.

type labGoalDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SectionID string `json:"section_id"`
}

type labProgressDTO struct {
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	HintsUsed     int        `json:"hints_used"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

func toLabProgress(lp domain.LabProgress, ok bool) labProgressDTO {
	if !ok {
		return labProgressDTO{Status: domain.LabNotStarted}
	}
	return labProgressDTO{Status: lp.Status, Attempts: lp.Attempts, HintsUsed: lp.HintsUsed, LastAttemptAt: lp.LastAttemptAt, CompletedAt: lp.CompletedAt}
}

type labSummaryDTO struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	Difficulty       int            `json:"difficulty"`
	EstimatedMinutes int            `json:"estimated_minutes"`
	Topology         string         `json:"topology"`
	Goals            []labGoalDTO   `json:"goals"`
	Tasks            int            `json:"tasks"`
	Progress         labProgressDTO `json:"progress"`
}

func goalsOf(c *content.Course, ids []string) []labGoalDTO {
	out := []labGoalDTO{}
	for _, id := range ids {
		if l, ok := c.Lesson(id); ok {
			out = append(out, labGoalDTO{ID: id, Title: l.Meta.Title, SectionID: l.SectionID})
		}
	}
	return out
}

func summarizeLab(c *content.Course, l *sim.Lab, p map[string]domain.LabProgress) labSummaryDTO {
	lp, ok := p[l.ID]
	return labSummaryDTO{ID: l.ID, Title: l.Title, Summary: l.Summary, Difficulty: l.Difficulty, EstimatedMinutes: l.EstimatedMinutes, Topology: l.Topology,
		Goals: goalsOf(c, l.Goals), Tasks: len(l.Tasks), Progress: toLabProgress(lp, ok)}
}

type labHandlers struct{ progressHandlers }

// ListLabs returns every lab with the learner's status (GET /api/labs).
func (h *labHandlers) list(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		p, err := h.d.Progress.LabProgressAll(r.Context())
		if err != nil {
			h.fail(w, "load lab progress", err)
			return
		}
		out := []labSummaryDTO{}
		for _, l := range c.Labs() {
			if listedLab(l.Lab.ID) {
				out = append(out, summarizeLab(c, l.Lab, p))
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"labs": out})
	})(w, r)
}

type labTaskDTO struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Prompt  string          `json:"prompt"`
	Choices []sim.LabChoice `json:"choices"`
}

type labFixDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type labScreenDTO struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type labDetailDTO struct {
	labSummaryDTO
	Symptom string         `json:"symptom"`
	Screens []labScreenDTO `json:"screens"`
	Flow    *sim.FlowSpec  `json:"flow,omitempty"`
	RelyOn  []int          `json:"relies_on_assumptions"`
	Tasks   []labTaskDTO   `json:"task_list"`
	Fixes   []labFixDTO    `json:"fixes"`
	Hints   int            `json:"hint_count"`
}

// get returns a lab without its answers (GET /api/labs/{id}).
func (h *labHandlers) get(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		lab, ok := listedLabOf(c, id)
		if !ok {
			notFound(w, "lab", id)
			return
		}
		p, err := h.d.Progress.LabProgressAll(r.Context())
		if err != nil {
			h.fail(w, "load lab progress", err)
			return
		}
		l := lab.Lab
		d := labDetailDTO{labSummaryDTO: summarizeLab(c, l, p), Symptom: l.Symptom, Flow: l.Flow, RelyOn: append([]int{}, l.RelyOn...), Hints: len(l.Hints),
			Screens: []labScreenDTO{}, Tasks: []labTaskDTO{}, Fixes: []labFixDTO{}}
		for _, s := range l.Screens {
			d.Screens = append(d.Screens, labScreenDTO{Label: s.Label, Path: s.Path})
		}
		for _, t := range l.Tasks {
			ch := t.Choices
			if ch == nil {
				ch = []sim.LabChoice{}
			}
			d.Tasks = append(d.Tasks, labTaskDTO{ID: t.ID, Type: t.Type, Prompt: t.Prompt, Choices: ch})
		}
		for _, f := range l.Fixes {
			d.Fixes = append(d.Fixes, labFixDTO{ID: f.ID, Title: f.Title, Description: f.Description})
		}
		writeJSON(w, http.StatusOK, d)
	})(w, r)
}

type checkRequest struct {
	Answer string   `json:"answer"`
	Fixes  []string `json:"fixes"`
}

type assumptionNote struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type checkResponse struct {
	TaskID      string           `json:"task_id"`
	Correct     bool             `json:"correct"`
	Given       string           `json:"given"`
	Expected    string           `json:"expected"`
	Explanation string           `json:"explanation"`
	Detail      string           `json:"detail"`
	Assumptions []assumptionNote `json:"assumptions"`
	Progress    labProgressDTO   `json:"progress"`
}

func assumptionNotes(nums []int) []assumptionNote {
	out := []assumptionNote{}
	for _, n := range nums {
		for _, r := range sim.Rules {
			if r.Status == sim.StatusAssumption && r.Assumption == n {
				out = append(out, assumptionNote{Number: n, Text: r.Text})
			}
		}
	}
	return out
}

// check grades one task on the server (POST /api/labs/{id}/tasks/{taskId}/check). Nothing is ever blocked.
func (h *labHandlers) check(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id, taskID := chi.URLParam(r, "id"), chi.URLParam(r, "taskId")
		lab, ok := listedLabOf(c, id)
		if !ok {
			notFound(w, "lab", id)
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
		res, err := lab.Lab.Check(topo.T, taskID, req.Answer, req.Fixes)
		if err != nil {
			var pe *sim.PatchError
			if asPatchError(err, &pe) {
				badRequest(w, pe.Message)
				return
			}
			h.fail(w, "check task", err)
			return
		}
		lp, err := h.d.Progress.RecordLabAttempt(r.Context(), id, h.d.Now())
		if err != nil {
			h.fail(w, "record lab attempt", err)
			return
		}
		writeJSON(w, http.StatusOK, checkResponse{TaskID: taskID, Correct: res.Correct, Given: res.Given, Expected: res.Expected, Explanation: res.Explanation,
			Detail: res.Detail, Assumptions: assumptionNotes(res.Assumptions), Progress: toLabProgress(lp, true)})
	})(w, r)
}

type fixRequest struct {
	FixID   string   `json:"fix_id"`
	Applied []string `json:"applied"`
}

type fixResponse struct {
	Applied []string         `json:"applied"`
	Trace   *sim.TraceResult `json:"trace,omitempty"`
}

// fix applies one allowed patch on top of the fixes already applied and returns the recomputed trace of the lab's
// flow (POST /api/labs/{id}/fix). The authored topology file is never changed.
func (h *labHandlers) fix(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		lab, ok := listedLabOf(c, id)
		if !ok {
			notFound(w, "lab", id)
			return
		}
		var req fixRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if !lab.Lab.HasFix(req.FixID) {
			badRequest(w, "unknown fix "+req.FixID)
			return
		}
		applied := append([]string{}, req.Applied...)
		dup := false
		for _, a := range applied {
			dup = dup || a == req.FixID
		}
		if !dup {
			applied = append(applied, req.FixID)
		}
		topo, ok := c.Topology(lab.Lab.Topology)
		if !ok {
			notFound(w, "topology", lab.Lab.Topology)
			return
		}
		st, err := lab.Lab.State(topo.T, applied)
		if err != nil {
			badRequest(w, err.Error())
			return
		}
		resp := fixResponse{Applied: applied}
		if lab.Lab.Flow != nil {
			tr, err := sim.TraceFlow(st.Topo, st.Cond, st.As, *lab.Lab.Flow, sim.SystemClock())
			if err != nil {
				h.fail(w, "trace after fix", err)
				return
			}
			resp.Trace = tr
		}
		writeJSON(w, http.StatusOK, resp)
	})(w, r)
}

type labStatusRequest struct {
	Status string `json:"status"`
}

// status records started, completed or skipped (PUT /api/labs/{id}/status). Always allowed.
func (h *labHandlers) status(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		if _, ok := listedLabOf(c, id); !ok {
			notFound(w, "lab", id)
			return
		}
		var req labStatusRequest
		if !decodeBody(w, r, &req) {
			return
		}
		switch req.Status {
		case domain.LabStarted, domain.LabCompleted, domain.LabSkipped:
		default:
			badRequest(w, `status must be "started", "completed" or "skipped"`)
			return
		}
		lp, err := h.d.Progress.SetLabStatus(r.Context(), id, req.Status, h.d.Now())
		if err != nil {
			h.fail(w, "save lab status", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lab_id": id, "progress": toLabProgress(lp, true)})
	})(w, r)
}

// hint reveals hint n (1-based) once the previous ones were revealed (POST /api/labs/{id}/hints/{n}).
func (h *labHandlers) hint(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		lab, ok := listedLabOf(c, id)
		if !ok {
			notFound(w, "lab", id)
			return
		}
		n, err := strconv.Atoi(chi.URLParam(r, "n"))
		hints := lab.Lab.Hints
		if err != nil || n < 1 || n > len(hints) {
			badRequest(w, "hint number must be between 1 and "+strconv.Itoa(len(hints)))
			return
		}
		all, err := h.d.Progress.LabProgressAll(r.Context())
		if err != nil {
			h.fail(w, "load lab progress", err)
			return
		}
		if used := all[id].HintsUsed; n > used+1 {
			badRequest(w, "reveal the hints in order: the next one is number "+strconv.Itoa(used+1))
			return
		}
		lp, err := h.d.Progress.UseLabHint(r.Context(), id, n, h.d.Now())
		if err != nil {
			h.fail(w, "record hint", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"index": n, "total": len(hints), "task": hints[n-1].Task, "text": strings.TrimSpace(hints[n-1].Text), "progress": toLabProgress(lp, true)})
	})(w, r)
}

// debrief returns the lab's debrief (GET /api/labs/{id}/debrief). It names the root causes, so it is sent only once the
// lab is completed or skipped; before that the response says it is not available yet. Nothing is locked either way: the
// learner can always mark the lab skipped to read it.
func (h *labHandlers) debrief(w http.ResponseWriter, r *http.Request) {
	h.withCourse(func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		lab, ok := listedLabOf(c, id)
		if !ok {
			notFound(w, "lab", id)
			return
		}
		all, err := h.d.Progress.LabProgressAll(r.Context())
		if err != nil {
			h.fail(w, "load lab progress", err)
			return
		}
		items := []sim.LabDebrief{}
		available := false
		if st := all[id].Status; st == domain.LabCompleted || st == domain.LabSkipped {
			available = true
			items = append(items, lab.Lab.Debrief...)
		}
		writeJSON(w, http.StatusOK, map[string]any{"lab_id": id, "available": available, "has_debrief": len(lab.Lab.Debrief) > 0, "items": items})
	})(w, r)
}

// labsBlock is the informational labs summary of GET /api/progress. Orphaned lab ids are ignored.
type labsBlockDTO struct {
	Total     int                `json:"total"`
	Started   int                `json:"started"`
	Completed int                `json:"completed"`
	Skipped   int                `json:"skipped"`
	Items     []labsBlockItemDTO `json:"items"`
}

type labsBlockItemDTO struct {
	LabID     string `json:"lab_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
	HintsUsed int    `json:"hints_used"`
}

func (h *progressHandlers) labsBlock(r *http.Request, c *content.Course) (labsBlockDTO, error) {
	p, err := h.d.Progress.LabProgressAll(r.Context())
	if err != nil {
		return labsBlockDTO{}, err
	}
	b := labsBlockDTO{Items: []labsBlockItemDTO{}}
	for _, l := range c.Labs() {
		if !listedLab(l.Lab.ID) {
			continue
		}
		b.Total++
		item := labsBlockItemDTO{LabID: l.Lab.ID, Title: l.Lab.Title, Status: domain.LabNotStarted}
		if lp, ok := p[l.Lab.ID]; ok {
			item.Status, item.Attempts, item.HintsUsed = lp.Status, lp.Attempts, lp.HintsUsed
		}
		switch item.Status {
		case domain.LabStarted:
			b.Started++
		case domain.LabCompleted:
			b.Completed++
		case domain.LabSkipped:
			b.Skipped++
		}
		b.Items = append(b.Items, item)
	}
	return b, nil
}

func asPatchError(err error, target **sim.PatchError) bool {
	pe, ok := err.(*sim.PatchError)
	if ok {
		*target = pe
	}
	return ok
}
