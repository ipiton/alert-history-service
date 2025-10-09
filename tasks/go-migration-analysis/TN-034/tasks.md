# TN-034: Enrichment Mode System - Чек-лист

**Обновлено**: 2025-10-09 (Validation Report 2.0)
**Статус**: ✅ ГОТОВА К РЕАЛИЗАЦИИ (0% выполнено)
**Ветка**: `feature/TN-034-enrichment-modes`
**Базовая ветка**: `feature/use-LLM`
**Validation Score**: ✅ **8.5/10 (Very Good)** - [Validation Report](./VALIDATION_REPORT_2025-10-09.md)

---

## 📊 Прогресс

**Phase 1 (Core Infrastructure)**: 0/38 задач (0%) - ✅ МОЖНО НАЧИНАТЬ
**Phase 2 (Integration)**: 0/17 задач (0%) - ✅ TN-33 ЗАВЕРШЕН (блокер устранен!)
**Phase 3 (Advanced Features)**: 0/10 задач (0%) - ℹ️ Опционально

**ИТОГО**: 0/65 задач (0%) - честная оценка ✅

---

## 🎯 Phase 1: Core Infrastructure (МОЖНО НАЧИНАТЬ)

### 1. Создать internal/core/services/enrichment.go
- [ ] 1.1. Определить EnrichmentMode type
  - [ ] 1.1.1. Константа `EnrichmentModeTransparent`
  - [ ] 1.1.2. Константа `EnrichmentModeEnriched`
  - [ ] 1.1.3. Константа `EnrichmentModeTransparentWithRecommendations`
  - [ ] 1.1.4. Метод `IsValid()`
  - [ ] 1.1.5. Метод `String()`
  - [ ] 1.1.6. Метод `ToMetricValue()`

- [ ] 1.2. Определить EnrichmentModeManager interface
  - [ ] 1.2.1. Метод `GetMode(ctx) (EnrichmentMode, error)`
  - [ ] 1.2.2. Метод `GetModeWithSource(ctx) (EnrichmentMode, string, error)`
  - [ ] 1.2.3. Метод `SetMode(ctx, mode) error`
  - [ ] 1.2.4. Метод `ValidateMode(mode) error`
  - [ ] 1.2.5. Метод `GetStats(ctx) (*EnrichmentStats, error)`
  - [ ] 1.2.6. Метод `RefreshCache(ctx) error`

- [ ] 1.3. Реализовать enrichmentModeManager struct
  - [ ] 1.3.1. Поля: cache, logger, metrics
  - [ ] 1.3.2. Поля: currentMode, currentSource, lastRefresh
  - [ ] 1.3.3. Поля: totalSwitches, lastSwitchTime, lastSwitchFrom
  - [ ] 1.3.4. Mutex для thread-safety
  - [ ] 1.3.5. Конструктор `NewEnrichmentModeManager()`

- [ ] 1.4. Реализовать GetMode() с fallback chain
  - [ ] 1.4.1. Read lock для currentMode
  - [ ] 1.4.2. Auto-refresh при stale cache (> 30s)
  - [ ] 1.4.3. Background refresh через goroutine
  - [ ] 1.4.4. Error handling

- [ ] 1.5. Реализовать SetMode() с Redis + memory
  - [ ] 1.5.1. Валидация режима
  - [ ] 1.5.2. Сохранение в Redis (primary)
  - [ ] 1.5.3. Fallback на memory при Redis failure
  - [ ] 1.5.4. Track mode switches (metrics)
  - [ ] 1.5.5. Logging mode changes

- [ ] 1.6. Реализовать RefreshCache()
  - [ ] 1.6.1. Попытка читать из Redis
  - [ ] 1.6.2. Fallback на ENV variable `ENRICHMENT_MODE`
  - [ ] 1.6.3. Fallback на default (`enriched`)
  - [ ] 1.6.4. Update in-memory cache
  - [ ] 1.6.5. Update metrics

- [ ] 1.7. Реализовать ValidateMode()
  - [ ] 1.7.1. Проверка через `IsValid()`
  - [ ] 1.7.2. Descriptive error message

- [ ] 1.8. Реализовать GetStats()
  - [ ] 1.8.1. Собрать текущее состояние
  - [ ] 1.8.2. Вернуть `EnrichmentStats`

