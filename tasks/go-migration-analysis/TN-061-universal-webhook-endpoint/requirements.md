# TN-061: POST /webhook - Universal Webhook Endpoint
## 📋 TECHNICAL REQUIREMENTS SPECIFICATION

**Version**: 1.0  
**Date**: 2025-11-15  
**Status**: Draft  
**Target Quality**: 150% Enterprise Grade (Grade A++)

---

## 1. OVERVIEW

### 1.1 Purpose
Создать production-ready REST API endpoint `POST /webhook` для универсального приема webhook notifications от различных alert-систем (Prometheus, Alertmanager, generic webhooks) с автоматическим определением формата, валидацией, обработкой и публикацией алертов.

### 1.2 Scope
Endpoint покрывает:
- ✅ HTTP request handling (POST method)
- ✅ Middleware stack (10+ middleware components)
- ✅ Webhook format auto-detection (Alertmanager, Generic)
- ✅ Payload parsing and validation
- ✅ Domain model conversion (Webhook → core.Alert)
- ✅ Async alert processing (classification, deduplication, publishing)
- ✅ Response formatting (success, partial success, failure)
- ✅ Metrics collection (15+ Prometheus metrics)
- ✅ Security (rate limiting, authentication, input validation)

### 1.3 Out of Scope
- ❌ GraphQL API (future: TN-200+)
- ❌ WebSocket streaming (future: TN-210+)
- ❌ Batch file upload (future: TN-220+)
- ❌ Historical alert replay (future: TN-230+)
- ❌ Alert query API (covered by TN-063)

### 1.4 Success Criteria
Endpoint считается успешным при выполнении **всех** следующих критериев:

#### 1.4.1 Functional Criteria
- ✅ Accepts Alertmanager webhook format (v0.25+)
- ✅ Accepts generic webhook format (custom JSON)
- ✅ Auto-detects format with >95% accuracy
- ✅ Validates all required fields
- ✅ Processes 100% of valid alerts
- ✅ Returns appropriate HTTP status codes
- ✅ Provides detailed error messages

#### 1.4.2 Non-Functional Criteria (150% Targets)
- ✅ **Performance**: <5ms p99 latency, >10K req/s throughput
- ✅ **Reliability**: 99.95% uptime, <0.01% error rate
- ✅ **Security**: OWASP Top 10 compliant, rate limiting, auth support
- ✅ **Observability**: 15+ Prometheus metrics, structured logging, Grafana dashboard
- ✅ **Quality**: 95%+ test coverage, 80+ tests, zero linter warnings
- ✅ **Documentation**: 5,000+ LOC (API guide, troubleshooting, ADRs)

---

## 2. FUNCTIONAL REQUIREMENTS

### FR-1: HTTP Request Handling
**Priority**: CRITICAL  
**Status**: Required

#### FR-1.1 HTTP Method Support
- **MUST**: Accept `POST` method
- **MUST**: Reject `GET`, `PUT`, `DELETE`, `PATCH` with 405 Method Not Allowed
- **SHOULD**: Include `Allow: POST` header in 405 responses

#### FR-1.2 Content-Type Support
- **MUST**: Accept `application/json`
- **SHOULD**: Accept `application/x-www-form-urlencoded` (for legacy clients)
- **MUST**: Reject unsupported content types with 415 Unsupported Media Type

#### FR-1.3 Request Size Limits
- **MUST**: Enforce maximum request size: 10MB
- **MUST**: Return 413 Payload Too Large if exceeded
- **MUST**: Include `Retry-After` header with 413 response

#### FR-1.4 Request Timeout
- **MUST**: Set request timeout: 30 seconds
- **MUST**: Return 408 Request Timeout if exceeded
- **SHOULD**: Allow timeout configuration via environment variable

### FR-2: Webhook Format Detection
**Priority**: CRITICAL  
**Status**: Required

#### FR-2.1 Alertmanager Format Detection
- **MUST**: Detect Alertmanager webhook format based on:
  - Presence of `"alerts"` array at root level
  - Presence of `"groupKey"`, `"status"`, `"receiver"` fields
  - Alert structure with `"labels"`, `"annotations"`, `"startsAt"`, `"endsAt"`
- **MUST**: Confidence score ≥0.8 for Alertmanager format

#### FR-2.2 Generic Format Detection
- **MUST**: Detect generic webhook format for:
  - Valid JSON structure
  - Does not match Alertmanager pattern
  - Fallback for unknown formats
- **MUST**: Confidence score 0.5-0.8 for generic format

#### FR-2.3 Detection Error Handling
- **MUST**: Return 400 Bad Request for invalid JSON
- **MUST**: Include detection confidence in response (debug mode)
- **SHOULD**: Log detection results for analysis

### FR-3: Payload Parsing
**Priority**: CRITICAL  
**Status**: Required

