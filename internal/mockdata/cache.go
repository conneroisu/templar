// Package mockdata provides high-performance caching for generated mock data.
//
// This package implements multiple caching strategies optimized for different use cases:
// - MemoryMockDataCache: Simple in-memory cache with TTL expiration
// - LRUMockDataCache: Least Recently Used eviction with bounded memory usage
// - TTLMockDataCache: Time-based expiration with configurable default TTL
//
// Design philosophy: All caches are thread-safe and designed for high-concurrency
// access patterns typical in mock data generation workloads.
package mockdata

import (
	"sync"
	"time"
)

// cacheEntry represents a cached mock data entry with expiration.
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// MemoryMockDataCache provides in-memory caching for mock data with TTL expiration.
//
// Design decisions:
// - Uses RWMutex for read-heavy workloads (multiple generators reading cached values)
// - TTL-based expiration prevents stale data while allowing configurable cache lifetime
// - Background cleanup is disabled by default to avoid goroutine leaks in tests
// - Channel-based cleanup coordination allows graceful shutdown
type MemoryMockDataCache struct {
	data    map[string]*cacheEntry // Thread-safe map of cached values with expiration
	mutex   sync.RWMutex           // Read-write mutex optimized for read-heavy access
	cleanup chan struct{}          // Channel to trigger manual cleanup
	done    chan struct{}          // Channel to signal shutdown
}

// NewMemoryMockDataCache creates a new memory-based cache.
func NewMemoryMockDataCache() MockDataCache {
	cache := &MemoryMockDataCache{
		data:    make(map[string]*cacheEntry),
		cleanup: make(chan struct{}),
		done:    make(chan struct{}),
	}

	// Start background cleanup goroutine (disabled for tests)
	// go cache.cleanupExpired()

	return cache
}

// Get retrieves cached mock data with automatic expiration checking.
//
// Performance optimization: Uses read lock only and defers deletion to background
// cleanup to avoid lock escalation. This maintains high read throughput even when
// expired entries are present.
func (c *MemoryMockDataCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Check if expired - avoids lock escalation by not deleting immediately
	if time.Now().After(entry.expiresAt) {
		// Don't delete here to avoid write lock, let cleanup handle it
		return nil, false
	}

	return entry.value, true
}

// Set stores mock data in cache with TTL.
func (c *MemoryMockDataCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiresAt := time.Now().Add(ttl)
	c.data[key] = &cacheEntry{
		value:     value,
		expiresAt: expiresAt,
	}
}

// Clear removes all cached data.
func (c *MemoryMockDataCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*cacheEntry)
}

// Size returns current cache size.
func (c *MemoryMockDataCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.data)
}

// cleanupExpired removes expired entries periodically.
func (c *MemoryMockDataCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.cleanup:
			c.removeExpired()
		case <-c.done:
			return
		}
	}
}

// removeExpired removes expired entries from the cache.
func (c *MemoryMockDataCache) removeExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.expiresAt) {
			delete(c.data, key)
		}
	}
}

// Close stops the cleanup goroutine.
func (c *MemoryMockDataCache) Close() {
	close(c.done)
}

// LRUMockDataCache provides an LRU (Least Recently Used) cache for mock data.
//
// Design rationale: Implements classic LRU with doubly-linked list for O(1) operations.
// This prevents unbounded memory growth by evicting least recently used entries when
// the cache reaches its maximum size. Particularly useful for long-running generators
// that process many different components.
//
// Implementation notes:
// - Uses dummy head/tail nodes to simplify edge cases in list manipulation
// - Combines hash map (O(1) lookup) with doubly-linked list (O(1) reordering)
// - All operations are atomic to maintain consistency under concurrent access
type LRUMockDataCache struct {
	maxSize int                  // Maximum number of entries before eviction
	data    map[string]*lruEntry // Hash map for O(1) key lookup
	head    *lruEntry            // Dummy head node for list operations
	tail    *lruEntry            // Dummy tail node for list operations
	mutex   sync.RWMutex         // Protects both map and linked list
	cleanup chan struct{}        // Manual cleanup trigger
	done    chan struct{}        // Shutdown signal
}

// lruEntry represents an entry in the LRU cache with doubly-linked list pointers.
type lruEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
	prev      *lruEntry
	next      *lruEntry
}

