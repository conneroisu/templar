// Package build provides build cache functionality with LRU eviction and TTL support.
package build

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/types"
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
	// Statistics tracking (atomic for thread safety)
	hits       int64
	misses     int64
	sets       int64
	deletes    int64
	evictions  int64
}

// Ensure BuildCache implements the interfaces.CacheStats interface
var _ interfaces.CacheStats = (*BuildCache)(nil)

// CacheEntry represents a cached build result
type CacheEntry struct {
	Key        string
	Value      []byte
	Hash       string
	CreatedAt  time.Time
	AccessedAt time.Time
	Size       int64
	// AST caching support
	ASTData     []byte // Cached AST parsing results
	Metadata    *CacheMetadata
	// LRU doubly-linked list pointers
	prev *CacheEntry
	next *CacheEntry
}

// CacheMetadata stores additional metadata for cache entries
type CacheMetadata struct {
	ComponentInfo *types.ComponentInfo `json:"component_info,omitempty"`
	ParseTime     time.Duration        `json:"parse_time"`
	CacheType     string               `json:"cache_type"` // "build", "ast", "hash"
	Version       string               `json:"version"`
}

// ASTParseResult represents cached AST parsing results
type ASTParseResult struct {
	Component    *types.ComponentInfo
	Parameters   []types.ParameterInfo
	Dependencies []string
	ParseTime    time.Duration
	CachedAt     time.Time
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
		atomic.AddInt64(&bc.misses, 1)
		return nil, false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		atomic.AddInt64(&bc.misses, 1)
		return nil, false
	}

	// Move to front (mark as recently used)
	bc.moveToFront(entry)
	entry.AccessedAt = time.Now()
	atomic.AddInt64(&bc.hits, 1)
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
		atomic.AddInt64(&bc.sets, 1)
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
		Metadata: &CacheMetadata{
			CacheType: "build",
			Version:   "1.0",
		},
	}

	bc.entries[key] = entry
	bc.currentSize += entry.Size
	bc.addToFront(entry)
	atomic.AddInt64(&bc.sets, 1)
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
		atomic.AddInt64(&bc.evictions, 1)
	}
}

// getCurrentSize returns the current cache size
func (bc *BuildCache) getCurrentSize() int64 {
	return bc.currentSize
}

// Clear clears all cache entries and resets statistics
func (bc *BuildCache) Clear() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Clear all entries
	bc.entries = make(map[string]*CacheEntry)
	bc.currentSize = 0
	
	// Reset LRU list
	bc.head.next = bc.tail
	bc.tail.prev = bc.head
	
	// Reset statistics
	atomic.StoreInt64(&bc.hits, 0)
	atomic.StoreInt64(&bc.misses, 0)
	atomic.StoreInt64(&bc.sets, 0)
	atomic.StoreInt64(&bc.deletes, 0)
	atomic.StoreInt64(&bc.evictions, 0)
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

// GetHash retrieves a cached hash for a metadata key
// This method is thread-safe and properly handles LRU updates
func (bc *BuildCache) GetHash(key string) (string, bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	entry, exists := bc.entries[key]
	if !exists {
		return "", false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		return "", false
	}

	// Move to front (mark as recently used) and update access time
	bc.moveToFront(entry)
	entry.AccessedAt = time.Now()
	return entry.Hash, true
}

// SetHash stores a hash in the cache with a metadata key
// This method is thread-safe and handles eviction properly
func (bc *BuildCache) SetHash(key string, hash string) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Calculate entry size (key + hash + minimal overhead)
	entrySize := int64(len(key) + len(hash))

	// Check if entry already exists
	if existingEntry, exists := bc.entries[key]; exists {
		// Update existing entry - adjust current size
		sizeDiff := entrySize - existingEntry.Size
		existingEntry.Hash = hash
		existingEntry.AccessedAt = time.Now()
		existingEntry.Size = entrySize
		bc.currentSize += sizeDiff
		bc.moveToFront(existingEntry)
		return
	}

	// Check if we need to evict old entries before adding new one
	bc.evictIfNeeded(entrySize)

	// Create new entry for hash storage
	entry := &CacheEntry{
		Key:        key,
		Value:      nil, // Only cache the hash, not the content
		Hash:       hash,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Size:       entrySize,
	}

	// Add to cache with proper size tracking
	bc.entries[key] = entry
	bc.addToFront(entry)
	bc.currentSize += entry.Size
}

// CacheStats interface implementation

// GetSize returns the current cache size in bytes
func (bc *BuildCache) GetSize() int64 {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	return bc.currentSize
}

// GetHits returns the number of cache hits
func (bc *BuildCache) GetHits() int64 {
	return atomic.LoadInt64(&bc.hits)
}

