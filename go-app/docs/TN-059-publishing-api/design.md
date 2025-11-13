# TN-059: Publishing API Endpoints - Architecture Design

**Version:** 1.0
**Date:** 2025-11-13
**Status:** APPROVED
**Quality Target:** 150% (Grade A+)
**Author:** Enterprise Architecture Team

---

## Executive Summary

This document defines the **technical architecture** for the unified Publishing API, including:
- Layered architecture design
- HTTP routing and middleware stack
- OpenAPI 3.0 specification structure
- Authentication & authorization mechanisms
- Performance optimization strategies
- Security hardening measures

The design builds upon the **proven architecture patterns** from TN-056, TN-057, and TN-058 while introducing enterprise-grade enhancements for scalability, security, and developer experience.

---

## 1. System Architecture

### 1.1 High-Level Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                         API Gateway Layer                       │
│  ┌──────────────┬──────────────┬──────────────┬──────────────┐│
│  │  Routing     │  Versioning  │  CORS        │  Compression ││
│  │  (mux)       │  (path)      │  (middleware)│  (gzip)      ││
│  └──────────────┴──────────────┴──────────────┴──────────────┘│
└────────────────────────┬───────────────────────────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐             ┌────────▼─────────┐
│  Auth Middleware │             │  Rate Limiter    │
│  (API Key/JWT)   │             │  (Token Bucket)  │
└───────┬──────────┘             └────────┬─────────┘
        │                                 │
        └────────────────┬────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐             ┌────────▼─────────┐
│  Validation      │             │  Request ID      │
│  Middleware      │             │  Tracking        │
└───────┬──────────┘             └────────┬─────────┘
        │                                 │
        └────────────────┬────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐             ┌────────▼─────────┐
│  Metrics         │             │  Logging         │
│  (Prometheus)    │             │  (slog)          │
└───────┬──────────┘             └────────┬─────────┘
        │                                 │
        └────────────────┬────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │         Handler Router          │
        │     (gorilla/mux patterns)      │
        └────────────────┬────────────────┘
                         │
        ┌────────────────┴────────────────────────────┐
        │                │                │           │
┌───────▼──────┐  ┌──────▼──────┐  ┌─────▼──────┐  ┌▼──────────┐
│ Publishing   │  │  Stats      │  │ Parallel   │  │ Class.    │
│ Handlers     │  │  Handlers   │  │ Handlers   │  │ Handlers  │
│ (TN-056)     │  │  (TN-057)   │  │ (TN-058)   │  │ (NEW)     │
└───────┬──────┘  └──────┬──────┘  └─────┬──────┘  └┬──────────┘
        │                │                │           │
        └────────────────┴────────────────┴───────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐             ┌────────▼─────────┐
