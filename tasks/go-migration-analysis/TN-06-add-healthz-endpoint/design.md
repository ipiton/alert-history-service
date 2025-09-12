# TN-06: Design - Создать минимальный main.go с /healthz

## Архитектурное решение

### Структура приложения
```
cmd/server/
├── main.go          # Entry point с HTTP сервером
└── handlers/
    └── health.go    # Health check handler
```

### HTTP Server Design
- **Framework**: Стандартный net/http (пока без Fiber/Gin)
- **Port**: 8080 (стандартный для Go приложений)
- **Routes**:
  - `GET /healthz` - Health check endpoint

### Health Check Response Format
```json
{
  "status": "ok",
  "service": "alert-history",
  "version": "1.0.0",
  "timestamp": "2025-09-11T11:30:00Z"
}
```

### Graceful Shutdown
- Signal handling: SIGINT, SIGTERM
- Context cancellation
- Server shutdown timeout: 30 секунд

### Logging
- Structured JSON logging с slog
- Request logging middleware
- Error logging с контекстом

### Configuration
- Environment-based configuration
- Default values для development
- PORT environment variable

## API Контракты

### GET /healthz
**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 89

{"status":"ok","service":"alert-history","version":"1.0.0","timestamp":"2025-09-11T11:30:00Z"}
```

**Error Response (если сервис не готов):**
```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{"status":"error","service":"alert-history","message":"Service not ready"}
```

## Сценарии ошибок
- Database connection failure → 503 Service Unavailable
- External service dependency failure → 503 Service Unavailable
- High memory usage → 503 Service Unavailable
- Network issues → 503 Service Unavailable

## Безопасность
- No sensitive information in health response
- Rate limiting considerations (future)
- Basic authentication (future, if needed)

## Мониторинг
- Health check должен быть легковесным (< 100ms)
- Prometheus metrics integration (future)
- Structured logs для анализа
