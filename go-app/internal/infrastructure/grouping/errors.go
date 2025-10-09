package grouping

import (
	"fmt"
	"strings"
)

// ParseError represents an error that occurred during YAML parsing.
// It includes context about where the error occurred (field, value, line, column).
type ParseError struct {
	// Field is the name of the field that caused the error
	Field string

	// Value is the problematic value that failed to parse
	Value string

	// Line is the line number in the YAML file (1-based)
	Line int

	// Column is the column number in the YAML file (1-based)
	Column int

	// Err is the underlying error
	Err error
}

// Error implements the error interface for ParseError.
// It provides a detailed, user-friendly error message including line/column info if available.
func (e *ParseError) Error() string {
	var b strings.Builder

	b.WriteString("parse error")

	if e.Line > 0 && e.Column > 0 {
		b.WriteString(fmt.Sprintf(" at line %d, column %d", e.Line, e.Column))
	} else if e.Line > 0 {
		b.WriteString(fmt.Sprintf(" at line %d", e.Line))
	}

	if e.Field != "" {
		b.WriteString(fmt.Sprintf(": field '%s'", e.Field))
	}

	if e.Value != "" {
		b.WriteString(fmt.Sprintf(" with value '%s'", e.Value))
	}

	if e.Err != nil {
		b.WriteString(fmt.Sprintf(": %v", e.Err))
	}

	return b.String()
}

// Unwrap returns the underlying error for error chain support.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError creates a new ParseError with the given details.
func NewParseError(field, value string, err error) *ParseError {
	return &ParseError{
		Field: field,
		Value: value,
		Err:   err,
	}
}

// ValidationError represents a validation error for a specific field.
// It includes the validation rule that failed and a human-readable message.
type ValidationError struct {
	// Field is the name of the field that failed validation
	Field string

	// Value is the invalid value
	Value string

	// Rule is the validation rule that failed (e.g., "required", "min", "labelname")
	Rule string

	// Message is a human-readable description of the validation failure
	Message string
}

// Error implements the error interface for ValidationError.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s' failed validation '%s': %s (value: '%s')",
		e.Field, e.Rule, e.Message, e.Value)
}

// NewValidationError creates a new ValidationError with the given details.
func NewValidationError(field, value, rule, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// ValidationErrors represents a collection of validation errors.
// It allows accumulating multiple errors during validation.
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors.
// It formats all validation errors into a multi-line message.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("validation failed with %d error(s):\n", len(ve)))

	for i, err := range ve {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Message))
		if err.Field != "" {
			b.WriteString(fmt.Sprintf("      Field: %s\n", err.Field))
		}
		if err.Value != "" {
			b.WriteString(fmt.Sprintf("      Value: %s\n", err.Value))
		}
		if err.Rule != "" {
			b.WriteString(fmt.Sprintf("      Rule: %s\n", err.Rule))
		}
	}

	return b.String()
}

// Add appends a new validation error to the collection.
func (ve *ValidationErrors) Add(field, value, rule, message string) {
	*ve = append(*ve, NewValidationError(field, value, rule, message))
}

// AddError appends an existing ValidationError to the collection.
func (ve *ValidationErrors) AddError(err ValidationError) {
	*ve = append(*ve, err)
}

// HasErrors returns true if there are any validation errors.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Count returns the number of validation errors.
func (ve ValidationErrors) Count() int {
	return len(ve)
}

// ToError converts ValidationErrors to an error interface.
// Returns nil if there are no errors.
func (ve ValidationErrors) ToError() error {
	if !ve.HasErrors() {
		return nil
	}
	return ve
}

// ConfigError represents a general configuration error.
// Used for high-level configuration issues.
type ConfigError struct {
	// Message is the error message
	Message string

	// Source is the config file path or source identifier
	Source string

	// Err is the underlying error
	Err error
}

// Error implements the error interface for ConfigError.
func (e *ConfigError) Error() string {
	var b strings.Builder

	b.WriteString("configuration error")

	if e.Source != "" {
		b.WriteString(fmt.Sprintf(" in '%s'", e.Source))
	}

	if e.Message != "" {
		b.WriteString(fmt.Sprintf(": %s", e.Message))
	}

	if e.Err != nil {
		b.WriteString(fmt.Sprintf(": %v", e.Err))
	}

	return b.String()
}

// Unwrap returns the underlying error for error chain support.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError.
func NewConfigError(message, source string, err error) *ConfigError {
	return &ConfigError{
		Message: message,
		Source:  source,
		Err:     err,
	}
}

