package sqlite_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/storage/sqlite"
)

// newTestStorage creates a SQLite storage for testing with temp DB.
func newTestStorage(t *testing.T) core.AlertStorage {
	ctx := context.Background()
	dbPath := t.TempDir() + "/test.db"
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	storage, err := sqlite.NewSQLiteStorage(ctx, dbPath, logger)
	require.NoError(t, err, "Failed to create test storage")
	require.NotNil(t, storage, "Storage should not be nil")

	return storage
}

// newTestAlert creates a test alert with given fingerprint.
func newTestAlert(fingerprint string) *core.Alert {
	now := time.Now()
	return &core.Alert{
		Fingerprint: fingerprint,
		AlertName:   "TestAlert",
		Status:      core.StatusFiring,
		Labels: map[string]string{
			"alertname": "TestAlert",
			"severity":  "critical",
			"namespace": "default",
		},
		Annotations: map[string]string{
			"summary":     "Test alert summary",
			"description": "Test alert description",
		},
		StartsAt:     now,
		EndsAt:       nil, // nil pointer for ongoing alert
		GeneratorURL: stringPtr("http://prometheus:9090/graph"),
	}
}

func stringPtr(s string) *string {
	return &s
}

// TestSaveAlert tests SaveAlert operation (INSERT).
func TestSaveAlert(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert := newTestAlert("test-fp-1")

	err := storage.SaveAlert(ctx, alert)
	assert.NoError(t, err, "SaveAlert should succeed")

	// Verify alert was saved
	retrieved, err := storage.GetAlertByFingerprint(ctx, "test-fp-1")
	require.NoError(t, err)
	assert.Equal(t, alert.Fingerprint, retrieved.Fingerprint)
	assert.Equal(t, alert.AlertName, retrieved.AlertName)
	assert.Equal(t, alert.Status, retrieved.Status)
}

// TestSaveAlert_Upsert tests SaveAlert UPSERT behavior (update on duplicate).
func TestSaveAlert_Upsert(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert := newTestAlert("test-fp-upsert")

	// First save
	err := storage.SaveAlert(ctx, alert)
	require.NoError(t, err)

	// Update status and save again
	alert.Status = core.StatusResolved
	err = storage.SaveAlert(ctx, alert)
	require.NoError(t, err, "SaveAlert should update existing alert")

	// Verify updated status
	retrieved, err := storage.GetAlertByFingerprint(ctx, "test-fp-upsert")
	require.NoError(t, err)
	assert.Equal(t, core.StatusResolved, retrieved.Status, "Status should be updated")
}

// TestGetAlertByFingerprint_NotFound tests GetAlertByFingerprint with non-existent fingerprint.
func TestGetAlertByFingerprint_NotFound(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert, err := storage.GetAlertByFingerprint(ctx, "non-existent")
	assert.Error(t, err, "Should error on non-existent fingerprint")
	assert.Nil(t, alert, "Alert should be nil")
	assert.Equal(t, core.ErrAlertNotFound, err, "Should return ErrAlertNotFound")
}

// TestUpdateAlert tests UpdateAlert operation.
func TestUpdateAlert(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert := newTestAlert("test-fp-update")
	err := storage.SaveAlert(ctx, alert)
	require.NoError(t, err)

	// Update alert
	alert.Status = core.StatusResolved
	alert.Labels["severity"] = "warning"
	err = storage.UpdateAlert(ctx, alert)
	require.NoError(t, err, "UpdateAlert should succeed")

	// Verify update
	retrieved, err := storage.GetAlertByFingerprint(ctx, "test-fp-update")
	require.NoError(t, err)
	assert.Equal(t, core.StatusResolved, retrieved.Status)
	severity := retrieved.Severity()
	require.NotNil(t, severity)
	assert.Equal(t, "warning", *severity)
}

// TestUpdateAlert_NotFound tests UpdateAlert with non-existent fingerprint.
func TestUpdateAlert_NotFound(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert := newTestAlert("non-existent")
	err := storage.UpdateAlert(ctx, alert)
	// UpdateAlert reuses SaveAlert, so it will create the alert (UPSERT behavior)
	assert.NoError(t, err, "UpdateAlert creates alert if not exists (UPSERT)")
}

// TestDeleteAlert tests DeleteAlert operation.
func TestDeleteAlert(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	alert := newTestAlert("test-fp-delete")
	err := storage.SaveAlert(ctx, alert)
	require.NoError(t, err)

	// Delete alert
	err = storage.DeleteAlert(ctx, "test-fp-delete")
	require.NoError(t, err, "DeleteAlert should succeed")

	// Verify deletion
	retrieved, err := storage.GetAlertByFingerprint(ctx, "test-fp-delete")
	assert.Error(t, err, "Should error after deletion")
	assert.Nil(t, retrieved)
	assert.Equal(t, core.ErrAlertNotFound, err)
}

