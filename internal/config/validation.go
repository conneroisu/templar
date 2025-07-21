package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/conneroisu/templar/internal/validation"
)

// ValidationError represents a configuration validation error with suggestions
type ValidationError struct {
	Field       string
	Value       interface{}
	Message     string
	Suggestions []string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation error in %s: %s", ve.Field, ve.Message)
}

// ValidationResult holds the result of configuration validation
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationError
}

// HasErrors returns true if there are any validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings returns true if there are any validation warnings
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// String returns a formatted string of all validation issues
func (vr *ValidationResult) String() string {
	var builder strings.Builder

	if len(vr.Errors) > 0 {
		builder.WriteString("❌ Validation Errors:\n")
		for _, err := range vr.Errors {
			builder.WriteString(fmt.Sprintf("  • %s: %s\n", err.Field, err.Message))
			for _, suggestion := range err.Suggestions {
				builder.WriteString(fmt.Sprintf("    💡 %s\n", suggestion))
			}
		}
		builder.WriteString("\n")
	}

	if len(vr.Warnings) > 0 {
		builder.WriteString("⚠️  Validation Warnings:\n")
		for _, warning := range vr.Warnings {
			builder.WriteString(fmt.Sprintf("  • %s: %s\n", warning.Field, warning.Message))
			for _, suggestion := range warning.Suggestions {
				builder.WriteString(fmt.Sprintf("    💡 %s\n", suggestion))
			}
		}
	}

	return builder.String()
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
	// Validate port
	if config.Port < 0 || config.Port > 65535 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "server.port",
			Value:   config.Port,
			Message: fmt.Sprintf("port %d is not in valid range 0-65535", config.Port),
			Suggestions: []string{
				"Use a port between 1024-65535 for non-privileged access",
				"Common development ports: 3000, 8080, 8000, 3001",
				"Port 0 allows system to assign an available port",
			},
		})
	} else if config.Port > 0 && config.Port < 1024 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "server.port",
			Value:   config.Port,
			Message: "port below 1024 requires elevated privileges",
			Suggestions: []string{
				"Consider using a port above 1024 for development",
				"Use sudo if you need to bind to privileged ports",
			},
		})
	}

	// Validate host
	if config.Host != "" {
		if err := validateHostname(config.Host); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "server.host",
				Value:   config.Host,
				Message: err.Error(),
				Suggestions: []string{
					"Use 'localhost' for local development",
					"Use '0.0.0.0' to bind to all interfaces",
					"Use a valid IP address or hostname",
				},
			})
		}
	}

	// Validate environment
	validEnvs := []string{"development", "production", "testing"}
	if config.Environment != "" && !contains(validEnvs, config.Environment) {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "server.environment",
			Value:   config.Environment,
			Message: "unknown environment type",
			Suggestions: []string{
				"Use 'development' for local development",
				"Use 'production' for production deployments",
				"Use 'testing' for automated testing",
			},
		})
	}

	// Validate middleware
	validMiddleware := []string{"cors", "logger", "security", "ratelimit"}
	for _, middleware := range config.Middleware {
		if !contains(validMiddleware, middleware) {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   "server.middleware",
				Value:   middleware,
				Message: fmt.Sprintf("unknown middleware '%s'", middleware),
				Suggestions: []string{
					"Available middleware: " + strings.Join(validMiddleware, ", "),
					"Check plugin documentation for additional middleware",
				},
			})
		}
	}
}

func validateBuildConfigDetails(config *BuildConfig, result *ValidationResult) {
	// Validate build command
	if config.Command == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "build.command",
			Value:   config.Command,
			Message: "build command cannot be empty",
			Suggestions: []string{
				"Use 'templ generate' for standard templ projects",
				"Use 'go generate ./...' for Go projects with generate directives",
				"Specify custom build command if needed",
			},
		})
	} else if err := validateBuildCommand(config.Command); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "build.command",
			Value:   config.Command,
			Message: err.Error(),
			Suggestions: []string{
				"Avoid shell metacharacters in build commands",
				"Use absolute paths or ensure commands are in PATH",
				"Test the command manually before using in configuration",
			},
		})
	}

	// Validate cache directory
	if config.CacheDir != "" {
		if err := validation.ValidatePath(config.CacheDir); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "build.cache_dir",
				Value:   config.CacheDir,
				Message: err.Error(),
				Suggestions: []string{
					"Use relative paths like '.templar/cache'",
					"Avoid parent directory references (..)",
					"Ensure directory is writable",
				},
			})
		}
	}

	// Validate watch patterns
	if len(config.Watch) == 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "build.watch",
			Value:   config.Watch,
			Message: "no watch patterns specified - auto-rebuild disabled",
			Suggestions: []string{
				"Add '**/*.templ' to watch templ files",
				"Add '**/*.go' to watch Go files",
				"Use specific patterns for better performance",
			},
		})
	}

	// Validate ignore patterns
	recommendedIgnore := []string{"node_modules", ".git", "vendor", "*_test.go"}
	for _, recommended := range recommendedIgnore {
		if !containsPattern(config.Ignore, recommended) {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   "build.ignore",
				Value:   config.Ignore,
				Message: fmt.Sprintf("consider ignoring '%s' for better performance", recommended),
				Suggestions: []string{
					fmt.Sprintf("Add '%s' to ignore patterns", recommended),
				},
			})
		}
	}
}

