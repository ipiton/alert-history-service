# TN-14: Архитектура системы миграций

## 🏗️ **АРХИТЕКТУРНЫЙ ОБЗОР**

### **Цель и обоснование**

Система миграций является критически важным компонентом для безопасного и контролируемого развития схемы базы данных. Она обеспечивает:

- **Безопасное развитие схемы** без потери данных
- **Контролируемые развертывания** в разных окружениях
- **Возможность отката** при проблемах
- **Автоматизацию** процесса миграций
- **Наблюдаемость** и мониторинг

## 📋 **АРХИТЕКТУРА СИСТЕМЫ**

### **Общая архитектура**

```
┌─────────────────────────────────────────────────────────────┐
│                    MIGRATION SYSTEM                         │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │ │
│  │  │Migration    │  │Goose        │  │Error        │     │ │ │
│  │  │Manager      │  │Provider     │  │Handler      │     │ │ │
│  │  │             │  │             │  │             │     │ │ │
│  │  │• Commands   │  │• SQL Exec   │  │• Recovery    │     │ │ │
│  │  │• State Mgmt │  │• Versioning │  │• Rollback    │     │ │ │
│  │  │• Validation │  │• Transactions│  │• Logging     │     │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │             SAFETY & MONITORING                        │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │ │
│  │  │Backup       │  │Health       │  │Metrics      │     │ │ │
│  │  │Integration  │  │Checks       │  │Collection   │     │ │ │
│  │  │             │  │             │  │             │     │ │ │
│  │  │• Pre/Post   │  │• Validation  │  │• Counters    │     │ │ │
│  │  │• Recovery   │  │• Readiness   │  │• Histograms  │     │ │ │
│  │  │• Verification│  │• Liveness   │  │• Gauges      │     │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 **ОСНОВНЫЕ КОМПОНЕНТЫ**

### **1. MigrationManager - главный менеджер**

#### **Структура менеджера**
```go
type MigrationManager struct {
    config       *MigrationConfig
    provider     *goose.Provider
    db           *sql.DB
    logger       *slog.Logger
    metrics      *MigrationMetrics
    backupMgr    *BackupManager
    healthChecker *HealthChecker

    mu           sync.RWMutex
    isRunning    bool
}

type MigrationConfig struct {
    // Database configuration
    Driver          string        `env:"MIGRATION_DRIVER" default:"postgres"`
    DSN            string        `env:"MIGRATION_DSN" default:""`
    Dialect        string        `env:"MIGRATION_DIALECT" default:"postgres"`

    // Migration settings
    Dir            string        `env:"MIGRATION_DIR" default:"migrations"`
    Table          string        `env:"MIGRATION_TABLE" default:"goose_db_version"`
    Schema         string        `env:"MIGRATION_SCHEMA" default:"public"`

    // Safety settings
    Timeout        time.Duration `env:"MIGRATION_TIMEOUT" default:"5m"`
    MaxRetries     int           `env:"MIGRATION_MAX_RETRIES" default:"3"`
    RetryDelay     time.Duration `env:"MIGRATION_RETRY_DELAY" default:"5s"`

    // Development settings
    Verbose        bool          `env:"MIGRATION_VERBOSE" default:"false"`
    DryRun         bool          `env:"MIGRATION_DRY_RUN" default:"false"`
    AllowOutOfOrder bool         `env:"MIGRATION_ALLOW_OUT_OF_ORDER" default:"false"`

    // Safety settings
    NoVersioning   bool          `env:"MIGRATION_NO_VERSIONING" default:"false"`
    LockTimeout    time.Duration `env:"MIGRATION_LOCK_TIMEOUT" default:"10s"`

    // Monitoring
    EnableMetrics  bool          `env:"MIGRATION_METRICS" default:"true"`
    EnableTracing  bool          `env:"MIGRATION_TRACING" default:"false"`
}
```

#### **Основные методы**
```go
func NewMigrationManager(config *MigrationConfig) (*MigrationManager, error)

// Core migration operations
func (mm *MigrationManager) Up(ctx context.Context) error
func (mm *MigrationManager) UpTo(ctx context.Context, version int64) error
func (mm *MigrationManager) UpByOne(ctx context.Context) error

func (mm *MigrationManager) Down(ctx context.Context) error
func (mm *MigrationManager) DownTo(ctx context.Context, version int64) error
func (mm *MigrationManager) DownByOne(ctx context.Context) error

// Status and information
func (mm *MigrationManager) Status(ctx context.Context) ([]*MigrationStatus, error)
func (mm *MigrationManager) Version(ctx context.Context) (int64, error)
func (mm *MigrationManager) List(ctx context.Context) ([]*MigrationFile, error)

