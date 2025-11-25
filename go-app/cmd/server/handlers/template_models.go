package handlers

import (
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core/domain"
)

// ================================================================================
// TN-155: Template API (CRUD) - HTTP Models
// ================================================================================
// Request and response DTOs for template API endpoints.
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-25

// ================================================================================
// Request Models
// ================================================================================

// CreateTemplateRequest represents POST /api/v2/templates request
type CreateTemplateRequest struct {
	Name        string                 `json:"name" validate:"required,min=3,max=64"`
	Type        string                 `json:"type" validate:"required,oneof=slack pagerduty email webhook generic"`
	Content     string                 `json:"content" validate:"required,max=65536"`
	Description string                 `json:"description" validate:"max=500"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// UpdateTemplateRequest represents PUT /api/v2/templates/{name} request
type UpdateTemplateRequest struct {
	Content     *string                `json:"content,omitempty" validate:"omitempty,max=65536"`
	Description *string                `json:"description,omitempty" validate:"omitempty,max=500"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ValidateTemplateRequest represents POST /api/v2/templates/validate request
type ValidateTemplateRequest struct {
	Content  string                 `json:"content" validate:"required"`
	Type     string                 `json:"type" validate:"required,oneof=slack pagerduty email webhook generic"`
	TestData map[string]interface{} `json:"test_data,omitempty"`
}

// RollbackTemplateRequest represents POST /api/v2/templates/{name}/rollback request
type RollbackTemplateRequest struct {
	Version int    `json:"version" validate:"required,min=1"`
	Reason  string `json:"reason" validate:"required,max=500"`
}

// BatchCreateTemplatesRequest represents POST /api/v2/templates/batch request
type BatchCreateTemplatesRequest struct {
	Operation string                   `json:"operation" validate:"required,oneof=create update delete"`
	Templates []CreateTemplateRequest  `json:"templates" validate:"required,dive"`
}

// TestTemplateRequest represents POST /api/v2/templates/{name}/test request
type TestTemplateRequest struct {
	Alert map[string]interface{} `json:"alert" validate:"required"`
}

// ================================================================================
// Response Models
// ================================================================================

// TemplateResponse represents a full template with content
type TemplateResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	UpdatedBy   string                 `json:"updated_by,omitempty"`
}

// TemplateSummaryResponse represents a template without content (for lists)
type TemplateSummaryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by,omitempty"`
}

// ListTemplatesResponse represents GET /api/v2/templates response
type ListTemplatesResponse struct {
	Templates  []TemplateSummaryResponse `json:"templates"`
	Pagination PaginationInfo            `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// TemplateVersionResponse represents a template version
type TemplateVersionResponse struct {
	ID            string                 `json:"id"`
	TemplateID    string                 `json:"template_id"`
	Version       int                    `json:"version"`
	Content       string                 `json:"content"`
	Description   string                 `json:"description"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by,omitempty"`
	ChangeSummary string                 `json:"change_summary,omitempty"`
}

// ListVersionsResponse represents GET /api/v2/templates/{name}/versions response
type ListVersionsResponse struct {
	Versions []TemplateVersionResponse `json:"versions"`
	Total    int                       `json:"total"`
	Limit    int                       `json:"limit"`
	Offset   int                       `json:"offset"`
}

// ValidationResultResponse represents template validation result
type ValidationResultResponse struct {
	Valid          bool                       `json:"valid"`
	SyntaxErrors   []TemplateValidationError  `json:"syntax_errors"`
	Warnings       []string                   `json:"warnings"`
	FunctionsUsed  []string                   `json:"functions_used"`
	VariablesUsed  []string                   `json:"variables_used"`
	RenderedOutput string                     `json:"rendered_output,omitempty"`
}

// TemplateValidationError represents a template validation error
type TemplateValidationError struct {
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// TemplateDiffResponse represents diff between versions
type TemplateDiffResponse struct {
	TemplateName string    `json:"template_name"`
	FromVersion  int       `json:"from_version"`
	ToVersion    int       `json:"to_version"`
	Diff         string    `json:"diff"`
	ChangedAt    time.Time `json:"changed_at"`
	ChangedBy    string    `json:"changed_by,omitempty"`
}

// TemplateStatsResponse represents template statistics
type TemplateStatsResponse struct {
	TotalTemplates      int                   `json:"total_templates"`
	ByType              map[string]int        `json:"by_type"`
	MostUsed            []TemplateUsageStat   `json:"most_used,omitempty"`
	ValidationErrorRate float64               `json:"validation_error_rate"`
	AverageContentSize  int                   `json:"average_content_size"`
}

// TemplateUsageStat represents usage statistics for a template
type TemplateUsageStat struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	UsageCount int    `json:"usage_count"`
}

// TestTemplateResponse represents template test result
type TestTemplateResponse struct {
	RenderedOutput string `json:"rendered_output"`
	ExecutionTime  int64  `json:"execution_time_ms"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

// BatchCreateResponse represents batch operation result
type BatchCreateResponse struct {
	Created []TemplateResponse `json:"created"`
	Errors  []BatchError       `json:"errors,omitempty"`
}

// BatchError represents an error in batch operation
type BatchError struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// TemplateErrorResponse represents a template API error response
type TemplateErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ================================================================================
// Conversion functions
// ================================================================================

// ToTemplateResponse converts domain.Template to TemplateResponse
func ToTemplateResponse(t *domain.Template) TemplateResponse {
	return TemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Type:        string(t.Type),
		Content:     t.Content,
		Description: t.Description,
		Metadata:    t.Metadata,
		Version:     t.Version,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CreatedBy:   t.CreatedBy,
		UpdatedBy:   t.UpdatedBy,
	}
}

// ToTemplateSummaryResponse converts domain.TemplateSummary to TemplateSummaryResponse
func ToTemplateSummaryResponse(s domain.TemplateSummary) TemplateSummaryResponse {
	return TemplateSummaryResponse{
		ID:          s.ID,
		Name:        s.Name,
		Type:        string(s.Type),
		Description: s.Description,
		Version:     s.Version,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		CreatedBy:   s.CreatedBy,
	}
}

// ToVersionResponse converts domain.TemplateVersion to TemplateVersionResponse
func ToVersionResponse(v *domain.TemplateVersion) TemplateVersionResponse {
	return TemplateVersionResponse{
		ID:            v.ID,
		TemplateID:    v.TemplateID,
		Version:       v.Version,
		Content:       v.Content,
		Description:   v.Description,
		Metadata:      v.Metadata,
		CreatedAt:     v.CreatedAt,
		CreatedBy:     v.CreatedBy,
		ChangeSummary: v.ChangeSummary,
	}
}

// ToValidationResultResponse converts domain.ValidationResult to ValidationResultResponse
func ToValidationResultResponse(v *domain.ValidationResult) ValidationResultResponse {
	errors := make([]TemplateValidationError, len(v.SyntaxErrors))
	for i, e := range v.SyntaxErrors {
		errors[i] = TemplateValidationError{
			Line:       e.Line,
			Column:     e.Column,
			Message:    e.Message,
			Suggestion: e.Suggestion,
		}
	}

	return ValidationResultResponse{
		Valid:          v.Valid,
		SyntaxErrors:   errors,
		Warnings:       v.Warnings,
		FunctionsUsed:  v.FunctionsUsed,
		VariablesUsed:  v.VariablesUsed,
		RenderedOutput: v.RenderedOutput,
	}
}

// ================================================================================
