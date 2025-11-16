# Metrics Endpoint Integration Guide

**Guide for integrating Prometheus with the `/metrics` endpoint (TN-65).**

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Prometheus Configuration](#prometheus-configuration)
3. [Scraping Configuration](#scraping-configuration)
4. [Service Discovery](#service-discovery)
5. [Rate Limiting Considerations](#rate-limiting-considerations)
6. [Performance Tuning](#performance-tuning)
7. [Monitoring the Endpoint](#monitoring-the-endpoint)
8. [Best Practices](#best-practices)

---

## Overview

The `/metrics` endpoint is designed to be scraped by Prometheus. This guide covers:

- Configuring Prometheus to scrape the endpoint
- Service discovery options
- Rate limiting considerations
- Performance optimization
- Monitoring the endpoint itself

---

## Prometheus Configuration

### Basic Scrape Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'alert-history-service'
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: '/metrics'
    scheme: 'http'
    static_configs:
      - targets:
          - 'localhost:8080'
```

### With Labels

```yaml
scrape_configs:
  - job_name: 'alert-history-service'
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: '/metrics'
    scheme: 'http'
    static_configs:
      - targets:
          - 'localhost:8080'
        labels:
          service: 'alert-history'
          environment: 'production'
          team: 'platform'
```

---

## Scraping Configuration

### Scrape Interval

**Recommended:** 15-30 seconds

```yaml
scrape_interval: 15s  # Scrape every 15 seconds
```

**Considerations:**
- Shorter intervals (5-10s) increase load on the endpoint
- Longer intervals (60s+) reduce metric freshness
- Default endpoint rate limit: 60 req/min per client

### Scrape Timeout

**Recommended:** 10 seconds

```yaml
scrape_timeout: 10s  # Timeout after 10 seconds
```

**Considerations:**
- Should be less than endpoint's `GatherTimeout` (default: 5s)
- Allows time for network latency
- Prevents Prometheus from hanging

### Metrics Path

**Default:** `/metrics`

```yaml
metrics_path: '/metrics'
```

**Note:** Can be configured via `EndpointConfig.Path` in the application.

---

## Service Discovery

### Kubernetes Service Discovery

```yaml
scrape_configs:
  - job_name: 'alert-history-k8s'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      # Only scrape pods with label app=alert-history
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: alert-history
      # Use pod IP and port
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      # Add labels
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
```

### Consul Service Discovery

```yaml
scrape_configs:
  - job_name: 'alert-history-consul'
    consul_sd_configs:
      - server: 'consul:8500'
        services:
          - 'alert-history'
    relabel_configs:
      - source_labels: [__meta_consul_service]
        target_label: service
      - source_labels: [__meta_consul_node]
        target_label: node
```

### DNS Service Discovery

```yaml
scrape_configs:
  - job_name: 'alert-history-dns'
    dns_sd_configs:
      - names:
          - 'alert-history.service.consul'
        type: A
        port: 8080
```

---

## Rate Limiting Considerations

### Default Rate Limits

- **Per-client limit:** 60 requests/minute
- **Burst:** 10 requests
- **Algorithm:** Token Bucket

### Multiple Prometheus Instances

If you have multiple Prometheus instances scraping the same endpoint:

**Option 1: Increase Rate Limit**

```go
config := metrics.DefaultEndpointConfig()
config.RateLimitPerMinute = 200  // 200 req/min per client
config.RateLimitBurst = 50
```

**Option 2: Disable Rate Limiting** (Not Recommended)

```go
config := metrics.DefaultEndpointConfig()
config.RateLimitEnabled = false
```

**Option 3: Use Different IPs**

Configure Prometheus instances with different source IPs or use a load balancer.

### Rate Limit Headers

Prometheus doesn't use rate limit headers, but you can monitor them:

```promql
# Rate limit violations
rate(alert_history_metrics_endpoint_errors_total{error_type="rate_limit"}[5m])
```

---

## Performance Tuning

### Enable Caching

For high-traffic scenarios, enable caching:

```go
config := metrics.DefaultEndpointConfig()
config.CacheEnabled = true
config.CacheTTL = 5 * time.Second  # Match scrape interval
```

**Benefits:**
- ~66x faster latency
- ~71x higher throughput
- Reduced CPU usage

**Considerations:**
- Metrics may be up to TTL seconds stale
- Set TTL to match or slightly less than scrape interval

### Scrape Interval vs Cache TTL

**Recommended:** Cache TTL = Scrape Interval

```yaml
# Prometheus config
scrape_interval: 15s

# Application config
config.CacheTTL = 15 * time.Second
```

This ensures Prometheus always gets fresh metrics while benefiting from caching.

### Multiple Scrapers

If multiple Prometheus instances scrape with different intervals:

```go
config.CacheTTL = 5 * time.Second  // Shortest scrape interval
```

---

## Monitoring the Endpoint

### Self-Observability Metrics

The endpoint exposes its own metrics:

```promql
# Request rate
rate(alert_history_metrics_endpoint_requests_total[5m])

# Request duration (P95)
histogram_quantile(0.95,
  rate(alert_history_metrics_endpoint_request_duration_seconds_bucket[5m])
)

# Error rate
rate(alert_history_metrics_endpoint_errors_total[5m])

# Active requests
alert_history_metrics_endpoint_active_requests

# Response size (P95)
histogram_quantile(0.95,
  rate(alert_history_metrics_endpoint_response_size_bytes_bucket[5m])
)
```

### Grafana Dashboard

Create a dashboard with:

1. **Request Rate Panel**
   ```promql
   rate(alert_history_metrics_endpoint_requests_total[5m])
   ```

2. **Request Duration Panel** (P50, P95, P99)
   ```promql
   histogram_quantile(0.50, rate(...))
   histogram_quantile(0.95, rate(...))
   histogram_quantile(0.99, rate(...))
   ```

3. **Error Rate Panel**
   ```promql
   rate(alert_history_metrics_endpoint_errors_total[5m])
   ```

4. **Active Requests Panel**
   ```promql
   alert_history_metrics_endpoint_active_requests
   ```

5. **Cache Hit Rate** (if caching enabled)
   ```promql
   rate(alert_history_metrics_endpoint_cache_hits_total[5m]) /
   rate(alert_history_metrics_endpoint_requests_total[5m])
   ```

### Alerts

**High Error Rate:**
```yaml
groups:
  - name: metrics_endpoint
    rules:
      - alert: MetricsEndpointHighErrorRate
        expr: |
          rate(alert_history_metrics_endpoint_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate on /metrics endpoint"
          description: "Error rate is {{ $value }} errors/sec"
```

**Slow Requests:**
```yaml
      - alert: MetricsEndpointSlowRequests
        expr: |
          histogram_quantile(0.95,
            rate(alert_history_metrics_endpoint_request_duration_seconds_bucket[5m])
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow requests on /metrics endpoint"
          description: "P95 latency is {{ $value }}s"
```

**Rate Limit Violations:**
```yaml
      - alert: MetricsEndpointRateLimitViolations
        expr: |
          rate(alert_history_metrics_endpoint_errors_total{error_type="rate_limit"}[5m]) > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Rate limit violations on /metrics endpoint"
          description: "Rate limit exceeded {{ $value }} times/sec"
```

---

## Best Practices

### 1. Scrape Interval

- **Production:** 15-30 seconds
- **Development:** 30-60 seconds
- **High-frequency:** 5-10 seconds (with caching)

### 2. Timeout Configuration

- Set `scrape_timeout` > endpoint's `GatherTimeout`
- Default: `scrape_timeout: 10s` for `GatherTimeout: 5s`

### 3. Service Discovery

- Use Kubernetes service discovery for K8s deployments
- Use Consul/DNS for other environments
- Avoid static configs in production

### 4. Rate Limiting

- Keep rate limiting enabled in production
- Adjust limits based on number of scrapers
- Monitor rate limit violations

### 5. Caching

- Enable caching for high-traffic scenarios
- Set TTL to match scrape interval
- Monitor cache hit rate

### 6. Monitoring

- Monitor endpoint's own metrics
- Set up alerts for errors and slow requests
- Track cache hit rate (if enabled)

### 7. Security

- Use HTTPS in production
- Configure authentication if needed
- Monitor security headers

### 8. Performance

- Enable caching for >100 req/min
- Use buffer pooling (automatic)
- Monitor response sizes

---

## Example: Complete Prometheus Configuration

```yaml
global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'alert-history-service'
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: '/metrics'
    scheme: 'https'  # Use HTTPS in production
    tls_config:
      insecure_skip_verify: false
    static_configs:
      - targets:
          - 'alert-history.example.com:8080'
        labels:
          service: 'alert-history'
          environment: 'production'
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
```

---

## Troubleshooting

See [Troubleshooting Guide](../runbooks/metrics-endpoint-troubleshooting.md) for common issues and solutions.

---

## See Also

- [API Documentation](../api/metrics-endpoint.md)
- [Troubleshooting Guide](../runbooks/metrics-endpoint-troubleshooting.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
