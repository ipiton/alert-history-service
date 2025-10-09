#!/bin/bash

# ФАЗА 4: Core Business Logic (TN-31 до TN-45)

# TN-031: Alert domain models
cat > TN-031/requirements.md << 'EOF'
# TN-031: Alert Domain Models

## 1. Обоснование
Определить основные доменные модели для работы с алертами, классификацией и публикацией.

## 2. Сценарий
Разработчик импортирует domain модели и использует их в сервисах.

## 3. Требования
- Alert struct с полями Alertmanager
- Classification struct для LLM результатов
- PublishingTarget struct для внешних систем
- Validation tags и JSON serialization
- Type safety для всех полей

## 4. Критерии приёмки
- [ ] Все domain models определены
- [ ] JSON tags корректны
- [ ] Validation работает
- [ ] Unit тесты для моделей
EOF

cat > TN-031/design.md << 'EOF'
# TN-031: Design Domain Models

## Структуры
```go
// Alert represents an alert from Alertmanager
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"startsAt"`
    EndsAt       *time.Time        `json:"endsAt,omitempty"`
    GeneratorURL string            `json:"generatorURL,omitempty"`
}

type AlertStatus string
const (
    AlertStatusFiring   AlertStatus = "firing"
    AlertStatusResolved AlertStatus = "resolved"
)

// Classification represents LLM classification result
type Classification struct {
    Fingerprint     string    `json:"fingerprint"`
    Severity        Severity  `json:"severity"`
    Confidence      float64   `json:"confidence" validate:"min=0,max=1"`
    Reasoning       string    `json:"reasoning"`
    Recommendations []string  `json:"recommendations"`
    ProcessingTime  float64   `json:"processing_time"`
    CreatedAt       time.Time `json:"created_at"`
}

type Severity string
const (
    SeverityCritical Severity = "critical"
    SeverityHigh     Severity = "high"
    SeverityMedium   Severity = "medium"
    SeverityLow      Severity = "low"
    SeverityInfo     Severity = "info"
)

// PublishingTarget represents external system
type PublishingTarget struct {
    Name     string                 `json:"name"`
    Type     PublishingType         `json:"type"`
    URL      string                 `json:"url"`
    Config   map[string]interface{} `json:"config"`
    Enabled  bool                   `json:"enabled"`
    LastSeen time.Time              `json:"last_seen"`
}

type PublishingType string
const (
    PublishingTypeRootly     PublishingType = "rootly"
    PublishingTypePagerDuty  PublishingType = "pagerduty"
    PublishingTypeSlack      PublishingType = "slack"
    PublishingTypeWebhook    PublishingType = "webhook"
)
```
EOF

cat > TN-031/tasks.md << 'EOF'
# TN-031: Чек-лист

- [ ] 1. Создать internal/core/domain/alert.go
- [ ] 2. Создать internal/core/domain/classification.go
- [ ] 3. Создать internal/core/domain/publishing.go
- [ ] 4. Добавить validation tags: `go get github.com/go-playground/validator/v10`
- [ ] 5. Создать domain_test.go с unit тестами
- [ ] 6. Добавить JSON serialization тесты
- [ ] 7. Коммит: `feat(go): TN-031 add domain models`
EOF

# TN-032: AlertStorage interface
cat > TN-032/requirements.md << 'EOF'
# TN-032: AlertStorage Interface & PostgreSQL

## 1. Обоснование
Интерфейс для работы с хранилищем алертов и его PostgreSQL реализация.

## 2. Сценарий
Сервисы используют AlertStorage для сохранения и поиска алертов.

## 3. Требования
- AlertStorage интерфейс с CRUD операциями
- PostgreSQL реализация с pgx
- Поддержка фильтрации и pagination
- Оптимизированные индексы
- Connection pooling

## 4. Критерии приёмки
- [ ] Интерфейс определён
- [ ] PostgreSQL adapter реализован
- [ ] Pagination работает
- [ ] Индексы созданы
- [ ] Unit и integration тесты
EOF

cat > TN-032/design.md << 'EOF'
# TN-032: AlertStorage Design

## Интерфейс
```go
type AlertStorage interface {
    SaveAlert(ctx context.Context, alert *domain.Alert) error
    GetAlert(ctx context.Context, fingerprint string) (*domain.Alert, error)
    ListAlerts(ctx context.Context, filters AlertFilters) (*AlertList, error)
    UpdateAlert(ctx context.Context, alert *domain.Alert) error
    DeleteAlert(ctx context.Context, fingerprint string) error
    GetStats(ctx context.Context) (*AlertStats, error)
}

