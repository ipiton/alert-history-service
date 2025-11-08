# TN-047: Target Discovery Manager - Requirements

**Module**: PHASE 5 - Publishing System
**Task ID**: TN-047
**Status**: 🟡 IN PROGRESS
**Priority**: HIGH (blocks TN-048, TN-049, TN-51-60)
**Estimated Effort**: 8-10 hours
**Dependencies**: ✅ TN-046 (K8s Client - COMPLETE)
**Blocks**: TN-048 (Refresh Mechanism), TN-049 (Health Monitoring), All Publishing Tasks
**Target Quality**: 150% (Enterprise-Grade, following TN-046 standards)
**Quality Reference**: TN-046 (150%+), TN-134 (150%+), TN-136 (150%)

---

## 📋 Executive Summary

Реализовать **enterprise-grade Target Discovery Manager** для автоматического обнаружения publishing targets из Kubernetes Secrets с поддержкой:
- **Dynamic Discovery**: Автоматическое обнаружение targets через K8s label selectors
- **Secret Parsing**: Парсинг base64-encoded secret data в PublishingTarget structures
- **Validation**: Comprehensive валидация configuration с detailed error messages
- **In-Memory Cache**: Fast O(1) lookups для active targets (~50-100ns)
- **Thread-Safe Operations**: Concurrent-safe access к target registry
- **Observability**: 6 Prometheus metrics, structured logging, health checks

### Business Value

| Ценность | Описание | Impact |
|----------|----------|--------|
| **Zero Downtime Updates** | Dynamic discovery без перезапуска приложения | HIGH |
| **GitOps Ready** | Targets управляются через K8s manifests (CI/CD integration) | HIGH |
| **Security** | Credentials в K8s Secrets (encrypted at rest, RBAC) | HIGH |
| **Multi-Tenancy** | Support для разных environments через namespaces | MEDIUM |
| **Fast Access** | In-memory cache для publishing hot path (<100ns) | HIGH |

---

## 1. Обоснование задачи

### 1.1 Проблема

Publishing System требует конфигурации external targets (Rootly, PagerDuty, Slack, Webhooks). Текущие проблемы:
1. **Static Configuration**: Targets hardcoded в config файлах → требуется restart для изменений
2. **Secret Management**: Credentials в environment variables → security risk
3. **No Discovery**: Manual configuration → prone to errors
4. **No Validation**: Malformed configs cause runtime errors → poor UX

### 1.2 Решение

Target Discovery Manager обеспечивает:
- **Kubernetes Secrets Integration**: Native K8s secret management с encryption at rest
- **Label-Based Discovery**: Flexible filtering через label selectors (`publishing-target=true`)
- **Automatic Parsing**: Base64 decoding + JSON parsing + validation
- **Fail-Safe Design**: Graceful degradation при errors
- **Real-Time Updates**: Foundation для TN-048 (refresh mechanism)

### 1.3 Блокирует

- **TN-048**: Target Refresh Mechanism (periodic + manual)
- **TN-049**: Target Health Monitoring
- **TN-051**: Alert Formatter
- **TN-052-055**: All Publishers (Rootly, PagerDuty, Slack, Generic)
- **TN-056**: Publishing Queue
- **TN-057-059**: Publishing Metrics, Parallel Publishing, API

**CRITICAL**: Без TN-047 вся PHASE 5 (Publishing System) заблокирована.

---

## 2. Пользовательский сценарий

### Scenario 1: Initial Discovery (Service Startup)

```
1. Target Discovery Manager инициализируется
2. Calls K8sClient.ListSecrets(namespace, "publishing-target=true")
3. Parses каждый secret в PublishingTarget
4. Validates configuration (required fields, URL format)
5. Stores valid targets в in-memory cache
6. Logs statistics (N targets discovered, M invalid)
7. Ready для publishing operations
```

**Expected Time**: <2s для 20 secrets (NFR-1)

### Scenario 2: Get Target for Publishing

```
1. Publishing pipeline requests target: GetTarget("rootly-prod")
2. Discovery Manager lookup в in-memory cache (O(1))
3. Returns PublishingTarget или NotFoundError
4. Publishing proceeds with target configuration
```

**Expected Time**: <100ns (in-memory lookup)

### Scenario 3: List Targets by Type

