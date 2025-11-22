# TN-150: POST /api/v2/config - Update Configuration

**Date**: 2025-11-22
**Task ID**: TN-150
**Phase**: Phase 10 - Config Management
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Planning Phase

---

## 🎯 Executive Summary

**TN-150** реализует endpoint **POST /api/v2/config** для динамического обновления конфигурации приложения без перезапуска сервиса. Это критически важный компонент для управления конфигурацией в энтерпрайз-среде, обеспечивающий zero-downtime reconfiguration.

### Стратегическая ценность

1. **Zero-Downtime Updates**: Обновление конфигурации без перезапуска сервиса (99.999% uptime)
2. **Dynamic Reconfiguration**: Изменение параметров "на лету" для быстрой реакции на инциденты
3. **Operational Excellence**: Снижение MTTR (Mean Time To Recovery) при конфигурационных проблемах
4. **GitOps Integration**: Интеграция с CI/CD пайплайнами для автоматического развертывания конфигураций
5. **Audit & Compliance**: Полный аудит всех изменений конфигурации с трекингом версий
6. **Alertmanager Compatibility**: Совместимость с Alertmanager API v2 для миграции

### Бизнес-ценность

- **Снижение downtime**: ~95% (от часов до секунд при изменении конфигурации)
- **Ускорение deployment**: ~10x (от минут до секунд)
- **Снижение рисков**: Атомарное обновление с rollback на failure
- **Compliance**: Полный audit trail всех изменений

---

## 📋 Requirements Analysis

### 1. Функциональные требования (FR)

#### FR-1: Обновление конфигурации через POST запрос
- **Приоритет**: P0 (Critical)
- **Описание**: Endpoint принимает новую конфигурацию в JSON/YAML формате и применяет её атомарно
- **Acceptance Criteria**:
  - ✅ POST /api/v2/config принимает JSON body
  - ✅ POST /api/v2/config?format=yaml принимает YAML body
  - ✅ Content-Type validation (application/json, text/yaml)
  - ✅ Максимальный размер payload: 10MB (защита от DoS)
  - ✅ Response возвращает статус обновления + diff изменений
  - ✅ HTTP 200 OK при успехе, 4xx/5xx при ошибках

#### FR-2: Валидация конфигурации
- **Приоритет**: P0 (Critical)
- **Описание**: Многоуровневая валидация перед применением конфигурации
- **Acceptance Criteria**:
  - ✅ **Syntax Validation**: Корректность JSON/YAML синтаксиса
  - ✅ **Schema Validation**: Соответствие структуре Config struct
  - ✅ **Type Validation**: Правильность типов полей (int, string, duration, etc.)
  - ✅ **Range Validation**: Проверка диапазонов (ports 1-65535, positive integers, etc.)
  - ✅ **Semantic Validation**: Бизнес-правила (e.g., MaxConnections >= MinConnections)
  - ✅ **Dependency Validation**: Проверка зависимостей между полями
  - ✅ **Security Validation**: Валидация секретов, паролей, токенов (не пустые, минимальная длина)
  - ✅ **Cross-Field Validation**: Консистентность между секциями (e.g., если LLM.Enabled=true, то LLM.APIKey обязателен)
  - ✅ Детальные ошибки с указанием поля и причины (JSON schema violations)

#### FR-3: Атомарное применение конфигурации
- **Приоритет**: P0 (Critical)
- **Описание**: Конфигурация применяется атомарно - либо полностью, либо не применяется вообще
- **Acceptance Criteria**:
  - ✅ Транзакционное обновление (all-or-nothing)
  - ✅ Откат к предыдущей конфигурации при ошибке применения
  - ✅ Graceful degradation: сервис продолжает работать на старой конфигурации при ошибке
  - ✅ Backing up старой конфигурации перед применением новой
  - ✅ Integrity check: SHA256 hash конфигурации до и после

