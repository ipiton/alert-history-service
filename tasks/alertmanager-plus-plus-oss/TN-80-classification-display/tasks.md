# TN-80: Classification Display - Implementation Tasks

## Обзор

**Цель:** Реализовать расширенное отображение классификации алертов в UI с качеством 150%

**Целевое качество:** 150% (превышение базовых требований на 50%)

**Оценка времени:** ~24 часа (с учетом 150% качества)

**Статус:** 🔄 In Progress

---

## Phase 0: Analysis & Documentation ✅

**Цель:** Провести комплексный анализ задачи и создать документацию

**Время:** 2 часа

**Статус:** ✅ COMPLETE

- [x] **T0.1**: Провести комплексный анализ задачи
  - [x] Изучить существующие компоненты классификации (TN-33, TN-71, TN-72)
  - [x] Изучить существующие UI компоненты (TN-76, TN-77, TN-79)
  - [x] Изучить структуру данных ClassificationResult
  - [x] Изучить зависимости и интеграции

- [x] **T0.2**: Создать requirements.md
  - [x] Обоснование задачи
  - [x] Пользовательские сценарии (4 US)
  - [x] Функциональные требования (5 FR)
  - [x] Нефункциональные требования (5 NFR)
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

## Phase 1: Branch Setup & Environment

**Цель:** Подготовить рабочую ветку и окружение

**Время:** 30 минут

**Статус:** ⏳ PENDING

- [ ] **T1.1**: Создать git branch
  - [ ] `git checkout -b feature/TN-80-classification-display-150pct`
  - [ ] Убедиться, что базируется на main (latest)

- [ ] **T1.2**: Проверить зависимости
  - [ ] ClassificationService доступен (TN-33)
  - [ ] Template Engine доступен (TN-76)
  - [ ] AlertListUIHandler доступен (TN-79)
  - [ ] Cache доступен (TN-16)

- [ ] **T1.3**: Настроить локальное окружение
  - [ ] Запустить PostgreSQL и Redis
  - [ ] Запустить приложение
  - [ ] Проверить доступность classification endpoints

---

## Phase 2: Classification Enricher Implementation

**Цель:** Реализовать компонент для обогащения алертов данными классификации

**Время:** 4 часа

**Статус:** ⏳ PENDING

- [ ] **T2.1**: Создать интерфейс ClassificationEnricher
  - [ ] Определить интерфейс в `go-app/internal/ui/classification_enricher.go`
  - [ ] Методы: EnrichAlerts, EnrichAlert, BatchEnrich
  - [ ] Определить структуру EnrichedAlert

- [ ] **T2.2**: Реализовать DefaultClassificationEnricher
  - [ ] Реализовать EnrichAlerts с cache lookup
  - [ ] Реализовать batch processing (10-20 алертов)
  - [ ] Реализовать graceful degradation
  - [ ] Добавить request-scoped cache

- [ ] **T2.3**: Интеграция с ClassificationService
  - [ ] Использовать GetCachedClassification для cache lookup
  - [ ] Использовать ClassifyBatch для batch classification
  - [ ] Обработка ошибок (fallback на label-based severity)

- [ ] **T2.4**: Unit Tests
  - [ ] Тесты для EnrichAlerts (cache hit, cache miss, batch)
  - [ ] Тесты для graceful degradation
  - [ ] Тесты для error handling
  - [ ] Целевое покрытие: 90%+

---

## Phase 3: Enhanced Alert Card Template

**Цель:** Расширить alert-card partial для отображения классификации

**Время:** 3 часа

**Статус:** ⏳ PENDING

- [ ] **T3.1**: Обновить alert-card.html
  - [ ] Добавить classification badge с severity и confidence
  - [ ] Добавить expandable секцию для reasoning и recommendations
  - [ ] Добавить метаданные (processing time, source)
  - [ ] Добавить ARIA labels и accessibility attributes

- [ ] **T3.2**: Добавить CSS стили
  - [ ] Color coding для severity (critical=red, warning=yellow, info=blue, noise=gray)
  - [ ] Progress bar для confidence
  - [ ] Smooth transitions для expand/collapse
  - [ ] Responsive design (mobile-first)

- [ ] **T3.3**: Добавить JavaScript для интерактивности
  - [ ] Toggle expand/collapse для classification details
  - [ ] Keyboard navigation (Enter, Escape)
  - [ ] Accessibility support (ARIA expanded)

- [ ] **T3.4**: Template Functions
  - [ ] classificationSeverityClass() - CSS класс для severity
  - [ ] classificationConfidencePercent() - процент confidence
  - [ ] classificationConfidenceColor() - цвет для confidence
  - [ ] formatClassificationReasoning() - форматирование reasoning (markdown)

---

## Phase 4: AlertListUIHandler Integration

