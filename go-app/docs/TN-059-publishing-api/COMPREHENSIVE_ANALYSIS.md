# TN-059: Publishing API Endpoints - Comprehensive Multi-Level Analysis

**Task ID:** TN-059
**Title:** Publishing API endpoints
**Phase:** ФАЗА 5: Publishing System
**Target Quality:** 150% (Grade A+)
**Status:** Analysis Phase
**Date:** 2025-11-13
**Analyst:** AI Agent (Enterprise Architecture)

---

## Executive Summary

**TN-059** represents the consolidation and enhancement of the Publishing System's HTTP API layer, building upon the successful completion of TN-046 through TN-058. This task aims to create a unified, enterprise-grade RESTful API that exposes all publishing functionality through a consistent, well-documented, and highly performant interface.

### Strategic Importance

The Publishing API endpoints serve as the **primary interface** for:
- External systems integration (Alertmanager, monitoring tools)
- Internal service communication (microservices architecture)
- Operations and SRE teams (manual interventions, testing)
- Future UI/Dashboard integration (Фаза 7)

### Current State Assessment

| Aspect | Current Status | Target State |
|--------|---------------|--------------|
| **API Coverage** | Partial (60%) | Complete (100%) |
| **Consistency** | Mixed (/api/v1, /api/v2) | Unified (/api/v2/publishing) |
| **Documentation** | Scattered | OpenAPI 3.0 Spec |
| **Performance** | Unknown | <10ms p99 |
| **Security** | Basic | Enterprise-grade |
| **Testing** | Minimal | 90%+ coverage |

---

## 1. Comprehensive API Inventory

### 1.1 Existing API Endpoints (Across 3 Sources)

#### **Source A: TN-056 Publishing Queue Handlers** (`internal/infrastructure/publishing/handlers.go`)
**Base Path:** `/api/v1/publishing`
**Status:** ✅ Implemented (TN-056, 150% quality)

| # | Method | Endpoint | Handler | Purpose | Status |
|---|--------|----------|---------|---------|--------|
| 1 | GET | `/targets` | ListTargets | List all publishing targets | ✅ |
| 2 | GET | `/targets/{name}` | GetTarget | Get specific target details | ✅ |
| 3 | POST | `/targets/refresh` | RefreshTargets | Manually refresh target discovery | ✅ |
| 4 | POST | `/targets/{name}/test` | TestTarget | Test connectivity to a target | ✅ |
| 5 | GET | `/stats` | GetStats | Basic publishing statistics | ✅ |
| 6 | GET | `/queue` | GetQueueStatus | Queue status (size, utilization) | ✅ |
| 7 | GET | `/mode` | GetPublishingMode | Current publishing mode | ✅ |
| 8 | POST | `/submit` | SubmitAlert | Submit alert to queue | ✅ |
| 9 | GET | `/queue/stats` | GetDetailedQueueStats | Detailed queue metrics | ✅ |
| 10 | GET | `/jobs` | ListJobs | List all jobs (with filters) | ✅ |
| 11 | GET | `/jobs/{id}` | GetJob | Get job status by ID | ✅ |
| 12 | GET | `/dlq` | ListDLQEntries | List Dead Letter Queue entries | ✅ |
| 13 | POST | `/dlq/{id}/replay` | ReplayDLQEntry | Replay failed DLQ entry | ✅ |
| 14 | DELETE | `/dlq/purge` | PurgeDLQ | Purge old DLQ entries | ✅ |

**Subtotal:** 14 endpoints

---

#### **Source B: TN-057 Publishing Metrics & Stats** (`cmd/server/handlers/publishing_stats.go`)
**Base Path:** `/api/v2/publishing`
**Status:** ✅ Implemented (TN-057, 150% quality, 820-2,300x performance)

| # | Method | Endpoint | Handler | Purpose | Status |
|---|--------|----------|---------|---------|--------|
| 15 | GET | `/metrics` | GetMetrics | Raw Prometheus metrics (JSON) | ✅ |
| 16 | GET | `/stats` | GetStats | Aggregated statistics | ✅ |
| 17 | GET | `/health` | GetHealth | System health status | ✅ |
| 18 | GET | `/stats/{target}` | GetTargetStats | Per-target statistics | ✅ |
| 19 | GET | `/trends` | GetTrends | Historical trend analysis | ✅ |

