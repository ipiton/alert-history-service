package security

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
)

// InputValidator validates and sanitizes input parameters
type InputValidator struct {
	logger *slog.Logger
	config ValidatorConfig
}

// ValidatorConfig contains validation configuration
type ValidatorConfig struct {
	MaxQueryParamLength int           // Max length for query parameters
	MaxTimeRangeDays    int           // Max time range in days
	MaxLabelCount       int           // Max number of labels
	MaxPage             int           // Max page number
	MaxPerPage          int           // Max per_page value
	AllowedSortFields   []string      // Allowed sort fields
	RegexTimeout        time.Duration // Regex compilation timeout
}

// DefaultValidatorConfig returns default validation configuration
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		MaxQueryParamLength: 1000,
		MaxTimeRangeDays:    90,
		MaxLabelCount:       20,
		MaxPage:             10000,
		MaxPerPage:          1000,
		AllowedSortFields:   []string{"starts_at", "ends_at", "created_at", "status", "severity"},
		RegexTimeout:        5 * time.Second,
	}
}

// NewInputValidator creates a new input validator
func NewInputValidator(logger *slog.Logger, config ValidatorConfig) *InputValidator {
	if logger == nil {
		logger = slog.Default()
	}

	return &InputValidator{
		logger: logger,
		config: config,
	}
}

// ValidateQueryParams validates query parameters
func (v *InputValidator) ValidateQueryParams(r *http.Request) error {
	queryParams := r.URL.Query()
	requestID := middleware.GetRequestID(r.Context())

	// Validate page parameter
	if pageStr := queryParams.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 || page > v.config.MaxPage {
			return apierrors.ValidationError(fmt.Sprintf("page must be between 1 and %d", v.config.MaxPage)).WithRequestID(requestID)
		}
	}

	// Validate per_page parameter
	if perPageStr := queryParams.Get("per_page"); perPageStr != "" {
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage < 1 || perPage > v.config.MaxPerPage {
			return apierrors.ValidationError(fmt.Sprintf("per_page must be between 1 and %d", v.config.MaxPerPage)).WithRequestID(requestID)
		}
	}

	// Validate sort_field
	if sortField := queryParams.Get("sort_field"); sortField != "" {
		allowed := false
		for _, field := range v.config.AllowedSortFields {
			if field == sortField {
				allowed = true
				break
			}
		}
		if !allowed {
			return apierrors.ValidationError(fmt.Sprintf("sort_field must be one of: %s", strings.Join(v.config.AllowedSortFields, ", "))).WithRequestID(requestID)
		}
	}

	// Validate time range
	if fromStr := queryParams.Get("from"); fromStr != "" {
		if err := v.validateTimeParam(fromStr, "from"); err != nil {
			return err
		}
	}
	if toStr := queryParams.Get("to"); toStr != "" {
		if err := v.validateTimeParam(toStr, "to"); err != nil {
			return err
		}
	}

	// Validate time range span
	if fromStr := queryParams.Get("from"); fromStr != "" && queryParams.Get("to") != "" {
		from, err1 := time.Parse(time.RFC3339, fromStr)
		to, err2 := time.Parse(time.RFC3339, queryParams.Get("to"))
		if err1 == nil && err2 == nil {
			days := int(to.Sub(from).Hours() / 24)
			if days > v.config.MaxTimeRangeDays {
				return apierrors.ValidationError(fmt.Sprintf("time range exceeds maximum of %d days", v.config.MaxTimeRangeDays)).WithRequestID(requestID)
			}
		}
	}

	// Validate query parameter lengths
	for key, values := range queryParams {
		for _, value := range values {
			if len(value) > v.config.MaxQueryParamLength {
				return apierrors.ValidationError(fmt.Sprintf("query parameter %s exceeds maximum length of %d", key, v.config.MaxQueryParamLength)).WithRequestID(requestID)
			}

			// Check for suspicious patterns (SQL injection attempts)
			if v.containsSQLInjection(value) {
				v.logger.Warn("Potential SQL injection attempt detected",
					"request_id", requestID,
					"parameter", key,
					"value", v.sanitizeForLog(value))
				return apierrors.ValidationError("Invalid input detected").WithRequestID(requestID)
			}

			// Check for XSS patterns
			if v.containsXSS(value) {
				v.logger.Warn("Potential XSS attempt detected",
					"request_id", requestID,
					"parameter", key,
					"value", v.sanitizeForLog(value))
				return apierrors.ValidationError("Invalid input detected").WithRequestID(requestID)
			}
		}
	}

	// Validate label count
	if labels := queryParams["labels"]; len(labels) > v.config.MaxLabelCount {
		return apierrors.ValidationError(fmt.Sprintf("too many labels: maximum %d allowed", v.config.MaxLabelCount)).WithRequestID(requestID)
	}

	return nil
}

