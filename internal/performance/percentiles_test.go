package performance

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// TestSkipList_BasicOperations tests basic skip list functionality.
func TestSkipList_BasicOperations(t *testing.T) {
	sl := NewSkipList()

	// Test empty skip list
	if sl.Size() != 0 {
		t.Errorf("Expected size 0, got %d", sl.Size())
	}

	// Test insertion
	values := []float64{5.0, 2.0, 8.0, 1.0, 9.0, 3.0}
	for _, v := range values {
		sl.Insert(v)
	}

	if sl.Size() != len(values) {
		t.Errorf("Expected size %d, got %d", len(values), sl.Size())
	}

	// Test deletion
	if !sl.Delete(5.0) {
		t.Error("Expected to delete 5.0 successfully")
	}

	if sl.Size() != len(values)-1 {
		t.Errorf("Expected size %d after deletion, got %d", len(values)-1, sl.Size())
	}

	// Test deleting non-existent value
	if sl.Delete(100.0) {
		t.Error("Expected deletion of non-existent value to fail")
	}
}

// TestSkipList_PercentileCalculation tests percentile calculation accuracy.
func TestSkipList_PercentileCalculation(t *testing.T) {
	sl := NewSkipList()

	// Test with known dataset
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, v := range values {
		sl.Insert(v)
	}

	tests := []struct {
		percentile float64
		expected   float64
	}{
		{0, 1.0},    // 0th percentile (minimum)
		{50, 5.0},   // 50th percentile (median)
		{90, 9.0},   // 90th percentile
		{95, 9.0},   // 95th percentile (for 10 elements, P95 is 9th element)
		{100, 10.0}, // 100th percentile (maximum)
	}

	for _, test := range tests {
		result := sl.GetPercentile(test.percentile)
		if result != test.expected {
			t.Errorf(
				"Percentile %.0f: expected %.1f, got %.1f",
				test.percentile,
				test.expected,
				result,
			)
		}
	}
}

// TestPercentileCalculator_BasicFunctionality tests the percentile calculator.
func TestPercentileCalculator_BasicFunctionality(t *testing.T) {
	pc := NewPercentileCalculator(100)

	// Test empty calculator
	if pc.GetSize() != 0 {
		t.Errorf("Expected size 0, got %d", pc.GetSize())
	}

	// Add values
	values := []float64{1, 5, 3, 9, 2, 8, 4, 7, 6, 10}
	for _, v := range values {
		pc.AddValue(v)
	}

	if pc.GetSize() != len(values) {
		t.Errorf("Expected size %d, got %d", len(values), pc.GetSize())
	}

	// Test percentiles
	p95 := pc.GetP95()
	p99 := pc.GetP99()

	// For 10 values, P95 should be around the 9th value when sorted
	// For 10 values, P99 should be around the 9th-10th value when sorted
	if p95 < 8.0 || p95 > 10.0 {
		t.Errorf("P95 out of expected range: got %.2f", p95)
	}

	if p99 < 9.0 || p99 > 10.0 {
		t.Errorf("P99 out of expected range: got %.2f", p99)
	}
}

// TestPercentileCalculator_RingBufferEviction tests FIFO eviction behavior.
func TestPercentileCalculator_RingBufferEviction(t *testing.T) {
	maxSize := 5
	pc := NewPercentileCalculator(maxSize)

	// Fill beyond capacity
	for i := 1; i <= 10; i++ {
		pc.AddValue(float64(i))
	}

	// Should only keep the last 5 values
	if pc.GetSize() != maxSize {
		t.Errorf("Expected size %d after eviction, got %d", maxSize, pc.GetSize())
	}

	// Values should be 6, 7, 8, 9, 10
	all := pc.GetAll()
	expected := []float64{6, 7, 8, 9, 10}

	if len(all) != len(expected) {
		t.Errorf("Expected %d values, got %d", len(expected), len(all))
	}

	for i, v := range expected {
		if i >= len(all) || all[i] != v {
			t.Errorf("Expected value %.0f at position %d, got %.0f", v, i, all[i])
		}
	}
}

// TestPercentileCalculator_AccuracyVsStandardSort compares with standard sorting.
func TestPercentileCalculator_AccuracyVsStandardSort(t *testing.T) {
	pc := NewPercentileCalculator(1000)

	// Generate test data
	rng := rand.New(rand.NewSource(42)) // Deterministic test
	values := make([]float64, 500)
	for i := range values {
		values[i] = rng.Float64() * 1000
		pc.AddValue(values[i])
	}

	// Calculate percentiles using standard sort
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	calculateStandardPercentile := func(data []float64, percentile float64) float64 {
		index := int(float64(len(data)-1) * percentile / 100.0)
		if index >= len(data) {
			index = len(data) - 1
		}

		return data[index]
	}

	percentiles := []float64{50, 75, 90, 95, 99}
	tolerance := 0.01 // Allow small floating point differences

	for _, p := range percentiles {
		skipListResult := pc.GetPercentile(p)
		standardResult := calculateStandardPercentile(sorted, p)

		diff := math.Abs(skipListResult - standardResult)
		if diff > tolerance {
			t.Errorf("Percentile %.0f: skip list %.6f vs standard %.6f (diff: %.6f)",
				p, skipListResult, standardResult, diff)
		}
	}
}

