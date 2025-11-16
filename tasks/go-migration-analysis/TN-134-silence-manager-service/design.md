# TN-134: Silence Manager Service - Design Document

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-134
**Version**: 1.0
**Last Updated**: 2025-11-06
**Status**: 🟡 DESIGN PHASE

---

## 🎯 Design Overview

Silence Manager Service - это **централизованный координатор** для управления жизненным циклом silence rules, объединяющий:
- **Storage Layer** (TN-133): PostgreSQL repository для персистентности
- **Matcher Layer** (TN-132): Regex-based matching engine
- **Business Logic**: Lifecycle management, cache, background workers

### Key Design Principles

| Principle | Implementation | Trade-offs |
|-----------|----------------|------------|
| **Separation of Concerns** | Manager координирует, но не знает детали storage/matching | Чистая архитектура, но больше кода |
| **In-Memory Cache** | Активные silences в памяти для fast lookups | Требует синхронизации с DB |
| **Background Workers** | Автоматическая GC/sync через ticker goroutines | Небольшой overhead (~0.1% CPU) |
| **Graceful Degradation** | Продолжает работу при ошибках cache/workers | Может быть temporary inconsistency |
| **Thread Safety** | RWMutex для cache, WaitGroup для workers | Slightly slower writes |
| **Fail-Safe Design** | Errors logged, operations continue | May miss some cleanups |

---

## 📐 Architecture

### Component Diagram

```
┌───────────────────────────────────────────────────────────────────────────┐
│                          Application Layer                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐           │
│  │ AlertProcessor  │  │  API Endpoints  │  │  Health Checks  │           │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘           │
│           │                    │                     │                     │
│           └──────────────────────┬──────────────────────┘                 │
│                                  │ Uses                                    │
└──────────────────────────────────┼─────────────────────────────────────────┘
                                   │
                                   ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                      SilenceManager Interface                              │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │  Core Operations:                                                  │  │
│  │  - CreateSilence(ctx, *Silence) (*Silence, error)                 │  │
│  │  - GetSilence(ctx, id) (*Silence, error)                          │  │
│  │  - UpdateSilence(ctx, *Silence) error                             │  │
│  │  - DeleteSilence(ctx, id) error                                   │  │
│  │  - ListSilences(ctx, filter) ([]*Silence, error)                  │  │
│  │                                                                     │  │
│  │  Alert Integration:                                                │  │
│  │  - IsAlertSilenced(ctx, alert) (bool, []string, error)            │  │
│  │  - GetActiveSilences(ctx) ([]*Silence, error)                     │  │
│  │                                                                     │  │
│  │  Lifecycle:                                                        │  │
│  │  - Start(ctx) error                                                │  │
│  │  - Stop(ctx) error                                                 │  │
│  │                                                                     │  │
│  │  Status:                                                           │  │
│  │  - GetStats(ctx) (*SilenceManagerStats, error)                    │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ Implements
                                   ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                    DefaultSilenceManager (TN-134)                          │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  Components:                                                        │ │
│  │                                                                      │ │
│  │  1. Storage Integration:                                           │ │
│  │     - repo: SilenceRepository (TN-133)                             │ │
│  │     - CRUD operations delegation                                   │ │
│  │                                                                      │ │
│  │  2. Matcher Integration:                                           │ │
│  │     - matcher: SilenceMatcher (TN-132)                             │ │
│  │     - Alert filtering logic                                        │ │
│  │                                                                      │ │
│  │  3. In-Memory Cache:                                               │ │
│  │     - cache: *silenceCache                                         │ │
│  │     - Active silences map (ID → Silence)                           │ │
│  │     - Status index (Status → IDs)                                  │ │
│  │     - RWMutex for thread safety                                    │ │
│  │                                                                      │ │
│  │  4. Background Workers:                                            │ │
│  │     - gcWorker: GC cleanup worker                                  │ │
│  │     - syncWorker: Cache sync worker                                │ │
│  │     - Both run in separate goroutines                              │ │
│  │                                                                      │ │
│  │  5. Observability:                                                 │ │
│  │     - metrics: *SilenceMetrics (8 metrics)                         │ │
│  │     - logger: *slog.Logger                                         │ │
│  │                                                                      │ │
│  │  6. Lifecycle Management:                                          │ │
│  │     - started: atomic.Bool                                         │ │
│  │     - shutdown: atomic.Bool                                        │ │
│  │     - wg: sync.WaitGroup (goroutine tracking)                      │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
└───────────────┬───────────────────────────────┬───────────────────────────┘
                │                               │
                │ Uses                          │ Uses
                ▼                               ▼
┌──────────────────────────────┐  ┌──────────────────────────────────┐
│   SilenceRepository          │  │   SilenceMatcher                 │
│   (TN-133)                   │  │   (TN-132)                       │
│                              │  │                                  │
│  - CreateSilence()           │  │  - Matches(alert, silence)       │
│  - GetSilenceByID()          │  │  - MatchesAny(alert, silences)   │
│  - ListSilences()            │  │  - Regex caching                 │
│  - UpdateSilence()           │  │  - Context cancellation          │
│  - DeleteSilence()           │  │                                  │
│  - ExpireSilences()          │  └──────────────────────────────────┘
│  - GetExpiringSoon()         │
│  - BulkUpdateStatus()        │
│  - CountSilences()           │
│                              │
└──────────────┬───────────────┘
               │
               │ Uses
               ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                        PostgreSQL Database                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Table: silences                                                   │ │
│  │  - 7 indexes (status, time ranges, matchers JSONB, creator)       │ │
│  │  - JSONB matchers for flexible querying                           │ │
│  │  - Timestamps with timezone support                               │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 🏗️ Component Design

### 1. DefaultSilenceManager

**Struct Definition**:
```go
package silencing

