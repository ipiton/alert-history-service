package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core"
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

// ===== Integration Tests for ClassifyAlert Endpoint =====

// TestClassifyAlert_Integration_FullFlow tests end-to-end flow with ClassificationService
func TestClassifyAlert_Integration_FullFlow(t *testing.T) {
	mockService := &MockClassificationService{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityCritical,
				Confidence:     0.98,
				Reasoning:      "Integration test classification",
				Recommendations: []string{"Check logs", "Restart service"},
				ProcessingTime: 0.15,
				Metadata: map[string]interface{}{
					"model": "gpt-4",
					"source": "llm",
				},
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "integration-test-123",
			AlertName:   "IntegrationTestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
			Labels: map[string]string{
				"severity": "critical",
				"namespace": "production",
			},
			Annotations: map[string]string{
				"summary": "Integration test alert",
			},
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "integration-test-123"))

	w := httptest.NewRecorder()
	handlers.ClassifyAlert(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response ClassifyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.NotNil(t, response.Result)
	assert.Equal(t, core.SeverityCritical, response.Result.Severity)
	assert.Equal(t, 0.98, response.Result.Confidence)
	assert.Equal(t, "gpt-4", response.Model)
	assert.False(t, response.Timestamp.IsZero())
	assert.NotEmpty(t, response.ProcessingTime)
}

// TestClassifyAlert_Integration_CacheFlow tests cache integration flow
func TestClassifyAlert_Integration_CacheFlow(t *testing.T) {
	cachedResult := &core.ClassificationResult{
		Severity:       core.SeverityWarning,
		Confidence:     0.85,
		Reasoning:      "Cached result",
		ProcessingTime: 0.01,
	}

	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			if fingerprint == "cache-test-123" {
				return cachedResult, nil
			}
			return nil, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "cache-test-123",
			AlertName:   "CacheTestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "cache-test-123"))

	w := httptest.NewRecorder()
	handlers.ClassifyAlert(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ClassifyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify cache was used
	assert.True(t, response.Cached)
	assert.Equal(t, cachedResult.Severity, response.Result.Severity)
	assert.Equal(t, cachedResult.Confidence, response.Result.Confidence)
}

// TestClassifyAlert_Integration_ForceFlow tests force flag integration flow
func TestClassifyAlert_Integration_ForceFlow(t *testing.T) {
	mockService := &MockClassificationService{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityInfo,
				Confidence:     0.75,
				Reasoning:      "Forced classification",
				ProcessingTime: 0.2,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "force-test-123",
			AlertName:   "ForceTestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "force-test-123"))

	w := httptest.NewRecorder()
	handlers.ClassifyAlert(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify cache invalidation was called
	assert.True(t, mockService.invalidateCacheCalled)
	assert.Equal(t, "force-test-123", mockService.invalidateCacheFingerprint)

	var response ClassifyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify force classification (not cached)
	assert.False(t, response.Cached)
	assert.Equal(t, core.SeverityInfo, response.Result.Severity)
}

// TestClassifyAlert_Integration_ErrorHandling tests error handling integration
func TestClassifyAlert_Integration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		classifyFunc   func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error)
		expectedStatus int
	}{
		{
			name: "timeout error",
			classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
				return nil, context.DeadlineExceeded
			},
			expectedStatus: http.StatusGatewayTimeout,
		},
		{
			name: "service unavailable",
			classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
				return nil, errors.New("circuit breaker is open - service unavailable")
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "generic error",
			classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
				return nil, errors.New("classification failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockClassificationService{
				classifyFunc: tt.classifyFunc,
			}
			handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

			reqBody := ClassifyRequest{
				Alert: &core.Alert{
					Fingerprint: "error-test-123",
					AlertName:   "ErrorTestAlert",
					Status:      core.StatusFiring,
					StartsAt:    time.Now(),
				},
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.WithValue(req.Context(), "request_id", "error-test-123"))

			w := httptest.NewRecorder()
			handlers.ClassifyAlert(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestClassifyAlert_Integration_ConcurrentAccess tests concurrent access
func TestClassifyAlert_Integration_ConcurrentAccess(t *testing.T) {
	mockService := &MockClassificationService{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.90,
				Reasoning:      "Concurrent test",
				ProcessingTime: 0.05,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	const numRequests = 50
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			reqBody := ClassifyRequest{
				Alert: &core.Alert{
					Fingerprint: "concurrent-test-" + string(rune('0'+id%10)),
					AlertName:   "ConcurrentTestAlert",
					Status:      core.StatusFiring,
					StartsAt:    time.Now(),
				},
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.WithValue(req.Context(), "request_id", "concurrent-test"))

			w := httptest.NewRecorder()
			handlers.ClassifyAlert(w, req)
			results <- w.Code
		}(i)
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
