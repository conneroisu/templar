package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/conneroisu/templar/internal/registry"
)

// ErrorSuggestion represents a suggestion for fixing an error.
type ErrorSuggestion struct {
	Title       string
	Description string
	Command     string
	Example     string
}

// SuggestionContext provides context for generating suggestions.
// Deprecated: Use ProjectContext for enhanced context-aware suggestions.
type SuggestionContext struct {
	Registry          *registry.ComponentRegistry
	AvailableCommands []string
	ConfigPath        string
	ComponentsPath    []string
	LastKnownError    string
}

// ContextualSuggestionProvider provides intelligent, context-aware error suggestions.
type ContextualSuggestionProvider struct {
	projectCtx *ProjectContext
}

// NewContextualSuggestionProvider creates a new contextual suggestion provider.
func NewContextualSuggestionProvider(projectCtx *ProjectContext) *ContextualSuggestionProvider {
	return &ContextualSuggestionProvider{
		projectCtx: projectCtx,
	}
}

// GetEnhancedSuggestions provides context-aware suggestions with learning resources.
func (csp *ContextualSuggestionProvider) GetEnhancedSuggestions(err error) []ErrorSuggestion {
	var te *TemplarError
	if !errors.As(err, &te) {
		return csp.getGenericEnhancedSuggestions(err)
	}

	// Track this error for pattern analysis
	context := csp.extractErrorContext(te)
	csp.projectCtx.AddRecentError(err, context)

	// Get contextual suggestions based on error type
	suggestions := csp.projectCtx.GetContextualSuggestions(
		csp.categorizeError(te),
		context,
	)

	// Add learning resources
	suggestions = append(suggestions, csp.getLearningResources(te)...)

	// Add configuration-aware suggestions
	suggestions = append(suggestions, csp.getConfigAwareSuggestions(te)...)

	// Add pattern-based suggestions from recent errors
	suggestions = append(suggestions, csp.getPatternBasedSuggestions()...)

	return csp.limitAndPrioritizeSuggestions(suggestions)
}

// GetComponentNotFoundSuggestions provides enhanced component not found suggestions.
func (csp *ContextualSuggestionProvider) GetComponentNotFoundSuggestions(componentName string) []ErrorSuggestion {
	context := map[string]interface{}{
		"component":  componentName,
		"error_type": "component_not_found",
	}

	return csp.projectCtx.GetContextualSuggestions("component_not_found", context)
}

// GetBuildFailureSuggestions provides enhanced build failure suggestions.
func (csp *ContextualSuggestionProvider) GetBuildFailureSuggestions(buildOutput string, component string) []ErrorSuggestion {
	context := map[string]interface{}{
		"build_output": buildOutput,
		"component":    component,
		"error_type":   "build_failed",
	}

	suggestions := csp.projectCtx.GetContextualSuggestions("build_failed", context)

	// Add build-specific learning resources
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Learn about templ syntax",
		Description: "Official documentation for templ component syntax",
		Command:     "xdg-open https://templ.guide/syntax-and-usage",
		Example:     "templ ComponentName(props Type) { <div>Content</div> }",
	})

	return suggestions
}

// GetServerStartSuggestions provides enhanced server startup suggestions.
func (csp *ContextualSuggestionProvider) GetServerStartSuggestions(err error, port int) []ErrorSuggestion {
	context := map[string]interface{}{
		"port":          port,
		"error_type":    "server_start",
		"error_message": err.Error(),
	}

	suggestions := csp.projectCtx.GetContextualSuggestions("server_start", context)

	// Add server-specific learning resources
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Learn about development server",
		Description: "Documentation for Templar development server features",
		Command:     "templar serve --help",
		Example:     "templar serve --port 3000 --no-open",
	})

	return suggestions
}

