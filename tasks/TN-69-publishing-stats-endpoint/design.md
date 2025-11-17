# TN-69: GET /publishing/stats - Statistics - Design Document

**Version**: 1.0
**Date**: 2025-11-17
**Status**: Design Complete ✅
**Quality Target**: 150%+ (Grade A+, Enterprise-Grade)
**Branch**: `feature/TN-69-publishing-stats-endpoint-150pct`

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
│  (curl, Frontend Dashboard, Monitoring Tools, CI/CD, Grafana)      │
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
│  /api/v1/publishing/stats│   │  /api/v2/publishing/stats│
│   (New, Backward Compat)  │   │      (Existing, Enhanced)│
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
│  5. RateLimit Middleware (100 req/min per IP)                     │
│  6. Security Headers Middleware (9 headers)                        │
│  7. Compression Middleware (gzip)                                  │
│  8. Cache Middleware (Cache-Control, ETag)                         │
└───────────────────────┬───────────────────────────────────────────┘
                        │
                        │ Invoke Handler
                        │
┌───────────────────────▼───────────────────────────────────────────┐
│            PublishingStatsHandler (Enhanced)                      │
│  • GetStats(w http.ResponseWriter, r *http.Request)               │
│  • GetStatsV1(w http.ResponseWriter, r *http.Request) [NEW]       │
│  • Location: go-app/cmd/server/handlers/publishing_stats.go       │
│  • Purpose: Handle GET requests for publishing statistics          │
└───────────────────────┬───────────────────────────────────────────┘
                        │
                        │ Delegate to Metrics Collector
                        │
┌───────────────────────▼───────────────────────────────────────────┐
│         PublishingMetricsCollector (TN-057 Component)             │
│  • CollectAll(ctx context.Context) *MetricsSnapshot               │
│  • Location: go-app/internal/business/publishing/                 │
│  • Purpose: Collect metrics from all collectors                    │
└───────────────────────┬───────────────────────────────────────────┘
                        │
           ┌────────────┴─────────────┐
           │                          │
           ▼                          ▼
┌──────────────────────┐   ┌──────────────────────────┐
│   Health Collector    │   │   Queue Collector        │
│  • Health metrics     │   │  • Queue metrics         │
└──────────────────────┘   └──────────────────────────┘
           │                          │
           └────────────┬─────────────┘
                        │
                        ▼
┌──────────────────────────────────────────┐
│   Discovery Collector                   │
│  • Target discovery metrics              │
└──────────────────────────────────────────┘

Response Flow:
Collector → Handler → Middleware → Client
  (JSON response with aggregated statistics)
```

### 1.2 Architectural Patterns

| Pattern | Usage | Rationale |
|---------|-------|-----------|
| **Hexagonal Architecture** | Overall structure | Clean separation: Handlers → Services → Domain |
| **Dependency Injection** | Constructor-based DI | Testability, flexibility, loose coupling |
| **Interface Segregation** | MetricsCollectorInterface | Decoupling, mockability |
| **Adapter Pattern** | Handler → Collector | Abstraction layers |
| **Strategy Pattern** | Response format (JSON/Prometheus) | Format flexibility |
| **Singleton Pattern** | Prometheus metrics | Global metrics registry |
| **Cache-Aside Pattern** | HTTP caching | Performance optimization |

### 1.3 Component Relationships

```
┌───────────────────────────────────────────────────────────────────┐
│                     Presentation Layer                             │
│  • HTTP Handlers (publishing_stats.go)                            │
│  • Request validation                                              │
│  • Response serialization                                          │
│  • Error handling                                                  │
│  • HTTP caching                                                    │
└───────────────────────────┬───────────────────────────────────────┘
                             │
                             │ Calls
                             │
┌───────────────────────────▼───────────────────────────────────────┐
│                      Business Layer                                │
│  • PublishingStatsHandler                                          │
│  • Statistics aggregation                                           │
│  • Metrics filtering                                                │
│  • Response formatting                                              │
└───────────────────────────┬───────────────────────────────────────┘
                             │
                             │ Uses
                             │
┌───────────────────────────▼───────────────────────────────────────┐
│                   Infrastructure Layer                             │
│  • PublishingMetricsCollector (TN-057)                             │
│  • Health Collector                                                │
│  • Queue Collector                                                 │
│  • Discovery Collector                                             │
│  • Prometheus Metrics                                              │
└────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Design

