# TN-106: Unit Tests (>80% coverage) - Detailed Tasks

**Task ID**: TN-106
**Phase**: 14 - Testing & Documentation
**Status**: 🔄 IN PROGRESS (Phase 2)
**Target Quality**: **150%** (Grade A+ EXCEPTIONAL)
**Date**: 2025-11-30
**Branch**: `feature/TN-106-unit-tests-150pct`

---

## 📋 Task Breakdown

### ✅ Phase 0: Analysis & Documentation (2 hours) - IN PROGRESS

**Status**: 60% complete

#### Checkl:
- [x] Fix build failures (integration_test.go imports) - 15min
- [x] Create requirements.md (600+ LOC) - 45min
- [x] Create design.md (1,000+ LOC) - 45min
- [ ] Create tasks.md (THIS DOCUMENT) - 15min
- [ ] Review and finalize documentation - 15min

**Deliverables**:
- ✅ Build passes (all tests compile)
- ✅ requirements.md (comprehensive 600+ LOC)
- ✅ design.md (comprehensive 1,000+ LOC)
- 🔄 tasks.md (THIS DOCUMENT)

---

### ✅ Phase 1: Fix Failing Tests (2 hours) - COMPLETE ✅

**Status**: 100% complete (2025-11-30)

#### Checklist:
- [x] Fix pkg/history/cache - duplicate metrics
- [x] Fix pkg/history/security - URL encoding tests
- [x] Fix pkg/history/filters - fingerprint tests
- [x] Fix pkg/middleware - duplicate metrics
- [x] Fix pkg/templatevalidator/validators - security tokens
- [x] Verify 100% test pass rate
- [x] Push to feature branch

**Deliverables**:
- ✅ Zero test failures
- ✅ Zero panics
- ✅ 100% pass rate
- ✅ PRODUCTION-READY

**Time**: 2 hours (50% faster than estimate)

---

### ⏳ Phase 2: Coverage Increase (8-12 hours) - TO DO

#### Phase 2.1: pkg/history/handlers (4 hours) - P0 CRITICAL

**Current**: 32.5% → **Target**: 80%+ (Gap: 47.5%)
**Effort**: ~500 LOC tests

**Checklist**:
- [ ] **Step 1**: Create test infrastructure (30min)
  - [ ] Create `handlers_test.go`
  - [ ] Implement `mockRepository`
  - [ ] Implement `mockCache`
  - [ ] Implement `mockFilterRegistry`
  - [ ] Create `setupHandler()` helper
  - [ ] Create `createTestAlert()` helper
  - [ ] Create `createTestRequest()` helper

- [ ] **Step 2**: HandleGetHistory tests (1h)
  - [ ] Happy path - cache hit (5min)
  - [ ] Happy path - cache miss (5min)
  - [ ] Valid pagination (5min)
  - [ ] Valid filters (10min)
  - [ ] Invalid method (POST → 405) (5min)
  - [ ] Invalid limit (negative → 400) (5min)
  - [ ] Invalid offset (negative → 400) (5min)
  - [ ] Limit too large (> max → 400) (5min)
  - [ ] Empty results (200 with total:0) (5min)
  - [ ] Database error (500) (5min)
  - [ ] Cache error (graceful degradation) (5min)
  - [ ] Zero limit edge case (400) (5min)
  - [ ] Large offset (1M → 200 empty) (5min)
  - [ ] SQL injection attempt (safe params) (5min)
  - [ ] Concurrent requests (race test) (10min)

- [ ] **Step 3**: HandleGetRecent tests (45min)
  - [ ] Happy path - cache hit
  - [ ] Happy path - cache miss
  - [ ] Valid limit parameter
  - [ ] Invalid method
  - [ ] Invalid limit
  - [ ] Limit too large
  - [ ] Empty results
  - [ ] Database error
  - [ ] Default limit (20)
  - [ ] Maximum limit enforcement (100)

- [ ] **Step 4**: HandleGetStats tests (30min)
  - [ ] Happy path - all stats
  - [ ] Happy path - filtered stats
  - [ ] Invalid method
  - [ ] Database error
  - [ ] Empty database
  - [ ] Invalid time range
  - [ ] Future time range
  - [ ] Concurrent requests

