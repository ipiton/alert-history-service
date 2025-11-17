# TN-71: GET /classification/stats - LLM Statistics Endpoint

## 1. Обоснование задачи

### 1.1 Бизнес-контекст
Система классификации алертов с использованием LLM является критически важным компонентом Alert History Service. Для эффективного мониторинга, оптимизации и отладки системы классификации необходимо предоставить операторам и аналитикам детальную статистику работы LLM классификатора через REST API.

### 1.2 Пользовательские сценарии

#### US-1: DevOps Engineer - Мониторинг производительности
**Как** DevOps инженер
**Я хочу** видеть статистику работы LLM классификатора
**Чтобы** отслеживать производительность, cache hit rate и выявлять проблемы

**Критерии приемки:**
- [ ] GET /api/v2/classification/stats возвращает 200 OK
- [ ] Ответ содержит метрики: total_classified, cache_hit_rate, avg_latency_ms
- [ ] Метрики обновляются в реальном времени
- [ ] Время ответа < 50ms (p95)

#### US-2: Data Scientist - Анализ качества классификации
**Как** Data Scientist
**Я хочу** видеть распределение классификаций по severity и confidence
**Чтобы** анализировать качество работы LLM и корректировать модели

**Критерии приемки:**
- [ ] Ответ содержит by_severity статистику (count, avg_confidence)
- [ ] Ответ содержит распределение confidence scores
- [ ] Доступна статистика по моделям (если используется несколько)
- [ ] Данные агрегируются за последние 24 часа

#### US-3: SRE - Troubleshooting
**Как** SRE инженер
**Я хочу** видеть ошибки и fallback статистику
**Чтобы** быстро выявлять проблемы с LLM сервисом

**Критерии приемки:**
- [ ] Ответ содержит error_rate и fallback_rate
- [ ] Ответ содержит last_error и last_error_time
- [ ] Ответ содержит LLM success rate
- [ ] Доступна статистика по источникам классификации (LLM vs fallback vs cache)

#### US-4: Platform Team - Capacity Planning
**Как** Platform Team
**Я хочу** видеть нагрузку на LLM классификатор
**Чтобы** планировать масштабирование и оптимизацию

**Критерии приемки:**
- [ ] Ответ содержит throughput метрики (requests/sec)
- [ ] Ответ содержит latency percentiles (p50, p95, p99)
- [ ] Доступна статистика по времени суток (опционально)
- [ ] Метрики интегрированы с Prometheus

## 2. Функциональные требования

### FR-1: Базовые метрики классификации
**Приоритет:** HIGH
**Описание:** Endpoint должен возвращать базовые метрики работы классификатора

**Детали:**
- `total_classified` - общее количество классифицированных алертов
- `total_requests` - общее количество запросов на классификацию
- `classification_rate` - процент успешных классификаций (total_classified / total_requests)
- `avg_confidence` - средний confidence score всех классификаций
- `avg_latency_ms` - средняя задержка классификации в миллисекундах

**Источники данных:**
- ClassificationService.GetStats() - базовые метрики
- BusinessMetrics Prometheus метрики - расширенные метрики

### FR-2: Статистика по severity
**Приоритет:** HIGH
**Описание:** Endpoint должен возвращать распределение классификаций по severity уровням

**Детали:**
- Для каждого severity (critical, warning, info, noise):
  - `count` - количество классификаций
  - `avg_confidence` - средний confidence для этого severity
  - `percentage` - процент от общего количества (опционально)

**Источники данных:**
- Prometheus метрика: `alert_history_business_llm_classifications_total{severity="..."}`
- Агрегация из ClassificationService (если доступна)

### FR-3: Cache статистика
**Приоритет:** HIGH
**Описание:** Endpoint должен возвращать статистику использования кэша

**Детали:**
- `cache_hit_rate` - процент попаданий в кэш (L1 + L2)
- `l1_cache_hits` - количество попаданий в L1 (memory) кэш
- `l2_cache_hits` - количество попаданий в L2 (Redis) кэш
- `cache_misses` - количество промахов кэша
- `cache_size` - текущий размер кэша (опционально)

**Источники данных:**
- ClassificationService.GetStats() - CacheHitRate
- Prometheus метрики:
  - `alert_history_business_classification_l1_cache_hits_total`
  - `alert_history_business_classification_l2_cache_hits_total`

