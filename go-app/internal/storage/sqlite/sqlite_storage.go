// Package sqlite implements core.AlertStorage interface using SQLite embedded database.
// Designed for Lite deployment profile (single-node, no external dependencies).
//
// Features:
//   - WAL mode enabled (concurrent reads during writes)
//   - Foreign keys enabled (data integrity)
//   - Secure file permissions (0600, owner read/write only)
//   - Thread-safe operations (RWMutex)
//   - UPSERT logic (idempotent CreateAlert)
//   - Compatible schema with PostgreSQL adapter
//
// Performance Targets:
//   - CreateAlert: < 3ms (p95)
//   - GetAlert: < 1ms (p95)
//   - ListAlerts (100 rows): < 20ms (p95)
//   - CountAlerts: < 5ms (p95)
//
// Use Cases:
//   - Development environments (no Postgres required)
//   - Testing environments (fast, isolated)
//   - Small-scale production (< 1K alerts/day)
//   - Edge deployments (no network dependencies)
//
// Limitations:
//   - No horizontal scaling (single-node only)
//   - Limited concurrency (max 10 connections)
//   - Disk space constrained (PVC size)
//   - No HA support (single file)
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// Pure Go SQLite driver (no CGO, easier cross-compilation)
	_ "modernc.org/sqlite"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// SQLiteStorage implements core.AlertStorage interface using SQLite database.
// Thread-safe for concurrent access (up to 10 goroutines).
type SQLiteStorage struct {
	db     *sql.DB      // SQLite database connection
	logger *slog.Logger // Structured logger
	path   string       // File path to SQLite database
	mu     sync.RWMutex // Protects connection state (not data - SQLite handles that)
}

// NewSQLiteStorage creates a new SQLite storage instance.
// Path must be absolute or relative to current working directory.
// File will be created with mode 0600 (owner read/write only).
// Parent directory will be created with mode 0700 if not exists.
//
// Configuration:
//   - WAL mode enabled (?_journal_mode=WAL)
//   - Shared cache enabled (?cache=shared)
//   - Foreign keys enabled (PRAGMA foreign_keys=ON)
//   - Max open connections: 10 (Lite profile, single-node)
//   - Max idle connections: 5 (keep some connections warm)
//   - Connection max lifetime: 1 hour
//   - Connection max idle time: 10 minutes
//
// Performance:
//   - File creation: ~10-20ms
//   - Schema initialization: ~30-50ms
//   - Total initialization: ~50-100ms
func NewSQLiteStorage(
	ctx context.Context,
	path string,
	logger *slog.Logger,
) (*SQLiteStorage, error) {
	// Validate path
	if path == "" {
		return nil, fmt.Errorf("sqlite path cannot be empty")
	}

	// Security: Prevent directory traversal attacks
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path contains '..': %s", path)
	}

	// Security: Prevent access to forbidden system directories
	forbiddenPrefixes := []string{"/etc", "/sys", "/proc", "/dev"}
	for _, prefix := range forbiddenPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil, fmt.Errorf("forbidden path prefix %s: %s", prefix, path)
		}
	}

	// Create parent directory if not exists (mode 0700, owner only)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database connection with optimal flags
	// ?cache=shared: Enable shared cache for concurrent access
	// ?mode=rwc: Read-write-create mode
	// ?_journal_mode=WAL: Write-Ahead Logging for better concurrency
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Configure connection pool
	// Lite profile: single-node deployment, limited concurrency
	db.SetMaxOpenConns(10)                   // Max 10 concurrent connections
	db.SetMaxIdleConns(5)                    // Keep 5 connections warm
	db.SetConnMaxLifetime(time.Hour)         // Recycle connections every hour
	db.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections after 10 minutes

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}

	// Enable foreign keys (important for data integrity)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create storage instance
	s := &SQLiteStorage{
		db:     db,
		logger: logger,
		path:   path,
	}

	// Initialize schema (tables + indexes)
	if err := s.initSchema(ctx); err != nil {
		db.Close()
		return nil, err // Already wrapped in ErrSchemaInitFailed
	}

	// Set secure file permissions (0600, owner read/write only)
	if err := os.Chmod(path, 0600); err != nil {
		logger.Warn("Failed to set file permissions to 0600",
			"path", path,
			"error", err,
		)
	}

	logger.Info("SQLite storage initialized",
		"path", path,
		"wal_mode", true,
		"max_open_conns", 10,
		"max_idle_conns", 5,
	)

	// Note: Metrics should be set by caller (factory.go) to avoid circular import

	return s, nil
}

