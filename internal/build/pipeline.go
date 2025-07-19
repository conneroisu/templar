package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/registry"
)

// BuildPipeline manages the build process for templ components
type BuildPipeline struct {
	compiler    *TemplCompiler
	cache       *BuildCache
	queue       *BuildQueue
	workers     int
	registry    *registry.ComponentRegistry
	errorParser *errors.ErrorParser
	metrics     *BuildMetrics
	callbacks   []BuildCallback
	workerWg    sync.WaitGroup
	resultWg    sync.WaitGroup
	cancel      context.CancelFunc
	// Object pools for memory optimization
	objectPools *ObjectPools
	slicePools  *SlicePools
	workerPool  *WorkerPool
}

// BuildTask represents a build task
type BuildTask struct {
	Component *registry.ComponentInfo
	Priority  int
	Timestamp time.Time
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Component    *registry.ComponentInfo
	Output       []byte
	Error        error
	ParsedErrors []*errors.ParsedError
	Duration     time.Duration
	CacheHit     bool
	Hash         string
}

// BuildCallback is called when a build completes
type BuildCallback func(result BuildResult)

// BuildMetrics tracks build performance
type BuildMetrics struct {
	TotalBuilds      int64
	SuccessfulBuilds int64
	FailedBuilds     int64
	CacheHits        int64
	AverageDuration  time.Duration
	TotalDuration    time.Duration
	mutex            sync.RWMutex
}

// TemplCompiler handles templ compilation
type TemplCompiler struct {
	command string
	args    []string
}

// BuildCache caches build results
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

// BuildQueue manages build tasks
type BuildQueue struct {
	tasks    chan BuildTask
	results  chan BuildResult
	priority chan BuildTask
}

// NewBuildPipeline creates a new build pipeline
func NewBuildPipeline(workers int, registry *registry.ComponentRegistry) *BuildPipeline {
	compiler := &TemplCompiler{
		command: "templ",
		args:    []string{"generate"},
	}

	cache := &BuildCache{
		entries: make(map[string]*CacheEntry),
		maxSize: 100 * 1024 * 1024, // 100MB
		ttl:     time.Hour,
	}

	// Initialize LRU doubly-linked list with dummy head and tail
	cache.head = &CacheEntry{}
	cache.tail = &CacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	queue := &BuildQueue{
		tasks:    make(chan BuildTask, 100),
		results:  make(chan BuildResult, 100),
		priority: make(chan BuildTask, 10),
	}

	metrics := &BuildMetrics{}

	return &BuildPipeline{
		compiler:    compiler,
		cache:       cache,
		queue:       queue,
		workers:     workers,
		registry:    registry,
		errorParser: errors.NewErrorParser(),
		metrics:     metrics,
		callbacks:   make([]BuildCallback, 0),
		// Initialize object pools for memory optimization
		objectPools: NewObjectPools(),
		slicePools:  NewSlicePools(),
		workerPool:  NewWorkerPool(),
	}
}

// Start starts the build pipeline
func (bp *BuildPipeline) Start(ctx context.Context) {
	// Create cancellable context
	ctx, bp.cancel = context.WithCancel(ctx)

	// Start workers
	for i := 0; i < bp.workers; i++ {
		bp.workerWg.Add(1)
		go bp.worker(ctx)
	}

	// Start result processor
	bp.resultWg.Add(1)
	go bp.processResults(ctx)
}

// Stop stops the build pipeline and waits for all goroutines to finish
func (bp *BuildPipeline) Stop() {
	if bp.cancel != nil {
		bp.cancel()
	}

	// Wait for all workers to finish
	bp.workerWg.Wait()

	// Wait for result processor to finish
	bp.resultWg.Wait()
}

// Build queues a component for building
func (bp *BuildPipeline) Build(component *registry.ComponentInfo) {
	task := BuildTask{
		Component: component,
		Priority:  1,
		Timestamp: time.Now(),
	}

	select {
	case bp.queue.tasks <- task:
	default:
		// Queue full, skip
	}
}

// BuildWithPriority queues a component for building with high priority
func (bp *BuildPipeline) BuildWithPriority(component *registry.ComponentInfo) {
	task := BuildTask{
		Component: component,
		Priority:  10,
		Timestamp: time.Now(),
	}

	select {
	case bp.queue.priority <- task:
	default:
		// Queue full, skip
	}
}