**Цель:** Интегрировать ClassificationEnricher в AlertListUIHandler

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T4.1**: Обновить AlertListUIHandler
  - [ ] Добавить ClassificationService и ClassificationEnricher в структуру
  - [ ] Обновить конструктор NewAlertListUIHandler
  - [ ] Добавить инициализацию enricher в main.go

- [ ] **T4.2**: Обновить RenderAlertList метод
  - [ ] Вызвать enricher.EnrichAlerts после получения алертов
  - [ ] Обработать ошибки (graceful degradation)
  - [ ] Передать enriched alerts в template

- [ ] **T4.3**: Обновить template data structure
  - [ ] Добавить ClassificationDisplayData в AlertCardData
  - [ ] Конвертировать EnrichedAlert в template-friendly формат
  - [ ] Обработать отсутствие classification (nil-safe)

- [ ] **T4.4**: Integration Tests
  - [ ] Тест: RenderAlertList с classification data
  - [ ] Тест: RenderAlertList без classification (graceful degradation)
  - [ ] Тест: RenderAlertList с batch enrichment

---

## Phase 5: Classification Filters & Sorting

**Цель:** Добавить фильтрацию и сортировку по полям классификации

**Время:** 3 часа

**Статус:** ⏳ PENDING

- [ ] **T5.1**: Расширить AlertListFilters
  - [ ] Добавить ClassificationSeverity (*string)
  - [ ] Добавить MinConfidence и MaxConfidence (*float64)
  - [ ] Добавить HasClassification (*bool)
  - [ ] Добавить ClassificationSource (*string)

- [ ] **T5.2**: Обновить parseFilters метод
  - [ ] Парсинг classification_severity из query params
  - [ ] Парсинг min_confidence и max_confidence
  - [ ] Парсинг has_classification
  - [ ] Валидация параметров

- [ ] **T5.3**: Расширить AlertListSorting
  - [ ] Добавить сортировку по confidence (ASC/DESC)
  - [ ] Добавить сортировку по severity (custom order)
  - [ ] Обновить parseSorting метод

- [ ] **T5.4**: Обновить SQL queries (если classification в БД)
  - [ ] Добавить JOIN на alert_classifications table
  - [ ] Добавить WHERE условия для classification filters
  - [ ] Добавить ORDER BY для classification sorting
  - [ ] Добавить индексы для производительности

- [ ] **T5.5**: Обновить UI фильтры
  - [ ] Добавить dropdown для classification severity
  - [ ] Добавить range slider для confidence
  - [ ] Добавить checkbox для has_classification
  - [ ] Добавить опции сортировки в dropdown

- [ ] **T5.6**: Unit Tests
  - [ ] Тесты для parseFilters с classification params
  - [ ] Тесты для parseSorting с classification fields
  - [ ] Тесты для SQL query generation

---

## Phase 6: Classification Detail Modal

**Цель:** Реализовать модальное окно с детальной информацией о классификации

**Время:** 3 часа

**Статус:** ⏳ PENDING

- [ ] **T6.1**: Создать modal template
  - [ ] Создать `partials/classification-modal.html`
  - [ ] Структура: reasoning, recommendations, metadata
  - [ ] Добавить ARIA modal pattern

- [ ] **T6.2**: Добавить CSS для modal
  - [ ] Overlay и backdrop
  - [ ] Modal container (centered, responsive)
  - [ ] Close button
  - [ ] Smooth animations (fade in/out)

- [ ] **T6.3**: Добавить JavaScript для modal
  - [ ] Open modal при клике на classification badge
  - [ ] Close modal (button, overlay, Escape key)
  - [ ] Focus trap (accessibility)
  - [ ] ARIA announcements

- [ ] **T6.4**: API endpoint для classification details
  - [ ] GET /api/v2/alerts/{fingerprint}/classification
  - [ ] Handler: GetAlertClassification
  - [ ] Response: ClassificationResult + metadata

- [ ] **T6.5**: Integration Tests
  - [ ] Тест: открытие modal
  - [ ] Тест: закрытие modal (все способы)
  - [ ] Тест: keyboard navigation
  - [ ] Тест: screen reader support

---

## Phase 7: Performance Optimization

**Цель:** Оптимизировать производительность отображения классификации

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T7.1**: Request-scoped cache
  - [ ] Реализовать in-memory cache для запроса
  - [ ] Избежание дублирующих запросов к ClassificationService
  - [ ] Очистка cache после завершения запроса

- [ ] **T7.2**: Batch optimization
  - [ ] Оптимизировать batch size (10-20 алертов)
  - [ ] Parallel processing для независимых алертов
  - [ ] Early exit при ошибках

- [ ] **T7.3**: Lazy loading
  - [ ] Initial render: только severity и confidence
  - [ ] Expand details: загрузка reasoning и recommendations
  - [ ] Modal view: полная информация по требованию

