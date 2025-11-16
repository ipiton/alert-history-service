// Package metrics provides Prometheus metrics collection for HTTP requests.
//
// This package includes:
//   - Unified MetricsRegistry for business, technical, and infrastructure metrics
//   - HTTPMetrics middleware for request metrics
//   - MetricsEndpointHandler for enterprise-grade /metrics endpoint
//
// The MetricsEndpointHandler provides:
//   - Performance optimization (caching, buffer pooling)
//   - Security (rate limiting, security headers)
//   - Observability (self-metrics, structured logging)
//   - Reliability (graceful error handling, partial metrics)
//
// Example:
//
//	// Create handler
//	config := metrics.DefaultEndpointConfig()
//	registry := metrics.DefaultRegistry()
//	handler, err := metrics.NewMetricsEndpointHandler(config, registry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Set logger
//	handler.SetLogger(&metricsLoggerAdapter{logger: slog.Default()})
//
//	// Register endpoint
//	http.Handle("/metrics", handler)
//
// See docs/api/metrics-endpoint.md for detailed documentation.
package metrics

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"golang.org/x/time/rate"
)

// cachedResponse holds a cached metrics response.
type cachedResponse struct {
	data      []byte
	timestamp time.Time
	mu        sync.RWMutex
}

// rateLimiter implements per-client rate limiting using token bucket algorithm.
type rateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // Requests per second
	burst    int        // Burst capacity
}

// newRateLimiter creates a new rate limiter.
func newRateLimiter(requestsPerMinute int, burst int) *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:    burst,
	}
}

// allow checks if a request from the given client is allowed.
func (rl *rateLimiter) allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[clientID] = limiter
	}

	return limiter.Allow()
}

// cleanup removes stale limiters (full token bucket = inactive).
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, limiter := range rl.limiters {
		// If limiter has full tokens, it hasn't been used recently
		if limiter.TokensAt(now) == float64(rl.burst) {
			delete(rl.limiters, key)
		}
	}
}

// MetricsEndpointHandler handles GET /metrics requests.
// Provides enterprise-grade features: performance optimization, error handling,
// self-observability, and security.
//
// Usage:
//
//	config := DefaultEndpointConfig()
//	handler, err := NewMetricsEndpointHandler(config, metricsRegistry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	http.Handle("/metrics", handler)
type MetricsEndpointHandler struct {
	// Core handler
	handler http.Handler

	// Configuration
	config EndpointConfig

	// Self-observability metrics
	requestsTotal   prometheus.Counter
	requestDuration prometheus.Histogram
	requestErrors   prometheus.Counter
	requestSize     prometheus.Histogram
	activeRequests  prometheus.Gauge

	// Error handling
	errorHandler ErrorHandler

	// Performance optimization
	gatherer prometheus.Gatherer
	registry *prometheus.Registry
	customGatherer prometheus.Gatherer

	// Caching (optional)
	cachedResp *cachedResponse
	cacheMu    sync.RWMutex

	// Rate limiting (optional)
	rateLimiter *rateLimiter
	rateLimitMu sync.RWMutex

	// Thread safety
	mu sync.RWMutex
}

