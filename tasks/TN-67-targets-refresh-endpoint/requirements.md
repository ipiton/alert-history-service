# TN-67: POST /publishing/targets/refresh - Refresh Discovery

## 🎯 Цель задачи

Реализовать **Enterprise-grade** endpoint для ручного запуска обновления (refresh) списка publishing targets из Kubernetes Secrets с уровнем качества **150%** от базовых требований.

## 🔍 Обоснование задачи

### Проблема

1. **Endpoint НЕ подключен**: Handler `HandleRefreshTargets` существует, но в роутере используется `PlaceholderHandler` → endpoint не работает
2. **Отсутствует полноценное тестирование**: Нет unit/integration/performance тестов
3. **Нет security hardening**: Отсутствует rate limiting, input validation, security headers
4. **Недостаточная observability**: Нет детальных метрик, трейсинга, structured logging
5. **Минимальная документация**: Нет OpenAPI spec, API guide, runbooks

### Зачем делаем

**Бизнес-ценность:**
- Администраторы могут **немедленно** применить изменения в K8s secrets без ожидания автоматического refresh (5 минут)
- Критично для **incident response**: быстрое переключение targets при проблемах с publishing
- Обязательно для **CI/CD pipelines**: автоматизация deployment targets после обновления конфигурации

**Технические преимущества:**
- **Async execution**: endpoint возвращает 202 Accepted мгновенно, refresh выполняется в background
- **Idempotency**: повторные вызовы безопасны (503 если уже running)
- **Rate limiting**: защита от abuse (max 1 refresh/minute)
- **Observability**: полное логирование и метрики для troubleshooting

## 👥 Пользовательские сценарии

### Сценарий 1: Экстренное переключение targets (P0)

**Актор:** DevOps Engineer (Admin role)

**Контекст:** Production incident - Rootly target недоступен, нужно переключиться на backup Slack

**Шаги:**
1. DevOps обновляет K8s secret `rootly-prod` (меняет URL на backup)
2. **Проблема:** Автоматический refresh через 5 минут - слишком долго!
3. **Решение:** DevOps вызывает `POST /api/v2/publishing/targets/refresh`
4. Endpoint возвращает `202 Accepted` с `request_id` для tracking
5. Background worker выполняет refresh (~2s)
6. Новый target active, alerts публикуются на backup

**Ожидаемый результат:**
- Refresh выполнен за **<5 секунд** (vs 5 минут автоматического)
- **Zero downtime**: alerts продолжают публиковаться во время refresh
- **Request tracking**: `request_id` для correlation с логами

### Сценарий 2: Автоматизация в CI/CD (P1)

**Актор:** CI/CD Pipeline (Service Account with Admin role)

**Контекст:** Terraform применил обновление targets в K8s, нужно активировать изменения

**Шаги:**
```bash
# Terraform apply
terraform apply -auto-approve

# Trigger refresh via API
curl -X POST https://alert-history.prod/api/v2/publishing/targets/refresh \
  -H "Authorization: Bearer $SERVICE_TOKEN" \
  -H "Content-Type: application/json"

# Wait for completion (optional)
sleep 5

# Verify targets updated
curl https://alert-history.prod/api/v2/publishing/targets | jq '.data | length'
```

**Ожидаемый результат:**
- **Declarative infrastructure**: targets обновляются автоматически после Terraform
- **No manual intervention**: DevOps не нужно заходить в UI
- **Idempotent**: можно вызывать в retry loops безопасно

### Сценарий 3: Rate limiting защита (Security)

**Актор:** Malicious User (получил скомпрометированный token)

**Контекст:** Попытка DDoS атаки через частые refresh requests

**Шаги:**
1. Attacker вызывает endpoint 100 раз в секунду
2. **Первый запрос**: `202 Accepted` - refresh запущен
3. **Запросы 2-100 в течение минуты**: `429 Too Many Requests`
4. **Через 60 секунд**: rate limit reset, следующий запрос успешен

**Ожидаемый результат:**
- **Service доступен**: rate limiting блокирует abuse
- **K8s API защищен**: не более 1 discovery request в минуту
- **Metrics фиксируют**: `publishing_refresh_rate_limit_exceeded_total` для alerting

## 📐 Требования

### Функциональные требования (FR)

#### FR-1: Async Refresh Trigger
- **FR-1.1**: Endpoint принимает `POST /api/v2/publishing/targets/refresh` (no body required)
- **FR-1.2**: Возвращает `202 Accepted` немедленно (async behavior)
- **FR-1.3**: Генерирует уникальный `request_id` (UUID) для tracking
- **FR-1.4**: Запускает refresh в отдельной goroutine (non-blocking)