#### FR-3.1 Alertmanager Payload Parsing
- **MUST**: Parse `alerts` array (1-1000 alerts)
- **MUST**: Extract for each alert:
  - `status`: "firing" or "resolved"
  - `labels`: map[string]string
  - `annotations`: map[string]string
  - `startsAt`: RFC3339 timestamp
  - `endsAt`: RFC3339 timestamp (optional for firing alerts)
  - `generatorURL`: string (optional)
- **MUST**: Parse root-level fields:
  - `version`: string (e.g., "4")
  - `groupKey`: string
  - `status`: "firing" or "resolved"
  - `receiver`: string
  - `groupLabels`: map[string]string
  - `commonLabels`: map[string]string
  - `commonAnnotations`: map[string]string
  - `externalURL`: string

#### FR-3.2 Generic Payload Parsing
- **MUST**: Parse top-level fields as key-value pairs
- **MUST**: Support nested JSON objects (max depth: 10)
- **SHOULD**: Extract alert-like structure if present
- **SHOULD**: Generate synthetic alert from payload

#### FR-3.3 Parsing Error Handling
- **MUST**: Return 400 Bad Request for malformed JSON
- **MUST**: Include detailed error message (e.g., "line 5, column 12")
- **MUST**: Log parsing errors with payload sample (first 500 bytes)

### FR-4: Payload Validation
**Priority**: CRITICAL  
**Status**: Required

#### FR-4.1 Alertmanager Payload Validation
- **MUST**: Validate required fields:
  - Each alert must have `status` (enum: "firing", "resolved")
  - Each alert must have `labels` (non-empty map)
  - Each alert must have `labels.alertname` (non-empty string)
  - Each alert must have `startsAt` (valid RFC3339 timestamp)
  - Resolved alerts must have `endsAt` (valid RFC3339 timestamp)
- **SHOULD**: Validate optional fields:
  - `labels`: alphanumeric + underscore, max 1000 labels per alert
  - `annotations`: UTF-8 strings, max 1000 annotations per alert
  - `generatorURL`: valid URL format

#### FR-4.2 Generic Payload Validation
- **MUST**: Validate basic structure (non-null, non-empty)
- **SHOULD**: Validate field types (string, number, boolean, object, array)
- **SHOULD**: Check for suspicious patterns (SQL injection, XSS)

#### FR-4.3 Validation Error Handling
- **MUST**: Return 400 Bad Request for validation failures
- **MUST**: Include field-level error details in response:
  ```json
  {
    "status": "validation_failed",
    "message": "Webhook validation failed",
    "errors": [
      "Alert 0: missing required field 'status'",
      "Alert 3: invalid timestamp format for 'startsAt': '2023-13-45T...'",
      "Alert 7: label name 'invalid-label' contains invalid characters"
    ]
  }
  ```
- **MUST**: Log validation failures with full payload (debug level)

### FR-5: Domain Model Conversion
**Priority**: CRITICAL  
**Status**: Required

#### FR-5.1 Alert Conversion (Webhook → core.Alert)
- **MUST**: Convert each webhook alert to `core.Alert` struct
- **MUST**: Generate fingerprint using FNV64a hash (labels-based)
- **MUST**: Set timestamps:
  - `StartsAt`: from webhook `startsAt` field
  - `EndsAt`: from webhook `endsAt` field (if resolved)
  - `Timestamp`: current time (processing timestamp)
- **MUST**: Normalize labels and annotations:
  - Lowercase label names
  - Trim whitespace
  - Limit label values to 1000 characters
- **MUST**: Set instance ID (from config or hostname)

#### FR-5.2 Metadata Enrichment
- **SHOULD**: Add processing metadata:
  - `webhook_type`: "alertmanager" or "generic"
  - `received_at`: RFC3339 timestamp
  - `request_id`: UUID from X-Request-ID header
  - `source_ip`: client IP address
- **SHOULD**: Add default values:
  - `severity`: "unknown" (if not in labels)
  - `namespace`: "default" (if not in labels)

### FR-6: Alert Processing
**Priority**: CRITICAL  
**Status**: Required

#### FR-6.1 Async Processing
- **MUST**: Submit alerts to async worker pool (non-blocking)
- **MUST**: Process alerts in parallel (bounded parallelism)
- **MUST**: Wait for processing completion with timeout (30s)
- **MUST**: Collect processing results (success/failure per alert)

#### FR-6.2 Processing Pipeline
For each alert, **MUST** execute:
1. Deduplication (check fingerprint cache)
2. Classification (LLM severity detection, with circuit breaker)
3. Enrichment (add LLM recommendations)
4. Filtering (namespace, severity filters)
5. Grouping (group_by labels, if configured)
6. Inhibition (check inhibit rules)
7. Silencing (check silence rules)
8. Storage (PostgreSQL persistence)
9. Publishing (dispatch to targets: Rootly, PagerDuty, Slack, Webhook)

