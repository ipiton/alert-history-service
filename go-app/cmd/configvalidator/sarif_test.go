package main

import (
	"testing"

	"github.com/vitaliisemenov/alert-history/pkg/configvalidator"
	"github.com/vitaliisemenov/alert-history/pkg/configvalidator/types"
)

// TestConvertToSARIFResults_IncludesSuggestions verifies that suggestions
// are included in SARIF output (bugfix verification for TN-151)
func TestConvertToSARIFResults_IncludesSuggestions(t *testing.T) {
	// Create a result with all types of issues
	result := types.NewResult()

	// Add one of each type
	result.AddError(
		"E001",
		"Test error",
		&types.Location{File: "config.yml", Line: 1, Column: 1},
		"field1", "section1", "", "", "",
	)

	result.AddWarning(
		"W001",
		"Test warning",
		&types.Location{File: "config.yml", Line: 2, Column: 1},
		"field2", "section2", "", "", "",
	)

	result.AddInfo(
		"I001",
		"Test info",
		&types.Location{File: "config.yml", Line: 3, Column: 1},
		"field3",
	)

	result.AddSuggestion(
		"S001",
		"Test suggestion",
		&types.Location{File: "config.yml", Line: 4, Column: 1},
		"field4",
	)

	// Convert to configvalidator.Result (wrapper)
	wrapperResult := &configvalidator.Result{
		Errors:      result.Errors,
		Warnings:    result.Warnings,
		Info:        result.Info,
		Suggestions: result.Suggestions,
	}

	// Convert to SARIF
	sarifResults := convertToSARIFResults(wrapperResult)

	// Verify counts
	expectedCount := 4 // 1 error + 1 warning + 1 info + 1 suggestion
	if len(sarifResults) != expectedCount {
		t.Errorf("convertToSARIFResults() returned %d results, want %d (1 error + 1 warning + 1 info + 1 suggestion)",
			len(sarifResults), expectedCount)
		t.Logf("Results breakdown:")
		for i, r := range sarifResults {
			t.Logf("  [%d] ruleId=%v, level=%v", i, r["ruleId"], r["level"])
		}
	}

	// Verify that suggestion is present
	foundSuggestion := false
	for _, result := range sarifResults {
		if ruleId, ok := result["ruleId"].(string); ok && ruleId == "S001" {
			foundSuggestion = true
			// Verify level is "note" for suggestions
			if level, ok := result["level"].(string); !ok || level != "note" {
				t.Errorf("Suggestion has level=%v, want 'note'", level)
			}
			break
		}
	}

	if !foundSuggestion {
		t.Error("Suggestion (S001) not found in SARIF results - suggestions are missing from SARIF output!")
	}
}

// TestConvertToSARIFResults_EmptyResult verifies handling of empty results
func TestConvertToSARIFResults_EmptyResult(t *testing.T) {
	result := &configvalidator.Result{}
	sarifResults := convertToSARIFResults(result)

	if len(sarifResults) != 0 {
		t.Errorf("convertToSARIFResults() for empty result returned %d results, want 0", len(sarifResults))
	}
}

// TestConvertToSARIFResults_OnlySuggestions verifies that suggestions-only results work
func TestConvertToSARIFResults_OnlySuggestions(t *testing.T) {
	result := types.NewResult()

	// Add multiple suggestions
	result.AddSuggestion("S001", "Suggestion 1", &types.Location{File: "config.yml", Line: 1}, "field1")
	result.AddSuggestion("S002", "Suggestion 2", &types.Location{File: "config.yml", Line: 2}, "field2")
	result.AddSuggestion("S003", "Suggestion 3", &types.Location{File: "config.yml", Line: 3}, "field3")

	wrapperResult := &configvalidator.Result{
		Suggestions: result.Suggestions,
	}

	sarifResults := convertToSARIFResults(wrapperResult)

	expectedCount := 3
	if len(sarifResults) != expectedCount {
		t.Errorf("convertToSARIFResults() returned %d results, want %d suggestions", len(sarifResults), expectedCount)
	}

	// Verify all are suggestions with correct level
	for i, r := range sarifResults {
		if level, ok := r["level"].(string); !ok || level != "note" {
			t.Errorf("Result[%d] has level=%v, want 'note'", i, level)
		}
	}
}
