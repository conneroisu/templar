package performance

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
)

// PerformanceOptimizer manages performance optimizations
type PerformanceOptimizer struct {
	buildPipeline *build.BuildPipeline
	registry      *registry.ComponentRegistry
	logger        logging.Logger
	
	// Performance metrics
	metrics       *PerformanceMetrics
	cpuOptimizer  *CPUOptimizer
	memOptimizer  *MemoryOptimizer
	ioOptimizer   *IOOptimizer
	cacheOptimizer *CacheOptimizer
	
	// Optimization settings
	config        *OptimizationConfig
	isOptimizing  int32 // atomic flag
}

// PerformanceMetrics tracks system performance
type PerformanceMetrics struct {
	mutex           sync.RWMutex
	CPUUsage        float64
	MemoryUsage     int64
	GoroutineCount  int
	GCPauseTime     time.Duration
	DiskIORate      float64
	NetworkIORate   float64
	CacheHitRate    float64
	BuildThroughput float64
	LastUpdated     time.Time
}

// OptimizationConfig holds performance optimization settings
type OptimizationConfig struct {
	EnableCPUOptimization    bool
	EnableMemoryOptimization bool
	EnableIOOptimization     bool
	EnableCacheOptimization  bool
	
	MaxGoroutines           int
	GCTargetPercent         int
	IOConcurrencyLimit      int
	CacheOptimizationLevel  int
	
	MonitoringInterval      time.Duration
	OptimizationInterval    time.Duration
	MemoryThreshold         float64 // 0.0 to 1.0
	CPUThreshold            float64 // 0.0 to 1.0
}

// DefaultOptimizationConfig returns default optimization settings
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		EnableCPUOptimization:    true,
		EnableMemoryOptimization: true,
		EnableIOOptimization:     true,
		EnableCacheOptimization:  true,
		
		MaxGoroutines:           runtime.GOMAXPROCS(0) * 4,
		GCTargetPercent:         100,
		IOConcurrencyLimit:      runtime.GOMAXPROCS(0) * 2,
		CacheOptimizationLevel:  2,
		
		MonitoringInterval:      5 * time.Second,
		OptimizationInterval:    30 * time.Second,
		MemoryThreshold:         0.8, // 80%
		CPUThreshold:            0.9, // 90%
	}
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(buildPipeline *build.BuildPipeline, registry *registry.ComponentRegistry, logger logging.Logger) *PerformanceOptimizer {
	config := DefaultOptimizationConfig()
	
	return &PerformanceOptimizer{
		buildPipeline:  buildPipeline,
		registry:       registry,
		logger:         logger,
		metrics:        &PerformanceMetrics{},
		cpuOptimizer:   NewCPUOptimizer(config),
		memOptimizer:   NewMemoryOptimizer(config),
		ioOptimizer:    NewIOOptimizer(config),
		cacheOptimizer: NewCacheOptimizer(config),
		config:         config,
	}
}

// Start begins performance monitoring and optimization
func (po *PerformanceOptimizer) Start(ctx context.Context) {
	// Start monitoring goroutine
	go po.monitorPerformance(ctx)
	
	// Start optimization goroutine
	go po.optimizePerformance(ctx)
	
	if po.logger != nil {
		po.logger.Info(ctx, "Performance optimizer started",
			"cpu_optimization", po.config.EnableCPUOptimization,
			"memory_optimization", po.config.EnableMemoryOptimization,
			"io_optimization", po.config.EnableIOOptimization,
			"cache_optimization", po.config.EnableCacheOptimization)
	}
}

// monitorPerformance continuously monitors system performance
func (po *PerformanceOptimizer) monitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(po.config.MonitoringInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			po.updateMetrics()
		}
	}
}

// optimizePerformance applies optimizations based on metrics
func (po *PerformanceOptimizer) optimizePerformance(ctx context.Context) {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if atomic.CompareAndSwapInt32(&po.isOptimizing, 0, 1) {
				po.performOptimizations(ctx)
				atomic.StoreInt32(&po.isOptimizing, 0)
			}
		}
	}
}

// updateMetrics collects current performance metrics
func (po *PerformanceOptimizer) updateMetrics() {
	po.metrics.mutex.Lock()
	defer po.metrics.mutex.Unlock()
	
	// Update runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	po.metrics.MemoryUsage = int64(memStats.Alloc)
	po.metrics.GoroutineCount = runtime.NumGoroutine()
	po.metrics.GCPauseTime = time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
	po.metrics.LastUpdated = time.Now()
	
	// Calculate cache hit rate from build pipeline
	if po.buildPipeline != nil {
		buildMetrics := po.buildPipeline.GetMetrics()
		if buildMetrics.TotalBuilds > 0 {
			po.metrics.CacheHitRate = float64(buildMetrics.CacheHits) / float64(buildMetrics.TotalBuilds)
		}
		
		// Calculate build throughput (builds per second)
		if buildMetrics.TotalDuration > 0 {
			po.metrics.BuildThroughput = float64(buildMetrics.TotalBuilds) / buildMetrics.TotalDuration.Seconds()
		}
	}
}

