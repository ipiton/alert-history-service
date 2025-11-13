# Metrics-Only Fallback Mode

## Overview

The Publishing System implements a **metrics-only fallback mode** that provides graceful degradation when no publishing targets are available. This ensures the system continues to function and collect observability data even when external integrations are temporarily unavailable.

## How It Works

### Automatic Mode Detection

The system automatically detects and switches between modes based on target availability:

```
┌─────────────────────────────────────────┐
│     Publishing System                   │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │  Check Enabled Targets            │ │
│  │  enabled_count > 0?               │ │
│  └────────────┬──────────────────────┘ │
│               │                         │
│         ┌─────┴─────┐                   │
│         │    YES    │    NO             │
│         ▼           ▼                   │
│  ┌──────────┐ ┌──────────────┐         │
│  │  Normal  │ │ Metrics-Only │         │
│  │   Mode   │ │     Mode     │         │
│  └──────────┘ └──────────────┘         │
│       │              │                  │
│       ▼              ▼                  │
│  Publish +      Metrics Only           │
│  Metrics        (No Publishing)        │
└─────────────────────────────────────────┘
```

### Mode States

#### Normal Mode
- **Condition**: `enabled_targets > 0`
- **Behavior**:
  - Alerts are queued for publishing
  - Publishers attempt delivery to targets
  - Metrics are collected
  - Circuit breakers monitor target health

#### Metrics-Only Mode
- **Condition**: `enabled_targets == 0`
- **Behavior**:
  - Alert processing continues
  - NO external publishing attempts
  - Metrics continue to be collected
  - Queue remains empty
  - System stays healthy and observable

## Architecture (TN-060)

### ModeManager Component

The TN-060 implementation introduces a centralized `ModeManager` component that manages mode state and transitions:

```
┌─────────────────────────────────────────────────┐
│              ModeManager                        │
│  ┌───────────────────────────────────────────┐ │
│  │  State: currentMode (Normal/MetricsOnly)  │ │
│  │  Transitions: tracked with atomic counter │ │
│  │  Caching: <100ns read performance         │ │
│  │  Subscribers: event-driven notifications  │ │
│  └───────────────────────────────────────────┘ │
│                                                 │
│  ┌──────────────────┬──────────────────┐       │
│  │  Integration     │  Integration     │       │
│  │  - Handlers      │  - Queue         │       │
│  │  - Coordinator   │  - Publisher     │       │
│  └──────────────────┴──────────────────┘       │
└─────────────────────────────────────────────────┘
```

**Key Features**:
- **Automatic mode detection**: Based on enabled target count
- **Thread-safe**: `sync.RWMutex` for concurrent access
- **High performance**: <100ns for mode checks (0 allocations)
- **Event-driven**: Subscribe to mode change notifications
- **Metrics**: Prometheus integration for observability
- **Periodic checking**: Background goroutine (5s interval)
- **Graceful shutdown**: Clean lifecycle management

### Performance Characteristics

**Benchmarks** (on Apple Silicon M1):
- `GetCurrentMode()`: **34 ns/op** (0 allocs)
- `IsMetricsOnly()`: **35 ns/op** (0 allocs)
- `CheckModeTransition()`: **173 ns/op** (1 alloc)
- Concurrent access: **141 ns/op** (0 allocs)

**Throughput**: >29M ops/sec (concurrent reads)

## Implementation

### Endpoint: GET /api/v1/publishing/mode (Enhanced)

```bash
curl http://localhost:8080/api/v1/publishing/mode
```

**Response (Normal Mode)** - Enhanced with TN-060 metrics:
```json
{
  "mode": "normal",
  "targets_available": true,
  "enabled_targets": 5,
  "metrics_only_active": false,
  "transition_count": 12,
  "current_mode_duration_seconds": 3600.5,
  "last_transition_time": "2025-11-14T10:30:00Z",
  "last_transition_reason": "targets_available"
}
```

**Response (Metrics-Only Mode)** - Enhanced with TN-060 metrics:
```json
{
  "mode": "metrics-only",
  "targets_available": false,
  "enabled_targets": 0,
  "metrics_only_active": true,
  "transition_count": 13,
  "current_mode_duration_seconds": 120.3,
  "last_transition_time": "2025-11-14T12:30:00Z",
  "last_transition_reason": "no_enabled_targets"
}
```

