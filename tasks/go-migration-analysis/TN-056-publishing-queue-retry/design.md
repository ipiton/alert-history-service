# TN-056: Publishing Queue с Retry - Technical Design Document

**Дата**: 2025-11-12
**Автор**: AI Assistant
**Версия**: 1.0 (Target: 150% Quality)
**Статус**: 📐 DESIGN PHASE

---

## 🎯 Executive Summary

Этот документ описывает **техническую архитектуру** TN-056 Publishing Queue с акцентом на **150% качество** (Grade A+). Design фокусируется на:

1. **Reliability**: Circuit breakers, DLQ, retry с error classification
2. **Performance**: <10ms latency, >1,000 jobs/sec throughput
3. **Observability**: 12+ Prometheus metrics, structured logging
4. **Scalability**: Horizontal scaling (1-10 replicas), dynamic worker pool

---

## 📊 1. System Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Alert History Service                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐      ┌───────────────────┐                       │
│  │ AlertProcessor│─────▶│PublishingCoordinator│                       │
│  │  (TN-033)    │      │   (existing)       │                       │
│  └──────────────┘      └──────────┬─────────┘                       │
│                                    │                                  │
│                                    │ Submit(alert, targets)           │
│                                    ▼                                  │
│               ┌────────────────────────────────────┐                │
│               │      PublishingQueue (TN-056)      │                │
│               ├────────────────────────────────────┤                │
│               │                                     │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │  Priority Queues (3 tiers)   │  │                │
│               │  ├──────────────────────────────┤  │                │
│               │  │  HIGH    ▒▒▒▒▒▒ (500 cap)    │  │                │
│               │  │  MEDIUM  ▒▒▒▒▒▒▒▒ (1000 cap) │  │                │
│               │  │  LOW     ▒▒▒▒▒▒ (500 cap)    │  │                │
│               │  └──────────────────────────────┘  │                │
│               │                                     │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │   Worker Pool (10 workers)    │  │                │
│               │  ├──────────────────────────────┤  │                │
│               │  │  Worker 0  │  Worker 1  │... │  │                │
│               │  │     │           │         │   │  │                │
│               │  │     ▼           ▼         ▼   │  │                │
│               │  │  ┌─────────────────────────┐ │  │                │
│               │  │  │   Retry Engine          │ │  │                │
│               │  │  │   (Exponential Backoff) │ │  │                │
│               │  │  └─────────────────────────┘ │  │                │
│               │  │     │           │         │   │  │                │
│               │  │     ▼           ▼         ▼   │  │                │
│               │  │  ┌─────────────────────────┐ │  │                │
│               │  │  │  Circuit Breakers (CB)  │ │  │                │
│               │  │  │  (per target)           │ │  │                │
│               │  │  └─────────────────────────┘ │  │                │
│               │  └──────────────────────────────┘  │                │
│               │                                     │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │  PublisherFactory (TN-051)   │  │                │
│               │  ├──────────────────────────────┤  │                │
│               │  │  Rootly │ PagerDuty │ Slack  │  │                │
│               │  │  Webhook │ ...                │  │                │
│               │  └──────────────────────────────┘  │                │
│               │          │                          │                │
│               │          ▼                          │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │  Dead Letter Queue (DLQ)     │  │                │
│               │  │  (PostgreSQL)                │  │                │
│               │  │  TTL: 7 days                 │  │                │
│               │  └──────────────────────────────┘  │                │
│               │                                     │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │  Job Tracking Store          │  │                │
│               │  │  (LRU cache, 1000 jobs)      │  │                │
│               │  └──────────────────────────────┘  │                │
│               │                                     │                │
│               │  ┌──────────────────────────────┐  │                │
│               │  │  PublishingMetrics           │  │                │
│               │  │  (12+ Prometheus metrics)    │  │                │
│               │  └──────────────────────────────┘  │                │
│               └────────────────────────────────────┘                │
│                           │                                           │
│                           ▼                                           │
│               ┌────────────────────────┐                             │
│               │  External Systems       │                             │
│               ├────────────────────────┤                             │
│               │  Rootly   │ PagerDuty  │                             │
│               │  Slack    │ Webhooks   │                             │
│               └────────────────────────┘                             │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

| Component | Responsibility | Status |
|-----------|----------------|--------|
| **AlertProcessor** | Classify/filter alerts → submit to coordinator | ✅ Existing |
| **PublishingCoordinator** | Resolve targets → submit to queue | ✅ Existing |
| **PublishingQueue** | Async processing, retry, circuit breakers | 🔧 ENHANCE (65% → 150%) |
| **Priority Queues** | 3-tier job prioritization (HIGH/MED/LOW) | ❌ NEW (150% feature) |
| **Worker Pool** | Concurrent job processing (10 workers) | ✅ Existing |
| **Retry Engine** | Exponential backoff, error classification | 🔧 ENHANCE |
| **Circuit Breakers** | Per-target fail-fast protection | ✅ Existing |
| **PublisherFactory** | Create publishers (Rootly/PD/Slack/Webhook) | ✅ Existing (TN-051-055) |
| **DLQ** | Persistent storage для failed jobs | ❌ NEW (150% feature) |
| **Job Tracking** | Real-time job status API | ❌ NEW (150% feature) |
| **PublishingMetrics** | 12+ Prometheus metrics | ❌ NEW (150% critical) |

---

## 📊 2. Data Models

### 2.1 PublishingJob (ENHANCED)

