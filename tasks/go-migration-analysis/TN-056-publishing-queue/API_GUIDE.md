# TN-056: Publishing Queue API Guide

**Version**: 1.0
**Date**: 2025-11-12
**Status**: Ready for Use
**Author**: AI Assistant

---

## 📋 Table of Contents

1. [Quick Start](#1-quick-start)
2. [Basic Usage](#2-basic-usage)
3. [Configuration](#3-configuration)
4. [HTTP API Endpoints](#4-http-api-endpoints)
5. [Job Lifecycle](#5-job-lifecycle)
6. [Dead Letter Queue](#6-dead-letter-queue)
7. [Metrics & Monitoring](#7-metrics--monitoring)
8. [Best Practices](#8-best-practices)
9. [Code Examples](#9-code-examples)
10. [Integration Patterns](#10-integration-patterns)

---

## 1. Quick Start

### 1.1 5-Minute Setup

**Step 1**: Initialize the queue in your application:

```go
package main

import (
    "context"
    "database/sql"
    "log/slog"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

func main() {
    // 1. Create dependencies
    logger := slog.Default()
    publisherFactory := publishing.NewPublisherFactory(/* ... */)
    dlqRepo := publishing.NewPostgresDLQRepository(db, logger)
    jobTracking := publishing.NewLRUJobTrackingStore(10000)
    publishingMetrics := metrics.NewPublishingMetrics()

    // 2. Create queue with default configuration
    config := publishing.DefaultPublishingQueueConfig()
    queue := publishing.NewPublishingQueue(
        config,
        publisherFactory,
        dlqRepo,
        jobTracking,
        publishingMetrics,
        logger,
    )

    // 3. Start the queue
    queue.Start()
    defer func() {
        if err := queue.Stop(30 * time.Second); err != nil {
            logger.Error("Failed to stop queue", "error", err)
        }
    }()

    // 4. Submit jobs
    err := queue.Submit(enrichedAlert, target)
    if err != nil {
        logger.Error("Failed to submit job", "error", err)
    }
}
```

**Step 2**: Run database migration:

```bash
migrate -path go-app/migrations -database "postgres://localhost:5432/alert_history" up
```

**Step 3**: Start your application and check metrics:

```bash
curl http://localhost:8080/metrics | grep publishing
```

**Done!** 🎉 Your publishing queue is running.

---

## 2. Basic Usage

### 2.1 Submit a Job

```go
import (
    "github.com/vitaliisemenov/alert-history/internal/core"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// Create an enriched alert
enrichedAlert := &core.EnrichedAlert{
    Alert: &core.Alert{
        Fingerprint: "abc123",
        Labels:      map[string]string{"severity": "critical"},
        Annotations: map[string]string{"summary": "High CPU usage"},
        Status:      core.StatusFiring,
    },
    Classification: &core.Classification{
        Severity:   core.SeverityCritical,
        Confidence: 0.95,
    },
}

// Create a publishing target
target := &core.PublishingTarget{
    Name: "production-pagerduty",
    Type: "pagerduty",
    Config: map[string]interface{}{
        "routing_key": "your-routing-key",
    },
}

// Submit to queue
err := queue.Submit(enrichedAlert, target)
if err != nil {
    // Handle error (e.g., queue full)
    logger.Error("Failed to submit job", "error", err)
}
```

**What happens next?**
1. Job is assigned a **priority** (High/Medium/Low)
2. Job is added to the appropriate **priority queue**
3. A **worker** picks up the job (High priority first)
4. Job is **published** to the external system
5. On failure, **retry** with exponential backoff
6. If max retries exhausted, sent to **Dead Letter Queue**

### 2.2 Check Queue Status

```go
// Get queue size by priority
highSize := queue.GetQueueSizeByPriority(publishing.PriorityHigh)
mediumSize := queue.GetQueueSizeByPriority(publishing.PriorityMedium)
lowSize := queue.GetQueueSizeByPriority(publishing.PriorityLow)

logger.Info("Queue status",
    "high", highSize,
    "medium", mediumSize,
    "low", lowSize,
)
```

### 2.3 Graceful Shutdown

```go
// Stop accepting new jobs and wait for workers to finish
timeout := 30 * time.Second
err := queue.Stop(timeout)
if err != nil {
    logger.Error("Queue did not stop gracefully", "error", err)
}
```

---

## 3. Configuration

### 3.1 Configuration Options

```go
type PublishingQueueConfig struct {
    // Worker pool
    WorkerCount int // Default: 10

    // Queue capacities
    HighPriorityQueueSize   int // Default: 1000
    MediumPriorityQueueSize int // Default: 5000
    LowPriorityQueueSize    int // Default: 10000

    // Retry behavior
    MaxRetries    int           // Default: 3
    RetryInterval time.Duration // Default: 100ms

    // Circuit breaker
    CircuitFailureThreshold int           // Default: 5
    CircuitSuccessThreshold int           // Default: 2
    CircuitTimeout          time.Duration // Default: 30s

    // Job tracking
    JobTrackingCapacity int // Default: 10000
}
```

### 3.2 Environment Variables

```bash
# Worker pool
export PUBLISHING_WORKER_COUNT=10

# Queue capacities
export PUBLISHING_QUEUE_SIZE_HIGH=1000
export PUBLISHING_QUEUE_SIZE_MEDIUM=5000
export PUBLISHING_QUEUE_SIZE_LOW=10000

# Retry behavior
export PUBLISHING_MAX_RETRIES=3
export PUBLISHING_RETRY_INTERVAL=100ms

# Circuit breaker
export PUBLISHING_CIRCUIT_FAILURE_THRESHOLD=5
export PUBLISHING_CIRCUIT_SUCCESS_THRESHOLD=2
export PUBLISHING_CIRCUIT_TIMEOUT=30s

# Job tracking
export PUBLISHING_JOB_TRACKING_CAPACITY=10000
```

### 3.3 Custom Configuration

```go
config := publishing.PublishingQueueConfig{
    WorkerCount:             20, // Double workers for high load
    HighPriorityQueueSize:   2000,
    MediumPriorityQueueSize: 10000,
    LowPriorityQueueSize:    20000,
    MaxRetries:              5, // More retries for flaky networks
    RetryInterval:           50 * time.Millisecond, // Faster retry
    CircuitTimeout:          1 * time.Minute, // Longer timeout
    JobTrackingCapacity:     50000, // Track more jobs
}

queue := publishing.NewPublishingQueue(config, /* ... */)
```

---

## 4. HTTP API Endpoints

### 4.1 Queue Status

**GET /api/v2/publishing/queue/status**

Returns queue size by priority.

```bash
curl http://localhost:8080/api/v2/publishing/queue/status
```

Response:
```json
{
    "high": 5,
    "medium": 120,
    "low": 340,
    "total": 465
}
```

### 4.2 Queue Statistics

**GET /api/v2/publishing/queue/stats**

Returns detailed queue statistics.

```bash
curl http://localhost:8080/api/v2/publishing/queue/stats
```

Response:
```json
{
    "worker_count": 10,
    "high_priority_capacity": 1000,
    "medium_priority_capacity": 5000,
    "low_priority_capacity": 10000,
    "circuit_breakers": {
        "production-pagerduty": "closed",
        "staging-slack": "half-open"
    }
}
```

### 4.3 List Recent Jobs

**GET /api/v2/publishing/jobs**

Query parameters:
- `state` (queued/processing/retrying/succeeded/failed/dlq)
- `priority` (high/medium/low)
- `target` (target name)
- `limit` (default: 100, max: 1000)

```bash
curl "http://localhost:8080/api/v2/publishing/jobs?state=failed&limit=10"
```

Response:
```json
{
    "jobs": [
        {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "fingerprint": "abc123",
            "target_name": "production-pagerduty",
            "priority": "high",
            "state": "failed",
            "submitted_at": 1699800000,
            "started_at": 1699800001,
            "completed_at": 1699800010,
            "retry_count": 3,
            "last_error": "connection timeout",
            "error_type": "transient"
        }
    ],
    "total": 10
}
```

### 4.4 Get Job Status

**GET /api/v2/publishing/jobs/:id**

```bash
curl http://localhost:8080/api/v2/publishing/jobs/123e4567-e89b-12d3-a456-426614174000
```

Response:
```json
{
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "fingerprint": "abc123",
    "target_name": "production-pagerduty",
    "priority": "high",
    "state": "succeeded",
    "submitted_at": 1699800000,
    "started_at": 1699800001,
    "completed_at": 1699800005,
    "retry_count": 0
}
```

### 4.5 Query DLQ

**GET /api/v2/publishing/dlq**

Query parameters:
- `target` (filter by target name)
- `priority` (high/medium/low)
- `error_type` (transient/permanent/unknown)
- `limit` (default: 100, max: 1000)
- `offset` (pagination)

```bash
curl "http://localhost:8080/api/v2/publishing/dlq?target=production-pagerduty&limit=10"
```

Response:
```json
{
    "entries": [
        {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "job_id": "123e4567-e89b-12d3-a456-426614174000",
            "fingerprint": "abc123",
            "target_name": "production-pagerduty",
            "target_type": "pagerduty",
            "error_message": "401 Unauthorized",
            "error_type": "permanent",
            "retry_count": 3,
            "priority": "high",
            "failed_at": "2025-11-12T10:00:00Z",
            "created_at": "2025-11-12T10:00:01Z",
            "replayed": false
        }
    ],
    "total": 10
}
```

### 4.6 Replay DLQ Entry

**POST /api/v2/publishing/dlq/:id/replay**

Resubmit a failed job to the main queue.

```bash
curl -X POST http://localhost:8080/api/v2/publishing/dlq/550e8400-e29b-41d4-a716-446655440000/replay
```

Response:
```json
{
    "success": true,
    "job_id": "123e4567-e89b-12d3-a456-426614174000",
    "message": "Job resubmitted successfully"
}
```

### 4.7 Purge Old DLQ Entries

**DELETE /api/v2/publishing/dlq/purge**

Query parameters:
- `older_than` (duration, e.g., "168h" for 7 days)

```bash
curl -X DELETE "http://localhost:8080/api/v2/publishing/dlq/purge?older_than=168h"
```

Response:
```json
{
    "success": true,
    "deleted_count": 1234,
    "message": "Purged 1234 entries older than 168h"
}
```

---

## 5. Job Lifecycle

### 5.1 State Transitions

```
QUEUED → PROCESSING → SUCCEEDED
                  ↓
                RETRYING → PROCESSING (retry)
                  ↓
                FAILED → DLQ
```

### 5.2 Job States

| State | Description | Next State |
|-------|-------------|------------|
| `QUEUED` | In priority queue, awaiting worker | `PROCESSING` |
| `PROCESSING` | Active publishing attempt | `SUCCEEDED`, `RETRYING`, `FAILED` |
| `RETRYING` | Waiting for backoff before next attempt | `PROCESSING` |
| `SUCCEEDED` | Successfully published | (terminal) |
| `FAILED` | Max retries exhausted | `DLQ` |
| `DLQ` | Stored in Dead Letter Queue | (terminal) |

### 5.3 Retry Behavior

**Exponential Backoff**:
- Attempt 0: 100ms + jitter (0-1000ms)
- Attempt 1: 200ms + jitter
- Attempt 2: 400ms + jitter
- Attempt 3: 800ms + jitter (max retries reached)

**Smart Retry Decision**:
- **Transient errors**: Retry (408, 429, 502, 503, 504, network timeout)
- **Permanent errors**: No retry (400, 401, 403, 404, 405, 422)
- **Unknown errors**: Retry with caution

---

## 6. Dead Letter Queue

### 6.1 When Jobs Go to DLQ

Jobs are sent to the DLQ when:
1. **Max retries exhausted** (3 attempts by default)
2. **Permanent error** (400, 401, 403, 404, etc.)
3. **Circuit breaker permanently open** (rare)

### 6.2 Query DLQ

```go
filters := publishing.DLQFilters{
    TargetName: "production-pagerduty",
    Priority:   publishing.PriorityHigh,
    ErrorType:  publishing.QueueErrorTypePermanent,
    Limit:      100,
    Offset:     0,
}

entries, err := dlqRepo.List(ctx, filters)
if err != nil {
    logger.Error("Failed to query DLQ", "error", err)
}

for _, entry := range entries {
    logger.Info("DLQ entry",
        "id", entry.ID,
        "fingerprint", entry.Fingerprint,
        "target", entry.TargetName,
        "error", entry.ErrorMessage,
    )
}
```

### 6.3 Replay Failed Jobs

```go
// Replay single entry
err := dlqRepo.Replay(ctx, entryID)
if err != nil {
    logger.Error("Failed to replay entry", "error", err)
}
```

### 6.4 Purge Old Entries

```go
// Purge entries older than 7 days
olderThan := 7 * 24 * time.Hour
deletedCount, err := dlqRepo.Purge(ctx, olderThan)
if err != nil {
    logger.Error("Failed to purge DLQ", "error", err)
}

logger.Info("Purged DLQ entries", "count", deletedCount)
```

### 6.5 DLQ Statistics

```go
stats, err := dlqRepo.GetStats(ctx)
if err != nil {
    logger.Error("Failed to get DLQ stats", "error", err)
}

logger.Info("DLQ stats",
    "total_entries", stats.TotalEntries,
    "by_target", stats.ByTarget,
    "by_priority", stats.ByPriority,
    "by_error_type", stats.ByErrorType,
)
```

---

## 7. Metrics & Monitoring

### 7.1 Prometheus Metrics

**Queue Metrics**:
```
# Queue size by priority
publishing_queue_size{priority="high"} 5
publishing_queue_size{priority="medium"} 120
publishing_queue_size{priority="low"} 340

# Queue capacity
publishing_queue_capacity{priority="high"} 1000

# Job submissions
publishing_queue_submissions_total{priority="high"} 1234

# Submission duration
publishing_queue_submission_duration_seconds_bucket{le="0.001"} 1000
```

**Retry Metrics**:
```
# Retry attempts
publishing_retry_attempts_total{target="production-pagerduty",error_type="transient"} 567

# Backoff duration
publishing_retry_backoff_seconds_sum 123.45

# Retry successes
publishing_retry_successes_total{target="production-pagerduty"} 450
```

**DLQ Metrics**:
```
# DLQ writes
publishing_dlq_writes_total{target="production-pagerduty",error_type="permanent"} 12

# DLQ size
publishing_dlq_size 234

# DLQ reads
publishing_dlq_reads_total 56
```

**Publishing Metrics**:
```
# Successful publishing
publishing_successes_total{target="production-pagerduty",priority="high"} 5678

# Publishing failures
publishing_failures_total{target="production-pagerduty",priority="high"} 12
```

### 7.2 PromQL Queries

**Success Rate**:
```promql
rate(publishing_successes_total[5m]) /
(rate(publishing_successes_total[5m]) + rate(publishing_failures_total[5m]))
```

**Average Queue Latency (p95)**:
```promql
histogram_quantile(0.95, rate(publishing_queue_submission_duration_seconds_bucket[5m]))
```

**DLQ Entry Rate**:
```promql
rate(publishing_dlq_writes_total[5m])
```

**Circuit Breaker Open Targets**:
```promql
count(publishing_circuit_breaker_state{state="open"})
```

### 7.3 Grafana Dashboard

Create a dashboard with these panels:

1. **Publishing Success Rate** (Gauge)
   - Target: 99.9%
   - Alert: < 99%

2. **Queue Size by Priority** (Graph)
   - High (red), Medium (yellow), Low (green)

3. **Publishing Latency** (Heatmap)
   - p50, p95, p99

4. **Retry Rate by Error Type** (Stacked graph)
   - Transient, Permanent, Unknown

5. **DLQ Entry Rate** (Counter)
   - Alert: > 0.1% of total

6. **Circuit Breaker States** (Table)
   - Target, State, Last Failure

7. **Top Failing Targets** (Bar chart)
   - By error count

---

## 8. Best Practices

### 8.1 Configuration

✅ **DO**:
- Set `WorkerCount` to 2x CPU cores for I/O-bound workloads
- Set queue capacities based on traffic patterns
- Configure retry intervals based on external system SLAs
- Enable job tracking for debugging

❌ **DON'T**:
- Set `WorkerCount` > 50 (diminishing returns)
- Set queue capacity < 100 (too small for bursts)
- Set `MaxRetries` > 10 (excessive)
- Disable DLQ (data loss risk)

### 8.2 Error Handling

✅ **DO**:
- Log all DLQ entries with full context
- Set up alerts for DLQ entry rate > 0.1%
- Review DLQ entries weekly
- Replay failed jobs after fixing root cause

❌ **DON'T**:
- Ignore DLQ entries
- Automatically replay without investigation
- Retry permanent errors (400, 401, 404)

### 8.3 Monitoring

✅ **DO**:
- Monitor queue size (alert > 80% capacity)
- Monitor publishing success rate (alert < 99%)
- Monitor DLQ entry rate (alert > 0.1%)
- Monitor circuit breaker states (alert if open > 5 min)

❌ **DON'T**:
- Ignore metrics spikes
- Set alert thresholds too low (noise)
- Forget to export metrics to Prometheus

### 8.4 Performance

✅ **DO**:
- Use priority queues for critical alerts
- Batch DLQ purges (weekly)
- Monitor job tracking cache hit rate
- Profile under load

❌ **DON'T**:
- Submit jobs synchronously (blocks caller)
- Query DLQ in hot path (slow)
- Disable circuit breaker (cascading failures)

### 8.5 Security

✅ **DO**:
- Store target credentials in K8s Secrets
- Rotate credentials regularly
- Use TLS for external systems
- Sanitize errors before logging

❌ **DON'T**:
- Log credentials (even masked)
- Store credentials in DLQ
- Expose DLQ HTTP API publicly

---

## 9. Code Examples

### 9.1 Submit High-Priority Job

```go
// Critical alert → High priority
enrichedAlert := &core.EnrichedAlert{
    Alert: &core.Alert{
        Labels: map[string]string{"severity": "critical"},
        Status: core.StatusFiring,
    },
}

target := &core.PublishingTarget{Name: "production-pagerduty"}

err := queue.Submit(enrichedAlert, target)
// Job automatically gets PriorityHigh
```

### 9.2 Custom Retry Configuration

```go
config := publishing.QueueRetryConfig{
    MaxRetries:    5,              // More retries for flaky network
    BaseInterval:  50 * time.Millisecond, // Faster retry
    MaxBackoff:    10 * time.Second, // Lower max backoff
    JitterEnabled: true,
    JitterMax:     500 * time.Millisecond,
}

// Use in NewPublishingQueue() initialization
```

### 9.3 Monitor Job Status

```go
// Submit job
err := queue.Submit(enrichedAlert, target)

// Get job ID from tracking store (if needed)
filters := publishing.JobFilters{
    TargetName: target.Name,
    Limit:      1,
}
jobs := jobTracking.List(filters)

if len(jobs) > 0 {
    job := jobs[0]
    logger.Info("Job status",
        "id", job.ID,
        "state", job.State,
        "retry_count", job.RetryCount,
    )
}
```

### 9.4 Handle DLQ Entries

```go
// Query DLQ for permanent errors
filters := publishing.DLQFilters{
    ErrorType: publishing.QueueErrorTypePermanent,
    Limit:     100,
}

entries, err := dlqRepo.List(ctx, filters)

for _, entry := range entries {
    // Investigate root cause
    logger.Error("Permanent failure",
        "fingerprint", entry.Fingerprint,
        "target", entry.TargetName,
        "error", entry.ErrorMessage,
    )

    // After fixing (e.g., correcting credentials):
    // dlqRepo.Replay(ctx, entry.ID)
}
```

---

## 10. Integration Patterns

### 10.1 AlertProcessor Integration

```go
func (p *AlertProcessor) Process(enrichedAlert *core.EnrichedAlert) error {
    // Get publishing targets
    targets := p.targetManager.GetTargets()

    for _, target := range targets {
        // Submit to queue (non-blocking)
        err := p.queue.Submit(enrichedAlert, target)
        if err != nil {
            p.logger.Error("Failed to submit job", "error", err)
            // Continue to next target
        }
    }

    return nil
}
```

### 10.2 HTTP Handler Integration

```go
func (h *PublishingHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "high":   h.queue.GetQueueSizeByPriority(publishing.PriorityHigh),
        "medium": h.queue.GetQueueSizeByPriority(publishing.PriorityMedium),
        "low":    h.queue.GetQueueSizeByPriority(publishing.PriorityLow),
    }

    json.NewEncoder(w).Encode(status)
}
```

### 10.3 Graceful Shutdown

```go
func main() {
    // ... initialize queue ...

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    <-sigChan
    logger.Info("Shutting down...")

    // Stop queue with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := queue.Stop(30 * time.Second); err != nil {
        logger.Error("Queue did not stop gracefully", "error", err)
    }

    logger.Info("Shutdown complete")
}
```

---

## 📚 Additional Resources

- **requirements.md**: Detailed functional and non-functional requirements
- **design.md**: Architecture, state machines, implementation details
- **tasks.md**: Implementation checklist and progress tracking
- **TROUBLESHOOTING.md**: Common issues and debugging guide

---

**Document Status**: ✅ COMPLETE
**Last Updated**: 2025-11-12
**Next**: TROUBLESHOOTING.md
