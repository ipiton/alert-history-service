package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultProxyWebhookConfig tests default configuration creation
func TestDefaultProxyWebhookConfig(t *testing.T) {
	config := DefaultProxyWebhookConfig()

	require.NotNil(t, config)

	// HTTP config
	assert.Equal(t, 10*1024*1024, config.MaxRequestSize) // 10MB
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
	assert.Equal(t, 100, config.MaxAlertsPerRequest)

	// Pipeline toggles
	assert.True(t, config.EnableClassification)
	assert.True(t, config.EnableFiltering)
	assert.True(t, config.EnablePublishing)

	// Pipeline configs
	assert.NotNil(t, config.Classification)
	assert.NotNil(t, config.Filtering)
	assert.NotNil(t, config.Publishing)

	// Classification defaults
	assert.Equal(t, 5*time.Second, config.Classification.Timeout)
	assert.Equal(t, 15*time.Minute, config.Classification.CacheTTL)
	assert.True(t, config.Classification.EnableFallback)

	// Filtering defaults
	assert.Equal(t, 1*time.Second, config.Filtering.Timeout)
	assert.Equal(t, string(FilterActionAllow), config.Filtering.DefaultAction)

	// Publishing defaults
	assert.Equal(t, 10*time.Second, config.Publishing.Timeout)
	assert.True(t, config.Publishing.EnableParallel)
	assert.Equal(t, 3, config.Publishing.MaxRetries)
	assert.True(t, config.Publishing.EnableDLQ)

	// Timeouts
	assert.Equal(t, 5*time.Second, config.ClassificationTimeout)
	assert.Equal(t, 1*time.Second, config.FilteringTimeout)
	assert.Equal(t, 10*time.Second, config.PublishingTimeout)

	// Concurrency
	assert.Equal(t, 10, config.MaxConcurrentAlerts)
	assert.Equal(t, 10, config.MaxConcurrentTargets)

	// Error handling
	assert.False(t, config.ContinueOnError)
}

