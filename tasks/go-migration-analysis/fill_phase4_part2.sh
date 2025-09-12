#!/bin/bash

# ФАЗА 4: Core Business Logic (TN-36 до TN-45) - Часть 2

# TN-036: Alert deduplication
cat > TN-036/requirements.md << 'EOF'
# TN-036: Alert Deduplication & Fingerprinting

## 1. Обоснование
Система дедупликации алертов по fingerprint и группировка похожих алертов.

## 2. Сценарий
При получении алерта система проверяет, не является ли он дубликатом существующего.

## 3. Требования
- Fingerprint generation по Alertmanager алгоритму
- Deduplication по fingerprint
- Alert grouping по labels
- Update existing alerts при изменении статуса
- Metrics для дубликатов

## 4. Критерии приёмки
- [ ] Fingerprinting работает корректно
- [ ] Дубликаты не создаются
- [ ] Группировка функционирует
- [ ] Метрики собираются
- [ ] Unit тесты покрывают сценарии
EOF

cat > TN-036/design.md << 'EOF'
# TN-036: Deduplication Design

## Fingerprint Generator
```go
type FingerprintGenerator interface {
    Generate(alert *domain.Alert) string
    GenerateFromLabels(labels map[string]string) string
}

type alertmanagerFingerprinting struct {
    logger *slog.Logger
}

func (f *alertmanagerFingerprinting) Generate(alert *domain.Alert) string {
    // Use same algorithm as Alertmanager
    return f.GenerateFromLabels(alert.Labels)
}

func (f *alertmanagerFingerprinting) GenerateFromLabels(labels map[string]string) string {
    // Sort labels by key for consistent fingerprinting
    keys := make([]string, 0, len(labels))
    for k := range labels {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    h := fnv.New64a()
    for _, k := range keys {
        h.Write([]byte(k))
        h.Write([]byte(labels[k]))
    }

    return fmt.Sprintf("%016x", h.Sum64())
}

// Deduplication Service
type DeduplicationService interface {
    ProcessAlert(ctx context.Context, alert *domain.Alert) (*ProcessResult, error)
    GetDuplicateStats(ctx context.Context) (*DuplicateStats, error)
}

type ProcessResult struct {
    Action      ProcessAction `json:"action"`
    Alert       *domain.Alert `json:"alert"`
    ExistingID  *string       `json:"existing_id,omitempty"`
    IsUpdate    bool          `json:"is_update"`
}

type ProcessAction string
const (
    ProcessActionCreated ProcessAction = "created"
    ProcessActionUpdated ProcessAction = "updated"
    ProcessActionIgnored ProcessAction = "ignored"
)

type deduplicationService struct {
    storage     AlertStorage
    fingerprint FingerprintGenerator
    metrics     *prometheus.CounterVec
    logger      *slog.Logger
}

func (s *deduplicationService) ProcessAlert(ctx context.Context, alert *domain.Alert) (*ProcessResult, error) {
    // Generate fingerprint
    if alert.Fingerprint == "" {
        alert.Fingerprint = s.fingerprint.Generate(alert)
    }

    // Check if alert exists
    existing, err := s.storage.GetAlert(ctx, alert.Fingerprint)
    if err != nil && !errors.Is(err, ErrAlertNotFound) {
        return nil, err
    }

    if existing == nil {
        // New alert
        if err := s.storage.SaveAlert(ctx, alert); err != nil {
            return nil, err
        }
        s.metrics.WithLabelValues("created").Inc()
        return &ProcessResult{
            Action: ProcessActionCreated,
            Alert:  alert,
        }, nil
    }

    // Update existing alert if status changed
    if existing.Status != alert.Status || !existing.EndsAt.Equal(*alert.EndsAt) {
        existing.Status = alert.Status
        existing.EndsAt = alert.EndsAt
        existing.UpdatedAt = time.Now()

        if err := s.storage.UpdateAlert(ctx, existing); err != nil {
            return nil, err
        }

        s.metrics.WithLabelValues("updated").Inc()
        return &ProcessResult{
            Action:     ProcessActionUpdated,
            Alert:      existing,
            IsUpdate:   true,
        }, nil
    }

    // Duplicate - ignore
    s.metrics.WithLabelValues("ignored").Inc()
    return &ProcessResult{
        Action:      ProcessActionIgnored,
        Alert:       existing,
        ExistingID:  &existing.ID,
    }, nil
}
```
EOF

