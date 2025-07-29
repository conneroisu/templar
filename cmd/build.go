package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/services"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Build all components without starting development server",
	Long: `Build all components in the project without starting the development server.

Runs templ generate to compile template files and optionally performs production
optimizations including minification, compression, and asset optimization.
Ideal for CI/CD pipelines and production deployments.

Examples:
  templar build                   # Build all components with defaults
  templar build --production      # Enable production optimizations
  templar build --output dist     # Build to custom output directory
  templar build --analyze         # Generate detailed build analysis
  templar build --workers 8       # Use 8 parallel build workers
  templar build --clean           # Clean output directory before build
  templar build --verbose         # Enable detailed build logging
  templar build --config prod.yml # Use production configuration

Build Features:
  • Parallel component compilation
  • Production optimization pipeline
  • Asset minification and compression
  • Build caching for faster rebuilds
  • Comprehensive error reporting
  • Build performance analysis

Production Optimizations:
  • CSS and JavaScript minification
  • Image optimization and compression
  • Dead code elimination
  • Bundle size analysis
  • Cache-friendly file naming

Pro Tips:
  • Use --production for deployment builds
  • Check build output with --analyze for optimization opportunities
  • Build cache improves rebuild performance significantly
  • Use --workers to optimize build speed for large projects

See also: templar serve, templar watch, templar list`,
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
	buildCmd.Flags().BoolVar(&buildClean, FlagClean, false, FlagDescClean)
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

		return errors.New("build process failed")
	}

	fmt.Printf("⏱️  Total build time: %v\n", time.Since(startTime))

	return nil
}
