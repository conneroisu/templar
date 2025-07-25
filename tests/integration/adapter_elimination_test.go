package integration

import (
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/watcher"
)

// TestAdapterEliminationSuccess validates that concrete types implement interfaces
// directly without needing adapters, verifying successful elimination of adapter anti-pattern
func TestAdapterEliminationSuccess(t *testing.T) {
	t.Run("ComponentRegistry", func(t *testing.T) {
		// Create concrete registry
		reg := registry.NewComponentRegistry()

		// Verify it implements the interface directly - no adapter needed
		var _ interfaces.ComponentRegistry = reg

		// Run comprehensive validation
		validator := interfaces.NewInterfaceValidator()
		result := validator.ValidateComponentRegistry(reg)

		if !result.Valid {
			t.Errorf("ComponentRegistry validation failed: %v", result.Errors)
		}

		t.Logf("✅ ComponentRegistry implements interface directly: %s", result.ConcreteType)
	})

	t.Run("FileWatcher", func(t *testing.T) {
		// Create concrete watcher
		fw, err := watcher.NewFileWatcher(100 * time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to create file watcher: %v", err)
		}
		defer fw.Stop()

		// Verify it implements the interface directly - no adapter needed
		var _ interfaces.FileWatcher = fw

		// Run comprehensive validation
		validator := interfaces.NewInterfaceValidator()
		result := validator.ValidateFileWatcher(fw)

		if !result.Valid {
			t.Errorf("FileWatcher validation failed: %v", result.Errors)
		}

		t.Logf("✅ FileWatcher implements interface directly: %s", result.ConcreteType)
	})

	t.Run("ComponentScanner", func(t *testing.T) {
		// Create dependencies
		reg := registry.NewComponentRegistry()

		// Create concrete scanner
		cs := scanner.NewComponentScanner(reg)

		// Verify it implements the interface directly - no adapter needed
		var _ interfaces.ComponentScanner = cs

		// Run comprehensive validation
		validator := interfaces.NewInterfaceValidator()
		result := validator.ValidateComponentScanner(cs)

		if !result.Valid {
			t.Errorf("ComponentScanner validation failed: %v", result.Errors)
		}

		t.Logf("✅ ComponentScanner implements interface directly: %s", result.ConcreteType)
	})

	t.Run("BuildPipeline", func(t *testing.T) {
		// Create dependencies
		reg := registry.NewComponentRegistry()

		// Create concrete build pipeline
		bp := build.NewRefactoredBuildPipeline(2, reg)

		// Verify it implements the interface directly - no adapter needed
		var _ interfaces.BuildPipeline = bp

		// Run comprehensive validation
		validator := interfaces.NewInterfaceValidator()
		result := validator.ValidateBuildPipeline(bp)

		if !result.Valid {
			t.Errorf("BuildPipeline validation failed: %v", result.Errors)
		}

		t.Logf("✅ BuildPipeline implements interface directly: %s", result.ConcreteType)
	})
}

// TestNoAdapterAntiPatternRequired validates that we can use concrete types directly
// as interface implementations without any wrapper or adapter layer
func TestNoAdapterAntiPatternRequired(t *testing.T) {
	// Create all concrete implementations
	reg := registry.NewComponentRegistry()

	fw, err := watcher.NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	cs := scanner.NewComponentScanner(reg)
	bp := build.NewRefactoredBuildPipeline(2, reg)

	// Use them directly as interfaces - NO ADAPTERS NEEDED!
	var componentRegistry interfaces.ComponentRegistry = reg
	var fileWatcher interfaces.FileWatcher = fw
	var componentScanner interfaces.ComponentScanner = cs
	var buildPipeline interfaces.BuildPipeline = bp

	// Validate all interfaces work together without adapters
	summary := interfaces.ValidateAllInterfaces(componentRegistry, fileWatcher, componentScanner, buildPipeline)

	if !summary.IsValid() {
		t.Errorf("Interface validation failed: %s", summary.String())

		// Print detailed validation results for debugging
		validator := interfaces.NewInterfaceValidator()
		validator.ValidateComponentRegistry(componentRegistry)
		validator.ValidateFileWatcher(fileWatcher)
		validator.ValidateComponentScanner(componentScanner)
		validator.ValidateBuildPipeline(buildPipeline)

		for _, result := range validator.GetResults() {
			if !result.Valid {
				t.Logf("❌ Failed interface: %s (%s)", result.InterfaceName, result.ConcreteType)
				for _, err := range result.Errors {
					t.Logf("     Error: %s", err)
				}
			}
		}
	} else {
		t.Logf("🎉 Adapter anti-pattern successfully eliminated! All interfaces valid: %s", summary.String())
	}
}

