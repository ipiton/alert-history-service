# TN-153: Template Engine Integration — Technical Design

**Date**: 2025-11-22
**Task ID**: TN-153
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Design Complete → Ready for Implementation

---

## 📊 Architecture Overview

### High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                    Notification Pipeline                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Routing Engine (TN-137-141)                     │
│  • Route matching                                                │
│  • Receiver selection                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              🎯 TEMPLATE ENGINE (TN-153)                         │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  1. Template Parser                                       │  │
│  │     • Parse Go text/template                              │  │
│  │     • Validate syntax                                     │  │
│  │     • Cache parsed templates (LRU)                        │  │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  2. Template Executor                                     │  │
│  │     • Execute with TemplateData                           │  │
│  │     • Apply custom functions (50+)                        │  │
│  │     • Handle errors gracefully                            │  │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  3. Function Library                                      │  │
│  │     • Time functions (date, humanize, since)              │  │
│  │     • String functions (toUpper, truncate, join)          │  │
│  │     • URL functions (urlEncode, pathJoin)                 │  │
│  │     • Math functions (add, humanize, round)               │  │
│  │     • Conditional functions (default, empty)              │  │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  4. Template Cache                                        │  │
│  │     • LRU cache (1000 templates)                          │  │
│  │     • SHA256 key generation                               │  │
│  │     • Hot reload support (SIGHUP invalidation)            │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Alert Formatter (TN-051)                          │
│  • Format for Slack/PagerDuty/Email                             │
│  • Apply rendered templates                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Publishers (TN-053, TN-054)                     │
│  • Send to Slack/PagerDuty/Email                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🏗️ Component Design

### 1. NotificationTemplateEngine

**Purpose**: Core template engine для notification messages.

**Interface**:
```go
// NotificationTemplateEngine handles template parsing and execution
// for notification messages (Slack, PagerDuty, Email).
//
// Thread Safety: Safe for concurrent use.
//
// Performance:
//   - Parse: < 10ms p95
//   - Execute (cached): < 5ms p95
//   - Execute (uncached): < 20ms p95
//
// Example:
//
//	engine := NewNotificationTemplateEngine(opts)
//	result, err := engine.Execute(ctx, tmpl, data)
type NotificationTemplateEngine interface {
	// Execute parses and executes a template with given data
	//
	// Parameters:
	//   - ctx: Context with timeout (5s max)
	//   - tmpl: Template string (Go text/template syntax)
	//   - data: Template data (TemplateData struct)
	//
	// Returns:
	//   - string: Rendered result
	//   - error: Parse or execution error
	//
	// Caching:
	//   - Templates are cached by SHA256(tmpl)
	//   - Cache hit: < 1ms
	//   - Cache miss: parse + cache + execute
	//
	// Error Handling:
	//   - Parse errors: return ErrTemplateParse
	//   - Execution errors: fallback to raw template
	//   - Timeout: return ErrTemplateTimeout
	Execute(ctx context.Context, tmpl string, data *TemplateData) (string, error)

	// ExecuteMultiple executes multiple templates in parallel
	//
	// Useful for processing multiple fields (title, text, pretext).
	// Returns results in same order as input templates.
	//
	// Parameters:
	//   - ctx: Context with timeout
	//   - templates: Map of field name → template string
	//   - data: Template data
	//
	// Returns:
	//   - map[string]string: field name → rendered result
	//   - error: If any template fails
	ExecuteMultiple(ctx context.Context, templates map[string]string, data *TemplateData) (map[string]string, error)

	// InvalidateCache clears template cache
	//
	// Called on config reload (SIGHUP).
	// Thread-safe operation.
	InvalidateCache()

	// GetCacheStats returns cache statistics
	//
	// Returns:
	//   - CacheStats: hit/miss counts, size, hit ratio
	GetCacheStats() CacheStats
}
```

