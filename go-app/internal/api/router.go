package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
)

// RouterConfig holds router configuration
type RouterConfig struct {
	// Middleware configuration
	EnableAuth        bool
	EnableRateLimit   bool
	EnableCompression bool
	EnableCORS        bool
	EnableMetrics     bool

	// Auth configuration
	AuthConfig middleware.AuthConfig

	// Rate limit configuration (requests per minute, burst)
	RateLimitPerMinute int
	RateLimitBurst     int

	// CORS configuration
	CORSConfig middleware.CORSConfig

	// Logger
	Logger *slog.Logger

	// Business services (TN-67: Publishing targets refresh)
	RefreshManager publishing.RefreshManager

	// TN-68: Publishing mode endpoint
	ModeService apiservices.ModeService
}

// DefaultRouterConfig returns default router configuration
func DefaultRouterConfig(logger *slog.Logger) RouterConfig {
	return RouterConfig{
		EnableAuth:         true,
		EnableRateLimit:    true,
		EnableCompression:  true,
		EnableCORS:         true,
		EnableMetrics:      true,
		RateLimitPerMinute: 100,
		RateLimitBurst:     20,
		CORSConfig:         middleware.DefaultCORSConfig(),
		Logger:             logger,
		AuthConfig: middleware.AuthConfig{
			EnableAPIKey: true,
			EnableJWT:    false,
			APIKeys:      make(map[string]*middleware.User),
		},
	}
}

// NewRouter creates a new API router with all middleware configured
//
// The middleware stack is applied in order:
//  1. RequestID (always)
//  2. Logging (always)
//  3. Metrics (if enabled)
//  4. CORS (if enabled)
//  5. Compression (if enabled)
//  6. Route-specific: Auth, RateLimit, Validation
//
// @title Alert History Publishing API
// @version 2.0.0
// @description Unified RESTful API for Alert History Publishing System
// @contact.name Platform Team
// @contact.email platform@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /api/v2
// @schemes http https
func NewRouter(config RouterConfig) *mux.Router {
	router := mux.NewRouter()

	// Apply global middleware (order matters!)
	router.Use(middleware.RequestIDMiddleware)
	router.Use(middleware.LoggingMiddleware(config.Logger))

	if config.EnableMetrics {
		router.Use(middleware.MetricsMiddleware)
	}

	if config.EnableCORS {
		router.Use(middleware.CORSMiddleware(config.CORSConfig))
	}

	if config.EnableCompression {
		router.Use(middleware.CompressionMiddleware)
	}

	// API v2 routes
	setupAPIv2Routes(router, config)

	// API v1 routes (backward compatibility)
	setupAPIv1Routes(router, config)

	// Documentation
	setupDocumentationRoutes(router)

	return router
}

// setupAPIv2Routes configures /api/v2 routes
func setupAPIv2Routes(router *mux.Router, config RouterConfig) {
	v2 := router.PathPrefix("/api/v2").Subrouter()

	// System health endpoint (public, no auth)
	v2.HandleFunc("/health", HealthCheckHandler(config.Logger)).Methods("GET")

	// Publishing routes
	setupPublishingRoutes(v2, config)

	// Classification routes (Phase 4)
	setupClassificationRoutes(v2, config)

	// History routes (Phase 4)
	setupHistoryRoutes(v2, config)

	// Enrichment routes (existing)
	// setupEnrichmentRoutes(v2, config)
}

