//go:build error_injection
// +build error_injection

package build

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	testingpkg "github.com/conneroisu/templar/internal/testing"
)

// TestBuildPipeline_ErrorInjection demonstrates error injection in the build pipeline
func TestBuildPipeline_ErrorInjection(t *testing.T) {
	// Create error injector and resource tracker
	injector := testingpkg.NewErrorInjector()
	tracker := testingpkg.NewResourceTracker("build_pipeline_error_injection")
	defer tracker.CheckLeaks(t)

	// Set up test registry and pipeline
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(2, reg)

	// Create test component
	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		Package:  "components",
		FilePath: "test.templ",
	}

	// Configure error injection scenarios
	t.Run("File Permission Errors", func(t *testing.T) {
		// Inject file permission errors
		injector.InjectErrorCount("file.read", testingpkg.ErrPermissionDenied, 3)

		// Create a mock compiler that uses error injection
		mockCompiler := &MockTemplCompilerWithInjection{
			injector: injector,
		}
		pipeline.compiler = mockCompiler

		// Start pipeline
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		pipeline.Start(ctx)

		// Queue build tasks that should fail
		for i := 0; i < 3; i++ {
			task := BuildTask{
				Component: component,
				Priority:  1,
				Timestamp: time.Now(),
			}

			select {
			case pipeline.queue.tasks <- task:
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Failed to queue task")
			}
		}

		// Wait for results and verify failures
		failureCount := 0
		for i := 0; i < 3; i++ {
			select {
			case result := <-pipeline.queue.results:
				if result.Error != nil {
					failureCount++
					if !errors.Is(result.Error, testingpkg.ErrPermissionDenied) {
						t.Errorf("Expected permission denied error, got: %v", result.Error)
					}
				}
			case <-time.After(500 * time.Millisecond):
				t.Fatal("Timeout waiting for build result")
			}
		}

		if failureCount != 3 {
			t.Errorf("Expected 3 failures, got %d", failureCount)
		}

		pipeline.Stop()
	})

	t.Run("Command Execution Timeouts", func(t *testing.T) {
		// Clear previous injections
		injector.Clear()

		// Inject command execution timeouts with delay
		injector.InjectErrorWithDelay("exec.command", errors.New("command timeout"), 200*time.Millisecond)

		mockCompiler := &MockTemplCompilerWithInjection{
			injector: injector,
		}
		pipeline.compiler = mockCompiler

		// Start pipeline
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		pipeline.Start(ctx)

		// Queue a task
		task := BuildTask{
			Component: component,
			Priority:  1,
			Timestamp: time.Now(),
		}

		start := time.Now()
		pipeline.queue.tasks <- task

		// Wait for result and verify timeout
		select {
		case result := <-pipeline.queue.results:
			elapsed := time.Since(start)
			if result.Error == nil {
				t.Error("Expected command timeout error")
			}
			if elapsed < 200*time.Millisecond {
				t.Errorf("Expected at least 200ms delay, got %v", elapsed)
			}
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for build result")
		}

		pipeline.Stop()
	})

	t.Run("Resource Exhaustion", func(t *testing.T) {
		// Clear previous injections
		injector.Clear()

		// Inject memory exhaustion errors
		injector.InjectError("memory.alloc", testingpkg.ErrOutOfMemory)

		mockCompiler := &MockTemplCompilerWithInjection{
			injector: injector,
		}
		pipeline.compiler = mockCompiler

		// Start pipeline
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		pipeline.Start(ctx)

		// Queue a task
		task := BuildTask{
			Component: component,
			Priority:  1,
			Timestamp: time.Now(),
		}

		pipeline.queue.tasks <- task

		// Wait for result and verify memory error
		select {
		case result := <-pipeline.queue.results:
			if result.Error == nil {
				t.Error("Expected memory exhaustion error")
			}
			if !errors.Is(result.Error, testingpkg.ErrOutOfMemory) {
				t.Errorf("Expected out of memory error, got: %v", result.Error)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for build result")
		}

		pipeline.Stop()
	})
}

// TestBuildPipeline_ErrorRecovery tests that the pipeline recovers from errors
func TestBuildPipeline_ErrorRecovery(t *testing.T) {
	injector := testingpkg.NewErrorInjector()
	tracker := testingpkg.NewResourceTracker("build_pipeline_recovery")
	defer tracker.CheckLeaks(t)

	// Set up test environment
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(1, reg)

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		Package:  "components",
		FilePath: "test.templ",
	}

	// Configure injector to fail first 2 attempts, then succeed
	injector.InjectErrorCount("file.read", testingpkg.ErrPermissionDenied, 2)

	mockCompiler := &MockTemplCompilerWithInjection{
		injector: injector,
	}
	pipeline.compiler = mockCompiler

	// Start pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pipeline.Start(ctx)

	// Queue 3 tasks
	for i := 0; i < 3; i++ {
		task := BuildTask{
			Component: component,
			Priority:  1,
			Timestamp: time.Now(),
		}
		pipeline.queue.tasks <- task
	}

	// Collect results
	var results []BuildResult
	for i := 0; i < 3; i++ {
		select {
		case result := <-pipeline.queue.results:
			results = append(results, result)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for build results")
		}
	}

	// Verify first 2 failed, third succeeded
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for i := 0; i < 2; i++ {
		if results[i].Error == nil {
			t.Errorf("Expected result %d to fail", i)
		}
	}

	if results[2].Error != nil {
		t.Errorf("Expected result 2 to succeed, got error: %v", results[2].Error)
	}

	pipeline.Stop()
}

