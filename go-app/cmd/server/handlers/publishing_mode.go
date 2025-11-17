package handlers

import (
	"log/slog"

	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
	apihandlers "github.com/vitaliisemenov/alert-history/internal/api/handlers/publishing"
)

// NewPublishingModeHandler creates a new publishing mode handler.
//
// This is a wrapper around the internal API handler to maintain consistency
// with other handlers in the cmd/server/handlers package.
//
// Parameters:
//   - service: ModeService for business logic (required)
//   - logger: Logger for structured logging (defaults to slog.Default if nil)
//
// Returns:
//   - *apihandlers.PublishingModeHandler: Handler instance
//
// Example:
//   handler := NewPublishingModeHandler(modeService, logger)
//   router.HandleFunc("/api/v1/publishing/mode", handler.GetPublishingMode).Methods("GET")
//   router.HandleFunc("/api/v2/publishing/mode", handler.GetPublishingMode).Methods("GET")
func NewPublishingModeHandler(service apiservices.ModeService, logger *slog.Logger) *apihandlers.PublishingModeHandler {
	return apihandlers.NewPublishingModeHandler(service, logger)
}

