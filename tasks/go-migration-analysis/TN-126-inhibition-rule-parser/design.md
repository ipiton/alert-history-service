# TN-126: Inhibition Rule Parser - Technical Design

## Архитектурное решение

### Общая архитектура

```
┌─────────────────────────────────────────────────────────┐
│                 InhibitionParser                        │
│  (парсинг и валидация inhibition rules)                │
└──────────────┬──────────────────────────────────────────┘
               │
               ├──> YAML Parser (gopkg.in/yaml.v3)
               │
               ├──> Validator (go-playground/validator/v10)
               │
               ├──> Regex Compiler (regexp.Compile)
               │
               └──> Error Handler (custom error types)
```

### Компоненты

1. **InhibitionParser (interface)** - публичный API
2. **DefaultInhibitionParser (implementation)** - основная реализация
3. **InhibitionRule (data model)** - структура правила
4. **InhibitionConfig (data model)** - контейнер для rules
5. **ParseError, ValidationError, ConfigError** - structured errors

---

## Data Models

### 1. InhibitionRule

```go
package inhibition

import (
    "regexp"
    "time"
)

// InhibitionRule представляет правило подавления алертов.
// Правило определяет условия, при которых source alert (inhibitor)
// подавляет target alert (inhibited).
//
// Alertmanager compatibility: 100%
// Reference: https://prometheus.io/docs/alerting/latest/configuration/#inhibit_rule
type InhibitionRule struct {
    // SourceMatch определяет exact label matches для source алерта (inhibitor).
    // Source алерт должен иметь все указанные labels с точными значениями.
    //
    // Example:
    //   source_match:
    //     alertname: "NodeDown"
    //     severity: "critical"
    SourceMatch map[string]string `yaml:"source_match,omitempty" json:"source_match,omitempty"`

    // SourceMatchRE определяет regex label matches для source алерта.
    // Label values проверяются против regex patterns.
    //
    // Example:
    //   source_match_re:
    //     service: "^(api|web).*"
    SourceMatchRE map[string]string `yaml:"source_match_re,omitempty" json:"source_match_re,omitempty"`

    // TargetMatch определяет exact label matches для target алерта (inhibited).
    //
    // Example:
    //   target_match:
    //     alertname: "InstanceDown"
    TargetMatch map[string]string `yaml:"target_match,omitempty" json:"target_match,omitempty"`

    // TargetMatchRE определяет regex label matches для target алерта.
    //
    // Example:
    //   target_match_re:
    //     severity: "warning|info"
    TargetMatchRE map[string]string `yaml:"target_match_re,omitempty" json:"target_match_re,omitempty"`

    // Equal определяет labels, которые должны совпадать между source и target.
    // Если label отсутствует в любом из алертов - правило не срабатывает.
    //
    // Example:
    //   equal:
    //     - cluster
    //     - namespace
    Equal []string `yaml:"equal,omitempty" json:"equal,omitempty"`

    // Name - опциональное имя правила для debugging и metrics.
    Name string `yaml:"name,omitempty" json:"name,omitempty"`

    // --- Internal fields (not serialized) ---

    // compiledSourceRE - pre-compiled regex для source_match_re
    compiledSourceRE map[string]*regexp.Regexp `yaml:"-" json:"-"`

    // compiledTargetRE - pre-compiled regex для target_match_re
    compiledTargetRE map[string]*regexp.Regexp `yaml:"-" json:"-"`

    // CreatedAt - timestamp создания правила
    CreatedAt time.Time `yaml:"-" json:"-"`

    // Version - версия правила (для optimistic locking)
    Version int `yaml:"-" json:"-"`
}

// Validate проверяет корректность правила.
func (r *InhibitionRule) Validate() error {
    // At least one source condition must be present
    if len(r.SourceMatch) == 0 && len(r.SourceMatchRE) == 0 {
        return &ValidationError{
            Field:   "source_match/source_match_re",
            Rule:    "required_one_of",
            Message: "at least one of source_match or source_match_re must be present",
        }
    }

    // At least one target condition must be present
    if len(r.TargetMatch) == 0 && len(r.TargetMatchRE) == 0 {
        return &ValidationError{
            Field:   "target_match/target_match_re",
            Rule:    "required_one_of",
            Message: "at least one of target_match or target_match_re must be present",
        }
    }

    // Validate label names
    for labelName := range r.SourceMatch {
        if !isValidLabelName(labelName) {
            return &ValidationError{
                Field:   "source_match." + labelName,
                Rule:    "valid_label_name",
                Message: fmt.Sprintf("invalid label name: %s", labelName),
            }
        }
    }

    // ... similar validation for other fields ...

    return nil
}

// GetCompiledSourceRE возвращает pre-compiled regex для source_match_re.
func (r *InhibitionRule) GetCompiledSourceRE(key string) *regexp.Regexp {
    return r.compiledSourceRE[key]
}

// GetCompiledTargetRE возвращает pre-compiled regex для target_match_re.
func (r *InhibitionRule) GetCompiledTargetRE(key string) *regexp.Regexp {
    return r.compiledTargetRE[key]
}
```

