# Alert History Service - API Documentation

Полная документация REST API для Alert History Service с примерами запросов и ответов.

## 📋 Base Information

- **Base URL**: `https://your-domain.com` или `http://localhost:8080`
- **API Version**: v1
- **Content-Type**: `application/json`
- **Authentication**: В development — без аутентификации, в production — рекомендуется mTLS/OIDC

---

## 🏥 Health & Status Endpoints

### GET /healthz
Проверка базового состояния сервиса.

**Response**: `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2024-12-28T10:30:00Z"
}
```

### GET /readyz
Проверка готовности сервиса к обработке запросов.

**Response**: `200 OK`
```json
{
  "status": "ready",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "llm_service": "unavailable"
  },
  "timestamp": "2024-12-28T10:30:00Z"
}
```

### GET /metrics
Prometheus метрики в формате exposition.

**Response**: `200 OK`
```
# HELP alert_history_webhook_events_total Total webhook events received
# TYPE alert_history_webhook_events_total counter
alert_history_webhook_events_total{alertname="CPUThrottlingHigh",status="firing"} 42
```

---

## 📨 Webhook Endpoints

### POST /webhook
Legacy webhook endpoint для backward compatibility.

**Request Body**:
```json
{
  "receiver": "alert-history",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "CPUThrottlingHigh",
        "namespace": "production",
        "severity": "warning"
      },
      "annotations": {
        "summary": "High CPU throttling detected",
        "description": "CPU throttling is above 50%"
      },
      "startsAt": "2024-12-28T10:15:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?g0.expr=..."
    }
  ]
}
```

**Response**: `200 OK`
```json
{
  "status": "ok",
  "processed_alerts": 1,
  "timestamp": "2024-12-28T10:30:00Z"
}
```

### POST /webhook/proxy
Intelligent proxy endpoint с LLM классификацией и автоматической публикацией.

**Request Body**: Аналогично `/webhook`

**Response**: `200 OK`
```json
{
  "status": "success",
  "processing_summary": {
    "total_alerts": 1,
    "published_alerts": 1,
    "filtered_alerts": 0,
    "enrichment_mode": "enriched"
  },
  "classification_results": {
    "CPUThrottlingHigh": {
      "severity": "warning",
      "confidence": 0.85,
      "category": "performance",
      "model": "gpt-4"
    }
  },
  "publishing_results": {
    "rootly": {
      "status": "success",
      "incident_id": "INC-12345"
    },
    "slack": {
      "status": "success",
      "message_ts": "1640688600.123"
    }
  },
  "metrics_only_mode": false,
  "timestamp": "2024-12-28T10:30:00Z"
}
```

---

## 📊 History & Analytics Endpoints

### GET /history
Получение истории алертов с фильтрацией.

**Query Parameters**:
- `alertname` (string) — фильтр по имени алерта
- `namespace` (string) — фильтр по namespace
- `status` (string) — фильтр по статусу: `firing`, `resolved`
- `fingerprint` (string) — фильтр по fingerprint
- `since` (ISO 8601) — начальная дата
- `until` (ISO 8601) — конечная дата
- `limit` (int) — максимальное количество записей
- `offset` (int) — смещение для пагинации

**Example Request**:
```bash
GET /history?alertname=CPUThrottlingHigh&namespace=production&since=2024-12-28T00:00:00Z&limit=10
```

**Response**: `200 OK`
```json
{
  "alerts": [
    {
      "id": 12345,
      "alertname": "CPUThrottlingHigh",
      "namespace": "production",
      "status": "firing",
      "severity": "warning",
      "fingerprint": "abc123def456",
      "labels": {
        "alertname": "CPUThrottlingHigh",
        "namespace": "production",
        "pod": "web-server-1"
      },
      "annotations": {
        "summary": "High CPU throttling detected",
        "description": "CPU throttling is above 50%"
      },
      "starts_at": "2024-12-28T10:15:00Z",
      "ends_at": null,
      "timestamp": "2024-12-28T10:15:05Z",
      "classification": {
        "severity": "warning",
        "confidence": 0.85,
        "category": "performance",
        "model": "gpt-4",
        "classified_at": "2024-12-28T10:15:06Z"
      }
    }
  ],
  "total": 25,
  "limit": 10,
  "offset": 0
}
```

