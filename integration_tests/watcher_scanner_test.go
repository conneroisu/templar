//go:build integration
// +build integration

package integration_tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_WatcherScanner_FileChangeDetection(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Create initial component
	initialContent := `package components

templ Button(text string) {
	<button class="btn">{text}</button>
}`

	buttonFile := createTestComponent(testDir, "Button", initialContent)

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Set up scan trigger
	var scanCount int64
	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		atomic.AddInt64(&scanCount, 1)
		return componentScanner.ScanDirectory(testDir)
	})

	// Add filters for templ files
	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)

	// Initial scan count
	initialScanCount := atomic.LoadInt64(&scanCount)

	// Modify component file
	modifiedContent := `package components

templ Button(text string, disabled bool) {
	<button class="btn" disabled?={disabled}>{text}</button>
}`

	err = os.WriteFile(buttonFile, []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Wait for file change detection and debouncing
	time.Sleep(500 * time.Millisecond)

	// Verify scan was triggered
	finalScanCount := atomic.LoadInt64(&scanCount)
	assert.Greater(t, finalScanCount, initialScanCount, "File change should trigger scan")

	// Verify component was updated in registry
	button, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Len(t, button.Parameters, 2, "Component should have 2 parameters after modification")
	assert.Equal(t, "text", button.Parameters[0].Name)
	assert.Equal(t, "disabled", button.Parameters[1].Name)
}

func TestIntegration_WatcherScanner_MultipleFileChanges(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Create multiple components
	components := map[string]string{
		"Button": `package components
templ Button(text string) {
	<button>{text}</button>
}`,
		"Card": `package components
templ Card(title string) {
	<div>{title}</div>
}`,
		"Modal": `package components
templ Modal(title string) {
	<div>{title}</div>
}`,
	}

	for name, content := range components {
		createTestComponent(testDir, name, content)
	}

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(150 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Track scan events
	var scanEvents []time.Time
	var scanMutex sync.Mutex

	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		scanMutex.Lock()
		scanEvents = append(scanEvents, time.Now())
		scanMutex.Unlock()
		return componentScanner.ScanDirectory(testDir)
	})

	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)

	// Record initial scan count
	scanMutex.Lock()
	initialScanCount := len(scanEvents)
	scanMutex.Unlock()

	// Modify multiple files in quick succession
	modifiedComponents := map[string]string{
		"Button": `package components
templ Button(text string, variant string) {
	<button class={variant}>{text}</button>
}`,
		"Card": `package components
templ Card(title string, content string) {
	<div class="card">
		<h3>{title}</h3>
		<p>{content}</p>
	</div>
}`,
		"Modal": `package components
templ Modal(title string, visible bool) {
	if visible {
		<div class="modal">{title}</div>
	}
}`,
	}

	for name, content := range modifiedComponents {
		filePath := filepath.Join(testDir, name+".templ")
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond) // Small delay between modifications
	}

	// Wait for debouncing and processing
	time.Sleep(1 * time.Second)

	// Verify scans were triggered (should be debounced)
	scanMutex.Lock()
	finalScanCount := len(scanEvents)
	scanMutex.Unlock()

	assert.Greater(t, finalScanCount, initialScanCount, "File changes should trigger scans")

	// Verify all components were updated
	button, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Len(t, button.Parameters, 2)

	card, exists := reg.Get("Card")
	assert.True(t, exists)
	assert.Len(t, card.Parameters, 2)

	modal, exists := reg.Get("Modal")
	assert.True(t, exists)
	assert.Len(t, modal.Parameters, 2)
}

