package main

import (
	"fmt"
	"os"

	"github.com/vitaliisemenov/alert-history/cmd/template-validator/cmd"
)

// ================================================================================
// TN-156: Template Validator - CLI Entry Point
// ================================================================================
// Command-line interface for template validation.
//
// Features:
// - validate command (single file, batch, recursive)
// - version command
// - Multiple output formats (human, JSON, SARIF)
// - CI/CD integration (exit codes, machine-readable output)
//
// Usage:
//   template-validator validate <file>
//   template-validator validate <directory> --recursive
//   template-validator validate <file> --output=json
//   template-validator validate <directory> --output=sarif
//
// Exit Codes:
//   0: Success (no errors)
//   1: Validation failed (errors found)
//   2: Warnings only (validation passed with warnings)
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-25

// Version information (set by build)
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Set version info
	cmd.SetVersion(Version, BuildTime, GitCommit)

	// Execute root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ================================================================================

