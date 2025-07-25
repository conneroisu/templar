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

// TailwindPlugin implements CSSFrameworkPlugin for Tailwind CSS framework
type TailwindPlugin struct {
	name         string
	version      string
	config       map[string]interface{}
	tailwindPath string
}

// NewTailwindPlugin creates a new Tailwind plugin instance
func NewTailwindPlugin() *TailwindPlugin {
	return &TailwindPlugin{
		name:    "tailwind",
		version: "1.0.0",
		config:  make(map[string]interface{}),
	}
}

// GetName returns the plugin name
func (p *TailwindPlugin) GetName() string {
	return p.name
}

// GetVersion returns the plugin version
func (p *TailwindPlugin) GetVersion() string {
	return p.version
}

// Initialize initializes the Tailwind plugin
func (p *TailwindPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.config = config

	// Check if Tailwind CLI is available
	tailwindPath, err := exec.LookPath("tailwindcss")
	if err != nil {
		// Try npx
		if _, err := exec.LookPath("npx"); err == nil {
			p.tailwindPath = "npx tailwindcss"
		} else {
			return fmt.Errorf("tailwindcss not found in PATH and npx not available")
		}
	} else {
		p.tailwindPath = tailwindPath
	}

	return nil
}

// Cleanup cleans up plugin resources
func (p *TailwindPlugin) Cleanup() error {
	return nil
}

// GetFrameworkName returns the framework name
func (p *TailwindPlugin) GetFrameworkName() string {
	return "tailwind"
}

// GetSupportedVersions returns supported Tailwind versions
func (p *TailwindPlugin) GetSupportedVersions() []string {
	return []string{"3.4.0", "3.3.6", "3.3.5", "3.3.4", "3.3.3", "3.3.2", "3.3.1", "3.3.0"}
}

// GetDefaultConfig returns default configuration for Tailwind
func (p *TailwindPlugin) GetDefaultConfig() FrameworkConfig {
	return FrameworkConfig{
		Name:          "tailwind",
		Version:       "3.4.0",
		InstallMethod: "npm",
		ConfigFile:    "tailwind.config.js",
		EntryPoint:    "src/input.css",
		OutputPath:    "dist/styles.css",
		SourcePaths:   []string{"src/**/*.{templ,html,js,ts}"},

		Preprocessing: []string{"postcss"},

		Optimization: OptimizationConfig{
			Enabled:      true,
			Purge:        true,
			Minify:       true,
			TreeShake:    true,
			Compress:     true,
			RemoveUnused: true,
		},

		Theming: ThemingConfig{
			Enabled:          true,
			ExtractVariables: true,
			GenerateTokens:   true,
			StyleGuide:       true,
			OutputFormat:     "css",
			OutputFile:       "src/theme.css",
		},

		Variables: map[string]string{
			// Default Tailwind theme colors
			"primary":   "#3b82f6",
			"secondary": "#6b7280",
			"accent":    "#f59e0b",
			"neutral":   "#374151",
			"base-100":  "#ffffff",
			"info":      "#0ea5e9",
			"success":   "#10b981",
			"warning":   "#f59e0b",
			"error":     "#ef4444",
		},

		Options: map[string]interface{}{
			"jit":           true,
			"purge_enabled": true,
			"dark_mode":     "class",
			"important":     false,
			"prefix":        "",
			"separator":     ":",
		},
	}
}

