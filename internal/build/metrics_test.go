package build

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildMetrics_InterfaceCompliance validates that BuildMetrics implements interfaces.BuildMetrics
func TestBuildMetrics_InterfaceCompliance(t *testing.T) {
	metrics := NewBuildMetrics()
	
	// Verify interface compliance at compile time and runtime
	var _ interfaces.BuildMetrics = metrics
	
	// Test all interface methods are callable
	assert.Equal(t, int64(0), metrics.GetBuildCount())
	assert.Equal(t, int64(0), metrics.GetSuccessCount())
	assert.Equal(t, int64(0), metrics.GetFailureCount())
	assert.Equal(t, time.Duration(0), metrics.GetAverageDuration())
	assert.Equal(t, 0.0, metrics.GetCacheHitRate())
	assert.Equal(t, 0.0, metrics.GetSuccessRate())
}

// TestNewBuildMetrics validates proper initialization
func TestNewBuildMetrics(t *testing.T) {
	metrics := NewBuildMetrics()
	
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.DropReasons)
	assert.Equal(t, int64(0), metrics.TotalBuilds)
	assert.Equal(t, int64(0), metrics.SuccessfulBuilds)
	assert.Equal(t, int64(0), metrics.FailedBuilds)
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, time.Duration(0), metrics.AverageDuration)
	assert.Equal(t, time.Duration(0), metrics.TotalDuration)
	
	// Verify atomic counters are initialized to zero
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.ParallelFileDiscoveries))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.ASTCacheHits))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.PoolHits))
	assert.Equal(t, int32(0), atomic.LoadInt32(&metrics.ConcurrencyLevel))
	assert.Equal(t, int32(0), atomic.LoadInt32(&metrics.PeakConcurrency))
}

// TestBuildMetrics_RecordBuild validates build recording functionality
func TestBuildMetrics_RecordBuild(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("successful build without cache", func(t *testing.T) {
		result := BuildResult{
			Component: &types.ComponentInfo{Name: "TestComponent"},
			Duration:  100 * time.Millisecond,
			Error:     nil,
			CacheHit:  false,
		}
		
		metrics.RecordBuild(result)
		
		assert.Equal(t, int64(1), metrics.GetBuildCount())
		assert.Equal(t, int64(1), metrics.GetSuccessCount())
		assert.Equal(t, int64(0), metrics.GetFailureCount())
		assert.Equal(t, int64(0), metrics.CacheHits)
		assert.Equal(t, 100*time.Millisecond, metrics.AverageDuration)
		assert.Equal(t, 100*time.Millisecond, metrics.TotalDuration)
	})
	
	t.Run("failed build with cache hit", func(t *testing.T) {
		result := BuildResult{
			Component: &types.ComponentInfo{Name: "FailedComponent"},
			Duration:  50 * time.Millisecond,
			Error:     errors.NewBuildError("BUILD_FAILED", "compilation failed", nil),
			CacheHit:  true,
		}
		
		metrics.RecordBuild(result)
		
		assert.Equal(t, int64(2), metrics.GetBuildCount())
		assert.Equal(t, int64(1), metrics.GetSuccessCount())
		assert.Equal(t, int64(1), metrics.GetFailureCount())
		assert.Equal(t, int64(1), metrics.CacheHits)
		
		// Average duration should be (100ms + 50ms) / 2 = 75ms
		assert.Equal(t, 75*time.Millisecond, metrics.AverageDuration)
		assert.Equal(t, 150*time.Millisecond, metrics.TotalDuration)
	})
}

// TestBuildMetrics_RateCalculations validates rate calculation methods
func TestBuildMetrics_RateCalculations(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("zero builds", func(t *testing.T) {
		assert.Equal(t, 0.0, metrics.GetCacheHitRate())
		assert.Equal(t, 0.0, metrics.GetSuccessRate())
	})
	
	t.Run("mixed builds", func(t *testing.T) {
		// 4 total builds: 3 successful (2 cache hits), 1 failed (1 cache hit)
		builds := []BuildResult{
			{Duration: 100 * time.Millisecond, Error: nil, CacheHit: true},      // success + cache
			{Duration: 150 * time.Millisecond, Error: nil, CacheHit: true},      // success + cache  
			{Duration: 200 * time.Millisecond, Error: nil, CacheHit: false},     // success, no cache
			{Duration: 50 * time.Millisecond, Error: errors.NewBuildError("BUILD_FAILED", "fail", nil), CacheHit: true}, // failed + cache
		}
		
		for _, build := range builds {
			metrics.RecordBuild(build)
		}
		
		// Cache hit rate: 3 cache hits / 4 total builds = 75%
		assert.Equal(t, 75.0, metrics.GetCacheHitRate())
		
		// Success rate: 3 successful / 4 total builds = 75%
		assert.Equal(t, 75.0, metrics.GetSuccessRate())
		
		// Average duration: (100 + 150 + 200 + 50) / 4 = 125ms
		assert.Equal(t, 125*time.Millisecond, metrics.GetAverageDuration())
	})
}