type AlertFilters struct {
    Status      *domain.AlertStatus `json:"status,omitempty"`
    Severity    *domain.Severity    `json:"severity,omitempty"`
    Namespace   *string             `json:"namespace,omitempty"`
    Labels      map[string]string   `json:"labels,omitempty"`
    TimeRange   *TimeRange          `json:"time_range,omitempty"`
    Limit       int                 `json:"limit"`
    Offset      int                 `json:"offset"`
}

type AlertList struct {
    Alerts []domain.Alert `json:"alerts"`
    Total  int            `json:"total"`
    Limit  int            `json:"limit"`
    Offset int            `json:"offset"`
}
```

## PostgreSQL Schema
```sql
CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    fingerprint VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL,
    labels JSONB,
    annotations JSONB,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP,
    generator_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_alerts_fingerprint ON alerts(fingerprint);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_starts_at ON alerts(starts_at);
CREATE INDEX idx_alerts_labels_gin ON alerts USING GIN(labels);
```
EOF

cat > TN-032/tasks.md << 'EOF'
# TN-032: Чек-лист

- [ ] 1. Создать internal/core/interfaces/storage.go
- [ ] 2. Создать internal/infrastructure/storage/postgresql.go
- [ ] 3. Добавить SQL схему в migrations/
- [ ] 4. Реализовать все методы интерфейса
- [ ] 5. Добавить connection pooling
- [ ] 6. Создать storage_test.go
- [ ] 7. Коммит: `feat(go): TN-032 implement AlertStorage`
EOF

# TN-033: Alert classification service
cat > TN-033/requirements.md << 'EOF'
# TN-033: Alert Classification Service

## 1. Обоснование
Сервис для классификации алертов через LLM с кэшированием и fallback.

## 2. Сценарий
При получении алерта сервис отправляет его в LLM и возвращает классификацию.

## 3. Требования
- Интеграция с LLM proxy
- Кэширование результатов в Redis
- Fallback при недоступности LLM
- Метрики производительности
- Configurable prompts

## 4. Критерии приёмки
- [ ] LLM интеграция работает
- [ ] Кэширование функционирует
- [ ] Fallback реализован
- [ ] Метрики собираются
- [ ] Unit тесты покрывают сценарии
EOF

cat > TN-033/design.md << 'EOF'
# TN-033: Classification Service Design

## Service Interface
```go
type AlertClassificationService interface {
    ClassifyAlert(ctx context.Context, alert *domain.Alert) (*domain.Classification, error)
    GetCachedClassification(ctx context.Context, fingerprint string) (*domain.Classification, error)
    GetStats(ctx context.Context) (*ClassificationStats, error)
}

type ClassificationService struct {
    llmClient    LLMClient
    cache        cache.Cache
    storage      AlertStorage
    logger       *slog.Logger
    metrics      *prometheus.CounterVec
    prompts      *PromptManager
}