// GetConfigurationSuggestions provides configuration-aware suggestions.
func (csp *ContextualSuggestionProvider) GetConfigurationSuggestions(configError string) []ErrorSuggestion {
	context := map[string]interface{}{
		"config_error": configError,
		"error_type":   "config_error",
		"config_path":  csp.projectCtx.ConfigPath,
	}

	suggestions := csp.projectCtx.GetContextualSuggestions("config_error", context)

	// Add configuration learning resources
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Learn about configuration",
		Description: "Complete guide to Templar configuration options",
		Command:     "cat CONFIGURATION.md",
		Example:     "server:\n  port: 8080\ncomponents:\n  scan_paths: [\"./components\"]",
	})

	return suggestions
}

// extractErrorContext extracts relevant context from a TemplarError.
func (csp *ContextualSuggestionProvider) extractErrorContext(te *TemplarError) map[string]interface{} {
	context := make(map[string]interface{})

	context["error_type"] = string(te.Type)
	context["error_code"] = te.Code
	context["component"] = te.Component
	context["file_path"] = te.FilePath
	context["line"] = te.Line
	context["column"] = te.Column
	context["recoverable"] = te.Recoverable

	// Merge in existing context
	for k, v := range te.Context {
		context[k] = v
	}

	return context
}

// categorizeError categorizes errors for suggestion lookup.
func (csp *ContextualSuggestionProvider) categorizeError(te *TemplarError) string {
	errMsg := strings.ToLower(te.Message)

	switch {
	case strings.Contains(errMsg, "component not found") || te.Code == ErrCodeComponentNotFound:
		return "component_not_found"
	case strings.Contains(errMsg, "build failed") || te.Code == ErrCodeBuildFailed:
		return "build_failed"
	case strings.Contains(errMsg, "config") || te.Type == ErrorTypeConfig:
		return "config_error"
	case strings.Contains(errMsg, "server") || strings.Contains(errMsg, "port"):
		return "server_start"
	case strings.Contains(errMsg, "path") || te.Code == ErrCodeInvalidPath:
		return "path_error"
	default:
		return "general"
	}
}

// getLearningResources adds educational content to suggestions.
func (csp *ContextualSuggestionProvider) getLearningResources(te *TemplarError) []ErrorSuggestion {
	var suggestions []ErrorSuggestion

	// Add learning resources based on error type
	switch te.Type {
	case ErrorTypeBuild:
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "📚 Learn: templ Component Basics",
			Description: "Understand the fundamentals of creating templ components",
			Command:     "templar tutorial build",
			Example:     "Visit https://templ.guide for comprehensive documentation",
		})

	case ErrorTypeValidation:
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "📚 Learn: CLI Command Usage",
			Description: "Master Templar CLI commands and flags",
			Command:     "templar --help",
			Example:     "Each command has detailed help: templar serve --help",
		})

	case ErrorTypeConfig:
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "📚 Learn: Configuration Best Practices",
			Description: "Optimize your Templar configuration for your workflow",
			Command:     "cat docs/CONFIGURATION.md",
			Example:     "Use environment variables for different deployment environments",
		})

	case ErrorTypeNetwork:
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "📚 Learn: Development Server Features",
			Description: "Maximize productivity with hot reload and live preview",
			Command:     "templar tutorial server",
			Example:     "WebSocket connections provide real-time updates",
		})

	case ErrorTypeSecurity, ErrorTypeIO, ErrorTypeInternal:
		// No specific learning resources for these error types yet
	}

	// Add general troubleshooting resource
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "🔧 Troubleshooting Guide",
		Description: "Common issues and solutions for Templar development",
		Command:     "cat docs/TROUBLESHOOTING.md",
		Example:     "Step-by-step debugging for common scenarios",
	})

	return suggestions
}

