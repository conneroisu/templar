package interfaces_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/watcher"
)

// TestComponentRegistryInterface validates that concrete registry implements ComponentRegistry interface
func TestComponentRegistryInterface(t *testing.T) {
	// Create concrete registry
	concreteRegistry := registry.NewComponentRegistry()

	// Verify it implements the interface
	var iface interfaces.ComponentRegistry = concreteRegistry
	if iface == nil {
		t.Fatal("Registry does not implement ComponentRegistry interface")
	}

	// Test basic operations
	testComponent := &types.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/test/path.templ",
		Package:  "test",
	}

	// Test Register
	iface.Register(testComponent)

	// Test Get
	retrieved, exists := iface.Get("TestComponent")
	if !exists {
		t.Error("Component should exist after registration")
	}
	if retrieved.Name != "TestComponent" {
		t.Errorf("Retrieved component name mismatch: got %s, want TestComponent", retrieved.Name)
	}

	// Test Count
	if count := iface.Count(); count != 1 {
		t.Errorf("Component count mismatch: got %d, want 1", count)
	}

	// Test GetAll
	all := iface.GetAll()
	if len(all) != 1 {
		t.Errorf("GetAll count mismatch: got %d, want 1", len(all))
	}

	// Test Watch channel
	ch := iface.Watch()
	if ch == nil {
		t.Error("Watch channel should not be nil")
	}

	// Test DetectCircularDependencies
	cycles := iface.DetectCircularDependencies()
	// Should return a slice (either empty or with cycles), not nil
	_ = cycles
}

// TestFileWatcherInterface validates that concrete watcher implements FileWatcher interface
func TestFileWatcherInterface(t *testing.T) {
	// Create concrete watcher
	concreteWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer concreteWatcher.Stop()

	// Use concrete type directly and verify interface implementation
	var iface interfaces.FileWatcher = concreteWatcher
	if iface == nil {
		t.Fatal("Watcher does not implement FileWatcher interface")
	}

	// Test filter addition
	testFilter := interfaces.FileFilterFunc(func(path string) bool {
		return true
	})
	iface.AddFilter(testFilter)

	// Test handler addition
	testHandler := func(events []interface{}) error {
		return nil
	}
	iface.AddHandler(testHandler)

	// Test Start/Stop
	ctx := context.Background()
	if err := iface.Start(ctx); err != nil {
		t.Errorf("Failed to start watcher: %v", err)
	}

	iface.Stop()
}

// TestComponentScannerInterface validates that concrete scanner implements ComponentScanner interface
func TestComponentScannerInterface(t *testing.T) {
	// Create dependencies
	reg := registry.NewComponentRegistry()

	// Create concrete scanner
	concreteScanner := scanner.NewComponentScanner(reg)

	// Use concrete type directly and verify interface implementation
	var iface interfaces.ComponentScanner = concreteScanner
	if iface == nil {
		t.Fatal("Scanner does not implement ComponentScanner interface")
	}

	// Test ScanFile (should not panic even with non-existent file)
	err := iface.ScanFile("/non/existent/file.templ")
	// Error is expected for non-existent file, but shouldn't panic
	if err == nil {
		t.Log("ScanFile returned no error for non-existent file (this might be expected)")
	}

	// Test ScanDirectory (should not panic even with non-existent directory)
	err = iface.ScanDirectory("/non/existent/directory")
	// Error is expected for non-existent directory, but shouldn't panic
	if err == nil {
		t.Log("ScanDirectory returned no error for non-existent directory (this might be expected)")
	}
}

