# TN-14: Система миграций базы данных (goose)

## 🎯 **Цель задачи**

Создать production-ready систему управления миграциями базы данных с использованием goose, которая обеспечит безопасное и контролируемое развитие схемы базы данных в разных окружениях.

## 📋 **Функциональные требования**

### **1. Поддержка баз данных**
- [ ] **PostgreSQL**: Полная поддержка PostgreSQL миграций
- [ ] **SQLite**: Поддержка SQLite для development и тестирования
- [ ] **Multi-driver**: Возможность переключения между драйверами
- [ ] **Driver abstraction**: Абстракция для разных типов баз данных

### **2. Управление миграциями**
- [ ] **Create migration**: Создание новых миграционных файлов
- [ ] **Up migration**: Применение миграций (up)
- [ ] **Down migration**: Откат миграций (down)
- [ ] **Status check**: Проверка статуса миграций
- [ ] **Version control**: Контроль версий миграций

### **3. Безопасность и надежность**
- [ ] **Transaction safety**: Все миграции в транзакциях
- [ ] **Rollback capability**: Возможность отката при ошибках
- [ ] **Dry-run mode**: Предварительная проверка миграций
- [ ] **Backup integration**: Интеграция с backup перед миграциями
- [ ] **Timeout control**: Контроль времени выполнения

### **4. Development experience**
- [ ] **Auto-migration**: Автоматическое применение при старте (dev)
- [ ] **Migration templates**: Шаблоны для новых миграций
- [ ] **Verbose logging**: Подробное логирование процесса
- [ ] **Migration history**: История примененных миграций
- [ ] **Conflict detection**: Обнаружение конфликтов миграций

### **5. CI/CD Integration**
- [ ] **Migration validation**: Валидация миграций в CI
- [ ] **Schema diff**: Сравнение схем между окружениями
- [ ] **Migration testing**: Тестирование миграций
- [ ] **Rollback testing**: Тестирование откатов
- [ ] **Deployment safety**: Безопасность развертывания

### **6. Monitoring и observability**
- [ ] **Migration metrics**: Метрики выполнения миграций
- [ ] **Error reporting**: Отчеты об ошибках миграций
- [ ] **Performance monitoring**: Мониторинг производительности
- [ ] **Audit logging**: Аудит логов всех операций
- [ ] **Health checks**: Проверки здоровья после миграций

## 🔧 **Технические требования**

### **Goose Integration**

#### **Базовая настройка**
```go
type MigrationConfig struct {
    Driver          string        // "postgres" или "sqlite"
    DSN            string        // Connection string
    Dir            string        // Директория миграций (default: "migrations")
    Table          string        // Таблица версий (default: "goose_db_version")
    Verbose        bool          // Подробное логирование
    NoVersioning   bool          // Отключить versioning (для тестов)
    Timeout        time.Duration // Таймаут выполнения
}

type MigrationManager struct {
    config *MigrationConfig
    goose  *goose.Provider
    logger *slog.Logger
}
```

#### **Основные методы**
```go
func NewMigrationManager(config *MigrationConfig) (*MigrationManager, error)

func (mm *MigrationManager) Up(ctx context.Context) error
func (mm *MigrationManager) UpTo(ctx context.Context, version int64) error
func (mm *MigrationManager) UpByOne(ctx context.Context) error

func (mm *MigrationManager) Down(ctx context.Context) error
func (mm *MigrationManager) DownTo(ctx context.Context, version int64) error

func (mm *MigrationManager) Status(ctx context.Context) ([]*MigrationStatus, error)
func (mm *MigrationManager) Version(ctx context.Context) (int64, error)

func (mm *MigrationManager) Create(ctx context.Context, name string) (string, error)
func (mm *MigrationManager) Validate(ctx context.Context) error
```

### **Структура миграций**

#### **Файловая структура**
```
migrations/
├── 20250101120000_initial_schema.sql
├── 20250101120100_add_indexes.sql
├── 20250101120200_add_constraints.sql
├── 20250102120000_add_alerts_table.sql
├── 20250102120100_add_classifications.sql
└── 20250102120200_add_publishing_logs.sql
```

