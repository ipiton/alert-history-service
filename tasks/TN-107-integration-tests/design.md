# TN-107: Integration Tests - Technical Design

**Task ID**: TN-107
**Date**: 2025-11-30
**Author**: AI Assistant
**Status**: IN PROGRESS

---

## 1. Architecture Overview

### 1.1 Test Infrastructure

```
┌─────────────────────────────────────────────────────────────┐
│                    Integration Test Suite                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │  API Tests    │  │ Database Tests│  │  Cache Tests  │   │
│  │               │  │               │  │               │   │
│  │ • HTTP calls  │  │ • CRUD ops    │  │ • L1/L2 cache │   │
│  │ • Validation  │  │ • Transactions│  │ • Failover    │   │
│  │ • Errors      │  │ • Migrations  │  │ • TTL         │   │
│  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘   │
│          │                  │                  │             │
│          └──────────────────┼──────────────────┘             │
│                             │                                │
│                    ┌────────▼────────┐                       │
│                    │  Test Fixtures  │                       │
│                    │  & Helpers      │                       │
│                    └─────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                  Test Infrastructure                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────────┐  │
│  │PostgreSQL│  │  Redis   │  │ Mock LLM  │  │ Mock K8s  │  │
│  │Container │  │Container │  │ Server    │  │ API       │  │
│  └──────────┘  └──────────┘  └───────────┘  └───────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Test Execution Flow

```
Start Test
    │
    ├──> Setup (once per package)
    │    ├── Start Docker containers (PostgreSQL, Redis)
    │    ├── Run database migrations
    │    ├── Initialize test data
    │    └── Start mock servers (LLM, K8s)
    │
    ├──> Test Execution (parallel)
    │    ├── Test 1: API endpoint
    │    ├── Test 2: Database operations
    │    ├── Test 3: Cache behavior
    │    └── ... more tests
    │
    └──> Teardown (once per package)
         ├── Stop mock servers
         ├── Clean database
         └── Stop Docker containers
```

---

## 2. Component Design

### 2.1 Test Infrastructure (`test/integration/infra.go`)

```go
package integration

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

// TestInfrastructure manages test infrastructure lifecycle
type TestInfrastructure struct {
    PostgresContainer testcontainers.Container
    RedisContainer    testcontainers.Container
    DB                *sql.DB
    RedisClient       *redis.Client
    MockLLMServer     *MockLLMServer
    MockK8sServer     *MockK8sServer
}

// SetupTestInfrastructure starts all required infrastructure
func SetupTestInfrastructure(ctx context.Context) (*TestInfrastructure, error)

// Teardown stops all infrastructure
func (ti *TestInfrastructure) Teardown(ctx context.Context) error

// ResetDatabase truncates all tables for clean test state
func (ti *TestInfrastructure) ResetDatabase(ctx context.Context) error
```

**Key Features**:
- Testcontainers-go for PostgreSQL + Redis (no Docker Compose needed)
- Automatic port allocation (no conflicts)
- Health checks before tests run
- Graceful cleanup on test failure

### 2.2 API Test Helper (`test/integration/api_helper.go`)

```go
// APITestHelper provides utilities for API testing
type APITestHelper struct {
    BaseURL    string
    HTTPClient *http.Client
    DB         *sql.DB
    Redis      *redis.Client
}

// NewAPITestHelper creates test helper with infrastructure
func NewAPITestHelper(infra *TestInfrastructure) *APITestHelper

// MakeRequest performs HTTP request and returns response
func (h *APITestHelper) MakeRequest(method, path string, body interface{}) (*http.Response, error)

// AssertResponse validates HTTP response
func (h *APITestHelper) AssertResponse(t *testing.T, resp *http.Response, expectedStatus int)

// GetAlertFromDB retrieves alert from database by fingerprint
func (h *APITestHelper) GetAlertFromDB(ctx context.Context, fingerprint string) (*core.Alert, error)

// SeedTestData inserts test alerts into database
func (h *APITestHelper) SeedTestData(ctx context.Context, alerts []*core.Alert) error
```

### 2.3 Mock LLM Server (`test/integration/mock_llm.go`)

```go
// MockLLMServer simulates LLM API for testing
type MockLLMServer struct {
    server   *httptest.Server
    requests []*LLMRequest
    mu       sync.Mutex
}

