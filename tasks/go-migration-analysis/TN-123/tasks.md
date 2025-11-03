# TN-123: Alert Group Manager - Implementation Tasks

**Status**: 🔲 NOT STARTED → 🎯 TARGET: 150% QUALITY
**Started**: 2025-11-03
**Target Completion**: 2025-11-03 (same day, ~19 hours)
**Dependencies**: ✅ TN-121 (completed), ✅ TN-122 (completed)

---

## 📊 Progress Overview

**Overall Progress**: 0 / 72 tasks (0%)

| Phase | Tasks | Completed | Progress |
|-------|-------|-----------|----------|
| Phase 1: Data Models & Interfaces | 10 | 0 | 0% |
| Phase 2: Core Implementation | 15 | 0 | 0% |
| Phase 3: Metrics & Observability | 8 | 0 | 0% |
| Phase 4: Integration | 12 | 0 | 0% |
| Phase 5: Testing (95%+ coverage) | 18 | 0 | 0% |
| Phase 6: Documentation | 5 | 0 | 0% |
| Phase 7: Performance Optimization | 4 | 0 | 0% |

---

## Phase 1: Data Models & Interfaces (2 hours)

### 1.1 Core Data Structures
- [ ] **Task 1.1.1**: Создать `internal/infrastructure/grouping/manager.go`
  - [ ] Определить `AlertGroup` struct
  - [ ] Определить `GroupMetadata` struct
  - [ ] Определить `GroupState` enum (firing/resolved/mixed/silenced)
  - [ ] Добавить `sync.RWMutex` в AlertGroup для thread-safety
  - [ ] Добавить JSON tags для serialization

- [ ] **Task 1.1.2**: Создать `AlertGroupManager` interface
  - [ ] `AddAlertToGroup(ctx, alert, groupKey)` - добавить алерт в группу
  - [ ] `RemoveAlertFromGroup(ctx, fingerprint, groupKey)` - удалить алерт
  - [ ] `GetGroup(ctx, groupKey)` - получить группу
  - [ ] `ListGroups(ctx, filters)` - список групп
  - [ ] `UpdateGroupState(ctx, groupKey)` - обновить состояние
  - [ ] `CleanupExpiredGroups(ctx, maxAge)` - очистить истекшие
  - [ ] `GetGroupByFingerprint(ctx, fingerprint)` - найти группу по fingerprint
  - [ ] `GetMetrics(ctx)` - получить метрики
  - [ ] `GetStats(ctx)` - получить статистику (150% enhancement)

- [ ] **Task 1.1.3**: Создать `GroupStorage` interface (abstraction для Redis)
  - [ ] `Store(ctx, group)` - сохранить группу
  - [ ] `Load(ctx, groupKey)` - загрузить группу
  - [ ] `Delete(ctx, groupKey)` - удалить группу
  - [ ] `ListKeys(ctx)` - список ключей всех групп
  - [ ] `Size(ctx)` - количество групп

- [ ] **Task 1.1.4**: Создать helper structures
  - [ ] `GroupFilters` struct для фильтрации ListGroups
  - [ ] `GroupMetrics` struct для GetMetrics
  - [ ] `GroupStats` struct для GetStats (150% enhancement)
  - [ ] `DefaultGroupManagerConfig` struct для конструктора

### 1.2 Error Types
- [ ] **Task 1.2.1**: Создать `internal/infrastructure/grouping/errors.go`
  - [ ] `InvalidAlertError` - невалидный алерт
  - [ ] `GroupNotFoundError` - группа не найдена
  - [ ] `StorageError` - ошибка хранилища
  - [ ] Реализовать `error.Unwrap()` для StorageError

### 1.3 Helper Methods
- [ ] **Task 1.3.1**: Методы для AlertGroup
  - [ ] `Clone()` - deep copy группы (150% enhancement)
  - [ ] `Size()` - количество алертов
  - [ ] `GetFiringCount()` - количество firing алертов
  - [ ] `GetResolvedCount()` - количество resolved алертов
  - [ ] `IsExpired(maxAge)` - проверка истечения

- [ ] **Task 1.3.2**: Методы для GroupMetadata
  - [ ] `UpdateState()` - пересчет состояния
  - [ ] `Touch()` - обновить UpdatedAt
  - [ ] `MarkResolved()` - пометить как resolved

