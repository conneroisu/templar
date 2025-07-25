package css

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
)

// FrameworkManager manages CSS framework integration and setup
type FrameworkManager struct {
	registry        *FrameworkRegistry
	config          *config.Config
	projectPath     string
	activeFramework string
}

// NewFrameworkManager creates a new framework manager
func NewFrameworkManager(cfg *config.Config, projectPath string) *FrameworkManager {
	return &FrameworkManager{
		registry:    NewFrameworkRegistry(),
		config:      cfg,
		projectPath: projectPath,
	}
}

// Initialize initializes the framework manager and registers built-in frameworks
func (m *FrameworkManager) Initialize(ctx context.Context) error {
	// Register built-in frameworks
	if err := m.registerBuiltinFrameworks(); err != nil {
		return fmt.Errorf("failed to register builtin frameworks: %w", err)
	}

	// Detect active framework
	detected, err := m.registry.DetectFramework(m.projectPath)
	if err != nil {
		return fmt.Errorf("failed to detect frameworks: %w", err)
	}

	if len(detected) > 0 {
		m.activeFramework = detected[0] // Use first detected framework
	}

	return nil
}

// registerBuiltinFrameworks registers the built-in CSS framework plugins
func (m *FrameworkManager) registerBuiltinFrameworks() error {
	// Register Tailwind CSS
	tailwind := NewTailwindPlugin()
	if err := m.registry.Register(tailwind); err != nil {
		return fmt.Errorf("failed to register tailwind: %w", err)
	}

	// Register Bootstrap
	bootstrap := NewBootstrapPlugin()
	if err := m.registry.Register(bootstrap); err != nil {
		return fmt.Errorf("failed to register bootstrap: %w", err)
	}

	// Register Bulma
	bulma := NewBulmaPlugin()
	if err := m.registry.Register(bulma); err != nil {
		return fmt.Errorf("failed to register bulma: %w", err)
	}

	return nil
}

// SetupFramework sets up a CSS framework with the given configuration
func (m *FrameworkManager) SetupFramework(ctx context.Context, frameworkName string, setupConfig FrameworkSetupConfig) error {
	plugin, exists := m.registry.Get(frameworkName)
	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	// Check if framework is already installed
	if plugin.IsInstalled() && !setupConfig.Force {
		return fmt.Errorf("framework %s is already installed (use --force to reinstall)", frameworkName)
	}

	// Create framework configuration
	config := plugin.GetDefaultConfig()
	config.Name = frameworkName
	config.InstallMethod = setupConfig.InstallMethod
	config.OutputPath = setupConfig.OutputPath
	config.SourcePaths = setupConfig.SourcePaths

	// Apply setup options
	if setupConfig.CDNUrl != "" {
		config.CDNUrl = setupConfig.CDNUrl
	}
	if setupConfig.Version != "" {
		config.Version = setupConfig.Version
	}

	// Merge custom options
	if config.Options == nil {
		config.Options = make(map[string]interface{})
	}
	for key, value := range setupConfig.Options {
		config.Options[key] = value
	}

	// Setup the framework
	if err := plugin.Setup(ctx, config); err != nil {
		return fmt.Errorf("failed to setup framework %s: %w", frameworkName, err)
	}

	// Generate configuration file
	if setupConfig.GenerateConfig {
		configContent, err := plugin.GenerateConfig(config)
		if err != nil {
			return fmt.Errorf("failed to generate config for %s: %w", frameworkName, err)
		}

		configPath := filepath.Join(m.projectPath, config.ConfigFile)
		if err := os.WriteFile(configPath, configContent, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", configPath, err)
		}
	}

	// Store configuration
	m.registry.SetConfig(frameworkName, config)
	m.activeFramework = frameworkName

	// Update project configuration
	if err := m.updateProjectConfig(frameworkName, config); err != nil {
		return fmt.Errorf("failed to update project config: %w", err)
	}

	return nil
}