### GET /report
Получение аналитических отчетов.

**Query Parameters**:
- `top` (int) — количество топ алертов (default: 10)
- `min_flap` (int) — минимальное количество flapping событий
- `namespace` (string) — фильтр по namespace
- `since` (ISO 8601) — начальная дата
- `until` (ISO 8601) — конечная дата

**Example Request**:
```bash
GET /report?top=5&min_flap=3&since=2024-12-27T00:00:00Z
```

**Response**: `200 OK`
```json
{
  "summary": {
    "total_alerts": 1250,
    "unique_alerts": 45,
    "flapping_alerts": 8,
    "avg_duration_minutes": 15.5,
    "period": {
      "since": "2024-12-27T00:00:00Z",
      "until": "2024-12-28T10:30:00Z"
    }
  },
  "top_alerts": [
    {
      "alertname": "CPUThrottlingHigh",
      "namespace": "production",
      "event_count": 156,
      "avg_confidence": 0.87,
      "last_seen": "2024-12-28T10:20:00Z"
    }
  ],
  "flapping_alerts": [
    {
      "alertname": "DiskSpaceWarning",
      "namespace": "staging",
      "flap_count": 12,
      "frequency_minutes": 8.5,
      "recommendation": "Increase disk cleanup threshold"
    }
  ]
}
```

---

## 🎯 Publishing Endpoints

### GET /publishing/targets
Получение списка discovered publishing targets.

**Response**: `200 OK`
```json
{
  "targets": [
    {
      "name": "rootly-config",
      "namespace": "alert-targets",
      "format": "rootly",
      "active": true,
      "last_discovered": "2024-12-28T10:25:00Z",
      "config": {
        "url": "https://api.rootly.com",
        "organization_id": "org-123"
      }
    },
    {
      "name": "slack-webhook",
      "namespace": "alert-targets",
      "format": "slack",
      "active": true,
      "last_discovered": "2024-12-28T10:25:00Z"
    }
  ],
  "total_targets": 2,
  "last_discovery": "2024-12-28T10:25:00Z"
}
```

### POST /publishing/targets/refresh
Принудительное обновление списка publishing targets.

**Response**: `200 OK`
```json
{
  "status": "success",
  "discovered_targets": 2,
  "new_targets": 0,
  "removed_targets": 1,
  "discovery_duration_ms": 150,
  "timestamp": "2024-12-28T10:30:00Z"
}
```

### GET /publishing/mode
Получение текущего режима publishing.

**Response**: `200 OK`
```json
{
  "mode": "normal",
  "metrics_only": false,
  "active_targets": 2,
  "reason": "targets_available"
}
```

### GET /publishing/stats
Статистика публикации алертов.

**Response**: `200 OK`
```json
{
  "stats": {
    "total_published": 1250,
    "successful_published": 1205,
    "failed_published": 45,
    "success_rate": 0.964,
    "last_24h": {
      "published": 156,
      "success_rate": 0.987
    }
  },
  "by_target": {
    "rootly": {
      "published": 850,
      "success_rate": 0.975,
      "avg_latency_ms": 245
    },
    "slack": {
      "published": 400,
      "success_rate": 0.995,
      "avg_latency_ms": 120
    }
  }
}
```

---

## 🧠 Classification Endpoints

### GET /classification/stats
Статистика LLM классификации.

