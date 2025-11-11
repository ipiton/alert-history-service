package publishing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

func BenchmarkTriggerEvent(b *testing.B) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(EventResponse{
			Status:   "success",
			DedupKey: "test-dedup",
		})
	}))
	defer server.Close()

	config := PagerDutyClientConfig{
		BaseURL: server.URL,
	}
	client := NewPagerDutyEventsClient(config, slog.Default())

	req := &TriggerEventRequest{
		RoutingKey: "test-key",
		Payload: TriggerEventPayload{
			Summary:  "Benchmark alert",
			Source:   "benchmark",
			Severity: "critical",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.TriggerEvent(ctx, req)
	}
}

func BenchmarkAcknowledgeEvent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(EventResponse{Status: "success"})
	}))
	defer server.Close()

	config := PagerDutyClientConfig{BaseURL: server.URL}
	client := NewPagerDutyEventsClient(config, slog.Default())

	req := &AcknowledgeEventRequest{
		RoutingKey: "test-key",
		DedupKey:   "test-dedup",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.AcknowledgeEvent(ctx, req)
	}
}

func BenchmarkResolveEvent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(EventResponse{Status: "success"})
	}))
	defer server.Close()

	config := PagerDutyClientConfig{BaseURL: server.URL}
	client := NewPagerDutyEventsClient(config, slog.Default())

	req := &ResolveEventRequest{
		RoutingKey: "test-key",
		DedupKey:   "test-dedup",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ResolveEvent(ctx, req)
	}
}

func BenchmarkSendChangeEvent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(ChangeEventResponse{Status: "success"})
	}))
	defer server.Close()

	config := PagerDutyClientConfig{BaseURL: server.URL}
	client := NewPagerDutyEventsClient(config, slog.Default())

	req := &ChangeEventRequest{
		RoutingKey: "test-key",
		Payload: ChangeEventPayload{
			Summary: "Deployment v1.0.0",
			Source:  "ci-cd",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.SendChangeEvent(ctx, req)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewEventKeyCache(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("fingerprint", "dedup-key")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewEventKeyCache(24 * time.Hour)
	cache.Set("fingerprint", "dedup-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("fingerprint")
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	cache := NewEventKeyCache(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent")
	}
}

func BenchmarkPublisher_Publish_FiringAlert(b *testing.B) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(EventResponse{
			Status:   "success",
			DedupKey: "test-dedup",
		})
	}))
	defer server.Close()

	// Setup
	config := PagerDutyClientConfig{BaseURL: server.URL}
	client := NewPagerDutyEventsClient(config, slog.Default())
	cache := NewEventKeyCache(24 * time.Hour)
	metrics := NewPagerDutyMetrics()

	// Simple formatter
	formatter := &simpleFormatter{}

	publisher := NewEnhancedPagerDutyPublisher(
		client,
		cache,
		metrics,
		formatter,
		slog.Default(),
	)

	enrichedAlert := &core.EnrichedAlert{
		Alert: &core.Alert{
			Fingerprint: "test-fp",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			Labels:      map[string]string{"severity": "critical"},
			StartsAt:    time.Now(),
		},
		Classification: &core.AlertClassification{
			Severity:   core.SeverityCritical,
			Confidence: 0.95,
		},
	}

	target := &core.PublishingTarget{
		Headers: map[string]string{"routing_key": "test-key"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		publisher.Publish(ctx, enrichedAlert, target)
	}
}

// Simple formatter for benchmarks
type simpleFormatter struct{}

func (f *simpleFormatter) FormatAlert(ctx context.Context, enrichedAlert *core.EnrichedAlert, format core.PublishingFormat) (map[string]any, error) {
	return map[string]any{
		"summary":  "Test alert",
		"severity": "critical",
		"source":   "benchmark",
		"payload": map[string]any{
			"custom_details": map[string]any{
				"fingerprint": enrichedAlert.Alert.Fingerprint,
			},
		},
	}, nil
}
