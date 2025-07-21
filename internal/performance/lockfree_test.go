package performance

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Global generator for benchmark tests (non-deterministic)
var benchRng = rand.New(rand.NewSource(time.Now().UnixNano()))

// TestLockFreeRingBuffer tests basic ring buffer operations
func TestLockFreeRingBuffer(t *testing.T) {
	buffer := NewLockFreeRingBuffer(8) // Power of 2

	// Test basic write/read
	metric := Metric{
		Type:      MetricTypeBuildTime,
		Value:     100.0,
		Timestamp: time.Now(),
	}

	buffer.Write(metric)

	// Verify buffer state
	if buffer.writePos != 1 {
		t.Errorf("Expected writePos 1, got %d", buffer.writePos)
	}

	// Test multiple writes
	for i := 0; i < 10; i++ {
		buffer.Write(Metric{Value: float64(i)})
	}

	// Buffer should have wrapped around (size 8)
	if buffer.writePos != 11 {
		t.Errorf("Expected writePos 11, got %d", buffer.writePos)
	}
}

// TestLockFreeMetricCollector tests basic collector functionality
func TestLockFreeMetricCollector(t *testing.T) {
	collector := NewLockFreeMetricCollector(1000)

	// Test recording metrics
	metrics := []Metric{
		{Type: MetricTypeBuildTime, Value: 100.0},
		{Type: MetricTypeBuildTime, Value: 200.0},
		{Type: MetricTypeMemoryUsage, Value: 1000.0},
	}

	for _, metric := range metrics {
		collector.Record(metric)
	}

	// Test aggregates
	buildAgg := collector.GetAggregate(MetricTypeBuildTime)
	if buildAgg == nil {
		t.Fatal("Expected build time aggregate")
	}

	if buildAgg.Count != 2 {
		t.Errorf("Expected count 2, got %d", buildAgg.Count)
	}

	if buildAgg.Sum != 300.0 {
		t.Errorf("Expected sum 300.0, got %f", buildAgg.Sum)
	}

	memAgg := collector.GetAggregate(MetricTypeMemoryUsage)
	if memAgg == nil {
		t.Fatal("Expected memory usage aggregate")
	}

	if memAgg.Count != 1 {
		t.Errorf("Expected count 1, got %d", memAgg.Count)
	}
}

// TestLockFreeCollector_ConcurrentRecording tests concurrent metric recording
func TestLockFreeCollector_ConcurrentRecording(t *testing.T) {
	collector := NewLockFreeMetricCollector(10000)

	const numGoroutines = 10
	const metricsPerGoroutine = 1000

	var wg sync.WaitGroup
	start := make(chan struct{})

	// Start multiple goroutines recording metrics concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start // Wait for signal to start

			for j := 0; j < metricsPerGoroutine; j++ {
				metric := Metric{
					Type:      MetricTypeBuildTime,
					Value:     float64(id*1000 + j),
					Timestamp: time.Now(),
				}
				collector.Record(metric)
			}
		}(i)
	}

	// Start all goroutines simultaneously
	close(start)
	wg.Wait()

	// Verify all metrics were recorded
	agg := collector.GetAggregate(MetricTypeBuildTime)
	expectedCount := int64(numGoroutines * metricsPerGoroutine)

	if agg.Count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, agg.Count)
	}

	// Verify min/max values
	if agg.Min != 0.0 {
		t.Errorf("Expected min 0.0, got %f", agg.Min)
	}

	expectedMax := float64((numGoroutines-1)*1000 + metricsPerGoroutine - 1)
	if agg.Max != expectedMax {
		t.Errorf("Expected max %f, got %f", expectedMax, agg.Max)
	}
}

// TestLockFreeCollector_AtomicOperations tests atomic operation correctness
func TestLockFreeCollector_AtomicOperations(t *testing.T) {
	collector := NewLockFreeMetricCollector(1000)

	// Test atomic min/max updates with concurrent access
	const numGoroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Record metrics with different values to test atomic min/max
			for j := 0; j < 100; j++ {
				value := float64(id*100 + j)
				metric := Metric{
					Type:  MetricTypeMemoryUsage,
					Value: value,
				}
				collector.Record(metric)
			}
		}(i)
	}

	wg.Wait()

	agg := collector.GetAggregate(MetricTypeMemoryUsage)
	if agg == nil {
		t.Fatal("Expected memory usage aggregate")
	}

	// Verify atomic operations worked correctly
	expectedCount := int64(numGoroutines * 100)
	if agg.Count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, agg.Count)
	}

	if agg.Min != 0.0 {
		t.Errorf("Expected min 0.0, got %f", agg.Min)
	}

	expectedMax := float64((numGoroutines-1)*100 + 99)
	if agg.Max != expectedMax {
		t.Errorf("Expected max %f, got %f", expectedMax, agg.Max)
	}
}

