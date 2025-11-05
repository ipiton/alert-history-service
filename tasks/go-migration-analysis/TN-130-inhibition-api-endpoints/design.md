# TN-130: Inhibition API Endpoints - Technical Design

**Version**: 1.0
**Date**: 2025-11-05
**Status**: READY FOR IMPLEMENTATION
**Target Quality**: 150% (Grade A+)

---

## Executive Summary

Реализовать **Alertmanager-совместимые REST API endpoints** для управления и мониторинга inhibition rules системы. Обеспечить полную интеграцию с существующими компонентами (TN-126/127/128/129) и достичь **enterprise-grade качества** с comprehensive testing, OpenAPI documentation и performance <5ms p99.

### Key Objectives

1. ✅ **Handler Already Exists**: `handlers/inhibition.go` (238 LOC) реализован
2. 🎯 **Integration Required**: Register endpoints in main.go
3. 🎯 **Testing**: 80%+ coverage, comprehensive test suite
4. 🎯 **Documentation**: OpenAPI 3.0 spec + module README
5. 🎯 **Performance**: <5ms p99 latency, 1000+ RPS throughput

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│               TN-130: API Endpoints Layer                 │
│                 (Alertmanager Compatible)                 │
└────────────────────┬─────────────────────────────────────┘
                     │
         ┌───────────┴──────────┐
         │  InhibitionHandler   │
         │   (238 LOC Ready)    │
         └───────────┬──────────┘
                     │
         ┌───────────┴────────────────────────┐
         │                                    │
    ┌────▼────┐  ┌──────▼──────┐  ┌─────▼──────┐
    │ Parser  │  │   Matcher   │  │   State    │
    │ (TN-126)│  │  (TN-127)   │  │  Manager   │
    │         │  │             │  │  (TN-129)  │
    └─────────┘  └─────────────┘  └────────────┘
         ▲              ▲                ▲
         │              │                │
         └──────────────┴────────────────┘
                   Cache (TN-128)
```

### Component Dependencies

| Component | Status | Quality | Coverage | Purpose |
|-----------|--------|---------|----------|---------|
| TN-126 Parser | ✅ DONE | 155% | 82.6% | YAML → Rules |
| TN-127 Matcher | ✅ DONE | 150% | 95% | Inhibition Logic |
| TN-128 Cache | ✅ DONE | 165% | 86.6% | Active Alerts |
| TN-129 State Manager | ✅ DONE | 150% | 65% | State Tracking |
| **TN-130 API** | 🎯 **TODO** | **150%** | **80%+** | **REST API** |

---

## API Endpoints Design

### 1. GET /api/v2/inhibition/rules

**Purpose**: List all loaded inhibition rules
**Compatibility**: Alertmanager v0.25+
**Performance Target**: <2ms p99

#### Request

```http
GET /api/v2/inhibition/rules HTTP/1.1
Host: alert-history:8080
Accept: application/json
```

#### Response (200 OK)

```json
{
  "rules": [
    {
      "name": "node-down-inhibits-instance-down",
      "source_match": {
        "alertname": "NodeDown",
        "severity": "critical"
      },
      "target_match": {
        "alertname": "InstanceDown"
      },
      "equal": ["node", "cluster"]
    }
  ],
  "count": 1
}
```

#### Implementation Notes

- ✅ Handler method exists: `InhibitionHandler.GetRules()`
- Returns config from `parser.GetConfig()`
- No database queries (config only)
- Ultra-fast response (<1ms typical)

---

### 2. GET /api/v2/inhibition/status

**Purpose**: Get all active inhibition relationships
**Compatibility**: Alertmanager extension
**Performance Target**: <5ms p99

#### Request

```http
GET /api/v2/inhibition/status HTTP/1.1
Host: alert-history:8080
Accept: application/json
```

#### Optional Query Parameters

- `?fingerprint=abc123` - Filter by target fingerprint
- `?rule_name=NodeDown` - Filter by rule name
- `?source_fingerprint=def456` - Filter by source

#### Response (200 OK)

```json
{
  "active": [
    {
      "target_fingerprint": "abc123",
      "source_fingerprint": "def456",
      "rule_name": "NodeDown_inhibits_InstanceDown",
      "inhibited_at": "2025-11-05T10:00:00Z",
      "expires_at": "2025-11-05T10:05:00Z"
    }
  ],
  "count": 1
}
```

#### Implementation Notes

- ✅ Handler method exists: `InhibitionHandler.GetStatus()`
- Calls `stateManager.GetActiveInhibitions(ctx)`
- 🎯 **Enhancement**: Add query parameter filtering (150% quality)
- May query Redis (L2 cache) if needed

---

### 3. POST /api/v2/inhibition/check

**Purpose**: Check if an alert would be inhibited
**Compatibility**: Alertmanager extension
**Performance Target**: <3ms p99

#### Request

```http
POST /api/v2/inhibition/check HTTP/1.1
Host: alert-history:8080
Content-Type: application/json