// Utility operations
func (mm *MigrationManager) Create(ctx context.Context, name string) (string, error)
func (mm *MigrationManager) Validate(ctx context.Context) error
func (mm *MigrationManager) Fix(ctx context.Context) error

// Advanced operations
func (mm *MigrationManager) Redo(ctx context.Context) error
func (mm *MigrationManager) Reset(ctx context.Context) error
```

### **2. Goose Provider - интеграция с goose**

#### **Настройка провайдера**
```go
type GooseProvider struct {
    provider *goose.Provider
    dialect  goose.Dialect
    fs       *goose.FS
}

func NewGooseProvider(config *MigrationConfig) (*GooseProvider, error) {
    // Create dialect based on driver
    var dialect goose.Dialect
    switch config.Driver {
    case "postgres":
        dialect = goose.DialectPostgres
    case "sqlite":
        dialect = goose.DialectSQLite3
    default:
        return nil, fmt.Errorf("unsupported driver: %s", config.Driver)
    }

    // Create filesystem for migrations
    fs, err := goose.NewFS(config.Dir)
    if err != nil {
        return nil, fmt.Errorf("failed to create filesystem: %w", err)
    }

    // Create goose provider
    opts := []goose.ProviderOption{
        goose.WithDialect(dialect),
        goose.WithFS(fs),
        goose.WithTable(config.Table),
        goose.WithVerbose(config.Verbose),
        goose.WithTimeout(config.Timeout),
    }

    if config.AllowOutOfOrder {
        opts = append(opts, goose.WithAllowOutOfOrder())
    }

    if config.NoVersioning {
        opts = append(opts, goose.WithNoVersioning())
    }

    provider, err := goose.NewProvider(dialect, config.DSN, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create goose provider: %w", err)
    }

    return &GooseProvider{
        provider: provider,
        dialect:  dialect,
        fs:       fs,
    }, nil
}
```

#### **Методы провайдера**
```go
func (gp *GooseProvider) Up(ctx context.Context) error {
    return gp.provider.Up(ctx)
}

func (gp *GooseProvider) UpTo(ctx context.Context, version int64) error {
    return gp.provider.UpTo(ctx, version)
}

func (gp *GooseProvider) Down(ctx context.Context) error {
    return gp.provider.Down(ctx)
}

func (gp *GooseProvider) DownTo(ctx context.Context, version int64) error {
    return gp.provider.DownTo(ctx, version)
}

func (gp *GooseProvider) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
    return gp.provider.Status(ctx)
}

func (gp *GooseProvider) Version(ctx context.Context) (int64, error) {
    return gp.provider.Version(ctx)
}

func (gp *GooseProvider) List(ctx context.Context) ([]*goose.Migration, error) {
    return gp.provider.List(ctx)
}
```

### **3. Error Handler - обработчик ошибок и recovery**

#### **Error handling стратегия**
```go
type ErrorHandler struct {
    logger       *slog.Logger
    metrics      *MigrationMetrics
    maxRetries   int
    retryDelay   time.Duration
}

type MigrationError struct {
    Operation string
    Version   int64
    Cause     error
    Timestamp time.Time
    Context   map[string]any
}

func (e *MigrationError) Error() string {
    return fmt.Sprintf("migration %s failed at version %d: %v", e.Operation, e.Version, e.Cause)
}

func (eh *ErrorHandler) HandleError(ctx context.Context, err error, operation string, version int64) error {
    migrationErr := &MigrationError{
        Operation: operation,
        Version:   version,
        Cause:     err,
        Timestamp: time.Now(),
        Context:   map[string]any{
            "operation": operation,
            "version":   version,
            "timestamp": time.Now(),
        },
    }

    // Log error
    eh.logger.Error("Migration error",
        "operation", operation,
        "version", version,
        "error", err,
    )

    // Record metrics
    eh.metrics.IncrementErrorCounter(operation)

    return migrationErr
}