│  Business Layer  │             │  Infrastructure  │
│  (Services)      │◄────────────┤  (Repositories)  │
└──────────────────┘             └──────────────────┘
```

---

### 1.2 Component Responsibilities

#### **API Gateway Layer**
- **Routing:** URL pattern matching, method validation
- **Versioning:** Path-based versioning (/api/v2)
- **CORS:** Cross-origin request handling
- **Compression:** gzip/brotli response compression

#### **Security Layer**
- **Authentication:** API key validation, JWT token verification
- **Authorization:** Role-based access control (RBAC)
- **Rate Limiting:** Token bucket algorithm (100 req/min)
- **Input Validation:** JSON schema validation

#### **Observability Layer**
- **Request Tracking:** Unique request ID generation
- **Metrics:** Prometheus instrumentation (per-endpoint)
- **Logging:** Structured logging with slog
- **Tracing:** OpenTelemetry integration (future)

#### **Handler Layer**
- **HTTP Request Handling:** Parse, validate, dispatch
- **Response Formatting:** JSON marshaling, error handling
- **Business Logic Delegation:** Call services/repositories
- **Cache Management:** ETags, Cache-Control headers

#### **Business Layer**
- **Publishing Services:** Queue, discovery, health, parallel
- **Classification Service:** LLM-based classification
- **Alert Processing:** Enrichment, filtering, deduplication

#### **Infrastructure Layer**
- **Data Access:** PostgreSQL, Redis repositories
- **External Integrations:** Kubernetes, LLM providers
- **System Resources:** Database pools, cache connections

---

## 2. API Structure & Routing

### 2.1 URL Hierarchy

```
/api/v2/
├── publishing/                      # Publishing System (27 endpoints)
│   ├── targets                      # Target Management (7)
│   │   ├── GET    /                 # List all targets
│   │   ├── GET    /{name}           # Get target by name
│   │   ├── POST   /refresh          # Refresh discovery
│   │   ├── POST   /{name}/test      # Test target connectivity
│   │   └── health/                  # Health Monitoring (4)
│   │       ├── GET    /             # All targets health
│   │       ├── GET    /{name}       # Target health by name
│   │       ├── POST   /{name}/check # Force health check
│   │       └── GET    /stats        # Health statistics
│   │
│   ├── queue/                       # Publishing Queue (7)
│   │   ├── GET    /status           # Queue status
│   │   ├── GET    /stats            # Detailed statistics
│   │   ├── POST   /submit           # Submit alert
│   │   └── jobs/                    # Job Management (3)
│   │       ├── GET    /             # List jobs (filters)
│   │       └── GET    /{id}         # Get job by ID
│   │
│   ├── dlq/                         # Dead Letter Queue (3)
│   │   ├── GET    /                 # List DLQ entries
│   │   ├── POST   /{id}/replay      # Replay entry
│   │   └── DELETE /purge            # Purge old entries
│   │
│   ├── parallel/                    # Parallel Publishing (4)
│   │   ├── POST   /targets          # Publish to specific targets
│   │   ├── POST   /all              # Publish to all
│   │   ├── POST   /healthy          # Publish to healthy only
│   │   └── GET    /status           # Parallel publisher status
│   │
│   ├── metrics/                     # Metrics & Stats (4)
│   │   ├── GET    /raw              # Raw Prometheus metrics (JSON)
│   │   ├── GET    /stats            # Aggregated statistics
│   │   ├── GET    /trends           # Historical trends
│   │   └── GET    /targets/{name}   # Per-target stats
│   │
│   └── health                       # Overall Health (1)
│       └── GET    /                 # Publishing system health
│
├── classification/                  # LLM Classification (NEW, 3 endpoints)
│   ├── GET    /stats                # Classification statistics
│   ├── POST   /classify             # Manual classification
│   └── GET    /models               # Available LLM models
│
├── enrichment/                      # Alert Enrichment (2 endpoints)
│   ├── GET    /mode                 # Current enrichment mode
│   └── POST   /mode                 # Switch enrichment mode
│
├── health                           # System Health (1 endpoint)
│   └── GET    /                     # Overall system health
│
└── docs                             # API Documentation (1 endpoint)
    └── GET    /                     # Swagger UI

/api/v1/publishing/                  # Legacy Endpoints (Backward Compat)
└── [All TN-056 endpoints]           # Maintained for 12 months
```

**Total Endpoints:** 33 (27 publishing + 3 classification + 2 enrichment + 1 health)

---

### 2.2 Routing Implementation (gorilla/mux)

```go
package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
)

// SetupRouter configures all API routes
func SetupRouter() *mux.Router {
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(RequestIDMiddleware)
	router.Use(LoggingMiddleware)
	router.Use(MetricsMiddleware)
	router.Use(CompressionMiddleware)
	router.Use(CORSMiddleware)

	// API v2 routes
	v2 := router.PathPrefix("/api/v2").Subrouter()

	// Publishing routes
	setupPublishingRoutes(v2)
	setupClassificationRoutes(v2)
	setupEnrichmentRoutes(v2)

	// Health & documentation
	v2.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	v2.HandleFunc("/docs", SwaggerUIHandler).Methods("GET")

	// API v1 routes (backward compatibility)
	v1 := router.PathPrefix("/api/v1").Subrouter()
	setupLegacyRoutes(v1)

	return router
}

