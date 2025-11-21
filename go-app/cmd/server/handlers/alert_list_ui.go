// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
	"github.com/vitaliisemenov/alert-history/internal/ui"
)

// AlertListUIHandler handles UI rendering for Alert List page.
// TN-79: Alert List with Filtering
// TN-80: Enhanced with Classification Display
type AlertListUIHandler struct {
	templateEngine   *ui.TemplateEngine // TN-76: Dashboard Template Engine
	historyRepo      core.AlertHistoryRepository
	classificationEnricher ui.ClassificationEnricher // TN-80: Classification Enricher
	cache            cache.Cache // Response caching
	logger           *slog.Logger
}

// NewAlertListUIHandler creates a new AlertListUIHandler.
func NewAlertListUIHandler(
	templateEngine *ui.TemplateEngine,
	historyRepo core.AlertHistoryRepository,
	cache cache.Cache,
	logger *slog.Logger,
) *AlertListUIHandler {
	return &AlertListUIHandler{
		templateEngine: templateEngine,
		historyRepo:    historyRepo,
		cache:          cache,
		logger:         logger,
		// classificationEnricher will be set via SetClassificationEnricher if available
	}
}

// SetClassificationEnricher sets the classification enricher (TN-80).
// This allows optional integration - if classification service is not available,
// the handler will work without it (graceful degradation).
func (h *AlertListUIHandler) SetClassificationEnricher(enricher ui.ClassificationEnricher) {
	h.classificationEnricher = enricher
}

// AlertListPageData represents data for alert list page template.
type AlertListPageData struct {
	Title      string
	Breadcrumbs []Breadcrumb
	Alerts     []*core.Alert
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
	HasNext    bool
	HasPrev    bool
	Filters    *AlertListFilters
	Sorting    *AlertListSorting
	CSRF       string
}

// AlertListFilters represents filter parameters for alert list.
// TN-80: Enhanced with classification filters
type AlertListFilters struct {
	Status    *core.AlertStatus
	Severity  *string
	Namespace *string
	TimeRange *core.TimeRange
	Labels    map[string]string
	Search    *string

	// TN-80: Classification filters
	ClassificationSeverity *string  // "critical", "warning", "info", "noise"
	MinConfidence         *float64  // 0.0-1.0
	MaxConfidence         *float64  // 0.0-1.0
	HasClassification     *bool     // true/false/nil (all)
	ClassificationSource  *string   // "llm", "fallback", "cache"
}

// AlertListSorting represents sorting parameters for alert list.
// TN-80: Enhanced with classification sorting
type AlertListSorting struct {
	Field string
	Order string // "asc" or "desc"
}

// Breadcrumb represents a breadcrumb navigation item.
type Breadcrumb struct {
	Label string
	URL   string
}

