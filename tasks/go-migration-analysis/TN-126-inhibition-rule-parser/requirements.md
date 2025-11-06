# TN-126: Inhibition Rule Parser - Requirements

## Обзор

**Задача**: Реализовать парсер для inhibition rules из YAML конфигурации с полной совместимостью с Alertmanager.

**Приоритет**: HIGH (критический компонент для Модуля 2)

**Зависимости**: Нет (можно начинать параллельно)

**Блокирует**: TN-127 (Matcher Engine), TN-130 (API Endpoints)

---

## Бизнес-требования

### 1. Функциональные требования

#### FR-1: YAML Parsing
- Парсинг `inhibit_rules` из YAML конфигурации
- Поддержка Alertmanager config format
- Backwards compatibility с Alertmanager v0.25+

#### FR-2: Rule Validation
- Валидация структуры правил (source_match, target_match, equal)
- Проверка корректности regex patterns
- Валидация label names (соответствие Prometheus naming conventions)
- Detailed error messages с указанием поля и причины ошибки

#### FR-3: Hot Reload Support
- Возможность перезагрузки конфигурации без рестарта
- Atomic swap конфигурации (thread-safe)
- Validation перед применением новой конфигурации

#### FR-4: Config Sources
- Парсинг из файла (ParseFile)
- Парсинг из байтов (Parse)
- Парсинг из строки (ParseString)

### 2. Non-Functional Requirements

#### NFR-1: Performance
- **Target**: Парсинг одного правила < 10µs
- **Justification**: Config reload должен быть быстрым, не блокировать обработку алертов
- **Measurement**: Benchmarks в _test.go файлах

#### NFR-2: Test Coverage
- **Target**: 90%+ test coverage
- **Justification**: Парсинг критичен - ошибки могут привести к неправильному inhibition поведению
- **Measurement**: `go test -cover`

#### NFR-3: Error Handling
- Detailed error messages с контекстом (field name, value, reason)
- Structured errors (custom error types)
- No panics - все ошибки должны быть recoverable

#### NFR-4: API Compatibility
- 100% совместимость с Alertmanager YAML format
- Поддержка всех полей: source_match, source_match_re, target_match, target_match_re, equal
- Graceful handling неизвестных полей (warnings, но не errors)

---

## Технические требования

### 1. Data Models

#### InhibitionRule
```yaml
source_match:
  alertname: "HighCPU"
  severity: "critical"
source_match_re:
  service: "^(api|web).*"
target_match:
  alertname: ".*"
target_match_re:
  severity: "warning|info"
equal:
  - cluster
  - namespace
```

**Структура**:
- `source_match`: map[string]string - exact label matches для source алерта
- `source_match_re`: map[string]string - regex label matches для source алерта
- `target_match`: map[string]string - exact label matches для target алерта
- `target_match_re`: map[string]string - regex label matches для target алерта
- `equal`: []string - labels, которые должны совпадать между source и target

#### InhibitionConfig
```yaml
inhibit_rules:
  - source_match:
      alertname: "NodeDown"
    target_match:
      alertname: "InstanceDown"
    equal:
      - node
  - source_match:
      severity: "critical"
    target_match_re:
      severity: "warning|info"
    equal:
      - cluster
```

### 2. Validation Rules

#### Label Name Validation
- Must match regex: `^[a-zA-Z_][a-zA-Z0-9_]*$`
- Reserved names: `__name__` (allowed, but special)
- Empty names: NOT allowed

#### Regex Pattern Validation
- Must compile with Go `regexp` package
- Invalid patterns: return ParseError с detailed message
- Empty patterns: allowed (matches empty string)

#### Rule Consistency
- At least ONE of (source_match, source_match_re) must be present
- At least ONE of (target_match, target_match_re) must be present
- `equal` can be empty (no equality checks)
- `equal` label names must be valid Prometheus label names

### 3. Error Types

```go
type ParseError struct {
    Field   string      // YAML field where error occurred
    Value   interface{} // Value that caused error
    Err     error       // Underlying error
}

type ValidationError struct {
    Field   string      // Field that failed validation
    Rule    string      // Validation rule that failed
    Message string      // Human-readable message
}

type ConfigError struct {
    Message string      // High-level error message
    Errors  []error     // Multiple validation errors
}
```

---

## Критерии приёмки

### Обязательные (Must Have)

1. **Parsing**
   - [x] Парсит валидный Alertmanager YAML без ошибок
   - [x] Возвращает structured errors для невалидного YAML
   - [x] Поддерживает все поля (source_match, source_match_re, target_match, target_match_re, equal)

2. **Validation**
   - [x] Валидирует label names (Prometheus conventions)
   - [x] Компилирует и валидирует regex patterns
   - [x] Проверяет наличие обязательных полей
   - [x] Возвращает detailed error messages

3. **Performance**
   - [x] Benchmark: парсинг одного правила < 10µs
   - [x] Benchmark: парсинг 100 правил < 1ms

4. **Testing**
   - [x] Unit tests: 30+ tests covering:
     - Valid configs (happy path)
     - Invalid YAML syntax
     - Missing required fields
     - Invalid regex patterns
     - Invalid label names
     - Edge cases (empty configs, very large configs)
   - [x] Test coverage: 90%+

5. **Documentation**
   - [x] Godoc comments для всех exported types и functions
   - [x] README.md с примерами использования
   - [x] Design document (design.md)

### Желательные (Should Have)

1. **Hot Reload**
   - [ ] ParseFile с file watcher integration (будущая фича)
   - [ ] Atomic config swap mechanism

2. **Advanced Validation**
   - [ ] Warning для неиспользуемых полей
   - [ ] Suggestions для распространенных ошибок

