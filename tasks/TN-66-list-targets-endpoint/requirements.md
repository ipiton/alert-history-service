# TN-66: GET /publishing/targets - List Targets Endpoint

**Дата создания:** 2025-11-16
**Статус:** В разработке
**Приоритет:** Высокий
**Целевой показатель качества:** 150%
**Версия:** 1.0

---

## 1. Обоснование задачи

### 1.1 Бизнес-контекст

Endpoint `GET /publishing/targets` является критически важным компонентом системы управления publishing targets в enterprise-среде. Он обеспечивает:

- **Visibility**: Полная видимость всех настроенных publishing targets (Rootly, PagerDuty, Slack, Webhook)
- **Operational Management**: Возможность мониторинга и управления конфигурацией targets через API
- **Integration Support**: Основа для интеграции с внешними системами управления инцидентами
- **Troubleshooting**: Быстрая диагностика проблем с конфигурацией targets
- **Audit & Compliance**: Отслеживание изменений в конфигурации targets для compliance

### 1.2 Технический контекст

Текущая реализация `GET /publishing/targets` существует, но требует комплексного улучшения для соответствия enterprise-стандартам:

- ✅ Базовый endpoint реализован в `go-app/internal/api/handlers/publishing/handlers.go`
- ✅ Интеграция с `TargetDiscoveryManager` для получения списка targets
- ✅ Базовая структура ответа `TargetResponse`
- ⚠️ Отсутствие фильтрации (по типу, статусу enabled)
- ⚠️ Отсутствие пагинации (limit/offset)
- ⚠️ Отсутствие сортировки
- ⚠️ Недостаточное тестирование endpoint'а
- ⚠️ Отсутствие валидации входных параметров
- ⚠️ Нет оптимизации производительности для высоких нагрузок
- ⚠️ Неполная обработка ошибок
- ⚠️ Отсутствие middleware stack (rate limiting, auth, metrics, logging)
- ⚠️ Неполная документация по использованию

### 1.3 Зависимости

**Блокирует:**
- Мониторинг publishing targets в production
- Интеграция с внешними системами управления
- SLA compliance для enterprise-клиентов
- Операционное управление targets через API

**Зависит от:**
- TN-059: Publishing API Design (✅ Завершена)
- `internal/business/publishing/discovery.go` - TargetDiscoveryManager interface
- `internal/business/publishing/discovery_impl.go` - DefaultTargetDiscoveryManager
- `internal/core/interfaces.go` - PublishingTarget struct
- `internal/api/middleware` - Middleware stack (recovery, logging, metrics, rate limit, auth)

**Связанные задачи:**
- TN-67: POST /publishing/targets/refresh
- TN-68: GET /publishing/mode
- TN-69: GET /publishing/stats
- TN-70: POST /publishing/test/{target}

---

## 2. Пользовательский сценарий

### 2.1 Основной сценарий: Получение списка всех targets

**Актор:** SRE Engineer / API Client
**Цель:** Получить список всех настроенных publishing targets

**Шаги:**
1. Клиент выполняет HTTP GET запрос на `/api/v2/publishing/targets`
2. Сервер возвращает список всех targets с их конфигурацией
3. Клиент анализирует список для операционных задач

**Ожидаемый результат:**
- HTTP 200 OK
- Content-Type: `application/json`
- Массив объектов `TargetResponse` с полной информацией
- Время ответа < 10ms (p95) для 20 targets

### 2.2 Сценарий: Фильтрация по типу

**Актор:** API Client
**Цель:** Получить только Slack targets

**Шаги:**
1. Клиент выполняет `GET /api/v2/publishing/targets?type=slack`
2. Сервер фильтрует targets по типу
3. Возвращает только Slack targets

**Ожидаемый результат:**
- HTTP 200 OK
- Только targets с `type=slack`
- Время ответа < 5ms (p95)

### 2.3 Сценарий: Фильтрация по статусу enabled

**Актор:** API Client
**Цель:** Получить только активные targets

**Шаги:**
1. Клиент выполняет `GET /api/v2/publishing/targets?enabled=true`
2. Сервер фильтрует targets по статусу enabled
3. Возвращает только активные targets

**Ожидаемый результат:**
- HTTP 200 OK
- Только targets с `enabled=true`
- Время ответа < 5ms (p95)

### 2.4 Сценарий: Пагинация для большого количества targets

**Актор:** API Client
**Цель:** Получить targets с пагинацией

**Шаги:**
1. Клиент выполняет `GET /api/v2/publishing/targets?limit=10&offset=0`
2. Сервер возвращает первые 10 targets
3. Клиент запрашивает следующую страницу: `?limit=10&offset=10`

**Ожидаемый результат:**
- HTTP 200 OK
- Массив из максимум 10 targets
- Метаданные пагинации в ответе (total, limit, offset, has_more)
- Время ответа < 10ms (p95)

