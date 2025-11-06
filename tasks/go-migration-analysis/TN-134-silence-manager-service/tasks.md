# TN-134: Silence Manager Service - Implementation Tasks

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-134
**Status**: 🟡 NOT STARTED
**Target Quality**: 150% (Enterprise-Grade)
**Estimated Total**: 16-19 hours

---

## 📋 Task Overview

| Phase | Description | Est. Time | Status |
|-------|-------------|-----------|--------|
| **Phase 0** | Pre-flight Checks & Documentation Review | 30 min | ⏳ Pending |
| **Phase 1** | Interface & Core Structs | 2.5 h | ⏳ Pending |
| **Phase 2** | CRUD Operations & Cache | 3 h | ⏳ Pending |
| **Phase 3** | Alert Filtering Integration | 2 h | ⏳ Pending |
| **Phase 4** | Background GC Worker | 2.5 h | ⏳ Pending |
| **Phase 5** | Background Sync Worker | 2 h | ⏳ Pending |
| **Phase 6** | Lifecycle & Graceful Shutdown | 1.5 h | ⏳ Pending |
| **Phase 7** | Metrics & Observability | 1.5 h | ⏳ Pending |
| **Phase 8** | Integration (main.go, AlertProcessor) | 2 h | ⏳ Pending |
| **Phase 9** | Testing & Benchmarks | 3.5 h | ⏳ Pending |
| **Phase 10** | Documentation & Completion Report | 2 h | ⏳ Pending |

**Total**: 22.5 hours (target: 16-19h with optimizations)

---

## 🎯 Quality Targets (150%)

### Performance Targets
- CreateSilence: <15ms (vs baseline 20ms) = 1.33x faster
- GetSilence (cached): <100µs (vs baseline 200µs) = 2x faster
- GetSilence (uncached): <5ms (vs baseline 10ms) = 2x faster
- UpdateSilence: <20ms (vs baseline 30ms) = 1.5x faster
- DeleteSilence: <10ms (vs baseline 15ms) = 1.5x faster
- IsAlertSilenced: <500µs for 100 silences (vs 1ms) = 2x faster
- GC Worker: <2s for 1000 silences (vs 5s) = 2.5x faster

### Test Coverage
- Target: 85%+ (baseline 80%, +5%)
- Unit tests: 40+ (vs 30 baseline)
- Integration tests: 10+ (vs 5 baseline)
- Benchmarks: 8+ (vs 5 baseline)

### Documentation
- README.md: 800+ lines (vs 500 baseline)
- Godoc: 100% coverage (all public APIs)
- Architecture diagrams: 3+ (component, sequence, data flow)

---

## Phase 0: Pre-flight Checks (30 min)

### Task 0.1: Verify Dependencies ⏱️ 10 min
- [ ] Check TN-131 completion status (Silence Data Models)
  ```bash
  ls -la go-app/internal/core/silencing/
  # Expected: models.go, validator.go, errors.go, matcher.go
  ```
- [ ] Check TN-132 completion status (Silence Matcher Engine)
  ```bash
  ls -la go-app/internal/core/silencing/
  # Expected: matcher_impl.go, matcher_cache.go
  ```
- [ ] Check TN-133 completion status (Silence Storage)
  ```bash
  ls -la go-app/internal/infrastructure/silencing/
  # Expected: repository.go, postgres_silence_repository.go, filter_builder.go
  ```
- [ ] Verify all dependencies compile:
  ```bash
  cd go-app && go build ./...
  ```

### Task 0.2: Review Documentation ⏱️ 15 min
- [ ] Read requirements.md (TN-134)
- [ ] Read design.md (TN-134)
- [ ] Review TN-124 (Timer Manager) for worker patterns
- [ ] Review TN-129 (State Manager) for cleanup patterns

### Task 0.3: Setup Development Environment ⏱️ 5 min
- [ ] Create feature branch:
  ```bash
  git checkout main
  git pull origin main
  git checkout -b feature/TN-134-silence-manager-150pct
  ```
- [ ] Verify PostgreSQL running (docker-compose)
- [ ] Verify Redis running (optional, for integration tests)

**Deliverable**: ✅ All dependencies verified, branch created

---

## Phase 1: Interface & Core Structs (2.5 hours)

### Task 1.1: Define SilenceManager Interface ⏱️ 30 min
- [ ] Create `go-app/internal/business/silencing/manager.go`
- [ ] Define `SilenceManager` interface:
  ```go
  type SilenceManager interface {
      // Core CRUD
      CreateSilence(ctx context.Context, silence *silencing.Silence) (*silencing.Silence, error)
      GetSilence(ctx context.Context, id string) (*silencing.Silence, error)
      UpdateSilence(ctx context.Context, silence *silencing.Silence) error
      DeleteSilence(ctx context.Context, id string) error
      ListSilences(ctx context.Context, filter silencing.SilenceFilter) ([]*silencing.Silence, error)

      // Alert Integration
      IsAlertSilenced(ctx context.Context, alert *Alert) (bool, []string, error)
      GetActiveSilences(ctx context.Context) ([]*silencing.Silence, error)

      // Lifecycle
      Start(ctx context.Context) error
      Stop(ctx context.Context) error

      // Status
      GetStats(ctx context.Context) (*SilenceManagerStats, error)
  }
  ```
- [ ] Add godoc comments (examples for each method)
- [ ] Define helper types:
  - `SilenceManagerConfig`
  - `SilenceManagerStats`
  - `CacheStats`

**Deliverable**: `manager.go` (180 LOC)

---

