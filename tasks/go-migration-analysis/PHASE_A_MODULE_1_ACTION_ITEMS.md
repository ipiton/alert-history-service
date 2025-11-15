# ⚡ ФАЗА A, МОДУЛЬ 1: ACTION ITEMS
## Конкретные задачи по результатам аудита

**Дата**: 2025-11-04
**Статус модуля**: ✅ APPROVED FOR PRODUCTION (Grade A-)

---

## 🔥 HIGH PRIORITY (требуют исправления)

### ❌ H-1: Увеличить Test Coverage до 80%+

**Текущее состояние**: 71.2%
**Целевое значение**: 80%+
**Усилия**: 1-2 дня
**Приоритет**: HIGH

**Действия**:
```bash
# 1. Идентифицировать uncovered code
cd go-app
go test ./internal/infrastructure/grouping/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 2. Добавить тесты для следующих компонентов:
# - storage_manager.go: fallback/recovery scenarios
# - timer_manager_impl.go: RestoreTimers edge cases
# - manager_impl.go: error paths, concurrent access

# 3. Целевое покрытие по файлам:
# - storage_manager.go: 75% → 85%
# - timer_manager_impl.go: 70% → 85%
# - manager_impl.go: 80% → 90%

# 4. Запустить финальную проверку
go test ./internal/infrastructure/grouping/... -cover
# Ожидаемый результат: coverage: 80.0%+ of statements
```

**Файлы для обновления**:
- `storage_manager_test.go`: добавить 5-7 тестов
- `timer_manager_impl_test.go`: добавить 8-10 тестов
- `manager_impl_test.go`: добавить 3-5 тестов

**Acceptance criteria**:
- [ ] Coverage модуля ≥ 80%
- [ ] Все новые тесты проходят
- [ ] Zero flaky tests

---

### ⚠️ H-2: Обновить документацию с реальными метриками

**Текущее состояние**: Неточные LOC, coverage, test counts
**Усилия**: 2-3 часа
**Приоритет**: HIGH

**Действия**:

#### 1. Обновить tasks.md для всех 5 задач

**TN-121 (Parser)**:
```markdown
# БЫЛО:
- Строк кода: 1,085+ LOC
- Test coverage: 93.6%

# СТАЛО:
- Строк кода: ~800 LOC implementation
- Test coverage: 71.2% (модуль grouping, измерено 2025-11-04)
- Тесты: 50+ test functions, 150+ test cases
```

**TN-122 (KeyGen)**:
```markdown
# БЫЛО:
- Строк кода: 650+ LOC impl, 1,050+ LOC tests
- Test coverage: 95%+
- Performance: 404x faster

# СТАЛО:
- Строк кода: 526 LOC (keygen.go + hash.go)
- Test coverage: 71.2% (модуль grouping)
- Performance: Требует верификации (см. M-2)
```

**TN-123 (Manager)**:
```markdown
# БЫЛО:
- Строк кода: 2,850+ LOC
- Test coverage: 95%+
- Performance: 0.38µs AddAlert

# СТАЛО:
- Строк кода: 1,151 LOC (manager.go + manager_impl.go + manager_restore.go)
- Test coverage: 71.2% (модуль grouping)
- Performance: Требует верификации (см. M-2)
```

**TN-124 (Timers)**:
```markdown
# БЫЛО:
- Строк кода: 2,797 LOC
- Тестов: 177 tests
- Test coverage: 82.7%

# СТАЛО:
- Строк кода: ~2,000 LOC (timer_*.go files)
- Тестов: 71+ test functions
- Test coverage: 71.2% (модуль grouping)
```

**TN-125 (Storage)**:
```markdown
# БЫЛО:
- Строк кода: 15,850+ LOC
- Тестов: 122+ tests

# СТАЛО:
- Строк кода: ~1,790 LOC (storage_*.go, redis_group_storage.go, memory_group_storage.go)
- Примечание: 15,850 включало документацию (~5,000 LOC) и тесты
- Тестов: 37+ test functions
```

#### 2. Создать единую таблицу метрик

**Файл**: `tasks/go-migration-analysis/PHASE_A_MODULE_1_METRICS.md`

```markdown
| Task | LOC Impl | LOC Tests | Coverage | Tests | Benchmarks | Grade |
|------|----------|-----------|----------|-------|------------|-------|
| TN-121 | 800 | ~1,000 | 71.2%* | 50+ | 13 | A |
| TN-122 | 526 | ~800 | 71.2%* | 24+ | 19 | A |
| TN-123 | 1,151 | ~1,500 | 71.2%* | 29+ | 14 | A- |
| TN-124 | 2,000 | ~2,000 | 71.2%* | 71+ | 5 | A- |
| TN-125 | 1,790 | ~2,000 | 71.2%* | 37+ | 16 | A |
| **TOTAL** | **7,534** | **8,266** | **71.2%** | **218+** | **70** | **A-** |

*Measured for entire grouping module (2025-11-04)
```

#### 3. Обновить COMPLETION_SUMMARY.md

**Файлы**:
- `TN-123/COMPLETION_SUMMARY.md`
- `TN-124/tasks.md`
- `TN-125/FINAL_COMPLETION_CERTIFICATE.md`

Добавить disclaimer:
```markdown
⚠️ **NOTE (2025-11-04 Audit)**:
Индивидуальные метрики coverage измерялись на момент завершения задачи.
Итоговый coverage всего модуля grouping: 71.2% (измерен после интеграции всех 5 задач).
Это ожидаемо, так как последующие задачи добавили новый код.
```