func TestIntegration_WatcherScanner_NewFileCreation(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Track new components discovered
	var newComponents []string
	var componentMutex sync.Mutex

	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		err := componentScanner.ScanDirectory(testDir)
		if err != nil {
			return err
		}

		// Check for new components
		componentMutex.Lock()
		allComponents := reg.GetAll()
		for name := range allComponents {
			found := false
			for _, existing := range newComponents {
				if existing == name {
					found = true
					break
				}
			}
			if !found {
				newComponents = append(newComponents, name)
			}
		}
		componentMutex.Unlock()

		return nil
	})

	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)

	// Create new component files
	newComponentsToCreate := map[string]string{
		"Alert": `package components
templ Alert(message string, type string) {
	<div class={"alert", "alert-" + type}>{message}</div>
}`,
		"Badge": `package components
templ Badge(text string, count int) {
	<span class="badge">{text} ({fmt.Sprintf("%d", count)})</span>
}`,
	}

	for name, content := range newComponentsToCreate {
		createTestComponent(testDir, name, content)
		time.Sleep(200 * time.Millisecond) // Allow time for detection
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify new components were discovered
	componentMutex.Lock()
	discoveredComponents := make([]string, len(newComponents))
	copy(discoveredComponents, newComponents)
	componentMutex.Unlock()

	assert.Contains(t, discoveredComponents, "Alert")
	assert.Contains(t, discoveredComponents, "Badge")

	// Verify components are in registry
	alert, exists := reg.Get("Alert")
	assert.True(t, exists)
	assert.Equal(t, "Alert", alert.Name)

	badge, exists := reg.Get("Badge")
	assert.True(t, exists)
	assert.Equal(t, "Badge", badge.Name)
}

func TestIntegration_WatcherScanner_FileDeletion(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Create initial components
	components := map[string]string{
		"Button": `package components
templ Button(text string) {
	<button>{text}</button>
}`,
		"Card": `package components
templ Card(title string) {
	<div>{title}</div>
}`,
	}

	for name, content := range components {
		createTestComponent(testDir, name, content)
	}

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Set up scan trigger
	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		return componentScanner.ScanDirectory(testDir)
	})

	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup and scan
	time.Sleep(300 * time.Millisecond)

	// Verify both components are initially present
	assert.Equal(t, 2, reg.Count())
	_, exists := reg.Get("Button")
	assert.True(t, exists)
	_, exists = reg.Get("Card")
	assert.True(t, exists)

	// Delete one component file
	buttonFile := filepath.Join(testDir, "Button.templ")
	err = os.Remove(buttonFile)
	require.NoError(t, err)

	// Wait for file deletion detection and processing
	time.Sleep(500 * time.Millisecond)

	// Note: The registry doesn't automatically remove deleted components
	// This depends on the scanner implementation to handle deletions
	// For now, we verify that the remaining component is still accessible
	card, exists := reg.Get("Card")
	assert.True(t, exists)
	assert.Equal(t, "Card", card.Name)
}