// getConfigAwareSuggestions provides suggestions based on current configuration.
func (csp *ContextualSuggestionProvider) getConfigAwareSuggestions(te *TemplarError) []ErrorSuggestion {
	var suggestions []ErrorSuggestion

	if csp.projectCtx.Config == nil {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "⚙️ Initialize Configuration",
			Description: "Create a configuration file to customize Templar behavior",
			Command:     "templar config init > " + csp.projectCtx.ConfigPath,
			Example:     "Configuration enables custom scan paths, server settings, and more",
		})

		return suggestions
	}

	cfg := csp.projectCtx.Config

	// Server configuration suggestions
	if te.Type == ErrorTypeNetwork {
		suggestions = append(suggestions, ErrorSuggestion{
			Title: "⚙️ Current Server Config",
			Description: fmt.Sprintf("Port: %d, Host: %s, Open Browser: %t",
				cfg.Server.Port, cfg.Server.Host, cfg.Server.Open),
			Command: fmt.Sprintf("templar serve --port %d", cfg.Server.Port+1),
			Example: "Modify server settings in your configuration file",
		})
	}

	// Component scanning suggestions
	if te.Type == ErrorTypeValidation && strings.Contains(te.Message, "component") {
		scanPaths := cfg.Components.ScanPaths
		if len(scanPaths) == 0 {
			scanPaths = []string{"./components"}
		}

		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "⚙️ Current Scan Paths",
			Description: "Scanning: " + strings.Join(scanPaths, ", "),
			Command:     fmt.Sprintf("find %s -name '*.templ' -type f", strings.Join(scanPaths, " ")),
			Example:     "Add more directories to scan_paths in configuration",
		})
	}

	// Build configuration suggestions
	if te.Type == ErrorTypeBuild {
		suggestions = append(suggestions, ErrorSuggestion{
			Title: "⚙️ Build Configuration",
			Description: fmt.Sprintf("Build command: %s, Cache dir: %s",
				cfg.Build.Command, cfg.Build.CacheDir),
			Command: cfg.Build.Command,
			Example: "Customize build behavior in the build section of your config",
		})
	}

	return suggestions
}

// getPatternBasedSuggestions analyzes recent errors for patterns.
func (csp *ContextualSuggestionProvider) getPatternBasedSuggestions() []ErrorSuggestion {
	patterns := csp.projectCtx.GetRecentErrorPatterns()
	var suggestions []ErrorSuggestion

	// Suggest solutions based on error patterns
	for pattern, count := range patterns {
		if count >= 3 { // Only suggest if pattern repeats 3+ times
			switch pattern {
			case "component_not_found":
				suggestions = append(suggestions, ErrorSuggestion{
					Title:       "🔄 Recurring Issue: Component Discovery",
					Description: fmt.Sprintf("You've had %d component not found errors recently", count),
					Command:     "templar list --verbose",
					Example:     "Consider updating your scan paths configuration",
				})

			case "build_failed":
				suggestions = append(suggestions, ErrorSuggestion{
					Title:       "🔄 Recurring Issue: Build Failures",
					Description: fmt.Sprintf("You've had %d build failures recently", count),
					Command:     "templ fmt ./...",
					Example:     "Run format and syntax check on all components",
				})

			case "port_in_use":
				suggestions = append(suggestions, ErrorSuggestion{
					Title:       "🔄 Recurring Issue: Port Conflicts",
					Description: fmt.Sprintf("You've had %d port conflicts recently", count),
					Command:     "templar serve --port 0", // Use random available port
					Example:     "Use --port 0 to automatically find an available port",
				})
			}
		}
	}

	return suggestions
}

// limitAndPrioritizeSuggestions limits suggestions and orders them by priority.
func (csp *ContextualSuggestionProvider) limitAndPrioritizeSuggestions(suggestions []ErrorSuggestion) []ErrorSuggestion {
	// Priority order: immediate fixes, configuration, learning resources, patterns
	var prioritized []ErrorSuggestion
	var learning []ErrorSuggestion
	var config []ErrorSuggestion
	var patterns []ErrorSuggestion
	var other []ErrorSuggestion

	for _, suggestion := range suggestions {
		title := suggestion.Title
		switch {
		case strings.Contains(title, "📚"):
			learning = append(learning, suggestion)
		case strings.Contains(title, "⚙️"):
			config = append(config, suggestion)
		case strings.Contains(title, "🔄"):
			patterns = append(patterns, suggestion)
		default:
			other = append(other, suggestion)
		}
	}

	// Combine in priority order, limiting each category
	prioritized = append(prioritized, other[:min(4, len(other))]...)
	prioritized = append(prioritized, config[:min(2, len(config))]...)
	prioritized = append(prioritized, learning[:min(2, len(learning))]...)
	prioritized = append(prioritized, patterns[:min(1, len(patterns))]...)

	// Overall limit
	if len(prioritized) > 8 {
		prioritized = prioritized[:8]
	}

	return prioritized
}

