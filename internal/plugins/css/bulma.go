package css

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// BulmaPlugin implements CSSFrameworkPlugin for Bulma CSS framework.
type BulmaPlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

// NewBulmaPlugin creates a new Bulma plugin instance.
func NewBulmaPlugin() *BulmaPlugin {
	return &BulmaPlugin{
		name:    "bulma",
		version: "1.0.0",
		config:  make(map[string]interface{}),
	}
}

// GetName returns the plugin name.
func (p *BulmaPlugin) GetName() string {
	return p.name
}

// GetVersion returns the plugin version.
func (p *BulmaPlugin) GetVersion() string {
	return p.version
}

// Initialize initializes the Bulma plugin.
func (p *BulmaPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.config = config

	return nil
}

// Cleanup cleans up plugin resources.
func (p *BulmaPlugin) Cleanup() error {
	return nil
}

// GetFrameworkName returns the framework name.
func (p *BulmaPlugin) GetFrameworkName() string {
	return "bulma"
}

// GetSupportedVersions returns supported Bulma versions.
func (p *BulmaPlugin) GetSupportedVersions() []string {
	return []string{"1.0.2", "0.9.4", "0.9.3", "0.9.2", "0.9.1"}
}

// GetDefaultConfig returns default configuration for Bulma.
func (p *BulmaPlugin) GetDefaultConfig() FrameworkConfig {
	return FrameworkConfig{
		Name:          "bulma",
		Version:       "1.0.2",
		InstallMethod: "npm",
		ConfigFile:    "bulma.config.js",
		EntryPoint:    "src/sass/main.sass",
		OutputPath:    "dist/css/bulma.min.css",
		SourcePaths:   []string{"src/**/*.{templ,html,js,ts}"},

		Preprocessing: []string{"sass"},

		Optimization: OptimizationConfig{
			Enabled:   true,
			Purge:     true,
			Minify:    true,
			TreeShake: true,
			Compress:  true,
		},

		Theming: ThemingConfig{
			Enabled:          true,
			ExtractVariables: true,
			GenerateTokens:   true,
			StyleGuide:       true,
			OutputFormat:     "sass",
			OutputFile:       "src/sass/_variables.sass",
		},

		Variables: map[string]string{
			// Colors
			"primary": "#00d1b2",
			"link":    "#485fc7",
			"info":    "#3e8ed0",
			"success": "#48c78e",
			"warning": "#ffe08a",
			"danger":  "#f14668",

			// Typography
			"family-sans-serif": "BlinkMacSystemFont, -apple-system, 'Segoe UI', 'Roboto', 'Oxygen', " +
				"'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue', 'Helvetica', 'Arial', sans-serif",
			"family-monospace": "monospace",
			"size-1":           "3rem",
			"size-2":           "2.5rem",
			"size-3":           "2rem",
			"size-4":           "1.5rem",
			"size-5":           "1.25rem",
			"size-6":           "1rem",
			"size-7":           "0.75rem",

			// Layout
			"gap":        "0.75rem",
			"tablet":     "769px",
			"desktop":    "1024px",
			"widescreen": "1216px",
			"fullhd":     "1408px",
		},

		Options: map[string]interface{}{
			"enable_columns":    true,
			"enable_components": true,
			"enable_elements":   true,
			"enable_form":       true,
			"enable_grid":       true,
			"enable_helpers":    true,
			"enable_layout":     true,
		},
	}
}

// IsInstalled checks if Bulma is installed.
func (p *BulmaPlugin) IsInstalled() bool {
	// Check for npm package
	if _, err := os.Stat("node_modules/bulma"); err == nil {
		return true
	}

	// Check for standalone CSS files
	if _, err := os.Stat("css/bulma.css"); err == nil {
		return true
	}

	if _, err := os.Stat("dist/css/bulma.css"); err == nil {
		return true
	}

	return false
}

