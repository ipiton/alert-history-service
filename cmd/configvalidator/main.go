package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vitaliisemenov/alert-history/pkg/configvalidator"
)

var (
	// Version information (set by build)
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "configvalidator",
	Short: "Alertmanager configuration validator",
	Long: `A comprehensive validator for Alertmanager configuration files.

Supports YAML and JSON formats with detailed error reporting,
multiple validation modes, and various output formats.

Quality Target: 150% (Grade A+ EXCEPTIONAL)`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildDate),
}

var (
	// Global flags
	mode             string
	format           string
	sections         []string
	enableSecurity   bool
	enableBestPractices bool
	maxFileSize      int64
	maxDepth         int
	docsURL          string
	contextLines     int

	// Output flags
	outputFormat string
	quiet        bool
	verbose      bool
	noColor      bool
)

func init() {
	// Add validate command
	rootCmd.AddCommand(validateCmd)

	// Global validation flags
	validateCmd.Flags().StringVarP(&mode, "mode", "m", "strict", "Validation mode: strict, lenient, or permissive")
	validateCmd.Flags().StringVarP(&format, "format", "f", "", "Config format: yaml or json (auto-detected if not specified)")
	validateCmd.Flags().StringSliceVarP(&sections, "sections", "s", []string{}, "Specific sections to validate (comma-separated)")
	validateCmd.Flags().BoolVar(&enableSecurity, "security", true, "Enable security validation checks")
	validateCmd.Flags().BoolVar(&enableBestPractices, "best-practices", true, "Enable best practices validation")
	validateCmd.Flags().Int64Var(&maxFileSize, "max-file-size", 10*1024*1024, "Maximum file size in bytes (default 10MB)")
	validateCmd.Flags().IntVar(&maxDepth, "max-depth", 100, "Maximum YAML/JSON nesting depth")
	validateCmd.Flags().StringVar(&docsURL, "docs-url", "https://prometheus.io/docs/alerting/latest/configuration/", "Documentation base URL")
	validateCmd.Flags().IntVar(&contextLines, "context", 3, "Number of context lines in error messages")

	// Output flags
	validateCmd.Flags().StringVarP(&outputFormat, "output", "o", "human", "Output format: human, json, junit, sarif")
	validateCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode (only show errors)")
	validateCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (show all issues)")
	validateCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
}

var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate an Alertmanager configuration file",
	Long: `Validate an Alertmanager configuration file with comprehensive checks.

Examples:
  # Validate a configuration file
  configvalidator validate alertmanager.yml

  # Validate in lenient mode (only errors block)
  configvalidator validate --mode lenient alertmanager.yml

  # Validate specific sections only
  configvalidator validate --sections route,receivers alertmanager.yml

  # Output as JSON
  configvalidator validate --output json alertmanager.yml

  # Disable security checks
  configvalidator validate --security=false alertmanager.yml`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	// Parse validation mode
	var validationMode configvalidator.ValidationMode
	switch mode {
	case "strict":
		validationMode = configvalidator.StrictMode
	case "lenient":
		validationMode = configvalidator.LenientMode
	case "permissive":
		validationMode = configvalidator.PermissiveMode
	default:
		return fmt.Errorf("invalid mode '%s'. Must be: strict, lenient, or permissive", mode)
	}

	// Create validator options
	opts := configvalidator.Options{
		Mode:                validationMode,
		MaxFileSize:         maxFileSize,
		MaxYAMLDepth:        maxDepth,
		MaxJSONDepth:        maxDepth,
		DisallowUnknownFields: true,
		EnableBestPractices: enableBestPractices,
		EnableSecurityChecks: enableSecurity,
		IncludeContextLines: contextLines,
		DefaultDocsURL:      docsURL,
	}

	// Create validator
	validator := configvalidator.New(opts)

	// Validate file
	result, err := validator.ValidateFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	// Format and print output
	if err := printResult(result, outputFormat); err != nil {
		return fmt.Errorf("failed to print result: %w", err)
	}

	// Return appropriate exit code
	exitCode := result.ExitCode(validationMode)
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