// IsInstalled checks if Tailwind is installed
func (p *TailwindPlugin) IsInstalled() bool {
	// Check for npm package
	if _, err := os.Stat("node_modules/tailwindcss"); err == nil {
		return true
	}

	// Check for standalone binary
	if _, err := exec.LookPath("tailwindcss"); err == nil {
		return true
	}

	// Check for config file
	configPaths := []string{
		"tailwind.config.js",
		"tailwind.config.ts",
		"tailwind.config.cjs",
		"tailwind.config.mjs",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// Setup sets up Tailwind with the given configuration
func (p *TailwindPlugin) Setup(ctx context.Context, config FrameworkConfig) error {
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

// setupWithNPM sets up Tailwind using npm
func (p *TailwindPlugin) setupWithNPM(ctx context.Context, config FrameworkConfig) error {
	// Install Tailwind via npm
	cmd := exec.CommandContext(ctx, "npm", "install", "-D", fmt.Sprintf("tailwindcss@%s", config.Version))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Tailwind via npm: %w", err)
	}

	// Install PostCSS and autoprefixer if needed
	if contains(config.Preprocessing, "postcss") {
		cmd = exec.CommandContext(ctx, "npm", "install", "-D", "postcss", "autoprefixer")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install PostCSS: %w", err)
		}
	}

	// Create entry point CSS file
	if err := p.createEntryPoint(config); err != nil {
		return fmt.Errorf("failed to create entry point: %w", err)
	}

	// Initialize Tailwind config
	cmd = exec.CommandContext(ctx, "npx", "tailwindcss", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize Tailwind config: %w", err)
	}

	return nil
}

// setupWithCDN sets up Tailwind using CDN
func (p *TailwindPlugin) setupWithCDN(ctx context.Context, config FrameworkConfig) error {
	cdnUrl := config.CDNUrl
	if cdnUrl == "" {
		cdnUrl = fmt.Sprintf("https://cdn.tailwindcss.com/%s", config.Version)
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

// setupStandalone sets up Tailwind as standalone
func (p *TailwindPlugin) setupStandalone(ctx context.Context, config FrameworkConfig) error {
	// Download Tailwind CLI binary (simplified - in practice, you'd download from GitHub releases)
	outputDir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create basic Tailwind CSS (simplified)
	basicCSS := p.generateBasicTailwindCSS(config)
	if err := os.WriteFile(config.OutputPath, []byte(basicCSS), 0644); err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}

	return nil
}

// createEntryPoint creates the main CSS entry point file
func (p *TailwindPlugin) createEntryPoint(config FrameworkConfig) error {
	entryDir := filepath.Dir(config.EntryPoint)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return fmt.Errorf("failed to create entry point directory: %w", err)
	}

	// Generate CSS content
	cssContent := p.generateTailwindCSS(config)

	if err := os.WriteFile(config.EntryPoint, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("failed to write entry point file: %w", err)
	}

	return nil
}

// generateTailwindCSS generates the main Tailwind CSS file
func (p *TailwindPlugin) generateTailwindCSS(config FrameworkConfig) string {
	var css strings.Builder

	// Add Tailwind directives
	css.WriteString("@tailwind base;\n")
	css.WriteString("@tailwind components;\n")
	css.WriteString("@tailwind utilities;\n\n")

	// Add custom CSS layer if variables are defined
	if len(config.Variables) > 0 {
		css.WriteString("@layer base {\n")
		css.WriteString("  :root {\n")
		for key, value := range config.Variables {
			css.WriteString(fmt.Sprintf("    --%s: %s;\n", key, value))
		}
		css.WriteString("  }\n")
		css.WriteString("}\n\n")
	}

	// Add custom components layer
	css.WriteString("@layer components {\n")
	css.WriteString("  /* Custom component styles */\n")
	css.WriteString("}\n\n")

	// Add custom utilities layer
	css.WriteString("@layer utilities {\n")
	css.WriteString("  /* Custom utility styles */\n")
	css.WriteString("}\n")

	return css.String()
}

// generateBasicTailwindCSS generates basic Tailwind CSS for standalone setup
func (p *TailwindPlugin) generateBasicTailwindCSS(config FrameworkConfig) string {
	css := `/* Tailwind CSS Framework */

/* Reset */
*, ::before, ::after {
  box-sizing: border-box;
  border-width: 0;
  border-style: solid;
  border-color: #e5e7eb;
}

/* Base styles */
html {
  line-height: 1.5;
  -webkit-text-size-adjust: 100%;
  -moz-tab-size: 4;
  tab-size: 4;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif;
}

/* Container */
.container {
  width: 100%;
  margin-right: auto;
  margin-left: auto;
  padding-right: 1rem;
  padding-left: 1rem;
}

/* Flexbox */
.flex { display: flex; }
.inline-flex { display: inline-flex; }
.flex-col { flex-direction: column; }
.flex-row { flex-direction: row; }
.flex-wrap { flex-wrap: wrap; }
.items-center { align-items: center; }
.justify-center { justify-content: center; }
.justify-between { justify-content: space-between; }

/* Grid */
.grid { display: grid; }
.grid-cols-1 { grid-template-columns: repeat(1, minmax(0, 1fr)); }
.grid-cols-2 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.grid-cols-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.grid-cols-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }

/* Spacing */
.p-0 { padding: 0; }
.p-1 { padding: 0.25rem; }
.p-2 { padding: 0.5rem; }
.p-4 { padding: 1rem; }
.p-8 { padding: 2rem; }
.m-0 { margin: 0; }
.m-1 { margin: 0.25rem; }
.m-2 { margin: 0.5rem; }
.m-4 { margin: 1rem; }
.m-8 { margin: 2rem; }

/* Typography */
.text-xs { font-size: 0.75rem; line-height: 1rem; }
.text-sm { font-size: 0.875rem; line-height: 1.25rem; }
.text-base { font-size: 1rem; line-height: 1.5rem; }
.text-lg { font-size: 1.125rem; line-height: 1.75rem; }
.text-xl { font-size: 1.25rem; line-height: 1.75rem; }
.text-2xl { font-size: 1.5rem; line-height: 2rem; }
.text-3xl { font-size: 1.875rem; line-height: 2.25rem; }

.font-normal { font-weight: 400; }
.font-medium { font-weight: 500; }
.font-semibold { font-weight: 600; }
.font-bold { font-weight: 700; }

.text-center { text-align: center; }
.text-left { text-align: left; }
.text-right { text-align: right; }

/* Colors */
.text-gray-900 { color: #111827; }
.text-gray-700 { color: #374151; }
.text-gray-500 { color: #6b7280; }
.text-white { color: #ffffff; }

.bg-white { background-color: #ffffff; }
.bg-gray-100 { background-color: #f3f4f6; }
.bg-gray-500 { background-color: #6b7280; }
.bg-blue-500 { background-color: #3b82f6; }
.bg-green-500 { background-color: #10b981; }
.bg-red-500 { background-color: #ef4444; }

/* Borders */
.border { border-width: 1px; }
.border-2 { border-width: 2px; }
.border-gray-300 { border-color: #d1d5db; }
.rounded { border-radius: 0.25rem; }
.rounded-md { border-radius: 0.375rem; }
.rounded-lg { border-radius: 0.5rem; }

/* Shadows */
.shadow { box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06); }
.shadow-md { box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06); }
.shadow-lg { box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05); }

/* Width and Height */
.w-full { width: 100%; }
.w-1/2 { width: 50%; }
.w-1/3 { width: 33.333333%; }
.w-1/4 { width: 25%; }
.h-full { height: 100%; }
.h-screen { height: 100vh; }

/* Display */
.block { display: block; }
.inline { display: inline; }
.inline-block { display: inline-block; }
.hidden { display: none; }

/* Position */
.relative { position: relative; }
.absolute { position: absolute; }
.fixed { position: fixed; }
.sticky { position: sticky; }
`

	// Apply custom variables to CSS if they exist
	for key, value := range config.Variables {
		// Replace CSS custom properties with actual values
		css = strings.ReplaceAll(css, fmt.Sprintf("var(--%s)", key), value)
	}

	return css
}

// GenerateConfig generates a Tailwind configuration file
func (p *TailwindPlugin) GenerateConfig(config FrameworkConfig) ([]byte, error) {
	configContent := fmt.Sprintf(`/** @type {import('tailwindcss').Config} */
module.exports = {
  content: %s,
  theme: {
    extend: {
      colors: %s,
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui'],
      },
    },
  },
  plugins: [],
  darkMode: '%s',
  important: %t,
  prefix: '%s',
  separator: '%s',
  corePlugins: {
    preflight: true,
  },
  experimental: {
    optimizeUniversalDefaults: true,
  },
}
`,
		formatArrayAsJS(config.SourcePaths),
		formatVariablesAsJS(config.Variables),
		config.Options["dark_mode"].(string),
		config.Options["important"].(bool),
		config.Options["prefix"].(string),
		config.Options["separator"].(string),
	)

	return []byte(configContent), nil
}

// ValidateConfig validates Tailwind configuration
func (p *TailwindPlugin) ValidateConfig(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read and validate config file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Basic validation - check for required exports
	if !strings.Contains(string(content), "module.exports") && !strings.Contains(string(content), "export default") {
		return fmt.Errorf("config file must export a configuration object")
	}

	return nil
}

// ProcessCSS processes CSS using Tailwind
func (p *TailwindPlugin) ProcessCSS(ctx context.Context, input []byte, options ProcessingOptions) ([]byte, error) {
	// Create temporary input file
	tmpDir := os.TempDir()
	inputFile := filepath.Join(tmpDir, "input.css")
	outputFile := filepath.Join(tmpDir, "output.css")

	if err := os.WriteFile(inputFile, input, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temporary input file: %w", err)
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)

	// Build Tailwind command
	var cmd *exec.Cmd
	if strings.Contains(p.tailwindPath, "npx") {
		args := []string{"npx", "tailwindcss", "-i", inputFile, "-o", outputFile}
		if options.Minify {
			args = append(args, "--minify")
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	} else {
		args := []string{"-i", inputFile, "-o", outputFile}
		if options.Minify {
			args = append(args, "--minify")
		}
		cmd = exec.CommandContext(ctx, p.tailwindPath, args...)
	}

	// Run Tailwind CSS generation
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tailwind CSS generation failed: %w", err)
	}

	// Read output
	output, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled CSS: %w", err)
	}

	return output, nil
}

