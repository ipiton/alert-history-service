package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// InfraMetrics contains all infrastructure-level metrics for Alert History Service.
//
// Infrastructure metrics track low-level system resources:
//   - Database connection pools (connections, queries, latency)
//   - Cache operations (hits, misses, evictions)
//   - Repository queries (duration, errors, results)
//
// All metrics follow the taxonomy:
// alert_history_infra_<subsystem>_<metric_name>_<unit>
//
// Example:
//
//	im := NewInfraMetrics("alert_history")
//	im.DB.ConnectionsActive.Set(42)
//	im.Cache.HitsTotal.WithLabelValues("redis").Inc()
type InfraMetrics struct {
	namespace string

	// DB subsystem - database connection pool metrics
	DB *DatabaseMetrics

	// Cache subsystem - cache (Redis) metrics
	Cache *CacheMetrics

	// Repository subsystem - data repository metrics
	Repository *RepositoryMetrics
}

// NewInfraMetrics creates a new InfraMetrics instance with all subsystems initialized.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *InfraMetrics: Initialized infrastructure metrics manager
func NewInfraMetrics(namespace string) *InfraMetrics {
	return &InfraMetrics{
		namespace:  namespace,
		DB:         NewDatabaseMetrics(namespace),
		Cache:      NewCacheMetrics(namespace),
		Repository: NewRepositoryMetrics(namespace),
	}
}

// DatabaseMetrics contains metrics for database connection pool.
//
// Tracks database health, connection usage, query performance, and errors.
// These metrics are populated by the PrometheusExporter in database/postgres/prometheus.go.
//
// Example:
//
//	db.ConnectionsActive.Set(float64(activeConns))
//	db.QueryDurationSeconds.WithLabelValues("SELECT").Observe(0.05)
type DatabaseMetrics struct {
	// Connection pool metrics
	ConnectionsActive  prometheus.Gauge   // Number of active database connections
	ConnectionsIdle    prometheus.Gauge   // Number of idle connections in pool
	ConnectionsTotal   prometheus.Counter // Total number of connections created (cumulative)

	// Performance metrics
	ConnectionWaitDurationSeconds prometheus.Histogram    // Time spent waiting for a connection
	QueryDurationSeconds          *prometheus.HistogramVec // Duration of database queries

	// Operation metrics
	QueriesTotal *prometheus.CounterVec // Total number of queries executed

	// Error metrics
	ErrorsTotal *prometheus.CounterVec // Total number of database errors
}

// NewDatabaseMetrics creates database connection pool metrics.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *DatabaseMetrics: Initialized database metrics
func NewDatabaseMetrics(namespace string) *DatabaseMetrics {
	return &DatabaseMetrics{
		ConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "infra_db",
			Name:      "connections_active",
			Help:      "Number of active database connections currently in use",
		}),

		ConnectionsIdle: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "infra_db",
			Name:      "connections_idle",
			Help:      "Number of idle database connections in the pool",
		}),

		ConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "infra_db",
			Name:      "connections_total",
			Help:      "Total number of database connections created (cumulative)",
		}),

		ConnectionWaitDurationSeconds: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "infra_db",
			Name:      "connection_wait_duration_seconds",
			Help:      "Time spent waiting for a database connection from the pool",
			// Buckets optimized for connection wait: 1ms to 1s
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		}),

		QueryDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "infra_db",
				Name:      "query_duration_seconds",
				Help:      "Duration of database queries in seconds",
				// Buckets optimized for SQL queries: 1ms to 1s
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"operation"}, // operation: SELECT|INSERT|UPDATE|DELETE
		),

		QueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_db",
				Name:      "queries_total",
				Help:      "Total number of database queries executed",
			},
			[]string{"operation", "status"}, // operation: SELECT|INSERT|UPDATE|DELETE, status: success|error
		),

		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_db",
				Name:      "errors_total",
				Help:      "Total number of database errors encountered",
			},
			[]string{"error_type"}, // error_type: connection|query|timeout|constraint
		),
	}
}