#### FR-6.3 Processing Error Handling
- **MUST**: Continue processing other alerts if one fails
- **MUST**: Collect error details for failed alerts
- **MUST**: Return partial success response if some alerts fail:
  ```json
  {
    "status": "partial_success",
    "message": "Processed 7 of 10 alerts",
    "alerts_received": 10,
    "alerts_processed": 7,
    "errors": [
      "Alert 3 (HighCPU): database timeout",
      "Alert 8 (DiskFull): publishing failed: Slack webhook timeout",
      "Alert 9 (NetworkError): classification failed: LLM service unavailable"
    ]
  }
  ```

### FR-7: Response Formatting
**Priority**: CRITICAL  
**Status**: Required

#### FR-7.1 Success Response (HTTP 200)
```json
{
  "status": "success",
  "message": "Webhook processed successfully",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 10,
  "processing_time": "45.2ms",
  "request_id": "req-abc123..."
}
```

#### FR-7.2 Partial Success Response (HTTP 207 Multi-Status)
```json
{
  "status": "partial_success",
  "message": "Processed 7 of 10 alerts",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 7,
  "errors": [
    "Alert 3 (HighCPU): database timeout",
    "Alert 8 (DiskFull): publishing failed"
  ],
  "processing_time": "52.8ms",
  "request_id": "req-abc123..."
}
```

#### FR-7.3 Validation Failure Response (HTTP 400)
```json
{
  "status": "validation_failed",
  "message": "Webhook validation failed",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 0,
  "errors": [
    "Alert 0: missing required field 'status'",
    "Alert 3: invalid timestamp format"
  ],
  "processing_time": "5.2ms",
  "request_id": "req-abc123..."
}
```

#### FR-7.4 Processing Failure Response (HTTP 500)
```json
{
  "status": "failure",
  "message": "All alerts failed to process",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 0,
  "errors": [
    "Database connection failed",
    "Worker pool exhausted"
  ],
  "processing_time": "30.1s",
  "request_id": "req-abc123..."
}
```

#### FR-7.5 Rate Limit Response (HTTP 429)
```json
{
  "status": "rate_limited",
  "message": "Too many requests. Please retry after 60 seconds",
  "limit_type": "per_ip",
  "limit": "100 requests per minute",
  "retry_after": 60,
  "request_id": "req-abc123..."
}
```

#### FR-7.6 Response Headers
- **MUST**: Include `Content-Type: application/json`
- **MUST**: Include `X-Request-ID: <uuid>`
- **MUST**: Include `X-Processing-Time: <duration_ms>` (milliseconds)
- **SHOULD**: Include `X-Webhook-Type: alertmanager|generic`
- **SHOULD**: Include `X-Alerts-Processed: <count>`
- **MUST**: Include `Retry-After: <seconds>` (for 429, 503 responses)

---

## 3. NON-FUNCTIONAL REQUIREMENTS

### NFR-1: Performance Requirements (150% Target)
**Priority**: CRITICAL  
**Status**: Required

#### NFR-1.1 Latency Requirements
| Metric | Baseline (100%) | Target (150%) | Measurement |
|--------|----------------|---------------|-------------|
| **p50 latency** | 5ms | 3ms | k6 histogram |
| **p95 latency** | 8ms | 4.5ms | k6 histogram |
| **p99 latency** | 10ms | **5ms** | k6 histogram |
| **p99.9 latency** | 50ms | 30ms | k6 histogram |

**Measurement Method**:
- k6 load test with 10,000 req/s for 10 minutes
- Histogram buckets: .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10

**Acceptance Criteria**:
- ✅ p99 < 5ms under normal load (1K req/s)
- ✅ p99 < 10ms under peak load (10K req/s)
- ✅ p99.9 < 30ms under stress load (15K req/s)

#### NFR-1.2 Throughput Requirements
| Metric | Baseline (100%) | Target (150%) | Measurement |
|--------|----------------|---------------|-------------|
| **Sustained throughput** | 1,000 req/s | **10,000 req/s** | k6 steady state |
| **Peak throughput** | 5,000 req/s | **20,000 req/s** | k6 spike test |
| **Concurrent requests** | 1,000 | 5,000 | k6 VUs |

**Measurement Method**:
- k6 ramp-up test: 0 → 10K req/s over 5 minutes
- k6 spike test: 1K → 20K → 1K req/s (30s spike)

**Acceptance Criteria**:
- ✅ Sustain 10K req/s for 10 minutes without degradation
- ✅ Handle 20K req/s spike without errors
- ✅ 5,000 concurrent connections without memory exhaustion