// setupPublishingRoutes configures /api/v2/publishing/* routes
func setupPublishingRoutes(router *mux.Router, config RouterConfig) {
	pub := router.PathPrefix("/publishing").Subrouter()

	// --- Targets Management ---
	targets := pub.PathPrefix("/targets").Subrouter()

	// Public endpoints (no auth required)
	targets.HandleFunc("", PlaceholderHandler("ListTargets")).Methods("GET")
	targets.HandleFunc("/{name}", PlaceholderHandler("GetTarget")).Methods("GET")

	// Protected endpoints (require auth)
	targetsProtected := targets.PathPrefix("").Subrouter()
	if config.EnableAuth {
		targetsProtected.Use(middleware.AuthMiddleware(config.AuthConfig))
	}
	if config.EnableRateLimit {
		targetsProtected.Use(middleware.RateLimitMiddleware(config.RateLimitPerMinute, config.RateLimitBurst))
	}

	// Admin-only endpoints
	targetsAdmin := targetsProtected.NewRoute().Subrouter()
	if config.EnableAuth {
		targetsAdmin.Use(middleware.AdminMiddleware)
	}
	// TN-67: POST /api/v2/publishing/targets/refresh - Manual target refresh
	if config.RefreshManager != nil {
		targetsAdmin.HandleFunc("/refresh", handlers.HandleRefreshTargets(config.RefreshManager)).Methods("POST")
	} else {
		targetsAdmin.HandleFunc("/refresh", PlaceholderHandler("RefreshTargets")).Methods("POST")
	}

	// Operator+ endpoints
	targetsOperator := targetsProtected.NewRoute().Subrouter()
	if config.EnableAuth {
		targetsOperator.Use(middleware.OperatorMiddleware)
	}
	targetsOperator.HandleFunc("/{name}/test", PlaceholderHandler("TestTarget")).Methods("POST")

	// --- Targets Health ---
	health := targets.PathPrefix("/health").Subrouter()
	health.HandleFunc("", PlaceholderHandler("ListTargetsHealth")).Methods("GET")
	health.HandleFunc("/{name}", PlaceholderHandler("GetTargetHealth")).Methods("GET")
	health.HandleFunc("/stats", PlaceholderHandler("GetHealthStats")).Methods("GET")

	healthOperator := health.NewRoute().Subrouter()
	if config.EnableAuth {
		healthOperator.Use(middleware.AuthMiddleware(config.AuthConfig))
		healthOperator.Use(middleware.OperatorMiddleware)
	}
	healthOperator.HandleFunc("/{name}/check", PlaceholderHandler("CheckTargetHealth")).Methods("POST")

	// --- Queue Management ---
	queue := pub.PathPrefix("/queue").Subrouter()
	queue.HandleFunc("/status", PlaceholderHandler("GetQueueStatus")).Methods("GET")
	queue.HandleFunc("/stats", PlaceholderHandler("GetQueueStats")).Methods("GET")

	queueProtected := queue.PathPrefix("").Subrouter()
	if config.EnableAuth {
		queueProtected.Use(middleware.AuthMiddleware(config.AuthConfig))
		queueProtected.Use(middleware.OperatorMiddleware)
	}
	if config.EnableRateLimit {
		queueProtected.Use(middleware.RateLimitMiddleware(config.RateLimitPerMinute, config.RateLimitBurst))
	}
	queueProtected.Use(middleware.ValidationMiddleware)
	queueProtected.HandleFunc("/submit", PlaceholderHandler("SubmitAlert")).Methods("POST")

	// Jobs
	jobs := queue.PathPrefix("/jobs").Subrouter()
	jobs.HandleFunc("", PlaceholderHandler("ListJobs")).Methods("GET")
	jobs.HandleFunc("/{id}", PlaceholderHandler("GetJob")).Methods("GET")

	// --- DLQ Management ---
	dlq := pub.PathPrefix("/dlq").Subrouter()
	dlq.HandleFunc("", PlaceholderHandler("ListDLQEntries")).Methods("GET")

	dlqAdmin := dlq.PathPrefix("").Subrouter()
	if config.EnableAuth {
		dlqAdmin.Use(middleware.AuthMiddleware(config.AuthConfig))
		dlqAdmin.Use(middleware.AdminMiddleware)
	}
	dlqAdmin.HandleFunc("/{id}/replay", PlaceholderHandler("ReplayDLQEntry")).Methods("POST")
	dlqAdmin.HandleFunc("/purge", PlaceholderHandler("PurgeDLQ")).Methods("DELETE")

	// --- Parallel Publishing ---
	parallel := pub.PathPrefix("/parallel").Subrouter()
	parallel.HandleFunc("/status", PlaceholderHandler("GetParallelStatus")).Methods("GET")

	parallelProtected := parallel.PathPrefix("").Subrouter()
	if config.EnableAuth {
		parallelProtected.Use(middleware.AuthMiddleware(config.AuthConfig))
		parallelProtected.Use(middleware.OperatorMiddleware)
	}
	parallelProtected.Use(middleware.ValidationMiddleware)
	parallelProtected.HandleFunc("/targets", PlaceholderHandler("PublishToTargets")).Methods("POST")
	parallelProtected.HandleFunc("/all", PlaceholderHandler("PublishToAll")).Methods("POST")
	parallelProtected.HandleFunc("/healthy", PlaceholderHandler("PublishToHealthy")).Methods("POST")

	// --- Metrics & Stats ---
	metrics := pub.PathPrefix("/metrics").Subrouter()
	metrics.HandleFunc("/raw", PlaceholderHandler("GetRawMetrics")).Methods("GET")
	metrics.HandleFunc("/stats", PlaceholderHandler("GetMetricsStats")).Methods("GET")
	metrics.HandleFunc("/trends", PlaceholderHandler("GetTrends")).Methods("GET")
	metrics.HandleFunc("/targets/{name}", PlaceholderHandler("GetTargetMetrics")).Methods("GET")

	// --- Overall Health ---
	pub.HandleFunc("/health", PlaceholderHandler("GetPublishingHealth")).Methods("GET")

	// --- Mode Information (TN-68) ---
	// Public endpoint (no auth required)
	mode := pub.PathPrefix("/mode").Subrouter()
	if config.ModeService != nil {
		modeHandler := handlers.NewPublishingModeHandler(config.ModeService, config.Logger)
		mode.HandleFunc("", modeHandler.GetPublishingMode).Methods("GET")
	} else {
		mode.HandleFunc("", PlaceholderHandler("GetPublishingMode")).Methods("GET")
	}
}

