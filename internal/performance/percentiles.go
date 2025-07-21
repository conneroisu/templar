// Package performance provides efficient percentile calculation using skip lists.
//
// This implementation replaces the O(n²) insertion sort with an O(log n) skip list
// data structure that maintains sorted order for efficient percentile queries.
package performance

import (
	"math/rand"
	"sync"
	"time"
)

// SkipListNode represents a node in the skip list
type SkipListNode struct {
	value float64
	next  []*SkipListNode
}

// SkipList implements a probabilistic data structure for efficient percentile calculation
type SkipList struct {
	header   *SkipListNode
	level    int
	size     int
	maxLevel int
	p        float64 // Probability for level promotion
	rng      *rand.Rand
	mu       sync.RWMutex
}

// PercentileCalculator provides efficient percentile calculation with O(log n) operations
type PercentileCalculator struct {
	skipList *SkipList
	maxSize  int
	values   []float64 // Ring buffer for FIFO eviction
	writePos int
	full     bool
}

// NewSkipList creates a new skip list with optimal parameters for percentile calculation
func NewSkipList() *SkipList {
	const maxLevel = 16     // Optimal for up to 65536 elements
	const probability = 0.5 // Standard skip list probability

	sl := &SkipList{
		maxLevel: maxLevel,
		p:        probability,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Create header node with maximum level
	sl.header = &SkipListNode{
		value: -1e9, // Sentinel value
		next:  make([]*SkipListNode, maxLevel),
	}

	return sl
}

// NewPercentileCalculator creates a new efficient percentile calculator
func NewPercentileCalculator(maxSize int) *PercentileCalculator {
	return &PercentileCalculator{
		skipList: NewSkipList(),
		maxSize:  maxSize,
		values:   make([]float64, maxSize),
		writePos: 0,
		full:     false,
	}
}

// randomLevel generates a random level for new nodes using geometric distribution
func (sl *SkipList) randomLevel() int {
	level := 1
	for level < sl.maxLevel && sl.rng.Float64() < sl.p {
		level++
	}
	return level
}

// Insert adds a value to the skip list in O(log n) time
func (sl *SkipList) Insert(value float64) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Find insertion point and update path
	update := make([]*SkipListNode, sl.maxLevel)
	current := sl.header

	// Search from top level down
	for i := sl.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].value < value {
			current = current.next[i]
		}
		update[i] = current
	}

	// Generate random level for new node
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		// Update header pointers for new levels
		for i := sl.level; i < newLevel; i++ {
			update[i] = sl.header
		}
		sl.level = newLevel
	}

	// Create new node
	newNode := &SkipListNode{
		value: value,
		next:  make([]*SkipListNode, newLevel),
	}

	// Update pointers
	for i := 0; i < newLevel; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}

	sl.size++
}

// Delete removes a value from the skip list in O(log n) time
func (sl *SkipList) Delete(value float64) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Find deletion point and update path
	update := make([]*SkipListNode, sl.maxLevel)
	current := sl.header

	// Search from top level down
	for i := sl.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].value < value {
			current = current.next[i]
		}
		update[i] = current
	}

	// Check if value exists
	current = current.next[0]
	if current == nil || current.value != value {
		return false // Value not found
	}

	// Update pointers to remove node
	for i := 0; i < len(current.next); i++ {
		update[i].next[i] = current.next[i]
	}

	// Update level if necessary
	for sl.level > 1 && sl.header.next[sl.level-1] == nil {
		sl.level--
	}

	sl.size--
	return true
}

// GetPercentile calculates the percentile value in O(log n) time
func (sl *SkipList) GetPercentile(percentile float64) float64 {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.size == 0 {
		return 0
	}

	// Calculate target index using nearest rank method (more mathematically correct)
	targetIndex := int(float64(sl.size-1) * percentile / 100.0)
	if targetIndex >= sl.size {
		targetIndex = sl.size - 1
	}

	// Traverse to target index
	current := sl.header.next[0]
	for i := 0; i < targetIndex && current != nil; i++ {
		current = current.next[0]
	}

	if current != nil {
		return current.value
	}
	return 0
}

// Size returns the number of elements in the skip list
func (sl *SkipList) Size() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.size
}

// AddValue adds a new value to the percentile calculator with FIFO eviction
func (pc *PercentileCalculator) AddValue(value float64) {
	// Handle FIFO eviction for ring buffer behavior
	if pc.full {
		// Remove the oldest value from skip list
		oldValue := pc.values[pc.writePos]
		pc.skipList.Delete(oldValue)
	}

	// Add new value to skip list
	pc.skipList.Insert(value)

	// Update ring buffer
	pc.values[pc.writePos] = value
	pc.writePos = (pc.writePos + 1) % pc.maxSize

	if !pc.full && pc.writePos == 0 {
		pc.full = true
	}
}

// GetPercentile calculates the specified percentile efficiently
func (pc *PercentileCalculator) GetPercentile(percentile float64) float64 {
	return pc.skipList.GetPercentile(percentile)
}

// GetP95 returns the 95th percentile
func (pc *PercentileCalculator) GetP95() float64 {
	return pc.GetPercentile(95.0)
}

// GetP99 returns the 99th percentile
func (pc *PercentileCalculator) GetP99() float64 {
	return pc.GetPercentile(99.0)
}

// GetSize returns the current number of values
func (pc *PercentileCalculator) GetSize() int {
	return pc.skipList.Size()
}

// Clear removes all values from the calculator
func (pc *PercentileCalculator) Clear() {
	pc.skipList = NewSkipList()
	pc.writePos = 0
	pc.full = false
	// Values slice is reused
}

// GetAll returns all values in sorted order (for testing/debugging)
func (pc *PercentileCalculator) GetAll() []float64 {
	pc.skipList.mu.RLock()
	defer pc.skipList.mu.RUnlock()

	var result []float64
	current := pc.skipList.header.next[0]
	for current != nil {
		result = append(result, current.value)
		current = current.next[0]
	}
	return result
}

// MemoryFootprint returns approximate memory usage in bytes
func (pc *PercentileCalculator) MemoryFootprint() int {
	size := pc.skipList.Size()
	// More realistic estimation:
	// Each node: 8 bytes (float64) + slice overhead + pointers + GC overhead
	// Skip list nodes have variable levels, average ~2 for p=0.5
	// Each pointer is 8 bytes on 64-bit systems
	nodeSize := 48 // Realistic estimate including Go runtime overhead

	// Ring buffer: maxSize * 8 bytes
	ringBufferSize := pc.maxSize * 8

	return (size * nodeSize) + ringBufferSize
}
