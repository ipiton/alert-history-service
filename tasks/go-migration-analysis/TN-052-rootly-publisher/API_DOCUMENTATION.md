# TN-052: Rootly Publisher - API Documentation

**Date**: 2025-11-10
**Phase**: Phase 8 - API Documentation
**Version**: 1.0.0
**API**: Rootly Incidents API v1

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Rate Limiting](#rate-limiting)
4. [Error Handling](#error-handling)
5. [API Endpoints](#api-endpoints)
6. [Data Models](#data-models)
7. [Code Examples](#code-examples)
8. [Best Practices](#best-practices)

---

## 🎯 Overview

The **EnhancedRootlyPublisher** provides a complete Go client for the [Rootly Incidents API v1](https://docs.rootly.com/api/incidents), enabling automated incident management for Alert History.

**Capabilities:**
- ✅ Create incidents from firing alerts
- ✅ Update incidents from alert changes
- ✅ Resolve incidents from resolved alerts
- ✅ Rate limiting (60 req/min)
- ✅ Automatic retries with exponential backoff
- ✅ TLS 1.2+ security
- ✅ Prometheus metrics

**Base URL:** `https://api.rootly.com/v1`

---

## 🔐 Authentication

### API Key

All requests require Bearer token authentication:

```http
Authorization: Bearer rtly_abc123xyz...
```

### Obtaining an API Key

1. Log in to [Rootly](https://rootly.com)
2. Navigate to **Settings** → **API Tokens**
3. Click **Create Token**
4. Set permissions: **incidents:write** (required)
5. Copy the token (format: `rtly_XXXXXXXX`)

### Security Best Practices

- ✅ Store API keys in Kubernetes Secrets
- ✅ Use RBAC to restrict access
- ✅ Rotate keys every 90 days
- ✅ Use different keys per environment
- ❌ Never commit keys to version control
- ❌ Never share keys across teams

---

## ⏱️ Rate Limiting

### Limits

| Tier | Requests/Minute | Burst |
|------|----------------|-------|
| **Free** | 30 | 10 |
| **Starter** | 60 | 20 |
| **Pro** | 120 | 40 |
| **Enterprise** | Custom | Custom |

### Implementation

The client implements token bucket rate limiting:

```go
config := ClientConfig{
    BaseURL:   "https://api.rootly.com/v1",
    APIKey:    "rtly_abc123",
    RateLimit: 60,   // requests per minute
    RateBurst: 10,   // burst capacity
}

client := NewRootlyIncidentsClient(config, logger)
```

### Rate Limit Headers

Responses include rate limit information:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 1699564800
```

### Handling Rate Limits

The client automatically handles 429 errors:

1. **Detect**: HTTP 429 Too Many Requests
2. **Wait**: Exponential backoff (100ms → 5s)
3. **Retry**: Up to 3 attempts
4. **Metric**: `rootly_rate_limit_hits_total` counter

---

## 🚨 Error Handling

### Error Response Format

```json
{
  "errors": [
    {
      "status": "422",
      "title": "Validation failed",
      "detail": "title can't be blank",
      "source": {
        "pointer": "/data/attributes/title"
      }
    }
  ]
}
```

### Error Classification

#### Retryable Errors (Auto-Retry)

| Status | Title | Action |
|--------|-------|--------|
| **429** | Rate limit exceeded | Wait + retry (exponential backoff) |
| **500** | Internal server error | Retry (exponential backoff) |
| **502** | Bad gateway | Retry (exponential backoff) |
| **503** | Service unavailable | Retry (exponential backoff) |
| **504** | Gateway timeout | Retry (exponential backoff) |

#### Permanent Errors (No Retry)

| Status | Title | Action |
|--------|-------|--------|
| **400** | Bad request | Fix request payload |
| **401** | Unauthorized | Check API key |
| **403** | Forbidden | Check permissions |
| **404** | Not found | Verify incident ID |
| **409** | Conflict | Handle gracefully (already resolved) |
| **422** | Validation failed | Fix validation errors |

### Error Handling in Code

```go
resp, err := client.CreateIncident(ctx, req)
if err != nil {
    if rootlyErr, ok := err.(*RootlyAPIError); ok {
        switch {
        case rootlyErr.IsRateLimit():
            // Rate limit - client auto-retries
            log.Warn("Rate limit hit", "status", rootlyErr.StatusCode)
        case rootlyErr.IsAuth():
            // Auth error - permanent, check API key
            log.Error("Auth failed", "detail", rootlyErr.Detail)
        case rootlyErr.IsValidation():
            // Validation error - fix payload
            log.Error("Validation failed", "source", rootlyErr.Source)
        case rootlyErr.IsRetryable():
            // Transient error - client auto-retries
            log.Warn("Transient error", "status", rootlyErr.StatusCode)
        default:
            // Unknown error
            log.Error("Unknown error", "err", err)
        }
    }
    return err
}
```

---

## 🌐 API Endpoints

### 1. Create Incident

Creates a new incident in Rootly.

**Endpoint:** `POST /incidents`

**Request:**
```json
{
  "title": "[AlertName] Alert in production",
  "description": "Alert details with AI classification",
  "severity": "critical",
  "started_at": "2024-01-15T10:30:00Z",
  "tags": ["alert", "production", "database"],
  "custom_fields": {
    "alert_name": "DatabaseDown",
    "fingerprint": "abc123",
    "llm_severity": "critical",
    "llm_confidence": 0.95
  }
}
```

**Response:** `201 Created`
```json
{
  "data": {
    "id": "01HKXYZ...",
    "type": "incidents",
    "attributes": {
      "title": "[AlertName] Alert in production",
      "severity": "critical",
      "status": "started",
      "permalink": "https://app.rootly.com/incidents/01HKXYZ",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

**Go Example:**
```go
req := &CreateIncidentRequest{
    Title:       "[DatabaseDown] Alert in production",
    Description: "Database connection pool exhausted",
    Severity:    "critical",
    StartedAt:   time.Now(),
    Tags:        []string{"database", "production"},
    CustomFields: map[string]interface{}{
        "alert_name":    "DatabaseDown",
        "fingerprint":   "abc123",
        "llm_severity":  "critical",
        "llm_confidence": 0.95,
    },
}

resp, err := client.CreateIncident(ctx, req)
if err != nil {
    return fmt.Errorf("create incident: %w", err)
}

log.Info("Incident created",
    "id", resp.GetID(),
    "permalink", resp.Data.Attributes.Permalink)
```

---

### 2. Update Incident

Updates an existing incident.

**Endpoint:** `PATCH /incidents/{id}`

**Request:**
```json
{
  "description": "Updated description with new information",
  "custom_fields": {
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

**Response:** `200 OK`
```json
{
  "data": {
    "id": "01HKXYZ...",
    "type": "incidents",
    "attributes": {
      "title": "[AlertName] Alert in production",
      "status": "started",
      "updated_at": "2024-01-15T10:35:00Z"
    }
  }
}
```

**Go Example:**
```go
req := &UpdateIncidentRequest{
    Description: "Alert still firing, investigating...",
    CustomFields: map[string]interface{}{
        "last_seen": time.Now(),
    },
}

resp, err := client.UpdateIncident(ctx, incidentID, req)
if err != nil {
    if IsNotFoundError(err) {
        // Incident deleted in Rootly, recreate
        log.Warn("Incident not found, recreating")
        return recreateIncident(ctx, alert)
    }
    return fmt.Errorf("update incident: %w", err)
}

log.Info("Incident updated", "id", incidentID)
```

---

### 3. Resolve Incident

Resolves an incident.

**Endpoint:** `POST /incidents/{id}/resolve`

**Request:**
```json
{
  "summary": "Alert resolved: Database connection restored"
}
```

**Response:** `200 OK`
```json
{
  "data": {
    "id": "01HKXYZ...",
    "type": "incidents",
    "attributes": {
      "status": "resolved",
      "resolved_at": "2024-01-15T11:00:00Z"
    }
  }
}
```

**Go Example:**
```go
req := &ResolveIncidentRequest{
    Summary: "Alert resolved: Database connection restored",
}

resp, err := client.ResolveIncident(ctx, incidentID, req)
if err != nil {
    if IsConflictError(err) {
        // Incident already resolved, not an error
        log.Info("Incident already resolved", "id", incidentID)
        return nil
    }
    return fmt.Errorf("resolve incident: %w", err)
}

log.Info("Incident resolved",
    "id", incidentID,
    "resolved_at", resp.Data.Attributes.ResolvedAt)
```

---

## 📦 Data Models

### CreateIncidentRequest

```go
type CreateIncidentRequest struct {
    Title        string                 `json:"title"`         // Required: 5-255 chars
    Description  string                 `json:"description"`   // Required: 10-10000 chars
    Severity     string                 `json:"severity"`      // Required: critical|major|minor|low
    StartedAt    time.Time              `json:"started_at"`    // Required: Incident start time
    Tags         []string               `json:"tags,omitempty"` // Optional: Max 20 tags
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // Optional: Custom data
}
```

**Validation Rules:**
- `title`: 5-255 characters
- `description`: 10-10,000 characters
- `severity`: Must be one of: `critical`, `major`, `minor`, `low`
- `tags`: Maximum 20 tags

---

### UpdateIncidentRequest

```go
type UpdateIncidentRequest struct {
    Description  string                 `json:"description,omitempty"`   // Optional: 10-10000 chars
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // Optional: Custom data
}
```

**Notes:**
- All fields are optional
- Only specified fields are updated
- Cannot update `title` or `severity` after creation

---

### ResolveIncidentRequest

```go
type ResolveIncidentRequest struct {
    Summary string `json:"summary,omitempty"` // Optional: Resolution summary (max 1000 chars)
}
```

---

### IncidentResponse

```go
type IncidentResponse struct {
    Data struct {
        ID         string    `json:"id"`   // Incident ID (e.g., "01HKXYZ...")
        Type       string    `json:"type"` // "incidents"
        Attributes struct {
            Title      string     `json:"title"`
            Severity   string     `json:"severity"`
            Status     string     `json:"status"`     // "started"|"resolved"
            Permalink  string     `json:"permalink"`  // Rootly UI link
            CreatedAt  time.Time  `json:"created_at"`
            UpdatedAt  time.Time  `json:"updated_at,omitempty"`
            ResolvedAt *time.Time `json:"resolved_at,omitempty"`
        } `json:"attributes"`
    } `json:"data"`
}

// Helper methods
func (r *IncidentResponse) GetID() string { return r.Data.ID }
func (r *IncidentResponse) GetStatus() string { return r.Data.Attributes.Status }
func (r *IncidentResponse) IsResolved() bool { return r.Data.Attributes.Status == "resolved" }
```

---

## 💻 Code Examples

### Complete Incident Lifecycle

```go
package main

import (
    "context"
    "log/slog"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

func main() {
    logger := slog.Default()

    // Create client
    config := publishing.ClientConfig{
        BaseURL: "https://api.rootly.com/v1",
        APIKey:  "rtly_abc123xyz...",
        Timeout: 10 * time.Second,
    }
    client := publishing.NewRootlyIncidentsClient(config, logger)

    ctx := context.Background()

    // 1. Create incident
    createReq := &publishing.CreateIncidentRequest{
        Title:       "[DatabaseDown] Alert in production",
        Description: "Database connection pool exhausted",
        Severity:    "critical",
        StartedAt:   time.Now(),
        Tags:        []string{"database", "production"},
    }

    incident, err := client.CreateIncident(ctx, createReq)
    if err != nil {
        logger.Error("Failed to create incident", "error", err)
        return
    }
    logger.Info("Incident created", "id", incident.GetID())

    // 2. Update incident
    updateReq := &publishing.UpdateIncidentRequest{
        Description: "Investigation ongoing...",
    }

    _, err = client.UpdateIncident(ctx, incident.GetID(), updateReq)
    if err != nil {
        logger.Error("Failed to update incident", "error", err)
        return
    }
    logger.Info("Incident updated")

    // 3. Resolve incident
    resolveReq := &publishing.ResolveIncidentRequest{
        Summary: "Database connection restored",
    }

    resolved, err := client.ResolveIncident(ctx, incident.GetID(), resolveReq)
    if err != nil {
        logger.Error("Failed to resolve incident", "error", err)
        return
    }
    logger.Info("Incident resolved", "resolved_at", resolved.Data.Attributes.ResolvedAt)
}
```

---

### Using EnhancedRootlyPublisher

```go
package main

import (
    "context"
    "log/slog"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

func main() {
    logger := slog.Default()

    // Create dependencies
    client := publishing.NewRootlyIncidentsClient(
        publishing.ClientConfig{
            BaseURL: "https://api.rootly.com/v1",
            APIKey:  "rtly_abc123",
        },
        logger,
    )
    cache := publishing.NewIncidentIDCache(24 * time.Hour)
    metrics := publishing.NewRootlyMetrics()
    formatter := publishing.NewAlertFormatter()

    // Create publisher
    publisher := publishing.NewEnhancedRootlyPublisher(
        client,
        cache,
        metrics,
        formatter,
        logger,
    )

    // Publish firing alert
    enrichedAlert := &core.EnrichedAlert{
        Alert: &core.Alert{
            Fingerprint: "abc123",
            AlertName:   "DatabaseDown",
            Status:      core.StatusFiring,
            StartsAt:    time.Now(),
            Labels:      map[string]string{"severity": "critical"},
        },
    }

    target := &core.PublishingTarget{
        Name: "rootly-prod",
        Type: "rootly",
        URL:  "https://api.rootly.com/v1/incidents",
    }

    err := publisher.Publish(context.Background(), enrichedAlert, target)
    if err != nil {
        logger.Error("Failed to publish", "error", err)
        return
    }

    logger.Info("Alert published to Rootly")
}
```

---

### Error Handling Patterns

```go
func handleIncidentCreation(ctx context.Context, client RootlyIncidentsClient, req *CreateIncidentRequest) error {
    resp, err := client.CreateIncident(ctx, req)
    if err != nil {
        // Type assertion to RootlyAPIError
        if rootlyErr, ok := err.(*RootlyAPIError); ok {
            switch {
            case rootlyErr.IsRateLimit():
                // Rate limit - client auto-retries, log warning
                log.Warn("Rate limit encountered",
                    "status", rootlyErr.StatusCode,
                    "retry_count", rootlyErr.RetryCount)
                return nil // Retry succeeded

            case rootlyErr.IsAuth():
                // Authentication failure - permanent error
                log.Error("Authentication failed",
                    "detail", rootlyErr.Detail)
                // Alert ops team, rotate API key
                alertOpsTeam("Rootly API key invalid")
                return fmt.Errorf("auth error: %w", rootlyErr)

            case rootlyErr.IsValidation():
                // Validation error - fix payload
                log.Error("Validation failed",
                    "field", rootlyErr.Source,
                    "detail", rootlyErr.Detail)
                // Fix validation issue and retry
                return fixValidationAndRetry(req, rootlyErr)

            case rootlyErr.IsNotFound():
                // Resource not found (for updates/resolves)
                log.Warn("Incident not found, may have been deleted")
                return nil // Not an error

            case rootlyErr.IsConflict():
                // Already resolved
                log.Info("Incident already resolved")
                return nil // Not an error

            case rootlyErr.IsRetryable():
                // Transient error - client auto-retries
                log.Warn("Transient error",
                    "status", rootlyErr.StatusCode)
                // Client handled retries, this is final failure
                return fmt.Errorf("retries exhausted: %w", rootlyErr)

            default:
                // Unknown error
                log.Error("Unknown Rootly error",
                    "status", rootlyErr.StatusCode,
                    "title", rootlyErr.Title,
                    "detail", rootlyErr.Detail)
                return fmt.Errorf("unknown error: %w", rootlyErr)
            }
        }

        // Non-Rootly error (network, timeout, etc.)
        log.Error("Network error", "error", err)
        return fmt.Errorf("network error: %w", err)
    }

    // Success
    log.Info("Incident created successfully",
        "id", resp.GetID(),
        "permalink", resp.Data.Attributes.Permalink)
    return nil
}
```

---

## 🎯 Best Practices

### 1. Incident Fingerprinting

Use stable, unique fingerprints:

```go
// ✅ Good: Consistent fingerprint
fingerprint := fmt.Sprintf("%s-%s-%s",
    alert.Name,
    alert.Namespace,
    hashLabels(alert.Labels))

// ❌ Bad: Timestamp-based (creates duplicates)
fingerprint := fmt.Sprintf("alert-%d", time.Now().Unix())
```

### 2. Custom Fields

Store structured data for debugging:

```go
CustomFields: map[string]interface{}{
    "alert_name":          alert.Name,
    "fingerprint":         alert.Fingerprint,
    "generator_url":       alert.GeneratorURL,
    "llm_severity":        classification.Severity,
    "llm_confidence":      classification.Confidence,
    "llm_recommendations": classification.Recommendations,
    "k8s_namespace":       alert.Namespace,
    "k8s_pod":             alert.Labels["pod"],
}
```

### 3. Retry Strategy

Configure retries based on environment:

```go
// Production: Aggressive retries
RetryConfig: RetryConfig{
    MaxRetries: 5,
    BaseDelay:  200 * time.Millisecond,
    MaxDelay:   10 * time.Second,
}

// Development: Fast fail
RetryConfig: RetryConfig{
    MaxRetries: 1,
    BaseDelay:  100 * time.Millisecond,
    MaxDelay:   1 * time.Second,
}
```

### 4. Monitoring

Set up alerts on key metrics:

```promql
# Alert on high error rate
rate(rootly_api_errors_total[5m]) > 0.1

# Alert on rate limit hits
rate(rootly_rate_limit_hits_total[5m]) > 10

# Alert on slow API responses
histogram_quantile(0.95, rate(rootly_api_duration_seconds_bucket[5m])) > 2
```

---

## 🔗 Related Documentation

- **Rootly API Official Docs**: https://docs.rootly.com/api/incidents
- **Integration Guide**: `INTEGRATION_GUIDE.md`
- **Design Document**: `design.md`
- **Requirements**: `requirements.md`
- **Test Suite**: `go-app/internal/infrastructure/publishing/rootly_*_test.go`

---

## 📊 Metrics Reference

See `INTEGRATION_GUIDE.md` for complete metrics documentation.

---

**Status**: ✅ **PHASE 8 COMPLETE**
**Total**: ~1,000 LOC API Documentation
**Next**: Final summary
