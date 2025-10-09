# TN-137: Tasks - Аудит и унификация метрик Prometheus

**Дата создания:** 2025-10-09
**Статус:** NOT_STARTED
**Прогресс:** 0% (0/68 задач)

## 📊 Общий прогресс по фазам

```
Phase 1: Аудит           [ ] 0% (0/12)
Phase 2: Design          [ ] 0% (0/10)
Phase 3: Implementation  [ ] 0% (0/25)
Phase 4: Migration       [ ] 0% (0/12)
Phase 5: Testing         [ ] 0% (0/6)
Phase 6: Documentation   [ ] 0% (0/3)
```

---

## Phase 1: Аудит (2 часа) - 0/12

### 1.1 Инвентаризация метрик (45 мин)

- [ ] **T1.1.1:** Найти все файлы с определениями Prometheus метрик
  - Команда: `grep -r "promauto.New" go-app/`
  - Output: список файлов с метриками

- [ ] **T1.1.2:** Извлечь все имена метрик и их параметры
  - Namespace, Subsystem, Name, Help, Labels
  - Создать CSV: `tasks/TN-137-metrics-audit-unification/metrics_inventory.csv`

- [ ] **T1.1.3:** Классифицировать метрики по категориям
  - Business, Technical, Infrastructure
  - Пометить проблемные метрики (missing subsystem, inconsistent naming)

- [ ] **T1.1.4:** Проверить Database Pool metrics
  - Верифицировать, что они НЕ экспортируются в Prometheus
  - Задокументировать требуемые метрики

### 1.2 Анализ использования (45 мин)

- [ ] **T1.2.1:** Извлечь метрики из Grafana dashboards
  - Файл: `alert_history_grafana_dashboard_v3_enrichment.json`
  - Список всех используемых метрик

- [ ] **T1.2.2:** Найти recording rules (если есть)
  - Поиск в Prometheus config / Kubernetes ConfigMaps
  - Документировать существующие rules

- [ ] **T1.2.3:** Найти алерты, использующие метрики
  - PrometheusRule CRDs или alerting rules
  - Список метрик в alerts

- [ ] **T1.2.4:** Создать dependency graph
  - Какие метрики используются в дашбордах
  - Какие метрики используются в алертах
  - Риск breaking changes

### 1.3 Анализ проблем (30 мин)

- [ ] **T1.3.1:** Идентифицировать метрики с проблемами именования
  - Список метрик без subsystem
  - Список метрик с inconsistent naming

- [ ] **T1.3.2:** Найти дубликаты и overlapping метрики
  - Метрики с похожими names но разными labels
  - Возможности для consolidation

- [ ] **T1.3.3:** Оценить cardinality риски
  - Метрики с high cardinality labels (path, uuid)
  - Рекомендации по optimization

- [ ] **T1.3.4:** Создать аудит отчет
  - Файл: `tasks/TN-137-metrics-audit-unification/AUDIT_REPORT.md`
  - Summary проблем и recommendations

---

## Phase 2: Design (3 часа) - 0/10

### 2.1 Taxonomy разработка (60 мин)

- [ ] **T2.1.1:** Финализировать category structure
  - Business: alerts, llm, publishing
  - Technical: http, llm_cb, filter, enrichment
  - Infra: db, cache, repository

- [ ] **T2.1.2:** Определить naming conventions
  - Pattern: `<namespace>_<category>_<subsystem>_<name>_<unit>`
  - Examples для каждой категории

- [ ] **T2.1.3:** Создать taxonomy document
  - Файл: `tasks/TN-137-metrics-audit-unification/TAXONOMY.md`
  - Полное описание всех категорий и subsystems

### 2.2 Migration mapping (60 мин)

- [ ] **T2.2.1:** Создать mapping table старые → новые имена
  - CSV: `tasks/TN-137-metrics-audit-unification/metrics_migration_mapping.csv`
  - Columns: old_name, new_name, breaking_change, migration_strategy

- [ ] **T2.2.2:** Идентифицировать breaking changes
  - Метрики, которые НЕЛЬЗЯ мигрировать через recording rules
  - План митигации для каждого breaking change

