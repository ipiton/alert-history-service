# TN-151: Config Validator - Technical Design

**Date**: 2025-11-22
**Task ID**: TN-151
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Design Phase

---

## 🏗️ Architecture Overview

### High-Level Component Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         CLI Entry Point                                  │
│           cmd/alertmanager-config-validator/main.go                      │
│                                                                          │
│  Commands:                                                               │
│  - validate <file>     Validate configuration file                      │
│  - version             Show version info                                │
│  - help                Show help                                        │
│                                                                          │
│  Flags:                                                                  │
│  --mode=strict|lenient|permissive                                       │
│  --format=json|yaml|human                                               │
│  --sections=route,receivers,inhibition                                  │
│  --output=file.json                                                     │
│  --color/--no-color                                                     │
└────────────────────────────┬─────────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                    Validator Core Library                                │
│             pkg/configvalidator/validator.go                             │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                      Validator Facade                              ││
│  │  • New(options Options) *Validator                                 ││
│  │  • ValidateFile(path string) (*Result, error)                      ││
│  │  • ValidateBytes(data []byte) (*Result, error)                     ││
│  │  • ValidateConfig(cfg *Config) (*Result, error)                    ││
│  └────────────────────────────────────────────────────────────────────┘│
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                         Parser Layer                               ││
│  │  pkg/configvalidator/parser/                                       ││
│  │                                                                    ││
│  │  • YAMLParser: Parse YAML → Config                                ││
│  │  • JSONParser: Parse JSON → Config                                ││
│  │  • SchemaValidator: Validate against schema                       ││
│  │  • SyntaxChecker: Check syntax errors                             ││
│  └────────────────────────────────────────────────────────────────────┘│
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                    Validation Pipeline                             ││
│  │  pkg/configvalidator/validators/                                   ││
│  │                                                                    ││
│  │  1. StructuralValidator                                           ││
│  │     - Type validation (validator tags)                            ││
│  │     - Required fields                                             ││
│  │     - Format validation (URLs, emails, durations)                 ││
│  │     - Range validation (min/max)                                  ││
│  │                                                                    ││
│  │  2. RouteValidator                                                ││
│  │     - Route tree structure                                        ││
│  │     - Receiver references                                         ││
│  │     - Label matchers syntax                                       ││
│  │     - Interval validations                                        ││
│  │     - Dead route detection                                        ││
│  │                                                                    ││
│  │  3. ReceiverValidator                                             ││
│  │     - Unique names                                                ││
│  │     - Required integrations                                       ││
│  │     - Slack, PagerDuty, Webhook, Email configs                    ││
│  │     - Template references                                         ││
│  │                                                                    ││
│  │  4. InhibitionValidator                                           ││
│  │     - Source/target matchers                                      ││
│  │     - Equal labels                                                ││
│  │     - Duplicate rules                                             ││
│  │     - Self-inhibition detection                                   ││
│  │                                                                    ││
│  │  5. SilenceValidator                                              ││
│  │     - Matcher syntax                                              ││
│  │     - Time range validation                                       ││
│  │     - Required fields                                             ││
│  │                                                                    ││
│  │  6. TemplateValidator                                             ││
│  │     - Template file existence                                     ││
│  │     - Go template syntax                                          ││
│  │     - Function availability                                       ││
│  │                                                                    ││
│  │  7. GlobalValidator                                               ││
│  │     - Resolve timeout                                             ││
│  │     - SMTP configuration                                          ││
│  │     - HTTP client config                                          ││
│  │                                                                    ││
│  │  8. SecurityValidator                                             ││
│  │     - Hardcoded secrets detection                                 ││
│  │     - Weak passwords                                              ││
│  │     - Insecure configurations                                     ││
│  │                                                                    ││
│  │  9. BestPracticesValidator                                        ││
│  │     - Naming conventions                                          ││
│  │     - Performance recommendations                                 ││
│  │     - Grouping suggestions                                        ││
│  └────────────────────────────────────────────────────────────────────┘│
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                      Result Formatter                              ││
│  │  pkg/configvalidator/formatter/                                    ││
│  │                                                                    ││
│  │  • HumanFormatter: Colored terminal output                        ││
│  │  • JSONFormatter: Machine-readable JSON                           ││
│  │  • JUnitFormatter: JUnit XML for CI/CD                            ││
│  │  • SarifFormatter: SARIF format for GitHub                        ││
│  └────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 📦 Package Structure

