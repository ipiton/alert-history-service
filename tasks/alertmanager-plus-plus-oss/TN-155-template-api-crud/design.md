# TN-155: Template API (CRUD) - Technical Design

**Task ID**: TN-155
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Author**: AI Assistant
**Date**: 2025-11-25

---

## 🏗️ Architecture Overview

### Layered Architecture (Clean Architecture Pattern)

```
┌──────────────────────────────────────────────────────────────────┐
│                     HTTP Layer (Presentation)                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │           handlers/template.go (TemplateHandler)           │  │
│  │  • Request/Response DTOs                                   │  │
│  │  • Input validation                                        │  │
│  │  • HTTP status codes                                       │  │
│  │  • Middleware integration (Auth, Metrics, Logging)        │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                              ↓ depends on
┌──────────────────────────────────────────────────────────────────┐
│                  Business Layer (Domain Logic)                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │      business/template/manager.go (TemplateManager)        │  │
│  │  • CRUD orchestration                                      │  │
│  │  • Business rules validation                               │  │
│  │  • Version control logic                                   │  │
│  │  • Transaction management                                  │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │    business/template/validator.go (TemplateValidator)      │  │
│  │  • Syntax validation (TN-153)                              │  │
│  │  • Semantic validation                                     │  │
│  │  • Security checks                                         │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                              ↓ depends on
┌──────────────────────────────────────────────────────────────────┐
│              Infrastructure Layer (Data Access)                   │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  infrastructure/template/repository.go (PostgreSQL)        │  │
│  │  • CRUD operations                                         │  │
│  │  • Query builder                                           │  │
│  │  • Transaction support                                     │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  infrastructure/template/cache.go (Redis + Memory)         │  │
│  │  • L1 cache (in-memory LRU)                                │  │
│  │  • L2 cache (Redis)                                        │  │
│  │  • Cache invalidation                                      │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                              ↓ persists to
┌──────────────────────────────────────────────────────────────────┐
│                        Data Layer (PostgreSQL)                    │
│  • templates (main table)                                        │
│  • template_versions (history table)                             │
└──────────────────────────────────────────────────────────────────┘
```

---

## 📊 Database Design

### Table: `templates`

```sql
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL,
    type VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT templates_name_unique UNIQUE (name),
    CONSTRAINT templates_name_format CHECK (name ~ '^[a-z0-9_]+$'),
    CONSTRAINT templates_type_enum CHECK (type IN ('slack', 'pagerduty', 'email', 'webhook', 'generic')),
    CONSTRAINT templates_content_length CHECK (length(content) > 0 AND length(content) <= 65536),
    CONSTRAINT templates_version_positive CHECK (version > 0)
);

-- Indexes for performance
CREATE INDEX idx_templates_name ON templates(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_templates_type ON templates(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_templates_created_at ON templates(created_at DESC);
CREATE INDEX idx_templates_updated_at ON templates(updated_at DESC);
CREATE INDEX idx_templates_deleted_at ON templates(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_templates_metadata_gin ON templates USING GIN (metadata);

-- Full-text search index
CREATE INDEX idx_templates_search ON templates USING GIN (
    to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, ''))
) WHERE deleted_at IS NULL;

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**Size Estimation** (100,000 templates):
- Row size: ~2KB average (1KB content + 1KB metadata)
- Total size: ~200MB data + ~50MB indexes = **250MB**

---

### Table: `template_versions`

```sql
CREATE TABLE IF NOT EXISTS template_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    change_summary TEXT,

    -- Unique constraint on template + version
    CONSTRAINT template_versions_unique UNIQUE (template_id, version),
    CONSTRAINT template_versions_version_positive CHECK (version > 0)
);

-- Indexes
CREATE INDEX idx_template_versions_template_id ON template_versions(template_id);
CREATE INDEX idx_template_versions_template_version ON template_versions(template_id, version DESC);
CREATE INDEX idx_template_versions_created_at ON template_versions(created_at DESC);
```

**Retention Policy**: Keep last 50 versions per template (automatic cleanup via cron job)

---

## 🔧 Component Design

### 1. HTTP Handler Layer

**File**: `go-app/cmd/server/handlers/template.go`

```go
// TemplateHandler handles HTTP requests for template management
type TemplateHandler struct {
    manager   business.TemplateManager   // Business logic
    validator business.TemplateValidator // Validation
    metrics   *metrics.BusinessMetrics   // Prometheus metrics
    logger    *slog.Logger               // Structured logging
    cache     cache.Cache                // Response cache
}

