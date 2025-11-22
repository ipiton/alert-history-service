// Package examples provides integration examples for Silence UI Components.
// This file demonstrates how to integrate Silence UI Components into a Go application.
package examples

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
	businesssilencing "github.com/vitaliisemenov/alert-history/internal/business/silencing"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

// ExampleBasicIntegration demonstrates basic integration.
func ExampleBasicIntegration() {
	logger := slog.Default()

	// Initialize dependencies
	manager := // ... your SilenceManager instance
	apiHandler := // ... your SilenceHandler instance
	wsHub := handlers.NewWebSocketHub(logger)
	cache := // ... your cache instance

	// Create UI handler
	uiHandler, err := handlers.NewSilenceUIHandler(manager, apiHandler, wsHub, cache, logger)
	if err != nil {
		log.Fatal(err)
	}

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ui/silences", uiHandler.RenderDashboard)
	mux.HandleFunc("/ui/silences/create", uiHandler.RenderCreateForm)
	mux.HandleFunc("/ui/silences/{id}", uiHandler.RenderDetailView)
	mux.HandleFunc("/ui/silences/{id}/edit", uiHandler.RenderEditForm)
	mux.HandleFunc("/ui/silences/templates", uiHandler.RenderTemplates)
	mux.HandleFunc("/ui/silences/analytics", uiHandler.RenderAnalytics)
	mux.HandleFunc("/ws/silences", uiHandler.wsHub.HandleWebSocket)

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	logger.Info("Server starting", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// ExampleWithCompression demonstrates integration with compression.
func ExampleWithCompression() {
	logger := slog.Default()

	// ... initialize handler (same as above)

	// Enable compression
	mux.HandleFunc("/ui/silences", uiHandler.EnableCompression(uiHandler.RenderDashboard))
	mux.HandleFunc("/ui/silences/create", uiHandler.EnableCompression(uiHandler.RenderCreateForm))
	// ... other routes
}

// ExampleWithRateLimiting demonstrates integration with rate limiting.
func ExampleWithRateLimiting() {
	logger := slog.Default()

	// ... initialize handler

	// Configure rate limiting
	limiter := handlers.NewSilenceUIRateLimiter(100, 1*time.Minute, logger)
	uiHandler.SetRateLimiter(limiter)

	// Apply rate limiting middleware
	mux.HandleFunc("/ui/silences", uiHandler.RateLimitMiddleware(uiHandler.RenderDashboard))
	// ... other routes
}

// ExampleWithSecurity demonstrates integration with security features.
func ExampleWithSecurity() {
	logger := slog.Default()

	// ... initialize handler

	// Configure security
	config := handlers.DefaultSecurityConfig()
	config.AllowedOrigins = []string{"https://app.example.com", "https://*.example.com"}
	config.RateLimitEnabled = true
	config.RateLimitPerIP = 200
	config.RateLimitWindow = 1 * time.Minute
	uiHandler.SetSecurityConfig(&config)

	// Apply security middleware
	mux.HandleFunc("/ui/silences", uiHandler.SecurityMiddleware(uiHandler.RenderDashboard))
	// ... other routes
}

// ExampleWithMetrics demonstrates integration with Prometheus metrics.
func ExampleWithMetrics() {
	logger := slog.Default()

	// ... initialize handler

	// Metrics are automatically initialized in NewSilenceUIHandler
	// Access metrics via handler.metrics

	// Example: Record custom user action
	uiHandler.metrics.RecordUserAction("custom_action", "success")

	// Example: Record error
	uiHandler.metrics.RecordError("validation_error", "dashboard")
}

// ExampleWithWebSocket demonstrates WebSocket integration.
func ExampleWithWebSocket() {
	logger := slog.Default()

	// ... initialize handler

	// Start WebSocket hub
	ctx := context.Background()
	go uiHandler.wsHub.Run(ctx)

	// Broadcast events
	uiHandler.wsHub.Broadcast("silence_created", map[string]interface{}{
		"id":      "silence-123",
		"creator": "ops@example.com",
	})

	// Register WebSocket endpoint
	mux.HandleFunc("/ws/silences", uiHandler.wsHub.HandleWebSocket)
}

// ExampleWithHealthCheck demonstrates health check integration.
func ExampleWithHealthCheck() {
	logger := slog.Default()

	// ... initialize handler

	// Add health check endpoint
	mux.HandleFunc("/health/ui", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := uiHandler.HealthCheck(ctx)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})
}

// ExampleFullIntegration demonstrates complete integration with all features.
func ExampleFullIntegration() {
	logger := slog.Default()

	// Initialize all components
	manager := // ... your SilenceManager instance
	apiHandler := // ... your SilenceHandler instance
	wsHub := handlers.NewWebSocketHub(logger)
	cache := // ... your cache instance

	// Create UI handler
	uiHandler, err := handlers.NewSilenceUIHandler(manager, apiHandler, wsHub, cache, logger)
	if err != nil {
		log.Fatal(err)
	}

	// Configure security
	config := handlers.DefaultSecurityConfig()
	config.AllowedOrigins = []string{"https://app.example.com"}
	uiHandler.SetSecurityConfig(&config)

	// Configure rate limiting
	limiter := handlers.NewSilenceUIRateLimiter(200, 1*time.Minute, logger)
	uiHandler.SetRateLimiter(limiter)

	// Start WebSocket hub
	ctx := context.Background()
	go uiHandler.wsHub.Run(ctx)

	// Register routes with middleware
	mux := http.NewServeMux()

	// Dashboard with compression and rate limiting
	mux.HandleFunc("/ui/silences",
		uiHandler.SecurityMiddleware(
			uiHandler.RateLimitMiddleware(
				uiHandler.EnableCompression(
					uiHandler.RenderDashboard,
				),
			),
		),
	)

	// Other routes...
	mux.HandleFunc("/ui/silences/create",
		uiHandler.SecurityMiddleware(
			uiHandler.RateLimitMiddleware(
				uiHandler.EnableCompression(
					uiHandler.RenderCreateForm,
				),
			),
		),
	)

	// WebSocket endpoint
	mux.HandleFunc("/ws/silences", uiHandler.wsHub.HandleWebSocket)

	// Health check
	mux.HandleFunc("/health/ui", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := uiHandler.HealthCheck(ctx)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	logger.Info("Server starting with full integration", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
