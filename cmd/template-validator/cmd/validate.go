package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vitaliisemenov/alert-history/pkg/templatevalidator"
	"github.com/vitaliisemenov/alert-history/pkg/templatevalidator/formatters"
)

// ================================================================================
// TN-156: Template Validator - Validate Command
// ================================================================================
// Validate command implementation with batch processing and multiple output formats.
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-25

// Validation flags
var (
	// mode controls validation strictness
	mode string

	// templateType for type-specific validation
	templateType string

	// failOnWarning treats warnings as errors
	failOnWarning bool

	// maxErrors limits error collection
	maxErrors int

	// phases controls which validators run
	phases []string

	// output format (human, json, sarif)
	output string

	// quiet suppresses non-error output
	quiet bool

	// recursive enables recursive directory traversal
	recursive bool

	// pattern is file glob pattern
	pattern string

	// parallel is number of parallel workers
	parallel int
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate <file|directory>",
	Short: "Validate template files",
	Long: `Validate notification template files for syntax, semantics, security, and best practices.

Validation Phases:
  - syntax: Go text/template syntax
  - semantic: Alertmanager data model compatibility
  - security: XSS, hardcoded secrets, template injection
  - best_practices: Performance, readability, maintainability

Validation Modes:
  - strict: Fail on warnings (no warnings allowed)
  - lenient: Allow warnings (default)
  - permissive: Allow warnings and some errors

Output Formats:
  - human: Human-readable with colors (default)
  - json: Machine-readable JSON
  - sarif: SARIF v2.1.0 for GitHub/GitLab Code Scanning

Examples:
  # Validate single template
  template-validator validate slack.tmpl

  # Validate directory (recursive)
  template-validator validate templates/ --recursive

  # Strict validation (fail on warnings)
  template-validator validate templates/ --mode=strict

  # JSON output for CI/CD
  template-validator validate templates/ --output=json

  # SARIF output for code scanning
  template-validator validate templates/ --output=sarif > results.sarif

  # Only syntax and security (skip semantic and best_practices)
  template-validator validate templates/ --phases=syntax,security
`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	// Validation mode flags
	validateCmd.Flags().StringVar(&mode, "mode", "lenient", "Validation mode: strict, lenient, permissive")
	validateCmd.Flags().StringVar(&templateType, "type", "generic", "Template type: slack, pagerduty, email, webhook, generic")
	validateCmd.Flags().BoolVar(&failOnWarning, "fail-on-warning", false, "Treat warnings as errors (exit code 1)")
	validateCmd.Flags().IntVar(&maxErrors, "max-errors", 0, "Stop after N errors (0 = collect all)")
	validateCmd.Flags().StringSliceVar(&phases, "phases", []string{}, "Validation phases (default: all)")

	// Output flags
	validateCmd.Flags().StringVar(&output, "output", "human", "Output format: human, json, sarif")
	validateCmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")

	// Batch processing flags
	validateCmd.Flags().BoolVar(&recursive, "recursive", false, "Recursive directory traversal")
	validateCmd.Flags().StringVar(&pattern, "pattern", "*.tmpl", "File pattern (glob)")
	validateCmd.Flags().IntVar(&parallel, "parallel", 0, "Parallel workers (0 = CPU count)")
}

// ================================================================================

// runValidate executes the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Check if target exists
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target not found: %s", target)
	}

	// Create validator (using nil engine for now - will integrate TN-153 in Phase 9)
	validator := createValidator()

	// Build validation options
	opts := buildValidateOptions()

	// Determine if target is file or directory
	var results []templatevalidator.ValidationResult
	var paths []string

	if info.IsDir() {
		// Directory validation
		paths, err = findTemplateFiles(target, recursive, pattern)
		if err != nil {
			return fmt.Errorf("failed to find template files: %w", err)
		}

		if !quiet {
			fmt.Printf("Validating %d template(s)...\n\n", len(paths))
		}

		// Batch validation
		results, err = validateBatch(cmd.Context(), validator, paths, opts)
		if err != nil {
			return fmt.Errorf("batch validation failed: %w", err)
		}
	} else {
		// Single file validation
		paths = []string{target}

		result, err := validator.ValidateFile(cmd.Context(), target, opts)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		results = []templatevalidator.ValidationResult{*result}
	}

	// Format and print results
	exitCode, err := printResults(results, paths)
	if err != nil {
		return fmt.Errorf("failed to print results: %w", err)
	}

	// Exit with appropriate code
	os.Exit(exitCode)
	return nil
}

// ================================================================================

