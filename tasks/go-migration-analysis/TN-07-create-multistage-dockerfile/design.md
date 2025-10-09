# TN-07: Design - Сформировать multi-stage Dockerfile

## Архитектура Dockerfile

### Multi-Stage Build Structure
```
Build Stage (golang:1.21-alpine)
├── Install dependencies
├── Download Go modules
├── Compile optimized binary
└── Strip debug symbols

Runtime Stage (scratch)
├── Copy CA certificates
├── Copy binary from build stage
├── Set non-root user
├── Expose port 8080
└── Set health check
```

### Build Optimization
- **Go Build Flags**: `-ldflags="-w -s -X main.version=${VERSION}"`
- **CGO**: Disabled для статической линковки
- **Optimization**: Strip symbols и debug info
- **Compression**: UPX для дополнительного сжатия (опционально)

### Security Features
- **Non-root user**: uid=65534 (nobody)
- **Read-only filesystem**: где возможно
- **Minimal base image**: scratch для минимальной attack surface
- **No shell**: предотвращает shell injection

### Health Check Integration
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1
```

### Environment Variables
- `PORT`: HTTP server port (default: 8080)
- `VERSION`: Application version для build info

### Performance Optimizations
- **Layer Caching**: Go modules в отдельном layer
- **Multi-platform**: linux/amd64, linux/arm64
- **Binary size**: < 10MB финальный образ
- **Build time**: < 2 минуты при повторных сборках

## Dockerfile Structure

### Build Stage
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o server ./cmd/server
```

### Runtime Stage
```dockerfile
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /server
USER 65534:65534
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "--health-check"]
ENTRYPOINT ["/server"]
```

## Build & Deploy Process
1. **Build**: `docker build -t alert-history:latest .`
2. **Test**: `docker run -p 8080:8080 alert-history:latest`
3. **Health check**: `curl http://localhost:8080/healthz`
4. **Registry**: Push to container registry

## Alternative Approaches
- **Distroless**: gcr.io/distroless/static вместо scratch
- **Alpine runtime**: Для debugging capabilities
- **Multi-arch**: Buildx для ARM64 support

## Monitoring & Observability
- **Logs**: Structured JSON to stdout/stderr
- **Metrics**: Prometheus endpoint (future)
- **Health checks**: Kubernetes readiness/liveness probes
- **Tracing**: OpenTelemetry (future)
