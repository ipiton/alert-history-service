package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HistoryMetrics contains all Prometheus metrics for history endpoint
type HistoryMetrics struct {
	// HTTP Metrics (6 metrics)
	HTTPRequestsTotal      *prometheus.CounterVec   // Total HTTP requests
	HTTPRequestDuration    *prometheus.HistogramVec // Request duration
	HTTPRequestSize        *prometheus.HistogramVec // Request size
	HTTPResponseSize       *prometheus.HistogramVec // Response size
	HTTPErrorsTotal        *prometheus.CounterVec   // HTTP errors
	HTTPActiveRequests     prometheus.Gauge         // Active requests

	// Filter Metrics (4 metrics)
	FilterOperationsTotal  *prometheus.CounterVec   // Filter operations
	FilterDuration          *prometheus.HistogramVec // Filter processing duration
	FilterErrorsTotal       *prometheus.CounterVec   // Filter errors
	FiltersApplied          *prometheus.HistogramVec // Number of filters applied per request

	// Query Metrics (3 metrics)
	QueryDuration           *prometheus.HistogramVec // Query execution duration
	QueryResults            *prometheus.HistogramVec // Query result count
	QueryErrorsTotal        *prometheus.CounterVec   // Query errors

	// Cache Metrics (already exist, but tracked here)
	CacheHitsTotal          *prometheus.CounterVec   // Cache hits
	CacheMissesTotal        *prometheus.CounterVec   // Cache misses
	CacheSize               *prometheus.GaugeVec    // Cache size

	// Security Metrics (3 metrics)
	SecurityEventsTotal     *prometheus.CounterVec   // Security events
	AuthFailuresTotal       *prometheus.CounterVec   // Authentication failures
	RateLimitViolationsTotal *prometheus.CounterVec // Rate limit violations

	// Performance Metrics (2 metrics)
	P95Latency              prometheus.Gauge         // p95 latency
	P99Latency              prometheus.Gauge         // p99 latency
}

// NewHistoryMetrics creates new history metrics
func NewHistoryMetrics() *HistoryMetrics {
	return &HistoryMetrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_errors_total",
				Help:      "Total number of HTTP errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),
		HTTPActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "api_history",
				Name:      "http_active_requests",
				Help:      "Number of active HTTP requests",
			},
		),

		// Filter Metrics
		FilterOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_filters",
				Name:      "operations_total",
				Help:      "Total number of filter operations",
			},
			[]string{"filter_type", "status"},
		),
		FilterDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_filters",
				Name:      "duration_seconds",
				Help:      "Filter processing duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05},
			},
			[]string{"filter_type"},
		),
		FilterErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_filters",
				Name:      "errors_total",
				Help:      "Total number of filter errors",
			},
			[]string{"filter_type", "error_type"},
		),
		FiltersApplied: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_filters",
				Name:      "applied_count",
				Help:      "Number of filters applied per request",
				Buckets:   []float64{0, 1, 2, 3, 5, 10, 15, 20},
			},
			[]string{"endpoint"},
		),

		// Query Metrics (reuse from repository, but add here for completeness)
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_query",
				Name:      "duration_seconds",
				Help:      "Query execution duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"operation", "status"},
		),
		QueryResults: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_query",
				Name:      "results_count",
				Help:      "Number of results returned by queries",
				Buckets:   []float64{0, 1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"operation"},
		),
		QueryErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_query",
				Name:      "errors_total",
				Help:      "Total number of query errors",
			},
			[]string{"operation", "error_type"},
		),

		// Cache Metrics (reference existing)
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_layer"},
		),
		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_layer"},
		),
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "size_entries",
				Help:      "Current number of entries in cache",
			},
			[]string{"cache_layer"},
		),

		// Security Metrics
		SecurityEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_security",
				Name:      "events_total",
				Help:      "Total number of security events",
			},
			[]string{"event_type", "severity"},
		),
		AuthFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_security",
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"auth_type", "reason"},
		),
		RateLimitViolationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_security",
				Name:      "rate_limit_violations_total",
				Help:      "Total number of rate limit violations",
			},
			[]string{"limit_type", "endpoint"},
		),

		// Performance Metrics
		P95Latency: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_performance",
				Name:      "p95_latency_seconds",
				Help:      "p95 latency in seconds",
			},
		),
		P99Latency: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_performance",
				Name:      "p99_latency_seconds",
				Help:      "p99 latency in seconds",
			},
		),
	}
}

// DefaultHistoryMetrics returns the default history metrics instance
var defaultHistoryMetrics *HistoryMetrics

func init() {
	defaultHistoryMetrics = NewHistoryMetrics()
}

// Default returns the default history metrics instance
func Default() *HistoryMetrics {
	return defaultHistoryMetrics
}
