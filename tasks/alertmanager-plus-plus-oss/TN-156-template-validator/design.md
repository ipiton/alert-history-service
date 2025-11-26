# TN-156: Template Validator - Technical Design Specification

**Task ID**: TN-156
**Phase**: Phase 11 - Template System
**Priority**: P1 (High)
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Date**: 2025-11-25

---

## 🏗️ Architecture Overview

### System Context Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                     TN-156: Template Validator                       │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │               CLI Layer (cmd/template-validator/)               │ │
│  │  • cobra CLI framework                                          │ │
│  │  • validate command (single file, batch, recursive)             │ │
│  │  • Output formatters (human, JSON, SARIF)                       │ │
│  │  • Exit code handling (0, 1, 2)                                 │ │
│  └──────────────────────────┬─────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │           Library Layer (pkg/templatevalidator/)                │ │
│  │  ┌──────────────────────────────────────────────────────────┐  │ │
│  │  │ Validator Facade                                          │  │ │
│  │  │  • New() factory                                          │  │ │
│  │  │  • Validate() single validation                           │  │ │
│  │  │  • ValidateBatch() parallel validation                    │  │ │
│  │  └──────────────────────┬───────────────────────────────────┘  │ │
│  │                         │                                        │ │
│  │                         ▼                                        │ │
│  │  ┌──────────────────────────────────────────────────────────┐  │ │
│  │  │ Validation Pipeline (sequential phases)                   │  │ │
│  │  │  Phase 1: Syntax Validator                                │  │ │
│  │  │  Phase 2: Semantic Validator                              │  │ │
│  │  │  Phase 3: Security Validator                              │  │ │
│  │  │  Phase 4: Best Practices Validator                        │  │ │
│  │  │  • Error aggregation                                      │  │ │
│  │  │  • Early exit on FailFast                                 │  │ │
│  │  │  • Context cancellation support                           │  │ │
│  │  └──────────────────────┬───────────────────────────────────┘  │ │
│  │                         │                                        │ │
│  │                         ▼                                        │ │
│  │  ┌──────────────────────────────────────────────────────────┐  │ │
│  │  │ SubValidators (independent validators)                    │  │ │
│  │  │  • SyntaxValidator (TN-153 integration)                   │  │ │
│  │  │  • SemanticValidator (Alertmanager model)                 │  │ │
│  │  │  • SecurityValidator (XSS, secrets, injection)            │  │ │
│  │  │  • BestPracticesValidator (performance, readability)      │  │ │
│  │  └──────────────────────┬───────────────────────────────────┘  │ │
│  │                         │                                        │ │
│  │                         ▼                                        │ │
│  │  ┌──────────────────────────────────────────────────────────┐  │ │
│  │  │ Supporting Components                                      │  │ │
│  │  │  • FuzzyMatcher (Levenshtein distance for suggestions)    │  │ │
│  │  │  • ErrorParser (extract line:column from Go errors)       │  │ │
│  │  │  • Formatters (human, JSON, SARIF)                        │  │ │
│  │  └───────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │      Integration with TN-153 Template Engine                    │ │
│  │  • NotificationTemplateEngine.Execute()                         │ │
│  │  • Parse templates with mock Alertmanager data                  │ │
│  │  • Function registry for fuzzy matching                         │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 📦 Package Structure

```
pkg/templatevalidator/
├── validator.go              # Main Validator interface & factory
├── options.go                # ValidateOptions, ValidationMode enums
├── result.go                 # ValidationResult, Error/Warning/Suggestion models
├── pipeline.go               # Validation pipeline orchestration
│
├── validators/
│   ├── validator.go          # SubValidator interface
│   ├── syntax.go             # SyntaxValidator (TN-153 integration)
│   ├── semantic.go           # SemanticValidator (Alertmanager model)
│   ├── security.go           # SecurityValidator (XSS, secrets, injection)
│   └── bestpractices.go      # BestPracticesValidator (performance, readability)
│
├── fuzzy/
│   ├── matcher.go            # FuzzyMatcher interface
│   └── levenshtein.go        # Levenshtein distance implementation
│
├── parser/
│   └── error_parser.go       # Go template error parsing (line:column extraction)
│
└── formatters/
    ├── formatter.go          # OutputFormatter interface
    ├── human.go              # HumanFormatter (colors, emojis)
    ├── json.go               # JSONFormatter
    └── sarif.go              # SARIFFormatter (GitHub/GitLab)

cmd/template-validator/
├── main.go                   # CLI entry point
├── cmd/
│   ├── root.go               # Root command setup
│   ├── validate.go           # Validate command implementation
│   └── version.go            # Version command
└── internal/
    ├── config.go             # CLI configuration
    └── output.go             # Output rendering helpers
```

