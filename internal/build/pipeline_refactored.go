// Package build provides a refactored concurrent build pipeline following
// single responsibility principle with clean separation of concerns.
//
// The refactored pipeline orchestrates specialized components:
// - TaskQueueManager: Handles task queuing with priority support
// - WorkerManager: Manages worker pool lifecycle and task processing
// - HashProvider: Provides efficient content hash generation
// - ResultProcessor: Handles result processing and callback management
package build

import (
	"context"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/types"
)

// RefactoredBuildPipeline orchestrates the build process using separated components.
// It follows the single responsibility principle by delegating specific concerns
// to specialized components while maintaining high performance and clean interfaces.
type RefactoredBuildPipeline struct {
	// queueManager handles task queuing with priority support
	queueManager *TaskQueueManager
	// workerManager manages worker pool lifecycle and task processing
	workerManager interfaces.WorkerManager
	// resultProcessor handles result processing and callback management
	resultProcessor interfaces.ResultProcessor
	// hashProvider provides efficient content hash generation
	hashProvider interfaces.HashProvider
	// cache provides LRU-based build result caching
	cache *BuildCache
	// metrics tracks build performance and success rates
	metrics *BuildMetrics
	// registry provides component information and change notifications
	registry interfaces.ComponentRegistry
	// cancel terminates all pipeline operations gracefully
	cancel context.CancelFunc
	// mu protects concurrent access to pipeline state
	mu sync.RWMutex
	// started indicates if the pipeline is currently running
	started bool
}

// NewRefactoredBuildPipeline creates a new refactored build pipeline with separated components.
func NewRefactoredBuildPipeline(
	workers int,
	registry interfaces.ComponentRegistry,
) *RefactoredBuildPipeline {
	// Initialize core components
	cache := NewBuildCache(100*1024*1024, time.Hour) // 100MB, 1 hour TTL
	metrics := NewBuildMetrics()
	errorParser := errors.NewErrorParser()
	objectPools := NewObjectPools()

	// Create specialized components
	queueManager := NewTaskQueueManager(
		100,
		100,
		10,
		metrics,
	) // tasks, results, priority buffer sizes
	hashProvider := NewHashProvider(cache)
	compiler := NewTemplCompiler()

	workerManager := NewWorkerManager(
		workers,
		compiler,
		hashProvider,
		metrics,
		objectPools,
		errorParser,
	)
	resultProcessor := NewResultProcessor(metrics, errorParser)

	return &RefactoredBuildPipeline{
		queueManager:    queueManager,
		workerManager:   workerManager,
		resultProcessor: resultProcessor,
		hashProvider:    hashProvider,
		cache:           cache,
		metrics:         metrics,
		registry:        registry,
		started:         false,
	}
}

// Start begins the build pipeline with the given context.
// This starts all component managers and begins processing tasks.
func (rbp *RefactoredBuildPipeline) Start(ctx context.Context) error {
	rbp.mu.Lock()
	defer rbp.mu.Unlock()

	if rbp.started {
		return errors.NewBuildError(
			"ERR_PIPELINE_ALREADY_STARTED",
			"pipeline is already started",
			nil,
		)
	}

	// Create cancellable context for all components
	ctx, rbp.cancel = context.WithCancel(ctx)

	// Start result processor first
	resultChan := rbp.queueManager.GetResults()
	rbp.resultProcessor.ProcessResults(ctx, resultChan)

	// Start worker manager
	rbp.workerManager.StartWorkers(ctx, rbp.queueManager)

	rbp.started = true

	return nil
}

// Stop gracefully shuts down the build pipeline and waits for all components to finish.
func (rbp *RefactoredBuildPipeline) Stop() error {
	rbp.mu.Lock()
	defer rbp.mu.Unlock()

	if !rbp.started {
		return nil // Already stopped or never started
	}

	// Cancel context to signal shutdown
	if rbp.cancel != nil {
		rbp.cancel()
	}

	// Stop components in reverse order
	rbp.workerManager.StopWorkers()
	rbp.resultProcessor.Stop()
	rbp.queueManager.Close()

	rbp.started = false

	return nil
}