#### NFR-1.3 Resource Requirements
| Resource | Baseline (100%) | Target (150%) | Measurement |
|----------|----------------|---------------|-------------|
| **Memory per 10K requests** | 150MB | **100MB** | pprof heap |
| **CPU at 5K req/s** | 70% | **50%** | pprof CPU |
| **Goroutines** | 1,000 | 500 | pprof goroutine |
| **File descriptors** | 2,000 | 1,500 | lsof |

**Measurement Method**:
- pprof heap profile every 30 seconds during load test
- pprof CPU profile for 30 seconds at peak load
- Monitor goroutine count via expvar

**Acceptance Criteria**:
- ✅ Memory stable (no leaks) during 4-hour soak test
- ✅ CPU <50% at 5K req/s, <80% at 10K req/s
- ✅ Goroutine count stable (no goroutine leaks)
- ✅ File descriptors <1,500 (graceful connection reuse)

#### NFR-1.4 Database Performance
| Metric | Baseline (100%) | Target (150%) | Measurement |
|--------|----------------|---------------|-------------|
| **Query latency (p99)** | 10ms | **5ms** | Prometheus histogram |
| **Connection pool utilization** | 80% | 60% | pgx pool stats |
| **Transactions per second** | 1,000 | 5,000 | PostgreSQL stats |

**Measurement Method**:
- `alert_history_infra_repository_query_duration_seconds` histogram
- PostgreSQL `pg_stat_database` view

**Acceptance Criteria**:
- ✅ Database queries <5ms p99
- ✅ Connection pool utilization <60% at steady state
- ✅ No connection pool exhaustion under load

### NFR-2: Reliability Requirements (150% Target)
**Priority**: CRITICAL  
**Status**: Required

#### NFR-2.1 Availability
- **Baseline (100%)**: 99.9% uptime (43.2 minutes downtime/month)
- **Target (150%)**: **99.95% uptime** (21.6 minutes downtime/month)
- **Measurement**: Uptime monitoring (Prometheus)
- **Acceptance**: No unplanned downtime during 30-day monitoring period

#### NFR-2.2 Error Rate
- **Baseline (100%)**: <0.1% error rate
- **Target (150%)**: **<0.01% error rate**
- **Measurement**: `rate(alert_history_rest_webhook_errors_total[5m]) / rate(alert_history_rest_webhook_requests_total[5m])`
- **Acceptance**: Error rate <0.01% over 7-day period

#### NFR-2.3 Graceful Degradation
- **MUST**: Continue operation if LLM service unavailable (circuit breaker)
- **MUST**: Continue operation if publishing targets unavailable
- **MUST**: Return 207 Partial Success if some alerts fail
- **MUST**: Queue failed alerts for retry (DLQ)
- **SHOULD**: Cache classification results to reduce LLM dependency

#### NFR-2.4 Recovery
- **MUST**: Recover from panic without process crash (recovery middleware)
- **MUST**: Restore state after restart (<30s)
- **MUST**: Resume processing from DLQ after recovery
- **SHOULD**: Implement exponential backoff for retries (100ms → 5s)

### NFR-3: Security Requirements (150% Target)
**Priority**: CRITICAL  
**Status**: Required

#### NFR-3.1 OWASP Top 10 Compliance
- **A01:2021 - Broken Access Control**:
  - ✅ Rate limiting: 100 req/min per IP, 10K req/min global
  - ✅ IP whitelisting support (configurable)
  - ✅ Authentication middleware (API key, JWT)
- **A02:2021 - Cryptographic Failures**:
  - ✅ TLS 1.2+ enforcement
  - ✅ No secrets in logs (redaction)
  - ✅ Secure random for request IDs (crypto/rand)
- **A03:2021 - Injection**:
  - ✅ Parameterized SQL queries (pgx)
  - ✅ Input validation (JSON schema)
  - ✅ Content-Type validation
- **A04:2021 - Insecure Design**:
  - ✅ Defense in depth (multiple validation layers)
  - ✅ Fail-safe defaults (deny by default)
  - ✅ Principle of least privilege
- **A05:2021 - Security Misconfiguration**:
  - ✅ Secure defaults (config.yaml)
  - ✅ Security headers (X-Content-Type-Options, etc.)
  - ✅ Error message sanitization (no stack traces in prod)
- **A06:2021 - Vulnerable Components**:
  - ✅ Dependency scanning (nancy)
  - ✅ Regular updates (go get -u)
  - ✅ Go version 1.24.6+ (latest stable)
- **A07:2021 - Authentication Failures**:
  - ✅ API key authentication (HMAC-SHA256)
  - ✅ JWT token validation (RS256)
  - ✅ Failed auth logging
- **A08:2021 - Software Integrity Failures**:
  - ✅ Signature verification support (HMAC)
  - ✅ Checksum validation (webhook signatures)
- **A09:2021 - Logging Failures**:
  - ✅ Structured logging (slog JSON)
  - ✅ Audit trail (all requests logged)
  - ✅ Alerting on anomalies
