# TN-106: Unit Tests (>80% coverage) - Technical Design

**Task ID**: TN-106
**Phase**: 14 - Testing & Documentation
**Status**: 🔄 IN PROGRESS (Phase 2)
**Target Quality**: **150%** (Grade A+ EXCEPTIONAL)
**Date**: 2025-11-30

---

## 📐 Architecture Overview

### System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                     Alert History Service                        │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐           │
│  │   Handlers   │  │    Cache     │  │   Query     │           │
│  │   32.5%      │  │   40.8%      │  │   66.7%     │           │
│  │   → 80%+     │  │   → 80%+     │  │   → 80%+    │           │
│  └─────────────┘  └──────────────┘  └─────────────┘           │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐           │
│  │   Metrics    │  │  Middleware  │  │  Security   │           │
│  │   69.7%      │  │   88.4% ✅   │  │   51.1%     │           │
│  │   → 80%+     │  │   DONE       │  │   Future    │           │
│  └─────────────┘  └──────────────┘  └─────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

**Coverage Gap Analysis**:
- **Critical Gaps** (need work): handlers (47.5%), cache (39.2%), filters (53.8%)
- **Medium Gaps** (need work): query (13.3%), metrics (10.3%), security (28.9%)
- **Already Good** (maintain): middleware (88.4%)

**Total Coverage**: 40.3% → **Target**: 80%+ (150%: 85%+)

---

## 🎯 Design Principles

### 1. Test Pyramid Compliance
```
         /\
        /  \  E2E Tests (TN-108)       ← 10%
       /────\
      / Inte \  Integration (TN-107)   ← 20%
     /  grate \
    /──────────\
   /    Unit    \  Unit Tests (TN-106) ← 70%
  /    Tests     \
 /________________\
```

**TN-106 Focus**: Unit tests (70% of test pyramid)
- Fast execution (<30s total)
- No external dependencies (mocked)
- Isolated component testing
- High coverage (80%+)

### 2. Test-Driven Quality (150%)

**Baseline (100%)**:
- Tests compile and run
- Coverage ≥80%
- Pass rate 100%

**Target (150%)**:
- Coverage ≥85%
- Table-driven tests (multiple scenarios per LOC)
- Benchmarks for critical paths
- Concurrent tests for thread-safety
- Comprehensive edge cases
- Error path coverage 100%

### 3. Testing Strategy

#### 3.1 Table-Driven Pattern (Primary)
```go
func TestComponent_Scenario(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {"happy path", validInput, expectedOutput, false},
        {"empty input", emptyInput, defaultOutput, false},
        {"invalid input", invalidInput, nil, true},
        // ... 10+ scenarios per function
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ComponentFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Why**: Maximize coverage per LOC, comprehensive scenario testing

#### 3.2 Mock Pattern (Dependencies)
```go
type mockRepository struct {
    saveFunc func(*Alert) error
    getFunc  func(string) (*Alert, error)
}

func (m *mockRepository) Save(a *Alert) error {
    if m.saveFunc != nil {
        return m.saveFunc(a)
    }
    return nil
}
```

**Why**: Isolate component, control behavior, test error paths

#### 3.3 Benchmark Pattern (150% Bonus)
```go
func BenchmarkHandlerGet(b *testing.B) {
    handler := setupHandler()
    req := httptest.NewRequest("GET", "/api/v2/history", nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
    }
}
```

**Why**: Validate performance targets (p95 < 10ms), detect regressions

---

## 🔍 Component Design

### 1. pkg/history/handlers (Priority 1)

**Current**: 32.5% coverage
**Target**: 80%+ coverage
**Effort**: ~500 LOC tests
**Files**: 8 handlers

#### 1.1 Architecture

```go
// Handler interface
type HistoryHandler interface {
    HandleGetHistory(w http.ResponseWriter, r *http.Request)
    HandleGetRecent(w http.ResponseWriter, r *http.Request)
    HandleGetStats(w http.ResponseWriter, r *http.Request)
    // ... 5 more handlers
}