import (
    "context"
    "log/slog"
    "sync"
    "sync/atomic"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core/silencing"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// DefaultSilenceManager implements SilenceManager interface.
// It coordinates silence lifecycle, cache management, and background workers.
//
// Thread-safety: All public methods are safe for concurrent use.
// Lifecycle: Must call Start() before use, Stop() for graceful shutdown.
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

// NewDefaultSilenceManager creates a new silence manager.
//
// Parameters:
//   - repo: Silence repository (required)
//   - matcher: Silence matcher (required)
//   - logger: Structured logger (optional, defaults to slog.Default())
//   - config: Configuration (optional, defaults to DefaultSilenceManagerConfig())
//
// Returns:
//   - *DefaultSilenceManager: Initialized manager (not started)
//
// Example:
//
//     repo := silencing.NewPostgresSilenceRepository(pool, logger)
//     matcher := silencing.NewDefaultSilenceMatcher(logger)
//     manager := NewDefaultSilenceManager(repo, matcher, logger, nil)
//
//     if err := manager.Start(ctx); err != nil {
//         log.Fatal(err)
//     }
//     defer manager.Stop(ctx)
func NewDefaultSilenceManager(
    repo silencing.SilenceRepository,
    matcher silencing.SilenceMatcher,
    logger *slog.Logger,
    config *SilenceManagerConfig,
) *DefaultSilenceManager {
    if logger == nil {
        logger = slog.Default()
    }
    if config == nil {
        defaultCfg := DefaultSilenceManagerConfig()
        config = &defaultCfg
    }

    ctx, cancel := context.WithCancel(context.Background())

    sm := &DefaultSilenceManager{
        repo:    repo,
        matcher: matcher,
        cache:   newSilenceCache(),
        metrics: NewSilenceMetrics(),
        logger:  logger,
        config:  *config,
        ctx:     ctx,
        cancel:  cancel,
    }

    // Initialize workers
    sm.gcWorker = newGCWorker(repo, sm.cache, config.GCInterval, config.GCRetention, config.GCBatchSize, logger, sm.metrics)
    sm.syncWorker = newSyncWorker(repo, sm.cache, config.SyncInterval, logger, sm.metrics)

    return sm
}
```

---

### 2. In-Memory Cache

**Design Rationale**:
- **Fast lookups**: O(1) by ID, O(N) by status
- **Memory efficient**: Only active silences (~100 silences = ~100KB)
- **Thread-safe**: RWMutex for concurrent reads/writes
- **Self-healing**: Periodic sync worker rebuilds cache

**Implementation**:
```go
// silenceCache is an in-memory cache for active silences.
// Optimized for fast read access (RWMutex).
type silenceCache struct {
    mu sync.RWMutex

    // Primary store: ID → Silence
    silences map[string]*silencing.Silence

    // Secondary index: Status → IDs (for fast filtering)
    byStatus map[silencing.SilenceStatus][]string

    // Metadata
    lastSync time.Time
    size     int
}

// newSilenceCache creates a new cache.
func newSilenceCache() *silenceCache {
    return &silenceCache{
        silences: make(map[string]*silencing.Silence),
        byStatus: make(map[silencing.SilenceStatus][]string),
    }
}

// Get retrieves a silence by ID (thread-safe read).
func (c *silenceCache) Get(id string) (*silencing.Silence, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    silence, found := c.silences[id]
    return silence, found
}

// Set adds or updates a silence (thread-safe write).
func (c *silenceCache) Set(silence *silencing.Silence) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Add to primary store
    c.silences[silence.ID] = silence

    // Update status index
    c.rebuildStatusIndex()
    c.size = len(c.silences)
}

