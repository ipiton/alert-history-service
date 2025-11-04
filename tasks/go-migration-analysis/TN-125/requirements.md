# TN-125: Group Storage (Redis Backend, Distributed State)

**Дата создания**: 2025-11-04
**Статус**: 🟡 IN PROGRESS (TN-123 ✅, TN-124 ✅ COMPLETE)
**Приоритет**: 🔴 CRITICAL
**Target Quality**: 150%
**ID**: 10706364

---

## 1. Executive Summary

**TN-125** реализует **distributed storage для alert groups** с использованием Redis как backend, обеспечивая:
- **Persistent storage**: сохранение состояния групп алертов между рестартами
- **Distributed state**: синхронизация состояния между несколькими репликами сервиса
- **High Availability**: graceful degradation с fallback на in-memory storage
- **TTL management**: автоматическая очистка истекших групп
- **Optimistic locking**: предотвращение race conditions в distributed environment

### Критическая важность

TN-125 завершает **Alert Grouping System** (Модуль 1 из Phase A):
- ✅ **TN-121**: Grouping Configuration Parser (config-driven grouping)
- ✅ **TN-122**: Group Key Generator (FNV-1a hash-based keys)
- ✅ **TN-123**: Alert Group Manager (lifecycle management, 183.6% quality)
- ✅ **TN-124**: Group Wait/Interval Timers (Redis persistence, 152.6% quality)
- 🎯 **TN-125**: Group Storage (Redis Backend) ← THIS TASK

**Без TN-125:**
- ❌ Группы теряются при рестарте сервиса (state loss)
- ❌ Невозможно горизонтальное масштабирование (shared state problem)
- ❌ Каждый Pod создает свои группы (fragmentation)

**С TN-125:**
- ✅ Persistent state (survive restarts)
- ✅ Distributed state (multi-replica coordination)
- ✅ Horizontal scaling готовность (2-10 replicas with HPA)
- ✅ Полная замена Alertmanager grouping system

---

## 2. Обоснование задачи

### Проблема

**TN-123 (Alert Group Manager)** реализовал управление группами алертов с **in-memory хранилищем**:
- ✅ Создание/обновление/удаление групп
- ✅ Добавление алертов в группы
- ✅ Thread-safe concurrent access
- ✅ Fingerprint index для быстрого поиска
- ✅ Metrics и observability

**НО ОТСУТСТВУЕТ:**
- ❌ Persistence состояния групп (data loss при рестарте)
- ❌ Distributed state synchronization (multi-replica problem)
- ❌ Shared storage для horizontal scaling
- ❌ TTL-based automatic cleanup в Redis
- ❌ Optimistic locking для distributed updates

### Сценарий проблемы (без TN-125)

**Environment**: 3 replicas of alert-history service with HPA

1. **T=0**: Pod-1 получает алерт `HighCPU` от instance-1
   - Pod-1 создает группу `alertname=HighCPU` (in-memory)
   - Группа существует ТОЛЬКО в Pod-1

2. **T=10s**: Pod-2 получает алерт `HighCPU` от instance-2
   - Pod-2 НЕ ВИДИТ группу из Pod-1 (no shared state)
   - Pod-2 создает ДУБЛИРУЮЩУЮ группу `alertname=HighCPU`
   - **Результат**: Фрагментация (2 группы вместо 1)

3. **T=60s**: Pod-1 CRASHES (OOM, deploy, etc.)
   - Группа `alertname=HighCPU` из Pod-1 ПОТЕРЯНА
   - **Результат**: Data loss, нарушение group_wait logic

4. **T=120s**: HPA scales down (Pod-3 удален)
   - Группы из Pod-3 ПОТЕРЯНЫ
   - **Результат**: Нарушение Alertmanager compatibility

### Решение: GroupStorage с Redis Backend

Реализовать **GroupStorage interface** с двумя имплементациями:

```go
type GroupStorage interface {
    // Store сохраняет группу в storage (Redis/Memory)
    Store(ctx context.Context, group *AlertGroup) error

    // Load загружает группу по ключу
    Load(ctx context.Context, groupKey GroupKey) (*AlertGroup, error)

    // Delete удаляет группу
    Delete(ctx context.Context, groupKey GroupKey) error

    // ListKeys возвращает список всех ключей групп
    ListKeys(ctx context.Context) ([]GroupKey, error)

    // Size возвращает количество групп
    Size(ctx context.Context) (int, error)

    // LoadAll загружает все группы (для восстановления после рестарта)
    LoadAll(ctx context.Context) ([]*AlertGroup, error)
}
```

