# TN-72: POST /classification/classify - Manual Classification Endpoint

## 1. Обоснование задачи

### 1.1 Бизнес-контекст
Система классификации алертов с использованием LLM является критически важным компонентом Alert History Service. Для обеспечения гибкости и возможности ручной классификации алертов операторами, аналитиками и тестовыми системами необходимо предоставить REST API endpoint для ручной классификации алертов.

### 1.2 Пользовательские сценарии

#### US-1: DevOps Engineer - Ручная классификация алерта
**Как** DevOps инженер
**Я хочу** классифицировать алерт вручную через API
**Чтобы** протестировать классификацию или переклассифицировать алерт с принудительным обновлением

**Критерии приемки:**
- [ ] POST /api/v2/classification/classify принимает JSON с алертом
- [ ] Endpoint возвращает результат классификации (severity, confidence, reasoning)
- [ ] Поддерживается параметр `force` для принудительной переклассификации (обход кэша)
- [ ] Время ответа < 500ms (p95) для cache hit, < 2s для LLM call
- [ ] Результат кэшируется для последующих запросов

#### US-2: QA Engineer - Тестирование классификации
**Как** QA инженер
**Я хочу** тестировать классификацию различных типов алертов
**Чтобы** валидировать качество работы LLM классификатора

**Критерии приемки:**
- [ ] Endpoint принимает валидные алерты с различными комбинациями labels/annotations
- [ ] Endpoint возвращает детальные ошибки валидации при некорректном запросе
- [ ] Поддерживается batch testing через множественные запросы
- [ ] Результаты логируются для анализа

#### US-3: Data Scientist - Анализ качества классификации
**Как** Data Scientist
**Я хочу** классифицировать набор тестовых алертов
**Чтобы** оценить accuracy и precision модели классификации

**Критерии приемки:**
- [ ] Endpoint возвращает confidence score для каждой классификации
- [ ] Endpoint возвращает reasoning (обоснование классификации)
- [ ] Поддерживается принудительная классификация (force=true) для свежих результатов
- [ ] Результаты включают метаданные (model, timestamp, cached)

#### US-4: Integration Testing - Автоматизированное тестирование
**Как** CI/CD система
**Я хочу** тестировать классификацию через API
**Чтобы** валидировать интеграцию с внешними системами

**Критерии приемки:**
- [ ] Endpoint поддерживает rate limiting для защиты от злоупотреблений
- [ ] Endpoint возвращает консистентные результаты
- [ ] Endpoint обрабатывает таймауты корректно
- [ ] Endpoint логирует все запросы для аудита

## 2. Функциональные требования

### FR-1: Базовый функционал классификации
**Приоритет:** HIGH
**Описание:** Endpoint должен классифицировать переданный алерт и возвращать результат

**Детали:**
- Принимает JSON запрос с полем `alert` (обязательное)
- Опциональное поле `force` (boolean) для принудительной переклассификации
- Возвращает результат классификации:
  - `severity` (critical, warning, info, noise)
  - `confidence` (0.0-1.0)
  - `reasoning` (текстовое обоснование)
  - `recommendations` (массив рекомендаций)
  - `processing_time` (время обработки в секундах)
  - `cached` (был ли результат из кэша)
  - `model` (использованная модель, если LLM)

**Источники данных:**
- ClassificationService.ClassifyAlert() - основная логика классификации
- ClassificationService.GetCachedClassification() - проверка кэша (если force=false)

### FR-2: Валидация входных данных
**Приоритет:** HIGH
**Описание:** Endpoint должен валидировать входные данные перед обработкой

**Детали:**
- Валидация структуры JSON запроса
- Валидация обязательных полей алерта:
  - `fingerprint` (обязательное, непустое)
  - `alert_name` (обязательное, непустое)
  - `status` (обязательное, одно из: firing, resolved)
  - `starts_at` (обязательное, валидная дата)