### FR-4: LLM статистика
**Приоритет:** HIGH
**Описание:** Endpoint должен возвращать статистику использования LLM

**Детали:**
- `llm_requests` - общее количество запросов к LLM
- `llm_success_rate` - процент успешных запросов к LLM
- `llm_failures` - количество неудачных запросов
- `llm_avg_latency_ms` - средняя задержка LLM запросов
- `llm_usage_rate` - процент использования LLM от общего количества запросов

**Источники данных:**
- ClassificationService.GetStats() - LLMSuccessRate
- Prometheus метрика: `alert_history_business_classification_duration_seconds{source="llm"}`

### FR-5: Fallback статистика
**Приоритет:** MEDIUM
**Описание:** Endpoint должен возвращать статистику использования fallback классификации

**Детали:**
- `fallback_used` - количество использований fallback классификации
- `fallback_rate` - процент использования fallback от общего количества запросов
- `fallback_avg_latency_ms` - средняя задержка fallback классификации

**Источники данных:**
- ClassificationService.GetStats() - FallbackRate
- Prometheus метрика: `alert_history_business_classification_duration_seconds{source="fallback"}`

### FR-6: Error статистика
**Приоритет:** MEDIUM
**Описание:** Endpoint должен возвращать статистику ошибок

**Детали:**
- `errors_total` - общее количество ошибок
- `error_rate` - процент ошибок от общего количества запросов
- `last_error` - текст последней ошибки
- `last_error_time` - время последней ошибки

**Источники данных:**
- ClassificationService.GetStats() - LastError, LastErrorTime

### FR-7: Model статистика (опционально)
**Приоритет:** LOW
**Описание:** Endpoint может возвращать статистику по используемым моделям

**Детали:**
- Для каждой модели:
  - `requests` - количество запросов
  - `avg_latency_ms` - средняя задержка
  - `success_rate` - процент успешных запросов

**Источники данных:**
- Prometheus метрики (если доступны метки по моделям)
- ClassificationService конфигурация

### FR-8: Time-based агрегация (опционально)
**Приоритет:** LOW
**Описание:** Endpoint может поддерживать временные фильтры

**Детали:**
- Query параметр `time_range` (last_24h, last_7d, last_30d, all)
- Агрегация метрик за указанный период
- Использование Prometheus range queries для исторических данных

**Источники данных:**
- Prometheus range queries (если доступны)
- Database queries (если требуется точность)

## 3. Нефункциональные требования

### NFR-1: Производительность
**Приоритет:** HIGH
**Требования:**
- Время ответа p95 < 50ms (без запросов к БД)
- Время ответа p95 < 200ms (с запросами к Prometheus)
- Throughput > 1000 req/s
- Zero allocations в hot path (где возможно)

**Обоснование:**
- Endpoint будет использоваться для мониторинга и дашбордов
- Высокая частота запросов от Grafana и других систем мониторинга
- Не должен создавать нагрузку на систему

### NFR-2: Надежность
**Приоритет:** HIGH
**Требования:**
- Graceful degradation при недоступности Prometheus
- Fallback на ClassificationService.GetStats() при ошибках
- Thread-safe операции (concurrent access)
- Zero data races

**Обоснование:**
- Endpoint критичен для мониторинга
- Должен работать даже при частичных сбоях инфраструктуры

### NFR-3: Масштабируемость
**Приоритет:** MEDIUM
**Требования:**
- Поддержка concurrent requests (100+)
- Эффективное использование памяти (кэширование метрик)
- Опциональное кэширование ответа (ETag, Cache-Control)

**Обоснование:**
- Множественные клиенты могут запрашивать статистику одновременно
- Необходимо минимизировать нагрузку на Prometheus

### NFR-4: Безопасность
**Приоритет:** HIGH
**Требования:**
- OWASP Top 10 compliance
- Input validation (query parameters)
- Rate limiting (опционально, через middleware)
- Security headers (через middleware)

**Обоснование:**
- Endpoint публичный, требует защиты от злоупотреблений

### NFR-5: Observability
**Приоритет:** HIGH
**Требования:**
- Prometheus метрики для самого endpoint (requests_total, duration_seconds)
- Structured logging (slog)
- Request ID tracking
- Error tracking

