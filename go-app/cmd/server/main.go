// Package main is the entry point for Alert History Service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling endpoints
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
	"github.com/vitaliisemenov/alert-history/internal/database"
	"github.com/vitaliisemenov/alert-history/internal/database/postgres"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/grouping"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/inhibition"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/llm"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/repository"
	"github.com/vitaliisemenov/alert-history/pkg/logger"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
	pkgmiddleware "github.com/vitaliisemenov/alert-history/pkg/middleware"
)

const (
	defaultPort    = "8080"
	serviceName    = "alert-history"
	serviceVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	var healthCheck = flag.Bool("health-check", false, "Perform health check and exit")
	var configPathFlag = flag.String("config", "", "Path to config file (YAML)")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("%s version %s\n", serviceName, serviceVersion)
		os.Exit(0)
	}

	// Handle health check flag (for Docker health check)
	if *healthCheck {
		// Simple health check - try to connect to the health endpoint
		resp, err := http.Get("http://localhost:8080/healthz")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Health check failed: status %d\n", resp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("Health check passed")
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("Alert History Service - Intelligent Alert Proxy\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		fmt.Printf("  -version           Show version information\n")
		fmt.Printf("  -help              Show this help message\n")
		fmt.Printf("  -health-check      Perform health check and exit\n")
		fmt.Printf("  -config <path>     Path to YAML config file\n\n")
		fmt.Printf("Environment variables:\n")
		fmt.Printf("  CONFIG_FILE        YAML config path (used if -config is not set)\n")
		fmt.Printf("  SERVER_PORT        HTTP server port (overrides config)\n")
		fmt.Printf("  DATABASE_HOST      Database host (overrides config)\n")
		fmt.Printf("  ...see go-app/config.yaml for all keys, env uses dot->underscore mapping\n")
		fmt.Printf("Defaults are applied then overridden by file and env.\n")
		fmt.Printf("If neither -config nor CONFIG_FILE are set, ./config.yaml is used if present.\n")
		os.Exit(0)
	}

	// Resolve config file path with priority: -config > CONFIG_FILE > ./config.yaml (if exists) > env only
	var resolvedConfigPath string
	if cp := strings.TrimSpace(*configPathFlag); cp != "" {
		// Explicit -config requires file to exist, otherwise fail fast
		if _, err := os.Stat(cp); err != nil {
			fmt.Fprintf(os.Stderr, "Config file not found: %s\n", cp)
			os.Exit(1)
		}
		resolvedConfigPath = cp
	} else if cp := strings.TrimSpace(os.Getenv("CONFIG_FILE")); cp != "" {
		// For CONFIG_FILE we do not fail if missing; LoadConfig will fallback to defaults+env
		resolvedConfigPath = cp
	} else if _, err := os.Stat("config.yaml"); err == nil {
		resolvedConfigPath = "config.yaml"
	}

	// Load configuration
	var cfg *appconfig.Config
	var err error
	if resolvedConfigPath != "" {
		cfg, err = appconfig.LoadConfig(resolvedConfigPath)
	} else {
		cfg, err = appconfig.LoadConfigFromEnv()
	}
	if err != nil {
		// Configure a minimal logger to print the error in JSON
		tmpLogger := logger.NewLogger(logger.Config{
			Level:  "error",
			Format: "json",
			Output: "stdout",
		})
		tmpLogger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Configure structured logging from config
	appLogger := logger.NewLogger(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})
	slog.SetDefault(appLogger)

	slog.Info("Starting Alert History Service",
		"service", serviceName,
		"version", serviceVersion,
		"env", cfg.App.Environment,
		"debug", cfg.IsDebug(),
	)

	// TN-181: Initialize unified Metrics Registry (150% quality)
	// Initialize early so it's available for DB Pool exporter and other components
	slog.Info("Initializing unified Metrics Registry (TN-181)")
	metricsRegistry := metrics.DefaultRegistry()
	slog.Info("✅ Metrics Registry initialized",
		"categories", []string{"business", "technical", "infra"},
		"metrics_count", "~30 metrics")

	// Build Postgres config from app config
	dbCfg := postgres.DefaultConfig()
	dbCfg.Host = cfg.Database.Host
	dbCfg.Port = cfg.Database.Port
	dbCfg.Database = cfg.Database.Database
	dbCfg.User = cfg.Database.Username
	dbCfg.Password = cfg.Database.Password
	dbCfg.SSLMode = cfg.Database.SSLMode
	if cfg.Database.MaxConnections > 0 {
		dbCfg.MaxConns = int32(cfg.Database.MaxConnections)
	}
	if cfg.Database.MinConnections > 0 {
		dbCfg.MinConns = int32(cfg.Database.MinConnections)
	}
	if cfg.Database.MaxConnLifetime > 0 {
		dbCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
	}
	if cfg.Database.MaxConnIdleTime > 0 {
		dbCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	}
	if cfg.Database.ConnectTimeout > 0 {
		dbCfg.ConnectTimeout = cfg.Database.ConnectTimeout
	}
	// HealthCheckPeriod остается дефолтным, пока не добавим в общий конфиг

	// Initialize database connection and run migrations
	slog.Info("Initializing database connection...",
		"host", dbCfg.Host, "port", dbCfg.Port, "db", dbCfg.Database, "user", dbCfg.User)

	// Check if we should use mock mode for performance testing
	useMockMode := os.Getenv("MOCK_MODE") == "true" || os.Getenv("PERFORMANCE_TEST") == "true"

	// Initialize database pool and storage (available globally for handlers)
	var pool *postgres.PostgresPool
	var alertStorage core.AlertStorage
	var historyRepo core.AlertHistoryRepository

	if useMockMode {
		slog.Info("🚀 Running in MOCK MODE for performance testing - no database required")
	} else {
		pool = postgres.NewPostgresPool(dbCfg, appLogger)

		ctx := context.Background()
		if err := pool.Connect(ctx); err != nil {
			slog.Warn("Failed to connect to database, switching to MOCK MODE", "error", err)
			useMockMode = true
		} else {
			slog.Info("✅ Successfully connected to PostgreSQL!")

			// Run database migrations
			if err := database.RunMigrations(ctx, pool, appLogger); err != nil {
				slog.Error("Failed to run database migrations", "error", err)
				// Не завершаем работу, если миграции не удались - даем возможность ручного исправления
				slog.Warn("Continuing without migrations - manual intervention may be required")
			} else {
				slog.Info("✅ Database migrations completed successfully")
			}

			// Initialize AlertStorage (PostgreSQL implementation)
			pgConfig := &infrastructure.Config{
				DSN:             fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Database, dbCfg.SSLMode),
				MaxOpenConns:    int(dbCfg.MaxConns),
				MaxIdleConns:    int(dbCfg.MinConns),
				ConnMaxLifetime: dbCfg.MaxConnLifetime,
				ConnMaxIdleTime: dbCfg.MaxConnIdleTime,
				Logger:          appLogger,
			}
			pgStorage, err := infrastructure.NewPostgresDatabase(pgConfig)
			if err != nil {
				slog.Error("Failed to create PostgreSQL storage", "error", err)
			} else {
				// Use the existing pool connection
				alertStorage = pgStorage

				// TN-038: Initialize Alert History Repository with analytics
				historyRepo = repository.NewPostgresHistoryRepository(pool.Pool(), alertStorage, appLogger)
				slog.Info("✅ Alert History Repository initialized (with analytics: top alerts, flapping detection)")

				// TN-181: Initialize DB Pool Metrics Exporter (expose internal atomic metrics to Prometheus)
				dbMetrics := metricsRegistry.Infra().DB
				dbExporter := postgres.NewPrometheusExporter(pool, dbMetrics)
				dbExporter.Start(context.Background(), 10*time.Second) // Export every 10 seconds
				slog.Info("✅ DB Pool Metrics Exporter started",
					"interval", "10s",
					"metrics", []string{"connections_active", "connections_idle", "query_duration", "errors"})

				// Cleanup DB exporter on shutdown (add to graceful shutdown)
				defer dbExporter.Stop()
			}
		}
	}

	// Initialize Redis cache for enrichment mode
	var redisCache cache.Cache
	if cfg.Redis.Addr != "" {
		slog.Info("Initializing Redis cache", "addr", cfg.Redis.Addr)
		cacheConfig := cache.CacheConfig{
			Addr:                  cfg.Redis.Addr,
			Password:              cfg.Redis.Password,
			DB:                    cfg.Redis.DB,
			PoolSize:              cfg.Redis.PoolSize,
			MinIdleConns:          cfg.Redis.MinIdleConns,
			DialTimeout:           cfg.Redis.DialTimeout,
			ReadTimeout:           cfg.Redis.ReadTimeout,
			WriteTimeout:          cfg.Redis.WriteTimeout,
			MaxRetries:            cfg.Redis.MaxRetries,
			MinRetryBackoff:       cfg.Redis.MinRetryBackoff,
			MaxRetryBackoff:       cfg.Redis.MaxRetryBackoff,
			CircuitBreakerEnabled: true,
			MetricsEnabled:        cfg.Metrics.Enabled,
		}

		var err error
		redisCache, err = cache.NewRedisCache(&cacheConfig, appLogger)
		if err != nil {
			slog.Warn("Failed to initialize Redis cache, enrichment mode will fallback to ENV/default",
				"error", err)
			redisCache = nil
		} else {
			// Test connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := redisCache.Ping(ctx); err != nil {
				slog.Warn("Redis ping failed, enrichment mode will fallback to ENV/default",
					"error", err)
				redisCache = nil
			} else {
				slog.Info("✅ Redis cache initialized successfully")
			}
		}
	} else {
		slog.Info("Redis not configured, enrichment mode will use ENV/default fallback")
	}

	// Initialize Prometheus metrics if enabled (legacy MetricsManager for backward compatibility)
	var metricsManager *metrics.MetricsManager
	var businessMetrics *metrics.BusinessMetrics // TN-124: For grouping system
	if cfg.Metrics.Enabled {
		slog.Info("Initializing Prometheus metrics", "path", cfg.Metrics.Path)
		metricsConfig := metrics.Config{
			Enabled:   true,
			Namespace: "alert_history",
			Subsystem: "http",
		}
		metricsManager = metrics.NewMetricsManager(metricsConfig)

		// TN-124: Create BusinessMetrics for alert grouping system
		businessMetrics = metrics.NewBusinessMetrics("alert_history")
		slog.Info("✅ Business metrics initialized for grouping system")
	}

	// Initialize Enrichment Mode Manager
	slog.Info("Initializing Enrichment Mode Manager")
	enrichmentManager := services.NewEnrichmentModeManager(redisCache, appLogger, metricsManager)

	// Get and log current enrichment mode
	ctx := context.Background()
	currentMode, source, err := enrichmentManager.GetModeWithSource(ctx)
	if err != nil {
		slog.Warn("Failed to get initial enrichment mode", "error", err)
	} else {
		slog.Info("✅ Enrichment Mode Manager initialized",
			"mode", currentMode,
			"source", source)
	}

	// Create enrichment handlers
	enrichmentHandlers := handlers.NewEnrichmentHandlers(enrichmentManager, appLogger)

	// TN-121/122/123/124: Initialize Alert Grouping System
	var groupManager grouping.AlertGroupManager
	var timerManager grouping.GroupTimerManager

	// Check if we have a grouping config file
	if groupingConfigPath := os.Getenv("GROUPING_CONFIG_PATH"); groupingConfigPath != "" || true {
		// Use default path if not set
		if groupingConfigPath == "" {
			groupingConfigPath = "./config/grouping.yaml"
		}

		// Check if file exists
		if _, err := os.Stat(groupingConfigPath); err == nil {
			slog.Info("Initializing Alert Grouping System", "config", groupingConfigPath)

			// TN-121: Parse grouping configuration
			parser := grouping.NewParser()
			groupingConfig, err := parser.ParseFile(groupingConfigPath)
			if err != nil {
				slog.Warn("Failed to parse grouping config, grouping disabled", "error", err)
			} else {
				// TN-122: Create Group Key Generator (with default options)
				keyGenerator := grouping.NewGroupKeyGenerator(
					grouping.WithHashLongKeys(true),
					grouping.WithMaxKeyLength(256),
				)

				// TN-124: Create Timer Storage (Redis or in-memory fallback)
				var timerStorage grouping.TimerStorage
				if redisCache != nil {
					timerStorage, err = grouping.NewRedisTimerStorage(redisCache, appLogger)
					if err != nil {
						slog.Warn("Failed to create Redis timer storage, using in-memory fallback", "error", err)
						timerStorage = grouping.NewInMemoryTimerStorage(appLogger)
					} else {
						slog.Info("✅ Redis Timer Storage initialized")
					}
				} else {
					timerStorage = grouping.NewInMemoryTimerStorage(appLogger)
					slog.Info("Using in-memory timer storage (Redis not available)")
				}

				// TN-123: Create Alert Group Manager
				groupManagerCtx := context.Background()
				groupManager, err = grouping.NewDefaultGroupManager(groupManagerCtx, grouping.DefaultGroupManagerConfig{
					KeyGenerator: keyGenerator,
					Config:       groupingConfig,
					Logger:       appLogger,
					Metrics:      businessMetrics, // Now we have BusinessMetrics!
				})
				if err != nil {
					slog.Error("Failed to create group manager", "error", err)
				} else {
					slog.Info("✅ Alert Group Manager initialized")

					// TN-124: Create Timer Manager
					// Get concrete type for TimerManager (requires *DefaultGroupManager)
					concreteGroupManager, ok := groupManager.(*grouping.DefaultGroupManager)
					if !ok {
						slog.Warn("⚠️  Timer Manager initialization skipped (groupManager is not *DefaultGroupManager)")
					} else {
						timerManager, err = grouping.NewDefaultTimerManager(grouping.TimerManagerConfig{
							Storage:               timerStorage,
							GroupManager:          concreteGroupManager,
							DefaultGroupWait:      30 * time.Second,
							DefaultGroupInterval:  5 * time.Minute,
							DefaultRepeatInterval: 4 * time.Hour,
							Logger:                appLogger,
						})
						if err != nil {
							slog.Error("Failed to create timer manager", "error", err)
						} else {
							// Restore timers after restart (HA)
							restored, missed, err := timerManager.RestoreTimers(ctx)
							if err != nil {
								slog.Warn("Failed to restore timers", "error", err)
							} else {
								slog.Info("✅ Alert Grouping System fully initialized",
									"timers_restored", restored,
									"timers_missed", missed)
							}
						}
					}
				}
			}
		} else {
			slog.Info("Grouping config not found, grouping disabled", "path", groupingConfigPath)
		}
	}

	// TN-130: Initialize Inhibition Rules Engine (Module 2 - API Layer)
	var inhibitionParser inhibition.InhibitionParser
	var inhibitionMatcher inhibition.InhibitionMatcher
	var inhibitionStateManager inhibition.InhibitionStateManager
	var activeAlertCache inhibition.ActiveAlertCache
	var inhibitionHandler *handlers.InhibitionHandler

	// Check if we have an inhibition config file
	if inhibitionConfigPath := os.Getenv("INHIBITION_CONFIG_PATH"); inhibitionConfigPath != "" || true {
		// Use default path if not set
		if inhibitionConfigPath == "" {
			inhibitionConfigPath = "./config/inhibition.yaml"
		}

		// Check if file exists
		if _, err := os.Stat(inhibitionConfigPath); err == nil {
			slog.Info("Initializing Inhibition Rules Engine (TN-130)", "config", inhibitionConfigPath)

			// TN-126: Parse inhibition configuration
			inhibitionParser = inhibition.NewParser()
			config, err := inhibitionParser.ParseFile(inhibitionConfigPath)
			if err != nil {
				slog.Warn("Failed to parse inhibition config, inhibition disabled", "error", err)
			} else {
				slog.Info("✅ Loaded inhibition rules", "count", len(config.Rules))

				// TN-128: Create Active Alert Cache (L1 memory + L2 Redis)
				if redisCache != nil {
					activeAlertCache = inhibition.NewTwoTierAlertCache(redisCache, appLogger)
					slog.Info("✅ Active Alert Cache initialized (L1 memory + L2 Redis)")
				} else {
					activeAlertCache = inhibition.NewTwoTierAlertCache(nil, appLogger)
					slog.Info("Active Alert Cache initialized (memory-only, Redis not available)")
				}

				// Note: TwoTierAlertCache doesn't have cleanup worker (uses LRU eviction)

				// TN-127: Create Inhibition Matcher
				inhibitionMatcher = inhibition.NewMatcher(
					activeAlertCache,
					config.Rules,
					appLogger,
				)
				slog.Info("✅ Inhibition Matcher initialized", "rules", len(config.Rules))

				// TN-129: Create Inhibition State Manager
				inhibitionStateManager = inhibition.NewDefaultStateManager(
					redisCache,       // Redis for persistence (optional)
					appLogger,
					businessMetrics,  // Metrics
				)
				stateCleanupCtx := context.Background()
				inhibitionStateManager.(*inhibition.DefaultStateManager).StartCleanupWorker(stateCleanupCtx)
				defer inhibitionStateManager.(*inhibition.DefaultStateManager).StopCleanupWorker()
				slog.Info("✅ Inhibition State Manager initialized (Redis persistence + cleanup worker)")

				// TN-130: Create Inhibition API Handler
				inhibitionHandler = handlers.NewInhibitionHandler(
					inhibitionParser,
					inhibitionMatcher,
					inhibitionStateManager,
					businessMetrics,
					appLogger,
				)
				slog.Info("✅ Inhibition Handler initialized (ready for API endpoints)")
			}
		} else {
			slog.Info("Inhibition config not found, inhibition disabled", "path", inhibitionConfigPath)
		}
	}

	// Initialize filter engine and publisher
	filterEngine := services.NewSimpleFilterEngine(appLogger)
	publisher := services.NewSimplePublisher(appLogger)

	// TN-036 Phase 3: Initialize Deduplication Service
	var deduplicationService services.DeduplicationService
	if alertStorage != nil {
		slog.Info("Initializing Deduplication Service (TN-036)")
		fingerprintGen := services.NewFingerprintGenerator(&services.FingerprintConfig{
			Algorithm: services.AlgorithmFNV1a,
		})
		dedupConfig := &services.DeduplicationConfig{
			Storage:         alertStorage,
			Fingerprint:     fingerprintGen,
			Logger:          appLogger,
			BusinessMetrics: metricsRegistry.Business(), // TN-036: BusinessMetrics integration
		}
		var err error
		deduplicationService, err = services.NewDeduplicationService(dedupConfig)
		if err != nil {
			slog.Error("Failed to create deduplication service", "error", err)
			// Continue without deduplication (graceful degradation)
			deduplicationService = nil
		} else {
			slog.Info("✅ Deduplication Service initialized",
				"algorithm", "FNV-1a (Alertmanager-compatible)",
				"metrics", "enabled (4 Prometheus metrics)")
		}
	} else {
		slog.Warn("⚠️ Deduplication Service NOT initialized (database not available)")
	}

	// TN-033: Initialize Classification Service with two-tier caching
	var classificationService services.ClassificationService
	if cfg.LLM.Enabled {
		slog.Info("Initializing LLM Classification Service (TN-033)")

		// Initialize LLM client
		llmConfig := llm.Config{
			BaseURL:    cfg.LLM.BaseURL,
			APIKey:     cfg.LLM.APIKey,
			Model:      cfg.LLM.Model,
			Timeout:    cfg.LLM.Timeout,
			MaxRetries: cfg.LLM.MaxRetries,
		}
		llmClient := llm.NewHTTPLLMClient(llmConfig, appLogger)

		// Create classification service config
		classificationConfig := services.ClassificationServiceConfig{
			LLMClient:       llmClient,
			Cache:           redisCache,
			Storage:         alertStorage,
			Config:          services.DefaultClassificationConfig(),
			BusinessMetrics: metricsRegistry.Business(),
		}

		var err error
		classificationService, err = services.NewClassificationService(classificationConfig)
		if err != nil {
			slog.Error("Failed to create classification service", "error", err)
			// Continue without classification (graceful degradation)
			classificationService = nil
		} else {
			slog.Info("✅ Classification Service initialized",
				"features", []string{
					"Two-tier caching (L1 memory + L2 Redis)",
					"Intelligent fallback",
					"Batch processing",
					"6 Prometheus metrics",
				})
		}
	} else {
		slog.Info("LLM not enabled, classification service will not be available")
	}

	// Initialize AlertProcessor
	alertProcessorConfig := services.AlertProcessorConfig{
		EnrichmentManager: enrichmentManager,
		LLMClient:         classificationService,  // TN-033: ClassificationService with caching + fallback
		FilterEngine:      filterEngine,
		Publisher:         publisher,
		Deduplication:     deduplicationService,   // TN-036 Phase 3
		InhibitionMatcher: inhibitionMatcher,      // TN-130 Phase 6: Inhibition checking
		InhibitionState:   inhibitionStateManager, // TN-130 Phase 6: State tracking
		BusinessMetrics:   businessMetrics,        // TN-130 Phase 6: Business metrics
		Logger:            appLogger,
		Metrics:           metricsManager,
	}

	alertProcessor, err := services.NewAlertProcessor(alertProcessorConfig)
	if err != nil {
		slog.Error("Failed to create alert processor", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Alert Processor initialized successfully")

	// Create webhook handlers
	webhookHandlers := handlers.NewWebhookHandlers(alertProcessor, appLogger)

	// TN-038: Initialize History Handlers V2 with analytics support
	var historyHandlerV2 *handlers.HistoryHandlerV2
	if historyRepo != nil {
		historyHandlerV2 = handlers.NewHistoryHandlerV2(historyRepo, appLogger)
		slog.Info("✅ History Handlers V2 initialized with analytics support")
	}

	// Setup HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.HealthHandler)
	mux.HandleFunc("/webhook", webhookHandlers.HandleWebhook)

	// Legacy history endpoint (for backward compatibility)
	mux.HandleFunc("/history", handlers.HistoryHandler)

	// TN-038: Register analytics endpoints (if historyRepo available)
	if historyHandlerV2 != nil {
		mux.HandleFunc("/history/top", historyHandlerV2.HandleTopAlerts)
		mux.HandleFunc("/history/flapping", historyHandlerV2.HandleFlappingAlerts)
		mux.HandleFunc("/history/stats", historyHandlerV2.HandleStats)
		mux.HandleFunc("/history/recent", historyHandlerV2.HandleRecentAlerts)
		slog.Info("✅ Analytics endpoints registered",
			"endpoints", []string{
				"GET /history/top - Top firing alerts",
				"GET /history/flapping - Flapping detection",
				"GET /history/stats - Aggregated statistics",
				"GET /history/recent - Recent alerts",
			})
	} else {
		slog.Warn("⚠️ Analytics endpoints NOT available (database not connected or MOCK_MODE)")
	}

	// Register enrichment mode endpoints
	mux.HandleFunc("/enrichment/mode", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			enrichmentHandlers.GetMode(w, r)
		case http.MethodPost:
			enrichmentHandlers.SetMode(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	slog.Info("Enrichment mode endpoints enabled",
		"get_path", "GET /enrichment/mode",
		"set_path", "POST /enrichment/mode")

	// TN-130: Register Inhibition API endpoints (Alertmanager compatible)
	if inhibitionHandler != nil {
		mux.HandleFunc("GET /api/v2/inhibition/rules", inhibitionHandler.GetRules)
		mux.HandleFunc("GET /api/v2/inhibition/status", inhibitionHandler.GetStatus)
		mux.HandleFunc("POST /api/v2/inhibition/check", inhibitionHandler.CheckAlert)
		slog.Info("✅ Inhibition API endpoints registered",
			"endpoints", []string{
				"GET /api/v2/inhibition/rules - List all inhibition rules",
				"GET /api/v2/inhibition/status - Get active inhibition relationships",
				"POST /api/v2/inhibition/check - Check if alert would be inhibited",
			})
	} else {
		slog.Info("Inhibition API endpoints NOT available (config not found or initialization failed)")
	}

	// Add Prometheus metrics endpoint if enabled
	if cfg.Metrics.Enabled {
		mux.Handle(cfg.Metrics.Path, promhttp.Handler())
		slog.Info("Prometheus metrics endpoint enabled", "path", cfg.Metrics.Path)
	}

	// Add pprof endpoints for performance profiling
	// These endpoints are automatically registered by importing net/http/pprof
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)
	mux.HandleFunc("/debug/pprof/cmdline", http.DefaultServeMux.ServeHTTP)
	mux.HandleFunc("/debug/pprof/profile", http.DefaultServeMux.ServeHTTP)
	mux.HandleFunc("/debug/pprof/symbol", http.DefaultServeMux.ServeHTTP)
	mux.HandleFunc("/debug/pprof/trace", http.DefaultServeMux.ServeHTTP)
	slog.Info("pprof endpoints enabled for performance profiling", "base_path", "/debug/pprof/")

	slog.Info("Webhook endpoint enabled", "path", "/webhook")

	// Add middleware chain
	var handler http.Handler = mux

	// TN-181: Add Path Normalization middleware (reduce cardinality in HTTP metrics)
	// Must come before metrics middleware to normalize paths before recording
	pathNormalizer := pkgmiddleware.NewPathNormalizer()
	handler = pathNormalizer.Middleware()(handler)
	slog.Info("✅ Path Normalization middleware added (reduces cardinality for HTTP metrics)")

	// Add enrichment mode middleware (adds mode to context and response headers)
	enrichmentMiddleware := middleware.NewEnrichmentModeMiddleware(enrichmentManager, appLogger)
	handler = enrichmentMiddleware.Middleware(handler)

	// Add Prometheus metrics middleware if enabled
	if cfg.Metrics.Enabled && metricsManager != nil {
		handler = metricsManager.Middleware(handler)
	}

	// Add logging middleware
	handler = logger.LoggingMiddleware(appLogger)(handler)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register interrupt signals
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		slog.Info("HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("Shutting down server...")

	// Create context with timeout for graceful shutdown from config
	shutdownTimeout := cfg.Server.GracefulShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Graceful shutdown
	// TN-124: Shutdown timer manager first (if initialized)
	if timerManager != nil {
		slog.Info("Shutting down timer manager...")
		if err := timerManager.Shutdown(ctx); err != nil {
			slog.Error("Timer manager shutdown error", "error", err)
		} else {
			slog.Info("✅ Timer manager stopped")
		}
	}

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	close(done)
	slog.Info("Server exited")
}
