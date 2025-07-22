// Package build provides build metrics tracking and performance monitoring.
package build

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
)

// BuildMetrics tracks build performance and queue health
type BuildMetrics struct {
	TotalBuilds      int64
	SuccessfulBuilds int64
	FailedBuilds     int64
	CacheHits        int64
	AverageDuration  time.Duration
	TotalDuration    time.Duration

	// Queue monitoring for reliability
	DroppedTasks   int64            // Count of tasks dropped due to queue full
	DroppedResults int64            // Count of results dropped due to queue full
	DropReasons    map[string]int64 // Track reasons for drops

	// Parallel processing metrics (atomic for high-frequency updates)
	ParallelFileDiscoveries int64 // Total file discoveries
	FileDiscoveryDuration   int64 // Total time spent discovering files (nanoseconds)
	ParallelProcessingTime  int64 // Total time spent in parallel processing (nanoseconds)
	WorkerUtilization      int64 // Average worker utilization percentage
	BatchProcessingCount   int64 // Number of batch operations
	BatchSize              int64 // Average batch size
	ConcurrencyLevel       int32 // Current concurrency level
	PeakConcurrency        int32 // Peak concurrency achieved

	// AST caching metrics
	ASTCacheHits   int64 // AST cache hits
	ASTCacheMisses int64 // AST cache misses
	ASTParseTime   int64 // Total AST parsing time (nanoseconds)

	// Memory optimization metrics
	PoolHits       int64 // Object pool cache hits
	PoolMisses     int64 // Object pool cache misses
	MemoryReused   int64 // Bytes of memory reused from pools

	mutex sync.RWMutex
}

// Ensure BuildMetrics implements the interfaces.BuildMetrics interface
var _ interfaces.BuildMetrics = (*BuildMetrics)(nil)

// NewBuildMetrics creates a new build metrics tracker
func NewBuildMetrics() *BuildMetrics {
	return &BuildMetrics{
		DropReasons: make(map[string]int64),
	}
}

// RecordBuild records a build result in the metrics
func (bm *BuildMetrics) RecordBuild(result BuildResult) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.TotalBuilds++
	bm.TotalDuration += result.Duration

	if result.CacheHit {
		bm.CacheHits++
	}

	if result.Error != nil {
		bm.FailedBuilds++
	} else {
		bm.SuccessfulBuilds++
	}

	// Update average duration
	if bm.TotalBuilds > 0 {
		bm.AverageDuration = bm.TotalDuration / time.Duration(bm.TotalBuilds)
	}
}

// GetSnapshot returns a snapshot of current metrics
func (bm *BuildMetrics) GetSnapshot() BuildMetrics {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	// Copy the drop reasons map
	reasonsCopy := make(map[string]int64, len(bm.DropReasons))
	for k, v := range bm.DropReasons {
		reasonsCopy[k] = v
	}

	// Return a copy without the mutex to avoid lock copying issues
	return BuildMetrics{
		TotalBuilds:      bm.TotalBuilds,
		SuccessfulBuilds: bm.SuccessfulBuilds,
		FailedBuilds:     bm.FailedBuilds,
		CacheHits:        bm.CacheHits,
		AverageDuration:  bm.AverageDuration,
		TotalDuration:    bm.TotalDuration,
		DroppedTasks:     bm.DroppedTasks,
		DroppedResults:   bm.DroppedResults,
		DropReasons:      reasonsCopy,
		// mutex is intentionally omitted to prevent lock copying
	}
}

// Reset resets all metrics
func (bm *BuildMetrics) Reset() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.TotalBuilds = 0
	bm.SuccessfulBuilds = 0
	bm.FailedBuilds = 0
	bm.CacheHits = 0
	bm.AverageDuration = 0
	bm.TotalDuration = 0
	bm.DroppedTasks = 0
	bm.DroppedResults = 0
	bm.DropReasons = make(map[string]int64)

	// Reset parallel processing metrics
	atomic.StoreInt64(&bm.ParallelFileDiscoveries, 0)
	atomic.StoreInt64(&bm.FileDiscoveryDuration, 0)
	atomic.StoreInt64(&bm.ParallelProcessingTime, 0)
	atomic.StoreInt64(&bm.WorkerUtilization, 0)
	atomic.StoreInt64(&bm.BatchProcessingCount, 0)
	atomic.StoreInt64(&bm.BatchSize, 0)
	atomic.StoreInt32(&bm.ConcurrencyLevel, 0)
	atomic.StoreInt32(&bm.PeakConcurrency, 0)

	// Reset AST caching metrics
	atomic.StoreInt64(&bm.ASTCacheHits, 0)
	atomic.StoreInt64(&bm.ASTCacheMisses, 0)
	atomic.StoreInt64(&bm.ASTParseTime, 0)

	// Reset memory optimization metrics
	atomic.StoreInt64(&bm.PoolHits, 0)
	atomic.StoreInt64(&bm.PoolMisses, 0)
	atomic.StoreInt64(&bm.MemoryReused, 0)
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (bm *BuildMetrics) GetCacheHitRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.CacheHits) / float64(bm.TotalBuilds) * 100.0
}

