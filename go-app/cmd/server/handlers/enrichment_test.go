package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// mockEnrichmentManager implements services.EnrichmentModeManager for testing
type mockEnrichmentManager struct {
	getMode           func(ctx context.Context) (services.EnrichmentMode, error)
	getModeWithSource func(ctx context.Context) (services.EnrichmentMode, string, error)
	setMode           func(ctx context.Context, mode services.EnrichmentMode) error
	validateMode      func(mode services.EnrichmentMode) error
	getStats          func(ctx context.Context) (*services.EnrichmentStats, error)
	refreshCache      func(ctx context.Context) error
}

func (m *mockEnrichmentManager) GetMode(ctx context.Context) (services.EnrichmentMode, error) {
	if m.getMode != nil {
		return m.getMode(ctx)
	}
	return services.EnrichmentModeEnriched, nil
}

func (m *mockEnrichmentManager) GetModeWithSource(ctx context.Context) (services.EnrichmentMode, string, error) {
	if m.getModeWithSource != nil {
		return m.getModeWithSource(ctx)
	}
	return services.EnrichmentModeEnriched, "default", nil
}

func (m *mockEnrichmentManager) SetMode(ctx context.Context, mode services.EnrichmentMode) error {
	if m.setMode != nil {
		return m.setMode(ctx, mode)
	}
	return nil
}

func (m *mockEnrichmentManager) ValidateMode(mode services.EnrichmentMode) error {
	if m.validateMode != nil {
		return m.validateMode(mode)
	}
	return nil
}

func (m *mockEnrichmentManager) GetStats(ctx context.Context) (*services.EnrichmentStats, error) {
	if m.getStats != nil {
		return m.getStats(ctx)
	}
	return &services.EnrichmentStats{}, nil
}

func (m *mockEnrichmentManager) RefreshCache(ctx context.Context) error {
	if m.refreshCache != nil {
		return m.refreshCache(ctx)
	}
	return nil
}

// TestNewEnrichmentHandlers tests the constructor
func TestNewEnrichmentHandlers(t *testing.T) {
	t.Run("creates handlers with logger", func(t *testing.T) {
		mockManager := &mockEnrichmentManager{}
		logger := slog.Default()

		handlers := NewEnrichmentHandlers(mockManager, logger)
		require.NotNil(t, handlers)
		assert.Equal(t, mockManager, handlers.manager)
		assert.Equal(t, logger, handlers.logger)
	})

	t.Run("creates handlers with nil logger", func(t *testing.T) {
		mockManager := &mockEnrichmentManager{}

		handlers := NewEnrichmentHandlers(mockManager, nil)
		require.NotNil(t, handlers)
		assert.NotNil(t, handlers.logger) // Should use default logger
	})
}

// TestEnrichmentHandlers_GetMode tests GET /enrichment/mode endpoint
func TestEnrichmentHandlers_GetMode(t *testing.T) {
	tests := []struct {
		name                  string
		mockGetModeWithSource func(ctx context.Context) (services.EnrichmentMode, string, error)
		expectedStatus        int
		expectedMode          string
		expectedSource        string
		expectError           bool
	}{
		{
			name: "returns current mode successfully",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeEnriched, "redis", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "enriched",
			expectedSource: "redis",
			expectError:    false,
		},
		{
			name: "returns transparent mode",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparent, "env", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent",
			expectedSource: "env",
			expectError:    false,
		},
		{
			name: "returns transparent_with_recommendations mode",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparentWithRecommendations, "default", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent_with_recommendations",
			expectedSource: "default",
			expectError:    false,
		},
		{
			name: "handles error gracefully",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return "", "", errors.New("failed to get mode")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock manager
			mockManager := &mockEnrichmentManager{
				getModeWithSource: tt.mockGetModeWithSource,
			}

			// Create handlers
			handlers := NewEnrichmentHandlers(mockManager, slog.Default())

			// Create request
			req, err := http.NewRequest("GET", "/enrichment/mode", nil)
			require.NoError(t, err)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handlers.GetMode(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check content type
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			// Parse response
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EnrichmentModeResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMode, response.Mode)
				assert.Equal(t, tt.expectedSource, response.Source)
			}
		})
	}
}