- **A10:2021 - SSRF**:
  - ✅ URL validation (no internal IPs in webhooks)
  - ✅ Timeout enforcement (30s)

#### NFR-3.2 Rate Limiting
**Tier 1: Per-IP Rate Limiting**
- Limit: 100 requests/minute per IP
- Window: Sliding window (1 minute)
- Storage: Redis (distributed)
- Response: 429 Too Many Requests
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`

**Tier 2: Global Rate Limiting**
- Limit: 10,000 requests/minute (global)
- Window: Fixed window (1 minute)
- Storage: In-memory counter
- Response: 503 Service Unavailable
- Headers: `Retry-After`

**Tier 3: Authenticated Rate Limiting**
- Limit: 1,000 requests/minute per API key
- Window: Sliding window (1 minute)
- Storage: Redis
- Response: 429 Too Many Requests
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`

#### NFR-3.3 Authentication & Authorization
**Optional (configurable via environment)**:
- **API Key Authentication**:
  - Header: `X-API-Key: <key>`
  - Storage: Environment variable or K8s Secret
  - Validation: Constant-time comparison
  - Expiry: Configurable (default: never)
- **JWT Authentication**:
  - Header: `Authorization: Bearer <token>`
  - Algorithm: RS256 (RSA public key)
  - Validation: exp, iat, nbf claims
  - Issuer: Configurable (e.g., Keycloak)
- **HMAC Signature Verification**:
  - Header: `X-Webhook-Signature: sha256=<hmac>`
  - Algorithm: HMAC-SHA256
  - Secret: Shared secret from config
  - Body: Hash of raw request body

#### NFR-3.4 Input Validation
- **MUST**: Enforce max request size: 10MB
- **MUST**: Validate JSON structure (max depth: 10)
- **MUST**: Validate field types (string, number, boolean)
- **MUST**: Sanitize error messages (no raw payload in prod)
- **SHOULD**: Check for suspicious patterns (SQL keywords, script tags)

#### NFR-3.5 Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'none'
```

#### NFR-3.6 Security Scanning
- **MUST**: Pass `gosec` scan (zero high/medium issues)
- **MUST**: Pass `nancy` dependency check (zero vulnerabilities)
- **SHOULD**: Pass Trivy container scan (zero critical vulnerabilities)

### NFR-4: Observability Requirements (150% Target)
**Priority**: CRITICAL  
**Status**: Required

#### NFR-4.1 Prometheus Metrics (15+ metrics)
**Request Metrics**:
```prometheus
# Counter: Total requests by status
alert_history_rest_webhook_requests_total{method="POST", status="success|partial|failure"}

# Histogram: Request duration (seconds)
alert_history_rest_webhook_request_duration_seconds{method="POST"}

# Histogram: Payload size (bytes)
alert_history_rest_webhook_payload_size_bytes{method="POST"}

# Gauge: Active requests
alert_history_rest_webhook_active_requests{method="POST"}
```

**Processing Metrics**:
```prometheus
# Histogram: Stage duration (seconds)
alert_history_rest_webhook_stage_duration_seconds{stage="detection|parsing|validation|conversion|processing", type="alertmanager|generic"}

# Counter: Alerts received
alert_history_rest_webhook_alerts_received_total{type="alertmanager|generic"}

# Counter: Alerts processed
alert_history_rest_webhook_alerts_processed_total{type="alertmanager|generic", status="success|failure"}
```

**Error Metrics**:
```prometheus
# Counter: Errors by type
alert_history_rest_webhook_errors_total{error_type="detection|parsing|validation|processing|timeout"}

# Counter: Rate limit hits
alert_history_rest_webhook_rate_limit_hits_total{limit_type="per_ip|global"}

