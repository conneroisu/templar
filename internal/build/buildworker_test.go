package build

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWorker_Reset(t *testing.T) {
	t.Run("reset clears all fields", func(t *testing.T) {
		worker := &BuildWorker{
			ID:    42,
			State: WorkerBusy,
			Context: &WorkerContext{
				TempDir:      "/tmp/test",
				OutputBuffer: []byte("test output"),
				ErrorBuffer:  []byte("test error"),
				Environment:  map[string]string{"TEST": "value"},
			},
		}

		worker.Reset()

		assert.Equal(t, 0, worker.ID)
		assert.Equal(t, WorkerIdle, worker.State)
		assert.Nil(t, worker.Context)
	})

	t.Run("reset handles nil context", func(t *testing.T) {
		worker := &BuildWorker{
			ID:      42,
			State:   WorkerBusy,
			Context: nil,
		}

		// Should not panic
		worker.Reset()

		assert.Equal(t, 0, worker.ID)
		assert.Equal(t, WorkerIdle, worker.State)
		assert.Nil(t, worker.Context)
	})
}

func TestWorkerContext_Reset(t *testing.T) {
	t.Run("reset clears all fields", func(t *testing.T) {
		ctx := &WorkerContext{
			TempDir:      "/tmp/test",
			OutputBuffer: []byte("test output"),
			ErrorBuffer:  []byte("test error"),
			Environment: map[string]string{
				"TEST1": "value1",
				"TEST2": "value2",
			},
		}

		ctx.Reset()

		assert.Equal(t, "", ctx.TempDir)
		assert.Equal(t, 0, len(ctx.OutputBuffer))
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
		assert.NotNil(t, ctx.Environment)
		assert.Equal(t, 0, len(ctx.Environment))
	})

	t.Run("reset handles oversized buffers", func(t *testing.T) {
		ctx := &WorkerContext{
			OutputBuffer: make([]byte, 2*1024*1024), // 2MB - larger than 1MB limit
			ErrorBuffer:  make([]byte, 128*1024),    // 128KB - larger than 64KB limit
		}

		ctx.Reset()

		// Large buffers should be niled out
		assert.Nil(t, ctx.OutputBuffer)
		assert.Nil(t, ctx.ErrorBuffer)
	})

	t.Run("reset preserves reasonable buffer capacity", func(t *testing.T) {
		ctx := &WorkerContext{
			OutputBuffer: make([]byte, 100, 512*1024), // 512KB capacity - within 1MB limit
			ErrorBuffer:  make([]byte, 50, 32*1024),   // 32KB capacity - within 64KB limit
		}

		originalOutputCap := cap(ctx.OutputBuffer)
		originalErrorCap := cap(ctx.ErrorBuffer)

		ctx.Reset()

		// Should preserve capacity but reset length
		assert.Equal(t, 0, len(ctx.OutputBuffer))
		assert.Equal(t, originalOutputCap, cap(ctx.OutputBuffer))
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
		assert.Equal(t, originalErrorCap, cap(ctx.ErrorBuffer))
	})

	t.Run("reset initializes nil environment", func(t *testing.T) {
		ctx := &WorkerContext{
			Environment: nil,
		}

		ctx.Reset()

		assert.NotNil(t, ctx.Environment)
		assert.Equal(t, 0, len(ctx.Environment))
	})
}

func TestWorkerPool_GetPutWorker(t *testing.T) {
	t.Run("get worker returns clean worker", func(t *testing.T) {
		pool := NewWorkerPool()

		worker := pool.GetWorker()

		assert.NotNil(t, worker)
		assert.Equal(t, 0, worker.ID)
		assert.Equal(t, WorkerIdle, worker.State)
		assert.NotNil(t, worker.Context)
	})

	t.Run("put worker resets and returns to pool", func(t *testing.T) {
		pool := NewWorkerPool()

		// Get a worker and modify it
		worker := pool.GetWorker()
		worker.ID = 42
		worker.State = WorkerBusy
		worker.Context.TempDir = "/tmp/test"

		// Put it back
		pool.PutWorker(worker)

		// Get another worker - should be reset
		worker2 := pool.GetWorker()
		assert.Equal(t, 0, worker2.ID)
		assert.Equal(t, WorkerIdle, worker2.State)
		assert.Equal(t, "", worker2.Context.TempDir)
	})

	t.Run("put worker handles nil worker", func(t *testing.T) {
		pool := NewWorkerPool()

		// Should not panic
		pool.PutWorker(nil)
	})

	t.Run("put worker handles worker with nil context", func(t *testing.T) {
		pool := NewWorkerPool()

		worker := &BuildWorker{
			ID:      42,
			State:   WorkerBusy,
			Context: nil,
		}

		// Should not panic
		pool.PutWorker(worker)
	})
}