---

## 🔌 Core Interfaces

### 1. Validator Interface

**File**: `pkg/templatevalidator/validator.go`

```go
package templatevalidator

import (
    "context"
)

// Validator is the main facade for template validation
type Validator interface {
    // Validate validates a single template content
    Validate(ctx context.Context, content string, opts ValidateOptions) (*ValidationResult, error)

    // ValidateFile validates a template file
    ValidateFile(ctx context.Context, filepath string, opts ValidateOptions) (*ValidationResult, error)

    // ValidateBatch validates multiple templates in parallel
    ValidateBatch(ctx context.Context, templates []TemplateInput) ([]ValidationResult, error)
}

// TemplateInput represents a template to validate
type TemplateInput struct {
    // Name is the template identifier (filename or logical name)
    Name string

    // Content is the template content
    Content string

    // Options are validation options for this template
    Options ValidateOptions
}

// New creates a new Validator with default configuration
func New(engine TemplateEngine) Validator {
    return &defaultValidator{
        engine:     engine,
        validators: createValidators(engine),
    }
}

// TemplateEngine is the interface for TN-153 integration
type TemplateEngine interface {
    // Parse parses template content
    Parse(ctx context.Context, content string) error

    // Execute executes template with mock data
    Execute(ctx context.Context, content string, data interface{}) (string, error)

    // Functions returns available template functions
    Functions() []string
}
```

---

### 2. ValidateOptions

**File**: `pkg/templatevalidator/options.go`

```go
package templatevalidator

// ValidationMode controls validation strictness
type ValidationMode string

const (
    // ModeStrict fails on warnings
    ModeStrict ValidationMode = "strict"

    // ModeLenient allows warnings
    ModeLenient ValidationMode = "lenient"

    // ModePermissive allows warnings and some errors
    ModePermissive ValidationMode = "permissive"
)

// ValidationPhase represents a validation phase
type ValidationPhase string

const (
    PhaseSyntax        ValidationPhase = "syntax"
    PhaseSemantic      ValidationPhase = "semantic"
    PhaseSecurity      ValidationPhase = "security"
    PhaseBestPractices ValidationPhase = "best_practices"
)

// ValidateOptions controls validation behavior
type ValidateOptions struct {
    // Mode controls validation strictness (default: ModeLenient)
    Mode ValidationMode

    // Phases controls which validators run (default: all phases)
    Phases []ValidationPhase

    // TemplateType for type-specific validation
    // Values: slack, pagerduty, email, webhook, generic
    TemplateType string

    // MaxErrors limits error collection (0 = collect all, default: 0)
    MaxErrors int

    // FailFast stops validation on first error (default: false)
    FailFast bool

    // ParallelWorkers for batch validation (0 = CPU count, default: 0)
    ParallelWorkers int

    // Timeout for validation (default: 30s)
    Timeout time.Duration
}

// DefaultValidateOptions returns default validation options
func DefaultValidateOptions() ValidateOptions {
    return ValidateOptions{
        Mode:            ModeLenient,
        Phases:          AllPhases(),
        TemplateType:    "generic",
        MaxErrors:       0,
        FailFast:        false,
        ParallelWorkers: 0,
        Timeout:         30 * time.Second,
    }
}

// AllPhases returns all validation phases
func AllPhases() []ValidationPhase {
    return []ValidationPhase{
        PhaseSyntax,
        PhaseSemantic,
        PhaseSecurity,
        PhaseBestPractices,
    }
}
```

---

### 3. ValidationResult

**File**: `pkg/templatevalidator/result.go`

