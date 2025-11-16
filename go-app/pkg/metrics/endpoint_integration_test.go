// +build integration

package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndpointIntegration tests end-to-end integration with real HTTP server.
func TestMetricsEndpointIntegration(t *testing.T) {
	t.Run("end-to-end HTTP server integration", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Create HTTP server
		server := httptest.NewServer(handler)
		defer server.Close()

		// Make request to metrics endpoint
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
		assert.Contains(t, resp.Header.Get("Content-Type"), "version=0.0.4")
	})

	t.Run("metrics format validation", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		lines := strings.Split(body, "\n")

		// Check Prometheus format basics
		hasHelp := false
		hasType := false
		hasMetric := false

		for _, line := range lines {
			if strings.HasPrefix(line, "# HELP") {
				hasHelp = true
			}
			if strings.HasPrefix(line, "# TYPE") {
				hasType = true
			}
			if !strings.HasPrefix(line, "#") && strings.Contains(line, "_") && !strings.HasPrefix(line, " ") {
				hasMetric = true
			}
		}

		assert.True(t, hasHelp, "Response should contain HELP comments")
		assert.True(t, hasType, "Response should contain TYPE comments")
		assert.True(t, hasMetric, "Response should contain metric lines")
	})
}

// TestMetricsRegistryIntegration tests integration with MetricsRegistry.
func TestMetricsRegistryIntegration(t *testing.T) {
	t.Run("all MetricsRegistry metrics are exported", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		// Initialize metrics to ensure they're registered
		business := registry.Business()
		technical := registry.Technical()
		infra := registry.Infra()

		// Use some metrics to ensure they're created
		business.AlertsProcessedTotal.WithLabelValues("test").Inc()
		technical.HTTP.RecordRequest("GET", "/test", 200, 0.1)
		infra.DB.ConnectionsActive.Set(10)

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		err = handler.RegisterMetricsRegistry(registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()

		// Check for business metrics
		assert.Contains(t, body, "alert_history_business_alerts_processed_total")

		// Check for technical metrics
		assert.Contains(t, body, "alert_history_technical_http")

		// Check for infra metrics
		assert.Contains(t, body, "alert_history_infra_db_connections_active")
	})

	t.Run("metrics registry lazy initialization works", func(t *testing.T) {
		config := DefaultEndpointConfig()
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Register before accessing metrics (should trigger lazy init)
		err = handler.RegisterMetricsRegistry(registry)
		require.NoError(t, err)

		// Now access metrics
		business := registry.Business()
		technical := registry.Technical()
		infra := registry.Infra()

		assert.NotNil(t, business)
		assert.NotNil(t, technical)
		assert.NotNil(t, infra)
	})
}

// TestHTTPMetricsIntegration tests integration with HTTPMetrics.
func TestHTTPMetricsIntegration(t *testing.T) {
	t.Run("HTTP metrics are exported", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"

		// Create HTTPMetrics and use it
		httpMetrics := NewHTTPMetrics()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrappedHandler := httpMetrics.Middleware(testHandler)

		// Execute a request to generate metrics
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		// Create endpoint handler
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		err = handler.RegisterHTTPMetrics(httpMetrics)
		require.NoError(t, err)

		// Request metrics
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsW := httptest.NewRecorder()
		handler.ServeHTTP(metricsW, metricsReq)

		body := metricsW.Body.String()

		// Check for HTTP metrics
		assert.Contains(t, body, "alert_history_http_requests_total")
		assert.Contains(t, body, "alert_history_http_request_duration_seconds")
	})

	t.Run("HTTP metrics from MetricsManager integration", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"

		// Create MetricsManager
		metricsConfig := Config{
			Enabled:   true,
			Namespace: "alert_history",
			Subsystem: "http",
		}
		metricsManager := NewMetricsManager(metricsConfig)

		// Create endpoint handler
		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		// Register HTTP metrics
		if httpMetrics := metricsManager.Metrics(); httpMetrics != nil {
			err = handler.RegisterHTTPMetrics(httpMetrics)
			require.NoError(t, err)
		}

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAllMetricsPresent tests that all expected metrics are present in response.
func TestAllMetricsPresent(t *testing.T) {
	t.Run("self-observability metrics are present", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.EnableSelfMetrics = true
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Make a request to generate self-metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()

		// Check all self-observability metrics
		expectedMetrics := []string{
			"alert_history_metrics_endpoint_requests_total",
			"alert_history_metrics_endpoint_request_duration_seconds",
			"alert_history_metrics_endpoint_errors_total",
			"alert_history_metrics_endpoint_response_size_bytes",
			"alert_history_metrics_endpoint_active_requests",
		}

		for _, metric := range expectedMetrics {
			assert.Contains(t, body, metric, "Expected metric %s to be present", metric)
		}
	})

	t.Run("custom metrics are included", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"

		// Create custom registry with custom metric
		customRegistry := prometheus.NewRegistry()
		customCounter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "custom_test_metric_total",
			Help: "Custom test metric",
		})
		customRegistry.MustRegister(customCounter)
		customCounter.Inc()

		config.CustomGatherer = customRegistry

		handler, err := NewMetricsEndpointHandler(config, nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()
		assert.Contains(t, body, "custom_test_metric_total")
	})
}

// TestPrometheusFormatCompliance tests Prometheus format compliance.
func TestPrometheusFormatCompliance(t *testing.T) {
	t.Run("response follows Prometheus text format", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()
		lines := strings.Split(body, "\n")

		// Validate Prometheus format structure
		// Should have HELP and TYPE comments before metrics
		foundHelp := false
		foundType := false
		foundMetric := false

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			if strings.HasPrefix(trimmed, "# HELP") {
				foundHelp = true
				// Next line should be TYPE
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(nextLine, "# TYPE") {
						foundType = true
					}
				}
			}

			if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "_") {
				foundMetric = true
				// Metric line should not start with space (unless continuation)
				assert.False(t, strings.HasPrefix(trimmed, " "), "Metric line should not start with space: %s", trimmed)
			}
		}

		assert.True(t, foundHelp, "Should have HELP comments")
		assert.True(t, foundType, "Should have TYPE comments")
		assert.True(t, foundMetric, "Should have metric lines")
	})

	t.Run("content type header is correct", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", contentType)
	})
}