# Counter: Auth failures
alert_history_rest_webhook_auth_failures_total{auth_type="api_key|jwt"}
```

#### NFR-4.2 Structured Logging
**Log Format**: JSON (slog)
**Log Levels**:
- **DEBUG**: Request/response details (payload, headers)
- **INFO**: Request received, processing complete, stats
- **WARN**: Validation failures, rate limiting, partial success
- **ERROR**: Processing errors, database failures
- **FATAL**: Panic, unrecoverable errors

**Required Fields**:
```json
{
  "time": "2025-11-15T10:30:45.123Z",
  "level": "INFO",
  "msg": "Webhook processed successfully",
  "request_id": "req-abc123...",
  "trace_id": "trace-xyz789...",
  "remote_addr": "10.0.1.5:45678",
  "method": "POST",
  "path": "/webhook",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 10,
  "duration_ms": 45.2,
  "status": "success"
}
```

#### NFR-4.3 Distributed Tracing (Optional)
- **Protocol**: OpenTelemetry
- **Exporter**: OTLP (gRPC or HTTP)
- **Sampling**: 1% of requests (configurable)
- **Spans**: Detection, Parsing, Validation, Conversion, Processing (per alert)
- **Attributes**: webhook_type, alerts_count, request_id, user_agent

#### NFR-4.4 Grafana Dashboard (8+ panels)
- Panel 1: Request rate (QPS)
- Panel 2: Latency percentiles (p50, p95, p99, p99.9)
- Panel 3: Success rate (%)
- Panel 4: Error rate by type
- Panel 5: Active requests (gauge)
- Panel 6: Payload size distribution
- Panel 7: Rate limiting hits
- Panel 8: Processing stage breakdown

#### NFR-4.5 Alerting Rules (5+ rules)
- Rule 1: High error rate (>1% for 5 minutes)
- Rule 2: High latency (p99 >10ms for 5 minutes)
- Rule 3: Rate limiting active (>10 hits/s for 10 minutes)
- Rule 4: Low success rate (<99.9% for 10 minutes)
- Rule 5: High processing time (p99 >50ms for 5 minutes)

### NFR-5: Testability Requirements (150% Target)
**Priority**: CRITICAL  
**Status**: Required

#### NFR-5.1 Unit Tests
- **Target**: 95%+ code coverage
- **Count**: 50+ unit tests
- **Tools**: `go test`, `testify/assert`, `testify/mock`
- **Scope**:
  - Handler tests: 20 tests (request parsing, response formatting, error handling)
  - Middleware tests: 20 tests (rate limiting, auth, logging, metrics)
  - Error handling tests: 10 tests (panic recovery, timeout, validation)

#### NFR-5.2 Integration Tests
- **Target**: 100% critical paths covered
- **Count**: 10+ integration tests
- **Tools**: `testcontainers`, `httptest`
- **Scope**:
  - Full flow: Webhook → PostgreSQL → Publishing (5 tests)
  - Failure scenarios: DB timeout, LLM failure, publishing failure (5 tests)

#### NFR-5.3 E2E Tests
- **Target**: 100% user scenarios covered
- **Count**: 5+ E2E tests
- **Tools**: `httptest`, `testcontainers`, real services
- **Scope**:
  - Prometheus → Alert History → Rootly
  - Alertmanager → Alert History → Multi-target
  - Generic webhook → Alert History → Storage
  - Rate limiting scenarios
  - Graceful degradation (LLM failure)

#### NFR-5.4 Benchmark Tests
- **Target**: All performance targets met
- **Count**: 15+ benchmarks
- **Tools**: `go test -bench`, `pprof`
- **Scope**:
  - Request handling: 5 benchmarks (Alertmanager, Generic, batches)
  - Middleware: 5 benchmarks (rate limit, auth, logging, metrics)
  - Processing stages: 5 benchmarks (detection, parsing, validation, conversion)

#### NFR-5.5 Load Tests
- **Target**: 150% performance targets
- **Count**: 4 k6 scenarios
- **Tools**: `k6`, `Grafana`
- **Scope**:
  - Steady state: 1K req/s for 10 minutes
  - Spike: 1K → 10K → 1K req/s
  - Stress: Ramp up until breaking point (>10K target)
  - Soak: 2K req/s for 4 hours

---

## 4. INTERFACE REQUIREMENTS

### IR-1: REST API Interface
**Endpoint**: `POST /webhook`
**Content-Type**: `application/json`
**Auth**: Optional (API key or JWT)

#### IR-1.1 Request Headers
```
POST /webhook HTTP/1.1
Host: alert-history.example.com
Content-Type: application/json
Content-Length: 1234
X-Request-ID: req-abc123... (optional, generated if missing)
X-API-Key: your-api-key (optional, if auth enabled)
Authorization: Bearer <jwt-token> (optional, if JWT auth enabled)
User-Agent: Prometheus/2.45.0
X-Webhook-Signature: sha256=<hmac> (optional, for signature verification)
```

#### IR-1.2 Request Body (Alertmanager Format)
```json
{
  "version": "4",
  "groupKey": "{}:{alertname=\"HighCPU\"}",
  "status": "firing",
  "receiver": "webhook",
  "groupLabels": {
    "alertname": "HighCPU"
  },
  "commonLabels": {
    "alertname": "HighCPU",
    "severity": "warning",
    "namespace": "production"
  },
  "commonAnnotations": {
    "summary": "High CPU usage detected",
    "description": "CPU usage is above 80%"
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighCPU",
        "severity": "warning",
        "namespace": "production",
        "instance": "node-1"
      },
      "annotations": {
        "summary": "High CPU usage on node-1",
        "description": "CPU usage is 85%"
      },
      "startsAt": "2025-11-15T10:30:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?g0.expr=...",
      "fingerprint": "abc123..."
    }
  ]
}
```

#### IR-1.3 Request Body (Generic Format)
```json
{
  "alertname": "CustomAlert",
  "status": "firing",
  "severity": "critical",
  "message": "Something went wrong",
  "timestamp": "2025-11-15T10:30:00Z",
  "custom_field_1": "value1",
  "custom_field_2": 42
}
```

#### IR-1.4 Response Headers (Success)
```
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-ID: req-abc123...
X-Processing-Time: 45.2
X-Webhook-Type: alertmanager
X-Alerts-Processed: 10
Date: Fri, 15 Nov 2025 10:30:45 GMT
```

#### IR-1.5 Response Body (Success)
```json
{
  "status": "success",
  "message": "Webhook processed successfully",
  "webhook_type": "alertmanager",
  "alerts_received": 10,
  "alerts_processed": 10,
  "processing_time": "45.2ms",
  "request_id": "req-abc123..."
}
```

### IR-2: Prometheus Metrics Interface
**Endpoint**: `GET /metrics`
**Format**: Prometheus text exposition format

#### IR-2.1 Metrics Export
```prometheus
# HELP alert_history_rest_webhook_requests_total Total webhook requests
# TYPE alert_history_rest_webhook_requests_total counter
alert_history_rest_webhook_requests_total{method="POST",status="success"} 12345