---

## Phase 2: Core Implementation (4 hours)

### 2.1 DefaultGroupManager Setup
- [ ] **Task 2.1.1**: Реализовать constructor `NewDefaultGroupManager`
  - [ ] Валидация входных параметров (KeyGenerator, Config required)
  - [ ] Инициализация maps (groups, fingerprintIndex)
  - [ ] Настройка logger defaults
  - [ ] Создание groupStats

- [ ] **Task 2.1.2**: Реализовать внутреннее хранилище
  - [ ] `map[GroupKey]*AlertGroup` - основное хранилище
  - [ ] `map[string]GroupKey` - обратный индекс fingerprint → groupKey
  - [ ] `sync.RWMutex` для thread-safety
  - [ ] Инициализация в constructor

### 2.2 Lifecycle Management Methods
- [ ] **Task 2.2.1**: Реализовать `AddAlertToGroup`
  - [ ] Валидация входных данных (nil check, fingerprint check)
  - [ ] Lock acquisition (m.mu.Lock)
  - [ ] Get or create group
  - [ ] Add alert to group (thread-safe)
  - [ ] Update fingerprint index
  - [ ] Update group state
  - [ ] Update stats (totalAdds)
  - [ ] Record metrics
  - [ ] Structured logging
  - [ ] Return updated group

- [ ] **Task 2.2.2**: Реализовать `RemoveAlertFromGroup`
  - [ ] Lock acquisition
  - [ ] Find group (GroupNotFoundError if not exists)
  - [ ] Remove alert from group
  - [ ] Remove from fingerprint index
  - [ ] Delete group if empty
  - [ ] Update group state if not empty
  - [ ] Update stats (totalRemoves)
  - [ ] Record metrics
  - [ ] Return bool (was removed)

- [ ] **Task 2.2.3**: Реализовать `UpdateGroupState`
  - [ ] Lock acquisition (RLock for read, Lock for write)
  - [ ] Get group
  - [ ] Count firing/resolved alerts
  - [ ] Determine new state (firing/resolved/mixed)
  - [ ] Update metadata timestamps
  - [ ] Record metrics if state changed

- [ ] **Task 2.2.4**: Реализовать `CleanupExpiredGroups`
  - [ ] Calculate cutoff time (now - maxAge)
  - [ ] Lock acquisition
  - [ ] Iterate groups, find expired
  - [ ] Batch delete expired groups
  - [ ] Cleanup fingerprint index
  - [ ] Update stats (totalCleanups, lastCleanupTime)
  - [ ] Record metrics
  - [ ] Return deleted count

### 2.3 Query Methods
- [ ] **Task 2.3.1**: Реализовать `GetGroup`
  - [ ] RLock acquisition
  - [ ] Map lookup
  - [ ] GroupNotFoundError if not exists
  - [ ] Return shallow copy (150% enhancement)

- [ ] **Task 2.3.2**: Реализовать `ListGroups`
  - [ ] RLock acquisition
  - [ ] Apply filters (state, minSize, maxAge)
  - [ ] Pre-allocate result slice
  - [ ] Pagination support (limit, offset) - 150% enhancement
  - [ ] Return shallow copies

- [ ] **Task 2.3.3**: Реализовать `GetGroupByFingerprint`
  - [ ] RLock acquisition
  - [ ] Lookup in fingerprintIndex
  - [ ] Get group by key
  - [ ] Return (groupKey, group, error)

### 2.4 Internal Helper Methods
- [ ] **Task 2.4.1**: Реализовать `createNewGroup`
  - [ ] Create AlertGroup with empty Alerts map
  - [ ] Initialize GroupMetadata (CreatedAt, UpdatedAt)
  - [ ] Set GroupBy from config
  - [ ] Return new group

- [ ] **Task 2.4.2**: Реализовать `updateGroupStateUnsafe` (caller must hold lock)
  - [ ] Count firing/resolved alerts
  - [ ] Determine state
  - [ ] Update metadata timestamps
  - [ ] Update FiringCount, ResolvedCount

- [ ] **Task 2.4.3**: Реализовать `isGroupExpired`
  - [ ] Check if all resolved + resolvedAt < cutoff
  - [ ] Check if updatedAt < cutoff
  - [ ] Return bool