// createValidator creates a Validator instance
//
// TODO: Phase 9 - Integrate with TN-153 NotificationTemplateEngine
func createValidator() templatevalidator.Validator {
	// For now, create mock validator
	// In Phase 9, we'll integrate with TN-153 engine
	return nil // Placeholder
}

// buildValidateOptions builds ValidateOptions from CLI flags
func buildValidateOptions() templatevalidator.ValidateOptions {
	opts := templatevalidator.DefaultValidateOptions()

	// Set mode
	switch mode {
	case "strict":
		opts.Mode = templatevalidator.ModeStrict
	case "lenient":
		opts.Mode = templatevalidator.ModeLenient
	case "permissive":
		opts.Mode = templatevalidator.ModePermissive
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown mode '%s', using 'lenient'\n", mode)
		opts.Mode = templatevalidator.ModeLenient
	}

	// Set template type
	opts.TemplateType = templateType

	// Set max errors
	opts.MaxErrors = maxErrors

	// Set parallel workers
	opts.ParallelWorkers = parallel

	// Set phases
	if len(phases) > 0 {
		opts.Phases = []templatevalidator.ValidationPhase{}
		for _, p := range phases {
			switch p {
			case "syntax":
				opts.Phases = append(opts.Phases, templatevalidator.PhaseSyntax)
			case "semantic":
				opts.Phases = append(opts.Phases, templatevalidator.PhaseSemantic)
			case "security":
				opts.Phases = append(opts.Phases, templatevalidator.PhaseSecurity)
			case "best_practices":
				opts.Phases = append(opts.Phases, templatevalidator.PhaseBestPractices)
			default:
				fmt.Fprintf(os.Stderr, "Warning: unknown phase '%s', skipping\n", p)
			}
		}
	}

	return opts
}

// ================================================================================

// findTemplateFiles finds template files matching pattern
func findTemplateFiles(dir string, recursive bool, pattern string) ([]string, error) {
	var files []string

	if recursive {
		// Recursive traversal
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}
				if matched {
					files = append(files, path)
				}
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	} else {
		// Non-recursive (only direct children)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				matched, err := filepath.Match(pattern, entry.Name())
				if err != nil {
					return nil, err
				}
				if matched {
					files = append(files, filepath.Join(dir, entry.Name()))
				}
			}
		}
	}

	return files, nil
}

// ================================================================================

// validateBatch validates multiple templates
func validateBatch(
	ctx context.Context,
	validator templatevalidator.Validator,
	paths []string,
	opts templatevalidator.ValidateOptions,
) ([]templatevalidator.ValidationResult, error) {
	// Create template inputs
	inputs := make([]templatevalidator.TemplateInput, len(paths))
	for i, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		inputs[i] = templatevalidator.TemplateInput{
			Name:    path,
			Content: string(content),
			Options: opts,
		}
	}

	// Batch validate with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	results, err := validator.ValidateBatch(ctx, inputs)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// ================================================================================

// printResults formats and prints validation results
//
// Returns exit code:
// - 0: success (no errors)
// - 1: errors found
// - 2: warnings only
func printResults(results []templatevalidator.ValidationResult, paths []string) (int, error) {
	// Create formatter
	var formatter formatters.OutputFormatter

	switch output {
	case "json":
		formatter = formatters.NewJSONFormatter()
	case "sarif":
		formatter = formatters.NewSARIFFormatter()
	case "human":
		fallthrough
	default:
		formatter = formatters.NewHumanFormatter()
	}

	// Format results
	formattedOutput, err := formatter.Format(results, paths)
	if err != nil {
		return 1, fmt.Errorf("failed to format results: %w", err)
	}

	// Print output
	fmt.Print(formattedOutput)

	// Determine exit code
	exitCode := determineExitCode(results)

	return exitCode, nil
}

// ================================================================================

// determineExitCode determines exit code based on results
//
// Exit codes:
// - 0: no errors
// - 1: errors found
// - 2: warnings only (if failOnWarning is false)
func determineExitCode(results []templatevalidator.ValidationResult) int {
	hasErrors := false
	hasWarnings := false

	for _, result := range results {
		if result.HasErrors() {
			hasErrors = true
		}
		if result.HasWarnings() {
			hasWarnings = true
		}
	}

	// Exit code 1: errors found
	if hasErrors {
		return 1
	}

	// Exit code 1: warnings found and failOnWarning enabled
	if hasWarnings && failOnWarning {
		return 1
	}

	// Exit code 2: warnings only (and failOnWarning disabled)
	if hasWarnings {
		return 2
	}

	// Exit code 0: success
	return 0
}

// ================================================================================