### Directory Layout

```
alert-history/
├── cmd/
│   └── alertmanager-config-validator/
│       ├── main.go                 # CLI entry point
│       ├── cmd_validate.go         # Validate command
│       ├── cmd_version.go          # Version command
│       └── flags.go                # CLI flags definition
│
├── pkg/
│   └── configvalidator/
│       ├── validator.go            # Main Validator facade
│       ├── options.go              # Validator options
│       ├── result.go               # ValidationResult models
│       ├── errors.go               # Error types
│       │
│       ├── parser/
│       │   ├── parser.go           # Parser interface
│       │   ├── yaml_parser.go     # YAML parser
│       │   ├── json_parser.go     # JSON parser
│       │   └── schema.go           # Schema validation
│       │
│       ├── validators/
│       │   ├── validator.go        # Validator interface
│       │   ├── structural.go       # Structural validator
│       │   ├── route.go            # Route tree validator
│       │   ├── receiver.go         # Receiver validator
│       │   ├── inhibition.go       # Inhibition validator
│       │   ├── silence.go          # Silence validator
│       │   ├── template.go         # Template validator
│       │   ├── global.go           # Global config validator
│       │   ├── security.go         # Security validator
│       │   └── bestpractices.go    # Best practices validator
│       │
│       ├── matcher/
│       │   ├── matcher.go          # Label matcher parser
│       │   ├── regex.go            # Regex validation
│       │   └── operators.go        # Matcher operators
│       │
│       ├── formatter/
│       │   ├── formatter.go        # Formatter interface
│       │   ├── human.go            # Human-readable formatter
│       │   ├── json.go             # JSON formatter
│       │   ├── junit.go            # JUnit XML formatter
│       │   └── sarif.go            # SARIF formatter
│       │
│       └── testdata/
│           ├── valid/              # Valid config examples
│           ├── invalid/            # Invalid config examples
│           └── real/               # Real-world configs
│
├── internal/
│   └── alertmanager/
│       └── config/
│           ├── models.go           # Alertmanager config models
│           ├── route.go            # Route models
│           ├── receiver.go         # Receiver models
│           ├── inhibit.go          # Inhibition models
│           └── silence.go          # Silence models
│
└── docs/
    ├── validator/
    │   ├── USER_GUIDE.md           # User guide
    │   ├── EXAMPLES.md             # Usage examples
    │   └── ERROR_CODES.md          # Error code reference
    └── integration/
        └── CI_CD.md                # CI/CD integration guide
```

---

## 🔧 Core Components Design

### 1. Validator Facade

**File**: `pkg/configvalidator/validator.go`