// NewLRUMockDataCache creates a new LRU cache with the specified maximum size.
func NewLRUMockDataCache(maxSize int) MockDataCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}

	cache := &LRUMockDataCache{
		maxSize: maxSize,
		data:    make(map[string]*lruEntry),
		cleanup: make(chan struct{}),
		done:    make(chan struct{}),
	}

	// Initialize dummy head and tail nodes
	cache.head = &lruEntry{}
	cache.tail = &lruEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	// Start background cleanup goroutine (disabled for tests)
	// go cache.cleanupExpired()

	return cache
}

// Get retrieves cached mock data and moves it to the front (most recently used).
//
// LRU mechanics: Every access updates the entry's position in the usage order,
// ensuring frequently accessed items stay in cache longer. Uses full write lock
// because list manipulation requires exclusive access.
func (c *LRUMockDataCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock() // Write lock required for list manipulation
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Check if expired - immediately remove to free space
	if time.Now().After(entry.expiresAt) {
		c.removeEntry(entry)
		return nil, false
	}

	// Move to front (most recently used) - updates LRU order
	c.moveToFront(entry)

	return entry.value, true
}

// Set stores mock data in cache with TTL, evicting LRU entries if necessary.
func (c *LRUMockDataCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if key already exists
	if existing, exists := c.data[key]; exists {
		// Update existing entry
		existing.value = value
		existing.expiresAt = time.Now().Add(ttl)
		c.moveToFront(existing)
		return
	}

	// Create new entry
	entry := &lruEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	// Add to cache
	c.data[key] = entry
	c.addToFront(entry)

	// Check if we need to evict LRU entry
	if len(c.data) > c.maxSize {
		c.evictLRU()
	}
}

// Clear removes all cached data.
func (c *LRUMockDataCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*lruEntry)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Size returns current cache size.
func (c *LRUMockDataCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.data)
}

// moveToFront moves an entry to the front of the LRU list.
func (c *LRUMockDataCache) moveToFront(entry *lruEntry) {
	c.removeFromList(entry)
	c.addToFront(entry)
}

// addToFront adds an entry to the front of the LRU list.
func (c *LRUMockDataCache) addToFront(entry *lruEntry) {
	entry.prev = c.head
	entry.next = c.head.next
	c.head.next.prev = entry
	c.head.next = entry
}

// removeFromList removes an entry from the doubly-linked list.
func (c *LRUMockDataCache) removeFromList(entry *lruEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
}

// removeEntry removes an entry from both the map and the list.
func (c *LRUMockDataCache) removeEntry(entry *lruEntry) {
	delete(c.data, entry.key)
	c.removeFromList(entry)
}

// evictLRU removes the least recently used entry.
func (c *LRUMockDataCache) evictLRU() {
	if c.tail.prev != c.head {
		lru := c.tail.prev
		c.removeEntry(lru)
	}
}

// cleanupExpired removes expired entries periodically.
func (c *LRUMockDataCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpiredLRU()
		case <-c.cleanup:
			c.removeExpiredLRU()
		case <-c.done:
			return
		}
	}
}

// removeExpiredLRU removes expired entries from the LRU cache.
func (c *LRUMockDataCache) removeExpiredLRU() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Collect expired entries to avoid modifying map while iterating
	var expiredEntries []*lruEntry
	for _, entry := range c.data {
		if now.After(entry.expiresAt) {
			expiredEntries = append(expiredEntries, entry)
		}
	}

	// Remove expired entries
	for _, entry := range expiredEntries {
		c.removeEntry(entry)
	}
}

// Close stops the cleanup goroutine.
func (c *LRUMockDataCache) Close() {
	close(c.done)
}

// TTLMockDataCache provides a simple TTL-based cache without LRU eviction.
//
// Design rationale: Extends MemoryMockDataCache with automatic TTL application.
// Useful when you want consistent expiration times without the complexity of LRU.
// The embedded struct pattern allows code reuse while adding TTL convenience.
type TTLMockDataCache struct {
	MemoryMockDataCache               // Embedded for method inheritance
	defaultTTL          time.Duration // Applied when TTL is not specified
}

// NewTTLMockDataCache creates a new TTL-based cache with a default TTL.
func NewTTLMockDataCache(defaultTTL time.Duration) MockDataCache {
	return &TTLMockDataCache{
		MemoryMockDataCache: MemoryMockDataCache{
			data:    make(map[string]*cacheEntry),
			cleanup: make(chan struct{}),
			done:    make(chan struct{}),
		},
		defaultTTL: defaultTTL,
	}
}

// Set stores mock data with the default TTL if none specified.
func (c *TTLMockDataCache) Set(key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	c.MemoryMockDataCache.Set(key, value, ttl)
}
