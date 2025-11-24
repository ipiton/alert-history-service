package configvalidator

import (
	"encoding/json"
	"fmt"
	"time"
)

// ================================================================================
// Common Types for Config Validator
// ================================================================================
// Shared types used across validator, parsers, and formatters (TN-151).
//
// This file contains types that need to be shared between subpackages
// to avoid import cycles.
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-24

// Error represents a validation error.
type Error struct {
	// Type is error type (e.g., "syntax", "reference", "type", "security")
	Type string `json:"type"`

	// Code is error code (e.g., "E001", "E002", "E100")
	Code string `json:"code"`

	// Message is human-readable error message
	Message string `json:"message"`

	// Location is error location in file
	Location Location `json:"location"`

	// Context is surrounding code context (lines before/after)
	Context string `json:"context,omitempty"`

	// Suggestion is how to fix the error
	Suggestion string `json:"suggestion,omitempty"`

	// DocsURL is link to relevant documentation
	DocsURL string `json:"docs_url,omitempty"`
}

// Warning represents a validation warning.
type Warning struct {
	// Type is warning type
	Type string `json:"type"`

	// Code is warning code (e.g., "W100", "W200")
	Code string `json:"code"`

	// Message is human-readable warning message
	Message string `json:"message"`

	// Location is warning location
	Location Location `json:"location"`

	// Suggestion is recommended action
	Suggestion string `json:"suggestion,omitempty"`

	// DocsURL is link to documentation
	DocsURL string `json:"docs_url,omitempty"`
}

// Info represents informational message.
type Info struct {
	// Type is info type
	Type string `json:"type"`

	// Message is informational message
	Message string `json:"message"`

	// Location is optional location
	Location Location `json:"location,omitempty"`
}

// Suggestion represents improvement suggestion.
type Suggestion struct {
	// Type is suggestion type
	Type string `json:"type"`

	// Message is suggestion message
	Message string `json:"message"`

	// Before is current state (optional)
	Before string `json:"before,omitempty"`

	// After is suggested state (optional)
	After string `json:"after,omitempty"`
}

// Location represents location in configuration file.
type Location struct {
	// File is file path (optional)
	File string `json:"file,omitempty"`

	// Line is line number (1-based)
	Line int `json:"line"`

	// Column is column number (1-based, optional)
	Column int `json:"column,omitempty"`

	// Field is field path (e.g., "route.receiver")
	Field string `json:"field,omitempty"`

	// Section is config section (e.g., "route", "receivers", "inhibit_rules")
	Section string `json:"section,omitempty"`
}

// String returns human-readable location string.
func (l Location) String() string {
	if l.File != "" {
		if l.Column > 0 {
			return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
		}
		return fmt.Sprintf("%s:%d", l.File, l.Line)
	}
	if l.Field != "" {
		return fmt.Sprintf("%s (line %d)", l.Field, l.Line)
	}
	if l.Line > 0 {
		return fmt.Sprintf("line %d", l.Line)
	}
	return "unknown location"
}

// Issue is a generic validation issue (for CLI output).
//
// This is a union type that can represent Error, Warning, Info, or Suggestion.
// Used by CLI formatters to have a unified interface.
type Issue struct {
	// Level is issue severity: "error", "warning", "info", "suggestion"
	Level string `json:"level"`

	// Code is issue code
	Code string `json:"code"`

	// Message is issue message
	Message string `json:"message"`

	// Location is issue location
	Location *Location `json:"location,omitempty"`

	// Context is code context
	Context string `json:"context,omitempty"`

	// Suggestion is recommended fix
	Suggestion string `json:"suggestion,omitempty"`

	// DocsURL is documentation link
	DocsURL string `json:"docs_url,omitempty"`

	// FieldPath is alternative to Location (for backward compat)
	FieldPath string `json:"field_path,omitempty"`
}

// Result represents validation result.
type Result struct {
	// Valid indicates if configuration is valid
	Valid bool `json:"valid"`

	// Errors are critical validation errors (block deployment)
	Errors []Error `json:"errors,omitempty"`

	// Warnings are potential problems (don't block, but should be fixed)
	Warnings []Warning `json:"warnings,omitempty"`

	// Info are recommendations and best practices
	Info []Info `json:"info,omitempty"`

	// Suggestions are actionable improvements
	Suggestions []Suggestion `json:"suggestions,omitempty"`

	// FilePath is the validated file path (optional)
	FilePath string `json:"file_path,omitempty"`

	// Duration is validation duration
	Duration time.Duration `json:"-"`

	// DurationMs is validation duration in milliseconds (for JSON)
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// NewResult creates a new empty validation result.
func NewResult() *Result {
	return &Result{
		Valid:       true,
		Errors:      make([]Error, 0),
		Warnings:    make([]Warning, 0),
		Info:        make([]Info, 0),
		Suggestions: make([]Suggestion, 0),
	}
}

// AddError adds an error to the result.
//
// This is a convenience method used by validators.
// Parameters match the Error struct fields for easy inline creation.
func (r *Result) AddError(code, message string, location *Location, field, section, context, suggestion, docsURL string) {
	if location == nil {
		location = &Location{}
	}
	if field != "" {
		location.Field = field
	}
	if section != "" {
		location.Section = section
	}

	r.Errors = append(r.Errors, Error{
		Type:       "validation",
		Code:       code,
		Message:    message,
		Location:   *location,
		Context:    context,
		Suggestion: suggestion,
		DocsURL:    docsURL,
	})
	r.Valid = false
}

// AddWarning adds a warning to the result.
func (r *Result) AddWarning(code, message string, location *Location, field, section, context, suggestion, docsURL string) {
	if location == nil {
		location = &Location{}
	}
	if field != "" {
		location.Field = field
	}
	if section != "" {
		location.Section = section
	}

	r.Warnings = append(r.Warnings, Warning{
		Type:       "validation",
		Code:       code,
		Message:    message,
		Location:   *location,
		Suggestion: suggestion,
		DocsURL:    docsURL,
	})
}

// Merge merges another result into this result.
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}

	r.Errors = append(r.Errors, other.Errors...)
	r.Warnings = append(r.Warnings, other.Warnings...)
	r.Info = append(r.Info, other.Info...)
	r.Suggestions = append(r.Suggestions, other.Suggestions...)

	if len(other.Errors) > 0 {
		r.Valid = false
	}
}

// Summary returns a summary string.
func (r *Result) Summary() string {
	if r.Valid {
		return fmt.Sprintf("✓ Configuration is valid (validated in %dms)", r.DurationMs)
	}

	return fmt.Sprintf("✗ Configuration is invalid: %d errors, %d warnings",
		len(r.Errors), len(r.Warnings))
}

// ExitCode returns appropriate exit code for CLI based on validation mode.
//
// Exit codes:
//   - 0: Success (no errors, or warnings ignored)
//   - 1: Errors found
//   - 2: Warnings found (strict mode only)
func (r *Result) ExitCode(mode ValidationMode) int {
	if len(r.Errors) > 0 {
		return 1 // Errors always fail
	}

	if mode == StrictMode && len(r.Warnings) > 0 {
		return 2 // Warnings fail in strict mode
	}

	return 0 // Success
}

// MarshalJSON implements json.Marshaler to add DurationMs field.
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		*Alias
		DurationMs int64 `json:"duration_ms"`
	}{
		Alias:      (*Alias)(r),
		DurationMs: r.Duration.Milliseconds(),
	})
}
