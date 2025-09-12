# 🚀 Быстрый старт - Alert History Service

## 📋 Параметры для тестирования режимов

### 🔧 Основные параметры

| Параметр | Описание | Значения | По умолчанию |
|----------|----------|----------|--------------|
| `mode` | Режим обогащения | `transparent`, `transparent_with_recommendations`, `enriched` | `transparent` |
| `receiver` | Получатель webhook | Любая строка | - |
| `status` | Статус алерта | `firing`, `resolved` | `firing` |
| `fingerprint` | Уникальный ID алерта | Строка | - |
| `alertname` | Название алерта | Строка | - |
| `instance` | Инстанс | Строка | - |
| `severity` | Важность | `critical`, `warning`, `info` | - |

### 🎯 Примеры тестирования

#### 1. Transparent режим
```bash
# Установить режим
curl -X POST http://localhost:8000/enrichment/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "transparent"}'

# Отправить тестовый алерт
curl -X POST http://localhost:8000/webhook/ \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "test",
    "status": "firing",
    "alerts": [{
      "fingerprint": "test-1",
      "status": "firing",
      "labels": {
        "alertname": "HighCPUUsage",
        "instance": "web-server-1",
        "severity": "warning"
      },
      "annotations": {
        "description": "CPU usage is high",
        "summary": "High CPU usage detected"
      },
      "startsAt": "2024-01-01T10:00:00Z",
      "endsAt": "2024-01-01T10:05:00Z"
    }]
  }'
```

#### 2. Transparent with Recommendations режим
```bash
# Установить режим
curl -X POST http://localhost:8000/enrichment/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "transparent_with_recommendations"}'

# Отправить тестовый алерт (тот же формат)
curl -X POST http://localhost:8000/webhook/ \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "test",
    "status": "firing",
    "alerts": [{
      "fingerprint": "test-2",
      "status": "firing",
      "labels": {
        "alertname": "DiskSpaceLow",
        "instance": "db-server-1",
        "severity": "critical"
      },
      "annotations": {
        "description": "Disk space is running low",
        "summary": "Critical disk space issue"
      },
      "startsAt": "2024-01-01T10:00:00Z",
      "endsAt": "2024-01-01T10:05:00Z"
    }]
  }'
```

#### 3. Enriched режим
```bash
# Установить режим
curl -X POST http://localhost:8000/enrichment/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "enriched"}'

# Отправить тестовый алерт (тот же формат)
curl -X POST http://localhost:8000/webhook/ \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "test",
    "status": "firing",
    "alerts": [{
      "fingerprint": "test-3",
      "status": "firing",
      "labels": {
        "alertname": "HighMemoryUsage",
        "instance": "app-server-1",
        "severity": "info"
      },
      "annotations": {
        "description": "Memory usage is elevated",
        "summary": "Memory usage monitoring"
      },
      "startsAt": "2024-01-01T10:00:00Z",
      "endsAt": "2024-01-01T10:05:00Z"
    }]
  }'
```

### 📊 Проверка результатов

#### Проверить текущий режим
```bash
curl http://localhost:8000/enrichment/mode
```

#### Проверить статистику
```bash
curl http://localhost:8000/classification/stats
```

#### Проверить метрики
```bash
curl http://localhost:8000/metrics | grep enrichment
```

### 🎯 Ожидаемые результаты

#### Transparent режим
```json
{
  "message": "Webhook processed successfully (legacy mode)",
  "processed_alerts": 1,
  "published_alerts": 0,
  "filtered_alerts": 0,
  "mode": "legacy"
}
```

#### Transparent with Recommendations режим
```json
{
  "message": "Webhook processed successfully (legacy mode)",
  "processed_alerts": 1,
  "published_alerts": 0,
  "filtered_alerts": 0,
  "mode": "legacy",
  "classification_results": {
    "test-2": {
      "severity": "critical",
      "confidence": 0.95,
      "reasoning": "Disk space issue requires immediate attention",
      "recommendations": [
        "Increase disk space monitoring frequency",
        "Add automated cleanup procedures"
      ]
    }
  }
}
```

#### Enriched режим
```json
{
  "message": "Webhook processed successfully (intelligent mode)",
  "processed_alerts": 1,
  "published_alerts": 1,
  "filtered_alerts": 0,
  "mode": "intelligent",
  "classification_results": {
    "test-3": {
      "severity": "info",
      "confidence": 0.8,
      "reasoning": "Memory usage is within normal range",
      "recommendations": [
        "Consider increasing memory threshold"
      ]
    }
  }
}
```

### 🔄 Быстрое переключение режимов

```bash
#!/bin/bash

# Функция для быстрого переключения
switch_mode() {
    echo "Переключаемся на режим: $1"
    curl -X POST http://localhost:8000/enrichment/mode \
      -H "Content-Type: application/json" \
      -d "{\"mode\": \"$1\"}"
    echo ""
}

# Тестируем все режимы
switch_mode "transparent"
switch_mode "transparent_with_recommendations"
switch_mode "enriched"

# Проверяем текущий режим
echo "Текущий режим:"
curl http://localhost:8000/enrichment/mode
```

### 🎛️ Dashboard

Откройте dashboard для визуального управления:
```bash
open http://localhost:8000/dashboard
```

В dashboard вы можете:
- Переключать режимы кнопками
- Видеть статистику в реальном времени
- Мониторить метрики
- Управлять настройками

### 🚨 Важные замечания

1. **LLM недоступен** - система работает в legacy mode
2. **Все алерты проходят** - нет фильтрации без LLM
3. **Метрики сохраняются** - ваши dashboard'ы работают
4. **Безопасное тестирование** - можно переключаться между режимами

### 📝 Следующие шаги

1. Протестируйте все три режима
2. Изучите dashboard
3. Настройте LLM для полной функциональности
4. Переходите к продакшен использованию