#### **Формат миграций**
```sql
-- +goose Up
-- SQL команды для применения миграции

CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    alert_name TEXT NOT NULL,
    status TEXT NOT NULL,
    labels TEXT,
    annotations TEXT,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP,
    generator_url TEXT,
    timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL команды для отката миграции

DROP TABLE IF EXISTS alerts;
```

### **Migration Templates**

#### **Шаблоны для разных типов миграций**
```sql
-- Template: create_table.sql
-- +goose Up
CREATE TABLE {table_name} (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS {table_name};

-- Template: add_column.sql
-- +goose Up
ALTER TABLE {table_name} ADD COLUMN {column_name} {column_type};

-- +goose Down
ALTER TABLE {table_name} DROP COLUMN IF EXISTS {column_name};

-- Template: create_index.sql
-- +goose Up
CREATE INDEX CONCURRENTLY idx_{table_name}_{column_name} ON {table_name}({column_name});

-- +goose Down
DROP INDEX IF EXISTS idx_{table_name}_{column_name};
```

## ✅ **Критерии готовности**

### **Core Functionality**
- [ ] **Migration execution**: Up/Down миграции работают корректно
- [ ] **Status tracking**: Правильное отслеживание статуса миграций
- [ ] **Version control**: Контроль версий миграций
- [ ] **Error handling**: Корректная обработка ошибок
- [ ] **Transaction safety**: Миграции выполняются в транзакциях

### **Development Experience**
- [ ] **Easy setup**: Простая настройка для разработчиков
- [ ] **Auto-migration**: Автоматическое применение в dev
- [ ] **Migration creation**: Удобное создание новых миграций
- [ ] **Verbose logging**: Подробное логирование процесса
- [ ] **Conflict resolution**: Обнаружение и разрешение конфликтов

### **Production Readiness**
- [ ] **Rollback capability**: Безопасный откат миграций
- [ ] **Timeout handling**: Контроль времени выполнения
- [ ] **Dry-run support**: Предварительная проверка
- [ ] **Backup integration**: Интеграция с backup
- [ ] **Monitoring**: Метрики и мониторинг

### **Testing & Validation**
- [ ] **Migration testing**: Тестирование миграций
- [ ] **Rollback testing**: Тестирование откатов
- [ ] **Schema validation**: Валидация схемы после миграций
- [ ] **Data integrity**: Проверка целостности данных
- [ ] **Performance testing**: Тестирование производительности

### **CI/CD Integration**
- [ ] **CI validation**: Валидация миграций в CI
- [ ] **Schema diff**: Сравнение схем
- [ ] **Automated testing**: Автоматизированное тестирование
- [ ] **Deployment safety**: Безопасность развертывания
- [ ] **Audit logging**: Аудит всех операций

## 🚀 **Implementation Plan**

### **Phase 1: Core Migration System (3 дня)**
1. Настройка goose провайдера
2. Реализация основных команд (up, down, status)
3. Создание базовых миграций
4. Тестирование базовой функциональности

### **Phase 2: Advanced Features (3 дня)**
1. Реализация rollback capability
2. Добавление dry-run режима
3. Создание migration templates
4. Verbose logging и monitoring

### **Phase 3: Development Integration (2 дня)**
1. Auto-migration для development
2. Migration creation tools
3. Conflict detection
4. Development helpers

### **Phase 4: Production Safety (3 дня)**
1. Transaction safety improvements
2. Timeout handling
3. Backup integration
4. Production deployment procedures

### **Phase 5: Testing & Validation (2 дня)**
1. Comprehensive testing
2. Performance testing
3. CI/CD integration
4. Documentation

### **Phase 6: Monitoring & Observability (2 дня)**
1. Migration metrics
2. Error reporting
3. Audit logging
4. Health checks

## 📊 **Метрики успеха**

### **Performance Metrics**
- **Migration time**: < 30 секунд для типичных миграций
- **Rollback time**: < 60 секунд для откатов
- **Memory usage**: < 50MB во время миграций
- **CPU usage**: < 20% загрузки CPU