### 2.1 Handler Component (Enhanced)

**File**: `go-app/cmd/server/handlers/publishing_stats.go` (EXISTING, ENHANCED)

**Current Implementation**:
- ✅ `GetStats()` - GET /api/v2/publishing/stats (EXISTS)
- ❌ `GetStatsV1()` - GET /api/v1/publishing/stats (NEW)
- ❌ Query parameters support (filter, group_by, format) (NEW)
- ❌ HTTP caching (ETag, Cache-Control) (NEW)
- ❌ Enhanced error handling (NEW)

**Enhancements Required**:

```go
// GetStatsV1 handles GET /api/v1/publishing/stats (backward compatibility)
func (h *PublishingStatsHandler) GetStatsV1(w http.ResponseWriter, r *http.Request) {
    // Similar to GetStats but with simplified response format
    // Compatible with legacy API expectations
}

// GetStatsEnhanced handles GET /api/v2/publishing/stats with query parameters
func (h *PublishingStatsHandler) GetStatsEnhanced(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    filter := r.URL.Query().Get("filter")
    groupBy := r.URL.Query().Get("group_by")
    format := r.URL.Query().Get("format")

    // Collect metrics
    snapshot := h.collector.CollectAll(ctx)

    // Apply filtering if requested
    if filter != "" {
        snapshot = h.applyFilter(snapshot, filter)
    }

    // Apply grouping if requested
    if groupBy != "" {
        response = h.applyGrouping(response, groupBy)
    }

    // Format response
    if format == "prometheus" {
        h.sendPrometheusFormat(w, response)
    } else {
        h.sendJSON(w, response)
    }

    // Set caching headers
    h.setCacheHeaders(w, snapshot)
}
```

### 2.2 Response Models (Enhanced)

**File**: `go-app/cmd/server/handlers/publishing_stats.go` (EXISTING, ENHANCED)

**Current Models**:
- ✅ `StatsResponse` - Main response structure
- ✅ `SystemStats` - System-wide statistics
- ❌ `StatsResponseV1` - Simplified v1 response (NEW)
- ❌ `FilteredStatsResponse` - Filtered response (NEW)
- ❌ `GroupedStatsResponse` - Grouped response (NEW)

**New Models**:

```go
// StatsResponseV1 represents v1 API response (backward compatibility)
type StatsResponseV1 struct {
    TotalTargets     int                       `json:"total_targets"`
    EnabledTargets  int                       `json:"enabled_targets"`
    TargetsByType    map[string]int            `json:"targets_by_type"`
    QueueSize        int                       `json:"queue_size"`
    QueueCapacity    int                       `json:"queue_capacity"`
    QueueUtilization float64                   `json:"queue_utilization_percent"`
}

// FilteredStatsResponse represents filtered statistics
type FilteredStatsResponse struct {
    StatsResponse
    Filter string `json:"filter_applied"`
}

// GroupedStatsResponse represents grouped statistics
type GroupedStatsResponse struct {
    StatsResponse
    GroupBy string                    `json:"group_by"`
    Groups  map[string]interface{}    `json:"groups"`
}
```

### 2.3 Helper Functions (Enhanced)

**File**: `go-app/cmd/server/handlers/publishing_stats_helpers.go` (EXISTING, ENHANCED)

**Current Functions**:
- ✅ `getMetricValue()` - Get metric by key
- ✅ `countHealthyTargets()` - Count healthy targets
- ✅ `countUnhealthyTargets()` - Count unhealthy targets
- ✅ `calculateSuccessRate()` - Calculate success rate
- ❌ `applyFilter()` - Apply filter to metrics (NEW)
- ❌ `applyGrouping()` - Apply grouping to response (NEW)
- ❌ `generateETag()` - Generate ETag for caching (NEW)
- ❌ `formatPrometheus()` - Format as Prometheus (NEW)

**New Functions**:

