package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// EnhancedPluginManager provides comprehensive plugin management with configuration integration
type EnhancedPluginManager struct {
	*PluginManager // Embed the existing manager

	// Configuration integration
	config       *config.PluginsConfig
	logger       logging.Logger
	errorHandler *errors.ErrorHandler

	// Plugin state management
	enabledPlugins    map[string]bool
	pluginStates      map[string]PluginState
	discoveredPlugins map[string]EnhancedPluginInfo

	// Core system integration
	registry      *registry.ComponentRegistry
	buildPipeline BuildPipelineIntegration
	server        ServerIntegration
	watcher       WatcherIntegration

	// Discovery and loading
	discoveryPaths []string
	loadedPlugins  map[string]LoadedPlugin

	mu sync.RWMutex
}

// PluginState represents the current state of a plugin
type PluginState string

const (
	PluginStateUnknown     PluginState = "unknown"
	PluginStateDiscovered  PluginState = "discovered"
	PluginStateLoaded      PluginState = "loaded"
	PluginStateInitialized PluginState = "initialized"
	PluginStateEnabled     PluginState = "enabled"
	PluginStateDisabled    PluginState = "disabled"
	PluginStateError       PluginState = "error"
)

// EnhancedPluginInfo contains metadata about a discovered plugin
type EnhancedPluginInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	License     string                 `json:"license,omitempty"`
	Source      string                 `json:"source"` // "builtin", "file", "url"
	Path        string                 `json:"path,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`

	// Plugin capabilities
	Interfaces []string `json:"interfaces"`
	Extensions []string `json:"extensions,omitempty"`
	Priority   int      `json:"priority,omitempty"`
}

// LoadedPlugin represents a plugin that has been loaded into memory
type LoadedPlugin struct {
	Info     EnhancedPluginInfo
	Instance Plugin
	State    PluginState
	Health   PluginHealth
	Config   PluginConfig
	LoadedAt time.Time
}

// Core system integration interfaces
type BuildPipelineIntegration interface {
	RegisterPreBuildHook(plugin BuildPlugin) error
	RegisterPostBuildHook(plugin BuildPlugin) error
	RemovePlugin(pluginName string) error
}

type ServerIntegration interface {
	RegisterPlugin(plugin ServerPlugin) error
	RemovePlugin(pluginName string) error
}

type WatcherIntegration interface {
	RegisterPlugin(plugin WatcherPlugin) error
	RemovePlugin(pluginName string) error
}

// NewEnhancedPluginManager creates a new enhanced plugin manager
func NewEnhancedPluginManager(
	config *config.PluginsConfig,
	logger logging.Logger,
	errorHandler *errors.ErrorHandler,
	registry *registry.ComponentRegistry,
) *EnhancedPluginManager {
	baseManager := NewPluginManager()

	manager := &EnhancedPluginManager{
		PluginManager:     baseManager,
		config:            config,
		logger:            logger,
		errorHandler:      errorHandler,
		registry:          registry,
		enabledPlugins:    make(map[string]bool),
		pluginStates:      make(map[string]PluginState),
		discoveredPlugins: make(map[string]EnhancedPluginInfo),
		loadedPlugins:     make(map[string]LoadedPlugin),
		discoveryPaths:    config.DiscoveryPaths,
	}

	// Initialize enabled/disabled state from config
	for _, name := range config.Enabled {
		manager.enabledPlugins[name] = true
	}
	for _, name := range config.Disabled {
		manager.enabledPlugins[name] = false
	}

	return manager
}

// SetIntegrations sets the core system integrations
func (epm *EnhancedPluginManager) SetIntegrations(
	buildPipeline BuildPipelineIntegration,
	server ServerIntegration,
	watcher WatcherIntegration,
) {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	epm.buildPipeline = buildPipeline
	epm.server = server
	epm.watcher = watcher
}

