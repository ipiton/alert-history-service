// Package metrics provides metrics collection for alert filtering system.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// FilterMetrics holds Prometheus metrics for alert filtering system.
type FilterMetrics struct {
	alertsFiltered    *prometheus.CounterVec
	filterDuration    *prometheus.HistogramVec
	blockedAlerts     *prometheus.CounterVec
	filterValidations *prometheus.CounterVec
}

// NewFilterMetrics creates a new FilterMetrics instance.
func NewFilterMetrics() *FilterMetrics {
	return NewFilterMetricsWithNamespace("alert_history", "filter")
}

// NewFilterMetricsWithNamespace creates a new FilterMetrics instance with custom namespace and subsystem.
func NewFilterMetricsWithNamespace(namespace, subsystem string) *FilterMetrics {
	return &FilterMetrics{
		alertsFiltered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "alerts_filtered_total",
				Help:      "Total number of alerts processed by filter engine",
			},
			[]string{"result"}, // result: "allowed", "blocked"
		),
		filterDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "duration_seconds",
				Help:      "Duration of filter processing in seconds",
				Buckets:   []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1}, // nanoseconds to milliseconds
			},
			[]string{"result"},
		),
		blockedAlerts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "blocked_alerts_total",
				Help:      "Total number of alerts blocked by filter rules",
			},
			[]string{"reason"}, // reason: "test_alert", "noise", "low_confidence"
		),
		filterValidations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "validations_total",
				Help:      "Total number of filter validations",
			},
			[]string{"status"}, // status: "success", "failed"
		),
	}
}

// RecordAlertFiltered records an alert being processed by filter.
func (m *FilterMetrics) RecordAlertFiltered(result string) {
	m.alertsFiltered.WithLabelValues(result).Inc()
}

// RecordFilterDuration records the duration of filter processing.
func (m *FilterMetrics) RecordFilterDuration(durationSeconds float64, result string) {
	m.filterDuration.WithLabelValues(result).Observe(durationSeconds)
}

// RecordBlockedAlert records an alert being blocked with reason.
func (m *FilterMetrics) RecordBlockedAlert(reason string) {
	m.blockedAlerts.WithLabelValues(reason).Inc()
}

// RecordValidation records a filter validation result.
func (m *FilterMetrics) RecordValidation(success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.filterValidations.WithLabelValues(status).Inc()
}
