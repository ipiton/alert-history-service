# TN-13: Реализация SQLite адаптера

## 🎯 **Цель задачи**

Создать полнофункциональный SQLite адаптер для development среды с полной совместимостью интерфейса PostgreSQL адаптера.

## 📋 **Чек-лист выполнения**

### **Phase 1: Core Infrastructure (3 дня)**
- [ ] Создать основную структуру SQLiteAdapter
- [ ] Реализовать базовые интерфейсы (Connect/Disconnect/Health)
- [ ] Настроить SQLite соединение с оптимальными pragmas
- [ ] Добавить базовую конфигурацию и валидацию
- [ ] Создать unit тесты для основных функций
- [ ] Интегрировать slog для логирования

### **Phase 2: Schema Management (2 дня)**
- [ ] Создать SQL схему для SQLite (alerts, classifications, publishing)
- [ ] Реализовать автоматическую миграцию при первом запуске
- [ ] Синхронизировать схему с PostgreSQL структурой
- [ ] Добавить валидацию схемы и integrity checks
- [ ] Создать индексы для производительности
- [ ] Добавить тестовые данные для development

### **Phase 3: CRUD Operations - Alerts (3 дня)**
- [ ] Реализовать CreateAlert с JSON marshaling
- [ ] Реализовать GetAlert с error handling
- [ ] Реализовать UpdateAlert с optimistic locking
- [ ] Реализовать DeleteAlert с cascade
- [ ] Реализовать ListAlerts с фильтрацией и пагинацией
- [ ] Добавить prepared statements для performance
- [ ] Написать comprehensive unit тесты

### **Phase 4: CRUD Operations - Classifications (2 дня)**
- [ ] Реализовать CreateClassification
- [ ] Реализовать GetClassification
- [ ] Реализовать UpdateClassification
- [ ] Реализовать DeleteClassification
- [ ] Реализовать ListClassifications с фильтрами
- [ ] Добавить foreign key constraints
- [ ] Интегрировать с alerts CRUD

### **Phase 5: CRUD Operations - Publishing (2 дня)**
- [ ] Реализовать CreatePublishing
- [ ] Реализовать GetPublishing
- [ ] Реализовать UpdatePublishing
- [ ] Реализовать DeletePublishing
- [ ] Реализовать ListPublishing с фильтрами
- [ ] Добавить status tracking
- [ ] Интегрировать с alerts CRUD

### **Phase 6: Advanced Features (3 дня)**
- [ ] Реализовать QueryBuilder для сложных запросов
- [ ] Добавить transaction support
- [ ] Реализовать batch operations
- [ ] Добавить connection pooling
- [ ] Реализовать statement caching
- [ ] Добавить metrics collection

### **Phase 7: Development Features (3 дня)**
- [ ] Реализовать debug mode с SQL logging
- [ ] Добавить query profiling и timing
- [ ] Создать data inspection tools
- [ ] Реализовать hot reload для схемы
- [ ] Добавить development fixtures
- [ ] Создать CLI tools для debugging

### **Phase 8: Error Handling & Mapping (2 дня)**
- [ ] Создать SQLiteError type
- [ ] Реализовать mapping SQLite ошибок в application ошибки
- [ ] Добавить error recovery mechanisms
- [ ] Реализовать circuit breaker pattern
- [ ] Добавить error logging и monitoring
- [ ] Написать тесты для error scenarios

### **Phase 9: Testing Support (4 дня)**
- [ ] Создать TestHelper для isolated тестов
- [ ] Реализовать in-memory database setup
- [ ] Добавить fixture loading
- [ ] Реализовать automatic cleanup
- [ ] Поддержать parallel test execution
- [ ] Создать benchmark тесты
- [ ] Написать integration тесты

### **Phase 10: Performance Optimization (3 дня)**
- [ ] Оптимизировать SQLite pragmas
- [ ] Реализовать connection pooling
- [ ] Добавить prepared statement caching
- [ ] Оптимизировать JSON marshaling/unmarshaling
- [ ] Реализовать batch operations
- [ ] Добавить memory management
- [ ] Создать performance benchmarks