```go
package configvalidator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/configvalidator/parser"
	"github.com/vitaliisemenov/alert-history/pkg/configvalidator/validators"
)

// Validator validates Alertmanager configuration files
type Validator struct {
	opts       Options
	parser     parser.Parser
	validators []validators.Validator
}

// New creates a new Validator with given options
func New(opts Options) *Validator {
	if opts.Mode == "" {
		opts.Mode = StrictMode
	}

	return &Validator{
		opts:   opts,
		parser: parser.NewMultiFormatParser(),
		validators: []validators.Validator{
			validators.NewStructuralValidator(),
			validators.NewRouteValidator(),
			validators.NewReceiverValidator(),
			validators.NewInhibitionValidator(),
			validators.NewSilenceValidator(),
			validators.NewTemplateValidator(),
			validators.NewGlobalValidator(),
			validators.NewSecurityValidator(),
			validators.NewBestPracticesValidator(),
		},
	}
}

// ValidateFile validates configuration from a file
//
// Parameters:
//   - path: Path to configuration file (YAML or JSON)
//
// Returns:
//   - *Result: Validation result with errors/warnings/info
//   - error: Error if file cannot be read or parsed
//
// Performance: < 100ms p95 for typical configs
func (v *Validator) ValidateFile(path string) (*Result, error) {
	startTime := time.Now()

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Validate
	result, err := v.ValidateBytes(data)
	if err != nil {
		return nil, err
	}

	// Set metadata
	result.FilePath = path
	result.Duration = time.Since(startTime)

	return result, nil
}

// ValidateBytes validates configuration from bytes
//
// Parameters:
//   - data: Configuration data (YAML or JSON)
//
// Returns:
//   - *Result: Validation result
//   - error: Error if parsing fails
func (v *Validator) ValidateBytes(data []byte) (*Result, error) {
	// Parse configuration
	cfg, parseErrors := v.parser.Parse(data)
	if len(parseErrors) > 0 {
		return &Result{
			Valid:  false,
			Errors: parseErrors,
		}, nil
	}

	// Validate configuration
	return v.ValidateConfig(cfg)
}

// ValidateConfig validates a parsed configuration
//
// Parameters:
//   - cfg: Parsed Alertmanager configuration
//
// Returns:
//   - *Result: Validation result with all errors/warnings/suggestions
//   - error: Always nil (validation errors returned in Result)
func (v *Validator) ValidateConfig(cfg *Config) (*Result, error) {
	result := &Result{
		Valid:       true,
		Errors:      make([]Error, 0),
		Warnings:    make([]Warning, 0),
		Info:        make([]Info, 0),
		Suggestions: make([]Suggestion, 0),
	}

	ctx := context.Background()

	// Run all validators
	for _, validator := range v.validators {
		// Skip validators for specific sections if filtered
		if len(v.opts.Sections) > 0 && !v.shouldRunValidator(validator, v.opts.Sections) {
			continue
		}

		vResult := validator.Validate(ctx, cfg)
		result.Merge(vResult)
	}

	// Determine validity based on mode
	result.Valid = v.isValid(result)

	return result, nil
}

// shouldRunValidator checks if validator should run for given sections
func (v *Validator) shouldRunValidator(validator validators.Validator, sections []string) bool {
	for _, section := range sections {
		if validator.Supports(section) {
			return true
		}
	}
	return false
}

// isValid determines if result is valid based on validation mode
func (v *Validator) isValid(result *Result) bool {
	switch v.opts.Mode {
	case StrictMode:
		return len(result.Errors) == 0 && len(result.Warnings) == 0
	case LenientMode:
		return len(result.Errors) == 0
	case PermissiveMode:
		return true
	default:
		return len(result.Errors) == 0
	}
}
```

### 2. Validation Result Models

**File**: `pkg/configvalidator/result.go`

```go
package configvalidator

import (
	"encoding/json"
	"fmt"
	"time"
)

// Result represents validation result
type Result struct {
	// Valid indicates if configuration is valid
	Valid bool `json:"valid"`

	// Errors are critical validation errors (block deployment)
	Errors []Error `json:"errors,omitempty"`

	// Warnings are potential problems (don't block, but should be fixed)
	Warnings []Warning `json:"warnings,omitempty"`

	// Info are recommendations and best practices
	Info []Info `json:"info,omitempty"`

	// Suggestions are actionable improvements
	Suggestions []Suggestion `json:"suggestions,omitempty"`

	// FilePath is the validated file path
	FilePath string `json:"file_path,omitempty"`

	// Duration is validation duration
	Duration time.Duration `json:"duration,omitempty"`
}

// Error represents a validation error
type Error struct {
	// Type is error type (e.g., "syntax", "reference", "type")
	Type string `json:"type"`

	// Code is error code (e.g., "E001", "E002")
	Code string `json:"code"`

	// Message is human-readable error message
	Message string `json:"message"`

	// Location is error location in file
	Location Location `json:"location"`

	// Context is surrounding code context
	Context string `json:"context,omitempty"`

	// Suggestion is how to fix the error
	Suggestion string `json:"suggestion,omitempty"`

	// DocsURL is link to relevant documentation
	DocsURL string `json:"docs_url,omitempty"`
}

// Warning represents a validation warning
type Warning struct {
	Type       string   `json:"type"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Location   Location `json:"location"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// Info represents informational message
type Info struct {
	Type     string   `json:"type"`
	Message  string   `json:"message"`
	Location Location `json:"location,omitempty"`
}

// Suggestion represents improvement suggestion
type Suggestion struct {
	Type    string   `json:"type"`
	Message string   `json:"message"`
	Before  string   `json:"before,omitempty"`
	After   string   `json:"after,omitempty"`
}

// Location represents location in configuration file
type Location struct {
	// File is file path
	File string `json:"file,omitempty"`

	// Line is line number (1-based)
	Line int `json:"line"`

	// Column is column number (1-based)
	Column int `json:"column,omitempty"`

	// Field is field path (e.g., "route.receiver")
	Field string `json:"field,omitempty"`
}

// Merge merges another result into this result
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}

	r.Errors = append(r.Errors, other.Errors...)
	r.Warnings = append(r.Warnings, other.Warnings...)
	r.Info = append(r.Info, other.Info...)
	r.Suggestions = append(r.Suggestions, other.Suggestions...)

	if len(other.Errors) > 0 {
		r.Valid = false
	}
}

// Summary returns a summary string
func (r *Result) Summary() string {
	if r.Valid {
		return fmt.Sprintf("✓ Configuration is valid (validated in %s)", r.Duration)
	}

	return fmt.Sprintf("✗ Configuration is invalid: %d errors, %d warnings",
		len(r.Errors), len(r.Warnings))
}

// ExitCode returns appropriate exit code for CLI
func (r *Result) ExitCode(mode ValidationMode) int {
	if len(r.Errors) > 0 {
		return 1 // Errors always fail
	}

	if mode == StrictMode && len(r.Warnings) > 0 {
		return 2 // Warnings fail in strict mode
	}

	return 0 // Success
}

// MarshalJSON implements json.Marshaler
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		*Alias
		DurationMS int64 `json:"duration_ms"`
	}{
		Alias:      (*Alias)(r),
		DurationMS: r.Duration.Milliseconds(),
	})
}
```