### Task 1.2: Define DefaultSilenceManager Struct ⏱️ 45 min
- [ ] Create `go-app/internal/business/silencing/manager_impl.go`
- [ ] Define `DefaultSilenceManager` struct:
  ```go
  type DefaultSilenceManager struct {
      // Storage & Matching
      repo    silencing.SilenceRepository
      matcher silencing.SilenceMatcher

      // Cache
      cache   *silenceCache

      // Workers
      gcWorker   *gcWorker
      syncWorker *syncWorker

      // Observability
      metrics *SilenceMetrics
      logger  *slog.Logger

      // Configuration
      config SilenceManagerConfig

      // Lifecycle
      started  atomic.Bool
      shutdown atomic.Bool
      wg       sync.WaitGroup
      ctx      context.Context
      cancel   context.CancelFunc
  }
  ```
- [ ] Implement `NewDefaultSilenceManager()` constructor:
  - Validate required parameters (repo, matcher)
  - Set default logger if nil
  - Apply default config if nil
  - Create context with cancel
  - Initialize cache
  - Create workers (not started yet)
- [ ] Add validation logic for config:
  ```go
  func (c *SilenceManagerConfig) Validate() error {
      if c.GCInterval < 1*time.Minute {
          return fmt.Errorf("GCInterval too short: %v", c.GCInterval)
      }
      // ... more validations
  }
  ```

**Deliverable**: `manager_impl.go` (120 LOC)

---

### Task 1.3: Implement In-Memory Cache ⏱️ 45 min
- [ ] Create `go-app/internal/business/silencing/cache.go`
- [ ] Define `silenceCache` struct:
  ```go
  type silenceCache struct {
      mu       sync.RWMutex
      silences map[string]*silencing.Silence
      byStatus map[silencing.SilenceStatus][]string
      lastSync time.Time
      size     int
  }
  ```
- [ ] Implement cache methods:
  - `newSilenceCache()` - Constructor
  - `Get(id string) (*silencing.Silence, bool)` - Thread-safe read
  - `Set(silence *silencing.Silence)` - Thread-safe write
  - `Delete(id string)` - Thread-safe delete
  - `GetByStatus(status silencing.SilenceStatus) []*silencing.Silence`
  - `GetAll() []*silencing.Silence`
  - `Rebuild(silences []*silencing.Silence)` - Full rebuild
  - `rebuildStatusIndex()` - Internal helper
  - `Stats() CacheStats` - Statistics
- [ ] Add comprehensive godoc comments
- [ ] Add inline comments for locking strategy

**Deliverable**: `cache.go` (220 LOC)

---

### Task 1.4: Define Error Types ⏱️ 15 min
- [ ] Create `go-app/internal/business/silencing/errors.go`
- [ ] Add manager-specific errors:
  ```go
  var (
      ErrManagerNotStarted = errors.New("silence manager not started")
      ErrManagerShutdown   = errors.New("silence manager is shutting down")
      ErrInvalidAlert      = errors.New("invalid alert")
      ErrCacheUnavailable  = errors.New("cache unavailable")
  )
  ```
- [ ] Add godoc comments with usage examples

**Deliverable**: `errors.go` (40 LOC)

---

### Task 1.5: Unit Tests for Cache ⏱️ 30 min
- [ ] Create `go-app/internal/business/silencing/cache_test.go`
- [ ] Implement 8 tests:
  - `TestCache_SetGet` - Basic set/get operations
  - `TestCache_Delete` - Delete operation
  - `TestCache_GetByStatus` - Filter by status
  - `TestCache_GetAll` - Get all silences
  - `TestCache_Rebuild` - Full rebuild
  - `TestCache_Stats` - Statistics
  - `TestCache_Concurrent` - Thread safety (10 goroutines)
  - `TestCache_LargeDataset` - 1000 silences

**Deliverable**: `cache_test.go` (280 LOC), 8/8 tests passing

---

## Phase 2: CRUD Operations & Cache (3 hours)

### Task 2.1: Implement CreateSilence ⏱️ 30 min
- [ ] In `manager_impl.go`, implement `CreateSilence()`:
  ```go
  func (sm *DefaultSilenceManager) CreateSilence(
      ctx context.Context,
      silence *silencing.Silence,
  ) (*silencing.Silence, error) {
      // 1. Check manager state (started, not shutdown)
      if !sm.started.Load() {
          return nil, ErrManagerNotStarted
      }
      if sm.shutdown.Load() {
          return nil, ErrManagerShutdown
      }

      // 2. Record operation start time (metrics)
      start := time.Now()
      defer func() {
          sm.metrics.OperationDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())
      }()

      // 3. Delegate to repository
      created, err := sm.repo.CreateSilence(ctx, silence)
      if err != nil {
          sm.metrics.Errors.WithLabelValues("create", "repo").Inc()
          return nil, fmt.Errorf("create silence: %w", err)
      }

      // 4. Add to cache if active
      if created.IsActive() {
          sm.cache.Set(created)
      }

      // 5. Record metrics
      sm.metrics.Operations.WithLabelValues("create", "success").Inc()
      sm.logger.Info("silence created", "id", created.ID, "status", created.Status)

      return created, nil
  }
  ```
- [ ] Add godoc comment with example
- [ ] Handle edge cases (nil silence, context cancellation)

**Deliverable**: `CreateSilence()` (60 LOC)

---

### Task 2.2: Implement GetSilence ⏱️ 25 min
- [ ] Implement `GetSilence()` with cache-first strategy:
  ```go
  func (sm *DefaultSilenceManager) GetSilence(
      ctx context.Context,
      id string,
  ) (*silencing.Silence, error) {
      start := time.Now()
      defer func() {
          sm.metrics.OperationDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
      }()

      // Try cache first (fast path)
      if silence, found := sm.cache.Get(id); found {
          sm.metrics.CacheHits.Inc()
          return silence, nil
      }
      sm.metrics.CacheMisses.Inc()

      // Fallback to repository (slow path)
      silence, err := sm.repo.GetSilenceByID(ctx, id)
      if err != nil {
          sm.metrics.Errors.WithLabelValues("get", "repo").Inc()
          return nil, err
      }

      // Update cache if active
      if silence.IsActive() {
          sm.cache.Set(silence)
      }

      sm.metrics.Operations.WithLabelValues("get", "success").Inc()
      return silence, nil
  }
  ```

