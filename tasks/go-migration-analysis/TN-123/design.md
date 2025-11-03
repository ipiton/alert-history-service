# TN-123: Alert Group Manager - Technical Design

## 1. Архитектурное решение

Alert Group Manager реализует **Repository + Manager Pattern** для управления жизненным циклом групп алертов.

### Архитектурная диаграмма

```
┌─────────────────────────────────────────────────────────────┐
│                    AlertProcessor                           │
│  (orchestrates alert processing pipeline)                   │
└───────────────┬─────────────────────────────────────────────┘
                │
                ├──> Deduplication Service (TN-036)
                │
                ├──> AlertGroupManager (TN-123) ◄─── THIS TASK
                │         │
                │         ├──> GroupKeyGenerator (TN-122)
                │         │
                │         ├──> AlertGroupStorage (in-memory / Redis)
                │         │
                │         └──> GroupMetrics (Prometheus)
                │
                ├──> Classification Service (TN-033)
                │
                └──> Publisher
```

### Ключевые компоненты

1. **AlertGroupManager** (interface) - публичный API для работы с группами
2. **DefaultGroupManager** (implementation) - in-memory хранилище + бизнес-логика
3. **AlertGroup** (data model) - контейнер для алертов в группе
4. **GroupStorage** (interface) - абстракция хранилища (готовность к Redis)
5. **GroupMetrics** (observability) - Prometheus metrics

---

## 2. Data Models

### 2.1 AlertGroup

```go
// AlertGroup представляет группу связанных алертов
type AlertGroup struct {
    // Ключ группировки (из TN-122)
    Key grouping.GroupKey `json:"key"`

    // Алерты в группе (map для быстрого доступа по fingerprint)
    Alerts map[string]*core.Alert `json:"alerts"`

    // Метаданные группы
    Metadata *GroupMetadata `json:"metadata"`

    // Mutex для thread-safe доступа (150% enhancement)
    mu sync.RWMutex `json:"-"`
}
```

### 2.2 GroupMetadata

```go
// GroupMetadata содержит метаданные о состоянии группы
type GroupMetadata struct {
    // Состояние группы (firing/resolved/mixed)
    State GroupState `json:"state"`

    // Время создания группы
    CreatedAt time.Time `json:"created_at"`

    // Время последнего обновления
    UpdatedAt time.Time `json:"updated_at"`

    // Время первого firing алерта
    FirstFiringAt *time.Time `json:"first_firing_at,omitempty"`

    // Время, когда все алерты стали resolved
    ResolvedAt *time.Time `json:"resolved_at,omitempty"`

    // Количество firing алертов
    FiringCount int `json:"firing_count"`

    // Количество resolved алертов
    ResolvedCount int `json:"resolved_count"`

    // Конфигурация группировки (для reference)
    GroupBy []string `json:"group_by"`

    // Версия состояния (для optimistic locking, Redis)
    Version int64 `json:"version"`
}
```

### 2.3 GroupState

```go
// GroupState представляет состояние группы
type GroupState string

const (
    // GroupStateFiring - все алерты firing
    GroupStateFiring GroupState = "firing"

    // GroupStateResolved - все алерты resolved
    GroupStateResolved GroupState = "resolved"

    // GroupStateMixed - есть и firing, и resolved алерты
    GroupStateMixed GroupState = "mixed"

    // GroupStateSilenced - группа silenced (TN-133+)
    GroupStateSilenced GroupState = "silenced"
)
```

### 2.4 GroupMetrics

```go
// GroupMetrics содержит метрики по группам
type GroupMetrics struct {
    // Количество активных групп
    ActiveGroups int `json:"active_groups"`

    // Количество алертов по группам
    AlertsPerGroup map[string]int `json:"alerts_per_group"` // key -> count

    // Распределение размеров групп
    SizeDistribution map[string]int `json:"size_distribution"` // "1-10", "11-50", etc.

    // Операции с группами (add/remove/cleanup)
    Operations map[string]int64 `json:"operations"` // "add", "remove", "cleanup"

    // Snapshot timestamp
    Timestamp time.Time `json:"timestamp"`
}
```

---

## 3. Interfaces

### 3.1 AlertGroupManager (Core Interface)

