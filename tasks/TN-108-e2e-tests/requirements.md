# TN-108: E2E Tests for Critical Flows - Requirements

**Status**: In Progress
**Priority**: P2 (Testing & Documentation)
**Estimated Effort**: 6-8 hours
**Actual Effort**: TBD
**Quality Target**: 150%+

---

## Overview

End-to-end tests validate critical user journeys through the entire system, from alert ingestion to publishing. These tests ensure that all components work together correctly in realistic scenarios.

## Objectives

1. **Validate Critical Flows**: Test complete user journeys end-to-end
2. **Integration Verification**: Ensure all components integrate correctly
3. **Regression Prevention**: Catch breaking changes before production
4. **Production Confidence**: Validate system behavior under realistic conditions

## Scope

### In Scope

1. **Alert Ingestion Flows**
   - Alertmanager webhook → storage
   - Prometheus format → storage
   - Generic webhook → storage

2. **Classification Pipeline**
   - Alert ingestion → LLM classification → enrichment → storage
   - Cache hit scenarios (L1 and L2)
   - LLM fallback (rule-based)

3. **Publishing Flows**
   - Single target publishing (Slack, PagerDuty, Rootly)
   - Multi-target parallel publishing
   - Publishing retry on failure
   - Target discovery via K8s Secrets

4. **History & Query Flows**
   - Alert storage → retrieval with pagination
   - Filtering (severity, namespace, status)
   - Aggregation queries (stats, top alerts)

5. **Error Scenarios**
   - LLM timeout/failure → graceful degradation
   - Database connection failure → error handling
   - Publishing target unreachable → retry logic
   - Invalid input → proper error responses

### Out of Scope

- UI tests (no dashboard yet)
- Performance/load tests (covered in TN-109)
- Chaos engineering (future work)
- Multi-cluster scenarios

---

## Functional Requirements

### FR-1: Alert Ingestion E2E

**Description**: Complete alert ingestion from webhook to storage

**Acceptance Criteria**:
- AC1: POST /webhook with Alertmanager format stores alert in DB
- AC2: Alert fingerprint is generated correctly (SHA-256)
- AC3: Alert is retrievable via GET /api/v2/history
- AC4: Duplicate alerts (same fingerprint) update existing record
- AC5: Invalid alerts return proper error responses (400/422)

**Test Scenarios**:
1. Happy path: Valid alert → storage → retrieval
2. Duplicate detection: Same alert twice → single DB record
3. Multiple alerts: Batch ingestion → all stored
4. Invalid format: Malformed JSON → 400 error
5. Missing required fields → 422 error

---

### FR-2: Classification Pipeline E2E

**Description**: Complete classification flow with LLM integration

**Acceptance Criteria**:
- AC1: Alert → LLM classification → enriched alert in DB
- AC2: Classification result cached in L1 (memory)
- AC3: Classification result cached in L2 (Redis)
- AC4: Cache hit skips LLM call (verified via metrics)
- AC5: LLM failure → rule-based fallback

**Test Scenarios**:
1. First classification: Alert → LLM API → enriched → cached
2. Cache hit L1: Same alert → memory cache → no LLM call
3. Cache hit L2: Same alert (new pod) → Redis cache → no LLM call
4. LLM timeout: Alert → timeout → fallback classification
5. LLM unavailable: Alert → error → fallback classification

---

### FR-3: Publishing Fanout E2E

**Description**: Multi-target publishing with parallel fanout

**Acceptance Criteria**:
- AC1: Alert published to multiple targets in parallel
- AC2: Partial success (some targets fail) handled gracefully
- AC3: Failed targets retry with exponential backoff
- AC4: Publishing results recorded in alert history
- AC5: Target health checks prevent failed publishing

**Test Scenarios**:
1. Single target: Alert → Slack → success recorded
2. Multiple targets: Alert → Slack + PagerDuty + Rootly → all succeed
3. Partial failure: Alert → 2 succeed + 1 fails → partial success (207)
4. Retry logic: Target fails → retry → eventually succeeds
5. Circuit breaker: Target unhealthy → skipped