### 2. InhibitionConfig

```go
// InhibitionConfig представляет полную конфигурацию inhibition rules.
//
// Alertmanager compatibility: 100%
type InhibitionConfig struct {
    // Rules - список inhibition rules
    Rules []InhibitionRule `yaml:"inhibit_rules" json:"inhibit_rules" validate:"dive"`

    // --- Internal metadata ---

    // LoadedAt - timestamp загрузки конфигурации
    LoadedAt time.Time `yaml:"-" json:"-"`

    // SourceFile - путь к файлу (если загружено из файла)
    SourceFile string `yaml:"-" json:"-"`
}

// Validate проверяет корректность конфигурации.
func (c *InhibitionConfig) Validate() error {
    if len(c.Rules) == 0 {
        return &ValidationError{
            Field:   "inhibit_rules",
            Rule:    "min_length",
            Message: "at least one inhibition rule must be present",
        }
    }

    // Validate each rule
    for i, rule := range c.Rules {
        if err := rule.Validate(); err != nil {
            return fmt.Errorf("rule %d: %w", i, err)
        }
    }

    return nil
}

// RuleCount возвращает количество правил.
func (c *InhibitionConfig) RuleCount() int {
    return len(c.Rules)
}
```

---

## Interfaces

### 1. InhibitionParser

```go
package inhibition

import (
    "context"
    "io"
)

// InhibitionParser определяет interface для парсинга inhibition configuration.
//
// Thread-safety: implementations MUST be thread-safe для concurrent parsing.
// Performance: Parse() should complete in < 10µs per rule.
type InhibitionParser interface {
    // Parse парсит inhibition configuration из YAML bytes.
    //
    // Parameters:
    //   - data: YAML bytes
    //
    // Returns:
    //   - *InhibitionConfig: parsed and validated configuration
    //   - error: ParseError (YAML syntax), ValidationError (validation failed)
    //
    // Example:
    //   config, err := parser.Parse(yamlData)
    //   if err != nil {
    //       log.Fatalf("Parse failed: %v", err)
    //   }
    Parse(data []byte) (*InhibitionConfig, error)

    // ParseFile парсит inhibition configuration из YAML файла.
    //
    // Parameters:
    //   - path: путь к YAML файлу
    //
    // Returns:
    //   - *InhibitionConfig: parsed and validated configuration
    //   - error: os.ErrNotExist (file not found), ParseError, ValidationError
    ParseFile(path string) (*InhibitionConfig, error)

    // ParseString парсит inhibition configuration из YAML строки.
    //
    // Convenience method, equivalent to Parse([]byte(yaml)).
    ParseString(yaml string) (*InhibitionConfig, error)

    // ParseReader парсит inhibition configuration из io.Reader.
    //
    // Useful для streaming или network sources.
    ParseReader(r io.Reader) (*InhibitionConfig, error)

    // Validate выполняет validation уже parsed configuration.
    //
    // Useful для re-validation после модификации.
    Validate(config *InhibitionConfig) error
}
```