// Implementation
type handler struct {
    repo      AlertRepository
    cache     Cache
    filters   FilterRegistry
    metrics   MetricsCollector
    validator RequestValidator
}
```

**Dependencies (to mock)**:
1. `AlertRepository` - database operations
2. `Cache` - caching layer
3. `FilterRegistry` - filter creation
4. `MetricsCollector` - metrics recording
5. `RequestValidator` - input validation

#### 1.2 Test Strategy

**Coverage Plan** (500 LOC tests):

| Handler | Scenarios | LOC | Priority |
|---------|-----------|-----|----------|
| `HandleGetHistory` | 15 | 150 | P0 |
| `HandleGetRecent` | 10 | 100 | P0 |
| `HandleGetStats` | 8 | 80 | P1 |
| `HandleGetByFingerprint` | 10 | 100 | P0 |
| `HandleGetTop` | 6 | 60 | P1 |
| `HandleGetFlapping` | 5 | 50 | P1 |
| `HandlePost` | 8 | 80 | P0 |
| `HandleDelete` | 5 | 50 | P1 |

**Scenario Types**:
1. **Happy Path** (20%): Valid request → 200 OK
2. **Validation Errors** (30%): Invalid params → 400 Bad Request
3. **Not Found** (10%): Missing resource → 404 Not Found
4. **Server Errors** (20%): Database errors → 500 Internal Server Error
5. **Edge Cases** (20%): Empty results, pagination boundaries, large datasets

**Example Test Plan - HandleGetHistory**:
```go
func TestHandleGetHistory(t *testing.T) {
    tests := []struct {
        name       string
        method     string
        url        string
        repoResult []*Alert
        repoError  error
        cacheHit   bool
        wantStatus int
        wantBody   string
    }{
        // Happy paths
        {"valid request - cache hit", "GET", "/api/v2/history?status=firing", nil, nil, true, 200, `"total":10`},
        {"valid request - cache miss", "GET", "/api/v2/history?status=firing", alerts, nil, false, 200, `"total":10`},
        {"valid pagination", "GET", "/api/v2/history?limit=50&offset=100", alerts, nil, false, 200, `"total":50`},
        {"valid filters", "GET", "/api/v2/history?severity=critical&namespace=prod", alerts, nil, false, 200, `"total":5`},

        // Validation errors
        {"invalid method", "POST", "/api/v2/history", nil, nil, false, 405, "Method Not Allowed"},
        {"invalid limit", "GET", "/api/v2/history?limit=-1", nil, nil, false, 400, "invalid limit"},
        {"invalid offset", "GET", "/api/v2/history?offset=-1", nil, nil, false, 400, "invalid offset"},
        {"limit too large", "GET", "/api/v2/history?limit=10000", nil, nil, false, 400, "limit exceeds maximum"},

        // Not found
        {"empty results", "GET", "/api/v2/history?status=unknown", []*Alert{}, nil, false, 200, `"total":0`},

        // Server errors
        {"database error", "GET", "/api/v2/history", nil, errors.New("db connection failed"), false, 500, "internal server error"},
        {"cache error (graceful)", "GET", "/api/v2/history", alerts, nil, false, 200, `"total":10`},

        // Edge cases
        {"zero limit", "GET", "/api/v2/history?limit=0", nil, nil, false, 400, "invalid limit"},
        {"large offset", "GET", "/api/v2/history?offset=1000000", []*Alert{}, nil, false, 200, `"total":0`},
        {"special characters in params", "GET", "/api/v2/history?namespace=%27DROP%20TABLE%27", alerts, nil, false, 200, `"total":0`},
        {"concurrent requests (race test)", "GET", "/api/v2/history", alerts, nil, false, 200, `"total":10`},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            repo := &mockRepository{
                listFunc: func(...) ([]*Alert, int, error) {
                    return tt.repoResult, len(tt.repoResult), tt.repoError
                },
            }
            cache := &mockCache{hitFunc: func() bool { return tt.cacheHit }}
            handler := NewHandler(repo, cache, nil, nil, nil)

            // Execute request
            req := httptest.NewRequest(tt.method, tt.url, nil)
            rr := httptest.NewRecorder()
            handler.HandleGetHistory(rr, req)

            // Assertions
            if rr.Code != tt.wantStatus {
                t.Errorf("status = %v, want %v", rr.Code, tt.wantStatus)
            }
            if !strings.Contains(rr.Body.String(), tt.wantBody) {
                t.Errorf("body = %v, want substring %v", rr.Body.String(), tt.wantBody)
            }
        })
    }
}
```

**150% Enhancements**:
```go
// Benchmark test
func BenchmarkHandleGetHistory(b *testing.B) {
    handler := setupHandler()
    req := httptest.NewRequest("GET", "/api/v2/history?status=firing", nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rr := httptest.NewRecorder()
        handler.HandleGetHistory(rr, req)
    }
}

