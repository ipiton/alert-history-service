package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/inhibition"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// InhibitionHandler handles HTTP requests for inhibition rules and status.
// It provides Alertmanager-compatible API endpoints for:
//   - GET /api/v2/inhibition/rules - List all loaded inhibition rules
//   - GET /api/v2/inhibition/status - Get active inhibition relationships
//   - POST /api/v2/inhibition/check - Check if an alert would be inhibited
type InhibitionHandler struct {
	parser       inhibition.InhibitionParser
	matcher      inhibition.InhibitionMatcher
	stateManager inhibition.InhibitionStateManager
	metrics      *metrics.BusinessMetrics
	logger       *slog.Logger
}

// NewInhibitionHandler creates a new InhibitionHandler instance.
func NewInhibitionHandler(
	parser inhibition.InhibitionParser,
	matcher inhibition.InhibitionMatcher,
	stateManager inhibition.InhibitionStateManager,
	businessMetrics *metrics.BusinessMetrics,
	logger *slog.Logger,
) *InhibitionHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &InhibitionHandler{
		parser:       parser,
		matcher:      matcher,
		stateManager: stateManager,
		metrics:      businessMetrics,
		logger:       logger,
	}
}

// InhibitionRulesResponse represents the response for GET /api/v2/inhibition/rules
type InhibitionRulesResponse struct {
	Rules []inhibition.InhibitionRule `json:"rules"`
	Count int                         `json:"count"`
}

// InhibitionStatusResponse represents the response for GET /api/v2/inhibition/status
type InhibitionStatusResponse struct {
	Active []*inhibition.InhibitionState `json:"active"`
	Count  int                           `json:"count"`
}

// InhibitionCheckRequest represents the request body for POST /api/v2/inhibition/check
type InhibitionCheckRequest struct {
	Alert *core.Alert `json:"alert"`
}

// InhibitionCheckResponse represents the response for POST /api/v2/inhibition/check
type InhibitionCheckResponse struct {
	Alert       *core.Alert             `json:"alert"`
	Inhibited   bool                    `json:"inhibited"`
	InhibitedBy *core.Alert             `json:"inhibited_by,omitempty"`
	Rule        *inhibition.InhibitionRule `json:"rule,omitempty"`
	LatencyMs   int64                   `json:"latency_ms"`
}

// GetRules handles GET /api/v2/inhibition/rules
// Returns a list of all loaded inhibition rules.
//
// Response example:
//
//	{
//	  "rules": [
//	    {
//	      "source_match": {"alertname": "NodeDown"},
//	      "target_match": {"alertname": "InstanceDown"},
//	      "equal": ["node", "cluster"]
//	    }
//	  ],
//	  "count": 1
//	}
func (h *InhibitionHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	config := h.parser.GetConfig()

	response := InhibitionRulesResponse{
		Rules: config.Rules,
		Count: len(config.Rules),
	}

	h.logger.Debug("Returning inhibition rules", "count", response.Count)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetStatus handles GET /api/v2/inhibition/status
// Returns all currently active inhibition relationships.
//
// Response example:
//
//	{
//	  "active": [
//	    {
//	      "target_fingerprint": "abc123",
//	      "source_fingerprint": "def456",
//	      "rule_name": "NodeDown_inhibits_InstanceDown",
//	      "inhibited_at": "2025-11-04T10:00:00Z"
//	    }
//	  ],
//	  "count": 1
//	}
func (h *InhibitionHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	states, err := h.stateManager.GetActiveInhibitions(ctx)
	if err != nil {
		h.logger.Error("Failed to get active inhibitions", "error", err)
		h.sendError(w, "Failed to retrieve inhibition status", http.StatusInternalServerError)
		return
	}

	response := InhibitionStatusResponse{
		Active: states,
		Count:  len(states),
	}

	h.logger.Debug("Returning active inhibitions", "count", response.Count)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CheckAlert handles POST /api/v2/inhibition/check
// Checks if the provided alert would be inhibited by any currently firing alerts.
//
// Request body:
//
//	{
//	  "alert": {
//	    "labels": {
//	      "alertname": "InstanceDown",
//	      "node": "node1"
//	    }
//	  }
//	}
//
// Response:
//
//	{
//	  "alert": {...},
//	  "inhibited": true,
//	  "inhibited_by": {...},
//	  "rule": {...},
//	  "latency_ms": 2
//	}
func (h *InhibitionHandler) CheckAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req InhibitionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Alert == nil {
		h.sendError(w, "Alert is required", http.StatusBadRequest)
		return
	}

	// Check if alert would be inhibited
	result, err := h.matcher.ShouldInhibit(ctx, req.Alert)
	if err != nil {
		h.logger.Error("Inhibition check failed", "error", err, "fingerprint", req.Alert.Fingerprint)
		h.sendError(w, "Inhibition check failed", http.StatusInternalServerError)
		return
	}

	// Build response
	response := InhibitionCheckResponse{
		Alert:     req.Alert,
		Inhibited: result.Matched,
		LatencyMs: result.MatchDuration.Milliseconds(),
	}

	if result.Matched {
		response.InhibitedBy = result.InhibitedBy
		response.Rule = result.Rule
	}

	h.logger.Debug("Inhibition check complete",
		"fingerprint", req.Alert.Fingerprint,
		"inhibited", result.Matched,
		"latency_ms", response.LatencyMs,
	)

	// Record metrics
	if h.metrics != nil {
		resultLabel := "allowed"
		if result.Matched {
			resultLabel = "inhibited"
		}
		h.metrics.InhibitionChecksTotal.WithLabelValues(resultLabel).Inc()
		h.metrics.InhibitionDurationSeconds.WithLabelValues("check").Observe(result.MatchDuration.Seconds())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error response (reuses ErrorResponse from enrichment.go)
func (h *InhibitionHandler) sendError(w http.ResponseWriter, message string, code int) {
	response := struct {
		Error string `json:"error"`
	}{
		Error: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}
