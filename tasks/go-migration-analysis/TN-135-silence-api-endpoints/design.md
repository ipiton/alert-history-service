# TN-135: Silence API Endpoints - Technical Design

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-135
**Status**: 🟡 IN PROGRESS
**Created**: 2025-11-06
**Target Quality**: 150% (Enterprise-Grade)

---

## 📐 Architecture Overview

TN-135 implements REST API endpoints for silence management, acting as the HTTP interface layer on top of the Silence Manager Service (TN-134).

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP API Layer (TN-135)                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │        SilenceHandler (handlers/silence.go)          │   │
│  │  - POST /api/v2/silences                             │   │
│  │  - GET /api/v2/silences                              │   │
│  │  - GET /api/v2/silences/{id}                         │   │
│  │  - PUT /api/v2/silences/{id}                         │   │
│  │  - DELETE /api/v2/silences/{id}                      │   │
│  │  - POST /api/v2/silences/check                       │   │
│  │  - POST /api/v2/silences/bulk/delete                 │   │
│  └──────────────────────────────────────────────────────┘   │
│             ↓ delegates to                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │   SilenceManager (business/silencing/manager.go)     │   │
│  │  - CreateSilence()    - ListSilences()               │   │
│  │  - GetSilence()       - UpdateSilence()              │   │
│  │  - DeleteSilence()    - IsAlertSilenced()            │   │
│  └──────────────────────────────────────────────────────┘   │
│             ↓ uses                                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  SilenceRepository (infrastructure/silencing/)       │   │
│  │  - PostgreSQL storage with indexes                   │   │
│  │  - Filter builder for complex queries                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Component Interaction

```
 HTTP Request                    Handler                   Manager                Repository
     │                              │                         │                        │
     ├── POST /silences ─────────>  │                         │                        │
     │                              ├── Parse & Validate ──>  │                        │
     │                              │                         ├── CreateSilence() ──>  │
     │                              │                         │                        ├── INSERT
     │                              │                         │                        │   + Cache
     │                              │                         │ <── Silence Created ── │
     │                              │ <── Return Silence ──── │                        │
     │ <── 201 Created ──────────── │                         │                        │
     │                              │                         │                        │
     ├── GET /silences?status=active>                         │                        │
     │                              ├── Parse Query Params    │                        │
     │                              │                         ├── ListSilences() ───>  │
     │                              │                         │   (cache-first)        │
     │                              │                         │ <── [Silences] ─────── │
     │                              │ <── Format Response ─── │                        │
     │ <── 200 OK ───────────────── │                         │                        │
```

---

## 🏗️ Component Design

### 1. SilenceHandler

**File**: `go-app/cmd/server/handlers/silence.go`

**Responsibility**: HTTP request/response handling for silence endpoints

**Structure**:
```go
type SilenceHandler struct {
    manager  silencing.SilenceManager  // Business logic
    metrics  *metrics.APIMetrics       // Prometheus metrics
    logger   *slog.Logger              // Structured logging
    cache    cache.Cache               // Response caching (ETag)
}

// Constructor
func NewSilenceHandler(
    manager silencing.SilenceManager,
    metrics *metrics.APIMetrics,
    logger *slog.Logger,
    cache cache.Cache,
) *SilenceHandler
```

**Methods** (7 HTTP handlers):
```go
// Core CRUD
func (h *SilenceHandler) CreateSilence(w http.ResponseWriter, r *http.Request)
func (h *SilenceHandler) ListSilences(w http.ResponseWriter, r *http.Request)
func (h *SilenceHandler) GetSilence(w http.ResponseWriter, r *http.Request)
func (h *SilenceHandler) UpdateSilence(w http.ResponseWriter, r *http.Request)
func (h *SilenceHandler) DeleteSilence(w http.ResponseWriter, r *http.Request)

// Advanced (150% features)
func (h *SilenceHandler) CheckAlert(w http.ResponseWriter, r *http.Request)
func (h *SilenceHandler) BulkDelete(w http.ResponseWriter, r *http.Request)

// Helper methods
func (h *SilenceHandler) sendError(w http.ResponseWriter, message string, code int)
func (h *SilenceHandler) sendJSON(w http.ResponseWriter, data interface{}, code int)
func (h *SilenceHandler) parseQueryParams(r *http.Request) (*ListSilencesParams, error)
func (h *SilenceHandler) validateSilenceInput(silence *silencing.Silence) error
func (h *SilenceHandler) generateETag(data interface{}) string
func (h *SilenceHandler) checkETag(r *http.Request, etag string) bool
```