**New Fields (TN-060)**:
- `transition_count`: Total number of mode transitions since startup
- `current_mode_duration_seconds`: How long system has been in current mode
- `last_transition_time`: When the last transition occurred
- `last_transition_reason`: Why the transition happened (`targets_available`, `no_enabled_targets`, `targets_disabled`)

### Integration Examples

#### 1. Checking Mode Before Publishing (Handlers)

```go
func (h *PublishingHandlers) SubmitAlert(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...

    // TN-060: Check if in metrics-only mode
    if h.modeManager != nil && h.modeManager.IsMetricsOnly() {
        h.logger.Info("Alert submission rejected (metrics-only mode)",
            "alert_fingerprint", req.Alert.Fingerprint)

        h.sendJSON(w, http.StatusOK, SubmitAlertResponse{
            Success: false,
            Message: "Alert not submitted: system is in metrics-only mode",
            Mode:    "metrics-only",
        })
        return
    }

    // ... proceed with submission ...
}
```

#### 2. Skipping Jobs in Queue Workers

```go
func (q *PublishingQueue) worker(id int) {
    for {
        job := q.selectNextJob()

        if job != nil {
            // TN-060: Skip processing in metrics-only mode
            if q.modeManager != nil && q.modeManager.IsMetricsOnly() {
                q.logger.Debug("Job skipped (metrics-only mode)",
                    "job_id", job.ID,
                    "target", job.Target.Name)
                continue
            }

            // ... process job ...
        }
    }
}
```

#### 3. Subscribing to Mode Changes

```go
// Subscribe to mode change events
modeManager.Subscribe(func(from, to publishing.Mode, reason string) {
    log.Printf("Mode changed: %v -> %v (reason: %s)", from, to, reason)

    // Send alert to monitoring
    if to == publishing.ModeMetricsOnly {
        sendAlert("Publishing system entered metrics-only mode")
    }
})
```

#### 4. Monitoring Mode via Prometheus

```promql
# Current mode (0=normal, 1=metrics-only)
alert_history_publishing_mode_current

# Total transitions
alert_history_publishing_mode_transitions_total

# Time in each mode
histogram_quantile(0.99,
  rate(alert_history_publishing_mode_duration_seconds_bucket[5m])
)

# Mode check performance
histogram_quantile(0.99,
  rate(alert_history_publishing_mode_check_duration_seconds_bucket[5m])
)
```

### Code Implementation (Legacy)

From `handlers.go` (pre-TN-060):

```go
func (h *PublishingHandlers) GetPublishingMode(w http.ResponseWriter, r *http.Request) {
    // Count enabled targets
    targets := h.discoveryManager.ListTargets()
    enabledCount := 0
    for _, t := range targets {
        if t.Enabled {
            enabledCount++
        }
    }
    targetsAvailable := enabledCount > 0

    mode := "normal"
    metricsOnly := false

    if !targetsAvailable {
        mode = "metrics-only"
        metricsOnly = true
    }

    response := PublishingModeResponse{
        Mode:              mode,
        TargetsAvailable:  targetsAvailable,
        EnabledTargets:    enabledCount,
        MetricsOnlyActive: metricsOnly,
    }

    h.sendJSON(w, http.StatusOK, response)
}
```

## Benefits

### 1. Graceful Degradation
- System continues operating without errors
- No cascade failures from missing targets
- Alert processing pipeline stays healthy

### 2. Observability Maintained
- Metrics continue to be collected
- Prometheus scraping works normally
- Dashboards show accurate state

### 3. Easy Recovery
- When targets are added, system automatically resumes publishing
- No manual intervention required
- Seamless transition back to normal mode

### 4. Operational Visibility
- Clear mode indication via API
- Prometheus metrics show target availability
- Alerting on metrics-only mode possible

## Prometheus Metrics (TN-060)

### Mode-Specific Metrics

The ModeManager exposes comprehensive Prometheus metrics:

1. **`publishing_mode_current`** (Gauge)
   - Current mode: `0` = normal, `1` = metrics-only
   - Use for alerting and dashboards

2. **`publishing_mode_transitions_total`** (Counter)
   - Total number of mode transitions
   - Labeled by: `from_mode`, `to_mode`
   - High transition count may indicate instability

3. **`publishing_mode_duration_seconds`** (Histogram)
   - Time spent in each mode
   - Labeled by: `mode` (normal, metrics-only)
   - Buckets: 1s to 1h

