# 📋 ФАЗА A: Action Items - Конкретные Рекомендации

**Дата**: 2025-11-04
**Основано на**: [PHASE_A_COMPREHENSIVE_AUDIT_FINAL.md](./PHASE_A_COMPREHENSIVE_AUDIT_FINAL.md)
**Статус**: 🔴 **ТРЕБУЕТСЯ ДЕЙСТВИЕ**

---

## 🔴 CRITICAL (Блокеры Production Deployment)

### C-1: Увеличить Test Coverage до 80%+

**Приоритет**: 🔴 CRITICAL
**Усилия**: 2-3 дня
**Ответственный**: TBD
**Deadline**: TBD

#### Текущее состояние

| Модуль | Actual | Target | Gap | Tests Needed |
|--------|--------|--------|-----|--------------|
| Grouping | 71.2% | 80% | **-8.8%** | ~50 tests |
| Inhibition | 66% | 80% | **-14%** | ~30 tests |
| **Total** | **73%** | **80%** | **-7%** | **~80 tests** |

#### Детальный план

**Day 1: Grouping Module (50 tests)**

1. **parser.go** (добавить 15 тестов):
   - Edge cases: пустые файлы, invalid YAML
   - Error handling: file not found, parse errors
   - Validation: invalid durations, negative values
   - Regex: pre-compile failures

2. **keygen.go** (добавить 10 тестов):
   - Special chars: URL encoding edge cases
   - Hash collisions: test with similar labels
   - Empty/nil inputs
   - Concurrent access

3. **manager_impl.go** (добавить 15 тестов):
   - Concurrent AddAlert/RemoveAlert
   - Storage failures (Redis down)
   - Timer failures
   - Edge cases: nil alerts, empty groups

4. **storage_manager.go** (добавить 10 тестов):
   - Failover scenarios (Redis → Memory)
   - Health check edge cases
   - Concurrent reads/writes
   - TTL expiration

**Day 2: Inhibition Module (30 tests)**

5. **matcher_impl.go** (добавить 12 тестов):
   - Complex regex patterns
   - Multiple equal labels
   - Missing labels
   - Concurrent matching

6. **cache.go** (добавить 10 тестов):
   - LRU eviction logic
   - Concurrent add/remove
   - Redis fallback scenarios
   - Background cleanup edge cases

7. **parser.go** (добавить 8 тестов):
   - Invalid regex patterns
   - Duplicate rule names
   - Empty rules
   - Large configs (100+ rules)

**Day 3: Integration & Verification**

8. Запустить coverage: `go test ./... -coverprofile=coverage.out`
9. Проверить uncovered lines: `go tool cover -html=coverage.out`
10. Добавить недостающие тесты
11. Финальная проверка: coverage ≥ 80%

#### Acceptance Criteria

- [ ] Grouping module: coverage ≥ 80%
- [ ] Inhibition module: coverage ≥ 80%
- [ ] Silencing module: coverage ≥ 80% (already 98.2% ✅)
- [ ] Overall Phase A: coverage ≥ 80%
- [ ] All tests passing: 100%
- [ ] Zero linter errors

#### Команды

```bash
# Запуск тестов с coverage
cd go-app
go test ./internal/infrastructure/grouping/... -coverprofile=coverage_grouping.out
go test ./internal/infrastructure/inhibition/... -coverprofile=coverage_inhibition.out
go test ./internal/core/silencing/... -coverprofile=coverage_silencing.out

# Проверка coverage
go tool cover -func=coverage_grouping.out | grep total
go tool cover -func=coverage_inhibition.out | grep total
go tool cover -func=coverage_silencing.out | grep total

# HTML отчет
go tool cover -html=coverage_grouping.out -o coverage_grouping.html
open coverage_grouping.html
```

---

### C-2: Завершить Модуль 3 (Silencing System)

**Приоритет**: 🔴 CRITICAL
**Усилия**: 2-3 недели
**Ответственный**: TBD
**Deadline**: TBD

#### Текущее состояние

| Задача | Status | Priority | Effort |
|--------|--------|----------|--------|
| TN-131: Models | ✅ DONE | - | - |
| TN-132: Matcher | ❌ TODO | HIGH | 2 дня |
| TN-133: Storage | ❌ TODO | HIGH | 2 дня |
| TN-134: Manager | ❌ TODO | HIGH | 2 дня |
| TN-135: API | ❌ TODO | MEDIUM | 1 день |
| TN-136: UI | ❌ TODO | MEDIUM | 2 дня |

#### Детальный план (2-3 недели)

**Week 1: Core Components (TN-132, TN-133)**

**Day 1-2: TN-132 Silence Matcher Engine**

