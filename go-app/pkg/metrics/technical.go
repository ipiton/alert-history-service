package metrics

// TechnicalMetrics aggregates all technical-level metrics for Alert History Service.
//
// Technical metrics track system internals:
//   - HTTP requests (via existing HTTPMetrics)
//   - Webhook processing (via WebhookMetrics - TN-045)
//   - Filter operations (via existing FilterMetrics)
//   - Enrichment mode (via existing EnrichmentMetrics)
//   - LLM Circuit Breaker (renamed metrics)
//
// This is an aggregator struct that groups existing metrics under the technical category.
// Most metrics are already implemented in separate files (prometheus.go, filter.go, enrichment.go, webhook.go).
//
// Example:
//
//	tm := NewTechnicalMetrics("alert_history")
//	tm.HTTP.RecordRequest("GET", "/api/alerts", 200, 0.123)
//	tm.Webhook.RecordRequest("alertmanager", "success", 0.045)
//	tm.Filter.RecordAlertFiltered("allowed")
type TechnicalMetrics struct {
	namespace string

	// HTTP subsystem - existing metrics from prometheus.go
	HTTP *HTTPMetrics

	// Webhook subsystem - webhook processing metrics from webhook.go (TN-045)
	Webhook *WebhookMetrics

	// Filter subsystem - existing metrics from filter.go
	Filter *FilterMetrics

	// Enrichment subsystem - existing metrics from enrichment.go
	Enrichment *EnrichmentMetrics

	// LLMCB (LLM Circuit Breaker) subsystem - metrics from llm/circuit_breaker_metrics.go
	// Note: This will use renamed metrics (technical_llm_cb_* instead of llm_circuit_breaker_*)
	// TODO: Implement after Circuit Breaker metrics refactor
	// LLMCB *LLMCircuitBreakerMetrics
}

// NewTechnicalMetrics creates a new TechnicalMetrics aggregator.
// Reuses existing metric implementations for HTTP, Webhook, Filter, and Enrichment.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *TechnicalMetrics: Initialized technical metrics aggregator
func NewTechnicalMetrics(namespace string) *TechnicalMetrics {
	return &TechnicalMetrics{
		namespace:  namespace,
		HTTP:       NewHTTPMetrics(),       // Uses existing implementation
		Webhook:    NewWebhookMetrics(),    // TN-045: Webhook processing metrics
		Filter:     NewFilterMetrics(),     // Uses existing implementation
		Enrichment: NewEnrichmentMetrics(), // Uses existing implementation
		// LLMCB: Will be added after Circuit Breaker refactor
	}
}
