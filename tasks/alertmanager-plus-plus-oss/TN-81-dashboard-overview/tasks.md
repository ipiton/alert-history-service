# TN-81: GET /api/dashboard/overview - Implementation Tasks

## Обзор

**Цель:** Реализовать GET /api/dashboard/overview endpoint с качеством 150%

**Целевое качество:** 150% (превышение базовых требований на 50%)

**Оценка времени:** ~10 часов (с учетом 150% качества)

**Статус:** 🔄 In Progress

---

## Phase 0: Analysis & Documentation ✅

**Цель:** Провести комплексный анализ задачи и создать документацию

**Время:** 1.5 часа

**Статус:** ✅ COMPLETE

- [x] **T0.1**: Провести комплексный анализ задачи
  - [x] Изучить legacy Python реализацию
  - [x] Изучить существующие компоненты (repository, classification, publishing)
  - [x] Определить источники данных

- [x] **T0.2**: Создать requirements.md
  - [x] Обоснование задачи
  - [x] Пользовательские сценарии (1 US)
  - [x] Функциональные требования (4 FR)
  - [x] Нефункциональные требования (2 NFR)
  - [x] Риски и митигация
  - [x] Критерии приемки

- [x] **T0.3**: Создать design.md
  - [x] Архитектурный обзор
  - [x] Детальный дизайн компонентов
  - [x] Формат данных и API контракты
  - [x] Интеграция с существующими компонентами
  - [x] Тестирование стратегия
  - [x] Производительность и безопасность

- [x] **T0.4**: Создать tasks.md (этот файл)

---

## Phase 1: Handler Implementation

**Цель:** Реализовать DashboardOverviewHandler

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T1.1**: Создать DashboardOverviewHandler структуру
  - [ ] Определить структуру в `go-app/cmd/server/handlers/dashboard_overview.go`
  - [ ] Поля: historyRepo, classificationService, publishingStats, cache, logger
  - [ ] Конструктор NewDashboardOverviewHandler

- [ ] **T1.2**: Реализовать GetOverview метод
  - [ ] Параллельный сбор статистики (goroutines)
  - [ ] Агрегация данных
  - [ ] Форматирование response
  - [ ] Обработка ошибок

- [ ] **T1.3**: Реализовать helper методы
  - [ ] collectAlertStats - сбор статистики алертов
  - [ ] collectClassificationStats - сбор статистики classification
  - [ ] collectPublishingStats - сбор статистики publishing
  - [ ] collectSystemHealth - сбор системного здоровья
  - [ ] aggregateStats - агрегация всех статистик

- [ ] **T1.4**: Response Models
  - [ ] DashboardOverviewResponse структура
  - [ ] Все поля из design.md

---

## Phase 2: Statistics Collection

**Цель:** Реализовать сбор статистики из разных источников

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T2.1**: Alert Statistics Collection
  - [ ] Использовать AlertHistoryRepository.GetHistory()
  - [ ] Подсчет total_alerts, active_alerts, resolved_alerts
  - [ ] Подсчет alerts_last_24h (фильтр по времени)

- [ ] **T2.2**: Classification Statistics Collection
  - [ ] Использовать ClassificationService.GetStats()
  - [ ] Использовать ClassificationService.Health()
  - [ ] Graceful degradation при отсутствии service

- [ ] **T2.3**: Publishing Statistics Collection
  - [ ] Использовать TargetDiscoveryManager или метрики
  - [ ] Получение publishing_mode
  - [ ] Получение successful/failed publishes
  - [ ] Graceful degradation при отсутствии publishing

- [ ] **T2.4**: System Health Collection
  - [ ] Cache.HealthCheck() для Redis
  - [ ] ClassificationService.Health() для LLM
  - [ ] Параллельные health checks с timeout

---

## Phase 3: Parallel Collection & Timeout

**Цель:** Реализовать параллельный сбор с timeout protection

**Время:** 1.5 часа

**Статус:** ⏳ PENDING

- [ ] **T3.1**: Parallel Collection
  - [ ] Использовать goroutines для каждого компонента
  - [ ] WaitGroup для синхронизации
  - [ ] Context с timeout (10 секунд общий)