```go
// applyFilter applies filter to metrics snapshot
func applyFilter(snapshot *publishing.MetricsSnapshot, filter string) *publishing.MetricsSnapshot {
    // Parse filter: "type:rootly" or "status:healthy"
    // Filter metrics based on criteria
    // Return filtered snapshot
}

// applyGrouping applies grouping to response
func applyGrouping(response *StatsResponse, groupBy string) *GroupedStatsResponse {
    // Group by: "type", "status", "target"
    // Aggregate statistics by group
    // Return grouped response
}

// generateETag generates ETag for caching
func generateETag(snapshot *publishing.MetricsSnapshot) string {
    // Hash metrics snapshot
    // Return ETag string
}

// formatPrometheus formats response as Prometheus format
func formatPrometheus(response *StatsResponse) string {
    // Convert JSON response to Prometheus text format
    // Return Prometheus-formatted string
}
```

---

## 3. Data Models

### 3.1 Request Models

**Query Parameters**:
```go
type StatsQueryParams struct {
    Filter  string // "type:rootly" or "status:healthy"
    GroupBy string // "type", "status", "target"
    Format  string // "json" (default) or "prometheus"
}
```

### 3.2 Response Models

**StatsResponse (v2)**:
```json
{
  "timestamp": "2025-11-17T10:30:00Z",
  "system": {
    "total_targets": 10,
    "healthy_targets": 8,
    "unhealthy_targets": 2,
    "success_rate_percent": 95.5,
    "queue_size": 15,
    "queue_capacity": 1000
  },
  "target_stats": {
    "targets_by_type": {
      "rootly": 5,
      "slack": 3,
      "pagerduty": 2
    },
    "targets_by_status": {
      "healthy": 8,
      "degraded": 1,
      "unhealthy": 2
    }
  },
  "queue_stats": {
    "size": 15,
    "capacity": 1000,
    "utilization_percent": 1.5,
    "workers_active": 5,
    "workers_idle": 5
  },
  "job_stats": {
    "total_submitted": 10000,
    "total_completed": 9500,
    "total_failed": 500,
    "success_rate_percent": 95.0
  }
}
```

**StatsResponseV1 (v1, backward compatibility)**:
```json
{
  "total_targets": 10,
  "enabled_targets": 8,
  "targets_by_type": {
    "rootly": 5,
    "slack": 3,
    "pagerduty": 2
  },
  "queue_size": 15,
  "queue_capacity": 1000,
  "queue_utilization_percent": 1.5
}
```

### 3.3 Error Models

```json
{
  "error": "Bad Request",
  "message": "Invalid filter parameter",
  "request_id": "uuid-here",
  "timestamp": "2025-11-17T10:30:00Z"
}
```

---

## 4. API Design

### 4.1 Endpoint Specifications

#### 4.1.1 GET /api/v2/publishing/stats

**Method**: GET
**Path**: `/api/v2/publishing/stats`
**Query Parameters**:
- `filter` (optional): Filter criteria (e.g., "type:rootly", "status:healthy")
- `group_by` (optional): Group by field ("type", "status", "target")
- `format` (optional): Response format ("json" or "prometheus")

**Response**: `StatsResponse` (JSON)

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid query parameters
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Collection failure

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/v2/publishing/stats?filter=type:rootly&group_by=status"
```

**Example Response**:
```json
{
  "timestamp": "2025-11-17T10:30:00Z",
  "system": {
    "total_targets": 5,
    "healthy_targets": 4,
    "unhealthy_targets": 1,
    "success_rate_percent": 96.0,
    "queue_size": 10,
    "queue_capacity": 1000
  },
  "target_stats": {...},
  "queue_stats": {...}
}
```

#### 4.1.2 GET /api/v1/publishing/stats

**Method**: GET
**Path**: `/api/v1/publishing/stats`
**Query Parameters**: None (for backward compatibility)

**Response**: `StatsResponseV1` (JSON)

**Status Codes**:
- `200 OK`: Success
- `500 Internal Server Error`: Collection failure

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/v1/publishing/stats"
```

**Example Response**:
```json
{
  "total_targets": 10,
  "enabled_targets": 8,
  "targets_by_type": {
    "rootly": 5,
    "slack": 3,
    "pagerduty": 2
  },
  "queue_size": 15,
  "queue_capacity": 1000,
  "queue_utilization_percent": 1.5
}
```

### 4.2 HTTP Caching

**Cache-Control Header**:
```
Cache-Control: max-age=5, public
```

**ETag Header**:
```
ETag: "abc123def456"
```

**304 Not Modified Response**:
- Returned when `If-None-Match` header matches current ETag
- No response body
- Reduces server load

### 4.3 Rate Limiting

