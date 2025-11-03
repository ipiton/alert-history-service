# TN-123: Alert Group Manager (Lifecycle Management, Metrics)

## 1. Обоснование задачи

Alert Group Manager - критически важный компонент для реализации Alertmanager-совместимой группировки алертов. Он управляет жизненным циклом групп алертов, обеспечивая корректное добавление/удаление алертов, обновление состояния групп и сбор метрик.

### Проблема

Текущая реализация (TN-121, TN-122) предоставляет:
- ✅ Парсинг конфигурации группировки (GroupingConfig)
- ✅ Генерацию ключей группировки (GroupKeyGenerator)

**НО ОТСУТСТВУЕТ:**
- ❌ Управление жизненным циклом групп (создание, обновление, удаление)
- ❌ Хранение состояния групп в памяти/Redis
- ❌ Отслеживание активных алертов внутри группы
- ❌ Метрики по группам (active_groups, alerts_per_group)
- ❌ Интеграция с AlertProcessor для автоматической группировки

### Решение

Реализовать AlertGroupManager - центральный компонент для управления группами алертов:

```go
// AlertGroupManager управляет жизненным циклом групп алертов
type AlertGroupManager interface {
    // AddAlertToGroup добавляет алерт в соответствующую группу
    AddAlertToGroup(ctx context.Context, alert *Alert, groupKey GroupKey) error

    // RemoveAlertFromGroup удаляет алерт из группы
    RemoveAlertFromGroup(ctx context.Context, fingerprint string, groupKey GroupKey) error

    // GetGroup возвращает группу по ключу
    GetGroup(ctx context.Context, groupKey GroupKey) (*AlertGroup, error)

    // ListGroups возвращает список активных групп
    ListGroups(ctx context.Context) ([]*AlertGroup, error)

    // UpdateGroupState обновляет состояние группы
    UpdateGroupState(ctx context.Context, groupKey GroupKey, state GroupState) error

    // CleanupExpiredGroups очищает истекшие группы
    CleanupExpiredGroups(ctx context.Context, maxAge time.Duration) (int, error)

    // GetMetrics возвращает метрики по группам
    GetMetrics(ctx context.Context) (*GroupMetrics, error)
}
```

---

## 2. Пользовательский сценарий

### Use Case 1: Автоматическая группировка алертов по alertname

**Конфигурация:**
```yaml
route:
  group_by: ['alertname']
  group_wait: 30s
```

**Сценарий:**
1. Приходит алерт `HighCPU` от instance-1
   - AlertGroupManager создает новую группу с ключом `alertname=HighCPU`
   - Добавляет алерт в группу
   - Метрика `active_groups` = 1
   - Метрика `alerts_per_group{group="alertname=HighCPU"}` = 1

2. Через 10s приходит алерт `HighCPU` от instance-2
   - AlertGroupManager определяет, что группа уже существует
   - Добавляет алерт в существующую группу
   - Метрика `alerts_per_group{group="alertname=HighCPU"}` = 2

3. Через 50s оба алерта resolved
   - AlertGroupManager удаляет resolved алерты из группы
   - Если группа пустая - удаляет группу
   - Метрика `active_groups` = 0

---

### Use Case 2: Группировка по namespace и severity

**Конфигурация:**
```yaml
route:
  group_by: ['namespace', 'severity']
```

**Сценарий:**
1. Приходит 5 алертов:
   - 3x `HighCPU` в namespace=production, severity=critical
   - 2x `DiskFull` в namespace=staging, severity=warning

2. AlertGroupManager создает **2 группы:**
   - Группа 1: `namespace=production,severity=critical` (3 алерта)
   - Группа 2: `namespace=staging,severity=warning` (2 алерта)

3. Метрики:
   - `active_groups` = 2
   - `alerts_per_group{group="namespace=production,severity=critical"}` = 3
   - `alerts_per_group{group="namespace=staging,severity=warning"}` = 2

---

### Use Case 3: Cleanup истекших групп

