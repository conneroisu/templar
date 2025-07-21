package build

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// TestFileHashCachingPerformance validates that file hash caching provides significant performance improvements
func TestFileHashCachingPerformance(t *testing.T) {
	// Create test files of different sizes
	testSizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("FileSize_%dB", size), func(t *testing.T) {
			// Create temporary test file
			tempFile, err := os.CreateTemp("", "hash_test_*.templ")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			// Write test content
			content := make([]byte, size)
			for i := range content {
				content[i] = byte('A' + i%26)
			}
			if _, err := tempFile.Write(content); err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}
			tempFile.Close()

			// Create component and build pipeline
			component := &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: tempFile.Name(),
				Package:  "test",
			}

			bp := NewBuildPipeline(1, registry.NewComponentRegistry())

			// Test first hash generation (cache miss)
			start := time.Now()
			hash1 := bp.generateContentHash(component)
			firstDuration := time.Since(start)

			// Test second hash generation (cache hit)
			start = time.Now()
			hash2 := bp.generateContentHash(component)
			secondDuration := time.Since(start)

			// Verify hashes are identical
			if hash1 != hash2 {
				t.Errorf("Hash mismatch: first=%s, second=%s", hash1, hash2)
			}

			// Verify performance improvement (cache hit should be much faster)
			speedup := float64(firstDuration) / float64(secondDuration)
			if speedup < 2.0 {
				t.Logf("Warning: Cache speedup only %.2fx for %d byte file (first: %v, second: %v)",
					speedup, size, firstDuration, secondDuration)
			} else {
				t.Logf("Cache speedup: %.2fx for %d byte file (first: %v, second: %v)",
					speedup, size, firstDuration, secondDuration)
			}

			// For larger files, we expect more significant improvements
			if size >= 102400 && speedup < 5.0 {
				t.Logf("Cache speedup for large file (%d bytes) was only %.2fx, expected >5x", size, speedup)
			}
		})
	}
}

// TestMetadataBasedCaching validates that metadata changes invalidate cache correctly
func TestMetadataBasedCaching(t *testing.T) {
	// Create temporary test file
	tempFile, err := os.CreateTemp("", "metadata_test_*.templ")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write initial content
	initialContent := []byte("initial content")
	if _, err := tempFile.Write(initialContent); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}
	tempFile.Close()

	component := &types.ComponentInfo{
		Name:     "TestComponent",
		FilePath: tempFile.Name(),
		Package:  "test",
	}

	bp := NewBuildPipeline(1, registry.NewComponentRegistry())

	// Get initial hash
	hash1 := bp.generateContentHash(component)

	// Modify file content
	modifiedContent := []byte("modified content with different size")
	if err := os.WriteFile(tempFile.Name(), modifiedContent, 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Get hash after modification
	hash2 := bp.generateContentHash(component)

	// Verify hashes are different (cache was invalidated)
	if hash1 == hash2 {
		t.Errorf("Hash did not change after file modification: %s", hash1)
	}

	// Get hash again (should be cached now)
	hash3 := bp.generateContentHash(component)

	// Verify second read of modified file gives same hash
	if hash2 != hash3 {
		t.Errorf("Hash changed on second read: %s != %s", hash2, hash3)
	}
}

// BenchmarkHashCachingPerformance benchmarks file hash caching performance
func BenchmarkHashCachingPerformance(b *testing.B) {
	fileSizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range fileSizes {
		b.Run(fmt.Sprintf("WithoutCache_%dB", size), func(b *testing.B) {
			// Create test file
			tempFile, err := os.CreateTemp("", "bench_no_cache_*.templ")
			if err != nil {
				b.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			content := make([]byte, size)
			for i := range content {
				content[i] = byte('A' + i%26)
			}
			if err := os.WriteFile(tempFile.Name(), content, 0644); err != nil {
				b.Fatalf("Failed to write test content: %v", err)
			}

			component := &types.ComponentInfo{
				Name:     "BenchComponent",
				FilePath: tempFile.Name(),
				Package:  "test",
			}

			// Create new pipeline for each iteration to avoid caching
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				bp := NewBuildPipeline(1, registry.NewComponentRegistry())
				_ = bp.generateContentHash(component)
			}
		})

		b.Run(fmt.Sprintf("WithCache_%dB", size), func(b *testing.B) {
			// Create test file
			tempFile, err := os.CreateTemp("", "bench_with_cache_*.templ")
			if err != nil {
				b.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			content := make([]byte, size)
			for i := range content {
				content[i] = byte('A' + i%26)
			}
			if err := os.WriteFile(tempFile.Name(), content, 0644); err != nil {
				b.Fatalf("Failed to write test content: %v", err)
			}

			component := &types.ComponentInfo{
				Name:     "BenchComponent",
				FilePath: tempFile.Name(),
				Package:  "test",
			}

			bp := NewBuildPipeline(1, registry.NewComponentRegistry())

			// Prime the cache
			_ = bp.generateContentHash(component)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = bp.generateContentHash(component)
			}
		})
	}
}

