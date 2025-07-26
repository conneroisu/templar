package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/di"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List all discovered components",
	Long: `List all discovered components in the project with their metadata.
Shows component names, file paths, and optionally parameters and dependencies.

Examples:
  templar list                    # List all components in table format
  templar list -f json            # Output as JSON (short flag)
  templar list --format csv       # Output as CSV
  templar list -p                 # Include component properties (short flag)
  templar list -d                 # Include dependencies (short flag)
  templar list -pd -f yaml        # Include properties and deps, output as YAML`,
	RunE: runList,
}

var (
	listFlags     *StandardFlags
	listWithDeps  bool
	listWithProps bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	// Use standardized flags
	listFlags = AddStandardFlags(listCmd, "output")

	// Add list-specific flags with short aliases
	listCmd.Flags().
		BoolVarP(&listWithDeps, "with-deps", "d", false, "Include component dependencies")
	listCmd.Flags().
		BoolVarP(&listWithProps, "with-props", "p", false, "Include component properties/parameters")

	// Add format validation
	AddFlagValidation(listCmd, "format", func(format string) error {
		return ValidateFormatWithSuggestion(format, []string{"table", "json", "yaml", "csv"})
	})
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize dependency injection container
	container := di.NewServiceContainer(cfg)
	if err := container.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize service container: %w", err)
	}
	defer func() {
		if shutdownErr := container.Shutdown(context.Background()); shutdownErr != nil {
			fmt.Printf("Warning: Error during container shutdown: %v\n", shutdownErr)
		}
	}()

	// Get services from container
	componentRegistry, err := container.GetRegistry()
	if err != nil {
		return fmt.Errorf("failed to get component registry: %w", err)
	}

	componentScanner, err := container.GetScanner()
	if err != nil {
		return fmt.Errorf("failed to get component scanner: %w", err)
	}

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

	// Validate flags
	if err := listFlags.ValidateFlags(); err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	// Output in requested format
	switch strings.ToLower(listFlags.Format) {
	case "json":
		return outputListJSON(componentSlice)
	case "yaml":
		return outputYAML(componentSlice)
	case "table":
		return outputTable(componentSlice)
	case "csv":
		return outputListCSV(componentSlice)
	default:
		return fmt.Errorf("unsupported format: %s", listFlags.Format)
	}
}

func outputListJSON(components []*types.ComponentInfo) error {
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
	separator := strings.Repeat(
		"-",
		4,
	) + "\t" + strings.Repeat(
		"-",
		7,
	) + "\t" + strings.Repeat(
		"-",
		4,
	) + "\t" + strings.Repeat(
		"-",
		8,
	)
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

func outputListCSV(components []*types.ComponentInfo) error {
	// Write header
	header := "name,package,file_path,function"
	if listWithProps {
		header += ",parameters"
	}
	if listWithDeps {
		header += ",dependencies"
	}
	fmt.Println(header)

	// Write components
	for _, component := range components {
		row := fmt.Sprintf("%s,%s,%s,%s",
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
			row += "," + strings.Join(params, ";")
		}

		if listWithDeps {
			row += "," // Empty for now
		}

		fmt.Println(row)
	}

	return nil
}