**Implementation**:
```go
// DefaultNotificationTemplateEngine implements NotificationTemplateEngine
type DefaultNotificationTemplateEngine struct {
	// cache stores parsed templates (LRU, 1000 max)
	cache *TemplateCache

	// funcs are custom template functions
	funcs template.FuncMap

	// metrics tracks Prometheus metrics
	metrics *TemplateMetrics

	// logger for structured logging
	logger *slog.Logger

	// opts controls engine behavior
	opts TemplateEngineOptions
}

// TemplateEngineOptions configures the engine
type TemplateEngineOptions struct {
	// CacheSize is max number of cached templates (default: 1000)
	CacheSize int

	// ExecutionTimeout is max execution time per template (default: 5s)
	ExecutionTimeout time.Duration

	// EnableMetrics enables Prometheus metrics (default: true)
	EnableMetrics bool

	// FallbackOnError returns raw template on execution error (default: true)
	FallbackOnError bool
}

// DefaultTemplateEngineOptions returns default options
func DefaultTemplateEngineOptions() TemplateEngineOptions {
	return TemplateEngineOptions{
		CacheSize:        1000,
		ExecutionTimeout: 5 * time.Second,
		EnableMetrics:    true,
		FallbackOnError:  true,
	}
}
```

---

### 2. TemplateData

**Purpose**: Data structure passed to templates.

**Definition**:
```go
// TemplateData contains all data available to notification templates.
//
// Compatible with Alertmanager template data structure.
//
// Example Template:
//
//	Title: "🔥 {{ .GroupLabels.alertname }} - {{ .Status }}"
//	Text: |
//	  *Severity*: {{ .Labels.severity | default "unknown" }}
//	  *Instance*: {{ .Labels.instance }}
//	  *Started*: {{ .StartsAt | humanizeTimestamp }}
type TemplateData struct {
	// ===================================================================
	// Alert Fields
	// ===================================================================

	// Status is alert status: "firing" or "resolved"
	Status string

	// Labels are alert labels (e.g., alertname, severity, instance)
	Labels map[string]string

	// Annotations are alert annotations (e.g., summary, description)
	Annotations map[string]string

	// StartsAt is when alert started firing
	StartsAt time.Time

	// EndsAt is when alert resolved (zero if still firing)
	EndsAt time.Time

	// GeneratorURL is Prometheus generator URL
	GeneratorURL string

	// Fingerprint is unique alert fingerprint
	Fingerprint string

	// Value is alert value (if available)
	// For threshold alerts: current metric value
	Value float64

	// ===================================================================
	// Group Fields (for grouped notifications)
	// ===================================================================

	// GroupLabels are labels used for grouping
	// Example: {"alertname": "HighCPU", "cluster": "prod"}
	GroupLabels map[string]string

	// CommonLabels are labels common to all alerts in group
	CommonLabels map[string]string

	// CommonAnnotations are annotations common to all alerts in group
	CommonAnnotations map[string]string

	// GroupKey is unique group identifier
	GroupKey string

	// ===================================================================
	// External URLs
	// ===================================================================

	// ExternalURL is Alert History external URL
	// Example: "https://alerts.company.com"
	ExternalURL string

	// SilenceURL is direct link to create silence for this alert
	// Example: "https://alerts.company.com/silences/new?filter=..."
	SilenceURL string

	// ===================================================================
	// Receiver Context
	// ===================================================================

	// Receiver is receiver name
	Receiver string

	// ReceiverType is receiver type: "slack", "pagerduty", "email"
	ReceiverType string
}

// NewTemplateData creates TemplateData from alert and group info
func NewTemplateData(alert *core.Alert, groupInfo *GroupInfo, receiver string) *TemplateData {
	data := &TemplateData{
		Status:       alert.Status,
		Labels:       alert.Labels,
		Annotations:  alert.Annotations,
		StartsAt:     alert.StartsAt,
		EndsAt:       alert.EndsAt,
		GeneratorURL: alert.GeneratorURL,
		Fingerprint:  alert.Fingerprint,
		Receiver:     receiver,
	}

	// Add group info if available
	if groupInfo != nil {
		data.GroupLabels = groupInfo.GroupLabels
		data.CommonLabels = groupInfo.CommonLabels
		data.CommonAnnotations = groupInfo.CommonAnnotations
		data.GroupKey = groupInfo.GroupKey
	}

	return data
}
```

