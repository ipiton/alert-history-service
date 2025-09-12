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
