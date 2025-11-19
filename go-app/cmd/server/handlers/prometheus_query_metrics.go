// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusQueryMetrics tracks metrics for GET /api/v2/alerts endpoint.
//
// Provides comprehensive observability for monitoring query performance,
// error rates, and usage patterns.
//
// Metrics:
//   - alerthistory_prometheus_query_requests_total: Total requests by status
//   - alerthistory_prometheus_query_duration_seconds: Request duration histogram
//   - alerthistory_prometheus_query_results_total: Number of results returned
//   - alerthistory_prometheus_query_errors_total: Errors by type
//   - alerthistory_prometheus_query_validation_errors_total: Validation errors by parameter
//   - alerthistory_prometheus_query_concurrent_requests: Active concurrent requests
type PrometheusQueryMetrics struct {
	RequestsTotal           *prometheus.CounterVec
	RequestDuration         *prometheus.HistogramVec
	ResultsTotal            prometheus.Histogram
	ErrorsTotal             *prometheus.CounterVec
	ValidationErrorsTotal   *prometheus.CounterVec
	ConcurrentRequests      prometheus.Gauge
}

// NewPrometheusQueryMetrics initializes metrics for GET /api/v2/alerts endpoint.
//
// All metrics are automatically registered with the default Prometheus registry.
//
// Returns:
//   - *PrometheusQueryMetrics: Initialized metrics collector
func NewPrometheusQueryMetrics() *PrometheusQueryMetrics {
	return &PrometheusQueryMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerthistory_prometheus_query_requests_total",
				Help: "Total HTTP requests to GET /api/v2/alerts endpoint by status",
			},
			[]string{"status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alerthistory_prometheus_query_duration_seconds",
				Help:    "Request processing duration for GET /api/v2/alerts endpoint",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"status"},
		),
		ResultsTotal: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "alerthistory_prometheus_query_results_total",
				Help:    "Number of alerts returned in query results",
				Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerthistory_prometheus_query_errors_total",
				Help: "Total errors for GET /api/v2/alerts endpoint by type",
			},
			[]string{"error_type"},
		),
		ValidationErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerthistory_prometheus_query_validation_errors_total",
				Help: "Total validation errors by parameter name",
			},
			[]string{"parameter"},
		),
		ConcurrentRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "alerthistory_prometheus_query_concurrent_requests",
				Help: "Current number of concurrent requests being processed by GET /api/v2/alerts",
			},
		),
	}
}

// RecordRequest records metrics for a completed request.
//
// Parameters:
//   - status: Request status ("success", "error", "validation_failed")
//   - alertCount: Number of alerts returned
//   - duration: Request processing duration
func (m *PrometheusQueryMetrics) RecordRequest(status string, alertCount int, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(status).Inc()
	m.RequestDuration.WithLabelValues(status).Observe(duration.Seconds())
	if alertCount > 0 {
		m.ResultsTotal.Observe(float64(alertCount))
	}
}

// RecordValidationError records a validation error.
//
// Parameters:
//   - parameter: Parameter name that failed validation
func (m *PrometheusQueryMetrics) RecordValidationError(parameter string) {
	m.ValidationErrorsTotal.WithLabelValues(parameter).Inc()
}

// RecordError records an error.
//
// Parameters:
//   - errorType: Type of error (e.g., "database_error", "conversion_error")
func (m *PrometheusQueryMetrics) RecordError(errorType string) {
	m.ErrorsTotal.WithLabelValues(errorType).Inc()
}

// IncrementConcurrent increments the concurrent requests gauge.
func (m *PrometheusQueryMetrics) IncrementConcurrent() {
	m.ConcurrentRequests.Inc()
}

// DecrementConcurrent decrements the concurrent requests gauge.
func (m *PrometheusQueryMetrics) DecrementConcurrent() {
	m.ConcurrentRequests.Dec()
}
