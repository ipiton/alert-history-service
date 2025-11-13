package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "endpoint"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)
)

// MetricsMiddleware instruments HTTP requests with Prometheus metrics
//
// Metrics collected:
//   - api_http_requests_total (counter) - Total requests by method, endpoint, status
//   - api_http_request_duration_seconds (histogram) - Request duration
//   - api_http_requests_in_flight (gauge) - Active requests
//   - api_http_request_size_bytes (histogram) - Request size
//   - api_http_response_size_bytes (histogram) - Response size
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Normalize endpoint path for metrics (avoid high cardinality)
		endpoint := normalizeEndpoint(r.URL.Path)
		method := r.Method

		// Track in-flight requests
		httpRequestsInFlight.WithLabelValues(method, endpoint).Inc()
		defer httpRequestsInFlight.WithLabelValues(method, endpoint).Dec()

		// Wrap response writer to capture status and size
		rw := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Record request size
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(r.ContentLength))
		}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record metrics
		status := strconv.Itoa(rw.statusCode)
		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
		httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(rw.size))
	})
}

// metricsResponseWriter wraps http.ResponseWriter for metrics collection
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// normalizeEndpoint normalizes URL paths to reduce cardinality in metrics
// Replaces dynamic segments (UUIDs, IDs) with placeholders
//
// Examples:
//   - /api/v2/publishing/targets/rootly-prod -> /api/v2/publishing/targets/{name}
//   - /api/v2/publishing/jobs/123e4567-e89b -> /api/v2/publishing/jobs/{id}
func normalizeEndpoint(path string) string {
	// TODO: Implement path normalization
	// For now, return path as-is
	// In production, use a proper path pattern matcher (gorilla/mux patterns)
	return path
}
