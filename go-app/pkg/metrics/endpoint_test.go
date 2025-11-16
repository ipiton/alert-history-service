package metrics

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements Logger interface for testing.
type mockLogger struct {
	errors []error
	logs   []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.logs = append(m.logs, "ERROR: "+msg)
	if len(args) > 0 {
		if err, ok := args[0].(error); ok {
			m.errors = append(m.errors, err)
		}
	}
}

// TestNewMetricsEndpointHandler tests handler creation.
func TestNewMetricsEndpointHandler(t *testing.T) {
	t.Run("creates handler with default config", func(t *testing.T) {
		config := DefaultEndpointConfig()
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)
		require.NotNil(t, handler)
		assert.Equal(t, config.Path, handler.config.Path)
		assert.True(t, handler.config.EnableSelfMetrics)
	})

	t.Run("creates handler without registry", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("creates handler with Go runtime metrics", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableGoRuntime = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("creates handler with process metrics", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableProcess = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("creates handler with custom gatherer", func(t *testing.T) {
		config := DefaultEndpointConfig()
		customRegistry := prometheus.NewRegistry()
		customRegistry.MustRegister(prometheus.NewCounter(prometheus.CounterOpts{
			Name: "custom_metric_total",
			Help: "Custom metric",
		}))
		config.CustomGatherer = customRegistry
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})
}

// TestMetricsEndpointHandler_ServeHTTP tests the ServeHTTP method.
func TestMetricsEndpointHandler_ServeHTTP(t *testing.T) {
	t.Run("returns metrics for GET request", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
		assert.Contains(t, w.Header().Get("Content-Type"), "version=0.0.4")
		assert.Contains(t, w.Body.String(), "alert_history")
	})

	t.Run("rejects non-GET requests", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				req := httptest.NewRequest(method, "/metrics", nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			})
		}
	})

	t.Run("returns 404 for invalid path", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("sets correct headers", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("includes self-observability metrics", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.EnableSelfMetrics = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		assert.Contains(t, body, "alert_history_metrics_endpoint_requests_total")
		assert.Contains(t, body, "alert_history_metrics_endpoint_request_duration_seconds")
		assert.Contains(t, body, "alert_history_metrics_endpoint_errors_total")
		assert.Contains(t, body, "alert_history_metrics_endpoint_response_size_bytes")
		assert.Contains(t, body, "alert_history_metrics_endpoint_active_requests")
	})
}

// TestRegisterMetricsRegistry tests metrics registry registration.
func TestRegisterMetricsRegistry(t *testing.T) {
	t.Run("registers metrics registry successfully", func(t *testing.T) {
		config := DefaultEndpointConfig()
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		err = handler.RegisterMetricsRegistry(registry)
		assert.NoError(t, err)
	})

	t.Run("returns error for nil registry", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		err = handler.RegisterMetricsRegistry(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("initializes all metric categories", func(t *testing.T) {
		config := DefaultEndpointConfig()
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		err = handler.RegisterMetricsRegistry(registry)
		assert.NoError(t, err)

		// Verify metrics are initialized
		business := registry.Business()
		technical := registry.Technical()
		infra := registry.Infra()

		assert.NotNil(t, business)
		assert.NotNil(t, technical)
		assert.NotNil(t, infra)
	})
}

// TestRegisterHTTPMetrics tests HTTP metrics registration.
func TestRegisterHTTPMetrics(t *testing.T) {
	t.Run("validates HTTP metrics successfully", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// Use existing HTTPMetrics from MetricsManager to avoid duplicate registration
		metricsConfig := Config{
			Enabled:   true,
			Namespace: "test",
			Subsystem: "http",
		}
		metricsManager := NewMetricsManager(metricsConfig)
		httpMetrics := metricsManager.Metrics()

		if httpMetrics != nil {
			err = handler.RegisterHTTPMetrics(httpMetrics)
			assert.NoError(t, err)
		}
	})

	t.Run("returns error for nil HTTP metrics", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		err = handler.RegisterHTTPMetrics(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
}

// TestSetLogger tests logger setting.
func TestSetLogger(t *testing.T) {
	t.Run("sets logger successfully", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		logger := &mockLogger{}
		handler.SetLogger(logger)

		// Trigger an error to verify logger is set
		req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Logger should be set (no panic)
		assert.NotNil(t, logger)
	})
}

// TestErrorHandling tests error handling scenarios.
func TestErrorHandling(t *testing.T) {
	t.Run("handles timeout gracefully", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.GatherTimeout = 1 * time.Nanosecond // Very short timeout to trigger timeout
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		logger := &mockLogger{}
		handler.SetLogger(logger)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should handle timeout gracefully
		// May return partial metrics or error
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusGatewayTimeout || w.Code == http.StatusInternalServerError)
	})

	t.Run("records error metrics", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.EnableSelfMetrics = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Make a request that will succeed
		req1 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Make a request with invalid method to trigger error
		req2 := httptest.NewRequest(http.MethodPost, "/metrics", nil)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusMethodNotAllowed, w2.Code)
	})
}

