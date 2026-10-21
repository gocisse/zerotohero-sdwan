// Package http wires the chi router and HTTP handlers.
package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps are the collaborators handlers need. Now is injectable so tests control the clock.
type Deps struct {
	Health   Pinger
	Content  ContentSource
	Progress ProgressStore
	Now      func() time.Time
	Log      *slog.Logger
}

// NewRouter builds the API router. The frontend reaches it through the Vite
// dev proxy, so no CORS configuration is needed.
func NewRouter(d Deps) http.Handler {
	if d.Now == nil {
		d.Now = func() time.Time { return time.Now().UTC() }
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, requestLogger(d.Log))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", HealthHandler(d.Health, d.Log))

		r.Get("/course", CourseHandler(d.Content, d.Log))
		r.Get("/lessons/{id}", LessonHandler(d.Content, d.Log))
		r.Get("/sections/{id}/quiz", QuizHandler(d.Content, d.Log))
		r.Get("/diagrams/{id}", DiagramHandler(d.Content, d.Log))
		r.Get("/glossary", GlossaryHandler(d.Content, d.Log))
		r.Get("/runbooks", RunbooksHandler(d.Content, d.Log))
		r.Get("/runbooks/{id}", RunbookHandler(d.Content, d.Log))

		r.Get("/topologies", TopologiesHandler(d.Content, d.Log))
		r.Get("/topologies/{id}", TopologyHandler(d.Content, d.Log))
		r.Get("/topologies/{id}/edges/{edgeId}/effective-config", EffectiveConfigHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/what-if", WhatIfHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/tunnels", TunnelsHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/timeline", TimelineHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/paths", PathsHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/routes", RoutesHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/reachability", ReachabilityHandler(d.Content, d.Log))
		r.Post("/topologies/{id}/trace", TraceHandler(d.Content, d.Log))

		r.Get("/self-check", SelfCheck(d.Content, d))
		r.Post("/self-check/grade", SelfCheckGrade(d.Content, d))
		r.Post("/self-check/screens/{taskId}/check", SelfCheckScreen(d.Content, d))

		h := &progressHandlers{d: d}
		r.Post("/sections/{id}/quiz/attempts", h.submitAttempt)
		r.Get("/sections/{id}/quiz/attempts", h.listAttempts)
		r.Post("/sections/{id}/quiz/skip", h.skipQuiz)
		r.Put("/lessons/{id}/progress", h.putLessonProgress)
		r.Get("/progress", h.progress)
		r.Get("/progress/topics", h.topics)
		r.Get("/study-guide", h.studyGuide)
		r.Get("/activity", h.activity)
		r.Get("/review/queue", h.reviewQueue)
		r.Post("/review/answers", h.reviewAnswers)

		lh := &labHandlers{progressHandlers{d: d}}
		r.Get("/labs", lh.list)
		r.Get("/labs/{id}", lh.get)
		r.Post("/labs/{id}/tasks/{taskId}/check", lh.check)
		r.Post("/labs/{id}/fix", lh.fix)
		r.Put("/labs/{id}/status", lh.status)
		r.Post("/labs/{id}/hints/{n}", lh.hint)
		r.Get("/labs/{id}/debrief", lh.debrief)
	})
	return r
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("request", "method", r.Method, "path", r.URL.Path,
				"status", ww.Status(), "dur_ms", time.Since(start).Milliseconds())
		})
	}
}
