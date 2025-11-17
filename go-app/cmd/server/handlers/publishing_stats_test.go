package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// ============================================================================
// Test Helpers & Mocks
// ============================================================================

// MockCollector implements PublishingMetricsCollector for testing.
type MockCollector struct {
	snapshot *publishing.MetricsSnapshot
}

func (m *MockCollector) RegisterCollector(collector publishing.MetricsCollector) {}

func (m *MockCollector) CollectAll(ctx context.Context) *publishing.MetricsSnapshot {
	if m.snapshot != nil {
		return m.snapshot
	}
	return &publishing.MetricsSnapshot{
		Timestamp:           time.Now(),
		Metrics:             make(map[string]float64),
		CollectionDuration:  time.Microsecond * 100,
		AvailableCollectors: []string{"health", "refresh"},
		Errors:              make(map[string]error),
	}
}

func (m *MockCollector) GetCollectorNames() []string {
	return []string{"health", "refresh"}
}

func (m *MockCollector) CollectorCount() int {
	return 2
}

// createTestHandler creates handler with mock collector.
func createTestHandler(snapshot *publishing.MetricsSnapshot) *PublishingStatsHandler {
	mockCollector := &MockCollector{snapshot: snapshot}
	trendDetector := publishing.NewTrendDetector() // Add trend detector
	return &PublishingStatsHandler{
		collector:     mockCollector,
		trendDetector: trendDetector,
		logger:        slog.Default(),
	}
}

// createTestMetrics creates sample metrics for testing.
func createTestMetrics() map[string]float64 {
	return map[string]float64{
		"targets_total":                                         10.0,
		"health_status{target=\"rootly-prod\",type=\"rootly\"}": 1.0, // healthy
		"success_rate{target=\"rootly-prod\"}":                  99.5,
		"consecutive_failures{target=\"rootly-prod\"}":          0.0,
		"jobs_processed_total{target=\"rootly-prod\",state=\"succeeded\"}": 995.0,
		"jobs_processed_total{target=\"rootly-prod\",state=\"failed\"}":    5.0,
		"queue_size_total":                                                  15.0,
		"queue_capacity":                                                    1000.0,
		"jobs_submitted_total":                                              1000.0,
		"jobs_completed_total":                                              950.0,
	}
}

// ============================================================================
// Endpoint Tests
// ============================================================================

// TestGetMetrics tests GET /api/v2/publishing/metrics endpoint.
func TestGetMetrics(t *testing.T) {
	t.Run("Returns metrics snapshot successfully", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh", "discovery"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/metrics", nil)
		w := httptest.NewRecorder()

		handler.GetMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response MetricsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.MetricsCount != len(snapshot.Metrics) {
			t.Errorf("Expected %d metrics, got %d", len(snapshot.Metrics), response.MetricsCount)
		}

		if len(response.AvailableCollectors) != 3 {
			t.Errorf("Expected 3 collectors, got %d", len(response.AvailableCollectors))
		}
	})

	t.Run("Rejects non-GET requests", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/metrics", nil)
		w := httptest.NewRecorder()

		handler.GetMetrics(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestGetStats tests GET /api/v2/publishing/stats endpoint.
func TestGetStats(t *testing.T) {
	t.Run("Returns aggregated stats successfully", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response StatsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.System.TotalTargets != 10 {
			t.Errorf("Expected 10 total targets, got %d", response.System.TotalTargets)
		}

		if response.System.QueueSize != 15 {
			t.Errorf("Expected queue size 15, got %d", response.System.QueueSize)
		}
	})

	t.Run("Rejects non-GET requests", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("Supports query parameters", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=type:rootly", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Validates invalid filter parameter", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=invalid", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Supports Prometheus format", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?format=prometheus", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected Prometheus content type, got %s", contentType)
		}
	})
}

