package css

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// BootstrapPlugin implements CSSFrameworkPlugin for Bootstrap CSS framework
type BootstrapPlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

// NewBootstrapPlugin creates a new Bootstrap plugin instance
func NewBootstrapPlugin() *BootstrapPlugin {
	return &BootstrapPlugin{
		name:    "bootstrap",
		version: "1.0.0",
		config:  make(map[string]interface{}),
	}
}

// GetName returns the plugin name
func (p *BootstrapPlugin) GetName() string {
	return p.name
}

// GetVersion returns the plugin version
func (p *BootstrapPlugin) GetVersion() string {
	return p.version
}

// Initialize initializes the Bootstrap plugin
func (p *BootstrapPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.config = config
	return nil
}

// Cleanup cleans up plugin resources
func (p *BootstrapPlugin) Cleanup() error {
	return nil
}

// GetFrameworkName returns the framework name
func (p *BootstrapPlugin) GetFrameworkName() string {
	return "bootstrap"
}

// GetSupportedVersions returns supported Bootstrap versions
func (p *BootstrapPlugin) GetSupportedVersions() []string {
	return []string{"5.3.0", "5.2.3", "5.1.3", "5.0.2", "4.6.2"}
}

// GetDefaultConfig returns default configuration for Bootstrap
func (p *BootstrapPlugin) GetDefaultConfig() FrameworkConfig {
	return FrameworkConfig{
		Name:         "bootstrap",
		Version:      "5.3.0",
		InstallMethod: "npm",
		ConfigFile:   "bootstrap.config.js",
		EntryPoint:   "src/scss/main.scss",
		OutputPath:   "dist/css/bootstrap.min.css",
		SourcePaths:  []string{"src/**/*.{templ,html,js,ts}"},
		
		Preprocessing: []string{"scss"},
		
		Optimization: OptimizationConfig{
			Enabled:    true,
			Purge:      true,
			Minify:     true,
			TreeShake:  true,
			Compress:   true,
		},
		
		Theming: ThemingConfig{
			Enabled:          true,
			ExtractVariables: true,
			GenerateTokens:   true,
			StyleGuide:       true,
			OutputFormat:     "scss",
			OutputFile:       "src/scss/_variables.scss",
		},
		
		Variables: map[string]string{
			"primary":   "#0d6efd",
			"secondary": "#6c757d",
			"success":   "#198754",
			"info":      "#0dcaf0",
			"warning":   "#ffc107",
			"danger":    "#dc3545",
			"light":     "#f8f9fa",
			"dark":      "#212529",
		},
		
		Options: map[string]interface{}{
			"enable_grid":      true,
			"enable_utilities": true,
			"enable_print":     true,
			"rtl":             false,
		},
	}
}

// IsInstalled checks if Bootstrap is installed
func (p *BootstrapPlugin) IsInstalled() bool {
	// Check for npm package
	if _, err := os.Stat("node_modules/bootstrap"); err == nil {
		return true
	}
	
	// Check for CDN usage in HTML files
	// This is a simplified check - in practice, you'd scan HTML files for CDN links
	
	return false
}

// Setup sets up Bootstrap with the given configuration
func (p *BootstrapPlugin) Setup(ctx context.Context, config FrameworkConfig) error {
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

// setupWithNPM sets up Bootstrap using npm
func (p *BootstrapPlugin) setupWithNPM(ctx context.Context, config FrameworkConfig) error {
	// Install Bootstrap via npm
	cmd := exec.CommandContext(ctx, "npm", "install", fmt.Sprintf("bootstrap@%s", config.Version))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Bootstrap via npm: %w", err)
	}
	
	// Install SCSS if needed
	if contains(config.Preprocessing, "scss") {
		cmd = exec.CommandContext(ctx, "npm", "install", "--save-dev", "sass")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install sass: %w", err)
		}
	}
	
	// Create entry point SCSS file
	if err := p.createEntryPoint(config); err != nil {
		return fmt.Errorf("failed to create entry point: %w", err)
	}
	
	return nil
}