### 2.5 Сценарий: Комбинированная фильтрация

**Актор:** API Client
**Цель:** Получить активные Rootly targets с пагинацией

**Шаги:**
1. Клиент выполняет `GET /api/v2/publishing/targets?type=rootly&enabled=true&limit=5&offset=0`
2. Сервер применяет все фильтры
3. Возвращает результат с пагинацией

**Ожидаемый результат:**
- HTTP 200 OK
- Только активные Rootly targets
- Максимум 5 результатов
- Метаданные пагинации

### 2.6 Сценарий: Высокая нагрузка

**Актор:** Monitoring System
**Цель:** Обеспечить стабильность при высокой нагрузке

**Шаги:**
1. Мониторинг выполняет запросы каждые 5 секунд
2. Одновременно 10+ клиентов запрашивают список targets
3. Система обрабатывает бизнес-логику (webhooks, alerts)

**Ожидаемый результат:**
- Endpoint не влияет на производительность основного сервиса
- Время ответа стабильно < 20ms (p99)
- Нет memory leaks
- Rate limiting предотвращает злоупотребление

---

## 3. Ограничения и требования

### 3.1 Функциональные требования

**FR-1: Базовый список targets**
- Endpoint должен возвращать все доступные publishing targets
- Каждый target должен содержать: name, type, url, enabled, format, headers

**FR-2: Фильтрация по типу**
- Поддержка фильтрации по типу: `?type=rootly|pagerduty|slack|webhook`
- Валидация типа (enum validation)
- Case-insensitive сравнение

**FR-3: Фильтрация по статусу enabled**
- Поддержка фильтрации: `?enabled=true|false`
- Boolean parsing с валидацией

**FR-4: Пагинация**
- Поддержка `limit` (1-1000, default: 100)
- Поддержка `offset` (>=0, default: 0)
- Валидация параметров пагинации

**FR-5: Сортировка**
- Поддержка сортировки по полям: name, type, enabled (default: name)
- Поддержка направления: asc, desc (default: asc)

**FR-6: Метаданные ответа**
- Общее количество targets (total)
- Количество в текущей странице (count)
- Информация о пагинации (limit, offset, has_more)

### 3.2 Нефункциональные требования

**NFR-1: Производительность**
- P50: < 5ms для 20 targets
- P95: < 10ms для 20 targets
- P99: < 20ms для 20 targets
- Throughput: > 1000 req/s

**NFR-2: Масштабируемость**
- Поддержка до 1000 targets без деградации производительности
- Эффективная фильтрация (O(n) или лучше)
- Оптимизация памяти (избегать копирования больших структур)

**NFR-3: Надежность**
- Graceful degradation при ошибках discovery manager
- Валидация всех входных параметров
- Обработка edge cases (пустой список, невалидные фильтры)

**NFR-4: Безопасность**
- Rate limiting: 100 req/min per IP
- Input validation (SQL injection prevention, XSS prevention)
- Security headers (CORS, CSP, etc.)
- Аудит доступа (логирование запросов)

**NFR-5: Observability**
- Prometheus metrics (requests_total, request_duration_seconds, errors_total)
- Structured logging (request ID, filters, response size)
- Tracing support (OpenTelemetry)

**NFR-6: Совместимость**
- Обратная совместимость с существующими клиентами
- API versioning (v2)
- OpenAPI 3.0 спецификация

### 3.3 Технические ограничения

**TC-1: Платформа**
- Go 1.21+
- HTTP/1.1 и HTTP/2
- Kubernetes environment

**TC-2: Зависимости**
- `TargetDiscoveryManager` должен быть thread-safe
- Использование существующего middleware stack
- Интеграция с существующей системой метрик

**TC-3: Производительность**
- Минимальное использование памяти
- Избегать блокирующих операций
- Кэширование результатов (если применимо)

---

## 4. Внешние зависимости

### 4.1 Системные зависимости

- **Kubernetes API**: Для discovery targets из secrets (через TargetDiscoveryManager)
- **PostgreSQL**: Не требуется (targets хранятся в K8s secrets)
- **Redis**: Не требуется (in-memory cache в discovery manager)

### 4.2 Внутренние зависимости

- **TargetDiscoveryManager**: `internal/business/publishing/discovery.go`
- **Middleware Stack**: `internal/api/middleware`
- **Metrics Registry**: `pkg/metrics`
- **Error Handling**: `internal/api/errors`
- **Logging**: `log/slog`

### 4.3 API зависимости

- **OpenAPI Spec**: Должен быть обновлен с новыми параметрами
- **API Versioning**: v2 (`/api/v2/publishing/targets`)

---

## 5. Критерии приемки

### 5.1 Функциональные критерии