**Design Patterns**:
- **Dependency Injection**: Manager, metrics, logger injected via constructor
- **Single Responsibility**: Each method handles one HTTP endpoint
- **Error Handling**: Consistent error responses with proper HTTP codes
- **Logging**: All operations logged with request context
- **Metrics**: All operations recorded in Prometheus metrics

---

### 2. Request/Response Models

**File**: `go-app/cmd/server/handlers/silence_models.go`

#### Request Models

```go
// CreateSilenceRequest - POST /api/v2/silences
type CreateSilenceRequest struct {
    CreatedBy string                 `json:"createdBy" validate:"required,email,max=255"`
    Comment   string                 `json:"comment" validate:"required,min=3,max=1024"`
    StartsAt  time.Time              `json:"startsAt" validate:"required"`
    EndsAt    time.Time              `json:"endsAt" validate:"required,gtfield=StartsAt"`
    Matchers  []silencing.Matcher    `json:"matchers" validate:"required,min=1,max=100,dive"`
}

// UpdateSilenceRequest - PUT /api/v2/silences/{id}
type UpdateSilenceRequest struct {
    Comment  *string                `json:"comment,omitempty" validate:"omitempty,min=3,max=1024"`
    EndsAt   *time.Time             `json:"endsAt,omitempty"`
    Matchers *[]silencing.Matcher   `json:"matchers,omitempty" validate:"omitempty,min=1,max=100,dive"`
}

// CheckAlertRequest - POST /api/v2/silences/check
type CheckAlertRequest struct {
    Labels map[string]string       `json:"labels" validate:"required,min=1"`
}

// BulkDeleteRequest - POST /api/v2/silences/bulk/delete
type BulkDeleteRequest struct {
    IDs []string                    `json:"ids" validate:"required,min=1,max=100,dive,uuid"`
}

// ListSilencesParams - GET /api/v2/silences query parameters
type ListSilencesParams struct {
    // Filters
    Status       *string    // pending/active/expired
    CreatedBy    *string    // Email filter
    MatcherName  *string    // Filter by matcher name
    MatcherValue *string    // Filter by matcher value
    StartsAfter  *time.Time // Time range filter
    StartsBefore *time.Time // Time range filter
    EndsAfter    *time.Time
    EndsBefore   *time.Time

    // Pagination
    Limit  int    // Default: 100, Max: 1000
    Offset int    // Default: 0

    // Sorting
    Sort  string  // created_at, starts_at, ends_at, status
    Order string  // asc, desc (default: desc)
}
```

#### Response Models

```go
// SilenceResponse - Single silence response
type SilenceResponse struct {
    ID        string                 `json:"id"`
    CreatedBy string                 `json:"createdBy"`
    Comment   string                 `json:"comment"`
    StartsAt  time.Time              `json:"startsAt"`
    EndsAt    time.Time              `json:"endsAt"`
    Matchers  []silencing.Matcher    `json:"matchers"`
    Status    silencing.SilenceStatus `json:"status"`
    CreatedAt time.Time              `json:"createdAt"`
    UpdatedAt *time.Time             `json:"updatedAt,omitempty"`
}

// ListSilencesResponse - GET /api/v2/silences response
type ListSilencesResponse struct {
    Silences []*SilenceResponse     `json:"silences"`
    Total    int64                  `json:"total"`
    Limit    int                    `json:"limit"`
    Offset   int                    `json:"offset"`
}

// CheckAlertResponse - POST /api/v2/silences/check response
type CheckAlertResponse struct {
    Silenced   bool               `json:"silenced"`
    SilenceIDs []string           `json:"silenceIDs,omitempty"`
    Silences   []*SilenceResponse `json:"silences,omitempty"`
    LatencyMs  int64              `json:"latencyMs"`
}

// BulkDeleteResponse - POST /api/v2/silences/bulk/delete response
type BulkDeleteResponse struct {
    Deleted int                    `json:"deleted"`
    Errors  []BulkDeleteError      `json:"errors,omitempty"`
}

type BulkDeleteError struct {
    ID    string `json:"id"`
    Error string `json:"error"`
}

// ErrorResponse - Standard error response
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Details map[string]string      `json:"details,omitempty"`
    Code    string                 `json:"code,omitempty"`
}
```

**Conversion Helpers**:
```go
func toSilenceResponse(s *silencing.Silence) *SilenceResponse
func toSilenceResponses(silences []*silencing.Silence) []*SilenceResponse
func fromCreateRequest(req *CreateSilenceRequest) *silencing.Silence
func applyUpdateRequest(silence *silencing.Silence, req *UpdateSilenceRequest)
```

---

