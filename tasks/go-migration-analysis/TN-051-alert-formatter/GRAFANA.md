# TN-051 Alert Formatter - Grafana Dashboards & Monitoring

**Version**: 1.0
**Date**: 2025-11-10

---

## 📊 Overview

This document provides Grafana dashboard configurations and PromQL queries for monitoring alert formatting performance and observability.

---

## 🎯 Key Metrics

### 1. Format Duration (p50, p95, p99)

**PromQL**:
```promql
# p50
histogram_quantile(0.50, sum(rate(alert_history_publishing_format_duration_seconds_bucket[5m])) by (le, format))

# p95
histogram_quantile(0.95, sum(rate(alert_history_publishing_format_duration_seconds_bucket[5m])) by (le, format))

# p99
histogram_quantile(0.99, sum(rate(alert_history_publishing_format_duration_seconds_bucket[5m])) by (le, format))
```

**Dashboard Panel**:
- **Type**: Graph
- **Title**: "Format Duration by Format Type"
- **Y-axis**: Seconds
- **Legend**: `{{format}} p{{quantile}}`

---

### 2. Format Success Rate

**PromQL**:
```promql
sum(rate(alert_history_publishing_format_total{status="success"}[5m])) by (format)
/
sum(rate(alert_history_publishing_format_total[5m])) by (format) * 100
```

**Dashboard Panel**:
- **Type**: Gauge
- **Title**: "Format Success Rate (%)"
- **Threshold**: Red < 95%, Yellow 95-99%, Green ≥ 99%

---

### 3. Cache Hit Rate

**PromQL**:
```promql
sum(rate(alert_history_publishing_cache_hits_total[5m])) by (format)
/
(sum(rate(alert_history_publishing_cache_hits_total[5m])) by (format) + sum(rate(alert_history_publishing_cache_misses_total[5m])) by (format)) * 100
```

**Dashboard Panel**:
- **Type**: Stat
- **Title**: "Cache Hit Rate (%)"
- **Threshold**: Red < 50%, Yellow 50-80%, Green ≥ 80%

---

### 4. Format Error Rate by Type

**PromQL**:
```promql
sum(rate(alert_history_publishing_format_errors_total[5m])) by (error_type)
```

**Dashboard Panel**:
- **Type**: Bar Chart
- **Title**: "Error Rate by Type"
- **Legend**: `{{error_type}}`

---

### 5. Validation Failure Rate by Rule

**PromQL**:
```promql
topk(10, sum(rate(alert_history_publishing_validation_failures_total[5m])) by (rule))
```

**Dashboard Panel**:
- **Type**: Table
- **Title**: "Top 10 Validation Failures"
- **Columns**: Rule, Rate

---

### 6. Format Payload Size (p50, p95, p99)

**PromQL**:
```promql
histogram_quantile(0.50, sum(rate(alert_history_publishing_format_bytes_bucket[5m])) by (le, format))
histogram_quantile(0.95, sum(rate(alert_history_publishing_format_bytes_bucket[5m])) by (le, format))
histogram_quantile(0.99, sum(rate(alert_history_publishing_format_bytes_bucket[5m])) by (le, format))
```

**Dashboard Panel**:
- **Type**: Graph
- **Title**: "Payload Size Distribution"
- **Y-axis**: Bytes

---

## 🚨 Alerting Rules

### High Error Rate Alert

```yaml
alert: HighFormatErrorRate
expr: |
  sum(rate(alert_history_publishing_format_errors_total[5m])) by (format)
  / sum(rate(alert_history_publishing_format_total[5m])) by (format) > 0.05
for: 5m
labels:
  severity: warning
  component: alert-formatter
annotations:
  summary: "High format error rate for {{ $labels.format }}"
  description: "Format {{ $labels.format }} has error rate > 5% (current: {{ $value | humanizePercentage }})"
```

### Low Cache Hit Rate Alert

```yaml
alert: LowCacheHitRate
expr: |
  sum(rate(alert_history_publishing_cache_hits_total[10m])) by (format)
  / (sum(rate(alert_history_publishing_cache_hits_total[10m])) by (format) + sum(rate(alert_history_publishing_cache_misses_total[10m])) by (format)) < 0.30
for: 10m
labels:
  severity: info
  component: alert-formatter
annotations:
  summary: "Low cache hit rate for {{ $labels.format }}"
  description: "Cache hit rate < 30% for {{ $labels.format }} (current: {{ $value | humanizePercentage }})"
```