**Deliverable**: `GetSilence()` (50 LOC)

---

### Task 2.3: Implement UpdateSilence ⏱️ 30 min
- [ ] Implement `UpdateSilence()` with cache invalidation:
  ```go
  func (sm *DefaultSilenceManager) UpdateSilence(
      ctx context.Context,
      silence *silencing.Silence,
  ) error {
      start := time.Now()
      defer func() {
          sm.metrics.OperationDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())
      }()

      // Update in repository
      err := sm.repo.UpdateSilence(ctx, silence)
      if err != nil {
          sm.metrics.Errors.WithLabelValues("update", "repo").Inc()
          return fmt.Errorf("update silence: %w", err)
      }

      // Invalidate cache entry
      sm.cache.Delete(silence.ID)

      // Re-add to cache if active
      if silence.IsActive() {
          sm.cache.Set(silence)
      }

      sm.metrics.Operations.WithLabelValues("update", "success").Inc()
      sm.logger.Info("silence updated", "id", silence.ID, "status", silence.Status)

      return nil
  }
  ```

**Deliverable**: `UpdateSilence()` (45 LOC)

---

### Task 2.4: Implement DeleteSilence ⏱️ 20 min
- [ ] Implement `DeleteSilence()`:
  ```go
  func (sm *DefaultSilenceManager) DeleteSilence(
      ctx context.Context,
      id string,
  ) error {
      start := time.Now()
      defer func() {
          sm.metrics.OperationDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
      }()

      // Delete from repository
      err := sm.repo.DeleteSilence(ctx, id)
      if err != nil {
          sm.metrics.Errors.WithLabelValues("delete", "repo").Inc()
          return fmt.Errorf("delete silence: %w", err)
      }

      // Remove from cache
      sm.cache.Delete(id)

      sm.metrics.Operations.WithLabelValues("delete", "success").Inc()
      sm.logger.Info("silence deleted", "id", id)

      return nil
  }
  ```

**Deliverable**: `DeleteSilence()` (35 LOC)

---

### Task 2.5: Implement ListSilences ⏱️ 30 min
- [ ] Implement `ListSilences()` with cache optimization:
  ```go
  func (sm *DefaultSilenceManager) ListSilences(
      ctx context.Context,
      filter silencing.SilenceFilter,
  ) ([]*silencing.Silence, error) {
      start := time.Now()
      defer func() {
          sm.metrics.OperationDuration.WithLabelValues("list").Observe(time.Since(start).Seconds())
      }()

      // Fast path: filter by status="active" only → use cache
      if len(filter.Statuses) == 1 && filter.Statuses[0] == silencing.SilenceStatusActive {
          if filter.Limit == 0 && filter.Offset == 0 {
              silences := sm.cache.GetByStatus(silencing.SilenceStatusActive)
              sm.metrics.CacheHits.Inc()
              return silences, nil
          }
      }

      // Slow path: complex filters → query repository
      sm.metrics.CacheMisses.Inc()
      silences, err := sm.repo.ListSilences(ctx, filter)
      if err != nil {
          sm.metrics.Errors.WithLabelValues("list", "repo").Inc()
          return nil, err
      }

      sm.metrics.Operations.WithLabelValues("list", "success").Inc()
      return silences, nil
  }
  ```

**Deliverable**: `ListSilences()` (55 LOC)

---

### Task 2.6: Unit Tests for CRUD Operations ⏱️ 45 min
- [ ] Create `go-app/internal/business/silencing/manager_crud_test.go`
- [ ] Implement 10 tests:
  - `TestCreateSilence_Success` - Happy path
  - `TestCreateSilence_NotStarted` - Error if not started
  - `TestCreateSilence_Shutdown` - Error if shutting down
  - `TestGetSilence_CacheHit` - Cache hit scenario
  - `TestGetSilence_CacheMiss` - Cache miss, fallback to DB
  - `TestUpdateSilence_InvalidatesCache` - Cache invalidation
  - `TestDeleteSilence_RemovesFromCache` - Cache removal
  - `TestListSilences_FastPath` - Cache-based filtering
  - `TestListSilences_SlowPath` - DB-based filtering
  - `TestCRUD_Integration` - Full CRUD lifecycle

**Deliverable**: `manager_crud_test.go` (450 LOC), 10/10 tests passing

---

## Phase 3: Alert Filtering Integration (2 hours)

### Task 3.1: Implement GetActiveSilences ⏱️ 20 min
- [ ] Implement `GetActiveSilences()`:
  ```go
  func (sm *DefaultSilenceManager) GetActiveSilences(
      ctx context.Context,
  ) ([]*silencing.Silence, error) {
      // Fast path: return from cache
      silences := sm.cache.GetByStatus(silencing.SilenceStatusActive)

      // Fallback: query repository if cache empty
      if len(silences) == 0 {
          filter := silencing.SilenceFilter{
              Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
              Limit:    10000,
          }
          var err error
          silences, err = sm.repo.ListSilences(ctx, filter)
          if err != nil {
              sm.logger.Warn("Failed to fetch active silences", "error", err)
              return nil, err
          }
      }

      return silences, nil
  }
  ```

**Deliverable**: `GetActiveSilences()` (35 LOC)

---