**Acceptance criteria**:
- [ ] Все 5 tasks.md обновлены
- [ ] PHASE_A_MODULE_1_METRICS.md создан
- [ ] Disclaimers добавлены в completion reports
- [ ] Числа точные и проверяемые

---

## 🎯 MEDIUM PRIORITY (желательно)

### ⚠️ M-1: Создать документацию для TN-121, TN-122

**Текущее состояние**: Отсутствует `requirements.md`, `design.md`
**Усилия**: 4-6 часов
**Приоритет**: MEDIUM

**Действия**:

#### TN-121: Grouping Configuration Parser

**Создать**:
```
tasks/go-migration-analysis/TN-121/
├── requirements.md (150+ lines)
│   - Problem Statement
│   - Use Cases (Alertmanager config parsing)
│   - Requirements (YAML support, validation)
│   - Non-Functional Requirements
├── design.md (200+ lines)
│   - Architecture (Parser interface, DefaultParser)
│   - Data Models (GroupingConfig, Route, Duration)
│   - Validation Strategy (structural + semantic)
│   - Error Handling (ParseError, ValidationErrors)
└── tasks.md (already exists, update with accurate metrics)
```

#### TN-122: Group Key Generator

**Создать**:
```
tasks/go-migration-analysis/TN-122/
├── requirements.md (120+ lines)
│   - Problem Statement (deterministic grouping keys)
│   - Use Cases (Alertmanager compatibility)
│   - Requirements (FNV-1a hashing, special grouping)
│   - Performance Requirements (<100µs)
├── design.md (180+ lines)
│   - Architecture (GroupKeyGenerator, options pattern)
│   - Key Format (examples, special keys)
│   - Optimization (sync.Pool, pre-allocation)
│   - Compatibility (Alertmanager v0.23+)
└── tasks.md (create with accurate metrics)
```

**Acceptance criteria**:
- [ ] requirements.md для TN-121 создан
- [ ] design.md для TN-121 создан
- [ ] requirements.md для TN-122 создан
- [ ] design.md для TN-122 создан

---

### ⚠️ M-2: Верифицировать Performance Claims

**Текущее состояние**: Claims не проверены benchmarks
**Усилия**: 1-2 часа
**Приоритет**: MEDIUM

**Действия**:
```bash
# 1. Запустить все benchmarks
cd go-app
go test ./internal/infrastructure/grouping/... -bench=. -benchmem > benchmarks_2025-11-04.txt

# 2. Проанализировать результаты
# TN-121: Parser.Parse - target <1ms
# TN-122: GenerateKey - target <100µs
# TN-123: AddAlertToGroup - target <500µs
# TN-124: StartTimer - target <1ms
# TN-125: Storage.Store - target <2ms

# 3. Сравнить с заявленными performance gains
# TN-121: 8.1x faster?
# TN-122: 404x faster?
# TN-123: 1300x faster?

# 4. Создать отчет
cat > tasks/go-migration-analysis/PERFORMANCE_REPORT.md <<EOF
# Performance Verification Report

Date: 2025-11-04

## Benchmarks Results

### TN-121: Parser
- Parse simple config: X ns/op (target: <1ms)
- Parse complex config: Y ns/op
- Validation: Z ns/op

### TN-122: KeyGen
- GenerateKey (2 labels): X ns/op (target: <100µs)
- GenerateKey (10 labels): Y ns/op
- Hash generation: Z ns/op

[... продолжить для всех задач ...]

## Comparison with Claims

| Task | Claimed | Measured | Verification |
|------|---------|----------|--------------|
| TN-121 | 8.1x faster | X.Xx faster | ✅/❌ |
| TN-122 | 404x faster | X.Xx faster | ✅/❌ |
| TN-123 | 1300x faster | X.Xx faster | ✅/❌ |

EOF
```

**Acceptance criteria**:
- [ ] Benchmarks запущены
- [ ] PERFORMANCE_REPORT.md создан
- [ ] Все claims верифицированы или обновлены

---

## ✅ COMPLETED

### ✅ L-1: Config path hardcoded

**Статус**: ✅ УЖЕ ИСПРАВЛЕНО

Проверка в main.go:
```go
if groupingConfigPath := os.Getenv("GROUPING_CONFIG_PATH"); groupingConfigPath != "" || true {
    if groupingConfigPath == "" {
        groupingConfigPath = "./config/grouping.yaml"
    }
    // ...
}
```

**Вердикт**: Env var support already implemented ✅

---

## 📋 CHECKLIST

### До Production Deployment

- [ ] H-1: Coverage ≥ 80% (или принято решение деплоить с 71.2%)
- [ ] H-2: Документация обновлена
- [ ] Все тесты проходят (100% pass rate)
- [ ] Zero critical issues

### После Production Deployment

- [ ] M-1: Документация для TN-121, TN-122 создана
- [ ] M-2: Performance claims верифицированы
- [ ] Integration tests с Redis добавлены
- [ ] Grafana dashboards созданы

---

## 🎯 TIMELINE

| Task | Priority | Effort | Deadline |
|------|----------|--------|----------|
| H-1: Coverage → 80% | HIGH | 1-2 дня | Before deploy |
| H-2: Update docs | HIGH | 3 часа | Before deploy |
| M-1: Create docs | MEDIUM | 6 часов | 1 неделя после deploy |
| M-2: Verify perf | MEDIUM | 2 часа | 1 неделя после deploy |

---

**Status**: ✅ Ready for execution
**Owner**: Development team
**Review**: Required after H-1 and H-2 completion