#### FR-4: Dry-Run режим
- **Приоритет**: P1 (High)
- **Описание**: Возможность проверить конфигурацию без применения (query param `?dry_run=true`)
- **Acceptance Criteria**:
  - ✅ `?dry_run=true` валидирует конфигурацию без применения
  - ✅ Response содержит результат валидации + diff preview
  - ✅ Response показывает какие компоненты будут затронуты
  - ✅ HTTP 200 OK если валидация успешна (даже если конфигурация не применена)
  - ✅ HTTP 422 Unprocessable Entity если валидация провалилась

#### FR-5: Partial Update (секционное обновление)
- **Приоритет**: P1 (High)
- **Описание**: Возможность обновить только определенные секции конфигурации через query param `?sections=server,database`
- **Acceptance Criteria**:
  - ✅ `?sections=server,redis` обновляет только указанные секции
  - ✅ Остальные секции остаются без изменений
  - ✅ Валидация только изменённых секций
  - ✅ Cross-section validation (если секции зависят друг от друга)
  - ✅ Merge strategy: deep merge, не перезаписывает незаданные поля

#### FR-6: Diff визуализация изменений
- **Приоритет**: P1 (High)
- **Описание**: Response содержит diff между старой и новой конфигурацией
- **Acceptance Criteria**:
  - ✅ JSON patch format (RFC 6902) или unified diff
  - ✅ Показывает added, modified, deleted поля
  - ✅ Скрывает секреты в diff (показывает `***REDACTED***`)
  - ✅ Highlight критичных изменений (e.g., изменение database host)

#### FR-7: Версионирование и история изменений
- **Приоритет**: P1 (High)
- **Описание**: Трекинг всех изменений конфигурации с метаданными
- **Acceptance Criteria**:
  - ✅ Каждое обновление создаёт новую версию (monotonic version counter)
  - ✅ Сохраняется timestamp, user (из auth context), source (API/GitOps/manual)
  - ✅ Сохраняется diff изменений
  - ✅ SHA256 hash новой конфигурации
  - ✅ Rollback support: GET /api/v2/config/history/{version} для восстановления

#### FR-8: Hot Reload механизм (интеграция с TN-152)
- **Приоритет**: P0 (Critical)
- **Описание**: После успешного обновления конфигурации, сигнализировать компонентам о необходимости перезагрузить конфигурацию
- **Acceptance Criteria**:
  - ✅ Trigger reload event для всех зарегистрированных компонентов
  - ✅ Компоненты подписываются на config change events
  - ✅ Graceful reload: без прерывания активных запросов
  - ✅ Parallel reload компонентов (где возможно)
  - ✅ Error handling: откат если компонент не смог применить конфигурацию
  - ✅ Timeout: 30s для reload операций

#### FR-9: Авторизация и аудит
- **Приоритет**: P0 (Critical)
- **Описание**: Только admin пользователи могут обновлять конфигурацию, все изменения логируются
- **Acceptance Criteria**:
  - ✅ Требуется admin роль (через auth middleware)
  - ✅ HTTP 403 Forbidden для non-admin пользователей
  - ✅ Audit log: кто, когда, что изменил (structured logging)
  - ✅ Rate limiting: 10 req/min per user (защита от abuse)
  - ✅ RBAC: возможность настроить permissions per секция

#### FR-10: Rollback поддержка
- **Приоритет**: P1 (High)
- **Описание**: Возможность откатить конфигурацию к предыдущей версии
- **Acceptance Criteria**:
  - ✅ POST /api/v2/config/rollback?version=N откатывает к версии N
  - ✅ Автоматический rollback при ошибке применения
  - ✅ Хранение последних N версий (default: 10, configurable)
  - ✅ Rollback валидация: проверка что старая конфигурация всё ещё валидна

---

### 2. Нефункциональные требования (NFR)

#### NFR-1: Производительность
- **Validation latency**: < 50ms p95 (для полной конфигурации)
- **Apply latency**: < 500ms p95 (включая reload компонентов)
- **Dry-run latency**: < 30ms p95 (только валидация)
- **Throughput**: > 100 updates/s (теоретически, но rate limited в production)
- **Memory overhead**: < 10MB для хранения истории конфигураций (last 10 versions)