// setupPublishingRoutes configures /api/v2/publishing/* routes
func setupPublishingRoutes(router *mux.Router) {
	pub := router.PathPrefix("/publishing").Subrouter()

	// Targets
	targets := pub.PathPrefix("/targets").Subrouter()
	targets.HandleFunc("", ListTargetsHandler).Methods("GET")
	targets.HandleFunc("/{name}", GetTargetHandler).Methods("GET")
	targets.HandleFunc("/refresh", RefreshTargetsHandler).Methods("POST").
		Middleware(AuthMiddleware, RateLimitMiddleware)
	targets.HandleFunc("/{name}/test", TestTargetHandler).Methods("POST").
		Middleware(AuthMiddleware)

	// Targets health
	health := targets.PathPrefix("/health").Subrouter()
	health.HandleFunc("", ListTargetsHealthHandler).Methods("GET")
	health.HandleFunc("/{name}", GetTargetHealthHandler).Methods("GET")
	health.HandleFunc("/{name}/check", CheckTargetHealthHandler).Methods("POST").
		Middleware(AuthMiddleware)
	health.HandleFunc("/stats", GetHealthStatsHandler).Methods("GET")

	// Queue
	queue := pub.PathPrefix("/queue").Subrouter()
	queue.HandleFunc("/status", GetQueueStatusHandler).Methods("GET")
	queue.HandleFunc("/stats", GetQueueStatsHandler).Methods("GET")
	queue.HandleFunc("/submit", SubmitAlertHandler).Methods("POST").
		Middleware(AuthMiddleware, ValidationMiddleware, RateLimitMiddleware)

	// Jobs
	jobs := queue.PathPrefix("/jobs").Subrouter()
	jobs.HandleFunc("", ListJobsHandler).Methods("GET")
	jobs.HandleFunc("/{id}", GetJobHandler).Methods("GET")

	// DLQ
	dlq := pub.PathPrefix("/dlq").Subrouter()
	dlq.HandleFunc("", ListDLQEntriesHandler).Methods("GET")
	dlq.HandleFunc("/{id}/replay", ReplayDLQEntryHandler).Methods("POST").
		Middleware(AuthMiddleware, AdminRoleMiddleware)
	dlq.HandleFunc("/purge", PurgeDLQHandler).Methods("DELETE").
		Middleware(AuthMiddleware, AdminRoleMiddleware)

	// Parallel publishing
	parallel := pub.PathPrefix("/parallel").Subrouter()
	parallel.HandleFunc("/targets", PublishToTargetsHandler).Methods("POST").
		Middleware(AuthMiddleware, ValidationMiddleware)
	parallel.HandleFunc("/all", PublishToAllHandler).Methods("POST").
		Middleware(AuthMiddleware, ValidationMiddleware)
	parallel.HandleFunc("/healthy", PublishToHealthyHandler).Methods("POST").
		Middleware(AuthMiddleware, ValidationMiddleware)
	parallel.HandleFunc("/status", GetParallelStatusHandler).Methods("GET")

	// Metrics
	metrics := pub.PathPrefix("/metrics").Subrouter()
	metrics.HandleFunc("/raw", GetRawMetricsHandler).Methods("GET")
	metrics.HandleFunc("/stats", GetMetricsStatsHandler).Methods("GET")
	metrics.HandleFunc("/trends", GetTrendsHandler).Methods("GET")
	metrics.HandleFunc("/targets/{name}", GetTargetMetricsHandler).Methods("GET")

	// Health
	pub.HandleFunc("/health", GetPublishingHealthHandler).Methods("GET")
}
```

---

## 3. Middleware Stack

### 3.1 Middleware Chain

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│ RequestIDMiddleware │  1. Generate/extract X-Request-ID
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ LoggingMiddleware   │  2. Log request (method, path, IP)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ MetricsMiddleware   │  3. Record Prometheus metrics
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ CORSMiddleware      │  4. Add CORS headers
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ CompressionMidd.    │  5. gzip/brotli compression
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ AuthMiddleware      │  6. Validate API key/JWT (conditional)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ RBACMiddleware      │  7. Check permissions (conditional)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ RateLimitMiddleware │  8. Token bucket rate limiting (conditional)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ ValidationMiddleware│  9. JSON schema validation (conditional)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ CacheMiddleware     │ 10. ETag/Cache-Control (conditional)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  Handler Function   │ 11. Business logic
└─────────┬───────────┘
          │
          ▼
     HTTP Response
```

