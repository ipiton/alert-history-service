# 🚀 ФАЗА A: План Улучшений
## Roadmap к Production-Ready (95%+)

**Дата начала**: 2025-11-07
**Целевая дата**: 2025-11-14 (1 неделя)
**Текущая готовность**: 92-95%
**Целевая готовность**: 97-98%

---

## 📋 WEEK 1: IMMEDIATE FIXES (Priority 1)

### 🔴 Task 1: Исправить 22 Failing Tests

**Timeline**: 2-3 дня
**Status**: 🔄 IN PROGRESS
**Priority**: CRITICAL

#### Категория 1: Silencing (3 tests)

1. **TestMultiMatcher_TenMatchers** ⚠️
   - **Файл**: `go-app/internal/core/silencing/matcher_test.go`
   - **Проблема**: Expected match (10 matchers), got no match
   - **Причина**: Вероятно, логика AND для множественных matchers
   - **Решение**: Проверить логику MatchesAll vs MatchesAny
   - **Усилия**: 1-2 часа

2. **TestMatchesAny_ContextCancelledDuringIteration** ⚠️
   - **Файл**: `go-app/internal/core/silencing/matcher_test.go`
   - **Проблема**: Expected ErrContextCancelled, got nil
   - **Причина**: Context cancellation не проверяется в цикле
   - **Решение**: Добавить ctx.Done() checks
   - **Усилия**: 1 час

3. **TestGetSilenceByID_InvalidUUID** ⚠️
   - **Файл**: `go-app/internal/infrastructure/silencing/postgres_silence_repository_test.go`
   - **Проблема**: Empty string UUID validation
   - **Причина**: Вероятно, missing error handling
   - **Решение**: Добавить early validation
   - **Усилия**: 30 минут

#### Категория 2: LLM (1 test)

4. **TestHTTPLLMClient_RetryLogic** ⚠️
   - **Файл**: `go-app/internal/infrastructure/llm/client_test.go`
   - **Проблема**: Retry logic assertion failure
   - **Причина**: Timing issue или mock server problem
   - **Решение**: Проверить retry delays и mock responses
   - **Усилия**: 1-2 часа

#### Категория 3: Migrations (8 tests)

5-12. **TestMigrationManager_*** (8 tests) ⚠️
   - **Файл**: `go-app/internal/infrastructure/migrations/manager_test.go`
   - **Проблема**: Database setup issues (testcontainers?)
   - **Причина**: Missing test database или неправильная конфигурация
   - **Решение**:
     - Option A: Skip integration tests (unit tests only)
     - Option B: Setup testcontainers properly
   - **Усилия**: 2-4 часа (Option A: 1 час, Option B: 4 часа)

**Total Effort**: 8-12 часов (1-1.5 дня)

---

### 🟡 Task 2: Увеличить Grouping Coverage до 80%+

**Timeline**: 1-2 дня
**Status**: ⏳ PENDING
**Priority**: HIGH
**Current**: 71.2%
**Target**: 80%+
**Gap**: +8.8%

#### Файлы для улучшения coverage

1. **parser.go** (207 LOC)
   - Добавить тесты для edge cases (empty config, invalid YAML)
   - **+5 tests** (1 час)

2. **validator.go** (271 LOC)
   - Тесты для всех validation rules
   - **+8 tests** (1.5 часа)

3. **manager_impl.go** (650+ LOC)
   - Error handling paths
   - Concurrent access scenarios
   - **+10 tests** (2 часа)

4. **timer_manager_impl.go** (840 LOC)
   - Timer cancellation edge cases
   - Graceful shutdown scenarios
   - **+8 tests** (1.5 часа)

5. **storage_manager.go** (380 LOC)
   - Redis failure scenarios
   - State recovery edge cases
   - **+5 tests** (1 час)

**Total**: ~36 новых тестов, **7 часов** работы

---

### 🟢 Task 3: Security Review

**Timeline**: 1-2 дня
**Status**: ⏳ PENDING
**Priority**: HIGH

#### Checklist

##### 3.1 Input Validation (4 часа)
- [ ] SQL injection prevention (prepared statements) ✅ Verify
- [ ] XSS prevention (HTML escaping) ⚠️ Check templates
- [ ] Path traversal prevention ⚠️ Check file operations
- [ ] YAML bomb prevention ⚠️ Check config parsing
- [ ] JSON injection prevention ✅ Using encoding/json

##### 3.2 Authentication & Authorization (2 часа)
- [ ] API authentication (если есть) ⚠️ Check endpoints
- [ ] RBAC implementation ⚠️ TN-130 API needs auth
- [ ] Session management ⚠️ Check cookies/JWT

##### 3.3 Secrets Management (2 часа)
- [ ] No hardcoded secrets ✅ Verify
- [ ] Environment variables usage ✅ Verify
- [ ] Redis password protection ✅ Verify
- [ ] PostgreSQL credentials ✅ Verify

##### 3.4 Error Handling (1 час)
- [ ] No sensitive data in errors ⚠️ Review error messages
- [ ] Proper logging (no passwords) ✅ Verify slog usage

##### 3.5 Rate Limiting (2 часа)
- [ ] API rate limiting ⚠️ Not implemented
- [ ] WebSocket rate limiting ⚠️ Check TN-136

**Total**: ~11 часов работы

---

### 🔵 Task 4: Integration Tests

**Timeline**: 1 день
**Status**: ⏳ PENDING
**Priority**: MEDIUM

#### Test Scenarios

##### 4.1 Redis Integration (2 часа)
- [ ] GroupStorage with Redis
- [ ] TimerStorage with Redis
- [ ] Cache with Redis
- [ ] State Manager with Redis
- [ ] Failover scenarios (Redis down)

