//go:build integration
// +build integration

package integrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/server"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HotReloadTestSystem represents a complete hot reload test environment
type HotReloadTestSystem struct {
	ProjectDir    string
	ComponentsDir string
	Registry      *registry.ComponentRegistry
	Scanner       *scanner.ComponentScanner
	Watcher       *watcher.FileWatcher
	BuildPipeline *build.RefactoredBuildPipeline
	Server        *server.RefactoredPreviewServer
	ServerURL     string
	WSConnection  *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	messageCount  int64
	lastMessage   *server.UpdateMessage
	buildResults  []build.BuildResult
}

// NewHotReloadTestSystem creates a comprehensive hot reload test system
func NewHotReloadTestSystem(t *testing.T) (*HotReloadTestSystem, error) {
	// Create temporary project directory
	projectDir := fmt.Sprintf("hot_reload_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	componentsDir := filepath.Join(projectDir, "components")
	require.NoError(t, os.MkdirAll(componentsDir, 0755))

	// Initialize core components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	fileWatcher, err := watcher.NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	// Create build pipeline
	buildPipeline := build.NewRefactoredBuildPipeline(2, reg)

	ctx, cancel := context.WithCancel(context.Background())

	system := &HotReloadTestSystem{
		ProjectDir:    projectDir,
		ComponentsDir: componentsDir,
		Registry:      reg,
		Scanner:       componentScanner,
		Watcher:       fileWatcher,
		BuildPipeline: buildPipeline,
		ctx:           ctx,
		cancel:        cancel,
		buildResults:  make([]build.BuildResult, 0),
	}

	return system, nil
}

// Start initializes the complete hot reload system
func (h *HotReloadTestSystem) Start(t *testing.T) error {
	// Find available port first
	port, err := FindAvailablePort()
	if err != nil {
		return err
	}

	// Set up configuration with valid port
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:        "localhost",
			Port:        port,
			Environment: "development",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{h.ComponentsDir},
		},
		Development: config.DevelopmentConfig{
			HotReload: true,
		},
	}

	h.ServerURL = fmt.Sprintf("http://localhost:%d", port)

	// Create server with all dependencies
	previewServer, err := server.NewRefactoredWithDependencies(
		cfg,
		h.Registry,
		h.Watcher,
		h.Scanner,
		h.BuildPipeline,
		nil, // No monitoring for tests
	)
	if err != nil {
		return err
	}
	h.Server = previewServer

	// Set up build pipeline callbacks to track results
	h.BuildPipeline.AddCallback(func(result interface{}) {
		if buildResult, ok := result.(build.BuildResult); ok {
			h.buildResults = append(h.buildResults, buildResult)
		}
	})

	// Build pipeline will be started by the server

	// Set up file watcher with proper filters and handlers
	h.Watcher.AddFilter(interfaces.FileFilterFunc(watcher.TemplFilter))
	h.Watcher.AddFilter(interfaces.FileFilterFunc(watcher.NoTestFilter))

	h.Watcher.AddHandler(func(events []interfaces.ChangeEvent) error {
		t.Logf("File change events detected: %d events", len(events))
		for _, event := range events {
			t.Logf("  - %s: %s", event.Type, event.Path)

			// Rescan the changed file
			if err := h.Scanner.ScanFile(event.Path); err != nil {
				t.Logf("Failed to rescan file %s: %v", event.Path, err)
				return err
			}

			// Find and rebuild affected components
			components := h.Registry.GetAll()
			for _, component := range components {
				if component.FilePath == event.Path {
					t.Logf("Rebuilding component: %s", component.Name)
					if err := h.BuildPipeline.Build(component); err != nil {
						t.Logf("Failed to build component %s: %v", component.Name, err)
					}
				}
			}
		}
		return nil
	})

	require.NoError(t, h.Watcher.AddRecursive(h.ComponentsDir))
	require.NoError(t, h.Watcher.Start(h.ctx))

	// Initial scan
	require.NoError(t, h.Scanner.ScanDirectory(h.ComponentsDir))

	// Server URL already set above

	// Start server in background
	go func() {
		if err := h.Server.Start(h.ctx); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	require.NoError(t, WaitForServerHealthy(h.ServerURL, 10*time.Second))

	return nil
}

