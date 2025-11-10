# Alert History Metrics Runbook

**TN-181 Metrics Audit & Unification**
**Version:** 1.0
**Last Updated:** 2025-10-10

---

## Purpose

This runbook provides operational procedures for **monitoring, troubleshooting, and maintaining** Alert History Prometheus metrics. Use this guide for incident response, performance tuning, and routine health checks.

---

## Table of Contents

1. [Quick Health Check](#quick-health-check)
2. [Common Alerts & Resolution](#common-alerts--resolution)
3. [Metrics Debugging](#metrics-debugging)
4. [Performance Issues](#performance-issues)
5. [Capacity Planning](#capacity-planning)
6. [Incident Response](#incident-response)
7. [Maintenance Tasks](#maintenance-tasks)

---

## Quick Health Check

### 1-Minute Health Dashboard

Run these queries in Prometheus/Grafana to assess system health:

```promql
# 1. HTTP Request Rate (should be > 0)
rate(alert_history_technical_http_requests_total[5m])

# 2. HTTP Error Rate (should be < 5%)
(
  sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m]))
  /
  sum(rate(alert_history_technical_http_requests_total[5m]))
) * 100

# 3. P95 Latency (should be < 500ms)
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m]))

# 4. Database Connections (should be < 90% utilization)
alert_history_infra_db_connections_active
/
(alert_history_infra_db_connections_active + alert_history_infra_db_connections_idle) * 100

# 5. Circuit Breaker State (should be 0 = CLOSED)
alert_history_technical_llm_cb_state
```

**Expected Results:**
- HTTP Request Rate: > 0 req/s
- HTTP Error Rate: < 5%
- P95 Latency: < 500ms
- DB Connection Utilization: < 90%
- Circuit Breaker State: 0 (CLOSED)

---

## Common Alerts & Resolution

### 🔴 Critical: High HTTP Error Rate

**Alert Definition:**
```promql
(
  sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m]))
  /
  sum(rate(alert_history_technical_http_requests_total[5m]))
) > 0.05
```

**Symptoms:**
- HTTP error rate > 5%
- Increased 4xx or 5xx responses
- Users reporting failures

**Investigation Steps:**

1. **Identify Error Type (4xx vs. 5xx)**
   ```promql
   # 4xx errors (client errors)
   sum(rate(alert_history_technical_http_requests_total{status_code=~"4.."}[5m])) by (path, status_code)

   # 5xx errors (server errors)
   sum(rate(alert_history_technical_http_requests_total{status_code=~"5.."}[5m])) by (path, status_code)
   ```

2. **Check Affected Endpoints**
   ```promql
   topk(5, sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m])) by (path))
   ```

3. **Check Backend Health**
   - Database connectivity: `alert_history_infra_db_errors_total`
   - LLM service: `alert_history_technical_llm_cb_state`
   - Cache: `alert_history_infra_cache_errors_total`

**Resolution:**

- **If 4xx errors (e.g., 400, 404):**
  - Check client requests (API contract changes?)
  - Review recent deployments
  - Validate input validation logic

- **If 5xx errors (e.g., 500, 503):**
  - Check application logs: `kubectl logs -n alert-history deployment/alert-history --tail=100`
  - Restart unhealthy pods: `kubectl rollout restart deployment/alert-history -n alert-history`
  - Scale up if overloaded: `kubectl scale deployment/alert-history --replicas=5 -n alert-history`

**Escalation:** If errors persist > 15 minutes, escalate to on-call engineer.

---

### 🟠 Warning: High HTTP Latency

**Alert Definition:**
```promql
histogram_quantile(0.95, rate(alert_history_technical_http_request_duration_seconds_bucket[5m])) > 1.0
```

**Symptoms:**
- P95 latency > 1s
- Slow API responses
- Users experiencing delays

**Investigation Steps:**

1. **Identify Slow Endpoints**
   ```promql
   histogram_quantile(0.95, sum(rate(alert_history_technical_http_request_duration_seconds_bucket[5m])) by (path, le))
   ```

2. **Check Database Query Performance**
   ```promql
   histogram_quantile(0.95, rate(alert_history_infra_repository_query_duration_seconds_bucket[5m]))
   ```

3. **Check LLM Call Latency**
   ```promql
   histogram_quantile(0.95, rate(alert_history_technical_llm_cb_call_duration_seconds_bucket[5m]))
   ```

4. **Check Resource Utilization**
   - CPU: `rate(container_cpu_usage_seconds_total{pod=~"alert-history.*"}[5m])`
   - Memory: `container_memory_usage_bytes{pod=~"alert-history.*"}`

**Resolution:**

- **If database queries are slow:**
  - Check for missing indexes: Review query plans in PostgreSQL
  - Optimize slow queries: Analyze `pg_stat_statements`
  - Scale database: Increase connection pool size or add read replicas

- **If LLM calls are slow:**
  - Check LLM service health
  - Adjust circuit breaker thresholds (`slow_call_duration`)
  - Consider caching LLM responses

- **If CPU/memory is high:**
  - Scale horizontally: `kubectl scale deployment/alert-history --replicas=5 -n alert-history`
  - Optimize application code (profiling)

**Escalation:** If latency > 2s for > 30 minutes, escalate to on-call engineer.

---

### 🟡 Warning: Database Connection Pool Exhaustion

**Alert Definition:**
```promql
(
  alert_history_infra_db_connections_active
  /
  (alert_history_infra_db_connections_active + alert_history_infra_db_connections_idle)
) > 0.9
```

**Symptoms:**
- Connection utilization > 90%
- Slow database queries
- Connection timeout errors

**Investigation Steps:**

1. **Check Current Connection Usage**
   ```promql
   alert_history_infra_db_connections_active
   alert_history_infra_db_connections_idle
   ```

2. **Check Connection Wait Time**
   ```promql
   histogram_quantile(0.95, rate(alert_history_infra_db_connection_wait_duration_seconds_bucket[5m]))
   ```

3. **Identify Long-Running Queries (PostgreSQL)**
   ```sql
   SELECT pid, usename, state, now() - query_start AS duration, query
   FROM pg_stat_activity
   WHERE state != 'idle'
   ORDER BY duration DESC
   LIMIT 10;
   ```

**Resolution:**

1. **Immediate (< 5 minutes):**
   - Increase max connections in config: Edit `config.yaml` → `database.max_connections`
   - Restart service: `kubectl rollout restart deployment/alert-history -n alert-history`

2. **Short-term (< 1 hour):**
   - Kill long-running queries: `SELECT pg_terminate_backend(<pid>);`
   - Optimize connection pooling settings (idle timeout, max lifetime)

3. **Long-term:**
   - Add database read replicas
   - Implement connection pooling at application level (PgBouncer)
   - Optimize queries to reduce connection hold time

**Escalation:** If connection pool exhaustion persists > 10 minutes, escalate to DBA.

---

### 🔴 Critical: Circuit Breaker Open

**Alert Definition:**
```promql
alert_history_technical_llm_cb_state == 1
```

**Symptoms:**
- LLM circuit breaker is open
- LLM enrichment disabled
- Alerts not being classified

**Investigation Steps:**

1. **Check Circuit Breaker Metrics**
   ```promql
   # Failure rate
   rate(alert_history_technical_llm_cb_failures_total[5m])

   # Failures by error type
   sum(rate(alert_history_technical_llm_cb_failures_total[5m])) by (error_type)
   ```

2. **Check LLM Service Health**
   - Service availability: `curl http://llm-service:8080/health`
   - Network connectivity: `ping llm-service`
   - DNS resolution: `nslookup llm-service`

3. **Check LLM Latency**
   ```promql
   histogram_quantile(0.95, rate(alert_history_technical_llm_cb_call_duration_seconds_bucket[5m]))
   ```

**Resolution:**

1. **If LLM service is down:**
   - Restart LLM service: `kubectl rollout restart deployment/llm-service -n alert-history`
   - Check LLM service logs: `kubectl logs -n alert-history deployment/llm-service --tail=100`

2. **If LLM service is slow:**
   - Scale LLM service: `kubectl scale deployment/llm-service --replicas=3 -n alert-history`
   - Adjust circuit breaker thresholds (increase `slow_call_duration`)

3. **If circuit breaker threshold is too sensitive:**
   - Review circuit breaker config: `failure_threshold`, `timeout`, `slow_call_duration`
   - Adjust thresholds if false positives

4. **Manual Circuit Breaker Reset (Emergency Only):**
   - Not recommended (circuit breaker will auto-recover in HALF_OPEN state)
   - If needed: Restart alert-history service

**Escalation:** If circuit breaker remains open > 30 minutes, escalate to ML/LLM team.

---

### 🟠 Warning: Publishing Failures

**Alert Definition:**
```promql
(
  sum(rate(alert_history_business_publishing_failed_total[5m]))
  /
  (sum(rate(alert_history_business_publishing_success_total[5m])) + sum(rate(alert_history_business_publishing_failed_total[5m])))
) > 0.1
```

**Symptoms:**
- Publishing failure rate > 10%
- Alerts not reaching destinations (Slack, PagerDuty, etc.)

**Investigation Steps:**

1. **Identify Failed Destinations**
   ```promql
   sum(rate(alert_history_business_publishing_failed_total[5m])) by (destination)
   ```

2. **Check Error Types**
   ```promql
   sum(rate(alert_history_business_publishing_failed_total[5m])) by (error_type)
   ```

3. **Check Destination Health**
   - Slack: `curl https://slack.com/api/api.test`
   - Webhook: `curl -I https://webhook.example.com/health`
   - PagerDuty: Check PagerDuty status page

**Resolution:**

- **If timeout errors:**
  - Increase timeout in config: Edit `config.yaml` → `publishing.timeout`
  - Check network latency to destination

- **If 4xx errors (authentication/authorization):**
  - Verify API keys/tokens
  - Check destination API docs for changes
  - Rotate credentials if expired

- **If 5xx errors (destination service down):**
  - Contact destination service provider
  - Enable retry logic (already in place)
  - Consider failover destinations

**Escalation:** If publishing failures > 50% for > 15 minutes, escalate to integrations team.

---

## Metrics Debugging

### Missing Metrics

**Problem:** Expected metric is not showing up in Prometheus.

**Checklist:**

1. ✅ **Check if metric is registered**
   ```bash
   curl http://localhost:9090/metrics | grep "alert_history_<metric_name>"
   ```

2. ✅ **Check MetricsRegistry initialization**
   - Verify `metrics.DefaultRegistry()` is called in `main.go`
   - Check for registration errors in logs

3. ✅ **Check metric naming**
   - Follow naming convention: `<namespace>_<category>_<subsystem>_<name>_<unit>`
   - Use `ValidateMetricName()` to verify

4. ✅ **Check Prometheus scrape targets**
   - Prometheus UI → Status → Targets
   - Ensure alert-history endpoint is UP

5. ✅ **Check Prometheus scrape interval**
   - Metrics may not appear immediately (default: 15s scrape interval)

---

### High Cardinality Warning

**Problem:** Prometheus alerting: "High cardinality detected for metric X"

**Cause:** Too many unique label combinations (> 10,000).

**Investigation:**
```promql
# Check label cardinality
count by(__name__) ({__name__=~"alert_history.*"})

# Identify high-cardinality labels
topk(10, count by(path) (alert_history_technical_http_requests_total))
```

**Resolution:**

1. **Implement Path Normalization**
   - Use `PathNormalizer` middleware (already in place)
   - Replace dynamic segments (UUIDs, IDs) with placeholders

2. **Review Label Usage**
   - Remove unnecessary labels
   - Aggregate labels at query time (not at metric registration)

3. **Use Recording Rules**
   - Pre-aggregate high-cardinality metrics with recording rules

---

### Stale Metrics

**Problem:** Metric shows old/stale data.

**Investigation:**

1. **Check metric timestamp**
   ```bash
   curl http://localhost:9090/metrics | grep "alert_history_<metric_name>"
   # Look for timestamp (if present)
   ```

2. **Check Prometheus retention**
   - Prometheus UI → Status → Runtime & Build Information
   - Default retention: 15 days

3. **Check scrape interval**
   - Prometheus UI → Status → Configuration
   - Look for `scrape_interval` (default: 15s)

**Resolution:**

- If data is truly stale: Restart alert-history service
- If Prometheus retention issue: Extend retention period
- If scrape interval too long: Reduce interval (e.g., 5s)

---

## Performance Issues

### Slow PromQL Queries

**Problem:** Grafana dashboards loading slowly.

**Investigation:**

1. **Identify Slow Queries**
   - Grafana → Explore → Query Inspector
   - Look for queries with > 1s execution time

2. **Check Query Complexity**
   - Avoid `count()` on raw metrics (use recording rules)
   - Use `sum()` instead of `count()` where possible
   - Limit time ranges (e.g., `[5m]` instead of `[1d]`)

**Resolution:**

1. **Use Recording Rules**
   - Pre-compute expensive queries
   - Example:
     ```yaml
     - record: alert_history:http:error_rate:5m
       expr: |
         sum(rate(alert_history_technical_http_requests_total{status_code=~"[45].."}[5m]))
         /
         sum(rate(alert_history_technical_http_requests_total[5m]))
     ```

2. **Optimize Query Structure**
   - Use `by()` to reduce cardinality
   - Avoid regex in labels (`{label=~".*"}`)
   - Use `topk()` instead of listing all series

3. **Increase Prometheus Resources**
   - CPU: Increase CPU allocation for Prometheus pod
   - Memory: Increase memory allocation (especially for high cardinality)

---

### Prometheus High Memory Usage

**Problem:** Prometheus pod OOMKilled or memory usage > 80%.

**Investigation:**

1. **Check Memory Usage**
   ```bash
   kubectl top pod -n monitoring prometheus-0
   ```

2. **Check Metrics Count**
   ```promql
   count({__name__=~".+"})
   ```

3. **Identify High-Cardinality Metrics**
   ```promql
   topk(20, count by(__name__) ({__name__=~".+"}))
   ```

**Resolution:**

1. **Reduce Cardinality**
   - Implement path normalization
   - Remove unused labels
   - Delete stale metrics: `curl -X POST http://prometheus:9090/api/v1/admin/tsdb/delete_series?match[]={__name__=~"old_metric.*"}`

2. **Increase Retention Period**
   - Reduce retention from 15d to 7d (if acceptable)
   - Edit Prometheus config: `--storage.tsdb.retention.time=7d`

3. **Increase Prometheus Memory**
   - Edit Helm values: `prometheus.resources.memory=8Gi`
   - Redeploy: `helm upgrade prometheus -n monitoring`

---

## Capacity Planning

### Request Volume Forecasting

**Query:**
```promql
# Week-over-week growth rate
(
  sum(rate(alert_history_technical_http_requests_total[7d]))
  /
  sum(rate(alert_history_technical_http_requests_total[7d] offset 7d))
) * 100 - 100
```

**Interpretation:**
- < 10% growth: Normal, no action needed
- 10-30% growth: Plan capacity increase in 30 days
- > 30% growth: Immediate capacity planning required

**Actions:**
- Scale horizontally (add replicas)
- Optimize database queries
- Review caching strategy

---

### Database Connection Pool Sizing

**Query:**
```promql
# Max connections used in past 7 days
max_over_time(alert_history_infra_db_connections_active[7d])

# Average connections used
avg_over_time(alert_history_infra_db_connections_active[7d])
```

**Recommendation:**
```
max_connections = max_observed * 1.5 + 10
```

**Example:**
- Max observed: 40
- Recommended: 40 * 1.5 + 10 = 70 connections

---

## Incident Response

### Incident Severity Matrix

| Severity | HTTP Error Rate | P95 Latency | DB Utilization | Circuit Breaker |
|----------|-----------------|-------------|----------------|-----------------|
| **SEV-1** | > 25% | > 5s | > 95% | Open > 1h |
| **SEV-2** | 10-25% | 2-5s | 90-95% | Open > 30m |
| **SEV-3** | 5-10% | 1-2s | 80-90% | Open > 15m |
| **SEV-4** | < 5% | < 1s | < 80% | CLOSED |

### Incident Response Flow

1. **Acknowledge (< 5 min)**
   - Acknowledge alert in PagerDuty
   - Check #incidents Slack channel

2. **Investigate (< 15 min)**
   - Run Quick Health Check
   - Identify affected component (HTTP/DB/LLM/Publishing)
   - Gather evidence (metrics, logs)

3. **Mitigate (< 30 min)**
   - Follow runbook resolution steps
   - Scale if needed
   - Restart if needed

4. **Resolve (< 1 hour)**
   - Verify metrics return to normal
   - Monitor for 15 minutes
   - Mark incident resolved

5. **Post-Incident (< 24 hours)**
   - Write post-mortem
   - Identify root cause
   - Create action items

---

## Maintenance Tasks

### Weekly Health Check

**Schedule:** Every Monday, 10:00 AM UTC

**Tasks:**
1. ✅ Review all alert thresholds (still relevant?)
2. ✅ Check Prometheus retention (enough storage?)
3. ✅ Verify recording rules are working
4. ✅ Review high-cardinality metrics (any new issues?)
5. ✅ Check Grafana dashboard health (all panels loading?)

---

### Monthly Capacity Review

**Schedule:** First Monday of each month

**Tasks:**
1. ✅ Review request volume trends (week-over-week growth)
2. ✅ Analyze database connection pool usage (adjust max_connections?)
3. ✅ Review LLM circuit breaker metrics (adjust thresholds?)
4. ✅ Check publishing destinations (any new failures?)
5. ✅ Update capacity planning forecast

---

### Quarterly Metrics Audit

**Schedule:** Every 3 months

**Tasks:**
1. ✅ Audit all metrics (are they still useful?)
2. ✅ Remove unused metrics (reduce cardinality)
3. ✅ Update metric documentation (add new metrics, remove old)
4. ✅ Review alert thresholds (adjust based on historical data)
5. ✅ Update runbook (new scenarios, resolutions)

---

## References

- [TN-181 Metrics Naming Guide](METRICS_NAMING_GUIDE.md)
- [PromQL Examples](PROMQL_EXAMPLES.md)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)

---

**Runbook Owner:** SRE Team
**Last Review:** 2025-10-10
**Next Review:** 2026-01-10

**Questions? Contact:** #sre-team, #observability