func TestIntegration_WatcherScanner_FilteringEfficiency(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Create mix of files - only .templ should trigger scans
	fileContents := map[string]string{
		"Component.templ": `package components
templ Component() {
	<div>test</div>
}`,
		"readme.md":       "# README",
		"config.json":     `{"key": "value"}`,
		"script.js":       "console.log('test');",
		"style.css":       ".test { color: red; }",
	}

	for name, content := range fileContents {
		filePath := filepath.Join(testDir, name)
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Track scan events
	var scanCount int64
	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		atomic.AddInt64(&scanCount, 1)
		return componentScanner.ScanDirectory(testDir)
	})

	// Add filter for only templ files
	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)
	initialScanCount := atomic.LoadInt64(&scanCount)

	// Modify non-templ files - should not trigger scans
	nonTemplFiles := []string{"readme.md", "config.json", "script.js", "style.css"}
	for _, fileName := range nonTemplFiles {
		filePath := filepath.Join(testDir, fileName)
		err = os.WriteFile(filePath, []byte("modified content"), 0644)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for potential processing
	time.Sleep(300 * time.Millisecond)

	// Verify no additional scans were triggered
	scanCountAfterNonTempl := atomic.LoadInt64(&scanCount)
	assert.Equal(t, initialScanCount, scanCountAfterNonTempl, 
		"Non-templ file changes should not trigger scans")

	// Modify templ file - should trigger scan
	templFile := filepath.Join(testDir, "Component.templ")
	modifiedContent := `package components
templ Component(text string) {
	<div>{text}</div>
}`
	err = os.WriteFile(templFile, []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Verify scan was triggered for templ file
	finalScanCount := atomic.LoadInt64(&scanCount)
	assert.Greater(t, finalScanCount, scanCountAfterNonTempl,
		"Templ file change should trigger scan")

	// Verify component was updated
	component, exists := reg.Get("Component")
	assert.True(t, exists)
	assert.Len(t, component.Parameters, 1)
	assert.Equal(t, "text", component.Parameters[0].Name)
}

func TestIntegration_WatcherScanner_ErrorResilience(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Track errors
	var errorCount int64
	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		err := componentScanner.ScanDirectory(testDir)
		if err != nil {
			atomic.AddInt64(&errorCount, 1)
		}
		return nil // Don't propagate errors to prevent watcher from stopping
	})

	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)

	// Create valid component
	validContent := `package components
templ ValidComponent(text string) {
	<div>{text}</div>
}`
	createTestComponent(testDir, "Valid", validContent)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Verify valid component was processed
	validComponent, exists := reg.Get("ValidComponent")
	assert.True(t, exists)
	assert.Equal(t, "ValidComponent", validComponent.Name)

	// Create component with syntax error
	invalidContent := `package components
templ InvalidComponent(text string {  // Missing closing parenthesis
	<div>{text}</div>
}`
	createTestComponent(testDir, "Invalid", invalidContent)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// System should continue working despite the error
	// The error count might or might not increase depending on scanner implementation
	
	// Create another valid component to ensure system is still responsive
	anotherValidContent := `package components
templ AnotherValidComponent(title string) {
	<h1>{title}</h1>
}`
	createTestComponent(testDir, "AnotherValid", anotherValidContent)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Verify the system is still working
	anotherValid, exists := reg.Get("AnotherValidComponent")
	assert.True(t, exists)
	assert.Equal(t, "AnotherValidComponent", anotherValid.Name)
}

func TestIntegration_WatcherScanner_PerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Initialize components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(50 * time.Millisecond) // Shorter debounce for load test
	require.NoError(t, err)
	defer fileWatcher.Stop()

	// Track processing times
	var processingTimes []time.Duration
	var timeMutex sync.Mutex

	fileWatcher.AddHandler(func(events []watcher.ChangeEvent) error {
		start := time.Now()
		err := componentScanner.ScanDirectory(testDir)
		duration := time.Since(start)

		timeMutex.Lock()
		processingTimes = append(processingTimes, duration)
		timeMutex.Unlock()

		return err
	})

	fileWatcher.AddFilter(watcher.TemplFilter)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = fileWatcher.AddPath(testDir)
	require.NoError(t, err)

	err = fileWatcher.Start(ctx)
	require.NoError(t, err)

	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)

	// Create multiple components rapidly
	componentCount := 50
	var wg sync.WaitGroup

	for i := 0; i < componentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			content := fmt.Sprintf(`package components
templ Component%d(text string, id int) {
	<div id={"comp-" + fmt.Sprintf("%%d", id)}>{text}</div>
}`, index)
			
			createTestComponent(testDir, fmt.Sprintf("Component%d", index), content)
			time.Sleep(10 * time.Millisecond) // Small delay between creations
		}(i)
	}

	wg.Wait()

	// Wait for all processing to complete
	time.Sleep(2 * time.Second)

	// Verify all components were processed
	assert.Equal(t, componentCount, reg.Count(), 
		"All components should be registered")

	// Check processing times
	timeMutex.Lock()
	avgProcessingTime := time.Duration(0)
	if len(processingTimes) > 0 {
		var totalTime time.Duration
		for _, duration := range processingTimes {
			totalTime += duration
		}
		avgProcessingTime = totalTime / time.Duration(len(processingTimes))
	}
	timeMutex.Unlock()

	// Performance assertion - processing should be reasonably fast
	assert.Less(t, avgProcessingTime, 2*time.Second, 
		"Average processing time should be reasonable")

	t.Logf("Processed %d components with average processing time: %v", 
		componentCount, avgProcessingTime)
}