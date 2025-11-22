package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ================================================================================
// TN-152: Config Reload Metrics
// ================================================================================
// Prometheus metrics for configuration hot reload operations.
//
// Metrics:
// - config_reload_total: Total reload attempts by status
// - config_reload_duration_seconds: Reload duration histogram
// - config_reload_phase_duration_seconds: Duration by phase
// - config_reload_component_duration_seconds: Component reload duration
// - config_reload_errors_total: Errors by type
// - config_reload_last_success_timestamp_seconds: Last successful reload
// - config_reload_rollbacks_total: Rollback count by reason
// - config_reload_version: Current config version
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-22

var (
	// ConfigReloadTotal tracks total reload attempts by status
	//
	// Labels:
	//   - status: success, error, validation_failed, rolled_back
	//
	// Usage:
	//   ConfigReloadTotal.WithLabelValues("success").Inc()
	ConfigReloadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_reload_total",
			Help: "Total number of config reload attempts by status",
		},
		[]string{"status"},
	)

	// ConfigReloadDuration tracks reload duration histogram
	//
	// Buckets optimized for < 500ms target:
	//   10ms, 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s
	//
	// Usage:
	//   ConfigReloadDuration.Observe(duration.Seconds())
	ConfigReloadDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "config_reload_duration_seconds",
			Help:    "Duration of config reload operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
	)

	// ConfigReloadPhaseDuration tracks duration by phase
	//
	// Labels:
	//   - phase: load, validate, diff, apply, reload, health_check
	//
	// Buckets optimized for phase-specific targets:
	//   1ms, 5ms, 10ms, 50ms, 100ms, 200ms, 500ms
	//
	// Usage:
	//   ConfigReloadPhaseDuration.WithLabelValues("load").Observe(duration.Seconds())
	ConfigReloadPhaseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "config_reload_phase_duration_seconds",
			Help:    "Duration of config reload phases",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.2, 0.5},
		},
		[]string{"phase"},
	)

	// ConfigReloadComponentDuration tracks component reload duration
	//
	// Labels:
	//   - component: routing, receivers, inhibition, silencing, grouping, llm, database, redis
	//
	// Buckets optimized for component reload (< 300ms target):
	//   1ms, 10ms, 50ms, 100ms, 200ms, 500ms, 1s
	//
	// Usage:
	//   ConfigReloadComponentDuration.WithLabelValues("routing").Observe(duration.Seconds())
	ConfigReloadComponentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "config_reload_component_duration_seconds",
			Help:    "Duration of component reload operations",
			Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.2, 0.5, 1.0},
		},
		[]string{"component"},
	)

	// ConfigReloadErrors tracks reload errors by type
	//
	// Labels:
	//   - type: load_failed, validation_failed, apply_failed, reload_failed, timeout, rollback_failed
	//
	// Usage:
	//   ConfigReloadErrors.WithLabelValues("validation_failed").Inc()
	ConfigReloadErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_reload_errors_total",
			Help: "Total number of config reload errors by type",
		},
		[]string{"type"},
	)

	// ConfigReloadLastSuccess tracks last successful reload timestamp
	//
	// Usage:
	//   ConfigReloadLastSuccess.SetToCurrentTime()
	ConfigReloadLastSuccess = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "config_reload_last_success_timestamp_seconds",
			Help: "Timestamp of last successful config reload (Unix epoch)",
		},
	)

	// ConfigReloadRollbacks tracks rollback count by reason
	//
	// Labels:
	//   - reason: critical_failed, timeout, health_check_failed
	//
	// Usage:
	//   ConfigReloadRollbacks.WithLabelValues("critical_failed").Inc()
	ConfigReloadRollbacks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_reload_rollbacks_total",
			Help: "Total number of config reload rollbacks by reason",
		},
		[]string{"reason"},
	)

	// ConfigReloadVersion tracks current config version
	//
	// Usage:
	//   ConfigReloadVersion.Set(float64(version))
	ConfigReloadVersion = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "config_reload_version",
			Help: "Current configuration version number",
		},
	)
)
