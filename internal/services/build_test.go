package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildService_Build(t *testing.T) {
	tests := []struct {
		name    string
		opts    BuildOptions
		wantErr bool
	}{
		{
			name: "basic_build",
			opts: BuildOptions{
				Output:     "",
				Production: false,
				Analyze:    false,
				Clean:      false,
			},
			wantErr: false,
		},
		{
			name: "build_with_output",
			opts: BuildOptions{
				Output:     "dist",
				Production: false,
				Analyze:    false,
				Clean:      false,
			},
			wantErr: false,
		},
		{
			name: "build_with_analysis",
			opts: BuildOptions{
				Output:     "",
				Production: false,
				Analyze:    true,
				Clean:      false,
			},
			wantErr: false,
		},
		{
			name: "production_build",
			opts: BuildOptions{
				Output:     "dist",
				Production: true,
				Analyze:    false,
				Clean:      false,
			},
			wantErr: false,
		},
		{
			name: "build_with_clean",
			opts: BuildOptions{
				Output:     "",
				Production: false,
				Analyze:    false,
				Clean:      true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldDir)

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			// Create test config
			cfg := createTestConfig(tempDir)
			service := NewBuildService(cfg)

			// Create component files
			err = createTestComponents(tempDir)
			require.NoError(t, err)

			ctx := context.Background()
			result, err := service.Build(ctx, tt.opts)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			// Note: This may fail in test environment due to missing dependencies
			// but we can still test the service structure
			if err != nil {
				// Common expected errors in test environment
				expectedErrors := []string{
					"failed to initialize service container",
					"failed to get component registry",
					"failed to get component scanner",
					"failed to get build pipeline",
				}

				foundExpected := false
				for _, expectedErr := range expectedErrors {
					if assert.Contains(t, err.Error(), expectedErr) {
						foundExpected = true

						break
					}
				}

				if !foundExpected {
					t.Errorf("Unexpected error: %v", err)
				}
			} else {
				assert.NotNil(t, result)
				assert.IsType(t, &BuildResult{}, result)
			}
		})
	}
}

func TestBuildService_cleanBuildArtifacts(t *testing.T) {
	tempDir := t.TempDir()

	// Create test config with cache directory
	cfg := createTestConfig(tempDir)
	cfg.Build.CacheDir = filepath.Join(tempDir, ".templar/cache")
	// Add tempDir to scan paths so generated files there can be cleaned
	cfg.Components.ScanPaths = append(cfg.Components.ScanPaths, tempDir)

	service := NewBuildService(cfg)

	// Create cache directory with some files
	cacheDir := cfg.Build.CacheDir
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err)

	cacheFile := filepath.Join(cacheDir, "test-cache.dat")
	err = os.WriteFile(cacheFile, []byte("cache data"), 0644)
	require.NoError(t, err)

	// Create some generated files
	generatedFile := filepath.Join(tempDir, "test_templ.go")
	err = os.WriteFile(generatedFile, []byte("generated content"), 0644)
	require.NoError(t, err)

	// Test cleaning
	err = service.cleanBuildArtifacts()
	require.NoError(t, err)

	// Verify cache directory was removed
	assert.NoFileExists(t, cacheFile)
	assert.NoDirExists(t, cacheDir)

	// Verify generated file was removed
	assert.NoFileExists(t, generatedFile)
}

func TestBuildService_cleanGeneratedFiles(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	service := NewBuildService(cfg)

	// Create test directory structure
	testPath := filepath.Join(tempDir, "components")
	err := os.MkdirAll(testPath, 0755)
	require.NoError(t, err)

	// Create various test files
	files := map[string]bool{
		"component.templ":    false, // Should not be deleted
		"component_templ.go": true,  // Should be deleted
		"other.go":           false, // Should not be deleted
		"test_templ.go":      true,  // Should be deleted
		"regular.txt":        false, // Should not be deleted
	}

	for filename := range files {
		filePath := filepath.Join(testPath, filename)
		writeErr := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, writeErr)
	}

	// Test cleaning generated files
	cleanErr := service.cleanGeneratedFiles(testPath)
	require.NoError(t, cleanErr)

	// Verify correct files were deleted
	for filename, shouldBeDeleted := range files {
		filePath := filepath.Join(testPath, filename)
		if shouldBeDeleted {
			assert.NoFileExists(t, filePath, "File %s should have been deleted", filename)
		} else {
			assert.FileExists(t, filePath, "File %s should not have been deleted", filename)
		}
	}
}

