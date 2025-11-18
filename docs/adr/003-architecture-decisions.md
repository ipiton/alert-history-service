# ADR 003: Core Architecture Patterns & Principles

**Status:** Accepted
**Date:** 2025-11-18
**Task:** TN-11 (Architecture Decisions Documentation)
**Decision Maker:** Architecture Team
**Stakeholders:** All Engineering Teams

---

## Context

Alert History Service (Alertmanager++ OSS Core) requires a solid architectural foundation for:
- High scalability (100k+ alerts/minute target)
- Extensibility (LLM integration, multiple publishers)
- Maintainability (clean codebase, clear boundaries)
- Testability (unit, integration, E2E tests)
- Production reliability (graceful degradation, observability)

The architecture must support:
1. **Core Domain**: Alert processing, classification, filtering
2. **Infrastructure**: Database, cache, LLM, K8s integration
3. **Business Logic**: Grouping, inhibition, silencing, publishing
4. **API Layer**: REST endpoints, WebSocket, middleware

---

## Decision

**We adopt Clean Architecture with Hexagonal Pattern**

### Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                     API Layer (HTTP/WebSocket)           │
│  cmd/server/handlers/, internal/api/handlers/           │
│  - Gin routes, request validation, response formatting  │
└────────────┬─────────────────────────────┬──────────────┘
             │                             │
┌────────────▼─────────────────┐ ┌────────▼──────────────┐
│  Business Services Layer     │ │   Middleware Layer    │
│  internal/business/          │ │   pkg/middleware/,    │
│  - Grouping, Inhibition      │ │   internal/middleware/│
│  - Silencing, Publishing     │ │   - Auth, Metrics,    │
│  - Proxy/Routing             │ │     Rate Limiting     │
└────────────┬─────────────────┘ └───────────────────────┘
             │
┌────────────▼─────────────────────────────────────────────┐
│              Core Domain Layer (Pure Business Logic)      │
│  internal/core/                                          │
│  - Domain models (Alert, ClassificationResult, etc.)    │
│  - Core interfaces (AlertClassifier, AlertPublisher)    │
│  - Processing pipeline (AlertProcessor)                 │
└────────────┬─────────────────────────────────────────────┘
             │
┌────────────▼─────────────────────────────────────────────┐
│           Infrastructure Layer (External Dependencies)    │
│  internal/infrastructure/                                │
│  - Database (PostgreSQL repository)                     │
│  - Cache (Redis)                                        │
│  - LLM (HTTP client)                                    │
│  - K8s (client-go)                                      │
│  - Webhook (Alertmanager parser, formatters)           │
└──────────────────────────────────────────────────────────┘
```

### Key Principles

1. **Dependency Inversion**: Core domain depends on NO external libraries
2. **Interface Segregation**: Small, focused interfaces (e.g., AlertClassifier)
3. **Dependency Injection**: All dependencies injected via constructors
4. **Single Responsibility**: Each package has ONE clear purpose
5. **Repository Pattern**: Abstract database access
6. **Service Layer**: Orchestrate business logic
7. **Adapter Pattern**: Convert between layers (e.g., webhook → domain model)

---

## Architecture Patterns

### 1. Repository Pattern (Infrastructure)

```go
// internal/infrastructure/repository/postgres_history.go
type AlertHistoryRepository interface {
    GetHistory(ctx context.Context, filters AlertFilters) ([]*core.Alert, error)
    SaveAlert(ctx context.Context, alert *core.Alert) error
    GetAlertsByFingerprint(ctx context.Context, fingerprint string) ([]*core.Alert, error)
}

type PostgresHistoryRepository struct {
    pool *pgxpool.Pool
}

func (r *PostgresHistoryRepository) SaveAlert(ctx context.Context, alert *core.Alert) error {
    // SQL implementation
}
```

**Benefits:**
- ✅ Testable (mock repository for unit tests)
- ✅ Swappable (can switch to MySQL/MongoDB)
- ✅ Clear boundary between business logic & persistence

---

### 2. Service Layer Pattern (Business Logic)

```go
// internal/business/silencing/manager.go
type SilenceManager interface {
    CreateSilence(ctx context.Context, silence *core.Silence) error
    IsAlertSilenced(ctx context.Context, alert *core.Alert) (bool, error)
    GetActiveSilences(ctx context.Context) ([]*core.Silence, error)
}

type DefaultSilenceManager struct {
    repository  SilenceRepository
    matcher     SilenceMatcher
    cache       SilenceCache
    metrics     *SilenceMetrics
}

func (m *DefaultSilenceManager) IsAlertSilenced(ctx context.Context, alert *core.Alert) (bool, error) {
    // 1. Check cache
    // 2. Query repository
    // 3. Match against silences
    // 4. Record metrics
}
```

**Benefits:**
- ✅ Orchestrates multiple repositories
- ✅ Contains business rules (not in handlers)
- ✅ Transactional logic encapsulated

---

### 3. Adapter Pattern (Layer Translation)

```go
// internal/infrastructure/webhook/parser.go
type WebhookParser interface {
    Parse(payload []byte) (*ParsedWebhook, error)
}