```go
package templatevalidator

import "time"

// ValidationResult contains validation results
type ValidationResult struct {
    // Valid is true if template has no blocking errors
    Valid bool `json:"valid"`

    // Errors are blocking validation errors
    Errors []ValidationError `json:"errors"`

    // Warnings are non-blocking issues
    Warnings []ValidationWarning `json:"warnings"`

    // Info are informational messages
    Info []ValidationInfo `json:"info"`

    // Suggestions are improvement suggestions
    Suggestions []ValidationSuggestion `json:"suggestions"`

    // Metrics contains performance metrics
    Metrics ValidationMetrics `json:"metrics"`
}

// ValidationError represents a blocking error
type ValidationError struct {
    // Phase is the validation phase (syntax, semantic, security, best_practices)
    Phase string `json:"phase"`

    // Severity is error severity (critical, high, medium, low)
    Severity string `json:"severity"`

    // Line is the line number (1-indexed)
    Line int `json:"line"`

    // Column is the column number (1-indexed)
    Column int `json:"column"`

    // Message is the error message
    Message string `json:"message"`

    // Suggestion is an actionable suggestion to fix the error
    Suggestion string `json:"suggestion,omitempty"`

    // Code is the error code (e.g., "syntax-error", "unknown-function")
    Code string `json:"code"`
}

// ValidationWarning represents a non-blocking warning
type ValidationWarning struct {
    Phase      string `json:"phase"`
    Line       int    `json:"line"`
    Column     int    `json:"column"`
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
    Code       string `json:"code"`
}

// ValidationInfo represents informational message
type ValidationInfo struct {
    Message string `json:"message"`
}

// ValidationSuggestion represents an improvement suggestion
type ValidationSuggestion struct {
    Phase      string `json:"phase"`
    Line       int    `json:"line"`
    Column     int    `json:"column"`
    Message    string `json:"message"`
    Suggestion string `json:"suggestion"`
}

// ValidationMetrics contains performance metrics
type ValidationMetrics struct {
    // Duration is total validation duration
    Duration time.Duration `json:"duration_ms"`

    // PhaseDurations is duration per phase
    PhaseDurations map[string]time.Duration `json:"phase_durations"`

    // TemplateSize is template size in bytes
    TemplateSize int `json:"template_size_bytes"`

    // FunctionsFound is count of functions found
    FunctionsFound int `json:"functions_found"`

    // VariablesFound is count of variables found
    VariablesFound int `json:"variables_found"`
}
```

---

### 4. SubValidator Interface

**File**: `pkg/templatevalidator/validators/validator.go`

```go
package validators

import "context"

// SubValidator is the interface for phase validators
type SubValidator interface {
    // Name returns the validator name
    Name() string

    // Phase returns the validation phase
    Phase() string

    // Validate validates template content
    Validate(ctx context.Context, content string, opts ValidateOptions) ([]ValidationError, []ValidationWarning, []ValidationSuggestion, error)

    // Enabled returns true if validator should run for given options
    Enabled(opts ValidateOptions) bool
}
```

---

## 🔧 Validation Pipeline

### Pipeline Flow

```go
// File: pkg/templatevalidator/pipeline.go

type validationPipeline struct {
    validators []validators.SubValidator
}

func (p *validationPipeline) Run(ctx context.Context, content string, opts ValidateOptions) (*ValidationResult, error) {
    result := &ValidationResult{
        Valid:       true,
        Errors:      []ValidationError{},
        Warnings:    []ValidationWarning{},
        Suggestions: []ValidationSuggestion{},
        Metrics:     ValidationMetrics{},
    }

    startTime := time.Now()
    phaseDurations := make(map[string]time.Duration)

    for _, validator := range p.validators {
        // Skip disabled validators
        if !validator.Enabled(opts) {
            continue
        }

        // Check context cancellation
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // Run validator
        phaseStart := time.Now()
        errors, warnings, suggestions, err := validator.Validate(ctx, content, opts)
        phaseDurations[validator.Phase()] = time.Since(phaseStart)

        if err != nil {
            return nil, fmt.Errorf("%s validator failed: %w", validator.Name(), err)
        }

        // Aggregate results
        result.Errors = append(result.Errors, errors...)
        result.Warnings = append(result.Warnings, warnings...)
        result.Suggestions = append(result.Suggestions, suggestions...)

        // Check FailFast
        if opts.FailFast && len(errors) > 0 {
            result.Valid = false
            break
        }

        // Check MaxErrors
        if opts.MaxErrors > 0 && len(result.Errors) >= opts.MaxErrors {
            result.Valid = false
            break
        }
    }

    // Final status
    result.Valid = len(result.Errors) == 0
    result.Metrics = ValidationMetrics{
        Duration:       time.Since(startTime),
        PhaseDurations: phaseDurations,
        TemplateSize:   len(content),
    }

    return result, nil
}
```

