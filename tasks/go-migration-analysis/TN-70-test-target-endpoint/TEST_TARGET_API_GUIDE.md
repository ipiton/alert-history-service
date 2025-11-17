# Test Target API Guide

## Overview

The `POST /api/v2/publishing/targets/{name}/test` endpoint allows you to test publishing target connectivity by sending a test alert. This is useful for:

- **Configuration Validation**: Verify target setup before production deployment
- **Diagnostics**: Troubleshoot connectivity, authentication, or formatting issues
- **Health Monitoring**: Regular checks of target availability
- **CI/CD Integration**: Automated validation in deployment pipelines

## Quick Start

### Basic Test (Minimal Request)

```bash
curl -X POST http://localhost:8080/api/v2/publishing/targets/rootly-prod/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{}'
```

**Response:**
```json
{
  "success": true,
  "message": "Test alert sent",
  "target_name": "rootly-prod",
  "status_code": 200,
  "response_time_ms": 150,
  "test_timestamp": "2025-11-17T19:00:00Z"
}
```

### Custom Test Alert

```bash
curl -X POST http://localhost:8080/api/v2/publishing/targets/rootly-prod/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "alert_name": "CustomTestAlert",
    "test_alert": {
      "labels": {
        "alertname": "CustomTestAlert",
        "severity": "warning"
      },
      "annotations": {
        "summary": "Custom test alert"
      }
    },
    "timeout_seconds": 60
  }'
```

## Request Parameters

### Path Parameters

- **name** (required): Target name as configured in K8s secrets
  - Example: `rootly-prod`, `pagerduty-critical`, `slack-alerts`

### Request Body (Optional)

```json
{
  "alert_name": "string (optional)",
  "test_alert": {
    "fingerprint": "string (optional)",
    "labels": {
      "key": "value"
    },
    "annotations": {
      "key": "value"
    },
    "status": "firing|resolved"
  },
  "timeout_seconds": 30
}
```

#### Fields

- **alert_name** (string, optional): Custom alert name (default: "TestAlert")
- **test_alert** (object, optional): Custom test alert payload
  - **fingerprint** (string, optional): Custom fingerprint
  - **labels** (object, optional): Alert labels (test label added automatically)
  - **annotations** (object, optional): Alert annotations
  - **status** (string, optional): "firing" or "resolved" (default: "firing")
- **timeout_seconds** (integer, optional): Timeout in seconds (default: 30, min: 1, max: 300)

## Response Format

### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Test alert sent",
  "target_name": "rootly-prod",
  "status_code": 200,
  "response_time_ms": 150,
  "test_timestamp": "2025-11-17T19:00:00Z"
}
```

### Failure Response (200 OK with success: false)

```json
{
  "success": false,
  "message": "Test failed",
  "target_name": "rootly-prod",
  "response_time_ms": 200,
  "error": "HTTP 401: Unauthorized",
  "test_timestamp": "2025-11-17T19:00:00Z"
}
```

### Error Responses

#### 400 Bad Request

```json
{
  "error": "validation_error",
  "message": "Timeout must be between 1 and 300 seconds",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### 404 Not Found

```json
{
  "error": "not_found",
  "message": "Target 'invalid-target' does not exist",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Examples

### Using curl

#### Basic Test
```bash
curl -X POST http://localhost:8080/api/v2/publishing/targets/rootly-prod/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key"
```

#### Custom Alert with Timeout
```bash
curl -X POST http://localhost:8080/api/v2/publishing/targets/pagerduty-critical/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "alert_name": "CriticalTest",
    "test_alert": {
      "labels": {
        "alertname": "CriticalTest",
        "severity": "critical"
      },
      "status": "firing"
    },
    "timeout_seconds": 10
  }'
```

#### Test Resolved Alert
```bash
curl -X POST http://localhost:8080/api/v2/publishing/targets/slack-alerts/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "test_alert": {
      "status": "resolved"
    }
  }'