// ExtractClasses extracts Tailwind classes from content
func (p *TailwindPlugin) ExtractClasses(content string) ([]string, error) {
	var classes []string

	// Extract from class attributes
	classRegex := regexp.MustCompile(`class="([^"]*)"`)
	matches := classRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			classNames := strings.Fields(match[1])
			for _, className := range classNames {
				if p.isTailwindClass(className) {
					classes = append(classes, className)
				}
			}
		}
	}

	// Extract from class={ } expressions (templ syntax)
	templClassRegex := regexp.MustCompile(`class=\{[^}]*"([^"]*)"[^}]*\}`)
	templMatches := templClassRegex.FindAllStringSubmatch(content, -1)

	for _, match := range templMatches {
		if len(match) > 1 {
			classNames := strings.Fields(match[1])
			for _, className := range classNames {
				if p.isTailwindClass(className) {
					classes = append(classes, className)
				}
			}
		}
	}

	return removeDuplicates(classes), nil
}

// isTailwindClass checks if a class name is a Tailwind class
func (p *TailwindPlugin) isTailwindClass(className string) bool {
	// Tailwind utility prefixes
	tailwindPrefixes := []string{
		"bg-", "text-", "p-", "m-", "w-", "h-", "flex", "grid", "rounded", "shadow",
		"border", "font-", "leading-", "tracking-", "space-", "divide-", "transform",
		"transition", "duration-", "ease-", "scale-", "rotate-", "translate-", "skew-",
		"origin-", "opacity-", "cursor-", "select-", "resize-", "outline-", "ring-",
		"filter", "blur-", "brightness-", "contrast-", "grayscale", "hue-rotate-",
		"invert", "saturate-", "sepia", "backdrop-", "absolute", "relative", "fixed",
		"sticky", "inset-", "top-", "right-", "bottom-", "left-", "z-", "visible",
		"invisible", "collapse", "table", "inline-", "block", "hidden", "overflow-",
		"object-", "clear-", "float-", "box-", "display-", "aspect-", "columns-",
		"break-", "decoration-", "underline-", "list-", "placeholder-", "caret-",
		"accent-", "appearance-", "pointer-", "resize", "scroll-", "snap-", "touch-",
		"user-", "will-", "content-",
	}

	for _, prefix := range tailwindPrefixes {
		if strings.HasPrefix(className, prefix) {
			return true
		}
	}

	// Check for responsive prefixes
	responsivePrefixes := []string{"sm:", "md:", "lg:", "xl:", "2xl:"}
	for _, prefix := range responsivePrefixes {
		if strings.HasPrefix(className, prefix) {
			rest := className[len(prefix):]
			return p.isTailwindClass(rest)
		}
	}

	// Check for state prefixes
	statePrefixes := []string{
		"hover:", "focus:", "active:", "visited:", "target:", "first:", "last:",
		"odd:", "even:", "disabled:", "enabled:", "checked:", "indeterminate:",
		"default:", "required:", "valid:", "invalid:", "in-range:", "out-of-range:",
		"placeholder-shown:", "autofill:", "read-only:", "group-hover:", "group-focus:",
		"group-active:", "group-visited:", "group-target:", "group-first:", "group-last:",
		"group-odd:", "group-even:", "group-disabled:", "group-enabled:", "group-checked:",
		"motion-safe:", "motion-reduce:", "dark:", "portrait:", "landscape:", "contrast-more:",
		"contrast-less:", "print:",
	}

	for _, prefix := range statePrefixes {
		if strings.HasPrefix(className, prefix) {
			rest := className[len(prefix):]
			return p.isTailwindClass(rest)
		}
	}

	// Check for exact utility matches
	utilities := []string{
		"container", "prose", "not-prose", "sr-only", "not-sr-only",
	}

	for _, utility := range utilities {
		if className == utility {
			return true
		}
	}

	return false
}

