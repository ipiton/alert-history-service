# TN-72: POST /classification/classify - Implementation Tasks

## Обзор

**Цель:** Реализовать POST /api/v2/classification/classify endpoint для ручной классификации алертов с качеством 150%.

**Целевое качество:** 150% (превышение базовых требований на 50%)

**Оценка времени:** ~40 часов (с учетом 150% качества)

**Статус:** 🔄 In Progress

---

## Phase 0: Analysis & Documentation ✅

**Цель:** Провести комплексный анализ задачи и создать документацию

**Время:** 2 часа

### Задачи:
- [x] **T0.1**: Провести анализ существующих компонентов классификации
  - [x] Изучить ClassificationService интерфейс и реализацию
  - [x] Изучить существующий handler ClassifyAlert
  - [x] Изучить router integration
  - [x] Изучить зависимости (LLM client, cache, metrics)

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
  - [x] API contract
  - [x] Интеграция с существующими компонентами
  - [x] Тестирование стратегия
  - [x] Производительность и безопасность

- [x] **T0.4**: Создать tasks.md (этот файл)
  - [x] Детальный план реализации (9 фаз)
  - [x] Чеклисты для каждой фазы
  - [x] Критерии готовности

**Результат:** Полная документация (requirements + design + tasks)

---

## Phase 1: Git Branch Setup

**Цель:** Создать рабочую ветку и настроить окружение

**Время:** 0.5 часа

### Задачи:
- [ ] **T1.1**: Создать feature ветку
  ```bash
  git checkout -b feature/TN-72-manual-classification-endpoint-150pct
  ```

- [ ] **T1.2**: Проверить зависимости
  - [ ] Убедиться что ClassificationService доступен
  - [ ] Убедиться что LLM client доступен
  - [ ] Убедиться что Cache доступен
  - [ ] Убедиться что Metrics доступны

- [ ] **T1.3**: Создать структуру файлов
  - [ ] `go-app/internal/api/handlers/classification/classify_handler.go` (новый или обновить существующий)
  - [ ] `go-app/internal/api/handlers/classification/classify_handler_test.go`
  - [ ] `go-app/internal/api/handlers/classification/validation.go` (если нужно)

**Результат:** Готовая ветка для разработки

---

## Phase 2: Core Implementation

**Цель:** Реализовать основной функционал handler

**Время:** 4 часа

### Задачи:
- [ ] **T2.1**: Обновить ClassifyRequest модель
  - [ ] Добавить поле `Force bool` (опциональное)
  - [ ] Добавить validation tags
  - [ ] Добавить JSON tags

- [ ] **T2.2**: Обновить ClassifyResponse модель
  - [ ] Добавить поле `Cached bool`
  - [ ] Добавить поле `Model string` (опциональное)
  - [ ] Добавить поле `Timestamp time.Time`
  - [ ] Улучшить форматирование `ProcessingTime`

- [ ] **T2.3**: Реализовать ClassifyAlert handler
  - [ ] Parse request (JSON decoding)
  - [ ] Validate input (structural + business validation)
  - [ ] Extract force flag (default: false)
  - [ ] Create context with timeout (5s)
  - [ ] Handle force flag logic:
    - [ ] If force=true: invalidate cache + force classification
    - [ ] If force=false: check cache first, then classify
  - [ ] Format response
  - [ ] Record metrics
  - [ ] Send JSON response

- [ ] **T2.4**: Реализовать валидацию
  - [ ] Structural validation (validator/v10)
  - [ ] Business validation (custom validators)
  - [ ] Error formatting (детальные сообщения)

- [ ] **T2.5**: Реализовать обработку ошибок
  - [ ] Validation errors (400)
  - [ ] Service errors (500/503)
  - [ ] Rate limit errors (429)
  - [ ] Error response formatting

**Результат:** Рабочий handler с базовым функционалом

---

## Phase 3: Router Integration

**Цель:** Интегрировать handler в router

**Время:** 1 час

