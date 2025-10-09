# TN-20: Дизайн логирования

## Архитектура

Пакет `pkg/logger` предоставляет единый интерфейс для структурированного логирования с использованием `log/slog` из Go 1.21+.

### Основные компоненты

1. **Config** - конфигурация логгера
2. **NewLogger** - фабрика для создания логгера
3. **LoggingMiddleware** - HTTP middleware для net/http
4. **Request ID** - генерация и протаскивание через контекст

## API

### Конфигурация
```go
type Config struct {
    Level      string // debug, info, warn, error
    Format     string // json, text
    Output     string // stdout, stderr, file
    Filename   string // путь к файлу (для output=file)
    MaxSize    int    // максимальный размер файла в MB
    MaxBackups int    // количество backup файлов
    MaxAge     int    // максимальный возраст файлов в днях
    Compress   bool   // сжимать ли backup файлы
}
```

### Инициализация
```go
func NewLogger(cfg Config) *slog.Logger
func ParseLevel(level string) slog.Level
func SetupWriter(cfg Config) io.Writer
```

### HTTP Middleware для net/http
```go
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Генерация request ID
            requestID := r.Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = GenerateRequestID()
            }

            // Добавление в контекст и заголовок ответа
            ctx := WithRequestID(r.Context(), requestID)
            r = r.WithContext(ctx)
            w.Header().Set("X-Request-ID", requestID)

            // Обертка для захвата status code
            wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

            next.ServeHTTP(wrapped, r)

            // Логирование запроса
            logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", wrapped.statusCode,
                "duration", time.Since(start),
                "request_id", requestID,
                "remote_addr", r.RemoteAddr,
                "user_agent", r.UserAgent(),
            )
        })
    }
}
```

### Request ID
```go
func GenerateRequestID() string
func WithRequestID(ctx context.Context, requestID string) context.Context
func GetRequestID(ctx context.Context) string
func FromContext(ctx context.Context, logger *slog.Logger) *slog.Logger
```

## Интеграция с конфигурацией

Пакет интегрируется с `internal/config.LogConfig`:

```go
appLogger := logger.NewLogger(logger.Config{
    Level:      cfg.Log.Level,
    Format:     cfg.Log.Format,
    Output:     cfg.Log.Output,
    Filename:   cfg.Log.Filename,
    MaxSize:    cfg.Log.MaxSize,
    MaxBackups: cfg.Log.MaxBackups,
    MaxAge:     cfg.Log.MaxAge,
    Compress:   cfg.Log.Compress,
})
```

## Формат логов

### JSON (production)
```json
{
  "time": "2025-09-12T15:45:00Z",
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/healthz",
  "status": 200,
  "duration": "1.234ms",
  "request_id": "req_a1b2c3d4e5f6g7h8",
  "remote_addr": "127.0.0.1:12345",
  "user_agent": "curl/7.68.0"
}
```

### Text (development)
```
time=2025-09-12T15:45:00Z level=INFO msg=request method=GET path=/healthz status=200 duration=1.234ms request_id=req_a1b2c3d4e5f6g7h8
```

## Ротация файлов

При `output=file` используется lumberjack для ротации:
- Максимальный размер файла (MaxSize)
- Количество backup файлов (MaxBackups)
- Максимальный возраст файлов (MaxAge)
- Сжатие backup файлов (Compress)