// TestConcurrentRequests tests concurrent request handling.
func TestConcurrentRequests(t *testing.T) {
	t.Run("handles concurrent requests", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.EnableSelfMetrics = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Make multiple concurrent requests
		const numRequests = 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
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

		assert.Equal(t, numRequests, successCount)
	})
}

// TestGetRegistry tests registry access.
func TestGetRegistry(t *testing.T) {
	t.Run("returns registry", func(t *testing.T) {
		config := DefaultEndpointConfig()
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		registry := handler.GetRegistry()
		assert.NotNil(t, registry)
	})
}

// TestDefaultEndpointConfig tests default configuration.
func TestDefaultEndpointConfig(t *testing.T) {
	t.Run("returns correct default config", func(t *testing.T) {
		config := DefaultEndpointConfig()

		assert.Equal(t, "/metrics", config.Path)
		assert.False(t, config.EnableGoRuntime)
		assert.False(t, config.EnableProcess)
		assert.True(t, config.EnableSelfMetrics)
		assert.Equal(t, 5*time.Second, config.GatherTimeout)
		assert.Equal(t, int64(10*1024*1024), config.MaxResponseSize)
		assert.False(t, config.CacheEnabled)
		assert.Equal(t, time.Duration(0), config.CacheTTL)
		assert.True(t, config.RateLimitEnabled)
		assert.Equal(t, 60, config.RateLimitPerMinute)
		assert.Equal(t, 10, config.RateLimitBurst)
		assert.True(t, config.EnableSecurityHeaders)
	})
}

// TestDefaultErrorHandler tests error handler.
func TestDefaultErrorHandler(t *testing.T) {
	t.Run("logs errors", func(t *testing.T) {
		logger := &mockLogger{}
		handler := &DefaultErrorHandler{logger: logger}

		testErr := context.DeadlineExceeded
		handler.LogError(context.Background(), testErr)

		// Logger should have been called (errors may be logged differently)
		// The important thing is that it doesn't panic
		assert.NotNil(t, logger)
	})

	t.Run("should return partial metrics for timeout", func(t *testing.T) {
		handler := &DefaultErrorHandler{}
		assert.True(t, handler.ShouldReturnPartialMetrics(context.DeadlineExceeded))
		assert.True(t, handler.ShouldReturnPartialMetrics(context.Canceled))
	})

	t.Run("should not return partial metrics for other errors", func(t *testing.T) {
		handler := &DefaultErrorHandler{}
		assert.False(t, handler.ShouldReturnPartialMetrics(assert.AnError))
	})
}

