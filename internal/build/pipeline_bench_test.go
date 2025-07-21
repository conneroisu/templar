package build

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// createTestComponent creates a test component for benchmarking
func createTestComponent(name string, complexity string) *types.ComponentInfo {
	var parameters []types.ParameterInfo

	switch complexity {
	case "simple":
		parameters = []types.ParameterInfo{
			{Name: "text", Type: "string"},
		}
	case "medium":
		parameters = []types.ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "active", Type: "bool"},
		}
	case "complex":
		parameters = []types.ParameterInfo{
			{Name: "id", Type: "int"},
			{Name: "title", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "tags", Type: "[]string"},
			{Name: "metadata", Type: "map[string]interface{}"},
			{Name: "active", Type: "bool"},
			{Name: "timestamp", Type: "time.Time"},
		}
	}

	return &types.ComponentInfo{
		Name:       name,
		Package:    "components",
		FilePath:   fmt.Sprintf("components/%s.templ", name),
		Parameters: parameters,
		Hash:       fmt.Sprintf("hash_%s", name),
		LastMod:    time.Now(),
	}
}

// createTestCacheEntry creates a test cache entry
func createTestCacheEntry(size int) []byte {
	return make([]byte, size)
}

// BenchmarkBuildPipeline_Build benchmarks component building performance
func BenchmarkBuildPipeline_Build(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}
	complexities := []string{"simple", "medium", "complex"}

	for _, workers := range workerCounts {
		for _, complexity := range complexities {
			b.Run(fmt.Sprintf("workers-%d-%s", workers, complexity), func(b *testing.B) {
				reg := NewMockComponentRegistry()
				pipeline := NewBuildPipeline(workers, reg)
				defer pipeline.Stop()

				component := createTestComponent("TestComponent", complexity)

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					pipeline.Build(component)
				}
			})
		}
	}
}

// BenchmarkBuildPipeline_ConcurrentBuilds benchmarks concurrent build performance
func BenchmarkBuildPipeline_ConcurrentBuilds(b *testing.B) {
	reg := NewMockComponentRegistry()
	pipeline := NewBuildPipeline(4, reg)
	defer pipeline.Stop()

	// Create test components
	components := make([]*types.ComponentInfo, 100)
	for i := 0; i < 100; i++ {
		complexity := []string{"simple", "medium", "complex"}[i%3]
		components[i] = createTestComponent(fmt.Sprintf("Component%d", i), complexity)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		componentIndex := 0
		for pb.Next() {
			component := components[componentIndex%len(components)]
			pipeline.Build(component)
			componentIndex++
		}
	})
}

// BenchmarkBuildCache_Operations benchmarks cache operations
func BenchmarkBuildCache_Operations(b *testing.B) {
	maxMemory := 100 * 1024 * 1024 // 100MB

	b.Run("Set", func(b *testing.B) {
		cache := newTestCache(int64(maxMemory), time.Hour)
		entrySizes := []int{1024, 10 * 1024, 100 * 1024} // 1KB, 10KB, 100KB

		for _, size := range entrySizes {
			b.Run(fmt.Sprintf("size-%dKB", size/1024), func(b *testing.B) {
				entry := createTestCacheEntry(size)

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("key_%d", i)
					cache.Set(key, entry)
				}
			})
		}
	})

	b.Run("Get", func(b *testing.B) {
		cache := newTestCache(int64(maxMemory), time.Hour)

		// Pre-populate cache
		for i := 0; i < 500; i++ {
			key := fmt.Sprintf("key_%d", i)
			entry := createTestCacheEntry(1024)
			cache.Set(key, entry)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%500)
			_, _ = cache.Get(key)
		}
	})

	b.Run("Mixed", func(b *testing.B) {
		cache := newTestCache(int64(maxMemory), time.Hour)

		// Pre-populate cache
		for i := 0; i < 250; i++ {
			key := fmt.Sprintf("key_%d", i)
			entry := createTestCacheEntry(1024)
			cache.Set(key, entry)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)

			// 80% reads, 20% writes
			if i%5 == 0 {
				entry := createTestCacheEntry(1024)
				cache.Set(key, entry)
			} else {
				cache.Get(key)
			}
		}
	})
}

// BenchmarkBuildCache_Eviction benchmarks cache eviction performance
func BenchmarkBuildCache_Eviction(b *testing.B) {
	b.Run("LRU_Eviction", func(b *testing.B) {
		// Small cache to force evictions
		cache := newTestCache(50*1024, time.Hour) // 50KB max

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			entry := createTestCacheEntry(10 * 1024) // 10KB per entry
			cache.Set(key, entry)
		}
	})

	b.Run("Memory_Pressure_Eviction", func(b *testing.B) {
		// Cache limited by memory
		cache := newTestCache(10*1024, time.Hour) // 10KB max memory

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			entry := createTestCacheEntry(5 * 1024) // 5KB per entry
			cache.Set(key, entry)
		}
	})
}

