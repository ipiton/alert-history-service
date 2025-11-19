# TN-148: Prometheus-Compatible Response Format - Comprehensive Analysis

**Date**: 2025-11-19
**Task**: GET /api/v2/alerts endpoint (Alertmanager-compatible)
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Dependencies**: TN-146 ✅, TN-147 ✅
**Status**: 📋 Analysis Phase

---

## 🎯 Executive Summary

**TN-148** implements a **GET /api/v2/alerts** endpoint that returns alerts in Prometheus/Alertmanager-compatible format. This completes the bidirectional Prometheus compatibility:

- ✅ **TN-147**: POST /api/v2/alerts (receive alerts from Prometheus)
- 🎯 **TN-148**: GET /api/v2/alerts (query alerts for Prometheus/Grafana)

### Strategic Value

1. **Full Prometheus Ecosystem Compatibility**: Enables using Alert History Service as backend for Prometheus/Grafana
2. **Query & Visualization**: Allows Grafana dashboards to query historical alerts
3. **API Completeness**: Completes Phase 1 Alert Ingestion (100%)
4. **Alertmanager Replacement**: Can be used as drop-in replacement for Alertmanager queries

---

## 📋 Requirements Analysis

### What is TN-148?

Based on codebase search and PHASE1_AUDIT:

```go
// TN-148: GET /api/v2/alerts - Query alerts in Alertmanager format
type AlertmanagerResponse struct {
    Status string `json:"status"`  // "success" or "error"
    Data   struct {
        Alerts []AlertmanagerAlert `json:"alerts"`
    } `json:"data"`
}
```

### Core Functionality

1. **GET /api/v2/alerts** endpoint
   - Query alerts from database
   - Filter by: status, labels, time ranges, severity
   - Return in Alertmanager v2 compatible format
   - Support pagination
   - Support sorting

2. **Response Format Compatibility**
   - 100% Alertmanager API v2 compatible
   - Compatible with Grafana Alerting data source
   - Compatible with Prometheus alert queries
   - Compatible with amtool CLI

3. **Integration Points**
   - **TN-037**: AlertHistoryRepository (data source)
   - **TN-146**: PrometheusParser (format conversion)
   - **TN-147**: Shared response models
   - **TN-032**: AlertStorage for queries

---

## 🏗️ Architecture Design

### High-Level Flow

```
GET /api/v2/alerts?status=firing&severity=critical
           ↓
   PrometheusQueryHandler
           ↓
   Parse query parameters → AlertFilters
           ↓
   Query database (TN-037 HistoryRepository)
           ↓
   core.Alert[] → AlertmanagerAlert[]
           ↓
   Build AlertmanagerResponse
           ↓
   JSON response (200 OK)
```

### Components

1. **PrometheusQueryHandler**
   - HTTP handler for GET /api/v2/alerts
   - Query parameter parsing
   - Pagination support
   - Response formatting

2. **AlertmanagerResponseBuilder**
   - Convert []core.Alert → AlertmanagerResponse
   - Handle empty results
   - Apply time zone conversions
   - Format labels/annotations

3. **QueryParameterParser**
   - Parse filter, receiver, silenced, inhibited, active
   - Parse label matchers (alertname=~"Foo.*")
   - Parse time ranges (startsAt, endsAt)
   - Validate parameters

---

## 📊 API Specification (Alertmanager v2 Compatible)

### Endpoint

```
GET /api/v2/alerts
```

### Query Parameters (Alertmanager Compatible)

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `filter` | string | Matcher expression | `{alertname="HighCPU"}` |
| `receiver` | string | Receiver name filter | `team-ops` |
| `silenced` | bool | Include silenced alerts | `false` |
| `inhibited` | bool | Include inhibited alerts | `false` |
| `active` | bool | Include active only | `true` |
| `page` | int | Page number (pagination) | `1` |
| `limit` | int | Results per page | `100` |
| `sort` | string | Sort field | `startsAt:desc` |

### Response Format (Alertmanager v2)

**200 OK (Success)**:
```json
{
  "status": "success",
  "data": {
    "alerts": [
      {
        "labels": {
          "alertname": "HighCPU",
          "severity": "critical",
          "instance": "node-1"
        },
        "annotations": {
          "summary": "CPU usage above 90%",
          "description": "CPU usage is 95% for 5 minutes"
        },
        "startsAt": "2025-11-19T10:00:00Z",
        "endsAt": "2025-11-19T10:15:00Z",
        "generatorURL": "http://prometheus:9090/graph?...",
        "status": {
          "state": "active",
          "silencedBy": [],
          "inhibitedBy": []
        },
        "receivers": ["team-ops"],
        "fingerprint": "abc123def456"
      }
    ]
  }
}
```

