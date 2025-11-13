# TN-060: Metrics-Only Mode Fallback - Design Document

**Version**: 1.0
**Date**: 2025-01-13
**Status**: Design Complete
**Quality Target**: 150%+ (Grade A+, Enterprise-Grade)
**Branch**: `feature/TN-060-metrics-only-mode-150pct`

---

## 📋 Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Design](#component-design)
3. [Integration Points](#integration-points)
4. [Data Structures](#data-structures)
5. [Algorithms](#algorithms)
6. [Performance Considerations](#performance-considerations)
7. [Error Handling](#error-handling)
8. [Testing Strategy](#testing-strategy)
9. [Implementation Plan](#implementation-plan)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Publishing System                           │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         ModeManager (Centralized State)              │   │
│  │  - Current mode: ModeNormal | ModeMetricsOnly        │   │
│  │  - Transition detection                              │   │
│  │  - Event notifications                               │   │
│  │  - Metrics collection                                │   │
│  └───────────────┬──────────────────────────────────────┘   │
│                  │                                           │
│         ┌────────┴────────┐                                 │
│         │                 │                                 │
│    ┌────▼────┐      ┌─────▼─────┐                          │
│    │ Normal  │      │ Metrics-  │                          │
│    │  Mode   │◄────►│  Only Mode│                          │
│    └────┬────┘      └─────┬──────┘                          │
│         │                 │                                 │
│    ┌────┴─────────────────┴────┐                          │
│    │                             │                          │
│ ┌──▼──────────┐         ┌────────▼──────┐                 │
│ │ SubmitAlert │         │ Queue Workers │                  │
│ │  (checks    │         │  (skip in     │                  │
│ │   mode)     │         │   metrics-    │                  │
│ └─────────────┘         │   only mode)  │                  │
│                         └───────────────┘                 │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         Observability Layer                           │   │
│  │  - Prometheus metrics                                 │   │
│  │  - Structured logging                                 │   │
│  │  - API endpoints                                      │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Interaction

```
ModeManager
    │
    ├──► Reads: TargetDiscoveryManager.ListTargets()
    │
    ├──► Injects: PublishingQueue (mode check in workers)
    │
    ├──► Injects: PublishingCoordinator (mode check before publish)
    │
    ├──► Injects: ParallelPublisher (mode check before parallel publish)
    │
    ├──► Injects: SubmitAlert Handler (mode check before queue)
    │
    └──► Exports: Prometheus Metrics, Structured Logs
```

### 1.3 Design Principles

1. **Centralized State Management**: Единый источник истины для режима
2. **Event-Driven Updates**: Реагирование на изменения targets
3. **Non-Blocking**: Проверки режима не блокируют hot paths
4. **Thread-Safe**: Все операции thread-safe
5. **Observable**: Полная наблюдаемость через метрики и логи

---

## 2. Component Design

### 2.1 ModeManager Interface

```go
package publishing

import (
    "context"
    "time"
)

// Mode represents the current publishing mode
type Mode int

const (
    ModeNormal Mode = iota
    ModeMetricsOnly
)

func (m Mode) String() string {
    switch m {
    case ModeNormal:
        return "normal"
    case ModeMetricsOnly:
        return "metrics-only"
    default:
        return "unknown"
    }
}

// ModeChangeCallback is called when mode changes
type ModeChangeCallback func(from Mode, to Mode, reason string)

// UnsubscribeFunc unsubscribes from mode changes
type UnsubscribeFunc func()

// ModeMetrics contains metrics about mode state
type ModeMetrics struct {
    CurrentMode              Mode
    CurrentModeDuration      time.Duration
    TransitionCount          int64
    LastTransitionTime       time.Time
    LastTransitionReason     string
    ModeCheckDuration        time.Duration
}

// ModeManager manages publishing mode state
type ModeManager interface {
    // GetCurrentMode returns the current mode (cached, fast)
    GetCurrentMode() Mode

    // IsMetricsOnly returns true if in metrics-only mode
    IsMetricsOnly() bool

    // CheckModeTransition checks if mode should change and returns new mode
    // Returns: (newMode, changed, error)
    CheckModeTransition() (Mode, bool, error)

    // OnTargetsChanged is called when targets change (event-driven)
    OnTargetsChanged() error

    // Subscribe subscribes to mode change events
    Subscribe(callback ModeChangeCallback) UnsubscribeFunc

    // GetModeMetrics returns current mode metrics
    GetModeMetrics() ModeMetrics

    // Start starts the mode manager (periodic checking)
    Start(ctx context.Context) error

    // Stop stops the mode manager
    Stop() error
}
```

### 2.2 DefaultModeManager Implementation

```go
package publishing

import (
    "context"
    "sync"
    "sync/atomic"
    "time"

    "log/slog"
)

// DefaultModeManager implements ModeManager
type DefaultModeManager struct {
    discoveryManager TargetDiscoveryManager
    logger           *slog.Logger

    // State (protected by mu)
    currentMode      Mode
    modeChangedAt    time.Time
    transitionCount  int64
    lastTransitionReason string

    // Subscribers (protected by mu)
    subscribers      []ModeChangeCallback
    subscribersMu   sync.RWMutex

    // Caching (for performance)
    cachedMode       Mode
    cachedModeAt     time.Time
    cacheTTL         time.Duration

    // Metrics
    modeCheckDuration time.Duration

    // Control
    mu               sync.RWMutex
    stopCh           chan struct{}
    wg               sync.WaitGroup
}

// NewModeManager creates a new mode manager
func NewModeManager(
    discoveryManager TargetDiscoveryManager,
    logger *slog.Logger,
) ModeManager {
    if logger == nil {
        logger = slog.Default()
    }

    return &DefaultModeManager{
        discoveryManager: discoveryManager,
        logger:           logger,
        currentMode:      ModeNormal, // Default to normal
        modeChangedAt:    time.Now(),
        cacheTTL:         time.Second, // Cache for 1s
        stopCh:           make(chan struct{}),
    }
}

// GetCurrentMode returns cached mode (fast path)
func (m *DefaultModeManager) GetCurrentMode() Mode {
    m.mu.RLock()
    defer m.mu.RUnlock()

    // Return cached mode if still valid
    if time.Since(m.cachedModeAt) < m.cacheTTL {
        return m.cachedMode
    }

    // Cache expired, return current mode
    return m.currentMode
}

// IsMetricsOnly returns true if in metrics-only mode
func (m *DefaultModeManager) IsMetricsOnly() bool {
    return m.GetCurrentMode() == ModeMetricsOnly
}

// CheckModeTransition checks if mode should change
func (m *DefaultModeManager) CheckModeTransition() (Mode, bool, error) {
    start := time.Now()
    defer func() {
        m.modeCheckDuration = time.Since(start)
    }()

    // Count enabled targets
    targets := m.discoveryManager.ListTargets()
    enabledCount := 0
    for _, t := range targets {
        if t.Enabled {
            enabledCount++
        }
    }

    // Determine new mode
    var newMode Mode
    if enabledCount > 0 {
        newMode = ModeNormal
    } else {
        newMode = ModeMetricsOnly
    }

    // Check if mode changed
    m.mu.Lock()
    changed := m.currentMode != newMode
    if changed {
        oldMode := m.currentMode
        m.currentMode = newMode
        m.modeChangedAt = time.Now()
        m.transitionCount++
        m.lastTransitionReason = m.getTransitionReason(enabledCount)
        m.cachedMode = newMode
        m.cachedModeAt = time.Now()

        // Notify subscribers
        m.notifySubscribers(oldMode, newMode, m.lastTransitionReason)

        m.logger.Info("Mode transition detected",
            "from", oldMode.String(),
            "to", newMode.String(),
            "enabled_targets", enabledCount,
            "reason", m.lastTransitionReason,
        )
    } else {
        // Update cache even if no change
        m.cachedMode = newMode
        m.cachedModeAt = time.Now()
    }
    m.mu.Unlock()

    return newMode, changed, nil
}

// OnTargetsChanged is called when targets change
func (m *DefaultModeManager) OnTargetsChanged() error {
    _, _, err := m.CheckModeTransition()
    return err
}

// Subscribe subscribes to mode change events
func (m *DefaultModeManager) Subscribe(callback ModeChangeCallback) UnsubscribeFunc {
    m.subscribersMu.Lock()
    m.subscribers = append(m.subscribers, callback)
    index := len(m.subscribers) - 1
    m.subscribersMu.Unlock()

    return func() {
        m.subscribersMu.Lock()
        defer m.subscribersMu.Unlock()

        // Remove callback
        m.subscribers = append(m.subscribers[:index], m.subscribers[index+1:]...)
    }
}

// GetModeMetrics returns current mode metrics
func (m *DefaultModeManager) GetModeMetrics() ModeMetrics {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return ModeMetrics{
        CurrentMode:          m.currentMode,
        CurrentModeDuration: time.Since(m.modeChangedAt),
        TransitionCount:     atomic.LoadInt64(&m.transitionCount),
        LastTransitionTime:   m.modeChangedAt,
        LastTransitionReason: m.lastTransitionReason,
        ModeCheckDuration:    m.modeCheckDuration,
    }
}

// Start starts periodic mode checking
func (m *DefaultModeManager) Start(ctx context.Context) error {
    m.wg.Add(1)
    go m.periodicCheck(ctx)
    return nil
}

// Stop stops the mode manager
func (m *DefaultModeManager) Stop() error {
    close(m.stopCh)
    m.wg.Wait()
    return nil
}

// periodicCheck periodically checks mode
func (m *DefaultModeManager) periodicCheck(ctx context.Context) {
    defer m.wg.Done()

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-m.stopCh:
            return
        case <-ticker.C:
            m.CheckModeTransition()
        }
    }
}

// notifySubscribers notifies all subscribers
func (m *DefaultModeManager) notifySubscribers(from, to Mode, reason string) {
    m.subscribersMu.RLock()
    subscribers := make([]ModeChangeCallback, len(m.subscribers))
    copy(subscribers, m.subscribers)
    m.subscribersMu.RUnlock()

    for _, callback := range subscribers {
        callback(from, to, reason)
    }
}

// getTransitionReason returns reason for transition
func (m *DefaultModeManager) getTransitionReason(enabledCount int) string {
    if enabledCount == 0 {
        return "no_enabled_targets"
    }
    return "targets_available"
}
```

### 2.3 Integration Points

#### 2.3.1 SubmitAlert Handler Integration

```go
// In handlers.go

func (h *PublishingHandlers) SubmitAlert(w http.ResponseWriter, r *http.Request) {
    // ... existing validation ...

    // Check mode before submitting
    if h.modeManager.IsMetricsOnly() {
        h.logger.Info("Alert submission rejected (metrics-only mode)",
            "alert_fingerprint", req.Alert.Fingerprint,
        )

        // Record metric
        if h.metrics != nil {
            h.metrics.RecordSubmissionRejected("metrics_only")
        }

        // Return informative response
        response := SubmitAlertResponse{
            Success: false,
            Message: "Alert not submitted: system is in metrics-only mode",
            Mode:    "metrics-only",
        }
        h.sendJSON(w, http.StatusOK, response)
        return
    }

    // ... existing submission logic ...
}
```

#### 2.3.2 PublishingQueue Worker Integration

```go
// In queue.go

func (q *PublishingQueue) worker(id int) {
    for {
        select {
        case <-q.ctx.Done():
            return
        case job := <-q.selectJob():
            // Check mode before processing
            if q.modeManager != nil && q.modeManager.IsMetricsOnly() {
                q.logger.Debug("Job skipped (metrics-only mode)",
                    "job_id", job.ID,
                    "target", job.Target.Name,
                )

                // Record metric
                if q.metrics != nil {
                    q.metrics.RecordJobSkipped("metrics_only")
                }

                // Skip processing
                continue
            }

            // ... existing processing logic ...
        }
    }
}
```

#### 2.3.3 PublishingCoordinator Integration

```go
// In coordinator.go

func (c *PublishingCoordinator) PublishToTargets(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    targetNames []string,
) ([]*PublishingResult, error) {
    // Check mode before publishing
    if c.modeManager != nil && c.modeManager.IsMetricsOnly() {
        c.logger.Info("Publishing skipped (metrics-only mode)",
            "fingerprint", enrichedAlert.Alert.Fingerprint,
        )

        // Record metric
        if c.metrics != nil {
            c.metrics.RecordPublicationSkipped("metrics_only")
        }

        // Return empty results
        return []*PublishingResult{}, nil
    }

    // ... existing publishing logic ...
}
```

---

## 3. Data Structures

### 3.1 Mode Type

```go
type Mode int

const (
    ModeNormal Mode = iota
    ModeMetricsOnly
)
```

### 3.2 ModeMetrics

```go
type ModeMetrics struct {
    CurrentMode              Mode
    CurrentModeDuration      time.Duration
    TransitionCount          int64
    LastTransitionTime       time.Time
    LastTransitionReason     string
    ModeCheckDuration        time.Duration
}
```

### 3.3 PublishingModeResponse (Enhanced)

```go
type PublishingModeResponse struct {
    Mode                      string    `json:"mode"`
    TargetsAvailable          bool      `json:"targets_available"`
    EnabledTargets            int       `json:"enabled_targets"`
    MetricsOnlyActive         bool      `json:"metrics_only_active"`
    TransitionCount           int64     `json:"transition_count"`
    CurrentModeDurationSeconds float64   `json:"current_mode_duration_seconds"`
    LastTransitionTime        time.Time `json:"last_transition_time"`
    LastTransitionReason      string    `json:"last_transition_reason"`
}
```

---

## 4. Algorithms

### 4.1 Mode Detection Algorithm

```
Algorithm: DetectMode
Input: targets []*PublishingTarget
Output: Mode

1. enabledCount = 0
2. FOR each target in targets:
3.     IF target.Enabled:
4.         enabledCount++
5. END FOR
6. IF enabledCount > 0:
7.     RETURN ModeNormal
8. ELSE:
9.     RETURN ModeMetricsOnly
```

**Time Complexity**: O(n) where n = number of targets
**Space Complexity**: O(1)

### 4.2 Transition Detection Algorithm

```
Algorithm: CheckTransition
Input: currentMode Mode, newMode Mode
Output: bool (changed)

1. IF currentMode != newMode:
2.     UPDATE currentMode = newMode
3.     RECORD transition timestamp
4.     INCREMENT transition count
5.     NOTIFY subscribers
6.     LOG transition
7.     RETURN true
8. ELSE:
9.     UPDATE cache
10.    RETURN false
```

**Time Complexity**: O(1) + O(s) where s = number of subscribers
**Space Complexity**: O(1)

---

## 5. Performance Considerations

### 5.1 Caching Strategy

- **Cache TTL**: 1 second
- **Cache Invalidation**: On mode change or TTL expiry
- **Cache Hit Rate Target**: >99% (most checks use cached value)

### 5.2 Optimization Techniques

1. **Lazy Evaluation**: Mode checked only when needed
2. **Cached Reads**: Fast path for GetCurrentMode() (<100ns)
3. **Batch Updates**: Periodic checking (5s interval) instead of per-request
4. **Lock-Free Reads**: RLock for reads, Lock only for writes

### 5.3 Performance Targets

| Operation | Baseline | 150% Target |
|-----------|----------|-------------|
| GetCurrentMode() | <1µs | <100ns |
| IsMetricsOnly() | <1µs | <100ns |
| CheckModeTransition() | <10µs | <5µs |
| API Response | <50ms | <10ms |

---

## 6. Error Handling

### 6.1 Error Scenarios

1. **DiscoveryManager unavailable**: Use last known mode (graceful degradation)
2. **Targets list empty**: Mode = MetricsOnly (expected behavior)
3. **Concurrent access**: Protected by mutex (thread-safe)

### 6.2 Error Recovery

- **Discovery failure**: Continue with cached mode
- **Transition failure**: Log error, continue with current mode
- **Subscriber failure**: Log error, continue with other subscribers

---

## 7. Testing Strategy

### 7.1 Unit Tests

- ModeManager state management
- Transition detection logic
- Caching behavior
- Thread-safety (race detector)

### 7.2 Integration Tests

- SubmitAlert handler integration
- Queue worker integration
- Coordinator integration
- ParallelPublisher integration

### 7.3 Benchmark Tests

- GetCurrentMode() performance
- IsMetricsOnly() performance
- CheckModeTransition() performance
- Concurrent access performance

### 7.4 Test Coverage Target

- **Unit Tests**: 95%+ coverage
- **Integration Tests**: All integration points covered
- **Benchmark Tests**: All hot paths benchmarked

---

## 8. Implementation Plan

### Phase 1: ModeManager Core (6h)
1. Define ModeManager interface
2. Implement DefaultModeManager
3. Add state management
4. Add transition detection
5. Add event notifications
6. Unit tests

### Phase 2: Integration (7.5h)
1. Integrate into SubmitAlert handler
2. Integrate into PublishingQueue
3. Integrate into PublishingCoordinator
4. Integrate into ParallelPublisher
5. Integration tests

### Phase 3: Observability (4.5h)
1. Prometheus metrics
2. Structured logging
3. API endpoint enhancement
4. Grafana dashboard

### Phase 4: Testing & Validation (9h)
1. Comprehensive unit tests
2. Integration tests
3. Benchmark tests
4. Race detector tests
5. Load tests

### Phase 5: Documentation (4.5h)
1. API documentation
2. Architecture documentation
3. Troubleshooting guide
4. Examples

---

**Design Date**: 2025-01-13
**Designer**: AI Assistant
**Status**: ✅ Design Complete, Ready for Implementation
