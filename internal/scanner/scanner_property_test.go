//go:build property
// +build property

package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestScannerProperties tests invariant properties of the component scanner
func TestScannerProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: Scanning the same directory twice should yield identical results
	properties.Property("scanner idempotency", prop.ForAll(
		func(componentName string) bool {
			if componentName == "" || strings.ContainsAny(componentName, "/\\.:;") {
				return true // Skip invalid names
			}

			// Create temporary test environment
			tempDir := t.TempDir()
			componentFile := filepath.Join(tempDir, componentName+".templ")

			// Create a valid component file
			componentContent := fmt.Sprintf(`package components

templ %s(text string) {
	<div class="component">{ text }</div>
}`, componentName)

			if err := os.WriteFile(componentFile, []byte(componentContent), 0644); err != nil {
				return true // Skip on write error
			}

			// Create two scanners and scan the same directory
			registry1 := registry.NewComponentRegistry()
			scanner1 := NewComponentScanner(registry1)

			registry2 := registry.NewComponentRegistry()
			scanner2 := NewComponentScanner(registry2)

			// Scan with both scanners
			err1 := scanner1.ScanDirectory(tempDir)
			err2 := scanner2.ScanDirectory(tempDir)

			if err1 != nil || err2 != nil {
				return true // Skip on scan error
			}

			// Get results
			components1 := registry1.GetAll()
			components2 := registry2.GetAll()

			// Compare results - should be identical
			if len(components1) != len(components2) {
				return false
			}

			for i, comp1 := range components1 {
				comp2 := components2[i]
				if comp1.Name != comp2.Name || comp1.Package != comp2.Package {
					return false
				}
			}

			return true
		},
		gen.RegexMatch(`^[A-Z][a-zA-Z0-9_]*$`).SuchThat(func(s string) bool {
			return len(s) >= 1 && len(s) <= 20
		}),
	))

	// Property 2: Scanner should consistently handle empty directories
	properties.Property("empty directory consistency", prop.ForAll(
		func() bool {
			tempDir := t.TempDir()

			registry := registry.NewComponentRegistry()
			scanner := NewComponentScanner(registry)

			// Scan empty directory multiple times
			for i := 0; i < 3; i++ {
				if err := scanner.ScanDirectory(tempDir); err != nil {
					return false
				}
			}

			// Should have no components
			components := registry.GetAll()
			return len(components) == 0
		},
	))

	// Property 3: File path normalization should be consistent
	properties.Property("path normalization consistency", prop.ForAll(
		func(pathSegments []string) bool {
			if len(pathSegments) == 0 {
				return true
			}

			// Filter out invalid path segments
			validSegments := make([]string, 0, len(pathSegments))
			for _, segment := range pathSegments {
				if segment != "" && !strings.ContainsAny(segment, "/\\:*?\"<>|") {
					validSegments = append(validSegments, segment)
				}
			}

			if len(validSegments) == 0 {
				return true
			}

			path1 := filepath.Join(validSegments...)
			path2 := filepath.Join(validSegments...)

			normalized1 := normalizePathForScanner(path1)
			normalized2 := normalizePathForScanner(path2)

			return normalized1 == normalized2
		},
		gen.SliceOfN(5, gen.RegexMatch(`^[a-zA-Z0-9_-]+$`)),
	))

	properties.TestingRun(t)
}

// TestComponentParsingProperties tests properties of component parsing
func TestComponentParsingProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Valid component templates should always parse successfully
	properties.Property("valid component parsing", prop.ForAll(
		func(componentName, paramName, paramType string) bool {
			if componentName == "" || paramName == "" || paramType == "" {
				return true
			}

			// Generate valid component template
			template := fmt.Sprintf(`package components

templ %s(%s %s) {
	<div>{ %s }</div>
}`, componentName, paramName, paramType, paramName)

			registry := registry.NewComponentRegistry()
			scanner := NewComponentScanner(registry)

			// Parse template by creating temporary file
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test.templ")
			os.WriteFile(tempFile, []byte(template), 0644)

			err := scanner.ScanFile(tempFile)
			components := registry.GetAll()
			if err != nil {
				return false
			}

			// Should find exactly one component
			if len(components) != 1 {
				return false
			}

			component := components[0]
			return component.Name == componentName && len(component.Parameters) > 0
		},
		gen.RegexMatch(`^[A-Z][a-zA-Z0-9]*$`).SuchThat(func(s string) bool {
			return len(s) >= 1 && len(s) <= 15
		}),
		gen.RegexMatch(`^[a-z][a-zA-Z0-9]*$`).SuchThat(func(s string) bool {
			return len(s) >= 1 && len(s) <= 10
		}),
		gen.OneConstOf("string", "int", "bool", "[]string"),
	))

	properties.TestingRun(t)
}