### 2. Создать internal/core/services/enrichment_test.go
- [ ] 2.1. Unit tests для GetMode()
  - [ ] 2.1.1. Test: Redis available
  - [ ] 2.1.2. Test: Redis unavailable, fallback to ENV
  - [ ] 2.1.3. Test: Fallback to default
  - [ ] 2.1.4. Test: Auto-refresh on stale cache

- [ ] 2.2. Unit tests для SetMode()
  - [ ] 2.2.1. Test: Valid mode (transparent)
  - [ ] 2.2.2. Test: Valid mode (enriched)
  - [ ] 2.2.3. Test: Valid mode (transparent_with_recommendations)
  - [ ] 2.2.4. Test: Invalid mode
  - [ ] 2.2.5. Test: Redis save success
  - [ ] 2.2.6. Test: Redis save failure (memory fallback)
  - [ ] 2.2.7. Test: Metrics updated

- [ ] 2.3. Unit tests для ValidateMode()
  - [ ] 2.3.1. Test: All valid modes
  - [ ] 2.3.2. Test: Invalid mode

- [ ] 2.4. Unit tests для fallback chain
  - [ ] 2.4.1. Test: Redis → ENV → default
  - [ ] 2.4.2. Test: Priority order

- [ ] 2.5. Unit tests для error handling
  - [ ] 2.5.1. Test: Redis connection error
  - [ ] 2.5.2. Test: Invalid Redis data
  - [ ] 2.5.3. Test: Nil cache

### 3. Создать cmd/server/handlers/enrichment.go
- [ ] 3.1. Реализовать EnrichmentHandlers struct
  - [ ] 3.1.1. Поле: manager EnrichmentModeManager
  - [ ] 3.1.2. Поле: logger *slog.Logger
  - [ ] 3.1.3. Конструктор `NewEnrichmentHandlers()`

- [ ] 3.2. Реализовать GET /enrichment/mode
  - [ ] 3.2.1. Call `GetModeWithSource()`
  - [ ] 3.2.2. Response: `{"mode": "...", "source": "..."}`
  - [ ] 3.2.3. Error handling
  - [ ] 3.2.4. Logging

- [ ] 3.3. Реализовать POST /enrichment/mode
  - [ ] 3.3.1. Parse JSON request
  - [ ] 3.3.2. Validate mode
  - [ ] 3.3.3. Call `SetMode()`
  - [ ] 3.3.4. Response с новым режимом
  - [ ] 3.3.5. Error handling (400, 500)
  - [ ] 3.3.6. Logging

- [ ] 3.4. Request/Response types
  - [ ] 3.4.1. `EnrichmentModeResponse`
  - [ ] 3.4.2. `SetEnrichmentModeRequest`

### 4. Создать cmd/server/handlers/enrichment_test.go
- [ ] 4.1. HTTP tests для GET endpoint
  - [ ] 4.1.1. Test: GET returns current mode
  - [ ] 4.1.2. Test: GET returns source
  - [ ] 4.1.3. Test: GET error handling

- [ ] 4.2. HTTP tests для POST endpoint
  - [ ] 4.2.1. Test: POST valid mode (transparent)
  - [ ] 4.2.2. Test: POST valid mode (enriched)
  - [ ] 4.2.3. Test: POST valid mode (transparent_with_recommendations)
  - [ ] 4.2.4. Test: POST invalid mode (400 error)
  - [ ] 4.2.5. Test: POST invalid JSON (400 error)
  - [ ] 4.2.6. Test: POST server error (500)

- [ ] 4.3. Tests для validation errors
  - [ ] 4.3.1. Test: Empty mode
  - [ ] 4.3.2. Test: Unknown mode

- [ ] 4.4. Tests для response format
  - [ ] 4.4.1. Test: Response schema validation
  - [ ] 4.4.2. Test: Content-Type header

### 5. Добавить метрики в pkg/metrics/manager.go
- [ ] 5.1. Метрика: `enrichment_mode_switches_total`
  - [ ] 5.1.1. Type: Counter
  - [ ] 5.1.2. Labels: `from_mode`, `to_mode`
  - [ ] 5.1.3. Help text

