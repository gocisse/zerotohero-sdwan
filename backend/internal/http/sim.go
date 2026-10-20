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

type topologySummaryDTO struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Summary  string      `json:"summary"`
	Teaching bool        `json:"teaching"`
	Sites    int         `json:"sites"`
	Edges    int         `json:"edges"`
	Profiles int         `json:"profiles"`
	Warnings []sim.Issue `json:"warnings"`
}

// settingDTO describes one device setting the simulator knows, so the UI can offer valid what-if changes.
type settingDTO struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	Section      string    `json:"section"`
	Kind         string    `json:"kind"`
	SectionKnown bool      `json:"section_known"`
	HasDefault   bool      `json:"has_default"`
	Default      any       `json:"default,omitempty"`
	Basis        sim.Basis `json:"basis"`
}

type topologyDTO struct {
	Topology *sim.Topology `json:"topology"`
	Warnings []sim.Issue   `json:"warnings"`
	Settings []settingDTO  `json:"settings"`
	// UIBases is the evidence behind screen-level claims (tab names, transport groups, the build stub).
	UIBases map[string]sim.Basis `json:"ui_bases"`
	// Lab is set when the topology is a lab's faulted state: the conditions and assumption values the lab starts from.
	Lab *labStateDTO `json:"lab,omitempty"`
}

type labStateDTO struct {
	ID          string         `json:"id"`
	Conditions  sim.Conditions `json:"conditions"`
	Assumptions map[string]any `json:"assumptions"`
	Fixes       []string       `json:"fixes"`
	Flow        *sim.FlowSpec  `json:"flow,omitempty"`
}

func settingDTOs() []settingDTO {
	out := make([]settingDTO, 0, len(sim.Settings))
	for _, s := range sim.Settings {
		out = append(out, settingDTO{ID: s.ID, Label: s.Label, Section: s.Section, Kind: string(s.Kind), SectionKnown: s.SectionKnown,
			HasDefault: s.HasDefault, Default: s.Default, Basis: sim.RuleByID(sim.RSettingOverridable).Basis()})
	}
	return out
}

func summarize(t *content.Topology) topologySummaryDTO {
	edges := 0
	for _, s := range t.T.Sites {
		edges += len(s.Edges)
	}
	w := t.Warnings
	if w == nil {
		w = []sim.Issue{}
	}
	return topologySummaryDTO{ID: t.T.Meta.ID, Title: t.T.Meta.Title, Summary: t.T.Meta.Summary, Teaching: t.T.Meta.Teaching,
		Sites: len(t.T.Sites), Edges: edges, Profiles: len(t.T.Profiles), Warnings: w}
}

// TopologiesHandler lists the teaching topologies (GET /api/topologies).
func TopologiesHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, _ *http.Request, c *content.Course) {
		out := []topologySummaryDTO{}
		for _, t := range c.Topologies() {
			out = append(out, summarize(t))
		}
		writeJSON(w, http.StatusOK, map[string]any{"topologies": out})
	})
}

// TopologyHandler returns one authored topology (GET /api/topologies/{id}).
func TopologyHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		warns := t.Warnings
		if warns == nil {
			warns = []sim.Issue{}
		}
		dto := topologyDTO{Topology: t.T, Warnings: warns, Settings: settingDTOs(), UIBases: sim.UIBases()}
		if labID := r.URL.Query().Get("lab"); labID != "" {
			if lab, ok := c.Lab(labID); ok {
				fixes := []string{}
				if f := r.URL.Query().Get("fixes"); f != "" {
					fixes = strings.Split(f, ",")
				}
				as := lab.Lab.Fault.Assumptions
				if as == nil {
					as = map[string]any{}
				}
				dto.Lab = &labStateDTO{ID: labID, Conditions: lab.Lab.Fault.Conditions, Assumptions: as, Fixes: fixes, Flow: lab.Lab.Flow}
			}
		}
		writeJSON(w, http.StatusOK, dto)
	})
}