### **Phase 11: Integration & Documentation (3 дня)**
- [ ] Интегрировать с основным приложением
- [ ] Добавить configuration management
- [ ] Создать comprehensive documentation
- [ ] Написать usage examples
- [ ] Создать troubleshooting guide
- [ ] Добавить migration guide
- [ ] Создать video demo/tutorial

### **Phase 12: Final Testing & Release (2 дня)**
- [ ] Запустить полный test suite
- [ ] Провести performance testing
- [ ] Выполнить integration testing
- [ ] Провести security review
- [ ] Создать release notes
- [ ] Опубликовать документацию
- [ ] Подготовить demo для команды

## 🔧 **Технические детали реализации**

### **Основные компоненты**

#### **1. SQLiteAdapter Core**
```go
// Core structure
type SQLiteAdapter struct {
    db         *sql.DB
    config     *SQLiteConfig
    logger     *slog.Logger
    metrics    *AdapterMetrics
    queryCache map[string]*sql.Stmt
    mu         sync.RWMutex
}

// Configuration
type SQLiteConfig struct {
    DatabasePath     string
    DebugMode        bool
    AutoMigrate      bool
    JournalMode      string
    SynchronousMode  string
    CacheSize        int
}
```

#### **2. Database Interface**
```go
type Database interface {
    // Core methods
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
    CreateClassification(ctx context.Context, cls *Classification) error
    GetClassification(ctx context.Context, alertID string) (*Classification, error)

    // Publishing operations
    CreatePublishing(ctx context.Context, pub *Publishing) error
    GetPublishingHistory(ctx context.Context, alertID string) ([]*Publishing, error)

    // Development methods
    MigrateUp(ctx context.Context) error
    GetStats(ctx context.Context) (map[string]interface{}, error)
}
```

#### **3. Schema Definition**
```sql
-- SQLite schema
CREATE TABLE alerts (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT,
    labels TEXT, -- JSON
    annotations TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE classifications (
    id TEXT PRIMARY KEY,
    alert_id TEXT NOT NULL,
    category TEXT NOT NULL,
    confidence REAL,
    metadata TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (alert_id) REFERENCES alerts(id)
);

CREATE TABLE publishing (
    id TEXT PRIMARY KEY,
    alert_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    status TEXT NOT NULL,
    message_id TEXT,
    error_message TEXT,
    sent_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (alert_id) REFERENCES alerts(id)
);
```

### **Ключевые алгоритмы**

#### **1. Connection Setup**
```go
func (a *SQLiteAdapter) setupConnection(ctx context.Context) error {
    // Open connection
    db, err := sql.Open("sqlite3", a.config.DatabasePath)
    if err != nil {
        return fmt.Errorf("failed to open SQLite: %w", err)
    }

    // Configure pragmas
    pragmas := map[string]string{
        "journal_mode":    a.config.JournalMode,
        "synchronous":     a.config.SynchronousMode,
        "cache_size":      fmt.Sprintf("%d", a.config.CacheSize),
        "foreign_keys":    "ON",
        "busy_timeout":    "30000",
    }

    for pragma, value := range pragmas {
        if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA %s = %s;", pragma, value)); err != nil {
            return fmt.Errorf("failed to set pragma %s: %w", pragma, err)
        }
    }

    a.db = db
    return nil
}
```

#### **2. Auto Migration**
```go
func (a *SQLiteAdapter) migrateUp(ctx context.Context) error {
    schemas := []string{
        createAlertsTable,
        createClassificationsTable,
        createPublishingTable,
        createIndexes,
    }

    for _, schema := range schemas {
        if _, err := a.db.ExecContext(ctx, schema); err != nil {
            return fmt.Errorf("failed to execute schema: %w", err)
        }
    }

    // Insert development data
    return a.insertDevelopmentData(ctx)
}
```