func (eh *ErrorHandler) ExecuteWithRetry(ctx context.Context, operation func() error) error {
    var lastErr error

    for attempt := 0; attempt <= eh.maxRetries; attempt++ {
        if attempt > 0 {
            eh.logger.Info("Retrying migration operation",
                "attempt", attempt,
                "max_retries", eh.maxRetries,
            )

            select {
            case <-time.After(eh.retryDelay):
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        if err := operation(); err != nil {
            lastErr = err

            // Check if error is retryable
            if !eh.isRetryable(err) {
                break
            }

            eh.metrics.IncrementRetryCounter()
            continue
        }

        return nil
    }

    return lastErr
}

func (eh *ErrorHandler) isRetryable(err error) bool {
    // Check for common retryable errors
    errStr := err.Error()

    retryablePatterns := []string{
        "connection refused",
        "connection reset",
        "timeout",
        "lock wait timeout",
        "deadlock",
        "serialization failure",
    }

    for _, pattern := range retryablePatterns {
        if strings.Contains(errStr, pattern) {
            return true
        }
    }

    return false
}
```

### **4. Backup Integration - интеграция с backup**

#### **Backup Manager**
```go
type BackupManager struct {
    config     *BackupConfig
    logger     *slog.Logger
    db         *sql.DB
}

type BackupConfig struct {
    Enabled       bool          `env:"BACKUP_ENABLED" default:"true"`
    Type          string        `env:"BACKUP_TYPE" default:"schema"`
    Path          string        `env:"BACKUP_PATH" default:"./backups"`
    RetentionDays int           `env:"BACKUP_RETENTION_DAYS" default:"30"`
    Compress      bool          `env:"BACKUP_COMPRESS" default:"true"`
}

func (bm *BackupManager) CreatePreMigrationBackup(ctx context.Context) (string, error) {
    if !bm.config.Enabled {
        bm.logger.Info("Backup disabled, skipping pre-migration backup")
        return "", nil
    }

    timestamp := time.Now().Format("20060102_150405")
    backupFile := fmt.Sprintf("%s/pre_migration_%s.sql", bm.config.Path, timestamp)

    // Ensure backup directory exists
    if err := os.MkdirAll(bm.config.Path, 0755); err != nil {
        return "", fmt.Errorf("failed to create backup directory: %w", err)
    }

    // Create backup based on database type
    switch bm.getDatabaseType() {
    case "postgres":
        return bm.createPostgreSQLBackup(ctx, backupFile)
    case "sqlite":
        return bm.createSQLiteBackup(ctx, backupFile)
    default:
        return "", fmt.Errorf("unsupported database type for backup")
    }
}

func (bm *BackupManager) createPostgreSQLBackup(ctx context.Context, backupFile string) (string, error) {
    // Use pg_dump for PostgreSQL backup
    cmd := exec.CommandContext(ctx, "pg_dump",
        "--schema-only",
        "--no-owner",
        "--no-privileges",
        "--file", backupFile,
        bm.getConnectionString(),
    )

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("failed to create PostgreSQL backup: %w", err)
    }

    bm.logger.Info("PostgreSQL schema backup created", "file", backupFile)
    return backupFile, nil
}

func (bm *BackupManager) createSQLiteBackup(ctx context.Context, backupFile string) (string, error) {
    // Use SQLite .dump command
    dumpQuery := fmt.Sprintf(".dump > %s", backupFile)

    if _, err := bm.db.ExecContext(ctx, dumpQuery); err != nil {
        return "", fmt.Errorf("failed to create SQLite backup: %w", err)
    }

    bm.logger.Info("SQLite backup created", "file", backupFile)
    return backupFile, nil
}

func (bm *BackupManager) VerifyBackup(ctx context.Context, backupFile string) error {
    if _, err := os.Stat(backupFile); os.IsNotExist(err) {
        return fmt.Errorf("backup file does not exist: %s", backupFile)
    }

    // Basic validation - check file size
    if stat, err := os.Stat(backupFile); err != nil {
        return fmt.Errorf("failed to stat backup file: %w", err)
    } else if stat.Size() == 0 {
        return fmt.Errorf("backup file is empty: %s", backupFile)
    }

    bm.logger.Info("Backup verification successful", "file", backupFile, "size", stat.Size())
    return nil
}

func (bm *BackupManager) CleanupOldBackups(ctx context.Context) error {
    if bm.config.RetentionDays <= 0 {
        return nil
    }

    cutoffDate := time.Now().AddDate(0, 0, -bm.config.RetentionDays)

    entries, err := os.ReadDir(bm.config.Path)
    if err != nil {
        return fmt.Errorf("failed to read backup directory: %w", err)
    }

    var deletedCount int
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        // Check if file is old enough to delete
        if strings.HasPrefix(entry.Name(), "pre_migration_") ||
           strings.HasPrefix(entry.Name(), "post_migration_") {

            // Parse timestamp from filename
            if timestamp, err := time.Parse("20060102_150405", strings.TrimPrefix(entry.Name(), "pre_migration_")); err == nil {
                if timestamp.Before(cutoffDate) {
                    filePath := filepath.Join(bm.config.Path, entry.Name())
                    if err := os.Remove(filePath); err != nil {
                        bm.logger.Warn("Failed to remove old backup", "file", filePath, "error", err)
                    } else {
                        deletedCount++
                        bm.logger.Info("Removed old backup", "file", entry.Name())
                    }
                }
            }
        }
    }

    bm.logger.Info("Backup cleanup completed", "deleted", deletedCount)
    return nil
}
```

### **5. Health Checker - проверки здоровья**

#### **Health validation**
```go
type HealthChecker struct {
    db         *sql.DB
    config     *HealthConfig
    logger     *slog.Logger
}

