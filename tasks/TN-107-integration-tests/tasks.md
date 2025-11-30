# TN-107: Integration Tests - Implementation Tasks

**Task ID**: TN-107
**Target Quality**: 150%
**Estimated Effort**: 9 hours
**Status**: IN PROGRESS
**Started**: 2025-11-30

---

## Phase 1: Infrastructure Setup (2h) - IN PROGRESS

### Setup Testcontainers
- [ ] Add testcontainers-go dependency to go.mod
- [ ] Create `test/integration/infra.go` (400 LOC)
  - [ ] `SetupTestInfrastructure()` - Start PostgreSQL + Redis containers
  - [ ] `Teardown()` - Stop containers
  - [ ] `ResetDatabase()` - Clean state between tests
- [ ] Create `test/integration/helpers.go` (300 LOC)
  - [ ] `APITestHelper` struct
  - [ ] `MakeRequest()` - HTTP client wrapper
  - [ ] `AssertResponse()` - Response validation
  - [ ] `GetAlertFromDB()` - Database query helper
  - [ ] `SeedTestData()` - Insert test data

### Mock Servers
- [ ] Create `test/integration/mock_llm.go` (200 LOC)
  - [ ] `MockLLMServer` with configurable responses
  - [ ] `SetResponse()`, `SetError()`, `SetLatency()`
  - [ ] Request tracking for assertions
- [ ] Create `test/integration/mock_k8s.go` (150 LOC)
  - [ ] Mock Kubernetes API for target discovery
  - [ ] Secret listing with labels
  - [ ] Health check endpoints

### Test Fixtures
- [ ] Create `test/fixtures/alerts.json` (200 lines)
  - [ ] Alertmanager webhook payload
  - [ ] Prometheus v1/v2 payloads
  - [ ] Generic webhook payloads
- [ ] Create `test/fixtures/silences.json` (150 lines)
  - [ ] Valid silence examples
  - [ ] Edge cases (expired, invalid matchers)
- [ ] Create `test/fixtures/config.yaml` (100 lines)
  - [ ] Minimal test configuration
  - [ ] Full configuration
  - [ ] Invalid configurations

**Phase 1 Deliverables**: ~1,500 LOC infrastructure + fixtures

---

## Phase 2: API Endpoint Tests (2h)

### Webhook Tests
- [ ] `test/integration/api_webhook_test.go` (350 LOC)
  - [ ] Test: POST /webhook with Alertmanager payload
  - [ ] Test: POST /webhook with Prometheus v1/v2
  - [ ] Test: POST /webhook/proxy with classification
  - [ ] Test: Invalid payloads (400 responses)
  - [ ] Test: Request ID propagation
  - [ ] Test: Metrics recording

### Classification Tests
- [ ] `test/integration/api_classification_test.go` (250 LOC)
  - [ ] Test: POST /classification/classify with cache hit
  - [ ] Test: POST /classification/classify with cache miss (LLM call)
  - [ ] Test: force=true bypasses cache
  - [ ] Test: LLM timeout → rule-based fallback
  - [ ] Test: GET /classification/stats

### History Tests
- [ ] `test/integration/api_history_test.go` (400 LOC)
  - [ ] Test: GET /history with pagination
  - [ ] Test: GET /history with filtering (status, severity, namespace)
  - [ ] Test: GET /history/{fingerprint} timeline
  - [ ] Test: GET /history/top (top firing alerts)
  - [ ] Test: GET /history/flapping (flapping detection)
  - [ ] Test: GET /history/stats (aggregated statistics)

### Configuration Tests
- [ ] `test/integration/api_config_test.go` (300 LOC)
  - [ ] Test: GET /config exports configuration
  - [ ] Test: POST /config updates configuration
  - [ ] Test: POST /config with invalid config (validation)
  - [ ] Test: POST /config/rollback to previous version
  - [ ] Test: Hot reload triggered on update

### Silence/Inhibition Tests
- [ ] `test/integration/api_silence_test.go` (350 LOC)
  - [ ] Test: POST /silences creates silence
  - [ ] Test: GET /silences lists with filters
  - [ ] Test: PUT /silences/{id} updates silence
  - [ ] Test: DELETE /silences/{id} removes silence
  - [ ] Test: POST /silences/check validates alert
- [ ] `test/integration/api_inhibition_test.go` (250 LOC)
  - [ ] Test: GET /inhibition/rules lists rules
  - [ ] Test: GET /inhibition/status shows relationships
  - [ ] Test: POST /inhibition/check validates alert

**Phase 2 Deliverables**: ~1,900 LOC API tests

---

## Phase 3: Database Integration Tests (1.5h)

### Migration Tests
- [ ] `test/integration/database_migrations_test.go` (250 LOC)
  - [ ] Test: All migrations apply successfully
  - [ ] Test: Migration rollback works
  - [ ] Test: Idempotent migrations (re-run safe)
  - [ ] Test: Schema version tracking