// NewTemplateHandler creates a new handler
func NewTemplateHandler(
    manager business.TemplateManager,
    validator business.TemplateValidator,
    metrics *metrics.BusinessMetrics,
    logger *slog.Logger,
    cache cache.Cache,
) *TemplateHandler

// HTTP Methods (7 endpoints)
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) ValidateTemplate(w http.ResponseWriter, r *http.Request)
func (h *TemplateHandler) ListTemplateVersions(w http.ResponseWriter, r *http.Request)
```

**Request/Response DTOs**:

```go
// CreateTemplateRequest - POST /api/v2/templates
type CreateTemplateRequest struct {
    Name        string                 `json:"name" validate:"required,min=3,max=64,alphanum_underscore"`
    Type        string                 `json:"type" validate:"required,oneof=slack pagerduty email webhook generic"`
    Content     string                 `json:"content" validate:"required,max=65536"`
    Description string                 `json:"description" validate:"max=500"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// UpdateTemplateRequest - PUT /api/v2/templates/{name}
type UpdateTemplateRequest struct {
    Content     *string                `json:"content,omitempty" validate:"omitempty,max=65536"`
    Description *string                `json:"description,omitempty" validate:"omitempty,max=500"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateResponse - Response for Create/Get/Update
type TemplateResponse struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Type        string                 `json:"type"`
    Content     string                 `json:"content"`
    Description string                 `json:"description"`
    Metadata    map[string]interface{} `json:"metadata"`
    Version     int                    `json:"version"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    CreatedBy   string                 `json:"created_by,omitempty"`
    UpdatedBy   string                 `json:"updated_by,omitempty"`
}

// ListTemplatesResponse - Response for List
type ListTemplatesResponse struct {
    Templates  []TemplateSummary `json:"templates"`
    Pagination PaginationInfo    `json:"pagination"`
}

type TemplateSummary struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Type        string    `json:"type"`
    Description string    `json:"description"`
    Version     int       `json:"version"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

---

### 2. Business Logic Layer

**File**: `go-app/internal/business/template/manager.go`

```go
// TemplateManager orchestrates template operations
type TemplateManager interface {
    // CRUD operations
    CreateTemplate(ctx context.Context, req CreateTemplateInput) (*Template, error)
    GetTemplate(ctx context.Context, name string) (*Template, error)
    ListTemplates(ctx context.Context, filters ListFilters) (*TemplateList, error)
    UpdateTemplate(ctx context.Context, name string, req UpdateTemplateInput) (*Template, error)
    DeleteTemplate(ctx context.Context, name string, opts DeleteOptions) error

    // Version control
    ListVersions(ctx context.Context, name string, filters VersionFilters) (*VersionList, error)
    GetVersion(ctx context.Context, name string, version int) (*Template, error)
    RollbackToVersion(ctx context.Context, name string, version int, reason string) (*Template, error)

    // Advanced features (150%)
    BatchCreate(ctx context.Context, templates []CreateTemplateInput) ([]Template, error)
    GetDiff(ctx context.Context, name string, from, to int) (*TemplateDiff, error)
    GetStats(ctx context.Context) (*TemplateStats, error)
}

// DefaultTemplateManager implements TemplateManager
type DefaultTemplateManager struct {
    repo      TemplateRepository
    validator TemplateValidator
    cache     TemplateCache
    logger    *slog.Logger
}
```

**Business Rules**:
1. Template name must be unique
2. Content must pass TN-153 syntax validation
3. Deleted templates cannot be updated
4. Version numbers auto-increment on update
5. Rollback creates new version (preserves history)

---

### 3. Validation Layer

**File**: `go-app/internal/business/template/validator.go`

```go
// TemplateValidator validates template syntax and semantics
type TemplateValidator interface {
    // Validate template syntax using TN-153 engine
    ValidateSyntax(ctx context.Context, content string, templateType string) (*ValidationResult, error)

    // Validate with test data
    ValidateWithData(ctx context.Context, content string, data interface{}) (*ValidationResult, error)

    // Validate business rules
    ValidateBusinessRules(ctx context.Context, template *Template) error
}

type ValidationResult struct {
    Valid          bool             `json:"valid"`
    SyntaxErrors   []SyntaxError    `json:"syntax_errors"`
    Warnings       []string         `json:"warnings"`
    FunctionsUsed  []string         `json:"functions_used"`
    VariablesUsed  []string         `json:"variables_used"`
    RenderedOutput string           `json:"rendered_output,omitempty"`
}

type SyntaxError struct {
    Line       int    `json:"line"`
    Column     int    `json:"column"`
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
}
```

**Integration with TN-153**:

```go
func (v *DefaultTemplateValidator) ValidateSyntax(
    ctx context.Context,
    content string,
    templateType string,
) (*ValidationResult, error) {
    // Use TN-153 NotificationTemplateEngine
    engine := template.NewNotificationTemplateEngine(
        template.DefaultTemplateEngineOptions(),
    )

    // Create mock template data
    mockData := template.NewTemplateData(
        "firing",
        map[string]string{"alertname": "TestAlert"},
        map[string]string{"summary": "Test"},
        time.Now(),
    )

    // Try to execute template
    _, err := engine.Execute(ctx, content, mockData)

    // Parse errors and return ValidationResult
    return parseValidationResult(err)
}
```

---

### 4. Repository Layer

**File**: `go-app/internal/infrastructure/template/repository.go`

```go
// TemplateRepository handles data persistence
type TemplateRepository interface {
    // CRUD operations
    Create(ctx context.Context, template *Template) error
    GetByName(ctx context.Context, name string) (*Template, error)
    GetByID(ctx context.Context, id string) (*Template, error)
    List(ctx context.Context, filters ListFilters) ([]*Template, int, error)
    Update(ctx context.Context, template *Template) error
    Delete(ctx context.Context, name string, soft bool) error

    // Version operations
    CreateVersion(ctx context.Context, version *TemplateVersion) error
    ListVersions(ctx context.Context, templateID string, filters VersionFilters) ([]*TemplateVersion, error)
    GetVersion(ctx context.Context, templateID string, version int) (*TemplateVersion, error)

    // Utility
    Exists(ctx context.Context, name string) (bool, error)
    CountByType(ctx context.Context) (map[string]int, error)
}

// PostgresTemplateRepository implements TemplateRepository
type PostgresTemplateRepository struct {
    db     *pgxpool.Pool
    logger *slog.Logger
}
```

**Query Examples**:

```go
// List with filters
func (r *PostgresTemplateRepository) List(
    ctx context.Context,
    filters ListFilters,
) ([]*Template, int, error) {
    query := `
        SELECT id, name, type, description, version,
               created_at, updated_at, created_by
        FROM templates
        WHERE deleted_at IS NULL
          AND ($1 = '' OR type = $1)
          AND ($2 = '' OR to_tsvector('english', name || ' ' || coalesce(description, '')) @@ plainto_tsquery('english', $2))
        ORDER BY
            CASE WHEN $3 = 'name' THEN name END ASC,
            CASE WHEN $3 = 'created_at' THEN created_at END DESC
        LIMIT $4 OFFSET $5
    `

    rows, err := r.db.Query(ctx, query,
        filters.Type,
        filters.Search,
        filters.Sort,
        filters.Limit,
        filters.Offset,
    )
    // ... scan and return
}
```

---

### 5. Cache Layer

**File**: `go-app/internal/infrastructure/template/cache.go`

```go
// TemplateCache provides two-tier caching
type TemplateCache interface {
    // Get from cache (L1 → L2 → miss)
    Get(ctx context.Context, name string) (*Template, error)

    // Set in cache (L1 + L2)
    Set(ctx context.Context, template *Template) error

    // Invalidate specific template
    Invalidate(ctx context.Context, name string) error

    // Invalidate all
    InvalidateAll(ctx context.Context) error

    // Stats
    GetStats() CacheStats
}

type TwoTierTemplateCache struct {
    l1Cache *lru.Cache        // In-memory LRU (1000 entries)
    l2Cache cache.Cache       // Redis
    logger  *slog.Logger
}
```

**Cache Strategy**:
1. **L1** (memory): LRU cache, 1000 entries, ~2MB total
2. **L2** (Redis): All templates, 5min TTL
3. **Invalidation**: On create/update/delete

**Cache Key Format**: `template:v1:{name}`

---

## 🔄 API Flow Diagrams

### Create Template Flow

```
Client → POST /api/v2/templates
   ↓
Handler: Validate request body
   ↓
Handler: Check admin auth
   ↓
Manager: Validate business rules
   ↓
Validator: Check syntax (TN-153)
   ↓
Repository: Check name uniqueness
   ↓
Repository: BEGIN TRANSACTION
   ↓
Repository: INSERT INTO templates
   ↓
Repository: INSERT INTO template_versions (v1)
   ↓
Repository: COMMIT
   ↓
Cache: Invalidate (if exists)
   ↓
Metrics: Record create
   ↓
Handler: Return 201 Created
```

---

### Get Template Flow (With Cache)

```
Client → GET /api/v2/templates/{name}
   ↓
Handler: Check If-None-Match (ETag)
   ↓
Cache: Try L1 (memory)
   ├─ HIT → Return 200 OK (< 10ms)
   └─ MISS ↓
Cache: Try L2 (Redis)
   ├─ HIT → Set L1, Return 200 OK (< 50ms)
   └─ MISS ↓
Repository: SELECT FROM templates
   ↓
Cache: Set L2 + L1
   ↓
Handler: Generate ETag
   ↓
Handler: Return 200 OK (< 100ms)
```

---

### Update Template Flow

```
Client → PUT /api/v2/templates/{name}
   ↓
Handler: Validate request
   ↓
Manager: Get current template
   ↓
Validator: Validate new content
   ↓
Repository: BEGIN TRANSACTION
   ↓
Repository: UPDATE templates SET version=version+1
   ↓
Repository: INSERT INTO template_versions (old content)
   ↓
Repository: COMMIT
   ↓
Cache: Invalidate L1 + L2
   ↓
Handler: Return 200 OK
```

---

## 🎨 OpenAPI 3.0 Specification (Excerpt)

```yaml
openapi: 3.0.3
info:
  title: Template API
  version: 2.0.0
  description: REST API for notification template management

paths:
  /api/v2/templates:
    post:
      summary: Create a new template
      operationId: createTemplate
      tags: [Templates]
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTemplateRequest'
      responses:
        '201':
          description: Template created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TemplateResponse'
        '400':
          $ref: '#/components/responses/BadRequest'
        '409':
          $ref: '#/components/responses/Conflict'
        '422':
          $ref: '#/components/responses/UnprocessableEntity'

    get:
      summary: List templates
      operationId: listTemplates
      tags: [Templates]
      parameters:
        - name: type
          in: query
          schema:
            type: string
            enum: [slack, pagerduty, email, webhook, generic]
        - name: search
          in: query
          schema:
            type: string
        - name: limit
          in: query
          schema:
            type: integer
            default: 50
            maximum: 200
        - name: offset
          in: query
          schema:
            type: integer
            default: 0
      responses:
        '200':
          description: List of templates
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ListTemplatesResponse'

components:
  schemas:
    CreateTemplateRequest:
      type: object
      required: [name, type, content]
      properties:
        name:
          type: string
          minLength: 3
          maxLength: 64
          pattern: '^[a-z0-9_]+$'
        type:
          type: string
          enum: [slack, pagerduty, email, webhook, generic]
        content:
          type: string
          maxLength: 65536
        description:
          type: string
          maxLength: 500
        metadata:
          type: object
          additionalProperties: true
```

---

## 📈 Performance Optimization

### 1. Database Optimization

**Indexes Strategy**:
- B-tree indexes: name, type, timestamps
- GIN index: metadata JSONB, full-text search
- Partial indexes: active templates only (WHERE deleted_at IS NULL)

**Query Optimization**:
- Use `EXPLAIN ANALYZE` for slow queries
- Connection pooling (max 20 connections)
- Prepared statements for common queries

### 2. Caching Strategy

**Cache Invalidation**:
```
On CREATE: No invalidation (new entry)
On UPDATE: Invalidate template:{name}
On DELETE: Invalidate template:{name}
On ROLLBACK: Invalidate template:{name}
```

**Cache Warming** (optional):
- Pre-load top 100 most-used templates on startup
- Background refresh every 5 minutes

### 3. Rate Limiting

**Implementation**: Token bucket algorithm

```go
// Per-user rate limits
const (
    ReadLimit  = 100 // requests per minute
    WriteLimit = 20  // requests per minute
)

// Middleware
func RateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 🔒 Security Design

### 1. Authentication & Authorization

**RBAC Model**:
```
Role: admin
  - template:create
  - template:read
  - template:update
  - template:delete

Role: viewer
  - template:read
  - template:validate
```

**Implementation**:
```go
// Middleware
func RequireAdmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r.Context())
        if user.Role != "admin" {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 2. Input Validation

**Sanitization**:
```go
func sanitizeTemplateName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)

    // Remove invalid characters
    reg := regexp.MustCompile(`[^a-z0-9_]`)
    name = reg.ReplaceAllString(name, "")

    return name
}
```

**SQL Injection Prevention**:
- Use parameterized queries (pgx)
- Never concatenate user input into SQL

### 3. Template Injection Prevention

**Sandboxing** (handled by TN-153):
- Execution timeout (5s max)
- Function whitelist (only Alertmanager-compatible)
- No filesystem/network access

---

## 📊 Observability Design

### Prometheus Metrics

```go
var (
    templateAPIRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "template_api_requests_total",
            Help: "Total number of template API requests",
        },
        []string{"method", "endpoint", "status"},
    )

    templateAPIDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "template_api_duration_seconds",
            Help:    "Template API request duration",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"method", "endpoint"},
    )

    templateCacheHitRatio = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "template_cache_hit_ratio",
            Help: "Template cache hit ratio (L1 + L2)",
        },
    )

    templateValidationErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "template_validation_errors_total",
            Help: "Template validation errors by type",
        },
        []string{"type", "error_code"},
    )
)
```

### Structured Logging

```go
// Example log entry
logger.Info("template created",
    "template_id", template.ID,
    "template_name", template.Name,
    "template_type", template.Type,
    "user_id", userID,
    "duration_ms", time.Since(start).Milliseconds(),
    "request_id", requestID,
)
```

---

## 🧪 Testing Strategy

### Unit Tests (80%+ coverage target)

**Test Files**:
- `template_handler_test.go` - HTTP handler tests
- `template_manager_test.go` - Business logic tests
- `template_validator_test.go` - Validation tests
- `template_repository_test.go` - Database tests
- `template_cache_test.go` - Cache tests

**Test Categories**:
1. **Happy Path**: Valid inputs, successful operations
2. **Error Cases**: Invalid inputs, database errors, conflicts
3. **Edge Cases**: Empty strings, max lengths, special characters
4. **Concurrent Access**: Race condition tests

### Integration Tests

**Test Scenarios**:
1. Create template → Get template → Verify content
2. Update template → Check version increment → Get old version
3. Delete template → Verify soft delete → List excludes deleted
4. Cache behavior → Create → Get (cached) → Update → Get (invalidated)

### Benchmarks

**Performance Tests**:
```go
func BenchmarkGetTemplateCached(b *testing.B) {
    // Benchmark cached GET
}

func BenchmarkGetTemplateUncached(b *testing.B) {
    // Benchmark uncached GET (cold cache)
}

func BenchmarkCreateTemplate(b *testing.B) {
    // Benchmark template creation
}
```

---

## 🚀 Migration Strategy

### Phase 1: Schema Migration

```sql
-- Migration: 20251125_create_templates_tables.sql
-- Run with: goose up

BEGIN;

-- Create templates table
CREATE TABLE templates (...);

-- Create template_versions table
CREATE TABLE template_versions (...);

-- Seed default templates from TN-154
INSERT INTO templates (name, type, content, description, created_by)
SELECT ...
FROM default_templates;

COMMIT;
```

### Phase 2: Data Seeding

```go
// Seed default templates from TN-154
func SeedDefaultTemplates(ctx context.Context, repo TemplateRepository) error {
    registry := defaults.GetDefaultTemplates()

    // Slack templates
    for _, tmpl := range registry.Slack.GetAll() {
        err := repo.Create(ctx, tmpl)
        if err != nil {
            return err
        }
    }

    // Repeat for PagerDuty, Email, WebHook
    return nil
}
```

---

## 📝 Implementation Checklist

### Database Layer
- [ ] Create migration files
- [ ] Define Go models
- [ ] Implement TemplateRepository interface
- [ ] Write repository tests

### Business Layer
- [ ] Implement TemplateManager
- [ ] Implement TemplateValidator (TN-153 integration)
- [ ] Write business logic tests

### HTTP Layer
- [ ] Implement TemplateHandler
- [ ] Define request/response DTOs
- [ ] Add middleware (auth, metrics)
- [ ] Write handler tests

### Cache Layer
- [ ] Implement TwoTierTemplateCache
- [ ] Cache invalidation logic
- [ ] Write cache tests

### Integration
- [ ] Register routes in main.go
- [ ] Initialize dependencies
- [ ] Add Prometheus metrics
- [ ] Add structured logging

---

**Status**: ✅ Design COMPLETE
**Next**: Implementation Tasks Document
**Author**: AI Assistant
**Date**: 2025-11-25
