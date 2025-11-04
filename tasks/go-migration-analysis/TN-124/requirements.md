# TN-124: Group Wait/Interval Timers (Redis Persistence)

**Дата создания**: 2025-11-03
**Статус**: 🟡 IN PROGRESS (TN-123 ✅ COMPLETE)
**Приоритет**: 🔴 CRITICAL
**Target Quality**: 150%

---

## 1. Executive Summary

**TN-124** реализует систему таймеров для Alertmanager-совместимой группировки алертов, обеспечивая:
- **group_wait**: задержка перед первой отправкой новой группы (default: 30s)
- **group_interval**: интервал между повторными отправками при изменениях (default: 5m)
- **repeat_interval**: интервал между повторными отправками без изменений (default: 4h)
- **Redis persistence**: сохранение состояния таймеров для High Availability и горизонтального масштабирования

### Критическая важность

TN-124 является **ключевым блоком** для полной замены Alertmanager:
- ❌ **Без TN-124**: группы создаются, но нотификации отправляются немедленно (alert fatigue, спам)
- ✅ **С TN-124**: корректная отправка нотификаций с учетом group_wait/interval/repeat (как в Alertmanager)

---

## 2. Обоснование задачи

### Проблема

**TN-123 (Alert Group Manager)** реализовал управление группами алертов:
- ✅ Создание/обновление/удаление групп
- ✅ Добавление алертов в группы
- ✅ Отслеживание состояния групп
- ✅ Метрики и мониторинг

**НО ОТСУТСТВУЕТ:**
- ❌ Механизм задержки отправки нотификаций (group_wait)
- ❌ Интервалы повторных отправок (group_interval, repeat_interval)
- ❌ Persistence таймеров в Redis для HA
- ❌ Graceful cancellation при изменении группы
- ❌ Recovery после рестарта сервиса

### Alertmanager Behavior (Reference)

```yaml
route:
  group_by: ['alertname', 'namespace']
  group_wait: 30s        # TN-124: ждем 30s перед первой отправкой
  group_interval: 5m     # TN-124: интервал при изменениях в группе
  repeat_interval: 4h    # TN-124: интервал при отсутствии изменений
```

**Пример сценария:**
1. **T=0s**: Создана новая группа `alertname=HighCPU` (1 алерт)
   - **Action**: Start group_wait timer (30s)
   - **Notification**: NO (ждем 30s)

2. **T=10s**: Добавлен 2-й алерт в группу
   - **Action**: Timer не сбрасывается (продолжаем ждать 30s)
   - **Notification**: NO

3. **T=30s**: group_wait timer expires
   - **Action**: Отправить нотификацию (batch 2 alerts)
   - **Next Timer**: group_interval (5m)

4. **T=2m**: Добавлен 3-й алерт в группу
   - **Action**: Cancel group_interval timer, start new group_interval (5m)
   - **Notification**: NO (ждем 5m)

5. **T=7m** (2m + 5m): group_interval timer expires
   - **Action**: Отправить нотификацию (batch 3 alerts)
   - **Next Timer**: repeat_interval (4h)

6. **T=4h 7m**: repeat_interval timer expires
   - **Action**: Отправить нотификацию (batch все алерты)
   - **Next Timer**: repeat_interval (4h)

### Решение

Реализовать **Group Timer Manager** с тремя типами таймеров:

```go
type TimerType string

const (
    // GroupWaitTimer - задержка перед первой отправкой
    GroupWaitTimer TimerType = "group_wait"

    // GroupIntervalTimer - интервал при изменениях
    GroupIntervalTimer TimerType = "group_interval"

    // RepeatIntervalTimer - интервал без изменений
    RepeatIntervalTimer TimerType = "repeat_interval"
)

// GroupTimerManager управляет таймерами групп алертов
type GroupTimerManager interface {
    // StartTimer запускает таймер для группы
    StartTimer(ctx context.Context, groupKey GroupKey, timerType TimerType, duration time.Duration) error

    // CancelTimer отменяет активный таймер группы
    CancelTimer(ctx context.Context, groupKey GroupKey) error

    // GetTimer возвращает информацию о таймере
    GetTimer(ctx context.Context, groupKey GroupKey) (*GroupTimer, error)

    // OnTimerExpired регистрирует callback при истечении таймера
    OnTimerExpired(callback TimerCallback)

    // RestoreTimers восстанавливает таймеры после рестарта (из Redis)
    RestoreTimers(ctx context.Context) (int, error)
}
```