3. **Performance**
   - [ ] Config caching (parse once, reuse)
   - [ ] Lazy regex compilation

### Опциональные (Nice to Have)

1. **Config Conversion**
   - [ ] Convert Alertmanager config → internal format
   - [ ] Export internal format → YAML

2. **Config Analysis**
   - [ ] Detect overlapping rules
   - [ ] Suggest rule optimizations

---

## Интеграционные требования

### Зависимости от других компонентов

#### TN-127 (Matcher Engine)
- **Зависимость**: Matcher использует parsed InhibitionRule
- **Interface**: InhibitionRule struct должен быть accessible
- **Contract**: Regex patterns должны быть pre-compiled

#### TN-130 (API Endpoints)
- **Зависимость**: GET /api/v2/inhibition/rules возвращает parsed rules
- **Interface**: Parser.GetLoadedRules() method
- **Contract**: Thread-safe access к loaded rules

### Используемые библиотеки

```go
import (
    "gopkg.in/yaml.v3"                    // YAML parsing
    "github.com/go-playground/validator/v10" // Validation
    "regexp"                               // Regex compilation
    "fmt"                                  // Error formatting
    "time"                                 // Timestamps
)
```

---

## Примеры использования

### Example 1: Parse from file

```go
parser := inhibition.NewParser()
config, err := parser.ParseFile("config/inhibition.yaml")
if err != nil {
    log.Fatalf("Failed to parse config: %v", err)
}

log.Printf("Loaded %d inhibition rules", len(config.Rules))
for _, rule := range config.Rules {
    log.Printf("Rule: source_match=%v, target_match=%v, equal=%v",
        rule.SourceMatch, rule.TargetMatch, rule.Equal)
}
```

### Example 2: Parse from bytes with validation

```go
yamlData := []byte(`
inhibit_rules:
  - source_match:
      alertname: "NodeDown"
    target_match:
      alertname: "InstanceDown"
    equal:
      - node
`)

parser := inhibition.NewParser()
config, err := parser.Parse(yamlData)
if err != nil {
    var parseErr *inhibition.ParseError
    if errors.As(err, &parseErr) {
        log.Printf("Parse error at field %s: %v", parseErr.Field, parseErr.Err)
    }
    return err
}

// Validate
if err := parser.Validate(config); err != nil {
    var validationErr *inhibition.ValidationError
    if errors.As(err, &validationErr) {
        log.Printf("Validation error in field %s: %s",
            validationErr.Field, validationErr.Message)
    }
    return err
}
```

### Example 3: Hot reload

```go
// Load initial config
config, _ := parser.ParseFile("config/inhibition.yaml")

// ... later, reload without restart ...
newConfig, err := parser.ParseFile("config/inhibition.yaml")
if err != nil {
    log.Printf("Failed to reload config: %v", err)
    // Keep using old config
    return
}

// Atomic swap (thread-safe)
atomic.SwapPointer(&currentConfig, unsafe.Pointer(newConfig))
log.Println("Config reloaded successfully")
```

---

## Ограничения и допущения

### Ограничения

1. **YAML Only**: Поддержка только YAML формата (не JSON, не TOML)
2. **Alertmanager Format**: Строгое следование Alertmanager format (no custom extensions)
3. **Regex Engine**: Go standard library regexp (RE2 syntax, no backreferences)

### Допущения

1. **Config Size**: Ожидается до 1000 inhibition rules max
2. **File Access**: Parser имеет read access к config файлу
3. **Memory**: Config полностью загружается в память (no streaming)

---

## Риски и митигация

### Риск 1: Performance деградация при большом количестве rules

**Вероятность**: MEDIUM
**Влияние**: HIGH
**Митигация**:
- Benchmarks для 100, 1000, 10000 rules
- Optimization: pre-compiled regex, indexed lookups
- Limit: warning при > 1000 rules

### Риск 2: Breaking changes в Alertmanager format

**Вероятность**: LOW
**Влияние**: HIGH
**Митигация**:
- Версионирование parser
- Backwards compatibility tests
- Graceful handling новых полей (warnings)

### Риск 3: Regex DoS (ReDoS)

**Вероятность**: MEDIUM
**Влияние**: MEDIUM
**Митигация**:
- Go RE2 engine не подвержен ReDoS
- Timeout для regex compilation (future)
- Validation сложности regex pattern (future)

---

## Метрики успеха

### Качество

- **Test Coverage**: 90%+ (target achieved if ≥90%)
- **Code Quality**: golangci-lint pass с zero errors
- **Documentation**: 100% exported symbols documented

### Performance

- **Parse Time**: < 10µs per rule (measured via benchmarks)
- **Memory**: < 1KB per rule (measured via memory profiling)

### Reliability

- **Error Handling**: 100% errors handled gracefully (no panics)
- **Backwards Compatibility**: 100% Alertmanager v0.25+ configs parse successfully

---

## Timeline

- **Design**: 2 часа (completed)
- **Implementation**: 6 часов
  - Data models: 1 час
  - Parser: 2 часа
  - Validator: 2 часа
  - Error handling: 1 час
- **Testing**: 4 часа
  - Unit tests: 3 часа
  - Benchmarks: 1 час
- **Documentation**: 2 часа
- **Total**: 14 часов (~2 дня)

---

## Ссылки

- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/#inhibit_rule)
- [Prometheus Label Naming](https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels)
- [Go RE2 Syntax](https://github.com/google/re2/wiki/Syntax)
- [go-playground/validator](https://github.com/go-playground/validator)

---

**Дата создания**: 2025-11-04
**Автор**: AlertHistory Team
**Версия**: 1.0
**Статус**: READY FOR IMPLEMENTATION