**400 Bad Request (Invalid parameters)**:
```json
{
  "status": "error",
  "error": "invalid filter expression: {alertname"
}
```

**500 Internal Server Error (Database error)**:
```json
{
  "status": "error",
  "error": "internal server error"
}
```

---

## 🔗 Dependencies

### Required (All Complete ✅)

1. **TN-146**: Prometheus Alert Parser ✅
   - Status: COMPLETE (159% quality, Grade A+)
   - Usage: Format conversion (core.Alert → AlertmanagerAlert)
   - Integration: Reverse conversion (domain → wire format)

2. **TN-147**: POST /api/v2/alerts ✅
   - Status: COMPLETE (152% quality, Grade A+ EXCEPTIONAL)
   - Usage: Shared response models (AlertmanagerAlert, AlertmanagerResponse)
   - Integration: Common handler patterns

3. **TN-037**: Alert History Repository ✅
   - Status: COMPLETE (150% quality, Grade A+)
   - Usage: Query alerts from database
   - Methods: GetHistory(), GetRecentAlerts(), GetAlertsByFingerprint()

4. **TN-032**: AlertStorage Interface ✅
   - Status: COMPLETE (95% quality)
   - Usage: Storage abstraction for queries
   - Integration: PostgreSQL backend

### Optional (Enhancement)

1. **TN-035**: Alert Filtering Engine ✅
   - Status: COMPLETE (150% quality)
   - Usage: Advanced filtering capabilities
   - Integration: Label matcher expressions

2. **TN-133**: Silence Storage ✅
   - Status: COMPLETE (152.7% quality)
   - Usage: Check if alerts are silenced
   - Integration: `status.silencedBy` field

3. **TN-129**: Inhibition State Manager ✅
   - Status: COMPLETE (150% quality)
   - Usage: Check if alerts are inhibited
   - Integration: `status.inhibitedBy` field

---

## 📈 Scope & Features

### Minimum Viable Product (MVP) - 100%

1. ✅ GET /api/v2/alerts endpoint
2. ✅ Query all alerts (no filters)
3. ✅ Return AlertmanagerResponse format
4. ✅ Basic pagination (limit, offset)
5. ✅ Error handling (400, 500)

### Target Features (150% Quality)

6. ✅ Filter by status (firing/resolved)
7. ✅ Filter by labels (label matchers)
8. ✅ Filter by time range (startsAt, endsAt)
9. ✅ Filter by severity
10. ✅ Sorting (startsAt, severity)
11. ✅ Pagination metadata (total count)
12. ✅ Silence/inhibition status (via TN-133/129)
13. ✅ Prometheus metrics (query duration, error rate)
14. ✅ Comprehensive tests (unit + integration)
15. ✅ Performance optimization (< 100ms p95 for 1000 alerts)

---

## 🎯 Success Criteria (150% Quality)

### Functional Requirements (10)

1. **GET /api/v2/alerts** endpoint registered and accessible
2. **Alertmanager v2 compatible** response format (100%)
3. **Query from database** via TN-037 HistoryRepository
4. **Filter support**: status, labels, time ranges, severity
5. **Pagination**: limit, offset, total count
6. **Sorting**: by startsAt, severity, alertname
7. **Error handling**: 400 (invalid params), 500 (database error)
8. **Empty results**: Return empty array (not null)
9. **Silence/inhibition status**: Include silencedBy, inhibitedBy arrays
10. **Format conversion**: core.Alert → AlertmanagerAlert

### Non-Functional Requirements (10)

1. **Performance**: < 100ms p95 latency for 1000 alerts
2. **Throughput**: > 200 req/s
3. **Test coverage**: 85%+ (25+ tests)
4. **Documentation**: 2,000+ LOC (requirements, design, tasks, README)
5. **Prometheus metrics**: 6+ metrics (query duration, errors, results count)
6. **Error messages**: User-friendly and actionable
7. **Database efficiency**: Use indexes, avoid N+1 queries
8. **Memory efficiency**: Stream large results (not load all in memory)
9. **Backward compatible**: No breaking changes to TN-147
10. **Production-ready**: Zero technical debt

---

## 📊 Estimated Effort

### Timeline: **16-20 hours** (vs TN-147 12h, +33% complexity due to query logic)

| Phase | Tasks | Effort | LOC |
|-------|-------|--------|-----|
| **Phase 0**: Analysis | Requirements, design, dependencies | 2h | - |
| **Phase 1**: Documentation | requirements.md, design.md, tasks.md | 3h | 2,500 |
| **Phase 2**: Models | AlertmanagerResponse, query params | 2h | 200 |
| **Phase 3**: Handler | PrometheusQueryHandler, parameter parsing | 3h | 400 |
| **Phase 4**: Integration | HistoryRepository, format conversion | 2h | 200 |
| **Phase 5**: Testing | Unit + integration tests | 4h | 600 |
| **Phase 6**: Metrics | Prometheus metrics, logging | 2h | 150 |
| **Phase 7**: Certification | Performance testing, final report | 2h | 500 |
| **TOTAL** | - | **20h** | **4,550** |

