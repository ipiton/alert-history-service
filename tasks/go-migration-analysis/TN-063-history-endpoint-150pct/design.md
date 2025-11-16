# TN-063: GET /history - Design Document

**Project**: Alert History Endpoint with Advanced Filters  
**Version**: 1.0  
**Date**: 2025-11-16  
**Status**: Draft  
**Target Quality**: 150% Enterprise Grade (A++)  

---

## TABLE OF CONTENTS

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Data Flow](#3-data-flow)
4. [Enhanced Filter System](#4-enhanced-filter-system)
5. [Caching Strategy](#5-caching-strategy)
6. [Middleware Stack](#6-middleware-stack)
7. [Security Design](#7-security-design)
8. [Performance Optimization](#8-performance-optimization)
9. [Error Handling](#9-error-handling)
10. [Monitoring & Observability](#10-monitoring--observability)

---

## 1. ARCHITECTURE OVERVIEW

### 1.1 High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                          Client Layer                             │
│   (SRE Tools, Dashboards, Automation Scripts, Analytics)        │
└────────────────────────────────────┬─────────────────────────────┘
                                     │ HTTPS/JSON
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                       API Gateway Layer                           │
│   • Load Balancer (2-50 instances)                              │
│   • TLS Termination                                              │
│   • Connection Pooling                                           │
└────────────────────────────────────┬─────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Application Layer (Go)                         │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              Middleware Stack (10 layers)                   │ │
│  │  Recovery → RequestID → Logging → Metrics → RateLimit →    │ │
│  │  Auth → Authz → CORS → Compression → Timeout               │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                API Handlers Layer                           │ │
│  │  • HistoryHandler (main endpoint)                          │ │
│  │  • TopAlertsHandler                                        │ │
│  │  • FlappingAlertsHandler                                   │ │
│  │  • RecentAlertsHandler                                     │ │
│  │  • StatsHandler                                            │ │
│  │  • SingleAlertHandler                                      │ │
│  │  • AdvancedSearchHandler                                   │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              Request Validation Layer                       │ │
│  │  • Query Parameter Parsing                                 │ │
│  │  • Filter Validation (15+ types)                          │ │
│  │  • Pagination Validation                                   │ │
│  │  • Sorting Validation                                      │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                 Caching Layer (2-Tier)                     │ │
│  │  L1: Ristretto (in-memory, 10K entries, 100MB, 5min TTL) │ │
│  │  L2: Redis (distributed, 1M entries, 1h TTL)              │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │               Repository Layer                              │ │
│  │  • PostgresHistoryRepository                               │ │
│  │  • QueryBuilder (SQL generation)                           │ │
│  │  • ResultMapper (DB → Domain)                              │ │
│  └────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────┬─────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Data Layer                                   │
│  ┌────────────────┐        ┌────────────────┐                   │
│  │  PostgreSQL    │        │  Redis         │                   │
│  │  (Primary DB)  │        │  (L2 Cache)    │                   │
│  │                │        │                │                   │
│  │  • alerts      │        │  • cache:*     │                   │
│  │  • 8 indexes   │        │  • ttl: 1h     │                   │
│  │  • pool: 50    │        │  • pool: 50    │                   │
│  └────────────────┘        └────────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
                                     ▲
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                  Observability Layer                              │
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐   │
│  │  Prometheus    │  │  Grafana       │  │  Logs (JSON)    │   │
│  │  (Metrics)     │  │  (Dashboards)  │  │  (slog)         │   │
│  │  • 18+ metrics │  │  • 8+ panels   │  │  • Structured   │   │
│  │  • /metrics    │  │  • 6+ alerts   │  │  • Request ID   │   │
│  └────────────────┘  └────────────────┘  └─────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 Design Principles

**1. Separation of Concerns**:
- Handlers: HTTP request/response handling
- Validation: Input validation & sanitization
- Caching: Result caching & invalidation
- Repository: Data access & query building
- Database: Data storage & indexing

**2. Dependency Injection**:
- All dependencies injected via constructors
- Easy to test (mock dependencies)
- Clear dependency graph

**3. Interface-Driven Design**:
- Repository interfaces defined in core package
- Implementation in infrastructure package
- Swappable implementations

**4. Fail-Fast Validation**:
- Validate inputs early (before expensive operations)
- Return descriptive errors
- Prevent invalid data from reaching database

**5. Performance-First**:
- Caching at multiple levels
- Database query optimization
- Connection pooling
- Efficient data structures

**6. Security-By-Design**:
- Authentication required (API key)
- Authorization checks (RBAC)
- Input validation & sanitization
- Rate limiting
- Audit logging

---

## 2. COMPONENT DESIGN

### 2.1 Enhanced Filter System

#### Filter Registry Pattern
```go
// pkg/history/filters/registry.go
package filters

import (
	"fmt"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// FilterType represents a type of filter
type FilterType string

const (
	FilterTypeStatus       FilterType = "status"
	FilterTypeSeverity     FilterType = "severity"
	FilterTypeNamespace    FilterType = "namespace"
	FilterTypeFingerprint  FilterType = "fingerprint"
	FilterTypeAlertName    FilterType = "alert_name"
	FilterTypeLabels       FilterType = "labels"
	FilterTypeTimeRange    FilterType = "time_range"
	FilterTypeDuration     FilterType = "duration"
	FilterTypeSearch       FilterType = "search"
	// ... more filter types
)

// Filter interface defines common operations for all filters
type Filter interface {
	Type() FilterType
	Validate() error
	ApplyToQuery(qb *QueryBuilder) error
	CacheKey() string
}

// FilterRegistry manages all available filters
type FilterRegistry struct {
	filters map[FilterType]FilterFactory
}

// FilterFactory creates filter instances
type FilterFactory func(params map[string]interface{}) (Filter, error)

// NewFilterRegistry creates a new filter registry
func NewFilterRegistry() *FilterRegistry {
	registry := &FilterRegistry{
		filters: make(map[FilterType]FilterFactory),
	}
	
	// Register all filter types
	registry.Register(FilterTypeStatus, NewStatusFilter)
	registry.Register(FilterTypeSeverity, NewSeverityFilter)
	registry.Register(FilterTypeNamespace, NewNamespaceFilter)
	registry.Register(FilterTypeFingerprint, NewFingerprintFilter)
	registry.Register(FilterTypeAlertName, NewAlertNameFilter)
	registry.Register(FilterTypeLabels, NewLabelsFilter)
	registry.Register(FilterTypeTimeRange, NewTimeRangeFilter)
	registry.Register(FilterTypeDuration, NewDurationFilter)
	registry.Register(FilterTypeSearch, NewSearchFilter)
	// ... register more filters
	
	return registry
}

// Register adds a filter factory to the registry
func (r *FilterRegistry) Register(typ FilterType, factory FilterFactory) {
	r.filters[typ] = factory
}

// Create creates a filter instance from parameters
func (r *FilterRegistry) Create(typ FilterType, params map[string]interface{}) (Filter, error) {
	factory, ok := r.filters[typ]
	if !ok {
		return nil, fmt.Errorf("unknown filter type: %s", typ)
	}
	return factory(params)
}

// BuildFilters builds all filters from query parameters
func (r *FilterRegistry) BuildFilters(queryParams map[string][]string) ([]Filter, error) {
	var filters []Filter
	
	// Parse status filter
	if values, ok := queryParams["status"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeStatus, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	
	// Parse severity filter
	if values, ok := queryParams["severity"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeSeverity, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	
	// ... parse more filters
	
	return filters, nil
}
```

#### Specific Filter Implementations
```go
// pkg/history/filters/status_filter.go
package filters

import (
	"fmt"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// StatusFilter filters alerts by status (firing, resolved)
type StatusFilter struct {
	values []core.AlertStatus
}

// NewStatusFilter creates a new status filter
func NewStatusFilter(params map[string]interface{}) (Filter, error) {
	values, ok := params["values"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid status filter params")
	}
	
	filter := &StatusFilter{
		values: make([]core.AlertStatus, 0, len(values)),
	}
	
	for _, v := range values {
		status := core.AlertStatus(v)
		if status != core.StatusFiring && status != core.StatusResolved {
			return nil, fmt.Errorf("invalid status: %s", v)
		}
		filter.values = append(filter.values, status)
	}
	
	return filter, nil
}

func (f *StatusFilter) Type() FilterType {
	return FilterTypeStatus
}

func (f *StatusFilter) Validate() error {
	if len(f.values) == 0 {
		return fmt.Errorf("status filter requires at least one value")
	}
	return nil
}

func (f *StatusFilter) ApplyToQuery(qb *QueryBuilder) error {
	if len(f.values) == 1 {
		qb.AddWhere("status = ?", f.values[0])
	} else {
		placeholders := make([]string, len(f.values))
		args := make([]interface{}, len(f.values))
		for i, v := range f.values {
			placeholders[i] = "?"
			args[i] = v
		}
		qb.AddWhere(fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")), args...)
	}
	return nil
}

func (f *StatusFilter) CacheKey() string {
	values := make([]string, len(f.values))
	for i, v := range f.values {
		values[i] = string(v)
	}
	sort.Strings(values)
	return fmt.Sprintf("status:%s", strings.Join(values, ","))
}
```

```go
// pkg/history/filters/labels_filter.go
package filters

import (
	"fmt"
	"strings"
)

// LabelsFilter filters alerts by label key-value pairs
// Supports operators: =, !=, =~, !~
type LabelsFilter struct {
	exact      map[string]string // = operator
	notEqual   map[string]string // != operator
	regex      map[string]string // =~ operator
	notRegex   map[string]string // !~ operator
	exists     []string          // label key must exist
	notExists  []string          // label key must not exist
}

// NewLabelsFilter creates a new labels filter
func NewLabelsFilter(params map[string]interface{}) (Filter, error) {
	filter := &LabelsFilter{
		exact:     make(map[string]string),
		notEqual:  make(map[string]string),
		regex:     make(map[string]string),
		notRegex:  make(map[string]string),
		exists:    []string{},
		notExists: []string{},
	}
	
	// Parse exact match labels
	if exact, ok := params["exact"].(map[string]string); ok {
		filter.exact = exact
	}
	
	// Parse not equal labels
	if notEqual, ok := params["not_equal"].(map[string]string); ok {
		filter.notEqual = notEqual
	}
	
	// Parse regex labels
	if regex, ok := params["regex"].(map[string]string); ok {
		// Validate regex patterns
		for key, pattern := range regex {
			if _, err := regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf("invalid regex pattern for label %s: %w", key, err)
			}
		}
		filter.regex = regex
	}
	
	// Parse not regex labels
	if notRegex, ok := params["not_regex"].(map[string]string); ok {
		// Validate regex patterns
		for key, pattern := range notRegex {
			if _, err := regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf("invalid regex pattern for label %s: %w", key, err)
			}
		}
		filter.notRegex = notRegex
	}
	
	// Parse exists labels
	if exists, ok := params["exists"].([]string); ok {
		filter.exists = exists
	}
	
	// Parse not exists labels
	if notExists, ok := params["not_exists"].([]string); ok {
		filter.notExists = notExists
	}
	
	return filter, nil
}

func (f *LabelsFilter) Type() FilterType {
	return FilterTypeLabels
}

func (f *LabelsFilter) Validate() error {
	totalLabels := len(f.exact) + len(f.notEqual) + len(f.regex) + len(f.notRegex)
	if totalLabels > 20 {
		return fmt.Errorf("too many label filters: max 20, got %d", totalLabels)
	}
	
	for key, value := range f.exact {
		if len(key) > 255 {
			return fmt.Errorf("label key too long: max 255 chars")
		}
		if len(value) > 255 {
			return fmt.Errorf("label value too long: max 255 chars")
		}
	}
	
	return nil
}

func (f *LabelsFilter) ApplyToQuery(qb *QueryBuilder) error {
	// Apply exact match filters
	for key, value := range f.exact {
		qb.AddWhere("labels @> jsonb_build_object(?, ?)", key, value)
	}
	
	// Apply not equal filters
	for key, value := range f.notEqual {
		qb.AddWhere("NOT (labels @> jsonb_build_object(?, ?))", key, value)
	}
	
	// Apply regex filters
	for key, pattern := range f.regex {
		qb.AddWhere("labels->>? ~ ?", key, pattern)
	}
	
	// Apply not regex filters
	for key, pattern := range f.notRegex {
		qb.AddWhere("NOT (labels->>? ~ ?)", key, pattern)
	}
	
	// Apply exists filters
	for _, key := range f.exists {
		qb.AddWhere("labels ? ?", key)
	}
	
	// Apply not exists filters
	for _, key := range f.notExists {
		qb.AddWhere("NOT (labels ? ?)", key)
	}
	
	return nil
}

func (f *LabelsFilter) CacheKey() string {
	var parts []string
	
	// Sort keys for consistent cache keys
	if len(f.exact) > 0 {
		keys := make([]string, 0, len(f.exact))
		for k := range f.exact {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("l=%s:%s", k, f.exact[k]))
		}
	}
	
	// ... similar for other operators
	
	return strings.Join(parts, "&")
}
```

### 2.2 Query Builder Design

```go
// pkg/history/query/builder.go
package query

import (
	"fmt"
	"strings"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/filters"
)

// QueryBuilder builds optimized SQL queries
type QueryBuilder struct {
	baseQuery    string
	whereClauses []string
	args         []interface{}
	argCounter   int
	orderBy      []string
	limit        int
	offset       int
	
	// Performance optimization flags
	useGINIndex   bool  // Use GIN index for JSONB queries
	usePartialIdx bool  // Use partial index for common queries
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		baseQuery:    "SELECT * FROM alerts",
		whereClauses: []string{"1=1"},
		args:         []interface{}{},
		argCounter:   0,
		orderBy:      []string{},
	}
}

// AddWhere adds a WHERE clause with arguments
func (qb *QueryBuilder) AddWhere(clause string, args ...interface{}) {
	// Replace ? placeholders with $N
	numArgs := strings.Count(clause, "?")
	for i := 0; i < numArgs; i++ {
		qb.argCounter++
		clause = strings.Replace(clause, "?", fmt.Sprintf("$%d", qb.argCounter), 1)
	}
	
	qb.whereClauses = append(qb.whereClauses, clause)
	qb.args = append(qb.args, args...)
}

// AddOrderBy adds an ORDER BY clause
func (qb *QueryBuilder) AddOrderBy(field string, order core.SortOrder) {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", field, order))
}

// SetLimit sets the LIMIT clause
func (qb *QueryBuilder) SetLimit(limit int) {
	qb.limit = limit
}

// SetOffset sets the OFFSET clause
func (qb *QueryBuilder) SetOffset(offset int) {
	qb.offset = offset
}

// ApplyFilters applies all filters to the query
func (qb *QueryBuilder) ApplyFilters(filters []filters.Filter) error {
	for _, filter := range filters {
		if err := filter.ApplyToQuery(qb); err != nil {
			return fmt.Errorf("failed to apply filter %s: %w", filter.Type(), err)
		}
	}
	return nil
}

// Build builds the final SQL query
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var parts []string
	
	// SELECT clause
	parts = append(parts, qb.baseQuery)
	
	// WHERE clause
	if len(qb.whereClauses) > 1 {  // Skip "1=1" if there are no other clauses
		parts = append(parts, "WHERE "+strings.Join(qb.whereClauses, " AND "))
	}
	
	// ORDER BY clause
	if len(qb.orderBy) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(qb.orderBy, ", "))
	} else {
		parts = append(parts, "ORDER BY starts_at DESC")  // Default sort
	}
	
	// LIMIT clause
	if qb.limit > 0 {
		qb.argCounter++
		parts = append(parts, fmt.Sprintf("LIMIT $%d", qb.argCounter))
		qb.args = append(qb.args, qb.limit)
	}
	
	// OFFSET clause
	if qb.offset > 0 {
		qb.argCounter++
		parts = append(parts, fmt.Sprintf("OFFSET $%d", qb.argCounter))
		qb.args = append(qb.args, qb.offset)
	}
	
	query := strings.Join(parts, " ")
	return query, qb.args
}

// BuildCount builds a COUNT query
func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	var parts []string
	
	// SELECT COUNT(*) clause
	parts = append(parts, "SELECT COUNT(*) FROM alerts")
	
	// WHERE clause (reuse from main query)
	if len(qb.whereClauses) > 1 {
		parts = append(parts, "WHERE "+strings.Join(qb.whereClauses, " AND "))
	}
	
	query := strings.Join(parts, " ")
	return query, qb.args
}

// OptimizationHints returns query optimization hints
func (qb *QueryBuilder) OptimizationHints() []string {
	var hints []string
	
	if qb.useGINIndex {
		hints = append(hints, "Use GIN index for JSONB queries")
	}
	if qb.usePartialIdx {
		hints = append(hints, "Use partial index for status=firing")
	}
	
	return hints
}
```

### 2.3 Caching Layer Design

#### 2-Tier Cache Manager
```go
// pkg/history/cache/manager.go
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	
	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// CacheManager manages 2-tier caching (L1: Ristretto, L2: Redis)
type CacheManager struct {
	l1Cache      *ristretto.Cache
	l2Cache      *redis.Client
	l1TTL        time.Duration
	l2TTL        time.Duration
	logger       *slog.Logger
	metrics      *CacheMetrics
	compression  bool
}

// CacheMetrics contains Prometheus metrics for cache operations
type CacheMetrics struct {
	Hits      *prometheus.CounterVec
	Misses    *prometheus.CounterVec
	Evictions *prometheus.CounterVec
	Errors    *prometheus.CounterVec
	Size      *prometheus.GaugeVec
	Latency   *prometheus.HistogramVec
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	L1MaxEntries  int64
	L1MaxSizeMB   int64
	L1TTL         time.Duration
	L2TTL         time.Duration
	Compression   bool
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cfg *CacheConfig, logger *slog.Logger) (*CacheManager, error) {
	// Initialize L1 cache (Ristretto)
	l1Cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cfg.L1MaxEntries * 10,  // Bloom filter size
		MaxCost:     cfg.L1MaxSizeMB << 20,   // Convert MB to bytes
		BufferItems: 64,                       // Size of buffer for async ops
		Metrics:     true,                     // Enable metrics
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}
	
	// Initialize L2 cache (Redis)
	l2Cache := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		MaxRetries:   3,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	
	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l2Cache.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	return &CacheManager{
		l1Cache:     l1Cache,
		l2Cache:     l2Cache,
		l1TTL:       cfg.L1TTL,
		l2TTL:       cfg.L2TTL,
		logger:      logger,
		compression: cfg.Compression,
		metrics:     initCacheMetrics(),
	}, nil
}

// Get retrieves a value from cache (L1 first, then L2)
func (cm *CacheManager) Get(ctx context.Context, key string) (*core.HistoryResponse, bool) {
	start := time.Now()
	
	// Try L1 cache first
	if value, found := cm.l1Cache.Get(key); found {
		cm.metrics.Hits.WithLabelValues("l1").Inc()
		cm.metrics.Latency.WithLabelValues("l1", "hit").Observe(time.Since(start).Seconds())
		return value.(*core.HistoryResponse), true
	}
	
	cm.metrics.Misses.WithLabelValues("l1").Inc()
	
	// Try L2 cache (Redis)
	l2Start := time.Now()
	data, err := cm.l2Cache.Get(ctx, key).Bytes()
	if err == redis.Nil {
		cm.metrics.Misses.WithLabelValues("l2").Inc()
		cm.metrics.Latency.WithLabelValues("l2", "miss").Observe(time.Since(l2Start).Seconds())
		return nil, false
	}
	if err != nil {
		cm.metrics.Errors.WithLabelValues("l2", "get").Inc()
		cm.logger.Error("L2 cache get error", "error", err, "key", key)
		return nil, false
	}
	
	// Decompress if needed
	if cm.compression {
		data, err = cm.decompress(data)
		if err != nil {
			cm.metrics.Errors.WithLabelValues("l2", "decompress").Inc()
			cm.logger.Error("Failed to decompress L2 cache data", "error", err)
			return nil, false
		}
	}
	
	// Deserialize
	var response core.HistoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		cm.metrics.Errors.WithLabelValues("l2", "unmarshal").Inc()
		cm.logger.Error("Failed to unmarshal L2 cache data", "error", err)
		return nil, false
	}
	
	cm.metrics.Hits.WithLabelValues("l2").Inc()
	cm.metrics.Latency.WithLabelValues("l2", "hit").Observe(time.Since(l2Start).Seconds())
	
	// Populate L1 cache for next time
	cm.l1Cache.SetWithTTL(key, &response, 1, cm.l1TTL)
	
	return &response, true
}

// Set stores a value in both L1 and L2 caches
func (cm *CacheManager) Set(ctx context.Context, key string, value *core.HistoryResponse) error {
	start := time.Now()
	
	// Store in L1 cache (in-memory)
	cm.l1Cache.SetWithTTL(key, value, 1, cm.l1TTL)
	cm.metrics.Latency.WithLabelValues("l1", "set").Observe(time.Since(start).Seconds())
	
	// Serialize for L2 cache
	data, err := json.Marshal(value)
	if err != nil {
		cm.metrics.Errors.WithLabelValues("l2", "marshal").Inc()
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}
	
	// Compress if enabled
	if cm.compression {
		data, err = cm.compress(data)
		if err != nil {
			cm.metrics.Errors.WithLabelValues("l2", "compress").Inc()
			return fmt.Errorf("failed to compress cache value: %w", err)
		}
	}
	
	// Store in L2 cache (Redis)
	l2Start := time.Now()
	if err := cm.l2Cache.Set(ctx, key, data, cm.l2TTL).Err(); err != nil {
		cm.metrics.Errors.WithLabelValues("l2", "set").Inc()
		return fmt.Errorf("failed to set L2 cache: %w", err)
	}
	cm.metrics.Latency.WithLabelValues("l2", "set").Observe(time.Since(l2Start).Seconds())
	
	return nil
}

// Invalidate removes a key from both caches
func (cm *CacheManager) Invalidate(ctx context.Context, key string) error {
	// Remove from L1
	cm.l1Cache.Del(key)
	
	// Remove from L2
	if err := cm.l2Cache.Del(ctx, key).Err(); err != nil && err != redis.Nil {
		cm.metrics.Errors.WithLabelValues("l2", "del").Inc()
		return fmt.Errorf("failed to invalidate L2 cache: %w", err)
	}
	
	return nil
}

// InvalidatePattern removes all keys matching a pattern from L2 cache
// Note: L1 cache (Ristretto) doesn't support pattern deletion, it uses TTL
func (cm *CacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	// Scan for matching keys
	var cursor uint64
	var deletedCount int
	
	for {
		keys, newCursor, err := cm.l2Cache.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			cm.metrics.Errors.WithLabelValues("l2", "scan").Inc()
			return fmt.Errorf("failed to scan keys: %w", err)
		}
		
		// Delete matching keys
		if len(keys) > 0 {
			if err := cm.l2Cache.Del(ctx, keys...).Err(); err != nil {
				cm.metrics.Errors.WithLabelValues("l2", "del").Inc()
				return fmt.Errorf("failed to delete keys: %w", err)
			}
			deletedCount += len(keys)
		}
		
		cursor = newCursor
		if cursor == 0 {
			break
		}
	}
	
	cm.logger.Info("Invalidated cache pattern", "pattern", pattern, "deleted_count", deletedCount)
	return nil
}

// GenerateCacheKey generates a cache key from request parameters
func (cm *CacheManager) GenerateCacheKey(req *core.HistoryRequest) string {
	// Serialize request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		cm.logger.Error("Failed to marshal request for cache key", "error", err)
		return ""
	}
	
	// Generate SHA-256 hash
	hash := sha256.Sum256(data)
	hashStr := base64.URLEncoding.EncodeToString(hash[:])
	
	// Format: "history:v2:{hash}"
	return fmt.Sprintf("history:v2:%s", hashStr)
}

// compress compresses data using gzip
func (cm *CacheManager) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompress decompresses data using gzip
func (cm *CacheManager) decompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	
	return io.ReadAll(gzipReader)
}

// Stats returns cache statistics
func (cm *CacheManager) Stats() map[string]interface{} {
	l1Metrics := cm.l1Cache.Metrics
	
	return map[string]interface{}{
		"l1": map[string]interface{}{
			"hits":      l1Metrics.Hits(),
			"misses":    l1Metrics.Misses(),
			"hit_ratio": l1Metrics.Ratio(),
			"cost":      l1Metrics.CostAdded() - l1Metrics.CostEvicted(),
		},
		"l2": map[string]interface{}{
			"connected": cm.l2Cache.Ping(context.Background()).Err() == nil,
		},
	}
}

// Close closes cache connections
func (cm *CacheManager) Close() error {
	cm.l1Cache.Close()
	return cm.l2Cache.Close()
}
```

#### Cache Warming Strategy
```go
// pkg/history/cache/warmer.go
package cache

import (
	"context"
	"log/slog"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// CacheWarmer pre-populates cache with popular queries
type CacheWarmer struct {
	cacheManager *CacheManager
	repository   core.AlertHistoryRepository
	logger       *slog.Logger
	stopCh       chan struct{}
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(
	cacheManager *CacheManager,
	repository core.AlertHistoryRepository,
	logger *slog.Logger,
) *CacheWarmer {
	return &CacheWarmer{
		cacheManager: cacheManager,
		repository:   repository,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the cache warming background worker
func (cw *CacheWarmer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Warm cache immediately on start
	cw.warmCache(ctx)
	
	for {
		select {
		case <-ticker.C:
			cw.warmCache(ctx)
		case <-cw.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the cache warming worker
func (cw *CacheWarmer) Stop() {
	close(cw.stopCh)
}

// warmCache pre-populates cache with popular queries
func (cw *CacheWarmer) warmCache(ctx context.Context) {
	cw.logger.Info("Starting cache warming")
	start := time.Now()
	
	// Define popular query patterns
	popularQueries := []struct {
		name string
		req  *core.HistoryRequest
	}{
		{
			name: "recent_firing_critical",
			req: &core.HistoryRequest{
				Filters: &core.AlertFilters{
					Status:   &core.StatusFiring,
					Severity: ptrString("critical"),
				},
				Pagination: &core.Pagination{
					Page:    1,
					PerPage: 50,
				},
			},
		},
		{
			name: "recent_all",
			req: &core.HistoryRequest{
				Filters:    &core.AlertFilters{},
				Pagination: &core.Pagination{
					Page:    1,
					PerPage: 50,
				},
			},
		},
		// Add more popular query patterns...
	}
	
	// Warm cache for each popular query
	warmed := 0
	for _, pq := range popularQueries {
		// Check if already cached
		cacheKey := cw.cacheManager.GenerateCacheKey(pq.req)
		if _, found := cw.cacheManager.Get(ctx, cacheKey); found {
			continue  // Already cached
		}
		
		// Query from database
		response, err := cw.repository.GetHistory(ctx, pq.req)
		if err != nil {
			cw.logger.Warn("Failed to warm cache for query",
				"query", pq.name,
				"error", err)
			continue
		}
		
		// Store in cache
		if err := cw.cacheManager.Set(ctx, cacheKey, response); err != nil {
			cw.logger.Warn("Failed to cache warmed query",
				"query", pq.name,
				"error", err)
			continue
		}
		
		warmed++
	}
	
	duration := time.Since(start)
	cw.logger.Info("Cache warming complete",
		"warmed_queries", warmed,
		"duration_ms", duration.Milliseconds())
}

func ptrString(s string) *string {
	return &s
}
```

---

## Continued in next response due to length...

**Document Status**: 🔄 IN PROGRESS (50% Complete)  
**Next Section**: Middleware Stack Design, Security Design, Performance Optimization  
**Estimated Completion**: 2 more responses  

Shall I continue with the remaining sections?

