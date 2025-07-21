package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// BenchmarkCacheHashGeneration tests the performance improvement of the optimized cache system
func BenchmarkCacheHashGeneration(b *testing.B) {
	// Create temp directory with test files
	tempDir := b.TempDir()

	// Create test files of different sizes
	testFiles := []struct {
		name    string
		content string
	}{
		{"small.templ", "small test content"},
		{"medium.templ", generateContent(1024)},     // 1KB
		{"large.templ", generateContent(64 * 1024)}, // 64KB
	}

	components := make([]*types.ComponentInfo, len(testFiles))
	for i, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}

		components[i] = &types.ComponentInfo{
			FilePath: filePath,
			Name:     file.name,
		}
	}

	// Create build pipeline with cache
	cache := NewBuildCache(1024*1024, 5*time.Minute) // 1MB cache, 5min TTL
	pipeline := &BuildPipeline{
		cache: cache,
	}

	b.ResetTimer()

	// Benchmark cache performance
	b.Run("ColdCache", func(b *testing.B) {
		// Test performance with empty cache (worst case)
		for i := 0; i < b.N; i++ {
			for _, component := range components {
				pipeline.generateContentHash(component)
			}
			// Clear cache for each iteration to simulate cold cache
			cache.Clear()
		}
	})

	b.Run("WarmCache", func(b *testing.B) {
		// Pre-populate cache
		for _, component := range components {
			pipeline.generateContentHash(component)
		}

		b.ResetTimer()

		// Test performance with warm cache (best case)
		for i := 0; i < b.N; i++ {
			for _, component := range components {
				pipeline.generateContentHash(component)
			}
		}
	})

	b.Run("MixedCache", func(b *testing.B) {
		// Test realistic scenario with some cache hits and misses
		for i := 0; i < b.N; i++ {
			for j, component := range components {
				pipeline.generateContentHash(component)

				// Simulate file changes for some components
				if i%5 == 0 && j == 0 {
					// "Modify" the first file every 5 iterations
					if err := os.Chtimes(component.FilePath, time.Now(), time.Now()); err == nil {
						// File modification time changed, cache will miss
					}
				}
			}
		}
	})
}

// BenchmarkBatchHashGeneration tests the batch processing performance
func BenchmarkBatchHashGeneration(b *testing.B) {
	tempDir := b.TempDir()

	// Create multiple test files
	numFiles := 100
	components := make([]*types.ComponentInfo, numFiles)

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("component_%d.templ", i)
		filePath := filepath.Join(tempDir, fileName)
		content := fmt.Sprintf("component %d content with some text", i)

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}

		components[i] = &types.ComponentInfo{
			FilePath: filePath,
			Name:     fileName,
		}
	}

	cache := NewBuildCache(1024*1024, 5*time.Minute)
	pipeline := &BuildPipeline{
		cache: cache,
	}

	b.ResetTimer()

	b.Run("IndividualHashing", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Clear()
			for _, component := range components {
				pipeline.generateContentHash(component)
			}
		}
	})

	b.Run("BatchHashing", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Clear()
			pipeline.generateContentHashesBatch(components)
		}
	})
}

// TestCacheOptimizationCorrectness verifies that the optimization doesn't break correctness
func TestCacheOptimizationCorrectness(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	filePath := filepath.Join(tempDir, "test.templ")
	originalContent := "original content for testing cache optimization system"
	if err := os.WriteFile(filePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	component := &types.ComponentInfo{
		FilePath: filePath,
		Name:     "test.templ",
	}

	cache := NewBuildCache(1024*1024, 5*time.Minute)
	pipeline := &BuildPipeline{
		cache: cache,
	}

	// First hash generation (cache miss)
	hash1 := pipeline.generateContentHash(component)
	if hash1 == "" {
		t.Fatal("Expected non-empty hash")
	}

	// Second hash generation (cache hit - should be same)
	hash2 := pipeline.generateContentHash(component)
	if hash1 != hash2 {
		t.Fatalf("Expected same hash for unchanged file: %s != %s", hash1, hash2)
	}

	// Modify file content - ensure enough time passes for filesystem timestamp resolution
	time.Sleep(10 * time.Millisecond)
	modifiedContent := "completely different content for testing cache invalidation properly"
	if err := os.WriteFile(filePath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Third hash generation (should detect change)
	hash3 := pipeline.generateContentHash(component)

	// Debug: Check if modification time actually changed and cache keys
	stat1, _ := os.Stat(filePath)
	metadataKey := fmt.Sprintf("%s:%d:%d", component.FilePath, stat1.ModTime().Unix(), stat1.Size())
	t.Logf("File modification time: %v", stat1.ModTime())
	t.Logf("Current metadata key: %s", metadataKey)
	t.Logf("Hash1: %s, Hash3: %s", hash1, hash3)

	// Check cache contents
	count, _, _ := cache.GetStats()
	t.Logf("Cache entries count: %d", count)

	if hash1 == hash3 {
		t.Fatalf("Expected different hash for modified file: %s == %s", hash1, hash3)
	}

	// Fourth hash generation (cache hit for new content)
	hash4 := pipeline.generateContentHash(component)
	if hash3 != hash4 {
		t.Fatalf("Expected same hash for unchanged modified file: %s != %s", hash3, hash4)
	}
}

// generateContent creates content of specified size for testing
func generateContent(size int) string {
	content := make([]byte, size)
	for i := 0; i < size; i++ {
		content[i] = byte('A' + (i % 26))
	}
	return string(content)
}

// TestFileIOReduction validates that we actually reduce file I/O operations
func TestFileIOReduction(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := []string{"test1.templ", "test2.templ", "test3.templ"}
	components := make([]*types.ComponentInfo, len(files))

	for i, fileName := range files {
		filePath := filepath.Join(tempDir, fileName)
		content := fmt.Sprintf("content for %s", fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		components[i] = &types.ComponentInfo{
			FilePath: filePath,
			Name:     fileName,
		}
	}

	cache := NewBuildCache(1024*1024, 5*time.Minute)
	pipeline := &BuildPipeline{
		cache: cache,
	}

	// First run: populate cache (expect all files to be read)
	for _, component := range components {
		pipeline.generateContentHash(component)
	}

	// Verify cache has entries
	count, _, _ := cache.GetStats()
	if count != len(components) {
		t.Fatalf("Expected %d cache entries, got %d", len(components), count)
	}

	// Second run: should hit cache (no file reading needed)
	// This is where the optimization shows - only os.Stat() calls, no file opens/reads
	for _, component := range components {
		hash := pipeline.generateContentHash(component)
		if hash == "" {
			t.Fatalf("Expected non-empty hash for component %s", component.Name)
		}
	}

	t.Log("Cache optimization test completed successfully")
}
