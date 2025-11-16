// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// HistoryHandlerV2 handles history requests using AlertHistoryRepository
type HistoryHandlerV2 struct {
	repository core.AlertHistoryRepository
	logger     *slog.Logger
}

// NewHistoryHandlerV2 creates a new history handler
func NewHistoryHandlerV2(repository core.AlertHistoryRepository, logger *slog.Logger) *HistoryHandlerV2 {
	if logger == nil {
		logger = slog.Default()
	}

	return &HistoryHandlerV2{
		repository: repository,
		logger:     logger,
	}
}

// HandleHistory handles GET /history requests
func (h *HistoryHandlerV2) HandleHistory(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Log the history request
	h.logger.Info("History request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"query", r.URL.RawQuery,
	)

	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method for history", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Build HistoryRequest from query params
	req, err := h.parseHistoryRequest(query)
	if err != nil {
		h.logger.Error("Failed to parse history request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get history from repository
	ctx := r.Context()
	response, err := h.repository.GetHistory(ctx, req)
	if err != nil {
		h.logger.Error("Failed to get history", "error", err)
		http.Error(w, "Failed to retrieve alert history", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode history response", "error", err)
		return
	}

	processingTime := time.Since(startTime)
	h.logger.Info("History request completed",
		"page", response.Page,
		"per_page", response.PerPage,
		"total_alerts", response.Total,
		"returned_alerts", len(response.Alerts),
		"processing_time_ms", processingTime.Milliseconds(),
	)
}

// parseHistoryRequest parses query parameters into HistoryRequest
func (h *HistoryHandlerV2) parseHistoryRequest(query map[string][]string) (*core.HistoryRequest, error) {
	req := &core.HistoryRequest{
		Filters:    &core.AlertFilters{},
		Pagination: &core.Pagination{},
		Sorting:    nil,
	}

	// Parse pagination
	page := 1
	if pageStr := query["page"]; len(pageStr) > 0 {
		if p, err := strconv.Atoi(pageStr[0]); err == nil && p > 0 {
			page = p
		}
	}
	req.Pagination.Page = page

	perPage := 50
	if perPageStr := query["per_page"]; len(perPageStr) > 0 {
		if pp, err := strconv.Atoi(perPageStr[0]); err == nil && pp > 0 {
			perPage = pp
			if perPage > 1000 {
				perPage = 1000
			}
		}
	}
	req.Pagination.PerPage = perPage

	// Parse filters
	if statusStr := query["status"]; len(statusStr) > 0 {
		status := core.AlertStatus(statusStr[0])
		req.Filters.Status = &status
	}

	if severityStr := query["severity"]; len(severityStr) > 0 {
		req.Filters.Severity = &severityStr[0]
	}

	if namespaceStr := query["namespace"]; len(namespaceStr) > 0 {
		req.Filters.Namespace = &namespaceStr[0]
	}

	// Parse time range
	if fromStr := query["from"]; len(fromStr) > 0 {
		if from, err := time.Parse(time.RFC3339, fromStr[0]); err == nil {
			if req.Filters.TimeRange == nil {
				req.Filters.TimeRange = &core.TimeRange{}
			}
			req.Filters.TimeRange.From = &from
		}
	}

	if toStr := query["to"]; len(toStr) > 0 {
		if to, err := time.Parse(time.RFC3339, toStr[0]); err == nil {
			if req.Filters.TimeRange == nil {
				req.Filters.TimeRange = &core.TimeRange{}
			}
			req.Filters.TimeRange.To = &to
		}
	}

	// Parse sorting
	if sortField := query["sort_field"]; len(sortField) > 0 {
		req.Sorting = &core.Sorting{
			Field: sortField[0],
			Order: core.SortOrderDesc, // default desc
		}

		if sortOrder := query["sort_order"]; len(sortOrder) > 0 {
			req.Sorting.Order = core.SortOrder(sortOrder[0])
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return req, nil
}

// HandleRecentAlerts handles GET /history/recent
func (h *HistoryHandlerV2) HandleRecentAlerts(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	h.logger.Info("Recent alerts request received",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	// Get recent alerts
	ctx := r.Context()
	alerts, err := h.repository.GetRecentAlerts(ctx, limit)
	if err != nil {
		h.logger.Error("Failed to get recent alerts", "error", err)
		http.Error(w, "Failed to retrieve recent alerts", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"alerts":    alerts,
		"count":     len(alerts),
		"limit":     limit,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	h.logger.Info("Recent alerts completed",
		"count", len(alerts),
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)
}

// HandleStats handles GET /history/stats
func (h *HistoryHandlerV2) HandleStats(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	h.logger.Info("Stats request received",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse time range
	var timeRange *core.TimeRange
	query := r.URL.Query()

	if fromStr := query.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.From = &from
		}
	}

	if toStr := query.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.To = &to
		}
	}

	// Get aggregated stats
	ctx := r.Context()
	stats, err := h.repository.GetAggregatedStats(ctx, timeRange)
	if err != nil {
		h.logger.Error("Failed to get aggregated stats", "error", err)
		http.Error(w, "Failed to retrieve stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.Error("Failed to encode stats response", "error", err)
		return
	}

	h.logger.Info("Stats request completed",
		"total_alerts", stats.TotalAlerts,
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)
}

// HandleTopAlerts handles GET /history/top
func (h *HistoryHandlerV2) HandleTopAlerts(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	h.logger.Info("Top alerts request received",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse parameters
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse time range
	var timeRange *core.TimeRange
	query := r.URL.Query()

	if fromStr := query.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.From = &from
		}
	}

	if toStr := query.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.To = &to
		}
	}

	// Get top alerts
	ctx := r.Context()
	topAlerts, err := h.repository.GetTopAlerts(ctx, timeRange, limit)
	if err != nil {
		h.logger.Error("Failed to get top alerts", "error", err)
		http.Error(w, "Failed to retrieve top alerts", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"alerts":    topAlerts,
		"count":     len(topAlerts),
		"limit":     limit,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	h.logger.Info("Top alerts completed",
		"count", len(topAlerts),
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)
}

// HandleFlappingAlerts handles GET /history/flapping
func (h *HistoryHandlerV2) HandleFlappingAlerts(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	h.logger.Info("Flapping alerts request received",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse parameters
	threshold := 3
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if t, err := strconv.Atoi(thresholdStr); err == nil && t > 0 {
			threshold = t
		}
	}

	// Parse time range
	var timeRange *core.TimeRange
	query := r.URL.Query()

	if fromStr := query.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.From = &from
		}
	}

	if toStr := query.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.To = &to
		}
	}

	// Get flapping alerts
	ctx := r.Context()
	flappingAlerts, err := h.repository.GetFlappingAlerts(ctx, timeRange, threshold)
	if err != nil {
		h.logger.Error("Failed to get flapping alerts", "error", err)
		http.Error(w, "Failed to retrieve flapping alerts", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"alerts":    flappingAlerts,
		"count":     len(flappingAlerts),
		"threshold": threshold,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	h.logger.Info("Flapping alerts completed",
		"count", len(flappingAlerts),
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)
}

// ============================================================================
// TN-064: Analytics Report Endpoint
// ============================================================================

// HandleReport handles GET /api/v2/report - comprehensive analytics report
func (h *HistoryHandlerV2) HandleReport(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	h.logger.Info("Report request received",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"query", r.URL.RawQuery,
	)

	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method for report", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse and validate request
	req, err := h.parseReportRequest(r)
	if err != nil {
		h.logger.Warn("Invalid report request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate report (no cache in Phase 3, will add in Phase 5)
	report, err := h.generateReport(r.Context(), req)
	if err != nil {
		h.logger.Error("Failed to generate report", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set processing time
	elapsed := time.Since(startTime)
	report.Metadata.ProcessingTimeMs = elapsed.Milliseconds()

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode report response", "error", err)
		return
	}

	h.logger.Info("Report generated successfully",
		"processing_time_ms", elapsed.Milliseconds(),
		"total_alerts", report.Summary.TotalAlerts,
		"top_alerts_count", len(report.TopAlerts),
		"flapping_count", len(report.FlappingAlerts),
		"partial_failure", report.Metadata.PartialFailure,
	)
}

// parseReportRequest parses and validates query parameters
func (h *HistoryHandlerV2) parseReportRequest(r *http.Request) (*core.ReportRequest, error) {
	query := r.URL.Query()
	req := &core.ReportRequest{
		TopLimit:     10, // default
		MinFlapCount: 3,  // default
	}

	// Parse time range (from, to)
	if fromStr := query.Get("from"); fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return nil, &core.ValidationError{
				Field:   "from",
				Message: "invalid time format, expected RFC3339",
			}
		}
		if req.TimeRange == nil {
			req.TimeRange = &core.TimeRange{}
		}
		req.TimeRange.From = &from
	}

	if toStr := query.Get("to"); toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return nil, &core.ValidationError{
				Field:   "to",
				Message: "invalid time format, expected RFC3339",
			}
		}
		if req.TimeRange == nil {
			req.TimeRange = &core.TimeRange{}
		}
		req.TimeRange.To = &to
	}

	// Default time range: last 24 hours
	if req.TimeRange == nil {
		now := time.Now()
		from := now.Add(-24 * time.Hour)
		req.TimeRange = &core.TimeRange{From: &from, To: &now}
	}

	// Validate time range
	if req.TimeRange.From != nil && req.TimeRange.To != nil {
		if req.TimeRange.To.Before(*req.TimeRange.From) {
			return nil, &core.ValidationError{
				Field:   "to",
				Message: "'to' must be greater than or equal to 'from'",
			}
		}

		// Max 90 days
		maxRange := 90 * 24 * time.Hour
		if req.TimeRange.To.Sub(*req.TimeRange.From) > maxRange {
			return nil, &core.ValidationError{
				Field:   "time_range",
				Message: "time range too large: maximum 90 days allowed",
			}
		}
	}

	// Parse namespace filter
	if ns := query.Get("namespace"); ns != "" {
		if len(ns) > 255 {
			return nil, &core.ValidationError{
				Field:   "namespace",
				Message: "namespace too long: max 255 characters",
			}
		}
		req.Namespace = &ns
	}

	// Parse severity filter
	if sev := query.Get("severity"); sev != "" {
		validSeverities := map[string]bool{
			"critical": true,
			"warning":  true,
			"info":     true,
			"noise":    true,
		}
		if !validSeverities[sev] {
			return nil, &core.ValidationError{
				Field:   "severity",
				Message: "invalid severity: must be critical|warning|info|noise",
			}
		}
		req.Severity = &sev
	}

	// Parse top limit
	if topStr := query.Get("top"); topStr != "" {
		top, err := strconv.Atoi(topStr)
		if err != nil || top < 1 || top > 100 {
			return nil, &core.ValidationError{
				Field:   "top",
				Message: "invalid 'top' parameter: must be 1-100",
			}
		}
		req.TopLimit = top
	}

	// Parse min_flap
	if flapStr := query.Get("min_flap"); flapStr != "" {
		minFlap, err := strconv.Atoi(flapStr)
		if err != nil || minFlap < 1 || minFlap > 100 {
			return nil, &core.ValidationError{
				Field:   "min_flap",
				Message: "invalid 'min_flap' parameter: must be 1-100",
			}
		}
		req.MinFlapCount = minFlap
	}

	// Parse include_recent flag
	if includeRecent := query.Get("include_recent"); includeRecent == "true" {
		req.IncludeRecent = true
	}

	return req, nil
}

// generateReport generates the complete analytics report with parallel execution
func (h *HistoryHandlerV2) generateReport(ctx context.Context, req *core.ReportRequest) (*core.ReportResponse, error) {
	// Create timeout context (10s max)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use sync primitives for parallel execution
	type result struct {
		stats          *core.AggregatedStats
		topAlerts      []*core.TopAlert
		flappingAlerts []*core.FlappingAlert
		recentAlerts   []*core.Alert
		errors         map[string]error
	}

	resultChan := make(chan result, 1)

	go func() {
		var res result
		res.errors = make(map[string]error)

		// Use WaitGroup for parallel execution
		var wg sync.WaitGroup

		// Goroutine 1: GetAggregatedStats
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, err := h.repository.GetAggregatedStats(timeoutCtx, req.TimeRange)
			if err != nil {
				res.errors["stats"] = err
				h.logger.Error("Failed to get aggregated stats", "error", err)
			} else {
				res.stats = stats
			}
		}()

		// Goroutine 2: GetTopAlerts
		wg.Add(1)
		go func() {
			defer wg.Done()
			topAlerts, err := h.repository.GetTopAlerts(timeoutCtx, req.TimeRange, req.TopLimit)
			if err != nil {
				res.errors["top_alerts"] = err
				h.logger.Error("Failed to get top alerts", "error", err)
			} else {
				res.topAlerts = topAlerts
			}
		}()

		// Goroutine 3: GetFlappingAlerts
		wg.Add(1)
		go func() {
			defer wg.Done()
			flappingAlerts, err := h.repository.GetFlappingAlerts(timeoutCtx, req.TimeRange, req.MinFlapCount)
			if err != nil {
				res.errors["flapping_alerts"] = err
				h.logger.Error("Failed to get flapping alerts", "error", err)
			} else {
				res.flappingAlerts = flappingAlerts
			}
		}()

		// Optional Goroutine 4: GetRecentAlerts
		if req.IncludeRecent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				recentAlerts, err := h.repository.GetRecentAlerts(timeoutCtx, 20)
				if err != nil {
					res.errors["recent_alerts"] = err
					h.logger.Error("Failed to get recent alerts", "error", err)
				} else {
					res.recentAlerts = recentAlerts
				}
			}()
		}

		// Wait for all goroutines to complete
		wg.Wait()

		resultChan <- res
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		// Apply filters to results
		topAlerts := res.topAlerts
		flappingAlerts := res.flappingAlerts

		if req.Namespace != nil {
			topAlerts = filterTopAlertsByNamespace(topAlerts, *req.Namespace)
			flappingAlerts = filterFlappingAlertsByNamespace(flappingAlerts, *req.Namespace)
		}

		// Build response
		response := &core.ReportResponse{
			Metadata: &core.ReportMetadata{
				GeneratedAt:    time.Now(),
				CacheHit:       false,
				PartialFailure: len(res.errors) > 0,
			},
			Summary:        res.stats,
			TopAlerts:      topAlerts,
			FlappingAlerts: flappingAlerts,
			RecentAlerts:   res.recentAlerts,
		}

		// Add error messages if partial failure
		if len(res.errors) > 0 {
			errorMessages := []string{}
			for component, err := range res.errors {
				errorMessages = append(errorMessages, component+": "+err.Error())
			}
			response.Metadata.Errors = errorMessages
		}

		return response, nil

	case <-timeoutCtx.Done():
		return nil, &core.TimeoutError{
			Operation: "generate_report",
			Duration:  10 * time.Second,
		}
	}
}

// filterTopAlertsByNamespace filters top alerts by namespace
func filterTopAlertsByNamespace(alerts []*core.TopAlert, namespace string) []*core.TopAlert {
	if len(alerts) == 0 {
		return alerts
	}

	filtered := make([]*core.TopAlert, 0, len(alerts))
	for _, alert := range alerts {
		if alert.Namespace != nil && *alert.Namespace == namespace {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// filterFlappingAlertsByNamespace filters flapping alerts by namespace
func filterFlappingAlertsByNamespace(alerts []*core.FlappingAlert, namespace string) []*core.FlappingAlert {
	if len(alerts) == 0 {
		return alerts
	}

	filtered := make([]*core.FlappingAlert, 0, len(alerts))
	for _, alert := range alerts {
		if alert.Namespace != nil && *alert.Namespace == namespace {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}
