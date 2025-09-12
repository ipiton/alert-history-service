# Go Migration Architecture Design

## Предлагаемое архитектурное решение

### Общая архитектура

```
Go Alert History Service
├── cmd/
│   └── server/
│       └── main.go                 # Entry point
├── internal/
│   ├── api/                        # HTTP handlers (Gin/Fiber)
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── core/                       # Business logic
│   │   ├── domain/                 # Domain models
│   │   ├── services/               # Business services
│   │   └── interfaces/             # Interfaces (ports)
│   ├── infrastructure/             # External adapters
│   │   ├── database/               # PostgreSQL/SQLite
│   │   ├── redis/                  # Caching
│   │   ├── kubernetes/             # K8s client
│   │   ├── llm/                    # LLM proxy client
│   │   └── publishers/             # External publishers
│   └── config/                     # Configuration
└── pkg/                            # Shared packages
    ├── logger/                     # Structured logging
    ├── metrics/                    # Prometheus metrics
    └── utils/                      # Common utilities
```

### Выбор технологий

#### Web Framework: **Fiber** (рекомендация)
```go
// Преимущества Fiber:
// - Express.js-like API (знакомый синтаксис)
// - Высокая производительность (на основе fasthttp)
// - Богатая экосистема middleware
// - Отличная документация
// - Поддержка OpenAPI/Swagger

app := fiber.New(fiber.Config{
    Prefork:       false,
    CaseSensitive: true,
    StrictRouting: true,
    ServerHeader:  "AlertHistory",
    AppName:       "Alert History Service v2.0.0",
})
```

**Альтернативы:**
- **Gin**: Более популярный, но менее производительный
- **Echo**: Хороший баланс, но меньше middleware

#### Database: **pgx** + **GORM** (hybrid подход)
```go
// pgx для высокопроизводительных операций
conn, err := pgxpool.Connect(ctx, databaseURL)

// GORM для сложных запросов и миграций
db, err := gorm.Open(postgres.New(postgres.Config{
    Conn: stdlib.OpenDB(*pgxConfig),
}), &gorm.Config{})
```

#### Redis: **go-redis/redis/v9**
```go
rdb := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
```

#### Logging: **slog** (Go 1.21+)
```go
// Structured logging с JSON output
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

### Архитектурные паттерны

#### 1. Hexagonal Architecture (Ports & Adapters)

```go
// Core domain (internal/core/domain/)
type Alert struct {
    Fingerprint string            `json:"fingerprint"`
    AlertName   string            `json:"alert_name"`
    Status      AlertStatus       `json:"status"`
    Labels      map[string]string `json:"labels"`
    // ...
}

// Ports (internal/core/interfaces/)
type AlertRepository interface {
    Save(ctx context.Context, alert *Alert) error
    GetByFingerprint(ctx context.Context, fingerprint string) (*Alert, error)
    GetAlerts(ctx context.Context, filters map[string]interface{}) ([]*Alert, error)
}

type LLMClient interface {
    ClassifyAlert(ctx context.Context, alert *Alert) (*ClassificationResult, error)
}

// Adapters (internal/infrastructure/)
type PostgreSQLAlertRepository struct {
    db *gorm.DB
}

type LLMProxyClient struct {
    httpClient *http.Client
    baseURL    string
}
```

#### 2. Dependency Injection

```go
// Используем wire для compile-time DI
//go:generate wire
//+build wireinject

func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(
        // Repositories
        NewPostgreSQLAlertRepository,
        NewRedisCache,

        // Services
        NewAlertClassificationService,
        NewWebhookProcessor,
        NewAlertPublisher,

        // HTTP layer
        NewAlertHandlers,
        NewApp,
    )
    return &App{}, nil
}
```

#### 3. Configuration Management

```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    LLM      LLMConfig      `mapstructure:"llm"`
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AutomaticEnv()

    // 12-Factor App: приоритет ENV переменным
    viper.BindEnv("server.port", "PORT")
    viper.BindEnv("database.url", "DATABASE_URL")

    return cfg, viper.Unmarshal(&cfg)
}
```

## Формат данных и API контракты

### REST API Compatibility

Все существующие endpoints должны сохранить полную совместимость:

```go
// POST /webhook - Universal webhook
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
    var req WebhookRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "Invalid request body")
    }

    result, err := h.webhookService.ProcessWebhook(c.Context(), &req)
    if err != nil {
        return err
    }

    return c.JSON(result)
}

