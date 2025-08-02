// Package build provides build metrics tracking and performance monitoring.
package build

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
)

// BuildMetrics tracks build performance and queue health.
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
	WorkerUtilization       int64 // Average worker utilization percentage
	BatchProcessingCount    int64 // Number of batch operations
	BatchSize               int64 // Average batch size
	ConcurrencyLevel        int32 // Current concurrency level
	PeakConcurrency         int32 // Peak concurrency achieved

	// AST caching metrics
	ASTCacheHits   int64 // AST cache hits
	ASTCacheMisses int64 // AST cache misses
	ASTParseTime   int64 // Total AST parsing time (nanoseconds)

	// Memory optimization metrics
	PoolHits     int64 // Object pool cache hits
	PoolMisses   int64 // Object pool cache misses
	MemoryReused int64 // Bytes of memory reused from pools

	// Enhanced timing and performance metrics
	buildTimes           []time.Duration // Historical build times for analysis
	timingMutex          sync.RWMutex    // Separate mutex for timing data
	maxTimingHistory     int             // Maximum timing entries to keep
	performanceBaseline  time.Duration   // Performance baseline for regression detection
	lastPerformanceCheck time.Time       // Last performance analysis timestamp

	// Phase-specific timing metrics
	scanPhaseTime     int64 // Total time spent in scanning phase (nanoseconds)
	compilePhaseTime  int64 // Total time spent in compilation phase (nanoseconds)
	validatePhaseTime int64 // Total time spent in validation phase (nanoseconds)

	// Real-time performance tracking
	currentBuilds      map[string]*BuildTiming // Currently active builds
	currentBuildsMutex sync.RWMutex            // Mutex for current builds map

	// Performance trend analysis
	performanceTrends []PerformanceTrend // Historical performance trends
	trendMutex        sync.RWMutex       // Mutex for trend data

	mutex sync.RWMutex
}