// TestBuildPipelineInterface validates that concrete build pipeline implements BuildPipeline interface
func TestBuildPipelineInterface(t *testing.T) {
	// Create dependencies
	reg := registry.NewComponentRegistry()

	// Create concrete build pipeline
	concretePipeline := build.NewRefactoredBuildPipeline(2, reg)

	// Use concrete type directly and verify interface implementation
	var iface interfaces.BuildPipeline = concretePipeline
	if iface == nil {
		t.Fatal("Build pipeline does not implement BuildPipeline interface")
	}

	// Test Start/Stop
	ctx := context.Background()
	iface.Start(ctx)
	defer iface.Stop()

	// Test callback addition
	testCallback := func(result interface{}) {
		// Callback logic would go here
	}
	iface.AddCallback(testCallback)

	// Test component building
	testComponent := &types.ComponentInfo{
		Name:     "TestBuildComponent",
		FilePath: "/test/build.templ",
		Package:  "test",
	}

	iface.Build(testComponent)
	iface.BuildWithPriority(testComponent)

	// Test metrics (should not panic)
	metrics := iface.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should return non-nil result")
	}

	// Test cache operations
	cache := iface.GetCache()
	if cache == nil {
		t.Error("GetCache should return non-nil result")
	}

	iface.ClearCache()
}

// TestFullInterfaceIntegration tests the complete interface ecosystem
func TestFullInterfaceIntegration(t *testing.T) {
	// Create concrete implementations
	reg := registry.NewComponentRegistry()

	concreteWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer concreteWatcher.Stop()

	concreteScanner := scanner.NewComponentScanner(reg)
	concretePipeline := build.NewRefactoredBuildPipeline(2, reg)

	// Use concrete types directly
	watcherAdapter := concreteWatcher
	scannerAdapter := concreteScanner
	pipelineAdapter := concretePipeline

	// Validate all interfaces
	summary := interfaces.ValidateAllInterfaces(reg, watcherAdapter, scannerAdapter, pipelineAdapter)

	if !summary.IsValid() {
		t.Errorf("Interface validation failed: %s", summary.String())

		// Get detailed results for debugging
		validator := interfaces.NewInterfaceValidator()
		validator.ValidateComponentRegistry(reg)
		validator.ValidateFileWatcher(watcherAdapter)
		validator.ValidateComponentScanner(scannerAdapter)
		validator.ValidateBuildPipeline(pipelineAdapter)

		for _, result := range validator.GetResults() {
			if !result.Valid {
				t.Logf("Failed interface: %s (%s)", result.InterfaceName, result.ConcreteType)
				for _, err := range result.Errors {
					t.Logf("  Error: %s", err)
				}
				for _, warn := range result.Warnings {
					t.Logf("  Warning: %s", warn)
				}
			}
		}
	}

	t.Logf("Interface validation summary: %s", summary.String())
}

// TestInterfaceWorkflow tests a complete workflow using only interfaces
func TestInterfaceWorkflow(t *testing.T) {
	// Create concrete implementations
	reg := registry.NewComponentRegistry()

	concreteWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer concreteWatcher.Stop()

	concretePipeline := build.NewRefactoredBuildPipeline(2, reg)

	// Use only interfaces from this point forward
	var registry interfaces.ComponentRegistry = reg
	var fileWatcher interfaces.FileWatcher = concreteWatcher
	var buildPipeline interfaces.BuildPipeline = concretePipeline

	// Test workflow
	ctx := context.Background()

	// Start build pipeline
	buildPipeline.Start(ctx)
	defer buildPipeline.Stop()

	// Register a test component
	testComponent := &types.ComponentInfo{
		Name:     "WorkflowTest",
		FilePath: "/test/workflow.templ",
		Package:  "test",
	}
	registry.Register(testComponent)

	// Verify registration
	retrieved, exists := registry.Get("WorkflowTest")
	if !exists {
		t.Error("Component should exist after registration")
	}
	if retrieved.Name != "WorkflowTest" {
		t.Errorf("Retrieved component name mismatch: got %s, want WorkflowTest", retrieved.Name)
	}

	// Test build pipeline
	buildPipeline.Build(testComponent)
	buildPipeline.BuildWithPriority(testComponent)

	// Test metrics
	metrics := buildPipeline.GetMetrics()
	if metrics == nil {
		t.Error("Build metrics should not be nil")
	}

	// Test cache
	cache := buildPipeline.GetCache()
	if cache == nil {
		t.Error("Build cache should not be nil")
	}

	buildPipeline.ClearCache()

	// Test file watcher setup
	testFilter := interfaces.FileFilterFunc(func(path string) bool {
		return path != ""
	})
	fileWatcher.AddFilter(testFilter)

	testHandler := func(events []interface{}) error {
		return nil
	}
	fileWatcher.AddHandler(testHandler)

	// Start and stop watcher
	startCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := fileWatcher.Start(startCtx); err != nil {
		t.Errorf("Failed to start file watcher: %v", err)
	}
	fileWatcher.Stop()

	t.Log("Interface workflow test completed successfully")
}