// EndpointConfig holds configuration for the metrics endpoint.
//
// Example:
//
//	config := DefaultEndpointConfig()
//	config.CacheEnabled = true
//	config.CacheTTL = 5 * time.Second
//	config.RateLimitPerMinute = 100
type EndpointConfig struct {
	// Path for the metrics endpoint (default: "/metrics").
	// Must match exactly in requests (e.g., "/metrics" not "/metrics/").
	Path string

	// EnableGoRuntime enables Go runtime metrics (memstats, GC, etc.).
	// Disabled by default for performance. Enable for debugging.
	EnableGoRuntime bool

	// EnableProcess enables process metrics (CPU, memory, file descriptors).
	// Disabled by default for security. Enable for detailed monitoring.
	EnableProcess bool

	// GatherTimeout is the maximum time allowed for gathering metrics.
	// Default: 5 seconds. Should be less than Prometheus scrape_timeout.
	GatherTimeout time.Duration

	// MaxResponseSize is the maximum response size in bytes (0 = unlimited).
	// Default: 10MB. Responses exceeding this will return an error.
	MaxResponseSize int64

	// EnableSelfMetrics enables self-observability metrics for the endpoint.
	// Default: true. Metrics include requests_total, request_duration_seconds, etc.
	EnableSelfMetrics bool

	// CustomGatherer is an optional additional Prometheus gatherer.
	// Metrics from this gatherer will be included in the response.
	CustomGatherer prometheus.Gatherer

	// CacheEnabled enables in-memory caching of metrics responses.
	// Default: false. Enable for high-traffic scenarios (>100 req/min).
	CacheEnabled bool

	// CacheTTL is the time-to-live for cached responses.
	// Default: 0 (no cache). Recommended: match Prometheus scrape_interval.
	CacheTTL time.Duration

	// RateLimitEnabled enables per-client rate limiting.
	// Default: true. Uses token bucket algorithm.
	RateLimitEnabled bool

	// RateLimitPerMinute is the maximum requests per minute per client.
	// Default: 60. Adjust based on number of Prometheus scrapers.
	RateLimitPerMinute int

	// RateLimitBurst is the burst capacity (allows temporary spikes).
	// Default: 10. Should be ~10-20% of RateLimitPerMinute.
	RateLimitBurst int

	// EnableSecurityHeaders enables security headers in responses.
	// Default: true. Includes X-Content-Type-Options, CSP, HSTS, etc.
	EnableSecurityHeaders bool
}

// DefaultEndpointConfig returns default configuration for the metrics endpoint.
//
// Default values are optimized for production use:
//   - Rate limiting enabled (60 req/min)
//   - Security headers enabled
//   - Caching disabled (enable for high-traffic)
//   - Go runtime/process metrics disabled (performance/security)
//
// Example:
//
//	config := DefaultEndpointConfig()
//	config.CacheEnabled = true
//	config.CacheTTL = 15 * time.Second
//	handler, err := NewMetricsEndpointHandler(config, registry)
func DefaultEndpointConfig() EndpointConfig {
	return EndpointConfig{
		Path:              "/metrics",
		EnableGoRuntime:   false, // Disabled by default for performance
		EnableProcess:     false, // Disabled by default for security
		GatherTimeout:     5 * time.Second,
		MaxResponseSize:   10 * 1024 * 1024, // 10MB
		EnableSelfMetrics: true,
		CacheEnabled:      false, // Disabled by default (can be enabled for high-traffic scenarios)
		CacheTTL:          0,     // No cache by default
		RateLimitEnabled:  true, // Enabled by default for security
		RateLimitPerMinute: 60,  // 60 requests per minute per client
		RateLimitBurst:     10,  // Allow burst of 10 requests
		EnableSecurityHeaders: true, // Enabled by default
	}
}

// ErrorHandler handles errors in metrics endpoint.
type ErrorHandler interface {
	LogError(ctx context.Context, err error)
	ShouldReturnPartialMetrics(err error) bool
}

// DefaultErrorHandler is the default error handler.
type DefaultErrorHandler struct {
	logger Logger
}

// Logger interface for structured logging.
type Logger interface {
	// Debug logs a debug message with key-value pairs.
	Debug(msg string, args ...interface{})
	// Info logs an info message with key-value pairs.
	Info(msg string, args ...interface{})
	// Warn logs a warning message with key-value pairs.
	Warn(msg string, args ...interface{})
	// Error logs an error message with key-value pairs.
	Error(msg string, args ...interface{})
}

// LogError logs the error with context information.
func (h *DefaultErrorHandler) LogError(ctx context.Context, err error) {
	if h.logger != nil {
		// Extract request ID from context if available
		requestID := ""
		if reqID := ctx.Value("request_id"); reqID != nil {
			if id, ok := reqID.(string); ok {
				requestID = id
			}
		}

		if requestID != "" {
			h.logger.Error("metrics endpoint error",
				"error", err,
				"request_id", requestID,
			)
		} else {
			h.logger.Error("metrics endpoint error", "error", err)
		}
	}
}

// ShouldReturnPartialMetrics determines if partial metrics should be returned.
func (h *DefaultErrorHandler) ShouldReturnPartialMetrics(err error) bool {
	// Return partial metrics for context timeout, but not for other errors
	return err == context.DeadlineExceeded || err == context.Canceled
}

