package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheRaceConditionFixes validates that the race condition fixes work correctly
func TestCacheRaceConditionFixes(t *testing.T) {
	t.Run("concurrent hash storage and retrieval", func(t *testing.T) {
		cache := NewBuildCache(1024*1024, time.Hour)

		const numGoroutines = 50
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// Start multiple goroutines doing concurrent hash operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("key_%d_%d", id, j)
					hash := fmt.Sprintf("hash_%d_%d", id, j)

					// Store hash
					cache.SetHash(key, hash)

					// Retrieve hash
					retrievedHash, found := cache.GetHash(key)
					if !found {
						errors <- fmt.Errorf("hash not found for key %s", key)
						return
					}
					if retrievedHash != hash {
						errors <- fmt.Errorf("hash mismatch for key %s: expected %s, got %s", key, hash, retrievedHash)
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Error(err)
		}

		// Verify cache statistics are consistent
		count, size, maxSize := cache.GetStats()
		assert.Greater(t, count, 0, "Cache should have entries")
		assert.Greater(t, size, int64(0), "Cache should have non-zero size")
		assert.Equal(t, int64(1024*1024), maxSize, "Max size should be as configured")
	})

	t.Run("pipeline concurrent hash generation", func(t *testing.T) {
		// Create temporary test files
		tempDir := t.TempDir()
		testFiles := make([]string, 10)
		components := make([]*types.ComponentInfo, 10)

		for i := 0; i < 10; i++ {
			fileName := fmt.Sprintf("test_%d.templ", i)
			filePath := filepath.Join(tempDir, fileName)
			content := fmt.Sprintf("test content %d", i)

			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)

			testFiles[i] = filePath
			components[i] = &types.ComponentInfo{
				Name:     fileName,
				FilePath: filePath,
				Package:  "test",
			}
		}

		// Create build pipeline
		reg := NewMockComponentRegistry()
		pipeline := NewBuildPipeline(4, reg)

		const numGoroutines = 20
		var wg sync.WaitGroup
		results := make(chan string, numGoroutines*len(components))
		errors := make(chan error, numGoroutines)

		// Generate hashes concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for _, component := range components {
					hash := pipeline.generateContentHash(component)
					if hash == "" {
						errors <- fmt.Errorf("empty hash generated for component %s", component.Name)
						return
					}
					results <- fmt.Sprintf("%s:%s", component.Name, hash)
				}
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		// Check for errors
		for err := range errors {
			t.Error(err)
		}

		// Verify all results are consistent
		hashMap := make(map[string]string)
		for result := range results {
			parts := strings.SplitN(result, ":", 2)
			if len(parts) != 2 {
				t.Errorf("Invalid result format: %s", result)
				continue
			}

			componentName := parts[0]
			hash := parts[1]

			if existingHash, exists := hashMap[componentName]; exists {
				assert.Equal(t, existingHash, hash,
					"Hash should be consistent for component %s", componentName)
			} else {
				hashMap[componentName] = hash
			}
		}

		// Verify we got hashes for all components
		assert.Equal(t, len(components), len(hashMap),
			"Should have hashes for all components")
	})

	t.Run("memory exhaustion prevention under race conditions", func(t *testing.T) {
		// Small cache to test memory limits
		cache := NewBuildCache(1024, time.Hour) // 1KB limit

		const numGoroutines = 10
		const largeDataSize = 200 // 200 bytes per entry
		var wg sync.WaitGroup

		// Try to overwhelm cache with concurrent large entries
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 20; j++ {
					key := fmt.Sprintf("large_key_%d_%d", id, j)
					// Create large hash string
					hash := fmt.Sprintf("%0*d", largeDataSize, id*1000+j)
					cache.SetHash(key, hash)
				}
			}(i)
		}

		wg.Wait()

		// Verify cache didn't exceed memory limits
		count, size, maxSize := cache.GetStats()
		assert.LessOrEqual(t, size, maxSize,
			"Cache size should not exceed maximum: %d > %d", size, maxSize)

		t.Logf("Cache stats after memory pressure test: count=%d, size=%d, maxSize=%d",
			count, size, maxSize)
	})

	t.Run("cache size consistency under concurrent eviction", func(t *testing.T) {
		cache := NewBuildCache(512, time.Hour) // Small cache to force eviction

		const numGoroutines = 15
		const operationsPerGoroutine = 50
		var wg sync.WaitGroup

		// Create many entries to force eviction races
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("evict_test_%d_%d", id, j)
					hash := fmt.Sprintf("hash_%d_%d_with_extra_content", id, j)
					cache.SetHash(key, hash)

					// Also do some reads to trigger LRU updates
					if j%3 == 0 {
						cache.GetHash(key)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify cache is in consistent state
		count, size, maxSize := cache.GetStats()
		assert.LessOrEqual(t, size, maxSize,
			"Cache size should be within limits after concurrent eviction")
		assert.GreaterOrEqual(t, count, 0, "Cache count should be non-negative")

		// Manually verify size calculation consistency
		// This tests that currentSize tracking matches actual entry sizes
		cache.mutex.RLock()
		calculatedSize := int64(0)
		for _, entry := range cache.entries {
			calculatedSize += entry.Size
		}
		cache.mutex.RUnlock()

		assert.Equal(t, calculatedSize, size,
			"Calculated size should match tracked size")
	})
}

// TestAtomicSizeOperations tests that size operations are atomic
func TestAtomicSizeOperations(t *testing.T) {
	cache := NewBuildCache(2048, time.Hour)

	const numGoroutines = 20
	const operationsPerGoroutine = 100
	var wg sync.WaitGroup

	// Concurrent size-affecting operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("atomic_test_%d_%d", id, j)
				hash := fmt.Sprintf("hash_%d_%d", id, j)

				// Set hash
				cache.SetHash(key, hash)

				// Get hash (triggers LRU update)
				cache.GetHash(key)

				// Update with different hash (triggers size recalculation)
				newHash := fmt.Sprintf("updated_%d_%d", id, j)
				cache.SetHash(key, newHash)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	count, size, maxSize := cache.GetStats()
	assert.LessOrEqual(t, size, maxSize, "Final size within limits")
	assert.Greater(t, count, 0, "Cache should have entries")

	// Verify no negative sizes (which would indicate race conditions)
	assert.GreaterOrEqual(t, size, int64(0), "Size should never be negative")
}

// TestCacheMethodThreadSafety verifies thread safety of cache methods
func TestCacheMethodThreadSafety(t *testing.T) {
	cache := NewBuildCache(4096, time.Minute)

	const numGoroutines = 25
	var wg sync.WaitGroup

	// Mix of all cache operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Different operation patterns per goroutine
			switch id % 4 {
			case 0: // Heavy setters
				for j := 0; j < 50; j++ {
					cache.SetHash(fmt.Sprintf("setter_%d_%d", id, j), fmt.Sprintf("hash_%d", j))
				}
			case 1: // Heavy getters
				for j := 0; j < 100; j++ {
					cache.GetHash(
						fmt.Sprintf("setter_%d_%d", (id-1+numGoroutines)%numGoroutines, j%50),
					)
				}
			case 2: // Stats checkers
				for j := 0; j < 30; j++ {
					cache.GetStats()
					time.Sleep(time.Microsecond)
				}
			case 3: // Cache clearers (less frequent)
				for j := 0; j < 5; j++ {
					cache.Clear()
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional after all operations
	cache.SetHash("test_final", "test_hash")
	hash, found := cache.GetHash("test_final")
	assert.True(t, found, "Cache should still be functional")
	assert.Equal(t, "test_hash", hash, "Hash should be correct")
}