// Setup sets up Bulma with the given configuration.
func (p *BulmaPlugin) Setup(ctx context.Context, config FrameworkConfig) error {
	switch config.InstallMethod {
	case "npm":
		return p.setupWithNPM(ctx, config)
	case "cdn":
		return p.setupWithCDN(ctx, config)
	case "standalone":
		return p.setupStandalone(ctx, config)
	default:
		return fmt.Errorf("unsupported install method: %s", config.InstallMethod)
	}
}

// setupWithNPM sets up Bulma using npm.
func (p *BulmaPlugin) setupWithNPM(ctx context.Context, config FrameworkConfig) error {
	// Install Bulma via npm
	cmd := exec.CommandContext(ctx, "npm", "install", "bulma@"+config.Version)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Bulma via npm: %w", err)
	}

	// Install Sass if needed
	if contains(config.Preprocessing, "sass") {
		cmd = exec.CommandContext(ctx, "npm", "install", "--save-dev", "sass")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install sass: %w", err)
		}
	}

	// Create entry point Sass file
	if err := p.createEntryPoint(config); err != nil {
		return fmt.Errorf("failed to create entry point: %w", err)
	}

	return nil
}

// setupWithCDN sets up Bulma using CDN.
func (p *BulmaPlugin) setupWithCDN(ctx context.Context, config FrameworkConfig) error {
	cdnUrl := config.CDNUrl
	if cdnUrl == "" {
		cdnUrl = fmt.Sprintf(
			"https://cdn.jsdelivr.net/npm/bulma@%s/css/bulma.min.css",
			config.Version,
		)
	}

	// Create a simple CSS file that imports from CDN
	cssContent := fmt.Sprintf("@import url('%s');\n", cdnUrl)

	// Ensure output directory exists
	outputDir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write CSS file
	if err := os.WriteFile(config.OutputPath, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}

	return nil
}

// setupStandalone sets up Bulma as standalone files.
func (p *BulmaPlugin) setupStandalone(ctx context.Context, config FrameworkConfig) error {
	outputDir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create basic Bulma CSS (simplified)
	basicCSS := p.generateBasicBulmaCSS(config)
	if err := os.WriteFile(config.OutputPath, []byte(basicCSS), 0644); err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}

	return nil
}

// createEntryPoint creates the main Sass entry point file.
func (p *BulmaPlugin) createEntryPoint(config FrameworkConfig) error {
	entryDir := filepath.Dir(config.EntryPoint)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return fmt.Errorf("failed to create entry point directory: %w", err)
	}

	// Generate Sass content
	sassContent := p.generateBulmaSass(config)

	if err := os.WriteFile(config.EntryPoint, []byte(sassContent), 0644); err != nil {
		return fmt.Errorf("failed to write entry point file: %w", err)
	}

	return nil
}

// generateBulmaSass generates the main Bulma Sass file.
func (p *BulmaPlugin) generateBulmaSass(config FrameworkConfig) string {
	var sass strings.Builder

	// Add custom variables if defined
	if len(config.Variables) > 0 {
		sass.WriteString("// Custom Bulma Variables\n")
		for key, value := range config.Variables {
			sass.WriteString(fmt.Sprintf("$%s: %s\n", key, value))
		}
		sass.WriteString("\n")
	}

	// Import Bulma components selectively based on options
	sass.WriteString("// Import Bulma\n")

	// Always import utilities and base
	sass.WriteString("@import '~bulma/sass/utilities/_all'\n")
	sass.WriteString("@import '~bulma/sass/base/_all'\n")

	// Conditionally import components based on options
	if config.Options["enable_elements"].(bool) {
		sass.WriteString("@import '~bulma/sass/elements/_all'\n")
	}
	if config.Options["enable_form"].(bool) {
		sass.WriteString("@import '~bulma/sass/form/_all'\n")
	}
	if config.Options["enable_components"].(bool) {
		sass.WriteString("@import '~bulma/sass/components/_all'\n")
	}
	if config.Options["enable_grid"].(bool) {
		sass.WriteString("@import '~bulma/sass/grid/_all'\n")
	}
	if config.Options["enable_helpers"].(bool) {
		sass.WriteString("@import '~bulma/sass/helpers/_all'\n")
	}
	if config.Options["enable_layout"].(bool) {
		sass.WriteString("@import '~bulma/sass/layout/_all'\n")
	}

	sass.WriteString("\n")

	// Add custom styles section
	sass.WriteString("// Custom Styles\n")
	sass.WriteString("// Add your custom styles here\n")

	return sass.String()
}