// TestBuildMetrics_DroppedTasks validates task/result dropping functionality
func TestBuildMetrics_DroppedTasks(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("record dropped tasks", func(t *testing.T) {
		metrics.RecordDroppedTask("Component1", "queue_full")
		metrics.RecordDroppedTask("Component2", "queue_full")
		metrics.RecordDroppedTask("Component3", "timeout")
		
		droppedTasks, droppedResults, reasons := metrics.GetQueueHealthStatus()
		
		assert.Equal(t, int64(3), droppedTasks)
		assert.Equal(t, int64(0), droppedResults)
		assert.Equal(t, int64(2), reasons["queue_full"])
		assert.Equal(t, int64(1), reasons["timeout"])
	})
	
	t.Run("record dropped results", func(t *testing.T) {
		metrics.RecordDroppedResult("Component4", "channel_closed")
		metrics.RecordDroppedResult("Component5", "queue_full")
		
		droppedTasks, droppedResults, reasons := metrics.GetQueueHealthStatus()
		
		assert.Equal(t, int64(3), droppedTasks)
		assert.Equal(t, int64(2), droppedResults)
		assert.Equal(t, int64(3), reasons["queue_full"]) // 2 from tasks + 1 from results
		assert.Equal(t, int64(1), reasons["timeout"])
		assert.Equal(t, int64(1), reasons["channel_closed"])
	})
}

// TestBuildMetrics_ParallelProcessing validates parallel processing metrics
func TestBuildMetrics_ParallelProcessing(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("file discovery metrics", func(t *testing.T) {
		metrics.RecordFileDiscovery(100*time.Millisecond, 10)
		metrics.RecordFileDiscovery(200*time.Millisecond, 20)
		metrics.RecordFileDiscovery(50*time.Millisecond, 5)
		
		assert.Equal(t, int64(3), atomic.LoadInt64(&metrics.ParallelFileDiscoveries))
		
		// Average discovery time: (100 + 200 + 50) / 3 = 116.67ms (rounded down)
		avgTime := metrics.GetAverageFileDiscoveryTime()
		assert.True(t, avgTime >= 116*time.Millisecond && avgTime <= 117*time.Millisecond)
	})
	
	t.Run("parallel processing and concurrency", func(t *testing.T) {
		metrics.RecordParallelProcessing(500*time.Millisecond, 4)
		metrics.RecordParallelProcessing(300*time.Millisecond, 6)
		metrics.RecordParallelProcessing(200*time.Millisecond, 2)
		
		current, peak := metrics.GetConcurrencyStats()
		assert.Equal(t, int32(2), current) // Last recorded
		assert.Equal(t, int32(6), peak)    // Highest recorded
	})
	
	t.Run("batch processing", func(t *testing.T) {
		metrics.RecordBatchProcessing(10, 100*time.Millisecond)
		metrics.RecordBatchProcessing(20, 200*time.Millisecond)
		metrics.RecordBatchProcessing(15, 150*time.Millisecond)
		
		assert.Equal(t, int64(3), atomic.LoadInt64(&metrics.BatchProcessingCount))
		
		// Average batch size: (10 + 20 + 15) / 3 = 15.0
		assert.Equal(t, 15.0, metrics.GetAverageBatchSize())
	})
}

// TestBuildMetrics_ASTCaching validates AST caching metrics
func TestBuildMetrics_ASTCaching(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("AST cache operations", func(t *testing.T) {
		// Record cache hits and misses
		metrics.RecordASTCacheHit()
		metrics.RecordASTCacheHit()
		metrics.RecordASTCacheMiss(50 * time.Millisecond)
		metrics.RecordASTCacheHit()
		metrics.RecordASTCacheMiss(75 * time.Millisecond)
		
		assert.Equal(t, int64(3), atomic.LoadInt64(&metrics.ASTCacheHits))
		assert.Equal(t, int64(2), atomic.LoadInt64(&metrics.ASTCacheMisses))
		
		// Cache hit rate: 3 hits / (3 hits + 2 misses) = 60%
		assert.Equal(t, 60.0, metrics.GetASTCacheHitRate())
	})
	
	t.Run("zero cache operations", func(t *testing.T) {
		freshMetrics := NewBuildMetrics()
		assert.Equal(t, 0.0, freshMetrics.GetASTCacheHitRate())
	})
}