// TestConcurrentInterfaceAccess tests concurrent access to interfaces
func TestConcurrentInterfaceAccess(t *testing.T) {
	reg := registry.NewComponentRegistry()
	concretePipeline := build.NewRefactoredBuildPipeline(4, reg)
	pipelineAdapter := concretePipeline

	ctx := context.Background()
	pipelineAdapter.Start(ctx)
	defer pipelineAdapter.Stop()

	// Test concurrent registry access
	t.Run("ConcurrentRegistry", func(t *testing.T) {
		done := make(chan bool, 10)

		// Launch 10 goroutines that register components concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < 100; j++ {
					testComponent := &types.ComponentInfo{
						Name:     fmt.Sprintf("Concurrent_%d_%d", id, j),
						FilePath: fmt.Sprintf("/test/concurrent_%d_%d.templ", id, j),
						Package:  "test",
					}

					reg.Register(testComponent)

					if retrieved, exists := reg.Get(testComponent.Name); !exists || retrieved.Name != testComponent.Name {
						t.Errorf("Concurrent access error for component %s", testComponent.Name)
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	// Test concurrent build pipeline access
	t.Run("ConcurrentBuildPipeline", func(t *testing.T) {
		done := make(chan bool, 5)

		// Launch 5 goroutines that build components concurrently
		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < 50; j++ {
					testComponent := &types.ComponentInfo{
						Name:     fmt.Sprintf("Build_%d_%d", id, j),
						FilePath: fmt.Sprintf("/test/build_%d_%d.templ", id, j),
						Package:  "test",
					}

					pipelineAdapter.Build(testComponent)
					if j%10 == 0 {
						pipelineAdapter.BuildWithPriority(testComponent)
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	t.Log("Concurrent interface access test completed")
}

// TestInterfaceContractCompliance verifies that interfaces maintain their contracts
func TestInterfaceContractCompliance(t *testing.T) {
	reg := registry.NewComponentRegistry()

	// Test that Watch() returns valid channels (they can be different for multiple watchers)
	ch1 := reg.Watch()
	ch2 := reg.Watch()

	if ch1 == nil || ch2 == nil {
		t.Error("ComponentRegistry.Watch() should return non-nil channels")
	}

	// Test that Count() is consistent with GetAll()
	initialCount := reg.Count()
	initialAll := reg.GetAll()

	if initialCount != len(initialAll) {
		t.Errorf("Count() (%d) does not match len(GetAll()) (%d)", initialCount, len(initialAll))
	}

	// Add a component and verify consistency
	testComponent := &types.ComponentInfo{
		Name:     "ContractTest",
		FilePath: "/test/contract.templ",
		Package:  "test",
	}
	reg.Register(testComponent)

	newCount := reg.Count()
	newAll := reg.GetAll()

	if newCount != len(newAll) {
		t.Errorf("After registration: Count() (%d) does not match len(GetAll()) (%d)", newCount, len(newAll))
	}

	if newCount != initialCount+1 {
		t.Errorf("Count should increase by 1 after registration: got %d, want %d", newCount, initialCount+1)
	}

	// Test that Get() returns what was registered
	retrieved, exists := reg.Get("ContractTest")
	if !exists {
		t.Error("Get() should return true for registered component")
	}
	if retrieved.Name != "ContractTest" {
		t.Errorf("Get() returned wrong component: got %s, want ContractTest", retrieved.Name)
	}

	t.Log("Interface contract compliance test completed")
}
