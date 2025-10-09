# TN-18: Docker Compose конфигурация

## docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: alerthistory
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: dev
    ports:
      - "5432:5432"
    volumes:
      - ./migrations:/docker-entrypoint-initdb.d

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  app:
    build:
      context: go-app
      target: builder
    volumes:
      - ./go-app:/app
    environment:
      DATABASE_URL: postgres://dev:dev@postgres/alerthistory
      REDIS_URL: redis://redis:6379
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    command: air
```
