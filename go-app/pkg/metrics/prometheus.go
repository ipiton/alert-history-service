// Package metrics provides Prometheus metrics collection for HTTP requests.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics holds Prometheus metrics for HTTP requests.
type HTTPMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestSize      *prometheus.HistogramVec
	responseSize     *prometheus.HistogramVec
	activeRequests   prometheus.Gauge
}

// NewHTTPMetrics creates a new HTTPMetrics instance with default configuration.
func NewHTTPMetrics() *HTTPMetrics {
	return NewHTTPMetricsWithNamespace("alert_history", "http")
}

// NewHTTPMetricsWithNamespace creates a new HTTPMetrics instance with custom namespace and subsystem.
func NewHTTPMetricsWithNamespace(namespace, subsystem string) *HTTPMetrics {
	return &HTTPMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed",
			},
			[]string{"method", "path", "status_code"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "path", "status_code"},
		),
		requestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_size_bytes",
				Help:      "Size of HTTP requests in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		responseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "response_size_bytes",
				Help:      "Size of HTTP responses in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status_code"},
		),
		activeRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_requests",
				Help:      "Number of currently active HTTP requests",
			},
		),
	}
}

// responseWriter wraps http.ResponseWriter to capture response size and status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size.
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(size)
	return size, err
}

// Middleware returns an HTTP middleware that collects Prometheus metrics.
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics collection for the metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		m.activeRequests.Inc()

		// Wrap response writer to capture status code and response size
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
		}

		// Capture request size
		requestSize := r.ContentLength
		if requestSize > 0 {
			m.requestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(requestSize))
		}

		// Process request
		defer func() {
			duration := time.Since(start)
			statusCode := strconv.Itoa(rw.statusCode)

			// Record metrics
			m.requestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
			m.requestDuration.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(duration.Seconds())

			// Record response size if available
			if rw.responseSize > 0 {
				m.responseSize.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(float64(rw.responseSize))
			}

			m.activeRequests.Dec()
		}()

		next.ServeHTTP(rw, r)
	})
}

// Handler returns the Prometheus metrics HTTP handler.
func (m *HTTPMetrics) Handler() http.Handler {
	return promhttp.Handler()
}

// Config holds configuration for metrics collection.
type Config struct {
	Enabled         bool   `mapstructure:"enabled"`
	Path            string `mapstructure:"path"`
	Namespace       string `mapstructure:"namespace"`
	Subsystem       string `mapstructure:"subsystem"`
	CollectReqSize  bool   `mapstructure:"collect_request_size"`
	CollectRespSize bool   `mapstructure:"collect_response_size"`
}

// DefaultConfig returns default metrics configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Path:            "/metrics",
		Namespace:       "alert_history",
		Subsystem:       "http",
		CollectReqSize:  false,
		CollectRespSize: false,
	}
}

// MetricsManager manages HTTP metrics collection and configuration.
type MetricsManager struct {
	config  Config
	metrics *HTTPMetrics
}

// NewMetricsManager creates a new MetricsManager with the given configuration.
func NewMetricsManager(config Config) *MetricsManager {
	var metrics *HTTPMetrics
	if config.Enabled {
		metrics = NewHTTPMetricsWithNamespace(config.Namespace, config.Subsystem)
	}

	return &MetricsManager{
		config:  config,
		metrics: metrics,
	}
}

// Middleware returns the metrics middleware if enabled, otherwise returns a pass-through middleware.
func (mm *MetricsManager) Middleware(next http.Handler) http.Handler {
	if !mm.config.Enabled || mm.metrics == nil {
		return next
	}
	return mm.metrics.Middleware(next)
}

// Handler returns the metrics HTTP handler if enabled, otherwise returns a 404 handler.
func (mm *MetricsManager) Handler() http.Handler {
	if !mm.config.Enabled || mm.metrics == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	return mm.metrics.Handler()
}

// IsEnabled returns true if metrics collection is enabled.
func (mm *MetricsManager) IsEnabled() bool {
	return mm.config.Enabled
}

// GetPath returns the metrics endpoint path.
func (mm *MetricsManager) GetPath() string {
	return mm.config.Path
}