// Delete removes a silence (thread-safe write).
func (c *silenceCache) Delete(id string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    delete(c.silences, id)
    c.rebuildStatusIndex()
    c.size = len(c.silences)
}

// GetByStatus returns all silences with given status.
func (c *silenceCache) GetByStatus(status silencing.SilenceStatus) []*silencing.Silence {
    c.mu.RLock()
    defer c.mu.RUnlock()

    ids, found := c.byStatus[status]
    if !found {
        return nil
    }

    result := make([]*silencing.Silence, 0, len(ids))
    for _, id := range ids {
        if silence, ok := c.silences[id]; ok {
            result = append(result, silence)
        }
    }
    return result
}

// GetAll returns all cached silences.
func (c *silenceCache) GetAll() []*silencing.Silence {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result := make([]*silencing.Silence, 0, len(c.silences))
    for _, silence := range c.silences {
        result = append(result, silence)
    }
    return result
}

// Rebuild replaces the cache with new data.
// Used by sync worker.
func (c *silenceCache) Rebuild(silences []*silencing.Silence) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.silences = make(map[string]*silencing.Silence, len(silences))
    for _, silence := range silences {
        c.silences[silence.ID] = silence
    }

    c.rebuildStatusIndex()
    c.lastSync = time.Now()
    c.size = len(c.silences)
}

// rebuildStatusIndex rebuilds the status index.
// Must be called with c.mu.Lock() held.
func (c *silenceCache) rebuildStatusIndex() {
    c.byStatus = make(map[silencing.SilenceStatus][]string)

    for id, silence := range c.silences {
        status := silence.Status
        c.byStatus[status] = append(c.byStatus[status], id)
    }
}

// Stats returns cache statistics.
func (c *silenceCache) Stats() CacheStats {
    c.mu.RLock()
    defer c.mu.RUnlock()

    return CacheStats{
        Size:     c.size,
        LastSync: c.lastSync,
        ByStatus: map[silencing.SilenceStatus]int{
            silencing.SilenceStatusPending: len(c.byStatus[silencing.SilenceStatusPending]),
            silencing.SilenceStatusActive:  len(c.byStatus[silencing.SilenceStatusActive]),
            silencing.SilenceStatusExpired: len(c.byStatus[silencing.SilenceStatusExpired]),
        },
    }
}

