// Package main is the entry point for Alert History Service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
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
	var configPathFlag = flag.String("config", "", "Path to config file (YAML)")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("%s version %s\n", serviceName, serviceVersion)
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("Alert History Service - Intelligent Alert Proxy\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		fmt.Printf("  -version           Show version information\n")
		fmt.Printf("  -help              Show this help message\n")
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
	pool := postgres.NewPostgresPool(dbCfg, appLogger)

	ctx := context.Background()
	if err := pool.Connect(ctx); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Successfully connected to PostgreSQL!")

	// Run database migrations
	if err := database.RunMigrations(ctx, pool, appLogger); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		// Не завершаем работу, если миграции не удались - даем возможность ручного исправления
		slog.Warn("Continuing without migrations - manual intervention may be required")
	} else {
		slog.Info("✅ Database migrations completed successfully")
	}

	// Setup HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.HealthHandler)

	// Add logging middleware
	loggedMux := logger.LoggingMiddleware(appLogger)(mux)

	server := &http.Server{
		Addr:         addr,
		Handler:      loggedMux,
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