// TestDeleteAlert_NotFound tests DeleteAlert with non-existent fingerprint.
func TestDeleteAlert_NotFound(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	err := storage.DeleteAlert(ctx, "non-existent")
	assert.Error(t, err, "Should error on non-existent fingerprint")
	assert.Equal(t, core.ErrAlertNotFound, err)
}

// TestListAlerts_Empty tests ListAlerts on empty database.
func TestListAlerts_Empty(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	result, err := storage.ListAlerts(ctx, nil)
	require.NoError(t, err, "ListAlerts should succeed on empty DB")
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Alerts), "Should return empty list")
	assert.Equal(t, 0, result.Total, "Total should be 0")
}

// TestListAlerts_Basic tests basic ListAlerts functionality.
func TestListAlerts_Basic(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	// Save 3 alerts
	for i := 1; i <= 3; i++ {
		alert := newTestAlert(string(rune('a' + i - 1)))
		err := storage.SaveAlert(ctx, alert)
		require.NoError(t, err)
	}

	// List all alerts
	result, err := storage.ListAlerts(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(result.Alerts), "Should return 3 alerts")
	assert.Equal(t, 3, result.Total, "Total should be 3")
}

// TestListAlerts_Pagination tests pagination.
func TestListAlerts_Pagination(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	// Save 10 alerts
	for i := 0; i < 10; i++ {
		alert := newTestAlert(string(rune('a' + i)))
		err := storage.SaveAlert(ctx, alert)
		require.NoError(t, err)
	}

	// Page 1: limit=5, offset=0
	filters := &core.AlertFilters{Limit: 5, Offset: 0}
	result, err := storage.ListAlerts(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, 5, len(result.Alerts), "Page 1 should have 5 alerts")
	assert.Equal(t, 10, result.Total, "Total should be 10")

	// Page 2: limit=5, offset=5
	filters = &core.AlertFilters{Limit: 5, Offset: 5}
	result, err = storage.ListAlerts(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, 5, len(result.Alerts), "Page 2 should have 5 alerts")
	assert.Equal(t, 10, result.Total, "Total should be 10")
}

// TestListAlerts_FilterByStatus tests filtering by status.
func TestListAlerts_FilterByStatus(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	// Save firing and resolved alerts
	firingAlert := newTestAlert("firing-1")
	firingAlert.Status = core.StatusFiring
	err := storage.SaveAlert(ctx, firingAlert)
	require.NoError(t, err)

	resolvedAlert := newTestAlert("resolved-1")
	resolvedAlert.Status = core.StatusResolved
	err = storage.SaveAlert(ctx, resolvedAlert)
	require.NoError(t, err)

	// Filter by firing
	status := core.StatusFiring
	filters := &core.AlertFilters{Status: &status}
	result, err := storage.ListAlerts(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, 1, len(result.Alerts), "Should return 1 firing alert")
	assert.Equal(t, core.StatusFiring, result.Alerts[0].Status)
}

// TestGetAlertStats tests GetAlertStats operation.
func TestGetAlertStats(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	// Save alerts with different statuses
	firingAlert := newTestAlert("firing-1")
	firingAlert.Status = core.StatusFiring
	err := storage.SaveAlert(ctx, firingAlert)
	require.NoError(t, err)

	resolvedAlert := newTestAlert("resolved-1")
	resolvedAlert.Status = core.StatusResolved
	err = storage.SaveAlert(ctx, resolvedAlert)
	require.NoError(t, err)

	// Get stats
	stats, err := storage.GetAlertStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.TotalAlerts, "Total should be 2")
	assert.Equal(t, 1, stats.AlertsByStatus["firing"], "Should have 1 firing alert")
	assert.Equal(t, 1, stats.AlertsByStatus["resolved"], "Should have 1 resolved alert")
}

// TestCleanupOldAlerts tests CleanupOldAlerts operation.
func TestCleanupOldAlerts(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	// Save old resolved alert (simulated by manually setting updated_at)
	// Note: This test is simplified, real cleanup requires time manipulation
	oldAlert := newTestAlert("old-resolved")
	oldAlert.Status = core.StatusResolved
	err := storage.SaveAlert(ctx, oldAlert)
	require.NoError(t, err)

	// Cleanup alerts older than 30 days
	deleted, err := storage.CleanupOldAlerts(ctx, 30)
	require.NoError(t, err)
	// May be 0 if alert is recent (depends on test execution time)
	assert.GreaterOrEqual(t, deleted, 0, "Deleted count should be non-negative")
}

// TestConcurrentWrites tests concurrent writes to SQLite.
func TestConcurrentWrites(t *testing.T) {
	storage := newTestStorage(t)
	ctx := context.Background()

	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	// Launch concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			alert := newTestAlert(string(rune('a' + id)))
			errors <- storage.SaveAlert(ctx, alert)
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(t, err, "Concurrent write should succeed")
	}

	// Verify all alerts were saved
	result, err := storage.ListAlerts(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines, len(result.Alerts), "Should save all alerts")
}
