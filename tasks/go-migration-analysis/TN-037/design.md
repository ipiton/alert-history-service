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