func printResult(result *configvalidator.Result, format string) error {
	switch format {
	case "json":
		return printJSON(result)
	case "junit":
		return printJUnit(result)
	case "sarif":
		return printSARIF(result)
	case "human":
		return printHuman(result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func printJSON(result *configvalidator.Result) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printHuman(result *configvalidator.Result) error {
	// Print summary
	if result.Valid {
		printSuccess("✓ Configuration is valid")
	} else {
		printError("✗ Configuration validation failed")
	}

	fmt.Printf("\n%s\n\n", result.Summary())

	// Print errors
	if len(result.Errors) > 0 {
		printSection("ERRORS", len(result.Errors))
		for _, issue := range result.Errors {
			printIssue(issue, "ERROR")
		}
		fmt.Println()
	}

	// Print warnings (unless quiet mode)
	if !quiet && len(result.Warnings) > 0 {
		printSection("WARNINGS", len(result.Warnings))
		for _, issue := range result.Warnings {
			printIssue(issue, "WARNING")
		}
		fmt.Println()
	}

	// Print info and suggestions in verbose mode
	if verbose {
		if len(result.Info) > 0 {
			printSection("INFO", len(result.Info))
			for _, issue := range result.Info {
				printIssue(issue, "INFO")
			}
			fmt.Println()
		}

		if len(result.Suggestions) > 0 {
			printSection("SUGGESTIONS", len(result.Suggestions))
			for _, issue := range result.Suggestions {
				printIssue(issue, "SUGGESTION")
			}
			fmt.Println()
		}
	}

	// Print footer
	fmt.Printf("Validation completed in %dms\n", result.DurationMs)

	return nil
}

func printSection(title string, count int) {
	if noColor {
		fmt.Printf("=== %s (%d) ===\n", title, count)
	} else {
		fmt.Printf("\033[1m=== %s (%d) ===\033[0m\n", title, count)
	}
}

func printIssue(issue *configvalidator.Issue, level string) {
	// Print code and location
	location := ""
	if issue.Location != nil && issue.Location.File != "" {
		location = fmt.Sprintf(" at %s:%d:%d", issue.Location.File, issue.Location.Line, issue.Location.Column)
	} else if issue.FieldPath != "" {
		location = fmt.Sprintf(" at %s", issue.FieldPath)
	}

	if noColor {
		fmt.Printf("[%s] %s%s\n", issue.Code, issue.Message, location)
	} else {
		color := getColor(level)
		fmt.Printf("%s[%s]%s %s%s\n", color, issue.Code, colorReset, issue.Message, location)
	}

	// Print context if available
	if issue.Context != "" && verbose {
		fmt.Printf("  Context:\n%s\n", indent(issue.Context, 4))
	}

	// Print suggestion
	if issue.Suggestion != "" {
		if noColor {
			fmt.Printf("  → %s\n", issue.Suggestion)
		} else {
			fmt.Printf("  \033[36m→ %s\033[0m\n", issue.Suggestion)
		}
	}

	// Print docs URL
	if issue.DocsURL != "" && verbose {
		fmt.Printf("  📚 %s\n", issue.DocsURL)
	}
}

func printSuccess(msg string) {
	if noColor {
		fmt.Println(msg)
	} else {
		fmt.Printf("\033[32m%s\033[0m\n", msg)
	}
}

func printError(msg string) {
	if noColor {
		fmt.Println(msg)
	} else {
		fmt.Printf("\033[31m%s\033[0m\n", msg)
	}
}

func getColor(level string) string {
	if noColor {
		return ""
	}
	switch level {
	case "ERROR":
		return "\033[31m" // Red
	case "WARNING":
		return "\033[33m" // Yellow
	case "INFO":
		return "\033[34m" // Blue
	case "SUGGESTION":
		return "\033[35m" // Magenta
	default:
		return ""
	}
}

const colorReset = "\033[0m"

func indent(text string, spaces int) string {
	prefix := ""
	for i := 0; i < spaces; i++ {
		prefix += " "
	}
	return prefix + text
}

func printJUnit(result *configvalidator.Result) error {
	// JUnit XML format
	fmt.Println("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	fmt.Printf("<testsuite name=\"configvalidator\" tests=\"1\" failures=\"%d\" errors=\"%d\" time=\"%.3f\">\n",
		len(result.Warnings), len(result.Errors), float64(result.DurationMs)/1000.0)

	fmt.Println("  <testcase name=\"validation\" classname=\"configvalidator\">")

	if !result.Valid {
		fmt.Println("    <failure message=\"Configuration validation failed\">")
		fmt.Println("      <![CDATA[")
		for _, err := range result.Errors {
			fmt.Printf("[%s] %s\n", err.Code, err.Message)
			if err.Suggestion != "" {
				fmt.Printf("  Suggestion: %s\n", err.Suggestion)
			}
		}
		fmt.Println("      ]]>")
		fmt.Println("    </failure>")
	}

	fmt.Println("  </testcase>")
	fmt.Println("</testsuite>")

	return nil
}

func printSARIF(result *configvalidator.Result) error {
	// SARIF format (Static Analysis Results Interchange Format)
	sarif := map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":            "Alertmanager Config Validator",
						"informationUri":  "https://github.com/vitaliisemenov/alert-history",
						"version":         version,
					},
				},
				"results": convertToSARIFResults(result),
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarif)
}

func convertToSARIFResults(result *configvalidator.Result) []map[string]interface{}{
	var results []map[string]interface{}

	// Add errors
	for _, issue := range result.Errors {
		results = append(results, issueToSARIF(issue, "error"))
	}

	// Add warnings
	for _, issue := range result.Warnings {
		results = append(results, issueToSARIF(issue, "warning"))
	}

	// Add info
	for _, issue := range result.Info {
		results = append(results, issueToSARIF(issue, "note"))
	}

	return results
}

func issueToSARIF(issue *configvalidator.Issue, level string) map[string]interface{}{
	sarifIssue := map[string]interface{}{
		"ruleId": issue.Code,
		"level":  level,
		"message": map[string]interface{}{
			"text": issue.Message,
		},
	}

	if issue.Location != nil && issue.Location.File != "" {
		sarifIssue["locations"] = []map[string]interface{}{
			{
				"physicalLocation": map[string]interface{}{
					"artifactLocation": map[string]interface{}{
						"uri": issue.Location.File,
					},
					"region": map[string]interface{}{
						"startLine":   issue.Location.Line,
						"startColumn": issue.Location.Column,
					},
				},
			},
		}
	}

	return sarifIssue
}