// setupAPIv1Routes configures /api/v1 routes (backward compatibility)
func setupAPIv1Routes(router *mux.Router, config RouterConfig) {
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Add deprecation warning header
	v1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Deprecated", "true")
			w.Header().Set("X-API-Deprecation-Info", "API v1 is deprecated. Please migrate to /api/v2. Support ends 2026-11-13.")
			next.ServeHTTP(w, r)
		})
	})

	// TN-056 legacy routes
	v1Publishing := v1.PathPrefix("/publishing").Subrouter()
	v1Publishing.HandleFunc("/targets", PlaceholderHandler("ListTargets_v1")).Methods("GET")
	v1Publishing.HandleFunc("/submit", PlaceholderHandler("SubmitAlert_v1")).Methods("POST")

	// TN-68: Publishing mode endpoint (backward compatibility)
	if config.ModeService != nil {
		modeHandler := handlers.NewPublishingModeHandler(config.ModeService, config.Logger)
		v1Publishing.HandleFunc("/mode", modeHandler.GetPublishingMode).Methods("GET")
	} else {
		v1Publishing.HandleFunc("/mode", PlaceholderHandler("GetPublishingMode_v1")).Methods("GET")
	}
	// ... more legacy routes as needed
}

// setupDocumentationRoutes configures documentation routes
func setupDocumentationRoutes(router *mux.Router) {
	// Swagger UI
	router.PathPrefix("/api/v2/docs").Handler(httpSwagger.WrapHandler)

	// OpenAPI spec JSON
	router.HandleFunc("/api/v2/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Serve generated OpenAPI spec
		http.Error(w, "OpenAPI spec not yet generated", http.StatusNotImplemented)
	}).Methods("GET")
}

// HealthCheckHandler returns overall system health
//
// @Summary System health check
// @Description Returns health status of all subsystems
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{} "Healthy"
// @Failure 503 {object} map[string]interface{} "Unhealthy"
// @Router /health [get]
func HealthCheckHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement comprehensive health checks
		// - Database connectivity
		// - Redis connectivity
		// - Publishing queue status
		// - Discovery manager status

		response := map[string]interface{}{
			"status":  "healthy",
			"version": "2.0.0",
			"checks": map[string]string{
				"database": "healthy",
				"redis":    "healthy",
				"queue":    "healthy",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set(middleware.APIVersionHeader, "2.0.0")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("Failed to encode health response", "error", err)
		}
	}
}

// setupClassificationRoutes configures /api/v2/classification/* routes
func setupClassificationRoutes(router *mux.Router, config RouterConfig) {
	class := router.PathPrefix("/classification").Subrouter()

	// TODO: Initialize classification handlers with actual classifier
	// For now, use placeholder
	_ = class
	_ = config

	// Public endpoints (no auth required)
	class.HandleFunc("/models", PlaceholderHandler("ListClassificationModels")).Methods("GET")
	class.HandleFunc("/stats", PlaceholderHandler("GetClassificationStats")).Methods("GET")

	// Protected endpoints (require auth)
	classProtected := class.PathPrefix("").Subrouter()
	if config.EnableAuth {
		classProtected.Use(middleware.AuthMiddleware(config.AuthConfig))
	}
	if config.EnableRateLimit {
		classProtected.Use(middleware.RateLimitMiddleware(config.RateLimitPerMinute, config.RateLimitBurst))
	}

	classProtected.HandleFunc("/classify", PlaceholderHandler("ClassifyAlert")).Methods("POST")
}

// setupHistoryRoutes configures /api/v2/history/* routes
func setupHistoryRoutes(router *mux.Router, config RouterConfig) {
	hist := router.PathPrefix("/history").Subrouter()

	// TODO: Initialize history handlers with actual repository
	// For now, use placeholder
	_ = hist
	_ = config

	// Public endpoints (no auth required)
	hist.HandleFunc("/top", PlaceholderHandler("GetTopAlerts")).Methods("GET")
	hist.HandleFunc("/flapping", PlaceholderHandler("GetFlappingAlerts")).Methods("GET")
	hist.HandleFunc("/recent", PlaceholderHandler("GetRecentAlerts")).Methods("GET")
}

// PlaceholderHandler returns a placeholder handler for routes not yet implemented
// This allows router to compile and be tested while handlers are being migrated
func PlaceholderHandler(handlerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		err := apierrors.InternalError("Handler not yet implemented: " + handlerName).
			WithRequestID(requestID)
		apierrors.WriteError(w, err)
	}
}
