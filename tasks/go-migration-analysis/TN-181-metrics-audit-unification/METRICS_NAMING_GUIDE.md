# Metrics Naming Convention Guide

**TN-181 Metrics Audit & Unification**
**Version:** 1.0
**Last Updated:** 2025-10-10

---

## Overview

This document defines the **official** Prometheus metrics naming convention for the Alert History service. All new metrics MUST follow this taxonomy to ensure consistency, discoverability, and maintainability.

---

## Naming Pattern

```
<namespace>_<category>_<subsystem>_<metric_name>_<unit>
```

### Components

| Component | Required | Description | Examples |
|-----------|----------|-------------|----------|
| **namespace** | ✅ Yes | Service identifier | `alert_history` |
| **category** | ✅ Yes | Metric category (business/technical/infra) | `business`, `technical`, `infra` |
| **subsystem** | ✅ Yes | Functional subsystem | `alerts`, `llm`, `http`, `db`, `cache` |
| **metric_name** | ✅ Yes | Descriptive metric name (snake_case) | `processed`, `duration`, `errors` |
| **unit** | ⚠️ Conditional | Unit suffix (for counters/histograms) | `total`, `seconds`, `bytes`, `info` |

---

## Categories

### 1. Business (`business`)

**Purpose:** Track business KPIs and domain-specific metrics.

**Characteristics:**
- Directly related to product/service value
- Used by product/business teams
- High-level outcomes (alerts processed, recommendations generated)

**Examples:**
```
alert_history_business_alerts_processed_total
alert_history_business_llm_confidence_score
alert_history_business_publishing_duration_seconds
```

**Subsystems:**
- `alerts` - Alert processing metrics
- `llm` - LLM classification/recommendation metrics
- `publishing` - Publishing to external systems
- `filtering` - Alert filtering decisions

---

### 2. Technical (`technical`)

**Purpose:** Track application-level technical metrics.

**Characteristics:**
- Application logic and behavior
- API/HTTP performance
- Feature flags and modes
- Circuit breakers and resilience

**Examples:**
```
alert_history_technical_http_requests_total
alert_history_technical_http_request_duration_seconds
alert_history_technical_llm_cb_state
alert_history_technical_filter_results_total
```

**Subsystems:**
- `http` - HTTP API metrics
- `filter` - Filter engine metrics
- `enrichment` - Enrichment mode metrics
- `llm_cb` - LLM Circuit Breaker metrics

---

### 3. Infrastructure (`infra`)

**Purpose:** Track infrastructure and system-level metrics.

**Characteristics:**
- Database, cache, storage
- Connection pools, queues
- Low-level resource utilization
- Dependencies (PostgreSQL, Redis, etc.)

**Examples:**
```
alert_history_infra_db_connections_active
alert_history_infra_cache_hits_total
alert_history_infra_repository_query_duration_seconds
```

**Subsystems:**
- `db` - Database connection pool metrics
- `cache` - Cache operations (Redis, in-memory)
- `repository` - Data repository metrics
- `queue` - Message queue metrics

---

## Unit Suffixes

| Suffix | Type | Description | Example |
|--------|------|-------------|---------|
| `_total` | Counter | Monotonically increasing counter | `alerts_processed_total` |
| `_seconds` | Histogram | Time duration in seconds | `request_duration_seconds` |
| `_bytes` | Histogram | Size in bytes | `response_size_bytes` |
| `_info` | Gauge | Info/status metric (constant 1) | `build_info` |
| *(none)* | Gauge | Current value | `connections_active` |

**Rules:**
- ✅ Use `_total` for all Counters
- ✅ Use `_seconds` for all time-based Histograms
- ✅ Use `_bytes` for all size-based Histograms
- ❌ Do NOT use `_count`, `_sum`, or `_bucket` (reserved by Prometheus)

---

## Label Guidelines

### Required Labels

Labels are **optional** but highly recommended for dimensionality. Common patterns:

| Metric Type | Common Labels | Example |
|-------------|---------------|---------|
| HTTP | `method`, `path`, `status_code` | `method="GET", path="/api/alerts", status_code="200"` |
| Database | `operation`, `status` | `operation="SELECT", status="success"` |
| LLM | `severity`, `confidence` | `severity="critical", confidence="high"` |
| Publishing | `destination`, `error_type` | `destination="webhook", error_type="timeout"` |

### Label Cardinality

⚠️ **HIGH CARDINALITY WARNING:**
- Avoid dynamic user-generated values (UUIDs, timestamps)
- Limit label value combinations to < 10,000
- Use path normalization for HTTP paths (e.g., `/api/alerts/:id`)

**Example (BAD - High Cardinality):**
```
# DON'T: Dynamic UUIDs in labels
alert_history_technical_http_requests_total{path="/api/alerts/123e4567-e89b-12d3-a456-426614174000"}
```

**Example (GOOD - Normalized Path):**
```
# DO: Normalized path with placeholder
alert_history_technical_http_requests_total{path="/api/alerts/:id"}
```

---

## Quick Reference: Metric Types