// CacheStats holds cache statistics.
type CacheStats struct {
    Size     int
    LastSync time.Time
    ByStatus map[silencing.SilenceStatus]int
}
```

---

### 3. Background GC Worker

**Inspired by**: TN-124 (TTL Cleanup Worker), TN-129 (Cleanup Worker)

**Design**:
```go
// gcWorker handles periodic garbage collection of expired silences.
// Runs two phases: expire (status update) and delete (hard delete).
type gcWorker struct {
    repo      silencing.SilenceRepository
    cache     *silenceCache
    interval  time.Duration  // How often to run (default: 5m)
    retention time.Duration  // Keep expired for this long (default: 24h)
    batchSize int           // Max silences per run (default: 1000)

    logger  *slog.Logger
    metrics *SilenceMetrics

    stopCh chan struct{}
    doneCh chan struct{}
}

// newGCWorker creates a new GC worker (not started).
func newGCWorker(
    repo silencing.SilenceRepository,
    cache *silenceCache,
    interval, retention time.Duration,
    batchSize int,
    logger *slog.Logger,
    metrics *SilenceMetrics,
) *gcWorker {
    return &gcWorker{
        repo:      repo,
        cache:     cache,
        interval:  interval,
        retention: retention,
        batchSize: batchSize,
        logger:    logger,
        metrics:   metrics,
        stopCh:    make(chan struct{}),
        doneCh:    make(chan struct{}),
    }
}

// Start starts the GC worker in a goroutine.
func (w *gcWorker) Start(ctx context.Context) {
    go w.run(ctx)
    w.logger.Info("GC worker started", "interval", w.interval, "retention", w.retention)
}

// run is the main worker loop.
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

// runCleanup runs the two-phase cleanup.
func (w *gcWorker) runCleanup(ctx context.Context) {
    start := time.Now()

    // Phase 1: Expire active silences (status update)
    expiredCount, err := w.expireActivesilences(ctx)
    if err != nil {
        w.logger.Error("Failed to expire silences", "error", err)
    } else {
        w.logger.Info("Phase 1 complete", "expired_count", expiredCount)
    }

    // Phase 2: Delete old expired silences
    deletedCount, err := w.deleteOldExpired(ctx)
    if err != nil {
        w.logger.Error("Failed to delete old silences", "error", err)
    } else {
        w.logger.Info("Phase 2 complete", "deleted_count", deletedCount)
    }

    duration := time.Since(start)
    w.logger.Info("GC cleanup complete",
        "expired", expiredCount,
        "deleted", deletedCount,
        "duration", duration,
    )
}

// expireActiveSilences changes status from 'active' to 'expired' for silences past EndsAt.
func (w *gcWorker) expireActiveSilences(ctx context.Context) (int64, error) {
    start := time.Now()
    defer func() {
        w.metrics.GCDuration.WithLabelValues("expire").Observe(time.Since(start).Seconds())
    }()

    // Call repository ExpireSilences (deleteExpired=false for status update)
    count, err := w.repo.ExpireSilences(ctx, time.Now(), false)
    if err != nil {
        w.metrics.Errors.WithLabelValues("gc", "expire").Inc()
        return 0, err
    }

    w.metrics.GCRuns.WithLabelValues("expire").Inc()
    w.metrics.GCCleaned.WithLabelValues("expire").Add(float64(count))

    return count, nil
}

// deleteOldExpired deletes expired silences older than retention period.
func (w *gcWorker) deleteOldExpired(ctx context.Context) (int64, error) {
    start := time.Now()
    defer func() {
        w.metrics.GCDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
    }()

    // Delete silences where ends_at < NOW - retention AND status = 'expired'
    before := time.Now().Add(-w.retention)
    count, err := w.repo.ExpireSilences(ctx, before, true)
    if err != nil {
        w.metrics.Errors.WithLabelValues("gc", "delete").Inc()
        return 0, err
    }

    // Remove from cache if present
    // (Usually not needed since expired silences aren't cached)

    w.metrics.GCRuns.WithLabelValues("delete").Inc()
    w.metrics.GCCleaned.WithLabelValues("delete").Add(float64(count))

    return count, nil
}