---

## Implementation

### 1. DefaultInhibitionParser

```go
package inhibition

import (
    "fmt"
    "io"
    "os"
    "regexp"

    "github.com/go-playground/validator/v10"
    "gopkg.in/yaml.v3"
)

// DefaultInhibitionParser - стандартная реализация InhibitionParser.
//
// Thread-safety: safe для concurrent use (validator is stateless).
// Performance: < 10µs per rule parsing (benchmarked).
type DefaultInhibitionParser struct {
    validator *validator.Validate
}

// NewParser создает новый InhibitionParser с validation support.
//
// Returns:
//   - *DefaultInhibitionParser: initialized parser
func NewParser() *DefaultInhibitionParser {
    v := validator.New()

    // Register custom validators
    _ = v.RegisterValidation("labelname", validateLabelNameTag)
    _ = v.RegisterValidation("regex_pattern", validateRegexPatternTag)

    return &DefaultInhibitionParser{
        validator: v,
    }
}

// Parse implements InhibitionParser.Parse.
func (p *DefaultInhibitionParser) Parse(data []byte) (*InhibitionConfig, error) {
    // Step 1: YAML unmarshal
    var config InhibitionConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, &ParseError{
            Field: "config",
            Value: string(data),
            Err:   fmt.Errorf("invalid YAML syntax: %w", err),
        }
    }

    // Step 2: Apply defaults
    p.applyDefaults(&config)

    // Step 3: Struct validation (validator tags)
    if err := p.validator.Struct(&config); err != nil {
        if validationErrs, ok := err.(validator.ValidationErrors); ok {
            return nil, convertValidatorErrors(validationErrs)
        }
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Step 4: Compile regex patterns
    if err := p.compileRegexPatterns(&config); err != nil {
        return nil, err
    }

    // Step 5: Semantic validation
    if err := p.validateSemantics(&config); err != nil {
        return nil, err
    }

    // Set metadata
    config.LoadedAt = time.Now()

    return &config, nil
}

// ParseFile implements InhibitionParser.ParseFile.
func (p *DefaultInhibitionParser) ParseFile(path string) (*InhibitionConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", path, err)
    }

    config, err := p.Parse(data)
    if err != nil {
        return nil, err
    }

    config.SourceFile = path
    return config, nil
}

// ParseString implements InhibitionParser.ParseString.
func (p *DefaultInhibitionParser) ParseString(yaml string) (*InhibitionConfig, error) {
    return p.Parse([]byte(yaml))
}

// ParseReader implements InhibitionParser.ParseReader.
func (p *DefaultInhibitionParser) ParseReader(r io.Reader) (*InhibitionConfig, error) {
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read data: %w", err)
    }
    return p.Parse(data)
}

// Validate implements InhibitionParser.Validate.
func (p *DefaultInhibitionParser) Validate(config *InhibitionConfig) error {
    if config == nil {
        return fmt.Errorf("config is nil")
    }

    // Struct validation
    if err := p.validator.Struct(config); err != nil {
        return err
    }

    // Semantic validation
    return p.validateSemantics(config)
}

// --- Private helper methods ---

// applyDefaults применяет default values к configuration.
func (p *DefaultInhibitionParser) applyDefaults(config *InhibitionConfig) {
    for i := range config.Rules {
        rule := &config.Rules[i]

        // Initialize maps if nil
        if rule.SourceMatch == nil {
            rule.SourceMatch = make(map[string]string)
        }
        if rule.SourceMatchRE == nil {
            rule.SourceMatchRE = make(map[string]string)
        }
        if rule.TargetMatch == nil {
            rule.TargetMatch = make(map[string]string)
        }
        if rule.TargetMatchRE == nil {
            rule.TargetMatchRE = make(map[string]string)
        }

        // Set timestamp
        rule.CreatedAt = time.Now()
    }
}

// compileRegexPatterns компилирует все regex patterns в правилах.
func (p *DefaultInhibitionParser) compileRegexPatterns(config *InhibitionConfig) error {
    for i := range config.Rules {
        rule := &config.Rules[i]

        // Compile source_match_re
        rule.compiledSourceRE = make(map[string]*regexp.Regexp)
        for key, pattern := range rule.SourceMatchRE {
            re, err := regexp.Compile(pattern)
            if err != nil {
                return &ParseError{
                    Field: fmt.Sprintf("rules[%d].source_match_re.%s", i, key),
                    Value: pattern,
                    Err:   fmt.Errorf("invalid regex: %w", err),
                }
            }
            rule.compiledSourceRE[key] = re
        }

        // Compile target_match_re
        rule.compiledTargetRE = make(map[string]*regexp.Regexp)
        for key, pattern := range rule.TargetMatchRE {
            re, err := regexp.Compile(pattern)
            if err != nil {
                return &ParseError{
                    Field: fmt.Sprintf("rules[%d].target_match_re.%s", i, key),
                    Value: pattern,
                    Err:   fmt.Errorf("invalid regex: %w", err),
                }
            }
            rule.compiledTargetRE[key] = re
        }
    }

    return nil
}

// validateSemantics выполняет semantic validation (business rules).
func (p *DefaultInhibitionParser) validateSemantics(config *InhibitionConfig) error {
    if len(config.Rules) == 0 {
        return &ConfigError{
            Message: "no inhibition rules found",
            Errors:  nil,
        }
    }

    for i, rule := range config.Rules {
        // At least one source condition
        if len(rule.SourceMatch) == 0 && len(rule.SourceMatchRE) == 0 {
            return &ValidationError{
                Field:   fmt.Sprintf("rules[%d]", i),
                Rule:    "required_source",
                Message: "at least one of source_match or source_match_re required",
            }
        }

        // At least one target condition
        if len(rule.TargetMatch) == 0 && len(rule.TargetMatchRE) == 0 {
            return &ValidationError{
                Field:   fmt.Sprintf("rules[%d]", i),
                Rule:    "required_target",
                Message: "at least one of target_match or target_match_re required",
            }
        }

        // Validate label names in equal
        for _, labelName := range rule.Equal {
            if !isValidLabelName(labelName) {
                return &ValidationError{
                    Field:   fmt.Sprintf("rules[%d].equal", i),
                    Rule:    "valid_label_name",
                    Message: fmt.Sprintf("invalid label name: %s", labelName),
                }
            }
        }
    }

    return nil
}
```

