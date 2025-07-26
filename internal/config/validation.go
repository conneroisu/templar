// Package config provides validation functions for configuration values
// with security-focused checks and clear error messages.
package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidationError represents a configuration validation error with context
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult contains the result of configuration validation
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// HasErrors returns true if there are validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are validation warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// String returns a formatted string representation of the validation result
func (r *ValidationResult) String() string {
	if r.Valid {
		return "Configuration is valid"
	}

	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Error())
	}
	for _, warn := range r.Warnings {
		messages = append(messages, fmt.Sprintf("Warning: %s", warn.Error()))
	}

	return strings.Join(messages, "\n")
}

// ConfigValidator provides centralized validation for all configuration components
type ConfigValidator struct {
	errors []error
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errors: make([]error, 0),
	}
}

// ValidateAll performs comprehensive validation on the entire configuration
func (cv *ConfigValidator) ValidateAll(config *Config) error {
	cv.errors = cv.errors[:0] // Reset errors

	cv.validateServer(&config.Server)
	cv.validateBuild(&config.Build)
	cv.validateComponents(&config.Components)
	cv.validatePlugins(&config.Plugins)
	cv.validateMonitoring(&config.Monitoring)
	cv.validateProduction(&config.Production)

	if len(cv.errors) > 0 {
		return cv.combineErrors()
	}
	return nil
}

// validateServer validates server configuration with security checks
func (cv *ConfigValidator) validateServer(config *ServerConfig) {
	// Validate port range
	if config.Port < 0 || config.Port > 65535 {
		cv.addError("server", fmt.Errorf("port %d is not in valid range 0-65535", config.Port))
	}

	// Validate host for security
	if config.Host != "" {
		if err := cv.validateHostSecurity(config.Host); err != nil {
			cv.addError("server.host", err)
		}
	}

	// Validate environment
	validEnvs := []string{"development", "staging", "production", "test"}
	if config.Environment != "" && !cv.contains(validEnvs, config.Environment) {
		cv.addError("server.environment", fmt.Errorf("invalid environment '%s', must be one of: %v", config.Environment, validEnvs))
	}

	// Validate authentication
	cv.validateAuth(&config.Auth)
}

// validateAuth validates authentication configuration
func (cv *ConfigValidator) validateAuth(config *AuthConfig) {
	if !config.Enabled {
		return
	}

	validModes := []string{"token", "basic", "none"}
	if !cv.contains(validModes, config.Mode) {
		cv.addError("server.auth.mode", fmt.Errorf("invalid auth mode '%s', must be one of: %v", config.Mode, validModes))
	}

	// Validate mode-specific requirements
	switch config.Mode {
	case "token":
		if config.Token == "" && config.RequireAuth {
			cv.addError("server.auth.token", fmt.Errorf("token is required when auth mode is 'token'"))
		}
	case "basic":
		if (config.Username == "" || config.Password == "") && config.RequireAuth {
			cv.addError("server.auth.basic", fmt.Errorf("username and password are required when auth mode is 'basic'"))
		}
	}

	// Validate allowed IPs
	for i, ip := range config.AllowedIPs {
		if err := cv.validateIPAddress(ip); err != nil {
			cv.addError(fmt.Sprintf("server.auth.allowed_ips[%d]", i), err)
		}
	}
}

// validateBuild validates build configuration
func (cv *ConfigValidator) validateBuild(config *BuildConfig) {
	// Validate cache directory
	if config.CacheDir != "" {
		if err := cv.validateSecurePath(config.CacheDir); err != nil {
			cv.addError("build.cache_dir", err)
		}
	}

	// Validate build command (basic security check)
	if config.Command != "" {
		dangerousChars := []string{";", "&", "|", "$(", "`"}
		for _, char := range dangerousChars {
			if strings.Contains(config.Command, char) {
				cv.addError("build.command", fmt.Errorf("command contains potentially dangerous character: %s", char))
			}
		}
	}

	// Validate watch patterns
	for i, pattern := range config.Watch {
		if pattern == "" {
			cv.addError(fmt.Sprintf("build.watch[%d]", i), fmt.Errorf("empty watch pattern"))
		}
	}
}

// validateComponents validates components configuration
func (cv *ConfigValidator) validateComponents(config *ComponentsConfig) {
	// Validate scan paths
	if len(config.ScanPaths) == 0 {
		cv.addError("components.scan_paths", fmt.Errorf("at least one scan path is required"))
	}

	for i, path := range config.ScanPaths {
		if err := cv.validatePath(path); err != nil {
			cv.addError(fmt.Sprintf("components.scan_paths[%d]", i), err)
		}
	}

	// Validate exclude patterns
	for i, pattern := range config.ExcludePatterns {
		if pattern == "" {
			cv.addError(fmt.Sprintf("components.exclude_patterns[%d]", i), fmt.Errorf("empty exclude pattern"))
		}
	}
}

