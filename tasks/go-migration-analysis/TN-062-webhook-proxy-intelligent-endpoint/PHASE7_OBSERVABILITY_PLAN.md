# TN-062: Phase 7 - Observability Enhancement Plan

**Date**: 2025-11-16
**Status**: 🔄 IN PROGRESS
**Target**: 18+ Prometheus Metrics + Grafana Dashboard

---

## Executive Summary

Phase 7 focuses on comprehensive observability for the Intelligent Proxy Webhook endpoint, including:
- **18+ Prometheus metrics** for monitoring
- **Grafana dashboard** for visualization
- **Alerting rules** for operational awareness
- **Distributed tracing** integration (optional)
- **Performance profiling** endpoints

---

## 1. Metrics Architecture

### 1.1 Metric Categories

| Category | Metrics | Purpose |
|----------|---------|---------|
| **HTTP** | 6 | Request/response tracking |
| **Processing** | 5 | Pipeline performance |
| **Errors** | 3 | Error tracking |
| **Performance** | 4 | Latency & throughput |
| **Total** | **18+** | Comprehensive coverage |

### 1.2 Metric Naming Convention

Following Prometheus best practices:
```
alert_history_proxy_{subsystem}_{metric}_{unit}
```

Examples:
- `alert_history_proxy_http_requests_total`
- `alert_history_proxy_processing_duration_seconds`
- `alert_history_proxy_classification_errors_total`

---

## 2. Prometheus Metrics Specification

### 2.1 HTTP Metrics (6 metrics)

#### 1. **alert_history_proxy_http_requests_total**
- **Type**: Counter
- **Labels**: `method`, `path`, `status_code`
- **Description**: Total HTTP requests received
- **Usage**: Track request volume and status distribution

#### 2. **alert_history_proxy_http_request_duration_seconds**
- **Type**: Histogram
- **Labels**: `method`, `path`, `status_code`
- **Buckets**: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
- **Description**: HTTP request duration
- **Usage**: Calculate p50, p95, p99 latencies

#### 3. **alert_history_proxy_http_request_size_bytes**
- **Type**: Histogram
- **Labels**: `method`, `path`
- **Buckets**: `[100, 1000, 10000, 100000, 1000000, 10000000]`
- **Description**: HTTP request body size
- **Usage**: Monitor payload sizes

#### 4. **alert_history_proxy_http_response_size_bytes**
- **Type**: Histogram
- **Labels**: `method`, `path`, `status_code`
- **Buckets**: `[100, 1000, 10000, 100000]`
- **Description**: HTTP response body size
- **Usage**: Monitor response sizes

#### 5. **alert_history_proxy_http_requests_in_flight**
- **Type**: Gauge
- **Labels**: `method`
- **Description**: Current number of requests being processed
- **Usage**: Monitor concurrency

#### 6. **alert_history_proxy_http_errors_total**
- **Type**: Counter
- **Labels**: `method`, `path`, `error_type`
- **Description**: HTTP errors by type
- **Usage**: Track error patterns

### 2.2 Processing Metrics (5 metrics)

#### 7. **alert_history_proxy_alerts_received_total**
- **Type**: Counter
- **Labels**: `status`
- **Description**: Total alerts received (firing/resolved)
- **Usage**: Track alert volume

#### 8. **alert_history_proxy_alerts_processed_total**
- **Type**: Counter
- **Labels**: `status`, `result`
- **Description**: Alerts processed (success/filtered/failed)
- **Usage**: Track processing success rate

#### 9. **alert_history_proxy_classification_duration_seconds**
- **Type**: Histogram
- **Labels**: `cached`
- **Buckets**: `[0.001, 0.01, 0.1, 1.0, 5.0]`
- **Description**: LLM classification duration
- **Usage**: Monitor classification performance

#### 10. **alert_history_proxy_filtering_duration_seconds**
- **Type**: Histogram
- **Labels**: `action`
- **Buckets**: `[0.0001, 0.001, 0.01]`
- **Description**: Filtering pipeline duration
- **Usage**: Monitor filter performance

#### 11. **alert_history_proxy_publishing_duration_seconds**
- **Type**: Histogram
- **Labels**: `target_type`
- **Buckets**: `[0.1, 0.5, 1.0, 5.0, 10.0]`
- **Description**: Publishing pipeline duration
- **Usage**: Monitor publishing performance

### 2.3 Error Metrics (3 metrics)

#### 12. **alert_history_proxy_classification_errors_total**
- **Type**: Counter
- **Labels**: `error_type`
- **Description**: Classification pipeline errors
- **Usage**: Track LLM failures

#### 13. **alert_history_proxy_filtering_errors_total**
- **Type**: Counter
- **Labels**: `error_type`
- **Description**: Filtering pipeline errors
- **Usage**: Track filter failures