4. **`publishing_mode_check_duration_seconds`** (Histogram)
   - Mode check operation latency
   - Buckets: 1µs to 1ms
   - Should be <100µs

5. **`publishing_submissions_rejected_total`** (Counter)
   - Submissions rejected due to metrics-only mode
   - Labeled by: `reason="metrics_only"`

6. **`publishing_jobs_skipped_total`** (Counter)
   - Jobs skipped in queue due to metrics-only mode
   - Labeled by: `reason="metrics_only"`

### Example Queries

```promql
# Alert if system stays in metrics-only mode > 5 minutes
alert_history_publishing_mode_current == 1

# Rate of transitions (should be low)
rate(alert_history_publishing_mode_transitions_total[5m])

# Submissions rejected per second
rate(alert_history_publishing_submissions_rejected_total[1m])

# Jobs skipped per second
rate(alert_history_publishing_jobs_skipped_total[1m])
```

## Monitoring

### General Prometheus Metrics

Key metrics for monitoring mode state:

```promql
# Number of enabled targets (0 = metrics-only mode)
publishing_enabled_targets

# Number of discovered targets
publishing_discovered_targets

# Queue size (should be 0 in metrics-only mode)
publishing_queue_size

# Publishing errors (should be 0 in metrics-only mode)
publishing_alerts_errors_total
```

### Alert Rules

Example Prometheus alert for metrics-only mode:

```yaml
groups:
  - name: publishing
    rules:
      - alert: PublishingMetricsOnlyMode
        expr: publishing_enabled_targets == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Publishing system in metrics-only mode"
          description: "No enabled publishing targets available for {{ $value }} minutes"
```

## Use Cases

### 1. Initial Deployment
- System starts before targets are configured
- Runs in metrics-only mode until secrets are created
- Auto-transitions to normal mode after target discovery

### 2. Maintenance Window
- Targets temporarily disabled for maintenance
- System continues processing alerts
- Metrics show zero publishing attempts
- Re-enables automatically when targets return

### 3. Configuration Issues
- Invalid target configurations
- Network connectivity problems
- Authorization failures
- System degrades gracefully instead of failing

### 4. Testing
- Test alert processing without external calls
- Verify metrics collection
- Validate system behavior under degraded conditions

## Transition Scenarios

### Normal → Metrics-Only

**Triggers**:
- Last enabled target disabled
- All targets fail health checks
- Target discovery returns empty list

**Actions**:
- Stop publishing attempts
- Clear queue (optional)
- Update mode metric
- Log mode change

### Metrics-Only → Normal

**Triggers**:
- New target discovered
- Existing target re-enabled
- Target refresh succeeds with enabled targets

**Actions**:
- Resume publishing
- Process queued alerts
- Update mode metric
- Log mode change

## Configuration

### Automatic (Default)

No configuration required - mode switches automatically based on target availability.

### Manual Control (Optional)

For advanced use cases, mode can be controlled via:

1. **Target Labels**: Disable all targets to force metrics-only mode
2. **Feature Flags**: Environment variable to force mode
3. **API**: Future endpoint to set mode explicitly

## Best Practices

### 1. Monitoring
- Alert on metrics-only mode after threshold (e.g., 5 minutes)
- Track mode transitions in logs
- Dashboard showing current mode

### 2. Target Management
- Maintain at least 2 targets for redundancy
- Use health checks to auto-disable failing targets
- Test target configurations before enabling

### 3. Recovery
- Automated target discovery
- Health check retries with backoff
- Manual override capability via API

### 4. Testing
- Regularly test metrics-only mode behavior
- Verify alert processing continues
- Confirm metrics remain accurate

## Metrics in Metrics-Only Mode

Even in metrics-only mode, these metrics are collected:

```
publishing_discovered_targets{} 0
publishing_enabled_targets{} 0
publishing_queue_size{} 0
publishing_queue_capacity{} 1000
publishing_alerts_published_total{} 0 (no new publishes)
publishing_target_refreshes_total{} (continues incrementing)
publishing_target_discovery_duration_seconds{} (continues collecting)
```

## Troubleshooting (TN-060)

### Issue: Stuck in Metrics-Only Mode

**Symptoms**:
- `publishing_mode_current = 1` for extended period
- All submissions rejected
- No publishing jobs processed

