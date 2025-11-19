// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"time"
)

// QueryParameters represents parsed query parameters for GET /api/v2/alerts.
//
// This structure supports all Alertmanager-compatible query parameters plus
// additional filtering, pagination, and sorting capabilities.
//
// Alertmanager Standard Parameters:
//   - filter: Label matcher expression (e.g., {alertname="HighCPU",severity=~"critical|warning"})
//   - receiver: Filter by receiver name
//   - silenced: Include/exclude silenced alerts
//   - inhibited: Include/exclude inhibited alerts
//   - active: Include only active alerts
//
// Extended Parameters (150% quality):
//   - status: Filter by alert status (firing/resolved)
//   - severity: Filter by severity level
//   - startTime/endTime: Filter by time range
//   - page/limit: Pagination
//   - sort: Sort field and direction
//
// Example Usage:
//   params, err := ParseQueryParameters(r.URL.Query())
//   if err != nil {
//       return nil, fmt.Errorf("invalid query parameters: %w", err)
//   }
type QueryParameters struct {
	// Alertmanager standard filters
	Filter    string // Label matcher expression (e.g., {alertname="HighCPU"})
	Receiver  string // Filter by receiver name
	Silenced  *bool  // Include silenced alerts (nil = all, true = only silenced, false = only not silenced)
	Inhibited *bool  // Include inhibited alerts (nil = all, true = only inhibited, false = only not inhibited)
	Active    *bool  // Include only active alerts (nil = all, true = only active, false = include resolved)

	// Extended filters (150% quality)
	Status    string    // Filter by status: "firing", "resolved", or "" (all)
	Severity  string    // Filter by severity level (e.g., "critical", "warning")
	StartTime time.Time // Filter by start time (alerts starting after this time)
	EndTime   time.Time // Filter by end time (alerts ending before this time)

	// Pagination
	Page  int // Page number (1-indexed, default: 1)
	Limit int // Results per page (default: 100, max: 1000)

	// Sorting
	SortBy    string // Sort field (e.g., "startsAt", "severity", "alertname")
	SortOrder string // Sort direction: "asc" or "desc" (default: "desc")
}

// LabelMatcher represents a single label matcher in a filter expression.
//
// Supports Prometheus label matcher operators:
//   - = : Exact match
//   - != : Not equal
//   - =~ : Regex match
//   - !~ : Negative regex match
//
// Example:
//   {Name: "severity", Operator: "=", Value: "critical"}
//   {Name: "alertname", Operator: "=~", Value: "High.*"}
type LabelMatcher struct {
	Name     string // Label name
	Operator string // Operator: "=", "!=", "=~", "!~"
	Value    string // Label value or regex pattern
}

// AlertmanagerAlert represents a single alert in Alertmanager v2 format.
//
// This is the wire format used in GET /api/v2/alerts responses.
// It's compatible with:
//   - Alertmanager API v2
//   - Grafana Alerting data source
//   - Prometheus alert queries
//   - amtool CLI
//
// Example:
//   {
//     "labels": {
//       "alertname": "HighCPU",
//       "severity": "critical",
//       "instance": "node-1"
//     },
//     "annotations": {
//       "summary": "CPU usage above 90%",
//       "description": "CPU usage is 95% for 5 minutes"
//     },
//     "startsAt": "2025-11-19T10:00:00Z",
//     "endsAt": "2025-11-19T10:15:00Z",
//     "generatorURL": "http://prometheus:9090/graph?...",
//     "status": {
//       "state": "active",
//       "silencedBy": [],
//       "inhibitedBy": []
//     },
//     "receivers": ["team-ops"],
//     "fingerprint": "abc123def456"
//   }
type AlertmanagerAlert struct {
	Labels       map[string]string `json:"labels"`                // Alert labels
	Annotations  map[string]string `json:"annotations"`           // Alert annotations
	StartsAt     string            `json:"startsAt"`              // Start timestamp (RFC3339)
	EndsAt       string            `json:"endsAt,omitempty"`      // End timestamp (RFC3339, omitempty for active)
	GeneratorURL string            `json:"generatorURL"`          // Generator URL
	Status       AlertStatus       `json:"status"`                // Alert status
	Receivers    []string          `json:"receivers"`             // Receiver names
	Fingerprint  string            `json:"fingerprint"`           // Alert fingerprint
}