// TestBuildPipeline_ConcurrentErrorInjection tests error injection under concurrent load
func TestBuildPipeline_ConcurrentErrorInjection(t *testing.T) {
	injector := testingpkg.NewErrorInjector()
	tracker := testingpkg.NewResourceTracker("build_pipeline_concurrent")
	defer tracker.CheckLeaksWithLimits(t, testingpkg.ResourceLimits{
		MaxGoroutineIncrease: 10, // Allow some variance for worker goroutines
		MaxFileIncrease:      5,
		MaxMemoryIncrease:    10 * 1024 * 1024, // 10MB
		MaxObjectIncrease:    5000,
		TolerancePercent:     0.2, // 20% tolerance
	})

	// Set up test environment
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(4, reg) // More workers for concurrency

	// Configure probabilistic failures
	injector.InjectError("file.read", testingpkg.ErrPermissionDenied).WithProbability(0.3)
	injector.InjectError("exec.command", errors.New("command failed")).WithProbability(0.2)

	mockCompiler := &MockTemplCompilerWithInjection{
		injector: injector,
	}
	pipeline.compiler = mockCompiler

	// Start pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pipeline.Start(ctx)

	// Queue many tasks concurrently
	const numTasks = 50
	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		Package:  "components",
		FilePath: "test.templ",
	}

	// Queue tasks
	for i := 0; i < numTasks; i++ {
		task := BuildTask{
			Component: component,
			Priority:  1,
			Timestamp: time.Now(),
		}

		select {
		case pipeline.queue.tasks <- task:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Failed to queue task %d", i)
		}
	}

	// Collect results
	var successCount, failureCount int
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pipeline.queue.results:
			if result.Error != nil {
				failureCount++
			} else {
				successCount++
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for result %d", i)
		}
	}

	// Verify we got some failures and some successes
	if failureCount == 0 {
		t.Error("Expected some failures due to error injection")
	}
	if successCount == 0 {
		t.Error("Expected some successes")
	}

	t.Logf("Results: %d successes, %d failures", successCount, failureCount)

	pipeline.Stop()

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)
}

// MockTemplCompilerWithInjection simulates a compiler with error injection
type MockTemplCompilerWithInjection struct {
	injector *testingpkg.ErrorInjector
}

func (m *MockTemplCompilerWithInjection) CompileWithPools(component *registry.ComponentInfo, pools *ObjectPools) ([]byte, error) {
	// Check for file read errors
	if err := m.injector.ShouldFail("file.read"); err != nil {
		return nil, err
	}

	// Check for memory allocation errors
	if err := m.injector.ShouldFail("memory.alloc"); err != nil {
		return nil, err
	}

	// Check for command execution errors
	if err := m.injector.ShouldFail("exec.command"); err != nil {
		return nil, err
	}

	// Simulate successful compilation
	return []byte("package components\n// Generated code"), nil
}

func (m *MockTemplCompilerWithInjection) Compile(component *registry.ComponentInfo) ([]byte, error) {
	return m.CompileWithPools(component, NewObjectPools())
}

func (m *MockTemplCompilerWithInjection) validateCommand() error {
	return nil // Mock always validates successfully
}

// TestScenarioManager_BuildPipelineScenarios tests predefined scenarios with build pipeline
func TestScenarioManager_BuildPipelineScenarios(t *testing.T) {
	injector := testingpkg.NewErrorInjector()
	manager := testingpkg.NewScenarioManager(injector)
	tracker := testingpkg.NewResourceTracker("scenario_test")
	defer tracker.CheckLeaks(t)

	// Register and execute build failure scenario
	buildScenario := testingpkg.CreateBuildFailureScenario()
	manager.RegisterScenario(buildScenario)

	err := manager.ExecuteScenario("build_failure")
	if err != nil {
		t.Fatalf("Failed to execute build failure scenario: %v", err)
	}

	// Set up build pipeline with scenario
	reg := registry.NewComponentRegistry()
	pipeline := NewBuildPipeline(2, reg)

	mockCompiler := &MockTemplCompilerWithInjection{
		injector: injector,
	}
	pipeline.compiler = mockCompiler

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		Package:  "components",
		FilePath: "test.templ",
	}

	// Start pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pipeline.Start(ctx)

	// Queue several tasks to trigger scenario steps
	const numTasks = 10
	for i := 0; i < numTasks; i++ {
		task := BuildTask{
			Component: component,
			Priority:  1,
			Timestamp: time.Now(),
		}
		pipeline.queue.tasks <- task
	}

	// Collect results and verify some failed according to scenario
	var results []BuildResult
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pipeline.queue.results:
			results = append(results, result)
		case <-time.After(time.Second):
			t.Fatalf("Timeout waiting for result %d", i)
		}
	}

	// Count failures
	failureCount := 0
	for _, result := range results {
		if result.Error != nil {
			failureCount++
		}
	}

	if failureCount == 0 {
		t.Error("Expected some failures from build failure scenario")
	}

	t.Logf("Build failure scenario: %d/%d tasks failed", failureCount, numTasks)

	pipeline.Stop()
}
