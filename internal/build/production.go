package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/types"
)

// ProductionBuildPipeline handles production-optimized builds with asset bundling,
// minification, and deployment artifact generation.
type ProductionBuildPipeline struct {
	config    *config.Config
	outputDir string
	staticDir string
	assetsDir string
	bundler   *AssetBundler
	optimizer *AssetOptimizer
	generator *StaticSiteGenerator
	validator *BuildValidator
	// dockerBuilder   *DockerBuilder  // Temporarily disabled

	// Build metrics
	startTime    time.Time
	buildMetrics *ProductionBuildMetrics
}

// ProductionBuildOptions defines configuration for production builds
type ProductionBuildOptions struct {
	OutputDir        string `json:"output_dir"`
	StaticGeneration bool   `json:"static_generation"`
	AssetBundling    bool   `json:"asset_bundling"`
	Minification     bool   `json:"minification"`
	Compression      bool   `json:"compression"`
	Docker           bool   `json:"docker"`
	Environment      string `json:"environment"` // "production", "staging", "preview"
	CDNPath          string `json:"cdn_path,omitempty"`

	// Asset optimization
	OptimizeImages bool `json:"optimize_images"`
	OptimizeCSS    bool `json:"optimize_css"`
	OptimizeJS     bool `json:"optimize_js"`
	TreeShaking    bool `json:"tree_shaking"`
	CodeSplitting  bool `json:"code_splitting"`

	// Quality gates
	BundleSizeLimit int64 `json:"bundle_size_limit,omitempty"`
	ValidateAssets  bool  `json:"validate_assets"`
	SecurityScan    bool  `json:"security_scan"`

	// Advanced features
	ServiceWorker bool `json:"service_worker"`
	CriticalCSS   bool `json:"critical_css"`
	Prerendering  bool `json:"prerendering"`

	// Custom options
	CustomOptions map[string]interface{} `json:"custom_options,omitempty"`
}

