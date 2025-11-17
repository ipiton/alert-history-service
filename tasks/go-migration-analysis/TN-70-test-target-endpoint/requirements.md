# TN-70: POST /publishing/test/{target} - Test Target Endpoint

## 1. Обоснование задачи

### 1.1 Бизнес-контекст

Endpoint для тестирования publishing targets является критически важным инструментом для операторов и DevOps инженеров, позволяющим:

1. **Валидация конфигурации**: Проверка корректности настройки publishing targets перед production deployment
2. **Диагностика проблем**: Быстрое выявление проблем с connectivity, authentication, или форматированием
3. **Мониторинг здоровья**: Регулярная проверка работоспособности targets в production
4. **CI/CD интеграция**: Автоматическая валидация targets в deployment pipeline

### 1.2 Техническая необходимость

- **Отсутствие интеграции**: Endpoint существует в коде, но не интегрирован в router (используется PlaceholderHandler)
- **Ограниченная функциональность**: Текущая реализация не включает timeout, response_time_ms, status_code (как в Python версии)
- **Недостаточная observability**: Отсутствуют Prometheus метрики для тестовых операций
- **Неполная валидация**: Нет валидации входных данных (alert_name, custom alert payload)
- **Отсутствие тестов**: Нет unit/integration тестов для Go реализации

### 1.3 Связь с другими задачами

- **TN-46**: K8s Client - используется для discovery targets
- **TN-47**: Target Discovery Manager - используется для получения target по имени
- **TN-51**: Alert Formatter - используется для форматирования test alert
- **TN-52-55**: Publishers (Rootly, PagerDuty, Slack, Webhook) - используются для публикации
- **TN-56**: Publishing Queue - используется для асинхронной публикации
- **TN-58**: Parallel Publishing - может использоваться для тестирования
- **TN-59**: Publishing API - часть общего Publishing API
- **TN-66**: List Targets - связанный endpoint для получения списка targets

## 2. Пользовательские сценарии

### 2.1 Сценарий 1: Валидация нового target перед production

**Пользователь**: DevOps инженер
**Цель**: Проверить корректность настройки нового Rootly target

**Шаги**:
1. Создает K8s Secret с конфигурацией Rootly target
2. Выполняет `POST /api/v2/publishing/targets/rootly-prod/test`
3. Получает успешный ответ с response_time_ms и status_code
4. Проверяет в Rootly UI, что test alert был получен

**Ожидаемый результат**:
- HTTP 200 OK
- `success: true`
- `response_time_ms: < 1000`
- `status_code: 200` (от Rootly API)

### 2.2 Сценарий 2: Диагностика проблем с connectivity

**Пользователь**: SRE инженер
**Цель**: Выяснить причину failures в production

**Шаги**:
1. Замечает failures в метриках publishing
2. Выполняет `POST /api/v2/publishing/targets/pagerduty-oncall/test` с custom alert
3. Получает ошибку с детальным сообщением
4. Исправляет проблему (например, неверный routing_key)

**Ожидаемый результат**:
- HTTP 200 OK (test выполнен)
- `success: false`
- `error: "authentication failed: invalid routing key"`
- `response_time_ms: 150` (быстрый fail)

### 2.3 Сценарий 3: Автоматическая проверка в CI/CD

**Пользователь**: CI/CD pipeline
**Цель**: Валидировать targets перед deployment

**Шаги**:
1. После создания/обновления K8s Secret
2. CI/CD выполняет `POST /api/v2/publishing/targets/{target}/test` для всех targets
3. Проверяет `success: true` для всех targets
4. Блокирует deployment при failures

**Ожидаемый результат**:
- HTTP 200 OK для всех targets
- `success: true` для всех
- Детальные логи для debugging

### 2.4 Сценарий 4: Мониторинг здоровья targets

**Пользователь**: Monitoring system
**Цель**: Регулярная проверка работоспособности

**Шаги**:
1. Cron job выполняет `POST /api/v2/publishing/targets/{target}/test` каждые 5 минут
2. Анализирует метрики `publishing_test_duration_seconds` и `publishing_test_errors_total`
3. Создает alert при failures > 3 подряд

**Ожидаемый результат**:
- Метрики Prometheus для мониторинга
- Structured logging для анализа
- Низкая latency (< 500ms p95)

## 3. Функциональные требования (FR)

### FR-1: Endpoint должен принимать POST запрос на `/api/v2/publishing/targets/{target}/test`

**Приоритет**: HIGH
**Критерии приёмки**:
- Endpoint зарегистрирован в router
- Path parameter `target` извлекается корректно
- Поддерживается HTTP POST метод
- Content-Type: application/json

