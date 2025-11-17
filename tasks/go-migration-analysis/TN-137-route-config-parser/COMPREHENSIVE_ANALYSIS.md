# TN-137: Route Config Parser (YAML) — Comprehensive Analysis

**Date**: 2025-11-17
**Author**: Vitalii Semenov (AI-assisted)
**Target Quality**: 150% (Grade A+ Enterprise)
**Module**: Phase B: Advanced Features / Модуль 4: Advanced Routing

---

## Executive Summary

TN-137 extends the existing grouping configuration parser (TN-121) to create a **full Alertmanager-compatible routing engine**. This task is the foundation of Module 4 (Advanced Routing) and unblocks TN-138 through TN-141.

### Current State (TN-121 Baseline)

**✅ Implemented:**
- `Route` struct with nested routes support
- YAML parsing with `yaml.v3`
- Match/MatchRE matchers
- Continue flag for multi-receiver routing
- Timer configuration (group_wait, group_interval, repeat_interval)
- Semantic validation (label names, timer ranges, depth limits)
- Recursive defaults application

**📄 Key Files:**
- `go-app/internal/infrastructure/grouping/config.go` (277 LOC)
- `go-app/internal/infrastructure/grouping/parser.go` (327 LOC)
- `config/grouping.yaml` (76 LOC example)

**🎯 Test Coverage:**
- TN-121: 93.6% coverage
- 158 passing tests
- 12 benchmarks (8.1x faster than targets)

### Gap Analysis

| Component | Exists | Missing | Priority |
|-----------|--------|---------|----------|
| **Route Structure** | ✅ 95% | Receiver reference validation | CRITICAL |
| **Receivers Section** | ❌ 0% | Full implementation | CRITICAL |
| **Receiver Types** | ❌ 0% | Webhook/PagerDuty/Slack/Email configs | CRITICAL |
| **Cross-validation** | ⚠️ 20% | Route→Receiver, cycle detection | CRITICAL |
| **Hot Reload** | ❌ 0% | SIGHUP handler, versioning | HIGH |
| **Inheritance** | ⚠️ 50% | Explicit child→parent inheritance | HIGH |
| **Global Config** | ❌ 0% | resolve_timeout, smtp_* | MEDIUM |
| **API Endpoints** | ❌ 0% | GET/POST /api/v2/config | MEDIUM |

---

## Technical Architecture

### 1. Enhanced Data Models

#### 1.1 RouteConfig (Extended)

```go
// RouteConfig represents the complete Alertmanager-compatible configuration.
// Extends TN-121 GroupingConfig with receivers support.
type RouteConfig struct {
    // Global configuration (NEW)
    Global *GlobalConfig `yaml:"global,omitempty"`

    // Route tree (EXISTING - from TN-121)
    Route *Route `yaml:"route" validate:"required"`

    // Receivers configuration (NEW - CRITICAL)
    Receivers []*Receiver `yaml:"receivers" validate:"required,dive"`

    // Templates configuration (FUTURE - TN-153)
    Templates []string `yaml:"templates,omitempty"`

    // Inhibit rules (EXISTING - from TN-126)
    InhibitRules []InhibitRule `yaml:"inhibit_rules,omitempty"`

    // Internal metadata
    Version   int       `yaml:"-"`
    LoadedAt  time.Time `yaml:"-"`
    SourceFile string   `yaml:"-"`
}

// GlobalConfig defines global alerting parameters.
type GlobalConfig struct {
    // How long to wait for alert to resolve before notifying
    ResolveTimeout Duration `yaml:"resolve_timeout"`

    // SMTP configuration (FUTURE - TN-154)
    SMTPFrom      string `yaml:"smtp_from,omitempty"`
    SMTPSmartHost string `yaml:"smtp_smarthost,omitempty"`
    SMTPAuthUser  string `yaml:"smtp_auth_username,omitempty"`
    SMTPAuthPass  string `yaml:"smtp_auth_password,omitempty"`

    // HTTP client configuration
    HTTPConfig *HTTPConfig `yaml:"http_config,omitempty"`
}

// HTTPConfig defines HTTP client settings for receivers.
type HTTPConfig struct {
    ProxyURL         string        `yaml:"proxy_url,omitempty"`
    TLSConfig        *TLSConfig    `yaml:"tls_config,omitempty"`
    FollowRedirects  bool          `yaml:"follow_redirects"`
    ConnectTimeout   time.Duration `yaml:"connect_timeout"`
    RequestTimeout   time.Duration `yaml:"request_timeout"`
}

// TLSConfig defines TLS settings.
type TLSConfig struct {
    CAFile             string `yaml:"ca_file,omitempty"`
    CertFile           string `yaml:"cert_file,omitempty"`
    KeyFile            string `yaml:"key_file,omitempty"`
    ServerName         string `yaml:"server_name,omitempty"`
    InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}
```

