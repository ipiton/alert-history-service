# TN-051: Alert Formatter - Technical Design (Enterprise Quality, 150%)

**Version**: 1.0
**Date**: 2025-11-08
**Status**: 🎯 **ENHANCEMENT DESIGN** (Baseline → 150% Quality)
**Target Quality**: **Enterprise Grade A+**

---

## 📑 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Strategy Pattern Implementation](#3-strategy-pattern-implementation)
4. [Format Registry Architecture](#4-format-registry-architecture)
5. [Middleware Pipeline Design](#5-middleware-pipeline-design)
6. [Caching Strategy](#6-caching-strategy)
7. [Validation Framework](#7-validation-framework)
8. [Monitoring Integration](#8-monitoring-integration)
9. [Performance Optimization](#9-performance-optimization)
10. [Error Handling Strategy](#10-error-handling-strategy)
11. [Data Flow Diagrams](#11-data-flow-diagrams)
12. [API Contracts](#12-api-contracts)
13. [Testing Strategy](#13-testing-strategy)
14. [Security Considerations](#14-security-considerations)
15. [Migration Path](#15-migration-path)

---

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     Publishing System (Phase 5)                         │
│                                                                          │
│  ┌────────────────┐      ┌─────────────────┐      ┌─────────────────┐ │
│  │   K8s Client   │─────▶│ Target Discovery│─────▶│ Refresh Manager │ │
│  │   (TN-046)     │      │    (TN-047)     │      │    (TN-048)     │ │
│  └────────────────┘      └─────────────────┘      └─────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                    ALERT FORMATTER (TN-051)                        ││
│  │  ┌──────────────────────────────────────────────────────────────┐ ││
│  │  │              Middleware Pipeline (150%)                      │ ││
│  │  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐          │ ││
│  │  │  │Valid.│─▶│Cache │─▶│Trace │─▶│Metric│─▶│Rate  │          │ ││
│  │  │  │      │  │      │  │      │  │      │  │Limit │          │ ││
│  │  │  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘          │ ││
│  │  └──────────────────────────────────────────────────────────────┘ ││
│  │                                                                    ││
│  │  ┌──────────────────────────────────────────────────────────────┐ ││
│  │  │              Format Registry (150%)                          │ ││
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │ ││
│  │  │  │ Alertmanager│  │   Rootly    │  │  PagerDuty  │          │ ││
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘          │ ││
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │ ││
│  │  │  │    Slack    │  │   Webhook   │  │ [Custom...] │          │ ││
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘          │ ││
│  │  └──────────────────────────────────────────────────────────────┘ ││
│  │                                                                    ││
│  │  ┌──────────────────────────────────────────────────────────────┐ ││
│  │  │              Strategy Pattern (Baseline ✅)                  │ ││
│  │  │  formatAlertmanager() | formatRootly() | formatPagerDuty()  │ ││
│  │  │  formatSlack() | formatWebhook()                            │ ││
│  │  └──────────────────────────────────────────────────────────────┘ ││
│  └────────────────────────────────────────────────────────────────────┘│
│                                                                          │
│  ┌────────────────┐      ┌─────────────────┐      ┌─────────────────┐ │
│  │   Publishers   │◀─────│ Publishing Queue│◀─────│Parallel Publish │ │
│  │  (TN-052-055)  │      │    (TN-056)     │      │Coordinator      │ │
│  └────────────────┘      └─────────────────┘      │   (TN-058)      │ │
│                                                    └─────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘

                 ┌─────────────────────────────────┐
                 │   Monitoring (150% Enhancement) │
                 ├─────────────────────────────────┤
                 │  • Prometheus Metrics (6+)      │
                 │  • OpenTelemetry Tracing        │
                 │  • Structured Logging           │
                 │  • Health Checks                │
                 └─────────────────────────────────┘
```

### 1.2 Layered Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                       Layer 4: API Interface                      │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ AlertFormatter interface (public API)                       │ │
│  │   FormatAlert(ctx, enrichedAlert, format) -> (map, error)  │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│                    Layer 3: Middleware Pipeline                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ ValidationMiddleware → CachingMiddleware →                  │ │
│  │ TracingMiddleware → MetricsMiddleware → RateLimitMiddleware │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│                    Layer 2: Format Registry                       │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ FormatRegistry: Register/Unregister/List/Supports          │ │
│  │ Thread-safe access (RWMutex), versioning, validation       │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│                  Layer 1: Format Implementations                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ formatAlertmanager() | formatRootly() | formatPagerDuty()  │ │
│  │ formatSlack() | formatWebhook() | [custom formats...]      │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│                   Layer 0: Core Data Models                       │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ Alert, EnrichedAlert, ClassificationResult                 │ │
│  │ PublishingFormat enum, PublishingTarget                    │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

### 1.3 Component Interaction

```
   Client (Publishers, Queue)
           │
           ▼
   ┌──────────────────┐
   │ AlertFormatter   │ (public interface)
   │   .FormatAlert() │
   └──────────────────┘
           │
           ▼
   ┌──────────────────────────────────────┐
   │ Middleware Pipeline                  │
   │                                      │
   │  1. ValidationMiddleware             │
   │     - Validate enriched alert        │
   │     - Check format support           │
   │                                      │
   │  2. CachingMiddleware                │
   │     - Check cache (fingerprint+fmt)  │
   │     - Return cached if hit           │
   │                                      │
   │  3. TracingMiddleware                │
   │     - Create OTel span               │
   │     - Add attributes                 │
   │                                      │
   │  4. MetricsMiddleware                │
   │     - Record latency                 │
   │     - Increment counters             │
   │                                      │
   │  5. RateLimitMiddleware              │
   │     - Check rate limit               │
   │     - Return error if exceeded       │
   └──────────────────────────────────────┘
           │
           ▼
   ┌──────────────────┐
   │ Format Registry  │
   │  .Get(format)    │
   └──────────────────┘
           │
           ▼
   ┌──────────────────────────────────────┐
   │ Format-Specific Implementations      │
   │                                      │
   │  • formatAlertmanager()              │
   │  • formatRootly()                    │
   │  • formatPagerDuty()                 │
   │  • formatSlack()                     │
   │  • formatWebhook()                   │
   └──────────────────────────────────────┘
           │
           ▼
   ┌──────────────────┐
   │ Formatted Output │ (map[string]any)
   └──────────────────┘
           │
           ▼
   Client (Publisher sends to vendor)
```

---

## 2. Component Design

### 2.1 AlertFormatter Interface (Baseline - ✅ Exists)

```go
package publishing

import (
    "context"
    "github.com/vitaliisemenov/alert-history/internal/core"
)

// AlertFormatter defines the interface for formatting alerts for different publishing targets.
// This is the public API that clients (publishers, queue) interact with.
//
// Thread-safety: Implementations MUST be safe for concurrent use by multiple goroutines.
// Performance: Implementations SHOULD complete formatting in <500μs (p50).
//
// Example:
//   formatter := NewAlertFormatter()
//   result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatRootly)
//   if err != nil {
//       return fmt.Errorf("formatting failed: %w", err)
//   }
//   // result is map[string]any ready for JSON marshaling
type AlertFormatter interface {
    // FormatAlert formats an enriched alert for a specific target format.
    //
    // Parameters:
    //   ctx: Context for cancellation and deadlines (e.g., 5s timeout)
    //   enrichedAlert: Alert with LLM classification data (required, non-nil)
    //   format: Target format enum (Alertmanager, Rootly, PagerDuty, Slack, Webhook)
    //
    // Returns:
    //   map[string]any: Formatted alert ready for JSON marshaling
    //   error: nil on success, error on validation failure or formatting error
    //
    // Errors:
    //   - ValidationError: Invalid input (nil alert, empty fingerprint, etc.)
    //   - FormatError: Format-specific error (e.g., truncation failed)
    //   - context.DeadlineExceeded: Formatting took too long
    //
    // Thread-safety: Safe for concurrent calls.
    // Performance: Target <500μs (p50), <1ms (p99).
    FormatAlert(ctx context.Context, enrichedAlert *core.EnrichedAlert, format core.PublishingFormat) (map[string]any, error)
}
```

### 2.2 DefaultAlertFormatter (Baseline - ✅ Exists)

```go
// DefaultAlertFormatter implements AlertFormatter using strategy pattern.
// It delegates formatting to format-specific functions registered in a map.
//
// Thread-safety: Safe for concurrent use (read-only map after initialization).
// Performance: <500μs per call (target).
//
// Example:
//   formatter := NewAlertFormatter()
//   // formatter is ready to use, no additional setup needed
type DefaultAlertFormatter struct {
    // formatters maps PublishingFormat to format-specific implementation.
    // Initialized in NewAlertFormatter() and read-only thereafter.
    // Thread-safe: no writes after initialization.
    formatters map[core.PublishingFormat]formatFunc
}

// formatFunc is the function signature for format-specific implementations.
// Each function takes an enriched alert and returns formatted output.
//
// Parameters:
//   enrichedAlert: Alert with LLM classification (guaranteed non-nil by middleware)
//
// Returns:
//   map[string]any: Formatted alert (e.g., Alertmanager webhook structure)
//   error: nil on success, FormatError on failure
//
// Thread-safety: Must be stateless (no shared mutable state).
// Performance: Target <500μs execution time.
type formatFunc func(*core.EnrichedAlert) (map[string]any, error)

// NewAlertFormatter creates a new alert formatter with all built-in formats.
// This is the standard constructor for production use.
//
// Returns:
//   AlertFormatter: Ready-to-use formatter instance
//
// Thread-safety: Returned formatter is safe for concurrent use.
// Performance: Constructor is fast (<1μs), no I/O.
//
// Example:
//   formatter := NewAlertFormatter()
//   defer formatter.Close() // if we add cleanup in 150% version
func NewAlertFormatter() AlertFormatter {
    formatter := &DefaultAlertFormatter{
        formatters: make(map[core.PublishingFormat]formatFunc, 5),
    }

    // Register built-in format strategies
    formatter.formatters[core.FormatAlertmanager] = formatter.formatAlertmanager
    formatter.formatters[core.FormatRootly] = formatter.formatRootly
    formatter.formatters[core.FormatPagerDuty] = formatter.formatPagerDuty
    formatter.formatters[core.FormatSlack] = formatter.formatSlack
    formatter.formatters[core.FormatWebhook] = formatter.formatWebhook

    return formatter
}
```

### 2.3 EnhancedAlertFormatter (150% - 🎯 NEW)

```go
// EnhancedAlertFormatter implements AlertFormatter with advanced features:
// - Dynamic format registry
// - Middleware pipeline
// - Caching layer
// - Comprehensive monitoring
//
// Thread-safety: Safe for concurrent use (internal synchronization).
// Performance: <500μs per call (with cache hits <10μs).
//
// Architecture:
//   Client → Middleware Pipeline → Format Registry → Format Implementation → Output
//
// Example:
//   formatter := NewEnhancedAlertFormatter(
//       WithRegistry(customRegistry),
//       WithMiddleware(ValidationMiddleware, CachingMiddleware),
//       WithMetrics(prometheusRegistry),
//       WithTracing(otelprovider),
//   )
//   defer formatter.Close() // cleanup resources
type EnhancedAlertFormatter struct {
    // registry manages format-specific implementations
    registry FormatRegistry

    // middleware wraps format functions with preprocessing/postprocessing
    middleware *MiddlewareChain

    // cache stores recently formatted alerts (LRU, 1000 entries, 5min TTL)
    cache Cache

    // metrics records Prometheus metrics (latency, counts, cache hits)
    metrics *FormatterMetrics

    // tracer creates OpenTelemetry spans for distributed tracing
    tracer trace.Tracer

    // logger structured logger (slog) for audit trail
    logger *slog.Logger

    // mu protects concurrent access to internal state (if any)
    mu sync.RWMutex
}

// NewEnhancedAlertFormatter creates an enterprise-grade formatter with all features.
//
// Options:
//   - WithRegistry: Custom format registry (default: built-in 5 formats)
//   - WithMiddleware: Middleware chain (default: validation + caching + metrics)
//   - WithCache: Custom cache implementation (default: LRU 1000 entries, 5min TTL)
//   - WithMetrics: Prometheus metrics registry (default: DefaultRegisterer)
//   - WithTracing: OpenTelemetry tracer provider (default: global provider)
//   - WithLogger: Structured logger (default: slog.Default())
//
// Returns:
//   AlertFormatter: Fully configured formatter instance
//
// Example:
//   formatter := NewEnhancedAlertFormatter(
//       WithCache(NewLRUCache(2000, 10*time.Minute)),
//       WithMetrics(prometheus.DefaultRegisterer),
//   )
func NewEnhancedAlertFormatter(opts ...FormatterOption) (AlertFormatter, error) {
    f := &EnhancedAlertFormatter{
        registry:   NewDefaultFormatRegistry(),
        middleware: NewMiddlewareChain(),
        cache:      NewLRUCache(1000, 5*time.Minute),
        logger:     slog.Default(),
    }

    // Apply options
    for _, opt := range opts {
        if err := opt(f); err != nil {
            return nil, fmt.Errorf("failed to apply option: %w", err)
        }
    }

    // Initialize metrics (if not provided)
    if f.metrics == nil {
        f.metrics = NewFormatterMetrics(prometheus.DefaultRegisterer)
    }

    // Initialize tracer (if not provided)
    if f.tracer == nil {
        f.tracer = otel.Tracer("alert-history/formatter")
    }

    // Build middleware chain
    f.buildMiddlewareChain()

    return f, nil
}

// buildMiddlewareChain constructs the middleware pipeline in order:
// 1. Validation (check input validity)
// 2. Caching (check cache, return if hit)
// 3. Tracing (create OTel span)
// 4. Metrics (record latency)
// 5. RateLimiting (prevent abuse)
func (f *EnhancedAlertFormatter) buildMiddlewareChain() {
    f.middleware.Use(
        NewValidationMiddleware(),
        NewCachingMiddleware(f.cache),
        NewTracingMiddleware(f.tracer),
        NewMetricsMiddleware(f.metrics),
        NewRateLimitMiddleware(1000), // max 1000 req/sec
    )
}
```

---

## 3. Strategy Pattern Implementation

### 3.1 Pattern Overview

**Strategy Pattern** allows selecting formatting algorithm at runtime without modifying client code.

**Benefits**:
- ✅ **Extensibility**: Easy to add new formats (register new strategy)
- ✅ **Testability**: Mock individual formatters
- ✅ **Maintainability**: Each format is isolated, no if-else chains
- ✅ **Performance**: Direct map lookup (O(1))

### 3.2 Current Implementation (Baseline)

```go
// FormatAlert selects and invokes the appropriate format strategy
func (f *DefaultAlertFormatter) FormatAlert(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    format core.PublishingFormat,
) (map[string]any, error) {
    // Validate input
    if enrichedAlert == nil || enrichedAlert.Alert == nil {
        return nil, fmt.Errorf("enriched alert or alert is nil")
    }

    // Lookup format strategy
    formatFn, exists := f.formatters[format]
    if !exists {
        // Fallback to webhook format for unknown formats
        formatFn = f.formatWebhook
    }

    // Invoke strategy
    return formatFn(enrichedAlert)
}
```

### 3.3 Enhanced Implementation (150%)

```go
// FormatAlert with middleware pipeline and error handling
func (f *EnhancedAlertFormatter) FormatAlert(
    ctx context.Context,
    enrichedAlert *core.EnrichedAlert,
    format core.PublishingFormat,
) (map[string]any, error) {
    // Create tracing span
    ctx, span := f.tracer.Start(ctx, "formatter.FormatAlert",
        trace.WithAttributes(
            attribute.String("format", string(format)),
            attribute.String("fingerprint", enrichedAlert.Alert.Fingerprint),
        ),
    )
    defer span.End()

    // Validate input (middleware will also validate, but early check)
    if enrichedAlert == nil || enrichedAlert.Alert == nil {
        span.RecordError(fmt.Errorf("nil alert"))
        return nil, &ValidationError{Field: "enrichedAlert", Message: "cannot be nil"}
    }

    // Lookup format strategy from registry
    formatFn, err := f.registry.Get(format)
    if err != nil {
        // Try fallback to webhook
        f.logger.WarnContext(ctx, "format not found, using webhook fallback",
            slog.String("format", string(format)),
            slog.String("error", err.Error()),
        )
        formatFn, _ = f.registry.Get(core.FormatWebhook)
    }

    // Apply middleware chain
    wrappedFn := f.middleware.Apply(formatFn)

    // Invoke with context (for timeout)
    result, err := f.invokeWithContext(ctx, wrappedFn, enrichedAlert)
    if err != nil {
        span.RecordError(err)
        f.metrics.formatErrors.WithLabelValues(string(format)).Inc()
        return nil, fmt.Errorf("formatting failed for %s: %w", format, err)
    }

    // Record success
    span.SetStatus(codes.Ok, "formatted successfully")
    return result, nil
}

// invokeWithContext wraps format function with context timeout
func (f *EnhancedAlertFormatter) invokeWithContext(
    ctx context.Context,
    fn formatFunc,
    alert *core.EnrichedAlert,
) (map[string]any, error) {
    // Create result channel
    type result struct {
        data map[string]any
        err  error
    }
    resultChan := make(chan result, 1)

    // Run formatting in goroutine (to support cancellation)
    go func() {
        data, err := fn(alert)
        resultChan <- result{data: data, err: err}
    }()

    // Wait for result or context cancellation
    select {
    case res := <-resultChan:
        return res.data, res.err
    case <-ctx.Done():
        return nil, ctx.Err() // context.DeadlineExceeded or Canceled
    }
}
```

---

## 4. Format Registry Architecture

### 4.1 Interface Design

```go
// FormatRegistry manages dynamic format registration.
// Supports adding custom formats at runtime without code changes.
//
// Thread-safety: All methods are safe for concurrent use.
// Performance: Register/Unregister use write lock, Get uses read lock.
//
// Example:
//   registry := NewDefaultFormatRegistry()
//   registry.Register(core.PublishingFormat("opsgenie"), formatOpsgenie)
//   fn, _ := registry.Get(core.PublishingFormat("opsgenie"))
type FormatRegistry interface {
    // Register adds a new format or replaces existing one.
    //
    // Parameters:
    //   format: Unique format identifier (e.g., "opsgenie")
    //   fn: Format implementation function
    //
    // Returns:
    //   error: nil on success, RegistrationError if validation fails
    //
    // Validation:
    //   - format must not be empty
    //   - fn must not be nil
    //   - format name must match pattern: ^[a-z][a-z0-9_-]*$
    //
    // Thread-safety: Uses write lock (blocks other Register/Unregister).
    Register(format core.PublishingFormat, fn formatFunc) error

    // Unregister removes a format from the registry.
    //
    // Parameters:
    //   format: Format to remove
    //
    // Returns:
    //   error: nil on success, NotFoundError if format doesn't exist
    //
    // Safety: Cannot unregister while format is in use (reference counting).
    Unregister(format core.PublishingFormat) error

    // Get retrieves a format implementation.
    //
    // Parameters:
    //   format: Format to retrieve
    //
    // Returns:
    //   formatFunc: Format implementation
    //   error: nil on success, NotFoundError if not registered
    //
    // Thread-safety: Uses read lock (allows concurrent Gets).
    // Performance: O(1) map lookup, <10ns.
    Get(format core.PublishingFormat) (formatFunc, error)

    // Supports checks if a format is registered.
    //
    // Parameters:
    //   format: Format to check
    //
    // Returns:
    //   bool: true if registered, false otherwise
    //
    // Thread-safety: Uses read lock.
    Supports(format core.PublishingFormat) bool

    // List returns all registered formats.
    //
    // Returns:
    //   []PublishingFormat: Slice of format identifiers (sorted)
    //
    // Thread-safety: Uses read lock, returns copy (not live view).
    List() []core.PublishingFormat

    // Count returns the number of registered formats.
    //
    // Returns:
    //   int: Count of formats
    Count() int
}
```

### 4.2 Implementation

```go
// DefaultFormatRegistry implements FormatRegistry with thread-safe operations.
type DefaultFormatRegistry struct {
    // formats maps format identifier to implementation
    formats map[core.PublishingFormat]formatFunc

    // refCounts tracks active usage (for safe unregistration)
    refCounts map[core.PublishingFormat]*atomic.Int64

    // mu protects formats and refCounts maps
    mu sync.RWMutex

    // logger for audit trail
    logger *slog.Logger
}

// NewDefaultFormatRegistry creates a registry with built-in formats.
func NewDefaultFormatRegistry() FormatRegistry {
    r := &DefaultFormatRegistry{
        formats:   make(map[core.PublishingFormat]formatFunc, 10),
        refCounts: make(map[core.PublishingFormat]*atomic.Int64, 10),
        logger:    slog.Default(),
    }

    // Register built-in formats
    r.registerBuiltins()

    return r
}

// registerBuiltins adds the 5 standard formats
func (r *DefaultFormatRegistry) registerBuiltins() {
    // Create formatter instance to access methods
    baseFormatter := &DefaultAlertFormatter{}

    r.formats[core.FormatAlertmanager] = baseFormatter.formatAlertmanager
    r.formats[core.FormatRootly] = baseFormatter.formatRootly
    r.formats[core.FormatPagerDuty] = baseFormatter.formatPagerDuty
    r.formats[core.FormatSlack] = baseFormatter.formatSlack
    r.formats[core.FormatWebhook] = baseFormatter.formatWebhook

    // Initialize reference counts
    for format := range r.formats {
        r.refCounts[format] = &atomic.Int64{}
    }
}

// Register implements FormatRegistry.Register
func (r *DefaultFormatRegistry) Register(format core.PublishingFormat, fn formatFunc) error {
    // Validate inputs
    if format == "" {
        return &RegistrationError{Format: format, Message: "format cannot be empty"}
    }
    if fn == nil {
        return &RegistrationError{Format: format, Message: "format function cannot be nil"}
    }

    // Validate format name pattern
    if !isValidFormatName(string(format)) {
        return &RegistrationError{
            Format:  format,
            Message: "format name must match ^[a-z][a-z0-9_-]*$",
        }
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    // Check if already exists (log warning)
    if _, exists := r.formats[format]; exists {
        r.logger.Warn("overwriting existing format",
            slog.String("format", string(format)),
        )
    }

    // Register format
    r.formats[format] = fn
    if r.refCounts[format] == nil {
        r.refCounts[format] = &atomic.Int64{}
    }

    r.logger.Info("format registered",
        slog.String("format", string(format)),
        slog.Int("total_formats", len(r.formats)),
    )

    return nil
}

// Get implements FormatRegistry.Get with reference counting
func (r *DefaultFormatRegistry) Get(format core.PublishingFormat) (formatFunc, error) {
    r.mu.RLock()
    fn, exists := r.formats[format]
    refCount := r.refCounts[format]
    r.mu.RUnlock()

    if !exists {
        return nil, &NotFoundError{Format: format}
    }

    // Increment reference count (for safe unregistration)
    refCount.Add(1)

    // Return wrapped function that decrements on completion
    return func(alert *core.EnrichedAlert) (map[string]any, error) {
        defer refCount.Add(-1) // decrement when done
        return fn(alert)
    }, nil
}

// isValidFormatName validates format name pattern
func isValidFormatName(name string) bool {
    if len(name) == 0 {
        return false
    }
    // Must start with lowercase letter
    if name[0] < 'a' || name[0] > 'z' {
        return false
    }
    // Rest can be lowercase, digits, hyphen, underscore
    for i := 1; i < len(name); i++ {
        c := name[i]
        if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
            return false
        }
    }
    return true
}
```

---

## 5. Middleware Pipeline Design

### 5.1 Middleware Interface

```go
// FormatterMiddleware wraps a format function with preprocessing/postprocessing.
// Middleware can:
// - Validate input before formatting
// - Cache results to avoid redundant formatting
// - Add distributed tracing spans
// - Record metrics (latency, counts)
// - Implement rate limiting
//
// Pattern: Decorator/Chain of Responsibility
//
// Example:
//   mw := func(next formatFunc) formatFunc {
//       return func(alert *core.EnrichedAlert) (map[string]any, error) {
//           // Preprocessing
//           fmt.Println("Before formatting")
//
//           // Invoke next in chain
//           result, err := next(alert)
//
//           // Postprocessing
//           fmt.Println("After formatting")
//           return result, err
//       }
//   }
type FormatterMiddleware func(next formatFunc) formatFunc
```

### 5.2 Middleware Chain

```go
// MiddlewareChain manages an ordered list of middleware.
// Middleware are applied in registration order (FIFO).
//
// Example:
//   chain := NewMiddlewareChain()
//   chain.Use(ValidationMiddleware, CachingMiddleware, MetricsMiddleware)
//   wrapped := chain.Apply(baseFormatFunc)
type MiddlewareChain struct {
    middlewares []FormatterMiddleware
    mu          sync.RWMutex
}

// NewMiddlewareChain creates an empty middleware chain
func NewMiddlewareChain() *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: make([]FormatterMiddleware, 0, 5),
    }
}

// Use adds middleware to the chain (appends to end)
func (c *MiddlewareChain) Use(middlewares ...FormatterMiddleware) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.middlewares = append(c.middlewares, middlewares...)
}

// Apply wraps a format function with all middleware in the chain.
// Middleware are applied in order: first registered = outermost wrapper.
//
// Example:
//   chain.Use(A, B, C)
//   wrapped := chain.Apply(fn)
//   // Execution order: A → B → C → fn → C → B → A
func (c *MiddlewareChain) Apply(fn formatFunc) formatFunc {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Apply middleware in reverse order (so first registered is outermost)
    wrapped := fn
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        wrapped = c.middlewares[i](wrapped)
    }
    return wrapped
}
```

### 5.3 Built-in Middleware

#### 5.3.1 Validation Middleware

```go
// NewValidationMiddleware validates enriched alert before formatting.
// Checks:
// - Alert is not nil
// - Fingerprint is not empty (max 64 chars)
// - AlertName is not empty (max 255 chars)
// - Status is valid enum
// - StartsAt is not zero time
// - Classification (if present): valid severity, confidence 0-1
func NewValidationMiddleware() FormatterMiddleware {
    return func(next formatFunc) formatFunc {
        return func(alert *core.EnrichedAlert) (map[string]any, error) {
            // Validate alert structure
            if alert == nil || alert.Alert == nil {
                return nil, &ValidationError{Field: "alert", Message: "cannot be nil"}
            }

            a := alert.Alert

            // Validate fingerprint
            if a.Fingerprint == "" {
                return nil, &ValidationError{Field: "fingerprint", Message: "cannot be empty"}
            }
            if len(a.Fingerprint) > 64 {
                return nil, &ValidationError{Field: "fingerprint", Message: "max 64 characters"}
            }

            // Validate alert name
            if a.AlertName == "" {
                return nil, &ValidationError{Field: "alert_name", Message: "cannot be empty"}
            }
            if len(a.AlertName) > 255 {
                return nil, &ValidationError{Field: "alert_name", Message: "max 255 characters"}
            }

            // Validate status
            if a.Status != core.StatusFiring && a.Status != core.StatusResolved {
                return nil, &ValidationError{Field: "status", Message: "must be 'firing' or 'resolved'"}
            }

            // Validate starts_at
            if a.StartsAt.IsZero() {
                return nil, &ValidationError{Field: "starts_at", Message: "cannot be zero time"}
            }

            // Validate classification (if present)
            if alert.Classification != nil {
                c := alert.Classification

                // Validate severity
                validSeverities := map[core.AlertSeverity]bool{
                    core.SeverityCritical: true,
                    core.SeverityWarning:  true,
                    core.SeverityInfo:     true,
                    core.SeverityNoise:    true,
                }
                if !validSeverities[c.Severity] {
                    return nil, &ValidationError{Field: "classification.severity", Message: "invalid severity"}
                }

                // Validate confidence
                if c.Confidence < 0.0 || c.Confidence > 1.0 {
                    return nil, &ValidationError{Field: "classification.confidence", Message: "must be 0.0 to 1.0"}
                }

                // Validate reasoning length
                if len(c.Reasoning) > 1000 {
                    return nil, &ValidationError{Field: "classification.reasoning", Message: "max 1000 characters"}
                }

                // Validate recommendations count
                if len(c.Recommendations) > 10 {
                    return nil, &ValidationError{Field: "classification.recommendations", Message: "max 10 items"}
                }
            }

            // All validations passed, invoke next
            return next(alert)
        }
    }
}
```

#### 5.3.2 Caching Middleware

```go
// NewCachingMiddleware adds LRU caching to avoid redundant formatting.
// Cache key: fingerprint + format + classificationHash
// Cache TTL: 5 minutes
// Hit rate target: 30%+
func NewCachingMiddleware(cache Cache) FormatterMiddleware {
    return func(next formatFunc) formatFunc {
        return func(alert *core.EnrichedAlert) (map[string]any, error) {
            // Generate cache key
            key := generateCacheKey(alert)

            // Check cache
            if cached, ok := cache.Get(key); ok {
                // Cache hit! Return immediately
                return cached.(map[string]any), nil
            }

            // Cache miss, invoke next
            result, err := next(alert)
            if err != nil {
                return nil, err // don't cache errors
            }

            // Store in cache
            cache.Set(key, result, 5*time.Minute)

            return result, nil
        }
    }
}

// generateCacheKey creates a unique key for caching
func generateCacheKey(alert *core.EnrichedAlert) string {
    h := fnv.New128a()
    h.Write([]byte(alert.Alert.Fingerprint))

    // Include classification hash (if present)
    if alert.Classification != nil {
        h.Write([]byte(alert.Classification.Reasoning))
        h.Write([]byte(fmt.Sprintf("%.2f", alert.Classification.Confidence)))
    }

    return hex.EncodeToString(h.Sum(nil))
}
```

#### 5.3.3 Metrics Middleware

```go
// NewMetricsMiddleware records Prometheus metrics for formatting operations
func NewMetricsMiddleware(metrics *FormatterMetrics) FormatterMiddleware {
    return func(next formatFunc) formatFunc {
        return func(alert *core.EnrichedAlert) (map[string]any, error) {
            start := time.Now()

            // Invoke next in chain
            result, err := next(alert)

            // Record metrics
            duration := time.Since(start).Seconds()
            metrics.formatDuration.Observe(duration)

            if err != nil {
                metrics.formatErrors.Inc()
            } else {
                metrics.formatTotal.Inc()
            }

            return result, err
        }
    }
}
```

---

## 6. Caching Strategy

### 6.1 Cache Interface

```go
// Cache provides key-value storage with TTL
type Cache interface {
    Get(key string) (value interface{}, ok bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
    Stats() CacheStats
}

// CacheStats provides cache performance metrics
type CacheStats struct {
    Hits       int64
    Misses     int64
    Evictions  int64
    Size       int
    MaxSize    int
    HitRate    float64 // hits / (hits + misses)
}
```

### 6.2 LRU Implementation

```go
// LRUCache implements Cache using hashicorp/golang-lru
type LRUCache struct {
    cache     *lru.Cache
    ttl       time.Duration
    hits      atomic.Int64
    misses    atomic.Int64
    evictions atomic.Int64
}

// NewLRUCache creates an LRU cache with size limit and TTL
func NewLRUCache(maxEntries int, ttl time.Duration) Cache {
    cache, _ := lru.NewWithEvict(maxEntries, func(key interface{}, value interface{}) {
        // Track evictions
    })

    return &LRUCache{
        cache: cache,
        ttl:   ttl,
    }
}
```

### 6.3 Cache Key Design

**Format**: `{fingerprint}:{format}:{classificationHash}`

**Examples**:
- `abc123:rootly:a1b2c3d4` (Rootly format with classification)
- `def456:slack:` (Slack format without classification)

**Hash Algorithm**: FNV-1a (fast, low collision)

---

## 7. Validation Framework

### 7.1 Validation Errors

```go
// ValidationError indicates invalid input
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: field '%s': %s", e.Field, e.Message)
}
```

### 7.2 Validation Rules

| Field | Rules |
|-------|-------|
| `fingerprint` | Not empty, max 64 chars |
| `alert_name` | Not empty, max 255 chars |
| `status` | Enum: "firing" or "resolved" |
| `starts_at` | Not zero time |
| `classification.severity` | Enum: critical, warning, info, noise |
| `classification.confidence` | Range: 0.0 to 1.0 |
| `classification.reasoning` | Max 1,000 chars |
| `classification.recommendations` | Max 10 items |

---

## 8. Monitoring Integration

### 8.1 Prometheus Metrics

```go
type FormatterMetrics struct {
    // formatDuration measures formatting latency (histogram)
    // Labels: format (alertmanager, rootly, pagerduty, slack, webhook)
    formatDuration *prometheus.HistogramVec

    // formatTotal counts successful formatting operations (counter)
    // Labels: format
    formatTotal *prometheus.CounterVec

    // formatErrors counts formatting errors (counter)
    // Labels: format, error_type
    formatErrors *prometheus.CounterVec

    // cacheHits counts cache hits (counter)
    cacheHits prometheus.Counter

    // cacheMisses counts cache misses (counter)
    cacheMisses prometheus.Counter

    // registrySize tracks number of registered formats (gauge)
    registrySize prometheus.Gauge
}

// NewFormatterMetrics creates and registers Prometheus metrics
func NewFormatterMetrics(reg prometheus.Registerer) *FormatterMetrics {
    m := &FormatterMetrics{
        formatDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "formatter_format_duration_seconds",
                Help:    "Alert formatting latency in seconds",
                Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05}, // 100μs to 50ms
            },
            []string{"format"},
        ),
        formatTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "formatter_format_total",
                Help: "Total number of formatted alerts",
            },
            []string{"format"},
        ),
        // ... initialize other metrics
    }

    // Register with Prometheus
    reg.MustRegister(m.formatDuration, m.formatTotal, m.formatErrors, m.cacheHits, m.cacheMisses, m.registrySize)

    return m
}
```

### 8.2 OpenTelemetry Tracing

```go
// Span attributes for FormatAlert operation:
span.SetAttributes(
    attribute.String("format", string(format)),
    attribute.String("fingerprint", enrichedAlert.Alert.Fingerprint),
    attribute.String("alert_name", enrichedAlert.Alert.AlertName),
    attribute.Bool("has_classification", enrichedAlert.Classification != nil),
    attribute.Bool("cache_hit", cacheHit),
)

// Span events:
span.AddEvent("validation_start")
span.AddEvent("validation_complete")
span.AddEvent("cache_lookup")
span.AddEvent("formatting_start")
span.AddEvent("formatting_complete")
```

---

## 9. Performance Optimization

### 9.1 Target Latencies

| Format | Target (p50) | Target (p99) | Baseline |
|--------|--------------|--------------|----------|
| **Alertmanager** | <400μs | <800μs | ~2ms |
| **Rootly** | <500μs | <1ms | ~3ms |
| **PagerDuty** | <300μs | <600μs | ~1.5ms |
| **Slack** | <600μs | <1.2ms | ~4ms |
| **Webhook** | <200μs | <400μs | ~1ms |

### 9.2 Optimization Strategies

1. **String concatenation**: Use `strings.Builder` (not `+`)
   ```go
   var b strings.Builder
   b.Grow(256) // pre-allocate
   b.WriteString("**Alert:** ")
   b.WriteString(alert.AlertName)
   return b.String()
   ```

2. **Map pre-allocation**: Estimate capacity
   ```go
   payload := make(map[string]any, 10) // avoid resizing
   ```

3. **Avoid JSON marshal/unmarshal**: Direct map construction
   ```go
   // BAD: marshal → unmarshal → map
   classificationJSON, _ := json.Marshal(enrichedAlert.Classification)
   var classificationMap map[string]any
   json.Unmarshal(classificationJSON, &classificationMap)

   // GOOD: direct map construction
   classificationMap := map[string]any{
       "severity":    string(classification.Severity),
       "confidence":  classification.Confidence,
       "reasoning":   classification.Reasoning,
   }
   ```

4. **Caching**: 30%+ hit rate = 30% of calls <10μs

### 9.3 Benchmarking

```go
func BenchmarkFormatAlertmanager(b *testing.B) {
    formatter := NewAlertFormatter()
    alert := createTestEnrichedAlert()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := formatter.FormatAlert(context.Background(), alert, core.FormatAlertmanager)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Target: <500μs per op, <100 allocs/op
```

---

## 10. Error Handling Strategy

### 10.1 Error Types

```go
// ValidationError indicates invalid input
type ValidationError struct {
    Field   string
    Message string
}

// FormatError indicates format-specific error
type FormatError struct {
    Format  core.PublishingFormat
    Message string
    Cause   error
}

// RegistrationError indicates format registry error
type RegistrationError struct {
    Format  core.PublishingFormat
    Message string
}

// NotFoundError indicates format not registered
type NotFoundError struct {
    Format core.PublishingFormat
}

// CacheError indicates caching failure (non-fatal)
type CacheError struct {
    Operation string
    Cause     error
}
```

### 10.2 Error Handling Flow

```
Client calls FormatAlert()
        │
        ▼
┌───────────────────┐
│ ValidationMiddleware │
│   ├─ nil alert? → ValidationError
│   ├─ empty fingerprint? → ValidationError
│   └─ invalid status? → ValidationError
└───────────────────┘
        │ OK
        ▼
┌───────────────────┐
│ CachingMiddleware │
│   ├─ Cache hit? → return cached (no error)
│   └─ Cache error? → log warning, continue
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Format Registry   │
│   └─ Format not found? → fallback to webhook
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Format Function   │
│   └─ Formatting error? → FormatError
└───────────────────┘
        │
        ▼
Return result to client
```

---

## 11. Data Flow Diagrams

### 11.1 Formatting Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Formatting Flow (150%)                       │
└─────────────────────────────────────────────────────────────────┘

Input: EnrichedAlert + PublishingFormat
   │
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 1: Validation Middleware                                    │
│  - Check alert structure (non-nil, valid fields)                │
│  - Check classification (if present, valid severity/confidence) │
│  - Reject if invalid → return ValidationError                   │
└──────────────────────────────────────────────────────────────────┘
   │ OK
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 2: Caching Middleware                                       │
│  - Generate cache key: fingerprint + format + classificationHash│
│  - Lookup in cache (LRU, 1000 entries, 5min TTL)                │
│  - If HIT: return cached result (<10μs) ✓                       │
│  - If MISS: continue to formatting                              │
└──────────────────────────────────────────────────────────────────┘
   │ Cache MISS
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 3: Tracing Middleware                                       │
│  - Create OpenTelemetry span (formatter.FormatAlert)            │
│  - Add attributes: format, fingerprint, alert_name              │
│  - Add event: formatting_start                                  │
└──────────────────────────────────────────────────────────────────┘
   │
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 4: Metrics Middleware                                       │
│  - Start latency timer                                          │
│  - Increment formatter_format_total counter                     │
└──────────────────────────────────────────────────────────────────┘
   │
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 5: Rate Limit Middleware                                    │
│  - Check rate limit (1000 req/sec)                              │
│  - If exceeded: return RateLimitError                           │
└──────────────────────────────────────────────────────────────────┘
   │ OK
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 6: Format Registry Lookup                                   │
│  - Get format function from registry (O(1) map lookup)          │
│  - If not found: fallback to webhook format                     │
└──────────────────────────────────────────────────────────────────┘
   │
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 7: Format Function Execution                                │
│  - Execute format-specific logic:                               │
│    • Alertmanager: Build webhook v4 structure                   │
│    • Rootly: Build incident format with markdown               │
│    • PagerDuty: Build Events API v2 payload                    │
│    • Slack: Build Blocks API with rich formatting              │
│    • Webhook: Simple JSON passthrough                          │
│  - Inject LLM classification data                               │
│  - Truncate strings if needed (reasoning, recommendations)      │
│  - Target: <500μs execution time                                │
└──────────────────────────────────────────────────────────────────┘
   │
   ▼
┌──────────────────────────────────────────────────────────────────┐
│ Step 8: Post-Processing                                          │
│  - Cache result (store in LRU with 5min TTL)                    │
│  - Record metrics (latency histogram, success counter)          │
│  - Complete OTel span (add event: formatting_complete)          │
│  - Log result (if enabled): fingerprint, format, latency        │
└──────────────────────────────────────────────────────────────────┘
   │
   ▼
Output: map[string]any (formatted alert)
```

### 11.2 Error Handling Flow

```
Error Occurs
   │
   ├─ ValidationError (Step 1)
   │   └─ Return immediately, no retry
   │
   ├─ CacheError (Step 2)
   │   └─ Log warning, continue (non-fatal)
   │
   ├─ RateLimitError (Step 5)
   │   └─ Return 429 Too Many Requests
   │
   ├─ NotFoundError (Step 6)
   │   └─ Fallback to webhook format
   │
   ├─ FormatError (Step 7)
   │   └─ Record metric, return error
   │
   └─ context.DeadlineExceeded (any step)
       └─ Cancel operation, return timeout error
```

---

## 12. API Contracts

### 12.1 Input Contract

```go
// EnrichedAlert (required, non-nil)
type EnrichedAlert struct {
    Alert              *Alert                 // required, non-nil
    Classification     *ClassificationResult  // optional, can be nil
    EnrichmentMetadata map[string]any         // optional
}

// Alert (required fields)
type Alert struct {
    Fingerprint  string            // required, 1-64 chars
    AlertName    string            // required, 1-255 chars
    Status       AlertStatus       // required, "firing" or "resolved"
    Labels       map[string]string // optional, max 100 pairs
    Annotations  map[string]string // optional, max 100 pairs
    StartsAt     time.Time         // required, non-zero
    EndsAt       *time.Time        // optional
    GeneratorURL *string           // optional
}

// ClassificationResult (optional, if present must be valid)
type ClassificationResult struct {
    Severity        AlertSeverity // required, valid enum
    Confidence      float64       // required, 0.0-1.0
    Reasoning       string        // optional, max 1000 chars
    Recommendations []string      // optional, max 10 items
}
```

### 12.2 Output Contract

**Alertmanager Format**:
```json
{
  "receiver": "alert-history-proxy",
  "status": "firing",
  "alerts": [{
    "labels": {...},
    "annotations": {
      "llm_severity": "critical",
      "llm_confidence": "0.85",
      ...
    },
    "startsAt": "2025-11-08T12:00:00Z",
    "fingerprint": "abc123"
  }],
  "version": "4",
  ...
}
```

**Rootly Format**:
```json
{
  "title": "[AlertName] Alert in namespace (AI: critical, 85% confidence)",
  "description": "**Alert:** ...\n**AI Classification:**\n...",
  "severity": "critical",
  "status": "started",
  "tags": ["alertname:...", "severity:..."],
  "environment": "production",
  "started_at": "2025-11-08T12:00:00Z"
}
```

**PagerDuty Format**:
```json
{
  "event_action": "trigger",
  "dedup_key": "abc123",
  "payload": {
    "summary": "[AlertName] firing - AI: critical (85%)",
    "severity": "critical",
    "source": "alert-history-service",
    "custom_details": {
      "ai_classification": {...}
    }
  }
}
```

**Slack Format**:
```json
{
  "blocks": [
    {"type": "header", "text": {"type": "plain_text", "text": "🔴 *AlertName* - firing"}},
    {"type": "section", "fields": [...]},
    ...
  ],
  "attachments": [
    {"color": "#FF0000", "fields": [...]}
  ]
}
```

**Webhook Format**:
```json
{
  "alert_name": "AlertName",
  "fingerprint": "abc123",
  "status": "firing",
  "labels": {...},
  "annotations": {...},
  "starts_at": "2025-11-08T12:00:00Z",
  "classification": {
    "severity": "critical",
    "confidence": 0.85,
    "reasoning": "...",
    "recommendations": [...]
  }
}
```

---

## 13. Testing Strategy

### 13.1 Unit Tests (30+ tests)

**Coverage Target**: 95%+ line coverage

**Test Categories**:
1. **Format tests** (13 existing + 5 new)
   - Test each format with various inputs
   - Test LLM classification injection
   - Test missing classification (fallback)
   - Test edge cases (empty labels, nil fields)

2. **Validation tests** (8 new)
   - Test all validation rules
   - Test ValidationError messages
   - Test edge cases (max lengths, boundary values)

3. **Registry tests** (6 new)
   - Test Register/Unregister/Get/List
   - Test thread safety (concurrent access)
   - Test reference counting

4. **Middleware tests** (10 new)
   - Test each middleware independently
   - Test middleware chain composition
   - Test error propagation

5. **Caching tests** (5 new)
   - Test cache hit/miss scenarios
   - Test TTL expiration
   - Test LRU eviction

### 13.2 Benchmarks (10+ benchmarks)

```go
func BenchmarkFormatAlertmanager(b *testing.B)
func BenchmarkFormatRootly(b *testing.B)
func BenchmarkFormatPagerDuty(b *testing.B)
func BenchmarkFormatSlack(b *testing.B)
func BenchmarkFormatWebhook(b *testing.B)
func BenchmarkFormatWithCache(b *testing.B)         // cache hit scenario
func BenchmarkFormatWithoutCache(b *testing.B)      // cache miss scenario
func BenchmarkValidationMiddleware(b *testing.B)
func BenchmarkRegistryLookup(b *testing.B)
func BenchmarkMiddlewareChain(b *testing.B)
```

**Target**: <500μs per op, <100 allocs/op

### 13.3 Integration Tests (10+ tests)

```go
func TestFormatAlertmanager_RealPayload(t *testing.T)      // validate against schema
func TestFormatRootly_IncidentCreation(t *testing.T)       // test Rootly API (sandbox)
func TestFormatPagerDuty_EventCreation(t *testing.T)       // test PagerDuty API (sandbox)
func TestFormatSlack_MessagePosting(t *testing.T)          // test Slack API (sandbox)
func TestFormatterWithPublisher_EndToEnd(t *testing.T)     // full pipeline test
func TestCachePerformance_HighLoad(t *testing.T)           // load test cache
func TestMiddlewarePipeline_ErrorPropagation(t *testing.T) // test error handling
func TestRegistryThreadSafety_Concurrent(t *testing.T)     // concurrent access
func TestFormatterResilience_Timeout(t *testing.T)         // context timeout
func TestFormatterResilience_PanicRecovery(t *testing.T)   // panic handling
```

### 13.4 Fuzzing (1M+ inputs)

```go
func FuzzFormatAlert(f *testing.F) {
    formatter := NewAlertFormatter()

    // Seed corpus
    f.Add("alert-1", "firing", int64(1699459200))
    f.Add("alert-2", "resolved", int64(1699459300))

    f.Fuzz(func(t *testing.T, alertName string, status string, timestamp int64) {
        alert := &core.EnrichedAlert{
            Alert: &core.Alert{
                Fingerprint: "fuzz-" + alertName,
                AlertName:   alertName,
                Status:      core.AlertStatus(status),
                StartsAt:    time.Unix(timestamp, 0),
                Labels:      map[string]string{},
                Annotations: map[string]string{},
            },
        }

        // Should not panic
        _, _ = formatter.FormatAlert(context.Background(), alert, core.FormatWebhook)
    })
}
```

---

## 14. Security Considerations

### 14.1 Input Sanitization

**Markdown/HTML Escaping** (for Rootly/Slack):
```go
func escapeMarkdown(s string) string {
    replacer := strings.NewReplacer(
        "`", "\\`",
        "*", "\\*",
        "_", "\\_",
        "[", "\\[",
        "]", "\\]",
    )
    return replacer.Replace(s)
}
```

### 14.2 Size Limits

- **Alert size**: Max 1MB (K8s limit)
- **Classification reasoning**: Max 1,000 chars
- **Recommendations**: Max 10 items
- **Labels/annotations**: Max 100 pairs each

### 14.3 Rate Limiting

**RateLimitMiddleware**: Max 1,000 req/sec per formatter instance
- Prevents DoS attacks
- Token bucket algorithm
- Returns `429 Too Many Requests` if exceeded

### 14.4 Audit Logging

```go
// Log all format operations (for compliance)
logger.InfoContext(ctx, "alert formatted",
    slog.String("fingerprint", alert.Fingerprint),
    slog.String("format", string(format)),
    slog.Duration("latency", latency),
    slog.Bool("cache_hit", cacheHit),
)
```

---

## 15. Migration Path

### 15.1 Baseline to 150% Migration

**Step 1: Add Registry** (no breaking changes)
```go
// Old code (still works)
formatter := NewAlertFormatter()
result, _ := formatter.FormatAlert(ctx, alert, format)

// New code (with registry)
formatter := NewEnhancedAlertFormatter(
    WithRegistry(customRegistry),
)
result, _ := formatter.FormatAlert(ctx, alert, format)
```

**Step 2: Add Middleware** (opt-in)
```go
// Enable caching
formatter := NewEnhancedAlertFormatter(
    WithMiddleware(CachingMiddleware),
)
```

**Step 3: Enable Monitoring** (opt-in)
```go
// Enable metrics
formatter := NewEnhancedAlertFormatter(
    WithMetrics(prometheus.DefaultRegisterer),
    WithTracing(otelprovider),
)
```

### 15.2 Backward Compatibility

✅ **No breaking API changes**
✅ **DefaultAlertFormatter still works**
✅ **EnhancedAlertFormatter is opt-in**
✅ **Existing tests still pass**

---

## Document Metadata

**Version**: 1.0
**Author**: AI Assistant (TN-051 150% Quality Enhancement)
**Date**: 2025-11-08
**Status**: 🎯 **COMPLETE** (Phase 2 of 9)
**Next**: tasks.md (Implementation Plan, 900+ LOC)

**Change Log**:
- 2025-11-08: Comprehensive design document (1,100+ LOC)
- Architecture: 5-layer design, middleware pipeline, format registry
- Performance: Sub-millisecond targets, caching, optimization strategies
- Monitoring: Prometheus metrics, OpenTelemetry tracing
- Testing: 30+ unit tests, 10+ benchmarks, 10+ integration tests, fuzzing

---

**🎯 TN-051 Design Complete - Ready for Tasks Phase**

**Next Step**: Create `tasks.md` with detailed implementation plan (900+ LOC target).