```go
package grouping

import (
    "context"
    "time"
    "github.com/vitaliisemenov/alert-history/internal/core"
)

// AlertGroupManager управляет жизненным циклом групп алертов.
// Thread-safe, поддерживает concurrent access.
type AlertGroupManager interface {
    // === Lifecycle Management ===

    // AddAlertToGroup добавляет алерт в группу по ключу группировки.
    // Если группа не существует - создает новую.
    // Если алерт уже в группе - обновляет его.
    //
    // Parameters:
    //   - ctx: контекст с таймаутом и cancellation
    //   - alert: алерт для добавления (должен иметь fingerprint)
    //   - groupKey: ключ группировки (из GroupKeyGenerator)
    //
    // Returns:
    //   - *AlertGroup: обновленная группа
    //   - error: InvalidAlertError, StorageError
    AddAlertToGroup(ctx context.Context, alert *core.Alert, groupKey GroupKey) (*AlertGroup, error)

    // RemoveAlertFromGroup удаляет алерт из группы.
    // Если группа становится пустой - автоматически удаляет группу.
    //
    // Parameters:
    //   - ctx: контекст
    //   - fingerprint: fingerprint алерта для удаления
    //   - groupKey: ключ группы
    //
    // Returns:
    //   - bool: true если алерт был удален, false если не найден
    //   - error: GroupNotFoundError, StorageError
    RemoveAlertFromGroup(ctx context.Context, fingerprint string, groupKey GroupKey) (bool, error)

    // UpdateGroupState пересчитывает и обновляет состояние группы.
    // Вызывается автоматически при add/remove алертов.
    //
    // Parameters:
    //   - ctx: контекст
    //   - groupKey: ключ группы
    //
    // Returns:
    //   - *AlertGroup: обновленная группа с новым состоянием
    //   - error: GroupNotFoundError, StorageError
    UpdateGroupState(ctx context.Context, groupKey GroupKey) (*AlertGroup, error)

    // CleanupExpiredGroups удаляет группы, которые неактивны более maxAge.
    // Группа считается expired если:
    //   - Все алерты resolved ИЛИ
    //   - UpdatedAt > maxAge назад
    //
    // Parameters:
    //   - ctx: контекст с таймаутом
    //   - maxAge: максимальный возраст группы (e.g., 24h)
    //
    // Returns:
    //   - int: количество удаленных групп
    //   - error: StorageError
    CleanupExpiredGroups(ctx context.Context, maxAge time.Duration) (int, error)

    // === Query Operations ===

    // GetGroup возвращает группу по ключу.
    //
    // Returns:
    //   - *AlertGroup: группа с алертами
    //   - error: GroupNotFoundError, StorageError
    GetGroup(ctx context.Context, groupKey GroupKey) (*AlertGroup, error)

    // ListGroups возвращает список всех активных групп.
    // Supports pagination для больших объемов (150% enhancement).
    //
    // Parameters:
    //   - ctx: контекст
    //   - filters: опциональные фильтры (state, minSize, maxAge)
    //
    // Returns:
    //   - []*AlertGroup: список групп
    //   - error: StorageError
    ListGroups(ctx context.Context, filters *GroupFilters) ([]*AlertGroup, error)

    // GetGroupByFingerprint возвращает группу, содержащую алерт с fingerprint.
    // Полезно для поиска "в какой группе находится этот алерт".
    //
    // Returns:
    //   - GroupKey: ключ группы
    //   - *AlertGroup: группа
    //   - error: GroupNotFoundError
    GetGroupByFingerprint(ctx context.Context, fingerprint string) (GroupKey, *AlertGroup, error)

    // === Metrics & Observability ===

    // GetMetrics возвращает текущие метрики по группам.
    // Используется для Prometheus scraping и monitoring.
    //
    // Returns:
    //   - *GroupMetrics: snapshot метрик
    //   - error: StorageError
    GetMetrics(ctx context.Context) (*GroupMetrics, error)

    // GetStats возвращает детальную статистику по группам.
    // 150% enhancement для advanced monitoring.
    //
    // Returns:
    //   - *GroupStats: детальная статистика
    //   - error: StorageError
    GetStats(ctx context.Context) (*GroupStats, error)
}
```

### 3.2 GroupStorage (Abstraction Layer)

