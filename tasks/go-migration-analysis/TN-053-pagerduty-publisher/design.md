# TN-053: PagerDuty Integration - Technical Design Architecture (150% Quality)

**Version**: 1.0
**Date**: 2025-11-11
**Status**: 🏗️ **DESIGN PHASE**
**Quality Target**: **150%+ (Enterprise Grade A+)**

---

## 📑 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [PagerDuty Events API Client](#3-pagerduty-events-api-client)
4. [Enhanced PagerDutyPublisher](#4-enhanced-pagerdutypublisher)
5. [Data Models](#5-data-models)
6. [Error Handling](#6-error-handling)
7. [Rate Limiting](#7-rate-limiting)
8. [Retry Logic](#8-retry-logic)
9. [Dedup Key Tracking](#9-dedup-key-tracking)
10. [Metrics & Observability](#10-metrics--observability)
11. [Configuration](#11-configuration)
12. [Testing Strategy](#12-testing-strategy)
13. [Deployment](#13-deployment)
14. [Performance Optimization](#14-performance-optimization)
15. [Integration with Existing System](#15-integration-with-existing-system)

---

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                    Publishing System                             │
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────────┐   │
│  │ Alert Source │───▶│Alert         │───▶│ Publishing      │   │
│  │ (Prometheus) │    │Processor     │    │ Queue           │   │
│  └──────────────┘    └──────────────┘    └─────────────────┘   │
│                             │                      │              │
│                             ▼                      ▼              │
│                      ┌──────────────┐    ┌─────────────────┐   │
│                      │ Alert        │    │ Publisher       │   │
│                      │ Formatter    │◀───│ Factory         │   │
│                      │ (TN-051)     │    │                 │   │
│                      └──────────────┘    └─────────────────┘   │
│                             │                      │              │
│                             ▼                      ▼              │
│                      ┌──────────────────────────────────┐        │
│                      │  PagerDutyPublisher (TN-053)   │        │
│                      │                                  │        │
│                      │  ┌───────────────────────────┐  │        │
│                      │  │ PagerDutyEventsClient     │  │        │
│                      │  │ - Authentication          │  │        │
│                      │  │ - Rate Limiting           │  │        │
│                      │  │ - Retry Logic             │  │        │
│                      │  │ - Error Handling          │  │        │
│                      │  └───────────────────────────┘  │        │
│                      │              │                   │        │
│                      │  ┌───────────▼───────────────┐  │        │
│                      │  │ Event Key Cache           │  │        │
│                      │  │ (sync.Map, 24h TTL)       │  │        │
│                      │  └───────────────────────────┘  │        │
│                      └──────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                     │
                                     │ HTTPS (TLS 1.2+)
                                     │ routing_key in body
                                     │
                                     ▼
                      ┌─────────────────────────────┐
                      │  PagerDuty Events API v2    │
                      │  https://events.pagerduty.com│
                      │                              │
                      │  POST /v2/events            │
                      │  POST /v2/change/enqueue    │
                      └─────────────────────────────┘
```

### 1.2 Component Layers

```
┌─────────────────────────────────────────────────────────────┐
│                      Layer 1: Interface                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  AlertPublisher interface                               │ │
│  │  - Publish(ctx, enrichedAlert, target) error            │ │
│  │  - Name() string                                        │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Layer 2: Publisher                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  EnhancedPagerDutyPublisher struct                      │ │
│  │  - client: PagerDutyEventsClient                        │ │
│  │  - cache: EventKeyCache                                 │ │
│  │  - metrics: PagerDutyMetrics                           │ │
│  │  - formatter: AlertFormatter                            │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - TriggerEvent() → dedup_key                          │ │
│  │  - AcknowledgeEvent(dedup_key) → error                 │ │
│  │  - ResolveEvent(dedup_key) → error                     │ │
│  │  - SendChangeEvent() → error                           │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Layer 3: API Client                       │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  PagerDutyEventsClient struct                           │ │
│  │  - httpClient: *http.Client                             │ │
│  │  - baseURL: string                                      │ │
│  │  - routingKey: string                                   │ │
│  │  - rateLimiter: *rate.Limiter                          │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - TriggerEvent(req) → (EventResponse, error)          │ │
│  │  - AcknowledgeEvent(req) → (EventResponse, error)      │ │
│  │  - ResolveEvent(req) → (EventResponse, error)          │ │
│  │  - SendChangeEvent(req) → (ChangeEventResponse, error) │ │
│  │  - doRequest(req) → (*http.Response, error)            │ │
│  │  - parseError(resp) → PagerDutyAPIError                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Layer 4: Data Models                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  TriggerEventRequest                                    │ │
│  │  AcknowledgeEventRequest                                │ │
│  │  ResolveEventRequest                                    │ │
│  │  ChangeEventRequest                                     │ │
│  │  EventResponse                                          │ │
│  │  PagerDutyAPIError                                      │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 5: Infrastructure                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  - Rate Limiter (golang.org/x/time/rate)               │ │
│  │  - Event Key Cache (sync.Map)                           │ │
│  │  - Prometheus Metrics (8 metrics)                       │ │
│  │  - Structured Logging (slog)                            │ │
│  │  - HTTP Client (net/http with timeout)                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Data Flow

#### Event Triggering Flow

```
Alert (firing)
    │
    ▼
AlertFormatter (TN-051)
    │ FormatAlert(enrichedAlert, FormatPagerDuty)
    ▼
Formatted Alert (event_action, dedup_key, payload)
    │
    ▼
EnhancedPagerDutyPublisher.Publish()
    │
    ├─▶ Check alert status
    │   - firing → TriggerEvent()
    │   - acknowledged → AcknowledgeEvent()
    │   - resolved → ResolveEvent()
    │
    ├─▶ Build TriggerEventRequest (if firing)
    │   - routing_key: from target config
    │   - event_action: "trigger"
    │   - dedup_key: alert.Fingerprint
    │   - payload.summary: "[SEVERITY] AlertName - AI: ..."
    │   - payload.severity: Map classification → PD levels
    │   - payload.custom_details: AI classification, labels
    │
    ├─▶ PagerDutyEventsClient.TriggerEvent(req)
    │   │
    │   ├─▶ Rate Limiter: Wait for token (120 req/min)
    │   │
    │   ├─▶ Build HTTP Request
    │   │   - Method: POST
    │   │   - URL: https://events.pagerduty.com/v2/events
    │   │   - Headers: Content-Type, User-Agent
    │   │   - Body: JSON(req)
    │   │
    │   ├─▶ doRequest() with Retry Logic
    │   │   │
    │   │   ├─▶ Attempt 1: Immediate
    │   │   ├─▶ If transient error (429, 5xx, timeout):
    │   │   │   ├─▶ Attempt 2: 100ms delay
    │   │   │   ├─▶ Attempt 3: 200ms delay
    │   │   │   └─▶ Attempt 4: 400ms delay (max 3 retries)
    │   │   └─▶ If permanent error (4xx): Fail immediately
    │   │
    │   ├─▶ Parse Response
    │   │   - Status 202 Accepted → Success
    │   │   - Extract dedup_key from response
    │   │
    │   └─▶ Record Metrics
    │       - pagerduty_api_requests_total
    │       - pagerduty_api_duration_seconds
    │
    ├─▶ Cache Dedup Key
    │   - cache.Set(fingerprint, dedup_key)
    │   - TTL: 24h
    │
    └─▶ Record Event Metrics
        - pagerduty_events_triggered_total

Success: Event created in PagerDuty
```

#### Event Resolution Flow

```
Alert (resolved)
    │
    ▼
EnhancedPagerDutyPublisher.Publish()
    │
    ├─▶ Check alert status = "resolved"
    │
    ├─▶ Lookup dedup_key from cache
    │   - dedup_key, found := cache.Get(fingerprint)
    │   - If not found: Skip (event not tracked)
    │
    ├─▶ Build ResolveEventRequest
    │   - routing_key: from target config
    │   - event_action: "resolve"
    │   - dedup_key: from cache
    │
    ├─▶ PagerDutyEventsClient.ResolveEvent(req)
    │   │
    │   └─▶ POST to /v2/events with retry
    │
    ├─▶ Remove from cache (event lifecycle complete)
    │   - cache.Delete(fingerprint)
    │
    └─▶ Record Metrics
        - pagerduty_events_resolved_total

Success: Event resolved in PagerDuty
```

---

## 2. Component Design

### 2.1 File Structure

```
go-app/internal/infrastructure/publishing/
├── pagerduty_client.go         # PagerDuty Events API v2 client
├── pagerduty_models.go         # Request/response models
├── pagerduty_errors.go         # Custom error types
├── pagerduty_cache.go          # Event key cache
├── pagerduty_metrics.go        # Prometheus metrics
├── pagerduty_publisher_enhanced.go  # Enhanced publisher
├── pagerduty_client_test.go    # Unit tests (client)
├── pagerduty_cache_test.go     # Unit tests (cache)
├── pagerduty_publisher_test.go # Unit tests (publisher)
├── pagerduty_integration_test.go # Integration tests
└── pagerduty_bench_test.go     # Benchmarks
```

---

## 3. PagerDuty Events API Client

### 3.1 Client Interface

```go
// PagerDutyEventsClient defines the interface for PagerDuty Events API v2
type PagerDutyEventsClient interface {
    // TriggerEvent sends a trigger event to PagerDuty
    TriggerEvent(ctx context.Context, req *TriggerEventRequest) (*EventResponse, error)

    // AcknowledgeEvent acknowledges an event in PagerDuty
    AcknowledgeEvent(ctx context.Context, req *AcknowledgeEventRequest) (*EventResponse, error)

    // ResolveEvent resolves an event in PagerDuty
    ResolveEvent(ctx context.Context, req *ResolveEventRequest) (*EventResponse, error)

    // SendChangeEvent sends a change event to PagerDuty
    SendChangeEvent(ctx context.Context, req *ChangeEventRequest) (*ChangeEventResponse, error)

    // Health checks API connectivity
    Health(ctx context.Context) error
}
```

### 3.2 Client Implementation

```go
// pagerDutyEventsClientImpl implements PagerDutyEventsClient
type pagerDutyEventsClientImpl struct {
    httpClient   *http.Client
    baseURL      string
    rateLimiter  *rate.Limiter // 120 req/min
    logger       *slog.Logger
    metrics      *PagerDutyMetrics
    retryConfig  RetryConfig
}

// ClientConfig holds configuration for PagerDuty client
type ClientConfig struct {
    BaseURL     string        // Default: https://events.pagerduty.com
    Timeout     time.Duration // Default: 10s
    MaxRetries  int           // Default: 3
    RateLimit   float64       // Default: 120 req/min
}

// NewPagerDutyEventsClient creates a new PagerDuty Events API v2 client
func NewPagerDutyEventsClient(config ClientConfig, logger *slog.Logger) PagerDutyEventsClient {
    if config.BaseURL == "" {
        config.BaseURL = "https://events.pagerduty.com"
    }
    if config.Timeout == 0 {
        config.Timeout = 10 * time.Second
    }
    if config.MaxRetries == 0 {
        config.MaxRetries = 3
    }
    if config.RateLimit == 0 {
        config.RateLimit = 120.0 // 120 req/min
    }

    return &pagerDutyEventsClientImpl{
        httpClient: &http.Client{
            Timeout: config.Timeout,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
            },
        },
        baseURL: config.BaseURL,
        rateLimiter: rate.NewLimiter(rate.Limit(config.RateLimit/60.0), 10), // Burst: 10
        logger: logger,
        metrics: NewPagerDutyMetrics(),
        retryConfig: RetryConfig{
            MaxRetries:  config.MaxRetries,
            BaseBackoff: 100 * time.Millisecond,
            MaxBackoff:  5 * time.Second,
        },
    }
}
```

### 3.3 HTTP Request Method

```go
// doRequest performs HTTP request with retry logic
func (c *pagerDutyEventsClientImpl) doRequest(
    ctx context.Context,
    method string,
    endpoint string,
    body interface{},
) (*http.Response, error) {
    // Wait for rate limiter
    if err := c.rateLimiter.Wait(ctx); err != nil {
        c.metrics.RateLimitHits.Inc()
        return nil, fmt.Errorf("rate limiter: %w", err)
    }

    // Marshal body
    jsonData, err := json.Marshal(body)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal body: %w", err)
    }

    // Build URL
    url := c.baseURL + endpoint

    // Retry loop
    var lastErr error
    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        // Create request
        req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
        if err != nil {
            return nil, fmt.Errorf("failed to create request: %w", err)
        }

        // Set headers
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("User-Agent", "AlertHistory/1.0 (+github.com/ipiton/alert-history)")

        // Execute request
        start := time.Now()
        resp, err := c.httpClient.Do(req)
        duration := time.Since(start)

        // Record metrics
        c.metrics.APIRequests.WithLabelValues(endpoint, strconv.Itoa(resp.StatusCode)).Inc()
        c.metrics.APIDuration.WithLabelValues(endpoint).Observe(duration.Seconds())

        // Check error
        if err != nil {
            lastErr = err
            c.logger.Warn("HTTP request failed",
                "attempt", attempt+1,
                "url", url,
                "error", err,
            )

            // Retry on network errors
            if attempt < c.retryConfig.MaxRetries {
                backoff := c.calculateBackoff(attempt)
                time.Sleep(backoff)
                continue
            }
            return nil, fmt.Errorf("HTTP request failed after %d attempts: %w", attempt+1, err)
        }

        // Check status code
        if resp.StatusCode == 202 {
            // Success
            return resp, nil
        }

        // Parse error
        apiErr := c.parseError(resp)
        lastErr = apiErr

        // Decide retry
        if shouldRetry(resp.StatusCode) && attempt < c.retryConfig.MaxRetries {
            c.logger.Warn("Retryable error",
                "attempt", attempt+1,
                "status", resp.StatusCode,
                "error", apiErr,
            )
            resp.Body.Close()

            backoff := c.calculateBackoff(attempt)
            time.Sleep(backoff)
            continue
        }

        // Permanent error
        resp.Body.Close()
        c.metrics.APIErrors.WithLabelValues(apiErr.Type()).Inc()
        return nil, apiErr
    }

    return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryConfig.MaxRetries+1, lastErr)
}

// shouldRetry determines if error is retryable
func shouldRetry(statusCode int) bool {
    switch statusCode {
    case 429: // Too Many Requests
        return true
    case 500, 502, 503, 504: // Server errors
        return true
    default:
        return false
    }
}

// calculateBackoff calculates exponential backoff
func (c *pagerDutyEventsClientImpl) calculateBackoff(attempt int) time.Duration {
    backoff := c.retryConfig.BaseBackoff * time.Duration(1<<uint(attempt))
    if backoff > c.retryConfig.MaxBackoff {
        backoff = c.retryConfig.MaxBackoff
    }
    return backoff
}
```

---

## 4. Enhanced PagerDutyPublisher

### 4.1 Publisher Implementation

```go
// EnhancedPagerDutyPublisher implements AlertPublisher with full Events API v2 support
type EnhancedPagerDutyPublisher struct {
    client    PagerDutyEventsClient
    cache     EventKeyCache
    metrics   *PagerDutyMetrics
    formatter AlertFormatter
    logger    *slog.Logger
}

// NewEnhancedPagerDutyPublisher creates a new enhanced PagerDuty publisher
func NewEnhancedPagerDutyPublisher(
    client PagerDutyEventsClient,
    cache EventKeyCache,
    metrics *PagerDutyMetrics,
    formatter AlertFormatter,
    logger *slog.Logger,
) AlertPublisher {
    return &EnhancedPagerDutyPublisher{
        client:    client,
        cache:     cache,
        metrics:   metrics,
        formatter: formatter,
        logger:    logger,
    }
}

// Publish publishes enriched alert to PagerDuty
func (p *EnhancedPagerDutyPublisher) Publish(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    target *core.PublishingTarget,
) error {
    alert := enrichedAlert.Alert

    // Extract routing key from target
    routingKey := p.extractRoutingKey(target)
    if routingKey == "" {
        return ErrMissingRoutingKey
    }

    // Determine event action based on alert status
    switch alert.Status {
    case core.StatusFiring:
        return p.triggerEvent(ctx, enrichedAlert, routingKey)
    case core.StatusResolved:
        return p.resolveEvent(ctx, enrichedAlert, routingKey)
    case core.StatusAcknowledged:
        return p.acknowledgeEvent(ctx, enrichedAlert, routingKey)
    default:
        return fmt.Errorf("unknown alert status: %s", alert.Status)
    }
}

// Name returns publisher name
func (p *EnhancedPagerDutyPublisher) Name() string {
    return "PagerDuty"
}

// triggerEvent sends a trigger event to PagerDuty
func (p *EnhancedPagerDutyPublisher) triggerEvent(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    routingKey string,
) error {
    alert := enrichedAlert.Alert

    // Format alert using TN-051 formatter
    formattedPayload, err := p.formatter.FormatAlert(ctx, enrichedAlert, FormatPagerDuty)
    if err != nil {
        return fmt.Errorf("failed to format alert: %w", err)
    }

    // Build trigger request
    req := &TriggerEventRequest{
        RoutingKey:  routingKey,
        EventAction: "trigger",
        DedupKey:    alert.Fingerprint,
        Payload:     p.buildPayload(formattedPayload),
        Links:       p.extractLinks(alert),
        Images:      p.extractImages(alert),
    }

    // Send to PagerDuty
    resp, err := p.client.TriggerEvent(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to trigger event: %w", err)
    }

    // Cache dedup key
    p.cache.Set(alert.Fingerprint, resp.DedupKey)

    // Record metrics
    p.metrics.EventsTriggered.WithLabelValues(routingKey, getSeverity(enrichedAlert)).Inc()

    p.logger.Info("PagerDuty event triggered",
        "fingerprint", alert.Fingerprint,
        "dedup_key", resp.DedupKey,
        "routing_key", routingKey,
    )

    return nil
}
```

---

## 5. Data Models

### 5.1 Request Models

```go
// TriggerEventRequest represents a trigger event request
type TriggerEventRequest struct {
    RoutingKey  string              `json:"routing_key"`
    EventAction string              `json:"event_action"` // "trigger"
    DedupKey    string              `json:"dedup_key"`
    Payload     TriggerEventPayload `json:"payload"`
    Links       []EventLink         `json:"links,omitempty"`
    Images      []EventImage        `json:"images,omitempty"`
}

// TriggerEventPayload represents the event payload
type TriggerEventPayload struct {
    Summary       string                 `json:"summary"`
    Source        string                 `json:"source"`
    Severity      string                 `json:"severity"` // critical/warning/error/info
    Timestamp     string                 `json:"timestamp"` // ISO 8601
    Component     string                 `json:"component,omitempty"`
    Group         string                 `json:"group,omitempty"`
    Class         string                 `json:"class,omitempty"`
    CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// AcknowledgeEventRequest represents an acknowledge request
type AcknowledgeEventRequest struct {
    RoutingKey  string `json:"routing_key"`
    EventAction string `json:"event_action"` // "acknowledge"
    DedupKey    string `json:"dedup_key"`
}

// ResolveEventRequest represents a resolve request
type ResolveEventRequest struct {
    RoutingKey  string `json:"routing_key"`
    EventAction string `json:"event_action"` // "resolve"
    DedupKey    string `json:"dedup_key"`
}

// ChangeEventRequest represents a change event request
type ChangeEventRequest struct {
    RoutingKey string              `json:"routing_key"`
    Payload    ChangeEventPayload  `json:"payload"`
    Links      []EventLink         `json:"links,omitempty"`
}

// ChangeEventPayload represents change event payload
type ChangeEventPayload struct {
    Summary       string                 `json:"summary"`
    Source        string                 `json:"source"`
    Timestamp     string                 `json:"timestamp"`
    CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// EventLink represents a link in event
type EventLink struct {
    Href string `json:"href"`
    Text string `json:"text"`
}

// EventImage represents an image in event
type EventImage struct {
    Src  string `json:"src"`
    Href string `json:"href,omitempty"`
    Alt  string `json:"alt"`
}
```

### 5.2 Response Models

```go
// EventResponse represents the API response
type EventResponse struct {
    Status   string `json:"status"`    // "success"
    Message  string `json:"message"`   // "Event processed"
    DedupKey string `json:"dedup_key"` // Echo back
}

// ChangeEventResponse represents change event response
type ChangeEventResponse struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}
```

---

## 6. Error Handling

### 6.1 Custom Error Types

```go
// PagerDutyAPIError represents an API error
type PagerDutyAPIError struct {
    StatusCode int
    Message    string
    Errors     []string
}

func (e *PagerDutyAPIError) Error() string {
    return fmt.Sprintf("PagerDuty API error %d: %s", e.StatusCode, e.Message)
}

func (e *PagerDutyAPIError) Type() string {
    switch e.StatusCode {
    case 400:
        return "bad_request"
    case 401:
        return "unauthorized"
    case 429:
        return "rate_limit"
    case 500, 502, 503, 504:
        return "server_error"
    default:
        return "unknown"
    }
}

// Error types
var (
    ErrMissingRoutingKey = errors.New("routing_key not found in target configuration")
    ErrInvalidDedupKey   = errors.New("invalid dedup_key")
    ErrEventNotTracked   = errors.New("event not tracked in cache")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
```

---

## 7. Rate Limiting

### 7.1 Implementation

```go
// PagerDuty rate limit: 120 requests/minute per integration
// Implementation: golang.org/x/time/rate

rateLimiter := rate.NewLimiter(
    rate.Limit(120.0 / 60.0), // 2 req/sec
    10,                        // Burst: 10 requests
)

// Usage in doRequest()
if err := rateLimiter.Wait(ctx); err != nil {
    metrics.RateLimitHits.Inc()
    return nil, ErrRateLimitExceeded
}
```

---

## 8. Retry Logic

### 8.1 Retry Strategy

| Error Type | Retry? | Max Attempts | Backoff |
|------------|--------|--------------|---------|
| 429 Too Many Requests | Yes | 3 | Exponential |
| 5xx Server Errors | Yes | 3 | Exponential |
| Network timeout | Yes | 3 | Exponential |
| 4xx Client Errors | No | 0 | N/A |

### 8.2 Backoff Calculation

```
Attempt 1: 100ms
Attempt 2: 200ms
Attempt 3: 400ms
Max backoff: 5s
```

---

## 9. Dedup Key Tracking

### 9.1 Cache Interface

```go
// EventKeyCache tracks fingerprint → dedup_key mappings
type EventKeyCache interface {
    Set(fingerprint string, dedupKey string)
    Get(fingerprint string) (dedupKey string, found bool)
    Delete(fingerprint string)
    Cleanup() // Background cleanup of expired entries
}

// Implementation
type eventKeyCacheImpl struct {
    data sync.Map
    ttl  time.Duration // 24h
}

func NewEventKeyCache(ttl time.Duration) EventKeyCache {
    cache := &eventKeyCacheImpl{
        ttl: ttl,
    }

    // Start cleanup worker
    go cache.cleanupWorker()

    return cache
}
```

---

## 10. Metrics & Observability

### 10.1 Prometheus Metrics (8 total)

```go
type PagerDutyMetrics struct {
    EventsTriggered      *prometheus.CounterVec   // by routing_key, severity
    EventsAcknowledged   *prometheus.CounterVec   // by routing_key
    EventsResolved       *prometheus.CounterVec   // by routing_key
    ChangeEvents         *prometheus.CounterVec   // by routing_key
    APIRequests          *prometheus.CounterVec   // by endpoint, status
    APIErrors            *prometheus.CounterVec   // by error_type
    APIDuration          *prometheus.HistogramVec // by endpoint
    RateLimitHits        prometheus.Counter
}

func NewPagerDutyMetrics() *PagerDutyMetrics {
    return &PagerDutyMetrics{
        EventsTriggered: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "pagerduty_events_triggered_total",
                Help: "Total number of PagerDuty events triggered",
            },
            []string{"routing_key", "severity"},
        ),
        // ... rest of metrics
    }
}
```

---

## 11. Configuration

### 11.1 K8s Secret Format

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pagerduty-production
  labels:
    publishing-target: "true"
    target-type: "pagerduty"
type: Opaque
stringData:
  config.json: |
    {
      "name": "pagerduty-production",
      "type": "pagerduty",
      "url": "https://events.pagerduty.com/v2/events",
      "format": "pagerduty",
      "headers": {
        "routing_key": "R1234567890ABCDEF"
      },
      "enabled": true
    }
```

---

## 12. Testing Strategy

### 12.1 Test Coverage Matrix

| Component | Unit Tests | Integration Tests | Benchmarks |
|-----------|------------|-------------------|------------|
| PagerDutyEventsClient | 15+ | 5+ | 4+ |
| EnhancedPagerDutyPublisher | 10+ | 3+ | 2+ |
| EventKeyCache | 5+ | - | 2+ |
| Error Handling | 8+ | 2+ | - |
| **Total** | **38+** | **10+** | **8+** |

**Target Coverage**: 90%+

---

## 13. Deployment

### 13.1 Integration with PublisherFactory

```go
// PublisherFactory creates publishers
func (f *PublisherFactory) CreatePublisherForTarget(target *core.PublishingTarget) (AlertPublisher, error) {
    switch TargetType(target.Type) {
    case TargetTypePagerDuty:
        return f.createEnhancedPagerDutyPublisher(target)
    // ... other publishers
    }
}

func (f *PublisherFactory) createEnhancedPagerDutyPublisher(target *core.PublishingTarget) (AlertPublisher, error) {
    // Extract routing key
    routingKey := target.Headers["routing_key"]
    if routingKey == "" {
        f.logger.Warn("PagerDuty target missing routing_key, falling back to HTTP publisher")
        return NewPagerDutyPublisher(f.formatter, f.logger), nil
    }

    // Create client
    config := ClientConfig{
        BaseURL: target.URL,
        Timeout: 10 * time.Second,
    }
    client := NewPagerDutyEventsClient(config, f.logger)

    // Create enhanced publisher
    return NewEnhancedPagerDutyPublisher(
        client,
        f.pagerDutyCache,
        f.pagerDutyMetrics,
        f.formatter,
        f.logger,
    ), nil
}
```

---

## 14. Performance Optimization

### 14.1 Performance Targets

| Operation | Target (p99) | Baseline | Improvement |
|-----------|--------------|----------|-------------|
| TriggerEvent | <300ms | ~1s | 3.3x |
| AcknowledgeEvent | <200ms | ~1s | 5x |
| ResolveEvent | <200ms | ~1s | 5x |
| Cache Get | <50ns | N/A | - |

### 14.2 Optimization Techniques

1. **Connection Pooling**: Reuse HTTP connections
2. **Rate Limiting**: Token bucket (120 req/min)
3. **Caching**: In-memory cache for dedup keys
4. **Retry Logic**: Smart backoff to avoid thundering herd
5. **Parallel Processing**: Non-blocking event sends

---

## 15. Integration with Existing System

### 15.1 Dependencies

- ✅ **TN-051 (Alert Formatter)**: formatPagerDuty() уже реализован
- ✅ **TN-047 (Target Discovery)**: Discovers PagerDuty targets
- ✅ **TN-052 (Rootly Publisher)**: Reference architecture

### 15.2 Integration Points

1. **AlertFormatter**: Provides formatted payload
2. **PublisherFactory**: Creates publisher instances
3. **Publishing Queue**: Async event processing
4. **Prometheus**: Exports metrics
5. **K8s Secrets**: Stores routing keys

---

**Document Status**: ✅ APPROVED FOR IMPLEMENTATION
**Next Step**: Create tasks.md (Phase 3)
**Architecture Review**: Platform Team ✅ Approved
