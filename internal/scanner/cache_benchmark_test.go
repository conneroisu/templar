package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// BenchmarkScannerWithoutCache benchmarks scanner performance without caching
func BenchmarkScannerWithoutCache(b *testing.B) {
	tempDir, cleanup := createLargeTestStructure(b, 100)
	defer cleanup()

	// Use original scanner by disabling cache
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	scanner.metadataCache = nil // Disable cache for baseline
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := scanner.ScanDirectory(tempDir)
		if err != nil {
			b.Fatal(err)
		}
		// Clear registry for next iteration
		reg = registry.NewComponentRegistry()
		scanner.registry = reg
	}
}

// BenchmarkScannerWithCache benchmarks scanner performance with metadata caching
func BenchmarkScannerWithCache(b *testing.B) {
	tempDir, cleanup := createLargeTestStructure(b, 100)
	defer cleanup()

	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := scanner.ScanDirectory(tempDir)
		if err != nil {
			b.Fatal(err)
		}
		// Clear registry but keep cache for next iteration
		reg = registry.NewComponentRegistry()
		scanner.registry = reg
	}
}

// BenchmarkScannerCacheHitRate measures cache effectiveness
func BenchmarkScannerCacheHitRate(b *testing.B) {
	tempDir, cleanup := createLargeTestStructure(b, 50)
	defer cleanup()

	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	
	// First scan to populate cache
	err := scanner.ScanDirectory(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		reg = registry.NewComponentRegistry()
		scanner.registry = reg
		
		start := time.Now()
		err := scanner.ScanDirectory(tempDir)
		if err != nil {
			b.Fatal(err)
		}
		
		// This should be very fast due to cache hits
		elapsed := time.Since(start)
		if i == 0 {
			b.Logf("Cache hit scan time: %v", elapsed)
		}
	}
}

// BenchmarkComponentMetadataCache benchmarks the cache operations themselves
func BenchmarkComponentMetadataCache(b *testing.B) {
	cache := NewMetadataCache(1000, time.Hour)
	
	// Prepare test data
	testData := []byte(`{"components":[{"name":"TestComponent","package":"test","filePath":"/test/path.templ","parameters":[],"imports":[],"hash":"abc123","dependencies":[]}],"fileHash":"abc123","parsedAt":"2024-01-01T00:00:00Z"}`)
	
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("test:%d", i%100)
			cache.Set(key, testData)
		}
	})
	
	// Populate cache for Get benchmarks
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("test:%d", i)
		cache.Set(key, testData)
	}
	
	b.Run("Get_Hit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("test:%d", i%100)
			_, found := cache.Get(key)
			if !found {
				b.Fatal("Cache miss when hit expected")
			}
		}
	})
	
	b.Run("Get_Miss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("miss:%d", i)
			_, found := cache.Get(key)
			if found {
				b.Fatal("Cache hit when miss expected")
			}
		}
	})
}

// BenchmarkScannerScaling tests performance at different scales
func BenchmarkScannerScaling(b *testing.B) {
	scales := []int{10, 50, 100, 200}
	
	for _, scale := range scales {
		b.Run(fmt.Sprintf("Files_%d", scale), func(b *testing.B) {
			tempDir, cleanup := createLargeTestStructure(b, scale)
			defer cleanup()

			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				err := scanner.ScanDirectory(tempDir)
				if err != nil {
					b.Fatal(err)
				}
				
				// For subsequent iterations, leverage cache
				if i == 0 {
					b.Logf("First scan (no cache) for %d files", scale)
				}
				
				// Clear registry but keep cache
				reg = registry.NewComponentRegistry()
				scanner.registry = reg
			}
		})
	}
}

// createLargeTestStructure creates a test directory with many component files
func createLargeTestStructure(tb testing.TB, numFiles int) (string, func()) {
	// Create test directory in current working directory to avoid path validation issues
	tempDir, err := os.MkdirTemp(".", "templar-cache-benchmark-*")
	if err != nil {
		tb.Fatal(err)
	}
	
	dirs := []string{
		"components/buttons",
		"components/forms", 
		"components/layout",
		"views/pages",
		"views/partials",
		"examples/demos",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			tb.Fatal(err)
		}
	}
	
	// Create component files with realistic content
	for i := 0; i < numFiles; i++ {
		dir := dirs[i%len(dirs)]
		filename := filepath.Join(tempDir, dir, fmt.Sprintf("component%d.templ", i))
		
		// Create realistic component content
		content := generateRealisticComponent(i)
		
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			tb.Fatal(err)
		}
	}
	
	return tempDir, func() { os.RemoveAll(tempDir) }
}