### CRUD Tests
- [ ] `test/integration/database_crud_test.go` (400 LOC)
  - [ ] Test: Insert alert → retrieve by fingerprint
  - [ ] Test: Update alert status
  - [ ] Test: Delete alert (soft delete)
  - [ ] Test: Bulk insert (1000 alerts)
  - [ ] Test: Concurrent inserts (thread-safe)

### Transaction Tests
- [ ] `test/integration/database_tx_test.go` (300 LOC)
  - [ ] Test: Transaction commit persists data
  - [ ] Test: Transaction rollback undoes changes
  - [ ] Test: Nested transactions
  - [ ] Test: Deadlock detection & recovery

### Query Performance Tests
- [ ] `test/integration/database_perf_test.go` (350 LOC)
  - [ ] Test: Index usage verification (EXPLAIN ANALYZE)
  - [ ] Test: Query performance <100ms (simple)
  - [ ] Test: Query performance <500ms (complex)
  - [ ] Test: Connection pool under load

**Phase 3 Deliverables**: ~1,300 LOC database tests

---

## Phase 4: Cache Integration Tests (1h)

### Cache Behavior Tests
- [ ] `test/integration/cache_behavior_test.go` (400 LOC)
  - [ ] Test: L1 cache hit (in-memory)
  - [ ] Test: L2 cache hit (Redis)
  - [ ] Test: Cache miss → database fallback
  - [ ] Test: Cache invalidation on update
  - [ ] Test: TTL expiration (wait for expiry)
  - [ ] Test: Cache warming on startup

### Failure Scenarios
- [ ] `test/integration/cache_failure_test.go` (300 LOC)
  - [ ] Test: Redis unavailable → L1 only mode
  - [ ] Test: Redis reconnection → L2 resumes
  - [ ] Test: Memory pressure → L1 eviction
  - [ ] Test: Concurrent cache access (race-free)

**Phase 4 Deliverables**: ~700 LOC cache tests

---

## Phase 5: Publishing Integration Tests (1.5h)

### Publishing Flow Tests
- [ ] `test/integration/publishing_flow_test.go` (500 LOC)
  - [ ] Test: Alert → Classification → Filtering → Publishing
  - [ ] Test: Target discovery from K8s Secrets
  - [ ] Test: Queue submission → processing → completion
  - [ ] Test: Failed publishing → DLQ storage
  - [ ] Test: DLQ replay → retry → success
  - [ ] Test: Parallel publishing to 3+ targets
  - [ ] Test: Health check skip unhealthy targets

### Target Management Tests
- [ ] `test/integration/publishing_targets_test.go` (300 LOC)
  - [ ] Test: GET /publishing/targets lists all
  - [ ] Test: POST /publishing/targets/refresh discovers new
  - [ ] Test: Target health monitoring
  - [ ] Test: Rate limiting (1 refresh/min)

**Phase 5 Deliverables**: ~800 LOC publishing tests

---

## Phase 6: Error Handling & Recovery Tests (1h)

### Component Failure Tests
- [ ] `test/integration/errors_component_test.go` (400 LOC)
  - [ ] Test: PostgreSQL connection lost → 503 response
  - [ ] Test: Redis connection lost → degraded mode
  - [ ] Test: LLM timeout → rule-based fallback
  - [ ] Test: Publishing target unreachable → DLQ
  - [ ] Test: Invalid configuration → rollback

### Graceful Degradation Tests
- [ ] `test/integration/errors_degradation_test.go` (300 LOC)
  - [ ] Test: All components unavailable → metrics-only mode
  - [ ] Test: Partial failures → continue processing
  - [ ] Test: Recovery after component restart
  - [ ] Test: Circuit breaker opens on repeated failures

**Phase 6 Deliverables**: ~700 LOC error tests

---

## Phase 7: Documentation & CI/CD (1h)

### Documentation
- [ ] Create `test/integration/README.md` (400 LOC)
  - [ ] Quick start guide
  - [ ] Running tests locally
  - [ ] Troubleshooting
  - [ ] Test organization
  - [ ] Adding new tests

### CI/CD Integration
- [ ] Create `.github/workflows/integration-tests.yml` (150 LOC)
  - [ ] PostgreSQL service
  - [ ] Redis service
  - [ ] Go test execution
  - [ ] Coverage reporting
- [ ] Update `Makefile` (50 LOC)
  - [ ] `make test-integration` target
  - [ ] `make test-infra-up` target
  - [ ] `make test-infra-down` target

**Phase 7 Deliverables**: ~600 LOC docs + CI config

---

## Summary

**Total Estimated LOC**: ~7,400 lines
- Infrastructure: 1,500 LOC
- API tests: 1,900 LOC
- Database tests: 1,300 LOC
- Cache tests: 700 LOC
- Publishing tests: 800 LOC
- Error tests: 700 LOC
- Documentation: 600 LOC

**Total Estimated Time**: 9 hours

**Target Quality**: 150%
- 60+ integration tests (200% of baseline 30)
- 85%+ coverage (121% of baseline 70%)
- <3min execution (167% better than 5min target)

---

**Next**: Start Phase 1 (Infrastructure Setup)
