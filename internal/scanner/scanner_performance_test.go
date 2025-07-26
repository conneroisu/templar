package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
)

// BenchmarkPathValidationOriginal benchmarks the original path validation approach.
func BenchmarkPathValidationOriginal(b *testing.B) {
	testPaths := generateTestPaths(100)

	b.ResetTimer()

	for i := range b.N {
		path := testPaths[i%len(testPaths)]
		_, _ = validatePathOriginal(path)
	}
}

// BenchmarkPathValidationOptimized benchmarks the optimized path validation with caching.
func BenchmarkPathValidationOptimized(b *testing.B) {
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	testPaths := generateTestPaths(100)

	b.ResetTimer()

	for i := range b.N {
		path := testPaths[i%len(testPaths)]
		_, _ = scanner.validatePath(path)
	}
}

// BenchmarkPathValidationComparison runs both methods to show performance difference.
func BenchmarkPathValidationComparison(b *testing.B) {
	pathCounts := []int{10, 100, 1000}

	for _, pathCount := range pathCounts {
		b.Run(fmt.Sprintf("Original_%d_paths", pathCount), func(b *testing.B) {
			benchmarkOriginalWithPaths(b, pathCount)
		})

		b.Run(fmt.Sprintf("Optimized_%d_paths", pathCount), func(b *testing.B) {
			benchmarkOptimizedWithPaths(b, pathCount)
		})
	}
}

func benchmarkOriginalWithPaths(b *testing.B, pathCount int) {
	testPaths := generateTestPaths(pathCount)
	b.ResetTimer()

	for range b.N {
		for _, path := range testPaths {
			_, _ = validatePathOriginal(path)
		}
	}
}

func benchmarkOptimizedWithPaths(b *testing.B, pathCount int) {
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	testPaths := generateTestPaths(pathCount)

	b.ResetTimer()

	for range b.N {
		for _, path := range testPaths {
			_, _ = scanner.validatePath(path)
		}
	}
}

// BenchmarkDirectoryScanSimulation simulates real directory scanning workload.
func BenchmarkDirectoryScanSimulation(b *testing.B) {
	tempDir, cleanup := createTestDirectoryStructure(b)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		b.Fatal(err)
	}

	b.Run("Original_DirectoryScan", func(b *testing.B) {
		for range b.N {
			err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if filepath.Ext(path) == ".templ" {
					_, _ = validatePathOriginal(path)
				}

				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized_DirectoryScan", func(b *testing.B) {
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)

		for range b.N {
			err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if filepath.Ext(path) == ".templ" {
					_, _ = scanner.validatePath(path)
				}

				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func generateTestPaths(count int) []string {
	paths := make([]string, count)

	pathPatterns := []string{
		"components/button.templ",
		"views/layout.templ",
		"examples/demo.templ",
		"./components/card.templ",
		"../outside/dangerous.templ",
		"components/../views/page.templ",
		"deeply/nested/component/tree/item.templ",
		"simple.templ",
	}

	for i := range count {
		pattern := pathPatterns[i%len(pathPatterns)]
		if i > 0 {
			pattern = fmt.Sprintf("dir%d/%s", i%10, pattern)
		}
		paths[i] = pattern
	}

	return paths
}

func createTestDirectoryStructure(b *testing.B) (string, func()) {
	tempDir, err := os.MkdirTemp("", "templar-benchmark-*")
	if err != nil {
		b.Fatal(err)
	}

	dirs := []string{
		"components",
		"views",
		"examples",
		"deeply/nested/components",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			b.Fatal(err)
		}

		for i := range 10 {
			filename := filepath.Join(tempDir, dir, fmt.Sprintf("component%d.templ", i))
			content := fmt.Sprintf(
				"package components\n\ntempl Component%d() {\n\t<div>Component %d</div>\n}\n",
				i,
				i,
			)
			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				b.Fatal(err)
			}
		}
	}

	return tempDir, func() { os.RemoveAll(tempDir) }
}

// validatePathOriginal simulates the original implementation.
func validatePathOriginal(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	if !strings.HasPrefix(absPath, cwd) {
		return "", fmt.Errorf("path %s is outside current working directory", path)
	}

	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal: %s", path)
	}

	return cleanPath, nil
}

// TestPathValidationCorrectness ensures optimization doesn't break functionality.
func TestPathValidationCorrectness(t *testing.T) {
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)

	testCases := []struct {
		name        string
		path        string
		expectError bool
	}{
		{"valid_relative_path", "components/button.templ", false},
		{"valid_dot_relative", "./views/layout.templ", false},
		{"directory_traversal_attack", "../../../etc/passwd", true},
		{
			"dot_dot_in_path",
			"components/../views/page.templ",
			false,
		}, // This gets cleaned to "views/page.templ" which is valid
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := scanner.validatePath(tc.path)
			hasError := err != nil

			if hasError != tc.expectError {
				t.Errorf("Expected error=%v, got error=%v for path %s: %v",
					tc.expectError, hasError, tc.path, err)
			}
		})
	}
}