// NewMetricsEndpointHandler creates a new metrics endpoint handler.
//
// Parameters:
//   - config: Configuration for the endpoint
//   - registry: Optional MetricsRegistry to register metrics from
//
// Returns:
//   - *MetricsEndpointHandler: The handler instance
//   - error: If failed to create handler
func NewMetricsEndpointHandler(config EndpointConfig, registry *MetricsRegistry) (*MetricsEndpointHandler, error) {
	// Create Prometheus registry
	promRegistry := prometheus.NewRegistry()

	// Register default metrics
	if config.EnableGoRuntime {
		promRegistry.MustRegister(prometheus.NewGoCollector())
	}
	if config.EnableProcess {
		promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// Determine gatherer to use
	// Include default gatherer to get metrics registered via promauto (MetricsRegistry)
	gatherers := []prometheus.Gatherer{prometheus.DefaultGatherer, promRegistry}
	if config.CustomGatherer != nil {
		gatherers = append(gatherers, config.CustomGatherer)
	}
	gatherer := prometheus.Gatherers(gatherers)

	// Create handler
	handler := &MetricsEndpointHandler{
		config:         config,
		gatherer:       gatherer,
		registry:       promRegistry,
		customGatherer: config.CustomGatherer,
		handler:        promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
		errorHandler: &DefaultErrorHandler{
			logger: nil, // Will be set if logger is provided
		},
		cachedResp: &cachedResponse{},
	}

	// Register MetricsRegistry metrics if provided
	if registry != nil {
		if err := handler.RegisterMetricsRegistry(registry); err != nil {
			return nil, fmt.Errorf("failed to register metrics registry: %w", err)
		}
	}

	// Initialize self-observability metrics
	if config.EnableSelfMetrics {
		handler.initSelfMetrics()
	}

	// Initialize rate limiter if enabled
	if config.RateLimitEnabled {
		handler.rateLimiter = newRateLimiter(config.RateLimitPerMinute, config.RateLimitBurst)
		// Start cleanup goroutine for stale limiters
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				handler.rateLimiter.cleanup()
			}
		}()
	}

	return handler, nil
}

// SetLogger sets the logger for structured logging.
//
// The logger is used for:
//   - Request logging (start/completion)
//   - Error logging
//   - Performance metrics logging
//
// Example:
//
//	logger := slog.Default()
//	handler.SetLogger(&metricsLoggerAdapter{logger: logger})
//
// See Logger interface for required methods.
func (h *MetricsEndpointHandler) SetLogger(logger Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if defaultHandler, ok := h.errorHandler.(*DefaultErrorHandler); ok {
		defaultHandler.logger = logger
	}
}

// extractClientIP extracts client IP from request headers.
// Priority: X-Forwarded-For > X-Real-IP > RemoteAddr
func extractClientIP(r *http.Request) string {
	// Try X-Forwarded-For header first (behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// setSecurityHeaders sets security-related HTTP headers.
func (h *MetricsEndpointHandler) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	if !h.config.EnableSecurityHeaders {
		return
	}

	// Prevent MIME type sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS filter in older browsers
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Content Security Policy (strict for metrics endpoint)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

	// HTTP Strict Transport Security (HSTS) - only over HTTPS
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	// Referrer Policy
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Permissions Policy
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

	// Cache-Control (prevent caching of metrics for security)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")

	// Remove potentially sensitive server information
	w.Header().Del("Server")
	w.Header().Del("X-Powered-By")
}

// validateQueryParams validates query parameters (currently only allows empty or known params).
func (h *MetricsEndpointHandler) validateQueryParams(r *http.Request) bool {
	// Prometheus scraping doesn't typically use query params, but we allow empty query
	// Future: could allow specific params like ?format=prometheus
	query := r.URL.Query()
	if len(query) == 0 {
		return true
	}

	// Allow only known safe parameters (if any)
	// For now, reject all query params for security
	return false
}

