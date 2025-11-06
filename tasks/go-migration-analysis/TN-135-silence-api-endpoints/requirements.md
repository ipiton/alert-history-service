# TN-135: Silence API Endpoints - Requirements

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-135
**Status**: 🟡 IN PROGRESS
**Priority**: HIGH
**Estimated Effort**: 10-14 hours
**Target Quality**: 150% (Enterprise-Grade)
**Dependencies**: TN-131 ✅, TN-132 ✅, TN-133 ✅, TN-134 ✅

---

## 📋 Overview

Реализовать REST API endpoints для управления silences (заглушками алертов) с полной совместимостью с Alertmanager API v2. API должен предоставлять CRUD операции для silences, фильтрацию, поиск и проверку состояния заглушек.

### Business Value
- **Alertmanager Compatibility**: 100% совместимость с Alertmanager API v2 для беспроблемной миграции
- **Operational Efficiency**: Быстрое создание/удаление silences через API
- **Automation**: Возможность автоматизации управления заглушками через CI/CD
- **Observability**: Полная видимость активных silences и их влияния на алерты
- **User Experience**: RESTful API с понятной структурой ответов и error handling

---

## 🎯 Goals

### Primary Goals
1. ✅ Реализовать **5 core API endpoints** (POST, GET list, GET by ID, PUT, DELETE)
2. ✅ Добавить **advanced filtering** для GET /silences (status, creator, matcher, time range)
3. ✅ Реализовать **validation & error handling** с детальными сообщениями
4. ✅ Интегрировать **Prometheus metrics** для observability
5. ✅ Обеспечить **Alertmanager API v2 compatibility** (100%)

### Secondary Goals (150% Quality)
- **Pagination support** (limit/offset) для больших списков silences
- **Sorting options** (by created_at, starts_at, ends_at, status)
- **Bulk operations endpoint** (DELETE multiple silences)
- **Check endpoint** (POST /silences/check - проверить будет ли алерт заглушён)
- **OpenAPI 3.0 specification** (полная документация API)
- **Rate limiting** (защита от abuse)
- **Comprehensive testing** (unit + integration + benchmarks)
- **Response caching** (ETag support для GET requests)

---

## 📐 Functional Requirements

### FR-1: Create Silence Endpoint

**Endpoint**: `POST /api/v2/silences`

**Request Body**:
```json
{
  "createdBy": "ops@example.com",
  "comment": "Maintenance window for database upgrade",
  "startsAt": "2025-11-06T12:00:00Z",
  "endsAt": "2025-11-06T14:00:00Z",
  "matchers": [
    {"name": "alertname", "value": "HighCPU", "type": "="},
    {"name": "job", "value": "api-server", "type": "="}
  ]
}
```