**Subtotal:** 5 endpoints

---

#### **Source C: TN-058 Parallel Publishing** (`internal/api/handlers/parallel_publish_handler.go`)
**Base Path:** `/api/v1/publish/parallel`
**Status:** ✅ Implemented (TN-058, 150% quality, 5,076x throughput)

| # | Method | Endpoint | Handler | Purpose | Status |
|---|--------|----------|---------|---------|--------|
| 20 | POST | `/` (base) | PublishToTargets | Publish to specific targets | ⚠️ Not implemented |
| 21 | POST | `/all` | PublishToAll | Publish to all targets | ✅ |
| 22 | POST | `/healthy` | PublishToHealthy | Publish to healthy targets only | ✅ |
| 23 | GET | `/status` | GetStatus | Parallel publisher status | ✅ |

**Subtotal:** 4 endpoints (3 implemented, 1 pending)

---

#### **Source D: Health Monitoring (TN-049)** - COMMENTED OUT in main.go
**Base Path:** `/api/v2/publishing/targets/health`
**Status:** ⚠️ Code exists, not registered

| # | Method | Endpoint | Handler | Purpose | Status |
|---|--------|----------|---------|---------|--------|
| 24 | GET | `/stats` | GetHealthStats | Health monitoring statistics | 🔴 Not registered |
| 25 | GET | `/{name}` | GetHealthByName | Get health for specific target | 🔴 Not registered |
| 26 | POST | `/{name}/check` | CheckHealth | Force health check | 🔴 Not registered |
| 27 | GET | `/` (base) | GetHealth | Get all targets health | 🔴 Not registered |

**Subtotal:** 4 endpoints (commented out)

---

### 1.2 Missing API Endpoints (From Фаза 6: TN-61 to TN-75)

#### **Фаза 6 Requirements Analysis**

| TN # | Endpoint | Method | Purpose | Dependency | Priority |
|------|----------|--------|---------|------------|----------|
| TN-61 | `/webhook` | POST | Universal webhook endpoint | ✅ Exists (main.go) | Low |
| TN-62 | `/webhook/proxy` | POST | Intelligent proxy endpoint | ⚠️ Needs enhancement | Medium |
| TN-63 | `/history` | GET | Alert history with filters | ✅ Exists (TN-37) | Low |
| TN-64 | `/report` | GET | Analytics endpoint | ✅ Exists (TN-38) | Low |
| TN-65 | `/metrics` | GET | Prometheus metrics | ✅ Exists (TN-57) | Low |
| TN-66 | `/publishing/targets` | GET | List targets | ✅ Exists (TN-56) | Low |
| TN-67 | `/publishing/targets/refresh` | POST | Refresh discovery | ✅ Exists (TN-56) | Low |
| TN-68 | `/publishing/mode` | GET | Current mode | ✅ Exists (TN-56) | Low |
| TN-69 | `/publishing/stats` | GET | Statistics | ✅ Exists (TN-57) | Low |
| TN-70 | `/publishing/test/{target}` | POST | Test target | ✅ Exists (TN-56) | Low |
| TN-71 | `/classification/stats` | GET | LLM statistics | 🔴 Missing | **High** |
| TN-72 | `/classification/classify` | POST | Manual classification | 🔴 Missing | **High** |
| TN-73 | `/classification/models` | GET | Available models | 🔴 Missing | **High** |
| TN-74 | `/enrichment/mode` | GET | Current mode | ✅ Exists (main.go) | Low |
| TN-75 | `/enrichment/mode` | POST | Switch mode | ✅ Exists (main.go) | Low |

**Analysis:**
- **12/15 endpoints exist** (80% coverage)
- **3/15 endpoints missing** (20% gap)
- Missing endpoints are all in `/classification/*` domain
- Most Фаза 6 requirements are already covered by TN-46 to TN-58

---

## 2. Gap Analysis & Consolidation Requirements

### 2.1 Critical Issues

#### **Issue 1: API Versioning Inconsistency** 🔴 **CRITICAL**
- **Problem:** Mixed use of `/api/v1/publishing` and `/api/v2/publishing`
- **Impact:** Client confusion, breaking changes risk
- **Solution:** Standardize on `/api/v2/publishing` for all new endpoints, maintain v1 for backward compatibility
- **Effort:** 2-3 hours (refactoring + documentation)