**Response**: `200 OK`
```json
{
  "stats": {
    "total_classified": 1180,
    "classification_rate": 0.944,
    "avg_confidence": 0.83,
    "avg_latency_ms": 850,
    "cache_hit_rate": 0.65
  },
  "by_severity": {
    "critical": {"count": 85, "avg_confidence": 0.91},
    "warning": {"count": 650, "avg_confidence": 0.84},
    "info": {"count": 380, "avg_confidence": 0.78},
    "noise": {"count": 65, "avg_confidence": 0.88}
  },
  "model_stats": {
    "gpt-4": {"requests": 1180, "avg_latency_ms": 850},
    "cache": {"hits": 767, "misses": 413}
  }
}
```

### POST /classification/classify
Ручная классификация алерта.

**Request Body**:
```json
{
  "alert": {
    "alertname": "CustomAlert",
    "labels": {
      "severity": "warning",
      "namespace": "production"
    },
    "annotations": {
      "summary": "Custom alert for testing"
    }
  },
  "force": false
}
```

**Response**: `200 OK`
```json
{
  "classification": {
    "severity": "warning",
    "confidence": 0.82,
    "category": "custom",
    "reasoning": "Alert indicates a warning condition in production environment",
    "model": "gpt-4",
    "cached": false,
    "processing_time_ms": 920
  }
}
```

### GET /classification/models
Список доступных LLM моделей.

**Response**: `200 OK`
```json
{
  "models": [
    {
      "name": "gpt-4",
      "status": "available",
      "latency_p95_ms": 1200,
      "success_rate": 0.995
    },
    {
      "name": "gpt-3.5-turbo",
      "status": "available",
      "latency_p95_ms": 650,
      "success_rate": 0.987
    }
  ],
  "default_model": "gpt-4"
}
```

---

## 🔧 Enrichment Mode Endpoints

### GET /enrichment/mode
Получение текущего режима обогащения.

**Response**: `200 OK`
```json
{
  "mode": "enriched",
  "source": "redis"
}
```

Возможные значения:
- `mode`: `"transparent"` | `"enriched"`
- `source`: `"redis"` | `"memory"` | `"default"`

### POST /enrichment/mode
Изменение режима обогащения.

**Request Body**:
```json
{
  "mode": "transparent"
}
```

**Response**: `200 OK`
```json
{
  "mode": "transparent",
  "source": "redis"
}
```

---

## 🎛️ Dashboard Endpoints

### GET /dashboard/modern
HTML5 дашборд для визуализации данных.

**Response**: `200 OK` (HTML page)

### GET /api/dashboard/overview
Данные для overview дашборда.

**Response**: `200 OK`
```json
{
  "total_alerts": 1250,
  "active_alerts": 15,
  "resolved_alerts": 1235,
  "alerts_last_24h": 156,
  "classification_enabled": true,
  "classified_alerts": 1180,
  "classification_cache_hit_rate": 0.65,
  "publishing_targets": 2,
  "publishing_mode": "normal",
  "successful_publishes": 1205,
  "failed_publishes": 45,
  "system_healthy": true,
  "redis_connected": true,
  "llm_service_available": true,
  "last_updated": "2024-12-28T10:30:00Z"
}
```

### GET /api/dashboard/charts
Данные для графиков dashboard.

**Query Parameters**:
- `hours` (int) — количество часов для отображения (default: 24)

**Response**: `200 OK`
```json
{
  "time_series": [
    {
      "timestamp": "2024-12-28T09:00:00Z",
      "alerts_received": 12,
      "alerts_classified": 11,
      "alerts_published": 10
    }
  ],
  "severity_distribution": {
    "critical": 5,
    "warning": 45,
    "info": 25,
    "noise": 8
  },
  "confidence_distribution": {
    "high": 65,
    "medium": 25,
    "low": 10
  }
}
```

### GET /api/dashboard/health
Данные о здоровье системы.

**Response**: `200 OK`
```json
{
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 15,
      "connection_pool": "8/20"
    },
    "redis": {
      "status": "healthy",
      "latency_ms": 2,
      "memory_usage": "45MB"
    },
    "llm_service": {
      "status": "available",
      "latency_ms": 850,
      "requests_per_minute": 5.2
    }
  },
  "metrics": {
    "cpu_usage": 0.25,
    "memory_usage": 0.40,
    "request_rate": 12.5,
    "error_rate": 0.02
  }
}
```