// getGenericEnhancedSuggestions provides fallback suggestions for non-TemplarError types.
func (csp *ContextualSuggestionProvider) getGenericEnhancedSuggestions(err error) []ErrorSuggestion {
	return []ErrorSuggestion{
		{
			Title:       "Check project status",
			Description: "Verify your project structure and configuration",
			Command:     "templar list --verbose",
		},
		{
			Title:       "📚 Learn: Getting Started",
			Description: "Complete guide to using Templar effectively",
			Command:     "templar tutorial",
			Example:     "Interactive tutorials for common workflows",
		},
	}
}

// ComponentNotFoundError generates suggestions for component not found errors.
func ComponentNotFoundError(componentName string, ctx *SuggestionContext) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{
		{
			Title:       "Check component file exists",
			Description: "Verify the component file exists in one of the scanned directories",
			Command:     "ls -la components/ | grep -i " + strings.ToLower(componentName),
			Example:     "components/" + strings.ToLower(componentName) + ".templ",
		},
		{
			Title:       "Verify component function name",
			Description: "Ensure the templ function name matches the component name exactly",
			Example:     "templ " + componentName + "(props Type) { ... }",
		},
		{
			Title:       "List all discovered components",
			Description: "See what components Templar has found",
			Command:     "templar list",
		},
		{
			Title:       "Check scan paths configuration",
			Description: "Verify your .templar.yml scan_paths include the component directory",
			Command:     "cat " + ctx.ConfigPath,
			Example:     "components:\n  scan_paths:\n    - \"./components\"\n    - \"./views\"",
		},
	}

	// Add available components if registry is available
	if ctx.Registry != nil {
		components := ctx.Registry.GetAll()
		if len(components) > 0 {
			var componentNames []string
			for _, comp := range components {
				componentNames = append(componentNames, comp.Name)
			}

			suggestions = append(suggestions, ErrorSuggestion{
				Title: "Available components",
				Description: "These components are currently available: " + strings.Join(
					componentNames,
					", ",
				),
			})

			// Suggest similar component names
			for _, comp := range components {
				if strings.Contains(strings.ToLower(comp.Name), strings.ToLower(componentName)) ||
					strings.Contains(strings.ToLower(componentName), strings.ToLower(comp.Name)) {
					suggestions = append(suggestions, ErrorSuggestion{
						Title:       "Did you mean '" + comp.Name + "'?",
						Description: "Similar component found",
						Command:     "templar preview " + comp.Name,
					})

					break
				}
			}
		}
	}

	return suggestions
}

// BuildFailureError generates suggestions for build failures.
func BuildFailureError(buildOutput string, ctx *SuggestionContext) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{
		{
			Title:       "Check component syntax",
			Description: "Verify your templ component has valid syntax",
			Command:     "templ generate",
		},
		{
			Title:       "Review build output",
			Description: "Check the full error message for specific syntax issues",
		},
	}

	// Analyze build output for common issues
	output := strings.ToLower(buildOutput)

	if strings.Contains(output, "syntax error") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Fix syntax error",
			Description: "There's a syntax error in your templ file",
			Example:     "Check for missing braces, semicolons, or invalid Go syntax",
		})
	}

	if strings.Contains(output, "undefined") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Check imports and types",
			Description: "Ensure all types and functions are properly imported",
			Example:     "import \"your-project/types\"",
		})
	}

	if strings.Contains(output, "package") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Verify package declaration",
			Description: "Ensure your templ file has the correct package declaration",
			Example:     "package components",
		})
	}

	return suggestions
}

