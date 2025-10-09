# Contributing to Alert History Go Service

Добро пожаловать в проект Alert History Go Service! Этот документ поможет вам начать разработку на Go и следовать принятым стандартам.

## 📋 Table of Contents

- [Development Setup](#-development-setup)
- [Code Standards](#-code-standards)
- [Testing Guidelines](#-testing-guidelines)
- [Development Workflow](#-development-workflow)
- [Pull Request Process](#-pull-request-process)
- [Code Examples](#-code-examples)
- [Troubleshooting](#-troubleshooting)

## 🚀 Development Setup

### Prerequisites

1. **Go Installation** (версия 1.24.6+)
   ```bash
   # Проверить версию
   go version

   # Установка через официальный сайт
   # https://golang.org/dl/

   # Или через homebrew (macOS)
   brew install go

   # Или через apt (Ubuntu)
   sudo apt install golang-go
   ```

2. **IDE Setup**

   **VS Code (рекомендуется):**
   ```bash
   # Установить Go extension
   code --install-extension golang.go
   ```

   **GoLand/IntelliJ IDEA:**
   - Установить Go plugin
   - Настроить GOPATH и GOROOT

   **Vim/Neovim:**
   ```bash
   # vim-go plugin
   # https://github.com/fatih/vim-go
   ```

3. **Development Tools**
   ```bash
   # golangci-lint для линтинга
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

   # goimports для автоматического импорта
   go install golang.org/x/tools/cmd/goimports@latest

   # air для hot reload (опционально)
   go install github.com/cosmtrek/air@latest
   ```

### Local Environment Setup

1. **Clone Repository**
   ```bash
   git clone <repository-url>
   cd AlertHistory/go-app
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Environment Configuration**
   ```bash
   # Скопировать пример конфигурации
   cp ../env.example .env

   # Отредактировать переменные окружения
   vim .env
   ```

4. **Build and Run**
   ```bash
   # Сборка
   make build

   # Запуск в development режиме
   make run

   # Или с hot reload
   air

   # Запуск в mock режиме (без БД)
   MOCK_MODE=true ./server
   ```

5. **Verify Setup**
   ```bash
   # Проверить health endpoint
   curl http://localhost:8080/healthz

   # Проверить metrics endpoint
   curl http://localhost:8080/metrics
   ```

## 📝 Code Standards

### Go Code Style

Мы следуем стандартным Go conventions и дополнительным правилам:

#### 1. Formatting
```bash
# Автоматическое форматирование (обязательно перед коммитом)
make fmt

# Или напрямую
gofmt -w .
goimports -w .
```

#### 2. Naming Conventions

**Packages:**
```go
// ✅ Good
package handlers
package metrics
package database

// ❌ Bad
package handlerUtils
package MetricsCollector
```

**Functions and Variables:**
```go
// ✅ Good - exported functions start with capital
func NewHTTPServer() *http.Server {}
func (s *Server) Start() error {}

// ✅ Good - unexported functions start with lowercase
func parseConfig() (*Config, error) {}
func validateRequest(req *Request) error {}

// ✅ Good - variable names
var httpClient *http.Client
var maxRetries int
var isEnabled bool

// ❌ Bad
func new_http_server() {} // snake_case
func HTTPserver() {}      // mixed case
var HTTP_CLIENT *http.Client // snake_case
```

**Constants:**
```go
// ✅ Good
const (
    DefaultPort     = 8080
    MaxRetries      = 3
    TimeoutDuration = 30 * time.Second
)

// ❌ Bad
const default_port = 8080
const MAX_RETRIES = 3
```

#### 3. Error Handling

**Always handle errors explicitly:**
```go
// ✅ Good
func processWebhook(req *WebhookRequest) error {
    data, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("failed to marshal webhook request: %w", err)
    }

    if err := validateData(data); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    return nil
}

// ❌ Bad
func processWebhook(req *WebhookRequest) error {
    data, _ := json.Marshal(req) // ignoring error
    validateData(data)           // ignoring error
    return nil
}
```

**Error wrapping with context:**
```go
// ✅ Good
if err := db.SaveAlert(alert); err != nil {
    return fmt.Errorf("failed to save alert %s: %w", alert.ID, err)
}

// ❌ Bad
if err := db.SaveAlert(alert); err != nil {
    return err // no context
}
```

#### 4. Logging

Используем structured logging с `slog`:

```go
// ✅ Good
slog.Info("Processing webhook",
    "alert_name", req.AlertName,
    "status", req.Status,
    "processing_time", time.Since(start),
)

slog.Error("Failed to process webhook",
    "error", err,
    "alert_name", req.AlertName,
    "retry_count", retryCount,
)

// ❌ Bad
log.Println("Processing webhook:", req.AlertName) // unstructured
fmt.Printf("Error: %v\n", err)                   // not using slog
```

#### 5. Context Usage

**Always pass context for cancellation and timeouts:**
```go
// ✅ Good
func (s *Service) ProcessAlert(ctx context.Context, alert *Alert) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    return s.db.SaveAlert(ctx, alert)
}

// ❌ Bad
func (s *Service) ProcessAlert(alert *Alert) error {
    return s.db.SaveAlert(alert) // no context
}
```

#### 6. Interface Design

**Keep interfaces small and focused:**
```go
// ✅ Good
type AlertStorage interface {
    SaveAlert(ctx context.Context, alert *Alert) error
    GetAlert(ctx context.Context, id string) (*Alert, error)
}

type AlertProcessor interface {
    Process(ctx context.Context, alert *Alert) error
}

// ❌ Bad - too many responsibilities
type AlertManager interface {
    SaveAlert(ctx context.Context, alert *Alert) error
    GetAlert(ctx context.Context, id string) (*Alert, error)
    Process(ctx context.Context, alert *Alert) error
    SendNotification(ctx context.Context, alert *Alert) error
    ValidateAlert(alert *Alert) error
    FormatAlert(alert *Alert) string
}
```

### Project Structure

Следуем стандартной Go project layout:

```
go-app/
├── cmd/                    # Main applications
│   └── server/            # HTTP server
│       ├── main.go
│       └── handlers/      # HTTP handlers
├── internal/              # Private application code
│   ├── api/              # API layer
│   ├── config/           # Configuration
│   ├── core/             # Business logic
│   ├── database/         # Database layer
│   └── infrastructure/   # External services
├── pkg/                  # Public library code
│   ├── logger/           # Logging utilities
│   ├── metrics/          # Metrics collection
│   └── utils/            # Common utilities
├── migrations/           # Database migrations
├── benchmark/            # Benchmarks
└── Makefile             # Build automation
```

## 🧪 Testing Guidelines

### Test Structure

```go
// ✅ Good test structure
func TestWebhookHandler(t *testing.T) {
    tests := []struct {
        name           string
        payload        string
        expectedStatus int
        expectedBody   string
    }{
        {
            name:           "valid webhook payload",
            payload:        `{"alertname":"test","status":"firing"}`,
            expectedStatus: http.StatusOK,
            expectedBody:   "success",
        },
        {
            name:           "invalid JSON payload",
            payload:        `{invalid json}`,
            expectedStatus: http.StatusBadRequest,
            expectedBody:   "Invalid JSON",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tt.payload))
            w := httptest.NewRecorder()

            WebhookHandler(w, req)

            assert.Equal(t, tt.expectedStatus, w.Code)
            assert.Contains(t, w.Body.String(), tt.expectedBody)
        })
    }
}
```

### Test Categories

1. **Unit Tests** - тестируют отдельные функции
   ```bash
   # Запуск unit тестов
   make test

   # С покрытием
   make test-coverage
   ```

2. **Integration Tests** - тестируют взаимодействие компонентов
   ```go
   // +build integration

   func TestDatabaseIntegration(t *testing.T) {
       // Integration test code
   }
   ```

3. **Benchmark Tests** - тестируют производительность
   ```go
   func BenchmarkWebhookHandler(b *testing.B) {
       for i := 0; i < b.N; i++ {
           // Benchmark code
       }
   }
   ```

### Test Utilities

```go
// ✅ Good - test helpers
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    t.Cleanup(func() {
        db.Close()
    })

    return db
}

func createTestAlert() *Alert {
    return &Alert{
        ID:        "test-123",
        AlertName: "TestAlert",
        Status:    "firing",
        Labels:    map[string]string{"severity": "warning"},
    }
}
```

### Mocking

Используем интерфейсы для мокинга:

```go
// ✅ Good - mockable interface
type AlertStorage interface {
    SaveAlert(ctx context.Context, alert *Alert) error
}

// Mock implementation
type MockAlertStorage struct {
    SaveAlertFunc func(ctx context.Context, alert *Alert) error
}

func (m *MockAlertStorage) SaveAlert(ctx context.Context, alert *Alert) error {
    if m.SaveAlertFunc != nil {
        return m.SaveAlertFunc(ctx, alert)
    }
    return nil
}
```

## 🔄 Development Workflow

### Branch Naming

```bash
# Feature branches
git checkout -b feature/TN-XX-short-description
git checkout -b feature/add-llm-integration

# Bug fixes
git checkout -b fix/webhook-validation-error
git checkout -b hotfix/memory-leak

# Documentation
git checkout -b docs/update-contributing-guide

# Refactoring
git checkout -b refactor/extract-alert-processor
```

### Commit Messages

Следуем [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# ✅ Good commit messages
git commit -m "feat(webhook): add alert validation middleware"
git commit -m "fix(database): handle connection timeout properly"
git commit -m "docs(api): update webhook endpoint documentation"
git commit -m "test(handlers): add integration tests for webhook"
git commit -m "refactor(metrics): extract prometheus collector"

# ❌ Bad commit messages
git commit -m "fix bug"
git commit -m "update code"
git commit -m "WIP"
```

**Commit Types:**
- `feat`: новая функциональность
- `fix`: исправление бага
- `docs`: изменения в документации
- `test`: добавление или изменение тестов
- `refactor`: рефакторинг кода
- `perf`: улучшение производительности
- `chore`: обновление зависимостей, конфигурации

### Development Process

1. **Создание ветки**
   ```bash
   git checkout -b feature/TN-XX-description
   ```

2. **Разработка с TDD**
   ```bash
   # Написать тест
   # Запустить тест (должен упасть)
   make test

   # Написать минимальный код для прохождения теста
   # Запустить тест (должен пройти)
   make test

   # Рефакторинг
   # Запустить тесты (должны проходить)
   make test
   ```

3. **Code Quality Checks**
   ```bash
   # Форматирование
   make fmt

   # Линтинг
   make lint

   # Тесты с покрытием
   make test-coverage

   # Vet проверки
   make vet
   ```

4. **Коммит изменений**
   ```bash
   git add .
   git commit -m "feat(component): description"
   ```

## 📋 Pull Request Process

### Pre-PR Checklist

Перед созданием PR убедитесь:

- [ ] **Code Quality**
  - [ ] `make fmt` выполнен
  - [ ] `make lint` проходит без ошибок
  - [ ] `make vet` проходит без предупреждений
  - [ ] Нет `TODO` или `FIXME` комментариев

- [ ] **Testing**
  - [ ] `make test` проходит все тесты
  - [ ] Новый код покрыт тестами (>80%)
  - [ ] Integration тесты проходят
  - [ ] Benchmark тесты не показывают деградации

- [ ] **Documentation**
  - [ ] Публичные функции имеют godoc комментарии
  - [ ] README обновлен (если нужно)
  - [ ] API документация обновлена

- [ ] **Security**
  - [ ] `make security` проходит без критических уязвимостей
  - [ ] Нет хардкода паролей/ключей
  - [ ] Input validation реализована

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Tests pass locally
- [ ] Documentation updated
```

### Code Review Guidelines

**For Authors:**
- Создавайте небольшие, фокусированные PR
- Добавляйте подробное описание изменений
- Отвечайте на комментарии конструктивно
- Исправляйте замечания в отдельных коммитах

**For Reviewers:**
- Проверяйте логику, а не только стиль
- Предлагайте конкретные улучшения
- Одобряйте PR только после полной проверки
- Фокусируйтесь на читаемости и поддерживаемости

## 💡 Code Examples

### HTTP Handler Example

```go
// ✅ Good HTTP handler
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()

    // Log request
    slog.Info("Webhook request received",
        "method", r.Method,
        "path", r.URL.Path,
        "remote_addr", r.RemoteAddr,
    )

    // Validate method
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Read body with size limit
    body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
    if err != nil {
        slog.Error("Failed to read request body", "error", err)
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Parse JSON
    var req WebhookRequest
    if err := json.Unmarshal(body, &req); err != nil {
        slog.Error("Failed to parse JSON", "error", err)
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    // Validate request
    if err := validateWebhookRequest(&req); err != nil {
        slog.Warn("Invalid webhook request", "error", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Process webhook
    alertID, err := processWebhook(r.Context(), &req)
    if err != nil {
        slog.Error("Failed to process webhook", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Success response
    response := WebhookResponse{
        Status:         "success",
        AlertID:        alertID,
        ProcessingTime: time.Since(startTime),
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)

    if err := json.NewEncoder(w).Encode(response); err != nil {
        slog.Error("Failed to encode response", "error", err)
    }

    slog.Info("Webhook processed successfully",
        "alert_id", alertID,
        "processing_time", time.Since(startTime),
    )
}
```

### Service Layer Example

```go
// ✅ Good service implementation
type AlertService struct {
    storage AlertStorage
    logger  *slog.Logger
    metrics *metrics.Collector
}

func NewAlertService(storage AlertStorage, logger *slog.Logger, metrics *metrics.Collector) *AlertService {
    return &AlertService{
        storage: storage,
        logger:  logger,
        metrics: metrics,
    }
}

func (s *AlertService) ProcessAlert(ctx context.Context, alert *Alert) error {
    start := time.Now()
    defer func() {
        s.metrics.RecordProcessingTime(time.Since(start))
    }()

    // Validate alert
    if err := s.validateAlert(alert); err != nil {
        s.metrics.IncrementErrorCount("validation")
        return fmt.Errorf("alert validation failed: %w", err)
    }

    // Enrich alert with metadata
    if err := s.enrichAlert(ctx, alert); err != nil {
        s.logger.Warn("Failed to enrich alert", "error", err, "alert_id", alert.ID)
        // Continue processing even if enrichment fails
    }

    // Save to storage
    if err := s.storage.SaveAlert(ctx, alert); err != nil {
        s.metrics.IncrementErrorCount("storage")
        return fmt.Errorf("failed to save alert: %w", err)
    }

    s.metrics.IncrementProcessedCount()
    s.logger.Info("Alert processed successfully", "alert_id", alert.ID)

    return nil
}

func (s *AlertService) validateAlert(alert *Alert) error {
    if alert == nil {
        return errors.New("alert cannot be nil")
    }

    if alert.AlertName == "" {
        return errors.New("alert name is required")
    }

    if alert.Status == "" {
        return errors.New("alert status is required")
    }

    return nil
}
```

### Configuration Example

```go
// ✅ Good configuration structure
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Metrics  MetricsConfig  `mapstructure:"metrics"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
    Host                     string        `mapstructure:"host"`
    Port                     int           `mapstructure:"port"`
    ReadTimeout              time.Duration `mapstructure:"read_timeout"`
    WriteTimeout             time.Duration `mapstructure:"write_timeout"`
    GracefulShutdownTimeout  time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")

    // Environment variable overrides
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Default values
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("server.read_timeout", "30s")
    viper.SetDefault("server.write_timeout", "30s")
    viper.SetDefault("server.graceful_shutdown_timeout", "30s")

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return &config, nil
}
```

## 🔧 Troubleshooting

### Common Issues

1. **Import Cycle Detected**
   ```
   Error: import cycle not allowed
   ```
   **Solution:** Реорганизуйте пакеты, вынесите общие интерфейсы в отдельный пакет

2. **golangci-lint Errors**
   ```bash
   # Показать все проблемы
   golangci-lint run --verbose

   # Исправить автоматически исправимые проблемы
   golangci-lint run --fix
   ```

3. **Test Failures**
   ```bash
   # Запустить конкретный тест
   go test -v ./cmd/server/handlers -run TestWebhookHandler

   # Запустить тесты с race detection
   go test -race ./...
   ```

4. **Build Issues**
   ```bash
   # Очистить module cache
   go clean -modcache

   # Обновить зависимости
   go mod tidy

   # Пересобрать все
   go build -a ./...
   ```

### Performance Debugging

```bash
# CPU профилирование
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory профилирование
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine профилирование
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### Useful Commands

```bash
# Показать зависимости модуля
go mod graph

# Найти неиспользуемые зависимости
go mod tidy

# Показать информацию о пакете
go list -m all

# Обновить зависимость
go get -u github.com/package/name

# Vendor зависимости
go mod vendor
```

## 📚 Additional Resources

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [golangci-lint Configuration](https://golangci-lint.run/usage/configuration/)

## 🤝 Getting Help

- **Slack:** #go-development
- **GitHub Issues:** Для багов и feature requests
- **Code Review:** Создавайте draft PR для раннего feedback
- **Documentation:** Проверьте go-app/README.md для специфичной информации

---

**Помните:** Хороший код - это код, который легко читать, понимать и поддерживать. Пишите код для людей, а не только для компьютера!

*Последнее обновление: 2025-09-12*
