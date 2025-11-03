# 🚨 TN-121: СРОЧНЫЕ ACTION ITEMS

**Дата**: 2025-11-03
**Статус**: ⚠️ **ТРЕБУЕТСЯ СРОЧНОЕ ИСПРАВЛЕНИЕ**
**Текущая готовность**: 60% (вместо заявленных 100%)

---

## 🔴 КРИТИЧНЫЕ (выполнить СЕГОДНЯ)

### 1. ❌ Исправить тесты (1 минута)

**Проблема**:
```
FAIL [build failed]
internal/infrastructure/grouping/config_test.go:57:11: undefined: yaml
```

**Решение**:
```bash
cd /Users/vitaliisemenov/.cursor/worktrees/AlertHistory/7BDo8/go-app
```

Добавить в `internal/infrastructure/grouping/config_test.go`:
```go
import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3" // ✅ ДОБАВИТЬ ЭТУ СТРОКУ
)
```

**Проверка**:
```bash
go test -v ./internal/infrastructure/grouping/... -cover
```

**Ожидаемый результат**: Все тесты проходят, coverage >85%

---

### 2. ❌ Закоммитить код (10 минут)

**Команды**:
```bash
cd /Users/vitaliisemenov/.cursor/worktrees/AlertHistory/7BDo8

# Проверить статус
git status

# Добавить файлы
git add go-app/internal/infrastructure/grouping/

# Закоммитить
git commit -m "feat(go): TN-121 implement grouping config parser (60% complete)

- Add GroupingConfig, Route, Duration structs (config.go, 278 LOC)
- Add ParseError, ValidationError, ConfigError (errors.go, 208 LOC)
- Add Parser interface and DefaultParser (parser.go, 328 LOC)
- Add comprehensive validation (validator.go, 271 LOC)
- Add unit tests (config_test.go, 369 LOC)

TODO:
- Fix test build (missing yaml import)
- Add integration to main.go
- Add README.md and examples
- Add benchmarks
- Achieve >85% test coverage

Related: TN-121, PHASE-A Module 1"

# Запушить
git push origin main
```

**Проверка**:
```bash
git log --oneline -1
# Должен показать новый коммит с TN-121
```

---

### 3. ⚠️ Обновить статус в tasks.md (5 минут)

**Файл**: `/tasks/go-migration-analysis/tasks.md`

**Текущий статус** (НЕКОРРЕКТНЫЙ):
```markdown
- [x] **TN-121** Grouping Configuration Parser ✅ **ЗАВЕРШЕНА** (2025-01-09)
```

**Новый статус** (КОРРЕКТНЫЙ):
```markdown
- [x] **TN-121** Grouping Configuration Parser ⚠️ **60% COMPLETE** (2025-01-09, 1449 LOC, тесты broken, нет integration, нет git commits)
```

**Статус**: ✅ **УЖЕ ОБНОВЛЕНО** (2025-11-03)

---

## 🟡 ВАЖНЫЕ (выполнить в течение 1-2 дней)

### 4. ❌ Интегрировать в main.go (2-3 часа)

**Шаг 1**: Добавить в `internal/config/config.go`:
```go
type Config struct {
    // ... existing fields ...

    // Grouping configuration (TN-121)
    GroupingConfigPath string `mapstructure:"grouping_config_path"`
}
```

**Шаг 2**: Создать `internal/config/grouping_loader.go`:
```go
package config

import (
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
)

// LoadGroupingConfig loads and validates grouping configuration
func LoadGroupingConfig(path string) (*grouping.GroupingConfig, error) {
    parser := grouping.NewParser()
    config, err := parser.ParseFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to parse grouping config: %w", err)
    }

    // Validate Alertmanager compatibility
    if err := grouping.ValidateConfigCompat(config); err != nil {
        // Log warnings but don't fail
        log.Warn("Grouping config compatibility warnings", "error", err)
    }

    return config, nil
}
```

**Шаг 3**: Обновить `cmd/server/main.go`:
```go
func main() {
    // ... existing code ...

    // Load grouping configuration (TN-121)
    var groupingConfig *grouping.GroupingConfig
    if cfg.GroupingConfigPath != "" {
        var err error
        groupingConfig, err = config.LoadGroupingConfig(cfg.GroupingConfigPath)
        if err != nil {
            logger.Error("Failed to load grouping config", "error", err)
            // Fallback to default config
            groupingConfig = getDefaultGroupingConfig()
        } else {
            logger.Info("Loaded grouping config",
                "path", cfg.GroupingConfigPath,
                "group_by", groupingConfig.Route.GroupBy)
        }
    }

    // TODO: Pass groupingConfig to AlertGroupManager (TN-123)

    // ... rest of main ...
}

func getDefaultGroupingConfig() *grouping.GroupingConfig {
    return &grouping.GroupingConfig{
        Route: &grouping.Route{
            Receiver: "default",
            GroupBy: []string{"alertname"},
            GroupWait: &grouping.Duration{30 * time.Second},
            GroupInterval: &grouping.Duration{5 * time.Minute},
            RepeatInterval: &grouping.Duration{4 * time.Hour},
        },
    }
}
```

