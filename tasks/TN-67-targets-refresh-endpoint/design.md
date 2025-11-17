# TN-67: Design Document - POST /publishing/targets/refresh

## 🏗️ Архитектурное решение

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client (Admin User)                        │
└────────────────────────────┬────────────────────────────────────┘
                             │ POST /api/v2/publishing/targets/refresh
                             │ Authorization: Bearer <JWT>
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     API Gateway / Router                         │
│  ┌────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ AuthMiddleware │→│ AdminMiddleware │→│ HandleRefresh   │  │
│  │ (JWT verify)   │  │ (role=admin)    │  │ Targets         │  │
│  └────────────────┘  └─────────────────┘  └────────┬────────┘  │
└────────────────────────────────────────────────────┼────────────┘
                                                      │
                             ┌────────────────────────┘
                             │ RefreshNow()
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    RefreshManager (TN-048)                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ • Rate Limiting (1 req/min)                              │  │
│  │ • Single-Flight (only 1 refresh at a time)               │  │
│  │ • Async Execution (goroutine)                            │  │
│  │ • Retry Logic (exponential backoff)                      │  │
│  │ • Metrics (7 Prometheus metrics)                         │  │
│  └─────────────────────────┬────────────────────────────────┘  │
└────────────────────────────┼───────────────────────────────────┘
                             │ DiscoverTargets(ctx)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│              TargetDiscoveryManager (TN-047)                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ • K8s Secrets List (label selector)                      │  │
│  │ • Parse & Validate (base64 decode, JSON unmarshal)       │  │
│  │ • Update In-Memory Cache (atomic swap)                   │  │
│  │ • Record Statistics (total/valid/invalid)                │  │
│  └─────────────────────────┬────────────────────────────────┘  │
└────────────────────────────┼───────────────────────────────────┘
                             │ ListSecrets(namespace, labelSelector)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes API Server                        │
│  • Secrets in namespace "alert-history"                         │
│  • Label: "app=alert-history-target"                            │
└─────────────────────────────────────────────────────────────────┘
```

### Request Flow (Async Pattern)

```
Client                 Handler              RefreshManager        DiscoveryManager      K8s API
  │                       │                       │                       │                │
  │ POST /refresh         │                       │                       │                │
  ├──────────────────────>│                       │                       │                │
  │                       │                       │                       │                │
  │                       │ 1. Generate UUID      │                       │                │
  │                       │    (request_id)       │                       │                │
  │                       │                       │                       │                │
  │                       │ 2. Check Rate Limit   │                       │                │
  │                       │    (last call < 60s?) │                       │                │
  │                       │                       │                       │                │
  │                       │ 3. RefreshNow()       │                       │                │
  │                       ├──────────────────────>│                       │                │
  │                       │                       │                       │                │
  │                       │                       │ 4. Check inProgress   │                │
  │                       │                       │    (mutex lock)       │                │
  │                       │                       │                       │                │
  │                       │                       │ 5. Spawn goroutine    │                │
  │                       │                       │    (async)            │                │
  │                       │                       │    ┌──────────────────┤                │
  │                       │                       │    │                  │                │
  │                       │ 6. Return nil         │    │                  │                │
  │                       │<──────────────────────┤    │                  │                │
  │                       │                       │    │                  │                │
  │ 202 Accepted          │                       │    │                  │                │
  │ {request_id, ...}     │                       │    │                  │                │
  │<──────────────────────┤                       │    │                  │                │
  │                       │                       │    │                  │                │
  │                       │                       │    │ 7. DiscoverTargets(ctx)           │
  │                       │                       │    ├─────────────────>│                │
  │                       │                       │    │                  │                │
  │                       │                       │    │                  │ 8. ListSecrets │
  │                       │                       │    │                  ├───────────────>│
  │                       │                       │    │                  │                │
  │                       │                       │    │                  │ 9. []Secret    │
  │                       │                       │    │                  │<───────────────┤
  │                       │                       │    │                  │                │
  │                       │                       │    │ 10. Parse/Validate                │
  │                       │                       │    │                  │                │
  │                       │                       │    │ 11. Update Cache │                │
  │                       │                       │    │<─────────────────┤                │
  │                       │                       │    │                  │                │
  │                       │                       │    │ 12. Record Metrics                │
  │                       │                       │    └──────────────────┘                │
  │                       │                       │                       │                │
  │                       │                       │ 13. Update state      │                │
  │                       │                       │     (success/failed)  │                │
  │                       │                       │                       │                │
