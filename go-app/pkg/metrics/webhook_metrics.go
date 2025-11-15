package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// WebhookMetrics contains all Prometheus metrics for the webhook endpoint.
type WebhookMetrics struct {
	// Request metrics
	RequestsTotal       *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	RequestSizeBytes    prometheus.Histogram
	ResponseSizeBytes   prometheus.Histogram

	// Processing metrics
	AlertsReceivedTotal     prometheus.Counter
	AlertsProcessedTotal    *prometheus.CounterVec
	ProcessingDuration      *prometheus.HistogramVec
	AlertsPerRequest        prometheus.Histogram

	// Error metrics
	ErrorsTotal           *prometheus.CounterVec
	ValidationErrorsTotal *prometheus.CounterVec
	TimeoutsTotal         prometheus.Counter

	// Security metrics
	AuthFailuresTotal         *prometheus.CounterVec
	RateLimitHitsTotal        *prometheus.CounterVec
	SuspiciousActivityTotal   *prometheus.CounterVec

	// Resource metrics
	ActiveGoroutines prometheus.Gauge
	MemoryUsageBytes prometheus.Gauge
	DBConnectionsTotal *prometheus.GaugeVec
}

// NewWebhookMetrics creates and registers all webhook metrics.
func NewWebhookMetrics(registry *prometheus.Registry) *WebhookMetrics {
	factory := promauto.With(registry)

	return &WebhookMetrics{
		// Request metrics
		RequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_requests_total",
				Help: "Total number of webhook requests",
			},
			[]string{"status", "method"},
		),
		RequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webhook_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"endpoint", "status"},
		),
		RequestSizeBytes: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "webhook_request_size_bytes",
				Help:    "Request payload size in bytes",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
		),
		ResponseSizeBytes: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "webhook_response_size_bytes",
				Help:    "Response payload size in bytes",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
			},
		),

		// Processing metrics
		AlertsReceivedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "webhook_alerts_received_total",
				Help: "Total number of alerts received",
			},
		),
		AlertsProcessedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_alerts_processed_total",
				Help: "Total number of alerts processed",
			},
			[]string{"status"}, // success, failed, partial
		),
		ProcessingDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webhook_processing_duration_seconds",
				Help:    "Processing duration by stage in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"stage"}, // parse, validate, process, store
		),
		AlertsPerRequest: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "webhook_alerts_per_request",
				Help:    "Number of alerts per request",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),

		// Error metrics
		ErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "stage"}, // type: validation, processing, storage; stage: parse, validate, process
		),
		ValidationErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_validation_errors_total",
				Help: "Total number of validation errors",
			},
			[]string{"field"}, // labels, annotations, status, url
		),
		TimeoutsTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "webhook_timeouts_total",
				Help: "Total number of request timeouts",
			},
		),

		// Security metrics
		AuthFailuresTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"type", "reason"}, // type: apikey, hmac; reason: missing, invalid, expired
		),
		RateLimitHitsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"client_ip", "limit_type"}, // limit_type: per_ip, global
		),
		SuspiciousActivityTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_suspicious_activity_total",
				Help: "Total number of suspicious activity detections",
			},
			[]string{"pattern"}, // repeated_failures, invalid_format, malicious_payload
		),

		// Resource metrics
		ActiveGoroutines: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "webhook_active_goroutines",
				Help: "Number of active goroutines",
			},
		),
		MemoryUsageBytes: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "webhook_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),
		DBConnectionsTotal: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "webhook_db_connections_total",
				Help: "Number of database connections",
			},
			[]string{"state"}, // idle, in_use, open
		),
	}
}

// RecordRequest records a webhook request.
func (m *WebhookMetrics) RecordRequest(status, method string, duration time.Duration, requestSize, responseSize int) {
	m.RequestsTotal.WithLabelValues(status, method).Inc()
	m.RequestDuration.WithLabelValues("/webhook", status).Observe(duration.Seconds())
	m.RequestSizeBytes.Observe(float64(requestSize))
	m.ResponseSizeBytes.Observe(float64(responseSize))
}

// RecordProcessing records alert processing.
func (m *WebhookMetrics) RecordProcessing(alertCount int, status string, stage string, duration time.Duration) {
	m.AlertsReceivedTotal.Add(float64(alertCount))
	m.AlertsProcessedTotal.WithLabelValues(status).Add(float64(alertCount))
	m.ProcessingDuration.WithLabelValues(stage).Observe(duration.Seconds())
	m.AlertsPerRequest.Observe(float64(alertCount))
}

// RecordError records an error.
func (m *WebhookMetrics) RecordError(errorType, stage string) {
	m.ErrorsTotal.WithLabelValues(errorType, stage).Inc()
}

// RecordValidationError records a validation error.
func (m *WebhookMetrics) RecordValidationError(field string) {
	m.ValidationErrorsTotal.WithLabelValues(field).Inc()
}

// RecordTimeout records a timeout.
func (m *WebhookMetrics) RecordTimeout() {
	m.TimeoutsTotal.Inc()
}

// RecordAuthFailure records an authentication failure.
func (m *WebhookMetrics) RecordAuthFailure(authType, reason string) {
	m.AuthFailuresTotal.WithLabelValues(authType, reason).Inc()
}

// RecordRateLimitHit records a rate limit hit.
func (m *WebhookMetrics) RecordRateLimitHit(clientIP, limitType string) {
	m.RateLimitHitsTotal.WithLabelValues(clientIP, limitType).Inc()
}

// RecordSuspiciousActivity records suspicious activity.
func (m *WebhookMetrics) RecordSuspiciousActivity(pattern string) {
	m.SuspiciousActivityTotal.WithLabelValues(pattern).Inc()
}

// UpdateResourceMetrics updates resource metrics.
func (m *WebhookMetrics) UpdateResourceMetrics(goroutines int, memoryBytes uint64, dbIdle, dbInUse, dbOpen int) {
	m.ActiveGoroutines.Set(float64(goroutines))
	m.MemoryUsageBytes.Set(float64(memoryBytes))
	m.DBConnectionsTotal.WithLabelValues("idle").Set(float64(dbIdle))
	m.DBConnectionsTotal.WithLabelValues("in_use").Set(float64(dbInUse))
	m.DBConnectionsTotal.WithLabelValues("open").Set(float64(dbOpen))
}