// Initialize initializes the enhanced plugin manager and discovers plugins
func (epm *EnhancedPluginManager) Initialize(ctx context.Context) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	// Register built-in plugins first
	if err := epm.registerBuiltinPlugins(ctx); err != nil {
		return fmt.Errorf("failed to register builtin plugins: %w", err)
	}

	// Discover external plugins
	if err := epm.discoverPlugins(ctx); err != nil {
		epm.logger.Error(ctx, err, "Failed to discover external plugins")
		// Don't fail initialization if discovery fails
	}

	// Load and initialize enabled plugins
	if err := epm.loadEnabledPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load enabled plugins: %w", err)
	}

	return nil
}

// registerBuiltinPlugins registers the built-in plugins
func (epm *EnhancedPluginManager) registerBuiltinPlugins(ctx context.Context) error {
	// Built-in plugins will be registered via the SetBuiltinPlugins method
	// to avoid import cycles
	return nil
}

// SetBuiltinPlugins allows external registration of builtin plugins
func (epm *EnhancedPluginManager) SetBuiltinPlugins(plugins []Plugin) error {
	ctx := context.Background()

	for _, plugin := range plugins {
		if err := epm.registerPlugin(ctx, plugin, "builtin"); err != nil {
			return fmt.Errorf("failed to register builtin plugin %s: %w", plugin.Name(), err)
		}
	}

	return nil
}

// registerPlugin registers a plugin instance
func (epm *EnhancedPluginManager) registerPlugin(
	ctx context.Context,
	plugin Plugin,
	source string,
) error {
	name := plugin.Name()

	// Create plugin info
	info := EnhancedPluginInfo{
		Name:        name,
		Version:     plugin.Version(),
		Description: plugin.Description(),
		Source:      source,
		Interfaces:  epm.getPluginInterfaces(plugin),
	}

	// Add plugin-specific info
	if cp, ok := plugin.(ComponentPlugin); ok {
		info.Extensions = cp.SupportedExtensions()
		info.Priority = cp.Priority()
	}

	// Store discovered plugin info
	epm.discoveredPlugins[name] = info
	epm.pluginStates[name] = PluginStateDiscovered

	// Check if plugin should be enabled
	enabled, exists := epm.enabledPlugins[name]
	if !exists {
		// Default to enabled for builtin plugins if not explicitly configured
		enabled = source == "builtin"
	}

	if enabled {
		return epm.loadPlugin(ctx, plugin, info)
	}

	return nil
}

