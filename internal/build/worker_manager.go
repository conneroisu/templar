// Package build provides worker management for parallel build processing.
//
// WorkerManager implements a configurable worker pool that processes build tasks
// concurrently with proper lifecycle management, resource limits, and graceful
// shutdown capabilities. It achieves high throughput through optimized task
// distribution and memory pool utilization.
package build

import (
	"context"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
)

// WorkerManager manages a pool of build workers with configurable parallelism.
// It provides efficient task distribution, resource management, and monitoring
// capabilities for optimal build performance.
type WorkerManager struct {
	// workers defines the number of concurrent build workers
	workers int
	// workerPool manages the lifecycle of build workers
	workerPool *WorkerPool
	// compiler handles templ compilation with security validation
	compiler *TemplCompiler
	// hashProvider generates content hashes for cache keys
	hashProvider interfaces.HashProvider
	// metrics tracks build performance and worker utilization
	metrics *BuildMetrics
	// objectPools optimize memory allocation for frequently used objects
	objectPools *ObjectPools
	// errorParser processes build errors and provides detailed diagnostics
	errorParser *errors.ErrorParser
	// workerWg synchronizes worker goroutine lifecycle
	workerWg sync.WaitGroup
	// cancel terminates all worker operations gracefully
	cancel context.CancelFunc
	// mu protects concurrent access to worker state
	mu sync.RWMutex
}

// NewWorkerManager creates a new worker manager with the specified configuration.
func NewWorkerManager(
	workers int,
	compiler *TemplCompiler,
	hashProvider interfaces.HashProvider,
	metrics *BuildMetrics,
	objectPools *ObjectPools,
	errorParser *errors.ErrorParser,
) *WorkerManager {
	return &WorkerManager{
		workers:      workers,
		workerPool:   NewWorkerPool(),
		compiler:     compiler,
		hashProvider: hashProvider,
		metrics:      metrics,
		objectPools:  objectPools,
		errorParser:  errorParser,
	}
}

// StartWorkers begins worker goroutines with the given context and task queue.
// Workers will process tasks until the context is cancelled or StopWorkers is called.
func (wm *WorkerManager) StartWorkers(ctx context.Context, queue interfaces.TaskQueue) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Create cancellable context for workers
	ctx, wm.cancel = context.WithCancel(ctx)

	// Start the configured number of workers
	for i := 0; i < wm.workers; i++ {
		wm.workerWg.Add(1)
		go wm.worker(ctx, queue)
	}
}

// StopWorkers gracefully shuts down all workers and waits for completion.
func (wm *WorkerManager) StopWorkers() {
	wm.mu.RLock()
	cancel := wm.cancel
	wm.mu.RUnlock()

	if cancel != nil {
		cancel()
	}

	// Wait for all workers to finish
	wm.workerWg.Wait()
}

// SetWorkerCount adjusts the number of active workers.
// This allows dynamic scaling based on system load.
func (wm *WorkerManager) SetWorkerCount(count int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.workers = count
	// Note: Actual worker adjustment would require more complex logic
	// to gracefully stop/start workers. This is a simplified implementation.
}

// worker is the main worker goroutine that processes build tasks.
func (wm *WorkerManager) worker(ctx context.Context, queue interfaces.TaskQueue) {
	defer wm.workerWg.Done()

	taskChan := queue.GetNextTask()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-taskChan:
			if !ok {
				return // Queue closed
			}

			buildTask, ok := task.(BuildTask)
			if !ok {
				continue // Invalid task type
			}

			// Process the build task
			result := wm.processBuildTask(ctx, buildTask)

			// Publish the result - ignore publication errors as they don't affect core functionality
			_ = queue.PublishResult(result)
		}
	}
}

// processBuildTask processes a single build task and returns the result.
func (wm *WorkerManager) processBuildTask(ctx context.Context, task BuildTask) BuildResult {
	startTime := time.Now()

	// Use object pool for build result
	buildResult := wm.objectPools.GetBuildResult()
	buildResult.Component = task.Component
	buildResult.CacheHit = false
	buildResult.Hash = ""

	// Generate hash for caching
	hash := wm.hashProvider.GenerateContentHash(task.Component)
	buildResult.Hash = hash

	// Check cache first (if cache is integrated into the system)
	// For now, we'll proceed with compilation

	// Execute build with pooled output buffer
	output, err := wm.compiler.CompileWithPools(ctx, task.Component, wm.objectPools)

	// Parse errors if build failed
	var parsedErrors []*errors.ParsedError
	if err != nil {
		// Wrap the error with build context for better debugging
		err = errors.WrapBuild(err, errors.ErrCodeBuildFailed,
			"component compilation failed", task.Component.Name).
			WithLocation(task.Component.FilePath, 0, 0)
		parsedErrors = wm.errorParser.ParseError(string(output))
	}

	buildResult.Output = output
	buildResult.Error = err
	buildResult.ParsedErrors = parsedErrors
	buildResult.Duration = time.Since(startTime)

	// Update metrics
	if wm.metrics != nil {
		wm.metrics.RecordBuild(*buildResult)
	}

	return *buildResult
}

// GetWorkerStats returns current worker pool statistics.
func (wm *WorkerManager) GetWorkerStats() WorkerStats {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return WorkerStats{
		ActiveWorkers: wm.workers,
		// Note: Pool task tracking methods would need to be implemented
		TotalTasks:      0,
		CompletedTasks:  0,
		FailedTasks:     0,
		AverageTaskTime: 0,
	}
}

// WorkerStats provides worker pool performance metrics.
type WorkerStats struct {
	ActiveWorkers   int
	TotalTasks      int64
	CompletedTasks  int64
	FailedTasks     int64
	AverageTaskTime time.Duration
}

// Verify that WorkerManager implements the WorkerManager interface
var _ interfaces.WorkerManager = (*WorkerManager)(nil)