// TestGetStatsV1 tests GET /api/v1/publishing/stats endpoint.
func TestGetStatsV1(t *testing.T) {
	t.Run("Returns v1 stats successfully", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStatsV1(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response StatsResponseV1
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.TotalTargets == 0 && len(createTestMetrics()) > 0 {
			t.Error("Expected non-zero total targets")
		}
	})

	t.Run("Rejects non-GET requests", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStatsV1(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestGetStats_HTTPCaching tests HTTP caching functionality.
func TestGetStats_HTTPCaching(t *testing.T) {
	t.Run("Sets Cache-Control header", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "max-age=5, public" {
			t.Errorf("Expected Cache-Control 'max-age=5, public', got '%s'", cacheControl)
		}
	})

	t.Run("Sets ETag header", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		etag := w.Header().Get("ETag")
		if etag == "" {
			t.Error("Expected ETag header to be set")
		}
		if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
			t.Errorf("Expected ETag to be quoted, got '%s'", etag)
		}
	})

	t.Run("Returns 304 Not Modified for matching ETag", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		// First request to get ETag
		req1 := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w1 := httptest.NewRecorder()
		handler.GetStats(w1, req1)

		if w1.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w1.Code)
		}

		etag := w1.Header().Get("ETag")
		if etag == "" {
			t.Fatal("Expected ETag header from first request")
		}

		// Second request with If-None-Match header
		req2 := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		handler.GetStats(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Errorf("Expected status 304, got %d", w2.Code)
		}

		// Verify ETag is still present
		if w2.Header().Get("ETag") != etag {
			t.Error("ETag should match in 304 response")
		}

		// Verify no body content
		if w2.Body.Len() > 0 {
			t.Error("304 response should have no body")
		}
	})

	t.Run("Returns 200 OK for non-matching ETag", func(t *testing.T) {
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             createTestMetrics(),
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		req.Header.Set("If-None-Match", `"old-etag-value"`)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGetStats_QueryParameters tests query parameter functionality.
func TestGetStats_QueryParameters(t *testing.T) {
	t.Run("Validates group_by parameter", func(t *testing.T) {
		handler := createTestHandler(nil)

		testCases := []struct {
			groupBy     string
			expectError bool
		}{
			{"type", false},
			{"status", false},
			{"target", false},
			{"invalid", true},
			{"", false}, // Empty is allowed
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?group_by=%s", tc.groupBy), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if tc.expectError {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400 for group_by='%s', got %d", tc.groupBy, w.Code)
				}
			} else {
				if w.Code == http.StatusBadRequest {
					t.Errorf("Unexpected status 400 for group_by='%s'", tc.groupBy)
				}
			}
		}
	})

	t.Run("Validates format parameter", func(t *testing.T) {
		handler := createTestHandler(nil)

		testCases := []struct {
			format      string
			expectError bool
		}{
			{"json", false},
			{"prometheus", false},
			{"invalid", true},
			{"", false}, // Empty defaults to json
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?format=%s", tc.format), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if tc.expectError {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400 for format='%s', got %d", tc.format, w.Code)
				}
			} else {
				if w.Code == http.StatusBadRequest {
					t.Errorf("Unexpected status 400 for format='%s'", tc.format)
				}
			}
		}
	})

	t.Run("Applies filter correctly", func(t *testing.T) {
		metrics := createTestMetrics()
		// Add metrics with different types
		metrics["target_type_rootly"] = 5.0
		metrics["target_type_slack"] = 3.0

		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             metrics,
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=type:rootly", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response StatsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response structure is valid
		if response.System.TotalTargets < 0 {
			t.Error("Total targets should be non-negative")
		}
	})
}

// TestGetHealth tests GET /api/v2/publishing/health endpoint.
func TestGetHealth(t *testing.T) {
	t.Run("Returns healthy status", func(t *testing.T) {
		metrics := createTestMetrics()
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             metrics,
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health", "refresh"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/health", nil)
		w := httptest.NewRecorder()

		handler.GetHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response PublishingHealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if len(response.Checks) < 2 {
			t.Errorf("Expected at least 2 checks, got %d", len(response.Checks))
		}
	})

	t.Run("Returns degraded status with errors", func(t *testing.T) {
		metrics := createTestMetrics()
		// Add unhealthy target
		metrics["health_status{target=\"slack-prod\",type=\"slack\"}"] = 3.0 // unhealthy

		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             metrics,
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/health", nil)
		w := httptest.NewRecorder()

		handler.GetHealth(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		var response PublishingHealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "degraded" {
			t.Errorf("Expected status 'degraded', got '%s'", response.Status)
		}
	})
}