### 3. Parser Layer

**File**: `pkg/configvalidator/parser/parser.go`

```go
package parser

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Parser parses configuration files
type Parser interface {
	// Parse parses configuration data
	Parse(data []byte) (*Config, []Error)

	// ParseFile parses configuration from file
	ParseFile(path string) (*Config, []Error)
}

// MultiFormatParser supports both YAML and JSON
type MultiFormatParser struct {
	yamlParser *YAMLParser
	jsonParser *JSONParser
}

// NewMultiFormatParser creates a new multi-format parser
func NewMultiFormatParser() *MultiFormatParser {
	return &MultiFormatParser{
		yamlParser: NewYAMLParser(),
		jsonParser: NewJSONParser(),
	}
}

// Parse tries YAML first, then JSON
func (p *MultiFormatParser) Parse(data []byte) (*Config, []Error) {
	// Try YAML first
	cfg, yamlErrors := p.yamlParser.Parse(data)
	if len(yamlErrors) == 0 {
		return cfg, nil
	}

	// Try JSON
	cfg, jsonErrors := p.jsonParser.Parse(data)
	if len(jsonErrors) == 0 {
		return cfg, nil
	}

	// Both failed, return YAML errors (more common format)
	return nil, yamlErrors
}

// YAMLParser parses YAML configuration
type YAMLParser struct{}

// NewYAMLParser creates a new YAML parser
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse parses YAML data
func (p *YAMLParser) Parse(data []byte) (*Config, []Error) {
	var cfg Config

	// Parse with strict mode (fail on unknown fields)
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return nil, []Error{p.convertYAMLError(err)}
	}

	return &cfg, nil
}

// convertYAMLError converts YAML parsing error to validation error
func (p *YAMLParser) convertYAMLError(err error) Error {
	// Extract line/column from YAML error
	// YAML errors typically have format: "yaml: line X: ..."

	return Error{
		Type:     "syntax",
		Code:     "E001",
		Message:  fmt.Sprintf("YAML syntax error: %v", err),
		Location: extractLocationFromYAMLError(err),
	}
}

// JSONParser parses JSON configuration
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses JSON data
func (p *JSONParser) Parse(data []byte) (*Config, []Error) {
	var cfg Config

	// Parse with strict mode
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&cfg); err != nil {
		return nil, []Error{p.convertJSONError(err)}
	}

	return &cfg, nil
}

// convertJSONError converts JSON parsing error to validation error
func (p *JSONParser) convertJSONError(err error) Error {
	return Error{
		Type:     "syntax",
		Code:     "E002",
		Message:  fmt.Sprintf("JSON syntax error: %v", err),
		Location: extractLocationFromJSONError(err),
	}
}
```

### 4. Route Validator

**File**: `pkg/configvalidator/validators/route.go`

