//go:build enhanced_plugins

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/plugins"
	"github.com/conneroisu/templar/internal/plugins/builtin"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/spf13/cobra"
)

// Enhanced plugin commands using the new enhanced plugin manager

var enhancedPluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage Templar plugins (enhanced)",
	Long: `Manage Templar plugins for extended functionality with full configuration integration.

Plugins provide extensible functionality for component processing,
build customization, server enhancements, and file watching.`,
}

var enhancedPluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered and loaded plugins",
	Long: `List all discovered and loaded plugins with their status and information.

This shows plugin names, versions, descriptions, enabled status, health status,
and integration capabilities.`,
	RunE: runEnhancedPluginsList,
}

var enhancedPluginsEnableCmd = &cobra.Command{
	Use:   "enable [plugin-name]",
	Short: "Enable a plugin at runtime",
	Long: `Enable a plugin by name at runtime.

This will start the plugin and integrate it with core systems.`,
	Args: cobra.ExactArgs(1),
	RunE: runEnhancedPluginsEnable,
}

var enhancedPluginsDisableCmd = &cobra.Command{
	Use:   "disable [plugin-name]",
	Short: "Disable a plugin at runtime",
	Long: `Disable a plugin by name at runtime.

This will stop the plugin and remove it from core system integrations.`,
	Args: cobra.ExactArgs(1),
	RunE: runEnhancedPluginsDisable,
}

var enhancedPluginsInfoCmd = &cobra.Command{
	Use:   "info [plugin-name]",
	Short: "Show detailed plugin information",
	Long: `Show detailed information about a specific plugin including
configuration, capabilities, health status, and integration points.`,
	Args: cobra.ExactArgs(1),
	RunE: runEnhancedPluginsInfo,
}

var enhancedPluginsHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health status of all plugins",
	Long: `Check the health status of all loaded plugins.

This provides detailed health information including metrics, resource usage,
and any error conditions.`,
	RunE: runEnhancedPluginsHealth,
}

var enhancedPluginsDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover available plugins",
	Long: `Discover available plugins from configured discovery paths.

This scans the configured plugin directories for available plugins
and updates the discovery cache.`,
	RunE: runEnhancedPluginsDiscover,
}

// Command flags
var (
	enhancedPluginsOutputFormat string
	enhancedPluginsShowDisabled bool
	enhancedPluginsVerbose      bool
)