**Сценарий:**
1. Группа не получала новых алертов 24 часа
2. Все алерты в группе resolved более 1 часа назад
3. Periodic cleanup job вызывает `CleanupExpiredGroups(ctx, 1*time.Hour)`
4. AlertGroupManager:
   - Находит истекшие группы
   - Удаляет их из хранилища
   - Возвращает количество удаленных групп (metrics)
   - Обновляет метрику `active_groups`

---

## 3. Требования

### Функциональные требования

1. **Lifecycle Management**
   - [x] Создание групп при получении первого алерта
   - [x] Добавление алертов в существующие группы
   - [x] Удаление алертов при resolved
   - [x] Автоматическое удаление пустых групп
   - [x] Cleanup истекших групп (configurable TTL)

2. **State Management**
   - [x] Хранение состояния группы (firing/resolved/mixed)
   - [x] Отслеживание времени создания группы
   - [x] Отслеживание времени последнего обновления
   - [x] Подсчет количества firing/resolved алертов в группе

3. **Metrics & Observability**
   - [x] `active_groups` - количество активных групп
   - [x] `alerts_per_group` - количество алертов в каждой группе
   - [x] `group_operations_total` - счетчик операций (add/remove/cleanup)
   - [x] `group_operation_duration_seconds` - длительность операций
   - [x] Structured logging для всех операций

4. **Integration**
   - [x] Интеграция с AlertProcessor (automatic grouping)
   - [x] Интеграция с GroupKeyGenerator (TN-122)
   - [x] HTTP API endpoints для мониторинга групп
   - [x] Graceful degradation при ошибках

5. **Performance**
   - [x] AddAlertToGroup: <1ms (target)
   - [x] GetGroup: <500μs (target)
   - [x] ListGroups: <10ms для 1000 групп (target)
   - [x] Memory efficient: <10KB per group (target)

### Нефункциональные требования

1. **Reliability**
   - Zero data loss (alerts не теряются при ошибках)
   - Graceful degradation (fallback to ungrouped processing)
   - Automatic recovery при Redis failure

2. **Scalability**
   - Поддержка до 10,000 активных групп
   - Поддержка до 1000 алертов на группу
   - Horizontal scaling готовность (Redis backend)

3. **Maintainability**
   - Clean interface design (следовать SOLID)
   - Comprehensive unit tests (95%+ coverage)
   - Benchmark tests для всех операций
   - Production-ready documentation

4. **Compatibility**
   - Alertmanager v0.25+ совместимость
   - Backwards compatible с существующим AlertProcessor
   - Redis 6.0+ support (optional, для TN-125)

---

## 4. Критерии приёмки (150% Quality)

### Baseline (100%)
- [ ] AlertGroupManager interface определен
- [ ] DefaultGroupManager реализован (in-memory storage)
- [ ] AlertGroup и GroupMetadata структуры созданы
- [ ] Все методы lifecycle management работают корректно
- [ ] 4 Prometheus metrics зарегистрированы и собираются
- [ ] Unit tests (80%+ coverage)
- [ ] Integration с AlertProcessor
- [ ] HTTP API endpoints работают

### 150% Enhancements (сверх baseline)
- [ ] **Advanced State Tracking**: GroupState enum (firing/resolved/mixed/silenced)
- [ ] **Thread-safe implementation**: sync.RWMutex для concurrent access
- [ ] **Memory optimization**: Pointer reuse, object pooling
- [ ] **Extended metrics**: histogram для group size distribution
- [ ] **Comprehensive testing**: 95%+ coverage, edge cases, race tests
- [ ] **Benchmarks**: AddAlert, GetGroup, ListGroups, CleanupExpired
- [ ] **Error handling**: Typed errors (GroupNotFoundError, etc.)
- [ ] **Documentation**: Comprehensive README (500+ lines)
- [ ] **Production patterns**: Context support, timeouts, cancellation
- [ ] **Observability**: Structured logging с correlation IDs

