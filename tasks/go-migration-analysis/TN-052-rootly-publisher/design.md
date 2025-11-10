# TN-052: Rootly Publisher - Technical Design Architecture (150% Quality)

**Version**: 1.0
**Date**: 2025-11-08
**Status**: 🏗️ **DESIGN PHASE**
**Quality Target**: **150%+ (Enterprise Grade A+)**

---

## 📑 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Rootly API Client](#3-rootly-api-client)
4. [Enhanced RootlyPublisher](#4-enhanced-rootlypublisher)
5. [Data Models](#5-data-models)
6. [Error Handling](#6-error-handling)
7. [Rate Limiting](#7-rate-limiting)
8. [Retry Logic](#8-retry-logic)
9. [Incident ID Tracking](#9-incident-id-tracking)
10. [Metrics & Observability](#10-metrics--observability)
11. [Configuration](#11-configuration)
12. [Testing Strategy](#12-testing-strategy)
13. [Deployment](#13-deployment)
14. [Performance Optimization](#14-performance-optimization)

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
│                      │    RootlyPublisher (TN-052)     │        │
│                      │                                  │        │
│                      │  ┌───────────────────────────┐  │        │
│                      │  │ RootlyIncidentsClient     │  │        │
│                      │  │ - Authentication          │  │        │
│                      │  │ - Rate Limiting           │  │        │
│                      │  │ - Retry Logic             │  │        │
│                      │  │ - Error Handling          │  │        │
│                      │  └───────────────────────────┘  │        │
│                      │              │                   │        │
│                      │  ┌───────────▼───────────────┐  │        │
│                      │  │ Incident ID Cache         │  │        │
│                      │  │ (sync.Map, 24h TTL)       │  │        │
│                      │  └───────────────────────────┘  │        │
│                      └──────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                     │
                                     │ HTTPS (TLS 1.2+)
                                     │ Authorization: Bearer <API_KEY>
                                     │
                                     ▼
                      ┌─────────────────────────────┐
                      │   Rootly Incidents API v1   │
                      │   https://api.rootly.com/v1 │
                      │                              │
                      │  POST   /incidents          │
                      │  PATCH  /incidents/{id}     │
                      │  POST   /incidents/{id}/resolve │
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
│  │  RootlyPublisher struct                                 │ │
│  │  - client: RootlyIncidentsClient                        │ │
│  │  - cache: IncidentIDCache                              │ │
│  │  - metrics: RootlyMetrics                              │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - CreateIncident() → incident ID                      │ │
│  │  - UpdateIncident(id) → error                          │ │
│  │  - ResolveIncident(id) → error                         │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Layer 3: API Client                       │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  RootlyIncidentsClient struct                           │ │
│  │  - httpClient: *http.Client                             │ │
│  │  - baseURL: string                                      │ │
│  │  - apiKey: string                                       │ │
│  │  - rateLimiter: *rate.Limiter                          │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - CreateIncident(req) → (IncidentResponse, error)     │ │
│  │  - UpdateIncident(id, req) → (IncidentResponse, error) │ │
│  │  - ResolveIncident(id, req) → (IncidentResponse, error)│ │
│  │  - doRequest(req) → (*http.Response, error)            │ │
│  │  - parseError(resp) → RootlyAPIError                   │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Layer 4: Data Models                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  CreateIncidentRequest                                  │ │
│  │  UpdateIncidentRequest                                  │ │
│  │  ResolveIncidentRequest                                 │ │
│  │  IncidentResponse                                       │ │
│  │  RootlyAPIError                                         │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 5: Infrastructure                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  - Rate Limiter (golang.org/x/time/rate)               │ │
│  │  - Incident Cache (sync.Map)                            │ │
│  │  - Prometheus Metrics (8 metrics)                       │ │
│  │  - Structured Logging (slog)                            │ │
│  │  - HTTP Client (net/http with timeout)                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Data Flow

#### Incident Creation Flow

```
Alert (firing)
    │
    ▼
AlertFormatter (TN-051)
    │ FormatAlert(enrichedAlert, FormatRootly)
    ▼
Formatted Alert (title, description, severity, tags, custom_fields)
    │
    ▼
RootlyPublisher.Publish()
    │
    ├─▶ Check if alert status = "firing"
    │   (if not firing, skip create, try update/resolve)
    │
    ├─▶ Build CreateIncidentRequest
    │   - title: [SEVERITY] AlertName (AI: ...)
    │   - description: Alert details + AI classification
    │   - severity: Map classification → Rootly levels
    │   - started_at: alert.StartsAt
    │   - tags: Convert labels to tags
    │   - custom_fields: fingerprint, AI data
    │
    ├─▶ RootlyIncidentsClient.CreateIncident(req)
    │   │
    │   ├─▶ Rate Limiter: Wait for token (60 req/min)
    │   │
    │   ├─▶ Build HTTP Request
    │   │   - Method: POST
    │   │   - URL: https://api.rootly.com/v1/incidents
    │   │   - Headers: Authorization, Content-Type, User-Agent
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
    │   │   - Success (201 Created): Extract incident ID
    │   │   - Error (4xx/5xx): Parse Rootly error JSON
    │   │
    │   └─▶ Return (IncidentResponse, error)
    │
    ├─▶ Store incident ID in cache
    │   - Key: alert.Fingerprint
    │   - Value: response.Data.ID
    │   - TTL: 24 hours
    │
    ├─▶ Record Metrics
    │   - rootly_incidents_created_total +1
    │   - rootly_api_requests_total{endpoint="/incidents"} +1
    │   - rootly_api_duration_seconds observe(latency)
    │   - rootly_active_incidents_gauge +1
    │
    └─▶ Log Success
        - Level: INFO
        - Message: "Rootly incident created"
        - Fields: incident_id, fingerprint, severity, latency
```

#### Incident Resolution Flow

```
Alert (resolved)
    │
    ▼
RootlyPublisher.Publish()
    │
    ├─▶ Check if alert status = "resolved"
    │
    ├─▶ Lookup incident ID from cache
    │   - Key: alert.Fingerprint
    │   - If not found: Skip (incident not tracked)
    │
    ├─▶ Build ResolveIncidentRequest
    │   - summary: "Alert resolved: {name} in {namespace}"
    │
    ├─▶ RootlyIncidentsClient.ResolveIncident(id, req)
    │   │
    │   ├─▶ Rate Limiter: Wait for token
    │   │
    │   ├─▶ Build HTTP Request
    │   │   - Method: POST
    │   │   - URL: https://api.rootly.com/v1/incidents/{id}/resolve
    │   │   - Headers: Authorization, Content-Type
    │   │   - Body: JSON(req)
    │   │
    │   ├─▶ doRequest() with Retry Logic
    │   │
    │   ├─▶ Handle Response
    │   │   - 200 OK: Success
    │   │   - 404 Not Found: Incident deleted (log warning, continue)
    │   │   - 409 Conflict: Already resolved (log info, continue)
    │   │
    │   └─▶ Return error
    │
    ├─▶ Delete incident ID from cache
    │   - Key: alert.Fingerprint
    │
    ├─▶ Record Metrics
    │   - rootly_incidents_resolved_total +1
    │   - rootly_api_requests_total{endpoint="/incidents/:id/resolve"} +1
    │   - rootly_active_incidents_gauge -1
    │
    └─▶ Log Success
```

---

## 2. Component Design

### 2.1 RootlyPublisher

**Purpose**: Main publisher component implementing AlertPublisher interface

**Struct**:
```go
type RootlyPublisher struct {
    client   RootlyIncidentsClient  // API client
    cache    IncidentIDCache        // Incident ID tracking
    metrics  *RootlyMetrics         // Prometheus metrics
    logger   *slog.Logger           // Structured logging
    formatter AlertFormatter        // Alert formatting (TN-051)
}
```

**Methods**:

```go
// NewRootlyPublisher creates enhanced Rootly publisher
func NewRootlyPublisher(
    client RootlyIncidentsClient,
    cache IncidentIDCache,
    metrics *RootlyMetrics,
    formatter AlertFormatter,
    logger *slog.Logger,
) AlertPublisher {
    return &RootlyPublisher{
        client:    client,
        cache:     cache,
        metrics:   metrics,
        logger:    logger,
        formatter: formatter,
    }
}

// Publish implements AlertPublisher interface
func (p *RootlyPublisher) Publish(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    target *core.PublishingTarget,
) error {
    start := time.Now()

    // Format alert for Rootly
    payload, err := p.formatter.FormatAlert(ctx, enrichedAlert, core.FormatRootly)
    if err != nil {
        return fmt.Errorf("format failed: %w", err)
    }

    // Route based on alert status
    switch enrichedAlert.Alert.Status {
    case core.AlertStatusFiring:
        return p.createOrUpdateIncident(ctx, enrichedAlert, payload)
    case core.AlertStatusResolved:
        return p.resolveIncident(ctx, enrichedAlert)
    default:
        return fmt.Errorf("unknown alert status: %s", enrichedAlert.Alert.Status)
    }
}

// createOrUpdateIncident creates new incident or updates existing
func (p *RootlyPublisher) createOrUpdateIncident(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    payload map[string]interface{},
) error {
    fingerprint := enrichedAlert.Alert.Fingerprint

    // Check if incident exists
    incidentID, exists := p.cache.Get(fingerprint)

    if exists {
        // Update existing incident
        return p.updateIncident(ctx, incidentID, enrichedAlert, payload)
    }

    // Create new incident
    return p.createIncident(ctx, enrichedAlert, payload)
}

// createIncident creates new Rootly incident
func (p *RootlyPublisher) createIncident(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    payload map[string]interface{},
) error {
    // Build request
    req := &CreateIncidentRequest{
        Title:        payload["title"].(string),
        Description:  payload["description"].(string),
        Severity:     payload["severity"].(string),
        StartedAt:    enrichedAlert.Alert.StartsAt,
        Tags:         payload["tags"].([]string),
        CustomFields: payload["custom_fields"].(map[string]interface{}),
    }

    // Call API
    resp, err := p.client.CreateIncident(ctx, req)
    if err != nil {
        p.metrics.RecordError("create", err)
        return fmt.Errorf("create incident failed: %w", err)
    }

    // Store incident ID
    p.cache.Set(enrichedAlert.Alert.Fingerprint, resp.Data.ID)

    // Record success
    p.metrics.RecordIncidentCreated(req.Severity)
    p.logger.Info("Rootly incident created",
        "incident_id", resp.Data.ID,
        "fingerprint", enrichedAlert.Alert.Fingerprint,
        "severity", req.Severity,
    )

    return nil
}

// updateIncident updates existing incident
func (p *RootlyPublisher) updateIncident(
    ctx context.Context,
    incidentID string,
    enrichedAlert *core.EnrichedAlert,
    payload map[string]interface{},
) error {
    // Build request (only fields that changed)
    req := &UpdateIncidentRequest{
        Description:  payload["description"].(string),
        CustomFields: payload["custom_fields"].(map[string]interface{}),
    }

    // Call API
    _, err := p.client.UpdateIncident(ctx, incidentID, req)
    if err != nil {
        // If 404, incident was deleted in Rootly
        if IsNotFoundError(err) {
            p.cache.Delete(enrichedAlert.Alert.Fingerprint)
            p.logger.Warn("Incident not found (deleted in Rootly), recreating",
                "incident_id", incidentID,
                "fingerprint", enrichedAlert.Alert.Fingerprint,
            )
            // Recreate incident
            return p.createIncident(ctx, enrichedAlert, payload)
        }

        p.metrics.RecordError("update", err)
        return fmt.Errorf("update incident failed: %w", err)
    }

    // Record success
    p.metrics.RecordIncidentUpdated("annotation_change")
    p.logger.Info("Rootly incident updated",
        "incident_id", incidentID,
        "fingerprint", enrichedAlert.Alert.Fingerprint,
    )

    return nil
}

// resolveIncident resolves Rootly incident
func (p *RootlyPublisher) resolveIncident(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
) error {
    // Lookup incident ID
    incidentID, exists := p.cache.Get(enrichedAlert.Alert.Fingerprint)
    if !exists {
        p.logger.Debug("Incident ID not found, skipping resolution",
            "fingerprint", enrichedAlert.Alert.Fingerprint,
        )
        return nil // Not an error (incident might be manually resolved)
    }

    // Build request
    req := &ResolveIncidentRequest{
        Summary: fmt.Sprintf("Alert resolved: %s in %s",
            enrichedAlert.Alert.AlertName,
            *enrichedAlert.Alert.Namespace(),
        ),
    }

    // Call API
    _, err := p.client.ResolveIncident(ctx, incidentID, req)
    if err != nil {
        // If 404/409, incident already resolved/deleted
        if IsNotFoundError(err) || IsConflictError(err) {
            p.cache.Delete(enrichedAlert.Alert.Fingerprint)
            p.logger.Info("Incident already resolved or deleted",
                "incident_id", incidentID,
                "fingerprint", enrichedAlert.Alert.Fingerprint,
            )
            return nil // Not an error
        }

        p.metrics.RecordError("resolve", err)
        return fmt.Errorf("resolve incident failed: %w", err)
    }

    // Delete from cache
    p.cache.Delete(enrichedAlert.Alert.Fingerprint)

    // Record success
    p.metrics.RecordIncidentResolved()
    p.logger.Info("Rootly incident resolved",
        "incident_id", incidentID,
        "fingerprint", enrichedAlert.Alert.Fingerprint,
    )

    return nil
}

// Name returns publisher name
func (p *RootlyPublisher) Name() string {
    return "Rootly"
}
```

**Key Design Decisions**:
1. **Incident Lifecycle**: Automatic create → update → resolve based on alert status
2. **ID Tracking**: In-memory cache (sync.Map) для incident ID storage
3. **Graceful Degradation**: Skip operations если incident ID not found (не error)
4. **Error Recovery**: Auto-recreate incident если 404 (deleted in Rootly)
5. **Metrics**: Record all operations (create, update, resolve, errors)

---

## 3. Rootly API Client

### 3.1 RootlyIncidentsClient

**Purpose**: HTTP client для Rootly Incidents API v1

**Interface**:
```go
type RootlyIncidentsClient interface {
    CreateIncident(ctx context.Context, req *CreateIncidentRequest) (*IncidentResponse, error)
    UpdateIncident(ctx context.Context, id string, req *UpdateIncidentRequest) (*IncidentResponse, error)
    ResolveIncident(ctx context.Context, id string, req *ResolveIncidentRequest) (*IncidentResponse, error)
}
```

**Implementation**:
```go
type defaultRootlyIncidentsClient struct {
    httpClient   *http.Client
    baseURL      string
    apiKey       string
    rateLimiter  *rate.Limiter
    retryConfig  RetryConfig
    logger       *slog.Logger
}

// NewRootlyIncidentsClient creates API client
func NewRootlyIncidentsClient(config ClientConfig, logger *slog.Logger) RootlyIncidentsClient {
    return &defaultRootlyIncidentsClient{
        httpClient: &http.Client{
            Timeout: config.Timeout, // Default: 10s
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    MinVersion: tls.VersionTLS12, // TLS 1.2+
                },
            },
        },
        baseURL:     config.BaseURL,      // https://api.rootly.com/v1
        apiKey:      config.APIKey,
        rateLimiter: rate.NewLimiter(rate.Limit(config.RateLimit), config.RateBurst), // 60 req/min, burst 10
        retryConfig: config.RetryConfig,  // Exponential backoff
        logger:      logger,
    }
}

// CreateIncident creates new incident
func (c *defaultRootlyIncidentsClient) CreateIncident(
    ctx context.Context,
    req *CreateIncidentRequest,
) (*IncidentResponse, error) {
    // Wait for rate limiter
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }

    // Build HTTP request
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/incidents", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    // Set headers
    c.setHeaders(httpReq)

    // Execute with retry
    resp, err := c.doRequestWithRetry(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response
    if resp.StatusCode != http.StatusCreated {
        return nil, c.parseError(resp)
    }

    var incidentResp IncidentResponse
    if err := json.NewDecoder(resp.Body).Decode(&incidentResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &incidentResp, nil
}

// doRequestWithRetry executes request with exponential backoff
func (c *defaultRootlyIncidentsClient) doRequestWithRetry(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error

    backoff := c.retryConfig.BaseDelay // 100ms

    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        // Execute request
        resp, err = c.httpClient.Do(req)

        // Success
        if err == nil && !isRetryableStatus(resp.StatusCode) {
            return resp, nil
        }

        // Permanent error (4xx except 429)
        if err == nil && isPermanentError(resp.StatusCode) {
            return resp, nil
        }

        // Last attempt
        if attempt == c.retryConfig.MaxRetries {
            if err != nil {
                return nil, fmt.Errorf("max retries exceeded: %w", err)
            }
            return resp, nil
        }

        // Log retry
        c.logger.Warn("Request failed, retrying",
            "attempt", attempt+1,
            "max_retries", c.retryConfig.MaxRetries,
            "backoff", backoff,
            "error", err,
        )

        // Wait with backoff
        select {
        case <-time.After(backoff):
            // Continue
        case <-req.Context().Done():
            return nil, req.Context().Err()
        }

        // Exponential backoff (2x multiplier)
        backoff *= 2
        if backoff > c.retryConfig.MaxDelay {
            backoff = c.retryConfig.MaxDelay // Cap at 5s
        }
    }

    return resp, err
}

// setHeaders adds required headers
func (c *defaultRootlyIncidentsClient) setHeaders(req *http.Request) {
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "AlertHistory/1.0 (+github.com/ipiton/alert-history)")
    req.Header.Set("Accept", "application/vnd.rootly.v1+json")
}

// parseError parses Rootly API error
func (c *defaultRootlyIncidentsClient) parseError(resp *http.Response) error {
    body, _ := io.ReadAll(resp.Body)

    var errorResp struct {
        Errors []struct {
            Status string `json:"status"`
            Title  string `json:"title"`
            Detail string `json:"detail"`
            Source struct {
                Pointer string `json:"pointer"`
            } `json:"source"`
        } `json:"errors"`
    }

    if err := json.Unmarshal(body, &errorResp); err != nil {
        return &RootlyAPIError{
            StatusCode: resp.StatusCode,
            Title:      "Unknown Error",
            Detail:     string(body),
        }
    }

    if len(errorResp.Errors) == 0 {
        return &RootlyAPIError{
            StatusCode: resp.StatusCode,
            Title:      "Unknown Error",
            Detail:     string(body),
        }
    }

    firstError := errorResp.Errors[0]
    return &RootlyAPIError{
        StatusCode: resp.StatusCode,
        Title:      firstError.Title,
        Detail:     firstError.Detail,
        Source:     firstError.Source.Pointer,
    }
}

// Helper functions
func isRetryableStatus(code int) bool {
    return code == 429 || code >= 500
}

func isPermanentError(code int) bool {
    return code >= 400 && code < 500 && code != 429
}
```

**Key Design Decisions**:
1. **Rate Limiting**: Token bucket (golang.org/x/time/rate) for 60 req/min
2. **Retry Logic**: Exponential backoff (100ms → 5s max) for transient errors
3. **TLS Security**: TLS 1.2+ required, certificate validation enabled
4. **Error Parsing**: Detailed Rootly error structure parsing
5. **Context Support**: All requests context-aware (cancellation, timeout)

---

## 4. Enhanced RootlyPublisher

### 4.1 Enhancements Over Baseline

| Aspect | Baseline | Enhanced | Improvement |
|--------|----------|----------|-------------|
| **API Integration** | Generic HTTP POST | Full Rootly API v1 | +100% |
| **Lifecycle** | Fire-and-forget | Create → Update → Resolve | +100% |
| **Error Handling** | Generic | Rootly-specific parsing | +70% |
| **Retry Logic** | None | Exponential backoff | +100% |
| **Rate Limiting** | None | Token bucket (60 req/min) | +100% |
| **ID Tracking** | None | In-memory cache (24h TTL) | +100% |
| **Metrics** | 0 | 8 Prometheus metrics | +8 |
| **LOC** | 21 | 350 | +1,567% |

### 4.2 Backward Compatibility

**Fallback Strategy**: If Rootly client unavailable, fallback to baseline HTTPPublisher

```go
func (p *RootlyPublisher) Publish(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    target *core.PublishingTarget,
) error {
    // Try enhanced publisher
    if p.client != nil {
        err := p.publishEnhanced(ctx, enrichedAlert, target)
        if err == nil {
            return nil
        }

        // Log error but continue to fallback
        p.logger.Warn("Enhanced publisher failed, falling back to basic HTTP",
            "error", err,
        )
    }

    // Fallback to basic HTTP POST (baseline behavior)
    return p.publishBasic(ctx, enrichedAlert, target)
}
```

**Zero Breaking Changes**: Same AlertPublisher interface, same formatter integration

---

## 5. Data Models

### 5.1 Request Models

```go
// CreateIncidentRequest creates new Rootly incident
type CreateIncidentRequest struct {
    Title        string                 `json:"title"`                    // Required
    Description  string                 `json:"description"`              // Required
    Severity     string                 `json:"severity"`                 // Required: critical, major, minor, low
    StartedAt    time.Time              `json:"started_at"`               // Required
    Tags         []string               `json:"tags,omitempty"`           // Optional
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // Optional
}

// UpdateIncidentRequest updates existing incident
type UpdateIncidentRequest struct {
    Description  string                 `json:"description,omitempty"`
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// ResolveIncidentRequest resolves incident
type ResolveIncidentRequest struct {
    Summary string `json:"summary,omitempty"`
}
```

### 5.2 Response Models

```go
// IncidentResponse from Rootly API
type IncidentResponse struct {
    Data struct {
        ID         string `json:"id"`   // e.g., "01HKXYZ..."
        Type       string `json:"type"` // "incidents"
        Attributes struct {
            Title     string    `json:"title"`
            Severity  string    `json:"severity"`
            StartedAt time.Time `json:"started_at"`
            Status    string    `json:"status"` // "started", "resolved"
            CreatedAt time.Time `json:"created_at"`
            UpdatedAt time.Time `json:"updated_at,omitempty"`
            ResolvedAt *time.Time `json:"resolved_at,omitempty"`
        } `json:"attributes"`
    } `json:"data"`
}
```

### 5.3 Validation

```go
// Validate validates CreateIncidentRequest
func (r *CreateIncidentRequest) Validate() error {
    if r.Title == "" {
        return fmt.Errorf("title is required")
    }
    if len(r.Title) > 255 {
        return fmt.Errorf("title too long (max 255 chars)")
    }
    if r.Description == "" {
        return fmt.Errorf("description is required")
    }
    if !isValidSeverity(r.Severity) {
        return fmt.Errorf("invalid severity: %s (must be critical, major, minor, low)", r.Severity)
    }
    if r.StartedAt.IsZero() {
        return fmt.Errorf("started_at is required")
    }
    return nil
}

func isValidSeverity(s string) bool {
    return s == "critical" || s == "major" || s == "minor" || s == "low"
}
```

---

## 6. Error Handling

### 6.1 RootlyAPIError

```go
// RootlyAPIError represents Rootly API error
type RootlyAPIError struct {
    StatusCode int
    Title      string
    Detail     string
    Source     string // JSON pointer (e.g., "/data/attributes/title")
}

func (e *RootlyAPIError) Error() string {
    if e.Source != "" {
        return fmt.Sprintf("Rootly API error %d: %s - %s (field: %s)",
            e.StatusCode, e.Title, e.Detail, e.Source)
    }
    return fmt.Sprintf("Rootly API error %d: %s - %s",
        e.StatusCode, e.Title, e.Detail)
}

// IsRetryable returns true if error is transient
func (e *RootlyAPIError) IsRetryable() bool {
    return e.StatusCode == 429 || e.StatusCode >= 500
}

// IsRateLimit returns true if error is rate limit
func (e *RootlyAPIError) IsRateLimit() bool {
    return e.StatusCode == 429
}

// IsValidation returns true if error is validation
func (e *RootlyAPIError) IsValidation() bool {
    return e.StatusCode == 422
}

// IsAuth returns true if error is authentication
func (e *RootlyAPIError) IsAuth() bool {
    return e.StatusCode == 401
}

// IsNotFound returns true if error is not found
func (e *RootlyAPIError) IsNotFound() bool {
    return e.StatusCode == 404
}

// IsConflict returns true if error is conflict (already resolved)
func (e *RootlyAPIError) IsConflict() bool {
    return e.StatusCode == 409
}
```

### 6.2 Error Classification

```
Rootly API Errors
    │
    ├─▶ Transient (retry)
    │   ├─▶ 429: Rate Limit (wait Retry-After)
    │   ├─▶ 500: Internal Server Error
    │   ├─▶ 502: Bad Gateway
    │   ├─▶ 503: Service Unavailable
    │   └─▶ 504: Gateway Timeout
    │
    └─▶ Permanent (no retry)
        ├─▶ 400: Bad Request (malformed JSON)
        ├─▶ 401: Unauthorized (invalid API key)
        ├─▶ 403: Forbidden (insufficient permissions)
        ├─▶ 404: Not Found (incident deleted)
        ├─▶ 409: Conflict (already resolved)
        └─▶ 422: Validation Error (field errors)
```

---

## 7. Rate Limiting

### 7.1 Token Bucket Algorithm

```go
import "golang.org/x/time/rate"

// Rate limiter configuration
const (
    RootlyRateLimit = 60  // requests per minute
    RootlyRateBurst = 10  // burst capacity
)

// Create rate limiter
rateLimiter := rate.NewLimiter(
    rate.Limit(RootlyRateLimit / 60.0), // 1 req/sec average
    RootlyRateBurst,                     // burst of 10
)

// Wait for token before request
func (c *defaultRootlyIncidentsClient) waitForRateLimit(ctx context.Context) error {
    err := c.rateLimiter.Wait(ctx)
    if err != nil {
        // Context cancelled or deadline exceeded
        return fmt.Errorf("rate limiter wait failed: %w", err)
    }
    return nil
}
```

### 7.2 Rate Limit Monitoring

```go
// Record rate limit hit
func (m *RootlyMetrics) RecordRateLimitHit() {
    m.rateLimitHitsTotal.Inc()
}

// Alert if rate limit hits > 10 in 5 minutes
# Alert: RootlyRateLimitExceeded
expr: rate(rootly_rate_limit_hits_total[5m]) > 10
for: 5m
annotations:
  summary: "Rootly rate limit exceeded"
  description: "More than 10 rate limit hits in 5 minutes"
```

---

## 8. Retry Logic

### 8.1 Exponential Backoff

```go
type RetryConfig struct {
    MaxRetries int           // Default: 3
    BaseDelay  time.Duration // Default: 100ms
    MaxDelay   time.Duration // Default: 5s
    Multiplier float64       // Default: 2.0
}

// Retry schedule:
// Attempt 1: immediate
// Attempt 2: 100ms delay
// Attempt 3: 200ms delay
// Attempt 4: 400ms delay
// Max 3 retries = 4 total attempts
```

### 8.2 Retry Decision Matrix

| Status Code | Error Type | Retry? | Delay |
|-------------|------------|--------|-------|
| **200-299** | Success | No | - |
| **400** | Bad Request | No | - |
| **401** | Unauthorized | No | - |
| **403** | Forbidden | No | - |
| **404** | Not Found | No | - |
| **409** | Conflict | No | - |
| **422** | Validation | No | - |
| **429** | Rate Limit | Yes | Retry-After or backoff |
| **500** | Server Error | Yes | Exponential backoff |
| **502** | Bad Gateway | Yes | Exponential backoff |
| **503** | Unavailable | Yes | Exponential backoff |
| **504** | Timeout | Yes | Exponential backoff |

---

## 9. Incident ID Tracking

### 9.1 IncidentIDCache

```go
// IncidentIDCache stores incident IDs for updates/resolution
type IncidentIDCache interface {
    Set(fingerprint, incidentID string)
    Get(fingerprint string) (incidentID string, exists bool)
    Delete(fingerprint string)
    Size() int
}

// Implementation using sync.Map
type inMemoryIncidentCache struct {
    data      sync.Map
    ttl       time.Duration
    ticker    *time.Ticker
    stopChan  chan struct{}
}

func NewIncidentIDCache(ttl time.Duration) IncidentIDCache {
    cache := &inMemoryIncidentCache{
        data:     sync.Map{},
        ttl:      ttl, // Default: 24h
        ticker:   time.NewTicker(1 * time.Hour),
        stopChan: make(chan struct{}),
    }

    // Start cleanup goroutine
    go cache.cleanup()

    return cache
}

func (c *inMemoryIncidentCache) Set(fingerprint, incidentID string) {
    c.data.Store(fingerprint, cacheEntry{
        incidentID: incidentID,
        expiresAt:  time.Now().Add(c.ttl),
    })
}

func (c *inMemoryIncidentCache) Get(fingerprint string) (string, bool) {
    value, exists := c.data.Load(fingerprint)
    if !exists {
        return "", false
    }

    entry := value.(cacheEntry)

    // Check if expired
    if time.Now().After(entry.expiresAt) {
        c.data.Delete(fingerprint)
        return "", false
    }

    return entry.incidentID, true
}

func (c *inMemoryIncidentCache) Delete(fingerprint string) {
    c.data.Delete(fingerprint)
}

func (c *inMemoryIncidentCache) Size() int {
    count := 0
    c.data.Range(func(_, _ interface{}) bool {
        count++
        return true
    })
    return count
}

// cleanup removes expired entries every hour
func (c *inMemoryIncidentCache) cleanup() {
    for {
        select {
        case <-c.ticker.C:
            c.data.Range(func(key, value interface{}) bool {
                entry := value.(cacheEntry)
                if time.Now().After(entry.expiresAt) {
                    c.data.Delete(key)
                }
                return true
            })
        case <-c.stopChan:
            c.ticker.Stop()
            return
        }
    }
}

type cacheEntry struct {
    incidentID string
    expiresAt  time.Time
}
```

### 9.2 Cache Persistence (Future Enhancement, Phase 10+)

**Problem**: In-memory cache lost on pod restart → duplicate incidents

**Solution** (optional):
- Persistent cache using Redis
- Store: `SETEX alert:{fingerprint} 86400 {incident_id}`
- Retrieve: `GET alert:{fingerprint}`
- Delete: `DEL alert:{fingerprint}`

---

## 10. Metrics & Observability

### 10.1 Prometheus Metrics

```go
type RootlyMetrics struct {
    // Counter: Total incidents created
    incidentsCreatedTotal *prometheus.CounterVec // Labels: severity

    // Counter: Total incidents updated
    incidentsUpdatedTotal *prometheus.CounterVec // Labels: reason

    // Counter: Total incidents resolved
    incidentsResolvedTotal prometheus.Counter

    // Counter: Total API requests
    apiRequestsTotal *prometheus.CounterVec // Labels: endpoint, method, status

    // Histogram: API request duration
    apiDurationSeconds *prometheus.HistogramVec // Labels: endpoint, method

    // Counter: API errors
    apiErrorsTotal *prometheus.CounterVec // Labels: endpoint, error_type

    // Counter: Rate limit hits
    rateLimitHitsTotal prometheus.Counter

    // Gauge: Active incidents tracked in cache
    activeIncidentsGauge prometheus.Gauge
}

func NewRootlyMetrics() *RootlyMetrics {
    m := &RootlyMetrics{
        incidentsCreatedTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "rootly_incidents_created_total",
                Help: "Total number of Rootly incidents created",
            },
            []string{"severity"},
        ),
        incidentsUpdatedTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "rootly_incidents_updated_total",
                Help: "Total number of Rootly incidents updated",
            },
            []string{"reason"},
        ),
        incidentsResolvedTotal: prometheus.NewCounter(
            prometheus.CounterOpts{
                Name: "rootly_incidents_resolved_total",
                Help: "Total number of Rootly incidents resolved",
            },
        ),
        apiRequestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "rootly_api_requests_total",
                Help: "Total number of Rootly API requests",
            },
            []string{"endpoint", "method", "status"},
        ),
        apiDurationSeconds: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "rootly_api_duration_seconds",
                Help:    "Rootly API request duration in seconds",
                Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
            },
            []string{"endpoint", "method"},
        ),
        apiErrorsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "rootly_api_errors_total",
                Help: "Total number of Rootly API errors",
            },
            []string{"endpoint", "error_type"},
        ),
        rateLimitHitsTotal: prometheus.NewCounter(
            prometheus.CounterOpts{
                Name: "rootly_rate_limit_hits_total",
                Help: "Total number of rate limit hits",
            },
        ),
        activeIncidentsGauge: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "rootly_active_incidents_gauge",
                Help: "Number of active incidents tracked in cache",
            },
        ),
    }

    // Register all metrics
    prometheus.MustRegister(
        m.incidentsCreatedTotal,
        m.incidentsUpdatedTotal,
        m.incidentsResolvedTotal,
        m.apiRequestsTotal,
        m.apiDurationSeconds,
        m.apiErrorsTotal,
        m.rateLimitHitsTotal,
        m.activeIncidentsGauge,
    )

    return m
}

// Record methods
func (m *RootlyMetrics) RecordIncidentCreated(severity string) {
    m.incidentsCreatedTotal.WithLabelValues(severity).Inc()
    m.activeIncidentsGauge.Inc()
}

func (m *RootlyMetrics) RecordIncidentUpdated(reason string) {
    m.incidentsUpdatedTotal.WithLabelValues(reason).Inc()
}

func (m *RootlyMetrics) RecordIncidentResolved() {
    m.incidentsResolvedTotal.Inc()
    m.activeIncidentsGauge.Dec()
}

func (m *RootlyMetrics) RecordAPIRequest(endpoint, method string, status int, duration time.Duration) {
    m.apiRequestsTotal.WithLabelValues(endpoint, method, fmt.Sprintf("%d", status)).Inc()
    m.apiDurationSeconds.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

func (m *RootlyMetrics) RecordError(endpoint string, err error) {
    errorType := "unknown"
    if rootlyErr, ok := err.(*RootlyAPIError); ok {
        if rootlyErr.IsRateLimit() {
            errorType = "rate_limit"
        } else if rootlyErr.IsValidation() {
            errorType = "validation"
        } else if rootlyErr.IsAuth() {
            errorType = "auth"
        } else if rootlyErr.IsNotFound() {
            errorType = "not_found"
        } else if rootlyErr.StatusCode >= 500 {
            errorType = "server_error"
        }
    }

    m.apiErrorsTotal.WithLabelValues(endpoint, errorType).Inc()
}
```

### 10.2 Grafana Dashboard Queries

```promql
# Incident creation rate (per minute)
rate(rootly_incidents_created_total[5m]) * 60

# API latency p50, p95, p99
histogram_quantile(0.50, rate(rootly_api_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(rootly_api_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(rootly_api_duration_seconds_bucket[5m]))

# Error rate (%)
rate(rootly_api_errors_total[5m]) / rate(rootly_api_requests_total[5m]) * 100

# Rate limit hit rate
rate(rootly_rate_limit_hits_total[5m])

# Active incidents
rootly_active_incidents_gauge
```

---

## 11. Configuration

### 11.1 Environment Variables

```go
type RootlyConfig struct {
    // API Configuration
    APIKey     string        // ROOTLY_API_KEY (required)
    BaseURL    string        // ROOTLY_API_URL (default: https://api.rootly.com/v1)
    Timeout    time.Duration // ROOTLY_API_TIMEOUT (default: 10s)

    // Rate Limiting
    RateLimit  int           // ROOTLY_RATE_LIMIT (default: 60 req/min)
    RateBurst  int           // ROOTLY_RATE_BURST (default: 10)

    // Retry Logic
    MaxRetries    int           // ROOTLY_MAX_RETRIES (default: 3)
    RetryBaseDelay time.Duration // ROOTLY_RETRY_BASE_DELAY (default: 100ms)
    RetryMaxDelay  time.Duration // ROOTLY_RETRY_MAX_DELAY (default: 5s)

    // Incident Tracking
    CacheTTL time.Duration // ROOTLY_INCIDENT_CACHE_TTL (default: 24h)
}

func LoadRootlyConfig() (*RootlyConfig, error) {
    config := &RootlyConfig{
        // Defaults
        BaseURL:        getEnvOrDefault("ROOTLY_API_URL", "https://api.rootly.com/v1"),
        Timeout:        getEnvDurationOrDefault("ROOTLY_API_TIMEOUT", 10*time.Second),
        RateLimit:      getEnvIntOrDefault("ROOTLY_RATE_LIMIT", 60),
        RateBurst:      getEnvIntOrDefault("ROOTLY_RATE_BURST", 10),
        MaxRetries:     getEnvIntOrDefault("ROOTLY_MAX_RETRIES", 3),
        RetryBaseDelay: getEnvDurationOrDefault("ROOTLY_RETRY_BASE_DELAY", 100*time.Millisecond),
        RetryMaxDelay:  getEnvDurationOrDefault("ROOTLY_RETRY_MAX_DELAY", 5*time.Second),
        CacheTTL:       getEnvDurationOrDefault("ROOTLY_INCIDENT_CACHE_TTL", 24*time.Hour),
    }

    // Validate required fields
    config.APIKey = os.Getenv("ROOTLY_API_KEY")
    if config.APIKey == "" {
        return nil, fmt.Errorf("ROOTLY_API_KEY is required")
    }

    return config, nil
}
```

---

## 12. Testing Strategy

### 12.1 Unit Tests (50 tests)

**API Client Tests** (20 tests):
- Authentication header
- Rate limiting
- Request building
- Response parsing
- Error parsing
- Retry logic (exponential backoff)
- Transient vs permanent errors
- Context cancellation

**Publisher Tests** (15 tests):
- Create incident (firing alert)
- Update incident (annotation change)
- Resolve incident (resolved alert)
- Incident ID caching
- Cache lookup
- Error handling
- Metrics recording

**Model Tests** (10 tests):
- Request serialization
- Response deserialization
- Validation (title, severity, dates)
- Custom fields
- Tags

**Error Tests** (5 tests):
- RootlyAPIError parsing
- Error classification (retryable, rate limit, validation)
- Error wrapping
- Error messages

### 12.2 Integration Tests (15 tests)

**Mock Rootly API Server**:
```go
func TestRootlyIntegration(t *testing.T) {
    // Create mock Rootly API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify headers
        assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
        assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

        // Route to appropriate handler
        switch r.URL.Path {
        case "/incidents":
            handleCreateIncident(w, r)
        case "/incidents/01HKXYZ123":
            handleUpdateIncident(w, r)
        case "/incidents/01HKXYZ123/resolve":
            handleResolveIncident(w, r)
        default:
            http.NotFound(w, r)
        }
    }))
    defer server.Close()

    // Create client with mock server
    client := NewRootlyIncidentsClient(ClientConfig{
        BaseURL: server.URL,
        APIKey:  "test-api-key",
        Timeout: 5 * time.Second,
    }, slog.Default())

    // Test create incident
    resp, err := client.CreateIncident(context.Background(), &CreateIncidentRequest{
        Title:       "[CRITICAL] Test Alert",
        Description: "Test description",
        Severity:    "critical",
        StartedAt:   time.Now(),
    })

    assert.NoError(t, err)
    assert.NotEmpty(t, resp.Data.ID)
}
```

**Scenarios**:
1. Successful create → update → resolve flow
2. Rate limit (429) → retry → success
3. Server error (500) → retry → success
4. Validation error (422) → no retry → fail
5. Authentication error (401) → no retry → fail
6. Not found (404) → handle gracefully
7. Conflict (409) → handle gracefully
8. Network timeout → retry → success
9. Context cancellation → stop immediately
10. Concurrent requests (100 goroutines)
11. Cache expiration (24h TTL)
12. Incident ID lookup
13. Incident ID delete after resolution
14. Metrics recording
15. Error classification

### 12.3 Benchmarks (10 benchmarks)

```go
BenchmarkCreateIncident       // Target: <300ms
BenchmarkUpdateIncident       // Target: <250ms
BenchmarkResolveIncident      // Target: <200ms
BenchmarkRateLimiter          // Target: <1ms
BenchmarkIncidentCacheSet     // Target: <10μs
BenchmarkIncidentCacheGet     // Target: <10μs
BenchmarkIncidentCacheDelete  // Target: <10μs
BenchmarkJSONMarshal          // Request serialization
BenchmarkJSONUnmarshal        // Response deserialization
BenchmarkErrorParsing         // Error JSON parsing
```

---

## 13. Deployment

### 13.1 K8s Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-credentials
  namespace: alert-history
type: Opaque
stringData:
  api-key: "<rootly-api-key>"  # From Rootly dashboard

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: rootly-config
  namespace: alert-history
data:
  ROOTLY_API_URL: "https://api.rootly.com/v1"
  ROOTLY_API_TIMEOUT: "10s"
  ROOTLY_RATE_LIMIT: "60"
  ROOTLY_RATE_BURST: "10"
  ROOTLY_MAX_RETRIES: "3"
  ROOTLY_RETRY_BASE_DELAY: "100ms"
  ROOTLY_RETRY_MAX_DELAY: "5s"
  ROOTLY_INCIDENT_CACHE_TTL: "24h"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-history
  namespace: alert-history
spec:
  template:
    spec:
      containers:
      - name: alert-history
        image: alert-history:latest
        env:
        - name: ROOTLY_API_KEY
          valueFrom:
            secretKeyRef:
              name: rootly-credentials
              key: api-key
        envFrom:
        - configMapRef:
            name: rootly-config
```

### 13.2 Helm Chart

```yaml
# values.yaml
rootly:
  enabled: true
  apiUrl: "https://api.rootly.com/v1"
  apiKey: ""  # Must be provided
  timeout: "10s"
  rateLimit: 60
  rateBurst: 10
  maxRetries: 3
  retryBaseDelay: "100ms"
  retryMaxDelay: "5s"
  cacheTTL: "24h"

# templates/secret.yaml
{{- if .Values.rootly.enabled }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "alert-history.fullname" . }}-rootly
type: Opaque
stringData:
  api-key: {{ .Values.rootly.apiKey | required "rootly.apiKey is required" }}
{{- end }}
```

---

## 14. Performance Optimization

### 14.1 Optimization Strategies

1. **Connection Pooling**: Reuse HTTP connections (http.Transport)
2. **JSON Optimization**: Use json.Decoder for streaming
3. **Memory Reuse**: Reuse request/response buffers
4. **Goroutine Pool**: Limit concurrent requests (rate limiter)
5. **Cache**: In-memory incident ID storage (O(1) lookup)

### 14.2 Performance Targets

| Operation | p50 | p99 | p999 |
|-----------|-----|-----|------|
| **Create Incident** | <300ms | <500ms | <1s |
| **Update Incident** | <250ms | <400ms | <800ms |
| **Resolve Incident** | <200ms | <350ms | <700ms |
| **Rate Limiter** | <1ms | <2ms | <5ms |
| **Cache Lookup** | <10μs | <50μs | <100μs |

---

## Document Metadata

**Version**: 1.0
**Author**: AI Assistant (TN-052 Design - 150% Quality)
**Date**: 2025-11-08
**Status**: 🏗️ **DESIGN COMPLETE**
**Branch**: `feature/TN-052-rootly-publisher-150pct-comprehensive`
**Next**: tasks.md (Phase 3)

**Change Log**:
- 2025-11-08: Comprehensive technical design (1,800+ LOC)
- 5-layer architecture defined
- RootlyIncidentsClient designed (400 LOC)
- Enhanced RootlyPublisher designed (350 LOC)
- Data models specified
- Error handling strategy complete
- Rate limiting + retry logic designed
- Metrics + observability framework
- Testing strategy (75 tests)
- Deployment configuration

---

**🏗️ Design Complete - Ready for Phase 3 (Implementation Plan)**

**Key Design**: Full Rootly Incidents API v1 integration with incident lifecycle management (create → update → resolve), intelligent retry logic, rate limiting (60 req/min), 8 Prometheus metrics, 95%+ test coverage target.
