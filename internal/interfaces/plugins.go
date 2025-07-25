package interfaces

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// Note: Plugin interface is defined in core.go to avoid duplication

// ComponentPlugin extends Plugin for component-specific processing
//
// Component plugins can intercept and modify components during scanning,
// building, or rendering phases.
//
// Performance:
//   HandleComponent should complete within 100ms for typical components
//   to avoid blocking the build pipeline.
type ComponentPlugin interface {
	Plugin

	// HandleComponent processes a component and optionally modifies it
	// Called for each discovered component during scanning
	// Must return the (possibly modified) component or an error
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout
	//   component: The component to process
	//
	// Returns:
	//   Modified component (or original if unchanged)
	//   Error if processing failed
	HandleComponent(ctx context.Context, component *types.ComponentInfo) (*types.ComponentInfo, error)
}

// BuildPlugin extends Plugin for build process hooks
//
// Build plugins can execute custom logic before and after the build process,
// enabling custom validation, asset generation, or deployment tasks.
type BuildPlugin interface {
	Plugin

	// PreBuild executes before the build process starts
	// Receives all components that will be built
	// Can modify components or perform setup tasks
	PreBuild(ctx context.Context, components []*types.ComponentInfo) error

	// PostBuild executes after the build process completes
	// Receives all build results for analysis or further processing
	PostBuild(ctx context.Context, results []BuildResult) error
}

// ServerPlugin extends Plugin for HTTP server extensions
//
// Server plugins can register custom routes, middleware, and request handlers
// to extend the development server functionality.
type ServerPlugin interface {
	Plugin

	// RegisterRoutes registers custom HTTP routes with the server
	// Called during server initialization
	RegisterRoutes(router HTTPRouter) error

	// HandleRequest processes HTTP requests for plugin-specific routes
	// Called for each request to plugin routes
	HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
}

// WatcherPlugin extends Plugin for file watching extensions
//
// Watcher plugins can respond to file system changes and trigger
// custom actions like code generation or asset processing.
type WatcherPlugin interface {
	Plugin

	// HandleFileChange responds to file system change events
	// Called whenever watched files are modified
	HandleFileChange(ctx context.Context, event FileChangeEvent) error

	// GetWatchPatterns returns file patterns this plugin wants to watch
	// Called during watcher initialization
	GetWatchPatterns() []string
}

// CSSFrameworkPlugin provides CSS framework integration
//
// Note: This interface is currently large and should be split according
// to Interface Segregation Principle in future refactoring.
type CSSFrameworkPlugin interface {
	Plugin

	// Framework identification
	GetFrameworkName() string
	GetFrameworkVersion() string
	IsSupported(version string) bool

	// Setup and configuration
	Setup(ctx context.Context, config CSSFrameworkConfig) error
	GetConfiguration() CSSFrameworkConfig
	UpdateConfiguration(config CSSFrameworkConfig) error

	// Asset management
	GetAssetPaths() []string
	InstallAssets(ctx context.Context, targetDir string) error
	UpdateAssets(ctx context.Context) error

	// Component generation
	GenerateComponent(ctx context.Context, spec ComponentSpec) (*types.ComponentInfo, error)
	ListAvailableComponents() []ComponentSpec
	GetComponentTemplate(name string) (string, error)

	// Build integration
	ProcessCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error)
	OptimizeCSS(ctx context.Context, css []byte) ([]byte, error)
	ValidateCSS(ctx context.Context, css []byte) []ValidationError
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Component    *types.ComponentInfo
	Output       []byte
	Error        error
	Duration     time.Duration
	CacheHit     bool
	Hash         string
	ParsedErrors []ParsedError
}

// HTTPRouter defines routing capabilities for server plugins
type HTTPRouter interface {
	// GET registers a GET route handler
	GET(path string, handler HTTPHandlerFunc)
	
	// POST registers a POST route handler
	POST(path string, handler HTTPHandlerFunc)
	
	// PUT registers a PUT route handler
	PUT(path string, handler HTTPHandlerFunc)
	
	// DELETE registers a DELETE route handler
	DELETE(path string, handler HTTPHandlerFunc)
	
	// Use registers middleware for all routes
	Use(middleware HTTPMiddlewareFunc)
	
	// Group creates a route group with common prefix/middleware
	Group(prefix string) HTTPRouter
}

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	Method     string
	Path       string
	Headers    map[string]string
	Query      map[string]string
	Body       []byte
	RemoteAddr string
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// HTTPHandlerFunc is a function that handles HTTP requests
type HTTPHandlerFunc func(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)

// HTTPMiddlewareFunc is a function that processes requests before handlers
type HTTPMiddlewareFunc func(next HTTPHandlerFunc) HTTPHandlerFunc

// FileChangeEvent represents a file system change
type FileChangeEvent struct {
	Path      string
	Operation FileOperation
	Timestamp time.Time
}

// FileOperation represents the type of file system operation
type FileOperation int

const (
	FileOperationCreate FileOperation = iota
	FileOperationModify
	FileOperationDelete
	FileOperationRename
)

// CSSFrameworkConfig represents CSS framework configuration
type CSSFrameworkConfig struct {
	Name           string
	Version        string
	AssetDirectory string
	OutputPath     string
	Optimize       bool
	Minify         bool
	PurgeCSS       bool
	CustomConfig   map[string]interface{}
}

// ComponentSpec represents a component specification for generation
type ComponentSpec struct {
	Name        string
	Type        string
	Props       []PropSpec
	Template    string
	StyleSheet  string
	Description string
}

// PropSpec represents a component property specification
type PropSpec struct {
	Name         string
	Type         string
	Required     bool
	DefaultValue interface{}
	Description  string
	Validation   []ValidationRule
}

// ValidationRule represents a validation rule for component properties
type ValidationRule struct {
	Type    string
	Value   interface{}
	Message string
}

// ProcessingOptions represents options for CSS processing
type ProcessingOptions struct {
	InputPath    string
	OutputPath   string
	Optimize     bool
	Minify       bool
	SourceMaps   bool
	CustomVars   map[string]string
}

// ValidationError represents a CSS validation error
type ValidationError struct {
	Message  string
	Line     int
	Column   int
	Severity ValidationSeverity
}

// ValidationSeverity represents the severity of a validation error
type ValidationSeverity int

const (
	ValidationSeverityError ValidationSeverity = iota
	ValidationSeverityWarning
	ValidationSeverityInfo
)

// ParsedError represents a parsed build error with enhanced information
type ParsedError struct {
	Message   string
	File      string
	Line      int
	Column    int
	Severity  string
	Code      string
	Context   map[string]interface{}
}

// FormatError formats the error for display
func (pe *ParsedError) FormatError() string {
	if pe.File != "" && pe.Line > 0 {
		return fmt.Sprintf("[%s] %s in %s:%d:%d\n  %s\n", 
			pe.Severity, pe.Code, pe.File, pe.Line, pe.Column, pe.Message)
	}
	return fmt.Sprintf("[%s] %s: %s\n", pe.Severity, pe.Code, pe.Message)
}

// String returns the file operation as a string
func (op FileOperation) String() string {
	switch op {
	case FileOperationCreate:
		return "CREATE"
	case FileOperationModify:
		return "MODIFY"
	case FileOperationDelete:
		return "DELETE"
	case FileOperationRename:
		return "RENAME"
	default:
		return "UNKNOWN"
	}
}

// String returns the validation severity as a string
func (vs ValidationSeverity) String() string {
	switch vs {
	case ValidationSeverityError:
		return "ERROR"
	case ValidationSeverityWarning:
		return "WARNING"
	case ValidationSeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}