### GET /api/dashboard/alerts/recent
Последние алерты для дашборда.

**Query Parameters**:
- `limit` (int) — количество записей (default: 20)
- `min_confidence` (float) — минимальная confidence (0.0-1.0)

**Response**: `200 OK`
```json
{
  "alerts": [
    {
      "alertname": "CPUThrottlingHigh",
      "namespace": "production",
      "status": "firing",
      "severity": "warning",
      "confidence": 0.85,
      "timestamp": "2024-12-28T10:25:00Z",
      "published_to": ["rootly", "slack"]
    }
  ],
  "total": 156
}
```

### GET /api/dashboard/recommendations
Рекомендации для дашборда.

**Response**: `200 OK`
```json
{
  "recommendations": [
    {
      "type": "threshold_adjustment",
      "alert": "DiskSpaceWarning",
      "namespace": "staging",
      "description": "Consider increasing disk cleanup threshold",
      "confidence": 0.78,
      "impact": "medium",
      "suggested_action": "Update threshold from 80% to 85%"
    },
    {
      "type": "flapping_reduction",
      "alert": "PodCrashLoopBackOff",
      "namespace": "production",
      "description": "Alert is flapping every 5 minutes",
      "confidence": 0.92,
      "impact": "high",
      "suggested_action": "Increase evaluation_interval to 10m"
    }
  ],
  "total": 5
}
```

---

## 🚨 Error Responses

### Standard Error Format
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid enrichment mode specified",
    "details": {
      "field": "mode",
      "allowed_values": ["transparent", "enriched"]
    }
  },
  "timestamp": "2024-12-28T10:30:00Z"
}
```

### Common Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `INVALID_REQUEST` | 400 | Неверный формат запроса |
| `NOT_FOUND` | 404 | Ресурс не найден |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка сервера |
| `SERVICE_UNAVAILABLE` | 503 | Сервис временно недоступен |
| `LLM_UNAVAILABLE` | 503 | LLM сервис недоступен |
| `DATABASE_ERROR` | 503 | Ошибка базы данных |

---

## 📝 Rate Limits

- **General API**: 1000 requests/minute per IP
- **Webhook endpoints**: 500 requests/minute per IP
- **Classification endpoints**: 100 requests/minute per IP
- **Dashboard API**: 200 requests/minute per IP

Превышение лимитов возвращает `429 Too Many Requests`.

---

## 🔗 OpenAPI Specification

Полная OpenAPI спецификация доступна:
- **JSON**: `GET /openapi.json`
- **Interactive docs**: `GET /docs` (Swagger UI)
- **Alternative docs**: `GET /redoc` (ReDoc)

---

## 🧪 Testing Examples

### Using curl

```bash
# Test webhook
curl -X POST http://localhost:8080/webhook/proxy \
  -H "Content-Type: application/json" \
  -d @test-alert.json

# Get recent alerts
curl "http://localhost:8080/history?limit=5&since=2024-12-28T00:00:00Z"

# Switch enrichment mode
curl -X POST http://localhost:8080/enrichment/mode \
  -H "Content-Type: application/json" \
  -d '{"mode":"transparent"}'

# Check publishing targets
curl http://localhost:8080/publishing/targets
```

### Using Python requests

```python
import requests
import json

# Test classification
alert_data = {
    "alert": {
        "alertname": "TestAlert",
        "labels": {"severity": "warning"},
        "annotations": {"summary": "Test alert"}
    }
}

response = requests.post(
    "http://localhost:8080/classification/classify",
    json=alert_data
)
print(response.json())
```

---

Для получения дополнительной информации обращайтесь к [основной документации](../README.md) или используйте интерактивную документацию по адресу `/docs`.