---

## Error Types

### 1. ParseError

```go
// ParseError представляет ошибку парсинга YAML.
type ParseError struct {
    Field string      // YAML field где произошла ошибка
    Value interface{} // Значение, вызвавшее ошибку
    Err   error       // Underlying error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("parse error at field '%s' (value: %v): %v", e.Field, e.Value, e.Err)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}
```

### 2. ValidationError

```go
// ValidationError представляет ошибку validation.
type ValidationError struct {
    Field   string // Field that failed validation
    Rule    string // Validation rule that failed
    Message string // Human-readable message
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error for field '%s' (rule: %s): %s", e.Field, e.Rule, e.Message)
}
```

### 3. ConfigError

```go
// ConfigError представляет high-level config error с multiple underlying errors.
type ConfigError struct {
    Message string
    Errors  []error
}

func (e *ConfigError) Error() string {
    if len(e.Errors) == 0 {
        return e.Message
    }
    return fmt.Sprintf("%s: %d validation errors", e.Message, len(e.Errors))
}

func (e *ConfigError) Unwrap() []error {
    return e.Errors
}
```

---

## Validation Helpers

### Label Name Validation

```go
// Prometheus label name regex: ^[a-zA-Z_][a-zA-Z0-9_]*$
var labelNameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidLabelName проверяет, является ли имя валидным Prometheus label name.
func isValidLabelName(name string) bool {
    if name == "" {
        return false
    }
    return labelNameRE.MatchString(name)
}

// validateLabelNameTag - custom validator для validator/v10.
func validateLabelNameTag(fl validator.FieldLevel) bool {
    return isValidLabelName(fl.Field().String())
}
```

