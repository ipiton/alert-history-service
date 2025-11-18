# TN-146: Prometheus Alert Parser — Requirements

> **Статус**: 🚧 IN PROGRESS
> **Приоритет**: **P0 CRITICAL** (Blocks Alertmanager compatibility)
> **Target Quality**: **150% (Grade A+, Enterprise-level)**
> **Estimated Duration**: 3-5 days

---

## 📋 Executive Summary

Реализация Prometheus Alert Parser для парсинга нативных Prometheus alerts (формат `/api/v1/alerts` и `/api/v2/alerts`) с полной совместимостью с Alertmanager API v2 и Prometheus API v1.

### Context

**Критический GAP**: Phase 1 Alert Ingestion фактически завершена на **78.6%** (не 100%!), отсутствуют TN-146, TN-147, TN-148 — блокируют совместимость с Prometheus.

**Impact**: Без этого компонента система **НЕ МОЖЕТ** быть "drop-in replacement" для Alertmanager.

---

## 🎯 Business Requirements

### BR-1: Prometheus API Compatibility
**Priority**: P0 CRITICAL
**Description**: Парсер должен поддерживать оба формата Prometheus alerts:
- Prometheus v1 format (legacy `/api/v1/alerts`)
- Prometheus v2 format (modern `/api/v2/alerts`)
- Alertmanager webhook format (уже реализовано в TN-41)

**Rationale**: Обеспечить 100% совместимость с Prometheus ecosystem.

**Acceptance Criteria**:
- ✅ Parse Prometheus v1 alert format (JSON array)
- ✅ Parse Prometheus v2 alert format (с группировкой)
- ✅ Support all standard Prometheus alert fields
- ✅ Validate timestamps (RFC3339)
- ✅ Handle missing optional fields gracefully
- ✅ Generate fingerprints deterministically

---

### BR-2: Format Detection
**Priority**: P0 CRITICAL
**Description**: Автоматическое определение формата входящего alert (Prometheus vs Alertmanager).

**Rationale**: Поддержка multiple sources без explicit configuration.

**Acceptance Criteria**:
- ✅ Detect Prometheus v1 format (array of objects)
- ✅ Detect Prometheus v2 format (grouped alerts)
- ✅ Detect Alertmanager webhook (уже реализовано)
- ✅ Fallback to generic parser при неопределённом формате
- ✅ Log detection decisions для troubleshooting

---

### BR-3: Domain Model Conversion
**Priority**: P0 CRITICAL
**Description**: Конвертация Prometheus alerts в унифицированную domain model `core.Alert`.

**Rationale**: Единая модель для всех downstream компонентов (grouping, inhibition, storage).

**Acceptance Criteria**:
- ✅ Map Prometheus fields → core.Alert
- ✅ Extract `alertname` from labels
- ✅ Convert status: "firing" | "pending" | "inactive" → core.AlertStatus
- ✅ Handle `generatorURL` (required in Prometheus, optional in core)
- ✅ Preserve all labels and annotations
- ✅ Generate fingerprint (SHA256 of labels)

---

### BR-4: Backward Compatibility
**Priority**: P0 CRITICAL
**Description**: Не ломать существующий Alertmanager webhook parser (TN-41).

**Rationale**: Сохранить работоспособность существующих интеграций.

**Acceptance Criteria**:
- ✅ Existing AlertmanagerParser continues to work
- ✅ No breaking changes to core.Alert model
- ✅ Detector correctly distinguishes formats
- ✅ All existing tests pass

---

## 🔧 Functional Requirements

### FR-1: Prometheus Alert Structure Support
**Priority**: P0
**Description**: Поддержка всех полей Prometheus alert.

**Prometheus Alert Format (v1/v2)**:
```json
{
  "labels": {
    "alertname": "HighCPU",
    "instance": "server-1",
    "job": "api",
    "severity": "warning"
  },
  "annotations": {
    "summary": "CPU usage high",
    "description": "CPU > 80% for 5m"
  },
  "state": "firing",
  "activeAt": "2025-11-18T10:00:00Z",
  "value": "0.85",
  "generatorURL": "http://prometheus:9090/graph?g0.expr=...",
  "fingerprint": "abc123def456"
}
```

