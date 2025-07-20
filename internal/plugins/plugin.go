package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// Plugin represents a Templar plugin interface
type Plugin interface {
	// Name returns the unique name of the plugin
	Name() string
	
	// Version returns the version of the plugin
	Version() string
	
	// Description returns a description of what the plugin does
	Description() string
	
	// Initialize initializes the plugin with the given context and configuration
	Initialize(ctx context.Context, config PluginConfig) error
	
	// Shutdown gracefully shuts down the plugin
	Shutdown(ctx context.Context) error
	
	// Health returns the health status of the plugin
	Health() PluginHealth
}

// ComponentPlugin extends Plugin with component-specific functionality
type ComponentPlugin interface {
	Plugin
	
	// HandleComponent processes a component and returns modified component info
	HandleComponent(ctx context.Context, component *registry.ComponentInfo) (*registry.ComponentInfo, error)
	
	// SupportedExtensions returns file extensions this plugin can handle
	SupportedExtensions() []string
	
	// Priority returns the execution priority (lower numbers execute first)
	Priority() int
}

// BuildPlugin extends Plugin with build-specific functionality
type BuildPlugin interface {
	Plugin
	
	// PreBuild is called before the build process starts
	PreBuild(ctx context.Context, components []*registry.ComponentInfo) error
	
	// PostBuild is called after the build process completes
	PostBuild(ctx context.Context, components []*registry.ComponentInfo, buildResult BuildResult) error
	
	// TransformBuildCommand allows modifying the build command
	TransformBuildCommand(ctx context.Context, command []string) ([]string, error)
}

// ServerPlugin extends Plugin with server-specific functionality
type ServerPlugin interface {
	Plugin
	
	// RegisterRoutes registers additional HTTP routes
	RegisterRoutes(router Router) error
	
	// Middleware returns HTTP middleware functions
	Middleware() []MiddlewareFunc
	
	// WebSocketHandler handles WebSocket connections
	WebSocketHandler(ctx context.Context, conn WebSocketConnection) error
}

// WatcherPlugin extends Plugin with file watching functionality
type WatcherPlugin interface {
	Plugin
	
	// WatchPatterns returns additional file patterns to watch
	WatchPatterns() []string
	
	// HandleFileChange is called when a watched file changes
	HandleFileChange(ctx context.Context, event FileChangeEvent) error
	
	// ShouldIgnore determines if a file change should be ignored
	ShouldIgnore(filePath string) bool
}

// PluginConfig contains configuration for a plugin
type PluginConfig struct {
	// Name of the plugin
	Name string `json:"name"`
	
	// Configuration data specific to the plugin
	Config map[string]interface{} `json:"config"`
	
	// Whether the plugin is enabled
	Enabled bool `json:"enabled"`
	
	// Plugin-specific settings
	Settings PluginSettings `json:"settings"`
}

// PluginSettings contains plugin-specific settings
type PluginSettings struct {
	// Timeout for plugin operations
	Timeout time.Duration `json:"timeout"`
	
	// Maximum retries for failed operations
	MaxRetries int `json:"max_retries"`
	
	// Log level for the plugin
	LogLevel string `json:"log_level"`
	
	// Resource limits
	ResourceLimits ResourceLimits `json:"resource_limits"`
}

// ResourceLimits defines resource constraints for plugins
type ResourceLimits struct {
	// Maximum memory usage in MB
	MaxMemoryMB int `json:"max_memory_mb"`
	
	// Maximum CPU usage percentage
	MaxCPUPercent float64 `json:"max_cpu_percent"`
	
	// Maximum number of goroutines
	MaxGoroutines int `json:"max_goroutines"`
	
	// Maximum file descriptors
	MaxFileDescriptors int `json:"max_file_descriptors"`
}

