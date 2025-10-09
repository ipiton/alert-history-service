#!/bin/bash

# Массовое заполнение всех оставшихся задач (TN-039 до TN-120)

# ФАЗА 4: Завершение (TN-039 до TN-045)

# TN-039: Circuit breaker для LLM
cat > TN-039/requirements.md << 'EOF'
# TN-039: Circuit Breaker для LLM Calls

## 1. Обоснование
Защита от каскадных отказов при недоступности LLM сервиса.

## 2. Сценарий
При сбоях LLM circuit breaker открывается и переключается на fallback.

## 3. Требования
- Circuit breaker pattern
- Configurable thresholds
- Automatic recovery
- Metrics для состояний

## 4. Критерии приёмки
- [ ] Circuit breaker реализован
- [ ] Fallback работает
- [ ] Recovery автоматический
- [ ] Метрики собираются
EOF

cat > TN-039/design.md << 'EOF'
# TN-039: Circuit Breaker Design

```go
type CircuitBreaker interface {
    Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)
    State() CircuitState
    Metrics() *CircuitMetrics
}

type CircuitState string
const (
    StateClosed   CircuitState = "closed"
    StateOpen     CircuitState = "open"
    StateHalfOpen CircuitState = "half_open"
)

type circuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    failureCount    int64
    lastFailureTime time.Time
    state          CircuitState
    mutex          sync.RWMutex
}
```
EOF

cat > TN-039/tasks.md << 'EOF'
# TN-039: Чек-лист

- [ ] 1. Создать internal/core/resilience/circuit_breaker.go
- [ ] 2. Реализовать CircuitBreaker интерфейс
- [ ] 3. Интегрировать в LLM client
- [ ] 4. Добавить конфигурацию
- [ ] 5. Добавить метрики
- [ ] 6. Создать тесты
- [ ] 7. Коммит: `feat(go): TN-039 add circuit breaker`
EOF

# TN-040: Retry logic
cat > TN-040/requirements.md << 'EOF'
# TN-040: Retry Logic с Exponential Backoff

## 1. Обоснование
Устойчивость к временным сбоям внешних сервисов.

## 2. Сценарий
При временных ошибках система автоматически повторяет запросы.

## 3. Требования
- Exponential backoff
- Jitter для избежания thundering herd
- Configurable retry policies
- Context cancellation support

## 4. Критерии приёмки
- [ ] Retry mechanism работает
- [ ] Backoff корректный
- [ ] Jitter добавлен
- [ ] Context поддерживается
EOF

cat > TN-040/design.md << 'EOF'
# TN-040: Retry Logic Design

```go
type RetryPolicy struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
    Jitter      bool
}

func WithRetry(ctx context.Context, policy *RetryPolicy, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        if attempt < policy.MaxRetries {
            delay := calculateDelay(policy, attempt)
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }
    }
    return lastErr
}
```
EOF

cat > TN-040/tasks.md << 'EOF'
# TN-040: Чек-лист

- [ ] 1. Создать internal/core/resilience/retry.go
- [ ] 2. Реализовать RetryPolicy
- [ ] 3. Добавить exponential backoff
- [ ] 4. Интегрировать в HTTP clients
- [ ] 5. Добавить jitter
- [ ] 6. Создать тесты
- [ ] 7. Коммит: `feat(go): TN-040 add retry logic`
EOF

# TN-041: Alertmanager webhook parser
cat > TN-041/requirements.md << 'EOF'
# TN-041: Alertmanager Webhook Parser

## 1. Обоснование
Парсинг webhook payload от Alertmanager в доменные модели.

## 2. Сценарий
При получении webhook от Alertmanager данные парсятся в Alert структуры.

## 3. Требования
- Полная поддержка Alertmanager format
- Валидация входных данных
- Error handling для malformed data
- Support для различных версий

## 4. Критерии приёмки
- [ ] Парсинг работает корректно
- [ ] Валидация функционирует
- [ ] Errors обрабатываются
- [ ] Тесты покрывают edge cases
EOF

cat > TN-041/design.md << 'EOF'
# TN-041: Webhook Parser Design