### 3. Handler Implementation Details

#### 3.1 CreateSilence Handler

**Endpoint**: `POST /api/v2/silences`

**Processing Steps**:
1. Parse JSON request body → `CreateSilenceRequest`
2. Validate request (go-playground/validator)
3. Convert to `silencing.Silence` domain model
4. Call `manager.CreateSilence(ctx, silence)`
5. Record metrics (duration, result)
6. Return `201 Created` with silence object

**Error Handling**:
```go
// 400 Bad Request
- Invalid JSON: "Invalid request body"
- Validation errors: "Validation failed: {field}: {error}"
- Time range invalid: "endsAt must be after startsAt"

// 409 Conflict
- Duplicate silence: "Silence with same matchers and time range already exists"

// 500 Internal Server Error
- Database error: "Failed to create silence"
- Unexpected error: "Internal server error"
```

**Example Implementation**:
```go
func (h *SilenceHandler) CreateSilence(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Parse request
    var req CreateSilenceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate
    if err := h.validateSilenceInput(&req); err != nil {
        h.sendError(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Convert to domain model
    silence := fromCreateRequest(&req)

    // Create via manager
    created, err := h.manager.CreateSilence(ctx, silence)
    if err != nil {
        if errors.Is(err, infrasilencing.ErrDuplicateSilence) {
            h.sendError(w, "Silence already exists", http.StatusConflict)
        } else {
            h.logger.Error("Failed to create silence", "error", err)
            h.sendError(w, "Failed to create silence", http.StatusInternalServerError)
        }
        return
    }

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("POST", "/silences").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("POST", "/silences", "201").Inc()

    // Log success
    h.logger.Info("Silence created", "id", created.ID, "creator", created.CreatedBy)

    // Return response
    h.sendJSON(w, toSilenceResponse(created), http.StatusCreated)
}
```

---

#### 3.2 ListSilences Handler

**Endpoint**: `GET /api/v2/silences`

**Processing Steps**:
1. Parse query parameters → `ListSilencesParams`
2. Validate parameters (status enum, pagination limits)
3. Check cache (ETag) for fast path (`status=active` only)
4. Build `infrasilencing.SilenceFilter` from params
5. Call `manager.ListSilences(ctx, filter)`
6. Generate ETag for response
7. Return `200 OK` with silences array

**Cache Strategy**:
```go
// Fast path (cache hit)
GET /silences?status=active
→ Check cache key: "silences:active"
→ If hit: Return cached response (304 Not Modified if ETag matches)
→ If miss: Query database → Update cache

// Slow path (complex filters)
GET /silences?createdBy=ops@example.com&startsAfter=2025-11-06T00:00:00Z
→ Always query database (no cache)
→ Generate ETag for response
```

**Pagination**:
```go
// Default: limit=100, offset=0
GET /silences → Returns first 100 silences

// Custom pagination
GET /silences?limit=50&offset=100 → Returns silences 101-150

// Max limit: 1000
GET /silences?limit=5000 → Clamped to 1000
```

**Sorting**:
```go
// Sort by created_at (descending, default)
GET /silences → ORDER BY created_at DESC

// Sort by starts_at (ascending)
GET /silences?sort=starts_at&order=asc → ORDER BY starts_at ASC

// Supported sort fields: created_at, starts_at, ends_at, status
```

**Example Implementation**:
```go
func (h *SilenceHandler) ListSilences(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Parse query parameters
    params, err := h.parseQueryParams(r)
    if err != nil {
        h.sendError(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Check cache for fast path (status=active only)
    if params.Status != nil && *params.Status == "active" && params.isSimpleQuery() {
        cacheKey := "silences:active"
        if cached, found := h.cache.Get(ctx, cacheKey); found {
            etag := h.generateETag(cached)
            if h.checkETag(r, etag) {
                w.WriteHeader(http.StatusNotModified)
                return
            }
            w.Header().Set("ETag", etag)
            h.sendJSON(w, cached, http.StatusOK)
            h.metrics.SilenceCacheHitsTotal.WithLabelValues("/silences").Inc()
            return
        }
    }

    // Build filter
    filter := params.toSilenceFilter()

    // Query database
    silences, err := h.manager.ListSilences(ctx, filter)
    if err != nil {
        h.logger.Error("Failed to list silences", "error", err)
        h.sendError(w, "Failed to list silences", http.StatusInternalServerError)
        return
    }

    // Build response
    response := &ListSilencesResponse{
        Silences: toSilenceResponses(silences),
        Total:    int64(len(silences)),
        Limit:    params.Limit,
        Offset:   params.Offset,
    }

    // Cache if simple query
    if params.Status != nil && *params.Status == "active" && params.isSimpleQuery() {
        cacheKey := "silences:active"
        h.cache.Set(ctx, cacheKey, response, 30*time.Second)
    }

    // Set ETag
    etag := h.generateETag(response)
    w.Header().Set("ETag", etag)

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("GET", "/silences").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("GET", "/silences", "200").Inc()

    h.sendJSON(w, response, http.StatusOK)
}
```

