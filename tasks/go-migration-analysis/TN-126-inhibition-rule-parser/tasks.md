# TN-126: Inhibition Rule Parser - Implementation Tasks

## Общая информация

**Задача**: TN-126 - Inhibition Rule Parser
**Статус**: 🚧 IN PROGRESS
**Приоритет**: HIGH
**Оценка**: 14 часов (~2 дня)
**Дата начала**: 2025-11-04

---

## Checklist (24 задачи)

### Phase 1: Setup & Documentation ✅

- [x] **Task 1.1**: Создать директорию `tasks/go-migration-analysis/TN-126-inhibition-rule-parser/`
- [x] **Task 1.2**: Написать `requirements.md` (technical requirements, acceptance criteria)
- [x] **Task 1.3**: Написать `design.md` (architecture, data models, interfaces)
- [x] **Task 1.4**: Написать `tasks.md` (implementation checklist)

**Статус Phase 1**: ✅ COMPLETE (4/4 tasks)

---

### Phase 2: Data Models

- [ ] **Task 2.1**: Создать файл `go-app/internal/infrastructure/inhibition/models.go`
  - [ ] InhibitionRule struct с полями:
    - SourceMatch map[string]string
    - SourceMatchRE map[string]string
    - TargetMatch map[string]string
    - TargetMatchRE map[string]string
    - Equal []string
    - Name string (optional)
    - compiledSourceRE map[string]*regexp.Regexp (internal)
    - compiledTargetRE map[string]*regexp.Regexp (internal)
    - CreatedAt time.Time
    - Version int
  - [ ] YAML tags для всех полей
  - [ ] JSON tags для всех полей
  - [ ] Godoc comments

- [ ] **Task 2.2**: Добавить методы к InhibitionRule
  - [ ] Validate() error - валидация правила
  - [ ] GetCompiledSourceRE(key string) *regexp.Regexp
  - [ ] GetCompiledTargetRE(key string) *regexp.Regexp
  - [ ] String() string - для debugging

- [ ] **Task 2.3**: Создать InhibitionConfig struct в `models.go`
  - [ ] Rules []InhibitionRule
  - [ ] LoadedAt time.Time
  - [ ] SourceFile string
  - [ ] Validate() error
  - [ ] RuleCount() int

**Статус Phase 2**: ⏳ PENDING (0/3 tasks)
**Estimated**: 1 час

---

### Phase 3: Error Types

- [ ] **Task 3.1**: Создать файл `go-app/internal/infrastructure/inhibition/errors.go`
  - [ ] ParseError struct:
    - Field string
    - Value interface{}
    - Err error
    - Error() string method
    - Unwrap() error method
  - [ ] ValidationError struct:
    - Field string
    - Rule string
    - Message string
    - Error() string method
  - [ ] ConfigError struct:
    - Message string
    - Errors []error
    - Error() string method
    - Unwrap() []error method

- [ ] **Task 3.2**: Добавить error constructors
  - [ ] NewParseError(field string, value interface{}, err error) *ParseError
  - [ ] NewValidationError(field, rule, message string) *ValidationError
  - [ ] NewConfigError(message string, errors []error) *ConfigError

**Статус Phase 3**: ⏳ PENDING (0/2 tasks)
**Estimated**: 30 минут

---

### Phase 4: Validation Helpers

- [ ] **Task 4.1**: Создать файл `go-app/internal/infrastructure/inhibition/validation.go`
  - [ ] labelNameRE *regexp.Regexp (compiled pattern)
  - [ ] isValidLabelName(name string) bool
  - [ ] validateLabelNameTag(fl validator.FieldLevel) bool
  - [ ] validateRegexPatternTag(fl validator.FieldLevel) bool

- [ ] **Task 4.2**: Добавить validation helpers
  - [ ] convertValidatorErrors(errs validator.ValidationErrors) error
  - [ ] validateEqual(equal []string) error
  - [ ] validateMatchers(matchers map[string]string) error

**Статус Phase 4**: ⏳ PENDING (0/2 tasks)
**Estimated**: 1 час

---

### Phase 5: Parser Interface

- [ ] **Task 5.1**: Создать файл `go-app/internal/infrastructure/inhibition/parser.go`
  - [ ] InhibitionParser interface:
    - Parse(data []byte) (*InhibitionConfig, error)
    - ParseFile(path string) (*InhibitionConfig, error)
    - ParseString(yaml string) (*InhibitionConfig, error)
    - ParseReader(r io.Reader) (*InhibitionConfig, error)
    - Validate(config *InhibitionConfig) error
  - [ ] Godoc comments с примерами

**Статус Phase 5**: ⏳ PENDING (0/1 tasks)
**Estimated**: 30 минут

---

### Phase 6: Parser Implementation

