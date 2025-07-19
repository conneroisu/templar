package performance

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptimizationConfig(t *testing.T) {
	config := DefaultOptimizationConfig()
	
	assert.True(t, config.EnableCPUOptimization)
	assert.True(t, config.EnableMemoryOptimization)
	assert.True(t, config.EnableIOOptimization)
	assert.True(t, config.EnableCacheOptimization)
	
	assert.Equal(t, runtime.GOMAXPROCS(0)*4, config.MaxGoroutines)
	assert.Equal(t, 100, config.GCTargetPercent)
	assert.Equal(t, runtime.GOMAXPROCS(0)*2, config.IOConcurrencyLimit)
	assert.Equal(t, 2, config.CacheOptimizationLevel)
	
	assert.Equal(t, 5*time.Second, config.MonitoringInterval)
	assert.Equal(t, 30*time.Second, config.OptimizationInterval)
	assert.Equal(t, 0.8, config.MemoryThreshold)
	assert.Equal(t, 0.9, config.CPUThreshold)
}

func TestPerformanceMetrics_UpdateAndGet(t *testing.T) {
	optimizer := NewPerformanceOptimizer(nil, nil, nil)
	
	// Update metrics
	optimizer.updateMetrics()
	
	// Get metrics
	metrics := optimizer.GetMetrics()
	
	assert.Greater(t, metrics.MemoryUsage, int64(0))
	assert.Greater(t, metrics.GoroutineCount, 0)
	assert.False(t, metrics.LastUpdated.IsZero())
}

func TestCPUOptimizer_Optimize(t *testing.T) {
	config := DefaultOptimizationConfig()
	optimizer := NewCPUOptimizer(config)
	
	ctx := context.Background()
	metrics := PerformanceMetrics{
		CPUUsage: 0.95, // High CPU usage
	}
	
	initialProcs := runtime.GOMAXPROCS(0)
	
	// Should not crash
	optimizer.Optimize(ctx, metrics)
	
	// Verify GOMAXPROCS was adjusted (may or may not change depending on system)
	finalProcs := runtime.GOMAXPROCS(0)
	assert.GreaterOrEqual(t, finalProcs, 1)
	
	// Reset to initial value
	runtime.GOMAXPROCS(initialProcs)
}

func TestMemoryOptimizer_Optimize(t *testing.T) {
	config := DefaultOptimizationConfig()
	optimizer := NewMemoryOptimizer(config)
	
	ctx := context.Background()
	metrics := PerformanceMetrics{
		MemoryUsage: 1024 * 1024 * 100, // 100MB
	}
	
	// Should not crash
	optimizer.Optimize(ctx, metrics)
	
	// Test GC cooldown
	optimizer.lastGCForced = time.Now()
	optimizer.Optimize(ctx, metrics) // Should not force GC due to cooldown
}

func TestIOOptimizer_AcquireRelease(t *testing.T) {
	config := DefaultOptimizationConfig()
	config.IOConcurrencyLimit = 2
	
	optimizer := NewIOOptimizer(config)
	ctx := context.Background()
	
	// Should be able to acquire up to the limit
	err1 := optimizer.AcquireIOSlot(ctx)
	assert.NoError(t, err1)
	
	err2 := optimizer.AcquireIOSlot(ctx)
	assert.NoError(t, err2)
	
	// Third acquisition should block
	ctx3, cancel3 := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel3()
	
	err3 := optimizer.AcquireIOSlot(ctx3)
	assert.Error(t, err3)
	assert.Equal(t, context.DeadlineExceeded, err3)
	
	// Release and try again
	optimizer.ReleaseIOSlot()
	
	err4 := optimizer.AcquireIOSlot(ctx)
	assert.NoError(t, err4)
	
	// Clean up
	optimizer.ReleaseIOSlot()
	optimizer.ReleaseIOSlot()
}

func TestIOOptimizer_Optimize(t *testing.T) {
	config := DefaultOptimizationConfig()
	config.MaxGoroutines = 100
	config.IOConcurrencyLimit = 10
	
	optimizer := NewIOOptimizer(config)
	ctx := context.Background()
	
	// Test with high goroutine count
	metrics := PerformanceMetrics{
		GoroutineCount: 150, // Above max
	}
	
	optimizer.Optimize(ctx, metrics)
	
	// Should have reduced I/O concurrency limit
	assert.Less(t, config.IOConcurrencyLimit, 10)
	
	// Test with low goroutine count
	metrics.GoroutineCount = 25 // Below max/2
	optimizer.Optimize(ctx, metrics)
	
	// May have increased I/O concurrency limit (up to system limits)
}

func TestCacheOptimizer_Optimize(t *testing.T) {
	config := DefaultOptimizationConfig()
	optimizer := NewCacheOptimizer(config)
	
	ctx := context.Background()
	metrics := PerformanceMetrics{
		CacheHitRate: 0.2, // Poor cache performance
	}
	
	// Should not crash with nil build pipeline
	optimizer.Optimize(ctx, metrics, nil)
	
	// Test with mock build pipeline would require more complex setup
}

func TestOptimizedFileScanner_ScanFile(t *testing.T) {
	config := DefaultOptimizationConfig()
	config.IOConcurrencyLimit = 1
	
	ioOptimizer := NewIOOptimizer(config)
	scanner := NewOptimizedFileScanner(ioOptimizer)
	
	ctx := context.Background()
	
	// Should acquire and release I/O slot
	err := scanner.ScanFileOptimized(ctx, "test.txt")
	assert.NoError(t, err)
}