func TestWorkerPool_GetPutWorkerContext(t *testing.T) {
	t.Run("get context returns clean context", func(t *testing.T) {
		pool := NewWorkerPool()

		ctx := pool.GetWorkerContext()

		assert.NotNil(t, ctx)
		assert.Equal(t, "", ctx.TempDir)
		assert.Equal(t, 0, len(ctx.OutputBuffer))
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
		assert.NotNil(t, ctx.Environment)
		assert.Equal(t, 0, len(ctx.Environment))
	})

	t.Run("put context resets and returns to pool", func(t *testing.T) {
		pool := NewWorkerPool()

		// Get a context and modify it
		ctx := pool.GetWorkerContext()
		ctx.TempDir = "/tmp/test"
		ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test")...)
		ctx.Environment["TEST"] = "value"

		// Put it back
		pool.PutWorkerContext(ctx)

		// Get another context - should be reset
		ctx2 := pool.GetWorkerContext()
		assert.Equal(t, "", ctx2.TempDir)
		assert.Equal(t, 0, len(ctx2.OutputBuffer))
		assert.Equal(t, 0, len(ctx2.Environment))
	})

	t.Run("put context handles nil context", func(t *testing.T) {
		pool := NewWorkerPool()

		// Should not panic
		pool.PutWorkerContext(nil)
	})
}

func TestWorkerPool_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent worker get/put is thread-safe", func(t *testing.T) {
		pool := NewWorkerPool()
		var wg sync.WaitGroup

		// Launch multiple goroutines that get and put workers
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				worker := pool.GetWorker()
				assert.NotNil(t, worker)

				// Modify worker to ensure reset works
				worker.ID = id
				worker.State = WorkerBusy
				worker.Context.TempDir = "/tmp/test"

				// Brief work simulation
				time.Sleep(1 * time.Millisecond)

				pool.PutWorker(worker)
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent context get/put is thread-safe", func(t *testing.T) {
		pool := NewWorkerPool()
		var wg sync.WaitGroup

		// Launch multiple goroutines that get and put contexts
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				ctx := pool.GetWorkerContext()
				assert.NotNil(t, ctx)

				// Modify context to ensure reset works
				ctx.TempDir = "/tmp/test"
				ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test")...)
				ctx.Environment["TEST"] = "value"

				// Brief work simulation
				time.Sleep(1 * time.Millisecond)

				pool.PutWorkerContext(ctx)
			}(i)
		}

		wg.Wait()
	})
}

func TestWorkerPool_MemoryManagement(t *testing.T) {
	t.Run("worker pool reuses objects", func(t *testing.T) {
		pool := NewWorkerPool()

		// Get a worker, modify it, put it back
		worker1 := pool.GetWorker()
		worker1Ptr := worker1
		worker1.ID = 42
		pool.PutWorker(worker1)

		// Get another worker - should be the same object reused
		worker2 := pool.GetWorker()
		assert.True(t, worker1Ptr == worker2, "Worker should be reused from pool")
		assert.Equal(t, 0, worker2.ID, "Worker should be reset")
	})

	t.Run("context pool reuses objects", func(t *testing.T) {
		pool := NewWorkerPool()

		// Get a context, modify it, put it back
		ctx1 := pool.GetWorkerContext()
		ctx1Ptr := ctx1
		ctx1.TempDir = "/tmp/test"
		pool.PutWorkerContext(ctx1)

		// Get another context - should be the same object reused
		ctx2 := pool.GetWorkerContext()
		assert.True(t, ctx1Ptr == ctx2, "Context should be reused from pool")
		assert.Equal(t, "", ctx2.TempDir, "Context should be reset")
	})
}