---

## Phase 3: Metrics & Observability (1 hour)

### 3.1 Prometheus Metrics
- [ ] **Task 3.1.1**: Добавить metrics в `pkg/metrics/business.go`
  - [ ] `alert_groups_active_total` (Gauge) - количество активных групп
  - [ ] `alert_group_size` (Histogram) - распределение размеров групп
  - [ ] `alert_group_operations_total` (CounterVec) - операции (add/remove/cleanup)
  - [ ] `alert_group_operation_duration_seconds` (HistogramVec) - длительность операций

- [ ] **Task 3.1.2**: Реализовать metric recording methods
  - [ ] `recordAddMetrics(groupKey, isNew, duration)`
  - [ ] `recordRemoveMetrics(groupKey, duration)`
  - [ ] `recordCleanupMetrics(deletedCount, duration)`
  - [ ] `recordGroupSizeDistribution()` - периодический snapshot

### 3.2 Observability Methods
- [ ] **Task 3.2.1**: Реализовать `GetMetrics`
  - [ ] Aggregate active groups count
  - [ ] Build alerts_per_group map
  - [ ] Calculate size distribution (1-10, 11-50, etc.)
  - [ ] Aggregate operations from stats
  - [ ] Return GroupMetrics

- [ ] **Task 3.2.2**: Реализовать `GetStats` (150% enhancement)
  - [ ] Return detailed stats (totalAdds, totalRemoves, etc.)
  - [ ] Include lastCleanupTime
  - [ ] Include memory usage estimate
  - [ ] Include performance metrics (avg duration)

### 3.3 Structured Logging
- [ ] **Task 3.3.1**: Добавить contextual logging
  - [ ] Logger с correlation IDs
  - [ ] Log levels (Debug, Info, Warn, Error)
  - [ ] Consistent log fields (group_key, alert, fingerprint, operation)

---

## Phase 4: Integration (2 hours)

### 4.1 AlertProcessor Integration
- [ ] **Task 4.1.1**: Обновить `internal/core/services/alert_processor.go`
  - [ ] Добавить `groupManager grouping.AlertGroupManager` field
  - [ ] Обновить `AlertProcessorConfig` (добавить GroupManager, KeyGenerator)
  - [ ] Обновить constructor валидацию

- [ ] **Task 4.1.2**: Интегрировать grouping в `ProcessAlert`
  - [ ] После deduplication, перед classification
  - [ ] Generate group key using KeyGenerator
  - [ ] Call `groupManager.AddAlertToGroup(ctx, alert, groupKey)`
  - [ ] Graceful degradation при ошибках (log, continue)
  - [ ] Не блокировать processing при group errors

- [ ] **Task 4.1.3**: Добавить метод `generateGroupKey`
  - [ ] Get groupBy labels from config (or default)
  - [ ] Call keyGenerator.GenerateKey(alert.Labels, groupBy)
  - [ ] Handle errors (fallback to global group)

### 4.2 HTTP API Endpoints
- [ ] **Task 4.2.1**: Создать `internal/infrastructure/handlers/groups.go`
  - [ ] `HandleListGroups` - GET /api/v1/groups
  - [ ] `HandleGetGroup` - GET /api/v1/groups/:key
  - [ ] `HandleGroupMetrics` - GET /api/v1/groups/metrics
  - [ ] `HandleGroupCleanup` - DELETE /api/v1/groups/cleanup

- [ ] **Task 4.2.2**: Регистрация handlers в `cmd/server/main.go`
  - [ ] Initialize DefaultGroupManager
  - [ ] Register 4 HTTP endpoints
  - [ ] Add to API documentation

### 4.3 Main.go Setup
- [ ] **Task 4.3.1**: Обновить `cmd/server/main.go`
  - [ ] Initialize GroupKeyGenerator (TN-122)
  - [ ] Initialize DefaultGroupManager
  - [ ] Wire into AlertProcessor
  - [ ] Register HTTP handlers
  - [ ] Add graceful shutdown logic

### 4.4 Configuration Support
- [ ] **Task 4.4.1**: Обновить `internal/config/config.go`
  - [ ] Добавить `Grouping` section
  - [ ] Параметры: enabled, cleanup_interval, max_group_age
  - [ ] Validation rules