---

### 3. Template Functions Library

**Purpose**: 50+ custom functions для template formatting.

**Categories**:

#### 3.1 Time Functions (20 functions)

```go
// Time formatting
funcs["date"] = func(fmt string, t time.Time) string {
	return t.Format(fmt)
}

funcs["humanizeTimestamp"] = func(t time.Time) string {
	duration := time.Since(t)
	return humanizeDuration(duration) + " ago"
}

funcs["since"] = func(t time.Time) string {
	return humanizeDuration(time.Since(t))
}

funcs["until"] = func(t time.Time) string {
	return humanizeDuration(time.Until(t))
}

// Duration formatting
funcs["humanizeDuration"] = func(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// More time functions from sprig:
// - now, ago, toDate, mustToDate, unixEpoch
// - dateInZone, dateModify, mustDateModify
// - htmlDate, htmlDateInZone, dateAgo
```

#### 3.2 String Functions (15 functions)

```go
// Case conversion
funcs["toUpper"] = strings.ToUpper
funcs["toLower"] = strings.ToLower
funcs["title"] = strings.Title

// Truncation
funcs["truncate"] = func(max int, s string) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

funcs["truncateWords"] = func(max int, s string) string {
	words := strings.Fields(s)
	if len(words) <= max {
		return s
	}
	return strings.Join(words[:max], " ") + "..."
}

// Joining/splitting
funcs["join"] = func(sep string, items []string) string {
	return strings.Join(items, sep)
}

funcs["split"] = func(sep string, s string) []string {
	return strings.Split(s, sep)
}

// Trimming
funcs["trim"] = strings.TrimSpace
funcs["trimPrefix"] = strings.TrimPrefix
funcs["trimSuffix"] = strings.TrimSuffix

// More string functions from sprig:
// - repeat, substr, nospace, initials
// - wrap, wrapWith, quote, squote
```

#### 3.3 URL Functions (5 functions)

```go
// URL encoding
funcs["urlEncode"] = url.QueryEscape
funcs["urlDecode"] = url.QueryUnescape

// URL building
funcs["urlQuery"] = func(baseURL, key, value string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

funcs["pathJoin"] = func(parts ...string) string {
	return filepath.Join(parts...)
}

funcs["pathBase"] = filepath.Base
```

#### 3.4 Math Functions (10 functions)

```go
// Arithmetic
funcs["add"] = func(a, b float64) float64 { return a + b }
funcs["sub"] = func(a, b float64) float64 { return a - b }
funcs["mul"] = func(a, b float64) float64 { return a * b }
funcs["div"] = func(a, b float64) float64 { return a / b }
funcs["mod"] = func(a, b int) int { return a % b }

// Comparison
funcs["max"] = func(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

funcs["min"] = func(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Rounding
funcs["round"] = func(f float64) int { return int(math.Round(f)) }
funcs["ceil"] = func(f float64) int { return int(math.Ceil(f)) }
funcs["floor"] = func(f float64) int { return int(math.Floor(f)) }

// Humanize numbers
funcs["humanize"] = func(f float64) string {
	if f >= 1e9 {
		return fmt.Sprintf("%.2fG", f/1e9)
	}
	if f >= 1e6 {
		return fmt.Sprintf("%.2fM", f/1e6)
	}
	if f >= 1e3 {
		return fmt.Sprintf("%.2fk", f/1e3)
	}
	return fmt.Sprintf("%.2f", f)
}

funcs["humanize1024"] = func(f float64) string {
	if f >= 1<<30 {
		return fmt.Sprintf("%.2f GiB", f/(1<<30))
	}
	if f >= 1<<20 {
		return fmt.Sprintf("%.2f MiB", f/(1<<20))
	}
	if f >= 1<<10 {
		return fmt.Sprintf("%.2f KiB", f/(1<<10))
	}
	return fmt.Sprintf("%.2f B", f)
}
```