- [ ] 5.2. Метрика: `enrichment_mode_status`
  - [ ] 5.2.1. Type: Gauge
  - [ ] 5.2.2. Values: 0=transparent, 1=enriched, 2=transparent_with_recommendations
  - [ ] 5.2.3. Help text

- [ ] 5.3. Метрика: `enrichment_mode_requests_total`
  - [ ] 5.3.1. Type: Counter
  - [ ] 5.3.2. Labels: `method` (GET/POST), `mode`
  - [ ] 5.3.3. Help text

- [ ] 5.4. Метрика: `enrichment_mode_redis_errors_total`
  - [ ] 5.4.1. Type: Counter
  - [ ] 5.4.2. Help text

### 6. Интегрировать в cmd/server/main.go
- [ ] 6.1. Инициализировать EnrichmentModeManager
  - [ ] 6.1.1. Передать Redis cache
  - [ ] 6.1.2. Передать logger
  - [ ] 6.1.3. Передать metrics manager

- [ ] 6.2. Зарегистрировать HTTP handlers
  - [ ] 6.2.1. Route: GET /enrichment/mode
  - [ ] 6.2.2. Route: POST /enrichment/mode

- [ ] 6.3. Добавить в dependency injection
  - [ ] 6.3.1. Сделать EnrichmentModeManager доступным для других компонентов

- [ ] 6.4. Настроить ENV variables
  - [ ] 6.4.1. Документировать `ENRICHMENT_MODE`

### 7. Документация Phase 1
- [ ] 7.1. Создать OpenAPI spec
  - [ ] 7.1.1. Schema для GET /enrichment/mode
  - [ ] 7.1.2. Schema для POST /enrichment/mode
  - [ ] 7.1.3. Error responses

- [ ] 7.2. Обновить README.md
  - [ ] 7.2.1. Описание enrichment modes
  - [ ] 7.2.2. API endpoints usage
  - [ ] 7.2.3. ENV variables

- [ ] 7.3. Создать docs/ENRICHMENT_MODES.md
  - [ ] 7.3.1. Подробное описание режимов
  - [ ] 7.3.2. Use cases
  - [ ] 7.3.3. Configuration guide
  - [ ] 7.3.4. Troubleshooting

### 8. Коммит Phase 1
- [ ] 8.1. Код компилируется без ошибок
- [ ] 8.2. Все tests проходят (coverage > 80%)
- [ ] 8.3. golangci-lint проходит
- [ ] 8.4. gosec проходит
- [ ] 8.5. Git commit: `feat(go): TN-034 enrichment mode manager and API`
- [ ] 8.6. Push в feature branch

---

## 🔗 Phase 2: Integration (✅ TN-033 ГОТОВ)

✅ **БЛОКЕР УСТРАНЕН**: TN-033 (Classification Service) завершен и merged в feature/use-LLM
- ✅ Commit: cfa3155 "merge: TN-33 validation complete - PRODUCTION-READY"
- ✅ Status: 90% готовности, оценка A-
- ✅ LLM Classification полностью функционален
- ✅ Intelligent Alert Proxy работает

### 9. Интегрировать в Classification Service
- [ ] 9.1. Передать EnrichmentModeManager в ClassificationService
  - [ ] 9.1.1. Добавить поле в struct
  - [ ] 9.1.2. Обновить конструктор

- [ ] 9.2. Проверять режим перед классификацией
  - [ ] 9.2.1. Call `GetMode()` в начале `ClassifyAlert()`
  - [ ] 9.2.2. Обработка ошибок

- [ ] 9.3. Пропускать LLM в transparent режимах
  - [ ] 9.3.1. If mode == transparent → return nil
  - [ ] 9.3.2. If mode == transparent_with_recommendations → return nil
  - [ ] 9.3.3. If mode == enriched → normal flow
  - [ ] 9.3.4. Logging

- [ ] 9.4. Добавить tests для integration
  - [ ] 9.4.1. Test: Classification skipped in transparent
  - [ ] 9.4.2. Test: Classification works in enriched
  - [ ] 9.4.3. Test: Mode fallback на ошибках

### 10. Интегрировать в Webhook Processing
- [ ] 10.1. Добавить middleware для mode resolution
  - [ ] 10.1.1. Resolve mode в начале request
  - [ ] 10.1.2. Add mode to context
  - [ ] 10.1.3. Logging