- [ ] **Task 4.4.2**: Обновить `config.yaml` example
  - [ ] Добавить grouping configuration
  - [ ] Документация параметров
  - [ ] Defaults

---

## Phase 5: Testing - 95%+ Coverage (4 hours)

### 5.1 Unit Tests - Core Functionality
- [ ] **Task 5.1.1**: Создать `manager_test.go`
  - [ ] `TestNewDefaultGroupManager` - constructor validation
  - [ ] `TestAddAlertToGroup_NewGroup` - создание новой группы
  - [ ] `TestAddAlertToGroup_ExistingGroup` - добавление в существующую
  - [ ] `TestAddAlertToGroup_UpdateExisting` - обновление алерта
  - [ ] `TestAddAlertToGroup_NilAlert` - error handling
  - [ ] `TestAddAlertToGroup_EmptyFingerprint` - error handling

- [ ] **Task 5.1.2**: Unit tests - RemoveAlertFromGroup
  - [ ] `TestRemoveAlert_Success` - успешное удаление
  - [ ] `TestRemoveAlert_DeletesEmptyGroup` - удаление пустой группы
  - [ ] `TestRemoveAlert_NotFound` - алерт не найден
  - [ ] `TestRemoveAlert_GroupNotFound` - группа не найдена

- [ ] **Task 5.1.3**: Unit tests - GetGroup & ListGroups
  - [ ] `TestGetGroup_Success` - успешное получение
  - [ ] `TestGetGroup_NotFound` - группа не найдена
  - [ ] `TestGetGroup_ReturnsCopy` - возвращает копию (150%)
  - [ ] `TestListGroups_Empty` - пустой список
  - [ ] `TestListGroups_MultipleGroups` - несколько групп
  - [ ] `TestListGroups_WithFilters` - фильтрация (150%)
  - [ ] `TestListGroups_WithPagination` - пагинация (150%)

- [ ] **Task 5.1.4**: Unit tests - CleanupExpiredGroups
  - [ ] `TestCleanup_ExpiredByResolvedTime` - resolved groups
  - [ ] `TestCleanup_ExpiredByUpdateTime` - inactive groups
  - [ ] `TestCleanup_NoExpiredGroups` - ничего не удалено
  - [ ] `TestCleanup_UpdatesStats` - обновление статистики

- [ ] **Task 5.1.5**: Unit tests - UpdateGroupState
  - [ ] `TestUpdateState_AllFiring` - все firing
  - [ ] `TestUpdateState_AllResolved` - все resolved
  - [ ] `TestUpdateState_Mixed` - firing + resolved
  - [ ] `TestUpdateState_UpdatesTimestamps` - timestamps обновлены

### 5.2 Unit Tests - Edge Cases & Errors
- [ ] **Task 5.2.1**: Edge cases
  - [ ] `TestConcurrentAdds` - concurrent добавление
  - [ ] `TestConcurrentRemoves` - concurrent удаление
  - [ ] `TestLargeGroup_1000Alerts` - большая группа
  - [ ] `TestManyGroups_10000Groups` - много групп
  - [ ] `TestFingerprintIndexConsistency` - консистентность индекса

- [ ] **Task 5.2.2**: Error handling
  - [ ] `TestAddAlert_InvalidAlert` - InvalidAlertError
  - [ ] `TestGetGroup_StorageError` - StorageError (mock)
  - [ ] `TestCleanup_PartialFailure` - частичный сбой

### 5.3 Integration Tests
- [ ] **Task 5.3.1**: Integration with AlertProcessor
  - [ ] `TestIntegration_AlertProcessor_AutoGrouping` - автогруппировка
  - [ ] `TestIntegration_MultipleAlerts_SameGroup` - несколько алертов в одну группу
  - [ ] `TestIntegration_GracefulDegradation` - fallback при ошибках

- [ ] **Task 5.3.2**: Integration with HTTP API
  - [ ] `TestAPI_ListGroups` - HTTP endpoint
  - [ ] `TestAPI_GetGroup` - HTTP endpoint
  - [ ] `TestAPI_Metrics` - HTTP endpoint
  - [ ] `TestAPI_Cleanup` - HTTP endpoint