- [ ] **T2.2.3:** Определить приоритет миграции
  - High priority: метрики в production alerts
  - Medium: метрики в основных дашбордах
  - Low: метрики для debugging

### 2.3 Guidelines разработка (60 мин)

- [ ] **T2.3.1:** Написать naming guidelines для разработчиков
  - Когда создавать business vs technical vs infra метрики
  - Как выбирать subsystem
  - Naming conventions и best practices

- [ ] **T2.3.2:** Создать examples для common use cases
  - Добавление counter метрики
  - Добавление histogram метрики
  - Добавление метрики с labels

- [ ] **T2.3.3:** Code review checklist для метрик
  - Файл: `tasks/TN-137-metrics-audit-unification/METRICS_CODE_REVIEW_CHECKLIST.md`

- [ ] **T2.3.4:** SRE review и approval
  - Presentation дизайна SRE команде
  - Сбор feedback и adjustments

---

## Phase 3: Implementation (8 часов) - 0/25

### 3.1 Metrics Registry (2 часа)

- [ ] **T3.1.1:** Создать `pkg/metrics/registry.go`
  - MetricsRegistry struct
  - Singleton pattern implementation
  - Category managers (Business, Technical, Infra)

- [ ] **T3.1.2:** Реализовать validation logic
  - ValidateMetricName() function
  - Regex для проверки naming convention
  - Error messages для invalid names

- [ ] **T3.1.3:** Unit tests для registry
  - Test singleton behavior
  - Test metric name validation
  - Test category managers initialization

- [ ] **T3.1.4:** Integration с main.go
  - Создание global registry
  - Передача в components

### 3.2 Business Metrics (1.5 часа)

- [ ] **T3.2.1:** Создать `pkg/metrics/business.go`
  - BusinessMetrics struct
  - Alerts subsystem metrics
  - LLM subsystem metrics
  - Publishing subsystem metrics

- [ ] **T3.2.2:** Реализовать NewBusinessMetrics()
  - Proper namespace/subsystem usage
  - All required labels
  - Appropriate buckets для histograms

- [ ] **T3.2.3:** Unit tests для business metrics
  - Test metric creation
  - Test metric recording
  - Test label values

- [ ] **T3.2.4:** Интеграция в enrichment service
  - Заменить calls на новые метрики

### 3.3 Technical Metrics (2 часа)

- [ ] **T3.3.1:** Создать `pkg/metrics/technical.go`
  - TechnicalMetrics struct
  - Aggregation существующих HTTP, Filter, Enrichment metrics

- [ ] **T3.3.2:** Рефакторинг LLM Circuit Breaker metrics
  - Новые имена: `technical_llm_cb_*`
  - Обновить `internal/infrastructure/llm/circuit_breaker_metrics.go`
  - Dual emission (старые + новые метрики)

- [ ] **T3.3.3:** Unit tests для technical metrics
  - Test existing metrics integration
  - Test new LLM CB metric names

- [ ] **T3.3.4:** Обновить calls в Circuit Breaker
  - `internal/infrastructure/llm/circuit_breaker.go`
  - Использовать новые имена метрик

### 3.4 Infrastructure Metrics (2.5 часа)

- [ ] **T3.4.1:** Создать `pkg/metrics/infra.go`
  - InfraMetrics struct
  - DatabaseMetrics
  - CacheMetrics
  - RepositoryMetrics

- [ ] **T3.4.2:** Реализовать Database Pool Prometheus export
  - Новый файл: `internal/database/postgres/prometheus.go`
  - PrometheusExporter struct
  - Periodic export goroutine

- [ ] **T3.4.3:** Интеграция DatabaseMetrics с Pool
  - Обновить `internal/database/postgres/pool.go`
  - Start PrometheusExporter в NewPool()

- [ ] **T3.4.4:** Рефакторинг Repository metrics
  - Обновить `internal/infrastructure/repository/postgres_history.go`
  - Новые имена: `infra_repository_*`
  - Dual emission

