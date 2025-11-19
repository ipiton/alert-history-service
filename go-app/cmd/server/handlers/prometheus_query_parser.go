// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseQueryParameters parses HTTP query parameters for GET /api/v2/alerts.
//
// This function extracts and validates all supported query parameters:
//   - Alertmanager standard: filter, receiver, silenced, inhibited, active
//   - Extended: status, severity, startTime, endTime
//   - Pagination: page, limit
//   - Sorting: sort (format: "field:direction")
//
// Example Usage:
//   params, err := ParseQueryParameters(r.URL.Query())
//   if err != nil {
//       http.Error(w, err.Error(), http.StatusBadRequest)
//       return
//   }
//
// Parameters:
//   - query: URL query parameters from http.Request
//
// Returns:
//   - *QueryParameters: Parsed parameters with defaults
//   - error: Parsing or validation error
func ParseQueryParameters(query url.Values) (*QueryParameters, error) {
	params := DefaultQueryParameters()

	// Parse Alertmanager standard filters
	if filter := query.Get("filter"); filter != "" {
		params.Filter = filter
	}
	if receiver := query.Get("receiver"); receiver != "" {
		params.Receiver = receiver
	}

	// Parse boolean filters (silenced, inhibited, active)
	if silenced := query.Get("silenced"); silenced != "" {
		val, err := parseBool(silenced)
		if err != nil {
			return nil, fmt.Errorf("invalid 'silenced' parameter: %w", err)
		}
		params.Silenced = &val
	}
	if inhibited := query.Get("inhibited"); inhibited != "" {
		val, err := parseBool(inhibited)
		if err != nil {
			return nil, fmt.Errorf("invalid 'inhibited' parameter: %w", err)
		}
		params.Inhibited = &val
	}
	if active := query.Get("active"); active != "" {
		val, err := parseBool(active)
		if err != nil {
			return nil, fmt.Errorf("invalid 'active' parameter: %w", err)
		}
		params.Active = &val
	}

	// Parse extended filters
	if status := query.Get("status"); status != "" {
		if status != "firing" && status != "resolved" {
			return nil, fmt.Errorf("invalid status: must be 'firing' or 'resolved', got %q", status)
		}
		params.Status = status
	}
	if severity := query.Get("severity"); severity != "" {
		params.Severity = severity
	}

	// Parse time range
	if startTime := query.Get("startTime"); startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			return nil, fmt.Errorf("invalid 'startTime' format (use RFC3339): %w", err)
		}
		params.StartTime = t
	}
	if endTime := query.Get("endTime"); endTime != "" {
		t, err := time.Parse(time.RFC3339, endTime)
		if err != nil {
			return nil, fmt.Errorf("invalid 'endTime' format (use RFC3339): %w", err)
		}
		params.EndTime = t
	}

	// Validate time range
	if !params.StartTime.IsZero() && !params.EndTime.IsZero() && params.StartTime.After(params.EndTime) {
		return nil, fmt.Errorf("startTime must be before endTime")
	}

	// Parse pagination
	if page := query.Get("page"); page != "" {
		val, err := strconv.Atoi(page)
		if err != nil || val < 1 {
			return nil, fmt.Errorf("invalid 'page' parameter: must be positive integer, got %q", page)
		}
		params.Page = val
	}
	if limit := query.Get("limit"); limit != "" {
		val, err := strconv.Atoi(limit)
		if err != nil || val < 1 {
			return nil, fmt.Errorf("invalid 'limit' parameter: must be positive integer, got %q", limit)
		}
		if val > MaxAlertsPerPage {
			return nil, fmt.Errorf("limit exceeds maximum allowed (%d > %d)", val, MaxAlertsPerPage)
		}
		params.Limit = val
	}

	// Parse sorting
	if sort := query.Get("sort"); sort != "" {
		sortBy, sortOrder, err := parseSortParameter(sort)
		if err != nil {
			return nil, fmt.Errorf("invalid 'sort' parameter: %w", err)
		}
		params.SortBy = sortBy
		params.SortOrder = sortOrder
	}

	return params, nil
}