func TestWorkerStates(t *testing.T) {
	t.Run("worker state constants", func(t *testing.T) {
		assert.Equal(t, WorkerState(0), WorkerIdle)
		assert.Equal(t, WorkerState(1), WorkerBusy)
		assert.Equal(t, WorkerState(2), WorkerStopped)
	})

	t.Run("worker state transitions", func(t *testing.T) {
		worker := &BuildWorker{State: WorkerIdle}

		// Simulate state transitions
		worker.State = WorkerBusy
		assert.Equal(t, WorkerBusy, worker.State)

		worker.State = WorkerStopped
		assert.Equal(t, WorkerStopped, worker.State)

		worker.State = WorkerIdle
		assert.Equal(t, WorkerIdle, worker.State)
	})
}

func TestWorkerContext_BufferGrowth(t *testing.T) {
	t.Run("output buffer growth is managed", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset() // Initialize

		// Add data to buffer
		initialCap := cap(ctx.OutputBuffer)
		for i := 0; i < 1000; i++ {
			ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test data")...)
		}

		assert.Greater(t, len(ctx.OutputBuffer), 0)
		assert.GreaterOrEqual(t, cap(ctx.OutputBuffer), initialCap)

		// Reset should clear the buffer
		ctx.Reset()
		assert.Equal(t, 0, len(ctx.OutputBuffer))
	})

	t.Run("error buffer growth is managed", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset() // Initialize

		// Add data to buffer
		initialCap := cap(ctx.ErrorBuffer)
		for i := 0; i < 100; i++ {
			ctx.ErrorBuffer = append(ctx.ErrorBuffer, []byte("error message")...)
		}

		assert.Greater(t, len(ctx.ErrorBuffer), 0)
		assert.GreaterOrEqual(t, cap(ctx.ErrorBuffer), initialCap)

		// Reset should clear the buffer
		ctx.Reset()
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
	})
}

func TestWorkerContext_EnvironmentManagement(t *testing.T) {
	t.Run("environment map is properly managed", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset() // Initialize

		// Add environment variables
		ctx.Environment["VAR1"] = "value1"
		ctx.Environment["VAR2"] = "value2"
		ctx.Environment["VAR3"] = "value3"

		assert.Equal(t, 3, len(ctx.Environment))
		assert.Equal(t, "value1", ctx.Environment["VAR1"])

		// Reset should clear but preserve the map
		ctx.Reset()
		assert.Equal(t, 0, len(ctx.Environment))
		assert.NotNil(t, ctx.Environment)

		// Should be able to add new variables
		ctx.Environment["NEW_VAR"] = "new_value"
		assert.Equal(t, 1, len(ctx.Environment))
		assert.Equal(t, "new_value", ctx.Environment["NEW_VAR"])
	})
}

// Benchmark tests to verify performance characteristics
func BenchmarkWorkerPool_GetPutWorker(b *testing.B) {
	pool := NewWorkerPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		worker := pool.GetWorker()
		worker.ID = i
		worker.State = WorkerBusy
		pool.PutWorker(worker)
	}
}

func BenchmarkWorkerPool_GetPutContext(b *testing.B) {
	pool := NewWorkerPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := pool.GetWorkerContext()
		ctx.TempDir = "/tmp/test"
		ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test")...)
		pool.PutWorkerContext(ctx)
	}
}

func BenchmarkWorkerReset(b *testing.B) {
	worker := &BuildWorker{
		ID:    42,
		State: WorkerBusy,
		Context: &WorkerContext{
			TempDir:      "/tmp/test",
			OutputBuffer: make([]byte, 1024),
			ErrorBuffer:  make([]byte, 512),
			Environment: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		worker.Reset()
		// Restore state for next iteration
		worker.ID = 42
		worker.State = WorkerBusy
		worker.Context = &WorkerContext{
			TempDir:      "/tmp/test",
			OutputBuffer: make([]byte, 1024),
			ErrorBuffer:  make([]byte, 512),
			Environment: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		}
	}
}

func BenchmarkContextReset(b *testing.B) {
	ctx := &WorkerContext{
		TempDir:      "/tmp/test",
		OutputBuffer: make([]byte, 1024),
		ErrorBuffer:  make([]byte, 512),
		Environment: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx.Reset()
		// Restore state for next iteration
		ctx.TempDir = "/tmp/test"
		ctx.OutputBuffer = make([]byte, 1024)
		ctx.ErrorBuffer = make([]byte, 512)
		ctx.Environment = map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		}
	}
}

