// Package config provides configuration management for Templar applications
// using Viper for flexible configuration loading from files, environment
// variables, and command-line flags.
//
// The configuration system supports:
//   - YAML configuration files with customizable paths via TEMPLAR_CONFIG_FILE
//   - Environment variable overrides with TEMPLAR_ prefix (e.g., TEMPLAR_SERVER_PORT)
//   - Command-line flag overrides with highest precedence
//   - Comprehensive validation and security checks for all configuration values
//
// Configuration Loading Order:
//   1. Command-line flags (--config, --port, etc.)
//   2. TEMPLAR_CONFIG_FILE environment variable for custom config file paths
//   3. Individual environment variables (TEMPLAR_SERVER_PORT, TEMPLAR_SERVER_HOST, etc.)
//   4. Configuration file values (.templar.yml or custom path)
//   5. Built-in defaults
//
// The system manages server settings, component scanning paths, build pipeline
// configuration, development-specific options like hot reload and error overlays,
// plugin management, and monitoring configuration.
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
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	TargetFiles []string          `yaml:"-"` // CLI arguments, not from config file
}

type ServerConfig struct {
	Port           int        `yaml:"port"`
	Host           string     `yaml:"host"`
	Open           bool       `yaml:"open"`
	NoOpen         bool       `yaml:"no-open"`
	Middleware     []string   `yaml:"middleware"`
	AllowedOrigins []string   `yaml:"allowed_origins"`
	Environment    string     `yaml:"environment"`
	Auth           AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Mode            string   `yaml:"mode"`             // "token", "basic", "none"
	Token           string   `yaml:"token"`            // Simple token for token mode
	Username        string   `yaml:"username"`         // Username for basic auth
	Password        string   `yaml:"password"`         // Password for basic auth
	AllowedIPs      []string `yaml:"allowed_ips"`      // IP allowlist
	RequireAuth     bool     `yaml:"require_auth"`     // Require auth for non-localhost
	LocalhostBypass bool     `yaml:"localhost_bypass"` // Allow localhost without auth
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
	Enabled        []string                   `yaml:"enabled"`
	Disabled       []string                   `yaml:"disabled"`
	DiscoveryPaths []string                   `yaml:"discovery_paths"`
	Configurations map[string]PluginConfigMap `yaml:"configurations"`
}

type PluginConfigMap map[string]interface{}

type MonitoringConfig struct {
	Enabled       bool   `yaml:"enabled"`
	LogLevel      string `yaml:"log_level"`
	LogFormat     string `yaml:"log_format"`
	MetricsPath   string `yaml:"metrics_path"`
	HTTPPort      int    `yaml:"http_port"`
	AlertsEnabled bool   `yaml:"alerts_enabled"`
}

// Load reads configuration from all available sources and returns a fully populated Config struct.
//
// This function expects that Viper has already been configured by cmd.initConfig() with:
//   - Config file path (from --config flag, TEMPLAR_CONFIG_FILE env var, or default .templar.yml)
//   - Environment variable binding with TEMPLAR_ prefix
//   - Automatic environment variable reading enabled
//
// The function applies intelligent defaults, handles Viper's quirks with slice/bool values,
// and performs comprehensive security validation on all configuration values.
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

	// Apply default values for AuthConfig if not set
	if config.Server.Auth.Mode == "" {
		config.Server.Auth.Mode = "none"
	}
	if !viper.IsSet("server.auth.enabled") {
		config.Server.Auth.Enabled = false
	}
	if !viper.IsSet("server.auth.localhost_bypass") {
		config.Server.Auth.LocalhostBypass = true // Default to allowing localhost without auth
	}
	if !viper.IsSet("server.auth.require_auth") {
		config.Server.Auth.RequireAuth = false // Default to not requiring auth
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

	// Apply default values for MonitoringConfig if not set
	if !viper.IsSet("monitoring.enabled") {
		config.Monitoring.Enabled = true // Enable monitoring by default
	}
	if config.Monitoring.LogLevel == "" {
		config.Monitoring.LogLevel = "info"
	}
	if config.Monitoring.LogFormat == "" {
		config.Monitoring.LogFormat = "json"
	}
	if config.Monitoring.MetricsPath == "" {
		config.Monitoring.MetricsPath = "./logs/metrics.json"
	}
	if config.Monitoring.HTTPPort == 0 {
		config.Monitoring.HTTPPort = 8081
	}
	if !viper.IsSet("monitoring.alerts_enabled") {
		config.Monitoring.AlertsEnabled = false // Disable alerts by default
	}

	// Handle monitoring configuration set via viper
	if viper.IsSet("monitoring.enabled") {
		config.Monitoring.Enabled = viper.GetBool("monitoring.enabled")
	}
	if viper.IsSet("monitoring.log_level") {
		config.Monitoring.LogLevel = viper.GetString("monitoring.log_level")
	}
	if viper.IsSet("monitoring.log_format") {
		config.Monitoring.LogFormat = viper.GetString("monitoring.log_format")
	}
	if viper.IsSet("monitoring.metrics_path") {
		config.Monitoring.MetricsPath = viper.GetString("monitoring.metrics_path")
	}
	if viper.IsSet("monitoring.http_port") {
		config.Monitoring.HTTPPort = viper.GetInt("monitoring.http_port")
	}
	if viper.IsSet("monitoring.alerts_enabled") {
		config.Monitoring.AlertsEnabled = viper.GetBool("monitoring.alerts_enabled")
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

	// Validate monitoring configuration
	if err := validateMonitoringConfig(&config.Monitoring); err != nil {
		return fmt.Errorf("monitoring config: %w", err)
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

// validateMonitoringConfig validates monitoring configuration values
func validateMonitoringConfig(config *MonitoringConfig) error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log level '%s', must be one of: %v", config.LogLevel, validLogLevels)
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	isValidLogFormat := false
	for _, format := range validLogFormats {
		if config.LogFormat == format {
			isValidLogFormat = true
			break
		}
	}
	if !isValidLogFormat {
		return fmt.Errorf("invalid log format '%s', must be one of: %v", config.LogFormat, validLogFormats)
	}

	// Validate HTTP port
	if config.HTTPPort < 0 || config.HTTPPort > 65535 {
		return fmt.Errorf("HTTP port %d is not in valid range 0-65535", config.HTTPPort)
	}

	// Validate metrics path
	if config.MetricsPath != "" {
		if err := validatePath(config.MetricsPath); err != nil {
			return fmt.Errorf("invalid metrics path '%s': %w", config.MetricsPath, err)
		}
	}

	return nil
}
