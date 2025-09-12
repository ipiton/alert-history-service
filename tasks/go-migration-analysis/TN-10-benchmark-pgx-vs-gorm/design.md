# TN-10: Design - Benchmark pgx vs GORM

## Архитектура бенчмаркинга

### Тестовое приложение
```
benchmark-db/
├── cmd/
│   ├── pgx-server/
│   │   └── main.go          # pgx implementation
│   └── gorm-server/
│       └── main.go          # GORM implementation
├── internal/
│   ├── database/
│   │   ├── pgx/
│   │   │   ├── connection.go
│   │   │   ├── queries.go
│   │   │   └── migrations.go
│   │   └── gorm/
│   │       ├── models.go
│   │       ├── repository.go
│   │       └── migrations.go
│   └── models/
│       └── alert.go         # Common data models
└── benchmark/
    ├── scripts/
    │   ├── setup_database.sh
    │   ├── run_db_benchmarks.sh
    │   └── analyze_db_results.py
    └── results/
        └── db_benchmarks/
```

### Database Schema
```sql
-- Alerts table for benchmarking
CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);
CREATE INDEX idx_alerts_labels ON alerts USING GIN(labels);
```

### Тестовые операции

#### CRUD Operations
```go
// pgx implementation
func (r *pgxRepository) CreateAlert(ctx context.Context, alert *Alert) error {
    return r.pool.QueryRow(ctx, `
        INSERT INTO alerts (title, description, severity, status, labels, annotations)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at`,
        alert.Title, alert.Description, alert.Severity, alert.Status,
        alert.Labels, alert.Annotations).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)
}

// GORM implementation
func (r *gormRepository) CreateAlert(alert *Alert) error {
    return r.db.Create(alert).Error
}
```

#### Complex Queries
```go
// pgx with complex query
func (r *pgxRepository) FindAlertsWithFilters(ctx context.Context, filters map[string]interface{}) ([]*Alert, error) {
    query := `SELECT id, title, description, severity, status, labels, annotations, created_at, updated_at
              FROM alerts WHERE 1=1`
    args := []interface{}{}
    argCount := 0

    for key, value := range filters {
        argCount++
        query += fmt.Sprintf(" AND %s = $%d", key, argCount)
        args = append(args, value)
    }

    rows, err := r.pool.Query(ctx, query, args...)
    // Process results
}

// GORM with complex query
func (r *gormRepository) FindAlertsWithFilters(filters map[string]interface{}) ([]*Alert, error) {
    query := r.db
    for key, value := range filters {
        query = query.Where(fmt.Sprintf("%s = ?", key), value)
    }
    return query.Find(&alerts).Error
}
```

## Бенчмаркинг инструменты

### Database Load Testing
```bash
# pgbench for PostgreSQL load testing
pgbench -c 10 -j 2 -T 60 -f custom_queries.sql testdb

# Custom Go benchmarking
go test -bench=. -benchmem ./benchmark/db/

# Apache Bench for HTTP endpoints
ab -n 10000 -c 100 http://localhost:8080/api/alerts
```

### Profiling Tools
```bash
# Database query profiling
EXPLAIN ANALYZE SELECT * FROM alerts WHERE severity = 'critical';

# Go profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# PostgreSQL monitoring
SELECT * FROM pg_stat_activity;
SELECT * FROM pg_stat_statements;
```

### Metrics Collection
```sql
-- Query performance monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Connection pool monitoring
SELECT count(*) as active_connections FROM pg_stat_activity WHERE state = 'active';
```

## Тестовая методология

### 1. Setup Phase
```bash
# Create test database
createdb alert_benchmark

# Run migrations
./migrate up

# Generate test data
./seed --count 100000
```

### 2. Warm-up Phase
```bash
# Warm up database connections
hey -n 1000 -c 10 http://localhost:8080/api/alerts

# Warm up query cache
pgbench -c 1 -j 1 -T 10 testdb
```