- [ ] **T3.2**: Timeout Protection
  - [ ] Timeout на каждый компонент (5 секунд)
  - [ ] Graceful degradation при timeout
  - [ ] Логирование предупреждений

---

## Phase 4: Response Caching

**Цель:** Добавить response caching для производительности

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T4.1**: Реализовать caching
  - [ ] Cache key: `dashboard:overview`
  - [ ] Cache TTL: 10-30 секунд
  - [ ] Cache lookup перед collection

- [ ] **T4.2**: Cache invalidation
  - [ ] TTL-based expiration
  - [ ] Опциональная invalidation при изменениях

---

## Phase 5: Testing

**Цель:** Обеспечить высокое качество через comprehensive testing

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T5.1**: Unit Tests
  - [ ] Тесты для collectAlertStats
  - [ ] Тесты для collectClassificationStats
  - [ ] Тесты для collectPublishingStats
  - [ ] Тесты для collectSystemHealth
  - [ ] Тесты для aggregateStats
  - [ ] Тесты для error handling
  - [ ] Целевое покрытие: 90%+

- [ ] **T5.2**: Integration Tests
  - [ ] End-to-end тест: request → collection → response
  - [ ] Тест с всеми компонентами
  - [ ] Тест с отсутствующими компонентами (graceful degradation)
  - [ ] Тест с timeout scenarios

- [ ] **T5.3**: Performance Tests
  - [ ] Benchmark: response time < 200ms
  - [ ] Load test: > 50 req/s
  - [ ] Cache performance test

---

## Phase 6: Documentation & Finalization

**Цель:** Завершить документацию и подготовить к merge

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T6.1**: Обновить README
  - [ ] Добавить раздел о Dashboard Overview API
  - [ ] Примеры использования
  - [ ] Troubleshooting guide

- [ ] **T6.2**: Создать COMPLETION_REPORT.md
  - [ ] Итоговая статистика (LOC, tests, coverage)
  - [ ] Метрики качества (150% target)
  - [ ] Performance results
  - [ ] Lessons learned

- [ ] **T6.3**: Обновить CHANGELOG.md
  - [ ] Добавить запись о TN-81
  - [ ] Описание изменений

- [ ] **T6.4**: Code Review
  - [ ] Self-review кода
  - [ ] Проверка на code smells
  - [ ] Проверка на security issues

- [ ] **T6.5**: Final Validation
  - [ ] Все тесты проходят
  - [ ] Linter warnings исправлены
  - [ ] Documentation complete
  - [ ] Ready for merge

---

## Quality Gates

### Gate 1: Implementation Complete
- [ ] Все Phase 1-4 завершены
- [ ] Код компилируется без ошибок
- [ ] Базовые тесты проходят

### Gate 2: Testing Complete
- [ ] Все Phase 5 завершены
- [ ] Coverage: 90%+ для критических компонентов
- [ ] Все тесты проходят (unit, integration, performance)

### Gate 3: Performance Validated
- [ ] Все Phase 5 завершены
- [ ] Performance метрики соответствуют targets
- [ ] Load tests пройдены успешно

### Gate 4: Documentation Complete
- [ ] Все Phase 6 завершены
- [ ] README обновлен
- [ ] COMPLETION_REPORT создан
- [ ] CHANGELOG обновлен

### Gate 5: Production Ready
- [ ] Все quality gates пройдены
- [ ] Code review завершен
- [ ] Security audit пройден
- [ ] Ready for merge to main

---

## Dependencies

### Upstream (All Complete ✅)
- ✅ TN-37: Alert History Repository (150%, Grade A+)
- ✅ TN-33: Classification Service (150%, Grade A+)
- ✅ TN-77: Modern Dashboard Page (150%, Grade A+)
- ✅ TN-84: GET /api/dashboard/alerts/recent (150%, Grade A+)

### Downstream (Unblocked)
- 🎯 TN-83: GET /api/dashboard/health (может использовать этот endpoint)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Status:** 🔄 In Progress
**Target Quality:** 150%
**Estimated Completion:** 2025-11-21
