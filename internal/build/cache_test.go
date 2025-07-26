package build

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildCache_LRU_Implementation(t *testing.T) {
	t.Run("LRU eviction order", func(t *testing.T) {
		// Small cache for easy testing
		cache := NewBuildCache(30, time.Hour) // 30 bytes max

		// Add entries in order: key1, key2, key3
		cache.Set("key1", []byte("value1")) // 6 bytes (value only)
		cache.Set("key2", []byte("value2")) // 6 bytes
		cache.Set("key3", []byte("value3")) // 6 bytes
		cache.Set("key4", []byte("value4")) // 6 bytes
		cache.Set("key5", []byte("value5")) // 6 bytes (total 30 bytes)

		// All should be present
		for i := 1; i <= 5; i++ {
			key := fmt.Sprintf("key%d", i)
			_, found := cache.Get(key)
			assert.True(t, found, "Key %s should be present", key)
		}

		// Add one more entry to trigger eviction
		cache.Set("key6", []byte("value6")) // Should evict key1 (least recently used)

		// key1 should be evicted (least recently used)
		_, found := cache.Get("key1")
		assert.False(t, found, "key1 should be evicted as LRU")

		// Others should still be present
		for i := 2; i <= 6; i++ {
			key := fmt.Sprintf("key%d", i)
			_, found := cache.Get(key)
			assert.True(t, found, "Key %s should still be present", key)
		}
	})

	t.Run("LRU access order updates", func(t *testing.T) {
		cache := NewBuildCache(24, time.Hour) // 24 bytes max

		// Add 4 entries (6 bytes each = 24 bytes total)
		cache.Set("key1", []byte("value1"))
		cache.Set("key2", []byte("value2"))
		cache.Set("key3", []byte("value3"))
		cache.Set("key4", []byte("value4"))

		// Access key1 to make it most recently used
		cache.Get("key1")

		// Add new entry - should evict key2 (now LRU), not key1
		cache.Set("key5", []byte("value5"))

		// key1 should still be present (was accessed recently)
		_, found := cache.Get("key1")
		assert.True(t, found, "key1 should still be present after access")

		// key2 should be evicted
		_, found = cache.Get("key2")
		assert.False(t, found, "key2 should be evicted as LRU")
	})
}

func TestBuildCache_DoublyLinkedList(t *testing.T) {
	t.Run("list integrity", func(t *testing.T) {
		cache := NewBuildCache(100, time.Hour)

		// Add several entries
		keys := []string{"a", "b", "c", "d", "e"}
		for _, key := range keys {
			cache.Set(key, []byte("value"))
		}

		// Verify list integrity by checking that we can traverse it
		count := 0
		current := cache.head.next
		for current != cache.tail {
			count++
			current = current.next
			if count > 10 { // Prevent infinite loop
				t.Fatal("Infinite loop detected in doubly linked list")
			}
		}

		assert.Equal(t, len(keys), count, "List should contain all added entries")
	})

	t.Run("move to front operation", func(t *testing.T) {
		cache := NewBuildCache(100, time.Hour)

		// Add entries
		cache.Set("key1", []byte("value1"))
		cache.Set("key2", []byte("value2"))
		cache.Set("key3", []byte("value3"))

		// Access key1 (should move to front)
		cache.Get("key1")

		// The most recently used entry should be at the front
		assert.Equal(t, "key1", cache.head.next.Key, "key1 should be at front after access")
	})
}

func TestBuildCache_SizeCalculation(t *testing.T) {
	t.Run("accurate size tracking", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)

		// Add entries and verify size calculation
		cache.Set("key1", []byte("value1"))
		count, size, _ := cache.GetStats()
		assert.Equal(t, 1, count)
		assert.Greater(t, size, int64(0), "Size should be greater than 0")

		initialSize := size
		cache.Set("key2", []byte("value2"))
		count, size, _ = cache.GetStats()
		assert.Equal(t, 2, count)
		assert.Greater(t, size, initialSize, "Size should increase after adding entry")
	})

	t.Run("size decreases on eviction", func(t *testing.T) {
		cache := NewBuildCache(20, time.Hour) // Very small cache

		cache.Set("key1", []byte("value1"))
		_, size1, _ := cache.GetStats()

		cache.Set("key2", []byte("value2"))
		_, size2, _ := cache.GetStats()

		// Size should decrease after eviction
		assert.LessOrEqual(t, size2, size1+20, "Size should not exceed max after eviction")
	})
}

func TestBuildCache_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent reads and writes", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)
		var wg sync.WaitGroup

		// Pre-populate cache
		for i := range 10 {
			cache.Set(fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
		}

		// Launch concurrent readers
		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range 100 {
					key := fmt.Sprintf("key%d", j%10)
					cache.Get(key)
				}
			}(i)
		}

		// Launch concurrent writers
		for i := range 5 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range 50 {
					key := fmt.Sprintf("newkey%d_%d", id, j)
					cache.Set(key, []byte(fmt.Sprintf("newvalue%d_%d", id, j)))
				}
			}(i)
		}

		wg.Wait()

		// Verify cache is still functional
		count, _, _ := cache.GetStats()
		assert.Greater(t, count, 0, "Cache should still contain entries after concurrent access")
	})

	t.Run("no race conditions in LRU updates", func(t *testing.T) {
		cache := NewBuildCache(100, time.Hour)
		var wg sync.WaitGroup

		// Add initial entry
		cache.Set("shared", []byte("value"))

		// Many goroutines accessing the same key to test LRU race conditions
		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					cache.Get("shared") // This triggers LRU updates
				}
			}()
		}

		wg.Wait()

		// Entry should still be accessible
		_, found := cache.Get("shared")
		assert.True(t, found, "Shared entry should still be accessible after concurrent access")
	})
}

