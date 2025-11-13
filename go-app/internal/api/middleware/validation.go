package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidationMiddleware validates request body using struct tags
// This is a basic implementation - actual validation happens in handlers
func ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation for GET, DELETE methods (no body expected)
		if r.Method == http.MethodGet || r.Method == http.MethodDelete || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Check Content-Type for POST/PUT/PATCH
		contentType := r.Header.Get("Content-Type")
		if contentType != "" && contentType != "application/json" {
			writeValidationError(w, r, "Content-Type must be application/json")
			return
		}

		// Check Content-Length (max 1MB)
		const maxRequestSize = 1 << 20 // 1MB
		if r.ContentLength > maxRequestSize {
			writeValidationError(w, r, "Request body too large (max 1MB)")
			return
		}

		// Validation of actual request body happens in handlers
		// using ValidateStruct() function
		next.ServeHTTP(w, r)
	})
}

// ValidateStruct validates a struct using validator tags
//
// Example usage in handler:
//
//	type CreateAlertRequest struct {
//	    Fingerprint string `json:"fingerprint" validate:"required,min=1,max=128"`
//	    Severity    string `json:"severity" validate:"required,oneof=critical high medium low info"`
//	}
//
//	var req CreateAlertRequest
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//	    return err
//	}
//	if err := middleware.ValidateStruct(req); err != nil {
//	    return err
//	}
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
	Hint  string `json:"hint,omitempty"`
}

// FormatValidationErrors converts validator errors to ValidationError slice
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field: e.Field(),
				Issue: e.Tag(),
				Hint:  getValidationHint(e),
			})
		}
	}

	return errors
}

// getValidationHint returns a human-readable hint for validation error
func getValidationHint(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Must be at least " + e.Param() + " characters"
	case "max":
		return "Must be at most " + e.Param() + " characters"
	case "oneof":
		return "Must be one of: " + e.Param()
	case "uuid":
		return "Must be a valid UUID"
	case "url":
		return "Must be a valid URL"
	default:
		return "Validation failed: " + e.Tag()
	}
}

// writeValidationError writes validation error response
func writeValidationError(w http.ResponseWriter, r *http.Request, message string) {
	requestID := GetRequestID(r.Context())
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":       "VALIDATION_ERROR",
			"message":    message,
			"request_id": requestID,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(errorResponse)
}
