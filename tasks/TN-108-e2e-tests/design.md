# TN-108: E2E Tests - Technical Design

**Status**: In Progress
**Last Updated**: 2025-11-30

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                 E2E Test Framework                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  │
│  │ Test Runner  │──▶│ Infra Setup  │──▶│ Test Suite   │  │
│  └──────────────┘   └──────────────┘   └──────────────┘  │
│         │                   │                   │          │
│         ▼                   ▼                   ▼          │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  │
│  │  TestMain()  │   │ PostgreSQL   │   │ Scenarios    │  │
│  │  + Cleanup   │   │ Redis        │   │ Assertions   │  │
│  └──────────────┘   │ Mock LLM     │   │ Verifications│  │
│                     │ Mock Targets │   └──────────────┘  │
│                     └──────────────┘                      │
└─────────────────────────────────────────────────────────────┘
         │                     │                     │
         ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Application  │◀──▶│ Testcontainers│◀──▶│ Mock Services│
│ Under Test   │    │ (PG + Redis)  │    │ (LLM + Pub)  │
└──────────────┘    └──────────────┘    └──────────────┘
```

---

## Component Design

### 1. Test Infrastructure (Existing from TN-107)

**Already Implemented** ✅:
- `TestInfrastructure` - manages PostgreSQL, Redis containers
- `MockLLMServer` - simulates LLM responses
- `APITestHelper` - HTTP request utilities
- `Fixtures` - test data management

**To Add**:
- `MockPublishingTargets` - simulate Slack, PagerDuty, Rootly webhooks

---

### 2. Mock Publishing Targets

#### Design

```go
// MockPublishingTarget simulates external publishing endpoints
type MockPublishingTarget struct {
	Server      *httptest.Server
	Name        string
	Type        string // "slack", "pagerduty", "rootly"
	Requests    []*http.Request
	Responses   []MockResponse
	mu          sync.Mutex
}

// MockResponse configures target response
type MockResponse struct {
	StatusCode int
	Body       interface{}
	Delay      time.Duration
	ErrorRate  float64 // 0.0-1.0 probability
}

// NewMockPublishingTarget creates mock target
func NewMockPublishingTarget(name, targetType string) *MockPublishingTarget

// AddResponse configures response for next request
func (m *MockPublishingTarget) AddResponse(resp MockResponse)

// GetRequests returns all received requests
func (m *MockPublishingTarget) GetRequests() []*http.Request

// VerifyRequest asserts request was received
func (m *MockPublishingTarget) VerifyRequest(t *testing.T, expectedBody map[string]interface{})

// Close stops mock server
func (m *MockPublishingTarget) Close()
```

#### Mock Types

1. **Mock Slack**
   - Endpoint: POST /services/T00/B00/XXX
   - Response: `{"ok": true}`
   - Validates: JSON body, blocks format

2. **Mock PagerDuty**
   - Endpoint: POST /v2/enqueue
   - Response: `{"status": "success", "dedup_key": "xxx"}`
   - Validates: routing_key, event_action, payload

3. **Mock Rootly**
   - Endpoint: POST /v1/incidents
   - Response: `{"data": {"id": "inc_123"}}`
   - Validates: title, severity, summary

---

### 3. E2E Test Scenarios

#### Test Structure

```go
func TestE2E_AlertIngestion_HappyPath(t *testing.T) {
	// 1. Setup infrastructure
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// 2. Prepare test data
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("HighCPU", "critical")

	// 3. Execute test
	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 4. Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. Verify side effects
	alerts, err := helper.QueryAlerts(ctx, map[string]string{
		"alertname": "HighCPU",
	})
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "critical", alerts[0].Severity)
}
```

#### Scenario Categories

1. **Alert Ingestion (5 scenarios)**
   - Happy path: Alertmanager format → storage
   - Duplicate detection: same fingerprint updates
   - Batch ingestion: multiple alerts
   - Invalid format: 400 error
   - Missing fields: 422 error

2. **Classification Pipeline (5 scenarios)**
   - First time: LLM call → enriched
   - Cache hit L1: memory cache
   - Cache hit L2: Redis cache
   - LLM timeout: fallback
   - LLM unavailable: graceful degradation

3. **Publishing Flows (5 scenarios)**
   - Single target: Slack success
   - Multi-target: parallel fanout
   - Partial failure: 207 response
   - Retry logic: exponential backoff
   - Circuit breaker: unhealthy target skipped

4. **History & Query (3 scenarios)**
   - Pagination: limit/offset
   - Filtering: severity, namespace
   - Aggregation: stats, top alerts

5. **Error Handling (2 scenarios)**
   - LLM timeout: graceful
   - Database failure: 503 error

**Total**: 20 E2E scenarios

---

### 4. Test Execution Flow

#### Per-Test Flow

```
1. SetupTestInfrastructure()
   ├─ Start PostgreSQL container
   ├─ Start Redis container
   ├─ Start Mock LLM
   └─ Start Mock Publishing Targets