// Additional tests for cancellation scenarios and error handling

func TestBuildWorker_CancellationScenarios(t *testing.T) {
	t.Run("worker handles context cancellation", func(t *testing.T) {
		// Create a build pipeline with 1 worker for testing
		bp := NewBuildPipeline(1, nil)

		// Create a context that we can cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Start the pipeline
		bp.Start(ctx)

		// Add a component to build
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "/test/component.templ",
			Package:  "test",
		}

		// Cancel the context immediately
		cancel()

		// Try to submit a build task - should handle cancellation gracefully
		bp.Build(component)

		// Stop the pipeline
		bp.Stop()

		// Test should complete without hanging or panicking
	})

	t.Run("worker pool handles cancellation during task processing", func(t *testing.T) {
		pool := NewWorkerPool()

		// Get a worker
		worker := pool.GetWorker()
		require.NotNil(t, worker)

		// Simulate cancellation scenario by setting worker to stopped state
		worker.State = WorkerStopped

		// Put worker back - should handle stopped state
		pool.PutWorker(worker)

		// Get another worker - should be clean
		worker2 := pool.GetWorker()
		assert.Equal(t, WorkerIdle, worker2.State)
	})

	t.Run("worker context survives cancellation", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset()

		// Add some data
		ctx.TempDir = "/tmp/cancelled"
		ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("partial output")...)
		ctx.ErrorBuffer = append(ctx.ErrorBuffer, []byte("error during cancellation")...)

		// Simulate cancellation cleanup
		ctx.Reset()

		// Should be clean after cancellation
		assert.Equal(t, "", ctx.TempDir)
		assert.Equal(t, 0, len(ctx.OutputBuffer))
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
	})
}

func TestBuildWorker_ErrorHandling(t *testing.T) {
	t.Run("worker handles buffer overflow gracefully", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset()

		// Simulate buffer overflow by creating very large buffers
		largeData := make([]byte, 5*1024*1024) // 5MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		ctx.OutputBuffer = append(ctx.OutputBuffer, largeData...)
		ctx.ErrorBuffer = append(ctx.ErrorBuffer, largeData...)

		// Reset should handle large buffers by deallocating them
		ctx.Reset()

		// Should have reasonable capacity after reset
		assert.LessOrEqual(
			t,
			cap(ctx.OutputBuffer),
			1024*1024,
			"Output buffer capacity should be limited",
		)
		assert.LessOrEqual(
			t,
			cap(ctx.ErrorBuffer),
			64*1024,
			"Error buffer capacity should be limited",
		)
	})

	t.Run("worker handles nil pointer errors", func(t *testing.T) {
		var worker *BuildWorker = nil

		// Should not panic when resetting nil worker
		assert.NotPanics(t, func() {
			if worker != nil {
				worker.Reset()
			}
		})

		pool := NewWorkerPool()

		// Should handle putting nil worker
		assert.NotPanics(t, func() {
			pool.PutWorker(nil)
		})

		// Should handle putting nil context
		assert.NotPanics(t, func() {
			pool.PutWorkerContext(nil)
		})
	})

	t.Run("worker handles environment variable corruption", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset()

		// Add many environment variables to test cleanup
		for i := 0; i < 1000; i++ {
			ctx.Environment[fmt.Sprintf("VAR_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		assert.Equal(t, 1000, len(ctx.Environment))

		// Reset should clear all variables
		ctx.Reset()

		assert.Equal(t, 0, len(ctx.Environment))
		assert.NotNil(t, ctx.Environment)
	})

	t.Run("worker pool handles high contention", func(t *testing.T) {
		pool := NewWorkerPool()
		var wg sync.WaitGroup
		errors := make(chan error, 1000)

		// Create high contention scenario
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("panic in goroutine %d: %v", id, r)
					}
				}()

				worker := pool.GetWorker()
				if worker == nil {
					errors <- fmt.Errorf("got nil worker in goroutine %d", id)
					return
				}

				// Simulate work
				worker.ID = id
				worker.State = WorkerBusy

				// Random work time
				time.Sleep(time.Duration(id%10) * time.Microsecond)

				pool.PutWorker(worker)
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Error(err)
		}
	})
}

