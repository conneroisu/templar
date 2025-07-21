package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered components",
	Long: `List all discovered components in the project with their metadata.
Shows component names, file paths, and optionally parameters and dependencies.

Examples:
  templar list                    # List all components in table format
  templar list --format json     # Output as JSON
  templar list --with-props       # Include component properties
  templar list --with-deps        # Include dependencies`,
	RunE: runList,
}

var (
	listFormat    string
	listWithDeps  bool
	listWithProps bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "Output format (table, json, yaml)")
	listCmd.Flags().BoolVar(&listWithDeps, "with-deps", false, "Include dependencies")
	listCmd.Flags().BoolVar(&listWithProps, "with-props", false, "Include component properties")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Scan all configured paths
	for _, scanPath := range cfg.Components.ScanPaths {
		if err := componentScanner.ScanDirectory(scanPath); err != nil {
			// Log error but continue with other paths
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", scanPath, err)
		}
	}

	// Get all components
	components := componentRegistry.GetAll()

	if len(components) == 0 {
		fmt.Println("No components found.")
		return nil
	}

	// Convert map to slice for output
	componentSlice := make([]*types.ComponentInfo, 0, len(components))
	for _, comp := range components {
		componentSlice = append(componentSlice, comp)
	}

	// Output in requested format
	switch strings.ToLower(listFormat) {
	case "json":
		return outputJSON(componentSlice)
	case "yaml":
		return outputYAML(componentSlice)
	case "table":
		return outputTable(componentSlice)
	default:
		return fmt.Errorf("unsupported format: %s", listFormat)
	}
}

func outputJSON(components []*types.ComponentInfo) error {
	output := make([]map[string]interface{}, len(components))

	for i, component := range components {
		item := map[string]interface{}{
			"name":      component.Name,
			"package":   component.Package,
			"file_path": component.FilePath,
			"function":  component.Name,
		}

		if listWithProps {
			params := make([]map[string]string, len(component.Parameters))
			for j, param := range component.Parameters {
				params[j] = map[string]string{
					"name": param.Name,
					"type": param.Type,
				}
			}
			item["parameters"] = params
		}

		if listWithDeps {
			item["dependencies"] = component.Dependencies
		}

		output[i] = item
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputYAML(components []*types.ComponentInfo) error {
	output := make([]map[string]interface{}, len(components))

	for i, component := range components {
		item := map[string]interface{}{
			"name":      component.Name,
			"package":   component.Package,
			"file_path": component.FilePath,
			"function":  component.Name,
		}

		if listWithProps {
			params := make([]map[string]string, len(component.Parameters))
			for j, param := range component.Parameters {
				params[j] = map[string]string{
					"name": param.Name,
					"type": param.Type,
				}
			}
			item["parameters"] = params
		}

		if listWithDeps {
			item["dependencies"] = []string{}
		}

		output[i] = item
	}

	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(output)
}

func outputTable(components []*types.ComponentInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Write header
	header := "NAME\tPACKAGE\tFILE\tFUNCTION"
	if listWithProps {
		header += "\tPARAMETERS"
	}
	if listWithDeps {
		header += "\tDEPENDENCIES"
	}
	fmt.Fprintln(w, header)

	// Write separator
	separator := strings.Repeat("-", 4) + "\t" + strings.Repeat("-", 7) + "\t" + strings.Repeat("-", 4) + "\t" + strings.Repeat("-", 8)
	if listWithProps {
		separator += "\t" + strings.Repeat("-", 10)
	}
	if listWithDeps {
		separator += "\t" + strings.Repeat("-", 12)
	}
	fmt.Fprintln(w, separator)

	// Write components
	for _, component := range components {
		row := fmt.Sprintf("%s\t%s\t%s\t%s",
			component.Name,
			component.Package,
			component.FilePath,
			component.Name,
		)

		if listWithProps {
			var params []string
			for _, param := range component.Parameters {
				params = append(params, fmt.Sprintf("%s:%s", param.Name, param.Type))
			}
			row += "\t" + strings.Join(params, ", ")
		}

		if listWithDeps {
			row += "\t" + "" // Empty for now
		}

		fmt.Fprintln(w, row)
	}

	// Write summary
	fmt.Fprintf(w, "\nTotal: %d components\n", len(components))

	return nil
}
