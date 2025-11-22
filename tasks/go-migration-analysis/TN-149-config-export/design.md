# TN-149: GET /api/v2/config - Technical Design

**Date**: 2025-11-21
**Task ID**: TN-149
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Design Phase

---

## 🏗️ Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request                              │
│              GET /api/v2/config?format=json                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Router (gorilla/mux)                            │
│         /api/v2/config → ConfigHandler                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│            ConfigHandler (cmd/server/handlers/)              │
│  - Parse query parameters (format, sanitize, sections)      │
│  - Call ConfigService                                        │
│  - Serialize response (JSON/YAML)                           │
│  - Handle errors gracefully                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│         ConfigService (internal/config/service.go)           │
│  - Get current config (from main.go or context)            │
│  - Sanitize secrets (if enabled)                            │
│  - Filter sections (if requested)                           │
│  - Add metadata (version, source, loaded_at)                │
│  - Cache serialized config (optional, TTL 1s)               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│         ConfigSanitizer (internal/config/sanitizer.go)      │
│  - Redact passwords, API keys, secrets                       │
│  - Deep copy config before sanitization                      │
│  - Preserve structure while hiding values                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Response (JSON/YAML)                            │
│  {                                                           │
│    "version": "abc123...",                                   │
│    "source": "file",                                          │
│    "loaded_at": "2025-11-21T10:00:00Z",                      │
│    "config_file_path": "/etc/config.yaml",                   │
│    "config": { ... sanitized config ... }                    │
│  }                                                           │
└─────────────────────────────────────────────────────────────┘
```

---

## 📦 Component Design

### 1. ConfigHandler

**Location**: `go-app/cmd/server/handlers/config.go`

**Responsibilities**:
- HTTP request handling
- Query parameter parsing
- Response serialization (JSON/YAML)
- Error handling and status codes
- Content-Type headers

**Interface**:
```go
// HandleGetConfig handles GET /api/v2/config requests
func HandleGetConfig(
    configService ConfigService,
    logger *slog.Logger,
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Parse query parameters
        format := r.URL.Query().Get("format") // json, yaml
        sanitize := r.URL.Query().Get("sanitize") != "false"
        sections := parseSections(r.URL.Query().Get("sections"))

        // Get config from service
        configResp, err := configService.GetConfig(r.Context(), GetConfigOptions{
            Format:   format,
            Sanitize: sanitize,
            Sections: sections,
        })

        // Serialize and respond
        // ...
    }
}
```

**Query Parameters**:
- `format`: `json` (default) | `yaml`
- `sanitize`: `true` (default) | `false` (admin only)
- `sections`: comma-separated list (e.g., `server,database`)

**Response Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid query parameters
- `403 Forbidden`: Unauthorized access to unsanitized config
- `500 Internal Server Error`: Serialization/processing error

### 2. ConfigService

**Location**: `go-app/internal/config/service.go`

**Interface**:
```go
// ConfigService provides configuration export functionality
type ConfigService interface {
    // GetConfig exports current configuration
    GetConfig(ctx context.Context, opts GetConfigOptions) (*ConfigResponse, error)

    // GetConfigVersion returns version hash of current config
    GetConfigVersion() string

    // GetConfigSource returns source of configuration
    GetConfigSource() ConfigSource
}

// GetConfigOptions specifies export options
type GetConfigOptions struct {
    Format   string   // "json" or "yaml"
    Sanitize bool     // Whether to sanitize secrets
    Sections []string // Filter to specific sections (empty = all)
}

// ConfigResponse contains exported configuration
type ConfigResponse struct {
    Version         string                 `json:"version"`          // SHA256 hash
    Source          ConfigSource           `json:"source"`           // "file", "env", "defaults"
    LoadedAt        time.Time              `json:"loaded_at"`        // When config was loaded
    ConfigFilePath  string                 `json:"config_file_path,omitempty"` // Path if from file
    Config          map[string]interface{} `json:"config"`            // Actual config data
}

// ConfigSource represents configuration source
type ConfigSource string

const (
    ConfigSourceFile     ConfigSource = "file"
    ConfigSourceEnv      ConfigSource = "env"
    ConfigSourceDefaults ConfigSource = "defaults"
    ConfigSourceMixed    ConfigSource = "mixed" // file + env
)
```

**Implementation**:
```go
// DefaultConfigService implements ConfigService
type DefaultConfigService struct {
    config      *Config
    configPath  string
    loadedAt    time.Time
    source      ConfigSource
    sanitizer   ConfigSanitizer
    cache       *sync.RWMutex // Simple mutex-based cache
    cachedResp  *ConfigResponse
    cacheExpiry time.Time
}

