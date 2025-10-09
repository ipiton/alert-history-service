# TN-122: Group Key Generator

## Обоснование задачи

Group Key Generator - критический компонент системы группировки, который определяет, какие алерты объединяются в одну группу. Без корректной генерации ключей группировки:
- Алерты будут неправильно группироваться
- Увеличится alert fatigue
- Нарушится логика group_wait/group_interval
- Невозможно реализовать distributed grouping

### Проблема
После парсинга конфигурации (TN-121) нам нужен способ генерировать уникальные ключи для групп алертов на основе:
- Списка label names из `group_by: ['alertname', 'cluster']`
- Фактических значений этих labels в алерте
- Special values (`...` - все labels, `[]` - глобальная группа)

## Пользовательский сценарий

### Use Case 1: Базовая группировка по alertname

**Конфигурация:**
```yaml
group_by: ['alertname']
```

**Алерты:**
```json
Alert 1: {labels: {alertname: "HighCPU", instance: "server1"}}
Alert 2: {labels: {alertname: "HighCPU", instance: "server2"}}
Alert 3: {labels: {alertname: "DiskFull", instance: "server1"}}
```

**Ожидаемое поведение:**
- Alert 1 и Alert 2 → Group Key: `alertname=HighCPU`
- Alert 3 → Group Key: `alertname=DiskFull`
- Итого: 2 группы

### Use Case 2: Группировка по нескольким labels

**Конфигурация:**
```yaml
group_by: ['alertname', 'cluster', 'environment']
```

**Алерты:**
```json
Alert 1: {labels: {alertname: "HighCPU", cluster: "prod", environment: "us-east"}}
Alert 2: {labels: {alertname: "HighCPU", cluster: "prod", environment: "us-east"}}
Alert 3: {labels: {alertname: "HighCPU", cluster: "staging", environment: "us-east"}}
```

**Ожидаемое поведение:**
- Alert 1 и Alert 2 → Group Key: `alertname=HighCPU,cluster=prod,environment=us-east`
- Alert 3 → Group Key: `alertname=HighCPU,cluster=staging,environment=us-east`
- Итого: 2 группы

### Use Case 3: Special grouping '...'

**Конфигурация:**
```yaml
group_by: ['...']
```

**Поведение:**
- Каждый уникальный набор labels создает отдельную группу
- Эффективно отключает группировку (каждый алерт в своей группе)
- Group Key включает ВСЕ labels из алерта

### Use Case 4: Global group (empty group_by)

**Конфигурация:**
```yaml
group_by: []
```

**Поведение:**
- ВСЕ алерты идут в одну глобальную группу
- Group Key: `{global}` (константа)

### Use Case 5: Missing labels

**Конфигурация:**
```yaml
group_by: ['alertname', 'cluster']
```

**Алерт:**
```json
{labels: {alertname: "HighCPU"}}  # Нет label 'cluster'
```

**Поведение:**
- Group Key: `alertname=HighCPU,cluster=<missing>`
- Алерты с missing labels группируются отдельно

## Ограничения

### Технические
1. **Hash algorithm**: FNV-1a 64-bit (совместимость с Alertmanager)
2. **Key format**: `label1=value1,label2=value2,label3=value3` (sorted by label name)
3. **Max key length**: 2048 bytes (защита от DoS)
4. **Performance**: <100μs для генерации ключа
5. **Deterministic**: Одинаковые labels всегда дают одинаковый ключ

### Функциональные
1. **Label order**: Labels в ключе всегда отсортированы алфавитно
2. **Missing labels**: Обрабатываются как `<missing>` value
3. **Empty values**: Допустимы (group_by: ['label'], alert: {label: ""})
4. **Special characters**: URL encoding для спецсимволов в values
5. **Case sensitivity**: Labels case-sensitive (как в Prometheus)

## Внешние зависимости

### Go Libraries
- `hash/fnv` - FNV-1a hashing (standard library)
- `sort` - Сортировка labels (standard library)
- `net/url` - URL encoding для values (standard library)

