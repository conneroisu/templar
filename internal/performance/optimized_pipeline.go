package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
)

// OptimizedBuildPipeline extends the build pipeline with performance optimizations
type OptimizedBuildPipeline struct {
	*build.BuildPipeline
	optimizer       *PerformanceOptimizer
	logger          logging.Logger
	
	// Enhanced metrics
	enqueuedTasks   int64 // atomic
	completedTasks  int64 // atomic
	failedTasks     int64 // atomic
	avgBuildTime    int64 // atomic (nanoseconds)
	
	// Adaptive worker management
	workerPool      *AdaptiveWorkerPool
	taskQueue       *PriorityTaskQueue
	resultProcessor *OptimizedResultProcessor
	
	// Performance monitoring
	perfMonitor     *BuildPerformanceMonitor
	
	// Circuit breaker for error handling
	circuitBreaker  *CircuitBreaker
}

// AdaptiveWorkerPool manages workers that scale based on load
type AdaptiveWorkerPool struct {
	minWorkers    int
	maxWorkers    int
	currentWorkers int32 // atomic
	
	workers       []chan *OptimizedBuildTask
	workerWg      sync.WaitGroup
	stopCh        chan struct{}
	
	loadThreshold float64
	scaleUpDelay  time.Duration
	scaleDownDelay time.Duration
	lastScaleTime time.Time
	scaleMutex    sync.Mutex
}

// PriorityTaskQueue implements a priority queue for build tasks
type PriorityTaskQueue struct {
	highPriority chan *OptimizedBuildTask
	normalPriority chan *OptimizedBuildTask
	lowPriority  chan *OptimizedBuildTask
	
	// Queue statistics
	enqueuedHigh   int64 // atomic
	enqueuedNormal int64 // atomic
	enqueuedLow    int64 // atomic
}

// OptimizedBuildTask extends BuildTask with performance metadata
type OptimizedBuildTask struct {
	*build.BuildTask
	SubmittedAt    time.Time
	EstimatedTime  time.Duration
	Dependencies   []string
	CacheKey       string
	RetryCount     int
	MaxRetries     int
}

// OptimizedResultProcessor handles build results with batching and optimization
type OptimizedResultProcessor struct {
	resultCh       chan *OptimizedBuildResult
	batchSize      int
	batchTimeout   time.Duration
	processor      func([]*OptimizedBuildResult) error
	
	currentBatch   []*OptimizedBuildResult
	batchTimer     *time.Timer
	batchMutex     sync.Mutex
}

// OptimizedBuildResult extends BuildResult with performance metrics
type OptimizedBuildResult struct {
	*build.BuildResult
	QueueTime     time.Duration
	ProcessTime   time.Duration
	TotalTime     time.Duration
	WorkerID      int
	RetryCount    int
	CacheMetadata map[string]interface{}
}

// BuildPerformanceMonitor tracks detailed performance metrics
type BuildPerformanceMonitor struct {
	metrics       map[string]*ComponentMetrics
	metricsMutex  sync.RWMutex
	
	// Global metrics
	totalThroughput    float64
	avgQueueTime       time.Duration
	avgProcessTime     time.Duration
	successRate        float64
	
	// Resource utilization
	cpuUtilization     float64
	memoryUtilization  float64
	diskIOUtilization  float64
}

// ComponentMetrics tracks metrics for individual components
type ComponentMetrics struct {
	BuildCount      int64
	SuccessCount    int64
	FailureCount    int64
	AvgBuildTime    time.Duration
	LastBuildTime   time.Time
	CacheHitRate    float64
	DependencyCount int
}

// CircuitBreaker prevents cascade failures
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	state            int32 // 0=closed, 1=open, 2=half-open
	failures         int32
	lastFailureTime  time.Time
	mutex            sync.RWMutex
}