- Валидация опциональных полей:
  - `generator_url` (если указан, должен быть валидным URL)
  - `labels` (если указаны, должны быть map[string]string)
  - `annotations` (если указаны, должны быть map[string]string)
- Возврат детальных ошибок валидации (400 Bad Request)

**Валидация:**
- Использование validator/v10 для структурной валидации
- Кастомные валидаторы для бизнес-правил
- Детальные сообщения об ошибках с указанием проблемного поля

### FR-3: Кэширование результатов
**Приоритет:** HIGH
**Описание:** Endpoint должен использовать кэш для оптимизации производительности

**Детали:**
- Проверка L1 (memory) кэша перед LLM вызовом
- Проверка L2 (Redis) кэша если L1 промах
- Сохранение результата в кэш после LLM классификации
- Параметр `force=true` обходит кэш и принудительно вызывает LLM
- TTL кэша: 24 часа (настраиваемо)

**Источники данных:**
- ClassificationService.GetCachedClassification() - получение из кэша
- ClassificationService.InvalidateCache() - инвалидация кэша (при force=true)

### FR-4: Обработка ошибок
**Приоритет:** HIGH
**Описание:** Endpoint должен корректно обрабатывать все типы ошибок

**Детали:**
- **400 Bad Request**: невалидный запрос, ошибки валидации
- **429 Too Many Requests**: превышен rate limit
- **500 Internal Server Error**: ошибка классификации, недоступность LLM
- **503 Service Unavailable**: LLM сервис недоступен, circuit breaker открыт
- Детальные сообщения об ошибках с request ID для трейсинга
- Логирование всех ошибок с контекстом

**Обработка:**
- Graceful degradation: fallback на rule-based классификацию при недоступности LLM
- Circuit breaker protection через LLM client
- Таймауты: 5s для классификации (настраиваемо)

### FR-5: Метрики и observability
**Приоритет:** MEDIUM
**Описание:** Endpoint должен экспортировать метрики для мониторинга

**Детали:**
- Prometheus метрики:
  - `classification_api_requests_total{status, method}` - счетчик запросов
  - `classification_api_duration_seconds{method}` - гистограмма времени ответа
  - `classification_api_cache_hits_total` - попадания в кэш
  - `classification_api_cache_misses_total` - промахи кэша
  - `classification_api_errors_total{error_type}` - счетчик ошибок
- Structured logging с request ID
- Distributed tracing support (опционально)

**Интеграция:**
- BusinessMetrics для Prometheus метрик
- slog для structured logging
- Request ID через middleware

## 3. Нефункциональные требования

### NFR-1: Производительность
**Приоритет:** HIGH
**Описание:** Endpoint должен отвечать быстро даже при высокой нагрузке

**Требования:**
- Cache hit (L1): p95 < 5ms
- Cache hit (L2): p95 < 50ms
- Cache miss + LLM: p95 < 2s
- Fallback classification: p95 < 10ms
- Throughput: > 1000 req/s (cache hit scenario)

**Оптимизации:**
- Двухуровневое кэширование (L1 memory + L2 Redis)
- Асинхронная запись в кэш (не блокирует ответ)
- Connection pooling для LLM client
- Graceful degradation при недоступности LLM

### NFR-2: Надежность
**Приоритет:** HIGH
**Описание:** Endpoint должен быть устойчив к сбоям

**Требования:**
- Availability: 99.9% (с учетом fallback)
- Graceful degradation: fallback на rule-based классификацию
- Circuit breaker: защита от каскадных сбоев LLM
- Retry logic: экспоненциальный backoff для transient ошибок
- Timeout protection: максимальное время ожидания 5s

**Механизмы:**
- Circuit breaker через LLM client
- Fallback engine для rule-based классификации
- Health checks для зависимостей

### NFR-3: Безопасность
**Приоритет:** HIGH
**Описание:** Endpoint должен быть защищен от злоупотреблений

