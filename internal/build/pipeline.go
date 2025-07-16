package build

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// BuildPipeline manages the build process for templ components
type BuildPipeline struct {
	compiler *TemplCompiler
	cache    *BuildCache
	queue    *BuildQueue
	workers  int
	registry *registry.ComponentRegistry
}

// BuildTask represents a build task
type BuildTask struct {
	Component *registry.ComponentInfo
	Priority  int
	Timestamp time.Time
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Component *registry.ComponentInfo
	Output    []byte
	Error     error
	Duration  time.Duration
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
	mutex    sync.Mutex
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
	
	return &BuildPipeline{
		compiler: compiler,
		cache:    cache,
		queue:    queue,
		workers:  workers,
		registry: registry,
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
	
	// Check cache first
	if result, found := bp.cache.Get(task.Component.Hash); found {
		bp.queue.results <- BuildResult{
			Component: task.Component,
			Output:    result,
			Error:     nil,
			Duration:  time.Since(start),
		}
		return
	}
	
	// Execute build
	output, err := bp.compiler.Compile(task.Component)
	
	result := BuildResult{
		Component: task.Component,
		Output:    output,
		Error:     err,
		Duration:  time.Since(start),
	}
	
	// Cache successful builds
	if err == nil {
		bp.cache.Set(task.Component.Hash, output)
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
	if result.Error != nil {
		fmt.Printf("Build failed for %s: %v\n", result.Component.Name, result.Error)
	} else {
		fmt.Printf("Build succeeded for %s in %v\n", result.Component.Name, result.Duration)
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