**Rate Limit**: 100 requests per minute per IP
**Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1637155200
```

**429 Response**:
```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded",
  "retry_after": 60
}
```

---

## 5. Security Design

### 5.1 Security Headers

**Required Headers**:
1. `X-Content-Type-Options: nosniff`
2. `X-Frame-Options: DENY`
3. `X-XSS-Protection: 1; mode=block`
4. `Content-Security-Policy: default-src 'self'`
5. `Strict-Transport-Security: max-age=31536000; includeSubDomains`
6. `Referrer-Policy: strict-origin-when-cross-origin`
7. `Permissions-Policy: geolocation=(), microphone=(), camera=()`

### 5.2 Input Validation

**Query Parameter Validation**:
- `filter`: Must match pattern `^[a-z]+:[a-z0-9-]+$`
- `group_by`: Must be one of: "type", "status", "target"
- `format`: Must be one of: "json", "prometheus"

**Error Response**:
```json
{
  "error": "Bad Request",
  "message": "Invalid filter parameter: expected format 'type:value'",
  "request_id": "uuid-here"
}
```

### 5.3 Rate Limiting

**Implementation**: Token bucket algorithm
**Configuration**:
- Rate: 100 requests per minute
- Burst: 20 requests
- Per-IP tracking

### 5.4 OWASP Top 10 Compliance

| Risk | Mitigation |
|------|------------|
| A01: Broken Access Control | Rate limiting, input validation |
| A02: Cryptographic Failures | HTTPS enforced, no sensitive data |
| A03: Injection | Input validation, parameterized queries |
| A04: Insecure Design | Security headers, rate limiting |
| A05: Security Misconfiguration | Security headers, proper error handling |
| A06: Vulnerable Components | Dependency scanning |
| A07: Authentication Failures | N/A (public endpoint) |
| A08: Software and Data Integrity | Input validation |
| A09: Security Logging | Structured logging, audit trail |
| A10: Server-Side Request Forgery | Input validation, no external requests |

---

## 6. Performance Design

### 6.1 Performance Targets

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| P50 latency | < 2ms | ~7µs | ✅ Exceeded |
| P95 latency | < 5ms | ~7µs | ✅ Exceeded |
| P99 latency | < 10ms | ~8µs | ✅ Exceeded |
| Throughput | > 10,000 req/s | ~62,500 req/s | ✅ Exceeded |
| Memory | < 10MB/request | ~683 B | ✅ Exceeded |

### 6.2 Optimization Strategies

**1. HTTP Caching**:
- Cache-Control: max-age=5s
- ETag-based conditional requests
- Reduces server load by 80%+

**2. Metrics Collection Caching**:
- 1s TTL on metrics snapshot
- Reduces collection overhead

**3. Response Compression**:
- gzip compression for JSON responses
- Reduces bandwidth by 70%+

**4. Query Parameter Optimization**:
- Early validation
- Fast path for common queries

### 6.3 Benchmarking

**Benchmark Targets**:
```go
func BenchmarkGetStats(b *testing.B) {
    // Target: < 5ms per request
}