```

### Using Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type TestTargetRequest struct {
    AlertName      string           `json:"alert_name,omitempty"`
    TestAlert      *CustomTestAlert `json:"test_alert,omitempty"`
    TimeoutSeconds int              `json:"timeout_seconds,omitempty"`
}

type CustomTestAlert struct {
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
    Status      string            `json:"status,omitempty"`
}

type TestTargetResponse struct {
    Success        bool      `json:"success"`
    Message        string    `json:"message"`
    TargetName     string    `json:"target_name"`
    StatusCode     *int      `json:"status_code,omitempty"`
    ResponseTimeMs int       `json:"response_time_ms"`
    Error          string    `json:"error,omitempty"`
    TestTimestamp  time.Time `json:"test_timestamp"`
}

func TestTarget(targetName string, apiKey string) (*TestTargetResponse, error) {
    req := TestTargetRequest{
        TimeoutSeconds: 30,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest("POST",
        fmt.Sprintf("http://localhost:8080/api/v2/publishing/targets/%s/test", targetName),
        bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-API-Key", apiKey)

    client := &http.Client{Timeout: 35 * time.Second}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result TestTargetResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}
```

### Using Python

```python
import requests
import json
from datetime import datetime

def test_target(target_name, api_key, timeout=30, custom_alert=None):
    """
    Test publishing target connectivity.

    Args:
        target_name: Target name (e.g., 'rootly-prod')
        api_key: API key for authentication
        timeout: Timeout in seconds (default: 30)
        custom_alert: Optional custom alert payload

    Returns:
        dict: Test result with success status and details
    """
    url = f"http://localhost:8080/api/v2/publishing/targets/{target_name}/test"

    headers = {
        "Content-Type": "application/json",
        "X-API-Key": api_key
    }

    payload = {
        "timeout_seconds": timeout
    }

    if custom_alert:
        payload["test_alert"] = custom_alert

    response = requests.post(url, json=payload, headers=headers, timeout=35)
    response.raise_for_status()

    return response.json()

# Example usage
result = test_target(
    "rootly-prod",
    "your-api-key",
    timeout=60,
    custom_alert={
        "labels": {
            "alertname": "CustomTestAlert",
            "severity": "warning"
        },
        "annotations": {
            "summary": "Custom test alert"
        }
    }
)

print(f"Test {'succeeded' if result['success'] else 'failed'}")
print(f"Response time: {result['response_time_ms']}ms")
if result.get('error'):
    print(f"Error: {result['error']}")
```

### Using JavaScript/Node.js

```javascript
const axios = require('axios');

async function testTarget(targetName, apiKey, options = {}) {
  const {
    alertName,
    testAlert,
    timeoutSeconds = 30
  } = options;

  const url = `http://localhost:8080/api/v2/publishing/targets/${targetName}/test`;

  const payload = {
    timeout_seconds: timeoutSeconds
  };

  if (alertName) {
    payload.alert_name = alertName;
  }

  if (testAlert) {
    payload.test_alert = testAlert;
  }

  try {
    const response = await axios.post(url, payload, {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey
      },
      timeout: 35000
    });

    return response.data;
  } catch (error) {
    if (error.response) {
      throw new Error(`Test failed: ${error.response.data.message}`);
    }
    throw error;
  }
}

// Example usage
testTarget('rootly-prod', 'your-api-key', {
  alertName: 'CustomTestAlert',
  testAlert: {
    labels: {
      alertname: 'CustomTestAlert',
      severity: 'warning'
    },
    status: 'firing'
  },
  timeoutSeconds: 60
})
  .then(result => {
    console.log(`Test ${result.success ? 'succeeded' : 'failed'}`);
    console.log(`Response time: ${result.response_time_ms}ms`);
  })
  .catch(error => {
    console.error('Error:', error.message);
  });
