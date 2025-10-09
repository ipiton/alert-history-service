// Package metrics provides metrics collection for enrichment mode system.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EnrichmentMetrics holds Prometheus metrics for enrichment mode system.
type EnrichmentMetrics struct {
	modeSwitches *prometheus.CounterVec
	modeStatus   prometheus.Gauge
	modeRequests *prometheus.CounterVec
	redisErrors  prometheus.Counter
}

// NewEnrichmentMetrics creates a new EnrichmentMetrics instance.
func NewEnrichmentMetrics() *EnrichmentMetrics {
	return NewEnrichmentMetricsWithNamespace("alert_history", "enrichment")
}

// NewEnrichmentMetricsWithNamespace creates a new EnrichmentMetrics instance with custom namespace and subsystem.
func NewEnrichmentMetricsWithNamespace(namespace, subsystem string) *EnrichmentMetrics {
	return &EnrichmentMetrics{
		modeSwitches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "mode_switches_total",
				Help:      "Total number of enrichment mode switches",
			},
			[]string{"from_mode", "to_mode"},
		),
		modeStatus: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "mode_status",
				Help:      "Current enrichment mode (0=transparent, 1=enriched, 2=transparent_with_recommendations)",
			},
		),
		modeRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "mode_requests_total",
				Help:      "Total number of enrichment mode API requests",
			},
			[]string{"method", "mode"},
		),
		redisErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_errors_total",
				Help:      "Total number of Redis errors in enrichment mode manager",
			},
		),
	}
}

// RecordModeSwitch records a mode switch from one mode to another.
func (m *EnrichmentMetrics) RecordModeSwitch(fromMode, toMode string) {
	m.modeSwitches.WithLabelValues(fromMode, toMode).Inc()
}

// SetModeStatus sets the current mode status.
// Values: 0=transparent, 1=enriched, 2=transparent_with_recommendations
func (m *EnrichmentMetrics) SetModeStatus(value float64) {
	m.modeStatus.Set(value)
}

// RecordModeRequest records an API request to get or set enrichment mode.
func (m *EnrichmentMetrics) RecordModeRequest(method, mode string) {
	m.modeRequests.WithLabelValues(method, mode).Inc()
}

// RecordRedisError records a Redis error.
func (m *EnrichmentMetrics) RecordRedisError() {
	m.redisErrors.Inc()
}