#### 1.2 Receiver Models (NEW - CRITICAL)

```go
// Receiver represents a notification receiver with multiple integration types.
// Supports: webhook, PagerDuty, Slack, email, etc.
type Receiver struct {
    // Name is the unique identifier for this receiver.
    // Must be referenced by at least one route.
    Name string `yaml:"name" validate:"required,min=1,max=255"`

    // Webhook configurations
    WebhookConfigs []*WebhookConfig `yaml:"webhook_configs,omitempty"`

    // PagerDuty configurations
    PagerDutyConfigs []*PagerDutyConfig `yaml:"pagerduty_configs,omitempty"`

    // Slack configurations
    SlackConfigs []*SlackConfig `yaml:"slack_configs,omitempty"`

    // Email configurations (FUTURE - TN-154)
    EmailConfigs []*EmailConfig `yaml:"email_configs,omitempty"`
}

// WebhookConfig defines a webhook receiver configuration.
type WebhookConfig struct {
    // URL to send webhook payload
    URL string `yaml:"url" validate:"required,url,https"`

    // HTTP method (default: POST)
    HTTPMethod string `yaml:"http_method,omitempty"`

    // Custom headers
    HTTPHeaders map[string]string `yaml:"http_headers,omitempty"`

    // HTTP client configuration
    HTTPConfig *HTTPConfig `yaml:"http_config,omitempty"`

    // Notification interval parameters
    SendResolved bool `yaml:"send_resolved"`
    MaxAlerts    int  `yaml:"max_alerts,omitempty"`
}

// PagerDutyConfig defines a PagerDuty Events API v2 receiver.
// Integration с TN-053 (PagerDuty Publisher).
type PagerDutyConfig struct {
    // Integration key (routing_key)
    RoutingKey string `yaml:"routing_key" validate:"required"`

    // Service key (for legacy integration)
    ServiceKey string `yaml:"service_key,omitempty"`

    // API URL (default: https://events.pagerduty.com)
    URL string `yaml:"url,omitempty"`

    // Incident customization
    Severity    string            `yaml:"severity,omitempty"`
    Class       string            `yaml:"class,omitempty"`
    Component   string            `yaml:"component,omitempty"`
    Group       string            `yaml:"group,omitempty"`
    Details     map[string]string `yaml:"details,omitempty"`

    // Notification settings
    SendResolved bool `yaml:"send_resolved"`

    // HTTP client configuration
    HTTPConfig *HTTPConfig `yaml:"http_config,omitempty"`
}

// SlackConfig defines a Slack webhook receiver.
// Integration с TN-054 (Slack Publisher).
type SlackConfig struct {
    // Webhook URL (incoming webhook)
    APIURL string `yaml:"api_url" validate:"required,url"`

    // Channel override (default: webhook channel)
    Channel string `yaml:"channel,omitempty"`

    // Username to post as
    Username string `yaml:"username,omitempty"`

    // Emoji or URL for bot icon
    IconEmoji string `yaml:"icon_emoji,omitempty"`
    IconURL   string `yaml:"icon_url,omitempty"`

    // Message customization
    Title         string            `yaml:"title,omitempty"`
    TitleLink     string            `yaml:"title_link,omitempty"`
    Pretext       string            `yaml:"pretext,omitempty"`
    Text          string            `yaml:"text,omitempty"`
    Fields        []SlackField      `yaml:"fields,omitempty"`
    Actions       []SlackAction     `yaml:"actions,omitempty"`
    Color         string            `yaml:"color,omitempty"`

    // Notification settings
    SendResolved bool `yaml:"send_resolved"`
    ShortFields  bool `yaml:"short_fields"`

    // HTTP client configuration
    HTTPConfig *HTTPConfig `yaml:"http_config,omitempty"`
}

// SlackField represents a Slack attachment field.
type SlackField struct {
    Title string `yaml:"title"`
    Value string `yaml:"value"`
    Short bool   `yaml:"short,omitempty"`
}

// SlackAction represents a Slack attachment action button.
type SlackAction struct {
    Type  string `yaml:"type"`
    Text  string `yaml:"text"`
    URL   string `yaml:"url,omitempty"`
    Style string `yaml:"style,omitempty"`
}

// EmailConfig defines an email receiver (FUTURE - TN-154).
type EmailConfig struct {
    To           string            `yaml:"to" validate:"required,email"`
    From         string            `yaml:"from,omitempty"`
    Smarthost    string            `yaml:"smarthost,omitempty"`
    AuthUsername string            `yaml:"auth_username,omitempty"`
    AuthPassword string            `yaml:"auth_password,omitempty"`
    Headers      map[string]string `yaml:"headers,omitempty"`
    HTML         string            `yaml:"html,omitempty"`
    Text         string            `yaml:"text,omitempty"`
    RequireTLS   bool              `yaml:"require_tls"`
}
```

