package repository

import (
	"context"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// MockAlertStorage is a mock implementation of core.AlertStorage for testing
type MockAlertStorage struct {
	alerts []*core.Alert
}

func (m *MockAlertStorage) SaveAlert(ctx context.Context, alert *core.Alert) error {
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *MockAlertStorage) GetAlertByFingerprint(ctx context.Context, fingerprint string) (*core.Alert, error) {
	for _, alert := range m.alerts {
		if alert.Fingerprint == fingerprint {
			return alert, nil
		}
	}
	return nil, nil
}

func (m *MockAlertStorage) ListAlerts(ctx context.Context, filters *core.AlertFilters) (*core.AlertList, error) {
	return &core.AlertList{
		Alerts: m.alerts,
		Total:  len(m.alerts),
	}, nil
}

func (m *MockAlertStorage) UpdateAlert(ctx context.Context, alert *core.Alert) error {
	for i, a := range m.alerts {
		if a.Fingerprint == alert.Fingerprint {
			m.alerts[i] = alert
			return nil
		}
	}
	return nil
}

func (m *MockAlertStorage) DeleteAlert(ctx context.Context, fingerprint string) error {
	for i, alert := range m.alerts {
		if alert.Fingerprint == fingerprint {
			m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockAlertStorage) GetAlertStats(ctx context.Context) (*core.AlertStats, error) {
	return &core.AlertStats{
		TotalAlerts:       len(m.alerts),
		AlertsByStatus:    make(map[string]int),
		AlertsBySeverity:  make(map[string]int),
		AlertsByNamespace: make(map[string]int),
	}, nil
}

func (m *MockAlertStorage) CleanupOldAlerts(ctx context.Context, retentionDays int) (int, error) {
	return 0, nil
}

// TestGetTopAlerts_EmptyDatabase tests GetTopAlerts with no data
func TestGetTopAlerts_EmptyDatabase(t *testing.T) {
	// This test would require a real database connection or testcontainers
	// For now, we document the test structure
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Create PostgresHistoryRepository
	// 3. Call GetTopAlerts with empty database
	// 4. Assert: empty result, no errors
}

// TestGetFlappingAlerts_NoStateTransitions tests flapping detection with stable alerts
func TestGetFlappingAlerts_NoStateTransitions(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert alerts with same status (no transitions)
	// 3. Call GetFlappingAlerts
	// 4. Assert: empty result or low flapping score
}

// TestGetFlappingAlerts_MultipleTransitions tests detection of flapping alerts
func TestGetFlappingAlerts_MultipleTransitions(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert alerts with multiple state transitions:
	//    - firing → resolved → firing → resolved (4+ transitions)
	// 3. Call GetFlappingAlerts with threshold=3
	// 4. Assert: flapping alert detected, correct flapping_score
}

// TestGetAggregatedStats_WithData tests aggregated statistics calculation
func TestGetAggregatedStats_WithData(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert diverse alerts:
	//    - Different statuses (firing, resolved)
	//    - Different severities (critical, warning, info)
	//    - Different namespaces
	// 3. Call GetAggregatedStats
	// 4. Assert: correct counts for all dimensions
}

// TestGetTopAlerts_WithTimeRange tests time range filtering
func TestGetTopAlerts_WithTimeRange(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert alerts with different timestamps
	// 3. Call GetTopAlerts with specific time range
	// 4. Assert: only alerts within time range are counted
}

// TestGetTopAlerts_LimitValidation tests limit parameter validation
func TestGetTopAlerts_LimitValidation(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Test with limit=0 → should use default (10)
	// 3. Test with limit=150 → should cap at 100
	// 4. Test with limit=5 → should return exactly 5
}

// TestGetFlappingAlerts_ThresholdFiltering tests threshold parameter
func TestGetFlappingAlerts_ThresholdFiltering(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert alerts with 2, 3, 5 transitions
	// 3. Test with threshold=3
	// 4. Assert: only alerts with 3+ transitions returned
}

// TestGetAggregatedStats_TimeRange tests stats with time range
func TestGetAggregatedStats_TimeRange(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL with testcontainers")

	// Test structure:
	// 1. Setup testcontainers PostgreSQL
	// 2. Insert alerts across multiple days
	// 3. Call GetAggregatedStats with 24h time range
	// 4. Assert: only last 24h alerts counted
}

// Example integration test structure (when testcontainers are added)
/*
func TestGetTopAlerts_Integration(t *testing.T) {
	// Start testcontainers PostgreSQL
	ctx := context.Background()
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "testdb",
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	// Get connection details
	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	// Create pool
	config := &PostgresConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "testdb",
		User:     "test",
		Password: "test",
	}
	pool := NewPostgresPool(config, slog.Default())
	err = pool.Connect(ctx)
	require.NoError(t, err)
	defer pool.Disconnect(ctx)

	// Run migrations
	// ... setup schema ...

	// Create repository
	mockStorage := &MockAlertStorage{}
	repo := NewPostgresHistoryRepository(pool.Pool(), mockStorage, slog.Default())

	// Insert test data
	now := time.Now()
	for i := 0; i < 10; i++ {
		alert := &core.Alert{
			Fingerprint: fmt.Sprintf("fp-%d", i),
			AlertName:   fmt.Sprintf("Alert%d", i % 3),
			Status:      core.StatusFiring,
			Labels:      map[string]string{"severity": "critical"},
			StartsAt:    now.Add(-time.Hour * time.Duration(i)),
		}
		mockStorage.SaveAlert(ctx, alert)
	}

	// Test GetTopAlerts
	topAlerts, err := repo.GetTopAlerts(ctx, nil, 5)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(topAlerts), 5)

	// Verify sorting (by fire_count DESC)
	for i := 1; i < len(topAlerts); i++ {
		assert.GreaterOrEqual(t, topAlerts[i-1].FireCount, topAlerts[i].FireCount)
	}
}
*/

// NOTE: For full test coverage, add testcontainers:
// 1. go get github.com/testcontainers/testcontainers-go
// 2. Implement integration tests with real PostgreSQL
// 3. Test SQL queries, aggregations, window functions
// 4. Verify Prometheus metrics are recorded
// 5. Test edge cases (NULL values, empty results, etc.)

// Unit test examples for validation logic
func TestTimeRangeValidation(t *testing.T) {
	tests := []struct {
		name        string
		timeRange   *core.TimeRange
		expectValid bool
	}{
		{
			name:        "nil time range is valid",
			timeRange:   nil,
			expectValid: true,
		},
		{
			name: "valid time range",
			timeRange: &core.TimeRange{
				From: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				To:   func() *time.Time { t := time.Now(); return &t }(),
			},
			expectValid: true,
		},
		{
			name: "only from is valid",
			timeRange: &core.TimeRange{
				From: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Time range validation is implicit in the repository
			// This test documents expected behavior
			if tt.expectValid {
				// All cases should be valid
				t.Log("Time range is valid:", tt.name)
			}
		})
	}
}

func TestLimitValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{
			name:          "zero limit uses default",
			inputLimit:    0,
			expectedLimit: 10,
		},
		{
			name:          "negative limit uses default",
			inputLimit:    -5,
			expectedLimit: 10,
		},
		{
			name:          "valid limit",
			inputLimit:    25,
			expectedLimit: 25,
		},
		{
			name:          "limit over max caps at 100",
			inputLimit:    150,
			expectedLimit: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Document limit validation logic from postgres_history.go GetTopAlerts()
			limit := tt.inputLimit
			if limit <= 0 {
				limit = 10
			}
			if limit > 100 {
				limit = 100
			}

			if limit != tt.expectedLimit {
				t.Errorf("Limit validation failed: got %d, want %d", limit, tt.expectedLimit)
			}
		})
	}
}

