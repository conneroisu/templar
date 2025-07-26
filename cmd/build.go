package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/services"
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
	buildCmd.Flags().
		BoolVar(&buildProduction, "production", false, "Production build optimizations")
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

	fmt.Println("🔨 Starting build process...")

	// Create build service
	buildService := services.NewBuildService(cfg)

	// Configure build options
	opts := services.BuildOptions{
		Output:     buildOutput,
		Production: buildProduction,
		Analyze:    buildAnalyze,
		Clean:      buildClean,
	}

	// Perform the build
	result, err := buildService.Build(ctx, opts)
	if err != nil {
		return err
	}

	// Display results
	if result.Success {
		fmt.Printf("✅ Build completed successfully in %v\n", result.Duration)
		if result.ComponentCount > 0 {
			fmt.Printf("📦 Built %d components\n", result.ComponentCount)
		}
	} else {
		fmt.Printf("❌ Build failed after %v\n", result.Duration)
		for _, buildErr := range result.Errors {
			fmt.Printf("   Error: %v\n", buildErr)
		}
		return fmt.Errorf("build process failed")
	}

	fmt.Printf("⏱️  Total build time: %v\n", time.Since(startTime))
	return nil
}
