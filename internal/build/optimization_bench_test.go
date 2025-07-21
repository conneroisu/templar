package build

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// Realistic benchmarks that simulate actual build pipeline usage

func BenchmarkRealisticBuildPipeline(b *testing.B) {
	const numComponents = 100
	const numWorkers = 4

	// Create test components
	components := make([]*types.ComponentInfo, numComponents)
	for i := 0; i < numComponents; i++ {
		components[i] = &types.ComponentInfo{
			Name:     "Component",
			Package:  "components",
			FilePath: "component.templ",
		}
	}

	b.Run("Without Pools", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate build pipeline without pools
			results := make([]BuildResult, 0, numComponents)

			for j := 0; j < numComponents; j++ {
				// Simulate build output (typical templ generate output size)
				output := make([]byte, 2048) // 2KB typical output
				for k := range output {
					output[k] = byte(k % 256)
				}

				result := BuildResult{
					Component: components[j%len(components)],
					Output:    output,
					Duration:  time.Millisecond,
					Hash:      "abcd1234",
				}
				results = append(results, result)
			}

			// Simulate processing results
			for range results {
				// Process each result
			}
		}
	})

	b.Run("With Pools", func(b *testing.B) {
		b.ReportAllocs()
		pools := NewObjectPools()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate build pipeline with pools
			results := make([]*BuildResult, 0, numComponents)

			for j := 0; j < numComponents; j++ {
				result := pools.GetBuildResult()
				result.Component = components[j%len(components)]

				// Use pooled output buffer
				output := pools.GetOutputBuffer()
				// Simulate build output
				for k := 0; k < 2048; k++ {
					output = append(output, byte(k%256))
				}
				result.Output = make([]byte, len(output))
				copy(result.Output, output)
				pools.PutOutputBuffer(output)

				result.Duration = time.Millisecond
				result.Hash = "abcd1234"

				results = append(results, result)
			}

			// Simulate processing and cleanup
			for _, result := range results {
				pools.PutBuildResult(result)
			}
		}
	})
}

func BenchmarkConcurrentWorkerPool(b *testing.B) {
	const numWorkers = 8
	const tasksPerWorker = 50

	b.Run("Without Worker Pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for t := 0; t < tasksPerWorker; t++ {
						// Simulate worker without pooling
						worker := &BuildWorker{
							ID:    w,
							State: WorkerBusy,
							Context: &WorkerContext{
								TempDir:      "/tmp/worker",
								OutputBuffer: make([]byte, 0, 4096),
								ErrorBuffer:  make([]byte, 0, 1024),
								Environment:  make(map[string]string),
							},
						}

						// Simulate work
						worker.Context.OutputBuffer = append(worker.Context.OutputBuffer, []byte("build output")...)
						worker.Context.Environment["PATH"] = "/usr/bin"

						// Cleanup
						worker = nil
					}
				}()
			}

			wg.Wait()
		}
	})

	b.Run("With Worker Pool", func(b *testing.B) {
		b.ReportAllocs()
		workerPool := NewWorkerPool()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for t := 0; t < tasksPerWorker; t++ {
						// Get worker from pool
						worker := workerPool.GetWorker()
						worker.ID = w
						worker.State = WorkerBusy

						// Simulate work
						worker.Context.OutputBuffer = append(worker.Context.OutputBuffer, []byte("build output")...)
						worker.Context.Environment["PATH"] = "/usr/bin"

						// Return to pool
						workerPool.PutWorker(worker)
					}
				}()
			}

			wg.Wait()
		}
	})
}

func BenchmarkLargeSliceOperations(b *testing.B) {
	const numComponents = 1000

	// Create test data
	components := make([]*types.ComponentInfo, numComponents)
	for i := 0; i < numComponents; i++ {
		components[i] = &types.ComponentInfo{
			Name:     "Component" + string(rune(i)),
			Package:  "components",
			FilePath: "component.templ",
		}
	}

	b.Run("Without Slice Pools", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate component scanning without pools
			scannedComponents := make([]*types.ComponentInfo, 0, numComponents)
			filteredComponents := make([]*types.ComponentInfo, 0, numComponents/2)
			errorMessages := make([]string, 0, 10)

			// Simulate scanning
			for _, comp := range components {
				scannedComponents = append(scannedComponents, comp)
				if len(scannedComponents)%2 == 0 {
					filteredComponents = append(filteredComponents, comp)
				}
			}

			// Simulate error collection
			for j := 0; j < 5; j++ {
				errorMessages = append(errorMessages, "error message")
			}

			// Simulate processing
			_ = len(scannedComponents) + len(filteredComponents) + len(errorMessages)
		}
	})

	b.Run("With Slice Pools", func(b *testing.B) {
		b.ReportAllocs()
		slicePools := NewSlicePools()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Get slices from pools
			scannedComponents := slicePools.GetComponentInfoSlice()
			filteredComponents := slicePools.GetComponentInfoSlice()
			errorMessages := slicePools.GetStringSlice()

			// Simulate scanning
			for _, comp := range components {
				scannedComponents = append(scannedComponents, comp)
				if len(scannedComponents)%2 == 0 {
					filteredComponents = append(filteredComponents, comp)
				}
			}

			// Simulate error collection
			for j := 0; j < 5; j++ {
				errorMessages = append(errorMessages, "error message")
			}

			// Simulate processing
			_ = len(scannedComponents) + len(filteredComponents) + len(errorMessages)

			// Return to pools
			slicePools.PutComponentInfoSlice(scannedComponents)
			slicePools.PutComponentInfoSlice(filteredComponents)
			slicePools.PutStringSlice(errorMessages)
		}
	})
}

