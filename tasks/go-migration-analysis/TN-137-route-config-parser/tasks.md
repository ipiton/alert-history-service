# TN-137: Route Config Parser (YAML) — Implementation Tasks

**Task ID**: TN-137
**Module**: Phase B: Advanced Features / Модуль 4: Advanced Routing
**Priority**: CRITICAL
**Target Quality**: 150% (Grade A+ Enterprise)
**Status**: IN PROGRESS
**Started**: 2025-11-17

---

## Task Overview

**Objective**: Extend TN-121 Grouping Configuration Parser to implement full Alertmanager v0.27+ compatible routing engine with multiple receiver types (webhook, PagerDuty, Slack, email).

**Scope**:
- RouteConfig model с receivers support
- 4-layer validation (YAML → structural → semantic → cross-ref)
- Receiver types: WebhookConfig, PagerDutyConfig, SlackConfig
- Security: YAML bombs, SSRF, secret sanitization
- Performance: 200%+ better than targets

**Dependencies**:
- ✅ TN-121: Grouping Configuration Parser (COMPLETE)
- ✅ TN-122: Group Key Generator (COMPLETE)
- ✅ TN-046: Kubernetes Client (COMPLETE)
- ✅ TN-047: Target Discovery Manager (COMPLETE)

**Blocks**:
- TN-138: Route Tree Builder
- TN-139: Route Matcher
- TN-140: Route Evaluator
- TN-141: Multi-Receiver Support

---

## Phases & Timeline

| Phase | Description | Estimate | Status |
|-------|-------------|----------|--------|
| 0 | Comprehensive Analysis & Research | 2h | ✅ COMPLETE |
| 1 | Documentation (requirements, design, tasks) | 4h | ✅ COMPLETE |
| 2 | Git Branch Setup | 15min | 🔄 IN PROGRESS |
| 3 | Enhanced Route Models | 6h | ⏸️ PENDING |
| 4 | Extended Parser Implementation | 8h | ⏸️ PENDING |
| 5 | Comprehensive Testing | 10h | ⏸️ PENDING |
| 6 | Performance Optimization | 4h | ⏸️ PENDING |
| 7 | Security Hardening | 3h | ⏸️ PENDING |
| 8 | Observability Integration | 2h | ⏸️ PENDING |
| 9 | Final Certification | 3h | ⏸️ PENDING |
| **TOTAL** | **All Phases** | **42-45h** | **5% COMPLETE** |

**Target Completion**: 2025-11-24 (1 week, 6-7 days)

---

## Phase 0: Comprehensive Analysis ✅ COMPLETE

**Duration**: 2 hours
**Status**: ✅ COMPLETE (2025-11-17)

### Completed Tasks

- [x] Analyze existing TN-121 Route implementation
- [x] Review config.go (277 LOC)
- [x] Review parser.go (327 LOC)
- [x] Study Alertmanager v0.27+ specification
- [x] Identify gaps (receivers, global config, validation)
- [x] Map integration points (TN-046, TN-053, TN-054, TN-055)
- [x] Performance benchmarks review (TN-121: 8.1x faster)

### Deliverables

- ✅ COMPREHENSIVE_ANALYSIS.md (1,000+ LOC)
- ✅ Gap analysis (8 components identified)
- ✅ Architecture diagrams
- ✅ Integration strategy

---

## Phase 1: Documentation ✅ COMPLETE

**Duration**: 4 hours
**Status**: ✅ COMPLETE (2025-11-17)

### Completed Tasks

- [x] Create requirements.md (700+ LOC)
  - [x] FR-1: Route configuration (nested, Match/MatchRE)
  - [x] FR-2: Receiver configuration (4 types)
  - [x] FR-3: Global configuration
  - [x] FR-4: 4-layer validation
  - [x] FR-5: Configuration loading
  - [x] NFR-1 to NFR-7: Performance, reliability, security

- [x] Create design.md (1,200+ LOC)
  - [x] Architecture overview
  - [x] Data models (RouteConfig, Receiver, 4 config types)
  - [x] Parser implementation
  - [x] Validation logic (4 layers)
  - [x] Integration architecture
  - [x] Performance targets
  - [x] Security design

- [x] Create tasks.md (this file) (900+ LOC)

### Deliverables

- ✅ requirements.md (700+ LOC)
- ✅ design.md (1,200+ LOC)
- ✅ tasks.md (900+ LOC)
- ✅ **TOTAL**: 2,800+ LOC documentation (140% of 2,000 LOC target)

---

## Phase 2: Git Branch Setup 🔄 IN PROGRESS

**Duration**: 15 minutes
**Status**: 🔄 IN PROGRESS

### Tasks

- [ ] Create feature branch
  ```bash
  git checkout -b feature/TN-137-route-config-parser-150pct
  ```

- [ ] Verify clean state
  ```bash
  git status
  go build ./...
  golangci-lint run
  ```

- [ ] Create directory structure
  ```bash
  mkdir -p go-app/internal/infrastructure/routing
  mkdir -p go-app/internal/infrastructure/routing/testdata
  ```

- [ ] Create placeholder files
  ```bash
  touch go-app/internal/infrastructure/routing/{config,receiver,global,parser,parser_validate,parser_security,errors,utils}.go
  touch go-app/internal/infrastructure/routing/{config,parser,validation,parser_bench}_test.go
  touch go-app/internal/infrastructure/routing/README.md
  ```

- [ ] Initial commit
  ```bash
  git add .
  git commit -m "feat(TN-137): Initialize Route Config Parser structure"
  ```

### Acceptance Criteria

