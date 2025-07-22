package build

import (
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// BenchmarkBuildMetrics_NewBuildMetrics benchmarks metrics creation
func BenchmarkBuildMetrics_NewBuildMetrics(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := NewBuildMetrics()
		_ = metrics
	}
}

// BenchmarkBuildMetrics_RecordBuild benchmarks build result recording
func BenchmarkBuildMetrics_RecordBuild(b *testing.B) {
	metrics := NewBuildMetrics()
	
	result := BuildResult{
		Component: &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		},
		Output:   []byte("Success"),
		Error:    nil,
		Duration: 100 * time.Millisecond,
		CacheHit: false,
		Hash:     "abc123",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordBuild(result)
	}
}

// BenchmarkBuildMetrics_GetSnapshot benchmarks snapshot retrieval
func BenchmarkBuildMetrics_GetSnapshot(b *testing.B) {
	metrics := NewBuildMetrics()
	
	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		result := BuildResult{
			Component: &types.ComponentInfo{
				Name:     "Component",
				FilePath: "test.templ",
				Package:  "test",
			},
			Output:   []byte("Output"),
			Duration: time.Duration(i+1) * time.Millisecond,
			CacheHit: i%3 == 0,
			Hash:     "hash",
		}
		if i%10 == 0 {
			result.Error = newTestError("test error")
		}
		metrics.RecordBuild(result)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetSnapshot()
	}
}

// BenchmarkBuildMetrics_ConcurrentAccess benchmarks concurrent access
func BenchmarkBuildMetrics_ConcurrentAccess(b *testing.B) {
	metrics := NewBuildMetrics()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			result := BuildResult{
				Component: &types.ComponentInfo{
					Name:     "ConcurrentComponent",
					FilePath: "test.templ",
					Package:  "test",
				},
				Output:   []byte("Output"),
				Duration: time.Duration(i%1000) * time.Microsecond,
				CacheHit: i%5 == 0,
				Hash:     "hash",
			}
			if i%10 == 0 {
				result.Error = newTestError("concurrent error")
			}
			metrics.RecordBuild(result)
			i++
		}
	})
}

// BenchmarkBuildMetrics_Reset benchmarks metrics reset operation
func BenchmarkBuildMetrics_Reset(b *testing.B) {
	b.Run("empty_metrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics := NewBuildMetrics()
			metrics.Reset()
		}
	})
	
	b.Run("populated_metrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			metrics := NewBuildMetrics()
			// Populate with data
			for j := 0; j < 50; j++ {
				result := BuildResult{
					Component: &types.ComponentInfo{
						Name:     "PopulatedComponent",
						FilePath: "test.templ", 
						Package:  "test",
					},
					Output:   []byte("Output"),
					Duration: 50 * time.Millisecond,
					CacheHit: j%3 == 0,
					Hash:     "hash",
				}
				if j%5 == 0 {
					result.Error = newTestError("populated error")
				}
				metrics.RecordBuild(result)
			}
			b.StartTimer()
			
			metrics.Reset()
		}
	})
}

// newTestError creates a test error
func newTestError(msg string) error {
	return &testError{message: msg}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}