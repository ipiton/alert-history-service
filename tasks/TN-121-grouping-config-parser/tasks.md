# TN-121: Grouping Configuration Parser - Task Checklist

## Статус: 🔄 IN PROGRESS

**Started**: 2025-01-09
**Target Completion**: 2025-01-12
**Actual Completion**: TBD

---

## Implementation Checklist

### Phase 1: Project Setup ✅
- [x] Create task directory structure
- [x] Write requirements.md
- [x] Write design.md
- [x] Write tasks.md (this file)
- [ ] Create package directory `internal/infrastructure/grouping/`
- [ ] Initialize Go package with package documentation

### Phase 2: Data Structures Implementation
- [ ] **config.go** - Core data structures
  - [ ] Define `GroupingConfig` struct
  - [ ] Define `Route` struct with all fields
  - [ ] Define `Duration` wrapper struct
  - [ ] Implement `Duration.UnmarshalYAML()`
  - [ ] Implement `Duration.MarshalYAML()`
  - [ ] Implement `Route.Defaults()` method
  - [ ] Implement `Route.HasSpecialGrouping()` method
  - [ ] Implement `Route.IsGlobalGroup()` method
  - [ ] Add struct tags (yaml, validate)
  - [ ] Add godoc comments для всех exported types

- [ ] **errors.go** - Error types
  - [ ] Define `ParseError` struct
  - [ ] Implement `ParseError.Error()` method
  - [ ] Define `ValidationError` struct
  - [ ] Implement `ValidationError.Error()` method
  - [ ] Define `ValidationErrors` type (slice)
  - [ ] Implement `ValidationErrors.Error()` method
  - [ ] Add helper constructors для errors
  - [ ] Add godoc comments

### Phase 3: Parser Implementation
- [ ] **parser.go** - YAML parser
  - [ ] Define `Parser` interface
  - [ ] Define `DefaultParser` struct
  - [ ] Implement `NewParser()` constructor
  - [ ] Implement `Parse(data []byte)` method
  - [ ] Implement `ParseFile(path string)` method
  - [ ] Implement `ParseString(yaml string)` method
  - [ ] Add YAML unmarshaling with error handling
  - [ ] Integrate with validator
  - [ ] Add godoc comments для всех methods

### Phase 4: Validator Implementation
- [ ] **validator.go** - Validation logic
  - [ ] Implement `validateSemantics()` method
  - [ ] Implement `validateRoute()` recursive validation
  - [ ] Implement `isValidLabelName()` helper
  - [ ] Implement `validateLabelName()` custom validator
  - [ ] Implement `validateDurationRange()` custom validator
  - [ ] Implement `applyDefaults()` helper
  - [ ] Implement `applyRouteDefaults()` recursive defaults
  - [ ] Implement `convertValidationErrors()` converter
  - [ ] Implement `getValidationMessage()` message formatter
  - [ ] Add comprehensive error messages
  - [ ] Add godoc comments

### Phase 5: Unit Tests
- [ ] **config_test.go** - Tests для структур
  - [ ] Test `Duration.UnmarshalYAML()` - valid durations
  - [ ] Test `Duration.UnmarshalYAML()` - invalid durations
  - [ ] Test `Duration.MarshalYAML()` - serialization
  - [ ] Test `Route.Defaults()` - default values
  - [ ] Test `Route.HasSpecialGrouping()` - special value '...'
  - [ ] Test `Route.IsGlobalGroup()` - empty group_by
  - [ ] Test struct tags validation

- [ ] **parser_test.go** - Tests для парсера
  - [ ] Test `Parse()` - valid basic config
  - [ ] Test `Parse()` - valid nested routes
  - [ ] Test `Parse()` - special grouping '...'
  - [ ] Test `Parse()` - empty group_by
  - [ ] Test `Parse()` - invalid YAML syntax
  - [ ] Test `Parse()` - missing required fields
  - [ ] Test `Parse()` - invalid field types
  - [ ] Test `ParseFile()` - existing file
  - [ ] Test `ParseFile()` - non-existent file
  - [ ] Test `ParseString()` - various configs
  - [ ] Test defaults application
  - [ ] Test metadata (parsedAt, source)

- [ ] **validator_test.go** - Tests для валидатора
  - [ ] Test label name validation - valid names
  - [ ] Test label name validation - invalid names (dash, digit start, etc.)
  - [ ] Test group_wait range - valid values
  - [ ] Test group_wait range - out of range (negative, >1h)
  - [ ] Test group_interval range - valid values
  - [ ] Test group_interval range - out of range (<1s, >24h)
  - [ ] Test repeat_interval range - valid values
  - [ ] Test repeat_interval range - out of range (<1m, >7d)
  - [ ] Test nested routes validation
  - [ ] Test multiple validation errors
  - [ ] Test error message formatting