// OptimizeCSS optimizes CSS for Tailwind
func (p *TailwindPlugin) OptimizeCSS(ctx context.Context, css []byte, usedClasses []string) ([]byte, error) {
	// For Tailwind, optimization is typically done during the build process
	// We can implement purging here if needed

	if len(usedClasses) == 0 {
		return css, nil
	}

	// Simple purging implementation
	cssStr := string(css)

	// Remove comments
	cssStr = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(cssStr, "")

	// Remove extra whitespace
	cssStr = regexp.MustCompile(`\s+`).ReplaceAllString(cssStr, " ")
	cssStr = strings.TrimSpace(cssStr)

	return []byte(cssStr), nil
}

// ExtractVariables extracts CSS variables from Tailwind CSS
func (p *TailwindPlugin) ExtractVariables(css []byte) (map[string]string, error) {
	variables := make(map[string]string)

	// Extract CSS custom properties
	varRegex := regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9_-]*)\s*:\s*([^;]+);`)
	matches := varRegex.FindAllStringSubmatch(string(css), -1)

	for _, match := range matches {
		if len(match) > 2 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			variables[name] = value
		}
	}

	return variables, nil
}

// GenerateTheme generates a Tailwind theme with custom variables
func (p *TailwindPlugin) GenerateTheme(variables map[string]string) ([]byte, error) {
	var theme strings.Builder

	theme.WriteString("@tailwind base;\n")
	theme.WriteString("@tailwind components;\n")
	theme.WriteString("@tailwind utilities;\n\n")

	theme.WriteString("@layer base {\n")
	theme.WriteString("  :root {\n")

	// Add custom variables
	for name, value := range variables {
		theme.WriteString(fmt.Sprintf("    --%s: %s;\n", name, value))
	}

	theme.WriteString("  }\n")
	theme.WriteString("}\n")

	return []byte(theme.String()), nil
}

// GetDevServerConfig returns development server configuration
func (p *TailwindPlugin) GetDevServerConfig() DevServerConfig {
	return DevServerConfig{
		HotReload:      true,
		WatchPaths:     []string{"src/**/*.css", "tailwind.config.js"},
		ReloadDelay:    100, // Fast reload for Tailwind
		CSSInjection:   true,
		InjectTarget:   "head",
		ErrorOverlay:   true,
		SourceMaps:     true,
		LiveValidation: true,
		DevMode:        true,
		DevOptions: map[string]interface{}{
			"watch_mode":    true,
			"jit_mode":      true,
			"purge_enabled": false, // Disable purging in dev mode
		},
	}
}

// SupportsHotReload returns true if the framework supports hot reload
func (p *TailwindPlugin) SupportsHotReload() bool {
	return true
}

// GenerateStyleGuide generates a Tailwind style guide
func (p *TailwindPlugin) GenerateStyleGuide(ctx context.Context) ([]byte, error) {
	styleGuide := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tailwind CSS Style Guide</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 p-8">
    <div class="max-w-6xl mx-auto">
        <h1 class="text-4xl font-bold text-gray-900 mb-8">Tailwind CSS Style Guide</h1>
        
        <!-- Colors -->
        <section class="mb-12">
            <h2 class="text-2xl font-semibold text-gray-800 mb-6">Colors</h2>
            <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                <div class="text-center">
                    <div class="w-16 h-16 bg-blue-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Blue 500</p>
                </div>
                <div class="text-center">
                    <div class="w-16 h-16 bg-red-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Red 500</p>
                </div>
                <div class="text-center">
                    <div class="w-16 h-16 bg-green-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Green 500</p>
                </div>
                <div class="text-center">
                    <div class="w-16 h-16 bg-yellow-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Yellow 500</p>
                </div>
                <div class="text-center">
                    <div class="w-16 h-16 bg-purple-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Purple 500</p>
                </div>
                <div class="text-center">
                    <div class="w-16 h-16 bg-gray-500 rounded-lg mx-auto mb-2"></div>
                    <p class="text-sm font-medium">Gray 500</p>
                </div>
            </div>
        </section>
        
        <!-- Typography -->
        <section class="mb-12">
            <h2 class="text-2xl font-semibold text-gray-800 mb-6">Typography</h2>
            <div class="space-y-4">
                <h1 class="text-6xl font-thin">Heading 1</h1>
                <h2 class="text-5xl font-light">Heading 2</h2>
                <h3 class="text-4xl font-normal">Heading 3</h3>
                <h4 class="text-3xl font-medium">Heading 4</h4>
                <h5 class="text-2xl font-semibold">Heading 5</h5>
                <h6 class="text-xl font-bold">Heading 6</h6>
                <p class="text-base text-gray-700">This is a paragraph with normal text.</p>
                <p class="text-sm text-gray-600">This is smaller text.</p>
            </div>
        </section>
        
        <!-- Buttons -->
        <section class="mb-12">
            <h2 class="text-2xl font-semibold text-gray-800 mb-6">Buttons</h2>
            <div class="flex flex-wrap gap-4">
                <button class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Primary</button>
                <button class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600">Secondary</button>
                <button class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">Success</button>
                <button class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600">Danger</button>
                <button class="px-4 py-2 bg-yellow-500 text-black rounded hover:bg-yellow-600">Warning</button>
                <button class="px-4 py-2 border border-gray-300 text-gray-700 rounded hover:bg-gray-50">Outline</button>
            </div>
        </section>
        
        <!-- Cards -->
        <section class="mb-12">
            <h2 class="text-2xl font-semibold text-gray-800 mb-6">Cards</h2>
            <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                <div class="bg-white rounded-lg shadow-md p-6">
                    <h3 class="text-lg font-semibold mb-2">Card Title</h3>
                    <p class="text-gray-600 mb-4">This is a card description with some sample content.</p>
                    <button class="px-4 py-2 bg-blue-500 text-white rounded text-sm hover:bg-blue-600">Action</button>
                </div>
                <div class="bg-white rounded-lg shadow-lg p-6">
                    <h3 class="text-lg font-semibold mb-2">Card with Shadow</h3>
                    <p class="text-gray-600 mb-4">This card has a larger shadow for more emphasis.</p>
                    <button class="px-4 py-2 bg-green-500 text-white rounded text-sm hover:bg-green-600">Action</button>
                </div>
                <div class="bg-gradient-to-r from-purple-400 to-pink-400 text-white rounded-lg p-6">
                    <h3 class="text-lg font-semibold mb-2">Gradient Card</h3>
                    <p class="mb-4">This card uses a gradient background.</p>
                    <button class="px-4 py-2 bg-white text-purple-600 rounded text-sm hover:bg-gray-100">Action</button>
                </div>
            </div>
        </section>
        
        <!-- Layout -->
        <section class="mb-12">
            <h2 class="text-2xl font-semibold text-gray-800 mb-6">Layout</h2>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <div class="bg-blue-100 p-4 rounded">Column 1</div>
                <div class="bg-green-100 p-4 rounded">Column 2</div>
                <div class="bg-yellow-100 p-4 rounded">Column 3</div>
            </div>
        </section>
    </div>
</body>
</html>`

	return []byte(styleGuide), nil
}