// TestLockFreeCollector_Subscribers tests subscription mechanism
func TestLockFreeCollector_Subscribers(t *testing.T) {
	collector := NewLockFreeMetricCollector(1000)

	// Subscribe to metrics
	ch1 := collector.Subscribe()
	ch2 := collector.Subscribe()

	// Record a metric
	metric := Metric{
		Type:  MetricTypeBuildTime,
		Value: 100.0,
	}

	collector.Record(metric)

	// Verify both subscribers received the metric
	select {
	case received := <-ch1:
		if received.Value != 100.0 {
			t.Errorf("Expected value 100.0, got %f", received.Value)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 1 did not receive metric")
	}

	select {
	case received := <-ch2:
		if received.Value != 100.0 {
			t.Errorf("Expected value 100.0, got %f", received.Value)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 2 did not receive metric")
	}
}

// TestLockFreeCollector_PercentileUpdates tests percentile calculation updates
func TestLockFreeCollector_PercentileUpdates(t *testing.T) {
	collector := NewLockFreeMetricCollector(1000)

	// Record metrics in a known pattern
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, v := range values {
		metric := Metric{
			Type:  MetricTypeServerLatency,
			Value: v,
		}
		collector.Record(metric)
	}

	// Force percentile update
	collector.FlushMetrics()

	agg := collector.GetAggregate(MetricTypeServerLatency)
	if agg == nil {
		t.Fatal("Expected server latency aggregate")
	}

	// Verify percentiles are reasonable (exact values depend on implementation)
	if agg.P95 < 8.0 || agg.P95 > 10.0 {
		t.Errorf("P95 out of expected range: %f", agg.P95)
	}

	if agg.P99 < 9.0 || agg.P99 > 10.0 {
		t.Errorf("P99 out of expected range: %f", agg.P99)
	}
}

// BenchmarkLockFreeCollector_Record benchmarks lock-free recording
func BenchmarkLockFreeCollector_Record(t *testing.B) {
	collector := NewLockFreeMetricCollector(10000)

	metric := Metric{
		Type:  MetricTypeBuildTime,
		Value: 100.0,
	}

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		collector.Record(metric)
	}
}

// BenchmarkLockFreeCollector_ConcurrentRecord benchmarks concurrent recording
func BenchmarkLockFreeCollector_ConcurrentRecord(t *testing.B) {
	collector := NewLockFreeMetricCollector(100000)

	t.RunParallel(func(pb *testing.PB) {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: benchRng.Float64() * 1000,
		}

		for pb.Next() {
			collector.Record(metric)
		}
	})
}

// BenchmarkLockFreeVsOriginal_Record compares lock-free vs original implementation
func BenchmarkLockFreeVsOriginal_Record(t *testing.B) {
	t.Run("Original_Locked", func(b *testing.B) {
		collector := NewMetricCollector(10000)

		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: 100.0,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.Record(metric)
		}
	})

	t.Run("LockFree", func(b *testing.B) {
		collector := NewLockFreeMetricCollector(10000)

		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: 100.0,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.Record(metric)
		}
	})
}

// BenchmarkLockFreeVsOriginal_ConcurrentRecord compares concurrent performance
func BenchmarkLockFreeVsOriginal_ConcurrentRecord(t *testing.B) {
	t.Run("Original_Locked_Concurrent", func(b *testing.B) {
		collector := NewMetricCollector(100000)

		b.RunParallel(func(pb *testing.PB) {
			metric := Metric{
				Type:  MetricTypeBuildTime,
				Value: rand.Float64() * 1000,
			}

			for pb.Next() {
				collector.Record(metric)
			}
		})
	})

	t.Run("LockFree_Concurrent", func(b *testing.B) {
		collector := NewLockFreeMetricCollector(100000)

		b.RunParallel(func(pb *testing.PB) {
			metric := Metric{
				Type:  MetricTypeBuildTime,
				Value: rand.Float64() * 1000,
			}

			for pb.Next() {
				collector.Record(metric)
			}
		})
	})
}

// TestLockFreeCollector_MemoryUsage tests memory efficiency
func TestLockFreeCollector_MemoryUsage(t *testing.T) {
	collector := NewLockFreeMetricCollector(1000)

	// Record many metrics
	for i := 0; i < 10000; i++ {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(i),
		}
		collector.Record(metric)
	}

	// Verify buffer utilization
	utilization := collector.GetBufferUtilization()
	if utilization <= 0 || utilization > 100 {
		t.Errorf("Invalid buffer utilization: %f%%", utilization)
	}

	// Verify size is bounded
	size := collector.GetSize()
	if size > 1024 { // Next power of 2 after 1000
		t.Errorf("Buffer size exceeded maximum: %d", size)
	}
}

