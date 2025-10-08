# Prometheus Metrics для Go Alert History Service

## 📊 Обзор

В рамках задачи TN-21 была реализована система сбора HTTP метрик с использованием Prometheus для Go приложения Alert History Service. Middleware автоматически собирает метрики для всех HTTP запросов.

## 🎯 Реализованные метрики

### Основные метрики

1. **`http_requests_total`** (Counter)
   - Общее количество HTTP запросов
   - Labels: `method`, `path`, `status_code`

2. **`http_request_duration_seconds`** (Histogram)
   - Время выполнения HTTP запросов в секундах
   - Labels: `method`, `path`, `status_code`
   - Buckets: 0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0

### Дополнительные метрики

3. **`http_request_size_bytes`** (Histogram)
   - Размер входящих HTTP запросов в байтах
   - Labels: `method`, `path`

4. **`http_response_size_bytes`** (Histogram)
   - Размер исходящих HTTP ответов в байтах
   - Labels: `method`, `path`, `status_code`

5. **`http_requests_active`** (Gauge)
   - Количество активных HTTP запросов в данный момент
   - Labels: `method`, `path`

## 🔧 Архитектура

### Компоненты

- **`pkg/metrics/prometheus.go`** - основной middleware и метрики
- **`internal/config/config.go`** - конфигурация метрик
- **`cmd/server/main.go`** - интеграция middleware в HTTP сервер

### Структуры

```go
// HTTPMetrics содержит все Prometheus метрики для HTTP
type HTTPMetrics struct {
    requestsTotal     *prometheus.CounterVec
    requestDuration   *prometheus.HistogramVec
    requestSize       *prometheus.HistogramVec
    responseSize      *prometheus.HistogramVec
    activeRequests    *prometheus.GaugeVec
}

// MetricsManager управляет конфигурацией и жизненным циклом метрик
type MetricsManager struct {
    config  *config.MetricsConfig
    metrics *HTTPMetrics
}
```

## 🚀 Использование

### Конфигурация

Метрики настраиваются через структуру конфигурации:

```go
type MetricsConfig struct {
    Enabled bool   `json:"enabled" default:"true"`
    Path    string `json:"path" default:"/metrics"`
}
```

### Интеграция в HTTP сервер

```go
// Создание менеджера метрик
metricsManager := metrics.NewMetricsManager(cfg.Metrics)

// Получение middleware
metricsMiddleware := metricsManager.HTTPMiddleware()

// Интеграция в HTTP сервер
http.Handle("/metrics", promhttp.Handler())
http.Handle("/", metricsMiddleware(yourHandler))
```

### Endpoint метрик

Метрики доступны по адресу: `http://localhost:8080/metrics`

## 📈 Примеры метрик

```prometheus
# Общее количество запросов
http_requests_total{method="GET",path="/api/alerts",status_code="200"} 42

# Время выполнения запросов
http_request_duration_seconds_bucket{method="GET",path="/api/alerts",status_code="200",le="0.1"} 35
http_request_duration_seconds_sum{method="GET",path="/api/alerts",status_code="200"} 2.1
http_request_duration_seconds_count{method="GET",path="/api/alerts",status_code="200"} 42

# Размер запросов
http_request_size_bytes_bucket{method="POST",path="/api/alerts",le="1024"} 15
http_request_size_bytes_sum{method="POST",path="/api/alerts"} 12345
http_request_size_bytes_count{method="POST",path="/api/alerts"} 15

# Активные запросы
http_requests_active{method="GET",path="/api/alerts"} 3
```

## 🧪 Тестирование

### Unit тесты

Созданы comprehensive unit тесты в `pkg/metrics/prometheus_test.go`:

- Тест создания метрик
- Тест middleware функциональности
- Тест сбора метрик
- Тест response writer wrapper

### Запуск тестов

```bash
cd go-app
go test ./pkg/metrics/... -v
```

### Проверка компиляции

```bash
cd go-app
go build ./cmd/server
```

## 🔍 Мониторинг и алерты

### Полезные запросы PromQL

```promql
# Rate запросов в секунду
rate(http_requests_total[5m])

# 95-й перцентиль времени ответа
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Количество ошибок 5xx
sum(rate(http_requests_total{status_code=~"5.."}[5m]))

# Средний размер ответа
rate(http_response_size_bytes_sum[5m]) / rate(http_response_size_bytes_count[5m])
```

### Рекомендуемые алерты

1. **Высокий rate ошибок**
   ```promql
   rate(http_requests_total{status_code=~"5.."}[5m]) > 0.1
   ```

2. **Медленные запросы**
   ```promql
   histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1.0
   ```

3. **Много активных запросов**
   ```promql
   sum(http_requests_active) > 100
   ```

## 📝 Конфигурация Prometheus

### prometheus.yml

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'alert-history-go'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

## 🔧 Переменные окружения

```bash
# Включение/отключение метрик
METRICS_ENABLED=true

# Путь для endpoint метрик
METRICS_PATH=/metrics

# Порт сервера
SERVER_PORT=8080
```

## 📚 Дополнительные ресурсы

- [Prometheus Go Client Documentation](https://pkg.go.dev/github.com/prometheus/client_golang)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [HTTP Metrics Best Practices](https://prometheus.io/docs/practices/instrumentation/#http)

## 🎯 Следующие шаги

1. Интеграция с Grafana для визуализации
2. Добавление business-метрик (alerts processed, etc.)
3. Настройка алертов в Alertmanager
4. Добавление метрик для database операций