```go
// PublishingJob represents a single publishing task
type PublishingJob struct {
    // Core fields (EXISTING)
    EnrichedAlert *core.EnrichedAlert
    Target        *core.PublishingTarget
    RetryCount    int
    SubmittedAt   time.Time

    // NEW fields for 150% quality
    ID            string                // UUID v4 (crypto/rand)
    Priority      Priority               // HIGH/MEDIUM/LOW
    State         JobState               // queued/processing/retrying/succeeded/failed/dlq
    StartedAt     *time.Time            // When processing began
    CompletedAt   *time.Time            // When processing completed
    LastError     error                 // Most recent error
    ErrorType     ErrorType              // transient/permanent/unknown
}

// Priority levels for job processing order
type Priority int

const (
    PriorityHigh   Priority = 0  // Critical alerts (severity=critical)
    PriorityMedium Priority = 1  // Warning alerts (default)
    PriorityLow    Priority = 2  // Info alerts, resolved alerts
)

func (p Priority) String() string {
    switch p {
    case PriorityHigh: return "high"
    case PriorityMedium: return "medium"
    case PriorityLow: return "low"
    default: return "unknown"
    }
}

// JobState represents the current state of a job
type JobState int

const (
    JobStateQueued     JobState = iota  // Job submitted to queue
    JobStateProcessing                   // Worker picked up job
    JobStateRetrying                     // Job failed, retrying
    JobStateSucceeded                    // Job completed successfully
    JobStateFailed                       // Job failed (permanent error)
    JobStateDLQ                          // Job sent to DLQ after max retries
)

func (s JobState) String() string {
    switch s {
    case JobStateQueued: return "queued"
    case JobStateProcessing: return "processing"
    case JobStateRetrying: return "retrying"
    case JobStateSucceeded: return "succeeded"
    case JobStateFailed: return "failed"
    case JobStateDLQ: return "dlq"
    default: return "unknown"
    }
}

// ErrorType classifies errors for retry logic
type ErrorType int

const (
    ErrorTypeUnknown    ErrorType = iota  // Default, retry with caution
    ErrorTypeTransient                     // Network timeout, rate limit, 502/503/504 → RETRY
    ErrorTypePermanent                     // 400 bad request, 401 unauthorized, 404 → NO RETRY
)

func (e ErrorType) String() string {
    switch e {
    case ErrorTypeTransient: return "transient"
    case ErrorTypePermanent: return "permanent"
    default: return "unknown"
    }
}
```

### 2.2 PublishingQueue (ENHANCED STRUCT)

```go
// PublishingQueue manages async publishing with worker pool and retry logic
type PublishingQueue struct {
    // Core components (EXISTING)
    factory       *PublisherFactory
    workerCount   int
    maxRetries    int
    retryInterval time.Duration
    logger        *slog.Logger
    wg            sync.WaitGroup
    ctx           context.Context
    cancel        context.CancelFunc

    // Priority queues (NEW for 150%)
    highPriorityJobs   chan *PublishingJob  // capacity: 500
    mediumPriorityJobs chan *PublishingJob  // capacity: 1000
    lowPriorityJobs    chan *PublishingJob  // capacity: 500

    // Circuit breakers (EXISTING)
    circuitBreakers map[string]*CircuitBreaker  // key: target name
    cbMu            sync.RWMutex

    // Metrics (NEW for 150%)
    metrics *PublishingMetrics

    // Job tracking (NEW for 150%)
    jobStore *JobTrackingStore  // LRU cache of recent jobs

    // DLQ (NEW for 150%)
    dlq *DeadLetterQueue  // PostgreSQL backed

    // Error classifier (NEW for 150%)
    errorClassifier *ErrorClassifier
}

// PublishingQueueConfig holds configuration for publishing queue
type PublishingQueueConfig struct {
    // Worker pool
    WorkerCount    int           // Default: 10

    // Queue sizes
    HighPriorityQueueSize   int  // Default: 500
    MediumPriorityQueueSize int  // Default: 1000
    LowPriorityQueueSize    int  // Default: 500

    // Retry configuration
    MaxRetries         int           // Default: 3
    InitialRetryDelay  time.Duration // Default: 100ms
    MaxRetryDelay      time.Duration // Default: 30s
    RetryGrowthFactor  float64       // Default: 2.0 (exponential)
    RetryJitter        float64       // Default: 0.1 (±10%)

    // Circuit breaker (per target)
    CBFailureThreshold int           // Default: 5
    CBSuccessThreshold int           // Default: 2
    CBTimeout          time.Duration // Default: 30s

    // Job tracking
    JobTrackingEnabled  bool         // Default: true
    JobTrackingCapacity int          // Default: 1000 (LRU)
    JobTrackingTTL      time.Duration// Default: 1 hour

    // DLQ
    DLQEnabled bool                  // Default: true
    DLQTTL     time.Duration         // Default: 7 days
}

// DefaultPublishingQueueConfig returns production defaults
func DefaultPublishingQueueConfig() PublishingQueueConfig {
    return PublishingQueueConfig{
        WorkerCount:             10,
        HighPriorityQueueSize:   500,
        MediumPriorityQueueSize: 1000,
        LowPriorityQueueSize:    500,
        MaxRetries:              3,
        InitialRetryDelay:       100 * time.Millisecond,
        MaxRetryDelay:           30 * time.Second,
        RetryGrowthFactor:       2.0,
        RetryJitter:             0.1,
        CBFailureThreshold:      5,
        CBSuccessThreshold:      2,
        CBTimeout:               30 * time.Second,
        JobTrackingEnabled:      true,
        JobTrackingCapacity:     1000,
        JobTrackingTTL:          1 * time.Hour,
        DLQEnabled:              true,
        DLQTTL:                  7 * 24 * time.Hour,
    }
}
```

### 2.3 DLQ Schema (PostgreSQL)