// performOptimizations applies all enabled optimizations
func (po *PerformanceOptimizer) performOptimizations(ctx context.Context) {
	metrics := po.GetMetrics()
	
	if po.config.EnableMemoryOptimization {
		po.memOptimizer.Optimize(ctx, metrics)
	}
	
	if po.config.EnableCPUOptimization {
		po.cpuOptimizer.Optimize(ctx, metrics)
	}
	
	if po.config.EnableIOOptimization {
		po.ioOptimizer.Optimize(ctx, metrics)
	}
	
	if po.config.EnableCacheOptimization {
		po.cacheOptimizer.Optimize(ctx, metrics, po.buildPipeline)
	}
	
	if po.logger != nil {
		po.logger.Debug(ctx, "Performance optimization cycle completed",
			"memory_usage_mb", metrics.MemoryUsage/(1024*1024),
			"goroutine_count", metrics.GoroutineCount,
			"cache_hit_rate", metrics.CacheHitRate,
			"build_throughput", metrics.BuildThroughput)
	}
}

// GetMetrics returns current performance metrics
func (po *PerformanceOptimizer) GetMetrics() PerformanceMetrics {
	po.metrics.mutex.RLock()
	defer po.metrics.mutex.RUnlock()
	return *po.metrics
}

// CPUOptimizer optimizes CPU usage
type CPUOptimizer struct {
	config *OptimizationConfig
}

// NewCPUOptimizer creates a new CPU optimizer
func NewCPUOptimizer(config *OptimizationConfig) *CPUOptimizer {
	return &CPUOptimizer{config: config}
}

// Optimize applies CPU optimizations
func (co *CPUOptimizer) Optimize(ctx context.Context, metrics PerformanceMetrics) {
	// Adjust GOMAXPROCS based on load
	currentProcs := runtime.GOMAXPROCS(0)
	targetProcs := currentProcs
	
	if metrics.CPUUsage > co.config.CPUThreshold {
		// High CPU usage - consider reducing GOMAXPROCS slightly
		targetProcs = max(1, currentProcs-1)
	} else if metrics.CPUUsage < 0.5 {
		// Low CPU usage - can increase GOMAXPROCS
		maxProcs := runtime.NumCPU()
		targetProcs = min(maxProcs, currentProcs+1)
	}
	
	if targetProcs != currentProcs {
		runtime.GOMAXPROCS(targetProcs)
	}
}

// MemoryOptimizer optimizes memory usage
type MemoryOptimizer struct {
	config        *OptimizationConfig
	lastGCForced  time.Time
	gcCooldown    time.Duration
}

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer(config *OptimizationConfig) *MemoryOptimizer {
	return &MemoryOptimizer{
		config:     config,
		gcCooldown: 30 * time.Second,
	}
}

// Optimize applies memory optimizations
func (mo *MemoryOptimizer) Optimize(ctx context.Context, metrics PerformanceMetrics) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Calculate memory usage percentage
	memoryUsagePercent := float64(memStats.Alloc) / float64(memStats.Sys)
	
	// Force GC if memory usage is high and cooldown has passed
	if memoryUsagePercent > mo.config.MemoryThreshold {
		if time.Since(mo.lastGCForced) > mo.gcCooldown {
			runtime.GC()
			mo.lastGCForced = time.Now()
		}
	}
	
	// Adjust GC target percentage based on memory pressure
	if memoryUsagePercent > 0.9 {
		// High memory pressure - trigger GC more frequently
		runtime.GOMAXPROCS(50) // Lower GC target
	} else if memoryUsagePercent < 0.3 {
		// Low memory pressure - allow more garbage before collection
		runtime.GOMAXPROCS(200) // Higher GC target
	}
}

// IOOptimizer optimizes I/O operations
type IOOptimizer struct {
	config      *OptimizationConfig
	ioLimiter   chan struct{}
	ioLimiterMu sync.Mutex
}

// NewIOOptimizer creates a new I/O optimizer
func NewIOOptimizer(config *OptimizationConfig) *IOOptimizer {
	optimizer := &IOOptimizer{config: config}
	optimizer.initIOLimiter()
	return optimizer
}

// initIOLimiter initializes the I/O concurrency limiter
func (io *IOOptimizer) initIOLimiter() {
	io.ioLimiterMu.Lock()
	defer io.ioLimiterMu.Unlock()
	
	if io.ioLimiter != nil {
		// Drain existing limiter
		close(io.ioLimiter)
	}
	
	io.ioLimiter = make(chan struct{}, io.config.IOConcurrencyLimit)
}