// initSchema creates necessary tables and indexes.
// Schema is compatible with PostgreSQL adapter (same column names, types).
// Uses INTEGER for timestamps (Unix milliseconds, compatible with Go time.Time).
func (s *SQLiteStorage) initSchema(ctx context.Context) error {
	schema := `
-- Alerts table (compatible with PostgreSQL schema)
CREATE TABLE IF NOT EXISTS alerts (
    fingerprint TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK(status IN ('firing', 'resolved')),
    severity TEXT NOT NULL CHECK(severity IN ('critical', 'warning', 'info', 'unknown')),
    namespace TEXT NOT NULL,
    alert_name TEXT NOT NULL,
    labels TEXT NOT NULL,        -- JSON string (SQLite doesn't have native JSONB)
    annotations TEXT NOT NULL,   -- JSON string
    starts_at INTEGER NOT NULL,  -- Unix timestamp in milliseconds
    ends_at INTEGER,              -- Unix timestamp in milliseconds (nullable)
    generator_url TEXT,          -- Source URL (nullable)
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_namespace ON alerts(namespace);
CREATE INDEX IF NOT EXISTS idx_alerts_alert_name ON alerts(alert_name);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_starts_at ON alerts(starts_at);
`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	s.logger.Debug("SQLite schema initialized",
		"tables", 1,
		"indexes", 6,
	)

	return nil
}