2. ResetInfrastructure()
   ├─ Clear PostgreSQL tables
   ├─ Flush Redis
   └─ Clear mock request history

3. Execute Test Scenario
   ├─ Send HTTP request
   ├─ Assert response
   └─ Verify side effects

4. Teardown (defer)
   ├─ Stop containers
   └─ Clean up resources
```

#### TestMain Flow

```go
func TestMain(m *testing.M) {
	// Global setup (if needed)
	exitCode := m.Run()
	// Global teardown (if needed)
	os.Exit(exitCode)
}
```

---

### 5. Helper Functions

#### API Helpers (Existing)

```go
// MakeRequest performs HTTP request
func (h *APITestHelper) MakeRequest(method, path string, body interface{}) (*http.Response, error)

// AssertStatus asserts HTTP status code
func (h *APITestHelper) AssertStatus(t *testing.T, resp *http.Response, expected int)

// AssertJSONBody asserts JSON response body
func (h *APITestHelper) AssertJSONBody(t *testing.T, resp *http.Response, expected interface{})
```

#### Database Helpers (New)

```go
// QueryAlerts queries alerts from database
func (h *APITestHelper) QueryAlerts(ctx context.Context, filters map[string]string) ([]*Alert, error)

// GetAlertByFingerprint gets single alert
func (h *APITestHelper) GetAlertByFingerprint(ctx context.Context, fp string) (*Alert, error)

// SeedAlerts inserts test alerts into database
func (h *APITestHelper) SeedAlerts(ctx context.Context, alerts ...*Alert) error

// CountAlerts returns number of alerts matching filters
func (h *APITestHelper) CountAlerts(ctx context.Context, filters map[string]string) (int, error)
```

#### Cache Helpers (New)

```go
// GetCacheKey gets value from Redis
func (h *APITestHelper) GetCacheKey(ctx context.Context, key string) (string, error)

// VerifyCacheHit asserts cache hit metric incremented
func (h *APITestHelper) VerifyCacheHit(t *testing.T, metricName string, expectedCount int)

// FlushCache clears Redis cache
func (h *APITestHelper) FlushCache(ctx context.Context) error
```

#### Publishing Helpers (New)

```go
// SetupMockTargets creates mock publishing targets
func (h *APITestHelper) SetupMockTargets() map[string]*MockPublishingTarget

// VerifyPublished asserts alert was published to targets
func (h *APITestHelper) VerifyPublished(t *testing.T, fingerprint string, targets ...string) error

// GetPublishingResults retrieves publishing history for alert
func (h *APITestHelper) GetPublishingResults(ctx context.Context, fingerprint string) ([]PublishResult, error)
```

---

### 6. Assertion Patterns

#### Response Assertions

```go
// Assert HTTP status
assert.Equal(t, http.StatusOK, resp.StatusCode)

// Assert JSON response
var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
assert.Equal(t, "success", result["status"])

// Assert error response
assert.Contains(t, result["message"], "expected error text")
```

#### Database Assertions

```go
// Assert alert stored
alerts, err := helper.QueryAlerts(ctx, map[string]string{"alertname": "TestAlert"})
require.NoError(t, err)
assert.Len(t, alerts, 1)
assert.Equal(t, "firing", alerts[0].Status)