// loadPlugin loads and initializes a plugin
func (epm *EnhancedPluginManager) loadPlugin(
	ctx context.Context,
	plugin Plugin,
	info EnhancedPluginInfo,
) error {
	name := plugin.Name()

	// Get plugin configuration
	pluginConfig := epm.getPluginConfig(name)

	// Initialize the plugin
	if err := plugin.Initialize(ctx, pluginConfig); err != nil {
		epm.pluginStates[name] = PluginStateError
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Register with base plugin manager
	if err := epm.RegisterPlugin(plugin, pluginConfig); err != nil {
		epm.pluginStates[name] = PluginStateError
		return fmt.Errorf("failed to register plugin %s: %w", name, err)
	}

	// Integrate with core systems
	if err := epm.integratePlugin(plugin); err != nil {
		epm.pluginStates[name] = PluginStateError
		epm.logger.Error(ctx, err, "Failed to integrate plugin with core systems", "plugin", name)
		// Continue anyway - plugin is still functional
	}

	// Store loaded plugin
	loadedPlugin := LoadedPlugin{
		Info:     info,
		Instance: plugin,
		State:    PluginStateEnabled,
		Health:   plugin.Health(),
		Config:   pluginConfig,
		LoadedAt: time.Now(),
	}
	epm.loadedPlugins[name] = loadedPlugin
	epm.pluginStates[name] = PluginStateEnabled

	epm.logger.Info(ctx, "Plugin loaded successfully", "plugin", name, "version", plugin.Version())

	return nil
}

// integratePlugin integrates a plugin with core systems
func (epm *EnhancedPluginManager) integratePlugin(plugin Plugin) error {
	name := plugin.Name()

	// Integrate with build pipeline
	if bp, ok := plugin.(BuildPlugin); ok && epm.buildPipeline != nil {
		if err := epm.buildPipeline.RegisterPreBuildHook(bp); err != nil {
			return fmt.Errorf("failed to register build pre-hook for %s: %w", name, err)
		}
		if err := epm.buildPipeline.RegisterPostBuildHook(bp); err != nil {
			return fmt.Errorf("failed to register build post-hook for %s: %w", name, err)
		}
	}

	// Integrate with server
	if sp, ok := plugin.(ServerPlugin); ok && epm.server != nil {
		if err := epm.server.RegisterPlugin(sp); err != nil {
			return fmt.Errorf("failed to register server plugin %s: %w", name, err)
		}
	}

	// Integrate with file watcher
	if wp, ok := plugin.(WatcherPlugin); ok && epm.watcher != nil {
		if err := epm.watcher.RegisterPlugin(wp); err != nil {
			return fmt.Errorf("failed to register watcher plugin %s: %w", name, err)
		}
	}

	return nil
}

// getPluginConfig gets configuration for a specific plugin
func (epm *EnhancedPluginManager) getPluginConfig(pluginName string) PluginConfig {
	// Start with default config
	config := PluginConfig{
		Name:    pluginName,
		Enabled: true,
		Config:  make(map[string]interface{}),
		Settings: PluginSettings{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			LogLevel:   "info",
			ResourceLimits: ResourceLimits{
				MaxMemoryMB:        100,
				MaxCPUPercent:      10.0,
				MaxGoroutines:      10,
				MaxFileDescriptors: 100,
			},
		},
	}

	// Apply plugin-specific configuration from config file
	if pluginConfigs, exists := epm.config.Configurations[pluginName]; exists {
		config.Config = pluginConfigs
	}

	return config
}

// getPluginInterfaces determines which interfaces a plugin implements
func (epm *EnhancedPluginManager) getPluginInterfaces(plugin Plugin) []string {
	var interfaces []string

	interfaces = append(interfaces, "Plugin")

	if _, ok := plugin.(ComponentPlugin); ok {
		interfaces = append(interfaces, "ComponentPlugin")
	}
	if _, ok := plugin.(BuildPlugin); ok {
		interfaces = append(interfaces, "BuildPlugin")
	}
	if _, ok := plugin.(ServerPlugin); ok {
		interfaces = append(interfaces, "ServerPlugin")
	}
	if _, ok := plugin.(WatcherPlugin); ok {
		interfaces = append(interfaces, "WatcherPlugin")
	}

	return interfaces
}

// discoverPlugins discovers plugins from configured discovery paths
func (epm *EnhancedPluginManager) discoverPlugins(ctx context.Context) error {
	for _, path := range epm.discoveryPaths {
		if err := epm.discoverPluginsInPath(ctx, path); err != nil {
			epm.logger.Error(ctx, err, "Failed to discover plugins in path", "path", path)
			// Continue with other paths
		}
	}
	return nil
}

// discoverPluginsInPath discovers plugins in a specific path
func (epm *EnhancedPluginManager) discoverPluginsInPath(ctx context.Context, path string) error {
	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Path doesn't exist, skip
	}

	// Walk the directory looking for plugin files
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for .so files (Go plugins) or plugin manifest files
		if strings.HasSuffix(filePath, ".so") || strings.HasSuffix(filePath, "plugin.json") {
			// TODO: Implement external plugin loading
			epm.logger.Info(ctx, "Found potential plugin file", "path", filePath)
		}

		return nil
	})
}

// loadEnabledPlugins loads all plugins that should be enabled
func (epm *EnhancedPluginManager) loadEnabledPlugins(ctx context.Context) error {
	var errors []error

	for name, enabled := range epm.enabledPlugins {
		if !enabled {
			continue
		}

		// Check if plugin is already loaded
		if _, exists := epm.loadedPlugins[name]; exists {
			continue
		}

		// Plugin is enabled but not loaded - this might be an external plugin
		epm.logger.Warn(ctx, nil, "Plugin enabled but not found", "plugin", name)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some enabled plugins: %v", errors)
	}

	return nil
}

