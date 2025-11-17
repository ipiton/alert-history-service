// Package handlers provides HTTP handlers for Alert History Service API.
package handlers

import (
	"log/slog"
	"net/http"

	apihandlers "github.com/vitaliisemenov/alert-history/internal/api/handlers/publishing"
	businesspublishing "github.com/vitaliisemenov/alert-history/internal/business/publishing"
	infrapub "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// HandleTestTarget creates HTTP handler for testing publishing targets (TN-70).
//
// This handler wraps PublishingHandlers.TestTarget method and provides
// a simple function signature for router integration.
//
// Request:
//
//	POST /api/v2/publishing/targets/{name}/test
//	Content-Type: application/json
//	Body: (optional)
//	{
//	  "alert_name": "string (optional)",
//	  "test_alert": { ... } (optional),
//	  "timeout_seconds": 30 (optional, 1-300)
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Test alert sent",
//	  "target_name": "rootly-prod",
//	  "status_code": 200,
//	  "response_time_ms": 150,
//	  "test_timestamp": "2025-01-17T10:00:00Z"
//	}
//
// Performance: <500ms p95 (including target API call)
//
// Example:
//
//	mux.HandleFunc("POST /api/v2/publishing/targets/{name}/test",
//	    handlers.HandleTestTarget(discoveryMgr, coordinator, logger))
func HandleTestTarget(
	discoveryManager businesspublishing.TargetDiscoveryManager,
	coordinator interface{}, // *infrapub.PublishingCoordinator (interface{} to avoid circular import)
	logger *slog.Logger,
) http.HandlerFunc {
	// Type assertion to *infrapub.PublishingCoordinator
	coord, ok := coordinator.(*infrapub.PublishingCoordinator)
	if !ok {
		// Return error handler if type assertion fails
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid coordinator type"}`))
		}
	}

	// Create PublishingHandlers instance
	handlers := apihandlers.NewPublishingHandlers(
		discoveryManager,
		nil, // refreshManager not needed for test endpoint
		nil, // queue not needed for test endpoint
		coord,
		logger,
	)

	// Return handler method
	return handlers.TestTarget
}