```go
package validators

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// RouteValidator validates route configuration
type RouteValidator struct{}

// NewRouteValidator creates a new route validator
func NewRouteValidator() *RouteValidator {
	return &RouteValidator{}
}

// Validate validates route configuration
func (v *RouteValidator) Validate(ctx context.Context, cfg *Config) *Result {
	result := NewResult()

	if cfg.Route == nil {
		result.AddError(Error{
			Type:    "missing",
			Code:    "E100",
			Message: "Missing required 'route' section",
			Location: Location{
				Field: "route",
			},
			Suggestion: "Add a 'route' section with at least a receiver",
			DocsURL:    "https://prometheus.io/docs/alerting/latest/configuration/#route",
		})
		return result
	}

	// Validate route tree
	v.validateRouteTree(cfg.Route, cfg.Receivers, result, "route", 0)

	// Detect dead routes
	v.detectDeadRoutes(cfg.Route, result)

	// Validate receiver references
	v.validateReceiverReferences(cfg.Route, cfg.Receivers, result)

	return result
}

// validateRouteTree validates route tree structure recursively
func (v *RouteValidator) validateRouteTree(
	route *Route,
	receivers []Receiver,
	result *Result,
	path string,
	depth int,
) {
	if depth > 100 {
		result.AddError(Error{
			Type:    "structure",
			Code:    "E101",
			Message: "Route tree too deep (max 100 levels)",
			Location: Location{Field: path},
		})
		return
	}

	// Validate receiver reference
	if route.Receiver != "" {
		if !v.receiverExists(route.Receiver, receivers) {
			result.AddError(Error{
				Type:    "reference",
				Code:    "E102",
				Message: fmt.Sprintf("Receiver '%s' not found", route.Receiver),
				Location: Location{Field: path + ".receiver"},
				Suggestion: fmt.Sprintf(
					"Add receiver '%s' to 'receivers' section or fix typo. Available: %s",
					route.Receiver,
					v.formatReceiverNames(receivers),
				),
			})
		}
	} else if depth == 0 {
		result.AddError(Error{
			Type:    "missing",
			Code:    "E103",
			Message: "Root route must have a receiver",
			Location: Location{Field: path},
		})
	}

	// Validate matchers
	for i, matcher := range route.Matchers {
		if err := v.validateMatcher(matcher); err != nil {
			result.AddError(Error{
				Type:    "matcher",
				Code:    "E104",
				Message: fmt.Sprintf("Invalid matcher: %v", err),
				Location: Location{
					Field: fmt.Sprintf("%s.matchers[%d]", path, i),
				},
			})
		}
	}

	// Validate group_by
	if len(route.GroupBy) == 0 && depth == 0 {
		result.AddWarning(Warning{
			Type:    "best_practice",
			Code:    "W100",
			Message: "Root route has no 'group_by', alerts will be grouped by all labels",
			Location: Location{Field: path + ".group_by"},
			Suggestion: "Consider adding group_by: ['alertname', 'cluster'] for better grouping",
		})
	}

	// Validate intervals
	if route.GroupWait != nil && *route.GroupWait <= 0 {
		result.AddError(Error{
			Type:    "value",
			Code:    "E105",
			Message: "group_wait must be positive",
			Location: Location{Field: path + ".group_wait"},
		})
	}

	if route.GroupInterval != nil && *route.GroupInterval <= 0 {
		result.AddError(Error{
			Type:    "value",
			Code:    "E106",
			Message: "group_interval must be positive",
			Location: Location{Field: path + ".group_interval"},
		})
	}

	if route.RepeatInterval != nil && *route.RepeatInterval <= 0 {
		result.AddError(Error{
			Type:    "value",
			Code:    "E107",
			Message: "repeat_interval must be positive",
			Location: Location{Field: path + ".repeat_interval"},
		})
	}

	// Validate child routes recursively
	for i, child := range route.Routes {
		childPath := fmt.Sprintf("%s.routes[%d]", path, i)
		v.validateRouteTree(&child, receivers, result, childPath, depth+1)
	}
}

// validateMatcher validates label matcher syntax
func (v *RouteValidator) validateMatcher(matcher string) error {
	// Matcher format: label=value, label!=value, label=~regex, label!~regex

	parts := strings.SplitN(matcher, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, expected label=value")
	}

	label := parts[0]
	operator := "="
	value := parts[1]

	// Check for != or =~ or !~
	if strings.HasSuffix(label, "!") {
		label = strings.TrimSuffix(label, "!")
		operator = "!="
	}

	if strings.HasSuffix(label, "~") {
		label = strings.TrimSuffix(label, "~")
		if operator == "!=" {
			operator = "!~"
		} else {
			operator = "=~"
		}
	}

	// Validate label name
	if !isValidLabelName(label) {
		return fmt.Errorf("invalid label name '%s'", label)
	}

	// Validate regex if regex operator
	if operator == "=~" || operator == "!~" {
		if _, err := regexp.Compile(value); err != nil {
			return fmt.Errorf("invalid regex '%s': %v", value, err)
		}
	}

	return nil
}

// detectDeadRoutes detects unreachable routes
func (v *RouteValidator) detectDeadRoutes(route *Route, result *Result) {
	// TODO: Implement dead route detection algorithm
	// Routes are dead if:
	// - Parent has matcher that makes child impossible
	// - Sibling route matches everything before this route
}

// receiverExists checks if receiver with given name exists
func (v *RouteValidator) receiverExists(name string, receivers []Receiver) bool {
	for _, r := range receivers {
		if r.Name == name {
			return true
		}
	}
	return false
}

// formatReceiverNames formats receiver names for suggestions
func (v *RouteValidator) formatReceiverNames(receivers []Receiver) string {
	names := make([]string, len(receivers))
	for i, r := range receivers {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

// Supports returns true if validator supports given section
func (v *RouteValidator) Supports(section string) bool {
	return section == "route" || section == "routes"
}

// isValidLabelName checks if label name is valid
func isValidLabelName(name string) bool {
	// Label names must match [a-zA-Z_][a-zA-Z0-9_]*
	if len(name) == 0 {
		return false
	}

	if !((name[0] >= 'a' && name[0] <= 'z') ||
		(name[0] >= 'A' && name[0] <= 'Z') ||
		name[0] == '_') {
		return false
	}

	for i := 1; i < len(name); i++ {
		if !((name[i] >= 'a' && name[i] <= 'z') ||
			(name[i] >= 'A' && name[i] <= 'Z') ||
			(name[i] >= '0' && name[i] <= '9') ||
			name[i] == '_') {
			return false
		}
	}

	return true
}
```