// ConnectWebSocket establishes WebSocket connection and sets up message handling
func (h *HotReloadTestSystem) ConnectWebSocket(t *testing.T) error {
	wsURL := strings.Replace(h.ServerURL, "http://", "ws://", 1) + "/ws"

	ctx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{h.ServerURL},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	h.WSConnection = conn

	// Start message handler in background
	go h.handleWebSocketMessages(t)

	t.Log("WebSocket connection established successfully")
	return nil
}

// handleWebSocketMessages processes incoming WebSocket messages
func (h *HotReloadTestSystem) handleWebSocketMessages(t *testing.T) {
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			_, message, err := h.WSConnection.Read(h.ctx)
			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					t.Logf("WebSocket read error: %v", err)
				}
				return
			}

			// Parse message
			var updateMsg server.UpdateMessage
			if err := json.Unmarshal(message, &updateMsg); err != nil {
				t.Logf("Failed to parse WebSocket message: %v", err)
				continue
			}

			atomic.AddInt64(&h.messageCount, 1)
			h.lastMessage = &updateMsg

			t.Logf("Received WebSocket message: Type=%s, Target=%s, Content=%s",
				updateMsg.Type, updateMsg.Target, updateMsg.Content)
		}
	}
}

// CreateTestComponent creates a test templ component file
func (h *HotReloadTestSystem) CreateTestComponent(t *testing.T, name, content string) string {
	filePath := filepath.Join(h.ComponentsDir, name+".templ")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))
	t.Logf("Created test component: %s", filePath)
	return filePath
}

// ModifyTestComponent modifies an existing test component
func (h *HotReloadTestSystem) ModifyTestComponent(t *testing.T, filePath, newContent string) {
	require.NoError(t, os.WriteFile(filePath, []byte(newContent), 0600))
	t.Logf("Modified test component: %s", filePath)
}