cat > TN-036/tasks.md << 'EOF'
# TN-036: Чек-лист

- [ ] 1. Создать internal/core/services/deduplication.go
- [ ] 2. Реализовать FingerprintGenerator
- [ ] 3. Создать DeduplicationService
- [ ] 4. Добавить ProcessResult типы
- [ ] 5. Интегрировать в webhook processing
- [ ] 6. Добавить Prometheus метрики
- [ ] 7. Создать deduplication_test.go
- [ ] 8. Коммит: `feat(go): TN-036 implement deduplication`
EOF

# TN-037: Alert history repository
cat > TN-037/requirements.md << 'EOF'
# TN-037: Alert History Repository

## 1. Обоснование
Repository для работы с историей алертов с поддержкой pagination и advanced queries.

## 2. Сценарий
API endpoints запрашивают историю алертов с различными фильтрами и pagination.

## 3. Требования
- Pagination с limit/offset
- Sorting по различным полям
- Advanced filtering
- Performance optimization
- Aggregate queries

## 4. Критерии приёмки
- [ ] Pagination реализован
- [ ] Сортировка работает
- [ ] Фильтрация эффективна
- [ ] Performance приемлемый
- [ ] Unit и integration тесты
EOF

cat > TN-037/design.md << 'EOF'
# TN-037: History Repository Design

## Repository Interface
```go
type AlertHistoryRepository interface {
    GetHistory(ctx context.Context, req *HistoryRequest) (*HistoryResponse, error)
    GetAlertsByFingerprint(ctx context.Context, fingerprint string) ([]*domain.Alert, error)
    GetRecentAlerts(ctx context.Context, limit int) ([]*domain.Alert, error)
    GetAggregatedStats(ctx context.Context, timeRange *TimeRange) (*AggregatedStats, error)
}

type HistoryRequest struct {
    Filters    *AlertFilters `json:"filters"`
    Pagination *Pagination   `json:"pagination"`
    Sorting    *Sorting      `json:"sorting"`
}

type HistoryResponse struct {
    Alerts     []*domain.Alert `json:"alerts"`
    Total      int64           `json:"total"`
    Page       int             `json:"page"`
    PerPage    int             `json:"per_page"`
    TotalPages int             `json:"total_pages"`
    HasNext    bool            `json:"has_next"`
    HasPrev    bool            `json:"has_prev"`
}

type Pagination struct {
    Page    int `json:"page" validate:"min=1"`
    PerPage int `json:"per_page" validate:"min=1,max=1000"`
}

type Sorting struct {
    Field string    `json:"field"`
    Order SortOrder `json:"order"`
}

type SortOrder string
const (
    SortOrderAsc  SortOrder = "asc"
    SortOrderDesc SortOrder = "desc"
)

// Repository Implementation
type alertHistoryRepository struct {
    db          *pgxpool.Pool
    filterEngine *FilterEngine
    logger      *slog.Logger
    metrics     *prometheus.HistogramVec
}

func (r *alertHistoryRepository) GetHistory(ctx context.Context, req *HistoryRequest) (*HistoryResponse, error) {
    start := time.Now()
    defer func() {
        r.metrics.WithLabelValues("get_history").Observe(time.Since(start).Seconds())
    }()

    // Build query with filters
    query, err := r.filterEngine.BuildQuery(ctx, req.Filters)
    if err != nil {
        return nil, err
    }

    // Add pagination
    offset := (req.Pagination.Page - 1) * req.Pagination.PerPage
    query.Limit = req.Pagination.PerPage
    query.Offset = offset

    // Add sorting
    if req.Sorting != nil {
        query.OrderBy = fmt.Sprintf("%s %s", req.Sorting.Field, req.Sorting.Order)
    }

    // Execute query
    finalQuery := r.buildFinalQuery(query)
    rows, err := r.db.Query(ctx, finalQuery, query.Args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var alerts []*domain.Alert
    for rows.Next() {
        alert := &domain.Alert{}
        if err := r.scanAlert(rows, alert); err != nil {
            return nil, err
        }
        alerts = append(alerts, alert)
    }

    // Get total count
    total, err := r.getTotalCount(ctx, query)
    if err != nil {
        return nil, err
    }

    // Build response
    totalPages := int(math.Ceil(float64(total) / float64(req.Pagination.PerPage)))

    return &HistoryResponse{
        Alerts:     alerts,
        Total:      total,
        Page:       req.Pagination.Page,
        PerPage:    req.Pagination.PerPage,
        TotalPages: totalPages,
        HasNext:    req.Pagination.Page < totalPages,
        HasPrev:    req.Pagination.Page > 1,
    }, nil
}
```

