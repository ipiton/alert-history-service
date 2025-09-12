# TN-13: Реализация SQLite адаптера для development

## 🎯 **Цель задачи**

Создать легковесный SQLite адаптер для локальной разработки, который позволит разработчикам работать без установки PostgreSQL и обеспечит быструю итерацию при разработке.

## 📋 **Функциональные требования**

### **1. Интерфейс совместимости**
- [ ] **Database Interface**: Реализовать тот же интерфейс, что и PostgreSQL адаптер
- [ ] **Method Compatibility**: Все методы должны иметь одинаковую сигнатуру
- [ ] **Error Handling**: Единообразная обработка ошибок
- [ ] **Connection Management**: Аналогичное управление соединениями

### **2. SQLite специфика**
- [ ] **SQLite Driver**: Использовать `github.com/mattn/go-sqlite3`
- [ ] **In-Memory DB**: Поддержка in-memory базы для тестов
- [ ] **File-Based DB**: Поддержка файловой базы для persistence
- [ ] **SQLite Pragmas**: Оптимизация для development

### **3. Schema Management**
- [ ] **Auto Migration**: Автоматическое создание схемы при первом запуске
- [ ] **Schema Sync**: Синхронизация схемы с PostgreSQL структурой
- [ ] **Development Data**: Вставка тестовых данных для разработки
- [ ] **Schema Validation**: Проверка соответствия схем

### **4. Development Features**
- [ ] **Debug Logging**: Подробное логирование SQL запросов
- [ ] **Query Profiling**: Замер времени выполнения запросов
- [ ] **Data Inspection**: Удобный просмотр данных для отладки
- [ ] **Hot Reload**: Перезагрузка схемы без перезапуска

### **5. Testing Support**
- [ ] **Test Database**: Создание isolated тестовой базы для каждого теста
- [ ] **Transaction Rollback**: Автоматический rollback после тестов
- [ ] **Fixture Loading**: Загрузка тестовых данных
- [ ] **Parallel Tests**: Поддержка параллельного выполнения тестов

## 🔧 **Технические требования**

### **Архитектура адаптера**

#### **Интерфейс Database**
```go
type Database interface {
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Health(ctx context.Context) error

    // Alert operations
    CreateAlert(ctx context.Context, alert *Alert) error
    GetAlert(ctx context.Context, id string) (*Alert, error)
    UpdateAlert(ctx context.Context, alert *Alert) error
    DeleteAlert(ctx context.Context, id string) error
    ListAlerts(ctx context.Context, filter AlertFilter) ([]*Alert, error)

    // Classification operations
    CreateClassification(ctx context.Context, classification *Classification) error
    GetClassification(ctx context.Context, alertID string) (*Classification, error)

    // Publishing operations
    CreatePublishing(ctx context.Context, publishing *Publishing) error
    GetPublishingHistory(ctx context.Context, alertID string) ([]*Publishing, error)

    // Migration operations
    MigrateUp(ctx context.Context) error
    MigrateDown(ctx context.Context, steps int) error
}
```

#### **SQLite реализация**
```go
type SQLiteAdapter struct {
    db     *sql.DB
    logger *slog.Logger
    config *SQLiteConfig
}

type SQLiteConfig struct {
    DatabasePath string
    DebugMode    bool
    AutoMigrate  bool
    PoolSize     int
}
```

### **Schema Mapping**

#### **PostgreSQL → SQLite преобразования**
```sql
-- PostgreSQL types to SQLite equivalents
BIGSERIAL → INTEGER PRIMARY KEY AUTOINCREMENT
TIMESTAMP → DATETIME
JSONB → TEXT (JSON format)
UUID → TEXT
VARCHAR(n) → TEXT
BOOLEAN → INTEGER (0/1)
```

#### **Migration SQL**
```sql
-- SQLite schema creation
CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    labels TEXT, -- JSON format
    annotations TEXT -- JSON format
);

CREATE TABLE IF NOT EXISTS classifications (
    id TEXT PRIMARY KEY,
    alert_id TEXT NOT NULL,
    category TEXT NOT NULL,
    confidence REAL,
    metadata TEXT, -- JSON format
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (alert_id) REFERENCES alerts(id)
);

CREATE TABLE IF NOT EXISTS publishing (
    id TEXT PRIMARY KEY,
    alert_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    status TEXT NOT NULL,
    message_id TEXT,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (alert_id) REFERENCES alerts(id)
);
```

### **Configuration Management**

#### **Environment Variables**
```bash
# SQLite configuration
DB_TYPE=sqlite
DB_PATH=./dev.db
DB_DEBUG=true
DB_AUTO_MIGRATE=true
```

