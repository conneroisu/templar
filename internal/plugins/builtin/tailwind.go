package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/plugins"
	"github.com/conneroisu/templar/internal/registry"
)

// TailwindPlugin provides Tailwind CSS integration for Templar components
type TailwindPlugin struct {
	config       plugins.PluginConfig
	tailwindPath string
	configPath   string
	enabled      bool
}

// NewTailwindPlugin creates a new Tailwind CSS plugin
func NewTailwindPlugin() *TailwindPlugin {
	return &TailwindPlugin{
		enabled: true,
	}
}

// Name returns the plugin name
func (tp *TailwindPlugin) Name() string {
	return "tailwind"
}

// Version returns the plugin version
func (tp *TailwindPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (tp *TailwindPlugin) Description() string {
	return "Tailwind CSS integration for automatic class detection and CSS generation"
}

// Initialize initializes the Tailwind plugin
func (tp *TailwindPlugin) Initialize(ctx context.Context, config plugins.PluginConfig) error {
	tp.config = config
	
	// Check if Tailwind CLI is available
	tailwindPath, err := exec.LookPath("tailwindcss")
	if err != nil {
		// Try npx
		if _, err := exec.LookPath("npx"); err == nil {
			tp.tailwindPath = "npx tailwindcss"
		} else {
			return fmt.Errorf("tailwindcss not found in PATH and npx not available")
		}
	} else {
		tp.tailwindPath = tailwindPath
	}
	
	// Look for Tailwind config
	configPaths := []string{
		"tailwind.config.js",
		"tailwind.config.ts",
		"tailwind.config.cjs",
		"tailwind.config.mjs",
	}
	
	for _, path := range configPaths {
		if _, err := exec.Command("test", "-f", path).Output(); err == nil {
			tp.configPath = path
			break
		}
	}
	
	return nil
}

// Shutdown shuts down the plugin
func (tp *TailwindPlugin) Shutdown(ctx context.Context) error {
	tp.enabled = false
	return nil
}

// Health returns the plugin health status
func (tp *TailwindPlugin) Health() plugins.PluginHealth {
	status := plugins.HealthStatusHealthy
	var errorMsg string
	
	if !tp.enabled {
		status = plugins.HealthStatusUnhealthy
		errorMsg = "plugin disabled"
	} else if tp.tailwindPath == "" {
		status = plugins.HealthStatusUnhealthy
		errorMsg = "tailwindcss not found"
	}
	
	return plugins.PluginHealth{
		Status:    status,
		LastCheck: time.Now(),
		Error:     errorMsg,
		Metrics: map[string]interface{}{
			"tailwind_path": tp.tailwindPath,
			"config_path":   tp.configPath,
		},
	}
}

// HandleComponent processes components to extract Tailwind classes
func (tp *TailwindPlugin) HandleComponent(ctx context.Context, component *registry.ComponentInfo) (*registry.ComponentInfo, error) {
	if !tp.enabled {
		return component, nil
	}
	
	// Read component file to extract Tailwind classes
	classes, err := tp.extractTailwindClasses(component.FilePath)
	if err != nil {
		return component, fmt.Errorf("failed to extract Tailwind classes: %w", err)
	}
	
	// Add metadata about Tailwind classes
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	
	component.Metadata["tailwind_classes"] = classes
	component.Metadata["tailwind_processed"] = true
	
	return component, nil
}

// SupportedExtensions returns the file extensions this plugin handles
func (tp *TailwindPlugin) SupportedExtensions() []string {
	return []string{".templ", ".html", ".tsx", ".jsx"}
}

// Priority returns the execution priority
func (tp *TailwindPlugin) Priority() int {
	return 10 // Lower priority to run after basic processing
}

// PreBuild is called before the build process starts
func (tp *TailwindPlugin) PreBuild(ctx context.Context, components []*registry.ComponentInfo) error {
	// Nothing to do before build for Tailwind
	return nil
}

// PostBuild generates CSS after components are built
func (tp *TailwindPlugin) PostBuild(ctx context.Context, components []*registry.ComponentInfo, buildResult plugins.BuildResult) error {
	if !tp.enabled || !buildResult.Success {
		return nil
	}
	
	// Collect all Tailwind classes from processed components
	allClasses := make(map[string]bool)
	for _, component := range components {
		if classes, ok := component.Metadata["tailwind_classes"].([]string); ok {
			for _, class := range classes {
				allClasses[class] = true
			}
		}
	}
	
	if len(allClasses) == 0 {
		return nil // No Tailwind classes found
	}
	
	// Generate CSS using Tailwind CLI
	return tp.generateCSS(ctx, allClasses)
}

// TransformBuildCommand allows modifying the build command
func (tp *TailwindPlugin) TransformBuildCommand(ctx context.Context, command []string) ([]string, error) {
	// For Tailwind, we don't need to modify the build command
	return command, nil
}

// extractTailwindClasses extracts Tailwind CSS classes from a component file
func (tp *TailwindPlugin) extractTailwindClasses(filePath string) ([]string, error) {
	// This is a simplified implementation
	// In practice, you'd want a more sophisticated parser
	
	content, err := exec.Command("cat", filePath).Output()
	if err != nil {
		return nil, err
	}
	
	var classes []string
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		// Look for class attributes
		if strings.Contains(line, "class=") {
			// Extract classes using simple string manipulation
			// This could be improved with proper HTML/template parsing
			start := strings.Index(line, `class="`)
			if start != -1 {
				start += 7 // Length of `class="`
				end := strings.Index(line[start:], `"`)
				if end != -1 {
					classStr := line[start : start+end]
					classNames := strings.Fields(classStr)
					
					// Filter for Tailwind-like classes
					for _, className := range classNames {
						if tp.isTailwindClass(className) {
							classes = append(classes, className)
						}
					}
				}
			}
		}
	}
	
	return tp.deduplicate(classes), nil
}