---

### 3.2 Middleware Implementations

#### **RequestIDMiddleware**

```go
package middleware

import (
	"context"
	"github.com/google/uuid"
	"net/http"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDContextKey = "request_id"

// RequestIDMiddleware generates or extracts request ID
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		r = r.WithContext(ctx)

		// Add to response headers
		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r)
	})
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return id
	}
	return ""
}
```

---

#### **AuthMiddleware**

```go
package middleware

import (
	"context"
	"net/http"
	"strings"
)

const AuthorizationHeader = "Authorization"
const UserContextKey = "user"

type User struct {
	ID       string
	Username string
	Role     string // viewer, operator, admin
	APIKey   string
}

// AuthMiddleware validates API key or JWT token
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(AuthorizationHeader)
		if authHeader == "" {
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Parse authorization header
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			http.Error(w, `{"error":"Invalid Authorization header format"}`, http.StatusUnauthorized)
			return
		}

		authType := parts[0]
		authValue := parts[1]

		var user *User
		var err error

		switch authType {
		case "Bearer":
			// JWT token validation
			user, err = validateJWTToken(authValue)
		case "ApiKey":
			// API key validation
			user, err = validateAPIKey(authValue)
		default:
			http.Error(w, `{"error":"Unsupported auth type"}`, http.StatusUnauthorized)
			return
		}

		if err != nil || user == nil {
			http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RBACMiddleware checks user permissions
func RBACMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(*User)
			if !ok || user == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check role hierarchy: admin > operator > viewer
			if !hasRequiredRole(user.Role, requiredRole) {
				http.Error(w, `{"error":"Insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AdminRoleMiddleware is a convenience wrapper for admin-only endpoints
func AdminRoleMiddleware(next http.Handler) http.Handler {
	return RBACMiddleware("admin")(next)
}

func hasRequiredRole(userRole, requiredRole string) bool {
	roles := map[string]int{"viewer": 1, "operator": 2, "admin": 3}
	return roles[userRole] >= roles[requiredRole]
}

// Placeholder functions (to be implemented)
func validateJWTToken(token string) (*User, error) {
	// TODO: Implement JWT validation
	return nil, nil
}

func validateAPIKey(apiKey string) (*User, error) {
	// TODO: Implement API key validation from config/database
	return nil, nil
}
```

---

#### **RateLimitMiddleware**

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket algorithm
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // per second
		burst:    burst,
	}
}

// GetLimiter returns limiter for client (by API key or IP)
func (rl *RateLimiter) GetLimiter(clientID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[clientID] = limiter
	}

	return limiter
}

// Cleanup removes stale limiters (run periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, limiter := range rl.limiters {
		if limiter.TokensAt(time.Now()) == float64(rl.burst) {
			delete(rl.limiters, key)
		}
	}
}

// RateLimitMiddleware applies rate limiting per client
func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := NewRateLimiter(100, 20) // 100 req/min, burst 20

	// Cleanup stale limiters every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.Cleanup()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client identifier (API key or IP)
		clientID := getClientID(r)

		// Check rate limit
		if !limiter.GetLimiter(clientID).Allow() {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getClientID(r *http.Request) string {
	// Try to get API key from context
	if user, ok := r.Context().Value(UserContextKey).(*User); ok && user != nil {
		return user.APIKey
	}

	// Fallback to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
```

---

#### **ValidationMiddleware**

```go
package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidationMiddleware validates request body using struct tags
func ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation for GET, DELETE methods
		if r.Method == http.MethodGet || r.Method == http.MethodDelete {
			next.ServeHTTP(w, r)
			return
		}

		// Parse request body (placeholder - actual validation in handler)
		// This middleware can be enhanced to validate generic JSON schemas

		next.ServeHTTP(w, r)
	})
}

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Issue   string `json:"issue"`
	Hint    string `json:"hint,omitempty"`
}

// FormatValidationErrors converts validator errors to ValidationError slice
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field: e.Field(),
				Issue: e.Tag(),
				Hint:  getValidationHint(e),
			})
		}
	}

	return errors
}

func getValidationHint(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return fmt.Sprintf("Must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("Must be at most %s characters", e.Param())
	default:
		return ""
	}
}
```

---

## 4. Authentication & Authorization

### 4.1 Authentication Strategies

#### **Strategy 1: API Key** (Simple, recommended for services)

**Flow:**
```
Client → Request + Header: "Authorization: ApiKey <key>"
    │
    ▼
API Gateway → Validate API key (config.yaml или database)
    │
    ├─ Valid → Add User to context → Continue
    └─ Invalid → 401 Unauthorized
```

**Configuration (config.yaml):**
```yaml
api:
  auth:
    enabled: true
    type: api_key
    api_keys:
      - key: "ak_prod_abc123..."
        user_id: "service_monitoring"
        role: "viewer"
      - key: "ak_prod_def456..."
        user_id: "service_alertmanager"
        role: "operator"
      - key: "ak_prod_ghi789..."
        user_id: "admin_user"
        role: "admin"
```

---

#### **Strategy 2: JWT Token** (Advanced, future)

**Flow:**
```
Client → Request + Header: "Authorization: Bearer <JWT>"
    │
    ▼
API Gateway → Validate JWT signature + expiry
    │
    ├─ Valid → Extract claims → Add User to context → Continue
    └─ Invalid → 401 Unauthorized
```

**JWT Claims:**
```json
{
  "sub": "user_123",
  "username": "john.doe",
  "role": "operator",
  "exp": 1700000000,
  "iat": 1699999000,
  "iss": "alert-history-api"
}
```

---

### 4.2 Authorization (RBAC)

#### **Role Hierarchy**

```
admin (Level 3)
  │
  ├─ Full access to all endpoints
  ├─ Can refresh targets
  ├─ Can replay/purge DLQ
  └─ Can modify system configuration

operator (Level 2)
  │
  ├─ Can submit alerts
  ├─ Can test targets
  ├─ Can classify alerts
  └─ Read access to all data

viewer (Level 1)
  │
  ├─ Read-only access
  └─ Cannot modify any resources
```

#### **Endpoint Permissions**

| Endpoint | Viewer | Operator | Admin |
|----------|--------|----------|-------|
| GET /targets | ✅ | ✅ | ✅ |
| POST /targets/refresh | ❌ | ❌ | ✅ |
| POST /targets/{name}/test | ❌ | ✅ | ✅ |
| POST /submit | ❌ | ✅ | ✅ |
| POST /classify | ❌ | ✅ | ✅ |
| POST /dlq/{id}/replay | ❌ | ❌ | ✅ |
| DELETE /dlq/purge | ❌ | ❌ | ✅ |

---

## 5. Performance Optimization

### 5.1 Response Caching

#### **Caching Strategy**

| Endpoint | TTL | Cache Key | Invalidation |
|----------|-----|-----------|--------------|
| GET /targets | 30s | `targets` | On refresh |
| GET /stats | 10s | `stats` | Time-based |
| GET /metrics/raw | 5s | `metrics_raw` | Time-based |
| GET /classification/models | 5m | `classification_models` | Manual |
| GET /health | 10s | `health` | Time-based |

#### **Cache Headers**

```http
GET /api/v2/publishing/targets
Response:
  Cache-Control: max-age=30, public
  ETag: "abc123def456"
  X-Cache: HIT
  X-Cache-TTL: 25
```

**Conditional Requests:**
```http
GET /api/v2/publishing/targets
If-None-Match: "abc123def456"

Response:
  304 Not Modified
```

---

### 5.2 Connection Pooling

```go
type PublishingAPI struct {
	dbPool    *pgxpool.Pool       // PostgreSQL connection pool (max 25)
	redisPool *redis.Client       // Redis connection pool (max 10)
	httpClient *http.Client       // HTTP client (timeout 30s)
}
```

---

### 5.3 Compression

```go
// CompressionMiddleware applies gzip compression
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzip.NewWriter(w)
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}
```

---

## 6. Error Handling

### 6.1 Error Response Structure

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      {
        "field": "alert.fingerprint",
        "issue": "required field missing",
        "hint": "Provide a unique identifier (1-128 alphanumeric characters)"
      }
    ],
    "request_id": "req_abc123",
    "timestamp": "2025-11-13T10:00:00Z",
    "documentation_url": "https://docs.example.com/errors/VALIDATION_ERROR"
  }
}
```

### 6.2 Error Type Mapping

| HTTP Status | Error Code | When to Use |
|-------------|------------|-------------|
| 400 | VALIDATION_ERROR | Invalid input (schema violation) |
| 401 | AUTHENTICATION_ERROR | Missing/invalid credentials |
| 403 | AUTHORIZATION_ERROR | Insufficient permissions |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource already exists |
| 429 | RATE_LIMIT_EXCEEDED | Too many requests |
| 500 | INTERNAL_ERROR | Unexpected server error |
| 502 | LLM_ERROR | LLM service unavailable |
| 503 | SERVICE_UNAVAILABLE | Subsystem unavailable (DB, Redis) |
| 503 | TARGET_UNAVAILABLE | Publishing target down |
| 503 | PUBLISHING_QUEUE_FULL | Queue capacity reached |
| 504 | CLASSIFICATION_TIMEOUT | LLM request timeout |

---

## 7. OpenAPI Specification Structure

### 7.1 OpenAPI Metadata

```yaml
openapi: 3.0.3
info:
  title: Alert History Publishing API
  version: 2.0.0
  description: |
    Unified RESTful API for Alert History Publishing System.
    Provides endpoints for target management, alert publishing,
    queue management, metrics, and LLM classification.
  contact:
    name: Platform Team
    email: platform@example.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: https://alert-history.example.com/api/v2
    description: Production
  - url: https://alert-history-staging.example.com/api/v2
    description: Staging
  - url: http://localhost:8080/api/v2
    description: Development

