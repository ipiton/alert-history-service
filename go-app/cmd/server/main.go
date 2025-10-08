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

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
	"github.com/vitaliisemenov/alert-history/internal/database"
	"github.com/vitaliisemenov/alert-history/internal/database/postgres"
	"github.com/vitaliisemenov/alert-history/pkg/logger"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	if useMockMode {
		slog.Info("🚀 Running in MOCK MODE for performance testing - no database required")
	} else {
		pool := postgres.NewPostgresPool(dbCfg, appLogger)

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
		}
	}

	// Initialize Prometheus metrics if enabled
	var metricsManager *metrics.MetricsManager
	if cfg.Metrics.Enabled {
		slog.Info("Initializing Prometheus metrics", "path", cfg.Metrics.Path)
		metricsConfig := metrics.Config{
			Namespace: "alert_history",
			Subsystem: "http",
		}
		metricsManager = metrics.NewMetricsManager(metricsConfig)
	}

	// Setup HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.HealthHandler)
	mux.HandleFunc("/webhook", handlers.WebhookHandler)
	mux.HandleFunc("/history", handlers.HistoryHandler)

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
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	close(done)
	slog.Info("Server exited")
}