#### 3.5 Conditional Functions (5 functions)

```go
// Default value
funcs["default"] = func(defaultVal, val interface{}) interface{} {
	if val == nil || val == "" {
		return defaultVal
	}
	return val
}

// Empty check
funcs["empty"] = func(val interface{}) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

// Ternary
funcs["ternary"] = func(trueVal, falseVal interface{}, cond bool) interface{} {
	if cond {
		return trueVal
	}
	return falseVal
}

// Has key
funcs["has"] = func(key string, m map[string]interface{}) bool {
	_, exists := m[key]
	return exists
}

// Coalesce (first non-empty value)
funcs["coalesce"] = func(vals ...interface{}) interface{} {
	for _, val := range vals {
		if val != nil && val != "" {
			return val
		}
	}
	return nil
}
```

#### 3.6 Collection Functions (10 functions)

```go
// Sort
funcs["sortAlpha"] = func(list []string) []string {
	sorted := make([]string, len(list))
	copy(sorted, list)
	sort.Strings(sorted)
	return sorted
}

// Reverse
funcs["reverse"] = func(list []string) []string {
	reversed := make([]string, len(list))
	for i, v := range list {
		reversed[len(list)-1-i] = v
	}
	return reversed
}

// Unique
funcs["uniq"] = func(list []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range list {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// More collection functions from sprig:
// - without, has, compact, slice, append
// - first, rest, last, initial, prepend
```

#### 3.7 Encoding Functions (5 functions)

```go
// Base64
funcs["b64enc"] = func(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

funcs["b64dec"] = func(s string) string {
	decoded, _ := base64.StdEncoding.DecodeString(s)
	return string(decoded)
}

// JSON
funcs["toJson"] = func(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

funcs["fromJson"] = func(s string) interface{} {
	var v interface{}
	json.Unmarshal([]byte(s), &v)
	return v
}

// More encoding functions from sprig:
// - toRawJson, toPrettyJson, mustToJson
```

---

### 4. Template Cache

**Purpose**: LRU cache для parsed templates.

**Implementation**:
```go
// TemplateCache provides thread-safe LRU cache for parsed templates
type TemplateCache struct {
	// cache is the underlying LRU cache
	cache *lru.Cache

	// mu protects cache access
	mu sync.RWMutex

	// metrics tracks cache statistics
	hits   uint64
	misses uint64
}

// NewTemplateCache creates a new cache with given size
func NewTemplateCache(size int) (*TemplateCache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}

	return &TemplateCache{
		cache: cache,
	}, nil
}

// Get retrieves a template from cache
func (c *TemplateCache) Get(key string) (*template.Template, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.cache.Get(key)
	if ok {
		atomic.AddUint64(&c.hits, 1)
		return val.(*template.Template), true
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// Set stores a template in cache
func (c *TemplateCache) Set(key string, tmpl *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Add(key, tmpl)
}

// Invalidate clears all cached templates
func (c *TemplateCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Purge()
}

// Stats returns cache statistics
func (c *TemplateCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses

	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:     hits,
		Misses:   misses,
		Size:     c.cache.Len(),
		HitRatio: hitRatio,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Hits     uint64  // Total cache hits
	Misses   uint64  // Total cache misses
	Size     int     // Current cache size
	HitRatio float64 // Hit ratio (0.0-1.0)
}

// generateCacheKey generates SHA256 hash of template string
func generateCacheKey(tmpl string) string {
	hash := sha256.Sum256([]byte(tmpl))
	return hex.EncodeToString(hash[:])
}
```

---

### 5. Receiver Integration

**Purpose**: Integrate template engine with receiver configs.

**Integration Points**:

#### 5.1 Slack Integration

