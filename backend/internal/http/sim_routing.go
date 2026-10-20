package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/sim"
)

type routesRequest struct {
	Conditions  sim.Conditions         `json:"conditions"`
	Assumptions sim.RoutingAssumptions `json:"assumptions"`
}

type reachabilityRequest struct {
	routesRequest
	Query sim.ReachQuery `json:"query"`
}

// defaultsRequest returns a request pre-filled with the documented assumption defaults, so a body that omits
// "assumptions" (or a field of it) gets the defaults.
func defaultsRequest() routesRequest {
	return routesRequest{Assumptions: sim.DefaultRoutingAssumptions()}
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
		t, ok := c.Topology(chi.URLParam(r, "id"))
		if !ok {
			notFound(w, "topology", chi.URLParam(r, "id"))
			return
		}
		req := defaultsRequest()
		if !decodeOptionalBody(w, r, &req) {
			return
		}
		rs, err := sim.ComputeRoutes(t.T, req.Conditions, req.Assumptions, sim.SystemClock())
		if !routingResult(w, log, "routes", err) {
			return
		}
		writeJSON(w, http.StatusOK, rs)
	})
}

// ReachabilityHandler answers a reachability question (POST /api/topologies/{id}/reachability).
func ReachabilityHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := c.Topology(chi.URLParam(r, "id"))
		if !ok {
			notFound(w, "topology", chi.URLParam(r, "id"))
			return
		}
		req := reachabilityRequest{routesRequest: defaultsRequest()}
		if !decodeBody(w, r, &req) {
			return
		}
		res, err := sim.Reachability(t.T, req.Conditions, req.Assumptions, req.Query, sim.SystemClock())
		if !routingResult(w, log, "reachability", err) {
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}
