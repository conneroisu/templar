package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	generateAll    bool
	generateFormat string
	generateOutput string
	generatePaths  []string
)

// generateCmd represents the generate command.
var generateCmd = &cobra.Command{
	Use:   "generate [component...]",
	Short: "Generate code for templ components",
	Long: `Generate code for templ components including:

- Go code generation from templ files
- Type definitions for component parameters
- Mock data generators for testing
- Component documentation

Examples:
  templar generate                     # Generate code for all components
  templar generate Button Card         # Generate code for specific components
  templar generate --format go        # Generate Go code only
  templar generate --output ./gen     # Output to specific directory`,
	RunE: runGenerateCommand,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().
		BoolVar(&generateAll, "all", false, "Generate code for all components (default if no components specified)")
	generateCmd.Flags().
		StringVarP(&generateFormat, "format", "f", "go", "Output format (go, types, mocks, docs)")
	generateCmd.Flags().
		StringVarP(&generateOutput, "output", "o", "", "Output directory (default: current directory)")
	generateCmd.Flags().
		StringSliceVar(&generatePaths, "path", nil, "Additional paths to scan for components")
}

type GenerateResult struct {
	Component string   `json:"component"`
	Files     []string `json:"files"`
	Success   bool     `json:"success"`
	Error     string   `json:"error,omitempty"`
}

type GenerateSummary struct {
	Total     int              `json:"total"`
	Success   int              `json:"success"`
	Failed    int              `json:"failed"`
	Results   []GenerateResult `json:"results"`
	OutputDir string           `json:"output_dir"`
}

func runGenerateCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set up component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Add configured scan paths
	scanPaths := cfg.Components.ScanPaths
	if len(generatePaths) > 0 {
		scanPaths = append(scanPaths, generatePaths...)
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
		fmt.Println("No components found to generate code for")

		return nil
	}

	// Determine which components to generate for
	var componentsToGenerate []*types.ComponentInfo
	if len(args) == 0 || generateAll {
		componentsToGenerate = allComponents
	} else {
		// Generate for specific components
		componentMap := make(map[string]*types.ComponentInfo)
		for _, comp := range allComponents {
			componentMap[comp.Name] = comp
		}

		for _, name := range args {
			if comp, exists := componentMap[name]; exists {
				componentsToGenerate = append(componentsToGenerate, comp)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: component '%s' not found\n", name)
			}
		}
	}

	if len(componentsToGenerate) == 0 {
		return errors.New("no valid components specified for generation")
	}

	// Set up output directory
	outputDir := generateOutput
	if outputDir == "" {
		outputDir = "."
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Perform generation
	summary := GenerateSummary{
		Total:     len(componentsToGenerate),
		Results:   make([]GenerateResult, 0, len(componentsToGenerate)),
		OutputDir: outputDir,
	}

	for _, component := range componentsToGenerate {
		result := generateComponentCode(component, outputDir, generateFormat)
		summary.Results = append(summary.Results, result)

		if result.Success {
			summary.Success++
		} else {
			summary.Failed++
		}
	}

	// Output results
	return outputGenerateResults(summary)
}

func generateComponentCode(
	component *types.ComponentInfo,
	outputDir, format string,
) GenerateResult {
	result := GenerateResult{
		Component: component.Name,
		Files:     make([]string, 0),
		Success:   true,
	}

	switch format {
	case "go":
		if err := generateGoCode(component, outputDir, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
		}
	case "types":
		if err := generateTypeDefinitions(component, outputDir, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
		}
	case "mocks":
		if err := generateMockDataFile(component, outputDir, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
		}
	case "docs":
		if err := generateDocumentation(component, outputDir, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
		}
	default:
		result.Success = false
		result.Error = "unsupported format: " + format
	}

	return result
}

func generateGoCode(
	component *types.ComponentInfo,
	outputDir string,
	result *GenerateResult,
) error {
	// For now, just create a placeholder Go file
	fileName := strings.ToLower(component.Name) + "_generated.go"
	filePath := filepath.Join(outputDir, fileName)

	content := fmt.Sprintf(`// Code generated by templar. DO NOT EDIT.

package main

import (
	"context"
)

// %sProps contains the properties for the %s component
type %sProps struct {
`, component.Name, component.Name, component.Name)

	for _, param := range component.Parameters {
		content += fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
			cases.Title(language.English).String(param.Name), param.Type, param.Name)
	}

	content += "}\n\n"

	content += fmt.Sprintf(`// Render%s renders the %s component
func Render%s(ctx context.Context, props %sProps) error {
	// Implementation would go here
	return nil
}
`, component.Name, component.Name, component.Name, component.Name)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Go file: %w", err)
	}

	result.Files = append(result.Files, filePath)

	return nil
}

