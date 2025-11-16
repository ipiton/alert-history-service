package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// SearchRequest represents a search request body
type SearchRequest struct {
	Query     string                 `json:"query" validate:"required"`
	Filters   *core.AlertFilters     `json:"filters,omitempty"`
	Pagination *core.Pagination      `json:"pagination,omitempty"`
	Sorting   *core.Sorting         `json:"sorting,omitempty"`
}

// SearchAlerts handles POST /api/v2/history/search - Advanced search
func (h *Handler) SearchAlerts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())
	
	// Parse request body
	var searchReq SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		h.logger.Warn("Failed to decode search request",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.ValidationError("Invalid request body: "+err.Error()).WithRequestID(requestID))
		return
	}
	
	// Validate query
	if searchReq.Query == "" {
		apierrors.WriteError(w, apierrors.ValidationError("query parameter is required").WithRequestID(requestID))
		return
	}
	
	// Set default pagination if not provided
	if searchReq.Pagination == nil {
		searchReq.Pagination = &core.Pagination{
			Page:    1,
			PerPage: 50,
		}
	}
	
	// Validate pagination
	if err := searchReq.Pagination.Validate(); err != nil {
		apierrors.WriteError(w, apierrors.ValidationError("Invalid pagination: "+err.Error()).WithRequestID(requestID))
		return
	}
	
	// Build HistoryRequest with search query
	// Note: Search is implemented as a filter in the filter system
	// For now, we'll use the basic GetHistory with filters
	// TODO: Integrate with FilterRegistry for full search support
	
	historyReq := &core.HistoryRequest{
		Filters:    searchReq.Filters,
		Pagination: searchReq.Pagination,
		Sorting:    searchReq.Sorting,
	}
	
	// If filters is nil, create empty filters
	if historyReq.Filters == nil {
		historyReq.Filters = &core.AlertFilters{}
	}
	
	// TODO: Apply search query as a filter
	// For now, search is a placeholder - will be implemented with FilterRegistry integration
	// The search filter should be applied to alert_name, annotations, etc.
	
	// Query repository
	response, err := h.repository.GetHistory(r.Context(), historyReq)
	if err != nil {
		h.logger.Error("Failed to search alerts",
			"request_id", requestID,
			"query", searchReq.Query,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to search alerts").WithRequestID(requestID))
		return
	}
	
	// Add search metadata to response
	searchResponse := map[string]interface{}{
		"query":    searchReq.Query,
		"alerts":   response.Alerts,
		"total":    response.Total,
		"page":     response.Page,
		"per_page": response.PerPage,
		"total_pages": response.TotalPages,
		"has_next": response.HasNext,
		"has_prev": response.HasPrev,
	}
	
	duration := time.Since(start)
	h.logger.Info("Search request completed",
		"request_id", requestID,
		"query", searchReq.Query,
		"total", response.Total,
		"duration_ms", duration.Milliseconds())
	
	h.sendJSON(w, http.StatusOK, searchResponse)
}

