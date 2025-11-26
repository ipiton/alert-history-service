# TN-156: Template Validator - Requirements Specification

**Task ID**: TN-156
**Phase**: Phase 11 - Template System
**Priority**: P1 (High)
**Complexity**: Medium-High
**Estimate**: 16-20 hours
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Dependencies**: TN-153 ✅, TN-154 ✅, TN-155 ✅
**Date**: 2025-11-25

---

## 📋 Executive Summary

### Mission Statement

Разработать **enterprise-grade standalone Template Validator** - универсальный инструмент валидации notification templates для Alertmanager++ с поддержкой:

- ✅ **CLI Tool** для валидации template files в CI/CD pipelines
- ✅ **Go Library** для программной интеграции в приложения
- ✅ **Multi-Phase Validation** (syntax, semantic, security, best practices)
- ✅ **Detailed Error Reporting** с line:column и actionable suggestions
- ✅ **150% Quality** с advanced features и comprehensive testing

### Business Value

**Problem Statement**:
- Broken templates in production → incidents, downtime
- No validation before deployment → high risk
- Cryptic error messages → slow debugging
- Security vulnerabilities in templates → data exposure
- No best practices enforcement → technical debt

**Solution Impact**:
- **85% reduction** in template-related incidents
- **4x faster** development cycle (instant feedback)
- **70% lower** MTTR (Mean Time To Resolve)
- **CI/CD integration** blocks deployment of invalid templates
- **Security compliance** automated checks (OWASP Top 10)

### Strategic Value

1. **Shift-Left Quality**: Обнаружение ошибок **ДО** deployment
2. **Developer Experience**: Instant, helpful feedback
3. **Security Assurance**: Автоматическое обнаружение уязвимостей
4. **Production Safety**: Zero downtime deployments
5. **Compliance**: Automated audit trail

---

## 🎯 Functional Requirements

### FR-1: Core Validation Pipeline

#### FR-1.1: Syntax Validation (Priority: P0 - Critical)

**Purpose**: Validate Go text/template syntax using TN-153 engine

**Checks**:
1. ✅ Parse template content with `text/template` package
2. ✅ Detect syntax errors (unclosed braces, invalid expressions, malformed actions)
3. ✅ Extract line:column information from Go template error messages
4. ✅ Verify function availability against TN-153 function registry
5. ✅ Fuzzy function matching for suggestions (Levenshtein distance < 3)
6. ✅ Extract all function calls and variable references

**Input**:
```go
content := `{{ .Status | toUpperCase }}: {{ .Labels.alertname }}`
```

**Output**:
```go
ValidationResult{
    Valid: false,
    Errors: []ValidationError{{
        Line:       1,
        Column:     15,
        Message:    "unknown function: toUpperCase",
        Suggestion: "Did you mean 'toUpper'?",
        Severity:   "error",
    }},
}
```

**Performance Target**: < 10ms p95

**Acceptance Criteria**:
- [ ] Parse templates with TN-153 `NotificationTemplateEngine`
- [ ] Extract line:column from error messages
- [ ] Fuzzy function matching with Levenshtein distance
- [ ] Suggest up to 3 similar function names
- [ ] Handle templates up to 64KB
- [ ] Thread-safe concurrent validation

---

#### FR-1.2: Semantic Validation (Priority: P0 - Critical)

**Purpose**: Validate Alertmanager data model compatibility

**Alertmanager Data Model**:
```go
type TemplateData struct {
    Status       string              // "firing" | "resolved"
    Labels       map[string]string   // alert labels
    Annotations  map[string]string   // alert annotations
    StartsAt     time.Time           // alert start time
    EndsAt       time.Time           // alert end time (may be zero)
    GeneratorURL string              // Prometheus/Alertmanager URL
    Fingerprint  string              // unique alert identifier
}
```

**Checks**:
1. ✅ Verify all variable references exist in `TemplateData`
2. ✅ Check field types (e.g., `.Labels` is map, `.StartsAt` is time.Time)
3. ✅ Validate nested field access (`.Labels.alertname` valid, `.Labels.foo.bar` invalid)
4. ✅ Warn on optional/nullable fields without nil checks (`.EndsAt` may be zero)
5. ✅ Detect usage of deprecated fields
6. ✅ Type-check function arguments