#### NFR-2: Безопасность
- **Authentication**: Required (admin-only)
- **Authorization**: RBAC with audit logging
- **Rate Limiting**: 10 req/min per user, 100 req/min global
- **Input Validation**: Strict schema validation, sanitization
- **Secret Management**: Секреты не логируются, хранятся encrypted
- **CORS**: Configurable, strict by default
- **DoS Protection**: Max payload 10MB, timeout 30s

#### NFR-3: Надежность
- **Availability**: 99.99% (должен быть доступен даже при сбоях других компонентов)
- **Atomicity**: 100% (либо применяется полностью, либо откатывается)
- **Durability**: Конфигурация сохраняется в persistent storage (PostgreSQL или file)
- **Consistency**: Cross-component consistency через distributed transaction pattern
- **Error Recovery**: Автоматический rollback при ошибках

#### NFR-4: Observability
- **Prometheus Metrics**:
  - `config_update_requests_total` (counter, by status, format, dry_run)
  - `config_update_duration_seconds` (histogram, by phase: validation/apply/reload)
  - `config_update_errors_total` (counter, by error_type)
  - `config_validation_errors_total` (counter, by validation_type)
  - `config_reload_duration_seconds` (histogram, by component)
  - `config_version` (gauge, current version number)
  - `config_rollbacks_total` (counter, by trigger: auto/manual)
- **Structured Logging**: Все операции с request_id, user_id, version
- **Distributed Tracing**: Integration with OpenTelemetry (если доступно)
- **Audit Trail**: PostgreSQL table с полной историей изменений

#### NFR-5: Совместимость
- **Alertmanager API v2**: 100% совместимость формата POST /api/v2/config
- **OpenAPI 3.0**: Полная спецификация
- **Backward Compatibility**: Поддержка старых форматов конфигурации (deprecated fields)
- **Forward Compatibility**: Graceful handling новых полей (не падать при unknown fields)

#### NFR-6: Scalability
- **Horizontal Scaling**: Работа в кластере с несколькими репликами
- **Consistency**: Leader election для обновления конфигурации (только один нода применяет)
- **Distribution**: Распространение конфигурации на все реплики через Redis Pub/Sub или etcd
- **Lock Management**: Distributed lock для предотвращения concurrent updates

#### NFR-7: Testability
- **Unit Tests**: ≥ 85% coverage
- **Integration Tests**: ≥ 15 сценариев (success, validation errors, rollback, etc.)
- **E2E Tests**: ≥ 5 сценариев (через real HTTP requests)
- **Benchmarks**: ≥ 5 benchmarks (validation, apply, rollback)
- **Chaos Testing**: Симуляция ошибок компонентов при reload

---

## 🔍 Technical Analysis

### 3. Архитектурный дизайн

#### 3.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client (Admin User)                     │
└──────────────────────┬──────────────────────────────────────┘
                       │ POST /api/v2/config
                       │ Content-Type: application/json
                       │ Authorization: Bearer <token>
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              API Router (gorilla/mux)                        │
│         POST /api/v2/config → ConfigHandler                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│            ConfigHandler (cmd/server/handlers/)              │
│  - Auth middleware (admin-only)                             │
│  - Rate limiting middleware                                 │
│  - Request validation (size, content-type)                  │
│  - Parse body (JSON/YAML)                                   │
│  - Call ConfigUpdateService                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│        ConfigUpdateService (internal/config/update/)         │
│  Phase 1: Validation                                        │
│    - Syntax validation (JSON/YAML parser)                   │
│    - Schema validation (struct unmarshal)                   │
│    - Type validation (validator tags)                       │
│    - Business validation (Validate() method)                │
│  Phase 2: Diff Calculation                                  │
│    - Compare old vs new config                              │
│    - Generate JSON patch or unified diff                    │
│  Phase 3: Atomic Apply (if !dry_run)                       │
│    - Backup old config                                       │
│    - Write new config to storage                            │
│    - Update version counter                                 │
│  Phase 4: Hot Reload (if !dry_run)                         │
│    - Notify all registered components                       │
│    - Parallel reload with timeout                           │
│    - Collect errors                                          │
│    - Rollback if critical component failed                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ├─────────────────┬──────────────────┐
                       ▼                 ▼                  ▼
          ┌────────────────┐  ┌────────────────┐  ┌────────────────┐
          │  Component A   │  │  Component B   │  │  Component C   │
          │  (Database)    │  │  (Redis)       │  │  (LLM)         │
          │  Reload()      │  │  Reload()      │  │  Reload()      │
          └────────────────┘  └────────────────┘  └────────────────┘
