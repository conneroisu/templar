package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// TestQueueOverflowProtection tests that the pipeline handles queue overflow gracefully
func TestQueueOverflowProtection(t *testing.T) {
	tempDir := t.TempDir()

	// Create a small queue to test overflow
	pipeline := &BuildPipeline{
		cache:    NewBuildCache(1024*1024, time.Hour),
		compiler: NewTemplCompiler(),
		queue: &BuildQueue{
			tasks:    make(chan BuildTask, 2),   // Very small queue
			results:  make(chan BuildResult, 2), // Very small results queue
			priority: make(chan BuildTask, 1),   // Tiny priority queue
		},
		metrics:     NewBuildMetrics(),
		callbacks:   make([]BuildCallback, 0),
		objectPools: NewObjectPools(),
		slicePools:  NewSlicePools(),
		workerPool:  NewWorkerPool(),
		workers:     1, // Single worker to slow processing
	}

	// Create test components
	numComponents := 10 // More than queue capacity
	components := make([]*types.ComponentInfo, numComponents)

	for i := 0; i < numComponents; i++ {
		componentName := fmt.Sprintf("OverflowComponent_%d", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("%s.templ", componentName))

		components[i] = &types.ComponentInfo{
			FilePath: filePath,
			Name:     componentName,
			Package:  "test",
		}
	}

	// Note: In a real test, we might capture output differently,
	// but for this test we'll rely on metrics to verify behavior

	// Test regular queue overflow
	t.Run("RegularQueueOverflow", func(t *testing.T) {
		// Try to queue more tasks than capacity
		for i := 0; i < numComponents; i++ {
			pipeline.Build(components[i])
		}

		// Check metrics for dropped tasks
		droppedTasks, _, dropReasons := pipeline.metrics.GetQueueHealthStatus()

		if droppedTasks == 0 {
			t.Errorf("Expected some tasks to be dropped due to queue overflow, but got 0")
		}

		// Check that we have drop reasons recorded
		if len(dropReasons) == 0 {
			t.Errorf("Expected drop reasons to be recorded, but got none")
		}

		// Check for expected drop reasons
		if _, exists := dropReasons["task_queue_full"]; !exists {
			t.Errorf("Expected 'task_queue_full' drop reason, got reasons: %v", dropReasons)
		}

		t.Logf("Dropped %d tasks with reasons: %v", droppedTasks, dropReasons)
	})

	// Test priority queue overflow
	t.Run("PriorityQueueOverflow", func(t *testing.T) {
		// Reset metrics
		pipeline.metrics.Reset()

		// Try to queue more priority tasks than capacity
		for i := 0; i < 5; i++ {
			pipeline.BuildWithPriority(components[i])
		}

		// Check metrics for dropped tasks
		droppedTasks, _, dropReasons := pipeline.metrics.GetQueueHealthStatus()

		if droppedTasks == 0 {
			t.Errorf("Expected some priority tasks to be dropped, but got 0")
		}

		// Check for priority queue drop reason
		if _, exists := dropReasons["priority_queue_full"]; !exists {
			t.Errorf("Expected 'priority_queue_full' drop reason, got reasons: %v", dropReasons)
		}

		t.Logf("Dropped %d priority tasks with reasons: %v", droppedTasks, dropReasons)
	})

}

