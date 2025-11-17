package classification

import (
	"context"
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// StatsAggregator aggregates classification statistics from multiple sources
type StatsAggregator struct {
	classifier      services.ClassificationService
	prometheusClient *PrometheusClient // Optional: for enhanced metrics
	logger          *slog.Logger
}

// NewStatsAggregator creates a new stats aggregator
func NewStatsAggregator(classifier services.ClassificationService, logger *slog.Logger) *StatsAggregator {
	if logger == nil {
		logger = slog.Default()
	}
	return &StatsAggregator{
		classifier:       classifier,
		prometheusClient: nil, // Optional, can be set via SetPrometheusClient
		logger:           logger,
	}
}

// NewStatsAggregatorWithPrometheus creates a new stats aggregator with Prometheus client
func NewStatsAggregatorWithPrometheus(classifier services.ClassificationService, promClient *PrometheusClient, logger *slog.Logger) *StatsAggregator {
	agg := NewStatsAggregator(classifier, logger)
	agg.prometheusClient = promClient
	return agg
}

// SetPrometheusClient sets the Prometheus client for enhanced metrics
func (a *StatsAggregator) SetPrometheusClient(client *PrometheusClient) {
	a.prometheusClient = client
}

// AggregateStats aggregates statistics from ClassificationService and optionally Prometheus
func (a *StatsAggregator) AggregateStats(ctx context.Context) (*StatsResponse, error) {
	// Get base stats from ClassificationService
	baseStatsPtr := a.classifier.GetStats()
	baseStats := &baseStatsPtr

	// Try to get Prometheus stats (optional, graceful degradation)
	var promStats *PrometheusStats
	if a.prometheusClient != nil && a.prometheusClient.IsEnabled() {
		var err error
		promStats, err = a.prometheusClient.QueryClassificationStats(ctx)
		if err != nil {
			// Log warning but continue with base stats
			a.logger.Warn("Failed to query Prometheus stats, using base stats only",
				"error", err)
		}
	}

	// Build response with aggregated data
	response := &StatsResponse{
		TotalRequests:     baseStats.TotalRequests,
		TotalClassified:   baseStats.TotalRequests, // Assuming all requests result in classification
		ClassificationRate: a.calculateClassificationRate(baseStats),
		AvgConfidence:     0.0, // Will be calculated from Prometheus if available
		AvgProcessing:     float64(baseStats.AvgResponseTime.Milliseconds()),
		BySeverity:        a.calculateSeverityStats(baseStats, promStats),
		CacheStats:        a.calculateCacheStats(baseStats, promStats),
		LLMStats:          a.calculateLLMStats(baseStats, promStats),
		FallbackStats:     a.calculateFallbackStats(baseStats),
		ErrorStats:        a.calculateErrorStats(baseStats),
		Timestamp:         time.Now(),
	}

	return response, nil
}

// calculateClassificationRate calculates the classification success rate
func (a *StatsAggregator) calculateClassificationRate(stats *services.ClassificationStats) float64 {
	if stats.TotalRequests == 0 {
		return 0.0
	}
	// Classification rate = (total requests - errors) / total requests
	// For now, we assume all requests result in classification
	// This can be enhanced with error tracking
	return 1.0 - stats.FallbackRate
}

// calculateSeverityStats calculates severity statistics
// If promStats is provided, uses Prometheus data; otherwise returns empty stats
func (a *StatsAggregator) calculateSeverityStats(baseStats *services.ClassificationStats, promStats *PrometheusStats) map[string]SeverityStats {
	severityStats := make(map[string]SeverityStats)
	severities := []string{"critical", "warning", "info", "noise"}

	// If Prometheus stats available, use them
	if promStats != nil && len(promStats.BySeverity) > 0 {
		total := int64(0)
		for _, count := range promStats.BySeverity {
			total += count
		}

		for _, severity := range severities {
			count := promStats.BySeverity[severity]
			percentage := 0.0
			if total > 0 {
				percentage = float64(count) / float64(total) * 100.0
			}

			severityStats[severity] = SeverityStats{
				Count:         count,
				AvgConfidence: 0.0, // Would need additional Prometheus query
				Percentage:    percentage,
			}
		}
	} else {
		// Return empty stats if Prometheus not available
		for _, severity := range severities {
			severityStats[severity] = SeverityStats{
				Count:         0,
				AvgConfidence: 0.0,
				Percentage:    0.0,
			}
		}
	}

	return severityStats
}

// calculateCacheStats calculates cache statistics
// If promStats is provided, uses Prometheus data for L1/L2 hits
func (a *StatsAggregator) calculateCacheStats(stats *services.ClassificationStats, promStats *PrometheusStats) CacheStats {
	cacheStats := CacheStats{
		HitRate: stats.CacheHitRate,
		L1Hits:  0, // Will be populated from Prometheus if available
		L2Hits:  0, // Will be populated from Prometheus if available
		Misses:  0, // Will be calculated if we have total requests
	}

	// Use Prometheus data if available
	if promStats != nil {
		cacheStats.L1Hits = promStats.L1CacheHits
		cacheStats.L2Hits = promStats.L2CacheHits
	}

	// Estimate misses from hit rate
	if stats.TotalRequests > 0 {
		cacheStats.Misses = int64(float64(stats.TotalRequests) * (1.0 - stats.CacheHitRate))
	}

	return cacheStats
}

// calculateLLMStats calculates LLM usage statistics
// If promStats is provided, uses Prometheus data for enhanced metrics
func (a *StatsAggregator) calculateLLMStats(stats *services.ClassificationStats, promStats *PrometheusStats) LLMStats {
	llmStats := LLMStats{
		SuccessRate: stats.LLMSuccessRate,
		Failures:    0, // Will be calculated from success rate
		AvgLatencyMs: 0.0, // Will be populated from Prometheus if available
		UsageRate:    0.0, // Will be calculated
	}

	// Use Prometheus data if available
	if promStats != nil {
		llmStats.Requests = promStats.LLMRequests
		llmStats.Failures = promStats.LLMFailures
		llmStats.AvgLatencyMs = promStats.LLMAvgLatency
		llmStats.SuccessRate = promStats.LLMSuccessRate

		if stats.TotalRequests > 0 {
			llmStats.UsageRate = float64(llmStats.Requests) / float64(stats.TotalRequests)
		}
	} else {
		// Estimate LLM requests from cache miss rate
		// LLM is called when cache misses
		if stats.TotalRequests > 0 {
			cacheMissRate := 1.0 - stats.CacheHitRate
			llmStats.Requests = int64(float64(stats.TotalRequests) * cacheMissRate)
			llmStats.UsageRate = cacheMissRate

			// Estimate failures from success rate
			if llmStats.Requests > 0 {
				if stats.LLMSuccessRate > 0 {
					llmStats.Failures = int64(float64(llmStats.Requests) * (1.0 - stats.LLMSuccessRate))
				} else {
					// If success rate is 0, all requests failed
					llmStats.Failures = llmStats.Requests
				}
			}
		}
	}

	return llmStats
}

// calculateFallbackStats calculates fallback classification statistics
func (a *StatsAggregator) calculateFallbackStats(stats *services.ClassificationStats) FallbackStats {
	fallbackStats := FallbackStats{
		Rate:         stats.FallbackRate,
		AvgLatencyMs: 2.0, // Estimated fallback latency (very fast)
	}

	// Calculate used count from rate
	if stats.TotalRequests > 0 {
		fallbackStats.Used = int64(float64(stats.TotalRequests) * stats.FallbackRate)
	}

	return fallbackStats
}

// calculateErrorStats calculates error statistics
func (a *StatsAggregator) calculateErrorStats(stats *services.ClassificationStats) ErrorStats {
	errorStats := ErrorStats{
		LastError:    stats.LastError,
		LastErrorTime: stats.LastErrorTime,
	}

	// Estimate total errors from LLM failures and fallback usage
	// Errors occur when LLM fails and fallback is used
	if stats.TotalRequests > 0 {
		// Simple estimation: errors = LLM failures that required fallback
		// If fallback rate is high, it means LLM failed
		cacheMissRate := 1.0 - stats.CacheHitRate
		if cacheMissRate > 0 {
			llmFailureRate := 1.0 - stats.LLMSuccessRate
			if stats.LLMSuccessRate == 0 && stats.FallbackRate > 0 {
				// All LLM calls failed, use fallback rate as error indicator
				errorStats.Total = int64(float64(stats.TotalRequests) * stats.FallbackRate)
			} else {
				errorStats.Total = int64(float64(stats.TotalRequests) * cacheMissRate * llmFailureRate)
			}
			errorStats.Rate = float64(errorStats.Total) / float64(stats.TotalRequests)
		}
	}

	return errorStats
}

// BuildStatsResponse builds a StatsResponse from ClassificationStats
// This is a helper method that can be used directly in the handler
func BuildStatsResponse(baseStats *services.ClassificationStats) *StatsResponse {
	aggregator := NewStatsAggregator(nil, nil) // Logger not needed for this operation

	// Create a mock context (not used in current implementation)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	if err != nil {
		// Fallback to basic response
		return &StatsResponse{
			TotalRequests:     baseStats.TotalRequests,
			TotalClassified:   baseStats.TotalRequests,
			ClassificationRate: 1.0 - baseStats.FallbackRate,
			AvgProcessing:     float64(baseStats.AvgResponseTime.Milliseconds()),
			BySeverity:        make(map[string]SeverityStats),
			CacheStats:        aggregator.calculateCacheStats(baseStats, nil),
			LLMStats:          aggregator.calculateLLMStats(baseStats, nil),
			FallbackStats:     aggregator.calculateFallbackStats(baseStats),
			ErrorStats:         aggregator.calculateErrorStats(baseStats),
			Timestamp:          time.Now(),
		}
	}

	return response
}