func init() {
	// Add subcommands
	enhancedPluginsCmd.AddCommand(enhancedPluginsListCmd)
	enhancedPluginsCmd.AddCommand(enhancedPluginsEnableCmd)
	enhancedPluginsCmd.AddCommand(enhancedPluginsDisableCmd)
	enhancedPluginsCmd.AddCommand(enhancedPluginsInfoCmd)
	enhancedPluginsCmd.AddCommand(enhancedPluginsHealthCmd)
	enhancedPluginsCmd.AddCommand(enhancedPluginsDiscoverCmd)

	// List command flags
	enhancedPluginsListCmd.Flags().StringVar(&enhancedPluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")
	enhancedPluginsListCmd.Flags().BoolVar(&enhancedPluginsShowDisabled, "show-disabled", false, 
		"Show disabled plugins")
	enhancedPluginsListCmd.Flags().BoolVar(&enhancedPluginsVerbose, "verbose", false, 
		"Show verbose plugin information")

	// Info command flags
	enhancedPluginsInfoCmd.Flags().StringVar(&enhancedPluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")

	// Health command flags
	enhancedPluginsHealthCmd.Flags().StringVar(&enhancedPluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")

	// Discover command flags
	enhancedPluginsDiscoverCmd.Flags().StringVar(&enhancedPluginsOutputFormat, "format", "table", 
		"Output format: table, json, yaml")
}

func runEnhancedPluginsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Get plugin information
	discoveredPlugins := epm.GetPluginInfo()
	loadedPlugins := epm.GetLoadedPlugins()

	// Combine and format data
	var pluginData []EnhancedPluginListItem
	for name, info := range discoveredPlugins {
		item := EnhancedPluginListItem{
			Name:        name,
			Version:     info.Version,
			Description: info.Description,
			Source:      info.Source,
			Interfaces:  info.Interfaces,
			State:       string(epm.GetPluginState(name)),
		}

		// Add loaded plugin information if available
		if loaded, exists := loadedPlugins[name]; exists {
			item.LoadedAt = &loaded.LoadedAt
			item.Health = &loaded.Health
			item.Priority = &info.Priority
		}

		// Filter disabled plugins if requested
		if !enhancedPluginsShowDisabled && item.State == string(plugins.PluginStateDisabled) {
			continue
		}

		pluginData = append(pluginData, item)
	}

	// Sort by name
	sort.Slice(pluginData, func(i, j int) bool {
		return pluginData[i].Name < pluginData[j].Name
	})

	switch enhancedPluginsOutputFormat {
	case "json":
		return outputJSON(pluginData, "")
	case "yaml":
		return outputYAML(pluginData, "")
	default:
		return displayEnhancedPluginsTable(pluginData)
	}
}

func runEnhancedPluginsEnable(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pluginName := args[0]
	
	fmt.Printf("🔌 Enabling plugin: %s\n", pluginName)
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Enable the plugin
	if err := epm.EnablePlugin(ctx, pluginName); err != nil {
		return fmt.Errorf("failed to enable plugin %s: %w", pluginName, err)
	}
	
	fmt.Printf("✅ Plugin %s enabled successfully\n", pluginName)
	return nil
}

func runEnhancedPluginsDisable(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pluginName := args[0]
	
	fmt.Printf("🔌 Disabling plugin: %s\n", pluginName)
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Disable the plugin
	if err := epm.DisablePlugin(ctx, pluginName); err != nil {
		return fmt.Errorf("failed to disable plugin %s: %w", pluginName, err)
	}
	
	fmt.Printf("✅ Plugin %s disabled successfully\n", pluginName)
	return nil
}

func runEnhancedPluginsInfo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pluginName := args[0]
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Get plugin information
	discoveredPlugins := epm.GetPluginInfo()
	loadedPlugins := epm.GetLoadedPlugins()

	info, exists := discoveredPlugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Create detailed info structure
	detailedInfo := EnhancedPluginDetailedInfo{
		Info:  info,
		State: string(epm.GetPluginState(pluginName)),
	}

	if loaded, exists := loadedPlugins[pluginName]; exists {
		detailedInfo.LoadedPlugin = &loaded
	}

	switch enhancedPluginsOutputFormat {
	case "json":
		return outputJSON(detailedInfo, "")
	case "yaml":
		return outputYAML(detailedInfo, "")
	default:
		return displayEnhancedPluginDetailedInfo(detailedInfo)
	}
}

func runEnhancedPluginsHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Get loaded plugins with health info
	loadedPlugins := epm.GetLoadedPlugins()

	var healthData []EnhancedPluginHealthItem
	for name, loaded := range loadedPlugins {
		if loaded.State == plugins.PluginStateEnabled {
			healthData = append(healthData, EnhancedPluginHealthItem{
				Name:   name,
				Health: loaded.Health,
				State:  string(loaded.State),
			})
		}
	}

	// Sort by name
	sort.Slice(healthData, func(i, j int) bool {
		return healthData[i].Name < healthData[j].Name
	})

	switch enhancedPluginsOutputFormat {
	case "json":
		return outputJSON(healthData, "")
	case "yaml":
		return outputYAML(healthData, "")
	default:
		return displayEnhancedPluginsHealthTable(healthData)
	}
}

func runEnhancedPluginsDiscover(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	fmt.Println("🔍 Discovering plugins...")
	
	// Create enhanced plugin manager
	epm, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}
	defer epm.Shutdown(ctx)

	// Get discovery results
	discoveredPlugins := epm.GetPluginInfo()

	fmt.Printf("✅ Found %d plugins\n", len(discoveredPlugins))

	switch enhancedPluginsOutputFormat {
	case "json":
		return outputJSON(discoveredPlugins, "")
	case "yaml":
		return outputYAML(discoveredPlugins, "")
	default:
		return displayDiscoveredPluginsTable(discoveredPlugins)
	}
}

// createEnhancedPluginManager creates and initializes an enhanced plugin manager
func createEnhancedPluginManager(ctx context.Context) (*plugins.EnhancedPluginManager, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create logger (simplified for CLI usage)
	logger := &SimpleLogger{}

	// Create error handler
	errorHandler := errors.NewErrorHandler(logger, nil)

	// Create registry
	registry := registry.NewComponentRegistry()

	// Create enhanced plugin manager
	epm := plugins.NewEnhancedPluginManager(&cfg.Plugins, logger, errorHandler, registry)

	// Create integrations
	buildAdapter := plugins.NewBuildPipelineAdapter()
	serverAdapter := plugins.NewServerAdapter()
	watcherAdapter := plugins.NewWatcherAdapter()

	epm.SetIntegrations(buildAdapter, serverAdapter, watcherAdapter)

	// Register builtin plugins
	builtinPlugins := []plugins.Plugin{
		builtin.NewTailwindPlugin(),
		builtin.NewHotReloadPlugin(),
	}
	if err := epm.SetBuiltinPlugins(builtinPlugins); err != nil {
		return nil, fmt.Errorf("failed to register builtin plugins: %w", err)
	}

	// Initialize
	if err := epm.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin manager: %w", err)
	}

	return epm, nil
}

// Data structures for enhanced plugin information