#### **Issue 2: Health Monitoring Endpoints Not Registered** 🟡 **MEDIUM**
- **Problem:** 4 TN-049 health endpoints exist but commented out in main.go (lines 920-935)
- **Impact:** Missing health monitoring API access
- **Solution:** Uncomment and register endpoints, add integration tests
- **Effort:** 1 hour

#### **Issue 3: Classification API Missing** 🔴 **HIGH PRIORITY**
- **Problem:** No HTTP API for LLM classification service (TN-33)
- **Impact:** Cannot trigger manual classification or view stats via API
- **Solution:** Create `/api/v2/classification/*` endpoints (3 new endpoints)
- **Effort:** 4-6 hours

#### **Issue 4: Parallel Publisher Target Resolution Not Implemented** 🟡 **MEDIUM**
- **Problem:** `PublishToTargets` handler has TODO (line 102-119 in parallel_publish_handler.go)
- **Impact:** Cannot publish to specific target names
- **Solution:** Implement target name → PublishingTarget resolution
- **Effort:** 2 hours

#### **Issue 5: No OpenAPI/Swagger Documentation** 🟡 **MEDIUM**
- **Problem:** API documentation scattered across Go comments
- **Impact:** Poor developer experience, integration friction
- **Solution:** Generate OpenAPI 3.0 specification
- **Effort:** 6-8 hours

#### **Issue 6: Lack of Unified API Testing** 🟡 **MEDIUM**
- **Problem:** Each handler has isolated tests, no E2E API tests
- **Impact:** Integration bugs, regression risks
- **Solution:** Create comprehensive API integration test suite
- **Effort:** 8-10 hours

---

### 2.2 API Consolidation Plan

#### **Strategy: Unified Publishing API v2**

```
/api/v2/publishing/
├── targets/                    # Target Discovery & Management
│   ├── GET    /                # List all targets (TN-056)
│   ├── GET    /{name}          # Get target details (TN-056)
│   ├── POST   /refresh         # Refresh discovery (TN-056)
│   ├── POST   /{name}/test     # Test target (TN-056)
│   └── health/                 # Health Monitoring (TN-049)
│       ├── GET    /            # All targets health
│       ├── GET    /{name}      # Target health by name
│       ├── POST   /{name}/check # Force health check
│       └── GET    /stats       # Health statistics
│
├── queue/                      # Publishing Queue (TN-056)
│   ├── GET    /status          # Queue status
│   ├── GET    /stats           # Detailed statistics
│   ├── POST   /submit          # Submit alert
│   └── jobs/                   # Job Management
│       ├── GET    /            # List jobs (with filters)
│       └── GET    /{id}        # Get job by ID
│
├── dlq/                        # Dead Letter Queue (TN-056)
│   ├── GET    /                # List DLQ entries
│   ├── POST   /{id}/replay     # Replay entry
│   └── DELETE /purge           # Purge old entries
│
├── parallel/                   # Parallel Publishing (TN-058)
│   ├── POST   /targets         # Publish to specific targets
│   ├── POST   /all             # Publish to all
│   ├── POST   /healthy         # Publish to healthy only
│   └── GET    /status          # Parallel publisher status
│
├── metrics/                    # Metrics & Stats (TN-057)
│   ├── GET    /raw             # Raw Prometheus metrics (JSON)
│   ├── GET    /stats           # Aggregated statistics
│   ├── GET    /trends          # Historical trends
│   └── GET    /targets/{name}  # Per-target stats
│
└── health                      # System Health (TN-057)
    └── GET    /                # Overall publishing health

/api/v2/classification/         # NEW: LLM Classification API
├── GET    /stats               # Classification statistics
├── POST   /classify            # Manual classification
└── GET    /models              # Available LLM models

/api/v2/enrichment/             # Alert Enrichment (existing)
├── GET    /mode                # Current enrichment mode
└── POST   /mode                # Switch enrichment mode

/api/v1/publishing/             # Legacy (Backward Compatibility)
└── [All TN-056 endpoints]      # Keep for existing clients
```

**Total Endpoints:**
- **Existing:** 23 (19 active, 4 commented)
- **New:** 3 (Classification API)
- **Reorganized:** 27 endpoints under unified structure
- **Total:** 27 unified endpoints