**Implementations:**
1. **RedisGroupStorage** (primary): Distributed storage с Redis
2. **MemoryGroupStorage** (fallback): In-memory storage при Redis failure

---

## 3. Пользовательский сценарий

### Use Case 1: Distributed State Synchronization

**Environment**: 3 replicas with Redis backend

**Сценарий:**
1. **T=0**: Pod-1 получает алерт `HighCPU` от instance-1
   - Pod-1 создает группу `alertname=HighCPU`
   - **GroupStorage.Store(ctx, group)** → Redis
   - Redis key: `group:alertname=HighCPU`
   - TTL: 24h (configurable)

2. **T=10s**: Pod-2 получает алерт `HighCPU` от instance-2
   - Pod-2 вызывает **GroupStorage.Load(ctx, "alertname=HighCPU")**
   - Redis возвращает существующую группу
   - Pod-2 добавляет alert в СУЩЕСТВУЮЩУЮ группу
   - **GroupStorage.Store(ctx, group)** → Redis (update)
   - **Результат**: ✅ ONE группа, 2 alerts (correct grouping)

3. **T=30s**: Pod-1 CRASHES
   - Группа остается в Redis
   - **Результат**: ✅ NO data loss

4. **T=60s**: New Pod-1' starts
   - Pod-1' загружает группы из Redis: **GroupStorage.LoadAll(ctx)**
   - Восстанавливает in-memory state
   - **Результат**: ✅ State restored, seamless recovery

---

### Use Case 2: Graceful Degradation (Redis Failure)

**Сценарий:**
1. **Normal Operation**: Redis healthy, all Pods use RedisGroupStorage

2. **T=100s**: Redis CONNECTION LOST
   - Pod-1 детектирует Redis failure (Ping() error)
   - Pod-1 переключается на **MemoryGroupStorage** (fallback)
   - **Metrics**: `alert_history_business_grouping_storage_fallback_total{reason="redis_error"}`
   - **Logging**: "Switched to in-memory storage due to Redis failure"

3. **T=100s-200s**: Degraded Mode
   - Pod-1, Pod-2, Pod-3 работают с in-memory storage
   - Группы ФРАГМЕНТИРОВАНЫ между Pods (как без TN-125)
   - **НО**: Alerting продолжает работать (no downtime)

4. **T=200s**: Redis RESTORED
   - Pod-1 детектирует Redis recovery (Ping() success)
   - Pod-1 переключается обратно на RedisGroupStorage
   - Pod-1 **НЕ ПЕРЕНОСИТ** in-memory группы в Redis (избегаем conflicts)
   - **Logging**: "Switched back to Redis storage"

5. **T=300s**: Normal Operation resumed
   - Новые группы создаются в Redis
   - Старые in-memory группы истекают через TTL
   - **Результат**: ✅ Graceful recovery

---

### Use Case 3: TTL-based Automatic Cleanup

**Сценарий:**
1. **T=0**: Группа `alertname=DiskFull` создана
   - Redis key: `group:alertname=DiskFull`
   - TTL: 24h (from config)

2. **T=12h**: Последний alert в группе resolved
   - GroupMetadata.State = "resolved"
   - GroupMetadata.ResolvedAt = T=12h
   - **GroupStorage.Store(ctx, group)** → обновляет в Redis

3. **T=24h**: TTL expires
   - Redis автоматически удаляет key
   - **Результат**: ✅ Automatic cleanup (no memory leak)

4. **T=24h+1s**: Pod пытается загрузить группу
   - **GroupStorage.Load(ctx, "alertname=DiskFull")** → ErrNotFound
   - Pod удаляет группу из in-memory cache
   - **Результат**: ✅ Synchronization between Redis and in-memory

---

## 4. Требования

### Функциональные требования