// NewMockLLMServer creates mock LLM server
func NewMockLLMServer() *MockLLMServer

// SetResponse configures mock response for specific alert
func (m *MockLLMServer) SetResponse(alertName string, response *ClassificationResult)

// SetError simulates LLM API error
func (m *MockLLMServer) SetError(statusCode int, errorMsg string)

// SetLatency simulates slow LLM API
func (m *MockLLMServer) SetLatency(duration time.Duration)

// GetRequestCount returns number of LLM requests made
func (m *MockLLMServer) GetRequestCount() int
```

**Use Cases**:
- Test cache hit behavior (0 LLM calls when cached)
- Test LLM timeout handling
- Test classification quality
- Test fallback to rule-based

---

## 3. Test Scenarios

### 3.1 API Endpoint Tests (`api_test.go`)

#### Test: Webhook Ingestion End-to-End
```go
func TestWebhookIngestion_EndToEnd(t *testing.T) {
    // Setup
    infra, _ := SetupTestInfrastructure(ctx)
    defer infra.Teardown(ctx)

    // Send webhook
    payload := fixtures.AlertmanagerWebhook()
    resp, _ := helper.MakeRequest("POST", "/webhook", payload)

    // Assert
    assert.Equal(t, 200, resp.StatusCode)

    // Verify database
    alert, _ := helper.GetAlertFromDB(ctx, expectedFingerprint)
    assert.NotNil(t, alert)
    assert.Equal(t, "firing", alert.Status)
}
```

#### Test: Classification with Cache Hit
```go
func TestClassification_CacheHit(t *testing.T) {
    // Setup with pre-seeded cache
    helper.SeedCache(ctx, fingerprint, classification)

    // Make request
    resp, _ := helper.MakeRequest("POST", "/classification/classify", alert)

    // Assert cache hit (0 LLM calls)
    assert.Equal(t, 0, mockLLM.GetRequestCount())
    assert.Equal(t, 200, resp.StatusCode)
}
```

#### Test: Publishing to Multiple Targets
```go
func TestPublishing_MultipleTargets(t *testing.T) {
    // Setup targets
    helper.SeedTargets(ctx, []*Target{
        {Name: "slack-eng", Type: "slack", Health: "healthy"},
        {Name: "pagerduty", Type: "pagerduty", Health: "healthy"},
    })

    // Publish alert
    resp, _ := helper.MakeRequest("POST", "/webhook/proxy", alertPayload)

    // Assert
    assert.Equal(t, 200, resp.StatusCode)

    // Verify published to both targets
    assert.Equal(t, 2, helper.GetPublishedCount(ctx))
}
```

### 3.2 Database Integration Tests (`database_test.go`)

#### Test: Migration Up/Down
```go
func TestMigrations_UpDown(t *testing.T) {
    // Run all migrations up
    err := goose.Up(db, "migrations")
    assert.NoError(t, err)

    // Verify tables exist
    tables := helper.GetTables(ctx)
    assert.Contains(t, tables, "alerts")
    assert.Contains(t, tables, "silences")

    // Rollback migrations
    err = goose.Down(db, "migrations")
    assert.NoError(t, err)
}
```

#### Test: Transaction Rollback
```go
func TestTransaction_Rollback(t *testing.T) {
    // Begin transaction
    tx, _ := db.Begin()

    // Insert alert
    _ = repo.CreateAlert(ctx, tx, alert)

    // Rollback
    tx.Rollback()

    // Assert not persisted
    _, err := repo.GetAlert(ctx, alert.Fingerprint)
    assert.Error(t, err) // should not exist
}
```

### 3.3 Cache Integration Tests (`cache_test.go`)

#### Test: L1 → L2 → Database Fallback
```go
func TestCache_FallbackChain(t *testing.T) {
    // L1 miss → check L2
    result, source := cache.Get(ctx, key)
    assert.Equal(t, "L2", source)

    // L2 miss → check database
    redis.FlushDB(ctx) // clear L2
    result, source = cache.Get(ctx, key)
    assert.Equal(t, "database", source)

    // L1 hit on subsequent call
    result, source = cache.Get(ctx, key)
    assert.Equal(t, "L1", source)
}
```

#### Test: Redis Unavailable → Graceful Degradation
```go
func TestCache_RedisUnavailable(t *testing.T) {
    // Stop Redis
    infra.RedisContainer.Stop(ctx)

    // Cache should fall back to L1 only
    result, source := cache.Get(ctx, key)
    assert.Equal(t, "database", source) // no panic

    // L1 caching still works
    result2, source2 := cache.Get(ctx, key)
    assert.Equal(t, "L1", source2)
}
```

---

## 4. Test Data Fixtures

### 4.1 Alert Fixtures (`test/fixtures/alerts.json`)

```json
{
  "alertmanager_webhook": {
    "alerts": [
      {
        "labels": {
          "alertname": "HighMemoryUsage",
          "severity": "critical",
          "namespace": "production"
        },
        "annotations": {
          "summary": "Memory usage at 95%"
        },
        "startsAt": "2025-11-30T12:00:00Z",
        "status": "firing"
      }
    ]
  },
  "prometheus_v2": {
    "data": {
      "alerts": [...]
    }
  }
}
```

### 4.2 Configuration Fixtures (`test/fixtures/config.yaml`)

```yaml
test_minimal:
  server:
    port: 8080
  database:
    host: localhost
    port: 5432
    dbname: alerthistory_test