```

### Component Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                         handlers/publishing_refresh.go                │
│                                                                        │
│  func HandleRefreshTargets(refreshMgr RefreshManager) http.HandlerFunc│
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ 1. Generate request_id (UUID)                                  │  │
│  │ 2. Log request (INFO level)                                    │  │
│  │ 3. Call refreshMgr.RefreshNow()                                │  │
│  │ 4. Handle errors:                                              │  │
│  │    • ErrRefreshInProgress   → 503 Service Unavailable         │  │
│  │    • ErrRateLimitExceeded   → 429 Too Many Requests           │  │
│  │    • ErrNotStarted          → 503 Service Unavailable         │  │
│  │    • Other                  → 500 Internal Server Error       │  │
│  │ 5. Return 202 Accepted (success)                              │  │
│  │ 6. Increment metrics (publishing_refresh_requests_total)      │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ refreshMgr.RefreshNow()
                                    ▼
┌──────────────────────────────────────────────────────────────────────┐
│               business/publishing/refresh_manager_impl.go             │
│                                                                        │
│  type DefaultRefreshManager struct { ... }                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ RefreshNow() error                                             │  │
│  │ ┌──────────────────────────────────────────────────────────┐  │  │
│  │ │ 1. Check rate limit (rateMu.Lock)                        │  │  │
│  │ │    if time.Since(lastManualRefresh) < 60s               │  │  │
│  │ │        return ErrRateLimitExceeded                       │  │  │
│  │ │                                                           │  │  │
│  │ │ 2. Check if already running (mu.Lock)                    │  │  │
│  │ │    if inProgress                                         │  │  │
│  │ │        return ErrRefreshInProgress                       │  │  │
│  │ │                                                           │  │  │
│  │ │ 3. Set inProgress = true                                 │  │  │
│  │ │    Update lastManualRefresh = now                        │  │  │
│  │ │                                                           │  │  │
│  │ │ 4. Spawn goroutine:                                      │  │  │
│  │ │    go rm.executeRefresh(ctx, "manual")                   │  │  │
│  │ │                                                           │  │  │
│  │ │ 5. Return nil (success)                                  │  │  │
│  │ └──────────────────────────────────────────────────────────┘  │  │
│  │                                                                │  │
│  │ executeRefresh(ctx context.Context, trigger string) error     │  │
│  │ ┌──────────────────────────────────────────────────────────┐  │  │
│  │ │ 1. Record start time                                     │  │  │
│  │ │ 2. Set state = "in_progress"                             │  │  │
│  │ │ 3. Call discovery.DiscoverTargets(ctx)                   │  │  │
│  │ │ 4. Handle result:                                        │  │  │
│  │ │    • Success: state = "success"                          │  │  │
│  │ │    • Error:   state = "failed", increment retries        │  │  │
│  │ │ 5. Record metrics:                                       │  │  │
│  │ │    • publishing_refresh_duration_seconds.Observe(...)    │  │  │
│  │ │    • publishing_refresh_requests_total{status}.Inc()     │  │  │
│  │ │    • publishing_refresh_last_success_timestamp (if OK)   │  │  │
│  │ │ 6. Set inProgress = false                                │  │  │
│  │ │ 7. Log completion (INFO/ERROR)                           │  │  │
│  │ └──────────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ discovery.DiscoverTargets(ctx)
                                    ▼
┌──────────────────────────────────────────────────────────────────────┐
│          business/publishing/discovery_manager_impl.go                │
│                                                                        │
│  type DefaultDiscoveryManager struct { ... }                          │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ DiscoverTargets(ctx context.Context) error                     │  │
│  │ ┌──────────────────────────────────────────────────────────┐  │  │
│  │ │ 1. List K8s secrets:                                     │  │  │
│  │ │    secrets, err := k8sClient.ListSecrets(                │  │  │
│  │ │        namespace,                                        │  │  │
│  │ │        "app=alert-history-target"                        │  │  │
│  │ │    )                                                      │  │  │
│  │ │                                                           │  │  │
│  │ │ 2. Parse each secret:                                    │  │  │
│  │ │    for _, secret := range secrets {                      │  │  │
│  │ │        rawData := secret.Data["target"]                  │  │  │
│  │ │        decoded := base64.Decode(rawData)                 │  │  │
│  │ │        target := json.Unmarshal(decoded)                 │  │  │
│  │ │        validate(target)                                  │  │  │
│  │ │        validTargets = append(target)                     │  │  │
│  │ │    }                                                      │  │  │
│  │ │                                                           │  │  │
│  │ │ 3. Update cache (atomic swap):                           │  │  │
│  │ │    mu.Lock()                                             │  │  │
│  │ │    cache = validTargets                                  │  │  │
│  │ │    mu.Unlock()                                           │  │  │
│  │ │                                                           │  │  │
│  │ │ 4. Record statistics:                                    │  │  │
│  │ │    totalDiscovered = len(secrets)                        │  │  │
│  │ │    validTargets = len(validTargets)                      │  │  │
│  │ │    invalidTargets = totalDiscovered - validTargets       │  │  │
│  │ │                                                           │  │  │
│  │ │ 5. Update metrics:                                       │  │  │
│  │ │    discovery_targets_total{status="valid"}.Set(valid)    │  │  │
│  │ │    discovery_targets_total{status="invalid"}.Set(invalid)│  │  │
│  │ │                                                           │  │  │
│  │ │ 6. Return nil (success)                                  │  │  │
│  │ └──────────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

## 📡 API Specification

### Endpoint Definition

```yaml
openapi: 3.0.3
info:
  title: Alert History Service API
  version: 2.0.0

