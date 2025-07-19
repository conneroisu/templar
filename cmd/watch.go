package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for file changes and rebuild components",
	Long: `Watch for file changes and automatically rebuild components without serving.
This is useful for development workflows where you want automatic rebuilds
but don't need the preview server.

Examples:
  templar watch                   # Watch all configured paths
  templar watch --verbose         # Watch with verbose output
  templar watch --command "make"  # Run custom command on changes`,
	RunE: runWatch,
}

var (
	watchVerbose bool
	watchCommand string
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().BoolVarP(&watchVerbose, "verbose", "v", false, "Verbose output")
	watchCmd.Flags().StringVarP(&watchCommand, "command", "c", "", "Custom command to run on changes")
}

func runWatch(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Create file watcher
	fileWatcher, err := watcher.NewFileWatcher(300 * time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer fileWatcher.Stop()

	// Add filters
	fileWatcher.AddFilter(watcher.TemplFilter)
	fileWatcher.AddFilter(watcher.GoFilter)
	fileWatcher.AddFilter(watcher.NoTestFilter)
	fileWatcher.AddFilter(watcher.NoVendorFilter)
	fileWatcher.AddFilter(watcher.NoGitFilter)

	// Add change handler
	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		if watchVerbose {
			fmt.Printf("📁 File changes detected:\n")
			for _, event := range events {
				fmt.Printf("   %s: %s\n", event.Type, event.Path)
			}
		} else {
			fmt.Printf("📁 %d file(s) changed\n", len(events))
		}

		// Rescan components
		if err := rescanComponents(componentScanner, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rescan components: %v\n", err)
		}

		// Run custom command if specified
		if watchCommand != "" {
			if err := runCustomCommand(watchCommand); err != nil {
				fmt.Fprintf(os.Stderr, "Custom command failed: %v\n", err)
			}
		} else {
			// Run default build command
			if err := runBuildCommand(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Build command failed: %v\n", err)
			}
		}

		return nil
	})

	// Add watch paths
	fmt.Println("🔍 Setting up file watching...")
	for _, path := range cfg.Components.ScanPaths {
		if err := fileWatcher.AddRecursive(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to watch path %s: %v\n", path, err)
		} else {
			fmt.Printf("   - Watching: %s\n", path)
		}
	}

	// Initial scan
	fmt.Println("📁 Performing initial scan...")
	if err := initialScan(componentScanner, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Initial scan failed: %v\n", err)
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fileWatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	fmt.Println("👀 Watching for changes... (Press Ctrl+C to stop)")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n🛑 Stopping file watcher...")
	cancel()

	return nil
}

func initialScan(scanner *scanner.ComponentScanner, cfg *config.Config) error {
	for _, path := range cfg.Components.ScanPaths {
		if err := scanner.ScanDirectory(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", path, err)
		}
	}

	// Count components
	registry := scanner.GetRegistry()
	components := registry.GetAll()
	totalComponents := len(components)

	if watchVerbose {
		fmt.Printf("Found %d components:\n", totalComponents)
		for _, component := range components {
			fmt.Printf("   - %s (%s)\n", component.Name, component.FilePath)
		}
	} else {
		fmt.Printf("Found %d components\n", totalComponents)
	}

	return nil
}

func rescanComponents(scanner *scanner.ComponentScanner, cfg *config.Config) error {
	for _, path := range cfg.Components.ScanPaths {
		if err := scanner.ScanDirectory(path); err != nil {
			return fmt.Errorf("failed to rescan directory %s: %w", path, err)
		}
	}

	return nil
}

func runCustomCommand(command string) error {
	fmt.Printf("🔨 Running custom command: %s\n", command)

	// Parse the command into parts
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New("empty custom command")
	}

	// For security, validate the command
	if err := validateCustomCommand(parts[0], parts[1:]); err != nil {
		return fmt.Errorf("invalid custom command: %w", err)
	}

	// Execute the command
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("custom command failed: %w", err)
	}

	return nil
}

