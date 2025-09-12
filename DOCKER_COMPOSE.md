# Docker Compose Development Environment

Этот документ описывает настройку и использование Docker Compose для локальной разработки Alert History Service.

## 🚀 Быстрый старт

### 1. Настройка окружения

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd AlertHistory

# Скопируйте файл с переменными окружения
cp env.example .env

# Запустите полную настройку
make dev-setup
```

### 2. Ручная настройка

```bash
# Создайте .env файл
cp env.example .env

# Соберите и запустите сервисы
make dev-build
make dev-up

# Запустите миграции
make dev-migrate-up
```

## 📊 Доступные сервисы

| Сервис | URL | Описание |
|--------|-----|----------|
| **App** | http://localhost:8080 | Основное приложение |
| **PostgreSQL** | localhost:5432 | База данных |
| **Redis** | localhost:6379 | Кэш и блокировки |
| **pgAdmin** | http://localhost:5050 | Управление БД (с --profile tools) |
| **Redis Commander** | http://localhost:8081 | Управление Redis (с --profile tools) |

## 🛠️ Команды управления

### Основные команды

```bash
# Запуск окружения
make dev-up

# Остановка окружения
make dev-down

# Просмотр логов
make dev-logs

# Статус сервисов
make dev-status

# Перезапуск
make dev-restart
```

### Разработка

```bash
# Запуск с hot-reload
make dev-up

# Просмотр логов приложения
make dev-logs-app

# Открыть shell в контейнере приложения
make dev-shell

# Запустить тесты
make dev-test

# Запустить линтер
make dev-lint
```

### База данных

```bash
# Подключиться к PostgreSQL
make dev-db-shell

# Запустить миграции
make dev-migrate-up

# Откатить миграции
make dev-migrate-down

# Статус миграций
make dev-migrate-status

# Создать новую миграцию
make dev-migrate-create
```

### Redis

```bash
# Подключиться к Redis CLI
make dev-redis-shell

# Просмотр логов Redis
make dev-logs-redis
```

### Управление данными

```bash
# Создать бэкап БД
make dev-backup

# Восстановить из бэкапа
make dev-restore

# Проверить здоровье сервисов
make dev-health
```

### Инструменты разработки

```bash
# Запуск с инструментами управления
make dev-tools

# Очистка всего окружения
make dev-clean

# Полный сброс
make dev-reset
```

## 🔧 Конфигурация

### Переменные окружения

Основные переменные в `.env`:

```bash
# База данных
DATABASE_URL=postgres://dev:dev@localhost:5432/alerthistory?sslmode=disable
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=alerthistory
POSTGRES_USER=dev
POSTGRES_PASSWORD=dev

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=dev
REDIS_DB=0

# Приложение
APP_PORT=8080
APP_HOST=0.0.0.0
LOG_LEVEL=debug
ENVIRONMENT=development
```

### Настройка hot-reload

Приложение использует `air` для hot-reload. Конфигурация в `go-app/.air.toml`:

- Автоматическая пересборка при изменении `.go` файлов
- Исключение тестовых файлов и vendor
- Логирование сборки

## 🐳 Docker Compose сервисы

### postgres
- **Образ**: postgres:15-alpine
- **Порт**: 5432
- **Данные**: alerthistory
- **Пользователь**: dev/dev
- **Health check**: pg_isready

### redis
- **Образ**: redis:7-alpine
- **Порт**: 6379
- **Пароль**: dev
- **Persistence**: AOF
- **Health check**: redis-cli ping

### app
- **Сборка**: из go-app/Dockerfile
- **Порт**: 8080
- **Hot-reload**: air
- **Volumes**: исходный код для hot-reload
- **Зависимости**: postgres, redis

### pgadmin (опционально)
- **Образ**: dpage/pgadmin4
- **Порт**: 5050
- **Профиль**: tools
- **Доступ**: admin@alert-history.local / admin

### redis-commander (опционально)
- **Образ**: rediscommander/redis-commander
- **Порт**: 8081
- **Профиль**: tools

## 🔍 Отладка

### Проблемы с подключением

```bash
# Проверить статус сервисов
make dev-status

# Проверить логи
make dev-logs

# Проверить здоровье
make dev-health
```

### Проблемы с миграциями

```bash
# Проверить статус миграций
make dev-migrate-status

# Перезапустить миграции
make dev-migrate-down
make dev-migrate-up
```

### Проблемы с hot-reload

```bash
# Проверить конфигурацию air
cat go-app/.air.toml

# Запустить air вручную
make dev-shell
air -c .air.toml
```

## 📁 Структура файлов

```
.
├── docker-compose.yml          # Основная конфигурация
├── Makefile.docker            # Команды для Docker Compose
├── env.example                # Пример переменных окружения
├── go-app/
│   ├── Dockerfile             # Dockerfile с поддержкой dev
│   ├── .air.toml             # Конфигурация hot-reload
│   └── ...
└── backups/                   # Бэкапы БД (создается автоматически)
```

## 🚨 Устранение неполадок

### Сервис не запускается

1. Проверьте доступность портов:
   ```bash
   lsof -i :8080 -i :5432 -i :6379
   ```

2. Очистите Docker:
   ```bash
   make dev-clean
   make dev-reset
   ```

### Проблемы с базой данных

1. Проверьте подключение:
   ```bash
   make dev-db-shell
   ```

2. Пересоздайте базу:
   ```bash
   make dev-down
   docker volume rm alerthistory_postgres_data
   make dev-up
   ```

### Проблемы с Redis

1. Проверьте подключение:
   ```bash
   make dev-redis-shell
   ```

2. Очистите Redis:
   ```bash
   make dev-redis-shell
   FLUSHALL
   ```

## 📚 Дополнительные ресурсы

- [Docker Compose документация](https://docs.docker.com/compose/)
- [Air hot-reload](https://github.com/cosmtrek/air)
- [PostgreSQL Docker образ](https://hub.docker.com/_/postgres)
- [Redis Docker образ](https://hub.docker.com/_/redis)
