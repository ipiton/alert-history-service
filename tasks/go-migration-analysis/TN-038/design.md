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