// PluginHealth represents the health status of a plugin
type PluginHealth struct {
	// Status of the plugin
	Status HealthStatus `json:"status"`
	
	// Last check timestamp
	LastCheck time.Time `json:"last_check"`
	
	// Error message if unhealthy
	Error string `json:"error,omitempty"`
	
	// Additional health metrics
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// HealthStatus represents the health status values
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// BuildResult contains the result of a build operation
type BuildResult struct {
	// Success indicates if the build was successful
	Success bool `json:"success"`
	
	// Duration of the build
	Duration time.Duration `json:"duration"`
	
	// Number of components built
	ComponentsBuilt int `json:"components_built"`
	
	// Build output
	Output string `json:"output"`
	
	// Error message if build failed
	Error string `json:"error,omitempty"`
}

// Router interface for registering HTTP routes
type Router interface {
	GET(path string, handler HandlerFunc)
	POST(path string, handler HandlerFunc)
	PUT(path string, handler HandlerFunc)
	DELETE(path string, handler HandlerFunc)
	Static(path, root string)
}

// HandlerFunc represents an HTTP handler function
type HandlerFunc func(ctx Context) error

// Context represents an HTTP request context
type Context interface {
	// Request methods
	Method() string
	Path() string
	Param(key string) string
	Query(key string) string
	Header(key string) string
	Body() ([]byte, error)
	
	// Response methods
	JSON(code int, data interface{}) error
	String(code int, data string) error
	File(filePath string) error
	Redirect(code int, url string) error
	
	// Additional context
	Set(key string, value interface{})
	Get(key string) interface{}
}

// MiddlewareFunc represents HTTP middleware
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection interface {
	// Send sends data to the WebSocket
	Send(data []byte) error
	
	// Receive receives data from the WebSocket
	Receive() ([]byte, error)
	
	// Close closes the WebSocket connection
	Close() error
	
	// RemoteAddr returns the remote address
	RemoteAddr() string
}

// FileChangeEvent represents a file system change event
type FileChangeEvent struct {
	// Path of the changed file
	Path string `json:"path"`
	
	// Type of change
	Type FileChangeType `json:"type"`
	
	// Timestamp of the change
	Timestamp time.Time `json:"timestamp"`
	
	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FileChangeType represents the type of file change
type FileChangeType string

const (
	FileChangeTypeCreate FileChangeType = "create"
	FileChangeTypeModify FileChangeType = "modify"
	FileChangeTypeDelete FileChangeType = "delete"
	FileChangeTypeRename FileChangeType = "rename"
)

// PluginManager manages the lifecycle of plugins
type PluginManager struct {
	plugins           map[string]Plugin
	componentPlugins  []ComponentPlugin
	buildPlugins      []BuildPlugin
	serverPlugins     []ServerPlugin
	watcherPlugins    []WatcherPlugin
	configs           map[string]PluginConfig
	healthChecks      map[string]PluginHealth
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	healthCheckTicker *time.Ticker
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PluginManager{
		plugins:       make(map[string]Plugin),
		configs:       make(map[string]PluginConfig),
		healthChecks:  make(map[string]PluginHealth),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// RegisterPlugin registers a new plugin
func (pm *PluginManager) RegisterPlugin(plugin Plugin, config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	// Initialize the plugin
	if err := plugin.Initialize(pm.ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}
	
	// Store the plugin
	pm.plugins[name] = plugin
	pm.configs[name] = config
	
	// Categorize the plugin
	if cp, ok := plugin.(ComponentPlugin); ok {
		pm.componentPlugins = append(pm.componentPlugins, cp)
	}
	if bp, ok := plugin.(BuildPlugin); ok {
		pm.buildPlugins = append(pm.buildPlugins, bp)
	}
	if sp, ok := plugin.(ServerPlugin); ok {
		pm.serverPlugins = append(pm.serverPlugins, sp)
	}
	if wp, ok := plugin.(WatcherPlugin); ok {
		pm.watcherPlugins = append(pm.watcherPlugins, wp)
	}
	
	return nil
}

// UnregisterPlugin unregisters a plugin
func (pm *PluginManager) UnregisterPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	// Shutdown the plugin
	shutdownCtx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()
	
	if err := plugin.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}
	
	// Remove from collections
	delete(pm.plugins, name)
	delete(pm.configs, name)
	delete(pm.healthChecks, name)
	
	// Remove from categorized lists
	pm.componentPlugins = removeComponentPlugin(pm.componentPlugins, plugin)
	pm.buildPlugins = removeBuildPlugin(pm.buildPlugins, plugin)
	pm.serverPlugins = removeServerPlugin(pm.serverPlugins, plugin)
	pm.watcherPlugins = removeWatcherPlugin(pm.watcherPlugins, plugin)
	
	return nil
}

// GetPlugin retrieves a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin, nil
}

// ListPlugins returns all registered plugins
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	var plugins []PluginInfo
	for name, plugin := range pm.plugins {
		config := pm.configs[name]
		health := pm.healthChecks[name]
		
		plugins = append(plugins, PluginInfo{
			Name:        name,
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Enabled:     config.Enabled,
			Health:      health,
		})
	}
	
	return plugins
}

