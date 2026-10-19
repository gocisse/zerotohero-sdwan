package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"velocloud-academy/backend/internal/domain"
)

// Pinger is the slice of the store the health handler depends on.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler reports service and database status. It answers 503 when the
// database is unreachable so that probes can act on the status code alone.
func HealthHandler(p Pinger, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		h := domain.NewHealth("velocloud-academy-backend", p.Ping(ctx))
		if h.Database.Error != "" {
			log.Warn("health: database ping failed", "err", h.Database.Error)
		}
		code := http.StatusOK
		if h.Status != domain.StatusOK {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, h)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