// TestGetTargetStats tests GET /api/v2/publishing/stats/{target} endpoint.
func TestGetTargetStats(t *testing.T) {
	t.Run("Returns target stats successfully", func(t *testing.T) {
		metrics := createTestMetrics()
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             metrics,
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats/rootly-prod", nil)
		w := httptest.NewRecorder()

		handler.GetTargetStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response TargetStatsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.TargetName != "rootly-prod" {
			t.Errorf("Expected target 'rootly-prod', got '%s'", response.TargetName)
		}

		if response.Health.Status != "healthy" {
			t.Errorf("Expected health 'healthy', got '%s'", response.Health.Status)
		}

		if response.Health.SuccessRate != 99.5 {
			t.Errorf("Expected success rate 99.5, got %f", response.Health.SuccessRate)
		}
	})

	t.Run("Returns 404 for unknown target", func(t *testing.T) {
		metrics := createTestMetrics()
		snapshot := &publishing.MetricsSnapshot{
			Timestamp:           time.Now(),
			Metrics:             metrics,
			CollectionDuration:  time.Microsecond * 85,
			AvailableCollectors: []string{"health"},
			Errors:              make(map[string]error),
		}
		handler := createTestHandler(snapshot)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats/unknown-target", nil)
		w := httptest.NewRecorder()

		handler.GetTargetStats(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("Returns 400 for missing target name", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats/", nil)
		w := httptest.NewRecorder()

		handler.GetTargetStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestGetTrends tests GET /api/v2/publishing/trends endpoint.
func TestGetTrends(t *testing.T) {
	t.Run("Returns trends analysis successfully", func(t *testing.T) {
		handler := createTestHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/trends", nil)
		w := httptest.NewRecorder()

		handler.GetTrends(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response TrendsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have trends data
		if response.Trends.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}

		// Should have summary
		if response.Summary == "" {
			t.Error("Expected non-empty summary")
		}
	})

	t.Run("Rejects non-GET requests", func(t *testing.T) {
		handler := createTestHandler(nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/trends", nil)
		w := httptest.NewRecorder()

		handler.GetTrends(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests helper functions.
func TestHelperFunctions(t *testing.T) {
	t.Run("countHealthyTargets", func(t *testing.T) {
		metrics := map[string]float64{
			"health_status{target=\"target1\",type=\"rootly\"}": 1.0, // healthy
			"health_status{target=\"target2\",type=\"slack\"}":  1.0, // healthy
			"health_status{target=\"target3\",type=\"slack\"}":  3.0, // unhealthy
		}
		count := countHealthyTargets(metrics)
		if count != 2 {
			t.Errorf("Expected 2 healthy targets, got %d", count)
		}
	})

	t.Run("countUnhealthyTargets", func(t *testing.T) {
		metrics := map[string]float64{
			"health_status{target=\"target1\",type=\"rootly\"}": 1.0, // healthy
			"health_status{target=\"target2\",type=\"slack\"}":  3.0, // unhealthy
			"health_status{target=\"target3\",type=\"slack\"}":  3.0, // unhealthy
		}
		count := countUnhealthyTargets(metrics)
		if count != 2 {
			t.Errorf("Expected 2 unhealthy targets, got %d", count)
		}
	})

	t.Run("calculateSuccessRate", func(t *testing.T) {
		metrics := map[string]float64{
			"jobs_submitted_total": 1000.0,
			"jobs_completed_total": 950.0,
		}
		rate := calculateSuccessRate(metrics)
		if rate != 95.0 {
			t.Errorf("Expected success rate 95.0%%, got %f%%", rate)
		}
	})

	t.Run("extractTargetHealthStatus", func(t *testing.T) {
		metrics := map[string]float64{
			"health_status{target=\"rootly-prod\",type=\"rootly\"}": 1.0,
		}
		status := extractTargetHealthStatus(metrics, "rootly-prod")
		if status != "healthy" {
			t.Errorf("Expected 'healthy', got '%s'", status)
		}
	})
}
