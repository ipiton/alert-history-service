# TN-121: Grouping Configuration Parser

## Обоснование задачи

Alert Grouping - это критически важная функциональность Alertmanager, которая снижает alert fatigue через объединение похожих алертов в группы. Без группировки операторы получают сотни отдельных нотификаций вместо одной сводной.

### Проблема
Текущий Alert History Service не поддерживает группировку алертов. Каждый алерт обрабатывается и отправляется индивидуально, что приводит к:
- Alert fatigue (перегрузка нотификациями)
- Высокая стоимость нотификаций (PagerDuty charges per alert)
- Сложность анализа связанных проблем
- Несовместимость с Alertmanager workflows

### Решение
Реализовать парсер конфигурации группировки, совместимый с Alertmanager YAML формат:
```yaml
route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
```

## Пользовательский сценарий

### Use Case 1: Группировка по alertname и namespace
```yaml
# alertmanager.yml
route:
  receiver: 'default'
  group_by: ['alertname', 'namespace']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
```

**Поведение:**
1. Поступает алерт `HighCPU` в namespace `production`
2. Система ждет 30s (group_wait) для накопления похожих алертов
3. Если приходят еще `HighCPU` алерты из `production` - они группируются
4. После 30s отправляется **одна** нотификация с группой алертов
5. Последующие обновления группы отправляются каждые 5m (group_interval)
6. Если алерты все еще активны через 12h - повторная нотификация (repeat_interval)

### Use Case 2: Разная группировка для разных routes
```yaml
route:
  receiver: 'default'
  group_by: ['alertname']
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
      group_by: ['alertname', 'instance']  # Override для critical
      group_wait: 10s  # Быстрее для critical
      group_interval: 3m
```

**Поведение:**
- Critical алерты группируются по `alertname + instance` с быстрой отправкой (10s)
- Остальные алерты группируются только по `alertname` с задержкой 30s (default)

### Use Case 3: Отключение группировки
```yaml
route:
  group_by: ['...']  # Special value: группировка по всем labels = no grouping
```

## Ограничения

### Технические
1. **YAML-only support**: JSON конфигурация не поддерживается (как в Alertmanager)
2. **Label names validation**: Допустимые символы `[a-zA-Z_][a-zA-Z0-9_]*`
3. **Timer limits**:
   - `group_wait`: 0s - 1h (default: 30s)
   - `group_interval`: 1s - 24h (default: 5m)
   - `repeat_interval`: 1m - 168h/7d (default: 4h)
4. **Special values**:
   - `group_by: ['...']` - группировка по всем labels (эффективно отключает группировку)
   - Пустой `group_by: []` - одна глобальная группа

### Функциональные
1. **Hot reload**: Изменения конфигурации применяются без потери активных групп
2. **Backward compatibility**: Совместимость с существующим Alertmanager config
3. **Validation**: Строгая валидация до применения конфигурации
4. **Error handling**: Детальные ошибки при парсинге

## Внешние зависимости

### Go Libraries
- `gopkg.in/yaml.v3` - YAML парсинг (v3 для лучшей error reporting)
- `github.com/go-playground/validator/v10` - структурная валидация
- `time.Duration` - парсинг временных интервалов

### Internal Dependencies
- Никаких зависимостей от других TN-задач (это foundation задача)
- Интеграция с config loader (существующий `internal/config/`)

## Критерии приёмки

### Функциональные
- [x] Парсит валидный Alertmanager YAML конфигурацию
- [x] Поддерживает nested routes с override параметрами
- [x] Валидирует label names (regex: `^[a-zA-Z_][a-zA-Z0-9_]*$`)
- [x] Валидирует timer ranges (min/max значения)
- [x] Обрабатывает special value `group_by: ['...']`
- [x] Возвращает детальные ошибки валидации с line numbers

### Non-functional
- [x] Performance: Парсинг <10ms для конфигурации <100KB
- [x] Memory: Парсер не держит копию raw YAML после парсинга
- [x] Error messages: User-friendly сообщения (не internal stack traces)
- [x] Unit tests: >85% coverage
- [x] Documentation: Godoc для всех публичных функций/структур

### Примеры тестов
```go
// Valid config
config := `
route:
  group_by: ['alertname', 'cluster']
  group_wait: 30s
  group_interval: 5m
`
cfg, err := ParseGroupingConfig(config)
assert.NoError(t, err)
assert.Equal(t, []string{"alertname", "cluster"}, cfg.GroupBy)

// Invalid label name
config := `
route:
  group_by: ['alert-name']  # Dash not allowed
`
_, err := ParseGroupingConfig(config)
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid label name")

// Out of range timer
config := `
route:
  group_wait: 2h  # Exceeds 1h max
`
_, err := ParseGroupingConfig(config)
assert.Error(t, err)
assert.Contains(t, err.Error(), "group_wait must be")
```

## Definition of Done

1. ✅ Code implementation:
   - Структуры данных (`GroupingConfig`, `RouteConfig`, `TimerConfig`)
   - Parser функция с YAML support
   - Validator с comprehensive rules
   - Error types для различных ошибок валидации

2. ✅ Tests:
   - Unit тесты для парсера (valid/invalid configs)
   - Edge cases (empty config, missing fields, special values)
   - Performance benchmarks
   - Coverage >85%

3. ✅ Documentation:
   - Godoc для всех публичных API
   - Examples в godoc
   - README в директории пакета

4. ✅ Integration:
   - Интегрирован с существующим config loader
   - Config validation в CI pipeline
   - Example конфигурационный файл

5. ✅ Review:
   - Code review passed
   - Architecture review passed
   - Security review (no code injection через YAML)

---

**Priority**: 🔴 CRITICAL  
**Estimated effort**: 3-4 дня  
**Dependencies**: None  
**Blocking**: TN-122, TN-123, TN-124, TN-125

