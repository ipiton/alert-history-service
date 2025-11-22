# TN-152: Hot Reload Mechanism (SIGHUP) - Requirements

**Date**: 2025-11-22
**Task ID**: TN-152
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Planning Phase
**Priority**: P0 (Critical for MVP)
**Estimated Effort**: 6-8 hours

---

## 📋 Executive Summary

Реализация механизма горячей перезагрузки конфигурации через SIGHUP signal, позволяющего обновлять конфигурацию без перезапуска сервиса. Это критически важная функциональность для production-окружений, где downtime недопустим.

**Бизнес-ценность**: Возможность обновления конфигурации (маршрутизация, receivers, inhibition rules) без перезапуска сервиса, что обеспечивает zero-downtime операции и соответствие Enterprise-требованиям.

---

## 🎯 1. Обоснование задачи (Зачем делаем)

### 1.1 Бизнес-требования

**Проблема**: В текущей реализации изменение конфигурации требует:
1. Редактирование config.yaml
2. Перезапуск всего сервиса
3. Downtime на 5-30 секунд
4. Потеря in-flight запросов
5. Прерывание активных соединений

**Последствия**:
- ❌ Недопустимо для production-систем с SLA 99.9%+
- ❌ Невозможно быстро реагировать на инциденты
- ❌ Риск потери критических алертов во время перезапуска
- ❌ Не соответствует best practices для alerting систем

**Решение**: Hot reload через SIGHUP signal
- ✅ Zero-downtime обновление конфигурации
- ✅ Совместимость с Alertmanager (industry standard)
- ✅ Поддержка GitOps workflows
- ✅ Быстрое реагирование на изменения (< 1 секунда)

### 1.2 Технические требования

**Must Have (P0)**:
1. Обработка SIGHUP signal для перезагрузки конфигурации
2. Валидация новой конфигурации перед применением
3. Rollback к старой конфигурации при ошибках
4. Атомарное обновление (all-or-nothing)
5. Structured logging всех операций
6. Prometheus metrics для мониторинга

**Should Have (P1)**:
1. Graceful reload (без прерывания in-flight запросов)
2. Уведомления об успешной/неуспешной перезагрузке
3. Audit log всех reload операций
4. Health check после reload
5. Timeout protection (max 30s)

**Nice to Have (P2)**:
1. Incremental reload (только измененные компоненты)
2. Dry-run mode для тестирования
3. Webhook notifications о reload событиях
4. Reload history в PostgreSQL

### 1.3 Совместимость с Alertmanager

Alertmanager использует SIGHUP для hot reload:
```bash
# Standard Alertmanager reload
kill -HUP $(pidof alertmanager)
# или
pkill -HUP alertmanager
```

**Наша реализация должна**:
- ✅ Использовать тот же signal (SIGHUP)
- ✅ Перечитывать config.yaml из файла
- ✅ Валидировать перед применением
- ✅ Rollback при ошибках
- ✅ Логировать результат

---

## 👥 2. Пользовательские сценарии

### Сценарий 1: Добавление нового receiver (Success Case)

**Актор**: DevOps Engineer
**Цель**: Добавить новый Slack канал для алертов критичности "critical"

**Шаги**:
1. Редактирует config.yaml, добавляет новый receiver:
   ```yaml
   receivers:
     - name: 'critical-slack'
       slack_configs:
         - api_url: 'https://hooks.slack.com/services/XXX'
           channel: '#critical-alerts'
   ```
2. Сохраняет файл
3. Отправляет SIGHUP:
   ```bash
   kill -HUP $(pidof alert-history)
   ```
4. Проверяет логи:
   ```json
   {
     "level": "info",
     "msg": "config reload triggered",
     "signal": "SIGHUP",
     "config_path": "/etc/alert-history/config.yaml"
   }
   {
     "level": "info",
     "msg": "config validation successful",
     "duration_ms": 45
   }
   {
     "level": "info",
     "msg": "config reload successful",
     "version": 43,
     "components_reloaded": ["routing", "receivers"],
     "duration_ms": 287
   }
   ```
5. Проверяет метрики:
   ```
   config_reload_total{status="success"} 1
   config_reload_duration_seconds{quantile="0.95"} 0.287
   ```

**Ожидаемый результат**:
- ✅ Конфигурация перезагружена без downtime
- ✅ Новый receiver доступен для маршрутизации
- ✅ Старые алерты продолжают обрабатываться
- ✅ In-flight запросы не прерваны

