# TN-125: Group Storage (Redis Backend) - Implementation Tasks

**Status**: 🟡 IN PROGRESS → 🎯 TARGET: 150% QUALITY
**Started**: 2025-11-04
**Target Completion**: 2025-11-18 (2 weeks, ~35 hours)
**Dependencies**: ✅ TN-121 (150%), ✅ TN-122 (200%), ✅ TN-123 (183.6%), ✅ TN-124 (152.6%)

---

## 📊 Progress Overview

**Overall Progress**: 0 / 85 tasks (0%)

| Phase | Tasks | Completed | Progress | Time Est |
|-------|-------|-----------|----------|----------|
| Phase 1: Interfaces & Data Models | 10 | 0 | 0% | 2 hours |
| Phase 2: RedisGroupStorage Implementation | 18 | 0 | 0% | 6 hours |
| Phase 3: MemoryGroupStorage Implementation | 8 | 0 | 0% | 2 hours |
| Phase 4: StorageManager (Fallback/Recovery) | 10 | 0 | 0% | 3 hours |
| Phase 5: DefaultGroupManager Integration | 10 | 0 | 0% | 3 hours |
| Phase 6: Prometheus Metrics | 6 | 0 | 0% | 2 hours |
| Phase 7: Testing (90%+ coverage) | 15 | 0 | 0% | 5 hours |
| Phase 8: Benchmarks & Performance | 5 | 0 | 0% | 3 hours |
| Phase 9: Integration Tests | 6 | 0 | 0% | 3 hours |
| Phase 10: Documentation | 5 | 0 | 0% | 3 hours |
| Phase 11: Validation & Production Readiness | 7 | 0 | 0% | 2 hours |

---

## Phase 1: Interfaces & Data Models (2 hours)

### 1.1 GroupStorage Interface Definition

- [ ] **Task 1.1.1**: Создать `internal/infrastructure/grouping/storage.go`
  - [ ] Определить `GroupStorage` interface с 9 методами
  - [ ] Добавить comprehensive godoc для каждого метода
  - [ ] Определить method signatures: Store, Load, Delete, ListKeys, Size, LoadAll, StoreAll, Ping
  - [ ] Указать thread-safety guarantees
  - [ ] Указать performance targets (baseline + 150%)

- [ ] **Task 1.1.2**: Определить константы для Redis schema
  - [ ] `groupKeyPrefix = "group:"`
  - [ ] `groupIndexKey = "group:index"`
  - [ ] `groupCountKey = "group:count"`
  - [ ] `groupLockPrefix = "lock:group:"`
  - [ ] `groupTTLDefault = 24h`
  - [ ] `groupTTLGracePeriod = 60s`
  - [ ] `lockTTL = 30s`

### 1.2 Configuration Structures

- [ ] **Task 1.2.1**: Создать `RedisGroupStorageConfig` struct
  - [ ] Field: `RedisCache cache.Cache` (required)
  - [ ] Field: `Logger *slog.Logger` (optional)
  - [ ] Field: `TTL time.Duration` (optional, default 24h)
  - [ ] Add validation method: `Validate() error`

- [ ] **Task 1.2.2**: Создать `MemoryGroupStorageConfig` struct
  - [ ] Field: `Logger *slog.Logger` (optional)

### 1.3 Error Types

- [ ] **Task 1.3.1**: Расширить `internal/infrastructure/grouping/errors.go`
  - [ ] `ErrVersionMismatch` - optimistic locking conflict
  - [ ] Fields: Key, ExpectedVersion, ActualVersion
  - [ ] Implement `error` interface
  - [ ] Add `NewVersionMismatchError()` constructor

---

## Phase 2: RedisGroupStorage Implementation (6 hours)

### 2.1 RedisGroupStorage Struct & Constructor

- [ ] **Task 2.1.1**: Создать `redis_group_storage.go`
  - [ ] Define `RedisGroupStorage` struct
  - [ ] Fields: `client *redis.Client`, `logger *slog.Logger`, `ttl time.Duration`
  - [ ] Add comprehensive package documentation

- [ ] **Task 2.1.2**: Реализовать constructor `NewRedisGroupStorage`
  - [ ] Accept `RedisGroupStorageConfig` parameter
  - [ ] Validate config (RedisCache not nil)
  - [ ] Type assert `cache.RedisCache` to extract `*redis.Client`
  - [ ] Set default logger (slog.Default if nil)
  - [ ] Set default TTL (24h if 0)
  - [ ] Ping Redis to verify connection
  - [ ] Return `(*RedisGroupStorage, error)`