## Optimized Queries
```sql
-- History query with joins
SELECT
    a.id, a.fingerprint, a.status, a.labels, a.annotations,
    a.starts_at, a.ends_at, a.generator_url, a.created_at, a.updated_at,
    c.severity, c.confidence, c.reasoning, c.recommendations
FROM alerts a
LEFT JOIN classifications c ON a.fingerprint = c.fingerprint
WHERE a.status = $1
    AND a.created_at >= $2
    AND a.labels->>'namespace' = $3
ORDER BY a.created_at DESC
LIMIT $4 OFFSET $5;

-- Count query
SELECT COUNT(*)
FROM alerts a
WHERE a.status = $1
    AND a.created_at >= $2
    AND a.labels->>'namespace' = $3;
```
EOF

cat > TN-037/tasks.md << 'EOF'
# TN-037: Чек-лист

- [ ] 1. Создать internal/core/interfaces/history.go
- [ ] 2. Создать internal/infrastructure/repository/history.go
- [ ] 3. Реализовать HistoryRequest/Response типы
- [ ] 4. Добавить pagination логику
- [ ] 5. Оптимизировать SQL queries
- [ ] 6. Добавить performance метрики
- [ ] 7. Создать history_test.go
- [ ] 8. Коммит: `feat(go): TN-037 implement history repository`
EOF

# TN-038: Alert analytics service
cat > TN-038/requirements.md << 'EOF'
# TN-038: Alert Analytics Service

## 1. Обоснование
Сервис для аналитики алертов: топ алерты, flapping detection, статистики.

## 2. Сценарий
Dashboard отображает аналитику: самые частые алерты, флапающие алерты, тренды.

## 3. Требования
- Top alerts по частоте
- Flapping detection
- Time-based trends
- Severity distribution
- Performance optimized queries

## 4. Критерии приёмки
- [ ] Top alerts работают
- [ ] Flapping detection функционирует
- [ ] Тренды вычисляются
- [ ] Performance приемлемый
- [ ] Unit тесты покрывают логику
EOF

cat > TN-038/design.md << 'EOF'
# TN-038: Analytics Service Design