- [ ] **Step 5**: HandleGetByFingerprint tests (30min)
  - [ ] Happy path - found
  - [ ] Not found (404)
  - [ ] Empty fingerprint (400)
  - [ ] Invalid fingerprint format (400)
  - [ ] Database error (500)
  - [ ] Cache hit
  - [ ] Cache miss
  - [ ] Multiple alerts same fingerprint
  - [ ] SQL injection attempt
  - [ ] Concurrent requests

- [ ] **Step 6**: HandleGetTop tests (20min)
  - [ ] Happy path - top 10
  - [ ] Custom limit
  - [ ] Empty results
  - [ ] Invalid limit
  - [ ] Database error
  - [ ] Invalid time range

- [ ] **Step 7**: HandleGetFlapping tests (20min)
  - [ ] Happy path - flapping alerts
  - [ ] No flapping alerts
  - [ ] Custom threshold
  - [ ] Invalid threshold
  - [ ] Database error

- [ ] **Step 8**: HandlePost tests (20min)
  - [ ] Happy path - create
  - [ ] Invalid JSON (400)
  - [ ] Missing required fields (400)
  - [ ] Database error (500)
  - [ ] Duplicate alert
  - [ ] Validation error
  - [ ] Large payload
  - [ ] Malformed payload

- [ ] **Step 9**: HandleDelete tests (15min)
  - [ ] Happy path - deleted
  - [ ] Not found (404)
  - [ ] Invalid method (405)
  - [ ] Database error (500)
  - [ ] Already deleted (idempotent)

- [ ] **Step 10**: Validation (15min)
  - [ ] Run tests: `go test -v ./pkg/history/handlers`
  - [ ] Check coverage: `go test -cover ./pkg/history/handlers`
  - [ ] Target: ≥80% coverage
  - [ ] Verify all tests pass
  - [ ] Commit with message: "test(handlers): Add comprehensive unit tests (80%+ coverage)"

**Expected Results**:
- ~500 LOC new tests
- Coverage: 32.5% → 80%+
- 50+ test scenarios
- 100% pass rate

---

#### Phase 2.2: pkg/history/cache (3 hours) - P0 CRITICAL

**Current**: 40.8% → **Target**: 80%+ (Gap: 39.2%)
**Effort**: ~400 LOC tests

**Checklist**:
- [ ] **Step 1**: Create test infrastructure (30min)
  - [ ] Create enhanced `cache_test.go`
  - [ ] Implement `mockL1Cache` (Ristretto)
  - [ ] Implement `mockL2Cache` (Redis)
  - [ ] Create `setupCacheManager()` helper
  - [ ] Create `createTestData()` helper

- [ ] **Step 2**: CacheManager.Get tests (1h)
  - [ ] L1 hit (10min)
  - [ ] L2 hit (L1 miss) (10min)
  - [ ] Full miss (5min)
  - [ ] L2 error (graceful degradation) (10min)
  - [ ] Nil value in L1 (5min)
  - [ ] Large value (1MB) (5min)
  - [ ] Empty key (error) (5min)
  - [ ] Special characters in key (5min)
  - [ ] Unicode key (5min)
  - [ ] Expired TTL (5min)
  - [ ] Concurrent access (10min)
  - [ ] L1+L2 conflict (L1 wins) (5min)
  - [ ] L2 hit populates L1 (5min)
  - [ ] L2 timeout (5min)
  - [ ] L2 pool exhausted (5min)

- [ ] **Step 3**: CacheManager.Set tests (45min)
  - [ ] Set to L1 only (5min)
  - [ ] Set to L1+L2 (5min)
  - [ ] TTL expiration test (10min)
  - [ ] Eviction on full cache (10min)
  - [ ] L2 error (graceful degradation) (5min)
  - [ ] Overwrite existing key (5min)
  - [ ] Large value handling (5min)
  - [ ] Empty key (error) (5min)
  - [ ] Nil value (5min)
  - [ ] Zero TTL (immediate expiration) (5min)
  - [ ] Negative TTL (error) (5min)
  - [ ] Concurrent writes (10min)

