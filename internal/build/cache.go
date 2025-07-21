// Package build provides build cache functionality with LRU eviction and TTL support.
package build

import (
	"sync"
	"time"
)

// BuildCache caches build results with LRU eviction and TTL
type BuildCache struct {
	entries     map[string]*CacheEntry
	mutex       sync.RWMutex
	maxSize     int64
	currentSize int64 // Track current size for O(1) access
	ttl         time.Duration
	// LRU implementation
	head *CacheEntry
	tail *CacheEntry
}

// CacheEntry represents a cached build result
type CacheEntry struct {
	Key        string
	Value      []byte
	Hash       string
	CreatedAt  time.Time
	AccessedAt time.Time
	Size       int64
	// LRU doubly-linked list pointers
	prev *CacheEntry
	next *CacheEntry
}

// NewBuildCache creates a new build cache
func NewBuildCache(maxSize int64, ttl time.Duration) *BuildCache {
	cache := &BuildCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Initialize LRU doubly-linked list with dummy head and tail
	cache.head = &CacheEntry{}
	cache.tail = &CacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// Get retrieves a value from the cache
func (bc *BuildCache) Get(key string) ([]byte, bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	entry, exists := bc.entries[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		return nil, false
	}

	// Move to front (mark as recently used)
	bc.moveToFront(entry)
	entry.AccessedAt = time.Now()
	return entry.Value, true
}

// Set stores a value in the cache
func (bc *BuildCache) Set(key string, value []byte) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Check if entry already exists
	if existingEntry, exists := bc.entries[key]; exists {
		// Update existing entry - adjust current size
		sizeDiff := int64(len(value)) - existingEntry.Size
		existingEntry.Value = value
		existingEntry.AccessedAt = time.Now()
		existingEntry.Size = int64(len(value))
		bc.currentSize += sizeDiff
		bc.moveToFront(existingEntry)
		return
	}

	// Check if we need to evict old entries
	bc.evictIfNeeded(int64(len(value)))

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Hash:       key,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Size:       int64(len(value)),
	}

	bc.entries[key] = entry
	bc.currentSize += entry.Size
	bc.addToFront(entry)
}

// evictIfNeeded evicts entries if cache would exceed max size
func (bc *BuildCache) evictIfNeeded(newSize int64) {
	if bc.currentSize+newSize <= bc.maxSize {
		return
	}

	// Efficient LRU eviction - remove from tail (least recently used)
	for bc.currentSize+newSize > bc.maxSize && bc.tail.prev != bc.head {
		// Remove the least recently used entry (tail.prev)
		lru := bc.tail.prev
		bc.removeFromList(lru)
		delete(bc.entries, lru.Key)
		bc.currentSize -= lru.Size
	}
}

// getCurrentSize returns the current cache size
func (bc *BuildCache) getCurrentSize() int64 {
	return bc.currentSize
}

// Clear clears all cache entries
func (bc *BuildCache) Clear() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	bc.entries = make(map[string]*CacheEntry)
	bc.currentSize = 0
	// Reset LRU list
	bc.head.next = bc.tail
	bc.tail.prev = bc.head
}

// GetStats returns cache statistics
func (bc *BuildCache) GetStats() (int, int64, int64) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	count := len(bc.entries)
	size := bc.getCurrentSize()
	maxSize := bc.maxSize

	return count, size, maxSize
}

// LRU doubly-linked list operations
func (bc *BuildCache) addToFront(entry *CacheEntry) {
	entry.prev = bc.head
	entry.next = bc.head.next
	bc.head.next.prev = entry
	bc.head.next = entry
}

func (bc *BuildCache) removeFromList(entry *CacheEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
}

func (bc *BuildCache) moveToFront(entry *CacheEntry) {
	bc.removeFromList(entry)
	bc.addToFront(entry)
}