**Требования:**
- Rate limiting: 100 req/min per IP (настраиваемо)
- Authentication: API key или JWT (опционально, через middleware)
- Input validation: защита от injection атак
- Request size limit: максимум 100KB для тела запроса
- Audit logging: логирование всех запросов

**Защита:**
- Rate limiting middleware
- Auth middleware (если включено)
- Request size limiter middleware
- Input sanitization

### NFR-4: Масштабируемость
**Приоритет:** MEDIUM
**Описание:** Endpoint должен масштабироваться горизонтально

**Требования:**
- Stateless design: не зависит от состояния сервера
- Distributed cache: Redis для L2 кэша
- Load balancing ready: работает за балансировщиком
- Horizontal scaling: поддержка множественных инстансов

**Архитектура:**
- Stateless handlers
- Shared Redis cache
- No server-side sessions

### NFR-5: Совместимость
**Приоритет:** MEDIUM
**Описание:** Endpoint должен быть совместим с существующими системами

**Требования:**
- OpenAPI 3.0 спецификация
- JSON request/response формат
- HTTP/1.1 и HTTP/2 поддержка
- Backward compatibility: не ломает существующие интеграции

**Стандарты:**
- RESTful API design
- JSON Schema для валидации
- OpenAPI 3.0 для документации

## 4. Ограничения и зависимости

### 4.1 Технические ограничения
- **LLM Latency**: LLM вызовы могут занимать до 2s (p95)
- **Cache Capacity**: L1 кэш ограничен памятью сервера (~1000 записей)
- **Rate Limits**: LLM провайдер может иметь собственные rate limits
- **Timeout**: Максимальное время ожидания 5s (настраиваемо)

### 4.2 Зависимости
- **TN-033**: ClassificationService (✅ завершена, 150% quality)
- **TN-029**: LLM Client (✅ завершена)
- **TN-016**: Redis Cache (✅ завершена)
- **TN-021**: Prometheus Metrics (✅ завершена)
- **TN-039**: Circuit Breaker (✅ завершена)

### 4.3 Внешние зависимости
- **LLM Proxy Service**: внешний сервис для классификации
- **Redis**: для L2 кэша
- **Prometheus**: для метрик (опционально)

## 5. Риски и митигация

### 5.1 Технические риски

#### РИСК-1: Высокая латентность LLM вызовов
**Вероятность:** HIGH
**Влияние:** MEDIUM
**Митигация:**
- Двухуровневое кэширование для снижения количества LLM вызовов
- Таймауты для предотвращения долгих ожиданий
- Fallback на rule-based классификацию при недоступности LLM

#### РИСК-2: Перегрузка LLM сервиса
**Вероятность:** MEDIUM
**Влияние:** HIGH
**Митигация:**
- Rate limiting на уровне API
- Circuit breaker для защиты от каскадных сбоев
- Graceful degradation на fallback классификацию

#### РИСК-3: Проблемы с кэшем
**Вероятность:** LOW
**Влияние:** MEDIUM
**Митигация:**
- L1 memory кэш как fallback при недоступности Redis
- Graceful degradation: работает без кэша (медленнее)
- Health checks для Redis

### 5.2 Бизнес-риски

#### РИСК-4: Недостаточная точность классификации
**Вероятность:** LOW
**Влияние:** MEDIUM
**Митигация:**
- Confidence score в ответе для оценки качества
- Reasoning в ответе для понимания логики
- Возможность принудительной переклассификации (force=true)

## 6. Критерии приемки

### 6.1 Функциональные критерии
- [ ] POST /api/v2/classification/classify принимает валидный JSON с алертом
- [ ] Endpoint возвращает результат классификации со всеми обязательными полями
- [ ] Параметр `force=true` обходит кэш и принудительно вызывает LLM
- [ ] Валидация входных данных возвращает детальные ошибки (400)
- [ ] Обработка ошибок возвращает корректные HTTP статусы
- [ ] Кэширование работает корректно (L1 + L2)
- [ ] Fallback классификация работает при недоступности LLM