```go
// ProcessSlackConfig renders templates in SlackConfig
func ProcessSlackConfig(
	ctx context.Context,
	engine NotificationTemplateEngine,
	config *SlackConfig,
	data *TemplateData,
) (*SlackConfig, error) {
	// Clone config to avoid mutation
	processed := config.Clone()

	// Render title
	if config.Title != "" {
		title, err := engine.Execute(ctx, config.Title, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render title: %w", err)
		}
		processed.Title = title
	}

	// Render text
	if config.Text != "" {
		text, err := engine.Execute(ctx, config.Text, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render text: %w", err)
		}
		processed.Text = text
	}

	// Render pretext
	if config.Pretext != "" {
		pretext, err := engine.Execute(ctx, config.Pretext, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render pretext: %w", err)
		}
		processed.Pretext = pretext
	}

	// Render fields
	for i, field := range config.Fields {
		if field.Title != "" {
			title, err := engine.Execute(ctx, field.Title, data)
			if err != nil {
				return nil, fmt.Errorf("failed to render field[%d].title: %w", i, err)
			}
			processed.Fields[i].Title = title
		}

		if field.Value != "" {
			value, err := engine.Execute(ctx, field.Value, data)
			if err != nil {
				return nil, fmt.Errorf("failed to render field[%d].value: %w", i, err)
			}
			processed.Fields[i].Value = value
		}
	}

	return processed, nil
}
```

#### 5.2 PagerDuty Integration

```go
// ProcessPagerDutyConfig renders templates in PagerDutyConfig
func ProcessPagerDutyConfig(
	ctx context.Context,
	engine NotificationTemplateEngine,
	config *PagerDutyConfig,
	data *TemplateData,
) (*PagerDutyConfig, error) {
	processed := config.Clone()

	// Render summary
	if config.Summary != "" {
		summary, err := engine.Execute(ctx, config.Summary, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render summary: %w", err)
		}
		processed.Summary = summary
	}

	// Render details (map[string]string)
	if config.Details != nil {
		processed.Details = make(map[string]string)
		for key, tmpl := range config.Details {
			value, err := engine.Execute(ctx, tmpl, data)
			if err != nil {
				return nil, fmt.Errorf("failed to render details[%s]: %w", key, err)
			}
			processed.Details[key] = value
		}
	}

	return processed, nil
}
```

#### 5.3 Email Integration (FUTURE - TN-154)

```go
// ProcessEmailConfig renders templates in EmailConfig
func ProcessEmailConfig(
	ctx context.Context,
	engine NotificationTemplateEngine,
	config *EmailConfig,
	data *TemplateData,
) (*EmailConfig, error) {
	processed := config.Clone()

	// Render subject
	if config.Subject != "" {
		subject, err := engine.Execute(ctx, config.Subject, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render subject: %w", err)
		}
		processed.Subject = subject
	}

	// Render body
	if config.Body != "" {
		body, err := engine.Execute(ctx, config.Body, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render body: %w", err)
		}
		processed.Body = body
	}

	return processed, nil
}
```

---

### 6. Prometheus Metrics

**Purpose**: Track template engine performance and errors.

