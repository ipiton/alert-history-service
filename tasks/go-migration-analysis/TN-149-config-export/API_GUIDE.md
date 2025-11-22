# TN-149: GET /api/v2/config - API Usage Guide

**Date**: 2025-11-21
**Endpoint**: `GET /api/v2/config`
**Status**: ✅ PRODUCTION-READY

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Query Parameters](#query-parameters)
3. [Response Formats](#response-formats)
4. [Error Handling](#error-handling)
5. [Security](#security)
6. [Examples](#examples)
7. [Troubleshooting](#troubleshooting)

---

## 🚀 Quick Start

### Basic Request

```bash
# Export configuration as JSON (default)
curl http://localhost:8080/api/v2/config
```

### Response

```json
{
  "status": "success",
  "data": {
    "version": "a1b2c3d4e5f6...",
    "source": "file",
    "loaded_at": "2025-11-21T10:00:00Z",
    "config_file_path": "/etc/config.yaml",
    "config": {
      "server": { "port": 8080, "host": "localhost" },
      "database": { "host": "localhost", "password": "***REDACTED***" },
      ...
    }
  }
}
```

---

## 🔧 Query Parameters

### `format` (optional)

**Type**: `string`
**Default**: `json`
**Values**: `json`, `yaml`

Определяет формат ответа.

**Examples**:
```bash
# JSON (default)
curl http://localhost:8080/api/v2/config

# YAML
curl http://localhost:8080/api/v2/config?format=yaml
```

### `sanitize` (optional)

**Type**: `boolean`
**Default**: `true`
**Values**: `true`, `false`

Включает/отключает санитизацию секретов. По умолчанию секреты скрываются.

**⚠️ Security**: `sanitize=false` требует admin доступа.

**Examples**:
```bash
# Sanitized (default)
curl http://localhost:8080/api/v2/config

# Unsanitized (admin only)
curl http://localhost:8080/api/v2/config?sanitize=false
```

### `sections` (optional)

**Type**: `string` (comma-separated)
**Default**: (all sections)

Фильтрует конфигурацию по секциям. Доступные секции:

- `server` - Server configuration
- `database` - Database configuration
- `redis` - Redis configuration
- `llm` - LLM configuration
- `log` - Logging configuration
- `cache` - Cache configuration
- `lock` - Distributed lock configuration
- `app` - Application configuration
- `metrics` - Metrics configuration
- `webhook` - Webhook configuration

**Examples**:
```bash
# Single section
curl "http://localhost:8080/api/v2/config?sections=server"

# Multiple sections
curl "http://localhost:8080/api/v2/config?sections=server,database,redis"

# All sections (default, omit parameter)
curl http://localhost:8080/api/v2/config
```

---

## 📄 Response Formats

### JSON Format (default)

**Content-Type**: `application/json`

```json
{
  "status": "success",
  "data": {
    "version": "a1b2c3d4e5f6...",
    "source": "file",
    "loaded_at": "2025-11-21T10:00:00Z",
    "config_file_path": "/etc/config.yaml",
    "config": {
      "Server": {
        "port": 8080,
        "host": "localhost"
      },
      "Database": {
        "host": "localhost",
        "port": 5432,
        "password": "***REDACTED***"
      }
    }
  }
}
```

### YAML Format

**Content-Type**: `text/yaml`

```yaml
version: a1b2c3d4e5f6...
source: file
loaded_at: 2025-11-21T10:00:00Z
config_file_path: /etc/config.yaml
config:
  Server:
    port: 8080
    host: localhost
  Database:
    host: localhost
    port: 5432
    password: "***REDACTED***"
```

---

## ⚠️ Error Handling

### Error Response Format

```json
{
  "status": "error",
  "error": "error message here"
}
```

### HTTP Status Codes

| Code | Description | Example |
|------|-------------|---------|
| 200 | Success | Configuration exported |
| 400 | Bad Request | Invalid format parameter |
| 403 | Forbidden | Unauthorized unsanitized access |
| 405 | Method Not Allowed | POST instead of GET |
| 500 | Internal Server Error | Serialization failure |

### Common Errors

#### Invalid Format

**Request**:
```bash
curl "http://localhost:8080/api/v2/config?format=xml"
```

**Response** (400 Bad Request):
```json
{
  "status": "error",
  "error": "invalid format: xml (supported: json, yaml)"
}
```

#### Method Not Allowed

**Request**:
```bash
curl -X POST http://localhost:8080/api/v2/config
```

**Response** (405 Method Not Allowed):
```json
{
  "status": "error",
  "error": "method not allowed"
}
```

#### Unauthorized Unsanitized Access

**Request** (non-admin):
```bash
curl "http://localhost:8080/api/v2/config?sanitize=false"
```

**Response** (403 Forbidden):
```json
{
  "status": "error",
  "error": "unauthorized: unsanitized config requires admin access"
}
```

---

## 🔐 Security

### Secret Sanitization

По умолчанию все секреты автоматически санитизируются:

| Field | Sanitized Value |
|-------|----------------|
| `database.password` | `***REDACTED***` |
| `redis.password` | `***REDACTED***` |
| `llm.api_key` | `***REDACTED***` |
| `webhook.authentication.api_key` | `***REDACTED***` |
| `webhook.authentication.jwt_secret` | `***REDACTED***` |
| `webhook.signature.secret` | `***REDACTED***` |

### Authorization

- **Public**: Sanitized config (default)
- **Admin**: Unsanitized config (`?sanitize=false`)
- **Rate Limiting**: 100 req/min per IP

### Best Practices

1. ✅ Always use sanitized config in logs/monitoring
2. ✅ Use unsanitized config only for debugging (admin only)
3. ✅ Monitor access to unsanitized config endpoint
4. ✅ Use section filtering to reduce payload size
5. ✅ Cache responses client-side (ETag support planned)

---

## 💡 Examples

### Example 1: Export Full Config (JSON)

```bash
curl http://localhost:8080/api/v2/config | jq
```

### Example 2: Export Server Config Only (YAML)

```bash
curl "http://localhost:8080/api/v2/config?sections=server&format=yaml"
```

### Example 3: Check Config Version

```bash
VERSION=$(curl -s http://localhost:8080/api/v2/config | jq -r '.data.version')
echo "Config version: $VERSION"
```

### Example 4: Export Database Config (for debugging)

```bash
curl "http://localhost:8080/api/v2/config?sections=database" | jq '.data.config.Database'
```

### Example 5: Compare Config Versions

```bash
# Get current version
CURRENT=$(curl -s http://localhost:8080/api/v2/config | jq -r '.data.version')

# Get version from another instance
REMOTE=$(curl -s http://other-host:8080/api/v2/config | jq -r '.data.version')

if [ "$CURRENT" != "$REMOTE" ]; then
  echo "Config versions differ!"
fi
```

### Example 6: Export Multiple Sections

```bash
curl "http://localhost:8080/api/v2/config?sections=server,database,redis&format=yaml" > config.yaml
```

---

## 🔍 Troubleshooting

### Problem: Empty Response

**Symptoms**: Response is empty or null

**Solutions**:
1. Check if service is running: `curl http://localhost:8080/healthz`
2. Check logs for errors
3. Verify config was loaded successfully

### Problem: Secrets Not Sanitized

**Symptoms**: Passwords visible in response

**Solutions**:
1. Check if `sanitize=false` was used (admin only)
2. Verify sanitizer is working: check logs
3. Ensure default is `sanitize=true`

### Problem: Invalid Format Error

**Symptoms**: `400 Bad Request` with "invalid format" message

**Solutions**:
1. Use only `json` or `yaml` for format parameter
2. Check query parameter spelling: `format`, not `form` or `fmt`

### Problem: Section Not Found

**Symptoms**: Requested section missing from response

**Solutions**:
1. Check section name spelling (lowercase: `server`, not `Server`)
2. Verify section exists in config
3. Check logs for warnings about unknown sections

### Problem: Slow Response

**Symptoms**: Response takes > 10ms

**Solutions**:
1. Check cache hit rate (should be high after first request)
2. Reduce number of sections requested
3. Use JSON instead of YAML (slightly faster)
4. Check system load

---

## 📊 Monitoring

### Prometheus Metrics

Monitor these metrics for health:

```promql
# Request rate
rate(alert_history_api_config_export_requests_total[5m])

# Error rate
rate(alert_history_api_config_export_errors_total[5m])

# p95 latency
histogram_quantile(0.95, alert_history_api_config_export_duration_seconds_bucket)

# Response size
rate(alert_history_api_config_export_size_bytes_sum[5m]) / rate(alert_history_api_config_export_size_bytes_count[5m])
```

### Alerting Rules

```yaml
# High error rate
- alert: ConfigExportHighErrorRate
  expr: rate(alert_history_api_config_export_errors_total[5m]) > 0.1
  for: 5m

# Slow responses
- alert: ConfigExportSlowResponse
  expr: histogram_quantile(0.95, alert_history_api_config_export_duration_seconds_bucket) > 0.01
  for: 5m
```

---

## 🔗 Related Documentation

- [README](./README.md) - Overview and quick reference
- [Requirements](./requirements.md) - Detailed requirements
- [Design](./design.md) - Technical architecture
- [Tasks](./tasks.md) - Implementation details

---

**Last Updated**: 2025-11-21
**Version**: 1.0
