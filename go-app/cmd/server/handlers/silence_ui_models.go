// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"strings"
	"time"

	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// ============================================================================
// Template Data Models
// ============================================================================

// DashboardData contains data для dashboard page.
type DashboardData struct {
	Silences   []*coresilencing.Silence `json:"silences"`
	Total      int                      `json:"total"`
	Filters    FilterParams             `json:"filters"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
	CSRF       string                   `json:"csrf"`
}

// FilterParams contains filter parameters для silences.
type FilterParams struct {
	Status       string    `json:"status"`        // active, pending, expired, all
	Creator      string    `json:"creator"`       // email filter
	Matcher      string    `json:"matcher"`       // matcher filter (name=value format)
	StartsAfter  time.Time `json:"starts_after"`  // time range filter
	StartsBefore time.Time `json:"starts_before"` // time range filter
	EndsAfter    time.Time `json:"ends_after"`    // time range filter
	EndsBefore   time.Time `json:"ends_before"`   // time range filter
	Limit        int       `json:"limit"`         // pagination limit
	Offset       int       `json:"offset"`        // pagination offset
	SortBy       string    `json:"sort_by"`       // sort field (created_at, starts_at, ends_at)
	SortOrder    string    `json:"sort_order"`    // sort order (asc, desc)
}

// CreateFormData contains data для create silence form.
type CreateFormData struct {
	CSRF        string         `json:"csrf"`
	Matchers    []MatcherInput `json:"matchers"` // Initial matchers (empty for new form)
	TimePresets []TimePreset   `json:"time_presets"`
	Error       string         `json:"error"` // Validation error message
	ErrorFields map[string]string `json:"error_fields"` // Field-specific errors
}

// EditFormData contains data для edit silence form.
type EditFormData struct {
	Silence     *coresilencing.Silence `json:"silence"`
	CSRF        string                 `json:"csrf"`
	TimePresets []TimePreset           `json:"time_presets"`
	Error       string                 `json:"error"`
	ErrorFields map[string]string      `json:"error_fields"`
}

// DetailViewData contains data для silence detail view.
type DetailViewData struct {
	Silence       *coresilencing.Silence `json:"silence"`
	MatchedCount  int                    `json:"matched_count"`  // Number of currently silenced alerts
	CSRF          string                 `json:"csrf"`
	RefreshRate   int                    `json:"refresh_rate"`   // Auto-refresh interval (seconds)
}

// TemplatesData contains data для templates page.
type TemplatesData struct {
	Templates []*SilenceTemplate `json:"templates"`
	CSRF      string             `json:"csrf"`
}

// AnalyticsData contains data для analytics dashboard.
type AnalyticsData struct {
	Stats        *SilenceStats `json:"stats"`
	Timeline     []TimelinePoint `json:"timeline"`      // Data points для timeline chart
	TopCreators  []CreatorStat   `json:"top_creators"`  // Top silence creators
	TopSilenced  []AlertStat     `json:"top_silenced"`  // Most silenced alerts
	TimeRange    string          `json:"time_range"`    // Selected time range
	RefreshRate  time.Duration   `json:"refresh_rate"`  // Auto-refresh interval
}

// ErrorData contains data для error page.
type ErrorData struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	RequestID  string `json:"request_id"`
	BackURL    string `json:"back_url"` // URL to navigate back
}

// ============================================================================
// Helper Data Structures
// ============================================================================

// MatcherInput represents a matcher input field in forms.
type MatcherInput struct {
	Name     string `json:"name"`
	Operator string `json:"operator"` // =, !=, =~, !~
	Value    string `json:"value"`
	IsRegex  bool   `json:"is_regex"`
}

// TimePreset represents a quick time preset button.
type TimePreset struct {
	Label    string        `json:"label"`    // Display label (e.g., "1 hour")
	Duration time.Duration `json:"duration"` // Duration value
}

// SilenceTemplate represents a pre-defined silence template.
type SilenceTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Matchers    []MatcherInput `json:"matchers"`
	Duration    time.Duration  `json:"duration"`
	Icon        string         `json:"icon"` // Emoji icon
}

// SilenceStats represents aggregated statistics.
type SilenceStats struct {
	Total        int     `json:"total"`         // Total silences (all time)
	Active       int     `json:"active"`        // Currently active
	Pending      int     `json:"pending"`       // Pending (future)
	Expired      int     `json:"expired"`       // Expired
	Expired24h   int     `json:"expired_24h"`   // Expired in last 24h
	AvgDuration  float64 `json:"avg_duration"`  // Average duration (hours)
	TotalMatchers int    `json:"total_matchers"` // Total matchers count
}

// TimelinePoint represents a data point in timeline chart.
type TimelinePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Pending   int       `json:"pending"`
	Active    int       `json:"active"`
	Expired   int       `json:"expired"`
}

// CreatorStat represents statistics per creator.
type CreatorStat struct {
	Creator     string    `json:"creator"`
	Count       int       `json:"count"`
	LastCreated time.Time `json:"last_created"`
}

// AlertStat represents statistics per alert.
type AlertStat struct {
	AlertName     string        `json:"alert_name"`
	TimesSilenced int           `json:"times_silenced"`
	TotalDuration time.Duration `json:"total_duration"`
}

// ============================================================================
// Built-In Templates
// ============================================================================

var builtInTemplates = []*SilenceTemplate{
	{
		ID:          "maintenance-window",
		Name:        "Maintenance Window",
		Description: "Silence all alerts during scheduled maintenance",
		Icon:        "🔧",
		Matchers: []MatcherInput{
			{Name: "type", Operator: "=", Value: "maintenance", IsRegex: false},
		},
		Duration: 2 * time.Hour,
	},
	{
		ID:          "oncall-handoff",
		Name:        "On-Call Handoff",
		Description: "Silence critical pages during on-call handoff",
		Icon:        "📞",
		Matchers: []MatcherInput{
			{Name: "alertname", Operator: "=", Value: "OnCallPageCritical", IsRegex: false},
		},
		Duration: 1 * time.Hour,
	},
	{
		ID:          "incident-response",
		Name:        "Incident Response",
		Description: "Silence critical alerts during active incident",
		Icon:        "🚨",
		Matchers: []MatcherInput{
			{Name: "severity", Operator: "=", Value: "critical", IsRegex: false},
			{Name: "incident", Operator: "=~", Value: "INC-.*", IsRegex: true},
		},
		Duration: 4 * time.Hour,
	},
}

// GetBuiltInTemplates returns all built-in silence templates.
func GetBuiltInTemplates() []*SilenceTemplate {
	return builtInTemplates
}

// GetTemplateByID returns a built-in template by ID.
func GetTemplateByID(id string) *SilenceTemplate {
	for _, tmpl := range builtInTemplates {
		if tmpl.ID == id {
			return tmpl
		}
	}
	return nil
}

// ============================================================================
// Default Values
// ============================================================================

var defaultTimePresets = []TimePreset{
	{Label: "1 hour", Duration: 1 * time.Hour},
	{Label: "4 hours", Duration: 4 * time.Hour},
	{Label: "8 hours", Duration: 8 * time.Hour},
	{Label: "24 hours", Duration: 24 * time.Hour},
	{Label: "7 days", Duration: 7 * 24 * time.Hour},
}

// GetDefaultTimePresets returns default time presets.
func GetDefaultTimePresets() []TimePreset {
	return defaultTimePresets
}

// ============================================================================
// Helper Functions
// ============================================================================

// NewFilterParams creates FilterParams with default values.
func NewFilterParams() FilterParams {
	return FilterParams{
		Status:    "all",
		Limit:     25,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// CalculateTotalPages calculates total pages based on total items and page size.
func CalculateTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

// GetCurrentPage calculates current page number from offset and limit.
func GetCurrentPage(offset, limit int) int {
	if limit <= 0 {
		return 0
	}
	return offset / limit
}

// ============================================================================
// Validation
// ============================================================================

// ValidateFilterParams validates filter parameters.
func (f *FilterParams) Validate() error {
	// Validate status
	validStatuses := []string{"all", "active", "pending", "expired"}
	validStatus := false
	for _, s := range validStatuses {
		if f.Status == s {
			validStatus = true
			break
		}
	}
	if !validStatus {
		f.Status = "all"
	}

	// Validate limit (max 1000)
	if f.Limit <= 0 {
		f.Limit = 25
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}

	// Validate offset (non-negative)
	if f.Offset < 0 {
		f.Offset = 0
	}

	// Validate sort_by
	validSortFields := []string{"created_at", "starts_at", "ends_at", "status", "creator"}
	validSort := false
	for _, s := range validSortFields {
		if f.SortBy == s {
			validSort = true
			break
		}
	}
	if !validSort {
		f.SortBy = "created_at"
	}

	// Validate sort_order
	if f.SortOrder != "asc" && f.SortOrder != "desc" {
		f.SortOrder = "desc"
	}

	return nil
}

// ToRepositoryFilters converts FilterParams to repository filter format.
// This method будет использоваться для передачи фильтров в SilenceRepository.
func (f *FilterParams) ToSilenceFilter() infrasilencing.SilenceFilter {
	filter := infrasilencing.SilenceFilter{
		Limit:  f.Limit,
		Offset: f.Offset,
	}

	// Status filter
	if f.Status != "all" && f.Status != "" {
		// Parse status string to SilenceStatus enum
		switch f.Status {
		case "active":
			filter.Statuses = []coresilencing.SilenceStatus{coresilencing.SilenceStatusActive}
		case "pending":
			filter.Statuses = []coresilencing.SilenceStatus{coresilencing.SilenceStatusPending}
		case "expired":
			filter.Statuses = []coresilencing.SilenceStatus{coresilencing.SilenceStatusExpired}
		}
	}

	// Creator filter
	if f.Creator != "" {
		filter.CreatedBy = f.Creator
	}

	// Matcher filter (parse "name=value" format)
	if f.Matcher != "" {
		// Simple parsing for UI matcher filter
		// Format: "alertname=HighCPU" or just "alertname"
		parts := strings.SplitN(f.Matcher, "=", 2)
		if len(parts) > 0 {
			filter.MatcherName = parts[0]
			if len(parts) == 2 {
				filter.MatcherValue = parts[1]
			}
		}
	}

	// Time range filters
	if !f.StartsAfter.IsZero() {
		filter.StartsAfter = &f.StartsAfter
	}
	if !f.StartsBefore.IsZero() {
		filter.StartsBefore = &f.StartsBefore
	}
	if !f.EndsAfter.IsZero() {
		filter.EndsAfter = &f.EndsAfter
	}
	if !f.EndsBefore.IsZero() {
		filter.EndsBefore = &f.EndsBefore
	}

	// Note: Sorting is not part of SilenceFilter, handled at application level

	return filter
}