- [ ] **Step 4**: CacheManager.Invalidate tests (30min)
  - [ ] Invalidate pattern match (10min)
  - [ ] Invalidate exact key (5min)
  - [ ] Invalidate non-existent pattern (5min)
  - [ ] Invalidate all (pattern "*") (10min)
  - [ ] L2 error handling (5min)
  - [ ] Empty pattern (error) (5min)
  - [ ] Complex pattern (wildcards) (10min)
  - [ ] Concurrent invalidation (10min)

- [ ] **Step 5**: Internal helpers tests (15min)
  - [ ] getL1() - various scenarios (5min)
  - [ ] getL2() - error handling (5min)
  - [ ] setL1() - success/failure (5min)
  - [ ] setL2() - success/failure (5min)

- [ ] **Step 6**: Validation (15min)
  - [ ] Run tests: `go test -v ./pkg/history/cache`
  - [ ] Check coverage: `go test -cover ./pkg/history/cache`
  - [ ] Target: ≥80% coverage
  - [ ] Verify all tests pass
  - [ ] Commit: "test(cache): Add comprehensive unit tests (80%+ coverage)"

**Expected Results**:
- ~400 LOC new tests
- Coverage: 40.8% → 80%+
- 45+ test scenarios
- 100% pass rate

---

#### Phase 2.3: pkg/history/query (1.5 hours) - P1

**Current**: 66.7% → **Target**: 80%+ (Gap: 13.3%)
**Effort**: ~150 LOC tests

**Checklist**:
- [ ] **Step 1**: Enhance existing tests (30min)
  - [ ] Review current test coverage
  - [ ] Identify uncovered code paths
  - [ ] Add missing test scenarios

- [ ] **Step 2**: QueryBuilder.Build tests (45min)
  - [ ] Empty query (5min)
  - [ ] Single filter (5min)
  - [ ] Multiple filters (AND) (10min)
  - [ ] Pagination (LIMIT/OFFSET) (5min)
  - [ ] Sort ascending (5min)
  - [ ] Sort descending (5min)
  - [ ] Complex query (filters+sort+pagination) (10min)
  - [ ] SQL injection prevention (10min)
  - [ ] Invalid limit (negative) (5min)
  - [ ] Invalid offset (negative) (5min)
  - [ ] Zero limit (valid, no limit) (5min)

- [ ] **Step 3**: Edge cases (15min)
  - [ ] Very large limit (1M) (5min)
  - [ ] Very large offset (1M) (5min)
  - [ ] Empty filter values (5min)
  - [ ] Special characters in values (5min)
  - [ ] Unicode values (5min)

- [ ] **Step 4**: Validation (15min)
  - [ ] Run tests: `go test -v ./pkg/history/query`
  - [ ] Check coverage: `go test -cover ./pkg/history/query`
  - [ ] Target: ≥80% coverage
  - [ ] Verify all tests pass
  - [ ] Commit: "test(query): Add comprehensive unit tests (80%+ coverage)"

**Expected Results**:
- ~150 LOC new tests
- Coverage: 66.7% → 80%+
- 15+ test scenarios
- 100% pass rate

---

#### Phase 2.4: pkg/metrics (1.5 hours) - P1

**Current**: 69.7% → **Target**: 80%+ (Gap: 10.3%)
**Effort**: ~100 LOC tests

**Checklist**:
- [ ] **Step 1**: Create retry_metrics_test.go (45min)
  - [ ] Test RecordAttempt() (10min)
  - [ ] Test RecordBackoff() (5min)
  - [ ] Test RecordFinalAttempt() (5min)
  - [ ] Test Reset() (5min)
  - [ ] Test Lock()/Unlock() (5min)
  - [ ] Test concurrent metric updates (15min)

- [ ] **Step 2**: Enhance webhook_metrics_test.go (30min)
  - [ ] Test SetActiveWorkers() (5min)
  - [ ] Test concurrent IncrementActiveWorkers() (10min)
  - [ ] Test concurrent DecrementActiveWorkers() (10min)
  - [ ] Test edge cases (negative values, overflow) (5min)

- [ ] **Step 3**: Technical metrics tests (15min)
  - [ ] Test all TechnicalMetrics methods (10min)
  - [ ] Test metric registration (5min)