// BenchmarkBuildCache_ConcurrentAccess benchmarks concurrent cache access
func BenchmarkBuildCache_ConcurrentAccess(b *testing.B) {
	cache := newTestCache(10*1024*1024, time.Hour) // 10MB

	// Pre-populate cache
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := createTestCacheEntry(1024)
		cache.Set(key, entry)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		keyIndex := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", keyIndex%1000)

			// Mix of reads and writes
			if keyIndex%10 == 0 {
				entry := createTestCacheEntry(1024)
				cache.Set(key, entry)
			} else {
				cache.Get(key)
			}
			keyIndex++
		}
	})
}

// BenchmarkBuildPipeline_WorkerPoolPerformance benchmarks worker pool efficiency
func BenchmarkBuildPipeline_WorkerPoolPerformance(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}
	taskCounts := []int{10, 100, 1000}

	for _, workers := range workerCounts {
		for _, tasks := range taskCounts {
			b.Run(fmt.Sprintf("workers-%d-tasks-%d", workers, tasks), func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					reg := NewMockComponentRegistry()
					pipeline := NewBuildPipeline(workers, reg)

					// Submit tasks
					var wg sync.WaitGroup
					for j := 0; j < tasks; j++ {
						wg.Add(1)
						go func(taskID int) {
							defer wg.Done()
							component := createTestComponent(fmt.Sprintf("Task%d", taskID), "simple")
							pipeline.Build(component)
						}(j)
					}

					wg.Wait()
					pipeline.Stop()
				}
			})
		}
	}
}

// BenchmarkBuildPipeline_MemoryUsage benchmarks memory usage patterns
func BenchmarkBuildPipeline_MemoryUsage(b *testing.B) {
	b.Run("SmallWorkload", func(b *testing.B) {
		benchmarkPipelineMemoryUsage(b, 2, 10)
	})

	b.Run("MediumWorkload", func(b *testing.B) {
		benchmarkPipelineMemoryUsage(b, 4, 100)
	})

	b.Run("LargeWorkload", func(b *testing.B) {
		benchmarkPipelineMemoryUsage(b, 8, 1000)
	})
}

func benchmarkPipelineMemoryUsage(b *testing.B, workers int, componentCount int) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reg := NewMockComponentRegistry()
		pipeline := NewBuildPipeline(workers, reg)

		// Create and process components
		components := make([]*types.ComponentInfo, componentCount)
		for j := 0; j < componentCount; j++ {
			complexity := []string{"simple", "medium", "complex"}[j%3]
			components[j] = createTestComponent(fmt.Sprintf("Component%d", j), complexity)
		}

		// Process all components
		var wg sync.WaitGroup
		for _, component := range components {
			wg.Add(1)
			go func(comp *types.ComponentInfo) {
				defer wg.Done()
				pipeline.Build(comp)
			}(component)
		}

		wg.Wait()
		pipeline.Stop()
	}
}

// BenchmarkBuildCache_MemoryEfficiency benchmarks cache memory efficiency
func BenchmarkBuildCache_MemoryEfficiency(b *testing.B) {
	memorySizes := []int{
		1 * 1024 * 1024,   // 1MB
		10 * 1024 * 1024,  // 10MB
		100 * 1024 * 1024, // 100MB
	}

	for _, memSize := range memorySizes {
		b.Run(fmt.Sprintf("memory-%dMB", memSize/(1024*1024)), func(b *testing.B) {
			cache := newTestCache(int64(memSize), time.Hour)

			b.ResetTimer()
			b.ReportAllocs()

			entrySize := 1024 // 1KB per entry
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i)
				entry := createTestCacheEntry(entrySize)
				cache.Set(key, entry)
			}
		})
	}
}

// BenchmarkBuildResult_Serialization benchmarks build result serialization
func BenchmarkBuildResult_Serialization(b *testing.B) {
	resultSizes := []string{"small", "medium", "large"}

	for _, size := range resultSizes {
		b.Run(size, func(b *testing.B) {
			var output []byte
			switch size {
			case "small":
				output = make([]byte, 1024) // 1KB
			case "medium":
				output = make([]byte, 100*1024) // 100KB
			case "large":
				output = make([]byte, 1024*1024) // 1MB
			}

			result := &BuildResult{
				Component: createTestComponent("TestComponent", "medium"),
				Output:    output,
				Error:     nil,
				Duration:  100 * time.Millisecond,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate serialization/deserialization overhead
				_ = len(result.Output)
				_ = result.Error
				_ = result.Duration
			}
		})
	}
}
