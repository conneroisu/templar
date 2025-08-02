package errors

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
)

// CLIErrorHandler provides enhanced error handling for CLI commands.
type CLIErrorHandler struct {
	suggestionProvider *ContextualSuggestionProvider
	projectCtx         *ProjectContext
	workingDir         string
}

// NewCLIErrorHandler creates a new CLI error handler with project context.
func NewCLIErrorHandler(workingDir string, cfg *config.Config, reg *registry.ComponentRegistry) *CLIErrorHandler {
	projectCtx := NewProjectContext(workingDir, cfg, reg)
	suggestionProvider := NewContextualSuggestionProvider(projectCtx)

	return &CLIErrorHandler{
		suggestionProvider: suggestionProvider,
		projectCtx:         projectCtx,
		workingDir:         workingDir,
	}
}

// FormatCLIError formats an error with context-aware suggestions for CLI display.
func (ch *CLIErrorHandler) FormatCLIError(err error) string {
	if err == nil {
		return ""
	}

	// Get enhanced suggestions
	suggestions := ch.suggestionProvider.GetEnhancedSuggestions(err)

	// Format the main error
	output := fmt.Sprintf("❌ Error: %s\n", err.Error())

	// Add project context summary if helpful
	if ch.shouldIncludeProjectSummary(err) {
		output += fmt.Sprintf("\n📁 %s\n", ch.projectCtx.GetProjectSummary())
	}

	// Add suggestions
	if len(suggestions) > 0 {
		output += "\n💡 Suggestions:\n"
		for i, suggestion := range suggestions {
			output += ch.formatSuggestion(i+1, suggestion)
		}
	}

	return output
}

// FormatCLIErrorWithHelp formats an error with contextual help for the current command.
func (ch *CLIErrorHandler) FormatCLIErrorWithHelp(err error, command string) string {
	if err == nil {
		return ""
	}

	output := ch.FormatCLIError(err)

	// Add command-specific help
	commandHelp := ch.getCommandContextualHelp(command, err)
	if commandHelp != "" {
		output += fmt.Sprintf("\n🔧 Command Help for '%s':\n%s\n", command, commandHelp)
	}

	return output
}

// formatSuggestion formats a single suggestion for CLI display.
func (ch *CLIErrorHandler) formatSuggestion(index int, suggestion ErrorSuggestion) string {
	var output string

	// Add suggestion title with index
	output += fmt.Sprintf("  %d. %s\n", index, suggestion.Title)

	// Add description if available
	if suggestion.Description != "" {
		output += fmt.Sprintf("     %s\n", suggestion.Description)
	}

	// Add copy-pasteable command if available
	if suggestion.Command != "" {
		// Convert relative paths to absolute paths for better usability
		command := ch.makeCommandCopyPasteable(suggestion.Command)
		output += fmt.Sprintf("     💻 Run: %s\n", command)
	}

	// Add example if available
	if suggestion.Example != "" {
		output += fmt.Sprintf("     📖 Example: %s\n", suggestion.Example)
	}

	output += "\n"

	return output
}

// makeCommandCopyPasteable converts commands to use real, absolute paths where possible.
func (ch *CLIErrorHandler) makeCommandCopyPasteable(command string) string {
	// Replace relative paths with absolute paths
	command = ch.expandRelativePaths(command)

	// Replace placeholder paths with real paths
	command = ch.replacePlaceholderPaths(command)

	// Add current directory context if needed
	if !strings.Contains(command, "/") && !strings.HasPrefix(command, "templar") {
		// This looks like a relative command, add cd context
		return fmt.Sprintf("cd %s && %s", ch.workingDir, command)
	}

	return command
}