**Проверка**:
```bash
cd go-app
go build ./cmd/server
./server --grouping-config-path=/path/to/config.yml
```

---

### 5. ❌ Добавить integration tests (1-2 часа)

**Создать**: `internal/infrastructure/grouping/integration_test.go`

```go
package grouping_test

import (
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
)

func TestIntegration_ParseRealAlertmanagerConfig(t *testing.T) {
    // Create temporary config file
    configYAML := `
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
      group_wait: 10s
`

    tmpFile, err := os.CreateTemp("", "alertmanager-*.yml")
    require.NoError(t, err)
    defer os.Remove(tmpFile.Name())

    _, err = tmpFile.WriteString(configYAML)
    require.NoError(t, err)
    tmpFile.Close()

    // Parse config
    parser := grouping.NewParser()
    config, err := parser.ParseFile(tmpFile.Name())
    require.NoError(t, err)
    require.NotNil(t, config)

    // Validate structure
    assert.Equal(t, "default", config.Route.Receiver)
    assert.Equal(t, []string{"alertname", "cluster", "service"}, config.Route.GroupBy)
    assert.Equal(t, 30*time.Second, config.Route.GroupWait.Duration)

    // Validate nested routes
    require.Len(t, config.Route.Routes, 1)
    assert.Equal(t, "pagerduty", config.Route.Routes[0].Receiver)
    assert.Equal(t, 10*time.Second, config.Route.Routes[0].GroupWait.Duration)
}

func TestIntegration_ValidateAlertmanagerCompatibility(t *testing.T) {
    parser := grouping.NewParser()
    config, err := parser.ParseString(`
route:
  receiver: 'default'
  group_by: ['alertname']
  group_wait: 1s  # Very short - should trigger warning
`)
    require.NoError(t, err)

    // Check compatibility warnings
    err = grouping.ValidateConfigCompat(config)
    // Should not fail, but may log warnings
    assert.NoError(t, err)
}
```

**Запуск**:
```bash
go test -v ./internal/infrastructure/grouping/... -run Integration
```

---

### 6. ❌ Добавить benchmarks (1-2 часа)

**Создать**: `internal/infrastructure/grouping/parser_bench_test.go`

```go
package grouping_test

import (
    "testing"

    "github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
)

var smallConfig = `
route:
  receiver: 'default'
  group_by: ['alertname']
  group_wait: 30s
`

var mediumConfig = `
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match: {severity: critical}
      receiver: 'pagerduty'
    - match: {severity: warning}
      receiver: 'slack'
`

var largeConfig = `
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service', 'namespace', 'pod']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match: {severity: critical}
      receiver: 'pagerduty'
      routes:
        - match: {team: frontend}
          receiver: 'pagerduty-frontend'
        - match: {team: backend}
          receiver: 'pagerduty-backend'
    - match: {severity: warning}
      receiver: 'slack'
      routes:
        - match: {team: frontend}
          receiver: 'slack-frontend'
        - match: {team: backend}
          receiver: 'slack-backend'
`

func BenchmarkParse_SmallConfig(b *testing.B) {
    parser := grouping.NewParser()
    data := []byte(smallConfig)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.Parse(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkParse_MediumConfig(b *testing.B) {
    parser := grouping.NewParser()
    data := []byte(mediumConfig)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.Parse(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkParse_LargeConfig(b *testing.B) {
    parser := grouping.NewParser()
    data := []byte(largeConfig)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.Parse(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkValidateRoute(b *testing.B) {
    parser := grouping.NewParser()
    config, _ := parser.ParseString(largeConfig)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = grouping.ValidateRoute(config.Route)
    }
}
```

**Запуск**:
```bash
go test -bench=. -benchmem ./internal/infrastructure/grouping/
```

**Цели производительности**:
- Small config (1KB): <1ms
- Medium config (10KB): <5ms
- Large config (100KB): <10ms

---

### 7. ❌ Написать README.md (2 часа)

**Создать**: `internal/infrastructure/grouping/README.md`

