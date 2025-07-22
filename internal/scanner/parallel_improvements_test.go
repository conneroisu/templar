package scanner

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// BenchmarkConfigurableConcurrency tests performance with different worker counts
func BenchmarkConfigurableConcurrency(b *testing.B) {
	fileCount := 500
	tempDir, cleanup := createLargeTestStructure(b, fileCount)
	defer cleanup()
	
	workerCounts := []int{1, 2, 4, 8, 16, 32}
	
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				reg := registry.NewComponentRegistry()
				scanner := NewComponentScannerWithConcurrency(reg, workers)
				
				start := time.Now()
				err := scanner.ScanDirectory(tempDir)
				if err != nil {
					b.Fatal(err)
				}
				elapsed := time.Since(start)
				
				metrics := scanner.GetMetrics()
				
				if i == 0 {
					b.Logf("Workers: %d, Time: %v, Files: %d, Components: %d, Cache: %d/%d (%.1f%% hit rate), Memory: %d KB", 
						workers, elapsed, metrics.FilesProcessed, metrics.ComponentsFound,
						metrics.CacheHits, metrics.CacheHits+metrics.CacheMisses,
						float64(metrics.CacheHits)/float64(metrics.CacheHits+metrics.CacheMisses)*100,
						metrics.PeakMemoryUsage/1024)
				}
				
				scanner.Close()
			}
		})
	}
}

// BenchmarkMemoryEfficiency tests memory usage during scanning
func BenchmarkMemoryEfficiency(b *testing.B) {
	scales := []int{100, 500, 1000}
	
	for _, scale := range scales {
		b.Run(fmt.Sprintf("Files_%d", scale), func(b *testing.B) {
			tempDir, cleanup := createLargeTestStructure(b, scale)
			defer cleanup()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				runtime.GC()
				
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)
				
				reg := registry.NewComponentRegistry()
				scanner := NewComponentScanner(reg)
				
				err := scanner.ScanDirectory(tempDir)
				if err != nil {
					b.Fatal(err)
				}
				
				runtime.ReadMemStats(&m2)
				
				metrics := scanner.GetMetrics()
				actualMemory := m2.Alloc - m1.Alloc
				
				if i == 0 {
					b.Logf("Files: %d, Scanner Memory: %d KB, Actual Memory: %d KB, Objects: %d", 
						scale, metrics.PeakMemoryUsage/1024, actualMemory/1024, m2.Mallocs-m1.Mallocs)
				}
				
				scanner.Close()
			}
		})
	}
}

// BenchmarkCacheEffectiveness measures cache performance improvements
func BenchmarkCacheEffectiveness(b *testing.B) {
	fileCount := 200
	tempDir, cleanup := createLargeTestStructure(b, fileCount)
	defer cleanup()
	
	b.Run("FirstScan_NoCacheHits", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)
			
			start := time.Now()
			err := scanner.ScanDirectory(tempDir)
			if err != nil {
				b.Fatal(err)
			}
			elapsed := time.Since(start)
			
			metrics := scanner.GetMetrics()
			
			if i == 0 {
				b.Logf("First scan - Time: %v, Cache hits: %d, Cache misses: %d", 
					elapsed, metrics.CacheHits, metrics.CacheMisses)
			}
			
			scanner.Close()
		}
	})
	
	b.Run("SecondScan_WithCacheHits", func(b *testing.B) {
		// Pre-populate cache
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)
		err := scanner.ScanDirectory(tempDir)
		if err != nil {
			b.Fatal(err)
		}
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			reg = registry.NewComponentRegistry()
			scanner.registry = reg
			scanner.ResetMetrics()
			
			start := time.Now()
			err := scanner.ScanDirectory(tempDir)
			if err != nil {
				b.Fatal(err)
			}
			elapsed := time.Since(start)
			
			metrics := scanner.GetMetrics()
			
			if i == 0 {
				hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
				b.Logf("Cached scan - Time: %v, Cache hits: %d, Cache misses: %d (%.1f%% hit rate)", 
					elapsed, metrics.CacheHits, metrics.CacheMisses, hitRate)
			}
		}
		
		scanner.Close()
	})
}