// TestResultsQueueOverflowProtection tests that results queue overflow is handled
func TestResultsQueueOverflowProtection(t *testing.T) {
	tempDir := t.TempDir()

	// Create pipeline with tiny results queue
	pipeline := &BuildPipeline{
		cache:    NewBuildCache(1024*1024, time.Hour),
		compiler: NewTemplCompiler(),
		queue: &BuildQueue{
			tasks:    make(chan BuildTask, 10),
			results:  make(chan BuildResult, 1), // Very small results queue
			priority: make(chan BuildTask, 5),
		},
		metrics:     NewBuildMetrics(),
		callbacks:   make([]BuildCallback, 0),
		objectPools: NewObjectPools(),
		slicePools:  NewSlicePools(),
		workerPool:  NewWorkerPool(),
		workers:     4, // Multiple workers to generate results quickly
	}

	// Start the pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline.Start(ctx)
	defer pipeline.Stop()

	// Create test components
	numComponents := 5
	components := make([]*types.ComponentInfo, numComponents)

	for i := 0; i < numComponents; i++ {
		componentName := fmt.Sprintf("ResultsComponent_%d", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("%s.templ", componentName))

		// Create actual files for successful compilation
		content := fmt.Sprintf(`package test

templ %s() {
	<div>Test content %d</div>
}`, componentName, i)

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		components[i] = &types.ComponentInfo{
			FilePath: filePath,
			Name:     componentName,
			Package:  "test",
		}
	}

	// Note: We'll rely on metrics instead of capturing output

	// Queue builds rapidly to overwhelm results queue
	for i := 0; i < numComponents; i++ {
		pipeline.Build(components[i])
	}

	// Wait for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Check metrics for dropped results
	_, droppedResults, dropReasons := pipeline.metrics.GetQueueHealthStatus()

	// May or may not have dropped results depending on timing, but should not hang
	t.Logf("Dropped %d results with reasons: %v", droppedResults, dropReasons)

	// Check that system is still responsive (no deadlocks)
	testComponent := &types.ComponentInfo{
		FilePath: filepath.Join(tempDir, "responsive_test.templ"),
		Name:     "ResponsiveTest",
		Package:  "test",
	}

	if err := os.WriteFile(testComponent.FilePath, []byte(`package test
templ ResponsiveTest() {
	<div>Responsive test</div>
}`), 0644); err != nil {
		t.Fatalf("Failed to create responsive test file: %v", err)
	}

	// This should not hang even if queues were previously full
	pipeline.Build(testComponent)

	t.Log("System remained responsive after queue overflow scenarios")
}

// TestQueueHealthMetrics validates that queue health metrics work correctly
func TestQueueHealthMetrics(t *testing.T) {
	metrics := NewBuildMetrics()

	// Test initial state
	droppedTasks, droppedResults, dropReasons := metrics.GetQueueHealthStatus()
	if droppedTasks != 0 || droppedResults != 0 || len(dropReasons) != 0 {
		t.Errorf("Expected empty initial metrics, got tasks=%d, results=%d, reasons=%v",
			droppedTasks, droppedResults, dropReasons)
	}

	// Test recording dropped tasks
	metrics.RecordDroppedTask("Component1", "task_queue_full")
	metrics.RecordDroppedTask("Component2", "priority_queue_full")
	metrics.RecordDroppedTask("Component3", "task_queue_full")

	// Test recording dropped results
	metrics.RecordDroppedResult("Component4", "results_queue_full")
	metrics.RecordDroppedResult("Component5", "results_queue_full_cache_hit")

	// Check metrics
	droppedTasks, droppedResults, dropReasons = metrics.GetQueueHealthStatus()

	if droppedTasks != 3 {
		t.Errorf("Expected 3 dropped tasks, got %d", droppedTasks)
	}

	if droppedResults != 2 {
		t.Errorf("Expected 2 dropped results, got %d", droppedResults)
	}

	expectedReasons := map[string]int64{
		"task_queue_full":              2,
		"priority_queue_full":          1,
		"results_queue_full":           1,
		"results_queue_full_cache_hit": 1,
	}

	for reason, expectedCount := range expectedReasons {
		if count, exists := dropReasons[reason]; !exists || count != expectedCount {
			t.Errorf("Expected drop reason '%s' with count %d, got count %d",
				reason, expectedCount, count)
		}
	}

	// Test reset
	metrics.Reset()
	droppedTasks, droppedResults, dropReasons = metrics.GetQueueHealthStatus()

	if droppedTasks != 0 || droppedResults != 0 || len(dropReasons) != 0 {
		t.Errorf("Expected empty metrics after reset, got tasks=%d, results=%d, reasons=%v",
			droppedTasks, droppedResults, dropReasons)
	}
}