// isTailwindClass checks if a class name looks like a Tailwind class
func (tp *TailwindPlugin) isTailwindClass(className string) bool {
	// Simple heuristics for Tailwind classes
	tailwindPrefixes := []string{
		"bg-", "text-", "p-", "m-", "w-", "h-", "flex", "grid",
		"rounded", "shadow", "border", "font-", "leading-",
		"tracking-", "space-", "divide-", "transform", "transition",
		"duration-", "ease-", "scale-", "rotate-", "translate-",
		"skew-", "origin-", "opacity-", "cursor-", "select-",
		"resize-", "outline-", "ring-", "filter", "blur-",
		"brightness-", "contrast-", "grayscale", "hue-rotate-",
		"invert", "saturate-", "sepia", "backdrop-",
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
			// Check if the rest is a Tailwind class
			rest := className[len(prefix):]
			return tp.isTailwindClass(rest)
		}
	}
	
	// Check for state prefixes
	statePrefixes := []string{"hover:", "focus:", "active:", "group-hover:", "group-focus:"}
	for _, prefix := range statePrefixes {
		if strings.HasPrefix(className, prefix) {
			rest := className[len(prefix):]
			return tp.isTailwindClass(rest)
		}
	}
	
	return false
}

// generateCSS generates CSS using Tailwind CLI
func (tp *TailwindPlugin) generateCSS(ctx context.Context, classes map[string]bool) error {
	// Create temporary input CSS file
	inputCSS := "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"
	
	// Write temporary file
	tempFile := filepath.Join("/tmp", "templar-tailwind-input.css")
	if err := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' > %s", inputCSS, tempFile)).Run(); err != nil {
		return fmt.Errorf("failed to create temporary CSS file: %w", err)
	}
	
	// Build Tailwind command
	var cmd *exec.Cmd
	outputFile := "dist/styles.css"
	
	if strings.Contains(tp.tailwindPath, "npx") {
		args := []string{"npx", "tailwindcss", "-i", tempFile, "-o", outputFile}
		if tp.configPath != "" {
			args = append(args, "--config", tp.configPath)
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	} else {
		args := []string{"-i", tempFile, "-o", outputFile}
		if tp.configPath != "" {
			args = append(args, "--config", tp.configPath)
		}
		cmd = exec.CommandContext(ctx, tp.tailwindPath, args...)
	}
	
	// Run Tailwind CSS generation
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tailwind CSS generation failed: %w\nOutput: %s", err, string(output))
	}
	
	// Clean up temporary file
	exec.Command("rm", tempFile).Run()
	
	return nil
}

// deduplicate removes duplicate strings from a slice
func (tp *TailwindPlugin) deduplicate(slice []string) []string {
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

// Ensure TailwindPlugin implements the required interfaces
var _ plugins.ComponentPlugin = (*TailwindPlugin)(nil)
var _ plugins.BuildPlugin = (*TailwindPlugin)(nil)