# TN-68: GET /publishing/mode - Current Mode - Design Document

**Version**: 1.0
**Date**: 2025-11-17
**Status**: Design Complete ✅
**Quality Target**: 150%+ (Grade A+, Enterprise-Grade)
**Branch**: `feature/TN-68-publishing-mode-endpoint-150pct`

---

## 📋 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Data Models](#3-data-models)
4. [API Design](#4-api-design)
5. [Security Design](#5-security-design)
6. [Performance Design](#6-performance-design)
7. [Observability Design](#7-observability-design)
8. [Error Handling Design](#8-error-handling-design)
9. [Testing Strategy](#9-testing-strategy)
10. [Deployment Strategy](#10-deployment-strategy)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                        Client Layer                                │
│  (curl, Frontend Dashboard, Monitoring Tools, CI/CD)               │
└────────────────────────┬──────────────────────────────────────────┘
                         │
                         │ HTTP GET
                         │
┌────────────────────────▼──────────────────────────────────────────┐
│                   API Gateway / Load Balancer                      │
│               (Kubernetes Ingress / Service)                       │
└────────────────────────┬──────────────────────────────────────────┘
                         │
          ┌──────────────┴─────────────┐
          │                            │
          ▼                            ▼
┌──────────────────────────┐   ┌──────────────────────────┐
│  /api/v1/publishing/mode │   │  /api/v2/publishing/mode │
│   (Existing, Enhanced)   │   │      (New, Consistent)   │
└──────────────────────────┘   └──────────────────────────┘
          │                            │
          │                            │
          └─────────────┬──────────────┘
                        │
                        │ Route to Handler
                        │
┌───────────────────────▼───────────────────────────────────────────┐
│                  Middleware Stack (Applied)                        │
│  1. Recovery Middleware (panic recovery)                           │
│  2. RequestID Middleware (UUID tracking)                           │
│  3. Logging Middleware (structured logs)                           │
│  4. Metrics Middleware (Prometheus metrics)                        │
│  5. RateLimit Middleware (60 req/min per IP)                       │
│  6. Security Headers Middleware (9 headers)                        │
│  7. Compression Middleware (gzip)                                  │
│  8. Cache Middleware (Cache-Control, ETag)                         │
└───────────────────────┬───────────────────────────────────────────┘
                        │
                        │ Invoke Handler
                        │
┌───────────────────────▼───────────────────────────────────────────┐
│               PublishingModeHandler (Shared)                       │
│  • GetPublishingMode(w http.ResponseWriter, r *http.Request)      │
│  • Location: go-app/internal/api/handlers/publishing/mode.go      │
│  • Purpose: Handle GET requests for publishing mode info          │
└───────────────────────┬───────────────────────────────────────────┘
                        │
                        │ Delegate to Service Layer
                        │
┌───────────────────────▼───────────────────────────────────────────┐
│              PublishingModeService (Business Logic)                │
│  • GetCurrentModeInfo() (*ModeInfo, error)                         │
│  • Location: go-app/internal/api/services/publishing/mode.go      │
│  • Purpose: Orchestrate mode detection, caching, metrics           │
└───────────────────────┬───────────────────────────────────────────┘
                        │
           ┌────────────┴─────────────┐
           │                          │
           ▼                          ▼
┌──────────────────────┐   ┌──────────────────────────┐
│    ModeManager       │   │  TargetDiscoveryManager  │
│  (TN-060 Component)  │   │   (TN-047 Component)     │
│  • GetCurrentMode()  │   │  • ListTargets()         │
│  • IsMetricsOnly()   │   │  • Count enabled targets │
│  • GetModeMetrics()  │   └──────────────────────────┘
│  • Thread-safe       │
│  • Cached (1s TTL)   │
└──────────────────────┘

Response Flow:
Service → Handler → Middleware → Client
  (JSON response with mode info)
```

### 1.2 Architectural Patterns

| Pattern | Usage | Rationale |
|---------|-------|-----------|
| **Hexagonal Architecture** | Overall structure | Clean separation: Handlers → Services → Domain |
| **Dependency Injection** | Constructor-based DI | Testability, flexibility, loose coupling |
| **Interface Segregation** | ModeManager, DiscoveryManager | Decoupling, mockability |
| **Adapter Pattern** | Handler → Service → Repositories | Abstraction layers |
| **Strategy Pattern** | Mode detection (ModeManager vs Fallback) | Graceful degradation |
| **Observer Pattern** | ModeManager subscriptions | Event-driven updates |
| **Singleton Pattern** | Prometheus metrics | Global metrics registry |

### 1.3 Component Relationships

```
┌───────────────────────────────────────────────────────────────────┐
│                     Presentation Layer                             │
│  • HTTP Handlers (mode.go)                                         │
│  • Request validation                                              │
│  • Response serialization                                          │
│  • Error handling                                                  │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                            │ Calls
                            │
┌───────────────────────────▼───────────────────────────────────────┐
│                      Service Layer                                 │
│  • PublishingModeService                                           │
│  • Business logic orchestration                                    │
│  • Caching logic                                                   │
│  • Metrics aggregation                                             │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                ┌───────────┴────────────┐
                │                        │
                ▼                        ▼
┌─────────────────────────┐   ┌─────────────────────────┐
│     Domain Layer        │   │   Infrastructure Layer   │
│  • Mode (enum)          │   │  • ModeManager           │
│  • ModeInfo (struct)    │   │  • DiscoveryManager      │
│  • ModeMetrics (struct) │   │  • Cache                 │
└─────────────────────────┘   │  • Prometheus Metrics    │
                              └─────────────────────────┘
```

---

## 2. Component Design

### 2.1 Handler Component

**File**: `go-app/internal/api/handlers/publishing/mode.go` (NEW)

```go
package publishing

import (
    "encoding/json"
    "net/http"
    "time"

    "log/slog"
    "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/middleware"
)

// PublishingModeHandler handles GET /publishing/mode requests
type PublishingModeHandler struct {
    service publishing.ModeService
    logger  *slog.Logger
}

// NewPublishingModeHandler creates a new handler
func NewPublishingModeHandler(service publishing.ModeService, logger *slog.Logger) *PublishingModeHandler {
    if logger == nil {
        logger = slog.Default()
    }
    return &PublishingModeHandler{
        service: service,
        logger:  logger,
    }
}

// GetPublishingMode handles GET requests
func (h *PublishingModeHandler) GetPublishingMode(w http.ResponseWriter, r *http.Request) {
    // Extract request ID from context
    requestID := middleware.GetRequestID(r.Context())

    // Log request start
    h.logger.Info("Handling GET /publishing/mode",
        "request_id", requestID,
        "method", r.Method,
        "path", r.URL.Path)

    // Get mode info from service
    startTime := time.Now()
    modeInfo, err := h.service.GetCurrentModeInfo(r.Context())
    duration := time.Since(startTime)

    // Handle errors
    if err != nil {
        h.logger.Error("Failed to get mode info",
            "request_id", requestID,
            "error", err,
            "duration_ms", duration.Milliseconds())

        h.sendError(w, http.StatusInternalServerError, "Internal server error", requestID)
        return
    }

    // Log success
    h.logger.Info("Successfully retrieved mode info",
        "request_id", requestID,
        "mode", modeInfo.Mode,
        "enabled_targets", modeInfo.EnabledTargets,
        "duration_ms", duration.Milliseconds())

    // Set caching headers
    h.setCacheHeaders(w, modeInfo)

    // Send JSON response
    h.sendJSON(w, http.StatusOK, modeInfo)
}

// setCacheHeaders sets HTTP caching headers
func (h *PublishingModeHandler) setCacheHeaders(w http.ResponseWriter, modeInfo *publishing.ModeInfo) {
    // Cache for 5 seconds (aligned with ModeManager periodic check)
    w.Header().Set("Cache-Control", "max-age=5, public")

    // Generate ETag based on mode and transition count
    etag := fmt.Sprintf(`"%s-%d-%d"`, modeInfo.Mode, modeInfo.EnabledTargets, modeInfo.TransitionCount)
    w.Header().Set("ETag", etag)
}

// sendJSON sends JSON response
func (h *PublishingModeHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)

    if err := json.NewEncoder(w).Encode(data); err != nil {
        h.logger.Error("Failed to encode JSON", "error", err)
    }
}

// sendError sends error response
func (h *PublishingModeHandler) sendError(w http.ResponseWriter, status int, message string, requestID string) {
    errorResponse := ErrorResponse{
        Error:     http.StatusText(status),
        Message:   message,
        RequestID: requestID,
        Timestamp: time.Now(),
    }
    h.sendJSON(w, status, errorResponse)
}
```

### 2.2 Service Component

**File**: `go-app/internal/api/services/publishing/mode.go` (NEW)

```go
package publishing

import (
    "context"
    "time"

    "log/slog"
    infrapublishing "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// ModeService provides business logic for publishing mode operations
type ModeService interface {
    GetCurrentModeInfo(ctx context.Context) (*ModeInfo, error)
}

// DefaultModeService implements ModeService
type DefaultModeService struct {
    modeManager      infrapublishing.ModeManager
    discoveryManager infrapublishing.TargetDiscoveryManager
    logger           *slog.Logger
}

// NewModeService creates a new mode service
func NewModeService(
    modeManager infrapublishing.ModeManager,
    discoveryManager infrapublishing.TargetDiscoveryManager,
    logger *slog.Logger,
) ModeService {
    if logger == nil {
        logger = slog.Default()
    }

    return &DefaultModeService{
        modeManager:      modeManager,
        discoveryManager: discoveryManager,
        logger:           logger,
    }
}

// GetCurrentModeInfo returns current mode information
func (s *DefaultModeService) GetCurrentModeInfo(ctx context.Context) (*ModeInfo, error) {
    // Use ModeManager if available (TN-060 integration)
    if s.modeManager != nil {
        return s.getModeInfoFromManager(ctx)
    }

    // Fallback to basic mode detection (backward compatibility)
    return s.getModeInfoFallback(ctx)
}

// getModeInfoFromManager gets mode info from ModeManager (enhanced)
func (s *DefaultModeService) getModeInfoFromManager(ctx context.Context) (*ModeInfo, error) {
    // Get current mode and metrics from ModeManager
    currentMode := s.modeManager.GetCurrentMode()
    modeMetrics := s.modeManager.GetModeMetrics()

    // Count enabled targets
    targets := s.discoveryManager.ListTargets()
    enabledCount := 0
    for _, t := range targets {
        if t.Enabled {
            enabledCount++
        }
    }
    targetsAvailable := enabledCount > 0

    // Build response
    modeInfo := &ModeInfo{
        Mode:                      currentMode.String(),
        TargetsAvailable:          targetsAvailable,
        EnabledTargets:            enabledCount,
        MetricsOnlyActive:         currentMode == infrapublishing.ModeMetricsOnly,
        TransitionCount:           modeMetrics.TransitionCount,
        CurrentModeDurationSeconds: modeMetrics.CurrentModeDuration.Seconds(),
        LastTransitionTime:        modeMetrics.LastTransitionTime,
        LastTransitionReason:      modeMetrics.LastTransitionReason,
    }

    return modeInfo, nil
}

// getModeInfoFallback gets mode info using basic detection (fallback)
func (s *DefaultModeService) getModeInfoFallback(ctx context.Context) (*ModeInfo, error) {
    // Count enabled targets
    targets := s.discoveryManager.ListTargets()
    enabledCount := 0
    for _, t := range targets {
        if t.Enabled {
            enabledCount++
        }
    }
    targetsAvailable := enabledCount > 0

    // Determine mode
    mode := "normal"
    metricsOnly := false
    if !targetsAvailable {
        mode = "metrics-only"
        metricsOnly = true
    }

    // Build response (basic fields only)
    modeInfo := &ModeInfo{
        Mode:              mode,
        TargetsAvailable:  targetsAvailable,
        EnabledTargets:    enabledCount,
        MetricsOnlyActive: metricsOnly,
        // Enhanced fields omitted in fallback mode
    }

    return modeInfo, nil
}
```

### 2.3 Data Models

**File**: `go-app/internal/api/services/publishing/models.go` (NEW)

```go
package publishing

import "time"

// ModeInfo represents current publishing mode information
type ModeInfo struct {
    // Basic fields (always present)
    Mode              string `json:"mode"`                // "normal" or "metrics-only"
    TargetsAvailable  bool   `json:"targets_available"`  // Whether any targets available
    EnabledTargets    int    `json:"enabled_targets"`    // Count of enabled targets
    MetricsOnlyActive bool   `json:"metrics_only_active"` // Whether in metrics-only mode

    // Enhanced fields (present if ModeManager available, TN-060)
    TransitionCount           int64     `json:"transition_count,omitempty"`            // Number of mode transitions
    CurrentModeDurationSeconds float64   `json:"current_mode_duration_seconds,omitempty"` // Duration in current mode
    LastTransitionTime        time.Time `json:"last_transition_time,omitempty"`       // Last transition timestamp
    LastTransitionReason      string    `json:"last_transition_reason,omitempty"`      // Reason for last transition
}

// ErrorResponse represents API error response
type ErrorResponse struct {
    Error     string    `json:"error"`      // HTTP status text
    Message   string    `json:"message"`    // Human-readable error message
    RequestID string    `json:"request_id"` // Request ID for tracing
    Timestamp time.Time `json:"timestamp"`  // Error timestamp
}
```

---

## 3. Data Models

### 3.1 Domain Models

#### Mode (Enum)

```go
// Mode represents the current publishing mode
type Mode int

const (
    // ModeNormal indicates normal publishing mode (targets available)
    ModeNormal Mode = iota
    // ModeMetricsOnly indicates metrics-only mode (no targets available)
    ModeMetricsOnly
)

func (m Mode) String() string {
    switch m {
    case ModeNormal:
        return "normal"
    case ModeMetricsOnly:
        return "metrics-only"
    default:
        return "unknown"
    }
}
```

#### ModeInfo (Response)

```go
type ModeInfo struct {
    // Basic fields
    Mode              string `json:"mode"`                // Required
    TargetsAvailable  bool   `json:"targets_available"`  // Required
    EnabledTargets    int    `json:"enabled_targets"`    // Required
    MetricsOnlyActive bool   `json:"metrics_only_active"` // Required

    // Enhanced fields (TN-060)
    TransitionCount           int64     `json:"transition_count,omitempty"`
    CurrentModeDurationSeconds float64   `json:"current_mode_duration_seconds,omitempty"`
    LastTransitionTime        time.Time `json:"last_transition_time,omitempty"`
    LastTransitionReason      string    `json:"last_transition_reason,omitempty"`
}
```

**Field Descriptions**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | ✅ | Current mode: `"normal"` or `"metrics-only"` |
| `targets_available` | boolean | ✅ | Whether any targets are available |
| `enabled_targets` | integer | ✅ | Count of enabled targets |
| `metrics_only_active` | boolean | ✅ | Whether system is in metrics-only mode |
| `transition_count` | integer | ❌ | Total number of mode transitions since startup |
| `current_mode_duration_seconds` | float64 | ❌ | Duration in current mode (seconds) |
| `last_transition_time` | string | ❌ | RFC3339 timestamp of last transition |
| `last_transition_reason` | string | ❌ | Reason for last transition |

**Transition Reasons**:
- `"targets_available"`: Transition to normal (targets became available)
- `"no_enabled_targets"`: Transition to metrics-only (all targets disabled)
- `"targets_disabled"`: Transition to metrics-only (targets manually disabled)
- `"startup"`: Initial mode at system startup

### 3.2 Error Models

```go
type ErrorResponse struct {
    Error     string    `json:"error"`      // HTTP status text (e.g., "Internal Server Error")
    Message   string    `json:"message"`    // Human-readable message
    RequestID string    `json:"request_id"` // Request ID for tracing
    Timestamp time.Time `json:"timestamp"`  // Error timestamp (RFC3339)
}
```

---

## 4. API Design

### 4.1 API v1 Endpoint (Enhanced)

**Endpoint**: `GET /api/v1/publishing/mode`

**Method**: GET
**Authentication**: None (public endpoint)
**Rate Limiting**: 60 requests/minute per IP (token bucket)

**Request**:
- **Headers**: None required
- **Query Params**: None
- **Body**: Empty

**Response (200 OK - Normal Mode)**:
```json
{
  "mode": "normal",
  "targets_available": true,
  "enabled_targets": 5,
  "metrics_only_active": false,
  "transition_count": 12,
  "current_mode_duration_seconds": 3600.5,
  "last_transition_time": "2025-11-17T10:30:00Z",
  "last_transition_reason": "targets_available"
}
```

**Response (200 OK - Metrics-Only Mode)**:
```json
{
  "mode": "metrics-only",
  "targets_available": false,
  "enabled_targets": 0,
  "metrics_only_active": true,
  "transition_count": 13,
  "current_mode_duration_seconds": 120.3,
  "last_transition_time": "2025-11-17T12:30:00Z",
  "last_transition_reason": "no_enabled_targets"
}
```

**Response (304 Not Modified)**:
- Empty body
- Same ETag as request `If-None-Match` header

**Response (429 Too Many Requests)**:
```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded: 60 requests per minute",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-17T12:35:00Z"
}
```

**Response (500 Internal Server Error)**:
```json
{
  "error": "Internal Server Error",
  "message": "Failed to retrieve mode information",
  "request_id": "550e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2025-11-17T12:36:00Z"
}
```

**Response Headers**:
```
Content-Type: application/json; charset=utf-8
Cache-Control: max-age=5, public
ETag: "normal-5-12"
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
X-Response-Time: 3.2ms

// Security Headers
Content-Security-Policy: default-src 'self'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: no-referrer
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

### 4.2 API v2 Endpoint (New)

**Endpoint**: `GET /api/v2/publishing/mode`

**Identical to v1** (for now, allows future enhancements):
- Same request format
- Same response format
- Same headers
- Same error handling
- Shared handler logic (DRY principle)

**Future Enhancements (v2 only)**:
- Query params support (e.g., `?include=history`)
- Extended response fields
- Versioned breaking changes

---

## 5. Security Design

### 5.1 OWASP Top 10 Compliance

| # | Vulnerability | Mitigation | Status |
|---|---------------|------------|--------|
| 1 | Injection | No user input in queries | ✅ N/A |
| 2 | Broken Authentication | Public endpoint, no auth required | ✅ N/A |
| 3 | Sensitive Data Exposure | No sensitive data in response | ✅ |
| 4 | XML External Entities | No XML parsing | ✅ N/A |
| 5 | Broken Access Control | Public endpoint, no access control | ✅ N/A |
| 6 | Security Misconfiguration | Security headers, rate limiting | ✅ |
| 7 | XSS | No user-generated content, CSP header | ✅ |
| 8 | Insecure Deserialization | No deserialization | ✅ N/A |
| 9 | Components with Vulnerabilities | Dependency management (go.mod) | ✅ |
| 10 | Insufficient Logging & Monitoring | Comprehensive logging | ✅ |

**Compliance**: 8/8 applicable (100%)

### 5.2 Security Headers

```go
// Security headers middleware
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'")

        // Prevent MIME sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")

        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")

        // XSS protection (legacy)
        w.Header().Set("X-XSS-Protection", "1; mode=block")

        // HTTPS enforcement (if HTTPS)
        if r.TLS != nil {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }

        // Referrer policy
        w.Header().Set("Referrer-Policy", "no-referrer")

        // Permissions policy
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

        next.ServeHTTP(w, r)
    })
}
```

### 5.3 Rate Limiting

**Algorithm**: Token Bucket
**Rate**: 60 requests/minute per IP
**Burst**: 10 requests

```go
// Rate limiting middleware (existing, reuse)
func RateLimitMiddleware(requestsPerMinute int, burst int) func(http.Handler) http.Handler {
    limiters := &sync.Map{} // IP -> *rate.Limiter

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract client IP
            ip := extractClientIP(r)

            // Get or create limiter for this IP
            limiterInterface, _ := limiters.LoadOrStore(ip, rate.NewLimiter(
                rate.Limit(requestsPerMinute)/60, // Per second
                burst,
            ))
            limiter := limiterInterface.(*rate.Limiter)

            // Check rate limit
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 5.4 Input Validation

```go
// Request validation
func validateRequest(r *http.Request) error {
    // Method validation
    if r.Method != http.MethodGet {
        return fmt.Errorf("method not allowed: %s", r.Method)
    }

    // Body validation (should be empty or ignored)
    if r.ContentLength > 0 {
        // Ignore body, but log warning
        slog.Warn("Unexpected request body", "content_length", r.ContentLength)
    }

    return nil
}
```

---

## 6. Performance Design

### 6.1 Performance Targets

| Metric | Baseline | Target (100%) | Target (150%) |
|--------|----------|---------------|---------------|
| P50 latency | ~5ms | <5ms | <3ms |
| P95 latency | ~10ms | <10ms | <5ms |
| P99 latency | - | <20ms | <10ms |
| Throughput | - | >1000 req/s | >2000 req/s |
| Memory | - | <500KB | <250KB |
| CPU overhead | - | <0.1% | <0.05% |

### 6.2 Caching Strategy

**Level 1: ModeManager Caching** (existing, TN-060)
- Cached mode with 1s TTL
- Zero-allocation reads
- Thread-safe (sync.RWMutex)
- Performance: 34 ns/op

**Level 2: HTTP Caching** (new, TN-68)
- `Cache-Control: max-age=5, public`
- ETag generation
- Conditional requests (304 Not Modified)
- Client-side caching

```go
// ETag generation
func generateETag(modeInfo *ModeInfo) string {
    return fmt.Sprintf(`"%s-%d-%d"`,
        modeInfo.Mode,
        modeInfo.EnabledTargets,
        modeInfo.TransitionCount)
}

// Conditional request handling
func handleConditionalRequest(r *http.Request, etag string) bool {
    ifNoneMatch := r.Header.Get("If-None-Match")
    return ifNoneMatch == etag
}
```

### 6.3 Optimization Techniques

1. **Zero-Allocation JSON Encoding**
   - Pre-allocate response structs
   - Reuse buffers where possible

2. **Fast Path Optimization**
   - ModeManager cached reads (34 ns/op)
   - No blocking operations
   - No database queries

3. **Connection Pooling**
   - HTTP/2 support (multiplexing)
   - Keep-Alive connections
   - Connection reuse

4. **Compression**
   - gzip middleware (optional)
   - Compress responses > 1KB

---

## 7. Observability Design

### 7.1 Structured Logging

**Log Levels**:
- `DEBUG`: Mode checks, cache hits/misses
- `INFO`: Request start/end, mode info retrieval
- `WARN`: Unexpected conditions (non-fatal)
- `ERROR`: Errors, failures

**Log Format**:
```json
{
  "level": "info",
  "msg": "Handling GET /publishing/mode",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/v1/publishing/mode",
  "timestamp": "2025-11-17T12:30:00Z"
}
```

```go
// Logging example
logger.Info("Successfully retrieved mode info",
    "request_id", requestID,
    "mode", modeInfo.Mode,
    "enabled_targets", modeInfo.EnabledTargets,
    "duration_ms", duration.Milliseconds())
```

### 7.2 Prometheus Metrics

**Metrics**:

1. **Request Counter**
   ```promql
   publishing_mode_api_requests_total{method="GET", path="/api/v1/publishing/mode", status="200"}
   ```

2. **Duration Histogram**
   ```promql
   publishing_mode_api_duration_seconds{method="GET", path="/api/v1/publishing/mode"}
   ```
   Buckets: [0.001, 0.005, 0.010, 0.050, 0.100, 0.500, 1.0]

3. **Error Counter**
   ```promql
   publishing_mode_api_errors_total{method="GET", path="/api/v1/publishing/mode", error_type="internal_error"}
   ```

4. **Cache Metrics**
   ```promql
   publishing_mode_api_cache_hits_total{hit="true|false"}
   publishing_mode_api_cache_size_bytes
   ```

5. **Active Requests Gauge**
   ```promql
   publishing_mode_api_active_requests{method="GET", path="/api/v1/publishing/mode"}
   ```

### 7.3 Distributed Tracing

**Request ID Propagation**:
- Generate UUID per request
- Inject into context
- Include in all logs
- Return in response header (`X-Request-ID`)

```go
// Request ID middleware (existing, reuse)
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)

        w.Header().Set("X-Request-ID", requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 8. Error Handling Design

### 8.1 Error Scenarios

| Scenario | HTTP Status | Error Code | Recovery |
|----------|-------------|------------|----------|
| Success | 200 | - | - |
| Not Modified | 304 | - | Use cached response |
| Bad Method | 405 | method_not_allowed | Reject request |
| Rate Limit Exceeded | 429 | rate_limit_exceeded | Retry after delay |
| ModeManager Unavailable | 200 | - | Fallback to basic detection |
| DiscoveryManager Unavailable | 500 | internal_error | Return error |
| Internal Error | 500 | internal_error | Log and return generic error |
| Panic | 500 | internal_error | Recover and return error |

### 8.2 Error Response Structure

```json
{
  "error": "Internal Server Error",
  "message": "Failed to retrieve mode information",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-17T12:36:00Z"
}
```

### 8.3 Panic Recovery

```go
// Recovery middleware (existing, reuse)
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Log panic
                slog.Error("Panic recovered",
                    "error", err,
                    "stack", string(debug.Stack()))

                // Return 500
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()

        next.ServeHTTP(w, r)
    })
}
```

---

## 9. Testing Strategy

### 9.1 Unit Tests

**Coverage Target**: 90%+

**Test Files**:
- `mode_test.go` - Handler tests
- `mode_service_test.go` - Service tests
- `models_test.go` - Model validation tests

**Test Categories**:
1. **Happy Path Tests** (10+ tests)
   - Normal mode response
   - Metrics-only mode response
   - Fallback mode detection
   - Cache hit/miss scenarios

2. **Error Handling Tests** (10+ tests)
   - ModeManager nil
   - DiscoveryManager nil
   - Internal errors
   - Panic recovery

3. **Edge Case Tests** (5+ tests)
   - Zero targets
   - Many targets (10000+)
   - Rapid mode transitions
   - Concurrent requests

### 9.2 Integration Tests

**Test Scenarios** (10+ tests):
1. End-to-end API request (v1 and v2)
2. Middleware stack integration
3. ModeManager integration
4. DiscoveryManager integration
5. HTTP caching behavior
6. Rate limiting enforcement
7. Security headers presence
8. Request ID propagation
9. Metrics collection
10. Logging integration

### 9.3 Security Tests

**Test Categories** (25+ tests):
1. **OWASP Tests** (8 tests)
2. **Rate Limiting** (5 tests)
3. **Security Headers** (9 tests)
4. **Input Validation** (3 tests)

### 9.4 Benchmarks

**Benchmarks** (5+ benchmarks):
1. `BenchmarkGetPublishingMode` - Overall handler performance
2. `BenchmarkGetPublishingMode_Cached` - With ModeManager caching
3. `BenchmarkGetPublishingMode_Fallback` - Fallback path
4. `BenchmarkGetPublishingMode_Parallel` - Concurrent requests
5. `BenchmarkJSONEncoding` - Response serialization

**Target**: P95 < 5ms, Throughput > 2000 req/s

### 9.5 Load Tests (k6)

**Scenarios**:
1. **Steady State**: 1000 req/s for 5 minutes
2. **Spike**: 0 → 5000 req/s → 0 (1 min spike)
3. **Stress**: Gradually increase to 10000 req/s
4. **Soak**: 500 req/s for 1 hour

**Metrics**:
- P50, P95, P99 latency
- Throughput (req/s)
- Error rate (%)
- Resource usage (CPU, memory)

---

## 10. Deployment Strategy

### 10.1 Git Branch Strategy

**Branch**: `feature/TN-68-publishing-mode-endpoint-150pct`

**Commit Structure**:
1. `docs: Add TN-68 documentation (requirements, design, tasks)`
2. `feat: Add PublishingModeService and models`
3. `feat: Add PublishingModeHandler with API v2 support`
4. `feat: Add HTTP caching headers and ETag support`
5. `feat: Integrate rate limiting and security headers`
6. `test: Add unit tests for handler and service`
7. `test: Add integration tests for API endpoints`
8. `test: Add security tests (OWASP, rate limiting, headers)`
9. `test: Add benchmarks for performance validation`
10. `test: Add k6 load tests`
11. `docs: Add OpenAPI spec and API guide`
12. `docs: Add troubleshooting guide and examples`
13. `chore: Update tasks.md with completion status`

### 10.2 Deployment Phases

**Phase 1: Staging Deployment**
- Deploy to staging environment
- Run comprehensive tests (unit, integration, security)
- Run load tests (k6)
- Monitor metrics and logs
- Performance validation

**Phase 2: Canary Deployment**
- Deploy to 10% of production traffic
- Monitor error rates, latency, resource usage
- Compare metrics with baseline
- Gradual rollout (10% → 25% → 50% → 100%)

**Phase 3: Full Rollout**
- Deploy to 100% of production
- Enable monitoring alerts
- Update documentation
- Communicate to stakeholders

### 10.3 Rollback Plan

**Trigger Conditions**:
- Error rate > 1%
- P95 latency > 20ms
- CPU usage > 80%
- Memory leaks detected

**Rollback Steps**:
1. Revert to previous deployment
2. Disable API v2 endpoint (if needed)
3. Investigate root cause
4. Fix and re-deploy

---

**Design Date**: 2025-11-17
**Author**: AI Assistant (Cursor)
**Status**: ✅ Design Complete, Ready for Implementation
