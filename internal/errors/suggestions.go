package errors

import (
	"fmt"
	"strings"

	"github.com/conneroisu/templar/internal/registry"
)

// ErrorSuggestion represents a suggestion for fixing an error
type ErrorSuggestion struct {
	Title       string
	Description string
	Command     string
	Example     string
}

// SuggestionContext provides context for generating suggestions
type SuggestionContext struct {
	Registry          *registry.ComponentRegistry
	AvailableCommands []string
	ConfigPath        string
	ComponentsPath    []string
	LastKnownError    string
}

// ComponentNotFoundError generates suggestions for component not found errors
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
				Title:       "Available components",
				Description: "These components are currently available: " + strings.Join(componentNames, ", "),
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

// BuildFailureError generates suggestions for build failures
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

// ServerStartError generates suggestions for server startup failures
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

// ConfigurationError generates suggestions for configuration issues
func ConfigurationError(configError string, configPath string, ctx *SuggestionContext) []ErrorSuggestion {
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

// WebSocketError generates suggestions for WebSocket connection issues
func WebSocketError(err error, ctx *SuggestionContext) []ErrorSuggestion {
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

// FormatSuggestions formats suggestions into a user-friendly string
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

// EnhancedError wraps an error with suggestions
type EnhancedError struct {
	OriginalError error
	Title         string
	Suggestions   []ErrorSuggestion
}

// Error implements the error interface
func (e *EnhancedError) Error() string {
	return FormatSuggestions(e.Title, e.Suggestions)
}

// Unwrap returns the original error
func (e *EnhancedError) Unwrap() error {
	return e.OriginalError
}

// NewEnhancedError creates a new enhanced error with suggestions
func NewEnhancedError(title string, originalError error, suggestions []ErrorSuggestion) *EnhancedError {
	return &EnhancedError{
		OriginalError: originalError,
		Title:         title,
		Suggestions:   suggestions,
	}
}
