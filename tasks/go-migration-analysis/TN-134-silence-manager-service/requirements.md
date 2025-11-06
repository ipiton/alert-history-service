# TN-134: Silence Manager Service - Requirements

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-134
**Status**: 🟡 NOT STARTED
**Priority**: HIGH
**Estimated Effort**: 12-16 hours
**Dependencies**: TN-131 (Silence Data Models ✅), TN-132 (Silence Matcher Engine ✅), TN-133 (Silence Storage ✅)
**Blocks**: TN-135 (Silence API Endpoints), TN-136 (Silence UI Components)
**Target Quality**: 150% (Enterprise-Grade)
**Quality Reference**: TN-124 (152.6%), TN-129 (150%), TN-133 (152.7%)

---

## 📋 Executive Summary

Реализовать **enterprise-grade Silence Manager Service** для управления жизненным циклом silence rules с поддержкой:
- **Lifecycle Management**: Автоматическая активация/деактивация silences по времени
- **Background GC Worker**: Автоматическая очистка expired silences (TTL-based cleanup)
- **Alert Filtering Integration**: Проверка алертов на соответствие активным silences
- **Status Synchronization**: Автоматический пересчет статусов (pending→active→expired)
- **High Availability**: Thread-safe операции, graceful shutdown, метрики
- **Observability**: 8 Prometheus metrics, structured logging, health checks

### Business Value

| Ценность | Описание | Impact |
|----------|----------|--------|
| **Automated Lifecycle** | Автоматическая активация/экспирация без ручного управления | HIGH |
| **Memory Management** | Автоматическая очистка старых silences (prevent memory leak) | HIGH |
| **Performance** | Fast in-memory cache для активных silences (<100µs lookup) | HIGH |
| **Reliability** | Graceful degradation, zero downtime updates | MEDIUM |
| **Maintainability** | Centralized silence management, clear separation of concerns | MEDIUM |

---

## 🎯 Goals

### Primary Goals (Must Have - 150%)

1. ✅ **SilenceManager Interface** - Определить core interface
   ```go
   type SilenceManager interface {
       // Core Operations
       CreateSilence(ctx, *silencing.Silence) (*silencing.Silence, error)
       GetSilence(ctx, id string) (*silencing.Silence, error)
       UpdateSilence(ctx, *silencing.Silence) error
       DeleteSilence(ctx, id string) error
       ListSilences(ctx, filter) ([]*silencing.Silence, error)

       // Alert Integration
       IsAlertSilenced(ctx, alert *Alert) (bool, []string, error)
       GetActiveSilences(ctx) ([]*silencing.Silence, error)

       // Lifecycle Management
       Start(ctx) error
       Stop(ctx) error

       // Status
       GetStats(ctx) (*SilenceManagerStats, error)
   }
   ```

2. ✅ **DefaultSilenceManager Implementation**
   - Full CRUD operations через `SilenceRepository`
   - In-memory cache для активных silences (fast lookup)
   - Cache invalidation при создании/обновлении/удалении
   - Thread-safe concurrent operations (sync.RWMutex)
   - Integration с `SilenceMatcher` для alert filtering
   - Context-aware operations (cancellation, timeouts)

3. ✅ **Background GC Worker** (inspired by TN-124 TTL worker)
   - Periodic cleanup (default: 5 minutes interval)
   - Two-phase cleanup:
     - Phase 1: Expire active silences (status update: active→expired)
     - Phase 2: Delete old expired silences (retention: 24 hours)
   - Batch processing (max 1000 silences per run)
   - Graceful shutdown support
   - Metrics tracking (cleaned_count, duration)

4. ✅ **Status Synchronization Worker**
   - Periodic status refresh (default: 1 minute interval)
   - Update cache with newly activated silences (pending→active)
   - Remove expired silences from cache
   - Handle timezone changes
   - Log status transitions

5. ✅ **In-Memory Active Silence Cache**
   ```go
   type silenceCache struct {
       mu       sync.RWMutex
       silences map[string]*silencing.Silence  // ID → Silence
       byStatus map[silencing.SilenceStatus][]string  // Status → IDs
       lastSync time.Time
   }
   ```
   - Fast lookup O(1) by ID
   - Fast filtering O(N) by status
   - Automatic refresh on status changes
   - Concurrency-safe operations

