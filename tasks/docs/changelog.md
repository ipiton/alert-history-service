# Changelog

Все значимые изменения в проекте Alert History будут документированы в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
и этот проект придерживается [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