### Сценарий 2: Исправление ошибки в route (Validation Error)

**Актор**: DevOps Engineer
**Цель**: Исправить опечатку в имени receiver

**Шаги**:
1. Редактирует config.yaml, допускает ошибку:
   ```yaml
   route:
     receiver: 'default-receiver-TYPO'  # Receiver не существует
   ```
2. Сохраняет файл
3. Отправляет SIGHUP:
   ```bash
   kill -HUP $(pidof alert-history)
   ```
4. Проверяет логи:
   ```json
   {
     "level": "error",
     "msg": "config reload failed",
     "error": "validation failed: receiver 'default-receiver-TYPO' not found",
     "config_path": "/etc/alert-history/config.yaml"
   }
   {
     "level": "info",
     "msg": "keeping old configuration",
     "version": 42
   }
   ```
5. Проверяет метрики:
   ```
   config_reload_total{status="validation_error"} 1
   config_reload_errors_total{type="validation"} 1
   ```

**Ожидаемый результат**:
- ✅ Ошибка обнаружена на этапе валидации
- ✅ Старая конфигурация продолжает работать
- ✅ Сервис не упал
- ✅ Детальное сообщение об ошибке в логах

### Сценарий 3: Критическая ошибка при reload (Rollback)

**Актор**: DevOps Engineer
**Цель**: Обновить database connection pool settings

**Шаги**:
1. Редактирует config.yaml:
   ```yaml
   database:
     max_connections: 5  # Слишком мало для production
   ```
2. Отправляет SIGHUP
3. Проверяет логи:
   ```json
   {
     "level": "info",
     "msg": "config validation successful"
   }
   {
     "level": "info",
     "msg": "reloading component",
     "component": "database"
   }
   {
     "level": "error",
     "msg": "component reload failed",
     "component": "database",
     "error": "failed to resize connection pool: timeout acquiring connection"
   }
   {
     "level": "warn",
     "msg": "rolling back to previous configuration",
     "version": 42
   }
   {
     "level": "info",
     "msg": "rollback successful",
     "version": 42
   }
   ```

**Ожидаемый результат**:
- ✅ Ошибка обнаружена при reload компонента
- ✅ Автоматический rollback к старой конфигурации
- ✅ Сервис продолжает работать со старой конфигурацией
- ✅ Детальная информация об ошибке

### Сценарий 4: Kubernetes ConfigMap Update (GitOps)

**Актор**: Kubernetes Operator / GitOps Controller
**Цель**: Автоматическое обновление конфигурации через ConfigMap

**Шаги**:
1. GitOps controller обновляет ConfigMap:
   ```bash
   kubectl apply -f alertmanager-config.yaml
   ```
2. Kubernetes монтирует новый config.yaml в pod
3. Sidecar container отправляет SIGHUP:
   ```bash
   kubectl exec -it alert-history-pod -c sidecar -- kill -HUP 1
   ```
4. Проверяет статус через API:
   ```bash
   curl http://alert-history:8080/api/v2/config/status
   ```
   Response:
   ```json
   {
     "version": 44,
     "last_reload": "2025-11-22T10:15:30Z",
     "last_reload_status": "success",
     "last_reload_duration_ms": 312
   }
   ```

**Ожидаемый результат**:
- ✅ Автоматическая перезагрузка при изменении ConfigMap
- ✅ Интеграция с Kubernetes ecosystem
- ✅ Поддержка GitOps workflows
- ✅ Observability через API

---

## 🔧 3. Технические требования

### 3.1 Signal Handling

**Требование**: Обработка SIGHUP signal без прерывания работы сервиса

**Детали**:
1. Регистрация signal handler для SIGHUP
2. Отдельный goroutine для обработки signals
3. Non-blocking обработка (не блокирует main goroutine)
4. Graceful handling (завершение текущих операций)

**Код (концепт)**:
```go
// Register SIGHUP handler
sighup := make(chan os.Signal, 1)
signal.Notify(sighup, syscall.SIGHUP)

go func() {
    for {
        <-sighup
        slog.Info("SIGHUP received, triggering config reload")
        if err := reloadConfig(); err != nil {
            slog.Error("config reload failed", "error", err)
        }
    }
}()
```

