# TN-052: Rootly Publisher - Integration Guide

**Date**: 2025-11-10
**Phase**: Phase 6 - Integration with Publishing System
**Status**: ✅ **COMPLETE**

---

## 📋 Overview

This guide explains how to integrate the **EnhancedRootlyPublisher** into the Alert History publishing system for full incident lifecycle management with Rootly.

**Key Features:**
- ✅ Automatic incident creation from firing alerts
- ✅ Automatic incident updates from alert changes
- ✅ Automatic incident resolution from resolved alerts
- ✅ Rate limiting (60 req/min)
- ✅ Retry logic with exponential backoff
- ✅ Incident ID caching (24h TTL)
- ✅ 8 Prometheus metrics
- ✅ TLS 1.2+ security

---

## 🚀 Quick Start

### 1. Create Rootly API Secret

Create a Kubernetes Secret with your Rootly API credentials:

```bash
# Create secret from literal values
kubectl create secret generic rootly-prod \
  --namespace=alert-history \
  --from-literal=name=rootly-prod \
  --from-literal=type=rootly \
  --from-literal=url=https://api.rootly.com/v1/incidents \
  --from-literal=api_key=YOUR_ROOTLY_API_KEY \
  --from-literal=enabled=true \
  --from-literal=format=rootly

# Label the secret for auto-discovery
kubectl label secret rootly-prod \
  --namespace=alert-history \
  publishing-target=true \
  target-type=rootly
```

**OR** use the example YAML:

```bash
# Edit the example with your API key
kubectl apply -f examples/k8s/rootly-secret-example.yaml
```

### 2. Verify Secret Discovery

The TargetDiscoveryManager will automatically discover the secret:

```bash
# Check discovery logs
kubectl logs -n alert-history deployment/alert-history | grep "Discovered target"

# Expected output:
# INFO Discovered target name=rootly-prod type=rootly url=https://api.rootly.com/v1/incidents
```

### 3. Test Publishing

Send a test alert to verify Rootly integration:

```bash
curl -X POST http://alert-history:8080/api/v1/alerts/publish \
  -H "Content-Type: application/json" \
  -d '{
    "alert": {
      "alertName": "TestAlert",
      "status": "firing",
      "labels": {"severity": "critical"},
      "fingerprint": "test-123"
    },
    "targets": ["rootly-prod"]
  }'
```

### 4. Monitor Metrics

Check Prometheus metrics to verify operation:

```bash
# Incidents created
curl http://alert-history:9090/metrics | grep rootly_incidents_created_total

# API request duration
curl http://alert-history:9090/metrics | grep rootly_api_duration_seconds

# Cache hits/misses
curl http://alert-history:9090/metrics | grep rootly_.*_cache
```

---

## 🔧 Configuration

### Required Secret Fields

| Field | Description | Example |
|-------|-------------|---------|
| **name** | Target identifier (must be unique) | `rootly-prod` |
| **type** | Must be `"rootly"` | `rootly` |
| **url** | Rootly Incidents API URL | `https://api.rootly.com/v1/incidents` |
| **api_key** | Rootly API key (from Rootly dashboard) | `rtly_abc123xyz...` |

### Optional Secret Fields

| Field | Description | Default |
|-------|-------------|---------|
| **enabled** | Enable/disable target | `true` |
| **format** | Alert format to use | `rootly` |

### API Key Generation

To generate a Rootly API key:

1. Log in to [Rootly](https://rootly.com)
2. Navigate to **Settings** → **API Tokens**
3. Click **Create Token**
4. Set permissions: `incidents:write`
5. Copy the token (starts with `rtly_`)

---

## 🏗 Architecture

### Component Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                    Publishing Pipeline                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Firing Alert  ────┬─→ [PublishingQueue] ─→ [PublisherFactory]  │
│                    │                                               │
│  Resolved Alert ───┘                         ↓                    │
│                                                                    │
│                              ┌────────────────────────────────┐  │
│                              │ CreatePublisherForTarget()    │  │
│                              │  - Detects target.Type       │  │
│                              │  - Creates EnhancedRootly    │  │
│                              └────────────────────────────────┘  │
│                                         ↓                          │
│                              ┌────────────────────────────────┐  │
│                              │  EnhancedRootlyPublisher      │  │
│                              │  ┌───────────────────────┐   │  │
│                              │  │ RootlyIncidentsClient │   │  │
│                              │  │  - Rate limiter       │   │  │
│                              │  │  - Retry logic        │   │  │
│                              │  │  - TLS 1.2+           │   │  │
│                              │  └───────────────────────┘   │  │
│                              │  ┌───────────────────────┐   │  │
│                              │  │ IncidentIDCache       │   │  │
│                              │  │  - 24h TTL            │   │  │
│                              │  │  - Thread-safe        │   │  │
│                              │  └───────────────────────┘   │  │
│                              │  ┌───────────────────────┐   │  │
│                              │  │ RootlyMetrics         │   │  │
│                              │  │  - 8 Prometheus metrics│   │  │
│                              │  └───────────────────────┘   │  │
│                              └────────────────────────────────┘  │
│                                         ↓                          │
│                              ┌────────────────────────────────┐  │
│                              │    Rootly Incidents API        │  │
│                              │  POST   /incidents             │  │
│                              │  PATCH  /incidents/{id}        │  │
│                              │  POST   /incidents/{id}/resolve│  │
│                              └────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Integration Points

#### 1. **PublisherFactory** (`publisher.go`)

The factory now supports two modes:

**Legacy Mode** (CreatePublisher):
```go
// Creates simple HTTP-based publisher (no incident lifecycle)
publisher := factory.CreatePublisher("rootly")
```

**Enhanced Mode** (CreatePublisherForTarget):
```go
// Creates full EnhancedRootlyPublisher with API integration
publisher := factory.CreatePublisherForTarget(target)
```

#### 2. **Shared Resources**

The factory maintains shared resources for all Rootly publishers:

```go
type PublisherFactory struct {
    formatter       AlertFormatter
    logger          *slog.Logger
    rootlyCache     IncidentIDCache             // Shared incident ID cache
    rootlyMetrics   *RootlyMetrics              // Shared Prometheus metrics
    rootlyClientMap map[string]RootlyIncidentsClient // Client pool by API key
}
```

**Benefits:**
- ✅ Single incident cache across all targets
- ✅ Unified metrics for all Rootly publishers
- ✅ Client reuse (one client per unique API key)

#### 3. **Incident Lifecycle**

```
1. Firing Alert (first time)
   └─→ Check cache for incident ID
   └─→ Not found → Create incident
   └─→ Store incident ID in cache (fingerprint → ID)
   └─→ Record metric: rootly_incidents_created_total

2. Firing Alert (subsequent)
   └─→ Check cache for incident ID
   └─→ Found → Update incident (PATCH)
   └─→ Record metric: rootly_incidents_updated_total

3. Resolved Alert
   └─→ Check cache for incident ID
   └─→ Found → Resolve incident (POST /resolve)
   └─→ Delete incident ID from cache
   └─→ Record metric: rootly_incidents_resolved_total
```

---

## 📊 Prometheus Metrics

### Available Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| **rootly_incidents_created_total** | Counter | severity | Total incidents created |
| **rootly_incidents_updated_total** | Counter | reason | Total incidents updated |
| **rootly_incidents_resolved_total** | Counter | - | Total incidents resolved |
| **rootly_api_requests_total** | Counter | endpoint, method, status | Total API requests |
| **rootly_api_duration_seconds** | Histogram | endpoint, method | API request duration |
| **rootly_api_errors_total** | Counter | endpoint, error_type | Total API errors |
| **rootly_rate_limit_hits_total** | Counter | - | Total rate limit hits |
| **rootly_active_incidents_gauge** | Gauge | - | Active incidents in cache |

### Example Queries

**Incident Creation Rate:**
```promql
rate(rootly_incidents_created_total[5m])
```

**API Error Rate:**
```promql
rate(rootly_api_errors_total[5m])
```

**Average API Response Time:**
```promql
histogram_quantile(0.95,
  rate(rootly_api_duration_seconds_bucket[5m]))
```

**Cache Hit Rate:**
```promql
rate(rootly_incident_cache_hits_total[5m]) /
  (rate(rootly_incident_cache_hits_total[5m]) +
   rate(rootly_incident_cache_misses_total[5m]))
```

---

## 🔍 Troubleshooting

### Common Issues

#### Issue 1: "Rootly target missing API key"

**Symptom:**
```
WARN Rootly target missing API key, falling back to HTTP publisher
```

**Solution:**
Ensure the secret has the `api_key` field:

```bash
kubectl get secret rootly-prod -o jsonpath='{.data.api_key}' | base64 -d
```

If missing, update the secret:

```bash
kubectl patch secret rootly-prod \
  --namespace=alert-history \
  --patch '{"data":{"api_key":"'$(echo -n "YOUR_KEY" | base64)'"}}'
```

---

#### Issue 2: Rate Limit Errors

**Symptom:**
```
ERROR Rootly API error 429: Rate limit exceeded - Try again in 30 seconds
```

**Solution:**
The client automatically handles rate limiting with:
- Token bucket algorithm (60 req/min)
- Exponential backoff on 429 errors

**Monitor rate limit hits:**
```bash
curl http://alert-history:9090/metrics | grep rootly_rate_limit_hits_total
```

**If persistent, consider:**
- Reducing alert volume
- Upgrading Rootly plan
- Batching updates

---

#### Issue 3: Incident Not Resolving

**Symptom:**
Alert is resolved but Rootly incident stays open.

**Possible Causes:**
1. Incident ID expired from cache (>24h TTL)
2. Incident manually closed in Rootly
3. Fingerprint mismatch

**Debug Steps:**

```bash
# Check cache size
curl http://alert-history:9090/metrics | grep rootly_active_incidents_gauge

# Check cache hit rate
curl http://alert-history:9090/metrics | grep rootly_incident_cache

# Check logs for cache misses
kubectl logs -n alert-history deployment/alert-history | grep "Incident ID not found in cache"
```

---

#### Issue 4: Authentication Errors

**Symptom:**
```
ERROR Rootly API error 401: Unauthorized - Invalid API key
```

**Solution:**

1. Verify API key is valid:
   ```bash
   curl -H "Authorization: Bearer YOUR_KEY" \
        https://api.rootly.com/v1/incidents
   ```

2. Check key permissions (must have `incidents:write`)

3. Regenerate key if expired

---

## 🧪 Testing

### Manual Testing

1. **Create Test Incident:**
```bash
curl -X POST http://alert-history:8080/api/v1/alerts/publish \
  -H "Content-Type: application/json" \
  -d '{
    "alert": {
      "alertName": "TestIncident",
      "status": "firing",
      "labels": {"severity": "critical", "namespace": "production"},
      "fingerprint": "test-' + $(date +%s) + '",
      "startsAt": "' + $(date -u +"%Y-%m-%dT%H:%M:%SZ") + '"
    },
    "targets": ["rootly-prod"]
  }'
```

2. **Verify in Rootly:**
   - Open [Rootly Incidents](https://app.rootly.com/incidents)
   - Look for incident with title: `[TestIncident] Alert in production`

3. **Resolve Test Incident:**
```bash
curl -X POST http://alert-history:8080/api/v1/alerts/publish \
  -H "Content-Type: application/json" \
  -d '{
    "alert": {
      "alertName": "TestIncident",
      "status": "resolved",
      "fingerprint": "test-SAME-AS-ABOVE",
      "endsAt": "' + $(date -u +"%Y-%m-%dT%H:%M:%SZ") + '"
    },
    "targets": ["rootly-prod"]
  }'
```

4. **Verify Resolution:**
   - Check Rootly incident is marked as "Resolved"

---

## 🚦 Best Practices

### 1. API Key Management

✅ **DO:**
- Store API keys in Kubernetes Secrets
- Use RBAC to restrict secret access
- Rotate keys regularly (every 90 days)
- Use different keys for prod/staging

❌ **DON'T:**
- Commit API keys to git
- Share keys across environments
- Use personal API keys in production

### 2. Alert Fingerprinting

Ensure alert fingerprints are stable and unique:

```yaml
# Good: Consistent fingerprint
fingerprint: "md5(alertName + namespace + labels)"

# Bad: Timestamp-based (creates duplicate incidents)
fingerprint: "alert-" + timestamp
```

### 3. Incident Grouping

Group related alerts using labels:

```yaml
labels:
  incident_group: "database-outage"
  severity: "critical"
  namespace: "production"
```

### 4. Rate Limiting

Monitor rate limit usage:

```bash
# Check current request rate
watch -n 5 'curl -s http://alert-history:9090/metrics | grep rootly_api_requests_total'
```

If approaching limit (60/min):
- Enable incident grouping
- Increase update debounce time
- Consider Rootly plan upgrade

---

## 📚 Additional Resources

- **Rootly API Documentation**: https://docs.rootly.com/api
- **TN-052 Design Document**: `tasks/go-migration-analysis/TN-052-rootly-publisher/design.md`
- **TN-052 Requirements**: `tasks/go-migration-analysis/TN-052-rootly-publisher/requirements.md`
- **Test Suite**: `go-app/internal/infrastructure/publishing/rootly_*_test.go`

---

## 🎯 Summary

**Phase 6 Integration Achievements:**
- ✅ Enhanced PublisherFactory with Rootly-specific logic
- ✅ Shared resource management (cache, metrics, clients)
- ✅ Client pooling for multiple API keys
- ✅ Kubernetes Secret integration
- ✅ Example configurations
- ✅ Comprehensive troubleshooting guide

**Total LOC Added**: ~70 LOC (publisher.go integration)
**Configuration Files**: 1 (K8s Secret example)
**Documentation**: 1 (Integration Guide)

---

**Status**: ✅ **PHASE 6 COMPLETE**
**Next**: Phase 8 (API Documentation)