6. ✅ **Prometheus Metrics** (8 metrics, inspired by TN-129)
   ```go
   // Operations
   alert_history_silence_operations_total{operation, status}  // Counter
   alert_history_silence_operation_duration_seconds{operation}  // Histogram

   // Active Silences
   alert_history_silence_active_total{status}  // Gauge (pending/active/expired)

   // Alert Filtering
   alert_history_silence_alert_checks_total{result}  // Counter (silenced/not_silenced)
   alert_history_silence_alert_check_duration_seconds  // Histogram

   // GC Worker
   alert_history_silence_gc_runs_total{phase}  // Counter (expire/delete)
   alert_history_silence_gc_cleaned_total{phase}  // Counter
   alert_history_silence_gc_duration_seconds{phase}  // Histogram
   ```

### Secondary Goals (Nice to Have)

- Bulk operations support (BulkDelete, BulkExpire)
- WebSocket notifications for silence status changes
- Silence templates (reusable patterns)
- Silence approval workflow (pending_approval status)

---

## 📐 Functional Requirements

### FR-1: Core CRUD Operations

**CreateSilence**:
- Validate silence via `silence.Validate()`
- Save to PostgreSQL via `repository.CreateSilence()`
- Add to in-memory cache if status == "active"
- Record metrics
- Log operation

**GetSilence**:
- Try cache first (O(1) lookup)
- Fallback to PostgreSQL if cache miss
- Update cache entry if needed

**UpdateSilence**:
- Validate silence
- Update in PostgreSQL via `repository.UpdateSilence()`
- Invalidate cache entry
- Re-add to cache if new status == "active"

**DeleteSilence**:
- Delete from PostgreSQL via `repository.DeleteSilence()`
- Remove from cache
- Record metrics

**ListSilences**:
- Support filtering by status, creator, time range
- Use cache for "active" filter (fast path)
- Query PostgreSQL for all other filters

**Performance Targets**:
- CreateSilence: <15ms (db write + cache update)
- GetSilence (cached): <100µs (in-memory lookup)
- GetSilence (uncached): <5ms (db read)
- UpdateSilence: <20ms (db write + cache invalidation)
- DeleteSilence: <10ms (db delete + cache removal)

---

### FR-2: Alert Filtering Integration

**IsAlertSilenced**:
```go
func (sm *DefaultSilenceManager) IsAlertSilenced(
    ctx context.Context,
    alert *Alert,
) (bool, []string, error)
```

**Logic**:
1. Get active silences from cache (fast path)
2. Iterate through silences (early exit on first match)
3. Use `SilenceMatcher.Matches()` to check if alert matches silence
4. Return (true, [matched_silence_ids]) if matched
5. Return (false, nil) if no matches

**Performance Target**: <500µs for 100 active silences

**Error Handling**:
- Graceful degradation if cache unavailable (fallback to PostgreSQL)
- Return `(false, nil, err)` on critical errors
- Log warnings for non-critical errors

**Integration Point**:
- Called by `AlertProcessor` before publishing alerts
- Silenced alerts: skip publishing, record metric
- Non-silenced alerts: continue normal processing

---

### FR-3: Background GC Worker

**Inspired by**: TN-124 (TTL Cleanup Worker), TN-129 (Cleanup Worker)

**Architecture**:
```go
type gcWorker struct {
    repo        SilenceRepository
    cache       *silenceCache
    interval    time.Duration  // Default: 5m
    retention   time.Duration  // Default: 24h
    batchSize   int           // Default: 1000
    logger      *slog.Logger
    metrics     *SilenceMetrics
    stopCh      chan struct{}
    doneCh      chan struct{}
    wg          sync.WaitGroup
}
```

**Lifecycle**:
1. **Start** (via `SilenceManager.Start()`):
   - Create ticker (5m interval)
   - Spawn goroutine
   - Run immediate cleanup on startup
   - Wait for tick or stop signal

2. **Run Cleanup** (two-phase):
   - **Phase 1: Expire Active Silences**
     ```go
     // Find silences where ends_at < NOW AND status = 'active'
     // UPDATE status = 'expired' (batch: 1000)
     count, err := repo.ExpireSilences(ctx, time.Now(), false)
     ```
   - **Phase 2: Delete Old Expired**
     ```go
     // Find silences where ends_at < NOW-24h AND status = 'expired'
     // DELETE FROM silences (batch: 1000)
     before := time.Now().Add(-retention)
     count, err := repo.ExpireSilences(ctx, before, true)
     ```

3. **Stop** (via `SilenceManager.Stop()`):
   - Send stop signal: `close(stopCh)`
   - Wait for goroutine: `<-doneCh`
   - Timeout: 30s

