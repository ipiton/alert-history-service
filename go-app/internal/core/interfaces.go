package core

import (
	"context"
	"time"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
	SeverityNoise    AlertSeverity = "noise"
)

// AlertStatus represents alert status values
type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
)

// PublishingFormat represents publishing format options
type PublishingFormat string

const (
	FormatAlertmanager PublishingFormat = "alertmanager"
	FormatRootly       PublishingFormat = "rootly"
	FormatPagerDuty    PublishingFormat = "pagerduty"
	FormatSlack        PublishingFormat = "slack"
	FormatWebhook      PublishingFormat = "webhook"
)

// Alert represents alert data model
type Alert struct {
	Fingerprint  string            `json:"fingerprint" validate:"required"`
	AlertName    string            `json:"alert_name" validate:"required"`
	Status       AlertStatus       `json:"status" validate:"required,oneof=firing resolved"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"starts_at" validate:"required"`
	EndsAt       *time.Time        `json:"ends_at,omitempty"`
	GeneratorURL *string           `json:"generator_url,omitempty" validate:"omitempty,url"`
	Timestamp    *time.Time        `json:"timestamp,omitempty"`
}

// Namespace returns alert namespace from labels
func (a *Alert) Namespace() *string {
	if ns, ok := a.Labels["namespace"]; ok {
		return &ns
	}
	return nil
}

// Severity returns alert severity from labels
func (a *Alert) Severity() *string {
	if sev, ok := a.Labels["severity"]; ok {
		return &sev
	}
	return nil
}

// ClassificationResult represents LLM classification result
type ClassificationResult struct {
	Severity        AlertSeverity  `json:"severity" validate:"required,oneof=critical warning info noise"`
	Confidence      float64        `json:"confidence" validate:"gte=0,lte=1"`
	Reasoning       string         `json:"reasoning" validate:"required"`
	Recommendations []string       `json:"recommendations"`
	ProcessingTime  float64        `json:"processing_time" validate:"gte=0"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// PublishingTarget represents publishing target configuration
type PublishingTarget struct {
	Name         string            `json:"name" validate:"required"`
	Type         string            `json:"type" validate:"required"`
	URL          string            `json:"url" validate:"required,url"`
	Enabled      bool              `json:"enabled"`
	FilterConfig map[string]any    `json:"filter_config"`
	Headers      map[string]string `json:"headers"`
	Format       PublishingFormat  `json:"format" validate:"required,oneof=alertmanager rootly pagerduty slack webhook"`
}

// EnrichedAlert represents alert enriched with classification data
type EnrichedAlert struct {
	Alert               *Alert                `json:"alert"`
	Classification      *ClassificationResult `json:"classification,omitempty"`
	EnrichmentMetadata  map[string]any        `json:"enrichment_metadata,omitempty"`
	ProcessingTimestamp *time.Time            `json:"processing_timestamp,omitempty"`
}

// Database interfaces following SOLID principles

// AlertStorage interface for alert storage operations
type AlertStorage interface {
	SaveAlert(ctx context.Context, alert *Alert) error
	GetAlertByFingerprint(ctx context.Context, fingerprint string) (*Alert, error)
	GetAlerts(ctx context.Context, filters map[string]any, limit, offset int) ([]*Alert, error)
	CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error)
}

// ClassificationStorage interface for classification storage operations
type ClassificationStorage interface {
	SaveClassification(ctx context.Context, fingerprint string, result *ClassificationResult) error
	GetClassification(ctx context.Context, fingerprint string) (*ClassificationResult, error)
}

// PublishingLogStorage interface for publishing log storage
type PublishingLogStorage interface {
	LogPublishingAttempt(ctx context.Context, fingerprint, targetName string, success bool, errorMessage *string, processingTime *float64) error
	GetPublishingHistory(ctx context.Context, fingerprint string) ([]*PublishingLog, error)
}

// PublishingLog represents publishing attempt log
type PublishingLog struct {
	ID             string    `json:"id"`
	Fingerprint    string    `json:"fingerprint"`
	TargetName     string    `json:"target_name"`
	Success        bool      `json:"success"`
	ErrorMessage   *string   `json:"error_message,omitempty"`
	ProcessingTime *float64  `json:"processing_time,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Combined Database interface for full functionality
type Database interface {
	// Core database operations
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Health(ctx context.Context) error

	// Alert operations
	AlertStorage

	// Classification operations
	ClassificationStorage

	// Publishing operations
	PublishingLogStorage

	// Migration operations
	MigrateUp(ctx context.Context) error
	MigrateDown(ctx context.Context, steps int) error

	// Utility operations
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// Cache interfaces

// Cache interface for generic caching
type Cache interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// DistributedLock interface for distributed locking
type DistributedLock interface {
	AcquireLock(ctx context.Context, key string, timeout time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error
}

// LLM Service interfaces

// LLMClient interface for LLM communication
type LLMClient interface {
	ClassifyAlert(ctx context.Context, alert *Alert, context map[string]any) (*ClassificationResult, error)
	GenerateRecommendations(ctx context.Context, alert *Alert, classification *ClassificationResult) ([]string, error)
}

// AlertClassifier interface for alert classification service
type AlertClassifier interface {
	Classify(ctx context.Context, alert *Alert) (*ClassificationResult, error)
}

// Publishing interfaces

// AlertFormatter interface for alert formatting
type AlertFormatter interface {
	FormatAlert(ctx context.Context, enrichedAlert *EnrichedAlert, targetFormat PublishingFormat) (map[string]any, error)
}

// AlertPublisher interface for alert publishing
type AlertPublisher interface {
	PublishAlert(ctx context.Context, enrichedAlert *EnrichedAlert, target *PublishingTarget) error
}

// FilterEngine interface for alert filtering
type FilterEngine interface {
	ShouldPublish(ctx context.Context, enrichedAlert *EnrichedAlert, target *PublishingTarget) (bool, error)
}

// Configuration Management interfaces

// ConfigurationManager interface for configuration management
type ConfigurationManager interface {
	GetConfig(ctx context.Context, key string, defaultValue any) (any, error)
	GetAllConfigs(ctx context.Context) (map[string]any, error)
	ReloadConfig(ctx context.Context) error
}

// SecretsManager interface for secrets management
type SecretsManager interface {
	GetSecret(ctx context.Context, key string) (string, error)
	ListSecrets(ctx context.Context, labelSelector string) (map[string]map[string]string, error)
}

// TargetDiscovery interface for dynamic target discovery
type TargetDiscovery interface {
	DiscoverTargets(ctx context.Context) ([]*PublishingTarget, error)
	RefreshTargets(ctx context.Context) error
}

// Health Check interface

// HealthChecker interface for health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context) (map[string]any, error)
	CheckReadiness(ctx context.Context) (map[string]any, error)
}

// Metrics interface

// MetricsCollector interface for metrics collection
type MetricsCollector interface {
	IncrementCounter(ctx context.Context, name string, labels map[string]string)
	SetGauge(ctx context.Context, name string, value float64, labels map[string]string)
	ObserveHistogram(ctx context.Context, name string, value float64, labels map[string]string)
}

// Event Processing interface

// EventProcessor interface for event processing strategies
type EventProcessor interface {
	ProcessEvent(ctx context.Context, eventData map[string]any) error
	CanHandle(eventType string) bool
}

// Repository Pattern interface

// Repository interface for generic repository operations
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id string) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters map[string]any, limit, offset int) ([]*T, error)
}
