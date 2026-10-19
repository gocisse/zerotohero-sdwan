// Package domain holds business types and logic, free of HTTP and SQL concerns.
package domain

// Status values reported by the health endpoint.
const (
	StatusOK    = "ok"
	StatusError = "error"
)

// Health is the aggregate service health report.
type Health struct {
	Status   string          `json:"status"`
	Service  string          `json:"service"`
	Database ComponentHealth `json:"database"`
}

// ComponentHealth is the health of one dependency.
type ComponentHealth struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// NewHealth derives the overall status from a database ping result.
func NewHealth(service string, dbErr error) Health {
	h := Health{Status: StatusOK, Service: service, Database: ComponentHealth{Status: StatusOK}}
	if dbErr != nil {
		h.Status = StatusError
		h.Database = ComponentHealth{Status: StatusError, Error: dbErr.Error()}
	}
	return h
}