// parseBool parses a boolean query parameter.
//
// Supports multiple formats:
//   - "true", "1", "yes", "on" → true
//   - "false", "0", "no", "off" → false
//
// Parameters:
//   - s: String value to parse
//
// Returns:
//   - bool: Parsed boolean value
//   - error: Parse error
func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q (use true/false, 1/0, yes/no, on/off)", s)
	}
}

// parseSortParameter parses the sort query parameter.
//
// Expected format: "field:direction"
// Examples:
//   - "startsAt:desc"
//   - "severity:asc"
//   - "alertname:desc"
//
// Supported fields:
//   - startsAt (default)
//   - endsAt
//   - severity
//   - alertname
//   - fingerprint
//
// Supported directions:
//   - asc (ascending)
//   - desc (descending, default)
//
// Parameters:
//   - s: Sort parameter string
//
// Returns:
//   - sortBy: Sort field
//   - sortOrder: Sort direction ("asc" or "desc")
//   - error: Parse error
func parseSortParameter(s string) (string, string, error) {
	parts := strings.Split(s, ":")
	if len(parts) == 0 || len(parts) > 2 {
		return "", "", fmt.Errorf("invalid format (use 'field:direction')")
	}

	sortBy := strings.TrimSpace(parts[0])
	sortOrder := "desc" // default

	if len(parts) == 2 {
		sortOrder = strings.ToLower(strings.TrimSpace(parts[1]))
	}

	// Validate sort field
	validSortFields := map[string]bool{
		"startsAt":    true,
		"endsAt":      true,
		"severity":    true,
		"alertname":   true,
		"fingerprint": true,
		"status":      true,
	}
	if !validSortFields[sortBy] {
		return "", "", fmt.Errorf("invalid sort field %q (use: startsAt, endsAt, severity, alertname, fingerprint, status)", sortBy)
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		return "", "", fmt.Errorf("invalid sort order %q (use: asc, desc)", sortOrder)
	}

	return sortBy, sortOrder, nil
}

// ParseLabelMatchers parses a label matcher expression.
//
// Supports Prometheus label matcher syntax:
//   {name="value"}                  - Single matcher
//   {name1="value1",name2="value2"} - Multiple matchers
//   {name=~"regex.*"}               - Regex matcher
//   {name!="value"}                 - Negative matcher
//   {name!~"regex.*"}               - Negative regex matcher
//
// Examples:
//   {alertname="HighCPU"}
//   {alertname="HighCPU",severity="critical"}
//   {alertname=~"High.*",severity!="info"}
//
// Parameters:
//   - expr: Label matcher expression
//
// Returns:
//   - []LabelMatcher: Parsed label matchers
//   - error: Parse error
func ParseLabelMatchers(expr string) ([]LabelMatcher, error) {
	if expr == "" {
		return nil, nil
	}

	// Trim whitespace
	expr = strings.TrimSpace(expr)

	// Check for curly braces
	if !strings.HasPrefix(expr, "{") || !strings.HasSuffix(expr, "}") {
		return nil, fmt.Errorf("expression must be enclosed in curly braces: {name=\"value\"}")
	}

	// Remove curly braces
	expr = strings.TrimPrefix(expr, "{")
	expr = strings.TrimSuffix(expr, "}")
	expr = strings.TrimSpace(expr)

	if expr == "" {
		return nil, nil // Empty matcher {} is valid (matches all)
	}

	// Split by comma (not inside quotes)
	matcherStrs := splitMatcherExpression(expr)

	matchers := make([]LabelMatcher, 0, len(matcherStrs))
	for _, matcherStr := range matcherStrs {
		matcher, err := parseSingleMatcher(matcherStr)
		if err != nil {
			return nil, fmt.Errorf("invalid matcher %q: %w", matcherStr, err)
		}
		matchers = append(matchers, matcher)
	}

	return matchers, nil
}