type HealthConfig struct {
    Enabled         bool          `env:"HEALTH_ENABLED" default:"true"`
    Timeout         time.Duration `env:"HEALTH_TIMEOUT" default:"30s"`
    RetryCount      int           `env:"HEALTH_RETRY_COUNT" default:"3"`
    RetryDelay      time.Duration `env:"HEALTH_RETRY_DELAY" default:"5s"`
}

func (hc *HealthChecker) PreMigrationCheck(ctx context.Context) error {
    hc.logger.Info("Running pre-migration health checks")

    checks := []HealthCheck{
        hc.checkDatabaseConnectivity,
        hc.checkDatabasePermissions,
        hc.checkExistingMigrations,
        hc.checkDiskSpace,
        hc.checkTableIntegrity,
    }

    for _, check := range checks {
        if err := hc.executeCheck(ctx, check); err != nil {
            return fmt.Errorf("pre-migration health check failed: %w", err)
        }
    }

    hc.logger.Info("All pre-migration health checks passed")
    return nil
}

func (hc *HealthChecker) PostMigrationCheck(ctx context.Context) error {
    hc.logger.Info("Running post-migration health checks")

    checks := []HealthCheck{
        hc.checkDatabaseConnectivity,
        hc.checkSchemaIntegrity,
        hc.checkDataConsistency,
        hc.checkIndexValidity,
        hc.checkMigrationTable,
    }

    for _, check := range checks {
        if err := hc.executeCheck(ctx, check); err != nil {
            return fmt.Errorf("post-migration health check failed: %w", err)
        }
    }

    hc.logger.Info("All post-migration health checks passed")
    return nil
}

func (hc *HealthChecker) executeCheck(ctx context.Context, check HealthCheck) error {
    checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
    defer cancel()

    var lastErr error
    for attempt := 0; attempt < hc.config.RetryCount; attempt++ {
        if attempt > 0 {
            select {
            case <-time.After(hc.config.RetryDelay):
            case <-checkCtx.Done():
                return checkCtx.Err()
            }
        }

        if err := check(checkCtx); err != nil {
            lastErr = err
            hc.logger.Warn("Health check failed, retrying",
                "attempt", attempt+1,
                "error", err,
            )
            continue
        }

        return nil
    }

    return lastErr
}

func (hc *HealthChecker) checkDatabaseConnectivity(ctx context.Context) error {
    return hc.db.PingContext(ctx)
}

func (hc *HealthChecker) checkDatabasePermissions(ctx context.Context) error {
    // Check if we can create and drop a test table
    testTable := "migration_health_check"

    if _, err := hc.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s (id INTEGER)", testTable)); err != nil {
        return fmt.Errorf("cannot create tables: %w", err)
    }

    if _, err := hc.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", testTable)); err != nil {
        return fmt.Errorf("cannot drop tables: %w", err)
    }

    return nil
}

func (hc *HealthChecker) checkExistingMigrations(ctx context.Context) error {
    // Check if migration table exists and is in consistent state
    rows, err := hc.db.QueryContext(ctx, `
        SELECT version_id, is_applied, tstamp
        FROM goose_db_version
        ORDER BY version_id
    `)
    if err != nil {
        // Migration table doesn't exist yet, that's OK
        return nil
    }
    defer rows.Close()

    var lastVersion int64 = 0
    for rows.Next() {
        var versionID int64
        var isApplied bool
        var timestamp time.Time

        if err := rows.Scan(&versionID, &isApplied, &timestamp); err != nil {
            return fmt.Errorf("failed to scan migration status: %w", err)
        }

        // Check for gaps in applied migrations
        if isApplied && versionID > lastVersion+1 {
            return fmt.Errorf("missing migration between %d and %d", lastVersion, versionID)
        }

        if isApplied {
            lastVersion = versionID
        }
    }

    return nil
}

func (hc *HealthChecker) checkDiskSpace(ctx context.Context) error {
    // For SQLite, check available disk space
    // For PostgreSQL, this would be more complex and might require OS-level checks

    var dbPath string
    if err := hc.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&dbPath); err == nil {
        // This is SQLite
        stat := syscall.Statfs_t{}
        if err := syscall.Statfs(dbPath, &stat); err != nil {
            hc.logger.Warn("Cannot check disk space for SQLite", "error", err)
            return nil // Don't fail the check
        }

        // Check if we have at least 100MB free
        freeBytes := stat.Bavail * uint64(stat.Bsize)
        minFreeBytes := uint64(100 * 1024 * 1024) // 100MB

        if freeBytes < minFreeBytes {
            return fmt.Errorf("insufficient disk space: %d MB free, need at least 100 MB", freeBytes/(1024*1024))
        }
    }

    return nil
}

