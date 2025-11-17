package classification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// TestGetClassificationStats_Integration tests the full handler flow with real service
func TestGetClassificationStats_Integration(t *testing.T) {
	// Create mock service with realistic stats
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  1000,
			CacheHitRate:   0.65,
			LLMSuccessRate: 0.98,
			FallbackRate:   0.02,
			AvgResponseTime: 50 * time.Millisecond,
		},
	}

	// Create handlers
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "test-123"))
	w := httptest.NewRecorder()

	// Execute handler
	handlers.GetClassificationStats(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Verify response body contains expected fields
	var response StatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), response.TotalRequests)
	assert.Equal(t, int64(1000), response.TotalClassified)
	assert.InDelta(t, 0.65, response.CacheStats.HitRate, 0.01)
}

// TestGetClassificationStats_CacheIntegration tests cache behavior
func TestGetClassificationStats_CacheIntegration(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  500,
			CacheHitRate:   0.70,
			LLMSuccessRate: 0.95,
			FallbackRate:   0.05,
			AvgResponseTime: 30 * time.Millisecond,
		},
	}

	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	// Set short TTL for testing
	handlers.statsCache.SetTTL(1 * time.Second)

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "test-cache"))

	// First request - should miss cache
	w1 := httptest.NewRecorder()
	handlers.GetClassificationStats(w1, req)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request immediately - should hit cache
	w2 := httptest.NewRecorder()
	handlers.GetClassificationStats(w2, req)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify responses are identical
	assert.Equal(t, w1.Body.String(), w2.Body.String())

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third request - should miss cache again
	w3 := httptest.NewRecorder()
	handlers.GetClassificationStats(w3, req)
	assert.Equal(t, http.StatusOK, w3.Code)
}

// TestGetClassificationStats_ConcurrentAccess tests concurrent access to handler
func TestGetClassificationStats_ConcurrentAccess(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  2000,
			CacheHitRate:   0.60,
			LLMSuccessRate: 0.99,
			FallbackRate:   0.01,
			AvgResponseTime: 40 * time.Millisecond,
		},
	}

	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	// Run 100 concurrent requests
	const numRequests = 100
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
			req = req.WithContext(context.WithValue(req.Context(), "request_id", "concurrent-test"))
			w := httptest.NewRecorder()
			handlers.GetClassificationStats(w, req)
			results <- w.Code
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(t, numRequests, successCount)
}

// TestGetClassificationStats_GracefulDegradation tests graceful degradation when service unavailable
func TestGetClassificationStats_GracefulDegradation(t *testing.T) {
	// Create handlers without service
	handlers := NewClassificationHandlers(nil, nil)

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "test-degradation"))
	w := httptest.NewRecorder()

	// Execute handler
	handlers.GetClassificationStats(w, req)

	// Should return 200 OK with empty stats (graceful degradation)
	assert.Equal(t, http.StatusOK, w.Code)

	var response StatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(0), response.TotalRequests)
	assert.Equal(t, int64(0), response.TotalClassified)
}