func (s *DefaultConfigService) GetConfig(
    ctx context.Context,
    opts GetConfigOptions,
) (*ConfigResponse, error) {
    // Check cache (if format matches and not expired)
    if s.cachedResp != nil && time.Now().Before(s.cacheExpiry) {
        // Return cached (but may need to re-sanitize/filter)
        return s.cachedResp, nil
    }

    // Deep copy config
    configCopy := s.deepCopyConfig()

    // Sanitize if requested
    if opts.Sanitize {
        configCopy = s.sanitizer.Sanitize(configCopy)
    }

    // Filter sections if requested
    if len(opts.Sections) > 0 {
        configCopy = s.filterSections(configCopy, opts.Sections)
    }

    // Convert to map for JSON/YAML serialization
    configMap := s.configToMap(configCopy)

    // Build response
    resp := &ConfigResponse{
        Version:        s.GetConfigVersion(),
        Source:         s.source,
        LoadedAt:       s.loadedAt,
        ConfigFilePath: s.configPath,
        Config:         configMap,
    }

    // Cache response (TTL 1s)
    s.cachedResp = resp
    s.cacheExpiry = time.Now().Add(1 * time.Second)

    return resp, nil
}
```

### 3. ConfigSanitizer

**Location**: `go-app/internal/config/sanitizer.go`

**Responsibilities**:
- Deep copy configuration before sanitization
- Redact sensitive fields (passwords, API keys, secrets)
- Preserve structure and types
- Configurable redaction patterns

**Interface**:
```go
// ConfigSanitizer sanitizes sensitive configuration data
type ConfigSanitizer interface {
    // Sanitize removes or redacts sensitive fields
    Sanitize(cfg *Config) *Config
}

// DefaultConfigSanitizer implements ConfigSanitizer
type DefaultConfigSanitizer struct {
    redactionValue string // Default: "***REDACTED***"
}

func (s *DefaultConfigSanitizer) Sanitize(cfg *Config) *Config {
    // Deep copy
    sanitized := cfg.deepCopy()

    // Redact passwords
    sanitized.Database.Password = s.redactionValue
    sanitized.Redis.Password = s.redactionValue

    // Redact API keys
    sanitized.LLM.APIKey = s.redactionValue

    // Redact webhook secrets
    sanitized.Webhook.Authentication.APIKey = s.redactionValue
    sanitized.Webhook.Authentication.JWTSecret = s.redactionValue
    sanitized.Webhook.Signature.Secret = s.redactionValue

    return sanitized
}
```

**Redaction Patterns**:
- Passwords: `password`, `passwd`, `pwd`
- API Keys: `api_key`, `apikey`, `apiKey`
- Secrets: `secret`, `token`, `jwt_secret`
- URLs with credentials: `postgres://user:***@host/db`

### 4. Config Models

**Location**: `go-app/cmd/server/handlers/config_models.go`

**Response Models**:
```go
// ConfigExportResponse represents the response for GET /api/v2/config
type ConfigExportResponse struct {
    Status string          `json:"status"` // "success" or "error"
    Data   *ConfigData    `json:"data,omitempty"`
    Error  string         `json:"error,omitempty"`
}

// ConfigData contains configuration export data
type ConfigData struct {
    Version        string                 `json:"version"`
    Source         string                 `json:"source"`
    LoadedAt       string                 `json:"loaded_at"` // RFC3339
    ConfigFilePath string                 `json:"config_file_path,omitempty"`
    Config         map[string]interface{} `json:"config"`
}
```

---

## 🔄 Data Flow

### Request Flow

1. **HTTP Request** → Router matches `/api/v2/config`
2. **Handler** → Parse query parameters, validate
3. **Service** → Get config, sanitize, filter
4. **Sanitizer** → Deep copy and redact secrets
5. **Serializer** → Convert to JSON/YAML
6. **Response** → Write to HTTP response

### Caching Strategy

- **Cache Key**: Config version hash + format + sanitize flag
- **Cache TTL**: 1 second (config rarely changes at runtime)
- **Cache Invalidation**: On config reload (future: TN-152)
- **Cache Storage**: In-memory (sync.RWMutex + struct)

