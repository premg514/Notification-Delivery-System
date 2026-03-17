package observability

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_http_requests_total",
			Help: "Total number of API HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
	httpPanicsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_http_panics_total",
			Help: "Number of HTTP panics recovered by middleware.",
		},
	)
	rateLimitBlockedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_rate_limit_blocked_total",
			Help: "Number of requests blocked by rate limiter.",
		},
	)
	rateLimitErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_rate_limit_errors_total",
			Help: "Number of rate limiter backend errors.",
		},
	)
	workerJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_worker_jobs_total",
			Help: "Worker jobs processed by pool and result.",
		},
		[]string{"pool", "result"},
	)
	workerJobDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_worker_job_duration_seconds",
			Help:    "Worker job processing duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pool", "result"},
	)
	deliveryRetryScheduledTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_delivery_retry_scheduled_total",
			Help: "Total number of delivery items scheduled for retry.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDurationSeconds,
		httpPanicsTotal,
		rateLimitBlockedTotal,
		rateLimitErrorsTotal,
		workerJobsTotal,
		workerJobDurationSeconds,
		deliveryRetryScheduledTotal,
	)
}

func ObserveHTTPRequest(method, route string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	httpRequestsTotal.WithLabelValues(method, route, status).Inc()
	httpRequestDurationSeconds.WithLabelValues(method, route, status).Observe(duration.Seconds())
}

func IncHTTPPanics() {
	httpPanicsTotal.Inc()
}

func IncRateLimitBlocked() {
	rateLimitBlockedTotal.Inc()
}

func IncRateLimitErrors() {
	rateLimitErrorsTotal.Inc()
}

func ObserveWorkerJob(pool string, success bool, duration time.Duration) {
	result := "success"
	if !success {
		result = "failed"
	}

	workerJobsTotal.WithLabelValues(pool, result).Inc()
	workerJobDurationSeconds.WithLabelValues(pool, result).Observe(duration.Seconds())
}

func AddDeliveryRetryScheduled(count int) {
	if count <= 0 {
		return
	}
	deliveryRetryScheduledTotal.Add(float64(count))
}