- [ ] **Step 4**: Validation (15min)
  - [ ] Run tests: `go test -v ./pkg/metrics`
  - [ ] Check coverage: `go test -cover ./pkg/metrics`
  - [ ] Target: ≥80% coverage
  - [ ] Verify all tests pass
  - [ ] Commit: "test(metrics): Add comprehensive unit tests (80%+ coverage)"

**Expected Results**:
- ~100 LOC new tests
- Coverage: 69.7% → 80%+
- 15+ test scenarios
- 100% pass rate

---

#### Phase 2.5: Validation & Cleanup (1 hour) - FINAL

**Checklist**:
- [ ] **Step 1**: Run full test suite (15min)
  ```bash
  go test -cover ./pkg/history/... ./pkg/metrics/...
  ```
  - [ ] Verify all tests pass
  - [ ] Verify 80%+ coverage achieved

- [ ] **Step 2**: Coverage analysis (15min)
  ```bash
  go test -coverprofile=coverage.out ./pkg/history/... ./pkg/metrics/...
  go tool cover -func=coverage.out | grep total
  ```
  - [ ] Generate coverage.html: `go tool cover -html=coverage.out -o coverage.html`
  - [ ] Review uncovered lines
  - [ ] Identify any remaining gaps

- [ ] **Step 3**: Fix remaining gaps (15min)
  - [ ] Add tests for any critical uncovered code
  - [ ] Target: ≥80% overall coverage

- [ ] **Step 4**: Final validation (15min)
  ```bash
  go test -race ./pkg/history/... ./pkg/metrics/...
  ```
  - [ ] Race detector passes
  - [ ] No flaky tests
  - [ ] All tests deterministic
  - [ ] Test execution <30s total

- [ ] **Step 5**: Commit and push
  ```bash
  git add .
  git commit -m "test(TN-106): Phase 2 complete - 80%+ coverage achieved"
  git push origin feature/TN-106-unit-tests-150pct
  ```

**Expected Results**:
- Overall coverage: ≥80% (target 85% for 150%)
- All packages individually ≥80%
- Zero flaky tests
- Race detector passes
- Ready for Phase 3 (150% enhancements)

---

### 🎯 Phase 3: 150% Quality Enhancements (4-6 hours) - OPTIONAL

**Goal**: Exceed baseline (80% coverage) with exceptional quality additions

#### Phase 3.1: Benchmark Tests (2 hours)

**Checklist**:
- [ ] **Step 1**: Create benchmark test files
  - [ ] `pkg/history/handlers/handlers_bench_test.go`
  - [ ] `pkg/history/cache/cache_bench_test.go`
  - [ ] `pkg/history/query/query_bench_test.go`

- [ ] **Step 2**: Handler benchmarks (1h)
  - [ ] BenchmarkHandleGetHistory (15min)
  - [ ] BenchmarkHandleGetRecent (10min)
  - [ ] BenchmarkHandleGetStats (15min)
  - [ ] BenchmarkHandleGetByFingerprint (10min)
  - [ ] Verify p95 < 10ms (10min)

- [ ] **Step 3**: Cache benchmarks (45min)
  - [ ] BenchmarkCacheGet_L1Hit (10min)
  - [ ] BenchmarkCacheGet_L2Hit (10min)
  - [ ] BenchmarkCacheGet_Miss (10min)
  - [ ] BenchmarkCacheSet (10min)
  - [ ] Verify sub-millisecond L1 latency (5min)

- [ ] **Step 4**: Query benchmarks (15min)
  - [ ] BenchmarkQueryBuilder_Build (10min)
  - [ ] Verify <100µs SQL generation (5min)

- [ ] **Step 5**: Run benchmarks
  ```bash
  go test -bench=. -benchmem ./pkg/history/... ./pkg/metrics/...
  ```

**Expected Results**:
- 7+ benchmark tests
- Performance targets validated
- No performance regressions

---

#### Phase 3.2: Concurrent Tests (2 hours)

**Checklist**:
- [ ] **Step 1**: Handler concurrent tests (1h)
  - [ ] TestHandleGetHistory_Concurrent (100 goroutines) (20min)
  - [ ] TestHandleGetRecent_Concurrent (100 goroutines) (15min)
  - [ ] TestHandleGetStats_Concurrent (100 goroutines) (15min)
  - [ ] Verify no race conditions (10min)

