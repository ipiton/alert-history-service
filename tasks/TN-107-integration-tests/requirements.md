# TN-107: Integration Tests for API Endpoints - Requirements

**Task ID**: TN-107
**Priority**: P2 (Production Ready)
**Estimated Effort**: 6-8 hours
**Target Quality**: 150% (comprehensive + edge cases)
**Status**: IN PROGRESS
**Started**: 2025-11-30

---

## 1. Executive Summary

Integration tests validate that all system components work correctly together in realistic scenarios. Unlike unit tests that test components in isolation, integration tests verify:

- End-to-end API flows (request → processing → response)
- Database operations (PostgreSQL, migrations, transactions)
- Cache integration (Redis L1/L2, fallback)
- External service integration (LLM classification, publishing targets)
- Error handling across component boundaries
- Performance under realistic loads

**Success Criteria**: 80%+ integration test coverage of critical paths with 100% test pass rate.

---

## 2. Functional Requirements

### FR-1: API Integration Tests
**Priority**: CRITICAL
**Description**: Test all REST API endpoints with realistic payloads

**Test Categories**:
- **POST /webhook** - Universal webhook ingestion (Alertmanager, Prometheus, Generic)
- **POST /webhook/proxy** - Intelligent proxy with classification & publishing
- **GET /history** - Alert history with filtering, pagination, sorting
- **GET /history/top** - Top firing alerts analytics
- **GET /history/flapping** - Flapping detection
- **POST /classification/classify** - LLM classification (with cache)
- **GET/POST /enrichment/mode** - Mode switching
- **GET/POST /config** - Configuration management
- **POST /config/rollback** - Configuration rollback
- **Silence APIs** - Create, update, delete, list silences
- **Inhibition APIs** - Rules, status, check

**Acceptance Criteria**:
- ✅ All endpoints return correct HTTP status codes
- ✅ Response bodies match expected schemas
- ✅ Database state changes are persisted correctly
- ✅ Cache invalidation works as expected
- ✅ Error responses include request IDs and details

### FR-2: Database Integration Tests
**Priority**: HIGH
**Description**: Validate PostgreSQL operations with real database

**Test Categories**:
- **Migrations**: All migrations run successfully, rollback works
- **CRUD Operations**: Insert, select, update, delete alerts
- **Transactions**: Atomic operations, rollback on error
- **Queries**: Complex queries with JOINs, aggregations, window functions
- **Indexes**: Query plans use indexes correctly
- **Connection Pool**: Pool exhaustion handling, connection reuse

**Acceptance Criteria**:
- ✅ All migrations apply cleanly
- ✅ Data integrity maintained across operations
- ✅ Transactions properly isolated
- ✅ Query performance meets targets (<100ms for simple, <500ms for complex)

### FR-3: Redis Integration Tests
**Priority**: HIGH
**Description**: Validate Redis caching and fallback behavior

**Test Categories**:
- **L1/L2 Cache**: Memory cache → Redis cache → database fallback
- **Cache Invalidation**: Updates trigger cache clears
- **TTL Expiration**: Cache entries expire correctly
- **Failure Scenarios**: Redis unavailable → fallback to L1 only
- **Distributed Locks**: Concurrent access with locks

**Acceptance Criteria**:
- ✅ Cache hit rate >90% for hot paths
- ✅ Graceful degradation when Redis fails
- ✅ No cache stampede under load
- ✅ TTL enforcement working

### FR-4: LLM Classification Integration Tests
**Priority**: MEDIUM
**Description**: Validate LLM integration with mocking

**Test Categories**:
- **HTTP Client**: LLM API calls with retries
- **Cache Hit**: L1/L2 cache bypass with force=true
- **Fallback**: Rule-based classification when LLM fails
- **Timeout Handling**: Context cancellation, circuit breaker
- **Batch Processing**: Multiple alerts classified concurrently

**Acceptance Criteria**:
- ✅ Classification completes within 1s (cached) or 5s (LLM call)
- ✅ Cache hit rate >95%
- ✅ Fallback works when LLM unavailable
- ✅ No goroutine leaks

### FR-5: Publishing Flow Integration Tests
**Priority**: HIGH
**Description**: Validate alert publishing to multiple targets

**Test Categories**:
- **Target Discovery**: K8s Secrets → Publishing targets
- **Queue Processing**: Submit → Process → Publish → DLQ on failure
- **Parallel Publishing**: Fan-out to multiple targets
- **Retry Logic**: Exponential backoff on failures
- **Health Monitoring**: Unhealthy targets skipped

