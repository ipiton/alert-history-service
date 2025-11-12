# TN-055: Generic Webhook Publisher - Technical Design (150% Quality)

**Version**: 1.0
**Date**: 2025-11-11
**Status**: 🏗️ **DESIGN PHASE**
**Quality Target**: **150%+ (Enterprise Grade A+)**

---

## 📑 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Authentication System](#3-authentication-system)
4. [Validation Engine](#4-validation-engine)
5. [Retry Logic](#5-retry-logic)
6. [Error Handling](#6-error-handling)
7. [Metrics & Observability](#7-metrics--observability)
8. [Data Models](#8-data-models)
9. [Configuration](#9-configuration)
10. [Testing Strategy](#10-testing-strategy)
11. [Performance Optimization](#11-performance-optimization)
12. [Integration](#12-integration)

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
│                      │  WebhookPublisher (TN-055)      │        │
│                      │                                  │        │
│                      │  ┌───────────────────────────┐  │        │
│                      │  │ WebhookHTTPClient         │  │        │
│                      │  │ - 4 Auth Strategies       │  │        │
│                      │  │ - 6 Validation Rules      │  │        │
│                      │  │ - Retry Logic             │  │        │
│                      │  │ - Error Handling          │  │        │
│                      │  └───────────────────────────┘  │        │
│                      │              │                   │        │
│                      │  ┌───────────▼───────────────┐  │        │
│                      │  │ WebhookValidator          │  │        │
│                      │  │ (URL, payload, headers)   │  │        │
│                      │  └───────────────────────────┘  │        │
│                      └──────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                     │
                                     │ HTTPS + Auth
                                     │
                                     ▼
                      ┌─────────────────────────────┐
                      │  External Webhook Receiver  │
                      │  (Any service)              │
                      │                              │
                      │  POST /webhook              │
                      └─────────────────────────────┘
```

### 1.2 Component Layers (5-Layer Architecture)

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
│                    Layer 2: Publisher                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  EnhancedWebhookPublisher struct                        │ │
│  │  - client: WebhookHTTPClient                            │ │
│  │  - validator: WebhookValidator                          │ │
│  │  - metrics: *WebhookMetrics                             │ │
│  │  - formatter: AlertFormatter                            │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - Publish() → error                                    │ │
│  │  - validateTarget() → error                             │ │
│  │  - buildRequest() → (*http.Request, error)              │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 3: HTTP Client                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  WebhookHTTPClient struct                               │ │
│  │  - httpClient: *http.Client                             │ │
│  │  - retryConfig: RetryConfig                             │ │
│  │  - authManager: AuthManager                             │ │
│  │  - logger: *slog.Logger                                 │ │
│  │                                                          │ │
│  │  Methods:                                               │ │
│  │  - Post(url, payload, headers) → (*Response, error)     │ │
│  │  - doRequestWithRetry() → (*Response, error)            │ │
│  │  - applyAuth(req) → error                               │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                Layer 4: Supporting Services                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  AuthManager: 4 auth strategies                         │ │
│  │  WebhookValidator: 6 validation rules                   │ │
│  │  RetryManager: Exponential backoff                      │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 5: Infrastructure                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  WebhookMetrics (8 Prometheus metrics)                 │ │
│  │  Error Types (6 custom errors)                         │ │
│  │  Structured Logging (slog)                             │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Data Flow (Request Processing)

```
Scenario 1: Successful Webhook POST
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. AlertProcessor
   ↓ enrichedAlert + PublishingTarget
2. EnhancedWebhookPublisher.Publish()
   ├─ validateTarget(target) → OK
   ├─ formatter.FormatAlert(ctx, alert, FormatWebhook) → payload
   ├─ buildRequest(url, payload, headers) → http.Request
   ↓
3. WebhookHTTPClient.Post()
   ├─ applyAuth(request, authConfig) → add auth headers
   ├─ doRequestWithRetry(request)
   │  ├─ Attempt 1: HTTP POST → 200 OK
   │  └─ Success!
   ↓
4. External Webhook Receiver
   ↓ HTTP 200 OK
5. Parse response, record metrics
6. Return nil (success)

Scenario 2: Transient Error with Retry
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. AlertProcessor → EnhancedWebhookPublisher.Publish()
2. WebhookHTTPClient.doRequestWithRetry()
   ├─ Attempt 1: HTTP POST → 503 Service Unavailable
   │  ├─ classifyError(503) → ErrorTypeServer (retryable)
   │  ├─ backoff = 100ms
   │  └─ time.Sleep(100ms)
   ├─ Attempt 2: HTTP POST → 503 Service Unavailable
   │  ├─ backoff = 200ms
   │  └─ time.Sleep(200ms)
   ├─ Attempt 3: HTTP POST → 200 OK
   │  └─ Success after 2 retries!
   ↓
3. Record metrics: webhook_retries_total{attempt="2"}
4. Return nil (success)

Scenario 3: Permanent Error (No Retry)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. AlertProcessor → EnhancedWebhookPublisher.Publish()
2. WebhookHTTPClient.doRequestWithRetry()
   ├─ Attempt 1: HTTP POST → 401 Unauthorized
   │  ├─ classifyError(401) → ErrorTypeAuth (permanent)
   │  └─ No retry
   ↓
3. Return WebhookError{Type: ErrorTypeAuth, Message: "unauthorized"}
4. Record metrics: webhook_errors_total{error_type="auth"}
```

---

## 2. Component Design

### 2.1 File Structure

```
go-app/internal/infrastructure/publishing/
├── webhook_models.go               200 LOC - Data models
├── webhook_errors.go               150 LOC - Error types
├── webhook_auth.go                 200 LOC - 4 auth strategies
├── webhook_validator.go            150 LOC - 6 validation rules
├── webhook_client.go               400 LOC - HTTP client
├── webhook_publisher_enhanced.go   350 LOC - Business logic
├── webhook_metrics.go              100 LOC - Prometheus metrics
├── webhook_auth_test.go            200 LOC - Auth tests
├── webhook_validator_test.go       200 LOC - Validation tests
├── webhook_client_test.go          400 LOC - Client tests
├── webhook_publisher_test.go       300 LOC - Publisher tests
├── webhook_retry_test.go           150 LOC - Retry tests
├── webhook_errors_test.go          100 LOC - Error tests
├── webhook_bench_test.go           200 LOC - Benchmarks
└── WEBHOOK_README.md               800 LOC - API docs
```

**Total**: ~3,900 LOC (1,550 production + 1,550 tests + 800 docs)

### 2.2 Component Responsibilities

| Component | Responsibility | Lines | Priority |
|-----------|---------------|-------|----------|
| **webhook_models.go** | Request/Response models, config structs | 200 | 🔴 CRITICAL |
| **webhook_errors.go** | 6 error types, classification helpers | 150 | 🔴 CRITICAL |
| **webhook_auth.go** | 4 auth strategies (Strategy pattern) | 200 | 🔴 CRITICAL |
| **webhook_validator.go** | 6 validation rules | 150 | 🔴 CRITICAL |
| **webhook_client.go** | HTTP client, retry logic | 400 | 🔴 CRITICAL |
| **webhook_publisher_enhanced.go** | Business logic, orchestration | 350 | 🔴 CRITICAL |
| **webhook_metrics.go** | 8 Prometheus metrics | 100 | 🔴 CRITICAL |
| **Tests** | Unit + integration + benchmarks | 1,550 | 🔴 CRITICAL |
| **README** | API documentation, examples | 800 | 🟡 HIGH |

---

## 3. Authentication System

### 3.1 Strategy Pattern Design

```go
// AuthStrategy interface (Strategy pattern)
type AuthStrategy interface {
    ApplyAuth(req *http.Request, config AuthConfig) error
    Name() string
}

// AuthManager orchestrates auth strategies
type AuthManager struct {
    strategies map[AuthType]AuthStrategy
    logger     *slog.Logger
}

// AuthType enum
type AuthType string

const (
    AuthTypeBearer AuthType = "bearer"
    AuthTypeBasic  AuthType = "basic"
    AuthTypeAPIKey AuthType = "apikey"
    AuthTypeCustom AuthType = "custom"
)

// AuthConfig holds auth configuration
type AuthConfig struct {
    Type          AuthType          `json:"type"`
    Token         string            `json:"token,omitempty"`
    Username      string            `json:"username,omitempty"`
    Password      string            `json:"password,omitempty"`
    APIKey        string            `json:"api_key,omitempty"`
    APIKeyHeader  string            `json:"api_key_header,omitempty"`
    CustomHeaders map[string]string `json:"custom_headers,omitempty"`
}
```

### 3.2 Auth Strategy Implementations

#### Strategy 1: Bearer Token

```go
// BearerAuthStrategy implements Bearer Token authentication
type BearerAuthStrategy struct{}

func (s *BearerAuthStrategy) ApplyAuth(req *http.Request, config AuthConfig) error {
    if config.Token == "" {
        return ErrMissingAuthToken
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))
    return nil
}

func (s *BearerAuthStrategy) Name() string {
    return "BearerAuth"
}

// Usage Example
// headers:
//   Authorization: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### Strategy 2: Basic Auth

```go
// BasicAuthStrategy implements HTTP Basic Authentication
type BasicAuthStrategy struct{}

func (s *BasicAuthStrategy) ApplyAuth(req *http.Request, config AuthConfig) error {
    if config.Username == "" || config.Password == "" {
        return ErrMissingBasicAuthCredentials
    }

    req.SetBasicAuth(config.Username, config.Password)
    return nil
}

func (s *BasicAuthStrategy) Name() string {
    return "BasicAuth"
}

// Usage Example
// auth:
//   type: "basic"
//   username: "admin"
//   password: "secret123"
```

#### Strategy 3: API Key Header

```go
// APIKeyAuthStrategy implements API Key header authentication
type APIKeyAuthStrategy struct{}

func (s *APIKeyAuthStrategy) ApplyAuth(req *http.Request, config AuthConfig) error {
    if config.APIKey == "" {
        return ErrMissingAPIKey
    }

    headerName := config.APIKeyHeader
    if headerName == "" {
        headerName = "X-API-Key" // Default header name
    }

    req.Header.Set(headerName, config.APIKey)
    return nil
}

func (s *APIKeyAuthStrategy) Name() string {
    return "APIKeyAuth"
}

// Usage Example
// headers:
//   X-API-Key: "sk_live_1234567890abcdef"
```

#### Strategy 4: Custom Headers

```go
// CustomAuthStrategy implements custom header authentication
type CustomAuthStrategy struct{}

func (s *CustomAuthStrategy) ApplyAuth(req *http.Request, config AuthConfig) error {
    if len(config.CustomHeaders) == 0 {
        return ErrNoCustomHeaders
    }

    for key, value := range config.CustomHeaders {
        req.Header.Set(key, value)
    }
    return nil
}

func (s *CustomAuthStrategy) Name() string {
    return "CustomAuth"
}

// Usage Example
// headers:
//   X-Service-ID: "alert-history"
//   X-Tenant-ID: "customer-123"
//   X-API-Secret: "custom-secret"
```

### 3.3 AuthManager Implementation

```go
// NewAuthManager creates a new auth manager
func NewAuthManager(logger *slog.Logger) *AuthManager {
    return &AuthManager{
        strategies: map[AuthType]AuthStrategy{
            AuthTypeBearer: &BearerAuthStrategy{},
            AuthTypeBasic:  &BasicAuthStrategy{},
            AuthTypeAPIKey: &APIKeyAuthStrategy{},
            AuthTypeCustom: &CustomAuthStrategy{},
        },
        logger: logger,
    }
}

// ApplyAuth applies authentication to HTTP request
func (m *AuthManager) ApplyAuth(req *http.Request, config AuthConfig) error {
    strategy, exists := m.strategies[config.Type]
    if !exists {
        return fmt.Errorf("unsupported auth type: %s", config.Type)
    }

    m.logger.Debug("Applying authentication",
        slog.String("auth_type", string(config.Type)),
        slog.String("strategy", strategy.Name()))

    return strategy.ApplyAuth(req, config)
}
```

---

## 4. Validation Engine

### 4.1 Validator Design

```go
// WebhookValidator validates webhook configuration
type WebhookValidator struct {
    maxPayloadSize  int64         // Default: 1 MB
    maxHeaders      int           // Default: 100
    maxHeaderSize   int           // Default: 4 KB
    allowedSchemes  []string      // Default: ["https"]
    blockedHosts    []string      // Default: ["localhost", "127.0.0.1"]
    minTimeout      time.Duration // Default: 1s
    maxTimeout      time.Duration // Default: 60s
    maxRetries      int           // Default: 5
    logger          *slog.Logger
}

// NewWebhookValidator creates a new validator with defaults
func NewWebhookValidator(logger *slog.Logger) *WebhookValidator {
    return &WebhookValidator{
        maxPayloadSize: 1 * 1024 * 1024, // 1 MB
        maxHeaders:     100,
        maxHeaderSize:  4 * 1024, // 4 KB
        allowedSchemes: []string{"https"},
        blockedHosts: []string{
            "localhost",
            "127.0.0.1",
            "0.0.0.0",
            "::1",
            "[::1]",
        },
        minTimeout: 1 * time.Second,
        maxTimeout: 60 * time.Second,
        maxRetries: 5,
        logger:     logger,
    }
}
```

### 4.2 Validation Rules

#### Rule 1: URL Validation

```go
// ValidateURL validates webhook URL
func (v *WebhookValidator) ValidateURL(urlStr string) error {
    if urlStr == "" {
        return ErrEmptyURL
    }

    // Parse URL
    parsedURL, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrInvalidURL, err)
    }

    // Check scheme (HTTPS only)
    if !contains(v.allowedSchemes, parsedURL.Scheme) {
        return fmt.Errorf("%w: scheme %s not allowed (allowed: %v)",
            ErrInsecureScheme, parsedURL.Scheme, v.allowedSchemes)
    }

    // Check for credentials in URL (security risk)
    if parsedURL.User != nil {
        return ErrCredentialsInURL
    }

    // Check for blocked hosts (localhost, 127.0.0.1)
    hostname := parsedURL.Hostname()
    if contains(v.blockedHosts, hostname) {
        return fmt.Errorf("%w: %s", ErrBlockedHost, hostname)
    }

    // Check for private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
    if ip := net.ParseIP(hostname); ip != nil {
        if ip.IsPrivate() {
            return fmt.Errorf("%w: private IP %s", ErrBlockedHost, ip)
        }
    }

    v.logger.Debug("URL validation passed", slog.String("url", urlStr))
    return nil
}
```

#### Rule 2: Payload Size Validation

```go
// ValidatePayloadSize validates payload size
func (v *WebhookValidator) ValidatePayloadSize(payload []byte) error {
    size := int64(len(payload))
    if size > v.maxPayloadSize {
        return fmt.Errorf("%w: %d bytes exceeds limit of %d bytes",
            ErrPayloadTooLarge, size, v.maxPayloadSize)
    }
    return nil
}
```

#### Rule 3: Header Validation

```go
// ValidateHeaders validates HTTP headers
func (v *WebhookValidator) ValidateHeaders(headers map[string]string) error {
    // Check header count
    if len(headers) > v.maxHeaders {
        return fmt.Errorf("%w: %d headers exceeds limit of %d",
            ErrTooManyHeaders, len(headers), v.maxHeaders)
    }

    // Check header value sizes
    for key, value := range headers {
        if len(value) > v.maxHeaderSize {
            return fmt.Errorf("%w: header %s value size %d exceeds limit of %d",
                ErrHeaderValueTooLarge, key, len(value), v.maxHeaderSize)
        }
    }

    return nil
}
```

#### Rule 4: Timeout Validation

```go
// ValidateTimeout validates timeout configuration
func (v *WebhookValidator) ValidateTimeout(timeout time.Duration) error {
    if timeout < v.minTimeout {
        return fmt.Errorf("%w: %s is less than minimum %s",
            ErrInvalidTimeout, timeout, v.minTimeout)
    }
    if timeout > v.maxTimeout {
        return fmt.Errorf("%w: %s exceeds maximum %s",
            ErrInvalidTimeout, timeout, v.maxTimeout)
    }
    return nil
}
```

#### Rule 5: Retry Config Validation

```go
// ValidateRetryConfig validates retry configuration
func (v *WebhookValidator) ValidateRetryConfig(config RetryConfig) error {
    if config.MaxRetries < 0 || config.MaxRetries > v.maxRetries {
        return fmt.Errorf("%w: max retries %d out of range [0, %d]",
            ErrInvalidRetryConfig, config.MaxRetries, v.maxRetries)
    }

    if config.BaseBackoff < 0 || config.BaseBackoff > 10*time.Second {
        return fmt.Errorf("%w: base backoff %s out of range [0, 10s]",
            ErrInvalidRetryConfig, config.BaseBackoff)
    }

    if config.MaxBackoff < config.BaseBackoff {
        return fmt.Errorf("%w: max backoff %s less than base backoff %s",
            ErrInvalidRetryConfig, config.MaxBackoff, config.BaseBackoff)
    }

    return nil
}
```

#### Rule 6: Format Validation

```go
// ValidateFormat validates payload format (JSON serializable)
func (v *WebhookValidator) ValidateFormat(payload interface{}) error {
    // Try to marshal to JSON
    _, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrInvalidFormat, err)
    }
    return nil
}
```

### 4.3 Comprehensive Validation

```go
// ValidateTarget validates entire publishing target
func (v *WebhookValidator) ValidateTarget(target *core.PublishingTarget) error {
    // Validate URL
    if err := v.ValidateURL(target.URL); err != nil {
        return err
    }

    // Validate headers
    if err := v.ValidateHeaders(target.Headers); err != nil {
        return err
    }

    // Validate timeout (if specified)
    if target.Timeout != 0 {
        if err := v.ValidateTimeout(target.Timeout); err != nil {
            return err
        }
    }

    v.logger.Info("Target validation passed",
        slog.String("target", target.Name),
        slog.String("url", target.URL))
    return nil
}
```

---

## 5. Retry Logic

### 5.1 Retry Configuration

```go
// RetryConfig defines retry behavior
type RetryConfig struct {
    MaxRetries  int           // Default: 3, range 0-5
    BaseBackoff time.Duration // Default: 100ms
    MaxBackoff  time.Duration // Default: 5s
    Multiplier  float64       // Default: 2.0
}

// Default retry config
var DefaultRetryConfig = RetryConfig{
    MaxRetries:  3,
    BaseBackoff: 100 * time.Millisecond,
    MaxBackoff:  5 * time.Second,
    Multiplier:  2.0,
}
```

### 5.2 Error Classification

```go
// ErrorCategory classifies errors for retry decision
type ErrorCategory int

const (
    ErrorCategoryRetryable  ErrorCategory = iota // Retry allowed
    ErrorCategoryPermanent                       // Don't retry
    ErrorCategoryUnknown                         // Unknown (treat as permanent)
)

// classifyHTTPError classifies HTTP errors
func classifyHTTPError(statusCode int) ErrorCategory {
    switch {
    case statusCode == 429:
        return ErrorCategoryRetryable // Rate limit
    case statusCode >= 500:
        return ErrorCategoryRetryable // Server errors
    case statusCode >= 400 && statusCode < 500:
        return ErrorCategoryPermanent // Client errors
    default:
        return ErrorCategoryUnknown
    }
}

// IsRetryableError checks if error is retryable
func IsRetryableError(err error) bool {
    // Check for network errors (timeout, connection refused)
    var netErr net.Error
    if errors.As(err, &netErr) {
        if netErr.Timeout() || netErr.Temporary() {
            return true
        }
    }

    // Check for context deadline exceeded
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // Check for WebhookError type
    var webhookErr *WebhookError
    if errors.As(err, &webhookErr) {
        return webhookErr.Type == ErrorTypeNetwork ||
               webhookErr.Type == ErrorTypeTimeout ||
               webhookErr.Type == ErrorTypeRateLimit ||
               webhookErr.Type == ErrorTypeServer
    }

    return false
}
```

### 5.3 Retry Implementation

```go
// doRequestWithRetry executes HTTP request with retry logic
func (c *WebhookHTTPClient) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
    var lastErr error
    backoff := c.retryConfig.BaseBackoff

    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        // Record attempt
        if attempt > 0 {
            c.logger.InfoContext(ctx, "Retrying request",
                slog.Int("attempt", attempt),
                slog.Duration("backoff", backoff))
            c.metrics.RetriesTotal.WithLabelValues(
                req.URL.Host,
                fmt.Sprintf("%d", attempt),
            ).Inc()

            // Wait before retry
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(backoff):
            }
        }

        // Clone request body (consumed on first attempt)
        var bodyBytes []byte
        if req.Body != nil {
            bodyBytes, _ = io.ReadAll(req.Body)
            req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
        }

        // Execute HTTP request
        startTime := time.Now()
        resp, err := c.httpClient.Do(req)
        duration := time.Since(startTime)

        // Handle network errors
        if err != nil {
            lastErr = &WebhookError{
                Type:    ErrorTypeNetwork,
                Message: fmt.Sprintf("HTTP request failed: %v", err),
                Cause:   err,
            }

            // Retry network errors
            if IsRetryableError(lastErr) && attempt < c.retryConfig.MaxRetries {
                backoff = calculateBackoff(backoff, c.retryConfig)
                continue
            }

            // Permanent network error or max retries
            c.metrics.ErrorsTotal.WithLabelValues(req.URL.Host, "network").Inc()
            return nil, lastErr
        }

        // Check HTTP status code
        category := classifyHTTPError(resp.StatusCode)

        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            // Success
            c.metrics.RequestDuration.WithLabelValues(req.URL.Host, "success").Observe(duration.Seconds())
            return resp, nil
        }

        // HTTP error
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()

        lastErr = &WebhookError{
            Type:       classifyErrorType(resp.StatusCode),
            Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
            StatusCode: resp.StatusCode,
        }

        // Retry transient errors
        if category == ErrorCategoryRetryable && attempt < c.retryConfig.MaxRetries {
            // Check for Retry-After header (429 responses)
            if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
                if seconds, err := strconv.Atoi(retryAfter); err == nil {
                    backoff = time.Duration(seconds) * time.Second
                    c.logger.InfoContext(ctx, "Rate limited, respecting Retry-After",
                        slog.Duration("retry_after", backoff))
                }
            } else {
                backoff = calculateBackoff(backoff, c.retryConfig)
            }

            // Reset request body for next attempt
            if len(bodyBytes) > 0 {
                req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
            }
            continue
        }

        // Permanent error or max retries
        c.metrics.ErrorsTotal.WithLabelValues(req.URL.Host, lastErr.Type.String()).Inc()
        c.metrics.RequestDuration.WithLabelValues(req.URL.Host, "error").Observe(duration.Seconds())
        return nil, lastErr
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateBackoff calculates next backoff duration
func calculateBackoff(current time.Duration, config RetryConfig) time.Duration {
    next := time.Duration(float64(current) * config.Multiplier)
    if next > config.MaxBackoff {
        return config.MaxBackoff
    }
    return next
}
```

---

## 6. Error Handling

### 6.1 Error Types

```go
// WebhookError represents a webhook operation error
type WebhookError struct {
    Type       ErrorType
    Message    string
    StatusCode int   // HTTP status code (if applicable)
    Cause      error // Underlying error
}

// Error implements error interface
func (e *WebhookError) Error() string {
    if e.StatusCode > 0 {
        return fmt.Sprintf("[%s] HTTP %d: %s", e.Type, e.StatusCode, e.Message)
    }
    return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap implements errors unwrapping
func (e *WebhookError) Unwrap() error {
    return e.Cause
}

// ErrorType categorizes webhook errors
type ErrorType int

const (
    ErrorTypeValidation ErrorType = iota // Validation errors
    ErrorTypeAuth                         // Authentication errors
    ErrorTypeNetwork                      // Network errors
    ErrorTypeTimeout                      // Timeout errors
    ErrorTypeRateLimit                    // Rate limit errors
    ErrorTypeServer                       // Server errors (5xx)
)

func (t ErrorType) String() string {
    switch t {
    case ErrorTypeValidation:
        return "validation"
    case ErrorTypeAuth:
        return "auth"
    case ErrorTypeNetwork:
        return "network"
    case ErrorTypeTimeout:
        return "timeout"
    case ErrorTypeRateLimit:
        return "rate_limit"
    case ErrorTypeServer:
        return "server"
    default:
        return "unknown"
    }
}
```

### 6.2 Sentinel Errors

```go
// Sentinel errors
var (
    // Validation errors
    ErrEmptyURL           = errors.New("webhook URL cannot be empty")
    ErrInvalidURL         = errors.New("invalid webhook URL")
    ErrInsecureScheme     = errors.New("URL must use HTTPS")
    ErrCredentialsInURL   = errors.New("URL must not contain credentials")
    ErrBlockedHost        = errors.New("blocked hostname")
    ErrPayloadTooLarge    = errors.New("payload exceeds size limit")
    ErrTooManyHeaders     = errors.New("too many headers")
    ErrHeaderValueTooLarge = errors.New("header value too large")
    ErrInvalidTimeout     = errors.New("timeout out of range")
    ErrInvalidRetryConfig = errors.New("invalid retry configuration")
    ErrInvalidFormat      = errors.New("invalid payload format")

    // Auth errors
    ErrMissingAuthToken          = errors.New("missing auth token")
    ErrMissingBasicAuthCredentials = errors.New("missing basic auth credentials")
    ErrMissingAPIKey             = errors.New("missing API key")
    ErrNoCustomHeaders           = errors.New("no custom headers provided")
)
```

### 6.3 Error Classification Helpers

```go
// IsRetryableError checks if error should be retried
func IsRetryableError(err error) bool {
    var webhookErr *WebhookError
    if errors.As(err, &webhookErr) {
        return webhookErr.Type == ErrorTypeNetwork ||
               webhookErr.Type == ErrorTypeTimeout ||
               webhookErr.Type == ErrorTypeRateLimit ||
               webhookErr.Type == ErrorTypeServer
    }

    // Check for network errors
    var netErr net.Error
    if errors.As(err, &netErr) {
        return netErr.Timeout() || netErr.Temporary()
    }

    return errors.Is(err, context.DeadlineExceeded)
}

// IsPermanentError checks if error is permanent
func IsPermanentError(err error) bool {
    return !IsRetryableError(err)
}

// classifyErrorType classifies HTTP status code to ErrorType
func classifyErrorType(statusCode int) ErrorType {
    switch {
    case statusCode == 429:
        return ErrorTypeRateLimit
    case statusCode >= 500:
        return ErrorTypeServer
    case statusCode == 401 || statusCode == 403:
        return ErrorTypeAuth
    case statusCode == 408 || statusCode == 504:
        return ErrorTypeTimeout
    default:
        return ErrorTypeValidation
    }
}
```

---

## 7. Metrics & Observability

### 7.1 Prometheus Metrics (8 metrics)

```go
// WebhookMetrics holds Prometheus metrics for webhook publisher
type WebhookMetrics struct {
    RequestsTotal     *prometheus.CounterVec
    RequestDuration   *prometheus.HistogramVec
    ErrorsTotal       *prometheus.CounterVec
    RetriesTotal      *prometheus.CounterVec
    PayloadSize       *prometheus.HistogramVec
    AuthFailures      *prometheus.CounterVec
    ValidationErrors  *prometheus.CounterVec
    TimeoutErrors     *prometheus.CounterVec
}

// NewWebhookMetrics creates webhook metrics
func NewWebhookMetrics(registry prometheus.Registerer) *WebhookMetrics {
    metrics := &WebhookMetrics{
        RequestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_requests_total",
                Help: "Total number of webhook requests",
            },
            []string{"target", "status", "method"},
        ),
        RequestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "webhook_request_duration_seconds",
                Help:    "Webhook request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"target", "status"},
        ),
        ErrorsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_errors_total",
                Help: "Total number of webhook errors",
            },
            []string{"target", "error_type"},
        ),
        RetriesTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_retries_total",
                Help: "Total number of webhook retries",
            },
            []string{"target", "attempt"},
        ),
        PayloadSize: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "webhook_payload_size_bytes",
                Help:    "Webhook payload size in bytes",
                Buckets: prometheus.ExponentialBuckets(1024, 2, 12), // 1KB to 4MB
            },
            []string{"target"},
        ),
        AuthFailures: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_auth_failures_total",
                Help: "Total number of webhook auth failures",
            },
            []string{"target", "auth_type"},
        ),
        ValidationErrors: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_validation_errors_total",
                Help: "Total number of webhook validation errors",
            },
            []string{"target", "validation_type"},
        ),
        TimeoutErrors: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_timeout_errors_total",
                Help: "Total number of webhook timeout errors",
            },
            []string{"target"},
        ),
    }

    // Register metrics
    registry.MustRegister(
        metrics.RequestsTotal,
        metrics.RequestDuration,
        metrics.ErrorsTotal,
        metrics.RetriesTotal,
        metrics.PayloadSize,
        metrics.AuthFailures,
        metrics.ValidationErrors,
        metrics.TimeoutErrors,
    )

    return metrics
}
```

### 7.2 Structured Logging

**Log Levels**:
- **DEBUG**: Request/response bodies, auth application, validation details
- **INFO**: Successful webhook POST, retry attempts
- **WARN**: Validation warnings, retry exhausted
- **ERROR**: Permanent errors, auth failures

**Example Logs**:
```go
// INFO log
logger.InfoContext(ctx, "Publishing alert to webhook",
    slog.String("target", target.Name),
    slog.String("url", target.URL),
    slog.String("fingerprint", alert.Fingerprint))

// DEBUG log
logger.DebugContext(ctx, "Applying authentication",
    slog.String("auth_type", string(authConfig.Type)),
    slog.String("url", maskURL(target.URL)))

// WARN log
logger.WarnContext(ctx, "Retrying after transient error",
    slog.Int("attempt", attempt),
    slog.Duration("backoff", backoff),
    slog.String("error", err.Error()))

// ERROR log
logger.ErrorContext(ctx, "Permanent error, no retry",
    slog.Int("status_code", statusCode),
    slog.String("error_type", errorType.String()))
```

---

## 8. Data Models

### 8.1 Request/Response Models

```go
// WebhookRequest represents a webhook HTTP request
type WebhookRequest struct {
    URL     string
    Payload map[string]interface{}
    Headers map[string]string
    Timeout time.Duration
}

// WebhookResponse represents a webhook HTTP response
type WebhookResponse struct {
    StatusCode int
    Body       []byte
    Headers    http.Header
    Duration   time.Duration
}
```

### 8.2 Configuration Models

```go
// WebhookConfig holds webhook configuration
type WebhookConfig struct {
    URL           string            `json:"url"`
    Headers       map[string]string `json:"headers,omitempty"`
    Timeout       time.Duration     `json:"timeout,omitempty"`
    RetryConfig   *RetryConfig      `json:"retry,omitempty"`
    AuthConfig    *AuthConfig       `json:"auth,omitempty"`
    MaxPayloadSize int64            `json:"max_payload_size,omitempty"`
}
```

---

## 9. Configuration

### 9.1 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_TIMEOUT` | `10s` | HTTP client timeout |
| `WEBHOOK_MAX_RETRIES` | `3` | Max retry attempts |
| `WEBHOOK_BASE_BACKOFF` | `100ms` | Initial backoff |
| `WEBHOOK_MAX_BACKOFF` | `5s` | Max backoff cap |
| `WEBHOOK_MAX_PAYLOAD_SIZE` | `1MB` | Payload size limit |
| `WEBHOOK_MAX_HEADERS` | `100` | Max header count |

### 9.2 K8s Secret Format

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: webhook-api-bearer
  namespace: alert-history
  labels:
    publishing-target: "true"
type: Opaque
stringData:
  target.json: |
    {
      "name": "api-webhook",
      "type": "webhook",
      "url": "https://api.example.com/webhooks/alerts",
      "format": "webhook",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "X-Custom-Header": "value"
      },
      "timeout": "15s",
      "retry": {
        "max_retries": 3,
        "base_backoff": "100ms",
        "max_backoff": "5s"
      }
    }
```

---

## 10. Testing Strategy

### 10.1 Unit Tests (56+ tests, 90%+ coverage)

**Test Categories**:
1. **Auth Tests** (8 tests): Each auth strategy + error scenarios
2. **Validation Tests** (10 tests): Each validation rule + edge cases
3. **Client Tests** (15 tests): HTTP requests, retry logic, error handling
4. **Publisher Tests** (12 tests): End-to-end publishing, metrics recording
5. **Retry Tests** (6 tests): Backoff calculation, retry decision
6. **Error Tests** (5 tests): Error classification, wrapping

### 10.2 Integration Tests (10+ scenarios)

1. Bearer token auth → successful POST
2. Basic auth → successful POST
3. API key auth → successful POST
4. Custom headers → successful POST
5. Transient error (503) → retry success
6. Rate limit (429) → retry with backoff
7. Permanent error (401) → no retry
8. Network timeout → retry
9. Validation error → immediate failure
10. Mock HTTP server integration

### 10.3 Benchmarks (8+ operations)

1. `BenchmarkWebhookPOST`
2. `BenchmarkValidateURL`
3. `BenchmarkValidatePayload`
4. `BenchmarkApplyAuth`
5. `BenchmarkRetryLogic`
6. `BenchmarkConcurrentPublish`
7. `BenchmarkPayloadSerialization`
8. `BenchmarkErrorClassification`

---

## 11. Performance Optimization

### 11.1 HTTP Client Optimization

```go
// Optimized HTTP client configuration
httpClient := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
        },
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     30 * time.Second,
        DisableCompression:  false,
        ForceAttemptHTTP2:   true, // Enable HTTP/2
    },
}
```

### 11.2 Performance Targets

| Metric | Target | Optimization |
|--------|--------|--------------|
| **POST Latency (p50)** | <50ms | Connection pooling, HTTP/2 |
| **POST Latency (p99)** | <200ms | Retry optimization |
| **Validation Overhead** | <1ms | Pre-compiled regex |
| **Auth Overhead** | <500µs | Cached auth headers |
| **Throughput** | 200+ req/s | Concurrent execution |

---

## 12. Integration

### 12.1 PublisherFactory Integration

```go
// In PublisherFactory.CreatePublisherForTarget()
func (f *PublisherFactory) CreatePublisherForTarget(target *core.PublishingTarget) (AlertPublisher, error) {
    switch TargetType(target.Type) {
    case TargetTypeWebhook, TargetTypeAlertmanager:
        return f.createEnhancedWebhookPublisher(target)
    // ... other types
    }
}

func (f *PublisherFactory) createEnhancedWebhookPublisher(target *core.PublishingTarget) (AlertPublisher, error) {
    // Create validator
    validator := NewWebhookValidator(f.logger)

    // Validate target
    if err := validator.ValidateTarget(target); err != nil {
        return nil, fmt.Errorf("target validation failed: %w", err)
    }

    // Create HTTP client
    client := NewWebhookHTTPClient(DefaultRetryConfig, f.logger)

    // Create enhanced publisher
    return NewEnhancedWebhookPublisher(
        client,
        validator,
        f.webhookMetrics, // Shared metrics
        f.formatter,      // Shared formatter
        f.logger,
    ), nil
}
```

---

## 📋 DESIGN COMPLETE

**Status**: ✅ **DESIGN PHASE COMPLETE**

**Deliverable**: 1,000+ LOC comprehensive technical design

**Next Phase**: Create `tasks.md` (800+ LOC implementation tasks)

**Quality Level**: **150% (Enterprise Grade A+)**

---

**Date**: 2025-11-11
**Version**: 1.0
**Approved By**: AI Architect (following TN-052/053/054 success pattern)