### Задачи:
- [ ] **T3.1**: Обновить router.go
  - [ ] Заменить PlaceholderHandler на реальный handler
  - [ ] Убедиться что middleware stack правильный
  - [ ] Проверить route path (`/api/v2/classification/classify`)

- [ ] **T3.2**: Обновить main.go (если нужно)
  - [ ] Убедиться что ClassificationHandlers инициализирован
  - [ ] Убедиться что ClassificationService доступен
  - [ ] Проверить dependency injection

- [ ] **T3.3**: Проверить компиляцию
  - [ ] `go build ./...`
  - [ ] Убедиться что нет ошибок компиляции
  - [ ] Убедиться что нет linter warnings

**Результат:** Handler интегрирован в router и доступен через API

---

## Phase 4: Unit Testing

**Цель:** Написать comprehensive unit tests

**Время:** 6 часов (150% quality = расширенное тестирование)

### Задачи:
- [ ] **T4.1**: Handler tests
  - [ ] Happy path (successful classification)
  - [ ] Cache hit scenario (L1 + L2)
  - [ ] Cache miss scenario (LLM call)
  - [ ] Force flag scenario (cache invalidation)
  - [ ] Fallback scenario (LLM unavailable)
  - [ ] Error scenarios (validation, service, timeout)

- [ ] **T4.2**: Validation tests
  - [ ] Valid requests (various alert formats)
  - [ ] Invalid requests (missing fields, wrong types)
  - [ ] Edge cases (empty strings, null values, special characters)
  - [ ] Business rule validation (fingerprint format, status values)

- [ ] **T4.3**: Error handling tests
  - [ ] All error types (400, 429, 500, 503)
  - [ ] Error response formatting
  - [ ] Request ID propagation
  - [ ] Error logging

- [ ] **T4.4**: Force flag tests
  - [ ] Force=true: cache invalidation
  - [ ] Force=true: new classification
  - [ ] Force=false: cache check first
  - [ ] Force=false: fallback to classification

- [ ] **T4.5**: Mock dependencies
  - [ ] Mock ClassificationService
  - [ ] Mock LLM Client
  - [ ] Mock Cache
  - [ ] Mock Metrics

**Целевое покрытие:** > 85% (target 80%)

**Результат:** Comprehensive unit test suite с > 85% coverage

---

## Phase 5: Integration Testing

**Цель:** Написать integration tests

**Время:** 4 часа

### Задачи:
- [ ] **T5.1**: End-to-end tests
  - [ ] Full flow (request → handler → service → response)
  - [ ] Cache integration (L1 + L2)
  - [ ] LLM integration (success + failure)
  - [ ] Fallback integration

- [ ] **T5.2**: Cache integration tests
  - [ ] L1 cache hit/miss
  - [ ] L2 cache hit/miss
  - [ ] Cache invalidation (force=true)
  - [ ] Cache TTL expiration

- [ ] **T5.3**: LLM integration tests
  - [ ] Successful LLM call
  - [ ] LLM timeout
  - [ ] LLM circuit breaker
  - [ ] LLM fallback

- [ ] **T5.4**: Router integration tests
  - [ ] Route registration
  - [ ] Middleware stack
  - [ ] Request/response flow

**Результат:** Comprehensive integration test suite

---

## Phase 6: Performance Optimization

**Цель:** Оптимизировать производительность

**Время:** 2 часа

### Задачи:
- [ ] **T6.1**: Benchmarks
  - [ ] Handler performance (cache hit/miss)
  - [ ] Validation performance
  - [ ] Serialization performance
  - [ ] Concurrent requests performance

- [ ] **T6.2**: Performance optimization
  - [ ] JSON parsing optimization
  - [ ] Response pooling
  - [ ] Early validation (fail fast)
  - [ ] Async cache writes (если нужно)

- [ ] **T6.3**: Performance validation
  - [ ] Cache hit p95 < 5ms ✅
  - [ ] Cache miss + LLM p95 < 2s ✅
  - [ ] Fallback p95 < 10ms ✅
  - [ ] Throughput > 1000 req/s ✅

