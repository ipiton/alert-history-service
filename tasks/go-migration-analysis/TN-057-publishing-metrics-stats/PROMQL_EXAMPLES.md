# PromQL Examples for Publishing Metrics

**Comprehensive PromQL query examples for monitoring Alert History Service publishing infrastructure.**

---

## 📋 Table of Contents

1. [Health Metrics](#health-metrics)
2. [Queue Metrics](#queue-metrics)
3. [Discovery Metrics](#discovery-metrics)
4. [Refresh Metrics](#refresh-metrics)
5. [Publisher-Specific Metrics](#publisher-specific-metrics)
6. [Alerting Rules](#alerting-rules)
7. [Grafana Dashboard Panels](#grafana-dashboard-panels)

---

## Health Metrics

### Target Health Status

```promql
# Current health status (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)
alert_history_business_publishing_health_status
```

### Unhealthy Targets Count

```promql
# Count targets with status=unhealthy (3)
count(alert_history_business_publishing_health_status == 3)
```

### Healthy Targets Percentage

```promql
# % of healthy targets
(
  count(alert_history_business_publishing_health_status == 1) /
  count(alert_history_business_publishing_health_status)
) * 100
```

### Success Rate by Target

```promql
# Success rate for all targets
alert_history_business_publishing_health_success_rate

# Success rate for specific target
alert_history_business_publishing_health_success_rate{target="rootly-prod"}
```

### Average Success Rate

```promql
# Average success rate across all targets
avg(alert_history_business_publishing_health_success_rate)
```

### Targets with Low Success Rate

```promql
# Targets with success rate < 90%
alert_history_business_publishing_health_success_rate < 90
```

### Consecutive Failures

```promql
# Targets with consecutive failures
alert_history_business_publishing_health_consecutive_failures > 0

# Targets with critical failures (3+)
alert_history_business_publishing_health_consecutive_failures >= 3
```

### Health Check Duration

```promql
# Average health check latency
avg(alert_history_business_publishing_health_check_duration_seconds)

# 95th percentile latency
histogram_quantile(0.95,
  rate(alert_history_business_publishing_health_check_duration_seconds_bucket[5m])
)

# Slow health checks (>5s)
alert_history_business_publishing_health_check_duration_seconds > 5
```

### Health Checks per Second

```promql
# Rate of health checks
sum(rate(alert_history_business_publishing_health_checks_total[5m]))

# By target
sum by(target) (rate(alert_history_business_publishing_health_checks_total[5m]))
```

### Health Check Errors

```promql
# Total error rate
sum(rate(alert_history_business_publishing_health_errors_total[5m]))

# By error type
sum by(error_type) (rate(alert_history_business_publishing_health_errors_total[5m]))

# Error ratio (errors / total checks)
sum(rate(alert_history_business_publishing_health_errors_total[5m])) /
sum(rate(alert_history_business_publishing_health_checks_total[5m]))
```

---

## Queue Metrics

### Queue Size & Capacity

```promql
# Current queue size
alert_history_infrastructure_queue_size

# Queue capacity
alert_history_infrastructure_queue_capacity

# Queue utilization percentage
(alert_history_infrastructure_queue_size /
 alert_history_infrastructure_queue_capacity) * 100
```

### Queue Growth Rate

```promql
# Jobs/minute added to queue
rate(alert_history_infrastructure_queue_jobs_submitted[1m]) * 60

# 5-minute average growth
rate(alert_history_infrastructure_queue_jobs_submitted[5m]) * 60
```

### Job Processing Rate

```promql
# Jobs/second processed
rate(alert_history_infrastructure_queue_jobs_completed[1m])

# Jobs/minute
rate(alert_history_infrastructure_queue_jobs_completed[1m]) * 60
```

### Job Success Rate

```promql
# Success rate %
(
  sum(rate(alert_history_infrastructure_queue_jobs_completed[5m])) /
  sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
) * 100

# Failure rate %
(
  sum(rate(alert_history_infrastructure_queue_jobs_failed[5m])) /
  sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
) * 100
```

### Average Processing Time

```promql
# Average job duration
avg(alert_history_infrastructure_queue_job_duration_seconds)

# 99th percentile
histogram_quantile(0.99,
  rate(alert_history_infrastructure_queue_job_duration_seconds_bucket[5m])
)
```

### Job Wait Time

```promql
# Average time in queue before processing
avg(alert_history_infrastructure_queue_wait_duration_seconds)

# 95th percentile wait time
histogram_quantile(0.95,
  rate(alert_history_infrastructure_queue_wait_duration_seconds_bucket[5m])
)
```

### Retry Statistics

```promql
# Retry attempts per minute
rate(alert_history_infrastructure_queue_retries_total[1m]) * 60

# By target
sum by(target) (rate(alert_history_infrastructure_queue_retries_total[1m])) * 60

# Average retries per job
sum(rate(alert_history_infrastructure_queue_retries_total[5m])) /
sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
```

### Dead Letter Queue (DLQ)

```promql
# DLQ additions per minute
rate(alert_history_infrastructure_queue_dlq_total[1m]) * 60

# Current DLQ size
alert_history_infrastructure_queue_dlq_size

# DLQ ratio (% of submitted jobs)
(
  sum(rate(alert_history_infrastructure_queue_dlq_total[5m])) /
  sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
) * 100
```

### Circuit Breaker

```promql
# Circuit breaker state (0=closed, 1=open, 2=half-open)
alert_history_infrastructure_queue_circuit_breaker_state

# Circuit breaker trips
sum by(target) (rate(alert_history_infrastructure_queue_circuit_breaker_trips[5m]))
```

### Worker Pool

```promql
# Active workers
alert_history_infrastructure_queue_workers_active

# Idle workers
alert_history_infrastructure_queue_workers_idle

# Worker utilization %
(alert_history_infrastructure_queue_workers_active /
 (alert_history_infrastructure_queue_workers_active +
  alert_history_infrastructure_queue_workers_idle)) * 100
```

---

## Discovery Metrics

### Total Discovered Targets

```promql
# Current target count
alert_history_business_publishing_discovery_total_targets

# By type
sum by(type) (alert_history_business_publishing_discovery_targets_by_type)
```

### Discovery Duration

```promql
# Average discovery latency
avg(alert_history_business_publishing_discovery_duration_seconds) * 1000

# 95th percentile (ms)
histogram_quantile(0.95,
  rate(alert_history_business_publishing_discovery_duration_seconds_bucket[5m])
) * 1000
```

### Discovery Errors

```promql
# Error rate
sum(rate(alert_history_business_publishing_discovery_errors_total[5m]))

# By error type
sum by(error_type) (rate(alert_history_business_publishing_discovery_errors_total[5m]))
```

### Secrets Processed

```promql
# Secrets processed per discovery
alert_history_business_publishing_discovery_secrets_processed

# Processing rate
rate(alert_history_business_publishing_discovery_secrets_processed[5m])
```

### Discovery Frequency

```promql
# Time since last successful discovery (seconds)
time() - alert_history_business_publishing_discovery_last_success_timestamp

# Convert to minutes
(time() - alert_history_business_publishing_discovery_last_success_timestamp) / 60
```

---

## Refresh Metrics

### Refresh Status

```promql
# Targets discovered in last refresh
alert_history_business_publishing_refresh_targets_discovered

# Refresh in progress (0=idle, 1=in-progress)
alert_history_business_publishing_refresh_in_progress
```

### Refresh Timing

```promql
# Time since last refresh (seconds)
time() - alert_history_business_publishing_refresh_last_timestamp

# Time until next refresh (seconds)
alert_history_business_publishing_refresh_next_timestamp - time()
```

### Refresh Duration

```promql
# Average refresh duration (ms)
avg(alert_history_business_publishing_refresh_duration_seconds) * 1000

# Last refresh duration
alert_history_business_publishing_refresh_duration_seconds * 1000
```

### Refresh Errors

```promql
# Refresh error rate
rate(alert_history_business_publishing_refresh_errors_total[5m])

# Total refresh attempts
rate(alert_history_business_publishing_refresh_total[5m])

# Error ratio
sum(rate(alert_history_business_publishing_refresh_errors_total[5m])) /
sum(rate(alert_history_business_publishing_refresh_total[5m]))
```

---

## Publisher-Specific Metrics

### Rootly Publisher

```promql
# Incidents created per minute
rate(rootly_incidents_created_total[1m]) * 60

# Incident creation success rate
(
  sum(rate(rootly_incidents_created_total{status="success"}[5m])) /
  sum(rate(rootly_incidents_created_total[5m]))
) * 100

# API request duration (ms)
histogram_quantile(0.95,
  rate(rootly_api_request_duration_seconds_bucket[5m])
) * 1000

# Rate limit hits
rate(rootly_rate_limit_hits_total[5m])
```

### PagerDuty Publisher

```promql
# Events triggered per minute
rate(pagerduty_events_published_total{event_type="trigger"}[1m]) * 60

# Event success rate
(
  sum(rate(pagerduty_events_published_total{status="success"}[5m])) /
  sum(rate(pagerduty_events_published_total[5m]))
) * 100

# API latency (ms)
histogram_quantile(0.95,
  rate(pagerduty_api_request_duration_seconds_bucket[5m])
) * 1000
```

### Slack Publisher

```promql
# Messages posted per minute
rate(slack_messages_posted_total[1m]) * 60

# Thread reply success rate
(
  sum(rate(slack_thread_replies_total{status="success"}[5m])) /
  sum(rate(slack_thread_replies_total[5m]))
) * 100

# Cache hit rate
(
  sum(rate(slack_cache_hits_total[5m])) /
  (sum(rate(slack_cache_hits_total[5m])) + sum(rate(slack_cache_misses_total[5m])))
) * 100
```

### Generic Webhook Publisher

```promql
# Webhook requests per minute
rate(webhook_requests_total[1m]) * 60

# Success rate
(
  sum(rate(webhook_requests_total{status="success"}[5m])) /
  sum(rate(webhook_requests_total[5m]))
) * 100

# Authentication failures
rate(webhook_auth_failures_total[5m])

# Timeout rate
(
  sum(rate(webhook_timeout_errors_total[5m])) /
  sum(rate(webhook_requests_total[5m]))
) * 100
```

---

## Alerting Rules

### Critical Alerts

```yaml
groups:
  - name: publishing_critical
    interval: 30s
    rules:
      # Multiple unhealthy targets
      - alert: PublishingMultipleUnhealthyTargets
        expr: count(alert_history_business_publishing_health_status == 3) >= 2
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Multiple publishing targets unhealthy"
          description: "{{ $value }} targets are unhealthy for >5m"

      # Queue near capacity
      - alert: PublishingQueueNearCapacity
        expr: |
          (alert_history_infrastructure_queue_size /
           alert_history_infrastructure_queue_capacity) * 100 > 80
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Publishing queue near capacity"
          description: "Queue utilization {{ $value | humanize }}% for >10m"

      # High failure rate
      - alert: PublishingHighFailureRate
        expr: |
          (
            sum(rate(alert_history_infrastructure_queue_jobs_failed[5m])) /
            sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
          ) * 100 > 20
        for: 15m
        labels:
          severity: critical
        annotations:
          summary: "High publishing failure rate"
          description: "{{ $value | humanize }}% jobs failing for >15m"
```

### Warning Alerts

```yaml
  - name: publishing_warning
    interval: 1m
    rules:
      # Degraded target
      - alert: PublishingTargetDegraded
        expr: alert_history_business_publishing_health_status == 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Publishing target degraded: {{ $labels.target }}"
          description: "Target {{ $labels.target }} degraded for >10m"

      # Low success rate
      - alert: PublishingLowSuccessRate
        expr: alert_history_business_publishing_health_success_rate < 95
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Low success rate: {{ $labels.target }}"
          description: "Success rate {{ $value | humanize }}% for >15m"

      # Queue growth
      - alert: PublishingQueueGrowing
        expr: deriv(alert_history_infrastructure_queue_size[10m]) > 10
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Publishing queue growing"
          description: "Queue growing at {{ $value | humanize }} jobs/sec"

      # Slow discovery
      - alert: PublishingSlowDiscovery
        expr: |
          alert_history_business_publishing_discovery_duration_seconds > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow target discovery"
          description: "Discovery taking {{ $value | humanize }}s"
```

---

## Grafana Dashboard Panels

### Panel 1: Health Overview

**Type:** Stat
**Query:**
```promql
count(alert_history_business_publishing_health_status == 1)
```
**Display:** Value + gauge
**Thresholds:**
- Red: <8
- Yellow: 8-9
- Green: >=10

---

### Panel 2: Success Rate

**Type:** Gauge
**Query:**
```promql
avg(alert_history_business_publishing_health_success_rate)
```
**Unit:** Percent (0-100)
**Thresholds:**
- Red: <90
- Yellow: 90-95
- Green: >=95

---

### Panel 3: Target Health Status

**Type:** Table
**Queries:**
```promql
# Status
alert_history_business_publishing_health_status

# Success rate
alert_history_business_publishing_health_success_rate

# Failures
alert_history_business_publishing_health_consecutive_failures
```
**Columns:** Target | Status | Success Rate | Failures

---

### Panel 4: Queue Utilization

**Type:** Time series
**Queries:**
```promql
# Queue size
alert_history_infrastructure_queue_size

# Capacity (constant)
alert_history_infrastructure_queue_capacity
```
**Display:** Area chart
**Y-axis:** Jobs

---

### Panel 5: Job Processing Rate

**Type:** Time series
**Queries:**
```promql
# Submitted
rate(alert_history_infrastructure_queue_jobs_submitted[1m]) * 60

# Completed
rate(alert_history_infrastructure_queue_jobs_completed[1m]) * 60

# Failed
rate(alert_history_infrastructure_queue_jobs_failed[1m]) * 60
```
**Unit:** Jobs/minute

---

### Panel 6: Publishing Latency

**Type:** Time series
**Query:**
```promql
histogram_quantile(0.95,
  rate(alert_history_infrastructure_queue_job_duration_seconds_bucket[5m])
) * 1000
```
**Unit:** Milliseconds
**Legend:** P95 Latency

---

### Panel 7: Discovery & Refresh

**Type:** Stat (multi-value)
**Queries:**
```promql
# Targets discovered
alert_history_business_publishing_discovery_total_targets

# Last refresh (minutes ago)
(time() - alert_history_business_publishing_refresh_last_timestamp) / 60
```

---

### Panel 8: Error Rates

**Type:** Time series
**Queries:**
```promql
# Health check errors
sum(rate(alert_history_business_publishing_health_errors_total[5m]))

# Queue failures
sum(rate(alert_history_infrastructure_queue_jobs_failed[5m]))

# Discovery errors
sum(rate(alert_history_business_publishing_discovery_errors_total[5m]))
```
**Unit:** Errors/second

---

### Complete Dashboard JSON

```json
{
  "dashboard": {
    "title": "Publishing Metrics & Statistics",
    "panels": [
      {
        "id": 1,
        "title": "Healthy Targets",
        "type": "stat",
        "targets": [
          {
            "expr": "count(alert_history_business_publishing_health_status == 1)"
          }
        ]
      },
      {
        "id": 2,
        "title": "Average Success Rate",
        "type": "gauge",
        "targets": [
          {
            "expr": "avg(alert_history_business_publishing_health_success_rate)"
          }
        ]
      },
      {
        "id": 3,
        "title": "Queue Utilization",
        "type": "timeseries",
        "targets": [
          {
            "expr": "alert_history_infrastructure_queue_size"
          },
          {
            "expr": "alert_history_infrastructure_queue_capacity"
          }
        ]
      }
    ]
  }
}
```

---

## Recording Rules

**Optimize frequently used queries:**

```yaml
groups:
  - name: publishing_recordings
    interval: 30s
    rules:
      # Pre-compute success rate
      - record: publishing:health_success_rate:avg
        expr: avg(alert_history_business_publishing_health_success_rate)

      # Pre-compute queue utilization
      - record: publishing:queue_utilization:percent
        expr: |
          (alert_history_infrastructure_queue_size /
           alert_history_infrastructure_queue_capacity) * 100

      # Pre-compute job success rate
      - record: publishing:job_success_rate:percent
        expr: |
          (
            sum(rate(alert_history_infrastructure_queue_jobs_completed[5m])) /
            sum(rate(alert_history_infrastructure_queue_jobs_submitted[5m]))
          ) * 100
```

---

**Last Updated:** 2025-11-13
