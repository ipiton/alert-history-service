# Config Validator - Go API Reference

Complete API reference for integrating the Alertmanager++ Config Validator into Go applications.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Packages](#core-packages)
  - [configvalidator](#package-configvalidator)
  - [types](#package-types)
  - [parser](#package-parser)
  - [validators](#package-validators)
  - [matcher](#package-matcher)
- [Type Reference](#type-reference)
- [Interface Reference](#interface-reference)
- [Examples](#examples)
- [Best Practices](#best-practices)

---

## Installation

```bash
go get github.com/vitaliisemenov/alert-history@latest
```

Import the validator in your Go code:

```go
import (
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func main() {
    // Create validator with default options
    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    // Validate a configuration file
    result, err := validator.ValidateFile("alertmanager.yml")
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    // Check results
    if result.Valid() {
        fmt.Println("✅ Configuration is valid!")
    } else {
        fmt.Printf("❌ Found %d errors and %d warnings\n",
            len(result.Errors), len(result.Warnings))
    }
}
```

---

## Core Packages

### Package: configvalidator

Main package providing the validation facade and high-level API.

#### Type: Validator

The main validator type that orchestrates configuration validation.

```go
type Validator struct {
    // Internal fields
}
```

##### Constructor: New

```go
func New(opts types.Options) *Validator
```

Creates a new Validator instance with the specified options.

**Parameters**:
- `opts types.Options` - Configuration options for the validator

**Returns**:
- `*Validator` - A new validator instance

**Example**:
```go
opts := types.Options{
    Mode: types.StrictMode,
    EnableBestPractices: true,
    EnableSecurityChecks: true,
}
validator := configvalidator.New(opts)
```

---

##### Method: ValidateFile

```go
func (v *Validator) ValidateFile(filename string) (*types.Result, error)
```

Validates an Alertmanager configuration file (YAML or JSON).

**Parameters**:
- `filename string` - Path to the configuration file

**Returns**:
- `*types.Result` - Validation result containing errors, warnings, etc.
- `error` - Error if file cannot be read or parsed

**Example**:
```go
result, err := validator.ValidateFile("alertmanager.yml")
if err != nil {
    return fmt.Errorf("validation error: %w", err)
}
```

**Errors**:
- Returns error if file doesn't exist
- Returns error if file cannot be read
- Returns error if file format is invalid

---

##### Method: ValidateConfig

```go
func (v *Validator) ValidateConfig(ctx context.Context, cfg *config.AlertmanagerConfig) *types.Result
```

Validates an in-memory Alertmanager configuration.

**Parameters**:
- `ctx context.Context` - Context for cancellation
- `cfg *config.AlertmanagerConfig` - Configuration to validate

**Returns**:
- `*types.Result` - Validation result

**Example**:
```go
cfg := &config.AlertmanagerConfig{
    Route: config.Route{
        Receiver: "default",
        GroupBy:  []string{"alertname"},
    },
    Receivers: []config.Receiver{
        {Name: "default"},
    },
}

result := validator.ValidateConfig(context.Background(), cfg)
```

---

##### Method: ValidateBytes

```go
func (v *Validator) ValidateBytes(data []byte) (*types.Result, error)
```

Validates a configuration from raw bytes (YAML or JSON).

**Parameters**:
- `data []byte` - Raw configuration data

**Returns**:
- `*types.Result` - Validation result
- `error` - Error if parsing fails

**Example**:
```go
data := []byte(`
route:
  receiver: default
receivers:
  - name: default
`)

result, err := validator.ValidateBytes(data)
```

---

### Package: types

Defines core types, options, and result structures.

#### Type: Options

Configuration options for the validator.

```go
type Options struct {
    // Mode specifies the validation strictness level
    Mode ValidationMode

    // EnableBestPractices enables best practice suggestions
    EnableBestPractices bool

    // EnableSecurityChecks enables security validation
    EnableSecurityChecks bool

    // EnableDeprecatedChecks warns about deprecated features
    EnableDeprecatedChecks bool

    // DefaultDocsURL is the base URL for documentation links
    DefaultDocsURL string

    // Logger for debug output
    Logger *slog.Logger
}
```

##### Function: DefaultOptions

```go
func DefaultOptions() Options
```

Returns default validation options.

**Returns**:
- `Options` - Default options with lenient mode

**Default Values**:
- `Mode`: `LenientMode`
- `EnableBestPractices`: `true`
- `EnableSecurityChecks`: `false`
- `EnableDeprecatedChecks`: `true`
- `DefaultDocsURL`: `"https://prometheus.io/docs/alerting/latest/configuration/"`

**Example**:
```go
opts := types.DefaultOptions()
opts.Mode = types.StrictMode
opts.EnableSecurityChecks = true
```

---

#### Type: ValidationMode

Enumeration of validation strictness levels.

```go
type ValidationMode int

const (
    StrictMode      ValidationMode = 0 // All warnings become errors
    LenientMode     ValidationMode = 1 // Balanced (default)
    PermissiveMode  ValidationMode = 2 // Minimal validation
)
```

**Modes**:

| Mode | Description | Use Case |
|------|-------------|----------|
| `StrictMode` | All warnings are treated as errors | Production deployments |
| `LenientMode` | Balanced validation (default) | General use, CI/CD |
| `PermissiveMode` | Minimal validation | Development, testing |

**Example**:
```go
opts := types.Options{
    Mode: types.StrictMode,  // Strictest validation
}
```

---

#### Type: Result

Validation result containing all issues found.

```go
type Result struct {
    // Errors contains all validation errors (blocking)
    Errors []Error

    // Warnings contains non-blocking warnings
    Warnings []Warning

    // Info contains informational messages
    Info []Info

    // Suggestions contains improvement suggestions
    Suggestions []Suggestion

    // Config is the parsed configuration (if successful)
    Config *config.AlertmanagerConfig
}
```

##### Method: Valid

```go
func (r *Result) Valid() bool
```

Returns true if the configuration has no errors.

**Returns**:
- `bool` - true if no errors, false otherwise

**Example**:
```go
if result.Valid() {
    fmt.Println("Configuration is valid")
} else {
    fmt.Printf("Found %d errors\n", len(result.Errors))
}
```

---

##### Method: ExitCode

```go
func (r *Result) ExitCode() int
```

Returns an appropriate exit code for CLI usage.

**Returns**:
- `int` - Exit code (0 = valid, 1 = warnings, 2 = errors)

**Exit Codes**:
- `0`: Configuration is valid (no errors or warnings)
- `1`: Configuration has warnings but no errors
- `2`: Configuration has errors

**Example**:
```go
result, _ := validator.ValidateFile("config.yml")
os.Exit(result.ExitCode())
```

---

##### Method: AddError

```go
func (r *Result) AddError(code, message string, location *Location,
    field, section, context, suggestion, docsURL string)
```

Adds an error to the validation result.

**Parameters**:
- `code string` - Error code (e.g., "E100")
- `message string` - Human-readable error message
- `location *Location` - Location in the configuration (can be nil)
- `field string` - Field path (e.g., "route.receiver")
- `section string` - Configuration section (e.g., "route")
- `context string` - Additional context
- `suggestion string` - How to fix the error
- `docsURL string` - Link to documentation

**Example**:
```go
result.AddError(
    "E100",
    "Receiver not found",
    &types.Location{Line: 5, Column: 3},
    "route.receiver",
    "route",
    "",
    "Add a receiver named 'default' to the receivers section",
    "https://example.com/docs#receivers",
)
```

---

##### Method: AddWarning

```go
func (r *Result) AddWarning(code, message string, location *Location,
    field, section, context, suggestion, docsURL string)
```

Adds a warning to the validation result.

**Parameters**: Same as `AddError`

---

##### Method: AddInfo

```go
func (r *Result) AddInfo(code, message string, location *Location,
    field, section, context, suggestion, docsURL string)
```

Adds an informational message to the validation result.

---

##### Method: AddSuggestion

```go
func (r *Result) AddSuggestion(code, message, before, after string,
    location *Location, field, section, context, suggestion, docsURL string)
```

Adds an improvement suggestion to the validation result.

**Additional Parameters**:
- `before string` - Current configuration snippet
- `after string` - Suggested configuration snippet

---

#### Type: Error

Represents a validation error.

```go
type Error struct {
    Type       string    // Always "error"
    Code       string    // Error code (e.g., "E100")
    Message    string    // Human-readable message
    Location   Location  // Location in config
    Field      string    // Field path
    Section    string    // Configuration section
    Context    string    // Additional context
    Suggestion string    // How to fix
    DocsURL    string    // Documentation link
}
```

---

#### Type: Warning

Represents a validation warning (same structure as Error, but Type = "warning").

```go
type Warning struct {
    Type       string
    Code       string
    Message    string
    Location   Location
    Field      string
    Section    string
    Context    string
    Suggestion string
    DocsURL    string
}
```

---

#### Type: Info

Represents an informational message.

```go
type Info struct {
    Type       string    // Always "info"
    Code       string    // Info code
    Message    string
    Location   Location
    Suggestion string
    DocsURL    string
}
```

---

#### Type: Suggestion

Represents an improvement suggestion.

```go
type Suggestion struct {
    Type       string    // Always "suggestion"
    Code       string    // Suggestion code
    Message    string
    Before     string    // Current config
    After      string    // Suggested config
    Location   Location
    Suggestion string
    DocsURL    string
}
```

---

#### Type: Location

Represents a location in the configuration file.

```go
type Location struct {
    Line    int    // Line number (1-based)
    Column  int    // Column number (1-based)
    File    string // Filename
    Field   string // Field path (e.g., "route.receiver")
    Section string // Section name (e.g., "route")
}
```

---

### Package: parser

Provides configuration parsing capabilities.

#### Type: Parser

Interface for configuration parsers.

```go
type Parser interface {
    Parse(data []byte) (*config.AlertmanagerConfig, []types.Error, error)
}
```

##### Method: Parse

Parses raw configuration data into a configuration struct.

**Parameters**:
- `data []byte` - Raw configuration data

**Returns**:
- `*config.AlertmanagerConfig` - Parsed configuration
- `[]types.Error` - Parsing errors (syntax errors, etc.)
- `error` - Fatal parsing error

---

#### Type: MultiFormatParser

Parser that auto-detects JSON or YAML format.

```go
type MultiFormatParser struct {
    // Internal fields
}
```

##### Constructor: NewMultiFormatParser

```go
func NewMultiFormatParser(opts types.Options, logger *slog.Logger) *MultiFormatParser
```

Creates a new multi-format parser.

**Example**:
```go
parser := parser.NewMultiFormatParser(opts, logger)
cfg, errors, err := parser.Parse(data)
```

---

#### Type: JSONParser

Specialized parser for JSON configurations.

```go
type JSONParser struct {
    // Internal fields
}
```

##### Constructor: NewJSONParser

```go
func NewJSONParser(opts types.Options, logger *slog.Logger) *JSONParser
```

---

#### Type: YAMLParser

Specialized parser for YAML configurations.

```go
type YAMLParser struct {
    // Internal fields
}
```

##### Constructor: NewYAMLParser

```go
func NewYAMLParser(opts types.Options, logger *slog.Logger) *YAMLParser
```

---

### Package: validators

Individual validators for different configuration aspects.

#### Type: RouteValidator

Validates route configuration.

```go
type RouteValidator struct {
    // Internal fields
}
```

##### Constructor: NewRouteValidator

```go
func NewRouteValidator(opts types.Options, logger *slog.Logger) *RouteValidator
```

##### Method: Validate

```go
func (rv *RouteValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
```

Validates route configuration, adding issues to the result.

---

#### Type: ReceiverValidator

Validates receiver configurations.

```go
type ReceiverValidator struct {
    // Internal fields
}
```

##### Constructor: NewReceiverValidator

```go
func NewReceiverValidator(opts types.Options, logger *slog.Logger) *ReceiverValidator
```

##### Method: Validate

```go
func (rv *ReceiverValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
```

---

#### Type: InhibitionValidator

Validates inhibition rules.

```go
type InhibitionValidator struct {
    // Internal fields
}
```

##### Constructor: NewInhibitionValidator

```go
func NewInhibitionValidator(opts types.Options, logger *slog.Logger) *InhibitionValidator
```

##### Method: Validate

```go
func (iv *InhibitionValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
```

---

#### Type: SecurityValidator

Performs security validation (secrets, TLS, etc.).

```go
type SecurityValidator struct {
    // Internal fields
}
```

##### Constructor: NewSecurityValidator

```go
func NewSecurityValidator(opts types.Options, logger *slog.Logger) *SecurityValidator
```

##### Method: Validate

```go
func (sv *SecurityValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
```

---

#### Type: StructuralValidator

Validates overall configuration structure.

```go
type StructuralValidator struct {
    // Internal fields
}
```

##### Constructor: NewStructuralValidator

```go
func NewStructuralValidator(opts types.Options, logger *slog.Logger) *StructuralValidator
```

##### Method: Validate

```go
func (sv *StructuralValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
```

---

### Package: matcher

Provides matcher parsing and validation.

#### Type: Matcher

Represents a parsed matcher.

```go
type Matcher struct {
    Name      string
    MatchType MatchType
    Value     string
}
```

##### Type: MatchType

```go
type MatchType int

const (
    MatchEqual        MatchType = 0  // label="value"
    MatchNotEqual     MatchType = 1  // label!="value"
    MatchRegex        MatchType = 2  // label=~"regex"
    MatchNotRegex     MatchType = 3  // label!~"regex"
)
```

##### Function: Parse

```go
func Parse(input string) (*Matcher, error)
```

Parses a matcher string into a Matcher struct.

**Parameters**:
- `input string` - Matcher string (e.g., `severity="critical"`)

**Returns**:
- `*Matcher` - Parsed matcher
- `error` - Parse error if invalid

**Example**:
```go
matcher, err := matcher.Parse(`severity="critical"`)
if err != nil {
    return fmt.Errorf("invalid matcher: %w", err)
}

fmt.Printf("Label: %s, Operator: %v, Value: %s\n",
    matcher.Name, matcher.MatchType, matcher.Value)
```

---

##### Method: Matches

```go
func (m *Matcher) Matches(labels map[string]string) bool
```

Tests if the matcher matches a label set.

**Parameters**:
- `labels map[string]string` - Label set to test

**Returns**:
- `bool` - true if labels match the matcher

**Example**:
```go
labels := map[string]string{
    "severity": "critical",
    "team":     "backend",
}

matcher, _ := matcher.Parse(`severity="critical"`)
if matcher.Matches(labels) {
    fmt.Println("Match!")
}
```

---

##### Method: String

```go
func (m *Matcher) String() string
```

Returns the matcher as a string.

---

## Type Reference

### Configuration Types

Configuration types are defined in `internal/alertmanager/config` package:

```go
// AlertmanagerConfig represents the full configuration
type AlertmanagerConfig struct {
    Global         GlobalConfig
    Route          Route
    Receivers      []Receiver
    InhibitRules   []InhibitRule
    MuteTimeIntervals []MuteTimeInterval
    Templates      []string
}

// GlobalConfig contains global settings
type GlobalConfig struct {
    ResolveTimeout   *Duration
    HTTPConfig       *HTTPConfig
    SMTPFrom         string
    SMTPSmarthost    string
    SMTPAuthUsername string
    SMTPAuthPassword string
    SMTPAuthSecret   string
    SMTPAuthIdentity string
    SMTPRequireTLS   *bool
    SlackAPIURL      *URL
    PagerdutyURL     *URL
    HipchatAPIURL    *URL
    HipchatAuthToken string
    OpsGenieAPIURL   *URL
    OpsGenieAPIKey   string
    WeChatAPIURL     *URL
    WeChatAPISecret  string
    WeChatAPICorpID  string
    VictorOpsAPIURL  *URL
    VictorOpsAPIKey  string
}

// Route defines an alert routing tree
type Route struct {
    Receiver       string
    GroupBy        []string
    GroupWait      *Duration
    GroupInterval  *Duration
    RepeatInterval *Duration
    Matchers       []string
    Continue       *bool
    Routes         []Route
}

// Receiver defines notification integrations
type Receiver struct {
    Name             string
    EmailConfigs     []EmailConfig
    PagerdutyConfigs []PagerdutyConfig
    SlackConfigs     []SlackConfig
    WebhookConfigs   []WebhookConfig
    OpsgenieConfigs  []OpsGenieConfig
    WechatConfigs    []WechatConfig
    // ... more integrations
}

// InhibitRule defines alert inhibition
type InhibitRule struct {
    SourceMatchers []string
    TargetMatchers []string
    Equal          []string
}
```

---

## Interface Reference

### Validator Interface

```go
type Validator interface {
    Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result)
}
```

All individual validators implement this interface.

### Parser Interface

```go
type Parser interface {
    Parse(data []byte) (*config.AlertmanagerConfig, []types.Error, error)
}
```

All parsers implement this interface.

---

## Examples

### Custom Validator

```go
package main

import (
    "context"

    "github.com/vitaliisemenov/alert-history/internal/alertmanager/config"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

type CustomValidator struct {
    opts types.Options
}

func (cv *CustomValidator) Validate(ctx context.Context, cfg *config.AlertmanagerConfig, result *types.Result) {
    // Custom validation logic
    if len(cfg.Receivers) > 100 {
        result.AddWarning(
            "W999",
            "Too many receivers defined",
            nil,
            "receivers",
            "receivers",
            "",
            "Consider consolidating receivers",
            "",
        )
    }
}
```

### Parsing Configuration

```go
package main

import (
    "os"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/parser"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func parseConfig(filename string) (*config.AlertmanagerConfig, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    opts := types.DefaultOptions()
    p := parser.NewMultiFormatParser(opts, nil)

    cfg, errors, err := p.Parse(data)
    if err != nil {
        return nil, err
    }

    if len(errors) > 0 {
        for _, e := range errors {
            fmt.Printf("Parse error: %s\n", e.Message)
        }
        return nil, fmt.Errorf("parsing failed")
    }

    return cfg, nil
}
```

### Testing Matchers

```go
package main

import (
    "testing"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/matcher"
)

func TestMatcher(t *testing.T) {
    m, err := matcher.Parse(`severity="critical"`)
    if err != nil {
        t.Fatalf("Parse error: %v", err)
    }

    labels := map[string]string{
        "severity": "critical",
    }

    if !m.Matches(labels) {
        t.Error("Expected matcher to match labels")
    }
}
```

---

## Best Practices

### 1. Reuse Validator Instances

```go
// ❌ Don't create validators repeatedly
func ValidateMultiple(files []string) {
    for _, file := range files {
        validator := configvalidator.New(types.DefaultOptions()) // Wasteful
        validator.ValidateFile(file)
    }
}

// ✅ Reuse validator instance
func ValidateMultiple(files []string) {
    validator := configvalidator.New(types.DefaultOptions())
    for _, file := range files {
        validator.ValidateFile(file)
    }
}
```

### 2. Handle All Result Types

```go
result, _ := validator.ValidateFile("config.yml")

// Check all issue types
if len(result.Errors) > 0 {
    // Handle blocking errors
}
if len(result.Warnings) > 0 {
    // Log warnings
}
if len(result.Suggestions) > 0 {
    // Consider improvements
}
```

### 3. Use Context for Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result := validator.ValidateConfig(ctx, cfg)
```

### 4. Configure Appropriate Validation Mode

```go
// Production: strict validation
prodOpts := types.Options{
    Mode: types.StrictMode,
    EnableSecurityChecks: true,
}

// Development: lenient validation
devOpts := types.Options{
    Mode: types.LenientMode,
}
```

### 5. Check Errors Before Using Config

```go
result, err := validator.ValidateFile("config.yml")
if err != nil {
    return err  // Fatal error (file not found, parse error)
}

if !result.Valid() {
    return fmt.Errorf("configuration has errors")  // Validation errors
}

// Safe to use result.Config
processConfig(result.Config)
```

---

## Thread Safety

- `Validator` instances are **thread-safe** and can be used concurrently
- `Result` objects are **not thread-safe** - each validation returns a new Result
- `Parser` instances are **thread-safe**
- Individual validators are **thread-safe**

**Example**:
```go
validator := configvalidator.New(opts)  // Thread-safe

// Safe to use from multiple goroutines
for _, file := range files {
    go func(f string) {
        result, _ := validator.ValidateFile(f)  // Each gets own Result
        // Process result
    }(file)
}
```

---

## Performance Considerations

- **Parsing**: O(n) where n is config size
- **Validation**: O(n) for most validators, O(n²) for some cross-references
- **Memory**: Approximately 2-5x the config file size

**Tips**:
- Reuse validator instances
- Use appropriate validation mode (permissive is faster)
- Disable unnecessary checks in development

---

## Versioning and Compatibility

- **API Stability**: The main `configvalidator` package API is stable
- **Internal Changes**: Internal validators may change between minor versions
- **Alertmanager Compatibility**: Supports Alertmanager v0.25+ configuration format

---

## Error Handling

```go
result, err := validator.ValidateFile("config.yml")

// Handle different error types
if err != nil {
    var pathErr *os.PathError
    if errors.As(err, &pathErr) {
        // File not found or permission denied
    }

    var syntaxErr *json.SyntaxError
    if errors.As(err, &syntaxErr) {
        // JSON syntax error
    }

    // Other fatal errors
    return err
}

// Check validation result
exitCode := result.ExitCode()
os.Exit(exitCode)
```

---

## Additional Resources

- [User Guide](USER_GUIDE.md) - Complete usage guide
- [Examples](EXAMPLES.md) - Real-world usage examples
- [Error Codes](ERROR_CODES.md) - Complete error reference
- [Alertmanager Documentation](https://prometheus.io/docs/alerting/latest/configuration/)

---

**Last Updated**: 2025-11-24
**API Version**: 1.0.0
**Package**: github.com/vitaliisemenov/alert-history/pkg/configvalidator
