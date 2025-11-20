# TN-84: GET /api/dashboard/alerts/recent - Implementation Tasks

## Обзор

**Цель:** Реализовать GET /api/dashboard/alerts/recent endpoint с качеством 150%

**Целевое качество:** 150% (превышение базовых требований на 50%)

**Оценка времени:** ~8 часов (с учетом 150% качества)

**Статус:** 🔄 In Progress

---

## Phase 0: Analysis & Documentation ✅

**Цель:** Провести комплексный анализ задачи и создать документацию

**Время:** 1 час

**Статус:** ✅ COMPLETE

- [x] **T0.1**: Провести комплексный анализ задачи
  - [x] Изучить существующие endpoints (/history/recent)
  - [x] Изучить AlertHistoryRepository.GetRecentAlerts
  - [x] Изучить ClassificationEnricher integration
  - [x] Определить различия с /history/recent

- [x] **T0.2**: Создать requirements.md
  - [x] Обоснование задачи
  - [x] Пользовательские сценарии (2 US)
  - [x] Функциональные требования (4 FR)
  - [x] Нефункциональные требования (3 NFR)
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

**Цель:** Реализовать DashboardAlertsHandler

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T1.1**: Создать DashboardAlertsHandler структуру
  - [ ] Определить структуру в `go-app/cmd/server/handlers/dashboard_alerts.go`
  - [ ] Поля: historyRepo, classificationEnricher, cache, logger
  - [ ] Конструктор NewDashboardAlertsHandler

- [ ] **T1.2**: Реализовать GetRecentAlerts метод
  - [ ] Парсинг query параметров (limit, status, severity, include_classification)
  - [ ] Валидация параметров
  - [ ] Вызов repository с фильтрами
  - [ ] Опциональное обогащение classification
  - [ ] Форматирование response

- [ ] **T1.3**: Реализовать helper методы
  - [ ] parseQueryParams - парсинг и валидация
  - [ ] formatResponse - форматирование в компактный формат
  - [ ] applyFilters - применение фильтров

- [ ] **T1.4**: Response Models
  - [ ] DashboardAlertResponse структура
  - [ ] DashboardAlert структура
  - [ ] ClassificationSummary структура

---

## Phase 2: Repository Integration

**Цель:** Интегрировать с AlertHistoryRepository

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T2.1**: Использовать GetRecentAlerts
  - [ ] Вызов repository.GetRecentAlerts с limit
  - [ ] Обработка ошибок

- [ ] **T2.2**: Применить фильтры
  - [ ] Создать AlertFilters с status и severity
  - [ ] Использовать GetHistory с фильтрами (если нужны фильтры)
  - [ ] Или фильтровать после получения (in-memory)

- [ ] **T2.3**: Оптимизация
  - [ ] Использовать существующие индексы
  - [ ] Минимизировать количество запросов

---

## Phase 3: Classification Integration

**Цель:** Интегрировать с ClassificationEnricher

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T3.1**: Опциональное обогащение
  - [ ] Проверка параметра include_classification
  - [ ] Вызов enricher.EnrichAlerts если запрошено
  - [ ] Graceful degradation при отсутствии enricher

- [ ] **T3.2**: Форматирование classification
  - [ ] Конвертация в ClassificationSummary формат
  - [ ] Включение только необходимых полей (severity, confidence, source)

---

## Phase 4: Response Caching (Optional)

**Цель:** Добавить response caching для производительности

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T4.1**: Реализовать caching
  - [ ] Cache key generation (включает все параметры)
  - [ ] Cache TTL: 5-10 секунд
  - [ ] Cache lookup перед repository call

- [ ] **T4.2**: Cache invalidation
  - [ ] Опциональная invalidation при новых алертах
  - [ ] Или просто TTL-based expiration

---

## Phase 5: Testing

**Цель:** Обеспечить высокое качество через comprehensive testing

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T5.1**: Unit Tests
  - [ ] Тесты для parseQueryParams
  - [ ] Тесты для formatResponse
  - [ ] Тесты для applyFilters
  - [ ] Тесты для error handling
  - [ ] Целевое покрытие: 90%+

- [ ] **T5.2**: Integration Tests
  - [ ] End-to-end тест: request → repository → response
  - [ ] Тест с classification enrichment
  - [ ] Тест без classification service
  - [ ] Тест с фильтрами

- [ ] **T5.3**: Performance Tests
  - [ ] Benchmark: response time < 100ms для 10 алертов
  - [ ] Benchmark: response time < 200ms для 50 алертов
  - [ ] Load test: > 100 req/s

---

## Phase 6: Documentation & Finalization

**Цель:** Завершить документацию и подготовить к merge

**Время:** 1 час

**Статус:** ⏳ PENDING

- [ ] **T6.1**: Обновить README
  - [ ] Добавить раздел о Dashboard API endpoints
  - [ ] Примеры использования
  - [ ] Troubleshooting guide

- [ ] **T6.2**: Создать COMPLETION_REPORT.md
  - [ ] Итоговая статистика (LOC, tests, coverage)
  - [ ] Метрики качества (150% target)
  - [ ] Performance results
  - [ ] Lessons learned

- [ ] **T6.3**: Обновить CHANGELOG.md
  - [ ] Добавить запись о TN-84
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
- ✅ TN-77: Modern Dashboard Page (150%, Grade A+)
- ✅ TN-80: Classification Display (150%, Grade A+)

### Downstream (Unblocked)
- 🎯 TN-81: GET /api/dashboard/overview (может использовать этот endpoint)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Status:** 🔄 In Progress
**Target Quality:** 150%
**Estimated Completion:** 2025-11-21