### 6.2 Нефункциональные критерии
- [ ] Производительность: cache hit p95 < 5ms, LLM call p95 < 2s
- [ ] Надежность: availability 99.9% с fallback
- [ ] Безопасность: rate limiting, input validation, audit logging
- [ ] Observability: Prometheus метрики, structured logging
- [ ] Документация: OpenAPI 3.0 spec, примеры использования

### 6.3 Критерии качества (150% target)
- [ ] Test coverage: > 85% (target 80%)
- [ ] Performance: превышает targets на 50%+
- [ ] Documentation: comprehensive (requirements + design + API guide)
- [ ] Security: OWASP Top 10 compliant
- [ ] Code quality: zero linter warnings, zero race conditions

## 7. Временные рамки

### 7.1 Оценка времени
- **Phase 0**: Analysis & Documentation (2 часа)
- **Phase 1**: Requirements & Design (3 часа)
- **Phase 2**: Git Branch Setup (0.5 часа)
- **Phase 3**: Core Implementation (4 часа)
- **Phase 4**: Testing (6 часов)
- **Phase 5**: Performance Optimization (2 часа)
- **Phase 6**: Security Hardening (2 часа)
- **Phase 7**: Observability Integration (2 часа)
- **Phase 8**: Documentation (3 часа)
- **Phase 9**: Certification & Validation (2 часа)

**Итого:** ~26.5 часов (базовая оценка)
**С учетом 150% качества:** ~40 часов (расширенное тестирование, документация, оптимизация)

### 7.2 Миленстоуны
- **M1**: Core Implementation Complete (Phase 3) - Day 1
- **M2**: Testing Complete (Phase 4) - Day 2
- **M3**: Production Ready (Phase 9) - Day 3-4

## 8. Ресурсное обеспечение

### 8.1 Команда
- **Backend Developer**: 1 человек (full-time)
- **QA Engineer**: 0.5 человека (тестирование)
- **DevOps**: 0.25 человека (deployment, monitoring)

### 8.2 Инфраструктура
- **Development Environment**: локальная разработка
- **Testing Environment**: доступ к Redis, LLM proxy (staging)
- **CI/CD**: GitHub Actions для автоматического тестирования

### 8.3 Инструменты
- **Go 1.24+**: язык разработки
- **validator/v10**: валидация входных данных
- **go-redis/v9**: Redis клиент
- **Prometheus**: метрики
- **slog**: structured logging

## 9. Метрики успешности

### 9.1 Технические метрики
- **Test Coverage**: > 85% (target 80%)
- **Performance**: cache hit p95 < 5ms, LLM p95 < 2s
- **Availability**: 99.9% с fallback
- **Error Rate**: < 0.1% (exclude LLM недоступность)

### 9.2 Качественные метрики
- **Code Quality**: zero linter warnings, zero race conditions
- **Documentation**: comprehensive (requirements + design + API guide + examples)
- **Security**: OWASP Top 10 compliant, security audit passed
- **Observability**: все метрики экспортируются, логи структурированы

### 9.3 Бизнес-метрики
- **Adoption**: endpoint используется в production
- **Satisfaction**: положительные отзывы от пользователей
- **Reliability**: отсутствие критических инцидентов

## 10. Принятие решения

### 10.1 Stakeholders
- **Product Owner**: утверждение требований
- **Technical Lead**: техническое ревью архитектуры
- **Security Team**: security audit
- **QA Team**: тестирование и валидация

### 10.2 Критерии Go/No-Go
- **GO**: все зависимости завершены, инфраструктура готова
- **NO-GO**: критические зависимости не завершены, недостаточно ресурсов

---

**Версия документа:** 1.0
**Дата создания:** 2025-11-17
**Автор:** AI Assistant
**Статус:** Draft → Ready for Review