// TestBuildMetrics_MemoryPooling validates memory pool metrics
func TestBuildMetrics_MemoryPooling(t *testing.T) {
	metrics := NewBuildMetrics()
	
	t.Run("pool operations", func(t *testing.T) {
		metrics.RecordPoolHit(1024)  // 1KB reused
		metrics.RecordPoolHit(2048)  // 2KB reused
		metrics.RecordPoolMiss()
		metrics.RecordPoolHit(512)   // 512B reused
		metrics.RecordPoolMiss()
		
		assert.Equal(t, int64(3), atomic.LoadInt64(&metrics.PoolHits))
		assert.Equal(t, int64(2), atomic.LoadInt64(&metrics.PoolMisses))
		assert.Equal(t, int64(3584), atomic.LoadInt64(&metrics.MemoryReused)) // 1024 + 2048 + 512
		
		// Pool efficiency: 3 hits / (3 hits + 2 misses) = 60%
		assert.Equal(t, 60.0, metrics.GetPoolEfficiency())
	})
	
	t.Run("zero pool operations", func(t *testing.T) {
		freshMetrics := NewBuildMetrics()
		assert.Equal(t, 0.0, freshMetrics.GetPoolEfficiency())
	})
}

// TestBuildMetrics_GetSnapshot validates snapshot functionality
func TestBuildMetrics_GetSnapshot(t *testing.T) {
	metrics := NewBuildMetrics()
	
	// Populate metrics with data
	result := BuildResult{
		Duration: 100 * time.Millisecond,
		Error:    nil,
		CacheHit: true,
	}
	metrics.RecordBuild(result)
	metrics.RecordDroppedTask("Component1", "queue_full")
	
	snapshot := metrics.GetSnapshot()
	
	// Verify snapshot contains correct data
	assert.Equal(t, int64(1), snapshot.TotalBuilds)
	assert.Equal(t, int64(1), snapshot.SuccessfulBuilds)
	assert.Equal(t, int64(0), snapshot.FailedBuilds)
	assert.Equal(t, int64(1), snapshot.CacheHits)
	assert.Equal(t, 100*time.Millisecond, snapshot.AverageDuration)
	assert.Equal(t, int64(1), snapshot.DroppedTasks)
	assert.Equal(t, int64(1), snapshot.DropReasons["queue_full"])
	
	// Verify snapshot is independent (modifying original doesn't affect snapshot)
	metrics.RecordBuild(result)
	assert.Equal(t, int64(1), snapshot.TotalBuilds) // Snapshot unchanged
	assert.Equal(t, int64(2), metrics.GetBuildCount()) // Original updated
}

// TestBuildMetrics_Reset validates reset functionality
func TestBuildMetrics_Reset(t *testing.T) {
	metrics := NewBuildMetrics()
	
	// Populate with various metrics
	result := BuildResult{Duration: 100 * time.Millisecond, Error: nil, CacheHit: true}
	metrics.RecordBuild(result)
	metrics.RecordDroppedTask("Component1", "queue_full")
	metrics.RecordFileDiscovery(50*time.Millisecond, 10)
	metrics.RecordASTCacheHit()
	metrics.RecordPoolHit(1024)
	metrics.RecordParallelProcessing(200*time.Millisecond, 4)
	
	// Verify data is present
	assert.Equal(t, int64(1), metrics.GetBuildCount())
	assert.Equal(t, int64(1), atomic.LoadInt64(&metrics.ParallelFileDiscoveries))
	assert.Equal(t, int64(1), atomic.LoadInt64(&metrics.ASTCacheHits))
	assert.Equal(t, int64(1), atomic.LoadInt64(&metrics.PoolHits))
	
	// Reset all metrics
	metrics.Reset()
	
	// Verify everything is reset to zero
	assert.Equal(t, int64(0), metrics.GetBuildCount())
	assert.Equal(t, int64(0), metrics.GetSuccessCount())
	assert.Equal(t, int64(0), metrics.GetFailureCount())
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, time.Duration(0), metrics.AverageDuration)
	assert.Equal(t, time.Duration(0), metrics.TotalDuration)
	assert.Equal(t, int64(0), metrics.DroppedTasks)
	assert.Equal(t, int64(0), metrics.DroppedResults)
	assert.Empty(t, metrics.DropReasons)
	
	// Verify atomic counters are reset
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.ParallelFileDiscoveries))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.FileDiscoveryDuration))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.ASTCacheHits))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.ASTCacheMisses))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.PoolHits))
	assert.Equal(t, int64(0), atomic.LoadInt64(&metrics.PoolMisses))
	assert.Equal(t, int32(0), atomic.LoadInt32(&metrics.ConcurrencyLevel))
	assert.Equal(t, int32(0), atomic.LoadInt32(&metrics.PeakConcurrency))
}