- [ ] **Task 6.1**: Реализовать DefaultInhibitionParser struct в `parser.go`
  - [ ] validator *validator.Validate field
  - [ ] NewParser() *DefaultInhibitionParser constructor
  - [ ] Register custom validators (labelname, regex_pattern)

- [ ] **Task 6.2**: Реализовать Parse() method
  - [ ] YAML unmarshal
  - [ ] Apply defaults
  - [ ] Struct validation
  - [ ] Compile regex patterns
  - [ ] Semantic validation
  - [ ] Set metadata

- [ ] **Task 6.3**: Реализовать ParseFile() method
  - [ ] os.ReadFile
  - [ ] Call Parse()
  - [ ] Set SourceFile

- [ ] **Task 6.4**: Реализовать ParseString() method
  - [ ] Convert string to []byte
  - [ ] Call Parse()

- [ ] **Task 6.5**: Реализовать ParseReader() method
  - [ ] io.ReadAll
  - [ ] Call Parse()

- [ ] **Task 6.6**: Реализовать Validate() method
  - [ ] Nil check
  - [ ] Struct validation
  - [ ] Semantic validation

- [ ] **Task 6.7**: Реализовать private helpers
  - [ ] applyDefaults(config *InhibitionConfig)
  - [ ] compileRegexPatterns(config *InhibitionConfig) error
  - [ ] validateSemantics(config *InhibitionConfig) error

**Статус Phase 6**: ⏳ PENDING (0/7 tasks)
**Estimated**: 2 часа

---

### Phase 7: Unit Tests

- [ ] **Task 7.1**: Создать файл `go-app/internal/infrastructure/inhibition/parser_test.go`
  - [ ] Test package setup
  - [ ] Helper functions (generateValidConfig, generateInvalidConfig)

- [ ] **Task 7.2**: Happy path tests (10 tests)
  - [ ] TestParse_ValidConfig - валидная конфигурация парсится
  - [ ] TestParse_MultipleRules - несколько правил
  - [ ] TestParse_AllFields - все поля заполнены
  - [ ] TestParse_MinimalRule - минимальная конфигурация
  - [ ] TestParseFile_Success - парсинг из файла
  - [ ] TestParseString_Success - парсинг из строки
  - [ ] TestParseReader_Success - парсинг из io.Reader
  - [ ] TestParse_EmptyMatchers - пустые matchers (valid)
  - [ ] TestParse_RegexPatterns - regex patterns компилируются
  - [ ] TestParse_EqualLabels - equal labels валидируются

- [ ] **Task 7.3**: Error handling tests (12 tests)
  - [ ] TestParse_InvalidYAML - невалидный YAML синтаксис
  - [ ] TestParse_MissingSourceMatch - missing source conditions
  - [ ] TestParse_MissingTargetMatch - missing target conditions
  - [ ] TestParse_InvalidRegex - invalid regex pattern
  - [ ] TestParse_InvalidLabelName - invalid label name в equal
  - [ ] TestParse_EmptyConfig - пустая конфигурация
  - [ ] TestParseFile_FileNotFound - файл не найден
  - [ ] TestParseFile_PermissionDenied - нет прав на чтение
  - [ ] TestValidate_NilConfig - nil config
  - [ ] TestValidate_EmptyRules - no rules
  - [ ] TestValidate_InvalidRule - правило не проходит валидацию
  - [ ] TestParse_LargeConfig - очень большая конфигурация

- [ ] **Task 7.4**: Edge cases tests (8 tests)
  - [ ] TestParse_UnicodeLabels - Unicode в label names
  - [ ] TestParse_SpecialCharactersRegex - special characters в regex
  - [ ] TestParse_VeryLongLabelName - очень длинное label name
  - [ ] TestParse_DuplicateRules - дублирующиеся правила (valid)
  - [ ] TestParse_ComplexRegex - сложные regex patterns
  - [ ] TestParse_ReservedLabelNames - __name__ и другие reserved
  - [ ] TestParse_CaseSensitivity - case sensitivity в label names
  - [ ] TestParse_WhitespaceHandling - whitespace в label values

**Статус Phase 7**: ⏳ PENDING (0/4 tasks)
**Estimated**: 3 часа
**Expected**: 30+ tests

---

### Phase 8: Benchmarks

- [ ] **Task 8.1**: Создать benchmarks в `parser_test.go`
  - [ ] BenchmarkParse_SingleRule
    - Target: < 10µs
  - [ ] BenchmarkParse_10Rules
    - Target: < 100µs
  - [ ] BenchmarkParse_100Rules
    - Target: < 1ms
  - [ ] BenchmarkParse_1000Rules
    - Target: < 10ms
  - [ ] BenchmarkParseFile_SingleRule
  - [ ] BenchmarkValidate_100Rules
  - [ ] BenchmarkCompileRegex_10Patterns
  - [ ] BenchmarkIsValidLabelName