---

### FR-4: History Query E2E

**Description**: Alert history retrieval with filtering

**Acceptance Criteria**:
- AC1: Pagination works correctly (limit/offset)
- AC2: Filtering by severity returns correct alerts
- AC3: Filtering by namespace returns correct alerts
- AC4: Date range filtering works
- AC5: Aggregation queries return correct stats

**Test Scenarios**:
1. Pagination: Insert 50 alerts → query with limit=10 → 5 pages
2. Filter by severity: Insert mixed → query severity=critical → only critical
3. Filter by namespace: Insert mixed → query namespace=prod → only prod
4. Date range: Insert old + new → query last 1h → only new
5. Top alerts: Insert varied → query top 5 → most frequent

---

### FR-5: Error Handling E2E

**Description**: System behavior under error conditions

**Acceptance Criteria**:
- AC1: LLM timeout handled gracefully (no panic)
- AC2: Database unavailable → proper error response (503)
- AC3: Invalid input → clear error message (400/422)
- AC4: Publishing failure → logged, retried, doesn't block ingestion
- AC5: Metrics updated correctly for all error scenarios

**Test Scenarios**:
1. LLM timeout: Set timeout=1ms → alert still processed (fallback)
2. DB connection failure: Disconnect DB → 503 error
3. Invalid JSON: POST malformed → 400 with clear message
4. Publishing target down: Mock 500 error → retry → eventually fail
5. Rate limit hit: Flood alerts → rate limiting active

---

## Non-Functional Requirements

### NFR-1: Test Reliability

- **Requirement**: Tests must be deterministic (no flaky tests)
- **Acceptance**: 100% pass rate on 10 consecutive runs
- **Implementation**:
  - Use testcontainers (isolated environments)
  - Mock external dependencies (LLM, publishing targets)
  - Proper cleanup between tests

### NFR-2: Test Speed

- **Requirement**: E2E test suite completes in <5 minutes
- **Acceptance**: Full suite runs in <300 seconds
- **Implementation**:
  - Parallel test execution where possible
  - Use lightweight containers (Alpine)
  - Efficient teardown/setup

### NFR-3: Test Coverage

- **Requirement**: Cover 80%+ of critical user journeys
- **Acceptance**: All P0 flows have E2E tests
- **Metrics**:
  - Alert ingestion: 100% covered
  - Classification: 100% covered
  - Publishing: 100% covered
  - History queries: 80% covered

### NFR-4: Test Maintainability

- **Requirement**: Tests easy to understand and modify
- **Implementation**:
  - Clear test names (TestE2E_AlertIngestion_HappyPath)
  - Helper functions for common operations
  - Well-documented test fixtures
  - Page object pattern for API interactions

---

## Dependencies

### Upstream (Required)

- **TN-107**: Integration test infrastructure ✅ (85% complete)
  - Testcontainers for PostgreSQL/Redis
  - Mock LLM server
  - API test helpers
  - Test fixtures

### Downstream (Blocked By This)

- **TN-109**: Load testing (can use E2E scenarios)
- **Production Deployment**: Confidence in system behavior

---

## Test Infrastructure

### Required Components

1. **Testcontainers**
   - PostgreSQL 16 (for alert storage)
   - Redis 7 (for caching)
   - Already implemented in TN-107 ✅

2. **Mock Services**
   - Mock LLM Server (configurable responses)
   - Mock Publishing Targets (Slack, PagerDuty, Rootly webhooks)
   - Already implemented in TN-107 (Mock LLM) ✅

3. **Test Helpers**
   - HTTP client for API requests
   - Database seeding utilities
   - Assertion helpers
   - Already implemented in TN-107 ✅

4. **Test Fixtures**
   - Sample alerts (Alertmanager format)
   - Classification responses
   - Publishing target configurations

---

## Test Scenarios (Detailed)