tags:
  - name: Targets
    description: Publishing target management
  - name: Queue
    description: Publishing queue operations
  - name: DLQ
    description: Dead letter queue management
  - name: Parallel
    description: Parallel publishing
  - name: Metrics
    description: Metrics and statistics
  - name: Classification
    description: LLM-based alert classification
  - name: Health
    description: Health monitoring

components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: Authorization
      description: 'Format: ApiKey <your-api-key>'
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

security:
  - ApiKeyAuth: []
  - BearerAuth: []
```

---

### 7.2 Example Endpoint Documentation (OpenAPI)

```yaml
paths:
  /publishing/targets:
    get:
      tags:
        - Targets
      summary: List all publishing targets
      description: |
        Returns a list of all configured publishing targets with their
        current status, type, and configuration.
      operationId: listTargets
      parameters:
        - name: type
          in: query
          description: Filter by target type (rootly, pagerduty, slack, webhook)
          required: false
          schema:
            type: string
            enum: [rootly, pagerduty, slack, webhook]
        - name: enabled
          in: query
          description: Filter by enabled status
          required: false
          schema:
            type: boolean
        - name: limit
          in: query
          description: Maximum number of results
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 1000
            default: 100
        - name: offset
          in: query
          description: Offset for pagination
          required: false
          schema:
            type: integer
            minimum: 0
            default: 0
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TargetListResponse'
              examples:
                success:
                  value:
                    data:
                      - name: rootly-prod
                        type: rootly
                        url: https://api.rootly.com
                        enabled: true
                        format: rootly
                        headers:
                          Authorization: Bearer xxx
                      - name: pagerduty-oncall
                        type: pagerduty
                        url: https://events.pagerduty.com
                        enabled: true
                        format: pagerduty
                    pagination:
                      total: 12
                      limit: 100
                      offset: 0
                      has_more: false
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '429':
          $ref: '#/components/responses/TooManyRequests'
        '500':
          $ref: '#/components/responses/InternalError'
      security:
        - {}  # Public endpoint (no auth required)