**Fields Mapping**:
| Prometheus Field | core.Alert Field | Type | Required | Notes |
|------------------|------------------|------|----------|-------|
| `labels` | `Labels` | map[string]string | ✅ Yes | Must contain `alertname` |
| `annotations` | `Annotations` | map[string]string | ❌ No | Optional |
| `state` | `Status` | AlertStatus | ✅ Yes | "firing", "pending", "inactive" |
| `activeAt` | `StartsAt` | time.Time | ✅ Yes | RFC3339 format |
| `value` | Annotations["value"] | string | ❌ No | Store in annotations |
| `generatorURL` | `GeneratorURL` | *string | ✅ Yes (Prom) | Required in Prometheus |
| `fingerprint` | `Fingerprint` | string | ❌ No | Generate if missing |

**Status Mapping**:
```go
"firing"   → core.StatusFiring
"pending"  → core.StatusFiring  // Treat pending as firing
"inactive" → core.StatusResolved
"resolved" → core.StatusResolved (Alertmanager format)
```

**Acceptance Criteria**:
- ✅ Parse all fields correctly
- ✅ Handle missing optional fields
- ✅ Validate required fields
- ✅ Return descriptive errors

---

### FR-2: Format Detection Algorithm
**Priority**: P0
**Description**: Алгоритм определения формата payload.

**Detection Logic**:
```go
func DetectPrometheusFormat(payload []byte) PrometheusFormatType {
    // 1. Try parse as JSON
    var data interface{}
    json.Unmarshal(payload, &data)

    // 2. Check structure
    switch v := data.(type) {
    case []interface{}:
        // Array → could be Prometheus v1
        if hasPrometheusFields(v[0]) {
            return PrometheusV1
        }
    case map[string]interface{}:
        // Object → could be Prometheus v2 or Alertmanager
        if hasField(v, "version") && hasField(v, "groupKey") {
            return Alertmanager  // Already handled by TN-41
        }
        if hasField(v, "alerts") && hasField(v, "groupLabels") {
            return PrometheusV2
        }
    }

    return Unknown
}

func hasPrometheusFields(alert interface{}) bool {
    m, ok := alert.(map[string]interface{})
    if !ok {
        return false
    }
    // Prometheus-specific: state, activeAt, generatorURL
    return hasField(m, "state") &&
           hasField(m, "activeAt") &&
           hasField(m, "labels")
}
```

**Format Characteristics**:
| Format | Structure | Key Fields | Example |
|--------|-----------|------------|---------|
| **Prometheus v1** | JSON Array | `state`, `activeAt`, `labels` | `[{state:"firing",...}]` |
| **Prometheus v2** | JSON Object | `alerts`, `groupLabels` | `{alerts:[...], groupLabels:{}}` |
| **Alertmanager** | JSON Object | `version`, `groupKey`, `receiver` | `{version:"4", alerts:[...]}` |

**Acceptance Criteria**:
- ✅ Detect Prometheus v1 with 100% accuracy
- ✅ Detect Prometheus v2 with 100% accuracy
- ✅ Distinguish from Alertmanager format
- ✅ Fallback gracefully for unknown formats

---

### FR-3: Parser Implementation
**Priority**: P0
**Description**: PrometheusParser реализует WebhookParser interface.

**Interface**:
```go
type WebhookParser interface {
    Parse(data []byte) (*AlertmanagerWebhook, error)
    Validate(webhook *AlertmanagerWebhook) *ValidationResult
    ConvertToDomain(webhook *AlertmanagerWebhook) ([]*core.Alert, error)
}
```

**Проблема**: Interface ожидает `*AlertmanagerWebhook`, но Prometheus имеет другую структуру!

**Решение**: Создать unified intermediate model:
```go
// PrometheusAlert represents a single Prometheus alert
type PrometheusAlert struct {
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    State        string            `json:"state"`        // firing, pending, inactive
    ActiveAt     time.Time         `json:"activeAt"`
    Value        string            `json:"value"`
    GeneratorURL string            `json:"generatorURL"`
    Fingerprint  string            `json:"fingerprint,omitempty"`
}

// PrometheusAlertGroup for v2 format
type PrometheusAlertGroup struct {
    Labels map[string]string  `json:"labels"`        // Group labels
    Alerts []PrometheusAlert  `json:"alerts"`        // Alerts in group
}

// PrometheusWebhook unified structure
type PrometheusWebhook struct {
    Alerts []PrometheusAlert      `json:"alerts,omitempty"` // v1: direct array
    Groups []PrometheusAlertGroup `json:"groups,omitempty"` // v2: grouped
}
```