// Concurrent test (race detection)
func TestHandleGetHistory_Concurrent(t *testing.T) {
    handler := setupHandler()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            req := httptest.NewRequest("GET", "/api/v2/history", nil)
            rr := httptest.NewRecorder()
            handler.HandleGetHistory(rr, req)
        }()
    }
    wg.Wait()
}
```

---

### 2. pkg/history/cache (Priority 2)

**Current**: 40.8% coverage
**Target**: 80%+ coverage
**Effort**: ~400 LOC tests

#### 2.1 Architecture

```go
type CacheManager interface {
    Get(key string) (interface{}, bool, error)
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
    Invalidate(pattern string) error
}

type cacheManager struct {
    l1    *ristretto.Cache  // In-memory L1
    l2    *redis.Client     // Redis L2
    metrics MetricsCollector
}
```

**Critical Functions to Test**:
1. `Get()` - Cache hit/miss scenarios
2. `Set()` - TTL handling, eviction
3. `Delete()` - Single key removal
4. `Invalidate()` - Pattern-based removal
5. `getL1()` - L1 cache logic
6. `getL2()` - L2 cache logic
7. `setL1()` - L1 write-through
8. `setL2()` - L2 write-through

#### 2.2 Test Strategy

**Coverage Plan** (400 LOC tests):

| Function | Scenarios | LOC | Priority |
|----------|-----------|-----|----------|
| `Get()` | 15 | 120 | P0 |
| `Set()` | 12 | 100 | P0 |
| `Delete()` | 6 | 50 | P1 |
| `Invalidate()` | 8 | 80 | P0 |
| `Internal helpers` | 10 | 80 | P1 |

**Scenario Types**:
1. **L1 Hit** (10%): Data in memory
2. **L2 Hit** (10%): Data in Redis
3. **Miss** (10%): Data not found
4. **L1 + L2 Miss** (10%): Full miss
5. **TTL Expiration** (15%): Time-based eviction
6. **Eviction** (15%): LRU eviction
7. **Redis Failures** (20%): L2 down → graceful degradation
8. **Concurrent Access** (10%): Thread-safety

**Example Test Plan - Get**:
```go
func TestCacheManager_Get(t *testing.T) {
    tests := []struct {
        name      string
        key       string
        l1Value   interface{}
        l1Exists  bool
        l2Value   interface{}
        l2Exists  bool
        l2Error   error
        wantValue interface{}
        wantHit   bool
        wantErr   bool
    }{
        // L1 hit
        {"l1 hit", "key1", "value1", true, nil, false, nil, "value1", true, false},

        // L2 hit (L1 miss)
        {"l2 hit", "key2", nil, false, "value2", true, nil, "value2", true, false},

        // Full miss
        {"miss", "key3", nil, false, nil, false, nil, nil, false, false},

        // L2 error (graceful degradation)
        {"l2 error", "key4", nil, false, nil, false, redis.ErrClosed, nil, false, false},

        // Nil value in L1
        {"nil value", "key5", nil, true, nil, false, nil, nil, true, false},

        // Large value
        {"large value", "key6", make([]byte, 1024*1024), true, nil, false, nil, make([]byte, 1024*1024), true, false},

        // Empty key
        {"empty key", "", nil, false, nil, false, nil, nil, false, true},

        // Special characters
        {"special chars", "key:with:colons", "value", true, nil, false, nil, "value", true, false},

        // Unicode key
        {"unicode key", "ключ", "значение", true, nil, false, nil, "значение", true, false},

        // Expired TTL (simulated)
        {"expired ttl", "key7", nil, false, nil, false, nil, nil, false, false},

        // Concurrent access (race test)
        {"concurrent", "key8", "value8", true, nil, false, nil, "value8", true, false},

        // L1 + L2 both have value (L1 wins)
        {"l1 l2 conflict", "key9", "l1value", true, "l2value", true, nil, "l1value", true, false},

        // L1 set after L2 hit
        {"l2 hit populates l1", "key10", nil, false, "value10", true, nil, "value10", true, false},

        // L2 timeout
        {"l2 timeout", "key11", nil, false, nil, false, context.DeadlineExceeded, nil, false, false},

        // L2 connection pool exhausted
        {"l2 pool exhausted", "key12", nil, false, nil, false, redis.PoolExhausted, nil, false, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            l1 := newMockL1Cache()
            l2 := newMockL2Cache()

            if tt.l1Exists {
                l1.data[tt.key] = tt.l1Value
            }
            if tt.l2Exists {
                l2.data[tt.key] = tt.l2Value
            }
            if tt.l2Error != nil {
                l2.getErr = tt.l2Error
            }

            cm := &cacheManager{l1: l1, l2: l2}

            // Execute
            got, hit, err := cm.Get(tt.key)

            // Assertions
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if hit != tt.wantHit {
                t.Errorf("hit = %v, wantHit %v", hit, tt.wantHit)
            }
            if !reflect.DeepEqual(got, tt.wantValue) {
                t.Errorf("got = %v, want %v", got, tt.wantValue)
            }
        })
    }
}
```

**150% Enhancements**:
```go
// Cache stampede test
func TestCacheManager_CacheStampede(t *testing.T) {
    cm := setupCacheManager()
    key := "hot-key"

    // Simulate 100 concurrent requests for same key
    var wg sync.WaitGroup
    results := make([]interface{}, 100)

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            val, _, _ := cm.Get(key)
            results[idx] = val
        }(i)
    }
    wg.Wait()

    // Verify only ONE database query was made (cache stampede prevention)
    // All goroutines should get same value
}