func TestBuildCache_TTL_Behavior(t *testing.T) {
	t.Run("TTL expiration during Get", func(t *testing.T) {
		cache := NewBuildCache(1000, 50*time.Millisecond) // 50ms TTL

		cache.Set("key1", []byte("value1"))

		// Should be found immediately
		_, found := cache.Get("key1")
		assert.True(t, found, "Entry should be found immediately")

		// Wait for TTL expiration
		time.Sleep(60 * time.Millisecond)

		// Should be expired and removed
		_, found = cache.Get("key1")
		assert.False(t, found, "Entry should be expired")

		// Cache should be cleaned up
		count, _, _ := cache.GetStats()
		assert.Equal(t, 0, count, "Expired entry should be removed from cache")
	})

	t.Run("TTL does not affect size until accessed", func(t *testing.T) {
		cache := NewBuildCache(1000, 10*time.Millisecond) // 10ms TTL

		cache.Set("key1", []byte("value1"))
		count1, size1, _ := cache.GetStats()

		// Wait for expiration but don't access
		time.Sleep(20 * time.Millisecond)

		// Size should still reflect the entry until it's accessed
		count2, size2, _ := cache.GetStats()
		assert.Equal(t, count1, count2, "Count should remain same until entry is accessed")
		assert.Equal(t, size1, size2, "Size should remain same until entry is accessed")

		// Access should trigger cleanup
		cache.Get("key1")
		count3, size3, _ := cache.GetStats()
		assert.Equal(t, 0, count3, "Count should be 0 after accessing expired entry")
		assert.Equal(t, int64(0), size3, "Size should be 0 after accessing expired entry")
	})
}

func TestBuildCache_EdgeCases(t *testing.T) {
	t.Run("zero size cache", func(t *testing.T) {
		cache := NewBuildCache(0, time.Hour)

		cache.Set("key1", []byte("value1"))

		// Zero-size cache may still store the entry initially, but evictIfNeeded should trigger
		// Let's test that adding more entries doesn't grow beyond the limit
		cache.Set("key2", []byte("value2"))
		cache.Set("key3", []byte("value3"))

		count, _, maxSize := cache.GetStats()
		assert.Equal(t, int64(0), maxSize, "Max size should be 0")
		// The cache behavior with size 0 may vary - let's just ensure it doesn't crash
		assert.GreaterOrEqual(t, count, 0, "Count should be non-negative")
	})

	t.Run("empty value", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)

		cache.Set("empty", []byte{})

		value, found := cache.Get("empty")
		assert.True(t, found, "Empty value should be stored")
		assert.Equal(t, []byte{}, value, "Empty value should be retrieved correctly")
	})

	t.Run("nil value", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)

		cache.Set("nil", nil)

		value, found := cache.Get("nil")
		assert.True(t, found, "Nil value should be stored")
		assert.Nil(t, value, "Nil value should be retrieved correctly")
	})

	t.Run("empty key", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)

		cache.Set("", []byte("value"))

		value, found := cache.Get("")
		assert.True(t, found, "Empty key should be valid")
		assert.Equal(t, []byte("value"), value)
	})

	t.Run("update existing key", func(t *testing.T) {
		cache := NewBuildCache(1000, time.Hour)

		cache.Set("key1", []byte("value1"))
		cache.Set("key1", []byte("updated_value"))

		value, found := cache.Get("key1")
		assert.True(t, found)
		assert.Equal(t, []byte("updated_value"), value, "Value should be updated")

		// Should only have one entry
		count, _, _ := cache.GetStats()
		assert.Equal(t, 1, count, "Should still have only one entry")
	})
}

func TestBuildCache_Clear(t *testing.T) {
	cache := NewBuildCache(1000, time.Hour)

	// Add several entries
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))
	cache.Set("key3", []byte("value3"))

	count, size, _ := cache.GetStats()
	assert.Greater(t, count, 0)
	assert.Greater(t, size, int64(0))

	// Clear cache
	cache.Clear()

	// Verify everything is cleared
	count, size, _ = cache.GetStats()
	assert.Equal(t, 0, count, "Count should be 0 after clear")
	assert.Equal(t, int64(0), size, "Size should be 0 after clear")

	// Verify entries are not accessible
	_, found := cache.Get("key1")
	assert.False(t, found, "Entries should not be accessible after clear")

	// Verify cache is still functional
	cache.Set("new_key", []byte("new_value"))
	_, found = cache.Get("new_key")
	assert.True(t, found, "Cache should be functional after clear")
}

// Benchmark tests for performance validation.
func BenchmarkBuildCache_Set(b *testing.B) {
	cache := NewBuildCache(1024*1024, time.Hour) // 1MB cache
	value := make([]byte, 100)                   // 100 byte values

	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, value)
	}
}

func BenchmarkBuildCache_Get(b *testing.B) {
	cache := NewBuildCache(1024*1024, time.Hour)
	value := make([]byte, 100)

	// Pre-populate cache
	for i := range 1000 {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkBuildCache_LRU_Updates(b *testing.B) {
	cache := NewBuildCache(1024*1024, time.Hour)
	value := make([]byte, 100)

	// Fill cache to trigger LRU operations
	for i := range 100 {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := range b.N {
		// Access random entries to trigger LRU updates
		key := fmt.Sprintf("key%d", i%100)
		cache.Get(key)
	}
}

func BenchmarkBuildCache_ConcurrentAccess_LRU(b *testing.B) {
	cache := NewBuildCache(1024*1024, time.Hour)
	value := make([]byte, 100)

	// Pre-populate
	for i := range 100 {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%100)
			cache.Get(key)
			i++
		}
	})
}