---

## 🧩 Validator Implementations

### 1. SyntaxValidator

**File**: `pkg/templatevalidator/validators/syntax.go`

```go
package validators

import (
    "context"
    "fmt"
    "strings"

    "github.com/vitaliisemenov/alert-history/pkg/templatevalidator"
)

type SyntaxValidator struct {
    engine       templatevalidator.TemplateEngine
    fuzzyMatcher FuzzyMatcher
}

func NewSyntaxValidator(engine templatevalidator.TemplateEngine) *SyntaxValidator {
    return &SyntaxValidator{
        engine:       engine,
        fuzzyMatcher: NewLevenshteinMatcher(),
    }
}

func (v *SyntaxValidator) Name() string {
    return "Syntax Validator"
}

func (v *SyntaxValidator) Phase() string {
    return "syntax"
}

func (v *SyntaxValidator) Validate(
    ctx context.Context,
    content string,
    opts templatevalidator.ValidateOptions,
) ([]templatevalidator.ValidationError, []templatevalidator.ValidationWarning, []templatevalidator.ValidationSuggestion, error) {
    errors := []templatevalidator.ValidationError{}
    warnings := []templatevalidator.ValidationWarning{}
    suggestions := []templatevalidator.ValidationSuggestion{}

    // Parse template with TN-153 engine
    err := v.engine.Parse(ctx, content)
    if err != nil {
        // Parse Go template error
        syntaxErr := v.parseTemplateError(err)

        // Add fuzzy function matching suggestion
        if strings.Contains(err.Error(), "function") {
            funcName := v.extractFunctionName(err.Error())
            if suggestion := v.suggestFunction(funcName); suggestion != "" {
                syntaxErr.Suggestion = fmt.Sprintf("Did you mean '%s'?", suggestion)
            }
        }

        errors = append(errors, syntaxErr)
    }

    // Extract functions and variables
    functions := v.extractFunctions(content)
    variables := v.extractVariables(content)

    // Check for common mistakes
    commonIssues := v.checkCommonIssues(content)
    warnings = append(warnings, commonIssues...)

    return errors, warnings, suggestions, nil
}

func (v *SyntaxValidator) parseTemplateError(err error) templatevalidator.ValidationError {
    // Parse "template: <name>:<line>:<column>: <message>"
    errStr := err.Error()
    parts := strings.Split(errStr, ":")

    syntaxErr := templatevalidator.ValidationError{
        Phase:    "syntax",
        Severity: "critical",
        Line:     1,
        Column:   1,
        Message:  errStr,
        Code:     "syntax-error",
    }

    // Extract line number
    if len(parts) >= 2 {
        var line int
        fmt.Sscanf(parts[1], "%d", &line)
        if line > 0 {
            syntaxErr.Line = line
        }
    }

    // Extract column number
    if len(parts) >= 3 {
        var col int
        fmt.Sscanf(parts[2], "%d", &col)
        if col > 0 {
            syntaxErr.Column = col
        }
    }

    // Extract message
    if len(parts) >= 4 {
        syntaxErr.Message = strings.TrimSpace(strings.Join(parts[3:], ":"))
    }

    return syntaxErr
}

func (v *SyntaxValidator) suggestFunction(funcName string) string {
    availableFunctions := v.engine.Functions()
    return v.fuzzyMatcher.FindClosest(funcName, availableFunctions, 3)
}
```

---

### 2. SecurityValidator

**File**: `pkg/templatevalidator/validators/security.go`

