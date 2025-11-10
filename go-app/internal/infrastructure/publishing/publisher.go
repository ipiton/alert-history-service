package publishing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// AlertPublisher interface for publishing alerts to external systems
type AlertPublisher interface {
	// Publish publishes an enriched alert to the target
	Publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error

	// Name returns the publisher name/type
	Name() string
}

// HTTPPublisher is a base HTTP client for all publishers
type HTTPPublisher struct {
	formatter  AlertFormatter
	httpClient *http.Client
	logger     *slog.Logger
}

// NewHTTPPublisher creates a new HTTP publisher with default settings
func NewHTTPPublisher(formatter AlertFormatter, logger *slog.Logger) *HTTPPublisher {
	if logger == nil {
		logger = slog.Default()
	}

	return &HTTPPublisher{
		formatter: formatter,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// publish is a helper method to perform HTTP POST with formatted payload
func (p *HTTPPublisher) publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	// Format alert for target format
	payload, err := p.formatter.FormatAlert(ctx, enrichedAlert, target.Format)
	if err != nil {
		return fmt.Errorf("failed to format alert: %w", err)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", target.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range target.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	p.logger.Debug("Alert published successfully",
		"target", target.Name,
		"status_code", resp.StatusCode,
	)

	return nil
}

// RootlyPublisher publishes alerts to Rootly
type RootlyPublisher struct {
	*HTTPPublisher
}

// NewRootlyPublisher creates a new Rootly publisher
func NewRootlyPublisher(formatter AlertFormatter, logger *slog.Logger) AlertPublisher {
	return &RootlyPublisher{
		HTTPPublisher: NewHTTPPublisher(formatter, logger),
	}
}

// Publish publishes alert to Rootly
func (p *RootlyPublisher) Publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	return p.publish(ctx, enrichedAlert, target)
}

// Name returns publisher name
func (p *RootlyPublisher) Name() string {
	return "Rootly"
}

// PagerDutyPublisher publishes alerts to PagerDuty
type PagerDutyPublisher struct {
	*HTTPPublisher
}

// NewPagerDutyPublisher creates a new PagerDuty publisher
func NewPagerDutyPublisher(formatter AlertFormatter, logger *slog.Logger) AlertPublisher {
	return &PagerDutyPublisher{
		HTTPPublisher: NewHTTPPublisher(formatter, logger),
	}
}

// Publish publishes alert to PagerDuty
func (p *PagerDutyPublisher) Publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	return p.publish(ctx, enrichedAlert, target)
}

// Name returns publisher name
func (p *PagerDutyPublisher) Name() string {
	return "PagerDuty"
}

// SlackPublisher publishes alerts to Slack
type SlackPublisher struct {
	*HTTPPublisher
}

// NewSlackPublisher creates a new Slack publisher
func NewSlackPublisher(formatter AlertFormatter, logger *slog.Logger) AlertPublisher {
	return &SlackPublisher{
		HTTPPublisher: NewHTTPPublisher(formatter, logger),
	}
}

// Publish publishes alert to Slack
func (p *SlackPublisher) Publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	return p.publish(ctx, enrichedAlert, target)
}

// Name returns publisher name
func (p *SlackPublisher) Name() string {
	return "Slack"
}

// WebhookPublisher publishes alerts to generic webhooks
type WebhookPublisher struct {
	*HTTPPublisher
}

// NewWebhookPublisher creates a new generic webhook publisher
func NewWebhookPublisher(formatter AlertFormatter, logger *slog.Logger) AlertPublisher {
	return &WebhookPublisher{
		HTTPPublisher: NewHTTPPublisher(formatter, logger),
	}
}

// Publish publishes alert to generic webhook
func (p *WebhookPublisher) Publish(ctx context.Context, enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	return p.publish(ctx, enrichedAlert, target)
}

// Name returns publisher name
func (p *WebhookPublisher) Name() string {
	return "Webhook"
}

// PublisherFactory creates publishers based on target type
type PublisherFactory struct {
	formatter       AlertFormatter
	logger          *slog.Logger
	rootlyCache     IncidentIDCache   // Shared Rootly incident cache
	rootlyMetrics   *RootlyMetrics    // Shared Rootly metrics
	rootlyClientMap map[string]RootlyIncidentsClient // Cache of Rootly clients by API key
}

// NewPublisherFactory creates a new publisher factory
func NewPublisherFactory(formatter AlertFormatter, logger *slog.Logger) *PublisherFactory {
	return &PublisherFactory{
		formatter:       formatter,
		logger:          logger,
		rootlyCache:     NewIncidentIDCache(24 * time.Hour), // 24h TTL for incident tracking
		rootlyMetrics:   NewRootlyMetrics(),
		rootlyClientMap: make(map[string]RootlyIncidentsClient),
	}
}

// CreatePublisher creates a publisher for the given target type
func (f *PublisherFactory) CreatePublisher(targetType string) (AlertPublisher, error) {
	switch TargetType(targetType) {
	case TargetTypeRootly:
		return NewRootlyPublisher(f.formatter, f.logger), nil
	case TargetTypePagerDuty:
		return NewPagerDutyPublisher(f.formatter, f.logger), nil
	case TargetTypeSlack:
		return NewSlackPublisher(f.formatter, f.logger), nil
	case TargetTypeWebhook, TargetTypeAlertmanager:
		return NewWebhookPublisher(f.formatter, f.logger), nil
	default:
		return NewWebhookPublisher(f.formatter, f.logger), nil // Default to webhook
	}
}

// CreatePublisherForTarget creates a publisher for a specific target with full configuration
func (f *PublisherFactory) CreatePublisherForTarget(target *core.PublishingTarget) (AlertPublisher, error) {
	switch TargetType(target.Type) {
	case TargetTypeRootly:
		return f.createEnhancedRootlyPublisher(target)
	case TargetTypePagerDuty:
		return NewPagerDutyPublisher(f.formatter, f.logger), nil
	case TargetTypeSlack:
		return NewSlackPublisher(f.formatter, f.logger), nil
	case TargetTypeWebhook, TargetTypeAlertmanager:
		return NewWebhookPublisher(f.formatter, f.logger), nil
	default:
		return NewWebhookPublisher(f.formatter, f.logger), nil
	}
}

// createEnhancedRootlyPublisher creates an EnhancedRootlyPublisher with full Rootly API integration
func (f *PublisherFactory) createEnhancedRootlyPublisher(target *core.PublishingTarget) (AlertPublisher, error) {
	// Extract API key from target headers
	apiKey := ""
	if auth, ok := target.Headers["Authorization"]; ok {
		// Remove "Bearer " prefix if present
		apiKey = strings.TrimPrefix(auth, "Bearer ")
	}

	if apiKey == "" {
		f.logger.Warn("Rootly target missing API key, falling back to HTTP publisher", "target", target.Name)
		return NewRootlyPublisher(f.formatter, f.logger), nil
	}

	// Get or create Rootly client for this API key
	client, ok := f.rootlyClientMap[apiKey]
	if !ok {
		// Create new client with configuration
		config := ClientConfig{
			BaseURL: target.URL,
			APIKey:  apiKey,
			Timeout: 10 * time.Second,
		}
		client = NewRootlyIncidentsClient(config, f.logger)
		f.rootlyClientMap[apiKey] = client
	}

	// Create EnhancedRootlyPublisher with shared cache and metrics
	return NewEnhancedRootlyPublisher(
		client,
		f.rootlyCache,
		f.rootlyMetrics,
		f.formatter,
		f.logger,
	), nil
}
