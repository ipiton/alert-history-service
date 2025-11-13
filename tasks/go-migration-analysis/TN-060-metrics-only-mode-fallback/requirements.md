# TN-060: Metrics-Only Mode Fallback - Requirements

**Version**: 1.0
**Date**: 2025-01-13
**Status**: Requirements Complete
**Quality Target**: 150%+ (Grade A+, Enterprise-Grade)
**Branch**: `feature/TN-060-metrics-only-mode-150pct`

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [Business Requirements](#business-requirements)
3. [Functional Requirements](#functional-requirements)
4. [Non-Functional Requirements](#non-functional-requirements)
5. [Technical Requirements](#technical-requirements)
6. [Dependencies](#dependencies)
7. [Constraints](#constraints)
8. [Acceptance Criteria](#acceptance-criteria)
9. [Success Metrics](#success-metrics)

---

## 1. Executive Summary

### 1.1 Purpose

Реализовать полноценную систему **Metrics-Only Mode Fallback** для Publishing System, обеспечивающую graceful degradation при отсутствии доступных publishing targets. Система должна автоматически переключаться между нормальным и metrics-only режимами, сохраняя полную наблюдаемость и работоспособность даже в деградированном состоянии.

### 1.2 Scope

**In Scope**:
- Централизованное управление состоянием режима (ModeManager)
- Автоматическое обнаружение переходов между режимами
- Интеграция во все компоненты Publishing System
- Comprehensive observability (метрики, логи, API)
- Graceful handling в Queue, Coordinator, ParallelPublisher
- Zero-downtime transitions
- Production-ready reliability

**Out of Scope**:
- Manual mode override API (future enhancement)
- Multi-region mode synchronization
- Historical mode analytics (beyond basic metrics)

### 1.3 Stakeholders

- **Primary**: DevOps Team, Platform Team, SRE Team
- **Secondary**: Monitoring Team, Security Team
- **End Users**: Alert recipients (indirectly affected)

### 1.4 Business Value

- **High Availability**: Система продолжает работать даже без publishing targets
- **Operational Excellence**: Полная наблюдаемость в любом состоянии
- **Cost Efficiency**: Избежание cascade failures и downtime
- **Compliance**: Audit trail для режимов и переходов
- **Developer Experience**: Единый API для проверки режима

---

## 2. Business Requirements

### BR-001: Graceful Degradation
**Priority**: Critical
**Description**: Система должна gracefully деградировать в metrics-only режим при отсутствии доступных targets, продолжая обрабатывать алерты и собирать метрики.

**Rationale**: Отсутствие publishing targets не должно приводить к полной остановке системы или потере наблюдаемости.

**Success Criteria**:
- Система автоматически переключается в metrics-only режим при `enabled_targets == 0`
- Alert processing продолжается без ошибок
- Метрики продолжают собираться
- API endpoints остаются доступными

### BR-002: Automatic Recovery
**Priority**: Critical
**Description**: Система должна автоматически восстанавливаться в нормальный режим при появлении доступных targets.

**Rationale**: Минимизация manual intervention и обеспечение быстрого восстановления.

**Success Criteria**:
- Автоматическое переключение в normal режим при `enabled_targets > 0`
- Плавный переход без потери данных
- Логирование всех переходов
- Метрики для отслеживания переходов

### BR-003: Operational Visibility
**Priority**: High
**Description**: Операторы должны иметь полную видимость текущего режима и истории переходов.

**Rationale**: Критично для troubleshooting и мониторинга состояния системы.

**Success Criteria**:
- API endpoint для проверки текущего режима
- Prometheus метрики для режима и переходов
- Structured logging для всех переходов
- Grafana dashboard для визуализации

### BR-004: Zero Data Loss
**Priority**: Critical
**Description**: При переходе между режимами не должно происходить потери алертов или метрик.

**Rationale**: Критично для reliability и compliance.

**Success Criteria**:
- Алерты не теряются при переходе в metrics-only режим
- Метрики продолжают собираться
- Queue сохраняет состояние
- Graceful shutdown в любом режиме

---

## 3. Functional Requirements

### FR-001: ModeManager Service
**Priority**: Critical
**Description**: Централизованный сервис для управления состоянием режима.

**Details**:
- **Interface**: `ModeManager` с методами:
  - `GetCurrentMode() Mode` - получить текущий режим
  - `IsMetricsOnly() bool` - проверка metrics-only режима
  - `CheckModeTransition() (Mode, bool, error)` - проверка перехода
  - `OnTargetsChanged() error` - обработка изменения targets
  - `Subscribe(callback ModeChangeCallback) UnsubscribeFunc` - подписка на изменения
  - `GetModeMetrics() ModeMetrics` - получение метрик

- **Modes**:
  - `ModeNormal` - нормальный режим (enabled_targets > 0)
  - `ModeMetricsOnly` - metrics-only режим (enabled_targets == 0)

- **State Management**:
  - Thread-safe (sync.RWMutex)
  - Cached mode (TTL 1s для performance)
  - Event-driven updates (on target discovery/refresh)
  - Periodic validation (every 5s)

**Acceptance Criteria**:
- [ ] ModeManager interface определен
- [ ] DefaultModeManager реализован
- [ ] Thread-safe state management
- [ ] Cached mode для performance
- [ ] Event-driven updates
- [ ] Unit tests (95%+ coverage)

---

### FR-002: Integration in SubmitAlert Handler
**Priority**: Critical
**Description**: Интеграция проверки режима в SubmitAlert handler.

**Details**:
- Проверка режима перед добавлением в очередь
- В metrics-only режиме:
  - Возврат информативного ответа (HTTP 200, mode: "metrics-only")
  - Метрика для rejected submissions
  - Логирование (info level)
- В normal режиме:
  - Стандартное поведение (добавление в очередь)

**Acceptance Criteria**:
- [ ] Проверка режима в SubmitAlert
- [ ] Информативный ответ в metrics-only режиме
- [ ] Метрики для rejected submissions
- [ ] Логирование режима
- [ ] Integration tests

---

### FR-003: Integration in PublishingQueue
**Priority**: Critical
**Description**: Интеграция проверки режима в PublishingQueue workers.

**Details**:
- Проверка режима в worker loop
- В metrics-only режиме:
  - Skip processing (graceful skip)
  - Метрика для skipped jobs
  - Логирование (debug level)
- В normal режиме:
  - Стандартное поведение (обработка jobs)

**Acceptance Criteria**:
- [ ] Проверка режима в worker loop
- [ ] Graceful skip в metrics-only режиме
- [ ] Метрики для skipped jobs
- [ ] Логирование skipped jobs
- [ ] Integration tests

---

### FR-004: Integration in PublishingCoordinator
**Priority**: High
**Description**: Интеграция проверки режима в PublishingCoordinator.

**Details**:
- Проверка режима перед публикацией
- В metrics-only режиме:
  - Early return (no publishing attempts)
  - Метрика для skipped publications
  - Логирование (info level)
- В normal режиме:
  - Стандартное поведение (публикация)

**Acceptance Criteria**:
- [ ] Проверка режима в coordinator
- [ ] Early return в metrics-only режиме
- [ ] Метрики для skipped publications
- [ ] Логирование skipped publications
- [ ] Integration tests

---

### FR-005: Integration in ParallelPublisher
**Priority**: High
**Description**: Интеграция проверки режима в ParallelPublisher.

**Details**:
- Проверка режима перед параллельной публикацией
- В metrics-only режиме:
  - Graceful handling (no parallel publishes)
  - Метрика для skipped parallel publishes
  - Логирование (info level)
- В normal режиме:
  - Стандартное поведение (параллельная публикация)

**Acceptance Criteria**:
- [ ] Проверка режима в parallel publisher
- [ ] Graceful handling в metrics-only режиме
- [ ] Метрики для skipped parallel publishes
- [ ] Логирование skipped parallel publishes
- [ ] Integration tests

---

### FR-006: Automatic Transition Detection
**Priority**: Critical
**Description**: Автоматическое обнаружение переходов между режимами.

**Details**:
- Event-driven detection (on target discovery/refresh)
- Periodic validation (every 5s)
- Hysteresis (debouncing) для предотвращения flapping:
  - Normal → Metrics-Only: immediate (enabled_targets == 0)
  - Metrics-Only → Normal: immediate (enabled_targets > 0)
- Transition logging (structured, info level)

**Acceptance Criteria**:
- [ ] Event-driven detection
- [ ] Periodic validation
- [ ] Hysteresis logic
- [ ] Transition logging
- [ ] Unit tests

---

### FR-007: Prometheus Metrics
**Priority**: High
**Description**: Prometheus метрики для отслеживания режима и переходов.

**Details**:
- `publishing_mode_current` (gauge) - текущий режим (0=normal, 1=metrics-only)
- `publishing_mode_transitions_total` (counter) - количество переходов
- `publishing_mode_duration_seconds` (histogram) - длительность в каждом режиме
- `publishing_mode_check_duration_seconds` (histogram) - время проверки режима
- `publishing_submissions_rejected_total{reason="metrics_only"}` (counter) - отклоненные submissions
- `publishing_jobs_skipped_total{reason="metrics_only"}` (counter) - пропущенные jobs

**Acceptance Criteria**:
- [ ] Все метрики определены
- [ ] Метрики экспортируются в Prometheus
- [ ] Метрики документированы
- [ ] Grafana dashboard готов

---

### FR-008: Structured Logging
**Priority**: High
**Description**: Structured logging для режимов и переходов.

**Details**:
- Log при переходе режима:
  - Level: INFO
  - Fields: `mode`, `previous_mode`, `enabled_targets`, `reason`
- Log при rejected submission:
  - Level: INFO
  - Fields: `mode`, `alert_fingerprint`, `reason`
- Log при skipped job:
  - Level: DEBUG
  - Fields: `mode`, `job_id`, `reason`

**Acceptance Criteria**:
- [ ] Structured logging для переходов
- [ ] Structured logging для rejected submissions
- [ ] Structured logging для skipped jobs
- [ ] Логи в JSON format
- [ ] Логи документированы

---

### FR-009: API Endpoint Enhancement
**Priority**: Medium
**Description**: Улучшение существующего API endpoint для режима.

**Details**:
- Endpoint: `GET /api/v1/publishing/mode` (существующий)
- Response enhancement:
  - Добавить `transition_count` - количество переходов
  - Добавить `current_mode_duration_seconds` - длительность в текущем режиме
  - Добавить `last_transition_time` - время последнего перехода
  - Добавить `last_transition_reason` - причина последнего перехода

**Acceptance Criteria**:
- [ ] API endpoint улучшен
- [ ] Response расширен
- [ ] API документирован
- [ ] Integration tests

---

## 4. Non-Functional Requirements

### NFR-001: Performance
**Priority**: High
**Description**: Проверка режима не должна влиять на performance.

**Requirements**:
- Mode check overhead: <1µs (150% target: <0.5µs)
- API response time: <10ms (150% target: <5ms)
- Memory overhead: <500KB (150% target: <250KB)
- CPU overhead: <0.1% (150% target: <0.05%)

**Acceptance Criteria**:
- [ ] Benchmarks для mode check
- [ ] Benchmarks для API endpoint
- [ ] Memory profiling
- [ ] CPU profiling

---

### NFR-002: Reliability
**Priority**: Critical
**Description**: Система должна быть надежной и устойчивой к ошибкам.

**Requirements**:
- Zero race conditions (validated with `go test -race`)
- Thread-safe operations
- Graceful error handling
- No data loss during transitions

**Acceptance Criteria**:
- [ ] Race detector tests
- [ ] Thread-safety tests
- [ ] Error handling tests
- [ ] Transition tests

---

### NFR-003: Scalability
**Priority**: Medium
**Description**: Система должна масштабироваться с ростом нагрузки.

**Requirements**:
- Support 10,000+ mode checks/second
- Support 1,000+ concurrent API requests
- Linear scaling with load

**Acceptance Criteria**:
- [ ] Load tests
- [ ] Scalability tests
- [ ] Performance benchmarks

---

### NFR-004: Observability
**Priority**: High
**Description**: Полная наблюдаемость режима и переходов.

**Requirements**:
- Prometheus metrics
- Structured logging
- API endpoints
- Grafana dashboard

**Acceptance Criteria**:
- [ ] Метрики экспортируются
- [ ] Логи структурированы
- [ ] API endpoints доступны
- [ ] Grafana dashboard готов

---

## 5. Technical Requirements

### TR-001: Go Version
**Priority**: Critical
**Requirement**: Go 1.24.6+

### TR-002: Dependencies
**Priority**: Critical
**Requirement**: Использовать существующие зависимости (no new deps)

### TR-003: Code Quality
**Priority**: High
**Requirement**:
- Zero golangci-lint warnings
- Zero race conditions
- 95%+ test coverage

### TR-004: Architecture
**Priority**: High
**Requirement**:
- Hexagonal architecture
- Dependency injection
- Interface-based design

---

## 6. Dependencies

### Internal Dependencies
- **TN-047**: Target Discovery Manager (для получения enabled targets count)
- **TN-048**: Target Refresh Mechanism (для event-driven updates)
- **TN-049**: Target Health Monitoring (для health-aware mode detection)
- **TN-056**: Publishing Queue (для интеграции)
- **TN-057**: Publishing Metrics & Stats (для метрик)
- **TN-058**: Parallel Publishing (для интеграции)
- **TN-059**: Publishing API (для API endpoints)

### External Dependencies
- None (используем существующие)

---

## 7. Constraints

### C-001: Backward Compatibility
**Constraint**: Не нарушать существующий API (`GET /api/v1/publishing/mode`)

### C-002: Performance Impact
**Constraint**: Минимальный overhead на hot paths (<1µs per check)

### C-003: Memory Usage
**Constraint**: Минимальное использование памяти (<500KB)

### C-004: Code Complexity
**Constraint**: Простота и читаемость кода (cyclomatic complexity <10)

---

## 8. Acceptance Criteria

### AC-001: ModeManager Implementation
- [ ] ModeManager interface определен
- [ ] DefaultModeManager реализован
- [ ] Thread-safe state management
- [ ] Unit tests (95%+ coverage)
- [ ] Benchmarks (<1µs overhead)

### AC-002: Integration Complete
- [ ] Интеграция в SubmitAlert handler
- [ ] Интеграция в PublishingQueue
- [ ] Интеграция в PublishingCoordinator
- [ ] Интеграция в ParallelPublisher
- [ ] Integration tests (all passing)

### AC-003: Observability Complete
- [ ] Prometheus metrics экспортируются
- [ ] Structured logging реализовано
- [ ] API endpoint улучшен
- [ ] Grafana dashboard готов

### AC-004: Quality Standards
- [ ] Zero linter warnings
- [ ] Zero race conditions
- [ ] 95%+ test coverage
- [ ] Comprehensive documentation
- [ ] Performance benchmarks passed

---

## 9. Success Metrics

### SM-001: Functional Success
- ✅ Mode detection accuracy: 99.9%
- ✅ Transition detection latency: <5s
- ✅ Zero data loss during transitions
- ✅ Automatic recovery working

### SM-002: Performance Success
- ✅ Mode check overhead: <1µs
- ✅ API response time: <10ms
- ✅ Memory overhead: <500KB
- ✅ CPU overhead: <0.1%

### SM-003: Quality Success
- ✅ Zero linter warnings
- ✅ Zero race conditions
- ✅ 95%+ test coverage
- ✅ Comprehensive documentation
- ✅ Grade A+ certification

### SM-004: Production Readiness
- ✅ All tests passing
- ✅ Benchmarks passed
- ✅ Documentation complete
- ✅ Production-approved
- ✅ Ready for deployment

---

**Requirements Date**: 2025-01-13
**Author**: AI Assistant
**Status**: ✅ Requirements Complete, Ready for Design