---

## 3. Пользовательские сценарии

### Use Case 1: Новая группа (group_wait)

**Конфигурация:**
```yaml
route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
```

**Сценарий:**
1. **T=0**: Приходит алерт `HighCPU` (создается новая группа)
   - **AlertGroupManager**: создает группу `alertname=HighCPU`
   - **TimerManager**: StartTimer(group, GroupWaitTimer, 30s)
   - **Redis**: сохраняет состояние таймера
   - **Notification**: NO (ждем 30s)

2. **T=10s**: Приходит 2-й алерт `HighCPU`
   - **AlertGroupManager**: добавляет алерт в группу
   - **TimerManager**: таймер НЕ сбрасывается (продолжает ждать 20s)
   - **Notification**: NO

3. **T=30s**: group_wait timer expires
   - **TimerManager**: вызывает callback `OnTimerExpired`
   - **Publisher**: отправляет нотификацию (batch 2 alerts)
   - **TimerManager**: StartTimer(group, GroupIntervalTimer, 5m)
   - **Redis**: обновляет состояние таймера

**Метрики:**
```
alert_history_business_grouping_timers_active_total{type="group_wait"} = 0
alert_history_business_grouping_timers_active_total{type="group_interval"} = 1
alert_history_business_grouping_timers_expired_total{type="group_wait"} = 1
alert_history_business_grouping_timer_notification_delay_seconds{type="group_wait"} = 30
```

---

### Use Case 2: Изменение группы (group_interval reset)

**Сценарий:**
1. **T=0**: Группа активна, group_interval timer (5m) запущен после последней нотификации
   - **Timer state**: expires_at = T + 5m

2. **T=2m**: Добавлен новый алерт в группу
   - **AlertGroupManager**: добавляет алерт
   - **TimerManager**:
     - CancelTimer(group)  // отменяет старый таймер (3m остаток)
     - StartTimer(group, GroupIntervalTimer, 5m)  // новый таймер на 5m
   - **Redis**: обновляет expires_at = T + 7m (2m + 5m)
   - **Notification**: NO (ждем 5m)

3. **T=7m**: group_interval timer expires
   - **TimerManager**: вызывает callback
   - **Publisher**: отправляет нотификацию (с новым алертом)
   - **TimerManager**: StartTimer(group, RepeatIntervalTimer, 4h)

**Метрики:**
```
alert_history_business_grouping_timer_resets_total{type="group_interval"} = 1
```

---

### Use Case 3: Repeat interval (без изменений)

**Сценарий:**
1. **T=0**: Группа стабильна, repeat_interval timer (4h) запущен
   - **Timer state**: expires_at = T + 4h

2. **T=2h**: Никаких изменений в группе
   - **Timer state**: продолжает работать (2h осталось)

3. **T=4h**: repeat_interval timer expires
   - **TimerManager**: вызывает callback
   - **Publisher**: отправляет нотификацию (periodic reminder)
   - **TimerManager**: StartTimer(group, RepeatIntervalTimer, 4h)  // новый цикл

**Метрики:**
```
alert_history_business_grouping_timers_expired_total{type="repeat_interval"} = 1
```

---

### Use Case 4: High Availability (сервис упал и восстановился)

**Сценарий:**
1. **T=0**: Сервис работает, 10 активных групп с таймерами
   - **Redis**: хранит 10 таймеров с expires_at

2. **T=1m**: Сервис упал (pod killed)
   - **In-memory timers**: потеряны
   - **Redis timers**: сохранены

3. **T=2m**: Новый под стартовал
   - **main.go**: инициализирует TimerManager
   - **TimerManager**: вызывает RestoreTimers(ctx)
     - Читает все таймеры из Redis
     - Восстанавливает только те, где expires_at > now
     - Пропускает expired таймеры (отправляет нотификации немедленно)
   - **Result**: 8 таймеров восстановлено, 2 expired обработаны

**Метрики:**
```
alert_history_business_grouping_timers_restored_total = 8
alert_history_business_grouping_timers_missed_total = 2
```

---

## 4. Функциональные требования

### 4.1 Timer Types (3 types)