#### 14. **alert_history_proxy_publishing_errors_total**
- **Type**: Counter
- **Labels**: `target_type`, `error_type`
- **Description**: Publishing pipeline errors
- **Usage**: Track publishing failures

### 2.4 Performance Metrics (4 metrics)

#### 15. **alert_history_proxy_pipeline_duration_seconds**
- **Type**: Histogram
- **Labels**: `pipeline`
- **Buckets**: `[0.001, 0.01, 0.1, 1.0, 10.0]`
- **Description**: End-to-end pipeline duration
- **Usage**: Monitor total processing time

#### 16. **alert_history_proxy_batch_size**
- **Type**: Histogram
- **Labels**: none
- **Buckets**: `[1, 5, 10, 50, 100, 500]`
- **Description**: Alerts per batch
- **Usage**: Monitor batch sizes

#### 17. **alert_history_proxy_concurrent_requests**
- **Type**: Gauge
- **Labels**: none
- **Description**: Current concurrent requests
- **Usage**: Monitor load

#### 18. **alert_history_proxy_publishing_targets_total**
- **Type**: Gauge
- **Labels**: `target_type`, `health_status`
- **Description**: Publishing targets by type and health
- **Usage**: Monitor target availability

---

## 3. Grafana Dashboard Design

### 3.1 Dashboard Layout (4 rows)

#### Row 1: Overview (4 panels)
1. **Request Rate** (Graph)
   - Metric: `rate(alert_history_proxy_http_requests_total[5m])`
   - Shows: Requests per second

2. **Error Rate** (Graph)
   - Metric: `rate(alert_history_proxy_http_errors_total[5m])`
   - Shows: Errors per second

3. **P95 Latency** (Graph)
   - Metric: `histogram_quantile(0.95, rate(alert_history_proxy_http_request_duration_seconds_bucket[5m]))`
   - Shows: 95th percentile latency

4. **Active Requests** (Gauge)
   - Metric: `alert_history_proxy_http_requests_in_flight`
   - Shows: Current concurrency

#### Row 2: Processing Pipeline (3 panels)
1. **Alerts Processed** (Graph)
   - Metric: `rate(alert_history_proxy_alerts_processed_total[5m])`
   - Shows: Alerts per second by result

2. **Pipeline Durations** (Heatmap)
   - Metrics: Classification, Filtering, Publishing
   - Shows: Duration distribution

3. **Success Rate** (Stat)
   - Metric: `rate(alert_history_proxy_alerts_processed_total{result="success"}[5m]) / rate(alert_history_proxy_alerts_processed_total[5m])`
   - Shows: % successful

#### Row 3: Errors & Failures (3 panels)
1. **Error Types** (Pie chart)
   - Metric: `alert_history_proxy_http_errors_total`
   - Shows: Error distribution

2. **Classification Errors** (Graph)
   - Metric: `rate(alert_history_proxy_classification_errors_total[5m])`
   - Shows: LLM failure rate

3. **Publishing Failures** (Graph)
   - Metric: `rate(alert_history_proxy_publishing_errors_total[5m])`
   - Shows: Publishing failures by target

#### Row 4: Performance Details (3 panels)
1. **Request Size Distribution** (Histogram)
   - Metric: `alert_history_proxy_http_request_size_bytes`
   - Shows: Payload size distribution

2. **Batch Sizes** (Graph)
   - Metric: `alert_history_proxy_batch_size`
   - Shows: Alerts per batch

3. **Publishing Targets Health** (Table)
   - Metric: `alert_history_proxy_publishing_targets_total`
   - Shows: Target status by type

### 3.2 Dashboard Variables

```
- $namespace: Kubernetes namespace
- $pod: Pod name (multi-select)
- $interval: Scrape interval (default: 5m)
```

---

## 4. Alerting Rules

### 4.1 Critical Alerts (P0)

#### 1. HighErrorRate
```yaml
alert: ProxyWebhookHighErrorRate
expr: |
  rate(alert_history_proxy_http_errors_total[5m]) > 10
for: 5m
severity: critical
annotations:
  summary: "High error rate on proxy webhook"
  description: "Error rate is {{ $value }} req/s (threshold: 10)"
```

#### 2. HighLatency
```yaml
alert: ProxyWebhookHighLatency
expr: |
  histogram_quantile(0.95,
    rate(alert_history_proxy_http_request_duration_seconds_bucket[5m])
  ) > 1.0
for: 5m
severity: critical
annotations:
  summary: "High p95 latency on proxy webhook"
  description: "P95 latency is {{ $value }}s (threshold: 1s)"
```

### 4.2 Warning Alerts (P1)

#### 3. ClassificationSlowdown
```yaml
alert: ProxyWebhookClassificationSlow
expr: |
  histogram_quantile(0.95,
    rate(alert_history_proxy_classification_duration_seconds_bucket[5m])
  ) > 5.0
for: 10m
severity: warning
annotations:
  summary: "LLM classification is slow"
  description: "P95 classification time is {{ $value }}s"
```