---

#### 3.3 GetSilence Handler

**Endpoint**: `GET /api/v2/silences/{id}`

**Processing Steps**:
1. Extract `id` from URL path
2. Validate UUID format
3. Call `manager.GetSilence(ctx, id)` (cache-first)
4. Return `200 OK` with silence object OR `404 Not Found`

**Example Implementation**:
```go
func (h *SilenceHandler) GetSilence(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Extract ID from path (e.g., /api/v2/silences/550e8400-...)
    id := extractIDFromPath(r.URL.Path, "/api/v2/silences/")

    // Validate UUID
    if !isValidUUID(id) {
        h.sendError(w, "Invalid silence ID format", http.StatusBadRequest)
        return
    }

    // Get from manager (cache-first)
    silence, err := h.manager.GetSilence(ctx, id)
    if err != nil {
        if errors.Is(err, infrasilencing.ErrSilenceNotFound) {
            h.sendError(w, "Silence not found", http.StatusNotFound)
        } else {
            h.logger.Error("Failed to get silence", "id", id, "error", err)
            h.sendError(w, "Failed to get silence", http.StatusInternalServerError)
        }
        return
    }

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("GET", "/silences/:id").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("GET", "/silences/:id", "200").Inc()

    h.sendJSON(w, toSilenceResponse(silence), http.StatusOK)
}
```

---

#### 3.4 UpdateSilence Handler

**Endpoint**: `PUT /api/v2/silences/{id}`

**Processing Steps**:
1. Extract `id` from URL path
2. Parse JSON request body → `UpdateSilenceRequest`
3. Validate request (optional fields)
4. Get existing silence via `manager.GetSilence(ctx, id)`
5. Apply updates to silence object
6. Call `manager.UpdateSilence(ctx, silence)`
7. Return `200 OK` with updated silence

**Partial Update Support**:
```json
// Update only comment
PUT /api/v2/silences/{id}
{
  "comment": "Extended maintenance"
}

// Update only endsAt
PUT /api/v2/silences/{id}
{
  "endsAt": "2025-11-06T16:00:00Z"
}

// Update multiple fields
PUT /api/v2/silences/{id}
{
  "comment": "Extended",
  "endsAt": "2025-11-06T16:00:00Z",
  "matchers": [...]
}
```

**Example Implementation**:
```go
func (h *SilenceHandler) UpdateSilence(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Extract ID
    id := extractIDFromPath(r.URL.Path, "/api/v2/silences/")
    if !isValidUUID(id) {
        h.sendError(w, "Invalid silence ID format", http.StatusBadRequest)
        return
    }

    // Parse request
    var req UpdateSilenceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get existing silence
    silence, err := h.manager.GetSilence(ctx, id)
    if err != nil {
        if errors.Is(err, infrasilencing.ErrSilenceNotFound) {
            h.sendError(w, "Silence not found", http.StatusNotFound)
        } else {
            h.sendError(w, "Failed to get silence", http.StatusInternalServerError)
        }
        return
    }

    // Apply updates
    applyUpdateRequest(silence, &req)

    // Validate updated silence
    if err := silence.Validate(); err != nil {
        h.sendError(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
        return
    }

    // Update via manager
    if err := h.manager.UpdateSilence(ctx, silence); err != nil {
        if errors.Is(err, infrasilencing.ErrSilenceNotFound) {
            h.sendError(w, "Silence not found", http.StatusNotFound)
        } else if errors.Is(err, infrasilencing.ErrSilenceConflict) {
            h.sendError(w, "Conflict: silence was modified by another request", http.StatusConflict)
        } else {
            h.logger.Error("Failed to update silence", "id", id, "error", err)
            h.sendError(w, "Failed to update silence", http.StatusInternalServerError)
        }
        return
    }

    // Get updated silence (with new updatedAt timestamp)
    updated, _ := h.manager.GetSilence(ctx, id)

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("PUT", "/silences/:id").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("PUT", "/silences/:id", "200").Inc()

    h.sendJSON(w, toSilenceResponse(updated), http.StatusOK)
}
```

---

#### 3.5 DeleteSilence Handler

**Endpoint**: `DELETE /api/v2/silences/{id}`

