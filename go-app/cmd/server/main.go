// Package main is the entry point for Alert History Service.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/handlers"
)

const (
	defaultPort = "8080"
	serviceName = "alert-history"
	serviceVersion = "1.0.0"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Alert History Service",
		"service", serviceName,
		"version", serviceVersion,
	)

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