1. Создать `matcher.go`:
   - Interface: `SilenceMatcher`
   - Methods: `Matches(alert Alert, silence Silence) bool`
   - Support: =, !=, =~, !~ operators

2. Создать `matcher_impl.go`:
   - Implement label matching
   - Regex compilation (cache patterns)
   - Equal/NotEqual/Regex/NotRegex logic

3. Создать `matcher_test.go`:
   - 20+ unit tests
   - Benchmarks (target: <100µs per match)
   - Edge cases: nil inputs, missing labels

**Day 3-4: TN-133 Silence Storage**

1. Создать `storage.go`:
   - Interface: `SilenceStorage`
   - Methods: Create, Get, List, Update, Delete, Cleanup

2. Создать `postgres_silence_storage.go`:
   - Implement PostgreSQL storage
   - Use existing `silences` table (migration 20251104120000)
   - JSONB queries для matchers

3. Создать `storage_test.go`:
   - 15+ unit tests
   - Integration tests с testcontainers
   - Query performance tests

**Day 5: Integration TN-132 + TN-133**

- Wire Matcher → Storage
- Integration tests
- Performance verification

**Week 2: Manager & API (TN-134, TN-135)**

**Day 6-7: TN-134 Silence Manager Service**

1. Создать `manager.go`:
   - Interface: `SilenceManager`
   - Methods: CreateSilence, GetSilence, ListSilences, DeleteSilence
   - Lifecycle: Pending → Active → Expired

2. Создать `manager_impl.go`:
   - Implement manager logic
   - Background GC (каждые 5 минут)
   - Metrics: active_silences, expired_silences

3. Создать `manager_test.go`:
   - 15+ unit tests
   - GC tests
   - Concurrent access tests

**Day 8: TN-135 Silence API Endpoints**

1. Создать `handlers/silence_handlers.go`:
   - POST /api/v2/silences - create
   - GET /api/v2/silences - list
   - GET /api/v2/silences/{id} - get
   - DELETE /api/v2/silences/{id} - delete

2. Alertmanager API compatibility:
   - Same request/response format
   - Status codes: 200, 201, 400, 404, 500

3. Integration в main.go:
   - Register handlers
   - Wire SilenceManager

**Day 9: Testing & Documentation**

- E2E tests для API
- OpenAPI spec (Swagger)
- Update README.md

**Week 3: UI & Final Integration (TN-136)**

**Day 10-11: TN-136 Silence UI Components**

1. Dashboard widget:
   - List active silences
   - Silence count badge
   - Quick actions (delete)

2. Create silence form:
   - Label matchers editor
   - Time range picker
   - Preview matched alerts

3. Silence details page:
   - View silence info
   - See matched alerts
   - Edit/Delete actions

**Day 12-14: Final Integration & Testing**

- Integration все компоненты Модуля 3
- E2E tests (UI → API → Manager → Storage)
- Performance tests
- Documentation update
- Final review

#### Acceptance Criteria

- [ ] TN-132: Matcher реализован, 20+ tests, <100µs performance
- [ ] TN-133: Storage реализован, 15+ tests, PostgreSQL working
- [ ] TN-134: Manager реализован, 15+ tests, GC working
- [ ] TN-135: API реализовано, Alertmanager compatible, 10+ tests
- [ ] TN-136: UI реализовано, responsive, user-friendly
- [ ] Integration: Все компоненты работают вместе
- [ ] Test coverage: ≥80% для Модуля 3
- [ ] Documentation: README, requirements, design, tasks
- [ ] Git: Committed, pushed, PR created

---

### C-3: Обновить Документацию с Реальными Метриками

**Приоритет**: 🔴 CRITICAL
**Усилия**: 4-6 часов
**Ответственный**: TBD
**Deadline**: TBD

#### Текущие проблемы

| Метрика | Заявлено | Фактически | Discrepancy |
|---------|----------|------------|-------------|
| **LOC (Модуль 1)** | 23,232 | 9,972 | **-57%** |
| **Test Coverage** | 80-95% | 73% | **-16%** |
| **Test Count** | 300+ | 218 functions | **-27%** |

#### Детальный план

**Hour 1-2: Измерить реальные метрики**