#### 4.1 Redis Storage Backend
- [x] **RedisGroupStorage** implementation с JSON serialization
- [x] Prefix для ключей: `group:{groupKey}` (namespace isolation)
- [x] TTL management: configurable per group (default: 24h)
- [x] Batch operations: SaveAll(), LoadAll() для efficiency
- [x] Atomic updates: optimistic locking через Version field
- [x] Pipeline support: Redis pipelining для batch writes (150% enhancement)

#### 4.2 In-Memory Fallback
- [x] **MemoryGroupStorage** implementation
- [x] Automatic fallback при Redis connection loss
- [x] Automatic recovery при Redis restoration
- [x] Metrics для fallback events
- [x] Graceful degradation (alerting продолжает работать)

#### 4.3 Storage Interface
- [x] **Store(ctx, group)**: save group to storage
- [x] **Load(ctx, groupKey)**: load group by key
- [x] **Delete(ctx, groupKey)**: delete group
- [x] **ListKeys(ctx)**: list all group keys
- [x] **Size(ctx)**: count groups
- [x] **LoadAll(ctx)**: load all groups (для startup recovery) (150% enhancement)
- [x] **Health check**: Ping() для мониторинга connection status

#### 4.4 Integration with DefaultGroupManager
- [x] **Constructor update**: accept GroupStorage parameter
- [x] **Store on create**: save group при создании
- [x] **Store on update**: save group при изменении (add alert, remove alert)
- [x] **Load on startup**: restore groups from Redis (LoadAll)
- [x] **Fallback strategy**: MemoryGroupStorage при Redis failure
- [x] **Lazy loading**: загружать группы по требованию (optional optimization)

#### 4.5 Optimistic Locking
- [x] **Version field**: GroupMetadata.Version (int64)
- [x] **Compare-and-swap**: Redis transaction с WATCH
- [x] **Conflict detection**: ErrVersionMismatch при concurrent updates
- [x] **Retry logic**: автоматический retry с exponential backoff (150% enhancement)

### Нефункциональные требования

#### 4.6 Performance
- [x] **Store latency**: <5ms (baseline), <2ms (150% target via pipelining)
- [x] **Load latency**: <5ms (baseline), <1ms (150% target)
- [x] **LoadAll latency**: <100ms для 1000 групп (parallel loading)
- [x] **TTL precision**: ±5s (Redis TTL accuracy)
- [x] **Memory overhead**: <100KB для RedisGroupStorage struct

#### 4.7 Reliability
- [x] **Zero data loss**: при Redis failure используем fallback
- [x] **Automatic recovery**: восстановление состояния при Redis restoration
- [x] **Connection pooling**: reuse Redis connections (from cache.RedisCache)
- [x] **Error handling**: typed errors (GroupNotFoundError, StorageError)
- [x] **Graceful shutdown**: flush pending writes при shutdown

#### 4.8 Scalability
- [x] **Horizontal scaling**: поддержка 2-10 replicas (HPA готовность)
- [x] **Redis cluster support**: готовность к Redis Cluster (future)
- [x] **Sharding готовность**: consistent hashing для group distribution (future)
- [x] **10K groups**: поддержка до 10,000 активных групп в Redis

#### 4.9 Observability
- [x] **6 Prometheus metrics**: storage operations, latency, errors, fallback
- [x] **Structured logging**: все операции с context и correlation IDs
- [x] **Health endpoint**: `/health` включает Redis connection status
- [x] **Metrics**: `alert_history_business_grouping_storage_*`

---

## 5. Критерии приёмки (150% Quality)

### Baseline (100%)
- [ ] GroupStorage interface определен
- [ ] RedisGroupStorage реализован с JSON serialization
- [ ] MemoryGroupStorage реализован для fallback
- [ ] DefaultGroupManager интегрирован с GroupStorage
- [ ] Automatic fallback/recovery при Redis failure
- [ ] TTL management для групп в Redis
- [ ] LoadAll() для восстановления state после рестарта
- [ ] 80%+ test coverage
- [ ] 6 Prometheus metrics
- [ ] HTTP health endpoint integration

