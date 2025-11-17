# TN-66: List Targets Endpoint - Design Document

**Version:** 1.0.0
**Last Updated:** 2025-11-16
**Status:** ✅ Approved

---

## Overview

This document describes the design for the `GET /api/v2/publishing/targets` endpoint that returns a paginated list of all configured publishing targets.

---

## Response Format

### Success Response

```json
{
  "data": [
    {
      "name": "slack-alerts",
      "type": "slack",
      "url": "https://hooks.slack.com/services/YOUR_WORKSPACE_ID/YOUR_CHANNEL_ID/YOUR_WEBHOOK_TOKEN",
      "enabled": true,
      "format": "slack",
      "headers": {}
    }
  ],
  "pagination": {
    "total": 1,
    "count": 1,
    "limit": 100,
    "offset": 0,
    "has_more": false
  },
  "metadata": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-11-16T23:45:00Z",
    "processing_time_ms": 0
  }
}
```

---

## Security Considerations

- Webhook URLs are included in responses but should use placeholders in documentation
- Sensitive headers are omitted from responses
- All input parameters are validated
- Rate limiting is applied per IP

---

## Performance Targets

- P50: < 3ms
- P95: < 5ms
- P99: < 10ms
- Throughput: > 1500 req/s

---

## Implementation Notes

- Uses parameterized queries to prevent SQL injection
- Implements input validation for all query parameters
- Supports filtering, sorting, and pagination
- Returns structured error responses
- Includes request metadata in responses

