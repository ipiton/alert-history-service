# TN-152: Hot Reload (SIGHUP) - User Guide

**Date**: 2025-11-22
**Task ID**: TN-152
**Version**: 1.0

---

## 📖 Overview

Hot Reload позволяет обновлять конфигурацию Alert History сервиса без перезапуска и без downtime. Это критически важная функциональность для production-окружений.

**Возможности**:
- ✅ Zero-downtime обновление конфигурации
- ✅ Автоматическая валидация перед применением
- ✅ Rollback при ошибках
- ✅ Мониторинг через Prometheus metrics
- ✅ Совместимость с Alertmanager

---

## 🚀 Быстрый старт

### 1. Обновите конфигурацию

Отредактируйте `config.yaml`:

```bash
vi /etc/alert-history/config.yaml
```

### 2. Отправьте SIGHUP signal

```bash
# Найдите PID процесса
pid=$(pidof alert-history)

# Отправьте SIGHUP
kill -HUP $pid
```

Или через `pkill`:

```bash
pkill -HUP alert-history
```

### 3. Проверьте результат

```bash
# Проверьте логи
tail -f /var/log/alert-history/app.log | grep reload

# Проверьте статус через API
curl http://localhost:8080/api/v2/config/status
```

---

## 📋 Детальное руководство

### Поддерживаемые изменения

Hot reload поддерживает обновление следующих секций:

| Секция | Reload Support | Критичность | Notes |
|--------|----------------|-------------|-------|
| `server` | ⚠️ Partial | Critical | Port требует рестарта |
| `database` | ✅ Yes | Critical | Connection pool пересоздается |
| `redis` | ✅ Yes | Non-Critical | Новые подключения |
| `llm` | ✅ Yes | Non-Critical | API key, model обновляются |
| `route` | ✅ Yes | Critical | Routing tree перестраивается |
| `receivers` | ✅ Yes | Critical | Publishers пересоздаются |
| `inhibit_rules` | ✅ Yes | Non-Critical | Rules обновляются |
| `grouping` | ✅ Yes | Critical | Timers перезапускаются |

**Изменения, требующие restart**:
- `server.port` - изменение порта
- `server.tls.*` - TLS конфигурация
- `metrics.path` - путь к /metrics

### Процесс Hot Reload

```
┌─────────────────────────────────────────────────────────────┐
│                   1. LOAD & PARSE                           │
│  • Чтение config.yaml                                       │
│  • Парсинг YAML/JSON                                        │
│  • Calculation SHA256 hash                                  │
│  Target: < 50ms                                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   2. VALIDATION                             │
│  • Синтаксическая валидация                                 │
│  • Бизнес-правила                                           │
│  • Cross-field проверки                                     │
│  • Проверка ссылок (receivers exist)                        │
│  ❌ If validation fails → ABORT                             │
│  Target: < 100ms                                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   3. DIFF CALCULATION                       │
│  • Сравнение old vs new                                     │
│  • Определение affected components                          │
│  • Если нет изменений → SKIP reload (no-op)                │
│  Target: < 20ms                                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   4. ATOMIC APPLY                           │
│  • Distributed lock (Redis)                                 │
│  • Backup old config                                        │
│  • Atomic swap (pointer replacement)                        │
│  • Version increment                                        │
│  • Audit log                                                │
│  Target: < 50ms                                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  5. COMPONENT RELOAD                        │
│  • Parallel reload affected components                      │
│  • 30s timeout per component                                │
│  ⚠️ Critical component failure → ROLLBACK                  │
│  Target: < 300ms                                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   6. HEALTH CHECK                           │
│  • Verify critical services                                 │
│  • Check database connectivity                              │
│  ⚠️ Health check failed → ROLLBACK                         │
│  Target: < 50ms                                             │
└─────────────────────────────────────────────────────────────┘

Total Target: < 500ms p95
```

---

## ✅ Примеры использования

### Пример 1: Добавление нового Slack receiver

**Сценарий**: Добавить новый канал для критических алертов

**Шаги**:

1. Добавьте receiver в `config.yaml`:
   ```yaml
   receivers:
     - name: 'critical-slack'
       slack_configs:
         - api_url: 'https://hooks.slack.com/services/XXX/YYY/ZZZ'
           channel: '#critical-alerts'
           title: '{{ .GroupLabels.alertname }}'
           text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
   ```

2. Обновите route для использования нового receiver:
   ```yaml
   route:
     routes:
       - match:
           severity: critical
         receiver: critical-slack
   ```