```bash
# LOC production
find ./internal/infrastructure/grouping -name "*.go" ! -name "*_test.go" | xargs wc -l | tail -1
find ./internal/infrastructure/inhibition -name "*.go" ! -name "*_test.go" | xargs wc -l | tail -1
find ./internal/core/silencing -name "*.go" ! -name "*_test.go" | xargs wc -l | tail -1

# LOC tests
find ./internal/infrastructure/grouping -name "*_test.go" | xargs wc -l | tail -1
find ./internal/infrastructure/inhibition -name "*_test.go" | xargs wc -l | tail -1
find ./internal/core/silencing -name "*_test.go" | xargs wc -l | tail -1

# Test coverage
go test ./internal/infrastructure/grouping/... -coverprofile=cov1.out
go test ./internal/infrastructure/inhibition/... -coverprofile=cov2.out
go test ./internal/core/silencing/... -coverprofile=cov3.out

grep "coverage:" cov1.out cov2.out cov3.out

# Test count
grep -r "^func Test" ./internal/infrastructure/grouping/ | wc -l
grep -r "^func Test" ./internal/infrastructure/inhibition/ | wc -l
grep -r "^func Test" ./internal/core/silencing/ | wc -l

# Benchmarks
grep -r "^func Benchmark" ./internal/infrastructure/grouping/ | wc -l
grep -r "^func Benchmark" ./internal/infrastructure/inhibition/ | wc -l
grep -r "^func Benchmark" ./internal/core/silencing/ | wc -l
```

**Hour 3-4: Обновить tasks.md**

Обновить следующие файлы:

1. **tasks/go-migration-analysis/tasks.md** (main file):
   - Обновить ФАЗА A статус: 61% (9.75/16)
   - Обновить метрики для TN-121 to TN-131
   - Добавить реальные LOC, coverage, test counts

2. **tasks/go-migration-analysis/TN-124/tasks.md**:
   - Coverage: 82.7% → 71.2% (модуль)
   - Tests: 177 → 71+ functions

3. **tasks/go-migration-analysis/TN-125/tasks.md**:
   - LOC: 15,850 → 7,534 (production)
   - Tests: 122+ → 37+ functions

4. **tasks/alertmanager-plus-plus/README.md**:
   - Update Module 1-3 status
   - Update overall Phase A progress

**Hour 5-6: Создать сводные таблицы**

5. Создать **PHASE_A_METRICS.md**:
   - Таблица всех задач с реальными метриками
   - Comparison: заявлено vs фактически
   - Explanation расхождений

6. Update **CHANGELOG.md**:
   - Add entry для audit completion
   - Highlight discrepancies fixed

#### Acceptance Criteria

- [ ] Все метрики измерены и документированы
- [ ] tasks.md обновлен с реальными цифрами
- [ ] PHASE_A_METRICS.md создан
- [ ] CHANGELOG.md обновлен
- [ ] No discrepancies > 10% без explanation

---

## 🟡 HIGH (После Critical)

### H-1: Верифицировать Performance Benchmarks

**Приоритет**: 🟡 HIGH
**Усилия**: 2-3 часа
**Ответственный**: TBD

#### План

1. Запустить все benchmarks:
```bash
cd go-app
go test -bench=. -benchmem ./internal/infrastructure/grouping/... > bench_grouping.txt
go test -bench=. -benchmem ./internal/infrastructure/inhibition/... > bench_inhibition.txt
go test -bench=. -benchmem ./internal/core/silencing/... > bench_silencing.txt
```

2. Создать **PERFORMANCE_REPORT.md**:
   - Все benchmark results
   - Comparison с targets
   - Performance achievements (e.g., 28-23,500x faster)

3. Update tasks.md с верифицированными цифрами

---

### H-2: Создать Документацию TN-121, TN-122

**Приоритет**: 🟡 HIGH
**Усилия**: 4-6 часов
**Ответственный**: TBD

#### План

**TN-121: Grouping Configuration Parser**

1. Создать `tasks/go-migration-analysis/TN-121/`:
   - `requirements.md`: Goals, constraints, acceptance criteria
   - `design.md`: Architecture, YAML format, validation logic
   - `tasks.md`: Implementation checklist

**TN-122: Group Key Generator**

2. Создать `tasks/go-migration-analysis/TN-122/`:
   - `requirements.md`: Hash algorithm, special grouping
   - `design.md`: FNV-1a implementation, object pooling
   - `tasks.md`: Implementation checklist

---

### H-3: Integration Tests с Real Redis

**Приоритет**: 🟡 HIGH
**Усилия**: 2-3 дня
**Ответственный**: TBD

#### План

1. Setup testcontainers для Redis
2. E2E tests:
   - Grouping: Store → Restore groups
   - Timers: Persist → Restore timers
   - Inhibition: Cache failover (Redis → Memory)
3. Failover scenarios:
   - Redis crash → Memory fallback
   - Redis recovery → Restore from Redis

---

## 🟢 MEDIUM (Среднесрочные)

### M-1: Реализовать TN-129 State Manager

