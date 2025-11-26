package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ================================================================================
// TN-156: Template Validator - Root Command
// ================================================================================
// Root command setup for template-validator CLI.
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-25

var (
	version   string
	buildTime string
	gitCommit string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "template-validator",
	Short: "Validate Alertmanager++ notification templates",
	Long: `Template Validator validates notification templates for Alertmanager++.

Validation includes:
  - Syntax validation (Go text/template)
  - Semantic validation (Alertmanager data model)
  - Security validation (XSS, secrets, injection)
  - Best practices validation (performance, readability)

Examples:
  # Validate single template
  template-validator validate slack.tmpl

  # Validate all templates in directory
  template-validator validate templates/ --recursive

  # Strict validation (fail on warnings)
  template-validator validate templates/ --mode=strict

  # JSON output for CI/CD
  template-validator validate templates/ --output=json

  # SARIF output for GitHub/GitLab
  template-validator validate templates/ --output=sarif > results.sarif

Exit Codes:
  0: Success (no errors)
  1: Validation failed (errors found)
  2: Warnings only (validation passed with warnings)
`,
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

// init initializes commands
func init() {
	// Add commands
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
}

// SetVersion sets version information
func SetVersion(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
}

// ================================================================================

// versionCmd displays version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("template-validator version %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		fmt.Printf("Git commit: %s\n", gitCommit)
	},
}

// ================================================================================