#### 4. PublishingFailures
```yaml
alert: ProxyWebhookPublishingFailures
expr: |
  rate(alert_history_proxy_publishing_errors_total[5m]) > 1
for: 10m
severity: warning
annotations:
  summary: "Publishing failures detected"
  description: "Publishing error rate: {{ $value }} errors/s"
```

#### 5. LowSuccessRate
```yaml
alert: ProxyWebhookLowSuccessRate
expr: |
  rate(alert_history_proxy_alerts_processed_total{result="success"}[5m])
  / rate(alert_history_proxy_alerts_processed_total[5m]) < 0.95
for: 15m
severity: warning
annotations:
  summary: "Low success rate on proxy webhook"
  description: "Success rate is {{ $value | humanizePercentage }}"
```

### 4.3 Info Alerts (P2)

#### 6. HighConcurrency
```yaml
alert: ProxyWebhookHighConcurrency
expr: |
  alert_history_proxy_http_requests_in_flight > 50
for: 5m
severity: info
annotations:
  summary: "High concurrency on proxy webhook"
  description: "{{ $value }} concurrent requests"
```

---

## 5. Implementation Plan

### 5.1 Metrics Collection (2h)

**File**: `go-app/pkg/metrics/proxy_webhook_metrics.go`

```go
type ProxyWebhookMetrics struct {
    // HTTP metrics
    httpRequestsTotal *prometheus.CounterVec
    httpRequestDuration *prometheus.HistogramVec
    httpRequestSize *prometheus.HistogramVec
    httpResponseSize *prometheus.HistogramVec
    httpRequestsInFlight prometheus.Gauge
    httpErrorsTotal *prometheus.CounterVec

    // Processing metrics
    alertsReceivedTotal *prometheus.CounterVec
    alertsProcessedTotal *prometheus.CounterVec
    classificationDuration *prometheus.HistogramVec
    filteringDuration *prometheus.HistogramVec
    publishingDuration *prometheus.HistogramVec

    // Error metrics
    classificationErrorsTotal *prometheus.CounterVec
    filteringErrorsTotal *prometheus.CounterVec
    publishingErrorsTotal *prometheus.CounterVec

    // Performance metrics
    pipelineDuration *prometheus.HistogramVec
    batchSize prometheus.Histogram
    concurrentRequests prometheus.Gauge
    publishingTargetsTotal *prometheus.GaugeVec
}
```

### 5.2 Dashboard Creation (1h)

**File**: `deployments/grafana/dashboards/proxy_webhook_dashboard.json`

- Export from Grafana UI or use code generation
- Include all 13 panels
- Add variables and templating
- Configure refresh intervals

### 5.3 Alerting Rules (1h)

**File**: `deployments/prometheus/rules/proxy_webhook_alerts.yaml`

```yaml
groups:
  - name: proxy_webhook
    interval: 30s
    rules:
      - alert: ProxyWebhookHighErrorRate
        # ... (6 alerts total)
```

### 5.4 Integration Testing (1h)

- Generate test load
- Verify metrics collection
- Test dashboard queries
- Validate alert firing

---

## 6. Success Criteria

### 6.1 Metrics Coverage

- [x] 18+ Prometheus metrics implemented
- [ ] All metrics tested
- [ ] Metrics documented
- [ ] No performance impact (< 100µs overhead)

### 6.2 Dashboard Functionality

- [ ] All panels rendering
- [ ] Queries optimized (< 1s)
- [ ] Variables working
- [ ] Auto-refresh enabled

### 6.3 Alerting Reliability

- [ ] All alerts tested
- [ ] No false positives
- [ ] Appropriate thresholds
- [ ] Runbook links added

---

## 7. Comparison with TN-061

| Aspect | TN-061 | TN-062 | Status |
|--------|--------|--------|--------|
| Prometheus Metrics | 15 | 18+ | Better ⬆️ |
| Grafana Panels | 10 | 13 | Better ⬆️ |
| Alert Rules | 4 | 6 | Better ⬆️ |
| Coverage | Good | Excellent | Better ⬆️ |

---

## 8. Timeline

| Task | Duration | Status |
|------|----------|--------|
| Metrics implementation | 2h | 🔄 In progress |
| Dashboard creation | 1h | ⏳ Pending |
| Alerting rules | 1h | ⏳ Pending |
| Testing & validation | 1h | ⏳ Pending |
| **Total** | **5h** | **20% complete** |

---

## 9. Next Steps

1. ✅ Create observability plan
2. 🔄 Implement Prometheus metrics
3. ⏳ Create Grafana dashboard
4. ⏳ Configure alerting rules
5. ⏳ Test and validate
6. ⏳ Document

---

**Status**: 🔄 IN PROGRESS
**Target**: 18+ metrics + Dashboard + 6 alerts
**Timeline**: 5h estimated