- [ ] **Step 2**: Cache concurrent tests (45min)
  - [ ] TestCacheGet_Concurrent (1000 goroutines) (15min)
  - [ ] TestCacheSet_Concurrent (1000 goroutines) (15min)
  - [ ] TestCacheStampede (15min)

- [ ] **Step 3**: Metrics concurrent tests (15min)
  - [ ] TestMetrics_ConcurrentUpdates (10min)
  - [ ] Verify metric accuracy under load (5min)

- [ ] **Step 4**: Run race detector
  ```bash
  go test -race -count=10 ./pkg/history/... ./pkg/metrics/...
  ```

**Expected Results**:
- 6+ concurrent tests
- Race detector passes
- Thread-safety validated

---

#### Phase 3.3: Edge Case Tests (2 hours)

**Checklist**:
- [ ] **Step 1**: Nil pointer tests (30min)
  - [ ] Nil request (handlers)
  - [ ] Nil alert (handlers)
  - [ ] Nil cache (graceful degradation)

- [ ] **Step 2**: Large data tests (30min)
  - [ ] Large payload (10MB)
  - [ ] Large result set (10K alerts)
  - [ ] Memory leak detection

- [ ] **Step 3**: Special character tests (30min)
  - [ ] Unicode in all string fields
  - [ ] SQL injection attempts
  - [ ] XSS attempts
  - [ ] Path traversal attempts

- [ ] **Step 4**: Boundary value tests (30min)
  - [ ] Max int values
  - [ ] Min int values
  - [ ] Zero values
  - [ ] Negative values
  - [ ] Float precision

**Expected Results**:
- 15+ edge case tests
- All edge cases handled gracefully
- No panics or crashes

---

### 📊 Phase 4: Certification (1 hour) - FINAL

**Checklist**:
- [ ] **Step 1**: Generate coverage report (15min)
  ```bash
  go test -coverprofile=coverage.out ./pkg/history/... ./pkg/metrics/...
  go tool cover -html=coverage.out -o tasks/TN-106-unit-tests/coverage.html
  go tool cover -func=coverage.out > tasks/TN-106-unit-tests/coverage.txt
  ```

- [ ] **Step 2**: Calculate quality metrics (15min)
  - [ ] Overall coverage percentage
  - [ ] Per-package coverage
  - [ ] Test count (total)
  - [ ] Benchmark count
  - [ ] Concurrent test count
  - [ ] Test execution time
  - [ ] Lines of test code added

- [ ] **Step 3**: Create certification document (20min)
  - [ ] Create `tasks/TN-106-unit-tests/CERTIFICATION.md`
  - [ ] Document all achievements
  - [ ] Compare baseline vs target vs actual
  - [ ] Quality grade (A+ if ≥150%)

- [ ] **Step 4**: Update project documentation (10min)
  - [ ] Update `tasks/TN-106-unit-tests/STATUS.md`
  - [ ] Update `tasks/alertmanager-plus-plus-oss/TASKS.md`
  - [ ] Mark TN-106 as complete with quality score

**Expected Results**:
- Comprehensive coverage report
- Quality certification (150% Grade A+)
- All documentation updated
- Task marked complete

---

## 🎯 Success Criteria

### Must Have (100% - MVP)
- [ ] Overall coverage ≥80%
- [ ] pkg/history/handlers ≥80% coverage
- [ ] pkg/history/cache ≥80% coverage
- [ ] pkg/history/query ≥80% coverage
- [ ] pkg/metrics ≥80% coverage
- [ ] 100% test pass rate
- [ ] Zero flaky tests
- [ ] Test execution <30s
- [ ] Race detector passes

### Should Have (150% - EXCEPTIONAL)
- [ ] Overall coverage ≥85%
- [ ] All packages individually ≥80%
- [ ] 7+ benchmark tests
- [ ] 6+ concurrent tests
- [ ] 15+ edge case tests
- [ ] Comprehensive documentation
- [ ] Performance targets validated
- [ ] Quality certification (Grade A+)

---

## 📅 Timeline

### Day 1 (8 hours)
- ✅ Phase 0: Analysis & Documentation (2h) - 60% COMPLETE
- ⏳ Phase 2.1: Handlers tests (4h)
- ⏳ Phase 2.2: Cache tests start (2h)