// setupWithCDN sets up Bootstrap using CDN
func (p *BootstrapPlugin) setupWithCDN(ctx context.Context, config FrameworkConfig) error {
	// For CDN setup, we just need to provide the CDN links
	// This would typically be integrated with HTML templates
	
	cdnUrl := config.CDNUrl
	if cdnUrl == "" {
		cdnUrl = fmt.Sprintf("https://cdn.jsdelivr.net/npm/bootstrap@%s/dist/css/bootstrap.min.css", config.Version)
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

// setupStandalone sets up Bootstrap as standalone files
func (p *BootstrapPlugin) setupStandalone(ctx context.Context, config FrameworkConfig) error {
	// Download Bootstrap CSS files
	// This is a simplified implementation - in practice, you'd download from official sources
	
	outputDir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create basic Bootstrap CSS (simplified)
	basicCSS := p.generateBasicBootstrapCSS(config)
	if err := os.WriteFile(config.OutputPath, []byte(basicCSS), 0644); err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}
	
	return nil
}

// createEntryPoint creates the main SCSS entry point file
func (p *BootstrapPlugin) createEntryPoint(config FrameworkConfig) error {
	entryDir := filepath.Dir(config.EntryPoint)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return fmt.Errorf("failed to create entry point directory: %w", err)
	}
	
	// Generate SCSS content
	scssContent := p.generateBootstrapSCSS(config)
	
	if err := os.WriteFile(config.EntryPoint, []byte(scssContent), 0644); err != nil {
		return fmt.Errorf("failed to write entry point file: %w", err)
	}
	
	return nil
}

// generateBootstrapSCSS generates the main Bootstrap SCSS file
func (p *BootstrapPlugin) generateBootstrapSCSS(config FrameworkConfig) string {
	var scss strings.Builder
	
	// Add custom variables if defined
	if len(config.Variables) > 0 {
		scss.WriteString("// Custom Bootstrap Variables\n")
		for key, value := range config.Variables {
			scss.WriteString(fmt.Sprintf("$%s: %s;\n", key, value))
		}
		scss.WriteString("\n")
	}
	
	// Import Bootstrap
	scss.WriteString("// Import Bootstrap\n")
	scss.WriteString("@import '~bootstrap/scss/bootstrap';\n\n")
	
	// Add custom styles section
	scss.WriteString("// Custom Styles\n")
	scss.WriteString("// Add your custom styles here\n")
	
	return scss.String()
}

// generateBasicBootstrapCSS generates basic Bootstrap CSS for standalone setup
func (p *BootstrapPlugin) generateBasicBootstrapCSS(config FrameworkConfig) string {
	// This is a very simplified version - in practice, you'd include full Bootstrap CSS
	css := `/* Bootstrap CSS Framework */
:root {
  --bs-blue: #0d6efd;
  --bs-indigo: #6610f2;
  --bs-purple: #6f42c1;
  --bs-pink: #d63384;
  --bs-red: #dc3545;
  --bs-orange: #fd7e14;
  --bs-yellow: #ffc107;
  --bs-green: #198754;
  --bs-teal: #20c997;
  --bs-cyan: #0dcaf0;
  --bs-primary: var(--bs-blue);
  --bs-secondary: #6c757d;
}

/* Container */
.container {
  max-width: 1320px;
  margin: 0 auto;
  padding: 0 15px;
}

/* Grid System */
.row {
  display: flex;
  flex-wrap: wrap;
  margin: 0 -15px;
}

.col {
  flex: 1;
  padding: 0 15px;
}

/* Buttons */
.btn {
  display: inline-block;
  padding: 0.375rem 0.75rem;
  font-size: 1rem;
  line-height: 1.5;
  border-radius: 0.25rem;
  border: 1px solid transparent;
  text-decoration: none;
  cursor: pointer;
}

.btn-primary {
  background-color: var(--bs-primary);
  border-color: var(--bs-primary);
  color: white;
}

/* Utilities */
.text-center { text-align: center; }
.text-left { text-align: left; }
.text-right { text-align: right; }
.d-none { display: none; }
.d-block { display: block; }
.d-flex { display: flex; }
`
	
	// Add custom variables to CSS
	for key, value := range config.Variables {
		css = strings.ReplaceAll(css, fmt.Sprintf("var(--bs-%s)", key), value)
	}
	
	return css
}