- [ ] **T7.4**: SQL optimization (если classification в БД)
  - [ ] Добавить индексы на classification fields
  - [ ] Оптимизировать JOIN queries
  - [ ] Использовать EXPLAIN для анализа

- [ ] **T7.5**: Performance Tests
  - [ ] Benchmark: EnrichAlerts для 100 алертов
  - [ ] Benchmark: RenderAlertList с classification
  - [ ] Load test: 1000 алертов с classification
  - [ ] Целевые метрики: < 500ms page load (p95)

---

## Phase 8: Testing & Quality Assurance

**Цель:** Обеспечить высокое качество через comprehensive testing

**Время:** 4 часа

**Статус:** ⏳ PENDING

- [ ] **T8.1**: Unit Tests
  - [ ] ClassificationEnricher (90%+ coverage)
  - [ ] Template functions (100% coverage)
  - [ ] Filter/Sort logic (90%+ coverage)
  - [ ] Error handling (100% coverage)

- [ ] **T8.2**: Integration Tests
  - [ ] AlertListUIHandler с ClassificationEnricher
  - [ ] Template rendering с classification data
  - [ ] Filter/Sort с classification fields
  - [ ] Cache integration

- [ ] **T8.3**: E2E Tests
  - [ ] User flow: view classification → expand details
  - [ ] User flow: filter by severity → verify results
  - [ ] User flow: sort by confidence → verify order
  - [ ] Mobile view: verify responsive layout

- [ ] **T8.4**: Accessibility Tests
  - [ ] Keyboard navigation (Tab, Enter, Escape)
  - [ ] Screen reader support (NVDA/JAWS)
  - [ ] Color contrast (WCAG 2.1 AA)
  - [ ] ARIA labels validation

- [ ] **T8.5**: Performance Tests
  - [ ] Page load time: < 500ms (p95)
  - [ ] Alert card render: < 10ms (p95)
  - [ ] Classification enrichment: < 50ms per batch
  - [ ] Cache hit rate: > 80%

---

## Phase 9: Documentation & Finalization

**Цель:** Завершить документацию и подготовить к merge

**Время:** 2 часа

**Статус:** ⏳ PENDING

- [ ] **T9.1**: Обновить README
  - [ ] Добавить раздел о Classification Display
  - [ ] Примеры использования
  - [ ] Troubleshooting guide

- [ ] **T9.2**: Создать COMPLETION_REPORT.md
  - [ ] Итоговая статистика (LOC, tests, coverage)
  - [ ] Метрики качества (150% target)
  - [ ] Performance results
  - [ ] Lessons learned

- [ ] **T9.3**: Обновить CHANGELOG.md
  - [ ] Добавить запись о TN-80
  - [ ] Описание изменений
  - [ ] Breaking changes (если есть)

- [ ] **T9.4**: Code Review
  - [ ] Self-review кода
  - [ ] Проверка на code smells
  - [ ] Проверка на security issues
  - [ ] Проверка на performance issues

- [ ] **T9.5**: Final Validation
  - [ ] Все тесты проходят
  - [ ] Linter warnings исправлены
  - [ ] Documentation complete
  - [ ] Ready for merge

---

## Quality Gates

### Gate 1: Implementation Complete
- [ ] Все Phase 1-6 завершены
- [ ] Код компилируется без ошибок
- [ ] Базовые тесты проходят

### Gate 2: Testing Complete
- [ ] Все Phase 8 завершены
- [ ] Coverage: 90%+ для критических компонентов
- [ ] Все тесты проходят (unit, integration, E2E)

### Gate 3: Performance Validated
- [ ] Все Phase 7 завершены
- [ ] Performance метрики соответствуют targets
- [ ] Load tests пройдены успешно

### Gate 4: Documentation Complete
- [ ] Все Phase 9 завершены
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
- ✅ TN-33: Alert Classification Service (150%, Grade A+)
- ✅ TN-71: GET /classification/stats (150%, Grade A+)
- ✅ TN-72: POST /classification/classify (150%, Grade A+)
- ✅ TN-76: Dashboard Template Engine (165.9%, Grade A+)
- ✅ TN-77: Modern Dashboard Page (150%, Grade A+)
- ✅ TN-79: Alert List with Filtering (150%, Grade A+)

### Downstream (Unblocked)
- 🎯 TN-81: GET /api/dashboard/overview (может использовать classification stats)
- 🎯 TN-83: GET /api/dashboard/health (может включать classification health)

---

## Risk Mitigation Checklist

- [ ] Performance degradation mitigation (caching, batch processing)
- [ ] Graceful degradation при отсутствии classification
- [ ] XSS protection (HTML escaping, sanitization)
- [ ] Accessibility compliance (WCAG 2.1 AA)
- [ ] Backward compatibility (legacy alerts без classification)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Status:** 🔄 In Progress
**Target Quality:** 150%
**Estimated Completion:** 2025-11-22