// AlertStatus represents the status of an alert.
//
// This includes:
//   - state: Current alert state ("active", "suppressed", "unprocessed")
//   - silencedBy: List of silence IDs that silence this alert
//   - inhibitedBy: List of alert fingerprints that inhibit this alert
//
// Example:
//   {
//     "state": "active",
//     "silencedBy": [],
//     "inhibitedBy": []
//   }
type AlertStatus struct {
	State       string   `json:"state"`                 // Alert state: "active", "suppressed", "unprocessed"
	SilencedBy  []string `json:"silencedBy"`            // Silence IDs (empty array if not silenced)
	InhibitedBy []string `json:"inhibitedBy,omitempty"` // Inhibiting alert fingerprints (omitempty)
}

// AlertmanagerListResponse represents the response for GET /api/v2/alerts.
//
// This is compatible with Alertmanager API v2 and follows the standard
// response format used by Prometheus ecosystem tools.
//
// Response format:
//   {
//     "status": "success",
//     "data": {
//       "alerts": [...],
//       "total": 42,
//       "page": 1,
//       "limit": 100
//     }
//   }
//
// Or for errors:
//   {
//     "status": "error",
//     "error": "invalid filter expression: {alertname"
//   }
type AlertmanagerListResponse struct {
	Status string                 `json:"status"` // "success" or "error"
	Data   *AlertmanagerListData  `json:"data,omitempty"`   // Response data (success only)
	Error  string                 `json:"error,omitempty"`  // Error message (error only)
}

// AlertmanagerListData contains the list of alerts and pagination metadata.
//
// Fields:
//   - Alerts: Array of alerts in Alertmanager format
//   - Total: Total number of alerts matching the filter (for pagination)
//   - Page: Current page number (1-indexed)
//   - Limit: Results per page
//
// Example:
//   {
//     "alerts": [{...}, {...}],
//     "total": 42,
//     "page": 1,
//     "limit": 100
//   }
type AlertmanagerListData struct {
	Alerts []AlertmanagerAlert `json:"alerts"`           // Array of alerts
	Total  int                 `json:"total,omitempty"`  // Total count (for pagination)
	Page   int                 `json:"page,omitempty"`   // Current page
	Limit  int                 `json:"limit,omitempty"`  // Results per page
}

// DefaultQueryParameters returns default query parameters.
//
// Defaults:
//   - Page: 1
//   - Limit: 100
//   - SortBy: "startsAt"
//   - SortOrder: "desc"
//   - Status: "" (all)
//   - Silenced, Inhibited, Active: nil (all)
//
// Returns:
//   - *QueryParameters: Configuration with default values
func DefaultQueryParameters() *QueryParameters {
	return &QueryParameters{
		Page:      1,
		Limit:     100,
		SortBy:    "startsAt",
		SortOrder: "desc",
		Status:    "", // All statuses
		// Silenced, Inhibited, Active: nil means include all
	}
}

// ValidationError represents a query parameter validation error.
//
// Provides detailed context about which parameter failed validation and why.
//
// Example:
//   {
//     "parameter": "limit",
//     "message": "limit must be between 1 and 1000",
//     "value": "5000"
//   }
type ValidationError struct {
	Parameter string      `json:"parameter"` // Parameter name
	Message   string      `json:"message"`   // Error message
	Value     interface{} `json:"value"`     // Invalid value
}

// QueryValidationResult represents the result of query parameter validation.
//
// Fields:
//   - Valid: Whether all parameters are valid
//   - Errors: List of validation errors (if any)
//
// Example:
//   result := ValidateQueryParameters(params)
//   if !result.Valid {
//       return handleValidationError(result.Errors)
//   }
type QueryValidationResult struct {
	Valid  bool              `json:"valid"`  // Whether parameters are valid
	Errors []ValidationError `json:"errors"` // Validation errors
}

// MaxAlertsPerPage is the maximum number of alerts that can be returned in a single page.
//
// This limit prevents:
//   - Out-of-memory errors from large result sets
//   - Slow response times
//   - Database overload
//
// Default: 1000 alerts per page (Alertmanager standard)
const MaxAlertsPerPage = 1000

// DefaultAlertsPerPage is the default number of alerts per page.
//
// This provides a good balance between:
//   - Response size (not too large)
//   - Number of requests (not too many)
//   - User experience (reasonable page size)
//
// Default: 100 alerts per page
const DefaultAlertsPerPage = 100