paths:
  /api/v2/publishing/targets/refresh:
    post:
      summary: Trigger manual target refresh
      description: |
        Immediately triggers discovery and refresh of publishing targets from Kubernetes Secrets.

        **Async Behavior:**
        - Returns 202 Accepted immediately
        - Refresh executes in background (~2s)
        - Only 1 refresh can run at a time

        **Rate Limiting:**
        - Max 1 manual refresh per minute
        - Rate limit per server instance (not distributed)

        **Use Cases:**
        - Emergency target updates during incidents
        - CI/CD automation after infrastructure changes
        - Testing new target configurations

      tags:
        - Targets Management

      security:
        - BearerAuth: []

      x-rbac-roles:
        - admin

      requestBody:
        description: No body required (endpoint accepts empty POST)
        required: false

      responses:
        '202':
          description: Refresh triggered successfully (async)
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
                    example: "Refresh triggered"
                  request_id:
                    type: string
                    format: uuid
                    example: "550e8400-e29b-41d4-a716-446655440000"
                  refresh_started_at:
                    type: string
                    format: date-time
                    example: "2025-11-17T10:30:45Z"
              examples:
                success:
                  summary: Successful trigger
                  value:
                    message: "Refresh triggered"
                    request_id: "550e8400-e29b-41d4-a716-446655440000"
                    refresh_started_at: "2025-11-17T10:30:45Z"

        '429':
          description: Rate limit exceeded
          headers:
            Retry-After:
              schema:
                type: integer
              description: Seconds until rate limit resets
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              examples:
                rate_limit:
                  summary: Rate limit exceeded
                  value:
                    error: "rate_limit_exceeded"
                    message: "Max 1 refresh per minute"
                    retry_after_seconds: 45

        '503':
          description: Service temporarily unavailable
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              examples:
                in_progress:
                  summary: Refresh already running
                  value:
                    error: "refresh_in_progress"
                    message: "Target refresh already running"
                    started_at: "2025-11-17T10:30:00Z"
                not_started:
                  summary: Manager not started
                  value:
                    error: "manager_not_started"
                    message: "Refresh manager not started"

        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              examples:
                unknown_error:
                  summary: Unexpected error
                  value:
                    error: "internal_error"
                    message: "Internal server error"

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    ErrorResponse:
      type: object
      properties:
        error:
          type: string
          description: Machine-readable error code
        message:
          type: string
          description: Human-readable error message
        retry_after_seconds:
          type: integer
          description: Seconds to wait before retry (rate limit only)
        started_at:
          type: string
          format: date-time
          description: Refresh start time (in_progress only)
