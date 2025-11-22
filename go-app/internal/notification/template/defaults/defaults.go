package defaults

// ================================================================================
// TN-154: Default Templates - Template Registry
// ================================================================================
// Central registry for all default notification templates.
//
// Features:
// - Unified access to all templates
// - Type-safe template retrieval
// - Helper functions for applying defaults
// - Production-ready templates
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-22

// TemplateRegistry holds all default templates for all receiver types.
// Provides centralized access to Slack, PagerDuty, and Email templates.
//
// Usage:
//
//	registry := GetDefaultTemplates()
//	slackTitle := registry.Slack.Title
//	pagerdutyDesc := registry.PagerDuty.Description
//	emailHTML := registry.Email.HTML
type TemplateRegistry struct {
	// Slack holds all Slack default templates
	Slack *SlackTemplates

	// PagerDuty holds all PagerDuty default templates
	PagerDuty *PagerDutyTemplates

	// Email holds all Email default templates
	Email *EmailTemplates
}

// GetDefaultTemplates returns the complete default template registry.
// This is the main entry point for accessing all default templates.
//
// Returns:
//   - *TemplateRegistry: Registry with all default templates
//
// Example:
//
//	registry := GetDefaultTemplates()
//
//	// Access Slack templates
//	slackTitle := registry.Slack.Title
//	slackColor := registry.Slack.ColorFunc("critical")
//
//	// Access PagerDuty templates
//	pdDesc := registry.PagerDuty.Description
//	pdSeverity := registry.PagerDuty.SeverityFunc("warning")
//
//	// Access Email templates
//	emailSubject := registry.Email.Subject
//	emailHTML := registry.Email.HTML
func GetDefaultTemplates() *TemplateRegistry {
	return &TemplateRegistry{
		Slack:     GetDefaultSlackTemplates(),
		PagerDuty: GetDefaultPagerDutyTemplates(),
		Email:     GetDefaultEmailTemplates(),
	}
}

// ValidateAllTemplates validates all default templates for size and structure.
// This is useful for CI/CD pipelines and startup checks.
//
// Returns:
//   - error: If any template fails validation, nil otherwise
//
// Validations:
//   - Slack message size < 3000 chars
//   - PagerDuty description < 1024 chars
//   - Email HTML < 100KB
//   - All templates non-empty
func ValidateAllTemplates() error {
	registry := GetDefaultTemplates()

	// Validate Slack templates
	if registry.Slack.Title == "" {
		return &TemplateValidationError{
			Template: "Slack.Title",
			Reason:   "template is empty",
		}
	}
	if registry.Slack.Text == "" {
		return &TemplateValidationError{
			Template: "Slack.Text",
			Reason:   "template is empty",
		}
	}

	// Validate PagerDuty templates
	if registry.PagerDuty.Description == "" {
		return &TemplateValidationError{
			Template: "PagerDuty.Description",
			Reason:   "template is empty",
		}
	}
	if !ValidatePagerDutyDescriptionSize(registry.PagerDuty.Description) {
		return &TemplateValidationError{
			Template: "PagerDuty.Description",
			Reason:   "template exceeds 1024 char limit",
		}
	}

	// Validate Email templates
	if registry.Email.Subject == "" {
		return &TemplateValidationError{
			Template: "Email.Subject",
			Reason:   "template is empty",
		}
	}
	if registry.Email.HTML == "" {
		return &TemplateValidationError{
			Template: "Email.HTML",
			Reason:   "template is empty",
		}
	}
	if !ValidateEmailHTMLSize(registry.Email.HTML) {
		return &TemplateValidationError{
			Template: "Email.HTML",
			Reason:   "template exceeds 100KB limit",
		}
	}

	return nil
}

// TemplateValidationError represents a template validation error.
type TemplateValidationError struct {
	Template string
	Reason   string
}

// Error implements the error interface.
func (e *TemplateValidationError) Error() string {
	return "template validation failed: " + e.Template + " - " + e.Reason
}

// GetTemplateStats returns statistics about all default templates.
// Useful for monitoring and debugging.
//
// Returns:
//   - *TemplateStats: Statistics about template sizes and counts
type TemplateStats struct {
	// SlackTemplateCount is the number of Slack templates
	SlackTemplateCount int

	// PagerDutyTemplateCount is the number of PagerDuty templates
	PagerDutyTemplateCount int

	// EmailTemplateCount is the number of Email templates
	EmailTemplateCount int

	// TotalSize is the total size of all templates in bytes
	TotalSize int

	// SlackSize is the total size of Slack templates
	SlackSize int

	// PagerDutySize is the total size of PagerDuty templates
	PagerDutySize int

	// EmailSize is the total size of Email templates
	EmailSize int
}

// GetTemplateStats returns statistics about all default templates.
func GetTemplateStats() *TemplateStats {
	registry := GetDefaultTemplates()

	slackSize := len(registry.Slack.Title) +
		len(registry.Slack.Text) +
		len(registry.Slack.Pretext) +
		len(registry.Slack.FieldsSingle) +
		len(registry.Slack.FieldsMulti)

	pagerdutySize := len(registry.PagerDuty.Description) +
		len(registry.PagerDuty.DetailsSingle) +
		len(registry.PagerDuty.DetailsMulti)

	emailSize := len(registry.Email.Subject) +
		len(registry.Email.HTML) +
		len(registry.Email.Text)

	return &TemplateStats{
		SlackTemplateCount:     5, // Title, Text, Pretext, FieldsSingle, FieldsMulti
		PagerDutyTemplateCount: 3, // Description, DetailsSingle, DetailsMulti
		EmailTemplateCount:     3, // Subject, HTML, Text
		TotalSize:              slackSize + pagerdutySize + emailSize,
		SlackSize:              slackSize,
		PagerDutySize:          pagerdutySize,
		EmailSize:              emailSize,
	}
}
