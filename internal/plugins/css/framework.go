package css

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/conneroisu/templar/internal/types"
)

// CSSFrameworkPlugin defines the interface for CSS framework plugins.
type CSSFrameworkPlugin interface {
	// Plugin base interface
	GetName() string
	GetVersion() string
	Initialize(ctx context.Context, config map[string]interface{}) error
	Cleanup() error

	// Framework-specific methods
	GetFrameworkName() string
	GetSupportedVersions() []string
	GetDefaultConfig() FrameworkConfig
	IsInstalled() bool

	// Setup and configuration
	Setup(ctx context.Context, config FrameworkConfig) error
	GenerateConfig(config FrameworkConfig) ([]byte, error)
	ValidateConfig(configPath string) error

	// CSS processing
	ProcessCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error)
	ExtractClasses(content string) ([]string, error)
	OptimizeCSS(ctx context.Context, css []byte, usedClasses []string) ([]byte, error)

	// Theming and variables
	ExtractVariables(css []byte) (map[string]string, error)
	GenerateTheme(variables map[string]string) ([]byte, error)

	// Development features
	GetDevServerConfig() DevServerConfig
	SupportsHotReload() bool
	GenerateStyleGuide(ctx context.Context) ([]byte, error)
}

// FrameworkConfig represents configuration for a CSS framework.
type FrameworkConfig struct {
	// Framework identification
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`

	// Installation options
	InstallMethod string `json:"install_method" yaml:"install_method"` // "npm", "cdn", "standalone"
	CDNUrl        string `json:"cdn_url,omitempty" yaml:"cdn_url,omitempty"`

	// Build configuration
	ConfigFile  string   `json:"config_file,omitempty" yaml:"config_file,omitempty"`
	EntryPoint  string   `json:"entry_point,omitempty" yaml:"entry_point,omitempty"`
	OutputPath  string   `json:"output_path" yaml:"output_path"`
	SourcePaths []string `json:"source_paths" yaml:"source_paths"`

	// Processing options
	Preprocessing []string           `json:"preprocessing,omitempty" yaml:"preprocessing,omitempty"` // "scss", "postcss", "less"
	Optimization  OptimizationConfig `json:"optimization" yaml:"optimization"`

	// Theming
	Theming ThemingConfig `json:"theming" yaml:"theming"`

	// Custom variables
	Variables map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`

	// Framework-specific options
	Options map[string]interface{} `json:"options,omitempty" yaml:"options,omitempty"`
}

// ProcessingOptions defines CSS processing options.
type ProcessingOptions struct {
	// Input/Output
	InputPath  string `json:"input_path,omitempty"`
	OutputPath string `json:"output_path,omitempty"`

	// Processing flags
	Minify     bool `json:"minify"`
	Purge      bool `json:"purge"`
	Optimize   bool `json:"optimize"`
	SourceMaps bool `json:"source_maps"`

	// Content analysis
	ContentPaths []string `json:"content_paths,omitempty"`
	UsedClasses  []string `json:"used_classes,omitempty"`

	// Environment
	Environment string `json:"environment"` // "development", "production"
}

// OptimizationConfig defines CSS optimization settings.
type OptimizationConfig struct {
	Enabled      bool `json:"enabled" yaml:"enabled"`
	Purge        bool `json:"purge" yaml:"purge"`
	Minify       bool `json:"minify" yaml:"minify"`
	TreeShake    bool `json:"tree_shake" yaml:"tree_shake"`
	Compress     bool `json:"compress" yaml:"compress"`
	RemoveUnused bool `json:"remove_unused" yaml:"remove_unused"`

	// Advanced options
	PurgeWhitelist []string `json:"purge_whitelist,omitempty" yaml:"purge_whitelist,omitempty"`
	PurgeContent   []string `json:"purge_content,omitempty" yaml:"purge_content,omitempty"`
}

// ThemingConfig defines theming and variable extraction settings.
type ThemingConfig struct {
	Enabled          bool `json:"enabled" yaml:"enabled"`
	ExtractVariables bool `json:"extract_variables" yaml:"extract_variables"`
	GenerateTokens   bool `json:"generate_tokens" yaml:"generate_tokens"`
	StyleGuide       bool `json:"style_guide" yaml:"style_guide"`

	// Theme customization
	PrimaryColor   string            `json:"primary_color,omitempty" yaml:"primary_color,omitempty"`
	SecondaryColor string            `json:"secondary_color,omitempty" yaml:"secondary_color,omitempty"`
	CustomColors   map[string]string `json:"custom_colors,omitempty" yaml:"custom_colors,omitempty"`

	// Typography
	FontFamily string            `json:"font_family,omitempty" yaml:"font_family,omitempty"`
	FontSizes  map[string]string `json:"font_sizes,omitempty" yaml:"font_sizes,omitempty"`

	// Spacing and layout
	Spacing      map[string]string `json:"spacing,omitempty" yaml:"spacing,omitempty"`
	BorderRadius map[string]string `json:"border_radius,omitempty" yaml:"border_radius,omitempty"`

	// Output options
	OutputFormat string `json:"output_format" yaml:"output_format"` // "css", "scss", "json"
	OutputFile   string `json:"output_file,omitempty" yaml:"output_file,omitempty"`
}

// DevServerConfig defines development server configuration for CSS frameworks.
type DevServerConfig struct {
	// Hot reload settings
	HotReload   bool     `json:"hot_reload"`
	WatchPaths  []string `json:"watch_paths"`
	ReloadDelay int      `json:"reload_delay"` // milliseconds

	// CSS injection
	CSSInjection bool   `json:"css_injection"`
	InjectTarget string `json:"inject_target"` // "head", "body"

	// Development features
	ErrorOverlay   bool `json:"error_overlay"`
	SourceMaps     bool `json:"source_maps"`
	LiveValidation bool `json:"live_validation"`

	// Framework-specific dev settings
	DevMode    bool                   `json:"dev_mode"`
	DevOptions map[string]interface{} `json:"dev_options,omitempty"`
}