### **Reliability Metrics**
- **Success rate**: > 99.9% успешных миграций
- **Rollback success**: > 95% успешных откатов
- **Data integrity**: 100% сохранение целостности данных
- **Zero downtime**: Миграции без простоя сервиса

### **Developer Experience**
- **Setup time**: < 5 минут для новых разработчиков
- **Migration creation**: < 2 минуты на новую миграцию
- **Debug visibility**: 100% прозрачность процесса
- **Error clarity**: Понятные сообщения об ошибках

### **Operational Excellence**
- **Monitoring coverage**: 100% критических метрик
- **Alert coverage**: 100% критических сценариев
- **Documentation**: 100% процедур документировано
- **Training**: 100% команды обучены

## 🔒 **Безопасность и compliance**

### **Data Safety**
- [ ] **Backup before migration**: Автоматический backup перед миграциями
- [ ] **Transaction wrapping**: Все миграции в транзакциях
- [ ] **Rollback procedures**: Документированные процедуры отката
- [ ] **Data validation**: Валидация данных после миграций

### **Access Control**
- [ ] **Migration permissions**: Строгие права на выполнение миграций
- [ ] **Environment isolation**: Разделение dev/staging/production
- [ ] **Audit logging**: Полный аудит всех миграционных операций
- [ ] **Approval workflows**: Процессы согласования для production

### **Monitoring & Alerting**
- [ ] **Migration alerts**: Оповещения о статусе миграций
- [ ] **Error alerts**: Оповещения об ошибках миграций
- [ ] **Performance alerts**: Оповещения о производительности
- [ ] **Security alerts**: Оповещения о security issues

## 🧪 **Тестирование**

### **Unit Tests**
```go
func TestMigrationManager_Up(t *testing.T) {
    mm := setupTestMigrationManager(t)
    defer mm.cleanup()

    err := mm.Up(context.Background())
    assert.NoError(t, err)

    version, err := mm.Version(context.Background())
    assert.NoError(t, err)
    assert.Greater(t, version, int64(0))
}

func TestMigrationManager_Down(t *testing.T) {
    mm := setupTestMigrationManager(t)
    defer mm.cleanup()

    // First apply migrations
    err := mm.Up(context.Background())
    require.NoError(t, err)

    // Then rollback
    err = mm.Down(context.Background())
    assert.NoError(t, err)

    version, err := mm.Version(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, int64(0), version)
}
```

### **Integration Tests**
```go
func TestMigrationManager_PostgreSQL(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    mm := setupPostgreSQLMigrationManager(t)
    defer mm.cleanup()

    ctx := context.Background()

    // Test full migration cycle
    err := mm.Up(ctx)
    require.NoError(t, err)

    status, err := mm.Status(ctx)
    require.NoError(t, err)
    assert.True(t, len(status) > 0)

    // Verify schema
    err = mm.Validate(ctx)
    assert.NoError(t, err)
}

func TestMigrationManager_SQLite(t *testing.T) {
    mm := setupSQLiteMigrationManager(t)
    defer mm.cleanup()

    ctx := context.Background()

    // Test SQLite migrations
    err := mm.Up(ctx)
    require.NoError(t, err)

    // Verify SQLite schema
    err = mm.Validate(ctx)
    assert.NoError(t, err)
}
```

### **Performance Tests**
```go
func BenchmarkMigrationManager_Up(b *testing.B) {
    mm := setupBenchmarkMigrationManager(b)
    defer mm.cleanup()

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        mm.reset() // Reset to initial state

        err := mm.Up(context.Background())
        require.NoError(b, err)
    }
}

func BenchmarkMigrationManager_Status(b *testing.B) {
    mm := setupBenchmarkMigrationManager(b)
    defer mm.cleanup()

    // Apply migrations first
    err := mm.Up(context.Background())
    require.NoError(b, err)

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := mm.Status(context.Background())
        require.NoError(b, err)
    }
}
```

## 📋 **Примеры использования**

