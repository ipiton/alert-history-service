package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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

// Migrate выполняет миграции схемы для PostgreSQL
func (p *PostgresDatabase) Migrate(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}

	// Создаем таблицу alerts если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS alerts (
		fingerprint TEXT PRIMARY KEY,
		alert_data JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
	CREATE INDEX IF NOT EXISTS idx_alerts_updated_at ON alerts(updated_at);
	`

	_, err := p.pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create alerts table: %w", err)
	}

	p.logger.Info("PostgreSQL schema migration completed successfully")
	return nil
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
