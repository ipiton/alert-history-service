# TN-137: Аудит и унификация метрик Prometheus

**Дата создания:** 2025-10-09
**Автор:** AI Assistant
**Приоритет:** HIGH
**Статус:** NOT_STARTED

## 📋 Цель

Провести комплексный аудит существующих Prometheus метрик в Alert History Service и унифицировать их именование для обеспечения консистентности, масштабируемости и удобства работы в Grafana.

## 🎯 Обоснование задачи

### Текущие проблемы

1. **Несогласованное именование метрик**
   - Часть метрик использует namespace/subsystem паттерн (`alert_history_http_requests_total`)
   - Часть метрик не использует namespace (`alert_history_query_duration_seconds`)
   - Отсутствие единого префикса для всех метрик системы

2. **Отсутствие четкой группировки**
   - Нет явного разделения на business, technical и infrastructure метрики
   - Метрики разбросаны по разным файлам без единой структуры
   - Нет taxonomy для типов метрик

3. **Проблемы с масштабируемостью**
   - При добавлении новых компонентов неясно, как именовать метрики
   - Отсутствие guidelines для разработчиков
   - Database Pool метрики не экспортируются в Prometheus

4. **Риски для мониторинга**
   - Сложность построения дашбордов из-за непредсказуемых имен
   - Возможные конфликты имен при добавлении новых сервисов
   - Отсутствие централизованной документации по всем метрикам

### Бизнес-ценность

- ✅ **Операционная эффективность:** упрощение поиска и визуализации метрик в Grafana
- ✅ **Скорость разработки:** четкие guidelines для добавления новых метрик
- ✅ **Надежность мониторинга:** предсказуемая структура метрик снижает риск пропуска инцидентов
- ✅ **Масштабируемость:** готовность к добавлению новых компонентов (Alertmanager++, Grouping, Inhibition)

## 🔍 Текущий инвентарь метрик

### 1. HTTP метрики (`pkg/metrics/prometheus.go`)

```
alert_history_http_requests_total{method,path,status_code}
alert_history_http_request_duration_seconds{method,path,status_code}
alert_history_http_request_size_bytes{method,path}
alert_history_http_response_size_bytes{method,path,status_code}
alert_history_http_active_requests
```

✅ **Статус:** Хорошо структурированы (namespace="alert_history", subsystem="http")

### 2. Filter метрики (`pkg/metrics/filter.go`)

```
alert_history_filter_alerts_filtered_total{result}
alert_history_filter_duration_seconds{result}
alert_history_filter_blocked_alerts_total{reason}
alert_history_filter_validations_total{status}
```

✅ **Статус:** Хорошо структурированы (namespace="alert_history", subsystem="filter")

### 3. Enrichment метрики (`pkg/metrics/enrichment.go`)

```
alert_history_enrichment_mode_switches_total{from_mode,to_mode}
alert_history_enrichment_mode_status
alert_history_enrichment_mode_requests_total{method,mode}
alert_history_enrichment_redis_errors_total
```

✅ **Статус:** Хорошо структурированы (namespace="alert_history", subsystem="enrichment")

### 4. Circuit Breaker метрики (`internal/infrastructure/llm/circuit_breaker_metrics.go`)

```
alert_history_llm_circuit_breaker_state
alert_history_llm_circuit_breaker_failures_total
alert_history_llm_circuit_breaker_successes_total
alert_history_llm_circuit_breaker_state_changes_total{from,to}
alert_history_llm_circuit_breaker_requests_blocked_total
alert_history_llm_circuit_breaker_half_open_requests_total
alert_history_llm_circuit_breaker_slow_calls_total
alert_history_llm_circuit_breaker_call_duration_seconds{result}
```

⚠️ **Проблема:** Слишком длинный префикс (`llm_circuit_breaker`), можно сократить до `llm_cb`

### 5. History Repository метрики (`internal/infrastructure/repository/postgres_history.go`)

```
alert_history_query_duration_seconds{operation,status}
alert_history_query_errors_total{operation,error_type}
alert_history_query_results_total{operation}
alert_history_cache_hits_total{cache_type}
```

❌ **Проблема:** Не используют subsystem! Должны быть `alert_history_repository_*` или `alert_history_history_*`

### 6. Database Pool метрики (`internal/database/postgres/metrics.go`)