**Приоритет**: 🟢 MEDIUM
**Усилия**: 1-2 дня

#### План

1. Создать `state_manager.go`:
   - Track inhibited alerts
   - Persist state в Redis
   - State recovery on startup

2. Integration с TN-127 Matcher

---

### M-2: Реализовать TN-130 API Endpoints

**Приоритет**: 🟢 MEDIUM
**Усилия**: 1 день

#### План

1. REST API endpoints:
   - GET /api/v2/inhibition/rules
   - GET /api/v2/inhibition/status
   - POST /api/v2/inhibition/check

2. OpenAPI spec

---

### M-3: Создать Grafana Dashboards

**Приоритет**: 🟢 MEDIUM
**Усилия**: 1-2 дня

#### План

1. Grouping Dashboard:
   - Active groups gauge
   - Alerts per group histogram
   - Timer duration

2. Inhibition Dashboard:
   - Inhibition checks counter
   - Cache hit rate
   - Matcher performance

3. Silencing Dashboard:
   - Active silences gauge
   - Silenced alerts counter
   - Silence duration

---

## ⚪ LOW (Долгосрочные)

### L-1: Performance Profiling

**Усилия**: 1 неделя

- CPU/Memory profiling
- Optimize hot paths
- Load testing (10K+ groups)

---

### L-2: Advanced Features

**Усилия**: 1+ месяц

- Clustering (multi-instance)
- GraphQL API
- ML insights

---

## 📊 TIMELINE

### Week 1: Critical Fixes

- Day 1-3: **C-1** Test Coverage → 80%+
- Day 4-6: **C-3** Documentation Update
- Day 7: **H-1** Verify Benchmarks

**Deliverables**: Coverage ≥80%, Docs updated, Benchmarks verified

### Week 2-4: Complete Module 3

- Day 8-9: **C-2.1** TN-132 Matcher
- Day 10-11: **C-2.2** TN-133 Storage
- Day 12-13: **C-2.3** TN-134 Manager
- Day 14: **C-2.4** TN-135 API
- Day 15-16: **C-2.5** TN-136 UI
- Day 17-21: Testing & Integration

**Deliverables**: Module 3 complete (100%)

### Week 5: Polish & Deployment

- Day 22-23: **H-2** Create TN-121/122 docs
- Day 24-26: **H-3** Integration tests
- Day 27-28: **M-3** Grafana dashboards
- Day 29-30: Final review & deployment

**Deliverables**: ФАЗА A 100% complete, production-ready

---

## ✅ ACCEPTANCE CRITERIA (Overall Phase A)

### Must Have (для Production)

- [ ] Test Coverage ≥ 80% (all modules)
- [ ] All 16 tasks completed (TN-121 to TN-136)
- [ ] All tests passing (100%)
- [ ] Documentation accurate (no >10% discrepancies)
- [ ] Integration tests passing
- [ ] Zero linter errors
- [ ] Performance benchmarks verified

### Should Have (желательно)

- [ ] Grafana dashboards created
- [ ] TN-129 State Manager implemented
- [ ] TN-130 API endpoints implemented
- [ ] E2E tests with real Redis
- [ ] Performance profiling done

### Nice to Have (опционально)

- [ ] Advanced features (clustering, ML)
- [ ] Load testing (10K+ groups)
- [ ] Security audit

---

## 📞 ОТВЕТСТВЕННОСТЬ

| Action Item | Owner | Deadline | Status |
|-------------|-------|----------|--------|
| **C-1** Coverage | TBD | TBD | ❌ TODO |
| **C-2** Module 3 | TBD | TBD | ❌ TODO |
| **C-3** Docs | TBD | TBD | ❌ TODO |
| **H-1** Benchmarks | TBD | TBD | ❌ TODO |
| **H-2** TN-121/122 docs | TBD | TBD | ❌ TODO |
| **H-3** Integration tests | TBD | TBD | ❌ TODO |

---

## 📝 TRACKING

Update this section as action items are completed:

- [ ] C-1: Coverage → 80%+ (0% complete)
- [ ] C-2: Module 3 complete (17% complete, TN-131 done)
- [ ] C-3: Docs updated (0% complete)
- [ ] H-1: Benchmarks verified (0% complete)
- [ ] H-2: TN-121/122 docs (0% complete)
- [ ] H-3: Integration tests (0% complete)

**Overall Progress**: **3% complete** (0.5/16 critical items)

---

**Created**: 2025-11-04
**Last Updated**: 2025-11-04
**Status**: 🔴 **ACTION REQUIRED**

*Этот документ содержит конкретные, выполнимые рекомендации для устранения блокеров и завершения ФАЗЫ A.*

