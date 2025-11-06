// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// templateFuncs returns a FuncMap with custom template functions.
// These functions are available in all Go html/template files.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// Time formatting
		"formatTime":     formatTime,
		"formatDateTime": formatDateTime,
		"formatDate":     formatDate,
		"humanDuration":  humanDuration,
		"timeAgo":        timeAgo,

		// String manipulation
		"truncate":    truncate,
		"truncateEnd": truncateEnd,
		"upper":       strings.ToUpper,
		"lower":       strings.ToLower,
		"title":       strings.Title,
		"contains":    contains,
		"join":        strings.Join,

		// Status helpers
		"statusBadge":  statusBadge,
		"statusClass":  statusClass,
		"statusIcon":   statusIcon,
		"severityBadge": severityBadge,

		// Math helpers
		"add":      add,
		"sub":      sub,
		"mul":      mul,
		"div":      div,
		"min":      min,
		"max":      max,
		"percent":  percent,

		// HTML helpers
		"safeHTML": safeHTML,
		"attr":     attr,

		// Comparison helpers
		"eq":  eq,
		"ne":  ne,
		"lt":  lt,
		"le":  le,
		"gt":  gt,
		"ge":  ge,
		"and": and,
		"or":  or,
		"not": not,

		// Collection helpers
		"len":   length,
		"first": first,
		"last":  last,
		"slice": sliceHelper,
	}
}

// ============================================================================
// Time Functions
// ============================================================================

// formatTime formats a time.Time to "2006-01-02 15:04" format.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// formatDateTime formats a time.Time to "2006-01-02 15:04:05" format.
func formatDateTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatDate formats a time.Time to "2006-01-02" format.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}

// humanDuration converts a duration to human-readable format.
// Examples: "5s", "2m", "3h", "2d"
func humanDuration(d time.Duration) string {
	if d < 0 {
		return "-"
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// timeAgo returns a human-readable "time ago" string.
// Examples: "just now", "5 minutes ago", "2 hours ago", "3 days ago"
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// ============================================================================
// String Functions
// ============================================================================

// truncate truncates a string to the specified length and adds "..." if truncated.
func truncate(s string, length int) string {
	if length <= 0 {
		return ""
	}
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// truncateEnd truncates a string from the end to the specified length.
func truncateEnd(s string, length int) string {
	if length <= 0 {
		return ""
	}
	if len(s) <= length {
		return s
	}
	return "..." + s[len(s)-length:]
}

// contains checks if a slice contains an item.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ============================================================================
// Status Functions
// ============================================================================

// statusBadge returns an HTML badge for a status.
func statusBadge(status string) template.HTML {
	class := statusClass(status)
	icon := statusIcon(status)
	return template.HTML(fmt.Sprintf(
		`<span class="badge badge-%s">%s %s</span>`,
		class, icon, strings.Title(status),
	))
}

// statusClass returns a CSS class for a status.
func statusClass(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return "success"
	case "pending":
		return "info"
	case "expired":
		return "secondary"
	case "error", "failed":
		return "danger"
	case "warning":
		return "warning"
	default:
		return "secondary"
	}
}

// statusIcon returns an emoji icon for a status.
func statusIcon(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return "✓"
	case "pending":
		return "⏳"
	case "expired":
		return "⏹"
	case "error", "failed":
		return "✗"
	case "warning":
		return "⚠"
	default:
		return "●"
	}
}

// severityBadge returns an HTML badge for an alert severity.
func severityBadge(severity string) template.HTML {
	var class string
	switch strings.ToLower(severity) {
	case "critical":
		class = "danger"
	case "warning":
		class = "warning"
	case "info":
		class = "info"
	default:
		class = "secondary"
	}
	return template.HTML(fmt.Sprintf(
		`<span class="badge badge-%s">%s</span>`,
		class, strings.ToUpper(severity),
	))
}

// ============================================================================
// Math Functions
// ============================================================================

// add adds two integers.
func add(a, b int) int {
	return a + b
}

// sub subtracts b from a.
func sub(a, b int) int {
	return a - b
}

// mul multiplies two integers.
func mul(a, b int) int {
	return a * b
}

// div divides a by b (returns 0 if b is 0).
func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// percent calculates percentage (a/b * 100).
func percent(a, b int) int {
	if b == 0 {
		return 0
	}
	return (a * 100) / b
}

// ============================================================================
// HTML Functions
// ============================================================================

// safeHTML marks a string as safe HTML (no escaping).
// WARNING: Only use with trusted content to prevent XSS.
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// attr formats an HTML attribute.
func attr(name, value string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`%s="%s"`, name, template.HTMLEscapeString(value)))
}

// ============================================================================
// Comparison Functions
// ============================================================================

// eq checks if two values are equal.
func eq(a, b interface{}) bool {
	return a == b
}

// ne checks if two values are not equal.
func ne(a, b interface{}) bool {
	return a != b
}

// lt checks if a < b.
func lt(a, b int) bool {
	return a < b
}

// le checks if a <= b.
func le(a, b int) bool {
	return a <= b
}

// gt checks if a > b.
func gt(a, b int) bool {
	return a > b
}

// ge checks if a >= b.
func ge(a, b int) bool {
	return a >= b
}

// and returns true if both a and b are true.
func and(a, b bool) bool {
	return a && b
}

// or returns true if either a or b is true.
func or(a, b bool) bool {
	return a || b
}

// not returns the negation of a.
func not(a bool) bool {
	return !a
}

// ============================================================================
// Collection Functions
// ============================================================================

// length returns the length of a collection.
func length(v interface{}) int {
	switch val := v.(type) {
	case []interface{}:
		return len(val)
	case []string:
		return len(val)
	case string:
		return len(val)
	default:
		return 0
	}
}

// first returns the first element of a slice (or nil if empty).
func first(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[0]
		}
	case []string:
		if len(val) > 0 {
			return val[0]
		}
	}
	return nil
}

// last returns the last element of a slice (or nil if empty).
func last(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[len(val)-1]
		}
	case []string:
		if len(val) > 0 {
			return val[len(val)-1]
		}
	}
	return nil
}

// sliceHelper returns a slice of a slice (similar to Python slice).
func sliceHelper(start, end int, v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if start < 0 {
			start = 0
		}
		if end > len(val) {
			end = len(val)
		}
		if start >= end {
			return []interface{}{}
		}
		return val[start:end]
	case []string:
		if start < 0 {
			start = 0
		}
		if end > len(val) {
			end = len(val)
		}
		if start >= end {
			return []string{}
		}
		return val[start:end]
	}
	return nil
}