func generateTypeDefinitions(
	component *types.ComponentInfo,
	outputDir string,
	result *GenerateResult,
) error {
	fileName := strings.ToLower(component.Name) + "_types.ts"
	filePath := filepath.Join(outputDir, fileName)

	content := fmt.Sprintf("// Type definitions for %s component\n\n", component.Name)
	content += fmt.Sprintf("export interface %sProps {\n", component.Name)

	for _, param := range component.Parameters {
		tsType := convertGoTypeToTypeScript(param.Type)
		optional := ""
		if param.Optional {
			optional = "?"
		}
		content += fmt.Sprintf("  %s%s: %s;\n", param.Name, optional, tsType)
	}

	content += "}\n"

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write types file: %w", err)
	}

	result.Files = append(result.Files, filePath)

	return nil
}

func generateMockDataFile(
	component *types.ComponentInfo,
	outputDir string,
	result *GenerateResult,
) error {
	fileName := strings.ToLower(component.Name) + "_mock.json"
	filePath := filepath.Join(outputDir, fileName)

	mockData := make(map[string]interface{})
	for _, param := range component.Parameters {
		mockData[param.Name] = generateMockValueForType(param.Type)
	}

	data, err := json.MarshalIndent(mockData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mock data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mock file: %w", err)
	}

	result.Files = append(result.Files, filePath)

	return nil
}

func generateDocumentation(
	component *types.ComponentInfo,
	outputDir string,
	result *GenerateResult,
) error {
	fileName := strings.ToLower(component.Name) + ".md"
	filePath := filepath.Join(outputDir, fileName)

	content := fmt.Sprintf("# %s Component\n\n", component.Name)
	content += fmt.Sprintf("File: `%s`\n\n", component.FilePath)

	if len(component.Parameters) > 0 {
		content += "## Parameters\n\n"
		content += "| Name | Type | Optional | Default |\n"
		content += "|------|------|----------|----------|\n"

		for _, param := range component.Parameters {
			optional := "No"
			if param.Optional {
				optional = "Yes"
			}
			defaultVal := "-"
			if param.Default != nil {
				defaultVal = fmt.Sprintf("%v", param.Default)
			}
			content += fmt.Sprintf("| %s | %s | %s | %s |\n",
				param.Name, param.Type, optional, defaultVal)
		}
		content += "\n"
	}

	if len(component.Dependencies) > 0 {
		content += "## Dependencies\n\n"
		for _, dep := range component.Dependencies {
			content += fmt.Sprintf("- %s\n", dep)
		}
		content += "\n"
	}

	content += "## Usage\n\n"
	content += fmt.Sprintf("```templ\n@%s(", component.Name)

	if len(component.Parameters) > 0 {
		for i, param := range component.Parameters {
			if i > 0 {
				content += ", "
			}
			content += fmt.Sprintf("%s: %s", param.Name, generateExampleValue(param.Type))
		}
	}

	content += ")\n```\n"

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write documentation file: %w", err)
	}

	result.Files = append(result.Files, filePath)

	return nil
}

func convertGoTypeToTypeScript(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "[]string":
		return "string[]"
	case "[]int":
		return "number[]"
	default:
		return "any"
	}
}

func generateMockValueForType(goType string) interface{} {
	switch goType {
	case "string":
		return "Sample text"
	case "int", "int32", "int64":
		return 42
	case "float32", "float64":
		return 3.14
	case "bool":
		return true
	case "[]string":
		return []string{"item1", "item2"}
	case "[]int":
		return []int{1, 2, 3}
	default:
		return nil
	}
}

func generateExampleValue(goType string) string {
	switch goType {
	case "string":
		return `"example"`
	case "int", "int32", "int64":
		return "123"
	case "float32", "float64":
		return "3.14"
	case "bool":
		return "true"
	case "[]string":
		return `["item1", "item2"]`
	case "[]int":
		return "[1, 2, 3]"
	default:
		return "nil"
	}
}

func outputGenerateResults(summary GenerateSummary) error {
	fmt.Printf("Code Generation Summary:\n")
	fmt.Printf("  Total components: %d\n", summary.Total)
	fmt.Printf("  Successful: %d\n", summary.Success)
	fmt.Printf("  Failed: %d\n", summary.Failed)
	fmt.Printf("  Output directory: %s\n", summary.OutputDir)
	fmt.Println()

	// Show component results
	for _, result := range summary.Results {
		status := "✅"
		if !result.Success {
			status = "❌"
		}

		fmt.Printf("%s %s\n", status, result.Component)

		if result.Success {
			for _, file := range result.Files {
				fmt.Printf("    Generated: %s\n", file)
			}
		} else {
			fmt.Printf("    Error: %s\n", result.Error)
		}

		fmt.Println()
	}

	if summary.Failed > 0 {
		return fmt.Errorf("code generation failed for %d components", summary.Failed)
	}

	fmt.Println("✅ Code generation completed successfully!")

	return nil
}