type AlertmanagerParser struct{}

func (p *AlertmanagerParser) Parse(payload []byte) (*ParsedWebhook, error) {
    // Convert Alertmanager JSON → domain model
    var amWebhook AlertmanagerWebhook
    json.Unmarshal(payload, &amWebhook)

    // Map to domain
    return &ParsedWebhook{
        Alerts: amWebhook.Alerts.toDomainAlerts(),
        Status: amWebhook.Status,
    }, nil
}
```

**Benefits:**
- ✅ Isolates external formats (Alertmanager, PagerDuty, Slack)
- ✅ Domain model stays clean
- ✅ Easy to add new webhook types

---

### 4. Middleware Chain Pattern (HTTP)

```go
// internal/middleware/builder.go
func BuildWebhookMiddlewareStack(config *MiddlewareConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        handler := next
        handler = TimeoutMiddleware(handler, config.RequestTimeout)
        handler = RateLimitMiddleware(handler, config.RateLimiter)
        handler = AuthMiddleware(handler, config.AuthConfig)
        handler = MetricsMiddleware(handler, config.MetricsRegistry)
        handler = RecoveryMiddleware(handler)
        return handler
    }
}
```

**Benefits:**
- ✅ Composable (add/remove middleware easily)
- ✅ Reusable across endpoints
- ✅ Clear separation of concerns

---

### 5. Factory Pattern (Publisher Creation)

```go
// internal/infrastructure/publishing/factory.go
type PublisherFactory interface {
    CreatePublisher(target *PublishingTarget) (AlertPublisher, error)
}

func (f *DefaultPublisherFactory) CreatePublisher(target *PublishingTarget) (AlertPublisher, error) {
    switch target.Type {
    case "rootly":
        return NewRootlyPublisher(target.Config, f.httpClient)
    case "pagerduty":
        return NewPagerDutyPublisher(target.Config, f.httpClient)
    case "slack":
        return NewSlackPublisher(target.Config, f.httpClient)
    default:
        return nil, fmt.Errorf("unknown publisher type: %s", target.Type)
    }
}
```

**Benefits:**
- ✅ Centralized publisher creation
- ✅ Easy to add new publisher types
- ✅ Shared dependencies (HTTP client, cache)

---

## Technology Stack (Decided in TN-09, TN-10)

| Component | Technology | Reason |
|-----------|------------|--------|
| **Web Framework** | Gin | Stdlib compatibility, Prometheus integration |
| **Database Driver** | pgx v5 | 2x faster than GORM, full PostgreSQL features |
| **Connection Pool** | pgxpool | Built-in health checks, adaptive sizing |
| **Migrations** | goose | Simple, SQL-first, version control friendly |
| **Cache** | go-redis v9 | Redis protocol, pipeline support |
| **Config** | viper | 12-factor app, env overrides |
| **Logging** | slog (stdlib) | Structured, zero dependencies |
| **Metrics** | prometheus/client_golang | Industry standard, Grafana integration |
| **HTTP Client** | net/http (stdlib) | Connection pooling, context support |
| **K8s Client** | client-go v0.29 | Official Kubernetes Go client |

---

## Directory Structure (TN-02)

```
go-app/
├── cmd/
│   └── server/              # Application entry point
│       ├── main.go          # Server initialization
│       └── handlers/        # HTTP handlers (thin layer)
├── internal/
│   ├── core/                # Domain models & interfaces (no external deps)
│   │   ├── models.go        # Alert, ClassificationResult, etc.
│   │   ├── interfaces.go    # AlertClassifier, AlertPublisher
│   │   └── processing/      # Alert processing pipeline
│   ├── business/            # Business services (orchestration)
│   │   ├── grouping/        # TN-121 to TN-125
│   │   ├── inhibition/      # TN-126 to TN-130
│   │   ├── silencing/       # TN-131 to TN-136
│   │   ├── publishing/      # TN-046 to TN-060
│   │   └── proxy/           # TN-062 Intelligent Proxy
│   ├── infrastructure/      # External integrations
│   │   ├── repository/      # Database access (pgx)
│   │   ├── cache/           # Redis client
│   │   ├── llm/             # LLM HTTP client
│   │   ├── k8s/             # Kubernetes client
│   │   └── webhook/         # Webhook parsing/formatting
│   ├── api/                 # API layer (handlers, middleware)
│   │   ├── handlers/        # REST endpoint handlers
│   │   └── middleware/      # HTTP middleware
│   ├── config/              # Configuration loading
│   └── database/            # Database connection management
├── pkg/                     # Shared packages (external-safe)
│   ├── logger/              # Logging utilities
│   ├── metrics/             # Prometheus metrics
│   └── middleware/          # Reusable HTTP middleware
├── migrations/              # SQL migrations (goose)
└── docs/                    # Documentation
    └── adr/                 # Architecture Decision Records