## Service Interface
```go
type AlertAnalyticsService interface {
    GetTopAlerts(ctx context.Context, req *TopAlertsRequest) (*TopAlertsResponse, error)
    GetFlappingAlerts(ctx context.Context, req *FlappingRequest) (*FlappingResponse, error)
    GetTrends(ctx context.Context, req *TrendsRequest) (*TrendsResponse, error)
    GetSeverityDistribution(ctx context.Context, req *DistributionRequest) (*DistributionResponse, error)
    GetSummary(ctx context.Context, req *SummaryRequest) (*SummaryResponse, error)
}

// Top Alerts
type TopAlertsRequest struct {
    TimeRange *TimeRange `json:"time_range"`
    Limit     int        `json:"limit" validate:"min=1,max=100"`
    GroupBy   string     `json:"group_by"` // "alertname", "namespace", "severity"
}

type TopAlertsResponse struct {
    Alerts    []*TopAlert `json:"alerts"`
    TimeRange *TimeRange  `json:"time_range"`
}

type TopAlert struct {
    AlertName   string  `json:"alert_name"`
    Namespace   string  `json:"namespace,omitempty"`
    Severity    string  `json:"severity,omitempty"`
    Count       int64   `json:"count"`
    Frequency   float64 `json:"frequency"` // alerts per hour
    LastSeen    time.Time `json:"last_seen"`
    Labels      map[string]string `json:"labels"`
}

// Flapping Detection
type FlappingRequest struct {
    TimeRange     *TimeRange `json:"time_range"`
    MinFlaps      int        `json:"min_flaps" validate:"min=3"`
    FlappingWindow time.Duration `json:"flapping_window"`
}

type FlappingResponse struct {
    FlappingAlerts []*FlappingAlert `json:"flapping_alerts"`
    Summary        *FlappingSummary `json:"summary"`
}

type FlappingAlert struct {
    Fingerprint   string    `json:"fingerprint"`
    AlertName     string    `json:"alert_name"`
    FlapsCount    int       `json:"flaps_count"`
    FlapsPerHour  float64   `json:"flaps_per_hour"`
    FirstFlap     time.Time `json:"first_flap"`
    LastFlap      time.Time `json:"last_flap"`
    StateChanges  []*StateChange `json:"state_changes"`
}

type StateChange struct {
    Timestamp time.Time           `json:"timestamp"`
    FromState domain.AlertStatus  `json:"from_state"`
    ToState   domain.AlertStatus  `json:"to_state"`
}

// Implementation
type alertAnalyticsService struct {
    db      *pgxpool.Pool
    cache   cache.Cache
    logger  *slog.Logger
    metrics *prometheus.HistogramVec
}

func (s *alertAnalyticsService) GetTopAlerts(ctx context.Context, req *TopAlertsRequest) (*TopAlertsResponse, error) {
    cacheKey := s.buildCacheKey("top_alerts", req)
    if cached := s.getCached(ctx, cacheKey); cached != nil {
        return cached.(*TopAlertsResponse), nil
    }

    query := `
        SELECT
            labels->>'alertname' as alert_name,
            labels->>'namespace' as namespace,
            COUNT(*) as count,
            MAX(created_at) as last_seen,
            labels
        FROM alerts
        WHERE created_at >= $1 AND created_at <= $2
        GROUP BY labels->>'alertname', labels->>'namespace', labels
        ORDER BY count DESC
        LIMIT $3
    `

    rows, err := s.db.Query(ctx, query, req.TimeRange.Start, req.TimeRange.End, req.Limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var alerts []*TopAlert
    totalHours := req.TimeRange.End.Sub(req.TimeRange.Start).Hours()

    for rows.Next() {
        alert := &TopAlert{}
        var labelsJSON []byte

        if err := rows.Scan(
            &alert.AlertName, &alert.Namespace, &alert.Count,
            &alert.LastSeen, &labelsJSON,
        ); err != nil {
            return nil, err
        }

        json.Unmarshal(labelsJSON, &alert.Labels)
        alert.Frequency = float64(alert.Count) / totalHours
        alerts = append(alerts, alert)
    }

    response := &TopAlertsResponse{
        Alerts:    alerts,
        TimeRange: req.TimeRange,
    }

    // Cache for 5 minutes
    s.cache.Set(ctx, cacheKey, response, 5*time.Minute)

    return response, nil
}

func (s *alertAnalyticsService) GetFlappingAlerts(ctx context.Context, req *FlappingRequest) (*FlappingResponse, error) {
    // Complex query to detect state changes
    query := `
        WITH state_changes AS (
            SELECT
                fingerprint,
                status,
                created_at,
                LAG(status) OVER (PARTITION BY fingerprint ORDER BY created_at) as prev_status,
                labels->>'alertname' as alert_name
            FROM alerts
            WHERE created_at >= $1 AND created_at <= $2
        ),
        flaps AS (
            SELECT
                fingerprint,
                alert_name,
                COUNT(*) as flaps_count,
                MIN(created_at) as first_flap,
                MAX(created_at) as last_flap
            FROM state_changes
            WHERE status != prev_status AND prev_status IS NOT NULL
            GROUP BY fingerprint, alert_name
            HAVING COUNT(*) >= $3
        )
        SELECT * FROM flaps
        ORDER BY flaps_count DESC
    `

    // Implementation continues...
}
```
EOF

cat > TN-038/tasks.md << 'EOF'
# TN-038: Чек-лист

- [ ] 1. Создать internal/core/services/analytics.go
- [ ] 2. Реализовать TopAlerts запросы
- [ ] 3. Добавить Flapping detection
- [ ] 4. Создать Trends calculation
- [ ] 5. Добавить кэширование результатов
- [ ] 6. Оптимизировать SQL queries
- [ ] 7. Создать analytics_test.go
- [ ] 8. Коммит: `feat(go): TN-038 implement analytics service`
EOF

echo "ФАЗА 4 часть 2 (TN-036 до TN-038) заполнена. Продолжаем..."
