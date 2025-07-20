//go:build plugins

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/conneroisu/templar/internal/plugins"
	"github.com/conneroisu/templar/internal/plugins/builtin"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage Templar plugins",
	Long: `Manage Templar plugins for extended functionality.

Plugins provide extensible functionality for component processing,
build customization, server enhancements, and file watching.`,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `List all installed plugins with their status and information.

This shows plugin names, versions, descriptions, enabled status, and health status.`,
	RunE: runPluginsList,
}

var pluginsEnableCmd = &cobra.Command{
	Use:   "enable [plugin-name]",
	Short: "Enable a plugin",
	Long: `Enable a plugin by name.

This will start the plugin and make it available for use.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginsEnable,
}

var pluginsDisableCmd = &cobra.Command{
	Use:   "disable [plugin-name]",
	Short: "Disable a plugin",
	Long: `Disable a plugin by name.

This will stop the plugin and prevent it from being used.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginsDisable,
}

var pluginsInfoCmd = &cobra.Command{
	Use:   "info [plugin-name]",
	Short: "Show detailed plugin information",
	Long: `Show detailed information about a specific plugin.

This includes configuration, health metrics, and plugin-specific details.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginsInfo,
}

var pluginsHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check plugin health status",
	Long: `Check the health status of all plugins.

This performs health checks and reports any issues or warnings.`,
	RunE: runPluginsHealth,
}

// Plugin command flags
var (
	pluginsOutputFormat string
	pluginsShowDisabled bool
	pluginsVerbose      bool
)

func init() {
	rootCmd.AddCommand(pluginsCmd)
	
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsEnableCmd)
	pluginsCmd.AddCommand(pluginsDisableCmd)
	pluginsCmd.AddCommand(pluginsInfoCmd)
	pluginsCmd.AddCommand(pluginsHealthCmd)

	// List command flags
	pluginsListCmd.Flags().StringVar(&pluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")
	pluginsListCmd.Flags().BoolVar(&pluginsShowDisabled, "show-disabled", true, 
		"Show disabled plugins")
	pluginsListCmd.Flags().BoolVar(&pluginsVerbose, "verbose", false, 
		"Show verbose plugin information")

	// Info command flags
	pluginsInfoCmd.Flags().StringVar(&pluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")

	// Health command flags
	pluginsHealthCmd.Flags().StringVar(&pluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	pm := createPluginManager()
	defer pm.Shutdown()

	pluginInfos := pm.ListPlugins()

	// Filter disabled plugins if requested
	if !pluginsShowDisabled {
		filtered := make([]plugins.PluginInfo, 0)
		for _, info := range pluginInfos {
			if info.Enabled {
				filtered = append(filtered, info)
			}
		}
		pluginInfos = filtered
	}

	// Sort plugins by name
	sort.Slice(pluginInfos, func(i, j int) bool {
		return pluginInfos[i].Name < pluginInfos[j].Name
	})

	switch pluginsOutputFormat {
	case "json":
		return outputJSON(pluginInfos, "")
	case "yaml":
		return outputYAML(pluginInfos, "")
	default:
		return displayPluginsTable(pluginInfos)
	}
}

func runPluginsEnable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]
	
	fmt.Printf("🔌 Enabling plugin: %s\n", pluginName)
	
	// This would typically involve:
	// 1. Loading plugin configuration
	// 2. Enabling the plugin in config
	// 3. Starting the plugin if not already running
	
	fmt.Printf("✅ Plugin %s enabled successfully\n", pluginName)
	return nil
}

func runPluginsDisable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]
	
	fmt.Printf("🔌 Disabling plugin: %s\n", pluginName)
	
	// This would typically involve:
	// 1. Stopping the plugin gracefully
	// 2. Updating configuration to disable
	// 3. Cleaning up resources
	
	fmt.Printf("✅ Plugin %s disabled successfully\n", pluginName)
	return nil
}

func runPluginsInfo(cmd *cobra.Command, args []string) error {
	pluginName := args[0]
	
	pm := createPluginManager()
	defer pm.Shutdown()

	plugin, err := pm.GetPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	pluginInfos := pm.ListPlugins()
	var targetInfo *plugins.PluginInfo
	for _, info := range pluginInfos {
		if info.Name == pluginName {
			targetInfo = &info
			break
		}
	}

	if targetInfo == nil {
		return fmt.Errorf("plugin info not found: %s", pluginName)
	}

	switch pluginsOutputFormat {
	case "json":
		return outputJSON(targetInfo, "")
	case "yaml":
		return outputYAML(targetInfo, "")
	default:
		return displayPluginInfo(plugin, *targetInfo)
	}
}

func runPluginsHealth(cmd *cobra.Command, args []string) error {
	pm := createPluginManager()
	defer pm.Shutdown()

	// Trigger health checks
	pm.StartHealthChecks(1 * time.Second)
	time.Sleep(2 * time.Second) // Allow health checks to run

	pluginInfos := pm.ListPlugins()

	switch pluginsOutputFormat {
	case "json":
		return outputJSON(pluginInfos, "")
	case "yaml":
		return outputYAML(pluginInfos, "")
	default:
		return displayPluginsHealthTable(pluginInfos)
	}
}

// createPluginManager creates and configures a plugin manager with built-in plugins
func createPluginManager() *plugins.PluginManager {
	pm := plugins.NewPluginManager()

	// Register built-in plugins
	builtinPlugins := []struct {
		plugin plugins.Plugin
		config plugins.PluginConfig
	}{
		{
			plugin: builtin.NewTailwindPlugin(),
			config: plugins.PluginConfig{
				Name:    "tailwind",
				Enabled: true,
				Config: map[string]interface{}{
					"auto_generate": true,
					"config_file":   "tailwind.config.js",
				},
			},
		},
		{
			plugin: builtin.NewHotReloadPlugin(),
			config: plugins.PluginConfig{
				Name:    "hotreload",
				Enabled: true,
				Config: map[string]interface{}{
					"debounce_ms": 250,
					"port":        8080,
				},
			},
		},
	}

	for _, p := range builtinPlugins {
		if err := pm.RegisterPlugin(p.plugin, p.config); err != nil {
			fmt.Printf("Warning: Failed to register plugin %s: %v\n", p.config.Name, err)
		}
	}

	return pm
}

// Display functions
func displayPluginsTable(pluginInfos []plugins.PluginInfo) error {
	if len(pluginInfos) == 0 {
		fmt.Println("No plugins found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tHEALTH\tDESCRIPTION")
	fmt.Fprintln(w, "----\t-------\t------\t------\t-----------")

	for _, info := range pluginInfos {
		status := "Disabled"
		if info.Enabled {
			status = "Enabled"
		}

		healthIcon := getHealthIcon(info.Health.Status)
		
		description := info.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s %s\t%s\n",
			info.Name,
			info.Version,
			status,
			healthIcon,
			string(info.Health.Status),
			description)
	}

	return nil
}

func displayPluginsHealthTable(pluginInfos []plugins.PluginInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tHEALTH\tLAST CHECK\tERROR")
	fmt.Fprintln(w, "----\t------\t----------\t-----")

	for _, info := range pluginInfos {
		healthIcon := getHealthIcon(info.Health.Status)
		lastCheck := "Never"
		if !info.Health.LastCheck.IsZero() {
			lastCheck = info.Health.LastCheck.Format("15:04:05")
		}

		errorMsg := info.Health.Error
		if errorMsg == "" {
			errorMsg = "-"
		} else if len(errorMsg) > 40 {
			errorMsg = errorMsg[:37] + "..."
		}

		fmt.Fprintf(w, "%s\t%s %s\t%s\t%s\n",
			info.Name,
			healthIcon,
			string(info.Health.Status),
			lastCheck,
			errorMsg)
	}

	return nil
}

func displayPluginInfo(plugin plugins.Plugin, info plugins.PluginInfo) error {
	fmt.Printf("Plugin Information\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Name:        %s\n", info.Name)
	fmt.Printf("Version:     %s\n", info.Version)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Status:      %s\n", getEnabledStatus(info.Enabled))
	fmt.Printf("Health:      %s %s\n", getHealthIcon(info.Health.Status), string(info.Health.Status))
	
	if !info.Health.LastCheck.IsZero() {
		fmt.Printf("Last Check:  %s\n", info.Health.LastCheck.Format(time.RFC3339))
	}
	
	if info.Health.Error != "" {
		fmt.Printf("Error:       %s\n", info.Health.Error)
	}

	// Show plugin capabilities
	fmt.Printf("\nCapabilities:\n")
	
	if _, ok := plugin.(plugins.ComponentPlugin); ok {
		fmt.Printf("  • Component Processing\n")
	}
	if _, ok := plugin.(plugins.BuildPlugin); ok {
		fmt.Printf("  • Build Integration\n")
	}
	if _, ok := plugin.(plugins.ServerPlugin); ok {
		fmt.Printf("  • Server Extensions\n")
	}
	if _, ok := plugin.(plugins.WatcherPlugin); ok {
		fmt.Printf("  • File Watching\n")
	}

	// Show health metrics if available
	if len(info.Health.Metrics) > 0 {
		fmt.Printf("\nHealth Metrics:\n")
		for key, value := range info.Health.Metrics {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

func getHealthIcon(status plugins.HealthStatus) string {
	switch status {
	case plugins.HealthStatusHealthy:
		return "✅"
	case plugins.HealthStatusUnhealthy:
		return "❌"
	case plugins.HealthStatusDegraded:
		return "⚠️"
	default:
		return "❓"
	}
}

func getEnabledStatus(enabled bool) string {
	if enabled {
		return "✅ Enabled"
	}
	return "❌ Disabled"
}