**Metrics**:
- `silence_gc_runs_total{phase="expire|delete"}` - Counter
- `silence_gc_cleaned_total{phase="expire|delete"}` - Counter
- `silence_gc_duration_seconds{phase="expire|delete"}` - Histogram

**Error Handling**:
- Log errors, continue processing
- Do NOT stop worker on errors
- Exponential backoff on repeated failures (max 5m)

---

### FR-4: Status Synchronization Worker

**Purpose**: Keep in-memory cache synchronized with database

**Architecture**:
```go
type syncWorker struct {
    repo      SilenceRepository
    cache     *silenceCache
    interval  time.Duration  // Default: 1m
    logger    *slog.Logger
    stopCh    chan struct{}
    doneCh    chan struct{}
}
```

**Logic** (every 1 minute):
1. **Refresh Active Silences**:
   ```go
   // Query database for all active silences
   filter := SilenceFilter{Statuses: []silencing.SilenceStatus{"active"}}
   silences, err := repo.ListSilences(ctx, filter)

   // Rebuild cache
   cache.mu.Lock()
   cache.silences = buildSilenceMap(silences)
   cache.byStatus = buildStatusIndex(silences)
   cache.lastSync = time.Now()
   cache.mu.Unlock()
   ```

2. **Remove Expired from Cache**:
   - Calculate status for each cached silence
   - Remove if status changed to "expired"

3. **Add Newly Activated**:
   - Find silences where status changed to "active" (pending→active)
   - Add to cache

**Performance Target**: <50ms for 1000 active silences

**Metrics**:
- `silence_sync_runs_total` - Counter
- `silence_sync_duration_seconds` - Histogram
- `silence_sync_added_total` - Counter
- `silence_sync_removed_total` - Counter

---

### FR-5: Thread Safety

**Requirements** (inspired by TN-124, TN-129):
- All public methods must be thread-safe
- Use `sync.RWMutex` for cache access:
  - `RLock()` for reads (GetSilence, IsAlertSilenced)
  - `Lock()` for writes (CreateSilence, UpdateSilence, DeleteSilence)
- No data races (verified by `go test -race`)
- Goroutine leak prevention (WaitGroup tracking)

**Concurrency Patterns**:
```go
// Read-heavy operations (GetSilence)
sm.cache.mu.RLock()
silence, found := sm.cache.silences[id]
sm.cache.mu.RUnlock()

// Write operations (CreateSilence)
sm.cache.mu.Lock()
sm.cache.silences[id] = silence
sm.cache.byStatus[silence.Status] = append(...)
sm.cache.mu.Unlock()
```

---

### FR-6: Graceful Shutdown

**Inspired by**: TN-124 Timer Manager

**Shutdown Sequence** (via `SilenceManager.Stop()`):
1. Set shutdown flag (reject new operations)
2. Stop all workers (GC, Sync)
3. Wait for in-flight operations (WaitGroup)
4. Close repository connections (if needed)
5. Return within 30s timeout

**Implementation**:
```go
func (sm *DefaultSilenceManager) Stop(ctx context.Context) error {
    sm.logger.Info("Stopping silence manager")
    startTime := time.Now()

    // Set shutdown flag
    sm.mu.Lock()
    sm.shutdown = true
    sm.mu.Unlock()

    // Stop workers
    close(sm.gcWorker.stopCh)
    close(sm.syncWorker.stopCh)

    // Wait for workers
    done := make(chan struct{})
    go func() {
        sm.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        sm.logger.Info("Silence manager stopped", "duration", time.Since(startTime))
        return nil
    case <-ctx.Done():
        return fmt.Errorf("shutdown timeout: %w", ctx.Err())
    }
}
```

---

### FR-7: Error Handling

**Error Types** (extend `silence_repository_errors.go`):
```go
var (
    ErrManagerNotStarted = errors.New("silence manager not started")
    ErrManagerShutdown   = errors.New("silence manager is shutting down")
    ErrCacheNotAvailable = errors.New("cache not available")
    ErrInvalidAlert      = errors.New("invalid alert")
)
```

**Error Propagation**:
- Repository errors: wrap and propagate (`fmt.Errorf("repo operation: %w", err)`)
- Matcher errors: log warning, return false (graceful degradation)
- Cache errors: fallback to database, log warning

**Retry Logic**:
- No automatic retries (client responsibility)
- GC/Sync workers: retry on next tick
- Context cancellation: return immediately

---

