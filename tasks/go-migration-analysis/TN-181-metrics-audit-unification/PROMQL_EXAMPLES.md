# PromQL Query Examples for Alert History Metrics

**TN-181 Metrics Audit & Unification**
**Version:** 1.0
**Last Updated:** 2025-10-10

---

## Table of Contents

1. [Business Metrics](#business-metrics)
2. [Technical Metrics](#technical-metrics)
3. [Infrastructure Metrics](#infrastructure-metrics)
4. [Dashboards & Alerts](#dashboards--alerts)
5. [Advanced Queries](#advanced-queries)
6. [Troubleshooting](#troubleshooting)

---

## Business Metrics

### Alert Processing

#### Total Alerts Processed (Rate)
```promql
# Alerts processed per second (5-minute average)
rate(alert_history_business_alerts_processed_total[5m])

# Alerts processed per minute (grouped by source)
sum(rate(alert_history_business_alerts_processed_total[5m])) by (source) * 60

# Total alerts processed in last hour
increase(alert_history_business_alerts_processed_total[1h])
```

#### Alert Enrichment Success Rate
```promql
# Success rate for enriched mode
sum(rate(alert_history_business_alerts_enriched_total{result="success"}[5m]))
/
sum(rate(alert_history_business_alerts_enriched_total[5m])) * 100

# Enrichment success rate by mode
sum(rate(alert_history_business_alerts_enriched_total{result="success"}[5m])) by (mode)
/
sum(rate(alert_history_business_alerts_enriched_total[5m])) by (mode) * 100
```

#### Alert Filtering Effectiveness
```promql
# Percentage of alerts blocked
sum(rate(alert_history_business_alerts_filtered_total{result="blocked"}[5m]))
/
sum(rate(alert_history_business_alerts_filtered_total[5m])) * 100

# Blocked alerts by reason
sum(rate(alert_history_business_alerts_filtered_total{result="blocked"}[5m])) by (reason)

# Top 5 block reasons
topk(5,
  sum(rate(alert_history_business_alerts_filtered_total{result="blocked"}[5m])) by (reason)
)
```

### LLM Metrics

#### LLM Classification Rate
```promql
# LLM classifications per second
rate(alert_history_business_llm_classifications_total[5m])

# Classifications by severity
sum(rate(alert_history_business_llm_classifications_total[5m])) by (severity)

# High-confidence critical alerts
sum(rate(alert_history_business_llm_classifications_total{severity="critical", confidence="high"}[5m]))
```

#### LLM Confidence Score Distribution
```promql
# Average confidence score
avg(alert_history_business_llm_confidence_score)

# P50, P90, P95, P99 confidence scores
histogram_quantile(0.50, rate(alert_history_business_llm_confidence_score_bucket[5m]))
histogram_quantile(0.90, rate(alert_history_business_llm_confidence_score_bucket[5m]))
histogram_quantile(0.95, rate(alert_history_business_llm_confidence_score_bucket[5m]))
histogram_quantile(0.99, rate(alert_history_business_llm_confidence_score_bucket[5m]))
```

#### LLM Recommendation Rate
```promql
# Recommendations generated per second
rate(alert_history_business_llm_recommendations_total[5m])

# Ratio of recommendations to classifications
rate(alert_history_business_llm_recommendations_total[5m])
/
rate(alert_history_business_llm_classifications_total[5m]) * 100
```

### Publishing Metrics

#### Publishing Success Rate
```promql
# Overall success rate
sum(rate(alert_history_business_publishing_success_total[5m]))
/
(sum(rate(alert_history_business_publishing_success_total[5m])) + sum(rate(alert_history_business_publishing_failed_total[5m]))) * 100

# Success rate by destination
sum(rate(alert_history_business_publishing_success_total[5m])) by (destination)
/
(sum(rate(alert_history_business_publishing_success_total[5m])) by (destination) + sum(rate(alert_history_business_publishing_failed_total[5m])) by (destination)) * 100
```

#### Publishing Latency
```promql
# P95 publishing duration (all destinations)
histogram_quantile(0.95, sum(rate(alert_history_business_publishing_duration_seconds_bucket[5m])) by (le))

# P95 publishing duration by destination
histogram_quantile(0.95, sum(rate(alert_history_business_publishing_duration_seconds_bucket[5m])) by (destination, le))

# Average publishing duration
rate(alert_history_business_publishing_duration_seconds_sum[5m])
/
rate(alert_history_business_publishing_duration_seconds_count[5m])
```

#### Publishing Errors
```promql
# Publishing errors by destination and error type
sum(rate(alert_history_business_publishing_failed_total[5m])) by (destination, error_type)

# Top 5 error types
topk(5,
  sum(rate(alert_history_business_publishing_failed_total[5m])) by (error_type)
)

# Publishing error rate (errors per second)
sum(rate(alert_history_business_publishing_failed_total[5m]))
```

---

## Technical Metrics

### HTTP API

#### Request Rate
```promql
# Total HTTP requests per second
rate(alert_history_technical_http_requests_total[5m])

# Requests per second by method
sum(rate(alert_history_technical_http_requests_total[5m])) by (method)

# Requests per second by endpoint (normalized path)
sum(rate(alert_history_technical_http_requests_total[5m])) by (path)
```

#### HTTP Latency
```promql
# P50, P95, P99 HTTP latency
histogram_quantile(0.50, rate(alert_history_technical_http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(alert_history_technical_http_request_duration_seconds_bucket[5m]))

# P95 latency by endpoint
histogram_quantile(0.95, sum(rate(alert_history_technical_http_request_duration_seconds_bucket[5m])) by (path, le))

# Average request duration
rate(alert_history_technical_http_request_duration_seconds_sum[5m])
/
rate(alert_history_technical_http_request_duration_seconds_count[5m])
```

#### HTTP Error Rate
```promql
# 4xx error rate
sum(rate(alert_history_technical_http_requests_total{status_code=~"4.."}[5m]))
/
sum(rate(alert_history_technical_http_requests_total[5m])) * 100

# 5xx error rate
sum(rate(alert_history_technical_http_requests_total{status_code=~"5.."}[5m]))
/
sum(rate(alert_history_technical_http_requests_total[5m])) * 100

# Error rate by endpoint
sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m])) by (path)
```

#### HTTP Response Size
```promql
# Average response size (bytes)
rate(alert_history_technical_http_response_size_bytes_sum[5m])
/
rate(alert_history_technical_http_response_size_bytes_count[5m])

# P95 response size
histogram_quantile(0.95, rate(alert_history_technical_http_response_size_bytes_bucket[5m]))

# Total bandwidth (bytes per second)
rate(alert_history_technical_http_response_size_bytes_sum[5m])
```

### LLM Circuit Breaker

#### Circuit Breaker State
```promql
# Current state (0=closed, 1=open, 2=half-open)
alert_history_technical_llm_cb_state

# State changes per hour
increase(alert_history_technical_llm_cb_state_changes_total[1h])

# Time in open state (minutes in last hour)
sum_over_time(alert_history_technical_llm_cb_state{state="open"}[1h]) * 60 / 3600
```

#### Circuit Breaker Failures
```promql
# Failure rate
rate(alert_history_technical_llm_cb_failures_total[5m])

# Failure rate by error type
sum(rate(alert_history_technical_llm_cb_failures_total[5m])) by (error_type)

# Success vs. failure ratio
rate(alert_history_technical_llm_cb_successes_total[5m])
/
rate(alert_history_technical_llm_cb_failures_total[5m])
```

#### Circuit Breaker Blocked Requests
```promql
# Blocked requests per second
rate(alert_history_technical_llm_cb_requests_blocked_total[5m])

# Percentage of requests blocked
rate(alert_history_technical_llm_cb_requests_blocked_total[5m])
/
(rate(alert_history_technical_llm_cb_successes_total[5m]) + rate(alert_history_technical_llm_cb_failures_total[5m]) + rate(alert_history_technical_llm_cb_requests_blocked_total[5m])) * 100
```

#### Circuit Breaker Latency
```promql
# P95 LLM call duration
histogram_quantile(0.95, rate(alert_history_technical_llm_cb_call_duration_seconds_bucket[5m]))

# Slow call rate (>3s)
rate(alert_history_technical_llm_cb_slow_calls_total[5m])

# Percentage of slow calls
rate(alert_history_technical_llm_cb_slow_calls_total[5m])
/
(rate(alert_history_technical_llm_cb_successes_total[5m]) + rate(alert_history_technical_llm_cb_failures_total[5m])) * 100
```

### Filter Engine

#### Filter Operations
```promql
# Filters applied per second
rate(alert_history_technical_filter_alerts_filtered_total[5m])

# Filters by action (pass, block, modify)
sum(rate(alert_history_technical_filter_alerts_filtered_total[5m])) by (action)

# Filter match rate
rate(alert_history_technical_filter_alerts_filtered_total{action="block"}[5m])
/
rate(alert_history_technical_filter_alerts_filtered_total[5m]) * 100
```

### Enrichment Mode

#### Mode Switches
```promql
# Mode switches per hour
increase(alert_history_technical_enrichment_mode_switches_total[1h])

# Current enrichment mode (0=disabled, 1=enriched, 2=transparent_recommendations)
alert_history_technical_enrichment_mode_status

# Time in each mode (percentage over 24h)
avg_over_time(alert_history_technical_enrichment_mode_status{mode="enriched"}[24h]) * 100
```

---

## Infrastructure Metrics

### Database Connection Pool

#### Connection Utilization
```promql
# Active connections (current)
alert_history_infra_db_connections_active

# Connection utilization percentage
alert_history_infra_db_connections_active
/
(alert_history_infra_db_connections_active + alert_history_infra_db_connections_idle) * 100

# Idle connections
alert_history_infra_db_connections_idle
```

#### Connection Wait Time
```promql
# P95 connection wait time
histogram_quantile(0.95, rate(alert_history_infra_db_connection_wait_duration_seconds_bucket[5m]))

# Average connection wait time
rate(alert_history_infra_db_connection_wait_duration_seconds_sum[5m])
/
rate(alert_history_infra_db_connection_wait_duration_seconds_count[5m])

# Connections acquired per second
rate(alert_history_infra_db_connections_total[5m])
```

#### Database Query Performance
```promql
# P95 query duration
histogram_quantile(0.95, rate(alert_history_infra_db_query_duration_seconds_bucket[5m]))

# P95 query duration by operation
histogram_quantile(0.95, sum(rate(alert_history_infra_db_query_duration_seconds_bucket[5m])) by (operation, le))

# Queries per second
rate(alert_history_infra_db_queries_total[5m])

# Queries per second by operation
sum(rate(alert_history_infra_db_queries_total[5m])) by (operation)
```

#### Database Errors
```promql
# Database error rate
rate(alert_history_infra_db_errors_total[5m])

# Error rate by type
sum(rate(alert_history_infra_db_errors_total[5m])) by (error_type)

# Connection errors vs. query errors
sum(rate(alert_history_infra_db_errors_total{error_type="connection"}[5m]))
sum(rate(alert_history_infra_db_errors_total{error_type="query"}[5m]))
```

### Cache

#### Cache Hit Rate
```promql
# Cache hit rate (percentage)
sum(rate(alert_history_infra_cache_hits_total[5m]))
/
(sum(rate(alert_history_infra_cache_hits_total[5m])) + sum(rate(alert_history_infra_cache_misses_total[5m]))) * 100

# Cache hit rate by cache type
sum(rate(alert_history_infra_cache_hits_total[5m])) by (cache_type)
/
(sum(rate(alert_history_infra_cache_hits_total[5m])) by (cache_type) + sum(rate(alert_history_infra_cache_misses_total[5m])) by (cache_type)) * 100
```

#### Cache Evictions
```promql
# Cache evictions per second
rate(alert_history_infra_cache_evictions_total[5m])

# Evictions by reason
sum(rate(alert_history_infra_cache_evictions_total[5m])) by (reason)
```

### Repository

#### Repository Query Performance
```promql
# P95 repository query duration
histogram_quantile(0.95, rate(alert_history_infra_repository_query_duration_seconds_bucket[5m]))

# P95 query duration by operation (GetTopAlerts, GetFlappingAlerts, etc.)
histogram_quantile(0.95, sum(rate(alert_history_infra_repository_query_duration_seconds_bucket[5m])) by (operation, le))

# Average query duration
rate(alert_history_infra_repository_query_duration_seconds_sum[5m])
/
rate(alert_history_infra_repository_query_duration_seconds_count[5m])
```

#### Repository Errors
```promql
# Repository error rate
rate(alert_history_infra_repository_query_errors_total[5m])

# Error rate by operation and error type
sum(rate(alert_history_infra_repository_query_errors_total[5m])) by (operation, error_type)
```

#### Repository Result Sizes
```promql
# P95 result size (number of rows)
histogram_quantile(0.95, rate(alert_history_infra_repository_query_results_total_bucket[5m]))

# Average result size by operation
rate(alert_history_infra_repository_query_results_total_sum[5m]) by (operation)
/
rate(alert_history_infra_repository_query_results_total_count[5m]) by (operation)
```

---

## Dashboards & Alerts

### SLI/SLO Queries

#### Availability SLI (HTTP 200-299 / All Requests)
```promql
# Availability SLI (99.9% target)
sum(rate(alert_history_technical_http_requests_total{status_code=~"2.."}[5m]))
/
sum(rate(alert_history_technical_http_requests_total[5m])) * 100
```

#### Latency SLI (P95 < 500ms)
```promql
# Latency SLI (P95 < 500ms)
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m])) < 0.5
```

#### Error Budget (30-day window)
```promql
# Error budget remaining (%)
(1 -
  (sum(increase(alert_history_technical_http_requests_total{status_code=~"[45].."}[30d]))
  /
  sum(increase(alert_history_technical_http_requests_total[30d])))
) * 100 / 0.001  # 99.9% SLO = 0.1% error budget
```

### Alert Rules

#### High Error Rate
```promql
# Alert if HTTP error rate > 5% for 5 minutes
(
  sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m]))
  /
  sum(rate(alert_history_technical_http_requests_total[5m]))
) > 0.05
```

#### High Latency
```promql
# Alert if P95 latency > 1s for 5 minutes
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m])) > 1.0
```

#### Database Connection Pool Exhaustion
```promql
# Alert if connection utilization > 90%
(
  alert_history_infra_db_connections_active
  /
  (alert_history_infra_db_connections_active + alert_history_infra_db_connections_idle)
) > 0.9
```

#### Circuit Breaker Open
```promql
# Alert if circuit breaker is open
alert_history_technical_llm_cb_state == 1
```

#### Publishing Failures
```promql
# Alert if publishing failure rate > 10%
(
  sum(rate(alert_history_business_publishing_failed_total[5m]))
  /
  (sum(rate(alert_history_business_publishing_success_total[5m])) + sum(rate(alert_history_business_publishing_failed_total[5m])))
) > 0.1
```

---

## Advanced Queries

### Multi-Metric Analysis

#### End-to-End Alert Processing Time
```promql
# Estimate: (Enrichment Time + Publishing Time)
rate(alert_history_technical_http_request_duration_seconds_sum[5m])
/
rate(alert_history_technical_http_request_duration_seconds_count[5m])
+
rate(alert_history_business_publishing_duration_seconds_sum[5m])
/
rate(alert_history_business_publishing_duration_seconds_count[5m])
```

#### Database Queries Per Alert
```promql
# Average DB queries per alert processed
rate(alert_history_infra_db_queries_total[5m])
/
rate(alert_history_business_alerts_processed_total[5m])
```

#### LLM Classification Accuracy (Proxy: High Confidence %)
```promql
# Percentage of high-confidence classifications
sum(rate(alert_history_business_llm_classifications_total{confidence="high"}[5m]))
/
sum(rate(alert_history_business_llm_classifications_total[5m])) * 100
```

### Capacity Planning

#### Request Growth Rate (Week-over-Week)
```promql
# Week-over-week request growth
(
  sum(rate(alert_history_technical_http_requests_total[7d]))
  /
  sum(rate(alert_history_technical_http_requests_total[7d] offset 7d))
) * 100 - 100
```

#### Database Connection Pool Forecast
```promql
# Predicted max connections needed in 30 days (linear extrapolation)
predict_linear(alert_history_infra_db_connections_active[7d], 30*24*3600)
```

---

## Troubleshooting

### No Data Returned

**Problem:** Query returns no results.

**Checklist:**
1. ✅ Check metric name spelling
2. ✅ Verify time range (use `[5m]` for rate, `[1h]` for increase)
3. ✅ Confirm label selectors match actual data
4. ✅ Check if metric exists: `curl http://localhost:9090/metrics | grep <metric_name>`

**Example:**
```promql
# Check if metric exists
{__name__=~"alert_history.*"}
```

### High Cardinality Issues

**Problem:** Queries are slow due to high label cardinality.

**Solution:** Use aggregation to reduce cardinality:
```promql
# BAD: High cardinality (many unique paths)
alert_history_technical_http_requests_total

# GOOD: Aggregate by method only
sum(rate(alert_history_technical_http_requests_total[5m])) by (method)
```

### Rate() vs. Increase()

**Difference:**
- `rate()`: Per-second average rate over time range
- `increase()`: Total increase over time range

**When to Use:**
- Use `rate()` for "per second" metrics (dashboards, alerts)
- Use `increase()` for "total count" metrics (summaries, reports)

**Example:**
```promql
# Requests per second (use rate)
rate(alert_history_technical_http_requests_total[5m])

# Total requests in last hour (use increase)
increase(alert_history_technical_http_requests_total[1h])
```

---

## References

- [Prometheus Query Functions](https://prometheus.io/docs/prometheus/latest/querying/functions/)
- [PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Alert History Metrics Naming Guide](METRICS_NAMING_GUIDE.md)
- [Grafana Dashboard Variables](https://grafana.com/docs/grafana/latest/variables/)

---

**Questions? Contact:** SRE Team / #observability
