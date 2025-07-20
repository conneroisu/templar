package build

import (
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

func TestObjectPools(t *testing.T) {
	pools := NewObjectPools()

	t.Run("BuildResult Pool", func(t *testing.T) {
		// Get a result from pool
		result1 := pools.GetBuildResult()
		if result1 == nil {
			t.Fatal("Expected non-nil BuildResult from pool")
		}

		// Populate it
		result1.Component = &registry.ComponentInfo{Name: "Test"}
		result1.Duration = time.Second
		result1.CacheHit = true

		// Return to pool
		pools.PutBuildResult(result1)

		// Get another result - should be the same object, but reset
		result2 := pools.GetBuildResult()
		if result2 == nil {
			t.Fatal("Expected non-nil BuildResult from pool")
		}

		// Should be reset
		if result2.Component != nil {
			t.Error("Expected Component to be nil after reset")
		}
		if result2.Duration != 0 {
			t.Error("Expected Duration to be 0 after reset")
		}
		if result2.CacheHit {
			t.Error("Expected CacheHit to be false after reset")
		}

		pools.PutBuildResult(result2)
	})

	t.Run("BuildTask Pool", func(t *testing.T) {
		task1 := pools.GetBuildTask()
		if task1 == nil {
			t.Fatal("Expected non-nil BuildTask from pool")
		}

		// Populate it
		task1.Component = &registry.ComponentInfo{Name: "Test"}
		task1.Priority = 5
		task1.Timestamp = time.Now()

		// Return to pool
		pools.PutBuildTask(task1)

		// Get another task - should be reset
		task2 := pools.GetBuildTask()
		if task2 == nil {
			t.Fatal("Expected non-nil BuildTask from pool")
		}

		// Should be reset
		if task2.Component != nil {
			t.Error("Expected Component to be nil after reset")
		}
		if task2.Priority != 0 {
			t.Error("Expected Priority to be 0 after reset")
		}

		pools.PutBuildTask(task2)
	})

	t.Run("Output Buffer Pool", func(t *testing.T) {
		buffer1 := pools.GetOutputBuffer()
		if buffer1 == nil {
			t.Fatal("Expected non-nil buffer from pool")
		}
		if len(buffer1) != 0 {
			t.Error("Expected empty buffer from pool")
		}

		// Use the buffer
		buffer1 = append(buffer1, []byte("test data")...)
		if len(buffer1) != 9 {
			t.Error("Expected buffer length to be 9")
		}

		// Return to pool
		pools.PutOutputBuffer(buffer1)

		// Get another buffer - should be reset
		buffer2 := pools.GetOutputBuffer()
		if len(buffer2) != 0 {
			t.Error("Expected empty buffer after reset")
		}

		pools.PutOutputBuffer(buffer2)
	})
}

func TestSlicePools(t *testing.T) {
	pools := NewSlicePools()

	t.Run("ComponentInfo Slice Pool", func(t *testing.T) {
		slice1 := pools.GetComponentInfoSlice()
		if slice1 == nil {
			t.Fatal("Expected non-nil slice from pool")
		}
		if len(slice1) != 0 {
			t.Error("Expected empty slice from pool")
		}

		// Use the slice
		component := &registry.ComponentInfo{Name: "Test"}
		slice1 = append(slice1, component)
		if len(slice1) != 1 {
			t.Error("Expected slice length to be 1")
		}

		// Return to pool
		pools.PutComponentInfoSlice(slice1)

		// Get another slice - should be reset
		slice2 := pools.GetComponentInfoSlice()
		if len(slice2) != 0 {
			t.Error("Expected empty slice after reset")
		}

		pools.PutComponentInfoSlice(slice2)
	})

	t.Run("String Slice Pool", func(t *testing.T) {
		slice1 := pools.GetStringSlice()
		if slice1 == nil {
			t.Fatal("Expected non-nil slice from pool")
		}

		slice1 = append(slice1, "test1", "test2")
		if len(slice1) != 2 {
			t.Error("Expected slice length to be 2")
		}

		pools.PutStringSlice(slice1)

		slice2 := pools.GetStringSlice()
		if len(slice2) != 0 {
			t.Error("Expected empty slice after reset")
		}

		pools.PutStringSlice(slice2)
	})
}

func TestWorkerPool(t *testing.T) {
	pool := NewWorkerPool()

	t.Run("Worker Pool Basic", func(t *testing.T) {
		worker1 := pool.GetWorker()
		if worker1 == nil {
			t.Fatal("Expected non-nil worker from pool")
		}
		if worker1.Context == nil {
			t.Fatal("Expected worker to have context")
		}

		// Use the worker
		worker1.ID = 123
		worker1.State = WorkerBusy
		worker1.Context.TempDir = "/tmp/test"

		// Return to pool
		pool.PutWorker(worker1)

		// Get another worker - should be reset
		worker2 := pool.GetWorker()
		if worker2.ID != 0 {
			t.Error("Expected worker ID to be reset")
		}
		if worker2.State != WorkerIdle {
			t.Error("Expected worker state to be idle")
		}
		if worker2.Context.TempDir != "" {
			t.Error("Expected worker context to be reset")
		}

		pool.PutWorker(worker2)
	})
}

// Benchmark tests to measure memory allocation improvements

func BenchmarkBuildResultCreation(b *testing.B) {
	b.Run("Without Pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := &BuildResult{
				Component: &registry.ComponentInfo{Name: "Test"},
				Duration:  time.Millisecond,
				CacheHit:  false,
			}
			result.Reset() // Simulate cleanup
		}
	})

	b.Run("With Pool", func(b *testing.B) {
		pools := NewObjectPools()
		for i := 0; i < b.N; i++ {
			result := pools.GetBuildResult()
			result.Component = &registry.ComponentInfo{Name: "Test"}
			result.Duration = time.Millisecond
			result.CacheHit = false
			pools.PutBuildResult(result)
		}
	})
}

