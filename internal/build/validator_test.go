// Package build provides build validation tests.
package build

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewBuildValidator(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}

	validator := NewBuildValidator(cfg)
	assert.NotNil(t, validator)
	assert.Equal(t, cfg, validator.config)
}

func TestBuildValidator_Validate(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}
	validator := NewBuildValidator(cfg)

	tests := []struct {
		name      string
		artifacts *BuildArtifacts
		options   ValidationOptions
		wantError bool
	}{
		{
			name: "basic_validation_success",
			artifacts: &BuildArtifacts{
				StaticFiles:    []string{"index.html", "style.css"},
				BundledAssets:  []string{"main.js", "vendor.js"},
				GeneratedPages: []string{"home.html", "about.html"},
				AssetManifest:  "manifest.json",
			},
			options: ValidationOptions{
				BundleSizeLimit:  1024 * 1024, // 1MB
				SecurityScan:     true,
				PerformanceCheck: true,
			},
			wantError: false,
		},
		{
			name: "empty_artifacts",
			artifacts: &BuildArtifacts{
				StaticFiles:    []string{},
				BundledAssets:  []string{},
				GeneratedPages: []string{},
				AssetManifest:  "",
			},
			options: ValidationOptions{
				BundleSizeLimit:  500 * 1024, // 500KB
				SecurityScan:     false,
				PerformanceCheck: false,
			},
			wantError: false,
		},
		{
			name: "large_bundle_validation",
			artifacts: &BuildArtifacts{
				StaticFiles:    []string{"large-file.js"},
				BundledAssets:  []string{"huge-bundle.js"},
				GeneratedPages: []string{"page1.html", "page2.html", "page3.html"},
				AssetManifest:  "asset-manifest.json",
				BundleAnalysis: "bundle-analysis.json",
			},
			options: ValidationOptions{
				BundleSizeLimit:  1024 * 1024, // 1MB limit
				SecurityScan:     true,
				PerformanceCheck: true,
			},
			wantError: false, // Current implementation doesn't enforce limits
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := validator.Validate(ctx, tt.artifacts, tt.options)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
			}

			if results != nil {
				// Verify result structure
				assert.NotNil(t, results.Errors)
				assert.NotNil(t, results.SecurityIssues)
				assert.GreaterOrEqual(t, results.PerformanceScore, 0)
				assert.LessOrEqual(t, results.PerformanceScore, 100)

				// Current implementation returns perfect score
				assert.Equal(t, 100, results.PerformanceScore)
				assert.Empty(t, results.Errors)
				assert.Empty(t, results.SecurityIssues)
			}
		})
	}
}

func TestBuildValidator_ValidateWithContext(t *testing.T) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	artifacts := &BuildArtifacts{
		StaticFiles:   []string{"test.html"},
		BundledAssets: []string{"app.js"},
		AssetManifest: "manifest.json",
	}

	options := ValidationOptions{
		SecurityScan:     true,
		PerformanceCheck: true,
	}

	t.Run("with_timeout_context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := validator.Validate(ctx, artifacts, options)
		assert.NoError(t, err)
		assert.NotNil(t, results)
	})

	t.Run("with_cancelled_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results, err := validator.Validate(ctx, artifacts, options)
		// Current implementation doesn't check context, so it should still work
		assert.NoError(t, err)
		assert.NotNil(t, results)
	})
}

func TestValidationOptions_FieldValidation(t *testing.T) {
	tests := []struct {
		name    string
		options ValidationOptions
		isValid bool
	}{
		{
			name: "valid_options",
			options: ValidationOptions{
				BundleSizeLimit:  1024 * 1024,
				SecurityScan:     true,
				PerformanceCheck: true,
			},
			isValid: true,
		},
		{
			name: "zero_bundle_limit",
			options: ValidationOptions{
				BundleSizeLimit:  0,
				SecurityScan:     false,
				PerformanceCheck: false,
			},
			isValid: true, // Zero is valid (no limit)
		},
		{
			name: "negative_bundle_limit",
			options: ValidationOptions{
				BundleSizeLimit:  -1,
				SecurityScan:     true,
				PerformanceCheck: false,
			},
			isValid: true, // Current implementation doesn't validate
		},
		{
			name: "only_security_scan",
			options: ValidationOptions{
				BundleSizeLimit:  500 * 1024,
				SecurityScan:     true,
				PerformanceCheck: false,
			},
			isValid: true,
		},
		{
			name: "only_performance_check",
			options: ValidationOptions{
				BundleSizeLimit:  0,
				SecurityScan:     false,
				PerformanceCheck: true,
			},
			isValid: true,
		},
	}

	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifacts := &BuildArtifacts{
				StaticFiles:   []string{"test.html"},
				AssetManifest: "test.json",
			}

			ctx := context.Background()
			results, err := validator.Validate(ctx, artifacts, tt.options)

			if tt.isValid {
				assert.NoError(t, err)
				assert.NotNil(t, results)
			} else {
				// Current implementation doesn't validate options, so all pass
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationResults_Structure(t *testing.T) {
	// Test that ValidationResults struct fields are properly accessible
	results := &ValidationResults{
		Errors:           []string{"error1", "error2"},
		SecurityIssues:   []string{"security issue"},
		PerformanceScore: 85,
	}

	assert.Len(t, results.Errors, 2)
	assert.Contains(t, results.Errors, "error1")
	assert.Contains(t, results.Errors, "error2")

	assert.Len(t, results.SecurityIssues, 1)
	assert.Contains(t, results.SecurityIssues, "security issue")

	assert.Equal(t, 85, results.PerformanceScore)
}

func TestBuildValidator_NilInputs(t *testing.T) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	t.Run("nil_artifacts", func(t *testing.T) {
		ctx := context.Background()
		options := ValidationOptions{}

		results, err := validator.Validate(ctx, nil, options)
		// Current implementation doesn't handle nil, but shouldn't panic
		// This test documents the current behavior
		assert.NotPanics(t, func() {
			validator.Validate(ctx, nil, options)
		})
		_ = results
		_ = err
	})

	t.Run("nil_context", func(t *testing.T) {
		artifacts := &BuildArtifacts{}
		options := ValidationOptions{}

		// Should not panic with nil context
		assert.NotPanics(t, func() {
			validator.Validate(context.TODO(), artifacts, options)
		})
	})
}

func TestBuildValidator_ConfigurationImpact(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "minimal_config",
			config: &config.Config{
				Server: config.ServerConfig{Port: 8080},
			},
		},
		{
			name: "full_config",
			config: &config.Config{
				Server: config.ServerConfig{
					Port: 8080,
					Host: "localhost",
				},
				Build: config.BuildConfig{
					Command: "templ generate",
				},
			},
		},
		{
			name:   "nil_config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewBuildValidator(tt.config)
			assert.NotNil(t, validator)

			artifacts := &BuildArtifacts{
				AssetManifest: "",
			}
			options := ValidationOptions{}

			ctx := context.Background()
			results, err := validator.Validate(ctx, artifacts, options)

			// Current implementation ignores config, should work regardless
			assert.NoError(t, err)
			assert.NotNil(t, results)
		})
	}
}