// Memory leak test
func TestCacheManager_MemoryLeak(t *testing.T) {
    cm := setupCacheManager()

    var m1, m2 runtime.MemStats
    runtime.ReadMemStats(&m1)

    // Insert 10K items
    for i := 0; i < 10000; i++ {
        cm.Set(fmt.Sprintf("key%d", i), "value", 1*time.Minute)
    }

    // Force GC
    runtime.GC()
    runtime.ReadMemStats(&m2)

    // Verify memory usage is reasonable
    if m2.Alloc-m1.Alloc > 50*1024*1024 { // 50MB threshold
        t.Errorf("Memory leak detected: %v bytes", m2.Alloc-m1.Alloc)
    }
}
```

---

### 3. pkg/history/query (Priority 3)

**Current**: 66.7% coverage
**Target**: 80%+ coverage
**Effort**: ~150 LOC tests

#### 3.1 Architecture

```go
type QueryBuilder interface {
    WithFilters(filters []Filter) QueryBuilder
    WithSort(field string, order string) QueryBuilder
    WithPagination(limit, offset int) QueryBuilder
    Build() (query string, args []interface{}, err error)
}

type queryBuilder struct {
    base    string
    filters []Filter
    sorts   []SortOption
    limit   int
    offset  int
}
```

**Critical Functions to Test**:
1. `Build()` - SQL generation
2. `WithFilters()` - Filter application
3. `WithSort()` - ORDER BY generation
4. `WithPagination()` - LIMIT/OFFSET
5. `applyFilters()` - Internal filter logic
6. `applySorts()` - Internal sort logic

#### 3.2 Test Strategy

**Coverage Plan** (150 LOC tests):

| Function | Scenarios | LOC | Priority |
|----------|-----------|-----|----------|
| `Build()` | 10 | 60 | P0 |
| `WithFilters()` | 8 | 40 | P0 |
| `WithSort()` | 5 | 25 | P1 |
| `WithPagination()` | 5 | 25 | P1 |

**Scenario Types**:
1. **Single Filter** (15%): One filter condition
2. **Multiple Filters** (25%): AND combinations
3. **Sort Variations** (15%): Different fields + orders
4. **Pagination** (15%): Various limit/offset
5. **Complex Queries** (20%): Filters + Sort + Pagination
6. **SQL Injection Prevention** (10%): Malicious inputs

**Example Test Plan - Build**:
```go
func TestQueryBuilder_Build(t *testing.T) {
    tests := []struct {
        name       string
        filters    []Filter
        sorts      []SortOption
        limit      int
        offset     int
        wantQuery  string
        wantArgs   []interface{}
        wantErr    bool
    }{
        {
            name:      "empty query",
            wantQuery: "SELECT * FROM alerts",
            wantArgs:  []interface{}{},
            wantErr:   false,
        },
        {
            name:      "single filter",
            filters:   []Filter{{Field: "status", Op: "=", Value: "firing"}},
            wantQuery: "SELECT * FROM alerts WHERE status = $1",
            wantArgs:  []interface{}{"firing"},
            wantErr:   false,
        },
        {
            name: "multiple filters",
            filters: []Filter{
                {Field: "status", Op: "=", Value: "firing"},
                {Field: "severity", Op: "=", Value: "critical"},
            },
            wantQuery: "SELECT * FROM alerts WHERE status = $1 AND severity = $2",
            wantArgs:  []interface{}{"firing", "critical"},
            wantErr:   false,
        },
        {
            name:      "pagination",
            limit:     50,
            offset:    100,
            wantQuery: "SELECT * FROM alerts LIMIT $1 OFFSET $2",
            wantArgs:  []interface{}{50, 100},
            wantErr:   false,
        },
        {
            name:      "sort ascending",
            sorts:     []SortOption{{Field: "created_at", Order: "ASC"}},
            wantQuery: "SELECT * FROM alerts ORDER BY created_at ASC",
            wantArgs:  []interface{}{},
            wantErr:   false,
        },
        {
            name: "complex query",
            filters: []Filter{
                {Field: "status", Op: "=", Value: "firing"},
            },
            sorts:     []SortOption{{Field: "severity", Order: "DESC"}},
            limit:     20,
            offset:    0,
            wantQuery: "SELECT * FROM alerts WHERE status = $1 ORDER BY severity DESC LIMIT $2 OFFSET $3",
            wantArgs:  []interface{}{"firing", 20, 0},
            wantErr:   false,
        },
        {
            name:      "sql injection attempt",
            filters:   []Filter{{Field: "status", Op: "=", Value: "'; DROP TABLE alerts; --"}},
            wantQuery: "SELECT * FROM alerts WHERE status = $1",
            wantArgs:  []interface{}{"'; DROP TABLE alerts; --"},
            wantErr:   false,
        },
        {
            name:      "invalid limit",
            limit:     -1,
            wantQuery: "",
            wantArgs:  nil,
            wantErr:   true,
        },
        {
            name:      "invalid offset",
            offset:    -1,
            wantQuery: "",
            wantArgs:  nil,
            wantErr:   true,
        },
        {
            name:      "zero limit (valid)",
            limit:     0,
            wantQuery: "SELECT * FROM alerts",
            wantArgs:  []interface{}{},
            wantErr:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            qb := NewQueryBuilder("SELECT * FROM alerts")
            if len(tt.filters) > 0 {
                qb = qb.WithFilters(tt.filters)
            }
            if len(tt.sorts) > 0 {
                for _, sort := range tt.sorts {
                    qb = qb.WithSort(sort.Field, sort.Order)
                }
            }
            if tt.limit > 0 || tt.offset > 0 {
                qb = qb.WithPagination(tt.limit, tt.offset)
            }

            query, args, err := qb.Build()

            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if query != tt.wantQuery {
                t.Errorf("query = %v, want %v", query, tt.wantQuery)
            }
            if !reflect.DeepEqual(args, tt.wantArgs) {
                t.Errorf("args = %v, want %v", args, tt.wantArgs)
            }
        })
    }
}
```

---

### 4. pkg/metrics (Priority 4)

**Current**: 69.7% coverage
**Target**: 80%+ coverage
**Effort**: ~100 LOC tests

#### 4.1 Current Gap Analysis

From coverage output:
- `Lock()`: 0.0% ❌
- `Unlock()`: 0.0% ❌
- `RecordAttempt()`: 0.0% ❌
- `RecordBackoff()`: 0.0% ❌
- `RecordFinalAttempt()`: 0.0% ❌
- `Reset()`: 0.0% ❌
- `SetActiveWorkers()`: 0.0% ❌

**Root Cause**: Retry metrics methods not tested in webhook metrics tests

#### 4.2 Test Strategy

**Coverage Plan** (100 LOC tests):

| Component | Scenarios | LOC | Priority |
|-----------|-----------|-----|----------|
| Retry metrics | 10 | 50 | P0 |
| Webhook metrics edge cases | 5 | 30 | P1 |
| Technical metrics | 3 | 20 | P1 |

**Example Test Plan - Retry Metrics**:
```go
func TestRetryMetrics_RecordAttempt(t *testing.T) {
    registry := prometheus.NewRegistry()
    rm := NewRetryMetrics(registry)

    tests := []struct {
        name      string
        operation string
        attempt   int
        wantCount float64
    }{
        {"first attempt", "classify", 1, 1.0},
        {"second attempt", "classify", 2, 1.0},
        {"max attempt", "classify", 5, 1.0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rm.RecordAttempt(tt.operation, tt.attempt)

            // Verify metric was recorded
            mfs, _ := registry.Gather()
            found := false
            for _, mf := range mfs {
                if mf.GetName() == "retry_attempts_total" {
                    for _, m := range mf.GetMetric() {
                        if getLabel(m, "operation") == tt.operation {
                            if m.GetCounter().GetValue() == tt.wantCount {
                                found = true
                            }
                        }
                    }
                }
            }
            if !found {
                t.Errorf("Metric not found or incorrect value")
            }
        })
    }
}
```

---

## 🛠️ Implementation Strategy

### Phase 2.1: pkg/history/handlers (4 hours)

**Day 1 - Morning (2h)**:
1. Create `handlers_test.go` with test infrastructure (30min)
   - Mock repository
   - Mock cache
   - Mock filters
   - Helper functions
2. Implement `TestHandleGetHistory` (15 scenarios) (1h)
3. Implement `TestHandleGetRecent` (10 scenarios) (30min)

**Day 1 - Afternoon (2h)**:
4. Implement `TestHandleGetStats` (8 scenarios) (45min)
5. Implement `TestHandleGetByFingerprint` (10 scenarios) (45min)
6. Implement remaining handlers (5 handlers × 30min) (30min)

**Validation**:
```bash
go test -cover ./pkg/history/handlers
# Target: 80%+ coverage
```

### Phase 2.2: pkg/history/cache (3 hours)

**Day 2 - Morning (3h)**:
1. Create `cache_test.go` with test infrastructure (30min)
   - Mock L1 cache
   - Mock L2 cache (Redis)
   - Helper functions
2. Implement `TestCacheManager_Get` (15 scenarios) (1h)
3. Implement `TestCacheManager_Set` (12 scenarios) (45min)
4. Implement `TestCacheManager_Invalidate` (8 scenarios) (45min)

**Validation**:
```bash
go test -cover ./pkg/history/cache
# Target: 80%+ coverage
```

### Phase 2.3: pkg/history/query (1.5 hours)

**Day 2 - Afternoon (1.5h)**:
1. Enhance existing `query_test.go` (30min)
2. Add `TestQueryBuilder_Build` (10 scenarios) (45min)
3. Add edge case tests (15min)

**Validation**:
```bash
go test -cover ./pkg/history/query
# Target: 80%+ coverage
```

### Phase 2.4: pkg/metrics (1.5 hours)

**Day 2 - Evening (1.5h)**:
1. Create `retry_metrics_test.go` (45min)
2. Implement retry metrics tests (10 scenarios) (30min)
3. Add edge case tests for webhook metrics (15min)

**Validation**:
```bash
go test -cover ./pkg/metrics
# Target: 80%+ coverage
```

### Phase 2.5: Validation & Cleanup (1 hour)

**Day 3 - Morning (1h)**:
1. Run full test suite (15min)
   ```bash
   go test -cover ./pkg/history/... ./pkg/metrics/...
   ```
2. Verify 80%+ coverage achieved (15min)
3. Fix any remaining gaps (15min)
4. Final validation (15min)
   ```bash
   go test -race ./pkg/history/... ./pkg/metrics/...
   ```

---

## 🎯 150% Quality Enhancements

### 1. Benchmark Tests (2 hours)

**Critical Paths to Benchmark**:
1. `HandleGetHistory` - HTTP handler latency
2. `CacheManager.Get` - Cache lookup performance
3. `QueryBuilder.Build` - SQL generation speed

**Example Benchmarks**:
```go
// pkg/history/handlers/handlers_bench_test.go
func BenchmarkHandleGetHistory(b *testing.B) {
    handler := setupHandler()
    req := httptest.NewRequest("GET", "/api/v2/history", nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rr := httptest.NewRecorder()
        handler.HandleGetHistory(rr, req)
    }
}

