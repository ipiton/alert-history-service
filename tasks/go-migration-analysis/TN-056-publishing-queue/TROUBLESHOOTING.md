# TN-056: Publishing Queue Troubleshooting Guide

**Version**: 1.0
**Date**: 2025-11-12
**Status**: Ready for Use
**Author**: AI Assistant

---

## 📋 Table of Contents

1. [Common Issues](#1-common-issues)
2. [Debugging](#2-debugging)
3. [Performance Issues](#3-performance-issues)
4. [DLQ Investigation](#4-dlq-investigation)
5. [Circuit Breaker](#5-circuit-breaker)
6. [Metrics Interpretation](#6-metrics-interpretation)
7. [Log Analysis](#7-log-analysis)
8. [FAQ](#8-faq)

---

## 1. Common Issues

### 1.1 Issue: Queue is Full (Submit Fails)

**Symptoms**:
```
ERROR Failed to submit job error="queue is full"
```

**Root Cause**:
- Workers cannot keep up with job submission rate
- External system is slow or down (circuit breaker open)
- Too many retries clogging the queue

**Solution**:
```go
// Option 1: Increase worker count
config.WorkerCount = 20 // Double workers

// Option 2: Increase queue capacity
config.HighPriorityQueueSize = 2000
config.MediumPriorityQueueSize = 10000

// Option 3: Check circuit breaker states
for target, cb := range queue.circuitBreakers {
    logger.Info("Circuit breaker", "target", target, "state", cb.state)
}
```

**Prevention**:
- Monitor queue size: Alert if > 80% capacity
- Scale workers based on load
- Fix external system issues quickly

---

### 1.2 Issue: High DLQ Entry Rate

**Symptoms**:
```
publishing_dlq_writes_total increasing rapidly
```

**Root Cause**:
- External system returns permanent errors (401, 404)
- Invalid credentials
- Malformed alert format

**Solution**:
```bash
# 1. Query DLQ for error patterns
curl "http://localhost:8080/api/v2/publishing/dlq?limit=100" | jq '.entries[] | {target, error_message, error_type}'

# 2. Check for 401 errors (invalid credentials)
curl "http://localhost:8080/api/v2/publishing/dlq" | jq '.entries[] | select(.error_message | contains("401"))'

# 3. Fix root cause (e.g., update credentials in K8s Secret)
kubectl edit secret production-pagerduty-secret

# 4. Replay failed jobs
for id in $(curl "http://localhost:8080/api/v2/publishing/dlq?limit=100" | jq -r '.entries[].id'); do
    curl -X POST "http://localhost:8080/api/v2/publishing/dlq/$id/replay"
done
```

**Prevention**:
- Validate credentials before deployment
- Test alert format in staging
- Monitor DLQ entry rate (alert > 0.1%)

---

### 1.3 Issue: Circuit Breaker Stuck Open

**Symptoms**:
```
WARN Circuit breaker open target=production-pagerduty
publishing_circuit_breaker_state{target="production-pagerduty",state="open"} 1
```

**Root Cause**:
- External system is down for > 30s
- Network connectivity issues
- External system timeout

**Solution**:
```go
// Option 1: Check external system health
curl https://events.pagerduty.com/v2/health
curl https://hooks.slack.com/services/health

// Option 2: Reset circuit breaker manually (if external system is back)
cb := queue.getCircuitBreaker("production-pagerduty")
cb.Reset()

// Option 3: Increase circuit breaker timeout
config.CircuitTimeout = 1 * time.Minute // Give more time to recover
```

**Prevention**:
- Monitor external system uptime
- Set up alerts for circuit breaker state changes
- Configure appropriate timeouts based on SLA

---

### 1.4 Issue: Jobs Stuck in RETRYING State

**Symptoms**:
```
publishing_queue_size{state="retrying"} increasing
```

**Root Cause**:
- Transient errors (429, 503) persisting
- External system rate limiting
- Network issues

**Solution**:
```bash
# 1. Check job states
curl "http://localhost:8080/api/v2/publishing/jobs?state=retrying&limit=10"

# 2. Identify error types
curl "http://localhost:8080/api/v2/publishing/jobs?state=retrying" | jq '.jobs[] | {target, error_type, retry_count}'

# 3. If rate limited (429), slow down submission rate
# 4. If network issues (timeout), check connectivity
ping events.pagerduty.com
```

**Prevention**:
- Implement backpressure at submission layer
- Monitor retry rate (alert if > 10% of submissions)
- Configure appropriate retry intervals

---

### 1.5 Issue: Memory Leak (Job Tracking)

**Symptoms**:
```
Memory usage increasing over time
publishing_job_tracking_cache_size growing unbounded
```

**Root Cause**:
- Job tracking cache not evicting old entries
- LRU capacity set too high
- Jobs not transitioning to terminal states

**Solution**:
```go
// Option 1: Reduce job tracking capacity
config.JobTrackingCapacity = 5000 // Halve capacity

// Option 2: Clear job tracking manually
jobTracking.Clear()

// Option 3: Check job state distribution
states := make(map[string]int)
jobs := jobTracking.List(publishing.JobFilters{Limit: 10000})
for _, job := range jobs {
    states[job.State]++
}
// Expect: most jobs in "succeeded" or "failed" states
```

**Prevention**:
- Set job tracking capacity appropriately (10k default)
- Monitor cache size
- Ensure jobs reach terminal states

---

### 1.6 Issue: Slow Publishing Latency

**Symptoms**:
```
publishing_queue_submission_duration_seconds p95 > 1s
```

**Root Cause**:
- External system slow response times
- Network latency
- Too few workers

**Solution**:
```bash
# 1. Check external system latency
curl -w "@curl-format.txt" -o /dev/null -s https://events.pagerduty.com/v2/enqueue

# 2. Increase worker count
config.WorkerCount = 20

# 3. Use priority queues for critical alerts
# (Critical alerts automatically get PriorityHigh)
```

**Prevention**:
- Monitor publishing latency (alert p95 > 2s)
- Profile external system response times
- Scale workers based on load

---

## 2. Debugging

### 2.1 Enable Debug Logging

```go
// Set log level to DEBUG
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

queue := publishing.NewPublishingQueue(config, factory, dlqRepo, jobTracking, metrics, logger)
```

**Debug Logs**:
```
DEBUG Worker started id=1
DEBUG Job submitted id=123e4567 priority=high target=production-pagerduty
DEBUG Job picked by worker id=1 job_id=123e4567
DEBUG Publishing attempt job_id=123e4567 attempt=0
DEBUG Retry backoff job_id=123e4567 attempt=1 backoff=200ms
```

### 2.2 Inspect Queue State

```bash
# Get queue size by priority
curl http://localhost:8080/api/v2/publishing/queue/status

# Get recent jobs
curl "http://localhost:8080/api/v2/publishing/jobs?limit=100"

# Filter by state
curl "http://localhost:8080/api/v2/publishing/jobs?state=failed&limit=10"

# Filter by target
curl "http://localhost:8080/api/v2/publishing/jobs?target=production-pagerduty&limit=10"
```

### 2.3 Trace Job Lifecycle

```go
// 1. Submit job
err := queue.Submit(enrichedAlert, target)

// 2. Check job ID (from logs or job tracking)
jobs := jobTracking.List(publishing.JobFilters{
    TargetName: target.Name,
    Limit:      1,
})
jobID := jobs[0].ID

// 3. Poll job status
for {
    job := jobTracking.Get(jobID)
    logger.Info("Job status", "id", jobID, "state", job.State, "retry_count", job.RetryCount)

    if job.State == "succeeded" || job.State == "failed" || job.State == "dlq" {
        break
    }

    time.Sleep(1 * time.Second)
}
```

### 2.4 PostgreSQL DLQ Query

```sql
-- Get recent DLQ entries
SELECT
    id,
    job_id,
    fingerprint,
    target_name,
    error_message,
    error_type,
    retry_count,
    failed_at
FROM publishing_dlq
ORDER BY failed_at DESC
LIMIT 100;

-- Count by target
SELECT
    target_name,
    COUNT(*) as entry_count
FROM publishing_dlq
GROUP BY target_name
ORDER BY entry_count DESC;

-- Count by error type
SELECT
    error_type,
    COUNT(*) as entry_count
FROM publishing_dlq
GROUP BY error_type;
```

---

## 3. Performance Issues

### 3.1 Issue: Low Throughput

**Symptoms**:
- Queue size growing faster than workers can process
- Publishing rate < expected

**Diagnosis**:
```promql
# Current publishing rate (jobs/sec)
rate(publishing_successes_total[1m])

# Expected: > 100 jobs/sec with 10 workers
```

**Solution**:
```go
// Option 1: Increase worker count
config.WorkerCount = 20 // 2x throughput

// Option 2: Reduce retry interval for faster retries
config.RetryInterval = 50 * time.Millisecond

// Option 3: Profile hot paths
import _ "net/http/pprof"
go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
// Then: go tool pprof http://localhost:6060/debug/pprof/profile
```

### 3.2 Issue: High Memory Usage

**Diagnosis**:
```bash
# Check memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Top allocations
(pprof) top10
```

**Solution**:
```go
// Option 1: Reduce queue capacities
config.HighPriorityQueueSize = 500
config.MediumPriorityQueueSize = 2000
config.LowPriorityQueueSize = 5000

// Option 2: Reduce job tracking capacity
config.JobTrackingCapacity = 5000

// Option 3: Increase GC frequency (last resort)
debug.SetGCPercent(50) // Default: 100
```

### 3.3 Issue: High CPU Usage

**Diagnosis**:
```bash
# Check CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Top CPU consumers
(pprof) top10
```

**Solution**:
```go
// Option 1: Reduce worker count if over-saturating CPU
config.WorkerCount = runtime.NumCPU() * 2

// Option 2: Increase retry interval to reduce retry overhead
config.RetryInterval = 200 * time.Millisecond
```

---

## 4. DLQ Investigation

### 4.1 Identify Root Cause of Failures

```bash
# Step 1: Query DLQ
curl "http://localhost:8080/api/v2/publishing/dlq?limit=100" > dlq.json

# Step 2: Group by error message
cat dlq.json | jq '.entries[] | .error_message' | sort | uniq -c | sort -rn

# Step 3: Group by target
cat dlq.json | jq '.entries[] | .target_name' | sort | uniq -c | sort -rn

# Step 4: Group by error type
cat dlq.json | jq '.entries[] | .error_type' | sort | uniq -c | sort -rn
```

### 4.2 Fix Common DLQ Issues

**401 Unauthorized**:
```bash
# Update credentials in K8s Secret
kubectl edit secret production-pagerduty-secret

# Verify credentials
curl -H "Authorization: Bearer <routing_key>" https://events.pagerduty.com/v2/enqueue
```

**404 Not Found**:
```bash
# Verify webhook URL
curl -I https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

**400 Bad Request**:
```go
// Check alert format
formatter := publishing.NewFormatter()
formatted, err := formatter.Format(enrichedAlert, "pagerduty")
logger.Info("Formatted alert", "payload", formatted)

// Validate against external API spec
```

### 4.3 Replay DLQ Entries

```bash
# Replay single entry
curl -X POST http://localhost:8080/api/v2/publishing/dlq/550e8400-e29b-41d4-a716-446655440000/replay

# Replay all entries for a target (after fixing root cause)
for id in $(curl "http://localhost:8080/api/v2/publishing/dlq?target=production-pagerduty" | jq -r '.entries[].id'); do
    curl -X POST "http://localhost:8080/api/v2/publishing/dlq/$id/replay"
    sleep 0.1 # Rate limit to avoid overwhelming queue
done
```

---

## 5. Circuit Breaker

### 5.1 Monitor Circuit Breaker States

```promql
# Count circuit breakers by state
count by (state) (publishing_circuit_breaker_state)

# Targets with open circuit breakers
publishing_circuit_breaker_state{state="open"}

# Circuit breaker trip rate
rate(publishing_circuit_breaker_trips_total[5m])
```

### 5.2 Reset Circuit Breaker

```go
// Manual reset (use with caution!)
cb := queue.getCircuitBreaker("production-pagerduty")
cb.Reset()
logger.Info("Circuit breaker reset", "target", "production-pagerduty")
```

### 5.3 Tune Circuit Breaker

```go
// More lenient (allow more failures before opening)
config.CircuitFailureThreshold = 10 // Default: 5

// Longer timeout (give more time to recover)
config.CircuitTimeout = 1 * time.Minute // Default: 30s

// More strict recovery (require more successes to close)
config.CircuitSuccessThreshold = 5 // Default: 2
```

---

## 6. Metrics Interpretation

### 6.1 Publishing Success Rate < 99%

**Meaning**: Too many publishing failures

**Investigation**:
```promql
# Success rate by target
rate(publishing_successes_total[5m]) /
(rate(publishing_successes_total[5m]) + rate(publishing_failures_total[5m]))

# Failure rate by target
rate(publishing_failures_total{target="production-pagerduty"}[5m])
```

**Action**:
- Check DLQ for error patterns
- Check circuit breaker states
- Check external system health

### 6.2 Queue Size > 80% Capacity

**Meaning**: Backpressure building up

**Investigation**:
```promql
# Queue size vs capacity by priority
publishing_queue_size{priority="high"} / publishing_queue_capacity{priority="high"}
```

**Action**:
- Increase worker count
- Increase queue capacity
- Check external system latency

### 6.3 Retry Rate > 10%

**Meaning**: High transient error rate

**Investigation**:
```promql
# Retry rate by error type
rate(publishing_retry_attempts_total[5m])

# By target
rate(publishing_retry_attempts_total{target="production-pagerduty"}[5m])
```

**Action**:
- Check external system rate limits (429)
- Check network latency
- Check external system availability (503)

### 6.4 DLQ Entry Rate > 0.1%

**Meaning**: Permanent errors occurring

**Investigation**:
```promql
# DLQ entry rate
rate(publishing_dlq_writes_total[5m])

# By target
rate(publishing_dlq_writes_total{target="production-pagerduty"}[5m])

# By error type
rate(publishing_dlq_writes_total{error_type="permanent"}[5m])
```

**Action**:
- Query DLQ for error patterns
- Fix root cause (credentials, URL, format)
- Replay failed jobs

---

## 7. Log Analysis

### 7.1 Log Levels

| Level | Usage | Example |
|-------|-------|---------|
| DEBUG | Detailed flow | `DEBUG Worker picked job id=123` |
| INFO | Normal operations | `INFO Job submitted id=123 priority=high` |
| WARN | Recoverable issues | `WARN Circuit breaker open target=production-pagerduty` |
| ERROR | Failures | `ERROR Publishing failed job_id=123 error="401 Unauthorized"` |

### 7.2 Common Log Patterns

**Successful Publishing**:
```
INFO Job submitted id=123e4567 priority=high target=production-pagerduty
DEBUG Worker picked job id=123e4567
DEBUG Publishing attempt job_id=123e4567 attempt=0
INFO Job succeeded id=123e4567 duration=150ms
```

**Retry Flow**:
```
INFO Job submitted id=123e4567
ERROR Publishing failed job_id=123e4567 error="connection timeout" error_type=transient
DEBUG Retry backoff job_id=123e4567 attempt=1 backoff=200ms
INFO Job succeeded id=123e4567 retry_count=1
```

**DLQ Flow**:
```
INFO Job submitted id=123e4567
ERROR Publishing failed job_id=123e4567 error="401 Unauthorized" error_type=permanent
ERROR Job sent to DLQ job_id=123e4567 fingerprint=abc123
```

### 7.3 Grep for Errors

```bash
# All errors
grep "ERROR" app.log | tail -100

# Publishing errors
grep "Publishing failed" app.log | tail -100

# DLQ writes
grep "Job sent to DLQ" app.log | tail -100

# Circuit breaker events
grep "Circuit breaker" app.log | tail -100
```

---

## 8. FAQ

### 8.1 How do I increase throughput?

**Answer**: Increase worker count and queue capacities:
```go
config.WorkerCount = 20 // 2x workers
config.HighPriorityQueueSize = 2000
config.MediumPriorityQueueSize = 10000
```

### 8.2 How do I prioritize critical alerts?

**Answer**: Critical alerts are automatically assigned `PriorityHigh`:
```go
// Severity "critical" → PriorityHigh
alert.Labels["severity"] = "critical"

// Or LLM classification → PriorityHigh
enrichedAlert.Classification.Severity = core.SeverityCritical
```

### 8.3 What happens if the queue is full?

**Answer**: `Submit()` returns an error:
```go
err := queue.Submit(enrichedAlert, target)
if err != nil {
    // Handle: log error, return 503 to caller, backpressure
}
```

### 8.4 How do I disable retries for a specific target?

**Answer**: Create a custom retry config:
```go
config.MaxRetries = 0 // No retries
// Or classify errors as permanent:
// Custom error classifier in publisher implementation
```

### 8.5 Can I replay all DLQ entries at once?

**Answer**: Yes, but rate-limit to avoid overwhelming the queue:
```bash
for id in $(curl http://localhost:8080/api/v2/publishing/dlq | jq -r '.entries[].id'); do
    curl -X POST http://localhost:8080/api/v2/publishing/dlq/$id/replay
    sleep 0.1 # 10 jobs/sec
done
```

### 8.6 How do I monitor the queue in production?

**Answer**: Use Prometheus + Grafana:
- Alert: Queue size > 80% capacity
- Alert: Publishing success rate < 99%
- Alert: DLQ entry rate > 0.1%
- Alert: Circuit breaker open > 5 min

### 8.7 What is the maximum retry backoff?

**Answer**: 30 seconds by default:
```go
config.MaxBackoff = 30 * time.Second
// Exponential: 100ms, 200ms, 400ms, 800ms, 1.6s, 3.2s, 6.4s, 12.8s, 25.6s, 30s (capped)
```

### 8.8 How do I test the queue locally?

**Answer**:
```go
// Use mock publisher
mockPublisher := &MockPublisher{}
factory := publishing.NewPublisherFactory(mockPublisher)

// Create queue with in-memory DLQ (no PostgreSQL)
dlqRepo := publishing.NewInMemoryDLQRepository()

// Submit test jobs
queue.Submit(testAlert, testTarget)
```

### 8.9 Can I change queue configuration at runtime?

**Answer**: No. Configuration is set at initialization. To change:
1. Update environment variables
2. Restart application
3. New config takes effect

### 8.10 What happens during graceful shutdown?

**Answer**:
1. Stop accepting new jobs (`Submit()` returns error)
2. Wait for workers to finish current jobs (30s timeout by default)
3. Cancel all pending jobs
4. Close connections
5. Return error if timeout exceeded

### 8.11 How do I debug a stuck job?

**Answer**:
```bash
# 1. Find job ID
curl http://localhost:8080/api/v2/publishing/jobs?state=processing

# 2. Check logs
grep "job_id=123e4567" app.log | tail -50

# 3. Check metrics
curl http://localhost:8080/metrics | grep "job_id"

# 4. Check circuit breaker
curl http://localhost:8080/api/v2/publishing/queue/stats
```

### 8.12 How long are DLQ entries retained?

**Answer**: Indefinitely by default. Set up periodic purge:
```bash
# Cron job: purge entries older than 7 days
0 0 * * 0 curl -X DELETE "http://localhost:8080/api/v2/publishing/dlq/purge?older_than=168h"
```

### 8.13 Can I use the queue without PostgreSQL?

**Answer**: No. PostgreSQL is required for DLQ persistence. Alternative:
- Use in-memory DLQ (data loss on restart)
- Implement custom `DLQRepository` (e.g., Redis, S3)

### 8.14 How do I handle rate limiting (429)?

**Answer**: Automatic retry with exponential backoff:
```
Attempt 0: 100ms + jitter → 429 → Retry
Attempt 1: 200ms + jitter → 429 → Retry
Attempt 2: 400ms + jitter → 429 → Retry
Attempt 3: 800ms + jitter → Success
```

### 8.15 What is the performance overhead of job tracking?

**Answer**: Minimal:
- Add: 265 ns/op
- Get: 82 ns/op
- Memory: ~100 bytes/job × 10,000 = ~1 MB

---

## 📚 Additional Resources

- **requirements.md**: Functional and non-functional requirements
- **design.md**: Architecture and implementation details
- **tasks.md**: Implementation checklist
- **API_GUIDE.md**: Usage examples and best practices

---

**Document Status**: ✅ COMPLETE
**Last Updated**: 2025-11-12
**Support**: For production issues, escalate to Platform Team