**Результат:** Производительность превышает targets на 50%+

---

## Phase 7: Security Hardening

**Цель:** Усилить безопасность

**Время:** 2 часа

### Задачи:
- [ ] **T7.1**: Input validation
  - [ ] JSON injection protection
  - [ ] Path traversal protection (generator_url)
  - [ ] XSS protection (labels/annotations)
  - [ ] DoS protection (request size limits)

- [ ] **T7.2**: Rate limiting
  - [ ] Per-IP rate limiting (100 req/min)
  - [ ] Global rate limiting (1000 req/min)
  - [ ] Rate limit error handling

- [ ] **T7.3**: Security tests
  - [ ] Injection attack tests
  - [ ] Rate limiting tests
  - [ ] Request size limit tests
  - [ ] Authentication tests (если включено)

- [ ] **T7.4**: Audit logging
  - [ ] Request logging (request ID, fingerprint, force)
  - [ ] Error logging (error type, message)
  - [ ] Security event logging (rate limit hits, auth failures)

**Результат:** Security hardened endpoint (OWASP Top 10 compliant)

---

## Phase 8: Observability Integration

**Цель:** Интегрировать observability (метрики, логи, трейсинг)

**Время:** 2 часа

### Задачи:
- [ ] **T8.1**: Prometheus metrics
  - [ ] `classification_api_requests_total{status, method}`
  - [ ] `classification_api_duration_seconds{method}`
  - [ ] `classification_api_cache_hits_total{level}`
  - [ ] `classification_api_cache_misses_total`
  - [ ] `classification_api_errors_total{error_type}`

- [ ] **T8.2**: Structured logging
  - [ ] DEBUG logs (детальная информация)
  - [ ] INFO logs (успешные классификации)
  - [ ] WARN logs (fallback, cache misses)
  - [ ] ERROR logs (ошибки классификации)
  - [ ] Request ID в всех логах

- [ ] **T8.3**: Distributed tracing (опционально)
  - [ ] OpenTelemetry spans (если доступно)
  - [ ] Tags (fingerprint, force, cached, severity)
  - [ ] Events (cache_hit, cache_miss, llm_call, fallback)

- [ ] **T8.4**: Metrics validation
  - [ ] Убедиться что все метрики экспортируются
  - [ ] Убедиться что метрики корректны
  - [ ] Проверить интеграцию с Prometheus

**Результат:** Полная observability интеграция

---

## Phase 9: Documentation

**Цель:** Создать comprehensive документацию

**Время:** 3 часа

### Задачи:
- [ ] **T9.1**: OpenAPI 3.0 specification
  - [ ] Request schema
  - [ ] Response schema
  - [ ] Error schemas
  - [ ] Examples

- [ ] **T9.2**: API Guide
  - [ ] Quick start
  - [ ] Request examples (curl, Go, Python)
  - [ ] Response examples
  - [ ] Error handling guide
  - [ ] Best practices

- [ ] **T9.3**: Integration guide
  - [ ] How to use endpoint
  - [ ] Force flag usage
  - [ ] Cache behavior
  - [ ] Error handling
  - [ ] Rate limiting

- [ ] **T9.4**: Troubleshooting guide
  - [ ] Common issues
  - [ ] Error codes
  - [ ] Performance tuning
  - [ ] Debug tips

- [ ] **T9.5**: Godoc comments
  - [ ] Handler documentation
  - [ ] Request/Response models documentation
  - [ ] Error types documentation

**Результат:** Comprehensive документация (OpenAPI + API Guide + Integration Guide + Troubleshooting)

---

## Phase 10: Final Validation & Certification

**Цель:** Финальная валидация и сертификация качества

**Время:** 2 часа

### Задачи:
- [ ] **T10.1**: Code review checklist
  - [ ] Code quality (zero linter warnings)
  - [ ] Test coverage (> 85%)
  - [ ] Performance (превышает targets на 50%+)
  - [ ] Security (OWASP Top 10 compliant)
  - [ ] Documentation (comprehensive)