```markdown
# Alert Grouping Configuration Parser

Package `grouping` provides Alertmanager-compatible configuration parsing for alert grouping.

## Features

- ✅ Full Alertmanager YAML format support
- ✅ Nested route configurations
- ✅ Comprehensive validation
- ✅ Special grouping modes (`...` and `[]`)
- ✅ Duration parsing (30s, 5m, 4h)
- ✅ Detailed error messages with line/column numbers

## Installation

```go
import "github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
```

## Quick Start

```go
// Parse from file
parser := grouping.NewParser()
config, err := parser.ParseFile("/etc/alertmanager/config.yml")
if err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Printf("Group by: %v\n", config.Route.GroupBy)
fmt.Printf("Group wait: %s\n", config.Route.GetEffectiveGroupWait())
```

## Configuration Format

### Basic Example

```yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
```

### Nested Routes

```yaml
route:
  receiver: 'default'
  group_by: ['alertname']
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
      group_wait: 10s
    - match:
        severity: warning
      receiver: 'slack'
```

### Special Grouping

**Group by all labels** (no grouping):
```yaml
group_by: ['...']
```

**Single global group**:
```yaml
group_by: []
```

## API Reference

### Parser Interface

```go
type Parser interface {
    Parse(data []byte) (*GroupingConfig, error)
    ParseFile(path string) (*GroupingConfig, error)
    ParseString(yaml string) (*GroupingConfig, error)
}
```

### GroupingConfig

```go
type GroupingConfig struct {
    Route *Route `yaml:"route"`
}
```

### Route

```go
type Route struct {
    Receiver       string            `yaml:"receiver"`
    GroupBy        []string          `yaml:"group_by"`
    GroupWait      *Duration         `yaml:"group_wait,omitempty"`
    GroupInterval  *Duration         `yaml:"group_interval,omitempty"`
    RepeatInterval *Duration         `yaml:"repeat_interval,omitempty"`
    Match          map[string]string `yaml:"match,omitempty"`
    MatchRE        map[string]string `yaml:"match_re,omitempty"`
    Continue       bool              `yaml:"continue,omitempty"`
    Routes         []*Route          `yaml:"routes,omitempty"`
}
```

## Validation

The parser performs comprehensive validation:

- ✅ Label name format (Prometheus standard)
- ✅ Duration ranges (group_wait: 0s-1h, etc.)
- ✅ Required fields (receiver, group_by)
- ✅ Nested route depth (max 10 levels)
- ✅ Alertmanager compatibility checks

## Error Handling

```go
config, err := parser.ParseFile("config.yml")
if err != nil {
    switch e := err.(type) {
    case *grouping.ParseError:
        fmt.Printf("Parse error at line %d: %s\n", e.Line, e.Error())
    case grouping.ValidationErrors:
        fmt.Printf("Validation failed with %d errors:\n", e.Count())
        for _, ve := range e {
            fmt.Printf("  - %s\n", ve.Message)
        }
    default:
        fmt.Printf("Error: %s\n", err)
    }
}
```

## Performance

- Small config (1KB): <1ms
- Medium config (10KB): <5ms
- Large config (100KB): <10ms

## Alertmanager Compatibility

✅ Compatible with Alertmanager v0.23+

- Same YAML format
- Same validation rules
- Same default values
- Can migrate without changes

## Examples

See `examples/` directory for complete examples:

- `basic_grouping.yaml` - Simple grouping configuration
- `nested_routes.yaml` - Hierarchical routing
- `full_featured.yaml` - All features demonstrated

## Testing

```bash
# Run tests
go test -v ./internal/infrastructure/grouping/...

# Run with coverage
go test -cover ./internal/infrastructure/grouping/...

# Run benchmarks
go test -bench=. -benchmem ./internal/infrastructure/grouping/...
```

## License

MIT License

## Related

- TN-121: Grouping Configuration Parser
- TN-122: Group Key Generator
- TN-123: Alert Group Manager
```

---

### 8. ❌ Создать examples/ (1 час)

**Создать**: `internal/infrastructure/grouping/examples/`

**Файл 1**: `basic_grouping.yaml`
```yaml
# Basic alert grouping configuration
# Groups alerts by alertname and cluster

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 30s        # Wait 30s before sending first notification
  group_interval: 5m     # Wait 5m before sending updates
  repeat_interval: 4h    # Re-send every 4h for long-running alerts
```

**Файл 2**: `nested_routes.yaml`
```yaml
# Hierarchical routing with nested routes
# Routes critical alerts to PagerDuty, warnings to Slack

route:
  receiver: 'default'
  group_by: ['alertname']

  routes:
    # Critical alerts → PagerDuty
    - match:
        severity: critical
      receiver: 'pagerduty'
      group_wait: 10s      # Faster notification for critical

    # Warning alerts → Slack
    - match:
        severity: warning
      receiver: 'slack'
      group_wait: 1m       # Can wait longer for warnings
```

