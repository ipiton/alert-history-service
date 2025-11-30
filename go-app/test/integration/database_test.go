//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabase_Connection tests database connectivity
func TestDatabase_Connection(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err, "failed to setup infrastructure")
	defer infra.Teardown(ctx)

	// Test ping
	err = infra.DB.PingContext(ctx)
	assert.NoError(t, err, "should ping database successfully")

	// Test query
	var result int
	err = infra.DB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

// TestDatabase_CRUD tests basic CRUD operations
func TestDatabase_CRUD(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	t.Run("create alert", func(t *testing.T) {
		alert := NewTestAlert("CRUDTest").WithSeverity("critical")
		alerts := []*Alert{alert}

		err := helper.SeedTestData(ctx, alerts)
		require.NoError(t, err, "should insert alert")

		// Verify inserted
		count, err := helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("read alert", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		alert := NewTestAlert("ReadTest").
			WithSeverity("warning").
			WithNamespace("production")

		err := helper.SeedTestData(ctx, []*Alert{alert})
		require.NoError(t, err)

		// Read back
		retrieved, err := helper.GetAlertFromDB(ctx, alert.Fingerprint)
		require.NoError(t, err)
		assert.Equal(t, alert.AlertName, retrieved.AlertName)
		assert.Equal(t, alert.Severity, retrieved.Severity)
		assert.Equal(t, alert.Namespace, retrieved.Namespace)
	})

	t.Run("update alert", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		alert := NewTestAlert("UpdateTest").WithSeverity("warning")
		err := helper.SeedTestData(ctx, []*Alert{alert})
		require.NoError(t, err)

		// Update status to resolved
		query := `UPDATE alert_history SET status = $1 WHERE fingerprint = $2`
		_, err = infra.DB.ExecContext(ctx, query, "resolved", alert.Fingerprint)
		require.NoError(t, err)

		// Verify updated
		retrieved, err := helper.GetAlertFromDB(ctx, alert.Fingerprint)
		require.NoError(t, err)
		assert.Equal(t, "resolved", retrieved.Status)
	})

	t.Run("delete alert", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		alert := NewTestAlert("DeleteTest")
		err := helper.SeedTestData(ctx, []*Alert{alert})
		require.NoError(t, err)

		// Delete
		query := `DELETE FROM alert_history WHERE fingerprint = $1`
		result, err := infra.DB.ExecContext(ctx, query, alert.Fingerprint)
		require.NoError(t, err)

		rows, err := result.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rows)

		// Verify deleted
		_, err = helper.GetAlertFromDB(ctx, alert.Fingerprint)
		assert.Error(t, err, "should not find deleted alert")
	})
}