**Processing Steps**:
1. Extract `id` from URL path
2. Validate UUID format
3. Call `manager.DeleteSilence(ctx, id)`
4. Return `204 No Content` OR `404 Not Found`

**Example Implementation**:
```go
func (h *SilenceHandler) DeleteSilence(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Extract ID
    id := extractIDFromPath(r.URL.Path, "/api/v2/silences/")
    if !isValidUUID(id) {
        h.sendError(w, "Invalid silence ID format", http.StatusBadRequest)
        return
    }

    // Delete via manager
    if err := h.manager.DeleteSilence(ctx, id); err != nil {
        if errors.Is(err, infrasilencing.ErrSilenceNotFound) {
            h.sendError(w, "Silence not found", http.StatusNotFound)
        } else {
            h.logger.Error("Failed to delete silence", "id", id, "error", err)
            h.sendError(w, "Failed to delete silence", http.StatusInternalServerError)
        }
        return
    }

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("DELETE", "/silences/:id").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("DELETE", "/silences/:id", "204").Inc()

    h.logger.Info("Silence deleted", "id", id)

    // Return 204 No Content
    w.WriteHeader(http.StatusNoContent)
}
```

---

#### 3.6 CheckAlert Handler (150% Feature)

**Endpoint**: `POST /api/v2/silences/check`

**Processing Steps**:
1. Parse JSON request body → `CheckAlertRequest`
2. Validate labels map (not empty)
3. Convert to `silencing.Alert` domain model
4. Call `manager.IsAlertSilenced(ctx, alert)`
5. Return `200 OK` with silenced flag + matching silence IDs

**Use Case**: Check if alert would be silenced before firing

**Example Implementation**:
```go
func (h *SilenceHandler) CheckAlert(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Parse request
    var req CheckAlertRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate
    if len(req.Labels) == 0 {
        h.sendError(w, "Labels are required", http.StatusBadRequest)
        return
    }

    // Convert to Alert
    alert := &silencing.Alert{
        Labels: req.Labels,
    }

    // Check if silenced
    silenced, silenceIDs, err := h.manager.IsAlertSilenced(ctx, alert)
    if err != nil {
        h.logger.Error("Failed to check alert", "error", err)
        // Fail-safe: return not silenced on error
        silenced = false
        silenceIDs = nil
    }

    // Get full silence objects if silenced
    var silences []*SilenceResponse
    if silenced && len(silenceIDs) > 0 {
        for _, id := range silenceIDs {
            if silence, err := h.manager.GetSilence(ctx, id); err == nil {
                silences = append(silences, toSilenceResponse(silence))
            }
        }
    }

    // Build response
    duration := time.Since(start)
    response := &CheckAlertResponse{
        Silenced:   silenced,
        SilenceIDs: silenceIDs,
        Silences:   silences,
        LatencyMs:  duration.Milliseconds(),
    }

    // Record metrics
    h.metrics.SilenceRequestDuration.WithLabelValues("POST", "/silences/check").Observe(duration.Seconds())
    h.metrics.SilenceRequestsTotal.WithLabelValues("POST", "/silences/check", "200").Inc()

    h.sendJSON(w, response, http.StatusOK)
}
```

---

#### 3.7 BulkDelete Handler (150% Feature)

**Endpoint**: `POST /api/v2/silences/bulk/delete`

**Processing Steps**:
1. Parse JSON request body → `BulkDeleteRequest`
2. Validate IDs array (1-100 IDs, all valid UUIDs)
3. Iterate through IDs, call `manager.DeleteSilence(ctx, id)` for each
4. Collect successes + errors
5. Return `200 OK` (all deleted) OR `207 Multi-Status` (partial success)

**Response Examples**:
```json
// All deleted (200 OK)
{
  "deleted": 5,
  "errors": []
}

// Partial success (207 Multi-Status)
{
  "deleted": 3,
  "errors": [
    {"id": "...", "error": "silence not found"},
    {"id": "...", "error": "database error"}
  ]
}
```