3. Отправьте SIGHUP:
   ```bash
   kill -HUP $(pidof alert-history)
   ```

4. Проверьте логи:
   ```bash
   tail -f /var/log/alert-history/app.log
   ```

   Ожидаемый output:
   ```json
   {
     "level": "info",
     "msg": "SIGHUP received, triggering config reload",
     "signal": "SIGHUP",
     "config_path": "/etc/alert-history/config.yaml"
   }
   {
     "level": "info",
     "msg": "config reload successful",
     "version": 43,
     "components_reloaded": ["routing", "receivers"],
     "duration_ms": 287
   }
   ```

### Пример 2: Обновление LLM API Key

**Сценарий**: Ротация OpenAI API key

**Шаги**:

1. Обновите `config.yaml`:
   ```yaml
   llm:
     enabled: true
     provider: openai
     api_key: "sk-new-key-here"  # ⚠️ Используйте secrets в production!
   ```

2. Отправьте SIGHUP:
   ```bash
   pkill -HUP alert-history
   ```

3. Проверьте через API:
   ```bash
   curl http://localhost:8080/api/v2/config/status
   ```

   Response:
   ```json
   {
     "version": 44,
     "status": "success",
     "last_reload": "2025-11-22T10:15:30Z",
     "last_reload_unix": 1700000000
   }
   ```

### Пример 3: Изменение database connection pool

**Сценарий**: Увеличение max_connections для production

**Шаги**:

1. Обновите `config.yaml`:
   ```yaml
   database:
     host: postgres.local
     port: 5432
     max_connections: 100  # Было: 50
     min_connections: 10
   ```

2. Отправьте SIGHUP:
   ```bash
   kill -HUP $(pidof alert-history)
   ```

3. Мониторинг через Prometheus:
   ```promql
   # Успешность reload
   config_reload_total{status="success"}

   # Длительность reload
   config_reload_duration_seconds

   # Версия конфигурации
   config_reload_version
   ```

---

## 🔍 Мониторинг и диагностика

### Prometheus Metrics

```promql
# Total reload attempts
config_reload_total{status="success"}
config_reload_total{status="validation_error"}
config_reload_total{status="error"}
config_reload_total{status="rolled_back"}

# Reload duration (p95)
histogram_quantile(0.95, config_reload_duration_seconds)

# Reload errors by type
config_reload_errors_total{type="load_failed"}
config_reload_errors_total{type="validation_failed"}
config_reload_errors_total{type="component_failed"}

# Last successful reload (timestamp)
config_reload_last_success_timestamp_seconds

# Current config version
config_reload_version

# Rollback count
config_reload_rollbacks_total{reason="critical_failed"}
config_reload_rollbacks_total{reason="health_check_failed"}
```

### Grafana Dashboard Query Examples

**Success Rate (last 24h)**:
```promql
sum(rate(config_reload_total{status="success"}[24h]))
/
sum(rate(config_reload_total[24h]))
* 100
```

**P95 Reload Duration**:
```promql
histogram_quantile(0.95,
  rate(config_reload_duration_seconds_bucket[5m])
)
```

**Failed Reloads (last 1h)**:
```promql
sum(increase(config_reload_total{status!="success"}[1h]))
```

### Status API Endpoint

```bash
curl http://localhost:8080/api/v2/config/status | jq
```

Response:
```json
{
  "version": 43,
  "status": "success",
  "last_reload": "2025-11-22T10:15:30Z",
  "last_reload_unix": 1700000000
}
```

**Status Values**:
- `initial` - Изначальная конфигурация при запуске
- `success` - Последний reload успешен
- `load_failed` - Ошибка чтения/парсинга файла
- `validation_failed` - Ошибка валидации
- `apply_failed` - Ошибка применения
- `rolled_back` - Произошел rollback

---

## ⚠️ Обработка ошибок

### Validation Errors

**Симптомы**:
- Reload не применился
- Логи содержат "validation failed"
- Old config продолжает работать

**Пример**:
```json
{
  "level": "error",
  "msg": "config reload failed",
  "error": "validation failed: 1 error(s)"
}
{
  "level": "error",
  "msg": "validation error",
  "field": "route.receiver",
  "message": "receiver 'unknown-receiver' not found",
  "code": "E102"
}
```

**Решение**:
1. Проверьте синтаксис YAML
2. Проверьте бизнес-правила (receivers exist, ports valid, etc.)
3. Исправьте ошибки
4. Повторите SIGHUP

### Component Reload Failure