// generateBasicBulmaCSS generates basic Bulma CSS for standalone setup.
func (p *BulmaPlugin) generateBasicBulmaCSS(config FrameworkConfig) string {
	css := `/* Bulma CSS Framework */

/* Reset */
html {
  box-sizing: border-box;
}

*, *::before, *::after {
  box-sizing: inherit;
}

/* Colors */
:root {
  --bulma-primary: #00d1b2;
  --bulma-link: #485fc7;
  --bulma-info: #3e8ed0;
  --bulma-success: #48c78e;
  --bulma-warning: #ffe08a;
  --bulma-danger: #f14668;
  --bulma-dark: #363636;
  --bulma-text: #4a4a4a;
  --bulma-white: #ffffff;
}

/* Container */
.container {
  flex-grow: 1;
  margin: 0 auto;
  position: relative;
  width: auto;
  max-width: 1344px;
  padding: 0 1.5rem;
}

/* Columns */
.columns {
  display: flex;
  margin: -0.75rem;
}

.column {
  display: block;
  flex-basis: 0;
  flex-grow: 1;
  flex-shrink: 1;
  padding: 0.75rem;
}

/* Buttons */
.button {
  background-color: white;
  border: 1px solid #dbdbdb;
  border-radius: 4px;
  color: #363636;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5em 1em;
  text-align: center;
  white-space: nowrap;
  text-decoration: none;
  font-size: 1rem;
  line-height: 1.5;
}

.button.is-primary {
  background-color: var(--bulma-primary);
  border-color: transparent;
  color: white;
}

.button.is-link {
  background-color: var(--bulma-link);
  border-color: transparent;
  color: white;
}

.button.is-info {
  background-color: var(--bulma-info);
  border-color: transparent;
  color: white;
}

.button.is-success {
  background-color: var(--bulma-success);
  border-color: transparent;
  color: white;
}

.button.is-warning {
  background-color: var(--bulma-warning);
  border-color: transparent;
  color: rgba(0, 0, 0, 0.7);
}

.button.is-danger {
  background-color: var(--bulma-danger);
  border-color: transparent;
  color: white;
}

/* Typography */
.title {
  color: #363636;
  font-size: 2rem;
  font-weight: 600;
  line-height: 1.125;
  margin-bottom: 1.5rem;
}

.subtitle {
  color: #4a4a4a;
  font-size: 1.25rem;
  font-weight: 400;
  line-height: 1.25;
  margin-bottom: 1.5rem;
}

/* Box */
.box {
  background-color: white;
  border-radius: 6px;
  box-shadow: 0 0.5em 1em -0.125em rgba(10, 10, 10, 0.1), 0 0px 0 1px rgba(10, 10, 10, 0.02);
  color: #4a4a4a;
  display: block;
  padding: 1.25rem;
}

/* Notification */
.notification {
  background-color: #f5f5f5;
  border-radius: 4px;
  position: relative;
  padding: 1.25rem 2.5rem 1.25rem 1.5rem;
}

.notification.is-primary {
  background-color: var(--bulma-primary);
  color: white;
}

.notification.is-info {
  background-color: var(--bulma-info);
  color: white;
}

.notification.is-success {
  background-color: var(--bulma-success);
  color: white;
}

.notification.is-warning {
  background-color: var(--bulma-warning);
  color: rgba(0, 0, 0, 0.7);
}

.notification.is-danger {
  background-color: var(--bulma-danger);
  color: white;
}

/* Hero */
.hero {
  align-items: stretch;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.hero-body {
  flex-grow: 1;
  flex-shrink: 0;
  padding: 3rem 1.5rem;
}

/* Section */
.section {
  padding: 3rem 1.5rem;
}
`

	// Apply custom variables to CSS
	for key, value := range config.Variables {
		css = strings.ReplaceAll(css, fmt.Sprintf("var(--bulma-%s)", key), value)
	}

	return css
}