**Обоснование:**
- Необходимо отслеживать использование endpoint
- Отладка проблем с производительностью

### NFR-6: Совместимость
**Приоритет:** MEDIUM
**Требования:**
- Backward compatibility с существующим API (docs/API.md)
- OpenAPI 3.0 спецификация
- JSON response format

**Обоснование:**
- Существующие клиенты должны продолжать работать
- Необходима документация для новых клиентов

## 4. Ограничения и зависимости

### 4.1 Технические ограничения
- **Prometheus доступность:** Метрики могут быть недоступны (graceful degradation)
- **Memory limits:** Агрегация больших объемов данных в памяти
- **Performance:** Запросы к Prometheus могут быть медленными

### 4.2 Зависимости
- **TN-033:** ClassificationService с GetStats() методом ✅ (завершена)
- **TN-021:** Prometheus metrics middleware ✅ (завершена)
- **TN-181:** Unified metrics taxonomy ✅ (завершена)
- **BusinessMetrics:** Prometheus метрики для классификации ✅ (существует)

### 4.3 Внешние зависимости
- **Prometheus:** Для расширенных метрик (опционально)
- **Redis:** Для L2 cache статистики (опционально)

## 5. Риски и митigation

### Risk-1: Производительность Prometheus запросов
**Вероятность:** MEDIUM
**Влияние:** HIGH
**Митigation:**
- Кэширование результатов на 5-10 секунд
- Использование ClassificationService.GetStats() как primary source
- Prometheus метрики как optional enhancement
- Timeout на Prometheus запросы (100ms)

### Risk-2: Несоответствие данных между источниками
**Вероятность:** LOW
**Влияние:** MEDIUM
**Митigation:**
- Четкое определение приоритетов источников данных
- Документация расхождений
- Валидация данных перед возвратом

### Risk-3: Memory overhead при агрегации
**Вероятность:** LOW
**Влияние:** LOW
**Митigation:**
- Ленивая агрегация (только при запросе)
- Ограничение размера агрегированных данных
- Использование streaming для больших объемов

## 6. Критерии приемки

### AC-1: Базовый функционал
- [ ] GET /api/v2/classification/stats возвращает 200 OK
- [ ] Response содержит все обязательные поля (FR-1)
- [ ] Response соответствует OpenAPI спецификации
- [ ] Все тесты проходят (unit + integration)

### AC-2: Производительность
- [ ] p95 latency < 50ms (без Prometheus)
- [ ] p95 latency < 200ms (с Prometheus)
- [ ] Throughput > 1000 req/s
- [ ] Zero race conditions (race detector clean)

### AC-3: Надежность
- [ ] Graceful degradation при недоступности Prometheus
- [ ] Fallback на ClassificationService при ошибках
- [ ] Thread-safe concurrent access
- [ ] Error handling для всех edge cases

### AC-4: Качество кода
- [ ] Test coverage > 85%
- [ ] Zero linter warnings
- [ ] Comprehensive documentation
- [ ] OpenAPI 3.0 spec complete

### AC-5: Интеграция
- [ ] Интегрирован в main.go router
- [ ] Prometheus метрики для endpoint
- [ ] Structured logging работает
- [ ] Middleware stack применен

## 7. Метрики успешности

### Метрика-1: Время реализации
- **Target:** 8-12 часов
- **Stretch goal:** 6-8 часов (150% quality)

### Метрика-2: Качество кода
- **Target:** Test coverage > 85%
- **Stretch goal:** Test coverage > 90%

### Метрика-3: Производительность
- **Target:** p95 < 50ms
- **Stretch goal:** p95 < 30ms (150% quality)

### Метрика-4: Документация
- **Target:** 500+ LOC документации
- **Stretch goal:** 1000+ LOC (150% quality)

## 8. Приоритизация

**Приоритет:** HIGH
**Обоснование:**
- Критичен для мониторинга системы классификации
- Блокирует другие задачи (TN-72, TN-73)
- Требуется для production readiness

**Зависимости:**
- Блокирует: TN-72 (POST /classification/classify - manual classification)
- Блокирует: TN-73 (GET /classification/models - available models)