type EnhancedPluginListItem struct {
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	Source      string                  `json:"source"`
	Interfaces  []string                `json:"interfaces"`
	State       string                  `json:"state"`
	LoadedAt    *time.Time              `json:"loaded_at,omitempty"`
	Health      *plugins.PluginHealth   `json:"health,omitempty"`
	Priority    *int                    `json:"priority,omitempty"`
}

type EnhancedPluginDetailedInfo struct {
	Info         plugins.EnhancedPluginInfo `json:"info"`
	State        string                     `json:"state"`
	LoadedPlugin *plugins.LoadedPlugin      `json:"loaded_plugin,omitempty"`
}

type EnhancedPluginHealthItem struct {
	Name   string                `json:"name"`
	Health plugins.PluginHealth  `json:"health"`
	State  string                `json:"state"`
}

// Display functions

func displayEnhancedPluginsTable(plugins []EnhancedPluginListItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	if enhancedPluginsVerbose {
		fmt.Fprintln(w, "NAME\tVERSION\tSOURCE\tSTATE\tINTERFACES\tHEALTH\tDESCRIPTION")
	} else {
		fmt.Fprintln(w, "NAME\tVERSION\tSOURCE\tSTATE\tDESCRIPTION")
	}

	// Plugins
	for _, plugin := range plugins {
		healthStatus := "unknown"
		if plugin.Health != nil {
			healthStatus = string(plugin.Health.Status)
		}

		if enhancedPluginsVerbose {
			interfaces := strings.Join(plugin.Interfaces, ",")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				plugin.Name, plugin.Version, plugin.Source, plugin.State,
				interfaces, healthStatus, plugin.Description)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				plugin.Name, plugin.Version, plugin.Source, plugin.State, plugin.Description)
		}
	}

	return nil
}

func displayEnhancedPluginDetailedInfo(info EnhancedPluginDetailedInfo) error {
	fmt.Printf("Plugin: %s\n", info.Info.Name)
	fmt.Printf("Version: %s\n", info.Info.Version)
	fmt.Printf("Description: %s\n", info.Info.Description)
	fmt.Printf("Source: %s\n", info.Info.Source)
	fmt.Printf("State: %s\n", info.State)
	fmt.Printf("Interfaces: %s\n", strings.Join(info.Info.Interfaces, ", "))

	if info.Info.Author != "" {
		fmt.Printf("Author: %s\n", info.Info.Author)
	}
	if info.Info.License != "" {
		fmt.Printf("License: %s\n", info.Info.License)
	}
	if info.Info.Path != "" {
		fmt.Printf("Path: %s\n", info.Info.Path)
	}

	if len(info.Info.Extensions) > 0 {
		fmt.Printf("Supported Extensions: %s\n", strings.Join(info.Info.Extensions, ", "))
	}

	if info.Info.Priority != 0 {
		fmt.Printf("Priority: %d\n", info.Info.Priority)
	}

	if info.LoadedPlugin != nil {
		fmt.Printf("Loaded At: %s\n", info.LoadedPlugin.LoadedAt.Format(time.RFC3339))
		fmt.Printf("Health Status: %s\n", info.LoadedPlugin.Health.Status)
		if info.LoadedPlugin.Health.Error != "" {
			fmt.Printf("Health Error: %s\n", info.LoadedPlugin.Health.Error)
		}
		if info.LoadedPlugin.Health.LastCheck.Unix() > 0 {
			fmt.Printf("Last Health Check: %s\n", info.LoadedPlugin.Health.LastCheck.Format(time.RFC3339))
		}
	}

	return nil
}

func displayEnhancedPluginsHealthTable(plugins []EnhancedPluginHealthItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tSTATE\tHEALTH\tLAST CHECK\tERROR")

	for _, plugin := range plugins {
		lastCheck := "never"
		if !plugin.Health.LastCheck.IsZero() {
			lastCheck = plugin.Health.LastCheck.Format("15:04:05")
		}

		errorMsg := plugin.Health.Error
		if errorMsg == "" {
			errorMsg = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			plugin.Name, plugin.State, plugin.Health.Status, lastCheck, errorMsg)
	}

	return nil
}

func displayDiscoveredPluginsTable(plugins map[string]plugins.EnhancedPluginInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tVERSION\tSOURCE\tINTERFACES\tDESCRIPTION")

	// Convert to slice and sort
	var pluginList []plugins.EnhancedPluginInfo
	for _, info := range plugins {
		pluginList = append(pluginList, info)
	}

	sort.Slice(pluginList, func(i, j int) bool {
		return pluginList[i].Name < pluginList[j].Name
	})

	for _, plugin := range pluginList {
		interfaces := strings.Join(plugin.Interfaces, ",")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			plugin.Name, plugin.Version, plugin.Source, interfaces, plugin.Description)
	}

	return nil
}

// SimpleLogger implements logging.Logger for CLI usage
type SimpleLogger struct{}

func (sl *SimpleLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
	}
}

func (sl *SimpleLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "WARN: %s\n", msg)
	}
}