// Stop stops the GC worker gracefully.
func (w *gcWorker) Stop() {
    close(w.stopCh)
    <-w.doneCh
}
```

---

### 4. Background Sync Worker

**Purpose**: Keep cache synchronized with PostgreSQL

**Design**:
```go
// syncWorker handles periodic cache synchronization.
// Refreshes cache with active silences from database.
type syncWorker struct {
    repo     silencing.SilenceRepository
    cache    *silenceCache
    interval time.Duration

    logger  *slog.Logger
    metrics *SilenceMetrics

    stopCh chan struct{}
    doneCh chan struct{}
}

// newSyncWorker creates a new sync worker (not started).
func newSyncWorker(
    repo silencing.SilenceRepository,
    cache *silenceCache,
    interval time.Duration,
    logger *slog.Logger,
    metrics *SilenceMetrics,
) *syncWorker {
    return &syncWorker{
        repo:     repo,
        cache:    cache,
        interval: interval,
        logger:   logger,
        metrics:  metrics,
        stopCh:   make(chan struct{}),
        doneCh:   make(chan struct{}),
    }
}

// Start starts the sync worker.
func (w *syncWorker) Start(ctx context.Context) {
    go w.run(ctx)
    w.logger.Info("Sync worker started", "interval", w.interval)
}

// run is the main worker loop.
func (w *syncWorker) run(ctx context.Context) {
    defer close(w.doneCh)

    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    // Run immediately on startup
    w.runSync(ctx)

    for {
        select {
        case <-ctx.Done():
            w.logger.Info("Sync worker stopped (context cancelled)")
            return

        case <-w.stopCh:
            w.logger.Info("Sync worker stopped (explicit stop)")
            return

        case <-ticker.C:
            w.runSync(ctx)
        }
    }
}

// runSync synchronizes cache with database.
func (w *syncWorker) runSync(ctx context.Context) {
    start := time.Now()

    // Fetch all active silences from database
    filter := silencing.SilenceFilter{
        Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
        Limit:    10000, // Max active silences
    }

    silences, err := w.repo.ListSilences(ctx, filter)
    if err != nil {
        w.logger.Error("Failed to sync cache", "error", err)
        w.metrics.Errors.WithLabelValues("sync", "list").Inc()
        return
    }

    // Rebuild cache
    oldSize := w.cache.Stats().Size
    w.cache.Rebuild(silences)
    newSize := len(silences)

    duration := time.Since(start)

    w.logger.Info("Cache synchronized",
        "old_size", oldSize,
        "new_size", newSize,
        "added", newSize-oldSize,
        "duration", duration,
    )

    w.metrics.SyncRuns.Inc()
    w.metrics.SyncDuration.Observe(duration.Seconds())
    w.metrics.SyncAdded.Add(float64(max(0, newSize-oldSize)))
    w.metrics.SyncRemoved.Add(float64(max(0, oldSize-newSize)))
}

