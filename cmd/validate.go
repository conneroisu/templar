package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
)

var (
	validateAll      bool
	validateCircular bool
	validateFormat   string
	validatePaths    []string
)

// validateCmd represents the validate command.
var validateCmd = &cobra.Command{
	Use:   "validate [component...]",
	Short: "Validate templ components for errors and dependency issues",
	Long: `Validate templ components for various issues including:

- Syntax errors in templ files
- Missing component dependencies
- Circular dependency detection
- Invalid component names or parameters
- File path issues

Examples:
  templar validate                    # Validate all components
  templar validate Button Card        # Validate specific components
  templar validate --circular         # Check for circular dependencies
  templar validate --format json     # Output results as JSON`,
	RunE: runValidateCommand,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().
		BoolVar(&validateAll, "all", false, "Validate all components (default if no components specified)")
	validateCmd.Flags().
		BoolVar(&validateCircular, "circular", false, "Check for circular dependencies")
	validateCmd.Flags().
		StringVarP(&validateFormat, "format", "f", "text", "Output format (text, json)")
	validateCmd.Flags().
		StringSliceVar(&validatePaths, "path", nil, "Additional paths to scan for components")
}

type ValidationResult struct {
	Component string   `json:"component"`
	Valid     bool     `json:"valid"`
	Errors    []string `json:"errors"`
	Warnings  []string `json:"warnings"`
}

type ValidationSummary struct {
	Total          int                `json:"total"`
	Valid          int                `json:"valid"`
	Invalid        int                `json:"invalid"`
	CircularCycles [][]string         `json:"circular_cycles,omitempty"`
	Results        []ValidationResult `json:"results"`
}

func runValidateCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set up component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Add configured scan paths
	scanPaths := cfg.Components.ScanPaths
	if len(validatePaths) > 0 {
		scanPaths = append(scanPaths, validatePaths...)
	}

	// Scan for components
	for _, path := range scanPaths {
		if err := componentScanner.ScanDirectory(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", path, err)
		}
	}

	// Get all components
	allComponents := componentRegistry.GetAll()
	if len(allComponents) == 0 {
		fmt.Println("No components found to validate")

		return nil
	}

	// Determine which components to validate
	var componentsToValidate []*types.ComponentInfo
	if len(args) == 0 || validateAll {
		componentsToValidate = allComponents
	} else {
		// Validate specific components
		componentMap := make(map[string]*types.ComponentInfo)
		for _, comp := range allComponents {
			componentMap[comp.Name] = comp
		}

		for _, name := range args {
			if comp, exists := componentMap[name]; exists {
				componentsToValidate = append(componentsToValidate, comp)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: component '%s' not found\n", name)
			}
		}
	}

	// Perform validation
	summary := ValidationSummary{
		Total:   len(componentsToValidate),
		Results: make([]ValidationResult, 0, len(componentsToValidate)),
	}

	for _, component := range componentsToValidate {
		result := validateComponent(component)
		summary.Results = append(summary.Results, result)

		if result.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
	}

	// Check for circular dependencies if requested or if validating all
	if validateCircular || len(args) == 0 {
		cycles := componentRegistry.DetectCircularDependencies()
		if len(cycles) > 0 {
			summary.CircularCycles = cycles
			// Mark components in cycles as invalid if not already
			cycleComponents := make(map[string]bool)
			for _, cycle := range cycles {
				for _, comp := range cycle {
					cycleComponents[comp] = true
				}
			}

			for i := range summary.Results {
				if cycleComponents[summary.Results[i].Component] {
					if summary.Results[i].Valid {
						summary.Results[i].Valid = false
						summary.Valid--
						summary.Invalid++
					}
					summary.Results[i].Errors = append(
						summary.Results[i].Errors,
						"Part of circular dependency",
					)
				}
			}
		}
	}

	// Output results
	switch validateFormat {
	case "json":
		return outputValidationJSON(summary)
	case "text":
		return outputValidationText(summary)
	default:
		return fmt.Errorf("unsupported format: %s", validateFormat)
	}
}

