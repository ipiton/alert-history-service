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

// SaveAlert implements core.AlertStorage.SaveAlert.
// Uses UPSERT logic (INSERT ... ON CONFLICT DO UPDATE) for idempotency.
// If fingerprint already exists, updates status, severity, timestamps.
//
// Performance: < 3ms (p95)
// Thread-safe: Yes (SQLite handles locking)
func (s *SQLiteStorage) SaveAlert(ctx context.Context, alert *core.Alert) error {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Serialize labels and annotations to JSON
	labelsJSON, err := json.Marshal(alert.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	annotationsJSON, err := json.Marshal(alert.Annotations)
	if err != nil {
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
		return fmt.Errorf("failed to save alert: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)

	s.logger.Debug("Alert saved/updated",
		"fingerprint", alert.Fingerprint,
		"status", alert.Status,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// GetAlertByFingerprint implements core.AlertStorage.GetAlertByFingerprint.
// Retrieves alert by fingerprint (primary key).
//
// Performance: < 1ms (p95, indexed lookup)
// Thread-safe: Yes (read-only operation)
func (s *SQLiteStorage) GetAlertByFingerprint(ctx context.Context, fingerprint string) (*core.Alert, error) {
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
	var severity, namespace string
	var startsAt, createdAt, updatedAt int64
	var endsAtMs sql.NullInt64
	var generatorURL sql.NullString

	err := s.db.QueryRowContext(ctx, query, fingerprint).Scan(
		&alert.Fingerprint,
		&alert.Status,
		&severity,
		&namespace,
		&alert.AlertName,
		&labelsJSON,
		&annotationsJSON,
		&startsAt,
		&endsAtMs,
		&generatorURL,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// Alert not found (not an error, expected case)
		return nil, core.ErrAlertNotFound
	}

	if err != nil {
		// Actual error (connection, query, etc.)
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(labelsJSON), &alert.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	if err := json.Unmarshal([]byte(annotationsJSON), &alert.Annotations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	// Set severity and namespace in labels (where they're stored)
	if alert.Labels == nil {
		alert.Labels = make(map[string]string)
	}
	if severity != "" {
		alert.Labels["severity"] = severity
	}
	if namespace != "" {
		alert.Labels["namespace"] = namespace
	}

	// Convert timestamps
	alert.StartsAt = time.UnixMilli(startsAt)
	if endsAtMs.Valid {
		endsAt := time.UnixMilli(endsAtMs.Int64)
		alert.EndsAt = &endsAt
	}

	// Handle nullable generator_url
	if generatorURL.Valid {
		genURL := generatorURL.String
		alert.GeneratorURL = &genURL
	}

	// Record success metrics
	duration := time.Since(startTime)

	s.logger.Debug("Alert retrieved",
		"fingerprint", fingerprint,
		"duration_ms", duration.Milliseconds(),
	)

	return &alert, nil
}

// UpdateAlert implements core.AlertStorage.UpdateAlert.
// Reuses SaveAlert logic (UPSERT handles both insert and update).
func (s *SQLiteStorage) UpdateAlert(ctx context.Context, alert *core.Alert) error {
	return s.SaveAlert(ctx, alert)
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
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	// Check if alert existed
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return core.ErrAlertNotFound
	}

	// Record success metrics
	duration := time.Since(startTime)

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
		return fmt.Errorf("database connection is nil")
	}

	err := s.db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// GetAlertStats implements core.AlertStorage.GetAlertStats.
// Aggregates statistics about alerts by status and severity.
//
// Performance: < 10ms (simple COUNT queries)
// Thread-safe: Yes (read-only operation)
func (s *SQLiteStorage) GetAlertStats(ctx context.Context) (*core.AlertStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &core.AlertStats{
		AlertsByStatus:   make(map[string]int),
		AlertsBySeverity: make(map[string]int),
	}

	// Get total count
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM alerts").Scan(&stats.TotalAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get alerts by status
	rows, err := s.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM alerts GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		stats.AlertsByStatus[status] = count
	}

	// Get alerts by severity
	rows, err = s.db.QueryContext(ctx, "SELECT severity, COUNT(*) FROM alerts WHERE severity != '' GROUP BY severity")
	if err != nil {
		return nil, fmt.Errorf("failed to get severity counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity row: %w", err)
		}
		stats.AlertsBySeverity[severity] = count
	}

	s.logger.Debug("Alert stats retrieved",
		"total", stats.TotalAlerts,
		"by_status", stats.AlertsByStatus,
		"by_severity", stats.AlertsBySeverity,
	)

	return stats, nil
}

// CleanupOldAlerts implements core.AlertStorage.CleanupOldAlerts.
// Removes resolved alerts older than retentionDays.
// Only resolved alerts are deleted (firing alerts are kept).
//
// Performance: < 100ms for 10K alerts (indexed DELETE)
// Thread-safe: Yes (write lock)
func (s *SQLiteStorage) CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate cutoff timestamp (Unix milliseconds)
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()

	// Delete only resolved alerts older than retention period
	// Firing alerts are never deleted (they represent active issues)
	query := `
		DELETE FROM alerts
		WHERE status = ?
		  AND updated_at < ?
	`
	result, err := s.db.ExecContext(ctx, query, string(core.StatusResolved), cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old alerts: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	s.logger.Info("Cleaned up old resolved alerts",
		"retention_days", retentionDays,
		"deleted_count", deleted,
	)

	return int(deleted), nil
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