```sql
-- Dead Letter Queue table (persistent failed jobs)
CREATE TABLE IF NOT EXISTS publishing_dlq (
    id UUID PRIMARY KEY,                      -- Job ID (matches PublishingJob.ID)
    job_data JSONB NOT NULL,                  -- Full job serialized (EnrichedAlert + Target)
    target_name VARCHAR(255) NOT NULL,        -- Target name for filtering
    target_type VARCHAR(50) NOT NULL,         -- rootly/pagerduty/slack/webhook
    fingerprint VARCHAR(64) NOT NULL,         -- Alert fingerprint
    priority VARCHAR(20) NOT NULL,            -- high/medium/low
    error_message TEXT,                       -- Final error message
    error_type VARCHAR(50),                   -- transient/permanent/unknown
    retry_count INT NOT NULL,                 -- How many retries attempted
    failed_at TIMESTAMP NOT NULL,             -- When job failed (final)
    expires_at TIMESTAMP NOT NULL,            -- TTL expiration (failed_at + 7 days)
    replayed BOOLEAN DEFAULT FALSE,           -- Was this job replayed by admin?
    replayed_at TIMESTAMP,                    -- When replayed
    replay_result VARCHAR(50),                -- success/failure (after replay)
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_dlq_target_name ON publishing_dlq(target_name);
CREATE INDEX IF NOT EXISTS idx_dlq_target_type ON publishing_dlq(target_type);
CREATE INDEX IF NOT EXISTS idx_dlq_failed_at ON publishing_dlq(failed_at DESC);
CREATE INDEX IF NOT EXISTS idx_dlq_expires_at ON publishing_dlq(expires_at);
CREATE INDEX IF NOT EXISTS idx_dlq_replayed ON publishing_dlq(replayed) WHERE NOT replayed;
CREATE INDEX IF NOT EXISTS idx_dlq_fingerprint ON publishing_dlq(fingerprint);
```

### 2.4 Job Tracking Store (In-Memory LRU)

```go
// JobTrackingStore stores recent job status for API queries
type JobTrackingStore struct {
    cache    *lru.Cache    // LRU cache (max 1000 jobs)
    mu       sync.RWMutex
    ttl      time.Duration // 1 hour TTL per job
}

// JobStatus represents current job state for API
type JobStatus struct {
    ID                  string                `json:"id"`
    State               JobState              `json:"state"`
    Priority            Priority              `json:"priority"`
    TargetName          string                `json:"target"`
    TargetType          string                `json:"target_type"`
    Fingerprint         string                `json:"fingerprint"`
    SubmittedAt         time.Time             `json:"submitted_at"`
    StartedAt           *time.Time            `json:"started_at,omitempty"`
    CompletedAt         *time.Time            `json:"completed_at,omitempty"`
    RetryCount          int                   `json:"retry_count"`
    MaxRetries          int                   `json:"max_retries"`
    LastError           string                `json:"last_error,omitempty"`
    ErrorType           ErrorType             `json:"error_type,omitempty"`
    CircuitBreakerState CircuitBreakerState   `json:"circuit_breaker_state"`

    // Computed fields
    QueueDuration       *time.Duration        `json:"queue_duration,omitempty"`  // StartedAt - SubmittedAt
    ProcessingDuration  *time.Duration        `json:"processing_duration,omitempty"` // CompletedAt - StartedAt
}

// Methods:
// - Store(job *PublishingJob) error
// - Get(id string) (*JobStatus, error)
// - List(limit, offset int) ([]*JobStatus, error)
// - CleanupExpired() // Background worker
```

---

## 📊 3. Core Components Design

### 3.1 Priority Queue System

**Goal**: Process critical alerts first (HIGH > MEDIUM > LOW)

#### 3.1.1 Priority Determination

```go
// determinePriority classifies jobs into priority tiers
func determinePriority(enrichedAlert *core.EnrichedAlert) Priority {
    alert := enrichedAlert.Alert

    // HIGH priority: Critical firing alerts
    if alert.Severity == "critical" && alert.Status == "firing" {
        return PriorityHigh
    }

    // HIGH priority: LLM confidence = critical
    if enrichedAlert.Classification != nil {
        if enrichedAlert.Classification.Severity == "critical" {
            return PriorityHigh
        }
    }

    // LOW priority: Resolved alerts, info severity
    if alert.Status == "resolved" || alert.Severity == "info" {
        return PriorityLow
    }

    // DEFAULT: MEDIUM priority
    return PriorityMedium
}
```

#### 3.1.2 Submit with Priority

```go
// Submit submits job to appropriate priority queue
func (q *PublishingQueue) Submit(enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
    // Generate job ID
    jobID := uuid.NewString()

    // Determine priority
    priority := determinePriority(enrichedAlert)

    // Create job
    job := &PublishingJob{
        ID:            jobID,
        EnrichedAlert: enrichedAlert,
        Target:        target,
        Priority:      priority,
        State:         JobStateQueued,
        RetryCount:    0,
        SubmittedAt:   time.Now(),
    }

    // Submit to appropriate queue
    var targetQueue chan *PublishingJob
    switch priority {
    case PriorityHigh:
        targetQueue = q.highPriorityJobs
    case PriorityMedium:
        targetQueue = q.mediumPriorityJobs
    case PriorityLow:
        targetQueue = q.lowPriorityJobs
    default:
        targetQueue = q.mediumPriorityJobs
    }

    // Non-blocking submit with metrics
    select {
    case targetQueue <- job:
        q.metrics.RecordQueueSubmission(priority, true)
        q.metrics.UpdateQueueSize(priority, len(targetQueue), cap(targetQueue))

        // Store in job tracking
        if q.jobStore != nil {
            q.jobStore.Store(job)
        }

        q.logger.Debug("Job submitted",
            "job_id", jobID,
            "priority", priority,
            "target", target.Name,
            "fingerprint", enrichedAlert.Alert.Fingerprint,
        )
        return nil

    case <-q.ctx.Done():
        q.metrics.RecordQueueSubmission(priority, false)
        return fmt.Errorf("queue shutting down")

    default:
        // Queue full
        q.metrics.RecordQueueSubmission(priority, false)
        return fmt.Errorf("queue full (priority=%s, capacity=%d)", priority, cap(targetQueue))
    }
}
```

#### 3.1.3 Worker Priority Selection