// Stop stops the sync worker.
func (w *syncWorker) Stop() {
    close(w.stopCh)
    <-w.doneCh
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

---

## 🔄 Sequence Diagrams

### 5.1 CreateSilence Flow

```
Client            SilenceManager        Cache             Repository
  │                    │                  │                    │
  │  CreateSilence()   │                  │                    │
  ├───────────────────>│                  │                    │
  │                    │                  │                    │
  │                    │  Validate()      │                    │
  │                    ├─────────────────>│                    │
  │                    │                  │                    │
  │                    │  CreateSilence() │                    │
  │                    ├─────────────────────────────────────>│
  │                    │                  │                    │
  │                    │                  │      INSERT        │
  │                    │                  │   (PostgreSQL)     │
  │                    │                  │<───────────────────│
  │                    │                  │                    │
  │                    │  Silence{ID=...} │                    │
  │                    │<─────────────────────────────────────│
  │                    │                  │                    │
  │                    │  IsActive()?     │                    │
  │                    ├──────┐           │                    │
  │                    │      │           │                    │
  │                    │<─────┘           │                    │
  │                    │                  │                    │
  │                    │  cache.Set()     │                    │
  │                    ├─────────────────>│                    │
  │                    │                  │                    │
  │                    │  OK              │                    │
  │                    │<─────────────────│                    │
  │                    │                  │                    │
  │                    │  metrics.Inc()   │                    │
  │                    ├──────┐           │                    │
  │                    │      │           │                    │
  │                    │<─────┘           │                    │
  │                    │                  │                    │
  │  Silence{ID=...}   │                  │                    │
  │<───────────────────│                  │                    │
```

### 5.2 IsAlertSilenced Flow

```
AlertProcessor    SilenceManager        Cache           Matcher
  │                    │                  │                 │
  │  IsAlertSilenced() │                  │                 │
  ├───────────────────>│                  │                 │
  │                    │                  │                 │
  │                    │  GetActiveSilences()              │
  │                    ├─────────────────>│                 │
  │                    │                  │                 │
  │                    │  []*Silence      │                 │
  │                    │<─────────────────│                 │
  │                    │                  │                 │
  │                    │  for each silence│                 │
  │                    ├──────────────────┼────────────────>│
  │                    │                  │   Matches()?    │
  │                    │                  │<────────────────│
  │                    │                  │   true/false    │
  │                    │<─────────────────┼─────────────────│
  │                    │                  │                 │
  │                    │  metrics.Inc()   │                 │
  │                    ├──────┐           │                 │
  │                    │      │           │                 │
  │                    │<─────┘           │                 │
  │                    │                  │                 │
  │  (true, [IDs])     │                  │                 │
  │<───────────────────│                  │                 │
```

### 5.3 GC Worker Flow

```
GCWorker          Repository           Cache           Database
  │                    │                  │                 │
  │  Timer tick (5m)   │                  │                 │
  ├────────────────────┼──────────────────┼────────────────>│
  │                    │                  │                 │
  │  runCleanup()      │                  │                 │
  ├──────┐             │                  │                 │
  │      │             │                  │                 │
  │<─────┘             │                  │                 │
  │                    │                  │                 │
  │  Phase 1: Expire   │                  │                 │
  │  ExpireSilences()  │                  │                 │
  ├───────────────────>│                  │                 │
  │                    │                  │                 │
  │                    │  UPDATE status   │                 │
  │                    ├─────────────────────────────────>│
  │                    │                  │                 │
  │                    │  count=N         │                 │
  │                    │<─────────────────────────────────│
  │                    │                  │                 │
  │  count=N           │                  │                 │
  │<───────────────────│                  │                 │
  │                    │                  │                 │
  │  Phase 2: Delete   │                  │                 │
  │  ExpireSilences()  │                  │                 │
  ├───────────────────>│                  │                 │
  │                    │                  │                 │
  │                    │  DELETE          │                 │
  │                    ├─────────────────────────────────>│
  │                    │                  │                 │
  │                    │  count=M         │                 │
  │                    │<─────────────────────────────────│
  │                    │                  │                 │
  │  count=M           │                  │                 │
  │<───────────────────│                  │                 │
  │                    │                  │                 │
  │  metrics.Observe() │                  │                 │
  ├──────┐             │                  │                 │
  │      │             │                  │                 │
  │<─────┘             │                  │                 │
```

---

## 📊 Data Structures

### SilenceManagerConfig

```go
// SilenceManagerConfig holds configuration for DefaultSilenceManager.
type SilenceManagerConfig struct {
    // GC Worker Settings
    GCInterval    time.Duration  // How often to run GC (default: 5m)
    GCRetention   time.Duration  // Keep expired for this long (default: 24h)
    GCBatchSize   int           // Max silences per GC run (default: 1000)

    // Sync Worker Settings
    SyncInterval  time.Duration  // How often to sync cache (default: 1m)

    // Cache Settings
    CacheEnabled  bool          // Enable in-memory cache (default: true)
    CacheTTL      time.Duration  // Not used (cache always fresh)

    // Shutdown Settings
    ShutdownTimeout time.Duration  // Max time for graceful shutdown (default: 30s)
}

// DefaultSilenceManagerConfig returns default configuration.
func DefaultSilenceManagerConfig() SilenceManagerConfig {
    return SilenceManagerConfig{
        GCInterval:      5 * time.Minute,
        GCRetention:     24 * time.Hour,
        GCBatchSize:     1000,
        SyncInterval:    1 * time.Minute,
        CacheEnabled:    true,
        CacheTTL:        5 * time.Minute, // Not used
        ShutdownTimeout: 30 * time.Second,
    }
}
```

### SilenceManagerStats

```go
// SilenceManagerStats holds statistics about the silence manager.
type SilenceManagerStats struct {
    // Cache Stats
    CacheSize       int
    CacheLastSync   time.Time
    CacheByStatus   map[silencing.SilenceStatus]int

    // Repository Stats
    TotalSilences   int64
    ActiveSilences  int64
    PendingSilences int64
    ExpiredSilences int64

    // Worker Stats
    GCLastRun       time.Time
    GCTotalRuns     int64
    GCTotalCleaned  int64
    SyncLastRun     time.Time
    SyncTotalRuns   int64
}
```

---

## 🎯 Error Handling Strategy

### Error Types

```go
package silencing

import "errors"

var (
    // Manager lifecycle errors
    ErrManagerNotStarted = errors.New("silence manager not started")
    ErrManagerShutdown   = errors.New("silence manager is shutting down")

    // Operation errors
    ErrInvalidAlert      = errors.New("invalid alert")
    ErrCacheUnavailable  = errors.New("cache unavailable")

    // Worker errors (internal, not returned to client)
    errWorkerStopped     = errors.New("worker stopped")
    errSyncFailed        = errors.New("cache sync failed")
)
```

### Error Handling Patterns

**1. CRUD Operations**:
```go
func (sm *DefaultSilenceManager) CreateSilence(ctx context.Context, silence *silencing.Silence) (*silencing.Silence, error) {
    // Check manager state
    if !sm.started.Load() {
        return nil, ErrManagerNotStarted
    }
    if sm.shutdown.Load() {
        return nil, ErrManagerShutdown
    }

    // Delegate to repository (propagate errors)
    created, err := sm.repo.CreateSilence(ctx, silence)
    if err != nil {
        sm.metrics.Errors.WithLabelValues("create", "repo").Inc()
        return nil, fmt.Errorf("create silence: %w", err)
    }

    // Update cache (log errors, don't fail operation)
    if silence.IsActive() {
        sm.cache.Set(created)
    }

    return created, nil
}
```

**2. Alert Filtering** (fail-safe):
```go
func (sm *DefaultSilenceManager) IsAlertSilenced(ctx context.Context, alert *Alert) (bool, []string, error) {
    // Validate input
    if alert == nil || alert.Labels == nil {
        return false, nil, ErrInvalidAlert
    }

    // Get active silences (graceful degradation)
    silences, err := sm.GetActiveSilences(ctx)
    if err != nil {
        sm.logger.Warn("Failed to get active silences, assuming not silenced", "error", err)
        return false, nil, nil  // Fail open (don't block alerts)
    }

    // Check each silence (log errors, continue)
    var matchedIDs []string
    for _, silence := range silences {
        matched, err := sm.matcher.Matches(ctx, alert, silence)
        if err != nil {
            sm.logger.Warn("Matcher error, skipping silence", "silence_id", silence.ID, "error", err)
            continue
        }
        if matched {
            matchedIDs = append(matchedIDs, silence.ID)
        }
    }

    return len(matchedIDs) > 0, matchedIDs, nil
}
```

**3. Background Workers** (log and continue):
```go
func (w *gcWorker) runCleanup(ctx context.Context) {
    // Phase 1: Expire (don't fail on error)
    expiredCount, err := w.expireActiveSilences(ctx)
    if err != nil {
        w.logger.Error("Failed to expire silences", "error", err)
        // Continue to phase 2
    }

    // Phase 2: Delete (don't fail on error)
    deletedCount, err := w.deleteOldExpired(ctx)
    if err != nil {
        w.logger.Error("Failed to delete old silences", "error", err)
        // Continue (retry on next tick)
    }
}
```

---

## 📈 Performance Optimization

### 1. Cache Optimization

**Read-heavy workload**:
```go
// Use RWMutex for concurrent reads
func (c *silenceCache) Get(id string) (*silencing.Silence, bool) {
    c.mu.RLock()  // Multiple readers allowed
    defer c.mu.RUnlock()

    silence, found := c.silences[id]
    return silence, found
}
```

**Memory efficiency**:
- Only cache active silences (~100-1000 silences)
- Expected memory: ~100KB for 1000 silences
- No TTL (cache always synchronized)

### 2. Alert Filtering Optimization

**Early exit**:
```go
func (sm *DefaultSilenceManager) IsAlertSilenced(ctx context.Context, alert *Alert) (bool, []string, error) {
    silences := sm.cache.GetByStatus(silencing.SilenceStatusActive)

    for _, silence := range silences {
        matched, _ := sm.matcher.Matches(ctx, alert, silence)
        if matched {
            return true, []string{silence.ID}, nil  // Early exit on first match
        }
    }

    return false, nil, nil
}
```

**Performance target**: <500µs for 100 active silences

### 3. Database Query Optimization

**Use repository indexes**:
- `idx_silences_status` - Fast filtering by status
- `idx_silences_active` - Composite (status, ends_at) for active silences
- `idx_silences_ends_at` - Fast GC queries (WHERE ends_at < ?)

---

## 🔒 Thread Safety

### Concurrency Guarantees

1. **Cache Access**: Protected by `sync.RWMutex`
2. **Manager State**: Protected by `atomic.Bool` (started, shutdown)
3. **Workers**: Independent goroutines, no shared state
4. **Repository**: Thread-safe (pgxpool handles concurrency)

### Race Condition Prevention

**Test command**:
```bash
go test -race -count=10 ./internal/infrastructure/silencing/...
```

**Expected**: Zero race conditions detected

---

## 📚 Dependencies

### Internal Dependencies (Required)
- `internal/core/silencing` - Domain models (Silence, Matcher, SilenceStatus)
- `internal/infrastructure/silencing` - Repository interface, PostgresSilenceRepository
- `internal/infrastructure/silencing` - SilenceMatcher interface, DefaultSilenceMatcher

### External Dependencies
- `github.com/prometheus/client_golang/prometheus` - Metrics
- `log/slog` - Structured logging
- `context` - Cancellation, timeouts
- `sync` - Thread safety (RWMutex, WaitGroup)
- `sync/atomic` - Atomic operations (Bool)
- `time` - Timers, tickers

---

## 🎯 Testing Strategy

### Unit Tests (40+ tests)
- CRUD operations (10 tests)
- Cache operations (8 tests)
- Alert filtering (10 tests)
- GC worker (6 tests)
- Sync worker (4 tests)
- Thread safety (5 tests)
- Error handling (5 tests)

### Integration Tests (10+ tests)
- Manager + Repository integration
- Manager + Matcher integration
- End-to-end silence lifecycle
- Background workers
- Graceful shutdown

### Benchmarks (8+ benchmarks)
- `BenchmarkGetSilence_Cached` - Target: <100µs
- `BenchmarkIsAlertSilenced_100Silences` - Target: <500µs
- `BenchmarkCreateSilence` - Target: <15ms
- `BenchmarkGCWorker_1000Silences` - Target: <2s
- `BenchmarkSyncWorker_1000Silences` - Target: <50ms

---

## 📖 API Documentation

See `README.md` for comprehensive API documentation.

---

**Version**: 1.0
**Author**: TN-134 Implementation Team
**Last Updated**: 2025-11-06
**Status**: Ready for Implementation 🚀