### 2. Enhanced Parser Logic

#### 2.1 Parsing Flow

```
YAML File → Parse → Validate Structure → Apply Defaults → Validate Semantics → Compile Regex → Build Receiver Index → Return Config
                      ↓                    ↓                    ↓                ↓                   ↓
                   YAML errors        Required fields      Custom rules     MatchRE patterns   receiver→route map
```

#### 2.2 Validation Layers

**Layer 1: YAML Syntax** (existing)
- `yaml.Unmarshal` errors
- Invalid field types
- Duplicate keys

**Layer 2: Structural Validation** (existing - via validator tags)
- Required fields
- Min/max constraints
- Format validation (URL, email)

**Layer 3: Semantic Validation** (ENHANCED)

```go
func (p *RouteConfigParser) validateSemantics(config *RouteConfig) error {
    var errors ValidationErrors

    // 1. Build receiver index (name → receiver)
    receiverIndex := p.buildReceiverIndex(config.Receivers)

    // 2. Validate route tree references
    p.validateRouteTreeReferences(config.Route, receiverIndex, &errors, "route")

    // 3. Detect cycles in route tree
    if cycle := p.detectCycles(config.Route); cycle != nil {
        errors.Add("route", "", "cycle", fmt.Sprintf("cycle detected: %v", cycle))
    }

    // 4. Validate receiver configurations
    p.validateReceivers(config.Receivers, &errors)

    // 5. Check for unused receivers (WARNING only)
    p.checkUnusedReceivers(config.Route, receiverIndex, &errors)

    // 6. Validate regex patterns in MatchRE
    p.validateRegexPatterns(config.Route, &errors)

    if errors.HasErrors() {
        return errors
    }
    return nil
}
```

**Layer 4: Cross-Reference Validation** (NEW)

- **Receiver References**: Every `route.receiver` must exist in `receivers`
- **Duplicate Detection**: Receiver names must be unique
- **Unused Receivers**: Warning if receiver defined but never used
- **Cycle Detection**: No route can reference itself via nested routes

**Layer 5: Type-Specific Validation** (NEW)

```go
func (p *RouteConfigParser) validateWebhookConfig(cfg *WebhookConfig, path string, errors *ValidationErrors) {
    // URL validation (HTTPS only in production)
    if cfg.URL != "" {
        if !strings.HasPrefix(cfg.URL, "https://") {
            errors.Add(fmt.Sprintf("%s.url", path), cfg.URL, "https_required",
                "webhook URLs must use HTTPS in production")
        }

        // SSRF protection (no private IPs)
        if err := validateURLNotPrivate(cfg.URL); err != nil {
            errors.Add(fmt.Sprintf("%s.url", path), cfg.URL, "ssrf", err.Error())
        }
    }

    // Header validation (no sensitive data in plain text)
    for key, value := range cfg.HTTPHeaders {
        if isSensitiveHeader(key) && !isSecretReference(value) {
            errors.Add(fmt.Sprintf("%s.http_headers.%s", path, key), "***", "sensitive_data",
                "sensitive headers should use secret references, not plain text")
        }
    }

    // Method validation
    if cfg.HTTPMethod != "" {
        if !isValidHTTPMethod(cfg.HTTPMethod) {
            errors.Add(fmt.Sprintf("%s.http_method", path), cfg.HTTPMethod, "invalid_method",
                "http_method must be GET, POST, PUT, DELETE, or PATCH")
        }
    }
}
```