// GenerateConfig generates a Bootstrap configuration file
func (p *BootstrapPlugin) GenerateConfig(config FrameworkConfig) ([]byte, error) {
	configContent := fmt.Sprintf(`// Bootstrap Configuration
module.exports = {
  // Framework settings
  framework: '%s',
  version: '%s',
  
  // Build settings
  entry: '%s',
  output: '%s',
  
  // SCSS settings
  sassOptions: {
    includePaths: ['node_modules'],
    sourceMap: true
  },
  
  // Optimization
  optimization: {
    purge: %t,
    minify: %t,
    removeUnused: %t
  },
  
  // Custom variables
  variables: %s,
  
  // Bootstrap features
  features: {
    grid: %t,
    utilities: %t,
    print: %t,
    rtl: %t
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
		config.Options["enable_grid"].(bool),
		config.Options["enable_utilities"].(bool),
		config.Options["enable_print"].(bool),
		config.Options["rtl"].(bool),
	)
	
	return []byte(configContent), nil
}

// ValidateConfig validates Bootstrap configuration
func (p *BootstrapPlugin) ValidateConfig(configPath string) error {
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
		return fmt.Errorf("config file must export a configuration object")
	}
	
	return nil
}

// ProcessCSS processes CSS using Bootstrap
func (p *BootstrapPlugin) ProcessCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error) {
	// For Bootstrap, we primarily work with SCSS compilation
	if strings.Contains(options.InputPath, ".scss") {
		return p.compileSCSS(ctx, input, options)
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

// compileSCSS compiles SCSS to CSS using sass
func (p *BootstrapPlugin) compileSCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error) {
	// Create temporary input file
	tmpDir := os.TempDir()
	inputFile := filepath.Join(tmpDir, "input.scss")
	outputFile := filepath.Join(tmpDir, "output.css")
	
	if err := os.WriteFile(inputFile, input, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temporary input file: %w", err)
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)
	
	// Run sass compiler
	args := []string{inputFile, outputFile}
	if options.SourceMaps {
		args = append(args, "--source-map")
	}
	if options.Minify {
		args = append(args, "--style=compressed")
	}
	
	cmd := exec.CommandContext(ctx, "sass", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile SCSS: %w", err)
	}
	
	// Read output
	output, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled CSS: %w", err)
	}
	
	return output, nil
}

// optimizeCSS applies CSS optimizations
func (p *BootstrapPlugin) optimizeCSS(css []byte, options ProcessingOptions) ([]byte, error) {
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

// purgeUnusedClasses removes unused CSS classes
func (p *BootstrapPlugin) purgeUnusedClasses(css string, usedClasses []string) string {
	// Create a map for fast lookup
	usedMap := make(map[string]bool)
	for _, class := range usedClasses {
		usedMap[class] = true
	}
	
	// Simple purging - remove class rules that aren't used
	// This is a basic implementation - a full implementation would use a CSS parser
	
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

// shouldKeepRule determines if a CSS rule should be kept based on used classes
func (p *BootstrapPlugin) shouldKeepRule(rule string, usedClasses map[string]bool) bool {
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

// ExtractClasses extracts Bootstrap classes from content
func (p *BootstrapPlugin) ExtractClasses(content string) ([]string, error) {
	var classes []string
	classRegex := regexp.MustCompile(`class="([^"]*)"`)
	
	matches := classRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			classNames := strings.Fields(match[1])
			for _, className := range classNames {
				if p.isBootstrapClass(className) {
					classes = append(classes, className)
				}
			}
		}
	}
	
	return removeDuplicates(classes), nil
}

// isBootstrapClass checks if a class name is a Bootstrap class
func (p *BootstrapPlugin) isBootstrapClass(className string) bool {
	bootstrapPrefixes := []string{
		"btn", "container", "row", "col", "d-", "text-", "bg-", "border-",
		"p-", "m-", "px-", "py-", "mx-", "my-", "pt-", "pb-", "pl-", "pr-",
		"mt-", "mb-", "ml-", "mr-", "w-", "h-", "position-", "flex-",
		"justify-", "align-", "order-", "card", "nav", "navbar", "form",
		"input", "table", "badge", "alert", "progress", "spinner",
	}
	
	for _, prefix := range bootstrapPrefixes {
		if strings.HasPrefix(className, prefix) {
			return true
		}
	}
	
	return false
}

// OptimizeCSS optimizes CSS for Bootstrap
func (p *BootstrapPlugin) OptimizeCSS(ctx context.Context, css []byte, usedClasses []string) ([]byte, error) {
	options := ProcessingOptions{
		Purge:       true,
		Optimize:    true,
		UsedClasses: usedClasses,
		Environment: "production",
	}
	
	return p.optimizeCSS(css, options)
}