// GET /history - Alert history with filters
func (h *AlertHandler) GetHistory(c *fiber.Ctx) error {
    filters := parseHistoryFilters(c)
    alerts, err := h.alertService.GetAlerts(c.Context(), filters)
    if err != nil {
        return err
    }

    return c.JSON(map[string]interface{}{
        "alerts": alerts,
        "total":  len(alerts),
    })
}
```

### Database Schema Compatibility

```go
// Полная совместимость с существующей PostgreSQL схемой
type Alert struct {
    ID          uint      `gorm:"primaryKey"`
    Fingerprint string    `gorm:"uniqueIndex;not null"`
    AlertName   string    `gorm:"not null"`
    Status      string    `gorm:"not null"`
    Labels      datatypes.JSON
    Annotations datatypes.JSON
    StartsAt    time.Time
    EndsAt      *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Classification struct {
    ID            uint    `gorm:"primaryKey"`
    Fingerprint   string  `gorm:"uniqueIndex;not null"`
    Severity      string  `gorm:"not null"`
    Confidence    float64 `gorm:"not null"`
    Reasoning     string
    Recommendations datatypes.JSON
    ProcessingTime  float64
    CreatedAt     time.Time
}
```

### Message Formats

```go
// LLM Classification Request/Response
type ClassificationRequest struct {
    Alert   Alert             `json:"alert"`
    Context map[string]interface{} `json:"context,omitempty"`
}

type ClassificationResult struct {
    Severity        AlertSeverity `json:"severity"`
    Confidence      float64       `json:"confidence"`
    Reasoning       string        `json:"reasoning"`
    Recommendations []string      `json:"recommendations"`
    ProcessingTime  float64       `json:"processing_time"`
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Publishing Formats (Rootly, PagerDuty, Slack)
type RootlyPayload struct {
    Title       string            `json:"title"`
    Description string            `json:"description"`
    Severity    string            `json:"severity"`
    Labels      map[string]string `json:"labels"`
    Source      string            `json:"source"`
}
```

## Сценарии ошибок и edge cases

### 1. Database Connection Issues

```go
type DatabaseHealthChecker struct {
    db *gorm.DB
}

func (h *DatabaseHealthChecker) Check(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    var result int
    err := h.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
    if err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }

    return nil
}

// Graceful degradation
func (s *AlertService) SaveAlert(ctx context.Context, alert *Alert) error {
    if err := s.repo.Save(ctx, alert); err != nil {
        // Fallback to local file storage
        s.logger.Error("Database save failed, using fallback",
            "error", err, "fingerprint", alert.Fingerprint)
        return s.fallbackStorage.Save(ctx, alert)
    }
    return nil
}
```

### 2. LLM Service Unavailable

```go
type CircuitBreaker struct {
    failures    int
    lastFailure time.Time
    timeout     time.Duration
    threshold   int
}

func (s *LLMService) ClassifyAlert(ctx context.Context, alert *Alert) (*ClassificationResult, error) {
    if s.circuitBreaker.IsOpen() {
        s.metrics.IncCounter("llm_circuit_breaker_open")
        return s.fallbackClassification(alert), nil
    }

    result, err := s.client.ClassifyAlert(ctx, alert)
    if err != nil {
        s.circuitBreaker.RecordFailure()
        s.logger.Warn("LLM classification failed, using fallback",
            "error", err, "fingerprint", alert.Fingerprint)
        return s.fallbackClassification(alert), nil
    }

    s.circuitBreaker.RecordSuccess()
    return result, nil
}
```

### 3. Redis Cache Failures

```go
func (c *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
    val, err := c.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, ErrCacheKeyNotFound
    }
    if err != nil {
        c.logger.Error("Redis get failed", "error", err, "key", key)
        c.metrics.IncCounter("cache_errors", map[string]string{
            "operation": "get",
            "error_type": "connection",
        })
        return nil, err
    }

    c.metrics.IncCounter("cache_hits")
    return val, nil
}
```

### 4. High Load Scenarios

```go
// Rate limiting middleware
func RateLimitMiddleware(rps int) fiber.Handler {
    limiter := rate.NewLimiter(rate.Limit(rps), rps*2)

    return func(c *fiber.Ctx) error {
        if !limiter.Allow() {
            return fiber.NewError(429, "Too Many Requests")
        }
        return c.Next()
    }
}

// Request timeout middleware
func TimeoutMiddleware(timeout time.Duration) fiber.Handler {
    return func(c *fiber.Ctx) error {
        ctx, cancel := context.WithTimeout(c.Context(), timeout)
        defer cancel()

        c.SetUserContext(ctx)
        return c.Next()
    }
}
```

### 5. Graceful Shutdown

```go
func (a *App) Start() error {
    go func() {
        if err := a.server.Listen(":8080"); err != nil {
            a.logger.Error("Server failed to start", "error", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    a.logger.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return a.server.ShutdownWithContext(ctx)
}
```

## Performance Considerations

### Memory Management

```go
// Object pooling для часто используемых объектов
var alertPool = sync.Pool{
    New: func() interface{} {
        return &Alert{}
    },
}

func (s *AlertService) ProcessAlert(data []byte) error {
    alert := alertPool.Get().(*Alert)
    defer alertPool.Put(alert)

    // Reset alert fields
    *alert = Alert{}

    if err := json.Unmarshal(data, alert); err != nil {
        return err
    }

    return s.processAlert(alert)
}
```

### Goroutine Management

```go
// Worker pool для обработки алертов
type WorkerPool struct {
    workers    int
    jobQueue   chan Job
    workerPool chan chan Job
    quit       chan bool
}

func (w *WorkerPool) Start() {
    for i := 0; i < w.workers; i++ {
        worker := NewWorker(w.workerPool)
        worker.Start()
    }

    go w.dispatch()
}

func (w *WorkerPool) dispatch() {
    for {
        select {
        case job := <-w.jobQueue:
            go func() {
                workerJobQueue := <-w.workerPool
                workerJobQueue <- job
            }()
        case <-w.quit:
            return
        }
    }
}
```

### Database Optimization

```go
// Connection pooling
func NewPostgreSQLDB(cfg *DatabaseConfig) (*gorm.DB, error) {
    config, err := pgxpool.ParseConfig(cfg.URL)
    if err != nil {
        return nil, err
    }

    // Настройка пула соединений
    config.MaxConns = cfg.MaxConnections
    config.MinConns = cfg.MinConnections
    config.MaxConnLifetime = cfg.MaxConnLifetime
    config.MaxConnIdleTime = cfg.MaxConnIdleTime

    pool, err := pgxpool.ConnectConfig(context.Background(), config)
    if err != nil {
        return nil, err
    }

    return gorm.Open(postgres.New(postgres.Config{
        Conn: stdlib.OpenDB(*config.ConnConfig),
    }), &gorm.Config{
        PrepareStmt: true, // Подготовленные запросы для производительности
    })
}
```