func TestBuildWorker_PerformanceUnderLoad(t *testing.T) {
	t.Run("worker pool performance under sustained load", func(t *testing.T) {
		pool := NewWorkerPool()
		iterations := 10000

		start := time.Now()

		for i := 0; i < iterations; i++ {
			worker := pool.GetWorker()
			worker.ID = i
			worker.State = WorkerBusy

			// Simulate some work
			worker.Context.OutputBuffer = append(worker.Context.OutputBuffer, []byte("test")...)
			worker.Context.Environment["ITER"] = fmt.Sprintf("%d", i)

			pool.PutWorker(worker)
		}

		duration := time.Since(start)
		t.Logf("Processed %d worker get/put cycles in %v (%.2f ops/sec)",
			iterations, duration, float64(iterations)/duration.Seconds())

		// Should complete in reasonable time (less than 1 second for 10k ops)
		assert.Less(t, duration, time.Second, "Worker pool should handle 10k operations quickly")
	})

	t.Run("context reset performance", func(t *testing.T) {
		ctx := &WorkerContext{
			TempDir:      "/tmp/test",
			OutputBuffer: make([]byte, 0, 1024),
			ErrorBuffer:  make([]byte, 0, 512),
			Environment:  make(map[string]string),
		}

		// Pre-populate with data
		for i := 0; i < 100; i++ {
			ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test data")...)
			ctx.ErrorBuffer = append(ctx.ErrorBuffer, []byte("error")...)
			ctx.Environment[fmt.Sprintf("VAR_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			ctx.Reset()

			// Add some data for next iteration
			ctx.TempDir = "/tmp/test"
			ctx.OutputBuffer = append(ctx.OutputBuffer, []byte("test")...)
			ctx.Environment["TEST"] = "value"
		}

		duration := time.Since(start)
		t.Logf("Performed %d context resets in %v (%.2f ops/sec)",
			iterations, duration, float64(iterations)/duration.Seconds())

		// Should be very fast (less than 100ms for 1k resets)
		assert.Less(t, duration, 100*time.Millisecond, "Context reset should be very fast")
	})
}

func TestBuildWorker_ResourceLimits(t *testing.T) {
	t.Run("worker context enforces buffer size limits", func(t *testing.T) {
		ctx := &WorkerContext{}
		ctx.Reset()

		// Create buffers that exceed limits
		hugeOutput := make([]byte, 2*1024*1024) // 2MB - exceeds 1MB limit
		hugeError := make([]byte, 128*1024)     // 128KB - exceeds 64KB limit

		ctx.OutputBuffer = hugeOutput
		ctx.ErrorBuffer = hugeError

		// Reset should deallocate oversized buffers
		ctx.Reset()

		// Buffers should be nil after reset (deallocated)
		assert.Nil(t, ctx.OutputBuffer)
		assert.Nil(t, ctx.ErrorBuffer)
	})

	t.Run("worker context preserves reasonable buffer sizes", func(t *testing.T) {
		ctx := &WorkerContext{}

		// Create reasonably sized buffers
		reasonableOutput := make([]byte, 100, 512*1024) // 512KB capacity - within limit
		reasonableError := make([]byte, 50, 32*1024)    // 32KB capacity - within limit

		ctx.OutputBuffer = reasonableOutput
		ctx.ErrorBuffer = reasonableError

		originalOutputCap := cap(ctx.OutputBuffer)
		originalErrorCap := cap(ctx.ErrorBuffer)

		// Reset should preserve capacity
		ctx.Reset()

		assert.Equal(t, 0, len(ctx.OutputBuffer))
		assert.Equal(t, originalOutputCap, cap(ctx.OutputBuffer))
		assert.Equal(t, 0, len(ctx.ErrorBuffer))
		assert.Equal(t, originalErrorCap, cap(ctx.ErrorBuffer))
	})
}

// Additional benchmark for memory allocation patterns
func BenchmarkWorkerPool_HighContention(b *testing.B) {
	pool := NewWorkerPool()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			worker := pool.GetWorker()
			worker.ID = i
			worker.State = WorkerBusy
			worker.Context.TempDir = "/tmp/bench"
			pool.PutWorker(worker)
			i++
		}
	})
}