// BuildTiming tracks timing information for individual builds.
type BuildTiming struct {
	BuildID       string        `json:"build_id"`
	ComponentName string        `json:"component_name"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	Duration      time.Duration `json:"duration"`

	// Phase timings
	ScanTime     time.Duration `json:"scan_time"`
	CompileTime  time.Duration `json:"compile_time"`
	ValidateTime time.Duration `json:"validate_time"`

	// Context information
	CacheHit bool   `json:"cache_hit"`
	WorkerID int    `json:"worker_id"`
	Success  bool   `json:"success"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// PerformanceTrend represents performance analysis over time.
type PerformanceTrend struct {
	Timestamp        time.Time     `json:"timestamp"`
	WindowStart      time.Time     `json:"window_start"`
	WindowEnd        time.Time     `json:"window_end"`
	BuildCount       int64         `json:"build_count"`
	AverageTime      time.Duration `json:"average_time"`
	MedianTime       time.Duration `json:"median_time"`
	P95Time          time.Duration `json:"p95_time"`
	SuccessRate      float64       `json:"success_rate"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	TrendDirection   string        `json:"trend_direction"`   // "improving", "degrading", "stable"
	PerformanceScore float64       `json:"performance_score"` // 0-100 overall score
}

// TimingStats provides statistical analysis of build times.
type TimingStats struct {
	Count    int64         `json:"count"`
	Min      time.Duration `json:"min"`
	Max      time.Duration `json:"max"`
	Mean     time.Duration `json:"mean"`
	Median   time.Duration `json:"median"`
	P90      time.Duration `json:"p90"`
	P95      time.Duration `json:"p95"`
	P99      time.Duration `json:"p99"`
	StdDev   time.Duration `json:"std_dev"`
	Variance time.Duration `json:"variance"`
}

// Ensure BuildMetrics implements the interfaces.BuildMetrics interface.
var _ interfaces.BuildMetrics = (*BuildMetrics)(nil)

// NewBuildMetrics creates a new build metrics tracker.
func NewBuildMetrics() *BuildMetrics {
	return &BuildMetrics{
		DropReasons:          make(map[string]int64),
		buildTimes:           make([]time.Duration, 0, 1000),
		maxTimingHistory:     1000,
		performanceBaseline:  5 * time.Second, // Default 5s baseline
		lastPerformanceCheck: time.Now(),
		currentBuilds:        make(map[string]*BuildTiming),
		performanceTrends:    make([]PerformanceTrend, 0, 100),
	}
}

// RecordBuild records a build result in the metrics.
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

	// Record timing history
	bm.recordBuildTiming(result.Duration)

	// Log performance warning if build exceeds baseline
	if result.Duration > bm.performanceBaseline {
		log.Printf("Build performance warning: %v exceeds baseline %v",
			result.Duration, bm.performanceBaseline)
	}
}

// recordBuildTiming adds a build time to the timing history.
func (bm *BuildMetrics) recordBuildTiming(duration time.Duration) {
	bm.timingMutex.Lock()
	defer bm.timingMutex.Unlock()

	bm.buildTimes = append(bm.buildTimes, duration)

	// Keep history within limits
	if len(bm.buildTimes) > bm.maxTimingHistory {
		// Remove oldest 10% when at capacity
		removeCount := bm.maxTimingHistory / 10
		copy(bm.buildTimes, bm.buildTimes[removeCount:])
		bm.buildTimes = bm.buildTimes[:len(bm.buildTimes)-removeCount]
	}
}

// GetSnapshot returns a snapshot of current metrics.
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

// Reset resets all metrics.
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

// GetCacheHitRate returns the cache hit rate as a percentage.
func (bm *BuildMetrics) GetCacheHitRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.CacheHits) / float64(bm.TotalBuilds) * 100.0
}

// GetSuccessRate returns the success rate as a percentage.
func (bm *BuildMetrics) GetSuccessRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.SuccessfulBuilds) / float64(bm.TotalBuilds) * 100.0
}

// RecordDroppedTask records when a task is dropped due to queue full.
func (bm *BuildMetrics) RecordDroppedTask(componentName, reason string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.DroppedTasks++
	bm.DropReasons[reason]++
}

// RecordDroppedResult records when a result is dropped due to queue full.
func (bm *BuildMetrics) RecordDroppedResult(componentName, reason string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.DroppedResults++
	bm.DropReasons[reason]++
}

// GetQueueHealthStatus returns queue health information.
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

// GetBuildCount returns the total number of builds.
func (bm *BuildMetrics) GetBuildCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	return bm.TotalBuilds
}

// GetSuccessCount returns the number of successful builds.
func (bm *BuildMetrics) GetSuccessCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	return bm.SuccessfulBuilds
}

// GetFailureCount returns the number of failed builds.
func (bm *BuildMetrics) GetFailureCount() int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	return bm.FailedBuilds
}

// GetAverageDuration returns the average build duration.
func (bm *BuildMetrics) GetAverageDuration() time.Duration {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0
	}

	return time.Duration(int64(bm.TotalDuration) / bm.TotalBuilds)
}

// Parallel processing metrics methods

// RecordFileDiscovery records file discovery performance.
func (bm *BuildMetrics) RecordFileDiscovery(duration time.Duration, filesFound int64) {
	atomic.AddInt64(&bm.ParallelFileDiscoveries, 1)
	atomic.AddInt64(&bm.FileDiscoveryDuration, int64(duration))
}

// RecordParallelProcessing records parallel processing performance.
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

// RecordBatchProcessing records batch processing metrics.
func (bm *BuildMetrics) RecordBatchProcessing(batchSize int64, duration time.Duration) {
	atomic.AddInt64(&bm.BatchProcessingCount, 1)
	atomic.AddInt64(&bm.BatchSize, batchSize)
}

// AST caching metrics methods

// RecordASTCacheHit records an AST cache hit.
func (bm *BuildMetrics) RecordASTCacheHit() {
	atomic.AddInt64(&bm.ASTCacheHits, 1)
}

// RecordASTCacheMiss records an AST cache miss with parse time.
func (bm *BuildMetrics) RecordASTCacheMiss(parseTime time.Duration) {
	atomic.AddInt64(&bm.ASTCacheMisses, 1)
	atomic.AddInt64(&bm.ASTParseTime, int64(parseTime))
}

// Memory optimization metrics methods

// RecordPoolHit records object pool reuse.
func (bm *BuildMetrics) RecordPoolHit(bytesReused int64) {
	atomic.AddInt64(&bm.PoolHits, 1)
	atomic.AddInt64(&bm.MemoryReused, bytesReused)
}

// RecordPoolMiss records object pool allocation.
func (bm *BuildMetrics) RecordPoolMiss() {
	atomic.AddInt64(&bm.PoolMisses, 1)
}

// Performance analysis methods

// GetAverageFileDiscoveryTime returns average file discovery time.
func (bm *BuildMetrics) GetAverageFileDiscoveryTime() time.Duration {
	discoveries := atomic.LoadInt64(&bm.ParallelFileDiscoveries)
	if discoveries == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&bm.FileDiscoveryDuration)

	return time.Duration(totalTime / discoveries)
}

// GetAverageParallelProcessingTime returns average parallel processing time.
func (bm *BuildMetrics) GetAverageParallelProcessingTime() time.Duration {
	batches := atomic.LoadInt64(&bm.BatchProcessingCount)
	if batches == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&bm.ParallelProcessingTime)

	return time.Duration(totalTime / batches)
}

// GetASTCacheHitRate returns AST cache hit rate as percentage.
func (bm *BuildMetrics) GetASTCacheHitRate() float64 {
	hits := atomic.LoadInt64(&bm.ASTCacheHits)
	misses := atomic.LoadInt64(&bm.ASTCacheMisses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// GetPoolEfficiency returns memory pool efficiency as percentage.
func (bm *BuildMetrics) GetPoolEfficiency() float64 {
	hits := atomic.LoadInt64(&bm.PoolHits)
	misses := atomic.LoadInt64(&bm.PoolMisses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// GetConcurrencyStats returns concurrency statistics.
func (bm *BuildMetrics) GetConcurrencyStats() (current int32, peak int32) {
	return atomic.LoadInt32(&bm.ConcurrencyLevel), atomic.LoadInt32(&bm.PeakConcurrency)
}

// GetAverageBatchSize returns average batch size.
func (bm *BuildMetrics) GetAverageBatchSize() float64 {
	batches := atomic.LoadInt64(&bm.BatchProcessingCount)
	if batches == 0 {
		return 0.0
	}
	totalSize := atomic.LoadInt64(&bm.BatchSize)

	return float64(totalSize) / float64(batches)
}

// GetPerformanceSummary returns a comprehensive performance summary.
func (bm *BuildMetrics) GetPerformanceSummary() map[string]interface{} {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	current, peak := bm.GetConcurrencyStats()
	timingStats := bm.GetTimingStatistics()

	return map[string]interface{}{
		"build_performance": map[string]interface{}{
			"total_builds":     bm.TotalBuilds,
			"success_rate":     bm.GetSuccessRate(),
			"cache_hit_rate":   bm.GetCacheHitRate(),
			"average_duration": bm.AverageDuration,
		},
		"parallel_processing": map[string]interface{}{
			"file_discoveries":    atomic.LoadInt64(&bm.ParallelFileDiscoveries),
			"avg_discovery_time":  bm.GetAverageFileDiscoveryTime(),
			"avg_processing_time": bm.GetAverageParallelProcessingTime(),
			"current_concurrency": current,
			"peak_concurrency":    peak,
			"avg_batch_size":      bm.GetAverageBatchSize(),
			"batch_count":         atomic.LoadInt64(&bm.BatchProcessingCount),
		},
		"caching_performance": map[string]interface{}{
			"ast_cache_hit_rate":  bm.GetASTCacheHitRate(),
			"ast_cache_hits":      atomic.LoadInt64(&bm.ASTCacheHits),
			"ast_cache_misses":    atomic.LoadInt64(&bm.ASTCacheMisses),
			"pool_efficiency":     bm.GetPoolEfficiency(),
			"memory_reused_bytes": atomic.LoadInt64(&bm.MemoryReused),
		},
		"queue_health": map[string]interface{}{
			"dropped_tasks":   bm.DroppedTasks,
			"dropped_results": bm.DroppedResults,
			"drop_reasons":    bm.DropReasons,
		},
		"timing_analysis": timingStats,
		"performance_baseline": map[string]interface{}{
			"baseline_threshold": bm.performanceBaseline,
			"last_check":         bm.lastPerformanceCheck,
		},
	}
}

// Enhanced timing and performance methods

// StartBuildTiming starts timing tracking for a specific build.
func (bm *BuildMetrics) StartBuildTiming(buildID, componentName string, workerID int) {
	bm.currentBuildsMutex.Lock()
	defer bm.currentBuildsMutex.Unlock()

	bm.currentBuilds[buildID] = &BuildTiming{
		BuildID:       buildID,
		ComponentName: componentName,
		StartTime:     time.Now(),
		WorkerID:      workerID,
	}
}

// RecordBuildPhase records timing for a specific build phase.
func (bm *BuildMetrics) RecordBuildPhase(buildID string, phase string, duration time.Duration) {
	bm.currentBuildsMutex.Lock()
	defer bm.currentBuildsMutex.Unlock()

	if buildTiming, exists := bm.currentBuilds[buildID]; exists {
		switch phase {
		case "scanning":
			buildTiming.ScanTime = duration
			atomic.AddInt64(&bm.scanPhaseTime, int64(duration))
		case "compiling":
			buildTiming.CompileTime = duration
			atomic.AddInt64(&bm.compilePhaseTime, int64(duration))
		case "validating":
			buildTiming.ValidateTime = duration
			atomic.AddInt64(&bm.validatePhaseTime, int64(duration))
		}
	}
}

// FinishBuildTiming completes timing tracking for a build.
func (bm *BuildMetrics) FinishBuildTiming(buildID string, success bool, cacheHit bool, errorMsg string) *BuildTiming {
	bm.currentBuildsMutex.Lock()
	defer bm.currentBuildsMutex.Unlock()

	buildTiming, exists := bm.currentBuilds[buildID]
	if !exists {
		return nil
	}

	now := time.Now()
	buildTiming.EndTime = &now
	buildTiming.Duration = now.Sub(buildTiming.StartTime)
	buildTiming.Success = success
	buildTiming.CacheHit = cacheHit
	buildTiming.ErrorMsg = errorMsg

	// Remove from active builds
	delete(bm.currentBuilds, buildID)

	return buildTiming
}

// GetTimingStatistics returns detailed timing statistics.
func (bm *BuildMetrics) GetTimingStatistics() TimingStats {
	bm.timingMutex.RLock()
	defer bm.timingMutex.RUnlock()

	if len(bm.buildTimes) == 0 {
		return TimingStats{}
	}

	// Create a sorted copy for percentile calculations
	sortedTimes := make([]time.Duration, len(bm.buildTimes))
	copy(sortedTimes, bm.buildTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})

	stats := TimingStats{
		Count:  int64(len(sortedTimes)),
		Min:    sortedTimes[0],
		Max:    sortedTimes[len(sortedTimes)-1],
		Median: sortedTimes[len(sortedTimes)/2],
	}

	// Calculate mean
	var total time.Duration
	for _, t := range sortedTimes {
		total += t
	}
	stats.Mean = total / time.Duration(len(sortedTimes))

	// Calculate percentiles
	if len(sortedTimes) >= 10 {
		stats.P90 = sortedTimes[int(float64(len(sortedTimes))*0.90)]
		stats.P95 = sortedTimes[int(float64(len(sortedTimes))*0.95)]
		if len(sortedTimes) >= 100 {
			stats.P99 = sortedTimes[int(float64(len(sortedTimes))*0.99)]
		}
	}

	// Calculate standard deviation and variance
	var sumSquaredDiff float64
	meanNanos := float64(stats.Mean.Nanoseconds())
	for _, t := range sortedTimes {
		diff := float64(t.Nanoseconds()) - meanNanos
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / float64(len(sortedTimes))
	stats.Variance = time.Duration(int64(variance))
	stats.StdDev = time.Duration(int64(variance))

	return stats
}

// GetPhaseTimingBreakdown returns timing breakdown by build phases.
func (bm *BuildMetrics) GetPhaseTimingBreakdown() map[string]interface{} {
	scanTime := time.Duration(atomic.LoadInt64(&bm.scanPhaseTime))
	compileTime := time.Duration(atomic.LoadInt64(&bm.compilePhaseTime))
	validateTime := time.Duration(atomic.LoadInt64(&bm.validatePhaseTime))
	totalPhaseTime := scanTime + compileTime + validateTime

	breakdown := map[string]interface{}{
		"scan_phase": map[string]interface{}{
			"total_time": scanTime,
			"percentage": 0.0,
		},
		"compile_phase": map[string]interface{}{
			"total_time": compileTime,
			"percentage": 0.0,
		},
		"validate_phase": map[string]interface{}{
			"total_time": validateTime,
			"percentage": 0.0,
		},
		"total_phase_time": totalPhaseTime,
	}

	if totalPhaseTime > 0 {
		breakdown["scan_phase"].(map[string]interface{})["percentage"] =
			float64(scanTime.Nanoseconds()) / float64(totalPhaseTime.Nanoseconds()) * 100
		breakdown["compile_phase"].(map[string]interface{})["percentage"] =
			float64(compileTime.Nanoseconds()) / float64(totalPhaseTime.Nanoseconds()) * 100
		breakdown["validate_phase"].(map[string]interface{})["percentage"] =
			float64(validateTime.Nanoseconds()) / float64(totalPhaseTime.Nanoseconds()) * 100
	}

	return breakdown
}

// AnalyzePerformanceTrends analyzes performance trends over time.
func (bm *BuildMetrics) AnalyzePerformanceTrends(windowDuration time.Duration) PerformanceTrend {
	bm.timingMutex.RLock()
	defer bm.timingMutex.RUnlock()

	now := time.Now()
	windowStart := now.Add(-windowDuration)

	// For this implementation, we'll analyze the recent build times
	// In a full implementation, we'd filter by timestamp
	recentTimes := bm.buildTimes
	if len(recentTimes) > 100 {
		recentTimes = bm.buildTimes[len(bm.buildTimes)-100:]
	}

	if len(recentTimes) == 0 {
		return PerformanceTrend{
			Timestamp:        now,
			WindowStart:      windowStart,
			WindowEnd:        now,
			TrendDirection:   "stable",
			PerformanceScore: 100.0,
		}
	}

	// Calculate statistics
	var total time.Duration
	successCount := int64(len(recentTimes)) // Simplified - actual implementation would track success/failure

	sortedTimes := make([]time.Duration, len(recentTimes))
	copy(sortedTimes, recentTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})

	for _, t := range recentTimes {
		total += t
	}

	trend := PerformanceTrend{
		Timestamp:    now,
		WindowStart:  windowStart,
		WindowEnd:    now,
		BuildCount:   int64(len(recentTimes)),
		AverageTime:  total / time.Duration(len(recentTimes)),
		MedianTime:   sortedTimes[len(sortedTimes)/2],
		SuccessRate:  float64(successCount) / float64(len(recentTimes)) * 100,
		CacheHitRate: bm.GetCacheHitRate(), // Use overall cache hit rate
	}

	if len(sortedTimes) > 10 {
		trend.P95Time = sortedTimes[int(float64(len(sortedTimes))*0.95)]
	}

	// Determine trend direction (simplified)
	switch {
	case trend.AverageTime < bm.performanceBaseline:
		trend.TrendDirection = "improving"
		trend.PerformanceScore = 85.0
	case trend.AverageTime > bm.performanceBaseline*2:
		trend.TrendDirection = "degrading"
		trend.PerformanceScore = 45.0
	default:
		trend.TrendDirection = "stable"
		trend.PerformanceScore = 70.0
	}

	// Store trend in history
	bm.trendMutex.Lock()
	bm.performanceTrends = append(bm.performanceTrends, trend)
	if len(bm.performanceTrends) > 100 {
		bm.performanceTrends = bm.performanceTrends[1:]
	}
	bm.trendMutex.Unlock()

	return trend
}

// GetPerformanceTrendHistory returns historical performance trends.
func (bm *BuildMetrics) GetPerformanceTrendHistory(limit int) []PerformanceTrend {
	bm.trendMutex.RLock()
	defer bm.trendMutex.RUnlock()

	trends := make([]PerformanceTrend, len(bm.performanceTrends))
	copy(trends, bm.performanceTrends)

	if limit > 0 && len(trends) > limit {
		trends = trends[len(trends)-limit:]
	}

	return trends
}

// SetPerformanceBaseline updates the performance baseline threshold.
func (bm *BuildMetrics) SetPerformanceBaseline(baseline time.Duration) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.performanceBaseline = baseline
	bm.lastPerformanceCheck = time.Now()

	log.Printf("Performance baseline updated to %v", baseline)
}

// GetCurrentBuildTimings returns currently active build timings.
func (bm *BuildMetrics) GetCurrentBuildTimings() map[string]*BuildTiming {
	bm.currentBuildsMutex.RLock()
	defer bm.currentBuildsMutex.RUnlock()

	timings := make(map[string]*BuildTiming, len(bm.currentBuilds))
	for k, v := range bm.currentBuilds {
		// Create a copy to avoid race conditions
		timing := *v
		timings[k] = &timing
	}

	return timings
}

// CreateWebSocketTimingMessage creates a WebSocket message for timing updates.
func (bm *BuildMetrics) CreateWebSocketTimingMessage() ([]byte, error) {
	stats := bm.GetTimingStatistics()
	phaseBreakdown := bm.GetPhaseTimingBreakdown()
	currentBuilds := bm.GetCurrentBuildTimings()

	message := map[string]interface{}{
		"type": "timing_metrics",
		"data": map[string]interface{}{
			"statistics":      stats,
			"phase_breakdown": phaseBreakdown,
			"current_builds":  currentBuilds,
			"timestamp":       time.Now(),
		},
	}

	return json.Marshal(message)
}

// DetectPerformanceRegressions analyzes recent performance for regressions.
func (bm *BuildMetrics) DetectPerformanceRegressions() []string {
	stats := bm.GetTimingStatistics()
	regressions := make([]string, 0)

	// Check if mean exceeds baseline significantly
	if stats.Mean > bm.performanceBaseline*2 {
		regressions = append(regressions,
			fmt.Sprintf("Average build time (%.2fs) is 2x slower than baseline (%.2fs)",
				stats.Mean.Seconds(), bm.performanceBaseline.Seconds()))
	}

	// Check for high variance
	if stats.Count > 10 && stats.StdDev > stats.Mean/2 {
		regressions = append(regressions,
			fmt.Sprintf("High build time variance detected (stddev: %.2fs, mean: %.2fs)",
				stats.StdDev.Seconds(), stats.Mean.Seconds()))
	}

	// Check P95 times
	if stats.P95 > bm.performanceBaseline*3 {
		regressions = append(regressions,
			fmt.Sprintf("P95 build time (%.2fs) is 3x slower than baseline",
				stats.P95.Seconds()))
	}

	return regressions
}