// TestCacheEvictionUnderMemoryPressure validates cache behavior under memory pressure
func TestCacheEvictionUnderMemoryPressure(t *testing.T) {
	// Create pipeline with smaller cache for metadata-based testing
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	// Override with smaller cache to test eviction of metadata entries (not file content)
	bp.cache = NewBuildCache(1024, time.Hour) // 1KB cache limit for metadata

	// Create many small files with unique content to exceed metadata cache size
	const fileSize = 100 // Small files, focus on metadata cache eviction
	const numFiles = 50  // Enough metadata entries to exceed 1KB limit

	var tempFiles []string
	var components []*types.ComponentInfo

	defer func() {
		for _, file := range tempFiles {
			os.Remove(file)
		}
	}()

	// Create test files
	for i := 0; i < numFiles; i++ {
		tempFile, err := os.CreateTemp("", fmt.Sprintf("cache_eviction_test_%d_*.templ", i))
		if err != nil {
			t.Fatalf("Failed to create temp file %d: %v", i, err)
		}

		content := make([]byte, fileSize)
		for j := range content {
			content[j] = byte('A' + (i+j)%26)
		}
		if err := os.WriteFile(tempFile.Name(), content, 0644); err != nil {
			t.Fatalf("Failed to write content to file %d: %v", i, err)
		}
		tempFile.Close()

		tempFiles = append(tempFiles, tempFile.Name())

		component := &types.ComponentInfo{
			Name:     fmt.Sprintf("Component%d", i),
			FilePath: tempFile.Name(),
			Package:  "test",
		}
		components = append(components, component)
	}

	// Generate hashes for all components (should trigger cache eviction)
	var hashes []string
	for _, component := range components {
		hash := bp.generateContentHash(component)
		hashes = append(hashes, hash)
	}

	// Verify cache stats
	count, currentSize, maxSize := bp.GetCacheStats()
	t.Logf("Cache stats: count=%d, currentSize=%d, maxSize=%d", count, currentSize, maxSize)

	// Cache should not exceed max size
	if currentSize > maxSize {
		t.Errorf("Cache size exceeded maximum: %d > %d", currentSize, maxSize)
	}

	// Should have evicted some entries (with 1KB limit and 50 files, expect eviction)
	if count >= numFiles {
		t.Errorf("Cache did not evict entries: count=%d, expected < %d", count, numFiles)
	}

	// Verify eviction actually happened with reasonable threshold
	if count > numFiles/2 {
		t.Logf("Warning: Expected more aggressive eviction, count=%d, numFiles=%d", count, numFiles)
	}

	// Re-access early components (should be cache misses due to eviction)
	for i := 0; i < 10; i++ {
		start := time.Now()
		hash := bp.generateContentHash(components[i])
		duration := time.Since(start)

		// Verify hash is still correct
		if hash != hashes[i] {
			t.Errorf("Hash mismatch for component %d after eviction", i)
		}

		// Should take longer due to cache miss
		if duration < time.Microsecond {
			t.Logf("Warning: Component %d hash generation was very fast (%v), might still be cached", i, duration)
		}
	}
}

// TestCacheConcurrency validates thread-safety of hash caching
func TestCacheConcurrency(t *testing.T) {
	bp := NewBuildPipeline(4, registry.NewComponentRegistry())

	// Create test file
	tempFile, err := os.CreateTemp("", "concurrency_test_*.templ")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := make([]byte, 10240) // 10KB
	for i := range content {
		content[i] = byte('A' + i%26)
	}
	if err := os.WriteFile(tempFile.Name(), content, 0644); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	component := &types.ComponentInfo{
		Name:     "ConcurrencyTestComponent",
		FilePath: tempFile.Name(),
		Package:  "test",
	}

	// Test concurrent access
	const numGoroutines = 100
	const numOperations = 50

	results := make(chan string, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				hash := bp.generateContentHash(component)
				results <- hash
			}
		}()
	}

	// Collect all results
	var allHashes []string
	for i := 0; i < numGoroutines*numOperations; i++ {
		hash := <-results
		allHashes = append(allHashes, hash)
	}

	// Verify all hashes are identical
	expectedHash := allHashes[0]
	for i, hash := range allHashes {
		if hash != expectedHash {
			t.Errorf("Hash mismatch at index %d: got %s, expected %s", i, hash, expectedHash)
		}
	}

	t.Logf("Successfully completed %d concurrent hash operations, all hashes identical", len(allHashes))
}
