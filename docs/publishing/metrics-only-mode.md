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

## Implementation

### Endpoint: GET /api/v1/publishing/mode

```bash
curl http://localhost:8080/api/v1/publishing/mode
```

**Response (Normal Mode)**:
```json
{
  "mode": "normal",
  "targets_available": true,
  "enabled_targets": 5,
  "metrics_only_active": false
}
```

**Response (Metrics-Only Mode)**:
```json
{
  "mode": "metrics-only",
  "targets_available": false,
  "enabled_targets": 0,
  "metrics_only_active": true
}
```

### Code Implementation

From `handlers.go`:

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

## Monitoring

### Prometheus Metrics

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

## Troubleshooting

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