```

#### 3.2 Component Responsibilities

**ConfigHandler** (HTTP Layer):
- Request validation (auth, rate limiting, size, content-type)
- Body parsing (JSON/YAML)
- Response serialization
- Error handling и HTTP status codes

**ConfigUpdateService** (Business Logic):
- Multi-phase validation pipeline
- Diff calculation
- Atomic config update
- Version management
- Hot reload orchestration

**ConfigValidator** (Validation):
- Syntax validation
- Schema validation
- Type validation
- Business rule validation
- Cross-field validation

**ConfigStorage** (Persistence):
- Save/Load config to/from PostgreSQL or file
- Version history management
- Backup/Restore operations

**ConfigReloader** (Reload Orchestration):
- Component registry (register/unregister)
- Parallel reload with timeout
- Error collection и rollback decision
- Health check после reload

**Components** (Consumers):
- Implement `Reloadable` interface
- Subscribe to config change events
- Graceful reload без прерывания запросов

#### 3.3 Data Models

```go
// UpdateConfigRequest represents POST request body
type UpdateConfigRequest struct {
    Config   map[string]interface{} `json:"config" yaml:"config"`
    Metadata UpdateMetadata          `json:"metadata,omitempty"`
}

// UpdateMetadata contains update metadata
type UpdateMetadata struct {
    Source      string `json:"source"`       // "api", "gitops", "manual"
    Description string `json:"description"`  // Change description
    Ticket      string `json:"ticket"`       // JIRA/GitHub issue
}

// UpdateConfigResponse represents response
type UpdateConfigResponse struct {
    Status  string                 `json:"status"`   // "success", "error"
    Message string                 `json:"message"`
    Version int64                  `json:"version"`  // New version number
    Diff    *ConfigDiff            `json:"diff,omitempty"`
    Errors  []ValidationError      `json:"errors,omitempty"`
}

// ConfigDiff represents changes
type ConfigDiff struct {
    Added    map[string]interface{} `json:"added"`
    Modified map[string]DiffEntry   `json:"modified"`
    Deleted  []string               `json:"deleted"`
}

// DiffEntry represents single field change
type DiffEntry struct {
    OldValue interface{} `json:"old_value"`
    NewValue interface{} `json:"new_value"`
}

// ValidationError represents validation error
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"` // "required", "invalid_type", "out_of_range"
}
```

#### 3.4 Validation Pipeline

```
Input Config (JSON/YAML)
         │
         ▼
┌─────────────────────┐
│ Phase 1: Syntax     │  ← JSON/YAML parser
│ Validation          │
└──────┬──────────────┘
       │ Pass
       ▼
┌─────────────────────┐
│ Phase 2: Schema     │  ← Unmarshal to Config struct
│ Validation          │
└──────┬──────────────┘
       │ Pass
       ▼
┌─────────────────────┐
│ Phase 3: Type       │  ← validator tags (required, min, max, etc.)
│ Validation          │
└──────┬──────────────┘
       │ Pass
       ▼
┌─────────────────────┐
│ Phase 4: Business   │  ← config.Validate() method
│ Rule Validation     │    - Port ranges
│                     │    - MinConn <= MaxConn
│                     │    - Required secrets in production
└──────┬──────────────┘
       │ Pass
       ▼
┌─────────────────────┐
│ Phase 5: Cross-     │  ← Cross-field validation
│ Field Validation    │    - If LLM.Enabled, then LLM.APIKey required
└──────┬──────────────┘
       │ Pass
       ▼
    Valid Config ✅
