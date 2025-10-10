package metrics

import (
	"testing"
)

func TestNewBusinessMetrics(t *testing.T) {
	bm := NewBusinessMetrics("test")

	if bm == nil {
		t.Fatal("NewBusinessMetrics returned nil")
	}

	if bm.namespace != "test" {
		t.Errorf("namespace = %q, want %q", bm.namespace, "test")
	}

	// Check that all metrics are initialized
	if bm.AlertsProcessedTotal == nil {
		t.Error("AlertsProcessedTotal not initialized")
	}
	if bm.AlertsEnrichedTotal == nil {
		t.Error("AlertsEnrichedTotal not initialized")
	}
	if bm.AlertsFilteredTotal == nil {
		t.Error("AlertsFilteredTotal not initialized")
	}
	if bm.LLMClassificationsTotal == nil {
		t.Error("LLMClassificationsTotal not initialized")
	}
	if bm.LLMRecommendationsTotal == nil {
		t.Error("LLMRecommendationsTotal not initialized")
	}
	if bm.LLMConfidenceScore == nil {
		t.Error("LLMConfidenceScore not initialized")
	}
	if bm.PublishingSuccessTotal == nil {
		t.Error("PublishingSuccessTotal not initialized")
	}
	if bm.PublishingFailedTotal == nil {
		t.Error("PublishingFailedTotal not initialized")
	}
	if bm.PublishingDurationSeconds == nil {
		t.Error("PublishingDurationSeconds not initialized")
	}
}

func TestBusinessMetrics_AllRecordMethods(t *testing.T) {
	// Use single instance to avoid duplicate registration
	bm := NewBusinessMetrics("test_business")

	t.Run("RecordAlertProcessed", func(t *testing.T) {
		// Should not panic
		bm.RecordAlertProcessed("alertmanager")
		bm.RecordAlertProcessed("webhook")
		bm.RecordAlertProcessed("api")
	})

	t.Run("RecordAlertEnriched", func(t *testing.T) {
		tests := []struct {
			mode    string
			success bool
		}{
			{"enriched", true},
			{"enriched", false},
			{"transparent_recommendations", true},
			{"transparent_recommendations", false},
		}

		for _, tt := range tests {
			// Should not panic
			bm.RecordAlertEnriched(tt.mode, tt.success)
		}
	})

	t.Run("RecordAlertFiltered", func(t *testing.T) {
		tests := []struct {
			result string
			reason string
		}{
			{"allowed", "valid"},
			{"blocked", "test_alert"},
			{"blocked", "noise"},
			{"blocked", "low_confidence"},
		}

		for _, tt := range tests {
			// Should not panic
			bm.RecordAlertFiltered(tt.result, tt.reason)
		}
	})

	t.Run("RecordLLMClassification", func(t *testing.T) {
		tests := []struct {
			severity   string
			confidence string
		}{
			{"critical", "high"},
			{"warning", "medium"},
			{"info", "low"},
		}

		for _, tt := range tests {
			// Should not panic
			bm.RecordLLMClassification(tt.severity, tt.confidence)
		}
	})

	t.Run("RecordLLMRecommendation", func(t *testing.T) {
		// Should not panic
		bm.RecordLLMRecommendation()
		bm.RecordLLMRecommendation()
	})

	t.Run("RecordLLMConfidenceScore", func(t *testing.T) {
		tests := []float64{
			0.5,
			0.75,
			0.85,
			0.95,
			0.99,
			1.0,
		}

		for _, score := range tests {
			// Should not panic
			bm.RecordLLMConfidenceScore(score)
		}
	})

	t.Run("RecordPublishingSuccess", func(t *testing.T) {
		tests := []struct {
			destination string
			duration    float64
		}{
			{"webhook", 0.123},
			{"slack", 0.456},
			{"pagerduty", 0.789},
		}

		for _, tt := range tests {
			// Should not panic
			bm.RecordPublishingSuccess(tt.destination, tt.duration)
		}
	})

	t.Run("RecordPublishingFailure", func(t *testing.T) {
		tests := []struct {
			destination string
			errorType   string
			duration    float64
		}{
			{"webhook", "timeout", 5.0},
			{"slack", "connection_refused", 0.1},
			{"pagerduty", "4xx", 0.5},
			{"webhook", "5xx", 1.0},
		}

		for _, tt := range tests {
			// Should not panic
			bm.RecordPublishingFailure(tt.destination, tt.errorType, tt.duration)
		}
	})
}

func BenchmarkBusinessMetrics_RecordAlertProcessed(b *testing.B) {
	bm := NewBusinessMetrics("bench_business1")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bm.RecordAlertProcessed("alertmanager")
	}
}

func BenchmarkBusinessMetrics_RecordLLMConfidenceScore(b *testing.B) {
	bm := NewBusinessMetrics("bench_business2")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bm.RecordLLMConfidenceScore(0.95)
	}
}

func BenchmarkBusinessMetrics_RecordPublishingSuccess(b *testing.B) {
	bm := NewBusinessMetrics("bench_business3")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bm.RecordPublishingSuccess("webhook", 0.123)
	}
}