```go
type AlertmanagerWebhook struct {
    Version           string                 `json:"version"`
    GroupKey          string                 `json:"groupKey"`
    TruncatedAlerts   int                   `json:"truncatedAlerts"`
    Status            string                `json:"status"`
    Receiver          string                `json:"receiver"`
    GroupLabels       map[string]string     `json:"groupLabels"`
    CommonLabels      map[string]string     `json:"commonLabels"`
    CommonAnnotations map[string]string     `json:"commonAnnotations"`
    ExternalURL       string                `json:"externalURL"`
    Alerts            []AlertmanagerAlert   `json:"alerts"`
}

type WebhookParser interface {
    Parse(data []byte) (*AlertmanagerWebhook, error)
    Validate(webhook *AlertmanagerWebhook) error
    ConvertToDomain(webhook *AlertmanagerWebhook) ([]*domain.Alert, error)
}
```
EOF

cat > TN-041/tasks.md << 'EOF'
# TN-041: Чек-лист

- [ ] 1. Создать internal/infrastructure/webhook/parser.go
- [ ] 2. Реализовать WebhookParser интерфейс
- [ ] 3. Добавить валидацию
- [ ] 4. Создать конвертер в domain модели
- [ ] 5. Обработать edge cases
- [ ] 6. Создать тесты с примерами
- [ ] 7. Коммит: `feat(go): TN-041 add webhook parser`
EOF

# TN-042: Universal webhook handler
cat > TN-042/requirements.md << 'EOF'
# TN-042: Universal Webhook Handler

## 1. Обоснование
Универсальный обработчик webhook с auto-detection формата.

## 2. Сценарий
Endpoint /webhook принимает различные форматы и автоматически их обрабатывает.

## 3. Требования
- Auto-detection формата payload
- Support Alertmanager, generic webhooks
- Routing к соответствующим parsers
- Error handling и logging

## 4. Критерии приёмки
- [ ] Auto-detection работает
- [ ] Различные форматы поддерживаются
- [ ] Routing корректный
- [ ] Errors логируются
EOF

cat > TN-042/design.md << 'EOF'
# TN-042: Universal Handler Design

```go
type WebhookHandler struct {
    parsers            map[WebhookType]WebhookParser
    deduplicationSvc   DeduplicationService
    classificationSvc  AlertClassificationService
    publishingSvc      PublishingService
    enrichmentManager  EnrichmentModeManager
    logger            *slog.Logger
    metrics           *prometheus.CounterVec
}

func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
    body := c.Body()

    // Auto-detect webhook type
    webhookType, err := h.detectWebhookType(body)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "unknown webhook format"})
    }

    // Parse webhook
    parser := h.parsers[webhookType]
    webhook, err := parser.Parse(body)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid webhook payload"})
    }

    // Process alerts
    alerts, err := parser.ConvertToDomain(webhook)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "conversion failed"})
    }

    return h.processAlerts(c.Context(), alerts)
}
```
EOF

cat > TN-042/tasks.md << 'EOF'
# TN-042: Чек-лист

- [ ] 1. Создать internal/api/handlers/webhook.go
- [ ] 2. Реализовать WebhookHandler
- [ ] 3. Добавить auto-detection логику
- [ ] 4. Интегрировать все parsers
- [ ] 5. Добавить error handling
- [ ] 6. Создать integration тесты
- [ ] 7. Коммит: `feat(go): TN-042 add universal webhook handler`
EOF

# TN-043: Webhook validation
cat > TN-043/requirements.md << 'EOF'
# TN-043: Webhook Validation & Error Handling

## 1. Обоснование
Валидация webhook данных и обработка ошибок.

## 2. Сценарий
Невалидные webhook отклоняются с понятными error messages.

## 3. Требования
- Schema validation
- Required fields checking
- Format validation
- Detailed error messages

## 4. Критерии приёмки
- [ ] Валидация работает
- [ ] Error messages понятные
- [ ] Schema проверяется
- [ ] Edge cases покрыты
EOF

cat > TN-043/design.md << 'EOF'
# TN-043: Validation Design

```go
type WebhookValidator interface {
    ValidateAlertmanager(webhook *AlertmanagerWebhook) error
    ValidateGeneric(data map[string]interface{}) error
}

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   interface{} `json:"value,omitempty"`
}

type ValidationResult struct {
    Valid  bool              `json:"valid"`
    Errors []*ValidationError `json:"errors,omitempty"`
}
```
EOF

cat > TN-043/tasks.md << 'EOF'
# TN-043: Чек-лист

- [ ] 1. Создать internal/infrastructure/webhook/validator.go
- [ ] 2. Реализовать WebhookValidator
- [ ] 3. Добавить validation rules
- [ ] 4. Создать ValidationError types
- [ ] 5. Интегрировать в webhook handler
- [ ] 6. Создать тесты
- [ ] 7. Коммит: `feat(go): TN-043 add webhook validation`
EOF