// AddCallback adds a callback to be called when builds complete
func (bp *BuildPipeline) AddCallback(callback BuildCallback) {
	bp.callbacks = append(bp.callbacks, callback)
}

// GetMetrics returns the current build metrics
func (bp *BuildPipeline) GetMetrics() BuildMetrics {
	bp.metrics.mutex.RLock()
	defer bp.metrics.mutex.RUnlock()
	return *bp.metrics
}

// ClearCache clears the build cache
func (bp *BuildPipeline) ClearCache() {
	bp.cache.Clear()
}

// GetCacheStats returns cache statistics
func (bp *BuildPipeline) GetCacheStats() (int, int64, int64) {
	return bp.cache.GetStats()
}

// worker processes build tasks
func (bp *BuildPipeline) worker(ctx context.Context) {
	defer bp.workerWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-bp.queue.priority:
			bp.processBuildTask(task)
		case task := <-bp.queue.tasks:
			bp.processBuildTask(task)
		}
	}
}

func (bp *BuildPipeline) processBuildTask(task BuildTask) {
	start := time.Now()

	// Generate content hash for caching
	contentHash := bp.generateContentHash(task.Component)

	// Check cache first
	if result, found := bp.cache.Get(contentHash); found {
		// Use object pool for cache hit result
		buildResult := bp.objectPools.GetBuildResult()
		buildResult.Component = task.Component
		buildResult.Output = result
		buildResult.Error = nil
		buildResult.ParsedErrors = nil
		buildResult.Duration = time.Since(start)
		buildResult.CacheHit = true
		buildResult.Hash = contentHash
		
		bp.queue.results <- *buildResult
		bp.objectPools.PutBuildResult(buildResult)
		return
	}

	// Execute build with pooled output buffer
	output, err := bp.compiler.CompileWithPools(task.Component, bp.objectPools)

	// Parse errors if build failed
	var parsedErrors []*errors.ParsedError
	if err != nil {
		parsedErrors = bp.errorParser.ParseError(string(output))
	}

	// Use object pool for build result
	buildResult := bp.objectPools.GetBuildResult()
	buildResult.Component = task.Component
	buildResult.Output = output
	buildResult.Error = err
	buildResult.ParsedErrors = parsedErrors
	buildResult.Duration = time.Since(start)
	buildResult.CacheHit = false
	buildResult.Hash = contentHash

	// Cache successful builds
	if err == nil {
		bp.cache.Set(contentHash, output)
	}

	bp.queue.results <- *buildResult
	bp.objectPools.PutBuildResult(buildResult)
}

func (bp *BuildPipeline) processResults(ctx context.Context) {
	defer bp.resultWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case result := <-bp.queue.results:
			bp.handleBuildResult(result)
		}
	}
}

func (bp *BuildPipeline) handleBuildResult(result BuildResult) {
	// Update metrics
	bp.updateMetrics(result)

	// Print result
	if result.Error != nil {
		fmt.Printf("Build failed for %s: %v\n", result.Component.Name, result.Error)
		if len(result.ParsedErrors) > 0 {
			fmt.Println("Parsed errors:")
			for _, err := range result.ParsedErrors {
				fmt.Print(err.FormatError())
			}
		}
	} else {
		status := "succeeded"
		if result.CacheHit {
			status = "cached"
		}
		fmt.Printf("Build %s for %s in %v\n", status, result.Component.Name, result.Duration)
	}

	// Call callbacks
	for _, callback := range bp.callbacks {
		callback(result)
	}
}

// TemplCompiler methods
func (tc *TemplCompiler) Compile(component *registry.ComponentInfo) ([]byte, error) {
	// Validate command and arguments to prevent command injection
	if err := tc.validateCommand(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Run templ generate command
	cmd := exec.Command(tc.command, tc.args...)
	cmd.Dir = "." // Run in current directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("templ generate failed: %w\nOutput: %s", err, output)
	}

	return output, nil
}