```go
package validators

import (
    "context"
    "regexp"
    "strings"
)

type SecurityValidator struct {
    secretPatterns []SecretPattern
}

type SecretPattern struct {
    Name     string
    Pattern  *regexp.Regexp
    Severity string
    Message  string
}

func NewSecurityValidator() *SecurityValidator {
    return &SecurityValidator{
        secretPatterns: []SecretPattern{
            {
                Name:     "API Key",
                Pattern:  regexp.MustCompile(`(?i)(api[-_]?key|apikey)\s*[:=]\s*[\"\']?[a-zA-Z0-9]{16,}`),
                Severity: "critical",
                Message:  "Hardcoded API key detected. Use environment variables or secret management.",
            },
            {
                Name:     "Password",
                Pattern:  regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*[\"\'][^\"\']{8,}`),
                Severity: "critical",
                Message:  "Hardcoded password detected. Never store passwords in templates.",
            },
            {
                Name:     "Bearer Token",
                Pattern:  regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_-]{20,}`),
                Severity: "high",
                Message:  "Bearer token detected. Use secret management instead.",
            },
            // ... more patterns
        },
    }
}

func (v *SecurityValidator) Validate(
    ctx context.Context,
    content string,
    opts ValidateOptions,
) ([]ValidationError, []ValidationWarning, []ValidationSuggestion, error) {
    errors := []ValidationError{}
    warnings := []ValidationWarning{}

    lines := strings.Split(content, "\n")

    // Check for hardcoded secrets
    for i, line := range lines {
        for _, pattern := range v.secretPatterns {
            if pattern.Pattern.MatchString(line) {
                errors = append(errors, ValidationError{
                    Phase:      "security",
                    Severity:   pattern.Severity,
                    Line:       i + 1,
                    Column:     strings.Index(line, pattern.Pattern.FindString(line)) + 1,
                    Message:    pattern.Message,
                    Code:       "hardcoded-secret",
                    Suggestion: "Use environment variables or K8s secrets.",
                })
            }
        }
    }

    // Check for XSS vulnerabilities
    xssWarnings := v.checkXSS(lines)
    warnings = append(warnings, xssWarnings...)

    // Check for template injection
    injectionErrors := v.checkTemplateInjection(lines)
    errors = append(errors, injectionErrors...)

    return errors, warnings, []ValidationSuggestion{}, nil
}

func (v *SecurityValidator) checkXSS(lines []string) []ValidationWarning {
    // Detect unescaped HTML output
    warnings := []ValidationWarning{}

    htmlPattern := regexp.MustCompile(`{{\s*\.[\w.]+\s*}}`)

    for i, line := range lines {
        if htmlPattern.MatchString(line) && !strings.Contains(line, "| html") {
            warnings = append(warnings, ValidationWarning{
                Phase:      "security",
                Line:       i + 1,
                Column:     strings.Index(line, "{{") + 1,
                Message:    "Unescaped output may contain HTML. Consider using '| html' filter if HTML is expected.",
                Code:       "potential-xss",
                Suggestion: "Add '| html' filter for HTML output or verify output is text-only.",
            })
        }
    }

    return warnings
}
```

---

## 🎨 Output Formatters

### SARIF Formatter

**File**: `pkg/templatevalidator/formatters/sarif.go`

```go
package formatters

import (
    "encoding/json"
)

// SARIFFormatter formats results as SARIF (GitHub/GitLab)
type SARIFFormatter struct{}

// SARIFReport is SARIF v2.1.0 schema
type SARIFReport struct {
    Version string     `json:"version"`
    Schema  string     `json:"$schema"`
    Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
    Tool    SARIFTool      `json:"tool"`
    Results []SARIFResult  `json:"results"`
}

type SARIFTool struct {
    Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type SARIFResult struct {
    RuleID    string          `json:"ruleId"`
    Level     string          `json:"level"`
    Message   SARIFMessage    `json:"message"`
    Locations []SARIFLocation `json:"locations"`
}

func (f *SARIFFormatter) Format(results []ValidationResult) (string, error) {
    report := SARIFReport{
        Version: "2.1.0",
        Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
        Runs: []SARIFRun{
            {
                Tool: SARIFTool{
                    Driver: SARIFDriver{
                        Name:    "template-validator",
                        Version: "1.0.0",
                    },
                },
                Results: f.convertToSARIFResults(results),
            },
        },
    }

    data, err := json.MarshalIndent(report, "", "  ")
    if err != nil {
        return "", err
    }

    return string(data), nil
}
```

---

## 🎯 Performance Optimization

### 1. Parallel Batch Validation

```go
func (v *defaultValidator) ValidateBatch(
    ctx context.Context,
    templates []TemplateInput,
) ([]ValidationResult, error) {
    workers := v.opts.ParallelWorkers
    if workers == 0 {
        workers = runtime.NumCPU()
    }

    results := make([]ValidationResult, len(templates))
    jobs := make(chan int, len(templates))
    errors := make(chan error, workers)

    // Worker pool
    var wg sync.WaitGroup
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for idx := range jobs {
                result, err := v.Validate(ctx, templates[idx].Content, templates[idx].Options)
                if err != nil {
                    errors <- err
                    return
                }
                results[idx] = *result
            }
        }()
    }

    // Submit jobs
    for i := range templates {
        jobs <- i
    }
    close(jobs)

    // Wait for completion
    wg.Wait()
    close(errors)

    // Check for errors
    if err := <-errors; err != nil {
        return nil, err
    }

    return results, nil
}
```

### 2. Regex Compilation Cache

```go
var (
    secretPatternsOnce sync.Once
    cachedPatterns     []SecretPattern
)

func getSecretPatterns() []SecretPattern {
    secretPatternsOnce.Do(func() {
        cachedPatterns = []SecretPattern{
            // ... compile patterns once
        }
    })
    return cachedPatterns
}
```

---

## 📊 Metrics & Observability

### Performance Targets

- **Syntax validation**: < 10ms p95
- **Semantic validation**: < 5ms p95
- **Security validation**: < 15ms p95
- **Best practices validation**: < 10ms p95
- **Total validation**: < 20ms p95 (single template)
- **Batch validation**: < 500ms (100 templates, parallel)

### Benchmarks Required

```go
// Benchmarks in pkg/templatevalidator/validator_bench_test.go

func BenchmarkValidate_SmallTemplate(b *testing.B)        // < 1KB
func BenchmarkValidate_MediumTemplate(b *testing.B)       // 10KB
func BenchmarkValidate_LargeTemplate(b *testing.B)        // 64KB
func BenchmarkValidateBatch_100Templates(b *testing.B)    // 100 templates
func BenchmarkSyntaxValidator(b *testing.B)               // Syntax only
func BenchmarkSecurityValidator(b *testing.B)             // Security only
func BenchmarkFuzzyMatching_Levenshtein(b *testing.B)     // Fuzzy matcher
```

---

## 🎯 Quality Assurance

### Test Coverage Targets

- **Overall**: 90%+ (target: 95%)
- **validator.go**: 95%+
- **validators/*.go**: 90%+
- **formatters/*.go**: 85%+
- **CLI**: 80%+ (integration tests)

### Test Categories

1. **Unit Tests** (100+ tests):
   - Syntax validator: 30 tests
   - Semantic validator: 20 tests
   - Security validator: 25 tests
   - Best practices validator: 15 tests
   - Formatters: 10 tests

2. **Integration Tests** (20+ tests):
   - End-to-end validation
   - CLI commands
   - File I/O
   - Batch processing

3. **Benchmarks** (15+ benchmarks):
   - Single validation
   - Batch validation
   - Each validator
   - Fuzzy matching

---

## 🏁 Success Criteria

### Definition of Done

- [ ] All 4 validators implemented
- [ ] CLI tool fully functional
- [ ] 90%+ test coverage achieved
- [ ] 100+ tests passing
- [ ] 15+ benchmarks passing
- [ ] Performance targets met
- [ ] Zero linter errors
- [ ] Zero race conditions
- [ ] Documentation complete
- [ ] Merged to main

### 150% Quality Checklist

- [ ] Advanced features: fuzzy matching, SARIF output, parallel batch
- [ ] Comprehensive testing: 100+ unit tests, 20+ integration tests
- [ ] Detailed documentation: README 800+ LOC, design/requirements/tasks 2,200+ LOC
- [ ] Performance optimization: < 20ms p95, zero allocations hot path
- [ ] Clean code: zero linter errors, zero technical debt

---

*Design Date: 2025-11-25*
*Author: AI Assistant*
*Status: ✅ DESIGN COMPLETE*