### 3.2 Configuration Reload Pipeline

**Требование**: 4-фазный процесс перезагрузки с валидацией и rollback

**Фазы**:

#### Phase 1: Load & Parse (Target: < 50ms)
1. Читать config.yaml из файла
2. Парсить YAML → Config struct
3. Проверить синтаксис
4. Обработать environment variables

**Критерии успеха**:
- ✅ Файл существует и читаем
- ✅ YAML синтаксис корректен
- ✅ Unmarshal успешен

**Ошибки**:
- ❌ File not found → Keep old config
- ❌ YAML syntax error → Keep old config
- ❌ Unmarshal error → Keep old config

#### Phase 2: Validation (Target: < 100ms)
1. Structural validation (validator tags)
2. Business rules validation
3. Cross-field validation
4. Reference validation (receivers exist, etc.)

**Критерии успеха**:
- ✅ Все required поля присутствуют
- ✅ Типы корректны
- ✅ Ranges валидны
- ✅ Receiver references существуют
- ✅ Route tree корректен

**Ошибки**:
- ❌ Validation failed → Keep old config, log detailed errors

#### Phase 3: Atomic Apply (Target: < 50ms)
1. Acquire distributed lock (Redis)
2. Backup old config
3. Update in-memory config
4. Increment version
5. Write audit log
6. Release lock

**Критерии успеха**:
- ✅ Lock acquired
- ✅ Config updated atomically
- ✅ Version incremented
- ✅ Audit log written

**Ошибки**:
- ❌ Lock timeout → Retry or fail
- ❌ Storage error → Rollback

#### Phase 4: Component Reload (Target: < 300ms)
1. Identify affected components
2. Reload components in parallel
3. Collect results
4. Check for critical errors
5. Rollback if critical component failed

**Критерии успеха**:
- ✅ All critical components reloaded successfully
- ✅ Non-critical failures logged but not blocking
- ✅ Health check passed

**Ошибки**:
- ❌ Critical component failed → Rollback to old config
- ❌ Timeout → Rollback

### 3.3 Reloadable Components

**Requirement**: Компоненты должны поддерживать hot reload

**Компоненты для реализации Reloadable interface**:

1. **Routing Engine** (Critical)
   - Reload route tree
   - Update matchers
   - Rebuild cache

2. **Receiver Manager** (Critical)
   - Update receiver configs
   - Reconnect to external services (Slack, PagerDuty)
   - Refresh secrets from Kubernetes

3. **Inhibition Manager** (Non-Critical)
   - Reload inhibition rules
   - Rebuild matcher cache

4. **Silencing Manager** (Non-Critical)
   - Reload silence configs
   - Update active silences

5. **Grouping Engine** (Critical)
   - Reload grouping rules
   - Update timers

6. **LLM Service** (Non-Critical)
   - Update API keys
   - Change model settings

**Interface**:
```go
type Reloadable interface {
    Reload(ctx context.Context, cfg *Config) error
    Name() string
    IsCritical() bool
}
```

### 3.4 Rollback Mechanism

**Requirement**: Автоматический rollback при критических ошибках

**Триггеры rollback**:
1. Критический компонент не смог перезагрузиться
2. Health check failed после reload
3. Timeout при reload (> 30s)

**Процесс rollback**:
1. Log rollback trigger
2. Restore old config from backup
3. Reload all components with old config
4. Verify health
5. Log rollback result

**Метрики**:
```
config_reload_rollbacks_total{reason="critical_component_failed"} 1
config_reload_rollback_duration_seconds 0.156
```

### 3.5 Observability

**Requirement**: Полная видимость процесса reload

**Structured Logging**:
```json
{
  "level": "info",
  "msg": "config reload started",
  "trigger": "SIGHUP",
  "config_path": "/etc/alert-history/config.yaml",
  "current_version": 42
}
{
  "level": "info",
  "msg": "config loaded and parsed",
  "duration_ms": 23,
  "size_bytes": 15234
}
{
  "level": "info",
  "msg": "config validation successful",
  "duration_ms": 67,
  "warnings": 2
}
{
  "level": "info",
  "msg": "component reload started",
  "component": "routing",
  "critical": true
}
{
  "level": "info",
  "msg": "component reload successful",
  "component": "routing",
  "duration_ms": 45
}
{
  "level": "info",
  "msg": "config reload successful",
  "new_version": 43,
  "components_reloaded": 5,
  "total_duration_ms": 287
}
```