### 2.2 Core Operations

- [ ] **Task 2.2.1**: Реализовать `Store(ctx, group)`
  - [ ] Validate group (key not empty, metadata not nil)
  - [ ] Serialize group to JSON: `json.Marshal(group)`
  - [ ] Calculate TTL: `calculateTTL(group)` (resolved: 1h, firing: 24h)
  - [ ] Create Redis pipeline: `client.Pipeline()`
  - [ ] Pipeline command 1: `SET group:{key} {json} EX {ttl}`
  - [ ] Pipeline command 2: `ZADD group:index {updated_at} {key}`
  - [ ] Execute pipeline: `pipe.Exec(ctx)`
  - [ ] Handle errors: `NewStorageError("store", err)`
  - [ ] Log success: "Stored group to Redis"
  - [ ] **Performance target**: <5ms (baseline), <2ms (150% with pipelining)

- [ ] **Task 2.2.2**: Реализовать `Load(ctx, groupKey)`
  - [ ] Build Redis key: `groupKeyPrefix + groupKey`
  - [ ] Execute: `client.Get(ctx, key)`
  - [ ] Handle `redis.Nil` → return `GroupNotFoundError`
  - [ ] Deserialize JSON: `json.Unmarshal(data, &group)`
  - [ ] Validate deserialized data
  - [ ] Log success: "Loaded group from Redis"
  - [ ] **Performance target**: <5ms (baseline), <1ms (150%)

- [ ] **Task 2.2.3**: Реализовать `Delete(ctx, groupKey)`
  - [ ] Create Redis pipeline
  - [ ] Pipeline command 1: `DEL group:{key}`
  - [ ] Pipeline command 2: `ZREM group:index {key}`
  - [ ] Execute pipeline
  - [ ] Handle errors
  - [ ] Log success: "Deleted group from Redis"
  - [ ] **Performance target**: <5ms (baseline), <2ms (150%)

### 2.3 Query Operations

- [ ] **Task 2.3.1**: Реализовать `ListKeys(ctx)`
  - [ ] Execute: `client.ZRange(ctx, groupIndexKey, 0, -1)`
  - [ ] Convert []string to []GroupKey
  - [ ] Handle empty result (return empty slice, not nil)
  - [ ] Log count: "Listed N group keys from Redis"
  - [ ] **Performance target**: <10ms for 1000 keys

- [ ] **Task 2.3.2**: Реализовать `Size(ctx)`
  - [ ] Execute: `client.ZCard(ctx, groupIndexKey)`
  - [ ] Return int count
  - [ ] Handle errors
  - [ ] **Performance target**: <5ms (baseline), <1ms (150%)

### 2.4 Batch Operations (150% Enhancement)

