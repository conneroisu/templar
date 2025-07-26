package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/spf13/cobra"
)

// buildProductionCmd represents the production build command
var buildProductionCmd = &cobra.Command{
	Use:   "production",
	Short: "Build for production deployment",
	Long: `Build the project for production deployment with optimizations, asset bundling,
static site generation, and deployment artifacts.

This command performs:
- Template compilation and component scanning
- Asset bundling and minification
- Static site generation
- Image and asset optimization
- Docker image creation (optional)
- Build validation and quality checks
- Deployment artifact generation

Examples:
  templar build production
  templar build production --static-only
  templar build production --docker --env staging
  templar build production --output dist/prod --validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProductionBuildCommand(cmd, args)
	},
}

func runProductionBuildCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get command line flags
	outputDir, _ := cmd.Flags().GetString("output")
	staticOnly, _ := cmd.Flags().GetBool("static-only")
	dockerBuild, _ := cmd.Flags().GetBool("docker")
	environment, _ := cmd.Flags().GetString("env")
	validate, _ := cmd.Flags().GetBool("validate")
	bundleAssets, _ := cmd.Flags().GetBool("bundle")
	minify, _ := cmd.Flags().GetBool("minify")
	compress, _ := cmd.Flags().GetBool("compress")
	cdnPath, _ := cmd.Flags().GetString("cdn-path")

	// Apply environment-specific configuration overrides
	if err := cfg.ApplyEnvironmentOverrides(environment); err != nil {
		fmt.Printf("Warning: failed to apply environment overrides: %v\n", err)
	}

	// Get production configuration with environment overrides
	prodConfig, err := cfg.GetProductionConfig(environment)
	if err != nil {
		return fmt.Errorf(
			"failed to get production config for environment '%s': %w",
			environment,
			err,
		)
	}

	// Override with command line flags if provided
	if outputDir == "dist" && prodConfig.OutputDir != "" {
		outputDir = prodConfig.OutputDir
	}
	if !cmd.Flags().Changed("bundle") {
		bundleAssets = prodConfig.Bundling.Enabled
	}
	if !cmd.Flags().Changed("minify") {
		minify = prodConfig.Minification.CSS && prodConfig.Minification.JavaScript
	}
	if !cmd.Flags().Changed("compress") {
		compress = prodConfig.Compression.Enabled
	}
	if cdnPath == "" && prodConfig.CDN.BasePath != "" {
		cdnPath = prodConfig.CDN.BasePath
	}

	// Create build options
	options := build.ProductionBuildOptions{
		OutputDir:        outputDir,
		StaticGeneration: true,
		AssetBundling:    bundleAssets,
		Minification:     minify,
		Compression:      compress,
		Docker:           dockerBuild,
		Environment:      environment,
		CDNPath:          cdnPath,

		OptimizeImages: !staticOnly,
		OptimizeCSS:    true,
		OptimizeJS:     true,
		TreeShaking:    true,
		CodeSplitting:  true,

		ValidateAssets: validate,
		SecurityScan:   validate,

		CriticalCSS:  true,
		Prerendering: false,

		CustomOptions: make(map[string]interface{}),
	}

	// Override defaults based on flags
	if staticOnly {
		options.AssetBundling = false
		options.OptimizeImages = false
		options.Docker = false
	}

	fmt.Printf("Starting production build for %s environment...\n", environment)

	// Initialize component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	ctx := context.Background()

	// Scan for components
	fmt.Println("Scanning components...")
	for _, scanPath := range cfg.Components.ScanPaths {
		if err := componentScanner.ScanDirectory(scanPath); err != nil {
			fmt.Printf("Warning: failed to scan directory %s: %v\n", scanPath, err)
		}
	}

	components := componentRegistry.GetAll()
	if len(components) == 0 {
		return fmt.Errorf("no components found to build")
	}
	fmt.Printf("Found %d components\n", len(components))

	// Initialize production build pipeline
	pipeline := build.NewProductionBuildPipeline(cfg, outputDir)

	// Execute build
	fmt.Println("Building for production...")
	startTime := time.Now()

	artifacts, err := pipeline.Build(ctx, components, options)
	if err != nil {
		return fmt.Errorf("production build failed: %w", err)
	}

	duration := time.Since(startTime)

	// Get build metrics
	metrics := pipeline.GetMetrics()

	// Display build results
	fmt.Printf("\n✅ Production build completed in %v\n", duration)
	fmt.Printf("📊 Build Summary:\n")
	fmt.Printf("   Components built: %d\n", metrics.ComponentsBuilt)
	fmt.Printf("   Static files: %d\n", len(artifacts.StaticFiles))
	fmt.Printf("   Bundled assets: %d\n", len(artifacts.BundledAssets))
	fmt.Printf("   Generated pages: %d\n", len(artifacts.GeneratedPages))

	if metrics.OriginalSize > 0 && metrics.OptimizedSize > 0 {
		compressionRatio := float64(metrics.OptimizedSize) / float64(metrics.OriginalSize) * 100
		fmt.Printf("   Size reduction: %.1f%% (%.2f MB → %.2f MB)\n",
			100-compressionRatio,
			float64(metrics.OriginalSize)/1024/1024,
			float64(metrics.OptimizedSize)/1024/1024)
	}

	if len(metrics.ValidationErrors) > 0 {
		fmt.Printf("⚠️  Validation warnings: %d\n", len(metrics.ValidationErrors))
	}

	if artifacts.DockerImage != "" {
		fmt.Printf("🐳 Docker image: %s\n", artifacts.DockerImage)
	}

	fmt.Printf("\n📁 Output directory: %s\n", outputDir)

	// Display asset manifest location
	if artifacts.AssetManifest != "" {
		fmt.Printf("📋 Asset manifest: %s\n", artifacts.AssetManifest)
	}

	// Display reports
	if artifacts.BundleAnalysis != "" {
		fmt.Printf("📈 Bundle analysis: %s\n", artifacts.BundleAnalysis)
	}

	// Success message
	fmt.Printf("\n🚀 Production build ready for deployment!\n")

	return nil
}

func init() {
	buildCmd.AddCommand(buildProductionCmd)

	// Output options
	buildProductionCmd.Flags().
		StringP("output", "o", "dist", "Output directory for production build")
	buildProductionCmd.Flags().
		StringP("env", "e", "production", "Environment (production, staging, preview)")

	// Build options
	buildProductionCmd.Flags().
		Bool("static-only", false, "Generate static files only (no asset processing)")
	buildProductionCmd.Flags().Bool("bundle", true, "Bundle and optimize assets")
	buildProductionCmd.Flags().Bool("minify", true, "Minify CSS and JavaScript")
	buildProductionCmd.Flags().Bool("compress", true, "Compress assets with gzip/brotli")

	// Asset optimization
	buildProductionCmd.Flags().Bool("optimize-images", true, "Optimize images")
	buildProductionCmd.Flags().Bool("critical-css", true, "Extract critical CSS")
	buildProductionCmd.Flags().Bool("tree-shake", true, "Remove unused code")

	// Docker options
	buildProductionCmd.Flags().Bool("docker", false, "Build Docker image")
	buildProductionCmd.Flags().String("docker-tag", "", "Docker image tag")

	// CDN and deployment
	buildProductionCmd.Flags().String("cdn-path", "", "CDN base path for assets")
	buildProductionCmd.Flags().String("base-url", "", "Base URL for sitemap generation")

	// Quality assurance
	buildProductionCmd.Flags().Bool("validate", true, "Validate build output")
	buildProductionCmd.Flags().Bool("security-scan", false, "Run security scan on assets")
	buildProductionCmd.Flags().Int64("bundle-size-limit", 5*1024*1024, "Bundle size limit in bytes")

	// Advanced options
	buildProductionCmd.Flags().Bool("source-maps", true, "Generate source maps")
	buildProductionCmd.Flags().Bool("sitemap", true, "Generate sitemap.xml")
	buildProductionCmd.Flags().Bool("robots", true, "Generate robots.txt")
	buildProductionCmd.Flags().
		StringSlice("error-pages", []string{"404", "500"}, "Error pages to generate")
}