// ServeHTTP implements http.Handler interface.
func (h *MetricsEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate request method (only GET allowed)
	if r.Method != http.MethodGet {
		h.setSecurityHeaders(w, r)
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate path
	if r.URL.Path != h.config.Path {
		h.setSecurityHeaders(w, r)
		http.NotFound(w, r)
		return
	}

	// Validate query parameters
	if !h.validateQueryParams(r) {
		h.setSecurityHeaders(w, r)
		http.Error(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	// Rate limiting check (before processing request)
	if h.config.RateLimitEnabled && h.rateLimiter != nil {
		clientID := extractClientIP(r)
		if !h.rateLimiter.allow(clientID) {
			h.setSecurityHeaders(w, r)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.config.RateLimitPerMinute))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate_limit_exceeded","message":"Too many requests. Please retry after 60 seconds.","limit":%d,"retry_after":60}`, h.config.RateLimitPerMinute)
			return
		}
	}

	// Set security headers
	h.setSecurityHeaders(w, r)

	// Log request start (if logger is available)
	h.logRequestStart(r)

	start := time.Now()

	// Optimized: check active requests without lock if not needed
	var active prometheus.Gauge
	if h.config.EnableSelfMetrics {
		h.mu.RLock()
		active = h.activeRequests
		h.mu.RUnlock()
		if active != nil {
			active.Inc()
			defer active.Dec()
		}
	}

	// Set content type (security headers already set above)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	// Cache-Control is set in setSecurityHeaders, but we ensure it's set here too
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Try cache first if enabled
	var responseSize int64
	if h.config.CacheEnabled && h.config.CacheTTL > 0 {
		h.cacheMu.RLock()
		cached := h.cachedResp
		if cached != nil && time.Since(cached.timestamp) < h.config.CacheTTL && len(cached.data) > 0 {
			data := cached.data
			h.cacheMu.RUnlock()

			// Serve from cache
			written, err := w.Write(data)
			if err == nil {
				duration := time.Since(start)
				h.recordMetrics(r, duration, http.StatusOK, int64(written))
				h.logRequestComplete(r, duration, http.StatusOK, int64(written), true)
				return
			}
			// If write fails, fall through to normal path
		} else {
			h.cacheMu.RUnlock()
		}
	}

	// Gather metrics with timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.config.GatherTimeout)
	defer cancel()

	// Gather metrics
	metricFamilies, err := h.gatherMetrics(ctx)
	if err != nil {
		duration := time.Since(start)
		h.handleError(w, r, err, duration)
		return
	}

	// Write response
	responseSize, err = h.writeResponse(w, metricFamilies)
	if err != nil {
		duration := time.Since(start)
		h.handleError(w, r, err, duration)
		return
	}

	// Cache response if enabled
	if h.config.CacheEnabled && h.config.CacheTTL > 0 && responseSize > 0 {
		h.cacheMu.Lock()
		// Re-encode for cache (or reuse buffer from writeResponse)
		buf := getBuffer()
		encoder := expfmt.NewEncoder(buf, expfmt.FmtText)
		for _, family := range metricFamilies {
			encoder.Encode(family)
		}
		h.cachedResp.data = make([]byte, buf.Len())
		copy(h.cachedResp.data, buf.Bytes())
		h.cachedResp.timestamp = time.Now()
		putBuffer(buf)
		h.cacheMu.Unlock()
	}

	// Record metrics
	duration := time.Since(start)
	h.recordMetrics(r, duration, http.StatusOK, responseSize)

	// Log request completion with performance metrics
	h.logRequestComplete(r, duration, http.StatusOK, responseSize, false)
}

// gatherMetrics gathers all metrics from registered collectors.
// Optimized version: uses direct call instead of goroutine+channel for better performance.
func (h *MetricsEndpointHandler) gatherMetrics(ctx context.Context) ([]*dto.MetricFamily, error) {
	// Check context first (fast path)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Direct gather (faster than goroutine+channel for typical cases)
	// For timeout protection, we rely on context cancellation at higher level
	families, err := h.gatherer.Gather()

	// Check context again after gather
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return families, err
	}
}

// bufferPool is a pool of bytes.Buffer for reuse to reduce allocations.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// getBuffer gets a buffer from the pool.
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool.
func putBuffer(buf *bytes.Buffer) {
	// Don't pool buffers larger than 1MB to avoid memory bloat
	if buf.Len() > 1024*1024 {
		return
	}
	bufferPool.Put(buf)
}