// TestDatabase_BulkInsert tests bulk insert performance
func TestDatabase_BulkInsert(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// Create 100 alerts
	alerts := make([]*Alert, 100)
	for i := 0; i < 100; i++ {
		alerts[i] = NewTestAlert("BulkAlert").
			WithSeverity("warning").
			WithNamespace("production")
	}

	start := time.Now()
	err = helper.SeedTestData(ctx, alerts)
	duration := time.Since(start)

	require.NoError(t, err, "should insert 100 alerts")
	assert.Less(t, duration, 1*time.Second, "bulk insert should be fast")

	// Verify count
	count, err := helper.GetAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// TestDatabase_Transactions tests transaction behavior
func TestDatabase_Transactions(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	infra.ResetDatabase(ctx)

	t.Run("commit transaction", func(t *testing.T) {
		tx, err := infra.DB.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Insert in transaction
		query := `
			INSERT INTO alert_history
			(fingerprint, alert_name, status, severity, namespace, labels, annotations, starts_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err = tx.ExecContext(ctx, query,
			"fp_commit_test", "CommitTest", "firing", "warning", "default",
			`{"alertname":"CommitTest"}`, `{"summary":"test"}`,
			time.Now(), time.Now())
		require.NoError(t, err)

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify persisted
		var count int
		err = infra.DB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM alert_history WHERE fingerprint = $1",
			"fp_commit_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("rollback transaction", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		tx, err := infra.DB.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Insert in transaction
		query := `
			INSERT INTO alert_history
			(fingerprint, alert_name, status, severity, namespace, labels, annotations, starts_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err = tx.ExecContext(ctx, query,
			"fp_rollback_test", "RollbackTest", "firing", "warning", "default",
			`{"alertname":"RollbackTest"}`, `{"summary":"test"}`,
			time.Now(), time.Now())
		require.NoError(t, err)

		// Rollback
		err = tx.Rollback()
		require.NoError(t, err)

		// Verify NOT persisted
		var count int
		err = infra.DB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM alert_history WHERE fingerprint = $1",
			"fp_rollback_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "rolled back data should not persist")
	})

	t.Run("transaction isolation", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		// Start transaction 1
		tx1, err := infra.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		})
		require.NoError(t, err)
		defer tx1.Rollback()

		// Insert in tx1
		_, err = tx1.ExecContext(ctx, `
			INSERT INTO alert_history
			(fingerprint, alert_name, status, severity, namespace, labels, annotations, starts_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, "fp_iso_test", "IsolationTest", "firing", "warning", "default",
			`{"alertname":"IsolationTest"}`, `{"summary":"test"}`,
			time.Now(), time.Now())
		require.NoError(t, err)

		// Start transaction 2 (should not see uncommitted data)
		tx2, err := infra.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		})
		require.NoError(t, err)
		defer tx2.Rollback()

		var count int
		err = tx2.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM alert_history WHERE fingerprint = $1",
			"fp_iso_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "tx2 should not see uncommitted data from tx1")
	})
}

// TestDatabase_ConcurrentWrites tests concurrent write operations
func TestDatabase_ConcurrentWrites(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// 10 concurrent writers
	const numWriters = 10
	done := make(chan error, numWriters)

	for i := 0; i < numWriters; i++ {
		go func(n int) {
			alert := NewTestAlert("ConcurrentTest").
				WithSeverity("warning").
				WithNamespace("default")

			err := helper.SeedTestData(ctx, []*Alert{alert})
			done <- err
		}(i)
	}

	// Wait for all writes
	for i := 0; i < numWriters; i++ {
		err := <-done
		assert.NoError(t, err, "concurrent write should succeed")
	}

	// Verify all written
	count, err := helper.GetAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, numWriters, count)
}

// TestDatabase_QueryPerformance tests query performance
func TestDatabase_QueryPerformance(t *testing.T) {
	t.Skip("Performance test - run manually when needed")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// Seed 1000 alerts
	alerts := make([]*Alert, 1000)
	for i := 0; i < 1000; i++ {
		alerts[i] = NewTestAlert("PerfTest").
			WithSeverity("warning").
			WithNamespace("production")
	}
	err = helper.SeedTestData(ctx, alerts)
	require.NoError(t, err)

	t.Run("simple select", func(t *testing.T) {
		start := time.Now()

		var count int
		err := infra.DB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM alert_history").Scan(&count)
		require.NoError(t, err)

		duration := time.Since(start)
		assert.Less(t, duration, 100*time.Millisecond,
			"simple query should be < 100ms")
	})

	t.Run("filtered select", func(t *testing.T) {
		start := time.Now()

		rows, err := infra.DB.QueryContext(ctx, `
			SELECT fingerprint, alert_name, severity
			FROM alert_history
			WHERE severity = $1 AND namespace = $2
			LIMIT 10
		`, "warning", "production")
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}

		duration := time.Since(start)
		assert.Less(t, duration, 100*time.Millisecond,
			"filtered query should be < 100ms")
		assert.Equal(t, 10, count)
	})
}

// TestDatabase_ConnectionPool tests connection pool behavior
func TestDatabase_ConnectionPool(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	// Set pool limits
	infra.DB.SetMaxOpenConns(5)
	infra.DB.SetMaxIdleConns(2)
	infra.DB.SetConnMaxLifetime(1 * time.Minute)

	// Verify pool stats
	stats := infra.DB.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 5)

	// Execute concurrent queries
	const numQueries = 20
	done := make(chan error, numQueries)

	for i := 0; i < numQueries; i++ {
		go func() {
			var result int
			err := infra.DB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			done <- err
		}()
	}

	// Wait for all queries
	for i := 0; i < numQueries; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Check pool didn't exhaust
	stats = infra.DB.Stats()
	assert.LessOrEqual(t, stats.OpenConnections, 5,
		"should not exceed max open connections")
}

// TestDatabase_JSONBOperations tests JSONB field operations
func TestDatabase_JSONBOperations(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// Insert alert with JSONB labels
	alert := NewTestAlert("JSONBTest").
		WithLabel("environment", "production").
		WithLabel("team", "platform").
		WithAnnotation("runbook", "https://example.com")

	err = helper.SeedTestData(ctx, []*Alert{alert})
	require.NoError(t, err)

	t.Run("query by JSONB field", func(t *testing.T) {
		// Query using JSONB operator
		var count int
		err := infra.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM alert_history
			WHERE labels->>'environment' = $1
		`, "production").Scan(&count)

		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("query by nested JSONB", func(t *testing.T) {
		var count int
		err := infra.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM alert_history
			WHERE labels @> $1::jsonb
		`, `{"team":"platform"}`).Scan(&count)

		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}