**Parser Implementation**:
```go
type prometheusParser struct {
    validator WebhookValidator
}

func NewPrometheusParser() WebhookParser {
    return &prometheusParser{
        validator: NewWebhookValidator(),
    }
}

func (p *prometheusParser) Parse(data []byte) (*AlertmanagerWebhook, error) {
    // Parse Prometheus → Convert to AlertmanagerWebhook format
    // (для совместимости с existing interface)
}
```

**Acceptance Criteria**:
- ✅ Implement WebhookParser interface
- ✅ Parse Prometheus v1 format
- ✅ Parse Prometheus v2 format
- ✅ Convert to core.Alert
- ✅ Generate fingerprints
- ✅ Validate parsed data

---

### FR-4: Validation Rules
**Priority**: P1
**Description**: Comprehensive validation для Prometheus alerts.

**Validation Rules**:
1. **Labels**:
   - ✅ `alertname` is required
   - ✅ Label names match `[a-zA-Z_][a-zA-Z0-9_]*`
   - ✅ Label values are non-empty strings

2. **Status**:
   - ✅ Must be one of: "firing", "pending", "inactive", "resolved"

3. **Timestamps**:
   - ✅ `activeAt` is required and valid RFC3339
   - ✅ `activeAt` not in the future (with tolerance 5m)

4. **GeneratorURL**:
   - ✅ Valid URL format (if present)
   - ✅ Required in Prometheus format

**Error Messages**:
```go
var (
    ErrMissingAlertname     = errors.New("missing required label 'alertname'")
    ErrInvalidState         = errors.New("invalid state, must be firing|pending|inactive")
    ErrMissingActiveAt      = errors.New("activeAt is required")
    ErrInvalidTimestamp     = errors.New("invalid RFC3339 timestamp")
    ErrInvalidGeneratorURL  = errors.New("invalid generatorURL format")
)
```

**Acceptance Criteria**:
- ✅ Validate all required fields
- ✅ Return descriptive error messages
- ✅ Support partial validation (non-blocking warnings)

---

### FR-5: Fingerprint Generation
**Priority**: P0
**Description**: Генерация deterministic fingerprint для Prometheus alerts.

**Algorithm** (совместимый с TN-41):
```go
func generateFingerprint(alertName string, labels map[string]string) string {
    // 1. Sort label keys
    keys := make([]string, 0, len(labels))
    for k := range labels {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // 2. Build canonical string
    parts := []string{alertName}
    for _, k := range keys {
        parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
    }
    canonical := strings.Join(parts, "|")

    // 3. SHA256 hash
    hash := sha256.Sum256([]byte(canonical))
    return fmt.Sprintf("%x", hash)
}
```

**Rationale**: Тот же алгоритм что в Alertmanager parser (TN-41) для consistency.

**Acceptance Criteria**:
- ✅ Same labels → same fingerprint
- ✅ Different labels → different fingerprint
- ✅ Deterministic across restarts
- ✅ Compatible with TN-41 algorithm

---

## 📊 Non-Functional Requirements

### NFR-1: Performance
**Priority**: P0
**Description**: Ultra-fast parsing для high-throughput scenarios.

**Targets**:
| Metric | Target | Stretch Goal | Baseline |
|--------|--------|--------------|----------|
| Parse single alert | < 10µs | < 5µs | 20µs (TN-41) |
| Parse 100 alerts | < 1ms | < 500µs | 2ms |
| Fingerprint generation | < 1µs | < 500ns | 82ns (TN-36) |
| Memory per alert | < 1KB | < 500B | - |
| Zero allocations | Hot path | All paths | - |

**Rationale**: Prometheus может отправлять 10,000+ alerts/sec в burst scenarios.