| Type | Purpose | Default | Trigger | Next State |
|------|---------|---------|---------|------------|
| **group_wait** | Задержка перед первой отправкой | 30s | Создание новой группы | group_interval |
| **group_interval** | Интервал при изменениях группы | 5m | Изменение группы после отправки | repeat_interval |
| **repeat_interval** | Интервал без изменений | 4h | Отправка без изменений | repeat_interval |

### 4.2 Timer Lifecycle

```
┌────────────────────────────────────────────────────────────────┐
│                     TIMER STATE MACHINE                         │
└────────────────────────────────────────────────────────────────┘

  [Group Created]
        │
        ▼
  ┌──────────────┐
  │ group_wait   │  (30s)
  └──────┬───────┘
         │ timer expires
         ▼
  [Send Notification]
         │
         ▼
  ┌──────────────┐
  │group_interval│  (5m)
  └──────┬───────┘
         │ alert added → reset timer
         ├──────────────────┐
         │                  │
         │ timer expires    ▼
         ▼            [Alert Added]
  [Send Notification]       │
         │                  │
         ▼                  │
  ┌──────────────┐         │
  │repeat_interval│  (4h)   │
  └──────┬───────┘         │
         │                  │
         │ timer expires    │
         ▼                  │
  [Send Notification]       │
         │                  │
         └──────────────────┘
              loop
```

### 4.3 Core Operations

#### StartTimer
- Создает новый таймер для группы
- Отменяет существующий таймер (если есть)
- Сохраняет состояние в Redis
- Запускает goroutine для ожидания
- **Performance**: <1ms

#### CancelTimer
- Останавливает активный таймер
- Удаляет из Redis
- Освобождает ресурсы
- **Performance**: <500µs

#### RestoreTimers (HA Recovery)
- Читает все таймеры из Redis
- Фильтрует expired таймеры
- Восстанавливает активные таймеры
- Отправляет нотификации для missed таймеров
- **Performance**: <100ms для 1000 таймеров

#### OnTimerExpired (Callback)
- Вызывается при истечении таймера
- Передает groupKey и timerType
- Thread-safe execution
- **Performance**: <1µs callback latency

### 4.4 Redis Persistence Schema

```
# Timer state key
Key: "timer:{groupKey}"
Type: Hash
Fields:
  - timer_type: "group_wait" | "group_interval" | "repeat_interval"
  - expires_at: Unix timestamp (int64)
  - started_at: Unix timestamp (int64)
  - duration_sec: int64
  - group_key: string
  - receiver: string (optional)

TTL: duration + 60s (grace period)

# Example
HSET timer:alertname=HighCPU timer_type group_wait
HSET timer:alertname=HighCPU expires_at 1730678400
HSET timer:alertname=HighCPU started_at 1730678370
HSET timer:alertname=HighCPU duration_sec 30
EXPIRE timer:alertname=HighCPU 90

# Timer index (для быстрого сканирования)
Key: "timers:index"
Type: Sorted Set
Score: expires_at timestamp
Member: groupKey

ZADD timers:index 1730678400 "alertname=HighCPU"
```

---

## 5. Нефункциональные требования

### 5.1 Performance Targets (150% Quality)

| Metric | Baseline Target | 150% Target | Implementation |
|--------|-----------------|-------------|----------------|
| StartTimer | <5ms | <1ms | Redis pipelining, pre-allocated goroutines |
| CancelTimer | <2ms | <500µs | Direct Redis DELETE, minimal sync |
| GetTimer | <5ms | <1ms | Redis GET, no deserialization overhead |
| RestoreTimers (1K) | <500ms | <100ms | Parallel restoration, pipeline reads |
| Timer accuracy | ±1s | ±100ms | time.Timer + Redis sync |
| Memory/timer | <1KB | <512B | Lean structs, no unnecessary fields |

### 5.2 Reliability

#### R1: Zero Timer Loss
- **Requirement**: Таймеры НЕ теряются при рестарте
- **Implementation**: Redis persistence с TTL
- **Validation**: Integration test (kill pod, verify timers restored)

#### R2: Exactly Once Notification
- **Requirement**: Нотификация отправляется ровно 1 раз при expire
- **Implementation**: Redis-based distributed lock
- **Validation**: Multi-instance test (2+ pods, verify single notification)

#### R3: Graceful Degradation
- **Requirement**: При недоступности Redis таймеры работают in-memory
- **Implementation**: Fallback to in-memory storage
- **Validation**: Redis failure test

