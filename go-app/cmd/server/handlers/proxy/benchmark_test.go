package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Comprehensive benchmarks for performance testing

// BenchmarkProxyWebhookRequest_Marshal benchmarks request marshaling
func BenchmarkProxyWebhookRequest_Marshal(b *testing.B) {
	req := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "warning",
					"instance":  "server-01",
				},
				Annotations: map[string]string{
					"summary": "Test alert",
				},
				StartsAt: time.Now(),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkProxyWebhookRequest_Unmarshal benchmarks request unmarshaling
func BenchmarkProxyWebhookRequest_Unmarshal(b *testing.B) {
	req := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "warning",
				},
				StartsAt: time.Now(),
			},
		},
	}

	data, _ := json.Marshal(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed ProxyWebhookRequest
		_ = json.Unmarshal(data, &parsed)
	}
}

// BenchmarkAlertPayload_ConvertToAlert_Small benchmarks conversion with minimal fields
func BenchmarkAlertPayload_ConvertToAlert_Small(b *testing.B) {
	payload := AlertPayload{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "Test",
		},
		StartsAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = payload.ConvertToAlert()
	}
}

// BenchmarkAlertPayload_ConvertToAlert_Large benchmarks conversion with many fields
func BenchmarkAlertPayload_ConvertToAlert_Large(b *testing.B) {
	payload := AlertPayload{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "ComplexAlert",
			"severity":  "critical",
			"instance":  "server-01",
			"namespace": "production",
			"cluster":   "us-east-1",
			"job":       "api-server",
			"env":       "prod",
			"team":      "platform",
		},
		Annotations: map[string]string{
			"summary":     "Complex alert with many annotations",
			"description": "Long description text",
			"runbook":     "https://runbook.example.com",
			"dashboard":   "https://grafana.example.com",
		},
		StartsAt:     time.Now(),
		GeneratorURL: "http://prometheus:9090/alerts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = payload.ConvertToAlert()
	}
}

// BenchmarkProxyWebhookResponse_Marshal benchmarks response marshaling
func BenchmarkProxyWebhookResponse_Marshal(b *testing.B) {
	resp := ProxyWebhookResponse{
		Status:    "success",
		Message:   "All alerts processed",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:  10,
			TotalProcessed: 10,
			TotalClassified: 10,
			TotalPublished: 10,
		},
		ProcessingTime: 150 * time.Millisecond,
		AlertResults: []AlertProcessingResult{
			{
				Fingerprint: "abc123",
				AlertName:   "TestAlert",
				Status:      "success",
				Classification: &ClassificationResult{
					Severity:   "warning",
					Category:   "test",
					Confidence: 0.85,
				},
				ClassificationTime: 50 * time.Millisecond,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}

// BenchmarkBatchProcessing_10Alerts benchmarks processing 10 alerts
func BenchmarkBatchProcessing_10Alerts(b *testing.B) {
	alerts := make([]AlertPayload, 10)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{"alertname": "TestAlert"},
			StartsAt: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, alert := range alerts {
			_, _ = alert.ConvertToAlert()
		}
	}
}

// BenchmarkBatchProcessing_50Alerts benchmarks processing 50 alerts
func BenchmarkBatchProcessing_50Alerts(b *testing.B) {
	alerts := make([]AlertPayload, 50)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{"alertname": "TestAlert"},
			StartsAt: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, alert := range alerts {
			_, _ = alert.ConvertToAlert()
		}
	}
}

// BenchmarkBatchProcessing_100Alerts benchmarks processing 100 alerts
func BenchmarkBatchProcessing_100Alerts(b *testing.B) {
	alerts := make([]AlertPayload, 100)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{"alertname": "TestAlert"},
			StartsAt: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, alert := range alerts {
			_, _ = alert.ConvertToAlert()
		}
	}
}

// BenchmarkClassificationResult_ConfidenceBucket_High benchmarks high confidence
func BenchmarkClassificationResult_ConfidenceBucket_High(b *testing.B) {
	cr := ClassificationResult{Confidence: 0.95}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cr.ConfidenceBucket()
	}
}

// BenchmarkClassificationResult_ConfidenceBucket_Medium benchmarks medium confidence
func BenchmarkClassificationResult_ConfidenceBucket_Medium(b *testing.B) {
	cr := ClassificationResult{Confidence: 0.65}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cr.ConfidenceBucket()
	}
}

// BenchmarkClassificationResult_ConfidenceBucket_Low benchmarks low confidence
func BenchmarkClassificationResult_ConfidenceBucket_Low(b *testing.B) {
	cr := ClassificationResult{Confidence: 0.25}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cr.ConfidenceBucket()
	}
}

// BenchmarkJSON_Encode benchmarks JSON encoding
func BenchmarkJSON_Encode(b *testing.B) {
	data := ProxyWebhookRequest{
		Receiver: "test",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "Test"},
				StartsAt: time.Now(),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(data)
	}
}

// BenchmarkJSON_Decode benchmarks JSON decoding
func BenchmarkJSON_Decode(b *testing.B) {
	data := ProxyWebhookRequest{
		Receiver: "test",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "Test"},
				StartsAt: time.Now(),
			},
		},
	}

	jsonData, _ := json.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed ProxyWebhookRequest
		_ = json.NewDecoder(bytes.NewReader(jsonData)).Decode(&parsed)
	}
}