### Internal Dependencies
- `internal/infrastructure/grouping` - Route конфигурация (TN-121) ✅
- `internal/core/interfaces.go` - Alert struct (существующий)

### Blocks
- TN-123 (Alert Group Manager) - зависит от Group Key Generator

## Критерии приёмки

### Функциональные
- [x] Генерирует детерминированные ключи для одинаковых label sets
- [x] Поддерживает special grouping `...` (все labels)
- [x] Поддерживает global grouping `[]` (одна группа)
- [x] Обрабатывает missing labels (`<missing>`)
- [x] Обрабатывает empty label values
- [x] Сортирует labels алфавитно в ключе
- [x] URL encodes специальные символы в values
- [x] Генерирует FNV-1a 64-bit хеши (опционально, для short keys)

### Performance
- [x] Генерация ключа <100μs (benchmark)
- [x] Memory allocation <1KB per key generation
- [x] Concurrent-safe (multiple goroutines)

### Quality
- [x] Unit tests >90% coverage
- [x] Benchmark tests для performance verification
- [x] Property-based tests (same labels → same key)
- [x] Edge case tests (empty, nil, special chars)

### Examples

```go
// Test 1: Basic grouping
labels := map[string]string{"alertname": "HighCPU", "instance": "server1"}
groupBy := []string{"alertname"}
key := GenerateGroupKey(labels, groupBy)
assert.Equal(t, "alertname=HighCPU", key)

// Test 2: Multiple labels (sorted)
labels := map[string]string{"instance": "server1", "alertname": "HighCPU"}
groupBy := []string{"alertname", "instance"}
key := GenerateGroupKey(labels, groupBy)
assert.Equal(t, "alertname=HighCPU,instance=server1", key) // Sorted!

// Test 3: Missing label
labels := map[string]string{"alertname": "HighCPU"}
groupBy := []string{"alertname", "cluster"}
key := GenerateGroupKey(labels, groupBy)
assert.Equal(t, "alertname=HighCPU,cluster=<missing>", key)

// Test 4: Special grouping
labels := map[string]string{"alertname": "HighCPU", "cluster": "prod"}
groupBy := []string{"..."}
key := GenerateGroupKey(labels, groupBy)
assert.Contains(t, key, "alertname=HighCPU")
assert.Contains(t, key, "cluster=prod")

// Test 5: Global group
labels := map[string]string{"alertname": "HighCPU"}
groupBy := []string{}
key := GenerateGroupKey(labels, groupBy)
assert.Equal(t, "{global}", key)

// Test 6: Deterministic (same input → same output)
key1 := GenerateGroupKey(labels1, groupBy)
key2 := GenerateGroupKey(labels1, groupBy) // Same labels
assert.Equal(t, key1, key2)

// Test 7: FNV hash (optional, for short keys)
hash := GenerateGroupHash(labels, groupBy)
assert.Len(t, hash, 16) // 64-bit hex string
```

## Definition of Done

1. ✅ Code implementation:
   - `keygen.go` - Core key generation logic
   - `hash.go` - FNV-1a hashing implementation
   - `keygen_test.go` - Comprehensive unit tests
   - `keygen_bench_test.go` - Performance benchmarks

2. ✅ Tests:
   - Unit tests (valid cases, edge cases, error cases)
   - Property-based tests (determinism)
   - Performance benchmarks (<100μs)
   - Coverage >90%

3. ✅ Documentation:
   - Godoc для всех exported functions
   - Examples в godoc
   - Algorithm description
   - Compatibility notes (Alertmanager)

4. ✅ Integration:
   - Используется в TN-123 (Group Manager)
   - Example usage в README

5. ✅ Review:
   - Code review passed
   - Algorithm review passed
   - Performance verification passed

---

**Priority**: 🔴 CRITICAL  
**Estimated effort**: 2-3 дня  
**Dependencies**: TN-121 ✅  
**Blocking**: TN-123 (Alert Group Manager)