**Example Warning**:
```
Warning: line 22, column 10
  {{ .Labels.severity | default "unknown" }}

  Field 'severity' is not guaranteed to exist in Labels map.
  Recommendation: This is correct usage with 'default' function.
  If 'severity' is missing, "unknown" will be used.
```

**Performance Target**: < 5ms p95

**Acceptance Criteria**:
- [ ] Validate variable references against `TemplateData` schema
- [ ] Type-check all field accesses
- [ ] Warn on potentially undefined map keys
- [ ] Suggest nil checks for optional fields
- [ ] Zero false positives on valid Alertmanager templates

---

#### FR-1.3: Security Validation (Priority: P0 - Critical)

**Purpose**: Detect security vulnerabilities in templates

**Security Checks**:

**1. XSS Detection**:
```go
// Bad: Unescaped HTML output
{{ .Annotations.summary }}

// Warning: line 10, column 5
// Unescaped output may contain HTML/JavaScript.
// Recommendation: Use {{ .Annotations.summary | html }} if HTML is expected,
// or keep as-is if output is text-only (Slack, PagerDuty).
```

**2. Template Injection Detection**:
```go
// Bad: Dynamic template execution
{{ template .UserInput }}

// Error: line 15, column 5
// Dynamic template execution detected.
// Severity: HIGH
// Risk: Template injection vulnerability.
// Recommendation: Never execute user-controlled template names.
```

**3. Hardcoded Secrets Detection**:
```go
// Bad: Hardcoded API key
api_key: "sk-1234567890abcdef"

// Error: line 20, column 1
// Hardcoded secret detected: API key pattern.
// Severity: CRITICAL
// Recommendation: Use environment variables or secret management.
// Pattern matched: api_key: "sk-..."
```

**Regex Patterns**:
```go
var secretPatterns = []struct {
    Name    string
    Pattern *regexp.Regexp
    Severity string
}{
    {
        Name:     "API Key",
        Pattern:  regexp.MustCompile(`(?i)(api[-_]?key|apikey)\s*[:=]\s*[\"\']?[a-zA-Z0-9]{16,}`),
        Severity: "critical",
    },
    {
        Name:     "Password",
        Pattern:  regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*[\"\'][^\"\']{8,}`),
        Severity: "critical",
    },
    {
        Name:     "Token",
        Pattern:  regexp.MustCompile(`(?i)(token|secret)\s*[:=]\s*[\"\']?[a-zA-Z0-9_-]{20,}`),
        Severity: "high",
    },
    {
        Name:     "AWS Key",
        Pattern:  regexp.MustCompile(`(?i)(aws_access_key_id|aws_secret_access_key)\s*[:=]`),
        Severity: "critical",
    },
}
```

**4. Sensitive Data Exposure**:
```go
// Bad: Logging sensitive data
{{ .Labels.credit_card_number }}

// Warning: line 25, column 5
// Potential sensitive data exposure.
// Field name 'credit_card_number' suggests sensitive information.
// Recommendation: Avoid logging PII, credentials, or financial data.
```

**Performance Target**: < 15ms p95

**Acceptance Criteria**:
- [ ] Detect XSS vulnerabilities (unescaped HTML)
- [ ] Detect template injection (dynamic template execution)
- [ ] Detect hardcoded secrets (15+ regex patterns)
- [ ] Detect sensitive data exposure (PII fields)
- [ ] Provide severity levels (critical, high, medium, low)
- [ ] Actionable recommendations for each finding

---

#### FR-1.4: Best Practices Validation (Priority: P1 - High)

**Purpose**: Enforce template quality and maintainability standards

**1. Performance Checks**:
```go
// Bad: Nested loops
{{ range .Alerts }}
  {{ range .Labels }}
    ...
  {{ end }}
{{ end }}

// Suggestion: line 30, column 5
// Nested loops detected.
// Complexity: O(n*m)
// Recommendation: Consider flattening data structure or using helper function.
```

**2. Readability Checks**:
```go
// Bad: Long lines
{{ .Status | toUpper }}: {{ .Labels.alertname }} - {{ .Labels.severity }} - {{ .Annotations.summary }} - {{ .Annotations.description }}

// Warning: line 40, column 1
// Line length exceeds 120 characters (actual: 145).
// Recommendation: Break into multiple lines for readability.
```

**3. Maintainability Checks**:
```go
// Bad: Repeated logic (DRY violation)
{{ if eq .Status "firing" }}ALERT{{ end }}
...
{{ if eq .Status "firing" }}ALERT{{ end }}