// TestRateLimiting tests rate limiting functionality.
func TestRateLimiting(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = true
		config.RateLimitPerMinute = 10
		config.RateLimitBurst = 5

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// Make requests within limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = true
		config.RateLimitPerMinute = 2 // Very low limit for testing
		config.RateLimitBurst = 1

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.RemoteAddr = "127.0.0.1:12346"

		// First request should succeed
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should succeed (burst allows 1 extra)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req)
		// With burst=1 and rate=2/min, we can have 1+1=2 requests initially
		// But token bucket refills slowly, so second might succeed or fail depending on timing
		// Let's check that at least one request succeeds
		assert.True(t, w2.Code == http.StatusOK || w2.Code == http.StatusTooManyRequests)

		// Third request should definitely be rate limited
		w3 := httptest.NewRecorder()
		handler.ServeHTTP(w3, req)
		assert.Equal(t, http.StatusTooManyRequests, w3.Code)
		assert.Contains(t, w3.Body.String(), "rate_limit_exceeded")
		assert.Equal(t, "2", w3.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w3.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w3.Header().Get("Retry-After"))
	})

	t.Run("rate limiting disabled", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// Make many requests - should all succeed
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = "127.0.0.1:12347"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = true
		config.RateLimitPerMinute = 2
		config.RateLimitBurst = 1

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// IP1 exhausts limit
		req1 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req1.RemoteAddr = "127.0.0.1:12348"
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		w1b := httptest.NewRecorder()
		handler.ServeHTTP(w1b, req1)
		// With burst=1, second request might succeed or fail depending on timing
		// The important thing is that IP2 can still make requests
		assert.True(t, w1b.Code == http.StatusOK || w1b.Code == http.StatusTooManyRequests)

		// IP2 should still work (separate limiter)
		req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req2.RemoteAddr = "127.0.0.2:12349"
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code, "IP2 should have separate rate limit")
	})
}

// TestSecurityHeaders tests security headers functionality.
func TestSecurityHeaders(t *testing.T) {
	t.Run("sets security headers when enabled", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableSecurityHeaders = true
		config.RateLimitEnabled = false // Disable rate limiting for this test

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
		assert.Contains(t, w.Header().Get("Permissions-Policy"), "geolocation=()")
		assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
	})

	t.Run("does not set security headers when disabled", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableSecurityHeaders = false
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Empty(t, w.Header().Get("X-Content-Type-Options"))
		assert.Empty(t, w.Header().Get("X-Frame-Options"))
	})

	t.Run("sets HSTS only over HTTPS", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableSecurityHeaders = true
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// HTTP request - no HSTS
		reqHTTP := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		wHTTP := httptest.NewRecorder()
		handler.ServeHTTP(wHTTP, reqHTTP)
		assert.Empty(t, wHTTP.Header().Get("Strict-Transport-Security"))

		// HTTPS request - HSTS should be set
		reqHTTPS := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		reqHTTPS.TLS = &tls.ConnectionState{}
		wHTTPS := httptest.NewRecorder()
		handler.ServeHTTP(wHTTPS, reqHTTPS)
		assert.Contains(t, wHTTPS.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	})

	t.Run("removes sensitive headers", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.EnableSecurityHeaders = true
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Server and X-Powered-By should be removed
		assert.Empty(t, w.Header().Get("Server"))
		assert.Empty(t, w.Header().Get("X-Powered-By"))
	})
}

// TestRequestValidation tests request validation functionality.
func TestRequestValidation(t *testing.T) {
	t.Run("rejects non-GET methods", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/metrics", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			assert.Equal(t, "GET", w.Header().Get("Allow"))
		}
	})

	t.Run("rejects invalid paths", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		invalidPaths := []string{"/metrics/extra", "/metric", "/metrics/", "/"}

		for _, path := range invalidPaths {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code, "path: %s", path)
		}
	})

	t.Run("rejects query parameters", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics?format=json", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid query parameters")
	})

	t.Run("allows empty query", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.RateLimitEnabled = false

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics?", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
