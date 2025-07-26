// Package build provides hash generation and file I/O optimization tests.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHashProvider(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.cache)
	assert.NotNil(t, provider.crcTable)
	assert.NotNil(t, provider.fileMmaps)
}

func TestHashProvider_GenerateContentHash(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:    "small_file_basic_content",
			content: "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
		},
		{
			name:    "empty_file",
			content: "",
		},
		{
			name:    "single_character",
			content: "a",
		},
		{
			name:    "unicode_content",
			content: "こんにちは世界\n🎉🚀✨",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, tt.name+".templ")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			component := &types.ComponentInfo{
				Name:     tt.name,
				FilePath: testFile,
			}

			// Generate hash
			hash1 := provider.GenerateContentHash(component)
			assert.NotEmpty(t, hash1)
			assert.NotEqual(t, testFile, hash1) // Should not fallback to filepath

			// Second call should use cache and return same hash
			hash2 := provider.GenerateContentHash(component)
			assert.Equal(t, hash1, hash2)

			// Verify cache hit by checking cache stats
			stats := provider.GetCacheStats()
			assert.Greater(t, stats.MetadataHits, int64(0))
		})
	}
}

func TestHashProvider_GenerateContentHash_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create large file (>64KB to trigger mmap)
	largeContent := strings.Repeat("This is a test line for memory mapping optimization.\n", 2000)
	testFile := filepath.Join(tempDir, "large_file.templ")
	err := os.WriteFile(testFile, []byte(largeContent), 0644)
	require.NoError(t, err)

	component := &types.ComponentInfo{
		Name:     "large_component",
		FilePath: testFile,
	}

	hash := provider.GenerateContentHash(component)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, testFile, hash)

	// Verify file size is indeed large enough to trigger mmap
	stat, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(64*1024))
}

func TestHashProvider_GenerateContentHash_FileNotFound(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	component := &types.ComponentInfo{
		Name:     "nonexistent",
		FilePath: "/path/to/nonexistent/file.templ",
	}

	hash := provider.GenerateContentHash(component)
	// Should fallback to filepath
	assert.Equal(t, component.FilePath, hash)
}