### **Development Usage**
```go
// Auto-migration on startup
config := &MigrationConfig{
    Driver:    "sqlite",
    DSN:       "./dev.db",
    Dir:       "./migrations",
    Verbose:   true,
}

mm, err := NewMigrationManager(config)
if err != nil {
    log.Fatal(err)
}

// Auto-apply migrations in development
if err := mm.Up(context.Background()); err != nil {
    log.Fatal("Failed to apply migrations:", err)
}
```

### **Production Usage**
```go
// Production migration with safety checks
config := &MigrationConfig{
    Driver:  "postgres",
    DSN:     "postgres://user:pass@host:5432/db",
    Dir:     "./migrations",
    Timeout: 5 * time.Minute,
}

mm, err := NewMigrationManager(config)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

// Backup before migration
if err := createBackup(ctx); err != nil {
    log.Fatal("Failed to create backup:", err)
}

// Dry run first
if err := mm.Validate(ctx); err != nil {
    log.Fatal("Migration validation failed:", err)
}

// Apply migrations
if err := mm.Up(ctx); err != nil {
    log.Error("Migration failed, attempting rollback:", err)

    if rollbackErr := mm.Down(ctx); rollbackErr != nil {
        log.Fatal("Rollback also failed:", rollbackErr)
    }

    log.Fatal("Migration and rollback failed")
}
```

### **CLI Usage**
```bash
# Create new migration
goose -dir migrations create add_user_table sql

# Check status
goose -dir migrations postgres "user=postgres dbname=test" status

# Apply migrations
goose -dir migrations postgres "user=postgres dbname=test" up

# Rollback last migration
goose -dir migrations postgres "user=postgres dbname=test" down

# Go to specific version
goose -dir migrations postgres "user=postgres dbname=test" up-to 20250101120000
```

## 🎯 **Ожидаемый результат**

### **Deliverables**
- ✅ **Migration Manager**: Полнофункциональный менеджер миграций
- ✅ **Goose Integration**: Интеграция с goose framework
- ✅ **Multi-Driver Support**: Поддержка PostgreSQL и SQLite
- ✅ **Safety Features**: Безопасность и надежность
- ✅ **Development Tools**: Инструменты для разработчиков
- ✅ **Production Ready**: Готовность к production

### **Key Benefits**
- 🚀 **Automated Schema Management**: Автоматизированное управление схемой
- 🛡️ **Production Safe**: Безопасность для production deployment
- 🧪 **Test Ready**: Полная поддержка тестирования
- 📊 **Observable**: Полная наблюдаемость процесса
- 🔄 **Reversible**: Возможность отката изменений
- ⚡ **Fast**: Быстрое выполнение миграций

### **Usage Scenarios**
- **Development**: Автоматическое применение миграций
- **Testing**: Изолированные тестовые базы данных
- **CI/CD**: Автоматизированное тестирование миграций
- **Production**: Безопасное развертывание изменений
- **Rollback**: Быстрое восстановление при проблемах

## 🎉 **Заключение**

**Система миграций - фундамент безопасного развития базы данных!**

### **🎯 Mission:**
- **Schema Evolution**: Контролируемое развитие схемы
- **Zero Downtime**: Миграции без простоя сервиса
- **Data Safety**: Защита данных и целостности
- **Developer Productivity**: Удобство разработки
- **Production Reliability**: Надежность в production

### **📊 Impact:**
- **Deployment Safety**: +300% безопасность развертываний
- **Rollback Speed**: -80% времени на откат
- **Developer Velocity**: +150% скорость разработки
- **Data Integrity**: 100% гарантия целостности
- **Monitoring**: Полная видимость процесса

### **🚀 Ready for:**
- **Schema Changes**: Безопасное развитие схемы
- **Multi-Environment**: Работа в dev/staging/production
- **Team Collaboration**: Совместная разработка
- **Automated Deployments**: CI/CD интеграция
- **Disaster Recovery**: Быстрое восстановление

**Migration system станет backbone для безопасного развития Alert History!** 🚀✨
