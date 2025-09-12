package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// PostgresDatabase адаптер для PostgreSQL, реализующий общий интерфейс Database
type PostgresDatabase struct {
	pool   *pgxpool.Pool
	config *Config
	logger *slog.Logger
	db     *sql.DB // Для совместимости с sql.Result
}

// NewPostgresDatabase создает новый PostgreSQL адаптер
func NewPostgresDatabase(config *Config) (*PostgresDatabase, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	pg := &PostgresDatabase{
		config: config,
		logger: config.Logger,
	}

	return pg, nil
}

// Connect устанавливает соединение с PostgreSQL
func (p *PostgresDatabase) Connect(ctx context.Context) error {
	p.logger.Info("Connecting to PostgreSQL", "dsn", p.config.DSN)

	// Создаем конфигурацию pgxpool
	poolConfig, err := pgxpool.ParseConfig(p.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse database DSN: %w", err)
	}

	// Настраиваем параметры pool
	poolConfig.MaxConns = int32(p.config.MaxOpenConns)
	poolConfig.MinConns = int32(p.config.MaxIdleConns)
	poolConfig.MaxConnLifetime = p.config.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = p.config.ConnMaxIdleTime

	// Устанавливаем таймаут подключения
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Тестируем соединение
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.pool = pool
	p.db = stdlib.OpenDBFromPool(pool)

	connectionTime := time.Since(start)
	p.logger.Info("Successfully connected to PostgreSQL",
		"connection_time", connectionTime,
		"max_conns", p.config.MaxOpenConns)

	return nil
}

// Disconnect закрывает соединение с PostgreSQL
func (p *PostgresDatabase) Disconnect(ctx context.Context) error {
	if p.pool == nil {
		return nil
	}

	p.logger.Info("Disconnecting from PostgreSQL")

	p.pool.Close()
	p.pool = nil
	p.db = nil

	p.logger.Info("Successfully disconnected from PostgreSQL")
	return nil
}

// IsConnected проверяет состояние соединения
func (p *PostgresDatabase) IsConnected() bool {
	if p.pool == nil {
		return false
	}

	// Проверяем состояние pool
	stats := p.pool.Stat()
	return stats.TotalConns() > 0
}

// Health выполняет проверку здоровья базы данных
func (p *PostgresDatabase) Health(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	return p.pool.Ping(ctx)
}

// Exec выполняет SQL команду (адаптер для pgx)
func (p *PostgresDatabase) Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	tag, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	// Преобразуем pgconn.CommandTag в sql.Result совместимый формат
	return &postgresResult{tag: tag}, nil
}

// Query выполняет SQL запрос (адаптер для pgx)
func (p *PostgresDatabase) Query(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	if p.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	return p.db.QueryContext(ctx, sql, args...)
}

// QueryRow выполняет SQL запрос и возвращает одну строку
func (p *PostgresDatabase) QueryRow(ctx context.Context, sql string, args ...interface{}) *sql.Row {
	if p.db == nil {
		return nil
	}

	return p.db.QueryRowContext(ctx, sql, args...)
}

// Begin начинает новую транзакцию
func (p *PostgresDatabase) Begin(ctx context.Context) (*sql.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	return p.db.BeginTx(ctx, nil)
}

