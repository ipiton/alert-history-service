# TN-058: Troubleshooting Guide

**Status**: ✅ Production-Ready
**Version**: 1.0.0
**Last Updated**: 2025-11-13

## Table of Contents

1. [Common Issues](#common-issues)
2. [Error Messages](#error-messages)
3. [Performance Issues](#performance-issues)
4. [Health Check Issues](#health-check-issues)
5. [Configuration Issues](#configuration-issues)
6. [Debugging Tips](#debugging-tips)
7. [Monitoring & Alerts](#monitoring--alerts)

---

## Common Issues

### Issue 1: All Targets Failed

**Symptoms**:
- Error: `all targets failed`
- Metrics: `parallel_publish_failure_total` increasing
- Logs: Multiple target failures

**Possible Causes**:

1. **Network Issues**:
   - DNS resolution failure
   - Network partition
   - Firewall blocking outbound connections

2. **Target Configuration**:
   - Invalid API keys
   - Expired credentials
   - Incorrect webhook URLs

3. **Target Service Outage**:
   - Rootly/PagerDuty/Slack down
   - Target returning 5xx errors

**Diagnostic Steps**:

```bash
# Check target health
curl -s http://localhost:8080/api/v1/health/targets | jq '.'

# Check recent publishing errors
curl -s http://localhost:8080/api/v1/publishing/stats | jq '.errors'

# Check Prometheus metrics
curl -s http://localhost:9090/metrics | grep parallel_publish_failure
```

**Solutions**:

1. **Verify Target Configuration**:
```go
// Check if targets are enabled
targets := discoveryMgr.ListTargets()
for _, target := range targets {
    log.Info("Target", "name", target.Name, "enabled", target.Enabled)
}
```

2. **Test Individual Targets**:
```bash
# Test Rootly
curl -X POST "https://api.rootly.com/v1/incidents" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"title": "test"}'

# Test PagerDuty
curl -X POST "https://events.pagerduty.com/v2/enqueue" \
  -H "Content-Type: application/json" \
  -d '{"routing_key": "YOUR_KEY", "event_action": "trigger", "payload": {"summary": "test", "severity": "info", "source": "test"}}'

# Test Slack
curl -X POST "YOUR_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{"text": "test"}'
```

3. **Check Circuit Breakers**:
```bash
# Check if circuit breakers are open
curl -s http://localhost:8080/api/v1/health/targets | jq '.[] | select(.circuit_state == "open")'
```

4. **Temporary Workaround**: Disable health checks
```go
opts := &ParallelPublishOptions{
    CheckHealth: false, // Temporarily disable
    // ... other options
}
```

---

### Issue 2: Partial Success (Some Targets Failed)

**Symptoms**:
- Result: `IsPartialSuccess = true`
- Metrics: `parallel_publish_partial_success_total` increasing
- Logs: Some targets succeeded, some failed

**Possible Causes**:

1. **Degraded Targets**: Some targets are slow or intermittently failing
2. **Rate Limiting**: Some targets rejecting requests due to rate limits
3. **Target-Specific Issues**: Configuration valid for some targets, invalid for others

**Diagnostic Steps**:

```go
// Check which targets failed
result, err := publisher.PublishToMultiple(ctx, alert, targets)
if result != nil && result.IsPartialSuccess {
    for _, tr := range result.Results {
        if !tr.Success && !tr.Skipped {
            log.Error("Target failed",
                "target_name", tr.TargetName,
                "error", tr.Error,
                "status_code", tr.StatusCode,
            )
        }
    }
}
```

**Solutions**:

1. **Investigate Failed Targets**:
```bash
# Check target health history
curl -s http://localhost:8080/api/v1/health/targets/rootly-prod | jq '.failure_history'
```

2. **Adjust Health Check Strategy**:
```go
opts := &ParallelPublishOptions{
    HealthStrategy: SkipUnhealthyAndDegraded, // Skip degraded targets
}
```

3. **Increase Timeout** (if targets are slow):
```go
opts := &ParallelPublishOptions{
    Timeout: 60 * time.Second, // Increase from 30s
}
```

---

### Issue 3: Timeout Exceeded

**Symptoms**:
- Error: `context deadline exceeded`
- Metrics: `parallel_publish_duration_seconds` histogram showing high p99
- Logs: "Publishing timed out"

**Possible Causes**:

1. **Slow Targets**: One or more targets taking too long
2. **Network Latency**: High network latency to targets
3. **Too Many Targets**: Timeout too short for large target count

**Diagnostic Steps**:

```go
// Log per-target durations
for _, tr := range result.Results {
    log.Info("Target duration",
        "target_name", tr.TargetName,
        "duration_ms", tr.Duration.Milliseconds(),
    )
}
```

**Solutions**:

1. **Increase Timeout**:
```go
opts := &ParallelPublishOptions{
    Timeout: 60 * time.Second, // or 120s for very large target counts
}
```

2. **Skip Slow Targets**:
```go
// Use health-aware publishing
result, err := publisher.PublishToHealthy(ctx, alert)
```

3. **Increase Concurrency**:
```go
opts := &ParallelPublishOptions{
    MaxConcurrent: 200, // Allow more parallel publishes
}
```

---

### Issue 4: No Healthy Targets

**Symptoms**:
- Error: `no healthy targets available`
- Metrics: All targets marked unhealthy
- Logs: "All targets skipped due to health checks"

**Possible Causes**:

1. **All Targets Unhealthy**: Legitimate outage of all targets
2. **Overly Aggressive Health Checks**: Health check threshold too strict
3. **Configuration Issue**: Health monitor not working correctly

**Diagnostic Steps**:

```bash
# Check all target health
curl -s http://localhost:8080/api/v1/health/targets | jq '.[] | {name: .target_name, status: .status, consecutive_failures: .consecutive_failures}'
```

**Solutions**:

1. **Temporarily Disable Health Checks**:
```go
// For testing/emergency
result, err := publisher.PublishToAll(ctx, alert) // Ignores health
```

2. **Adjust Health Check Strategy**:
```go
opts := &ParallelPublishOptions{
    HealthStrategy: SkipUnhealthy, // Only skip truly unhealthy (not degraded)
}
```

3. **Reset Circuit Breakers** (manual intervention):
```bash
# Reset circuit breakers for all targets
curl -X POST http://localhost:8080/api/v1/health/targets/reset-circuit-breakers
```

---

### Issue 5: High Memory Usage

**Symptoms**:
- Memory usage growing over time
- OOMKilled in Kubernetes
- Metrics: `go_memstats_heap_alloc_bytes` increasing

**Possible Causes**:

1. **Memory Leak**: Goroutine or channel leak
2. **Too Many Concurrent Publishes**: `MaxConcurrent` too high
3. **Large Alert Payloads**: Alerts with very large data

**Diagnostic Steps**:

```bash
# Check goroutine count
curl -s http://localhost:8080/debug/pprof/goroutine?debug=2

# Check memory profile
curl -s http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Solutions**:

1. **Reduce Concurrency**:
```go
opts := &ParallelPublishOptions{
    MaxConcurrent: 50, // Reduce from 200
}
```

2. **Check for Goroutine Leaks**:
```go
// Add metrics tracking
go func() {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        log.Info("Goroutines", "count", runtime.NumGoroutine())
    }
}()
```

3. **Limit Alert Payload Size**:
```go
// Truncate large fields
if len(alert.Annotations) > 1000 {
    alert.Annotations = alert.Annotations[:1000] + "...(truncated)"
}
```

---

## Error Messages

### `ErrInvalidInput`

**Message**: `invalid input`

**Cause**: Alert is nil or targets list is empty

**Solution**:
```go
if alert == nil {
    return fmt.Errorf("alert is nil")
}
if len(targets) == 0 {
    return fmt.Errorf("targets list is empty")
}
```

### `ErrAllTargetsFailed`

**Message**: `all targets failed`

**Cause**: Every target failed (success_count = 0)

**Solution**: See [Issue 1: All Targets Failed](#issue-1-all-targets-failed)

### `ErrContextTimeout`

**Message**: `context deadline exceeded`

**Cause**: Publishing took longer than timeout

**Solution**: See [Issue 3: Timeout Exceeded](#issue-3-timeout-exceeded)

### `ErrNoHealthyTargets`

**Message**: `no healthy targets available`

**Cause**: All targets are unhealthy and health checks are enabled

**Solution**: See [Issue 4: No Healthy Targets](#issue-4-no-healthy-targets)

### `ErrNoEnabledTargets`

**Message**: `no enabled targets found`

**Cause**: All targets have `Enabled = false`

**Solution**:
```bash
# Check target configuration
kubectl get secrets -l type=publishing-target -o json | jq '.items[] | {name: .metadata.name, enabled: .data.enabled | @base64d}'
```

---

## Performance Issues

### High Latency (> 100ms)

**Symptoms**:
- `parallel_publish_duration_seconds` p99 > 100ms
- Slow publishing times

**Diagnostic Steps**:

```bash
# Check latency distribution
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile(0.99,parallel_publish_duration_seconds_bucket)

# Check per-target latencies
for target in $(curl -s http://localhost:8080/api/v1/targets | jq -r '.[].name'); do
    echo "$target: $(curl -s http://localhost:8080/api/v1/health/targets/$target | jq '.average_response_time_ms')ms"
done
```

**Solutions**:

1. **Identify Slow Targets**: Log per-target durations, disable slow targets
2. **Increase Concurrency**: Allow more parallel publishes
3. **Optimize Network**: Use regional endpoints, reduce network hops

### Low Throughput (< 100 targets/s)

**Symptoms**:
- Cannot publish to many targets quickly
- Queue backlog growing

**Diagnostic Steps**:

```bash
# Check publishing rate
rate(parallel_publish_total[1m])

# Check goroutine count
parallel_publish_active_goroutines
```

**Solutions**:

1. **Increase `MaxConcurrent`**:
```go
opts := &ParallelPublishOptions{
    MaxConcurrent: 200, // or higher
}
```

2. **Batch Alerts**: Group similar alerts together
3. **Scale Horizontally**: Deploy multiple publisher instances

---

## Health Check Issues

### False Positives (Healthy Targets Marked Unhealthy)

**Symptoms**:
- Targets marked unhealthy but actually working
- `SkippedCount` high even though targets are healthy

**Diagnostic Steps**:

```bash
# Check false positive rate
curl -s http://localhost:8080/api/v1/health/targets | jq '.[] | select(.status == "unhealthy") | {name: .target_name, last_error: .last_error}'
```

**Solutions**:

1. **Adjust Health Check Threshold**:
```go
// Increase consecutive failure threshold
healthMon.SetThreshold(5) // Default: 3
```

2. **Disable Health Checks Temporarily**:
```go
opts := &ParallelPublishOptions{
    CheckHealth: false,
}
```

### False Negatives (Unhealthy Targets Not Detected)

**Symptoms**:
- Publishing to unhealthy targets
- High failure rate despite health checks

**Diagnostic Steps**:

```bash
# Check health check effectiveness
curl -s http://localhost:8080/api/v1/health/stats | jq '.detection_rate'
```

**Solutions**:

1. **Adjust Health Check Strategy**:
```go
opts := &ParallelPublishOptions{
    HealthStrategy: SkipUnhealthyAndDegraded, // More aggressive
}
```

2. **Reduce Health Check Interval**: Check more frequently

---

## Configuration Issues

### Invalid Options

**Symptoms**:
- Error: `invalid options`
- Publishing fails at startup

**Diagnostic Steps**:

```go
opts := &ParallelPublishOptions{...}
if err := opts.Validate(); err != nil {
    log.Error("Invalid options", "error", err)
}
```

**Solutions**:

1. **Use Default Options**:
```go
opts := DefaultParallelPublishOptions()
```

2. **Validate Before Use**:
```go
if err := opts.Validate(); err != nil {
    return fmt.Errorf("invalid options: %w", err)
}
```

---

## Debugging Tips

### Enable Debug Logging

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug, // Enable debug logs
}))
```

### Add Request IDs

```go
ctx = context.WithValue(ctx, "request_id", uuid.New().String())
log.Info("Publishing", "request_id", ctx.Value("request_id"))
```

### Trace Individual Publishes

```go
for _, tr := range result.Results {
    log.Debug("Target result",
        "target_name", tr.TargetName,
        "success", tr.Success,
        "duration_ms", tr.Duration.Milliseconds(),
        "status_code", tr.StatusCode,
        "error", tr.Error,
    )
}
```

### Use pprof

```bash
# CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

---

## Monitoring & Alerts

### Key Metrics to Monitor

1. **Publishing Rate**:
   - `rate(parallel_publish_total[1m])`
   - Alert if < 10/min (low activity)

2. **Success Rate**:
   - `parallel_publish_success_total / parallel_publish_total`
   - Alert if < 95%

3. **Latency**:
   - `histogram_quantile(0.99, parallel_publish_duration_seconds_bucket)`
   - Alert if > 5s

4. **Partial Success Rate**:
   - `parallel_publish_partial_success_total / parallel_publish_total`
   - Alert if > 10% (high failure rate)

5. **Goroutine Count**:
   - `parallel_publish_active_goroutines`
   - Alert if > 1000 (potential leak)

### Recommended Alerts

```yaml
# PrometheusRule
groups:
  - name: parallel_publishing
    rules:
      - alert: HighPublishingFailureRate
        expr: rate(parallel_publish_failure_total[5m]) / rate(parallel_publish_total[5m]) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High publishing failure rate (> 10%)"

      - alert: PublishingLatencyHigh
        expr: histogram_quantile(0.99, parallel_publish_duration_seconds_bucket) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High publishing latency (p99 > 5s)"

      - alert: NoHealthyTargets
        expr: sum(target_health_status{status="healthy"}) == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No healthy targets available"
```

---

## Getting Help

If you encounter issues not covered in this guide:

1. **Check Logs**: Look for ERROR and WARN messages
2. **Check Metrics**: Review Prometheus metrics dashboard
3. **Check Health Status**: Review target health endpoints
4. **Enable Debug Logging**: Get more detailed logs
5. **Profile the Service**: Use pprof to identify bottlenecks
6. **Contact Team**: Escalate to on-call engineer

---

**Troubleshooting Guide Version**: 1.0.0
**Last Updated**: 2025-11-13
**Maintained By**: TN-058 Implementation Team
