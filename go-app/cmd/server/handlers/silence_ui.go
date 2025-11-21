// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	businesssilencing "github.com/vitaliisemenov/alert-history/internal/business/silencing"
	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

//go:embed templates/silences/*.html templates/common/*.html
var templatesFS embed.FS

//go:embed static/*.json static/*.js
var staticFS embed.FS

// SilenceUIHandler handles UI rendering for Silence Management.
type SilenceUIHandler struct {
	manager            businesssilencing.SilenceManager // Business logic
	apiHandler         *SilenceHandler                    // API handler (reuse for data fetching)
	templates          *template.Template                // Parsed templates
	wsHub              *WebSocketHub                     // WebSocket hub for real-time updates
	cache              cache.Cache                       // Response caching
	templateCache      *TemplateCache                    // Template rendering cache (Phase 10 enhancement)
	csrfManager        *CSRFManager                     // CSRF token manager (Phase 12 enhancement)
	metrics            *SilenceUIMetrics                // Prometheus metrics (Phase 14 enhancement)
	securityConfig        *SecurityConfig           // Security configuration (Phase 13 enhancement)
	compressionMiddleware *CompressionMiddleware    // Compression middleware (Phase 10 enhancement)
	rateLimiter           *RateLimiter             // Rate limiter (Phase 13 enhancement)
	logger                *slog.Logger
}