// validateCustomCommand validates custom commands with a security-focused allowlist
func validateCustomCommand(command string, args []string) error {
	// Allowlist of essential development commands only (security-hardened)
	allowedCommands := map[string]bool{
		"templ": true, // Template generation
		"go":    true, // Go build/test/run commands
		"npm":   true, // Node package manager
		"yarn":  true, // Alternative Node package manager
		"pnpm":  true, // Alternative Node package manager
		"make":  true, // Build automation
		"git":   true, // Version control (read-only operations recommended)
		"echo":  true, // Safe output command
	}

	// Check if command is in allowlist
	if err := validateCommand(command, allowedCommands); err != nil {
		return fmt.Errorf("custom command validation failed: %w", err)
	}

	// Enhanced validation for potentially dangerous commands
	if err := validateCommandSpecific(command, args); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	// Validate arguments - prevent shell metacharacters and path traversal
	if err := validateArguments(args); err != nil {
		return fmt.Errorf("argument validation failed: %w", err)
	}

	return nil
}

// validateCommandSpecific provides enhanced validation for specific commands
func validateCommandSpecific(command string, args []string) error {
	switch command {
	case "git":
		return validateGitCommand(args)
	case "npm", "yarn", "pnpm":
		return validatePackageManagerCommand(args)
	case "go":
		return validateGoCommand(args)
	}
	return nil
}

// validateGitCommand ensures git commands are safe (read-only operations)
func validateGitCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("git command requires arguments")
	}

	// Allow only safe, read-only git operations
	safeGitCommands := map[string]bool{
		"status":    true,
		"log":       true,
		"show":      true,
		"diff":      true,
		"branch":    true,
		"tag":       true,
		"remote":    true,
		"ls-files":  true,
		"ls-tree":   true,
		"rev-parse": true,
	}

	subcommand := args[0]
	if !safeGitCommands[subcommand] {
		return fmt.Errorf("git subcommand '%s' is not allowed (only read-only operations permitted)", subcommand)
	}

	return nil
}

// validatePackageManagerCommand ensures package manager commands are safe
func validatePackageManagerCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package manager command requires arguments")
	}

	// Allow common build/development operations
	safeCommands := map[string]bool{
		"install":  true,
		"ci":       true,
		"run":      true,
		"build":    true,
		"test":     true,
		"start":    true,
		"dev":      true,
		"lint":     true,
		"format":   true,
		"check":    true,
		"audit":    true,
		"outdated": true,
	}

	subcommand := args[0]
	if !safeCommands[subcommand] {
		return fmt.Errorf("package manager subcommand '%s' is not allowed", subcommand)
	}

	return nil
}

// validateGoCommand ensures go commands are safe
func validateGoCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("go command requires arguments")
	}

	// Allow common development operations
	safeCommands := map[string]bool{
		"build":    true,
		"run":      true,
		"test":     true,
		"generate": true,
		"fmt":      true,
		"vet":      true,
		"mod":      true,
		"version":  true,
		"env":      true,
		"list":     true,
	}

	subcommand := args[0]
	if !safeCommands[subcommand] {
		return fmt.Errorf("go subcommand '%s' is not allowed", subcommand)
	}

	return nil
}

func runBuildCommand(cfg *config.Config) error {
	buildCmd := cfg.Build.Command
	if buildCmd == "" {
		buildCmd = "templ generate"
	}

	fmt.Printf("🔨 Running build command: %s\n", buildCmd)

	// Split command into parts
	parts := strings.Fields(buildCmd)
	if len(parts) == 0 {
		return errors.New("empty build command")
	}

	// Validate command before execution (reuse validation from build.go)
	if err := validateBuildCommand(parts[0], parts[1:]); err != nil {
		return fmt.Errorf("invalid build command: %w", err)
	}

	// Check if templ is available
	if parts[0] == "templ" {
		if _, err := exec.LookPath("templ"); err != nil {
			return errors.New("templ command not found. Please install it with: go install github.com/a-h/templ/cmd/templ@latest")
		}
	}

	// Execute the command
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}

	return nil
}