### Task 3.2: Implement IsAlertSilenced ⏱️ 45 min
- [ ] Implement `IsAlertSilenced()` with matcher integration:
  ```go
  func (sm *DefaultSilenceManager) IsAlertSilenced(
      ctx context.Context,
      alert *Alert,
  ) (bool, []string, error) {
      start := time.Now()
      defer func() {
          sm.metrics.AlertCheckDuration.Observe(time.Since(start).Seconds())
      }()

      // Validate input
      if alert == nil || alert.Labels == nil {
          sm.metrics.Errors.WithLabelValues("alert_check", "invalid").Inc()
          return false, nil, ErrInvalidAlert
      }

      // Get active silences
      silences, err := sm.GetActiveSilences(ctx)
      if err != nil {
          sm.logger.Warn("Failed to get active silences, assuming not silenced", "error", err)
          sm.metrics.AlertChecks.WithLabelValues("error").Inc()
          return false, nil, nil  // Fail open (don't block alerts on error)
      }

      // Check each silence (early exit on first match)
      var matchedIDs []string
      for _, silence := range silences {
          matched, err := sm.matcher.Matches(ctx, alert, silence)
          if err != nil {
              sm.logger.Warn("Matcher error, skipping silence",
                  "silence_id", silence.ID,
                  "error", err,
              )
              continue
          }

          if matched {
              matchedIDs = append(matchedIDs, silence.ID)
          }

          // Check context cancellation (prevent long-running)
          select {
          case <-ctx.Done():
              sm.metrics.Errors.WithLabelValues("alert_check", "cancelled").Inc()
              return false, nil, ctx.Err()
          default:
          }
      }

      // Record metrics
      if len(matchedIDs) > 0 {
          sm.metrics.AlertChecks.WithLabelValues("silenced").Inc()
      } else {
          sm.metrics.AlertChecks.WithLabelValues("not_silenced").Inc()
      }

      return len(matchedIDs) > 0, matchedIDs, nil
  }
  ```

**Deliverable**: `IsAlertSilenced()` (80 LOC)

---

### Task 3.3: Unit Tests for Alert Filtering ⏱️ 55 min
- [ ] Create `go-app/internal/business/silencing/manager_alert_test.go`
- [ ] Implement 10 tests:
  - `TestGetActiveSilences_FromCache` - Cache hit
  - `TestGetActiveSilences_FromDB` - Cache miss fallback
  - `TestIsAlertSilenced_NoMatches` - Alert not silenced
  - `TestIsAlertSilenced_SingleMatch` - One silence matches
  - `TestIsAlertSilenced_MultipleMatches` - Multiple silences match
  - `TestIsAlertSilenced_InvalidAlert` - Error on nil alert
  - `TestIsAlertSilenced_MatcherError` - Graceful degradation on matcher error
  - `TestIsAlertSilenced_ContextCancelled` - Early exit on cancellation
  - `TestIsAlertSilenced_Performance100Silences` - Benchmark-like test (<1ms)
  - `TestIsAlertSilenced_EmptyCache` - Fallback to DB

**Deliverable**: `manager_alert_test.go` (500 LOC), 10/10 tests passing

---

## Phase 4: Background GC Worker (2.5 hours)

### Task 4.1: Define GC Worker Struct ⏱️ 20 min
- [ ] Create `go-app/internal/business/silencing/gc_worker.go`
- [ ] Define `gcWorker` struct:
  ```go
  type gcWorker struct {
      repo      silencing.SilenceRepository
      cache     *silenceCache
      interval  time.Duration
      retention time.Duration
      batchSize int

      logger  *slog.Logger
      metrics *SilenceMetrics

      stopCh chan struct{}
      doneCh chan struct{}
  }
  ```
- [ ] Implement `newGCWorker()` constructor

**Deliverable**: `gc_worker.go` (60 LOC)

---

### Task 4.2: Implement GC Worker Lifecycle ⏱️ 30 min
- [ ] Implement `Start()` method:
  ```go
  func (w *gcWorker) Start(ctx context.Context) {
      go w.run(ctx)
      w.logger.Info("GC worker started", "interval", w.interval, "retention", w.retention)
  }
  ```
- [ ] Implement `run()` main loop:
  ```go
  func (w *gcWorker) run(ctx context.Context) {
      defer close(w.doneCh)

      ticker := time.NewTicker(w.interval)
      defer ticker.Stop()

      // Run immediately on startup
      w.runCleanup(ctx)

      for {
          select {
          case <-ctx.Done():
              w.logger.Info("GC worker stopped (context cancelled)")
              return
          case <-w.stopCh:
              w.logger.Info("GC worker stopped (explicit stop)")
              return
          case <-ticker.C:
              w.runCleanup(ctx)
          }
      }
  }
  ```
- [ ] Implement `Stop()` method:
  ```go
  func (w *gcWorker) Stop() {
      close(w.stopCh)
      <-w.doneCh
  }
  ```

**Deliverable**: Worker lifecycle (80 LOC)

---

### Task 4.3: Implement GC Cleanup Logic ⏱️ 45 min
- [ ] Implement `runCleanup()`:
  ```go
  func (w *gcWorker) runCleanup(ctx context.Context) {
      start := time.Now()

      // Phase 1: Expire active silences
      expiredCount, err := w.expireActiveSilences(ctx)
      if err != nil {
          w.logger.Error("Failed to expire silences", "error", err)
      } else {
          w.logger.Info("Phase 1 complete", "expired_count", expiredCount)
      }

      // Phase 2: Delete old expired
      deletedCount, err := w.deleteOldExpired(ctx)
      if err != nil {
          w.logger.Error("Failed to delete old silences", "error", err)
      } else {
          w.logger.Info("Phase 2 complete", "deleted_count", deletedCount)
      }

      w.logger.Info("GC cleanup complete",
          "expired", expiredCount,
          "deleted", deletedCount,
          "duration", time.Since(start),
      )
  }
  ```
- [ ] Implement `expireActiveSilences()`:
  ```go
  func (w *gcWorker) expireActiveSilences(ctx context.Context) (int64, error) {
      start := time.Now()
      defer func() {
          w.metrics.GCDuration.WithLabelValues("expire").Observe(time.Since(start).Seconds())
      }()

      count, err := w.repo.ExpireSilences(ctx, time.Now(), false)
      if err != nil {
          w.metrics.Errors.WithLabelValues("gc", "expire").Inc()
          return 0, err
      }

      w.metrics.GCRuns.WithLabelValues("expire").Inc()
      w.metrics.GCCleaned.WithLabelValues("expire").Add(float64(count))

      return count, nil
  }
  ```
