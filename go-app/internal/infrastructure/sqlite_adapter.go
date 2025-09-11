package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SQLiteDatabase адаптер для SQLite, реализующий общий интерфейс Database
type SQLiteDatabase struct {
	db     *sql.DB
	config *Config
	logger *slog.Logger
}

// NewSQLiteDatabase создает новый SQLite адаптер
func NewSQLiteDatabase(config *Config) (*SQLiteDatabase, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	sqlite := &SQLiteDatabase{
		config: config,
		logger: config.Logger,
	}

	return sqlite, nil
}

// Connect устанавливает соединение с SQLite
func (s *SQLiteDatabase) Connect(ctx context.Context) error {
	dbPath := s.config.SQLiteFile
	if dbPath == "" {
		dbPath = ":memory:"
	}

	// Создаем директорию для файла БД, если указан путь к файлу
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	s.logger.Info("Connecting to SQLite", "filepath", dbPath)

	// Открываем соединение с SQLite
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Настраиваем параметры соединения
	if s.config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(s.config.MaxOpenConns)
	}
	if s.config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(s.config.MaxIdleConns)
	}
	if s.config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(s.config.ConnMaxLifetime)
	}
	if s.config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(s.config.ConnMaxIdleTime)
	}

	// Включаем foreign keys
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Включаем WAL mode для лучшей производительности
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		s.logger.Warn("Failed to enable WAL mode", "error", err)
	}

	// Тестируем соединение
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	s.db = db

	s.logger.Info("Successfully connected to SQLite",
		"filepath", dbPath,
		"max_open_conns", s.config.MaxOpenConns,
		"max_idle_conns", s.config.MaxIdleConns)

	return nil
}

// Disconnect закрывает соединение с SQLite
func (s *SQLiteDatabase) Disconnect(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	s.logger.Info("Disconnecting from SQLite")

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close SQLite database: %w", err)
	}

	s.db = nil
	s.logger.Info("Successfully disconnected from SQLite")

	return nil
}

// IsConnected проверяет состояние соединения
func (s *SQLiteDatabase) IsConnected() bool {
	if s.db == nil {
		return false
	}

	// Проверяем соединение с помощью простого запроса
	return s.db.Ping() == nil
}

// Health выполняет проверку здоровья базы данных
func (s *SQLiteDatabase) Health(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("not connected")
	}

	// Выполняем простой запрос для проверки работоспособности
	_, err := s.db.ExecContext(ctx, "SELECT 1")
	return err
}

// Exec выполняет SQL команду
func (s *SQLiteDatabase) Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	if s.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	return s.db.ExecContext(ctx, sql, args...)
}

// Query выполняет SQL запрос
func (s *SQLiteDatabase) Query(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	if s.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	return s.db.QueryContext(ctx, sql, args...)
}

// QueryRow выполняет SQL запрос и возвращает одну строку
func (s *SQLiteDatabase) QueryRow(ctx context.Context, sql string, args ...interface{}) *sql.Row {
	if s.db == nil {
		return nil
	}

	return s.db.QueryRowContext(ctx, sql, args...)
}

// Begin начинает новую транзакцию
func (s *SQLiteDatabase) Begin(ctx context.Context) (*sql.Tx, error) {
	if s.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	return s.db.BeginTx(ctx, nil)
}

// Migrate выполняет миграции схемы для SQLite
func (s *SQLiteDatabase) Migrate(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("not connected")
	}

	// Создаем таблицу alerts если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS alerts (
		fingerprint TEXT PRIMARY KEY,
		alert_data TEXT NOT NULL, -- JSON как текст для SQLite
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
	CREATE INDEX IF NOT EXISTS idx_alerts_updated_at ON alerts(updated_at);

	-- Триггер для автоматического обновления updated_at
	CREATE TRIGGER IF NOT EXISTS update_alerts_updated_at
		AFTER UPDATE ON alerts
		FOR EACH ROW
		BEGIN
			UPDATE alerts SET updated_at = CURRENT_TIMESTAMP WHERE fingerprint = OLD.fingerprint;
		END;
	`

	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create alerts table: %w", err)
	}

	s.logger.Info("SQLite schema migration completed successfully")
	return nil
}