// GetSuccessRate returns the success rate as a percentage
func (bm *BuildMetrics) GetSuccessRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.SuccessfulBuilds) / float64(bm.TotalBuilds) * 100.0
}

// RecordDroppedTask records when a task is dropped due to queue full
func (bm *BuildMetrics) RecordDroppedTask(componentName, reason string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.DroppedTasks++
	bm.DropReasons[reason]++
}

// RecordDroppedResult records when a result is dropped due to queue full
func (bm *BuildMetrics) RecordDroppedResult(componentName, reason string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.DroppedResults++
	bm.DropReasons[reason]++
}

// GetQueueHealthStatus returns queue health information
func (bm *BuildMetrics) GetQueueHealthStatus() (droppedTasks, droppedResults int64, dropReasons map[string]int64) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	// Copy the map to avoid race conditions
	reasonsCopy := make(map[string]int64, len(bm.DropReasons))
	for k, v := range bm.DropReasons {
		reasonsCopy[k] = v
	}

	return bm.DroppedTasks, bm.DroppedResults, reasonsCopy
}

// Interface compliance methods for interfaces.BuildMetrics

// GetBuildCount returns the total number of builds
func (bm *BuildMetrics) GetBuildCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.TotalBuilds
}

// GetSuccessCount returns the number of successful builds
func (bm *BuildMetrics) GetSuccessCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.SuccessfulBuilds
}

// GetFailureCount returns the number of failed builds
func (bm *BuildMetrics) GetFailureCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.FailedBuilds
}

// GetAverageDuration returns the average build duration
func (bm *BuildMetrics) GetAverageDuration() time.Duration {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	
	if bm.TotalBuilds == 0 {
		return 0
	}
	
	return time.Duration(int64(bm.TotalDuration) / bm.TotalBuilds)
}

// Parallel processing metrics methods

// RecordFileDiscovery records file discovery performance
func (bm *BuildMetrics) RecordFileDiscovery(duration time.Duration, filesFound int64) {
	atomic.AddInt64(&bm.ParallelFileDiscoveries, 1)
	atomic.AddInt64(&bm.FileDiscoveryDuration, int64(duration))
}

// RecordParallelProcessing records parallel processing performance
func (bm *BuildMetrics) RecordParallelProcessing(duration time.Duration, concurrency int32) {
	atomic.AddInt64(&bm.ParallelProcessingTime, int64(duration))
	
	// Update peak concurrency atomically
	for {
		current := atomic.LoadInt32(&bm.PeakConcurrency)
		if concurrency <= current {
			break
		}
		if atomic.CompareAndSwapInt32(&bm.PeakConcurrency, current, concurrency) {
			break
		}
	}
	
	atomic.StoreInt32(&bm.ConcurrencyLevel, concurrency)
}

// RecordBatchProcessing records batch processing metrics
func (bm *BuildMetrics) RecordBatchProcessing(batchSize int64, duration time.Duration) {
	atomic.AddInt64(&bm.BatchProcessingCount, 1)
	atomic.AddInt64(&bm.BatchSize, batchSize)
}

// AST caching metrics methods

// RecordASTCacheHit records an AST cache hit
func (bm *BuildMetrics) RecordASTCacheHit() {
	atomic.AddInt64(&bm.ASTCacheHits, 1)
}

// RecordASTCacheMiss records an AST cache miss with parse time
func (bm *BuildMetrics) RecordASTCacheMiss(parseTime time.Duration) {
	atomic.AddInt64(&bm.ASTCacheMisses, 1)
	atomic.AddInt64(&bm.ASTParseTime, int64(parseTime))
}