**Example Implementation**:
```go
func (h *SilenceHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    start := time.Now()

    // Parse request
    var req BulkDeleteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate
    if len(req.IDs) == 0 || len(req.IDs) > 100 {
        h.sendError(w, "IDs must be between 1 and 100", http.StatusBadRequest)
        return
    }

    // Delete each silence
    var deleted int
    var errors []BulkDeleteError

    for _, id := range req.IDs {
        if !isValidUUID(id) {
            errors = append(errors, BulkDeleteError{
                ID:    id,
                Error: "invalid UUID format",
            })
            continue
        }

        if err := h.manager.DeleteSilence(ctx, id); err != nil {
            errorMsg := "failed to delete"
            if errors.Is(err, infrasilencing.ErrSilenceNotFound) {
                errorMsg = "silence not found"
            }
            errors = append(errors, BulkDeleteError{
                ID:    id,
                Error: errorMsg,
            })
        } else {
            deleted++
        }
    }

    // Build response
    response := &BulkDeleteResponse{
        Deleted: deleted,
        Errors:  errors,
    }

    // Record metrics
    duration := time.Since(start)
    h.metrics.SilenceRequestDuration.WithLabelValues("POST", "/silences/bulk/delete").Observe(duration.Seconds())

    // Return appropriate status code
    statusCode := http.StatusOK
    if len(errors) > 0 && deleted > 0 {
        statusCode = http.StatusMultiStatus // 207
    } else if len(errors) > 0 && deleted == 0 {
        statusCode = http.StatusBadRequest // 400 (all failed)
    }

    h.metrics.SilenceRequestsTotal.WithLabelValues("POST", "/silences/bulk/delete", fmt.Sprint(statusCode)).Inc()

    h.logger.Info("Bulk delete completed", "deleted", deleted, "errors", len(errors))

    h.sendJSON(w, response, statusCode)
}
```

---

### 4. Metrics Design

**File**: `go-app/pkg/metrics/api_metrics.go`

**New Metrics for TN-135**:
```go
type APIMetrics struct {
    // ... existing metrics ...

    // TN-135: Silence API Metrics
    SilenceRequestsTotal        *prometheus.CounterVec   // method, endpoint, status
    SilenceRequestDuration      *prometheus.HistogramVec // method, endpoint
    SilenceValidationErrors     *prometheus.CounterVec   // field
    SilenceOperationsTotal      *prometheus.CounterVec   // operation, result
    SilenceActiveSilences       prometheus.Gauge         // current count
    SilenceCacheHitsTotal       *prometheus.CounterVec   // endpoint
    SilenceResponseSizeBytes    *prometheus.HistogramVec // endpoint
    SilenceRateLimitExceeded    *prometheus.CounterVec   // endpoint
}
```

**Metric Definitions**:
```go
// Request metrics
SilenceRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "alert_history_api_silence_requests_total",
        Help: "Total number of silence API requests",
    },
    []string{"method", "endpoint", "status"},
)

SilenceRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "alert_history_api_silence_request_duration_seconds",
        Help: "Duration of silence API requests",
        Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
    },
    []string{"method", "endpoint"},
)

// Validation metrics
SilenceValidationErrors = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "alert_history_api_silence_validation_errors_total",
        Help: "Total number of silence validation errors",
    },
    []string{"field"},
)

// Operation metrics
SilenceOperationsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "alert_history_api_silence_operations_total",
        Help: "Total number of silence operations",
    },
    []string{"operation", "result"}, // operation: create/update/delete/check, result: success/error
)

// Active silences gauge
SilenceActiveSilences = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "alert_history_api_silence_active_silences",
        Help: "Current number of active silences",
    },
)

// Cache metrics
SilenceCacheHitsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "alert_history_api_silence_cache_hits_total",
        Help: "Total number of cache hits for silence API",
    },
    []string{"endpoint"},
)

// Response size metrics
SilenceResponseSizeBytes = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "alert_history_api_silence_response_size_bytes",
        Help: "Size of silence API responses in bytes",
        Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100, 200, 400, ..., 51200
    },
    []string{"endpoint"},
)

// Rate limiting metrics
SilenceRateLimitExceeded = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "alert_history_api_silence_rate_limit_exceeded_total",
        Help: "Total number of rate limit exceeded events",
    },
    []string{"endpoint"},
)
```

---

### 5. Integration with main.go

**File**: `go-app/cmd/server/main.go`

