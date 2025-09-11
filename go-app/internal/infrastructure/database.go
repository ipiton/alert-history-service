package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"log/slog"
)

// Database определяет общий интерфейс для работы с базой данных
// Поддерживает как PostgreSQL, так и SQLite адаптеры
type Database interface {
	// Lifecycle management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool

	// Health monitoring
	Health(ctx context.Context) error

	// Query execution (обобщенные методы для совместимости)
	Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) *sql.Row

	// Transaction support
	Begin(ctx context.Context) (*sql.Tx, error)

	// Schema management
	Migrate(ctx context.Context) error
}

// Config определяет конфигурацию для базы данных
type Config struct {
	Driver   string // "postgres" или "sqlite"
	DSN      string
	Logger   *slog.Logger

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// SQLite specific
	SQLiteFile string
}

// NewDatabase создает новый экземпляр базы данных на основе конфигурации
func NewDatabase(config *Config) (Database, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	switch config.Driver {
	case "postgres":
		return NewPostgresDatabase(config)
	case "sqlite":
		return NewSQLiteDatabase(config)
	default:
		return nil, &UnsupportedDriverError{Driver: config.Driver}
	}
}

// UnsupportedDriverError возвращается когда указан неподдерживаемый драйвер БД
type UnsupportedDriverError struct {
	Driver string
}

func (e *UnsupportedDriverError) Error() string {
	return "unsupported database driver: " + e.Driver
}