```

## 🔐 Security Design

### Authentication & Authorization Flow

```
1. Client Request
   └─> Header: Authorization: Bearer <JWT_TOKEN>

2. AuthMiddleware (router.go)
   ├─> Validate JWT signature (RS256)
   ├─> Verify exp, nbf, iat claims
   ├─> Extract user_id, roles from claims
   └─> Set context: ctx = WithUser(ctx, user)

3. AdminMiddleware (router.go)
   ├─> Get user from context
   ├─> Check user.Roles contains "admin"
   └─> If not admin → 403 Forbidden

4. HandleRefreshTargets (handler)
   ├─> Execute business logic
   └─> Audit log: user_id, IP, action, result
```

### Security Controls

| Control | Implementation | Purpose |
|---------|----------------|---------|
| **Authentication** | JWT Bearer token (RS256) | Verify user identity |
| **Authorization** | RBAC (role=admin only) | Limit access to admins |
| **Rate Limiting** | 1 req/min per instance | Prevent abuse / DoS |
| **Request Validation** | Reject non-empty body | Prevent injection |
| **Request Size Limit** | Max 1KB | Prevent payload attacks |
| **Audit Logging** | Log all attempts | Security monitoring |
| **Security Headers** | CSP, HSTS, X-Frame-Options | Browser protection |
| **HTTPS Only** | TLS 1.3 required | Encrypt in transit |

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **T1: Unauthorized Access** | JWT auth + RBAC (admin only) |
| **T2: Token Theft** | Short-lived tokens (15m), refresh rotation |
| **T3: DoS via Rapid Refresh** | Rate limiting (1 req/min) + single-flight pattern |
| **T4: K8s API DoS** | Max 1 discovery at a time, timeout 30s |
| **T5: Data Injection** | Validate request body (must be empty) |
| **T6: MITM Attacks** | HTTPS only, HSTS header |
| **T7: XSS via Response** | Content-Type: application/json, no HTML |

## 📊 Data Formats

### Request Format

```http
POST /api/v2/publishing/targets/refresh HTTP/1.1
Host: alert-history.example.com
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
Content-Length: 0

