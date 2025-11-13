# Publishing Metrics & Statistics API Guide

**Comprehensive API documentation for TN-057 Publishing Metrics system.**

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Base URL](#base-url)
4. [Endpoints](#endpoints)
   - [GET /metrics](#get-apiv2publishingmetrics)
   - [GET /stats](#get-apiv2publishingstats)
   - [GET /health](#get-apiv2publishinghealth)
   - [GET /stats/{target}](#get-apiv2publishingstatstarget)
   - [GET /trends](#get-apiv2publishingtrends)
5. [Response Formats](#response-formats)
6. [Error Handling](#error-handling)
7. [Rate Limiting](#rate-limiting)
8. [Examples](#examples)

---

## Overview

The Publishing Metrics API provides programmatic access to:
- Real-time metrics from all publishing subsystems
- Aggregated statistics with human-readable summaries
- Health status for individual targets
- Per-target detailed statistics
- Trend analysis with predictive insights

**Protocol:** HTTP/1.1
**Format:** JSON
**Encoding:** UTF-8
**Timeout:** 10s (configurable)

---

## Authentication

Currently **no authentication required** (internal API).

**Future:** Bearer token authentication planned for production.

```bash
# Future example
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v2/publishing/metrics
```

---

## Base URL

```
http://localhost:8080/api/v2/publishing
```

**Production:**
```
https://alert-history.helpfull.com/api/v2/publishing
```

---

## Endpoints

### GET /api/v2/publishing/metrics

**Returns raw metrics snapshot from all registered collectors.**

#### Request

```http
GET /api/v2/publishing/metrics HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Query Parameters:** None

**Request Headers:**
- `Accept: application/json` (optional, default)

#### Response

**Status:** `200 OK`

**Headers:**
- `Content-Type: application/json`
- `X-Collection-Duration-Ms: 12.5`

**Body:**
```json
{
  "timestamp": "2025-11-13T10:30:00Z",
  "collector_count": 4,
  "collection_duration_ms": 12.5,
  "metrics": {
    "health_status{target=\"rootly-prod\",type=\"rootly\"}": 1.0,
    "health_success_rate{target=\"rootly-prod\"}": 99.5,
    "health_consecutive_failures{target=\"rootly-prod\"}": 0,
    "refresh_last_refresh_timestamp": 1699876200,
    "refresh_targets_discovered": 10,
    "discovery_total_targets": 10,
    "discovery_latency_ms": 50,
    "queue_size": 15,
    "queue_capacity": 1000,
    "queue_jobs_submitted": 10000,
    "queue_jobs_completed": 9500,
    "queue_jobs_failed": 500
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string (ISO8601) | Collection timestamp |
| `collector_count` | integer | Number of active collectors |
| `collection_duration_ms` | float | Total collection time |
| `metrics` | object | Key-value metric pairs |

#### Metric Keys Format

**Health Metrics:**
- `health_status{target="<name>",type="<type>"}` - Status (0-3)
- `health_success_rate{target="<name>"}` - Success rate (0-100%)
- `health_consecutive_failures{target="<name>"}` - Failure count

**Queue Metrics:**
- `queue_size` - Current queue size
- `queue_capacity` - Max capacity
- `queue_jobs_submitted` - Total submitted
- `queue_jobs_completed` - Total completed
- `queue_jobs_failed` - Total failed

**Discovery Metrics:**
- `discovery_total_targets` - Discovered targets count
- `discovery_latency_ms` - Discovery latency

#### Use Cases

- **Prometheus scraping** - Direct metric ingestion
- **Custom dashboards** - Raw data for visualization
- **Debugging** - Inspect all metrics at once

#### Error Responses

**500 Internal Server Error:**
```json
{
  "error": "Failed to collect metrics",
  "details": "context deadline exceeded"
}
```

---

### GET /api/v2/publishing/stats

**Returns aggregated statistics with human-readable summaries.**

#### Request

```http
GET /api/v2/publishing/stats HTTP/1.1
Host: localhost:8080
Accept: application/json
```

#### Response

**Status:** `200 OK`

**Body:**
```json
{
  "timestamp": "2025-11-13T10:30:00Z",
  "health": {
    "total_targets": 10,
    "healthy": 8,
    "degraded": 1,
    "unhealthy": 1,
    "success_rate": 95.5,
    "message": "8 of 10 targets healthy (95.5% success rate)"
  },
  "queue": {
    "size": 15,
    "capacity": 1000,
    "utilization": 1.5,
    "jobs_submitted": 10000,
    "jobs_completed": 9500,
    "jobs_failed": 500,
    "success_rate": 95.0,
    "message": "Queue: 15/1000 (1.5% utilization), 95.0% success rate"
  },
  "refresh": {
    "last_refresh": "2025-11-13T10:25:00Z",
    "next_refresh": "2025-11-13T10:30:00Z",
    "refresh_interval": "5m",
    "targets_discovered": 10,
    "state": "idle",
    "message": "Last refresh: 5m ago, discovered 10 targets"
  },
  "discovery": {
    "total_targets": 10,
    "last_discovery": "2025-11-13T10:25:00Z",
    "latency_ms": 50,
    "message": "10 targets discovered in 50ms"
  }
}
```

#### Response Fields

**Health Section:**
| Field | Type | Description |
|-------|------|-------------|
| `total_targets` | integer | Total targets configured |
| `healthy` | integer | Targets with status=healthy |
| `degraded` | integer | Targets with status=degraded |
| `unhealthy` | integer | Targets with status=unhealthy |
| `success_rate` | float | Overall success rate (%) |
| `message` | string | Human-readable summary |

**Queue Section:**
| Field | Type | Description |
|-------|------|-------------|
| `size` | integer | Current queue size |
| `capacity` | integer | Max queue capacity |
| `utilization` | float | Utilization percentage |
| `jobs_submitted` | integer | Total jobs submitted |
| `jobs_completed` | integer | Total jobs completed |
| `jobs_failed` | integer | Total jobs failed |
| `success_rate` | float | Job success rate (%) |

#### Use Cases

- **Operational dashboards** - High-level overview
- **Health checks** - Quick system status
- **Alerting** - Monitor key metrics

---

### GET /api/v2/publishing/health

**Returns health status for all publishing targets.**

#### Request

```http
GET /api/v2/publishing/health HTTP/1.1
Host: localhost:8080
```

#### Response

**Status:** `200 OK`

**Body:**
```json
{
  "timestamp": "2025-11-13T10:30:00Z",
  "status": "healthy",
  "message": "8 of 10 targets healthy",
  "checks": [
    {
      "target": "rootly-prod",
      "type": "rootly",
      "status": "healthy",
      "success_rate": 99.5,
      "consecutive_failures": 0,
      "last_check": "2025-11-13T10:29:50Z",
      "latency_ms": 150
    },
    {
      "target": "pagerduty-prod",
      "type": "pagerduty",
      "status": "healthy",
      "success_rate": 98.0,
      "consecutive_failures": 0,
      "last_check": "2025-11-13T10:29:50Z"
    },
    {
      "target": "slack-prod",
      "type": "slack",
      "status": "degraded",
      "success_rate": 85.0,
      "consecutive_failures": 1,
      "last_check": "2025-11-13T10:29:45Z",
      "latency_ms": 5500
    }
  ]
}
```

#### Response Fields

**Root Level:**
| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Overall status (healthy/degraded/unhealthy) |
| `message` | string | Summary message |
| `checks` | array | Health checks for each target |

**Check Object:**
| Field | Type | Description |
|-------|------|-------------|
| `target` | string | Target name |
| `type` | string | Target type (rootly/pagerduty/slack/webhook) |
| `status` | string | Health status (healthy/degraded/unhealthy/unknown) |
| `success_rate` | float | Success rate (0-100%) |
| `consecutive_failures` | integer | Consecutive failure count |
| `last_check` | string (ISO8601) | Last check timestamp |
| `latency_ms` | integer (optional) | Last check latency |

#### Status Values

| Status | Meaning | Criteria |
|--------|---------|----------|
| `healthy` | Target operational | consecutive_failures=0, latency<5s |
| `degraded` | Target slow | latency>=5s OR 1-2 failures |
| `unhealthy` | Target down | consecutive_failures>=3 |
| `unknown` | No checks yet | Initial state |

#### Use Cases

- **Health monitoring** - Track target availability
- **Alerting** - Trigger alerts on unhealthy targets
- **Debugging** - Identify problematic targets

---

### GET /api/v2/publishing/stats/{target}

**Returns detailed statistics for a specific target.**

#### Request

```http
GET /api/v2/publishing/stats/rootly-prod HTTP/1.1
Host: localhost:8080
```

**Path Parameters:**
- `target` (required) - Target name (e.g., "rootly-prod")

#### Response

**Status:** `200 OK`

**Body:**
```json
{
  "target_name": "rootly-prod",
  "timestamp": "2025-11-13T10:30:00Z",
  "health": {
    "status": "healthy",
    "success_rate": 99.5,
    "consecutive_failures": 0
  },
  "jobs": {
    "processed": 1000,
    "succeeded": 995,
    "failed": 5,
    "success_rate": 99.5
  },
  "metrics": {
    "health_status{target=\"rootly-prod\",type=\"rootly\"}": 1.0,
    "health_success_rate{target=\"rootly-prod\"}": 99.5,
    "health_consecutive_failures{target=\"rootly-prod\"}": 0,
    "queue_jobs_processed{target=\"rootly-prod\"}": 1000,
    "queue_jobs_completed{target=\"rootly-prod\"}": 995,
    "queue_jobs_failed{target=\"rootly-prod\"}": 5
  }
}
```

#### Error Responses

**400 Bad Request:**
```json
{
  "error": "Missing target name in path"
}
```

**404 Not Found:**
```json
{
  "error": "Target not found",
  "target": "unknown-target"
}
```

#### Use Cases

- **Per-target debugging** - Deep dive into specific target
- **Performance analysis** - Track target-specific metrics
- **Capacity planning** - Analyze target load

---

### GET /api/v2/publishing/trends

**Returns trend analysis for the publishing system.**

#### Request

```http
GET /api/v2/publishing/trends HTTP/1.1
Host: localhost:8080
```

**Query Parameters:** None

#### Response

**Status:** `200 OK`

**Body:**
```json
{
  "timestamp": "2025-11-13T10:30:00Z",
  "trends": {
    "success_rate": {
      "trend": "stable",
      "current": 95.5,
      "ema": 95.3,
      "std_dev": 1.2,
      "change_pct": 0.21
    },
    "latency": {
      "trend": "improving",
      "current_ms": 150,
      "ema_ms": 180,
      "std_dev_ms": 20,
      "change_pct": -16.67
    },
    "error_spike": {
      "detected": false,
      "current_errors": 5,
      "ema_errors": 4.8,
      "threshold": 15,
      "std_dev": 3.2
    },
    "queue_growth": {
      "rate_per_min": 2.5,
      "trend": "stable",
      "ema_rate": 2.4
    }
  },
  "summary": "System stable: 95.5% success rate (stable), 150ms latency (improving), no error spikes, queue growth 2.5 jobs/min (stable)"
}
```

#### Response Fields

**Success Rate Trend:**
| Field | Type | Description |
|-------|------|-------------|
| `trend` | string | increasing / stable / decreasing |
| `current` | float | Current success rate (%) |
| `ema` | float | Exponential moving average |
| `std_dev` | float | Standard deviation |
| `change_pct` | float | % change from EMA |

**Latency Trend:**
| Field | Type | Description |
|-------|------|-------------|
| `trend` | string | improving / stable / degrading |
| `current_ms` | float | Current latency (ms) |
| `ema_ms` | float | EMA latency |
| `std_dev_ms` | float | Standard deviation |
| `change_pct` | float | % change from EMA |

**Error Spike:**
| Field | Type | Description |
|-------|------|-------------|
| `detected` | boolean | Spike detected (>3σ) |
| `current_errors` | integer | Current error count |
| `ema_errors` | float | EMA error count |
| `threshold` | float | Detection threshold (EMA + 3σ) |

**Queue Growth:**
| Field | Type | Description |
|-------|------|-------------|
| `rate_per_min` | float | Jobs/minute rate |
| `trend` | string | increasing / stable / decreasing |
| `ema_rate` | float | EMA rate |

#### Trend Classification

**Success Rate:**
- `increasing`: >+5% from EMA
- `decreasing`: <-5% from EMA
- `stable`: within ±5%

**Latency:**
- `improving`: >+15% decrease from EMA
- `degrading`: >+15% increase from EMA
- `stable`: within ±15%

**Error Spike:**
- `detected`: current > EMA + 3σ
- `not detected`: within normal range

#### Use Cases

- **Predictive monitoring** - Detect degradation early
- **Capacity planning** - Track growth trends
- **Anomaly detection** - Identify unusual patterns

---

## Response Formats

### Success Response

**Status:** `200 OK`
**Content-Type:** `application/json`

### Error Response

**Format:**
```json
{
  "error": "Error message",
  "details": "Additional context (optional)"
}
```

**Status Codes:**
- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

---

## Error Handling

### Common Errors

#### 1. Collection Timeout

**Response:**
```json
{
  "error": "Failed to collect metrics",
  "details": "context deadline exceeded"
}
```

**Cause:** Collector took >10s
**Solution:** Check subsystem availability

#### 2. Target Not Found

**Response:**
```json
{
  "error": "Target not found",
  "target": "unknown-target"
}
```

**Cause:** Invalid target name
**Solution:** Use `GET /health` to list valid targets

#### 3. Method Not Allowed

**Response:**
```json
{
  "error": "Method not allowed"
}
```

**Cause:** Wrong HTTP method (e.g., POST to GET endpoint)
**Solution:** Use correct HTTP method

---

## Rate Limiting

**Current:** No rate limiting

**Future:** 100 requests/minute per IP

**Headers (future):**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699876260
```

---

## Examples

### Example 1: Health Check Script

```bash
#!/bin/bash

ENDPOINT="http://localhost:8080/api/v2/publishing/health"

# Check health status
STATUS=$(curl -s $ENDPOINT | jq -r '.status')

if [ "$STATUS" != "healthy" ]; then
  echo "ALERT: Publishing system unhealthy"
  curl -s $ENDPOINT | jq '.checks[] | select(.status != "healthy")'
  exit 1
fi

echo "OK: Publishing system healthy"
```

### Example 2: Prometheus Scrape

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'publishing-metrics'
    metrics_path: '/api/v2/publishing/metrics'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8080']
```

### Example 3: Dashboard Widget (JavaScript)

```javascript
async function fetchPublishingStats() {
  const response = await fetch('/api/v2/publishing/stats');
  const data = await response.json();

  document.getElementById('queue-size').textContent = data.queue.size;
  document.getElementById('success-rate').textContent =
    `${data.health.success_rate.toFixed(1)}%`;
}

// Update every 5 seconds
setInterval(fetchPublishingStats, 5000);
```

### Example 4: Trend Alert

```bash
#!/bin/bash

TRENDS=$(curl -s http://localhost:8080/api/v2/publishing/trends)

# Check for error spikes
SPIKE_DETECTED=$(echo $TRENDS | jq -r '.trends.error_spike.detected')

if [ "$SPIKE_DETECTED" = "true" ]; then
  echo "ALERT: Error spike detected!"
  echo $TRENDS | jq '.trends.error_spike'
fi

# Check latency degradation
LATENCY_TREND=$(echo $TRENDS | jq -r '.trends.latency.trend')

if [ "$LATENCY_TREND" = "degrading" ]; then
  echo "WARNING: Latency degrading"
  echo $TRENDS | jq '.trends.latency'
fi
```

---

## Versioning

**Current Version:** v2
**Base Path:** `/api/v2/publishing`

**Breaking Changes:** Will increment major version (v3)

---

## Support

**Issues:** Create ticket in JIRA (TN-057)
**Questions:** #alert-history-service Slack channel

---

**Last Updated:** 2025-11-13
