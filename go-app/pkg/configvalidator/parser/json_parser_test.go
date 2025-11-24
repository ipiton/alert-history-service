package parser

import (
	"testing"

	"github.com/vitaliisemenov/alert-history/internal/alertmanager/config"
)

// ================================================================================
// JSON Parser Tests (TN-151 Phase 2C)
// ================================================================================
// Comprehensive tests for JSON configuration parsing
//
// Coverage Target: 80%+
// Quality Target: 150% (Grade A+ EXCEPTIONAL)

func TestJSONParser_Parse_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, cfg *config.AlertmanagerConfig)
	}{
		{
			name: "minimal valid config",
			input: `{
				"route": {
					"receiver": "default"
				},
				"receivers": [
					{
						"name": "default"
					}
				]
			}`,
			validate: func(t *testing.T, cfg *config.AlertmanagerConfig) {
				if cfg.Route == nil {
					t.Fatal("Route is nil")
				}
				if cfg.Route.Receiver != "default" {
					t.Errorf("Route.Receiver = %s, want default", cfg.Route.Receiver)
				}
				if len(cfg.Receivers) != 1 {
					t.Errorf("len(Receivers) = %d, want 1", len(cfg.Receivers))
				}
				if cfg.Receivers[0].Name != "default" {
					t.Errorf("Receivers[0].Name = %s, want default", cfg.Receivers[0].Name)
				}
			},
		},
		{
			name: "config with global settings",
			input: `{
				"global": {
					"resolve_timeout": "5m",
					"smtp_smarthost": "localhost:25",
					"smtp_from": "alertmanager@example.com"
				},
				"route": {
					"receiver": "default"
				},
				"receivers": [
					{
						"name": "default"
					}
				]
			}`,
			validate: func(t *testing.T, cfg *config.AlertmanagerConfig) {
				if cfg.Global == nil {
					t.Fatal("Global is nil")
				}
				if cfg.Global.ResolveTimeout.String() != "5m0s" {
					t.Errorf("Global.ResolveTimeout = %s, want 5m0s", cfg.Global.ResolveTimeout)
				}
				if cfg.Global.SMTPSmartHost != "localhost:25" {
					t.Errorf("Global.SMTPSmartHost = %s, want localhost:25", cfg.Global.SMTPSmartHost)
				}
			},
		},
		{
			name: "config with route matchers",
			input: `{
				"route": {
					"receiver": "default",
					"matchers": ["severity=critical", "team=backend"]
				},
				"receivers": [
					{
						"name": "default"
					}
				]
			}`,
			validate: func(t *testing.T, cfg *config.AlertmanagerConfig) {
				if len(cfg.Route.Matchers) != 2 {
					t.Errorf("len(Route.Matchers) = %d, want 2", len(cfg.Route.Matchers))
				}
			},
		},
		{
			name: "config with nested routes",
			input: `{
				"route": {
					"receiver": "default",
					"routes": [
						{
							"receiver": "team-a",
							"matchers": ["team=a"]
						},
						{
							"receiver": "team-b",
							"matchers": ["team=b"]
						}
					]
				},
				"receivers": [
					{"name": "default"},
					{"name": "team-a"},
					{"name": "team-b"}
				]
			}`,
			validate: func(t *testing.T, cfg *config.AlertmanagerConfig) {
				if len(cfg.Route.Routes) != 2 {
					t.Errorf("len(Route.Routes) = %d, want 2", len(cfg.Route.Routes))
				}
				if len(cfg.Receivers) != 3 {
					t.Errorf("len(Receivers) = %d, want 3", len(cfg.Receivers))
				}
			},
		},
		{
			name: "config with inhibit rules",
			input: `{
				"route": {
					"receiver": "default"
				},
				"receivers": [
					{"name": "default"}
				],
				"inhibit_rules": [
					{
						"source_matchers": ["severity=critical"],
						"target_matchers": ["severity=warning"],
						"equal": ["alertname", "instance"]
					}
				]
			}`,
			validate: func(t *testing.T, cfg *config.AlertmanagerConfig) {
				if len(cfg.InhibitRules) != 1 {
					t.Errorf("len(InhibitRules) = %d, want 1", len(cfg.InhibitRules))
				}
				if len(cfg.InhibitRules[0].Equal) != 2 {
					t.Errorf("len(InhibitRules[0].Equal) = %d, want 2", len(cfg.InhibitRules[0].Equal))
				}
			},
		},
	}

	parser := NewJSONParser(true) // strict mode

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errors := parser.Parse([]byte(tt.input))

			if len(errors) > 0 {
				t.Fatalf("Parse() returned errors: %v", errors)
			}

			if cfg == nil {
				t.Fatal("Parse() returned nil config")
			}

			tt.validate(t, cfg)
		})
	}
}

func TestJSONParser_Parse_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrCode string
		errContains string
	}{
		{
			name:        "empty input",
			input:       "",
			wantErrCode: "E002", // JSON parser uses E002 for syntax errors
			errContains: "EOF",
		},
		{
			name:        "invalid JSON syntax",
			input:       `{"route": invalid}`,
			wantErrCode: "E002", // JSON parser uses E002 for syntax errors
			errContains: "invalid character",
		},
		{
			name:        "unclosed bracket",
			input:       `{"route": {"receiver": "default"`,
			wantErrCode: "E002", // JSON parser uses E002 for syntax errors
			errContains: "unexpected",
		},
	}

	parser := NewJSONParser(true) // strict mode

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errors := parser.Parse([]byte(tt.input))

			if len(errors) == 0 {
				t.Fatal("Parse() expected errors, got none")
			}

			// Check error code
			if errors[0].Code != tt.wantErrCode {
				t.Errorf("Error code = %s, want %s", errors[0].Code, tt.wantErrCode)
			}

			// Check error message contains expected substring
			if tt.errContains != "" {
				found := false
				for _, err := range errors {
					if contains(err.Message, tt.errContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Error message does not contain %q. Errors: %v", tt.errContains, errors)
				}
			}

			// Config may or may not be nil depending on error type
			_ = cfg
		})
	}
}

func TestJSONParser_Parse_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "empty JSON object",
			input:     `{}`,
			expectErr: false, // Parser doesn't validate, just parses
		},
		{
			name:      "whitespace only",
			input:     `   `,
			expectErr: true,
		},
		{
			name: "null values",
			input: `{
				"route": {
					"receiver": "default",
					"group_by": null
				},
				"receivers": [
					{"name": "default"}
				]
			}`,
			expectErr: false,
		},
	}

	parser := NewJSONParser(true) // strict mode

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errors := parser.Parse([]byte(tt.input))

			if tt.expectErr {
				if len(errors) == 0 {
					t.Error("Parse() expected errors, got none")
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("Parse() unexpected errors: %v", errors)
				}
				if cfg == nil {
					t.Error("Parse() returned nil config")
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