### 5.4 Race Tests
- [ ] **Task 5.4.1**: Race detector tests
  - [ ] `TestRace_ConcurrentAddsRemoves` - add + remove одновременно
  - [ ] `TestRace_ReadWhileWrite` - чтение во время записи
  - [ ] Run with `go test -race`

### 5.5 Benchmarks
- [ ] **Task 5.5.1**: Создать `manager_bench_test.go`
  - [ ] `BenchmarkAddAlertToGroup` - target <500μs
  - [ ] `BenchmarkGetGroup` - target <100μs
  - [ ] `BenchmarkListGroups_1000Groups` - target <5ms
  - [ ] `BenchmarkRemoveAlert` - target <500μs
  - [ ] `BenchmarkCleanupExpired` - target <50ms
  - [ ] `BenchmarkConcurrentAdds_Parallel` - throughput test

---

## Phase 6: Documentation (2 hours)

### 6.1 Code Documentation
- [ ] **Task 6.1.1**: Godoc comments
  - [ ] Package comment в manager.go
  - [ ] Interface documentation (AlertGroupManager)
  - [ ] Method documentation (все public methods)
  - [ ] Example usage в godoc

- [ ] **Task 6.1.2**: Inline comments
  - [ ] Сложные алгоритмы
  - [ ] Thread-safety considerations
  - [ ] Performance optimizations

### 6.2 README
- [ ] **Task 6.2.1**: Создать `internal/infrastructure/grouping/README_GROUP_MANAGER.md`
  - [ ] Overview (что такое AlertGroupManager)
  - [ ] Quick Start (основные примеры)
  - [ ] API Reference (все методы)
  - [ ] Architecture (диаграммы, thread-safety)
  - [ ] Metrics (Prometheus metrics)
  - [ ] Performance (benchmarks, targets)
  - [ ] Integration (with AlertProcessor, HTTP API)
  - [ ] FAQ
  - [ ] Target: 500+ lines

### 6.3 Examples
- [ ] **Task 6.3.1**: Создать `examples/` директорию
  - [ ] `basic_usage.go` - базовое использование
  - [ ] `with_filters.go` - фильтрация групп
  - [ ] `periodic_cleanup.go` - периодическая очистка
  - [ ] `metrics_monitoring.go` - мониторинг метрик

### 6.4 Migration Guide
- [ ] **Task 6.4.1**: Создать `MIGRATION_TN123.md`
  - [ ] Как интегрировать в существующий AlertProcessor
  - [ ] Breaking changes (если есть)
  - [ ] Configuration changes
  - [ ] Rollback plan

---

## Phase 7: Performance Optimization (2 hours)

### 7.1 Profiling
- [ ] **Task 7.1.1**: CPU profiling
  - [ ] Run benchmarks с `-cpuprofile`
  - [ ] Analyze с pprof
  - [ ] Identify hotspots

- [ ] **Task 7.1.2**: Memory profiling
  - [ ] Run benchmarks с `-memprofile`
  - [ ] Analyze allocations
  - [ ] Optimize high-allocation paths

### 7.2 Optimizations
- [ ] **Task 7.2.1**: Reduce allocations
  - [ ] Use object pooling для AlertGroup (if needed)
  - [ ] Pre-allocate slices in ListGroups
  - [ ] Avoid unnecessary copies

- [ ] **Task 7.2.2**: Lock contention optimization
  - [ ] Minimize lock hold time
  - [ ] Use RLock where possible
  - [ ] Consider lock-free algorithms (150%)

### 7.3 Validation
- [ ] **Task 7.3.1**: Validate performance targets
  - [ ] AddAlertToGroup < 500μs ✅
  - [ ] GetGroup < 100μs ✅
  - [ ] ListGroups (1K) < 5ms ✅
  - [ ] Memory per group < 5KB ✅

---

## Phase 8: Validation & Production Readiness (1 hour)

### 8.1 Code Quality
- [ ] **Task 8.1.1**: Linting
  - [ ] Run `golangci-lint run`
  - [ ] Fix all warnings
  - [ ] Achieve Grade A+