```
1. API endpoint requests: GetTargetsByType("slack")
2. Discovery Manager filters cached targets by type
3. Returns slice of matching PublishingTarget
4. API returns JSON response
```

**Expected Time**: <1µs для 20 targets

### Scenario 4: Invalid Secret Handling

```
1. Secret "invalid-target" has malformed JSON in data field
2. Discovery Manager attempts parsing
3. Validation fails with detailed error message
4. Error logged with secret name + validation errors
5. Invalid target skipped (не блокирует discovery)
6. Valid targets remain operational (graceful degradation)
```

**Expected Behavior**: No crash, detailed logs, partial success

---

## 3. Функциональные требования

### FR-1: Target Discovery (CRITICAL)

**Description**: List и parse Kubernetes Secrets matching label selector.

**Input**:
- `namespace`: K8s namespace (e.g., "production", "staging")
- `labelSelector`: Label query (e.g., "publishing-target=true,environment=prod")

**Output**:
- `[]*PublishingTarget`: Slice of parsed targets
- `error`: Discovery error (if K8s API unavailable)

**Behavior**:
- Call `k8sClient.ListSecrets(ctx, namespace, labelSelector)`
- Iterate через secrets и parse каждый
- Validate parsed targets (FR-3)
- Store valid targets в cache (FR-4)
- Log invalid targets (no crash)

**Error Handling**:
- K8s API unavailable → return error, keep old cache
- Some secrets invalid → log warnings, continue with valid
- Zero secrets found → empty cache, no error

**Acceptance Criteria**:
- [ ] Calls K8sClient.ListSecrets with correct parameters
- [ ] Parses secret.Data["config"] (base64 + JSON)
- [ ] Validates each target (FR-3)
- [ ] Stores valid targets в cache
- [ ] Logs statistics (discovered/invalid counts)

---

### FR-2: Secret Parsing (CRITICAL)

**Description**: Parse K8s Secret data в PublishingTarget structure.

**Secret Format** (YAML example):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-prod
  namespace: default
  labels:
    publishing-target: "true"
    environment: prod
type: Opaque
data:
  config: <base64-encoded JSON>
```

**Config JSON Structure**:
```json
{
  "name": "rootly-prod",
  "type": "rootly",
  "url": "https://api.rootly.io/v1/incidents",
  "format": "rootly",
  "enabled": true,
  "headers": {
    "Authorization": "Bearer secret-token-xxx"
  },
  "filter_config": {
    "min_severity": "warning"
  }
}
```

**Parsing Steps**:
1. Extract `secret.Data["config"]` ([]byte, base64-encoded)
2. Base64 decode → JSON string
3. JSON unmarshal → PublishingTarget struct
4. Populate defaults (enabled=true if missing)
5. Return target + validation errors

**Supported Fields**:
- `name` (string, REQUIRED): Target identifier (unique)
- `type` (string, REQUIRED): Target type (rootly/pagerduty/slack/webhook)
- `url` (string, REQUIRED): Webhook/API URL (must be valid URL)
- `format` (string, REQUIRED): Message format (alertmanager/rootly/pagerduty/slack/webhook)
- `enabled` (bool, optional): Enable/disable target (default: true)
- `headers` (map[string]string, optional): HTTP headers (auth tokens)
- `filter_config` (map[string]any, optional): Target-specific filters

**Acceptance Criteria**:
- [ ] Decodes base64 data["config"]
- [ ] Unmarshals JSON в PublishingTarget
- [ ] Handles missing optional fields (defaults)
- [ ] Returns detailed errors for malformed data
- [ ] Thread-safe parsing (no shared state)

---

### FR-3: Target Validation (CRITICAL)

**Description**: Validate PublishingTarget configuration.

**Validation Rules**:

1. **Required Fields**:
   - `name` must be non-empty string (alphanumeric + hyphens)
   - `type` must be one of: rootly, pagerduty, slack, webhook
   - `url` must be valid HTTP/HTTPS URL
   - `format` must be one of: alertmanager, rootly, pagerduty, slack, webhook

2. **URL Validation**:
   - Must start with `http://` or `https://`
   - Must have valid host (no spaces, valid domain/IP)
   - Path optional
   - Query params optional

3. **Type-Format Compatibility**:
   - `type=rootly` → `format=rootly` (strict)
   - `type=pagerduty` → `format=pagerduty`
   - `type=slack` → `format=slack`
   - `type=webhook` → `format=alertmanager|webhook` (flexible)