### Scenario 1: Complete Alert Lifecycle

```
1. POST /webhook (Alertmanager format)
   - Alert: HighCPUUsage, severity=critical
   - Expected: 200 OK, fingerprint returned

2. Verify storage
   - Query: GET /api/v2/history?limit=1
   - Expected: Alert present with correct data

3. Verify classification (if enabled)
   - Check: Alert has classifications field
   - Expected: LLM result with severity, confidence

4. Verify publishing
   - Check: Alert published_to field
   - Expected: Array of targets (Slack, PagerDuty)

5. Verify retrieval
   - Query: GET /api/v2/history/{id}
   - Expected: Complete alert with all metadata
```

### Scenario 2: Classification Cache Hit

```
1. POST /webhook (Alert A, first time)
   - Expected: LLM API called (verify via mock LLM request count)

2. POST /webhook (Alert A, second time)
   - Expected: LLM API NOT called (cache hit)

3. Verify cache metrics
   - Query: GET /metrics
   - Expected: classification_l1_cache_hits_total incremented
```

### Scenario 3: Multi-Target Publishing

```
1. Setup: Create 3 publishing targets (Slack, PagerDuty, Rootly)
   - Mock HTTP servers for each

2. POST /webhook (Alert)
   - Expected: 200 OK

3. Verify parallel publishing
   - Check: All 3 mock servers received POST
   - Check: Requests were parallel (not sequential)

4. Verify publishing results
   - Query: GET /api/v2/history/{id}
   - Expected: published_to has 3 entries, all successful
```

### Scenario 4: Graceful Degradation

```
1. Setup: Mock LLM returns 500 error

2. POST /webhook (Alert)
   - Expected: 200 OK (alert still processed)

3. Verify fallback classification
   - Query: GET /api/v2/history/{id}
   - Expected: Alert has classification (rule-based fallback)

4. Verify metrics
   - Query: GET /metrics
   - Expected: llm_errors_total incremented
```

---

## Success Metrics

### Quantitative

- **Test Count**: 20+ E2E scenarios
- **Coverage**: 80%+ of critical flows
- **Pass Rate**: 100% (no flaky tests)
- **Execution Time**: <5 minutes
- **Lines of Code**: 1,500+ LOC (tests + helpers)

### Qualitative

- **Confidence**: High confidence in production deployment
- **Documentation**: Clear test scenarios with expected behavior
- **Maintainability**: Easy to add new scenarios
- **Debugging**: Clear failure messages, easy to diagnose

---

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Flaky tests | High | Medium | Use deterministic fixtures, proper cleanup |
| Slow execution | Medium | Low | Parallel execution, lightweight containers |
| Complex setup | Medium | Low | Reuse TN-107 infrastructure, helper functions |
| Insufficient coverage | High | Low | Plan scenarios upfront, track coverage |

---

## Timeline

| Phase | Duration | Description |
|-------|----------|-------------|
| **Phase 1** | 30min | Requirements & Design |
| **Phase 2** | 1h | E2E infrastructure setup (mock publishing) |
| **Phase 3** | 3-4h | Implement 20+ E2E scenarios |
| **Phase 4** | 1h | CI/CD integration, documentation |
| **Total** | 6-8h | Complete E2E test suite |

---

## Acceptance Criteria (Task Complete)

- [ ] 20+ E2E test scenarios implemented
- [ ] All tests passing (100% pass rate)
- [ ] Test execution time <5 minutes
- [ ] 80%+ coverage of critical flows
- [ ] CI/CD integration (GitHub Actions workflow)
- [ ] Documentation complete (README with scenarios)
- [ ] No flaky tests (10 consecutive runs pass)
- [ ] Mock publishing targets implemented

---

## References

- **TN-107**: Integration test infrastructure
- **TN-109**: Load testing (uses E2E scenarios)
- **Testcontainers**: https://golang.testcontainers.org/
- **Go testing**: https://pkg.go.dev/testing