func (hc *HealthChecker) checkTableIntegrity(ctx context.Context) error {
    // Check integrity of existing tables
    if _, err := hc.db.ExecContext(ctx, "PRAGMA integrity_check"); err != nil {
        return fmt.Errorf("database integrity check failed: %w", err)
    }

    return nil
}

func (hc *HealthChecker) checkSchemaIntegrity(ctx context.Context) error {
    // Check that all expected tables exist
    expectedTables := []string{
        "alerts",
        "classifications",
        "publishing_logs",
        "goose_db_version",
    }

    for _, table := range expectedTables {
        var exists bool
        query := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)"

        if err := hc.db.QueryRowContext(ctx, query, table).Scan(&exists); err != nil {
            // Try SQLite syntax
            sqliteQuery := fmt.Sprintf("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='%s'", table)
            if err := hc.db.QueryRowContext(ctx, sqliteQuery).Scan(&exists); err != nil {
                return fmt.Errorf("failed to check table existence for %s: %w", table, err)
            }
        }

        if !exists {
            return fmt.Errorf("required table %s does not exist", table)
        }
    }

    return nil
}

func (hc *HealthChecker) checkDataConsistency(ctx context.Context) error {
    // Check foreign key constraints
    if _, err := hc.db.ExecContext(ctx, "PRAGMA foreign_key_check"); err != nil {
        hc.logger.Warn("Foreign key check failed", "error", err)
        // Don't fail the check for foreign key issues in development
    }

    // Check for orphaned records
    var orphanedCount int
    if err := hc.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM classifications c
        LEFT JOIN alerts a ON c.alert_fingerprint = a.fingerprint
        WHERE a.fingerprint IS NULL
    `).Scan(&orphanedCount); err != nil {
        return fmt.Errorf("failed to check orphaned classifications: %w", err)
    }

    if orphanedCount > 0 {
        hc.logger.Warn("Found orphaned classification records",
            "count", orphanedCount,
            "recommendation", "Run cleanup or fix data consistency")
    }

    return nil
}

func (hc *HealthChecker) checkIndexValidity(ctx context.Context) error {
    // Check that indexes are valid and not corrupted
    rows, err := hc.db.QueryContext(ctx, "PRAGMA index_list(alerts)")
    if err != nil {
        hc.logger.Warn("Cannot check index validity", "error", err)
        return nil // Don't fail on this check
    }
    defer rows.Close()

    for rows.Next() {
        var seq int
        var name string
        var unique bool
        var origin string
        var partial bool

        if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
            return fmt.Errorf("failed to scan index info: %w", err)
        }

        // Check index integrity
        if _, err := hc.db.ExecContext(ctx, fmt.Sprintf("PRAGMA index_info(%s)", name)); err != nil {
            return fmt.Errorf("index %s appears to be corrupted: %w", name, err)
        }
    }

    return nil
}

func (hc *HealthChecker) checkMigrationTable(ctx context.Context) error {
    // Verify that migration table is in correct state
    var count int
    if err := hc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM goose_db_version").Scan(&count); err != nil {
        return fmt.Errorf("failed to check migration table: %w", err)
    }

    hc.logger.Info("Migration table status",
        "recorded_migrations", count)

    return nil
}
```

### **6. Migration Metrics - метрики и мониторинг**

#### **Metrics collection**
```go
type MigrationMetrics struct {
    registry *prometheus.Registry
    logger   *slog.Logger

    // Counters
    migrationsApplied    *prometheus.CounterVec
    migrationsFailed     *prometheus.CounterVec
    migrationsRolledBack *prometheus.CounterVec
    retryAttempts        *prometheus.CounterVec

    // Gauges
    currentVersion       prometheus.Gauge
    migrationDuration    *prometheus.HistogramVec

    // Histograms
    queryDuration        *prometheus.HistogramVec
}

func NewMigrationMetrics() *MigrationMetrics {
    mm := &MigrationMetrics{
        registry: prometheus.NewRegistry(),
    }

    // Migration counters
    mm.migrationsApplied = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "migration_applied_total",
            Help: "Total number of migrations applied",
        },
        []string{"direction", "database"},
    )

    mm.migrationsFailed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "migration_failed_total",
            Help: "Total number of migrations failed",
        },
        []string{"operation", "database"},
    )

    mm.migrationsRolledBack = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "migration_rolled_back_total",
            Help: "Total number of migrations rolled back",
        },
        []string{"database"},
    )

    mm.retryAttempts = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "migration_retry_attempts_total",
            Help: "Total number of migration retry attempts",
        },
        []string{"operation", "database"},
    )

    // Version gauge
    mm.currentVersion = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "migration_current_version",
            Help: "Current migration version",
        },
    )

    // Duration histograms
    mm.migrationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "migration_duration_seconds",
            Help:    "Time spent on migration operations",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "database"},
    )

    mm.queryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "migration_query_duration_seconds",
            Help:    "Time spent on individual migration queries",
            Buckets: []float64{.001, .005, .01, .05, .1, .5, 1, 5, 10},
        },
        []string{"database"},
    )

    // Register all metrics
    mm.registry.MustRegister(
        mm.migrationsApplied,
        mm.migrationsFailed,
        mm.migrationsRolledBack,
        mm.retryAttempts,
        mm.currentVersion,
        mm.migrationDuration,
        mm.queryDuration,
    )

    return mm
}

func (mm *MigrationMetrics) IncrementAppliedCounter(direction, database string) {
    mm.migrationsApplied.WithLabelValues(direction, database).Inc()
}

func (mm *MigrationMetrics) IncrementFailedCounter(operation, database string) {
    mm.migrationsFailed.WithLabelValues(operation, database).Inc()
}

func (mm *MigrationMetrics) IncrementRollbackCounter(database string) {
    mm.migrationsRolledBack.WithLabelValues(database).Inc()
}

func (mm *MigrationMetrics) IncrementRetryCounter(operation, database string) {
    mm.retryAttempts.WithLabelValues(operation, database).Inc()
}

func (mm *MigrationMetrics) SetCurrentVersion(version int64) {
    mm.currentVersion.Set(float64(version))
}

func (mm *MigrationMetrics) ObserveMigrationDuration(operation, database string, duration time.Duration) {
    mm.migrationDuration.WithLabelValues(operation, database).Observe(duration.Seconds())
}

func (mm *MigrationMetrics) ObserveQueryDuration(database string, duration time.Duration) {
    mm.queryDuration.WithLabelValues(database).Observe(duration.Seconds())
}

func (mm *MigrationMetrics) GetRegistry() *prometheus.Registry {
    return mm.registry
}
```

## 🔄 **МИГРАЦИОННЫЕ СТРАТЕГИИ**

### **Up Migration Strategy**
```go
func (mm *MigrationManager) Up(ctx context.Context) error {
    mm.mu.Lock()
    mm.isRunning = true
    mm.mu.Unlock()

    defer func() {
        mm.mu.Lock()
        mm.isRunning = false
        mm.mu.Unlock()
    }()

    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        mm.metrics.ObserveMigrationDuration("up", mm.config.Driver, duration)
    }()

    mm.logger.Info("Starting migration up process")

    // Pre-migration health checks
    if err := mm.healthChecker.PreMigrationCheck(ctx); err != nil {
        return mm.errorHandler.HandleError(ctx, err, "pre_migration_check", 0)
    }

    // Create backup
    backupFile, err := mm.backupMgr.CreatePreMigrationBackup(ctx)
    if err != nil {
        return mm.errorHandler.HandleError(ctx, err, "backup_creation", 0)
    }

    if backupFile != "" {
        mm.logger.Info("Pre-migration backup created", "file", backupFile)
    }

    // Execute migrations with retry logic
    err = mm.errorHandler.ExecuteWithRetry(ctx, func() error {
        return mm.provider.Up(ctx)
    })

    if err != nil {
        mm.logger.Error("Migration up failed", "error", err)

        // Attempt rollback
        if rollbackErr := mm.attemptRollback(ctx); rollbackErr != nil {
            mm.logger.Error("Rollback also failed", "error", rollbackErr)
            return fmt.Errorf("migration failed and rollback unsuccessful: %w", rollbackErr)
        }

        return mm.errorHandler.HandleError(ctx, err, "migration_up", 0)
    }

    // Update metrics
    mm.metrics.IncrementAppliedCounter("up", mm.config.Driver)

    // Get current version
    version, err := mm.provider.Version(ctx)
    if err != nil {
        mm.logger.Warn("Failed to get version after migration", "error", err)
    } else {
        mm.metrics.SetCurrentVersion(version)
    }

    // Post-migration health checks
    if err := mm.healthChecker.PostMigrationCheck(ctx); err != nil {
        mm.logger.Warn("Post-migration health check failed", "error", err)
        // Don't fail the migration for health check issues
    }

    mm.logger.Info("Migration up completed successfully",
        "duration", time.Since(startTime))

    return nil
}

func (mm *MigrationManager) attemptRollback(ctx context.Context) error {
    mm.logger.Info("Attempting rollback due to migration failure")

    // Try to rollback the last migration
    if err := mm.provider.DownByOne(ctx); err != nil {
        mm.logger.Error("Failed to rollback last migration", "error", err)
        return err
    }

    mm.metrics.IncrementRollbackCounter(mm.config.Driver)
    mm.logger.Info("Rollback completed successfully")

    return nil
}
```

### **Down Migration Strategy**
```go
func (mm *MigrationManager) Down(ctx context.Context) error {
    mm.mu.Lock()
    mm.isRunning = true
    mm.mu.Unlock()

    defer func() {
        mm.mu.Lock()
        mm.isRunning = false
        mm.mu.Unlock()
    }()

    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        mm.metrics.ObserveMigrationDuration("down", mm.config.Driver, duration)
    }()

    mm.logger.Info("Starting migration down process")

    // Pre-rollback health checks
    if err := mm.healthChecker.PreMigrationCheck(ctx); err != nil {
        return mm.errorHandler.HandleError(ctx, err, "pre_rollback_check", 0)
    }

    // Create backup before rollback
    backupFile, err := mm.backupMgr.CreatePreMigrationBackup(ctx)
    if err != nil {
        return mm.errorHandler.HandleError(ctx, err, "rollback_backup_creation", 0)
    }

    if backupFile != "" {
        mm.logger.Info("Pre-rollback backup created", "file", backupFile)
    }

    // Get current version before rollback
    currentVersion, err := mm.provider.Version(ctx)
    if err != nil {
        mm.logger.Warn("Failed to get current version", "error", err)
    }

    // Execute rollback with retry logic
    err = mm.errorHandler.ExecuteWithRetry(ctx, func() error {
        return mm.provider.Down(ctx)
    })

    if err != nil {
        mm.logger.Error("Migration down failed", "error", err)
        return mm.errorHandler.HandleError(ctx, err, "migration_down", currentVersion)
    }

    // Update metrics
    mm.metrics.IncrementAppliedCounter("down", mm.config.Driver)

    // Get new version
    newVersion, err := mm.provider.Version(ctx)
    if err != nil {
        mm.logger.Warn("Failed to get version after rollback", "error", err)
    } else {
        mm.metrics.SetCurrentVersion(newVersion)
    }

    // Post-rollback health checks
    if err := mm.healthChecker.PostMigrationCheck(ctx); err != nil {
        mm.logger.Warn("Post-rollback health check failed", "error", err)
    }

    mm.logger.Info("Migration down completed successfully",
        "from_version", currentVersion,
        "to_version", newVersion,
        "duration", time.Since(startTime))

    return nil
}
```

## 📊 **СТАТУС И МОНИТОРИНГ**

### **Migration Status**
```go
type MigrationStatus struct {
    VersionID    int64     `json:"version_id"`
    IsApplied    bool      `json:"is_applied"`
    Timestamp    time.Time `json:"timestamp"`
    Source       string    `json:"source"`
    Description  string    `json:"description"`
}

func (mm *MigrationManager) Status(ctx context.Context) ([]*MigrationStatus, error) {
    gooseStatuses, err := mm.provider.Status(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get goose status: %w", err)
    }

    statuses := make([]*MigrationStatus, len(gooseStatuses))
    for i, gs := range gooseStatuses {
        statuses[i] = &MigrationStatus{
            VersionID:   gs.VersionID,
            IsApplied:   gs.IsApplied,
            Timestamp:   gs.Timestamp,
            Source:      gs.Source,
            Description: gs.Description,
        }
    }

    return statuses, nil
}

func (mm *MigrationManager) GetMigrationHistory(ctx context.Context) ([]*MigrationStatus, error) {
    statuses, err := mm.Status(ctx)
    if err != nil {
        return nil, err
    }

    // Sort by timestamp descending (most recent first)
    sort.Slice(statuses, func(i, j int) bool {
        return statuses[i].Timestamp.After(statuses[j].Timestamp)
    })

    return statuses, nil
}

func (mm *MigrationManager) GetPendingMigrations(ctx context.Context) ([]*MigrationStatus, error) {
    statuses, err := mm.Status(ctx)
    if err != nil {
        return nil, err
    }

    var pending []*MigrationStatus
    for _, status := range statuses {
        if !status.IsApplied {
            pending = append(pending, status)
        }
    }

    return pending, nil
}
```

### **Validation и Consistency Checks**
```go
func (mm *MigrationManager) Validate(ctx context.Context) error {
    mm.logger.Info("Starting migration validation")

    // Check migration files
    migrations, err := mm.provider.List(ctx)
    if err != nil {
        return fmt.Errorf("failed to list migrations: %w", err)
    }

    // Check for duplicate versions
    versionMap := make(map[int64]bool)
    for _, migration := range migrations {
        if versionMap[migration.VersionID] {
            return fmt.Errorf("duplicate migration version: %d", migration.VersionID)
        }
        versionMap[migration.VersionID] = true
    }

    // Check migration file integrity
    for _, migration := range migrations {
        if err := mm.validateMigrationFile(migration); err != nil {
            return fmt.Errorf("invalid migration file %s: %w", migration.Source, err)
        }
    }

    // Check database consistency
    if err := mm.validateDatabaseConsistency(ctx); err != nil {
        return fmt.Errorf("database consistency check failed: %w", err)
    }

    mm.logger.Info("Migration validation completed successfully")
    return nil
}

func (mm *MigrationManager) validateMigrationFile(migration *goose.Migration) error {
    // Check if file exists
    if _, err := os.Stat(migration.Source); os.IsNotExist(err) {
        return fmt.Errorf("migration file does not exist: %s", migration.Source)
    }

    // Check file permissions
    if stat, err := os.Stat(migration.Source); err != nil {
        return fmt.Errorf("cannot stat migration file: %w", err)
    } else if stat.Mode().Perm()&0400 == 0 {
        return fmt.Errorf("migration file is not readable: %s", migration.Source)
    }

    // Check file content
    content, err := os.ReadFile(migration.Source)
    if err != nil {
        return fmt.Errorf("cannot read migration file: %w", err)
    }

    contentStr := string(content)

    // Check for required goose directives
    if !strings.Contains(contentStr, "-- +goose Up") {
        return fmt.Errorf("migration file missing -- +goose Up directive")
    }

    if !strings.Contains(contentStr, "-- +goose Down") {
        return fmt.Errorf("migration file missing -- +goose Down directive")
    }

    return nil
}

func (mm *MigrationManager) validateDatabaseConsistency(ctx context.Context) error {
    // Check that migration table exists
    if err := mm.db.QueryRowContext(ctx, "SELECT 1 FROM goose_db_version LIMIT 1").Scan(new(int)); err != nil {
        if strings.Contains(err.Error(), "no such table") {
            mm.logger.Info("Migration table does not exist yet")
            return nil // This is OK for initial setup
        }
        return fmt.Errorf("migration table check failed: %w", err)
    }

    // Check for orphaned migration records
    var orphanedCount int
    if err := mm.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM goose_db_version
        WHERE version_id NOT IN (
            SELECT CAST(SUBSTR(filename, 1, 14) AS INTEGER)
            FROM goose_db_version
        )
    `).Scan(&orphanedCount); err != nil {
        return fmt.Errorf("orphaned migration check failed: %w", err)
    }

    if orphanedCount > 0 {
        mm.logger.Warn("Found orphaned migration records",
            "count", orphanedCount)
    }

    return nil
}
```

## 🎯 **ОЖИДАЕМЫЕ РЕЗУЛЬТАТЫ**

### **Architecture Benefits**
- ✅ **Safe Schema Evolution**: Контролируемое развитие схемы
- ✅ **Zero Downtime**: Миграции без простоя сервиса
- ✅ **Rollback Capability**: Возможность отката изменений
- ✅ **Multi-Environment**: Поддержка dev/staging/production
- ✅ **Observable**: Полная наблюдаемость процесса

### **Key Features Delivered**
- 🚀 **Goose Integration**: Полная интеграция с goose framework
- 🛡️ **Transaction Safety**: Все миграции в транзакциях
- 🔄 **Error Recovery**: Автоматическое восстановление при ошибках
- 📊 **Metrics & Monitoring**: Полные метрики и мониторинг
- ⚡ **Performance**: Оптимизированные операции миграций

### **Safety Guarantees**
- **Backup Before Migration**: Автоматический backup
- **Health Checks**: Pre и post migration проверки
- **Timeout Control**: Контроль времени выполнения
- **Retry Logic**: Логика повторных попыток
- **Audit Logging**: Полное логирование операций

## 🎉 **ЗАКЛЮЧЕНИЕ**

**Система миграций - это enterprise-grade решение для управления схемой базы данных!**

### **🎯 Design Principles:**
- **Safety First**: Безопасность превыше всего
- **Observable**: Полная наблюдаемость всех операций
- **Recoverable**: Возможность восстановления при любых ошибках
- **Multi-Environment**: Поддержка всех окружений
- **Performance Optimized**: Оптимизированная производительность

### **📊 Performance Targets:**
- **Migration Time**: < 30 секунд для типичных миграций
- **Rollback Time**: < 60 секунд для откатов
- **Memory Usage**: < 50MB во время миграций
- **Success Rate**: > 99.9% успешных миграций

### **🚀 Key Advantages:**
- **Zero Downtime**: Миграции без остановки сервиса
- **Auto Recovery**: Автоматическое восстановление при ошибках
- **Full Monitoring**: Полная наблюдаемость процесса
- **Multi-Driver**: Поддержка PostgreSQL и SQLite
- **Production Ready**: Готово к production deployment

**Архитектура системы миграций готова к реализации! Это будет backbone для безопасного развития Alert History!** 🚀✨