// pkg/history/cache/cache_bench_test.go
func BenchmarkCacheGet_L1Hit(b *testing.B) {
    cm := setupCacheManager()
    cm.Set("key", "value", 1*time.Minute)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cm.Get("key")
    }
}
```

### 2. Concurrent Tests (2 hours)

**Thread-Safety Validation**:
1. Handlers - Concurrent HTTP requests
2. Cache - Concurrent reads/writes
3. Metrics - Concurrent metric updates

**Example Concurrent Tests**:
```go
func TestHandleGetHistory_Concurrent(t *testing.T) {
    handler := setupHandler()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            req := httptest.NewRequest("GET", "/api/v2/history", nil)
            rr := httptest.NewRecorder()
            handler.HandleGetHistory(rr, req)
            if rr.Code != 200 {
                t.Errorf("Concurrent request failed: %v", rr.Code)
            }
        }()
    }
    wg.Wait()
}
```

### 3. Edge Case Tests (2 hours)

**Edge Cases to Cover**:
1. Empty inputs
2. Nil pointers
3. Very large datasets
4. Unicode characters
5. SQL injection attempts
6. Buffer overflows
7. Negative numbers
8. Zero values

---

## 📊 Success Metrics

### Coverage Targets

| Metric | Baseline (100%) | Target (150%) | Current |
|--------|----------------|---------------|---------|
| Overall coverage | 80% | 85% | 40.3% |
| pkg/history/handlers | 80% | 85% | 32.5% |
| pkg/history/cache | 80% | 85% | 40.8% |
| pkg/history/query | 80% | 85% | 66.7% |
| pkg/metrics | 80% | 85% | 69.7% |

### Quality Targets

| Metric | Baseline (100%) | Target (150%) | Current |
|--------|----------------|---------------|---------|
| Test pass rate | 100% | 100% | 100% ✅ |
| Flaky tests | 0 | 0 | 0 ✅ |
| Test execution time | <30s | <20s | ~5s ✅ |
| Benchmarks | 0 | 5+ | 0 |
| Concurrent tests | 0 | 3+ | 0 |

---

## 🔒 Quality Gates

### Gate 1: Phase 2.1 Complete (Handlers)
- [ ] pkg/history/handlers coverage ≥80%
- [ ] All handler tests passing
- [ ] Zero flaky tests

### Gate 2: Phase 2.2 Complete (Cache)
- [ ] pkg/history/cache coverage ≥80%
- [ ] All cache tests passing
- [ ] Concurrent tests passing

### Gate 3: Phase 2.3-2.4 Complete (Query + Metrics)
- [ ] pkg/history/query coverage ≥80%
- [ ] pkg/metrics coverage ≥80%
- [ ] All tests passing

### Gate 4: 150% Enhancements
- [ ] 5+ benchmark tests implemented
- [ ] 3+ concurrent tests implemented
- [ ] Race detector passes
- [ ] Overall coverage ≥85%

### Gate 5: Final Certification
- [ ] All quality gates passed
- [ ] Coverage report generated
- [ ] Certification document created
- [ ] 150% quality achieved

---

**Document Version**: 1.0
**Last Review**: 2025-11-30
**Next Review**: After Phase 2 complete
**Owner**: Vitalii Semenov
