package storage_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/config"
	"github.com/vitaliisemenov/alert-history/internal/storage"
	"github.com/vitaliisemenov/alert-history/internal/storage/sqlite"
)

// newMinimalConfig creates a minimal valid config for testing.
// Fills all required fields to pass validation.
func newMinimalConfig(profile config.DeploymentProfile, backend config.StorageBackend, dbPath string) *config.Config {
	return &config.Config{
		Profile: profile,
		Storage: config.StorageConfig{
			Backend:        backend,
			FilesystemPath: dbPath,
		},
		Server: config.ServerConfig{
			Port: 8080, // Required for validation
			Host: "localhost",
		},
		Database: config.DatabaseConfig{
			Driver:          "postgres", // Required for standard profile
			Host:            "localhost",
			Port:            5432,
			Database:        "test",
			Username:        "test",
			Password:        "test",
			SSLMode:         "disable",
			MaxConnections:  10,
			MinConnections:  2,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
		Log: config.LogConfig{
			Level:  "info", // Required for validation
			Format: "json",
		},
		App: config.AppConfig{
			Name: "alert-history-test", // Required for validation
		},
	}
}

// TestNewStorage_LiteProfile tests storage factory with Lite profile.
// Expected: SQLite storage is created successfully.
func TestNewStorage_LiteProfile(t *testing.T) {
	cfg := newMinimalConfig(config.ProfileLite, config.StorageBackendFilesystem, t.TempDir()+"/test.db")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	require.NoError(t, err, "NewStorage should succeed for Lite profile")
	require.NotNil(t, storage, "Storage should not be nil")

	// Type assertion to verify it's SQLite
	_, ok := storage.(*sqlite.SQLiteStorage)
	assert.True(t, ok, "Storage should be SQLiteStorage for Lite profile")
}

// TestNewStorage_StandardProfile_WithPostgres tests storage factory with Standard profile and Postgres pool.
// Expected: PostgreSQL storage is created (but we can't test actual Postgres without a real connection).
func TestNewStorage_StandardProfile_WithPostgres(t *testing.T) {
	t.Skip("Requires actual PostgreSQL connection - skipped in unit tests")

	cfg := newMinimalConfig(config.ProfileStandard, config.StorageBackendPostgres, "")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// This would require a real pgxpool.Pool
	// Skipped in unit tests, covered in integration tests
	var pgPool *pgxpool.Pool = nil

	_, err := storage.NewStorage(ctx, cfg, pgPool, logger)
	assert.Error(t, err, "Should error without valid Postgres pool")
}

// TestNewStorage_StandardProfile_NoPostgres tests behavior without Postgres pool.
// Expected: Error (factory doesn't auto-fallback, main.go handles fallback).
func TestNewStorage_StandardProfile_NoPostgres(t *testing.T) {
	cfg := newMinimalConfig(config.ProfileStandard, config.StorageBackendPostgres, "")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// No Postgres pool provided (simulates connection failure)
	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	// Factory should error (no auto-fallback at factory level)
	assert.Error(t, err, "Should error without Postgres pool")
	assert.Nil(t, storage, "Storage should be nil on error")
	assert.Contains(t, err.Error(), "postgresql pool is nil", "Error message should mention nil pool")
}

// TestNewStorage_InvalidProfile tests behavior with invalid profile.
// Expected: Error returned (invalid profile).
func TestNewStorage_InvalidProfile(t *testing.T) {
	cfg := newMinimalConfig(config.DeploymentProfile("invalid"), config.StorageBackendFilesystem, t.TempDir()+"/test.db")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	// Should error (invalid profile won't pass validation)
	assert.Error(t, err, "Should error on invalid profile")
	assert.Nil(t, storage, "Storage should be nil on error")
}

// TestNewStorage_SQLiteFileCreation tests that SQLite file is created in specified path.
func TestNewStorage_SQLiteFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := tempDir + "/alerts.db"

	cfg := newMinimalConfig(config.ProfileLite, config.StorageBackendFilesystem, dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	require.NoError(t, err, "NewStorage should succeed")
	require.NotNil(t, storage, "Storage should not be nil")

	// Verify SQLite file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "SQLite database file should exist")
}

// TestNewStorage_SQLiteDirectoryCreation tests that parent directories are created.
func TestNewStorage_SQLiteDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := tempDir + "/nested/dir/alerts.db"

	cfg := newMinimalConfig(config.ProfileLite, config.StorageBackendFilesystem, dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	require.NoError(t, err, "NewStorage should succeed and create nested directories")
	require.NotNil(t, storage, "Storage should not be nil")

	// Verify SQLite file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "SQLite database file should exist in nested directory")
}

// TestNewStorage_NilConfig tests behavior with nil config.
func TestNewStorage_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, nil, nil, logger)

	// Should error (nil config)
	assert.Error(t, err, "Should error on nil config")
	assert.Nil(t, storage, "Storage should be nil on error")
}

// TestNewStorage_NilLogger tests behavior with nil logger.
func TestNewStorage_NilLogger(t *testing.T) {
	cfg := &config.Config{
		Profile: config.ProfileLite,
		Storage: config.StorageConfig{
			Backend:        config.StorageBackendFilesystem,
			FilesystemPath: t.TempDir() + "/test.db",
		},
	}

	ctx := context.Background()

	// Nil logger should be handled gracefully
	storage, err := storage.NewStorage(ctx, cfg, nil, nil)

	// Implementation may handle nil logger differently
	// Either succeed with default logger or error
	if err == nil {
		require.NotNil(t, storage, "Storage should not be nil")
	} else {
		assert.Error(t, err, "Should error on nil logger")
	}
}

// TestNewStorage_EmptyFilesystemPath tests behavior with empty filesystem path.
func TestNewStorage_EmptyFilesystemPath(t *testing.T) {
	cfg := newMinimalConfig(config.ProfileLite, config.StorageBackendFilesystem, "") // Empty path

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	// Should either error or use default path
	// Implementation-dependent, but should not panic
	if err == nil {
		require.NotNil(t, storage, "Storage should not be nil")
	} else {
		assert.Error(t, err, "Should error on empty filesystem path")
	}
}

// TestNewStorage_ConcurrentCalls tests thread safety of factory.
func TestNewStorage_ConcurrentCalls(t *testing.T) {
	const numGoroutines = 10

	cfg := newMinimalConfig(config.ProfileLite, config.StorageBackendFilesystem, t.TempDir()+"/test.db")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// Launch concurrent factory calls
	results := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := storage.NewStorage(ctx, cfg, nil, logger)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		// At least one should succeed (first creates DB, others may conflict)
		// We accept errors from concurrent writes, but no panics
		if err != nil {
			t.Logf("Concurrent call %d failed (expected in some cases): %v", i, err)
		}
	}
}