(no body)
```

### Success Response (202)

```json
{
  "message": "Refresh triggered",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "refresh_started_at": "2025-11-17T10:30:45.123Z"
}
```

### Error Response (429 Rate Limit)

```json
{
  "error": "rate_limit_exceeded",
  "message": "Max 1 refresh per minute",
  "retry_after_seconds": 45
}
```

### Error Response (503 In Progress)

```json
{
  "error": "refresh_in_progress",
  "message": "Target refresh already running",
  "started_at": "2025-11-17T10:30:00.000Z"
}
```

## 🎯 Error Scenarios

### Scenario 1: Rate Limit Exceeded

**Trigger:** Second request within 60 seconds

**Flow:**
1. Request 1 at `10:30:00` → 202 Accepted
2. Request 2 at `10:30:30` → 429 Too Many Requests
3. Wait until `10:31:00`
4. Request 3 at `10:31:05` → 202 Accepted

**Response:**
```json
{
  "error": "rate_limit_exceeded",
  "message": "Max 1 refresh per minute",
  "retry_after_seconds": 30
}
```

**Metrics:**
- `publishing_refresh_rate_limit_exceeded_total` +1
- `publishing_refresh_requests_total{status="rate_limited"}` +1

### Scenario 2: Refresh Already Running

**Trigger:** Request while background refresh in progress

**Flow:**
1. Background refresh started at `10:30:00`
2. Manual request at `10:30:01` (refresh still running)
3. Handler checks `inProgress` flag → true
4. Return 503 Service Unavailable

**Response:**
```json
{
  "error": "refresh_in_progress",
  "message": "Target refresh already running",
  "started_at": "2025-11-17T10:30:00.000Z"
}
```

**Metrics:**
- `publishing_refresh_requests_total{status="in_progress"}` +1

### Scenario 3: RefreshManager Not Started

**Trigger:** Application started but `Start()` not called on RefreshManager

**Flow:**
1. App initializes RefreshManager but doesn't call `Start()`
2. Request arrives
3. Handler calls `RefreshNow()` → returns `ErrNotStarted`
4. Return 503 Service Unavailable

**Response:**
```json
{
  "error": "manager_not_started",
  "message": "Refresh manager not started"
}
```

### Scenario 4: K8s API Failure

**Trigger:** K8s API unavailable or permission denied

**Flow:**
1. Request arrives → 202 Accepted (async)
2. Background goroutine calls `DiscoverTargets()`
3. K8s API call fails → returns error
4. RefreshManager logs error, updates state to "failed"
5. Metrics incremented: `publishing_refresh_errors_total{error_type="k8s_api"}`
6. **Client doesn't see error** (async pattern)

**Observability:**
- Log: `ERROR: Target discovery failed, error=k8s api unavailable, request_id=...`
- Metric: `publishing_refresh_errors_total{error_type="k8s_api"}` +1
- Metric: `publishing_refresh_requests_total{status="error"}` +1

## 🔍 Edge Cases

| Edge Case | Behavior | Rationale |
|-----------|----------|-----------|
| **Empty K8s secrets list** | Success, cache cleared | Valid state (no targets configured) |
| **All secrets invalid** | Success, cache cleared | Partial success (discovery worked, validation failed) |
| **K8s API timeout** | Error, keep old cache | Stale data better than no data |
| **Concurrent manual requests** | First succeeds, others get 429/503 | Protect K8s API |
| **Request with body** | 400 Bad Request | Validation (no body expected) |
| **Request > 1KB** | 413 Payload Too Large | Security (size limit) |
| **Non-admin user** | 403 Forbidden | Authorization (RBAC) |
| **Invalid JWT** | 401 Unauthorized | Authentication |
| **Refresh during shutdown** | 503 Service Unavailable | Graceful shutdown |

## 📈 Performance Considerations

### Latency Breakdown

**Target: P95 ≤ 100ms**

```
Handler execution:
├─ JWT validation:         ~5ms   (middleware)
├─ RBAC check:             ~1ms   (middleware)
├─ Request ID generation:  ~0.1ms (UUID v4)
├─ Rate limit check:       ~0.5ms (mutex + time comparison)
├─ RefreshNow() call:      ~0.5ms (mutex + goroutine spawn)
├─ JSON marshaling:        ~1ms   (response body)
└─ Network I/O:            ~10ms  (client latency)
──────────────────────────────────
Total:                     ~18ms  (well under 100ms target ✅)
```

**Refresh execution (background, not in latency):**
```
Refresh execution:
├─ K8s API call:           ~800ms  (network + API processing)
├─ Base64 decode:          ~10ms   (20 secrets)
├─ JSON unmarshal:         ~20ms   (20 secrets)
├─ Validation:             ~5ms    (20 targets)
├─ Cache update:           ~1ms    (atomic swap)
└─ Metrics update:         ~2ms    (Prometheus)
──────────────────────────────────
Total:                     ~838ms  (< 2s target ✅)
```

### Optimization Strategies

1. **Async Pattern**: Handler returns immediately (202), refresh in background
2. **Single-Flight**: Only 1 refresh at a time (no duplicate K8s calls)
3. **Mutex Optimization**: Separate locks for rate limiting vs state
4. **Connection Pooling**: K8s client reuses connections
5. **Context Timeout**: 30s timeout on K8s API calls

## 🎓 Architecture Decision Records (ADRs)

### ADR-1: Async Execution Pattern

**Decision:** Return 202 Accepted immediately, execute refresh in background goroutine

**Alternatives Considered:**
1. **Sync execution** (return 200 after completion)
   - ❌ Slow (2s latency)
   - ❌ Client timeout risk
   - ❌ Poor UX
2. **Job queue** (submit to work queue)
   - ❌ Over-engineered for simple use case
   - ❌ Additional infra (Redis/RabbitMQ)
   - ❌ Complexity

**Rationale:**
- ✅ Fast response (<100ms)
- ✅ No client timeout risk
- ✅ Standard HTTP pattern (202 Accepted)
- ✅ Simple implementation (goroutine)

### ADR-2: Rate Limiting (1 req/min)

**Decision:** Hard-coded 1 manual refresh per minute, not configurable

**Alternatives Considered:**
1. **No rate limit**
   - ❌ DDoS risk
   - ❌ K8s API abuse
2. **Configurable rate limit** (env var)
   - ❌ Operators may set too high
   - ❌ Defeats security purpose

**Rationale:**
- ✅ Protects K8s API from abuse
- ✅ Forces intentional use (not auto-retry loops)
- ✅ 60s reasonable for manual operations
- ✅ Simplicity (no config drift)

**Note:** Periodic auto-refresh (5m) NOT affected by rate limit

### ADR-3: Single-Flight Pattern

**Decision:** Only 1 refresh (manual or auto) can run at a time

**Alternatives Considered:**
1. **Parallel refreshes**
   - ❌ Duplicate K8s API calls (waste)
   - ❌ Race conditions on cache update
   - ❌ K8s API load
2. **Queue multiple requests**
   - ❌ Complexity (queue management)
   - ❌ Confusing UX (why wait?)

**Rationale:**
- ✅ Prevents duplicate work
- ✅ Protects K8s API
- ✅ Atomic cache updates
- ✅ Clear error (503 if busy)

## 🧪 Testing Strategy

### Unit Tests

**File:** `go-app/cmd/server/handlers/publishing_refresh_test.go`

```
1. TestHandleRefreshTargets_Success
   • Mock RefreshManager returns nil
   • Expect: 202 Accepted
   • Verify: response has request_id, refresh_started_at

