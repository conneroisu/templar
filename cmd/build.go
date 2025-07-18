package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build all components without serving",
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
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	fmt.Println("🔨 Starting build process...")
	
	// Clean build artifacts if requested
	if buildClean {
		if err := cleanBuildArtifacts(cfg); err != nil {
			return fmt.Errorf("failed to clean build artifacts: %w", err)
		}
	}
	
	// Create component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)
	
	// Scan all configured paths
	fmt.Println("📁 Scanning for components...")
	totalComponents := 0
	for _, scanPath := range cfg.Components.ScanPaths {
		if err := componentScanner.ScanDirectory(scanPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", scanPath, err)
		}
	}
	
	components := componentRegistry.GetAll()
	totalComponents = len(components)
	
	if totalComponents == 0 {
		fmt.Println("No components found to build.")
		return nil
	}
	
	fmt.Printf("Found %d components\n", totalComponents)
	
	// Run templ generate
	fmt.Println("⚡ Running templ generate...")
	if err := runTemplGenerate(cfg); err != nil {
		return fmt.Errorf("failed to run templ generate: %w", err)
	}
	
	// Run Go build if production mode
	if buildProduction {
		fmt.Println("🏗️  Running production build...")
		if err := runProductionBuild(cfg); err != nil {
			return fmt.Errorf("failed to run production build: %w", err)
		}
	}
	
	// Copy static assets if output directory is specified
	if buildOutput != "" {
		fmt.Println("📦 Copying static assets...")
		if err := copyStaticAssets(cfg, buildOutput); err != nil {
			return fmt.Errorf("failed to copy static assets: %w", err)
		}
	}
	
	// Generate build analysis if requested
	if buildAnalyze {
		fmt.Println("📊 Generating build analysis...")
		// Convert map to slice for analysis
		componentSlice := make([]*registry.ComponentInfo, 0, len(components))
		for _, comp := range components {
			componentSlice = append(componentSlice, comp)
		}
		if err := generateBuildAnalysis(cfg, componentSlice); err != nil {
			return fmt.Errorf("failed to generate build analysis: %w", err)
		}
	}
	
	duration := time.Since(startTime)
	fmt.Printf("✅ Build completed successfully in %v\n", duration)
	fmt.Printf("   - %d components processed\n", totalComponents)
	
	if buildOutput != "" {
		fmt.Printf("   - Output written to: %s\n", buildOutput)
	}
	
	return nil
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
	
	// Check if templ is available
	if parts[0] == "templ" {
		if _, err := exec.LookPath("templ"); err != nil {
			return fmt.Errorf("templ command not found. Please install it with: go install github.com/a-h/templ/cmd/templ@latest")
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

func generateBuildAnalysis(cfg *config.Config, components []*registry.ComponentInfo) error {
	analysisPath := "build-analysis.json"
	if buildOutput != "" {
		analysisPath = filepath.Join(buildOutput, "build-analysis.json")
	}
	
	// Create build analysis data
	analysis := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"total_components": len(components),
		"components": make([]map[string]interface{}, len(components)),
		"build_config": map[string]interface{}{
			"command":     cfg.Build.Command,
			"watch":       cfg.Build.Watch,
			"ignore":      cfg.Build.Ignore,
			"cache_dir":   cfg.Build.CacheDir,
		},
		"scan_paths": cfg.Components.ScanPaths,
	}
	
	// Add component details
	for i, component := range components {
		analysis["components"].([]map[string]interface{})[i] = map[string]interface{}{
			"name":           component.Name,
			"package":        component.Package,
			"file_path":      component.FilePath,
			"function":       component.Name,
			"parameter_count": len(component.Parameters),
			"parameters":     component.Parameters,
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