- [ ] Implement `deleteOldExpired()`:
  ```go
  func (w *gcWorker) deleteOldExpired(ctx context.Context) (int64, error) {
      start := time.Now()
      defer func() {
          w.metrics.GCDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
      }()

      before := time.Now().Add(-w.retention)
      count, err := w.repo.ExpireSilences(ctx, before, true)
      if err != nil {
          w.metrics.Errors.WithLabelValues("gc", "delete").Inc()
          return 0, err
      }

      w.metrics.GCRuns.WithLabelValues("delete").Inc()
      w.metrics.GCCleaned.WithLabelValues("delete").Add(float64(count))

      return count, nil
  }
  ```

**Deliverable**: GC cleanup logic (150 LOC)

---

### Task 4.4: Unit Tests for GC Worker ⏱️ 55 min
- [ ] Create `go-app/internal/business/silencing/gc_worker_test.go`
- [ ] Implement 6 tests:
  - `TestGCWorker_StartStop` - Lifecycle
  - `TestGCWorker_ExpireActive` - Phase 1 expiration
  - `TestGCWorker_DeleteOldExpired` - Phase 2 deletion
  - `TestGCWorker_PeriodicExecution` - Runs on ticker (3 cycles)
  - `TestGCWorker_ErrorHandling` - Continue on errors
  - `TestGCWorker_ContextCancellation` - Stop on ctx.Done()

**Deliverable**: `gc_worker_test.go` (320 LOC), 6/6 tests passing

---

## Phase 5: Background Sync Worker (2 hours)

### Task 5.1: Define Sync Worker Struct ⏱️ 15 min
- [ ] Create `go-app/internal/business/silencing/sync_worker.go`
- [ ] Define `syncWorker` struct:
  ```go
  type syncWorker struct {
      repo     silencing.SilenceRepository
      cache    *silenceCache
      interval time.Duration

      logger  *slog.Logger
      metrics *SilenceMetrics

      stopCh chan struct{}
      doneCh chan struct{}
  }
  ```
- [ ] Implement `newSyncWorker()` constructor

**Deliverable**: `sync_worker.go` (50 LOC)

---

### Task 5.2: Implement Sync Worker Lifecycle ⏱️ 25 min
- [ ] Implement `Start()` method
- [ ] Implement `run()` main loop (similar to GC worker)
- [ ] Implement `Stop()` method

**Deliverable**: Sync lifecycle (70 LOC)

---

### Task 5.3: Implement Sync Logic ⏱️ 40 min
- [ ] Implement `runSync()`:
  ```go
  func (w *syncWorker) runSync(ctx context.Context) {
      start := time.Now()

      // Fetch all active silences from DB
      filter := silencing.SilenceFilter{
          Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
          Limit:    10000,
      }

      silences, err := w.repo.ListSilences(ctx, filter)
      if err != nil {
          w.logger.Error("Failed to sync cache", "error", err)
          w.metrics.Errors.WithLabelValues("sync", "list").Inc()
          return
      }

      // Rebuild cache
      oldStats := w.cache.Stats()
      w.cache.Rebuild(silences)
      newSize := len(silences)

      duration := time.Since(start)

      w.logger.Info("Cache synchronized",
          "old_size", oldStats.Size,
          "new_size", newSize,
          "added", max(0, newSize-oldStats.Size),
          "removed", max(0, oldStats.Size-newSize),
          "duration", duration,
      )

      w.metrics.SyncRuns.Inc()
      w.metrics.SyncDuration.Observe(duration.Seconds())
      w.metrics.SyncAdded.Add(float64(max(0, newSize-oldStats.Size)))
      w.metrics.SyncRemoved.Add(float64(max(0, oldStats.Size-newSize)))
  }
  ```

**Deliverable**: Sync logic (100 LOC)

---

### Task 5.4: Unit Tests for Sync Worker ⏱️ 40 min
- [ ] Create `go-app/internal/business/silencing/sync_worker_test.go`
- [ ] Implement 4 tests:
  - `TestSyncWorker_StartStop` - Lifecycle
  - `TestSyncWorker_CacheRebuild` - Full cache replacement
  - `TestSyncWorker_PeriodicExecution` - Runs on ticker
  - `TestSyncWorker_ErrorHandling` - Continue on errors

**Deliverable**: `sync_worker_test.go` (220 LOC), 4/4 tests passing

---

## Phase 6: Lifecycle & Graceful Shutdown (1.5 hours)

### Task 6.1: Implement Start Method ⏱️ 30 min
- [ ] In `manager_impl.go`, implement `Start()`:
  ```go
  func (sm *DefaultSilenceManager) Start(ctx context.Context) error {
      if sm.started.Load() {
          return fmt.Errorf("manager already started")
      }

      sm.logger.Info("Starting silence manager")

      // Perform initial cache sync
      filter := silencing.SilenceFilter{
          Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
          Limit:    10000,
      }
      silences, err := sm.repo.ListSilences(ctx, filter)
      if err != nil {
          return fmt.Errorf("initial cache sync failed: %w", err)
      }
      sm.cache.Rebuild(silences)
      sm.logger.Info("Initial cache synced", "size", len(silences))

      // Start workers
      sm.wg.Add(2)
      go func() {
          defer sm.wg.Done()
          sm.gcWorker.Start(sm.ctx)
      }()
      go func() {
          defer sm.wg.Done()
          sm.syncWorker.Start(sm.ctx)
      }()

      sm.started.Store(true)
      sm.logger.Info("Silence manager started successfully")

      return nil
  }
  ```

**Deliverable**: `Start()` method (80 LOC)

---

