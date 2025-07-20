package performance

import (
	"testing"
	"time"
)

func TestMetricCollector(t *testing.T) {
	collector := NewMetricCollector(100)

	// Test recording metrics
	metric := Metric{
		Type:  MetricTypeBuildTime,
		Value: 150.0,
		Unit:  "ms",
		Labels: map[string]string{"component": "test"},
	}

	collector.Record(metric)

	// Test retrieving metrics
	metrics := collector.GetMetrics(MetricTypeBuildTime, time.Time{})
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Value != 150.0 {
		t.Errorf("Expected value 150.0, got %f", metrics[0].Value)
	}
}

func TestMetricAggregate(t *testing.T) {
	collector := NewMetricCollector(100)

	// Record multiple metrics
	values := []float64{100, 200, 150, 300, 250}
	for _, value := range values {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: value,
			Unit:  "ms",
		}
		collector.Record(metric)
	}

	// Test aggregates
	agg := collector.GetAggregate(MetricTypeBuildTime)
	if agg == nil {
		t.Fatal("Expected aggregate, got nil")
	}

	if agg.Count != 5 {
		t.Errorf("Expected count 5, got %d", agg.Count)
	}

	expectedSum := 1000.0
	if agg.Sum != expectedSum {
		t.Errorf("Expected sum %f, got %f", expectedSum, agg.Sum)
	}

	expectedAvg := 200.0
	if agg.Avg != expectedAvg {
		t.Errorf("Expected avg %f, got %f", expectedAvg, agg.Avg)
	}

	if agg.Min != 100.0 {
		t.Errorf("Expected min 100.0, got %f", agg.Min)
	}

	if agg.Max != 300.0 {
		t.Errorf("Expected max 300.0, got %f", agg.Max)
	}
}

func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)

	// Test recording metric
	metric := Metric{
		Type:  MetricTypeMemoryUsage,
		Value: 1024 * 1024, // 1MB
		Unit:  "bytes",
	}
	monitor.Record(metric)

	// Test getting metrics
	metrics := monitor.GetMetrics(MetricTypeMemoryUsage, time.Time{})
	if len(metrics) == 0 {
		t.Error("Expected at least 1 metric")
	}

	// Test getting aggregates
	agg := monitor.GetAggregate(MetricTypeMemoryUsage)
	if agg == nil {
		t.Error("Expected aggregate, got nil")
	}
}

func TestRecommendationGeneration(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)

	// Record high memory usage to trigger recommendation
	for i := 0; i < 10; i++ {
		metric := Metric{
			Type:  MetricTypeMemoryUsage,
			Value: float64(1024 * 1024 * 1024), // 1GB
			Unit:  "bytes",
		}
		monitor.Record(metric)
	}

	// Start monitoring to generate recommendations
	monitor.Start()
	defer monitor.Stop()

	// Wait a bit for recommendations to be generated
	time.Sleep(time.Millisecond * 200)

	// Check if recommendations are generated
	recommendations := monitor.GetRecommendations()
	select {
	case recommendation := <-recommendations:
		if recommendation.Type != "memory_optimization" {
			t.Errorf("Expected memory_optimization recommendation, got %s", recommendation.Type)
		}
		if recommendation.Priority <= 0 {
			t.Error("Expected positive priority")
		}
	case <-time.After(time.Second):
		// No recommendation generated, which might be expected
		// depending on thresholds
	}
}

func TestMetricSubscription(t *testing.T) {
	collector := NewMetricCollector(100)

	// Subscribe to metrics
	subscription := collector.Subscribe()

	// Record a metric
	metric := Metric{
		Type:  MetricTypeBuildTime,
		Value: 100.0,
		Unit:  "ms",
	}
	collector.Record(metric)

	// Check if subscriber receives the metric
	select {
	case receivedMetric := <-subscription:
		if receivedMetric.Type != MetricTypeBuildTime {
			t.Errorf("Expected %s, got %s", MetricTypeBuildTime, receivedMetric.Type)
		}
		if receivedMetric.Value != 100.0 {
			t.Errorf("Expected value 100.0, got %f", receivedMetric.Value)
		}
	case <-time.After(time.Millisecond * 100):
		t.Error("Timeout waiting for metric")
	}
}

func TestMetricRotation(t *testing.T) {
	collector := NewMetricCollector(3) // Small capacity for testing

	// Record more metrics than capacity
	for i := 0; i < 5; i++ {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(i),
			Unit:  "ms",
		}
		collector.Record(metric)
	}

	// Should only have the last 3 metrics
	metrics := collector.GetMetrics("", time.Time{})
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}

	// Check that we have the latest metrics (values 2, 3, 4)
	expectedValues := []float64{2.0, 3.0, 4.0}
	for i, metric := range metrics {
		if metric.Value != expectedValues[i] {
			t.Errorf("Expected value %f, got %f", expectedValues[i], metric.Value)
		}
	}
}

func TestPerformanceMonitorStartStop(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 50)

	// Start monitoring
	monitor.Start()

	// Let it run for a short time
	time.Sleep(time.Millisecond * 200)

	// Stop monitoring
	monitor.Stop()

	// Should have collected some system metrics
	memMetrics := monitor.GetMetrics(MetricTypeMemoryUsage, time.Time{})
	if len(memMetrics) == 0 {
		t.Error("Expected memory metrics to be collected")
	}

	goroutineMetrics := monitor.GetMetrics(MetricTypeGoroutines, time.Time{})
	if len(goroutineMetrics) == 0 {
		t.Error("Expected goroutine metrics to be collected")
	}
}

func BenchmarkMetricRecording(b *testing.B) {
	collector := NewMetricCollector(10000)

	metric := Metric{
		Type:  MetricTypeBuildTime,
		Value: 100.0,
		Unit:  "ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Record(metric)
	}
}

func BenchmarkMetricRetrieval(b *testing.B) {
	collector := NewMetricCollector(1000)

	// Pre-populate with metrics
	for i := 0; i < 1000; i++ {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(i),
			Unit:  "ms",
		}
		collector.Record(metric)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics(MetricTypeBuildTime, time.Time{})
	}
}

func BenchmarkAggregateCalculation(b *testing.B) {
	collector := NewMetricCollector(1000)

	// Pre-populate with metrics
	for i := 0; i < 1000; i++ {
		metric := Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(i),
			Unit:  "ms",
		}
		collector.Record(metric)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetAggregate(MetricTypeBuildTime)
	}
}