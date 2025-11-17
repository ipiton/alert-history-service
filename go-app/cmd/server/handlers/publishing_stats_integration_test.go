package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// TestIntegration_StatsEndpoints tests end-to-end flow of stats endpoints.
func TestIntegration_StatsEndpoints(t *testing.T) {
	snapshot := &publishing.MetricsSnapshot{
		Timestamp:           time.Now(),
		Metrics:             createTestMetrics(),
		CollectionDuration:  time.Microsecond * 85,
		AvailableCollectors: []string{"health", "refresh", "discovery", "queue"},
		Errors:              make(map[string]error),
	}
	handler := createTestHandler(snapshot)

	t.Run("API v1 and v2 return consistent data", func(t *testing.T) {
		// Get v1 stats
		reqV1 := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/stats", nil)
		wV1 := httptest.NewRecorder()
		handler.GetStatsV1(wV1, reqV1)

		if wV1.Code != http.StatusOK {
			t.Fatalf("Expected 200 for v1, got %d", wV1.Code)
		}

		var responseV1 StatsResponseV1
		if err := json.NewDecoder(wV1.Body).Decode(&responseV1); err != nil {
			t.Fatalf("Failed to decode v1 response: %v", err)
		}

		// Get v2 stats
		reqV2 := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		wV2 := httptest.NewRecorder()
		handler.GetStats(wV2, reqV2)

		if wV2.Code != http.StatusOK {
			t.Fatalf("Expected 200 for v2, got %d", wV2.Code)
		}

		var responseV2 StatsResponse
		if err := json.NewDecoder(wV2.Body).Decode(&responseV2); err != nil {
			t.Fatalf("Failed to decode v2 response: %v", err)
		}

		// Verify consistency: total targets should match
		if responseV1.TotalTargets != responseV2.System.TotalTargets {
			t.Errorf("Total targets mismatch: v1=%d, v2=%d",
				responseV1.TotalTargets, responseV2.System.TotalTargets)
		}

		// Queue size should match
		if responseV1.QueueSize != responseV2.System.QueueSize {
			t.Errorf("Queue size mismatch: v1=%d, v2=%d",
				responseV1.QueueSize, responseV2.System.QueueSize)
		}
	})

	t.Run("Filter and format work together", func(t *testing.T) {
		// Test filter + Prometheus format
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=type:rootly&format=prometheus", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected Prometheus content type, got %s", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "publishing_stats_total_targets") {
			t.Error("Prometheus format should contain metric names")
		}
	})

	t.Run("Cache headers are set correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		// Verify cache headers
		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "max-age=5, public" {
			t.Errorf("Expected Cache-Control 'max-age=5, public', got '%s'", cacheControl)
		}

		etag := w.Header().Get("ETag")
		if etag == "" {
			t.Error("Expected ETag header")
		}
	})

	t.Run("Conditional request returns 304", func(t *testing.T) {
		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w1 := httptest.NewRecorder()
		handler.GetStats(w1, req1)

		if w1.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w1.Code)
		}

		etag := w1.Header().Get("ETag")
		if etag == "" {
			t.Fatal("Expected ETag from first request")
		}

		// Second request with If-None-Match
		req2 := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		handler.GetStats(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Errorf("Expected 304, got %d", w2.Code)
		}

		if w2.Body.Len() > 0 {
			t.Error("304 response should have no body")
		}
	})
}

// TestIntegration_ErrorHandling tests error handling in integration scenarios.
func TestIntegration_ErrorHandling(t *testing.T) {
	handler := createTestHandler(nil)

	t.Run("Invalid query parameters return 400", func(t *testing.T) {
		testCases := []struct {
			query     string
			expect400 bool
		}{
			{"filter=invalid", true},
			{"filter=type:rootly", false},
			{"group_by=invalid", true},
			{"format=invalid", true},
			{"filter=type:rootly&format=prometheus", false},
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?"+tc.query, nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if tc.expect400 {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 for query '%s', got %d", tc.query, w.Code)
				}
			} else {
				if w.Code == http.StatusBadRequest {
					t.Errorf("Unexpected 400 for query '%s'", tc.query)
				}
			}
		}
	})
}

// TestIntegration_MetricsCollection tests metrics collection integration.
func TestIntegration_MetricsCollection(t *testing.T) {
	// Create a mock collector that simulates real collection
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := createTestMetrics()
			// Add more realistic metrics
			metrics["targets_total"] = 10.0
			metrics["queue_size_total"] = 15.0
			metrics["queue_capacity"] = 1000.0

			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  time.Microsecond * 85,
				AvailableCollectors: []string{"health", "refresh", "discovery", "queue"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	t.Run("Metrics are collected and aggregated correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		var response StatsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify metrics are aggregated
		if response.System.TotalTargets != 10 {
			t.Errorf("Expected 10 total targets, got %d", response.System.TotalTargets)
		}

		if response.System.QueueSize != 15 {
			t.Errorf("Expected queue size 15, got %d", response.System.QueueSize)
		}
	})
}