// FrameworkSetupConfig represents setup configuration for a framework
type FrameworkSetupConfig struct {
	InstallMethod  string                 `json:"install_method"`
	Version        string                 `json:"version,omitempty"`
	CDNUrl         string                 `json:"cdn_url,omitempty"`
	OutputPath     string                 `json:"output_path"`
	SourcePaths    []string               `json:"source_paths"`
	GenerateConfig bool                   `json:"generate_config"`
	Force          bool                   `json:"force"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// GetAvailableFrameworks returns a list of available CSS frameworks
func (m *FrameworkManager) GetAvailableFrameworks() []FrameworkInfo {
	var frameworks []FrameworkInfo

	for _, name := range m.registry.List() {
		info, err := m.registry.GetFrameworkInfo(name)
		if err != nil {
			continue
		}
		frameworks = append(frameworks, *info)
	}

	return frameworks
}

// GetActiveFramework returns the currently active framework
func (m *FrameworkManager) GetActiveFramework() string {
	return m.activeFramework
}

// ProcessCSS processes CSS for the active framework
func (m *FrameworkManager) ProcessCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error) {
	if m.activeFramework == "" {
		return input, nil // No framework active, return input as-is
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.ProcessCSS(ctx, input, options)
}

// ExtractClasses extracts CSS classes from content for the active framework
func (m *FrameworkManager) ExtractClasses(content string) ([]string, error) {
	if m.activeFramework == "" {
		return nil, nil
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.ExtractClasses(content)
}

// OptimizeCSS optimizes CSS for the active framework
func (m *FrameworkManager) OptimizeCSS(ctx context.Context, css []byte, usedClasses []string) ([]byte, error) {
	if m.activeFramework == "" {
		return css, nil
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.OptimizeCSS(ctx, css, usedClasses)
}

// ExtractVariables extracts CSS variables from the active framework
func (m *FrameworkManager) ExtractVariables(css []byte) (map[string]string, error) {
	if m.activeFramework == "" {
		return nil, nil
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.ExtractVariables(css)
}

// GenerateTheme generates a theme with custom variables
func (m *FrameworkManager) GenerateTheme(variables map[string]string) ([]byte, error) {
	if m.activeFramework == "" {
		return nil, fmt.Errorf("no active framework for theme generation")
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.GenerateTheme(variables)
}

// GenerateStyleGuide generates a style guide for the active framework
func (m *FrameworkManager) GenerateStyleGuide(ctx context.Context) ([]byte, error) {
	if m.activeFramework == "" {
		return nil, fmt.Errorf("no active framework for style guide generation")
	}

	plugin, exists := m.registry.Get(m.activeFramework)
	if !exists {
		return nil, fmt.Errorf("active framework %s not found", m.activeFramework)
	}

	return plugin.GenerateStyleGuide(ctx)
}

// GetFrameworkConfig returns the configuration for a framework
func (m *FrameworkManager) GetFrameworkConfig(frameworkName string) (FrameworkConfig, error) {
	config, exists := m.registry.GetConfig(frameworkName)
	if !exists {
		return FrameworkConfig{}, fmt.Errorf("no configuration found for framework %s", frameworkName)
	}

	return config, nil
}

// ValidateFramework validates the configuration and setup of a framework
func (m *FrameworkManager) ValidateFramework(frameworkName string) error {
	plugin, exists := m.registry.Get(frameworkName)
	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	config, exists := m.registry.GetConfig(frameworkName)
	if !exists {
		return fmt.Errorf("no configuration found for framework %s", frameworkName)
	}

	// Validate basic configuration
	if err := ValidateFrameworkConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if framework is properly installed
	if !plugin.IsInstalled() {
		return fmt.Errorf("framework %s is not installed", frameworkName)
	}

	// Validate framework-specific configuration file
	if config.ConfigFile != "" {
		configPath := filepath.Join(m.projectPath, config.ConfigFile)
		if err := plugin.ValidateConfig(configPath); err != nil {
			return fmt.Errorf("invalid framework config file %s: %w", configPath, err)
		}
	}

	return nil
}

// GetComponentTemplates returns available component templates for a framework
func (m *FrameworkManager) GetComponentTemplates(frameworkName string) ([]ComponentTemplate, error) {
	_, exists := m.registry.Get(frameworkName)
	if !exists {
		return nil, fmt.Errorf("framework %s not found", frameworkName)
	}

	// For now, return built-in templates
	// In the future, this could be extended to load templates from files or external sources
	return m.getBuiltinTemplates(frameworkName), nil
}

// getBuiltinTemplates returns built-in component templates for a framework
func (m *FrameworkManager) getBuiltinTemplates(frameworkName string) []ComponentTemplate {
	switch strings.ToLower(frameworkName) {
	case "tailwind":
		return getTailwindTemplates()
	case "bootstrap":
		return getBootstrapTemplates()
	case "bulma":
		return getBulmaTemplates()
	default:
		return []ComponentTemplate{}
	}
}

// updateProjectConfig updates the project configuration with framework settings
func (m *FrameworkManager) updateProjectConfig(frameworkName string, frameworkConfig FrameworkConfig) error {
	// This would integrate with the main config system
	// For now, we'll just ensure the CSS section exists in the config

	if m.config.CSS == nil {
		m.config.CSS = &config.CSSConfig{}
	}

	m.config.CSS.Framework = frameworkName
	m.config.CSS.OutputPath = frameworkConfig.OutputPath
	m.config.CSS.SourcePaths = frameworkConfig.SourcePaths

	// Set optimization settings
	if frameworkConfig.Optimization.Enabled {
		m.config.CSS.Optimization = &config.OptimizationConfig{
			Purge:  frameworkConfig.Optimization.Purge,
			Minify: frameworkConfig.Optimization.Minify,
		}
	}

	// Set theming settings
	if frameworkConfig.Theming.Enabled {
		m.config.CSS.Theming = &config.ThemingConfig{
			ExtractVariables: frameworkConfig.Theming.ExtractVariables,
			StyleGuide:       frameworkConfig.Theming.StyleGuide,
		}
	}

	return nil
}

// SwitchFramework switches to a different CSS framework
func (m *FrameworkManager) SwitchFramework(ctx context.Context, frameworkName string) error {
	plugin, exists := m.registry.Get(frameworkName)
	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	if !plugin.IsInstalled() {
		return fmt.Errorf("framework %s is not installed", frameworkName)
	}

	// Validate the framework configuration
	if err := m.ValidateFramework(frameworkName); err != nil {
		return fmt.Errorf("cannot switch to invalid framework: %w", err)
	}

	m.activeFramework = frameworkName

	// Update project configuration
	config, _ := m.registry.GetConfig(frameworkName)
	if err := m.updateProjectConfig(frameworkName, config); err != nil {
		return fmt.Errorf("failed to update project config: %w", err)
	}

	return nil
}

// RemoveFramework removes a CSS framework from the project
func (m *FrameworkManager) RemoveFramework(ctx context.Context, frameworkName string) error {
	plugin, exists := m.registry.Get(frameworkName)
	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	config, exists := m.registry.GetConfig(frameworkName)
	if !exists {
		return fmt.Errorf("no configuration found for framework %s", frameworkName)
	}

	// Remove configuration file
	if config.ConfigFile != "" {
		configPath := filepath.Join(m.projectPath, config.ConfigFile)
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config file %s: %w", configPath, err)
		}
	}

	// Remove output files
	if config.OutputPath != "" {
		outputPath := filepath.Join(m.projectPath, config.OutputPath)
		if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove output file %s: %w", outputPath, err)
		}
	}

	// Clean up plugin resources
	if err := plugin.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup framework %s: %w", frameworkName, err)
	}

	// Remove from registry
	delete(m.registry.configs, frameworkName)

	// Update active framework
	if m.activeFramework == frameworkName {
		m.activeFramework = ""
	}

	return nil
}