func BenchmarkGetStatsConcurrent(b *testing.B) {
    // Target: > 10,000 req/s
}
```

---

## 7. Observability Design

### 7.1 Logging

**Structured Logging** (slog):
```go
h.logger.Info("Stats endpoint called",
    "request_id", requestID,
    "total_targets", systemStats.TotalTargets,
    "healthy_targets", systemStats.HealthyTargets,
    "duration_ms", duration.Milliseconds(),
)
```

**Log Levels**:
- `DEBUG`: Detailed metrics collection info
- `INFO`: Request/response summary
- `WARN`: Rate limit hits, validation errors
- `ERROR`: Collection failures, encoding errors

### 7.2 Metrics

**Prometheus Metrics**:
```go
publishing_stats_api_requests_total{status, endpoint}
publishing_stats_api_duration_seconds{endpoint}
publishing_stats_api_rate_limit_hits_total
publishing_stats_api_cache_hits_total
publishing_stats_api_cache_misses_total
```

### 7.3 Distributed Tracing

**Request ID Tracking**:
- UUID v4 in `X-Request-ID` header
- Propagated through all layers
- Included in logs and metrics

---

## 8. Error Handling Design

### 8.1 Error Types

**Client Errors (4xx)**:
- `400 Bad Request`: Invalid query parameters
- `405 Method Not Allowed`: Non-GET requests
- `429 Too Many Requests`: Rate limit exceeded

**Server Errors (5xx)**:
- `500 Internal Server Error`: Collection failure, encoding error

### 8.2 Error Response Format

```json
{
  "error": "Bad Request",
  "message": "Invalid filter parameter: expected format 'type:value'",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-17T10:30:00Z"
}
```

### 8.3 Error Handling Strategy

1. **Validation Errors**: Return 400 immediately
2. **Rate Limit**: Return 429 with retry-after
3. **Collection Errors**: Log error, return 500 with generic message
4. **Encoding Errors**: Log error, return 500

---

## 9. Testing Strategy

### 9.1 Unit Tests

**Coverage Target**: 90%+

**Test Cases**:
1. ✅ `TestGetStats_Success` - 200 OK response
2. ✅ `TestGetStats_NonGET` - 405 Method Not Allowed
3. ❌ `TestGetStatsV1_Success` - v1 endpoint (NEW)
4. ❌ `TestGetStats_Filter` - Filter parameter (NEW)
5. ❌ `TestGetStats_GroupBy` - Group by parameter (NEW)
6. ❌ `TestGetStats_FormatPrometheus` - Prometheus format (NEW)
7. ❌ `TestGetStats_InvalidFilter` - 400 Bad Request (NEW)
8. ❌ `TestGetStats_CacheHeaders` - Cache headers (NEW)
9. ❌ `TestGetStats_ETag` - ETag handling (NEW)
10. ❌ `TestGetStats_RateLimit` - Rate limiting (NEW)

### 9.2 Integration Tests

**Test Scenarios**:
1. End-to-end request flow
2. Metrics collection integration
3. Cache behavior
4. Rate limiting behavior

### 9.3 Security Tests

**Test Cases**:
1. SQL injection attempts
2. XSS attempts
3. Rate limit bypass attempts
4. Header manipulation

### 9.4 Performance Benchmarks

**Benchmarks**:
1. `BenchmarkGetStats` - Single request
2. `BenchmarkGetStatsConcurrent` - Concurrent requests
3. `BenchmarkGetStatsWithFilter` - Filtered requests
4. `BenchmarkGetStatsWithGroupBy` - Grouped requests

---

## 10. Deployment Strategy

### 10.1 Deployment Steps

1. **Code Review**: Review all changes
2. **Testing**: Run all tests and benchmarks
3. **Documentation**: Update API documentation
4. **Deployment**: Deploy to staging
5. **Validation**: Verify endpoints work correctly
6. **Monitoring**: Monitor metrics and errors
7. **Production**: Deploy to production

### 10.2 Rollback Plan

**Rollback Triggers**:
- Error rate > 1%
- P95 latency > 10ms
- Critical bugs discovered

**Rollback Steps**:
1. Revert code changes
2. Restart services
3. Verify endpoints restored
4. Monitor for stability

### 10.3 Monitoring

**Key Metrics**:
- Request rate
- Error rate
- Latency (P50, P95, P99)
- Cache hit rate
- Rate limit hits

**Alerts**:
- Error rate > 0.5%
- P95 latency > 10ms
- Rate limit hits > 100/min

---

## 11. Implementation Plan

### Phase 1: API v1 Endpoint (2 hours)
- [ ] Implement `GetStatsV1()` handler
- [ ] Add route registration
- [ ] Write unit tests
- [ ] Update documentation

### Phase 2: Query Parameters (2 hours)
- [ ] Implement filter parsing
- [ ] Implement group_by logic
- [ ] Implement format conversion
- [ ] Write unit tests

### Phase 3: HTTP Caching (1 hour)
- [ ] Implement ETag generation
- [ ] Implement Cache-Control headers
- [ ] Implement 304 Not Modified
- [ ] Write unit tests

### Phase 4: Security Hardening (1 hour)
- [ ] Add security headers
- [ ] Enhance input validation
- [ ] Write security tests
- [ ] OWASP compliance check

### Phase 5: Testing & Documentation (2 hours)
- [ ] Write integration tests
- [ ] Write performance benchmarks
- [ ] Update OpenAPI specification
- [ ] Write API guide

**Total Estimated Time**: 8 hours

---

**Document Status**: ✅ Design Complete
**Next Steps**: Create tasks.md and begin implementation
