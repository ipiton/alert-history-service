package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// MetricsMiddleware wraps HTTP handler with metrics collection
func MetricsMiddleware(metrics *HistoryMetrics) func(http.Handler) http.Handler {
	if metrics == nil {
		metrics = Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Increment active requests
			metrics.HTTPActiveRequests.Inc()
			defer metrics.HTTPActiveRequests.Dec()

			// Track request size
			if r.ContentLength > 0 {
				metrics.HTTPRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
			}

			// Wrap response writer to capture status code and size
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Extract endpoint (simplified path)
			endpoint := extractEndpoint(r.URL.Path)
			statusCode := strconv.Itoa(rw.statusCode)

			// Record metrics
			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, endpoint, statusCode).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, endpoint, statusCode).Observe(duration)

			if rw.size > 0 {
				metrics.HTTPResponseSize.WithLabelValues(r.Method, endpoint, statusCode).Observe(float64(rw.size))
			}

			// Record errors
			if rw.statusCode >= 400 {
				errorType := getErrorType(rw.statusCode)
				metrics.HTTPErrorsTotal.WithLabelValues(r.Method, endpoint, errorType).Inc()
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// extractEndpoint extracts endpoint name from path
func extractEndpoint(path string) string {
	// Map paths to endpoint names
	endpoints := map[string]string{
		"/api/v2/history":           "history",
		"/api/v2/history/top":       "top",
		"/api/v2/history/flapping":  "flapping",
		"/api/v2/history/recent":   "recent",
		"/api/v2/history/stats":    "stats",
		"/api/v2/history/search":   "search",
	}

	// Check exact match first
	if endpoint, ok := endpoints[path]; ok {
		return endpoint
	}

	// Check prefix match for /api/v2/history/{fingerprint}
	if len(path) > 20 && path[:20] == "/api/v2/history/" {
		return "timeline"
	}

	return "unknown"
}

// getErrorType maps HTTP status code to error type
func getErrorType(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "server_error"
	case statusCode == 429:
		return "rate_limit"
	case statusCode == 401:
		return "authentication"
	case statusCode == 403:
		return "authorization"
	case statusCode >= 400:
		return "client_error"
	default:
		return "unknown"
	}
}