**Симптомы**:
- Reload failed с rollback
- Логи содержат "critical component failed"
- Config откачен к предыдущей версии

**Пример**:
```json
{
  "level": "error",
  "msg": "critical component reload failed",
  "component": "database",
  "error": "failed to resize connection pool: timeout"
}
{
  "level": "warn",
  "msg": "rolling back due to critical component failure"
}
{
  "level": "info",
  "msg": "rollback successful",
  "version": 42
}
```

**Решение**:
1. Проверьте доступность критических сервисов (database, Redis)
2. Проверьте ресурсы (CPU, memory, connections)
3. Увеличьте таймауты если необходимо
4. Постепенно увеличивайте изменения (incremental changes)

### Concurrent Reload Attempts

**Симптомы**:
- Reload failed с "lock already held"
- Другой reload уже выполняется

**Пример**:
```json
{
  "level": "error",
  "msg": "config reload failed",
  "error": "phase 4 (apply) failed: failed to acquire lock: concurrent update in progress"
}
```

**Решение**:
1. Подождите завершения текущего reload (max 30s)
2. Повторите SIGHUP

---

## 🔐 Best Practices

### 1. Testing Before Production

**Всегда тестируйте в staging**:
```bash
# В staging environment
kill -HUP $(pidof alert-history)

# Проверьте метрики
curl http://staging:8080/metrics | grep config_reload

# Если успешно, применяйте в production
```

### 2. Incremental Changes

**Делайте небольшие изменения**:
- ✅ Добавляйте по одному receiver за раз
- ✅ Обновляйте routes постепенно
- ❌ Избегайте массовых изменений всех секций

### 3. Backup Configuration

**Создавайте backup перед изменениями**:
```bash
cp /etc/alert-history/config.yaml /etc/alert-history/config.yaml.backup.$(date +%Y%m%d_%H%M%S)
```

### 4. Use Secrets Management

**Не храните secrets в config.yaml**:

❌ **Bad**:
```yaml
llm:
  api_key: "sk-1234567890"  # Hardcoded secret!
```

✅ **Good**:
```yaml
llm:
  api_key_file: "/secrets/openai-key"  # Read from file
```

Или используйте environment variables:
```yaml
llm:
  api_key: "${OPENAI_API_KEY}"
```

### 5. Monitor Reload Operations

**Настройте alerting для failed reloads**:
```yaml
# prometheus-alerts.yaml
groups:
  - name: config-reload
    rules:
      - alert: ConfigReloadFailed
        expr: increase(config_reload_total{status!="success"}[5m]) > 0
        for: 1m
        annotations:
          summary: "Config reload failed"
          description: "Alert History config reload failed: {{ $labels.status }}"
```

### 6. Document Changes

**Ведите changelog**:
```bash
# В /etc/alert-history/CHANGELOG.md
## 2025-11-22
- Added critical-slack receiver
- Increased database.max_connections to 100
- Reload successful: version 43 → 44
```

---

## 🚨 Troubleshooting

### Issue 1: SIGHUP не срабатывает

**Проверка**:
```bash
# Проверьте что процесс запущен
ps aux | grep alert-history

# Проверьте права на отправку сигналов
kill -0 $(pidof alert-history)

# Проверьте логи
tail -f /var/log/alert-history/app.log | grep SIGHUP
```

**Возможные причины**:
- Процесс запущен под другим пользователем
- Недостаточно прав для отправки сигналов
- Signal handler не зарегистрирован (bug)

### Issue 2: Reload занимает > 500ms

**Диагностика**:
```bash
# Проверьте per-phase duration
curl http://localhost:8080/metrics | grep config_reload_phase_duration
```

**Возможные причины**:
- Медленная валидация (большой config)
- Медленный reload компонентов
- Проблемы с network (database, Redis)

**Решение**:
- Оптимизируйте config (уберите неиспользуемые receivers)
- Увеличьте ресурсы (CPU, memory)
- Проверьте network latency

### Issue 3: Rollback после успешной валидации

**Проверка**:
```bash
# Найдите причину в логах
tail -f /var/log/alert-history/app.log | grep -A 10 rollback
```

**Возможные причины**:
- Critical component не смог reload (database, routing)
- Health check failed после reload
- Timeout при reload компонента (> 30s)

---

## 📚 Additional Resources

- [Design Document](design.md) - Техническая архитектура
- [Tasks](tasks.md) - Детальный план задачи
- [Kubernetes Guide](KUBERNETES.md) - Интеграция с K8s
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Детальная диагностика

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