// NewSilenceUIHandler creates a new SilenceUIHandler.
func NewSilenceUIHandler(
	manager businesssilencing.SilenceManager,
	apiHandler *SilenceHandler,
	wsHub *WebSocketHub,
	cache cache.Cache,
	logger *slog.Logger,
) (*SilenceUIHandler, error) {
	// Parse templates from embed.FS
	tmpl, err := template.New("").
		Funcs(templateFuncs()).
		ParseFS(templatesFS,
			"templates/silences/*.html",
			"templates/common/*.html",
		)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	handler := &SilenceUIHandler{
		manager:       manager,
		apiHandler:    apiHandler,
		templates:     tmpl,
		wsHub:         wsHub,
		cache:         cache,
		templateCache: NewTemplateCache(100, 5*time.Minute, logger), // Phase 10: Template caching
		csrfManager:   NewCSRFManager(nil, 24*time.Hour, logger),  // Phase 12: CSRF protection
		metrics:       NewSilenceUIMetrics(logger),                  // Phase 14: Prometheus metrics
		securityConfig: nil, // Will be set via SetSecurityConfig if needed (Phase 13)
		logger:        logger,
	}

	logger.Info("✅ SilenceUIHandler initialized successfully",
		"templates", "silences/*.html, common/*.html",
		"static_assets", "embedded",
	)

	return handler, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// RenderDashboard renders the silence dashboard page.
// GET /ui/silences
func (h *SilenceUIHandler) RenderDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Parse query params (filters, pagination)
	filters := h.parseFilterParams(r.URL.Query())
	if err := filters.Validate(); err != nil {
		h.logger.Warn("Invalid filter parameters", "error", err)
		// Continue with sanitized filters
	}

	// Fetch silences from manager
	silences, err := h.manager.ListSilences(ctx, filters.ToSilenceFilter())
	if err != nil {
		h.logger.Error("Failed to list silences", "error", err)
		h.renderError(w, r, "Failed to load silences", http.StatusInternalServerError)
		return
	}

	// Count total silences (for pagination)
	total := len(silences) // TODO: Get actual count from manager
	if total > filters.Limit {
		total = filters.Limit // Temporary workaround
	}

	// Calculate pagination
	currentPage := GetCurrentPage(filters.Offset, filters.Limit)
	totalPages := CalculateTotalPages(total, filters.Limit)

	// Prepare template data
	data := DashboardData{
		Silences:   silences,
		Total:      total,
		Filters:    filters,
		Page:       currentPage,
		PageSize:   filters.Limit,
		TotalPages: totalPages,
		CSRF:       h.generateCSRFToken(r),
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "dashboard.html", data); err != nil {
		h.logger.Error("Failed to render dashboard template", "error", err)
		h.renderError(w, r, "Failed to render page", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)

	// Record metrics (Phase 14 enhancement)
	if h.metrics != nil {
		h.metrics.RecordPageRender("dashboard", duration, "success")
	}

	h.logger.Debug("Dashboard rendered",
		"duration_ms", duration.Milliseconds(),
		"silences_count", len(silences),
		"filters", filters.Status,
	)
}

// RenderCreateForm renders the create silence form.
// GET /ui/silences/create
func (h *SilenceUIHandler) RenderCreateForm(w http.ResponseWriter, r *http.Request) {
	// Check if template ID is provided (create from template)
	templateID := r.URL.Query().Get("template")
	var initialMatchers []MatcherInput

	if templateID != "" {
		tmpl := GetTemplateByID(templateID)
		if tmpl != nil {
			initialMatchers = tmpl.Matchers
		}
	}

	data := CreateFormData{
		CSRF:        h.generateCSRFToken(r),
		Matchers:    initialMatchers,
		TimePresets: GetDefaultTimePresets(),
		Error:       "",
		ErrorFields: make(map[string]string),
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "create_form.html", data); err != nil {
		h.logger.Error("Failed to render create form", "error", err)
		h.renderError(w, r, "Failed to render form", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Create form rendered", "template_id", templateID)
}

// RenderDetailView renders the silence detail view.
// GET /ui/silences/{id}
func (h *SilenceUIHandler) RenderDetailView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from URL path
	id := h.extractIDFromPath(r.URL.Path, "/ui/silences/")
	if id == "" {
		h.renderError(w, r, "Invalid silence ID", http.StatusBadRequest)
		return
	}

	// Fetch silence
	silence, err := h.manager.GetSilence(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get silence", "error", err, "id", id)
		h.renderError(w, r, "Silence not found", http.StatusNotFound)
		return
	}

	// Count matched alerts (TODO: implement via manager)
	matchedCount := 0 // Placeholder

	data := DetailViewData{
		Silence:      silence,
		MatchedCount: matchedCount,
		CSRF:         h.generateCSRFToken(r),
		RefreshRate:  10, // 10 seconds
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "detail_view.html", data); err != nil {
		h.logger.Error("Failed to render detail view", "error", err)
		h.renderError(w, r, "Failed to render page", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Detail view rendered", "id", id)
}

// RenderEditForm renders the edit silence form.
// GET /ui/silences/{id}/edit
func (h *SilenceUIHandler) RenderEditForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from URL path
	id := h.extractIDFromPath(r.URL.Path, "/ui/silences/")
	if id == "" || !strings.HasSuffix(r.URL.Path, "/edit") {
		h.renderError(w, r, "Invalid silence ID", http.StatusBadRequest)
		return
	}

	// Remove "/edit" suffix to get actual ID
	id = strings.TrimSuffix(id, "/edit")

	// Fetch silence
	silence, err := h.manager.GetSilence(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get silence", "error", err, "id", id)
		h.renderError(w, r, "Silence not found", http.StatusNotFound)
		return
	}

	data := EditFormData{
		Silence:     silence,
		CSRF:        h.generateCSRFToken(r),
		TimePresets: GetDefaultTimePresets(),
		Error:       "",
		ErrorFields: make(map[string]string),
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "edit_form.html", data); err != nil {
		h.logger.Error("Failed to render edit form", "error", err)
		h.renderError(w, r, "Failed to render form", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Edit form rendered", "id", id)
}

// RenderTemplates renders the templates page.
// GET /ui/silences/templates
func (h *SilenceUIHandler) RenderTemplates(w http.ResponseWriter, r *http.Request) {
	data := TemplatesData{
		Templates: GetBuiltInTemplates(),
		CSRF:      h.generateCSRFToken(r),
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "templates.html", data); err != nil {
		h.logger.Error("Failed to render templates page", "error", err)
		h.renderError(w, r, "Failed to render page", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Templates page rendered", "templates_count", len(data.Templates))
}

// RenderAnalytics renders the analytics dashboard.
// GET /ui/silences/analytics
func (h *SilenceUIHandler) RenderAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Fetch silence stats from manager
	stats, err := h.manager.GetStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get silence stats", "error", err)
		h.renderError(w, r, "Failed to load analytics", http.StatusInternalServerError)
		return
	}

	// Convert stats to our format
	analyticsStats := &SilenceStats{
		Total:        int(stats.TotalSilences),
		Active:       int(stats.ActiveSilences),
		Pending:      int(stats.PendingSilences),
		Expired:      int(stats.ExpiredSilences),
		Expired24h:   0, // TODO: Calculate from stats
		AvgDuration:  0, // TODO: Calculate from stats
		TotalMatchers: 0, // TODO: Calculate from stats
	}

	// TODO: Fetch timeline data, top creators, top silenced alerts

	data := AnalyticsData{
		Stats:       analyticsStats,
		Timeline:    []TimelinePoint{},    // TODO: Implement
		TopCreators: []CreatorStat{},      // TODO: Implement
		TopSilenced: []AlertStat{},        // TODO: Implement
		TimeRange:   "last_7_days",
		RefreshRate: 5 * time.Minute,
	}

	// Render template with caching (Phase 10 enhancement)
	if err := h.renderWithCache(w, r, "analytics.html", data); err != nil {
		h.logger.Error("Failed to render analytics page", "error", err)
		h.renderError(w, r, "Failed to render page", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Analytics dashboard rendered")
}

// HandleDynamicRoutes handles dynamic routes (ID-based).
// This is a catch-all handler for /ui/silences/{id} and /ui/silences/{id}/edit
func (h *SilenceUIHandler) HandleDynamicRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check if it's edit route
	if strings.HasSuffix(path, "/edit") {
		h.RenderEditForm(w, r)
		return
	}

	// Otherwise, it's detail view
	h.RenderDetailView(w, r)
}

// ============================================================================
// Helper Methods
// ============================================================================

// parseFilterParams parses filter parameters from query string.
func (h *SilenceUIHandler) parseFilterParams(query url.Values) FilterParams {
	filters := NewFilterParams()

	// Status filter
	if status := query.Get("status"); status != "" {
		filters.Status = status
	}

	// Creator filter
	if creator := query.Get("creator"); creator != "" {
		filters.Creator = creator
	}

	// Matcher filter
	if matcher := query.Get("matcher"); matcher != "" {
		filters.Matcher = matcher
	}

	// Time range filters
	if startsAfter := query.Get("starts_after"); startsAfter != "" {
		if t, err := time.Parse(time.RFC3339, startsAfter); err == nil {
			filters.StartsAfter = t
		}
	}
	if startsBefore := query.Get("starts_before"); startsBefore != "" {
		if t, err := time.Parse(time.RFC3339, startsBefore); err == nil {
			filters.StartsBefore = t
		}
	}

	// Pagination
	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}
	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}
	if page := query.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filters.Offset = p * filters.Limit
		}
	}

	// Sorting
	if sortBy := query.Get("sort_by"); sortBy != "" {
		filters.SortBy = sortBy
	}
	if sortOrder := query.Get("sort_order"); sortOrder != "" {
		filters.SortOrder = sortOrder
	}

	return filters
}

// extractIDFromPath extracts ID from URL path.
// Example: /ui/silences/550e8400-e29b-41d4-a716-446655440000 → 550e8400-e29b-41d4-a716-446655440000
func (h *SilenceUIHandler) extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	// Remove prefix
	id := strings.TrimPrefix(path, prefix)

	// Remove trailing slash
	id = strings.TrimSuffix(id, "/")

	// Remove /edit suffix if present
	id = strings.TrimSuffix(id, "/edit")

	return id
}

// generateCSRFToken generates a CSRF token for forms.
// Phase 12: CSRF protection implementation.
// Note: Actual implementation is in silence_ui_csrf.go

// renderError renders an error page.
func (h *SilenceUIHandler) renderError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	data := ErrorData{
		Message:    message,
		StatusCode: statusCode,
		RequestID:  h.extractRequestID(r.Context()),
		BackURL:    "/ui/silences",
	}

	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		h.logger.Error("Failed to render error page", "error", err)
		// Fallback to plain text error
		http.Error(w, message, statusCode)
		return
	}

	h.logger.Debug("Error page rendered",
		"status_code", statusCode,
		"message", message,
		"request_id", data.RequestID,
	)
}

// extractRequestID extracts request ID from context.
func (h *SilenceUIHandler) extractRequestID(ctx context.Context) string {
	// TODO: Extract from context if available
	return "request-id-placeholder"
}

// countMatchedAlerts counts alerts currently matched by a silence.
// TODO: Implement via manager.IsAlertSilenced or dedicated method.
func (h *SilenceUIHandler) countMatchedAlerts(ctx context.Context, silence *coresilencing.Silence) int {
	// Placeholder implementation
	// In production:
	// 1. Fetch active alerts
	// 2. For each alert, call manager.IsAlertSilenced
	// 3. Count matches
	return 0
}

// ============================================================================
// Static Assets Handler
// ============================================================================

// GetStaticFS returns the embedded static filesystem.
func (h *SilenceUIHandler) GetStaticFS() embed.FS {
	return staticFS
}