// EnablePlugin enables a plugin at runtime
func (epm *EnhancedPluginManager) EnablePlugin(ctx context.Context, name string) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	// Check if plugin is discovered
	info, exists := epm.discoveredPlugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Check if already enabled
	if loaded, exists := epm.loadedPlugins[name]; exists && loaded.State == PluginStateEnabled {
		return nil // Already enabled
	}

	// Enable the plugin
	epm.enabledPlugins[name] = true

	// If it's a builtin plugin that's not loaded, we need to load it
	if info.Source == "builtin" {
		// TODO: Reload builtin plugin
		return fmt.Errorf("runtime enabling of builtin plugins not yet implemented")
	}

	// TODO: Load external plugin
	return fmt.Errorf("runtime enabling of external plugins not yet implemented")
}

// DisablePlugin disables a plugin at runtime
func (epm *EnhancedPluginManager) DisablePlugin(ctx context.Context, name string) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	// Check if plugin is loaded
	loaded, exists := epm.loadedPlugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	// Disable the plugin
	epm.enabledPlugins[name] = false

	// Remove from core system integrations
	if epm.buildPipeline != nil {
		epm.buildPipeline.RemovePlugin(name)
	}
	if epm.server != nil {
		epm.server.RemovePlugin(name)
	}
	if epm.watcher != nil {
		epm.watcher.RemovePlugin(name)
	}

	// Shutdown the plugin
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := loaded.Instance.Shutdown(shutdownCtx); err != nil {
		epm.logger.Error(ctx, err, "Failed to shutdown plugin gracefully", "plugin", name)
	}

	// Unregister from base manager
	epm.UnregisterPlugin(name)

	// Update state
	loaded.State = PluginStateDisabled
	epm.loadedPlugins[name] = loaded
	epm.pluginStates[name] = PluginStateDisabled

	epm.logger.Info(ctx, "Plugin disabled successfully", "plugin", name)

	return nil
}

// GetPluginInfo returns information about all discovered plugins
func (epm *EnhancedPluginManager) GetPluginInfo() map[string]EnhancedPluginInfo {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	result := make(map[string]EnhancedPluginInfo)
	for name, info := range epm.discoveredPlugins {
		result[name] = info
	}

	return result
}

// GetLoadedPlugins returns information about all loaded plugins
func (epm *EnhancedPluginManager) GetLoadedPlugins() map[string]LoadedPlugin {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	result := make(map[string]LoadedPlugin)
	for name, plugin := range epm.loadedPlugins {
		result[name] = plugin
	}

	return result
}

// GetPluginState returns the current state of a plugin
func (epm *EnhancedPluginManager) GetPluginState(name string) PluginState {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	if state, exists := epm.pluginStates[name]; exists {
		return state
	}

	return PluginStateUnknown
}

// ProcessComponent processes a component through all enabled component plugins
func (epm *EnhancedPluginManager) ProcessComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	// Get component plugins in priority order
	plugins := epm.getComponentPluginsByPriority()

	result := component
	for _, plugin := range plugins {
		var err error
		result, err = plugin.HandleComponent(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("plugin %s failed to process component %s: %w",
				plugin.Name(), component.Name, err)
		}
	}

	return result, nil
}

// getComponentPluginsByPriority returns component plugins sorted by priority
func (epm *EnhancedPluginManager) getComponentPluginsByPriority() []ComponentPlugin {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	var plugins []ComponentPlugin
	for _, loaded := range epm.loadedPlugins {
		if loaded.State == PluginStateEnabled {
			if cp, ok := loaded.Instance.(ComponentPlugin); ok {
				plugins = append(plugins, cp)
			}
		}
	}

	// Sort by priority (lower numbers first)
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Priority() < plugins[j].Priority()
	})

	return plugins
}

// Shutdown gracefully shuts down all plugins
func (epm *EnhancedPluginManager) Shutdown(ctx context.Context) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	var errors []error

	// Shutdown all loaded plugins
	for name, loaded := range epm.loadedPlugins {
		if loaded.State == PluginStateEnabled {
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := loaded.Instance.Shutdown(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to shutdown plugin %s: %w", name, err))
			}
			cancel()
		}
	}

	// Shutdown base manager
	if err := epm.PluginManager.Shutdown(); err != nil {
		errors = append(errors, fmt.Errorf("failed to shutdown base plugin manager: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
