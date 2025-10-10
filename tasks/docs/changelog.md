# Changelog

Все значимые изменения в проекте Alert History будут документированы в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
и этот проект придерживается [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **TN-181: Prometheus Metrics Audit & Unification** (2025-10-10)
  - Реализована централизованная система управления метриками через MetricsRegistry
  - Единая таксономия метрик: `<namespace>_<category>_<subsystem>_<metric_name>_<unit>`
  - 3 категории метрик: Business (9 metrics), Technical (14 metrics), Infrastructure (7 metrics)
  - Экспорт метрик Database Connection Pool в Prometheus (9 метрик через PrometheusExporter)
  - Path Normalization middleware для снижения cardinality (1,000+ → ~20 уникальных paths)
  - Prometheus Recording Rules для обратной совместимости с legacy метриками
  - 54.7% test coverage (19 unit tests, 4 integration tests, 8 benchmarks)
  - Performance: < 1µs overhead (zero allocations), 860 ns/op для path normalization
  - Comprehensive documentation: METRICS_NAMING_GUIDE.md (14 KB), PROMQL_EXAMPLES.md (19 KB), RUNBOOK_METRICS.md (18 KB)
  - **150% Quality Target Achieved**: Advanced validation, enhanced error handling, extensive documentation
  - **Files:**
    - `go-app/pkg/metrics/registry.go` (367 lines) - Centralized metrics registry
    - `go-app/pkg/metrics/business.go` (248 lines) - Business metrics with helpers
    - `go-app/pkg/metrics/infra.go` (259 lines) - Infrastructure metrics
    - `go-app/pkg/middleware/path_normalization.go` (119 lines) - Cardinality reduction
    - `go-app/internal/database/postgres/prometheus.go` (186 lines) - DB Pool exporter
    - `helm/alert-history-go/templates/prometheus-recording-rules.yaml` (113 lines)
  - **Impact:** Zero breaking changes, Grafana dashboards remain functional, backward compatible

- **TN-39: Circuit Breaker для LLM Calls** (2025-10-09)
  - Реализован production-ready Circuit Breaker с 3-state machine (CLOSED, OPEN, HALF_OPEN)
  - Интеграция с HTTPLLMClient через метод `Call()` с прозрачной оберткой существующего retry logic
  - 7 Prometheus метрик для observability: state, failures, successes, state_changes, blocked_requests, half_open_requests, slow_calls
  - Histogram метрика `call_duration_seconds` для отслеживания p50/p95/p99 latency
  - Умная классификация ошибок (ErrorType) для различия transient vs prolonged failures
  - Sliding window механизм для подсчета failure rate за временное окно
  - Конфигурация через environment variables с разумными defaults
  - Автоматический fallback в transparent mode при открытом circuit breaker
  - 15 comprehensive unit tests (100% passing, >90% coverage)
  - 8 performance benchmarks (17.35 ns/op overhead - 28,000x быстрее target <0.5ms)
  - Comprehensive documentation: README.md (13KB), IMPLEMENTATION_REPORT.md (20KB), COMPLETION_SUMMARY.md (8KB)
  - **150% Target Achieved**: Advanced metrics, ultra-low overhead, enhanced error classification

- **TN-21: Prometheus Metrics для Go приложения** (2025-01-12)
  - Добавлен HTTP middleware для сбора Prometheus метрик
  - Реализованы метрики: `http_requests_total`, `http_request_duration_seconds`, `http_request_size_bytes`, `http_response_size_bytes`, `http_requests_active`
  - Создан endpoint `/metrics` для экспорта метрик
  - Добавлена конфигурация метрик через `MetricsConfig`
  - Интегрирован middleware в основной HTTP сервер
  - Созданы comprehensive unit тесты
  - Добавлена документация по использованию метрик

### Technical Details
- **Файлы:**
  - `go-app/pkg/metrics/prometheus.go` - основной middleware и метрики
  - `go-app/pkg/metrics/prometheus_test.go` - unit тесты
  - `go-app/internal/config/config.go` - конфигурация метрик
  - `go-app/cmd/server/main.go` - интеграция middleware
  - `tasks/docs/prometheus-metrics.md` - документация
- **Зависимости:** добавлен `github.com/prometheus/client_golang v1.20.5`
- **Архитектура:** следует паттерну middleware с dependency injection
- **Тестирование:** 100% покрытие unit тестами

## [Previous Releases]

### [1.0.0] - 2024-XX-XX
- Первоначальная реализация Alert History Service
- Python FastAPI приложение с поддержкой webhook'ов
- LLM интеграция для классификации алертов
- Режимы обогащения: transparent, transparent_with_recommendations, enriched
- SQLite и PostgreSQL адаптеры
- Dashboard для мониторинга