**Integration Steps**:
```go
// 1. Initialize Silence Manager (already done in TN-134)
silenceManager := silencing.NewDefaultSilenceManager(silenceRepo, silenceMatcher, logger, nil)
if err := silenceManager.Start(ctx); err != nil {
    slog.Error("Failed to start silence manager", "error", err)
    os.Exit(1)
}

// 2. Create Silence Handler
silenceHandler := handlers.NewSilenceHandler(
    silenceManager,
    apiMetrics,
    logger,
    cacheInstance,
)

// 3. Register endpoints
mux.HandleFunc("POST /api/v2/silences", silenceHandler.CreateSilence)
mux.HandleFunc("GET /api/v2/silences", silenceHandler.ListSilences)
mux.HandleFunc("GET /api/v2/silences/{id}", silenceHandler.GetSilence)
mux.HandleFunc("PUT /api/v2/silences/{id}", silenceHandler.UpdateSilence)
mux.HandleFunc("DELETE /api/v2/silences/{id}", silenceHandler.DeleteSilence)

// Advanced endpoints (150% features)
mux.HandleFunc("POST /api/v2/silences/check", silenceHandler.CheckAlert)
mux.HandleFunc("POST /api/v2/silences/bulk/delete", silenceHandler.BulkDelete)

slog.Info("✅ Silence API endpoints registered",
    "endpoints", []string{
        "POST /api/v2/silences - Create silence",
        "GET /api/v2/silences - List silences",
        "GET /api/v2/silences/{id} - Get silence",
        "PUT /api/v2/silences/{id} - Update silence",
        "DELETE /api/v2/silences/{id} - Delete silence",
        "POST /api/v2/silences/check - Check alert silenced",
        "POST /api/v2/silences/bulk/delete - Bulk delete silences",
    })
```

---

## 🧪 Testing Strategy

### Unit Tests

**File**: `go-app/cmd/server/handlers/silence_test.go`

**Test Coverage** (target: 95%+):
```go
// 1. CreateSilence tests (10 tests)
TestCreateSilence_Success
TestCreateSilence_InvalidJSON
TestCreateSilence_ValidationErrors
TestCreateSilence_DuplicateSilence
TestCreateSilence_DatabaseError
TestCreateSilence_MetricsRecorded

// 2. ListSilences tests (12 tests)
TestListSilences_Success_NoFilters
TestListSilences_Success_StatusFilter
TestListSilences_Success_CreatorFilter
TestListSilences_Success_TimeRangeFilter
TestListSilences_Success_Pagination
TestListSilences_Success_Sorting
TestListSilences_CacheHit
TestListSilences_EmptyResult
TestListSilences_InvalidParams

// 3. GetSilence tests (6 tests)
TestGetSilence_Success
TestGetSilence_InvalidUUID
TestGetSilence_NotFound
TestGetSilence_CacheHit
TestGetSilence_DatabaseError

// 4. UpdateSilence tests (8 tests)
TestUpdateSilence_Success_PartialUpdate
TestUpdateSilence_Success_FullUpdate
TestUpdateSilence_InvalidJSON
TestUpdateSilence_NotFound
TestUpdateSilence_ValidationErrors
TestUpdateSilence_ConflictError

// 5. DeleteSilence tests (5 tests)
TestDeleteSilence_Success
TestDeleteSilence_InvalidUUID
TestDeleteSilence_NotFound
TestDeleteSilence_DatabaseError

// 6. CheckAlert tests (6 tests)
TestCheckAlert_Silenced
TestCheckAlert_NotSilenced
TestCheckAlert_InvalidJSON
TestCheckAlert_EmptyLabels
TestCheckAlert_ManagerError_Failsafe

// 7. BulkDelete tests (7 tests)
TestBulkDelete_Success_AllDeleted
TestBulkDelete_PartialSuccess
TestBulkDelete_AllFailed
TestBulkDelete_InvalidIDs
TestBulkDelete_EmptyArray
TestBulkDelete_TooManyIDs

// Total: 54 unit tests
```

### Integration Tests

**File**: `go-app/cmd/server/handlers/silence_integration_test.go`

**Test Scenarios** (10 tests):
```go
TestSilenceAPI_EndToEnd_CreateListGetUpdateDelete
TestSilenceAPI_ConcurrentRequests
TestSilenceAPI_CacheInvalidation
TestSilenceAPI_Pagination_LargeDataset
TestSilenceAPI_CheckAlert_WithActiveSilences
TestSilenceAPI_BulkDelete_Performance
TestSilenceAPI_ErrorRecovery
TestSilenceAPI_MetricsRecorded
```

### Benchmark Tests

**File**: `go-app/cmd/server/handlers/silence_bench_test.go`

**Benchmarks** (8 benchmarks):
```go
BenchmarkCreateSilence
BenchmarkListSilences_CacheHit
BenchmarkListSilences_CacheMiss
BenchmarkGetSilence_CacheHit
BenchmarkGetSilence_CacheMiss
BenchmarkUpdateSilence
BenchmarkDeleteSilence
BenchmarkCheckAlert_100Silences
```

---

## 📊 Performance Optimization

### 1. Cache Strategy

**Cache Layers**:
```go
// L1: In-memory cache (SilenceManager)
- Active silences cached
- ~50ns lookup time
- Automatically invalidated on CRUD

// L2: Redis cache (ETag)
- GET /silences responses cached
- 30s TTL
- ETag-based cache validation
```