// Build processes a single component through the pipeline.
func (rbp *RefactoredBuildPipeline) Build(component *types.ComponentInfo) error {
	if !rbp.started {
		return errors.NewBuildError("ERR_PIPELINE_NOT_STARTED", "pipeline is not started", nil)
	}

	task := BuildTask{
		Component: component,
		Priority:  0, // Normal priority
		Timestamp: time.Now(),
	}

	return rbp.queueManager.Enqueue(task)
}

// BuildWithPriority builds a component with high priority.
func (rbp *RefactoredBuildPipeline) BuildWithPriority(component *types.ComponentInfo) {
	if !rbp.started {
		return // Cannot enqueue if not started
	}

	task := BuildTask{
		Component: component,
		Priority:  1, // High priority
		Timestamp: time.Now(),
	}

	rbp.queueManager.EnqueuePriority(task)
}

// AddCallback registers a callback for build completion events.
func (rbp *RefactoredBuildPipeline) AddCallback(callback interfaces.BuildCallbackFunc) {
	rbp.resultProcessor.AddCallback(callback)
}

// GetMetrics returns current build metrics.
func (rbp *RefactoredBuildPipeline) GetMetrics() interfaces.BuildMetrics {
	return rbp.metrics
}

// GetCache returns cache statistics.
func (rbp *RefactoredBuildPipeline) GetCache() interfaces.CacheStats {
	return rbp.cache
}

// ClearCache clears the build cache.
func (rbp *RefactoredBuildPipeline) ClearCache() {
	rbp.cache.Clear()
	// Note: ClearMmapCache method will be added to interface if needed
}

// GetQueueStats returns current queue statistics.
func (rbp *RefactoredBuildPipeline) GetQueueStats() QueueStats {
	return rbp.queueManager.GetQueueStats()
}

// GetWorkerStats returns current worker statistics.
func (rbp *RefactoredBuildPipeline) GetWorkerStats() WorkerStats {
	// Use concrete type access for extended functionality
	if concreteWorker, ok := rbp.workerManager.(*WorkerManager); ok {
		return concreteWorker.GetWorkerStats()
	}

	return WorkerStats{}
}

// GetHashStats returns hash provider statistics.
func (rbp *RefactoredBuildPipeline) GetHashStats() HashCacheStats {
	// Use concrete type access for extended functionality
	if concreteHash, ok := rbp.hashProvider.(*HashProvider); ok {
		return concreteHash.GetCacheStats()
	}

	return HashCacheStats{}
}

// SetWorkerCount adjusts the number of active workers.
func (rbp *RefactoredBuildPipeline) SetWorkerCount(count int) {
	rbp.workerManager.SetWorkerCount(count)
}

// GetComponentRegistry returns the associated component registry.
func (rbp *RefactoredBuildPipeline) GetComponentRegistry() interfaces.ComponentRegistry {
	return rbp.registry
}

// IsStarted returns whether the pipeline is currently running.
func (rbp *RefactoredBuildPipeline) IsStarted() bool {
	rbp.mu.RLock()
	defer rbp.mu.RUnlock()

	return rbp.started
}

// GetPipelineStats returns comprehensive pipeline statistics.
func (rbp *RefactoredBuildPipeline) GetPipelineStats() PipelineStats {
	return PipelineStats{
		Started:      rbp.IsStarted(),
		QueueStats:   rbp.GetQueueStats(),
		WorkerStats:  rbp.GetWorkerStats(),
		HashStats:    rbp.GetHashStats(),
		MetricsStats: rbp.metrics,
	}
}

// PipelineStats provides comprehensive pipeline performance metrics.
type PipelineStats struct {
	Started      bool
	QueueStats   QueueStats
	WorkerStats  WorkerStats
	HashStats    HashCacheStats
	MetricsStats interface{}
}

// Verify that RefactoredBuildPipeline implements the BuildPipeline interface.
var _ interfaces.BuildPipeline = (*RefactoredBuildPipeline)(nil)