### FR-8: Configuration

**Config Struct**:
```go
type SilenceManagerConfig struct {
    // GC Worker
    GCInterval    time.Duration  // Default: 5m
    GCRetention   time.Duration  // Default: 24h
    GCBatchSize   int           // Default: 1000

    // Sync Worker
    SyncInterval  time.Duration  // Default: 1m

    // Cache
    CacheEnabled  bool          // Default: true
    CacheTTL      time.Duration  // Default: 5m

    // Shutdown
    ShutdownTimeout time.Duration  // Default: 30s
}

func DefaultSilenceManagerConfig() SilenceManagerConfig {
    return SilenceManagerConfig{
        GCInterval:      5 * time.Minute,
        GCRetention:     24 * time.Hour,
        GCBatchSize:     1000,
        SyncInterval:    1 * time.Minute,
        CacheEnabled:    true,
        CacheTTL:        5 * time.Minute,
        ShutdownTimeout: 30 * time.Second,
    }
}
```

**Environment Variables** (12-factor app):
```bash
SILENCE_GC_INTERVAL=5m
SILENCE_GC_RETENTION=24h
SILENCE_SYNC_INTERVAL=1m
SILENCE_CACHE_ENABLED=true
```

---

## 🎯 Non-Functional Requirements

### NFR-1: Performance

| Operation | Target | Measured | Status |
|-----------|--------|----------|--------|
| CreateSilence | <15ms | TBD | 🔄 |
| GetSilence (cached) | <100µs | TBD | 🔄 |
| GetSilence (uncached) | <5ms | TBD | 🔄 |
| UpdateSilence | <20ms | TBD | 🔄 |
| DeleteSilence | <10ms | TBD | 🔄 |
| IsAlertSilenced (100 silences) | <500µs | TBD | 🔄 |
| GC Worker (1000 silences) | <2s | TBD | 🔄 |
| Sync Worker (1000 silences) | <50ms | TBD | 🔄 |

### NFR-2: Scalability

- Support 10K+ active silences in cache (<100MB RAM)
- Handle 100K+ total silences in PostgreSQL
- Process 1000 alerts/second with silence checking
- GC worker handles 1M+ silences (batched cleanup)

### NFR-3: Reliability

- Zero downtime during deployment (graceful shutdown)
- No goroutine leaks (WaitGroup tracking)
- No memory leaks (cache size limits)
- Graceful degradation on PostgreSQL failures
- Retry logic for transient errors

### NFR-4: Observability

- **Metrics**: 8 Prometheus metrics (operations, cache, GC, sync)
- **Logging**: Structured logging (slog) at INFO/WARN/ERROR levels
- **Tracing**: Context propagation for distributed tracing
- **Health Checks**: `/healthz` endpoint integration

### NFR-5: Security

- No sensitive data in logs (silence comments may contain secrets)
- SQL injection prevention (parameterized queries via pgx)
- Context cancellation support (prevent DoS)
- Rate limiting (client responsibility)

---

## 📊 Acceptance Criteria

### AC-1: Core Operations
- [ ] All CRUD operations implemented and tested
- [ ] In-memory cache working correctly
- [ ] Cache invalidation on updates/deletes
- [ ] IsAlertSilenced returns correct results
- [ ] GetActiveSilences returns only active silences

### AC-2: Background Workers
- [ ] GC worker runs periodically (5m interval)
- [ ] GC worker expires active silences (phase 1)
- [ ] GC worker deletes old expired (phase 2)
- [ ] Sync worker refreshes cache (1m interval)
- [ ] Both workers stop gracefully on shutdown

### AC-3: Performance
- [ ] GetSilence (cached): <100µs (benchmark verified)
- [ ] IsAlertSilenced: <500µs for 100 silences
- [ ] GC worker: <2s for 1000 silences
- [ ] Sync worker: <50ms for 1000 silences
- [ ] Zero allocations in hot paths (benchmark verified)

### AC-4: Testing
- [ ] 40+ unit tests (80%+ coverage)
- [ ] 10+ integration tests (PostgreSQL + cache)
- [ ] 8+ benchmark tests (performance validation)
- [ ] 5+ concurrent tests (race detector)
- [ ] 100% tests passing (`go test -v -race`)

### AC-5: Documentation
- [ ] Comprehensive README.md (800+ lines)
- [ ] Godoc comments for all public types/methods
- [ ] Architecture diagrams (component, sequence)
- [ ] Integration guide (main.go example)
- [ ] Metrics guide (Prometheus, Grafana)