**Diagnosis**:
```bash
# Check current mode
curl http://localhost:8080/api/v1/publishing/mode

# Check target discovery
curl http://localhost:8080/api/v1/publishing/targets

# Check Prometheus metrics
curl http://localhost:8080/metrics | grep publishing_mode
```

**Common Causes**:
1. **No targets configured** - Check Kubernetes secrets with `publishing-target=true` label
2. **All targets disabled** - Verify target `enabled: true` in secrets
3. **Discovery failure** - Check logs for K8s API errors
4. **RBAC issues** - Ensure service account has `secrets:get/list` permissions

**Resolution**:
```bash
# 1. Verify K8s secret exists
kubectl get secrets -l publishing-target=true

# 2. Check secret content
kubectl get secret <secret-name> -o yaml

# 3. Force target refresh (if available)
curl -X POST http://localhost:8080/api/v1/publishing/targets/refresh

# 4. Restart service (last resort)
kubectl rollout restart deployment/alert-history
```

### Issue: Frequent Mode Transitions

**Symptoms**:
- High `publishing_mode_transitions_total` rate
- Flapping between modes
- Unstable publishing

**Diagnosis**:
```promql
# Check transition rate
rate(alert_history_publishing_mode_transitions_total[5m])

# Check mode duration (should be > 1 minute)
rate(alert_history_publishing_mode_duration_seconds[5m])
```

**Common Causes**:
1. **Target health issues** - Targets going up/down frequently
2. **Discovery instability** - K8s API timeouts/errors
3. **Config changes** - Frequent secret updates

**Resolution**:
- Investigate target health issues
- Check K8s API stability
- Review secret update patterns
- Consider adding hysteresis/debouncing (future enhancement)

### Issue: High Mode Check Latency

**Symptoms**:
- `publishing_mode_check_duration_seconds` > 1ms
- Slow request handling

**Diagnosis**:
```promql
# Check P99 latency
histogram_quantile(0.99,
  rate(alert_history_publishing_mode_check_duration_seconds_bucket[5m])
)
```

**Common Causes**:
1. **Lock contention** - High concurrent access
2. **Large target list** - Slow iteration
3. **Resource constraints** - CPU throttling

**Resolution**:
- Review concurrent request patterns
- Optimize target discovery
- Scale horizontally
- Cache is already enabled (should be <100ns)

### Issue: Memory Leak or High Allocations

**Diagnosis**:
```bash
# Run benchmarks
go test -bench=. -benchmem ./internal/infrastructure/publishing

# Profile memory
go test -memprofile=mem.prof ./internal/infrastructure/publishing
go tool pprof mem.prof
```

**Expected**: 0 allocations for `GetCurrentMode()` and `IsMetricsOnly()`

**Resolution**:
- Check for subscriber leaks (unsubscribe not called)
- Review metric label cardinality
- Monitor goroutine count

## Troubleshooting (Legacy)

### Issue: Stuck in Metrics-Only Mode

**Symptoms**:
- Mode API returns `metrics_only: true`
- No alerts being published
- `publishing_enabled_targets` = 0

**Solutions**:
1. Check target secrets exist:
   ```bash
   kubectl get secrets -l publishing-target=true
   ```

2. Verify target enabled field:
   ```bash
   kubectl get secret <target-name> -o yaml | grep enabled
   ```

3. Trigger manual refresh:
   ```bash
   curl -X POST http://localhost:8080/api/v1/publishing/targets/refresh
   ```

4. Check discovery logs:
   ```bash
   kubectl logs -l app=alert-history | grep "target discovery"
   ```

### Issue: Mode Not Switching

**Symptoms**:
- Targets exist but mode stays metrics-only
- OR targets removed but mode stays normal

**Solutions**:
1. Check target discovery interval
2. Force manual refresh via API
3. Restart publishing system
4. Verify RBAC permissions for secrets access

## References

- [Publishing System Architecture](./architecture.md)
- [Target Discovery](./target-discovery.md)
- [Metrics Reference](./metrics.md)
- [API Documentation](./api.md)

## Summary

The metrics-only fallback mode provides:
- ✅ Graceful degradation
- ✅ Continued observability
- ✅ Automatic recovery
- ✅ Zero-downtime transitions
- ✅ Production-ready reliability

No additional configuration required - works out of the box!