{
  "alert": {
    "labels": {
      "alertname": "InstanceDown",
      "node": "node1",
      "cluster": "prod"
    },
    "annotations": {
      "summary": "Instance down on node1"
    }
  }
}
```

#### Response (200 OK) - Inhibited

```json
{
  "alert": {
    "labels": {
      "alertname": "InstanceDown",
      "node": "node1"
    }
  },
  "inhibited": true,
  "inhibited_by": {
    "labels": {
      "alertname": "NodeDown",
      "node": "node1",
      "severity": "critical"
    }
  },
  "rule": {
    "name": "node-down-inhibits-instance-down",
    "source_match": {...},
    "target_match": {...},
    "equal": ["node", "cluster"]
  },
  "latency_ms": 2
}
```

#### Response (200 OK) - Not Inhibited

```json
{
  "alert": {...},
  "inhibited": false,
  "latency_ms": 1
}
```

#### Implementation Notes

- ✅ Handler method exists: `InhibitionHandler.CheckAlert()`
- Calls `matcher.ShouldInhibit(ctx, alert)`
- Records metrics: `InhibitionChecksTotal`, `InhibitionDurationSeconds`
- Includes latency in response (observability)

---

### 4. Error Responses

All endpoints return standard error format:

```json
{
  "error": "Failed to retrieve inhibition status",
  "code": 500
}
```

#### HTTP Status Codes

| Code | Meaning | When |
|------|---------|------|
| 200 | Success | Request processed successfully |
| 400 | Bad Request | Invalid JSON, missing required fields |
| 500 | Internal Error | Database failure, matcher error |
| 503 | Service Unavailable | Startup, graceful shutdown |

---

## Main.go Integration

### Initialization Sequence

```go
// In main.go (after TN-129 State Manager initialization)

// Step 1: Initialize InhibitionParser (if not already done)
inhibitionParser, err := inhibition.NewParser(
    "config/inhibition.yaml",
    appLogger,
)
if err != nil {
    slog.Error("Failed to load inhibition config", "error", err)
    os.Exit(1)
}
config := inhibitionParser.GetConfig()
slog.Info("Loaded inhibition rules", "count", len(config.Rules))

// Step 2: Initialize InhibitionMatcher
inhibitionMatcher := inhibition.NewMatcher(
    activeAlertCache,  // TN-128
    config.Rules,
    appLogger,
)

// Step 3: Initialize InhibitionStateManager
inhibitionStateManager := inhibition.NewDefaultStateManager(
    redisCache,                // Redis for persistence
    "inhibition:state:",       // Key prefix
    5 * time.Minute,           // State TTL
    businessMetrics,           // Metrics
    appLogger,
)
inhibitionStateManager.StartCleanupWorker(1 * time.Minute)
defer inhibitionStateManager.StopCleanupWorker()

// Step 4: Create InhibitionHandler
inhibitionHandler := handlers.NewInhibitionHandler(
    inhibitionParser,
    inhibitionMatcher,
    inhibitionStateManager,
    businessMetrics,
    appLogger,
)