### 5. Security Validator

**File**: `pkg/configvalidator/validators/security.go`

```go
package validators

import (
	"context"
	"regexp"
	"strings"
)

// SecurityValidator validates security aspects
type SecurityValidator struct{
	secretPatterns []*regexp.Regexp
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		secretPatterns: []*regexp.Regexp{
			// API keys
			regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?token)[\s]*[:=][\s]*["']?([a-zA-Z0-9_-]{20,})["']?`),
			// Passwords
			regexp.MustCompile(`(?i)(password|passwd|pwd)[\s]*[:=][\s]*["']?([^"'\s]{8,})["']?`),
			// Bearer tokens
			regexp.MustCompile(`(?i)(bearer|token)[\s]*[:=][\s]*["']?([a-zA-Z0-9_.-]{20,})["']?`),
			// AWS keys
			regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			// Private keys
			regexp.MustCompile(`-----BEGIN (RSA|EC|OPENSSH) PRIVATE KEY-----`),
		},
	}
}

// Validate validates security aspects
func (v *SecurityValidator) Validate(ctx context.Context, cfg *Config) *Result {
	result := NewResult()

	// Check for hardcoded secrets
	v.checkHardcodedSecrets(cfg, result)

	// Check for weak configurations
	v.checkWeakConfigurations(cfg, result)

	// Check for insecure HTTP
	v.checkInsecureHTTP(cfg, result)

	return result
}

// checkHardcodedSecrets checks for hardcoded secrets in configuration
func (v *SecurityValidator) checkHardcodedSecrets(cfg *Config, result *Result) {
	// Convert config to string for pattern matching
	cfgStr := fmt.Sprintf("%+v", cfg)

	for _, pattern := range v.secretPatterns {
		matches := pattern.FindAllString(cfgStr, -1)
		for _, match := range matches {
			result.AddError(Error{
				Type:    "security",
				Code:    "E300",
				Message: "Hardcoded secret detected",
				Suggestion: "Use *_file suffix to read secret from file, or use environment variables",
				DocsURL: "https://prometheus.io/docs/alerting/latest/configuration/#_file",
			})
		}
	}
}