// GenerateConfig generates a Bulma configuration file.
func (p *BulmaPlugin) GenerateConfig(config FrameworkConfig) ([]byte, error) {
	configContent := fmt.Sprintf(`// Bulma Configuration
module.exports = {
  // Framework settings
  framework: '%s',
  version: '%s',
  
  // Build settings
  entry: '%s',
  output: '%s',
  
  // Sass settings
  sassOptions: {
    includePaths: ['node_modules'],
    sourceMap: true,
    indentedSyntax: true
  },
  
  // Optimization
  optimization: {
    purge: %t,
    minify: %t,
    removeUnused: %t
  },
  
  // Custom variables
  variables: %s,
  
  // Bulma modules
  modules: {
    elements: %t,
    form: %t,
    components: %t,
    grid: %t,
    helpers: %t,
    layout: %t
  }
};
`,
		config.Name,
		config.Version,
		config.EntryPoint,
		config.OutputPath,
		config.Optimization.Purge,
		config.Optimization.Minify,
		config.Optimization.RemoveUnused,
		formatVariablesAsJS(config.Variables),
		config.Options["enable_elements"].(bool),
		config.Options["enable_form"].(bool),
		config.Options["enable_components"].(bool),
		config.Options["enable_grid"].(bool),
		config.Options["enable_helpers"].(bool),
		config.Options["enable_layout"].(bool),
	)

	return []byte(configContent), nil
}

// ValidateConfig validates Bulma configuration.
func (p *BulmaPlugin) ValidateConfig(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read and validate config file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Basic validation - check for required exports
	if !strings.Contains(string(content), "module.exports") {
		return errors.New("config file must export a configuration object")
	}

	return nil
}

// ProcessCSS processes CSS using Bulma.
func (p *BulmaPlugin) ProcessCSS(
	ctx context.Context,
	input []byte,
	options ProcessingOptions,
) ([]byte, error) {
	// For Bulma, we primarily work with Sass compilation
	if strings.HasSuffix(options.InputPath, ".sass") ||
		strings.HasSuffix(options.InputPath, ".scss") {
		return p.compileSass(ctx, input, options)
	}

	// For regular CSS, apply optimizations if requested
	output := input
	var err error

	if options.Optimize {
		output, err = p.optimizeCSS(output, options)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize CSS: %w", err)
		}
	}

	return output, nil
}

// compileSass compiles Sass to CSS using sass.
func (p *BulmaPlugin) compileSass(
	ctx context.Context,
	input []byte,
	options ProcessingOptions,
) ([]byte, error) {
	// Create temporary input file
	tmpDir := os.TempDir()
	inputFile := filepath.Join(tmpDir, "input.sass")
	outputFile := filepath.Join(tmpDir, "output.css")

	if err := os.WriteFile(inputFile, input, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temporary input file: %w", err)
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)

	// Run sass compiler
	args := []string{inputFile, outputFile, "--indented"}
	if options.SourceMaps {
		args = append(args, "--source-map")
	}
	if options.Minify {
		args = append(args, "--style=compressed")
	}

	cmd := exec.CommandContext(ctx, "sass", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile Sass: %w", err)
	}

	// Read output
	output, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled CSS: %w", err)
	}

	return output, nil
}

// optimizeCSS applies CSS optimizations.
func (p *BulmaPlugin) optimizeCSS(css []byte, options ProcessingOptions) ([]byte, error) {
	cssStr := string(css)

	// Remove comments
	cssStr = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(cssStr, "")

	// Remove extra whitespace
	cssStr = regexp.MustCompile(`\s+`).ReplaceAllString(cssStr, " ")
	cssStr = strings.TrimSpace(cssStr)

	// Purge unused classes if requested and used classes are provided
	if options.Purge && len(options.UsedClasses) > 0 {
		cssStr = p.purgeUnusedClasses(cssStr, options.UsedClasses)
	}

	return []byte(cssStr), nil
}