**Rationale**:
- Config changes rarely at runtime
- 1s TTL balances freshness vs performance
- In-memory cache avoids external dependencies

---

## 🔐 Security Design

### Authentication & Authorization

- **Public Access**: Sanitized config (default)
- **Admin Access**: Unsanitized config (`?sanitize=false`)
- **Rate Limiting**: 100 req/min per IP (standard)
- **Audit Logging**: All requests logged with request_id

### Secret Sanitization

**Fields Redacted**:
1. `database.password`
2. `redis.password`
3. `llm.api_key`
4. `webhook.authentication.api_key`
5. `webhook.authentication.jwt_secret`
6. `webhook.signature.secret`

**Redaction Value**: `***REDACTED***` (configurable)

**Deep Copy**: Always performed before sanitization to avoid mutating original config

---

## 📊 Observability

### Prometheus Metrics

**Namespace**: `alert_history_api_config`

1. **config_export_requests_total** (Counter)
   - Labels: `format` (json/yaml), `sanitized` (true/false), `status` (success/error)
   - Description: Total number of config export requests

2. **config_export_duration_seconds** (Histogram)
   - Labels: `format`, `sanitized`
   - Buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0]
   - Description: Time to export configuration

3. **config_export_errors_total** (Counter)
   - Labels: `error_type` (serialization/sanitization/filter)
   - Description: Total number of export errors

4. **config_export_size_bytes** (Histogram)
   - Labels: `format`
   - Buckets: [100, 500, 1000, 5000, 10000, 50000, 100000]
   - Description: Size of exported configuration in bytes

### Structured Logging

**Log Levels**:
- `DEBUG`: Query parameters, cache hits/misses
- `INFO`: Successful exports (with format, sanitized flag)
- `WARN`: Invalid query parameters, cache misses
- `ERROR`: Serialization errors, service errors

**Log Fields**:
- `request_id`: Request ID from middleware
- `format`: json/yaml
- `sanitized`: true/false
- `sections`: Requested sections (if any)
- `duration_ms`: Processing time
- `response_size_bytes`: Response size
- `error`: Error message (if any)

---

## 🧪 Testing Strategy

### Unit Tests

**Test Files**:
1. `config_handler_test.go` - Handler tests
2. `config_service_test.go` - Service tests
3. `config_sanitizer_test.go` - Sanitizer tests
4. `config_models_test.go` - Model validation tests

**Test Coverage**:
- ✅ Query parameter parsing (format, sanitize, sections)
- ✅ JSON serialization
- ✅ YAML serialization
- ✅ Secret sanitization (all fields)
- ✅ Section filtering
- ✅ Error handling (invalid format, serialization errors)
- ✅ Cache behavior
- ✅ Version generation
- ✅ Source detection

### Integration Tests

**Test Scenarios**:
1. **Full Export**: GET /api/v2/config (JSON, sanitized)
2. **YAML Export**: GET /api/v2/config?format=yaml
3. **Unsanitized**: GET /api/v2/config?sanitize=false (admin)
4. **Section Filter**: GET /api/v2/config?sections=server,database
5. **Invalid Format**: GET /api/v2/config?format=xml (400 error)
6. **Unauthorized Unsanitized**: Non-admin requesting unsanitized (403)

### Benchmarks

**Benchmark Scenarios**:
1. `BenchmarkConfigExportJSON` - JSON serialization
2. `BenchmarkConfigExportYAML` - YAML serialization
3. `BenchmarkConfigSanitize` - Sanitization overhead
4. `BenchmarkConfigFilterSections` - Section filtering
5. `BenchmarkConfigCacheHit` - Cache performance

**Performance Targets**:
- JSON serialization: < 1ms
- YAML serialization: < 2ms
- Sanitization: < 0.5ms
- Cache hit: < 0.1ms

---

## 🚀 Performance Optimizations

### 1. Response Caching
- **TTL**: 1 second
- **Storage**: In-memory (sync.RWMutex)
- **Key**: Config version + format + sanitize flag
- **Benefit**: Reduces serialization overhead for repeated requests

### 2. Lazy Serialization
- Serialize only when needed (not in service layer)
- Use streaming JSON encoder for large configs (future)

### 3. Section Filtering
- Filter before serialization (reduce payload size)
- Early return if no matching sections