// checkWeakConfigurations checks for weak/insecure configurations
func (v *SecurityValidator) checkWeakConfigurations(cfg *Config, result *Result) {
	// Check for missing TLS verification
	for _, receiver := range cfg.Receivers {
		for _, webhook := range receiver.WebhookConfigs {
			if webhook.HTTPConfig != nil && webhook.HTTPConfig.TLSConfig != nil {
				if webhook.HTTPConfig.TLSConfig.InsecureSkipVerify {
					result.AddWarning(Warning{
						Type:    "security",
						Code:    "W300",
						Message: fmt.Sprintf("Receiver '%s': insecure_skip_verify is enabled", receiver.Name),
						Suggestion: "Enable TLS verification for production environments",
					})
				}
			}
		}
	}
}

// checkInsecureHTTP checks for HTTP usage where HTTPS should be used
func (v *SecurityValidator) checkInsecureHTTP(cfg *Config, result *Result) {
	for _, receiver := range cfg.Receivers {
		for _, webhook := range receiver.WebhookConfigs {
			if strings.HasPrefix(webhook.URL, "http://") {
				result.AddWarning(Warning{
					Type:    "security",
					Code:    "W301",
					Message: fmt.Sprintf("Receiver '%s': webhook uses HTTP instead of HTTPS", receiver.Name),
					Suggestion: "Use HTTPS for webhook URLs in production",
				})
			}
		}

		for _, slack := range receiver.SlackConfigs {
			if slack.APIURL != "" && strings.HasPrefix(slack.APIURL, "http://") {
				result.AddWarning(Warning{
					Type:    "security",
					Code:    "W302",
					Message: fmt.Sprintf("Receiver '%s': Slack API URL uses HTTP", receiver.Name),
				})
			}
		}
	}
}