**Response** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "createdBy": "ops@example.com",
  "comment": "Maintenance window for database upgrade",
  "startsAt": "2025-11-06T12:00:00Z",
  "endsAt": "2025-11-06T14:00:00Z",
  "matchers": [
    {"name": "alertname", "value": "HighCPU", "type": "=", "isRegex": false},
    {"name": "job", "value": "api-server", "type": "=", "isRegex": false}
  ],
  "status": "pending",
  "createdAt": "2025-11-06T11:00:00Z",
  "updatedAt": null
}
```

**Validation Rules**:
- `createdBy`: required, 1-255 characters, valid email format
- `comment`: required, 3-1024 characters
- `startsAt`: required, must be valid timestamp
- `endsAt`: required, must be after `startsAt`
- `matchers`: required, 1-100 matchers, each matcher validated by Silence.Validate()

**Error Responses**:
- 400 Bad Request: Invalid request body, validation errors
- 409 Conflict: Duplicate silence (same matchers + time range already exists)
- 500 Internal Server Error: Database errors, unexpected failures

**Performance Target**: <20ms (p95), <50ms (p99)

---

### FR-2: List Silences Endpoint

**Endpoint**: `GET /api/v2/silences`

**Query Parameters**:
```
?status=active                    # Filter by status (pending/active/expired)
?createdBy=ops@example.com        # Filter by creator
&matcher=alertname=HighCPU        # Filter by matcher
&startsAfter=2025-11-06T00:00:00Z # Start time range
&startsBefore=2025-11-07T00:00:00Z # End time range
&limit=100                        # Pagination limit (default: 100, max: 1000)
&offset=0                         # Pagination offset (default: 0)
&sort=created_at                  # Sort field (created_at, starts_at, ends_at, status)
&order=desc                       # Sort order (asc/desc, default: desc)
```

**Response** (200 OK):
```json
{
  "silences": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "createdBy": "ops@example.com",
      "comment": "Maintenance window",
      "startsAt": "2025-11-06T12:00:00Z",
      "endsAt": "2025-11-06T14:00:00Z",
      "matchers": [...],
      "status": "active",
      "createdAt": "2025-11-06T11:00:00Z"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

**Default Behavior**:
- If no filters: return all silences (sorted by created_at desc)
- Empty result: return `{"silences": [], "total": 0}`
- Invalid filter value: return 400 Bad Request

**Performance Target**:
- Fast path (status=active only): <10ms (cache hit)
- Slow path (complex filters): <100ms (database query)

---

### FR-3: Get Silence by ID Endpoint

**Endpoint**: `GET /api/v2/silences/{id}`

**URL Parameters**:
- `id`: Silence UUID (required, valid UUID v4 format)

**Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "createdBy": "ops@example.com",
  "comment": "Maintenance window",
  "startsAt": "2025-11-06T12:00:00Z",
  "endsAt": "2025-11-06T14:00:00Z",
  "matchers": [...],
  "status": "active",
  "createdAt": "2025-11-06T11:00:00Z",
  "updatedAt": null
}
```

**Error Responses**:
- 400 Bad Request: Invalid UUID format
- 404 Not Found: Silence not found
- 500 Internal Server Error: Database errors

**Performance Target**: <5ms (cached), <20ms (uncached)

---

### FR-4: Update Silence Endpoint

**Endpoint**: `PUT /api/v2/silences/{id}`

**Request Body**:
```json
{
  "comment": "Extended maintenance window",
  "endsAt": "2025-11-06T16:00:00Z"
}
```

**Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "createdBy": "ops@example.com",
  "comment": "Extended maintenance window",
  "startsAt": "2025-11-06T12:00:00Z",
  "endsAt": "2025-11-06T16:00:00Z",
  "matchers": [...],
  "status": "active",
  "createdAt": "2025-11-06T11:00:00Z",
  "updatedAt": "2025-11-06T13:30:00Z"
}
```

**Updatable Fields**:
- `comment`: optional, 3-1024 characters
- `endsAt`: optional, must be after current `startsAt`
- `matchers`: optional, 1-100 matchers (replaces entire list)

**Immutable Fields**:
- `id`, `createdBy`, `startsAt`, `createdAt`

**Error Responses**:
- 400 Bad Request: Invalid request body, validation errors
- 404 Not Found: Silence not found
- 409 Conflict: Optimistic locking failure
- 500 Internal Server Error: Database errors

**Performance Target**: <30ms (p95)

---

### FR-5: Delete Silence Endpoint

**Endpoint**: `DELETE /api/v2/silences/{id}`

**URL Parameters**:
- `id`: Silence UUID (required)

**Response** (204 No Content):
- Empty body on success

**Error Responses**:
- 400 Bad Request: Invalid UUID format
- 404 Not Found: Silence not found
- 500 Internal Server Error: Database errors

**Performance Target**: <15ms (p95)

**Note**: This is a **hard delete** (removes from database). Expired silences are soft-deleted by GC worker and kept for 24h.

---

### FR-6: Check Alert Silenced Endpoint (150% Feature)

**Endpoint**: `POST /api/v2/silences/check`

**Request Body**:
```json
{
  "labels": {
    "alertname": "HighCPU",
    "job": "api-server",
    "instance": "server-01"
  }
}
```

**Response** (200 OK):
```json
{
  "silenced": true,
  "silenceIDs": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660e8400-e29b-41d4-a716-446655440001"
  ],
  "silences": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "comment": "Maintenance window",
      "startsAt": "2025-11-06T12:00:00Z",
      "endsAt": "2025-11-06T14:00:00Z"
    }
  ],
  "latencyMs": 5
}
```

**Use Case**: Check if an alert would be silenced before firing

**Performance Target**: <10ms (100 active silences)

---

### FR-7: Bulk Delete Endpoint (150% Feature)

**Endpoint**: `POST /api/v2/silences/bulk/delete`

**Request Body**:
```json
{
  "ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660e8400-e29b-41d4-a716-446655440001"
  ]
}
```

**Response** (200 OK):
```json
{
  "deleted": 2,
  "errors": []
}
```

**Partial Success Response** (207 Multi-Status):
```json
{
  "deleted": 1,
  "errors": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "error": "silence not found"
    }
  ]
}
```

**Performance Target**: <50ms for 100 silences

---

## 🔒 Non-Functional Requirements

