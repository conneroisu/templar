package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigWizard provides an interactive setup experience for new projects
type ConfigWizard struct {
	reader            *bufio.Reader
	config            *Config
	projectDir        string
	detectedStructure *ProjectStructure
}

// ProjectStructure represents detected project characteristics
type ProjectStructure struct {
	HasGoMod         bool
	HasNodeModules   bool
	HasTailwindCSS   bool
	HasTypeScript    bool
	HasExistingTempl bool
	ProjectType      string // "web", "api", "fullstack", "library"
	ComponentDirs    []string
}

// NewConfigWizard creates a new configuration wizard
func NewConfigWizard() *ConfigWizard {
	return &ConfigWizard{
		reader: bufio.NewReader(os.Stdin),
		config: &Config{},
	}
}

// NewConfigWizardWithProjectDir creates a new configuration wizard for a specific project directory
func NewConfigWizardWithProjectDir(projectDir string) *ConfigWizard {
	wizard := &ConfigWizard{
		reader:     bufio.NewReader(os.Stdin),
		config:     &Config{},
		projectDir: projectDir,
	}
	wizard.detectProjectStructure()
	return wizard
}

// Run executes the interactive configuration wizard
func (w *ConfigWizard) Run() (*Config, error) {
	fmt.Println("🧙 Templar Configuration Wizard")
	fmt.Println("================================")
	fmt.Println("This wizard will help you set up your Templar project configuration.")

	// Show detected project structure if available
	if w.detectedStructure != nil {
		fmt.Println()
		fmt.Printf("🔍 Detected project type: %s\n", w.detectedStructure.ProjectType)

		if len(w.detectedStructure.ComponentDirs) > 0 {
			fmt.Printf(
				"📁 Found existing directories: %s\n",
				strings.Join(w.detectedStructure.ComponentDirs, ", "),
			)
		}

		if w.detectedStructure.HasExistingTempl {
			fmt.Println("✨ Found existing .templ files")
		}

		fmt.Println("💡 Smart defaults will be applied based on your project structure.")
	}

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

	// Monitoring configuration
	if err := w.configureMonitoring(); err != nil {
		return nil, fmt.Errorf("monitoring configuration failed: %w", err)
	}

	// Apply full defaults to ensure all required fields are set
	loadDefaults(w.config)

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

	// Scan paths with smart defaults
	fmt.Println("Component scan paths (directories to search for .templ files):")
	scanPaths := []string{}

	// Use detected component directories if available
	if w.detectedStructure != nil && len(w.detectedStructure.ComponentDirs) > 0 {
		fmt.Printf(
			"Using detected directories: %s\n",
			strings.Join(w.detectedStructure.ComponentDirs, ", "),
		)
		for _, path := range w.detectedStructure.ComponentDirs {
			if w.askBool(fmt.Sprintf("Include %s", path), true) {
				scanPaths = append(scanPaths, path)
			}
		}
	} else {
		// Use default paths
		defaultPaths := []string{"./components", "./views", "./examples"}
		for _, path := range defaultPaths {
			if w.askBool(fmt.Sprintf("Include %s", path), true) {
				scanPaths = append(scanPaths, path)
			}
		}
	}

	// Allow custom paths
	for {
		if !w.askBool("Add custom scan path", false) {
			break
		}
		customPath := w.askString("Custom scan path", "")
		if customPath != "" && customPath != "y" && customPath != "n" {
			scanPaths = append(scanPaths, customPath)
		} else if customPath == "" || customPath == "n" || customPath == "no" {
			break // Exit loop if user enters nothing or no
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
		if customPattern != "" && customPattern != "y" && customPattern != "n" {
			excludePatterns = append(excludePatterns, customPattern)
		} else if customPattern == "" || customPattern == "n" || customPattern == "no" {
			break // Exit loop if user enters nothing or no
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

	// Built-in plugins with smart defaults based on project structure
	enabledPlugins := []string{}

	// Always suggest hotreload
	if w.askBool("Enable hotreload plugin (recommended)", true) {
		enabledPlugins = append(enabledPlugins, "hotreload")
	}

	// Suggest tailwind if detected or for web projects
	tailwindDefault := false
	if w.detectedStructure != nil {
		tailwindDefault = w.detectedStructure.HasTailwindCSS ||
			w.detectedStructure.ProjectType == "web" ||
			w.detectedStructure.ProjectType == "fullstack"
	}
	if tailwindDefault {
		fmt.Println("💡 Tailwind CSS detected or recommended for web projects")
	}
	if w.askBool("Enable tailwind plugin", tailwindDefault) {
		enabledPlugins = append(enabledPlugins, "tailwind")
	}

	// Suggest TypeScript plugin if detected
	if w.detectedStructure != nil && w.detectedStructure.HasTypeScript {
		fmt.Println("💡 TypeScript configuration detected")
		if w.askBool("Enable typescript plugin", true) {
			enabledPlugins = append(enabledPlugins, "typescript")
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

func (w *ConfigWizard) configureMonitoring() error {
	fmt.Println("📊 Monitoring Configuration")
	fmt.Println("---------------------------")

	w.config.Monitoring.Enabled = w.askBool("Enable application monitoring", true)
	w.config.Monitoring.LogLevel = w.askChoice(
		"Log level",
		[]string{"debug", "info", "warn", "error", "fatal"},
		"info",
	)
	w.config.Monitoring.LogFormat = w.askChoice("Log format", []string{"json", "text"}, "json")
	w.config.Monitoring.AlertsEnabled = w.askBool("Enable alerts", false)

	if w.config.Monitoring.Enabled {
		w.config.Monitoring.MetricsPath = w.askString("Metrics file path", "./logs/metrics.json")
		w.config.Monitoring.HTTPPort = 8081 // Use default
		if w.askBool("Custom monitoring port", false) {
			port, err := w.askInt("Monitoring HTTP port", 8081, 1024, 65535)
			if err != nil {
				return err
			}
			w.config.Monitoring.HTTPPort = port
		}
	}

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

	// Handle common false inputs that should use defaults
	if input == "y" || input == "yes" || input == "n" || input == "no" {
		if defaultValue != "" {
			return defaultValue
		}
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
			fmt.Printf(
				"❌ Number out of range. Please enter a number between %d and %d.\n",
				min,
				max,
			)
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
			if strings.EqualFold(input, choice) {
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
		overwrite := w.askBool(
			fmt.Sprintf("Configuration file %s already exists. Overwrite", filename),
			false,
		)
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
	builder.WriteString(
		fmt.Sprintf("  state_preservation: %t\n", w.config.Development.StatePreservation),
	)
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

	// Monitoring configuration
	if w.config.Monitoring.Enabled || w.config.Monitoring.LogLevel != "" ||
		w.config.Monitoring.LogFormat != "" {
		builder.WriteString("\nmonitoring:\n")
		builder.WriteString(fmt.Sprintf("  enabled: %t\n", w.config.Monitoring.Enabled))
		if w.config.Monitoring.LogLevel != "" {
			builder.WriteString(fmt.Sprintf("  log_level: %s\n", w.config.Monitoring.LogLevel))
		}
		if w.config.Monitoring.LogFormat != "" {
			builder.WriteString(fmt.Sprintf("  log_format: %s\n", w.config.Monitoring.LogFormat))
		}
		if w.config.Monitoring.MetricsPath != "" {
			builder.WriteString(
				fmt.Sprintf("  metrics_path: \"%s\"\n", w.config.Monitoring.MetricsPath),
			)
		}
		if w.config.Monitoring.HTTPPort != 0 && w.config.Monitoring.HTTPPort != 8081 {
			builder.WriteString(fmt.Sprintf("  http_port: %d\n", w.config.Monitoring.HTTPPort))
		}
		builder.WriteString(
			fmt.Sprintf("  alerts_enabled: %t\n", w.config.Monitoring.AlertsEnabled),
		)
	}

	return builder.String()
}

// detectProjectStructure analyzes the project directory to determine smart defaults
func (w *ConfigWizard) detectProjectStructure() {
	if w.projectDir == "" {
		return
	}

	w.detectedStructure = &ProjectStructure{
		ComponentDirs: []string{},
	}

	// Check for existing files and directories
	w.detectedStructure.HasGoMod = w.fileExists("go.mod")
	w.detectedStructure.HasNodeModules = w.fileExists("node_modules")
	w.detectedStructure.HasTailwindCSS = w.fileExists("tailwind.config.js") ||
		w.fileExists("tailwind.config.ts")
	w.detectedStructure.HasTypeScript = w.fileExists("tsconfig.json")

	// Scan for existing .templ files
	w.detectedStructure.HasExistingTempl = w.hasTemplFiles()

	// Detect existing component directories
	possibleDirs := []string{"components", "views", "examples", "templates", "ui", "pages"}
	for _, dir := range possibleDirs {
		if w.fileExists(dir) {
			w.detectedStructure.ComponentDirs = append(w.detectedStructure.ComponentDirs, "./"+dir)
		}
	}

	// Determine project type based on structure
	w.detectedStructure.ProjectType = w.inferProjectType()
}

// fileExists checks if a file or directory exists in the project directory
func (w *ConfigWizard) fileExists(path string) bool {
	if w.projectDir == "" {
		return false
	}
	fullPath := w.projectDir + "/" + path
	_, err := os.Stat(fullPath)
	return err == nil
}

// hasTemplFiles recursively checks for .templ files
func (w *ConfigWizard) hasTemplFiles() bool {
	if w.projectDir == "" {
		return false
	}

	found := false
	filepath.Walk(w.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, ".templ") {
			found = true
			return filepath.SkipDir
		}
		return nil
	})

	return found
}

// inferProjectType determines the project type based on detected structure
func (w *ConfigWizard) inferProjectType() string {
	if w.detectedStructure.HasNodeModules && w.detectedStructure.HasGoMod {
		return "fullstack"
	}
	if w.detectedStructure.HasTailwindCSS || len(w.detectedStructure.ComponentDirs) > 0 {
		return "web"
	}
	if w.detectedStructure.HasGoMod && !w.detectedStructure.HasExistingTempl {
		return "api"
	}
	return "web" // default
}