# TN-044: Async webhook processing
cat > TN-044/requirements.md << 'EOF'
# TN-044: Async Webhook Processing

## 1. Обоснование
Асинхронная обработка webhook для улучшения производительности.

## 2. Сценарий
Webhook быстро принимаются и обрабатываются в background workers.

## 3. Требования
- Worker pool для обработки
- Queue для задач
- Retry для failed jobs
- Monitoring обработки

## 4. Критерии приёмки
- [ ] Worker pool работает
- [ ] Queue функционирует
- [ ] Retry реализован
- [ ] Метрики собираются
EOF

cat > TN-044/design.md << 'EOF'
# TN-044: Async Processing Design

```go
type WebhookProcessor interface {
    SubmitJob(ctx context.Context, job *WebhookJob) error
    Start(ctx context.Context) error
    Stop() error
    Stats() *ProcessorStats
}

type WebhookJob struct {
    ID        string    `json:"id"`
    Type      WebhookType `json:"type"`
    Payload   []byte    `json:"payload"`
    CreatedAt time.Time `json:"created_at"`
    Attempts  int       `json:"attempts"`
}

type webhookProcessor struct {
    workers     int
    jobQueue    chan *WebhookJob
    workerPool  chan chan *WebhookJob
    quit        chan bool
    wg          sync.WaitGroup
}
```
EOF

cat > TN-044/tasks.md << 'EOF'
# TN-044: Чек-лист

- [ ] 1. Создать internal/core/processing/webhook_processor.go
- [ ] 2. Реализовать Worker pool
- [ ] 3. Добавить Job queue
- [ ] 4. Интегрировать retry logic
- [ ] 5. Добавить monitoring
- [ ] 6. Создать тесты
- [ ] 7. Коммит: `feat(go): TN-044 add async processing`
EOF

# TN-045: Webhook metrics
cat > TN-045/requirements.md << 'EOF'
# TN-045: Webhook Metrics & Monitoring

## 1. Обоснование
Мониторинг производительности и здоровья webhook processing.

## 2. Сценарий
Prometheus собирает метрики обработки webhook для alerting.

## 3. Требования
- Request rate metrics
- Processing time histograms
- Error rate tracking
- Queue size monitoring

## 4. Критерии приёмки
- [ ] Все метрики собираются
- [ ] Dashboards работают
- [ ] Alerting настроен
- [ ] Performance отслеживается
EOF

cat > TN-045/design.md << 'EOF'
# TN-045: Webhook Metrics Design

```go
type WebhookMetrics struct {
    RequestsTotal     *prometheus.CounterVec
    RequestDuration   *prometheus.HistogramVec
    ProcessingTime    *prometheus.HistogramVec
    QueueSize         prometheus.Gauge
    ActiveWorkers     prometheus.Gauge
    ErrorsTotal       *prometheus.CounterVec
}

func NewWebhookMetrics() *WebhookMetrics {
    return &WebhookMetrics{
        RequestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_requests_total",
                Help: "Total number of webhook requests",
            },
            []string{"type", "status"},
        ),
        RequestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "webhook_request_duration_seconds",
                Help: "Webhook request duration",
            },
            []string{"type"},
        ),
    }
}
```
EOF

cat > TN-045/tasks.md << 'EOF'
# TN-045: Чек-лист

- [ ] 1. Создать internal/core/metrics/webhook.go
- [ ] 2. Определить все метрики
- [ ] 3. Интегрировать в webhook handlers
- [ ] 4. Добавить в /metrics endpoint
- [ ] 5. Создать Grafana dashboard
- [ ] 6. Настроить alerting rules
- [ ] 7. Коммит: `feat(go): TN-045 add webhook metrics`
EOF

echo "ФАЗА 4 полностью заполнена (TN-031 до TN-045)!"

# ФАЗА 5: Publishing System (TN-046 до TN-060)

# TN-046: Kubernetes client
cat > TN-046/requirements.md << 'EOF'
# TN-046: Kubernetes Client для Secrets Discovery

## 1. Обоснование
Kubernetes client для обнаружения publishing targets из secrets.

## 2. Сценарий
Приложение автоматически обнаруживает новые targets из K8s secrets.

