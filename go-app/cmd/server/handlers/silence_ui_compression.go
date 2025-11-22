// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

// CompressionMiddleware provides gzip compression for UI responses.
// Phase 10: Performance Optimization enhancement.
type CompressionMiddleware struct {
	pool sync.Pool
	logger *slog.Logger
}

// NewCompressionMiddleware creates a new compression middleware.
func NewCompressionMiddleware(logger *slog.Logger) *CompressionMiddleware {
	return &CompressionMiddleware{
		pool: sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(nil)
			},
		},
		logger: logger,
	}
}

// gzipResponseWriter wraps http.ResponseWriter to add gzip compression.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Middleware returns a middleware function that compresses responses.
func (cm *CompressionMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip encoding
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next(w, r)
			return
		}

		// Check content type (only compress HTML, CSS, JS, JSON)
		contentType := r.Header.Get("Content-Type")
		if contentType != "" {
			compressible := strings.HasPrefix(contentType, "text/html") ||
				strings.HasPrefix(contentType, "text/css") ||
				strings.HasPrefix(contentType, "application/javascript") ||
				strings.HasPrefix(contentType, "application/json") ||
				strings.HasPrefix(contentType, "text/plain")

			if !compressible {
				next(w, r)
				return
			}
		}

		// Get gzip writer from pool
		gz := cm.pool.Get().(*gzip.Writer)
		defer cm.pool.Put(gz)
		defer gz.Close()

		// Reset writer
		gz.Reset(w)

		// Set headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		// Wrap response writer
		gzw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
		}

		// Call next handler
		next(gzw, r)
	}
}

// EnableCompression wraps a handler with compression middleware.
func (h *SilenceUIHandler) EnableCompression(next http.HandlerFunc) http.HandlerFunc {
	if h.compressionMiddleware == nil {
		h.compressionMiddleware = NewCompressionMiddleware(h.logger)
	}
	return h.compressionMiddleware.Middleware(next)
}
