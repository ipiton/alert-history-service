# TN-150: POST /api/v2/config - Technical Design

**Date**: 2025-11-22
**Task ID**: TN-150
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Design Phase

---

## 🏗️ Architecture Overview

### High-Level Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          HTTP Request Layer                              │
│  POST /api/v2/config?dry_run=false&sections=server,database            │
│  Content-Type: application/json                                          │
│  Authorization: Bearer <admin_token>                                     │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
                                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      Middleware Stack                                    │
│  1. Authentication (JWT/API Key validation)                             │
│  2. Authorization (Admin-only check)                                    │
│  3. Rate Limiting (10 req/min per user, 100 global)                    │
│  4. Request Size Limit (10MB max)                                       │
│  5. Metrics (Prometheus)                                                │
│  6. Logging (Request ID tracking)                                       │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
                                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      ConfigUpdateHandler                                 │
│  Location: go-app/cmd/server/handlers/config_update.go                 │
│                                                                          │
│  Responsibilities:                                                       │
│  • Parse request body (JSON/YAML)                                       │
│  • Extract query parameters (dry_run, sections, format)                 │
│  • Call ConfigUpdateService                                             │
│  • Format response (success/error)                                      │
│  • Set HTTP status codes                                                │
│  • Record metrics                                                        │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
                                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                     ConfigUpdateService                                  │
│  Location: go-app/internal/config/update_service.go                    │
│                                                                          │
│  Core Update Pipeline (4 phases):                                       │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │ Phase 1: VALIDATION (50ms target)                          │        │
│  │  • Syntax validation (JSON/YAML parse)                     │        │
│  │  • Schema validation (unmarshal to Config struct)          │        │
│  │  • Type validation (validator tags)                        │        │
│  │  • Business rule validation (Validate() method)            │        │
│  │  • Cross-field validation                                  │        │
│  │  ✅ Exit here if validation fails or dry_run=true         │        │
│  └────────────────────────────────────────────────────────────┘        │
│                                 │                                        │
│                                 ▼                                        │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │ Phase 2: DIFF CALCULATION (20ms target)                    │        │
│  │  • Deep compare old vs new config                          │        │
│  │  • Generate structured diff (added/modified/deleted)       │        │
│  │  • Sanitize secrets in diff                                │        │
│  │  • Identify affected components                            │        │
│  └────────────────────────────────────────────────────────────┘        │
│                                 │                                        │
│                                 ▼                                        │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │ Phase 3: ATOMIC APPLY (100ms target)                       │        │
│  │  • Acquire distributed lock (Redis)                        │        │
│  │  • Backup old config to storage                            │        │
│  │  • Write new config to storage                             │        │
│  │  • Increment version counter                               │        │
│  │  • Calculate SHA256 hash                                   │        │
│  │  • Write audit log entry                                   │        │
│  │  • Release lock                                             │        │
│  │  ⚠️  Rollback on any error                                 │        │
│  └────────────────────────────────────────────────────────────┘        │
│                                 │                                        │
│                                 ▼                                        │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │ Phase 4: HOT RELOAD (300ms target)                         │        │
│  │  • Notify all registered components                        │        │
│  │  • Parallel reload with 30s timeout                        │        │
│  │  • Collect reload results                                  │        │
│  │  • Health check after reload                               │        │
│  │  ⚠️  Rollback if critical component fails                  │        │
│  └────────────────────────────────────────────────────────────┘        │
│                                                                          │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
                                 ├─────────────────────┬────────────────────┐
                                 ▼                     ▼                    ▼
                    ┌──────────────────────┐  ┌──────────────┐  ┌──────────────┐
                    │  ConfigValidator     │  │  ConfigStorage│  │  ConfigReloader│
                    │  (validation.go)     │  │  (storage.go) │  │  (reloader.go) │
                    └──────────────────────┘  └──────────────┘  └──────────────┘
                                                      │
                                                      ▼
                                         ┌──────────────────────────┐
                                         │  PostgreSQL / Filesystem │
                                         │  - config_versions table │
                                         │  - config_audit_log table│
                                         └──────────────────────────┘
```

---

## 📦 Component Design

### 1. ConfigUpdateHandler (HTTP Layer)

**File**: `go-app/cmd/server/handlers/config_update.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "strings"
    "time"

    appconfig "github.com/vitaliisemenov/alert-history/internal/config"
    "gopkg.in/yaml.v3"
)

// ConfigUpdateHandler handles configuration update requests
type ConfigUpdateHandler struct {
    updateService appconfig.ConfigUpdateService
    logger        *slog.Logger
    metrics       *ConfigUpdateMetrics
}