##### 4.2 PostgreSQL Integration (2 часа)
- [ ] AlertStorage CRUD operations
- [ ] SilenceRepository CRUD operations
- [ ] Transaction rollback scenarios
- [ ] Connection pool exhaustion

##### 4.3 E2E Scenarios (4 часа)
- [ ] Alert grouping flow (ingestion → grouping → storage)
- [ ] Inhibition flow (source alert → inhibit target)
- [ ] Silencing flow (create → match → expire → cleanup)
- [ ] High availability (pod restart → state recovery)

**Total**: ~8 часов работы

---

## 📊 WEEK 2-3: OPTIONAL IMPROVEMENTS (Priority 2)

### 🟣 Task 5: Реализовать TN-130 Inhibition API

**Timeline**: 1-2 дня
**Status**: ⏳ DEFERRED
**Priority**: MEDIUM (optional)

#### Endpoints to implement

1. **GET /api/v2/inhibition/rules**
   - List all loaded inhibition rules
   - Query params: none
   - Response: []InhibitionRule
   - **2 часа**

2. **GET /api/v2/inhibition/status**
   - Get active inhibition relationships
   - Query params: filter by source/target
   - Response: []InhibitionStatus
   - **2 часа**

3. **POST /api/v2/inhibition/check**
   - Check if alert would be inhibited
   - Body: Alert JSON
   - Response: {inhibited: bool, by: []string}
   - **2 часа**

4. **OpenAPI Spec** (1 час)
5. **Tests** (2 часа)
6. **Integration** (1 час)

**Total**: ~10 часов работы

---

### 🟣 Task 6: Load Testing

**Timeline**: 2-3 дня
**Status**: ⏳ PENDING
**Priority**: MEDIUM

#### Test Scenarios (k6)

1. **Alert Ingestion** (4 часа)
   - 1K alerts/sec sustained
   - 5K alerts/sec peak
   - 10K alerts/sec spike

2. **Grouping Performance** (2 часа)
   - 100 concurrent groups
   - 1K concurrent groups
   - 10K concurrent groups

3. **Silence Matching** (2 часа)
   - 100 active silences
   - 1K active silences
   - Match performance

4. **Redis Load** (2 часа)
   - Connection pool stress
   - Memory usage
   - Failover recovery time

5. **PostgreSQL Load** (2 часа)
   - Query performance under load
   - Connection pool stress
   - Transaction throughput

**Total**: ~12 часов работы

---

### 🟣 Task 7: Performance Profiling

**Timeline**: 1-2 дня
**Status**: ⏳ PENDING
**Priority**: MEDIUM

#### Profiling Tasks

1. **CPU Profiling** (4 часа)
   - Identify hot paths
   - Optimize regex compilation
   - Optimize JSON marshaling
   - Reduce allocations

2. **Memory Profiling** (4 часа)
   - Identify memory leaks
   - Optimize cache sizes
   - Reduce GC pressure
   - Object pooling

3. **Goroutine Profiling** (2 часа)
   - Check goroutine leaks
   - Optimize worker pools
   - Review context cancellation

**Total**: ~10 часов работы

---

## 📈 PROGRESS TRACKING

### Week 1 (Immediate)

| Task | Timeline | Status | Progress |
|------|----------|--------|----------|
| 1. Fix Failing Tests | 2-3 дня | 🔄 IN PROGRESS | 0% |
| 2. Increase Coverage | 1-2 дня | ⏳ PENDING | 0% |
| 3. Security Review | 1-2 дня | ⏳ PENDING | 0% |
| 4. Integration Tests | 1 день | ⏳ PENDING | 0% |

**Total Week 1**: 5-8 дней (с параллельной работой: 3-5 дней)

### Week 2-3 (Optional)

| Task | Timeline | Status | Progress |
|------|----------|--------|----------|
| 5. TN-130 API | 1-2 дня | ⏳ DEFERRED | 0% |
| 6. Load Testing | 2-3 дня | ⏳ PENDING | 0% |
| 7. Performance Profiling | 1-2 дня | ⏳ PENDING | 0% |

**Total Week 2-3**: 4-7 дней

---

## 🎯 SUCCESS CRITERIA

### Week 1 Completion (97%+ Quality)

- ✅ 100% test pass rate (0 failing tests)
- ✅ 80%+ coverage для всех модулей
- ✅ Security audit complete (no HIGH/CRITICAL issues)
- ✅ Integration tests passing
- ✅ Build SUCCESS
- ✅ Ready for production deployment

### Full Completion (100% Quality)

- ✅ All Week 1 criteria
- ✅ TN-130 API implemented (optional)
- ✅ Load testing complete (10K+ alerts/sec)
- ✅ Performance profiling complete
- ✅ Documentation updated
- ✅ Grade A+ (98%+)

---

## 📊 ESTIMATED EFFORT

| Category | Hours | Days (8h) |
|----------|-------|-----------|
| **Week 1 (Immediate)** | 34-38h | 4-5 дней |
| **Week 2-3 (Optional)** | 32h | 4 дня |
| **TOTAL** | 66-70h | 8-9 дней |

**Рекомендация**: Сфокусироваться на Week 1 для production deployment, Week 2-3 опционально для 100% quality.

---

## 🚀 NEXT ACTIONS

1. ✅ Создан план улучшений
2. 🔄 **CURRENT**: Анализ failing tests
3. ⏳ Исправление Silencing tests (3)
4. ⏳ Исправление LLM test (1)
5. ⏳ Исправление Migration tests (8)
6. ⏳ Увеличение Grouping coverage
7. ⏳ Security review
8. ⏳ Integration tests

**Status**: 🔄 ACTIVE - Task 1 in progress

---

**Created**: 2025-11-07
**Last Updated**: 2025-11-07
**Owner**: Development Team
**Priority**: HIGH

