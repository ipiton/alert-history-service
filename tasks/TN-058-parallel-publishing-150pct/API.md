# TN-058: Parallel Publishing API Documentation

**Status**: ✅ Production-Ready
**Version**: 1.0.0
**Last Updated**: 2025-11-13

## Table of Contents

1. [Overview](#overview)
2. [Core Interfaces](#core-interfaces)
3. [Data Structures](#data-structures)
4. [Usage Examples](#usage-examples)
5. [Configuration](#configuration)
6. [Error Handling](#error-handling)
7. [Performance Considerations](#performance-considerations)
8. [Integration Guide](#integration-guide)

---

## Overview

The Parallel Publishing system enables concurrent publishing of alert data to multiple targets (Rootly, PagerDuty, Slack, Webhook, AlertManager) with:

- **Concurrent Execution**: Fan-out/fan-in pattern for maximum throughput
- **Health-Aware Routing**: Skip unhealthy targets based on health checks
- **Partial Success Handling**: Track which targets succeeded/failed
- **Comprehensive Metrics**: Prometheus metrics for all operations
- **Structured Logging**: Detailed logging with `slog`
- **Configurable Behavior**: Flexible options for timeouts, concurrency, health checks

**Performance Targets** (150% Quality):
- ⚡ Latency: < 5ms per target (excluding network I/O)
- 🚀 Throughput: 100+ targets/second
- 📊 Success Rate: 99.9%+ (for healthy targets)
- 🔧 Memory: < 50MB for 1000 targets

---

## Core Interfaces

### ParallelPublisher

Main interface for parallel publishing operations.

```go
type ParallelPublisher interface {
    // PublishToMultiple publishes an alert to multiple specific targets in parallel.
    //
    // This method:
    //   - Validates input (alert != nil, targets not empty)
    //   - Publishes to all enabled targets concurrently
    //   - Collects results from all targets
    //   - Returns aggregate result with per-target details
    //   - Respects context timeout/cancellation
    //
    // Health checking behavior is controlled by ParallelPublishOptions.CheckHealth:
    //   - If CheckHealth = true: Uses health monitor to filter targets
    //   - If CheckHealth = false: Publishes to all enabled targets
    //
    // Parameters:
    //   - ctx: Context for timeout/cancellation
    //   - alert: Alert to publish (must not be nil)
    //   - targets: List of targets to publish to (must not be empty)
    //
    // Returns:
    //   - *ParallelPublishResult: Aggregate result with counts and per-target details
    //   - error: Error if input is invalid or operation fails
    //
    // Example:
    //   result, err := publisher.PublishToMultiple(ctx, alert, targets)
    //   if err != nil {
    //       log.Error("Publish failed", "error", err)
    //       return err
    //   }
    //   log.Info("Published", "success_count", result.SuccessCount)
    PublishToMultiple(ctx context.Context, alert *core.EnrichedAlert, targets []*core.PublishingTarget) (*ParallelPublishResult, error)

    // PublishToAll discovers all available targets and publishes to them in parallel.
    //
    // This method:
    //   - Calls TargetDiscoveryManager.ListTargets() to get all targets
    //   - Filters out disabled targets (Enabled = false)
    //   - Calls PublishToMultiple with filtered targets
    //
    // Parameters:
    //   - ctx: Context for timeout/cancellation
    //   - alert: Alert to publish (must not be nil)
    //
    // Returns:
    //   - *ParallelPublishResult: Aggregate result
    //   - error: Error if no enabled targets or operation fails
    //
    // Example:
    //   result, err := publisher.PublishToAll(ctx, alert)
    PublishToAll(ctx context.Context, alert *core.EnrichedAlert) (*ParallelPublishResult, error)

    // PublishToHealthy discovers all targets and publishes only to healthy ones.
    //
    // This method:
    //   - Calls TargetDiscoveryManager.ListTargets() to get all targets
    //   - Filters out disabled targets (Enabled = false)
    //   - Filters out unhealthy targets (uses health monitor)
    //   - Calls PublishToMultiple with filtered targets
    //
    // Health filtering strategy is controlled by ParallelPublishOptions.HealthStrategy:
    //   - SkipUnhealthy: Skip only unhealthy targets (default)
    //   - SkipUnhealthyAndDegraded: Skip unhealthy and degraded targets
    //   - PublishToAll: No health filtering (same as PublishToAll)
    //
    // Parameters:
    //   - ctx: Context for timeout/cancellation
    //   - alert: Alert to publish (must not be nil)
    //
    // Returns:
    //   - *ParallelPublishResult: Aggregate result
    //   - error: Error if no healthy targets or operation fails
    //
    // Example:
    //   result, err := publisher.PublishToHealthy(ctx, alert)
    PublishToHealthy(ctx context.Context, alert *core.EnrichedAlert) (*ParallelPublishResult, error)
}
```

---

## Data Structures

### ParallelPublishResult

Aggregate result of parallel publishing to multiple targets.

```go
type ParallelPublishResult struct {
    // Aggregate Counts
    TotalTargets int `json:"total_targets"` // Total targets attempted
    SuccessCount int `json:"success_count"` // Successful publishes
    FailureCount int `json:"failure_count"` // Failed publishes
    SkippedCount int `json:"skipped_count"` // Skipped targets (unhealthy, disabled)

    // Per-Target Results
    Results []TargetPublishResult `json:"results"` // Detailed results

    // Timing
    Duration time.Duration `json:"duration"` // Total execution time (parallel)

    // Status
    IsPartialSuccess bool `json:"is_partial_success"` // Some succeeded, some failed
}

// Helper Methods
func (r *ParallelPublishResult) Success() bool        // At least one success
func (r *ParallelPublishResult) AllSucceeded() bool   // All succeeded
func (r *ParallelPublishResult) AllFailed() bool      // All failed
func (r *ParallelPublishResult) SuccessRate() float64 // Success rate (0-100%)
```

**Example Usage**:

```go
result, err := publisher.PublishToMultiple(ctx, alert, targets)
if err != nil {
    log.Error("Publish failed", "error", err)
    return err
}

log.Info("Publish completed",
    "total_targets", result.TotalTargets,
    "success_count", result.SuccessCount,
    "failure_count", result.FailureCount,
    "skipped_count", result.SkippedCount,
    "duration_ms", result.Duration.Milliseconds(),
    "is_partial_success", result.IsPartialSuccess,
)

// Check if at least one target succeeded
if result.Success() {
    log.Info("At least one target succeeded")
}

// Check if all targets succeeded
if result.AllSucceeded() {
    log.Info("All targets succeeded")
}

// Calculate success rate
successRate := result.SuccessRate()
log.Info("Success rate", "rate", successRate)
if successRate < 80.0 {
    log.Warn("Low success rate", "rate", successRate)
}
```

### TargetPublishResult

Result of publishing to a single target.

```go
type TargetPublishResult struct {
    // Target Info
    TargetName string `json:"target_name"` // e.g., "rootly-prod"
    TargetType string `json:"target_type"` // rootly, pagerduty, slack, webhook

    // Result
    Success  bool          `json:"success"`  // Publish succeeded
    Error    error         `json:"error,omitempty"` // Error details (if failed)
    Duration time.Duration `json:"duration"` // Time taken

    // HTTP Details (optional)
    StatusCode *int `json:"status_code,omitempty"` // HTTP status code

    // Skip Details (optional)
    Skipped    bool    `json:"skipped"`              // Target was skipped
    SkipReason *string `json:"skip_reason,omitempty"` // Reason for skipping
}
```

**Example Usage**:

```go
for _, targetResult := range result.Results {
    if targetResult.Success {
        log.Info("Target succeeded",
            "target_name", targetResult.TargetName,
            "duration_ms", targetResult.Duration.Milliseconds(),
            "status_code", *targetResult.StatusCode,
        )
    } else if targetResult.Skipped {
        log.Warn("Target skipped",
            "target_name", targetResult.TargetName,
            "skip_reason", *targetResult.SkipReason,
        )
    } else {
        log.Error("Target failed",
            "target_name", targetResult.TargetName,
            "error", targetResult.Error,
            "status_code", targetResult.StatusCode,
        )
    }
}
```

### ParallelPublishOptions

Configuration options for parallel publishing.

```go
type ParallelPublishOptions struct {
    // Timeout is the maximum time to wait for all targets to complete.
    // Default: 30 seconds
    Timeout time.Duration

    // CheckHealth determines whether to check target health before publishing.
    // Default: true
    CheckHealth bool

    // HealthStrategy determines which targets to skip based on health status.
    // Default: SkipUnhealthy
    HealthStrategy HealthCheckStrategy

    // MaxConcurrent is the maximum number of concurrent publishes.
    // Default: 50 (0 = unlimited)
    MaxConcurrent int

    // EnableCircuitBreaker determines whether to use circuit breaker pattern.
    // Default: true (not yet implemented)
    EnableCircuitBreaker bool
}

type HealthCheckStrategy int

const (
    SkipUnhealthy             HealthCheckStrategy = 0 // Skip only unhealthy targets
    PublishToAll              HealthCheckStrategy = 1 // No health filtering
    SkipUnhealthyAndDegraded  HealthCheckStrategy = 2 // Skip unhealthy & degraded
)

func DefaultParallelPublishOptions() *ParallelPublishOptions
func (o *ParallelPublishOptions) Validate() error
```

**Example Usage**:

```go
// Use default options
opts := DefaultParallelPublishOptions()

// Custom options
opts := &ParallelPublishOptions{
    Timeout:                  60 * time.Second, // 1 minute timeout
    CheckHealth:              true,             // Enable health checks
    HealthStrategy:           SkipUnhealthyAndDegraded,
    MaxConcurrent:            100,              // Allow 100 concurrent publishes
    EnableCircuitBreaker:     true,
}

// Validate options
if err := opts.Validate(); err != nil {
    log.Error("Invalid options", "error", err)
    return err
}
```

---

## Usage Examples

### Example 1: Publish to Specific Targets

```go
package main

import (
    "context"
    "time"
    "log/slog"

    "github.com/vitaliisemenov/alert-history/internal/core"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

func example1() error {
    // Create publisher
    publisher, err := publishing.NewDefaultParallelPublisher(
        factory,     // PublisherFactory
        healthMon,   // HealthMonitor
        discoveryMgr,// TargetDiscoveryManager
        metrics,     // ParallelPublishMetrics
        logger,      // *slog.Logger
        nil,         // nil = use default options
    )
    if err != nil {
        return err
    }

    // Create alert
    alert := &core.EnrichedAlert{
        Fingerprint: "alert-123",
        Status:      "firing",
        Labels: map[string]string{
            "severity": "critical",
            "service":  "api",
        },
        // ... more fields
    }

    // Define targets
    targets := []*core.PublishingTarget{
        {Name: "rootly-prod", Type: "rootly", Enabled: true},
        {Name: "pagerduty-oncall", Type: "pagerduty", Enabled: true},
        {Name: "slack-alerts", Type: "slack", Enabled: true},
    }

    // Publish with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    result, err := publisher.PublishToMultiple(ctx, alert, targets)
    if err != nil {
        slog.Error("Publish failed", "error", err)
        return err
    }

    // Check result
    slog.Info("Publish completed",
        "total_targets", result.TotalTargets,
        "success_count", result.SuccessCount,
        "failure_count", result.FailureCount,
        "duration_ms", result.Duration.Milliseconds(),
    )

    return nil
}
```

### Example 2: Publish to All Targets

```go
func example2() error {
    // Create publisher (same as Example 1)
    publisher, err := publishing.NewDefaultParallelPublisher(...)
    if err != nil {
        return err
    }

    // Create alert
    alert := &core.EnrichedAlert{...}

    // Publish to all discovered targets
    ctx := context.Background()
    result, err := publisher.PublishToAll(ctx, alert)
    if err != nil {
        slog.Error("Publish failed", "error", err)
        return err
    }

    // Log per-target results
    for _, targetResult := range result.Results {
        if targetResult.Success {
            slog.Info("Target succeeded", "target", targetResult.TargetName)
        } else {
            slog.Error("Target failed",
                "target", targetResult.TargetName,
                "error", targetResult.Error,
            )
        }
    }

    return nil
}
```

### Example 3: Publish to Healthy Targets Only

```go
func example3() error {
    // Create publisher with custom options
    opts := &publishing.ParallelPublishOptions{
        Timeout:        60 * time.Second,
        CheckHealth:    true,
        HealthStrategy: publishing.SkipUnhealthyAndDegraded, // Skip degraded too
        MaxConcurrent:  100,
    }

    publisher, err := publishing.NewDefaultParallelPublisher(
        factory, healthMon, discoveryMgr, metrics, logger, opts,
    )
    if err != nil {
        return err
    }

    // Create alert
    alert := &core.EnrichedAlert{...}

    // Publish to healthy targets only
    ctx := context.Background()
    result, err := publisher.PublishToHealthy(ctx, alert)
    if err != nil {
        if errors.Is(err, publishing.ErrNoHealthyTargets) {
            slog.Warn("No healthy targets available")
            return nil
        }
        return err
    }

    // Check success rate
    successRate := result.SuccessRate()
    if successRate < 80.0 {
        slog.Warn("Low success rate", "rate", successRate)
    }

    return nil
}
```

---

## Configuration

### Default Configuration

```go
DefaultParallelPublishOptions() = &ParallelPublishOptions{
    Timeout:                  30 * time.Second,
    CheckHealth:              true,
    HealthStrategy:           SkipUnhealthy,
    MaxConcurrent:            50,
    EnableCircuitBreaker:     true,
}
```

### Production Configuration

```go
// High-throughput configuration
opts := &ParallelPublishOptions{
    Timeout:                  60 * time.Second,
    CheckHealth:              true,
    HealthStrategy:           SkipUnhealthyAndDegraded,
    MaxConcurrent:            200,    // Higher concurrency
    EnableCircuitBreaker:     true,
}

// Conservative configuration
opts := &ParallelPublishOptions{
    Timeout:                  15 * time.Second,
    CheckHealth:              true,
    HealthStrategy:           SkipUnhealthyAndDegraded,
    MaxConcurrent:            20,     // Lower concurrency
    EnableCircuitBreaker:     true,
}
```

---

## Error Handling

### Error Types

```go
var (
    // ErrInvalidInput is returned when input validation fails
    ErrInvalidInput = errors.New("invalid input")

    // ErrAllTargetsFailed is returned when all targets failed
    ErrAllTargetsFailed = errors.New("all targets failed")

    // ErrContextTimeout is returned when context deadline exceeded
    ErrContextTimeout = errors.New("context deadline exceeded")

    // ErrContextCancelled is returned when context was cancelled
    ErrContextCancelled = errors.New("context cancelled")

    // ErrNoHealthyTargets is returned when no healthy targets available
    ErrNoHealthyTargets = errors.New("no healthy targets available")

    // ErrNoEnabledTargets is returned when no enabled targets found
    ErrNoEnabledTargets = errors.New("no enabled targets found")
)
```

### Error Handling Example

```go
result, err := publisher.PublishToMultiple(ctx, alert, targets)
if err != nil {
    switch {
    case errors.Is(err, publishing.ErrInvalidInput):
        log.Error("Invalid input", "error", err)
        return err
    case errors.Is(err, publishing.ErrAllTargetsFailed):
        log.Error("All targets failed")
        // Maybe retry or alert operations team
    case errors.Is(err, publishing.ErrContextTimeout):
        log.Warn("Timeout exceeded")
        // Maybe increase timeout or investigate slow targets
    case errors.Is(err, publishing.ErrNoHealthyTargets):
        log.Warn("No healthy targets")
        // Maybe alert operations team
    default:
        log.Error("Unexpected error", "error", err)
    }
    return err
}

// Check for partial success
if result.IsPartialSuccess {
    log.Warn("Partial success",
        "success_count", result.SuccessCount,
        "failure_count", result.FailureCount,
    )
    // Investigate failed targets
    for _, tr := range result.Results {
        if !tr.Success && !tr.Skipped {
            log.Error("Target failed",
                "target", tr.TargetName,
                "error", tr.Error,
            )
        }
    }
}
```

---

## Performance Considerations

### Concurrency Control

The `MaxConcurrent` option controls how many goroutines can publish simultaneously:

- **0** (unlimited): Best throughput, highest memory usage
- **50** (default): Balanced throughput and memory
- **100-200**: High throughput for production systems
- **10-20**: Conservative for low-memory environments

**Recommendation**: Start with default (50), increase to 100-200 for production.

### Timeout Configuration

The `Timeout` option determines the maximum wait time:

- **15s**: Conservative, may timeout on slow networks
- **30s** (default): Balanced for most use cases
- **60s**: Recommended for production with many targets
- **120s+**: For batch operations or very slow networks

**Recommendation**: Use 60s for production.

### Memory Usage

Approximate memory per target:
- **1KB**: Target metadata
- **5KB**: Alert data (shared across targets)
- **2KB**: Result structure

Total memory = `(1KB + 2KB) * num_targets + 5KB`

Example:
- 10 targets: ~35KB
- 100 targets: ~305KB
- 1000 targets: ~3.5MB

**Recommendation**: For 1000+ targets, consider batching.

---

## Integration Guide

### Step 1: Create Dependencies

```go
// Create publisher factory
factory := publishing.NewPublisherFactory(logger, formatter)

// Create health monitor
healthMon := publishing.NewHealthMonitor(...)

// Create discovery manager
discoveryMgr := publishing.NewTargetDiscoveryManager(...)

// Create metrics
metrics := publishing.NewParallelPublishMetrics()
```

### Step 2: Create Parallel Publisher

```go
opts := publishing.DefaultParallelPublishOptions()
publisher, err := publishing.NewDefaultParallelPublisher(
    factory,
    healthMon,
    discoveryMgr,
    metrics,
    logger,
    opts,
)
if err != nil {
    return err
}
```

### Step 3: Publish Alerts

```go
result, err := publisher.PublishToHealthy(ctx, alert)
if err != nil {
    log.Error("Publish failed", "error", err)
    return err
}

log.Info("Published", "success_count", result.SuccessCount)
```

---

## Next Steps

- **[Performance Benchmarks](BENCHMARKS.md)**: Detailed performance analysis
- **[Troubleshooting Guide](TROUBLESHOOTING.md)**: Common issues and solutions
- **[Examples](examples/)**: Full working examples
- **[Design Document](design.md)**: Architecture and design decisions

---

**Documentation Version**: 1.0.0
**Last Updated**: 2025-11-13
**Author**: TN-058 Implementation Team