4. **Headers Validation**:
   - Keys: non-empty strings
   - Values: non-empty strings
   - No duplicate keys (case-insensitive)

**Validation Errors** (structured):
```go
type ValidationError struct {
    Field   string // "name", "url", "type"
    Message string // "field is required"
    Value   any    // actual value (for debugging)
}
```

**Acceptance Criteria**:
- [ ] Validates all required fields
- [ ] URL validation (go-playground/validator `url` tag)
- [ ] Enum validation for type/format
- [ ] Returns []ValidationError for multiple issues
- [ ] Zero false positives (no valid config rejected)

---

### FR-4: In-Memory Cache (CRITICAL)

**Description**: Fast O(1) lookups для active targets.

**Cache Structure**:
```go
type targetCache struct {
    targets map[string]*PublishingTarget // key: target.Name
    mu      sync.RWMutex                  // thread-safe access
}
```

**Operations**:
- **Set**: `cache.Set(targets []*PublishingTarget)` - replace entire cache
- **Get**: `cache.Get(name string) (*PublishingTarget, bool)` - O(1) lookup
- **List**: `cache.List() []*PublishingTarget` - return all
- **Filter**: `cache.GetByType(targetType string) []*PublishingTarget` - filter by type

**Thread Safety**:
- RWMutex для concurrent reads + single writer
- No race conditions (verified with `-race`)
- Read-heavy optimization (RLock для Get/List)

**Performance Targets**:
- Get: <100ns (target <500ns) = 5x better
- List: <1µs для 20 targets (target <5µs) = 5x better
- Set: <10µs для 20 targets (target <50µs) = 5x better

**Acceptance Criteria**:
- [ ] O(1) Get operation
- [ ] Thread-safe (passes `-race`)
- [ ] Zero allocations в hot path (Get)
- [ ] Benchmarks exceed targets 2x+

---

### FR-5: Target Management API (HIGH)

**Description**: Public API для target access.

**Interface**:
```go
type TargetDiscoveryManager interface {
    // DiscoverTargets lists secrets и refreshes cache
    DiscoverTargets(ctx context.Context) error

    // GetTarget returns target by name (O(1))
    GetTarget(name string) (*PublishingTarget, error)

    // ListTargets returns all active targets
    ListTargets() []*PublishingTarget

    // GetTargetsByType filters targets by type
    GetTargetsByType(targetType string) []*PublishingTarget

    // GetStats returns discovery statistics
    GetStats() DiscoveryStats

    // Health checks manager + K8s client health
    Health(ctx context.Context) error
}
```

**DiscoveryStats**:
```go
type DiscoveryStats struct {
    TotalTargets     int       // Total discovered targets
    ValidTargets     int       // Valid targets in cache
    InvalidTargets   int       // Invalid/skipped targets
    LastDiscovery    time.Time // Last successful discovery
    DiscoveryErrors  int       // Total discovery errors
}
```

**Acceptance Criteria**:
- [ ] All interface methods implemented
- [ ] GetTarget returns NotFoundError for missing
- [ ] GetStats returns accurate metrics
- [ ] Health checks K8s client connectivity

---

## 4. Non-Functional Requirements

### NFR-1: Performance (HIGH)

**Targets** (baseline 100%):
- **Discovery Time**: <2s for 20 secrets (target)
- **Get Target**: <500ns (target), <100ns (goal 150%) ⭐
- **List Targets**: <5µs for 20 targets (target), <1µs (goal 150%) ⭐
- **Parse Secret**: <1ms per secret (target), <500µs (goal 150%) ⭐

**150% Quality Goals**:
- Get: 5x faster than target
- List: 5x faster than target
- Parse: 2x faster than target

**Benchmarks**:
```go
BenchmarkGetTarget
BenchmarkListTargets
BenchmarkParseSecret
BenchmarkDiscoverTargets
```

**Acceptance Criteria**:
- [ ] All benchmarks exceed baseline targets
- [ ] Zero memory allocations in Get hot path
- [ ] Discovery <2s for 20 secrets (real cluster)

---

### NFR-2: Reliability (HIGH)

**Fail-Safe Design**:
- K8s API unavailable → keep old cache, return error
- Some secrets invalid → skip with warning, continue
- Zero secrets found → empty cache, no panic