### High Format Duration Alert

```yaml
alert: HighFormatDuration
expr: |
  histogram_quantile(0.95, sum(rate(alert_history_publishing_format_duration_seconds_bucket[5m])) by (le, format)) > 0.1
for: 5m
labels:
  severity: warning
  component: alert-formatter
annotations:
  summary: "High format duration for {{ $labels.format }}"
  description: "p95 format duration > 100ms for {{ $labels.format }} (current: {{ $value }}s)"
```

---

## 📋 Complete Grafana Dashboard JSON

```json
{
  "dashboard": {
    "title": "Alert Formatter Monitoring",
    "panels": [
      {
        "id": 1,
        "title": "Format Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(alert_history_publishing_format_duration_seconds_bucket[5m])) by (le, format))"
          }
        ],
        "type": "graph"
      },
      {
        "id": 2,
        "title": "Success Rate",
        "targets": [
          {
            "expr": "sum(rate(alert_history_publishing_format_total{status=\"success\"}[5m])) by (format) / sum(rate(alert_history_publishing_format_total[5m])) by (format) * 100"
          }
        ],
        "type": "gauge"
      },
      {
        "id": 3,
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "sum(rate(alert_history_publishing_cache_hits_total[5m])) by (format) / (sum(rate(alert_history_publishing_cache_hits_total[5m])) by (format) + sum(rate(alert_history_publishing_cache_misses_total[5m])) by (format)) * 100"
          }
        ],
        "type": "stat"
      }
    ]
  }
}
```

---

## 🔍 Distributed Tracing Integration

### Jaeger Query Examples

**Find slow formatting requests**:
```
service:alert-history operation:FormatAlert minDuration:100ms
```

**Find validation errors**:
```
service:alert-history operation:Validation tags:validation.errors_count>0
```

**Trace cache performance**:
```
service:alert-history operation:CacheCheck tags:cache.hit=false
```

---

## 🚀 Production Deployment

### 1. Enable Metrics

```go
// Create formatter metrics
metrics := publishing.NewFormatterMetrics("alert_history", "publishing")

// Wrap formatter with metrics middleware
formatter := publishing.NewMiddlewareChain(
    baseFormatter,
    publishing.MetricsMiddleware(metrics),
)
```

### 2. Enable Tracing

```go
// Create tracer
tracer := publishing.NewSimpleTracer(logger)

// Or use OpenTelemetry
// tracer := otel.Tracer("alert-history/publishing")

// Wrap formatter with tracing middleware
formatter := publishing.NewMiddlewareChain(
    baseFormatter,
    publishing.TracingMiddleware(tracer),
)
```

### 3. Combined Stack

```go
// Full observability stack
formatter := publishing.NewMiddlewareChain(
    baseFormatter,
    // Validation (first)
    publishing.TracingValidationMiddleware(tracer, validator),
    // Caching (with tracing)
    publishing.TracingCacheMiddleware(tracer, cache, 5*time.Minute, logger),
    // Metrics (record all operations)
    publishing.MetricsMiddleware(metrics),
    // Tracing (root span)
    publishing.TracingMiddleware(tracer),
)
```

---

## 📊 Sample Dashboard Screenshot

```
+-----------------------------------------------------------+
| Alert Formatter Monitoring                                 |
+-----------------------------------------------------------+
| Format Duration (p95)                 Success Rate        |
| Alertmanager: 0.8ms                   99.9%               |
| Rootly: 1.2ms                         99.5%               |
| PagerDuty: 1.0ms                      99.8%               |
| Slack: 0.9ms                          99.7%               |
+-----------------------------------------------------------+
| Cache Hit Rate                        Error Rate          |
| Overall: 87%                          0.01%               |
+-----------------------------------------------------------+
```

---

## ✅ Monitoring Checklist

- [x] 7 Prometheus metrics exported
- [x] Grafana dashboards configured
- [x] Alerting rules defined
- [x] Distributed tracing integrated
- [x] Cache observability enabled
- [x] Validation metrics tracked
- [x] Error classification implemented

---

**Status**: Production-Ready ✅