// TestBuildMetrics_ConcurrentAccess validates thread safety
func TestBuildMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewBuildMetrics()
	const numGoroutines = 10
	const operationsPerGoroutine = 100
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 5) // 5 different operation types
	
	// Concurrent build recording
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				result := BuildResult{
					Duration: time.Duration(j) * time.Millisecond,
					Error:    nil,
					CacheHit: j%2 == 0, // Every other is a cache hit
				}
				metrics.RecordBuild(result)
			}
		}()
	}
	
	// Concurrent task dropping
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				metrics.RecordDroppedTask(fmt.Sprintf("Component%d_%d", id, j), "queue_full")
			}
		}(i)
	}
	
	// Concurrent parallel processing recording
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				metrics.RecordParallelProcessing(time.Duration(j)*time.Millisecond, int32(j%8+1))
			}
		}()
	}
	
	// Concurrent AST cache operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if j%2 == 0 {
					metrics.RecordASTCacheHit()
				} else {
					metrics.RecordASTCacheMiss(time.Duration(j) * time.Microsecond)
				}
			}
		}()
	}
	
	// Concurrent pool operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if j%3 == 0 {
					metrics.RecordPoolHit(int64(j * 10))
				} else {
					metrics.RecordPoolMiss()
				}
			}
		}()
	}
	
	wg.Wait()
	
	// Verify final counts are consistent
	totalExpectedBuilds := int64(numGoroutines * operationsPerGoroutine)
	totalExpectedDroppedTasks := int64(numGoroutines * operationsPerGoroutine)
	totalExpectedASTOps := int64(numGoroutines * operationsPerGoroutine)
	totalExpectedPoolOps := int64(numGoroutines * operationsPerGoroutine)
	
	assert.Equal(t, totalExpectedBuilds, metrics.GetBuildCount())
	
	droppedTasks, _, reasons := metrics.GetQueueHealthStatus()
	assert.Equal(t, totalExpectedDroppedTasks, droppedTasks)
	assert.Equal(t, totalExpectedDroppedTasks, reasons["queue_full"])
	
	astHits := atomic.LoadInt64(&metrics.ASTCacheHits)
	astMisses := atomic.LoadInt64(&metrics.ASTCacheMisses)
	assert.Equal(t, totalExpectedASTOps, astHits+astMisses)
	
	poolHits := atomic.LoadInt64(&metrics.PoolHits)
	poolMisses := atomic.LoadInt64(&metrics.PoolMisses)
	assert.Equal(t, totalExpectedPoolOps, poolHits+poolMisses)
	
	// Verify peak concurrency was tracked
	_, peak := metrics.GetConcurrencyStats()
	assert.True(t, peak > 0, "Peak concurrency should be greater than 0")
}