// NewOptimizedBuildPipeline creates an optimized build pipeline
func NewOptimizedBuildPipeline(workers int, registry *registry.ComponentRegistry, logger logging.Logger) *OptimizedBuildPipeline {
	basePipeline := build.NewBuildPipeline(workers, registry)
	
	workerPool := &AdaptiveWorkerPool{
		minWorkers:     max(1, workers/2),
		maxWorkers:     workers * 2,
		currentWorkers: int32(workers),
		loadThreshold:  0.8,
		scaleUpDelay:   30 * time.Second,
		scaleDownDelay: 60 * time.Second,
		stopCh:         make(chan struct{}),
	}
	
	taskQueue := &PriorityTaskQueue{
		highPriority:   make(chan *OptimizedBuildTask, 50),
		normalPriority: make(chan *OptimizedBuildTask, 200),
		lowPriority:    make(chan *OptimizedBuildTask, 100),
	}
	
	resultProcessor := &OptimizedResultProcessor{
		resultCh:     make(chan *OptimizedBuildResult, 100),
		batchSize:    10,
		batchTimeout: 500 * time.Millisecond,
	}
	
	perfMonitor := &BuildPerformanceMonitor{
		metrics: make(map[string]*ComponentMetrics),
	}
	
	circuitBreaker := &CircuitBreaker{
		failureThreshold: 10,
		resetTimeout:     60 * time.Second,
	}
	
	optimizer := NewPerformanceOptimizer(basePipeline, registry, logger)
	
	return &OptimizedBuildPipeline{
		BuildPipeline:   basePipeline,
		optimizer:       optimizer,
		logger:          logger,
		workerPool:      workerPool,
		taskQueue:       taskQueue,
		resultProcessor: resultProcessor,
		perfMonitor:     perfMonitor,
		circuitBreaker:  circuitBreaker,
	}
}

// Start starts the optimized build pipeline
func (obp *OptimizedBuildPipeline) Start(ctx context.Context) error {
	// Start base pipeline
	obp.BuildPipeline.Start(ctx)
	
	// Start performance optimizer
	obp.optimizer.Start(ctx)
	
	// Start worker pool
	obp.workerPool.Start(ctx, obp.taskQueue, obp.resultProcessor.resultCh)
	
	// Start result processor
	obp.resultProcessor.Start(ctx, obp.processBuildResults)
	
	// Start performance monitor
	go obp.perfMonitor.Start(ctx)
	
	// Start adaptive scaling
	go obp.adaptiveScaling(ctx)
	
	if obp.logger != nil {
		obp.logger.Info(ctx, "Optimized build pipeline started",
			"min_workers", obp.workerPool.minWorkers,
			"max_workers", obp.workerPool.maxWorkers,
			"current_workers", obp.workerPool.currentWorkers)
	}
	
	return nil
}

// BuildOptimized queues a component for optimized building
func (obp *OptimizedBuildPipeline) BuildOptimized(component *registry.ComponentInfo, priority int) error {
	if !obp.circuitBreaker.AllowRequest() {
		return &logging.StructuredError{
			Category:  logging.ErrorCategorySystem,
			Operation: "build_optimized",
			Message:   "Circuit breaker is open",
			Retryable: true,
		}
	}
	
	task := &OptimizedBuildTask{
		BuildTask: &build.BuildTask{
			Component: component,
			Priority:  priority,
			Timestamp: time.Now(),
		},
		SubmittedAt:   time.Now(),
		EstimatedTime: obp.estimateBuildTime(component),
		CacheKey:      obp.generateCacheKey(component),
		MaxRetries:    3,
	}
	
	atomic.AddInt64(&obp.enqueuedTasks, 1)
	
	// Route to appropriate priority queue
	switch {
	case priority >= 8:
		obp.taskQueue.enqueueHigh(task)
	case priority >= 4:
		obp.taskQueue.enqueueNormal(task)
	default:
		obp.taskQueue.enqueueLow(task)
	}
	
	return nil
}

// adaptiveScaling monitors load and adjusts worker count
func (obp *OptimizedBuildPipeline) adaptiveScaling(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			obp.adjustWorkerCount()
		}
	}
}