```

---

## 8. Testing Strategy

### 8.1 Test Pyramid

```
                ┌────────────────┐
                │  E2E Tests     │  5% (k6 load tests, integration)
                │  (20 tests)    │
                └────────────────┘
              ┌────────────────────┐
              │  Integration Tests │  15% (HTTP handler tests)
              │  (60 tests)        │
              └────────────────────┘
            ┌──────────────────────────┐
            │  Unit Tests              │  80% (handler logic, middleware)
            │  (320 tests)             │
            └──────────────────────────┘
```

### 8.2 Test Coverage Targets

| Component | Target Coverage | Priority |
|-----------|----------------|----------|
| Handlers | 90%+ | CRITICAL |
| Middleware | 95%+ | CRITICAL |
| Validation | 100% | CRITICAL |
| Error handling | 90%+ | HIGH |
| Integration | 80%+ | MEDIUM |

---

## 9. Deployment Architecture

### 9.1 Production Deployment

```
┌─────────────────────────────────────────────────────────┐
│                    Load Balancer (Nginx)                │
│               (TLS termination, rate limiting)          │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼─────────┐       ┌───────▼─────────┐
│  API Instance 1 │       │  API Instance 2 │
│  (pod 1)        │       │  (pod 2)        │
└───────┬─────────┘       └───────┬─────────┘
        │                         │
        └────────────┬────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼─────────┐       ┌───────▼─────────┐