**Graceful Degradation**:
- Discovery fails → publishing uses old targets (stale OK)
- Parse error → log warning + stacktrace, skip secret
- Validation error → detailed error message, skip

**Error Handling**:
- No panics (all errors wrapped + logged)
- Detailed error messages (secret name + field + reason)
- Structured logging (slog with context)

**Acceptance Criteria**:
- [ ] No panics on invalid input
- [ ] All errors logged with context
- [ ] Partial success supported (some targets fail)

---

### NFR-3: Observability (MEDIUM)

**Prometheus Metrics** (6 total):

1. **alert_history_publishing_discovery_targets_total** (Gauge)
   - Labels: `type`, `enabled`
   - Description: Total discovered targets by type

2. **alert_history_publishing_discovery_duration_seconds** (Histogram)
   - Labels: `operation` (discover/parse/validate)
   - Description: Operation duration

3. **alert_history_publishing_discovery_errors_total** (Counter)
   - Labels: `error_type` (k8s_api/parse/validate)
   - Description: Discovery errors by type

4. **alert_history_publishing_discovery_secrets_total** (Counter)
   - Labels: `status` (valid/invalid/skipped)
   - Description: Processed secrets by status

5. **alert_history_publishing_target_lookups_total** (Counter)
   - Labels: `operation` (get/list/get_by_type), `status` (hit/miss)
   - Description: Cache lookups

6. **alert_history_publishing_discovery_last_success_timestamp** (Gauge)
   - Description: Unix timestamp of last successful discovery

**Structured Logging** (slog):
- Discovery start/end with duration
- Each parsed target (name, type, enabled)
- Invalid secrets (name + validation errors)
- Cache refresh statistics
- Health check results

**Acceptance Criteria**:
- [ ] All 6 metrics registered
- [ ] Metrics updated on every operation
- [ ] Logs include contextual information
- [ ] DEBUG level for verbose details

---

### NFR-4: Testing (HIGH)

**Test Coverage**: 80%+ (target), 85%+ (goal 150%)

**Unit Tests** (минимум 15):
1. Discovery with valid secrets (happy path)
2. Discovery with invalid secrets (parse errors)
3. Discovery with mixed valid/invalid
4. Discovery with zero secrets
5. Parse valid secret
6. Parse invalid base64
7. Parse invalid JSON
8. Parse missing required fields
9. Validate valid target
10. Validate invalid URL
11. Validate invalid type/format
12. GetTarget found
13. GetTarget not found
14. ListTargets
15. GetTargetsByType

**Concurrent Access Tests** (2):
- Concurrent Get + Set (no race)
- Concurrent List + Set (no race)

**Benchmarks** (4):
- BenchmarkGetTarget
- BenchmarkListTargets
- BenchmarkParseSecret
- BenchmarkDiscoverTargets

**Acceptance Criteria**:
- [ ] 15+ unit tests passing
- [ ] 80%+ coverage (go test -cover)
- [ ] Zero race conditions (-race)
- [ ] All benchmarks pass

---

### NFR-5: Documentation (MEDIUM)

**Required Documentation**:

1. **README.md** (800-1000 lines):
   - Quick Start
   - Architecture overview
   - Usage examples (discovery, get, list)
   - Secret format specification
   - Integration guide
   - Troubleshooting (6+ problems)
   - API reference

2. **INTEGRATION_EXAMPLE.md** (300-400 lines):
   - Full integration example
   - Main.go wiring
   - Error handling patterns
   - Testing examples

3. **COMPLETION_REPORT.md** (400-500 lines):
   - Quality metrics
   - Performance results
   - Test coverage summary
   - Production readiness checklist

**Acceptance Criteria**:
- [ ] README.md complete (800+ lines)
- [ ] Integration examples working
- [ ] API reference comprehensive
- [ ] Troubleshooting guide helpful

---

## 5. Dependencies

### 5.1 Satisfied Dependencies

- ✅ **TN-046**: K8s Client (COMPLETE)
  - K8sClient interface
  - ListSecrets/GetSecret methods
  - Retry logic + error handling
  - 72.8% coverage, 46 tests

- ✅ **core.PublishingTarget**: Domain model (COMPLETE)
  - Defined в `go-app/internal/core/interfaces.go`
  - Validation tags (go-playground/validator)
  - JSON serialization