// writeResponse writes metrics in Prometheus text format.
// Optimized with buffer pooling to reduce allocations.
func (h *MetricsEndpointHandler) writeResponse(w http.ResponseWriter, families []*dto.MetricFamily) (int64, error) {
	// Get buffer from pool for reuse
	buf := getBuffer()
	defer putBuffer(buf)

	// Encode metrics
	encoder := expfmt.NewEncoder(buf, expfmt.FmtText)
	for _, family := range families {
		if err := encoder.Encode(family); err != nil {
			return 0, fmt.Errorf("failed to encode metric family: %w", err)
		}
	}

	responseSize := int64(buf.Len())

	// Check max response size
	if h.config.MaxResponseSize > 0 && responseSize > h.config.MaxResponseSize {
		return 0, fmt.Errorf("response size %d exceeds maximum %d", responseSize, h.config.MaxResponseSize)
	}

	// Write response directly from buffer
	written, err := w.Write(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("failed to write response: %w", err)
	}

	return int64(written), nil
}

// logRequestStart logs the start of a request.
func (h *MetricsEndpointHandler) logRequestStart(r *http.Request) {
	if h.errorHandler == nil {
		return
	}

	defaultHandler, ok := h.errorHandler.(*DefaultErrorHandler)
	if !ok || defaultHandler.logger == nil {
		return
	}

	// Extract request ID from context if available
	requestID := ""
	if reqID := r.Context().Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			requestID = id
		}
	}

	clientIP := extractClientIP(r)
	logger := defaultHandler.logger

	if requestID != "" {
		logger.Debug("metrics endpoint request started",
			"method", r.Method,
			"path", r.URL.Path,
			"client_ip", clientIP,
			"request_id", requestID,
		)
	} else {
		logger.Debug("metrics endpoint request started",
			"method", r.Method,
			"path", r.URL.Path,
			"client_ip", clientIP,
		)
	}
}

// logRequestComplete logs the completion of a request with performance metrics.
func (h *MetricsEndpointHandler) logRequestComplete(r *http.Request, duration time.Duration, statusCode int, responseSize int64, fromCache bool) {
	if h.errorHandler == nil {
		return
	}

	defaultHandler, ok := h.errorHandler.(*DefaultErrorHandler)
	if !ok || defaultHandler.logger == nil {
		return
	}

	// Extract request ID from context if available
	requestID := ""
	if reqID := r.Context().Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			requestID = id
		}
	}

	clientIP := extractClientIP(r)
	logger := defaultHandler.logger

	// Log at appropriate level based on status code and duration
	logArgs := []interface{}{
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
		"duration_sec", duration.Seconds(),
		"response_size_bytes", responseSize,
		"client_ip", clientIP,
		"from_cache", fromCache,
	}

	if requestID != "" {
		logArgs = append(logArgs, "request_id", requestID)
	}

	// Use appropriate log level
	if statusCode >= 500 {
		logger.Error("metrics endpoint request completed with server error", logArgs...)
	} else if statusCode >= 400 {
		logger.Warn("metrics endpoint request completed with client error", logArgs...)
	} else if duration > 1*time.Second {
		// Log slow requests at warn level
		logger.Warn("metrics endpoint request completed (slow)", logArgs...)
	} else {
		logger.Info("metrics endpoint request completed", logArgs...)
	}
}