**Acceptance Criteria**:
- ✅ Benchmarks for all operations
- ✅ Meet or exceed targets
- ✅ Zero allocations in hot path
- ✅ CPU profiling shows no bottlenecks

---

### NFR-2: Test Coverage
**Priority**: P0
**Description**: Comprehensive test suite (unit + integration + benchmarks).

**Targets**:
| Type | Target | Stretch Goal |
|------|--------|--------------|
| **Unit tests** | 85%+ | 95%+ |
| **Line coverage** | 90%+ | 98%+ |
| **Benchmarks** | 8+ | 12+ |
| **Test cases** | 30+ | 50+ |

**Test Categories**:
1. **Format Detection** (10 tests)
   - Prometheus v1 detection
   - Prometheus v2 detection
   - Alertmanager detection (regression)
   - Unknown format handling
   - Edge cases (empty, invalid JSON)

2. **Parsing** (15 tests)
   - Valid Prometheus v1 alert
   - Valid Prometheus v2 alert
   - Missing required fields
   - Invalid timestamps
   - Invalid status values
   - Large payloads (1000+ alerts)

3. **Validation** (10 tests)
   - Required fields validation
   - Label name validation
   - Timestamp validation
   - GeneratorURL validation
   - Error messages

4. **Conversion** (10 tests)
   - Prometheus → core.Alert
   - Status mapping
   - Fingerprint generation
   - Field preservation
   - Nil handling

5. **Integration** (5 tests)
   - End-to-end: Parse → Validate → Convert
   - Detector + Parser integration
   - Handler integration
   - Error propagation

**Acceptance Criteria**:
- ✅ 85%+ test coverage
- ✅ 100% test pass rate
- ✅ Zero race conditions (go test -race)
- ✅ All benchmarks pass

---

### NFR-3: Error Handling
**Priority**: P1
**Description**: Graceful error handling с descriptive messages.

**Error Categories**:
1. **Parse Errors** (recoverable)
   - Invalid JSON → return error
   - Missing fields → return error with field name
   - Invalid format → return error with detection result

2. **Validation Errors** (recoverable)
   - Invalid timestamp → return error with value
   - Invalid status → return error with valid options
   - Missing alertname → return error with field path

3. **System Errors** (non-recoverable)
   - Out of memory → panic with recovery
   - Context cancelled → return context.Err()

**Error Response Format**:
```go
type ParserError struct {
    Type    string            `json:"type"`    // "parse_error", "validation_error"
    Message string            `json:"message"` // Human-readable
    Field   string            `json:"field"`   // Field path (e.g. "alerts[0].labels")
    Value   interface{}       `json:"value"`   // Invalid value
}
```

**Acceptance Criteria**:
- ✅ All errors have descriptive messages
- ✅ Field path указан для validation errors
- ✅ No generic "error parsing webhook"
- ✅ Error logs include context

---

### NFR-4: Observability
**Priority**: P1
**Description**: Comprehensive metrics и logging.

**Prometheus Metrics** (8 total):
```go
// Existing metrics (from TN-45)
webhook_requests_total{format="prometheus_v1|prometheus_v2|alertmanager", status="success|failure"}
webhook_processing_duration_seconds{format, stage="parse|validate|convert"}
webhook_parse_errors_total{format, error_type}
webhook_payload_size_bytes{format}

// New metrics for Prometheus
prometheus_alerts_parsed_total{version="v1|v2", status="success|failure"}
prometheus_format_detection_total{detected_format, actual_format}
prometheus_fingerprint_generation_duration_seconds
prometheus_validation_errors_total{error_type}
```

**Structured Logging**:
```go
logger.Info("Prometheus alert parsed",
    "version", "v1",
    "alert_count", 5,
    "duration_ms", 0.8,
    "fingerprints", []string{"abc", "def"},
)

logger.Error("Failed to parse Prometheus alert",
    "error", err,
    "payload_size", len(data),
    "detected_format", "prometheus_v1",
)
```

**Acceptance Criteria**:
- ✅ 8 Prometheus metrics instrumented
- ✅ Structured logging with context
- ✅ Debug logs за detection decisions
- ✅ Performance metrics recorded

---