// NewConfigUpdateHandler creates a new ConfigUpdateHandler
func NewConfigUpdateHandler(
    updateService appconfig.ConfigUpdateService,
    logger *slog.Logger,
) *ConfigUpdateHandler {
    if logger == nil {
        logger = slog.Default()
    }

    return &ConfigUpdateHandler{
        updateService: updateService,
        logger:        logger,
        metrics:       NewConfigUpdateMetrics(),
    }
}

// HandleUpdateConfig handles POST /api/v2/config requests
//
// Query Parameters:
//   - format: "json" (default) or "yaml"
//   - dry_run: "true" or "false" (default)
//   - sections: comma-separated list of sections to update (empty = all)
//
// Request Body: New configuration in JSON or YAML format
//
// Response:
//   - 200 OK: Update successful (with diff)
//   - 400 Bad Request: Invalid request (syntax, content-type, size)
//   - 401 Unauthorized: Missing or invalid auth
//   - 403 Forbidden: Not admin
//   - 409 Conflict: Concurrent update detected
//   - 422 Unprocessable Entity: Validation failed
//   - 500 Internal Server Error: Server error
func (h *ConfigUpdateHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    ctx := r.Context()
    requestID := extractRequestID(r)

    h.logger.Info("config update request received",
        "method", r.Method,
        "path", r.URL.Path,
        "query", r.URL.RawQuery,
        "remote_addr", r.RemoteAddr,
        "request_id", requestID,
    )

    // Step 1: Validate HTTP method
    if r.Method != http.MethodPost {
        h.respondError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
        h.metrics.RecordError("method_not_allowed")
        return
    }

    // Step 2: Parse query parameters
    opts, err := h.parseUpdateOptions(r)
    if err != nil {
        h.logger.Warn("invalid query parameters", "error", err, "request_id", requestID)
        h.respondError(w, http.StatusBadRequest, err.Error(), nil)
        h.metrics.RecordError("invalid_query_params")
        return
    }

    // Step 3: Read and validate request body
    body, err := h.readRequestBody(r)
    if err != nil {
        h.logger.Warn("failed to read request body", "error", err, "request_id", requestID)
        h.respondError(w, http.StatusBadRequest, err.Error(), nil)
        h.metrics.RecordError("invalid_body")
        return
    }

    // Step 4: Parse body based on format
    configMap, err := h.parseConfigBody(body, opts.Format)
    if err != nil {
        h.logger.Warn("failed to parse config body", "error", err, "request_id", requestID)
        h.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s syntax: %v", opts.Format, err), nil)
        h.metrics.RecordError("syntax_error")
        return
    }

    // Step 5: Call update service
    result, err := h.updateService.UpdateConfig(ctx, configMap, opts)
    if err != nil {
        h.handleUpdateError(w, err, requestID, opts, startTime)
        return
    }

    // Step 6: Success response
    h.respondSuccess(w, result, opts)
    h.metrics.RecordRequest(opts.Format, opts.DryRun, len(opts.Sections), "success", time.Since(startTime))

    h.logger.Info("config update successful",
        "version", result.Version,
        "dry_run", opts.DryRun,
        "sections", opts.Sections,
        "duration_ms", time.Since(startTime).Milliseconds(),
        "request_id", requestID,
    )
}

// parseUpdateOptions parses query parameters into UpdateOptions
func (h *ConfigUpdateHandler) parseUpdateOptions(r *http.Request) (appconfig.UpdateOptions, error) {
    opts := appconfig.UpdateOptions{
        Format:   "json",
        DryRun:   false,
        Sections: nil,
    }

    query := r.URL.Query()

    // Parse format
    if format := query.Get("format"); format != "" {
        format = strings.ToLower(format)
        if format != "json" && format != "yaml" {
            return opts, fmt.Errorf("invalid format: %s (supported: json, yaml)", format)
        }
        opts.Format = format
    }

    // Parse dry_run
    if dryRun := query.Get("dry_run"); dryRun == "true" {
        opts.DryRun = true
    }

    // Parse sections
    if sections := query.Get("sections"); sections != "" {
        sectionList := strings.Split(sections, ",")
        opts.Sections = make([]string, 0, len(sectionList))
        for _, s := range sectionList {
            s = strings.TrimSpace(s)
            if s != "" {
                opts.Sections = append(opts.Sections, s)
            }
        }
    }

    return opts, nil
}

