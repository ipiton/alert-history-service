# TN-19: Дизайн — Loader конфигурации (viper)

## Цель
Интегрировать централизованную загрузку конфигурации с использованием `spf13/viper` в рантайм `cmd/server`, обеспечить приоритеты источников и маппинг параметров в компоненты (HTTP сервер, PostgreSQL пул).

## Источники конфигурации и приоритет
1) Defaults (заданы в `internal/config/config.go::setDefaults`)
2) YAML файл:
   - если указан через флаг `-config /path/to/config.yaml`
   - если задан `CONFIG_FILE=/path/to/config.yaml`
   - если присутствует `./config.yaml` в текущей рабочей директории
3) Переменные окружения (наивысший приоритет), формат KEY: `SERVER_PORT`, `DATABASE_HOST`, и т.д. (replacer `.` -> `_`).

Если файл не найден — это не ошибка. Ошибкой считаются: синтаксические ошибки YAML или невалидные значения (валидация `Config.Validate()`).

## Архитектура
- Пакет `internal/config` уже содержит:
  - структуры `Config` и вложенные типы
  - `LoadConfig(path string)` — с чтением файла (если указан)
  - `LoadConfigFromEnv()` — только defaults + env
  - `Validate()`, `GetDatabaseURL()` и хелперы среды
- В `cmd/server/main.go`:
  - Парсинг флагов: `-version`, `-help`, `-config`.
  - Определение пути к конфигу:
    - если `-config` задан — использовать его
    - иначе если `CONFIG_FILE` задан — использовать его
    - иначе если существует `./config.yaml` — использовать его
    - иначе — `LoadConfigFromEnv()`
  - Инициализация логгера `slog` c уровнем из `cfg.Log.Level` (debug/info/warn/error).
  - Маппинг `cfg.Database` → `postgres.PostgresConfig`:
    - Host, Port, Database, Username→User, Password, SSLMode
    - MaxConnections→MaxConns (int→int32)
    - MinConnections→MinConns (int→int32)
    - MaxConnLifetime, MaxConnIdleTime, ConnectTimeout — по одному к одному
    - HealthCheckPeriod — если не задан в `cfg.Database`, берем дефолт `30s`
  - Инициализация пула `postgres.NewPostgresPool(dbCfg, logger)` и `Connect(ctx)`.
  - Миграции уже запускаются через `internal/database.RunMigrations` (без изменений).
  - HTTP сервер:
    - Адрес `cfg.Server.Host:cfg.Server.Port`
    - Таймауты: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`
    - Graceful shutdown timeout из `cfg.Server.GracefulShutdownTimeout`.

## Ошибки и поведение
- Невалидный конфиг → лог + exit(1).
- Нет файла при явном указании `-config` → ошибка старта.
- Нет файла при авто-поиске `./config.yaml` → не ошибка, продолжаем с defaults + env.
- Ошибки подключения к БД → текущая логика сохранена (exit при невозможности подключения; при сбое миграций — продолжаем с предупреждением).

## Точки расширения
- TN-20: расширить логирование (формат/вывод/ротация) с конфигом.
- В будущем: добавить `database.health_check_period` в `internal/config` для прямой передачи в пул.

## Безопасность
- Чувствительные данные (пароли, токены) — через env/секреты, не хардкодятся.
- Конфиг-файл не должен попадать в VCS с секретами (проверено `.gitignore`).

## Тестирование
- Unit-тесты `internal/config/config_test.go`:
  - Defaults без файла/окружения
  - Загрузка из временного YAML файла
  - Override через env (перекрывает файл)
- Интеграционно (при необходимости): smoke для `cmd/server` с defaults.

## Формат данных (фрагменты)
- YAML (`go-app/config.yaml`) соответствует `internal/config.Config`.
- Env: `SERVER_PORT`, `DATABASE_HOST`, `APP_ENVIRONMENT`, `APP_DEBUG`, и т.п.
- Пример маппинга БД:
  - `cfg.Database.Username` → `postgres.PostgresConfig.User`
  - `cfg.Database.MaxConnections` → `postgres.PostgresConfig.MaxConns`

## Edge cases
- Пустые строковые значения из env могут «обнулить» defaults в viper. В тестах избегаем установки пустых значений, вместо этого очищаем переменные окружения.
- Если задано и `-config`, и `CONFIG_FILE` — побеждает `-config` (флаг > env).