// validatePlugins validates plugins configuration
func (cv *ConfigValidator) validatePlugins(config *PluginsConfig) {
	// Validate discovery paths
	for i, path := range config.DiscoveryPaths {
		if err := cv.validatePath(path); err != nil {
			cv.addError(fmt.Sprintf("plugins.discovery_paths[%d]", i), err)
		}
	}

	// Check for conflicts between enabled and disabled
	for _, plugin := range config.Enabled {
		if cv.contains(config.Disabled, plugin) {
			cv.addError("plugins.enabled", fmt.Errorf("plugin '%s' is both enabled and disabled", plugin))
		}
	}
}

// validateMonitoring validates monitoring configuration
func (cv *ConfigValidator) validateMonitoring(config *MonitoringConfig) {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !cv.contains(validLogLevels, config.LogLevel) {
		cv.addError("monitoring.log_level", fmt.Errorf("invalid log level '%s', must be one of: %v", config.LogLevel, validLogLevels))
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	if !cv.contains(validLogFormats, config.LogFormat) {
		cv.addError("monitoring.log_format", fmt.Errorf("invalid log format '%s', must be one of: %v", config.LogFormat, validLogFormats))
	}

	// Validate HTTP port
	if config.HTTPPort < 0 || config.HTTPPort > 65535 {
		cv.addError("monitoring.http_port", fmt.Errorf("HTTP port %d is not in valid range 0-65535", config.HTTPPort))
	}

	// Validate metrics path
	if config.MetricsPath != "" {
		if err := cv.validatePath(config.MetricsPath); err != nil {
			cv.addError("monitoring.metrics_path", err)
		}
	}
}

// validateProduction validates production configuration
func (cv *ConfigValidator) validateProduction(config *ProductionConfig) {
	// Validate output directories
	if err := cv.validatePath(config.OutputDir); err != nil {
		cv.addError("production.output_dir", err)
	}
	if err := cv.validatePath(config.StaticDir); err != nil {
		cv.addError("production.static_dir", err)
	}
	if err := cv.validatePath(config.AssetsDir); err != nil {
		cv.addError("production.assets_dir", err)
	}

	// Validate compression settings
	cv.validateCompression(&config.Compression)

	// Validate security settings
	cv.validateSecurity(&config.Security)

	// Validate deployment settings
	cv.validateDeployment(&config.Deployment)
}

// validateCompression validates compression settings
func (cv *ConfigValidator) validateCompression(config *CompressionSettings) {
	if !config.Enabled {
		return
	}

	// Validate compression level
	if config.Level < 1 || config.Level > 9 {
		cv.addError("production.compression.level", fmt.Errorf("compression level %d is not in valid range 1-9", config.Level))
	}

	// Validate algorithms
	validAlgorithms := []string{"gzip", "brotli", "deflate"}
	for i, algo := range config.Algorithms {
		if !cv.contains(validAlgorithms, algo) {
			cv.addError(fmt.Sprintf("production.compression.algorithms[%d]", i), fmt.Errorf("invalid compression algorithm '%s'", algo))
		}
	}

	// Validate file extensions
	for i, ext := range config.Extensions {
		if !strings.HasPrefix(ext, ".") {
			cv.addError(fmt.Sprintf("production.compression.extensions[%d]", i), fmt.Errorf("file extension '%s' must start with '.'", ext))
		}
	}
}

// validateSecurity validates security settings
func (cv *ConfigValidator) validateSecurity(config *SecuritySettings) {
	// Validate X-Frame-Options
	validFrameOptions := []string{"DENY", "SAMEORIGIN"}
	if config.XFrameOptions != "" && !cv.contains(validFrameOptions, config.XFrameOptions) {
		cv.addError("production.security.x_frame_options", fmt.Errorf("invalid X-Frame-Options '%s'", config.XFrameOptions))
	}

	// Validate CSP
	if config.CSP.Enabled && config.CSP.ReportURI != "" {
		if !strings.HasPrefix(config.CSP.ReportURI, "http") {
			cv.addError("production.security.csp.report_uri", fmt.Errorf("CSP report URI must be a valid HTTP URL"))
		}
	}
}

// validateDeployment validates deployment settings
func (cv *ConfigValidator) validateDeployment(config *DeploymentSettings) {
	validTargets := []string{"static", "docker", "serverless"}
	if config.Target != "" && !cv.contains(validTargets, config.Target) {
		cv.addError("production.deployment.target", fmt.Errorf("invalid deployment target '%s'", config.Target))
	}

	// Validate redirects
	for i, redirect := range config.Redirects {
		if redirect.From == "" || redirect.To == "" {
			cv.addError(fmt.Sprintf("production.deployment.redirects[%d]", i), fmt.Errorf("redirect from and to fields are required"))
		}
		if redirect.Status < 300 || redirect.Status >= 400 {
			cv.addError(fmt.Sprintf("production.deployment.redirects[%d].status", i), fmt.Errorf("invalid redirect status %d", redirect.Status))
		}
	}
}

// Security validation helpers

// validateHostSecurity checks for dangerous characters in host configuration
func (cv *ConfigValidator) validateHostSecurity(host string) error {
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(host, char) {
			return fmt.Errorf("host contains dangerous character: %s", char)
		}
	}
	return nil
}