**Risk Buffer**: +4h (20% contingency)
**Total with Buffer**: **24h**

---

## 🚀 Implementation Strategy

### Phase Breakdown (7 phases)

**Phase 0: Analysis** (2h) ✅ CURRENT
- Analyze existing code (TN-146, TN-147, TN-037)
- Design API specification
- Define response format
- Create dependency matrix

**Phase 1: Documentation** (3h)
- requirements.md: Functional & non-functional requirements
- design.md: Architecture, data flow, API spec
- tasks.md: Detailed task breakdown

**Phase 2: Models** (2h)
- QueryParameters struct
- AlertmanagerResponse struct (reuse from TN-147 if exists)
- Format conversion functions

**Phase 3: Handler** (3h)
- PrometheusQueryHandler struct
- Query parameter parsing
- Database query via TN-037
- Response formatting

**Phase 4: Integration** (2h)
- Register endpoint in main.go
- Wire dependencies
- Error handling
- Logging

**Phase 5: Testing** (4h)
- Unit tests: Parameter parsing, format conversion
- Integration tests: End-to-end query flow
- Edge cases: Empty results, invalid params

**Phase 6: Metrics** (2h)
- Prometheus metrics: query duration, error rate, results count
- Structured logging
- Performance profiling

**Phase 7: Certification** (2h)
- Performance benchmarks
- Load testing (k6 or similar)
- Final quality report
- COMPLETION_REPORT.md

---

## 🎯 Quality Targets (150%)

| Metric | Baseline (100%) | Target (150%) | Stretch (200%) |
|--------|-----------------|---------------|----------------|
| Implementation | 400 LOC | 600 LOC | 800+ LOC |
| Tests | 15 tests | 25+ tests | 35+ tests |
| Test Coverage | 75% | 85%+ | 90%+ |
| Documentation | 1,500 LOC | 2,500+ LOC | 3,000+ LOC |
| Performance (p95) | < 200ms | < 100ms | < 50ms |
| Throughput | > 100 req/s | > 200 req/s | > 500 req/s |
| Metrics | 3 | 6+ | 10+ |

**Target Score**: **98+/100** (Grade A+ EXCEPTIONAL)

---

## 🔍 Risk Assessment

### High Risks

1. **Query Performance**
   - Risk: Slow queries for large datasets (100K+ alerts)
   - Mitigation: Use TN-032 indexes, pagination, query optimization
   - Impact: Performance target miss

2. **Format Compatibility**
   - Risk: Subtle differences from Alertmanager API
   - Mitigation: Comprehensive integration tests with amtool/Grafana
   - Impact: Compatibility issues with Prometheus ecosystem

### Medium Risks

3. **Filter Complexity**
   - Risk: Complex label matcher expressions (regex, negative matchers)
   - Mitigation: Reuse TN-035 filter engine, extensive testing
   - Impact: Missing filter features

4. **Silence/Inhibition Status**
   - Risk: Performance overhead checking TN-133/129
   - Mitigation: Batch queries, caching, optional feature
   - Impact: Slow queries with status checks

### Low Risks

5. **Pagination Edge Cases**
   - Risk: Off-by-one errors, empty pages
   - Mitigation: Comprehensive unit tests
   - Impact: Minor bugs in pagination logic

---

## 📦 Deliverables (Estimated)

### Code (1,550 LOC)
1. `prometheus_query_handler.go` (500 LOC): Main handler
2. `prometheus_query_models.go` (200 LOC): Request/response models
3. `prometheus_query_parser.go` (200 LOC): Parameter parsing
4. `prometheus_query_converter.go` (250 LOC): Format conversion
5. `prometheus_query_metrics.go` (150 LOC): Prometheus metrics
6. `prometheus_query_test.go` (600 LOC): Unit tests
7. `main.go` (+50 LOC): Integration

### Documentation (2,500+ LOC)
1. `requirements.md` (800 LOC): Requirements & acceptance criteria
2. `design.md` (1,000 LOC): Architecture & API spec
3. `tasks.md` (500 LOC): Phase breakdown & timeline
4. `COMPLETION_REPORT.md` (200+ LOC): Final certification

**Total**: ~4,050 LOC

---

## ✅ Analysis Complete

**Status**: ✅ Phase 0 Analysis COMPLETE
**Duration**: ~2 hours
**Next Phase**: Phase 1 Documentation (requirements.md, design.md, tasks.md)

**Ready to proceed with 150% quality implementation!** 🚀

---

**Date**: 2025-11-19
**Analyst**: AI Assistant
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