- ✅ Branch created and checked out
- ✅ Directory structure created
- ✅ All files have package declaration
- ✅ Zero compilation errors
- ✅ Initial commit pushed

---

## Phase 3: Enhanced Route Models

**Duration**: 6 hours
**Status**: ⏸️ PENDING

### 3.1: RouteConfig Model (2h)

**File**: `go-app/internal/infrastructure/routing/config.go` (500 LOC)

#### Tasks

- [ ] Import dependencies
  ```go
  import (
      "regexp"
      "time"
      "github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
  )
  ```

- [ ] Define RouteConfig struct
  - [ ] Global *GlobalConfig (optional)
  - [ ] Route *grouping.Route (required, inherited from TN-121)
  - [ ] Receivers []*Receiver (required, min=1)
  - [ ] Templates []string (optional, FUTURE)
  - [ ] InhibitRules []InhibitRule (optional, from TN-126)
  - [ ] ReceiverIndex map[string]*Receiver (internal)
  - [ ] CompiledRegex map (internal)
  - [ ] Version, LoadedAt, SourceFile (metadata)

- [ ] Implement methods
  - [ ] GetReceiver(name string) (*Receiver, bool) — O(1) lookup
  - [ ] ListReceivers() []*Receiver
  - [ ] GetCompiledRegex(route, key) (*regexp.Regexp, bool)
  - [ ] Validate() error
  - [ ] Clone() *RouteConfig — deep copy

- [ ] Add comprehensive godoc comments (100% coverage)

#### Acceptance Criteria

- ✅ RouteConfig struct defined
- ✅ All 5 methods implemented
- ✅ 100% godoc coverage
- ✅ Zero compilation errors
- ✅ Linter clean

### 3.2: Receiver Models (2h)

**File**: `go-app/internal/infrastructure/routing/receiver.go` (400 LOC)

#### Tasks

- [ ] Define Receiver struct
  - [ ] Name string (required, unique, 1-255 chars)
  - [ ] WebhookConfigs []*WebhookConfig
  - [ ] PagerDutyConfigs []*PagerDutyConfig
  - [ ] SlackConfigs []*SlackConfig
  - [ ] EmailConfigs []*EmailConfig (FUTURE)
  - [ ] Referenced bool (internal)

- [ ] Implement Receiver methods
  - [ ] Validate() error — at least one config
  - [ ] GetConfigCount() int
  - [ ] Clone() *Receiver
  - [ ] Sanitize() *Receiver — redact secrets

- [ ] Define WebhookConfig struct (100 LOC)
  - [ ] URL string (required, HTTPS)
  - [ ] HTTPMethod string (default: POST)
  - [ ] HTTPHeaders map[string]string
  - [ ] HTTPConfig *HTTPConfig
  - [ ] SendResolved *bool
  - [ ] MaxAlerts int (0-1000)
  - [ ] Defaults(), Clone(), Sanitize() methods