```go
// GroupStorage абстрагирует хранилище групп.
// Реализации: InMemoryStorage (TN-123), RedisStorage (TN-125).
type GroupStorage interface {
    // Store сохраняет группу
    Store(ctx context.Context, group *AlertGroup) error

    // Load загружает группу по ключу
    Load(ctx context.Context, groupKey GroupKey) (*AlertGroup, error)

    // Delete удаляет группу
    Delete(ctx context.Context, groupKey GroupKey) error

    // ListKeys возвращает список ключей всех групп
    ListKeys(ctx context.Context) ([]GroupKey, error)

    // Size возвращает количество групп
    Size(ctx context.Context) (int, error)
}
```

### 3.3 GroupFilters (Query Support)

```go
// GroupFilters определяет фильтры для ListGroups
type GroupFilters struct {
    // Фильтр по состоянию
    State *GroupState `json:"state,omitempty"`

    // Минимальное количество алертов в группе
    MinSize *int `json:"min_size,omitempty"`

    // Максимальный возраст группы
    MaxAge *time.Duration `json:"max_age,omitempty"`

    // Пагинация (150% enhancement)
    Limit  int `json:"limit,omitempty"`
    Offset int `json:"offset,omitempty"`
}
```

---

## 4. Implementation: DefaultGroupManager

### 4.1 Структура

```go
package grouping

import (
    "context"
    "fmt"
    "log/slog"
    "sync"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// DefaultGroupManager - in-memory реализация AlertGroupManager
type DefaultGroupManager struct {
    // Хранилище групп: map[GroupKey]*AlertGroup
    groups map[GroupKey]*AlertGroup

    // Обратный индекс: map[fingerprint]GroupKey
    // Для быстрого поиска группы по fingerprint (150% enhancement)
    fingerprintIndex map[string]GroupKey

    // Mutex для thread-safe access
    mu sync.RWMutex

    // GroupKeyGenerator для генерации ключей
    keyGenerator *GroupKeyGenerator

    // Конфигурация группировки (from TN-121)
    config *GroupingConfig

    // Observability
    logger  *slog.Logger
    metrics *metrics.BusinessMetrics

    // Statistics (in-memory, for GetStats)
    stats *groupStats
}

// groupStats хранит статистику операций
type groupStats struct {
    totalAdds       int64
    totalRemoves    int64
    totalCleanups   int64
    totalUpdates    int64
    lastCleanupTime time.Time
    mu              sync.RWMutex
}
```

### 4.2 Constructor

```go
// NewDefaultGroupManager создает новый DefaultGroupManager
func NewDefaultGroupManager(config DefaultGroupManagerConfig) (*DefaultGroupManager, error) {
    // Validation
    if config.KeyGenerator == nil {
        return nil, fmt.Errorf("key generator is required")
    }
    if config.Config == nil {
        return nil, fmt.Errorf("grouping config is required")
    }

    // Defaults
    if config.Logger == nil {
        config.Logger = slog.Default()
    }

    return &DefaultGroupManager{
        groups:           make(map[GroupKey]*AlertGroup),
        fingerprintIndex: make(map[string]GroupKey),
        keyGenerator:     config.KeyGenerator,
        config:           config.Config,
        logger:           config.Logger,
        metrics:          config.Metrics,
        stats:            &groupStats{},
    }, nil
}
```

### 4.3 Core Methods

#### AddAlertToGroup

```go
func (m *DefaultGroupManager) AddAlertToGroup(
    ctx context.Context,
    alert *core.Alert,
    groupKey GroupKey,
) (*AlertGroup, error) {
    startTime := time.Now()

    // Validation
    if alert == nil {
        return nil, &InvalidAlertError{Reason: "alert is nil"}
    }
    if alert.Fingerprint == "" {
        return nil, &InvalidAlertError{Reason: "alert fingerprint is empty"}
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    // Get or create group
    group, exists := m.groups[groupKey]
    if !exists {
        group = m.createNewGroup(groupKey)
        m.groups[groupKey] = group

        m.logger.Info("Created new alert group",
            "group_key", groupKey,
            "alert", alert.AlertName)
    }

    // Add alert to group (thread-safe)
    group.mu.Lock()
    isNew := group.Alerts[alert.Fingerprint] == nil
    group.Alerts[alert.Fingerprint] = alert
    group.mu.Unlock()

    // Update fingerprint index
    m.fingerprintIndex[alert.Fingerprint] = groupKey

    // Update group state
    m.updateGroupStateUnsafe(group)

    // Update stats
    m.stats.mu.Lock()
    m.stats.totalAdds++
    m.stats.mu.Unlock()

    // Metrics
    if m.metrics != nil {
        m.recordAddMetrics(groupKey, isNew, time.Since(startTime))
    }

    m.logger.Debug("Added alert to group",
        "group_key", groupKey,
        "alert", alert.AlertName,
        "fingerprint", alert.Fingerprint,
        "group_size", len(group.Alerts),
        "is_new", isNew)

    return group, nil
}
```

