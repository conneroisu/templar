package plugins

import (
	"context"
	"sync"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// RegistryIntegration provides plugin integration with the component registry
type RegistryIntegration struct {
	registry      *registry.ComponentRegistry
	pluginManager *EnhancedPluginManager
	mu            sync.RWMutex
}

// NewRegistryIntegration creates a new registry integration
func NewRegistryIntegration(registry *registry.ComponentRegistry, pluginManager *EnhancedPluginManager) *RegistryIntegration {
	return &RegistryIntegration{
		registry:      registry,
		pluginManager: pluginManager,
	}
}

// ProcessComponent processes a component through plugins before registering
func (ri *RegistryIntegration) ProcessComponent(ctx context.Context, component *types.ComponentInfo) error {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	// Process through plugins first
	processedComponent, err := ri.pluginManager.ProcessComponent(ctx, component)
	if err != nil {
		return err
	}

	// Update the original component with processed data
	*component = *processedComponent

	return nil
}

// BuildPipelineAdapter adapts plugins to the build pipeline
type BuildPipelineAdapter struct {
	preHooks  []BuildPlugin
	postHooks []BuildPlugin
	mu        sync.RWMutex
}

// NewBuildPipelineAdapter creates a new build pipeline adapter
func NewBuildPipelineAdapter() *BuildPipelineAdapter {
	return &BuildPipelineAdapter{
		preHooks:  make([]BuildPlugin, 0),
		postHooks: make([]BuildPlugin, 0),
	}
}

// RegisterPreBuildHook registers a pre-build hook
func (bpa *BuildPipelineAdapter) RegisterPreBuildHook(plugin BuildPlugin) error {
	bpa.mu.Lock()
	defer bpa.mu.Unlock()

	bpa.preHooks = append(bpa.preHooks, plugin)
	return nil
}

// RegisterPostBuildHook registers a post-build hook
func (bpa *BuildPipelineAdapter) RegisterPostBuildHook(plugin BuildPlugin) error {
	bpa.mu.Lock()
	defer bpa.mu.Unlock()

	bpa.postHooks = append(bpa.postHooks, plugin)
	return nil
}

// RemovePlugin removes a plugin from both pre and post hooks
func (bpa *BuildPipelineAdapter) RemovePlugin(pluginName string) error {
	bpa.mu.Lock()
	defer bpa.mu.Unlock()

	// Remove from pre-hooks
	filtered := make([]BuildPlugin, 0)
	for _, plugin := range bpa.preHooks {
		if plugin.Name() != pluginName {
			filtered = append(filtered, plugin)
		}
	}
	bpa.preHooks = filtered

	// Remove from post-hooks
	filtered = make([]BuildPlugin, 0)
	for _, plugin := range bpa.postHooks {
		if plugin.Name() != pluginName {
			filtered = append(filtered, plugin)
		}
	}
	bpa.postHooks = filtered

	return nil
}

// ExecutePreBuildHooks executes all registered pre-build hooks
func (bpa *BuildPipelineAdapter) ExecutePreBuildHooks(ctx context.Context, components []*types.ComponentInfo) error {
	bpa.mu.RLock()
	defer bpa.mu.RUnlock()

	for _, plugin := range bpa.preHooks {
		if err := plugin.PreBuild(ctx, components); err != nil {
			return err
		}
	}

	return nil
}

// ExecutePostBuildHooks executes all registered post-build hooks
func (bpa *BuildPipelineAdapter) ExecutePostBuildHooks(ctx context.Context, components []*types.ComponentInfo, result BuildResult) error {
	bpa.mu.RLock()
	defer bpa.mu.RUnlock()

	for _, plugin := range bpa.postHooks {
		if err := plugin.PostBuild(ctx, components, result); err != nil {
			return err
		}
	}

	return nil
}

// ServerAdapter adapts plugins to the HTTP server
type ServerAdapter struct {
	plugins map[string]ServerPlugin
	mu      sync.RWMutex
}

// NewServerAdapter creates a new server adapter
func NewServerAdapter() *ServerAdapter {
	return &ServerAdapter{
		plugins: make(map[string]ServerPlugin),
	}
}

// RegisterPlugin registers a server plugin
func (sa *ServerAdapter) RegisterPlugin(plugin ServerPlugin) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	sa.plugins[plugin.Name()] = plugin
	return nil
}

// RemovePlugin removes a server plugin
func (sa *ServerAdapter) RemovePlugin(pluginName string) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	delete(sa.plugins, pluginName)
	return nil
}

// GetPlugins returns all registered server plugins
func (sa *ServerAdapter) GetPlugins() map[string]ServerPlugin {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	result := make(map[string]ServerPlugin)
	for name, plugin := range sa.plugins {
		result[name] = plugin
	}

	return result
}

// WatcherAdapter adapts plugins to the file watcher
type WatcherAdapter struct {
	plugins map[string]WatcherPlugin
	mu      sync.RWMutex
}

// NewWatcherAdapter creates a new watcher adapter
func NewWatcherAdapter() *WatcherAdapter {
	return &WatcherAdapter{
		plugins: make(map[string]WatcherPlugin),
	}
}

// RegisterPlugin registers a watcher plugin
func (wa *WatcherAdapter) RegisterPlugin(plugin WatcherPlugin) error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	wa.plugins[plugin.Name()] = plugin
	return nil
}

// RemovePlugin removes a watcher plugin
func (wa *WatcherAdapter) RemovePlugin(pluginName string) error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	delete(wa.plugins, pluginName)
	return nil
}

// GetWatchPatterns returns all watch patterns from registered plugins
func (wa *WatcherAdapter) GetWatchPatterns() []string {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	var patterns []string
	for _, plugin := range wa.plugins {
		patterns = append(patterns, plugin.WatchPatterns()...)
	}

	return patterns
}

// HandleFileChange notifies all watcher plugins of a file change
func (wa *WatcherAdapter) HandleFileChange(ctx context.Context, event FileChangeEvent) error {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	for _, plugin := range wa.plugins {
		if err := plugin.HandleFileChange(ctx, event); err != nil {
			// Log error but continue with other plugins
			// TODO: Add proper error logging
		}
	}

	return nil
}
