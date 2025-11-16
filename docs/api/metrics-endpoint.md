# Metrics Endpoint API Documentation

**Comprehensive API documentation for GET /metrics endpoint (TN-65).**

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [HTTP API](#http-api)
3. [Go API](#go-api)
4. [Configuration](#configuration)
5. [Response Format](#response-format)
6. [Error Handling](#error-handling)
7. [Security](#security)
8. [Examples](#examples)

---

## Overview

The `/metrics` endpoint provides Prometheus-compatible metrics in text exposition format. It's designed for enterprise-grade production use with features including:

- **Performance Optimization**: Optional caching, buffer pooling, optimized gathering
- **Security**: Rate limiting, security headers, request validation
- **Observability**: Self-observability metrics, structured logging
- **Reliability**: Graceful error handling, partial metrics support

**Protocol:** HTTP/1.1
**Format:** Prometheus Text Exposition Format (0.0.4)
**Encoding:** UTF-8
**Default Path:** `/metrics`

---

## HTTP API

### GET /metrics

Returns Prometheus metrics in text exposition format.

#### Request

```http
GET /metrics HTTP/1.1
Host: localhost:8080
Accept: text/plain
```

**Method:** `GET` (only method allowed)

**Path:** `/metrics` (configurable via `EndpointConfig.Path`)

**Query Parameters:** None (all query parameters are rejected for security)

**Request Headers:**
- `Accept: text/plain` (optional, default)
- `X-Forwarded-For: <ip>` (for rate limiting behind proxy)
- `X-Real-IP: <ip>` (for rate limiting behind proxy)

#### Response

**Status:** `200 OK`

**Headers:**
- `Content-Type: text/plain; version=0.0.4; charset=utf-8`
- `Cache-Control: no-cache, no-store, must-revalidate, max-age=0`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` (HTTPS only)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`

**Body:** Prometheus text exposition format

```
# HELP alert_history_metrics_endpoint_requests_total Total number of requests to /metrics endpoint
# TYPE alert_history_metrics_endpoint_requests_total counter
alert_history_metrics_endpoint_requests_total 42

# HELP alert_history_metrics_endpoint_request_duration_seconds Duration of /metrics endpoint requests
# TYPE alert_history_metrics_endpoint_request_duration_seconds histogram
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="0.001"} 10
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="0.005"} 25
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="0.01"} 35
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="+Inf"} 42
alert_history_metrics_endpoint_request_duration_seconds_sum 0.234
alert_history_metrics_endpoint_request_duration_seconds_count 42

# HELP alert_history_business_webhook_events_total Total webhook events received
# TYPE alert_history_business_webhook_events_total counter
alert_history_business_webhook_events_total{alertname="CPUThrottlingHigh",status="firing"} 100
```

#### Error Responses

**429 Too Many Requests** (Rate Limit Exceeded)
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after 60 seconds.",
  "limit": 60,
  "retry_after": 60
}
```
Headers:
- `X-RateLimit-Limit: 60`
- `X-RateLimit-Remaining: 0`
- `X-RateLimit-Reset: <unix_timestamp>`
- `Retry-After: 60`

**408 Request Timeout** (Gathering Timeout)
```
Error gathering metrics: context deadline exceeded
```
Headers:
- `X-Metrics-Partial: true` (if partial metrics returned)
- `X-Metrics-Error: context deadline exceeded`

**500 Internal Server Error** (Other Errors)
```
Error gathering metrics: <error message>
```

**405 Method Not Allowed** (Non-GET Request)
```
Method not allowed
```
Headers:
- `Allow: GET`

**404 Not Found** (Invalid Path)
```
404 page not found
```

**400 Bad Request** (Invalid Query Parameters)
```
Invalid query parameters
```

---

## Go API

### Package: `github.com/vitaliisemenov/alert-history/pkg/metrics`

### Types

#### `MetricsEndpointHandler`

Main handler for the `/metrics` endpoint.

```go
type MetricsEndpointHandler struct {
    // ... internal fields
}
```

**Methods:**

- `ServeHTTP(w http.ResponseWriter, r *http.Request)` - Implements `http.Handler`
- `SetLogger(logger Logger)` - Sets logger for structured logging
- `RegisterMetricsRegistry(registry *MetricsRegistry) error` - Registers unified metrics registry
- `RegisterHTTPMetrics(metrics *HTTPMetrics) error` - Registers HTTP metrics
- `GetRegistry() *prometheus.Registry` - Returns Prometheus registry

#### `EndpointConfig`

Configuration for the metrics endpoint.

```go
type EndpointConfig struct {
    // Path for the metrics endpoint (default: "/metrics")
    Path string

    // Enable Go runtime metrics
    EnableGoRuntime bool

    // Enable process metrics
    EnableProcess bool

    // Timeout for gathering metrics
    GatherTimeout time.Duration

    // Maximum response size (0 = unlimited)
    MaxResponseSize int64

    // Enable self-observability metrics
    EnableSelfMetrics bool

    // Custom gatherer (optional)
    CustomGatherer prometheus.Gatherer

    // Cache configuration (optional, for performance)
    CacheEnabled bool
    CacheTTL     time.Duration

    // Rate limiting configuration (optional, for security)
    RateLimitEnabled   bool
    RateLimitPerMinute int // Requests per minute per client
    RateLimitBurst     int // Burst capacity

    // Security headers configuration
    EnableSecurityHeaders bool
}
```

#### `DefaultEndpointConfig()`

Returns default configuration:

```go
func DefaultEndpointConfig() EndpointConfig {
    return EndpointConfig{
        Path:              "/metrics",
        EnableGoRuntime:   false,
        EnableProcess:     false,
        GatherTimeout:     5 * time.Second,
        MaxResponseSize:   10 * 1024 * 1024, // 10MB
        EnableSelfMetrics: true,
        CacheEnabled:      false,
        CacheTTL:          0,
        RateLimitEnabled:  true,
        RateLimitPerMinute: 60,
        RateLimitBurst:     10,
        EnableSecurityHeaders: true,
    }
}
```

#### `NewMetricsEndpointHandler()`

Creates a new metrics endpoint handler.

```go
func NewMetricsEndpointHandler(
    config EndpointConfig,
    registry *MetricsRegistry,
) (*MetricsEndpointHandler, error)
```

**Parameters:**
- `config`: Endpoint configuration
- `registry`: Optional unified metrics registry

**Returns:**
- `*MetricsEndpointHandler`: Handler instance
- `error`: Error if failed to create handler

#### `Logger` Interface

Structured logging interface.

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

---

## Configuration

### Default Configuration

```go
config := metrics.DefaultEndpointConfig()
```

**Default Values:**
- Path: `/metrics`
- Go Runtime Metrics: Disabled (performance)
- Process Metrics: Disabled (security)
- Gather Timeout: 5 seconds
- Max Response Size: 10MB
- Self Metrics: Enabled
- Cache: Disabled
- Rate Limiting: Enabled (60 req/min, burst 10)
- Security Headers: Enabled

### Custom Configuration Examples

#### High-Traffic Configuration

```go
config := metrics.DefaultEndpointConfig()
config.CacheEnabled = true
config.CacheTTL = 5 * time.Second
config.RateLimitPerMinute = 200
config.RateLimitBurst = 50
```

#### Development Configuration

```go
config := metrics.DefaultEndpointConfig()
config.EnableGoRuntime = true
config.EnableProcess = true
config.RateLimitEnabled = false
config.EnableSecurityHeaders = false
```

#### Production Configuration

```go
config := metrics.DefaultEndpointConfig()
config.CacheEnabled = true
config.CacheTTL = 10 * time.Second
config.RateLimitPerMinute = 100
config.RateLimitBurst = 20
config.GatherTimeout = 10 * time.Second
```

---

## Response Format

### Prometheus Text Exposition Format

The endpoint returns metrics in [Prometheus Text Exposition Format](https://prometheus.io/docs/instrumenting/exposition_formats/#text-format-example) version 0.0.4.

**Format:**
```
# HELP <metric_name> <help_text>
# TYPE <metric_name> <metric_type>
<metric_name>{<labels>} <value>
```

**Metric Types:**
- `counter` - Monotonically increasing counter
- `gauge` - Value that can go up or down
- `histogram` - Distribution of values
- `summary` - Summary statistics

**Example:**
```
# HELP alert_history_metrics_endpoint_requests_total Total number of requests to /metrics endpoint
# TYPE alert_history_metrics_endpoint_requests_total counter
alert_history_metrics_endpoint_requests_total 42

# HELP alert_history_metrics_endpoint_request_duration_seconds Duration of /metrics endpoint requests
# TYPE alert_history_metrics_endpoint_request_duration_seconds histogram
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="0.001"} 10
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="0.005"} 25
alert_history_metrics_endpoint_request_duration_seconds_bucket{le="+Inf"} 42
alert_history_metrics_endpoint_request_duration_seconds_sum 0.234
alert_history_metrics_endpoint_request_duration_seconds_count 42
```

---

## Error Handling

### Error Types

1. **Rate Limit Exceeded** (429)
   - Too many requests from same client
   - Returns JSON error with retry information

2. **Request Timeout** (408)
   - Metrics gathering exceeded timeout
   - May return partial metrics if available

3. **Internal Server Error** (500)
   - Unexpected errors during gathering
   - Error message in response body

4. **Method Not Allowed** (405)
   - Non-GET requests rejected
   - `Allow: GET` header included

5. **Not Found** (404)
   - Invalid path (not `/metrics`)

6. **Bad Request** (400)
   - Invalid query parameters

### Graceful Degradation

On timeout, the endpoint attempts to return partial metrics:
- Sets `X-Metrics-Partial: true` header
- Sets `X-Metrics-Error: <error>` header
- Returns available metrics with 408 status

---

## Security

### Rate Limiting

- **Type:** Per-client (by IP address)
- **Algorithm:** Token Bucket
- **Default:** 60 requests/minute, burst 10
- **Headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`

### Security Headers

All responses include security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`
- `Strict-Transport-Security` (HTTPS only)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `Cache-Control: no-cache, no-store, must-revalidate, max-age=0`

### Request Validation

- Only `GET` method allowed
- Exact path match required (`/metrics`)
- All query parameters rejected

---

## Examples

### Basic Usage

```go
package main

import (
    "net/http"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

func main() {
    // Create handler with default config
    config := metrics.DefaultEndpointConfig()
    handler, err := metrics.NewMetricsEndpointHandler(config, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Register endpoint
    http.Handle("/metrics", handler)
    http.ListenAndServe(":8080", nil)
}
```

### With Unified Metrics Registry

```go
import (
    "net/http"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

func main() {
    // Create unified metrics registry
    registry := metrics.DefaultRegistry()

    // Create handler with registry
    config := metrics.DefaultEndpointConfig()
    handler, err := metrics.NewMetricsEndpointHandler(config, registry)
    if err != nil {
        log.Fatal(err)
    }

    // Register endpoint
    http.Handle("/metrics", handler)
    http.ListenAndServe(":8080", nil)
}
```

### With Structured Logging

```go
import (
    "log/slog"
    "net/http"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

func main() {
    // Create logger
    logger := slog.Default()

    // Create handler
    config := metrics.DefaultEndpointConfig()
    handler, err := metrics.NewMetricsEndpointHandler(config, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Set logger (adapts slog.Logger to metrics.Logger)
    handler.SetLogger(&metricsLoggerAdapter{logger: logger})

    // Register endpoint
    http.Handle("/metrics", handler)
    http.ListenAndServe(":8080", nil)
}

// metricsLoggerAdapter adapts slog.Logger to metrics.Logger
type metricsLoggerAdapter struct {
    logger *slog.Logger
}

func (a *metricsLoggerAdapter) Debug(msg string, args ...interface{}) {
    a.logger.Debug(msg, args...)
}

func (a *metricsLoggerAdapter) Info(msg string, args ...interface{}) {
    a.logger.Info(msg, args...)
}

func (a *metricsLoggerAdapter) Warn(msg string, args ...interface{}) {
    a.logger.Warn(msg, args...)
}

func (a *metricsLoggerAdapter) Error(msg string, args ...interface{}) {
    a.logger.Error(msg, args...)
}
```

### With Custom Configuration

```go
config := metrics.DefaultEndpointConfig()
config.CacheEnabled = true
config.CacheTTL = 5 * time.Second
config.RateLimitPerMinute = 100
config.RateLimitBurst = 20
config.GatherTimeout = 10 * time.Second

handler, err := metrics.NewMetricsEndpointHandler(config, registry)
```

### cURL Examples

**Basic Request:**
```bash
curl http://localhost:8080/metrics
```

**With Rate Limit Headers:**
```bash
curl -v http://localhost:8080/metrics
# Response includes:
# X-RateLimit-Limit: 60
# X-RateLimit-Remaining: 59
```

**Rate Limited Request:**
```bash
# Make 61 requests quickly
for i in {1..61}; do curl http://localhost:8080/metrics > /dev/null; done
# Last request returns 429 with JSON error
```

---

## Self-Observability Metrics

The endpoint exposes its own metrics:

- `alert_history_metrics_endpoint_requests_total` (Counter)
- `alert_history_metrics_endpoint_request_duration_seconds` (Histogram)
- `alert_history_metrics_endpoint_errors_total` (Counter)
- `alert_history_metrics_endpoint_response_size_bytes` (Histogram)
- `alert_history_metrics_endpoint_active_requests` (Gauge)

These metrics are included in the `/metrics` response and can be used to monitor the endpoint itself.

---

## Performance

### Caching

Optional in-memory caching with TTL:
- **Enabled:** `config.CacheEnabled = true`
- **TTL:** `config.CacheTTL = 5 * time.Second`
- **Performance:** ~66x faster latency, ~71x higher throughput

### Buffer Pooling

Automatic buffer pooling for serialization:
- Reduces allocations by ~99% with cache
- Reuses buffers for better memory efficiency

### Benchmarks

**Without Cache:**
- Latency (P95): ~210ms
- Throughput: ~5,481 req/s
- Memory: ~208KB per request

**With Cache (5s TTL):**
- Latency (P95): ~3.2ms (66x faster)
- Throughput: ~388K req/s (71x higher)
- Memory: ~19KB per request (11x less)

---

## See Also

- [Integration Guide](../guides/metrics-integration.md)
- [Troubleshooting Guide](../runbooks/metrics-endpoint-troubleshooting.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