### 150% Enhancements (сверх baseline)
- [ ] **Optimistic locking**: Version-based concurrent update protection
- [ ] **Redis pipelining**: batch operations для >2x latency improvement
- [ ] **Parallel loading**: LoadAll() с goroutines (<100ms для 1K groups)
- [ ] **Retry logic**: exponential backoff для transient Redis errors
- [ ] **Comprehensive testing**: 90%+ coverage, race tests, chaos tests
- [ ] **Benchmarks**: Store, Load, LoadAll, Lock operations
- [ ] **Advanced metrics**: latency histograms (p50, p95, p99)
- [ ] **Documentation**: 500+ line README с примерами и runbook
- [ ] **Production patterns**: circuit breaker, timeout, context cancellation
- [ ] **Integration tests**: multi-replica scenarios, Redis failure simulation

### Performance Targets (150%)

| Metric | Baseline Target | 150% Target | How to Achieve |
|--------|-----------------|-------------|----------------|
| Store() | <5ms | <2ms | Redis pipelining, async writes |
| Load() | <5ms | <1ms | Redis connection pooling, optimized deserialization |
| LoadAll() (1K groups) | <200ms | <100ms | Parallel goroutines, batch GET |
| TTL precision | ±10s | ±5s | Redis EXPIRE precision |
| Memory/storage | <200KB | <100KB | Lean struct, pointer reuse |
| Test coverage | 80% | 90% | Comprehensive edge cases, race tests |
| Code quality | A | A+ | golangci-lint, SOLID principles |

---

## 6. Зависимости

### Upstream (завершены, разблокируют TN-125)
- ✅ **TN-121**: Grouping Configuration Parser (GroupingConfig) - 150% ✅
- ✅ **TN-122**: Group Key Generator (GroupKey) - 200% ✅
- ✅ **TN-123**: Alert Group Manager (AlertGroup, DefaultGroupManager) - 183.6% ✅
- ✅ **TN-124**: Group Wait/Interval Timers (Redis persistence patterns) - 152.6% ✅
- ✅ **TN-016**: Redis Cache wrapper (cache.RedisCache) - 100% ✅
- ✅ **TN-021**: Prometheus metrics infrastructure - 100% ✅

### Downstream (блокированы TN-125, будут разблокированы)
- 🔒 **TN-126**: Inhibition Rule Parser (requires persistent storage patterns)
- 🔒 **TN-133**: Notification Scheduler (requires distributed group access)
- 🔒 **TN-097**: HPA configuration (requires distributed state for scaling)

### Integration Points
- **TN-123 DefaultGroupManager**: интеграция GroupStorage interface
- **TN-124 TimerManager**: coordination между timer persistence и group persistence
- **main.go**: initialization с Redis fallback chain

---

## 7. Технические риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| **Redis connection loss** | Высокая | Среднее | Automatic fallback на MemoryGroupStorage |
| **Version conflicts** (optimistic locking) | Средняя | Низкое | Retry logic с exponential backoff |
| **Serialization overhead** | Низкая | Низкое | Benchmarks, JSON optimization, protobuf future |
| **TTL precision issues** | Низкая | Низкое | Grace period (TTL+60s), CleanupExpiredGroups fallback |
| **Memory leak** (fallback mode) | Средняя | Высокое | CleanupExpiredGroups в MemoryGroupStorage |
| **Race conditions** (multi-replica) | Средняя | Высокое | Optimistic locking, distributed locks (future) |

---

## 8. Текущий статус

**Статус**: 🟡 READY TO START (dependencies completed)
**Блокеры**: НЕТ (TN-123 ✅ 183.6%, TN-124 ✅ 152.6%)
**Прогресс**: 0% → 150% (target)
**Приоритет**: 🔴 CRITICAL (completes Alert Grouping System)

### Dependency Status Validation ✅

1. **TN-123 (Alert Group Manager)**:
   - Status: ✅ MERGED to main (commit b19e3a4)
   - Quality: 183.6% (Grade A+)
   - Interface: AlertGroupManager готов к интеграции
   - **VALIDATED**: manager.go:121 упоминает "Version is used for optimistic locking (future: Redis storage in TN-125)"

2. **TN-124 (Group Timers)**:
   - Status: ✅ MERGED to main (commit c030f69)
   - Quality: 152.6% (Grade A+)
   - Pattern: RedisTimerStorage реализован → можем повторно использовать паттерны
   - **VALIDATED**: redis_timer_storage.go использует JSON + Redis pipelining + TTL