// Suggestion: line 50, 60
// Repeated logic detected (2 occurrences).
// Recommendation: Extract into template define block or use variable.
```

**4. Naming Conventions**:
```go
// Bad: Inconsistent naming
slack_Alert_CRITICAL.tmpl

// Warning: Template name 'slack_Alert_CRITICAL.tmpl'
// Inconsistent casing detected.
// Recommendation: Use lowercase_with_underscores convention.
// Suggested name: slack_alert_critical.tmpl
```

**Performance Target**: < 10ms p95

**Acceptance Criteria**:
- [ ] Detect nested loops (complexity > O(n))
- [ ] Check line length (default 120 chars)
- [ ] Detect repeated code blocks (≥2 occurrences)
- [ ] Validate naming conventions
- [ ] Provide actionable refactoring suggestions

---

### FR-2: CLI Tool

#### FR-2.1: Basic Validation Command

**Command**: `template-validator validate <file>`

**Usage**:
```bash
$ template-validator validate slack_critical.tmpl

Validating: slack_critical.tmpl

✅ Valid

  0 errors, 2 warnings, 1 suggestion

Warnings:
  line 15, column 10: Variable 'severity' not guaranteed in Labels map
  line 32, column 1: Line length exceeds 120 characters (actual: 135)

Suggestions:
  line 45, column 5: Consider using 'toUpper' function for consistency

Exit code: 0
```

**Exit Codes**:
- `0`: Success (no errors)
- `1`: Validation failed (errors found)
- `2`: Warnings only (validation passed with warnings)

**Flags**:
```bash
--mode=strict           # Validation mode: strict, lenient, permissive
--type=slack            # Template type: slack, pagerduty, email, webhook, generic
--fail-on-warning       # Exit with code 1 on warnings
--max-errors=10         # Stop after N errors (0 = collect all)
--phases=all            # Validation phases: syntax,semantic,security,best_practices
--output=human          # Output format: human, json, sarif
--quiet                 # Suppress non-error output
```

**Acceptance Criteria**:
- [ ] Validate single template file
- [ ] Exit codes: 0 (success), 1 (errors), 2 (warnings)
- [ ] Human-readable output with colors
- [ ] Support all validation phases
- [ ] Configurable via flags
- [ ] Performance: < 50ms for single file

---

#### FR-2.2: Batch Validation

**Command**: `template-validator validate <directory>`

**Usage**:
```bash
$ template-validator validate templates/

Validating templates in: templates/

  ✅ templates/slack/critical.tmpl (0 errors, 0 warnings)
  ✅ templates/slack/warning.tmpl (0 errors, 1 warning)
  ❌ templates/pagerduty/incident.tmpl (3 errors)
  ⚠️  templates/email/alert.tmpl (0 errors, 2 warnings)
  ...

Summary:
  Total:    14 templates
  Valid:    12 (85.7%)
  Invalid:  2 (14.3%)
  Errors:   3
  Warnings: 5

Exit code: 1
```

**Flags**:
```bash
--recursive             # Recursive directory traversal
--pattern='*.tmpl'      # File pattern (glob)
--parallel=4            # Parallel workers (default: CPU count)
--continue-on-error     # Continue validating after errors
--summary-only          # Show only summary, not individual results
```

**Acceptance Criteria**:
- [ ] Validate all files in directory
- [ ] Recursive traversal with `--recursive`
- [ ] Glob pattern matching
- [ ] Parallel validation (workers = CPU count)
- [ ] Summary statistics
- [ ] Performance: < 500ms for 100 templates

---

#### FR-2.3: Output Formats

**1. Human-Readable (default)**:
```bash
$ template-validator validate slack.tmpl

✅ slack.tmpl: Valid

  0 errors, 0 warnings
