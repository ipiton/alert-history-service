# Alert History API Integration Guide

**Version**: 2.0.0
**Last Updated**: 2025-11-16

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Authentication](#authentication)
3. [Basic Usage](#basic-usage)
4. [Filtering](#filtering)
5. [Pagination](#pagination)
6. [Sorting](#sorting)
7. [Advanced Features](#advanced-features)
8. [Error Handling](#error-handling)
9. [Best Practices](#best-practices)
10. [Performance Tips](#performance-tips)

---

## Quick Start

### 1. Get API Key

Contact your administrator to obtain an API key.

### 2. Make Your First Request

```bash
curl -H "X-API-Key: your-api-key" \
  "https://api.alert-history.example.com/api/v2/history?page=1&per_page=10"
```

### 3. Parse Response

```json
{
  "alerts": [
    {
      "fingerprint": "a1b2c3d4...",
      "alert_name": "HighCPUUsage",
      "status": "firing",
      "severity": "critical",
      "starts_at": "2025-11-16T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "per_page": 10,
  "total_pages": 10,
  "has_next": true,
  "has_prev": false
}
```

---

## Authentication

All requests require API key authentication:

```bash
# Header format
X-API-Key: your-api-key-here
```

### Example

```bash
curl -H "X-API-Key: abc123def456" \
  "https://api.alert-history.example.com/api/v2/history"
```

---

## Basic Usage

### Get All Alerts

```bash
GET /api/v2/history
```

### Get Specific Alert Timeline

```bash
GET /api/v2/history/{fingerprint}
```

### Get Top Alerts

```bash
GET /api/v2/history/top?limit=10
```

### Get Recent Alerts

```bash
GET /api/v2/history/recent?limit=50
```

### Get Statistics

```bash
GET /api/v2/history/stats?from=2025-11-01T00:00:00Z&to=2025-11-16T23:59:59Z
```

---

## Filtering

### Status Filter

```bash
GET /api/v2/history?status=firing
GET /api/v2/history?status=resolved
```

### Severity Filter

```bash
GET /api/v2/history?severity=critical
GET /api/v2/history?severity=warning
```

### Namespace Filter

```bash
GET /api/v2/history?namespace=production
```

### Time Range Filter

```bash
GET /api/v2/history?from=2025-11-01T00:00:00Z&to=2025-11-16T23:59:59Z
```

### Combined Filters

```bash
GET /api/v2/history?status=firing&severity=critical&namespace=production&from=2025-11-01T00:00:00Z
```

### Label Filters

```bash
# Exact match
GET /api/v2/history?labels_exact[instance]=server-01

# Regex match
GET /api/v2/history?labels_regex[env]=prod.*

# Exists
GET /api/v2/history?labels_exists=instance
```

---

## Pagination

### Basic Pagination

```bash
GET /api/v2/history?page=1&per_page=50
```

### Pagination Metadata

Response includes:
- `total`: Total number of alerts
- `page`: Current page number
- `per_page`: Items per page
- `total_pages`: Total number of pages
- `has_next`: Whether there is a next page
- `has_prev`: Whether there is a previous page

### Limits

- **Max Page**: 10,000
- **Max Per-Page**: 1,000
- **Default**: Page 1, 50 items per page

---

## Sorting

### Sort by Field

```bash
GET /api/v2/history?sort_field=starts_at&sort_order=desc
```

### Available Sort Fields

- `starts_at` (default)
- `ends_at`
- `created_at`
- `status`
- `severity`

### Sort Order

- `asc`: Ascending
- `desc`: Descending (default)

---

## Advanced Features

### Search

```bash
POST /api/v2/history/search
Content-Type: application/json

{
  "query": "HighCPUUsage",
  "pagination": {
    "page": 1,
    "per_page": 50
  }
}
```

### Flapping Detection

```bash
GET /api/v2/history/flapping?threshold=5&from=2025-11-01T00:00:00Z
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid query parameter",
    "request_id": "req-12345",
    "timestamp": "2025-11-16T12:00:00Z"
  }
}
```

### Common Error Codes

- `VALIDATION_ERROR` (400): Invalid input
- `AUTHENTICATION_ERROR` (401): Invalid API key
- `AUTHORIZATION_ERROR` (403): Insufficient permissions
- `RATE_LIMIT_EXCEEDED` (429): Too many requests
- `INTERNAL_ERROR` (500): Server error

### Error Handling Example

```python
import requests

response = requests.get(
    "https://api.alert-history.example.com/api/v2/history",
    headers={"X-API-Key": "your-key"}
)

if response.status_code == 200:
    data = response.json()
    # Process data
elif response.status_code == 429:
    retry_after = response.headers.get("Retry-After")
    # Wait and retry
else:
    error = response.json()["error"]
    print(f"Error: {error['code']} - {error['message']}")
```

---

## Best Practices

### 1. Use Appropriate Page Size

- **Small datasets**: 10-50 items per page
- **Large datasets**: 50-100 items per page
- **Maximum**: 1000 items per page

### 2. Cache Responses

- Cache responses for frequently accessed queries
- Use `If-None-Match` header for conditional requests (future)

### 3. Handle Rate Limits

- Implement exponential backoff
- Respect `Retry-After` header
- Monitor rate limit headers

### 4. Use Filters Efficiently

- Combine filters to reduce result set
- Use indexed fields (status, severity, namespace)
- Limit time ranges to reasonable windows

### 5. Error Handling

- Always check status codes
- Log error details for debugging
- Implement retry logic for transient errors

---

## Performance Tips

### 1. Use Indexed Filters

Prefer filters on indexed columns:
- ✅ `status=firing`
- ✅ `severity=critical`
- ✅ `namespace=production`
- ✅ `from=...&to=...` (time range)

### 2. Limit Time Ranges

- **Recommended**: Last 24-48 hours
- **Maximum**: 90 days
- **Avoid**: Very large time ranges (> 30 days)

### 3. Use Pagination

- Don't fetch all results at once
- Use appropriate page size (50-100)
- Avoid deep pagination (> 1000 pages)

### 4. Cache Popular Queries

- Cache frequently accessed queries
- Use cache TTL based on data freshness needs
- Monitor cache hit rate

### 5. Monitor Performance

- Track request latency
- Monitor error rates
- Watch for rate limit violations

---

## Code Examples

### Python

```python
import requests
from datetime import datetime, timedelta

class AlertHistoryClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {"X-API-Key": api_key}

    def get_history(self, status=None, severity=None, page=1, per_page=50):
        params = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        if severity:
            params["severity"] = severity

        response = requests.get(
            f"{self.base_url}/api/v2/history",
            headers=self.headers,
            params=params
        )
        response.raise_for_status()
        return response.json()

    def get_recent_alerts(self, limit=50):
        response = requests.get(
            f"{self.base_url}/api/v2/history/recent",
            headers=self.headers,
            params={"limit": limit}
        )
        response.raise_for_status()
        return response.json()

# Usage
client = AlertHistoryClient(
    "https://api.alert-history.example.com",
    "your-api-key"
)

# Get firing critical alerts
alerts = client.get_history(status="firing", severity="critical")
print(f"Found {alerts['total']} alerts")

# Get recent alerts
recent = client.get_recent_alerts(limit=20)
print(f"Recent alerts: {len(recent['alerts'])}")
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type AlertHistoryClient struct {
    BaseURL string
    APIKey  string
}

func (c *AlertHistoryClient) GetHistory(status, severity string, page, perPage int) (*HistoryResponse, error) {
    u, _ := url.Parse(c.BaseURL + "/api/v2/history")
    q := u.Query()
    q.Set("page", fmt.Sprintf("%d", page))
    q.Set("per_page", fmt.Sprintf("%d", perPage))
    if status != "" {
        q.Set("status", status)
    }
    if severity != "" {
        q.Set("severity", severity)
    }
    u.RawQuery = q.Encode()

    req, _ := http.NewRequest("GET", u.String(), nil)
    req.Header.Set("X-API-Key", c.APIKey)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result HistoryResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// Usage
client := &AlertHistoryClient{
    BaseURL: "https://api.alert-history.example.com",
    APIKey:  "your-api-key",
}

alerts, err := client.GetHistory("firing", "critical", 1, 50)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d alerts\n", alerts.Total)
```

---

## Troubleshooting

### High Latency

- Check cache hit rate
- Review query filters
- Verify indexes are used
- Check database load

### Rate Limit Errors

- Reduce request frequency
- Implement exponential backoff
- Use caching to reduce requests
- Contact administrator for higher limits

### Authentication Errors

- Verify API key is correct
- Check API key hasn't expired
- Ensure header name is `X-API-Key`
- Contact administrator if issues persist

---

**For more information**, see:
- [OpenAPI Specification](../api/history/openapi.yaml)
- [Architecture Decision Records](../adrs/)
- [Runbooks](../runbooks/)