func TestHashProvider_readFileWithMmap(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create test file
	testContent := "This is test content for memory mapping"
	testFile := filepath.Join(tempDir, "mmap_test.txt")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer file.Close()

	stat, err := file.Stat()
	require.NoError(t, err)

	content, err := provider.readFileWithMmap(file, stat.Size())
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestHashProvider_readFileWithMmap_InvalidFd(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create a file and close it to get an invalid fd
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	file, err := os.Open(testFile)
	require.NoError(t, err)
	stat, err := file.Stat()
	require.NoError(t, err)
	file.Close() // Close to make fd invalid

	_, err = provider.readFileWithMmap(file, stat.Size())
	assert.Error(t, err) // Should fail with invalid fd
}

func TestHashProvider_GenerateHashBatch(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create multiple test files
	components := make([]*types.ComponentInfo, 0, 10)
	for i := range 10 {
		content := fmt.Sprintf("Component %d content\npackage component%d", i, i)
		testFile := filepath.Join(tempDir, fmt.Sprintf("comp%d.templ", i))
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		components = append(components, &types.ComponentInfo{
			Name:     fmt.Sprintf("Component%d", i),
			FilePath: testFile,
		})
	}

	// Generate hashes for all components
	results := provider.GenerateHashBatch(components)

	// Verify results
	assert.Len(t, results, 10)
	for i := range 10 {
		componentName := fmt.Sprintf("Component%d", i)
		hash, exists := results[componentName]
		assert.True(t, exists, "Hash should exist for component %s", componentName)
		assert.NotEmpty(t, hash)
	}

	// Second batch call should use cache
	results2 := provider.GenerateHashBatch(components)
	assert.Equal(t, results, results2)

	// Verify cache performance
	stats := provider.GetCacheStats()
	assert.Greater(t, stats.MetadataHits, int64(0))
	assert.Greater(t, stats.HitRatio, 0.0)
}

func TestHashProvider_GenerateHashBatch_SmallBatch(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create small batch (≤5 components) to test synchronous processing
	components := make([]*types.ComponentInfo, 0, 3)
	for i := range 3 {
		content := fmt.Sprintf("Small batch component %d", i)
		testFile := filepath.Join(tempDir, fmt.Sprintf("small%d.templ", i))
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		components = append(components, &types.ComponentInfo{
			Name:     fmt.Sprintf("SmallComponent%d", i),
			FilePath: testFile,
		})
	}

	results := provider.GenerateHashBatch(components)
	assert.Len(t, results, 3)

	for i := range 3 {
		componentName := fmt.Sprintf("SmallComponent%d", i)
		hash := results[componentName]
		assert.NotEmpty(t, hash)
	}
}

func TestHashProvider_GenerateHashBatch_EmptyBatch(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	results := provider.GenerateHashBatch([]*types.ComponentInfo{})
	assert.Empty(t, results)
}

func TestHashProvider_GenerateHashBatch_AllCached(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create components
	components := make([]*types.ComponentInfo, 0, 5)
	for i := range 5 {
		content := fmt.Sprintf("Cached component %d", i)
		testFile := filepath.Join(tempDir, fmt.Sprintf("cached%d.templ", i))
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		components = append(components, &types.ComponentInfo{
			Name:     fmt.Sprintf("CachedComponent%d", i),
			FilePath: testFile,
		})
	}

	// First call to populate cache
	results1 := provider.GenerateHashBatch(components)
	assert.Len(t, results1, 5)

	// Second call should use all cached results
	results2 := provider.GenerateHashBatch(components)
	assert.Equal(t, results1, results2)

	// Verify cache is being used
	stats := provider.GetCacheStats()
	assert.Greater(t, stats.MetadataHits, int64(0)) // Should have some cache hits
}

func TestHashProvider_ClearMmapCache(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Add some entries to mmap cache
	provider.fileMmaps["test1"] = []byte("test content")
	provider.fileMmaps["test2"] = []byte("more content")
	assert.Len(t, provider.fileMmaps, 2)

	// Clear cache
	provider.ClearMmapCache()
	assert.Empty(t, provider.fileMmaps)
}

func TestHashProvider_GetCacheStats(t *testing.T) {
	t.Run("with_cache", func(t *testing.T) {
		cache := NewBuildCache(1024*1024, 5*time.Minute)
		provider := NewHashProvider(cache)

		// Add some cache activity
		cache.Set("test1", []byte("content"))
		cache.Get("test1") // Hit
		cache.Get("test2") // Miss

		stats := provider.GetCacheStats()
		assert.Greater(t, stats.MetadataHits, int64(0))
		assert.Greater(t, stats.MetadataMisses, int64(0))
		assert.Greater(t, stats.HitRatio, 0.0)
		assert.Greater(t, stats.Size, int64(0))
		assert.Equal(t, int64(100*1024*1024), stats.MaxSize)
	})

	t.Run("without_cache", func(t *testing.T) {
		provider := &HashProvider{cache: nil}
		stats := provider.GetCacheStats()
		assert.Equal(t, HashCacheStats{}, stats)
	})
}

func TestHashProvider_ContentHashConsistency(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create test file
	testContent := "consistent content for hashing"
	testFile := filepath.Join(tempDir, "consistency_test.templ")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	component := &types.ComponentInfo{
		Name:     "ConsistentComponent",
		FilePath: testFile,
	}

	// Generate hash multiple times
	hash1 := provider.GenerateContentHash(component)
	hash2 := provider.GenerateContentHash(component)
	hash3 := provider.GenerateContentHash(component)

	// All hashes should be identical
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
}

func TestHashProvider_FileModificationDetection(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	testFile := filepath.Join(tempDir, "modification_test.templ")
	component := &types.ComponentInfo{
		Name:     "ModificationComponent",
		FilePath: testFile,
	}

	// Create initial file
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)
	hash1 := provider.GenerateContentHash(component)

	// Wait a bit and modify file
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	hash2 := provider.GenerateContentHash(component)

	// Hashes should be different for different content
	assert.NotEqual(t, hash1, hash2)
}

func TestHashProvider_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create test files
	components := make([]*types.ComponentInfo, 20)
	for i := range 20 {
		content := fmt.Sprintf("Concurrent test content %d", i)
		testFile := filepath.Join(tempDir, fmt.Sprintf("concurrent%d.templ", i))
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		components[i] = &types.ComponentInfo{
			Name:     fmt.Sprintf("ConcurrentComponent%d", i),
			FilePath: testFile,
		}
	}

	// Run concurrent hash generation
	done := make(chan bool, 10)
	for i := range 10 {
		go func(workerID int) {
			defer func() { done <- true }()

			// Each worker processes different components
			startIdx := workerID * 2
			endIdx := startIdx + 2
			batch := components[startIdx:endIdx]

			results := provider.GenerateHashBatch(batch)
			assert.Len(t, results, 2)

			// Verify results are consistent
			for _, component := range batch {
				hash := results[component.Name]
				assert.NotEmpty(t, hash)

				// Generate same hash individually and compare
				individualHash := provider.GenerateContentHash(component)
				assert.Equal(t, hash, individualHash)
			}
		}(i)
	}

	// Wait for all workers to complete
	for range 10 {
		<-done
	}
}