- [ ] 10.2. Обновить WebhookHandler
  - [ ] 10.2.1. Log current mode
  - [ ] 10.2.2. Pass mode через context

- [ ] 10.3. Добавить graceful mode switching
  - [ ] 10.3.1. Не прерывать активные requests
  - [ ] 10.3.2. Context-based mode resolution

- [ ] 10.4. Добавить integration tests
  - [ ] 10.4.1. Test: Webhook в transparent mode
  - [ ] 10.4.2. Test: Webhook в enriched mode
  - [ ] 10.4.3. Test: Mode switch во время processing

### 11. Интегрировать в Filter Engine
- [ ] 11.1. Передать EnrichmentModeManager в FilterEngine
  - [ ] 11.1.1. Добавить поле в struct
  - [ ] 11.1.2. Обновить конструктор

- [ ] 11.2. Пропускать фильтрацию в transparent_with_recommendations
  - [ ] 11.2.1. Call `GetMode()` в `ShouldPublish()`
  - [ ] 11.2.2. If mode == transparent_with_recommendations → return true
  - [ ] 11.2.3. Logging

- [ ] 11.3. Добавить tests
  - [ ] 11.3.1. Test: Filtering skipped в transparent_with_recommendations
  - [ ] 11.3.2. Test: Filtering applied в других режимах

### 12. End-to-End тесты
- [ ] 12.1. Test: transparent mode (без LLM)
  - [ ] 12.1.1. Отправить webhook
  - [ ] 12.1.2. Verify: LLM не вызван
  - [ ] 12.1.3. Verify: фильтрация применена
  - [ ] 12.1.4. Verify: алерт сохранен

- [ ] 12.2. Test: enriched mode (с LLM)
  - [ ] 12.2.1. Отправить webhook
  - [ ] 12.2.2. Verify: LLM вызван
  - [ ] 12.2.3. Verify: фильтрация применена
  - [ ] 12.2.4. Verify: алерт enriched

- [ ] 12.3. Test: transparent_with_recommendations (без фильтрации)
  - [ ] 12.3.1. Отправить webhook
  - [ ] 12.3.2. Verify: LLM не вызван
  - [ ] 12.3.3. Verify: фильтрация пропущена
  - [ ] 12.3.4. Verify: все алерты published

- [ ] 12.4. Test: mode switching под нагрузкой
  - [ ] 12.4.1. Параллельные requests
  - [ ] 12.4.2. Switch mode во время processing
  - [ ] 12.4.3. Verify: graceful switching

### 13. Коммит Phase 2
- [ ] 13.1. Все integration tests проходят
- [ ] 13.2. E2E tests проходят
- [ ] 13.3. Git commit: `feat(go): TN-034 integrate enrichment modes with processing pipeline`

---

## 🚀 Phase 3: Advanced Features (ОПЦИОНАЛЬНО)

### 14. Redis Pub/Sub для синхронизации
- [ ] 14.1. Реализовать Redis Pub/Sub listener
  - [ ] 14.1.1. Subscribe на channel `enrichment:mode:updates`
  - [ ] 14.1.2. Handle published events

- [ ] 14.2. Publish на mode change
  - [ ] 14.2.1. В `SetMode()` publish event
  - [ ] 14.2.2. Event format: `{"mode": "...", "timestamp": ...}`

- [ ] 14.3. Subscribe в каждом pod
  - [ ] 14.3.1. Start listener в `NewEnrichmentModeManager()`
  - [ ] 14.3.2. Handle reconnection

- [ ] 14.4. Обновлять in-memory cache
  - [ ] 14.4.1. При получении event → RefreshCache()
  - [ ] 14.4.2. Logging

### 15. Graceful Switching
- [ ] 15.1. Context-based mode resolution
  - [ ] 15.1.1. Resolve mode once per request
  - [ ] 15.1.2. Store в context
  - [ ] 15.1.3. Use context mode везде

- [ ] 15.2. Не прерывать активные requests
  - [ ] 15.2.1. In-flight requests используют старый mode
  - [ ] 15.2.2. Новые requests используют новый mode

- [ ] 15.3. Tests для graceful behavior
  - [ ] 15.3.1. Test: Concurrent requests с mode switch
  - [ ] 15.3.2. Test: No errors во время switch