// Assert count
count, err := helper.CountAlerts(ctx, nil)
require.NoError(t, err)
assert.Equal(t, 5, count)
```

#### Cache Assertions

```go
// Assert cache hit
val, err := helper.GetCacheKey(ctx, "classification:fp_123")
require.NoError(t, err)
assert.NotEmpty(t, val)

// Assert cache miss
_, err := helper.GetCacheKey(ctx, "nonexistent")
assert.Error(t, err)
```

#### Publishing Assertions

```go
// Assert published
mockSlack.VerifyRequest(t, map[string]interface{}{
	"text": "Alert: HighCPU",
})

// Assert parallel fanout
assert.Len(t, mockSlack.GetRequests(), 1)
assert.Len(t, mockPagerDuty.GetRequests(), 1)
assert.Len(t, mockRootly.GetRequests(), 1)
```

---

### 7. Test Data Management

#### Fixtures (Existing)

```go
fixtures := NewFixtures()

// Load from JSON
alerts, err := fixtures.LoadAlerts("sample_alerts.json")

// Build dynamically
alert := NewTestAlert("HighCPU").
	WithSeverity("critical").
	WithNamespace("production").
	WithLabel("instance", "web-01")
```

#### Webhook Builders (New)

```go
// Alertmanager webhook
webhook := NewAlertmanagerWebhook().
	AddFiringAlert("HighCPU", "critical").
	AddResolvedAlert("OldAlert", "warning")

// Prometheus webhook
webhook := NewPrometheusWebhook().
	AddFiringAlert("HighMemory", "critical")
```

---

### 8. Performance Considerations

#### Parallel Execution

```go
t.Parallel() // Enable for independent tests
```

#### Test Isolation

```go
// Each test resets infrastructure
func TestXXX(t *testing.T) {
	infra, _ := SetupTestInfrastructure(ctx)
	defer infra.Teardown(ctx)

	// Ensure clean state
	require.NoError(t, infra.ResetDatabase(ctx))
	require.NoError(t, infra.ResetRedis(ctx))

	// Run test...
}
```

#### Container Reuse (Optional)

```go
// For faster execution, reuse containers
var sharedInfra *TestInfrastructure

func TestMain(m *testing.M) {
	ctx := context.Background()
	sharedInfra, _ = SetupTestInfrastructure(ctx)
	exitCode := m.Run()
	sharedInfra.Teardown(ctx)
	os.Exit(exitCode)
}

func TestXXX(t *testing.T) {
	// Reset state only
	sharedInfra.ResetDatabase(context.Background())
	sharedInfra.ResetRedis(context.Background())
	// Use sharedInfra...
}
```

---

### 9. Error Handling

#### Expected Errors

```go
// Test error response
resp, err := helper.MakeRequest("POST", "/webhook", malformedJSON)
require.NoError(t, err) // HTTP call succeeds
assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

var errResp map[string]interface{}
json.NewDecoder(resp.Body).Decode(&errResp)
assert.Contains(t, errResp["message"], "invalid JSON")
```

#### Unexpected Errors

```go
// Fail fast on unexpected errors
resp, err := helper.MakeRequest("POST", "/webhook", validPayload)
require.NoError(t, err, "HTTP request should succeed")
defer resp.Body.Close()

// Log response body for debugging
if resp.StatusCode != http.StatusOK {
	body, _ := io.ReadAll(resp.Body)
	t.Logf("Unexpected response: %s", body)
	t.FailNow()
}
```

---

### 10. CI/CD Integration

#### GitHub Actions Workflow

```yaml
name: E2E Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run E2E Tests
        working-directory: ./go-app
        run: |
          go test -v -tags=e2e ./test/e2e/... \
            -timeout=10m \
            -count=1

      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: e2e-test-results
          path: go-app/test-results/
```

---

### 11. Test Organization

#### File Structure

```
go-app/test/e2e/
├── e2e_ingestion_test.go       # Alert ingestion tests (5)
├── e2e_classification_test.go  # Classification pipeline (5)
├── e2e_publishing_test.go      # Publishing flows (5)
├── e2e_history_test.go         # History & query (3)
├── e2e_errors_test.go          # Error handling (2)
├── mock_publishing.go          # Mock targets
├── helpers_publishing.go       # Publishing helpers
└── README.md                   # E2E test documentation
```

#### Test Naming Convention

```
TestE2E_{Feature}_{Scenario}_{Condition}