```

### Package Responsibilities

| Package | Depends On | Used By | Purpose |
|---------|------------|---------|---------|
| `internal/core` | NONE | All layers | Domain models, interfaces |
| `internal/business` | core, infrastructure | API handlers | Business logic |
| `internal/infrastructure` | core, external libs | Business services | External integrations |
| `internal/api` | business, middleware | cmd/server | HTTP endpoints |
| `pkg/*` | stdlib, external libs | All internal | Reusable utilities |

---

## Design Principles

### 1. Dependency Direction
```
cmd/server → API → Business → Core ← Infrastructure
                                ↑           ↓
                                └─────────┘
                            (Infrastructure implements Core interfaces)
```

**Rule:** Dependencies point INWARD (toward core domain)

### 2. Interface Ownership
```go
// Core domain OWNS interfaces
// internal/core/interfaces.go
type AlertClassifier interface {
    Classify(ctx context.Context, alert *Alert) (*ClassificationResult, error)
}

// Infrastructure IMPLEMENTS interfaces
// internal/infrastructure/llm/classifier.go
type LLMClassifier struct {
    client *HTTPLLMClient
}

func (c *LLMClassifier) Classify(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
    // Implementation
}
```

**Rule:** Core defines "what", infrastructure defines "how"

### 3. Dependency Injection
```go
// cmd/server/main.go
func main() {
    // Initialize infrastructure
    llmClient := llm.NewHTTPLLMClient(cfg.LLM)
    classifier := llm.NewLLMClassifier(llmClient)
    repository := repository.NewPostgresHistoryRepository(dbPool)

    // Initialize business service
    processor := processing.NewAlertProcessor(
        classifier,  // Interface injection
        repository,  // Interface injection
        logger,
    )

    // Initialize HTTP handlers
    handlers := handlers.NewWebhookHandlers(processor, logger)

    // Start server
    router.POST("/webhook", handlers.HandleWebhook)
}
```

**Rule:** Wire dependencies in main(), pass interfaces

---

## Testing Strategy

### Unit Tests (Fast, No External Dependencies)
```go
// internal/core/processing/processor_test.go
func TestAlertProcessor_ProcessAlert(t *testing.T) {
    // Mock dependencies
    mockClassifier := &MockClassifier{
        result: &core.ClassificationResult{Severity: "high"},
    }
    mockRepo := &MockRepository{}

    // Test business logic
    processor := NewAlertProcessor(mockClassifier, mockRepo, logger)
    err := processor.ProcessAlert(ctx, alert)

    assert.NoError(t, err)
    assert.Equal(t, 1, mockRepo.SaveCallCount)
}
```

### Integration Tests (Real Dependencies)
```go
// internal/infrastructure/repository/integration_test.go
func TestPostgresRepository_SaveAlert_Integration(t *testing.T) {
    t.Skip("Requires PostgreSQL")

    // Real database connection
    pool := testhelpers.CreateTestPool(t)
    defer pool.Close()

    repo := NewPostgresHistoryRepository(pool)
    err := repo.SaveAlert(ctx, testAlert)

    assert.NoError(t, err)
    // Verify in database
}
```

---

## Consequences

### Positive

1. **Testability**: 90%+ unit test coverage achievable (mocks for all interfaces)
2. **Maintainability**: Clear boundaries, easy to locate logic
3. **Extensibility**: Add new publishers/classifiers without touching core
4. **Team Productivity**: Parallel development (teams work on different layers)
5. **Production Stability**: Infrastructure failures don't crash core logic

### Negative

1. **Initial Complexity**: More boilerplate (interfaces, adapters, DI)
2. **Learning Curve**: Junior developers need training on patterns
3. **Code Volume**: +20-30% more code vs "simple" approach

### Mitigation

- Provide code templates (Makefile generators)
- Document patterns in internal wiki
- Conduct architecture review sessions (bi-weekly)
- Pair programming for first 2 weeks

---

## References

- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture (Ports & Adapters)](https://alistair.cockburn.us/hexagonal-architecture/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [TN-02: Directory Structure](../../tasks/alertmanager-plus-plus-oss/TASKS.md#tn-02)

---

## Related ADRs

- [ADR-001: Web Framework Selection](./001-gin-vs-fiber-framework.md)
- [ADR-002: PostgreSQL Driver Selection](./002-pgx-vs-gorm-driver.md)

---

## Review History

| Date | Reviewer | Decision |
|------|----------|----------|
| 2025-11-18 | Architect | Approved |
| 2025-11-18 | Tech Lead | Approved |
| 2025-11-18 | Backend Team | Approved |

---

**Status: ACCEPTED**
**Implementation: ACTIVE (Phase 0 foundation complete)**
**Next Review: After Phase 1 API Gateway completion**