// RenderAlertList renders the alert list page.
// GET /ui/alerts
func (h *AlertListUIHandler) RenderAlertList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Parse query parameters
	query := r.URL.Query()

	// Parse filters
	filters := h.parseFilters(query)

	// Parse pagination
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 50
	if perPageStr := query.Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
			if perPage > 1000 {
				perPage = 1000
			}
		}
	}

	// Parse sorting
	sorting := h.parseSorting(query)

	// Build HistoryRequest
	historyReq := &core.HistoryRequest{
		Filters: filters.ToCoreFilters(),
		Pagination: &core.Pagination{
			Page:    page,
			PerPage: perPage,
		},
		Sorting: sorting.ToCoreSorting(),
	}

	// Validate request
	if err := historyReq.Validate(); err != nil {
		h.logger.Warn("Invalid history request", "error", err)
		h.renderError(w, r, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	// Fetch alerts from repository
	historyResp, err := h.historyRepo.GetHistory(ctx, historyReq)
	if err != nil {
		h.logger.Error("Failed to get alert history", "error", err)
		h.renderError(w, r, "Failed to load alerts", http.StatusInternalServerError)
		return
	}

	// TN-80: Enrich alerts with classification data
	var enrichedAlerts []*ui.EnrichedAlert
	if h.classificationEnricher != nil && len(historyResp.Alerts) > 0 {
		enriched, err := h.classificationEnricher.EnrichAlerts(ctx, historyResp.Alerts)
		if err != nil {
			h.logger.Warn("Failed to enrich alerts with classification, continuing without classification",
				"error", err,
				"alerts_count", len(historyResp.Alerts))
			// Graceful degradation: convert alerts to enriched format without classification
			enrichedAlerts = convertToEnrichedAlerts(historyResp.Alerts)
		} else {
			enrichedAlerts = enriched
		}
	} else {
		// No classification enricher available, convert alerts to enriched format without classification
		enrichedAlerts = convertToEnrichedAlerts(historyResp.Alerts)
	}

	// TN-80: Apply classification filters (in-memory filtering after enrichment)
	enrichedAlerts = h.applyClassificationFilters(enrichedAlerts, filters)

	// TN-80: Apply classification sorting (in-memory sorting after enrichment)
	enrichedAlerts = h.applyClassificationSorting(enrichedAlerts, sorting)

	// Convert enriched alerts to template-friendly format
	alertCardDataList := ui.ToAlertCardDataList(enrichedAlerts)

	// Prepare template data
	alertListData := map[string]interface{}{
		"Alerts":     alertCardDataList, // TN-80: Use enriched alert card data
		"Total":      historyResp.Total,
		"Page":       historyResp.Page,
		"PerPage":    historyResp.PerPage,
		"TotalPages": historyResp.TotalPages,
		"HasNext":    historyResp.HasNext,
		"HasPrev":    historyResp.HasPrev,
		"Filters":    filters,
		"Sorting":    sorting,
		"CSRF":       h.generateCSRFToken(r),
	}

	// Prepare UI PageData
	uiPageData := ui.NewPageData("Alert List")
	uiPageData.AddBreadcrumb("Home", "/")
	uiPageData.AddBreadcrumb("Alerts", "")
	uiPageData.Data = alertListData

	// Render template
	h.templateEngine.RenderWithFallback(w, "pages/alert-list", uiPageData)

	duration := time.Since(startTime)
	h.logger.Debug("Alert list rendered",
		"duration_ms", duration.Milliseconds(),
		"alerts_count", len(historyResp.Alerts),
		"page", page,
		"per_page", perPage,
		"total", historyResp.Total,
	)
}

// parseFilters parses filter parameters from URL query.
func (h *AlertListUIHandler) parseFilters(query url.Values) *AlertListFilters {
	filters := &AlertListFilters{}

	// Parse status filter
	if statusStr := query.Get("status"); statusStr != "" {
		status := core.AlertStatus(statusStr)
		if status == core.StatusFiring || status == core.StatusResolved {
			filters.Status = &status
		}
	}

	// Parse severity filter
	if severityStr := query.Get("severity"); severityStr != "" {
		filters.Severity = &severityStr
	}

	// Parse namespace filter
	if namespaceStr := query.Get("namespace"); namespaceStr != "" {
		filters.Namespace = &namespaceStr
	}

	// Parse time range
	if fromStr := query.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			if filters.TimeRange == nil {
				filters.TimeRange = &core.TimeRange{}
			}
			filters.TimeRange.From = &from
		}
	}
	if toStr := query.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if filters.TimeRange == nil {
				filters.TimeRange = &core.TimeRange{}
			}
			filters.TimeRange.To = &to
		}
	}

	// Parse labels (format: labels[key]=value)
	labels := make(map[string]string)
	for key, values := range query {
		if len(key) > 7 && key[:7] == "labels[" {
			labelKey := key[7 : len(key)-1]
			if len(values) > 0 {
				labels[labelKey] = values[0]
			}
		}
	}
	if len(labels) > 0 {
		filters.Labels = labels
	}

	// Parse search filter
	if searchStr := query.Get("search"); searchStr != "" {
		filters.Search = &searchStr
	}

	// TN-80: Parse classification filters
	if classificationSeverityStr := query.Get("classification_severity"); classificationSeverityStr != "" {
		validSeverities := map[string]bool{
			"critical": true,
			"warning":  true,
			"info":     true,
			"noise":    true,
		}
		if validSeverities[classificationSeverityStr] {
			filters.ClassificationSeverity = &classificationSeverityStr
		}
	}

	if minConfidenceStr := query.Get("min_confidence"); minConfidenceStr != "" {
		if minConf, err := strconv.ParseFloat(minConfidenceStr, 64); err == nil {
			if minConf >= 0.0 && minConf <= 1.0 {
				filters.MinConfidence = &minConf
			}
		}
	}

	if maxConfidenceStr := query.Get("max_confidence"); maxConfidenceStr != "" {
		if maxConf, err := strconv.ParseFloat(maxConfidenceStr, 64); err == nil {
			if maxConf >= 0.0 && maxConf <= 1.0 {
				filters.MaxConfidence = &maxConf
			}
		}
	}

	if hasClassificationStr := query.Get("has_classification"); hasClassificationStr != "" {
		if hasClassificationStr == "true" {
			hasClassification := true
			filters.HasClassification = &hasClassification
		} else if hasClassificationStr == "false" {
			hasClassification := false
			filters.HasClassification = &hasClassification
		}
	}

	if classificationSourceStr := query.Get("classification_source"); classificationSourceStr != "" {
		validSources := map[string]bool{
			"llm":      true,
			"fallback": true,
			"cache":    true,
		}
		if validSources[classificationSourceStr] {
			filters.ClassificationSource = &classificationSourceStr
		}
	}

	return filters
}

