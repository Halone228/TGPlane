// Package metrics defines all Prometheus metrics for TGPlane.
// Use the top-level vars directly; they are registered with the default registry
// via promauto at init time.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Session metrics
var (
	// SessionsActive tracks currently active sessions by type and status.
	SessionsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tgplane",
		Name:      "sessions_active",
		Help:      "Number of active sessions, by type and status.",
	}, []string{"type", "status"})

	// SessionsTotal counts all sessions ever added, by type.
	SessionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "sessions_total",
		Help:      "Total number of sessions added since start, by type.",
	}, []string{"type"})

	// SessionErrors counts factory / init errors by type.
	SessionErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "session_errors_total",
		Help:      "Total number of session initialization errors, by type.",
	}, []string{"type"})
)

// Update metrics
var (
	// UpdatesReceived counts TDLib updates received by worker sessions.
	UpdatesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "updates_received_total",
		Help:      "Total Telegram updates received from TDLib.",
	}, []string{"worker_id", "session_type"})

	// UpdatesDispatched counts updates successfully dispatched to subscribers.
	UpdatesDispatched = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "updates_dispatched_total",
		Help:      "Total updates dispatched to Subscribe streams.",
	}, []string{"worker_id"})

	// UpdatesDropped counts updates dropped because subscriber channels were full.
	UpdatesDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "updates_dropped_total",
		Help:      "Total updates dropped due to slow subscribers.",
	}, []string{"worker_id"})
)

// HTTP metrics
var (
	// HTTPRequestsTotal counts HTTP requests by method, path pattern, and status code.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests handled.",
	}, []string{"method", "path", "status"})

	// HTTPRequestDuration tracks request latency by method and path pattern.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tgplane",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	}, []string{"method", "path"})
)

// Worker / gRPC metrics
var (
	// WorkerRPCTotal counts gRPC calls handled by worker nodes.
	WorkerRPCTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tgplane",
		Name:      "worker_rpc_total",
		Help:      "Total gRPC calls handled by this worker.",
	}, []string{"method", "status"})

	// WorkerSubscribers tracks the number of active Subscribe streams.
	WorkerSubscribers = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tgplane",
		Name:      "worker_subscribers",
		Help:      "Number of active Subscribe gRPC streams on this worker.",
	})
)

// Build info
var (
	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tgplane",
		Name:      "build_info",
		Help:      "Build information.",
	}, []string{"version", "node"})
)
