//go:build integration
// +build integration

package publishing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
	"github.com/vitaliisemenov/alert-history/internal/core"
	infrapublishing "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// TestIntegration_GetPublishingMode_V1 tests end-to-end API v1 endpoint
func TestIntegration_GetPublishingMode_V1(t *testing.T) {
	// Setup: Create real ModeManager and DiscoveryManager
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	
	// Add some targets
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Type:    "webhook",
		Enabled: true,
	})
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target2",
		Type:    "slack",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_v1", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	// Start mode manager
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	// Create service and handler
	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "max-age=5, public", w.Header().Get("Cache-Control"))
	assert.NotEmpty(t, w.Header().Get("ETag"))

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "normal", response.Mode)
	assert.True(t, response.TargetsAvailable)
	assert.Equal(t, 2, response.EnabledTargets)
	assert.False(t, response.MetricsOnlyActive)
}

// TestIntegration_GetPublishingMode_V2 tests end-to-end API v2 endpoint
func TestIntegration_GetPublishingMode_V2(t *testing.T) {
	// Setup: Same as v1 but different path
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Type:    "webhook",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_v2", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	// Create request for v2
	req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/mode", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "normal", response.Mode)
	assert.True(t, response.TargetsAvailable)
}

// TestIntegration_ConditionalRequest tests 304 Not Modified response
func TestIntegration_ConditionalRequest(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_conditional", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w1 := httptest.NewRecorder()
	handler.GetPublishingMode(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	etag := w1.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// Second request with If-None-Match header
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	handler.GetPublishingMode(w2, req2)

	// Assert: Should return 304 Not Modified
	assert.Equal(t, http.StatusNotModified, w2.Code)
	assert.Equal(t, etag, w2.Header().Get("ETag"))
	assert.Equal(t, "max-age=5, public", w2.Header().Get("Cache-Control"))
}

// TestIntegration_MetricsOnlyMode tests metrics-only mode detection
func TestIntegration_MetricsOnlyMode(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	// No targets added (all disabled)

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_metrics_only", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	// Trigger mode check
	_, _, _ = modeManager.CheckModeTransition()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "metrics-only", response.Mode)
	assert.False(t, response.TargetsAvailable)
	assert.Equal(t, 0, response.EnabledTargets)
	assert.True(t, response.MetricsOnlyActive)
}

// TestIntegration_FallbackMode tests fallback mode detection (no ModeManager)
func TestIntegration_FallbackMode(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Enabled: true,
	})

	// No ModeManager (nil)
	modeService := apiservices.NewModeService(nil, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "normal", response.Mode)
	assert.True(t, response.TargetsAvailable)
	// Enhanced fields should be omitted (zero values)
	assert.Zero(t, response.TransitionCount)
}

// TestIntegration_ErrorHandling tests error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	// Test with service that returns error
	mockService := &mockModeService{
		modeInfo: nil,
		err:      assert.AnError,
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	// This should handle service error gracefully
	handler.GetPublishingMode(w, req)

	// Should return 500 error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "Internal Server Error", errorResponse.Error)
	assert.Contains(t, errorResponse.Message, "Failed to retrieve")
	assert.NotEmpty(t, errorResponse.RequestID)
}

// TestIntegration_ConcurrentRequests tests concurrent access
func TestIntegration_ConcurrentRequests(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_concurrent", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	// Make 100 concurrent requests
	const numRequests = 100
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
			w := httptest.NewRecorder()
			handler.GetPublishingMode(w, req)
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

// TestIntegration_ETagConsistency tests ETag consistency across requests
func TestIntegration_ETagConsistency(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_etag", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	// Make multiple requests and verify ETag is consistent
	etags := make(map[string]int)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
		w := httptest.NewRecorder()
		handler.GetPublishingMode(w, req)

		etag := w.Header().Get("ETag")
		etags[etag]++
	}

	// All ETags should be the same (mode hasn't changed)
	assert.Len(t, etags, 1, "All ETags should be identical when mode hasn't changed")
}

// TestIntegration_ResponseTime tests response time is acceptable
func TestIntegration_ResponseTime(t *testing.T) {
	logger := slog.Default()
	stubDiscoveryMgr := infrapublishing.NewStubTargetDiscoveryManager(logger)
	stubDiscoveryMgr.AddTarget(&core.PublishingTarget{
		Name:    "target1",
		Enabled: true,
	})

	modeMetrics := infrapublishing.NewPublishingModeMetrics("test_perf", "publishing")
	modeManager := infrapublishing.NewModeManager(stubDiscoveryMgr, logger, modeMetrics)
	
	ctx := context.Background()
	_ = modeManager.Start(ctx)
	defer modeManager.Stop()

	modeService := apiservices.NewModeService(modeManager, stubDiscoveryMgr, nil)
	handler := NewPublishingModeHandler(modeService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	handler.GetPublishingMode(w, req)
	duration := time.Since(start)

	// Response should be fast (< 10ms for 150% target, but < 5ms is ideal)
	assert.Less(t, duration.Milliseconds(), int64(10), "Response time should be < 10ms")
	assert.Equal(t, http.StatusOK, w.Code)
}

