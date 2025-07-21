package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConfigWizard provides an interactive setup experience for new projects
type ConfigWizard struct {
	reader *bufio.Reader
	config *Config
}

// NewConfigWizard creates a new configuration wizard
func NewConfigWizard() *ConfigWizard {
	return &ConfigWizard{
		reader: bufio.NewReader(os.Stdin),
		config: &Config{},
	}
}

// Run executes the interactive configuration wizard
func (w *ConfigWizard) Run() (*Config, error) {
	fmt.Println("🧙 Templar Configuration Wizard")
	fmt.Println("================================")
	fmt.Println("This wizard will help you set up your Templar project configuration.")
	fmt.Println()

	// Server configuration
	if err := w.configureServer(); err != nil {
		return nil, fmt.Errorf("server configuration failed: %w", err)
	}

	// Components configuration
	if err := w.configureComponents(); err != nil {
		return nil, fmt.Errorf("components configuration failed: %w", err)
	}

	// Build configuration
	if err := w.configureBuild(); err != nil {
		return nil, fmt.Errorf("build configuration failed: %w", err)
	}

	// Development configuration
	if err := w.configureDevelopment(); err != nil {
		return nil, fmt.Errorf("development configuration failed: %w", err)
	}

	// Preview configuration
	if err := w.configurePreview(); err != nil {
		return nil, fmt.Errorf("preview configuration failed: %w", err)
	}

	// Plugins configuration
	if err := w.configurePlugins(); err != nil {
		return nil, fmt.Errorf("plugins configuration failed: %w", err)
	}

	// Validate the final configuration
	if err := validateConfig(w.config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Configuration completed successfully!")
	return w.config, nil
}

func (w *ConfigWizard) configureServer() error {
	fmt.Println("🌐 Server Configuration")
	fmt.Println("----------------------")

	// Port configuration
	port, err := w.askInt("Server port", 8080, 1, 65535)
	if err != nil {
		return err
	}
	w.config.Server.Port = port

	// Host configuration
	host := w.askString("Server host", "localhost")
	w.config.Server.Host = host

	// Auto-open browser
	w.config.Server.Open = w.askBool("Auto-open browser on start", true)

	// Environment
	env := w.askChoice("Environment", []string{"development", "production"}, "development")
	w.config.Server.Environment = env

	// Middleware
	if w.askBool("Enable CORS middleware", true) {
		w.config.Server.Middleware = append(w.config.Server.Middleware, "cors")
	}
	if w.askBool("Enable request logging", true) {
		w.config.Server.Middleware = append(w.config.Server.Middleware, "logger")
	}

	fmt.Println()
	return nil
}

func (w *ConfigWizard) configureComponents() error {
	fmt.Println("🔍 Components Configuration")
	fmt.Println("---------------------------")

	// Scan paths
	fmt.Println("Component scan paths (directories to search for .templ files):")
	scanPaths := []string{}

	defaultPaths := []string{"./components", "./views", "./examples"}
	for _, path := range defaultPaths {
		if w.askBool(fmt.Sprintf("Include %s", path), true) {
			scanPaths = append(scanPaths, path)
		}
	}

	// Allow custom paths
	for {
		if !w.askBool("Add custom scan path", false) {
			break
		}
		customPath := w.askString("Custom scan path", "")
		if customPath != "" {
			scanPaths = append(scanPaths, customPath)
		}
	}

	w.config.Components.ScanPaths = scanPaths

	// Exclude patterns
	excludePatterns := []string{}
	defaultExcludes := []string{"*_test.templ", "*.bak", "*.example.templ"}

	fmt.Println("File exclusion patterns:")
	for _, pattern := range defaultExcludes {
		if w.askBool(fmt.Sprintf("Exclude %s", pattern), true) {
			excludePatterns = append(excludePatterns, pattern)
		}
	}

	// Allow custom exclusion patterns
	for {
		if !w.askBool("Add custom exclusion pattern", false) {
			break
		}
		customPattern := w.askString("Custom exclusion pattern", "")
		if customPattern != "" {
			excludePatterns = append(excludePatterns, customPattern)
		}
	}

	w.config.Components.ExcludePatterns = excludePatterns
	fmt.Println()
	return nil
}

func (w *ConfigWizard) configureBuild() error {
	fmt.Println("🔨 Build Configuration")
	fmt.Println("----------------------")

	// Build command
	w.config.Build.Command = w.askString("Build command", "templ generate")

	// Watch patterns
	watchPatterns := []string{}
	defaultWatchPatterns := []string{"**/*.templ", "**/*.go"}

	fmt.Println("File watch patterns (for auto-rebuild):")
	for _, pattern := range defaultWatchPatterns {
		if w.askBool(fmt.Sprintf("Watch %s", pattern), true) {
			watchPatterns = append(watchPatterns, pattern)
		}
	}

	w.config.Build.Watch = watchPatterns

	// Ignore patterns
	ignorePatterns := []string{}
	defaultIgnorePatterns := []string{"node_modules", ".git", "*_test.go", "vendor/**"}

	fmt.Println("Build ignore patterns:")
	for _, pattern := range defaultIgnorePatterns {
		if w.askBool(fmt.Sprintf("Ignore %s", pattern), true) {
			ignorePatterns = append(ignorePatterns, pattern)
		}
	}

	w.config.Build.Ignore = ignorePatterns

	// Cache directory
	w.config.Build.CacheDir = w.askString("Build cache directory", ".templar/cache")

	fmt.Println()
	return nil
}

func (w *ConfigWizard) configureDevelopment() error {
	fmt.Println("🚀 Development Configuration")
	fmt.Println("----------------------------")

	w.config.Development.HotReload = w.askBool("Enable hot reload", true)
	w.config.Development.CSSInjection = w.askBool("Enable CSS injection", true)
	w.config.Development.StatePreservation = w.askBool("Enable state preservation", false)
	w.config.Development.ErrorOverlay = w.askBool("Enable error overlay", true)

	fmt.Println()
	return nil
}

func (w *ConfigWizard) configurePreview() error {
	fmt.Println("👁 Preview Configuration")
	fmt.Println("------------------------")

	// Mock data strategy
	mockStrategy := w.askChoice("Mock data strategy", []string{"auto", "manual", "none"}, "auto")
	if mockStrategy == "manual" {
		mockPath := w.askString("Mock data directory", "./mocks")
		w.config.Preview.MockData = mockPath
	} else {
		w.config.Preview.MockData = mockStrategy
	}

	// Preview wrapper
	if w.askBool("Use custom preview wrapper", false) {
		wrapper := w.askString("Preview wrapper template", "./preview/wrapper.templ")
		w.config.Preview.Wrapper = wrapper
	}

	w.config.Preview.AutoProps = w.askBool("Enable auto props generation", true)

	fmt.Println()
	return nil
}

func (w *ConfigWizard) configurePlugins() error {
	fmt.Println("🔌 Plugins Configuration")
	fmt.Println("------------------------")

	// Built-in plugins
	builtinPlugins := []string{"tailwind", "hotreload"}
	enabledPlugins := []string{}

	fmt.Println("Built-in plugins:")
	for _, plugin := range builtinPlugins {
		if w.askBool(fmt.Sprintf("Enable %s plugin", plugin), true) {
			enabledPlugins = append(enabledPlugins, plugin)
		}
	}

	w.config.Plugins.Enabled = enabledPlugins

	// Plugin discovery paths
	discoveryPaths := []string{"./plugins"}
	if w.askBool("Include global plugins (~/.templar/plugins)", false) {
		discoveryPaths = append(discoveryPaths, "~/.templar/plugins")
	}

	w.config.Plugins.DiscoveryPaths = discoveryPaths
	w.config.Plugins.Configurations = make(map[string]PluginConfigMap)

	fmt.Println()
	return nil
}

// Helper methods for user interaction

func (w *ConfigWizard) askString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := w.reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

func (w *ConfigWizard) askInt(prompt string, defaultValue, min, max int) (int, error) {
	for {
		fmt.Printf("%s [%d]: ", prompt, defaultValue)

		input, err := w.reader.ReadString('\n')
		if err != nil {
			return defaultValue, nil
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return defaultValue, nil
		}

		value, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("❌ Invalid number. Please enter a number between %d and %d.\n", min, max)
			continue
		}

		if value < min || value > max {
			fmt.Printf("❌ Number out of range. Please enter a number between %d and %d.\n", min, max)
			continue
		}

		return value, nil
	}
}