// CreateAlert implements core.AlertStorage.CreateAlert.
// Uses UPSERT logic (INSERT ... ON CONFLICT DO UPDATE) for idempotency.
// If fingerprint already exists, updates status, severity, timestamps.
//
// Performance: < 3ms (p95)
// Thread-safe: Yes (SQLite handles locking)
func (s *SQLiteStorage) CreateAlert(ctx context.Context, alert *core.Alert) error {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Serialize labels and annotations to JSON
	labelsJSON, err := json.Marshal(alert.Labels)
	if err != nil {
		storage.RecordError("create", "sqlite", storage.ErrorTypeValidation)
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	annotationsJSON, err := json.Marshal(alert.Annotations)
	if err != nil {
		storage.RecordError("create", "sqlite", storage.ErrorTypeValidation)
		return fmt.Errorf("failed to marshal annotations: %w", err)
	}

	// Convert timestamps to Unix milliseconds
	startsAt := alert.StartsAt.UnixMilli()
	var endsAt *int64
	if !alert.EndsAt.IsZero() {
		ms := alert.EndsAt.UnixMilli()
		endsAt = &ms
	}

	// UPSERT: Insert or update if fingerprint exists
	// This makes CreateAlert idempotent (can be called multiple times safely)
	query := `
INSERT INTO alerts (
    fingerprint, status, severity, namespace, alert_name,
    labels, annotations, starts_at, ends_at, generator_url
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(fingerprint) DO UPDATE SET
    status = excluded.status,
    severity = excluded.severity,
    labels = excluded.labels,
    annotations = excluded.annotations,
    ends_at = excluded.ends_at,
    updated_at = strftime('%s', 'now') * 1000
`

	_, err = s.db.ExecContext(ctx, query,
		alert.Fingerprint,
		alert.Status,
		alert.Severity,
		alert.Namespace,
		alert.AlertName,
		string(labelsJSON),
		string(annotationsJSON),
		startsAt,
		endsAt,
		alert.GeneratorURL,
	)

	if err != nil {
		// Record error metric
		storage.RecordOperation("create", "sqlite", "error")
		return fmt.Errorf("failed to create alert: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	storage.RecordOperation("create", "sqlite", "success")
	storage.RecordOperationDuration("create", "sqlite", duration.Seconds())

	s.logger.Debug("Alert created/updated",
		"fingerprint", alert.Fingerprint,
		"status", alert.Status,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// GetAlert implements core.AlertStorage.GetAlert.
// Retrieves alert by fingerprint (primary key).
//
// Performance: < 1ms (p95, indexed lookup)
// Thread-safe: Yes (read-only operation)
func (s *SQLiteStorage) GetAlert(ctx context.Context, fingerprint string) (*core.Alert, error) {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
SELECT fingerprint, status, severity, namespace, alert_name,
       labels, annotations, starts_at, ends_at, generator_url,
       created_at, updated_at
FROM alerts
WHERE fingerprint = ?
`

	var alert core.Alert
	var labelsJSON, annotationsJSON string
	var startsAt, createdAt, updatedAt int64
	var endsAt *int64
	var generatorURL sql.NullString

	err := s.db.QueryRowContext(ctx, query, fingerprint).Scan(
		&alert.Fingerprint,
		&alert.Status,
		&alert.Severity,
		&alert.Namespace,
		&alert.AlertName,
		&labelsJSON,
		&annotationsJSON,
		&startsAt,
		&endsAt,
		&generatorURL,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// Alert not found (not an error, expected case)
		storage.RecordOperation("get", "sqlite", "not_found")
		return nil, core.ErrAlertNotFound{Fingerprint: fingerprint}
	}

	if err != nil {
		// Actual error (connection, query, etc.)
		storage.RecordOperation("get", "sqlite", "error")
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(labelsJSON), &alert.Labels); err != nil {
		storage.RecordError("get", "sqlite", storage.ErrorTypeValidation)
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	if err := json.Unmarshal([]byte(annotationsJSON), &alert.Annotations); err != nil {
		storage.RecordError("get", "sqlite", storage.ErrorTypeValidation)
		return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	// Convert timestamps
	alert.StartsAt = time.UnixMilli(startsAt)
	if endsAt != nil {
		alert.EndsAt = time.UnixMilli(*endsAt)
	}
	alert.CreatedAt = time.UnixMilli(createdAt)
	alert.UpdatedAt = time.UnixMilli(updatedAt)

	// Handle nullable generator_url
	if generatorURL.Valid {
		alert.GeneratorURL = generatorURL.String
	}

	// Record success metrics
	duration := time.Since(startTime)
	storage.RecordOperation("get", "sqlite", "success")
	storage.RecordOperationDuration("get", "sqlite", duration.Seconds())

	return &alert, nil
}

// UpdateAlert implements core.AlertStorage.UpdateAlert.
// Reuses CreateAlert logic (UPSERT handles both insert and update).
func (s *SQLiteStorage) UpdateAlert(ctx context.Context, alert *core.Alert) error {
	return s.CreateAlert(ctx, alert)
}

// DeleteAlert implements core.AlertStorage.DeleteAlert.
// Removes alert by fingerprint.
//
// Performance: < 2ms (p95)
// Thread-safe: Yes (SQLite handles locking)
func (s *SQLiteStorage) DeleteAlert(ctx context.Context, fingerprint string) error {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `DELETE FROM alerts WHERE fingerprint = ?`
	result, err := s.db.ExecContext(ctx, query, fingerprint)

	if err != nil {
		storage.RecordOperation("delete", "sqlite", "error")
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	// Check if alert existed
	rows, _ := result.RowsAffected()
	if rows == 0 {
		storage.RecordOperation("delete", "sqlite", "not_found")
		return core.ErrAlertNotFound{Fingerprint: fingerprint}
	}

	// Record success metrics
	duration := time.Since(startTime)
	storage.RecordOperation("delete", "sqlite", "success")
	storage.RecordOperationDuration("delete", "sqlite", duration.Seconds())

	s.logger.Debug("Alert deleted",
		"fingerprint", fingerprint,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// Close implements core.AlertStorage.Close.
// Gracefully closes database connection.
// Idempotent (can be called multiple times).
func (s *SQLiteStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		err := s.db.Close()
		s.db = nil // Prevent double-close

		if err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}

		s.logger.Info("SQLite storage closed", "path", s.path)
		storage.SetHealthStatus("sqlite", 0) // 0 = unhealthy (closed)
	}

	return nil
}

// Health implements core.AlertStorage.Health.
// Checks database connection liveness via Ping.
//
// Performance: < 100ms
// Thread-safe: Yes
func (s *SQLiteStorage) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		storage.SetHealthStatus("sqlite", 0) // unhealthy
		return fmt.Errorf("database connection is nil")
	}

	err := s.db.PingContext(ctx)
	if err != nil {
		storage.SetHealthStatus("sqlite", 0) // unhealthy
		return fmt.Errorf("health check failed: %w", err)
	}

	storage.SetHealthStatus("sqlite", 1) // healthy
	return nil
}

// GetFileSize returns current SQLite file size in bytes.
// Returns 0 if file doesn't exist (no error).
// Thread-safe (reads file system, not database).
func (s *SQLiteStorage) GetFileSize() int64 {
	info, err := os.Stat(s.path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// GetPath returns SQLite database file path.
func (s *SQLiteStorage) GetPath() string {
	return s.path
}