func BenchmarkOutputBufferUsage(b *testing.B) {
	testData := []byte("This is test data for benchmarking buffer allocation patterns")

	b.Run("Without Pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := make([]byte, 0, 1024)
			buffer = append(buffer, testData...)
			_ = buffer
		}
	})

	b.Run("With Pool", func(b *testing.B) {
		pools := NewObjectPools()
		for i := 0; i < b.N; i++ {
			buffer := pools.GetOutputBuffer()
			buffer = append(buffer, testData...)
			pools.PutOutputBuffer(buffer)
		}
	})
}

func BenchmarkSliceAllocation(b *testing.B) {
	components := make([]*registry.ComponentInfo, 10)
	for i := range components {
		components[i] = &registry.ComponentInfo{Name: "Test"}
	}

	b.Run("Without Pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := make([]*registry.ComponentInfo, 0, 50)
			slice = append(slice, components...)
			_ = slice
		}
	})

	b.Run("With Pool", func(b *testing.B) {
		pools := NewSlicePools()
		for i := 0; i < b.N; i++ {
			slice := pools.GetComponentInfoSlice()
			slice = append(slice, components...)
			pools.PutComponentInfoSlice(slice)
		}
	})
}

func BenchmarkConcurrentPoolUsage(b *testing.B) {
	pools := NewObjectPools()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate concurrent build result creation
			result := pools.GetBuildResult()
			result.Component = &registry.ComponentInfo{Name: "Test"}
			result.Duration = time.Microsecond
			pools.PutBuildResult(result)

			// Simulate buffer usage
			buffer := pools.GetOutputBuffer()
			buffer = append(buffer, []byte("test")...)
			pools.PutOutputBuffer(buffer)
		}
	})
}

// Memory allocation tracking tests
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("BuildPipeline Without Pools", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate build pipeline operations without pools
			result := &BuildResult{
				Component: &registry.ComponentInfo{Name: "Test"},
				Output:    make([]byte, 1024),
				Duration:  time.Millisecond,
			}
			_ = result
		}
	})

	b.Run("BuildPipeline With Pools", func(b *testing.B) {
		b.ReportAllocs()
		pools := NewObjectPools()
		for i := 0; i < b.N; i++ {
			// Simulate build pipeline operations with pools
			result := pools.GetBuildResult()
			result.Component = &registry.ComponentInfo{Name: "Test"}
			result.Output = pools.GetOutputBuffer()
			result.Duration = time.Millisecond

			pools.PutOutputBuffer(result.Output)
			pools.PutBuildResult(result)
		}
	})
}

func BenchmarkWorkerPoolUsage(b *testing.B) {
	pool := NewWorkerPool()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		worker := pool.GetWorker()
		worker.ID = i
		worker.State = WorkerBusy
		worker.Context.TempDir = "/tmp"

		// Simulate some work
		worker.Context.OutputBuffer = append(worker.Context.OutputBuffer, []byte("output")...)

		pool.PutWorker(worker)
	}
}
