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
//
// Filters:
//   - Status: IN (firing, resolved)
//   - Severity: IN (critical, warning, info, unknown)
//   - Namespace: IN (...)
//   - Fingerprints: IN (...)
//   - Time ranges: starts_at, ends_at (not implemented yet)
//
// Pagination:
//   - Limit: max results (0 = no limit)
//   - Offset: skip N results
//
// Sorting:
//   - SortBy: created_at, starts_at, updated_at, alert_name
//   - SortOrder: ASC, DESC (default DESC)
//
// Performance: < 20ms (p95) for 100 rows with filters
// Thread-safe: Yes (read-only operation)
func (s *SQLiteStorage) ListAlerts(
	ctx context.Context,
	filter core.AlertFilter,
) ([]*core.Alert, error) {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build SQL query with filters
	query, args := s.buildListQuery(filter)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		storage.RecordOperation("list", "sqlite", "error")
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()

	// Scan results
	alerts := []*core.Alert{}
	for rows.Next() {
		alert, err := s.scanAlert(rows)
		if err != nil {
			storage.RecordError("list", "sqlite", storage.ErrorTypeValidation)
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		storage.RecordOperation("list", "sqlite", "error")
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	storage.RecordOperation("list", "sqlite", "success")
	storage.RecordOperationDuration("list", "sqlite", duration.Seconds())

	s.logger.Debug("Alerts listed",
		"count", len(alerts),
		"duration_ms", duration.Milliseconds(),
		"has_filters", filter.Status != nil || filter.Severity != nil || filter.Namespace != nil,
	)

	return alerts, nil
}

// CountAlerts implements core.AlertStorage.CountAlerts.
// Counts alerts matching filter criteria.
//
// Performance: < 5ms (p95) with filters
// Thread-safe: Yes
func (s *SQLiteStorage) CountAlerts(
	ctx context.Context,
	filter core.AlertFilter,
) (int, error) {
	startTime := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build COUNT query (reuse filter logic)
	query, args := s.buildCountQuery(filter)

	// Execute query
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)

	if err != nil {
		storage.RecordOperation("count", "sqlite", "error")
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	storage.RecordOperation("count", "sqlite", "success")
	storage.RecordOperationDuration("count", "sqlite", duration.Seconds())

	s.logger.Debug("Alerts counted",
		"count", count,
		"duration_ms", duration.Milliseconds(),
	)

	return count, nil
}

// buildListQuery constructs SELECT query with filters, pagination, sorting.
func (s *SQLiteStorage) buildListQuery(filter core.AlertFilter) (string, []interface{}) {
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
	query, args = s.applyFilters(query, args, filter)

	// Sorting
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at" // Default sort by creation time
	}

	// Validate sort field (prevent SQL injection)
	validSortFields := map[string]bool{
		"created_at":  true,
		"starts_at":   true,
		"updated_at":  true,
		"alert_name":  true,
		"status":      true,
		"severity":    true,
		"namespace":   true,
		"fingerprint": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Fallback to default
	}

	sortOrder := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	return query, args
}

// buildCountQuery constructs COUNT query with same filters as list query.
func (s *SQLiteStorage) buildCountQuery(filter core.AlertFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM alerts WHERE 1=1"
	args := []interface{}{}

	// Apply same filters as list query (but no sorting/pagination)
	query, args = s.applyFilters(query, args, filter)

	return query, args
}

// applyFilters adds WHERE clauses based on filter parameters.
func (s *SQLiteStorage) applyFilters(query string, args []interface{}, filter core.AlertFilter) (string, []interface{}) {
	// Filter by status (firing, resolved)
	if len(filter.Status) > 0 {
		placeholders := s.placeholders(len(filter.Status))
		query += " AND status IN (" + placeholders + ")"
		for _, status := range filter.Status {
			args = append(args, status)
		}
	}

	// Filter by severity (critical, warning, info, unknown)
	if len(filter.Severity) > 0 {
		placeholders := s.placeholders(len(filter.Severity))
		query += " AND severity IN (" + placeholders + ")"
		for _, severity := range filter.Severity {
			args = append(args, severity)
		}
	}

	// Filter by namespace
	if len(filter.Namespace) > 0 {
		placeholders := s.placeholders(len(filter.Namespace))
		query += " AND namespace IN (" + placeholders + ")"
		for _, ns := range filter.Namespace {
			args = append(args, ns)
		}
	}

	// Filter by fingerprints (useful for bulk operations)
	if len(filter.Fingerprints) > 0 {
		placeholders := s.placeholders(len(filter.Fingerprints))
		query += " AND fingerprint IN (" + placeholders + ")"
		for _, fp := range filter.Fingerprints {
			args = append(args, fp)
		}
	}

	// TODO: Add time range filters (starts_at, ends_at)
	// This would require additional filter fields in core.AlertFilter

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
	var startsAt, createdAt, updatedAt int64
	var endsAt *int64
	var generatorURL sql.NullString

	// Scan all columns
	if err := rows.Scan(
		&alert.Fingerprint,
		&alert.Status,
		&alert.Severity,
		&alert.Namespace,
		&alert.AlertName,
		&labelsJSON,
		&annotationsJSON,
		&startsAt,
		&endsAt,
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

	// Convert Unix milliseconds to time.Time
	alert.StartsAt = time.UnixMilli(startsAt)
	if endsAt != nil {
		alert.EndsAt = time.UnixMilli(*endsAt)
	}
	alert.CreatedAt = time.UnixMilli(createdAt)
	alert.UpdatedAt = time.UnixMilli(updatedAt)

	// Handle nullable generator_url
	if generatorURL.Valid {
		alert.GeneratorURL = generatorURL.String
	}

	return &alert, nil
}