### 5.3 Scalability

#### S1: Horizontal Scaling
- **Requirement**: Поддержка multi-instance deployment
- **Implementation**: Redis distributed storage, no in-memory shared state
- **Validation**: 3+ pods test

#### S2: High Load
- **Requirement**: Поддержка 10,000+ активных таймеров
- **Implementation**: Efficient Redis queries, parallel processing
- **Validation**: Load test (create 10K timers, verify <100ms latency)

### 5.4 Observability

#### O1: Prometheus Metrics (4+ metrics)
```
1. alert_history_business_grouping_timers_active_total{type}     (Gauge)
2. alert_history_business_grouping_timers_expired_total{type}    (Counter)
3. alert_history_business_grouping_timer_duration_seconds{type}  (Histogram)
4. alert_history_business_grouping_timer_resets_total{type}      (Counter)
5. alert_history_business_grouping_timers_restored_total         (Counter) [150%]
6. alert_history_business_grouping_timers_missed_total           (Counter) [150%]
```

#### O2: Structured Logging
- **Requirement**: Все операции логируются с context
- **Fields**: groupKey, timerType, duration, expires_at, action
- **Level**: Info (start/cancel/expire), Debug (state updates)

---

## 6. Зависимости

### 6.1 Upstream (Completed)
- ✅ **TN-121**: Grouping Configuration Parser (GroupingConfig, Route)
- ✅ **TN-122**: Group Key Generator (GroupKey generation)
- ✅ **TN-123**: Alert Group Manager (AlertGroupManager interface)
- ✅ **TN-016**: Redis Cache Wrapper (RedisCache, go-redis/v9)
- ✅ **TN-021**: Prometheus Metrics Infrastructure (metrics.BusinessMetrics)

### 6.2 Downstream (Blocked by TN-124)
- 🔒 **TN-125**: Group Storage (Redis Backend) - может использовать TimerManager
- 🔒 **TN-133**: Notification Scheduler - требует TimerManager для batching
- 🔒 **TN-140**: Silencing System - может интегрироваться с таймерами

### 6.3 Optional Integration
- **TN-033**: LLM Classification - таймеры для enriched groups
- **TN-037**: Alert History Repository - логирование timer events

---

## 7. Критерии приёмки (150% Quality)

### 7.1 Baseline (100%)
- [ ] GroupTimerManager interface определен
- [ ] DefaultTimerManager реализован (in-memory + Redis)
- [ ] 3 типа таймеров (group_wait, group_interval, repeat_interval)
- [ ] StartTimer, CancelTimer, GetTimer работают корректно
- [ ] RestoreTimers восстанавливает таймеры после рестарта
- [ ] OnTimerExpired callback механизм работает
- [ ] Redis persistence implemented
- [ ] 4 Prometheus metrics
- [ ] Unit tests (80%+ coverage)
- [ ] Integration test (timer lifecycle)

### 7.2 Enhanced (120%)
- [ ] Timer accuracy ±100ms (vs ±1s baseline)
- [ ] Distributed lock для exactly-once delivery
- [ ] Graceful degradation (Redis fallback)
- [ ] Timer index для fast scanning
- [ ] Parallel timer restoration
- [ ] Extended metrics (restored, missed counters)

### 7.3 Excellent (150%)
- [ ] **Performance**: <1ms StartTimer (5x faster than baseline)
- [ ] **Test Coverage**: 95%+ (vs 80% baseline)
- [ ] **Benchmarks**: 8+ benchmarks для всех операций
- [ ] **HA Validation**: Multi-pod integration test
- [ ] **Load Test**: 10K timers with <100ms latency
- [ ] **Documentation**: Comprehensive README (500+ lines)
- [ ] **Code Quality**: Zero technical debt, SOLID principles
- [ ] **Production Patterns**: Context support, graceful shutdown

---

## 8. Временные рамки (150% Quality)

| Phase | Задача | Время | Dependency |
|-------|--------|-------|------------|
| 1 | Requirements & Design Analysis | 3 часа | None |
| 2 | Data Models & Interfaces | 2 часа | Phase 1 |
| 3 | Redis Persistence Layer | 3 часа | Phase 2 |
| 4 | Timer Manager Implementation | 5 часов | Phase 3 |
| 5 | Prometheus Metrics | 1 час | Phase 4 |
| 6 | Comprehensive Testing (95%+) | 5 часов | Phase 5 |
| 7 | Integration with AlertGroupManager | 2 часа | Phase 6 |
| 8 | Production Validation & Docs | 2 часа | Phase 7 |