#### RemoveAlertFromGroup

```go
func (m *DefaultGroupManager) RemoveAlertFromGroup(
    ctx context.Context,
    fingerprint string,
    groupKey GroupKey,
) (bool, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    group, exists := m.groups[groupKey]
    if !exists {
        return false, &GroupNotFoundError{Key: groupKey}
    }

    // Remove alert from group
    group.mu.Lock()
    _, existed := group.Alerts[fingerprint]
    delete(group.Alerts, fingerprint)
    group.mu.Unlock()

    if !existed {
        return false, nil
    }

    // Remove from fingerprint index
    delete(m.fingerprintIndex, fingerprint)

    // If group is empty - delete group
    if len(group.Alerts) == 0 {
        delete(m.groups, groupKey)

        m.logger.Info("Deleted empty alert group",
            "group_key", groupKey)

        // Metrics: active_groups decrease
        if m.metrics != nil {
            m.metrics.RecordGroupDeleted()
        }
    } else {
        // Update group state
        m.updateGroupStateUnsafe(group)
    }

    // Stats
    m.stats.mu.Lock()
    m.stats.totalRemoves++
    m.stats.mu.Unlock()

    return true, nil
}
```

#### GetGroup

```go
func (m *DefaultGroupManager) GetGroup(
    ctx context.Context,
    groupKey GroupKey,
) (*AlertGroup, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    group, exists := m.groups[groupKey]
    if !exists {
        return nil, &GroupNotFoundError{Key: groupKey}
    }

    // Return a shallow copy to prevent external mutation (150% enhancement)
    return group.Clone(), nil
}
```

#### CleanupExpiredGroups

```go
func (m *DefaultGroupManager) CleanupExpiredGroups(
    ctx context.Context,
    maxAge time.Duration,
) (int, error) {
    startTime := time.Now()
    cutoffTime := startTime.Add(-maxAge)

    m.mu.Lock()
    defer m.mu.Unlock()

    expiredKeys := make([]GroupKey, 0)

    // Find expired groups
    for key, group := range m.groups {
        group.mu.RLock()
        isExpired := m.isGroupExpired(group, cutoffTime)
        group.mu.RUnlock()

        if isExpired {
            expiredKeys = append(expiredKeys, key)
        }
    }

    // Delete expired groups
    for _, key := range expiredKeys {
        group := m.groups[key]

        // Remove fingerprints from index
        group.mu.RLock()
        for fingerprint := range group.Alerts {
            delete(m.fingerprintIndex, fingerprint)
        }
        group.mu.RUnlock()

        // Delete group
        delete(m.groups, key)
    }

    deletedCount := len(expiredKeys)

    // Stats
    m.stats.mu.Lock()
    m.stats.totalCleanups += int64(deletedCount)
    m.stats.lastCleanupTime = startTime
    m.stats.mu.Unlock()

    m.logger.Info("Cleaned up expired groups",
        "deleted_count", deletedCount,
        "max_age", maxAge,
        "duration", time.Since(startTime))

    // Metrics
    if m.metrics != nil {
        m.metrics.RecordGroupsCleanedUp(deletedCount)
    }

    return deletedCount, nil
}

func (m *DefaultGroupManager) isGroupExpired(group *AlertGroup, cutoffTime time.Time) bool {
    // Group is expired if:
    // 1. All alerts are resolved AND resolved_at > cutoff
    // 2. OR updated_at > cutoff (no activity)

    if group.Metadata.State == GroupStateResolved {
        if group.Metadata.ResolvedAt != nil && group.Metadata.ResolvedAt.Before(cutoffTime) {
            return true
        }
    }

    if group.Metadata.UpdatedAt.Before(cutoffTime) {
        return true
    }

    return false
}
```