// PluginInfo contains information about a plugin
type PluginInfo struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Enabled     bool         `json:"enabled"`
	Health      PluginHealth `json:"health"`
}

// ProcessComponent processes a component through all component plugins
func (pm *PluginManager) ProcessComponent(ctx context.Context, component *registry.ComponentInfo) (*registry.ComponentInfo, error) {
	pm.mu.RLock()
	plugins := make([]ComponentPlugin, len(pm.componentPlugins))
	copy(plugins, pm.componentPlugins)
	pm.mu.RUnlock()
	
	// Sort plugins by priority
	for i := 0; i < len(plugins)-1; i++ {
		for j := 0; j < len(plugins)-i-1; j++ {
			if plugins[j].Priority() > plugins[j+1].Priority() {
				plugins[j], plugins[j+1] = plugins[j+1], plugins[j]
			}
		}
	}
	
	result := component
	for _, plugin := range plugins {
		var err error
		result, err = plugin.HandleComponent(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("plugin %s failed to process component: %w", plugin.Name(), err)
		}
	}
	
	return result, nil
}

// StartHealthChecks starts periodic health checks for all plugins
func (pm *PluginManager) StartHealthChecks(interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.healthCheckTicker != nil {
		pm.healthCheckTicker.Stop()
	}
	
	pm.healthCheckTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-pm.healthCheckTicker.C:
				pm.checkAllPluginHealth()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// checkAllPluginHealth performs health checks on all plugins
func (pm *PluginManager) checkAllPluginHealth() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for name, plugin := range pm.plugins {
		health := plugin.Health()
		health.LastCheck = time.Now()
		pm.healthChecks[name] = health
	}
}

// Shutdown gracefully shuts down all plugins
func (pm *PluginManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.healthCheckTicker != nil {
		pm.healthCheckTicker.Stop()
	}
	
	pm.cancel()
	
	var errors []error
	for name, plugin := range pm.plugins {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := plugin.Shutdown(shutdownCtx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown plugin %s: %w", name, err))
		}
		cancel()
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}
	
	return nil
}

// Helper functions for removing plugins from slices
func removeComponentPlugin(plugins []ComponentPlugin, target Plugin) []ComponentPlugin {
	for i, plugin := range plugins {
		if plugin.Name() == target.Name() {
			return append(plugins[:i], plugins[i+1:]...)
		}
	}
	return plugins
}

func removeBuildPlugin(plugins []BuildPlugin, target Plugin) []BuildPlugin {
	for i, plugin := range plugins {
		if plugin.Name() == target.Name() {
			return append(plugins[:i], plugins[i+1:]...)
		}
	}
	return plugins
}

func removeServerPlugin(plugins []ServerPlugin, target Plugin) []ServerPlugin {
	for i, plugin := range plugins {
		if plugin.Name() == target.Name() {
			return append(plugins[:i], plugins[i+1:]...)
		}
	}
	return plugins
}

func removeWatcherPlugin(plugins []WatcherPlugin, target Plugin) []WatcherPlugin {
	for i, plugin := range plugins {
		if plugin.Name() == target.Name() {
			return append(plugins[:i], plugins[i+1:]...)
		}
	}
	return plugins
}