// ProductionBuildMetrics tracks production build performance and results
type ProductionBuildMetrics struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// Component metrics
	ComponentsBuilt    int `json:"components_built"`
	TemplatesGenerated int `json:"templates_generated"`

	// Asset metrics
	AssetsProcessed  int `json:"assets_processed"`
	AssetsBundled    int `json:"assets_bundled"`
	AssetsMinified   int `json:"assets_minified"`
	AssetsCompressed int `json:"assets_compressed"`

	// Size metrics
	OriginalSize     int64   `json:"original_size_bytes"`
	OptimizedSize    int64   `json:"optimized_size_bytes"`
	CompressionRatio float64 `json:"compression_ratio"`

	// Bundle analysis
	BundleSizes map[string]int64 `json:"bundle_sizes"`
	ChunkSizes  map[string]int64 `json:"chunk_sizes,omitempty"`

	// Quality metrics
	ValidationErrors []string `json:"validation_errors,omitempty"`
	SecurityIssues   []string `json:"security_issues,omitempty"`
	PerformanceScore int      `json:"performance_score,omitempty"`

	// File counts
	StaticFiles    int `json:"static_files"`
	GeneratedFiles int `json:"generated_files"`

	// Success indicators
	Success  bool     `json:"success"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// BuildArtifacts represents the output of a production build
type BuildArtifacts struct {
	// Core artifacts
	StaticFiles    []string `json:"static_files"`
	BundledAssets  []string `json:"bundled_assets"`
	GeneratedPages []string `json:"generated_pages"`

	// Docker artifacts
	DockerImage string `json:"docker_image,omitempty"`
	Dockerfile  string `json:"dockerfile,omitempty"`

	// Deployment artifacts
	DeploymentConfig string `json:"deployment_config,omitempty"`
	ManifestFile     string `json:"manifest_file,omitempty"`

	// Asset maps
	AssetManifest string   `json:"asset_manifest"`
	SourceMaps    []string `json:"source_maps,omitempty"`

	// Quality reports
	BundleAnalysis    string `json:"bundle_analysis,omitempty"`
	SecurityReport    string `json:"security_report,omitempty"`
	PerformanceReport string `json:"performance_report,omitempty"`
}

// NewProductionBuildPipeline creates a new production build pipeline
func NewProductionBuildPipeline(cfg *config.Config, outputDir string) *ProductionBuildPipeline {
	staticDir := filepath.Join(outputDir, "static")
	assetsDir := filepath.Join(outputDir, "assets")

	return &ProductionBuildPipeline{
		config:    cfg,
		outputDir: outputDir,
		staticDir: staticDir,
		assetsDir: assetsDir,
		bundler:   NewAssetBundler(cfg, assetsDir),
		optimizer: NewAssetOptimizer(cfg),
		generator: NewStaticSiteGenerator(cfg, staticDir),
		validator: NewBuildValidator(cfg),
		// dockerBuilder: NewDockerBuilder(cfg, outputDir),  // Temporarily disabled
		buildMetrics: &ProductionBuildMetrics{},
	}
}

// Build executes the complete production build pipeline
func (p *ProductionBuildPipeline) Build(
	ctx context.Context, 
	components []*types.ComponentInfo, 
	options ProductionBuildOptions,
) (*BuildArtifacts, error) {
	p.startTime = time.Now()
	p.buildMetrics.StartTime = p.startTime

	artifacts := &BuildArtifacts{
		StaticFiles:    make([]string, 0),
		BundledAssets:  make([]string, 0),
		GeneratedPages: make([]string, 0),
		SourceMaps:     make([]string, 0),
	}

	// Ensure output directories exist
	if err := p.createOutputDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create output directories: %w", err)
	}

	// Phase 1: Template Generation and Compilation
	if err := p.generateTemplates(ctx, components); err != nil {
		return nil, fmt.Errorf("template generation failed: %w", err)
	}

	// Phase 2: Asset Discovery and Collection
	assetManifest, err := p.discoverAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("asset discovery failed: %w", err)
	}

	// Phase 3: Asset Bundling and Optimization
	if options.AssetBundling {
		bundledAssets, err := p.bundleAssets(ctx, assetManifest, options)
		if err != nil {
			return nil, fmt.Errorf("asset bundling failed: %w", err)
		}
		artifacts.BundledAssets = append(artifacts.BundledAssets, bundledAssets...)
	}

	// Phase 4: Static Site Generation
	if options.StaticGeneration {
		staticPages, err := p.generateStaticSite(ctx, components, options)
		if err != nil {
			return nil, fmt.Errorf("static site generation failed: %w", err)
		}
		artifacts.GeneratedPages = append(artifacts.GeneratedPages, staticPages...)
	}

	// Phase 5: Asset Optimization
	if options.Minification || options.Compression || options.OptimizeImages {
		if err := p.optimizeAssets(ctx, options); err != nil {
			return nil, fmt.Errorf("asset optimization failed: %w", err)
		}
	}

	// Phase 6: Generate Asset Manifest
	manifestPath, err := p.generateAssetManifest(ctx, artifacts)
	if err != nil {
		return nil, fmt.Errorf("asset manifest generation failed: %w", err)
	}
	artifacts.AssetManifest = manifestPath

	// Phase 7: Build Validation
	if options.ValidateAssets {
		if err := p.validateBuild(ctx, artifacts, options); err != nil {
			return nil, fmt.Errorf("build validation failed: %w", err)
		}
	}

	// Phase 8: Docker Image Building
	if options.Docker {
		dockerImage, dockerfile, err := p.buildDockerImage(ctx, artifacts, options)
		if err != nil {
			return nil, fmt.Errorf("docker build failed: %w", err)
		}
		artifacts.DockerImage = dockerImage
		artifacts.Dockerfile = dockerfile
	}

	// Phase 9: Generate Reports and Analysis
	if err := p.generateBuildReports(ctx, artifacts, options); err != nil {
		return nil, fmt.Errorf("report generation failed: %w", err)
	}

	// Finalize metrics
	p.buildMetrics.EndTime = time.Now()
	p.buildMetrics.Duration = p.buildMetrics.EndTime.Sub(p.buildMetrics.StartTime)
	p.buildMetrics.Success = true

	return artifacts, nil
}

// createOutputDirectories ensures all necessary output directories exist
func (p *ProductionBuildPipeline) createOutputDirectories() error {
	dirs := []string{
		p.outputDir,
		p.staticDir,
		p.assetsDir,
		filepath.Join(p.outputDir, "reports"),
		filepath.Join(p.outputDir, "docker"),
		filepath.Join(p.assetsDir, "js"),
		filepath.Join(p.assetsDir, "css"),
		filepath.Join(p.assetsDir, "images"),
		filepath.Join(p.assetsDir, "fonts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateTemplates runs templ generation for all components
func (p *ProductionBuildPipeline) generateTemplates(ctx context.Context, components []*types.ComponentInfo) error {
	// Use existing template compilation from the build pipeline
	compiler := NewTemplCompiler()

	for _, component := range components {
		if _, err := compiler.Compile(ctx, component); err != nil {
			return fmt.Errorf("failed to compile component %s: %w", component.Name, err)
		}
		p.buildMetrics.ComponentsBuilt++
	}

	return nil
}

// discoverAssets scans the project for assets that need processing
func (p *ProductionBuildPipeline) discoverAssets(ctx context.Context) (*AssetManifest, error) {
	return p.bundler.DiscoverAssets(ctx)
}

// bundleAssets performs asset bundling and processing
func (p *ProductionBuildPipeline) bundleAssets(
	ctx context.Context, 
	manifest *AssetManifest, 
	options ProductionBuildOptions,
) ([]string, error) {
	bundlerOptions := BundlerOptions{
		Minify:        options.Minification,
		TreeShaking:   options.TreeShaking,
		CodeSplitting: options.CodeSplitting,
		SourceMaps:    true, // Always generate source maps for production debugging
		Environment:   options.Environment,
	}

	return p.bundler.Bundle(ctx, manifest, bundlerOptions)
}

// generateStaticSite creates static HTML files from components
func (p *ProductionBuildPipeline) generateStaticSite(
	ctx context.Context, 
	components []*types.ComponentInfo, 
	options ProductionBuildOptions,
) ([]string, error) {
	generatorOptions := StaticGenerationOptions{
		Prerendering: options.Prerendering,
		CriticalCSS:  options.CriticalCSS,
		CDNPath:      options.CDNPath,
		Environment:  options.Environment,
	}

	return p.generator.Generate(ctx, components, generatorOptions)
}

// optimizeAssets performs post-bundle optimization
func (p *ProductionBuildPipeline) optimizeAssets(ctx context.Context, options ProductionBuildOptions) error {
	optimizerOptions := OptimizerOptions{
		Images:      options.OptimizeImages,
		CSS:         options.OptimizeCSS,
		JavaScript:  options.OptimizeJS,
		Compression: options.Compression,
	}

	return p.optimizer.Optimize(ctx, p.assetsDir, optimizerOptions)
}

// generateAssetManifest creates a manifest file for asset references
func (p *ProductionBuildPipeline) generateAssetManifest(ctx context.Context, artifacts *BuildArtifacts) (string, error) {
	manifestPath := filepath.Join(p.outputDir, "asset-manifest.json")

	manifest := AssetManifestFile{
		Version:       "1.0",
		BuildTime:     p.buildMetrics.StartTime,
		StaticFiles:   artifacts.StaticFiles,
		BundledAssets: artifacts.BundledAssets,
		SourceMaps:    artifacts.SourceMaps,
	}

	return manifestPath, writeJSONFile(manifestPath, manifest)
}

// validateBuild performs quality checks on the build output
func (p *ProductionBuildPipeline) validateBuild(ctx context.Context, artifacts *BuildArtifacts, options ProductionBuildOptions) error {
	validatorOptions := ValidationOptions{
		BundleSizeLimit:  options.BundleSizeLimit,
		SecurityScan:     options.SecurityScan,
		PerformanceCheck: true,
	}

	results, err := p.validator.Validate(ctx, artifacts, validatorOptions)
	if err != nil {
		return err
	}

	p.buildMetrics.ValidationErrors = results.Errors
	p.buildMetrics.SecurityIssues = results.SecurityIssues
	p.buildMetrics.PerformanceScore = results.PerformanceScore

	if len(results.Errors) > 0 {
		return fmt.Errorf("build validation failed with %d errors", len(results.Errors))
	}

	return nil
}

// buildDockerImage creates a production Docker image
func (p *ProductionBuildPipeline) buildDockerImage(
	ctx context.Context, 
	artifacts *BuildArtifacts, 
	options ProductionBuildOptions,
) (string, string, error) {
	// Temporarily disabled Docker functionality
	return "", "", fmt.Errorf("Docker functionality temporarily disabled")
}

// generateBuildReports creates analysis and performance reports
func (p *ProductionBuildPipeline) generateBuildReports(
	ctx context.Context, 
	artifacts *BuildArtifacts, 
	options ProductionBuildOptions,
) error {
	reportsDir := filepath.Join(p.outputDir, "reports")

	// Bundle size analysis
	bundleAnalysisPath := filepath.Join(reportsDir, "bundle-analysis.json")
	bundleAnalysis := p.bundler.AnalyzeBundles()
	if err := writeJSONFile(bundleAnalysisPath, bundleAnalysis); err != nil {
		return fmt.Errorf("failed to write bundle analysis: %w", err)
	}
	artifacts.BundleAnalysis = bundleAnalysisPath

	// Build metrics report
	metricsPath := filepath.Join(reportsDir, "build-metrics.json")
	if err := writeJSONFile(metricsPath, p.buildMetrics); err != nil {
		return fmt.Errorf("failed to write build metrics: %w", err)
	}

	return nil
}

// GetMetrics returns the current build metrics
func (p *ProductionBuildPipeline) GetMetrics() *ProductionBuildMetrics {
	return p.buildMetrics
}

// GetDefaultProductionOptions returns sensible defaults for production builds
func GetDefaultProductionOptions() ProductionBuildOptions {
	return ProductionBuildOptions{
		OutputDir:        "dist",
		StaticGeneration: true,
		AssetBundling:    true,
		Minification:     true,
		Compression:      true,
		Docker:           false,
		Environment:      "production",

		OptimizeImages: true,
		OptimizeCSS:    true,
		OptimizeJS:     true,
		TreeShaking:    true,
		CodeSplitting:  true,

		BundleSizeLimit: 5 * 1024 * 1024, // 5MB
		ValidateAssets:  true,
		SecurityScan:    true,

		ServiceWorker: false,
		CriticalCSS:   true,
		Prerendering:  false,
	}
}