// EffectiveConfigHandler returns the effective configuration of one Edge
// (GET /api/topologies/{id}/edges/{edgeId}/effective-config).
func EffectiveConfigHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		edge := chi.URLParam(r, "edgeId")
		ec, err := sim.EffectiveConfig(t.T, edge)
		if err != nil {
			if errors.Is(err, sim.ErrNotFound) {
				notFound(w, "edge", edge)
				return
			}
			log.Error("effective config", "topology", id, "edge", edge, "err", err)
			writeJSON(w, http.StatusInternalServerError, apiError{Error: "simulator_error", Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, ec)
	})
}

// WhatIfHandler applies a hypothetical change and returns the diff (POST /api/topologies/{id}/what-if).
func WhatIfHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		id := chi.URLParam(r, "id")
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		var ch sim.Change
		if !decodeBody(w, r, &ch) {
			return
		}
		res, err := sim.WhatIf(t.T, ch)
		if err != nil {
			var ce *sim.ChangeError
			if errors.As(err, &ce) {
				writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_change", Message: ce.Message})
				return
			}
			log.Error("what-if", "topology", id, "err", err)
			writeJSON(w, http.StatusInternalServerError, apiError{Error: "simulator_error", Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}

// TunnelsHandler computes every tunnel under the conditions in the body (POST /api/topologies/{id}/tunnels).
// The body may be empty: no conditions. Nothing is stored.
func TunnelsHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		var cond sim.Conditions
		if !decodeOptionalBody(w, r, &cond) {
			return
		}
		ts, err := sim.ComputeTunnels(t.T, cond, sim.SystemClock())
		if !simResult(w, log, "tunnels", err) {
			return
		}
		writeJSON(w, http.StatusOK, ts)
	})
}

type timelineRequest struct {
	Conditions sim.Conditions `json:"conditions"`
	Events     []sim.Event    `json:"events"`
}

// TimelineHandler replays timed events (POST /api/topologies/{id}/timeline).
func TimelineHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		var req timelineRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if len(req.Events) > 200 {
			badRequest(w, "at most 200 events")
			return
		}
		res, err := sim.Timeline(t.T, req.Conditions, req.Events)
		if !simResult(w, log, "timeline", err) {
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}

type pathRequest struct {
	Flow       sim.Flow       `json:"flow"`
	Conditions sim.Conditions `json:"conditions"`
}

// PathsHandler decides the path of a flow (POST /api/topologies/{id}/paths).
func PathsHandler(src ContentSource, log *slog.Logger) http.HandlerFunc {
	return withCourse(src, log, func(w http.ResponseWriter, r *http.Request, c *content.Course) {
		t, ok := resolveTopology(w, r, c)
		if !ok {
			return
		}
		var req pathRequest
		if !decodeBody(w, r, &req) {
			return
		}
		d, err := sim.SelectPath(t.T, req.Conditions, req.Flow, sim.SystemClock())
		if !simResult(w, log, "paths", err) {
			return
		}
		writeJSON(w, http.StatusOK, d)
	})
}

// decodeOptionalBody decodes a JSON body, treating an empty body as the zero value.
func decodeOptionalBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	return decodeBody(w, r, v)
}

// simResult maps simulator input errors to 400 and anything else to 500. It returns true when err is nil.
func simResult(w http.ResponseWriter, log *slog.Logger, what string, err error) bool {
	if err == nil {
		return true
	}
	var ce *sim.ConditionError
	var fe *sim.FlowError
	var te *sim.TimelineError
	switch {
	case errors.As(err, &ce):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_conditions", Message: ce.Error()})
	case errors.As(err, &fe):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_flow", Message: fe.Message})
	case errors.As(err, &te):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid_events", Message: te.Message})
	default:
		log.Error("simulator", "what", what, "err", err)
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "simulator_error", Message: err.Error()})
	}
	return false
}
