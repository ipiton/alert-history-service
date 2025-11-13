package history

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
)

// HistoryHandlers provides HTTP handlers for alert history operations
type HistoryHandlers struct {
	logger *slog.Logger
	// TODO: Add history repository/service
}

// NewHistoryHandlers creates new history handlers
func NewHistoryHandlers(logger *slog.Logger) *HistoryHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &HistoryHandlers{
		logger: logger,
	}
}

// AlertHistoryEntry represents a single alert history entry
type AlertHistoryEntry struct {
	Fingerprint  string    `json:"fingerprint"`
	AlertName    string    `json:"alert_name"`
	Status       string    `json:"status"`
	Severity     string    `json:"severity"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	ReceivedAt   time.Time `json:"received_at"`
}

// TopAlertsResponse represents top alerts response
type TopAlertsResponse struct {
	Alerts []TopAlertEntry `json:"alerts"`
	Period string          `json:"period"`
	Total  int             `json:"total"`
}

// TopAlertEntry represents a top alert entry
type TopAlertEntry struct {
	AlertName   string  `json:"alert_name"`
	Count       int64   `json:"count"`
	Severity    string  `json:"severity"`
	LastSeen    time.Time `json:"last_seen"`
	AvgDuration float64 `json:"avg_duration_seconds"`
}

// FlappingAlertsResponse represents flapping alerts response
type FlappingAlertsResponse struct {
	Alerts []FlappingAlertEntry `json:"alerts"`
	Period string               `json:"period"`
	Total  int                  `json:"total"`
}

// FlappingAlertEntry represents a flapping alert entry
type FlappingAlertEntry struct {
	AlertName      string    `json:"alert_name"`
	FlipCount      int64     `json:"flip_count"`
	LastFlip       time.Time `json:"last_flip"`
	FlappingScore  float64   `json:"flapping_score"`
	Status         string    `json:"status"`
}

// RecentAlertsResponse represents recent alerts response
type RecentAlertsResponse struct {
	Alerts []AlertHistoryEntry `json:"alerts"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// GetTopAlerts handles GET /api/v2/history/top
//
// @Summary Get top alerts
// @Description Returns the most frequently occurring alerts in a given period
// @Tags History
// @Produce json
// @Param period query string false "Time period (1h, 24h, 7d, 30d)" default(24h)
// @Param limit query int false "Maximum number of results" default(10)
// @Success 200 {object} TopAlertsResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Router /history/top [get]
func (h *HistoryHandlers) GetTopAlerts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	if period == "" {
		period = "24h"
	}

	limit := 10
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	_ = limit // Suppress unused warning

	// TODO: Implement actual query to history repository
	// For now, return mock data
	response := TopAlertsResponse{
		Period: period,
		Total:  0,
		Alerts: []TopAlertEntry{},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetFlappingAlerts handles GET /api/v2/history/flapping
//
// @Summary Get flapping alerts
// @Description Returns alerts that are frequently changing state (firing/resolved)
// @Tags History
// @Produce json
// @Param period query string false "Time period (1h, 24h, 7d, 30d)" default(24h)
// @Param threshold query int false "Minimum flip count to be considered flapping" default(5)
// @Param limit query int false "Maximum number of results" default(10)
// @Success 200 {object} FlappingAlertsResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Router /history/flapping [get]
func (h *HistoryHandlers) GetFlappingAlerts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	if period == "" {
		period = "24h"
	}

	threshold := 5
	if thresholdStr := query.Get("threshold"); thresholdStr != "" {
		if parsedThreshold, err := strconv.Atoi(thresholdStr); err == nil && parsedThreshold > 0 {
			threshold = parsedThreshold
		}
	}

	limit := 10
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	_ = threshold // Suppress unused warning
	_ = limit     // Suppress unused warning

	// TODO: Implement actual query to history repository
	// For now, return mock data
	response := FlappingAlertsResponse{
		Period: period,
		Total:  0,
		Alerts: []FlappingAlertEntry{},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetRecentAlerts handles GET /api/v2/history/recent
//
// @Summary Get recent alerts
// @Description Returns most recent alerts with pagination
// @Tags History
// @Produce json
// @Param limit query int false "Maximum number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Param status query string false "Filter by status (firing, resolved)"
// @Param severity query string false "Filter by severity (critical, warning, info, noise)"
// @Success 200 {object} RecentAlertsResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Router /history/recent [get]
func (h *HistoryHandlers) GetRecentAlerts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	status := query.Get("status")
	severity := query.Get("severity")

	_ = status   // Suppress unused warning
	_ = severity // Suppress unused warning

	// TODO: Implement actual query to history repository
	// For now, return mock data
	response := RecentAlertsResponse{
		Total:  0,
		Limit:  limit,
		Offset: offset,
		Alerts: []AlertHistoryEntry{},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Helper Methods =====

func (h *HistoryHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}