2. TestHandleRefreshTargets_RateLimitExceeded
   • Mock returns ErrRateLimitExceeded
   • Expect: 429 Too Many Requests
   • Verify: response has retry_after_seconds

3. TestHandleRefreshTargets_InProgress
   • Mock returns ErrRefreshInProgress
   • Expect: 503 Service Unavailable
   • Verify: response has started_at timestamp

4. TestHandleRefreshTargets_NotStarted
   • Mock returns ErrNotStarted
   • Expect: 503 Service Unavailable
   • Verify: response has manager_not_started error

5. TestHandleRefreshTargets_UnknownError
   • Mock returns generic error
   • Expect: 500 Internal Server Error
   • Verify: generic error response

6. TestHandleRefreshTargets_ConcurrentRequests
   • Spawn 10 goroutines calling handler
   • Expect: 1x 202, 9x 429/503
   • Verify: thread safety
```

### Integration Tests

**File:** `go-app/cmd/server/handlers/publishing_refresh_integration_test.go`

```
1. TestRefreshEndpoint_EndToEnd
   • Real RefreshManager + Mock K8s client
   • Call endpoint → verify refresh executed
   • Verify targets updated in cache

2. TestRefreshEndpoint_RateLimiting
   • Call endpoint twice rapidly
   • First: 202, Second: 429
   • Wait 60s, third: 202

3. TestRefreshEndpoint_Authentication
   • No token: 401
   • Invalid token: 401
   • Non-admin token: 403
   • Admin token: 202

4. TestRefreshEndpoint_K8sFailure
   • Mock K8s API returns error
   • Endpoint: 202 (async)
   • Verify: metrics show error, cache unchanged
```

### Performance Benchmarks

**File:** `go-app/cmd/server/handlers/publishing_refresh_bench_test.go`

```
BenchmarkHandleRefreshTargets_Success
BenchmarkHandleRefreshTargets_RateLimited
BenchmarkHandleRefreshTargets_Concurrent
```

**Targets:**
- `BenchmarkHandleRefreshTargets_Success`: < 100ms/op
- `BenchmarkHandleRefreshTargets_RateLimited`: < 10ms/op (fast path)
- `BenchmarkHandleRefreshTargets_Concurrent`: 100 req/s sustained

## 📚 Dependencies

### Required Components
1. **RefreshManager** (TN-048) - ✅ Complete
2. **TargetDiscoveryManager** (TN-047) - ✅ Complete
3. **AuthMiddleware** - ✅ Exists
4. **AdminMiddleware** - ✅ Exists
5. **Router** - ✅ Exists (needs endpoint registration)

### External Dependencies
1. **K8s API** - Required for discovery
2. **Prometheus** - Required for metrics
3. **JWT library** - Required for auth

## 🚀 Deployment

### Configuration

**Environment Variables:**
- `K8S_NAMESPACE`: Kubernetes namespace for secrets (default: `alert-history`)
- `TARGET_LABEL_SELECTOR`: Label selector (default: `app=alert-history-target`)
- `REFRESH_INTERVAL`: Auto-refresh interval (default: `5m`)
- `REFRESH_TIMEOUT`: K8s API timeout (default: `30s`)

**No configuration needed for:**
- Rate limit (hardcoded 1 req/min)
- Request size limit (hardcoded 1KB)
- Admin role requirement

### Health Checks

**Endpoint:** `GET /healthz`

**Refresh Health Criteria:**
- ✅ Healthy if: `time.Since(lastSuccessfulRefresh) < 10m`
- ⚠️ Degraded if: `10m < time.Since(lastSuccessfulRefresh) < 30m`
- ❌ Unhealthy if: `time.Since(lastSuccessfulRefresh) > 30m`

## 📊 Observability

### Prometheus Metrics

```prometheus
# Requests total (by status and trigger)
publishing_refresh_requests_total{status="success|error|rate_limited|in_progress", trigger="manual|auto"} counter