// Supports returns true if validator supports given section
func (v *SecurityValidator) Supports(section string) bool {
	return true // Security validation applies to all sections
}
```

---

## 🔄 Validation Flow

### Detailed Validation Pipeline

```
┌──────────────────────────────────────────────────────────────┐
│  INPUT: alertmanager.yml                                     │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  PHASE 1: PARSING                                            │
│  • Detect format (YAML/JSON)                                 │
│  • Parse syntax                                               │
│  • Build AST                                                  │
│  • Unmarshal to Config struct                                │
│  ────────────────────────────────────────                    │
│  Errors: E001 (YAML syntax), E002 (JSON syntax)             │
│  ────────────────────────────────────────                    │
│  Performance: 5-10ms                                          │
└──────────────────────┬───────────────────────────────────────┘
                       │ ✅ No syntax errors
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  PHASE 2: STRUCTURAL VALIDATION                              │
│  • Required fields present                                    │
│  • Types correct (string, int, duration, bool)               │
│  • Formats valid (URL, email, regex)                         │
│  • Ranges valid (min/max)                                    │
│  ────────────────────────────────────────                    │
│  Errors: E010-E050 (type, format, range errors)             │
│  ────────────────────────────────────────                    │
│  Performance: 5-10ms                                          │
└──────────────────────┬───────────────────────────────────────┘
                       │ ✅ Structure valid
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  PHASE 3: SEMANTIC VALIDATION (parallel)                     │
│                                                               │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────┐│
│  │ Route Validator │  │ Receiver Valid.  │  │ Inhibition  ││
│  │ • Tree struct   │  │ • Unique names   │  │ • Matchers  ││
│  │ • Receiver refs │  │ • Integrations   │  │ • Equality  ││
│  │ • Matchers      │  │ • Templates      │  │ • No dups   ││
│  │ • Dead routes   │  │ • URLs valid     │  │             ││
│  └─────────────────┘  └──────────────────┘  └─────────────┘│
│                                                               │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────┐│
│  │ Silence Valid.  │  │ Template Valid.  │  │ Global Val. ││
│  │ • Matchers      │  │ • Files exist    │  │ • Timeouts  ││
│  │ • Time ranges   │  │ • Syntax valid   │  │ • SMTP      ││
│  │ • Required flds │  │ • Funcs available│  │ • HTTP      ││
│  └─────────────────┘  └──────────────────┘  └─────────────┘│
│  ────────────────────────────────────────                    │
│  Errors: E100-E299 (semantic errors)                         │
│  Warnings: W100-W299 (semantic warnings)                     │
│  ────────────────────────────────────────                    │
│  Performance: 20-40ms (parallel)                              │
└──────────────────────┬───────────────────────────────────────┘
                       │ ✅ Semantics valid
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  PHASE 4: SECURITY VALIDATION                                │
│  • Hardcoded secrets detection                                │
│  • Weak passwords                                             │
│  • Insecure HTTP                                              │
│  • TLS verification disabled                                  │
│  ────────────────────────────────────────                    │
│  Errors: E300-E349 (critical security issues)                │
│  Warnings: W300-W349 (security recommendations)              │
│  ────────────────────────────────────────                    │
│  Performance: 5-10ms                                          │
└──────────────────────┬───────────────────────────────────────┘
                       │ ✅ Security OK
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  PHASE 5: BEST PRACTICES                                     │
│  • Naming conventions                                         │
│  • Grouping recommendations                                   │
│  • Performance optimizations                                  │
│  • Documentation suggestions                                  │
│  ────────────────────────────────────────                    │
│  Info: I001-I099 (recommendations)                           │
│  Suggestions: S001-S099 (improvements)                       │
│  ────────────────────────────────────────                    │
│  Performance: 5-10ms                                          │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  OUTPUT: ValidationResult                                    │
│  • Valid: true/false                                          │
│  • Errors: [] (0 if valid)                                   │
│  • Warnings: [] (may be present)                             │
│  • Info: [] (recommendations)                                │
│  • Suggestions: [] (improvements)                            │
│  • Duration: ~50-100ms                                        │
└──────────────────────────────────────────────────────────────┘
```

---

## 📈 Performance Optimization

### 1. Parallel Validation
- Run independent validators in parallel (goroutines)
- Route, Receiver, Inhibition validators independent
- Reduce total validation time by 2-3x

### 2. Caching
- Cache parsed configs (dev mode)
- Cache validation results for unchanged files
- Cache regex compilations

### 3. Incremental Validation
- Validate only changed sections (if diff available)
- Skip unchanged parts

### 4. Early Exit
- Stop validation on first critical error (optional)
- Configurable via `--fail-fast` flag

---

## 🧪 Testing Strategy

### 1. Unit Tests (≥60 tests, 95% coverage)
- Parser tests (YAML, JSON, error handling)
- Each validator separately
- Error message formatting
- Edge cases (empty config, huge config, malformed)

### 2. Integration Tests (≥20 real configs)
- Valid Alertmanager configs
- Invalid configs (various error types)
- Real-world production configs
- Alertmanager test fixtures

### 3. Fuzz Testing
- YAML parser fuzzing
- JSON parser fuzzing
- Regex matcher fuzzing

### 4. Benchmarks (≥5)
- Small config (<100 LOC)
- Medium config (~500 LOC)
- Large config (~5000 LOC)
- Parallel validation
- Sequential validation

### 5. Golden Tests
- Expected output for known configs
- Regression detection

---

## 📝 Implementation Checklist

### Phase 1: Core Infrastructure (3-4h)
- [ ] Package structure
- [ ] Validator facade
- [ ] Result models
- [ ] Options & modes
- [ ] Parser interface

### Phase 2: Parsers (2-3h)
- [ ] YAML parser
- [ ] JSON parser
- [ ] Schema validation
- [ ] Error handling

### Phase 3: Validators (8-10h)
- [ ] Structural validator
- [ ] Route validator
- [ ] Receiver validator
- [ ] Inhibition validator
- [ ] Silence validator
- [ ] Template validator
- [ ] Global validator
- [ ] Security validator
- [ ] Best practices validator

### Phase 4: CLI Tool (2-3h)
- [ ] CLI entry point
- [ ] Commands (validate, version)
- [ ] Flags parsing
- [ ] Output formatting

### Phase 5: Formatters (2-3h)
- [ ] Human formatter (colored)
- [ ] JSON formatter
- [ ] JUnit formatter
- [ ] SARIF formatter

### Phase 6: Testing (4-5h)
- [ ] Unit tests
- [ ] Integration tests
- [ ] Benchmarks
- [ ] Fuzz tests
- [ ] Golden tests

### Phase 7: Documentation (2-3h)
- [ ] USER_GUIDE.md
- [ ] EXAMPLES.md
- [ ] ERROR_CODES.md
- [ ] CI_CD.md

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Total Lines**: 1,150 LOC
