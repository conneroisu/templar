package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/testutils"
)

// BenchmarkMemoryUsage provides memory usage benchmarks for different component counts.
func BenchmarkMemoryUsage(b *testing.B) {
	tests := []struct {
		name           string
		componentCount int
	}{
		{"small_project", 10},
		{"medium_project", 100},
		{"large_project", 1000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkMemoryUsage(b, tt.componentCount)
		})
	}
}

func benchmarkMemoryUsage(b *testing.B, componentCount int) {
	tc := testutils.NewBenchmarkContext(b)
	fs := testutils.NewBenchmarkFileSystem(b, tc.TempDir())

	// Create components
	for i := range componentCount {
		content := fmt.Sprintf(`package components

templ MemoryTestComponent%d(title string, data []string) {
	<div class="memory-test-%d">
		<h1>{ title }</h1>
		for _, item := range data {
			<p>{ item }</p>
		}
	</div>
}`, i, i)
		fs.CreateFile(fmt.Sprintf("memory%d.templ", i), content)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		// Simulate complete workflow
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Perform operations that allocate memory
		performWorkflowOperations(tc.TempDir(), componentCount)

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Track memory growth
		if i == 0 {
			b.Logf("Memory usage for %d components: %d bytes",
				componentCount, m2.Alloc-m1.Alloc)
		}
	}
}

// BenchmarkConcurrency tests performance under concurrent load.
func BenchmarkConcurrency(b *testing.B) {
	tests := []struct {
		name       string
		goroutines int
		operations int
	}{
		{"low_concurrency", 2, 100},
		{"medium_concurrency", 10, 100},
		{"high_concurrency", 50, 100},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkConcurrency(b, tt.goroutines, tt.operations)
		})
	}
}

func benchmarkConcurrency(b *testing.B, goroutines, operations int) {
	tc := testutils.NewBenchmarkContext(b)
	fs := testutils.NewBenchmarkFileSystem(b, tc.TempDir())

	// Setup test data
	for i := range 50 {
		content := fmt.Sprintf(`package components
templ ConcurrentComponent%d() { <div>%d</div> }`, i, i)
		fs.CreateFile(fmt.Sprintf("concurrent%d.templ", i), content)
	}

	b.ResetTimer()

	for i := range b.N {
		var wg sync.WaitGroup
		start := time.Now()

		for g := range goroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for op := range operations {
					performConcurrentOperation(tc.TempDir(), goroutineID, op)
				}
			}(g)
		}

		wg.Wait()
		elapsed := time.Since(start)

		if i == 0 {
			b.Logf("Concurrent operations (%d goroutines, %d ops each): %v",
				goroutines, operations, elapsed)
		}
	}
}

// BenchmarkScaling tests how performance scales with project size.
func BenchmarkScaling(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("components_%d", size), func(b *testing.B) {
			benchmarkScaling(b, size)
		})
	}
}

func benchmarkScaling(b *testing.B, componentCount int) {
	tc := testutils.NewBenchmarkContext(b)
	fs := testutils.NewBenchmarkFileSystem(b, tc.TempDir())

	// Create components with varying complexity
	for i := range componentCount {
		complexity := i%3 + 1 // 1-3 complexity levels
		content := generateComplexComponent(i, complexity)
		fs.CreateFile(fmt.Sprintf("scale%d.templ", i), content)
	}

	b.ResetTimer()

	for i := range b.N {
		start := time.Now()
		performFullWorkflow(tc.TempDir())
		elapsed := time.Since(start)

		if i == 0 {
			throughput := float64(componentCount) / elapsed.Seconds()
			b.Logf("Scaling test (%d components): %.2f components/sec",
				componentCount, throughput)
		}
	}
}

// BenchmarkCacheEffectiveness tests caching performance.
func BenchmarkCacheEffectiveness(b *testing.B) {
	tests := []struct {
		name      string
		cacheSize int
		hitRatio  float64
	}{
		{"no_cache", 0, 0.0},
		{"small_cache", 50, 0.5},
		{"large_cache", 500, 0.9},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkCacheEffectiveness(b, tt.cacheSize, tt.hitRatio)
		})
	}
}

func benchmarkCacheEffectiveness(b *testing.B, cacheSize int, expectedHitRatio float64) {
	tc := testutils.NewBenchmarkContext(b)
	fs := testutils.NewBenchmarkFileSystem(b, tc.TempDir())

	// Create test components
	const componentCount = 100
	for i := range componentCount {
		content := fmt.Sprintf(`package components
templ CacheTestComponent%d() { <div>Cache test %d</div> }`, i, i)
		fs.CreateFile(fmt.Sprintf("cache%d.templ", i), content)
	}

	b.ResetTimer()

	for i := range b.N {
		// Simulate cache behavior
		cache := newMockCache(cacheSize)

		start := time.Now()
		hits, misses := simulateCacheOperations(cache, componentCount, expectedHitRatio)
		elapsed := time.Since(start)

		if i == 0 {
			actualHitRatio := float64(hits) / float64(hits+misses)
			b.Logf("Cache performance (size: %d): %.2f hit ratio, %v elapsed",
				cacheSize, actualHitRatio, elapsed)
		}
	}
}