```

### 4. Зависимости

#### 4.1 Прямые зависимости (блокируется)
- ✅ **TN-149**: GET /api/v2/config (export) - COMPLETED
- ✅ **TN-019**: Config Loader (viper) - COMPLETED
- ✅ **TN-021**: Prometheus Metrics - COMPLETED
- ❌ **TN-151**: Config Validator (можно реализовать inline в TN-150)
- ❌ **TN-152**: Hot Reload Mechanism (можно реализовать inline в TN-150)

#### 4.2 Обратные зависимости (блокирует)
- 🎯 **TN-152**: Hot Reload (SIGHUP) - будет использовать тот же механизм применения
- 🎯 **TN-116**: API Documentation (OpenAPI) - должна включать POST /api/v2/config
- 🎯 **GitOps Integration**: Автоматическое обновление конфигурации из Git

### 5. Риски и митигации

#### Risk-1: Concurrent Updates (Race Condition)
- **Вероятность**: Medium
- **Влияние**: Critical (может привести к inconsistent state)
- **Митигация**:
  - ✅ Distributed lock (Redis-based) на время обновления
  - ✅ Optimistic locking: version check перед apply
  - ✅ HTTP 409 Conflict если concurrent update detected

#### Risk-2: Partial Reload Failure
- **Вероятность**: Medium
- **Влияние**: High (сервис может быть в inconsistent state)
- **Митигация**:
  - ✅ Critical vs non-critical components classification
  - ✅ Автоматический rollback если critical component failed
  - ✅ Graceful degradation для non-critical components
  - ✅ Health check после reload с timeout

#### Risk-3: Invalid Config в Production
- **Вероятность**: Low (благодаря validation)
- **Влияние**: Critical
- **Митигация**:
  - ✅ Строгая multi-phase validation
  - ✅ Dry-run тестирование перед apply
  - ✅ Canary deployment: apply на одну ноду сначала
  - ✅ Automatic rollback on health check failure

#### Risk-4: Performance Degradation при Reload
- **Вероятность**: Low
- **Влияние**: Medium
- **Митигация**:
  - ✅ Parallel reload компонентов
  - ✅ Timeout для reload операций (30s)
  - ✅ Graceful reload без прерывания активных запросов
  - ✅ Benchmarking и performance testing

#### Risk-5: Lost Config History
- **Вероятность**: Low
- **Влияние**: High (нет rollback)
- **Митигация**:
  - ✅ Persistent storage в PostgreSQL
  - ✅ Backup на disk (filesystem)
  - ✅ Retention policy: хранить last 10 versions минимум
  - ✅ Periodic backup в S3/external storage (опционально)

---

## 📊 Success Metrics

### Quality Metrics (150% Target)

1. **Test Coverage**: ≥ 90% (target 85%+, +5% bonus для 150%)
2. **Performance**:
   - Validation: p95 < 50ms (target < 100ms, 2x better)
   - Apply: p95 < 500ms (target < 1s, 2x better)
   - Dry-run: p95 < 30ms (target < 50ms, 1.7x better)
3. **Documentation**: ≥ 2,500 LOC (comprehensive)
4. **Code Quality**: Zero linter warnings, zero race conditions, zero security issues
5. **Reliability**: Zero failed rollbacks, 100% atomic updates

### Quantitative Metrics

1. **Production Code**: ~800-1,200 LOC
   - Handler: ~200 LOC
   - Service: ~400 LOC
   - Validator: ~200 LOC
   - Reloader: ~200 LOC
   - Models: ~200 LOC

2. **Test Code**: ~1,500-2,000 LOC
   - Unit tests: ~1,000 LOC (20+ tests)
   - Integration tests: ~600 LOC (15+ tests)
   - Benchmarks: ~400 LOC (5+ benchmarks)

3. **Documentation**: ~3,000-4,000 LOC
   - requirements.md: ~800 LOC ✅
   - design.md: ~1,200 LOC
   - tasks.md: ~600 LOC
   - README.md: ~400 LOC
   - API_GUIDE.md: ~600 LOC
   - SECURITY.md: ~400 LOC

4. **Tests**: ≥ 35 tests total
   - Unit: ≥ 20
   - Integration: ≥ 15
   - Benchmarks: ≥ 5

5. **Prometheus Metrics**: ≥ 7 metrics

### Quality Gates

- ✅ All tests pass (100% pass rate)
- ✅ Coverage ≥ 90%
- ✅ Performance targets achieved
- ✅ Zero security vulnerabilities (gosec clean)
- ✅ Zero linter warnings (golangci-lint)
- ✅ Zero race conditions (go test -race)
- ✅ Documentation complete
- ✅ OpenAPI spec complete
- ✅ Rollback mechanism tested
- ✅ Hot reload mechanism tested

---

## 🎯 Acceptance Criteria

### Must Have (P0) - Critical for MVP

- [ ] POST /api/v2/config принимает JSON конфигурацию
- [ ] POST /api/v2/config?format=yaml принимает YAML конфигурацию
- [ ] Multi-phase validation (syntax, schema, type, business, cross-field)
- [ ] Атомарное применение конфигурации (all-or-nothing)
- [ ] Automatic rollback при ошибке
- [ ] Diff visualization (added/modified/deleted fields)
- [ ] Version tracking и increment
- [ ] Hot reload mechanism для компонентов
- [ ] Admin-only authorization
- [ ] Audit logging всех изменений
- [ ] Prometheus metrics (7+ метрик)
- [ ] Structured logging с request_id
- [ ] Unit tests ≥ 20, coverage ≥ 90%
- [ ] Integration tests ≥ 15
- [ ] OpenAPI spec

### Should Have (P1) - Enhanced Functionality

- [ ] Dry-run mode (?dry_run=true)
- [ ] Partial update (?sections=server,redis)
- [ ] Config history endpoint GET /api/v2/config/history
- [ ] Manual rollback endpoint POST /api/v2/config/rollback
- [ ] Distributed lock для concurrent update protection
- [ ] Canary deployment support (apply на одну ноду)
- [ ] Rate limiting (10 req/min per user)
- [ ] Benchmarks ≥ 5
- [ ] Security documentation

### Nice to Have (P2) - Optional Enhancements

- [ ] GraphQL mutation для обновления конфигурации
- [ ] WebSocket для real-time diff preview
- [ ] Config templates поддержка
- [ ] A/B testing конфигураций
- [ ] Config drift detection (отклонение от Git source of truth)
- [ ] Slack/PagerDuty notification при критичных изменениях
- [ ] Config encryption at rest

---

## 📚 User Stories

### US-1: DevOps Engineer - Emergency Config Update
**As a** DevOps Engineer
**I want to** update LLM API key without restarting service
**So that** I can quickly respond to API key rotation incident

**Acceptance Criteria**:
- Update takes < 5 seconds end-to-end
- Zero downtime (active requests не прерываются)
- Audit log записывает кто и когда обновил

### US-2: Platform Engineer - Gradual Rollout
**As a** Platform Engineer
**I want to** test new configuration on one node first
**So that** I can verify changes before applying to all nodes

**Acceptance Criteria**:
- Dry-run mode для preview изменений
- Apply на single node с health check
- Automatic rollback если health check fails

### US-3: Security Engineer - Audit Trail
**As a** Security Engineer
**I want to** see who changed what and when in configuration
**So that** I can comply with audit requirements

**Acceptance Criteria**:
- Full audit log в PostgreSQL
- Searchable по user, timestamp, field
- Retention ≥ 90 days

---

## 📝 Notes

- **Atomicity критична**: Partial update state недопустим в production
- **Performance критична**: Validation должна быть быстрой (< 50ms)
- **Security критична**: Only admin, rate limiting, audit logging обязательны
- **Hot reload**: Должен работать gracefully без прерывания запросов
- **Compatibility**: Формат должен быть совместим с Alertmanager v2

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Review Status**: Pending
**Total Lines**: 802 LOC