### FR-2: Endpoint должен поддерживать опциональный request body

**Приоритет**: MEDIUM
**Критерии приёмки**:
- Request body опционален (может быть пустым)
- Поддерживается `alert_name` для кастомизации test alert
- Поддерживается `test_alert` для полного custom alert payload
- Поддерживается `timeout_seconds` для настройки timeout (default: 30s)

**Request Schema**:
```json
{
  "alert_name": "string (optional, default: 'TestAlert')",
  "test_alert": {
    "fingerprint": "string (optional)",
    "labels": {"key": "value"},
    "annotations": {"key": "value"},
    "status": "firing|resolved (optional, default: 'firing')"
  },
  "timeout_seconds": "integer (optional, default: 30, min: 1, max: 300)"
}
```

### FR-3: Endpoint должен валидировать target existence и enabled status

**Приоритет**: HIGH
**Критерии приёмки**:
- Возвращает HTTP 404 если target не найден
- Возвращает HTTP 200 с `success: false` если target disabled
- Использует TargetDiscoveryManager для получения target

### FR-4: Endpoint должен создавать test alert

**Приоритет**: HIGH
**Критерии приёмки**:
- Создает EnrichedAlert с test метками
- Использует `alert_name` из request или default "TestAlert"
- Добавляет labels: `test: "true"`, `severity: "info"`
- Добавляет annotations: `summary`, `description`
- Генерирует уникальный fingerprint

### FR-5: Endpoint должен публиковать test alert в target

**Приоритет**: HIGH
**Критерии приёмки**:
- Использует PublishingCoordinator.PublishToTargets()
- Поддерживает context timeout из request
- Обрабатывает errors от coordinator
- Возвращает результат публикации

### FR-6: Endpoint должен возвращать детальный response

**Приоритет**: HIGH
**Критерии приёмки**:
- Возвращает HTTP 200 OK (даже при failure)
- Response включает: `success`, `message`, `error` (optional), `target_name`, `status_code` (optional), `response_time_ms`, `test_timestamp`
- Response time измеряется от начала до конца операции

**Response Schema**:
```json
{
  "success": "boolean",
  "message": "string",
  "target_name": "string",
  "status_code": "integer (optional, HTTP status от target API)",
  "response_time_ms": "integer",
  "error": "string (optional, только при success: false)",
  "test_timestamp": "string (ISO 8601)"
}
```

### FR-7: Endpoint должен поддерживать custom alert payload

**Приоритет**: MEDIUM
**Критерии приёмки**:
- Если `test_alert` предоставлен, использует его вместо default
- Валидирует структуру test_alert
- Сохраняет test метки для идентификации

### FR-8: Endpoint должен обрабатывать timeout

**Приоритет**: MEDIUM
**Критерии приёмки**:
- Применяет timeout из request или default 30s
- Возвращает error при timeout
- Измеряет response_time_ms даже при timeout

## 4. Нефункциональные требования (NFR)

### NFR-1: Производительность

**Требования**:
- P50 latency: < 100ms (для синхронной публикации)
- P95 latency: < 500ms
- P99 latency: < 1s
- Throughput: > 100 req/s (для тестирования)

**Обоснование**: Test endpoint должен быть быстрым для использования в CI/CD и мониторинге

### NFR-2: Надежность

**Требования**:
- Availability: 99.9% (вместе с основным сервисом)
- Error rate: < 1% (не считая failures самого target)
- Graceful degradation при недоступности target

**Обоснование**: Test endpoint не должен ломать основной функционал

### NFR-3: Безопасность

**Требования**:
- Authentication: Требуется для операторов (Operator+ role)
- Rate limiting: 10 req/min per IP (предотвращение abuse)
- Input validation: Валидация всех входных данных
- Error sanitization: Не раскрывать sensitive данные в errors

**Обоснование**: Предотвращение abuse и защита от DoS

### NFR-4: Observability

**Требования**:
- Prometheus metrics: `publishing_test_requests_total`, `publishing_test_duration_seconds`, `publishing_test_errors_total`
- Structured logging: Request ID, target name, success/failure, duration
- Distributed tracing: OpenTelemetry spans для debugging

**Обоснование**: Мониторинг и диагностика проблем

### NFR-5: Тестируемость

**Требования**:
- Unit tests: > 90% coverage для handler
- Integration tests: E2E тесты с mock publishers
- Benchmarks: Performance benchmarks для оптимизации

