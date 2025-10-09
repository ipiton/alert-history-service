# TN-07: Requirements - Сформировать multi-stage Dockerfile

## Обоснование
Необходим оптимизированный Dockerfile для production deployment Go приложения в Kubernetes с минимальным размером образа и максимальной безопасностью.

## Требования к функциональности
- Multi-stage build для оптимизации размера образа
- Non-root user для безопасности
- Health check инструкции для Kubernetes
- Proper Go build с оптимизациями
- Minimal runtime image (scratch или distroless)
- Support для multi-architecture builds

## Технические требования
- **Build stage**: golang:1.21-alpine для компиляции
- **Runtime stage**: scratch или distroless для минимального размера
- **Security**: non-root user (uid/gid 65534)
- **Build flags**: `-ldflags="-w -s"` для оптимизации
- **CGO**: `CGO_ENABLED=0` для статической линковки
- **Architecture**: linux/amd64, linux/arm64
- **Health check**: CMD для health check endpoint

## Безопасность
- Non-root container execution
- Minimal attack surface (scratch base image)
- No shell in runtime container
- Read-only filesystem где возможно

## Оптимизация
- Multi-stage build для уменьшения размера финального образа
- Layer caching для ускорения rebuilds
- Go modules caching в Docker layer
- Optimized binary size с strip symbols

## Критерии готовности
- [ ] Dockerfile собирается без ошибок
- [ ] Финальный образ < 10MB
- [ ] Non-root user (nobody:nobody)
- [ ] Health check работает в контейнере
- [ ] Multi-stage build правильно реализован
- [ ] Go приложение запускается в контейнере