// purgeUnusedClasses removes unused CSS classes.
func (p *BulmaPlugin) purgeUnusedClasses(css string, usedClasses []string) string {
	// Create a map for fast lookup
	usedMap := make(map[string]bool)
	for _, class := range usedClasses {
		usedMap[class] = true
	}

	// Simple purging - remove class rules that aren't used
	lines := strings.Split(css, "\n")
	var result []string

	inRule := false
	var currentRule strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, ".") && strings.Contains(line, "{") {
			// Start of a class rule
			inRule = true
			currentRule.Reset()
			currentRule.WriteString(line)
		} else if inRule {
			currentRule.WriteString("\n" + line)
			if strings.Contains(line, "}") {
				// End of rule
				rule := currentRule.String()
				if p.shouldKeepRule(rule, usedMap) {
					result = append(result, rule)
				}
				inRule = false
			}
		} else if !inRule {
			// Not in a class rule, keep as-is
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// shouldKeepRule determines if a CSS rule should be kept based on used classes.
func (p *BulmaPlugin) shouldKeepRule(rule string, usedClasses map[string]bool) bool {
	// Extract class names from the rule
	classRegex := regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	matches := classRegex.FindAllStringSubmatch(rule, -1)

	for _, match := range matches {
		if len(match) > 1 {
			className := match[1]
			if usedClasses[className] {
				return true
			}
		}
	}

	return false
}

// ExtractClasses extracts Bulma classes from content.
func (p *BulmaPlugin) ExtractClasses(content string) ([]string, error) {
	var classes []string
	classRegex := regexp.MustCompile(`class="([^"]*)"`)

	matches := classRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			classNames := strings.Fields(match[1])
			for _, className := range classNames {
				if p.isBulmaClass(className) {
					classes = append(classes, className)
				}
			}
		}
	}

	return removeDuplicates(classes), nil
}

// isBulmaClass checks if a class name is a Bulma class.
func (p *BulmaPlugin) isBulmaClass(className string) bool {
	bulmaPrefixes := []string{
		"button", "is-", "has-", "column", "columns", "container", "content",
		"delete", "icon", "image", "notification", "progress", "table", "tag",
		"title", "subtitle", "heading", "number", "breadcrumb", "card", "dropdown",
		"level", "media", "menu", "message", "modal", "navbar", "pagination",
		"panel", "tabs", "box", "field", "control", "input", "textarea", "select",
		"checkbox", "radio", "file", "label", "help", "hero", "section", "footer",
		"tile", "m-", "p-", "mt-", "mb-", "ml-", "mr-", "pt-", "pb-", "pl-", "pr-",
	}

	for _, prefix := range bulmaPrefixes {
		if strings.HasPrefix(className, prefix) {
			return true
		}
	}

	// Check for exact matches for common Bulma classes
	bulmaClasses := []string{
		"container", "section", "hero", "footer", "navbar", "menu", "panel",
		"card", "box", "notification", "message", "modal", "dropdown", "tabs",
		"pagination", "breadcrumb", "level", "media", "tile", "columns", "column",
		"button", "input", "textarea", "select", "field", "control", "label",
		"table", "tag", "progress", "delete", "icon", "image", "title", "subtitle",
		"heading", "number", "content",
	}

	for _, class := range bulmaClasses {
		if className == class {
			return true
		}
	}

	return false
}

// OptimizeCSS optimizes CSS for Bulma.
func (p *BulmaPlugin) OptimizeCSS(
	ctx context.Context,
	css []byte,
	usedClasses []string,
) ([]byte, error) {
	options := ProcessingOptions{
		Purge:       true,
		Optimize:    true,
		UsedClasses: usedClasses,
		Environment: "production",
	}

	return p.optimizeCSS(css, options)
}