// BenchmarkConfigValidation benchmarks config validation
func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultProxyWebhookConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

// BenchmarkProxyWebhookConfig_Creation benchmarks config creation
func BenchmarkProxyWebhookConfig_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultProxyWebhookConfig()
	}
}

// BenchmarkErrorResponse_Creation benchmarks error response creation
func BenchmarkErrorResponse_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ErrorResponse{
			Error: ErrorDetail{
				Code:      ErrCodeValidation,
				Message:   "Test error",
				Timestamp: time.Now(),
				RequestID: "test",
			},
		}
	}
}

// BenchmarkTargetPublishingResult_Creation benchmarks target result creation
func BenchmarkTargetPublishingResult_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TargetPublishingResult{
			TargetName:     "test-target",
			TargetType:     "slack",
			Success:        true,
			StatusCode:     200,
			ProcessingTime: 100 * time.Millisecond,
		}
	}
}

// BenchmarkAlertProcessingResult_Creation benchmarks alert result creation
func BenchmarkAlertProcessingResult_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AlertProcessingResult{
			Fingerprint: "abc123",
			AlertName:   "TestAlert",
			Status:      "success",
			Classification: &ClassificationResult{
				Severity:   "warning",
				Confidence: 0.85,
			},
			FilterAction: string(FilterActionAllow),
		}
	}
}

// BenchmarkProxyWebhookResponse_Aggregation benchmarks response aggregation
func BenchmarkProxyWebhookResponse_Aggregation(b *testing.B) {
	results := make([]AlertProcessingResult, 10)
	for i := range results {
		results[i] = AlertProcessingResult{
			Fingerprint: "abc123",
			AlertName:   "TestAlert",
			Status:      "success",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summary := AlertsProcessingSummary{
			TotalReceived: len(results),
		}

		for _, result := range results {
			if result.Status == "success" {
				summary.TotalProcessed++
			}
		}
	}
}

// BenchmarkMemoryAllocation_SmallRequest benchmarks memory allocation for small request
func BenchmarkMemoryAllocation_SmallRequest(b *testing.B) {
	req := ProxyWebhookRequest{
		Receiver: "test",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "Test"},
				StartsAt: time.Now(),
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkMemoryAllocation_LargeRequest benchmarks memory allocation for large request
func BenchmarkMemoryAllocation_LargeRequest(b *testing.B) {
	alerts := make([]AlertPayload, 100)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "Test",
				"severity":  "warning",
				"instance":  "server-01",
			},
			StartsAt: time.Now(),
		}
	}

	req := ProxyWebhookRequest{
		Receiver: "test",
		Status:   "firing",
		Alerts:   alerts,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkParallelProcessing benchmarks parallel alert processing simulation
func BenchmarkParallelProcessing(b *testing.B) {
	alerts := make([]AlertPayload, 10)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{"alertname": "Test"},
			StartsAt: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate parallel processing
		results := make(chan *core.Alert, len(alerts))
		for _, alert := range alerts {
			go func(a AlertPayload) {
				converted, _ := a.ConvertToAlert()
				results <- converted
			}(alert)
		}

		for range alerts {
			<-results
		}
	}
}

// BenchmarkContextWithTimeout benchmarks context creation with timeout
func BenchmarkContextWithTimeout(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		cancel()
		_ = timeoutCtx
	}
}

// BenchmarkFilterAction_Comparison benchmarks filter action comparison
func BenchmarkFilterAction_Comparison(b *testing.B) {
	action := FilterActionAllow

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action == FilterActionAllow
	}
}

// BenchmarkTimestampGeneration benchmarks timestamp generation
func BenchmarkTimestampGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = time.Now()
	}
}

// BenchmarkDurationCalculation benchmarks duration calculation
func BenchmarkDurationCalculation(b *testing.B) {
	start := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = time.Since(start)
	}
}

// BenchmarkMapAccess benchmarks map access performance
func BenchmarkMapAccess(b *testing.B) {
	labels := map[string]string{
		"alertname": "Test",
		"severity":  "warning",
		"instance":  "server-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = labels["alertname"]
		_ = labels["severity"]
	}
}

// BenchmarkMapIteration benchmarks map iteration performance
func BenchmarkMapIteration(b *testing.B) {
	labels := map[string]string{
		"alertname": "Test",
		"severity":  "warning",
		"instance":  "server-01",
		"namespace": "production",
		"cluster":   "us-east-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k, v := range labels {
			_ = k
			_ = v
		}
	}
}

// BenchmarkStringConcatenation benchmarks string concatenation
func BenchmarkStringConcatenation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = "alert-" + "fingerprint-" + "123"
	}
}

// BenchmarkSliceAppend benchmarks slice append performance
func BenchmarkSliceAppend(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make([]AlertProcessingResult, 0, 10)
		for j := 0; j < 10; j++ {
			results = append(results, AlertProcessingResult{
				Fingerprint: "test",
				AlertName:   "Test",
				Status:      "success",
			})
		}
	}
}