- [ ] **T3.4.5:** Unit tests для infra metrics
  - Test database metrics export
  - Test cache metrics
  - Test repository metrics

- [ ] **T3.4.6:** Integration test для DB metrics
  - Создать test pool
  - Выполнить queries
  - Verify metrics в Prometheus format

### 3.5 Cleanup и Optimization (1 час)

- [ ] **T3.5.1:** Удалить duplicate metric definitions
  - Consolidate overlapping metrics

- [ ] **T3.5.2:** Path normalization для HTTP metrics
  - Middleware для replace UUIDs в path
  - Reduce cardinality

- [ ] **T3.5.3:** Performance testing новых метрик
  - Benchmark overhead
  - Target: <1ms per metric recording

- [ ] **T3.5.4:** Code review и approval
  - Internal review
  - Address feedback

---

## Phase 4: Migration Support (3 часа) - 0/12

### 4.1 Recording Rules (1 час)

- [ ] **T4.1.1:** Создать Prometheus recording rules file
  - Файл: `helm/alert-history-go/templates/prometheus-rules.yaml` (или отдельно)
  - Mapping старых имен на новые

- [ ] **T4.1.2:** Recording rules для Repository metrics
  ```yaml
  - record: alert_history_query_duration_seconds
    expr: alert_history_infra_repository_query_duration_seconds
  ```

- [ ] **T4.1.3:** Recording rules для Circuit Breaker metrics
  ```yaml
  - record: alert_history_llm_circuit_breaker_state
    expr: alert_history_technical_llm_cb_state
  ```

- [ ] **T4.1.4:** Validation recording rules
  - Deploy в staging
  - Verify old metric names работают

### 4.2 Grafana Migration (1.5 часа)

- [ ] **T4.2.1:** Создать script для update Grafana dashboards
  - Python/Bash script
  - Автоматическая замена метрик в JSON

- [ ] **T4.2.2:** Обновить main dashboard
  - `alert_history_grafana_dashboard_v3_enrichment.json`
  - Заменить метрики на новые имена

- [ ] **T4.2.3:** Создать новые dashboard panels для DB metrics
  - Connection pool visualization
  - Query duration histogram
  - Error rates

- [ ] **T4.2.4:** Validation dashboards в staging
  - Deploy обновленные dashboards
  - Verify все panels работают

### 4.3 Documentation (30 мин)

- [ ] **T4.3.1:** Создать migration guide для SRE
  - Файл: `tasks/TN-137-metrics-audit-unification/MIGRATION_GUIDE.md`
  - Step-by-step instructions

- [ ] **T4.3.2:** Changelog для production release
  - Breaking changes
  - New metrics
  - Timeline для legacy support

- [ ] **T4.3.3:** Communication plan
  - Slack announcement
  - Documentation update notification

- [ ] **T4.3.4:** Runbook для troubleshooting
  - Common issues during migration
  - Rollback procedure

---

## Phase 5: Testing & Validation (2 часа) - 0/6

### 5.1 Unit Tests (45 мин)

- [ ] **T5.1.1:** Unit tests для MetricsRegistry
  - Test metric creation
  - Test validation
  - Test singleton

- [ ] **T5.1.2:** Unit tests для Business metrics
  - Test all subsystems
  - Test label combinations

### 5.2 Integration Tests (45 мин)

- [ ] **T5.2.1:** Integration test для Database Pool metrics
  - Create pool → verify metrics appear
  - Execute queries → verify duration metrics

- [ ] **T5.2.2:** Integration test для end-to-end flow
  - Send alert → verify all metrics recorded correctly
  - Check Business + Technical + Infra metrics

### 5.3 Performance Tests (30 мин)

- [ ] **T5.3.1:** Benchmark metrics overhead
  - Before/after comparison
  - Target: <1% latency increase

- [ ] **T5.3.2:** Load test с новыми метриками
  - High traffic scenario (1000 RPS)
  - Verify no memory leaks
  - Verify Prometheus scrape time acceptable

---

## Phase 6: Documentation (2 часа) - 0/3

