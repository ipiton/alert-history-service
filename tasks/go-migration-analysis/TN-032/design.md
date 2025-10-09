# TN-032: AlertStorage Design

**Дата последнего обновления**: 2025-10-08
**Статус**: Реализовано 70%, требуются доработки

## Реализованный интерфейс

**Файл**: `go-app/internal/core/interfaces.go` (строки 97-103)

```go
// AlertStorage interface for alert storage operations
type AlertStorage interface {
    SaveAlert(ctx context.Context, alert *Alert) error
    GetAlertByFingerprint(ctx context.Context, fingerprint string) (*Alert, error)
    GetAlerts(ctx context.Context, filters map[string]any, limit, offset int) ([]*Alert, error)
    CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error)
}
```

### Изменения от первоначального design

| Первоначально | Реализовано | Причина изменения |
|---------------|-------------|-------------------|
| `GetAlert()` | `GetAlertByFingerprint()` | Более явное имя |
| `ListAlerts(AlertFilters)` | `GetAlerts(map[string]any, limit, offset)` | Упрощённая реализация |
| `UpdateAlert()` | ❌ Не реализовано | SaveAlert использует UPSERT |
| `DeleteAlert()` | ❌ Не реализовано | Не требовалось для MVP |
| `GetStats()` | ⚠️ В Database интерфейсе | Вынесено на уровень выше |
| - | ➕ `CleanupOldAlerts()` | Добавлено для retention |

## Предлагаемый улучшенный интерфейс (v2)

```go
type AlertStorage interface {
    // Basic CRUD operations
    SaveAlert(ctx context.Context, alert *Alert) error
    GetAlertByFingerprint(ctx context.Context, fingerprint string) (*Alert, error)
    ListAlerts(ctx context.Context, filters *AlertFilters) (*AlertList, error)
    UpdateAlert(ctx context.Context, alert *Alert) error
    DeleteAlert(ctx context.Context, fingerprint string) error

    // Additional operations
    CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error)
    GetStats(ctx context.Context) (*AlertStats, error)
}

type AlertFilters struct {
    Status      *AlertStatus      `json:"status,omitempty"`
    Severity    *string           `json:"severity,omitempty"`
    Namespace   *string           `json:"namespace,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
    TimeRange   *TimeRange        `json:"time_range,omitempty"`
    Limit       int               `json:"limit" validate:"gte=0,lte=1000"`
    Offset      int               `json:"offset" validate:"gte=0"`
}

type AlertList struct {
    Alerts []*Alert `json:"alerts"`
    Total  int      `json:"total"`
    Limit  int      `json:"limit"`
    Offset int      `json:"offset"`
}

type AlertStats struct {
    TotalAlerts         int                `json:"total_alerts"`
    AlertsByStatus      map[string]int     `json:"alerts_by_status"`
    AlertsBySeverity    map[string]int     `json:"alerts_by_severity"`
    AlertsByNamespace   map[string]int     `json:"alerts_by_namespace"`
    OldestAlert         *time.Time         `json:"oldest_alert,omitempty"`
    NewestAlert         *time.Time         `json:"newest_alert,omitempty"`
}

type TimeRange struct {
    From *time.Time `json:"from,omitempty"`
    To   *time.Time `json:"to,omitempty"`
}
```

## PostgreSQL Schema (Реализованная)

**Файл миграции**: `go-app/migrations/20250911094416_initial_schema.sql`

```sql
-- Основная таблица alerts
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    fingerprint VARCHAR(64) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'firing',
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    starts_at TIMESTAMP WITH TIME ZONE,
    ends_at TIMESTAMP WITH TIME ZONE,
    generator_url TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Индексы для производительности
CREATE UNIQUE INDEX idx_alerts_fingerprint_unique ON alerts(fingerprint);
CREATE INDEX idx_alerts_alert_name ON alerts(alert_name);
CREATE INDEX idx_alerts_namespace ON alerts(namespace);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_timestamp ON alerts(timestamp DESC);
CREATE INDEX idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX idx_alerts_labels_gin ON alerts USING GIN(labels);
CREATE INDEX idx_alerts_annotations_gin ON alerts USING GIN(annotations);

