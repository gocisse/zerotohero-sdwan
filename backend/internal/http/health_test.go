package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantCode   int
		wantStatus string
		wantDB     string
	}{
		{"database up", nil, http.StatusOK, "ok", "ok"},
		{"database down", errors.New("connection refused"), http.StatusServiceUnavailable, "error", "error"},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewRouter(Deps{Health: fakePinger{err: tt.pingErr}, Log: log})
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", rec.Code, tt.wantCode)
			}
			var got struct {
				Status   string `json:"status"`
				Database struct {
					Status string `json:"status"`
				} `json:"database"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Status != tt.wantStatus || got.Database.Status != tt.wantDB {
				t.Errorf("got status=%q db=%q, want %q/%q", got.Status, got.Database.Status, tt.wantStatus, tt.wantDB)
			}
		})
	}
}