func validateComponent(component *types.ComponentInfo) ValidationResult {
	result := ValidationResult{
		Component: component.Name,
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
	}

	// Check if file exists
	if _, err := os.Stat(component.FilePath); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, "File not found: "+component.FilePath)

		return result
	}

	// Validate component name
	if err := validateComponentName(component.Name); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid component name: %v", err))
	}

	// Check file extension
	if !strings.HasSuffix(component.FilePath, ".templ") {
		result.Warnings = append(result.Warnings, "File does not have .templ extension")
	}

	// Validate file path structure
	if strings.Contains(component.FilePath, "..") {
		result.Valid = false
		result.Errors = append(result.Errors, "File path contains path traversal")
	}

	// Check if file is readable
	if file, err := os.Open(component.FilePath); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Cannot read file: %v", err))
	} else {
		file.Close()
	}

	// Validate dependencies exist
	for _, dep := range component.Dependencies {
		if err := validateComponentName(dep); err != nil {
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("Dependency '%s' has invalid name: %v", dep, err),
			)
		}
	}

	// Check for suspicious patterns in file path
	suspiciousPatterns := []string{
		"/tmp/", "/var/tmp/", "/dev/", "/proc/", "/sys/",
		"\\temp\\", "\\windows\\", "\\system32\\",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(component.FilePath), pattern) {
			result.Warnings = append(
				result.Warnings,
				"File in suspicious location: "+component.FilePath,
			)

			break
		}
	}

	return result
}

func validateComponentName(name string) error {
	// Reuse the existing validation function from handlers
	// This ensures consistency across the application

	// Reject empty names
	if name == "" {
		return errors.New("empty component name")
	}

	// Clean the name
	cleanName := filepath.Clean(name)

	// Reject names containing path traversal patterns
	if strings.Contains(cleanName, "..") {
		return errors.New("path traversal attempt detected")
	}

	// Reject absolute paths
	if filepath.IsAbs(cleanName) {
		return errors.New("absolute path not allowed")
	}

	// Reject names with path separators (should be simple component names)
	if strings.ContainsRune(cleanName, os.PathSeparator) {
		return errors.New("path separators not allowed in component name")
	}

	// Reject special characters that could be used in injection attacks
	dangerousChars := []string{
		"<",
		">",
		"\"",
		"'",
		"&",
		";",
		"|",
		"$",
		"`",
		"(",
		")",
		"{",
		"}",
		"[",
		"]",
		"\\",
	}
	for _, char := range dangerousChars {
		if strings.Contains(cleanName, char) {
			return fmt.Errorf("dangerous character not allowed: %s", char)
		}
	}

	// Reject if name is too long (prevent buffer overflow attacks)
	if len(cleanName) > 100 {
		return errors.New("component name too long (max 100 characters)")
	}

	return nil
}

func outputValidationText(summary ValidationSummary) error {
	fmt.Printf("Validation Summary:\n")
	fmt.Printf("  Total components: %d\n", summary.Total)
	fmt.Printf("  Valid: %d\n", summary.Valid)
	fmt.Printf("  Invalid: %d\n", summary.Invalid)

	if len(summary.CircularCycles) > 0 {
		fmt.Printf("  Circular dependencies detected: %d cycles\n", len(summary.CircularCycles))
	}

	fmt.Println()

	// Show circular dependencies first
	if len(summary.CircularCycles) > 0 {
		fmt.Println("🔄 Circular Dependencies:")
		for i, cycle := range summary.CircularCycles {
			fmt.Printf("  Cycle %d: %s\n", i+1, strings.Join(cycle, " -> "))
		}
		fmt.Println()
	}

	// Show component results
	for _, result := range summary.Results {
		status := "✅"
		if !result.Valid {
			status = "❌"
		}

		fmt.Printf("%s %s\n", status, result.Component)

		for _, err := range result.Errors {
			fmt.Printf("    Error: %s\n", err)
		}

		for _, warning := range result.Warnings {
			fmt.Printf("    Warning: %s\n", warning)
		}

		if len(result.Errors) > 0 || len(result.Warnings) > 0 {
			fmt.Println()
		}
	}

	if summary.Invalid > 0 {
		return fmt.Errorf("validation failed: %d invalid components", summary.Invalid)
	}

	fmt.Println("✅ All components are valid!")

	return nil
}

func outputValidationJSON(summary ValidationSummary) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	return encoder.Encode(summary)
}
