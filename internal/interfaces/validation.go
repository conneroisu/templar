package interfaces

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// ValidationResult represents the result of interface validation
type ValidationResult struct {
	Valid         bool
	InterfaceName string
	ConcreteType  string
	Errors        []string
	Warnings      []string
}

// InterfaceValidator provides runtime validation of interface implementations
type InterfaceValidator struct {
	results []ValidationResult
}

// NewInterfaceValidator creates a new interface validator
func NewInterfaceValidator() *InterfaceValidator {
	return &InterfaceValidator{
		results: make([]ValidationResult, 0),
	}
}

// ValidateComponentRegistry validates a ComponentRegistry implementation
func (v *InterfaceValidator) ValidateComponentRegistry(reg ComponentRegistry) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		InterfaceName: "ComponentRegistry",
		ConcreteType:  reflect.TypeOf(reg).String(),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// Test basic functionality
	testComponent := &types.ComponentInfo{
		Name:     "ValidationTest",
		FilePath: "/test/validation.templ",
		Package:  "test",
	}

	// Test Register - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Register method panicked: %v", r))
			}
		}()
		reg.Register(testComponent)
	}()

	// Test Get - should return the registered component
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Get method panicked: %v", r))
			}
		}()

		retrieved, exists := reg.Get("ValidationTest")
		if !exists {
			result.Warnings = append(result.Warnings, "Get method did not find registered component")
		} else if retrieved == nil {
			result.Errors = append(result.Errors, "Get method returned nil component")
			result.Valid = false
		} else if retrieved.Name != "ValidationTest" {
			result.Errors = append(result.Errors, "Get method returned wrong component")
			result.Valid = false
		}
	}()

	// Test Count - should be non-negative
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Count method panicked: %v", r))
			}
		}()

		count := reg.Count()
		if count < 0 {
			result.Errors = append(result.Errors, "Count method returned negative value")
			result.Valid = false
		}
	}()

	// Test GetAll - should not return nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("GetAll method panicked: %v", r))
			}
		}()

		all := reg.GetAll()
		if all == nil {
			result.Errors = append(result.Errors, "GetAll method returned nil")
			result.Valid = false
		}
	}()

	// Test Watch - should return non-nil channel
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Watch method panicked: %v", r))
			}
		}()

		ch := reg.Watch()
		if ch == nil {
			result.Errors = append(result.Errors, "Watch method returned nil channel")
			result.Valid = false
		}
	}()

	// Test DetectCircularDependencies - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("DetectCircularDependencies method panicked: %v", r))
			}
		}()

		cycles := reg.DetectCircularDependencies()
		if cycles == nil {
			result.Warnings = append(result.Warnings, "DetectCircularDependencies returned nil (should return empty slice)")
		}
	}()

	v.results = append(v.results, result)
	return result
}

// ValidateFileWatcher validates a FileWatcher implementation
func (v *InterfaceValidator) ValidateFileWatcher(watcher FileWatcher) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		InterfaceName: "FileWatcher",
		ConcreteType:  reflect.TypeOf(watcher).String(),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// Test AddFilter - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("AddFilter method panicked: %v", r))
			}
		}()

		testFilter := FileFilterFunc(func(string) bool { return true })
		watcher.AddFilter(testFilter)
	}()

	// Test AddHandler - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("AddHandler method panicked: %v", r))
			}
		}()

		testHandler := func([]interface{}) error { return nil }
		watcher.AddHandler(testHandler)
	}()

	// Test AddPath - should not panic (even with invalid path)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("AddPath method panicked: %v", r))
			}
		}()

		_ = watcher.AddPath("/nonexistent/path")
		// Error is expected, but should not panic
	}()

	// Test AddRecursive - should not panic (even with invalid path)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("AddRecursive method panicked: %v", r))
			}
		}()

		_ = watcher.AddRecursive("/nonexistent/path")
		// Error is expected, but should not panic
	}()

	// Test Start/Stop - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Start/Stop methods panicked: %v", r))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = watcher.Start(ctx)
		watcher.Stop()
	}()

	v.results = append(v.results, result)
	return result
}

// ValidateComponentScanner validates a ComponentScanner implementation
func (v *InterfaceValidator) ValidateComponentScanner(scanner ComponentScanner) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		InterfaceName: "ComponentScanner",
		ConcreteType:  reflect.TypeOf(scanner).String(),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// Test ScanFile - should not panic (even with invalid file)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("ScanFile method panicked: %v", r))
			}
		}()

		_ = scanner.ScanFile("/nonexistent/file.templ")
		// Error is expected, but should not panic
	}()

	// Test ScanDirectory - should not panic (even with invalid directory)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("ScanDirectory method panicked: %v", r))
			}
		}()

		_ = scanner.ScanDirectory("/nonexistent/directory")
		// Error is expected, but should not panic
	}()

	v.results = append(v.results, result)
	return result
}

