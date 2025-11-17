package ui

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TemplateMetrics tracks template engine metrics.
type TemplateMetrics struct {
	renderTotal    *prometheus.CounterVec
	renderDuration prometheus.Histogram
	cacheHits      prometheus.Counter
}

// NewTemplateMetrics creates a new metrics instance.
func NewTemplateMetrics() *TemplateMetrics {
	return &TemplateMetrics{
		renderTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "template",
				Name:      "render_total",
				Help:      "Total template renders by template and status",
			},
			[]string{"template", "status"},
		),
		renderDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "template",
				Name:      "render_duration_seconds",
				Help:      "Template render duration in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
		),
		cacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "template",
				Name:      "cache_hits_total",
				Help:      "Total template cache hits (production mode)",
			},
		),
	}
}

// RecordRender records a template render operation.
//
// Parameters:
//   - templateName: Name of the template rendered
//   - duration: Time taken to render
//   - success: Whether rendering succeeded
func (m *TemplateMetrics) RecordRender(templateName string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	m.renderTotal.WithLabelValues(templateName, status).Inc()
	m.renderDuration.Observe(duration.Seconds())
}

// RecordCacheHit records a template cache hit.
func (m *TemplateMetrics) RecordCacheHit() {
	m.cacheHits.Inc()
}