3. **TN-016 (Redis Cache)**:
   - Status: ✅ COMPLETE
   - Interface: cache.Cache, RedisCache готовы
   - **VALIDATED**: internal/infrastructure/cache/redis.go provides connection pooling

---

## 9. Временные рамки (оценка для 150% качества)

| Phase | Задача | Время | Статус |
|-------|--------|-------|--------|
| 1 | Requirements & Design Documentation | 3 часа | 🔲 Pending |
| 2 | GroupStorage Interface Definition | 1 час | 🔲 Pending |
| 3 | RedisGroupStorage Implementation | 4 часа | 🔲 Pending |
| 4 | MemoryGroupStorage Implementation | 2 часа | 🔲 Pending |
| 5 | Optimistic Locking & Retry Logic | 3 часа | 🔲 Pending |
| 6 | DefaultGroupManager Integration | 2 часа | 🔲 Pending |
| 7 | LoadAll() & Startup Recovery | 2 часа | 🔲 Pending |
| 8 | Prometheus Metrics (6 metrics) | 2 часа | 🔲 Pending |
| 9 | Comprehensive Testing (90%+ coverage) | 5 часов | 🔲 Pending |
| 10 | Benchmarks & Performance Optimization | 3 часа | 🔲 Pending |
| 11 | Integration Tests (multi-replica) | 3 часа | 🔲 Pending |
| 12 | Documentation (README, runbook) | 3 часа | 🔲 Pending |
| 13 | Validation & Production Readiness | 2 часа | 🔲 Pending |

**Итого**: ~35 часов для 150% качества (vs 22 часа baseline 100%)

---

## 10. Success Metrics

После завершения TN-125 мы сможем:
1. ✅ Сохранять группы алертов в Redis (persistent state)
2. ✅ Синхронизировать состояние между replicas (distributed state)
3. ✅ Восстанавливать группы после рестарта (LoadAll recovery)
4. ✅ Горизонтально масштабировать сервис (2-10 replicas with HPA)
5. ✅ Автоматически очищать истекшие группы (TTL-based cleanup)
6. ✅ Graceful degradation при Redis failure (fallback to memory)
7. ✅ Завершить Alert Grouping System (TN-121 to TN-125) - 100%
8. ✅ Разблокировать Inhibition Rules Engine (TN-126+)

**Target Quality**: **150%** (A+ grade, production-ready)

---

## 11. Compatibility Matrix

| Component | Version | Compatibility | Notes |
|-----------|---------|---------------|-------|
| Redis | 6.0+ | ✅ Required | JSON, TTL, Pipelining support |
| TN-123 (AlertGroupManager) | 183.6% | ✅ Full | GroupStorage interface integration |
| TN-124 (TimerManager) | 152.6% | ✅ Full | Shared Redis instance, consistent patterns |
| TN-016 (Redis Cache) | 100% | ✅ Full | Connection pooling, error handling |
| Alertmanager | v0.25+ | ✅ Compatible | Group persistence semantics |
| Kubernetes HPA | 2-10 replicas | ✅ Ready | Distributed state support |

---

## 12. Rollout Strategy

### Phase 1: Development & Testing (Week 1)
- [ ] Implement GroupStorage interface + RedisGroupStorage + MemoryGroupStorage
- [ ] 90%+ test coverage, benchmarks
- [ ] Integration tests с multi-replica scenarios

### Phase 2: Canary Deployment (Week 2)
- [ ] Deploy to staging environment (1 replica)
- [ ] Monitor metrics: `alert_history_business_grouping_storage_*`
- [ ] Validate LoadAll() recovery, fallback behavior

### Phase 3: Production Rollout (Week 3)
- [ ] Deploy to production (3 replicas with HPA 2-10)
- [ ] Monitor distributed state synchronization
- [ ] A/B test: compare with in-memory baseline (TN-123)

### Phase 4: Validation & Handoff (Week 4)
- [ ] Validate 150% quality criteria
- [ ] Performance benchmarks: <2ms Store, <1ms Load
- [ ] Documentation update: runbook, troubleshooting guide
- [ ] Unblock TN-126 (Inhibition Rules Engine)

**Target Completion**: 2025-11-18 (2 weeks from start)
