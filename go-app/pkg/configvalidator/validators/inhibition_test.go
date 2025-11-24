package validators

import (
	"context"
	"log/slog"
	"testing"

	"github.com/vitaliisemenov/alert-history/internal/alertmanager/config"
	"github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

// ================================================================================
// Inhibition Validator Tests (TN-151 Phase 2C)
// ================================================================================
// Comprehensive tests for inhibition rule validation
//
// Coverage Target: 80%+
// Quality Target: 150% (Grade A+ EXCEPTIONAL)

func TestInhibitionValidator_BasicValidation(t *testing.T) {
	tests := []struct {
		name           string
		inhibitRules   []config.InhibitRule
		expectError    bool
		expectWarning  bool
		errorCode      string
		warningCode    string
	}{
		{
			name: "valid inhibition rule with source and target matchers",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname", "instance"},
				},
			},
			expectError:   false,
			expectWarning: false,
		},
		{
			name: "inhibition rule without source matchers",
			inhibitRules: []config.InhibitRule{
				{
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E150",
		},
		{
			name: "inhibition rule without target matchers",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E151",
		},
		{
			name: "inhibition rule without equal labels",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
				},
			},
			expectWarning: true,
			warningCode:   "W154", // Actual warning code from inhibition.go
		},
		{
			name: "inhibition rule with empty equal labels list",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{},
				},
			},
			expectWarning: true,
			warningCode:   "W154", // Actual warning code from inhibition.go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewInhibitionValidator(types.DefaultOptions(), slog.Default())

			cfg := &config.AlertmanagerConfig{
				InhibitRules: tt.inhibitRules,
			}

			result := types.NewResult()
			validator.Validate(context.Background(), cfg, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error, got none")
				} else if tt.errorCode != "" {
					found := false
					for _, err := range result.Errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error code %s, got: %v", tt.errorCode, result.Errors)
					}
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}

			if tt.expectWarning {
				if len(result.Warnings) == 0 {
					t.Error("Expected warning, got none")
				} else if tt.warningCode != "" {
					found := false
					for _, warn := range result.Warnings {
						if warn.Code == tt.warningCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected warning code %s, got: %v", tt.warningCode, result.Warnings)
					}
				}
			}
		})
	}
}

func TestInhibitionValidator_MatcherSyntax(t *testing.T) {
	tests := []struct {
		name         string
		inhibitRules []config.InhibitRule
		expectError  bool
		errorCode    string
	}{
		{
			name: "valid matcher syntax",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical", "team=frontend"},
					TargetMatchers: []string{"severity=~warning|info"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid source matcher syntax",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"invalid-syntax"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E153", // Actual error code for matchers
		},
		{
			name: "invalid target matcher syntax",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"invalid-matcher-syntax"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E153", // Actual error code for matchers
		},
		{
			name: "invalid regex in source matcher",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=~[unclosed"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E153", // Actual error code for matchers
		},
		{
			name: "invalid regex in target matcher",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=~(unclosed"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: true,
			errorCode:   "E153", // Actual error code for matchers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewInhibitionValidator(types.DefaultOptions(), slog.Default())

			cfg := &config.AlertmanagerConfig{
				InhibitRules: tt.inhibitRules,
			}

			result := types.NewResult()
			validator.Validate(context.Background(), cfg, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error, got none")
				} else if tt.errorCode != "" {
					found := false
					for _, err := range result.Errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error code %s, got: %v", tt.errorCode, result.Errors)
					}
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestInhibitionValidator_EqualLabels(t *testing.T) {
	tests := []struct {
		name         string
		inhibitRules []config.InhibitRule
		expectError  bool
		errorCode    string
	}{
		{
			name: "valid equal labels",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname", "instance", "job"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid label name in equal",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"valid_label", "invalid-label!"},
				},
			},
			expectError: true,
			errorCode:   "E152", // Actual error code for invalid label in equal
		},
		{
			name: "empty label name in equal",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname", "", "instance"},
				},
			},
			expectError: true,
			errorCode:   "E152", // Actual error code for invalid label in equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewInhibitionValidator(types.DefaultOptions(), slog.Default())

			cfg := &config.AlertmanagerConfig{
				InhibitRules: tt.inhibitRules,
			}

			result := types.NewResult()
			validator.Validate(context.Background(), cfg, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error, got none")
				} else if tt.errorCode != "" {
					found := false
					for _, err := range result.Errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error code %s, got: %v", tt.errorCode, result.Errors)
					}
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestInhibitionValidator_MultipleRules(t *testing.T) {
	tests := []struct {
		name          string
		inhibitRules  []config.InhibitRule
		expectError   bool
		expectWarning bool
	}{
		{
			name: "multiple valid rules",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
				{
					SourceMatchers: []string{"alertname=HostDown"},
					TargetMatchers: []string{"alertname=HostNetworkDown"},
					Equal:          []string{"instance"},
				},
			},
			expectError:   false,
			expectWarning: false,
		},
		{
			name: "mixed valid and invalid rules",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
				{
					// Missing target matchers
					SourceMatchers: []string{"severity=critical"},
					Equal:          []string{"alertname"},
				},
			},
			expectError:   true,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewInhibitionValidator(types.DefaultOptions(), slog.Default())

			cfg := &config.AlertmanagerConfig{
				InhibitRules: tt.inhibitRules,
			}

			result := types.NewResult()
			validator.Validate(context.Background(), cfg, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error, got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestInhibitionValidator_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		inhibitRules []config.InhibitRule
		expectError  bool
	}{
		{
			name:         "no inhibition rules",
			inhibitRules: []config.InhibitRule{},
			expectError:  false,
		},
		{
			name:         "nil inhibition rules",
			inhibitRules: nil,
			expectError:  false,
		},
		{
			name: "inhibition rule with many source matchers",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{
						"severity=critical",
						"team=backend",
						"env=production",
						"region=us-east",
						"cluster=main",
					},
					TargetMatchers: []string{"severity=warning"},
					Equal:          []string{"alertname"},
				},
			},
			expectError: false,
		},
		{
			name: "inhibition rule with many equal labels",
			inhibitRules: []config.InhibitRule{
				{
					SourceMatchers: []string{"severity=critical"},
					TargetMatchers: []string{"severity=warning"},
					Equal: []string{
						"alertname",
						"instance",
						"job",
						"cluster",
						"region",
						"env",
						"team",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewInhibitionValidator(types.DefaultOptions(), slog.Default())

			cfg := &config.AlertmanagerConfig{
				InhibitRules: tt.inhibitRules,
			}

			result := types.NewResult()
			validator.Validate(context.Background(), cfg, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error, got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}
		})
	}
}