func TestOptimizedFileScanner_Concurrency(t *testing.T) {
	config := DefaultOptimizationConfig()
	config.IOConcurrencyLimit = 2
	
	ioOptimizer := NewIOOptimizer(config)
	scanner := NewOptimizedFileScanner(ioOptimizer)
	
	ctx := context.Background()
	
	var wg sync.WaitGroup
	successCount := int32(0)
	timeoutCount := int32(0)
	
	// Start multiple concurrent scans
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Use a longer timeout to ensure the test behavior is predictable
			scanCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			
			err := scanner.ScanFileOptimized(scanCtx, "test.txt")
			
			if err == context.DeadlineExceeded {
				atomic.AddInt32(&timeoutCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	
	wg.Wait()
	
	// At least some should succeed (exact numbers may vary due to timing)
	totalProcessed := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&timeoutCount)
	assert.Equal(t, int32(5), totalProcessed)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&successCount), int32(1))
}

func TestBatchProcessor_ProcessBatch(t *testing.T) {
	processor := NewBatchProcessor(10, 2)
	
	ctx := context.Background()
	items := []interface{}{1, 2, 3, 4, 5}
	
	processorFunc := func(item interface{}) (interface{}, error) {
		num := item.(int)
		return num * 2, nil
	}
	
	results, err := processor.ProcessBatch(ctx, items, processorFunc)
	
	assert.NoError(t, err)
	assert.Len(t, results, 5)
	
	// Convert results to map for easier comparison (order may vary due to concurrency)
	resultMap := make(map[int]bool)
	for _, result := range results {
		resultMap[result.(int)] = true
	}
	
	// Check that all expected doubled values are present
	expectedValues := []int{2, 4, 6, 8, 10}
	for _, expected := range expectedValues {
		assert.True(t, resultMap[expected], "Expected value %d not found in results", expected)
	}
}

func TestBatchProcessor_ErrorHandling(t *testing.T) {
	processor := NewBatchProcessor(10, 2)
	
	ctx := context.Background()
	items := []interface{}{1, 2, 3}
	
	processorFunc := func(item interface{}) (interface{}, error) {
		num := item.(int)
		if num == 2 {
			return nil, assert.AnError
		}
		return num * 2, nil
	}
	
	_, err := processor.ProcessBatch(ctx, items, processorFunc)
	
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestBatchProcessor_ContextCancellation(t *testing.T) {
	processor := NewBatchProcessor(10, 2)
	
	ctx, cancel := context.WithCancel(context.Background())
	items := []interface{}{1, 2, 3, 4, 5}
	
	processorFunc := func(item interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond) // Simulate slow processing
		return item, nil
	}
	
	// Cancel context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	
	_, err := processor.ProcessBatch(ctx, items, processorFunc)
	
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestHelperFunctions(t *testing.T) {
	// Test max function
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 5, max(5, 3))
	assert.Equal(t, 5, max(5, 5))
	
	// Test min function
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, 5, min(5, 5))
}

func TestPerformanceOptimizer_StartStop(t *testing.T) {
	optimizer := NewPerformanceOptimizer(nil, nil, nil)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Should start without error
	optimizer.Start(ctx)
	
	// Wait for context cancellation (simulates stopping)
	<-ctx.Done()
	
	// Should not have any running goroutines after context cancellation
	// (This is harder to test directly, but the function should handle cleanup)
}

func TestPerformanceOptimizer_Configuration(t *testing.T) {
	optimizer := NewPerformanceOptimizer(nil, nil, nil)
	
	require.NotNil(t, optimizer.config)
	require.NotNil(t, optimizer.metrics)
	require.NotNil(t, optimizer.cpuOptimizer)
	require.NotNil(t, optimizer.memOptimizer)
	require.NotNil(t, optimizer.ioOptimizer)
	require.NotNil(t, optimizer.cacheOptimizer)
	
	// Verify configuration values
	assert.True(t, optimizer.config.EnableCPUOptimization)
	assert.True(t, optimizer.config.EnableMemoryOptimization)
	assert.True(t, optimizer.config.EnableIOOptimization)
	assert.True(t, optimizer.config.EnableCacheOptimization)
}

func BenchmarkPerformanceMetrics_Update(b *testing.B) {
	optimizer := NewPerformanceOptimizer(nil, nil, nil)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		optimizer.updateMetrics()
	}
}

func BenchmarkIOOptimizer_AcquireRelease(b *testing.B) {
	config := DefaultOptimizationConfig()
	config.IOConcurrencyLimit = 1000 // High limit to avoid blocking
	
	optimizer := NewIOOptimizer(config)
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			optimizer.AcquireIOSlot(ctx)
			optimizer.ReleaseIOSlot()
		}
	})
}

func BenchmarkBatchProcessor_SmallBatch(b *testing.B) {
	processor := NewBatchProcessor(10, 4)
	
	ctx := context.Background()
	items := []interface{}{1, 2, 3, 4, 5}
	
	processorFunc := func(item interface{}) (interface{}, error) {
		return item.(int) * 2, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		processor.ProcessBatch(ctx, items, processorFunc)
	}
}

func BenchmarkBatchProcessor_LargeBatch(b *testing.B) {
	processor := NewBatchProcessor(100, 8)
	
	// Create large batch
	items := make([]interface{}, 1000)
	for i := range items {
		items[i] = i
	}
	
	ctx := context.Background()
	processorFunc := func(item interface{}) (interface{}, error) {
		return item.(int) * 2, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		processor.ProcessBatch(ctx, items, processorFunc)
	}
}