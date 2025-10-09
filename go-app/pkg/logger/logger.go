// Package logger provides structured logging utilities for the Alert History Service.
package logger

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger configuration.
type Config struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// NewLogger creates a new structured logger with the given configuration.
func NewLogger(cfg Config) *slog.Logger {
	// Parse log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure output writer
	var writer io.Writer
	switch cfg.Output {
	case "file":
		if cfg.Filename == "" {
			cfg.Filename = "app.log"
		}
		writer = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups, // number of backups
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // compress rotated files
		}
	case "stdout":
		fallthrough
	default:
		writer = os.Stdout
	}

	// Configure handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	case "json":
		fallthrough
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	return slog.New(handler)
}

// LoggingMiddleware returns an HTTP middleware that logs requests.
func LoggingMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
			next.ServeHTTP(w, r)
		})
	}
}
