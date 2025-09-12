# TN-06: Requirements - Создать минимальный main.go с /healthz

## Обоснование
Необходим минимальный HTTP сервер с health check endpoint для:
- Kubernetes liveness/readiness probes
- Load balancer health checks
- Service discovery
- Monitoring systems

## Требования к функциональности
- HTTP сервер на порту 8080 (стандартный для Go приложений)
- GET /healthz endpoint возвращающий 200 OK
- JSON ответ с базовой информацией о сервисе
- Graceful shutdown support
- Structured logging

## Технические требования
- Использовать стандартный net/http
- JSON response format: `{"status": "ok", "service": "alert-history", "version": "1.0.0"}`
- Response headers: Content-Type: application/json
- Timeout: 30 секунд для health check
- Logging: structured JSON logs

## Критерии готовности
- [ ] HTTP сервер запускается на порту 8080
- [ ] curl http://localhost:8080/healthz возвращает 200 OK
- [ ] JSON response содержит status, service, version
- [ ] Graceful shutdown работает (Ctrl+C)
- [ ] Логирование включено
- [ ] Тесты проходят (make test)
- [ ] Код проходит линтер (make lint)
