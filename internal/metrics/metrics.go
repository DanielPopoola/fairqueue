package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fairqueue_http_requests_total",
		Help: "Total HTTP requests by method, path, and status code",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fairqueue_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// Queue metrics
	QueueWaitingTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fairqueue_queue_waiting_total",
		Help: "Number of customers currently waiting in queue per event",
	}, []string{"event_id"})

	QueueAdmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fairqueue_queue_admitted_total",
		Help: "Total customers admitted from queue",
	}, []string{"event_id"})

	// Claim metrics
	ClaimsCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fairqueue_claims_created_total",
		Help: "Total claims created",
	}, []string{"event_id"})

	ClaimsExpiredTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fairqueue_claims_expired_total",
		Help: "Total claims expired without payment",
	}, []string{"event_id"})

	// Worker metrics
	WorkerTickDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fairqueue_worker_tick_duration_seconds",
		Help:    "Time taken per worker tick",
		Buckets: prometheus.DefBuckets,
	}, []string{"worker"})

	WorkerTickErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fairqueue_worker_tick_errors_total",
		Help: "Total worker tick errors",
	}, []string{"worker"})
)