- [ ] **Task 8.1.2**: Code review checklist
  - [ ] SOLID principles соблюдены
  - [ ] Error handling comprehensive
  - [ ] Thread-safety verified
  - [ ] No race conditions
  - [ ] Memory leaks addressed

### 8.2 Test Coverage
- [ ] **Task 8.2.1**: Measure coverage
  - [ ] Run `go test -cover ./internal/infrastructure/grouping/...`
  - [ ] Achieve 95%+ coverage
  - [ ] Identify uncovered lines
  - [ ] Add tests for uncovered cases

### 8.3 Final Validation
- [ ] **Task 8.3.1**: Integration testing
  - [ ] Run full test suite
  - [ ] Test with real AlertProcessor
  - [ ] Load testing (10K groups, 100K alerts)

- [ ] **Task 8.3.2**: Documentation review
  - [ ] README complete
  - [ ] Examples работают
  - [ ] API docs accurate

### 8.4 Completion Report
- [ ] **Task 8.4.1**: Создать `COMPLETION_REPORT_TN123.md`
  - [ ] Summary (что реализовано)
  - [ ] Metrics (coverage, performance, LOC)
  - [ ] Quality grade (A+)
  - [ ] Known limitations
  - [ ] Next steps (TN-124, TN-125)

---

## 📈 Success Criteria (150% Quality)

### Baseline (100%)
- [x] All interfaces defined ✅
- [ ] DefaultGroupManager реализован
- [ ] 80%+ test coverage
- [ ] 4 Prometheus metrics
- [ ] HTTP API работает
- [ ] Integration с AlertProcessor
- [ ] Performance: AddAlert <1ms, GetGroup <500μs

### 150% Enhancements (сверх baseline)
- [ ] **95%+ test coverage** (vs 80%)
- [ ] **Thread-safe with race tests** (sync.RWMutex, race detector)
- [ ] **Advanced filtering** (ListGroups с filters, pagination)
- [ ] **Comprehensive benchmarks** (6+ benchmarks, all targets achieved)
- [ ] **Extended metrics** (size distribution, operation stats)
- [ ] **Detailed documentation** (500+ line README, examples)
- [ ] **Performance optimization** (profiling, allocation reduction)
- [ ] **Graceful degradation** (fallback на ungrouped при ошибках)
- [ ] **Production patterns** (context support, timeouts, structured logging)
- [ ] **Code quality A+** (golangci-lint, SOLID, clean code)

---

## 🚀 Blocked Tasks (разблокируются после TN-123)

После завершения TN-123 будут разблокированы:
- **TN-124**: Group Wait/Interval Timers (требует AlertGroupManager)
- **TN-125**: Group Storage (Redis Backend) (требует AlertGroupManager interface)
- **TN-133**: Notification Scheduler (требует группы для batching)

---

## 📝 Notes & Decisions

### Design Decisions
1. **In-memory storage первоначально** - Redis в TN-125
2. **Thread-safe by default** - sync.RWMutex на всех операциях
3. **Graceful degradation** - не блокировать AlertProcessor при ошибках
4. **Fingerprint index** - O(1) поиск группы по alert fingerprint

### Performance Considerations
1. Map lookups O(1) для всех операций
2. RLock для read-only операций (GetGroup, ListGroups)
3. Pre-allocated slices для ListGroups
4. Minimal allocations в hot paths

### Future Enhancements (Post-150%)
1. Redis backend для distributed state (TN-125)
2. Group timers для notification scheduling (TN-124)
3. Advanced queries (label filters, time-range)
4. Clustering support (consistent hashing, replication)

---

## 🎯 Final Checklist (150% Completion)

- [ ] ✅ Все 72 задачи выполнены
- [ ] ✅ Test coverage 95%+
- [ ] ✅ Все benchmarks прошли targets
- [ ] ✅ golangci-lint Grade A+
- [ ] ✅ README 500+ lines
- [ ] ✅ Integration tests passing
- [ ] ✅ HTTP API работает
- [ ] ✅ AlertProcessor интегрирован
- [ ] ✅ Prometheus metrics зарегистрированы
- [ ] ✅ Completion report готов
- [ ] ✅ Смержен в main ветку
- [ ] ✅ TN-124, TN-125 разблокированы

**Target Completion Date**: 2025-11-03
**Quality Grade**: A+ (150% achieved) 🏆
