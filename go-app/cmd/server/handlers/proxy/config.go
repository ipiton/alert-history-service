// Package proxy provides configuration for proxy webhook handler.
package proxy

import (
	"fmt"
	"time"
)

// ProxyWebhookConfig holds configuration for the proxy webhook handler.
type ProxyWebhookConfig struct {
	// HTTP configuration
	MaxRequestSize   int           `yaml:"max_request_size"`   // Max payload size (10MB)
	RequestTimeout   time.Duration `yaml:"request_timeout"`    // Request timeout (30s)
	MaxAlertsPerReq  int           `yaml:"max_alerts_per_req"` // Max alerts per request (100)
	EnableMetrics    bool          `yaml:"enable_metrics"`     // Enable Prometheus metrics
	EnableValidation bool          `yaml:"enable_validation"`  // Enable request validation

	// Pipeline configuration
	Classification ClassificationPipelineConfig `yaml:"classification"`
	Filtering      FilteringPipelineConfig      `yaml:"filtering"`
	Publishing     PublishingPipelineConfig     `yaml:"publishing"`

	// Behavior configuration
	EnableClassification bool `yaml:"enable_classification"` // Enable LLM classification
	EnableFiltering      bool `yaml:"enable_filtering"`      // Enable filtering
	EnablePublishing     bool `yaml:"enable_publishing"`     // Enable publishing
	ContinueOnError      bool `yaml:"continue_on_error"`     // Continue on partial failures

	// Timeouts
	ClassificationTimeout time.Duration `yaml:"classification_timeout"` // 5s
	FilteringTimeout      time.Duration `yaml:"filtering_timeout"`      // 1s
	PublishingTimeout     time.Duration `yaml:"publishing_timeout"`     // 10s (5s per target)

	// Concurrency
	MaxConcurrentAlerts  int `yaml:"max_concurrent_alerts"`  // 10 (within batch)
	MaxPublishingTargets int `yaml:"max_publishing_targets"` // 10 (concurrent publishes)
}

// ClassificationPipelineConfig holds classification pipeline configuration.
type ClassificationPipelineConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Timeout         time.Duration `yaml:"timeout"`
	CacheTTL        time.Duration `yaml:"cache_ttl"`
	FallbackEnabled bool          `yaml:"fallback_enabled"`
}

// FilteringPipelineConfig holds filtering pipeline configuration.
type FilteringPipelineConfig struct {
	Enabled       bool   `yaml:"enabled"`
	DefaultAction string `yaml:"default_action"` // "allow" or "deny"
	RulesFile     string `yaml:"rules_file"`
}

// PublishingPipelineConfig holds publishing pipeline configuration.
type PublishingPipelineConfig struct {
	Enabled          bool          `yaml:"enabled"`
	Parallel         bool          `yaml:"parallel"`
	TimeoutPerTarget time.Duration `yaml:"timeout_per_target"`
	RetryEnabled     bool          `yaml:"retry_enabled"`
	RetryMaxAttempts int           `yaml:"retry_max_attempts"`
	DLQEnabled       bool          `yaml:"dlq_enabled"`
}

// DefaultProxyWebhookConfig returns default configuration.
func DefaultProxyWebhookConfig() *ProxyWebhookConfig {
	return &ProxyWebhookConfig{
		// HTTP defaults
		MaxRequestSize:   10 * 1024 * 1024, // 10MB
		RequestTimeout:   30 * time.Second,
		MaxAlertsPerReq:  100,
		EnableMetrics:    true,
		EnableValidation: true,

		// Pipeline defaults
		Classification: ClassificationPipelineConfig{
			Enabled:         true,
			Timeout:         5 * time.Second,
			CacheTTL:        15 * time.Minute,
			FallbackEnabled: true,
		},
		Filtering: FilteringPipelineConfig{
			Enabled:       true,
			DefaultAction: "allow",
			RulesFile:     "",
		},
		Publishing: PublishingPipelineConfig{
			Enabled:          true,
			Parallel:         true,
			TimeoutPerTarget: 5 * time.Second,
			RetryEnabled:     true,
			RetryMaxAttempts: 3,
			DLQEnabled:       true,
		},

		// Behavior defaults
		EnableClassification: true,
		EnableFiltering:      true,
		EnablePublishing:     true,
		ContinueOnError:      true,

		// Timeout defaults
		ClassificationTimeout: 5 * time.Second,
		FilteringTimeout:      1 * time.Second,
		PublishingTimeout:     10 * time.Second,

		// Concurrency defaults
		MaxConcurrentAlerts:  10,
		MaxPublishingTargets: 10,
	}
}

// Validate validates the configuration.
func (c *ProxyWebhookConfig) Validate() error {
	if c.MaxRequestSize <= 0 {
		return fmt.Errorf("max_request_size must be positive")
	}
	if c.MaxRequestSize > 100*1024*1024 {
		return fmt.Errorf("max_request_size cannot exceed 100MB")
	}
	if c.MaxAlertsPerReq <= 0 {
		return fmt.Errorf("max_alerts_per_req must be positive")
	}
	if c.MaxAlertsPerReq > 1000 {
		return fmt.Errorf("max_alerts_per_req cannot exceed 1000")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}
	if c.ClassificationTimeout <= 0 {
		return fmt.Errorf("classification_timeout must be positive")
	}
	if c.FilteringTimeout <= 0 {
		return fmt.Errorf("filtering_timeout must be positive")
	}
	if c.PublishingTimeout <= 0 {
		return fmt.Errorf("publishing_timeout must be positive")
	}
	if c.MaxConcurrentAlerts <= 0 {
		return fmt.Errorf("max_concurrent_alerts must be positive")
	}
	if c.MaxPublishingTargets <= 0 {
		return fmt.Errorf("max_publishing_targets must be positive")
	}

	// Validate filtering default action
	if c.Filtering.DefaultAction != "allow" && c.Filtering.DefaultAction != "deny" {
		return fmt.Errorf("filtering default_action must be 'allow' or 'deny'")
	}

	return nil
}