// TestProxyWebhookConfig_Validate tests configuration validation
func TestProxyWebhookConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      ProxyWebhookConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			config:      *DefaultProxyWebhookConfig(),
			expectError: false,
		},
		{
			name: "invalid max request size - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:      0, // Invalid
				RequestTimeout:      30 * time.Second,
				MaxAlertsPerRequest: 100,
			},
			expectError: true,
			errorMsg:    "max_request_size",
		},
		{
			name: "invalid max request size - negative",
			config: ProxyWebhookConfig{
				MaxRequestSize:      -1, // Invalid
				RequestTimeout:      30 * time.Second,
				MaxAlertsPerRequest: 100,
			},
			expectError: true,
			errorMsg:    "max_request_size",
		},
		{
			name: "invalid request timeout - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:      10 * 1024 * 1024,
				RequestTimeout:      0, // Invalid
				MaxAlertsPerRequest: 100,
			},
			expectError: true,
			errorMsg:    "request_timeout",
		},
		{
			name: "invalid max alerts per request - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:      10 * 1024 * 1024,
				RequestTimeout:      30 * time.Second,
				MaxAlertsPerRequest: 0, // Invalid
			},
			expectError: true,
			errorMsg:    "max_alerts_per_request",
		},
		{
			name: "invalid classification timeout - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:        10 * 1024 * 1024,
				RequestTimeout:        30 * time.Second,
				MaxAlertsPerRequest:   100,
				ClassificationTimeout: 0, // Invalid
			},
			expectError: true,
			errorMsg:    "classification_timeout",
		},
		{
			name: "invalid filtering timeout - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:        10 * 1024 * 1024,
				RequestTimeout:        30 * time.Second,
				MaxAlertsPerRequest:   100,
				ClassificationTimeout: 5 * time.Second,
				FilteringTimeout:      0, // Invalid
			},
			expectError: true,
			errorMsg:    "filtering_timeout",
		},
		{
			name: "invalid publishing timeout - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:        10 * 1024 * 1024,
				RequestTimeout:        30 * time.Second,
				MaxAlertsPerRequest:   100,
				ClassificationTimeout: 5 * time.Second,
				FilteringTimeout:      1 * time.Second,
				PublishingTimeout:     0, // Invalid
			},
			expectError: true,
			errorMsg:    "publishing_timeout",
		},
		{
			name: "invalid max concurrent alerts - zero",
			config: ProxyWebhookConfig{
				MaxRequestSize:        10 * 1024 * 1024,
				RequestTimeout:        30 * time.Second,
				MaxAlertsPerRequest:   100,
				ClassificationTimeout: 5 * time.Second,
				FilteringTimeout:      1 * time.Second,
				PublishingTimeout:     10 * time.Second,
				MaxConcurrentAlerts:   0, // Invalid
			},
			expectError: true,
			errorMsg:    "max_concurrent_alerts",
		},
		{
			name: "valid custom config",
			config: ProxyWebhookConfig{
				MaxRequestSize:        20 * 1024 * 1024, // 20MB
				RequestTimeout:        60 * time.Second,
				MaxAlertsPerRequest:   200,
				ClassificationTimeout: 10 * time.Second,
				FilteringTimeout:      2 * time.Second,
				PublishingTimeout:     20 * time.Second,
				MaxConcurrentAlerts:   20,
				MaxConcurrentTargets:  15,
				EnableClassification:  true,
				EnableFiltering:       true,
				EnablePublishing:      true,
				ContinueOnError:       true,
				Classification: &ClassificationPipelineConfig{
					Timeout:        10 * time.Second,
					CacheTTL:       30 * time.Minute,
					EnableFallback: true,
				},
				Filtering: &FilteringPipelineConfig{
					Timeout:       2 * time.Second,
					DefaultAction: string(FilterActionAllow),
				},
				Publishing: &PublishingPipelineConfig{
					Timeout:        20 * time.Second,
					EnableParallel: true,
					MaxRetries:     5,
					EnableDLQ:      true,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClassificationPipelineConfig tests classification pipeline configuration
func TestClassificationPipelineConfig(t *testing.T) {
	config := &ClassificationPipelineConfig{
		Timeout:        5 * time.Second,
		CacheTTL:       15 * time.Minute,
		EnableFallback: true,
	}

	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.Equal(t, 15*time.Minute, config.CacheTTL)
	assert.True(t, config.EnableFallback)
}

// TestFilteringPipelineConfig tests filtering pipeline configuration
func TestFilteringPipelineConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        FilteringPipelineConfig
		expectedValid bool
	}{
		{
			name: "allow default action",
			config: FilteringPipelineConfig{
				Timeout:       1 * time.Second,
				DefaultAction: string(FilterActionAllow),
			},
			expectedValid: true,
		},
		{
			name: "deny default action",
			config: FilteringPipelineConfig{
				Timeout:       1 * time.Second,
				DefaultAction: string(FilterActionDeny),
			},
			expectedValid: true,
		},
		{
			name: "with rules file",
			config: FilteringPipelineConfig{
				Timeout:       1 * time.Second,
				DefaultAction: string(FilterActionAllow),
				RulesFile:     "/etc/alerting/filter-rules.yaml",
			},
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedValid, tt.config.DefaultAction != "")
		})
	}
}

// TestPublishingPipelineConfig tests publishing pipeline configuration
func TestPublishingPipelineConfig(t *testing.T) {
	tests := []struct {
		name   string
		config PublishingPipelineConfig
	}{
		{
			name: "parallel publishing enabled",
			config: PublishingPipelineConfig{
				Timeout:          10 * time.Second,
				EnableParallel:   true,
				TimeoutPerTarget: 5 * time.Second,
				MaxRetries:       3,
				EnableDLQ:        true,
			},
		},
		{
			name: "sequential publishing",
			config: PublishingPipelineConfig{
				Timeout:          20 * time.Second,
				EnableParallel:   false,
				TimeoutPerTarget: 10 * time.Second,
				MaxRetries:       5,
				EnableDLQ:        false,
			},
		},
		{
			name: "no retries",
			config: PublishingPipelineConfig{
				Timeout:          10 * time.Second,
				EnableParallel:   true,
				TimeoutPerTarget: 5 * time.Second,
				MaxRetries:       0,
				EnableDLQ:        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.GreaterOrEqual(t, tt.config.Timeout, time.Duration(0))
			assert.GreaterOrEqual(t, tt.config.MaxRetries, 0)
		})
	}
}