## 3. Требования
- client-go integration
- Watch для secrets changes
- Label selector filtering
- RBAC permissions

## 4. Критерии приёмки
- [ ] K8s client работает
- [ ] Watch events обрабатываются
- [ ] Label filtering функционирует
- [ ] RBAC настроен
EOF

cat > TN-046/design.md << 'EOF'
# TN-046: Kubernetes Client Design

```go
type KubernetesClient interface {
    ListSecrets(ctx context.Context, namespace string, labelSelector string) (*v1.SecretList, error)
    WatchSecrets(ctx context.Context, namespace string, labelSelector string) (watch.Interface, error)
    GetSecret(ctx context.Context, namespace, name string) (*v1.Secret, error)
}

type kubernetesClient struct {
    clientset kubernetes.Interface
    config    *rest.Config
    logger    *slog.Logger
}

func NewKubernetesClient(kubeconfig string) (KubernetesClient, error) {
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        // Try in-cluster config
        config, err = rest.InClusterConfig()
        if err != nil {
            return nil, err
        }
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, err
    }

    return &kubernetesClient{
        clientset: clientset,
        config:    config,
        logger:    slog.Default(),
    }, nil
}
```
EOF

cat > TN-046/tasks.md << 'EOF'
# TN-046: Чек-лист

- [ ] 1. Добавить client-go: `go get k8s.io/client-go`
- [ ] 2. Создать internal/infrastructure/kubernetes/client.go
- [ ] 3. Реализовать KubernetesClient интерфейс
- [ ] 4. Добавить watch functionality
- [ ] 5. Создать RBAC манифесты
- [ ] 6. Создать тесты
- [ ] 7. Коммит: `feat(go): TN-046 add kubernetes client`
EOF

# TN-047: Target discovery manager
cat > TN-047/requirements.md << 'EOF'
# TN-047: Target Discovery Manager

## 1. Обоснование
Менеджер для обнаружения и управления publishing targets.

## 2. Сценарий
Система автоматически обнаруживает новые targets и обновляет конфигурацию.

## 3. Требования
- Dynamic target discovery
- Label selector support
- Target validation
- Configuration management

## 4. Критерии приёмки
- [ ] Discovery работает автоматически
- [ ] Targets валидируются
- [ ] Configuration обновляется
- [ ] Метрики собираются
EOF

cat > TN-047/design.md << 'EOF'
# TN-047: Target Discovery Design

```go
type TargetDiscoveryManager interface {
    Start(ctx context.Context) error
    Stop() error
    GetTargets() []*domain.PublishingTarget
    RefreshTargets(ctx context.Context) error
    GetStats() *DiscoveryStats
}

type targetDiscoveryManager struct {
    k8sClient     KubernetesClient
    namespace     string
    labelSelector string
    targets       map[string]*domain.PublishingTarget
    mutex         sync.RWMutex
    logger        *slog.Logger
    metrics       *prometheus.GaugeVec
}

func (m *targetDiscoveryManager) Start(ctx context.Context) error {
    // Initial discovery
    if err := m.RefreshTargets(ctx); err != nil {
        return err
    }

    // Start watching for changes
    watcher, err := m.k8sClient.WatchSecrets(ctx, m.namespace, m.labelSelector)
    if err != nil {
        return err
    }

    go m.handleSecretEvents(ctx, watcher)

    return nil
}

func (m *targetDiscoveryManager) handleSecretEvents(ctx context.Context, watcher watch.Interface) {
    defer watcher.Stop()

    for event := range watcher.ResultChan() {
        secret, ok := event.Object.(*v1.Secret)
        if !ok {
            continue
        }

        switch event.Type {
        case watch.Added, watch.Modified:
            target := m.convertSecretToTarget(secret)
            if target != nil {
                m.addOrUpdateTarget(target)
            }
        case watch.Deleted:
            m.removeTarget(secret.Name)
        }
    }
}
```
EOF

cat > TN-047/tasks.md << 'EOF'
# TN-047: Чек-лист

- [ ] 1. Создать internal/core/services/target_discovery.go
- [ ] 2. Реализовать TargetDiscoveryManager
- [ ] 3. Добавить watch functionality
- [ ] 4. Создать secret-to-target converter
- [ ] 5. Добавить validation
- [ ] 6. Интегрировать метрики
- [ ] 7. Коммит: `feat(go): TN-047 add target discovery`
EOF

echo "Создано еще 7 задач. Продолжаем заполнение остальных..."