// CacheMetrics contains metrics for cache operations (Redis).
//
// Tracks cache effectiveness (hit/miss ratio) and errors.
//
// Example:
//
//	cache.HitsTotal.WithLabelValues("redis").Inc()
//	cache.MissesTotal.WithLabelValues("redis").Inc()
type CacheMetrics struct {
	HitsTotal      *prometheus.CounterVec // Total number of cache hits
	MissesTotal    *prometheus.CounterVec // Total number of cache misses
	ErrorsTotal    *prometheus.CounterVec // Total number of cache errors
	EvictionsTotal prometheus.Counter     // Total number of cache evictions
	SizeBytes      prometheus.Gauge       // Current size of cache in bytes
}

// NewCacheMetrics creates cache operation metrics.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *CacheMetrics: Initialized cache metrics
func NewCacheMetrics(namespace string) *CacheMetrics {
	return &CacheMetrics{
		HitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_cache",
				Name:      "hits_total",
				Help:      "Total number of cache hits (successful cache lookups)",
			},
			[]string{"cache_type"}, // cache_type: redis|memory|llm
		),

		MissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_cache",
				Name:      "misses_total",
				Help:      "Total number of cache misses (cache lookups that failed)",
			},
			[]string{"cache_type"}, // cache_type: redis|memory|llm
		),

		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_cache",
				Name:      "errors_total",
				Help:      "Total number of cache errors encountered",
			},
			[]string{"cache_type", "error_type"}, // error_type: connection|timeout|serialization
		),

		EvictionsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_cache",
				Name:      "evictions_total",
				Help:      "Total number of cache evictions (items removed due to size/TTL)",
			},
		),

		SizeBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "infra_cache",
				Name:      "size_bytes",
				Help:      "Current size of cache in bytes",
			},
		),
	}
}

// RepositoryMetrics contains metrics for data repository operations.
//
// Tracks repository query performance, errors, and result sizes.
// This replaces the legacy metrics that were missing subsystem naming.
//
// Legacy metrics (deprecated):
//   - alert_history_query_duration_seconds       → alert_history_infra_repository_query_duration_seconds
//   - alert_history_query_errors_total           → alert_history_infra_repository_query_errors_total
//   - alert_history_query_results_total          → alert_history_infra_repository_query_results_total
//
// Example:
//
//	repo.QueryDurationSeconds.WithLabelValues("GetTopAlerts", "success").Observe(0.05)
//	repo.QueryErrorsTotal.WithLabelValues("GetFlappingAlerts", "timeout").Inc()
type RepositoryMetrics struct {
	QueryDurationSeconds *prometheus.HistogramVec // Duration of repository queries
	QueryErrorsTotal     *prometheus.CounterVec   // Total number of repository errors
	QueryResultsTotal    *prometheus.HistogramVec // Number of results returned by queries
}

// NewRepositoryMetrics creates repository operation metrics.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *RepositoryMetrics: Initialized repository metrics
func NewRepositoryMetrics(namespace string) *RepositoryMetrics {
	return &RepositoryMetrics{
		QueryDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "infra_repository",
				Name:      "query_duration_seconds",
				Help:      "Duration of repository query operations in seconds",
				// Buckets optimized for repository queries: 1ms to 5s
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"operation", "status"}, // operation: GetTopAlerts|GetFlappingAlerts|etc, status: success|error
		),

		QueryErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "infra_repository",
				Name:      "query_errors_total",
				Help:      "Total number of repository query errors encountered",
			},
			[]string{"operation", "error_type"}, // operation: GetTopAlerts|etc, error_type: timeout|not_found|internal
		),

		QueryResultsTotal: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "infra_repository",
				Name:      "query_results_total",
				Help:      "Number of results returned by repository queries",
				// Buckets optimized for result counts: 0 to 1000
				Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"operation"}, // operation: GetTopAlerts|GetFlappingAlerts|etc
		),
	}
}