func (s *ClassificationService) ClassifyAlert(ctx context.Context, alert *domain.Alert) (*domain.Classification, error) {
    // Check cache first
    if cached := s.getCached(alert.Fingerprint); cached != nil {
        return cached, nil
    }

    // Call LLM
    result, err := s.llmClient.Classify(ctx, alert, s.prompts.GetPrompt("default"))
    if err != nil {
        // Fallback to rule-based classification
        return s.fallbackClassification(alert), nil
    }

    // Cache result
    s.cache.Set(ctx, s.cacheKey(alert.Fingerprint), result, 1*time.Hour)

    return result, nil
}
```

## LLM Client Interface
```go
type LLMClient interface {
    Classify(ctx context.Context, alert *domain.Alert, prompt string) (*domain.Classification, error)
    GetModels(ctx context.Context) ([]string, error)
    HealthCheck(ctx context.Context) error
}
```
EOF

cat > TN-033/tasks.md << 'EOF'
# TN-033: Чек-лист

- [ ] 1. Создать internal/core/services/classification.go
- [ ] 2. Создать internal/infrastructure/llm/client.go
- [ ] 3. Реализовать LLMClient интерфейс
- [ ] 4. Добавить кэширование через Redis
- [ ] 5. Реализовать fallback classification
- [ ] 6. Добавить Prometheus метрики
- [ ] 7. Создать classification_test.go
- [ ] 8. Коммит: `feat(go): TN-033 implement classification service`
EOF

# TN-034: Enrichment mode system
cat > TN-034/requirements.md << 'EOF'
# TN-034: Enrichment Mode System

## 1. Обоснование
Система переключения между transparent и enriched режимами обработки.

## 2. Сценарий
Администратор переключает режим, и система меняет поведение обработки алертов.

## 3. Требования
- Два режима: transparent, enriched
- Переключение через API
- Сохранение состояния в Redis
- Метрики по режимам
- Graceful переключение

## 4. Критерии приёмки
- [ ] Режимы реализованы
- [ ] API переключения работает
- [ ] Состояние персистентно
- [ ] Метрики собираются
- [ ] Тесты покрывают переключения
EOF

cat > TN-034/design.md << 'EOF'
# TN-034: Enrichment Mode Design

## Mode Manager
```go
type EnrichmentMode string
const (
    EnrichmentModeTransparent EnrichmentMode = "transparent"
    EnrichmentModeEnriched    EnrichmentMode = "enriched"
)

type EnrichmentModeManager interface {
    GetMode(ctx context.Context) (EnrichmentMode, error)
    SetMode(ctx context.Context, mode EnrichmentMode) error
    GetStats(ctx context.Context) (*EnrichmentStats, error)
}

type enrichmentModeManager struct {
    cache   cache.Cache
    logger  *slog.Logger
    metrics *prometheus.CounterVec
}

func (m *enrichmentModeManager) GetMode(ctx context.Context) (EnrichmentMode, error) {
    mode, err := m.cache.Get(ctx, "enrichment:mode")
    if err != nil {
        // Default to enriched mode
        return EnrichmentModeEnriched, nil
    }
    return EnrichmentMode(mode.(string)), nil
}
```

## Processing Logic
```go
func (s *WebhookService) ProcessAlert(ctx context.Context, alert *domain.Alert) error {
    mode, _ := s.enrichmentManager.GetMode(ctx)

    switch mode {
    case EnrichmentModeTransparent:
        return s.processTransparent(ctx, alert)
    case EnrichmentModeEnriched:
        return s.processEnriched(ctx, alert)
    default:
        return s.processEnriched(ctx, alert)
    }
}

func (s *WebhookService) processTransparent(ctx context.Context, alert *domain.Alert) error {
    // Store alert without classification
    return s.storage.SaveAlert(ctx, alert)
}