- ✅ **Prometheus Metrics**: Infrastructure (COMPLETE)
  - pkg/metrics/registry.go
  - Business metrics namespace
  - Counter/Gauge/Histogram types

### 5.2 External Dependencies

**Go Packages**:
- `k8s.io/client-go` v0.28.0+: K8s API (via TN-046)
- `log/slog`: Structured logging
- `encoding/base64`: Secret decoding
- `encoding/json`: JSON parsing
- `github.com/go-playground/validator/v10`: Validation
- `sync`: Thread-safe cache (RWMutex)
- `github.com/prometheus/client_golang`: Metrics

**Infrastructure**:
- Kubernetes cluster (in-cluster config)
- ServiceAccount с RBAC permissions (TN-050)
- Prometheus для metrics scraping

---

## 6. Blocks Downstream Tasks

**CRITICAL** - TN-047 блокирует всю PHASE 5:

- **TN-048**: Target Refresh Mechanism
  - Requires: TargetDiscoveryManager.DiscoverTargets()
  - Usage: Periodic refresh (every 5m)

- **TN-049**: Target Health Monitoring
  - Requires: TargetDiscoveryManager.ListTargets()
  - Usage: Health checks для каждого target

- **TN-051**: Alert Formatter
  - Requires: PublishingTarget.Format field
  - Usage: Format alert по target format

- **TN-052-055**: All Publishers
  - Requires: TargetDiscoveryManager.GetTargetsByType()
  - Usage: Find targets by type (rootly/pagerduty/slack)

- **TN-056**: Publishing Queue
  - Requires: PublishingTarget configurations
  - Usage: Enqueue alerts для каждого target

- **TN-057-059**: Metrics, Parallel Publishing, API
  - Requires: TargetDiscoveryManager для target management
  - Usage: Expose targets через API

**Timeline Impact**: Delay в TN-047 блокирует 13 downstream tasks.

---

## 7. Acceptance Criteria

### 7.1 Implementation Checklist (14 items)

- [ ] 1. TargetDiscoveryManager interface defined (5 methods)
- [ ] 2. DefaultTargetDiscoveryManager struct implemented
- [ ] 3. DiscoverTargets() method (K8s integration)
- [ ] 4. parseSecret() function (base64 + JSON)
- [ ] 5. validateTarget() function (comprehensive validation)
- [ ] 6. targetCache struct (thread-safe map)
- [ ] 7. GetTarget() method (O(1) lookup)
- [ ] 8. ListTargets() method (return all)
- [ ] 9. GetTargetsByType() method (filter by type)
- [ ] 10. GetStats() method (discovery statistics)
- [ ] 11. Health() method (K8s client health)
- [ ] 12. Custom errors (NotFoundError, ValidationError)
- [ ] 13. Prometheus metrics registration (6 metrics)
- [ ] 14. Structured logging (slog integration)

### 7.2 Testing Checklist (6 items)

- [ ] 1. 15+ unit tests (80%+ coverage)
- [ ] 2. Concurrent access tests (no races)
- [ ] 3. Benchmarks (4 benchmarks)
- [ ] 4. Integration tests (with fake K8s client)
- [ ] 5. Error handling tests (invalid secrets)
- [ ] 6. Performance validation (all targets met)

### 7.3 Documentation Checklist (6 items)

- [ ] 1. README.md (800+ lines, 8 sections)
- [ ] 2. INTEGRATION_EXAMPLE.md (300+ lines)
- [ ] 3. Secret format specification (YAML examples)
- [ ] 4. Godoc comments (all public APIs)
- [ ] 5. Troubleshooting guide (6+ problems)
- [ ] 6. COMPLETION_REPORT.md (quality metrics)

### 7.4 Quality Checklist (8 items)

- [ ] 1. Zero compilation errors
- [ ] 2. Zero linter warnings (golangci-lint)
- [ ] 3. Zero race conditions (go test -race)
- [ ] 4. 80%+ test coverage (go test -cover)
- [ ] 5. Performance targets exceeded (2x+)
- [ ] 6. All Prometheus metrics working
- [ ] 7. Logging comprehensive (DEBUG/INFO/WARN/ERROR)
- [ ] 8. Documentation complete (requirements+design+tasks)

### 7.5 Production Readiness Checklist (10 items)

