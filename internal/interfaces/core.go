// Package interfaces provides core abstractions for the Templar CLI application.
// This package defines interfaces to reduce coupling between packages and improve
// testability by enabling dependency injection and mocking.
package interfaces

import (
	"context"

	"github.com/conneroisu/templar/internal/types"
)

// Forward declarations for concrete types from other packages
// to avoid circular dependencies

// FileFilter defines the interface for filtering files
type FileFilter interface {
	ShouldInclude(path string) bool
}

// FileFilterFunc is the concrete file filter function type that implements FileFilter
type FileFilterFunc func(path string) bool

// ShouldInclude implements the FileFilter interface
func (f FileFilterFunc) ShouldInclude(path string) bool {
	return f(path)
}

// ChangeHandlerFunc is the concrete change handler function type
type ChangeHandlerFunc func(events []interface{}) error

// BuildCallbackFunc is the concrete build callback function type
type BuildCallbackFunc func(result interface{})

// ComponentRegistry defines the interface for managing component information
type ComponentRegistry interface {
	// Register adds or updates a component in the registry
	Register(component *types.ComponentInfo)

	// Get retrieves a component by name
	Get(name string) (*types.ComponentInfo, bool)

	// GetAll returns all registered components
	GetAll() []*types.ComponentInfo

	// Watch returns a channel for component change events
	Watch() <-chan types.ComponentEvent

	// UnWatch removes a watcher and closes its channel
	UnWatch(ch <-chan types.ComponentEvent)

	// Count returns the number of registered components
	Count() int

	// DetectCircularDependencies returns any circular dependency chains
	DetectCircularDependencies() [][]string
}

// ComponentScanner defines the interface for discovering and parsing components
type ComponentScanner interface {
	// ScanDirectory scans a directory for templ components
	ScanDirectory(dir string) error

	// ScanDirectoryParallel scans with configurable worker count
	ScanDirectoryParallel(dir string, workers int) error

	// ScanFile scans a single file for components
	ScanFile(path string) error

	// GetRegistry returns the associated component registry
	GetRegistry() ComponentRegistry
}

// BuildPipeline defines the interface for building components
type BuildPipeline interface {
	// Build processes a single component
	Build(component *types.ComponentInfo) error

	// Start begins the build pipeline with the given context
	Start(ctx context.Context) error

	// Stop gracefully shuts down the build pipeline
	Stop() error

	// AddCallback registers a callback for build completion events
	AddCallback(callback BuildCallbackFunc)

	// BuildWithPriority builds a component with priority
	BuildWithPriority(component *types.ComponentInfo)

	// GetMetrics returns build metrics
	GetMetrics() interface{}

	// GetCache returns cache statistics
	GetCache() interface{}

	// ClearCache clears the build cache
	ClearCache()
}

// FileWatcher defines the interface for monitoring file system changes
type FileWatcher interface {
	// AddPath adds a path to watch
	AddPath(path string) error

	// Start begins watching with the given context
	Start(ctx context.Context) error

	// Stop stops watching and cleans up resources
	Stop() error

	// AddFilter adds a file filter function
	AddFilter(filter FileFilter)

	// AddHandler adds a change handler function
	AddHandler(handler ChangeHandlerFunc)

	// AddRecursive adds a recursive path to watch
	AddRecursive(root string) error
}

// PreviewServer defines the interface for the component preview server
type PreviewServer interface {
	// Start starts the preview server
	Start(ctx context.Context) error

	// Stop stops the preview server
	Stop() error

	// GetURL returns the server URL
	GetURL() string

	// SetRegistry sets the component registry
	SetRegistry(registry ComponentRegistry)
}

// TemplCompiler defines the interface for compiling templ components
type TemplCompiler interface {
	// Compile compiles a component to output
	Compile(component *types.ComponentInfo) ([]byte, error)

	// CompileWithContext compiles with a context for cancellation
	CompileWithContext(ctx context.Context, component *types.ComponentInfo) ([]byte, error)

	// Validate validates component syntax without compilation
	Validate(component *types.ComponentInfo) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Load loads configuration from files and environment
	Load() (interface{}, error)

	// Validate validates configuration values
	Validate(config interface{}) error

	// GetDefaults returns default configuration values
	GetDefaults() interface{}
}

// Plugin defines the interface for extensibility plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Initialize initializes the plugin with context
	Initialize(ctx context.Context) error

	// Shutdown gracefully shuts down the plugin
	Shutdown() error

	// IsEnabled returns whether the plugin is enabled
	IsEnabled() bool
}

// PluginManager defines the interface for managing plugins
type PluginManager interface {
	// LoadPlugin loads a plugin from a path
	LoadPlugin(path string) (Plugin, error)

	// UnloadPlugin unloads a plugin by name
	UnloadPlugin(name string) error

	// ListPlugins returns all loaded plugins
	ListPlugins() []Plugin

	// ReloadPlugin reloads a plugin by name
	ReloadPlugin(name string) error

	// EnablePlugin enables a plugin
	EnablePlugin(name string) error

	// DisablePlugin disables a plugin
	DisablePlugin(name string) error
}

// ErrorCollector defines the interface for collecting and managing build errors
type ErrorCollector interface {
	// AddError adds an error to the collection
	AddError(err error, component *types.ComponentInfo)

	// GetErrors returns all collected errors
	GetErrors() []interface{}

	// ClearErrors clears all collected errors
	ClearErrors()

	// HasErrors returns true if there are collected errors
	HasErrors() bool

	// GenerateOverlay generates an HTML error overlay
	GenerateOverlay() (string, error)
}

// ServiceContainer defines the interface for dependency injection
type ServiceContainer interface {
	// Register registers a service with the container
	Register(name string, service interface{}) error

	// Get retrieves a service by name
	Get(name string) (interface{}, error)

	// GetRequired retrieves a service and panics if not found
	GetRequired(name string) interface{}

	// Has checks if a service is registered
	Has(name string) bool

	// GetAll returns all registered services
	GetAll() map[string]interface{}
}