func (s *WebhookService) processEnriched(ctx context.Context, alert *domain.Alert) error {
    // Classify alert with LLM
    classification, err := s.classificationService.ClassifyAlert(ctx, alert)
    if err != nil {
        s.logger.Warn("Classification failed", "error", err)
    }

    // Store alert with classification
    alert.Classification = classification
    return s.storage.SaveAlert(ctx, alert)
}
```
EOF

cat > TN-034/tasks.md << 'EOF'
# TN-034: Чек-лист

- [ ] 1. Создать internal/core/services/enrichment.go
- [ ] 2. Реализовать EnrichmentModeManager
- [ ] 3. Добавить API endpoints для режимов
- [ ] 4. Интегрировать в webhook processing
- [ ] 5. Добавить метрики переключений
- [ ] 6. Создать enrichment_test.go
- [ ] 7. Коммит: `feat(go): TN-034 implement enrichment modes`
EOF

# TN-035: Alert filtering engine
cat > TN-035/requirements.md << 'EOF'
# TN-035: Alert Filtering Engine

## 1. Обоснование
Система фильтрации алертов по severity, namespace, labels и другим критериям.

## 2. Сценарий
При запросе истории алертов применяются фильтры для получения релевантных данных.

## 3. Требования
- Фильтрация по severity, confidence, namespace
- Label-based filtering
- Time range filtering
- Composable filters
- Performance optimized

## 4. Критерии приёмки
- [ ] Все типы фильтров работают
- [ ] Фильтры композируются
- [ ] Performance приемлемый
- [ ] Unit тесты для всех фильтров
EOF

cat > TN-035/design.md << 'EOF'
# TN-035: Filter Engine Design

## Filter Interface
```go
type AlertFilter interface {
    Apply(ctx context.Context, query *AlertQuery) *AlertQuery
    Validate() error
}

type AlertQuery struct {
    BaseQuery string
    Args      []interface{}
    Filters   []string
    Joins     []string
    OrderBy   string
    Limit     int
    Offset    int
}

// Severity Filter
type SeverityFilter struct {
    Severities []domain.Severity `json:"severities"`
}

func (f *SeverityFilter) Apply(ctx context.Context, query *AlertQuery) *AlertQuery {
    if len(f.Severities) == 0 {
        return query
    }

    placeholders := make([]string, len(f.Severities))
    for i, severity := range f.Severities {
        placeholders[i] = fmt.Sprintf("$%d", len(query.Args)+1)
        query.Args = append(query.Args, severity)
    }

    query.Filters = append(query.Filters,
        fmt.Sprintf("c.severity IN (%s)", strings.Join(placeholders, ",")))
    query.Joins = append(query.Joins, "LEFT JOIN classifications c ON a.fingerprint = c.fingerprint")

    return query
}

// Label Filter
type LabelFilter struct {
    Labels map[string]string `json:"labels"`
}

func (f *LabelFilter) Apply(ctx context.Context, query *AlertQuery) *AlertQuery {
    for key, value := range f.Labels {
        query.Filters = append(query.Filters,
            fmt.Sprintf("a.labels->>'%s' = $%d", key, len(query.Args)+1))
        query.Args = append(query.Args, value)
    }
    return query
}

// Filter Engine
type FilterEngine struct {
    logger *slog.Logger
}

func (e *FilterEngine) BuildQuery(ctx context.Context, filters []AlertFilter) (*AlertQuery, error) {
    query := &AlertQuery{
        BaseQuery: "SELECT a.* FROM alerts a",
        Args:      []interface{}{},
        Filters:   []string{},
        Joins:     []string{},
        OrderBy:   "a.created_at DESC",
    }

    for _, filter := range filters {
        if err := filter.Validate(); err != nil {
            return nil, err
        }
        query = filter.Apply(ctx, query)
    }

    return e.finalizeQuery(query), nil
}
```
EOF

cat > TN-035/tasks.md << 'EOF'
# TN-035: Чек-лист

- [ ] 1. Создать internal/core/services/filter.go
- [ ] 2. Реализовать AlertFilter интерфейс
- [ ] 3. Создать SeverityFilter, LabelFilter, TimeRangeFilter
- [ ] 4. Реализовать FilterEngine
- [ ] 5. Интегрировать в AlertStorage
- [ ] 6. Добавить валидацию фильтров
- [ ] 7. Создать filter_test.go
- [ ] 8. Коммит: `feat(go): TN-035 implement filtering engine`
EOF

echo "ФАЗА 4 (TN-031 до TN-035) заполнена. Продолжаем..."