// BenchmarkPercentileCalculator_OldVsNew compares performance.
func BenchmarkPercentileCalculator_OldVsNew(t *testing.B) {
	// Old O(n²) method simulation - single calculation
	oldCalculatePercentiles := func(values []float64) (float64, float64) {
		if len(values) == 0 {
			return 0, 0
		}

		sorted := make([]float64, len(values))
		copy(sorted, values)

		// Insertion sort (O(n²))
		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j] > key {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}

		p95Index := int(float64(len(sorted)) * 0.95)
		p99Index := int(float64(len(sorted)) * 0.99)

		if p95Index >= len(sorted) {
			p95Index = len(sorted) - 1
		}
		if p99Index >= len(sorted) {
			p99Index = len(sorted) - 1
		}

		return sorted[p95Index], sorted[p99Index]
	}

	// Test data
	values := make([]float64, 1000)
	rng := rand.New(rand.NewSource(42))
	for i := range values {
		values[i] = rng.Float64() * 1000
	}

	t.Run("Old_O(n²)_Single_Calculation", func(b *testing.B) {
		for range b.N {
			oldCalculatePercentiles(values)
		}
	})

	t.Run("New_SkipList_Single_Query", func(b *testing.B) {
		// Pre-populate the skip list once
		pc := NewPercentileCalculator(1000)
		for _, v := range values {
			pc.AddValue(v)
		}

		b.ResetTimer()
		for range b.N {
			pc.GetP95()
			pc.GetP99()
		}
	})

	t.Run("Old_O(n²)_Incremental_Simulation", func(b *testing.B) {
		// Simulate the old method with incremental updates
		var allValues []float64
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		b.ResetTimer()
		for i := range b.N {
			// Add a new value
			allValues = append(allValues, rng.Float64()*1000)

			// Recalculate percentiles every 10 additions (like the real usage)
			if i%10 == 0 {
				oldCalculatePercentiles(allValues)
			}
		}
	})

	t.Run("New_SkipList_Incremental", func(b *testing.B) {
		pc := NewPercentileCalculator(10000)
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		b.ResetTimer()
		for i := range b.N {
			// Add a new value
			pc.AddValue(rng.Float64() * 1000)

			// Calculate percentiles every 10 additions (like the real usage)
			if i%10 == 0 {
				pc.GetP95()
				pc.GetP99()
			}
		}
	})
}

// BenchmarkPercentileCalculator_ScalingPerformance tests performance at different scales.
func BenchmarkPercentileCalculator_ScalingPerformance(t *testing.B) {
	sizes := []int{100, 500, 1000, 5000, 10000}

	for _, size := range sizes {
		values := make([]float64, size)
		rng := rand.New(rand.NewSource(42))
		for i := range values {
			values[i] = rng.Float64() * 1000
		}

		t.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			for range b.N {
				pc := NewPercentileCalculator(size)
				for _, v := range values {
					pc.AddValue(v)
				}
				pc.GetP95()
				pc.GetP99()
			}
		})
	}
}

// BenchmarkPercentileCalculator_IncrementalUpdates tests incremental update performance.
func BenchmarkPercentileCalculator_IncrementalUpdates(t *testing.B) {
	pc := NewPercentileCalculator(10000)

	// Pre-populate with some data
	rng := rand.New(rand.NewSource(42))
	for range 1000 {
		pc.AddValue(rng.Float64() * 1000)
	}

	t.ResetTimer()
	for i := range t.N {
		pc.AddValue(rng.Float64() * 1000)
		if i%100 == 0 { // Calculate percentiles periodically
			pc.GetP95()
			pc.GetP99()
		}
	}
}

// TestPercentileCalculator_ConcurrentAccess tests thread safety.
func TestPercentileCalculator_ConcurrentAccess(t *testing.T) {
	pc := NewPercentileCalculator(1000)

	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := range 1000 {
			pc.AddValue(float64(i))
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for range 100 {
			pc.GetP95()
			pc.GetP99()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	if pc.GetSize() != 1000 {
		t.Errorf("Expected size 1000, got %d", pc.GetSize())
	}
}

// TestPercentileCalculator_MemoryEfficiency tests memory footprint.
func TestPercentileCalculator_MemoryEfficiency(t *testing.T) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		pc := NewPercentileCalculator(size)

		// Fill the calculator
		for i := range size {
			pc.AddValue(float64(i))
		}

		footprint := pc.MemoryFootprint()
		expectedRange := size * 56 // Skip list overhead: ~56 bytes per element (realistic)

		if footprint > int(
			float64(expectedRange)*1.5,
		) { // Allow 50% overhead for measurement variance
			t.Errorf("Memory footprint too high for size %d: %d bytes (expected ~%d)",
				size, footprint, expectedRange)
		}

		t.Logf("Size %d: Memory footprint %d bytes (%.2f bytes per element)",
			size, footprint, float64(footprint)/float64(size))
	}
}

// Unused test helper removed