// TestBuildMetrics_PerformanceSummary validates comprehensive summary generation
func TestBuildMetrics_PerformanceSummary(t *testing.T) {
	metrics := NewBuildMetrics()
	
	// Populate with sample data
	successResult := BuildResult{Duration: 100 * time.Millisecond, Error: nil, CacheHit: true}
	failResult := BuildResult{Duration: 50 * time.Millisecond, Error: errors.NewBuildError("BUILD_FAILED", "fail", nil), CacheHit: false}
	
	metrics.RecordBuild(successResult)
	metrics.RecordBuild(failResult)
	metrics.RecordDroppedTask("Component1", "queue_full")
	metrics.RecordFileDiscovery(25*time.Millisecond, 5)
	metrics.RecordParallelProcessing(200*time.Millisecond, 3)
	metrics.RecordBatchProcessing(10, 100*time.Millisecond)
	metrics.RecordASTCacheHit()
	metrics.RecordASTCacheMiss(30*time.Millisecond)
	metrics.RecordPoolHit(512)
	
	summary := metrics.GetPerformanceSummary()
	
	// Validate structure and key values
	require.Contains(t, summary, "build_performance")
	require.Contains(t, summary, "parallel_processing")
	require.Contains(t, summary, "caching_performance")
	require.Contains(t, summary, "queue_health")
	
	buildPerf := summary["build_performance"].(map[string]interface{})
	assert.Equal(t, int64(2), buildPerf["total_builds"])
	assert.Equal(t, 50.0, buildPerf["success_rate"])     // 1 success / 2 total = 50%
	assert.Equal(t, 50.0, buildPerf["cache_hit_rate"])   // 1 cache hit / 2 total = 50%
	assert.Equal(t, 75*time.Millisecond, buildPerf["average_duration"]) // (100 + 50) / 2 = 75ms
	
	parallelProc := summary["parallel_processing"].(map[string]interface{})
	assert.Equal(t, int64(1), parallelProc["file_discoveries"])
	assert.Equal(t, int32(3), parallelProc["current_concurrency"])
	assert.Equal(t, int32(3), parallelProc["peak_concurrency"])
	assert.Equal(t, 10.0, parallelProc["avg_batch_size"])
	assert.Equal(t, int64(1), parallelProc["batch_count"])
	
	cachingPerf := summary["caching_performance"].(map[string]interface{})
	assert.Equal(t, 50.0, cachingPerf["ast_cache_hit_rate"]) // 1 hit / (1 hit + 1 miss) = 50%
	assert.Equal(t, int64(1), cachingPerf["ast_cache_hits"])
	assert.Equal(t, int64(1), cachingPerf["ast_cache_misses"])
	assert.Equal(t, 100.0, cachingPerf["pool_efficiency"])   // 1 hit / 1 total = 100%
	assert.Equal(t, int64(512), cachingPerf["memory_reused_bytes"])
	
	queueHealth := summary["queue_health"].(map[string]interface{})
	assert.Equal(t, int64(1), queueHealth["dropped_tasks"])
	assert.Equal(t, int64(0), queueHealth["dropped_results"])
	
	reasons := queueHealth["drop_reasons"].(map[string]int64)
	assert.Equal(t, int64(1), reasons["queue_full"])
}

// TestBuildMetrics_EdgeCases validates edge case handling
func TestBuildMetrics_EdgeCases(t *testing.T) {
	t.Run("division by zero protection", func(t *testing.T) {
		metrics := NewBuildMetrics()
		
		// All rate calculations should return 0 when no operations recorded
		assert.Equal(t, 0.0, metrics.GetCacheHitRate())
		assert.Equal(t, 0.0, metrics.GetSuccessRate())
		assert.Equal(t, 0.0, metrics.GetASTCacheHitRate())
		assert.Equal(t, 0.0, metrics.GetPoolEfficiency())
		assert.Equal(t, 0.0, metrics.GetAverageBatchSize())
		assert.Equal(t, time.Duration(0), metrics.GetAverageFileDiscoveryTime())
		assert.Equal(t, time.Duration(0), metrics.GetAverageParallelProcessingTime())
		assert.Equal(t, time.Duration(0), metrics.GetAverageDuration())
	})
	
	t.Run("concurrent peak concurrency updates", func(t *testing.T) {
		metrics := NewBuildMetrics()
		const numGoroutines = 50
		
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		// Concurrent peak concurrency updates
		for i := 0; i < numGoroutines; i++ {
			go func(concurrency int32) {
				defer wg.Done()
				metrics.RecordParallelProcessing(time.Millisecond, concurrency)
			}(int32(i + 1))
		}
		
		wg.Wait()
		
		_, peak := metrics.GetConcurrencyStats()
		assert.Equal(t, int32(numGoroutines), peak) // Should be the highest value
	})
	
	t.Run("large duration calculations", func(t *testing.T) {
		metrics := NewBuildMetrics()
		
		// Test with very large durations to ensure no overflow
		largeDuration := time.Hour * 24
		result := BuildResult{Duration: largeDuration, Error: nil, CacheHit: false}
		
		metrics.RecordBuild(result)
		
		assert.Equal(t, largeDuration, metrics.GetAverageDuration())
		assert.Equal(t, largeDuration, metrics.TotalDuration)
	})
}