**Metrics**:
```go
// TemplateMetrics tracks Prometheus metrics
type TemplateMetrics struct {
	// Total executions
	executionsTotal *prometheus.CounterVec

	// Execution duration
	executionDuration *prometheus.HistogramVec

	// Parse errors
	parseErrors *prometheus.CounterVec

	// Cache metrics
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter
	cacheSize   prometheus.Gauge

	// Function calls
	functionCalls *prometheus.CounterVec
}

// NewTemplateMetrics creates metrics
func NewTemplateMetrics() *TemplateMetrics {
	m := &TemplateMetrics{
		executionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "template_executions_total",
				Help: "Total template executions",
			},
			[]string{"status"}, // success, error
		),

		executionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "template_execution_duration_seconds",
				Help:    "Template execution duration",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"cached"}, // true, false
		),

		parseErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "template_parse_errors_total",
				Help: "Total template parse errors",
			},
			[]string{"error_type"}, // syntax, timeout, invalid_data
		),

		cacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "template_cache_hits_total",
				Help: "Total template cache hits",
			},
		),

		cacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "template_cache_misses_total",
				Help: "Total template cache misses",
			},
		),

		cacheSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "template_cache_size",
				Help: "Current template cache size",
			},
		),

		functionCalls: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "template_function_calls_total",
				Help: "Total template function calls",
			},
			[]string{"function"}, // date, humanize, toUpper, etc.
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.executionsTotal,
		m.executionDuration,
		m.parseErrors,
		m.cacheHits,
		m.cacheMisses,
		m.cacheSize,
		m.functionCalls,
	)

	return m
}

// RecordExecution records template execution
func (m *TemplateMetrics) RecordExecution(status string, duration time.Duration, cached bool) {
	m.executionsTotal.WithLabelValues(status).Inc()
	m.executionDuration.WithLabelValues(fmt.Sprintf("%t", cached)).Observe(duration.Seconds())
}

// RecordParseError records parse error
func (m *TemplateMetrics) RecordParseError(errorType string) {
	m.parseErrors.WithLabelValues(errorType).Inc()
}

// RecordCacheHit records cache hit
func (m *TemplateMetrics) RecordCacheHit() {
	m.cacheHits.Inc()
}

// RecordCacheMiss records cache miss
func (m *TemplateMetrics) RecordCacheMiss() {
	m.cacheMisses.Inc()
}

// UpdateCacheSize updates cache size gauge
func (m *TemplateMetrics) UpdateCacheSize(size int) {
	m.cacheSize.Set(float64(size))
}

// RecordFunctionCall records function call
func (m *TemplateMetrics) RecordFunctionCall(function string) {
	m.functionCalls.WithLabelValues(function).Inc()
}
```

---

## 📦 Package Structure

```
go-app/internal/notification/
├── template/
│   ├── engine.go              # NotificationTemplateEngine interface + impl
│   ├── engine_test.go         # Unit tests (30+ tests)
│   ├── data.go                # TemplateData struct
│   ├── data_test.go           # TemplateData tests
│   ├── functions.go           # Template functions library (50+ funcs)
│   ├── functions_test.go      # Function tests
│   ├── cache.go               # TemplateCache implementation
│   ├── cache_test.go          # Cache tests
│   ├── metrics.go             # TemplateMetrics
│   ├── errors.go              # Error types
│   ├── integration.go         # Receiver integration helpers
│   ├── integration_test.go    # Integration tests
│   └── README.md              # Package documentation
```

---

## 🔄 Sequence Diagrams

### Template Execution Flow

```
┌──────┐     ┌────────┐     ┌──────┐     ┌───────┐     ┌──────────┐
│Client│     │Engine  │     │Cache │     │Parser│     │Executor  │
└──┬───┘     └───┬────┘     └──┬───┘     └───┬──┘     └────┬─────┘
   │             │              │             │             │
   │ Execute()   │              │             │             │
   ├────────────>│              │             │             │
   │             │              │             │             │
   │             │ Get(key)     │             │             │
   │             ├─────────────>│             │             │
   │             │              │             │             │
   │             │ [cache miss] │             │             │
   │             │<─────────────┤             │             │
   │             │              │             │             │
   │             │ Parse(tmpl)  │             │             │
   │             ├──────────────┼────────────>│             │
   │             │              │             │             │
   │             │ *template.Template         │             │
   │             │<─────────────┼─────────────┤             │
   │             │              │             │             │
   │             │ Set(key, tmpl)             │             │
   │             ├─────────────>│             │             │
   │             │              │             │             │
   │             │ Execute(tmpl, data)        │             │
   │             ├──────────────┼─────────────┼────────────>│
   │             │              │             │             │
   │             │ result       │             │             │
   │             │<─────────────┼─────────────┼─────────────┤
   │             │              │             │             │
   │ result      │              │             │             │
   │<────────────┤              │             │             │
   │             │              │             │             │
```

