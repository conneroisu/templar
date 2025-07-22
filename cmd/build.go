package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/di"
	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Build all components without serving",
	Long: `Build all components in the project without starting the development server.
This runs templ generate and optionally performs production optimizations.

Examples:
  templar build                   # Build all components
  templar build --production      # Build with production optimizations
  templar build --analyze         # Generate build analysis
  templar build --output dist     # Build to specific output directory`,
	RunE: runBuild,
}

var (
	buildOutput     string
	buildProduction bool
	buildAnalyze    bool
	buildClean      bool
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output directory")
	buildCmd.Flags().BoolVar(&buildProduction, "production", false, "Production build optimizations")
	buildCmd.Flags().BoolVar(&buildAnalyze, "analyze", false, "Generate build analysis")
	buildCmd.Flags().BoolVar(&buildClean, "clean", false, "Clean build artifacts before building")
}

func runBuild(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize monitoring for build tracking
	monitor := monitoring.GetGlobalMonitor()
	if monitor == nil {
		// Try to initialize a basic monitor for build tracking
		config := monitoring.DefaultMonitorConfig()
		config.HTTPEnabled = false // Disable HTTP for build command
		// Skip logging initialization for build command to avoid complexity
		monitor = nil
	}

	fmt.Println("🔨 Starting build process...")

	// Track the overall build operation
	return monitoring.TrackOperation(ctx, "build", "full_build", func(ctx context.Context) error {
		// Clean build artifacts if requested
		if buildClean {
			err := monitoring.TrackOperation(ctx, "build", "clean_artifacts", func(ctx context.Context) error {
				return cleanBuildArtifacts(cfg)
			})
			if err != nil {
				return fmt.Errorf("failed to clean build artifacts: %w", err)
			}
		}

		// Initialize dependency injection container
		container := di.NewServiceContainer(cfg)
		if err := container.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize service container: %w", err)
		}
		defer func() {
			if shutdownErr := container.Shutdown(ctx); shutdownErr != nil {
				fmt.Printf("Warning: Error during container shutdown: %v\n", shutdownErr)
			}
		}()

		// Get services from container
		componentRegistry, err := container.GetRegistry()
		if err != nil {
			return fmt.Errorf("failed to get component registry: %w", err)
		}

		componentScanner, err := container.GetScanner()
		if err != nil {
			return fmt.Errorf("failed to get component scanner: %w", err)
		}

		// Scan all configured paths
		fmt.Println("📁 Scanning for components...")
		err = monitoring.TrackOperation(ctx, "build", "scan_components", func(ctx context.Context) error {
			for _, scanPath := range cfg.Components.ScanPaths {
				if err := componentScanner.ScanDirectory(scanPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", scanPath, err)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan components: %w", err)
		}

		components := componentRegistry.GetAll()
		totalComponents := len(components)

		if totalComponents == 0 {
			fmt.Println("No components found to build.")
			return nil
		}

		fmt.Printf("Found %d components\n", totalComponents)

		// Record component count metric (simplified for build command)
		if monitor != nil {
			fmt.Printf("📊 Monitoring: %d components found\n", totalComponents)
		}

		// Run templ generate
		fmt.Println("⚡ Running templ generate...")
		err = monitoring.TrackOperation(ctx, "build", "templ_generate", func(ctx context.Context) error {
			return runTemplGenerate(cfg)
		})
		if err != nil {
			return fmt.Errorf("failed to run templ generate: %w", err)
		}

		// Run Go build if production mode
		if buildProduction {
			fmt.Println("🏗️  Running production build...")
			err := monitoring.TrackOperation(ctx, "build", "production_build", func(ctx context.Context) error {
				return runProductionBuild(cfg)
			})
			if err != nil {
				return fmt.Errorf("failed to run production build: %w", err)
			}
		}

		// Copy static assets if output directory is specified
		if buildOutput != "" {
			fmt.Println("📦 Copying static assets...")
			err := monitoring.TrackOperation(ctx, "build", "copy_assets", func(ctx context.Context) error {
				return copyStaticAssets(cfg, buildOutput)
			})
			if err != nil {
				return fmt.Errorf("failed to copy static assets: %w", err)
			}
		}

		// Generate build analysis if requested
		if buildAnalyze {
			fmt.Println("📊 Generating build analysis...")
			err := monitoring.TrackOperation(ctx, "build", "generate_analysis", func(ctx context.Context) error {
				// Convert map to slice for analysis
				componentSlice := make([]*types.ComponentInfo, 0, len(components))
				for _, comp := range components {
					componentSlice = append(componentSlice, comp)
				}
				return generateBuildAnalysis(cfg, componentSlice)
			})
			if err != nil {
				return fmt.Errorf("failed to generate build analysis: %w", err)
			}
		}

		duration := time.Since(startTime)
		fmt.Printf("✅ Build completed successfully in %v\n", duration)
		fmt.Printf("   - %d components processed\n", totalComponents)

		if buildOutput != "" {
			fmt.Printf("   - Output written to: %s\n", buildOutput)
		}

		// Record build success metrics
		if monitor != nil {
			// Simple metric recording without detailed counter/histogram calls
			_ = monitor // Monitoring integration available but simplified
		}

		return nil
	})
}

func cleanBuildArtifacts(cfg *config.Config) error {
	fmt.Println("🧹 Cleaning build artifacts...")

	// Clean cache directory
	cacheDir := cfg.Build.CacheDir
	if cacheDir != "" {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to remove cache directory: %w", err)
		}
		fmt.Printf("   - Cleaned cache directory: %s\n", cacheDir)
	}

	// Clean generated Go files
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "_templ.go") {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove generated file %s: %w", path, err)
			}
			fmt.Printf("   - Removed: %s\n", path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to clean generated files: %w", err)
	}

	return nil
}