---

## 5. Prometheus Metrics

### 5.1 Metrics Definition

```go
// Регистрация метрик в pkg/metrics/business.go

// Active groups gauge
activeGroupsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
    Namespace: "alert_history",
    Subsystem: "business",
    Name:      "alert_groups_active_total",
    Help:      "Number of currently active alert groups",
})

// Alerts per group histogram
alertsPerGroupHist = prometheus.NewHistogram(prometheus.HistogramOpts{
    Namespace: "alert_history",
    Subsystem: "business",
    Name:      "alert_group_size",
    Help:      "Distribution of alert group sizes",
    Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
})

// Group operations counter
groupOperationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
    Namespace: "alert_history",
    Subsystem: "business",
    Name:      "alert_group_operations_total",
    Help:      "Total number of group operations",
}, []string{"operation", "result"}) // operation: add/remove/cleanup, result: success/error

// Group operation duration histogram
groupOperationDurationHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
    Namespace: "alert_history",
    Subsystem: "business",
    Name:      "alert_group_operation_duration_seconds",
    Help:      "Duration of group operations",
    Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
}, []string{"operation"})
```

### 5.2 Usage Example

```go
// In DefaultGroupManager methods

func (m *DefaultGroupManager) recordAddMetrics(groupKey GroupKey, isNew bool, duration time.Duration) {
    m.metrics.RecordGroupOperation("add", "success")
    m.metrics.RecordGroupOperationDuration("add", duration)

    if isNew {
        m.metrics.IncActiveGroups()
    }

    // Record group size (async to avoid lock contention)
    go func() {
        group, _ := m.GetGroup(context.Background(), groupKey)
        if group != nil {
            m.metrics.RecordGroupSize(len(group.Alerts))
        }
    }()
}
```

---

## 6. Error Types

```go
// InvalidAlertError - алерт не прошел валидацию
type InvalidAlertError struct {
    Reason string
}

func (e *InvalidAlertError) Error() string {
    return fmt.Sprintf("invalid alert: %s", e.Reason)
}

// GroupNotFoundError - группа не найдена
type GroupNotFoundError struct {
    Key GroupKey
}

func (e *GroupNotFoundError) Error() string {
    return fmt.Sprintf("group not found: %s", e.Key)
}

// StorageError - ошибка хранилища (для Redis в TN-125)
type StorageError struct {
    Operation string
    Err       error
}

func (e *StorageError) Error() string {
    return fmt.Sprintf("storage error during %s: %v", e.Operation, e.Err)
}

func (e *StorageError) Unwrap() error {
    return e.Err
}
```

---

## 7. Integration Points

### 7.1 AlertProcessor Integration

```go
// In alert_processor.go

type AlertProcessor struct {
    // ... existing fields ...
    groupManager grouping.AlertGroupManager // NEW
}

func (p *AlertProcessor) ProcessAlert(ctx context.Context, alert *core.Alert) error {
    // ... existing deduplication ...

    // NEW: Group management
    if p.groupManager != nil {
        // Generate group key
        groupKey, err := p.generateGroupKey(alert)
        if err != nil {
            p.logger.Warn("Failed to generate group key", "error", err)
        } else {
            // Add alert to group
            _, err = p.groupManager.AddAlertToGroup(ctx, alert, groupKey)
            if err != nil {
                p.logger.Error("Failed to add alert to group", "error", err)
                // Continue processing (graceful degradation)
            }
        }
    }

    // ... existing classification, filtering, publishing ...
}

func (p *AlertProcessor) generateGroupKey(alert *core.Alert) (grouping.GroupKey, error) {
    // Get grouping config (from config manager or hardcoded)
    groupBy := []string{"alertname", "namespace"} // TODO: get from config

    // Generate key using TN-122 GroupKeyGenerator
    return p.keyGenerator.GenerateKey(alert.Labels, groupBy)
}
```

### 7.2 HTTP API Endpoints