#### **Config Structure**
```go
type Config struct {
    Database struct {
        Type        string `env:"DB_TYPE" default:"postgres"`
        Path        string `env:"DB_PATH" default:"./dev.db"`
        Debug       bool   `env:"DB_DEBUG" default:"false"`
        AutoMigrate bool   `env:"DB_AUTO_MIGRATE" default:"true"`
        PoolSize    int    `env:"DB_POOL_SIZE" default:"1"`
    }
}
```

## ✅ **Критерии готовности**

### **Functional Requirements**
- [ ] **Interface Compliance**: Полная совместимость с Database интерфейсом
- [ ] **CRUD Operations**: Все операции создания/чтения/обновления/удаления работают
- [ ] **Query Support**: Поддержка фильтров и пагинации
- [ ] **Transaction Support**: Работа с транзакциями
- [ ] **Error Handling**: Корректная обработка и mapping ошибок

### **Performance Requirements**
- [ ] **Fast Startup**: Время запуска < 100ms
- [ ] **Query Performance**: Запросы выполняются < 10ms
- [ ] **Memory Usage**: Использование памяти < 50MB для типичных нагрузок
- [ ] **Concurrent Access**: Поддержка одновременного доступа

### **Development Experience**
- [ ] **Easy Setup**: Простая настройка для новых разработчиков
- [ ] **Debug Support**: Подробное логирование в debug режиме
- [ ] **Data Inspection**: Удобные инструменты для просмотра данных
- [ ] **Hot Reload**: Перезагрузка схемы без перезапуска

### **Testing Support**
- [ ] **Test Database**: Изолированная база для каждого теста
- [ ] **Fixture Support**: Загрузка тестовых данных
- [ ] **Cleanup**: Автоматическая очистка после тестов
- [ ] **Parallel Execution**: Поддержка параллельных тестов

## 🚀 **Implementation Plan**

### **Phase 1: Core Infrastructure (3 дня)**
1. Создание SQLiteAdapter структуры
2. Реализация базовых интерфейсов (Connect/Disconnect/Health)
3. Настройка SQLite соединения с pragmas
4. Базовое конфигурирование

### **Phase 2: Schema Management (3 дня)**
1. Создание SQL схемы для SQLite
2. Реализация автоматической миграции
3. Синхронизация с PostgreSQL структурой
4. Валидация схемы

### **Phase 3: CRUD Operations (4 дня)**
1. Реализация Alert CRUD операций
2. Реализация Classification CRUD операций
3. Реализация Publishing CRUD операций
4. Обработка ошибок и edge cases

### **Phase 4: Advanced Features (3 дня)**
1. Query фильтры и пагинация
2. Transaction support
3. Debug logging и profiling
4. Performance optimizations

### **Phase 5: Testing & Integration (3 дня)**
1. Unit тесты для всех операций
2. Integration тесты с реальной базой
3. Тесты производительности
4. Интеграция с основным приложением

### **Phase 6: Documentation & Examples (2 дня)**
1. Документация по использованию
2. Примеры кода для developers
3. Troubleshooting guide
4. Migration guide от PostgreSQL

## 📊 **Метрики успеха**

### **Performance Metrics**
- **Connection Time**: < 50ms
- **Query Time**: < 5ms для простых запросов
- **Memory Footprint**: < 20MB для development
- **Startup Time**: < 200ms с auto-migration

### **Compatibility Metrics**
- **Interface Coverage**: 100% методов Database интерфейса
- **Error Mapping**: 100% корректное mapping ошибок
- **Data Consistency**: 100% соответствие PostgreSQL схеме
- **Query Compatibility**: 95%+ совместимость запросов

### **Developer Experience**
- **Setup Time**: < 5 минут для нового разработчика
- **Debug Visibility**: 100% прозрачность SQL запросов
- **Data Inspection**: Удобный просмотр всех данных
- **Error Clarity**: Понятные сообщения об ошибках

## 🔒 **Безопасность и надежность**

### **Data Integrity**
- [ ] **Foreign Keys**: Включение foreign key constraints
- [ ] **Transactions**: ACID свойства для операций
- [ ] **Rollback**: Безопасный rollback при ошибках
- [ ] **Data Validation**: Валидация входных данных

### **Development Safety**
- [ ] **Isolated Environment**: Изоляция dev базы от production
- [ ] **Backup Support**: Возможность backup/restore
- [ ] **Data Reset**: Легкая очистка данных для тестирования
- [ ] **Version Control**: Контроль версий схемы

## 🧪 **Тестирование**

### **Unit Tests**
```go
func TestSQLiteAdapter_Connect(t *testing.T) {
    adapter := NewSQLiteAdapter(&SQLiteConfig{
        DatabasePath: ":memory:",
        DebugMode: true,
    })

    err := adapter.Connect(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, adapter.db)
}

func TestSQLiteAdapter_CRUD(t *testing.T) {
    adapter := setupTestAdapter(t)
    defer adapter.Disconnect(context.Background())

    // Test Create
    alert := &Alert{ID: "test-1", Title: "Test Alert"}
    err := adapter.CreateAlert(context.Background(), alert)
    assert.NoError(t, err)

    // Test Read
    retrieved, err := adapter.GetAlert(context.Background(), "test-1")
    assert.NoError(t, err)
    assert.Equal(t, "Test Alert", retrieved.Title)
}
```