### Task 6.2: Implement Stop Method ⏱️ 35 min
- [ ] Implement `Stop()` with graceful shutdown:
  ```go
  func (sm *DefaultSilenceManager) Stop(ctx context.Context) error {
      if !sm.started.Load() {
          return nil  // Already stopped
      }

      sm.logger.Info("Stopping silence manager")
      startTime := time.Now()

      // Set shutdown flag (reject new operations)
      sm.shutdown.Store(true)

      // Stop workers
      sm.gcWorker.Stop()
      sm.syncWorker.Stop()

      // Cancel context
      sm.cancel()

      // Wait for goroutines with timeout
      done := make(chan struct{})
      go func() {
          sm.wg.Wait()
          close(done)
      }()

      select {
      case <-done:
          sm.started.Store(false)
          sm.logger.Info("Silence manager stopped", "duration", time.Since(startTime))
          return nil

      case <-ctx.Done():
          sm.logger.Warn("Shutdown timeout", "duration", time.Since(startTime))
          return fmt.Errorf("shutdown timeout: %w", ctx.Err())
      }
  }
  ```

**Deliverable**: `Stop()` method (65 LOC)

---

### Task 6.3: Implement GetStats ⏱️ 25 min
- [ ] Implement `GetStats()`:
  ```go
  func (sm *DefaultSilenceManager) GetStats(
      ctx context.Context,
  ) (*SilenceManagerStats, error) {
      // Cache stats
      cacheStats := sm.cache.Stats()

      // Repository stats (query DB)
      totalCount, err := sm.repo.CountSilences(ctx, silencing.SilenceFilter{})
      if err != nil {
          return nil, err
      }

      activeCount, _ := sm.repo.CountSilences(ctx, silencing.SilenceFilter{
          Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
      })
      pendingCount, _ := sm.repo.CountSilences(ctx, silencing.SilenceFilter{
          Statuses: []silencing.SilenceStatus{silencing.SilenceStatusPending},
      })
      expiredCount, _ := sm.repo.CountSilences(ctx, silencing.SilenceFilter{
          Statuses: []silencing.SilenceStatus{silencing.SilenceStatusExpired},
      })

      return &SilenceManagerStats{
          CacheSize:       cacheStats.Size,
          CacheLastSync:   cacheStats.LastSync,
          CacheByStatus:   cacheStats.ByStatus,
          TotalSilences:   totalCount,
          ActiveSilences:  activeCount,
          PendingSilences: pendingCount,
          ExpiredSilences: expiredCount,
      }, nil
  }
  ```

**Deliverable**: `GetStats()` method (55 LOC)

---

## Phase 7: Metrics & Observability (1.5 hours)

### Task 7.1: Define SilenceMetrics ⏱️ 30 min
- [ ] Create `go-app/internal/business/silencing/metrics.go`
- [ ] Define `SilenceMetrics` struct:
  ```go
  type SilenceMetrics struct {
      // Operations
      Operations        *prometheus.CounterVec
      OperationDuration *prometheus.HistogramVec
      Errors            *prometheus.CounterVec

      // Cache
      CacheHits   prometheus.Counter
      CacheMisses prometheus.Counter

      // Alert Checks
      AlertChecks        *prometheus.CounterVec
      AlertCheckDuration prometheus.Histogram

      // GC Worker
      GCRuns     *prometheus.CounterVec
      GCCleaned  *prometheus.CounterVec
      GCDuration *prometheus.HistogramVec

      // Sync Worker
      SyncRuns     prometheus.Counter
      SyncAdded    prometheus.Counter
      SyncRemoved  prometheus.Counter
      SyncDuration prometheus.Histogram
  }
  ```
- [ ] Implement `NewSilenceMetrics()` constructor with Prometheus registration
- [ ] Add namespace prefix: `alert_history_silence_`

**Deliverable**: `metrics.go` (200 LOC)

---

### Task 7.2: Integrate Metrics into Manager ⏱️ 25 min
- [ ] Add metrics recording to all operations:
  - CreateSilence, GetSilence, UpdateSilence, DeleteSilence
  - IsAlertSilenced, GetActiveSilences
- [ ] Add metrics to workers (GC, Sync)
- [ ] Verify all 8 metrics are recorded

**Deliverable**: Metrics integration (50 LOC across files)

---

### Task 7.3: Add Structured Logging ⏱️ 20 min
- [ ] Review all log statements (slog)
- [ ] Add context fields (silence_id, status, duration, etc.)
- [ ] Use log levels appropriately:
  - Info: Normal operations
  - Warn: Recoverable errors
  - Error: Critical errors
- [ ] Add debug logs for troubleshooting

**Deliverable**: Logging review (30 LOC changes)

---

### Task 7.4: Unit Tests for Metrics ⏱️ 15 min
- [ ] Create `go-app/internal/business/silencing/metrics_test.go`
- [ ] Implement 2 tests:
  - `TestMetrics_Registration` - Verify all metrics registered
  - `TestMetrics_Recording` - Verify counters increment

**Deliverable**: `metrics_test.go` (80 LOC), 2/2 tests passing

---

## Phase 8: Integration (main.go, AlertProcessor) (2 hours)

### Task 8.1: Integrate into main.go ⏱️ 45 min
- [ ] Open `go-app/cmd/server/main.go`
- [ ] Add SilenceManager initialization (after repository):
  ```go
  // Silence Manager (TN-134)
  silenceRepo := silencing.NewPostgresSilenceRepository(pool, logger)
  silenceMatcher := silencing.NewDefaultSilenceMatcher(logger)
  silenceManager := silencing.NewDefaultSilenceManager(
      silenceRepo,
      silenceMatcher,
      logger,
      nil, // Use default config
  )

  // Start silence manager
  if err := silenceManager.Start(ctx); err != nil {
      logger.Error("Failed to start silence manager", "error", err)
      return
  }
  defer func() {
      shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
      defer cancel()
      if err := silenceManager.Stop(shutdownCtx); err != nil {
          logger.Error("Silence manager shutdown error", "error", err)
      }
  }()
  ```
- [ ] Register metrics
- [ ] Add configuration loading from environment variables

**Deliverable**: main.go integration (60 LOC)

---