// expandRelativePaths converts relative paths in commands to absolute paths.
func (ch *CLIErrorHandler) expandRelativePaths(command string) string {
	// Common patterns to expand - order matters for correct replacement
	patterns := []struct {
		pattern     string
		replacement string
	}{
		{"./components", ch.getActualComponentsPath()},
		{".templar.yml", filepath.Join(ch.workingDir, ".templar.yml")},
		{"templar.yml", filepath.Join(ch.workingDir, "templar.yml")},
		{"components/", ch.getActualComponentsPath() + "/"},
		{"./", ch.workingDir + "/"},
	}

	result := command
	for _, p := range patterns {
		if strings.Contains(result, p.pattern) {
			// Only replace if the path actually exists or it's a config file
			if strings.Contains(p.pattern, "templar.yml") || ch.pathExists(p.replacement) {
				result = strings.ReplaceAll(result, p.pattern, p.replacement)
			}
		}
	}

	return result
}

// replacePlaceholderPaths replaces generic placeholders with real project paths.
func (ch *CLIErrorHandler) replacePlaceholderPaths(command string) string {
	// Get real paths from project analysis
	realPaths := ch.getRealProjectPaths()

	// Replace common placeholders
	replacements := map[string]string{
		"COMPONENTS_DIR": realPaths.ComponentsDir,
		"CONFIG_FILE":    realPaths.ConfigFile,
		"PROJECT_ROOT":   realPaths.ProjectRoot,
		"STATIC_DIR":     realPaths.StaticDir,
		"EXAMPLES_DIR":   realPaths.ExamplesDir,
	}

	result := command
	for placeholder, replacement := range replacements {
		if replacement != "" {
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}

	return result
}

// RealProjectPaths contains actual paths discovered in the project.
type RealProjectPaths struct {
	ProjectRoot   string
	ComponentsDir string
	ConfigFile    string
	StaticDir     string
	ExamplesDir   string
	BuildDir      string
}

// getRealProjectPaths discovers actual paths in the project.
func (ch *CLIErrorHandler) getRealProjectPaths() RealProjectPaths {
	paths := RealProjectPaths{
		ProjectRoot: ch.workingDir,
		ConfigFile:  ch.projectCtx.ConfigPath,
	}

	// Find actual components directory
	if len(ch.projectCtx.ProjectFiles.ComponentFiles) > 0 {
		firstComponent := ch.projectCtx.ProjectFiles.ComponentFiles[0]
		paths.ComponentsDir = filepath.Dir(filepath.Join(ch.workingDir, firstComponent))
	} else {
		paths.ComponentsDir = ch.getActualComponentsPath()
	}

	// Find static directory
	if len(ch.projectCtx.ProjectFiles.StaticDirs) > 0 {
		paths.StaticDir = filepath.Join(ch.workingDir, ch.projectCtx.ProjectFiles.StaticDirs[0])
	}

	// Find examples directory
	if len(ch.projectCtx.ProjectFiles.ExampleDirs) > 0 {
		paths.ExamplesDir = filepath.Join(ch.workingDir, ch.projectCtx.ProjectFiles.ExampleDirs[0])
	}

	return paths
}

// getActualComponentsPath returns the actual components path from config or default.
func (ch *CLIErrorHandler) getActualComponentsPath() string {
	if ch.projectCtx.Config != nil && len(ch.projectCtx.Config.Components.ScanPaths) > 0 {
		firstScanPath := ch.projectCtx.Config.Components.ScanPaths[0]
		if filepath.IsAbs(firstScanPath) {
			return firstScanPath
		}

		return filepath.Join(ch.workingDir, firstScanPath)
	}

	return filepath.Join(ch.workingDir, "components")
}

// pathExists checks if a path exists.
func (ch *CLIErrorHandler) pathExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// shouldIncludeProjectSummary determines if project summary should be included.
func (ch *CLIErrorHandler) shouldIncludeProjectSummary(err error) bool {
	var te *TemplarError
	if !errors.As(err, &te) {
		return false
	}

	// Include summary for errors that benefit from project context
	switch te.Type {
	case ErrorTypeValidation, ErrorTypeBuild, ErrorTypeConfig:
		return true
	case ErrorTypeNetwork:
		return strings.Contains(te.Message, "component") || strings.Contains(te.Message, "file")
	case ErrorTypeSecurity, ErrorTypeIO, ErrorTypeInternal:
		return false
	default:
		return false
	}
}

// getCommandContextualHelp provides command-specific help based on the error.
func (ch *CLIErrorHandler) getCommandContextualHelp(command string, err error) string {
	var te *TemplarError
	if !errors.As(err, &te) {
		return ""
	}

	switch command {
	case "serve":
		return ch.getServeCommandHelp(te)
	case "build":
		return ch.getBuildCommandHelp(te)
	case "list":
		return ch.getListCommandHelp(te)
	case "preview":
		return ch.getPreviewCommandHelp(te)
	case "init":
		return ch.getInitCommandHelp(te)
	default:
		return ch.getGenericCommandHelp(command, te)
	}
}

// getServeCommandHelp provides contextual help for the serve command.
func (ch *CLIErrorHandler) getServeCommandHelp(te *TemplarError) string {
	help := []string{}

	if te.Type == ErrorTypeNetwork {
		help = append(help, "The serve command starts a development server with hot reload.")
		help = append(help, "Common options:")
		help = append(help, "  --port <number>     Use specific port (default: 8080)")
		help = append(help, "  --host <address>    Bind to specific host (default: localhost)")
		help = append(help, "  --no-open          Don't auto-open browser")

		// Add current configuration context
		if ch.projectCtx.Config != nil {
			cfg := ch.projectCtx.Config
			help = append(help, fmt.Sprintf("Current config: port=%d, host=%s", cfg.Server.Port, cfg.Server.Host))
		}
	}

	if te.Type == ErrorTypeBuild || te.Type == ErrorTypeValidation {
		help = append(help, "The serve command needs valid components to serve.")
		help = append(help, "Check that your components directory exists and contains .templ files.")
	}

	return strings.Join(help, "\n")
}

// getBuildCommandHelp provides contextual help for the build command.
func (ch *CLIErrorHandler) getBuildCommandHelp(te *TemplarError) string {
	help := []string{
		"The build command compiles templ components to Go code.",
		"Common options:",
		"  --watch        Watch for changes and rebuild automatically",
		"  --production   Enable production optimizations",
	}

	if te.Type == ErrorTypeBuild {
		help = append(help, "Build errors are often caused by:")
		help = append(help, "• Syntax errors in .templ files")
		help = append(help, "• Missing imports or type definitions")
		help = append(help, "• Invalid templ function signatures")
	}

	return strings.Join(help, "\n")
}

// getListCommandHelp provides contextual help for the list command.
func (ch *CLIErrorHandler) getListCommandHelp(te *TemplarError) string {
	help := []string{
		"The list command shows discovered components in your project.",
		"Common options:",
		"  --format json    Output in JSON format",
		"  --verbose       Show detailed component information",
	}

	if len(ch.projectCtx.ProjectFiles.ComponentFiles) == 0 {
		help = append(help, "No components found. Check your scan paths configuration.")
	} else {
		help = append(help, fmt.Sprintf("Found %d components in your project.", len(ch.projectCtx.ProjectFiles.ComponentFiles)))
	}

	return strings.Join(help, "\n")
}

// getPreviewCommandHelp provides contextual help for the preview command.
func (ch *CLIErrorHandler) getPreviewCommandHelp(te *TemplarError) string {
	help := []string{
		"The preview command opens a component in the browser for testing.",
		"Usage: templar preview <component-name>",
		"Common options:",
		"  --props <json>    Pass props to the component",
		"  --mock <file>     Use mock data from file",
	}

	if te.Type == ErrorTypeValidation && strings.Contains(te.Message, "component") {
		if len(ch.projectCtx.ProjectFiles.ComponentFiles) > 0 {
			componentNames := []string{}
			for _, file := range ch.projectCtx.ProjectFiles.ComponentFiles[:min(5, len(ch.projectCtx.ProjectFiles.ComponentFiles))] {
				name := strings.TrimSuffix(filepath.Base(file), ".templ")
				componentNames = append(componentNames, name)
			}
			help = append(help, "Available components: "+strings.Join(componentNames, ", "))
		}
	}

	return strings.Join(help, "\n")
}

// getInitCommandHelp provides contextual help for the init command.
func (ch *CLIErrorHandler) getInitCommandHelp(te *TemplarError) string {
	help := []string{
		"The init command sets up a new Templar project.",
		"Common options:",
		"  --minimal       Create minimal project structure",
		"  --template <t>  Use specific project template",
	}

	if te.Type == ErrorTypeConfig || te.Type == ErrorTypeIO {
		help = append(help, "Init errors are often caused by:")
		help = append(help, "• Insufficient permissions in target directory")
		help = append(help, "• Directory not empty (use --force to override)")
		help = append(help, "• Missing dependencies")
	}

	return strings.Join(help, "\n")
}

// getGenericCommandHelp provides fallback help for unknown commands.
func (ch *CLIErrorHandler) getGenericCommandHelp(command string, te *TemplarError) string {
	return fmt.Sprintf("For detailed help with the '%s' command, run: templar %s --help", command, command)
}

// CreateEnhancedError creates a new error with contextual suggestions immediately available.
func (ch *CLIErrorHandler) CreateEnhancedError(errType ErrorType, code, message string, cause error) *EnhancedError {
	templErr := &TemplarError{
		Type:        errType,
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: errType == ErrorTypeValidation || errType == ErrorTypeBuild,
	}

	suggestions := ch.suggestionProvider.GetEnhancedSuggestions(templErr)

	return &EnhancedError{
		OriginalError: templErr,
		Title:         message,
		Suggestions:   suggestions,
	}
}

// TrackSuccessfulResolution tracks when an error is successfully resolved.
func (ch *CLIErrorHandler) TrackSuccessfulResolution(err error) {
	ch.projectCtx.MarkErrorResolved(err)
}

// GetProjectInsights returns insights about the current project for help systems.
func (ch *CLIErrorHandler) GetProjectInsights() ProjectInsights {
	return ProjectInsights{
		ComponentCount:       len(ch.projectCtx.ProjectFiles.ComponentFiles),
		ConfigurationExists:  ch.pathExists(ch.projectCtx.ConfigPath),
		RecentErrorPatterns:  ch.projectCtx.GetRecentErrorPatterns(),
		LastAnalysis:         ch.projectCtx.analysisTime,
		HasRecentActivity:    len(ch.projectCtx.ProjectFiles.RecentlyModified) > 0,
		PrimaryComponentsDir: ch.getActualComponentsPath(),
		ProjectSummary:       ch.projectCtx.GetProjectSummary(),
	}
}

// ProjectInsights provides insights about the current project state.
type ProjectInsights struct {
	ComponentCount       int
	ConfigurationExists  bool
	RecentErrorPatterns  map[string]int
	LastAnalysis         time.Time
	HasRecentActivity    bool
	PrimaryComponentsDir string
	ProjectSummary       string
}

// IsHealthy determines if the project appears to be in a healthy state.
func (pi ProjectInsights) IsHealthy() bool {
	// Basic health indicators
	hasComponents := pi.ComponentCount > 0
	hasConfig := pi.ConfigurationExists
	lowErrorRate := len(pi.RecentErrorPatterns) <= 2

	return hasComponents && hasConfig && lowErrorRate
}

// GetHealthSuggestions returns suggestions for improving project health.
func (pi ProjectInsights) GetHealthSuggestions() []string {
	var suggestions []string

	if pi.ComponentCount == 0 {
		suggestions = append(suggestions, "Create some components to get started: templar init")
	}

	if !pi.ConfigurationExists {
		suggestions = append(suggestions, "Create a configuration file: templar config init")
	}

	if len(pi.RecentErrorPatterns) > 3 {
		suggestions = append(suggestions, "Consider reviewing recent errors and addressing recurring issues")
	}

	if !pi.HasRecentActivity {
		suggestions = append(suggestions, "Project appears inactive - consider updating components or configuration")
	}

	return suggestions
}