// readRequestBody reads and validates request body
func (h *ConfigUpdateHandler) readRequestBody(r *http.Request) ([]byte, error) {
    // Check content-type
    contentType := r.Header.Get("Content-Type")
    if contentType == "" {
        return nil, fmt.Errorf("Content-Type header is required")
    }

    // Check body size (max 10MB)
    maxSize := int64(10 * 1024 * 1024) // 10MB
    if r.ContentLength > maxSize {
        return nil, fmt.Errorf("request body too large: %d bytes (max: %d)", r.ContentLength, maxSize)
    }

    // Read body with size limit
    body, err := io.ReadAll(io.LimitReader(r.Body, maxSize))
    if err != nil {
        return nil, fmt.Errorf("failed to read body: %w", err)
    }
    defer r.Body.Close()

    if len(body) == 0 {
        return nil, fmt.Errorf("request body is empty")
    }

    return body, nil
}

// parseConfigBody parses body based on format
func (h *ConfigUpdateHandler) parseConfigBody(body []byte, format string) (map[string]interface{}, error) {
    var configMap map[string]interface{}

    switch strings.ToLower(format) {
    case "json", "":
        if err := json.Unmarshal(body, &configMap); err != nil {
            return nil, fmt.Errorf("JSON parse error: %w", err)
        }
    case "yaml":
        if err := yaml.Unmarshal(body, &configMap); err != nil {
            return nil, fmt.Errorf("YAML parse error: %w", err)
        }
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }

    return configMap, nil
}

// handleUpdateError handles update service errors
func (h *ConfigUpdateHandler) handleUpdateError(
    w http.ResponseWriter,
    err error,
    requestID string,
    opts appconfig.UpdateOptions,
    startTime time.Time,
) {
    h.logger.Error("config update failed",
        "error", err,
        "request_id", requestID,
        "dry_run", opts.DryRun,
    )

    // Determine HTTP status code based on error type
    statusCode := http.StatusInternalServerError
    errorType := "server_error"

    switch e := err.(type) {
    case *appconfig.ValidationError:
        statusCode = http.StatusUnprocessableEntity
        errorType = "validation_error"
        h.respondError(w, statusCode, "validation failed", e.Errors)
    case *appconfig.ConflictError:
        statusCode = http.StatusConflict
        errorType = "conflict"
        h.respondError(w, statusCode, e.Error(), nil)
    default:
        h.respondError(w, statusCode, "failed to update configuration", nil)
    }

    h.metrics.RecordRequest(opts.Format, opts.DryRun, len(opts.Sections), errorType, time.Since(startTime))
    h.metrics.RecordError(errorType)
}