#### **3. CRUD Operations**
```go
func (a *SQLiteAdapter) CreateAlert(ctx context.Context, alert *Alert) error {
    query := `
        INSERT INTO alerts (id, title, description, severity, status, source, labels, annotations)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

    labelsJSON, err := json.Marshal(alert.Labels)
    if err != nil {
        return fmt.Errorf("failed to marshal labels: %w", err)
    }

    annotationsJSON, err := json.Marshal(alert.Annotations)
    if err != nil {
        return fmt.Errorf("failed to marshal annotations: %w", err)
    }

    _, err = a.db.ExecContext(ctx, query,
        alert.ID, alert.Title, alert.Description,
        alert.Severity, alert.Status, alert.Source,
        string(labelsJSON), string(annotationsJSON),
    )

    if err != nil {
        return a.mapError(err, query, []interface{}{
            alert.ID, alert.Title, alert.Description,
            alert.Severity, alert.Status, alert.Source,
            string(labelsJSON), string(annotationsJSON),
        })
    }

    return nil
}
```

#### **4. Debug Logging**
```go
func (a *SQLiteAdapter) logQuery(ctx context.Context, query string, args []interface{}, start time.Time) {
    if !a.config.DebugMode {
        return
    }

    duration := time.Since(start)
    a.logger.Debug("SQL Query",
        "query", query,
        "args", args,
        "duration", duration,
        "timestamp", time.Now(),
    )
}
```

### **Тестовая инфраструктура**

#### **1. Test Helper**
```go
type TestHelper struct {
    adapter *SQLiteAdapter
    dbPath  string
}

func NewTestHelper() *TestHelper {
    dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("test_%d.db", time.Now().UnixNano()))

    config := &SQLiteConfig{
        DatabasePath: dbPath,
        DebugMode:    true,
        AutoMigrate:  true,
    }

    return &TestHelper{
        adapter: NewSQLiteAdapter(config, slog.Default()),
        dbPath:  dbPath,
    }
}

func (th *TestHelper) Setup(t *testing.T) {
    err := th.adapter.Connect(context.Background())
    require.NoError(t, err)

    t.Cleanup(func() {
        th.adapter.Disconnect(context.Background())
        os.Remove(th.dbPath)
    })
}
```

#### **2. Benchmark Tests**
```go
func BenchmarkSQLiteAdapter_CRUD(b *testing.B) {
    helper := NewTestHelper()
    defer helper.adapter.Disconnect(context.Background())

    ctx := context.Background()
    err := helper.adapter.Connect(ctx)
    require.NoError(b, err)

    b.ResetTimer()

    b.Run("Create", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            alert := &Alert{
                ID:          fmt.Sprintf("bench-%d", i),
                Title:       fmt.Sprintf("Benchmark Alert %d", i),
                Description: "Benchmark description",
                Severity:    "high",
                Status:      "active",
            }
            _ = helper.adapter.CreateAlert(ctx, alert)
        }
    })
}
```

## 📊 **Метрики и KPI**

### **Performance Metrics**
- **Connection Time**: < 10ms
- **Simple Query**: < 1ms
- **Complex Query**: < 5ms
- **Memory Usage**: < 10MB
- **Startup Time**: < 100ms

### **Compatibility Metrics**
- **Interface Coverage**: 100%
- **Error Mapping**: 100%
- **Data Consistency**: 100%
- **Query Compatibility**: 95%

### **Quality Metrics**
- **Test Coverage**: > 90%
- **Linting**: 0 errors
- **Documentation**: 100%
- **Integration**: ✅ working

## 🚨 **Риски и mitigation**

### **Высокий риск**
- **Interface Incompatibility**: Несоответствие PostgreSQL интерфейсу
- **Performance Issues**: Недостаточная производительность
- **Data Corruption**: Потеря или повреждение данных

### **Средний риск**
- **Debug Complexity**: Сложность отладки
- **Migration Issues**: Проблемы с автоматической миграцией
- **Testing Complexity**: Сложность тестирования

### **Низкий риск**
- **Documentation**: Недостаточная документация
- **Error Handling**: Неполная обработка ошибок

### **Меры предосторожности**
- [ ] **Interface Verification**: Постоянная проверка совместимости
- [ ] **Performance Monitoring**: Мониторинг производительности
- [ ] **Data Integrity Checks**: Проверка целостности данных
- [ ] **Comprehensive Testing**: Полное покрытие тестами
- [ ] **Backup Strategy**: Стратегия резервного копирования

## 📋 **Примеры использования**

### **Development Setup**
```go
config := &SQLiteConfig{
    DatabasePath: "./dev.db",
    DebugMode:    true,
    AutoMigrate:  true,
}