// Helper functions

// formatArrayAsJS formats a string array as JavaScript array
func formatArrayAsJS(arr []string) string {
	var quoted []string
	for _, item := range arr {
		quoted = append(quoted, fmt.Sprintf("'%s'", item))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// getTailwindTemplates returns built-in Tailwind component templates
func getTailwindTemplates() []ComponentTemplate {
	return []ComponentTemplate{
		{
			Name:        "Button",
			Framework:   "tailwind",
			Category:    "button",
			Description: "Tailwind button component",
			Template: `templ Button(text string, variant string) {
	<button class={ "px-4", "py-2", "rounded", "font-medium", "transition-colors", 
		if variant == "primary" { "bg-blue-500 text-white hover:bg-blue-600" } 
		else if variant == "secondary" { "bg-gray-500 text-white hover:bg-gray-600" }
		else { "bg-gray-200 text-gray-800 hover:bg-gray-300" }
	}>
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
			Classes: []string{"px-4", "py-2", "rounded", "font-medium", "bg-blue-500", "text-white", "hover:bg-blue-600"},
		},
		{
			Name:        "Card",
			Framework:   "tailwind",
			Category:    "layout",
			Description: "Tailwind card component",
			Template: `templ Card(title string, content string) {
	<div class="bg-white rounded-lg shadow-md p-6">
		<h3 class="text-lg font-semibold mb-2">{ title }</h3>
		<p class="text-gray-600">{ content }</p>
	</div>
}`,
			Props: []types.ParameterInfo{
				{Name: "title", Type: "string", Optional: false},
				{Name: "content", Type: "string", Optional: false},
			},
			Classes: []string{"bg-white", "rounded-lg", "shadow-md", "p-6", "text-lg", "font-semibold", "mb-2", "text-gray-600"},
		},
	}
}