**Обоснование**: Гарантия качества и предотвращение регрессий

### NFR-6: Документация

**Требования**:
- OpenAPI 3.0 spec: Полная спецификация endpoint
- API Guide: Примеры использования (curl, Go, Python)
- Troubleshooting guide: Типичные проблемы и решения

**Обоснование**: Упрощение использования для операторов

## 5. Ограничения

### 5.1 Технические ограничения

- **Синхронная публикация**: Test endpoint использует синхронную публикацию (не через queue) для получения immediate feedback
- **Timeout**: Максимальный timeout 300s (5 минут) для предотвращения hanging requests
- **Rate limiting**: Ограничение 10 req/min per IP для предотвращения abuse
- **Target availability**: Endpoint зависит от доступности target (может быть недоступен)

### 5.2 Бизнес-ограничения

- **Operator role required**: Только операторы могут выполнять тесты (не public endpoint)
- **No production alerts**: Test alerts должны быть помечены как test (не создавать incidents в production)
- **Resource usage**: Тесты потребляют ресурсы (CPU, network), должны быть ограничены

## 6. Внешние зависимости

### 6.1 Внутренние зависимости

- **TN-46**: K8s Client ✅ (завершена)
- **TN-47**: Target Discovery Manager ✅ (завершена)
- **TN-51**: Alert Formatter ✅ (завершена)
- **TN-52-55**: Publishers ✅ (завершены)
- **TN-56**: Publishing Queue ✅ (завершена)
- **TN-58**: Parallel Publishing ✅ (завершена, опционально)
- **TN-59**: Publishing API ✅ (завершена, часть общего API)

### 6.2 Внешние зависимости

- **Target APIs**: Rootly, PagerDuty, Slack, Webhook APIs должны быть доступны
- **K8s Cluster**: Для discovery targets через K8s Secrets
- **Redis**: Для queue (если используется асинхронная публикация)

## 7. Риски и митigation

### 7.1 Технические риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Target недоступен | HIGH | MEDIUM | Graceful error handling, timeout |
| Timeout слишком короткий | MEDIUM | LOW | Настраиваемый timeout, default 30s |
| Rate limiting слишком строгий | MEDIUM | LOW | Настраиваемый rate limit |
| Memory leak при большом количестве тестов | LOW | HIGH | Context cancellation, proper cleanup |

### 7.2 Бизнес-риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Abuse endpoint для DoS | MEDIUM | HIGH | Rate limiting, authentication |
| Test alerts создают incidents | LOW | HIGH | Явная маркировка test alerts |
| Недостаточная observability | MEDIUM | MEDIUM | Comprehensive metrics и logging |

## 8. Критерии приёмки

### 8.1 Функциональные критерии

- [ ] Endpoint зарегистрирован в router на `/api/v2/publishing/targets/{target}/test`
- [ ] Поддерживается POST метод с опциональным request body
- [ ] Валидируется existence и enabled status target
- [ ] Создается test alert с правильными метками
- [ ] Публикуется alert в target через coordinator
- [ ] Возвращается детальный response с success/error
- [ ] Поддерживается custom alert payload
- [ ] Обрабатывается timeout корректно

### 8.2 Нефункциональные критерии

- [ ] P95 latency < 500ms
- [ ] Prometheus metrics интегрированы
- [ ] Structured logging с request ID
- [ ] Unit tests > 90% coverage
- [ ] Integration tests проходят
- [ ] OpenAPI 3.0 spec создана
- [ ] API Guide создан
- [ ] Security: Authentication + Rate limiting

### 8.3 Критерии качества (150% target)

- [ ] Performance: P95 < 250ms (2x лучше target)
- [ ] Test coverage: > 95% (превышает 90% target)
- [ ] Documentation: > 2000 LOC (comprehensive)
- [ ] Benchmarks: Все операции < 100µs
- [ ] Error handling: Все edge cases покрыты
- [ ] Observability: 5+ Prometheus metrics

## 9. Приоритеты

**Приоритет задачи**: HIGH
**Приоритет в фазе**: HIGH (критично для операций)
**Блокирует**: Ничего
**Блокируется**: Ничем (все зависимости завершены)

## 10. Оценка времени

**Базовая оценка**: 8 часов
**С учетом 150% качества**: 12 часов
**Breakdown**:
- Analysis & Documentation: 2 часа
- Implementation: 4 часа
- Testing: 3 часа
- Documentation: 2 часа
- Integration: 1 час

---

**Дата создания**: 2025-01-17
**Автор**: AI Assistant
**Статус**: Draft → Ready for Design