```go
// worker processes jobs with priority order (HIGH > MEDIUM > LOW)
func (q *PublishingQueue) worker(id int) {
    defer q.wg.Done()

    q.logger.Debug("Worker started", "worker_id", id)

    for {
        var job *PublishingJob
        var priority Priority

        // Priority-based select (HIGH checked first)
        select {
        case job = <-q.highPriorityJobs:
            priority = PriorityHigh
        case <-q.ctx.Done():
            return
        default:
            // Check medium, then low
            select {
            case job = <-q.mediumPriorityJobs:
                priority = PriorityMedium
            case <-q.ctx.Done():
                return
            default:
                // Check low
                select {
                case job = <-q.lowPriorityJobs:
                    priority = PriorityLow
                case <-q.ctx.Done():
                    return
                case <-time.After(100 * time.Millisecond):
                    // Idle timeout, loop back to check high priority
                    continue
                }
            }
        }

        if job != nil {
            // Update metrics
            q.metrics.RecordWorkerActive(id, true)

            // Process job
            q.processJob(job)

            // Update metrics
            q.metrics.RecordWorkerActive(id, false)
            q.metrics.UpdateQueueSize(priority, len(q.getQueueForPriority(priority)), cap(q.getQueueForPriority(priority)))
        }
    }
}
```

---

### 3.2 Retry Engine с Error Classification

**Goal**: Smart retry только для transient errors, skip permanent errors

#### 3.2.1 Error Classifier

```go
// ErrorClassifier classifies errors as transient/permanent for retry logic
type ErrorClassifier struct {
    logger *slog.Logger
}

// ClassifyError determines if error is retryable
func (ec *ErrorClassifier) ClassifyError(err error) ErrorType {
    if err == nil {
        return ErrorTypeUnknown
    }

    errMsg := err.Error()

    // Permanent errors (HTTP 4xx client errors)
    permanentPatterns := []string{
        "HTTP 400", "bad request",
        "HTTP 401", "unauthorized",
        "HTTP 403", "forbidden",
        "HTTP 404", "not found",
        "HTTP 405", "method not allowed",
        "HTTP 409", "conflict",
        "HTTP 410", "gone",
        "HTTP 422", "unprocessable entity",
        "invalid", "malformed", "parse error",
    }

    for _, pattern := range permanentPatterns {
        if strings.Contains(strings.ToLower(errMsg), pattern) {
            return ErrorTypePermanent
        }
    }

    // Transient errors (network, timeouts, server errors)
    transientPatterns := []string{
        "HTTP 429", "rate limit", "too many requests",
        "HTTP 500", "internal server error",
        "HTTP 502", "bad gateway",
        "HTTP 503", "service unavailable",
        "HTTP 504", "gateway timeout",
        "timeout", "timed out", "deadline exceeded",
        "connection refused", "connection reset",
        "no such host", "DNS", "network",
        "EOF", "broken pipe",
        "temporary failure", "try again",
    }

    for _, pattern := range transientPatterns {
        if strings.Contains(strings.ToLower(errMsg), pattern) {
            return ErrorTypeTransient
        }
    }

    // Default: unknown (retry with caution)
    return ErrorTypeUnknown
}

// ShouldRetry returns true if error is retryable
func (ec *ErrorClassifier) ShouldRetry(err error, attempt int, maxRetries int) bool {
    if err == nil {
        return false
    }

    if attempt >= maxRetries {
        return false
    }

    errorType := ec.ClassifyError(err)

    switch errorType {
    case ErrorTypeTransient:
        return true  // Always retry transient errors
    case ErrorTypePermanent:
        return false  // Never retry permanent errors
    case ErrorTypeUnknown:
        return attempt < 1  // Retry once for unknown errors (cautious)
    default:
        return false
    }
}
```

#### 3.2.2 Retry Logic с Jitter

```go
// retryPublish attempts publish with exponential backoff + jitter
func (q *PublishingQueue) retryPublish(publisher AlertPublisher, job *PublishingJob) error {
    var lastErr error

    for attempt := 0; attempt <= q.config.MaxRetries; attempt++ {
        // Update job state
        job.State = JobStateProcessing
        if attempt > 0 {
            job.State = JobStateRetrying
        }
        job.RetryCount = attempt
        q.jobStore.Update(job)

        // Try publish
        err := publisher.Publish(q.ctx, job.EnrichedAlert, job.Target)
        if err == nil {
            // Success
            job.State = JobStateSucceeded
            completedAt := time.Now()
            job.CompletedAt = &completedAt
            q.jobStore.Update(job)

            q.metrics.RecordJobSuccess(job.Target.Name, job.Priority, time.Since(job.SubmittedAt).Seconds())
            return nil
        }

        lastErr = err

        // Classify error
        errorType := q.errorClassifier.ClassifyError(err)
        job.ErrorType = errorType
        job.LastError = err

        q.logger.Warn("Publish attempt failed",
            "job_id", job.ID,
            "attempt", attempt+1,
            "max_retries", q.config.MaxRetries,
            "error_type", errorType,
            "error", err,
        )

        // Check if should retry
        if !q.errorClassifier.ShouldRetry(err, attempt, q.config.MaxRetries) {
            q.logger.Warn("Permanent error, not retrying",
                "job_id", job.ID,
                "error_type", errorType,
            )
            q.metrics.RecordRetryAttempt(job.Target.Name, errorType, false)
            break
        }

        // Record retry attempt
        q.metrics.RecordRetryAttempt(job.Target.Name, errorType, true)

        // Don't sleep after last attempt
        if attempt < q.config.MaxRetries {
            // Exponential backoff: initial * (growthFactor ^ attempt)
            backoff := float64(q.config.InitialRetryDelay) * math.Pow(q.config.RetryGrowthFactor, float64(attempt))

            // Cap at MaxRetryDelay
            if backoff > float64(q.config.MaxRetryDelay) {
                backoff = float64(q.config.MaxRetryDelay)
            }

            // Add jitter (±10%)
            jitter := backoff * q.config.RetryJitter * (2*rand.Float64() - 1)
            backoffDuration := time.Duration(backoff + jitter)

            q.logger.Debug("Retrying after backoff",
                "job_id", job.ID,
                "attempt", attempt+1,
                "backoff", backoffDuration,
            )

            select {
            case <-time.After(backoffDuration):
                // Continue to next attempt
            case <-q.ctx.Done():
                return q.ctx.Err()
            }
        }
    }

    // All retries exhausted, send to DLQ
    job.State = JobStateFailed
    completedAt := time.Now()
    job.CompletedAt = &completedAt
    q.jobStore.Update(job)

    if q.dlq != nil && q.config.DLQEnabled {
        if err := q.dlq.Write(job); err != nil {
            q.logger.Error("Failed to write to DLQ",
                "job_id", job.ID,
                "error", err,
            )
        } else {
            job.State = JobStateDLQ
            q.jobStore.Update(job)
            q.metrics.RecordDLQWrite(job.Target.Name, job.ErrorType)
        }
    }

    q.metrics.RecordJobFailure(job.Target.Name, job.Priority, job.ErrorType)
    return fmt.Errorf("failed after %d retries: %w", q.config.MaxRetries, lastErr)
}
```