// TestScannerMetricsAccuracy verifies metrics are tracked correctly
func TestScannerMetricsAccuracy(t *testing.T) {
	fileCount := 50
	tempDir, cleanup := createLargeTestStructure(t, fileCount)
	defer cleanup()
	
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)
	defer scanner.Close()
	
	// First scan
	err := scanner.ScanDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	
	metrics1 := scanner.GetMetrics()
	
	// Verify first scan metrics
	if metrics1.FilesProcessed != int64(fileCount) {
		t.Errorf("Expected %d files processed, got %d", fileCount, metrics1.FilesProcessed)
	}
	
	if metrics1.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits on first scan, got %d", metrics1.CacheHits)
	}
	
	if metrics1.CacheMisses != int64(fileCount) {
		t.Errorf("Expected %d cache misses on first scan, got %d", fileCount, metrics1.CacheMisses)
	}
	
	if metrics1.ComponentsFound == 0 {
		t.Error("Expected to find components, got 0")
	}
	
	if metrics1.TotalScanTime == 0 {
		t.Error("Expected scan time > 0")
	}
	
	t.Logf("First scan metrics: Files=%d, Components=%d, Cache=%d/%d, Time=%v, Memory=%dKB", 
		metrics1.FilesProcessed, metrics1.ComponentsFound, 
		metrics1.CacheHits, metrics1.CacheMisses, 
		metrics1.TotalScanTime, metrics1.PeakMemoryUsage/1024)
	
	// Second scan (should hit cache)
	reg2 := registry.NewComponentRegistry()
	scanner.registry = reg2
	scanner.ResetMetrics()
	
	err = scanner.ScanDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	
	metrics2 := scanner.GetMetrics()
	
	// Verify second scan has cache hits
	if metrics2.CacheHits == 0 {
		t.Error("Expected cache hits on second scan, got 0")
	}
	
	if metrics2.ComponentsFound != metrics1.ComponentsFound {
		t.Errorf("Component count mismatch: first=%d, second=%d", 
			metrics1.ComponentsFound, metrics2.ComponentsFound)
	}
	
	// Second scan should be faster due to cache
	if metrics2.TotalScanTime >= metrics1.TotalScanTime {
		t.Logf("WARNING: Second scan (%v) was not faster than first scan (%v)", 
			metrics2.TotalScanTime, metrics1.TotalScanTime)
	}
	
	hitRate := float64(metrics2.CacheHits) / float64(metrics2.CacheHits+metrics2.CacheMisses) * 100
	t.Logf("Second scan metrics: Files=%d, Components=%d, Cache=%d/%d (%.1f%% hit rate), Time=%v, Memory=%dKB", 
		metrics2.FilesProcessed, metrics2.ComponentsFound, 
		metrics2.CacheHits, metrics2.CacheMisses, hitRate,
		metrics2.TotalScanTime, metrics2.PeakMemoryUsage/1024)
}

// TestConfigurableConcurrency verifies different worker counts work correctly
func TestConfigurableConcurrency(t *testing.T) {
	tempDir, cleanup := createLargeTestStructure(t, 20)
	defer cleanup()
	
	workerCounts := []int{1, 2, 4, 8}
	
	for _, workers := range workerCounts {
		t.Run(fmt.Sprintf("Workers_%d", workers), func(t *testing.T) {
			reg := registry.NewComponentRegistry()
			scanner := NewComponentScannerWithConcurrency(reg, workers)
			defer scanner.Close()
			
			if scanner.GetWorkerCount() != workers {
				t.Errorf("Expected %d workers, got %d", workers, scanner.GetWorkerCount())
			}
			
			err := scanner.ScanDirectory(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			
			metrics := scanner.GetMetrics()
			if metrics.FilesProcessed == 0 {
				t.Error("No files processed")
			}
			
			if metrics.ComponentsFound == 0 {
				t.Error("No components found")
			}
			
			t.Logf("Workers: %d, Files: %d, Components: %d, Time: %v", 
				workers, metrics.FilesProcessed, metrics.ComponentsFound, metrics.TotalScanTime)
		})
	}
}