│  PostgreSQL     │       │  Redis          │
│  (StatefulSet)  │       │  (StatefulSet)  │
└─────────────────┘       └─────────────────┘
```

---

## 10. Monitoring & Observability

### 10.1 Prometheus Metrics

```prometheus
# Request metrics
api_http_requests_total{method="GET",endpoint="/targets",status="200"} 12345
api_http_request_duration_seconds{method="GET",endpoint="/targets",quantile="0.99"} 0.008

# Error metrics
api_validation_errors_total{endpoint="/submit",error_type="required_field"} 42
api_rate_limit_exceeded_total{endpoint="/submit",client="service_monitoring"} 5

# Cache metrics
api_cache_hits_total{endpoint="/targets"} 890
api_cache_misses_total{endpoint="/targets"} 110
api_cache_hit_ratio{endpoint="/targets"} 0.89
```

---

## 11. Security Hardening

### 11.1 Security Checklist

- [ ] TLS 1.2+ only
- [ ] Strong cipher suites
- [ ] API key rotation support
- [ ] Rate limiting (100 req/min)
- [ ] Input validation (all endpoints)
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS prevention (JSON escaping)
- [ ] CSRF protection (SameSite cookies)
- [ ] CORS whitelist
- [ ] Request size limits (1MB max)
- [ ] Timeout enforcement (30s)
- [ ] gosec scan (0 critical issues)

---

## 12. Conclusion

This design provides a **robust, scalable, and secure foundation** for the unified Publishing API. Key highlights:

1. **Layered Architecture:** Clear separation of concerns (gateway, auth, handlers, business)
2. **Comprehensive Middleware:** 10-layer middleware stack (auth, rate limiting, metrics)
3. **Enterprise Security:** RBAC, API keys, JWT support, rate limiting
4. **Performance Optimized:** Caching, compression, connection pooling (<10ms targets)
5. **Developer-Friendly:** OpenAPI/Swagger, consistent errors, pagination
6. **Production-Ready:** Health checks, metrics, graceful degradation

**Next Steps:** Proceed to Phase 3 (Implementation)

---

**Document Status:** ✅ **APPROVED** - Ready for implementation
**Version:** 1.0
**Last Updated:** 2025-11-13