- [ ] 1. Thread-safe operations (RWMutex)
- [ ] 2. Context cancellation support (all methods)
- [ ] 3. Graceful degradation (partial failures OK)
- [ ] 4. No panics (all errors wrapped)
- [ ] 5. Fail-safe design (keep old cache on error)
- [ ] 6. Memory efficient (zero allocs in hot path)
- [ ] 7. Secret decoding tested (base64 edge cases)
- [ ] 8. JSON parsing robust (malformed data)
- [ ] 9. URL validation comprehensive (RFC compliance)
- [ ] 10. Integration example working (main.go)

**Total Checklist**: 44 items

---

## 8. Success Metrics (150% Quality Target)

### 8.1 Implementation Quality

- **Completion**: 100% (all 14 implementation items)
- **Code Quality**: Zero linter warnings
- **Performance**: All benchmarks exceed targets 2x+ ⭐

### 8.2 Testing Quality

- **Coverage**: 85%+ (target 80%, +5%) ⭐
- **Tests**: 15+ passing (100% pass rate)
- **Benchmarks**: 4 benchmarks, all passing

### 8.3 Documentation Quality

- **README**: 800+ lines (comprehensive)
- **Examples**: Working integration code
- **API Docs**: 100% coverage (all public APIs)

### 8.4 Operational Quality

- **Metrics**: 6 Prometheus metrics operational
- **Logging**: Structured slog с context
- **Reliability**: No panics, graceful degradation

### 8.5 Grade Calculation

| Category | Weight | Target | Achieved | Score |
|----------|--------|--------|----------|-------|
| Implementation | 25% | 100% | TBD | TBD |
| Testing | 25% | 85% | TBD | TBD |
| Performance | 20% | 200% | TBD | TBD |
| Documentation | 15% | 800 LOC | TBD | TBD |
| Observability | 15% | 6 metrics | TBD | TBD |

**Target Score**: 93+/100 (Grade A+)

---

## 9. Risks & Mitigation

### Risk 1: K8s API Unavailable (MEDIUM)

**Impact**: Discovery fails → no targets → publishing blocked

**Mitigation**:
- Keep old cache on discovery failure (stale OK)
- Retry logic в K8sClient (TN-046)
- Health checks expose K8s status
- Alert on discovery failures (Prometheus)

### Risk 2: Invalid Secrets in Production (LOW)

**Impact**: Some targets unavailable

**Mitigation**:
- Comprehensive validation (early detection)
- Detailed error messages (easy debugging)
- Skip invalid secrets (partial success)
- Metrics track invalid_secrets_total

### Risk 3: Performance Regression (LOW)

**Impact**: Slow discovery blocks startup

**Mitigation**:
- Benchmark tests в CI
- Timeout на discovery (fail-fast)
- Async discovery optional (TN-048)
- Performance monitoring (histograms)

### Risk 4: Secret Format Changes (LOW)

**Impact**: Parse errors on schema evolution

**Mitigation**:
- Strict validation (explicit schema)
- Version field в secret (future-proof)
- Backward compatibility tests
- Documentation of format

---

## 10. Timeline

**Estimated Effort**: 8-10 hours (150% quality target)

**Phase Breakdown**:
- Phase 1: Analysis & Design (1h) - THIS DOCUMENT ✅
- Phase 2: Core Implementation (3h)
- Phase 3: Testing (2h)
- Phase 4: Metrics & Logging (1h)
- Phase 5: Documentation (2h)
- Phase 6: Final Review & Certification (1h)

**Dependencies Wait Time**: 0 hours (TN-046 complete)

**Total Timeline**: 10 hours (including documentation)

---

## 11. References

- **TN-046**: K8s Client Implementation
  - `/go-app/internal/infrastructure/k8s/client.go`
  - `/go-app/internal/infrastructure/k8s/README.md`

- **PublishingTarget Model**:
  - `/go-app/internal/core/interfaces.go` (lines 76-85)

- **Prometheus Metrics**:
  - `/go-app/pkg/metrics/business.go`

- **Similar Tasks** (quality reference):
  - TN-046: 150%+ quality, 72.8% coverage, 46 tests
  - TN-134: 150%+ quality, 90.1% coverage, 61 tests
  - TN-136: 150% quality, 5,800 LOC production

---

**Document Version**: 2.0
**Last Updated**: 2025-11-08
**Status**: ✅ READY FOR IMPLEMENTATION
**Next Step**: Create design.md with technical architecture