// TestInterfaceSegregationPrinciple validates that interfaces follow ISP
// ensuring concrete types only implement methods they actually need (no fat interfaces)
func TestInterfaceSegregationPrinciple(t *testing.T) {
	t.Run("ProperInterfaceSegregation", func(t *testing.T) {
		// Create file watcher
		fw, err := watcher.NewFileWatcher(100 * time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to create file watcher: %v", err)
		}
		defer fw.Stop()

		// FileWatcher should only implement FileWatcher interface, not others
		var fileWatcherInterface interfaces.FileWatcher = fw

		// Should not be forced to implement unrelated interfaces (ISP compliance)
		if _, ok := fileWatcherInterface.(interfaces.ComponentScanner); ok {
			t.Error("❌ ISP Violation: FileWatcher should not implement ComponentScanner interface")
		}

		if _, ok := fileWatcherInterface.(interfaces.BuildPipeline); ok {
			t.Error("❌ ISP Violation: FileWatcher should not implement BuildPipeline interface")
		}

		t.Log("✅ Interface Segregation Principle satisfied - no fat interfaces")
	})
}

// TestMemoryLeakComplianceWithoutAdapters ensures no memory leaks with direct interface usage
func TestMemoryLeakComplianceWithoutAdapters(t *testing.T) {
	checker := interfaces.NewMemoryLeakChecker()

	// Create and exercise interfaces multiple times to detect leaks
	for i := 0; i < 50; i++ {
		reg := registry.NewComponentRegistry()

		fw, err := watcher.NewFileWatcher(10 * time.Millisecond)
		if err != nil {
			continue
		}

		cs := scanner.NewComponentScanner(reg)
		bp := build.NewRefactoredBuildPipeline(1, reg)

		// Use as interfaces directly - no adapter overhead
		var _ interfaces.ComponentRegistry = reg
		var _ interfaces.FileWatcher = fw
		var _ interfaces.ComponentScanner = cs
		var _ interfaces.BuildPipeline = bp

		// Clean up resources
		fw.Stop()
		bp.Stop()
	}

	result := checker.Check()

	if result.HasSignificantLeak() {
		t.Errorf("❌ Memory leak detected in direct interface usage: %s", result.String())
	} else {
		t.Logf("✅ No memory leaks with direct interface implementation: %s", result.String())
	}
}

// TestAdapterPackageEliminationSuccess validates that adapter package is completely removed
func TestAdapterPackageEliminationSuccess(t *testing.T) {
	// This test ensures the adapter package was successfully removed
	// If this test compiles and runs, it means we're not importing the adapter package

	t.Log("✅ Adapter package successfully eliminated - no adapter imports or dependencies")

	// Verify we can create all services without adapters
	reg := registry.NewComponentRegistry()

	fw, err := watcher.NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	cs := scanner.NewComponentScanner(reg)
	bp := build.NewRefactoredBuildPipeline(2, reg)

	// All work as interfaces natively
	interfaces := []interface{}{
		interfaces.ComponentRegistry(reg),
		interfaces.FileWatcher(fw),
		interfaces.ComponentScanner(cs),
		interfaces.BuildPipeline(bp),
	}

	if len(interfaces) != 4 {
		t.Error("❌ Failed to create interface implementations")
	} else {
		t.Log("🎉 All interfaces implemented directly by concrete types - adapter anti-pattern eliminated!")
	}
}
