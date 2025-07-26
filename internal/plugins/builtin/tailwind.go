package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/plugins"
	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/validation"
)

// TailwindPlugin provides Tailwind CSS integration for Templar components.
type TailwindPlugin struct {
	config       plugins.PluginConfig
	tailwindPath string
	configPath   string
	enabled      bool
}

// NewTailwindPlugin creates a new Tailwind CSS plugin.
func NewTailwindPlugin() *TailwindPlugin {
	return &TailwindPlugin{
		enabled: true,
	}
}

// Name returns the plugin name.
func (tp *TailwindPlugin) Name() string {
	return "tailwind"
}

// Version returns the plugin version.
func (tp *TailwindPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description.
func (tp *TailwindPlugin) Description() string {
	return "Tailwind CSS integration for automatic class detection and CSS generation"
}

// Initialize initializes the Tailwind plugin.
func (tp *TailwindPlugin) Initialize(ctx context.Context, config plugins.PluginConfig) error {
	tp.config = config

	// Check if Tailwind CLI is available
	tailwindPath, err := exec.LookPath("tailwindcss")
	if err != nil {
		// Try npx
		if _, err := exec.LookPath("npx"); err == nil {
			tp.tailwindPath = "npx tailwindcss"
		} else {
			return errors.New("tailwindcss not found in PATH and npx not available")
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
		// Validate path before accessing
		if err := validation.ValidatePath(path); err != nil {
			continue // Skip invalid paths
		}
		if _, err := os.Stat(path); err == nil {
			tp.configPath = path

			break
		}
	}

	return nil
}

// Shutdown shuts down the plugin.
func (tp *TailwindPlugin) Shutdown(ctx context.Context) error {
	tp.enabled = false

	return nil
}

// Health returns the plugin health status.
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

// HandleComponent processes components to extract Tailwind classes.
func (tp *TailwindPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
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

// SupportedExtensions returns the file extensions this plugin handles.
func (tp *TailwindPlugin) SupportedExtensions() []string {
	return []string{".templ", ".html", ".tsx", ".jsx"}
}

// Priority returns the execution priority.
func (tp *TailwindPlugin) Priority() int {
	return 10 // Lower priority to run after basic processing
}

// PreBuild is called before the build process starts.
func (tp *TailwindPlugin) PreBuild(ctx context.Context, components []*types.ComponentInfo) error {
	// Nothing to do before build for Tailwind
	return nil
}

// PostBuild generates CSS after components are built.
func (tp *TailwindPlugin) PostBuild(
	ctx context.Context,
	components []*types.ComponentInfo,
	buildResult plugins.BuildResult,
) error {
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

// TransformBuildCommand allows modifying the build command.
func (tp *TailwindPlugin) TransformBuildCommand(
	ctx context.Context,
	command []string,
) ([]string, error) {
	// For Tailwind, we don't need to modify the build command
	return command, nil
}

// extractTailwindClasses extracts Tailwind CSS classes from a component file.
func (tp *TailwindPlugin) extractTailwindClasses(filePath string) ([]string, error) {
	// This is a simplified implementation
	// In practice, you'd want a more sophisticated parser

	// Validate file path for security
	if err := validation.ValidatePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	content, err := os.ReadFile(filePath)
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

// isTailwindClass checks if a class name looks like a Tailwind class.
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

// Note: validatePath function has been replaced with centralized validation
// in the validation package.

// sanitizeInput ensures input content is safe for file operations.
func sanitizeInput(input string) string {
	// Remove any shell metacharacters and dangerous sequences
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\\", "'", "\""}
	sanitized := input
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "")
	}

	// Remove dangerous command patterns
	dangerousPatterns := []string{
		"rm -rf", "rm ", "cat ", "echo ", "ls ", "mkdir ", "rmdir ",
		"/bin/", "/usr/bin/", "/sbin/", "/usr/sbin/", "/etc/", "/dev/",
		"sudo ", "su ", "chmod ", "chown ", "wget ", "curl ", "nc ",
		"netcat ", "bash ", "sh ", "/tmp/", "/var/", "passwd", "shadow",
	}

	for _, pattern := range dangerousPatterns {
		sanitized = strings.ReplaceAll(sanitized, pattern, "")
	}

	return sanitized
}

// generateCSS generates CSS using Tailwind CLI.
func (tp *TailwindPlugin) generateCSS(ctx context.Context, classes map[string]bool) error {
	// Create temporary input CSS file with sanitized content
	baseCSS := "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"
	inputCSS := sanitizeInput(baseCSS)

	// Create secure temporary file using os.CreateTemp
	tempFile, err := os.CreateTemp("", "templar-tailwind-input-*.css")
	if err != nil {
		return fmt.Errorf("failed to create temporary CSS file: %w", err)
	}
	defer tempFile.Close()

	tempFilePath := tempFile.Name()

	// Write content to temporary file
	if _, err := tempFile.Write([]byte(inputCSS)); err != nil {
		os.Remove(tempFilePath) // Cleanup on error

		return fmt.Errorf("failed to write to temporary CSS file: %w", err)
	}

	// Build Tailwind command
	var cmd *exec.Cmd
	outputFile := "dist/styles.css"

	// Validate output file path
	if err := validation.ValidatePath(outputFile); err != nil {
		return fmt.Errorf("invalid output file path: %w", err)
	}

	// Define allowed commands for security
	allowedCommands := map[string]bool{
		"npx":         true,
		"tailwindcss": true,
	}

	// Validate temp file path
	if err := validation.ValidatePath(tempFilePath); err != nil {
		os.Remove(tempFilePath) // Cleanup temp file

		return fmt.Errorf("invalid temp file path: %w", err)
	}

	if strings.Contains(tp.tailwindPath, "npx") {
		// Validate commands
		if err := validation.ValidateCommand("npx", allowedCommands); err != nil {
			os.Remove(tempFilePath) // Cleanup temp file

			return fmt.Errorf("command validation failed: %w", err)
		}
		if err := validation.ValidateCommand("tailwindcss", allowedCommands); err != nil {
			os.Remove(tempFilePath) // Cleanup temp file

			return fmt.Errorf("command validation failed: %w", err)
		}

		args := []string{"npx", "tailwindcss", "-i", tempFilePath, "-o", outputFile}
		if tp.configPath != "" {
			// Validate config path for security before use
			if err := validation.ValidatePath(tp.configPath); err != nil {
				os.Remove(tempFilePath) // Cleanup temp file

				return fmt.Errorf("invalid config path: %w", err)
			}
			if err := validation.ValidateArgument(tp.configPath); err != nil {
				os.Remove(tempFilePath) // Cleanup temp file

				return fmt.Errorf("invalid config path argument: %w", err)
			}
			args = append(args, "--config", tp.configPath)
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	} else {
		// Validate tailwind command path
		tailwindCmd := filepath.Base(tp.tailwindPath)
		if err := validation.ValidateCommand(tailwindCmd, allowedCommands); err != nil {
			os.Remove(tempFilePath) // Cleanup temp file

			return fmt.Errorf("command validation failed: %w", err)
		}

		args := []string{"-i", tempFilePath, "-o", outputFile}
		if tp.configPath != "" {
			// Validate config path for security before use
			if err := validation.ValidatePath(tp.configPath); err != nil {
				os.Remove(tempFilePath) // Cleanup temp file

				return fmt.Errorf("invalid config path: %w", err)
			}
			if err := validation.ValidateArgument(tp.configPath); err != nil {
				os.Remove(tempFilePath) // Cleanup temp file

				return fmt.Errorf("invalid config path argument: %w", err)
			}
			args = append(args, "--config", tp.configPath)
		}
		cmd = exec.CommandContext(ctx, tp.tailwindPath, args...)
	}

	// Run Tailwind CSS generation
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temporary file on error
		os.Remove(tempFilePath)

		return fmt.Errorf("tailwind CSS generation failed: %w\nOutput: %s", err, string(output))
	}

	// Clean up temporary file securely using os.Remove
	if err := os.Remove(tempFilePath); err != nil {
		// Log but don't fail the operation for cleanup errors
		fmt.Printf("Warning: failed to remove temporary file %s: %v\n", tempFilePath, err)
	}

	return nil
}

// deduplicate removes duplicate strings from a slice.
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

// Ensure TailwindPlugin implements the required interfaces.
var _ plugins.ComponentPlugin = (*TailwindPlugin)(nil)
var _ plugins.BuildPlugin = (*TailwindPlugin)(nil)
