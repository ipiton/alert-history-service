# Config Validator - Usage Examples

This document provides comprehensive examples for using the Alertmanager++ Config Validator in various scenarios.

## Table of Contents

- [Basic Examples](#basic-examples)
  - [CLI Validation](#cli-validation)
  - [Go API Integration](#go-api-integration)
- [Advanced Usage](#advanced-usage)
  - [Custom Validation Options](#custom-validation-options)
  - [Programmatic Error Handling](#programmatic-error-handling)
  - [Batch Validation](#batch-validation)
- [Real-World Scenarios](#real-world-scenarios)
  - [CI/CD Pipeline Integration](#cicd-pipeline-integration)
  - [Pre-Deployment Validation](#pre-deployment-validation)
  - [Configuration Migration](#configuration-migration)
- [Validation Examples](#validation-examples)
  - [Route Validation](#route-validation)
  - [Receiver Validation](#receiver-validation)
  - [Inhibition Rules](#inhibition-rules)
  - [Security Checks](#security-checks)
- [Error Handling Patterns](#error-handling-patterns)
- [Performance Optimization](#performance-optimization)
- [Best Practices](#best-practices)

---

## Basic Examples

### CLI Validation

#### Simple Validation

```bash
# Validate a YAML configuration
configvalidator validate alertmanager.yml

# Validate a JSON configuration
configvalidator validate alertmanager.json

# Validate with strict mode
configvalidator validate --strict alertmanager.yml

# Validate with security checks enabled
configvalidator validate --security alertmanager.yml
```

#### Output Formatting

```bash
# JSON output (default)
configvalidator validate --format json alertmanager.yml

# Human-readable output
configvalidator validate --format text alertmanager.yml

# SARIF format for static analysis tools
configvalidator validate --format sarif alertmanager.yml > results.sarif

# Quiet mode (exit code only)
configvalidator validate --quiet alertmanager.yml
echo $?  # 0 = valid, 1 = warnings, 2 = errors
```

#### Validation Modes

```bash
# Strict mode: All warnings become errors
configvalidator validate --mode strict alertmanager.yml

# Lenient mode: Some errors downgraded to warnings
configvalidator validate --mode lenient alertmanager.yml

# Permissive mode: Minimal validation
configvalidator validate --mode permissive alertmanager.yml
```

### Go API Integration

#### Basic Validation

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

    // Validate configuration file
    result, err := validator.ValidateFile("alertmanager.yml")
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    // Check validation results
    if !result.Valid() {
        fmt.Printf("Configuration has %d errors and %d warnings\n",
            len(result.Errors), len(result.Warnings))

        for _, err := range result.Errors {
            fmt.Printf("ERROR [%s]: %s at %s\n",
                err.Code, err.Message, err.Location.Field)
        }

        for _, warn := range result.Warnings {
            fmt.Printf("WARN [%s]: %s at %s\n",
                warn.Code, warn.Message, warn.Location.Field)
        }
    } else {
        fmt.Println("Configuration is valid!")
    }
}
```

#### Validation with Custom Options

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func main() {
    // Create custom logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    // Configure validation options
    opts := types.Options{
        Mode:                   types.StrictMode,
        EnableBestPractices:    true,
        EnableSecurityChecks:   true,
        EnableDeprecatedChecks: true,
        DefaultDocsURL:         "https://alertmanager-plus-plus.io/docs",
        Logger:                 logger,
    }

    // Create validator
    validator := configvalidator.New(opts)

    // Validate
    result, err := validator.ValidateFile("alertmanager.yml")
    if err != nil {
        logger.Error("Validation error", "error", err)
        os.Exit(2)
    }

    // Exit with appropriate code
    os.Exit(result.ExitCode())
}
```

#### Programmatic Configuration Validation

```go
package main

import (
    "context"
    "fmt"

    "github.com/vitaliisemenov/alert-history/internal/alertmanager/config"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func main() {
    // Build configuration programmatically
    cfg := &config.AlertmanagerConfig{
        Global: config.GlobalConfig{
            ResolveTimeout: ptrDuration("5m"),
            SMTPSmarthost:  "smtp.example.com:587",
            SMTPFrom:       "alertmanager@example.com",
        },
        Route: config.Route{
            Receiver:       "default",
            GroupBy:        []string{"alertname", "cluster"},
            GroupWait:      ptrDuration("10s"),
            GroupInterval:  ptrDuration("10s"),
            RepeatInterval: ptrDuration("1h"),
        },
        Receivers: []config.Receiver{
            {
                Name: "default",
                EmailConfigs: []config.EmailConfig{
                    {
                        To:      "team@example.com",
                        Headers: map[string]string{"Subject": "Alert: {{ .GroupLabels.alertname }}"},
                    },
                },
            },
        },
    }

    // Validate programmatically built configuration
    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    result := validator.ValidateConfig(context.Background(), cfg)

    if !result.Valid() {
        fmt.Println("Configuration has issues:")
        for _, err := range result.Errors {
            fmt.Printf("- %s\n", err.Message)
        }
    }
}

func ptrDuration(s string) *config.Duration {
    d := config.Duration(s)
    return &d
}
```

---

## Advanced Usage

### Custom Validation Options

```go
package main

import (
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func createValidator(environment string) *configvalidator.Validator {
    opts := types.DefaultOptions()

    switch environment {
    case "production":
        // Production: Strict validation with all checks
        opts.Mode = types.StrictMode
        opts.EnableBestPractices = true
        opts.EnableSecurityChecks = true
        opts.EnableDeprecatedChecks = true

    case "staging":
        // Staging: Lenient with security checks
        opts.Mode = types.LenientMode
        opts.EnableBestPractices = true
        opts.EnableSecurityChecks = true
        opts.EnableDeprecatedChecks = false

    case "development":
        // Development: Permissive
        opts.Mode = types.PermissiveMode
        opts.EnableBestPractices = false
        opts.EnableSecurityChecks = false
        opts.EnableDeprecatedChecks = false
    }

    return configvalidator.New(opts)
}
```

### Programmatic Error Handling

```go
package main

import (
    "fmt"
    "strings"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func validateWithDetailedReporting(filename string) error {
    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    result, err := validator.ValidateFile(filename)
    if err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Group errors by section
    errorsBySection := make(map[string][]types.Error)
    for _, err := range result.Errors {
        section := err.Location.Section
        if section == "" {
            section = "general"
        }
        errorsBySection[section] = append(errorsBySection[section], err)
    }

    // Report grouped errors
    if len(errorsBySection) > 0 {
        fmt.Println("Validation Errors by Section:")
        for section, errors := range errorsBySection {
            fmt.Printf("\n[%s] - %d errors:\n", strings.ToUpper(section), len(errors))
            for _, err := range errors {
                fmt.Printf("  • %s (code: %s)\n", err.Message, err.Code)
                if err.Suggestion != "" {
                    fmt.Printf("    💡 Suggestion: %s\n", err.Suggestion)
                }
                if err.DocsURL != "" {
                    fmt.Printf("    📖 Docs: %s\n", err.DocsURL)
                }
            }
        }
        return fmt.Errorf("configuration has %d errors", len(result.Errors))
    }

    // Report warnings
    if len(result.Warnings) > 0 {
        fmt.Printf("Configuration has %d warnings (non-blocking)\n", len(result.Warnings))
    }

    return nil
}
```

### Batch Validation

```go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sync"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

type ValidationResult struct {
    Filename string
    Result   *types.Result
    Error    error
}

func validateDirectory(dir string) ([]ValidationResult, error) {
    // Find all YAML/JSON files
    files, err := filepath.Glob(filepath.Join(dir, "*.{yml,yaml,json}"))
    if err != nil {
        return nil, err
    }

    // Create validator
    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    // Validate concurrently
    results := make([]ValidationResult, len(files))
    var wg sync.WaitGroup

    for i, file := range files {
        wg.Add(1)
        go func(idx int, filename string) {
            defer wg.Done()

            result, err := validator.ValidateFile(filename)
            results[idx] = ValidationResult{
                Filename: filepath.Base(filename),
                Result:   result,
                Error:    err,
            }
        }(i, file)
    }

    wg.Wait()
    return results, nil
}

func main() {
    results, err := validateDirectory("./configs")
    if err != nil {
        fmt.Printf("Error scanning directory: %v\n", err)
        os.Exit(2)
    }

    // Report results
    validCount := 0
    for _, r := range results {
        if r.Error != nil {
            fmt.Printf("❌ %s: Failed - %v\n", r.Filename, r.Error)
        } else if r.Result.Valid() {
            fmt.Printf("✅ %s: Valid\n", r.Filename)
            validCount++
        } else {
            fmt.Printf("⚠️  %s: %d errors, %d warnings\n",
                r.Filename, len(r.Result.Errors), len(r.Result.Warnings))
        }
    }

    fmt.Printf("\nSummary: %d/%d configurations are valid\n", validCount, len(results))
}
```

---

## Real-World Scenarios

### CI/CD Pipeline Integration

#### GitHub Actions

```yaml
# .github/workflows/validate-config.yml
name: Validate Alertmanager Config

on:
  push:
    paths:
      - 'configs/alertmanager.yml'
  pull_request:
    paths:
      - 'configs/alertmanager.yml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install Config Validator
        run: |
          go install github.com/vitaliisemenov/alert-history/cmd/configvalidator@latest

      - name: Validate Configuration
        run: |
          configvalidator validate \
            --strict \
            --security \
            --format sarif \
            configs/alertmanager.yml > results.sarif

      - name: Upload SARIF results
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: results.sarif
```

#### GitLab CI

```yaml
# .gitlab-ci.yml
validate-config:
  stage: test
  image: golang:1.21
  script:
    - go install github.com/vitaliisemenov/alert-history/cmd/configvalidator@latest
    - configvalidator validate --strict --security configs/alertmanager.yml
  only:
    changes:
      - configs/alertmanager.yml
```

#### Jenkins Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any

    stages {
        stage('Validate Config') {
            steps {
                script {
                    sh '''
                        go install github.com/vitaliisemenov/alert-history/cmd/configvalidator@latest
                        configvalidator validate \
                            --strict \
                            --security \
                            --format json \
                            configs/alertmanager.yml > validation-result.json
                    '''

                    def result = readJSON file: 'validation-result.json'
                    if (!result.valid) {
                        error("Configuration validation failed with ${result.errors.size()} errors")
                    }
                }
            }
        }
    }
}
```

### Pre-Deployment Validation

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func preDeploymentCheck(configPath string) error {
    fmt.Println("🔍 Pre-deployment validation started...")

    // Create strict validator for production
    opts := types.Options{
        Mode:                   types.StrictMode,
        EnableBestPractices:    true,
        EnableSecurityChecks:   true,
        EnableDeprecatedChecks: true,
    }

    validator := configvalidator.New(opts)

    // Validate configuration
    result, err := validator.ValidateFile(configPath)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    // Check for blocking issues
    if len(result.Errors) > 0 {
        fmt.Println("❌ Validation failed with errors:")
        for _, err := range result.Errors {
            fmt.Printf("  • [%s] %s\n", err.Code, err.Message)
        }
        return fmt.Errorf("configuration has %d errors", len(result.Errors))
    }

    // Report warnings
    if len(result.Warnings) > 0 {
        fmt.Println("⚠️  Configuration has warnings:")
        for _, warn := range result.Warnings {
            fmt.Printf("  • [%s] %s\n", warn.Code, warn.Message)
        }
    }

    // Report info messages
    if len(result.Info) > 0 {
        fmt.Println("ℹ️  Additional information:")
        for _, info := range result.Info {
            fmt.Printf("  • %s\n", info.Message)
        }
    }

    // Report suggestions
    if len(result.Suggestions) > 0 {
        fmt.Println("💡 Improvement suggestions:")
        for _, suggestion := range result.Suggestions {
            fmt.Printf("  • %s\n", suggestion.Message)
        }
    }

    fmt.Println("✅ Pre-deployment validation passed!")
    return nil
}

func main() {
    if err := preDeploymentCheck("alertmanager.yml"); err != nil {
        fmt.Printf("Deployment blocked: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Configuration is ready for deployment")
}
```

### Configuration Migration

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/vitaliisemenov/alert-history/internal/alertmanager/config"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func migrateConfiguration(oldConfigPath, newConfigPath string) error {
    fmt.Println("🔄 Starting configuration migration...")

    // Validate old configuration with permissive mode
    fmt.Println("1. Validating old configuration...")
    oldOpts := types.Options{
        Mode: types.PermissiveMode,
        EnableDeprecatedChecks: true,
    }
    oldValidator := configvalidator.New(oldOpts)

    oldResult, err := oldValidator.ValidateFile(oldConfigPath)
    if err != nil {
        return fmt.Errorf("failed to validate old config: %w", err)
    }

    // Collect deprecated features
    deprecatedFeatures := []string{}
    for _, warn := range oldResult.Warnings {
        if warn.Code[:1] == "W" && warn.Message contains "deprecated" {
            deprecatedFeatures = append(deprecatedFeatures, warn.Message)
        }
    }

    if len(deprecatedFeatures) > 0 {
        fmt.Println("⚠️  Found deprecated features:")
        for _, feature := range deprecatedFeatures {
            fmt.Printf("  • %s\n", feature)
        }
    }

    // Validate new configuration with strict mode
    fmt.Println("2. Validating new configuration...")
    newOpts := types.Options{
        Mode:                   types.StrictMode,
        EnableBestPractices:    true,
        EnableSecurityChecks:   true,
        EnableDeprecatedChecks: true,
    }
    newValidator := configvalidator.New(newOpts)

    newResult, err := newValidator.ValidateFile(newConfigPath)
    if err != nil {
        return fmt.Errorf("failed to validate new config: %w", err)
    }

    if !newResult.Valid() {
        fmt.Println("❌ New configuration has errors:")
        for _, err := range newResult.Errors {
            fmt.Printf("  • [%s] %s\n", err.Code, err.Message)
        }
        return fmt.Errorf("migration validation failed")
    }

    fmt.Println("✅ Configuration migration validation passed!")
    fmt.Printf("   • Old config: %d warnings\n", len(oldResult.Warnings))
    fmt.Printf("   • New config: %d warnings, %d suggestions\n",
        len(newResult.Warnings), len(newResult.Suggestions))

    return nil
}

func main() {
    if err := migrateConfiguration("alertmanager-old.yml", "alertmanager-new.yml"); err != nil {
        fmt.Printf("Migration failed: %v\n", err)
        os.Exit(1)
    }
}
```

---

## Validation Examples

### Route Validation

#### Valid Route Configuration

```yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

  routes:
    - receiver: 'database-team'
      matchers:
        - team="database"
      group_wait: 5s
      continue: false

    - receiver: 'critical-alerts'
      matchers:
        - severity="critical"
      group_interval: 5m
      repeat_interval: 30m

    - receiver: 'frontend-team'
      matchers:
        - team=~"frontend|web"
        - environment="production"
```

#### Common Route Errors

```yaml
# ERROR E100: Missing required receiver
route:
  group_by: ['alertname']
  # Missing receiver field

# ERROR E101: Receiver not found
route:
  receiver: 'nonexistent-receiver'

receivers:
  - name: 'default'

# ERROR E104: Invalid matcher syntax
route:
  receiver: 'default'
  matchers:
    - 'invalid matcher without operator'
    - 'team='  # Empty value

# ERROR E105: Invalid regex
route:
  receiver: 'default'
  matchers:
    - 'alertname=~"[invalid(regex"'

# WARNING W100: Deprecated continue field
route:
  receiver: 'default'
  continue: true  # Deprecated, use routes with continue
```

### Receiver Validation

#### Valid Receiver Configurations

```yaml
receivers:
  # Email receiver
  - name: 'email-team'
    email_configs:
      - to: 'team@example.com'
        from: 'alertmanager@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alertmanager'
        auth_password: '${SMTP_PASSWORD}'
        headers:
          Subject: '{{ .GroupLabels.alertname }} - {{ .Status }}'

  # Slack receiver
  - name: 'slack-notifications'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/T00/B00/XX'
        channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  # PagerDuty receiver
  - name: 'pagerduty-oncall'
    pagerduty_configs:
      - routing_key: '${PAGERDUTY_KEY}'
        description: '{{ .GroupLabels.alertname }}'
        severity: '{{ if eq .GroupLabels.severity "critical" }}critical{{ else }}error{{ end }}'

  # Webhook receiver
  - name: 'custom-webhook'
    webhook_configs:
      - url: 'https://api.example.com/alerts'
        send_resolved: true
        http_config:
          bearer_token: '${API_TOKEN}'
```

#### Common Receiver Errors

```yaml
# ERROR E110: No receivers defined
receivers: []

# ERROR E111: Missing receiver name
receivers:
  - email_configs:
      - to: 'team@example.com'

# ERROR E112: Duplicate receiver names
receivers:
  - name: 'default'
  - name: 'default'  # Duplicate

# ERROR E113: Missing webhook URL
receivers:
  - name: 'webhook'
    webhook_configs:
      - send_resolved: true
        # Missing url field

# ERROR E115: Missing email recipient
receivers:
  - name: 'email'
    email_configs:
      - from: 'alerts@example.com'
        # Missing to field

# ERROR E116: Invalid email format
receivers:
  - name: 'email'
    email_configs:
      - to: 'invalid-email'  # Not a valid email

# ERROR E120: Missing Slack API URL
receivers:
  - name: 'slack'
    slack_configs:
      - channel: '#alerts'
        # Missing api_url

# WARNING W100: Receiver without integrations
receivers:
  - name: 'empty-receiver'
    # No integrations defined
```

### Inhibition Rules

#### Valid Inhibition Configuration

```yaml
inhibit_rules:
  # Inhibit warning alerts when critical alerts are firing
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="warning"
    equal:
      - alertname
      - cluster
      - service

  # Inhibit node alerts when cluster is down
  - source_matchers:
      - alertname="ClusterDown"
    target_matchers:
      - alertname=~"Node.*"
    equal:
      - cluster

  # Inhibit instance alerts during maintenance
  - source_matchers:
      - alertname="MaintenanceMode"
    target_matchers:
      - severity=~"warning|info"
    equal:
      - instance
```

#### Common Inhibition Errors

```yaml
# ERROR E150: Missing source_matchers
inhibit_rules:
  - target_matchers:
      - severity="warning"
    equal:
      - alertname

# ERROR E151: Missing target_matchers
inhibit_rules:
  - source_matchers:
      - severity="critical"
    equal:
      - alertname

# ERROR E152: Invalid matcher syntax
inhibit_rules:
  - source_matchers:
      - 'invalid matcher'
    target_matchers:
      - severity="warning"

# WARNING W150: No equal labels
inhibit_rules:
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="warning"
    # Missing equal field - may inhibit unrelated alerts

# WARNING W151: Overlapping source and target
inhibit_rules:
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="critical"  # Same as source
    equal:
      - alertname
```

### Security Checks

#### Security Best Practices

```yaml
global:
  # Use environment variables for secrets
  smtp_auth_password: '${SMTP_PASSWORD}'

receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - routing_key: '${PAGERDUTY_KEY}'  # Good: env var

  - name: 'slack'
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'  # Good: env var
        channel: '#alerts'

  - name: 'webhook'
    webhook_configs:
      - url: 'https://api.example.com/alerts'  # Good: HTTPS
        http_config:
          bearer_token: '${API_TOKEN}'  # Good: env var
          tls_config:
            insecure_skip_verify: false  # Good: verify TLS
```

#### Common Security Issues

```yaml
# WARNING W300: Hardcoded secret
global:
  smtp_auth_password: 'hardcoded-password'  # Bad: hardcoded

# WARNING W301: Insecure protocol
receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'http://api.example.com/alerts'  # Bad: HTTP

# WARNING W311: Insecure TLS configuration
receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'https://api.example.com/alerts'
        http_config:
          tls_config:
            insecure_skip_verify: true  # Bad: skip TLS verification

# WARNING W302: Exposed PagerDuty key
receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - routing_key: 'abc123def456'  # Bad: hardcoded key
```

---

## Error Handling Patterns

### Graceful Error Handling

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func validateWithRecovery(filename string) (exitCode int) {
    // Setup panic recovery
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Panic during validation: %v\n", r)
            exitCode = 2
        }
    }()

    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    result, err := validator.ValidateFile(filename)

    // Handle different error types
    if err != nil {
        var pathErr *os.PathError
        if errors.As(err, &pathErr) {
            fmt.Printf("File error: %v\n", pathErr)
            return 2
        }

        fmt.Printf("Validation error: %v\n", err)
        return 2
    }

    return result.ExitCode()
}
```

### Retry Logic

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func validateWithRetry(filename string, maxRetries int) (*types.Result, error) {
    opts := types.DefaultOptions()
    validator := configvalidator.New(opts)

    var lastErr error
    for i := 0; i < maxRetries; i++ {
        result, err := validator.ValidateFile(filename)
        if err == nil {
            return result, nil
        }

        lastErr = err
        if i < maxRetries-1 {
            fmt.Printf("Retry %d/%d after error: %v\n", i+1, maxRetries, err)
            time.Sleep(time.Second * time.Duration(i+1))
        }
    }

    return nil, fmt.Errorf("validation failed after %d retries: %w", maxRetries, lastErr)
}
```

---

## Performance Optimization

### Caching Validator Instance

```go
package main

import (
    "sync"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

var (
    validatorInstance *configvalidator.Validator
    validatorOnce     sync.Once
)

func getValidator() *configvalidator.Validator {
    validatorOnce.Do(func() {
        opts := types.DefaultOptions()
        validatorInstance = configvalidator.New(opts)
    })
    return validatorInstance
}

// Use in multiple goroutines
func validateMultiple(files []string) {
    validator := getValidator()

    for _, file := range files {
        result, _ := validator.ValidateFile(file)
        // Process result
    }
}
```

### Parallel Validation

```go
package main

import (
    "context"
    "runtime"
    "sync"

    "github.com/vitaliisemenov/alert-history/pkg/configvalidator"
    "github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

func validateParallel(files []string) []*types.Result {
    numWorkers := runtime.NumCPU()
    jobs := make(chan string, len(files))
    results := make([]*types.Result, len(files))

    var wg sync.WaitGroup

    // Start workers
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            opts := types.DefaultOptions()
            validator := configvalidator.New(opts)

            for file := range jobs {
                result, _ := validator.ValidateFile(file)
                // Store result
            }
        }()
    }

    // Send jobs
    for _, file := range files {
        jobs <- file
    }
    close(jobs)

    wg.Wait()
    return results
}
```

---

## Best Practices

### 1. Always Use Appropriate Validation Mode

```go
// Production: Strict mode
productionOpts := types.Options{
    Mode: types.StrictMode,
    EnableBestPractices: true,
    EnableSecurityChecks: true,
}

// Development: Lenient mode
devOpts := types.Options{
    Mode: types.LenientMode,
    EnableBestPractices: false,
}
```

### 2. Enable Security Checks for Production

```yaml
# Always validate with security checks before deployment
configvalidator validate --security --strict production-config.yml
```

### 3. Handle All Issue Types

```go
result, _ := validator.ValidateFile("config.yml")

// Check errors (blocking)
if len(result.Errors) > 0 {
    // Handle errors
}

// Check warnings (should review)
if len(result.Warnings) > 0 {
    // Log warnings
}

// Check suggestions (improvements)
if len(result.Suggestions) > 0 {
    // Consider applying suggestions
}
```

### 4. Use Environment Variables for Secrets

```yaml
# Good: Use environment variables
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'

# Bad: Hardcoded secrets
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/T00/B00/XX'
```

### 5. Validate in CI/CD Pipeline

```bash
# Add to your CI/CD pipeline
configvalidator validate --strict --security --format sarif config.yml
```

### 6. Use Appropriate Exit Codes

```bash
configvalidator validate config.yml
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "Valid"
elif [ $EXIT_CODE -eq 1 ]; then
    echo "Warnings only"
else
    echo "Errors found"
    exit 1
fi
```

### 7. Version Control Your Configurations

```bash
# Validate before committing
git add alertmanager.yml
configvalidator validate --strict alertmanager.yml && git commit -m "Update config"
```

### 8. Document Configuration Changes

```yaml
# Add comments to explain configuration decisions
route:
  receiver: 'default'
  # Group by alertname and cluster for efficient notification grouping
  group_by: ['alertname', 'cluster']
  # Wait 30s to batch alerts
  group_wait: 30s
```

### 9. Test Configuration Changes in Staging

```bash
# Validate with lenient mode in staging
configvalidator validate --mode lenient staging-config.yml

# Validate with strict mode before production
configvalidator validate --strict --security production-config.yml
```

### 10. Monitor Validation Results

```go
// Track validation metrics
result, _ := validator.ValidateFile("config.yml")

metrics.RecordValidation(
    errorCount: len(result.Errors),
    warningCount: len(result.Warnings),
    validationTime: result.ValidationTime,
)
```

---

## Conclusion

This examples document covers the most common use cases for the Alertmanager++ Config Validator. For more information, see:

- [User Guide](USER_GUIDE.md) - Comprehensive usage documentation
- [API Reference](API_REFERENCE.md) - Go API documentation
- [Error Codes](ERROR_CODES.md) - Complete error code reference

For support and feedback, please open an issue on the project repository.