// adjustWorkerCount scales workers based on queue load
func (obp *OptimizedBuildPipeline) adjustWorkerCount() {
	obp.workerPool.scaleMutex.Lock()
	defer obp.workerPool.scaleMutex.Unlock()
	
	// Calculate queue load
	queueLoad := obp.calculateQueueLoad()
	currentWorkers := int(atomic.LoadInt32(&obp.workerPool.currentWorkers))
	
	// Scale up if load is high
	if queueLoad > obp.workerPool.loadThreshold && 
	   currentWorkers < obp.workerPool.maxWorkers &&
	   time.Since(obp.workerPool.lastScaleTime) > obp.workerPool.scaleUpDelay {
		
		newWorkerCount := min(obp.workerPool.maxWorkers, currentWorkers+1)
		obp.scaleWorkers(newWorkerCount)
		obp.workerPool.lastScaleTime = time.Now()
		
		if obp.logger != nil {
			obp.logger.Info(context.Background(), "Scaled up workers",
				"previous_count", currentWorkers,
				"new_count", newWorkerCount,
				"queue_load", queueLoad)
		}
	}
	
	// Scale down if load is low
	if queueLoad < obp.workerPool.loadThreshold*0.3 && 
	   currentWorkers > obp.workerPool.minWorkers &&
	   time.Since(obp.workerPool.lastScaleTime) > obp.workerPool.scaleDownDelay {
		
		newWorkerCount := max(obp.workerPool.minWorkers, currentWorkers-1)
		obp.scaleWorkers(newWorkerCount)
		obp.workerPool.lastScaleTime = time.Now()
		
		if obp.logger != nil {
			obp.logger.Info(context.Background(), "Scaled down workers",
				"previous_count", currentWorkers,
				"new_count", newWorkerCount,
				"queue_load", queueLoad)
		}
	}
}

// calculateQueueLoad calculates current queue load percentage
func (obp *OptimizedBuildPipeline) calculateQueueLoad() float64 {
	highCount := len(obp.taskQueue.highPriority)
	normalCount := len(obp.taskQueue.normalPriority)
	lowCount := len(obp.taskQueue.lowPriority)
	
	totalQueued := float64(highCount + normalCount + lowCount)
	totalCapacity := float64(cap(obp.taskQueue.highPriority) + 
							cap(obp.taskQueue.normalPriority) + 
							cap(obp.taskQueue.lowPriority))
	
	if totalCapacity == 0 {
		return 0
	}
	
	return totalQueued / totalCapacity
}

// scaleWorkers adjusts the number of active workers
func (obp *OptimizedBuildPipeline) scaleWorkers(targetCount int) {
	currentCount := int(atomic.LoadInt32(&obp.workerPool.currentWorkers))
	
	if targetCount > currentCount {
		// Scale up - start new workers
		for i := currentCount; i < targetCount; i++ {
			obp.workerPool.startWorker(i)
		}
	} else if targetCount < currentCount {
		// Scale down - signal workers to stop
		for i := targetCount; i < currentCount; i++ {
			if i < len(obp.workerPool.workers) && obp.workerPool.workers[i] != nil {
				close(obp.workerPool.workers[i])
				obp.workerPool.workers[i] = nil
			}
		}
	}
	
	atomic.StoreInt32(&obp.workerPool.currentWorkers, int32(targetCount))
}

// estimateBuildTime estimates build time based on historical data
func (obp *OptimizedBuildPipeline) estimateBuildTime(component *registry.ComponentInfo) time.Duration {
	obp.perfMonitor.metricsMutex.RLock()
	defer obp.perfMonitor.metricsMutex.RUnlock()
	
	if metrics, exists := obp.perfMonitor.metrics[component.Name]; exists {
		return metrics.AvgBuildTime
	}
	
	// Default estimate for new components
	return 5 * time.Second
}

// generateCacheKey generates an optimized cache key
func (obp *OptimizedBuildPipeline) generateCacheKey(component *registry.ComponentInfo) string {
	// Use fast hash for cache key generation
	return component.Hash
}

// processBuildResults processes a batch of build results
func (obp *OptimizedBuildPipeline) processBuildResults(results []*OptimizedBuildResult) error {
	for _, result := range results {
		obp.updateMetrics(result)
		obp.circuitBreaker.RecordResult(result.Error == nil)
		
		if result.Error != nil {
			atomic.AddInt64(&obp.failedTasks, 1)
		} else {
			atomic.AddInt64(&obp.completedTasks, 1)
		}
	}
	
	return nil
}