// handleError handles errors gracefully.
func (h *MetricsEndpointHandler) handleError(w http.ResponseWriter, r *http.Request, err error, duration time.Duration) {
	// Log error with context
	h.errorHandler.LogError(r.Context(), err)

	// Record error metric
	h.mu.RLock()
	errorsCounter := h.requestErrors
	h.mu.RUnlock()

	if errorsCounter != nil {
		errorsCounter.Inc()
	}

	// Determine status code based on error type
	statusCode := http.StatusInternalServerError
	if err == context.DeadlineExceeded || err == context.Canceled {
		statusCode = http.StatusRequestTimeout
	}

	// Log request completion with error
	h.logRequestComplete(r, duration, statusCode, 0, false)

	// Try to return partial metrics if possible
	if h.errorHandler.ShouldReturnPartialMetrics(err) {
		// Try to gather metrics without timeout
		families, gatherErr := h.gatherer.Gather()
		if gatherErr == nil && len(families) > 0 {
			// Return partial metrics
			w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("X-Metrics-Partial", "true")
			w.Header().Set("X-Metrics-Error", err.Error())
			_, writeErr := h.writeResponse(w, families)
			if writeErr != nil {
				// If writing partial metrics fails, fall through to 500
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// Status code already determined above, use it for error response

	// Return error response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "Error gathering metrics: %v\n", err)
}

// recordMetrics records self-observability metrics.
// Optimized: avoids locking when metrics are not enabled.
func (h *MetricsEndpointHandler) recordMetrics(r *http.Request, duration time.Duration, status int, size int64) {
	// Fast path: check if metrics are initialized without lock
	h.mu.RLock()
	requestsTotal := h.requestsTotal
	requestDuration := h.requestDuration
	requestSize := h.requestSize
	h.mu.RUnlock()

	// Record metrics without holding lock
	if requestsTotal != nil {
		requestsTotal.Inc()
	}
	if requestDuration != nil {
		requestDuration.Observe(duration.Seconds())
	}
	if requestSize != nil && size > 0 {
		requestSize.Observe(float64(size))
	}
}

// initSelfMetrics initializes self-observability metrics.
func (h *MetricsEndpointHandler) initSelfMetrics() {
	namespace := "alert_history"
	subsystem := "metrics_endpoint"

	h.mu.Lock()
	defer h.mu.Unlock()

	h.requestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "Total number of requests to /metrics endpoint",
	})

	h.requestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help:      "Duration of /metrics endpoint requests",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	})

	h.requestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "errors_total",
		Help:      "Total number of errors in /metrics endpoint",
	})

	h.requestSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "response_size_bytes",
		Help:      "Size of /metrics endpoint responses",
		Buckets:   prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to 1MB
	})

	h.activeRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "active_requests",
		Help:      "Number of active requests to /metrics endpoint",
	})

	// Register self-metrics
	h.registry.MustRegister(
		h.requestsTotal,
		h.requestDuration,
		h.requestErrors,
		h.requestSize,
		h.activeRequests,
	)
}

// RegisterMetricsRegistry registers all metrics from the unified MetricsRegistry.
//
// MetricsRegistry uses promauto, which automatically registers metrics
// in prometheus.DefaultRegisterer. The handler's gatherer already includes
// DefaultGatherer, so all metrics will be available. This method ensures
// lazy initialization and validates the registry.
//
// Example:
//
//	registry := metrics.DefaultRegistry()
//	err := handler.RegisterMetricsRegistry(registry)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (h *MetricsEndpointHandler) RegisterMetricsRegistry(registry *MetricsRegistry) error {
	if registry == nil {
		return fmt.Errorf("metrics registry is nil")
	}

	// Access metrics to ensure they're initialized (lazy initialization)
	// This triggers the creation of all metric instances via promauto
	business := registry.Business()
	technical := registry.Technical()
	infra := registry.Infra()

	// Validate that metrics were initialized
	if business == nil {
		return fmt.Errorf("failed to initialize business metrics")
	}
	if technical == nil {
		return fmt.Errorf("failed to initialize technical metrics")
	}
	if infra == nil {
		return fmt.Errorf("failed to initialize infra metrics")
	}

	// MetricsRegistry uses promauto, so metrics are automatically registered
	// in prometheus.DefaultRegisterer. Our gatherer includes DefaultGatherer,
	// so all metrics will be available without explicit registration.
	// This method ensures initialization and validates the registry.

	return nil
}

// RegisterHTTPMetrics registers HTTP metrics for collection.
//
// HTTPMetrics are typically created via MetricsManager and registered
// with promauto. This method validates that HTTPMetrics are accessible
// and ensures they're included in the metrics response.
//
// Example:
//
//	metricsManager := metrics.NewMetricsManager(...)
//	httpMetrics := metricsManager.Metrics()
//	err := handler.RegisterHTTPMetrics(httpMetrics)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (h *MetricsEndpointHandler) RegisterHTTPMetrics(httpMetrics *HTTPMetrics) error {
	if httpMetrics == nil {
		return fmt.Errorf("HTTP metrics is nil")
	}

	// HTTPMetrics uses promauto, so metrics are already registered
	// in prometheus.DefaultRegisterer. Our gatherer includes DefaultGatherer,
	// so all metrics will be available without explicit registration.
	// This method validates the metrics instance.

	return nil
}

// GetRegistry returns the Prometheus registry used by the handler.
//
// The registry contains self-observability metrics and optionally
// Go runtime/process metrics. Use this for advanced metric management.
//
// Example:
//
//	registry := handler.GetRegistry()
//	// Register additional collectors if needed
//	registry.MustRegister(customCollector)
func (h *MetricsEndpointHandler) GetRegistry() *prometheus.Registry {
	return h.registry
}