// ExtractVariables extracts CSS variables from Bulma CSS.
func (p *BulmaPlugin) ExtractVariables(css []byte) (map[string]string, error) {
	variables := make(map[string]string)

	// Extract CSS custom properties (--variable: value)
	varRegex := regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9_-]*)\s*:\s*([^;]+);`)
	matches := varRegex.FindAllStringSubmatch(string(css), -1)

	for _, match := range matches {
		if len(match) > 2 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			variables[name] = value
		}
	}

	// Extract Sass variables ($variable: value)
	sassVarRegex := regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_-]*)\s*:\s*([^;]+);`)
	sassMatches := sassVarRegex.FindAllStringSubmatch(string(css), -1)

	for _, match := range sassMatches {
		if len(match) > 2 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			variables[name] = value
		}
	}

	return variables, nil
}

// GenerateTheme generates a Bulma theme with custom variables.
func (p *BulmaPlugin) GenerateTheme(variables map[string]string) ([]byte, error) {
	var theme strings.Builder

	theme.WriteString("// Custom Bulma Theme\n")
	theme.WriteString("// Generated by Templar\n\n")

	// Add custom variables
	for name, value := range variables {
		theme.WriteString(fmt.Sprintf("$%s: %s\n", name, value))
	}

	theme.WriteString("\n// Import Bulma\n")
	theme.WriteString("@import '~bulma/bulma'\n")

	return []byte(theme.String()), nil
}

// GetDevServerConfig returns development server configuration.
func (p *BulmaPlugin) GetDevServerConfig() DevServerConfig {
	return DevServerConfig{
		HotReload:      true,
		WatchPaths:     []string{"src/**/*.sass", "src/**/*.scss", "src/**/*.css"},
		ReloadDelay:    300,
		CSSInjection:   true,
		InjectTarget:   "head",
		ErrorOverlay:   true,
		SourceMaps:     true,
		LiveValidation: true,
		DevMode:        true,
		DevOptions: map[string]interface{}{
			"sass_source_maps": true,
			"sass_indented":    true,
			"css_autoprefixer": false, // Bulma handles prefixes
		},
	}
}

// SupportsHotReload returns true if the framework supports hot reload.
func (p *BulmaPlugin) SupportsHotReload() bool {
	return true
}