// GetMisses returns the number of cache misses
func (bc *BuildCache) GetMisses() int64 {
	return atomic.LoadInt64(&bc.misses)
}

// GetHitRate returns the cache hit rate as a percentage (0.0 to 1.0)
func (bc *BuildCache) GetHitRate() float64 {
	hits := atomic.LoadInt64(&bc.hits)
	misses := atomic.LoadInt64(&bc.misses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// GetEvictions returns the number of cache evictions
func (bc *BuildCache) GetEvictions() int64 {
	return atomic.LoadInt64(&bc.evictions)
}

// SetAST stores AST parsing results in the cache
func (bc *BuildCache) SetAST(key string, astResult *ASTParseResult) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Serialize AST result to JSON
	astData, err := json.Marshal(astResult)
	if err != nil {
		return err
	}

	entrySize := int64(len(key) + len(astData))
	bc.evictIfNeeded(entrySize)

	entry := &CacheEntry{
		Key:        key,
		Value:      nil, // AST data stored separately
		ASTData:    astData,
		Hash:       key,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Size:       entrySize,
		Metadata: &CacheMetadata{
			ComponentInfo: astResult.Component,
			ParseTime:     astResult.ParseTime,
			CacheType:     "ast",
			Version:       "1.0",
		},
	}

	bc.entries[key] = entry
	bc.addToFront(entry)
	bc.currentSize += entry.Size
	atomic.AddInt64(&bc.sets, 1)

	return nil
}

// GetAST retrieves AST parsing results from the cache
func (bc *BuildCache) GetAST(key string) (*ASTParseResult, bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	entry, exists := bc.entries[key]
	if !exists || entry.ASTData == nil {
		atomic.AddInt64(&bc.misses, 1)
		return nil, false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		atomic.AddInt64(&bc.misses, 1)
		return nil, false
	}

	// Deserialize AST result
	var astResult ASTParseResult
	if err := json.Unmarshal(entry.ASTData, &astResult); err != nil {
		atomic.AddInt64(&bc.misses, 1)
		return nil, false
	}

	// Move to front and update access time
	bc.moveToFront(entry)
	entry.AccessedAt = time.Now()
	atomic.AddInt64(&bc.hits, 1)

	return &astResult, true
}

// SetWithMetadata stores a value with additional metadata
func (bc *BuildCache) SetWithMetadata(key string, value []byte, metadata *CacheMetadata) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Check if entry already exists
	if existingEntry, exists := bc.entries[key]; exists {
		// Update existing entry
		sizeDiff := int64(len(value)) - existingEntry.Size
		existingEntry.Value = value
		existingEntry.Metadata = metadata
		existingEntry.AccessedAt = time.Now()
		existingEntry.Size = int64(len(value))
		bc.currentSize += sizeDiff
		bc.moveToFront(existingEntry)
		return
	}

	entrySize := int64(len(value))
	bc.evictIfNeeded(entrySize)

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Hash:       key,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Size:       entrySize,
		Metadata:   metadata,
	}

	bc.entries[key] = entry
	bc.currentSize += entry.Size
	bc.addToFront(entry)
	atomic.AddInt64(&bc.sets, 1)
}

// GetWithMetadata retrieves a value with its metadata
func (bc *BuildCache) GetWithMetadata(key string) ([]byte, *CacheMetadata, bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	entry, exists := bc.entries[key]
	if !exists {
		atomic.AddInt64(&bc.misses, 1)
		return nil, nil, false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		atomic.AddInt64(&bc.misses, 1)
		return nil, nil, false
	}

	// Move to front and update access time
	bc.moveToFront(entry)
	entry.AccessedAt = time.Now()
	atomic.AddInt64(&bc.hits, 1)

	return entry.Value, entry.Metadata, true
}

// InvalidateByPattern removes cache entries matching a pattern
func (bc *BuildCache) InvalidateByPattern(pattern string) int {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	invalidated := 0
	for key, entry := range bc.entries {
		// Simple pattern matching - could be enhanced with regex
		if len(pattern) > 0 && key != pattern {
			continue
		}

		bc.removeFromList(entry)
		delete(bc.entries, key)
		bc.currentSize -= entry.Size
		invalidated++
	}

	atomic.AddInt64(&bc.deletes, int64(invalidated))
	return invalidated
}

// GetCacheTypeStats returns statistics by cache type
func (bc *BuildCache) GetCacheTypeStats() map[string]int {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	stats := make(map[string]int)
	for _, entry := range bc.entries {
		if entry.Metadata != nil {
			stats[entry.Metadata.CacheType]++
		} else {
			stats["unknown"]++
		}
	}

	return stats
}