**Итого**: ~23 часа для 150% качества (vs 15 часов baseline 100%)

---

## 9. Риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| **Timer drift** (accuracy issues) | Средняя | Высокое | time.Timer + Redis sync, integration tests |
| **Redis unavailability** | Средняя | Высокое | Fallback to in-memory, graceful degradation |
| **Duplicate notifications** (multi-instance) | Высокая | Критическое | Distributed lock, exactly-once delivery |
| **Memory leak** (goroutines) | Низкая | Высокое | Proper cancellation, context cleanup, leak tests |
| **Timer restoration overhead** | Средняя | Среднее | Parallel restoration, Redis pipelining |
| **Race conditions** | Средняя | Высокое | sync.RWMutex, race detector, concurrent tests |

---

## 10. Success Metrics

После завершения TN-124 мы сможем:

1. ✅ **Alertmanager compatibility**: group_wait/interval/repeat как в Alertmanager
2. ✅ **Reduce alert fatigue**: batch notifications вместо immediate spam
3. ✅ **High Availability**: таймеры сохраняются при рестартах
4. ✅ **Horizontal Scaling**: multi-instance support через Redis
5. ✅ **Production-ready monitoring**: 6+ Prometheus metrics
6. ✅ **Разблокировать TN-125, TN-133**: downstream tasks ready to start

### Качественные метрики

```
Component                        Target      150% Target
──────────────────────────────── ─────────── ────────────
Timer Accuracy                   ±1s         ±100ms      ✅
StartTimer Latency               <5ms        <1ms        ✅
Test Coverage                    80%         95%         ✅
Active Timers Support            1,000       10,000      ✅
HA Recovery Time                 <1s         <100ms      ✅
Documentation Lines              300+        500+        ✅
Code Quality Grade               A           A+          ✅
```

---

## 11. Acceptance Criteria Checklist

### Functionality
- [ ] ✅ Создание/отмена таймеров работает
- [ ] ✅ 3 типа таймеров реализованы
- [ ] ✅ Callback mechanism функционирует
- [ ] ✅ Redis persistence сохраняет состояние
- [ ] ✅ RestoreTimers восстанавливает таймеры
- [ ] ✅ Distributed lock предотвращает дубликаты

### Performance
- [ ] ✅ StartTimer <1ms (150% target)
- [ ] ✅ CancelTimer <500µs (150% target)
- [ ] ✅ RestoreTimers <100ms для 1000 таймеров (150% target)
- [ ] ✅ Timer accuracy ±100ms (150% target)
- [ ] ✅ 10,000+ active timers support

### Quality
- [ ] ✅ Test coverage 95%+ (vs 80% baseline)
- [ ] ✅ Benchmarks для всех операций
- [ ] ✅ Integration tests (HA, multi-instance)
- [ ] ✅ Race detector tests pass
- [ ] ✅ golangci-lint clean
- [ ] ✅ Zero technical debt

### Observability
- [ ] ✅ 6 Prometheus metrics operational
- [ ] ✅ Structured logging для всех операций
- [ ] ✅ Error tracking and reporting
- [ ] ✅ Timer state visualization готова

### Documentation
- [ ] ✅ requirements.md (this file)
- [ ] ✅ design.md (architecture, data models)
- [ ] ✅ tasks.md (implementation plan)
- [ ] ✅ README.md (usage examples, API reference)
- [ ] ✅ Inline code comments and godoc

---

## 12. Out of Scope (Future Enhancements)

Следующие фичи НЕ входят в TN-124, могут быть реализованы позже:

1. **Dynamic timer adjustment** - изменение duration на лету
2. **Timer batching** - группировка нескольких таймеров в один
3. **Advanced scheduling** - cron-like timer expressions
4. **Timer history** - логирование всех timer events в PostgreSQL
5. **Grafana dashboard** - visualization для timer metrics
6. **Timer webhooks** - HTTP callbacks при expire

---

**Prepared by**: AI Assistant
**Date**: 2025-11-03
**Status**: 🟡 IN PROGRESS → 150% QUALITY TARGET
**Branch**: TBD (feature/TN-124-group-timers-150pct)
