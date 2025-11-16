package query

import (
	"fmt"
	"strings"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Builder builds optimized SQL queries for alert history
type Builder struct {
	baseQuery    string
	whereClauses []string
	args         []interface{}
	argCounter   int
	orderBy      []string
	limit        int
	offset       int

	// Performance optimization flags
	useGINIndex   bool // Use GIN index for JSONB queries
	usePartialIdx bool // Use partial index for common queries
}

// NewBuilder creates a new query builder
func NewBuilder() *Builder {
	return &Builder{
		baseQuery:    "SELECT * FROM alerts",
		whereClauses: []string{"1=1"},
		args:         []interface{}{},
		argCounter:   0,
		orderBy:      []string{},
	}
}

// AddWhere adds a WHERE clause with arguments
// Placeholders '?' will be replaced with PostgreSQL placeholders '$N'
func (qb *Builder) AddWhere(clause string, args ...interface{}) {
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
func (qb *Builder) AddOrderBy(field string, order core.SortOrder) {
	// Validate field name to prevent SQL injection
	validFields := map[string]bool{
		"created_at": true,
		"starts_at":  true,
		"ends_at":    true,
		"updated_at": true,
		"status":     true,
		"severity":   false, // severity is in labels JSONB, handled specially
		"alert_name": true,
		"fingerprint": true,
	}

	if !validFields[field] {
		// For severity, we need to use labels->>'severity'
		if field == "severity" {
			qb.orderBy = append(qb.orderBy, fmt.Sprintf("labels->>'severity' %s", order))
		} else {
			// Default to starts_at if invalid field
			qb.orderBy = append(qb.orderBy, fmt.Sprintf("starts_at %s", core.SortOrderDesc))
		}
	} else {
		qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", field, order))
	}
}

// SetLimit sets the LIMIT clause
func (qb *Builder) SetLimit(limit int) {
	if limit > 0 {
		qb.limit = limit
	}
}

// SetOffset sets the OFFSET clause
func (qb *Builder) SetOffset(offset int) {
	if offset > 0 {
		qb.offset = offset
	}
}

// Build builds the final SQL query
func (qb *Builder) Build() (string, []interface{}) {
	var parts []string

	// SELECT clause
	parts = append(parts, qb.baseQuery)

	// WHERE clause
	if len(qb.whereClauses) > 1 { // Skip "1=1" if there are no other clauses
		parts = append(parts, "WHERE "+strings.Join(qb.whereClauses, " AND "))
	}

	// ORDER BY clause
	if len(qb.orderBy) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(qb.orderBy, ", "))
	} else {
		parts = append(parts, "ORDER BY starts_at DESC") // Default sort
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

// BuildCount builds a COUNT query (for pagination total)
func (qb *Builder) BuildCount() (string, []interface{}) {
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
func (qb *Builder) OptimizationHints() []string {
	var hints []string

	if qb.useGINIndex {
		hints = append(hints, "Use GIN index for JSONB queries")
	}
	if qb.usePartialIdx {
		hints = append(hints, "Use partial index for status=firing")
	}

	return hints
}

// MarkGINIndexUsage marks that GIN index should be used
func (qb *Builder) MarkGINIndexUsage() {
	qb.useGINIndex = true
}

// MarkPartialIndexUsage marks that partial index should be used
func (qb *Builder) MarkPartialIndexUsage() {
	qb.usePartialIdx = true
}