// Step 5: Register routes
mux.HandleFunc("GET /api/v2/inhibition/rules", inhibitionHandler.GetRules)
mux.HandleFunc("GET /api/v2/inhibition/status", inhibitionHandler.GetStatus)
mux.HandleFunc("POST /api/v2/inhibition/check", inhibitionHandler.CheckAlert)

slog.Info("Inhibition API endpoints registered")
```

### Graceful Shutdown

```go
// Add to shutdown sequence
inhibitionStateManager.StopCleanupWorker()
slog.Info("Inhibition state manager stopped")
```

---

## Performance Requirements (150% Quality)

### Targets

| Endpoint | Target p99 | Target p50 | Target RPS |
|----------|-----------|-----------|------------|
| GET /rules | <2ms | <500µs | 5000+ |
| GET /status | <5ms | <2ms | 2000+ |
| POST /check | <3ms | <1ms | 3000+ |

### Optimizations

1. **GET /rules**: No DB queries, config only (ultra-fast)
2. **GET /status**: In-memory cache (L1), Redis fallback (L2)
3. **POST /check**: Matcher already optimized (16.958µs per check)

### Metrics to Monitor

- `http_request_duration_seconds{handler="inhibition_rules"}`
- `http_request_duration_seconds{handler="inhibition_status"}`
- `http_request_duration_seconds{handler="inhibition_check"}`
- `alert_history_business_inhibition_checks_total{result="inhibited|allowed"}`

---

## Testing Strategy (80%+ Coverage)

### Test Categories

1. **Unit Tests** (50% of coverage)
   - Handler method tests
   - Request/response validation
   - Error handling
   - Metrics recording

2. **Integration Tests** (30%)
   - Full request → response flow
   - Parser + Matcher + StateManager integration
   - Redis persistence (optional)

3. **Edge Cases** (20%)
   - Invalid JSON
   - Missing alert fields
   - Nil pointer safety
   - Context cancellation
   - Concurrent requests

### Test Structure

```
go-app/cmd/server/handlers/
├── inhibition.go         (238 LOC) ✅ EXISTS
└── inhibition_test.go    (NEW, target: 600+ LOC)
```

### Test Examples

#### Test 1: GET /rules Success

```go
func TestInhibitionHandler_GetRules_Success(t *testing.T) {
    // Setup mock parser with 3 rules
    // Create handler
    // Make GET request
    // Assert: 200 OK, count=3, rules match config
}
```

#### Test 2: GET /status Empty

```go
func TestInhibitionHandler_GetStatus_NoActiveInhibitions(t *testing.T) {
    // Setup empty state manager
    // Make GET request
    // Assert: 200 OK, count=0, active=[]
}
```

#### Test 3: POST /check - Inhibited

```go
func TestInhibitionHandler_CheckAlert_Inhibited(t *testing.T) {
    // Setup matcher with NodeDown → InstanceDown rule
    // Add firing NodeDown alert to cache
    // POST InstanceDown alert
    // Assert: 200 OK, inhibited=true, rule name matches
}
```

#### Test 4: POST /check - Invalid JSON

```go
func TestInhibitionHandler_CheckAlert_InvalidJSON(t *testing.T) {
    // POST malformed JSON
    // Assert: 400 Bad Request, error message
}
```

#### Test 5: Metrics Recording

```go
func TestInhibitionHandler_MetricsRecorded(t *testing.T) {
    // Make multiple requests
    // Assert: Prometheus metrics incremented correctly
}
```

---

## OpenAPI 3.0 Specification

### File Location

```
docs/openapi-inhibition.yaml
```

### Key Sections

1. **Paths**:
   - `/api/v2/inhibition/rules`
   - `/api/v2/inhibition/status`
   - `/api/v2/inhibition/check`

2. **Schemas**:
   - `InhibitionRule`
   - `InhibitionState`
   - `InhibitionCheckRequest`
   - `InhibitionCheckResponse`
   - `ErrorResponse`

3. **Examples**: Include realistic request/response examples

4. **Compatibility**: Mark Alertmanager extensions clearly

---

## AlertProcessor Integration

### Current Flow

```
Webhook → Parser → Validator → AlertProcessor → Publishing
```

### Enhanced Flow (with Inhibition)

```
Webhook → Parser → Validator → AlertProcessor
                                      ↓
                              [Check Inhibition]
                                      ↓
                            ┌─────────┴──────────┐
                            ↓                    ↓
                       [Inhibited]          [Allowed]
                            ↓                    ↓
                      [Skip Publishing]    [Publishing]
                            ↓                    ↓
                       Record Metrics      Normal Flow