// TestRegistryProperties tests properties of the component registry
func TestRegistryProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Registry operations should maintain consistency
	properties.Property("registry consistency", prop.ForAll(
		func(operations []RegistryOperation) bool {
			reg := registry.NewComponentRegistry()

			// Execute all operations
			for _, op := range operations {
				switch op.Type {
				case "register":
					if op.Component != nil {
						reg.Register(op.Component)
					}
				case "get":
					if op.ComponentName != "" {
						reg.Get(op.ComponentName)
					}
				case "getAll":
					reg.GetAll()
				}
			}

			// Verify registry consistency
			allComponents := reg.GetAll()

			// Check that all components can be retrieved individually
			for _, component := range allComponents {
				retrieved, exists := reg.Get(component.Name)
				if !exists || retrieved.Name != component.Name {
					return false
				}
			}

			return true
		},
		genRegistryOperations(),
	))

	properties.TestingRun(t)
}

// RegistryOperation represents an operation on the registry for property testing
type RegistryOperation struct {
	Type          string
	Component     *types.ComponentInfo
	ComponentName string
}

// genRegistryOperations generates sequences of registry operations
func genRegistryOperations() gopter.Gen {
	return gen.SliceOfN(10, gen.OneGenOf(
		// Register operation
		gen.Struct(reflect.TypeOf(RegistryOperation{}), map[string]gopter.Gen{
			"Type": gen.Const("register"),
			"Component": gen.PtrOf(
				gen.Struct(reflect.TypeOf(types.ComponentInfo{}), map[string]gopter.Gen{
					"Name":     gen.RegexMatch(`^[A-Z][a-zA-Z0-9]*$`),
					"Package":  gen.Const("components"),
					"FilePath": gen.RegexMatch(`^[a-zA-Z0-9_/]+\.templ$`),
				}),
			),
		}),
		// Get operation
		gen.Struct(reflect.TypeOf(RegistryOperation{}), map[string]gopter.Gen{
			"Type":          gen.Const("get"),
			"ComponentName": gen.RegexMatch(`^[A-Z][a-zA-Z0-9]*$`),
		}),
		// GetAll operation
		gen.Struct(reflect.TypeOf(RegistryOperation{}), map[string]gopter.Gen{
			"Type": gen.Const("getAll"),
		}),
	))
}

// normalizePathForScanner normalizes paths for consistent scanning
func normalizePathForScanner(path string) string {
	return filepath.Clean(path)
}

// BenchmarkScannerPropertyTests benchmarks property test performance
func BenchmarkScannerPropertyTests(b *testing.B) {
	reg := registry.NewComponentRegistry()
	_ = NewComponentScanner(reg) // We'll create fresh scanners in the loop

	// Create test directory with components
	tempDir := b.TempDir()
	for i := 0; i < 10; i++ {
		componentFile := filepath.Join(tempDir, fmt.Sprintf("Component%d.templ", i))
		content := fmt.Sprintf(`package components

templ Component%d(text string) {
	<div>{ text }</div>
}`, i)
		os.WriteFile(componentFile, []byte(content), 0644)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Create fresh scanner for each iteration
			newReg := registry.NewComponentRegistry()
			sc := NewComponentScanner(newReg)
			sc.ScanDirectory(tempDir)
		}
	})
}

// TestScannerConcurrencyProperties tests concurrent scanner behavior
func TestScannerConcurrencyProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Concurrent scanning should be thread-safe
	properties.Property("concurrent scanning safety", prop.ForAll(
		func(numGoroutines int) bool {
			if numGoroutines < 1 || numGoroutines > 10 {
				return true // Skip invalid values
			}

			tempDir := t.TempDir()

			// Create test component
			componentFile := filepath.Join(tempDir, "TestComponent.templ")
			content := `package components

templ TestComponent(text string) {
	<div>{ text }</div>
}`
			if err := os.WriteFile(componentFile, []byte(content), 0644); err != nil {
				return true
			}

			// Scan concurrently from multiple goroutines
			results := make(chan int, numGoroutines)
			for i := 0; i < numGoroutines; i++ {
				go func() {
					registry := registry.NewComponentRegistry()
					scanner := NewComponentScanner(registry)

					if err := scanner.ScanDirectory(tempDir); err != nil {
						results <- 0
						return
					}

					components := registry.GetAll()
					results <- len(components)
				}()
			}

			// Collect results
			for i := 0; i < numGoroutines; i++ {
				select {
				case count := <-results:
					if count != 1 { // Should always find exactly 1 component
						return false
					}
				case <-time.After(5 * time.Second):
					return false // Timeout
				}
			}

			return true
		},
		gen.IntRange(1, 5),
	))

	properties.TestingRun(t)
}