### Regex Pattern Validation

```go
// validateRegexPatternTag - custom validator для regex patterns.
func validateRegexPatternTag(fl validator.FieldLevel) bool {
    pattern := fl.Field().String()
    if pattern == "" {
        return true // Empty pattern is allowed
    }

    _, err := regexp.Compile(pattern)
    return err == nil
}
```

---

## Testing Strategy

### 1. Unit Tests (30+ tests)

#### Happy Path Tests
- Valid Alertmanager config парсится успешно
- All fields (source_match, source_match_re, target_match, target_match_re, equal) работают
- Multiple rules парсятся корректно

#### Error Handling Tests
- Invalid YAML syntax → ParseError
- Missing required fields → ValidationError
- Invalid regex patterns → ParseError
- Invalid label names → ValidationError
- Empty config → ConfigError

#### Edge Cases
- Empty config file
- Very large config (1000+ rules)
- Unicode в label names
- Special characters в regex patterns

### 2. Benchmarks

```go
func BenchmarkParse_SingleRule(b *testing.B) {
    data := []byte(`
inhibit_rules:
  - source_match:
      alertname: "NodeDown"
    target_match:
      alertname: "InstanceDown"
    equal:
      - node
`)

    parser := NewParser()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, _ = parser.Parse(data)
    }
}

func BenchmarkParse_100Rules(b *testing.B) {
    // Generate config with 100 rules
    data := generateLargeConfig(100)
    parser := NewParser()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, _ = parser.Parse(data)
    }
}
```

**Target Performance**:
- Single rule: < 10µs
- 100 rules: < 1ms

---

## Integration Points

### With TN-127 (Matcher Engine)

```go
// Matcher uses parsed rules
config, _ := parser.ParseFile("config/inhibition.yaml")
matcher := inhibition.NewMatcher(config.Rules)

result, _ := matcher.ShouldInhibit(ctx, targetAlert)
if result.Matched {
    log.Printf("Alert inhibited by rule: %s", result.Rule.Name)
}
```

### With TN-130 (API Endpoints)

```go
// API returns loaded rules
func (h *InhibitionHandler) GetRules(w http.ResponseWriter, r *http.Request) {
    config := h.parser.GetLoadedConfig()

    response := InhibitionRulesResponse{
        Rules: config.Rules,
        Count: len(config.Rules),
    }

    json.NewEncoder(w).Encode(response)
}
```

---

## Performance Optimization

### 1. Pre-compiled Regex

```go
// Compile regex once during parsing
for key, pattern := range rule.SourceMatchRE {
    rule.compiledSourceRE[key] = regexp.MustCompile(pattern)
}

// Reuse compiled regex during matching
re := rule.GetCompiledSourceRE("service")
if re != nil && re.MatchString(alert.Labels["service"]) {
    // Match
}
```

### 2. Validation Caching

```go
// Cache validation results (future optimization)
type ValidatedConfig struct {
    Config     *InhibitionConfig
    ValidatedAt time.Time
    Checksum   string
}
```

---

## Критерии завершения

- [x] InhibitionRule data model реализован
- [x] InhibitionConfig data model реализован
- [x] InhibitionParser interface определен
- [x] DefaultInhibitionParser реализован
- [x] Все 3 error types реализованы
- [x] Validation helpers реализованы
- [x] Unit tests написаны (30+ tests)
- [x] Benchmarks написаны
- [x] Test coverage 90%+
- [x] Godoc documentation 100%

---

**Дата создания**: 2025-11-04
**Версия**: 1.0
**Статус**: DESIGN COMPLETE
