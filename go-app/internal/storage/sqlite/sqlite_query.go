// Package sqlite implements query operations (ListAlerts, CountAlerts) with filtering.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ListAlerts implements core.AlertStorage.ListAlerts.
// Supports filtering, pagination, and sorting.
// Returns *AlertList with pagination metadata.
//
// Performance: < 20ms (p95) for 100 rows with filters
// Thread-safe: Yes (read-only operation)
func (s *SQLiteStorage) ListAlerts(
	ctx context.Context,
	filters *core.AlertFilters,
) (*core.AlertList, error) {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Default filters if nil
	if filters == nil {
		filters = &core.AlertFilters{}
	}

	// Get total count (for pagination metadata)
	total, err := s.CountAlerts(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Build SQL query with filters
	query, args := s.buildListQuery(filters)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()

	// Scan results
	alerts := []*core.Alert{}
	for rows.Next() {
		alert, err := s.scanAlert(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)

	s.logger.Debug("Alerts listed",
		"count", len(alerts),
		"total", total,
		"duration_ms", duration.Milliseconds(),
	)

	// Return AlertList with pagination metadata
	return &core.AlertList{
		Alerts: alerts,
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}, nil
}

// CountAlerts counts alerts matching filter criteria (internal helper).
// Used by ListAlerts for pagination metadata.
//
// Performance: < 5ms (p95) with filters
// Thread-safe: Yes
func (s *SQLiteStorage) CountAlerts(
	ctx context.Context,
	filters *core.AlertFilters,
) (int, error) {
	// Default filters if nil
	if filters == nil {
		filters = &core.AlertFilters{}
	}

	// Build COUNT query (reuse filter logic)
	query, args := s.buildCountQuery(filters)

	// Execute query
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	return count, nil
}

// buildListQuery constructs SELECT query with filters, pagination, sorting.
func (s *SQLiteStorage) buildListQuery(filters *core.AlertFilters) (string, []interface{}) {
	// Base query
	query := `
SELECT fingerprint, status, severity, namespace, alert_name,
       labels, annotations, starts_at, ends_at, generator_url,
       created_at, updated_at
FROM alerts
WHERE 1=1
`
	args := []interface{}{}

	// Apply filters
	query, args = s.applyFilters(query, args, filters)

	// Sorting (AlertFilters doesn't have SortBy, use default)
	sortBy := "created_at"   // Default sort field
	sortOrder := "DESC"      // Default sort order

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	return query, args
}

// buildCountQuery constructs COUNT query with same filters as list query.
func (s *SQLiteStorage) buildCountQuery(filters *core.AlertFilters) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM alerts WHERE 1=1"
	args := []interface{}{}

	// Apply same filters as list query (but no sorting/pagination)
	query, args = s.applyFilters(query, args, filters)

	return query, args
}

// applyFilters adds WHERE clauses based on filter parameters.
// AlertFilters uses pointer fields, so need to check for nil.
func (s *SQLiteStorage) applyFilters(query string, args []interface{}, filters *core.AlertFilters) (string, []interface{}) {
	// Filter by status (pointer field)
	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filters.Status))
	}

	// Filter by severity (pointer field)
	if filters.Severity != nil {
		query += " AND severity = ?"
		args = append(args, *filters.Severity)
	}

	// Filter by namespace (pointer field)
	if filters.Namespace != nil {
		query += " AND namespace = ?"
		args = append(args, *filters.Namespace)
	}

	// Filter by labels (map field)
	// Simple implementation: match ALL labels (AND logic)
	// Uses SQL LIKE for simple matching (more efficient than JSON extraction)
	for key, value := range filters.Labels {
		query += " AND labels LIKE ?"
		args = append(args, "%\""+key+"\":\""+value+"\"%")
	}

	// Filter by time range (pointer field)
	if filters.TimeRange != nil {
		if filters.TimeRange.From != nil {
			query += " AND starts_at >= ?"
			args = append(args, filters.TimeRange.From.UnixMilli())
		}
		if filters.TimeRange.To != nil {
			query += " AND starts_at <= ?"
			args = append(args, filters.TimeRange.To.UnixMilli())
		}
	}

	return query, args
}

// placeholders generates SQL placeholders ("?", "?, ?", "?, ?, ?", etc.).
// Used for IN clauses with variable number of values.
func (s *SQLiteStorage) placeholders(count int) string {
	if count == 0 {
		return ""
	}

	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = "?"
	}

	return strings.Join(parts, ", ")
}

// scanAlert scans a single alert row from SQL result set.
// Handles JSON deserialization, timestamp conversion, NULL values.
func (s *SQLiteStorage) scanAlert(rows *sql.Rows) (*core.Alert, error) {
	var alert core.Alert
	var labelsJSON, annotationsJSON string
	var severity, namespace string
	var startsAt, createdAt, updatedAt int64
	var endsAtMs sql.NullInt64
	var generatorURL sql.NullString

	// Scan all columns
	if err := rows.Scan(
		&alert.Fingerprint,
		&alert.Status,
		&severity,
		&namespace,
		&alert.AlertName,
		&labelsJSON,
		&annotationsJSON,
		&startsAt,
		&endsAtMs,
		&generatorURL,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(labelsJSON), &alert.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	if err := json.Unmarshal([]byte(annotationsJSON), &alert.Annotations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	// Set severity and namespace in labels (where they're stored)
	if alert.Labels == nil {
		alert.Labels = make(map[string]string)
	}
	if severity != "" {
		alert.Labels["severity"] = severity
	}
	if namespace != "" {
		alert.Labels["namespace"] = namespace
	}

	// Convert Unix milliseconds to time.Time
	alert.StartsAt = time.UnixMilli(startsAt)
	if endsAtMs.Valid {
		endsAt := time.UnixMilli(endsAtMs.Int64)
		alert.EndsAt = &endsAt
	}

	// Handle nullable generator_url
	if generatorURL.Valid {
		genURL := generatorURL.String
		alert.GeneratorURL = &genURL
	}

	return &alert, nil
}