// CompileWithPools performs compilation using object pools for memory efficiency
func (tc *TemplCompiler) CompileWithPools(component *registry.ComponentInfo, pools *ObjectPools) ([]byte, error) {
	// Validate command and arguments to prevent command injection
	if err := tc.validateCommand(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Get pooled buffer for output
	outputBuffer := pools.GetOutputBuffer()
	defer pools.PutOutputBuffer(outputBuffer)

	// Run templ generate command
	cmd := exec.Command(tc.command, tc.args...)
	cmd.Dir = "." // Run in current directory

	// Use pooled buffers for command output
	var stdout, stderr []byte
	var err error
	
	if output, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		// Copy output to our buffer to avoid keeping the original allocation
		outputBuffer = append(outputBuffer, output...)
		err = fmt.Errorf("templ generate failed: %w\nOutput: %s", cmdErr, outputBuffer)
		return nil, err
	} else {
		// Copy successful output to our buffer
		outputBuffer = append(outputBuffer, output...)
	}

	// Return a copy of the buffer content (caller owns this memory)
	result := make([]byte, len(outputBuffer))
	copy(result, outputBuffer)
	return result, nil
}

// validateCommand validates the command and arguments to prevent command injection
func (tc *TemplCompiler) validateCommand() error {
	// Allowlist of permitted commands
	allowedCommands := map[string]bool{
		"templ": true,
		"go":    true,
	}

	// Check if command is in allowlist
	if !allowedCommands[tc.command] {
		return fmt.Errorf("command '%s' is not allowed", tc.command)
	}

	// Validate arguments - prevent shell metacharacters and path traversal
	for _, arg := range tc.args {
		if err := validateArgument(arg); err != nil {
			return fmt.Errorf("invalid argument '%s': %w", arg, err)
		}
	}

	return nil
}

// validateArgument validates a single command argument
func validateArgument(arg string) error {
	// Check for shell metacharacters that could be used for command injection
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\\", "\"", "'"}
	for _, char := range dangerous {
		if strings.Contains(arg, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(arg, "..") {
		return fmt.Errorf("contains path traversal: %s", arg)
	}

	// Check for absolute paths (prefer relative paths for security)
	if filepath.IsAbs(arg) && !strings.HasPrefix(arg, "/usr/bin/") && !strings.HasPrefix(arg, "/bin/") {
		return fmt.Errorf("absolute path not allowed: %s", arg)
	}

	return nil
}

// BuildCache methods
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

// generateContentHash generates a hash for component content with metadata-based optimization
func (bp *BuildPipeline) generateContentHash(component *registry.ComponentInfo) string {
	// Get file metadata first for fast comparison
	stat, err := os.Stat(component.FilePath)
	if err != nil {
		return component.FilePath
	}

	// Create metadata-based hash key for cache lookup
	metadataKey := fmt.Sprintf("%s:%d:%d", component.FilePath, stat.ModTime().Unix(), stat.Size())

	// Check if we have a cached hash for this metadata
	bp.cache.mutex.RLock()
	if entry, exists := bp.cache.entries[metadataKey]; exists {
		// Update access time and return cached hash
		entry.AccessedAt = time.Now()
		bp.cache.moveToFront(entry)
		bp.cache.mutex.RUnlock()
		return entry.Hash
	}
	bp.cache.mutex.RUnlock()

	// Only read and hash file content if metadata changed
	content, err := os.ReadFile(component.FilePath)
	if err != nil {
		// Fallback to metadata-based hash
		return fmt.Sprintf("%s:%d", component.FilePath, stat.ModTime().Unix())
	}

	// Generate content hash
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])

	// Cache the hash with metadata key for future lookups
	bp.cache.mutex.Lock()
	entry := &CacheEntry{
		Key:        metadataKey,
		Value:      nil, // Only cache the hash, not the content
		Hash:       contentHash,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Size:       int64(len(metadataKey) + len(contentHash)), // Minimal size for hash cache
	}

	// Add to cache if within size limits
	if bp.cache.currentSize+entry.Size <= bp.cache.maxSize {
		bp.cache.entries[metadataKey] = entry
		bp.cache.addToFront(entry)
		bp.cache.currentSize += entry.Size
	}
	bp.cache.mutex.Unlock()

	return contentHash
}

// updateMetrics updates build metrics
func (bp *BuildPipeline) updateMetrics(result BuildResult) {
	bp.metrics.mutex.Lock()
	defer bp.metrics.mutex.Unlock()

	bp.metrics.TotalBuilds++
	bp.metrics.TotalDuration += result.Duration

	if result.CacheHit {
		bp.metrics.CacheHits++
	}

	if result.Error != nil {
		bp.metrics.FailedBuilds++
	} else {
		bp.metrics.SuccessfulBuilds++
	}

	// Update average duration
	if bp.metrics.TotalBuilds > 0 {
		bp.metrics.AverageDuration = bp.metrics.TotalDuration / time.Duration(bp.metrics.TotalBuilds)
	}
}