### Receiver Integration Flow

```
┌────────┐     ┌────────┐     ┌────────┐     ┌─────────┐
│Routing │     │Template│     │Formatter│     │Publisher│
│Engine  │     │Engine  │     │         │     │         │
└───┬────┘     └───┬────┘     └────┬────┘     └────┬────┘
    │              │               │               │
    │ Route alert  │               │               │
    │ to receiver  │               │               │
    │              │               │               │
    │ ProcessSlackConfig()         │               │
    ├─────────────>│               │               │
    │              │               │               │
    │              │ Execute(title, data)          │
    │              ├──────────┐    │               │
    │              │          │    │               │
    │              │<─────────┘    │               │
    │              │               │               │
    │              │ Execute(text, data)           │
    │              ├──────────┐    │               │
    │              │          │    │               │
    │              │<─────────┘    │               │
    │              │               │               │
    │ Rendered config              │               │
    │<─────────────┤               │               │
    │              │               │               │
    │ FormatSlack(config)          │               │
    ├──────────────┼──────────────>│               │
    │              │               │               │
    │ Formatted payload            │               │
    │<─────────────┼───────────────┤               │
    │              │               │               │
    │ Publish(payload)             │               │
    ├──────────────┼───────────────┼──────────────>│
    │              │               │               │
```

---

## 🧪 Testing Strategy

### Unit Tests (30+ tests)

**Coverage Target**: 90%+

**Test Categories**:
1. **Engine Tests** (10 tests)
   - Parse valid template
   - Parse invalid template
   - Execute with valid data
   - Execute with missing fields
   - Execute with timeout
   - Cache hit
   - Cache miss
   - Cache invalidation
   - Concurrent execution
   - Fallback on error

2. **Function Tests** (15 tests)
   - Time functions (date, humanize, since)
   - String functions (toUpper, truncate, join)
   - URL functions (urlEncode, pathJoin)
   - Math functions (add, humanize, round)
   - Conditional functions (default, empty, ternary)

3. **Integration Tests** (5 tests)
   - ProcessSlackConfig
   - ProcessPagerDutyConfig
   - ProcessEmailConfig
   - Multiple receivers
   - Error handling

---

## 📊 Performance Benchmarks

```go
// Benchmark template parsing
func BenchmarkTemplateParse(b *testing.B) {
	engine := NewNotificationTemplateEngine(DefaultTemplateEngineOptions())
	tmpl := `{{ .GroupLabels.alertname }} - {{ .Status }}`
	data := &TemplateData{
		GroupLabels: map[string]string{"alertname": "HighCPU"},
		Status:      "firing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(context.Background(), tmpl, data)
	}
}

// Benchmark cached execution
func BenchmarkTemplateExecuteCached(b *testing.B) {
	engine := NewNotificationTemplateEngine(DefaultTemplateEngineOptions())
	tmpl := `{{ .GroupLabels.alertname }} - {{ .Status }}`
	data := &TemplateData{
		GroupLabels: map[string]string{"alertname": "HighCPU"},
		Status:      "firing",
	}

	// Warm up cache
	_, _ = engine.Execute(context.Background(), tmpl, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(context.Background(), tmpl, data)
	}
}

// Target Results:
// BenchmarkTemplateParse-8         100000    10000 ns/op  (< 10ms)
// BenchmarkTemplateExecuteCached-8 500000     2000 ns/op  (< 5ms)
```

---

## 🔒 Security Considerations

1. **Sandboxed Execution**: Templates cannot access filesystem or network
2. **Timeout Protection**: 5s max execution time per template
3. **No Arbitrary Code**: Only predefined functions allowed
4. **Input Validation**: Template data validated before execution
5. **Error Handling**: Errors logged but don't crash service

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Status**: ✅ APPROVED - Ready for Implementation