test_full:
  # Complete configuration for comprehensive tests
  ...
```

---

## 5. CI/CD Integration

### 5.1 GitHub Actions Workflow

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run integration tests
        run: |
          cd go-app
          go test -tags=integration -v ./test/integration/...
        env:
          TEST_POSTGRES_URL: postgres://postgres:test@localhost:5432/postgres
          TEST_REDIS_ADDR: localhost:6379
```

### 5.2 Local Development

```bash
# Start test infrastructure
make test-infra-up

# Run integration tests
make test-integration

# Run specific test
go test -tags=integration ./test/integration -run TestWebhookIngestion

# Stop infrastructure
make test-infra-down
```

---

## 6. Performance Targets

| Test Category | Target Time | Max Time |
|---------------|-------------|----------|
| API endpoint test | <1s | <3s |
| Database test | <2s | <5s |
| Cache test | <500ms | <2s |
| Publishing flow test | <3s | <10s |
| Error handling test | <1s | <3s |
| **Total suite** | **<3min** | **<5min** |

### Optimization Strategies:
- Parallel test execution (`t.Parallel()`)
- Shared test infrastructure (setup once per package)
- Connection pooling
- Minimal test data (only what's needed)
- In-memory fixtures (avoid disk I/O)

---

## 7. Error Scenarios

### 7.1 Database Failure Scenarios

```go
// Test: Database connection lost during request
func TestDatabase_ConnectionLost(t *testing.T) {
    // Stop PostgreSQL mid-request
    go func() {
        time.Sleep(100 * time.Millisecond)
        infra.PostgresContainer.Stop(ctx)
    }()

    // Make request
    resp, _ := helper.MakeRequest("GET", "/history", nil)

    // Should return 503 Service Unavailable
    assert.Equal(t, 503, resp.StatusCode)
}
```

### 7.2 Redis Failure Scenarios

```go
// Test: Redis connection lost → fallback to L1
func TestRedis_Fallback(t *testing.T) {
    // Kill Redis
    infra.RedisClient.Close()

    // Cache operations should still work (L1 only)
    result, _ := cache.Get(ctx, key)
    assert.NotNil(t, result)
}
```

### 7.3 LLM Timeout Scenarios

```go
// Test: LLM timeout → rule-based fallback
func TestLLM_Timeout(t *testing.T) {
    // Configure mock LLM with high latency
    mockLLM.SetLatency(10 * time.Second)

    // Make classification request (5s timeout)
    resp, _ := helper.MakeRequest("POST", "/classification/classify", alert)

    // Should fallback to rule-based (no error)
    assert.Equal(t, 200, resp.StatusCode)

    var result ClassificationResult
    json.NewDecoder(resp.Body).Decode(&result)
    assert.Equal(t, "rule-based", result.Model)
}
```

---

## 8. Test Organization

### 8.1 File Structure

```
go-app/test/integration/
├── api_test.go              (600 LOC) - API endpoint tests
│   ├── TestWebhookIngestion
│   ├── TestPrometheusAlerts
│   ├── TestClassification
│   ├── TestEnrichmentMode
│   ├── TestHistory
│   └── TestConfiguration
│
├── database_test.go         (400 LOC) - Database tests
│   ├── TestMigrations
│   ├── TestCRUD
│   ├── TestTransactions
│   ├── TestQueryPerformance
│   └── TestConnectionPool
│
├── cache_test.go            (300 LOC) - Cache tests
│   ├── TestCacheHitMiss
│   ├── TestCacheFallback
│   ├── TestCacheInvalidation
│   └── TestCacheTTL
│
├── classification_test.go   (350 LOC) - LLM tests
│   ├── TestClassificationFlow
│   ├── TestCacheBypass
│   ├── TestFallback
│   └── TestTimeout
│
├── publishing_test.go       (450 LOC) - Publishing tests
│   ├── TestTargetDiscovery
│   ├── TestPublishingQueue
│   ├── TestDLQ
│   └── TestParallelPublishing
│
├── silence_test.go          (350 LOC) - Silence tests
│   ├── TestSilenceCRUD
│   ├── TestSilenceMatching
│   └── TestSilenceExpiration
│
├── inhibition_test.go       (250 LOC) - Inhibition tests
│   ├── TestInhibitionRules
│   ├── TestInhibitionMatching
│   └── TestInhibitionState
│
├── errors_test.go           (200 LOC) - Error handling
│   ├── TestDatabaseErrors
│   ├── TestRedisErrors
│   └── TestGracefulDegradation
│
├── infra.go                 (400 LOC) - Infrastructure
├── helpers.go               (300 LOC) - Test utilities
├── fixtures.go              (250 LOC) - Test data
└── mock_llm.go              (200 LOC) - Mock LLM server

TOTAL: ~3,850 LOC (comprehensive test suite)
```

---

## 9. Implementation Phases

### Phase 1: Infrastructure Setup (2h)
- Setup testcontainers-go
- PostgreSQL + Redis containers
- Mock LLM server
- Mock K8s API server
- Test helpers & fixtures

### Phase 2: API Tests (2h)
- Webhook ingestion tests
- Classification endpoint tests
- History endpoint tests
- Enrichment endpoint tests
- Configuration endpoint tests

### Phase 3: Database Tests (1.5h)
- Migration tests
- CRUD operation tests
- Transaction tests
- Query performance tests

### Phase 4: Cache Tests (1h)
- Hit/miss scenarios
- Fallback chain tests
- Redis failure tests
- TTL expiration tests

### Phase 5: Publishing Tests (1.5h)
- Target discovery tests
- Queue processing tests
- DLQ tests
- Parallel publishing tests

### Phase 6: Error Handling Tests (1h)
- Component failure tests
- Graceful degradation tests
- Error propagation tests

**Total: 9 hours** (includes buffer for issues)

---

## 10. Quality Metrics (150% Target)

### Baseline (100%):
- 30+ integration tests
- 70%+ coverage of critical paths
- 100% test pass rate
- <5min test suite execution

### Enhanced (150%):
- **60+ integration tests** (200% of baseline)
- **85%+ coverage of critical paths** (121% of baseline)
- **100% pass rate + zero flaky tests**
- **<3min test suite** (167% better)
- **Comprehensive error scenarios** (50+ error tests)
- **Performance validation** (all targets met)
- **CI/CD integration** (GitHub Actions)
- **Documentation** (README + examples)

---

## 11. Success Criteria

### Must Have:
1. ✅ All critical API endpoints tested
2. ✅ Database operations validated
3. ✅ Cache behavior verified
4. ✅ Error handling works correctly
5. ✅ 100% test pass rate
6. ✅ CI/CD integration working

### Should Have (150%):
7. ✅ Publishing flow validated
8. ✅ Silence/Inhibition tested
9. ✅ Performance benchmarks
10. ✅ Comprehensive documentation

---

**Author**: AI Assistant
**Date**: 2025-11-30
**Version**: 1.0
