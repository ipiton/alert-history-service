# TN-121: Grouping Configuration Parser - Design Document

## Архитектурное решение

### Общая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    Config File (YAML)                       │
│  route:                                                     │
│    group_by: ['alertname', 'cluster']                      │
│    group_wait: 30s                                          │
│    group_interval: 5m                                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   YAML Parser (gopkg.in/yaml.v3)           │
│  - Unmarshal YAML → Go structs                             │
│  - Preserve line numbers для ошибок                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Structural Validator                      │
│  - Required fields check                                    │
│  - Type validation                                          │
│  - validator/v10 tags                                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Semantic Validator                        │
│  - Label name validation (regex)                            │
│  - Timer range validation                                   │
│  - Special values handling ('...')                          │
│  - Cross-field validation                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│              GroupingConfig (validated struct)              │
│  Ready для использования в Group Manager                    │
└─────────────────────────────────────────────────────────────┘
```

### Компонентная архитектура

#### Package Structure
```
go-app/internal/infrastructure/grouping/
├── config.go           // Структуры данных
├── parser.go           // YAML parser
├── validator.go        // Validation logic
├── errors.go           // Error types
├── config_test.go      // Tests для структур
├── parser_test.go      // Tests для парсера
├── validator_test.go   // Tests для валидатора
└── README.md           // Package documentation

go-app/config/examples/
└── alertmanager-grouping.yaml  // Example config
```

## Структуры данных

### Core Structures

```go
package grouping

import (
	"time"
	"regexp"
)

// GroupingConfig представляет полную конфигурацию группировки алертов.
// Совместимо с Alertmanager route configuration.
type GroupingConfig struct {
	Route *Route `yaml:"route" validate:"required"`
}

// Route определяет маршрут с параметрами группировки.
// Может содержать вложенные routes для иерархической маршрутизации.
type Route struct {
	// Receiver - имя получателя нотификаций
	Receiver string `yaml:"receiver" validate:"required"`

	// GroupBy - список label names для группировки
	// Special value: ['...'] означает группировку по всем labels
	GroupBy []string `yaml:"group_by" validate:"required,min=1"`

	// GroupWait - задержка перед первой отправкой группы
	// Позволяет накопить алерты в группе
	GroupWait *Duration `yaml:"group_wait,omitempty" validate:"omitempty,gte=0,lte=3600s"`

	// GroupInterval - интервал между обновлениями группы
	// Как часто отправлять обновления для активной группы
	GroupInterval *Duration `yaml:"group_interval,omitempty" validate:"omitempty,gte=1s,lte=86400s"`

	// RepeatInterval - интервал повторной нотификации
	// Как часто повторять нотификацию для долгоживущих алертов
	RepeatInterval *Duration `yaml:"repeat_interval,omitempty" validate:"omitempty,gte=60s,lte=604800s"`

	// Match - точное совпадение labels для routing
	Match map[string]string `yaml:"match,omitempty"`

	// MatchRE - regex совпадение labels для routing
	MatchRE map[string]string `yaml:"match_re,omitempty"`

	// Continue - продолжать поиск по route tree после match
	Continue bool `yaml:"continue,omitempty"`

	// Routes - вложенные routes (иерархия)
	Routes []*Route `yaml:"routes,omitempty"`

	// Metadata для debugging и hot reload
	parsedAt time.Time
	source   string // Путь к конфиг файлу
}

// Duration - обёртка над time.Duration для YAML parsing
type Duration struct {
	time.Duration
}

// UnmarshalYAML парсит duration из YAML (например "30s", "5m", "1h")
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return &ParseError{
			Field: "duration",
			Value: s,
			Err:   err,
		}
	}

	d.Duration = dur
	return nil
}

// MarshalYAML сериализует duration в YAML
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

// Defaults возвращает default значения для route parameters
func (r *Route) Defaults() {
	if r.GroupWait == nil {
		r.GroupWait = &Duration{30 * time.Second}
	}
	if r.GroupInterval == nil {
		r.GroupInterval = &Duration{5 * time.Minute}
	}
	if r.RepeatInterval == nil {
		r.RepeatInterval = &Duration{4 * time.Hour}
	}
}

// HasSpecialGrouping проверяет, используется ли special value '...'
func (r *Route) HasSpecialGrouping() bool {
	return len(r.GroupBy) == 1 && r.GroupBy[0] == "..."
}

// IsGlobalGroup проверяет, настроена ли одна глобальная группа
func (r *Route) IsGlobalGroup() bool {
	return len(r.GroupBy) == 0
}
```

### Error Types

```go
// ParseError - ошибка парсинга YAML
type ParseError struct {
	Field    string // Поле с ошибкой
	Value    string // Некорректное значение
	Line     int    // Номер строки в YAML
	Column   int    // Номер колонки в YAML
	Err      error  // Underlying error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("parse error at line %d, column %d: field '%s' with value '%s': %v",
			e.Line, e.Column, e.Field, e.Value, e.Err)
	}
	return fmt.Sprintf("parse error: field '%s' with value '%s': %v", e.Field, e.Value, e.Err)
}