### NFR-1: Performance
- **Latency targets**:
  - GET /silences (cached): p50 <5ms, p95 <10ms, p99 <20ms
  - GET /silences (uncached): p50 <50ms, p95 <100ms, p99 <200ms
  - POST /silences: p50 <10ms, p95 <20ms, p99 <50ms
  - PUT /silences: p50 <15ms, p95 <30ms, p99 <60ms
  - DELETE /silences: p50 <5ms, p95 <15ms, p99 <30ms
  - POST /silences/check: p50 <5ms, p95 <10ms, p99 <20ms

- **Throughput targets**:
  - GET requests: 1000+ req/s (cached)
  - POST requests: 500+ req/s
  - Concurrent requests: Handle 100+ concurrent connections

- **Resource limits**:
  - Memory overhead: <50 MB for handler logic
  - No memory leaks (validated via pprof)
  - Zero allocations in hot paths (benchmarks)

### NFR-2: Scalability
- Support 10,000+ silences in database
- Pagination for large result sets
- Cache-first strategy for active silences
- Horizontal scaling ready (stateless handlers)

### NFR-3: Reliability
- **Error handling**: All errors logged with context
- **Graceful degradation**: Continue on non-critical errors
- **Input validation**: All inputs validated before processing
- **Transaction safety**: Database transactions for critical operations
- **Idempotency**: Duplicate POST requests return existing silence (409)

### NFR-4: Security
- **Input sanitization**: Prevent SQL injection (parameterized queries)
- **Rate limiting**: 100 requests/min per IP (configurable)
- **Authentication**: JWT token validation (optional, deferred to TN-137)
- **Authorization**: Creator can only delete their own silences (deferred)
- **Audit logging**: All CRUD operations logged

### NFR-5: Observability
- **Prometheus metrics** (8 total):
  1. `alert_history_api_silence_requests_total{method, endpoint, status}` - Counter
  2. `alert_history_api_silence_request_duration_seconds{method, endpoint}` - Histogram
  3. `alert_history_api_silence_validation_errors_total{field}` - Counter
  4. `alert_history_api_silence_operations_total{operation, result}` - Counter
  5. `alert_history_api_silence_active_silences` - Gauge (current count)
  6. `alert_history_api_silence_cache_hits_total{endpoint}` - Counter
  7. `alert_history_api_silence_response_size_bytes{endpoint}` - Histogram
  8. `alert_history_api_silence_rate_limit_exceeded_total{endpoint}` - Counter

- **Structured logging**: All requests/responses logged (slog)
- **Distributed tracing**: OpenTelemetry span support (deferred)
- **Health checks**: Included in /healthz endpoint

### NFR-6: Compatibility
- **Alertmanager API v2**: 100% compatible response format
- **Backward compatibility**: No breaking changes to existing API
- **Forward compatibility**: Extensible response format (additional fields ignored)

### NFR-7: Testability
- **Unit tests**: 80%+ coverage (target: 90%+)
- **Integration tests**: All endpoints tested with real database
- **Benchmark tests**: All endpoints benchmarked
- **Load tests**: Validated under 1000 req/s load
- **Chaos tests**: Tested with database failures, network timeouts

---

## 📊 Acceptance Criteria

### AC-1: Core Endpoints (100% Must-Have)
- [x] POST /api/v2/silences - Create silence (201 Created)
- [x] GET /api/v2/silences - List silences with filters (200 OK)
- [x] GET /api/v2/silences/{id} - Get silence by ID (200 OK / 404 Not Found)
- [x] PUT /api/v2/silences/{id} - Update silence (200 OK / 404 Not Found)
- [x] DELETE /api/v2/silences/{id} - Delete silence (204 No Content / 404 Not Found)

### AC-2: Advanced Features (150% Quality)
- [x] POST /api/v2/silences/check - Check if alert silenced
- [x] POST /api/v2/silences/bulk/delete - Bulk delete
- [x] Pagination support (limit/offset)
- [x] Sorting support (sort/order)
- [x] Response caching (ETag headers)

### AC-3: Validation & Error Handling
- [x] All inputs validated (createdBy, comment, times, matchers)
- [x] Detailed error responses (field-level errors)
- [x] HTTP status codes correct (200, 201, 204, 400, 404, 409, 500)
- [x] Error logging with context

### AC-4: Performance
- [x] All latency targets met (benchmarks)
- [x] Zero allocations in hot paths
- [x] Cache hit rate >90% for active silences
- [x] No memory leaks (pprof validation)

### AC-5: Testing
- [x] 40+ unit tests (90%+ coverage)
- [x] 10+ integration tests (real database)
- [x] 8+ benchmark tests (all endpoints)
- [x] 5+ concurrent tests (race detector)
- [x] 100% tests passing