### AC-6: Integration
- [ ] Integrated into main.go
- [ ] Connected to AlertProcessor
- [ ] Metrics registered in MetricsRegistry
- [ ] Health checks implemented
- [ ] Configuration via environment variables

---

## 🔗 Dependencies

### Upstream Dependencies (Required)
- ✅ **TN-131**: Silence Data Models - `silencing.Silence`, `silencing.Matcher`
- ✅ **TN-132**: Silence Matcher Engine - `SilenceMatcher` interface
- ✅ **TN-133**: Silence Storage - `SilenceRepository` interface

### Downstream Consumers (Blocked by TN-134)
- ⏳ **TN-135**: Silence API Endpoints - REST API for silence management
- ⏳ **TN-136**: Silence UI Components - Dashboard widgets

### External Dependencies
- `github.com/jackc/pgx/v5/pgxpool` - PostgreSQL connection pool
- `github.com/google/uuid` - UUID generation
- `github.com/prometheus/client_golang` - Metrics
- `log/slog` - Structured logging
- `context` - Cancellation, timeouts
- `sync` - Thread safety (RWMutex, WaitGroup)
- `time` - Ticker, timers

---

## 📈 Success Metrics

### Quality Metrics (150% Target)
- **Test Coverage**: 85%+ (target: 80%, +5%)
- **Performance**: 2x better than targets
- **Documentation**: 800+ lines README
- **Zero Technical Debt**: No TODOs, no hacks
- **Zero Breaking Changes**: Backward compatible

### Business Metrics
- **Silence Creation**: <15ms P99 latency
- **Alert Filtering**: <500µs for 100 silences
- **Memory Usage**: <100MB for 10K silences
- **CPU Usage**: <1% idle, <5% under load
- **Uptime**: 99.9% (no crashes, graceful degradation)

---

## 🚀 Implementation Strategy

### Phase 1: Interface & Core (3h)
- Define `SilenceManager` interface
- Implement `DefaultSilenceManager` struct
- Implement CRUD operations (CreateSilence, GetSilence, etc.)
- Add in-memory cache
- Unit tests for CRUD

### Phase 2: Alert Filtering (2h)
- Implement `IsAlertSilenced()`
- Integrate with `SilenceMatcher`
- Add cache-based fast path
- Unit tests for filtering

### Phase 3: GC Worker (2h)
- Implement `gcWorker` struct
- Add two-phase cleanup (expire + delete)
- Integrate with `SilenceRepository`
- Unit tests for GC

### Phase 4: Sync Worker (2h)
- Implement `syncWorker` struct
- Add cache refresh logic
- Handle status transitions
- Unit tests for sync

### Phase 5: Lifecycle & Shutdown (1.5h)
- Implement `Start()` and `Stop()` methods
- Add graceful shutdown logic
- WaitGroup tracking
- Integration tests

### Phase 6: Metrics & Observability (1.5h)
- Add 8 Prometheus metrics
- Integrate with `MetricsRegistry`
- Add structured logging
- Metrics documentation

### Phase 7: Integration (2h)
- Integrate into main.go
- Connect to AlertProcessor
- Add configuration loading
- Health checks

### Phase 8: Testing & Benchmarks (3h)
- 40+ unit tests
- 10+ integration tests
- 8+ benchmarks
- Race detector validation
- Coverage report

### Phase 9: Documentation (2h)
- Comprehensive README.md (800+ lines)
- Architecture diagrams
- Integration guide
- Metrics guide
- Completion report

**Total Estimated Effort**: 19 hours (buffer: +3h over 16h target)

---

## 📚 References

### Internal Documentation
- [TN-131: Silence Data Models](../TN-131-silence-data-models/requirements.md)
- [TN-132: Silence Matcher Engine](../TN-132-silence-matcher-engine/requirements.md)
- [TN-133: Silence Storage](../TN-133-silence-storage-postgres/requirements.md)
- [TN-124: Group Timers (Reference for workers)](../TN-124/requirements.md)
- [TN-129: State Manager (Reference for cleanup)](../TN-129-inhibition-state-manager/requirements.md)

### External References
- [Alertmanager API v2 - Silences](https://prometheus.io/docs/alerting/latest/clients/)
- [Go Context Package](https://pkg.go.dev/context)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)

---

**Version**: 1.0
**Author**: TN-134 Implementation Team
**Last Updated**: 2025-11-06
**Status**: Ready for Implementation 🚀
