# TN-054: Slack Webhook Publisher - Implementation Tasks (150% Quality)

**Version**: 1.0
**Date**: 2025-11-11
**Status**: 🚀 **READY FOR IMPLEMENTATION**
**Quality Target**: **150%+ (Enterprise Grade A+)**
**Estimated Effort**: 80 hours (10 days)

---

## 📑 Table of Contents

1. [Overview](#1-overview)
2. [Phase 1-3: Documentation](#phase-1-3-documentation-4h)
3. [Phase 4: Slack Webhook Client](#phase-4-slack-webhook-client-12h)
4. [Phase 5: Enhanced Publisher](#phase-5-enhanced-publisher-10h)
5. [Phase 6: Unit Tests](#phase-6-unit-tests-8h)
6. [Phase 7: Benchmarks](#phase-7-benchmarks-2h)
7. [Phase 8: Integration Tests](#phase-8-integration-tests-6h)
8. [Phase 9: Message ID Cache](#phase-9-message-id-cache-6h)
9. [Phase 10: Metrics & Observability](#phase-10-metrics--observability-6h)
10. [Phase 11: API Documentation](#phase-11-api-documentation-8h)
11. [Phase 12: PublisherFactory Integration](#phase-12-publisherfactory-integration-4h)
12. [Phase 13: K8s Examples](#phase-13-k8s-examples-4h)
13. [Phase 14: Final Validation](#phase-14-final-validation-4h)
14. [Commit Strategy](#commit-strategy)
15. [Quality Gates](#quality-gates)
16. [Timeline](#timeline)

---

## 1. Overview

### 1.1 Goal

Transform **SlackPublisher** from minimal HTTP wrapper (21 LOC, Grade D+) to **comprehensive Slack Webhook integration** (8,000+ LOC, Grade A+) achieving **150%+ quality**.

### 1.2 Success Criteria

- ✅ Full Slack Webhook API v1 integration
- ✅ Message lifecycle (post, thread reply)
- ✅ 90%+ test coverage
- ✅ 8 Prometheus metrics operational
- ✅ < 200ms p99 latency
- ✅ Grade A+ certification

### 1.3 Deliverables

| Deliverable | LOC | Status |
|-------------|-----|--------|
| **Implementation** | 1,200 | ⏳ Pending |
| **Tests** | 900+ | ⏳ Pending |
| **Documentation** | 5,000+ | ⏳ Pending |
| **K8s Examples** | 50+ | ⏳ Pending |
| **CHANGELOG** | 100+ | ⏳ Pending |
| **Total** | **7,250+** | - |

---

## Phase 1-3: Documentation (4h)

**Status**: ✅ **COMPLETE**

### Checklist

- [x] **Phase 1**: Create COMPREHENSIVE_ANALYSIS.md (2h) ✅
- [x] **Phase 2**: Create requirements.md (1h) ✅
- [x] **Phase 3**: Create design.md (1h) ✅

### Output

- ✅ `COMPREHENSIVE_ANALYSIS.md` (2,150 LOC)
- ✅ `requirements.md` (605 LOC)
- ✅ `design.md` (1,100 LOC)
- ✅ `tasks.md` (this file)

**Commit**: ✅ `docs(TN-054): Phase 1-3 comprehensive analysis, requirements, design, tasks`

---

## Phase 4: Slack Webhook Client (12h)

**Goal**: Implement complete Slack Webhook API v1 client with rate limiting, retry logic, error handling.

### 4.1 Data Models (`slack_models.go`) - 2h

**File**: `go-app/internal/infrastructure/publishing/slack_models.go`

**Checklist**:
- [ ] Define `SlackMessage` struct
  - `Text string` (fallback)
  - `Blocks []Block` (Block Kit)
  - `ThreadTS string` (for threading)
  - `Attachments []Attachment` (color)
- [ ] Define `Block` struct
  - `Type string` (header, section, divider, context)
  - `Text *Text` (plain_text or mrkdwn)
  - `Fields []Field` (2-column layout)
- [ ] Define `Text` struct
  - `Type string` ("plain_text" or "mrkdwn")
  - `Text string` (content)
- [ ] Define `Field` struct
  - `Type string` ("mrkdwn")
  - `Text string` (field content)
- [ ] Define `Attachment` struct
  - `Color string` (hex color)
  - `Text string` (attachment text)
- [ ] Define `SlackResponse` struct
  - `OK bool`
  - `TS string` (message timestamp)
  - `Error string`
- [ ] Add JSON struct tags для всех полей
- [ ] Add godoc comments для всех типов
- [ ] Helper constructors:
  - `NewHeaderBlock(text) Block`
  - `NewSectionBlock(text) Block`
  - `NewSectionFields(fields...) Block`
  - `NewDividerBlock() Block`
  - `NewAttachment(color, text) Attachment`

**Output**: 200 LOC

---

### 4.2 Error Types (`slack_errors.go`) - 1.5h

**File**: `go-app/internal/infrastructure/publishing/slack_errors.go`

**Checklist**:
- [ ] Define `SlackAPIError` struct
  - `StatusCode int`
  - `Error string`
  - `RetryAfter int` (from Retry-After header)
- [ ] Implement `Error() string` method
- [ ] Define sentinel errors:
  - `ErrMissingWebhookURL`
  - `ErrInvalidWebhookURL`
  - `ErrMessageTooLarge`
- [ ] Error classification helpers:
  - `IsRetryableError(err) bool` (429, 503, network errors)
  - `IsRateLimitError(err) bool` (429)
  - `IsPermanentError(err) bool` (400, 403, 404, 500)
- [ ] `parseSlackError(resp, body) *SlackAPIError`
  - Parse error from response body
  - Extract Retry-After header
- [ ] Add godoc comments

**Output**: 150 LOC

---

### 4.3 Webhook Client Interface (`slack_client.go`) - 3h

**File**: `go-app/internal/infrastructure/publishing/slack_client.go`

**Checklist**:
- [ ] Define `SlackWebhookClient` interface
  - `PostMessage(ctx, message) → (SlackResponse, error)`
  - `ReplyInThread(ctx, threadTS, message) → (SlackResponse, error)`
  - `Health(ctx) → error`
- [ ] Define `HTTPSlackWebhookClient` struct
  - `httpClient *http.Client`
  - `webhookURL string`
  - `rateLimiter *rate.Limiter` (1 msg/sec)
  - `logger *slog.Logger`
- [ ] Implement `NewHTTPSlackWebhookClient(url, logger) SlackWebhookClient`
  - Create http.Client with 10s timeout
  - TLS 1.2+ config
  - Connection pooling (MaxIdleConns: 10, MaxIdleConnsPerHost: 2)
  - Create rate limiter: `rate.NewLimiter(rate.Every(1*time.Second), 1)`
- [ ] Implement `PostMessage(ctx, message)`
  - Rate limit check: `rateLimiter.Wait(ctx)`
  - Marshal message to JSON
  - Build HTTP POST request
  - Call `doRequestWithRetry(ctx, req)`
  - Parse response
  - Return SlackResponse
- [ ] Implement `ReplyInThread(ctx, threadTS, message)`
  - Set `message.ThreadTS = threadTS`
  - Call `PostMessage(ctx, message)`
- [ ] Implement `Health(ctx)`
  - Post minimal test message
  - Return error if failed
- [ ] Helper: `maskWebhookURL(url) string` (mask token in logs)

**Output**: 250 LOC

---

### 4.4 Retry Logic (`slack_client.go`) - 3h

**Checklist** (continue in same file):
- [ ] Implement `doRequestWithRetry(ctx, req) → (SlackResponse, error)`
  - Max retries: 3
  - Exponential backoff: 100ms → 200ms → 400ms → 800ms → 1.6s → 5s max
  - Clone request body for each attempt
  - Execute `httpClient.Do(req)`
  - Check status code:
    - 200 → parse response, return
    - 429 → parse error, respect Retry-After, retry
    - 503 → parse error, exponential backoff, retry
    - 400, 403, 404, 500 → parse error, don't retry, return
  - Network error → check if retryable, retry if yes
  - Max retries exceeded → return last error
- [ ] Helper: `isRetryableNetworkError(err) bool`
  - Check for timeout, connection refused, DNS errors
  - Return true if retryable

**Output**: +150 LOC

**Total Phase 4 Output**: 400 LOC

---

## Phase 5: Enhanced Publisher (10h)

**Goal**: Implement business logic layer with message lifecycle management.

### 5.1 Publisher Implementation (`slack_publisher_enhanced.go`) - 4h

**File**: `go-app/internal/infrastructure/publishing/slack_publisher_enhanced.go`

**Checklist**:
- [ ] Define `EnhancedSlackPublisher` struct
  - `client SlackWebhookClient`
  - `cache MessageIDCache`
  - `metrics *SlackMetrics`
  - `formatter AlertFormatter`
  - `logger *slog.Logger`
- [ ] Implement `NewEnhancedSlackPublisher(...) AlertPublisher`
  - Initialize struct with dependencies
  - Return EnhancedSlackPublisher
- [ ] Implement `Publish(ctx, enrichedAlert, target) error`
  - Extract fingerprint from alert
  - Check cache: `cache.Get(fingerprint)`
  - Route based on status:
    - `StatusFiring` + not found → `postMessage()`
    - `StatusFiring` + found → `replyInThread()` (still firing)
    - `StatusResolved` + found → `replyInThread()` (resolved)
    - `StatusResolved` + not found → `postMessage()` (cache miss, log warning)
- [ ] Implement `Name() string` → return "Slack"

**Output**: 150 LOC

---

### 5.2 Post Message Logic (`slack_publisher_enhanced.go`) - 3h

**Checklist** (continue in same file):
- [ ] Implement `postMessage(ctx, enrichedAlert, fingerprint) error`
  - Start timer: `startTime := time.Now()`
  - Format alert: `formatter.FormatAlert(ctx, enrichedAlert, core.FormatSlack)`
  - Handle formatter error → increment `metrics.MessageErrors`, return
  - Build SlackMessage: `buildMessage(formattedPayload)`
  - Post message: `client.PostMessage(ctx, message)`
  - Handle client error → increment `metrics.MessageErrors`, record duration, return
  - Cache message timestamp: `cache.Store(fingerprint, MessageEntry{ts, ts, time.Now()})`
  - Record metrics: `metrics.MessagesPosted.Inc()`, `metrics.APIDuration.Observe()`
  - Log success
  - Return nil
- [ ] Implement `buildMessage(payload) *SlackMessage`
  - Extract `text` from payload
  - Extract `blocks` array, convert to []Block
  - Extract `attachments` array, convert to []Attachment
  - Return SlackMessage

**Output**: +100 LOC

---

### 5.3 Thread Reply Logic (`slack_publisher_enhanced.go`) - 3h

**Checklist** (continue in same file):
- [ ] Implement `replyInThread(ctx, threadTS, enrichedAlert, statusText) error`
  - Start timer: `startTime := time.Now()`
  - Build simple reply message:
    - Text: `fmt.Sprintf("%s - %s", statusText, alert.AlertName)`
    - Blocks: 1 section with markdown text (status + timestamp)
  - Reply in thread: `client.ReplyInThread(ctx, threadTS, message)`
  - Handle client error → increment `metrics.MessageErrors`, record duration, return
  - Record metrics: `metrics.ThreadReplies.Inc()`, `metrics.CacheHits.Inc()`, `metrics.APIDuration.Observe()`
  - Log success
  - Return nil
- [ ] Helper: `classifyError(err) string`
  - Check if SlackAPIError → classify by status code (rate_limit, server_error, api_error)
  - Otherwise → "network_error"

**Output**: +100 LOC

**Total Phase 5 Output**: 350 LOC

---

## Phase 6: Unit Tests (8h)

**Goal**: Comprehensive unit test coverage (90%+ target).

### 6.1 Client Tests (`slack_client_test.go`) - 4h

**File**: `go-app/internal/infrastructure/publishing/slack_client_test.go`

**Checklist**:
- [ ] Test: `TestNewHTTPSlackWebhookClient`
  - Assert client initialized correctly
  - Assert rate limiter configured (1 msg/sec)
- [ ] Test: `TestPostMessage_Success`
  - Mock Slack server (httptest.NewServer)
  - Post message
  - Assert response parsed correctly
  - Assert message_ts returned
- [ ] Test: `TestPostMessage_RateLimitError`
  - Mock server returns 429 + Retry-After: 1
  - Assert retry respects Retry-After
  - Assert metrics incremented
- [ ] Test: `TestPostMessage_ServiceUnavailable`
  - Mock server returns 503
  - Assert retry with exponential backoff
  - Assert eventual success (3rd attempt)
- [ ] Test: `TestPostMessage_BadRequest`
  - Mock server returns 400
  - Assert no retry
  - Assert error returned
- [ ] Test: `TestPostMessage_Forbidden`
  - Mock server returns 403
  - Assert no retry
  - Assert error is permanent
- [ ] Test: `TestPostMessage_NetworkError`
  - Simulate network timeout
  - Assert retry
  - Assert max retries exceeded
- [ ] Test: `TestReplyInThread_Success`
  - Mock server
  - Reply in thread
  - Assert thread_ts parameter set
- [ ] Test: `TestReplyInThread_ThreadTSSet`
  - Assert message.ThreadTS = threadTS
- [ ] Test: `TestHealth_Success`
  - Mock server
  - Call Health()
  - Assert no error
- [ ] Test: `TestRetryLogic_ExponentialBackoff`
  - Mock server fails 2 times, succeeds 3rd
  - Assert backoff delays: 100ms, 200ms
- [ ] Test: `TestRetryLogic_MaxRetriesExceeded`
  - Mock server always fails
  - Assert max 3 attempts
  - Assert error returned
- [ ] Test: `TestRetryLogic_RespectRetryAfter`
  - Mock server returns 429 + Retry-After: 2
  - Assert wait 2 seconds before retry
- [ ] Test: `TestMaskWebhookURL`
  - Assert token masked in logs

**Output**: 500 LOC

---

### 6.2 Publisher Tests (`slack_publisher_test.go`) - 2h

**File**: `go-app/internal/infrastructure/publishing/slack_publisher_test.go`

**Checklist**:
- [ ] Test: `TestPublish_NewFiringAlert`
  - Mock client, cache, formatter
  - Publish firing alert (cache miss)
  - Assert postMessage() called
  - Assert cache.Store() called
  - Assert metrics recorded
- [ ] Test: `TestPublish_ResolvedAlert`
  - Mock dependencies
  - Publish resolved alert (cache hit)
  - Assert replyInThread() called
  - Assert "🟢 Resolved" message
- [ ] Test: `TestPublish_StillFiringAlert`
  - Cache hit
  - Publish firing alert
  - Assert replyInThread() called
  - Assert "🔴 Still firing" message
- [ ] Test: `TestPublish_CacheHit`
  - Cache returns entry
  - Assert cache_hits_total incremented
- [ ] Test: `TestPublish_CacheMiss`
  - Cache returns not found
  - Assert cache_misses_total incremented
- [ ] Test: `TestPublish_FormatterError`
  - Formatter returns error
  - Assert message_errors_total incremented
  - Assert error propagated
- [ ] Test: `TestPublish_ClientError`
  - Client returns error
  - Assert message_errors_total incremented
  - Assert error propagated
- [ ] Test: `TestPublish_MetricsRecorded`
  - Publish successful alert
  - Assert metrics_posted_total incremented
  - Assert api_duration recorded
- [ ] Test: `TestName`
  - Assert Name() returns "Slack"

**Output**: 300 LOC

---

### 6.3 Error Tests (`slack_errors_test.go`) - 1h

**File**: `go-app/internal/infrastructure/publishing/slack_errors_test.go`

**Checklist**:
- [ ] Test: `TestSlackAPIError_Error`
  - Assert error message format
  - Assert Retry-After included
- [ ] Test: `TestIsRetryableError`
  - Test 429 → retryable
  - Test 503 → retryable
  - Test network error → retryable
  - Test 400 → not retryable
- [ ] Test: `TestIsRateLimitError`
  - Test 429 → true
  - Test others → false
- [ ] Test: `TestIsPermanentError`
  - Test 400, 403, 404, 500 → true
  - Test 429, 503 → false
- [ ] Test: `TestParseSlackError`
  - Mock HTTP response
  - Assert error parsed correctly
  - Assert Retry-After extracted

**Output**: 100 LOC

---

## Phase 7: Benchmarks (2h)

**Goal**: Performance validation.

### 7.1 Benchmarks (`slack_bench_test.go`) - 2h

**File**: `go-app/internal/infrastructure/publishing/slack_bench_test.go`

**Checklist**:
- [ ] Benchmark: `BenchmarkPostMessage`
  - Mock server
  - Post message in loop
  - Measure throughput
- [ ] Benchmark: `BenchmarkReplyInThread`
  - Mock server
  - Reply in thread in loop
  - Measure throughput
- [ ] Benchmark: `BenchmarkCacheGet`
  - Populate cache
  - Get entries in loop
  - Target: < 50ns
- [ ] Benchmark: `BenchmarkCacheStore`
  - Store entries in loop
  - Target: < 100ns
- [ ] Benchmark: `BenchmarkRateLimiterWait`
  - Wait for tokens in loop
  - Measure overhead
  - Target: < 1ms
- [ ] Benchmark: `BenchmarkFormatMessage`
  - Format alert in loop
  - Measure formatter overhead
- [ ] Benchmark: `BenchmarkBuildMessage`
  - Build SlackMessage from payload
  - Measure conversion overhead
- [ ] Benchmark: `BenchmarkConcurrentPublish`
  - Publish from 10 goroutines
  - Measure concurrent throughput

**Output**: 200 LOC

---

## Phase 8: Integration Tests (6h)

**Goal**: End-to-end scenarios.

### 8.1 Integration Tests (`slack_integration_test.go`) - 4h

**File**: `go-app/internal/infrastructure/publishing/slack_integration_test.go`

**Checklist**:
- [ ] Test: `TestIntegration_PostAndThreadReply`
  - Mock Slack server
  - Post firing alert
  - Assert message posted
  - Reply with resolved alert
  - Assert thread reply posted
  - Assert cache hit
- [ ] Test: `TestIntegration_PublisherFactory`
  - Create target with slack type
  - Call factory.CreatePublisher(target)
  - Assert EnhancedSlackPublisher returned
  - Assert shared cache + metrics
- [ ] Test: `TestIntegration_MetricsRecording`
  - Publish alerts
  - Query Prometheus metrics
  - Assert metrics recorded correctly
- [ ] Test: `TestIntegration_RealSlackWebhook` (optional, skip in CI)
  - Use real Slack webhook URL (env var)
  - Post message
  - Assert success
  - Document setup in README

**Output**: 200 LOC

---

### 8.2 Cache Tests (`slack_cache_test.go`) - 2h

**File**: `go-app/internal/infrastructure/publishing/slack_cache_test.go`

**Checklist**:
- [ ] Test: `TestCache_StoreAndGet`
  - Store entry
  - Get entry
  - Assert values match
- [ ] Test: `TestCache_TTLExpired`
  - Store entry with CreatedAt = now - 25h
  - Get entry
  - Assert not found (expired)
- [ ] Test: `TestCache_Delete`
  - Store entry
  - Delete entry
  - Get entry
  - Assert not found
- [ ] Test: `TestCache_Size`
  - Store 10 entries
  - Assert Size() = 10
- [ ] Test: `TestCache_CleanupWorker`
  - Store 5 expired entries
  - Start cleanup worker
  - Wait 6 minutes
  - Assert entries deleted
  - Assert Size() = 0
- [ ] Test: `TestCache_ConcurrentAccess`
  - Run 10 goroutines (5 writers, 5 readers)
  - 1000 operations each
  - Assert no race conditions
- [ ] Test: `TestCache_GracefulShutdown`
  - Start cleanup worker
  - Cancel context
  - Assert worker stopped

**Output**: 150 LOC

---

## Phase 9: Message ID Cache (6h)

**Goal**: Implement message tracking cache.

### 9.1 Cache Implementation (`slack_cache.go`) - 4h

**File**: `go-app/internal/infrastructure/publishing/slack_cache.go`

**Checklist**:
- [ ] Define `MessageIDCache` interface
  - `Store(fingerprint, entry)`
  - `Get(fingerprint) → (entry, found)`
  - `Delete(fingerprint)`
  - `Size() int`
  - `StartCleanup(ctx)`
- [ ] Define `MessageEntry` struct
  - `MessageTS string`
  - `ThreadTS string`
  - `CreatedAt time.Time`
- [ ] Define `DefaultMessageIDCache` struct
  - `entries sync.Map` (map[string]*MessageEntry)
  - `ttl time.Duration`
  - `logger *slog.Logger`
- [ ] Implement `NewMessageIDCache(ttl, logger) MessageIDCache`
  - Initialize cache
  - Return DefaultMessageIDCache
- [ ] Implement `Store(fingerprint, entry)`
  - `entries.Store(fingerprint, entry)`
  - Log debug
- [ ] Implement `Get(fingerprint) → (entry, found)`
  - `entries.Load(fingerprint)`
  - Check TTL: `time.Since(entry.CreatedAt) > ttl` → delete, return not found
  - Return entry
- [ ] Implement `Delete(fingerprint)`
  - `entries.Delete(fingerprint)`
- [ ] Implement `Size() int`
  - Count entries using `entries.Range()`
- [ ] Implement `StartCleanup(ctx)`
  - Create ticker (5 minutes)
  - Loop: select ctx.Done() or ticker
  - Call `cleanup()`
  - Log completed
- [ ] Implement `cleanup()`
  - Range over entries
  - Check TTL, delete expired
  - Log deleted count

**Output**: 150 LOC

---

### 9.2 Cache Integration - 2h

**Checklist**:
- [ ] Update `EnhancedSlackPublisher` to use cache
  - Pass cache to constructor
  - Store message_ts after posting
  - Get message_ts before replying
- [ ] Add cache metrics recording
  - Increment cache_hits_total on hit
  - Increment cache_misses_total on miss
  - Update cache_size gauge periodically

---

## Phase 10: Metrics & Observability (6h)

**Goal**: Implement 8 Prometheus metrics.

### 10.1 Metrics Implementation (`slack_metrics.go`) - 3h

**File**: `go-app/internal/infrastructure/publishing/slack_metrics.go`

**Checklist**:
- [ ] Define `SlackMetrics` struct
  - `MessagesPosted *prometheus.CounterVec` (by status)
  - `MessageErrors *prometheus.CounterVec` (by error_type)
  - `APIDuration *prometheus.HistogramVec` (by operation, status)
  - `CacheHits prometheus.Counter`
  - `CacheMisses prometheus.Counter`
  - `CacheSize prometheus.Gauge`
  - `RateLimitHits prometheus.Counter`
  - `ThreadReplies prometheus.Counter`
- [ ] Implement `NewSlackMetrics(registry) *SlackMetrics`
  - Initialize all 8 metrics
  - Register with Prometheus registry
  - Return SlackMetrics
- [ ] Add godoc comments для всех метрик

**Output**: 100 LOC

---

### 10.2 Metrics Integration - 3h

**Checklist**:
- [ ] Update `EnhancedSlackPublisher` to record metrics
  - Increment `MessagesPosted` on success
  - Increment `MessageErrors` on error
  - Observe `APIDuration` for all operations
  - Increment `CacheHits` / `CacheMisses` on cache lookup
  - Increment `ThreadReplies` on thread reply
- [ ] Update `HTTPSlackWebhookClient` to record metrics
  - Increment `RateLimitHits` when rate limiter blocks
- [ ] Update cache to record metrics
  - Update `CacheSize` gauge on Store/Delete
- [ ] Add structured logging
  - Log INFO on message posted
  - Log WARN on rate limit hit
  - Log ERROR on API error

---

## Phase 11: API Documentation (8h)

**Goal**: Comprehensive API documentation.

### 11.1 README (`README_SLACK.md`) - 4h

**File**: `go-app/internal/infrastructure/publishing/README_SLACK.md`

**Checklist**:
- [ ] Section: **Overview**
  - What is Slack Publisher
  - Key features (Block Kit, threading, rate limiting)
- [ ] Section: **Quick Start**
  - 5-minute setup guide
  - Example: Post message to Slack
- [ ] Section: **Architecture**
  - 5-layer design diagram
  - Component responsibilities
- [ ] Section: **Configuration**
  - Environment variables
  - K8s Secret format
- [ ] Section: **Usage**
  - Code examples:
    - Create client
    - Post message
    - Reply in thread
    - Error handling
- [ ] Section: **Block Kit**
  - Supported blocks (header, section, divider)
  - Block Kit builder examples
  - Character limits
- [ ] Section: **Rate Limiting**
  - Slack limit (1 msg/sec)
  - Token bucket algorithm
  - How it works
- [ ] Section: **Retry Logic**
  - Exponential backoff
  - Retryable vs permanent errors
  - Max retries
- [ ] Section: **Metrics**
  - List all 8 metrics
  - Example PromQL queries
  - Grafana dashboard queries
- [ ] Section: **Troubleshooting**
  - Common errors + solutions:
    - "missing webhook URL"
    - "429 rate limit"
    - "invalid payload"
    - "webhook revoked (404)"
- [ ] Section: **Performance**
  - Latency targets
  - Throughput limits
  - Optimization tips

**Output**: 1,000 LOC

---

### 11.2 Integration Guide (`INTEGRATION_GUIDE.md`) - 4h

**File**: `tasks/go-migration-analysis/TN-054-slack-publisher/INTEGRATION_GUIDE.md`

**Checklist**:
- [ ] Section: **Prerequisites**
  - TN-046, TN-047, TN-050, TN-051 complete
- [ ] Section: **Step 1: Create Slack Webhook**
  - Navigate to Slack settings
  - Create Incoming Webhook
  - Copy webhook URL
- [ ] Section: **Step 2: Create K8s Secret**
  - Example YAML
  - Apply to cluster
  - Verify with kubectl
- [ ] Section: **Step 3: Configure RBAC**
  - ServiceAccount permissions
  - Role binding example
- [ ] Section: **Step 4: Deploy Service**
  - Helm values
  - kubectl apply
  - Verify deployment
- [ ] Section: **Step 5: Verify Integration**
  - Send test alert
  - Check Slack channel
  - Verify metrics
- [ ] Section: **Step 6: Production Deployment**
  - Gradual rollout (10%→50%→100%)
  - Monitoring checklist
  - Alerting rules

**Output**: 500 LOC

---

## Phase 12: PublisherFactory Integration (4h)

**Goal**: Integrate with existing PublisherFactory.

### 12.1 Factory Updates (`publisher.go`) - 2h

**File**: `go-app/internal/infrastructure/publishing/publisher.go`

**Checklist**:
- [ ] Update `DefaultPublisherFactory` struct
  - Add `slackCache MessageIDCache` (shared across all Slack publishers)
  - Add `slackMetrics *SlackMetrics` (shared)
- [ ] Update `NewDefaultPublisherFactory()`
  - Initialize `slackCache = NewMessageIDCache(24*time.Hour, logger)`
  - Initialize `slackMetrics = NewSlackMetrics(registry)`
  - Start cache cleanup worker: `go slackCache.StartCleanup(ctx)`
- [ ] Update `CreatePublisher()` method
  - Add case for `TargetTypeSlack`:
    - Extract webhook URL from target
    - Create `HTTPSlackWebhookClient`
    - Create `EnhancedSlackPublisher` with shared cache + metrics
    - Return publisher

**Output**: 100 LOC

---

### 12.2 Integration Testing - 2h

**Checklist**:
- [ ] Test factory creates Slack publisher
- [ ] Test shared cache across multiple publishers
- [ ] Test shared metrics
- [ ] Test graceful shutdown (stop cache cleanup worker)

---

## Phase 13: K8s Examples (4h)

**Goal**: K8s Secret examples and deployment guide.

### 13.1 K8s Secret Example (`slack-secret-example.yaml`) - 2h

**File**: `examples/k8s/slack-secret-example.yaml`

**Checklist**:
- [ ] Example 1: Production Slack (all alerts)
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: slack-alerts-prod
    namespace: alert-history
    labels:
      publishing-target: "true"
  type: Opaque
  stringData:
    target.json: |
      {
        "name": "slack-alerts-prod",
        "type": "slack",
        "url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
        "format": "slack"
      }
  ```
- [ ] Example 2: Critical Alerts Only
  ```yaml
  # With filter: only critical severity
  target.json: |
    {
      "name": "slack-alerts-critical",
      "type": "slack",
      "url": "https://hooks.slack.com/services/...",
      "format": "slack",
      "filters": {
        "severity": ["critical"]
      }
    }
  ```
- [ ] Example 3: Platform Team Channel
  ```yaml
  # With namespace filter
  target.json: |
    {
      "name": "slack-platform-team",
      "type": "slack",
      "url": "https://hooks.slack.com/services/...",
      "format": "slack",
      "filters": {
        "namespace": ["prod", "staging"]
      }
    }
  ```
- [ ] Add comments explaining each field

**Output**: 50 LOC

---

### 13.2 Deployment Guide - 2h

**Checklist**:
- [ ] Document RBAC requirements
- [ ] Document Secret format
- [ ] Document discovery mechanism (TN-047 label selector)
- [ ] Example kubectl commands
- [ ] Verification steps

---

## Phase 14: Final Validation (4h)

**Goal**: Validate implementation before merge.

### 14.1 Build Validation - 1h

**Checklist**:
- [ ] Run `cd go-app && go build ./...`
  - Assert zero compile errors
- [ ] Run `golangci-lint run ./internal/infrastructure/publishing/`
  - Assert zero linter errors
- [ ] Run `go mod tidy`
  - Assert dependencies resolved

---

### 14.2 Test Execution - 2h

**Checklist**:
- [ ] Run unit tests
  ```bash
  go test ./internal/infrastructure/publishing/ -v -run ".*Slack.*"
  ```
  - Assert 100% pass rate (all tests passing)
- [ ] Run benchmarks
  ```bash
  go test ./internal/infrastructure/publishing/ -bench=. -benchmem -run=^$ | grep Slack
  ```
  - Assert performance targets met
- [ ] Run with race detector
  ```bash
  go test ./internal/infrastructure/publishing/ -race -run ".*Slack.*"
  ```
  - Assert zero race conditions

---

### 14.3 Coverage Check - 30min

**Checklist**:
- [ ] Generate coverage report
  ```bash
  go test ./internal/infrastructure/publishing/ -coverprofile=coverage_slack.out -run ".*Slack.*"
  go tool cover -func=coverage_slack.out | grep slack
  ```
- [ ] Check coverage per file:
  - `slack_models.go`: Target 80%+ (mostly structs, low priority)
  - `slack_errors.go`: Target 90%+ (critical error handling)
  - `slack_client.go`: Target 90%+ (critical API client)
  - `slack_publisher_enhanced.go`: Target 90%+ (critical business logic)
  - `slack_cache.go`: Target 90%+ (critical cache operations)
  - `slack_metrics.go`: Target 80%+ (mostly registration, low priority)
- [ ] **Pass Criteria**: Average 80%+ (pragmatic, high-value paths)
- [ ] Document coverage in COMPLETION_REPORT.md

---

### 14.4 Final Checklist - 30min

**Checklist**:
- [ ] All acceptance criteria met (24/24 from requirements.md)
- [ ] Zero breaking changes to existing code
- [ ] Zero technical debt introduced
- [ ] All tests passing (100% pass rate)
- [ ] Coverage ≥ 80% (pragmatic target)
- [ ] Documentation complete (5,000+ LOC)
- [ ] K8s examples created
- [ ] CHANGELOG.md updated
- [ ] tasks.md marked complete
- [ ] Grade A+ certification earned (150%+ quality)

---

## Commit Strategy

### Commit 1: Documentation

```bash
git add tasks/go-migration-analysis/TN-054-slack-publisher/
git commit -m "docs(TN-054): Phase 1-3 comprehensive analysis, requirements, design, tasks"
```

**Files**: COMPREHENSIVE_ANALYSIS.md, requirements.md, design.md, tasks.md

---

### Commit 2: Core Implementation

```bash
git add go-app/internal/infrastructure/publishing/slack_*.go
git commit -m "feat(TN-054): Phase 4-5 Slack webhook client and enhanced publisher"
```

**Files**: slack_models.go, slack_errors.go, slack_client.go, slack_publisher_enhanced.go

---

### Commit 3: Testing Suite

```bash
git add go-app/internal/infrastructure/publishing/*_test.go
git commit -m "test(TN-054): Phase 6-8 comprehensive test suite (90%+ coverage)"
```

**Files**: slack_client_test.go, slack_publisher_test.go, slack_errors_test.go, slack_cache_test.go, slack_bench_test.go, slack_integration_test.go

---

### Commit 4: Cache & Metrics

```bash
git add go-app/internal/infrastructure/publishing/slack_cache.go
git add go-app/internal/infrastructure/publishing/slack_metrics.go
git commit -m "feat(TN-054): Phase 9-10 message ID cache and Prometheus metrics"
```

**Files**: slack_cache.go, slack_metrics.go

---

### Commit 5: API Documentation

```bash
git add go-app/internal/infrastructure/publishing/README_SLACK.md
git add tasks/go-migration-analysis/TN-054-slack-publisher/INTEGRATION_GUIDE.md
git commit -m "docs(TN-054): Phase 11 API documentation and integration guide"
```

**Files**: README_SLACK.md, INTEGRATION_GUIDE.md

---

### Commit 6: Integration

```bash
git add go-app/internal/infrastructure/publishing/publisher.go
git add examples/k8s/slack-secret-example.yaml
git commit -m "feat(TN-054): Phase 12-13 PublisherFactory integration and K8s examples"
```

**Files**: publisher.go (updates), slack-secret-example.yaml

---

### Commit 7: Finalization

```bash
git add CHANGELOG.md
git add tasks/go-migration-analysis/tasks.md
git add tasks/go-migration-analysis/TN-054-slack-publisher/COMPLETION_REPORT.md
git commit -m "docs(TN-054): Update CHANGELOG and mark task complete (150% quality, Grade A+)"
```

**Files**: CHANGELOG.md, tasks.md, COMPLETION_REPORT.md

---

## Quality Gates

### Gate 1: Build Validation (After Phase 5)

**Command**: `cd go-app && go build ./...`

**Pass Criteria**: Zero compile errors

**Action if Failed**: Fix compilation errors before proceeding

---

### Gate 2: Linter Validation (After Phase 5)

**Command**: `golangci-lint run ./internal/infrastructure/publishing/`

**Pass Criteria**: Zero linter errors

**Action if Failed**: Fix linter warnings before proceeding

---

### Gate 3: Test Execution (After Phase 8)

**Command**: `go test ./internal/infrastructure/publishing/ -v -run ".*Slack.*"`

**Pass Criteria**: 100% pass rate (all tests passing)

**Action if Failed**: Fix failing tests before proceeding

---

### Gate 4: Coverage Check (After Phase 8)

**Command**:
```bash
go test ./internal/infrastructure/publishing/ -coverprofile=coverage_slack.out -run ".*Slack.*"
go tool cover -func=coverage_slack.out | grep slack
```

**Pass Criteria**: Average 80%+ coverage (pragmatic target)

**Action if Failed**: Add tests for critical uncovered paths

---

### Gate 5: Performance Validation (After Phase 7)

**Command**: `go test ./internal/infrastructure/publishing/ -bench=. -benchmem -run=^$ | grep Slack`

**Pass Criteria**:
- PostMessage: < 200ms p99
- Cache operations: < 50ns
- Rate limiter: < 1ms overhead

**Action if Failed**: Optimize hot paths

---

### Gate 6: Integration Validation (After Phase 12)

**Checklist**:
- [ ] PublisherFactory creates EnhancedSlackPublisher from K8s Secret
- [ ] Shared cache works across multiple publishers
- [ ] Metrics recorded in Prometheus format
- [ ] Zero breaking changes to existing code

**Action if Failed**: Fix integration issues

---

## Timeline

### Week 1 (Days 1-5) - Implementation & Testing

**Day 1** (8h):
- ✅ Phase 1-3: Documentation (4h) - **COMPLETE**
- ⏳ Phase 4: Slack Webhook Client (4h of 12h)

**Day 2** (8h):
- ⏳ Phase 4: Slack Webhook Client (remaining 8h)
- ⏳ Phase 5: Enhanced Publisher (2h of 10h)

**Day 3** (8h):
- ⏳ Phase 5: Enhanced Publisher (remaining 8h)
- ⏳ Phase 6: Unit Tests (2h of 8h)

**Day 4** (8h):
- ⏳ Phase 6: Unit Tests (remaining 6h)
- ⏳ Phase 7: Benchmarks (2h)

**Day 5** (8h):
- ⏳ Phase 8: Integration Tests (6h)
- ⏳ Phase 9: Message ID Cache (2h of 6h)

---

### Week 2 (Days 6-10) - Metrics, Docs, Integration

**Day 6** (8h):
- ⏳ Phase 9: Message ID Cache (remaining 4h)
- ⏳ Phase 10: Metrics & Observability (4h of 6h)

**Day 7** (8h):
- ⏳ Phase 10: Metrics & Observability (remaining 2h)
- ⏳ Phase 11: API Documentation (6h of 8h)

**Day 8** (8h):
- ⏳ Phase 11: API Documentation (remaining 2h)
- ⏳ Phase 12: PublisherFactory Integration (4h)
- ⏳ Phase 13: K8s Examples (2h of 4h)

**Day 9** (8h):
- ⏳ Phase 13: K8s Examples (remaining 2h)
- ⏳ Phase 14: Final Validation (4h)
- ⏳ Buffer: Final review (2h)

**Day 10** (4h):
- ⏳ CHANGELOG update (1h)
- ⏳ COMPLETION_REPORT (1h)
- ⏳ Merge to main (1h)
- ⏳ Push to origin (1h)

---

### Milestones

| Milestone | Date | Status |
|-----------|------|--------|
| ✅ Documentation Complete | Day 1 | **COMPLETE** |
| ⏳ Core Implementation Complete | Day 3 | Pending |
| ⏳ Testing Complete (90%+ coverage) | Day 5 | Pending |
| ⏳ Integration Complete | Day 8 | Pending |
| ⏳ Production-Ready (Grade A+) | Day 10 | Pending |

---

## 📋 TASKS DOCUMENT COMPLETE

**Status**: ✅ **PHASE 1-3 COMPLETE**, READY FOR PHASE 4

**Deliverable**: 850+ LOC comprehensive implementation tasks

**Next Phase**: Begin Phase 4 (Slack Webhook Client implementation)

**Quality Level**: **150% (Enterprise Grade A+)**

**Progress**: **Phases 1-3 / 14 complete** (21%)

---

**Date**: 2025-11-11
**Version**: 1.0
**Prepared By**: AI Architect (following TN-052/TN-053 success patterns)
**Estimated Completion**: 10 days (80 hours)