### 6.1 Core Documentation (90 мин)

- [ ] **T6.1.1:** Обновить `tasks/docs/prometheus-metrics.md`
  - Полный список новых метрик
  - Taxonomy explanation
  - Examples для каждой категории

- [ ] **T6.1.2:** Создать `METRICS_NAMING_GUIDE.md`
  - Для разработчиков
  - How to add new metrics
  - Naming conventions
  - Best practices

- [ ] **T6.1.3:** Обновить `go-app/internal/infrastructure/llm/README.md`
  - Новые имена Circuit Breaker metrics
  - Migration notes

### 6.2 Examples & Queries (30 мин)

- [ ] **T6.2.1:** PromQL examples для новых метрик
  - Common queries
  - Alert examples
  - Dashboard queries

- [ ] **T6.2.2:** Code examples для разработчиков
  - How to use MetricsRegistry
  - How to add metrics to new component

- [ ] **T6.2.3:** Troubleshooting guide
  - Common issues
  - Debugging tips

---

## 🚀 Deployment Plan

### Stage 1: Development (Week 1)
- [ ] Complete Phase 1-3 (Audit, Design, Implementation)
- [ ] Unit tests pass
- [ ] Code review approved

### Stage 2: Staging (Week 2)
- [ ] Deploy с dual emission
- [ ] Deploy recording rules
- [ ] Update staging dashboards
- [ ] Validation testing

### Stage 3: Production Canary (Week 3)
- [ ] Deploy 10% rollout
- [ ] Monitor metrics overhead
- [ ] Monitor dashboard correctness
- [ ] Go/No-Go decision

### Stage 4: Production Full (Week 4)
- [ ] 100% rollout
- [ ] Monitor 48 hours
- [ ] Communicate success
- [ ] Plan legacy cleanup (30 days later)

---

## 📝 Checklist перед началом

- [ ] Прочитать requirements.md полностью
- [ ] Прочитать design.md полностью
- [ ] Согласовать taxonomy с SRE командой
- [ ] Получить approval на breaking changes
- [ ] Зарезервировать staging environment
- [ ] Создать backup существующих dashboards

---

## 📊 Метрики успеха задачи

| Метрика | Target | Текущее | Статус |
|---------|--------|---------|--------|
| Все метрики унифицированы | 100% | 0% | ⏳ |
| Database Pool metrics в Prometheus | Yes | No | ⏳ |
| Recording rules работают | 100% | 0% | ⏳ |
| Dashboards обновлены | 100% | 0% | ⏳ |
| Unit test coverage | >90% | 0% | ⏳ |
| Performance overhead | <1% | N/A | ⏳ |
| Documentation complete | 100% | 0% | ⏳ |

---

## 🐛 Known Issues / Tech Debt

*None yet - заполнять во время implementation*

---

## 📅 Timeline

| Phase | Start | End | Duration | Status |
|-------|-------|-----|----------|--------|
| Phase 1: Audit | TBD | TBD | 2h | ⏳ NOT_STARTED |
| Phase 2: Design | TBD | TBD | 3h | ⏳ NOT_STARTED |
| Phase 3: Implementation | TBD | TBD | 8h | ⏳ NOT_STARTED |
| Phase 4: Migration | TBD | TBD | 3h | ⏳ NOT_STARTED |
| Phase 5: Testing | TBD | TBD | 2h | ⏳ NOT_STARTED |
| Phase 6: Documentation | TBD | TBD | 2h | ⏳ NOT_STARTED |
| **Total** | - | - | **20h** | **0%** |

---

## 🔄 Обновления задачи

### 2025-10-09
- ✅ Создана начальная документация (requirements, design, tasks)
- ✅ Определена taxonomy метрик
- ✅ Создан план из 68 задач
- ⏳ Ожидание начала Phase 1

---

**Примечания:**
- Обновлять прогресс по каждой фазе в реальном времени
- Отмечать блокеры и риски немедленно
- Коммитить прогресс после каждой completed phase
- Создавать PR после Phase 3 для early review