### 3. Integration Points

#### 3.1 With Publishing System (TN-046 to TN-060)

```go
// RouteEvaluator (TN-140) uses RouteConfig to route alerts
type RouteEvaluator struct {
    config    *RouteConfig
    receivers map[string]*Receiver  // indexed by name
    matchers  *RouteMatcher         // TN-139
}

// Routing flow:
// Alert → RouteEvaluator.Evaluate(alert) → Matched Routes → Receivers → PublisherFactory → Publishers
```

**Integration Strategy:**
1. `RouteConfig.Receivers` defines WHAT to publish to (targets)
2. `PublishingTarget` (TN-047) stores HOW to connect (credentials from K8s Secrets)
3. `PublisherFactory` creates publishers based on receiver type
4. Credentials never in `RouteConfig` - always from K8s Secrets

#### 3.2 With Grouping System (TN-121 to TN-125)

```go
// AlertGroupManager uses Route configuration
type AlertGroupManager struct {
    config    *RouteConfig
    keyGen    *GroupKeyGenerator  // TN-122
    timers    *TimerManager       // TN-124
    storage   *GroupStorage       // TN-125
}

// Grouping flow:
// Alert → Route.Match → GroupBy labels → GroupKey → AlertGroupManager
```

**Inheritance Rules:**
- Child routes inherit `group_by`, `group_wait`, `group_interval`, `repeat_interval` from parent
- Child can override any inherited value
- Root route provides defaults for all unspecified values

#### 3.3 With Configuration API (TN-149 to TN-152)

```go
// GET /api/v2/config - export current config
func (h *ConfigHandler) GetConfig(c *gin.Context) {
    config := h.configManager.GetCurrent()

    // Sanitize secrets
    sanitized := sanitizeConfig(config)

    c.JSON(200, sanitized)
}

// POST /api/v2/config - dynamic update
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
    var newConfig RouteConfig
    if err := c.BindJSON(&newConfig); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Validate
    parser := NewRouteConfigParser()
    if err := parser.Validate(&newConfig); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Apply (hot reload)
    if err := h.configManager.Apply(&newConfig); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"status": "applied", "version": newConfig.Version})
}
```

---

## Performance Requirements

### Parsing Performance

| Operation | Target | Current (TN-121) | Gap |
|-----------|--------|------------------|-----|
| Parse 100-route config | < 10ms | ~2-3ms | ✅ Exceeds |
| Parse 1000-route config | < 100ms | N/A | Testing needed |
| Hot reload | < 100ms | N/A | Not implemented |
| Receiver validation | < 5ms | N/A | Not implemented |

### Memory Requirements

| Config Size | Routes | Receivers | Expected Memory | Max Memory |
|-------------|--------|-----------|-----------------|------------|
| Small | 10 | 5 | < 1 MB | 5 MB |
| Medium | 100 | 50 | < 10 MB | 50 MB |
| Large | 1000 | 500 | < 100 MB | 500 MB |
| Enterprise | 10,000 | 5,000 | < 1 GB | 5 GB |

### Validation Performance

```go
// Benchmarks (targets for 150% quality)
BenchmarkParseSmallConfig-8      1000000   1500 ns/op   (target: < 5µs)
BenchmarkParseMediumConfig-8      100000  15000 ns/op   (target: < 50µs)
BenchmarkParseLargeConfig-8        10000 150000 ns/op   (target: < 500µs)
BenchmarkValidateReceivers-8     1000000   2000 ns/op   (target: < 10µs)
BenchmarkDetectCycles-8           500000   3000 ns/op   (target: < 20µs)
```

---

## Security Considerations

### 1. YAML Bomb Protection

```go
// Max limits to prevent YAML bombs
const (
    MaxConfigSize      = 10 * 1024 * 1024  // 10 MB
    MaxRouteDepth      = 10                // 10 levels
    MaxRoutes          = 10000             // 10K routes
    MaxReceivers       = 5000              // 5K receivers
    MaxMatchersPerRoute = 100               // 100 matchers
)

func (p *RouteConfigParser) ParseFile(path string) (*RouteConfig, error) {
    // Check file size before reading
    info, err := os.Stat(path)
    if err != nil {
        return nil, err
    }
    if info.Size() > MaxConfigSize {
        return nil, fmt.Errorf("config file too large: %d bytes (max: %d)", info.Size(), MaxConfigSize)
    }

    // Read file
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    return p.Parse(data)
}
```

