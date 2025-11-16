package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/cache"
	"github.com/vitaliisemenov/alert-history/pkg/history/filters"
)

// Handler handles HTTP requests for alert history endpoints
type Handler struct {
	repository   core.AlertHistoryRepository
	filterRegistry *filters.Registry
	cacheManager  *cache.Manager
	logger        *slog.Logger
}

// NewHandler creates a new history handler
func NewHandler(
	repository core.AlertHistoryRepository,
	filterRegistry *filters.Registry,
	cacheManager *cache.Manager,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &Handler{
		repository:     repository,
		filterRegistry: filterRegistry,
		cacheManager:   cacheManager,
		logger:         logger,
	}
}

// GetHistory handles GET /api/v2/history - Main history endpoint
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())
	
	// Parse query parameters
	queryParams := r.URL.Query()
	
	// Build filters from query parameters
	// TODO: Implement CreateFromQueryParams in FilterRegistry
	// For now, parse basic filters manually
	alertFilters := &core.AlertFilters{}
	
	// Parse status filter
	if statusStr := queryParams.Get("status"); statusStr != "" {
		status := core.AlertStatus(statusStr)
		if status == core.StatusFiring || status == core.StatusResolved {
			alertFilters.Status = &status
		}
	}
	
	// Parse severity filter
	if severityStr := queryParams.Get("severity"); severityStr != "" {
		alertFilters.Severity = &severityStr
	}
	
	// Parse namespace filter
	if namespaceStr := queryParams.Get("namespace"); namespaceStr != "" {
		alertFilters.Namespace = &namespaceStr
	}
	
	// Parse time range
	if fromStr := queryParams.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			alertFilters.TimeRange = &core.TimeRange{
				From: &from,
			}
		}
	}
	if toStr := queryParams.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if alertFilters.TimeRange == nil {
				alertFilters.TimeRange = &core.TimeRange{}
			}
			alertFilters.TimeRange.To = &to
		}
	}
	
	// Parse pagination
	page := 1
	if pageStr := queryParams.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	perPage := 50
	if perPageStr := queryParams.Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
			if perPage > 1000 {
				perPage = 1000
			}
		}
	}
	
	// Parse sorting
	var sorting *core.Sorting
	if sortField := queryParams.Get("sort_field"); sortField != "" {
		sorting = &core.Sorting{
			Field: sortField,
			Order: core.SortOrderDesc, // default desc
		}
		if sortOrder := queryParams.Get("sort_order"); sortOrder != "" {
			sorting.Order = core.SortOrder(sortOrder)
		}
	}
	
	// Build HistoryRequest
	req := &core.HistoryRequest{
		Filters:    alertFilters,
		Pagination: &core.Pagination{
			Page:    page,
			PerPage: perPage,
		},
		Sorting: sorting,
	}
	
	// Generate cache key
	cacheKey := h.cacheManager.GenerateCacheKey(req)
	
	// Try cache first
	if cached, found := h.cacheManager.Get(r.Context(), cacheKey); found {
		h.logger.Debug("Cache hit",
			"request_id", requestID,
			"cache_key", cacheKey)
		
		h.sendJSON(w, http.StatusOK, cached)
		return
	}
	
	// Cache miss - query database
	h.logger.Debug("Cache miss, querying database",
		"request_id", requestID,
		"cache_key", cacheKey)
	
	response, err := h.repository.GetHistory(r.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get history",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve alert history").WithRequestID(requestID))
		return
	}
	
	// Store in cache
	if err := h.cacheManager.Set(r.Context(), cacheKey, response); err != nil {
		h.logger.Warn("Failed to cache result",
			"request_id", requestID,
			"error", err)
		// Continue - caching failure is not critical
	}
	
	duration := time.Since(start)
	h.logger.Info("History request completed",
		"request_id", requestID,
		"page", page,
		"per_page", perPage,
		"total", response.Total,
		"returned", len(response.Alerts),
		"duration_ms", duration.Milliseconds(),
		"cache_hit", false)
	
	h.sendJSON(w, http.StatusOK, response)
}

// sendJSON sends JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

