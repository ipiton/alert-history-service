package security

import (
	"log/slog"
	"net/http"
)

// SecurityMiddleware combines all security middleware
type SecurityMiddleware struct {
	inputValidator   *InputValidator
	requestSizeLimit *RequestSizeLimiter
	auditLogger      *AuditLogger
	logger           *slog.Logger
}

// SecurityConfig contains security middleware configuration
type SecurityConfig struct {
	EnableInputValidation bool
	EnableSizeLimiting    bool
	EnableAuditLogging    bool
	MaxRequestSize        int64 // Max request body size in bytes
	ValidatorConfig       ValidatorConfig
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableInputValidation: true,
		EnableSizeLimiting:    true,
		EnableAuditLogging:    true,
		MaxRequestSize:        10 * 1024 * 1024, // 10MB
		ValidatorConfig:       DefaultValidatorConfig(),
	}
}

// NewSecurityMiddleware creates a new security middleware stack
func NewSecurityMiddleware(logger *slog.Logger, config SecurityConfig) *SecurityMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	
	mw := &SecurityMiddleware{
		logger: logger,
	}
	
	if config.EnableInputValidation {
		mw.inputValidator = NewInputValidator(logger, config.ValidatorConfig)
	}
	
	if config.EnableSizeLimiting {
		mw.requestSizeLimit = NewRequestSizeLimiter(config.MaxRequestSize, logger)
	}
	
	if config.EnableAuditLogging {
		mw.auditLogger = NewAuditLogger(logger)
	}
	
	return mw
}

// Apply applies all security middleware to a handler
func (s *SecurityMiddleware) Apply(handler http.Handler) http.Handler {
	// Order matters:
	// 1. Request size limiting (before reading body)
	if s.requestSizeLimit != nil {
		handler = s.requestSizeLimit.Middleware()(handler)
	}
	
	// 2. Input validation (after size check)
	if s.inputValidator != nil {
		handler = s.inputValidator.Middleware()(handler)
	}
	
	// 3. Audit logging wrapper (logs all requests)
	if s.auditLogger != nil {
		handler = s.auditLoggingMiddleware(handler)
	}
	
	return handler
}

// auditLoggingMiddleware wraps handler with audit logging
func (s *SecurityMiddleware) auditLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(rw, r)
		
		// Log security events based on status code
		ctx := r.Context()
		if rw.statusCode == http.StatusUnauthorized {
			s.auditLogger.LogAuthenticationFailure(ctx, r, "Authentication failed")
		} else if rw.statusCode == http.StatusForbidden {
			// Try to get user ID from context or header
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = "unknown"
			}
			s.auditLogger.LogAuthorizationFailure(ctx, r, userID, "Authorization failed")
		} else if rw.statusCode == http.StatusTooManyRequests {
			s.auditLogger.LogRateLimitViolation(ctx, r, 0, 0)
		} else if rw.statusCode == http.StatusBadRequest {
			s.auditLogger.LogInputValidationFailure(ctx, r, "Input validation failed")
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetAuditLogger returns the audit logger
func (s *SecurityMiddleware) GetAuditLogger() *AuditLogger {
	return s.auditLogger
}

// GetInputValidator returns the input validator
func (s *SecurityMiddleware) GetInputValidator() *InputValidator {
	return s.inputValidator
}