- [ ] Endpoint возвращает список всех targets при запросе без фильтров
- [ ] Фильтрация по типу работает корректно для всех типов
- [ ] Фильтрация по enabled работает корректно
- [ ] Пагинация работает корректно (limit, offset)
- [ ] Сортировка работает корректно (name, type, enabled)
- [ ] Метаданные пагинации присутствуют в ответе
- [ ] Валидация входных параметров работает корректно
- [ ] Обработка ошибок работает корректно (400, 500)

### 5.2 Нефункциональные критерии

- [ ] Производительность соответствует требованиям (P95 < 10ms)
- [ ] Rate limiting работает корректно
- [ ] Prometheus metrics собираются корректно
- [ ] Structured logging работает корректно
- [ ] Security headers присутствуют в ответе
- [ ] OpenAPI спецификация обновлена
- [ ] Тесты покрывают > 90% кода
- [ ] Документация полная и актуальная

### 5.3 Критерии качества 150%

- [ ] Производительность превышает базовые требования на 50%+
- [ ] Расширенное тестирование (unit, integration, load tests)
- [ ] Улучшенная обработка ошибок (детализированные сообщения)
- [ ] Детализированная документация (API guide, troubleshooting)
- [ ] Внедрение передовых практик (SOLID, DRY, 12-factor)
- [ ] Мониторинг и алертинг настроены
- [ ] Security audit пройден (OWASP Top 10)

---

## 6. Риски и митигация

### 6.1 Технические риски

**Риск 1: Производительность при большом количестве targets**
- **Вероятность:** Средняя
- **Влияние:** Высокое
- **Митигация:** Оптимизация фильтрации, кэширование результатов, пагинация

**Риск 2: Race conditions при concurrent access**
- **Вероятность:** Низкая
- **Влияние:** Среднее
- **Митигация:** Использование thread-safe TargetDiscoveryManager, правильная синхронизация

**Риск 3: Несовместимость с существующими клиентами**
- **Вероятность:** Низкая
- **Влияние:** Среднее
- **Митигация:** Обратная совместимость, версионирование API, постепенный rollout

### 6.2 Операционные риски

**Риск 4: Высокая нагрузка на discovery manager**
- **Вероятность:** Средняя
- **Влияние:** Среднее
- **Митигация:** Rate limiting, кэширование, мониторинг производительности

**Риск 5: Недостаточное тестирование**
- **Вероятность:** Низкая
- **Влияние:** Высокое
- **Митигация:** Comprehensive test suite, code coverage > 90%, load testing

---

## 7. Временные рамки

### 7.1 Оценка времени

- **Phase 0: Analysis & Design** - 2 часа
- **Phase 1: Requirements & Design** - 4 часа
- **Phase 2: Git Branch Setup** - 0.5 часа
- **Phase 3: Core Implementation** - 8 часов
- **Phase 4: Testing** - 6 часов
- **Phase 5: Performance Optimization** - 4 часа
- **Phase 6: Security Hardening** - 3 часа
- **Phase 7: Observability** - 3 часа
- **Phase 8: Documentation** - 4 часа
- **Phase 9: Certification & Review** - 2 часа

**Итого:** ~36.5 часов (4.5 рабочих дня)

### 7.2 Milestones

- **M1:** Requirements & Design готовы (Day 1)
- **M2:** Core implementation завершена (Day 2)
- **M3:** Testing завершено (Day 3)
- **M4:** Performance & Security готовы (Day 4)
- **M5:** Documentation & Certification (Day 5)

---

## 8. Ресурсное обеспечение

### 8.1 Команда

- **Backend Developer** (1 FTE) - основная разработка
- **QA Engineer** (0.5 FTE) - тестирование
- **DevOps Engineer** (0.25 FTE) - мониторинг и алертинг
- **Technical Writer** (0.25 FTE) - документация

### 8.2 Инфраструктура

- **Development Environment**: Локальная разработка
- **Testing Environment**: Kubernetes cluster для интеграционных тестов
- **Performance Testing**: k6 для load testing
- **Monitoring**: Prometheus + Grafana для observability

---

## 9. Метрики успешности

### 9.1 Технические метрики

- **Code Coverage**: > 90%
- **Performance**: P95 < 10ms (target: < 5ms для 150%)
- **Error Rate**: < 0.1%
- **Availability**: > 99.9%

### 9.2 Качественные метрики

- **Security Score**: OWASP Top 10 compliance 100%
- **Documentation Quality**: Complete API guide + troubleshooting
- **Code Quality**: Linter passes, no critical issues
- **Test Quality**: All edge cases covered

---

## 10. Примечания

- Endpoint должен быть публичным (без authentication) для операционной видимости
- Rate limiting обязателен для предотвращения злоупотребления
- Все изменения должны быть обратно совместимыми
- Документация должна включать примеры использования для всех сценариев
