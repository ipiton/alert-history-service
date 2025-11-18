# Prometheus Parser Integration Guide (TN-146)

**Target Audience**: DevOps Engineers, SREs, Platform Engineers
**Prerequisites**: Alertmanager++ OSS Core deployed, Prometheus 2.0+ running
**Estimated Time**: 30-45 minutes
**Status**: Production-Ready

---

## 📋 Table of Contents

1. [Prometheus Configuration](#1-prometheus-configuration)
2. [Testing with Real Prometheus](#2-testing-with-real-prometheus)
3. [Endpoint Registration (TN-147 Preview)](#3-endpoint-registration-tn-147-preview)
4. [Monitoring & Observability](#4-monitoring--observability)
5. [Troubleshooting](#5-troubleshooting)

---

## 1. Prometheus Configuration

### 1.1 Configure Prometheus Remote Write

Prometheus doesn't natively send webhooks, so you'll need to use **Alertmanager** or a **custom exporter**. However, for direct integration, you can poll Prometheus `/api/v1/alerts` endpoint.

#### Option A: Poll Prometheus Alerts API (Recommended)

Create a **cron job** or **systemd timer** to periodically fetch alerts:

```bash
#!/bin/bash
# fetch-prometheus-alerts.sh

PROMETHEUS_URL="http://prometheus:9090"
WEBHOOK_URL="http://alerthistory:8080/webhook"

# Fetch alerts from Prometheus
alerts=$(curl -s "$PROMETHEUS_URL/api/v1/alerts" | jq -r '.data.alerts')

# Send to Alert History webhook
curl -X POST \
  -H "Content-Type: application/json" \
  -d "$alerts" \
  "$WEBHOOK_URL"
```

**Cron Schedule** (every 2 minutes):
```cron
*/2 * * * * /path/to/fetch-prometheus-alerts.sh
```

#### Option B: Use Alertmanager as Proxy

Configure Alertmanager to forward alerts to Alert History:

**alertmanager.yml**:
```yaml
route:
  receiver: 'alerthistory'
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'alerthistory'
    webhook_configs:
      - url: 'http://alerthistory:8080/webhook'
        send_resolved: true
        http_config:
          basic_auth:
            username: 'webhook'
            password: 'secret'
```

**Benefits**:
- Automatic webhook delivery
- Built-in grouping and deduplication
- Retry logic
- TLS support

#### Option C: Prometheus Federation + Webhook Bridge

Use `prometheus-webhook-dingtalk` or similar tool:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'federate'
    scrape_interval: 15s
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{__name__=~"ALERTS.*"}'
    static_configs:
      - targets:
        - 'prometheus:9090'
```

---

### 1.2 Prometheus Alert Rules

Define alert rules in Prometheus:

**alerts.yml**:
```yaml
groups:
  - name: example
    interval: 10s
    rules:
      - alert: HighCPU
        expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 2m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage is {{ $value }}%"

      - alert: HighMemory
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 85
        for: 5m
        labels:
          severity: critical
          team: platform
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
```

**Load into Prometheus**:
```bash
# prometheus.yml
rule_files:
  - "alerts.yml"

# Reload
curl -X POST http://prometheus:9090/-/reload
```

---

## 2. Testing with Real Prometheus

### 2.1 Setup Test Environment

```bash
# Start Prometheus with test configuration
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $PWD/prometheus.yml:/etc/prometheus/prometheus.yml \
  -v $PWD/alerts.yml:/etc/prometheus/alerts.yml \
  prom/prometheus:latest
```

### 2.2 Verify Alerts Endpoint

```bash
# Check Prometheus alerts API
curl http://localhost:9090/api/v1/alerts | jq

# Expected output (v1 format):
# {
#   "status": "success",
#   "data": {
#     "alerts": [
#       {
#         "labels": { "alertname": "HighCPU", ... },
#         "state": "firing",
#         "activeAt": "2025-11-18T10:00:00Z",
#         "generatorURL": "http://prometheus:9090/graph"
#       }
#     ]
#   }
# }
```

### 2.3 Send Test Alert to Alert History

```bash
# Simulate Prometheus v1 webhook
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '[
    {
      "labels": {
        "alertname": "TestAlert",
        "instance": "test-server:9100",
        "severity": "warning"
      },
      "annotations": {
        "summary": "This is a test alert"
      },
      "state": "firing",
      "activeAt": "2025-11-18T10:00:00Z",
      "generatorURL": "http://prometheus:9090/graph"
    }
  ]'

# Expected response:
# {
#   "status": "success",
#   "webhook_type": "prometheus",
#   "alerts_received": 1,
#   "alerts_processed": 1
# }
```

### 2.4 Test Prometheus v2 Grouped Format

```bash
# Send grouped alert (v2 format)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "groups": [
      {
        "labels": {
          "job": "api",
          "severity": "critical"
        },
        "alerts": [
          {
            "labels": {
              "alertname": "HighLatency",
              "instance": "api-1"
            },
            "annotations": {
              "summary": "API latency is high"
            },
            "state": "firing",
            "activeAt": "2025-11-18T10:05:00Z",
            "generatorURL": "http://prometheus:9090"
          }
        ]
      }
    ]
  }'
```

### 2.5 Verify Alert in Database

```bash
# Query Alert History API
curl http://localhost:8080/api/v2/alerts | jq

# Search by fingerprint
curl http://localhost:8080/api/v2/alerts?fingerprint=<hash> | jq

# Verify in PostgreSQL
psql -h localhost -U alerthistory -d alerthistory_db -c \
  "SELECT alert_name, status, labels, created_at FROM alerts ORDER BY created_at DESC LIMIT 5;"
```

---

## 3. Endpoint Registration (TN-147 Preview)

**Note**: This section previews **TN-147** (next task), which will add HTTP endpoint registration.

### 3.1 Endpoint Configuration

Once TN-147 is complete, the following endpoint will be available:

```
POST /webhook
  - Auto-detects webhook type (Alertmanager, Prometheus v1/v2, Generic)
  - Parses payload using appropriate parser
  - Validates and converts to domain model
  - Stores in PostgreSQL
  - Returns processing result
```

### 3.2 Nginx Reverse Proxy

**nginx.conf**:
```nginx
upstream alerthistory {
    server alerthistory-1:8080;
    server alerthistory-2:8080;
    server alerthistory-3:8080;
}

server {
    listen 443 ssl http2;
    server_name alerts.example.com;

    ssl_certificate /etc/ssl/certs/alerts.crt;
    ssl_certificate_key /etc/ssl/private/alerts.key;

    location /webhook {
        proxy_pass http://alerthistory;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 5s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;

        # Rate limiting
        limit_req zone=webhook_limit burst=10 nodelay;
    }
}

# Rate limit zone
limit_req_zone $binary_remote_addr zone=webhook_limit:10m rate=100r/s;
```

### 3.3 Kubernetes Ingress

**ingress.yaml**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: alerthistory-ingress
  namespace: alerthistory
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/rate-limit: "100"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - alerts.example.com
      secretName: alerthistory-tls
  rules:
    - host: alerts.example.com
      http:
        paths:
          - path: /webhook
            pathType: Prefix
            backend:
              service:
                name: alerthistory
                port:
                  number: 8080
```

---

## 4. Monitoring & Observability

### 4.1 Prometheus Metrics

Alert History exposes metrics at `/metrics`:

```bash
# Scrape metrics
curl http://localhost:8080/metrics | grep webhook

# Key metrics:
# - webhook_requests_total{type="prometheus",status="success"}
# - webhook_processing_duration_seconds{type="prometheus",stage="parse"}
# - webhook_errors_total{type="prometheus",error_type="validation"}
# - webhook_payload_size_bytes{type="prometheus"}
```

**Prometheus scrape config**:
```yaml
scrape_configs:
  - job_name: 'alerthistory'
    static_configs:
      - targets: ['alerthistory:8080']
    metrics_path: '/metrics'
```

### 4.2 Grafana Dashboard

**Sample PromQL queries**:

```promql
# Total webhook requests by type
sum(rate(webhook_requests_total[5m])) by (type)

# P95 parse latency
histogram_quantile(0.95, sum(rate(webhook_processing_duration_seconds_bucket{stage="parse"}[5m])) by (le, type))

# Error rate
sum(rate(webhook_errors_total[5m])) by (type, error_type)

# Throughput (alerts/sec)
sum(rate(webhook_alerts_processed_total[5m]))
```

**Dashboard Panels**:
- Webhook requests/sec by type
- Parse latency (p50, p95, p99)
- Error rate by type
- Payload size distribution
- Validation errors by field

---

## 5. Troubleshooting

### 5.1 Parser Not Detected

**Symptom**: Prometheus webhook detected as Alertmanager or Generic

**Check**:
```bash
# Enable debug logging
export LOG_LEVEL=DEBUG

# Check detector output
tail -f /var/log/alerthistory/app.log | grep "Webhook detected"
```

**Solution**: Verify payload structure matches Prometheus format

### 5.2 Validation Errors

**Symptom**: `validation failed: alertname is required`

**Check**:
```bash
# Validate payload manually
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '@test_payload.json' -v
```

**Solution**: Ensure all required fields present

### 5.3 Performance Issues

**Symptom**: High parse latency (>100ms)

**Check**:
```bash
# Run benchmarks
go test -bench=BenchmarkParse -benchmem ./internal/infrastructure/webhook/

# Check metrics
curl http://localhost:8080/metrics | grep webhook_processing_duration
```

**Solution**: Increase worker pool size or scale horizontally

---

## 📊 Quick Reference

| Task | Command |
|------|---------|
| Test webhook | `curl -X POST http://localhost:8080/webhook -d @payload.json` |
| Check metrics | `curl http://localhost:8080/metrics` |
| View logs | `kubectl logs -f deployment/alerthistory` |
| Run benchmarks | `go test -bench=. ./internal/infrastructure/webhook/` |
| Validate payload | Use online JSON validator + schema |

---

**Next Steps**: TN-147 (Webhook Endpoint Registration), TN-148 (E2E Integration Tests)

**Support**: Open issue on GitHub or contact platform team