// ValidationError - ошибка валидации конфигурации
type ValidationError struct {
	Field   string   // Поле с ошибкой
	Value   string   // Некорректное значение
	Rule    string   // Правило валидации
	Message string   // Human-readable сообщение
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s' failed validation '%s': %s (value: '%s')",
		e.Field, e.Rule, e.Message, e.Value)
}

// ValidationErrors - множественные ошибки валидации
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("validation failed with %d error(s):\n", len(ve)))
	for i, err := range ve {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return b.String()
}
```

## API Design

### Parser API

```go
package grouping

// Parser интерфейс для парсинга grouping конфигурации
type Parser interface {
	// Parse парсит YAML конфигурацию из bytes
	Parse(data []byte) (*GroupingConfig, error)

	// ParseFile парсит YAML конфигурацию из файла
	ParseFile(path string) (*GroupingConfig, error)

	// ParseString парсит YAML конфигурацию из строки (для тестов)
	ParseString(yaml string) (*GroupingConfig, error)
}

// DefaultParser - стандартная реализация Parser
type DefaultParser struct {
	validator *validator.Validate
}

// NewParser создаёт новый parser с validation
func NewParser() *DefaultParser {
	v := validator.New()

	// Регистрируем custom validators
	v.RegisterValidation("labelname", validateLabelName)
	v.RegisterValidation("duration_range", validateDurationRange)

	return &DefaultParser{
		validator: v,
	}
}

// Parse реализует Parser.Parse
func (p *DefaultParser) Parse(data []byte) (*GroupingConfig, error) {
	var config GroupingConfig

	// Парсим YAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, &ParseError{
			Err: err,
		}
	}

	// Применяем defaults
	applyDefaults(&config)

	// Валидация структуры
	if err := p.validator.Struct(&config); err != nil {
		return nil, convertValidationErrors(err)
	}

	// Semantic validation
	if err := p.validateSemantics(&config); err != nil {
		return nil, err
	}

	// Метаданные
	config.Route.parsedAt = time.Now()

	return &config, nil
}

// ParseFile реализует Parser.ParseFile
func (p *DefaultParser) ParseFile(path string) (*GroupingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config, err := p.Parse(data)
	if err != nil {
		return nil, err
	}

	config.Route.source = path
	return config, nil
}