```go
// In cmd/server/main.go

// GET /api/v1/groups - list all groups
app.Get("/api/v1/groups", handlers.HandleListGroups(groupManager))

// GET /api/v1/groups/:key - get specific group
app.Get("/api/v1/groups/:key", handlers.HandleGetGroup(groupManager))

// GET /api/v1/groups/metrics - get group metrics
app.Get("/api/v1/groups/metrics", handlers.HandleGroupMetrics(groupManager))

// DELETE /api/v1/groups/cleanup - trigger cleanup
app.Delete("/api/v1/groups/cleanup", handlers.HandleGroupCleanup(groupManager))
```

---

## 8. Testing Strategy

### 8.1 Unit Tests (95%+ coverage)

```go
// manager_test.go

func TestDefaultGroupManager_AddAlertToGroup(t *testing.T) {
    tests := []struct {
        name      string
        alert     *core.Alert
        groupKey  GroupKey
        wantErr   bool
        errType   error
    }{
        {
            name:     "add_first_alert_to_new_group",
            alert:    createTestAlert("HighCPU", "firing"),
            groupKey: "alertname=HighCPU",
            wantErr:  false,
        },
        {
            name:     "add_second_alert_to_existing_group",
            alert:    createTestAlert("HighCPU", "firing"),
            groupKey: "alertname=HighCPU",
            wantErr:  false,
        },
        {
            name:     "error_nil_alert",
            alert:    nil,
            groupKey: "alertname=HighCPU",
            wantErr:  true,
            errType:  &InvalidAlertError{},
        },
        // ... 20+ more test cases ...
    }
}
```

### 8.2 Integration Tests

```go
func TestGroupManager_Integration_WithAlertProcessor(t *testing.T) {
    // Setup
    groupManager := NewDefaultGroupManager(...)
    alertProcessor := services.NewAlertProcessor(...)

    // Send 10 alerts with same grouping labels
    for i := 0; i < 10; i++ {
        alert := createTestAlert(fmt.Sprintf("HighCPU-%d", i), "firing")
        err := alertProcessor.ProcessAlert(context.Background(), alert)
        require.NoError(t, err)
    }

    // Verify: all alerts grouped correctly
    groups, err := groupManager.ListGroups(context.Background(), nil)
    require.NoError(t, err)
    assert.Equal(t, 1, len(groups)) // Only one group
    assert.Equal(t, 10, len(groups[0].Alerts))
}
```

### 8.3 Benchmarks

```go
func BenchmarkAddAlertToGroup(b *testing.B) {
    manager := createBenchmarkManager()
    alert := createTestAlert("HighCPU", "firing")
    groupKey := GroupKey("alertname=HighCPU")
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = manager.AddAlertToGroup(ctx, alert, groupKey)
    }
}

// Target: <500μs per operation
```

---

## 9. Performance Targets (150%)

| Operation | Baseline Target | 150% Target | Implementation |
|-----------|-----------------|-------------|----------------|
| AddAlertToGroup | <1ms | <500μs | Map lookup + pointer assignment |
| GetGroup | <500μs | <100μs | Direct map access with RLock |
| ListGroups (1K) | <10ms | <5ms | Pre-allocated slice, no deep copy |
| RemoveAlert | <1ms | <500μs | Map deletion + index cleanup |
| CleanupExpired | <100ms | <50ms | Batch deletion, minimal allocations |
| Memory/group | <10KB | <5KB | Lean structs, shared pointers |

---

## 10. Future Extensions (Post-150%)

1. **Redis Backend (TN-125)**
   - Distributed storage
   - Persistence across restarts
   - Multi-instance support

2. **Group Timers (TN-124)**
   - group_wait, group_interval, repeat_interval
   - Notification scheduling
   - Timer persistence

3. **Advanced Queries**
   - Filter by labels
   - Time-range queries
   - Aggregations (top groups, trending)

4. **Clustering Support**
   - Consistent hashing for group distribution
   - Leader election for cleanup jobs
   - State replication

---

## 11. Acceptance Criteria (Design Validation)

- [x] All interfaces defined with comprehensive documentation
- [x] Data models support all required operations
- [x] Error types cover all failure modes
- [x] Integration points clearly defined
- [x] Prometheus metrics aligned with observability goals
- [x] Performance targets achievable with proposed design
- [x] Thread-safety guaranteed via sync.RWMutex
- [x] Extensibility for future features (Redis, timers, clustering)

**Design Status**: ✅ APPROVED FOR IMPLEMENTATION