### Performance Targets (150%)

| Metric | Baseline Target | 150% Target | How to Achieve |
|--------|-----------------|-------------|----------------|
| AddAlertToGroup | <1ms | <500μs | Optimized map lookups, pointer reuse |
| GetGroup | <500μs | <100μs | Direct map access, no allocations |
| ListGroups (1K groups) | <10ms | <5ms | Pre-allocated slices, efficient iteration |
| Memory per group | <10KB | <5KB | Lean AlertGroup struct, shared pointers |
| Test coverage | 80% | 95% | Comprehensive edge cases, error paths |
| Code quality | A | A+ | golangci-lint, code review standards |

---

## 5. Зависимости

### Upstream (завершены, разблокируют TN-123)
- ✅ **TN-121**: Grouping Configuration Parser (GroupingConfig, Route)
- ✅ **TN-122**: Group Key Generator (GroupKey, GroupKeyGenerator)
- ✅ **TN-031**: Alert domain models (Alert struct)
- ✅ **TN-036**: Deduplication & fingerprinting
- ✅ **TN-021**: Prometheus metrics infrastructure

### Downstream (блокированы TN-123, будут разблокированы)
- 🔒 **TN-124**: Group Wait/Interval Timers (требует AlertGroupManager)
- 🔒 **TN-125**: Group Storage (Redis Backend) (требует AlertGroupManager interface)
- 🔒 **TN-133**: Notification Scheduler (требует группы для batching)

### Optional Integration Points
- **TN-033**: LLM Classification (группировка enriched alerts)
- **TN-035**: Alert Filtering (группировка после фильтрации)
- **TN-037**: Alert History Repository (аналитика по группам)

---

## 6. Текущий статус

**Статус**: 🟡 READY TO START (dependencies completed)
**Блокеры**: НЕТ (TN-121 ✅, TN-122 ✅)
**Прогресс**: 0% → 150% (target)
**Приоритет**: 🔴 CRITICAL (blocks TN-124, TN-125)

---

## 7. Временные рамки (оценка для 150% качества)

| Phase | Задача | Время | Статус |
|-------|--------|-------|--------|
| 1 | Interfaces & Data Models | 2 часа | 🔲 Pending |
| 2 | Core Implementation (DefaultGroupManager) | 4 часа | 🔲 Pending |
| 3 | Prometheus Metrics Integration | 1 час | 🔲 Pending |
| 4 | AlertProcessor Integration | 2 часа | 🔲 Pending |
| 5 | HTTP API Endpoints | 1 час | 🔲 Pending |
| 6 | Comprehensive Testing (95%+ coverage) | 4 часа | 🔲 Pending |
| 7 | Benchmarks & Performance Optimization | 2 часа | 🔲 Pending |
| 8 | Documentation (README, examples) | 2 часа | 🔲 Pending |
| 9 | Validation & Production Readiness | 1 час | 🔲 Pending |

**Итого**: ~19 часов для 150% качества (vs 12 часов baseline 100%)

---

## 8. Риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Memory leak при большом количестве групп | Средняя | Высокое | Benchmarks, memory profiling, cleanup TTL |
| Race conditions при concurrent access | Средняя | Высокое | sync.RWMutex, race detector tests |
| Performance degradation при 10K+ групп | Низкая | Среднее | Benchmarks, optimization, sharding готовность |
| Breaking changes в AlertProcessor | Низкая | Среднее | Backwards compatible interface, optional feature |

---

## 9. Success Metrics

После завершения TN-123 мы сможем:
1. ✅ Автоматически группировать алерты по labels
2. ✅ Отслеживать состояние групп в реальном времени
3. ✅ Мониторить группы через Prometheus metrics
4. ✅ Управлять группами через HTTP API
5. ✅ Разблокировать TN-124 (Group Timers) и TN-125 (Redis Storage)
6. ✅ Снизить alert fatigue (10x меньше нотификаций)

**Target Quality**: **150%** (A+ grade, production-ready)