# HELP alert_history_rest_webhook_request_duration_seconds Webhook request duration
# TYPE alert_history_rest_webhook_request_duration_seconds histogram
alert_history_rest_webhook_request_duration_seconds_bucket{method="POST",le="0.005"} 10000
alert_history_rest_webhook_request_duration_seconds_bucket{method="POST",le="0.01"} 11500
alert_history_rest_webhook_request_duration_seconds_bucket{method="POST",le="+Inf"} 12345
alert_history_rest_webhook_request_duration_seconds_sum{method="POST"} 55.2
alert_history_rest_webhook_request_duration_seconds_count{method="POST"} 12345

# ... (15+ metrics total)
```

### IR-3: Configuration Interface
**Format**: YAML + Environment Variables

#### IR-3.1 Configuration File (config.yaml)
```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  graceful_shutdown_timeout: 30s

webhook:
  max_request_size: 10485760  # 10MB
  request_timeout: 30s
  max_alerts_per_request: 1000
  rate_limiting:
    enabled: true
    per_ip_limit: 100  # requests per minute
    global_limit: 10000  # requests per minute
  authentication:
    enabled: false
    type: "api_key"  # api_key, jwt, hmac
    api_key: "${WEBHOOK_API_KEY}"  # from environment
  signature_verification:
    enabled: false
    secret: "${WEBHOOK_SECRET}"  # from environment

processing:
  worker_count: 8  # NumCPU * 2
  queue_size: 1000
  processing_timeout: 30s

observability:
  metrics:
    enabled: true
    path: "/metrics"
  logging:
    level: "info"  # debug, info, warn, error
    format: "json"
  tracing:
    enabled: false
    exporter: "otlp"
    endpoint: "http://jaeger:4318"
```

#### IR-3.2 Environment Variables
```bash
# Server
SERVER_PORT=8080

# Webhook
WEBHOOK_MAX_REQUEST_SIZE=10485760
WEBHOOK_RATE_LIMITING_ENABLED=true
WEBHOOK_PER_IP_LIMIT=100
WEBHOOK_AUTHENTICATION_ENABLED=false
WEBHOOK_API_KEY=your-secret-key

# Processing
PROCESSING_WORKER_COUNT=8