func BenchmarkMemoryPressure(b *testing.B) {
	// This benchmark simulates memory pressure scenarios where pools shine
	const iterations = 10000

	b.Run("High Allocation Pressure Without Pools", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for j := 0; j < iterations; j++ {
				// Simulate frequent allocations
				result := &BuildResult{
					Component: &types.ComponentInfo{Name: "Test"},
					Output:    make([]byte, 1024), // 1KB
					Duration:  time.Microsecond,
				}

				buffer := make([]byte, 0, 2048) // 2KB
				buffer = append(buffer, result.Output...)

				// Simulate GC pressure
				if j%100 == 0 {
					runtime.GC()
				}

				_ = result
				_ = buffer
			}
		}
	})

	b.Run("High Allocation Pressure With Pools", func(b *testing.B) {
		b.ReportAllocs()
		pools := NewObjectPools()

		for i := 0; i < b.N; i++ {
			for j := 0; j < iterations; j++ {
				// Use pools for allocations
				result := pools.GetBuildResult()
				result.Component = &types.ComponentInfo{Name: "Test"}
				result.Output = make([]byte, 1024) // Still need to allocate this
				result.Duration = time.Microsecond

				buffer := pools.GetOutputBuffer()
				buffer = append(buffer, result.Output...)

				// Return to pools
				pools.PutBuildResult(result)
				pools.PutOutputBuffer(buffer)

				// Simulate GC pressure
				if j%100 == 0 {
					runtime.GC()
				}
			}
		}
	})
}

func BenchmarkBuildPipelineRealistic(b *testing.B) {
	// This benchmark simulates a realistic build pipeline scenario
	reg := &registry.ComponentRegistry{}

	b.Run("Standard Pipeline", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			pipeline := NewBuildPipeline(4, reg)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			// Start pipeline
			pipeline.Start(ctx)

			// Simulate 10 build tasks
			for j := 0; j < 10; j++ {
				component := &types.ComponentInfo{
					Name:     "TestComponent",
					Package:  "components",
					FilePath: "test.templ",
				}

				// Use BuildTask instead of Queue method
				task := BuildTask{
					Component: component,
					Priority:  1,
					Timestamp: time.Now(),
				}
				pipeline.queue.tasks <- task
			}

			// Stop pipeline
			cancel()
			pipeline.Stop()
		}
	})
}

// Memory usage measurement helpers
func BenchmarkMemoryUsageComparison(b *testing.B) {
	const numObjects = 1000

	var m1, m2 runtime.MemStats

	b.Run("Memory Without Pools", func(b *testing.B) {
		runtime.GC()
		runtime.ReadMemStats(&m1)

		for i := 0; i < b.N; i++ {
			objects := make([]*BuildResult, numObjects)
			for j := 0; j < numObjects; j++ {
				objects[j] = &BuildResult{
					Component: &types.ComponentInfo{Name: "Test"},
					Output:    make([]byte, 512),
					Duration:  time.Millisecond,
				}
			}
			_ = objects
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes/total-alloc")
	})

	b.Run("Memory With Pools", func(b *testing.B) {
		pools := NewObjectPools()
		runtime.GC()
		runtime.ReadMemStats(&m1)

		for i := 0; i < b.N; i++ {
			objects := make([]*BuildResult, numObjects)
			for j := 0; j < numObjects; j++ {
				result := pools.GetBuildResult()
				result.Component = &types.ComponentInfo{Name: "Test"}
				result.Output = make([]byte, 512)
				result.Duration = time.Millisecond
				objects[j] = result
			}

			// Return to pools
			for _, obj := range objects {
				pools.PutBuildResult(obj)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes/total-alloc")
	})
}