// Memory optimization metrics methods

// RecordPoolHit records object pool reuse
func (bm *BuildMetrics) RecordPoolHit(bytesReused int64) {
	atomic.AddInt64(&bm.PoolHits, 1)
	atomic.AddInt64(&bm.MemoryReused, bytesReused)
}

// RecordPoolMiss records object pool allocation
func (bm *BuildMetrics) RecordPoolMiss() {
	atomic.AddInt64(&bm.PoolMisses, 1)
}

// Performance analysis methods

// GetAverageFileDiscoveryTime returns average file discovery time
func (bm *BuildMetrics) GetAverageFileDiscoveryTime() time.Duration {
	discoveries := atomic.LoadInt64(&bm.ParallelFileDiscoveries)
	if discoveries == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&bm.FileDiscoveryDuration)
	return time.Duration(totalTime / discoveries)
}

// GetAverageParallelProcessingTime returns average parallel processing time
func (bm *BuildMetrics) GetAverageParallelProcessingTime() time.Duration {
	batches := atomic.LoadInt64(&bm.BatchProcessingCount)
	if batches == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&bm.ParallelProcessingTime)
	return time.Duration(totalTime / batches)
}

// GetASTCacheHitRate returns AST cache hit rate as percentage
func (bm *BuildMetrics) GetASTCacheHitRate() float64 {
	hits := atomic.LoadInt64(&bm.ASTCacheHits)
	misses := atomic.LoadInt64(&bm.ASTCacheMisses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100.0
}

// GetPoolEfficiency returns memory pool efficiency as percentage
func (bm *BuildMetrics) GetPoolEfficiency() float64 {
	hits := atomic.LoadInt64(&bm.PoolHits)
	misses := atomic.LoadInt64(&bm.PoolMisses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100.0
}

// GetConcurrencyStats returns concurrency statistics
func (bm *BuildMetrics) GetConcurrencyStats() (current int32, peak int32) {
	return atomic.LoadInt32(&bm.ConcurrencyLevel), atomic.LoadInt32(&bm.PeakConcurrency)
}

// GetAverageBatchSize returns average batch size
func (bm *BuildMetrics) GetAverageBatchSize() float64 {
	batches := atomic.LoadInt64(&bm.BatchProcessingCount)
	if batches == 0 {
		return 0.0
	}
	totalSize := atomic.LoadInt64(&bm.BatchSize)
	return float64(totalSize) / float64(batches)
}

// GetPerformanceSummary returns a comprehensive performance summary
func (bm *BuildMetrics) GetPerformanceSummary() map[string]interface{} {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	current, peak := bm.GetConcurrencyStats()
	
	return map[string]interface{}{
		"build_performance": map[string]interface{}{
			"total_builds":      bm.TotalBuilds,
			"success_rate":      bm.GetSuccessRate(),
			"cache_hit_rate":    bm.GetCacheHitRate(),
			"average_duration":  bm.AverageDuration,
		},
		"parallel_processing": map[string]interface{}{
			"file_discoveries":           atomic.LoadInt64(&bm.ParallelFileDiscoveries),
			"avg_discovery_time":         bm.GetAverageFileDiscoveryTime(),
			"avg_processing_time":        bm.GetAverageParallelProcessingTime(),
			"current_concurrency":        current,
			"peak_concurrency":          peak,
			"avg_batch_size":            bm.GetAverageBatchSize(),
			"batch_count":               atomic.LoadInt64(&bm.BatchProcessingCount),
		},
		"caching_performance": map[string]interface{}{
			"ast_cache_hit_rate":    bm.GetASTCacheHitRate(),
			"ast_cache_hits":        atomic.LoadInt64(&bm.ASTCacheHits),
			"ast_cache_misses":      atomic.LoadInt64(&bm.ASTCacheMisses),
			"pool_efficiency":       bm.GetPoolEfficiency(),
			"memory_reused_bytes":   atomic.LoadInt64(&bm.MemoryReused),
		},
		"queue_health": map[string]interface{}{
			"dropped_tasks":   bm.DroppedTasks,
			"dropped_results": bm.DroppedResults,
			"drop_reasons":    bm.DropReasons,
		},
	}
}