// respondSuccess writes success response
func (h *ConfigUpdateHandler) respondSuccess(
    w http.ResponseWriter,
    result *appconfig.UpdateResult,
    opts appconfig.UpdateOptions,
) {
    response := UpdateConfigResponse{
        Status:  "success",
        Message: h.buildSuccessMessage(result, opts),
        Version: result.Version,
        Diff:    result.Diff,
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Config-Version", fmt.Sprintf("%d", result.Version))
    w.WriteHeader(http.StatusOK)

    if err := json.NewEncoder(w).Encode(response); err != nil {
        h.logger.Error("failed to encode response", "error", err)
    }
}

// respondError writes error response
func (h *ConfigUpdateHandler) respondError(
    w http.ResponseWriter,
    statusCode int,
    message string,
    validationErrors []appconfig.ValidationErrorDetail,
) {
    response := UpdateConfigResponse{
        Status:  "error",
        Message: message,
        Errors:  validationErrors,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    if err := json.NewEncoder(w).Encode(response); err != nil {
        h.logger.Error("failed to encode error response", "error", err)
    }
}

// buildSuccessMessage builds success message
func (h *ConfigUpdateHandler) buildSuccessMessage(result *appconfig.UpdateResult, opts appconfig.UpdateOptions) string {
    if opts.DryRun {
        return "Configuration validated successfully (dry-run mode)"
    }

    sectionsInfo := "all sections"
    if len(opts.Sections) > 0 {
        sectionsInfo = fmt.Sprintf("sections: %s", strings.Join(opts.Sections, ", "))
    }

    return fmt.Sprintf("Configuration updated successfully (%s, version: %d)", sectionsInfo, result.Version)
}

// extractRequestID extracts request ID from context or header
func extractRequestID(r *http.Request) string {
    if id := r.Header.Get("X-Request-ID"); id != "" {
        return id
    }
    return fmt.Sprintf("%d", time.Now().UnixNano())
}

// UpdateConfigResponse represents update response
type UpdateConfigResponse struct {
    Status  string                               `json:"status"`
    Message string                               `json:"message"`
    Version int64                                `json:"version,omitempty"`
    Diff    *appconfig.ConfigDiff                `json:"diff,omitempty"`
    Errors  []appconfig.ValidationErrorDetail    `json:"errors,omitempty"`
}
```

### 2. ConfigUpdateService (Business Logic)

**File**: `go-app/internal/config/update_service.go`

```go
package config

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/go-playground/validator/v10"
)

// ConfigUpdateService handles configuration updates
type ConfigUpdateService interface {
    // UpdateConfig updates configuration with validation and hot reload
    UpdateConfig(ctx context.Context, configMap map[string]interface{}, opts UpdateOptions) (*UpdateResult, error)

    // RollbackConfig rolls back to a previous version
    RollbackConfig(ctx context.Context, version int64) (*UpdateResult, error)

    // GetHistory returns configuration history
    GetHistory(ctx context.Context, limit int) ([]*ConfigVersion, error)
}

// UpdateOptions specifies update options
type UpdateOptions struct {
    Format   string   // "json" or "yaml"
    DryRun   bool     // If true, validate only without applying
    Sections []string // If not empty, update only specified sections
}

// UpdateResult represents update result
type UpdateResult struct {
    Version   int64       `json:"version"`
    Diff      *ConfigDiff `json:"diff"`
    Applied   bool        `json:"applied"` // false for dry-run
    RolledBack bool       `json:"rolled_back"`
}

// ConfigDiff represents configuration changes
type ConfigDiff struct {
    Added    map[string]interface{}   `json:"added,omitempty"`
    Modified map[string]DiffEntry     `json:"modified,omitempty"`
    Deleted  []string                 `json:"deleted,omitempty"`
    Affected []string                 `json:"affected_components,omitempty"`
}

// DiffEntry represents a single field change
type DiffEntry struct {
    OldValue interface{} `json:"old_value"`
    NewValue interface{} `json:"new_value"`
}

// ValidationError represents validation error
type ValidationError struct {
    Message string                   `json:"message"`
    Errors  []ValidationErrorDetail  `json:"errors"`
}

func (e *ValidationError) Error() string {
    return e.Message
}

// ValidationErrorDetail represents single validation error
type ValidationErrorDetail struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

// ConflictError represents concurrent update conflict
type ConflictError struct {
    Message string `json:"message"`
}

func (e *ConflictError) Error() string {
    return e.Message
}

// ConfigVersion represents a historical configuration version
type ConfigVersion struct {
    Version    int64                  `json:"version"`
    Config     map[string]interface{} `json:"config"`
    CreatedAt  time.Time              `json:"created_at"`
    CreatedBy  string                 `json:"created_by"`
    Source     string                 `json:"source"`
    Hash       string                 `json:"hash"`
}

// DefaultConfigUpdateService implements ConfigUpdateService
type DefaultConfigUpdateService struct {
    currentConfig *Config
    storage       ConfigStorage
    validator     *ConfigValidator
    reloader      *ConfigReloader
    lockManager   LockManager
    logger        *slog.Logger
}

// NewConfigUpdateService creates a new ConfigUpdateService
func NewConfigUpdateService(
    currentConfig *Config,
    storage ConfigStorage,
    validator *ConfigValidator,
    reloader *ConfigReloader,
    lockManager LockManager,
    logger *slog.Logger,
) ConfigUpdateService {
    if logger == nil {
        logger = slog.Default()
    }

    return &DefaultConfigUpdateService{
        currentConfig: currentConfig,
        storage:       storage,
        validator:     validator,
        reloader:      reloader,
        lockManager:   lockManager,
        logger:        logger,
    }
}

// UpdateConfig implements ConfigUpdateService.UpdateConfig
func (s *DefaultConfigUpdateService) UpdateConfig(
    ctx context.Context,
    configMap map[string]interface{},
    opts UpdateOptions,
) (*UpdateResult, error) {
    s.logger.Info("starting config update",
        "dry_run", opts.DryRun,
        "sections", opts.Sections,
    )

    // Phase 1: VALIDATION
    newConfig, validationErrs := s.validateConfig(ctx, configMap, opts)
    if len(validationErrs) > 0 {
        return nil, &ValidationError{
            Message: fmt.Sprintf("validation failed: %d error(s)", len(validationErrs)),
            Errors:  validationErrs,
        }
    }

    // Phase 2: DIFF CALCULATION
    diff := s.calculateDiff(s.currentConfig, newConfig, opts.Sections)

    // If dry-run, return here
    if opts.DryRun {
        s.logger.Info("dry-run mode: validation successful, no changes applied")
        return &UpdateResult{
            Version: 0, // No version change
            Diff:    diff,
            Applied: false,
        }, nil
    }

    // Phase 3: ATOMIC APPLY
    version, err := s.atomicApply(ctx, newConfig, diff, opts)
    if err != nil {
        s.logger.Error("failed to apply config", "error", err)
        return nil, err
    }

    // Phase 4: HOT RELOAD
    if err := s.hotReload(ctx, newConfig, diff); err != nil {
        s.logger.Error("hot reload failed, rolling back", "error", err)
        // Rollback
        if rollbackErr := s.rollback(ctx, version-1); rollbackErr != nil {
            s.logger.Error("rollback failed", "error", rollbackErr)
            return nil, fmt.Errorf("hot reload failed and rollback failed: %w", rollbackErr)
        }
        return &UpdateResult{
            Version:    version - 1,
            Diff:       nil,
            Applied:    false,
            RolledBack: true,
        }, fmt.Errorf("hot reload failed, rolled back: %w", err)
    }

    s.logger.Info("config update successful", "version", version)
    return &UpdateResult{
        Version: version,
        Diff:    diff,
        Applied: true,
    }, nil
}

// validateConfig validates new configuration
func (s *DefaultConfigUpdateService) validateConfig(
    ctx context.Context,
    configMap map[string]interface{},
    opts UpdateOptions,
) (*Config, []ValidationErrorDetail) {
    // Convert map to Config struct
    configJSON, err := json.Marshal(configMap)
    if err != nil {
        return nil, []ValidationErrorDetail{{
            Field:   "config",
            Message: fmt.Sprintf("failed to serialize config: %v", err),
            Code:    "serialization_error",
        }}
    }

    var newConfig Config
    if err := json.Unmarshal(configJSON, &newConfig); err != nil {
        return nil, []ValidationErrorDetail{{
            Field:   "config",
            Message: fmt.Sprintf("failed to unmarshal config: %v", err),
            Code:    "unmarshal_error",
        }}
    }

    // Validate using ConfigValidator
    return &newConfig, s.validator.Validate(&newConfig, opts.Sections)
}

// calculateDiff calculates diff between old and new config
func (s *DefaultConfigUpdateService) calculateDiff(
    oldConfig *Config,
    newConfig *Config,
    sections []string,
) *ConfigDiff {
    // Convert to maps for comparison
    oldMap := configToMap(oldConfig)
    newMap := configToMap(newConfig)

    // Filter by sections if specified
    if len(sections) > 0 {
        oldMap = filterSections(oldMap, sections)
        newMap = filterSections(newMap, sections)
    }

    // Calculate diff
    diff := &ConfigDiff{
        Added:    make(map[string]interface{}),
        Modified: make(map[string]DiffEntry),
        Deleted:  make([]string, 0),
        Affected: identifyAffectedComponents(oldMap, newMap),
    }

    // ... diff calculation logic ...

    return diff
}

// atomicApply applies configuration atomically
func (s *DefaultConfigUpdateService) atomicApply(
    ctx context.Context,
    newConfig *Config,
    diff *ConfigDiff,
    opts UpdateOptions,
) (int64, error) {
    // Acquire distributed lock
    lockKey := "config:update"
    lock, err := s.lockManager.Acquire(ctx, lockKey, 30*time.Second)
    if err != nil {
        return 0, &ConflictError{Message: "failed to acquire lock: concurrent update in progress"}
    }
    defer lock.Release(ctx)

    // Backup old config
    if err := s.storage.Backup(ctx, s.currentConfig); err != nil {
        return 0, fmt.Errorf("failed to backup old config: %w", err)
    }

    // Save new config and increment version
    version, err := s.storage.Save(ctx, newConfig)
    if err != nil {
        return 0, fmt.Errorf("failed to save new config: %w", err)
    }

    // Write audit log
    s.writeAuditLog(ctx, version, diff, opts)

    // Update current config in memory
    s.currentConfig = newConfig

    return version, nil
}

// hotReload triggers hot reload for all registered components
func (s *DefaultConfigUpdateService) hotReload(
    ctx context.Context,
    newConfig *Config,
    diff *ConfigDiff,
) error {
    s.logger.Info("triggering hot reload", "affected_components", diff.Affected)

    // Reload all affected components
    if err := s.reloader.ReloadAll(ctx, newConfig, diff.Affected); err != nil {
        return fmt.Errorf("hot reload failed: %w", err)
    }

    return nil
}

// rollback rolls back to previous version
func (s *DefaultConfigUpdateService) rollback(ctx context.Context, version int64) error {
    s.logger.Warn("rolling back config", "target_version", version)

    // Load old config from storage
    oldConfig, err := s.storage.Load(ctx, version)
    if err != nil {
        return fmt.Errorf("failed to load version %d: %w", version, err)
    }

    // Apply old config
    s.currentConfig = oldConfig

    // Reload components with old config
    if err := s.reloader.ReloadAll(ctx, oldConfig, nil); err != nil {
        return fmt.Errorf("failed to reload with old config: %w", err)
    }

    return nil
}

// writeAuditLog writes audit log entry
func (s *DefaultConfigUpdateService) writeAuditLog(
    ctx context.Context,
    version int64,
    diff *ConfigDiff,
    opts UpdateOptions,
) {
    // TODO: Implement audit logging to PostgreSQL
    s.logger.Info("audit log",
        "version", version,
        "added_fields", len(diff.Added),
        "modified_fields", len(diff.Modified),
        "deleted_fields", len(diff.Deleted),
        "dry_run", opts.DryRun,
    )
}
```

### 3. ConfigValidator

**File**: `go-app/internal/config/validator.go`

```go
package config

import (
    "fmt"
    "strings"

    "github.com/go-playground/validator/v10"
)

// ConfigValidator validates configuration
type ConfigValidator struct {
    v *validator.Validate
}

// NewConfigValidator creates a new ConfigValidator
func NewConfigValidator() *ConfigValidator {
    v := validator.New()

    // Register custom validators
    _ = v.RegisterValidation("port", validatePort)
    _ = v.RegisterValidation("positive", validatePositive)

    return &ConfigValidator{v: v}
}

// Validate validates configuration
func (cv *ConfigValidator) Validate(cfg *Config, sections []string) []ValidationErrorDetail {
    errors := make([]ValidationErrorDetail, 0)

    // Structural validation (validator tags)
    if err := cv.v.Struct(cfg); err != nil {
        if validationErrs, ok := err.(validator.ValidationErrors); ok {
            for _, e := range validationErrs {
                errors = append(errors, ValidationErrorDetail{
                    Field:   e.Field(),
                    Message: formatValidationError(e),
                    Code:    e.Tag(),
                })
            }
        }
    }

    // Business rule validation (custom logic)
    errors = append(errors, cv.validateBusinessRules(cfg, sections)...)

    return errors
}

// validateBusinessRules validates business rules
func (cv *ConfigValidator) validateBusinessRules(cfg *Config, sections []string) []ValidationErrorDetail {
    errors := make([]ValidationErrorDetail, 0)

    // Example: MaxConnections >= MinConnections
    if cfg.Database.MaxConnections < cfg.Database.MinConnections {
        errors = append(errors, ValidationErrorDetail{
            Field:   "database.max_connections",
            Message: fmt.Sprintf("max_connections (%d) must be >= min_connections (%d)",
                cfg.Database.MaxConnections, cfg.Database.MinConnections),
            Code: "invalid_range",
        })
    }

    // Example: If LLM enabled, API key must be set
    if cfg.LLM.Enabled && cfg.LLM.APIKey == "" {
        errors = append(errors, ValidationErrorDetail{
            Field:   "llm.api_key",
            Message: "api_key is required when llm.enabled=true",
            Code:    "required_conditional",
        })
    }

    // ... more business rules ...

    return errors
}

// validatePort validates port number (1-65535)
func validatePort(fl validator.FieldLevel) bool {
    port := fl.Field().Int()
    return port > 0 && port <= 65535
}

// validatePositive validates positive integer
func validatePositive(fl validator.FieldLevel) bool {
    return fl.Field().Int() > 0
}

// formatValidationError formats validator error
func formatValidationError(e validator.FieldError) string {
    switch e.Tag() {
    case "required":
        return "field is required"
    case "port":
        return "must be a valid port number (1-65535)"
    case "positive":
        return "must be a positive number"
    default:
        return fmt.Sprintf("validation failed: %s", e.Tag())
    }
}
```

### 4. ConfigStorage Interface

**File**: `go-app/internal/config/storage.go`

```go
package config

import (
    "context"
    "time"
)

// ConfigStorage handles configuration persistence
type ConfigStorage interface {
    // Save saves configuration and returns new version number
    Save(ctx context.Context, cfg *Config) (version int64, err error)

    // Load loads configuration by version
    Load(ctx context.Context, version int64) (*Config, error)

    // GetLatestVersion returns latest version number
    GetLatestVersion(ctx context.Context) (int64, error)

    // Backup backs up current configuration
    Backup(ctx context.Context, cfg *Config) error

    // GetHistory returns configuration history
    GetHistory(ctx context.Context, limit int) ([]*ConfigVersion, error)
}

// PostgreSQLConfigStorage implements ConfigStorage using PostgreSQL
type PostgreSQLConfigStorage struct {
    db *pgxpool.Pool
}

// FileConfigStorage implements ConfigStorage using filesystem
type FileConfigStorage struct {
    basePath string
}
```

### 5. ConfigReloader

**File**: `go-app/internal/config/reloader.go`

```go
package config

import (
    "context"
    "fmt"
    "log/slog"
    "sync"
    "time"
)

// Reloadable is implemented by components that support hot reload
type Reloadable interface {
    // Reload reloads component with new configuration
    Reload(ctx context.Context, cfg *Config) error

    // Name returns component name
    Name() string

    // IsCritical returns true if component is critical (rollback on failure)
    IsCritical() bool
}

// ConfigReloader orchestrates hot reload for registered components
type ConfigReloader struct {
    components []Reloadable
    logger     *slog.Logger
    mu         sync.RWMutex
}

// NewConfigReloader creates a new ConfigReloader
func NewConfigReloader(logger *slog.Logger) *ConfigReloader {
    return &ConfigReloader{
        components: make([]Reloadable, 0),
        logger:     logger,
    }
}

// Register registers a component for hot reload
func (r *ConfigReloader) Register(component Reloadable) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.components = append(r.components, component)
    r.logger.Info("component registered for hot reload",
        "component", component.Name(),
        "critical", component.IsCritical(),
    )
}