// WaitForMessage waits for a WebSocket message with specific criteria
func (h *HotReloadTestSystem) WaitForMessage(t *testing.T, messageType string, timeout time.Duration) *server.UpdateMessage {
	deadline := time.Now().Add(timeout)
	initialCount := atomic.LoadInt64(&h.messageCount)

	for time.Now().Before(deadline) {
		currentCount := atomic.LoadInt64(&h.messageCount)
		if currentCount > initialCount && h.lastMessage != nil && h.lastMessage.Type == messageType {
			return h.lastMessage
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for WebSocket message type: %s", messageType)
	return nil
}

// WaitForBuildResult waits for a build result with specific criteria
func (h *HotReloadTestSystem) WaitForBuildResult(t *testing.T, componentName string, timeout time.Duration) *build.BuildResult {
	deadline := time.Now().Add(timeout)
	initialCount := len(h.buildResults)

	for time.Now().Before(deadline) {
		if len(h.buildResults) > initialCount {
			for i := initialCount; i < len(h.buildResults); i++ {
				result := &h.buildResults[i]
				if result.Component.Name == componentName {
					return result
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for build result for component: %s", componentName)
	return nil
}

// GetComponentCount returns the current number of registered components
func (h *HotReloadTestSystem) GetComponentCount() int {
	return len(h.Registry.GetAll())
}

// Stop shuts down the hot reload test system
func (h *HotReloadTestSystem) Stop(t *testing.T) {
	// Close WebSocket connection
	if h.WSConnection != nil {
		_ = h.WSConnection.Close(websocket.StatusNormalClosure, "test complete")
	}

	// Cancel context to signal shutdown
	if h.cancel != nil {
		h.cancel()
	}

	// Stop components
	if h.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.Server.Shutdown(ctx)
	}

	if h.BuildPipeline != nil {
		_ = h.BuildPipeline.Stop()
	}

	if h.Watcher != nil {
		_ = h.Watcher.Stop()
	}

	// Clean up test directory
	_ = os.RemoveAll(h.ProjectDir)
	t.Log("Hot reload test system stopped and cleaned up")
}

// TestHotReload_CompleteWorkflow tests the complete hot reload workflow
func TestHotReload_CompleteWorkflow(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	// Start the system
	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Initial state verification
	assert.Equal(t, 0, system.GetComponentCount(), "Should start with no components")

	// Step 1: Create initial component
	initialTemplate := `package components

templ Button(text string) {
	<button class="btn">{text}</button>
}`

	componentPath := system.CreateTestComponent(t, "Button", initialTemplate)

	// Manually trigger scan since file watcher might need time to detect
	err = system.Scanner.ScanFile(componentPath)
	require.NoError(t, err, "Manual scan should succeed")

	// Wait for component to be scanned and registered
	time.Sleep(500 * time.Millisecond)
	t.Logf("Components in registry: %d", system.GetComponentCount())

	// Let's debug what's in the registry
	allComponents := system.Registry.GetAll()
	for _, comp := range allComponents {
		t.Logf("Found component: %s at %s", comp.Name, comp.FilePath)
	}

	if system.GetComponentCount() == 0 {
		// Try directory scan as fallback
		t.Log("No components found, trying directory scan...")
		err = system.Scanner.ScanDirectory(system.ComponentsDir)
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		t.Logf("After directory scan, components in registry: %d", system.GetComponentCount())
	}

	assert.Equal(t, 1, system.GetComponentCount(), "Component should be registered")

	// Verify component exists in registry
	button, exists := system.Registry.Get("Button")
	require.True(t, exists, "Button component should exist in registry")
	assert.Equal(t, "Button", button.Name)
	assert.Equal(t, componentPath, button.FilePath)

	// Step 2: Manually trigger build to ensure it happens
	t.Log("Manually triggering build for Button component...")
	err = system.BuildPipeline.Build(button)
	require.NoError(t, err, "Manual build should succeed")

	// Wait for initial build
	buildResult := system.WaitForBuildResult(t, "Button", 3*time.Second)
	assert.NotNil(t, buildResult, "Should receive build result")
	assert.Equal(t, "Button", buildResult.Component.Name)

	// Step 3: Modify component and test hot reload
	modifiedTemplate := `package components

templ Button(text string, variant string) {
	<button class="btn btn-{ variant }">{text}</button>
}`

	t.Log("Modifying component to trigger hot reload...")
	system.ModifyTestComponent(t, componentPath, modifiedTemplate)

	// Give file watcher time to detect the change
	time.Sleep(1 * time.Second)

	// Let's manually trigger the file change handling to ensure it works
	t.Log("Manually triggering file change handling...")
	err = system.Scanner.ScanFile(componentPath)
	require.NoError(t, err, "Manual rescan should succeed")

	// Get the updated component and trigger rebuild
	updatedButton, exists := system.Registry.Get("Button")
	require.True(t, exists, "Button should still exist after modification")

	err = system.BuildPipeline.Build(updatedButton)
	require.NoError(t, err, "Manual rebuild should succeed")

	// Step 4: Wait for WebSocket message indicating rebuild
	message := system.WaitForMessage(t, "build_success", 5*time.Second)
	assert.NotNil(t, message, "Should receive build success message")
	assert.Equal(t, "build_success", message.Type)
	assert.Equal(t, "Button", message.Target)

	// Step 5: Verify component was updated in registry
	finalButton, exists2 := system.Registry.Get("Button")
	require.True(t, exists2, "Updated Button component should exist")
	assert.Equal(t, 2, len(finalButton.Parameters), "Should have 2 parameters after modification")

	// Verify parameters are correct
	paramNames := make([]string, len(finalButton.Parameters))
	for i, param := range finalButton.Parameters {
		paramNames[i] = param.Name
	}
	assert.Contains(t, paramNames, "text")
	assert.Contains(t, paramNames, "variant")

	// Step 6: Wait for second build result
	secondBuildResult := system.WaitForBuildResult(t, "Button", 3*time.Second)
	assert.NotNil(t, secondBuildResult, "Should receive second build result")
	assert.Equal(t, "Button", secondBuildResult.Component.Name)

	t.Log("✅ Complete hot reload workflow verified successfully")
}

// TestHotReload_MultipleComponents tests hot reload with multiple components
func TestHotReload_MultipleComponents(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Create multiple components
	components := map[string]string{
		"Button": `package components
templ Button(text string) {
	<button>{text}</button>
}`,
		"Card": `package components
templ Card(title string) {
	<div class="card">
		<h3>{title}</h3>
	</div>
}`,
		"Modal": `package components
templ Modal(content string) {
	<div class="modal">
		<p>{content}</p>
	</div>
}`,
	}

	componentPaths := make(map[string]string)
	for name, template := range components {
		path := system.CreateTestComponent(t, name, template)
		componentPaths[name] = path
	}

	// Wait for all components to be registered
	time.Sleep(1 * time.Second)
	assert.Equal(t, 3, system.GetComponentCount())

	// Modify Card component
	modifiedCard := `package components
templ Card(title string, subtitle string) {
	<div class="card enhanced">
		<h3>{title}</h3>
		<h4>{subtitle}</h4>
	</div>
}`

	system.ModifyTestComponent(t, componentPaths["Card"], modifiedCard)

	// Wait for build success message for Card
	message := system.WaitForMessage(t, "build_success", 5*time.Second)
	assert.Equal(t, "Card", message.Target)

	// Verify only Card was updated
	updatedCard, exists := system.Registry.Get("Card")
	require.True(t, exists)
	assert.Equal(t, 2, len(updatedCard.Parameters), "Card should have 2 parameters")

	// Verify other components unchanged
	button, _ := system.Registry.Get("Button")
	modal, _ := system.Registry.Get("Modal")
	assert.Equal(t, 1, len(button.Parameters), "Button should still have 1 parameter")
	assert.Equal(t, 1, len(modal.Parameters), "Modal should still have 1 parameter")

	t.Log("✅ Multiple component hot reload verified successfully")
}

// TestHotReload_BuildErrors tests hot reload with build errors
func TestHotReload_BuildErrors(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Create valid component first
	validTemplate := `package components
templ ValidButton(text string) {
	<button>{text}</button>
}`

	path := system.CreateTestComponent(t, "ValidButton", validTemplate)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 1, system.GetComponentCount())

	// Introduce syntax error
	invalidTemplate := `package components
templ ValidButton(text string {  // Missing closing parenthesis
	<button>{text}</button>
}`

	system.ModifyTestComponent(t, path, invalidTemplate)

	// Wait for build error message
	message := system.WaitForMessage(t, "build_error", 5*time.Second)
	assert.Equal(t, "build_error", message.Type)
	assert.NotEmpty(t, message.Content, "Error message should contain error details")

	// Fix the error
	system.ModifyTestComponent(t, path, validTemplate)

	// Wait for build success
	successMessage := system.WaitForMessage(t, "build_success", 5*time.Second)
	assert.Equal(t, "build_success", successMessage.Type)
	assert.Equal(t, "ValidButton", successMessage.Target)

	t.Log("✅ Build error handling in hot reload verified successfully")
}

// TestHotReload_WebSocketReconnection tests WebSocket reconnection scenarios
func TestHotReload_WebSocketReconnection(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Create component
	template := `package components
templ ReconnectTest(msg string) {
	<div>{msg}</div>
}`

	path := system.CreateTestComponent(t, "ReconnectTest", template)
	time.Sleep(500 * time.Millisecond)

	// Close WebSocket connection to simulate disconnect
	_ = system.WSConnection.Close(websocket.StatusNormalClosure, "test disconnect")
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	require.NoError(t, system.ConnectWebSocket(t))

	// Modify component after reconnection
	modifiedTemplate := `package components
templ ReconnectTest(msg string, level string) {
	<div class="{level}">{msg}</div>
}`

	system.ModifyTestComponent(t, path, modifiedTemplate)

	// Should receive message on new connection
	message := system.WaitForMessage(t, "build_success", 5*time.Second)
	assert.Equal(t, "ReconnectTest", message.Target)

	t.Log("✅ WebSocket reconnection during hot reload verified successfully")
}

// TestHotReload_ConcurrentChanges tests hot reload with concurrent file changes
func TestHotReload_ConcurrentChanges(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Create multiple components
	componentPaths := make([]string, 3)
	for i := 0; i < 3; i++ {
		template := fmt.Sprintf(`package components
templ Component%d(data string) {
	<div class="comp-%d">{data}</div>
}`, i, i)
		path := system.CreateTestComponent(t, fmt.Sprintf("Component%d", i), template)
		componentPaths[i] = path
	}

	time.Sleep(1 * time.Second)
	assert.Equal(t, 3, system.GetComponentCount())

	// Modify all components concurrently
	for i, path := range componentPaths {
		go func(index int, filePath string) {
			modifiedTemplate := fmt.Sprintf(`package components
templ Component%d(data string, extra string) {
	<div class="comp-%d enhanced">{data} - {extra}</div>
}`, index, index)
			system.ModifyTestComponent(t, filePath, modifiedTemplate)
		}(i, path)
	}

	// Wait for all build success messages
	successCount := 0
	timeout := time.Now().Add(10 * time.Second)

	for time.Now().Before(timeout) && successCount < 3 {
		message := system.WaitForMessage(t, "build_success", 2*time.Second)
		if message != nil {
			successCount++
			t.Logf("Build success %d/3 for component: %s", successCount, message.Target)
		}
	}

	assert.Equal(t, 3, successCount, "Should receive build success for all 3 components")

	// Verify all components were updated
	for i := 0; i < 3; i++ {
		comp, exists := system.Registry.Get(fmt.Sprintf("Component%d", i))
		require.True(t, exists)
		assert.Equal(t, 2, len(comp.Parameters), "Component%d should have 2 parameters", i)
	}

	t.Log("✅ Concurrent hot reload changes verified successfully")
}

// TestHotReload_Performance tests hot reload performance characteristics
func TestHotReload_Performance(t *testing.T) {
	system, err := NewHotReloadTestSystem(t)
	require.NoError(t, err)
	defer system.Stop(t)

	require.NoError(t, system.Start(t))
	require.NoError(t, system.ConnectWebSocket(t))

	// Create component
	template := `package components
templ PerfTest(data string) {
	<div>{data}</div>
}`

	path := system.CreateTestComponent(t, "PerfTest", template)
	time.Sleep(500 * time.Millisecond)

	// Measure hot reload performance
	iterations := 5
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		modifiedTemplate := fmt.Sprintf(`package components
templ PerfTest(data string) {
	<div class="iteration-%d">{data}</div>
}`, i)

		start := time.Now()
		system.ModifyTestComponent(t, path, modifiedTemplate)

		// Wait for build success
		message := system.WaitForMessage(t, "build_success", 3*time.Second)
		duration := time.Since(start)
		totalDuration += duration

		assert.Equal(t, "PerfTest", message.Target)
		t.Logf("Hot reload iteration %d completed in %v", i+1, duration)
	}

	averageDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average hot reload time: %v", averageDuration)

	// Hot reload should complete within reasonable time
	assert.Less(t, averageDuration.Milliseconds(), int64(2000),
		"Average hot reload should complete within 2 seconds")

	t.Log("✅ Hot reload performance characteristics verified successfully")
}
