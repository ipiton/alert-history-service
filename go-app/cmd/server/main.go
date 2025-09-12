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
	"syscall"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
	"github.com/vitaliisemenov/alert-history/internal/database"
	"github.com/vitaliisemenov/alert-history/internal/database/postgres"
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
		fmt.Printf("  -version    Show version information\n")
		fmt.Printf("  -help       Show this help message\n\n")
		fmt.Printf("Environment variables:\n")
		fmt.Printf("  PORT        HTTP server port (default: %s)\n\n", defaultPort)
		os.Exit(0)
	}

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Alert History Service",
		"service", serviceName,
		"version", serviceVersion,
	)

	// Initialize database connection and run migrations
	slog.Info("Initializing database connection...")
	config := postgres.LoadFromEnv()
	pool := postgres.NewPostgresPool(config, logger)

	ctx := context.Background()
	if err := pool.Connect(ctx); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Successfully connected to PostgreSQL!")

	// Run database migrations
	if err := database.RunMigrations(ctx, pool, logger); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		// Не завершаем работу, если миграции не удались - даем возможность ручного исправления
		slog.Warn("Continuing without migrations - manual intervention may be required")
	} else {
		slog.Info("✅ Database migrations completed successfully")
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.HealthHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register interrupt signals
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		slog.Info("HTTP server starting", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("Shutting down server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	close(done)
	slog.Info("Server exited")
}