// TestLockFreeCollector_RaceConditions tests for race conditions
func TestLockFreeCollector_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	collector := NewLockFreeMetricCollector(10000)

	// Run with race detector enabled
	const numGoroutines = 50
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup

	// Mixed read/write operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				// Record metric
				metric := Metric{
					Type:  MetricTypeBuildTime,
					Value: float64(id*opsPerGoroutine + j),
				}
				collector.Record(metric)

				// Read aggregate
				if j%10 == 0 {
					agg := collector.GetAggregate(MetricTypeBuildTime)
					if agg != nil && agg.Count < 0 {
						t.Errorf("Invalid count: %d", agg.Count)
					}
				}

				// Subscribe/unsubscribe
				if j%100 == 0 {
					ch := collector.Subscribe()
					_ = ch // Use channel
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	agg := collector.GetAggregate(MetricTypeBuildTime)
	if agg == nil {
		t.Fatal("Expected aggregate after concurrent operations")
	}

	expectedCount := int64(numGoroutines * opsPerGoroutine)
	if agg.Count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, agg.Count)
	}
}

// TestNextPowerOf2 tests the power of 2 calculation
func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{1, 2},
		{2, 2},
		{3, 4},
		{7, 8},
		{8, 8},
		{15, 16},
		{1000, 1024},
		{1024, 1024},
		{1025, 2048},
	}

	for _, test := range tests {
		result := nextPowerOf2(test.input)
		if result != test.expected {
			t.Errorf("nextPowerOf2(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

// TestLockFreeCollector_BufferOverflow tests buffer overflow handling
func TestLockFreeCollector_BufferOverflow(t *testing.T) {
	// Create small buffer to force overflow
	collector := NewLockFreeMetricCollector(8)

	// Record more metrics than buffer size
	for i := 0; i < 20; i++ {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(i),
		}
		collector.Record(metric)
	}

	// Verify buffer handled overflow gracefully
	agg := collector.GetAggregate(MetricTypeBuildTime)
	if agg == nil {
		t.Fatal("Expected aggregate after overflow")
	}

	if agg.Count != 20 {
		t.Errorf("Expected count 20, got %d", agg.Count)
	}

	// Buffer size should be limited
	size := collector.GetSize()
	if size > 16 { // Next power of 2 after 8
		t.Errorf("Buffer size exceeded limit: %d", size)
	}
}

// TestLockFreeCollector_GetMetrics tests metric retrieval
func TestLockFreeCollector_GetMetrics(t *testing.T) {
	collector := NewLockFreeMetricCollector(100)

	now := time.Now()

	// Record metrics with different timestamps
	for i := 0; i < 10; i++ {
		metric := Metric{
			Type:      MetricTypeBuildTime,
			Value:     float64(i),
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		collector.Record(metric)
	}

	// Get metrics since 5 seconds ago
	since := now.Add(5 * time.Second)
	metrics := collector.GetMetrics(MetricTypeBuildTime, since)

	// Should get metrics 5-9 (5 metrics)
	if len(metrics) != 5 {
		t.Errorf("Expected 5 metrics, got %d", len(metrics))
	}

	// Verify values
	for i, metric := range metrics {
		expectedValue := float64(5 + i)
		if metric.Value != expectedValue {
			t.Errorf("Expected value %f, got %f", expectedValue, metric.Value)
		}
	}
}

// TestLockFreeCollector_GetMetricTypes tests metric type enumeration
func TestLockFreeCollector_GetMetricTypes(t *testing.T) {
	collector := NewLockFreeMetricCollector(100)

	// Record metrics of different types
	types := []MetricType{
		MetricTypeBuildTime,
		MetricTypeMemoryUsage,
		MetricTypeServerLatency,
	}

	for _, metricType := range types {
		metric := Metric{
			Type:  metricType,
			Value: 100.0,
		}
		collector.Record(metric)
	}

	// Get all metric types
	resultTypes := collector.GetMetricTypes()

	if len(resultTypes) != len(types) {
		t.Errorf("Expected %d types, got %d", len(types), len(resultTypes))
	}

	// Verify all types are present
	typeMap := make(map[MetricType]bool)
	for _, t := range resultTypes {
		typeMap[t] = true
	}

	for _, expectedType := range types {
		if !typeMap[expectedType] {
			t.Errorf("Missing expected type: %s", expectedType)
		}
	}
}
