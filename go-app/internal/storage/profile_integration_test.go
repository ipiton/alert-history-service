package storage_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/config"
	"github.com/vitaliisemenov/alert-history/internal/storage"
	"github.com/vitaliisemenov/alert-history/internal/storage/sqlite"
)

// TestProfileIntegration_Lite tests Lite profile with SQLite.
// This is an integration test that validates the full profile → storage flow.
func TestProfileIntegration_Lite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Lite profile config
	cfg := newLiteConfig(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// Initialize storage via factory
	storage, err := storage.NewStorage(ctx, cfg, nil, logger)
	require.NoError(t, err, "Lite profile should initialize successfully")
	require.NotNil(t, storage, "Storage should not be nil")

	// Verify it's SQLite
	_, ok := storage.(*sqlite.SQLiteStorage)
	assert.True(t, ok, "Lite profile should use SQLiteStorage")

	t.Logf("✅ Lite profile integration test PASSED")
}

// TestProfileIntegration_Standard_WithoutPostgres tests Standard profile without Postgres.
// Expected: Error (no auto-fallback at factory level).
func TestProfileIntegration_Standard_WithoutPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Standard profile config
	cfg := newStandardConfig(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// Initialize storage without Postgres pool (simulates connection failure)
	storage, err := storage.NewStorage(ctx, cfg, nil, logger)

	// Factory should error (no auto-fallback at this level)
	assert.Error(t, err, "Standard profile without Postgres should error")
	assert.Nil(t, storage, "Storage should be nil on error")
	assert.Contains(t, err.Error(), "postgresql pool is nil", "Error should mention nil pool")

	t.Logf("✅ Standard profile (without Postgres) integration test PASSED")
}

// newLiteConfig creates a valid Lite profile config for testing.
func newLiteConfig(t *testing.T) *config.Config {
	return &config.Config{
		Profile: config.ProfileLite,
		Storage: config.StorageConfig{
			Backend:        config.StorageBackendFilesystem,
			FilesystemPath: t.TempDir() + "/lite-test.db",
		},
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Database: config.DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			Database:        "test",
			Username:        "test",
			Password:        "test",
			SSLMode:         "disable",
			MaxConnections:  10,
			MinConnections:  2,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
		App: config.AppConfig{
			Name: "alert-history-test",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
	}
}

// newStandardConfig creates a valid Standard profile config for testing.
func newStandardConfig(t *testing.T) *config.Config {
	return &config.Config{
		Profile: config.ProfileStandard,
		Storage: config.StorageConfig{
			Backend: config.StorageBackendPostgres,
		},
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Database: config.DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			Database:        "test",
			Username:        "test",
			Password:        "test",
			SSLMode:         "disable",
			MaxConnections:  10,
			MinConnections:  2,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
		App: config.AppConfig{
			Name: "alert-history-test",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
	}
}
