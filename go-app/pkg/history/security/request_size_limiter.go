package security

import (
	"log/slog"
	"net/http"
	
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
)

// RequestSizeLimiter limits request body size
type RequestSizeLimiter struct {
	maxSize int64
	logger  *slog.Logger
}

// NewRequestSizeLimiter creates a new request size limiter
func NewRequestSizeLimiter(maxSize int64, logger *slog.Logger) *RequestSizeLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &RequestSizeLimiter{
		maxSize: maxSize,
		logger:  logger,
	}
}

// Middleware returns HTTP middleware for request size limiting
func (r *RequestSizeLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Check Content-Length header
			if req.ContentLength > r.maxSize {
				r.logger.Warn("Request body too large",
					"content_length", req.ContentLength,
					"max_size", r.maxSize,
					"path", req.URL.Path)
				apierrors.WriteError(w, apierrors.ValidationError("Request body too large"))
				return
			}
			
			// Limit request body reader
			req.Body = http.MaxBytesReader(w, req.Body, r.maxSize)
			
			next.ServeHTTP(w, req)
		})
	}
}