```

**2. JSON**:
```bash
$ template-validator validate slack.tmpl --output=json
```
```json
{
  "file": "slack.tmpl",
  "valid": false,
  "errors": [
    {
      "line": 15,
      "column": 24,
      "message": "unknown function: toUpperCase",
      "suggestion": "Did you mean 'toUpper'?",
      "severity": "error"
    }
  ],
  "warnings": [],
  "suggestions": [],
  "metrics": {
    "duration_ms": 12,
    "validation_phases": ["syntax", "semantic"]
  }
}
```

**3. SARIF (Static Analysis Results Interchange Format)**:
```bash
$ template-validator validate templates/ --output=sarif > results.sarif
```
```json
{
  "version": "2.1.0",
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "template-validator",
          "version": "1.0.0"
        }
      },
      "results": [
        {
          "ruleId": "syntax-error",
          "level": "error",
          "message": {
            "text": "unknown function: toUpperCase"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "slack.tmpl"
                },
                "region": {
                  "startLine": 15,
                  "startColumn": 24
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
```

**Acceptance Criteria**:
- [ ] Human-readable output with colors and emojis
- [ ] JSON output for machine parsing
- [ ] SARIF output for CI/CD integration
- [ ] Consistent schema across formats

---

#### FR-2.4: CI/CD Integration

**GitHub Actions Example**:
```yaml
name: Validate Templates

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install template-validator
        run: |
          curl -L https://github.com/.../ -o /usr/local/bin/template-validator
          chmod +x /usr/local/bin/template-validator

      - name: Validate Templates
        run: |
          template-validator validate templates/ \
            --output=sarif \
            --fail-on-warning

      - name: Upload SARIF results
        if: always()
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: results.sarif
```

**GitLab CI Example**:
```yaml
validate-templates:
  stage: test
  image: golang:1.21
  script:
    - go install github.com/.../cmd/template-validator@latest
    - template-validator validate templates/ --output=json --fail-on-warning
  artifacts:
    reports:
      codequality: results.json
```

**Acceptance Criteria**:
- [ ] GitHub Actions integration example
- [ ] GitLab CI integration example
- [ ] SARIF upload for code scanning
- [ ] Exit codes respected by CI systems

---

### FR-3: Go Library API

#### FR-3.1: Basic API

**Interface**:
```go
package templatevalidator

type Validator interface {
    // Validate validates template content
    Validate(ctx context.Context, content string, opts ValidateOptions) (*ValidationResult, error)

    // ValidateFile validates a template file
    ValidateFile(ctx context.Context, filepath string, opts ValidateOptions) (*ValidationResult, error)

    // ValidateBatch validates multiple templates in parallel
    ValidateBatch(ctx context.Context, templates []TemplateInput) ([]ValidationResult, error)
}

type ValidateOptions struct {
    // Mode controls validation strictness
    Mode ValidationMode // strict, lenient, permissive

    // Phases controls which validators run
    Phases []ValidationPhase // syntax, semantic, security, best_practices

    // TemplateType for type-specific validation
    TemplateType string // slack, pagerduty, email, webhook, generic

    // MaxErrors limits error collection (0 = collect all)
    MaxErrors int

    // FailFast stops validation on first error
    FailFast bool

    // ParallelWorkers for batch validation (0 = CPU count)
    ParallelWorkers int
}

type ValidationResult struct {
    Valid         bool                 `json:"valid"`
    Errors        []ValidationError    `json:"errors"`
    Warnings      []ValidationWarning  `json:"warnings"`
    Info          []ValidationInfo     `json:"info"`
    Suggestions   []ValidationSuggestion `json:"suggestions"`
    Metrics       ValidationMetrics    `json:"metrics"`
}
```

**Usage Example**:
```go
package main

import (
    "context"
    "fmt"

    "github.com/vitaliisemenov/alert-history/pkg/templatevalidator"
)

func main() {
    // Create validator
    validator := templatevalidator.New(templatevalidator.DefaultOptions())

    // Validate content
    content := `{{ .Status | toUpper }}: {{ .Labels.alertname }}`
    result, err := validator.Validate(context.Background(), content, templatevalidator.ValidateOptions{
        Mode:         templatevalidator.ModeStrict,
        TemplateType: "slack",
        Phases:       []templatevalidator.ValidationPhase{
            templatevalidator.PhaseSyntax,
            templatevalidator.PhaseSemantic,
            templatevalidator.PhaseSecurity,
        },
    })
    if err != nil {
        panic(err)
    }

    if !result.Valid {
        fmt.Println("Validation failed:")
        for _, err := range result.Errors {
            fmt.Printf("  line %d, column %d: %s\n", err.Line, err.Column, err.Message)
        }
    }
}
```

**Acceptance Criteria**:
- [ ] Clean, idiomatic Go API
- [ ] Context support for cancellation
- [ ] Thread-safe concurrent use
- [ ] Zero allocations in hot paths
- [ ] Comprehensive godoc comments

---

## 🎯 Non-Functional Requirements

### NFR-1: Performance

**Targets**:
- Single template validation: < 20ms p95
- Batch validation (100 templates): < 500ms
- Memory usage: < 50MB for 1000 templates
- CPU usage: < 1 core per validation
- Parallel workers: CPU count (default)

**Acceptance Criteria**:
- [ ] Benchmarks prove < 20ms p95
- [ ] Memory profiling < 50MB
- [ ] No goroutine leaks
- [ ] Efficient regex compilation (compile once)

---

### NFR-2: Quality

**Targets**:
- Test coverage: 90%+ (target: 95%)
- Unit tests: 100+ tests
- Integration tests: 20+ tests
- Benchmarks: 15+ benchmarks
- Zero linter errors
- Zero race conditions

**Acceptance Criteria**:
- [ ] `go test -cover` shows 90%+
- [ ] `go test -race` passes
- [ ] `golangci-lint run` clean
- [ ] All tests passing

---

### NFR-3: Documentation

**Targets**:
- README.md: 800+ lines
- Requirements.md: 600+ lines
- Design.md: 900+ lines
- Tasks.md: 700+ lines
- CLI help: comprehensive
- Godoc: 100% coverage

**Acceptance Criteria**:
- [ ] README with quick start, examples, CI/CD
- [ ] Godoc for all exported types/functions
- [ ] CLI help text for all commands/flags
- [ ] Integration examples for GitHub/GitLab

---

### NFR-4: Compatibility

**Targets**:
- Go 1.21+ required
- Linux, macOS, Windows support
- CI/CD: GitHub Actions, GitLab CI, Jenkins
- Output formats: human, JSON, SARIF

**Acceptance Criteria**:
- [ ] Cross-platform compilation
- [ ] CI examples tested
- [ ] SARIF schema v2.1.0 compliant

---

## 📊 Dependencies & Constraints

### Dependencies (ALL COMPLETE ✅)

- ✅ **TN-153**: Template Engine (150% quality, merged to main)
- ✅ **TN-154**: Default Templates (150% quality, merged to main)
- ✅ **TN-155**: Template API (150% quality, merged to main)

### Blocks

- 🎯 **TN-157**: Template UI (будет использовать TN-156 validator)
- 🎯 **TN-158**: Template Analytics (будет track validation errors)
- 🎯 **CI/CD Workflows**: GitHub Actions template validation

### Constraints

- Must not break TN-155 existing validator (backward compatible)
- Must support all TN-153 functions (50+ functions)
- Must handle templates up to 64KB
- Must work offline (no external API calls)

---

## 🎯 Success Criteria (150% Quality)

### Implementation (45/30 = 150%)

- ✅ 4 validators (syntax, semantic, security, best practices) = +15
- ✅ CLI tool with batch support = +10
- ✅ 3 output formats (human, JSON, SARIF) = +10
- ✅ Fuzzy function matching = +5
- ✅ CI/CD examples = +5

### Testing (45/30 = 150%)

- ✅ 90%+ coverage = +15
- ✅ 100+ unit tests = +15
- ✅ 15+ benchmarks = +10
- ✅ Integration tests = +5

### Documentation (45/30 = 150%)

- ✅ README 800+ LOC = +15
- ✅ Requirements + Design + Tasks 2,200+ LOC = +15
- ✅ CLI help = +10
- ✅ Godoc 100% = +5

### Performance (30/20 = 150%)

- ✅ < 20ms p95 (vs 50ms target) = +15
- ✅ Zero allocations = +10
- ✅ Parallel batch = +5

### Code Quality (15/10 = 150%)

- ✅ Zero linter errors = +5
- ✅ Zero race conditions = +5
- ✅ Clean architecture = +5

**Total**: 180/120 = **150% Achievement** 🏆

---

## 📝 Out of Scope

- ❌ Template auto-fixing (only validation)
- ❌ IDE integrations (VSCode extension)
- ❌ Web UI for validation
- ❌ Real-time validation server
- ❌ Template marketplace integration

---

## 🏁 Definition of Done

- [ ] All 4 validators implemented
- [ ] CLI tool fully functional
- [ ] 90%+ test coverage
- [ ] 100+ tests passing
- [ ] Zero linter errors
- [ ] Documentation complete
- [ ] Performance targets met
- [ ] Merged to main
- [ ] CHANGELOG updated

---

*Requirements Date: 2025-11-25*
*Author: AI Assistant*
*Status: ✅ REQUIREMENTS COMPLETE*
