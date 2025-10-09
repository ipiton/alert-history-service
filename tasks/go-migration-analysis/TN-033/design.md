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