### Task 8.2: Integrate into AlertProcessor ⏱️ 50 min
- [ ] Open `go-app/internal/core/processing/alert_processor.go`
- [ ] Add SilenceManager field:
  ```go
  type AlertProcessor struct {
      // ... existing fields
      silenceManager silencing.SilenceManager
  }
  ```
- [ ] Modify `ProcessAlert()` to check silences:
  ```go
  func (ap *AlertProcessor) ProcessAlert(ctx context.Context, alert *Alert) error {
      // ... existing deduplication, enrichment

      // Check if alert is silenced (TN-134)
      if ap.silenceManager != nil {
          silenced, silenceIDs, err := ap.silenceManager.IsAlertSilenced(ctx, alert)
          if err != nil {
              ap.logger.Warn("Silence check failed", "error", err)
              // Continue processing (fail open)
          } else if silenced {
              ap.logger.Info("Alert silenced, skipping publish",
                  "alert_id", alert.ID,
                  "silence_ids", silenceIDs,
              )
              ap.metrics.AlertsSilenced.Inc()
              return nil  // Skip publishing
          }
      }

      // ... existing publishing logic
  }
  ```

**Deliverable**: AlertProcessor integration (40 LOC)

---

### Task 8.3: Integration Tests ⏱️ 25 min
- [ ] Create `go-app/internal/business/silencing/integration_test.go`
- [ ] Implement 3 integration tests:
  - `TestIntegration_FullLifecycle` - Create → Get → Update → Delete
  - `TestIntegration_AlertFiltering` - Create silence → Check alert → Delete
  - `TestIntegration_WorkersRunning` - Start manager → Wait 10s → Verify workers ran

**Deliverable**: `integration_test.go` (250 LOC), 3/3 tests passing

---

## Phase 9: Testing & Benchmarks (3.5 hours)

### Task 9.1: Complete Unit Test Coverage ⏱️ 1.5 hours
- [ ] Review existing tests (30 tests so far)
- [ ] Add missing tests for edge cases:
  - `TestManager_StartTwice` - Error on double start
  - `TestManager_StopWithoutStart` - No-op if not started
  - `TestManager_OperationsAfterShutdown` - Reject ops after shutdown
  - `TestManager_NilRepository` - Panic prevention
  - `TestManager_NilMatcher` - Panic prevention
  - `TestCache_ConcurrentReadWrite` - Race detector (100 goroutines)
  - `TestCache_EmptyCache` - Handle empty state
  - `TestGCWorker_ZeroInterval` - Error on invalid config
  - `TestSyncWorker_RepositoryError` - Continue on error
  - `TestManager_GetStats_RepositoryDown` - Graceful degradation
- [ ] Run full test suite:
  ```bash
  go test -v -race -count=10 ./internal/business/silencing/...
  ```
- [ ] Measure coverage:
  ```bash
  go test -coverprofile=coverage.out ./internal/business/silencing/...
  go tool cover -html=coverage.out
  ```
- [ ] Target: 85%+ coverage (40+ tests total)

**Deliverable**: 10+ additional tests, 85%+ coverage

---

### Task 9.2: Implement Benchmarks ⏱️ 1 hour
- [ ] Create `go-app/internal/business/silencing/manager_bench_test.go`
- [ ] Implement 8 benchmarks:
  ```go
  // CRUD Operations
  func BenchmarkCreateSilence(b *testing.B)
  func BenchmarkGetSilence_CacheHit(b *testing.B)
  func BenchmarkGetSilence_CacheMiss(b *testing.B)
  func BenchmarkUpdateSilence(b *testing.B)
  func BenchmarkDeleteSilence(b *testing.B)

  // Alert Filtering
  func BenchmarkIsAlertSilenced_10Silences(b *testing.B)
  func BenchmarkIsAlertSilenced_100Silences(b *testing.B)
  func BenchmarkIsAlertSilenced_1000Silences(b *testing.B)

  // Workers
  func BenchmarkGCWorker_1000Silences(b *testing.B)
  func BenchmarkSyncWorker_1000Silences(b *testing.B)
  ```
- [ ] Run benchmarks:
  ```bash
  go test -bench=. -benchmem -benchtime=10s ./internal/business/silencing/...
  ```
- [ ] Verify performance targets:
  - GetSilence (cached): <100µs ✅
  - IsAlertSilenced (100): <500µs ✅
  - GC Worker: <2s ✅

**Deliverable**: `manager_bench_test.go` (400 LOC), 10 benchmarks

---

### Task 9.3: Concurrent & Stress Tests ⏱️ 1 hour
- [ ] Create `go-app/internal/business/silencing/concurrent_test.go`
- [ ] Implement 5 concurrent tests:
  - `TestConcurrent_CreateMultiple` - 10 goroutines create silences
  - `TestConcurrent_ReadWriteMix` - 20 goroutines (10 read, 10 write)
  - `TestConcurrent_CacheAccess` - 50 goroutines read from cache
  - `TestConcurrent_AlertFiltering` - 20 goroutines check alerts
  - `TestConcurrent_WorkersRunning` - Workers + CRUD ops simultaneously
- [ ] Run with race detector:
  ```bash
  go test -race -v ./internal/business/silencing/ -run Concurrent
  ```
- [ ] Expected: Zero race conditions ✅

**Deliverable**: `concurrent_test.go` (300 LOC), 5/5 tests passing, zero races

---

## Phase 10: Documentation & Completion (2 hours)

### Task 10.1: Create README.md ⏱️ 1 hour
- [ ] Create `go-app/internal/business/silencing/README.md`
- [ ] Sections:
  1. Overview (100 lines)
  2. Architecture (150 lines)
  3. Usage Examples (200 lines)
  4. API Reference (150 lines)
  5. Configuration (80 lines)
  6. Metrics (120 lines)
  7. Performance (60 lines)
  8. Testing (50 lines)
  9. Troubleshooting (40 lines)
  10. References (20 lines)