Examples:
- TestE2E_Ingestion_HappyPath
- TestE2E_Ingestion_Duplicate
- TestE2E_Classification_CacheHitL1
- TestE2E_Publishing_MultiTarget
- TestE2E_Errors_LLMTimeout
```

---

### 12. Metrics & Observability

#### Test Metrics

```go
// Track test duration
start := time.Now()
defer func() {
	t.Logf("Test duration: %v", time.Since(start))
}()

// Track container startup time
infra, err := SetupTestInfrastructure(ctx)
t.Logf("Infrastructure ready in: %v", infra.StartupDuration)
```

#### Application Metrics Validation

```go
// Assert Prometheus metrics
metricsResp, err := helper.MakeRequest("GET", "/metrics", nil)
require.NoError(t, err)

body, _ := io.ReadAll(metricsResp.Body)
metricsText := string(body)

// Check specific metrics
assert.Contains(t, metricsText, "alerts_received_total")
assert.Contains(t, metricsText, "classification_l1_cache_hits_total")
```

---

### 13. Debugging Support

#### Verbose Logging

```go
// Enable verbose logging in tests
t.Setenv("LOG_LEVEL", "debug")

// Log important events
t.Logf("Sending webhook with %d alerts", len(webhook.Alerts))
t.Logf("Response status: %d", resp.StatusCode)
```

#### Request/Response Logging

```go
// Log full request/response for debugging
if testing.Verbose() {
	reqBody, _ := json.MarshalIndent(payload, "", "  ")
	t.Logf("Request: %s", reqBody)

	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("Response: %s", respBody)
	resp.Body = io.NopCloser(bytes.NewReader(respBody)) // Restore
}
```

---

### 14. Cleanup & Teardown

#### Resource Cleanup

```go
func TestXXX(t *testing.T) {
	// Setup
	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)

	// Ensure cleanup even on panic
	defer func() {
		if r := recover(); r != nil {
			infra.Teardown(ctx)
			panic(r) // Re-panic after cleanup
		}
	}()

	defer infra.Teardown(ctx)

	// Test body...
}
```

#### Goroutine Leak Detection

```go
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
```

---

### 15. Success Criteria

#### Quantitative

- **20+ E2E scenarios** implemented
- **100% pass rate** (no flaky tests)
- **Execution time** <5 minutes
- **Coverage** 80%+ critical flows
- **Code** 1,500+ LOC (tests + mocks + helpers)

#### Qualitative

- **Reliability**: Deterministic, no flakiness
- **Maintainability**: Clear structure, easy to extend
- **Documentation**: Clear README with examples
- **CI/CD Ready**: GitHub Actions integration

---

## Implementation Plan

### Phase 1: Requirements & Design ✅
- [x] requirements.md (387 LOC)
- [x] design.md (this document)

### Phase 2: Mock Publishing Targets (1h)
- [ ] mock_publishing.go (300 LOC)
- [ ] helpers_publishing.go (200 LOC)
- [ ] Test mock targets work

### Phase 3: E2E Scenarios (3-4h)
- [ ] e2e_ingestion_test.go (5 tests, 400 LOC)
- [ ] e2e_classification_test.go (5 tests, 450 LOC)
- [ ] e2e_publishing_test.go (5 tests, 500 LOC)
- [ ] e2e_history_test.go (3 tests, 300 LOC)
- [ ] e2e_errors_test.go (2 tests, 200 LOC)

### Phase 4: CI/CD + Docs (1h)
- [ ] .github/workflows/e2e-tests.yml
- [ ] test/e2e/README.md
- [ ] tasks/TN-108-e2e-tests/COMPLETION.md

---

## References

- TN-107: Integration test infrastructure
- Testcontainers: https://golang.testcontainers.org/
- Go testing: https://pkg.go.dev/testing
- GitHub Actions: https://docs.github.com/en/actions