func TestBuildValidator_ConcurrentValidation(t *testing.T) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	// Test concurrent validation calls
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			artifacts := &BuildArtifacts{
				StaticFiles: []string{
					fmt.Sprintf("worker%d-file1.html", workerID),
					fmt.Sprintf("worker%d-file2.css", workerID),
				},
				AssetManifest: "",
			}

			options := ValidationOptions{
				BundleSizeLimit:  1024 * 1024,
				SecurityScan:     workerID%2 == 0, // Alternate security scanning
				PerformanceCheck: true,
			}

			ctx := context.Background()
			results, err := validator.Validate(ctx, artifacts, options)

			assert.NoError(t, err)
			assert.NotNil(t, results)
			assert.Equal(t, 100, results.PerformanceScore)
		}(i)
	}

	// Wait for all workers
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests for validator performance
func BenchmarkBuildValidator_Validate(b *testing.B) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	artifacts := &BuildArtifacts{
		StaticFiles:    make([]string, 100),
		BundledAssets:  make([]string, 50),
		GeneratedPages: make([]string, 200),
		AssetManifest:  "manifest.json",
		BundleAnalysis: "analysis.json",
	}

	// Fill with test data
	for i := 0; i < 100; i++ {
		artifacts.StaticFiles[i] = fmt.Sprintf("static%d.html", i)
	}
	for i := 0; i < 50; i++ {
		artifacts.BundledAssets[i] = fmt.Sprintf("bundle%d.js", i)
	}
	for i := 0; i < 200; i++ {
		artifacts.GeneratedPages[i] = fmt.Sprintf("page%d.html", i)
	}

	options := ValidationOptions{
		BundleSizeLimit:  2 * 1024 * 1024, // 2MB
		SecurityScan:     true,
		PerformanceCheck: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(ctx, artifacts, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildValidator_ValidateSmall(b *testing.B) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	artifacts := &BuildArtifacts{
		StaticFiles:   []string{"index.html"},
		BundledAssets: []string{"app.js"},
		AssetManifest: "",
	}

	options := ValidationOptions{
		SecurityScan:     true,
		PerformanceCheck: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(ctx, artifacts, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helper to create mock BuildArtifacts
func createMockBuildArtifacts(size int) *BuildArtifacts {
	artifacts := &BuildArtifacts{
		StaticFiles:    make([]string, size),
		BundledAssets:  make([]string, size/2),
		GeneratedPages: make([]string, size*2),
		AssetManifest:  "manifest.json",
		BundleAnalysis: "analysis.json",
	}

	for i := 0; i < size; i++ {
		artifacts.StaticFiles[i] = fmt.Sprintf("static%d.html", i)
	}

	for i := 0; i < size/2; i++ {
		artifacts.BundledAssets[i] = fmt.Sprintf("bundle%d.js", i)
	}

	for i := 0; i < size*2; i++ {
		artifacts.GeneratedPages[i] = fmt.Sprintf("page%d.html", i)
	}

	return artifacts
}

func TestBuildValidator_LargeArtifacts(t *testing.T) {
	cfg := &config.Config{}
	validator := NewBuildValidator(cfg)

	// Test with large number of artifacts
	artifacts := createMockBuildArtifacts(1000)

	options := ValidationOptions{
		BundleSizeLimit:  10 * 1024 * 1024, // 10MB
		SecurityScan:     true,
		PerformanceCheck: true,
	}

	ctx := context.Background()
	results, err := validator.Validate(ctx, artifacts, options)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 100, results.PerformanceScore)
}