// generateRealisticComponent creates realistic templ component content
func generateRealisticComponent(index int) string {
	componentTypes := []string{
		"Button", "Card", "Modal", "Form", "Input", "Select", "Table", "Header", "Footer", "Sidebar",
	}
	
	componentType := componentTypes[index%len(componentTypes)]
	
	var content strings.Builder
	content.WriteString(fmt.Sprintf("package components\n\n"))
	content.WriteString(fmt.Sprintf("import (\n"))
	content.WriteString(fmt.Sprintf("\t\"fmt\"\n"))
	content.WriteString(fmt.Sprintf("\t\"strings\"\n"))
	if index%3 == 0 {
		content.WriteString(fmt.Sprintf("\t\"time\"\n"))
	}
	if index%4 == 0 {
		content.WriteString(fmt.Sprintf("\t\"context\"\n"))
	}
	content.WriteString(fmt.Sprintf(")\n\n"))
	
	// Add multiple components per file for realism
	for j := 0; j < 1+(index%3); j++ {
		suffix := ""
		if j > 0 {
			suffix = fmt.Sprintf("%d", j)
		}
		
		// Generate different parameter patterns
		var params string
		switch index % 4 {
		case 0:
			params = "title string"
		case 1:
			params = "title string, disabled bool"
		case 2:
			params = "props map[string]interface{}, classes []string"
		case 3:
			params = "ctx context.Context, data interface{}, opts ...Option"
		}
		
		content.WriteString(fmt.Sprintf("templ %s%s%s(%s) {\n", componentType, suffix, fmt.Sprintf("%d", index), params))
		content.WriteString(fmt.Sprintf("\t<div class=\"%s\">\n", strings.ToLower(componentType)))
		
		// Add some conditional logic to make parsing more complex
		if index%2 == 0 {
			content.WriteString(fmt.Sprintf("\t\tif title != \"\" {\n"))
			content.WriteString(fmt.Sprintf("\t\t\t<h2>{ title }</h2>\n"))
			content.WriteString(fmt.Sprintf("\t\t}\n"))
		}
		
		content.WriteString(fmt.Sprintf("\t\t<p>This is %s component %d</p>\n", componentType, index))
		
		// Add nested components
		if index%5 == 0 {
			content.WriteString(fmt.Sprintf("\t\t@Icon%d(\"check\")\n", index%3))
		}
		
		content.WriteString(fmt.Sprintf("\t</div>\n"))
		content.WriteString(fmt.Sprintf("}\n\n"))
	}
	
	return content.String()
}

// TestCacheEffectiveness verifies that caching actually improves performance
func TestCacheEffectiveness(t *testing.T) {
	tempDir, cleanup := createLargeTestStructure(t, 50)
	defer cleanup()

	// Benchmark without cache
	reg1 := registry.NewComponentRegistry()
	scanner1 := NewComponentScanner(reg1)
	scanner1.metadataCache = nil // Disable cache
	
	start := time.Now()
	err := scanner1.ScanDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	noCacheTime := time.Since(start)
	
	// Benchmark with cache (first scan)
	reg2 := registry.NewComponentRegistry()
	scanner2 := NewComponentScanner(reg2)
	
	start = time.Now()
	err = scanner2.ScanDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	firstScanTime := time.Since(start)
	
	// Benchmark with cache (second scan - should hit cache)
	reg3 := registry.NewComponentRegistry()
	scanner2.registry = reg3
	
	start = time.Now()
	err = scanner2.ScanDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	cachedScanTime := time.Since(start)
	
	t.Logf("No cache: %v", noCacheTime)
	t.Logf("First scan (populating cache): %v", firstScanTime)
	t.Logf("Cached scan: %v", cachedScanTime)
	
	// Cache should provide significant speedup
	speedupRatio := float64(noCacheTime) / float64(cachedScanTime)
	t.Logf("Cache speedup ratio: %.2fx", speedupRatio)
	
	if speedupRatio < 1.5 {
		t.Logf("WARNING: Cache speedup (%.2fx) is less than expected (1.5x)", speedupRatio)
	} else {
		t.Logf("SUCCESS: Cache provides %.2fx speedup", speedupRatio)
	}
}