// GenerateStyleGuide generates a Bulma style guide.
func (p *BulmaPlugin) GenerateStyleGuide(ctx context.Context) ([]byte, error) {
	styleGuide := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bulma Style Guide</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.2/css/bulma.min.css">
</head>
<body>
    <section class="section">
        <div class="container">
            <h1 class="title">Bulma Style Guide</h1>
            
            <section class="section">
                <h2 class="title is-4">Colors</h2>
                <div class="columns">
                    <div class="column">
                        <div class="notification is-primary">Primary</div>
                    </div>
                    <div class="column">
                        <div class="notification is-link">Link</div>
                    </div>
                    <div class="column">
                        <div class="notification is-info">Info</div>
                    </div>
                    <div class="column">
                        <div class="notification is-success">Success</div>
                    </div>
                    <div class="column">
                        <div class="notification is-warning">Warning</div>
                    </div>
                    <div class="column">
                        <div class="notification is-danger">Danger</div>
                    </div>
                </div>
            </section>
            
            <section class="section">
                <h2 class="title is-4">Buttons</h2>
                <div class="field is-grouped">
                    <p class="control">
                        <button class="button is-primary">Primary</button>
                    </p>
                    <p class="control">
                        <button class="button is-link">Link</button>
                    </p>
                    <p class="control">
                        <button class="button is-info">Info</button>
                    </p>
                    <p class="control">
                        <button class="button is-success">Success</button>
                    </p>
                    <p class="control">
                        <button class="button is-warning">Warning</button>
                    </p>
                    <p class="control">
                        <button class="button is-danger">Danger</button>
                    </p>
                </div>
            </section>
            
            <section class="section">
                <h2 class="title is-4">Typography</h2>
                <h1 class="title is-1">Title 1</h1>
                <h2 class="title is-2">Title 2</h2>
                <h3 class="title is-3">Title 3</h3>
                <h4 class="title is-4">Title 4</h4>
                <h5 class="title is-5">Title 5</h5>
                <h6 class="title is-6">Title 6</h6>
                <p class="subtitle">This is a subtitle</p>
                <p>This is a paragraph with <strong>bold text</strong> and <em>italic text</em>.</p>
            </section>
            
            <section class="section">
                <h2 class="title is-4">Columns</h2>
                <div class="columns">
                    <div class="column">
                        <div class="box">Column 1</div>
                    </div>
                    <div class="column">
                        <div class="box">Column 2</div>
                    </div>
                    <div class="column">
                        <div class="box">Column 3</div>
                    </div>
                </div>
            </section>
            
            <section class="section">
                <h2 class="title is-4">Components</h2>
                
                <div class="columns">
                    <div class="column">
                        <div class="card">
                            <div class="card-content">
                                <p class="title">Card Title</p>
                                <p class="subtitle">Card Subtitle</p>
                                <div class="content">
                                    Lorem ipsum dolor sit amet, consectetur adipiscing elit.
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="column">
                        <article class="message is-info">
                            <div class="message-header">
                                <p>Info Message</p>
                                <button class="delete" aria-label="delete"></button>
                            </div>
                            <div class="message-body">
                                This is an info message with useful information.
                            </div>
                        </article>
                    </div>
                </div>
            </section>
        </div>
    </section>
</body>
</html>`

	return []byte(styleGuide), nil
}

// getBulmaTemplates returns built-in Bulma component templates.
func getBulmaTemplates() []ComponentTemplate {
	return []ComponentTemplate{
		{
			Name:        "Button",
			Framework:   "bulma",
			Category:    "button",
			Description: "Bulma button component",
			Template: `templ Button(text string, color string) {
	<button class={ "button", "is-" + color }>
		{ text }
	</button>
}`,
			Props: []types.ParameterInfo{
				{Name: "text", Type: "string", Optional: false},
				{Name: "color", Type: "string", Optional: false},
			},
			Examples: []TemplateExample{
				{
					Name:        "Primary Button",
					Description: "A primary button example",
					Props: map[string]interface{}{
						"text":  "Click Me",
						"color": "primary",
					},
				},
			},
			Classes: []string{
				"button",
				"is-primary",
				"is-link",
				"is-info",
				"is-success",
				"is-warning",
				"is-danger",
			},
		},
		{
			Name:        "Card",
			Framework:   "bulma",
			Category:    "layout",
			Description: "Bulma card component",
			Template: `templ Card(title string, subtitle string, content string) {
	<div class="card">
		<div class="card-content">
			<p class="title">{ title }</p>
			if subtitle != "" {
				<p class="subtitle">{ subtitle }</p>
			}
			<div class="content">
				{ content }
			</div>
		</div>
	</div>
}`,
			Props: []types.ParameterInfo{
				{Name: "title", Type: "string", Optional: false},
				{Name: "subtitle", Type: "string", Optional: true},
				{Name: "content", Type: "string", Optional: false},
			},
			Classes: []string{"card", "card-content", "title", "subtitle", "content"},
		},
		{
			Name:        "Notification",
			Framework:   "bulma",
			Category:    "components",
			Description: "Bulma notification component",
			Template: `templ Notification(message string, color string) {
	<div class={ "notification", "is-" + color }>
		{ message }
	</div>
}`,
			Props: []types.ParameterInfo{
				{Name: "message", Type: "string", Optional: false},
				{Name: "color", Type: "string", Optional: false},
			},
			Classes: []string{
				"notification",
				"is-primary",
				"is-info",
				"is-success",
				"is-warning",
				"is-danger",
			},
		},
	}
}