// splitMatcherExpression splits a matcher expression by commas.
//
// This handles commas inside quoted values correctly.
//
// Example:
//   alertname="HighCPU",severity="critical"
//   → ["alertname=\"HighCPU\"", "severity=\"critical\""]
//
// Parameters:
//   - expr: Matcher expression (without outer braces)
//
// Returns:
//   - []string: Individual matcher strings
func splitMatcherExpression(expr string) []string {
	var matchers []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		switch ch {
		case '"':
			inQuotes = !inQuotes
			current.WriteByte(ch)
		case ',':
			if inQuotes {
				current.WriteByte(ch)
			} else {
				if current.Len() > 0 {
					matchers = append(matchers, strings.TrimSpace(current.String()))
					current.Reset()
				}
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		matchers = append(matchers, strings.TrimSpace(current.String()))
	}

	return matchers
}

// parseSingleMatcher parses a single label matcher.
//
// Supported formats:
//   name="value"    - Exact match
//   name!="value"   - Not equal
//   name=~"regex"   - Regex match
//   name!~"regex"   - Negative regex match
//
// Parameters:
//   - s: Matcher string
//
// Returns:
//   - LabelMatcher: Parsed matcher
//   - error: Parse error
func parseSingleMatcher(s string) (LabelMatcher, error) {
	// Regex to parse label matcher: name<operator>"value"
	// Operators: =, !=, =~, !~
	re := regexp.MustCompile(`^(\w+)\s*(=~|!~|!=|=)\s*"([^"]*)"$`)
	matches := re.FindStringSubmatch(s)

	if matches == nil {
		return LabelMatcher{}, fmt.Errorf("invalid format (use: name=\"value\", name=~\"regex\", name!=\"value\", name!~\"regex\")")
	}

	name := matches[1]
	operator := matches[2]
	value := matches[3]

	// Validate regex for =~ and !~ operators
	if operator == "=~" || operator == "!~" {
		if _, err := regexp.Compile(value); err != nil {
			return LabelMatcher{}, fmt.Errorf("invalid regex pattern %q: %w", value, err)
		}
	}

	return LabelMatcher{
		Name:     name,
		Operator: operator,
		Value:    value,
	}, nil
}

// ValidateQueryParameters validates parsed query parameters.
//
// Performs validation checks:
//   - Page must be positive
//   - Limit must be 1 <= limit <= MaxAlertsPerPage
//   - Time range must be valid (start < end)
//   - Sort field must be valid
//   - Label matchers (if present) must be valid
//
// Parameters:
//   - params: Parsed query parameters
//
// Returns:
//   - *QueryValidationResult: Validation result
func ValidateQueryParameters(params *QueryParameters) *QueryValidationResult {
	var errors []ValidationError

	// Validate pagination
	if params.Page < 1 {
		errors = append(errors, ValidationError{
			Parameter: "page",
			Message:   "must be a positive integer",
			Value:     params.Page,
		})
	}
	if params.Limit < 1 || params.Limit > MaxAlertsPerPage {
		errors = append(errors, ValidationError{
			Parameter: "limit",
			Message:   fmt.Sprintf("must be between 1 and %d", MaxAlertsPerPage),
			Value:     params.Limit,
		})
	}

	// Validate time range
	if !params.StartTime.IsZero() && !params.EndTime.IsZero() {
		if params.StartTime.After(params.EndTime) {
			errors = append(errors, ValidationError{
				Parameter: "time_range",
				Message:   "startTime must be before endTime",
				Value:     fmt.Sprintf("%s > %s", params.StartTime, params.EndTime),
			})
		}
	}

	// Validate label matchers (if present)
	if params.Filter != "" {
		if _, err := ParseLabelMatchers(params.Filter); err != nil {
			errors = append(errors, ValidationError{
				Parameter: "filter",
				Message:   fmt.Sprintf("invalid label matcher expression: %v", err),
				Value:     params.Filter,
			})
		}
	}

	return &QueryValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
