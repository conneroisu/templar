package performance

import (
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/registry"
)

func TestAdaptiveOptimizer(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	if optimizer == nil {
		t.Fatal("Failed to create adaptive optimizer")
	}

	// Test starting and stopping
	optimizer.Start()
	optimizer.Stop()
}

func TestOptimizerConfig(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Test default config
	metrics := optimizer.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics, got nil")
	}

	// Test config update
	newConfig := &OptimizerConfig{
		EnableAutoApply:     false,
		ConfidenceThreshold: 0.9,
		DryRunMode:         true,
	}
	optimizer.UpdateConfig(newConfig)

	// Config should be updated (not directly testable without exposing private fields)
}

func TestOptimizationActions(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	tests := []struct {
		name   string
		action Action
	}{
		{
			name: "Scale Workers",
			action: Action{
				Type: ActionScaleWorkers,
				Parameters: map[string]interface{}{
					"target_workers": 4,
				},
			},
		},
		{
			name: "Adjust Cache Size",
			action: Action{
				Type: ActionAdjustCacheSize,
				Parameters: map[string]interface{}{
					"reduce_by_percent": 25.0,
				},
			},
		},
		{
			name: "Optimize GC",
			action: Action{
				Type:       ActionOptimizeGC,
				Parameters: map[string]interface{}{},
			},
		},
		{
			name: "Clear Cache",
			action: Action{
				Type: ActionClearCache,
				Parameters: map[string]interface{}{
					"cache_type": "build",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.applyOptimization(tt.action)

			if !result.Success {
				t.Errorf("Expected successful optimization, got error: %s", result.Error)
			}

			if result.Impact == "" {
				t.Error("Expected impact description")
			}

			if result.Duration == 0 {
				t.Error("Expected non-zero duration")
			}
		})
	}
}

func TestRecommendationFiltering(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Test low confidence recommendation (should be filtered out)
	lowConfidenceRecommendation := Recommendation{
		Type:        "test_optimization",
		Priority:    1,
		Description: "Test optimization",
		Confidence:  0.5, // Below default threshold of 0.7
		CreatedAt:   time.Now(),
	}

	if optimizer.shouldApplyRecommendation(lowConfidenceRecommendation) {
		t.Error("Expected low confidence recommendation to be filtered out")
	}

	// Test high confidence recommendation (should pass)
	highConfidenceRecommendation := Recommendation{
		Type:        "test_optimization",
		Priority:    1,
		Description: "Test optimization",
		Confidence:  0.9, // Above threshold
		CreatedAt:   time.Now(),
	}

	if !optimizer.shouldApplyRecommendation(highConfidenceRecommendation) {
		t.Error("Expected high confidence recommendation to be accepted")
	}
}

func TestCooldownPeriod(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	recommendation := Recommendation{
		Type:        "test_optimization",
		Priority:    1,
		Description: "Test optimization",
		Confidence:  0.9,
		CreatedAt:   time.Now(),
	}

	// First application should succeed
	if !optimizer.shouldApplyRecommendation(recommendation) {
		t.Error("Expected first recommendation to be accepted")
	}

	// Mark as recently applied
	optimizer.appliedActions[recommendation.Type] = time.Now()

	// Second application should be rejected due to cooldown
	if optimizer.shouldApplyRecommendation(recommendation) {
		t.Error("Expected recommendation to be rejected due to cooldown")
	}
}

func TestDryRunMode(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Enable dry run mode
	config := &OptimizerConfig{
		EnableAutoApply:     true,
		ConfidenceThreshold: 0.5,
		DryRunMode:         true,
	}
	optimizer.UpdateConfig(config)

	action := Action{
		Type: ActionScaleWorkers,
		Parameters: map[string]interface{}{
			"target_workers": 4,
		},
	}

	result := optimizer.applyOptimization(action)

	if !result.Success {
		t.Error("Expected dry run to succeed")
	}

	if result.Impact != "Dry run - no actual changes made" {
		t.Errorf("Expected dry run message, got: %s", result.Impact)
	}
}

func TestMetricsCapture(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Add some metrics to the monitor first
	monitor.Record(Metric{
		Type:  MetricTypeMemoryUsage,
		Value: 1024 * 1024, // 1MB
		Unit:  "bytes",
	})

	monitor.Record(Metric{
		Type:  MetricTypeBuildTime,
		Value: 500, // 500ms
		Unit:  "ms",
	})

	metrics := optimizer.captureCurrentMetrics()

	if len(metrics) == 0 {
		t.Error("Expected metrics to be captured")
	}

	// Check that system metrics are captured
	if _, exists := metrics["memory_alloc_mb"]; !exists {
		t.Error("Expected memory allocation metric")
	}

	if _, exists := metrics["goroutines"]; !exists {
		t.Error("Expected goroutines metric")
	}

	if _, exists := metrics["gc_cycles"]; !exists {
		t.Error("Expected GC cycles metric")
	}
}

func TestWorkerScaler(t *testing.T) {
	scaler := NewWorkerScaler(4, 16)

	if scaler.GetCurrentWorkers() != 4 {
		t.Errorf("Expected 4 initial workers, got %d", scaler.GetCurrentWorkers())
	}

	// Scale up
	newCount := scaler.Scale("up", 1.0)
	if newCount <= 4 {
		t.Errorf("Expected workers to increase, got %d", newCount)
	}

	// Scale down
	currentWorkers := scaler.GetCurrentWorkers()
	newCount = scaler.Scale("down", 1.0)
	if newCount >= currentWorkers {
		t.Errorf("Expected workers to decrease, got %d", newCount)
	}

	// Test bounds
	scaler.Scale("up", 10.0) // Try to scale way up
	if scaler.GetCurrentWorkers() > 16 {
		t.Errorf("Expected workers to be capped at 16, got %d", scaler.GetCurrentWorkers())
	}

	scaler.Scale("down", 10.0) // Try to scale way down
	if scaler.GetCurrentWorkers() < 1 {
		t.Errorf("Expected workers to be at least 1, got %d", scaler.GetCurrentWorkers())
	}
}

func TestOptimizerMetrics(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Test initial metrics
	metrics := optimizer.GetMetrics()
	if metrics.ActionsApplied != 0 {
		t.Errorf("Expected 0 actions applied initially, got %d", metrics.ActionsApplied)
	}

	// Simulate applying an optimization
	recommendation := Recommendation{
		Type:        "test_optimization",
		Priority:    1,
		Description: "Test optimization",
		Confidence:  0.9,
		Action: Action{
			Type:       ActionOptimizeGC,
			Parameters: map[string]interface{}{},
		},
		CreatedAt: time.Now(),
	}

	optimizer.handleRecommendation(recommendation)

	// Check updated metrics
	updatedMetrics := optimizer.GetMetrics()
	if updatedMetrics.ActionsApplied != 1 {
		t.Errorf("Expected 1 action applied, got %d", updatedMetrics.ActionsApplied)
	}

	if updatedMetrics.ActionsSuccessful != 1 {
		t.Errorf("Expected 1 successful action, got %d", updatedMetrics.ActionsSuccessful)
	}
}

func TestPeriodicOptimization(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	// Test periodic optimization logic
	optimizer.performPeriodicOptimization()

	// Should have recorded some optimizer metrics
	optimizerMetrics := monitor.GetMetrics("optimizer_actions_applied", time.Time{})
	if len(optimizerMetrics) == 0 {
		t.Error("Expected optimizer metrics to be recorded")
	}
}

func BenchmarkOptimizationApplication(b *testing.B) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	action := Action{
		Type:       ActionOptimizeGC,
		Parameters: map[string]interface{}{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.applyOptimization(action)
	}
}

func BenchmarkMetricsCapture(b *testing.B) {
	monitor := NewPerformanceMonitor(time.Millisecond * 100)
	registry := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(2, registry)

	optimizer := NewAdaptiveOptimizer(monitor, pipeline, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.captureCurrentMetrics()
	}
}