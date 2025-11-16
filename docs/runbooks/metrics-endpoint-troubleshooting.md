# Metrics Endpoint Troubleshooting Guide

**Runbook for troubleshooting common issues with the `/metrics` endpoint (TN-65).**

---

## 📋 Table of Contents

1. [Common Issues](#common-issues)
2. [Diagnostic Commands](#diagnostic-commands)
3. [Error Codes](#error-codes)
4. [Performance Issues](#performance-issues)
5. [Rate Limiting Issues](#rate-limiting-issues)
6. [Monitoring & Debugging](#monitoring--debugging)

---

## Common Issues

### Issue: 429 Too Many Requests

**Symptoms:**
- Prometheus scraping fails intermittently
- Response includes `rate_limit_exceeded` error
- `X-RateLimit-Remaining: 0` header

**Causes:**
- Too many Prometheus instances scraping
- Scrape interval too frequent
- Burst of requests

**Solutions:**

1. **Increase Rate Limit**
   ```go
   config.RateLimitPerMinute = 200
   config.RateLimitBurst = 50
   ```

2. **Adjust Scrape Interval**
   ```yaml
   scrape_interval: 30s  # Increase from 15s
   ```

3. **Check Number of Scrapers**
   ```bash
   # Count unique IPs hitting the endpoint
   grep "metrics endpoint request" /var/log/app.log | \
     awk '{print $NF}' | sort -u | wc -l
   ```

4. **Monitor Rate Limit Violations**
   ```promql
   rate(alert_history_metrics_endpoint_errors_total{error_type="rate_limit"}[5m])
   ```

---

### Issue: 408 Request Timeout

**Symptoms:**
- Prometheus scraping times out
- Response includes `context deadline exceeded`
- `X-Metrics-Partial: true` header may be present

**Causes:**
- Too many metrics to gather
- Slow metric collectors
- Network latency
- GatherTimeout too short

**Solutions:**

1. **Increase Gather Timeout**
   ```go
   config.GatherTimeout = 10 * time.Second  // Increase from 5s
   ```

2. **Check Prometheus Scrape Timeout**
   ```yaml
   scrape_timeout: 15s  # Should be > GatherTimeout
   ```

3. **Investigate Slow Collectors**
   ```bash
   # Check which metrics take longest
   curl -s http://localhost:8080/metrics | \
     grep -E "^# HELP|^# TYPE" | head -20
   ```

4. **Enable Caching**
   ```go
   config.CacheEnabled = true
   config.CacheTTL = 15 * time.Second
   ```

5. **Check Response Size**
   ```bash
   curl -s http://localhost:8080/metrics | wc -c
   # If > 10MB, consider increasing MaxResponseSize
   ```

---

### Issue: 500 Internal Server Error

**Symptoms:**
- Prometheus scraping fails completely
- Error message in response body
- No partial metrics returned

**Causes:**
- Metric collection failure
- Registry corruption
- Memory issues

**Solutions:**

1. **Check Application Logs**
   ```bash
   grep "metrics endpoint error" /var/log/app.log | tail -20
   ```

2. **Verify Metric Registration**
   ```go
   // Check for duplicate registrations
   handler.GetRegistry().Gather()
   ```

3. **Check Memory Usage**
   ```bash
   # Check if response size exceeds limit
   curl -s http://localhost:8080/metrics | wc -c
   ```

4. **Restart Application**
   ```bash
   systemctl restart alert-history
   ```

---

### Issue: Slow Response Times

**Symptoms:**
- High P95 latency (>1s)
- Prometheus scrape duration high
- Slow dashboard loading

**Causes:**
- Too many metrics
- No caching enabled
- Slow metric collectors
- High load

**Solutions:**

1. **Enable Caching**
   ```go
   config.CacheEnabled = true
   config.CacheTTL = 15 * time.Second
   ```

2. **Check Metric Count**
   ```bash
   curl -s http://localhost:8080/metrics | \
     grep -c "^[^#]"  # Count metric lines
   ```

3. **Profile Metric Collection**
   ```bash
   # Use pprof
   curl http://localhost:8080/debug/pprof/profile?seconds=30
   ```

4. **Optimize Slow Collectors**
   - Review custom metric collectors
   - Check database query performance
   - Optimize metric calculation

---

### Issue: Missing Metrics

**Symptoms:**
- Expected metrics not in response
- Prometheus not collecting certain metrics
- Metrics appear/disappear

**Causes:**
- Metrics not registered
- Registry not included in gatherer
- Metrics filtered out

**Solutions:**

1. **Verify Metric Registration**
   ```go
   // Check if MetricsRegistry is registered
   handler.RegisterMetricsRegistry(registry)
   ```

2. **Check Gatherer Configuration**
   ```go
   // Ensure DefaultGatherer is included
   gatherers := []prometheus.Gatherer{
       prometheus.DefaultGatherer,  // For promauto metrics
       promRegistry,
   }
   ```

3. **Test Metric Collection**
   ```bash
   curl -s http://localhost:8080/metrics | \
     grep "alert_history_business_webhook_events_total"
   ```

4. **Check Logs for Errors**
   ```bash
   grep "failed to register" /var/log/app.log
   ```

---

## Diagnostic Commands

### Check Endpoint Health

```bash
# Basic health check
curl -v http://localhost:8080/metrics

# Check response headers
curl -I http://localhost:8080/metrics

# Check rate limit headers
curl -v http://localhost:8080/metrics 2>&1 | grep -i "rate"
```

### Check Response Size

```bash
# Get response size
curl -s http://localhost:8080/metrics | wc -c

# Get metric count
curl -s http://localhost:8080/metrics | grep -c "^[^#]"

# Get unique metric names
curl -s http://localhost:8080/metrics | \
  grep "^[^#]" | cut -d'{' -f1 | sort -u | wc -l
```

### Check Performance

```bash
# Measure response time
time curl -s http://localhost:8080/metrics > /dev/null

# Multiple requests
for i in {1..10}; do
  time curl -s http://localhost:8080/metrics > /dev/null
done

# With cache (if enabled)
curl -s http://localhost:8080/metrics > /dev/null  # Warm cache
time curl -s http://localhost:8080/metrics > /dev/null  # Cached
```

### Check Rate Limiting

```bash
# Test rate limit
for i in {1..70}; do
  curl -s http://localhost:8080/metrics > /dev/null
  echo "Request $i"
done
# Should see 429 after ~60 requests

# Check rate limit headers
curl -v http://localhost:8080/metrics 2>&1 | \
  grep -E "X-RateLimit|Retry-After"
```

### Check Logs

```bash
# Recent errors
grep "metrics endpoint error" /var/log/app.log | tail -20

# Request logs
grep "metrics endpoint request" /var/log/app.log | tail -20

# Slow requests (>1s)
grep "metrics endpoint request completed (slow)" /var/log/app.log

# Rate limit violations
grep "rate_limit_exceeded" /var/log/app.log
```

### Prometheus Queries

```promql
# Request rate
rate(alert_history_metrics_endpoint_requests_total[5m])

# Error rate
rate(alert_history_metrics_endpoint_errors_total[5m])

# P95 latency
histogram_quantile(0.95,
  rate(alert_history_metrics_endpoint_request_duration_seconds_bucket[5m])
)

# Active requests
alert_history_metrics_endpoint_active_requests

# Response size
histogram_quantile(0.95,
  rate(alert_history_metrics_endpoint_response_size_bytes_bucket[5m])
)
```

---

## Error Codes

### 200 OK
- **Meaning:** Success
- **Action:** None

### 400 Bad Request
- **Meaning:** Invalid query parameters
- **Action:** Remove query parameters from request

### 404 Not Found
- **Meaning:** Invalid path
- **Action:** Use `/metrics` path

### 405 Method Not Allowed
- **Meaning:** Non-GET request
- **Action:** Use GET method only

### 408 Request Timeout
- **Meaning:** Gathering exceeded timeout
- **Action:**
  - Increase `GatherTimeout`
  - Check for slow collectors
  - Enable caching

### 429 Too Many Requests
- **Meaning:** Rate limit exceeded
- **Action:**
  - Increase `RateLimitPerMinute`
  - Adjust scrape interval
  - Check number of scrapers

### 500 Internal Server Error
- **Meaning:** Internal error
- **Action:**
  - Check application logs
  - Verify metric registration
  - Restart application

---

## Performance Issues

### High Latency

**Diagnosis:**
```bash
# Check P95 latency
curl -s http://localhost:8080/metrics | time

# Check Prometheus scrape duration
promtool query instant 'scrape_duration_seconds{job="alert-history"}'
```

**Solutions:**
1. Enable caching
2. Reduce metric count
3. Optimize slow collectors
4. Increase `GatherTimeout`

### High Memory Usage

**Diagnosis:**
```bash
# Check response size
curl -s http://localhost:8080/metrics | wc -c

# Check memory usage
ps aux | grep alert-history
```

**Solutions:**
1. Reduce metric count
2. Increase `MaxResponseSize` if needed
3. Enable caching to reduce allocations
4. Review buffer pooling

### High CPU Usage

**Diagnosis:**
```bash
# Profile CPU
curl http://localhost:8080/debug/pprof/profile?seconds=30

# Check CPU usage
top -p $(pgrep alert-history)
```

**Solutions:**
1. Enable caching
2. Optimize metric collection
3. Reduce scrape frequency
4. Review custom collectors

---

## Rate Limiting Issues

### Too Many Rate Limit Violations

**Diagnosis:**
```promql
rate(alert_history_metrics_endpoint_errors_total{error_type="rate_limit"}[5m])
```

**Solutions:**
1. Increase `RateLimitPerMinute`
2. Increase `RateLimitBurst`
3. Adjust scrape intervals
4. Reduce number of scrapers

### Rate Limit Too Strict

**Symptoms:**
- Frequent 429 errors
- Prometheus scraping fails

**Solutions:**
```go
config.RateLimitPerMinute = 200  // Increase from 60
config.RateLimitBurst = 50       // Increase from 10
```

### Rate Limit Too Loose

**Symptoms:**
- High request rate
- Potential DoS vulnerability

**Solutions:**
```go
config.RateLimitPerMinute = 30   // Decrease from 60
config.RateLimitBurst = 5        // Decrease from 10
```

---

## Monitoring & Debugging

### Enable Debug Logging

```go
// Set logger with debug level
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
handler.SetLogger(&metricsLoggerAdapter{logger: logger})
```

### Monitor Self-Metrics

Create Grafana dashboard with:
- Request rate
- Error rate
- Latency (P50, P95, P99)
- Active requests
- Response size
- Cache hit rate (if enabled)

### Alert Rules

See [Integration Guide](../guides/metrics-integration.md#alerts) for alert rule examples.

---

## See Also

- [API Documentation](../api/metrics-endpoint.md)
- [Integration Guide](../guides/metrics-integration.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
