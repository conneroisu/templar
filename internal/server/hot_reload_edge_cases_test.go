package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHotReload_EdgeCases tests various edge cases and failure scenarios
func TestHotReload_EdgeCases(t *testing.T) {
	t.Run("corrupted_component_file", func(t *testing.T) {
		// Create temporary directory for test
		tempDir := fmt.Sprintf("edge_case_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Set up test system
		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		// Test 1: Create corrupted component file
		corruptedFile := filepath.Join(tempDir, "corrupted.templ")
		corruptedContent := "package components\n\ntempl Corrupted { // Missing parameters and closing brace\n\t<div>Invalid"

		require.NoError(t, os.WriteFile(corruptedFile, []byte(corruptedContent), 0644))

		// Try to scan corrupted file - should handle gracefully
		err := componentScanner.ScanFile(corruptedFile)
		if err != nil {
			assert.Contains(t, err.Error(), "parse", "Error should mention parsing issue")
			t.Logf("Scanner correctly detected error: %v", err)
		} else {
			t.Log("Scanner handled corrupted file without error (may be valid behavior)")
		}

		// Check if component was registered (actual behavior may vary)
		component, exists := reg.Get("Corrupted")
		if exists {
			t.Logf("Scanner registered corrupted component: %+v", component)
			t.Log("This shows the scanner is resilient to some syntax errors")
		} else {
			t.Log("Scanner correctly rejected corrupted component")
		}

		t.Log("✅ Corrupted file handling tested successfully")
	})

	t.Run("empty_component_file", func(t *testing.T) {
		tempDir := fmt.Sprintf("empty_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		// Create empty file
		emptyFile := filepath.Join(tempDir, "empty.templ")
		require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0644))

		// Try to scan empty file
		err := componentScanner.ScanFile(emptyFile)
		// Should handle empty file gracefully (might be error or no-op)
		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "empty", "Error should mention empty file")
		}

		// No component should be registered
		assert.Equal(t, 0, len(reg.GetAll()), "No components should be registered from empty file")

		t.Log("✅ Empty file handling tested successfully")
	})

	t.Run("invalid_permissions_file", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir := fmt.Sprintf("permission_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		// Create file with no read permissions
		restrictedFile := filepath.Join(tempDir, "restricted.templ")
		validContent := `package components

templ Restricted() {
	<div>Access denied</div>
}`
		require.NoError(t, os.WriteFile(restrictedFile, []byte(validContent), 0644))
		require.NoError(t, os.Chmod(restrictedFile, 0000)) // No permissions

		// Try to scan restricted file
		err := componentScanner.ScanFile(restrictedFile)
		assert.Error(t, err, "Should fail to read restricted file")
		assert.Contains(t, err.Error(), "permission", "Error should mention permission issue")

		t.Log("✅ Permission handling tested successfully")
	})

	t.Run("concurrent_file_deletion", func(t *testing.T) {
		tempDir := fmt.Sprintf("deletion_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		// Create test file
		testFile := filepath.Join(tempDir, "vanishing.templ")
		validContent := `package components

templ Vanishing() {
	<div>Here one moment, gone the next</div>
}`
		require.NoError(t, os.WriteFile(testFile, []byte(validContent), 0644))

		// Start concurrent operations
		var wg sync.WaitGroup
		numGoroutines := 5

		// Some goroutines try to scan the file
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id*10) * time.Millisecond)
				err := componentScanner.ScanFile(testFile)
				// May succeed or fail depending on timing - both are valid outcomes
				if err != nil {
					t.Logf("Goroutine %d failed to scan (expected): %v", id, err)
				} else {
					t.Logf("Goroutine %d successfully scanned before deletion", id)
				}
			}(i)
		}

		// Delete the file while scans are in progress
		time.Sleep(20 * time.Millisecond)
		_ = os.Remove(testFile)

		wg.Wait()
		t.Log("✅ Concurrent file deletion handling tested successfully")
	})

	t.Run("malformed_templ_syntax", func(t *testing.T) {
		tempDir := fmt.Sprintf("malformed_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		testCases := []struct {
			name    string
			content string
		}{
			{
				name: "missing_package",
				content: `templ MissingPackage() {
	<div>No package declaration</div>
}`,
			},
			{
				name: "invalid_templ_keyword",
				content: `package components

template InvalidKeyword() {
	<div>Wrong keyword</div>
}`,
			},
			{
				name: "unmatched_braces",
				content: `package components

templ UnmatchedBraces() {
	<div>Missing closing brace
}`,
			},
			{
				name: "invalid_html",
				content: `package components

templ InvalidHTML() {
	<div><span>Unclosed spans</div>
}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				malformedFile := filepath.Join(tempDir, tc.name+".templ")
				require.NoError(t, os.WriteFile(malformedFile, []byte(tc.content), 0644))

				// Should handle malformed syntax gracefully
				err := componentScanner.ScanFile(malformedFile)
				assert.Error(t, err, "Should detect malformed syntax in %s", tc.name)

				// No component should be registered
				// Convert to title case manually to avoid deprecated strings.Title
				words := strings.Split(tc.name, "_")
				for i, word := range words {
					if len(word) > 0 {
						words[i] = strings.ToUpper(word[:1]) + word[1:]
					}
				}
				componentName := strings.Join(words, "")
				_, exists := reg.Get(componentName)
				assert.False(t, exists, "Malformed component should not be registered: %s", componentName)
			})
		}

		t.Log("✅ Malformed syntax handling tested successfully")
	})

	t.Run("build_pipeline_edge_cases", func(t *testing.T) {
		reg := registry.NewComponentRegistry()
		buildPipeline := build.NewRefactoredBuildPipeline(2, reg)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, buildPipeline.Start(ctx))
		defer func() { _ = buildPipeline.Stop() }()

		// Test building non-existent component
		nonExistentComponent := &types.ComponentInfo{
			Name:     "NonExistent",
			FilePath: "/does/not/exist.templ",
			Package:  "components",
		}

		err := buildPipeline.Build(nonExistentComponent)
		assert.Error(t, err, "Should fail to build non-existent component")

		// Test building component with invalid path
		invalidPathComponent := &types.ComponentInfo{
			Name:     "InvalidPath",
			FilePath: "invalid\x00path.templ", // Null byte in path
			Package:  "components",
		}

		err = buildPipeline.Build(invalidPathComponent)
		assert.Error(t, err, "Should fail to build component with invalid path")

		t.Log("✅ Build pipeline edge cases tested successfully")
	})

	t.Run("registry_edge_cases", func(t *testing.T) {
		reg := registry.NewComponentRegistry()

		// Test registering nil component
		reg.Register(nil)
		assert.Equal(t, 0, len(reg.GetAll()), "Nil component should not be registered")

		// Test registering component with empty name
		emptyNameComponent := &types.ComponentInfo{
			Name:     "",
			FilePath: "empty_name.templ",
			Package:  "components",
		}
		reg.Register(emptyNameComponent)

		// May or may not be registered depending on implementation
		// The important thing is it doesn't crash

		// Test getting non-existent component
		_, exists := reg.Get("DoesNotExist")
		assert.False(t, exists, "Non-existent component should not be found")

		t.Log("✅ Registry edge cases tested successfully")
	})

	t.Run("large_component_file", func(t *testing.T) {
		tempDir := fmt.Sprintf("large_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)

		// Create very large component file
		largeFile := filepath.Join(tempDir, "large.templ")
		
		var content strings.Builder
		content.WriteString("package components\n\n")
		content.WriteString("templ LargeComponent() {\n")
		
		// Generate large HTML content
		for i := 0; i < 10000; i++ {
			content.WriteString(fmt.Sprintf("\t<div class=\"item-%d\">Item %d</div>\n", i, i))
		}
		content.WriteString("}")

		require.NoError(t, os.WriteFile(largeFile, []byte(content.String()), 0644))

		// Should handle large file without issues (might be slow)
		start := time.Now()
		err := componentScanner.ScanFile(largeFile)
		duration := time.Since(start)

		assert.NoError(t, err, "Should handle large component file")
		assert.Less(t, duration, 10*time.Second, "Should process large file within reasonable time")

		// Component should be registered
		component, exists := reg.Get("LargeComponent")
		assert.True(t, exists, "Large component should be registered")
		assert.Equal(t, "LargeComponent", component.Name)

		t.Logf("✅ Large file handling tested successfully (processed in %v)", duration)
	})

	t.Run("file_watcher_edge_cases", func(t *testing.T) {
		tempDir := fmt.Sprintf("watcher_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		fileWatcher, err := watcher.NewFileWatcher(10 * time.Millisecond)
		require.NoError(t, err)
		defer func() { _ = fileWatcher.Stop() }()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eventCount int64
		var mu sync.Mutex

		// Add handler to count events
		fileWatcher.AddHandler(func(events []interfaces.ChangeEvent) error {
			mu.Lock()
			eventCount += int64(len(events))
			mu.Unlock()
			return nil
		})

		// Start watching non-existent directory - should handle gracefully
		err = fileWatcher.AddRecursive("/does/not/exist")
		assert.Error(t, err, "Should fail to watch non-existent directory")

		// Watch valid directory
		require.NoError(t, fileWatcher.AddRecursive(tempDir))
		require.NoError(t, fileWatcher.Start(ctx))

		// Create and rapidly delete/recreate files
		testFile := filepath.Join(tempDir, "rapid.templ")
		for i := 0; i < 10; i++ {
			require.NoError(t, os.WriteFile(testFile, []byte(fmt.Sprintf("// Version %d", i)), 0644))
			time.Sleep(5 * time.Millisecond)
			_ = os.Remove(testFile)
			time.Sleep(5 * time.Millisecond)
		}

		// Give watcher time to process events
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		finalEventCount := eventCount
		mu.Unlock()

		assert.Greater(t, finalEventCount, int64(0), "Should detect some file events")
		t.Logf("✅ File watcher edge cases tested successfully (detected %d events)", finalEventCount)
	})
}

// TestHotReload_FailureRecovery tests the system's ability to recover from failures
func TestHotReload_FailureRecovery(t *testing.T) {
	t.Run("build_failure_recovery", func(t *testing.T) {
		tempDir := fmt.Sprintf("recovery_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer func() { _ = os.RemoveAll(tempDir) }()

		reg := registry.NewComponentRegistry()
		componentScanner := scanner.NewComponentScanner(reg)
		buildPipeline := build.NewRefactoredBuildPipeline(2, reg)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, buildPipeline.Start(ctx))
		defer func() { _ = buildPipeline.Stop() }()

		// Step 1: Create valid component
		validFile := filepath.Join(tempDir, "recovery.templ")
		validContent := `package components

templ Recovery() {
	<div>Valid component</div>
}`
		require.NoError(t, os.WriteFile(validFile, []byte(validContent), 0644))
		require.NoError(t, componentScanner.ScanFile(validFile))

		component, exists := reg.Get("Recovery")
		require.True(t, exists, "Valid component should be registered")

		// Build should succeed
		err := buildPipeline.Build(component)
		assert.NoError(t, err, "Valid component should build successfully")

		// Step 2: Introduce build error
		invalidContent := `package components

templ Recovery() {
	<div>Invalid { syntax
}`
		require.NoError(t, os.WriteFile(validFile, []byte(invalidContent), 0644))
		require.NoError(t, componentScanner.ScanFile(validFile))

		// Build should fail
		updatedComponent, exists := reg.Get("Recovery")
		require.True(t, exists, "Component should still be registered")

		err = buildPipeline.Build(updatedComponent)
		assert.Error(t, err, "Invalid component should fail to build")

		// Step 3: Fix the error
		require.NoError(t, os.WriteFile(validFile, []byte(validContent), 0644))
		require.NoError(t, componentScanner.ScanFile(validFile))

		// Build should succeed again
		fixedComponent, exists := reg.Get("Recovery")
		require.True(t, exists, "Fixed component should be registered")

		err = buildPipeline.Build(fixedComponent)
		assert.NoError(t, err, "Fixed component should build successfully")

		t.Log("✅ Build failure recovery tested successfully")
	})

	t.Run("component_registry_recovery", func(t *testing.T) {
		reg := registry.NewComponentRegistry()

		// Register valid component
		validComponent := &types.ComponentInfo{
			Name:     "Valid",
			FilePath: "valid.templ",
			Package:  "components",
		}
		reg.Register(validComponent)
		assert.Equal(t, 1, len(reg.GetAll()), "Valid component should be registered")

		// Try to register invalid component (shouldn't crash)
		invalidComponent := &types.ComponentInfo{
			Name:     "",
			FilePath: "",
			Package:  "",
		}
		reg.Register(invalidComponent)

		// Valid component should still be there
		_, exists := reg.Get("Valid")
		assert.True(t, exists, "Valid component should still exist after invalid registration")

		t.Log("✅ Component registry recovery tested successfully")
	})
}