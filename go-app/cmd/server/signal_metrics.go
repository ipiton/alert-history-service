package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ================================================================================
// Prometheus Metrics for SIGHUP Hot Reload (TN-152)
// ================================================================================

// SignalPrometheusMetrics holds Prometheus metrics for signal-based hot reload
type SignalPrometheusMetrics struct {
	// Counters
	reloadTotal        *prometheus.CounterVec // Total reload attempts by source and status
	validationFailures *prometheus.CounterVec // Validation failures by source

	// Histograms
	reloadDuration *prometheus.HistogramVec // Reload duration by source

	// Gauges
	lastSuccessTimestamp *prometheus.GaugeVec // Last successful reload timestamp by source
	lastFailureTimestamp *prometheus.GaugeVec // Last failed reload timestamp by source
}

// NewSignalPrometheusMetrics creates Prometheus metrics for signal handler
func NewSignalPrometheusMetrics() *SignalPrometheusMetrics {
	namespace := "alert_history"
	subsystem := "config"

	return &SignalPrometheusMetrics{
		reloadTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reload_total",
				Help:      "Total number of configuration reload attempts",
			},
			[]string{"source", "status"}, // source: sighup|api, status: success|failure
		),
		validationFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reload_validation_failures_total",
				Help:      "Total number of configuration validation failures during reload",
			},
			[]string{"source"}, // source: sighup|api
		),
		reloadDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reload_duration_seconds",
				Help:      "Duration of configuration reload operations in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 5.0}, // 10ms to 5s
			},
			[]string{"source"}, // source: sighup|api
		),
		lastSuccessTimestamp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reload_last_success_timestamp_seconds",
				Help:      "Unix timestamp of last successful configuration reload",
			},
			[]string{"source"}, // source: sighup|api
		),
		lastFailureTimestamp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reload_last_failure_timestamp_seconds",
				Help:      "Unix timestamp of last failed configuration reload",
			},
			[]string{"source"}, // source: sighup|api
		),
	}
}

// RecordReloadAttempt records a reload attempt
func (m *SignalPrometheusMetrics) RecordReloadAttempt(source, status string) {
	m.reloadTotal.WithLabelValues(source, status).Inc()
}

// RecordValidationFailure records a validation failure
func (m *SignalPrometheusMetrics) RecordValidationFailure(source string) {
	m.validationFailures.WithLabelValues(source).Inc()
}

// RecordReloadDuration records reload duration
func (m *SignalPrometheusMetrics) RecordReloadDuration(source string, duration float64) {
	m.reloadDuration.WithLabelValues(source).Observe(duration)
}

// RecordSuccessTimestamp records last success timestamp
func (m *SignalPrometheusMetrics) RecordSuccessTimestamp(source string, timestamp float64) {
	m.lastSuccessTimestamp.WithLabelValues(source).Set(timestamp)
}

// RecordFailureTimestamp records last failure timestamp
func (m *SignalPrometheusMetrics) RecordFailureTimestamp(source string, timestamp float64) {
	m.lastFailureTimestamp.WithLabelValues(source).Set(timestamp)
}
