package handlers

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ================================================================================
// Configuration Update Prometheus Metrics
// ================================================================================
// Prometheus metrics for configuration update operations (TN-150).
//
// Metrics:
// - config_update_requests_total: Total update requests
// - config_update_duration_seconds: Update duration histogram
// - config_update_errors_total: Total errors by type
// - config_validation_errors_total: Validation errors by type
// - config_reload_duration_seconds: Component reload duration
// - config_version: Current config version
// - config_rollbacks_total: Total rollbacks
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-22

// ConfigUpdateMetrics holds Prometheus metrics for config updates
type ConfigUpdateMetrics struct {
	// config_update_requests_total: Total config update requests
	// Labels: format (json/yaml), dry_run (true/false), sections_count, status (success/error/validation_error/conflict)
	requestsTotal *prometheus.CounterVec

	// config_update_duration_seconds: Config update duration distribution
	// Labels: format, dry_run, status
	// Buckets: 0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0 seconds
	duration *prometheus.HistogramVec

	// config_update_errors_total: Total errors by error type
	// Labels: error_type (validation_error, conflict, server_error, method_not_allowed, etc.)
	errorsTotal *prometheus.CounterVec

	// config_validation_errors_total: Validation errors by field
	// Labels: field, code (required, out_of_range, invalid_type, etc.)
	validationErrorsTotal *prometheus.CounterVec

	// config_reload_duration_seconds: Component reload duration
	// Labels: component (database, redis, llm, etc.), success (true/false)
	// Buckets: 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0 seconds
	reloadDuration *prometheus.HistogramVec

	// config_version: Current configuration version (gauge)
	// No labels
	version prometheus.Gauge

	// config_rollbacks_total: Total rollbacks
	// Labels: trigger (auto/manual), reason (critical_reload_failed, manual_request)
	rollbacksTotal *prometheus.CounterVec
}

// NewConfigUpdateMetrics creates and registers Prometheus metrics
func NewConfigUpdateMetrics() *ConfigUpdateMetrics {
	metrics := &ConfigUpdateMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "update_requests_total",
				Help:      "Total number of configuration update requests",
			},
			[]string{"format", "dry_run", "sections_count", "status"},
		),

		duration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "update_duration_seconds",
				Help:      "Configuration update duration in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"format", "dry_run", "status"},
		),

		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "update_errors_total",
				Help:      "Total number of configuration update errors by type",
			},
			[]string{"error_type"},
		),

		validationErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "validation_errors_total",
				Help:      "Total number of configuration validation errors by field and code",
			},
			[]string{"field", "code"},
		),

		reloadDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "reload_duration_seconds",
				Help:      "Component reload duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0},
			},
			[]string{"component", "success"},
		),

		version: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "version",
				Help:      "Current configuration version number",
			},
		),

		rollbacksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "config",
				Name:      "rollbacks_total",
				Help:      "Total number of configuration rollbacks",
			},
			[]string{"trigger", "reason"},
		),
	}

	return metrics
}

// RecordRequest records a config update request
func (m *ConfigUpdateMetrics) RecordRequest(
	format string,
	dryRun bool,
	sectionsCount int,
	status string,
	duration time.Duration,
) {
	dryRunStr := "false"
	if dryRun {
		dryRunStr = "true"
	}

	sectionsCountStr := "all"
	if sectionsCount > 0 {
		sectionsCountStr = "partial"
	}

	m.requestsTotal.WithLabelValues(format, dryRunStr, sectionsCountStr, status).Inc()
	m.duration.WithLabelValues(format, dryRunStr, status).Observe(duration.Seconds())
}

// RecordError records an error by type
func (m *ConfigUpdateMetrics) RecordError(errorType string) {
	m.errorsTotal.WithLabelValues(errorType).Inc()
}

// RecordValidationError records a validation error
func (m *ConfigUpdateMetrics) RecordValidationError(field string, code string) {
	m.validationErrorsTotal.WithLabelValues(field, code).Inc()
}

// RecordReload records a component reload
func (m *ConfigUpdateMetrics) RecordReload(component string, success bool, duration time.Duration) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	m.reloadDuration.WithLabelValues(component, successStr).Observe(duration.Seconds())
}

// SetVersion sets the current config version
func (m *ConfigUpdateMetrics) SetVersion(version int64) {
	m.version.Set(float64(version))
}

// RecordRollback records a rollback
func (m *ConfigUpdateMetrics) RecordRollback(trigger string, reason string) {
	m.rollbacksTotal.WithLabelValues(trigger, reason).Inc()
}