// ValidateBuildPipeline validates a BuildPipeline implementation
func (v *InterfaceValidator) ValidateBuildPipeline(pipeline BuildPipeline) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		InterfaceName: "BuildPipeline",
		ConcreteType:  reflect.TypeOf(pipeline).String(),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	testComponent := &types.ComponentInfo{
		Name:     "BuildValidationTest",
		FilePath: "/test/build.templ",
		Package:  "test",
	}

	// Test Start/Stop - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Start/Stop methods panicked: %v", r))
			}
		}()

		ctx := context.Background()
		pipeline.Start(ctx)
		defer pipeline.Stop()
	}()

	// Test AddCallback - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("AddCallback method panicked: %v", r))
			}
		}()

		testCallback := func(interface{}) {}
		pipeline.AddCallback(testCallback)
	}()

	// Test Build - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Build method panicked: %v", r))
			}
		}()

		pipeline.Build(testComponent)
	}()

	// Test BuildWithPriority - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("BuildWithPriority method panicked: %v", r))
			}
		}()

		pipeline.BuildWithPriority(testComponent)
	}()

	// Test GetMetrics - should return non-nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("GetMetrics method panicked: %v", r))
			}
		}()

		metrics := pipeline.GetMetrics()
		if metrics == nil {
			result.Errors = append(result.Errors, "GetMetrics returned nil")
			result.Valid = false
		}
	}()

	// Test GetCache - should return non-nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("GetCache method panicked: %v", r))
			}
		}()

		cache := pipeline.GetCache()
		if cache == nil {
			result.Errors = append(result.Errors, "GetCache returned nil")
			result.Valid = false
		}
	}()

	// Test ClearCache - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("ClearCache method panicked: %v", r))
			}
		}()

		pipeline.ClearCache()
	}()

	v.results = append(v.results, result)
	return result
}

// ValidateFileFilter validates a FileFilter implementation
func (v *InterfaceValidator) ValidateFileFilter(filter FileFilter) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		InterfaceName: "FileFilter",
		ConcreteType:  reflect.TypeOf(filter).String(),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// Test ShouldInclude - should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("ShouldInclude method panicked: %v", r))
			}
		}()

		_ = filter.ShouldInclude("/test/path")
		_ = filter.ShouldInclude("")
		_ = filter.ShouldInclude("/very/long/path/that/might/cause/issues.templ")
	}()

	v.results = append(v.results, result)
	return result
}

// GetResults returns all validation results
func (v *InterfaceValidator) GetResults() []ValidationResult {
	return v.results
}

// GetSummary returns a summary of validation results
func (v *InterfaceValidator) GetSummary() ValidationSummary {
	summary := ValidationSummary{
		TotalInterfaces: len(v.results),
		ValidInterfaces: 0,
		TotalErrors:     0,
		TotalWarnings:   0,
	}

	for _, result := range v.results {
		if result.Valid {
			summary.ValidInterfaces++
		}
		summary.TotalErrors += len(result.Errors)
		summary.TotalWarnings += len(result.Warnings)
	}

	return summary
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalInterfaces int
	ValidInterfaces int
	TotalErrors     int
	TotalWarnings   int
}

// IsValid returns true if all interfaces are valid
func (s ValidationSummary) IsValid() bool {
	return s.ValidInterfaces == s.TotalInterfaces
}

// String returns a string representation of the summary
func (s ValidationSummary) String() string {
	return fmt.Sprintf("Interfaces: %d/%d valid, Errors: %d, Warnings: %d",
		s.ValidInterfaces, s.TotalInterfaces, s.TotalErrors, s.TotalWarnings)
}

// ValidateAllInterfaces performs comprehensive validation of all interfaces
func ValidateAllInterfaces(
	registry ComponentRegistry,
	watcher FileWatcher,
	scanner ComponentScanner,
	pipeline BuildPipeline,
) ValidationSummary {
	validator := NewInterfaceValidator()

	if registry != nil {
		validator.ValidateComponentRegistry(registry)
	}

	if watcher != nil {
		validator.ValidateFileWatcher(watcher)
	}

	if scanner != nil {
		validator.ValidateComponentScanner(scanner)
	}

	if pipeline != nil {
		validator.ValidateBuildPipeline(pipeline)
	}

	// Test FileFilter with a simple implementation
	filter := FileFilterFunc(func(string) bool { return true })
	validator.ValidateFileFilter(filter)

	return validator.GetSummary()
}

// MemoryLeakChecker helps detect potential memory leaks in interface implementations
type MemoryLeakChecker struct {
	initialMem runtime.MemStats
	finalMem   runtime.MemStats
}

// NewMemoryLeakChecker creates a new memory leak checker
func NewMemoryLeakChecker() *MemoryLeakChecker {
	checker := &MemoryLeakChecker{}
	runtime.GC()
	runtime.ReadMemStats(&checker.initialMem)
	return checker
}

// Check performs the memory leak check
func (m *MemoryLeakChecker) Check() MemoryLeakResult {
	runtime.GC()
	runtime.ReadMemStats(&m.finalMem)

	return MemoryLeakResult{
		InitialAlloc: m.initialMem.Alloc,
		FinalAlloc:   m.finalMem.Alloc,
		AllocDelta:   int64(m.finalMem.Alloc) - int64(m.initialMem.Alloc),
		NumGC:        m.finalMem.NumGC - m.initialMem.NumGC,
	}
}

// MemoryLeakResult represents the result of a memory leak check
type MemoryLeakResult struct {
	InitialAlloc uint64
	FinalAlloc   uint64
	AllocDelta   int64
	NumGC        uint32
}

// HasSignificantLeak returns true if there's a significant memory increase
func (r MemoryLeakResult) HasSignificantLeak() bool {
	// Consider significant if allocation increased by more than 1MB
	return r.AllocDelta > 1024*1024
}

// String returns a string representation of the memory leak result
func (r MemoryLeakResult) String() string {
	return fmt.Sprintf("Memory: %d -> %d bytes (delta: %+d), GC cycles: %d",
		r.InitialAlloc, r.FinalAlloc, r.AllocDelta, r.NumGC)
}