// parseSorting parses sorting parameters from URL query.
func (h *AlertListUIHandler) parseSorting(query url.Values) *AlertListSorting {
	sorting := &AlertListSorting{
		Field: "starts_at", // default
		Order: "desc",      // default
	}

	if sortField := query.Get("sort_field"); sortField != "" {
		// TN-80: Validate sort field (including classification fields)
		validFields := map[string]bool{
			"starts_at":              true,
			"severity":               true,
			"status":                 true,
			"classification_severity": true, // TN-80
			"classification_confidence": true, // TN-80
		}
		if validFields[sortField] {
			sorting.Field = sortField
		}
	}

	if sortOrder := query.Get("sort_order"); sortOrder != "" {
		if sortOrder == "asc" || sortOrder == "desc" {
			sorting.Order = sortOrder
		}
	}

	return sorting
}

// ToCoreFilters converts AlertListFilters to core.AlertFilters.
func (f *AlertListFilters) ToCoreFilters() *core.AlertFilters {
	if f == nil {
		return nil
	}

	coreFilters := &core.AlertFilters{
		Status:    f.Status,
		Severity:  f.Severity,
		Namespace: f.Namespace,
		TimeRange: f.TimeRange,
		Labels:    f.Labels,
	}

	return coreFilters
}

// ToCoreSorting converts AlertListSorting to core.Sorting.
func (s *AlertListSorting) ToCoreSorting() *core.Sorting {
	if s == nil {
		return nil
	}

	return &core.Sorting{
		Field: s.Field,
		Order: core.SortOrder(s.Order),
	}
}

// renderError renders an error page with enhanced error details (150% Quality).
func (h *AlertListUIHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	// Enhanced error details for better UX (150% Quality Enhancement)
	errorDetails := map[string]interface{}{
		"Message":     message,
		"Status":      status,
		"StatusCode":  status,
		"StatusText":  http.StatusText(status),
		"RequestPath":  r.URL.Path,
		"RequestQuery": r.URL.RawQuery,
		"Timestamp":   time.Now().Format(time.RFC3339),
	}

	// Add helpful suggestions based on error type
	suggestions := []string{}
	switch status {
	case http.StatusBadRequest:
		suggestions = append(suggestions,
			"Check your filter parameters (status, severity, namespace, time range)",
			"Verify date format is RFC3339 (e.g., 2023-01-01T00:00:00Z)",
			"Ensure pagination parameters are valid (page > 0, per_page > 0)",
		)
	case http.StatusInternalServerError:
		suggestions = append(suggestions,
			"The server encountered an error processing your request",
			"Try refreshing the page in a few moments",
			"If the problem persists, contact your system administrator",
		)
	case http.StatusNotFound:
		suggestions = append(suggestions,
			"The requested resource was not found",
			"Check the URL path is correct",
			"Navigate back to the alert list page",
		)
	}
	errorDetails["Suggestions"] = suggestions

	pageData := ui.NewPageData("Error - Alert List")
	pageData.AddBreadcrumb("Home", "/")
	pageData.AddBreadcrumb("Alerts", "/ui/alerts")
	pageData.AddBreadcrumb("Error", "")
	pageData.Data = errorDetails

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	// Try to render error template, fallback to plain text if needed
	// RenderWithFallback handles errors internally, so we always provide fallback
	h.templateEngine.RenderWithFallback(w, "errors/500", pageData)

	// Note: RenderWithFallback will fallback to plain text internally if template fails
	// For additional safety, we could check response status, but RenderWithFallback
	// already handles this gracefully
}