### 2. SSRF Protection

```go
// Prevent receivers from calling private IPs
func validateURLNotPrivate(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return err
    }

    // Resolve hostname to IP
    ips, err := net.LookupIP(u.Hostname())
    if err != nil {
        return fmt.Errorf("failed to resolve hostname: %w", err)
    }

    // Check if any IP is private
    for _, ip := range ips {
        if isPrivateIP(ip) {
            return fmt.Errorf("receiver URL resolves to private IP: %s", ip)
        }
    }

    return nil
}

func isPrivateIP(ip net.IP) bool {
    // RFC 1918 private ranges
    privateRanges := []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",    // localhost
        "169.254.0.0/16", // link-local
        "::1/128",        // IPv6 localhost
        "fc00::/7",       // IPv6 unique local
    }

    for _, cidr := range privateRanges {
        _, network, _ := net.ParseCIDR(cidr)
        if network.Contains(ip) {
            return true
        }
    }

    return false
}
```

### 3. Secret Sanitization

```go
// Never log or expose secrets in API responses
func sanitizeConfig(config *RouteConfig) *RouteConfig {
    sanitized := config.Clone()

    for _, receiver := range sanitized.Receivers {
        // Sanitize webhook configs
        for _, wc := range receiver.WebhookConfigs {
            for key := range wc.HTTPHeaders {
                if isSensitiveHeader(key) {
                    wc.HTTPHeaders[key] = "***REDACTED***"
                }
            }
        }

        // Sanitize PagerDuty configs
        for _, pdc := range receiver.PagerDutyConfigs {
            if pdc.RoutingKey != "" {
                pdc.RoutingKey = "***REDACTED***"
            }
            if pdc.ServiceKey != "" {
                pdc.ServiceKey = "***REDACTED***"
            }
        }

        // Sanitize Slack configs
        for _, sc := range receiver.SlackConfigs {
            if sc.APIURL != "" {
                sc.APIURL = sanitizeWebhookURL(sc.APIURL)
            }
        }
    }

    return sanitized
}

func isSensitiveHeader(key string) bool {
    sensitive := []string{
        "authorization",
        "x-api-key",
        "x-auth-token",
        "apikey",
        "token",
        "bearer",
    }

    lowerKey := strings.ToLower(key)
    for _, s := range sensitive {
        if strings.Contains(lowerKey, s) {
            return true
        }
    }

    return false
}
```

---

## Testing Strategy

### Unit Tests (Target: 30+ tests, 85%+ coverage)

```go
// Config model tests (8 tests)
- TestRouteConfigUnmarshalYAML
- TestRouteConfigValidation
- TestReceiverValidation
- TestWebhookConfigValidation
- TestPagerDutyConfigValidation
- TestSlackConfigValidation
- TestGlobalConfigDefaults
- TestHTTPConfigValidation

// Parser tests (12 tests)
- TestParseValidConfig
- TestParseInvalidYAML
- TestParseMissingReceiver
- TestParseUnknownReceiverReference
- TestParseDuplicateReceiver
- TestParseCyclicRoute
- TestParseExcessiveNesting
- TestParseYAMLBomb
- TestParseWithDefaults
- TestParseHotReload
- TestParseReceiverIndex
- TestParseRegexCompilation

// Validation tests (10 tests)
- TestValidateReceiverReferences
- TestValidateCycleDetection
- TestValidateSSRFProtection
- TestValidateSensitiveHeaders
- TestValidateRegexPatterns
- TestValidateUnusedReceivers
- TestValidateInheritance
- TestValidateTimerRanges
- TestValidateMatcherCombinations
- TestValidateReceiverTypes
```

### Integration Tests (Target: 10+ tests)

```go
// End-to-end parsing (5 tests)
- TestParseProductionConfig
- TestParseMinimalConfig
- TestParseComplexNestedRoutes
- TestParseMultiReceiverContinue
- TestParseWithAllReceiverTypes

// Hot reload tests (3 tests)
- TestHotReloadSuccess
- TestHotReloadValidationFail
- TestHotReloadRollback

// Integration with other components (2 tests)
- TestIntegrationWithGroupingEngine
- TestIntegrationWithPublishingSystem
```

### Benchmarks (Target: 8+ benchmarks)