# Observability
OBSERVABILITY_LOGGING_LEVEL=info
```

---

## 5. DATA REQUIREMENTS

### DR-1: Input Data
- **Format**: JSON
- **Max Size**: 10MB
- **Max Alerts**: 1000 per request
- **Character Encoding**: UTF-8

### DR-2: Storage Data
- **Database**: PostgreSQL 15+
- **Table**: `alerts`
- **Retention**: Configurable (default: 90 days)
- **Indexes**: fingerprint, alertname, namespace, severity, starts_at

### DR-3: Cache Data
- **Storage**: Redis 7+
- **Keys**: 
  - Deduplication cache: `dedup:<fingerprint>` (TTL: 24h)
  - Rate limiting: `ratelimit:ip:<ip>` (TTL: 1 minute)
  - Classification cache: `classification:<fingerprint>` (TTL: 1h)
- **Max Size**: 1GB (eviction: LRU)

---

## 6. CONSTRAINTS & ASSUMPTIONS

### 6.1 Constraints
- **C1**: Must use Go 1.24.6+ (project standard)
- **C2**: Must use existing `UniversalWebhookHandler` from `internal/infrastructure/webhook`
- **C3**: Must integrate with existing processing pipeline (TN-040 to TN-045)
- **C4**: Must use PostgreSQL for storage (no alternative databases)
- **C5**: Must use Redis for caching and rate limiting (no alternative)
- **C6**: Must follow existing code structure (`cmd/`, `internal/`, `pkg/`)
- **C7**: Must maintain backward compatibility with existing webhook endpoint

### 6.2 Assumptions
- **A1**: PostgreSQL and Redis are available and healthy
- **A2**: LLM service may be unavailable (graceful degradation required)
- **A3**: Publishing targets may be unavailable (DLQ for retry)
- **A4**: Clients will retry on 429/503 responses (industry standard)
- **A5**: Network latency <1ms within cluster (same datacenter)
- **A6**: Alert payload is well-formed JSON (malformed payloads rejected)
- **A7**: Alertmanager webhook format follows v0.25+ specification

---

## 7. ACCEPTANCE CRITERIA

### 7.1 Functional Acceptance
- [ ] ✅ Accepts Alertmanager webhook format (v0.25+)
- [ ] ✅ Accepts generic webhook format (custom JSON)
- [ ] ✅ Auto-detects format with >95% accuracy (validated with 1000+ samples)
- [ ] ✅ Validates all required fields (per FR-4)
- [ ] ✅ Processes 100% of valid alerts
- [ ] ✅ Returns appropriate HTTP status codes (200, 207, 400, 429, 500, 503)
- [ ] ✅ Provides detailed error messages (field-level validation errors)
- [ ] ✅ Integrates with existing processing pipeline (TN-040 to TN-045)
- [ ] ✅ Publishes alerts to all targets (Rootly, PagerDuty, Slack, Generic)

### 7.2 Non-Functional Acceptance
- [ ] ✅ **Performance**: <5ms p99 latency, >10K req/s throughput (k6 validated)
- [ ] ✅ **Reliability**: 99.95% uptime, <0.01% error rate (30-day monitoring)
- [ ] ✅ **Security**: OWASP Top 10 compliant, rate limiting, auth support (gosec + nancy pass)
- [ ] ✅ **Observability**: 15+ Prometheus metrics, Grafana dashboard, 5+ alerting rules
- [ ] ✅ **Quality**: 95%+ test coverage, 80+ tests, zero linter warnings
- [ ] ✅ **Documentation**: 5,000+ LOC (API guide, troubleshooting, ADRs)

### 7.3 Testing Acceptance
- [ ] ✅ Unit tests: 50+ tests, 95%+ coverage, all passing
- [ ] ✅ Integration tests: 10+ tests, all passing
- [ ] ✅ E2E tests: 5+ tests, all passing
- [ ] ✅ Benchmark tests: 15+ benchmarks, all meet targets (<5ms p99)
- [ ] ✅ Load tests: 4 k6 scenarios, all pass
  - Steady state: 10K req/s for 10 min
  - Spike: 20K req/s handled
  - Stress: >10K req/s breaking point found
  - Soak: 2K req/s for 4 hours (no memory leaks)

### 7.4 Documentation Acceptance
- [ ] ✅ OpenAPI 3.0 specification complete (500+ LOC)
- [ ] ✅ API guide complete (3,000+ LOC)
- [ ] ✅ Troubleshooting guide complete (1,000+ LOC)
- [ ] ✅ 3+ ADRs created (middleware stack, rate limiting, error handling)
- [ ] ✅ Integration examples (Prometheus, Alertmanager, custom)
- [ ] ✅ Grafana dashboard created (8+ panels)
- [ ] ✅ Alerting rules created (5+ rules)

### 7.5 Quality Gate Acceptance
- [ ] ✅ Zero linter warnings (`golangci-lint`)
- [ ] ✅ Zero race conditions (`go test -race`)
- [ ] ✅ Zero memory leaks (pprof heap validated)
- [ ] ✅ Zero security issues (`gosec`, `nancy`)
- [ ] ✅ Cyclomatic complexity <10 per function
- [ ] ✅ Code review approved by maintainer
- [ ] ✅ 150% certification report complete (800+ LOC)
- [ ] ✅ Production readiness checklist complete

---

## 8. REVISION HISTORY

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 0.1 | 2025-11-15 | AI Assistant | Initial draft (Phase 0) |
| 0.2 | 2025-11-15 | AI Assistant | Added FR-1 to FR-7 |
| 0.3 | 2025-11-15 | AI Assistant | Added NFR-1 to NFR-5 |
| 0.4 | 2025-11-15 | AI Assistant | Added IR-1 to IR-3, DR-1 to DR-3 |
| 1.0 | 2025-11-15 | AI Assistant | Complete requirements (all sections) |

---

**Document Status**: ✅ COMPLETE (v1.0)  
**Next Action**: Proceed to Phase 1 - Design Document (design.md)  
**Author**: AI Assistant (Claude Sonnet 4.5)  
**Approver**: TBD (maintainer review required)

