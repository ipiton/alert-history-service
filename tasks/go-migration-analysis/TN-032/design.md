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