```
(Внутренние atomic метрики, не экспортируются в Prometheus)
```

❌ **Проблема:** Критические метрики БД не видны в Prometheus!

## 🎨 Требуемая схема унификации

### Принципы именования

1. **Иерархическая структура:** `<namespace>_<category>_<subsystem>_<metric_name>_<unit>`
2. **Консистентный namespace:** `alert_history` для всех метрик
3. **Категории метрик:**
   - `business` - бизнес-метрики (alerts processed, enrichments, classifications)
   - `technical` - технические метрики (HTTP, LLM calls, cache hits)
   - `infra` - инфраструктурные метрики (DB pools, Redis connections)

### Предлагаемая taxonomy

```yaml
Namespace: alert_history
├── Category: business
│   ├── subsystem: alerts
│   │   └── metrics: processed_total, enriched_total, filtered_total
│   ├── subsystem: llm
│   │   └── metrics: classifications_total, recommendations_total
│   └── subsystem: publishing
│       └── metrics: published_total, failed_total
│
├── Category: technical
│   ├── subsystem: http
│   │   └── metrics: requests_total, duration_seconds, size_bytes
│   ├── subsystem: llm_cb (circuit_breaker)
│   │   └── metrics: state, failures_total, duration_seconds
│   ├── subsystem: filter
│   │   └── metrics: alerts_filtered_total, duration_seconds
│   └── subsystem: enrichment
│       └── metrics: mode_switches_total, mode_status
│
└── Category: infra
    ├── subsystem: db
    │   └── metrics: connections_active, queries_total, duration_seconds
    ├── subsystem: cache
    │   └── metrics: hits_total, misses_total, evictions_total
    └── subsystem: repository
        └── metrics: query_duration_seconds, errors_total
```

## 📊 Mapping старых метрик на новые

### Изменения (Breaking Changes)

| Старое имя | Новое имя | Категория | Причина |
|------------|-----------|-----------|---------|
| `alert_history_query_duration_seconds` | `alert_history_infra_repository_query_duration_seconds` | infra | Добавлен subsystem |
| `alert_history_query_errors_total` | `alert_history_infra_repository_query_errors_total` | infra | Добавлен subsystem |
| `alert_history_query_results_total` | `alert_history_infra_repository_query_results_total` | infra | Добавлен subsystem |
| `alert_history_cache_hits_total` | `alert_history_infra_cache_hits_total` | infra | Добавлен category+subsystem |
| `alert_history_llm_circuit_breaker_*` | `alert_history_technical_llm_cb_*` | technical | Сокращение subsystem, добавлена категория |

### Новые метрики (Database Pool)

| Метрика | Тип | Labels | Описание |
|---------|-----|--------|----------|
| `alert_history_infra_db_connections_active` | Gauge | - | Активные соединения |
| `alert_history_infra_db_connections_idle` | Gauge | - | Idle соединения |
| `alert_history_infra_db_connections_total` | Counter | - | Всего соединений создано |
| `alert_history_infra_db_connection_wait_duration_seconds` | Histogram | - | Время ожидания соединения |
| `alert_history_infra_db_query_duration_seconds` | Histogram | - | Время выполнения запросов |
| `alert_history_infra_db_errors_total` | Counter | `error_type` | Ошибки БД |

## 🔧 Требования к реализации

### Фаза 1: Аудит (2 часа)

- [ ] Создать comprehensive inventory всех существующих метрик
- [ ] Документировать использование метрик в Grafana дашбордах
- [ ] Найти все recording rules в Prometheus
- [ ] Проверить алерты, использующие метрики

### Фаза 2: Design (3 часа)

- [ ] Разработать финальную taxonomy метрик
- [ ] Создать mapping-таблицу старые → новые имена
- [ ] Определить стратегию миграции (с поддержкой legacy или hard break)
- [ ] Написать guidelines для разработчиков

### Фаза 3: Implementation (8 часов)

- [ ] Реализовать новую схему именования метрик
- [ ] Добавить Prometheus метрики для Database Pool
- [ ] Обновить все файлы с метриками
- [ ] Создать helper functions для унифицированного создания метрик
- [ ] Добавить validation метрик при старте приложения

### Фаза 4: Migration Support (3 часа)