// Optimize applies I/O optimizations
func (io *IOOptimizer) Optimize(ctx context.Context, metrics PerformanceMetrics) {
	// Adjust I/O concurrency based on system load
	newLimit := io.config.IOConcurrencyLimit
	
	if metrics.GoroutineCount > io.config.MaxGoroutines {
		// Too many goroutines - reduce I/O concurrency
		newLimit = max(1, newLimit/2)
	} else if metrics.GoroutineCount < io.config.MaxGoroutines/2 {
		// Low goroutine count - can increase I/O concurrency
		newLimit = min(runtime.GOMAXPROCS(0)*4, newLimit*2)
	}
	
	if newLimit != io.config.IOConcurrencyLimit {
		io.config.IOConcurrencyLimit = newLimit
		io.initIOLimiter()
	}
}

// AcquireIOSlot acquires a slot for I/O operation (blocks if limit reached)
func (io *IOOptimizer) AcquireIOSlot(ctx context.Context) error {
	select {
	case io.ioLimiter <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseIOSlot releases an I/O slot
func (io *IOOptimizer) ReleaseIOSlot() {
	select {
	case <-io.ioLimiter:
	default:
		// Should not happen, but handle gracefully
	}
}

// CacheOptimizer optimizes cache performance
type CacheOptimizer struct {
	config *OptimizationConfig
}

// NewCacheOptimizer creates a new cache optimizer
func NewCacheOptimizer(config *OptimizationConfig) *CacheOptimizer {
	return &CacheOptimizer{config: config}
}

// Optimize applies cache optimizations
func (co *CacheOptimizer) Optimize(ctx context.Context, metrics PerformanceMetrics, buildPipeline *build.BuildPipeline) {
	if buildPipeline == nil {
		return
	}
	
	// Get current cache stats
	_, size, maxSize := buildPipeline.GetCacheStats()
	
	// Cache efficiency analysis
	cacheUsagePercent := float64(size) / float64(maxSize)
	
	// If cache hit rate is low and cache is full, consider clearing some entries
	if metrics.CacheHitRate < 0.3 && cacheUsagePercent > 0.9 {
		// Poor cache performance with high usage - might need cleanup
		// This would typically be handled by the LRU mechanism, but we can force it
		if co.config.CacheOptimizationLevel >= 2 {
			// Aggressive optimization - clear cache periodically
			buildPipeline.ClearCache()
		}
	}
	
	// Preemptive cache warming for frequently accessed components
	if co.config.CacheOptimizationLevel >= 3 {
		co.warmCache(ctx, buildPipeline)
	}
}

// warmCache preemptively warms the cache with frequently accessed components
func (co *CacheOptimizer) warmCache(ctx context.Context, buildPipeline *build.BuildPipeline) {
	// This would analyze component access patterns and preload frequently used ones
	// Implementation would depend on having access to component usage statistics
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OptimizedFileScanner provides optimized file scanning with I/O limiting
type OptimizedFileScanner struct {
	ioOptimizer *IOOptimizer
	scanner     interface{} // Would be the actual scanner interface
}

// NewOptimizedFileScanner creates an optimized file scanner
func NewOptimizedFileScanner(ioOptimizer *IOOptimizer) *OptimizedFileScanner {
	return &OptimizedFileScanner{
		ioOptimizer: ioOptimizer,
	}
}

// ScanFileOptimized scans a file with I/O optimization
func (ofs *OptimizedFileScanner) ScanFileOptimized(ctx context.Context, filePath string) error {
	// Acquire I/O slot
	if err := ofs.ioOptimizer.AcquireIOSlot(ctx); err != nil {
		return err
	}
	defer ofs.ioOptimizer.ReleaseIOSlot()
	
	// Perform the actual file scanning
	// This would call the underlying scanner implementation
	return nil
}

// BatchProcessor provides optimized batch processing
type BatchProcessor struct {
	batchSize     int
	workerCount   int
	processingCh  chan interface{}
	resultCh      chan interface{}
	errorCh       chan error
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize, workerCount int) *BatchProcessor {
	return &BatchProcessor{
		batchSize:    batchSize,
		workerCount:  workerCount,
		processingCh: make(chan interface{}, batchSize*2),
		resultCh:     make(chan interface{}, batchSize*2),
		errorCh:      make(chan error, workerCount),
	}
}

// ProcessBatch processes items in optimized batches
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, items []interface{}, processor func(interface{}) (interface{}, error)) ([]interface{}, error) {
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < bp.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range bp.processingCh {
				result, err := processor(item)
				if err != nil {
					bp.errorCh <- err
					return
				}
				bp.resultCh <- result
			}
		}()
	}
	
	// Send items for processing
	go func() {
		defer close(bp.processingCh)
		for _, item := range items {
			select {
			case bp.processingCh <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Collect results
	var results []interface{}
	var processingErrors []error
	
	go func() {
		wg.Wait()
		close(bp.resultCh)
		close(bp.errorCh)
	}()
	
	for {
		select {
		case result, ok := <-bp.resultCh:
			if !ok {
				goto done
			}
			results = append(results, result)
		case err, ok := <-bp.errorCh:
			if !ok {
				goto done
			}
			processingErrors = append(processingErrors, err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
done:
	if len(processingErrors) > 0 {
		return results, processingErrors[0] // Return first error
	}
	
	return results, nil
}