### NFR-5: Documentation
**Priority**: P1
**Description**: Comprehensive documentation (code, API, examples).

**Deliverables**:
1. **Code Documentation** (500+ lines godoc)
   - Package overview
   - Type definitions
   - Function documentation with examples
   - Performance notes

2. **API Documentation** (300+ lines markdown)
   - Prometheus format specification
   - Field mapping table
   - Example payloads (v1, v2)
   - Error responses

3. **README** (400+ lines)
   - Quick start
   - Format comparison table
   - Usage examples
   - Troubleshooting

4. **Integration Guide** (200+ lines)
   - Prometheus configuration
   - Testing with real Prometheus
   - Migration from Alertmanager

**Acceptance Criteria**:
- ✅ All public types documented
- ✅ Example code compiles
- ✅ README comprehensive
- ✅ Integration guide tested

---

## 🔗 Dependencies

### Upstream Dependencies
| Task | Status | Impact | Notes |
|------|--------|--------|-------|
| **TN-41** | ✅ Complete | High | Alertmanager parser (reuse patterns) |
| **TN-42** | ⚠️ Partial | High | Universal handler (need fix Mock) |
| **TN-31** | ✅ Complete | Critical | core.Alert model |
| **TN-36** | ✅ Complete | Medium | Fingerprint algorithm |

**Blockers**: NONE (все зависимости ready)

### Downstream Dependencies
| Task | Impact | Notes |
|------|--------|-------|
| **TN-147** | Critical | POST /api/v2/alerts endpoint (uses этот parser) |
| **TN-148** | High | Prometheus-compatible response |
| **Phase 3-8** | Medium | All phases work с core.Alert |

---

## 🎯 Success Criteria

### Must Have (150% Quality)
- ✅ **Implementation**: 100% (all FR + NFR implemented)
- ✅ **Testing**: 85%+ coverage, 30+ tests, 8+ benchmarks
- ✅ **Performance**: Meet all targets (< 10µs parse, < 1µs fingerprint)
- ✅ **Documentation**: 1,400+ lines (code + markdown)
- ✅ **Quality**: Grade A+ (95/100 points)
- ✅ **Zero Technical Debt**: No TODOs, no hacks
- ✅ **Backward Compatible**: All TN-41 tests pass

### Stretch Goals (200% Quality)
- 🌟 **Performance**: Exceed targets by 2x (< 5µs parse)
- 🌟 **Coverage**: 95%+ line coverage
- 🌟 **Tests**: 50+ test cases
- 🌟 **Benchmarks**: 12+ benchmarks
- 🌟 **Documentation**: 2,000+ lines

---

## 📅 Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1**: Requirements & Design | 4h | This doc + design.md |
| **Phase 2**: Format Detection | 6h | Detector + 10 tests |
| **Phase 3**: Parser Implementation | 8h | Parser + 15 tests |
| **Phase 4**: Validation | 4h | Validator + 10 tests |
| **Phase 5**: Conversion | 6h | Converter + 10 tests |
| **Phase 6**: Integration | 4h | Handler integration + 5 tests |
| **Phase 7**: Benchmarks | 3h | 8+ benchmarks |
| **Phase 8**: Documentation | 5h | README + godoc |
| **Phase 9**: QA & Polish | 4h | Cleanup + review |

**Total**: 44 hours = 5-6 days (150% quality)

---

## 📝 Notes

1. **Interface Compatibility**: Existing `WebhookParser` interface ожидает `*AlertmanagerWebhook`. Нужно либо:
   - Option A: Extend interface для support PrometheusWebhook
   - Option B: Convert Prometheus → Alertmanager format internally
   - **Recommendation**: Option B (меньше breaking changes)

2. **Status Mapping**: Prometheus has "pending" state, core.Alert has только "firing" | "resolved". Решение: map "pending" → "firing".

3. **GeneratorURL**: Required в Prometheus, optional в core.Alert. Решение: store в core.Alert.GeneratorURL.

4. **Fingerprint Algorithm**: Must be compatible with TN-41 для consistency.

---

**Prepared by**: Independent Technical Analysis
**Date**: 2025-11-18
**Target Start**: Immediately
**Target Completion**: T+5 days (150% quality)