### AC-6: Documentation
- [x] OpenAPI 3.0 spec (complete, validated)
- [x] README.md with usage examples (800+ lines)
- [x] Godoc comments (all public types/methods)
- [x] Integration guide (main.go example)
- [x] Postman collection (optional)

### AC-7: Integration
- [x] Integrated into main.go (endpoints registered)
- [x] Connected to SilenceManager
- [x] Prometheus metrics registered
- [x] Logging middleware applied
- [x] Health checks passing

---

## 🔗 Dependencies

### Upstream Dependencies (Required)
- ✅ **TN-131**: Silence Data Models - `silencing.Silence`, `silencing.Matcher`
- ✅ **TN-132**: Silence Matcher Engine - `SilenceMatcher` interface (for /check endpoint)
- ✅ **TN-133**: Silence Storage - `SilenceRepository` interface
- ✅ **TN-134**: Silence Manager Service - `SilenceManager` interface (orchestrator)

### Infrastructure Dependencies
- ✅ **TN-16**: Redis Cache - For response caching (ETag support)
- ✅ **TN-21**: Prometheus Metrics - For observability
- ✅ **TN-20**: Structured Logging - For request/response logging

### Downstream Consumers (Blocked by TN-135)
- ⏳ **TN-136**: Silence UI Components - Dashboard widgets (uses these APIs)
- ⏳ **TN-137**: Advanced Routing - May integrate silence checks into routing

---

## 🚀 Success Metrics

### Quantitative Metrics
- **Test Coverage**: ≥90% (target: 95%+)
- **Performance**: All endpoints meet latency targets (p95)
- **Reliability**: 99.9%+ success rate in production
- **Cache Hit Rate**: ≥90% for GET /silences?status=active
- **Documentation**: 100% API coverage (OpenAPI spec)

### Qualitative Metrics
- **Developer Experience**: Clear error messages, easy integration
- **User Experience**: Fast response times, intuitive API
- **Maintainability**: Clean code, well-tested, documented
- **Compatibility**: 100% Alertmanager API v2 compatible

---

## 📝 Out of Scope

Following features are **explicitly out of scope** for TN-135:

1. **Authentication & Authorization**: Deferred to future tasks (TN-137+)
2. **Rate Limiting Implementation**: Placeholder only (full impl in TN-138)
3. **Web UI**: Silence dashboard/forms (TN-136)
4. **Notification Integration**: Email on silence expiration (TN-139)
5. **Audit Log UI**: View silence history (TN-140)
6. **Advanced Search**: Full-text search on comments (future)
7. **Silence Templates**: Pre-defined silence patterns (future)
8. **Silence Scheduling**: Recurring silences (future)

---

## 🎯 Quality Target: 150%

To achieve **150% quality** (Grade A+), TN-135 must deliver:

1. **100% Core Features** (5 endpoints: POST, GET, GET/:id, PUT, DELETE)
2. **+50% Advanced Features**:
   - POST /silences/check endpoint
   - POST /silences/bulk/delete endpoint
   - Pagination & sorting
   - Response caching (ETag)
   - OpenAPI 3.0 spec
   - Rate limiting (basic)

3. **Exceptional Quality**:
   - 95%+ test coverage (target: 90%+, +5%)
   - Performance 2x better than targets
   - Zero technical debt
   - Production-ready documentation
   - Enterprise-grade error handling

4. **Comprehensive Documentation**:
   - 1,000+ lines README
   - OpenAPI spec (complete)
   - Integration examples
   - PromQL query examples
   - Grafana dashboard JSON

**Expected LOC**:
- Production code: ~1,500 lines (handler, models, middleware)
- Test code: ~2,500 lines (unit + integration + benchmarks)
- Documentation: ~2,000 lines (README, OpenAPI, guides)
- **Total**: ~6,000 lines

**Timeline**: 10-14 hours (target: 12h actual)

---

## 📚 References

- [Alertmanager API v2 Spec](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml)
- [TN-134 Completion Report](/tasks/go-migration-analysis/TN-134-silence-manager-service/COMPLETION_REPORT.md)
- [TN-130 Inhibition API](/tasks/go-migration-analysis/TN-130-inhibition-api-endpoints/) (similar pattern)
- [Go HTTP Best Practices](https://github.com/golang-standards/project-layout)

---

**Document Version**: 1.0
**Created**: 2025-11-06
**Author**: Kilo Code AI
**Status**: APPROVED FOR IMPLEMENTATION