func (w *ConfigWizard) askBool(prompt string, defaultValue bool) bool {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	input, err := w.reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes" || input == "true"
}

func (w *ConfigWizard) askChoice(prompt string, choices []string, defaultValue string) string {
	for {
		fmt.Printf("%s [%s] (options: %s): ", prompt, defaultValue, strings.Join(choices, ", "))

		input, err := w.reader.ReadString('\n')
		if err != nil {
			return defaultValue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return defaultValue
		}

		// Check if input is valid choice
		for _, choice := range choices {
			if strings.ToLower(input) == strings.ToLower(choice) {
				return choice
			}
		}

		fmt.Printf("❌ Invalid choice. Please select from: %s\n", strings.Join(choices, ", "))
	}
}

// WriteConfigFile writes the configuration to a YAML file
func (w *ConfigWizard) WriteConfigFile(filename string) error {
	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		overwrite := w.askBool(fmt.Sprintf("Configuration file %s already exists. Overwrite", filename), false)
		if !overwrite {
			return fmt.Errorf("configuration file already exists")
		}
	}

	// Generate YAML content
	content := w.generateYAMLConfig()

	// Write to file
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration saved to %s\n", filename)
	return nil
}

func (w *ConfigWizard) generateYAMLConfig() string {
	var builder strings.Builder

	builder.WriteString("# Templar configuration file\n")
	builder.WriteString("# Generated by Templar Configuration Wizard\n\n")

	// Server configuration
	builder.WriteString("server:\n")
	builder.WriteString(fmt.Sprintf("  port: %d\n", w.config.Server.Port))
	builder.WriteString(fmt.Sprintf("  host: %s\n", w.config.Server.Host))
	builder.WriteString(fmt.Sprintf("  open: %t\n", w.config.Server.Open))
	builder.WriteString(fmt.Sprintf("  environment: %s\n", w.config.Server.Environment))
	if len(w.config.Server.Middleware) > 0 {
		builder.WriteString("  middleware:\n")
		for _, middleware := range w.config.Server.Middleware {
			builder.WriteString(fmt.Sprintf("    - %s\n", middleware))
		}
	}
	builder.WriteString("\n")

	// Build configuration
	builder.WriteString("build:\n")
	builder.WriteString(fmt.Sprintf("  command: \"%s\"\n", w.config.Build.Command))
	if len(w.config.Build.Watch) > 0 {
		builder.WriteString("  watch:\n")
		for _, pattern := range w.config.Build.Watch {
			builder.WriteString(fmt.Sprintf("    - \"%s\"\n", pattern))
		}
	}
	if len(w.config.Build.Ignore) > 0 {
		builder.WriteString("  ignore:\n")
		for _, pattern := range w.config.Build.Ignore {
			builder.WriteString(fmt.Sprintf("    - \"%s\"\n", pattern))
		}
	}
	builder.WriteString(fmt.Sprintf("  cache_dir: \"%s\"\n", w.config.Build.CacheDir))
	builder.WriteString("\n")

	// Preview configuration
	builder.WriteString("preview:\n")
	builder.WriteString(fmt.Sprintf("  mock_data: \"%s\"\n", w.config.Preview.MockData))
	if w.config.Preview.Wrapper != "" {
		builder.WriteString(fmt.Sprintf("  wrapper: \"%s\"\n", w.config.Preview.Wrapper))
	}
	builder.WriteString(fmt.Sprintf("  auto_props: %t\n", w.config.Preview.AutoProps))
	builder.WriteString("\n")

	// Components configuration
	builder.WriteString("components:\n")
	if len(w.config.Components.ScanPaths) > 0 {
		builder.WriteString("  scan_paths:\n")
		for _, path := range w.config.Components.ScanPaths {
			builder.WriteString(fmt.Sprintf("    - \"%s\"\n", path))
		}
	}
	if len(w.config.Components.ExcludePatterns) > 0 {
		builder.WriteString("  exclude_patterns:\n")
		for _, pattern := range w.config.Components.ExcludePatterns {
			builder.WriteString(fmt.Sprintf("    - \"%s\"\n", pattern))
		}
	}
	builder.WriteString("\n")

	// Development configuration
	builder.WriteString("development:\n")
	builder.WriteString(fmt.Sprintf("  hot_reload: %t\n", w.config.Development.HotReload))
	builder.WriteString(fmt.Sprintf("  css_injection: %t\n", w.config.Development.CSSInjection))
	builder.WriteString(fmt.Sprintf("  state_preservation: %t\n", w.config.Development.StatePreservation))
	builder.WriteString(fmt.Sprintf("  error_overlay: %t\n", w.config.Development.ErrorOverlay))
	builder.WriteString("\n")

	// Plugins configuration
	if len(w.config.Plugins.Enabled) > 0 || len(w.config.Plugins.DiscoveryPaths) > 0 {
		builder.WriteString("plugins:\n")
		if len(w.config.Plugins.Enabled) > 0 {
			builder.WriteString("  enabled:\n")
			for _, plugin := range w.config.Plugins.Enabled {
				builder.WriteString(fmt.Sprintf("    - %s\n", plugin))
			}
		}
		if len(w.config.Plugins.DiscoveryPaths) > 0 {
			builder.WriteString("  discovery_paths:\n")
			for _, path := range w.config.Plugins.DiscoveryPaths {
				builder.WriteString(fmt.Sprintf("    - \"%s\"\n", path))
			}
		}
	}

	return builder.String()
}