```

## Troubleshooting

### Common Issues

#### 1. Target Not Found (404)

**Problem**: Target name doesn't exist in discovery cache

**Solution**:
- Verify target is configured in K8s secrets with label `publishing-target=true`
- Check target name spelling (case-sensitive)
- Trigger target refresh: `POST /api/v2/publishing/targets/refresh`

**Example Error**:
```json
{
  "error": "not_found",
  "message": "Target 'invalid-target' does not exist"
}
```

#### 2. Target Disabled (200 with success: false)

**Problem**: Target exists but is disabled

**Solution**:
- Check target configuration in K8s secret
- Verify `enabled: true` in target JSON
- Re-enable target and refresh discovery

**Example Response**:
```json
{
  "success": false,
  "message": "Target is disabled",
  "target_name": "rootly-prod"
}
```

#### 3. Test Timeout (200 with success: false)

**Problem**: Target API didn't respond within timeout

**Solution**:
- Increase `timeout_seconds` (max 300)
- Check target API availability
- Verify network connectivity
- Check firewall rules

**Example Response**:
```json
{
  "success": false,
  "message": "Test timeout",
  "error": "Test timeout after 30 seconds"
}
```

#### 4. Publishing Failure (200 with success: false)

**Problem**: Target API returned error

**Common Causes**:
- **401 Unauthorized**: Invalid API key or authentication
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Invalid target URL
- **500 Internal Server Error**: Target API issue

**Solution**:
- Verify authentication credentials in K8s secret
- Check target API logs
- Validate target URL format
- Contact target API support

**Example Response**:
```json
{
  "success": false,
  "message": "Test failed",
  "error": "HTTP 401: Unauthorized"
}
```

#### 5. Invalid Request Body (400)

**Problem**: Request validation failed

**Common Causes**:
- `timeout_seconds` outside range (1-300)
- Invalid JSON structure
- Invalid alert status (must be "firing" or "resolved")

**Solution**:
- Validate request body against schema
- Check timeout range
- Verify JSON syntax

**Example Error**:
```json
{
  "error": "validation_error",
  "message": "Timeout must be between 1 and 300 seconds"
}
```

### Performance Tips

1. **Use Short Timeouts for Quick Tests**: Set `timeout_seconds: 5` for fast validation
2. **Monitor Response Times**: Track `response_time_ms` for performance monitoring
3. **Batch Testing**: Test multiple targets in parallel (use separate requests)
4. **Cache Results**: Don't test same target more than once per minute

### Best Practices

1. **Use in CI/CD**: Validate targets before deployment
2. **Regular Health Checks**: Schedule periodic tests (e.g., every 5 minutes)
3. **Monitor Metrics**: Track test success rates and response times
4. **Error Handling**: Always check `success` field, not just HTTP status
5. **Custom Alerts**: Use custom alerts to test specific scenarios
6. **Timeout Configuration**: Set appropriate timeout based on target API latency

## Rate Limits

- **Default**: 10 requests/minute per IP
- **Operator Role**: Higher limits may apply
- **Exceeding Limit**: Returns `429 Too Many Requests`

## Security

- **Authentication**: Required (API key or JWT token)
- **Authorization**: Operator+ role required
- **Input Validation**: All inputs validated
- **Error Sanitization**: No sensitive data in error messages

## Monitoring

### Prometheus Metrics

The endpoint exposes metrics via existing middleware:
- `http_requests_total{method="POST", path="/api/v2/publishing/targets/{name}/test", status="200"}`
- `http_request_duration_seconds{method="POST", path="/api/v2/publishing/targets/{name}/test"}`

### Logging

Structured logs include:
- Request ID (UUID)
- Target name
- Success/failure status
- Response time
- Error details (if any)

Example log:
```
INFO Test target completed request_id="550e8400-e29b-41d4-a716-446655440000" target=rootly-prod success=true response_time_ms=150 publish_duration_ms=120
```

## Related Endpoints

- `GET /api/v2/publishing/targets` - List all targets
- `POST /api/v2/publishing/targets/refresh` - Refresh target discovery
- `GET /api/v2/publishing/targets/{name}` - Get target details

## Support

For issues or questions:
- Check troubleshooting guide above
- Review OpenAPI spec: `/docs` or `/openapi.json`
- Contact: team@alerthistory.io