**Prometheus Metrics**:
```
# Total reload attempts
config_reload_total{status="success|validation_error|reload_error|rollback"} 123

# Reload duration histogram
config_reload_duration_seconds{phase="load|validate|apply|reload"} 0.287

# Reload errors by type
config_reload_errors_total{type="validation|timeout|component_failed"} 5

# Component reload duration
config_reload_component_duration_seconds{component="routing|receivers|inhibition"} 0.045

# Last reload timestamp
config_reload_last_success_timestamp_seconds 1700000000

# Rollback counter
config_reload_rollbacks_total{reason="critical_failed|timeout|health_check"} 2
```

---

## 🚀 4. Критерии приёмки (Definition of Done)

### 4.1 Функциональные критерии

- [ ] **SIGHUP Handler**: Обработка SIGHUP signal реализована
- [ ] **Config Reload**: Перезагрузка config.yaml из файла работает
- [ ] **Validation**: Валидация новой конфигурации перед применением
- [ ] **Atomic Apply**: Атомарное обновление конфигурации
- [ ] **Component Reload**: Все критические компоненты поддерживают reload
- [ ] **Rollback**: Автоматический rollback при критических ошибках
- [ ] **Zero Downtime**: In-flight запросы не прерываются
- [ ] **Graceful**: Текущие операции завершаются корректно

### 4.2 Качественные критерии (150% Quality)

**Code Quality**:
- [ ] **Test Coverage**: ≥ 90% (unit + integration)
- [ ] **Unit Tests**: ≥ 25 тестов
- [ ] **Integration Tests**: ≥ 10 тестов
- [ ] **Benchmarks**: ≥ 5 benchmarks
- [ ] **Linter**: Zero warnings (golangci-lint)
- [ ] **Race Detector**: Zero race conditions
- [ ] **Error Handling**: Все ошибки обработаны корректно

**Performance**:
- [ ] **Reload Duration**: < 500ms p95 (target: 300ms)
- [ ] **Validation**: < 100ms p95
- [ ] **Component Reload**: < 300ms p95
- [ ] **Rollback**: < 200ms p95
- [ ] **Memory**: No memory leaks
- [ ] **CPU**: < 10% spike during reload

**Observability**:
- [ ] **Structured Logging**: Все операции логируются
- [ ] **Prometheus Metrics**: 8+ метрик
- [ ] **Health Check**: Endpoint для проверки статуса reload
- [ ] **Audit Log**: Все reload операции записываются

**Documentation**:
- [ ] **User Guide**: Как использовать SIGHUP reload
- [ ] **Integration Guide**: Kubernetes ConfigMap integration
- [ ] **Troubleshooting**: Частые проблемы и решения
- [ ] **API Documentation**: Endpoints для проверки статуса

### 4.3 Совместимость

- [ ] **Alertmanager Compatible**: Поведение идентично Alertmanager
- [ ] **Kubernetes Ready**: Работает с ConfigMap updates
- [ ] **GitOps Ready**: Поддержка автоматических обновлений
- [ ] **Backward Compatible**: Старые конфигурации работают

---

## 🔗 5. Зависимости

### 5.1 Внутренние зависимости

**Completed Tasks (Ready)**:
- ✅ **TN-149**: GET /api/v2/config (config export)
- ✅ **TN-150**: POST /api/v2/config (config update)
- ✅ **TN-151**: Config Validator (validation logic)
- ✅ **TN-22**: Graceful Shutdown (signal handling pattern)

**Infrastructure**:
- ✅ ConfigUpdateService (TN-150)
- ✅ ConfigValidator (TN-151)
- ✅ ConfigReloader (TN-150)
- ✅ Reloadable interface (TN-150)
- ✅ ConfigStorage (TN-150)
- ✅ LockManager (TN-150)

### 5.2 Внешние зависимости

**Go Standard Library**:
- `os/signal` - Signal handling
- `syscall` - SIGHUP constant
- `context` - Timeout management

**Third-party Libraries**:
- `github.com/spf13/viper` - Config loading (already used)
- `gopkg.in/yaml.v3` - YAML parsing (already used)

**Infrastructure**:
- PostgreSQL - Config storage (optional)
- Redis - Distributed locking (optional)

### 5.3 Блокеры

