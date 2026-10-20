package http

import (
	"bytes"
	"context"
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

	"go.yaml.in/yaml/v3"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/sim"
	"velocloud-academy/backend/internal/store"
)

// Lab tests run against the real content and a real Postgres.
func newLabEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, pool: newTestPool(t), now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := store.New(e.pool)
	e.srv = NewRouter(Deps{Health: st, Content: content.NewProvider("../../../content", log), Progress: st, Log: log, Now: func() time.Time { return e.now }})
	return e
}

func readLab(t *testing.T, id string) *sim.Lab {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../../content/labs", id+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var l sim.Lab
	if err := yaml.Unmarshal(data, &l); err != nil {
		t.Fatal(err)
	}
	return &l
}

func labFiles(t *testing.T) []string {
	files, _ := filepath.Glob("../../../content/labs/*.yaml")
	var ids []string
	for _, f := range files {
		if id := strings.TrimSuffix(filepath.Base(f), ".yaml"); listedLab(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) < 9 {
		t.Fatalf("want at least 6 labs, got %v", ids)
	}
	return ids
}

type labDetail struct {
	ID       string `json:"id"`
	Symptom  string `json:"symptom"`
	Hints    int    `json:"hint_count"`
	Progress struct {
		Status    string `json:"status"`
		Attempts  int    `json:"attempts"`
		HintsUsed int    `json:"hints_used"`
	} `json:"progress"`
	Tasks []struct {
		ID      string                      `json:"id"`
		Type    string                      `json:"type"`
		Choices []struct{ ID, Text string } `json:"choices"`
	} `json:"task_list"`
	Fixes []struct{ ID string } `json:"fixes"`
}

func TestLabListAndProgressBlock(t *testing.T) {
	e := newLabEnv(t)
	list := decode[struct {
		Labs []struct {
			ID       string `json:"id"`
			Goals    []struct{ ID, Title string }
			Tasks    int                     `json:"tasks"`
			Progress struct{ Status string } `json:"progress"`
		} `json:"labs"`
	}](t, e.must("GET", "/api/labs", nil, 200))
	if len(list.Labs) != len(labFiles(t)) {
		t.Fatalf("labs = %d", len(list.Labs))
	}
	for _, l := range list.Labs {
		if l.Progress.Status != "not_started" || len(l.Goals) == 0 || l.Goals[0].Title == "" || l.Tasks == 0 {
			t.Errorf("%+v", l)
		}
	}
	prog := decode[struct {
		Labs struct{ Total, Started, Completed, Skipped int } `json:"labs"`
	}](t, e.must("GET", "/api/progress", nil, 200))
	if prog.Labs.Total != len(list.Labs) || prog.Labs.Completed != 0 {
		t.Errorf("%+v", prog.Labs)
	}
}

// No lab endpoint reveals an answer before the learner asks for it.
func TestLabEndpointsNeverLeakAnswers(t *testing.T) {
	e := newLabEnv(t)
	bodies := [][]byte{e.must("GET", "/api/labs", nil, 200)}
	for _, id := range labFiles(t) {
		bodies = append(bodies, e.must("GET", "/api/labs/"+id, nil, 200))
		// The debrief names the root causes: nothing before the lab is completed or skipped.
		bodies = append(bodies, e.must("GET", "/api/labs/"+id+"/debrief", nil, 200))
	}
	var secrets []string
	for _, id := range labFiles(t) {
		l := readLab(t, id)
		for _, tk := range l.Tasks {
			secrets = append(secrets, tk.Explanation)
		}
		for _, h := range l.Hints {
			secrets = append(secrets, strings.TrimSpace(h.Text))
		}
		for _, d := range l.Debrief {
			secrets = append(secrets, d.Finding)
		}
		for _, f := range l.Fixes {
			for _, p := range f.Patches {
				secrets = append(secrets, p.Path)
			}
		}
		for _, p := range l.Fault.Patches {
			secrets = append(secrets, p.Path)
		}
	}
	for _, b := range bodies {
		s := string(b)
		for _, k := range []string{`"probe"`, `"explanation"`, `"equals"`, `"patches"`, `"value"`, `"correct"`} {
			if strings.Contains(s, k) {
				t.Errorf("a lab response carries the key %s: %.200s", k, s)
			}
		}
		for _, sec := range secrets {
			if sec != "" && strings.Contains(s, sec) {
				t.Errorf("a lab response leaks %q", sec)
			}
		}
	}
}

func TestLabCheckHintsFixAndStatus(t *testing.T) {
	e := newLabEnv(t)
	const id = "branch-b-cannot-reach"
	d := decode[labDetail](t, e.must("GET", "/api/labs/"+id, nil, 200))
	if d.Hints == 0 || len(d.Tasks) != 3 || len(d.Fixes) == 0 {
		t.Fatalf("%+v", d)
	}
	check := func(task, answer string, fixes []string) map[string]any {
		return decode[map[string]any](t, e.must("POST", "/api/labs/"+id+"/tasks/"+task+"/check", map[string]any{"answer": answer, "fixes": fixes}, 200))
	}
	// A wrong answer only gives feedback, and names the right one.
	r := check("stage", "known", nil)
	if r["correct"] != false || r["expected"] == "" || r["explanation"] == "" {
		t.Errorf("%+v", r)
	}
	if check("stage", "redistributed", nil)["correct"] != true {
		t.Error("the right answer is not accepted")
	}
	// The fix task fails before any fix and passes after the right one; the wrong fixes do not pass.
	if check("fix-it", "", nil)["correct"] != false {
		t.Error("the fix check must fail on the faulty network")
	}
	wrong := decode[fixResponse](t, e.must("POST", "/api/labs/"+id+"/fix", map[string]any{"fix_id": "external-flag", "applied": []string{}}, 200))
	if wrong.Trace == nil || wrong.Trace.Verdict == sim.VerdictDelivered || len(wrong.Applied) != 1 {
		t.Errorf("a wrong fix must not deliver: %+v", wrong.Trace)
	}
	if check("fix-it", "", []string{"external-flag"})["correct"] != false {
		t.Error("a wrong fix passed the check")
	}
	good := decode[fixResponse](t, e.must("POST", "/api/labs/"+id+"/fix", map[string]any{"fix_id": "overlay-prefix", "applied": wrong.Applied}, 200))
	if good.Trace == nil || good.Trace.Verdict != sim.VerdictDelivered || len(good.Applied) != 2 {
		t.Errorf("the right fix must deliver: %+v", good)
	}
	if check("fix-it", "", good.Applied)["correct"] != true {
		t.Error("the right fix did not pass")
	}
	// Errors.
	e.must("POST", "/api/labs/"+id+"/fix", map[string]any{"fix_id": "nope"}, 400)
	e.must("POST", "/api/labs/"+id+"/tasks/ghost/check", map[string]any{"answer": "x"}, 404)
	e.must("POST", "/api/labs/"+id+"/tasks/stage/check", map[string]any{"answer": "ghost"}, 400)
	e.must("POST", "/api/labs/ghost/tasks/stage/check", map[string]any{"answer": "x"}, 404)
	e.must("POST", "/api/labs/"+id+"/tasks/stage/check", nil, 400)
	// Hints come in order and are recorded.
	e.must("POST", "/api/labs/"+id+"/hints/2", nil, 400)
	h1 := decode[map[string]any](t, e.must("POST", "/api/labs/"+id+"/hints/1", nil, 200))
	if h1["text"] == "" {
		t.Error("hint 1 is empty")
	}
	e.must("POST", "/api/labs/"+id+"/hints/2", nil, 200)
	e.must("POST", "/api/labs/"+id+"/hints/99", nil, 400)
	// Progress: the attempts and hints are counted, the lab started.
	p := decode[labDetail](t, e.must("GET", "/api/labs/"+id, nil, 200)).Progress
	if p.Status != "started" || p.Attempts != 5 || p.HintsUsed != 2 {
		t.Errorf("%+v", p)
	}
	if e.count(`SELECT count(*) FROM lab_progress`) != 1 {
		t.Error("one row expected")
	}
	// Status: always allowed; completed is sticky.
	e.must("PUT", "/api/labs/"+id+"/status", map[string]string{"status": "completed"}, 200)
	e.must("PUT", "/api/labs/"+id+"/status", map[string]string{"status": "skipped"}, 200)
	e.must("PUT", "/api/labs/"+id+"/status", map[string]string{"status": "explode"}, 400)
	e.must("PUT", "/api/labs/ghost/status", map[string]string{"status": "started"}, 404)
	if st := decode[labDetail](t, e.must("GET", "/api/labs/"+id, nil, 200)).Progress.Status; st != "completed" {
		t.Errorf("completed must not be downgraded, got %s", st)
	}
}

func TestLabsNeverLockAnythingAndOrphansAreIgnored(t *testing.T) {
	e := newLabEnv(t)
	ids := labFiles(t)
	// Skip one, fail every check of another: nothing else changes.
	e.must("PUT", "/api/labs/"+ids[0]+"/status", map[string]string{"status": "skipped"}, 200)
	d := decode[labDetail](t, e.must("GET", "/api/labs/"+ids[1], nil, 200))
	for _, tk := range d.Tasks {
		if tk.Type != "fix" && len(tk.Choices) > 0 {
			e.must("POST", "/api/labs/"+ids[1]+"/tasks/"+tk.ID+"/check", map[string]any{"answer": tk.Choices[0].ID}, 200)
		}
	}
	for _, path := range []string{"/api/course", "/api/progress", "/api/labs", "/api/labs/" + ids[0], "/api/labs/" + ids[1], "/api/topologies"} {
		e.must("GET", path, nil, 200)
	}
	// A row for a lab that no longer exists is ignored everywhere and never deleted.
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO lab_progress (lab_id, status) VALUES ('retired-lab', 'completed')`); err != nil {
		t.Fatal(err)
	}
	prog := decode[struct {
		Labs struct {
			Total, Completed int
			Items            []struct {
				LabID string `json:"lab_id"`
			}
		}
	}](t, e.must("GET", "/api/progress", nil, 200))
	if prog.Labs.Completed != 0 || prog.Labs.Total != len(ids) {
		t.Errorf("%+v", prog.Labs)
	}
	for _, it := range prog.Labs.Items {
		if it.LabID == "retired-lab" {
			t.Error("an orphaned lab id must be ignored")
		}
	}
	if e.count(`SELECT count(*) FROM lab_progress WHERE lab_id = 'retired-lab'`) != 1 {
		t.Error("orphaned rows must never be deleted")
	}
}

// The faulted state is served through the ordinary topology endpoints, without touching the authored file.
func TestLabStateIsServedAsATopologyAndNeverChangesTheAuthoredOne(t *testing.T) {
	e := newLabEnv(t)
	const id = "branch-b-cannot-reach"
	authored := decode[struct{ Topology sim.Topology }](t, e.must("GET", "/api/topologies/routing-demo", nil, 200)).Topology
	faulty := decode[struct {
		Topology sim.Topology
		Lab      struct {
			ID    string
			Fixes []string
		}
	}](t, e.must("GET", "/api/topologies/routing-demo?lab="+id, nil, 200))
	if faulty.Lab.ID != id {
		t.Errorf("lab block = %+v", faulty.Lab)
	}
	rd := func(tp sim.Topology) *bool {
		for _, s := range tp.Sites {
			for _, ed := range s.Edges {
				if ed.ID == "b-edge" && ed.Routing != nil && ed.Routing.BGP != nil && ed.Routing.BGP.Redistribute != nil {
					return ed.Routing.BGP.Redistribute.OverlayPrefix
				}
			}
		}
		return nil
	}
	if rd(authored) != nil {
		t.Error("the authored topology must not set Overlay Prefix")
	}
	if v := rd(faulty.Topology); v == nil || *v {
		t.Error("the lab's faulted topology sets Overlay Prefix off explicitly")
	}
	fixed := decode[struct{ Topology sim.Topology }](t, e.must("GET", "/api/topologies/routing-demo?lab="+id+"&fixes=overlay-prefix", nil, 200))
	if v := rd(fixed.Topology); v == nil || !*v {
		t.Error("the fix turns Overlay Prefix on")
	}
	// The routing endpoints run on the faulted state.
	rr := decode[struct {
		Outcome string `json:"outcome"`
	}](t, post(t, e, "/api/topologies/routing-demo/reachability?lab="+id, `{"query":{"source":"b-edge","destination":"10.20.0.0/24"}}`))
	if rr.Outcome != "unreachable" {
		t.Errorf("faulty outcome = %s", rr.Outcome)
	}
	rr = decode[struct {
		Outcome string `json:"outcome"`
	}](t, post(t, e, "/api/topologies/routing-demo/reachability?lab="+id+"&fixes=overlay-prefix", `{"query":{"source":"b-edge","destination":"10.20.0.0/24"}}`))
	if rr.Outcome != "reachable" {
		t.Errorf("fixed outcome = %s", rr.Outcome)
	}
	e.must("GET", "/api/topologies/three-site?lab="+id, nil, 400)
	e.must("GET", "/api/topologies/routing-demo?lab=ghost", nil, 404)
	e.must("GET", "/api/topologies/routing-demo?lab="+id+"&fixes=ghost", nil, 400)
}

func post(t *testing.T, e *env, path, body string) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	e.srv.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("POST %s = %d: %s", path, rec.Code, rec.Body.String())
	}
	_ = json.Valid
	return rec.Body.Bytes()
}

// The debrief is served once the lab is completed or skipped, never before; nothing else is locked either way.
func TestLabDebriefIsServedOnlyAfterTheLabIsFinished(t *testing.T) {
	e := newLabEnv(t)
	type deb struct {
		Available bool `json:"available"`
		Has       bool `json:"has_debrief"`
		Items     []struct {
			Finding    string `json:"finding"`
			Where      string `json:"where"`
			Unverified bool   `json:"unverified"`
			Note       string `json:"note"`
		} `json:"items"`
	}
	id := "capstone-1-monday-morning"
	want := len(readLab(t, id).Debrief)
	if want == 0 {
		t.Fatal("the capstone labs end with a debrief")
	}
	before := decode[deb](t, e.must("GET", "/api/labs/"+id+"/debrief", nil, 200))
	if before.Available || len(before.Items) != 0 || !before.Has {
		t.Errorf("before the lab is finished: %+v", before)
	}
	e.must("PUT", "/api/labs/"+id+"/status", map[string]string{"status": "started"}, 200)
	if d := decode[deb](t, e.must("GET", "/api/labs/"+id+"/debrief", nil, 200)); d.Available {
		t.Error("a started lab must not reveal its debrief")
	}
	e.must("PUT", "/api/labs/"+id+"/status", map[string]string{"status": "skipped"}, 200)
	after := decode[deb](t, e.must("GET", "/api/labs/"+id+"/debrief", nil, 200))
	if !after.Available || len(after.Items) != want {
		t.Errorf("after skipping: %+v", after)
	}
	unverified := 0
	for _, it := range after.Items {
		if it.Finding == "" || it.Where == "" {
			t.Errorf("%+v", it)
		}
		if it.Unverified {
			unverified++
			if it.Note == "" {
				t.Errorf("an unverified location carries its note: %+v", it)
			}
		}
	}
	if unverified == 0 {
		t.Error("the capstone debrief marks at least one location the guides do not confirm")
	}
	e.must("GET", "/api/labs/no-such-lab/debrief", nil, 404)
}