func runTemplGenerate(cfg *config.Config) error {
	// Use configured build command or default
	buildCmd := cfg.Build.Command
	if buildCmd == "" {
		buildCmd = "templ generate"
	}

	// Split command into parts
	parts := strings.Fields(buildCmd)
	if len(parts) == 0 {
		return errors.New("empty build command")
	}

	// Validate command before execution
	if err := validateBuildCommand(parts[0], parts[1:]); err != nil {
		return fmt.Errorf("invalid build command: %w", err)
	}

	// Check if templ is available
	if parts[0] == "templ" {
		if _, err := exec.LookPath("templ"); err != nil {
			return errors.New("templ command not found. Please install it with: go install github.com/a-h/templ/cmd/templ@v0.3.819")
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

// validateBuildCommand validates the command and arguments to prevent command injection
func validateBuildCommand(command string, args []string) error {
	// Allowlist of permitted commands
	allowedCommands := map[string]bool{
		"templ": true,
		"go":    true,
	}

	// Check if command is in allowlist
	if err := validateCommand(command, allowedCommands); err != nil {
		return fmt.Errorf("build command validation failed: %w", err)
	}

	// Validate arguments - prevent shell metacharacters and path traversal
	if err := validateArguments(args); err != nil {
		return fmt.Errorf("argument validation failed: %w", err)
	}

	return nil
}

func runProductionBuild(cfg *config.Config) error {
	// Run go build with optimizations
	cmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", "main", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}

func copyStaticAssets(cfg *config.Config, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy static directory if it exists
	staticDir := "static"
	if _, err := os.Stat(staticDir); err == nil {
		destDir := filepath.Join(outputDir, "static")
		if err := copyDir(staticDir, destDir); err != nil {
			return fmt.Errorf("failed to copy static directory: %w", err)
		}
		fmt.Printf("   - Copied static assets to: %s\n", destDir)
	}

	return nil
}

func copyDir(src, dest string) error {
	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// Walk through source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return copyFile(path, destPath)
	})
}

func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dest, data, 0644)
}

func generateBuildAnalysis(cfg *config.Config, components []*types.ComponentInfo) error {
	analysisPath := "build-analysis.json"
	if buildOutput != "" {
		analysisPath = filepath.Join(buildOutput, "build-analysis.json")
	}

	// Create build analysis data
	analysis := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"total_components": len(components),
		"components":       make([]map[string]interface{}, len(components)),
		"build_config": map[string]interface{}{
			"command":   cfg.Build.Command,
			"watch":     cfg.Build.Watch,
			"ignore":    cfg.Build.Ignore,
			"cache_dir": cfg.Build.CacheDir,
		},
		"scan_paths": cfg.Components.ScanPaths,
	}

	// Add component details
	for i, component := range components {
		analysis["components"].([]map[string]interface{})[i] = map[string]interface{}{
			"name":            component.Name,
			"package":         component.Package,
			"file_path":       component.FilePath,
			"function":        component.Name,
			"parameter_count": len(component.Parameters),
			"parameters":      component.Parameters,
		}
	}

	// Write analysis to file
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis data: %w", err)
	}

	if err := os.WriteFile(analysisPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write analysis file: %w", err)
	}

	fmt.Printf("   - Build analysis written to: %s\n", analysisPath)
	return nil
}