// TestEnrichmentHandlers_SetMode tests POST /enrichment/mode endpoint
func TestEnrichmentHandlers_SetMode(t *testing.T) {
	tests := []struct {
		name                  string
		requestBody           map[string]string
		mockValidateMode      func(mode services.EnrichmentMode) error
		mockSetMode           func(ctx context.Context, mode services.EnrichmentMode) error
		mockGetModeWithSource func(ctx context.Context) (services.EnrichmentMode, string, error)
		expectedStatus        int
		expectedMode          string
		expectError           bool
	}{
		{
			name:        "sets transparent mode successfully",
			requestBody: map[string]string{"mode": "transparent"},
			mockValidateMode: func(mode services.EnrichmentMode) error {
				return nil
			},
			mockSetMode: func(ctx context.Context, mode services.EnrichmentMode) error {
				return nil
			},
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparent, "redis", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent",
			expectError:    false,
		},
		{
			name:        "sets enriched mode successfully",
			requestBody: map[string]string{"mode": "enriched"},
			mockValidateMode: func(mode services.EnrichmentMode) error {
				return nil
			},
			mockSetMode: func(ctx context.Context, mode services.EnrichmentMode) error {
				return nil
			},
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeEnriched, "redis", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "enriched",
			expectError:    false,
		},
		{
			name:        "sets transparent_with_recommendations mode successfully",
			requestBody: map[string]string{"mode": "transparent_with_recommendations"},
			mockValidateMode: func(mode services.EnrichmentMode) error {
				return nil
			},
			mockSetMode: func(ctx context.Context, mode services.EnrichmentMode) error {
				return nil
			},
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparentWithRecommendations, "redis", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent_with_recommendations",
			expectError:    false,
		},
		{
			name:        "rejects invalid mode",
			requestBody: map[string]string{"mode": "invalid"},
			mockValidateMode: func(mode services.EnrichmentMode) error {
				return errors.New("invalid enrichment mode: invalid")
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "rejects invalid JSON",
			requestBody:    nil, // Will send invalid JSON
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:        "handles SetMode error",
			requestBody: map[string]string{"mode": "transparent"},
			mockValidateMode: func(mode services.EnrichmentMode) error {
				return nil
			},
			mockSetMode: func(ctx context.Context, mode services.EnrichmentMode) error {
				return errors.New("failed to set mode")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock manager
			mockManager := &mockEnrichmentManager{
				validateMode:      tt.mockValidateMode,
				setMode:           tt.mockSetMode,
				getModeWithSource: tt.mockGetModeWithSource,
			}

			// Create handlers
			handlers := NewEnrichmentHandlers(mockManager, slog.Default())

			// Create request body
			var reqBody []byte
			var err error
			if tt.requestBody != nil {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			} else {
				reqBody = []byte("{invalid json")
			}

			// Create request
			req, err := http.NewRequest("POST", "/enrichment/mode", bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handlers.SetMode(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check content type
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			// Parse response
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EnrichmentModeResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMode, response.Mode)
			}
		})
	}
}

// TestEnrichmentHandlers_ResponseFormat tests response format consistency
func TestEnrichmentHandlers_ResponseFormat(t *testing.T) {
	t.Run("GET response format", func(t *testing.T) {
		mockManager := &mockEnrichmentManager{
			getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeEnriched, "redis", nil
			},
		}

		handlers := NewEnrichmentHandlers(mockManager, slog.Default())

		req, err := http.NewRequest("GET", "/enrichment/mode", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response EnrichmentModeResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response has required fields
		assert.NotEmpty(t, response.Mode)
		assert.NotEmpty(t, response.Source)
	})

	t.Run("POST response format", func(t *testing.T) {
		mockManager := &mockEnrichmentManager{
			validateMode: func(mode services.EnrichmentMode) error {
				return nil
			},
			setMode: func(ctx context.Context, mode services.EnrichmentMode) error {
				return nil
			},
			getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparent, "redis", nil
			},
		}

		handlers := NewEnrichmentHandlers(mockManager, slog.Default())

		reqBody, err := json.Marshal(map[string]string{"mode": "transparent"})
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/enrichment/mode", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handlers.SetMode(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response EnrichmentModeResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response has required fields
		assert.NotEmpty(t, response.Mode)
		assert.NotEmpty(t, response.Source)
	})
}
