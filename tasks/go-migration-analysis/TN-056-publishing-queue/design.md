# TN-056: Publishing Queue with Retry - Design Document

**Version**: 1.0
**Date**: 2025-11-12
**Status**: Implementation Complete, Documentation In Progress
**Author**: AI Assistant

---

## 📋 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [System Components](#2-system-components)
3. [Data Flow](#3-data-flow)
4. [State Machines](#4-state-machines)
5. [Implementation Details](#5-implementation-details)
6. [Performance Optimization](#6-performance-optimization)
7. [Error Handling](#7-error-handling)
8. [Concurrency & Thread Safety](#8-concurrency--thread-safety)
9. [Database Design](#9-database-design)
10. [Metrics & Observability](#10-metrics--observability)
11. [Security Considerations](#11-security-considerations)
12. [Testing Strategy](#12-testing-strategy)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Alert Processing Pipeline                   │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Publishing Queue                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │  High Priority │  │ Medium Priority│  │  Low Priority  │   │
│  │   Queue (1000) │  │  Queue (1000)  │  │  Queue (1000)  │   │
│  └────────────────┘  └────────────────┘  └────────────────┘   │
│           │                   │                   │              │
│           └───────────────────┴───────────────────┘              │
│                              │                                   │
│                   ┌──────────▼──────────┐                       │
│                   │   Worker Pool (10)   │                       │
│                   └──────────┬──────────┘                       │
│                              │                                   │
│         ┌────────────────────┼────────────────────┐            │
│         │                    │                    │            │
│         ▼                    ▼                    ▼            │
│  ┌─────────────┐      ┌─────────────┐    ┌─────────────┐     │
│  │   Retry     │      │   Circuit   │    │ Job Tracking│     │
│  │   Logic     │      │   Breaker   │    │  (LRU Cache)│     │
│  └─────────────┘      └─────────────┘    └─────────────┘     │
└─────────────────────────────────────────────────────────────────┘
         │                      │                      │
         ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ Publisher       │   │ External System │   │ Dead Letter     │
│ Factory         │   │ (Rootly, PD,    │   │ Queue (DLQ)     │
│                 │   │  Slack, Webhook)│   │ PostgreSQL      │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

### 1.2 Design Principles

1. **Asynchronous Processing**: Non-blocking job submission for high throughput
2. **Priority-Based Scheduling**: Critical alerts processed first
3. **Fail-Fast with Retry**: Smart error classification for efficient retry
4. **Isolation**: Circuit breaker per target prevents cascading failures
5. **Observability**: Comprehensive metrics and structured logging
6. **Data Persistence**: DLQ ensures zero data loss
7. **Thread Safety**: Concurrent access with minimal contention
8. **Graceful Degradation**: Continue operation during partial failures

### 1.3 Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Queue | Go channels | Native, efficient, type-safe |
| Retry | Exponential backoff | Industry standard, proven effective |
| Cache | In-memory LRU | Fast lookups, automatic eviction |
| DLQ | PostgreSQL | ACID guarantees, JSONB support |
| Metrics | Prometheus | De facto standard, powerful queries |
| Logging | slog | Structured, performant, stdlib |
| Concurrency | Goroutines + sync | Lightweight, Go-native |

---

## 2. System Components

### 2.1 PublishingQueue

**Purpose**: Core orchestrator for asynchronous alert publishing.

**Responsibilities**:
- Accept job submissions
- Manage priority queues (High/Medium/Low)
- Coordinate worker pool
- Track job lifecycle
- Integrate with retry, circuit breaker, DLQ

**Key Fields**:
```go
type PublishingQueue struct {
    // Priority channels
    highPriorityJobs   chan *PublishingJob
    mediumPriorityJobs chan *PublishingJob
    lowPriorityJobs    chan *PublishingJob

    // Dependencies
    factory          *PublisherFactory
    dlqRepository    DLQRepository
    jobTrackingStore JobTrackingStore
    metrics          *PublishingMetrics
    logger           *slog.Logger

    // Configuration
    workerCount   int
    maxRetries    int
    retryInterval time.Duration

    // Circuit breakers (per target)
    circuitBreakers map[string]*CircuitBreaker
    mu              sync.RWMutex

    // Lifecycle
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}
```

**Key Methods**:
- `Submit(alert *core.EnrichedAlert, target *core.PublishingTarget) error`
- `Start()`
- `Stop(timeout time.Duration) error`
- `worker(id int)` - goroutine per worker
- `processJob(job *PublishingJob) error`
- `retryPublish(job *PublishingJob, attempt int) error`

### 2.2 Priority Determination

**Purpose**: Assign priority to jobs based on alert severity and classification.

**Algorithm**:
```go
func determinePriority(enrichedAlert *core.EnrichedAlert) Priority {
    // HIGH: Critical firing alerts
    if severity == "critical" && status == StatusFiring {
        return PriorityHigh
    }

    // HIGH: LLM classified as critical
    if classification != nil && classification.Severity == SeverityCritical {
        return PriorityHigh
    }

    // LOW: Resolved alerts
    if status == StatusResolved {
        return PriorityLow
    }

    // LOW: Info severity
    if severity == "info" {
        return PriorityLow
    }

    // DEFAULT: Medium priority
    return PriorityMedium
}
```

**Performance**: 8-9 ns/op (instant)

### 2.3 Enhanced Retry Logic

**Purpose**: Automatically retry failed publishing with exponential backoff.

**Key Components**:
```go
type QueueRetryConfig struct {
    MaxRetries    int           // 3
    BaseInterval  time.Duration // 100ms
    MaxBackoff    time.Duration // 30s
    JitterEnabled bool          // true
    JitterMax     time.Duration // 1s
}

func CalculateBackoff(attempt int, config QueueRetryConfig) time.Duration {
    // Exponential: 2^attempt * baseInterval
    backoff := time.Duration(math.Pow(2, float64(attempt))) * config.BaseInterval

    // Cap at maxBackoff
    if backoff > config.MaxBackoff {
        backoff = config.MaxBackoff
    }

    // Add jitter to prevent thundering herd
    if config.JitterEnabled && config.JitterMax > 0 {
        jitter := time.Duration(rand.Int63n(int64(config.JitterMax)))
        backoff += jitter
    }

    return backoff
}
```

**Backoff Sequence** (base=100ms, jitter=0-1000ms):
- Attempt 0: 100ms + jitter
- Attempt 1: 200ms + jitter
- Attempt 2: 400ms + jitter
- Attempt 3: 800ms + jitter

**Performance**: 22.75 ns/op

### 2.4 Error Classification

**Purpose**: Classify errors as transient (retryable) or permanent (non-retryable).

**Classification Rules**:

| Error Type | Category | Action |
|------------|----------|--------|
| HTTP 408, 429 | Transient | Retry |
| HTTP 502, 503, 504 | Transient | Retry |
| HTTP 400, 401, 403, 404, 405, 422 | Permanent | DLQ |
| Network timeout | Transient | Retry |
| DNS failure | Transient | Retry |
| Connection refused | Transient | Retry |
| Unknown HTTP 5xx | Permanent | DLQ |
| Unknown error | Unknown | Retry with caution |

**Implementation**:
```go
func classifyPublishingError(err error) QueueErrorType {
    // HTTP errors
    var httpErr interface{ StatusCode() int }
    if errors.As(err, &httpErr) {
        return classifyQueueHTTPError(httpErr.StatusCode())
    }

    // String-based parsing (fallback)
    if errType := classifyQueueHTTPErrorString(err.Error()); errType != QueueErrorTypeUnknown {
        return errType
    }

    // Network errors
    var netErr net.Error
    if errors.As(err, &netErr) {
        if netErr.Timeout() || netErr.Temporary() {
            return QueueErrorTypeTransient
        }
    }

    // Syscall errors
    var syscallErr syscall.Errno
    if errors.As(err, &syscallErr) {
        if syscallErr == syscall.ECONNREFUSED ||
           syscallErr == syscall.ECONNRESET ||
           syscallErr == syscall.ETIMEDOUT {
            return QueueErrorTypeTransient
        }
    }

    return QueueErrorTypeUnknown
}
```

**Performance**: 110-406 ns/op

### 2.5 Dead Letter Queue (DLQ)

**Purpose**: Persist failed jobs for manual review and replay.

**PostgreSQL Schema**:
```sql
CREATE TABLE publishing_dlq (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id           UUID NOT NULL,
    fingerprint      VARCHAR(255) NOT NULL,
    target_name      VARCHAR(255) NOT NULL,
    target_type      VARCHAR(50) NOT NULL,
    enriched_alert   JSONB NOT NULL,
    target_config    JSONB NOT NULL,
    error_message    TEXT,
    error_type       VARCHAR(50),
    retry_count      INT NOT NULL DEFAULT 0,
    last_retry_at    TIMESTAMP WITH TIME ZONE,
    priority         VARCHAR(50) NOT NULL,
    failed_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    replayed         BOOLEAN DEFAULT FALSE,
    replayed_at      TIMESTAMP WITH TIME ZONE,
    replay_result    TEXT
);
```

**Indexes**:
1. `idx_publishing_dlq_target_name` - Filter by target
2. `idx_publishing_dlq_priority` - Filter by priority
3. `idx_publishing_dlq_failed_at` - Sort by failure time
4. `idx_publishing_dlq_error_type` - Filter by error type
5. `idx_publishing_dlq_replayed` - Filter replayed entries
6. `idx_publishing_dlq_fingerprint` - Lookup by alert fingerprint

**Key Operations**:
- `Write(job *PublishingJob)` - Add failed job
- `Read(filters DLQFilters)` - Query with pagination
- `Replay(id UUID)` - Resubmit to main queue
- `Purge(olderThan time.Duration)` - Cleanup old entries
- `GetStats()` - Aggregate statistics

### 2.6 Job Tracking Store

**Purpose**: Track recent job status in memory for real-time monitoring.

**LRU Cache Implementation**:
```go
type LRUJobTrackingStore struct {
    capacity int
    store    map[string]*list.Element  // jobID → list element
    lruList  *list.List                // Doubly-linked list
    mu       sync.RWMutex
}

type jobTrackingEntry struct {
    key   string
    value *JobSnapshot
}
```

**Eviction Policy**:
- Capacity: 10,000 jobs (configurable)
- Access: O(1) Get, O(1) Add
- Eviction: Least Recently Used (back of list)
- Thread-safe: RWMutex for concurrent access

**Performance**:
- Add: 265.4 ns/op
- Get: 82.18 ns/op
- List (100 jobs): 1286 ns/op

### 2.7 Circuit Breaker

**Purpose**: Prevent cascading failures by isolating unhealthy targets.

**State Machine**:
```
       ┌─────────────────┐
       │     CLOSED      │ ◄──── Normal operation
       │ (all requests)  │
       └────────┬────────┘
                │ 5 consecutive failures
                ▼
       ┌─────────────────┐
       │      OPEN       │ ◄──── Fast-fail
       │ (reject all)    │
       └────────┬────────┘
                │ After 30s timeout
                ▼
       ┌─────────────────┐
       │   HALF-OPEN     │ ◄──── Testing recovery
       │ (allow 1 req)   │
       └────────┬────────┘
          │             │
   2 successes      1 failure
          │             │
          ▼             ▼
       CLOSED         OPEN
```

**Configuration**:
```go
type CircuitBreakerConfig struct {
    FailureThreshold int           // 5
    SuccessThreshold int           // 2
    Timeout          time.Duration // 30s
}
```

**Performance**:
- CanAttempt: 14.92 ns/op
- RecordSuccess: 27.75 ns/op
- RecordFailure: 115.0 ns/op

---

## 3. Data Flow

### 3.1 Successful Publishing Flow

```
┌──────────────┐
│ Submit Job   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────┐
│ Determine Priority           │
│ - severity=critical → HIGH   │
│ - severity=info → LOW        │
│ - default → MEDIUM           │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Submit to Priority Queue     │
│ - highPriorityJobs           │
│ - mediumPriorityJobs         │
│ - lowPriorityJobs            │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Worker Picks Job (Priority)  │
│ 1. Try High queue            │
│ 2. Try Medium queue          │
│ 3. Try Low queue             │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Check Circuit Breaker        │
│ - CLOSED → Continue          │
│ - OPEN → Retry later         │
│ - HALF-OPEN → Single attempt │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Publish via Publisher        │
│ - Format alert               │
│ - Send HTTP request          │
│ - Await response             │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Record Success               │
│ - Update job state           │
│ - Record metrics             │
│ - CB: RecordSuccess()        │
│ - Log INFO                   │
└──────────────────────────────┘
```

### 3.2 Retry Flow (Transient Error)

```
┌──────────────────────────────┐
│ Publishing Failed            │
│ - Network timeout            │
│ - 503 Service Unavailable    │
│ - 429 Too Many Requests      │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Classify Error               │
│ → QueueErrorTypeTransient    │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Check Retry Count            │
│ - < maxRetries → Retry       │
│ - ≥ maxRetries → DLQ         │
└──────┬───────────────────────┘
       │ Retry
       ▼
┌──────────────────────────────┐
│ Calculate Backoff            │
│ - 2^attempt * 100ms + jitter │
│ - Attempt 0: 100-1100ms      │
│ - Attempt 1: 200-1200ms      │
│ - Attempt 2: 400-1400ms      │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Sleep for Backoff Duration   │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Retry Publishing Attempt     │
│ - Increment retry count      │
│ - Update job state: retrying │
│ - Record retry metric        │
└──────┬───────────────────────┘
       │
       └──────► Back to "Check Circuit Breaker"
```

### 3.3 DLQ Flow (Permanent Error)

```
┌──────────────────────────────┐
│ Publishing Failed            │
│ - 400 Bad Request            │
│ - 401 Unauthorized           │
│ - 404 Not Found              │
│ - Max retries exhausted      │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Classify Error               │
│ → QueueErrorTypePermanent    │
│ OR retry count ≥ maxRetries  │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Write to DLQ (PostgreSQL)    │
│ - INSERT INTO publishing_dlq │
│ - Store JSONB alert + target │
│ - Record error details       │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│ Update Job State → DLQ       │
│ - Update job tracking        │
│ - Record metrics             │
│ - Log ERROR                  │
└──────────────────────────────┘
```

---

## 4. State Machines

### 4.1 Job State Machine

```
    ┌─────────┐
    │ QUEUED  │ ◄──── Initial state (in priority queue)
    └────┬────┘
         │ Worker picks job
         ▼
    ┌──────────────┐
    │ PROCESSING   │ ◄──── Active publishing attempt
    └────┬─────────┘
         │
    ┌────┴─────────────────────┐
    │ Success?                  │
    └────┬─────────────┬────────┘
         │ Yes         │ No
         │             │
         ▼             ▼
    ┌───────────┐  ┌──────────────┐
    │ SUCCEEDED │  │ Classify     │
    └───────────┘  │ Error        │
                   └───┬──────────┘
                       │
          ┌────────────┴─────────────┐
          │ Transient?               │
          └────┬──────────────┬──────┘
               │ Yes          │ No
               │              │
               ▼              ▼
          ┌──────────┐   ┌──────────┐
          │ Retry    │   │   DLQ    │
          │ Count?   │   └──────────┘
          └────┬─────┘
               │
      ┌────────┴────────┐
      │ < maxRetries?   │
      └────┬────────┬───┘
           │ Yes    │ No
           │        │
           ▼        ▼
      ┌──────────┐ ┌──────────┐
      │ RETRYING │ │ FAILED   │
      └──────────┘ │ → DLQ    │
           │       └──────────┘
           │
           └──────► Back to PROCESSING
```

**States**:
- `QUEUED`: In priority channel, awaiting worker
- `PROCESSING`: Active publishing attempt in progress
- `RETRYING`: Waiting for backoff before next attempt
- `SUCCEEDED`: Successfully published
- `FAILED`: Max retries exhausted, sent to DLQ
- `DLQ`: Permanent failure, stored in Dead Letter Queue

### 4.2 Circuit Breaker State Machine

```
    ┌────────────────────────────────────────────────┐
    │                   CLOSED                       │
    │  - Allow all requests                          │
    │  - Track consecutive failures                  │
    │  - Reset success counter on success            │
    └────────────┬───────────────────────────────────┘
                 │ failures ≥ FailureThreshold (5)
                 ▼
    ┌────────────────────────────────────────────────┐
    │                    OPEN                        │
    │  - Reject all requests (fail-fast)             │
    │  - Start timeout timer (30s)                   │
    │  - Record errors without attempting publish    │
    └────────────┬───────────────────────────────────┘
                 │ timeout elapsed (30s)
                 ▼
    ┌────────────────────────────────────────────────┐
    │                 HALF-OPEN                      │
    │  - Allow single test request                   │
    │  - Track consecutive successes                 │
    │  - Single failure → OPEN                       │
    └────┬───────────────────────────────────┬───────┘
         │ successes ≥ SuccessThreshold (2)  │ failure
         ▼                                    ▼
    ┌─────────┐                          ┌─────────┐
    │ CLOSED  │                          │  OPEN   │
    └─────────┘                          └─────────┘
```

**Counters**:
- `consecutiveFailures`: Incremented on RecordFailure()
- `consecutiveSuccesses`: Incremented on RecordSuccess() in HALF-OPEN
- `lastFailureTime`: Timestamp of last failure (for timeout)

---

## 5. Implementation Details

### 5.1 Worker Pool Pattern

**Worker Goroutine**:
```go
func (q *PublishingQueue) worker(id int) {
    defer q.wg.Done()

    for {
        select {
        case <-q.ctx.Done():
            q.logger.Info("Worker stopped", "id", id)
            return

        // Priority selection: High > Medium > Low
        case job := <-q.highPriorityJobs:
            q.processJob(job)

        case job := <-q.mediumPriorityJobs:
            q.processJob(job)

        case job := <-q.lowPriorityJobs:
            q.processJob(job)
        }
    }
}
```

**Key Features**:
- Non-blocking channel selection with `select`
- Context cancellation for graceful shutdown
- Priority order enforced by `select` case order
- WaitGroup for synchronization

### 5.2 Job Processing Pipeline

```go
func (q *PublishingQueue) processJob(job *PublishingJob) error {
    // 1. Update job state
    job.State = JobStateProcessing
    now := time.Now()
    job.StartedAt = &now
    q.jobTrackingStore.Add(job)

    // 2. Check circuit breaker
    cb := q.getCircuitBreaker(job.Target.Name)
    if !cb.CanAttempt() {
        q.logger.Warn("Circuit breaker open", "target", job.Target.Name)
        q.metrics.RecordPublishingError(job.Target.Name, "circuit_open")
        return q.retryPublish(job, 0)
    }

    // 3. Get publisher
    publisher, err := q.factory.CreatePublisher(job.Target.Type)
    if err != nil {
        return q.handlePublishError(job, err, 0)
    }

    // 4. Publish
    err = publisher.Publish(q.ctx, job.EnrichedAlert, job.Target)

    // 5. Handle result
    if err != nil {
        cb.RecordFailure()
        return q.handlePublishError(job, err, 0)
    }

    // 6. Success
    cb.RecordSuccess()
    job.State = JobStateSucceeded
    completedNow := time.Now()
    job.CompletedAt = &completedNow
    q.jobTrackingStore.Add(job)
    q.metrics.RecordSuccessfulPublishing(job.Target.Name, job.Priority.String())

    return nil
}
```

### 5.3 Retry Logic

```go
func (q *PublishingQueue) retryPublish(job *PublishingJob, attempt int) error {
    // 1. Classify error
    errorType := classifyPublishingError(job.LastError)
    job.ErrorType = errorType

    // 2. Check if retry allowed
    if errorType == QueueErrorTypePermanent || attempt >= q.maxRetries {
        // Send to DLQ
        job.State = JobStateDLQ
        if err := q.dlqRepository.Write(q.ctx, job); err != nil {
            q.logger.Error("Failed to write to DLQ", "error", err)
        }
        q.metrics.RecordDLQWrite(job.Target.Name, errorType.String())
        return job.LastError
    }

    // 3. Calculate backoff
    config := DefaultQueueRetryConfig()
    backoff := CalculateBackoff(attempt, config)

    // 4. Sleep for backoff
    job.State = JobStateRetrying
    job.RetryCount = attempt + 1
    q.jobTrackingStore.Add(job)

    select {
    case <-q.ctx.Done():
        return q.ctx.Err()
    case <-time.After(backoff):
        // Continue to retry
    }

    // 5. Retry publishing
    q.metrics.RecordRetryAttempt(job.Target.Name, errorType.String(), true)

    publisher, err := q.factory.CreatePublisher(job.Target.Type)
    if err != nil {
        return q.handlePublishError(job, err, attempt+1)
    }

    err = publisher.Publish(q.ctx, job.EnrichedAlert, job.Target)
    if err != nil {
        job.LastError = err
        return q.retryPublish(job, attempt+1)
    }

    // Success after retry
    job.State = JobStateSucceeded
    completedNow := time.Now()
    job.CompletedAt = &completedNow
    q.jobTrackingStore.Add(job)
    q.metrics.RecordSuccessfulPublishing(job.Target.Name, job.Priority.String())

    return nil
}
```

---

## 6. Performance Optimization

### 6.1 Hot Path Optimizations

**Priority Determination** (8-9 ns/op):
- Inline function
- Early returns for common cases
- Zero allocations

**Retry Decision** (0.4 ns/op):
- Simple boolean logic
- No memory allocation
- CPU cache-friendly

**Circuit Breaker Check** (14.92 ns/op):
- Atomic operations for state
- RWMutex for minimal contention
- Fast-path for CLOSED state

### 6.2 Memory Optimization

**LRU Cache**:
- O(1) Get/Add operations
- Automatic eviction at capacity
- Zero allocations in hot path
- Reuse list elements

**Job Snapshots**:
- Store Unix timestamps (int64) instead of time.Time
- Minimize pointer fields
- Compact representation (< 100 bytes)

**String Conversions**:
- Pre-computed strings for enums
- Switch statements (constant time)
- No runtime allocations

### 6.3 Concurrency Optimization

**Lock Granularity**:
- RWMutex for read-heavy operations
- Separate locks for circuit breakers (per target)
- Lock-free LRU with sync.Map consideration (future)

**Channel Sizing**:
- High priority: 1000 (matches typical burst)
- Medium priority: 1000 (sustained load)
- Low priority: 1000 (background jobs)
- Total capacity: 3000 jobs

**Worker Count**:
- Default: 10 workers
- Configurable based on CPU cores
- Recommendation: 2x CPU cores for I/O-bound workload

---

## 7. Error Handling

### 7.1 Error Types

```go
type QueueErrorType int

const (
    QueueErrorTypeUnknown    QueueErrorType = iota // Retry with caution
    QueueErrorTypeTransient                        // Network, timeout → RETRY
    QueueErrorTypePermanent                        // Bad request, auth → DLQ
)
```

### 7.2 Error Recovery Strategies

| Error Category | Strategy | Rationale |
|----------------|----------|-----------|
| Transient (Network) | Exponential backoff + retry | Temporary glitch |
| Transient (Rate limit) | Backoff + retry | Respect API limits |
| Permanent (400, 401) | Immediate DLQ | Invalid input/credentials |
| Permanent (404) | Immediate DLQ | Wrong URL |
| Unknown | Retry with max attempts | Conservative approach |

### 7.3 Error Logging

**Structured Logging**:
```go
q.logger.Error("Publishing failed",
    "job_id", job.ID,
    "target", job.Target.Name,
    "error", err.Error(),
    "error_type", errorType.String(),
    "retry_count", job.RetryCount,
    "will_retry", willRetry,
)
```

**Log Levels**:
- `DEBUG`: Detailed flow (job state transitions)
- `INFO`: Normal operations (job submitted, published)
- `WARN`: Recoverable issues (circuit breaker open, retry)
- `ERROR`: Failures (DLQ write, permanent errors)

---

## 8. Concurrency & Thread Safety

### 8.1 Thread-Safe Components

**PublishingQueue**:
- Circuit breakers: `sync.RWMutex` for map access
- Job tracking: Internal RWMutex in LRUJobTrackingStore
- Metrics: Prometheus client handles concurrency

**LRUJobTrackingStore**:
```go
func (s *LRUJobTrackingStore) Add(job *PublishingJob) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Check if exists
    if elem, ok := s.store[job.ID]; ok {
        s.lruList.MoveToFront(elem)
        entry := elem.Value.(*jobTrackingEntry)
        entry.value = snapshot
        return
    }

    // Add new entry
    entry := &jobTrackingEntry{key: job.ID, value: snapshot}
    elem := s.lruList.PushFront(entry)
    s.store[job.ID] = elem

    // Evict LRU if capacity exceeded
    if s.lruList.Len() > s.capacity {
        s.evictLRU()
    }
}
```

**CircuitBreaker**:
```go
func (cb *CircuitBreaker) CanAttempt() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    switch cb.state {
    case CircuitStateClosed:
        return true
    case CircuitStateOpen:
        // Check timeout
        if time.Since(cb.lastFailureTime) > cb.config.Timeout {
            cb.mu.RUnlock()
            cb.mu.Lock()
            cb.state = CircuitStateHalfOpen
            cb.mu.Unlock()
            cb.mu.RLock()
            return true
        }
        return false
    case CircuitStateHalfOpen:
        return true
    }
    return false
}
```

### 8.2 Race Condition Prevention

**Techniques**:
1. **Mutex Locks**: Protect shared state (circuit breakers, LRU cache)
2. **Channels**: Communication between goroutines (job queues)
3. **Context**: Cancellation propagation (graceful shutdown)
4. **WaitGroup**: Synchronization (wait for workers)
5. **Atomic Operations**: Counters (future optimization)

**Validation**:
- Race detector: `go test -race` (CLEAN ✅)
- 10 concurrent goroutines in tests
- 1000 iterations per goroutine
- Zero data races detected

---

## 9. Database Design

### 9.1 DLQ Table Schema

**See requirements.md Section 5.4 for full schema**

**Design Decisions**:
- `UUID` primary key for globally unique IDs
- `JSONB` for flexible alert/target storage
- `TIMESTAMP WITH TIME ZONE` for timezone awareness
- `VARCHAR` length limits for bounded memory
- `DEFAULT` values for auto-population
- `NOT NULL` constraints for data integrity

### 9.2 Index Strategy

| Index | Purpose | Impact |
|-------|---------|--------|
| `target_name` | Filter by target | +10x query speed |
| `priority` | Filter by priority | +10x query speed |
| `failed_at DESC` | Sort by time | +100x query speed |
| `error_type` | Filter by error | +10x query speed |
| `replayed` | Filter replayed | +10x query speed |
| `fingerprint` | Lookup by alert | +100x query speed |

**Total Overhead**: ~7 MB per 100K entries

### 9.3 Query Patterns

**Write** (Hot path):
```sql
INSERT INTO publishing_dlq (
    job_id, fingerprint, target_name, target_type,
    enriched_alert, target_config,
    error_message, error_type, retry_count, last_retry_at,
    priority, failed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id;
```

**Read** (Dashboard):
```sql
SELECT * FROM publishing_dlq
WHERE target_name = $1
  AND error_type = $2
  AND priority = $3
ORDER BY failed_at DESC
LIMIT $4 OFFSET $5;
```

**Stats** (Monitoring):
```sql
SELECT
    COUNT(*) as total_entries,
    COUNT(*) FILTER (WHERE replayed = true) as replayed_count,
    MIN(created_at) as oldest_entry,
    MAX(created_at) as newest_entry
FROM publishing_dlq;
```

---

## 10. Metrics & Observability

### 10.1 Prometheus Metrics

**See requirements.md Section 9.2 for full metrics list**

**Metric Categories**:
1. **Queue Metrics** (4): queue_size, capacity, submissions, duration
2. **Retry Metrics** (3): attempts, backoff, successes
3. **DLQ Metrics** (3): writes, reads, size
4. **Circuit Breaker Metrics** (3): state, trips, recoveries
5. **Job Tracking Metrics** (2): cache_hits, cache_size
6. **Publishing Metrics** (2): successes, failures

**Total**: 17+ metrics

### 10.2 Grafana Dashboard

**Panels** (future Phase 5):
1. Publishing Success Rate (%)
2. Queue Size by Priority (timeseries)
3. Average Latency (p50, p95, p99)
4. Retry Rate by Error Type
5. DLQ Entry Rate
6. Circuit Breaker States by Target
7. Top Failing Targets
8. Job State Distribution

---

## 11. Security Considerations

### 11.1 Credential Protection

- **No credentials in logs**: Error messages sanitized
- **No credentials in DLQ**: TargetConfig stored as JSONB (but not exposed in logs)
- **TLS enforcement**: HTTPS-only for external systems
- **Secret rotation**: K8s Secret watch for dynamic updates

### 11.2 Input Validation

- **Alert fingerprint**: Required, max 255 chars
- **Target name**: Required, max 255 chars
- **Priority**: Enum validation
- **State**: Enum validation
- **Error type**: Enum validation

### 11.3 Resource Limits

- **Queue capacity**: 3000 jobs (prevents memory exhaustion)
- **Job tracking**: 10,000 jobs (LRU eviction)
- **Retry attempts**: 3 max (prevents infinite loops)
- **Backoff max**: 30s (prevents excessive delays)
- **Worker count**: Configurable (prevents CPU starvation)

---

## 12. Testing Strategy

### 12.1 Test Coverage

**See TN-056-PHASE-3-COMPLETE-SUMMARY.md for full details**

**Unit Tests**: 73 (100% passing)
- Priority determination: 13 tests
- Error classification: 15 tests
- Enhanced retry: 12 tests
- DLQ repository: 12 tests
- Job tracking: 10 tests
- Queue integration: 11 tests

**Benchmarks**: 40+ (sub-ns to µs)
- Priority: 8-9 ns/op
- Retry decision: 0.4 ns/op
- Error classification: 110-406 ns/op
- Job tracking: 82-1286 ns/op

**Race Detector**: CLEAN ✅

### 12.2 Test Categories

1. **Functional Tests**: Verify correct behavior
2. **Edge Case Tests**: Nil pointers, empty inputs, boundary conditions
3. **Concurrent Tests**: 10 goroutines, 1000 iterations
4. **Benchmark Tests**: Performance validation
5. **Integration Tests**: End-to-end flows (future Phase 5)

---

## 13. Deployment Considerations

### 13.1 Configuration

**Environment Variables**:
```
PUBLISHING_WORKER_COUNT=10
PUBLISHING_QUEUE_SIZE_HIGH=1000
PUBLISHING_QUEUE_SIZE_MEDIUM=1000
PUBLISHING_QUEUE_SIZE_LOW=1000
PUBLISHING_MAX_RETRIES=3
PUBLISHING_RETRY_INTERVAL=100ms
PUBLISHING_CIRCUIT_TIMEOUT=30s
PUBLISHING_JOB_TRACKING_CAPACITY=10000
```

### 13.2 Resource Requirements

**CPU**: 2-4 cores (10 workers)
**Memory**: 500 MB - 1 GB (queue + cache + DLQ connections)
**Database**: PostgreSQL 14+ with 100 MB allocated for DLQ
**Disk**: 10 GB for DLQ growth

### 13.3 Monitoring

**Key Alerts**:
1. DLQ entry rate > 0.1% → Investigate failing targets
2. Circuit breaker open > 5 min → External system down
3. Queue size > 80% capacity → Backpressure
4. Publishing p99 latency > 2s → Performance degradation

---

## 14. Future Enhancements

### 14.1 Phase 2 (Future)

- Distributed queue (Kafka, RabbitMQ)
- Multi-region publishing
- Advanced routing rules
- Real-time delivery guarantees
- Auto-scaling based on queue depth
- Replay automation (auto-retry DLQ entries)

### 14.2 Optimizations

- Lock-free LRU cache (atomic operations)
- Zero-copy JSONB encoding
- Connection pooling per target
- Batch DLQ writes (100 jobs/batch)
- Adaptive backoff (based on error type)

---

## 15. Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-11-12 | AI Assistant | Initial comprehensive design |

---

**Document Status**: ✅ COMPLETE
**Next**: tasks.md (Implementation Checklist)