---

## 3. Technical Architecture

### 3.1 Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway Layer                         │
│  (Routing, Versioning, Authentication, Rate Limiting)        │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
┌───────▼──────────┐         ┌────────▼─────────┐
│  Handler Layer   │         │  Middleware      │
│  (HTTP Handlers) │◄────────┤  (Metrics, Logs) │
└───────┬──────────┘         └──────────────────┘
        │
        │ ┌────────────────────────────────────┐
        ├─┤ Publishing Handlers (TN-056)       │
        ├─┤ Stats Handlers (TN-057)            │
        ├─┤ Parallel Publish Handlers (TN-058) │
        ├─┤ Health Handlers (TN-049)           │
        └─┤ Classification Handlers (NEW)      │
          └────────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
┌───────▼──────────┐         ┌────────▼─────────┐
│  Business Layer  │         │  Infrastructure  │
│  (Services)      │◄────────┤  (Repositories)  │
└──────────────────┘         └──────────────────┘
```

### 3.2 API Design Principles

1. **RESTful Design**
   - Resource-oriented URLs
   - Standard HTTP methods (GET, POST, PUT, DELETE)
   - HTTP status codes (200, 201, 400, 404, 500, 503)
   - JSON request/response bodies

2. **Consistency**
   - Unified naming conventions (snake_case for JSON)
   - Standardized error responses
   - Common pagination pattern
   - Consistent timestamp format (RFC3339)

3. **Performance**
   - Response time: <10ms p50, <50ms p99
   - Throughput: >1,000 req/s per endpoint
   - Caching: ETags, Cache-Control headers
   - Compression: gzip/brotli support

4. **Security**
   - Authentication: API keys, JWT tokens (future)
   - Authorization: RBAC (role-based access control)
   - Rate limiting: 100 req/min per client
   - Input validation: JSON schema validation
   - CORS: Configurable origins

5. **Observability**
   - Request ID tracking (X-Request-ID header)
   - Structured logging (slog)
   - Prometheus metrics (per-endpoint)
   - OpenTelemetry tracing (future)

---

## 4. Dependencies & Integration Points

### 4.1 Internal Dependencies

| Component | Package | Status | Version |
|-----------|---------|--------|---------|
| Publishing Queue | `internal/infrastructure/publishing` | ✅ | TN-056 |
| Stats Collector | `internal/business/publishing` | ✅ | TN-057 |
| Parallel Publisher | `internal/infrastructure/publishing` | ✅ | TN-058 |
| Health Monitor | `internal/business/publishing` | ✅ | TN-049 |
| Target Discovery | `internal/business/publishing` | ✅ | TN-047 |
| Classification Service | `internal/core/services` | ⚠️ | TN-033 (no HTTP API) |
| Alert Processor | `internal/core/services` | ✅ | TN-34 |

### 4.2 External Dependencies

| Dependency | Purpose | Version | Status |
|------------|---------|---------|--------|
| gorilla/mux | HTTP router | v1.8.1 | ✅ |
| Prometheus | Metrics | v1.19.0 | ✅ |
| slog | Structured logging | stdlib | ✅ |
| validator/v10 | Input validation | v10.22.0 | 🔴 Missing |
| swaggo/swag | OpenAPI generation | v1.16.3 | 🔴 Missing |

**Required Additions:**
1. `github.com/go-playground/validator/v10` - JSON schema validation
2. `github.com/swaggo/swag` - OpenAPI documentation generation
3. `github.com/swaggo/http-swagger` - Swagger UI serving

---

## 5. Risk Assessment

### 5.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Breaking Changes to v1 API** | Medium | High | Maintain v1 endpoints, deprecation plan |
| **Performance Regression** | Low | High | Comprehensive benchmarking, load testing |
| **Security Vulnerabilities** | Medium | Critical | Input validation, rate limiting, security audit |
| **Integration Complexity** | Medium | Medium | Incremental rollout, feature flags |
| **Documentation Drift** | High | Medium | Auto-generate from code annotations |

### 5.2 Schedule Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Scope Creep** | High | High | Strict scope definition, MVP approach |
| **Testing Time Underestimation** | Medium | Medium | Buffer 30% for testing phase |
| **Dependency Delays** | Low | Medium | Early identification, parallel work |

---

## 6. Success Criteria (150% Quality Target)

### 6.1 Baseline Requirements (100%)

| Criteria | Target | Measurement |
|----------|--------|-------------|
| API Coverage | All 27 endpoints | Endpoint count |
| Test Coverage | 80%+ | go test -cover |
| Response Time | <50ms p99 | Benchmark tests |
| Documentation | All endpoints documented | API docs completeness |
| Error Handling | All errors handled | Code review |

### 6.2 Enhanced Requirements (150%)

| Criteria | Target | Measurement |
|----------|--------|-------------|
| API Coverage | 100% + OpenAPI spec | Swagger UI |
| Test Coverage | 90%+ | go test -cover |
| Response Time | <10ms p99 | Benchmark tests |
| Documentation | OpenAPI + guides + examples | Docs LOC count |
| Error Handling | Comprehensive + typed errors | Error types count |
| **Performance** | >1,000 req/s per endpoint | Load testing (k6) |
| **Security** | Rate limiting + validation | Security audit |
| **Observability** | Metrics + tracing | Prometheus metrics |
| **Developer Experience** | Interactive API docs | Swagger UI quality |

### 6.3 Key Performance Indicators (KPIs)

| KPI | Baseline | 150% Target | Measurement Method |
|-----|----------|-------------|---------------------|
| **API Response Time** | <50ms | <10ms | k6 load tests, p50/p95/p99 |
| **Throughput** | 100 req/s | 1,000 req/s | k6 stress tests |
| **Test Coverage** | 80% | 90%+ | go test -cover |
| **Documentation LOC** | 1,000 | 3,000+ | Docs word count |
| **OpenAPI Completeness** | 0% | 100% | All endpoints documented |
| **Error Type Coverage** | Basic | 15+ types | Error taxonomy |
| **Security Score** | Basic | Grade A | gosec, security audit |

---

## 7. Implementation Phases

### Phase 0: Analysis & Planning ✅ (Current)
**Duration:** 4 hours
**Deliverables:**
- ✅ Comprehensive analysis document (this file)
- Next: Requirements document
- Next: Design document

### Phase 1: Requirements Engineering
**Duration:** 2 hours
**Deliverables:**
- Functional requirements (FR-1 to FR-15)
- Non-functional requirements (NFR-1 to NFR-10)
- Acceptance criteria
- 150% quality metrics

### Phase 2: API Architecture Design
**Duration:** 3 hours
**Deliverables:**
- Unified API architecture diagram
- OpenAPI 3.0 specification (initial)
- Versioning strategy
- Authentication/authorization design
- Rate limiting design

### Phase 3: API Consolidation
**Duration:** 6 hours
**Deliverables:**
- Merge existing handlers into unified structure
- Standardize endpoint naming
- Implement middleware stack
- Update main.go routing

### Phase 4: New Endpoints Implementation
**Duration:** 8 hours
**Deliverables:**
- Classification API (3 endpoints)
- Health monitoring API registration (4 endpoints)
- Parallel publisher target resolution (1 endpoint)
- Enhanced webhook proxy (1 endpoint)

### Phase 5: Comprehensive Testing
**Duration:** 10 hours
**Deliverables:**
- Unit tests for all handlers (90%+ coverage)
- Integration tests (E2E)
- Load tests (k6 scenarios)
- Security tests (gosec, input fuzzing)

### Phase 6: Enterprise Documentation
**Duration:** 8 hours
**Deliverables:**
- OpenAPI 3.0 specification (complete)
- Swagger UI integration
- API usage guides
- Code examples
- Troubleshooting guide

### Phase 7: Performance Optimization
**Duration:** 6 hours
**Deliverables:**
- Benchmarking all endpoints
- Response caching (Redis)
- Rate limiting implementation
- Performance monitoring dashboards

### Phase 8: Integration & Validation
**Duration:** 4 hours
**Deliverables:**
- Main.go full integration
- E2E validation tests
- Production readiness checklist
- Deployment guide

### Phase 9: 150% Quality Certification
**Duration:** 3 hours
**Deliverables:**
- Final audit report
- Performance validation results
- Documentation review
- Grade A+ certification

---

## 8. Estimated Effort

| Phase | Duration | LOC (Estimated) | Files |
|-------|----------|-----------------|-------|
| Phase 0 | 4h | 2,000 (docs) | 1 |
| Phase 1 | 2h | 800 (docs) | 1 |
| Phase 2 | 3h | 1,200 (docs + spec) | 2 |
| Phase 3 | 6h | 1,500 (code + tests) | 8 |
| Phase 4 | 8h | 2,000 (code + tests) | 12 |
| Phase 5 | 10h | 2,500 (tests) | 10 |
| Phase 6 | 8h | 3,000 (docs) | 8 |
| Phase 7 | 6h | 1,000 (code + benchmarks) | 6 |
| Phase 8 | 4h | 500 (integration) | 3 |
| Phase 9 | 3h | 1,000 (docs) | 2 |
| **Total** | **54h** | **15,500 LOC** | **53 files** |

**Timeline:**
- **Optimistic:** 5 days (10h/day)
- **Realistic:** 7 days (8h/day)
- **Pessimistic:** 10 days (6h/day)

**Comparison to Previous Tasks:**
- TN-057: 12h actual (vs 82h estimated, 85% faster) → Quality: 150%+
- TN-058: 4h actual (vs 12h estimated, 67% faster) → Quality: 150%+
- **TN-059 Target:** 40-50h actual (vs 54h estimated, ~25% faster) → Quality: 150%+

---

## 9. Deliverables Checklist

### Code Deliverables
- [ ] Unified API handler structure (`internal/api/handlers/publishing/`)
- [ ] Classification API handlers (3 endpoints)
- [ ] Health monitoring API registration (4 endpoints)
- [ ] Parallel publisher target resolution
- [ ] Middleware stack (auth, rate limit, metrics, logging)
- [ ] Input validation (JSON schema)
- [ ] Error handling improvements
- [ ] OpenAPI annotations (swag comments)

### Test Deliverables
- [ ] Unit tests (90%+ coverage)
- [ ] Integration tests (E2E)
- [ ] Load tests (k6 scenarios)
- [ ] Security tests (gosec, fuzzing)
- [ ] Benchmark tests

### Documentation Deliverables
- [ ] Comprehensive analysis (this document)
- [ ] Requirements document
- [ ] Design document
- [ ] OpenAPI 3.0 specification
- [ ] API usage guide
- [ ] Code examples
- [ ] Troubleshooting guide
- [ ] Performance benchmarks report
- [ ] Final certification report

### Infrastructure Deliverables
- [ ] Main.go integration
- [ ] Swagger UI endpoint
- [ ] Prometheus metrics
- [ ] Grafana dashboard (API metrics)
- [ ] Production deployment guide

---

## 10. Next Steps

### Immediate Actions (Next 1 hour)
1. ✅ Complete this comprehensive analysis
2. ⏭️ Create feature branch: `feature/TN-059-publishing-api-150pct`
3. ⏭️ Start Phase 1: Requirements Engineering
4. ⏭️ Write requirements.md document

### Short-term (Next 6 hours)
- Complete Phase 1-2 (Requirements + Design)
- Begin Phase 3 (API Consolidation)
- Set up OpenAPI generation infrastructure

### Medium-term (Next 2-3 days)
- Complete Phases 3-5 (Consolidation + New Endpoints + Testing)
- Achieve 90%+ test coverage
- Performance benchmarking

### Long-term (4-7 days)
- Complete Phases 6-9 (Documentation + Optimization + Certification)
- Full integration with main.go
- Production deployment readiness

---

## 11. Conclusion

**TN-059 Publishing API Endpoints** is a critical consolidation and enhancement task that will:
1. **Unify** 27 existing and new endpoints under a consistent API structure
2. **Enhance** developer experience with OpenAPI/Swagger documentation
3. **Improve** performance to <10ms response times
4. **Secure** the API with authentication, authorization, and rate limiting
5. **Achieve** 150% quality certification (Grade A+)

This task builds on the **exceptional foundation** laid by TN-046 through TN-058 (all completed at 150%+ quality) and represents the **final critical component** of the Publishing System's public interface.

**Estimated Completion:** 7 days (54 hours realistic, targeting 40-50h with efficiency gains)
**Risk Level:** Medium (well-defined scope, strong dependencies)
**Strategic Value:** **CRITICAL** (enables all external integrations and future UI)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-13
**Next Review:** After Phase 1 completion
**Status:** ✅ APPROVED - Proceed to Phase 1