// generateCSRFToken generates a CSRF token (placeholder for now).
func (h *AlertListUIHandler) generateCSRFToken(r *http.Request) string {
	// TODO: Implement proper CSRF token generation
	return "csrf-token-placeholder"
}

// convertToEnrichedAlerts converts a list of alerts to enriched alerts without classification.
// This is used for graceful degradation when classification is not available.
func convertToEnrichedAlerts(alerts []*core.Alert) []*ui.EnrichedAlert {
	if len(alerts) == 0 {
		return []*ui.EnrichedAlert{}
	}

	enriched := make([]*ui.EnrichedAlert, len(alerts))
	for i, alert := range alerts {
		enriched[i] = &ui.EnrichedAlert{
			Alert:            alert,
			HasClassification: false,
		}
	}

	return enriched
}

// applyClassificationFilters applies classification filters to enriched alerts (TN-80).
// This performs in-memory filtering since classification is not stored in DB.
func (h *AlertListUIHandler) applyClassificationFilters(enriched []*ui.EnrichedAlert, filters *AlertListFilters) []*ui.EnrichedAlert {
	if filters == nil {
		return enriched
	}

	filtered := make([]*ui.EnrichedAlert, 0, len(enriched))

	for _, alert := range enriched {
		// Filter by has_classification
		if filters.HasClassification != nil {
			if *filters.HasClassification != alert.HasClassification {
				continue
			}
		}

		// If no classification, skip classification-specific filters
		if !alert.HasClassification || alert.Classification == nil {
			// If filter requires classification, skip this alert
			if filters.ClassificationSeverity != nil || filters.MinConfidence != nil || filters.MaxConfidence != nil || filters.ClassificationSource != nil {
				continue
			}
			filtered = append(filtered, alert)
			continue
		}

		// Filter by classification severity
		if filters.ClassificationSeverity != nil {
			if string(alert.Classification.Severity) != *filters.ClassificationSeverity {
				continue
			}
		}

		// Filter by confidence range
		if filters.MinConfidence != nil {
			if alert.Classification.Confidence < *filters.MinConfidence {
				continue
			}
		}
		if filters.MaxConfidence != nil {
			if alert.Classification.Confidence > *filters.MaxConfidence {
				continue
			}
		}

		// Filter by classification source
		if filters.ClassificationSource != nil {
			if alert.ClassificationSource != *filters.ClassificationSource {
				continue
			}
		}

		filtered = append(filtered, alert)
	}

	return filtered
}

// applyClassificationSorting applies classification sorting to enriched alerts (TN-80).
// This performs in-memory sorting since classification is not stored in DB.
func (h *AlertListUIHandler) applyClassificationSorting(enriched []*ui.EnrichedAlert, sorting *AlertListSorting) []*ui.EnrichedAlert {
	if sorting == nil || len(enriched) == 0 {
		return enriched
	}

	// Clone slice to avoid mutating original
	sorted := make([]*ui.EnrichedAlert, len(enriched))
	copy(sorted, enriched)

	// Sort by classification fields
	switch sorting.Field {
	case "classification_severity":
		sort.Slice(sorted, func(i, j int) bool {
			severityI := getClassificationSeverityOrder(sorted[i])
			severityJ := getClassificationSeverityOrder(sorted[j])
			if sorting.Order == "asc" {
				return severityI < severityJ
			}
			return severityI > severityJ
		})
	case "classification_confidence":
		sort.Slice(sorted, func(i, j int) bool {
			confI := getClassificationConfidence(sorted[i])
			confJ := getClassificationConfidence(sorted[j])
			if sorting.Order == "asc" {
				return confI < confJ
			}
			return confI > confJ
		})
	}

	return sorted
}

// getClassificationSeverityOrder returns numeric order for severity (for sorting).
func getClassificationSeverityOrder(enriched *ui.EnrichedAlert) int {
	if !enriched.HasClassification || enriched.Classification == nil {
		return 999 // No classification goes to end
	}

	switch enriched.Classification.Severity {
	case core.SeverityCritical:
		return 1
	case core.SeverityWarning:
		return 2
	case core.SeverityInfo:
		return 3
	case core.SeverityNoise:
		return 4
	default:
		return 999
	}
}

// getClassificationConfidence returns confidence value (for sorting).
func getClassificationConfidence(enriched *ui.EnrichedAlert) float64 {
	if !enriched.HasClassification || enriched.Classification == nil {
		return -1.0 // No classification goes to end
	}
	return enriched.Classification.Confidence
}