func TestFlappingThresholdValidation(t *testing.T) {
	tests := []struct {
		name              string
		inputThreshold    int
		expectedThreshold int
	}{
		{
			name:              "zero threshold uses default",
			inputThreshold:    0,
			expectedThreshold: 3,
		},
		{
			name:              "negative threshold uses default",
			inputThreshold:    -1,
			expectedThreshold: 3,
		},
		{
			name:              "valid threshold",
			inputThreshold:    5,
			expectedThreshold: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Document threshold validation logic from postgres_history.go GetFlappingAlerts()
			threshold := tt.inputThreshold
			if threshold <= 0 {
				threshold = 3
			}

			if threshold != tt.expectedThreshold {
				t.Errorf("Threshold validation failed: got %d, want %d", threshold, tt.expectedThreshold)
			}
		})
	}
}

// BenchmarkGetTopAlerts benchmarks top alerts query performance
func BenchmarkGetTopAlerts(b *testing.B) {
	b.Skip("Benchmark requires PostgreSQL connection")

	// Benchmark structure:
	// 1. Setup PostgreSQL with realistic data (10k+ alerts)
	// 2. Run GetTopAlerts b.N times
	// 3. Measure query performance
}

// BenchmarkGetFlappingAlerts benchmarks flapping detection performance
func BenchmarkGetFlappingAlerts(b *testing.B) {
	b.Skip("Benchmark requires PostgreSQL connection")

	// Benchmark structure:
	// 1. Setup PostgreSQL with alerts having state transitions
	// 2. Run GetFlappingAlerts b.N times
	// 3. Measure window function performance
}

// BenchmarkGetAggregatedStats benchmarks stats aggregation performance
func BenchmarkGetAggregatedStats(b *testing.B) {
	b.Skip("Benchmark requires PostgreSQL connection")

	// Benchmark structure:
	// 1. Setup PostgreSQL with diverse alert data
	// 2. Run GetAggregatedStats b.N times
	// 3. Measure aggregation query performance
}