- [ ] **Task 2.4.1**: Реализовать `LoadAll(ctx)` с parallel loading
  - [ ] Step 1: `ListKeys(ctx)` → get all keys
  - [ ] Step 2: Create goroutine pool (workers=10)
  - [ ] Step 3: For each key: `Load(ctx, key)` in goroutine
  - [ ] Step 4: Use semaphore to limit concurrency
  - [ ] Step 5: Collect results in channel: `groupsChan chan *AlertGroup`
  - [ ] Step 6: Collect errors in channel: `errChan chan error` (log, don't fail)
  - [ ] Step 7: Wait for all goroutines via semaphore
  - [ ] Step 8: Close channels, collect results
  - [ ] Log: "Loaded N groups from Redis in Xms"
  - [ ] **Performance target**: <200ms для 1000 групп (baseline), <100ms (150%)

- [ ] **Task 2.4.2**: Реализовать `StoreAll(ctx, groups)` с pipelining
  - [ ] Create Redis pipeline
  - [ ] For each group: add Store commands to pipeline
  - [ ] Execute pipeline once (batched)
  - [ ] Handle partial failures (log, continue)
  - [ ] **Performance target**: <10ms для 100 групп

### 2.5 Health Check

- [ ] **Task 2.5.1**: Реализовать `Ping(ctx)`
  - [ ] Execute: `client.Ping(ctx)`
  - [ ] Return error if unhealthy
  - [ ] **Performance target**: <5ms

### 2.6 Helper Methods

- [ ] **Task 2.6.1**: Реализовать helper: `groupKey(groupKey GroupKey) string`
  - [ ] Return: `groupKeyPrefix + string(groupKey)`

- [ ] **Task 2.6.2**: Реализовать helper: `calculateTTL(group *AlertGroup) time.Duration`
  - [ ] If `group.Metadata.State == GroupStateResolved`: return 1h
  - [ ] Else: return default TTL (24h)
  - [ ] Add grace period: `baseTTL + groupTTLGracePeriod`

---

## Phase 3: MemoryGroupStorage Implementation (2 hours)

### 3.1 MemoryGroupStorage Struct & Constructor

- [ ] **Task 3.1.1**: Создать `memory_group_storage.go`
  - [ ] Define `MemoryGroupStorage` struct
  - [ ] Fields: `groups map[GroupKey]*AlertGroup`, `mu sync.RWMutex`, `logger *slog.Logger`
  - [ ] Add comprehensive package documentation

- [ ] **Task 3.1.2**: Реализовать constructor `NewMemoryGroupStorage`
  - [ ] Accept `logger *slog.Logger` parameter (optional)
  - [ ] Initialize map: `make(map[GroupKey]*AlertGroup)`
  - [ ] Set default logger
  - [ ] Log: "Initialized in-memory group storage"

### 3.2 Core Operations

- [ ] **Task 3.2.1**: Реализовать `Store(ctx, group)`
  - [ ] Lock: `mu.Lock()`, defer `mu.Unlock()`
  - [ ] Deep copy group: `group.Clone()`
  - [ ] Store in map: `groups[group.Key] = clone`
  - [ ] Log success

- [ ] **Task 3.2.2**: Реализовать `Load(ctx, groupKey)`
  - [ ] RLock: `mu.RLock()`, defer `mu.RUnlock()`
  - [ ] Lookup: `group, exists := groups[groupKey]`
  - [ ] If not exists: return `GroupNotFoundError`
  - [ ] Return deep copy: `group.Clone()`

- [ ] **Task 3.2.3**: Реализовать `Delete(ctx, groupKey)`
  - [ ] Lock: `mu.Lock()`, defer `mu.Unlock()`
  - [ ] Delete: `delete(groups, groupKey)`
  - [ ] Log success

- [ ] **Task 3.2.4**: Реализовать `ListKeys`, `Size`, `LoadAll`, `StoreAll`, `Ping`
  - [ ] `ListKeys`: iterate map, collect keys
  - [ ] `Size`: return `len(groups)`
  - [ ] `LoadAll`: iterate map, collect Clone() copies
  - [ ] `StoreAll`: iterate slice, store each group
  - [ ] `Ping`: always return `nil` (memory always healthy)

---

## Phase 4: StorageManager (Fallback/Recovery) (3 hours)

### 4.1 StorageManager Struct & Constructor

- [ ] **Task 4.1.1**: Создать `storage_manager.go`
  - [ ] Define `StorageManager` struct
  - [ ] Fields: `primary`, `fallback`, `current`, `mu`, `logger`, `metrics`, `healthTicker`
  - [ ] Add comprehensive documentation

- [ ] **Task 4.1.2**: Реализовать constructor `NewStorageManager`
  - [ ] Parameters: `primary`, `fallback`, `logger`, `metrics`
  - [ ] Set `current = primary` (start with Redis)
  - [ ] Call `startHealthCheck()` to start polling
  - [ ] Log: "Initialized storage manager with automatic fallback"

### 4.2 Health Check Poller

- [ ] **Task 4.2.1**: Реализовать `startHealthCheck()`
  - [ ] Create ticker: `time.NewTicker(30 * time.Second)`
  - [ ] Start goroutine: `go func() { for range ticker.C {...} }()`
  - [ ] In loop: `primary.Ping(ctx)` with 5s timeout
  - [ ] If error: switch to fallback, log, metrics
  - [ ] If success: switch back to primary (if was fallback), log, metrics
  - [ ] Use `mu.Lock()` to protect `current` field

### 4.3 Method Wrappers

- [ ] **Task 4.3.1**: Реализовать `Store(ctx, group)` wrapper
  - [ ] Read current storage: `mu.RLock()`, `storage := current`, `mu.RUnlock()`
  - [ ] Call: `storage.Store(ctx, group)`
  - [ ] If error AND using primary: switch to fallback, retry
  - [ ] Record metrics on fallback

- [ ] **Task 4.3.2**: Реализовать wrappers для `Load`, `Delete`, `ListKeys`, `Size`, `LoadAll`, `StoreAll`, `Ping`
  - [ ] Same pattern: read current, delegate, handle errors
  - [ ] No automatic fallback for read operations (only log)

### 4.4 Graceful Shutdown

- [ ] **Task 4.4.1**: Реализовать `Stop()`
  - [ ] Stop health check ticker: `healthTicker.Stop()`
  - [ ] Log: "Stopped storage manager"

---

## Phase 5: DefaultGroupManager Integration (3 hours)

### 5.1 Constructor Update

- [ ] **Task 5.1.1**: Обновить `DefaultGroupManagerConfig` struct
  - [ ] Add field: `Storage GroupStorage` (optional, defaults to MemoryGroupStorage)
  - [ ] Update godoc

- [ ] **Task 5.1.2**: Обновить constructor `NewDefaultGroupManager`
  - [ ] Check if `config.Storage == nil`
  - [ ] If nil: create `MemoryGroupStorage` as default
  - [ ] Log warning: "Using in-memory storage (data loss on restart)"
  - [ ] Assign: `storage: config.Storage`

### 5.2 Store on Create/Update

- [ ] **Task 5.2.1**: Обновить `AddAlertToGroup` method
  - [ ] After modifying group: call `storage.Store(ctx, group)`
  - [ ] Handle error: log warning, don't fail operation (graceful degradation)
  - [ ] Metrics: record storage operation

- [ ] **Task 5.2.2**: Обновить `RemoveAlertFromGroup` method
  - [ ] After removing alert: call `storage.Store(ctx, group)` if group not empty
  - [ ] If group empty: call `storage.Delete(ctx, groupKey)`
  - [ ] Handle errors gracefully

- [ ] **Task 5.2.3**: Обновить `UpdateGroupState` method
  - [ ] After updating state: call `storage.Store(ctx, group)`
  - [ ] Handle errors gracefully

### 5.3 LoadAll on Startup

- [ ] **Task 5.3.1**: Создать method `RestoreGroupsFromStorage(ctx)`
  - [ ] Call: `storage.LoadAll(ctx)`
  - [ ] Lock: `mu.Lock()`, defer `mu.Unlock()`
  - [ ] For each group: add to `groups` map
  - [ ] For each group: rebuild `fingerprintIndex`
  - [ ] Log: "Restored N groups from storage in Xms"
  - [ ] Metrics: `RecordGroupsRestored(count)`

- [ ] **Task 5.3.2**: Интегрировать в `main.go`
  - [ ] After creating `DefaultGroupManager`
  - [ ] Call: `groupManager.RestoreGroupsFromStorage(ctx)`
  - [ ] Handle error: log fatal if critical, warn if partial

### 5.4 Backward Compatibility

- [ ] **Task 5.4.1**: Добавить feature flag в config
  - [ ] Add: `grouping.storage_enabled` (bool, default: true)
  - [ ] If false: use MemoryGroupStorage (skip Redis)
  - [ ] Log: "Group storage disabled by config"

---

## Phase 6: Prometheus Metrics (2 hours)

### 6.1 Metrics Definition

- [ ] **Task 6.1.1**: Добавить metrics в `pkg/metrics/business.go`
  - [ ] Metric 1: `grouping_storage_operations_total` (CounterVec)
    - [ ] Labels: `operation` (store/load/delete), `result` (success/error)
  - [ ] Metric 2: `grouping_storage_duration_seconds` (HistogramVec)
    - [ ] Labels: `operation`
    - [ ] Buckets: [0.001, 0.002, 0.005, 0.010, 0.050, 0.100, 0.500]
  - [ ] Metric 3: `grouping_storage_fallback_total` (CounterVec)
    - [ ] Labels: `reason` (redis_unhealthy, store_error, etc.)
  - [ ] Metric 4: `grouping_storage_recovery_total` (Counter)
  - [ ] Metric 5: `grouping_groups_restored_total` (Counter)
  - [ ] Metric 6: `grouping_storage_health` (GaugeVec)
    - [ ] Labels: `backend` (redis, memory)
    - [ ] Values: 1=healthy, 0=unhealthy

### 6.2 Metrics Methods

- [ ] **Task 6.2.1**: Добавить methods в `BusinessMetrics`
  - [ ] `RecordStorageOperation(operation, result string)`
  - [ ] `RecordStorageDuration(operation string, duration time.Duration)`
  - [ ] `IncStorageFallback(reason string)`
  - [ ] `IncStorageRecovery()`
  - [ ] `RecordGroupsRestored(count int)`
  - [ ] `SetStorageHealth(backend string, healthy bool)` (1 or 0)

### 6.3 Metrics Integration

- [ ] **Task 6.3.1**: Интегрировать metrics в `RedisGroupStorage`
  - [ ] In `Store()`: record operation + duration
  - [ ] In `Load()`: record operation + duration
  - [ ] In `Delete()`: record operation + duration
  - [ ] In `LoadAll()`: record duration
  - [ ] In `Ping()`: update health gauge

- [ ] **Task 6.3.2**: Интегрировать metrics в `StorageManager`
  - [ ] In `startHealthCheck()`: update health gauge
  - [ ] On fallback: `IncStorageFallback(reason)`
  - [ ] On recovery: `IncStorageRecovery()`

---

## Phase 7: Testing (90%+ coverage) (5 hours)

### 7.1 RedisGroupStorage Unit Tests

- [ ] **Task 7.1.1**: Создать `redis_group_storage_test.go`
  - [ ] Test: `TestNewRedisGroupStorage_Success`
  - [ ] Test: `TestNewRedisGroupStorage_NilCache` (error)
  - [ ] Test: `TestNewRedisGroupStorage_InvalidCacheType` (error)
  - [ ] Test: `TestNewRedisGroupStorage_RedisUnhealthy` (error)

- [ ] **Task 7.1.2**: Создать tests для `Store()`
  - [ ] Test: `TestRedisGroupStorage_Store_Success`
  - [ ] Test: `TestRedisGroupStorage_Store_NilGroup` (error)
  - [ ] Test: `TestRedisGroupStorage_Store_EmptyKey` (error)
  - [ ] Test: `TestRedisGroupStorage_Store_RedisError` (error)
  - [ ] Test: `TestRedisGroupStorage_Store_VerifyTTL` (resolved: 1h, firing: 24h)

- [ ] **Task 7.1.3**: Создать tests для `Load()`
  - [ ] Test: `TestRedisGroupStorage_Load_Success`
  - [ ] Test: `TestRedisGroupStorage_Load_NotFound` (GroupNotFoundError)
  - [ ] Test: `TestRedisGroupStorage_Load_RedisError` (StorageError)
  - [ ] Test: `TestRedisGroupStorage_Load_InvalidJSON` (unmarshal error)

- [ ] **Task 7.1.4**: Создать tests для `Delete()`, `ListKeys()`, `Size()`, `Ping()`
  - [ ] 4 tests per method (success, error cases)

- [ ] **Task 7.1.5**: Создать tests для `LoadAll()`
  - [ ] Test: `TestRedisGroupStorage_LoadAll_Empty` (no groups)
  - [ ] Test: `TestRedisGroupStorage_LoadAll_Success` (10 groups)
  - [ ] Test: `TestRedisGroupStorage_LoadAll_ParallelLoading` (1000 groups, verify <100ms)
  - [ ] Test: `TestRedisGroupStorage_LoadAll_PartialFailure` (some groups fail, log, continue)

### 7.2 MemoryGroupStorage Unit Tests

- [ ] **Task 7.2.1**: Создать `memory_group_storage_test.go`
  - [ ] Test: `TestMemoryGroupStorage_Store_Success`
  - [ ] Test: `TestMemoryGroupStorage_Load_Success`
  - [ ] Test: `TestMemoryGroupStorage_Load_NotFound`
  - [ ] Test: `TestMemoryGroupStorage_Delete_Success`
  - [ ] Test: `TestMemoryGroupStorage_ListKeys`, `Size`, `LoadAll`, `Ping`
  - [ ] Test: `TestMemoryGroupStorage_ConcurrentAccess` (race detector)

### 7.3 StorageManager Unit Tests

- [ ] **Task 7.3.1**: Создать `storage_manager_test.go`
  - [ ] Test: `TestStorageManager_AutomaticFallback` (Redis unhealthy → switch to memory)
  - [ ] Test: `TestStorageManager_AutomaticRecovery` (Redis restored → switch back)
  - [ ] Test: `TestStorageManager_Store_WithFallback` (primary fails, retry with fallback)
  - [ ] Test: `TestStorageManager_HealthCheckPolling` (verify 30s ticker)
  - [ ] Test: `TestStorageManager_Stop` (graceful shutdown)

### 7.4 DefaultGroupManager Integration Tests

- [ ] **Task 7.4.1**: Создать `manager_storage_integration_test.go`
  - [ ] Test: `TestDefaultGroupManager_Integration_WithRedisStorage`
    - [ ] Create group, verify stored in Redis
    - [ ] Restart manager, verify restored from Redis
  - [ ] Test: `TestDefaultGroupManager_Integration_WithMemoryStorage`
    - [ ] Create group, restart, verify NOT restored (expected)
  - [ ] Test: `TestDefaultGroupManager_RestoreGroupsFromStorage`
    - [ ] Pre-populate Redis with 100 groups
    - [ ] Create manager, call RestoreGroupsFromStorage()
    - [ ] Verify all 100 groups loaded

---

## Phase 8: Benchmarks & Performance (3 hours)

### 8.1 RedisGroupStorage Benchmarks

- [ ] **Task 8.1.1**: Создать benchmarks в `redis_group_storage_bench_test.go`
  - [ ] Benchmark: `BenchmarkRedisGroupStorage_Store` (target: <2ms)
  - [ ] Benchmark: `BenchmarkRedisGroupStorage_Load` (target: <1ms)
  - [ ] Benchmark: `BenchmarkRedisGroupStorage_Delete` (target: <2ms)
  - [ ] Benchmark: `BenchmarkRedisGroupStorage_LoadAll_1K` (target: <100ms)
  - [ ] Benchmark: `BenchmarkRedisGroupStorage_StoreAll_100` (target: <10ms)

### 8.2 Performance Validation

- [ ] **Task 8.2.1**: Запустить benchmarks и валидировать
  - [ ] Run: `go test -bench=. -benchmem -count=5`
  - [ ] Verify: Store <2ms ✅
  - [ ] Verify: Load <1ms ✅
  - [ ] Verify: LoadAll_1K <100ms ✅
  - [ ] Document results в `PERFORMANCE_REPORT.md`

### 8.3 Memory Profiling

- [ ] **Task 8.3.1**: Профилировать memory usage
  - [ ] Run: `go test -bench=LoadAll -memprofile=mem.prof`
  - [ ] Analyze: `go tool pprof mem.prof`
  - [ ] Verify: RedisGroupStorage struct <100KB ✅
  - [ ] Optimize allocations if needed

---

## Phase 9: Integration Tests (3 hours)

### 9.1 Multi-Replica Scenarios

- [ ] **Task 9.1.1**: Создать integration test: `TestMultiReplica_DistributedState`
  - [ ] Setup: 3 DefaultGroupManagers with shared RedisGroupStorage
  - [ ] Manager-1: create group A
  - [ ] Manager-2: Load group A (verify exists)
  - [ ] Manager-3: Load group A (verify exists)
  - [ ] Verify: all 3 managers see same state

### 9.2 Redis Failure Simulation

- [ ] **Task 9.2.1**: Создать integration test: `TestRedisFailure_GracefulDegradation`
  - [ ] Setup: DefaultGroupManager with StorageManager
  - [ ] Step 1: Normal operation (Redis healthy)
  - [ ] Step 2: Stop Redis (simulate failure)
  - [ ] Step 3: Verify: automatic fallback to MemoryGroupStorage
  - [ ] Step 4: Create group (should succeed with memory)
  - [ ] Step 5: Restart Redis
  - [ ] Step 6: Verify: automatic recovery to RedisGroupStorage

### 9.3 Restart Recovery

- [ ] **Task 9.3.1**: Создать integration test: `TestRestart_StateRecovery`
  - [ ] Setup: Create 100 groups in Redis
  - [ ] Simulate restart: create new manager
  - [ ] Call: RestoreGroupsFromStorage()
  - [ ] Verify: all 100 groups loaded correctly
  - [ ] Verify: fingerprint index rebuilt correctly

---

## Phase 10: Documentation (3 hours)

### 10.1 README

- [ ] **Task 10.1.1**: Создать `README_GROUP_STORAGE.md` (500+ lines)
  - [ ] Section 1: Overview (what is GroupStorage)
  - [ ] Section 2: Architecture (Redis schema, interfaces)
  - [ ] Section 3: Usage Examples (initialization, Store/Load/LoadAll)
  - [ ] Section 4: Configuration (TTL, fallback, health check)
  - [ ] Section 5: Performance (benchmarks, targets, optimization)
  - [ ] Section 6: Metrics (6 Prometheus metrics с примерами PromQL)
  - [ ] Section 7: Troubleshooting (common issues, debugging)
  - [ ] Section 8: Multi-Replica Setup (HPA configuration)
  - [ ] Section 9: Disaster Recovery (backup, restore procedures)

### 10.2 API Documentation

- [ ] **Task 10.2.1**: Обновить godoc для всех public types
  - [ ] GroupStorage interface: comprehensive examples
  - [ ] RedisGroupStorage: usage examples, Redis schema
  - [ ] MemoryGroupStorage: fallback scenarios
  - [ ] StorageManager: automatic fallback examples

### 10.3 Runbook

- [ ] **Task 10.3.1**: Создать `RUNBOOK_GROUP_STORAGE.md`
  - [ ] Section 1: Health Check Procedures
  - [ ] Section 2: Alert Runbooks (storage fallback, recovery, errors)
  - [ ] Section 3: Performance Degradation (diagnosis, mitigation)
  - [ ] Section 4: Data Loss Recovery (LoadAll, manual restore)
  - [ ] Section 5: Scaling Procedures (add replicas, Redis cluster)

### 10.4 Migration Guide

- [ ] **Task 10.4.1**: Создать `MIGRATION_GUIDE_TN125.md`
  - [ ] Section 1: Pre-Migration Checklist
  - [ ] Section 2: Step-by-Step Migration (TN-123 in-memory → TN-125 Redis)
  - [ ] Section 3: Rollback Procedures
  - [ ] Section 4: Validation Steps

---

## Phase 11: Validation & Production Readiness (2 hours)

### 11.1 Test Coverage Validation

- [ ] **Task 11.1.1**: Измерить test coverage
  - [ ] Run: `go test -coverprofile=coverage.out ./internal/infrastructure/grouping/`
  - [ ] View: `go tool cover -html=coverage.out`
  - [ ] Verify: ≥90% coverage ✅
  - [ ] Document uncovered lines (if any)

### 11.2 Race Detector Tests

- [ ] **Task 11.2.1**: Запустить tests с race detector
  - [ ] Run: `go test -race ./internal/infrastructure/grouping/`
  - [ ] Verify: NO race conditions detected ✅

### 11.3 Linter & Code Quality

- [ ] **Task 11.3.1**: Запустить linters
  - [ ] Run: `golangci-lint run ./internal/infrastructure/grouping/`
  - [ ] Fix all issues (if any)
  - [ ] Verify: Grade A+ ✅

### 11.4 Integration Validation

- [ ] **Task 11.4.1**: End-to-End Test (staging environment)
  - [ ] Deploy to staging with Redis
  - [ ] Send 1000 alerts
  - [ ] Verify: groups created in Redis
  - [ ] Restart service
  - [ ] Verify: groups restored from Redis
  - [ ] Monitor metrics: `alert_history_business_grouping_storage_*`

### 11.5 Performance Benchmarks Validation

- [ ] **Task 11.5.1**: Валидировать performance targets
  - [ ] Store: <2ms ✅ (actual: ?)
  - [ ] Load: <1ms ✅ (actual: ?)
  - [ ] LoadAll (1K): <100ms ✅ (actual: ?)
  - [ ] Document results в `PERFORMANCE_REPORT.md`

### 11.6 Completion Report

- [ ] **Task 11.6.1**: Создать `COMPLETION_REPORT_TN125.md`
  - [ ] Summary (what was implemented)
  - [ ] Metrics (coverage, performance, LOC)
  - [ ] Quality grade (target: A+)
  - [ ] Known limitations
  - [ ] Next steps (TN-126 Inhibition Rules Engine)

### 11.7 Code Review

- [ ] **Task 11.7.1**: Self-review checklist
  - [ ] All methods have comprehensive godoc
  - [ ] Error handling follows project patterns (TN-124 reference)
  - [ ] Thread-safety guaranteed (sync.RWMutex, Redis atomicity)
  - [ ] Metrics integrated
  - [ ] Logging structured (slog with context)
  - [ ] Tests comprehensive (90%+ coverage)
  - [ ] Benchmarks passing targets
  - [ ] Documentation complete (README, runbook, migration guide)

---

## 📈 Success Criteria (150% Quality)

### Baseline (100%)
- [x] GroupStorage interface defined ✅
- [ ] RedisGroupStorage реализован
- [ ] MemoryGroupStorage реализован
- [ ] Automatic fallback/recovery
- [ ] DefaultGroupManager integration
- [ ] TTL management в Redis
- [ ] LoadAll() для восстановления state
- [ ] 80%+ test coverage
- [ ] 6 Prometheus metrics
- [ ] HTTP health endpoint integration

### 150% Enhancements (сверх baseline)
- [ ] **Optimistic locking**: Version-based concurrent update protection
- [ ] **Redis pipelining**: batch operations для >2x latency improvement
- [ ] **Parallel loading**: LoadAll() с goroutines (<100ms для 1K groups)
- [ ] **Retry logic**: exponential backoff для transient Redis errors
- [ ] **90%+ test coverage** (vs 80%)
- [ ] **Comprehensive benchmarks**: all operations с targets
- [ ] **Advanced metrics**: latency histograms (p50, p95, p99)
- [ ] **500+ line README**: с примерами и runbook
- [ ] **Production patterns**: circuit breaker, timeout, context cancellation
- [ ] **Integration tests**: multi-replica scenarios, Redis failure simulation

### Performance Targets (150%)

| Metric | Baseline Target | 150% Target | Achieved |
|--------|-----------------|-------------|----------|
| Store() | <5ms | <2ms | ? |
| Load() | <5ms | <1ms | ? |
| LoadAll() (1K) | <200ms | <100ms | ? |
| Test coverage | 80% | 90% | ? |
| Code quality | A | A+ | ? |

---

## 🚀 Blocked Tasks (разблокируются после TN-125)

После завершения TN-125 будут разблокированы:
- **TN-126**: Inhibition Rule Parser (requires persistent storage patterns)
- **TN-133**: Notification Scheduler (requires distributed group access)
- **TN-097**: HPA configuration (requires distributed state for scaling)

---

## 📝 Notes & Decisions

### Design Decisions
1. **Redis primary, Memory fallback** - balance persistence and availability
2. **JSON serialization** - human-readable, debuggable, Alertmanager compatible
3. **TTL-based cleanup** - prevent memory leaks, automatic expiration
4. **Optimistic locking** - Version field for concurrent update protection
5. **Parallel LoadAll** - 10 goroutines for <100ms target with 1K groups

### Technical Debt
- **Protobuf serialization**: future optimization (smaller size, faster)
- **Redis Cluster support**: future for >10K groups
- **Distributed locks**: Redis SET NX for strict consistency (future)

### Dependencies Validation ✅

- **TN-123 (AlertGroupManager)**: ✅ 183.6% MERGED (manager.go:121 mentions TN-125)
- **TN-124 (Group Timers)**: ✅ 152.6% MERGED (redis_timer_storage.go pattern reference)
- **TN-016 (Redis Cache)**: ✅ 100% COMPLETE (cache.RedisCache ready)

---

## 📅 Timeline (2 weeks)

| Week | Tasks | Deliverables |
|------|-------|--------------|
| Week 1 (Days 1-7) | Phase 1-6 | Interfaces, RedisGroupStorage, MemoryGroupStorage, StorageManager, Integration, Metrics |
| Week 2 (Days 8-14) | Phase 7-11 | Testing (90%+), Benchmarks, Integration Tests, Documentation, Validation |

**Target Completion**: 2025-11-18 (2 weeks from 2025-11-04)

**Estimated Effort**: ~35 hours for 150% quality (vs 22 hours baseline 100%)

---

## ✅ Completion Checklist

Before marking TN-125 as COMPLETE, verify:
- [ ] All 85 tasks completed
- [ ] 90%+ test coverage achieved
- [ ] All benchmarks pass performance targets
- [ ] No race conditions detected
- [ ] golangci-lint Grade A+
- [ ] 6 Prometheus metrics operational
- [ ] Integration tests passing (multi-replica, Redis failure, restart recovery)
- [ ] Documentation complete (README 500+ lines, runbook, migration guide)
- [ ] Staging validation passed
- [ ] COMPLETION_REPORT created
- [ ] TN-126 dependencies unblocked

**Target Quality**: **150%** (A+ grade, production-ready)
