package middleware

import (
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// MetricsMiddleware records Prometheus metrics for HTTP requests
func MetricsMiddleware(registry *metrics.Registry) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Record active requests (increment)
			// Note: This requires webhook metrics to expose active requests gauge
			// For now, we'll just record request/response metrics

			// Wrap ResponseWriter to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := determineStatus(rw.statusCode)

			// Record request metric (counter)
			// This assumes webhook metrics are exposed via registry
			// Actual implementation depends on metrics package structure

			// Log metrics recording (for now, until metrics package is fully integrated)
			_ = duration
			_ = status
			// TODO: registry.WebhookMetrics().RecordRequest(r.Method, status, duration)
		})
	}
}

// determineStatus maps HTTP status code to metric status label
func determineStatus(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "success"
	case statusCode == http.StatusMultiStatus: // 207
		return "partial"
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}