**Статус Phase 8**: ⏳ PENDING (0/1 tasks)
**Estimated**: 1 час

---

### Phase 9: Documentation

- [ ] **Task 9.1**: Добавить Godoc comments
  - [ ] Package-level comment в `parser.go`
  - [ ] Все exported types
  - [ ] Все exported functions
  - [ ] Examples в comments

- [ ] **Task 9.2**: Создать README.md в `inhibition/` директории
  - [ ] Overview
  - [ ] Quick start examples
  - [ ] API reference
  - [ ] Configuration examples
  - [ ] Performance benchmarks results

**Статус Phase 9**: ⏳ PENDING (0/2 tasks)
**Estimated**: 2 часа

---

### Phase 10: Integration & Testing

- [ ] **Task 10.1**: Запустить все unit tests
  ```bash
  cd go-app/internal/infrastructure/inhibition
  go test -v -race -cover
  ```
  - [ ] All tests pass
  - [ ] No race conditions
  - [ ] Coverage ≥ 90%

- [ ] **Task 10.2**: Запустить benchmarks
  ```bash
  go test -bench=. -benchmem
  ```
  - [ ] Single rule < 10µs ✅
  - [ ] 100 rules < 1ms ✅

- [ ] **Task 10.3**: Запустить golangci-lint
  ```bash
  golangci-lint run internal/infrastructure/inhibition/
  ```
  - [ ] Zero errors
  - [ ] Zero warnings

- [ ] **Task 10.4**: Generate coverage report
  ```bash
  go test -coverprofile=coverage.out
  go tool cover -html=coverage.out -o coverage.html
  ```
  - [ ] Review uncovered lines
  - [ ] Add tests if needed

**Статус Phase 10**: ⏳ PENDING (0/4 tasks)
**Estimated**: 1 час

---

## Progress Summary

### По фазам

| Phase | Tasks | Completed | Status |
|-------|-------|-----------|--------|
| 1. Setup & Documentation | 4 | 4 | ✅ COMPLETE |
| 2. Data Models | 3 | 0 | ⏳ PENDING |
| 3. Error Types | 2 | 0 | ⏳ PENDING |
| 4. Validation Helpers | 2 | 0 | ⏳ PENDING |
| 5. Parser Interface | 1 | 0 | ⏳ PENDING |
| 6. Parser Implementation | 7 | 0 | ⏳ PENDING |
| 7. Unit Tests | 4 | 0 | ⏳ PENDING |
| 8. Benchmarks | 1 | 0 | ⏳ PENDING |
| 9. Documentation | 2 | 0 | ⏳ PENDING |
| 10. Integration & Testing | 4 | 0 | ⏳ PENDING |
| **TOTAL** | **30** | **4** | **13.3%** |

### По времени

- **Estimated Total**: 14 часов
- **Completed**: 1 час (Phase 1)
- **Remaining**: 13 часов

---

## Dependencies

### Блокирует
- **TN-127**: Matcher Engine (нужны InhibitionRule data models)
- **TN-130**: API Endpoints (нужен Parser для GET /rules)

### Зависимости
- Нет (можно начинать параллельно с другими задачами)

---

## Критерии приёмки

### Функциональные
- [x] Парсит валидный Alertmanager YAML
- [ ] Поддерживает все поля (source_match, source_match_re, target_match, target_match_re, equal)
- [ ] Валидирует label names (Prometheus conventions)
- [ ] Компилирует regex patterns
- [ ] Возвращает detailed error messages

### Non-Functional
- [ ] Test coverage ≥ 90%
- [ ] Performance: < 10µs per rule
- [ ] Zero panics
- [ ] golangci-lint pass

### Documentation
- [ ] Godoc 100% для exported symbols
- [ ] README с примерами
- [ ] Benchmarks results documented

---

## Риски

### Риск 1: Regex compilation может быть медленной
**Mitigation**: Pre-compile во время parsing, cache compiled patterns

### Риск 2: Test coverage может быть < 90%
**Mitigation**: Добавить edge case tests, mock сложные сценарии

### Риск 3: Alertmanager format может измениться
**Mitigation**: Версионирование, backwards compatibility tests

---

## Notes

### Ключевые решения

1. **YAML Library**: Используем `gopkg.in/yaml.v3` (standard, well-tested)
2. **Validation Library**: `go-playground/validator/v10` (rich features, custom validators)
3. **Regex Engine**: Go standard `regexp` (RE2, no ReDoS risk)

### Best Practices

1. **Thread Safety**: Parser stateless, safe для concurrent use
2. **Error Handling**: Structured errors с контекстом
3. **Performance**: Pre-compile regex, avoid allocations в hot path
4. **Testing**: Table-driven tests, comprehensive edge cases

---

**Последнее обновление**: 2025-11-04
**Автор**: AlertHistory Team
**Статус**: 🚧 IN PROGRESS