// ExtractVariables extracts CSS variables from Bootstrap CSS
func (p *BootstrapPlugin) ExtractVariables(css []byte) (map[string]string, error) {
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
	
	// Extract SCSS variables ($variable: value)
	scssVarRegex := regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_-]*)\s*:\s*([^;]+);`)
	scssMatches := scssVarRegex.FindAllStringSubmatch(string(css), -1)
	
	for _, match := range scssMatches {
		if len(match) > 2 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			variables[name] = value
		}
	}
	
	return variables, nil
}

// GenerateTheme generates a Bootstrap theme with custom variables
func (p *BootstrapPlugin) GenerateTheme(variables map[string]string) ([]byte, error) {
	var theme strings.Builder
	
	theme.WriteString("// Custom Bootstrap Theme\n")
	theme.WriteString("// Generated by Templar\n\n")
	
	// Add custom variables
	for name, value := range variables {
		theme.WriteString(fmt.Sprintf("$%s: %s;\n", name, value))
	}
	
	theme.WriteString("\n// Import Bootstrap\n")
	theme.WriteString("@import '~bootstrap/scss/bootstrap';\n")
	
	return []byte(theme.String()), nil
}

// GetDevServerConfig returns development server configuration
func (p *BootstrapPlugin) GetDevServerConfig() DevServerConfig {
	return DevServerConfig{
		HotReload:      true,
		WatchPaths:     []string{"src/**/*.scss", "src/**/*.css"},
		ReloadDelay:    300,
		CSSInjection:   true,
		InjectTarget:   "head",
		ErrorOverlay:   true,
		SourceMaps:     true,
		LiveValidation: true,
		DevMode:        true,
		DevOptions: map[string]interface{}{
			"sass_source_maps": true,
			"css_autoprefixer": true,
		},
	}
}

// SupportsHotReload returns true if the framework supports hot reload
func (p *BootstrapPlugin) SupportsHotReload() bool {
	return true
}

// GenerateStyleGuide generates a Bootstrap style guide
func (p *BootstrapPlugin) GenerateStyleGuide(ctx context.Context) ([]byte, error) {
	styleGuide := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bootstrap Style Guide</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
    <div class="container py-5">
        <h1 class="mb-4">Bootstrap Style Guide</h1>
        
        <section class="mb-5">
            <h2>Colors</h2>
            <div class="row">
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-primary text-white text-center">Primary</div>
                </div>
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-secondary text-white text-center">Secondary</div>
                </div>
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-success text-white text-center">Success</div>
                </div>
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-danger text-white text-center">Danger</div>
                </div>
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-warning text-dark text-center">Warning</div>
                </div>
                <div class="col-md-2 mb-3">
                    <div class="p-3 bg-info text-dark text-center">Info</div>
                </div>
            </div>
        </section>
        
        <section class="mb-5">
            <h2>Buttons</h2>
            <div class="mb-3">
                <button class="btn btn-primary me-2">Primary</button>
                <button class="btn btn-secondary me-2">Secondary</button>
                <button class="btn btn-success me-2">Success</button>
                <button class="btn btn-danger me-2">Danger</button>
                <button class="btn btn-warning me-2">Warning</button>
                <button class="btn btn-info me-2">Info</button>
            </div>
        </section>
        
        <section class="mb-5">
            <h2>Typography</h2>
            <h1>Heading 1</h1>
            <h2>Heading 2</h2>
            <h3>Heading 3</h3>
            <h4>Heading 4</h4>
            <h5>Heading 5</h5>
            <h6>Heading 6</h6>
            <p>This is a paragraph with <strong>bold text</strong> and <em>italic text</em>.</p>
        </section>
        
        <section class="mb-5">
            <h2>Grid System</h2>
            <div class="row">
                <div class="col-md-4">
                    <div class="p-3 bg-light border">Column 1</div>
                </div>
                <div class="col-md-4">
                    <div class="p-3 bg-light border">Column 2</div>
                </div>
                <div class="col-md-4">
                    <div class="p-3 bg-light border">Column 3</div>
                </div>
            </div>
        </section>
    </div>
</body>
</html>`
	
	return []byte(styleGuide), nil
}

// Helper functions

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// formatVariablesAsJS formats variables as JavaScript object
func formatVariablesAsJS(variables map[string]string) string {
	var parts []string
	for key, value := range variables {
		parts = append(parts, fmt.Sprintf("    '%s': '%s'", key, value))
	}
	return "{\n" + strings.Join(parts, ",\n") + "\n  }"
}

// getBootstrapTemplates returns built-in Bootstrap component templates
func getBootstrapTemplates() []ComponentTemplate {
	return []ComponentTemplate{
		{
			Name:        "Button",
			Framework:   "bootstrap",
			Category:    "button",
			Description: "Bootstrap button component",
			Template: `templ Button(text string, variant string) {
	<button type="button" class={ "btn", "btn-" + variant }>
		{ text }
	</button>
}`,
			Props: []types.ParameterInfo{
				{Name: "text", Type: "string", Optional: false},
				{Name: "variant", Type: "string", Optional: false},
			},
			Examples: []TemplateExample{
				{
					Name:        "Primary Button",
					Description: "A primary button example",
					Props: map[string]interface{}{
						"text":    "Click Me",
						"variant": "primary",
					},
				},
			},
			Classes: []string{"btn", "btn-primary", "btn-secondary", "btn-success"},
		},
		{
			Name:        "Card",
			Framework:   "bootstrap",
			Category:    "layout",
			Description: "Bootstrap card component",
			Template: `templ Card(title string, content string) {
	<div class="card">
		<div class="card-body">
			<h5 class="card-title">{ title }</h5>
			<p class="card-text">{ content }</p>
		</div>
	</div>
}`,
			Props: []types.ParameterInfo{
				{Name: "title", Type: "string", Optional: false},
				{Name: "content", Type: "string", Optional: false},
			},
			Classes: []string{"card", "card-body", "card-title", "card-text"},
		},
	}
}