func validateComponentsConfigDetails(config *ComponentsConfig, result *ValidationResult) {
	// Validate scan paths
	if len(config.ScanPaths) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "components.scan_paths",
			Value:   config.ScanPaths,
			Message: "no scan paths specified - no components will be found",
			Suggestions: []string{
				"Add './components' to scan for components",
				"Add './views' to scan for view templates",
				"Add './examples' to scan for example components",
			},
		})
	}

	for i, path := range config.ScanPaths {
		if err := validation.ValidatePath(path); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("components.scan_paths[%d]", i),
				Value:   path,
				Message: err.Error(),
				Suggestions: []string{
					"Use relative paths from project root",
					"Ensure directories exist or will be created",
					"Avoid parent directory references (..)",
				},
			})
		}

		// Check if path exists
		if !pathExists(path) {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   fmt.Sprintf("components.scan_paths[%d]", i),
				Value:   path,
				Message: "directory does not exist",
				Suggestions: []string{
					"Create the directory: mkdir -p " + path,
					"Remove the path if not needed",
					"Check for typos in the path",
				},
			})
		}
	}

	// Validate exclude patterns
	if len(config.ExcludePatterns) == 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "components.exclude_patterns",
			Value:   config.ExcludePatterns,
			Message: "no exclusion patterns - test files and backups may be included",
			Suggestions: []string{
				"Add '*_test.templ' to exclude test files",
				"Add '*.bak' to exclude backup files",
				"Add '*.example.templ' to exclude example files",
			},
		})
	}
}

func validatePreviewConfigDetails(config *PreviewConfig, result *ValidationResult) {
	// Validate mock data configuration
	if config.MockData != "" && config.MockData != "auto" && config.MockData != "none" {
		// Assume it's a file path
		if err := validation.ValidatePath(config.MockData); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "preview.mock_data",
				Value:   config.MockData,
				Message: err.Error(),
				Suggestions: []string{
					"Use 'auto' for automatic mock data generation",
					"Use 'none' to disable mock data",
					"Specify a valid directory path for manual mock data",
				},
			})
		}
	}

	// Validate wrapper template
	if config.Wrapper != "" && config.Wrapper != "layout.templ" {
		if err := validation.ValidatePath(config.Wrapper); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "preview.wrapper",
				Value:   config.Wrapper,
				Message: err.Error(),
				Suggestions: []string{
					"Ensure wrapper template file exists",
					"Use relative path from project root",
					"Remove wrapper setting to use default",
				},
			})
		} else if !strings.HasSuffix(config.Wrapper, ".templ") {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   "preview.wrapper",
				Value:   config.Wrapper,
				Message: "wrapper should be a .templ file",
				Suggestions: []string{
					"Use a .templ file for the wrapper template",
					"Ensure the wrapper exports a component function",
				},
			})
		}
	}
}

func validateDevelopmentConfigDetails(config *DevelopmentConfig, result *ValidationResult) {
	// No specific validation errors for development config
	// Add performance warnings

	if config.HotReload && config.StatePreservation {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "development",
			Value:   "hot_reload + state_preservation",
			Message: "state preservation with hot reload may cause unexpected behavior",
			Suggestions: []string{
				"Disable state preservation if hot reload is unstable",
				"Test thoroughly when both features are enabled",
			},
		})
	}
}

func validatePluginsConfigDetails(config *PluginsConfig, result *ValidationResult) {
	// Validate plugin names
	knownPlugins := []string{"tailwind", "hotreload", "typescript", "postcss"}
	for i, plugin := range config.Enabled {
		if !contains(knownPlugins, plugin) {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   fmt.Sprintf("plugins.enabled[%d]", i),
				Value:   plugin,
				Message: fmt.Sprintf("unknown plugin '%s'", plugin),
				Suggestions: []string{
					"Check plugin name spelling",
					"Ensure plugin is installed",
					"Available built-in plugins: " + strings.Join(knownPlugins, ", "),
				},
			})
		}
	}

	// Validate discovery paths
	for i, path := range config.DiscoveryPaths {
		if err := validation.ValidatePath(path); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("plugins.discovery_paths[%d]", i),
				Value:   path,
				Message: err.Error(),
				Suggestions: []string{
					"Use relative paths from project root",
					"Common plugin paths: './plugins', '~/.templar/plugins'",
				},
			})
		}
	}

	// Check for conflicting plugins
	if contains(config.Enabled, "tailwind") && contains(config.Enabled, "postcss") {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "plugins.enabled",
			Value:   "tailwind + postcss",
			Message: "tailwind and postcss plugins may conflict",
			Suggestions: []string{
				"Use tailwind plugin alone if using Tailwind CSS",
				"Use postcss plugin for custom PostCSS configuration",
				"Test plugin combination thoroughly",
			},
		})
	}
}

// Helper validation functions

func validateHostname(host string) error {
	// Check for dangerous characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(host, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// Check if it's a valid IP address
	if net.ParseIP(host) != nil {
		return nil
	}

	// Check if it's localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}

	// Basic hostname validation
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(host) {
		return fmt.Errorf("invalid hostname format")
	}

	return nil
}

func validateBuildCommand(command string) error {
	// Check for dangerous shell metacharacters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(command, char) {
			return fmt.Errorf("contains potentially dangerous character: %s", char)
		}
	}

	// Ensure command is not empty
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsPattern(slice []string, pattern string) bool {
	for _, s := range slice {
		if strings.Contains(s, pattern) || s == pattern {
			return true
		}
	}
	return false
}