// validateSecurePath validates paths with enhanced security checks
func (cv *ConfigValidator) validateSecurePath(path string) error {
	if err := cv.validatePath(path); err != nil {
		return err
	}

	// Additional security: should be relative path for most configurations
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("path should be relative for security: %s", path)
	}

	return nil
}

// validatePath performs basic path validation
func (cv *ConfigValidator) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	cleanPath := filepath.Clean(path)

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains traversal: %s", path)
	}

	// Check for dangerous characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'"}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	return nil
}

// validateIPAddress performs basic IP address validation
func (cv *ConfigValidator) validateIPAddress(ip string) error {
	// Basic validation - should be enhanced with proper IP parsing
	if ip == "" {
		return fmt.Errorf("empty IP address")
	}

	// Check for dangerous characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(ip, char) {
			return fmt.Errorf("IP address contains dangerous character: %s", char)
		}
	}

	return nil
}

// Utility functions

// addError adds an error with context
func (cv *ConfigValidator) addError(field string, err error) {
	cv.errors = append(cv.errors, fmt.Errorf("%s: %w", field, err))
}

// contains checks if a slice contains a value
func (cv *ConfigValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// combineErrors combines multiple validation errors into a single error
func (cv *ConfigValidator) combineErrors() error {
	if len(cv.errors) == 1 {
		return cv.errors[0]
	}

	var messages []string
	for _, err := range cv.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Errorf("multiple validation errors:\n  - %s", strings.Join(messages, "\n  - "))
}

// ValidateConfigWithDetails performs comprehensive validation with detailed feedback
func ValidateConfigWithDetails(config *Config) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Validate server configuration
	validateServerConfigDetails(&config.Server, result)

	// Validate build configuration
	validateBuildConfigDetails(&config.Build, result)

	// Validate components configuration
	validateComponentsConfigDetails(&config.Components, result)

	// Validate preview configuration
	validatePreviewConfigDetails(&config.Preview, result)

	// Validate development configuration
	validateDevelopmentConfigDetails(&config.Development, result)

	// Validate plugins configuration
	validatePluginsConfigDetails(&config.Plugins, result)

	// Set overall validity
	result.Valid = !result.HasErrors()

	return result
}

func validateServerConfigDetails(config *ServerConfig, result *ValidationResult) {
	// Implementation from existing detailed validation
	// For brevity, using basic validation here - can be expanded
	if err := validateServerConfig(config); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "server",
			Value:   config,
			Message: err.Error(),
		})
	}
}

func validateBuildConfigDetails(config *BuildConfig, result *ValidationResult) {
	if err := validateBuildConfig(config); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "build",
			Value:   config,
			Message: err.Error(),
		})
	}
}

func validateComponentsConfigDetails(config *ComponentsConfig, result *ValidationResult) {
	if err := validateComponentsConfig(config); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "components",
			Value:   config,
			Message: err.Error(),
		})
	}
}

func validatePreviewConfigDetails(config *PreviewConfig, result *ValidationResult) {
	// Basic preview validation
	if config.MockData != "auto" && config.MockData != "none" && config.MockData != "" {
		// Validate as path
		if err := validatePath(config.MockData); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "preview.mock_data",
				Value:   config.MockData,
				Message: err.Error(),
			})
		}
	}
}

func validateDevelopmentConfigDetails(config *DevelopmentConfig, result *ValidationResult) {
	// No specific errors for development config, just warnings
	if config.HotReload && config.StatePreservation {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "development",
			Value:   "hot_reload + state_preservation",
			Message: "state preservation with hot reload may cause unexpected behavior",
		})
	}
}

func validatePluginsConfigDetails(config *PluginsConfig, result *ValidationResult) {
	if err := validatePluginsConfig(config); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "plugins",
			Value:   config,
			Message: err.Error(),
		})
	}
}

// Legacy validation functions for backward compatibility

// validateConfig is the main validation function used by the existing Load() function
func validateConfig(config *Config) error {
	validator := NewConfigValidator()
	return validator.ValidateAll(config)
}

// Individual validation functions for backward compatibility
func validateServerConfig(config *ServerConfig) error {
	validator := NewConfigValidator()
	validator.validateServer(config)
	if len(validator.errors) > 0 {
		return validator.combineErrors()
	}
	return nil
}

func validateBuildConfig(config *BuildConfig) error {
	validator := NewConfigValidator()
	validator.validateBuild(config)
	if len(validator.errors) > 0 {
		return validator.combineErrors()
	}
	return nil
}

func validateComponentsConfig(config *ComponentsConfig) error {
	validator := NewConfigValidator()
	validator.validateComponents(config)
	if len(validator.errors) > 0 {
		return validator.combineErrors()
	}
	return nil
}

func validatePluginsConfig(config *PluginsConfig) error {
	validator := NewConfigValidator()
	validator.validatePlugins(config)
	if len(validator.errors) > 0 {
		return validator.combineErrors()
	}
	return nil
}


// validatePath for backward compatibility
func validatePath(path string) error {
	validator := NewConfigValidator()
	return validator.validatePath(path)
}