// ComponentTemplate represents a framework-specific component template.
type ComponentTemplate struct {
	Name        string                `json:"name"`
	Framework   string                `json:"framework"`
	Category    string                `json:"category"` // "layout", "button", "form", "navigation", etc.
	Description string                `json:"description"`
	Template    string                `json:"template"`
	Props       []types.ParameterInfo `json:"props"`
	Examples    []TemplateExample     `json:"examples,omitempty"`

	// Framework-specific data
	Classes      []string          `json:"classes,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Requirements []string          `json:"requirements,omitempty"` // Dependencies or plugins needed
}

// TemplateExample represents an example usage of a component template.
type TemplateExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Props       map[string]interface{} `json:"props"`
	Preview     string                 `json:"preview,omitempty"`
}

// FrameworkInfo provides metadata about a CSS framework.
type FrameworkInfo struct {
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	Website       string `json:"website"`
	Documentation string `json:"documentation"`
	Repository    string `json:"repository"`

	// Support information
	SupportedVersions []string `json:"supported_versions"`
	InstallMethods    []string `json:"install_methods"`
	Prerequisites     []string `json:"prerequisites,omitempty"`

	// Framework characteristics
	Type     string   `json:"type"` // "utility", "component", "hybrid"
	Size     string   `json:"size"` // "small", "medium", "large"
	Features []string `json:"features"`

	// Integration details
	ConfigFiles      []string `json:"config_files"`
	OutputExtensions []string `json:"output_extensions"`
	SourceExtensions []string `json:"source_extensions"`
}

// FrameworkRegistry manages available CSS frameworks.
type FrameworkRegistry struct {
	frameworks map[string]CSSFrameworkPlugin
	configs    map[string]FrameworkConfig
}

// NewFrameworkRegistry creates a new framework registry.
func NewFrameworkRegistry() *FrameworkRegistry {
	return &FrameworkRegistry{
		frameworks: make(map[string]CSSFrameworkPlugin),
		configs:    make(map[string]FrameworkConfig),
	}
}

// Register registers a CSS framework plugin.
func (r *FrameworkRegistry) Register(plugin CSSFrameworkPlugin) error {
	name := plugin.GetFrameworkName()
	if name == "" {
		return errors.New("framework plugin must have a name")
	}

	if _, exists := r.frameworks[name]; exists {
		return fmt.Errorf("framework %s is already registered", name)
	}

	r.frameworks[name] = plugin

	return nil
}

// Get retrieves a framework plugin by name.
func (r *FrameworkRegistry) Get(name string) (CSSFrameworkPlugin, bool) {
	plugin, exists := r.frameworks[name]

	return plugin, exists
}

// List returns all registered framework names.
func (r *FrameworkRegistry) List() []string {
	names := make([]string, 0, len(r.frameworks))
	for name := range r.frameworks {
		names = append(names, name)
	}

	return names
}

// GetFrameworkInfo returns information about a framework.
func (r *FrameworkRegistry) GetFrameworkInfo(name string) (*FrameworkInfo, error) {
	plugin, exists := r.frameworks[name]
	if !exists {
		return nil, fmt.Errorf("framework %s not found", name)
	}

	// Build framework info from plugin
	info := &FrameworkInfo{
		Name:              plugin.GetFrameworkName(),
		DisplayName:       plugin.GetFrameworkName(),
		Version:           plugin.GetVersion(),
		SupportedVersions: plugin.GetSupportedVersions(),
	}

	return info, nil
}

// SetConfig sets configuration for a framework.
func (r *FrameworkRegistry) SetConfig(frameworkName string, config FrameworkConfig) {
	r.configs[frameworkName] = config
}

// GetConfig gets configuration for a framework.
func (r *FrameworkRegistry) GetConfig(frameworkName string) (FrameworkConfig, bool) {
	config, exists := r.configs[frameworkName]

	return config, exists
}

// DetectFramework attempts to detect which CSS framework is being used.
func (r *FrameworkRegistry) DetectFramework(projectPath string) ([]string, error) {
	var detected []string

	// Check for framework-specific config files
	configFiles := map[string][]string{
		"tailwind": {
			"tailwind.config.js",
			"tailwind.config.ts",
			"tailwind.config.cjs",
			"tailwind.config.mjs",
		},
		"bootstrap": {"bootstrap.config.js", "scss/bootstrap.scss", "css/bootstrap.css"},
		"bulma":     {"bulma.config.js", "sass/bulma.sass", "css/bulma.css"},
	}

	for framework, files := range configFiles {
		for _, file := range files {
			configPath := filepath.Join(projectPath, file)
			if fileExists(configPath) {
				detected = append(detected, framework)

				break
			}
		}
	}

	return detected, nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	if _, err := filepath.EvalSymlinks(path); err != nil {
		return false
	}

	return true
}

// ValidateFrameworkConfig validates a framework configuration.
func ValidateFrameworkConfig(config FrameworkConfig) error {
	if config.Name == "" {
		return errors.New("framework name is required")
	}

	if config.OutputPath == "" {
		return errors.New("output path is required")
	}

	if len(config.SourcePaths) == 0 {
		return errors.New("at least one source path is required")
	}

	// Validate install method
	validMethods := []string{"npm", "cdn", "standalone"}
	if config.InstallMethod != "" {
		valid := false
		for _, method := range validMethods {
			if config.InstallMethod == method {
				valid = true

				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid install method: %s", config.InstallMethod)
		}
	}

	return nil
}