- [ ] Define PagerDutyConfig struct (100 LOC)
  - [ ] RoutingKey string (required, 32 chars)
  - [ ] ServiceKey string (optional, legacy)
  - [ ] URL string (default: https://events.pagerduty.com)
  - [ ] Severity, Class, Component, Group, Details
  - [ ] SendResolved *bool
  - [ ] HTTPConfig *HTTPConfig
  - [ ] Defaults(), Clone(), Sanitize() methods

- [ ] Define SlackConfig struct (150 LOC)
  - [ ] APIURL string (required)
  - [ ] Channel, Username, IconEmoji, IconURL
  - [ ] Title, TitleLink, Pretext, Text
  - [ ] Fields []SlackField, Actions []SlackAction
  - [ ] Color string
  - [ ] SendResolved *bool, ShortFields bool
  - [ ] HTTPConfig *HTTPConfig
  - [ ] Defaults(), Clone(), Sanitize() methods

- [ ] Define SlackField struct (20 LOC)
- [ ] Define SlackAction struct (30 LOC)

#### Acceptance Criteria

- ✅ 5 receiver structs defined
- ✅ All Defaults(), Clone(), Sanitize() methods implemented
- ✅ Validation logic for each config type
- ✅ 100% godoc coverage
- ✅ Zero compilation errors

### 3.3: Global Configuration (1h)

**File**: `go-app/internal/infrastructure/routing/global.go` (200 LOC)

#### Tasks

- [ ] Define GlobalConfig struct
  - [ ] ResolveTimeout *Duration (default: 5m)
  - [ ] SMTPFrom, SMTPSmartHost, SMTPAuthUsername, SMTPAuthPassword
  - [ ] SMTPRequireTLS bool
  - [ ] HTTPConfig *HTTPConfig

- [ ] Implement GlobalConfig methods
  - [ ] Defaults()
  - [ ] Clone()

- [ ] Define HTTPConfig struct
  - [ ] ProxyURL string
  - [ ] TLSConfig *TLSConfig
  - [ ] FollowRedirects *bool (default: true)
  - [ ] ConnectTimeout time.Duration (default: 10s)
  - [ ] RequestTimeout time.Duration (default: 30s)
  - [ ] Defaults(), Clone() methods

- [ ] Define TLSConfig struct
  - [ ] CAFile, CertFile, KeyFile string
  - [ ] ServerName string
  - [ ] InsecureSkipVerify bool
  - [ ] Clone() method

#### Acceptance Criteria

- ✅ 3 structs defined (GlobalConfig, HTTPConfig, TLSConfig)
- ✅ All methods implemented
- ✅ Defaults applied correctly
- ✅ 100% godoc coverage

### 3.4: Write Unit Tests (1h)

**File**: `go-app/internal/infrastructure/routing/config_test.go` (400 LOC)

#### Tasks

- [ ] Test RouteConfig
  - [ ] TestRouteConfigUnmarshalYAML
  - [ ] TestRouteConfigGetReceiver
  - [ ] TestRouteConfigClone
  - [ ] TestRouteConfigValidate

- [ ] Test Receiver
  - [ ] TestReceiverValidate
  - [ ] TestReceiverClone
  - [ ] TestReceiverSanitize
  - [ ] TestReceiverGetConfigCount

- [ ] Test WebhookConfig
  - [ ] TestWebhookConfigDefaults
  - [ ] TestWebhookConfigClone
  - [ ] TestWebhookConfigSanitize

- [ ] Test PagerDutyConfig (3 tests)
- [ ] Test SlackConfig (3 tests)
- [ ] Test GlobalConfig (2 tests)
- [ ] Test HTTPConfig (2 tests)
- [ ] Test TLSConfig (1 test)

#### Acceptance Criteria

- ✅ 20+ tests written
- ✅ 100% test pass rate
- ✅ 80%+ code coverage (models)
- ✅ Zero race conditions (-race flag)

---

## Phase 4: Extended Parser Implementation

**Duration**: 8 hours
**Status**: ⏸️ PENDING

### 4.1: Parser Core (3h)

**File**: `go-app/internal/infrastructure/routing/parser.go` (600 LOC)

#### Tasks

- [ ] Define RouteConfigParser struct
  ```go
  type RouteConfigParser struct {
      validator *validator.Validate
      errors    ValidationErrors
  }
  ```

- [ ] Implement NewRouteConfigParser()
  - [ ] Initialize validator/v10
  - [ ] Register custom validators:
    - [ ] alphanum_hyphen (receiver names)
    - [ ] https_production (webhook URLs)
    - [ ] slack_channel (format: #channel or @user)
    - [ ] emoji (format: :emoji:)
    - [ ] slack_color (good|warning|danger|#hex)

- [ ] Implement Parse(data []byte) (*RouteConfig, error)
  - [ ] Step 1: YAML unmarshaling (yaml.v3)
  - [ ] Step 2: Validate required fields (route, receivers)
  - [ ] Step 3: Apply defaults recursively
  - [ ] Step 4: Structural validation (validator tags)
  - [ ] Step 5: Semantic validation (custom rules)
  - [ ] Step 6: Compile regex patterns
  - [ ] Step 7: Build receiver index
  - [ ] Step 8: Set metadata (LoadedAt, Version)

- [ ] Implement ParseFile(path string) (*RouteConfig, error)
  - [ ] Check file size ≤ 10 MB (YAML bomb protection)
  - [ ] Read file
  - [ ] Call Parse(data)
  - [ ] Set SourceFile metadata

- [ ] Implement ParseString(yaml string) (*RouteConfig, error)

- [ ] Implement applyDefaults(config *RouteConfig)
  - [ ] Apply global defaults
  - [ ] Recursively apply route defaults (via TN-121)
  - [ ] Apply receiver defaults (loop через receivers)

#### Acceptance Criteria

- ✅ RouteConfigParser struct defined
- ✅ 3 Parse methods implemented (file, bytes, string)
- ✅ Defaults applied correctly (recursive)
- ✅ YAML bomb protection (10 MB limit)
- ✅ Zero compilation errors

### 4.2: Semantic Validation (3h)

**File**: `go-app/internal/infrastructure/routing/parser_validate.go` (400 LOC)

#### Tasks

- [ ] Implement validateSemantics(config *RouteConfig) error
  - [ ] Build receiver index (name → receiver)
  - [ ] Validate route tree recursively
  - [ ] Validate receivers
  - [ ] Check unused receivers (warning)
  - [ ] Validate global config

- [ ] Implement validateRouteTree()
  - [ ] Cycle detection (visited map)
  - [ ] Receiver reference check (exists in index)
  - [ ] Label name validation (Prometheus syntax)
  - [ ] Timer range validation (group_wait, group_interval, repeat_interval)
  - [ ] Match/MatchRE label key validation
  - [ ] Nesting depth ≤ 10 levels

- [ ] Implement validateReceivers(receivers []*Receiver)
  - [ ] Loop через receivers
  - [ ] Call receiver.Validate() (at least one config)
  - [ ] Validate webhook configs
  - [ ] Validate PagerDuty configs
  - [ ] Validate Slack configs

- [ ] Implement validateWebhookConfig(cfg, path)
  - [ ] HTTPS validation (production mode)
  - [ ] SSRF protection (no private IPs)
  - [ ] Sensitive header validation (use secret refs)

- [ ] Implement validatePagerDutyConfig(cfg, path)
  - [ ] RoutingKey length (32 chars)
  - [ ] URL HTTPS validation
  - [ ] Severity enum validation

- [ ] Implement validateSlackConfig(cfg, path)
  - [ ] APIURL HTTPS validation
  - [ ] Channel format (#channel or @user)
  - [ ] Icon emoji/URL mutual exclusivity

- [ ] Implement checkUnusedReceivers()
  - [ ] Mark receivers as Referenced during validation
  - [ ] Warn if receiver.Referenced == false

#### Acceptance Criteria

- ✅ 4-layer validation implemented
- ✅ Cycle detection working
- ✅ Receiver reference validation
- ✅ Label name validation
- ✅ Clear error messages (field path + suggestion)

### 4.3: Security Validation (2h)

**File**: `go-app/internal/infrastructure/routing/parser_security.go` (200 LOC)

#### Tasks

- [ ] Implement validateURLNotPrivate(urlStr string) error
  - [ ] Parse URL
  - [ ] Resolve hostname to IPs (net.LookupIP)
  - [ ] Check if any IP is private (RFC 1918, localhost, link-local)
  - [ ] Return error if private IP found

- [ ] Implement isPrivateIP(ip net.IP) bool
  - [ ] Define private ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, 169.254.0.0/16)
  - [ ] IPv6 ranges (::1/128, fc00::/7, fe80::/10)
  - [ ] Check if IP in any range (net.ParseCIDR, network.Contains)

- [ ] Implement isSensitiveHeader(key string) bool
  - [ ] Define sensitive keywords (authorization, api-key, token, bearer, password, secret)
  - [ ] Case-insensitive contains check

- [ ] Implement isSecretReference(value string) bool
  - [ ] Check environment variable format: ${VAR_NAME}
  - [ ] Check K8s Secret format: secret:namespace/name/key

- [ ] Implement sanitizeURL(urlStr string) string
  - [ ] Parse URL
  - [ ] Redact query parameters (may contain API keys)
  - [ ] Keep scheme, host, path
  - [ ] Return sanitized URL

- [ ] Implement sanitizeWebhookURL(urlStr string) string
  - [ ] Similar to sanitizeURL
  - [ ] Redact sensitive path segments (tokens in path)

#### Acceptance Criteria

- ✅ SSRF protection implemented
- ✅ Private IP detection (IPv4 + IPv6)
- ✅ Sensitive header detection
- ✅ Secret sanitization
- ✅ URL sanitization

---

## Phase 5: Comprehensive Testing

**Duration**: 10 hours
**Status**: ⏸️ PENDING

### 5.1: Parser Tests (4h)

**File**: `go-app/internal/infrastructure/routing/parser_test.go` (600 LOC)

#### Tasks

- [ ] Test Parse() method (12 tests)
  - [ ] TestParseValidConfig (minimal, production, complex)
  - [ ] TestParseInvalidYAML (syntax errors)
  - [ ] TestParseMissingRoute (required field)
  - [ ] TestParseMissingReceivers (required field)
  - [ ] TestParseUnknownReceiverReference (semantic error)
  - [ ] TestParseDuplicateReceiver (semantic error)
  - [ ] TestParseCyclicRoute (cycle detection)
  - [ ] TestParseExcessiveNesting (depth > 10)
  - [ ] TestParseYAMLBomb (file > 10 MB)
  - [ ] TestParseWithDefaults (defaults applied)
  - [ ] TestParseRegexCompilation (valid + invalid)
  - [ ] TestParseReceiverIndex (O(1) lookup)

- [ ] Test ParseFile() method (3 tests)
  - [ ] TestParseFileSuccess
  - [ ] TestParseFileNotFound
  - [ ] TestParseFileTooBig (YAML bomb)

- [ ] Test ParseString() method (1 test)
  - [ ] TestParseStringSuccess

#### Acceptance Criteria

- ✅ 16 parser tests written
- ✅ 100% test pass rate
- ✅ All error paths covered
- ✅ Edge cases tested

### 5.2: Validation Tests (3h)

**File**: `go-app/internal/infrastructure/routing/validation_test.go` (350 LOC)

#### Tasks

- [ ] Test semantic validation (10 tests)
  - [ ] TestValidateReceiverReferences (exists check)
  - [ ] TestValidateCycleDetection (cycle algorithm)
  - [ ] TestValidateSSRFProtection (private IPs)
  - [ ] TestValidateSensitiveHeaders (secret refs)
  - [ ] TestValidateRegexPatterns (compile errors)
  - [ ] TestValidateUnusedReceivers (warning)
  - [ ] TestValidateInheritance (child → parent)
  - [ ] TestValidateTimerRanges (min/max)
  - [ ] TestValidateMatcherCombinations (Match + MatchRE)
  - [ ] TestValidateReceiverTypes (at least one config)

#### Acceptance Criteria

- ✅ 10 validation tests written
- ✅ All validation layers covered
- ✅ Error messages checked (clarity)
- ✅ Warning messages checked

### 5.3: Integration Tests (2h)

**File**: `go-app/internal/infrastructure/routing/parser_integration_test.go` (NEW, 300 LOC)

#### Tasks

- [ ] Test end-to-end parsing (10 tests)
  - [ ] TestIntegrationMinimalConfig
  - [ ] TestIntegrationProductionConfig
  - [ ] TestIntegrationComplexNestedRoutes
  - [ ] TestIntegrationMultiReceiverContinue
  - [ ] TestIntegrationAllReceiverTypes
  - [ ] TestIntegrationAlertmanagerExample1
  - [ ] TestIntegrationAlertmanagerExample2
  - [ ] TestIntegrationWithTN121Compatibility
  - [ ] TestIntegrationReceiverIndexLookup
  - [ ] TestIntegrationCompiledRegexUsage

#### Acceptance Criteria

- ✅ 10 integration tests written
- ✅ Use real YAML fixtures (testdata/)
- ✅ Test TN-121 compatibility
- ✅ Test receiver index correctness

### 5.4: Test Fixtures (1h)

**Directory**: `go-app/internal/infrastructure/routing/testdata/`

#### Tasks

- [ ] Create YAML fixtures
  - [ ] minimal.yaml (20 LOC) — simplest valid config
  - [ ] production.yaml (150 LOC) — realistic production config
  - [ ] complex.yaml (200 LOC) — nested routes, all receiver types
  - [ ] invalid_yaml_syntax.yaml — syntax errors
  - [ ] invalid_missing_route.yaml — required field missing
  - [ ] invalid_missing_receivers.yaml — no receivers
  - [ ] invalid_unknown_receiver.yaml — route references unknown receiver
  - [ ] invalid_duplicate_receiver.yaml — duplicate receiver names
  - [ ] invalid_cyclic_route.yaml — route cycle
  - [ ] invalid_excessive_nesting.yaml — depth > 10
  - [ ] alertmanager_example1.yaml — from official docs
  - [ ] alertmanager_example2.yaml — from official docs

#### Acceptance Criteria

- ✅ 12 YAML fixtures created
- ✅ Valid configs (3) parse successfully
- ✅ Invalid configs (7) fail with expected errors
- ✅ Alertmanager examples (2) parse successfully

---

## Phase 6: Performance Optimization

**Duration**: 4 hours
**Status**: ⏸️ PENDING

### 6.1: Benchmarks (2h)

**File**: `go-app/internal/infrastructure/routing/parser_bench_test.go` (250 LOC)

#### Tasks

- [ ] Create benchmark functions (10 benchmarks)
  - [ ] BenchmarkParseSmallConfig (10 routes, 5 receivers)
  - [ ] BenchmarkParseMediumConfig (100 routes, 50 receivers)
  - [ ] BenchmarkParseLargeConfig (1000 routes, 500 receivers)
  - [ ] BenchmarkValidateReceivers (1000 receivers)
  - [ ] BenchmarkDetectCycles (complex tree)
  - [ ] BenchmarkBuildReceiverIndex (5000 receivers)
  - [ ] BenchmarkApplyDefaults (deep route tree)
  - [ ] BenchmarkSanitizeConfig (large config with secrets)
  - [ ] BenchmarkCompileRegex (100 patterns)
  - [ ] BenchmarkCloneConfig (large config)

- [ ] Run benchmarks
  ```bash
  go test -bench=. -benchmem ./internal/infrastructure/routing/
  ```

- [ ] Record baseline results
  - [ ] Document in BENCHMARKS.md
  - [ ] Compare with targets (200%+ better)

#### Acceptance Criteria

- ✅ 10 benchmarks written
- ✅ All benchmarks run successfully
- ✅ Baseline results documented
- ✅ No memory allocations in hot paths

### 6.2: Optimization (2h)

#### Tasks

- [ ] Identify bottlenecks (via pprof)
  ```bash
  go test -bench=BenchmarkParseLargeConfig -cpuprofile=cpu.prof
  go tool pprof cpu.prof
  ```

- [ ] Optimize hot paths
  - [ ] Reduce allocations (use sync.Pool for temporary objects)
  - [ ] Pre-allocate slices (cap=len when known)
  - [ ] Cache regex compilation (already done)
  - [ ] Optimize receiver index build (single pass)

- [ ] Memory profiling
  ```bash
  go test -bench=. -memprofile=mem.prof
  go tool pprof mem.prof
  ```

- [ ] Verify improvements
  - [ ] Re-run benchmarks
  - [ ] Confirm 200%+ better than targets
  - [ ] Update BENCHMARKS.md

#### Acceptance Criteria

- ✅ Bottlenecks identified and fixed
- ✅ Memory allocations minimized
- ✅ Benchmarks exceed targets by 200%+
- ✅ No performance regressions

---

## Phase 7: Security Hardening

**Duration**: 3 hours
**Status**: ⏸️ PENDING

### 7.1: Security Validation (1h)

#### Tasks

- [ ] Implement YAML bomb protection
  - [ ] File size limit: 10 MB
  - [ ] Route nesting depth limit: 10 levels
  - [ ] Max routes: 10,000
  - [ ] Max receivers: 5,000
  - [ ] Max matchers per route: 100

- [ ] Implement SSRF protection
  - [ ] Private IP checks (implemented in Phase 4.3)
  - [ ] DNS validation (no localhost, link-local)
  - [ ] Optional allowlist/blocklist (FUTURE)

- [ ] Implement secret sanitization
  - [ ] Never log sensitive headers
  - [ ] Redact secrets in API responses (Sanitize() methods)
  - [ ] Mask webhook URLs in logs

#### Acceptance Criteria

- ✅ YAML bomb protection active
- ✅ SSRF protection working
- ✅ Secrets sanitized in logs/API

### 7.2: Security Tests (1h)

**File**: `go-app/internal/infrastructure/routing/security_test.go` (NEW, 200 LOC)

#### Tasks

- [ ] Test YAML bomb protection (3 tests)
  - [ ] TestYAMLBombFileSize (> 10 MB)
  - [ ] TestYAMLBombNestingDepth (> 10 levels)
  - [ ] TestYAMLBombMaxRoutes (> 10,000)

- [ ] Test SSRF protection (5 tests)
  - [ ] TestSSRFPrivateIPv4 (10.0.0.0/8, 192.168.0.0/16)
  - [ ] TestSSRFLocalhost (127.0.0.1, ::1)
  - [ ] TestSSRFLinkLocal (169.254.0.0/16)
  - [ ] TestSSRFValidPublicIP (allowed)
  - [ ] TestSSRFDNSValidation

- [ ] Test secret sanitization (3 tests)
  - [ ] TestSanitizeWebhookConfig
  - [ ] TestSanitizePagerDutyConfig
  - [ ] TestSanitizeSlackConfig

#### Acceptance Criteria

- ✅ 11 security tests written
- ✅ All tests pass
- ✅ Security vulnerabilities tested

### 7.3: Security Scan (1h)

#### Tasks

- [ ] Run gosec security scanner
  ```bash
  gosec -fmt=json -out=gosec-report.json ./internal/infrastructure/routing/
  ```

- [ ] Review findings
  - [ ] Fix all HIGH severity issues
  - [ ] Fix all MEDIUM severity issues
  - [ ] Document LOW severity issues (false positives)

- [ ] Run nancy dependency checker
  ```bash
  go list -json -m all | nancy sleuth
  ```

- [ ] Update dependencies if vulnerabilities found

- [ ] Run govulncheck
  ```bash
  govulncheck ./...
  ```

#### Acceptance Criteria

- ✅ gosec scan clean (zero HIGH/MEDIUM issues)
- ✅ nancy scan clean (zero vulnerabilities)
- ✅ govulncheck clean
- ✅ All dependencies up-to-date

---

## Phase 8: Observability Integration

**Duration**: 2 hours
**Status**: ⏸️ PENDING

### 8.1: Prometheus Metrics (1h)

**File**: `go-app/internal/infrastructure/routing/metrics.go` (NEW, 150 LOC)

#### Tasks

- [ ] Define metrics
  ```go
  var (
      // Parsing metrics
      routingConfigParseDuration = prometheus.NewHistogramVec(
          prometheus.HistogramOpts{
              Namespace: "alert_history",
              Subsystem: "routing",
              Name:      "config_parse_duration_seconds",
              Help:      "Time spent parsing routing configuration",
              Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
          },
          []string{"operation"}, // parse, validate, compile
      )

      // Validation metrics
      routingConfigValidationErrors = prometheus.NewCounterVec(
          prometheus.CounterOpts{
              Namespace: "alert_history",
              Subsystem: "routing",
              Name:      "config_validation_errors_total",
              Help:      "Total routing config validation errors",
          },
          []string{"error_type"}, // yaml, structural, semantic, cross_ref
      )

      // Hot reload metrics (FUTURE - TN-152)
      routingConfigHotReload = prometheus.NewCounterVec(
          prometheus.CounterOpts{
              Namespace: "alert_history",
              Subsystem: "routing",
              Name:      "config_hot_reload_total",
              Help:      "Total hot reload attempts",
          },
          []string{"status"}, // success, failure
      )
  )
  ```

- [ ] Register metrics
  - [ ] func init() — register with prometheus.MustRegister()
  - [ ] Add to MetricsRegistry (if exists)

- [ ] Instrument parser
  - [ ] Start timer at Parse() entry
  - [ ] Record duration for parse, validate, compile operations
  - [ ] Increment error counter on validation failures

#### Acceptance Criteria

- ✅ 3 Prometheus metrics defined
- ✅ Metrics registered on init
- ✅ Parser instrumented (duration + errors)
- ✅ Labels meaningful (operation, error_type)

### 8.2: Structured Logging (1h)

#### Tasks

- [ ] Add logging to parser
  ```go
  slog.Info("parsing config",
      "source", path,
      "size_bytes", len(data),
      "routes", countRoutes(config.Route),
      "receivers", len(config.Receivers),
  )
  ```

- [ ] Log validation errors
  ```go
  slog.Error("validation failed",
      "error_count", len(errors),
      "error_type", "semantic",
      "field", errors[0].Field,
      "message", errors[0].Message,
  )
  ```

- [ ] Log performance warnings
  ```go
  if duration > 100*time.Millisecond {
      slog.Warn("slow config parse",
          "duration_ms", duration.Milliseconds(),
          "routes", countRoutes(config.Route),
          "threshold_ms", 100,
      )
  }
  ```

- [ ] Log security warnings
  ```go
  if config.Global.HTTPConfig.TLSConfig.InsecureSkipVerify {
      slog.Warn("TLS verification disabled",
          "field", "global.http_config.tls_config.insecure_skip_verify",
          "security_risk", "HIGH",
      )
  }
  ```

#### Acceptance Criteria

- ✅ Info logs for parse start/end
- ✅ Error logs for validation failures
- ✅ Warn logs for performance/security issues
- ✅ Debug logs for detailed flow (optional)

---

## Phase 9: Final Certification

**Duration**: 3 hours
**Status**: ⏸️ PENDING

### 9.1: Quality Assessment (1h)

#### Tasks

- [ ] Run full test suite
  ```bash
  go test -v -race -coverprofile=coverage.out ./internal/infrastructure/routing/
  go tool cover -func=coverage.out
  ```

- [ ] Verify test coverage ≥ 90% (target for 150%)
  - [ ] config.go: 90%+
  - [ ] receiver.go: 90%+
  - [ ] parser.go: 90%+
  - [ ] parser_validate.go: 90%+
  - [ ] parser_security.go: 90%+

- [ ] Run benchmarks and verify targets
  ```bash
  go test -bench=. -benchmem ./internal/infrastructure/routing/ | tee benchmarks.txt
  ```
  - [ ] Parse small config: < 5ms (target: < 10ms) = 200% ✅
  - [ ] Parse medium config: < 50ms (target: < 100ms) = 200% ✅
  - [ ] Parse large config: < 500ms (target: < 1s) = 200% ✅

- [ ] Run linters
  ```bash
  golangci-lint run ./internal/infrastructure/routing/
  ```
  - [ ] Zero errors
  - [ ] Zero warnings

- [ ] Run security scans
  ```bash
  gosec ./internal/infrastructure/routing/
  nancy sleuth < go.list
  govulncheck ./...
  ```
  - [ ] Zero vulnerabilities

#### Acceptance Criteria

- ✅ Test coverage ≥ 90%
- ✅ All benchmarks exceed targets by 200%+
- ✅ Zero linter errors/warnings
- ✅ Zero security vulnerabilities

### 9.2: Documentation Completion (1h)

**File**: `go-app/internal/infrastructure/routing/README.md` (500 LOC)

#### Tasks

- [ ] Write README sections
  - [ ] Overview (50 LOC)
  - [ ] Quick Start (100 LOC)
  - [ ] Configuration Examples (150 LOC)
  - [ ] API Reference (100 LOC)
  - [ ] Troubleshooting (50 LOC)
  - [ ] Performance Tips (50 LOC)

- [ ] Add code examples
  - [ ] Basic usage
  - [ ] Advanced usage (all receiver types)
  - [ ] Error handling
  - [ ] Hot reload (FUTURE - TN-152)

- [ ] Add troubleshooting guide
  - [ ] Common errors (unknown receiver, cycle, etc.)
  - [ ] Performance issues
  - [ ] Security warnings

#### Acceptance Criteria

- ✅ README.md written (500+ LOC)
- ✅ All sections complete
- ✅ Code examples tested
- ✅ Troubleshooting guide comprehensive

### 9.3: Certification Report (1h)

**File**: `tasks/go-migration-analysis/TN-137-route-config-parser/CERTIFICATION.md` (NEW, 500 LOC)

#### Tasks

- [ ] Write certification report sections
  - [ ] Executive Summary (50 LOC)
  - [ ] Quality Metrics (100 LOC)
    - [ ] Test coverage: X% (target: 90%)
    - [ ] Test count: X (target: 35+)
    - [ ] Benchmark results: X (target: 200%+ better)
    - [ ] Documentation: X LOC (target: 3,000+)
  - [ ] Functional Requirements (100 LOC)
    - [ ] FR-1: Route configuration ✅
    - [ ] FR-2: Receiver configuration ✅
    - [ ] FR-3: Global configuration ✅
    - [ ] FR-4: 4-layer validation ✅
    - [ ] FR-5: Configuration loading ✅
  - [ ] Non-Functional Requirements (100 LOC)
    - [ ] NFR-1: Performance ✅ (200%+ better)
    - [ ] NFR-2: Reliability ✅ (zero crashes)
    - [ ] NFR-3: Security ✅ (zero vulnerabilities)
    - [ ] NFR-4: Observability ✅ (3 metrics)
    - [ ] NFR-5: Testability ✅ (35+ tests, 90%+ coverage)
    - [ ] NFR-6: Maintainability ✅ (100% godoc)
    - [ ] NFR-7: Compatibility ✅ (Alertmanager v0.27+)
  - [ ] 150% Achievement Evidence (150 LOC)
    - [ ] Implementation: 120% (extra features)
    - [ ] Testing: 120% (35 vs 30 tests)
    - [ ] Documentation: 140% (3,900 vs 2,800 LOC)
    - [ ] Performance: 200%+ (benchmarks)

- [ ] Calculate final grade
  - [ ] Functional: X/100
  - [ ] Non-Functional: X/150
  - [ ] Total: X/250
  - [ ] Grade: A+ (if X ≥ 225/250 = 90%)

- [ ] Sign-off
  - [ ] Technical Lead: ✅
  - [ ] Security Team: ✅
  - [ ] QA Team: ✅
  - [ ] Architecture Team: ✅

#### Acceptance Criteria

- ✅ CERTIFICATION.md written (500+ LOC)
- ✅ All metrics documented
- ✅ 150% achievement verified
- ✅ Grade A+ achieved (≥ 90%)

---

## Deliverables Summary

### Phase 0-1: Documentation (✅ COMPLETE)

| File | LOC | Status |
|------|-----|--------|
| COMPREHENSIVE_ANALYSIS.md | 1,000+ | ✅ COMPLETE |
| requirements.md | 700+ | ✅ COMPLETE |
| design.md | 1,200+ | ✅ COMPLETE |
| tasks.md (this file) | 900+ | ✅ COMPLETE |
| **TOTAL** | **3,800+** | **✅ COMPLETE** |

### Phase 2-8: Implementation (⏸️ PENDING)

| File | LOC | Tests | Status |
|------|-----|-------|--------|
| config.go | 500 | 8 | ⏸️ PENDING |
| receiver.go | 400 | 12 | ⏸️ PENDING |
| global.go | 200 | 5 | ⏸️ PENDING |
| parser.go | 600 | 16 | ⏸️ PENDING |
| parser_validate.go | 400 | 10 | ⏸️ PENDING |
| parser_security.go | 200 | 11 | ⏸️ PENDING |
| errors.go | 150 | - | ⏸️ PENDING |
| utils.go | 100 | - | ⏸️ PENDING |
| metrics.go | 150 | - | ⏸️ PENDING |
| **Production Code** | **2,700** | **-** | **⏸️ PENDING** |
| config_test.go | 400 | - | ⏸️ PENDING |
| parser_test.go | 600 | - | ⏸️ PENDING |
| validation_test.go | 350 | - | ⏸️ PENDING |
| parser_integration_test.go | 300 | - | ⏸️ PENDING |
| security_test.go | 200 | - | ⏸️ PENDING |
| parser_bench_test.go | 250 | - | ⏸️ PENDING |
| **Test Code** | **2,100** | **-** | **⏸️ PENDING** |
| README.md | 500 | - | ⏸️ PENDING |
| CERTIFICATION.md | 500 | - | ⏸️ PENDING |
| **Total LOC** | **5,800** | **62** | **⏸️ PENDING** |

### Grand Total

| Category | LOC | Status |
|----------|-----|--------|
| Documentation (Phase 0-1) | 3,800 | ✅ COMPLETE |
| Production Code | 2,700 | ⏸️ PENDING |
| Test Code | 2,100 | ⏸️ PENDING |
| Additional Docs | 1,000 | ⏸️ PENDING |
| **TOTAL** | **9,600+** | **5% COMPLETE** |

---

## Quality Checklist (150%)

### Functional Requirements (100%)

- [ ] FR-1: Route configuration (nested routes, Match/MatchRE, Continue)
- [ ] FR-2: Receiver configuration (webhook, PagerDuty, Slack)
- [ ] FR-3: Global configuration (resolve_timeout, HTTP config)
- [ ] FR-4: 4-layer validation (YAML → structural → semantic → cross-ref)
- [ ] FR-5: Configuration loading (file/bytes/string, defaults, index)

### Non-Functional Requirements (150%)

- [ ] NFR-1: Performance (200%+ better than targets)
  - [ ] Small config: < 5ms (target: < 10ms)
  - [ ] Medium config: < 50ms (target: < 100ms)
  - [ ] Large config: < 500ms (target: < 1s)

- [ ] NFR-2: Reliability (zero crashes, clear errors)
  - [ ] 100% error handling coverage
  - [ ] No panics on fuzzing
  - [ ] Clear error messages with suggestions

- [ ] NFR-3: Security (YAML bombs, SSRF, secrets)
  - [ ] gosec scan clean
  - [ ] YAML bomb protection active
  - [ ] SSRF protection working

- [ ] NFR-4: Observability (3 metrics, structured logging)
  - [ ] 3 Prometheus metrics implemented
  - [ ] Structured logging (slog)
  - [ ] Performance tracking

- [ ] NFR-5: Testability (35+ tests, 90%+ coverage)
  - [ ] 35+ unit tests (target: 30+) = 117%
  - [ ] 12+ integration tests (target: 10+) = 120%
  - [ ] 10+ benchmarks (target: 8+) = 125%
  - [ ] 90%+ coverage (target: 85%+) = 106%

- [ ] NFR-6: Maintainability (100% godoc, 3,000+ LOC docs)
  - [ ] Zero linter warnings
  - [ ] 100% godoc coverage
  - [ ] 3,900+ LOC docs (target: 2,800) = 139%

- [ ] NFR-7: Compatibility (Alertmanager v0.27+, backward compat)
  - [ ] Parse Alertmanager examples
  - [ ] TN-121 compatibility maintained

---

## Commit Strategy

### Commit Messages

```
feat(TN-137): [component] [description]

Examples:
- feat(TN-137): Initialize Route Config Parser structure
- feat(TN-137): Implement RouteConfig and Receiver models
- feat(TN-137): Implement parser core with 4-layer validation
- feat(TN-137): Add comprehensive test suite (35+ tests)
- feat(TN-137): Add benchmarks and performance optimization
- feat(TN-137): Add security validation (YAML bombs, SSRF)
- feat(TN-137): Add Prometheus metrics and structured logging
- docs(TN-137): Add comprehensive README and certification
- test(TN-137): Add security tests (YAML bomb, SSRF)
```

### Commit Frequency

- After each sub-phase (3-4 commits per phase)
- After fixing compilation errors
- After all tests pass
- Before requesting review

---

## Definition of Done

### Code

- [ ] All FR/NFR requirements implemented
- [ ] Zero compilation errors
- [ ] Zero linter warnings (golangci-lint)
- [ ] Zero security issues (gosec)
- [ ] 100% godoc coverage

### Testing

- [ ] 35+ unit tests (100% passing)
- [ ] 12+ integration tests (100% passing)
- [ ] 10+ benchmarks (all exceed targets)
- [ ] 90%+ test coverage
- [ ] Zero flaky tests
- [ ] Zero race conditions (-race flag)

### Documentation

- [ ] requirements.md (700+ LOC) ✅
- [ ] design.md (1,200+ LOC) ✅
- [ ] tasks.md (900+ LOC) ✅
- [ ] README.md (500+ LOC)
- [ ] CERTIFICATION.md (500+ LOC)

### Quality

- [ ] 150% quality checklist verified
- [ ] Grade A+ certification (≥ 90%)
- [ ] Production readiness review
- [ ] Peer review approved (optional)

### Deployment

- [ ] Merged to main branch
- [ ] CI/CD pipeline green
- [ ] Documentation updated (tasks.md)
- [ ] CHANGELOG entry added

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| **Breaking TN-121** | Extend, don't replace. Keep GroupingConfig working |
| **Performance regression** | Benchmark every change, target 200%+ margin |
| **Incomplete validation** | 4 layers of validation, comprehensive tests |
| **YAML bombs** | Size (10MB), depth (10), route (10K) limits |
| **SSRF** | Private IP checks, DNS validation |
| **Test coverage gaps** | 90%+ target, edge cases in fixtures |
| **Integration failures** | Mock dependencies, integration tests |
| **Time overrun** | Agile approach, deliver MVPs per phase |

---

## Next Steps

1. **Immediate** (Phase 2): Create Git branch and directory structure (15min)
2. **Short-term** (Phases 3-5): Implement models, parser, tests (24h)
3. **Medium-term** (Phases 6-8): Optimize, harden, observe (9h)
4. **Long-term** (Phase 9): Certify and document (3h)

**Total Estimated Effort**: 42-45 hours (6-7 days)

**Target Completion**: 2025-11-24 (1 week)

**Quality Target**: Grade A+ (150%+ achievement)

---

**End of Tasks Specification**

**Status**: Phase 0-1 COMPLETE (5%), Phase 2-9 PENDING (95%)

**Next Action**: Execute Phase 2 (Git Branch Setup)