// ServerStartError generates suggestions for server startup failures.
func ServerStartError(err error, port int, ctx *SuggestionContext) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	errStr := err.Error()

	if strings.Contains(errStr, "address already in use") || strings.Contains(errStr, "bind") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Port already in use",
			Description: fmt.Sprintf("Port %d is already being used by another process", port),
			Command:     fmt.Sprintf("lsof -i :%d", port),
		})

		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Use a different port",
			Description: "Start the server on a different port",
			Command:     fmt.Sprintf("templar serve --port %d", port+1000),
		})

		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Kill the process using the port",
			Description: "Stop the process that's using the port",
			Command:     fmt.Sprintf("lsof -ti :%d | xargs kill", port),
		})
	}

	if strings.Contains(errStr, "permission denied") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Permission denied",
			Description: "You don't have permission to bind to this port",
		})

		if port < 1024 {
			suggestions = append(suggestions, ErrorSuggestion{
				Title:       "Use unprivileged port",
				Description: "Ports below 1024 require root privileges",
				Command:     "templar serve --port 8080",
			})
		}
	}

	return suggestions
}

// ConfigurationErrorSuggestions generates suggestions for configuration issues.
func ConfigurationErrorSuggestions(
	configError string,
	configPath string,
	ctx *SuggestionContext,
) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{
		{
			Title:       "Check configuration file",
			Description: "Verify your .templar.yml file exists and has valid syntax",
			Command:     "cat " + configPath,
		},
		{
			Title:       "Validate configuration",
			Description: "Use the config validate command to check for issues",
			Command:     "templar config validate",
		},
	}

	if strings.Contains(configError, "yaml") || strings.Contains(configError, "unmarshal") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Fix YAML syntax",
			Description: "There's a syntax error in your YAML configuration",
			Example:     "Use proper indentation and avoid tabs",
		})
	}

	if strings.Contains(configError, "path") || strings.Contains(configError, "directory") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Check directory paths",
			Description: "Verify all paths in your configuration exist",
			Command:     "ls -la",
		})
	}

	return suggestions
}

// WebSocketErrorSuggestions generates suggestions for WebSocket connection issues.
func WebSocketErrorSuggestions(err error, ctx *SuggestionContext) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{
		{
			Title:       "Check browser console",
			Description: "Look for WebSocket errors in the browser's developer console",
		},
		{
			Title:       "Verify hot reload is enabled",
			Description: "Ensure hot_reload is set to true in your configuration",
			Example:     "development:\n  hot_reload: true",
		},
		{
			Title:       "Check firewall settings",
			Description: "Ensure your firewall isn't blocking WebSocket connections",
		},
	}

	errStr := err.Error()

	if strings.Contains(errStr, "origin") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Origin validation failed",
			Description: "The WebSocket connection was rejected due to origin validation",
		})
	}

	if strings.Contains(errStr, "upgrade") {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "WebSocket upgrade failed",
			Description: "The HTTP to WebSocket upgrade failed",
		})
	}

	return suggestions
}

// FormatSuggestions formats suggestions into a user-friendly string.
func FormatSuggestions(title string, suggestions []ErrorSuggestion) string {
	if len(suggestions) == 0 {
		return title
	}

	var output strings.Builder
	output.WriteString(title + "\n\n")
	output.WriteString("Suggestions:\n")

	for i, suggestion := range suggestions {
		output.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion.Title))
		if suggestion.Description != "" {
			output.WriteString(fmt.Sprintf("     %s\n", suggestion.Description))
		}
		if suggestion.Command != "" {
			output.WriteString(fmt.Sprintf("     Run: %s\n", suggestion.Command))
		}
		if suggestion.Example != "" {
			output.WriteString(fmt.Sprintf("     Example: %s\n", suggestion.Example))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// EnhancedError wraps an error with suggestions.
type EnhancedError struct {
	OriginalError error
	Title         string
	Suggestions   []ErrorSuggestion
}

// Error implements the error interface.
func (e *EnhancedError) Error() string {
	return FormatSuggestions(e.Title, e.Suggestions)
}

// Unwrap returns the original error.
func (e *EnhancedError) Unwrap() error {
	return e.OriginalError
}

// NewEnhancedError creates a new enhanced error with suggestions.
func NewEnhancedError(
	title string,
	originalError error,
	suggestions []ErrorSuggestion,
) *EnhancedError {
	return &EnhancedError{
		OriginalError: originalError,
		Title:         title,
		Suggestions:   suggestions,
	}
}