// MigrateUp выполняет миграции схемы для PostgreSQL
func (p *PostgresDatabase) MigrateUp(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	// Создаем таблицу alerts если она не существует
	createAlertsTableSQL := `
	CREATE TABLE IF NOT EXISTS alerts (
		fingerprint TEXT PRIMARY KEY,
		alert_data JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
	CREATE INDEX IF NOT EXISTS idx_alerts_updated_at ON alerts(updated_at);
	`

	_, err := p.pool.Exec(ctx, createAlertsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create alerts table: %w", err)
	}

	// Создаем таблицу classifications
	createClassificationsTableSQL := `
	CREATE TABLE IF NOT EXISTS classifications (
		id SERIAL PRIMARY KEY,
		alert_fingerprint TEXT NOT NULL REFERENCES alerts(fingerprint) ON DELETE CASCADE,
		classification_data JSONB NOT NULL,
		confidence REAL NOT NULL,
		category TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(alert_fingerprint)
	);

	CREATE INDEX IF NOT EXISTS idx_classifications_alert_fingerprint ON classifications(alert_fingerprint);
	CREATE INDEX IF NOT EXISTS idx_classifications_category ON classifications(category);
	CREATE INDEX IF NOT EXISTS idx_classifications_created_at ON classifications(created_at);
	`

	_, err = p.pool.Exec(ctx, createClassificationsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create classifications table: %w", err)
	}

	// Создаем таблицу publishing_logs
	createPublishingTableSQL := `
	CREATE TABLE IF NOT EXISTS publishing_logs (
		id SERIAL PRIMARY KEY,
		alert_fingerprint TEXT NOT NULL REFERENCES alerts(fingerprint) ON DELETE CASCADE,
		target_name TEXT NOT NULL,
		success BOOLEAN NOT NULL,
		error_message TEXT,
		processing_time REAL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_publishing_alert_fingerprint ON publishing_logs(alert_fingerprint);
	CREATE INDEX IF NOT EXISTS idx_publishing_target_name ON publishing_logs(target_name);
	CREATE INDEX IF NOT EXISTS idx_publishing_success ON publishing_logs(success);
	CREATE INDEX IF NOT EXISTS idx_publishing_created_at ON publishing_logs(created_at);
	`

	_, err = p.pool.Exec(ctx, createPublishingTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create publishing_logs table: %w", err)
	}

	p.logger.Info("PostgreSQL schema migration completed successfully",
		"tables_created", []string{"alerts", "classifications", "publishing_logs"})
	return nil
}

// SaveAlert сохраняет алерт в PostgreSQL
func (p *PostgresDatabase) SaveAlert(ctx context.Context, alert *core.Alert) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	alertData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	query := `
		INSERT INTO alerts (fingerprint, alert_data)
		VALUES ($1, $2)
		ON CONFLICT (fingerprint)
		DO UPDATE SET
			alert_data = EXCLUDED.alert_data,
			updated_at = NOW()`

	_, err = p.pool.Exec(ctx, query, alert.Fingerprint, alertData)
	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	return nil
}

// GetAlertByFingerprint получает алерт по fingerprint из PostgreSQL
func (p *PostgresDatabase) GetAlertByFingerprint(ctx context.Context, fingerprint string) (*core.Alert, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	query := "SELECT alert_data FROM alerts WHERE fingerprint = $1"

	var alertData []byte
	err := p.pool.QueryRow(ctx, query, fingerprint).Scan(&alertData)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Alert not found
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	var alert core.Alert
	if err := json.Unmarshal(alertData, &alert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	return &alert, nil
}

// GetAlerts получает список алертов с фильтрами
func (p *PostgresDatabase) GetAlerts(ctx context.Context, filters map[string]any, limit, offset int) ([]*core.Alert, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	query := "SELECT alert_data FROM alerts WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	// Добавляем фильтры
	if status, ok := filters["status"].(string); ok {
		argCount++
		query += fmt.Sprintf(" AND alert_data->>'status' = $%d", argCount)
		args = append(args, status)
	}

	if severity, ok := filters["severity"].(string); ok {
		argCount++
		query += fmt.Sprintf(" AND alert_data->'labels'->>'severity' = $%d", argCount)
		args = append(args, severity)
	}

	if namespace, ok := filters["namespace"].(string); ok {
		argCount++
		query += fmt.Sprintf(" AND alert_data->'labels'->>'namespace' = $%d", argCount)
		args = append(args, namespace)
	}

	// Добавляем сортировку
	query += " ORDER BY alert_data->>'starts_at' DESC"

	// Добавляем пагинацию
	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*core.Alert
	for rows.Next() {
		var alertData []byte
		if err := rows.Scan(&alertData); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		var alert core.Alert
		if err := json.Unmarshal(alertData, &alert); err != nil {
			return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// CleanupOldAlerts удаляет старые алерты из PostgreSQL
func (p *PostgresDatabase) CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("not connected")
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	query := "DELETE FROM alerts WHERE alert_data->>'starts_at' < $1"

	result, err := p.pool.Exec(ctx, query, cutoffDate.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old alerts: %w", err)
	}

	rowsAffected := result.RowsAffected()

	return int(rowsAffected), nil
}

// SaveClassification сохраняет результат классификации
func (p *PostgresDatabase) SaveClassification(ctx context.Context, fingerprint string, result *core.ClassificationResult) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	classificationData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal classification: %w", err)
	}

	query := `
		INSERT INTO classifications (alert_fingerprint, classification_data, confidence, category)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (alert_fingerprint)
		DO UPDATE SET
			classification_data = EXCLUDED.classification_data,
			confidence = EXCLUDED.confidence,
			category = EXCLUDED.category,
			updated_at = NOW()`

	_, err = p.pool.Exec(ctx, query, fingerprint, classificationData, result.Confidence, string(result.Severity))
	if err != nil {
		return fmt.Errorf("failed to save classification: %w", err)
	}

	return nil
}

// GetClassification получает результат классификации
func (p *PostgresDatabase) GetClassification(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	query := "SELECT classification_data FROM classifications WHERE alert_fingerprint = $1"

	var classificationData []byte
	err := p.pool.QueryRow(ctx, query, fingerprint).Scan(&classificationData)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Classification not found
		}
		return nil, fmt.Errorf("failed to get classification: %w", err)
	}

	var result core.ClassificationResult
	if err := json.Unmarshal(classificationData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal classification: %w", err)
	}

	return &result, nil
}