| Type | When to Use | Prometheus Function | Grafana Visualization |
|------|-------------|---------------------|----------------------|
| **Counter** | Monotonically increasing (total requests, errors) | `rate()`, `increase()` | Time Series (Rate) |
| **Gauge** | Current value (connections, queue length) | *(none)* | Gauge, Time Series |
| **Histogram** | Distribution (latency, size) | `histogram_quantile()`, `rate()` | Heatmap, Percentiles |
| **Summary** | Similar to Histogram (client-side quantiles) | `*_sum`, `*_count` | Time Series |

---

## Code Examples

### Go: Using MetricsRegistry

```go
import "github.com/vitaliisemenov/alert-history/pkg/metrics"

// Get global registry instance
registry := metrics.DefaultRegistry()

// Record business metric
registry.Business().AlertsProcessedTotal.WithLabelValues("alertmanager").Inc()

// Record technical metric
registry.Technical().HTTP.RecordRequest("GET", "/api/alerts", 200, 0.123)

// Record infrastructure metric
registry.Infra().DB.ConnectionsActive.Set(42)
```

### PromQL: Querying Metrics

```promql
# Rate of alerts processed per second
rate(alert_history_business_alerts_processed_total[5m])

# P95 HTTP latency
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m]))

# Database connections utilization
alert_history_infra_db_connections_active / alert_history_infra_db_connections_max * 100
```

---

## Migration from Legacy Metrics

### Backward Compatibility

For **renamed metrics**, Prometheus recording rules provide backward compatibility:

```yaml
# Recording rule example
- record: alert_history_query_duration_seconds
  expr: alert_history_infra_repository_query_duration_seconds
```

**Timeline:**
- **Phase 1 (2025-Q1):** Dual emission (old + new metrics)
- **Phase 2 (2025-Q2):** New metrics + recording rules
- **Phase 3 (2025-Q3):** Deprecate old metrics (recording rules remain)
- **Phase 4 (2025-Q4):** Remove recording rules (new metrics only)

---

## Validation

### Automated Validation

The `MetricsRegistry` validates metric names at creation time:

```go
registry := metrics.NewMetricsRegistry("alert_history")

// Validates against pattern: <namespace>_<category>_<subsystem>_<name>_<unit>
err := registry.ValidateMetricName(
    metrics.CategoryBusiness,
    "alerts",
    "processed",
    "total",
)
if err != nil {
    log.Fatalf("Invalid metric name: %v", err)
}
```

**Pattern:**
```regex
^alert_history_(business|technical|infra)_[a-z0-9_]+_[a-z0-9_]+(_(total|seconds|bytes|info))?$
```

---

## Best Practices

### ✅ DO

1. **Use MetricsRegistry for all new metrics**
   ```go
   registry := metrics.DefaultRegistry()
   registry.Business().AlertsProcessedTotal.Inc()
   ```

2. **Follow naming convention strictly**
   - `alert_history_business_alerts_processed_total` ✅
   - `alerts_processed` ❌ (missing namespace, category, unit)

3. **Use descriptive subsystem names**
   - `llm_cb` (LLM Circuit Breaker) ✅
   - `cb` ❌ (too vague)

4. **Add helpful Help text**
   ```go
   Help: "Total number of alerts processed by the system, labeled by source"
   ```

5. **Document labels in code comments**
   ```go
   // Labels:
   //   - source: alertmanager, webhook, api
   ```

### ❌ DON'T

1. **Don't create metrics outside MetricsRegistry**
   ```go
   // DON'T
   prometheus.NewCounter(prometheus.CounterOpts{Name: "my_metric_total"})
   ```

2. **Don't use dynamic/high-cardinality labels**
   ```go
   // DON'T
   metric.WithLabelValues(alertID, timestamp) // UUIDs, timestamps
   ```

3. **Don't mix categories**
   ```go
   // DON'T: HTTP metrics in business category
   alert_history_business_http_requests_total ❌
   // DO: HTTP metrics in technical category
   alert_history_technical_http_requests_total ✅
   ```

4. **Don't use abbreviations in metric names**
   - `connections_active` ✅
   - `conn_act` ❌

5. **Don't skip unit suffixes for counters/histograms**
   - `alerts_processed_total` ✅
   - `alerts_processed` ❌

---

## Troubleshooting

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `duplicate metrics collector registration` | Metric with same name already registered | Use unique namespace or check for existing metric |
| `metric name does not match expected pattern` | Invalid naming convention | Follow `<namespace>_<category>_<subsystem>_<name>_<unit>` pattern |
| `label cardinality too high` | Too many unique label combinations | Normalize labels (e.g., path normalization) |

### Debugging

```bash
# List all registered metrics
curl http://localhost:9090/metrics | grep "alert_history"

# Check metric type
curl http://localhost:9090/metrics | grep "# TYPE alert_history_business_alerts_processed_total"

# Validate metric in PromQL
curl -G 'http://localhost:9090/api/v1/query' --data-urlencode 'query=alert_history_business_alerts_processed_total'
```

---

## References

- [Prometheus Naming Best Practices](https://prometheus.io/docs/practices/naming/)
- [Prometheus Metric Types](https://prometheus.io/docs/concepts/metric_types/)
- [Alert History MetricsRegistry GoDoc](../../../go-app/pkg/metrics/registry.go)
- [TN-181 Design Document](design.md)

---

**Questions? Contact:** SRE Team / #observability