// updateMetrics updates performance metrics for a component
func (obp *OptimizedBuildPipeline) updateMetrics(result *OptimizedBuildResult) {
	obp.perfMonitor.metricsMutex.Lock()
	defer obp.perfMonitor.metricsMutex.Unlock()
	
	componentName := result.Component.Name
	metrics, exists := obp.perfMonitor.metrics[componentName]
	if !exists {
		metrics = &ComponentMetrics{}
		obp.perfMonitor.metrics[componentName] = metrics
	}
	
	metrics.BuildCount++
	if result.Error == nil {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}
	
	// Update average build time
	if metrics.BuildCount == 1 {
		metrics.AvgBuildTime = result.Duration
	} else {
		// Exponential moving average
		alpha := 0.1
		metrics.AvgBuildTime = time.Duration(float64(metrics.AvgBuildTime)*(1-alpha) + 
											 float64(result.Duration)*alpha)
	}
	
	metrics.LastBuildTime = time.Now()
	
	// Update cache hit rate
	if result.CacheHit {
		metrics.CacheHitRate = (metrics.CacheHitRate*float64(metrics.BuildCount-1) + 1.0) / float64(metrics.BuildCount)
	} else {
		metrics.CacheHitRate = (metrics.CacheHitRate*float64(metrics.BuildCount-1)) / float64(metrics.BuildCount)
	}
}

// GetOptimizedMetrics returns enhanced performance metrics
func (obp *OptimizedBuildPipeline) GetOptimizedMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	metrics["enqueued_tasks"] = atomic.LoadInt64(&obp.enqueuedTasks)
	metrics["completed_tasks"] = atomic.LoadInt64(&obp.completedTasks)
	metrics["failed_tasks"] = atomic.LoadInt64(&obp.failedTasks)
	metrics["current_workers"] = atomic.LoadInt32(&obp.workerPool.currentWorkers)
	
	// Queue statistics
	metrics["queue_high_priority"] = len(obp.taskQueue.highPriority)
	metrics["queue_normal_priority"] = len(obp.taskQueue.normalPriority)
	metrics["queue_low_priority"] = len(obp.taskQueue.lowPriority)
	
	// Performance optimizer metrics
	optimizerMetrics := obp.optimizer.GetMetrics()
	metrics["memory_usage_mb"] = optimizerMetrics.MemoryUsage / (1024 * 1024)
	metrics["goroutine_count"] = optimizerMetrics.GoroutineCount
	metrics["cache_hit_rate"] = optimizerMetrics.CacheHitRate
	
	// Circuit breaker state
	metrics["circuit_breaker_state"] = obp.circuitBreaker.GetState()
	
	return metrics
}

// Priority queue methods for OptimizedTaskQueue
func (ptq *PriorityTaskQueue) enqueueHigh(task *OptimizedBuildTask) {
	select {
	case ptq.highPriority <- task:
		atomic.AddInt64(&ptq.enqueuedHigh, 1)
	default:
		// Queue full, could implement overflow handling
	}
}

func (ptq *PriorityTaskQueue) enqueueNormal(task *OptimizedBuildTask) {
	select {
	case ptq.normalPriority <- task:
		atomic.AddInt64(&ptq.enqueuedNormal, 1)
	default:
		// Fallback to low priority if normal is full
		ptq.enqueueLow(task)
	}
}

func (ptq *PriorityTaskQueue) enqueueLow(task *OptimizedBuildTask) {
	select {
	case ptq.lowPriority <- task:
		atomic.AddInt64(&ptq.enqueuedLow, 1)
	default:
		// Queue full, task dropped
	}
}

// Worker pool methods
func (awp *AdaptiveWorkerPool) Start(ctx context.Context, taskQueue *PriorityTaskQueue, resultCh chan *OptimizedBuildResult) {
	awp.workers = make([]chan *OptimizedBuildTask, awp.maxWorkers)
	
	// Start initial workers
	for i := 0; i < int(awp.currentWorkers); i++ {
		awp.startWorker(i)
	}
	
	// Start task dispatcher
	go awp.dispatchTasks(ctx, taskQueue)
}

func (awp *AdaptiveWorkerPool) startWorker(workerID int) {
	if workerID >= len(awp.workers) {
		return
	}
	
	awp.workers[workerID] = make(chan *OptimizedBuildTask, 1)
	awp.workerWg.Add(1)
	
	go func(id int, taskCh chan *OptimizedBuildTask) {
		defer awp.workerWg.Done()
		
		for task := range taskCh {
			// Process the task
			awp.processTask(id, task)
		}
	}(workerID, awp.workers[workerID])
}

