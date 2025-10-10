// Package metrics provides centralized metrics management for Alert History Service.
//
// This package implements a unified taxonomy for Prometheus metrics:
//   - Business metrics: alerts processing, LLM classification, publishing
//   - Technical metrics: HTTP, filters, enrichment, circuit breakers
//   - Infrastructure metrics: database, cache, repositories
//
// All metrics follow the naming convention:
// alert_history_<category>_<subsystem>_<metric_name>_<unit>
//
// Example:
//
//	registry := metrics.DefaultRegistry()
//	registry.Business().AlertsProcessedTotal.Inc()
//	registry.Infra().DB.ConnectionsActive.Set(42)
package metrics

import (
	"sync"
)

// MetricCategory represents the category of a metric.
type MetricCategory string

const (
	// CategoryBusiness represents business-level metrics (alert processing, LLM, publishing)
	CategoryBusiness MetricCategory = "business"

	// CategoryTechnical represents technical metrics (HTTP, filters, enrichment, circuit breakers)
	CategoryTechnical MetricCategory = "technical"

	// CategoryInfra represents infrastructure metrics (database, cache, repositories)
	CategoryInfra MetricCategory = "infra"
)

// MetricsRegistry is the central registry for all Prometheus metrics.
// Provides organized access to metrics by category (Business, Technical, Infra).
//
// This is a simplified registry design (vs. full validation/map approach)
// for better maintainability and performance.
//
// Usage:
//
//	registry := metrics.DefaultRegistry()
//	registry.Business().AlertsProcessedTotal.Inc()
//
// Thread-safe: All Prometheus metrics are thread-safe by design.
// Singleton: Use DefaultRegistry() to get the global instance.
type MetricsRegistry struct {
	namespace string

	// Category managers (lazy-initialized)
	business  *BusinessMetrics
	technical *TechnicalMetrics
	infra     *InfraMetrics

	// Separate sync.Once for each category for true lazy initialization
	businessOnce  sync.Once
	technicalOnce sync.Once
	infraOnce     sync.Once
}

var (
	// Global singleton registry instance
	defaultRegistry     *MetricsRegistry
	defaultRegistryOnce sync.Once
)

// DefaultRegistry returns the global singleton MetricsRegistry.
// Safe for concurrent use. Initialized once on first call.
//
// Example:
//
//	registry := metrics.DefaultRegistry()
//	registry.Infra().DB.ConnectionsActive.Set(10)
func DefaultRegistry() *MetricsRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewMetricsRegistry("alert_history")
	})
	return defaultRegistry
}

// NewMetricsRegistry creates a new MetricsRegistry with the specified namespace.
// For most use cases, use DefaultRegistry() instead of calling this directly.
//
// Parameters:
//   - namespace: The Prometheus namespace for all metrics (typically "alert_history")
//
// Returns:
//   - *MetricsRegistry: A new registry instance
func NewMetricsRegistry(namespace string) *MetricsRegistry {
	if namespace == "" {
		namespace = "alert_history"
	}

	return &MetricsRegistry{
		namespace: namespace,
	}
}

// Business returns the Business metrics manager.
// Lazy-initialized on first access.
//
// Business metrics include:
//   - Alert processing (processed, enriched, filtered)
//   - LLM classification (classifications, recommendations, confidence)
//   - Publishing (success, failures, duration)
//
// Example:
//
//	registry.Business().AlertsProcessedTotal.Inc()
//	registry.Business().LLMConfidenceScore.Observe(0.95)
func (r *MetricsRegistry) Business() *BusinessMetrics {
	r.businessOnce.Do(func() {
		r.business = NewBusinessMetrics(r.namespace)
	})
	return r.business
}

// Technical returns the Technical metrics manager.
// Lazy-initialized on first access.
//
// Technical metrics include:
//   - HTTP requests (count, duration, size)
//   - Filter operations (filtered, blocked)
//   - Enrichment mode (switches, status)
//   - LLM Circuit Breaker (state, failures, duration)
//
// Example:
//
//	registry.Technical().HTTP.RecordRequest("GET", "/api/alerts", 200, 0.123)
//	registry.Technical().LLMCB.State.Set(0) // closed
func (r *MetricsRegistry) Technical() *TechnicalMetrics {
	r.technicalOnce.Do(func() {
		r.technical = NewTechnicalMetrics(r.namespace)
	})
	return r.technical
}

// Infra returns the Infrastructure metrics manager.
// Lazy-initialized on first access.
//
// Infrastructure metrics include:
//   - Database (connections, queries, errors)
//   - Cache (hits, misses, evictions)
//   - Repository (query duration, errors, results)
//
// Example:
//
//	registry.Infra().DB.ConnectionsActive.Set(42)
//	registry.Infra().Repository.QueryDuration.WithLabelValues("GetTopAlerts", "success").Observe(0.05)
func (r *MetricsRegistry) Infra() *InfraMetrics {
	r.infraOnce.Do(func() {
		r.infra = NewInfraMetrics(r.namespace)
	})
	return r.infra
}

// Namespace returns the configured namespace for this registry.
//
// Returns:
//   - string: The Prometheus namespace (e.g., "alert_history")
func (r *MetricsRegistry) Namespace() string {
	return r.namespace
}

// ValidateMetricName validates a metric name against naming conventions.
// Currently a placeholder for future validation logic.
//
// Naming convention:
// <namespace>_<category>_<subsystem>_<metric_name>_<unit>
//
// Examples:
// ✅ alert_history_business_alerts_processed_total
// ✅ alert_history_technical_http_request_duration_seconds
// ✅ alert_history_infra_db_connections_active
// ❌ alerts_processed (missing namespace)
// ❌ alert_history_processed (missing category/subsystem)
//
// Parameters:
//   - name: The metric name to validate
//
// Returns:
//   - error: nil if valid, error describing the problem otherwise
//
// TODO: Implement validation logic (regex, taxonomy check)
func (r *MetricsRegistry) ValidateMetricName(name string) error {
	// Placeholder for future validation
	// Could check:
	// 1. Starts with namespace
	// 2. Contains category (business/technical/infra)
	// 3. Follows snake_case
	// 4. Has appropriate unit suffix (_total, _seconds, etc.)
	return nil
}
