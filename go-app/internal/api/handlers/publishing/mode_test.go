package publishing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
)

// mockModeService is a mock implementation of ModeService for testing
type mockModeService struct {
	modeInfo *apiservices.ModeInfo
	err      error
}

func (m *mockModeService) GetCurrentModeInfo(ctx context.Context) (*apiservices.ModeInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.modeInfo, nil
}

func TestPublishingModeHandler_GetPublishingMode_Success(t *testing.T) {
	// Setup
	modeInfo := &apiservices.ModeInfo{
		Mode:              "normal",
		TargetsAvailable:  true,
		EnabledTargets:    5,
		MetricsOnlyActive: false,
		TransitionCount:   12,
		CurrentModeDurationSeconds: 3600.5,
		LastTransitionTime:        time.Now(),
		LastTransitionReason:      "targets_available",
	}

	mockService := &mockModeService{modeInfo: modeInfo}
	handler := NewPublishingModeHandler(mockService, nil)

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

	// Parse response
	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "normal", response.Mode)
	assert.True(t, response.TargetsAvailable)
	assert.Equal(t, 5, response.EnabledTargets)
	assert.False(t, response.MetricsOnlyActive)
	assert.Equal(t, int64(12), response.TransitionCount)
}

func TestPublishingModeHandler_GetPublishingMode_MetricsOnly(t *testing.T) {
	// Setup
	modeInfo := &apiservices.ModeInfo{
		Mode:              "metrics-only",
		TargetsAvailable:  false,
		EnabledTargets:    0,
		MetricsOnlyActive: true,
		TransitionCount:   13,
		CurrentModeDurationSeconds: 120.3,
		LastTransitionTime:        time.Now(),
		LastTransitionReason:      "no_enabled_targets",
	}

	mockService := &mockModeService{modeInfo: modeInfo}
	handler := NewPublishingModeHandler(mockService, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/mode", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "metrics-only", response.Mode)
	assert.False(t, response.TargetsAvailable)
	assert.Equal(t, 0, response.EnabledTargets)
	assert.True(t, response.MetricsOnlyActive)
}

func TestPublishingModeHandler_GetPublishingMode_ServiceError(t *testing.T) {
	// Setup
	mockService := &mockModeService{err: assert.AnError}
	handler := NewPublishingModeHandler(mockService, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "Internal Server Error", errorResponse.Error)
	assert.Contains(t, errorResponse.Message, "Failed to retrieve mode information")
	assert.NotEmpty(t, errorResponse.RequestID)
}

func TestPublishingModeHandler_GetPublishingMode_InvalidMethod(t *testing.T) {
	// Setup
	modeInfo := &apiservices.ModeInfo{Mode: "normal"}
	mockService := &mockModeService{modeInfo: modeInfo}
	handler := NewPublishingModeHandler(mockService, nil)

	// Create request with POST method (invalid)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "Method Not Allowed", errorResponse.Error)
}

func TestPublishingModeHandler_GetPublishingMode_ConditionalRequest(t *testing.T) {
	// Setup
	modeInfo := &apiservices.ModeInfo{
		Mode:              "normal",
		EnabledTargets:    5,
		TransitionCount:   12,
	}
	mockService := &mockModeService{modeInfo: modeInfo}
	handler := NewPublishingModeHandler(mockService, nil)

	// Generate ETag
	etag := handler.generateETag(modeInfo)

	// Create request with If-None-Match header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	req.Header.Set("If-None-Match", etag)
	w := httptest.NewRecorder()

	// Execute
	handler.GetPublishingMode(w, req)

	// Assert
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Equal(t, etag, w.Header().Get("ETag"))
	assert.Equal(t, "max-age=5, public", w.Header().Get("Cache-Control"))
}

func TestPublishingModeHandler_GenerateETag(t *testing.T) {
	handler := NewPublishingModeHandler(nil, nil)

	tests := []struct {
		name     string
		modeInfo *apiservices.ModeInfo
		expected string
	}{
		{
			name: "normal mode",
			modeInfo: &apiservices.ModeInfo{
				Mode:            "normal",
				EnabledTargets:  5,
				TransitionCount: 12,
			},
			expected: `"normal-5-12"`,
		},
		{
			name: "metrics-only mode",
			modeInfo: &apiservices.ModeInfo{
				Mode:            "metrics-only",
				EnabledTargets:  0,
				TransitionCount: 13,
			},
			expected: `"metrics-only-0-13"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := handler.generateETag(tt.modeInfo)
			assert.Equal(t, tt.expected, etag)
		})
	}
}

func TestPublishingModeHandler_SetCacheHeaders(t *testing.T) {
	handler := NewPublishingModeHandler(nil, nil)
	modeInfo := &apiservices.ModeInfo{
		Mode:            "normal",
		EnabledTargets:  5,
		TransitionCount: 12,
	}
	etag := handler.generateETag(modeInfo)

	w := httptest.NewRecorder()
	handler.setCacheHeaders(w, modeInfo, etag)

	assert.Equal(t, "max-age=5, public", w.Header().Get("Cache-Control"))
	assert.Equal(t, etag, w.Header().Get("ETag"))
}

func TestPublishingModeHandler_SendJSON(t *testing.T) {
	handler := NewPublishingModeHandler(nil, nil)

	data := map[string]string{"test": "value"}
	w := httptest.NewRecorder()

	handler.sendJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "value", response["test"])
}

func TestPublishingModeHandler_SendError(t *testing.T) {
	handler := NewPublishingModeHandler(nil, nil)

	w := httptest.NewRecorder()
	handler.sendError(w, http.StatusInternalServerError, "Test error", "test-request-id")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "Internal Server Error", errorResponse.Error)
	assert.Equal(t, "Test error", errorResponse.Message)
	assert.Equal(t, "test-request-id", errorResponse.RequestID)
	assert.False(t, errorResponse.Timestamp.IsZero())
}