- [ ] Добавить поддержку legacy имен через alias/recording rules
- [ ] Создать скрипты для обновления Grafana дашбордов
- [ ] Обновить документацию по метрикам
- [ ] Создать changelog для SRE/DevOps команд

### Фаза 5: Testing & Validation (2 часа)

- [ ] Unit тесты для новых метрик
- [ ] Integration тесты для Database Pool metrics
- [ ] Проверка compatibility с существующими дашбордами
- [ ] Load testing метрик (overhead check)

### Фаза 6: Documentation (2 часа)

- [ ] Обновить `tasks/docs/prometheus-metrics.md`
- [ ] Создать `METRICS_NAMING_GUIDE.md`
- [ ] Документировать процесс добавления новых метрик
- [ ] Создать примеры PromQL запросов для новых метрик

## 📝 Критерии приемки

### Must Have

1. ✅ Все метрики следуют единой схеме именования
2. ✅ Database Pool metrics экспортируются в Prometheus
3. ✅ Существующие Grafana дашборды продолжают работать (через alias)
4. ✅ Полная документация по всем метрикам
5. ✅ Guidelines для разработчиков
6. ✅ 100% покрытие тестами новых метрик

### Should Have

1. ✅ Recording rules для backwards compatibility (30 дней)
2. ✅ Автоматическая валидация метрик при старте приложения
3. ✅ Скрипты для массового обновления дашбордов
4. ✅ Changelog для SRE команд

### Nice to Have

1. 🎯 Grafana dashboard generator из метрик кода
2. 🎯 Linter для проверки naming conventions
3. 🎯 OpenMetrics format support
4. 🎯 Metrics registry для centralized управления

## 🚦 Риски и митигация

### Риск 1: Breaking changes в production дашбордах

**Вероятность:** HIGH
**Влияние:** CRITICAL
**Митигация:**
- Использовать recording rules для поддержки legacy имен (переходный период 30 дней)
- Обновить все дашборды до release
- Создать staging environment для тестирования

### Риск 2: Performance overhead от новых метрик

**Вероятность:** MEDIUM
**Влияние:** MEDIUM
**Митигация:**
- Benchmark новых метрик (target: <1ms overhead)
- Использовать singleton pattern для metrics registry
- Lazy initialization где возможно

### Риск 3: Несовместимость с будущими компонентами Alertmanager++

**Вероятность:** LOW
**Влияние:** HIGH
**Митигация:**
- Разработать taxonomy с учетом будущих компонентов (Grouping, Inhibition, Silencing)
- Зарезервировать subsystem prefixes для новых компонентов
- Документировать naming conventions для будущих задач

## 📅 Оценка времени

- **Фаза 1 (Аудит):** 2 часа
- **Фаза 2 (Design):** 3 часа
- **Фаза 3 (Implementation):** 8 часов
- **Фаза 4 (Migration Support):** 3 часа
- **Фаза 5 (Testing):** 2 часа
- **Фаза 6 (Documentation):** 2 часа

**Итого:** 20 часов (2.5 рабочих дня)

## 🔗 Зависимости

### Входящие зависимости

- ✅ TN-021: Prometheus middleware (завершена)
- ✅ TN-039: Circuit Breaker metrics (завершена)
- ✅ TN-038: Analytics Service metrics (завершена)

### Исходящие зависимости

- 🔄 TN-121 to TN-136: Alertmanager++ components (потребуют новые метрики)
- 🔄 Python Cleanup: унификация Python метрик перед sunset

## 📚 Справочные материалы

- [Prometheus Metric Naming Best Practices](https://prometheus.io/docs/practices/naming/)
- [OpenMetrics Specification](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md)
- [Google SRE Book: Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Internal: tasks/docs/prometheus-metrics.md](../docs/prometheus-metrics.md)

## 🎯 Определение успеха

Задача считается успешно выполненной, когда:

1. ✅ Все Prometheus метрики следуют единой naming convention
2. ✅ Database Pool metrics доступны в Grafana
3. ✅ Существующие дашборды работают без изменений
4. ✅ Документация обновлена и включает guidelines
5. ✅ SRE команда одобрила изменения
6. ✅ Staging тесты пройдены успешно

---

**Примечания:**
- Задача является foundation для будущих Alertmanager++ компонентов
- Требует тесной координации с SRE/DevOps командой
- Рекомендуется запланировать release window для production rollout