**Acceptance Criteria**:
- ✅ Alerts reach all healthy targets
- ✅ Failed alerts move to DLQ
- ✅ Retry logic exhausts attempts before DLQ
- ✅ Metrics recorded correctly

### FR-6: Error Handling & Recovery Tests
**Priority**: HIGH
**Description**: Validate graceful error handling

**Test Scenarios**:
- Database connection lost → graceful degradation
- Redis connection lost → fallback to L1 cache
- LLM timeout → rule-based fallback
- Publishing target unavailable → DLQ storage
- Configuration invalid → rollback to previous
- Concurrent requests → no race conditions

**Acceptance Criteria**:
- ✅ No panics under error conditions
- ✅ Errors logged with context
- ✅ Prometheus metrics record failures
- ✅ System remains available (degraded mode)

---

## 3. Non-Functional Requirements

### NFR-1: Test Infrastructure
**Requirement**: Reusable test infrastructure with Docker Compose

**Components**:
- PostgreSQL container (test database)
- Redis container (test cache)
- Mock LLM server (HTTP responses)
- Test data fixtures (realistic alert payloads)
- Helper functions (setup, teardown, assertions)

### NFR-2: Test Isolation
**Requirement**: Tests must be isolated and repeatable

**Rules**:
- Each test uses fresh database (migrations + cleanup)
- Tests can run in parallel without interference
- No shared mutable state between tests
- Idempotent (same result on re-run)

### NFR-3: Performance Targets
**Requirement**: Integration tests complete in reasonable time

**Targets**:
- Single test: <5s
- Test suite: <5min total
- Parallel execution: 4-8 workers
- CI/CD friendly (GitHub Actions)

### NFR-4: Documentation
**Requirement**: Comprehensive test documentation

**Deliverables**:
- Test organization README
- How to run tests locally
- CI/CD integration guide
- Troubleshooting common issues

---

## 4. Test Organization

### Structure:
```
go-app/
├── test/
│   ├── integration/
│   │   ├── api_test.go          # API endpoint tests
│   │   ├── database_test.go     # PostgreSQL tests
│   │   ├── cache_test.go        # Redis tests
│   │   ├── classification_test.go # LLM tests
│   │   ├── publishing_test.go   # Publishing flow tests
│   │   ├── errors_test.go       # Error handling tests
│   │   └── helpers.go           # Test utilities
│   ├── fixtures/
│   │   ├── alerts.json          # Sample alert payloads
│   │   ├── silences.json        # Sample silences
│   │   └── config.yaml          # Test configurations
│   └── docker-compose.test.yml  # Test infrastructure
```

### Test Tags:
- `//go:build integration` - Integration tests (skip in unit tests)
- Run with: `go test -tags=integration ./test/integration/...`

---

## 5. Dependencies

### Requires (Completed):
- ✅ TN-031: Domain Models
- ✅ TN-032: Storage Layer
- ✅ TN-033-036: LLM Classification
- ✅ TN-037: History Repository
- ✅ TN-046-060: Publishing System
- ✅ TN-131-136: Silencing System
- ✅ TN-126-130: Inhibition System

### Blocks:
- TN-108: E2E Tests (depends on integration test patterns)
- TN-109: Load Testing (depends on integration test stability)

---

## 6. Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Docker not available in CI | High | Medium | Use testcontainers-go, fallback to GitHub Services |
| Tests too slow (>10min) | Medium | High | Parallel execution, selective test runs |
| Flaky tests (timing issues) | High | Medium | Proper synchronization, retries, timeouts |
| Database state pollution | High | Low | Isolated databases per test, cleanup |
| External service dependencies | Medium | Medium | Mock servers, circuit breakers |

---

## 7. Acceptance Criteria

### Definition of Done:

1. ✅ **Coverage**: 80%+ integration test coverage of critical paths
2. ✅ **Pass Rate**: 100% test pass rate (no flaky tests)
3. ✅ **Performance**: Test suite completes in <5min
4. ✅ **CI/CD**: Tests run automatically on PR + merge
5. ✅ **Documentation**: Comprehensive README with examples
6. ✅ **Quality**: Tests follow Go best practices (table-driven, subtests, helpers)

---

## 8. Out of Scope

- ❌ Load testing (TN-109 - separate task)
- ❌ E2E UI tests (TN-108 - separate task)
- ❌ Performance profiling (covered by benchmarks)
- ❌ Security penetration testing (separate task)

---

**Author**: AI Assistant
**Date**: 2025-11-30
**Version**: 1.0
