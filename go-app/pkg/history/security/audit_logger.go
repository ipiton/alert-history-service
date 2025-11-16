package security

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
)

// AuditLogger logs security-relevant events
type AuditLogger struct {
	logger *slog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &AuditLogger{
		logger: logger,
	}
}

// LogSecurityEvent logs a security event
func (a *AuditLogger) LogSecurityEvent(ctx context.Context, event SecurityEvent) {
	requestID := middleware.GetRequestID(ctx)
	
	attrs := []interface{}{
		"event_type", event.Type,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"request_id", requestID,
		"severity", event.Severity,
	}
	
	if event.UserID != "" {
		attrs = append(attrs, "user_id", event.UserID)
	}
	if event.IP != "" {
		attrs = append(attrs, "ip", event.IP)
	}
	if event.Endpoint != "" {
		attrs = append(attrs, "endpoint", event.Endpoint)
	}
	if event.StatusCode > 0 {
		attrs = append(attrs, "status_code", event.StatusCode)
	}
	if event.Message != "" {
		attrs = append(attrs, "message", event.Message)
	}
	if event.Details != nil {
		attrs = append(attrs, "details", event.Details)
	}
	
	switch event.Severity {
	case "critical":
		a.logger.Error("Security event", attrs...)
	case "high":
		a.logger.Warn("Security event", attrs...)
	default:
		a.logger.Info("Security event", attrs...)
	}
}

// LogAuthenticationFailure logs authentication failure
func (a *AuditLogger) LogAuthenticationFailure(ctx context.Context, r *http.Request, reason string) {
	a.LogSecurityEvent(ctx, SecurityEvent{
		Type:       "authentication_failure",
		Severity:   "high",
		IP:         a.extractIP(r),
		Endpoint:   r.URL.Path,
		StatusCode: http.StatusUnauthorized,
		Message:    reason,
	})
}

// LogAuthorizationFailure logs authorization failure
func (a *AuditLogger) LogAuthorizationFailure(ctx context.Context, r *http.Request, userID, reason string) {
	a.LogSecurityEvent(ctx, SecurityEvent{
		Type:       "authorization_failure",
		Severity:   "high",
		UserID:     userID,
		IP:         a.extractIP(r),
		Endpoint:   r.URL.Path,
		StatusCode: http.StatusForbidden,
		Message:    reason,
	})
}

// LogRateLimitViolation logs rate limit violation
func (a *AuditLogger) LogRateLimitViolation(ctx context.Context, r *http.Request, limit, remaining int) {
	a.LogSecurityEvent(ctx, SecurityEvent{
		Type:       "rate_limit_violation",
		Severity:   "medium",
		IP:         a.extractIP(r),
		Endpoint:   r.URL.Path,
		StatusCode: http.StatusTooManyRequests,
		Message:    "Rate limit exceeded",
		Details: map[string]interface{}{
			"limit":     limit,
			"remaining": remaining,
		},
	})
}

// LogInputValidationFailure logs input validation failure
func (a *AuditLogger) LogInputValidationFailure(ctx context.Context, r *http.Request, reason string) {
	a.LogSecurityEvent(ctx, SecurityEvent{
		Type:       "input_validation_failure",
		Severity:   "medium",
		IP:         a.extractIP(r),
		Endpoint:   r.URL.Path,
		StatusCode: http.StatusBadRequest,
		Message:    reason,
	})
}

// LogSuspiciousActivity logs suspicious activity
func (a *AuditLogger) LogSuspiciousActivity(ctx context.Context, r *http.Request, activity string, details map[string]interface{}) {
	a.LogSecurityEvent(ctx, SecurityEvent{
		Type:       "suspicious_activity",
		Severity:   "high",
		IP:         a.extractIP(r),
		Endpoint:   r.URL.Path,
		Message:    activity,
		Details:    details,
	})
}

// extractIP extracts client IP from request
func (a *AuditLogger) extractIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP (client IP)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// SecurityEvent represents a security event
type SecurityEvent struct {
	Type       string
	Severity   string // critical, high, medium, low
	UserID     string
	IP         string
	Endpoint   string
	StatusCode int
	Message    string
	Details    map[string]interface{}
}

