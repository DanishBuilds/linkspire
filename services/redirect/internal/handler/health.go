// services/redirect/internal/handler/health.go
package handler

import (
	"context"
	"net/http"
	"time"
)

// Pinger is any dependency that can report readiness.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NamedCheck associates a readiness probe with a name.
type NamedCheck struct {
	Name  string
	Check Pinger
}

// Health serves liveness and readiness probes.
type Health struct {
	checks []NamedCheck
}

// NewHealth builds a Health handler from the given dependency checks.
func NewHealth(checks ...NamedCheck) *Health { return &Health{checks: checks} }

// Liveness always returns 200 once the process is running (used by the liveness probe).
func (h *Health) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readiness pings each dependency; it returns 503 if any is unhealthy so traffic is not routed
// to a pod that cannot serve. Dependencies with graceful degradation (Redis, Kafka) should be
// excluded from readiness — only hard dependencies belong here.
func (h *Health) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	for _, c := range h.checks {
		if err := c.Check.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready: " + c.Name))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