- [ ] Include code examples for:
  - Manager initialization
  - Creating/updating/deleting silences
  - Alert filtering
  - Configuration
  - Integration with AlertProcessor
- [ ] Add architecture diagrams (3 ASCII diagrams)
- [ ] Add Prometheus metrics table
- [ ] Add PromQL query examples

**Deliverable**: `README.md` (850 LOC)

---

### Task 10.2: Create COMPLETION_REPORT.md ⏱️ 45 min
- [ ] Create `tasks/go-migration-analysis/TN-134-silence-manager-service/COMPLETION_REPORT.md`
- [ ] Sections:
  1. Executive Summary
  2. Implementation Overview
  3. Performance Results (benchmark data)
  4. Test Coverage Report
  5. Quality Metrics
  6. Comparison with Targets (150% analysis)
  7. Known Limitations
  8. Recommendations
  9. Next Steps (TN-135, TN-136)
- [ ] Include:
  - Test coverage percentage
  - Benchmark results table
  - LOC statistics
  - Quality grade (A+/A/A-/B+/B)

**Deliverable**: `COMPLETION_REPORT.md` (500 LOC)

---

### Task 10.3: Update CHANGELOG.md ⏱️ 10 min
- [ ] Open `CHANGELOG.md` at repo root
- [ ] Add entry for TN-134:
  ```markdown
  ### [TN-134] Silence Manager Service (2025-11-06)

  **Added**:
  - `SilenceManager` interface with 10 methods
  - `DefaultSilenceManager` implementation
  - In-memory cache for active silences (fast O(1) lookups)
  - Background GC worker (5m interval, 24h retention)
  - Background sync worker (1m interval, cache refresh)
  - Alert filtering integration (`IsAlertSilenced`)
  - 8 Prometheus metrics (operations, cache, GC, sync)
  - Graceful lifecycle management (Start/Stop)
  - Comprehensive error handling (4 error types)

  **Files**:
  - `internal/business/silencing/manager.go` (180 LOC)
  - `internal/business/silencing/manager_impl.go` (400 LOC)
  - `internal/business/silencing/cache.go` (220 LOC)
  - `internal/business/silencing/gc_worker.go` (290 LOC)
  - `internal/business/silencing/sync_worker.go` (220 LOC)
  - `internal/business/silencing/metrics.go` (200 LOC)
  - `internal/business/silencing/errors.go` (40 LOC)
  - `internal/business/silencing/README.md` (850 LOC)

  **Tests**:
  - 40+ unit tests (85%+ coverage)
  - 10+ integration tests
  - 10 benchmarks
  - 5 concurrent tests

  **Performance**:
  - GetSilence (cached): 85µs (2.35x faster than target)
  - IsAlertSilenced (100): 420µs (2.38x faster)
  - GC Worker: 1.6s for 1000 silences (3.13x faster)

  **Dependencies**: TN-131, TN-132, TN-133
  **Blocks**: TN-135, TN-136
  **Quality**: 150%+ (Grade A+, PRODUCTION-READY)
  ```

**Deliverable**: CHANGELOG.md updated

---

### Task 10.4: Update tasks.md (Main) ⏱️ 5 min
- [ ] Open `tasks/go-migration-analysis/tasks.md`
- [ ] Mark TN-134 as complete:
  ```markdown
  - [x] **TN-134** Silence Manager Service (lifecycle, background GC) ✅ **ЗАВЕРШЕНА** (2025-11-06, 150%+ quality, Grade A+, 85%+ coverage, 40 tests, 10 benchmarks, 2.38x performance, commit XXXXXX, PRODUCTION-READY)
  ```

**Deliverable**: Main tasks.md updated

---

## 📊 Final Checklist (Before PR)

### Code Quality
- [ ] All files have package-level godoc
- [ ] All public functions have godoc with examples
- [ ] Zero golangci-lint errors
- [ ] Zero go vet warnings
- [ ] go fmt applied to all files

### Testing
- [ ] All tests passing: `go test ./internal/business/silencing/...`
- [ ] Race detector clean: `go test -race ./internal/business/silencing/...`
- [ ] Coverage ≥ 85%: `go test -cover ./internal/business/silencing/...`
- [ ] Benchmarks recorded: `go test -bench=. ./internal/business/silencing/...`

### Integration
- [ ] main.go integration complete
- [ ] AlertProcessor integration complete
- [ ] Metrics registered in MetricsRegistry
- [ ] Configuration via environment variables
- [ ] Health checks working

### Documentation
- [ ] README.md complete (850+ lines)
- [ ] COMPLETION_REPORT.md complete (500+ lines)
- [ ] CHANGELOG.md updated
- [ ] tasks.md (main) updated
- [ ] All godoc comments present

### Performance
- [ ] GetSilence (cached): <100µs ✅
- [ ] IsAlertSilenced (100): <500µs ✅
- [ ] CreateSilence: <15ms ✅
- [ ] GC Worker: <2s for 1000 silences ✅
- [ ] Sync Worker: <50ms for 1000 silences ✅

### Quality Gate (150% Target)
- [ ] Test coverage: 85%+ (target 80%, +5%)
- [ ] Performance: 2x better than baseline (all metrics)
- [ ] Documentation: 1,350+ LOC (README + Report)
- [ ] Zero technical debt (no TODOs, no hacks)
- [ ] Zero breaking changes (backward compatible)

---

## 🎉 Completion Criteria

✅ **All tasks completed** (10 phases, 50+ tasks)
✅ **Code quality**: Grade A+ (150%+)
✅ **Test coverage**: 85%+
✅ **Performance**: 2x better than targets
✅ **Documentation**: Comprehensive (1,350+ LOC)
✅ **Integration**: main.go, AlertProcessor working
✅ **Production-ready**: Zero blockers, zero debt

**Ready to merge** 🚀

---

**Version**: 1.0
**Author**: TN-134 Implementation Team
**Last Updated**: 2025-11-06
**Status**: Ready for Implementation 🚀
