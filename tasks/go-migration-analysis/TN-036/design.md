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