// TestProxyWebhookConfig_TimeoutHierarchy tests timeout configuration hierarchy
func TestProxyWebhookConfig_TimeoutHierarchy(t *testing.T) {
	config := DefaultProxyWebhookConfig()

	// Request timeout should be >= sum of pipeline timeouts
	totalPipelineTimeout := config.ClassificationTimeout +
		config.FilteringTimeout +
		config.PublishingTimeout

	assert.GreaterOrEqual(t, config.RequestTimeout, totalPipelineTimeout,
		"Request timeout should accommodate all pipeline timeouts")
}

// TestProxyWebhookConfig_ResourceLimits tests resource limit configurations
func TestProxyWebhookConfig_ResourceLimits(t *testing.T) {
	tests := []struct {
		name               string
		maxRequestSize     int
		maxAlertsPerReq    int
		maxConcurrentAlert int
		valid              bool
	}{
		{"small limits", 1 * 1024 * 1024, 10, 5, true},      // 1MB, 10 alerts, 5 concurrent
		{"medium limits", 10 * 1024 * 1024, 100, 10, true},  // 10MB, 100 alerts, 10 concurrent
		{"large limits", 50 * 1024 * 1024, 500, 50, true},   // 50MB, 500 alerts, 50 concurrent
		{"invalid size", 0, 100, 10, false},                 // 0MB is invalid
		{"invalid alerts", 10 * 1024 * 1024, 0, 10, false},  // 0 alerts is invalid
		{"invalid concurrent", 10 * 1024 * 1024, 100, 0, false}, // 0 concurrent is invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProxyWebhookConfig{
				MaxRequestSize:        tt.maxRequestSize,
				RequestTimeout:        30 * time.Second,
				MaxAlertsPerRequest:   tt.maxAlertsPerReq,
				ClassificationTimeout: 5 * time.Second,
				FilteringTimeout:      1 * time.Second,
				PublishingTimeout:     10 * time.Second,
				MaxConcurrentAlerts:   tt.maxConcurrentAlert,
				MaxConcurrentTargets:  10,
			}

			err := config.Validate()

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestProxyWebhookConfig_FeatureToggles tests feature toggle configurations
func TestProxyWebhookConfig_FeatureToggles(t *testing.T) {
	tests := []struct {
		name                 string
		enableClassification bool
		enableFiltering      bool
		enablePublishing     bool
	}{
		{"all enabled", true, true, true},
		{"only classification", true, false, false},
		{"only filtering", false, true, false},
		{"only publishing", false, false, true},
		{"classification + filtering", true, true, false},
		{"filtering + publishing", false, true, true},
		{"all disabled", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultProxyWebhookConfig()
			config.EnableClassification = tt.enableClassification
			config.EnableFiltering = tt.enableFiltering
			config.EnablePublishing = tt.enablePublishing

			err := config.Validate()
			assert.NoError(t, err) // All combinations are valid

			assert.Equal(t, tt.enableClassification, config.EnableClassification)
			assert.Equal(t, tt.enableFiltering, config.EnableFiltering)
			assert.Equal(t, tt.enablePublishing, config.EnablePublishing)
		})
	}
}

// TestProxyWebhookConfig_ErrorHandlingModes tests error handling configuration
func TestProxyWebhookConfig_ErrorHandlingModes(t *testing.T) {
	tests := []struct {
		name            string
		continueOnError bool
		description     string
	}{
		{
			name:            "fail fast mode",
			continueOnError: false,
			description:     "Stop processing on first error",
		},
		{
			name:            "continue on error mode",
			continueOnError: true,
			description:     "Process all alerts even if some fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultProxyWebhookConfig()
			config.ContinueOnError = tt.continueOnError

			assert.Equal(t, tt.continueOnError, config.ContinueOnError)
		})
	}
}

// BenchmarkDefaultProxyWebhookConfig benchmarks default config creation
func BenchmarkDefaultProxyWebhookConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultProxyWebhookConfig()
	}
}

// BenchmarkProxyWebhookConfig_Validate benchmarks config validation
func BenchmarkProxyWebhookConfig_Validate(b *testing.B) {
	config := DefaultProxyWebhookConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