// LogPublishingAttempt логирует попытку публикации
func (p *PostgresDatabase) LogPublishingAttempt(ctx context.Context, fingerprint, targetName string, success bool, errorMessage *string, processingTime *float64) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	query := `
		INSERT INTO publishing_logs (
			alert_fingerprint, target_name, success, error_message,
			processing_time, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())`

	_, err := p.pool.Exec(ctx, query, fingerprint, targetName, success, errorMessage, processingTime)
	if err != nil {
		return fmt.Errorf("failed to log publishing attempt: %w", err)
	}

	return nil
}

// GetPublishingHistory получает историю публикаций для алерта
func (p *PostgresDatabase) GetPublishingHistory(ctx context.Context, fingerprint string) ([]*core.PublishingLog, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	query := `
		SELECT id, alert_fingerprint, target_name, success,
			   error_message, processing_time, created_at
		FROM publishing_logs
		WHERE alert_fingerprint = $1
		ORDER BY created_at DESC`

	rows, err := p.pool.Query(ctx, query, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to query publishing history: %w", err)
	}
	defer rows.Close()

	var logs []*core.PublishingLog
	for rows.Next() {
		log := &core.PublishingLog{}
		err := rows.Scan(
			&log.ID, &log.Fingerprint, &log.TargetName, &log.Success,
			&log.ErrorMessage, &log.ProcessingTime, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan publishing log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// MigrateDown выполняет rollback миграций (не поддерживается для PostgreSQL)
func (p *PostgresDatabase) MigrateDown(ctx context.Context, steps int) error {
	p.logger.Info("PostgreSQL doesn't support complex migrations rollback",
		"recommendation", "Use database backups for rollback")
	return fmt.Errorf("PostgreSQL doesn't support complex migrations rollback")
}

// GetStats возвращает статистику PostgreSQL базы данных
func (p *PostgresDatabase) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	stats := make(map[string]interface{})

	// Общая статистика
	row := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM alerts")
	var alertCount int
	if err := row.Scan(&alertCount); err != nil {
		return nil, fmt.Errorf("failed to get alert count: %w", err)
	}
	stats["alerts_count"] = alertCount

	// Статистика по классификациям
	row = p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM classifications")
	var classificationCount int
	if err := row.Scan(&classificationCount); err != nil {
		return nil, fmt.Errorf("failed to get classification count: %w", err)
	}
	stats["classifications_count"] = classificationCount

	// Статистика по публикациям
	row = p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM publishing_logs")
	var publishingCount int
	if err := row.Scan(&publishingCount); err != nil {
		return nil, fmt.Errorf("failed to get publishing count: %w", err)
	}
	stats["publishing_count"] = publishingCount

	// Connection pool stats
	poolStats := p.pool.Stat()
	stats["total_connections"] = poolStats.TotalConns()
	stats["idle_connections"] = poolStats.IdleConns()
	stats["acquired_connections"] = poolStats.AcquiredConns()
	stats["empty_acquire_count"] = poolStats.EmptyAcquireCount()

	return stats, nil
}

// postgresResult адаптер для преобразования pgconn.CommandTag в sql.Result
type postgresResult struct {
	tag pgconn.CommandTag
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("LastInsertId not supported for PostgreSQL")
}

func (r *postgresResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}