adapter := NewSQLiteAdapter(config, logger)
err := adapter.Connect(ctx)
if err != nil {
    log.Fatal(err)
}
```

### **Testing Setup**
```go
func TestAlertCRUD(t *testing.T) {
    helper := NewTestHelper()
    helper.Setup(t)

    // Test CRUD operations
    alert := &Alert{ID: "test-1", Title: "Test Alert"}
    err := helper.adapter.CreateAlert(context.Background(), alert)
    assert.NoError(t, err)

    retrieved, err := helper.adapter.GetAlert(context.Background(), "test-1")
    assert.NoError(t, err)
    assert.Equal(t, "Test Alert", retrieved.Title)
}
```

### **Debug Mode**
```go
// Enable debug logging
config := &SQLiteConfig{
    DatabasePath: "./debug.db",
    DebugMode: true, // Will log all SQL queries
}

// All queries will be logged:
// DEBUG SQL Query query="SELECT * FROM alerts WHERE id = ?" args=["test-1"] duration=1.2ms
```

## 🎯 **Ожидаемые результаты**

### **Deliverables**
- ✅ **Full SQLite Adapter**: Полнофункциональный адаптер
- ✅ **Interface Compatible**: 100% совместимость с PostgreSQL
- ✅ **Development Optimized**: Специально для development
- ✅ **Performance Optimized**: Быстрые операции
- ✅ **Debug Friendly**: Отличная поддержка отладки
- ✅ **Test Ready**: Полная поддержка тестирования
- ✅ **Well Documented**: Полная документация

### **Key Benefits**
- 🚀 **Zero PostgreSQL Setup**: Нет необходимости в PostgreSQL
- 🧪 **Fast Testing**: Быстрые и изолированные тесты
- 📊 **Excellent Debugging**: Полная видимость SQL
- ⚡ **High Performance**: Оптимизированные запросы
- 🛡️ **Data Safety**: Целостность данных
- 🔄 **Easy Switch**: Легкое переключение между SQLite/PostgreSQL

### **Usage Scenarios**
- **Local Development**: Основной инструмент разработчиков
- **Unit Testing**: Быстрые изолированные тесты
- **Integration Testing**: Тесты с реальной базой
- **CI/CD**: Быстрое выполнение в пайплайнах
- **Prototyping**: Быстрое создание прототипов

## 🎉 **Заключение**

**TN-13 - это фундамент для эффективной development среды!**

### **🎯 Mission Critical:**
- **Zero-Setup Development**: Разработка без PostgreSQL
- **Fast Iteration**: Быстрое прототипирование
- **Excellent Debugging**: Полная прозрачность
- **Comprehensive Testing**: Все виды тестирования
- **Production Compatibility**: Совместимость с production

### **📊 Success Metrics:**
- **Setup Time**: < 5 минут для нового разработчика
- **Test Speed**: Ускорение тестов на 10-100x
- **Debug Visibility**: 100% прозрачность SQL
- **Interface Compatibility**: 100% совместимость
- **Performance**: < 1ms на простые запросы

### **🚀 Impact:**
- **Developer Productivity**: +200% для локальной разработки
- **Testing Speed**: +1000% для unit тестов
- **Debug Efficiency**: +300% эффективность отладки
- **Onboarding**: -80% времени на настройку
- **CI/CD Speed**: +50% скорость пайплайнов

**SQLite адаптер готов к созданию! Это будет game-changer для development experience!** 🚀✨