- [ ] **T10.2**: Integration validation
  - [ ] Router integration работает
  - [ ] Middleware stack работает
  - [ ] Service integration работает
  - [ ] Cache integration работает
  - [ ] LLM integration работает

- [ ] **T10.3**: Performance validation
  - [ ] Все benchmarks проходят
  - [ ] Производительность превышает targets
  - [ ] Нет memory leaks
  - [ ] Нет race conditions

- [ ] **T10.4**: Security validation
  - [ ] Security tests проходят
  - [ ] OWASP Top 10 compliant
  - [ ] Rate limiting работает
  - [ ] Input validation работает

- [ ] **T10.5**: Documentation validation
  - [ ] OpenAPI spec валидна
  - [ ] API Guide complete
  - [ ] Examples работают
  - [ ] Godoc comments complete

- [ ] **T10.6**: Create completion report
  - [ ] Summary of deliverables
  - [ ] Quality metrics
  - [ ] Performance results
  - [ ] Test coverage results
  - [ ] Certification (Grade A+)

**Результат:** ✅ PRODUCTION-READY, Grade A+, 150% Quality Certified

---

## Критерии готовности (Definition of Done)

### Функциональные критерии:
- [x] ✅ Requirements.md создан и утвержден
- [x] ✅ Design.md создан и утвержден
- [x] ✅ Tasks.md создан (этот файл)
- [ ] Handler реализован и работает
- [ ] Router integration завершена
- [ ] Все тесты проходят (> 85% coverage)
- [ ] Производительность превышает targets на 50%+
- [ ] Security hardened (OWASP Top 10 compliant)
- [ ] Observability интегрирована
- [ ] Документация complete

### Качественные критерии (150%):
- [ ] Test coverage: > 85% (target 80%) ✅
- [ ] Performance: превышает targets на 50%+ ✅
- [ ] Documentation: comprehensive (OpenAPI + API Guide + Integration Guide) ✅
- [ ] Security: OWASP Top 10 compliant ✅
- [ ] Code quality: zero linter warnings, zero race conditions ✅

### Production readiness:
- [ ] Zero breaking changes
- [ ] Backward compatible
- [ ] Graceful degradation работает
- [ ] Monitoring и alerting настроены
- [ ] Deployment готов

---

## Зависимости

### Требуемые зависимости (все завершены ✅):
- ✅ **TN-033**: ClassificationService (150% quality, Grade A+)
- ✅ **TN-029**: LLM Client (завершена)
- ✅ **TN-016**: Redis Cache (завершена)
- ✅ **TN-021**: Prometheus Metrics (завершена)
- ✅ **TN-039**: Circuit Breaker (завершена)
- ✅ **TN-071**: Classification Stats Endpoint (150% quality, Grade A+)

### Блокируемые задачи:
- 🎯 **TN-073**: GET /classification/models (может использовать похожую структуру)

---

## Риски и митигация

### РИСК-1: Высокая латентность LLM вызовов
**Митигация:** Двухуровневое кэширование, таймауты, fallback

### РИСК-2: Перегрузка LLM сервиса
**Митигация:** Rate limiting, circuit breaker, graceful degradation

### РИСК-3: Проблемы с кэшем
**Митигация:** L1 fallback, graceful degradation без кэша

---

## Метрики успешности

### Технические метрики:
- Test Coverage: > 85% ✅
- Performance: превышает targets на 50%+ ✅
- Availability: 99.9% с fallback ✅
- Error Rate: < 0.1% ✅

### Качественные метрики:
- Code Quality: zero linter warnings, zero race conditions ✅
- Documentation: comprehensive ✅
- Security: OWASP Top 10 compliant ✅
- Observability: все метрики экспортируются ✅

---

**Версия:** 1.0
**Дата создания:** 2025-11-17
**Последнее обновление:** 2025-11-17
**Статус:** 🔄 In Progress (Phase 0 Complete)