// BenchmarkResourceContention tests performance under resource contention.
func BenchmarkResourceContention(b *testing.B) {
	tests := []struct {
		name            string
		cpuIntensive    bool
		memoryIntensive bool
		ioIntensive     bool
	}{
		{"baseline", false, false, false},
		{"cpu_contention", true, false, false},
		{"memory_contention", false, true, false},
		{"io_contention", false, false, true},
		{"full_contention", true, true, true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkResourceContention(b, tt.cpuIntensive, tt.memoryIntensive, tt.ioIntensive)
		})
	}
}

func benchmarkResourceContention(b *testing.B, cpuIntensive, memoryIntensive, ioIntensive bool) {
	tc := testutils.NewBenchmarkContext(b)
	fs := testutils.NewBenchmarkFileSystem(b, tc.TempDir())

	// Setup contention scenarios
	var contentionCtx context.Context
	var cancel context.CancelFunc

	if cpuIntensive || memoryIntensive || ioIntensive {
		contentionCtx, cancel = context.WithCancel(context.Background())
		defer cancel()

		// Start background contention
		if cpuIntensive {
			startCPUContention(contentionCtx)
		}
		if memoryIntensive {
			startMemoryContention(contentionCtx)
		}
		if ioIntensive {
			startIOContention(contentionCtx, tc.TempDir())
		}
	}

	// Create test components
	for i := range 50 {
		content := fmt.Sprintf(`package components
templ ContentionComponent%d() { <div>Contention test %d</div> }`, i, i)
		fs.CreateFile(fmt.Sprintf("contention%d.templ", i), content)
	}

	b.ResetTimer()

	for i := range b.N {
		start := time.Now()
		performWorkflowOperations(tc.TempDir(), 50)
		elapsed := time.Since(start)

		if i == 0 {
			b.Logf("Resource contention test: %v", elapsed)
		}
	}
}

// Helper functions for benchmarks

func performWorkflowOperations(dir string, componentCount int) {
	// Mock workflow operations
	// In real implementation, this would call actual scanner, registry, and build operations
	time.Sleep(time.Millisecond * time.Duration(componentCount/10))
}

func performConcurrentOperation(dir string, goroutineID, operationID int) {
	// Mock concurrent operation
	time.Sleep(time.Microsecond * 100)
}

func performFullWorkflow(dir string) {
	// Mock full workflow
	time.Sleep(time.Millisecond * 10)
}

func generateComplexComponent(id, complexity int) string {
	base := fmt.Sprintf(`package components

templ ComplexComponent%d(`, id)

	// Add parameters based on complexity
	params := make([]string, complexity*2)
	for i := range complexity * 2 {
		params[i] = fmt.Sprintf("param%d string", i)
	}

	base += joinStrings(params, ", ") + ") {\n"
	base += fmt.Sprintf(`	<div class="complex-component-%d">`, id)

	// Add content based on complexity
	for i := range complexity {
		base += fmt.Sprintf(`
		<section class="section-%d">
			<h%d>Section %d</h%d>
			<p>{ param%d }</p>
		</section>`, i, (i%3)+1, i, (i%3)+1, i)
	}

	base += `
	</div>
}`

	return base
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}

// Mock cache for benchmarking.
type mockCache struct {
	size  int
	items map[string]interface{}
}

func newMockCache(size int) *mockCache {
	return &mockCache{
		size:  size,
		items: make(map[string]interface{}),
	}
}

func (c *mockCache) Get(key string) (interface{}, bool) {
	val, exists := c.items[key]

	return val, exists
}

func (c *mockCache) Set(key string, value interface{}) {
	if len(c.items) >= c.size && c.size > 0 {
		// Simple eviction - remove first item
		for k := range c.items {
			delete(c.items, k)

			break
		}
	}
	c.items[key] = value
}

func simulateCacheOperations(cache *mockCache, componentCount int, expectedHitRatio float64) (hits, misses int) {
	const operations = 1000

	for i := range operations {
		key := fmt.Sprintf("component_%d", i%componentCount)

		if _, exists := cache.Get(key); exists {
			hits++
		} else {
			misses++
			cache.Set(key, fmt.Sprintf("data_%d", i))
		}
	}

	return hits, misses
}

// Contention simulation functions.
func startCPUContention(ctx context.Context) {
	for range runtime.NumCPU() {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// CPU-intensive work
					for j := range 10000 {
						_ = j * j
					}
				}
			}
		}()
	}
}

func startMemoryContention(ctx context.Context) {
	go func() {
		var memoryBlocks [][]byte
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Allocate memory blocks
				block := make([]byte, 1024*1024) // 1MB
				memoryBlocks = append(memoryBlocks, block)

				// Limit memory usage
				if len(memoryBlocks) > 100 {
					memoryBlocks = memoryBlocks[1:]
				}
				time.Sleep(time.Millisecond)
			}
		}
	}()
}

func startIOContention(ctx context.Context, dir string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate I/O operations
				time.Sleep(time.Millisecond * 10)
			}
		}
	}()
}
