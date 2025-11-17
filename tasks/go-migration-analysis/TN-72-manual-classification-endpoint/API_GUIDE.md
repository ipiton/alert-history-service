# TN-72: Manual Classification API Guide

## Overview

The `POST /api/v2/classification/classify` endpoint provides manual alert classification functionality with support for force re-classification and cache bypass.

## Endpoint

```
POST /api/v2/classification/classify
```

## Authentication

Requires API key authentication via `Authorization` header:
```
Authorization: Bearer <api-key>
```

## Request

### Headers
- `Content-Type: application/json`
- `Authorization: Bearer <api-key>` (required)
- `X-Request-ID: <uuid>` (optional, for request tracking)

### Request Body

```json
{
  "alert": {
    "fingerprint": "string (required)",
    "alert_name": "string (required)",
    "status": "firing|resolved (required)",
    "starts_at": "RFC3339 timestamp (required)",
    "labels": {
      "key": "value"
    },
    "annotations": {
      "key": "value"
    },
    "generator_url": "https://prometheus.example.com (optional)"
  },
  "force": false
}
```

### Fields

#### `alert` (required)
- **fingerprint**: Unique alert identifier (required)
- **alert_name**: Alert name (required)
- **status**: Alert status - `firing` or `resolved` (required)
- **starts_at**: Alert start time in RFC3339 format (required)
- **labels**: Alert labels (optional)
- **annotations**: Alert annotations (optional)
- **generator_url**: Prometheus generator URL (optional, must be HTTP/HTTPS)

#### `force` (optional, default: false)
- If `true`: Bypasses cache and forces new classification
- If `false`: Checks cache first, then classifies if cache miss

## Response

### Success Response (200 OK)

```json
{
  "result": {
    "severity": "critical|warning|info|noise",
    "confidence": 0.95,
    "reasoning": "Classification reasoning",
    "recommendations": ["Action 1", "Action 2"],
    "processing_time": 0.15,
    "metadata": {
      "model": "gpt-4",
      "source": "llm"
    }
  },
  "processing_time": "50.00ms",
  "cached": false,
  "model": "gpt-4",
  "timestamp": "2025-11-17T21:00:00Z"
}
```

### Error Responses

#### 400 Bad Request - Validation Error
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid alert: fingerprint is required",
    "request_id": "req-123",
    "timestamp": "2025-11-17T21:00:00Z"
  }
}
```

#### 504 Gateway Timeout - Classification Timeout
```json
{
  "error": {
    "code": "CLASSIFICATION_TIMEOUT",
    "message": "LLM classification request timed out",
    "request_id": "req-123",
    "timestamp": "2025-11-17T21:00:00Z"
  }
}
```

#### 503 Service Unavailable - LLM Service Unavailable
```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "LLM service unavailable",
    "request_id": "req-123",
    "timestamp": "2025-11-17T21:00:00Z"
  }
}
```

#### 500 Internal Server Error - Classification Failed
```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Classification failed: <error details>",
    "request_id": "req-123",
    "timestamp": "2025-11-17T21:00:00Z"
  }
}
```

## Examples

### Example 1: Basic Classification (Cache Enabled)

```bash
curl -X POST https://api.example.com/api/v2/classification/classify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <api-key>" \
  -d '{
    "alert": {
      "fingerprint": "alert-123",
      "alert_name": "HighCPUUsage",
      "status": "firing",
      "starts_at": "2025-11-17T21:00:00Z",
      "labels": {
        "severity": "warning",
        "namespace": "production"
      }
    }
  }'
```

### Example 2: Force Re-classification

```bash
curl -X POST https://api.example.com/api/v2/classification/classify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <api-key>" \
  -d '{
    "alert": {
      "fingerprint": "alert-123",
      "alert_name": "HighCPUUsage",
      "status": "firing",
      "starts_at": "2025-11-17T21:00:00Z"
    },
    "force": true
  }'
```

### Example 3: Resolved Alert Classification

```bash
curl -X POST https://api.example.com/api/v2/classification/classify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <api-key>" \
  -d '{
    "alert": {
      "fingerprint": "alert-123",
      "alert_name": "HighCPUUsage",
      "status": "resolved",
      "starts_at": "2025-11-17T20:00:00Z",
      "ends_at": "2025-11-17T21:00:00Z"
    }
  }'
```

## Performance

- **Cache Hit**: ~5-10ms (p95)
- **Cache Miss (LLM)**: ~100-500ms (p95)
- **Cache Miss (Fallback)**: ~10-50ms (p95)
- **Force Classification**: ~100-500ms (p95)

## Rate Limiting

Default rate limit: 60 requests/minute per API key.

Rate limit headers:
- `X-RateLimit-Limit: 60`
- `X-RateLimit-Remaining: 59`
- `X-RateLimit-Reset: 1637100000`

## Caching

The endpoint uses a two-tier caching system:
- **L1 Cache**: In-memory cache (fast, ~50ns lookup)
- **L2 Cache**: Redis cache (persistent, ~1-5ms lookup)

Cache TTL: 24 hours (configurable)

### Cache Behavior

- **force=false**: Checks cache first, classifies only on cache miss
- **force=true**: Invalidates cache, forces new classification

## Best Practices

1. **Use force flag sparingly**: Only when you need fresh classification
2. **Monitor cache hit rate**: High cache hit rate indicates good performance
3. **Handle timeouts gracefully**: Implement retry logic with exponential backoff
4. **Use request IDs**: Include `X-Request-ID` header for request tracking
5. **Validate alerts**: Ensure all required fields are present before sending

## Monitoring

### Prometheus Metrics

- `api_http_requests_total{method="POST",endpoint="/classification/classify",status="200"}`
- `api_http_request_duration_seconds{method="POST",endpoint="/classification/classify"}`
- `alert_history_business_classification_duration_seconds{source="cache|llm|fallback"}`
- `alert_history_business_classification_l1_cache_hits_total`
- `alert_history_business_classification_l2_cache_hits_total`

### Key Metrics to Monitor

- **Request Rate**: Requests per second
- **Error Rate**: Percentage of failed requests
- **Cache Hit Rate**: Percentage of cache hits
- **P95 Latency**: 95th percentile response time
- **Timeout Rate**: Percentage of timeout errors

## Troubleshooting

See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues and solutions.

## Related Endpoints

- `GET /api/v2/classification/stats` - Get classification statistics
- `GET /api/v2/classification/models` - List available LLM models
