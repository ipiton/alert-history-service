package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

type mockEnrichmentManager struct {
	mode   services.EnrichmentMode
	source string
	err    error
}

func (m *mockEnrichmentManager) GetMode(ctx context.Context) (services.EnrichmentMode, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.mode, nil
}

func (m *mockEnrichmentManager) GetModeWithSource(ctx context.Context) (services.EnrichmentMode, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	return m.mode, m.source, nil
}

func (m *mockEnrichmentManager) SetMode(ctx context.Context, mode services.EnrichmentMode) error {
	m.mode = mode
	return nil
}

func (m *mockEnrichmentManager) ValidateMode(mode services.EnrichmentMode) error {
	if !mode.IsValid() {
		return errors.New("invalid mode")
	}
	return nil
}

func (m *mockEnrichmentManager) GetStats(ctx context.Context) (*services.EnrichmentStats, error) {
	return &services.EnrichmentStats{
		CurrentMode:    m.mode,
		Source:         m.source,
		RedisAvailable: true,
	}, nil
}

func (m *mockEnrichmentManager) RefreshCache(ctx context.Context) error {
	return nil
}

func TestEnrichmentModeMiddleware(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		manager := &mockEnrichmentManager{
			mode:   services.EnrichmentModeTransparent,
			source: "redis",
		}

		middleware := NewEnrichmentModeMiddleware(manager, slog.Default())

		// Create test handler
		var capturedMode services.EnrichmentMode
		var capturedSource string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode, ok := GetEnrichmentModeFromContext(r.Context())
			assert.True(t, ok)
			capturedMode = mode

			source, ok := GetEnrichmentSourceFromContext(r.Context())
			assert.True(t, ok)
			capturedSource = source

			w.WriteHeader(http.StatusOK)
		})

		// Wrap with middleware
		wrappedHandler := middleware.Middleware(handler)

		// Create test request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		// Execute
		wrappedHandler.ServeHTTP(rec, req)

		// Verify
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, services.EnrichmentModeTransparent, capturedMode)
		assert.Equal(t, "redis", capturedSource)

		// Check response headers
		assert.Equal(t, "transparent", rec.Header().Get("X-Enrichment-Mode"))
		assert.Equal(t, "redis", rec.Header().Get("X-Enrichment-Source"))
	})

	t.Run("error_fallback_to_default", func(t *testing.T) {
		manager := &mockEnrichmentManager{
			err: errors.New("manager error"),
		}

		middleware := NewEnrichmentModeMiddleware(manager, slog.Default())

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode := MustGetEnrichmentModeFromContext(r.Context())
			assert.Equal(t, services.EnrichmentModeEnriched, mode)
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := middleware.Middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "enriched", rec.Header().Get("X-Enrichment-Mode"))
		assert.Equal(t, "default", rec.Header().Get("X-Enrichment-Source"))
	})

	t.Run("enriched_mode", func(t *testing.T) {
		manager := &mockEnrichmentManager{
			mode:   services.EnrichmentModeEnriched,
			source: "env",
		}

		middleware := NewEnrichmentModeMiddleware(manager, slog.Default())

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode := MustGetEnrichmentModeFromContext(r.Context())
			assert.Equal(t, services.EnrichmentModeEnriched, mode)
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := middleware.Middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, "enriched", rec.Header().Get("X-Enrichment-Mode"))
		assert.Equal(t, "env", rec.Header().Get("X-Enrichment-Source"))
	})
}

func TestGetEnrichmentModeFromContext(t *testing.T) {
	t.Run("mode_present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), EnrichmentModeKey, services.EnrichmentModeTransparent)
		mode, ok := GetEnrichmentModeFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, services.EnrichmentModeTransparent, mode)
	})

	t.Run("mode_absent", func(t *testing.T) {
		ctx := context.Background()
		mode, ok := GetEnrichmentModeFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, mode)
	})
}

func TestMustGetEnrichmentModeFromContext(t *testing.T) {
	t.Run("mode_present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), EnrichmentModeKey, services.EnrichmentModeTransparent)
		mode := MustGetEnrichmentModeFromContext(ctx)
		assert.Equal(t, services.EnrichmentModeTransparent, mode)
	})

	t.Run("mode_absent_returns_default", func(t *testing.T) {
		ctx := context.Background()
		mode := MustGetEnrichmentModeFromContext(ctx)
		assert.Equal(t, services.EnrichmentModeEnriched, mode)
	})
}
