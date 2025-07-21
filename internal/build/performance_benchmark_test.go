package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// BenchmarkFileIOOptimization benchmarks the file I/O performance improvements
func BenchmarkFileIOOptimization(b *testing.B) {
	// Create test files
	tempDir := b.TempDir()
	components := createTestComponents(b, tempDir, 100)

	// Create pipeline with registry
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Test batch hash generation
		_ = pipeline.generateContentHashesBatch(components)
	}
}

// BenchmarkSingleFileIO benchmarks single file hash generation
func BenchmarkSingleFileIO(b *testing.B) {
	// Create test file
	tempDir := b.TempDir()
	components := createTestComponents(b, tempDir, 1)

	// Create pipeline with registry
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pipeline.generateContentHash(components[0])
	}
}

// BenchmarkBatchFileIO benchmarks batch file hash generation
func BenchmarkBatchFileIO(b *testing.B) {
	// Create test files
	tempDir := b.TempDir()
	components := createTestComponents(b, tempDir, 100)

	// Create pipeline with registry
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pipeline.generateContentHashesBatch(components)
	}
}

// BenchmarkLargeFilesMmap benchmarks memory-mapped file reading for large files
func BenchmarkLargeFilesMmap(b *testing.B) {
	// Create large test file (>64KB)
	tempDir := b.TempDir()
	components := createLargeTestComponents(b, tempDir, 1, 128*1024) // 128KB files

	// Create pipeline with registry
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pipeline.generateContentHash(components[0])
	}
}

// createTestComponents creates test component files
func createTestComponents(t testing.TB, tempDir string, count int) []*types.ComponentInfo {
	var components []*types.ComponentInfo

	for i := 0; i < count; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("component%d.templ", i))
		content := `package components

templ TestComponent() {
	<div class="test-component">
		<h1>Test Component</h1>
		<p>This is a test component for performance benchmarking.</p>
	</div>
}
`

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		components = append(components, &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: filePath,
			Package:  "components",
		})
	}

	return components
}

// createLargeTestComponents creates large test component files for mmap testing
func createLargeTestComponents(t testing.TB, tempDir string, count int, size int) []*types.ComponentInfo {
	var components []*types.ComponentInfo

	baseContent := `package components

templ LargeComponent() {
	<div class="large-component">
		<h1>Large Test Component</h1>
`

	// Create content of specified size
	for len(baseContent) < size {
		baseContent += `		<p>This is padding content to make the file large enough for mmap testing. ` +
			`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor ` +
			`incididunt ut labore et dolore magna aliqua.</p>
`
	}

	baseContent += `	</div>
}
`

	for i := 0; i < count; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("large_component%d.templ", i))

		if err := os.WriteFile(filePath, []byte(baseContent), 0644); err != nil {
			t.Fatalf("Failed to create large test file: %v", err)
		}

		components = append(components, &types.ComponentInfo{
			Name:     "LargeComponent",
			FilePath: filePath,
			Package:  "components",
		})
	}

	return components
}

// TestPerformanceImprovement validates that our optimizations provide measurable performance gains
func TestPerformanceImprovement(t *testing.T) {
	// Create test files
	tempDir := t.TempDir()
	components := createTestComponents(t, tempDir, 50)

	// Create pipeline with registry
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)

	// Measure batch processing time
	start := time.Now()
	results := pipeline.generateContentHashesBatch(components)
	batchDuration := time.Since(start)

	// Measure individual processing time
	start = time.Now()
	for _, component := range components {
		_ = pipeline.generateContentHash(component)
	}
	individualDuration := time.Since(start)

	// Batch processing should be faster due to cache efficiency
	if len(results) != len(components) {
		t.Errorf("Expected %d results, got %d", len(components), len(results))
	}

	t.Logf("Batch processing: %v", batchDuration)
	t.Logf("Individual processing: %v", individualDuration)

	// Cache should make second batch much faster
	start = time.Now()
	_ = pipeline.generateContentHashesBatch(components)
	cachedDuration := time.Since(start)

	t.Logf("Cached processing: %v", cachedDuration)

	if cachedDuration >= batchDuration {
		t.Errorf("Expected cached processing to be faster than initial batch processing")
	}
}