// ParseString реализует Parser.ParseString
func (p *DefaultParser) ParseString(yamlStr string) (*GroupingConfig, error) {
	return p.Parse([]byte(yamlStr))
}
```

### Validator API

```go
// validateSemantics выполняет семантическую валидацию конфигурации
func (p *DefaultParser) validateSemantics(config *GroupingConfig) error {
	var errors ValidationErrors

	// Валидация route tree
	if err := p.validateRoute(config.Route, &errors); err != nil {
		return err
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateRoute рекурсивно валидирует route и nested routes
func (p *DefaultParser) validateRoute(route *Route, errors *ValidationErrors) error {
	// Валидация label names в group_by
	if !route.HasSpecialGrouping() {
		for _, label := range route.GroupBy {
			if !isValidLabelName(label) {
				*errors = append(*errors, ValidationError{
					Field:   "group_by",
					Value:   label,
					Rule:    "labelname",
					Message: fmt.Sprintf("invalid label name '%s': must match [a-zA-Z_][a-zA-Z0-9_]*", label),
				})
			}
		}
	}

	// Валидация timer ranges
	if route.GroupWait != nil {
		if route.GroupWait.Duration < 0 || route.GroupWait.Duration > time.Hour {
			*errors = append(*errors, ValidationError{
				Field:   "group_wait",
				Value:   route.GroupWait.Duration.String(),
				Rule:    "range",
				Message: "group_wait must be between 0s and 1h",
			})
		}
	}

	if route.GroupInterval != nil {
		if route.GroupInterval.Duration < time.Second || route.GroupInterval.Duration > 24*time.Hour {
			*errors = append(*errors, ValidationError{
				Field:   "group_interval",
				Value:   route.GroupInterval.Duration.String(),
				Rule:    "range",
				Message: "group_interval must be between 1s and 24h",
			})
		}
	}

	if route.RepeatInterval != nil {
		if route.RepeatInterval.Duration < time.Minute || route.RepeatInterval.Duration > 7*24*time.Hour {
			*errors = append(*errors, ValidationError{
				Field:   "repeat_interval",
				Value:   route.RepeatInterval.Duration.String(),
				Rule:    "range",
				Message: "repeat_interval must be between 1m and 168h (7 days)",
			})
		}
	}

	// Рекурсивно валидируем nested routes
	for i, nestedRoute := range route.Routes {
		if err := p.validateRoute(nestedRoute, errors); err != nil {
			return fmt.Errorf("validation failed for nested route %d: %w", i, err)
		}
	}

	return nil
}

// isValidLabelName проверяет, валидно ли имя label
func isValidLabelName(name string) bool {
	// Prometheus label name regex: [a-zA-Z_][a-zA-Z0-9_]*
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

// validateLabelName - custom validator для validator/v10
func validateLabelName(fl validator.FieldLevel) bool {
	labelName := fl.Field().String()
	return isValidLabelName(labelName)
}
```

### Helper Functions

```go
// applyDefaults применяет default значения к конфигурации
func applyDefaults(config *GroupingConfig) {
	if config.Route != nil {
		applyRouteDefaults(config.Route)
	}
}

// applyRouteDefaults рекурсивно применяет defaults к route и nested routes
func applyRouteDefaults(route *Route) {
	route.Defaults()

	for _, nestedRoute := range route.Routes {
		applyRouteDefaults(nestedRoute)
	}
}

// convertValidationErrors конвертирует validator errors в ValidationErrors
func convertValidationErrors(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var errors ValidationErrors
	for _, fieldErr := range validationErrs {
		errors = append(errors, ValidationError{
			Field:   fieldErr.Field(),
			Value:   fmt.Sprintf("%v", fieldErr.Value()),
			Rule:    fieldErr.Tag(),
			Message: getValidationMessage(fieldErr),
		})
	}

	return errors
}

// getValidationMessage возвращает user-friendly сообщение для validation error
func getValidationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "this field is required"
	case "min":
		return fmt.Sprintf("must have at least %s items", fieldErr.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fieldErr.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fieldErr.Param())
	case "labelname":
		return "must be a valid Prometheus label name"
	default:
		return fmt.Sprintf("validation failed: %s", fieldErr.Tag())
	}
}
```

## Примеры использования

### Example 1: Basic parsing

```go
package main

import (
	"fmt"
	"log"

	"alert-history/internal/infrastructure/grouping"
)

func main() {
	yamlConfig := `
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
`

	parser := grouping.NewParser()
	config, err := parser.ParseString(yamlConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed config:\n")
	fmt.Printf("  Receiver: %s\n", config.Route.Receiver)
	fmt.Printf("  Group By: %v\n", config.Route.GroupBy)
	fmt.Printf("  Group Wait: %s\n", config.Route.GroupWait.Duration)
	fmt.Printf("  Group Interval: %s\n", config.Route.GroupInterval.Duration)
}
```

### Example 2: Nested routes

```go
yamlConfig := `
route:
  receiver: 'default'
  group_by: ['alertname']
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
      group_by: ['alertname', 'instance']
      group_wait: 10s
      group_interval: 3m
    - match:
        team: frontend
      receiver: 'slack-frontend'
      group_by: ['alertname', 'service']
`

parser := grouping.NewParser()
config, err := parser.ParseString(yamlConfig)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Root route: %s\n", config.Route.Receiver)
fmt.Printf("Nested routes: %d\n", len(config.Route.Routes))

for i, route := range config.Route.Routes {
	fmt.Printf("  Route %d: receiver=%s, match=%v\n",
		i+1, route.Receiver, route.Match)
}
```

### Example 3: Error handling

```go
yamlConfig := `
route:
  receiver: 'default'
  group_by: ['alert-name']  # Invalid: dash not allowed
  group_wait: 2h            # Invalid: exceeds 1h max
`

parser := grouping.NewParser()
_, err := parser.ParseString(yamlConfig)
if err != nil {
	if validationErrs, ok := err.(grouping.ValidationErrors); ok {
		fmt.Printf("Validation failed with %d errors:\n", len(validationErrs))
		for i, verr := range validationErrs {
			fmt.Printf("  %d. %s\n", i+1, verr.Message)
		}
	} else {
		log.Fatal(err)
	}
}

// Output:
// Validation failed with 2 errors:
//   1. invalid label name 'alert-name': must match [a-zA-Z_][a-zA-Z0-9_]*
//   2. group_wait must be between 0s and 1h
```

## Performance Considerations

### Memory Usage
- Parser не держит копию raw YAML после парсинга
- Структуры оптимизированы по размеру (используем pointers для optional fields)
- Nested routes shared pointer к parent defaults

### Parsing Speed
- Target: <10ms для конфигурации <100KB
- Используем `yaml.v3` (fastest Go YAML library)
- Validation выполняется параллельно структурной и семантической

### Benchmarks
```go
func BenchmarkParser_Parse(b *testing.B) {
	yamlConfig := loadTestConfig("testdata/large_config.yaml")
	parser := grouping.NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseString(yamlConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Expected result: ~5-8ms per operation
```

## Security Considerations

### YAML Bombing
- Ограничение размера конфигурационного файла: 10MB max
- Ограничение глубины вложенности routes: 10 levels max
- Timeout для парсинга: 30s max

### Code Injection
- Не используем `yaml.Unmarshal` с `!!python/object` tags
- Strict mode для YAML парсинга
- Sanitization всех строковых значений

### Validation Bypass
- Все поля валидируются (structural + semantic)
- No reflection-based bypasses
- Immutable конфигурация после парсинга

---

**Архитектор**: DevOps Team
**Дата создания**: 2025-01-09
**Версия**: 1.0
**Статус**: Ready for Implementation