### 4. Deep Copy Optimization
- Use reflection only when necessary
- Cache reflection metadata

### 5. Buffer Pooling
- Reuse buffers for JSON/YAML encoding (sync.Pool)

---

## 📝 Error Handling

### Error Types

```go
// ConfigExportError represents config export errors
type ConfigExportError struct {
    Type    ErrorType
    Message string
    Cause   error
}

type ErrorType string

const (
    ErrorTypeInvalidFormat    ErrorType = "invalid_format"
    ErrorTypeSerialization    ErrorType = "serialization"
    ErrorTypeSanitization     ErrorType = "sanitization"
    ErrorTypeUnauthorized     ErrorType = "unauthorized"
    ErrorTypeInvalidSections  ErrorType = "invalid_sections"
)
```

### Error Responses

**400 Bad Request**:
```json
{
  "status": "error",
  "error": "invalid format: xml (supported: json, yaml)"
}
```

**403 Forbidden**:
```json
{
  "status": "error",
  "error": "unauthorized: unsanitized config requires admin access"
}
```

**500 Internal Server Error**:
```json
{
  "status": "error",
  "error": "failed to serialize configuration: ..."
}
```

---

## 🔄 Integration Points

### 1. Router Integration

**Location**: `go-app/internal/api/router.go`

```go
func setupAPIv2Routes(router *mux.Router, config RouterConfig) {
    v2 := router.PathPrefix("/api/v2").Subrouter()

    // Config routes
    configRouter := v2.PathPrefix("/config").Subrouter()
    if config.ConfigService != nil {
        configHandler := handlers.NewConfigHandler(config.ConfigService, config.Logger)
        configRouter.HandleFunc("", configHandler.GetConfig).Methods("GET")
    } else {
        configRouter.HandleFunc("", PlaceholderHandler("GetConfig")).Methods("GET")
    }
}
```

### 2. Main.go Integration

**Location**: `go-app/cmd/server/main.go`

```go
// Initialize ConfigService
configService := config.NewConfigService(cfg, resolvedConfigPath, time.Now())

// Pass to router config
routerConfig := api.DefaultRouterConfig(appLogger)
routerConfig.ConfigService = configService
```

### 3. Metrics Integration

**Location**: `go-app/pkg/metrics/business.go`

```go
// ConfigExportMetrics holds metrics for config export
type ConfigExportMetrics struct {
    RequestsTotal    *prometheus.CounterVec
    DurationSeconds  *prometheus.HistogramVec
    ErrorsTotal      *prometheus.CounterVec
    SizeBytes        *prometheus.HistogramVec
}
```

---

## 📚 API Specification

### OpenAPI 3.0 Spec

```yaml
paths:
  /api/v2/config:
    get:
      summary: Export current configuration
      description: Returns the current application configuration in JSON or YAML format
      operationId: getConfig
      tags:
        - Configuration
      parameters:
        - name: format
          in: query
          schema:
            type: string
            enum: [json, yaml]
            default: json
        - name: sanitize
          in: query
          schema:
            type: boolean
            default: true
        - name: sections
          in: query
          schema:
            type: string
            description: Comma-separated list of sections (server,database,redis,llm,log,cache,lock,app,metrics,webhook)
      responses:
        '200':
          description: Configuration exported successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ConfigExportResponse'
            text/yaml:
              schema:
                type: string
        '400':
          description: Invalid query parameters
        '403':
          description: Unauthorized access to unsanitized config
        '500':
          description: Internal server error
```

---

## 🎯 Implementation Phases

### Phase 1: Core Implementation (4h)
- ConfigService interface and implementation
- ConfigSanitizer implementation
- Basic handler (JSON only)
- Unit tests (service, sanitizer)

### Phase 2: Format Support (2h)
- YAML serialization
- Format query parameter
- Integration tests (JSON + YAML)

### Phase 3: Advanced Features (3h)
- Section filtering
- Version tracking
- Source detection
- Cache implementation

### Phase 4: Observability (2h)
- Prometheus metrics
- Structured logging
- Error handling improvements

### Phase 5: Testing & Documentation (3h)
- Comprehensive unit tests
- Integration tests
- Benchmarks
- API documentation
- README

**Total Estimated Time**: 14 hours
**Target Quality**: 150% (21 hours effective)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-21
**Author**: AI Assistant
**Review Status**: Pending