```go
BenchmarkParseSmallConfig       // 10 routes, 5 receivers
BenchmarkParseMediumConfig      // 100 routes, 50 receivers
BenchmarkParseLargeConfig       // 1000 routes, 500 receivers
BenchmarkValidateReceivers      // 1000 receivers
BenchmarkDetectCycles           // Complex route tree
BenchmarkBuildReceiverIndex     // 5000 receivers
BenchmarkApplyDefaults          // Deep route tree
BenchmarkSanitizeConfig         // Large config with secrets
```

---

## Implementation Phases

### Phase 0: Analysis & Research ✅ (2h)

**Status**: COMPLETE

**Deliverables**:
- ✅ Existing code analysis (config.go, parser.go)
- ✅ Alertmanager spec review
- ✅ Gap identification
- ✅ Integration points mapping
- ✅ Performance benchmarks review (TN-121)

### Phase 1: Documentation (4h)

**Deliverables**:
- requirements.md (600+ LOC)
- design.md (1,000+ LOC)
- tasks.md (900+ LOC)

### Phase 2: Git Branch Setup (15min)

```bash
git checkout -b feature/TN-137-route-config-parser-150pct
```

### Phase 3: Enhanced Models (6h)

**Scope**:
- `RouteConfig` struct
- `Receiver` struct
- `WebhookConfig`, `PagerDutyConfig`, `SlackConfig`
- `GlobalConfig`, `HTTPConfig`, `TLSConfig`
- Validation tags
- Godoc comments

**Files to create/modify**:
- `go-app/internal/infrastructure/routing/config.go` (NEW - 500 LOC)
- `go-app/internal/infrastructure/routing/receiver.go` (NEW - 400 LOC)
- `go-app/internal/infrastructure/routing/validation.go` (NEW - 300 LOC)

### Phase 4: Extended Parser (8h)

**Scope**:
- RouteConfigParser implementation
- Receiver validation
- Cross-reference validation
- Cycle detection
- Regex compilation
- Hot reload mechanism
- Secret sanitization

**Files to create/modify**:
- `go-app/internal/infrastructure/routing/parser.go` (NEW - 600 LOC)
- `go-app/internal/infrastructure/routing/parser_validate.go` (NEW - 400 LOC)
- `go-app/internal/infrastructure/routing/parser_security.go` (NEW - 200 LOC)
- `go-app/internal/infrastructure/routing/errors.go` (NEW - 150 LOC)

### Phase 5: Comprehensive Testing (10h)

**Scope**:
- 30+ unit tests
- 10+ integration tests
- 8+ benchmarks
- Test fixtures (YAML configs)

**Target**: 85%+ coverage

**Files to create**:
- `go-app/internal/infrastructure/routing/config_test.go` (400 LOC)
- `go-app/internal/infrastructure/routing/parser_test.go` (600 LOC)
- `go-app/internal/infrastructure/routing/validation_test.go` (350 LOC)
- `go-app/internal/infrastructure/routing/parser_bench_test.go` (250 LOC)
- `go-app/internal/infrastructure/routing/testdata/*.yaml` (10 fixtures, 500 LOC)

### Phase 6: Performance Optimization (4h)

**Scope**:
- Benchmark-driven optimization
- Memory pooling для repeated parsing
- Regex caching
- Receiver index pre-building

**Target**:
- Parse 100-route config: < 10ms
- Parse 1000-route config: < 100ms
- Receiver validation: < 5ms

### Phase 7: Security Hardening (3h)

**Scope**:
- YAML bomb protection (size limits)
- SSRF prevention (private IP checks)
- Secret sanitization (header masking)
- Input validation (URL, email formats)

**Tools**:
- gosec scanner
- nancy dependency checker
- Custom validators

### Phase 8: Observability Integration (2h)

**Scope**:
- 3 Prometheus metrics:
  - `routing_config_parse_duration_seconds` (Histogram)
  - `routing_config_validation_errors_total` (Counter by error_type)
  - `routing_config_hot_reload_total` (Counter by status: success/failure)
- Structured logging (slog)
- Parse error tracking

### Phase 9: Certification (3h)