---

### 3.3 Dead Letter Queue (DLQ)

**Goal**: Persistent storage для failed jobs, admin replay capability

```go
// DeadLetterQueue stores failed jobs for later inspection/replay
type DeadLetterQueue struct {
    db     *sql.DB
    logger *slog.Logger
}

// NewDeadLetterQueue creates DLQ with PostgreSQL backend
func NewDeadLetterQueue(db *sql.DB, logger *slog.Logger) *DeadLetterQueue {
    return &DeadLetterQueue{
        db:     db,
        logger: logger,
    }
}

// Write stores failed job in DLQ
func (dlq *DeadLetterQueue) Write(job *PublishingJob) error {
    // Serialize job to JSON
    jobData, err := json.Marshal(job)
    if err != nil {
        return fmt.Errorf("failed to marshal job: %w", err)
    }

    // Calculate expiration (7 days from now)
    expiresAt := time.Now().Add(7 * 24 * time.Hour)

    // Insert into DLQ table
    query := `
        INSERT INTO publishing_dlq (
            id, job_data, target_name, target_type, fingerprint, priority,
            error_message, error_type, retry_count, failed_at, expires_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `

    _, err = dlq.db.Exec(query,
        job.ID,
        jobData,
        job.Target.Name,
        job.Target.Type,
        job.EnrichedAlert.Alert.Fingerprint,
        job.Priority.String(),
        job.LastError.Error(),
        job.ErrorType.String(),
        job.RetryCount,
        time.Now(),
        expiresAt,
    )

    if err != nil {
        return fmt.Errorf("failed to insert into DLQ: %w", err)
    }

    dlq.logger.Info("Job written to DLQ",
        "job_id", job.ID,
        "target", job.Target.Name,
        "expires_at", expiresAt,
    )

    return nil
}

// List returns DLQ jobs with pagination
func (dlq *DeadLetterQueue) List(limit, offset int, filters DLQFilters) ([]*DLQEntry, error) {
    query := `
        SELECT id, target_name, target_type, fingerprint, priority,
               error_message, error_type, retry_count, failed_at, expires_at,
               replayed, replayed_at, replay_result
        FROM publishing_dlq
        WHERE expires_at > NOW()
    `

    // Apply filters
    args := []interface{}{}
    argIdx := 1

    if filters.TargetName != "" {
        query += fmt.Sprintf(" AND target_name = $%d", argIdx)
        args = append(args, filters.TargetName)
        argIdx++
    }

    if filters.ErrorType != "" {
        query += fmt.Sprintf(" AND error_type = $%d", argIdx)
        args = append(args, filters.ErrorType)
        argIdx++
    }

    query += " ORDER BY failed_at DESC"
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
    args = append(args, limit, offset)

    rows, err := dlq.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    entries := []*DLQEntry{}
    for rows.Next() {
        entry := &DLQEntry{}
        err := rows.Scan(
            &entry.ID, &entry.TargetName, &entry.TargetType, &entry.Fingerprint, &entry.Priority,
            &entry.ErrorMessage, &entry.ErrorType, &entry.RetryCount, &entry.FailedAt, &entry.ExpiresAt,
            &entry.Replayed, &entry.ReplayedAt, &entry.ReplayResult,
        )
        if err != nil {
            return nil, err
        }
        entries = append(entries, entry)
    }

    return entries, nil
}

// Replay attempts to re-publish a DLQ job
func (dlq *DeadLetterQueue) Replay(jobID string, queue *PublishingQueue) error {
    // Fetch job from DLQ
    var jobData []byte
    query := `SELECT job_data FROM publishing_dlq WHERE id = $1 AND NOT replayed`
    err := dlq.db.QueryRow(query, jobID).Scan(&jobData)
    if err != nil {
        return fmt.Errorf("job not found or already replayed: %w", err)
    }

    // Deserialize job
    job := &PublishingJob{}
    if err := json.Unmarshal(jobData, job); err != nil {
        return fmt.Errorf("failed to unmarshal job: %w", err)
    }

    // Reset job state for replay
    job.RetryCount = 0
    job.State = JobStateQueued
    job.SubmittedAt = time.Now()
    job.StartedAt = nil
    job.CompletedAt = nil
    job.LastError = nil

    // Re-submit to queue
    err = queue.Submit(job.EnrichedAlert, job.Target)

    // Update DLQ entry
    updateQuery := `
        UPDATE publishing_dlq
        SET replayed = TRUE, replayed_at = NOW(), replay_result = $1
        WHERE id = $2
    `

    replayResult := "success"
    if err != nil {
        replayResult = "failure"
    }

    _, updateErr := dlq.db.Exec(updateQuery, replayResult, jobID)
    if updateErr != nil {
        dlq.logger.Error("Failed to update DLQ replay status", "error", updateErr)
    }

    return err
}

// CleanupExpired deletes expired DLQ entries (background worker)
func (dlq *DeadLetterQueue) CleanupExpired() error {
    query := `DELETE FROM publishing_dlq WHERE expires_at < NOW()`
    result, err := dlq.db.Exec(query)
    if err != nil {
        return err
    }

    rowsDeleted, _ := result.RowsAffected()
    if rowsDeleted > 0 {
        dlq.logger.Info("Cleaned up expired DLQ entries", "count", rowsDeleted)
    }

    return nil
}
```

---

### 3.4 Publishing Metrics (12+ Prometheus Metrics)

```go
// PublishingMetrics exports 12+ Prometheus metrics for queue observability
type PublishingMetrics struct {
    // Queue size metrics
    queueSize             *prometheus.GaugeVec      // Current queue depth by priority
    queueCapacityUtil     *prometheus.GaugeVec      // Queue utilization (0-1) by priority
    queueSubmissions      *prometheus.CounterVec    // Submissions by priority, result

    // Job processing metrics
    jobsProcessed         *prometheus.CounterVec    // Jobs by target, state
    jobDuration           *prometheus.HistogramVec  // Processing duration by target, priority
    jobWaitTime           *prometheus.HistogramVec  // Queue wait time by priority

    // Retry metrics
    retryAttempts         *prometheus.CounterVec    // Retries by target, error_type
    retrySuccessRate      *prometheus.HistogramVec  // Success rate by target, attempt

    // Circuit breaker metrics
    circuitBreakerState   *prometheus.GaugeVec      // CB state by target
    circuitBreakerTrips   *prometheus.CounterVec    // CB trips by target
    circuitBreakerRecoveries *prometheus.CounterVec // CB recoveries by target

    // Worker pool metrics
    workersActive         prometheus.Gauge          // Active workers
    workersIdle           prometheus.Gauge          // Idle workers
    workerProcessingTime  *prometheus.HistogramVec  // Processing time by worker_id

    // DLQ metrics
    dlqSize               *prometheus.GaugeVec      // DLQ size by target
    dlqWrites             *prometheus.CounterVec    // DLQ writes by target, error_type
    dlqReplays            *prometheus.CounterVec    // DLQ replays by target, result
}

// NewPublishingMetrics creates and registers all metrics
func NewPublishingMetrics(registry prometheus.Registerer) *PublishingMetrics {
    m := &PublishingMetrics{
        queueSize: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "queue_size",
                Help:      "Current queue depth by priority (high/medium/low)",
            },
            []string{"priority"},
        ),

        queueCapacityUtil: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "queue_capacity_utilization",
                Help:      "Queue capacity utilization (0-1) by priority",
            },
            []string{"priority"},
        ),

        queueSubmissions: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "queue_submissions_total",
                Help:      "Total queue submissions by priority and result (success/rejected)",
            },
            []string{"priority", "result"},
        ),

        jobsProcessed: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "jobs_processed_total",
                Help:      "Total jobs processed by target and state (succeeded/failed/dlq)",
            },
            []string{"target", "state"},
        ),

        jobDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "job_duration_seconds",
                Help:      "Job processing duration (queue to completion) by target and priority",
                Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
            },
            []string{"target", "priority"},
        ),

        jobWaitTime: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "job_wait_time_seconds",
                Help:      "Time spent in queue (submitted to started) by priority",
                Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
            },
            []string{"priority"},
        ),

        retryAttempts: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "retry_attempts_total",
                Help:      "Total retry attempts by target and error_type (transient/permanent/unknown)",
            },
            []string{"target", "error_type"},
        ),

        retrySuccessRate: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "retry_success_rate",
                Help:      "Retry success rate by target and attempt number",
                Buckets:   prometheus.LinearBuckets(0, 0.1, 11), // 0-1 in 0.1 steps
            },
            []string{"target", "attempt"},
        ),

        circuitBreakerState: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "circuit_breaker_state",
                Help:      "Circuit breaker state by target (0=closed, 1=halfopen, 2=open)",
            },
            []string{"target"},
        ),

        circuitBreakerTrips: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "circuit_breaker_trips_total",
                Help:      "Total circuit breaker trips (closed to open) by target",
            },
            []string{"target"},
        ),

        circuitBreakerRecoveries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "circuit_breaker_recoveries_total",
                Help:      "Total circuit breaker recoveries (halfopen to closed) by target",
            },
            []string{"target"},
        ),

        workersActive: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "workers_active",
                Help:      "Number of workers currently processing jobs",
            },
        ),

        workersIdle: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "workers_idle",
                Help:      "Number of idle workers waiting for jobs",
            },
        ),

        workerProcessingTime: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "worker_processing_duration_seconds",
                Help:      "Worker processing time per job by worker_id",
                Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
            },
            []string{"worker_id"},
        ),

        dlqSize: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "dlq_size",
                Help:      "Dead letter queue size by target",
            },
            []string{"target"},
        ),

        dlqWrites: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "dlq_writes_total",
                Help:      "Total DLQ writes by target and error_type",
            },
            []string{"target", "error_type"},
        ),

        dlqReplays: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "publishing",
                Name:      "dlq_replays_total",
                Help:      "Total DLQ replays by target and result (success/failure)",
            },
            []string{"target", "result"},
        ),
    }

    // Register all metrics
    registry.MustRegister(
        m.queueSize,
        m.queueCapacityUtil,
        m.queueSubmissions,
        m.jobsProcessed,
        m.jobDuration,
        m.jobWaitTime,
        m.retryAttempts,
        m.retrySuccessRate,
        m.circuitBreakerState,
        m.circuitBreakerTrips,
        m.circuitBreakerRecoveries,
        m.workersActive,
        m.workersIdle,
        m.workerProcessingTime,
        m.dlqSize,
        m.dlqWrites,
        m.dlqReplays,
    )

    return m
}

// Helper methods for metrics recording (RecordQueueSubmission, RecordJobSuccess, etc.)
```

---

## 📊 4. HTTP API Endpoints (6 NEW)

### 4.1 Job Status API

**GET /api/v2/queue/jobs**
- List recent jobs (pagination)
- Query params: `limit`, `offset`, `state`, `priority`, `target`
- Response: Array of JobStatus

**GET /api/v2/queue/jobs/{id}**
- Get single job status by ID
- Response: JobStatus object

**POST /api/v2/queue/jobs/{id}/cancel**
- Cancel queued job (if not yet processing)
- Response: Success/failure

### 4.2 Queue Statistics API

**GET /api/v2/queue/stats**
- Response:
```json
{
  "queue_sizes": {
    "high": 10,
    "medium": 50,
    "low": 20
  },
  "queue_capacities": {
    "high": 500,
    "medium": 1000,
    "low": 500
  },
  "workers": {
    "total": 10,
    "active": 7,
    "idle": 3
  },
  "jobs_processed_last_hour": 5000,
  "dlq_size": 5
}
```

### 4.3 DLQ API

**GET /api/v2/queue/dlq**
- List DLQ jobs (pagination, filtering)
- Query params: `limit`, `offset`, `target`, `error_type`
- Response: Array of DLQEntry

**POST /api/v2/queue/dlq/{id}/replay**
- Replay DLQ job
- Response: Job ID (new submission)

### 4.4 Circuit Breaker Admin API

**GET /api/v2/queue/circuit-breakers**
- List all circuit breaker states
- Response: Array of {target, state, failure_count, last_failure_time}

**POST /api/v2/queue/circuit-breakers/{target}/reset**
- Manual reset circuit breaker to closed
- Response: Success

---

## 📊 5. Performance Optimization Strategies

### 5.1 Queue Latency Optimization

**Target**: <10ms p99 submission latency

**Techniques**:
1. Non-blocking channel sends (select with default)
2. Pre-allocated job pool (sync.Pool for PublishingJob)
3. Lock-free reads where possible (RWMutex)
4. Batch metric updates (buffer 100 samples, flush every 1s)

### 5.2 Throughput Optimization

**Target**: >1,000 jobs/sec (10 workers)

**Techniques**:
1. Worker pool size tuning (10 workers = ~100 jobs/sec per worker)
2. Parallel publisher calls (no mutex in hot path)
3. Circuit breaker fast-path (atomic reads)
4. Priority queue bypass for empty queues

### 5.3 Memory Optimization

**Target**: <500 MB for 1,000 queued jobs

**Techniques**:
1. LRU cache for job tracking (1,000 max, evict oldest)
2. PostgreSQL for DLQ (not in-memory)
3. No job copying (pointers everywhere)
4. Bounded channels (prevent unbounded growth)

---

## 📊 6. Error Handling & Edge Cases

### 6.1 Queue Full Scenario

**Trigger**: All 3 queues reach capacity (500+1000+500 = 2000 jobs)

**Behavior**:
1. Submit() returns error: "queue full"
2. Metric: `queue_submissions_total{result="rejected"}` incremented
3. HTTP 429 returned to webhook sender (back-pressure)
4. Alert: Grafana dashboard triggers "Queue Near Capacity" alert

**Recovery**:
- Workers drain queue faster (dynamic scaling optional)
- Coordinator drops low-priority alerts (grace period)

### 6.2 Circuit Breaker Open

**Trigger**: 5 consecutive failures to target

**Behavior**:
1. Circuit opens, jobs to target skipped for 30s
2. Metric: `circuit_breaker_state{target="X"}` = 2 (open)
3. Jobs NOT sent to DLQ (circuit breaker prevents submission)
4. After 30s timeout, transition to half-open (1 test request)

**Recovery**:
- Health check integration: Circuit closes if health check succeeds
- Manual reset: POST /api/v2/queue/circuit-breakers/{target}/reset

### 6.3 DLQ Overflow

**Trigger**: DLQ table grows beyond 100K entries

**Behavior**:
1. Cleanup worker runs hourly (delete expired entries)
2. Alert triggered: "DLQ Size Critical" (>10K entries)
3. Admin reviews DLQ via API (GET /api/v2/queue/dlq)

**Recovery**:
- Fix downstream service (e.g., Rootly API back online)
- Replay DLQ jobs via API (POST /dlq/{id}/replay)

---

## 📊 7. Testing Strategy

### 7.1 Unit Tests (50+ tests)

**Categories**:
1. **Queue Operations** (15 tests):
   - Submit to each priority queue
   - Queue full rejection
   - Graceful shutdown
2. **Retry Logic** (10 tests):
   - Exponential backoff calculation
   - Error classification (transient/permanent)
   - Jitter randomness
3. **Circuit Breaker** (10 tests):
   - State transitions (closed → open → halfopen → closed)
   - Threshold triggering
   - Timeout recovery
4. **DLQ Operations** (10 tests):
   - Write to DLQ
   - List with filters
   - Replay job
   - Cleanup expired
5. **Metrics** (10 tests):
   - All 12+ metrics update correctly
   - No registration conflicts
6. **Job Tracking** (5 tests):
   - Store/get/list jobs
   - LRU eviction
   - TTL expiration

### 7.2 Integration Tests (5 tests)

1. **End-to-End Happy Path**:
   - Submit job → worker picks up → publisher called → success
2. **Retry Success**:
   - Submit job → publisher fails (transient) → retry → success
3. **DLQ Flow**:
   - Submit job → all retries fail → job written to DLQ
4. **Circuit Breaker**:
   - 5 failures → circuit opens → 30s timeout → half-open → 2 successes → closed
5. **Priority Ordering**:
   - Submit 10 LOW + 5 HIGH → verify HIGH processed first

### 7.3 Benchmarks (10+ benchmarks)

1. BenchmarkQueueSubmit (target: <1µs)
2. BenchmarkWorkerProcess (target: <10ms)
3. BenchmarkRetryBackoffCalculation (target: <100ns)
4. BenchmarkCircuitBreakerCheck (target: <100µs)
5. BenchmarkErrorClassifier (target: <1µs)
6. BenchmarkJobTrackingStore (target: <10µs)
7. BenchmarkDLQWrite (target: <5ms)
8. BenchmarkMetricsUpdate (target: <1µs)
9. BenchmarkPriorityDetermination (target: <100ns)
10. BenchmarkConcurrentSubmit (1000 goroutines, target: <100ms total)

### 7.4 Chaos Testing

**Scenarios**:
1. **Random Publisher Failures**: 30% failure rate → verify circuit breakers open
2. **Network Partitions**: Simulate timeout errors → verify transient error handling
3. **Queue Overflow**: Submit 10K jobs in 1 second → verify back-pressure
4. **Worker Panics**: Force panic in worker → verify recovery (defer/recover)

---

## 📊 8. Deployment & Operations

### 8.1 Configuration (Environment Variables)

```bash
# Worker Pool
PUBLISHING_WORKER_COUNT=10

# Queue Capacities
PUBLISHING_HIGH_QUEUE_SIZE=500
PUBLISHING_MEDIUM_QUEUE_SIZE=1000
PUBLISHING_LOW_QUEUE_SIZE=500

# Retry Configuration
PUBLISHING_MAX_RETRIES=3
PUBLISHING_INITIAL_RETRY_DELAY=100ms
PUBLISHING_MAX_RETRY_DELAY=30s
PUBLISHING_RETRY_GROWTH_FACTOR=2.0
PUBLISHING_RETRY_JITTER=0.1

# Circuit Breaker
PUBLISHING_CB_FAILURE_THRESHOLD=5
PUBLISHING_CB_SUCCESS_THRESHOLD=2
PUBLISHING_CB_TIMEOUT=30s

# Job Tracking
PUBLISHING_JOB_TRACKING_ENABLED=true
PUBLISHING_JOB_TRACKING_CAPACITY=1000
PUBLISHING_JOB_TRACKING_TTL=1h

# DLQ
PUBLISHING_DLQ_ENABLED=true
PUBLISHING_DLQ_TTL=168h  # 7 days
```

### 8.2 Grafana Dashboard Panels

**Panel 1: Queue Health**
- Queue sizes (stacked area chart)
- Queue capacity utilization (gauge)

**Panel 2: Job Processing**
- Jobs processed per minute (rate)
- Job duration (p50/p95/p99 heatmap)

**Panel 3: Retry Metrics**
- Retry attempts by error type (stacked bar)
- Retry success rate trend

**Panel 4: Circuit Breakers**
- Circuit breaker states (stat panel per target)
- Circuit breaker trips/recoveries (rate)

**Panel 5: Worker Pool**
- Active/idle workers (gauge)
- Worker processing time (histogram)

**Panel 6: DLQ**
- DLQ size trend (line chart)
- DLQ writes/replays (rate)

### 8.3 AlertManager Rules

```yaml
groups:
  - name: publishing_queue
    rules:
      # Queue near capacity
      - alert: PublishingQueueNearCapacity
        expr: alert_history_publishing_queue_capacity_utilization{priority="high"} > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High-priority queue near capacity"
          description: "Queue utilization {{ $value }} > 80%"

      # Circuit breaker open
      - alert: PublishingCircuitBreakerOpen
        expr: alert_history_publishing_circuit_breaker_state > 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Circuit breaker open for {{ $labels.target }}"
          description: "Circuit breaker has been open for 2 minutes"

      # DLQ growing
      - alert: PublishingDLQGrowing
        expr: alert_history_publishing_dlq_size > 1000
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "DLQ size critical ({{ $value }} jobs)"
          description: "DLQ has over 1000 failed jobs"
```

---

## 📊 9. Future Enhancements (Beyond 150%)

### 9.1 Dynamic Worker Scaling

**Goal**: Auto-scale workers based on queue depth

**Implementation**:
- Scale up: Add 5 workers if queue >80% full for 1 min
- Scale down: Remove 5 workers if queue <20% full for 5 min
- Min: 5 workers, Max: 50 workers

### 9.2 Redis-backed DLQ

**Goal**: Faster DLQ for high-throughput scenarios

**Implementation**:
- Use Redis Lists for DLQ (sub-ms writes)
- Background worker syncs to PostgreSQL every 1 min
- Best of both: speed + persistence

### 9.3 Distributed Queue (Multi-Replica)

**Goal**: Share queue across 10 replicas (Kubernetes HPA)

**Implementation**:
- Redis-backed distributed queue (RPOPLPUSH pattern)
- Leader election (Raft/etcd) for DLQ cleanup worker
- Prometheus metrics aggregation

---

## 📊 10. References & Dependencies

### 10.1 Internal Dependencies (ALL ✅ COMPLETE)

| Task | Files | Notes |
|------|-------|-------|
| TN-051 | formatter.go, formatter_metrics.go | Alert formatting (5 formats) |
| TN-052 | rootly_publisher_enhanced.go | Rootly incident creation |
| TN-053 | pagerduty_publisher_enhanced.go | PagerDuty Events API v2 |
| TN-054 | slack_publisher_enhanced.go | Slack webhook with threading |
| TN-055 | webhook_*.go | Generic webhook (4 auth strategies) |

### 10.2 External Libraries

```go
import (
    "github.com/google/uuid"                     // Job ID generation
    "github.com/hashicorp/golang-lru/v2"         // LRU cache for job tracking
    "github.com/prometheus/client_golang/prometheus" // Metrics
    "database/sql"                               // PostgreSQL DLQ
    _ "github.com/lib/pq"                        // PostgreSQL driver
)
```

---

**Дата последнего обновления**: 2025-11-12
**Автор**: AI Assistant
**Статус**: ✅ READY FOR IMPLEMENTATION
