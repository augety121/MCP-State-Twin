package server

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// AdmissionHandler rejects excess concurrent HTTP requests without creating
// an unbounded in-process queue. Data and control planes must use independent
// instances so one plane cannot consume the other's capacity.
type AdmissionHandler struct {
	next     http.Handler
	permits  chan struct{}
	active   atomic.Int64
	rejected atomic.Uint64
}

type AdmissionStats struct {
	Limit    int    `json:"limit"`
	Active   int64  `json:"active"`
	Rejected uint64 `json:"rejected"`
}

func NewAdmissionHandler(next http.Handler, limit int) (*AdmissionHandler, error) {
	if next == nil {
		return nil, &AdmissionConfigError{Message: "admission handler is required"}
	}
	if limit < 1 || limit > 1024 {
		return nil, &AdmissionConfigError{Message: "admission limit must be within 1..1024"}
	}
	return &AdmissionHandler{next: next, permits: make(chan struct{}, limit)}, nil
}

type AdmissionConfigError struct{ Message string }

func (e *AdmissionConfigError) Error() string { return e.Message }

func (a *AdmissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case a.permits <- struct{}{}:
		a.active.Add(1)
		defer func() {
			a.active.Add(-1)
			<-a.permits
		}()
		a.next.ServeHTTP(w, r)
	default:
		a.rejected.Add(1)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code": "SERVER_BUSY", "message": "server is at operational capacity",
			},
		})
	}
}

func (a *AdmissionHandler) Stats() AdmissionStats {
	return AdmissionStats{Limit: cap(a.permits), Active: a.active.Load(), Rejected: a.rejected.Load()}
}