**Файл 3**: `full_featured.yaml`
```yaml
# Full-featured configuration demonstrating all features

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

  routes:
    # Production critical alerts
    - match:
        severity: critical
        environment: production
      receiver: 'pagerduty-production'
      group_wait: 10s
      continue: true  # Continue to next routes

      routes:
        # Frontend team
        - match:
            team: frontend
          receiver: 'pagerduty-frontend'

        # Backend team
        - match:
            team: backend
          receiver: 'pagerduty-backend'

    # Staging alerts (regex match)
    - match_re:
        environment: ^(staging|dev)$
      receiver: 'slack-staging'
      group_wait: 1m

    # Special: Group by all labels (no grouping)
    - match:
        no_grouping: "true"
      receiver: 'email'
      group_by: ['...']
```

---

## 🟢 ЖЕЛАТЕЛЬНЫЕ (выполнить в течение 1-2 недель)

### 9. ❌ Добавить в CI/CD (30 минут)

**Обновить**: `.github/workflows/go.yml`

```yaml
# ... existing jobs ...

  test-grouping:
    name: Test Grouping Module
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24.6'

      - name: Run grouping tests
        run: |
          cd go-app
          go test -v -cover ./internal/infrastructure/grouping/... \
            -coverprofile=coverage-grouping.out

      - name: Check coverage
        run: |
          cd go-app
          coverage=$(go tool cover -func=coverage-grouping.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 85" | bc -l) )); then
            echo "Coverage $coverage% is below 85% threshold"
            exit 1
          fi

      - name: Run benchmarks
        run: |
          cd go-app
          go test -bench=. -benchmem ./internal/infrastructure/grouping/ \
            -benchtime=5s
```

---

### 10. ❌ Code review (1 час)

**Чеклист**:

- [ ] Все exported types имеют godoc комментарии
- [ ] Все функции имеют примеры использования
- [ ] Error handling корректен
- [ ] Нет race conditions
- [ ] Нет memory leaks
- [ ] Соответствует Go best practices
- [ ] Соответствует project coding standards

**Команды для проверки**:
```bash
cd go-app

# Linter
golangci-lint run ./internal/infrastructure/grouping/...

# Vet
go vet ./internal/infrastructure/grouping/...

# Race detector
go test -race ./internal/infrastructure/grouping/...

# Memory profiling
go test -memprofile=mem.prof ./internal/infrastructure/grouping/...
go tool pprof mem.prof
```

---

### 11. ❌ Security audit (1 час)

**Проверки**:

1. **YAML parsing security**
   - [ ] Защита от YAML bombing
   - [ ] Максимальный размер файла (10MB)
   - [ ] Timeout парсинга (30s)

2. **Input validation**
   - [ ] Label names валидируются (regex)
   - [ ] Duration ranges проверяются
   - [ ] Max route depth (10 levels)

3. **Injection vulnerabilities**
   - [ ] Нет eval() или exec()
   - [ ] Нет SQL injection
   - [ ] Нет command injection

**Команды**:
```bash
# gosec security scanner
gosec ./internal/infrastructure/grouping/...

# nancy vulnerability scanner
nancy sleuth
```

---

## 📊 ПРОГРЕСС

### Текущий статус: 60%

| Компонент | Статус | Приоритет |
|-----------|--------|-----------|
| ✅ Код реализован | 90% | - |
| ❌ Тесты работают | 0% | 🔴 P0 |
| ❌ Test coverage | 0% | 🔴 P0 |
| ❌ Git commits | 0% | 🔴 P0 |
| ❌ Integration | 0% | 🟡 P1 |
| ❌ Benchmarks | 0% | 🟡 P1 |
| ❌ README.md | 0% | 🟡 P1 |
| ❌ Examples | 0% | 🟢 P2 |
| ❌ CI/CD | 0% | 🟢 P2 |

### Цель: 100% в течение 1-2 дней

---

## 🎯 ИТОГОВЫЙ ЧЕКЛИСТ

### День 1 (СЕГОДНЯ):
- [ ] Исправить import в config_test.go (1 минута)
- [ ] Запустить тесты, проверить coverage (30 минут)
- [ ] Закоммитить код в git (10 минут)
- [ ] Обновить статус в tasks.md (5 минут) ✅ **DONE**

### День 2:
- [ ] Интегрировать в main.go (2-3 часа)
- [ ] Добавить integration tests (1-2 часа)
- [ ] Добавить benchmarks (1-2 часа)

### Неделя 1:
- [ ] Написать README.md (2 часа)
- [ ] Создать examples/ (1 час)
- [ ] Добавить в CI/CD (30 минут)
- [ ] Code review (1 час)
- [ ] Security audit (1 час)

### Результат:
- ✅ TN-121 завершен на 100%
- ✅ Готов к production deployment
- ✅ Разблокирован TN-122

---

**Ответственный**: TBD
**Дедлайн**: 2025-11-05 (2 дня)
**Приоритет**: 🔴 **КРИТИЧНЫЙ**