- [ ] **errors_test.go** - Tests для error types
  - [ ] Test `ParseError.Error()` formatting
  - [ ] Test `ValidationError.Error()` formatting
  - [ ] Test `ValidationErrors.Error()` multiple errors
  - [ ] Test line/column number reporting

### Phase 6: Examples and Documentation
- [ ] **examples/** directory
  - [ ] Create `basic_grouping.yaml` example
  - [ ] Create `nested_routes.yaml` example
  - [ ] Create `special_grouping.yaml` example
  - [ ] Create `full_featured.yaml` example

- [ ] **README.md** - Package documentation
  - [ ] Overview section
  - [ ] Installation instructions
  - [ ] Usage examples
  - [ ] API reference
  - [ ] Configuration reference
  - [ ] Troubleshooting section

- [ ] **Godoc comments**
  - [ ] Package-level documentation
  - [ ] All exported types documented
  - [ ] All exported functions documented
  - [ ] Examples in godoc format

### Phase 7: Integration
- [ ] **Config loader integration**
  - [ ] Integrate с `internal/config/loader.go`
  - [ ] Add grouping config to main config struct
  - [ ] Add config file path resolution
  - [ ] Add hot reload support hook

- [ ] **CI/CD integration**
  - [ ] Add config validation в CI pipeline
  - [ ] Add linting checks
  - [ ] Add test coverage reporting
  - [ ] Add benchmarks в CI

### Phase 8: Testing and Validation
- [ ] **Manual testing**
  - [ ] Test with real Alertmanager configs
  - [ ] Test error scenarios
  - [ ] Test performance with large configs
  - [ ] Test hot reload behavior

- [ ] **Performance benchmarks**
  - [ ] Benchmark `Parse()` - small config (<1KB)
  - [ ] Benchmark `Parse()` - medium config (10KB)
  - [ ] Benchmark `Parse()` - large config (100KB)
  - [ ] Verify <10ms target для <100KB configs
  - [ ] Memory profiling

- [ ] **Coverage verification**
  - [ ] Run `go test -cover`
  - [ ] Verify >85% coverage
  - [ ] Add tests для uncovered branches
  - [ ] Generate coverage report

### Phase 9: Code Review and QA
- [ ] **Code quality**
  - [ ] Run `golangci-lint`
  - [ ] Fix all linter issues
  - [ ] Run `go vet`
  - [ ] Check for race conditions (`go test -race`)

- [ ] **Security review**
  - [ ] Review YAML parsing security
  - [ ] Check for injection vulnerabilities
  - [ ] Verify input validation
  - [ ] Test with malicious configs

- [ ] **Documentation review**
  - [ ] Review all godoc comments
  - [ ] Review README completeness
  - [ ] Review example configs
  - [ ] Spell check documentation

### Phase 10: Deployment
- [ ] **Merge and deploy**
  - [ ] Create feature branch `feature/TN-121-grouping-config-parser`
  - [ ] Push code to repository
  - [ ] Create Pull Request
  - [ ] Address review comments
  - [ ] Merge to main branch
  - [ ] Tag release `v0.1.0-TN-121`

---

## Test Coverage Goals

| Component | Target Coverage | Actual Coverage |
|-----------|----------------|-----------------|
| config.go | >85% | TBD |
| parser.go | >90% | TBD |
| validator.go | >95% | TBD |
| errors.go | >80% | TBD |
| **Total** | **>85%** | **TBD** |

---

## Performance Goals

| Metric | Target | Actual |
|--------|--------|--------|
| Parse time (1KB config) | <1ms | TBD |
| Parse time (10KB config) | <5ms | TBD |
| Parse time (100KB config) | <10ms | TBD |
| Memory allocation per parse | <500KB | TBD |

---

## Dependencies

### External Dependencies
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/go-playground/validator/v10` - Struct validation

### Internal Dependencies
- None (foundation task)

### Blocks
- TN-122 (Group Key Generator)
- TN-123 (Alert Group Manager)
- TN-124 (Group Wait/Interval Timers)
- TN-125 (Group Storage Redis Backend)

---

## Notes

### Design Decisions
1. **Используем `yaml.v3`** вместо `yaml.v2` для лучшего error reporting с line numbers
2. **Используем `validator/v10`** для декларативной валидации через struct tags
3. **Duration wrapper** для удобного парсинга временных интервалов (30s, 5m, etc.)
4. **Recursive validation** для nested routes с accumulation ошибок

### Known Limitations
1. Максимальная глубина nested routes: 10 levels (защита от рекурсии)
2. Максимальный размер config файла: 10MB (защита от YAML bombing)
3. Timeout парсинга: 30s (для очень больших конфигов)

### Future Enhancements
1. Support для JSON конфигурации (low priority)
2. Config migration tool (Alertmanager → Alert History format)
3. Visual config editor (web UI)
4. Config templates library

---

**Last Updated**: 2025-01-09
**Status**: 🔄 IN PROGRESS (Phase 1 completed)
**Next Step**: Create package directory and start Phase 2
