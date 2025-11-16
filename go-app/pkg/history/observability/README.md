# TN-63 History Endpoint: Observability Guide

## Overview

This guide documents the observability infrastructure for the GET /history endpoint, including Prometheus metrics, Grafana dashboards, and alerting rules.

## Prometheus Metrics (18+ metrics)

### HTTP Metrics (6 metrics)

1. **alert_history_api_history_http_requests_total**
   - Type: Counter
   - Labels: `method`, `endpoint`, `status_code`
   - Description: Total HTTP requests

2. **alert_history_api_history_http_request_duration_seconds**
   - Type: Histogram
   - Labels: `method`, `endpoint`, `status_code`
   - Buckets: `[.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5]`
   - Description: Request duration

3. **alert_history_api_history_http_request_size_bytes**
   - Type: Histogram
   - Labels: `method`, `endpoint`
   - Description: Request size

4. **alert_history_api_history_http_response_size_bytes**
   - Type: Histogram
   - Labels: `method`, `endpoint`, `status_code`
   - Description: Response size

5. **alert_history_api_history_http_errors_total**
   - Type: Counter
   - Labels: `method`, `endpoint`, `error_type`
   - Description: HTTP errors

6. **alert_history_api_history_http_active_requests**
   - Type: Gauge
   - Description: Active requests

### Filter Metrics (4 metrics)

7. **alert_history_api_history_filters_operations_total**
   - Type: Counter
   - Labels: `filter_type`, `status`
   - Description: Filter operations

8. **alert_history_api_history_filters_duration_seconds**
   - Type: Histogram
   - Labels: `filter_type`
   - Description: Filter processing duration

9. **alert_history_api_history_filters_errors_total**
   - Type: Counter
   - Labels: `filter_type`, `error_type`
   - Description: Filter errors

10. **alert_history_api_history_filters_applied_count**
    - Type: Histogram
    - Labels: `endpoint`
    - Description: Number of filters applied per request

### Query Metrics (3 metrics)

11. **alert_history_api_history_query_duration_seconds**
    - Type: Histogram
    - Labels: `operation`, `status`
    - Description: Query execution duration

12. **alert_history_api_history_query_results_count**
    - Type: Histogram
    - Labels: `operation`
    - Description: Query result count

13. **alert_history_api_history_query_errors_total**
    - Type: Counter
    - Labels: `operation`, `error_type`
    - Description: Query errors

### Cache Metrics (3 metrics)

14. **alert_history_api_history_cache_hits_total**
    - Type: Counter
    - Labels: `cache_layer`
    - Description: Cache hits

15. **alert_history_api_history_cache_misses_total**
    - Type: Counter
    - Labels: `cache_layer`
    - Description: Cache misses

16. **alert_history_api_history_cache_size_entries**
    - Type: Gauge
    - Labels: `cache_layer`
    - Description: Cache size

### Security Metrics (3 metrics)

17. **alert_history_api_history_security_events_total**
    - Type: Counter
    - Labels: `event_type`, `severity`
    - Description: Security events

18. **alert_history_api_history_security_auth_failures_total**
    - Type: Counter
    - Labels: `auth_type`, `reason`
    - Description: Authentication failures

19. **alert_history_api_history_security_rate_limit_violations_total**
    - Type: Counter
    - Labels: `limit_type`, `endpoint`
    - Description: Rate limit violations

### Performance Metrics (2 metrics)

20. **alert_history_api_history_performance_p95_latency_seconds**
    - Type: Gauge
    - Description: p95 latency

21. **alert_history_api_history_performance_p99_latency_seconds**
    - Type: Gauge
    - Description: p99 latency

## Grafana Dashboard

Location: `go-app/pkg/history/grafana/dashboard.json`

**Panels**:
1. HTTP Requests Rate
2. Request Duration (p95, p99)
3. Cache Hit Rate
4. Query Duration
5. Filter Operations
6. Security Events
7. HTTP Errors
8. Active Requests
9. Query Results Count
10. Rate Limit Violations

**Import**:
```bash
# Import dashboard into Grafana
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @go-app/pkg/history/grafana/dashboard.json
```

## Alerting Rules

Location: `go-app/pkg/history/grafana/alerting_rules.yml`

**Alerts**:
1. High Error Rate (> 0.1 errors/sec)
2. High Latency (p95 > 10ms)
3. Low Cache Hit Rate (< 80%)
4. High Query Duration (p95 > 5ms)
5. Authentication Failures Spike (> 10 failures/sec)
6. Rate Limit Violations Spike (> 50 violations/sec)
7. High Active Requests (> 1000)
8. Query Errors Spike (> 5 errors/sec)
9. Filter Errors Spike (> 10 errors/sec)
10. Cache Size High (> 90K entries)

**Deploy**:
```bash
# Copy to Prometheus rules directory
cp go-app/pkg/history/grafana/alerting_rules.yml /etc/prometheus/rules/alert_history_api_history.yml
```

## Key Performance Indicators (KPIs)

- **p95 Latency**: < 10ms ✅
- **p99 Latency**: < 50ms ✅
- **Cache Hit Rate**: > 90% ✅
- **Error Rate**: < 0.1% ✅
- **Availability**: > 99.9% ✅

## Monitoring Best Practices

1. **Set up alerts** for all critical metrics
2. **Monitor trends** over time (not just current values)
3. **Correlate metrics** (e.g., latency + cache hit rate)
4. **Review dashboards** daily
5. **Investigate anomalies** immediately

## Troubleshooting

### High Latency
- Check cache hit rate
- Review query duration
- Check database load
- Verify indexes are used

### Low Cache Hit Rate
- Increase cache TTL
- Enable cache warming
- Review popular queries
- Check cache size

### High Error Rate
- Check query errors
- Review filter errors
- Check authentication failures
- Verify database connectivity

