# TN-058: Parallel Publishing to Multiple Targets - Design Document

**Version**: 1.0
**Date**: 2025-11-13
**Status**: Design Phase
**Target Quality**: 150% (Enterprise-Grade Excellence)

---

## 📋 Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Design](#component-design)
3. [Data Structures](#data-structures)
4. [Interface Design](#interface-design)
5. [Implementation Strategy](#implementation-strategy)
6. [Performance Optimization](#performance-optimization)
7. [Error Handling](#error-handling)
8. [Observability](#observability)
9. [Testing Strategy](#testing-strategy)

---

## 🏗️ Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Parallel Publishing System                   │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ParallelPublisher Interface                   │
│  • PublishToMultiple(alert, targets) → Result                   │
│  • PublishToAll(alert) → Result                                 │
│  • PublishToHealthy(alert) → Result                             │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│              DefaultParallelPublisher (Core Logic)               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 1. Health Check (optional)                              │   │
│  │    • Query HealthMonitor (cached, <10ms)                │   │
│  │    • Skip unhealthy targets                             │   │
│  │    • Check circuit breakers                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 2. Fan-Out (Parallel Execution)                         │   │
│  │    • Spawn goroutine per target                         │   │
│  │    • Create publisher via factory                       │   │
│  │    • Publish alert to target                            │   │
│  │    • Send result to channel                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 3. Fan-In (Result Collection)                           │   │
│  │    • Collect results from all goroutines                │   │
│  │    • Aggregate counts (success/failure/skipped)         │   │
│  │    • Calculate duration                                 │   │
│  │    • Determine partial success                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 4. Result Processing                                    │   │
│  │    • Update metrics (Prometheus)                        │   │
│  │    • Log results (structured logging)                   │   │
│  │    • Return aggregate result                            │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    ▼                           ▼
┌──────────────────────────────┐  ┌──────────────────────────────┐
│   External Dependencies      │  │   Internal Components        │
│  • HealthMonitor (TN-049)    │  │  • PublisherFactory          │
│  • TargetDiscoveryManager    │  │  • PublishingMetrics         │
│    (TN-047)                  │  │  • Logger (slog)             │
│  • PublishingQueue (TN-056)  │  │  • Context (timeout/cancel)  │
└──────────────────────────────┘  └──────────────────────────────┘
```

### Design Principles

1. **Concurrency**: Fan-out/fan-in pattern for parallel execution
2. **Resilience**: Partial success handling, graceful degradation
3. **Performance**: <500ms p99 latency, goroutine pooling
4. **Observability**: Comprehensive metrics, structured logging
5. **Testability**: Mockable interfaces, dependency injection
6. **Maintainability**: Clean code, clear separation of concerns

---

## 🧩 Component Design

### 1. ParallelPublisher Interface

**Purpose**: Define contract for parallel publishing operations

**Responsibilities**:
- Publish to multiple targets in parallel
- Publish to all enabled targets
- Publish to healthy targets only
- Return aggregate results

**Design**:
```go
// ParallelPublisher publishes alerts to multiple targets in parallel.
//
// Thread-Safety: All methods are safe for concurrent use.
// Performance: <500ms p99 for 5 targets, <1s p99 for 10 targets.
// Error Handling: Returns nil if ≥1 target succeeds, error if all fail.
type ParallelPublisher interface {
    // PublishToMultiple publishes alert to specified targets in parallel.
    PublishToMultiple(ctx context.Context, alert *core.EnrichedAlert, targets []*core.PublishingTarget) (*ParallelPublishResult, error)

    // PublishToAll publishes alert to all enabled targets.
    PublishToAll(ctx context.Context, alert *core.EnrichedAlert) (*ParallelPublishResult, error)

    // PublishToHealthy publishes alert to healthy targets only.
    PublishToHealthy(ctx context.Context, alert *core.EnrichedAlert) (*ParallelPublishResult, error)
}
```

---

### 2. DefaultParallelPublisher Implementation

**Purpose**: Core implementation of parallel publishing logic

**Responsibilities**:
- Health checks (optional, via HealthMonitor)
- Fan-out (spawn goroutines per target)
- Fan-in (collect results from all goroutines)
- Result aggregation (counts, duration, partial success)
- Metrics collection (Prometheus)
- Structured logging (slog)

**Design**:
```go
// DefaultParallelPublisher implements ParallelPublisher interface.
type DefaultParallelPublisher struct {
    factory        *PublisherFactory      // Creates publishers by type
    healthMonitor  HealthMonitor          // Health status checks (optional)
    discoveryMgr   *TargetDiscoveryManager // Target enumeration
    metrics        *ParallelPublishMetrics // Prometheus metrics
    logger         *slog.Logger           // Structured logging
    options        ParallelPublishOptions // Configuration options
}

// NewDefaultParallelPublisher creates a new parallel publisher.
func NewDefaultParallelPublisher(
    factory *PublisherFactory,
    healthMonitor HealthMonitor,
    discoveryMgr *TargetDiscoveryManager,
    metrics *ParallelPublishMetrics,
    logger *slog.Logger,
    options ParallelPublishOptions,
) *DefaultParallelPublisher {
    // Validate inputs
    // Set defaults
    // Return instance
}
```

**Key Methods**:

#### PublishToMultiple
```go
func (p *DefaultParallelPublisher) PublishToMultiple(
    ctx context.Context,
    alert *core.EnrichedAlert,
    targets []*core.PublishingTarget,
) (*ParallelPublishResult, error) {
    // 1. Validate inputs
    if alert == nil || len(targets) == 0 {
        return nil, ErrInvalidInput
    }

    // 2. Apply timeout (default 30s)
    ctx, cancel := context.WithTimeout(ctx, p.options.Timeout)
    defer cancel()

    // 3. Health checks (optional)
    if p.options.CheckHealth {
        targets = p.filterHealthyTargets(ctx, targets)
    }

    // 4. Fan-out: Spawn goroutines
    resultChan := make(chan TargetPublishResult, len(targets))
    for _, target := range targets {
        go p.publishToTarget(ctx, alert, target, resultChan)
    }

    // 5. Fan-in: Collect results
    results := make([]TargetPublishResult, 0, len(targets))
    for i := 0; i < len(targets); i++ {
        select {
        case result := <-resultChan:
            results = append(results, result)
        case <-ctx.Done():
            // Context timeout/cancellation
            break
        }
    }

    // 6. Aggregate results
    aggregateResult := p.aggregateResults(results, time.Since(startTime))

    // 7. Update metrics
    p.updateMetrics(aggregateResult)

    // 8. Log results
    p.logResults(aggregateResult)

    // 9. Return result
    if aggregateResult.SuccessCount == 0 {
        return aggregateResult, ErrAllTargetsFailed
    }
    return aggregateResult, nil
}
```

#### publishToTarget (goroutine worker)
```go
func (p *DefaultParallelPublisher) publishToTarget(
    ctx context.Context,
    alert *core.EnrichedAlert,
    target *core.PublishingTarget,
    resultChan chan<- TargetPublishResult,
) {
    startTime := time.Now()

    // 1. Create result structure
    result := TargetPublishResult{
        TargetName: target.Name,
        TargetType: target.Type,
    }

    // 2. Check circuit breaker (optional)
    if p.options.RespectCircuitBreakers {
        if !p.canPublishToTarget(target) {
            result.Skipped = true
            skipReason := "circuit_open"
            result.SkipReason = &skipReason
            resultChan <- result
            return
        }
    }

    // 3. Create publisher
    publisher, err := p.factory.CreatePublisherForTarget(target)
    if err != nil {
        result.Success = false
        result.Error = fmt.Errorf("failed to create publisher: %w", err)
        result.Duration = time.Since(startTime)
        resultChan <- result
        return
    }

    // 4. Publish alert
    err = publisher.Publish(ctx, alert, target)
    result.Duration = time.Since(startTime)

    // 5. Handle result
    if err != nil {
        result.Success = false
        result.Error = err
        // Extract status code if HTTP error
        if httpErr, ok := err.(*HTTPError); ok {
            result.StatusCode = &httpErr.StatusCode
        }
    } else {
        result.Success = true
    }

    // 6. Send result to channel
    resultChan <- result
}
```

#### aggregateResults
```go
func (p *DefaultParallelPublisher) aggregateResults(
    results []TargetPublishResult,
    duration time.Duration,
) *ParallelPublishResult {
    aggregate := &ParallelPublishResult{
        TotalTargets: len(results),
        Results:      results,
        Duration:     duration,
    }

    // Count success/failure/skipped
    for _, result := range results {
        if result.Skipped {
            aggregate.SkippedCount++
        } else if result.Success {
            aggregate.SuccessCount++
        } else {
            aggregate.FailureCount++
        }
    }

    // Determine partial success
    aggregate.IsPartialSuccess = aggregate.SuccessCount > 0 && aggregate.FailureCount > 0

    return aggregate
}
```

---

### 3. Health-Aware Routing

**Purpose**: Skip unhealthy targets before publishing

**Design**:
```go
// filterHealthyTargets filters targets based on health status.
func (p *DefaultParallelPublisher) filterHealthyTargets(
    ctx context.Context,
    targets []*core.PublishingTarget,
) []*core.PublishingTarget {
    if p.healthMonitor == nil {
        return targets // No health monitoring, return all
    }

    healthy := make([]*core.PublishingTarget, 0, len(targets))

    for _, target := range targets {
        // Get health status from cache (O(1), <10ms)
        health, err := p.healthMonitor.GetHealthByName(ctx, target.Name)
        if err != nil {
            // Health status unknown, include target (fail open)
            p.logger.Warn("Failed to get health status, including target",
                "target", target.Name,
                "error", err,
            )
            healthy = append(healthy, target)
            continue
        }

        // Apply health strategy
        switch p.options.HealthStrategy {
        case SkipUnhealthy:
            if !health.IsUnhealthy() {
                healthy = append(healthy, target)
            } else {
                p.logger.Debug("Skipping unhealthy target",
                    "target", target.Name,
                    "status", health.Status,
                )
            }
        case SkipUnhealthyAndDegraded:
            if health.IsHealthy() {
                healthy = append(healthy, target)
            } else {
                p.logger.Debug("Skipping unhealthy/degraded target",
                    "target", target.Name,
                    "status", health.Status,
                )
            }
        case PublishToAll:
            healthy = append(healthy, target)
        }
    }

    return healthy
}
```

---

## 📊 Data Structures

### ParallelPublishResult

```go
// ParallelPublishResult represents aggregate result of parallel publishing.
type ParallelPublishResult struct {
    // Aggregate Counts
    TotalTargets int `json:"total_targets"` // Total targets attempted
    SuccessCount int `json:"success_count"` // Successful publishes
    FailureCount int `json:"failure_count"` // Failed publishes
    SkippedCount int `json:"skipped_count"` // Skipped targets

    // Per-Target Results
    Results []TargetPublishResult `json:"results"` // Detailed per-target results

    // Timing
    Duration time.Duration `json:"duration"` // Total execution time (parallel)

    // Status
    IsPartialSuccess bool `json:"is_partial_success"` // Some succeeded, some failed
}

// Success returns true if at least one target succeeded.
func (r *ParallelPublishResult) Success() bool {
    return r.SuccessCount > 0
}

// AllSucceeded returns true if all targets succeeded.
func (r *ParallelPublishResult) AllSucceeded() bool {
    return r.SuccessCount == r.TotalTargets && r.FailureCount == 0
}

// AllFailed returns true if all targets failed.
func (r *ParallelPublishResult) AllFailed() bool {
    return r.FailureCount == r.TotalTargets && r.SuccessCount == 0
}

// SuccessRate returns success rate (0.0-1.0).
func (r *ParallelPublishResult) SuccessRate() float64 {
    if r.TotalTargets == 0 {
        return 0.0
    }
    return float64(r.SuccessCount) / float64(r.TotalTargets)
}
```

### TargetPublishResult

```go
// TargetPublishResult represents result for single target.
type TargetPublishResult struct {
    // Target Info
    TargetName string `json:"target_name"` // Target name (e.g., "rootly-prod")
    TargetType string `json:"target_type"` // Target type (rootly/pagerduty/slack/webhook)

    // Result
    Success  bool          `json:"success"`  // Did publish succeed?
    Error    error         `json:"error,omitempty"` // Error details (nil if success)
    Duration time.Duration `json:"duration"` // Publish duration

    // HTTP Details (optional)
    StatusCode *int `json:"status_code,omitempty"` // HTTP status code

    // Skip Details (optional)
    Skipped    bool    `json:"skipped"`              // Was target skipped?
    SkipReason *string `json:"skip_reason,omitempty"` // Skip reason
}
```

### ParallelPublishOptions

```go
// ParallelPublishOptions configures parallel publishing behavior.
type ParallelPublishOptions struct {
    // Timeout
    Timeout time.Duration // Max time for all publishes (default 30s)

    // Health Checks
    CheckHealth    bool                 // Check health before publishing (default true)
    HealthStrategy HealthCheckStrategy  // Health check strategy (default SkipUnhealthy)

    // Worker Pool
    MaxConcurrent int  // Max concurrent goroutines (default 10)
    UseWorkerPool bool // Use worker pool (default false)

    // Circuit Breakers
    RespectCircuitBreakers bool // Skip targets with open circuit breakers (default true)
}

// DefaultParallelPublishOptions returns default options.
func DefaultParallelPublishOptions() ParallelPublishOptions {
    return ParallelPublishOptions{
        Timeout:                30 * time.Second,
        CheckHealth:            true,
        HealthStrategy:         SkipUnhealthy,
        MaxConcurrent:          10,
        UseWorkerPool:          false,
        RespectCircuitBreakers: true,
    }
}
```

### HealthCheckStrategy

```go
// HealthCheckStrategy defines health check behavior.
type HealthCheckStrategy int

const (
    // SkipUnhealthy skips unhealthy targets (default).
    // Publishes to: healthy, degraded, unknown
    SkipUnhealthy HealthCheckStrategy = iota

    // PublishToAll publishes to all targets (ignore health).
    // Publishes to: healthy, unhealthy, degraded, unknown
    PublishToAll

    // SkipUnhealthyAndDegraded skips unhealthy and degraded targets.
    // Publishes to: healthy, unknown
    SkipUnhealthyAndDegraded
)

func (s HealthCheckStrategy) String() string {
    switch s {
    case SkipUnhealthy:
        return "skip_unhealthy"
    case PublishToAll:
        return "publish_to_all"
    case SkipUnhealthyAndDegraded:
        return "skip_unhealthy_and_degraded"
    default:
        return "unknown"
    }
}
```

---

## 🔌 Interface Design

### Dependencies

```go
// HealthMonitor interface (from TN-049)
type HealthMonitor interface {
    GetHealthByName(ctx context.Context, targetName string) (*TargetHealthStatus, error)
}

// TargetDiscoveryManager interface (from TN-047)
type TargetDiscoveryManager interface {
    GetTargets(ctx context.Context) ([]*core.PublishingTarget, error)
}

// PublisherFactory (from publisher.go)
type PublisherFactory struct {
    CreatePublisherForTarget(target *core.PublishingTarget) (AlertPublisher, error)
}

// AlertPublisher interface (from publisher.go)
type AlertPublisher interface {
    Publish(ctx context.Context, alert *core.EnrichedAlert, target *core.PublishingTarget) error
    Name() string
}
```

### Metrics Interface

```go
// ParallelPublishMetrics collects Prometheus metrics.
type ParallelPublishMetrics struct {
    // Duration histogram
    duration *prometheus.HistogramVec // labels: result (success/partial_success/failure)

    // Counters
    total          *prometheus.CounterVec // labels: result
    success        prometheus.Counter
    partialSuccess prometheus.Counter
    failure        prometheus.Counter

    // Per-target counters
    targetsTotal   *prometheus.CounterVec // labels: target_type
    targetsSuccess *prometheus.CounterVec // labels: target_name
    targetsFailure *prometheus.CounterVec // labels: target_name, error_type
    targetsSkipped *prometheus.CounterVec // labels: target_name, skip_reason

    // Goroutine gauge
    goroutines prometheus.Gauge
}

// NewParallelPublishMetrics creates metrics instance.
func NewParallelPublishMetrics(registry *prometheus.Registry) *ParallelPublishMetrics {
    // Create metrics
    // Register with registry
    // Return instance
}

// RecordPublish records parallel publish result.
func (m *ParallelPublishMetrics) RecordPublish(result *ParallelPublishResult) {
    // Update duration histogram
    // Update counters
    // Update per-target metrics
}
```

---

## 🚀 Implementation Strategy

### Phase 1: Core Implementation (4-6 hours)

1. **Create parallel_publisher.go**
   - Define `ParallelPublisher` interface
   - Implement `DefaultParallelPublisher`
   - Implement `PublishToMultiple` method
   - Implement `publishToTarget` goroutine worker
   - Implement `aggregateResults` helper

2. **Create parallel_publish_result.go**
   - Define `ParallelPublishResult` structure
   - Define `TargetPublishResult` structure
   - Implement helper methods (Success, AllSucceeded, etc.)

3. **Create parallel_publish_options.go**
   - Define `ParallelPublishOptions` structure
   - Define `HealthCheckStrategy` enum
   - Implement `DefaultParallelPublishOptions`

4. **Implement health-aware routing**
   - Implement `filterHealthyTargets` method
   - Integrate with `HealthMonitor`
   - Handle health check errors gracefully

### Phase 2: Advanced Features (2-3 hours)

5. **Implement PublishToAll**
   - Retrieve targets from `TargetDiscoveryManager`
   - Filter enabled targets
   - Call `PublishToMultiple`

6. **Implement PublishToHealthy**
   - Retrieve targets from `TargetDiscoveryManager`
   - Filter enabled targets
   - Filter healthy targets
   - Call `PublishToMultiple`

7. **Circuit breaker integration**
   - Check circuit breaker state before publishing
   - Skip targets with open circuit breakers
   - Log skip reasons

### Phase 3: Observability (2-3 hours)

8. **Create parallel_publish_metrics.go**
   - Define `ParallelPublishMetrics` structure
   - Implement Prometheus metrics (10+)
   - Implement `RecordPublish` method

9. **Structured logging**
   - Log publish start (debug level)
   - Log publish result (info level)
   - Log partial success (warn level)
   - Log total failure (error level)
   - Log per-target results (debug level)

### Phase 4: Testing (3-4 hours)

10. **Unit tests**
    - Test `PublishToMultiple` (happy path, error paths)
    - Test `PublishToAll` (0 targets, 1 target, 10 targets)
    - Test `PublishToHealthy` (all healthy, all unhealthy, mixed)
    - Test health-aware routing (skip unhealthy, skip degraded)
    - Test partial success handling
    - Test context timeout/cancellation
    - Test circuit breaker integration

11. **Integration tests**
    - Mock `HealthMonitor`
    - Mock `TargetDiscoveryManager`
    - Mock `PublisherFactory`
    - Test end-to-end flow

12. **Benchmarks**
    - Parallel vs sequential (5 targets)
    - Scaling (2, 5, 10 targets)
    - Goroutine spawn overhead

13. **Race detection**
    - Run all tests with `-race` flag
    - Fix any race conditions

14. **Goroutine leak detection**
    - Add goleak tests
    - Fix any goroutine leaks

---

## ⚡ Performance Optimization

### 1. Goroutine Pooling (Optional)

**Problem**: Spawning goroutines has overhead (~2-5µs per goroutine)

**Solution**: Reuse goroutines via worker pool

**Design**:
```go
// WorkerPool manages goroutine pool for parallel publishing.
type WorkerPool struct {
    workers   int
    jobChan   chan publishJob
    resultChan chan TargetPublishResult
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

type publishJob struct {
    ctx    context.Context
    alert  *core.EnrichedAlert
    target *core.PublishingTarget
}

// NewWorkerPool creates a worker pool.
func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    pool := &WorkerPool{
        workers:    workers,
        jobChan:    make(chan publishJob, workers*2),
        resultChan: make(chan TargetPublishResult, workers*2),
        ctx:        ctx,
        cancel:     cancel,
    }

    // Start workers
    for i := 0; i < workers; i++ {
        pool.wg.Add(1)
        go pool.worker(i)
    }

    return pool
}

// worker processes jobs from jobChan.
func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()

    for {
        select {
        case job := <-p.jobChan:
            result := p.processJob(job)
            p.resultChan <- result
        case <-p.ctx.Done():
            return
        }
    }
}

// Submit submits a job to the pool.
func (p *WorkerPool) Submit(job publishJob) {
    p.jobChan <- job
}

// Shutdown stops the worker pool.
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
    p.cancel()

    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("worker pool shutdown timeout")
    }
}
```

**Trade-offs**:
- ✅ **Pros**: Reduces goroutine spawn overhead, better resource control
- ❌ **Cons**: More complex code, potential bottleneck if pool size too small
- **Decision**: Implement as optional feature (default: direct goroutines)

### 2. Buffered Channels

**Problem**: Unbuffered channels can block goroutines

**Solution**: Use buffered channels (size = number of targets)

**Implementation**:
```go
// Use buffered channel to avoid blocking
resultChan := make(chan TargetPublishResult, len(targets))
```

**Benefits**:
- ✅ No blocking on channel send
- ✅ Goroutines can exit immediately after sending result
- ✅ Faster result collection

### 3. Context Timeout Optimization

**Problem**: Default 30s timeout may be too long for fast targets

**Solution**: Adaptive timeout based on target count

**Implementation**:
```go
// Adaptive timeout: 5s + 500ms per target
timeout := 5*time.Second + time.Duration(len(targets))*500*time.Millisecond
if timeout > 30*time.Second {
    timeout = 30 * time.Second
}
ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()
```

**Benefits**:
- ✅ Faster timeout for small target counts
- ✅ Reasonable timeout for large target counts
- ✅ Better user experience

---

## 🛡️ Error Handling

### Error Classification

```go
// Error types
var (
    ErrInvalidInput       = errors.New("invalid input: alert or targets nil/empty")
    ErrAllTargetsFailed   = errors.New("all targets failed")
    ErrContextTimeout     = errors.New("context timeout exceeded")
    ErrContextCancelled   = errors.New("context cancelled")
    ErrNoHealthyTargets   = errors.New("no healthy targets available")
    ErrNoEnabledTargets   = errors.New("no enabled targets available")
)
```

### Error Handling Strategy

1. **Input Validation**
   - Return `ErrInvalidInput` if alert or targets nil/empty
   - Log error at error level

2. **Partial Success**
   - Return `nil` error if ≥1 target succeeds
   - Set `IsPartialSuccess = true`
   - Log at warn level

3. **Total Failure**
   - Return `ErrAllTargetsFailed` if all targets fail
   - Include per-target errors in result
   - Log at error level

4. **Context Errors**
   - Return `ErrContextTimeout` on timeout
   - Return `ErrContextCancelled` on cancellation
   - Clean up goroutines gracefully

5. **Health Check Errors**
   - Fail open (include target if health check fails)
   - Log warning
   - Continue publishing

---

## 📊 Observability

### Prometheus Metrics

```go
// Metrics
const (
    // Duration histogram
    metricParallelPublishDuration = "alert_history_publishing_parallel_duration_seconds"

    // Counters
    metricParallelPublishTotal          = "alert_history_publishing_parallel_total"
    metricParallelPublishSuccess        = "alert_history_publishing_parallel_success_total"
    metricParallelPublishPartialSuccess = "alert_history_publishing_parallel_partial_success_total"
    metricParallelPublishFailure        = "alert_history_publishing_parallel_failure_total"

    // Per-target counters
    metricParallelPublishTargetsTotal   = "alert_history_publishing_parallel_targets_total"
    metricParallelPublishTargetsSuccess = "alert_history_publishing_parallel_targets_success_total"
    metricParallelPublishTargetsFailure = "alert_history_publishing_parallel_targets_failure_total"
    metricParallelPublishTargetsSkipped = "alert_history_publishing_parallel_targets_skipped_total"

    // Goroutine gauge
    metricParallelPublishGoroutines = "alert_history_publishing_parallel_goroutines"
)

// Histogram buckets (optimized for <500ms p99)
var durationBuckets = []float64{
    0.05,  // 50ms
    0.1,   // 100ms
    0.2,   // 200ms
    0.5,   // 500ms
    1.0,   // 1s
    2.0,   // 2s
    5.0,   // 5s
    10.0,  // 10s
}
```

### Structured Logging

```go
// Log levels
// - Debug: Per-target publish start/end, health checks
// - Info: Parallel publish result (success, partial success)
// - Warn: Partial success, skipped targets
// - Error: Total failure, aggregate errors

// Example logs
logger.Debug("Starting parallel publish",
    "alert_fingerprint", alert.Alert.Fingerprint,
    "total_targets", len(targets),
)

logger.Info("Parallel publish completed",
    "alert_fingerprint", alert.Alert.Fingerprint,
    "total_targets", result.TotalTargets,
    "success_count", result.SuccessCount,
    "failure_count", result.FailureCount,
    "skipped_count", result.SkippedCount,
    "duration_ms", result.Duration.Milliseconds(),
    "is_partial_success", result.IsPartialSuccess,
)

logger.Warn("Partial success: some targets failed",
    "alert_fingerprint", alert.Alert.Fingerprint,
    "success_count", result.SuccessCount,
    "failure_count", result.FailureCount,
)

logger.Error("All targets failed",
    "alert_fingerprint", alert.Alert.Fingerprint,
    "total_targets", result.TotalTargets,
    "errors", aggregateErrors(result.Results),
)
```

---

## 🧪 Testing Strategy

### Unit Tests (90%+ Coverage)

1. **PublishToMultiple Tests**
   - Happy path (all targets succeed)
   - Partial success (some succeed, some fail)
   - Total failure (all targets fail)
   - Empty targets (error)
   - Nil alert (error)
   - Context timeout
   - Context cancellation

2. **PublishToAll Tests**
   - 0 enabled targets
   - 1 enabled target
   - 10 enabled targets
   - Discovery manager error

3. **PublishToHealthy Tests**
   - All healthy targets
   - All unhealthy targets
   - Mixed (healthy + unhealthy)
   - Health monitor error (fail open)

4. **Health-Aware Routing Tests**
   - SkipUnhealthy strategy
   - PublishToAll strategy
   - SkipUnhealthyAndDegraded strategy

5. **Circuit Breaker Tests**
   - Open circuit breaker (skip target)
   - Closed circuit breaker (publish)

### Integration Tests

1. **End-to-End Flow**
   - Mock HealthMonitor
   - Mock TargetDiscoveryManager
   - Mock PublisherFactory
   - Test full flow (discovery → health check → publish → result)

2. **Error Scenarios**
   - Publisher creation failure
   - Publish failure (HTTP 500)
   - Health check failure
   - Discovery failure

### Benchmarks

1. **Parallel vs Sequential**
   - Benchmark 5 targets (parallel vs sequential)
   - Measure latency (p50, p95, p99)
   - Verify 10x improvement

2. **Scaling**
   - Benchmark 2, 5, 10 targets
   - Verify linear scaling (2x targets ≈ 1.1x latency)

3. **Goroutine Overhead**
   - Benchmark goroutine spawn overhead
   - Verify <10ms per target

### Race Detection

```bash
go test -race ./internal/infrastructure/publishing/...
```

### Goroutine Leak Detection

```go
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

---

## 📝 Conclusion

This design document defines the **architecture**, **components**, **data structures**, **interfaces**, and **implementation strategy** for **TN-058: Parallel Publishing to Multiple Targets**.

**Key Design Decisions**:
1. **Fan-out/fan-in pattern** for parallel execution
2. **Buffered channels** for result collection (no blocking)
3. **Health-aware routing** (cached health status, <10ms)
4. **Partial success handling** (return nil if ≥1 target succeeds)
5. **Comprehensive metrics** (10+ Prometheus metrics)
6. **Structured logging** (slog, debug/info/warn/error)
7. **Optional worker pool** (default: direct goroutines)

**Performance Targets**:
- <500ms p99 latency for 5 targets (10x faster than sequential)
- <1s p99 latency for 10 targets
- Linear scaling (2x targets ≈ 1.1x latency)
- <10ms overhead per target

**Quality Targets**:
- 90%+ unit test coverage
- 0 race conditions (validated with -race)
- 0 goroutine leaks (validated with goleak)
- 2000+ lines documentation

**Next Step**: Create `tasks.md` (implementation checklist)

---

**Design Complete** ✅
**Ready to Proceed to Implementation** 🚀
