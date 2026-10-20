package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/sim"
)

type routesRequest struct {
	Conditions  sim.Conditions       `json:"conditions"`
	Assumptions sim.TraceAssumptions `json:"assumptions"`
}

type reachabilityRequest struct {
	routesRequest
	Query sim.ReachQuery `json:"query"`
}

// defaultsRequest returns a request pre-filled with the documented assumption defaults, so a body that omits
// "assumptions" (or a field of it) gets the defaults.
func defaultsRequest() routesRequest {
	return routesRequest{Assumptions: sim.DefaultTraceAssumptions()}
}

// routingResult maps routing input errors to 400 and anything else to 500; it returns true when err is nil.
func routingResult(w http.ResponseWriter, log *slog.Logger, what string, err error) bool {
	if err == nil {
		return true
	}
	var qe *sim.QueryError
	var ce *sim.ConditionError
	switch {
	case errors.As(err, &qe):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_query", Message: qe.Message})
		return false
	case errors.As(err, &ce):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_conditions", Message: ce.Error()})
		return false
	}
	if sim.IsAssumptionError(err) {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_assumptions", Message: err.Error()})
		return false
	}
	return simResult(w, log, what, err)
}

// RoutesHandler computes every Edge's route tables (POST /api/topologies/{id}/routes). The body may be empty.
func RoutesHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		req := defaultsRequest()
		if !decodeOptionalBody(w, r, &req) {
			return
		}
		rs, err := sim.ComputeRoutes(t.T, req.Conditions, req.Assumptions.RoutingAssumptions, sim.SystemClock())
		if !routingResult(w, log, "routes", err) {
			return
		}
		writeJSON(w, http.StatusOK, rs)
	})
}

// ReachabilityHandler answers a reachability question (POST /api/topologies/{id}/reachability).
func ReachabilityHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		req := reachabilityRequest{routesRequest: defaultsRequest()}
		if !decodeBody(w, r, &req) {
			return
		}
		res, err := sim.Reachability(t.T, req.Conditions, req.Assumptions.RoutingAssumptions, req.Query, sim.SystemClock())
		if !routingResult(w, log, "reachability", err) {
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}

type traceRequest struct {
	routesRequest
	Flow sim.FlowSpec `json:"flow"`
}

// TraceHandler follows one flow end to end (POST /api/topologies/{id}/trace).
func TraceHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		req := traceRequest{routesRequest: defaultsRequest()}
		if !decodeBody(w, r, &req) {
			return
		}
		res, err := sim.TraceFlow(t.T, req.Conditions, req.Assumptions, req.Flow, sim.SystemClock())
		if !routingResult(w, log, "trace", err) {
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}

// resolveTopology finds the topology of a request. With ?lab=<id>&fixes=a,b the lab's faulted state is returned
// instead of the authored topology (patches applied to a copy; the file is never changed): every simulator screen and
// endpoint works unchanged on it, which is how a lab runner lets the learner inspect the faulty network.
func resolveTopology(w http.ResponseWriter, r *http.Request, c *content.Course) (*content.Topology, bool) {
	id := chi.URLParam(r, "id")
	t, ok := c.Topology(id)
	if !ok {
		notFound(w, "topology", id)
		return nil, false
	}
	labID := r.URL.Query().Get("lab")
	if labID == "" {
		return t, true
	}
	lab, ok := c.Lab(labID)
	if !ok {
		notFound(w, "lab", labID)
		return nil, false
	}
	if lab.Lab.Topology != id {
		badRequest(w, "lab "+labID+" runs on topology "+lab.Lab.Topology+", not "+id)
		return nil, false
	}
	var fixes []string
	if f := r.URL.Query().Get("fixes"); f != "" {
		fixes = strings.Split(f, ",")
	}
	st, err := lab.Lab.State(t.T, fixes)
	if err != nil {
		badRequest(w, "invalid lab state: "+err.Error())
		return nil, false
	}
	var warns []sim.Issue
	for _, is := range sim.Validate(st.Topo) {
		if is.Level == sim.LevelWarning {
			warns = append(warns, is)
		}
	}
	return &content.Topology{Path: t.Path, T: st.Topo, Warnings: warns}, true
}
