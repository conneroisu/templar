package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
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
	TotalBuilds     int64
	SuccessfulBuilds int64
	FailedBuilds    int64
	CacheHits       int64
	AverageDuration time.Duration
	TotalDuration   time.Duration
	mutex           sync.RWMutex
}

// TemplCompiler handles templ compilation
type TemplCompiler struct {
	command string
	args    []string
}

// BuildCache caches build results
type BuildCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int64
	ttl     time.Duration
}

// CacheEntry represents a cached build result
type CacheEntry struct {
	Key        string
	Value      []byte
	Hash       string
	CreatedAt  time.Time
	AccessedAt time.Time
	Size       int64
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
	}
}

// Start starts the build pipeline
func (bp *BuildPipeline) Start(ctx context.Context) {
	// Start workers
	for i := 0; i < bp.workers; i++ {
		go bp.worker(ctx)
	}
	
	// Start result processor
	go bp.processResults(ctx)
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
		bp.queue.results <- BuildResult{
			Component:    task.Component,
			Output:       result,
			Error:        nil,
			ParsedErrors: nil,
			Duration:     time.Since(start),
			CacheHit:     true,
			Hash:         contentHash,
		}
		return
	}
	
	// Execute build
	output, err := bp.compiler.Compile(task.Component)
	
	// Parse errors if build failed
	var parsedErrors []*errors.ParsedError
	if err != nil {
		parsedErrors = bp.errorParser.ParseError(string(output))
	}
	
	result := BuildResult{
		Component:    task.Component,
		Output:       output,
		Error:        err,
		ParsedErrors: parsedErrors,
		Duration:     time.Since(start),
		CacheHit:     false,
		Hash:         contentHash,
	}
	
	// Cache successful builds
	if err == nil {
		bp.cache.Set(contentHash, output)
	}
	
	bp.queue.results <- result
}

func (bp *BuildPipeline) processResults(ctx context.Context) {
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
	// Run templ generate command
	cmd := exec.Command(tc.command, tc.args...)
	cmd.Dir = "." // Run in current directory
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("templ generate failed: %w\nOutput: %s", err, output)
	}
	
	return output, nil
}

// BuildCache methods
func (bc *BuildCache) Get(key string) ([]byte, bool) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	entry, exists := bc.entries[key]
	if !exists {
		return nil, false
	}
	
	// Check TTL
	if time.Since(entry.CreatedAt) > bc.ttl {
		delete(bc.entries, key)
		return nil, false
	}
	
	// Update access time
	entry.AccessedAt = time.Now()
	return entry.Value, true
}

func (bc *BuildCache) Set(key string, value []byte) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
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
}

func (bc *BuildCache) evictIfNeeded(newSize int64) {
	currentSize := bc.getCurrentSize()
	if currentSize+newSize <= bc.maxSize {
		return
	}
	
	// Simple LRU eviction
	oldestKey := ""
	oldestTime := time.Now()
	
	for key, entry := range bc.entries {
		if entry.AccessedAt.Before(oldestTime) {
			oldestTime = entry.AccessedAt
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(bc.entries, oldestKey)
	}
}

func (bc *BuildCache) getCurrentSize() int64 {
	var size int64
	for _, entry := range bc.entries {
		size += entry.Size
	}
	return size
}

// Clear clears all cache entries
func (bc *BuildCache) Clear() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	bc.entries = make(map[string]*CacheEntry)
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

// generateContentHash generates a hash for component content
func (bp *BuildPipeline) generateContentHash(component *registry.ComponentInfo) string {
	// Read file content
	content, err := os.ReadFile(component.FilePath)
	if err != nil {
		// If we can't read the file, use file path and mod time
		stat, err := os.Stat(component.FilePath)
		if err != nil {
			return component.FilePath
		}
		return fmt.Sprintf("%s:%d", component.FilePath, stat.ModTime().Unix())
	}
	
	// Hash the content
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
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