### 16. Performance тесты
- [ ] 16.1. k6 load tests для mode switching
  - [ ] 16.1.1. Rapid GET /enrichment/mode
  - [ ] 16.1.2. Rapid POST /enrichment/mode

- [ ] 16.2. Benchmark для mode resolution
  - [ ] 16.2.1. Benchmark: GetMode() performance
  - [ ] 16.2.2. Target: < 1ms

- [ ] 16.3. Профилирование Redis latency
  - [ ] 16.3.1. Measure Redis GET latency
  - [ ] 16.3.2. Measure Redis SET latency

### 17. Финальный коммит Phase 3
- [ ] 17.1. Performance tests проходят
- [ ] 17.2. Git commit: `feat(go): TN-034 add advanced enrichment features`

---

## ✅ Definition of Done

### Code Quality
- [ ] Все unit tests проходят (coverage > 80%)
- [ ] Все integration tests проходят
- [ ] E2E tests для всех трех режимов проходят
- [ ] Go code проходит golangci-lint (zero errors)
- [ ] Go code проходит gosec (zero high/critical)
- [ ] Код соответствует Go Code Review Comments

### Documentation
- [ ] API документирован в OpenAPI/Swagger
- [ ] README.md обновлен с enrichment modes
- [ ] ENRICHMENT_MODES.md guide создан
- [ ] ENV variables документированы
- [ ] Комментарии в коде (godoc format)

### Observability
- [ ] Metrics экспортируются в Prometheus
- [ ] Logging на всех уровнях (debug, info, warn, error)
- [ ] Tracing (опционально)

### Parity & Compatibility
- [ ] Python parity: 100% (все 3 режима работают)
- [ ] API совместим с Python версией
- [ ] Нет breaking changes в API
- [ ] Redis format совместим с Python

### Production Readiness
- [ ] Graceful fallback при Redis failure
- [ ] Error handling везде
- [ ] Performance requirements выполнены (< 1ms mode resolution)
- [ ] Load tests пройдены
- [ ] Documentation complete

---

## 📈 Прогресс по фазам

### Phase 1: Core Infrastructure
**Статус**: ❌ НЕ НАЧАТА
**Прогресс**: 0/38 задач (0%)
**Блокеры**: ✅ НЕТ
**Можно начинать**: ✅ ДА (СЕЙЧАС!)
**Трудозатраты**: 2-3 дня

### Phase 2: Integration
**Статус**: ❌ НЕ НАЧАТА
**Прогресс**: 0/17 задач (0%)
**Блокеры**: ✅ НЕТ (TN-33 завершен!)
**Можно начинать**: ✅ ДА (после Phase 1)
**Трудозатраты**: 1-2 дня

### Phase 3: Advanced Features
**Статус**: ❌ НЕ НАЧАТА
**Прогресс**: 0/10 задач (0%)
**Блокеры**: Phase 1, Phase 2
**Можно начинать**: ✅ ДА (опционально)
**Трудозатраты**: 1 день

---

## 🔗 Зависимости

### Требуется для начала:
- ✅ TN-16: Redis Cache Wrapper (ГОТОВО)
- ✅ TN-21: Prometheus Metrics (ГОТОВО)
- ✅ TN-33: Classification Service (ГОТОВО, merged в feature/use-LLM)

### Блокирует:
- TN-35: Alert Filtering Engine
- TN-43: Webhook Validation

---

## 📋 Validation Information

**Validation Date**: 2025-10-09
**Validation Report**: [VALIDATION_REPORT_2025-10-09.md](./VALIDATION_REPORT_2025-10-09.md)
**Validation Score**: ✅ **8.5/10 (Very Good)** - READY FOR IMPLEMENTATION

**Key Findings**:
- ✅ Документация: 9.2/10 (Excellent)
- ✅ Готовность: 9.7/10 (Excellent)
- ✅ Блокеры: устранены (TN-33 завершен)
- ✅ Python reference: 100% функциональна
- ❌ Go implementation: 0% (честно)

**Рекомендация**: ✅ **ОДОБРЕНО ДЛЯ РЕАЛИЗАЦИИ**

---

**Последнее обновление**: 2025-10-09 (Validation 2.0)
**Автор**: AI Code Analyst
**Версия**: 2.1