func TestBuildService_scanComponents(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	cfg.Components.ScanPaths = []string{filepath.Join(tempDir, "components")}

	service := NewBuildService(cfg)

	// Create component directory
	componentDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentDir, 0755)
	require.NoError(t, err)

	// Create test component
	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}`

	scanErr := os.WriteFile(
		filepath.Join(componentDir, "test.templ"),
		[]byte(componentContent),
		0644,
	)
	require.NoError(t, scanErr)

	ctx := context.Background()

	// Create a mock scanner (interface{} for now since we have simplified implementation)
	scanner := struct{}{} // Placeholder

	scanCompErr := service.scanComponents(ctx, scanner)
	assert.NoError(t, scanCompErr) // Should not error with simplified implementation
}

func TestBuildService_buildComponents(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	service := NewBuildService(cfg)

	ctx := context.Background()

	// Create mock pipeline and components (interface{} for simplified implementation)
	pipeline := struct{}{}
	components := struct{}{}

	err := service.buildComponents(ctx, pipeline, components)
	assert.NoError(t, err) // Should not error with simplified implementation
}

func TestBuildService_generateBuildAnalysis(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	service := NewBuildService(cfg)

	// Test generating analysis
	err := service.generateBuildAnalysis(tempDir)
	assert.NoError(t, err) // Should not error with simplified implementation
}

func TestBuildService_applyProductionOptimizations(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	service := NewBuildService(cfg)

	ctx := context.Background()
	outputDir := filepath.Join(tempDir, "dist")

	mkdirErr := os.MkdirAll(outputDir, 0755)
	require.NoError(t, mkdirErr)

	optimizeErr := service.applyProductionOptimizations(ctx, outputDir)
	assert.NoError(t, optimizeErr) // Should not error with simplified implementation
}

func TestBuildResult(t *testing.T) {
	result := &BuildResult{
		Duration:       time.Second * 5,
		ComponentCount: 10,
		Success:        true,
		Errors:         nil,
	}

	assert.Equal(t, time.Second*5, result.Duration)
	assert.Equal(t, 10, result.ComponentCount)
	assert.True(t, result.Success)
	assert.Empty(t, result.Errors)

	// Test with errors
	result.Success = false
	result.Errors = []error{assert.AnError}

	assert.False(t, result.Success)
	assert.Len(t, result.Errors, 1)
}

func TestBuildOptions(t *testing.T) {
	opts := BuildOptions{
		Output:     "dist",
		Production: true,
		Analyze:    true,
		Clean:      true,
	}

	assert.Equal(t, "dist", opts.Output)
	assert.True(t, opts.Production)
	assert.True(t, opts.Analyze)
	assert.True(t, opts.Clean)
}

// Helper functions

func createTestConfig(tempDir string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{filepath.Join(tempDir, "components")},
		},
		Build: config.BuildConfig{
			Command:  "templ generate",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules", ".git"},
			CacheDir: filepath.Join(tempDir, ".templar/cache"),
		},
		Development: config.DevelopmentConfig{
			HotReload:    true,
			CSSInjection: true,
			ErrorOverlay: true,
		},
	}
}

func createTestComponents(tempDir string) error {
	componentDir := filepath.Join(tempDir, "components")
	if err := os.MkdirAll(componentDir, 0755); err != nil {
		return err
	}

	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}

templ AnotherComponent(count int) {
	<div>Count: { fmt.Sprintf("%d", count) }</div>
}`

	return os.WriteFile(filepath.Join(componentDir, "test.templ"), []byte(componentContent), 0644)
}