// validateTimeParam validates time parameter format
func (v *InputValidator) validateTimeParam(timeStr, paramName string) error {
	if len(timeStr) > v.config.MaxQueryParamLength {
		return apierrors.ValidationError(fmt.Sprintf("%s parameter too long", paramName))
	}

	_, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return apierrors.ValidationError(fmt.Sprintf("invalid %s format: must be RFC3339 (e.g., 2006-01-02T15:04:05Z07:00)", paramName))
	}

	return nil
}

// containsSQLInjection checks for SQL injection patterns
func (v *InputValidator) containsSQLInjection(input string) bool {
	sqlPatterns := []string{
		"';",
		"'; --",
		"'; /*",
		"UNION SELECT",
		"DROP TABLE",
		"DELETE FROM",
		"INSERT INTO",
		"UPDATE SET",
		"EXEC(",
		"EXECUTE(",
		"xp_cmdshell",
	}

	upperInput := strings.ToUpper(input)
	for _, pattern := range sqlPatterns {
		if strings.Contains(upperInput, strings.ToUpper(pattern)) {
			return true
		}
	}

	return false
}

// containsXSS checks for XSS patterns
func (v *InputValidator) containsXSS(input string) bool {
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onload=",
		"onclick=",
		"<iframe",
		"<img",
		"<svg",
		"eval(",
		"alert(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}

	return false
}

// sanitizeForLog sanitizes input for logging (removes sensitive data)
func (v *InputValidator) sanitizeForLog(input string) string {
	// Truncate long inputs
	if len(input) > 100 {
		return input[:100] + "..."
	}
	return input
}

// ValidateFingerprint validates fingerprint format
func (v *InputValidator) ValidateFingerprint(fingerprint string) error {
	if len(fingerprint) != 64 {
		return apierrors.ValidationError("fingerprint must be 64 hex characters")
	}

	hexPattern := regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	if !hexPattern.MatchString(fingerprint) {
		return apierrors.ValidationError("fingerprint must contain only hex characters")
	}

	return nil
}

// ValidateURL validates URL format
func (v *InputValidator) ValidateURL(urlStr string) error {
	if len(urlStr) > v.config.MaxQueryParamLength {
		return apierrors.ValidationError("URL too long")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return apierrors.ValidationError("invalid URL format")
	}

	// Prevent SSRF attacks - block internal IPs
	if parsed.Hostname() != "" {
		if v.isInternalIP(parsed.Hostname()) {
			return apierrors.ValidationError("internal IPs not allowed")
		}
	}

	return nil
}

// isInternalIP checks if hostname is an internal IP
func (v *InputValidator) isInternalIP(hostname string) bool {
	internalPatterns := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"10.",
		"172.16.",
		"172.17.",
		"172.18.",
		"172.19.",
		"172.20.",
		"172.21.",
		"172.22.",
		"172.23.",
		"172.24.",
		"172.25.",
		"172.26.",
		"172.27.",
		"172.28.",
		"172.29.",
		"172.30.",
		"172.31.",
		"192.168.",
		"169.254.",
	}

	for _, pattern := range internalPatterns {
		if strings.HasPrefix(hostname, pattern) {
			return true
		}
	}

	return false
}

// Middleware returns HTTP middleware for input validation
func (v *InputValidator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := v.ValidateQueryParams(r); err != nil {
				// Check if it's already an APIError
				if apiErr, ok := err.(*apierrors.APIError); ok {
					apierrors.WriteError(w, apiErr)
				} else {
					// Convert to APIError
					requestID := middleware.GetRequestID(r.Context())
					apierrors.WriteError(w, apierrors.ValidationError(err.Error()).WithRequestID(requestID))
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