# Request duration (endpoint latency, not refresh execution)
publishing_refresh_api_duration_seconds histogram

# Refresh execution duration (background task)
publishing_refresh_duration_seconds histogram

# Errors by type
publishing_refresh_errors_total{error_type="k8s_api|parsing|validation|timeout"} counter

# Rate limit hits
publishing_refresh_rate_limit_exceeded_total counter

# Current state
publishing_refresh_in_progress gauge  # 0 or 1

# Last successful refresh
publishing_refresh_last_success_timestamp gauge  # Unix timestamp
```

### Structured Logging

```go
// Success
logger.Info("Manual refresh triggered",
    "request_id", requestID,
    "user_id", user.ID,
    "ip", r.RemoteAddr,
    "refresh_started_at", time.Now().UTC())

// Rate limit
logger.Warn("Manual refresh rate limit exceeded",
    "request_id", requestID,
    "user_id", user.ID,
    "ip", r.RemoteAddr,
    "retry_after_seconds", retryAfter)

// Error
logger.Error("Manual refresh failed",
    "request_id", requestID,
    "error", err,
    "error_type", errorType)
```

### Tracing (Request ID)

- Generate UUID for each request
- Propagate `request_id` through entire pipeline:
  - Handler → RefreshManager → DiscoveryManager → K8s client
- Include `request_id` in all logs for correlation
- Return `request_id` in response for client tracking

## 🔧 Troubleshooting

### Issue: 429 Rate Limit Exceeded

**Symptoms:**
- Response: `{"error": "rate_limit_exceeded", ...}`
- Metric: `publishing_refresh_rate_limit_exceeded_total` increasing

**Diagnosis:**
```bash
# Check last manual refresh time
curl -s https://alert-history/api/v2/publishing/targets/status | jq '.last_refresh'

# Check rate limit metric
curl -s https://alert-history/metrics | grep publishing_refresh_rate_limit
```

**Solutions:**
1. Wait 60 seconds between manual refreshes
2. Use automatic refresh (5m interval) if not urgent
3. Check for retry loops in automation

### Issue: 503 Refresh In Progress

**Symptoms:**
- Response: `{"error": "refresh_in_progress", ...}`
- Metric: `publishing_refresh_in_progress` = 1

**Diagnosis:**
```bash
# Check refresh status
curl -s https://alert-history/api/v2/publishing/targets/status | jq '.status'

# Check refresh duration
curl -s https://alert-history/metrics | grep publishing_refresh_duration
```

**Solutions:**
1. Wait for current refresh to complete (~2s normally)
2. If stuck (> 30s), check K8s API connectivity
3. If hung, restart service (graceful shutdown kills goroutine)

### Issue: Refresh Completes But No Targets

**Symptoms:**
- Response: 202 Accepted (success)
- But `GET /publishing/targets` returns empty list

**Diagnosis:**
```bash
# Check discovery metrics
curl -s https://alert-history/metrics | grep discovery_targets_total

# Check logs for parsing errors
kubectl logs -n alert-history deploy/alert-history | grep "invalid target"
```

**Solutions:**
1. Verify K8s secrets exist: `kubectl get secrets -l app=alert-history-target`
2. Check secret format (base64 encoded JSON)
3. Verify validation rules (required fields, URL format)
4. Check namespace and label selector config