#### FR-2: Error Handling
- **FR-2.1**: Возвращает `503 Service Unavailable` если refresh уже running
  - Response body: `{"error": "refresh_in_progress", "message": "...", "started_at": "2025-11-17T10:30:00Z"}`
- **FR-2.2**: Возвращает `429 Too Many Requests` при превышении rate limit
  - Response body: `{"error": "rate_limit_exceeded", "message": "Max 1 refresh per minute", "retry_after_seconds": 45}`
- **FR-2.3**: Возвращает `503 Service Unavailable` если RefreshManager не стартован
- **FR-2.4**: Возвращает `500 Internal Server Error` при неожиданных ошибках

#### FR-3: Rate Limiting
- **FR-3.1**: Максимум **1 manual refresh в минуту**
- **FR-3.2**: Rate limit НЕ применяется к автоматическим periodic refreshes
- **FR-3.3**: Rate limit сбрасывается через 60 секунд после успешного запуска

#### FR-4: Integration с RefreshManager
- **FR-4.1**: Использует `RefreshManager.RefreshNow()` для запуска
- **FR-4.2**: Обрабатывает typed errors: `ErrRefreshInProgress`, `ErrRateLimitExceeded`, `ErrNotStarted`
- **FR-4.3**: Логирует результаты с `request_id` для correlation

### Нефункциональные требования (NFR)

#### NFR-1: Performance (150% Quality Target)
- **NFR-1.1**: P50 latency ≤ **50ms** (endpoint response, не refresh execution)
- **NFR-1.2**: P95 latency ≤ **100ms**
- **NFR-1.3**: P99 latency ≤ **200ms**
- **NFR-1.4**: Throughput ≥ **100 req/s** (для rate limit testing)
- **NFR-1.5**: Refresh execution time ≤ **2s** (K8s discovery + parsing)

#### NFR-2: Security (OWASP Top 10 Compliance)
- **NFR-2.1**: **Authentication required** (JWT token via `AuthMiddleware`)
- **NFR-2.2**: **Authorization**: Only `admin` role (via `AdminMiddleware`)
- **NFR-2.3**: **Rate limiting**: 1 req/min на IP (built-in handler logic)
- **NFR-2.4**: **Security headers**: CSP, HSTS, X-Content-Type-Options, X-Frame-Options
- **NFR-2.5**: **Input validation**: Request body должен быть пустым (reject non-empty)
- **NFR-2.6**: **Request size limit**: Max 1KB (защита от payload attacks)
- **NFR-2.7**: **Audit logging**: Все refresh attempts логируются с user_id, IP, timestamp

#### NFR-3: Observability (Enterprise-grade)
- **NFR-3.1**: **Prometheus metrics** (7 метрик):
  - `publishing_refresh_requests_total{status, trigger}` - counter (status: success/error/rate_limited/in_progress, trigger: manual/auto)
  - `publishing_refresh_duration_seconds` - histogram (execution time)
  - `publishing_refresh_errors_total{error_type}` - counter (error_type: k8s_api/parsing/validation)
  - `publishing_refresh_rate_limit_exceeded_total` - counter
  - `publishing_refresh_in_progress` - gauge (0/1)
  - `publishing_refresh_last_success_timestamp` - gauge (Unix timestamp)
  - `publishing_refresh_targets_discovered{status}` - gauge (status: total/valid/invalid)
- **NFR-3.2**: **Structured logging** (slog):
  - `INFO`: Successful refresh triggers (request_id, user_id, IP)
  - `WARN`: Rate limit exceeded, refresh in progress
  - `ERROR`: Refresh failures, K8s API errors
  - `DEBUG`: Refresh steps (discovery start/end, parsing, validation)
- **NFR-3.3**: **Request ID tracking**: Propagate `request_id` через весь refresh pipeline
- **NFR-3.4**: **Health checks**: `/healthz` включает refresh status (last_success age < 10m = healthy)

#### NFR-4: Testing (150% Coverage)
- **NFR-4.1**: **Unit tests** (≥80% coverage):
  - Handler logic: success, errors, rate limiting
  - RefreshManager integration: mock testing
  - Error handling: all error paths
- **NFR-4.2**: **Integration tests**:
  - Real K8s client (or test cluster)
  - End-to-end refresh flow
  - Concurrent requests handling
- **NFR-4.3**: **Performance benchmarks**:
  - Handler latency benchmarks
  - Refresh execution time benchmarks
  - Rate limit validation
- **NFR-4.4**: **Security tests**:
  - Unauthorized access (401)
  - Insufficient permissions (403)
  - Rate limit enforcement (429)
  - Request size limits (413)