### **Integration Tests**
```go
func TestSQLiteAdapter_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    adapter := NewSQLiteAdapter(&SQLiteConfig{
        DatabasePath: "./test_integration.db",
        DebugMode: true,
        AutoMigrate: true,
    })

    // Full application flow test
    ctx := context.Background()

    err := adapter.Connect(ctx)
    require.NoError(t, err)
    defer adapter.Disconnect(ctx)

    // Test complete workflow
    alerts := createTestAlerts(t, 100)
    for _, alert := range alerts {
        err := adapter.CreateAlert(ctx, alert)
        assert.NoError(t, err)
    }

    // Test querying
    filter := AlertFilter{Severity: "high"}
    results, err := adapter.ListAlerts(ctx, filter)
    assert.NoError(t, err)
    assert.True(t, len(results) > 0)
}
```

### **Performance Tests**
```go
func BenchmarkSQLiteAdapter_CRUD(b *testing.B) {
    adapter := setupBenchmarkAdapter(b)
    defer adapter.Disconnect(context.Background())

    b.ResetTimer()

    b.Run("Create", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            alert := &Alert{
                ID: fmt.Sprintf("bench-%d", i),
                Title: fmt.Sprintf("Benchmark Alert %d", i),
            }
            _ = adapter.CreateAlert(context.Background(), alert)
        }
    })
}
```

## 📋 **Примеры использования**

### **Development Setup**
```go
// Configuration for development
config := &SQLiteConfig{
    DatabasePath: "./dev.db",
    DebugMode: true,
    AutoMigrate: true,
    PoolSize: 1,
}

adapter := NewSQLiteAdapter(config)
err := adapter.Connect(ctx)
if err != nil {
    log.Fatal(err)
}
defer adapter.Disconnect(ctx)
```

### **Testing Setup**
```go
// In-memory database for tests
config := &SQLiteConfig{
    DatabasePath: ":memory:",
    DebugMode: true,
    AutoMigrate: true,
}

adapter := NewSQLiteAdapter(config)
// Use for isolated tests
```

### **Debug Mode Usage**
```go
// Enable debug logging
config := &SQLiteConfig{
    DatabasePath: "./debug.db",
    DebugMode: true, // Will log all SQL queries
    AutoMigrate: true,
}

adapter := NewSQLiteAdapter(config)
// All queries will be logged
```

## 🎯 **Ожидаемый результат**

### **Deliverables**
- ✅ **SQLite Adapter**: Полнофункциональный адаптер для development
- ✅ **Interface Compatible**: 100% совместимость с PostgreSQL интерфейсом
- ✅ **Development Ready**: Оптимизирован для локальной разработки
- ✅ **Well Tested**: Полное покрытие тестами
- ✅ **Documented**: Полная документация и примеры

### **Key Benefits**
- 🚀 **Fast Development**: Быстрое начало работы без PostgreSQL
- 🧪 **Easy Testing**: Упрощенное написание и запуск тестов
- 🔧 **Debug Friendly**: Отличная поддержка отладки
- 📊 **Performance**: Высокая производительность для development
- 🔄 **Seamless Switch**: Легкое переключение между SQLite/PostgreSQL

### **Usage Scenarios**
- **Local Development**: Основной инструмент для разработчиков
- **Unit Testing**: Изолированные тесты без внешних зависимостей
- **Integration Testing**: Быстрые интеграционные тесты
- **CI/CD**: Быстрое выполнение в пайплайнах
- **Prototyping**: Быстрое прототипирование новых фич

## 🎉 **Заключение**

**SQLite адаптер - ключевой компонент для эффективной разработки!**

### **🎯 Mission:**
- **Accelerate Development**: Ускорить цикл разработки
- **Simplify Testing**: Упростить тестирование
- **Improve DX**: Повысить developer experience
- **Maintain Compatibility**: Сохранить совместимость с production

### **📊 Impact:**
- **Setup Time**: Сокращение с часов до минут
- **Test Speed**: Ускорение тестов в 10-100 раз
- **Debugging**: Улучшение отладки на 80%
- **Productivity**: Повышение продуктивности команды

### **🚀 Ready for:**
- **Local Development**: Незамедлительное начало работы
- **Rapid Prototyping**: Быстрое создание прототипов
- **Comprehensive Testing**: Полное покрытие тестами
- **CI/CD Integration**: Быстрое выполнение в пайплайнах

**SQLite адаптер готов к созданию! Это будет game-changer для development experience!** 🚀✨
