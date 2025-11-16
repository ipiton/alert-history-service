# Integration Guide: Intelligent Proxy Webhook

**Target Audience**: Developers integrating with the API  
**Difficulty**: Intermediate  
**Prerequisites**: Familiarity with Alertmanager, REST APIs  
**Time**: 30-60 minutes

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Basic Integration](#basic-integration)
4. [Advanced Features](#advanced-features)
5. [Error Handling](#error-handling)
6. [Best Practices](#best-practices)
7. [Examples](#examples)
8. [Troubleshooting](#troubleshooting)

---

## Overview

The Intelligent Proxy Webhook provides advanced alert processing with:

- **LLM Classification**: Automatic categorization (category, severity, confidence)
- **Advanced Filtering**: 7 filter types (severity, time, geo, label, regex, frequency, health)
- **Multi-Target Publishing**: Parallel publishing to Rootly, PagerDuty, Slack
- **High Performance**: p95 < 50ms, 66K+ req/s
- **Enterprise Security**: OWASP compliant, API Key/JWT auth

### Architecture

```
Alertmanager → POST /webhook/proxy → [Classification → Filtering → Publishing] → Targets
```

---

## Prerequisites

### 1. Access & Authentication

**Get API Key**:

```bash
# From your administrator
API_KEY="ah_1234567890abcdef"

# Or from Kubernetes secret
kubectl get secret alert-history-api-keys -n alert-history \
  -o jsonpath='{.data.api-key}' | base64 -d
```

### 2. Network Access

Ensure your Alertmanager can reach the endpoint:

```bash
# Test connectivity
curl -I https://api.alerthistory.io/v1/health

# Expected: HTTP/1.1 200 OK
```

### 3. Alertmanager Version

- **Required**: Alertmanager v0.20+
- **Recommended**: Alertmanager v0.25+

---

## Basic Integration

### Step 1: Configure Alertmanager

Add the webhook receiver to `alertmanager.yml`:

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m

route:
  receiver: 'intelligent-proxy'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'intelligent-proxy'
    webhook_configs:
      - url: 'https://api.alerthistory.io/v1/webhook/proxy'
        send_resolved: true
        http_config:
          bearer_token: 'ah_1234567890abcdef'
        # Optional: custom headers
        headers:
          X-Environment: production
          X-Team: platform
```

### Step 2: Reload Alertmanager

```bash
# Docker
docker kill -s HUP alertmanager

# Kubernetes
kubectl exec -n monitoring alertmanager-0 -- kill -HUP 1

# Systemd
systemctl reload alertmanager

# Verify reload
curl http://localhost:9093/-/reload -X POST
```

### Step 3: Test Integration

Send a test alert:

```bash
# Using amtool
amtool alert add test_integration \
  alertname=TestIntegration \
  severity=warning \
  summary="Testing intelligent proxy integration" \
  --alertmanager.url=http://localhost:9093

# Verify alert sent
amtool alert query alertname=TestIntegration
```

### Step 4: Verify Processing

Check the response in Alert History logs:

```bash
# Kubernetes
kubectl logs -n alert-history -l app=alert-history --tail=50 | grep TestIntegration

# Expected output:
# INFO Proxy webhook request processed status=success alerts=1 classified=1 published=1
```

---

## Advanced Features

### 1. LLM Classification

Enable automatic alert classification:

**Configuration**:

```yaml
# config.yaml
proxy:
  classification:
    enabled: true
    timeout: 5s
    cache_ttl: 15m
    fallback_enabled: true
```

**Response includes**:

```json
{
  "alert_results": [{
    "classification": {
      "category": "performance",
      "severity": "high",
      "confidence": 0.95,
      "cached": true
    }
  }]
}
```

**Benefits**:
- Automatic categorization
- Severity normalization
- Confidence scoring
- 95%+ cache hit rate (fast!)

### 2. Advanced Filtering

Configure custom filter rules:

```yaml
# filter-rules.yaml
filters:
  # Severity-based filtering
  - name: "critical_only_production"
    type: "severity"
    action: "allow"
    config:
      min_severity: "critical"
    conditions:
      labels:
        environment: "production"

  # Time-based filtering (business hours only)
  - name: "business_hours_only"
    type: "time"
    action: "allow"
    config:
      timezone: "America/New_York"
      days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
      start_time: "09:00"
      end_time: "17:00"

  # Label-based filtering
  - name: "ignore_test_alerts"
    type: "label"
    action: "deny"
    config:
      label: "alertname"
      regex: "^(Test|Fake).*"

  # Geo-based filtering
  - name: "us_only"
    type: "geo"
    action: "allow"
    config:
      allowed_regions: ["us-east-1", "us-west-2"]
      label: "region"

  # Frequency-based filtering (throttling)
  - name: "throttle_chatty_alerts"
    type: "frequency"
    action: "deny"
    config:
      max_per_minute: 10
      window: "1m"

  # Health-based filtering
  - name: "ignore_flapping"
    type: "health"
    action: "deny"
    config:
      flap_threshold: 5
      flap_window: "5m"

  # Regex-based filtering
  - name: "high_value_alerts"
    type: "regex"
    action: "allow"
    config:
      field: "annotations.summary"
      pattern: "(critical|urgent|emergency)"
```

**Apply filters**:

```bash
# Create ConfigMap with rules
kubectl create configmap filter-rules \
  --from-file=filter-rules.yaml \
  -n alert-history

# Mount in deployment
# volumes:
# - name: filter-rules
#   configMap:
#     name: filter-rules
```

### 3. Multi-Target Publishing

Configure publishing targets:

```yaml
# publishing-config.yaml
publishing:
  targets:
    # Rootly (Incident Management)
    - name: "production-rootly"
      type: "rootly"
      enabled: true
      endpoint: "https://api.rootly.com/v1/events"
      api_key: "${ROOTLY_API_KEY}"
      conditions:
        labels:
          environment: "production"
          severity: "critical"

    # PagerDuty (On-Call)
    - name: "oncall-pagerduty"
      type: "pagerduty"
      enabled: true
      routing_key: "${PAGERDUTY_ROUTING_KEY}"
      conditions:
        labels:
          severity: "critical"

    # Slack (Team Notifications)
    - name: "platform-slack"
      type: "slack"
      enabled: true
      webhook_url: "${SLACK_WEBHOOK_URL}"
      channel: "#platform-alerts"
      conditions:
        labels:
          team: "platform"

    # Generic Webhook
    - name: "custom-webhook"
      type: "webhook"
      enabled: true
      endpoint: "https://custom.company.com/webhooks/alerts"
      method: "POST"
      headers:
        Authorization: "Bearer ${CUSTOM_API_TOKEN}"
```

**Response includes publishing results**:

```json
{
  "publishing_summary": {
    "total_targets": 3,
    "successful_targets": 3,
    "failed_targets": 0,
    "total_publish_time_ms": 250
  },
  "alert_results": [{
    "publishing": {
      "published_to": ["rootly", "pagerduty", "slack"],
      "failed_to": [],
      "details": [
        {
          "target_name": "production-rootly",
          "target_type": "rootly",
          "success": true,
          "status_code": 201,
          "processing_time_ms": 120
        }
      ]
    }
  }]
}
```

### 4. Batch Processing

Send multiple alerts in one request:

```bash
curl -X POST https://api.alerthistory.io/v1/webhook/proxy \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "batch-test",
    "status": "firing",
    "alerts": [
      {
        "status": "firing",
        "labels": {"alertname": "Alert1", "severity": "warning"},
        "annotations": {"summary": "First alert"},
        "startsAt": "2025-11-16T10:00:00Z"
      },
      {
        "status": "firing",
        "labels": {"alertname": "Alert2", "severity": "critical"},
        "annotations": {"summary": "Second alert"},
        "startsAt": "2025-11-16T10:01:00Z"
      },
      {
        "status": "firing",
        "labels": {"alertname": "Alert3", "severity": "info"},
        "annotations": {"summary": "Third alert"},
        "startsAt": "2025-11-16T10:02:00Z"
      }
    ]
  }'
```

**Benefits**:
- 3x reduction in HTTP overhead
- Lower latency (single round-trip)
- Atomic processing
- Max 100 alerts per request

### 5. Custom Headers & Metadata

Add custom context to requests:

```yaml
# alertmanager.yml
receivers:
  - name: 'intelligent-proxy'
    webhook_configs:
      - url: 'https://api.alerthistory.io/v1/webhook/proxy'
        http_config:
          bearer_token: 'ah_1234567890abcdef'
        headers:
          X-Environment: production
          X-Team: platform
          X-Service: alertmanager-prod-01
          X-Region: us-east-1
```

These headers are logged for correlation.

---

## Error Handling

### Understanding HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Alert processed successfully |
| 400 | Bad Request | Fix payload structure |
| 401 | Unauthorized | Check API key |
| 429 | Rate Limited | Implement backoff/retry |
| 500 | Internal Error | Check status page, retry |
| 503 | Unavailable | Service maintenance, retry |

### Retry Strategy

**Recommended retry logic**:

```python
import requests
from tenacity import retry, stop_after_attempt, wait_exponential

@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=1, max=10),
    retry=lambda e: isinstance(e, requests.exceptions.RequestException)
)
def send_alert(payload):
    response = requests.post(
        'https://api.alerthistory.io/v1/webhook/proxy',
        json=payload,
        headers={'X-API-Key': 'ah_your_key'},
        timeout=30
    )
    response.raise_for_status()
    return response.json()
```

**Alertmanager built-in retry**:

Alertmanager automatically retries failed webhooks:
- 3 attempts by default
- Exponential backoff
- Configure with `max_alerts` and `queue_capacity`

### Circuit Breaker

If error rate is high, implement circuit breaker:

```python
from pybreaker import CircuitBreaker

breaker = CircuitBreaker(
    fail_max=5,           # Open after 5 failures
    timeout_duration=60   # Try again after 60s
)

@breaker
def send_alert_with_breaker(payload):
    return send_alert(payload)
```

---

## Best Practices

### 1. Payload Optimization

**Do**:
- ✅ Send resolved alerts (`send_resolved: true`)
- ✅ Use meaningful alert names
- ✅ Include `annotations.summary`
- ✅ Group alerts (`group_by`)
- ✅ Set appropriate `group_wait` and `group_interval`

**Don't**:
- ❌ Send > 100 alerts per request
- ❌ Include sensitive data in labels
- ❌ Use excessive label cardinality
- ❌ Send test alerts to production

### 2. Rate Limiting

**Default limits**:
- Per-IP: 100 req/s
- Global: 1,000 req/s
- Burst: 50 requests

**Optimize**:
```yaml
# Reduce frequency in Alertmanager
route:
  group_wait: 30s        # Batch alerts for 30s
  group_interval: 5m     # Don't send more often than 5m
  repeat_interval: 4h    # Don't repeat more than every 4h
```

### 3. Authentication

**Use secrets**:

```yaml
# Don't hardcode API keys!
webhook_configs:
  - url: 'https://api.alerthistory.io/v1/webhook/proxy'
    http_config:
      bearer_token_file: /etc/alertmanager/api-key.txt  # ✅
      # bearer_token: 'hardcoded_key'  # ❌
```

### 4. Monitoring

**Track webhook health**:

```promql
# Success rate
rate(alertmanager_notifications_total{integration="webhook"}[5m]) 
/ 
rate(alertmanager_notifications_failed_total{integration="webhook"}[5m])

# Latency
histogram_quantile(0.95, 
  rate(alertmanager_notification_duration_seconds_bucket{integration="webhook"}[5m])
)
```

### 5. Classification Optimization

**Maximize cache hits**:
- Use consistent alert names
- Include meaningful summaries
- Avoid random/timestamp in labels
- Expected cache hit rate: 95%+

**Monitor cache**:

```promql
# Cache hit rate
rate(alert_history_proxy_classification_duration_seconds_count{cached="true"}[5m]) 
/ 
rate(alert_history_proxy_classification_duration_seconds_count[5m])
```

---

## Examples

### Example 1: Simple Prometheus Alert

**prometheus-rules.yaml**:

```yaml
groups:
  - name: example
    rules:
      - alert: HighCPUUsage
        expr: node_cpu_usage > 90
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage is {{ $value }}%"
```

**Expected webhook payload**:

```json
{
  "receiver": "intelligent-proxy",
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {
      "alertname": "HighCPUUsage",
      "severity": "warning",
      "team": "platform",
      "instance": "server-01"
    },
    "annotations": {
      "summary": "High CPU usage on server-01",
      "description": "CPU usage is 95%"
    },
    "startsAt": "2025-11-16T10:00:00Z",
    "generatorURL": "https://prometheus/graph?..."
  }]
}
```

### Example 2: Multi-Severity Routing

Route critical alerts to PagerDuty, others to Slack:

**alertmanager.yml**:

```yaml
route:
  receiver: 'default-slack'
  routes:
    # Critical -> PagerDuty + Slack
    - match:
        severity: critical
      receiver: 'intelligent-proxy-critical'
      continue: true
      
    # Warning -> Slack only
    - match:
        severity: warning
      receiver: 'intelligent-proxy-warning'

receivers:
  - name: 'intelligent-proxy-critical'
    webhook_configs:
      - url: 'https://api.alerthistory.io/v1/webhook/proxy'
        http_config:
          bearer_token: 'ah_critical_key'
        headers:
          X-Priority: critical
          
  - name: 'intelligent-proxy-warning'
    webhook_configs:
      - url: 'https://api.alerthistory.io/v1/webhook/proxy'
        http_config:
          bearer_token: 'ah_warning_key'
        headers:
          X-Priority: warning
```

### Example 3: Custom Integration (Python)

```python
#!/usr/bin/env python3
import requests
from datetime import datetime

class AlertHistoryClient:
    def __init__(self, api_key, base_url="https://api.alerthistory.io/v1"):
        self.api_key = api_key
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'X-API-Key': api_key,
            'Content-Type': 'application/json'
        })
    
    def send_alert(self, alert_name, severity, summary, labels=None, annotations=None):
        """Send a single alert"""
        payload = {
            "receiver": "custom-integration",
            "status": "firing",
            "alerts": [{
                "status": "firing",
                "labels": {
                    "alertname": alert_name,
                    "severity": severity,
                    **(labels or {})
                },
                "annotations": {
                    "summary": summary,
                    **(annotations or {})
                },
                "startsAt": datetime.utcnow().isoformat() + "Z"
            }]
        }
        
        response = self.session.post(
            f"{self.base_url}/webhook/proxy",
            json=payload,
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    
    def send_batch(self, alerts):
        """Send multiple alerts in batch"""
        payload = {
            "receiver": "batch-integration",
            "status": "firing",
            "alerts": [
                {
                    "status": "firing",
                    "labels": alert["labels"],
                    "annotations": alert.get("annotations", {}),
                    "startsAt": datetime.utcnow().isoformat() + "Z"
                }
                for alert in alerts
            ]
        }
        
        response = self.session.post(
            f"{self.base_url}/webhook/proxy",
            json=payload,
            timeout=30
        )
        response.raise_for_status()
        return response.json()

# Usage
client = AlertHistoryClient(api_key="ah_1234567890abcdef")

# Send single alert
result = client.send_alert(
    alert_name="CustomAlert",
    severity="warning",
    summary="Something happened",
    labels={"service": "api", "environment": "production"}
)
print(f"Alert processed: {result['status']}")

# Send batch
result = client.send_batch([
    {"labels": {"alertname": "Alert1", "severity": "info"}},
    {"labels": {"alertname": "Alert2", "severity": "warning"}},
    {"labels": {"alertname": "Alert3", "severity": "critical"}}
])
print(f"Batch processed: {result['alerts_summary']['total_processed']} alerts")
```

---

## Troubleshooting

### Issue 1: 401 Unauthorized

**Symptoms**: All requests return 401

**Diagnosis**:
```bash
# Test API key
curl -H "X-API-Key: $API_KEY" https://api.alerthistory.io/v1/health

# Check Alertmanager config
kubectl get configmap alertmanager-config -o yaml | grep bearer_token
```

**Solution**: Update API key in Alertmanager config

### Issue 2: 400 Validation Error

**Symptoms**: Requests rejected with validation errors

**Diagnosis**:
```bash
# Check logs for details
kubectl logs -n alert-history -l app=alert-history | grep "validation failed"

# Common errors:
# - Missing required field: labels, startsAt
# - Invalid timestamp format
# - Empty alerts array
```

**Solution**: Fix payload structure (see OpenAPI spec)

### Issue 3: Slow Classification

**Symptoms**: High latency, timeouts

**Diagnosis**:
```promql
# Check classification latency
histogram_quantile(0.95, 
  rate(alert_history_proxy_classification_duration_seconds_bucket[5m])
)

# Check cache hit rate
rate(alert_history_proxy_classification_duration_seconds_count{cached="true"}[5m]) 
/ 
rate(alert_history_proxy_classification_duration_seconds_count[5m])
```

**Solution**:
- Check LLM service health
- Verify Redis cache is working
- Increase cache TTL if needed

### Issue 4: Alerts Filtered Out

**Symptoms**: Alerts not published, filtered count high

**Diagnosis**:
```bash
# Check filter rules
kubectl get configmap filter-rules -o yaml

# Check filtered alerts in logs
kubectl logs -n alert-history -l app=alert-history | grep "filtered"
```

**Solution**: Adjust filter rules or disable filtering temporarily

---

## Next Steps

1. ✅ **Review Integration** - Ensure alerts are flowing
2. ✅ **Configure Filters** - Set up custom filtering rules
3. ✅ **Add Publishing Targets** - Configure Rootly, PagerDuty, Slack
4. ✅ **Monitor Performance** - Set up Grafana dashboards
5. ✅ **Test Edge Cases** - High volume, failures, timeouts

---

## Support & Resources

- **API Reference**: [OpenAPI Spec](../api/openapi.yaml)
- **Quick Start**: [5-Minute Setup](./quickstart.md)
- **Migration Guide**: [From TN-061](./migration-guide.md)
- **Runbooks**: [Operational Procedures](../runbooks/)
- **Support**: team@alerthistory.io
- **Status**: https://status.alerthistory.io

---

**Last Updated**: 2025-11-16  
**Version**: 1.0.0