**Cache Keys**:
```
silences:active           → All active silences
silences:{id}             → Individual silence
silences:list:{hash}      → List response (complex queries)
```

### 2. Database Query Optimization

**Index Usage** (from TN-133):
```sql
-- Fast lookups by status
CREATE INDEX idx_silences_status ON silences(status);

-- Time range queries
CREATE INDEX idx_silences_time_range ON silences(starts_at, ends_at);

-- Creator filtering
CREATE INDEX idx_silences_creator ON silences(created_by);

-- Matcher filtering (GIN index)
CREATE INDEX idx_silences_matchers ON silences USING GIN (matchers jsonb_path_ops);
```

### 3. Zero-Allocation Optimization

**Hot Paths**:
```go
// Use sync.Pool for response buffers
var responseBufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

// Reuse JSON encoder
encoder := json.NewEncoder(w)

// Pre-allocate slices
silences := make([]*SilenceResponse, 0, 100)
```

---

## 🔒 Security Considerations

### 1. Input Validation

- **UUID validation**: Prevent SQL injection via ID parameter
- **JSON validation**: Reject malformed JSON early
- **Field validation**: Length limits, regex patterns
- **Time validation**: Prevent invalid time ranges

### 2. Rate Limiting (Basic)

```go
// Per-IP rate limiting: 100 requests/min
type RateLimiter struct {
    requests map[string]*rate.Limiter
    mu       sync.RWMutex
}

func (rl *RateLimiter) Allow(ip string) bool {
    limiter := rl.getLimiter(ip)
    return limiter.Allow()
}
```

### 3. Error Message Safety

```go
// ✅ Safe: Don't leak internal details
"Silence not found"

// ❌ Unsafe: Leaks database info
"SELECT failed: pq: relation 'silences' does not exist"
```

---

## 📝 Documentation Deliverables

### 1. README.md (800+ lines)

**Sections**:
- Overview
- API Endpoints (with examples)
- Request/Response schemas
- Error handling
- Performance characteristics
- Metrics & monitoring
- Integration guide
- Troubleshooting

### 2. OpenAPI 3.0 Spec

**File**: `docs/openapi-silences.yaml`

Complete OpenAPI specification for all 7 endpoints with:
- Request/response schemas
- Example payloads
- Error responses
- Authentication (placeholder)

### 3. Postman Collection (Optional)

**File**: `docs/TN-135-Silences-API.postman_collection.json`

Ready-to-use Postman collection with:
- All 7 endpoints
- Example requests
- Environment variables

---

## 🎯 Success Criteria

### Functional
- ✅ All 7 endpoints implemented and working
- ✅ 100% Alertmanager API v2 compatibility
- ✅ All validation rules enforced
- ✅ Error handling covers all edge cases

### Performance
- ✅ All latency targets met (p95, p99)
- ✅ Cache hit rate >90% for active silences
- ✅ Zero allocations in hot paths
- ✅ Handles 1000+ req/s

### Quality
- ✅ 95%+ test coverage
- ✅ 100% tests passing
- ✅ Zero linter errors
- ✅ Zero race conditions
- ✅ Zero memory leaks

### Documentation
- ✅ 1,000+ lines README
- ✅ Complete OpenAPI spec
- ✅ Integration examples
- ✅ Metrics guide

---

## 📦 Deliverables

### Code Files (7 files)
1. `go-app/cmd/server/handlers/silence.go` (600 LOC)
2. `go-app/cmd/server/handlers/silence_models.go` (400 LOC)
3. `go-app/cmd/server/handlers/silence_test.go` (1,500 LOC)
4. `go-app/cmd/server/handlers/silence_integration_test.go` (600 LOC)
5. `go-app/cmd/server/handlers/silence_bench_test.go` (400 LOC)
6. `go-app/pkg/metrics/api_metrics.go` (+150 LOC)
7. `go-app/cmd/server/main.go` (+50 LOC integration)

### Documentation (4 files)
1. `tasks/go-migration-analysis/TN-135/README.md` (1,000+ LOC)
2. `docs/openapi-silences.yaml` (600 LOC)
3. `tasks/go-migration-analysis/TN-135/COMPLETION_REPORT.md` (500 LOC)
4. `tasks/go-migration-analysis/TN-135/INTEGRATION_EXAMPLES.md` (400 LOC)

**Total LOC**: ~6,200 lines (production + tests + docs)

---

**Document Version**: 1.0
**Created**: 2025-11-06
**Author**: Kilo Code AI
**Status**: READY FOR IMPLEMENTATION