-- Composite index для частых запросов
CREATE INDEX idx_alerts_name_status_time ON alerts(alert_name, status, timestamp DESC);

-- Trigger для автоматического обновления updated_at
CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Дополнительные таблицы

```sql
-- Таблица для LLM классификаций
CREATE TABLE IF NOT EXISTS alert_classifications (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint VARCHAR(64) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    confidence DECIMAL(4,3) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    reasoning TEXT,
    recommendations JSONB DEFAULT '[]',
    processing_time DECIMAL(8,3),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Таблица для истории публикаций
CREATE TABLE IF NOT EXISTS alert_publishing_history (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint VARCHAR(64) NOT NULL,
    target_name VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    processing_time DECIMAL(8,3),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

## Реализация адаптеров

### PostgreSQL Adapter

**Файл**: `go-app/internal/infrastructure/postgres_adapter.go`

**Особенности**:
- ✅ Connection pooling с `pgxpool`
- ✅ UPSERT операции (ON CONFLICT DO UPDATE)
- ✅ JSONB для labels и annotations
- ✅ Поддержка фильтрации через JSONB операторы
- ⚠️ **ПРОБЛЕМА**: SaveAlert записывает в `alert_data JSONB`, но миграция создаёт нормализованную структуру

**Пример реализации SaveAlert** (требует исправления):

```go
// Текущая версия (НЕ РАБОТАЕТ с миграцией)
query := `
    INSERT INTO alerts (fingerprint, alert_data)
    VALUES ($1, $2)
    ON CONFLICT (fingerprint)
    DO UPDATE SET alert_data = EXCLUDED.alert_data, updated_at = NOW()
`

// Требуется версия (совместимая с миграцией)
query := `
    INSERT INTO alerts (
        fingerprint, alert_name, status, labels, annotations,
        starts_at, ends_at, generator_url, namespace
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    ON CONFLICT (fingerprint)
    DO UPDATE SET
        alert_name = EXCLUDED.alert_name,
        status = EXCLUDED.status,
        labels = EXCLUDED.labels,
        annotations = EXCLUDED.annotations,
        starts_at = EXCLUDED.starts_at,
        ends_at = EXCLUDED.ends_at,
        generator_url = EXCLUDED.generator_url,
        namespace = EXCLUDED.namespace,
        updated_at = NOW()
`
```

### SQLite Adapter

**Файл**: `go-app/internal/infrastructure/sqlite_adapter.go`

**Особенности**:
- ✅ Используется для dev/testing окружения
- ✅ Нормализованная структура с отдельными колонками
- ✅ JSON для labels/annotations (TEXT тип)
- ✅ WAL mode для производительности
- ✅ Foreign keys enabled
- ✅ Полное тестовое покрытие

## Connection Pooling

**Конфигурация pgxpool**:

```go
poolConfig.MaxConns = int32(config.MaxOpenConns)      // По умолчанию: 25
poolConfig.MinConns = int32(config.MaxIdleConns)      // По умолчанию: 5
poolConfig.MaxConnLifetime = config.ConnMaxLifetime   // По умолчанию: 1 час
poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime   // По умолчанию: 30 минут
```

## Производительность

### Оптимизации
1. ✅ GIN индексы для JSONB колонок (labels, annotations)
2. ✅ Composite индексы для частых запросов
3. ✅ Connection pooling с настройкой параметров
4. ✅ Prepared statements через pgx
5. ⚠️ Pagination через LIMIT/OFFSET (может быть медленным на больших offset)

### Рекомендации для будущего
- Использовать keyset pagination вместо OFFSET
- Добавить materialized views для аналитики
- Партиционирование таблицы alerts по времени
- Read replicas для read-heavy workloads

## Безопасность

1. ✅ Prepared statements (защита от SQL injection)
2. ✅ Context support для таймаутов
3. ✅ Validation на уровне БД (CHECK constraints)
4. ⚠️ Требуется: Input validation в коде
5. ⚠️ Требуется: Rate limiting

---

**Дата**: 2025-10-08
**Автор обновления**: AI Assistant
**Следующие шаги**: Исправить конфликт SaveAlert с миграцией, добавить PostgreSQL тесты