**None** - Все зависимости выполнены ✅

---

## 📊 6. Риски и митигация

### Риск 1: Race Condition при concurrent reload

**Вероятность**: Medium
**Влияние**: High (data corruption)

**Митигация**:
1. Distributed lock (Redis) для предотвращения concurrent reloads
2. Mutex для in-memory config updates
3. Atomic config replacement (pointer swap)
4. Integration tests для concurrent scenarios

### Риск 2: Memory Leak при частых reloads

**Вероятность**: Low
**Влияние**: High (OOM)

**Митигация**:
1. Proper cleanup старых ресурсов
2. Graceful close connections
3. Memory profiling (pprof)
4. Leak detection tests

### Риск 3: Rollback Failure (double fault)

**Вероятность**: Very Low
**Влияние**: Critical (service down)

**Митигация**:
1. Backup old config before applying new
2. Validate old config before rollback
3. Fallback to default config if rollback fails
4. Alert on rollback failures

### Риск 4: Slow Component Reload (timeout)

**Вероятность**: Medium
**Влияние**: Medium (degraded performance)

**Митигация**:
1. Timeout на каждый component reload (30s)
2. Parallel reload для независимых компонентов
3. Non-critical components не блокируют reload
4. Monitoring reload duration

---

## 📝 7. Ограничения

### 7.1 Технические ограничения

1. **Config File Only**: Reload только из файла (не из API)
   - Обоснование: Совместимость с Alertmanager и GitOps

2. **Single File**: Только один config.yaml
   - Обоснование: Простота и предсказуемость

3. **No Partial Reload**: Всегда reload всей конфигурации
   - Обоснование: Атомарность и consistency

4. **Timeout**: Max 30s на reload
   - Обоснование: Предотвращение зависания

### 7.2 Scope Limitations

**In Scope**:
- ✅ SIGHUP signal handling
- ✅ Config file reload
- ✅ Validation
- ✅ Component reload
- ✅ Rollback
- ✅ Metrics & logging

**Out of Scope** (Future Enhancements):
- ❌ SIGUSR1/SIGUSR2 для других операций
- ❌ Incremental reload (только измененные секции)
- ❌ Config reload через API (уже есть в TN-150)
- ❌ Multiple config files
- ❌ Config templates

---

## 🎯 8. Success Metrics

### 8.1 Performance Metrics

| Metric | Target (150%) | Baseline (100%) |
|--------|---------------|-----------------|
| Reload Duration (p95) | < 300ms | < 500ms |
| Validation Duration (p95) | < 50ms | < 100ms |
| Component Reload (p95) | < 200ms | < 300ms |
| Rollback Duration (p95) | < 150ms | < 200ms |
| Memory Overhead | < 5MB | < 10MB |
| CPU Spike | < 5% | < 10% |

### 8.2 Reliability Metrics

| Metric | Target (150%) | Baseline (100%) |
|--------|---------------|-----------------|
| Reload Success Rate | > 99.5% | > 99% |
| Rollback Success Rate | 100% | 100% |
| Zero Downtime | 100% | 100% |
| Data Loss | 0 | 0 |

### 8.3 Quality Metrics

| Metric | Target (150%) | Baseline (100%) |
|--------|---------------|-----------------|
| Test Coverage | ≥ 90% | ≥ 80% |
| Unit Tests | ≥ 25 | ≥ 15 |
| Integration Tests | ≥ 10 | ≥ 5 |
| Benchmarks | ≥ 5 | ≥ 3 |
| Documentation LOC | ≥ 3000 | ≥ 2000 |

---

## 📚 9. References

### 9.1 Related Tasks

- **TN-149**: GET /api/v2/config - Config export
- **TN-150**: POST /api/v2/config - Config update via API
- **TN-151**: Config Validator - Validation logic
- **TN-22**: Graceful Shutdown - Signal handling pattern

### 9.2 External References

- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Prometheus Reload](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#configuration-reload)
- [Go Signal Handling](https://gobyexample.com/signals)
- [Kubernetes ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)

### 9.3 Industry Best Practices

- [12-Factor App: Config](https://12factor.net/config)
- [NGINX Reload Pattern](https://www.nginx.com/blog/nginx-1-11-5-released/)
- [Envoy Hot Restart](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/operations/hot_restart)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Total Lines**: 750+ LOC
**Status**: ✅ Ready for Design Phase