```

### Integration Code

```go
// In AlertProcessor.ProcessAlert()

// Check if alert should be inhibited
result, err := ap.inhibitionMatcher.ShouldInhibit(ctx, alert)
if err != nil {
    ap.logger.Warn("Inhibition check failed", "error", err)
    // Continue processing (fail-open)
} else if result.Matched {
    ap.logger.Info("Alert inhibited",
        "fingerprint", alert.Fingerprint,
        "rule", result.Rule.Name,
        "inhibited_by", result.InhibitedBy.Fingerprint,
    )

    // Record inhibition state
    state := &inhibition.InhibitionState{
        TargetFingerprint: alert.Fingerprint,
        SourceFingerprint: result.InhibitedBy.Fingerprint,
        RuleName:          result.Rule.Name,
        InhibitedAt:       time.Now(),
    }
    _ = ap.stateManager.RecordInhibition(ctx, state)

    // Record metrics
    ap.metrics.InhibitionChecksTotal.WithLabelValues("inhibited").Inc()

    // Skip publishing
    return nil
}

ap.metrics.InhibitionChecksTotal.WithLabelValues("allowed").Inc()

// Continue with normal processing...
```

---

## Documentation Requirements

### 1. Module README

**File**: `go-app/cmd/server/handlers/INHIBITION_API.md`

**Sections**:
- Overview
- API Endpoints (with examples)
- Integration Guide
- Performance Characteristics
- Troubleshooting

**Length**: 400+ lines

### 2. OpenAPI Spec

**File**: `docs/openapi-inhibition.yaml`

**Requirements**:
- OpenAPI 3.0.3
- Complete schemas
- Request/response examples
- Error responses
- Authentication (if applicable)

**Length**: 300+ lines

### 3. Example Usage

**File**: `docs/examples/inhibition-api-examples.md`

**Content**:
- curl examples
- HTTP client examples (Go, Python)
- Common use cases
- Troubleshooting scenarios

---

## Quality Criteria (150% Achievement)

### Base Requirements (100%)

- [x] 3 endpoints functional
- [ ] main.go integration
- [ ] Basic tests (60%+ coverage)
- [ ] OpenAPI spec

### Enhanced Requirements (150%)

- [ ] **80%+ test coverage** (vs 60% base)
- [ ] **Comprehensive error handling** (validation, nil checks)
- [ ] **Performance benchmarks** (<5ms p99)
- [ ] **Query parameter filtering** for GET /status
- [ ] **Detailed module documentation** (400+ lines)
- [ ] **AlertProcessor integration** (fail-safe)
- [ ] **Metrics recording** (all operations)
- [ ] **Context cancellation** support
- [ ] **Graceful degradation** (Redis failures)

### Quality Metrics

| Metric | Base (100%) | Enhanced (150%) | Achievement |
|--------|-------------|-----------------|-------------|
| Test Coverage | 60% | **80%+** | TBD |
| Tests Count | 10 | **20+** | TBD |
| Benchmarks | 0 | **3+** | TBD |
| Documentation | 200 lines | **700+ lines** | TBD |
| Performance p99 | <10ms | **<5ms** | TBD |
| Error Handling | Basic | **Comprehensive** | TBD |

---

## Success Criteria

### Functional

- ✅ All 3 endpoints return correct responses
- ✅ Alertmanager compatibility verified
- ✅ Integration with TN-126/127/128/129 working
- ✅ Metrics recording functional
- ✅ Graceful error handling

### Non-Functional

- ✅ **Performance**: <5ms p99 latency
- ✅ **Throughput**: 1000+ RPS per endpoint
- ✅ **Test Coverage**: 80%+ (20 tests)
- ✅ **Documentation**: 700+ lines
- ✅ **Zero Breaking Changes**

### Quality Grade

| Grade | Criteria |
|-------|----------|
| **A+ (150%)** | All enhanced requirements met, performance 2x better than targets |
| **A (130%)** | Most enhanced requirements, performance 1.5x better |
| **B (110%)** | All base + some enhanced requirements |
| **C (100%)** | All base requirements only |

**Target**: **A+ (150%)**

---

## Risks & Mitigations

### Risk 1: Handler Already Exists (Scope Creep)

**Risk**: Handler code may need refactoring
**Mitigation**: Review handler code, minimal changes, focus on integration

### Risk 2: Performance Degradation

**Risk**: API calls slow down alert processing
**Mitigation**: Keep matcher ultra-fast (<1ms), use async state recording

### Risk 3: Test Complexity

**Risk**: Testing full integration may be complex
**Mitigation**: Use mocks for external dependencies, incremental testing

### Risk 4: OpenAPI Spec Maintenance

**Risk**: Spec diverges from implementation
**Mitigation**: Generate spec from code annotations (future), manual review for now

---

## Timeline Estimate (150% Quality)

| Phase | Task | Time | Dependencies |
|-------|------|------|--------------|
| 1 | Documentation (this file + tasks.md) | 30m | None |
| 2 | Create branch | 5m | Phase 1 |
| 3 | Main.go integration | 30m | Phase 2 |
| 4 | Comprehensive tests (20+ tests) | 2h | Phase 3 |
| 5 | OpenAPI 3.0 spec | 45m | Phase 4 |
| 6 | AlertProcessor integration | 45m | Phase 3 |
| 7 | Performance benchmarks | 30m | Phase 4 |
| 8 | Module documentation | 1h | Phase 5-7 |
| 9 | Final validation & report | 30m | Phase 8 |

**Total Estimated Time**: **6.5 hours** (vs 2h base)

**Multiplier**: **3.25x** (150% quality requires 3x+ effort)

---

## Appendix A: Handler Code Review

**File**: `go-app/cmd/server/handlers/inhibition.go`
**Status**: ✅ **PRODUCTION-READY**
**LOC**: 238

### Strengths

- ✅ Clean interface (InhibitionHandler)
- ✅ All 3 methods implemented
- ✅ Proper error handling
- ✅ Metrics recording
- ✅ Structured logging
- ✅ JSON encoding/decoding

### Potential Enhancements (150%)

1. **Query Parameters**: Add filtering for GET /status
2. **Validation**: More comprehensive request validation
3. **Context Timeout**: Set explicit timeout for long operations
4. **Rate Limiting**: Consider rate limiting for public endpoints

### No Breaking Changes Required

Handler code is **production-ready** and requires **zero changes** for MVP.

---

## Appendix B: Module Structure

```
go-app/
├── cmd/server/
│   ├── handlers/
│   │   ├── inhibition.go          (238 LOC) ✅ EXISTS
│   │   └── inhibition_test.go     (600+ LOC) 🎯 NEW
│   └── main.go                    (UPDATE: +40 lines)
├── internal/infrastructure/inhibition/
│   ├── parser.go                  ✅ TN-126
│   ├── matcher.go                 ✅ TN-127
│   ├── cache.go                   ✅ TN-128
│   └── state_manager.go           ✅ TN-129
└── docs/
    ├── openapi-inhibition.yaml    🎯 NEW (300+ lines)
    └── examples/
        └── inhibition-api-examples.md  🎯 NEW (200+ lines)
```

---

## Appendix C: Dependencies Graph

```
TN-130 (API)
    │
    ├─► TN-126 (Parser) ✅
    │      └─► config/inhibition.yaml ✅
    │
    ├─► TN-127 (Matcher) ✅
    │      └─► TN-128 (Cache) ✅
    │             └─► Redis (L2)
    │
    └─► TN-129 (State Manager) ✅
           └─► Redis (Persistence)
```

**All dependencies RESOLVED** ✅

---

**Document Version**: 1.0
**Last Updated**: 2025-11-05
**Author**: AlertHistory Team
**Status**: APPROVED FOR IMPLEMENTATION