func TestHashProvider_MmapFallback(t *testing.T) {
	// This test verifies that the provider gracefully falls back to regular
	// file reading when mmap fails
	tempDir := t.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create large content to trigger mmap path
	largeContent := strings.Repeat("test content\n", 10000)
	testFile := filepath.Join(tempDir, "mmap_fallback.templ")
	err := os.WriteFile(testFile, []byte(largeContent), 0644)
	require.NoError(t, err)

	component := &types.ComponentInfo{
		Name:     "MmapFallbackComponent",
		FilePath: testFile,
	}

	// Should work even if mmap has issues (fallback to regular read)
	hash := provider.GenerateContentHash(component)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, testFile, hash)

	// Verify file is large enough to potentially trigger mmap
	stat, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(64*1024))
}

func TestHashProvider_ErrorHandling(t *testing.T) {
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	tests := []struct {
		name         string
		component    *types.ComponentInfo
		expectedHash string
	}{
		{
			name: "nonexistent_file",
			component: &types.ComponentInfo{
				Name:     "NonexistentComponent",
				FilePath: "/does/not/exist/file.templ",
			},
			expectedHash: "/does/not/exist/file.templ",
		},
		{
			name: "empty_filepath",
			component: &types.ComponentInfo{
				Name:     "EmptyPathComponent",
				FilePath: "",
			},
			expectedHash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := provider.GenerateContentHash(tt.component)
			assert.Equal(t, tt.expectedHash, hash)
		})
	}
}

// Benchmark tests.
func BenchmarkHashProvider_GenerateContentHash(b *testing.B) {
	tempDir := b.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create test file
	content := "benchmark test content for hash generation"
	testFile := filepath.Join(tempDir, "benchmark.templ")
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(b, err)

	component := &types.ComponentInfo{
		Name:     "BenchmarkComponent",
		FilePath: testFile,
	}

	b.ResetTimer()
	for range b.N {
		_ = provider.GenerateContentHash(component)
	}
}

func BenchmarkHashProvider_GenerateHashBatch(b *testing.B) {
	tempDir := b.TempDir()
	cache := NewBuildCache(1024*1024, 5*time.Minute)
	provider := NewHashProvider(cache)

	// Create multiple components
	components := make([]*types.ComponentInfo, 10)
	for i := range 10 {
		content := fmt.Sprintf("Benchmark batch content %d", i)
		testFile := filepath.Join(tempDir, fmt.Sprintf("batch%d.templ", i))
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(b, err)

		components[i] = &types.ComponentInfo{
			Name:     fmt.Sprintf("BatchComponent%d", i),
			FilePath: testFile,
		}
	}

	b.ResetTimer()
	for range b.N {
		_ = provider.GenerateHashBatch(components)
	}
}

func BenchmarkHashProvider_MmapVsRegular(b *testing.B) {
	tempDir := b.TempDir()

	// Create large file
	largeContent := strings.Repeat(
		"This is benchmark content for comparing mmap vs regular read.\n",
		2000,
	)
	testFile := filepath.Join(tempDir, "large_benchmark.templ")
	err := os.WriteFile(testFile, []byte(largeContent), 0644)
	require.NoError(b, err)

	component := &types.ComponentInfo{
		Name:     "LargeBenchmarkComponent",
		FilePath: testFile,
	}

	b.Run("with_cache", func(b *testing.B) {
		cache := NewBuildCache(1024*1024, 5*time.Minute)
		provider := NewHashProvider(cache)

		b.ResetTimer()
		for range b.N {
			_ = provider.GenerateContentHash(component)
		}
	})

	b.Run("without_cache", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			// Create fresh provider each time to avoid caching
			cache := NewBuildCache(1, time.Nanosecond) // Tiny cache that expires immediately
			provider := NewHashProvider(cache)
			_ = provider.GenerateContentHash(component)
		}
	})
}