func (awp *AdaptiveWorkerPool) dispatchTasks(ctx context.Context, taskQueue *PriorityTaskQueue) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-taskQueue.highPriority:
			awp.assignTask(task)
		case task := <-taskQueue.normalPriority:
			awp.assignTask(task)
		case task := <-taskQueue.lowPriority:
			awp.assignTask(task)
		}
	}
}

func (awp *AdaptiveWorkerPool) assignTask(task *OptimizedBuildTask) {
	// Find available worker
	currentWorkers := int(atomic.LoadInt32(&awp.currentWorkers))
	for i := 0; i < currentWorkers; i++ {
		if awp.workers[i] != nil {
			select {
			case awp.workers[i] <- task:
				return
			default:
				continue
			}
		}
	}
}

func (awp *AdaptiveWorkerPool) processTask(workerID int, task *OptimizedBuildTask) {
	// Implementation would process the actual build task
	// This is a placeholder for the actual build logic
}

// Circuit breaker methods
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	state := atomic.LoadInt32(&cb.state)
	
	switch state {
	case 0: // Closed
		return true
	case 1: // Open
		return time.Since(cb.lastFailureTime) > cb.resetTimeout
	case 2: // Half-open
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	if success {
		atomic.StoreInt32(&cb.failures, 0)
		atomic.StoreInt32(&cb.state, 0) // Closed
	} else {
		failures := atomic.AddInt32(&cb.failures, 1)
		cb.lastFailureTime = time.Now()
		
		if failures >= int32(cb.failureThreshold) {
			atomic.StoreInt32(&cb.state, 1) // Open
		}
	}
}

func (cb *CircuitBreaker) GetState() string {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case 0:
		return "closed"
	case 1:
		return "open"
	case 2:
		return "half-open"
	default:
		return "unknown"
	}
}

// Result processor methods
func (orp *OptimizedResultProcessor) Start(ctx context.Context, processor func([]*OptimizedBuildResult) error) {
	orp.processor = processor
	go orp.processBatches(ctx)
}

func (orp *OptimizedResultProcessor) processBatches(ctx context.Context) {
	orp.batchTimer = time.NewTimer(orp.batchTimeout)
	defer orp.batchTimer.Stop()
	
	for {
		select {
		case <-ctx.Done():
			orp.flushBatch()
			return
		case result := <-orp.resultCh:
			orp.addToBatch(result)
		case <-orp.batchTimer.C:
			orp.flushBatch()
			orp.batchTimer.Reset(orp.batchTimeout)
		}
	}
}

func (orp *OptimizedResultProcessor) addToBatch(result *OptimizedBuildResult) {
	orp.batchMutex.Lock()
	defer orp.batchMutex.Unlock()
	
	orp.currentBatch = append(orp.currentBatch, result)
	
	if len(orp.currentBatch) >= orp.batchSize {
		orp.flushBatch()
	}
}

func (orp *OptimizedResultProcessor) flushBatch() {
	orp.batchMutex.Lock()
	defer orp.batchMutex.Unlock()
	
	if len(orp.currentBatch) > 0 && orp.processor != nil {
		orp.processor(orp.currentBatch)
		orp.currentBatch = orp.currentBatch[:0] // Reset slice
	}
}

// Performance monitor methods
func (bpm *BuildPerformanceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bpm.updateGlobalMetrics()
		}
	}
}

func (bpm *BuildPerformanceMonitor) updateGlobalMetrics() {
	bpm.metricsMutex.RLock()
	defer bpm.metricsMutex.RUnlock()
	
	var totalBuilds, totalSuccesses int64
	var totalBuildTime time.Duration
	
	for _, metrics := range bpm.metrics {
		totalBuilds += metrics.BuildCount
		totalSuccesses += metrics.SuccessCount
		totalBuildTime += metrics.AvgBuildTime
	}
	
	if totalBuilds > 0 {
		bpm.successRate = float64(totalSuccesses) / float64(totalBuilds)
		bpm.avgProcessTime = totalBuildTime / time.Duration(len(bpm.metrics))
	}
}