### 3. Benchmarking Phase
```bash
# Single operations benchmark
go test -bench=BenchmarkSingleInsert ./benchmark/db/
go test -bench=BenchmarkSingleSelect ./benchmark/db/
go test -bench=BenchmarkSingleUpdate ./benchmark/db/

# Batch operations benchmark
go test -bench=BenchmarkBatchInsert ./benchmark/db/
go test -bench=BenchmarkComplexQuery ./benchmark/db/

# Concurrent operations
go test -bench=BenchmarkConcurrentOperations ./benchmark/db/
```

### 4. Load Testing Phase
```bash
# Simulate production load
pgbench -c 50 -j 4 -T 300 -S testdb

# Application load testing
hey -n 50000 -c 200 http://localhost:8080/api/alerts
```

## Метрики сбора

### Database Metrics
```json
{
  "query_execution_time_ms": 12.5,
  "connection_acquisition_time_ms": 2.1,
  "rows_affected": 1,
  "query_type": "SELECT",
  "table_name": "alerts",
  "index_used": true,
  "cache_hit_ratio": 0.85
}
```

### Application Metrics
```json
{
  "memory_usage_mb": 45.2,
  "cpu_usage_percent": 23.4,
  "goroutines_count": 156,
  "database_connections_active": 12,
  "database_connections_idle": 8,
  "query_cache_hit_ratio": 0.92
}
```

### PostgreSQL Metrics
```sql
-- Connection statistics
SELECT
    count(*) as total_connections,
    count(*) filter (where state = 'active') as active_connections,
    count(*) filter (where state = 'idle') as idle_connections
FROM pg_stat_activity;

-- Query performance
SELECT
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;
```

## Анализ результатов

### Performance Comparison
```python
# Calculate performance ratios
pgx_insert_time = measure_pgx_insert()
gorm_insert_time = measure_gorm_insert()

performance_ratio = pgx_insert_time / gorm_insert_time
print(f"pgx is {performance_ratio:.2f}x {'faster' if performance_ratio < 1 else 'slower'} for inserts")
```

### Memory Usage Analysis
```python
# Memory usage comparison
pgx_memory = measure_pgx_memory_usage()
gorm_memory = measure_gorm_memory_usage()

memory_ratio = pgx_memory / gorm_memory
print(f"pgx uses {memory_ratio:.2f}x {'more' if memory_ratio > 1 else 'less'} memory")
```

### Developer Productivity
```python
# Code complexity metrics
pgx_loc = count_lines_of_code("./internal/database/pgx/")
gorm_loc = count_lines_of_code("./internal/database/gorm/")

productivity_ratio = pgx_loc / gorm_loc
print(f"GORM requires {productivity_ratio:.2f}x {'more' if productivity_ratio > 1 else 'less'} code")
```

## Рекомендации

### Use Case Matrix
```python
scenarios = {
    'high_performance': 'pgx',
    'rapid_development': 'gorm',
    'complex_queries': 'pgx',
    'simple_crud': 'gorm',
    'memory_constrained': 'pgx',
    'type_safety': 'pgx',
    'migrations': 'gorm'
}
```

### Decision Framework
```python
def recommend_driver(requirements):
    score_pgx = 0
    score_gorm = 0

    if requirements.get('performance_critical'):
        score_pgx += 3
    if requirements.get('rapid_development'):
        score_gorm += 2
    if requirements.get('complex_queries'):
        score_pgx += 2
    if requirements.get('simple_crud'):
        score_gorm += 1

    return 'pgx' if score_pgx > score_gorm else 'gorm'
```

### Migration Strategy
```python
# Gradual migration approach
migration_steps = [
    'Add pgx/GORM alongside existing code',
    'Implement new features with new driver',
    'Migrate read operations first',
    'Migrate write operations',
    'Remove old database code',
    'Optimize and refactor'
]
```
