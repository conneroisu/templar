// Package config provides configuration management for Templar applications
// using Viper for flexible configuration loading from files, environment
// variables, and command-line flags.
//
// The configuration system supports YAML files, environment variable overrides
// with TEMPLAR_ prefix, validation, and security checks. It manages server
// settings, component scanning paths, build pipeline configuration, and
// development-specific options like hot reload and error overlays.
package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Build       BuildConfig       `yaml:"build"`
	Preview     PreviewConfig     `yaml:"preview"`
	Components  ComponentsConfig  `yaml:"components"`
	Development DevelopmentConfig `yaml:"development"`
	Plugins     PluginsConfig     `yaml:"plugins"`
	TargetFiles []string          `yaml:"-"` // CLI arguments, not from config file
}

type ServerConfig struct {
	Port           int      `yaml:"port"`
	Host           string   `yaml:"host"`
	Open           bool     `yaml:"open"`
	NoOpen         bool     `yaml:"no-open"`
	Middleware     []string `yaml:"middleware"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	Environment    string   `yaml:"environment"`
}

type BuildConfig struct {
	Command  string   `yaml:"command"`
	Watch    []string `yaml:"watch"`
	Ignore   []string `yaml:"ignore"`
	CacheDir string   `yaml:"cache_dir"`
}

type PreviewConfig struct {
	MockData  string `yaml:"mock_data"`
	Wrapper   string `yaml:"wrapper"`
	AutoProps bool   `yaml:"auto_props"`
}

type ComponentsConfig struct {
	ScanPaths       []string `yaml:"scan_paths"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

type DevelopmentConfig struct {
	HotReload         bool `yaml:"hot_reload"`
	CSSInjection      bool `yaml:"css_injection"`
	StatePreservation bool `yaml:"state_preservation"`
	ErrorOverlay      bool `yaml:"error_overlay"`
}

type PluginsConfig struct {
	Enabled        []string                    `yaml:"enabled"`
	Disabled       []string                    `yaml:"disabled"`
	DiscoveryPaths []string                    `yaml:"discovery_paths"`
	Configurations map[string]PluginConfigMap `yaml:"configurations"`
}

type PluginConfigMap map[string]interface{}

func Load() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Apply defaults for components scan paths only if not explicitly set
	if !viper.IsSet("components.scan_paths") && len(config.Components.ScanPaths) == 0 {
		config.Components.ScanPaths = []string{"./components", "./views", "./examples"}
	}

	// Handle scan_paths set via viper (workaround for viper slice handling)
	if viper.IsSet("components.scan_paths") && len(config.Components.ScanPaths) == 0 {
		scanPaths := viper.GetStringSlice("components.scan_paths")
		if len(scanPaths) > 0 {
			config.Components.ScanPaths = scanPaths
		}
	}

	// Handle development settings set via viper (workaround for viper bool handling)
	if viper.IsSet("development.hot_reload") {
		config.Development.HotReload = viper.GetBool("development.hot_reload")
	}
	if viper.IsSet("development.css_injection") {
		config.Development.CSSInjection = viper.GetBool("development.css_injection")
	}
	if viper.IsSet("development.state_preservation") {
		config.Development.StatePreservation = viper.GetBool("development.state_preservation")
	}
	if viper.IsSet("development.error_overlay") {
		config.Development.ErrorOverlay = viper.GetBool("development.error_overlay")
	}

	// Handle preview settings
	if viper.IsSet("preview.auto_props") {
		config.Preview.AutoProps = viper.GetBool("preview.auto_props")
	}

	// Handle exclude patterns set via viper (workaround for viper slice handling)
	if viper.IsSet("components.exclude_patterns") && len(config.Components.ExcludePatterns) == 0 {
		excludePatterns := viper.GetStringSlice("components.exclude_patterns")
		if len(excludePatterns) > 0 {
			config.Components.ExcludePatterns = excludePatterns
		}
	}

	// Apply default values for BuildConfig if not set
	if config.Build.Command == "" {
		config.Build.Command = "templ generate"
	}
	if len(config.Build.Watch) == 0 {
		config.Build.Watch = []string{"**/*.templ"}
	}
	if len(config.Build.Ignore) == 0 {
		config.Build.Ignore = []string{"node_modules", ".git"}
	}
	if config.Build.CacheDir == "" {
		config.Build.CacheDir = ".templar/cache"
	}

	// Apply default values for PreviewConfig if not set
	if config.Preview.MockData == "" {
		config.Preview.MockData = "auto"
	}
	if config.Preview.Wrapper == "" {
		config.Preview.Wrapper = "layout.templ"
	}
	if !viper.IsSet("preview.auto_props") {
		config.Preview.AutoProps = true
	}

	// Apply default values for ComponentsConfig if not set
	if len(config.Components.ExcludePatterns) == 0 {
		config.Components.ExcludePatterns = []string{"*_test.templ", "*.bak"}
	}

	// Apply default values for DevelopmentConfig if not set
	if !viper.IsSet("development.hot_reload") {
		config.Development.HotReload = true
	}
	if !viper.IsSet("development.css_injection") {
		config.Development.CSSInjection = true
	}
	if !viper.IsSet("development.error_overlay") {
		config.Development.ErrorOverlay = true
	}

	// Override no-open if explicitly set via flag
	if viper.IsSet("server.no-open") && viper.GetBool("server.no-open") {
		config.Server.Open = false
	}

	// Apply default values for PluginsConfig if not set
	if len(config.Plugins.DiscoveryPaths) == 0 {
		config.Plugins.DiscoveryPaths = []string{"./plugins", "~/.templar/plugins"}
	}
	if config.Plugins.Configurations == nil {
		config.Plugins.Configurations = make(map[string]PluginConfigMap)
	}

	// Handle plugin configuration set via viper
	if viper.IsSet("plugins.enabled") {
		config.Plugins.Enabled = viper.GetStringSlice("plugins.enabled")
	}
	if viper.IsSet("plugins.disabled") {
		config.Plugins.Disabled = viper.GetStringSlice("plugins.disabled")
	}
	if viper.IsSet("plugins.discovery_paths") {
		config.Plugins.DiscoveryPaths = viper.GetStringSlice("plugins.discovery_paths")
	}

	// Validate configuration values
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig validates configuration values for security and correctness
func validateConfig(config *Config) error {
	// Validate server configuration
	if err := validateServerConfig(&config.Server); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate build configuration
	if err := validateBuildConfig(&config.Build); err != nil {
		return fmt.Errorf("build config: %w", err)
	}

	// Validate components configuration
	if err := validateComponentsConfig(&config.Components); err != nil {
		return fmt.Errorf("components config: %w", err)
	}

	// Validate plugins configuration
	if err := validatePluginsConfig(&config.Plugins); err != nil {
		return fmt.Errorf("plugins config: %w", err)
	}

	return nil
}

// validateServerConfig validates server configuration values
func validateServerConfig(config *ServerConfig) error {
	// Validate port range (allow 0 for system-assigned ports in testing)
	if config.Port < 0 || config.Port > 65535 {
		return fmt.Errorf("port %d is not in valid range 0-65535", config.Port)
	}

	// Validate host
	if config.Host != "" {
		// Basic validation - no dangerous characters
		dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
		for _, char := range dangerousChars {
			if strings.Contains(config.Host, char) {
				return fmt.Errorf("host contains dangerous character: %s", char)
			}
		}
	}

	return nil
}

// validateBuildConfig validates build configuration values
func validateBuildConfig(config *BuildConfig) error {
	// Validate cache directory if specified
	if config.CacheDir != "" {
		// Clean the path
		cleanPath := filepath.Clean(config.CacheDir)

		// Reject path traversal attempts
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("cache_dir contains path traversal: %s", config.CacheDir)
		}

		// Should be relative path for security
		if filepath.IsAbs(cleanPath) {
			return fmt.Errorf("cache_dir should be relative path: %s", config.CacheDir)
		}
	}

	return nil
}

// validateComponentsConfig validates components configuration values
func validateComponentsConfig(config *ComponentsConfig) error {
	// Validate scan paths
	for _, path := range config.ScanPaths {
		if err := validatePath(path); err != nil {
			return fmt.Errorf("invalid scan path '%s': %w", path, err)
		}
	}

	return nil
}

// validatePath validates a file path for security
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Reject path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains traversal: %s", path)
	}

	// Reject dangerous characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'"}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	return nil
}
