package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/di"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/monitoring"
)

// BuildService handles component building business logic
type BuildService struct {
	config    *config.Config
	container *di.ServiceContainer
}

// NewBuildService creates a new build service
func NewBuildService(cfg *config.Config) *BuildService {
	return &BuildService{
		config: cfg,
	}
}

// BuildOptions contains options for the build process
type BuildOptions struct {
	Output     string
	Production bool
	Analyze    bool
	Clean      bool
}

// BuildResult contains the result of a build operation
type BuildResult struct {
	Duration       time.Duration
	ComponentCount int
	Success        bool
	Errors         []error
}

// Build performs the complete build process
func (s *BuildService) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	startTime := time.Now()
	result := &BuildResult{
		Success: true,
	}

	// Initialize monitoring for build tracking
	monitor := monitoring.GetGlobalMonitor()
	if monitor == nil {
		// Try to initialize a basic monitor for build tracking
		config := monitoring.DefaultMonitorConfig()
		config.HTTPEnabled = false // Disable HTTP for build command
		monitor = nil
	}

	// Track the overall build operation
	err := monitoring.TrackOperation(ctx, "build", "full_build", func(ctx context.Context) error {
		// Clean build artifacts if requested
		if opts.Clean {
			err := monitoring.TrackOperation(ctx, "build", "clean_artifacts", func(ctx context.Context) error {
				return s.cleanBuildArtifacts()
			})
			if err != nil {
				return fmt.Errorf("failed to clean build artifacts: %w", err)
			}
		}

		// Initialize dependency injection container
		container := di.NewServiceContainer(s.config)
		if err := container.Initialize(); err != nil {
			return errors.BuildServiceError("INIT_CONTAINER", "service container initialization failed", err)
		}
		s.container = container

		defer func() {
			if shutdownErr := container.Shutdown(ctx); shutdownErr != nil {
				// Log shutdown error but don't fail the build
			}
		}()

		// Get services from container
		componentRegistry, err := container.GetRegistry()
		if err != nil {
			return errors.BuildServiceError("GET_REGISTRY", "failed to get component registry", err)
		}

		scanner, err := container.GetScanner()
		if err != nil {
			return errors.BuildServiceError("GET_SCANNER", "failed to get component scanner", err)
		}

		buildPipeline, err := container.GetBuildPipeline()
		if err != nil {
			return errors.BuildServiceError("GET_PIPELINE", "failed to get build pipeline", err)
		}

		// Perform component scanning
		if err := s.scanComponents(ctx, scanner); err != nil {
			return errors.BuildServiceError("SCAN_COMPONENTS", "component scanning failed", err)
		}

		// Get component count
		result.ComponentCount = componentRegistry.Count()

		// Start build pipeline
		if err := buildPipeline.Start(ctx); err != nil {
			return errors.BuildServiceError("START_PIPELINE", "failed to start build pipeline", err)
		}
		defer buildPipeline.Stop()

		// Process all components
		components := componentRegistry.GetAll()
		if err := s.buildComponents(ctx, buildPipeline, components); err != nil {
			return errors.BuildServiceError("BUILD_COMPONENTS", "component building failed", err)
		}

		// Generate build analysis if requested
		if opts.Analyze {
			if err := s.generateBuildAnalysis(opts.Output); err != nil {
				return errors.BuildServiceError("ANALYZE", "failed to generate build analysis", err)
			}
		}

		// Production optimizations if requested
		if opts.Production {
			if err := s.applyProductionOptimizations(ctx, opts.Output); err != nil {
				return errors.BuildServiceError("OPTIMIZE", "production optimization failed", err)
			}
		}

		return nil
	})

	result.Duration = time.Since(startTime)
	if err != nil {
		result.Success = false
		result.Errors = []error{err}
	}

	return result, err
}

// cleanBuildArtifacts removes build artifacts and caches
func (s *BuildService) cleanBuildArtifacts() error {
	// Clean cache directory
	if s.config.Build.CacheDir != "" {
		if err := os.RemoveAll(s.config.Build.CacheDir); err != nil {
			return errors.FileOperationError("CLEAN", s.config.Build.CacheDir, "failed to clean cache directory", err)
		}
	}

	// Clean generated Go files
	for _, path := range s.config.Components.ScanPaths {
		// Check if the path exists before trying to clean it
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}
		if err := s.cleanGeneratedFiles(path); err != nil {
			return errors.FileOperationError("CLEAN_GENERATED", path, "failed to clean generated files", err)
		}
	}

	return nil
}

// cleanGeneratedFiles removes generated *_templ.go files
func (s *BuildService) cleanGeneratedFiles(path string) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(filePath) == ".go" {
			// Check if it's a generated templ file (ends with _templ.go)
			if len(info.Name()) > 9 && info.Name()[len(info.Name())-9:] == "_templ.go" {
				return os.Remove(filePath)
			}
		}
		return nil
	})
}

// scanComponents performs component scanning
func (s *BuildService) scanComponents(ctx context.Context, scanner interface{}) error {
	// Scan all configured paths
	for _, path := range s.config.Components.ScanPaths {
		if err := monitoring.TrackOperation(ctx, "build", "scan_path", func(ctx context.Context) error {
			// Use reflection or type assertion to call scanner methods
			// This is simplified - would need proper interface handling
			return nil
		}); err != nil {
			return errors.ScannerError("PATH", path, "failed to scan path", err)
		}
	}
	return nil
}

// buildComponents builds all components through the pipeline
func (s *BuildService) buildComponents(ctx context.Context, pipeline interface{}, components interface{}) error {
	// Build each component through the pipeline
	// This is simplified - would need proper interface handling
	return monitoring.TrackOperation(ctx, "build", "build_components", func(ctx context.Context) error {
		return nil
	})
}

// generateBuildAnalysis creates build analysis report
func (s *BuildService) generateBuildAnalysis(outputDir string) error {
	analysisFile := filepath.Join(outputDir, "build-analysis.json")

	// Create analysis data
	analysis := map[string]interface{}{
		"timestamp":       time.Now(),
		"build_config":    s.config.Build,
		"component_count": 0,    // Would be filled with actual data
		"build_time":      "0s", // Would be filled with actual data
	}

	// Write analysis file (simplified implementation)
	_ = analysis
	_ = analysisFile

	return nil
}

// applyProductionOptimizations applies production-specific optimizations
func (s *BuildService) applyProductionOptimizations(ctx context.Context, outputDir string) error {
	return monitoring.TrackOperation(ctx, "build", "production_optimize", func(ctx context.Context) error {
		// Minify CSS
		if err := s.minifyCSS(outputDir); err != nil {
			return errors.BuildServiceError("MINIFY_CSS", "CSS minification failed", err)
		}

		// Compress assets
		if err := s.compressAssets(outputDir); err != nil {
			return errors.BuildServiceError("COMPRESS_ASSETS", "asset compression failed", err)
		}

		// Generate manifest
		if err := s.generateManifest(outputDir); err != nil {
			return errors.BuildServiceError("GENERATE_MANIFEST", "manifest generation failed", err)
		}

		return nil
	})
}

// minifyCSS minifies CSS files for production
func (s *BuildService) minifyCSS(outputDir string) error {
	// Simplified implementation - would use actual CSS minifier
	return nil
}

// compressAssets compresses static assets
func (s *BuildService) compressAssets(outputDir string) error {
	// Simplified implementation - would use gzip/brotli compression
	return nil
}

// generateManifest creates asset manifest for production
func (s *BuildService) generateManifest(outputDir string) error {
	// Simplified implementation - would create manifest.json
	return nil
}