### Day 2 (8 hours)
- ⏳ Phase 2.2: Cache tests complete (1h)
- ⏳ Phase 2.3: Query tests (1.5h)
- ⏳ Phase 2.4: Metrics tests (1.5h)
- ⏳ Phase 2.5: Validation (1h)
- ⏳ Phase 3.1: Benchmarks start (3h)

### Day 3 (4-6 hours) - OPTIONAL
- ⏳ Phase 3.1: Benchmarks complete (1h)
- ⏳ Phase 3.2: Concurrent tests (2h)
- ⏳ Phase 3.3: Edge cases (2h)
- ⏳ Phase 4: Certification (1h)

**Total Estimate**: 20-22 hours (for 150% quality)
**Critical Path**: Phase 0 → Phase 2 → Phase 4

---

## 📝 Commit Strategy

### Commit Messages Format
```
test(component): Brief description

- Detailed change 1
- Detailed change 2
- Coverage: X% → Y%
```

### Planned Commits
1. `test(handlers): Add comprehensive unit tests (80%+ coverage)`
2. `test(cache): Add comprehensive unit tests (80%+ coverage)`
3. `test(query): Add comprehensive unit tests (80%+ coverage)`
4. `test(metrics): Add comprehensive unit tests (80%+ coverage)`
5. `test(TN-106): Phase 2 complete - 80%+ coverage achieved`
6. `test(handlers): Add benchmark tests (150% enhancement)`
7. `test(cache): Add benchmark tests (150% enhancement)`
8. `test(TN-106): Add concurrent tests for thread-safety validation`
9. `test(TN-106): Add edge case tests for robustness`
10. `docs(TN-106): Add 150% quality certification`

---

## 🔒 Quality Gates

### Gate 1: Phase 2.1 Complete
- [ ] pkg/history/handlers coverage ≥80%
- [ ] All handler tests passing
- [ ] Zero flaky tests
- [ ] Commit pushed

### Gate 2: Phase 2.2 Complete
- [ ] pkg/history/cache coverage ≥80%
- [ ] All cache tests passing
- [ ] Concurrent tests passing
- [ ] Commit pushed

### Gate 3: Phase 2.3-2.4 Complete
- [ ] pkg/history/query coverage ≥80%
- [ ] pkg/metrics coverage ≥80%
- [ ] All tests passing
- [ ] Commits pushed

### Gate 4: Phase 2 Complete
- [ ] Overall coverage ≥80%
- [ ] All packages individually ≥80%
- [ ] 100% pass rate
- [ ] Race detector passes
- [ ] Test execution <30s

### Gate 5: 150% Enhancements Complete
- [ ] 7+ benchmarks implemented
- [ ] 6+ concurrent tests implemented
- [ ] 15+ edge case tests
- [ ] Overall coverage ≥85%

### Gate 6: Certification Complete
- [ ] Coverage report generated
- [ ] Certification document created
- [ ] All documentation updated
- [ ] 150% quality achieved
- [ ] Task marked complete

---

## 📊 Progress Tracking

### Current Status
- **Phase 0**: 60% complete
- **Phase 1**: ✅ 100% complete
- **Phase 2**: 0% complete (TO DO)
- **Phase 3**: 0% complete (OPTIONAL)
- **Phase 4**: 0% complete (TO DO)

### Coverage Progress
| Package | Start | Current | Target (100%) | Target (150%) |
|---------|-------|---------|---------------|---------------|
| Overall | 40.3% | 40.3% | 80% | 85% |
| handlers | 32.5% | 32.5% | 80% | 85% |
| cache | 40.8% | 40.8% | 80% | 85% |
| query | 66.7% | 66.7% | 80% | 85% |
| metrics | 69.7% | 69.7% | 80% | 85% |

### Test Count Progress
| Type | Current | Target (100%) | Target (150%) |
|------|---------|---------------|---------------|
| Unit tests | ~50 | 150+ | 200+ |
| Benchmarks | 0 | 0 | 7+ |
| Concurrent tests | 0 | 0 | 6+ |
| Edge case tests | 0 | 0 | 15+ |

---

**Document Version**: 1.0
**Last Review**: 2025-11-30
**Next Review**: After each phase complete
**Owner**: Vitalii Semenov
