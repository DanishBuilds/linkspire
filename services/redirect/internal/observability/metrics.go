// services/redirect/internal/observability/metrics.go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics bundles the Prometheus collectors for the redirect hot path.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec   // by outcome
	RequestDuration *prometheus.HistogramVec // by outcome
	CacheEvents     *prometheus.CounterVec   // tier=l1|redis|pg|bloom, result=hit|miss
	EventsEmitted   prometheus.Counter
	EventsDropped   prometheus.Counter
	ResolveErrors   prometheus.Counter
	PasswordAttempts *prometheus.CounterVec // result=ok|fail|locked
}

// NewMetrics registers and returns the collectors on the given registry.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)
	return &Metrics{
		RequestsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "redirect_requests_total",
			Help: "Total redirect requests by outcome.",
		}, []string{"outcome"}),
		RequestDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "redirect_duration_seconds",
			Help:    "Redirect handling latency at the origin.",
			// Buckets tuned around the p50<10ms / p99<30ms SLO (and <5ms cache-hit target).
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.03, 0.05, 0.1, 0.25},
		}, []string{"outcome"}),
		CacheEvents: f.NewCounterVec(prometheus.CounterOpts{
			Name: "redirect_cache_events_total",
			Help: "Cache tier hit/miss counters.",
		}, []string{"tier", "result"}),
		EventsEmitted: f.NewCounter(prometheus.CounterOpts{
			Name: "redirect_click_events_emitted_total",
			Help: "Click events successfully enqueued for async delivery.",
		}),
		EventsDropped: f.NewCounter(prometheus.CounterOpts{
			Name: "redirect_click_events_dropped_total",
			Help: "Click events dropped due to a full buffer (backpressure; redirects never block).",
		}),
		ResolveErrors: f.NewCounter(prometheus.CounterOpts{
			Name: "redirect_resolve_errors_total",
			Help: "Errors from the resolve chain (cache/db).",
		}),
		PasswordAttempts: f.NewCounterVec(prometheus.CounterOpts{
			Name: "redirect_password_attempts_total",
			Help: "Password unlock attempts by result.",
		}, []string{"result"}),
	}
}