// ReloadAll reloads all registered components
func (r *ConfigReloader) ReloadAll(
    ctx context.Context,
    cfg *Config,
    affectedComponents []string,
) error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    r.logger.Info("reloading components",
        "total", len(r.components),
        "affected", affectedComponents,
    )

    // Reload components in parallel
    type result struct {
        component string
        critical  bool
        err       error
    }

    results := make(chan result, len(r.components))
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var wg sync.WaitGroup
    for _, comp := range r.components {
        // Skip if not affected (optimization)
        if len(affectedComponents) > 0 && !contains(affectedComponents, comp.Name()) {
            continue
        }

        wg.Add(1)
        go func(c Reloadable) {
            defer wg.Done()
            err := c.Reload(ctx, cfg)
            results <- result{
                component: c.Name(),
                critical:  c.IsCritical(),
                err:       err,
            }
        }(comp)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    criticalError := false
    for res := range results {
        if res.err != nil {
            r.logger.Error("component reload failed",
                "component", res.component,
                "critical", res.critical,
                "error", res.err,
            )
            if res.critical {
                criticalError = true
            }
        } else {
            r.logger.Info("component reloaded successfully", "component", res.component)
        }
    }

    if criticalError {
        return fmt.Errorf("critical component reload failed")
    }

    return nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

---

## 📊 Data Models

### Database Schema

```sql
-- Configuration versions table
CREATE TABLE config_versions (
    version BIGSERIAL PRIMARY KEY,
    config JSONB NOT NULL,
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    source VARCHAR(50) NOT NULL, -- 'api', 'gitops', 'manual', 'sighup'
    description TEXT
);

CREATE INDEX idx_config_versions_created_at ON config_versions(created_at DESC);
CREATE INDEX idx_config_versions_hash ON config_versions(hash);

-- Configuration audit log table
CREATE TABLE config_audit_log (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL REFERENCES config_versions(version),
    action VARCHAR(50) NOT NULL, -- 'create', 'update', 'rollback'
    user_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    diff JSONB,
    sections TEXT[], -- Updated sections
    dry_run BOOLEAN DEFAULT FALSE,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_audit_log_version ON config_audit_log(version);
CREATE INDEX idx_config_audit_log_user_id ON config_audit_log(user_id);
CREATE INDEX idx_config_audit_log_created_at ON config_audit_log(created_at DESC);
```

---

## 🔄 Sequence Diagrams

### Successful Update Flow

```
User -> API: POST /api/v2/config {new_config}
API -> Handler: HandleUpdateConfig()
Handler -> Service: UpdateConfig(config, opts)

Service -> Validator: Validate(config)
Validator -> Service: ✅ Valid

Service -> DiffCalculator: CalculateDiff(old, new)
DiffCalculator -> Service: Diff

Service -> LockManager: Acquire("config:update")
LockManager -> Service: ✅ Lock acquired

Service -> Storage: Backup(old_config)
Storage -> Service: ✅ Backed up

Service -> Storage: Save(new_config)
Storage -> Service: ✅ Saved, version=42

Service -> Reloader: ReloadAll(new_config)
Reloader -> Components: Reload(new_config) [parallel]
Components -> Reloader: ✅ All reloaded

Reloader -> Service: ✅ Success
Service -> LockManager: Release()
Service -> Handler: UpdateResult{version=42, diff, applied=true}
Handler -> User: 200 OK {version: 42, diff}
```

### Validation Error Flow

```
User -> API: POST /api/v2/config {invalid_config}
API -> Handler: HandleUpdateConfig()
Handler -> Service: UpdateConfig(config, opts)

Service -> Validator: Validate(config)
Validator -> Service: ❌ [port out of range, missing api_key]

Service -> Handler: ValidationError{errors: [...]}
Handler -> User: 422 Unprocessable Entity {errors: [...]}
```

### Rollback Flow

```
User -> API: POST /api/v2/config {new_config}
API -> Handler: HandleUpdateConfig()
Handler -> Service: UpdateConfig(config, opts)

Service -> Validator: Validate(config)
Validator -> Service: ✅ Valid

Service -> Storage: Save(new_config)
Storage -> Service: ✅ version=43

Service -> Reloader: ReloadAll(new_config)
Reloader -> DatabaseComponent: Reload()
DatabaseComponent -> Reloader: ❌ Connection failed

Reloader -> Service: ❌ Critical component failed

Service -> Storage: Load(version=42) [previous version]
Storage -> Service: ✅ old_config

Service -> Reloader: ReloadAll(old_config)
Reloader -> Service: ✅ Rolled back

Service -> Handler: Error + RolledBack=true
Handler -> User: 500 Internal Server Error {rolled_back: true}
```

---

## 🔐 Security Considerations

### 1. Authentication & Authorization
- **Admin-only access**: Only users with admin role can update config
- **JWT validation**: Validate JWT token on every request
- **API key support**: Alternative auth via X-API-Key header

### 2. Rate Limiting
- **Per-user limit**: 10 req/min
- **Global limit**: 100 req/min
- **Burst allowed**: 5 requests

### 3. Input Validation
- **Size limit**: Max 10MB payload
- **Content-Type validation**: Must be application/json or text/yaml
- **Schema validation**: Strict type checking
- **Sanitization**: Remove unknown fields (optional)

### 4. Audit Logging
- **What**: All update attempts (success + failure)
- **Who**: User ID, IP address, User-Agent
- **When**: Timestamp with millisecond precision
- **What changed**: Full diff с sanitized secrets
- **Retention**: 90 days minimum

### 5. Secret Management
- **Sanitization**: Secrets never logged in plain text
- **Diff sanitization**: Secrets replaced with `***REDACTED***` in diffs
- **Storage encryption**: Config encrypted at rest (optional)

---

## 📈 Performance Optimization

### 1. Validation Caching
- Cache validation results for identical configs (TTL: 1min)
- Use SHA256 hash as cache key

### 2. Parallel Reload
- Reload independent components in parallel
- Use goroutines with timeout and error collection

### 3. Diff Optimization
- Only calculate diff for changed sections
- Use efficient deep comparison algorithm

### 4. Connection Pooling
- Reuse database connections
- Pool size: min 5, max 20

---

## 🧪 Testing Strategy

### 1. Unit Tests (≥20 tests, 90% coverage)
- Handler request parsing
- Validator logic (each validation rule)
- Diff calculation
- Storage operations
- Reloader orchestration

### 2. Integration Tests (≥15 tests)
- End-to-end update flow
- Validation errors
- Rollback scenarios
- Concurrent updates
- Dry-run mode

### 3. Benchmarks (≥5 benchmarks)
- Validation performance
- Diff calculation performance
- Storage save/load performance
- Full update pipeline
- Concurrent updates

---

## 📝 Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] ConfigUpdateHandler implementation
- [ ] ConfigUpdateService implementation
- [ ] ConfigValidator implementation
- [ ] Basic validation rules
- [ ] Request/response models

### Phase 2: Storage & Persistence
- [ ] ConfigStorage interface
- [ ] PostgreSQL storage implementation
- [ ] File storage implementation (fallback)
- [ ] Version management
- [ ] Audit logging

### Phase 3: Hot Reload
- [ ] Reloadable interface
- [ ] ConfigReloader implementation
- [ ] Component registration
- [ ] Parallel reload with timeout
- [ ] Rollback mechanism

### Phase 4: Advanced Features
- [ ] Dry-run mode
- [ ] Partial updates (sections)
- [ ] Diff calculation
- [ ] Concurrent update protection (distributed lock)
- [ ] Rate limiting

### Phase 5: Testing & Documentation
- [ ] Unit tests (≥20, 90% coverage)
- [ ] Integration tests (≥15)
- [ ] Benchmarks (≥5)
- [ ] OpenAPI spec
- [ ] API guide documentation
- [ ] Security documentation

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Total Lines**: 1,247 LOC