**Scope**:
- Final quality assessment
- 150% checklist verification
- Production readiness review
- CERTIFICATION.md creation

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Alertmanager incompatibility** | LOW | HIGH | Follow official spec v0.27+, comprehensive compatibility tests |
| **Breaking changes to TN-121** | MEDIUM | HIGH | Extend, don't replace. Keep GroupingConfig for backward compat |
| **Performance regression** | LOW | MEDIUM | Benchmark every change, target 2x margin |
| **Incomplete validation** | MEDIUM | HIGH | 4-layer validation (YAML → struct → semantic → cross-ref) |
| **YAML bombs / DOS** | MEDIUM | HIGH | Size limits (10MB), depth limits (10 levels), timeouts |
| **SSRF vulnerabilities** | LOW | HIGH | Private IP checks, DNS validation, allowlist support |
| **Test coverage gaps** | LOW | MEDIUM | 85%+ target, comprehensive edge cases |

---

## Dependencies

### Requires (Completed)

- ✅ TN-121: Grouping Configuration Parser (93.6% coverage, 150% quality)
- ✅ TN-122: Group Key Generator (95%+ coverage, 200% quality)
- ✅ TN-046: Kubernetes Client (72.8% coverage, 150% quality)
- ✅ TN-047: Target Discovery Manager (88.6% coverage, 147% quality)

### Blocks (Downstream)

- ⏸️ **TN-138**: Route Tree Builder (needs RouteConfig structure)
- ⏸️ **TN-139**: Route Matcher (needs compiled regex)
- ⏸️ **TN-140**: Route Evaluator (needs receiver index)
- ⏸️ **TN-141**: Multi-Receiver Support (needs Continue flag logic)

### Optional Integration

- 🔄 **TN-149-152**: Configuration Management API (can use parser)
- 🔄 **TN-153-156**: Template System (extends receiver configs)

---

## Success Criteria (150% Quality)

### Functional Requirements (100%)

- ✅ Parse Alertmanager v0.27+ compatible YAML
- ✅ Support all receiver types (webhook, PagerDuty, Slack, email)
- ✅ Validate receiver references (route → receiver)
- ✅ Detect cycles in route tree
- ✅ Apply defaults recursively
- ✅ Compile regex patterns
- ✅ Build receiver index

### Non-Functional Requirements (150%)

**Performance** (50% extra):
- ✅ Parse 100-route config: < 5ms (target: < 10ms) = 200% 🚀
- ✅ Parse 1000-route config: < 50ms (target: < 100ms) = 200% 🚀
- ✅ Receiver validation: < 2ms (target: < 5ms) = 250% 🚀

**Testing** (50% extra):
- ✅ Unit tests: 35+ (target: 30+) = 117%
- ✅ Integration tests: 12+ (target: 10+) = 120%
- ✅ Benchmarks: 10+ (target: 8+) = 125%
- ✅ Coverage: 90%+ (target: 85%+) = 106%

**Documentation** (50% extra):
- ✅ requirements.md: 700+ LOC (target: 600+) = 117%
- ✅ design.md: 1,200+ LOC (target: 1,000+) = 120%
- ✅ tasks.md: 1,000+ LOC (target: 900+) = 111%
- ✅ Godoc: 100% coverage
- ✅ Examples: 10+ YAML fixtures

**Security** (50% extra):
- ✅ YAML bomb protection
- ✅ SSRF prevention
- ✅ Secret sanitization
- ✅ Input validation
- ✅ gosec scan clean

**Observability** (50% extra):
- ✅ 3+ Prometheus metrics
- ✅ Structured logging (slog)
- ✅ Error categorization
- ✅ Performance tracking

---

## References

### Alertmanager Specification

- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Route Configuration](https://prometheus.io/docs/alerting/latest/configuration/#route)
- [Receiver Configuration](https://prometheus.io/docs/alerting/latest/configuration/#receiver)
- [Webhook Configuration](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config)
- [PagerDuty Configuration](https://prometheus.io/docs/alerting/latest/configuration/#pagerduty_config)
- [Slack Configuration](https://prometheus.io/docs/alerting/latest/configuration/#slack_config)

### Internal Documentation

- TN-121: Grouping Configuration Parser
- TN-122: Group Key Generator
- TN-046: Kubernetes Client
- TN-047: Target Discovery Manager
- TN-053: PagerDuty Publisher
- TN-054: Slack Publisher
- TN-055: Generic Webhook Publisher

### External Libraries

- `gopkg.in/yaml.v3`: YAML parsing
- `github.com/go-playground/validator/v10`: Validation
- `regexp`: Regex compilation
- `net/url`: URL validation
- `net`: IP validation

---

## Appendix: Example Configurations

### A. Minimal Configuration

```yaml
route:
  receiver: default
  group_by: [alertname]

receivers:
  - name: default
    webhook_configs:
      - url: https://example.com/webhook
```

### B. Production Configuration

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: default
  group_by: [alertname, cluster]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h

  routes:
    - match:
        severity: critical
      receiver: pagerduty
      group_wait: 10s
      continue: true

    - match:
        severity: warning
      receiver: slack
      group_wait: 1m

    - match_re:
        service: ^database.*
      receiver: dba-team
      group_by: [alertname, instance]

receivers:
  - name: default
    webhook_configs:
      - url: https://alertmanager.example.com/webhook
        send_resolved: true

  - name: pagerduty
    pagerduty_configs:
      - routing_key: ${PAGERDUTY_ROUTING_KEY}
        severity: critical
        send_resolved: true

  - name: slack
    slack_configs:
      - api_url: ${SLACK_WEBHOOK_URL}
        channel: "#alerts"
        title: "Alert: {{ .GroupLabels.alertname }}"
        text: "{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}"
        send_resolved: true

  - name: dba-team
    slack_configs:
      - api_url: ${SLACK_DBA_WEBHOOK_URL}
        channel: "#dba-alerts"
        username: "Database Bot"
```

### C. Complex Multi-Receiver Configuration

```yaml
global:
  resolve_timeout: 10m
  http_config:
    follow_redirects: true
    connect_timeout: 10s
    request_timeout: 30s

route:
  receiver: catch-all
  group_by: [alertname, cluster, namespace]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 24h

  routes:
    # Critical alerts go to PagerDuty AND Slack
    - match:
        severity: critical
      receiver: critical-oncall
      group_wait: 0s
      repeat_interval: 1h
      continue: true

    # High-priority namespaces
    - match:
        namespace: production
      receiver: production-team
      group_wait: 10s
      repeat_interval: 4h

      routes:
        # Database alerts within production
        - match_re:
            alertname: ^(MySQL|PostgreSQL|Redis).*
          receiver: dba-team
          group_by: [alertname, instance]
          group_wait: 5s

        # API alerts within production
        - match_re:
            alertname: ^API.*
          receiver: api-team
          group_by: [alertname, endpoint]

    # Infrastructure alerts
    - match:
        category: infrastructure
      receiver: infra-team
      group_wait: 2m
      repeat_interval: 12h

    # Development environment (low priority)
    - match:
        namespace: development
      receiver: dev-slack
      group_wait: 5m
      repeat_interval: 24h

receivers:
  - name: catch-all
    webhook_configs:
      - url: https://webhook.example.com/catch-all
        send_resolved: false

  - name: critical-oncall
    pagerduty_configs:
      - routing_key: ${PAGERDUTY_ONCALL_KEY}
        severity: critical
        send_resolved: true
        http_config:
          request_timeout: 5s
    slack_configs:
      - api_url: ${SLACK_ONCALL_WEBHOOK}
        channel: "#critical-alerts"
        title: "🚨 CRITICAL: {{ .GroupLabels.alertname }}"
        color: danger
        send_resolved: true

  - name: production-team
    pagerduty_configs:
      - routing_key: ${PAGERDUTY_PROD_KEY}
        severity: warning
        send_resolved: true
    slack_configs:
      - api_url: ${SLACK_PROD_WEBHOOK}
        channel: "#prod-alerts"

  - name: dba-team
    slack_configs:
      - api_url: ${SLACK_DBA_WEBHOOK}
        channel: "#dba-alerts"
        username: "Database Monitor"
        icon_emoji: ":database:"

  - name: api-team
    slack_configs:
      - api_url: ${SLACK_API_WEBHOOK}
        channel: "#api-alerts"

  - name: infra-team
    webhook_configs:
      - url: https://infra.example.com/alerts
        send_resolved: true
        http_headers:
          X-Team: infrastructure

  - name: dev-slack
    slack_configs:
      - api_url: ${SLACK_DEV_WEBHOOK}
        channel: "#dev-alerts"
        send_resolved: false
```

---

**End of Comprehensive Analysis**

**Next Steps**:
1. Review and approve analysis
2. Create requirements.md
3. Create design.md
4. Create tasks.md
5. Begin implementation (Phase 2)

**Estimated Total Duration**: 40-50 hours (6-7 days)

**Target Completion Date**: 2025-11-24 (1 week)

**Quality Target**: Grade A+ (150%+ achievement)