#### NFR-5: Documentation (150% Completeness)
- **NFR-5.1**: **OpenAPI 3.0 Specification**:
  - Full endpoint definition with examples
  - All response codes documented (202, 429, 503, 500)
  - Security schemes (JWT Bearer)
- **NFR-5.2**: **API Integration Guide**:
  - cURL examples
  - Go client example
  - Python client example
  - Error handling patterns
- **NFR-5.3**: **Runbooks**:
  - Troubleshooting refresh failures
  - Rate limit debugging
  - K8s API connectivity issues
- **NFR-5.4**: **Code documentation**:
  - Handler godoc (100% coverage)
  - Architecture Decision Records (ADRs)

## 🔒 Ограничения

### Технические ограничения
1. **K8s API latency**: Discovery может занимать 1-2s при медленной K8s API
2. **Rate limit hardcoded**: 1 req/min - не configurable (by design для безопасности)
3. **Single-flight pattern**: Только 1 refresh одновременно (защита K8s API)
4. **No queuing**: Concurrent requests получают 503, не ставятся в очередь

### Внешние зависимости
1. **RefreshManager** (TN-048): Должен быть запущен (`Start()` called)
2. **TargetDiscoveryManager** (TN-047): Должен иметь доступ к K8s API
3. **K8s RBAC**: Service account должен иметь `secrets:list` permission
4. **Middleware stack**: `AuthMiddleware`, `AdminMiddleware` должны быть настроены

### Совместимость
1. **Backward compatibility**: Endpoint новый, ломающих изменений нет
2. **API versioning**: `/api/v2/publishing/targets/refresh` - v2 API
3. **Legacy v1**: Существующий endpoint `/api/v1/publishing/targets/refresh` остается (deprecated)

## 📊 Метрики успеха

### Критерии приемки (150% Quality)
- ✅ Endpoint подключен к роутеру (не Placeholder)
- ✅ Все 4 error cases обработаны (в том числе rate limit)
- ✅ Rate limiting работает (max 1 req/min)
- ✅ Performance: P95 ≤ 100ms
- ✅ Security: Auth + Admin RBAC + Security headers
- ✅ Observability: 7 Prometheus метрик + structured logging
- ✅ Testing: ≥80% coverage (unit + integration + benchmarks)
- ✅ Documentation: OpenAPI spec + API guide + runbooks
- ✅ Certification: Grade A+ (≥95/100 points)

### Quality Score Breakdown
- **Code Quality** (20 points): Clean architecture, SOLID, error handling
- **Testing** (20 points): Coverage, edge cases, benchmarks
- **Performance** (15 points): Latency targets, throughput
- **Security** (15 points): OWASP compliance, audit logging
- **Observability** (15 points): Metrics, logging, tracing
- **Documentation** (15 points): OpenAPI, guides, runbooks

**Total: 100 points → Grade A+ требует ≥95 points**

## 📅 Timeline

**Estimated effort**: 1.5 дня (12 часов)

- **Phase 0**: Analysis & Planning (1h) - ✅ COMPLETE
- **Phase 1**: Requirements & Design (1h)
- **Phase 2**: Git Branch Setup (0.5h)
- **Phase 3**: Implementation (3h)
- **Phase 4**: Testing (2h)
- **Phase 5**: Performance Optimization (1h)
- **Phase 6**: Security Hardening (1h)
- **Phase 7**: Observability (1h)
- **Phase 8**: Documentation (1.5h)
- **Phase 9**: Certification (1h)

## 🔗 Related Tasks

### Dependencies (Must Complete First)
- ✅ **TN-047**: Target Discovery Manager - **COMPLETE** (150% certified)
- ✅ **TN-048**: Target Refresh Mechanism - **COMPLETE** (150% certified)

### Blocks (Cannot Start Until This Complete)
- ❌ **TN-68**: GET /publishing/mode - требует refresh status
- ❌ **TN-69**: GET /publishing/stats - требует refresh metrics

### Related
- ✅ **TN-65**: GET /metrics - Prometheus endpoint (используется для observability)
- ✅ **TN-66**: GET /publishing/targets - List targets (используется для verification)

## 📋 Out of Scope

Следующие аспекты **НЕ входят** в TN-67:
1. **Webhook notifications**: Не отправляем уведомления о refresh completion
2. **Target validation**: Проверка URL accessibility - это responsibility TargetDiscoveryManager
3. **Target health monitoring**: Health checks targets - отдельная задача TN-049
4. **Rollback mechanism**: Откат к previous targets при failures
5. **Multi-tenancy**: Per-namespace refresh - сейчас global refresh только
