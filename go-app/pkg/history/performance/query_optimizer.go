package performance

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// QueryOptimizer provides query optimization hints and analysis
type QueryOptimizer struct {
	logger *slog.Logger
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(logger *slog.Logger) *QueryOptimizer {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &QueryOptimizer{
		logger: logger,
	}
}

// OptimizeQuery analyzes and optimizes a query based on filters
func (qo *QueryOptimizer) OptimizeQuery(qb *query.Builder, req *core.HistoryRequest) {
	// Analyze filters to determine optimal index usage
	if req.Filters != nil {
		// If filtering by status=firing, use partial index hint
		if req.Filters.Status != nil && *req.Filters.Status == core.StatusFiring {
			qb.MarkPartialIndexUsage()
		}
		
		// If filtering by labels, mark GIN index usage
		if req.Filters.Labels != nil && len(req.Filters.Labels) > 0 {
			qb.MarkGINIndexUsage()
		}
		
		// If filtering by severity, ensure severity index is used
		if req.Filters.Severity != nil {
			qb.MarkGINIndexUsage() // Severity is in labels JSONB
		}
		
		// If filtering by namespace, ensure namespace index is used
		if req.Filters.Namespace != nil {
			qb.MarkGINIndexUsage() // Namespace is in labels JSONB
		}
	}
	
	// Optimize sorting - prefer indexed columns
	if req.Sorting != nil {
		// Ensure sorting uses indexed column
		// starts_at is indexed, so prefer it
		if req.Sorting.Field == "starts_at" {
			// Already optimal
		} else if req.Sorting.Field == "created_at" {
			// created_at is also indexed
		}
	}
}

// AnalyzeQueryPlan analyzes query execution plan (requires EXPLAIN)
func (qo *QueryOptimizer) AnalyzeQueryPlan(ctx context.Context, sql string, args []interface{}) (string, error) {
	// This would execute EXPLAIN ANALYZE on the query
	// For now, return placeholder
	return "Query plan analysis not implemented (requires database connection)", nil
}

// SuggestIndexes suggests indexes based on query patterns
func (qo *QueryOptimizer) SuggestIndexes(queries []string) []string {
	suggestions := []string{}
	
	// Analyze query patterns and suggest indexes
	// This is a simplified version - real implementation would parse SQL
	
	for _, query := range queries {
		if strings.Contains(query, "status = 'firing'") && strings.Contains(query, "labels->>'severity'") {
			suggestions = append(suggestions, "idx_alerts_status_severity_time")
		}
		if strings.Contains(query, "labels->>'namespace'") && strings.Contains(query, "status = 'firing'") {
			suggestions = append(suggestions, "idx_alerts_namespace_status_time")
		}
		if strings.Contains(query, "ends_at IS NOT NULL") {
			suggestions = append(suggestions, "idx_alerts_ends_at")
		}
		if strings.Contains(query, "generator_url =") {
			suggestions = append(suggestions, "idx_alerts_generator_url")
		}
		if strings.Contains(query, "fingerprint =") && strings.Contains(query, "ORDER BY starts_at") {
			suggestions = append(suggestions, "idx_alerts_fingerprint_timeline")
		}
	}
	
	return suggestions
}

// EstimateQueryCost estimates query cost based on filters
func (qo *QueryOptimizer) EstimateQueryCost(req *core.HistoryRequest) QueryCost {
	cost := QueryCost{
		EstimatedRows: 1000, // Default estimate
		IndexUsed:     "unknown",
		CostLevel:     "low",
	}
	
	if req.Filters == nil {
		return cost
	}
	
	// Estimate based on filter selectivity
	selectivity := 1.0
	
	if req.Filters.Status != nil {
		selectivity *= 0.5 // Status filter reduces by ~50%
		cost.IndexUsed = "idx_alerts_status"
	}
	
	if req.Filters.Severity != nil {
		selectivity *= 0.25 // Severity filter reduces by ~75%
		cost.IndexUsed = "idx_alerts_status_severity_time"
	}
	
	if req.Filters.Namespace != nil {
		selectivity *= 0.1 // Namespace filter reduces by ~90%
		cost.IndexUsed = "idx_alerts_namespace_status_time"
	}
	
	if req.Filters.TimeRange != nil {
		selectivity *= 0.2 // Time range filter reduces by ~80%
	}
	
	cost.EstimatedRows = int64(float64(10000) * selectivity)
	
	// Determine cost level
	if cost.EstimatedRows < 100 {
		cost.CostLevel = "very_low"
	} else if cost.EstimatedRows < 1000 {
		cost.CostLevel = "low"
	} else if cost.EstimatedRows < 10000 {
		cost.CostLevel = "medium"
	} else {
		cost.CostLevel = "high"
	}
	
	return cost
}

// QueryCost represents estimated query cost
type QueryCost struct {
	EstimatedRows int64
	IndexUsed     string
	CostLevel     string // very_low, low, medium, high
}

// OptimizePagination optimizes pagination for large result sets
func (qo *QueryOptimizer) OptimizePagination(req *core.HistoryRequest) *core.Pagination {
	if req.Pagination == nil {
		return &core.Pagination{
			Page:    1,
			PerPage: 50,
		}
	}
	
	// Limit per_page to prevent expensive queries
	if req.Pagination.PerPage > 1000 {
		qo.logger.Warn("PerPage exceeds maximum, limiting to 1000",
			"requested", req.Pagination.PerPage)
		req.Pagination.PerPage = 1000
	}
	
	// For deep pagination (page > 100), suggest cursor-based pagination
	if req.Pagination.Page > 100 {
		qo.logger.Warn("Deep pagination detected, consider cursor-based pagination",
			"page", req.Pagination.Page)
	}
	
	return req.Pagination
}

// ValidateQueryComplexity validates query complexity to prevent expensive queries
func (qo *QueryOptimizer) ValidateQueryComplexity(req *core.HistoryRequest) error {
	complexity := 0
	
	if req.Filters != nil {
		// Count filter complexity
		if req.Filters.Labels != nil && len(req.Filters.Labels) > 10 {
			complexity += len(req.Filters.Labels) - 10
		}
		
		// Regex filters are expensive
		// This would be checked if we had regex filter info
	}
	
	// Deep pagination is expensive
	if req.Pagination != nil && req.Pagination.Page > 1000 {
		complexity += 10
	}
	
	if complexity > 20 {
		return fmt.Errorf("query complexity too high: %d (max 20)", complexity)
	}
	
	return nil
}

