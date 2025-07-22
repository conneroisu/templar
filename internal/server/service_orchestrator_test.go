package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
)

// MockComponentRegistry provides mock component registry for testing
type MockComponentRegistry struct {
	components []*types.ComponentInfo
	count      int
	watchers   []chan types.ComponentEvent
}

func (m *MockComponentRegistry) Register(info *types.ComponentInfo) {
	m.components = append(m.components, info)
	m.count++
}

func (m *MockComponentRegistry) Get(name string) (*types.ComponentInfo, bool) {
	for _, component := range m.components {
		if component.Name == name {
			return component, true
		}
	}
	return nil, false
}

func (m *MockComponentRegistry) GetAll() []*types.ComponentInfo {
	return m.components
}

func (m *MockComponentRegistry) Count() int {
	return m.count
}

func (m *MockComponentRegistry) Watch() <-chan types.ComponentEvent {
	ch := make(chan types.ComponentEvent)
	m.watchers = append(m.watchers, ch)
	return ch
}

func (m *MockComponentRegistry) UnWatch(ch <-chan types.ComponentEvent) {
	// Mock implementation - find and remove channel
	for i, watcher := range m.watchers {
		if watcher == ch {
			close(watcher)
			m.watchers = append(m.watchers[:i], m.watchers[i+1:]...)
			break
		}
	}
}

func (m *MockComponentRegistry) DetectCircularDependencies() [][]string {
	return nil // No circular dependencies in mock
}

// MockFileWatcher provides mock file watcher for testing
type MockFileWatcher struct {
	started   bool
	stopped   bool
	filters   []interfaces.FileFilter
	handlers  []interfaces.ChangeHandlerFunc
	watchPaths []string
}

func (m *MockFileWatcher) AddFilter(filter interfaces.FileFilter) {
	m.filters = append(m.filters, filter)
}

func (m *MockFileWatcher) AddHandler(handler interfaces.ChangeHandlerFunc) {
	m.handlers = append(m.handlers, handler)
}

func (m *MockFileWatcher) AddPath(path string) error {
	m.watchPaths = append(m.watchPaths, path)
	return nil
}

func (m *MockFileWatcher) AddRecursive(path string) error {
	m.watchPaths = append(m.watchPaths, path)
	return nil
}

func (m *MockFileWatcher) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *MockFileWatcher) Stop() error {
	m.stopped = true
	return nil
}

// MockComponentScanner provides mock component scanner for testing
type MockComponentScanner struct {
	scannedDirectories []string
	scannedFiles      []string
	registry          interfaces.ComponentRegistry
}

func (m *MockComponentScanner) ScanDirectory(path string) error {
	m.scannedDirectories = append(m.scannedDirectories, path)
	return nil
}

func (m *MockComponentScanner) ScanDirectoryParallel(dir string, workers int) error {
	m.scannedDirectories = append(m.scannedDirectories, dir)
	return nil
}

func (m *MockComponentScanner) ScanFile(path string) error {
	m.scannedFiles = append(m.scannedFiles, path)
	return nil
}

func (m *MockComponentScanner) GetRegistry() interfaces.ComponentRegistry {
	return m.registry
}

// MockBuildPipeline provides mock build pipeline for testing
type MockBuildPipeline struct {
	started   bool
	stopped   bool
	callbacks []interfaces.BuildCallbackFunc
	built     []*types.ComponentInfo
}

func (m *MockBuildPipeline) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *MockBuildPipeline) Stop() error {
	m.stopped = true
	return nil
}

func (m *MockBuildPipeline) Build(component *types.ComponentInfo) error {
	m.built = append(m.built, component)
	return nil
}

func (m *MockBuildPipeline) BuildWithPriority(component *types.ComponentInfo) {
	m.built = append(m.built, component)
}

func (m *MockBuildPipeline) AddCallback(callback interfaces.BuildCallbackFunc) {
	m.callbacks = append(m.callbacks, callback)
}

func (m *MockBuildPipeline) GetMetrics() interfaces.BuildMetrics {
	// Return mock metrics
	return &MockBuildMetrics{}
}

func (m *MockBuildPipeline) GetCache() interfaces.CacheStats {
	return &MockCacheStats{}
}

func (m *MockBuildPipeline) ClearCache() {
	// Mock implementation
}

// MockBuildMetrics provides mock build metrics for testing
type MockBuildMetrics struct{}

func (m *MockBuildMetrics) GetBuildCount() int64     { return 10 }
func (m *MockBuildMetrics) GetSuccessCount() int64   { return 8 }
func (m *MockBuildMetrics) GetFailureCount() int64   { return 2 }
func (m *MockBuildMetrics) GetAverageDuration() time.Duration { return 100 * time.Millisecond }
func (m *MockBuildMetrics) GetCacheHitRate() float64 { return 0.75 }
func (m *MockBuildMetrics) GetSuccessRate() float64  { return 0.8 }
func (m *MockBuildMetrics) Reset()                   {}

// MockCacheStats provides mock cache stats for testing
type MockCacheStats struct{}

func (m *MockCacheStats) GetSize() int64      { return 50 }
func (m *MockCacheStats) GetHits() int64      { return 100 }
func (m *MockCacheStats) GetMisses() int64    { return 20 }
func (m *MockCacheStats) GetHitRate() float64 { return 0.83 }
func (m *MockCacheStats) GetEvictions() int64 { return 5 }
func (m *MockCacheStats) Clear()              {}

// MockWebSocketManager provides mock WebSocket manager for testing
type MockWebSocketManager struct {
	broadcastMessages []UpdateMessage
	clientCount       int
}

func (m *MockWebSocketManager) BroadcastMessage(message UpdateMessage) {
	m.broadcastMessages = append(m.broadcastMessages, message)
}

func (m *MockWebSocketManager) GetConnectedClients() int {
	return m.clientCount
}

// createTestServiceDependencies creates valid dependencies for testing
func createTestServiceDependencies() ServiceDependencies {
	registry := &MockComponentRegistry{}
	scanner := &MockComponentScanner{registry: registry}
	
	return ServiceDependencies{
		Config: &config.Config{
			Components: config.ComponentsConfig{
				ScanPaths: []string{"./components", "./views"},
			},
			Server: config.ServerConfig{
				Open: false,
			},
		},
		Registry:      registry,
		FileWatcher:   &MockFileWatcher{},
		Scanner:       scanner,
		BuildPipeline: &MockBuildPipeline{},
		Renderer:      &renderer.ComponentRenderer{},
		Monitor:       nil, // Optional
		WSManager:     nil, // Use nil to simplify testing - will test separately
	}
}

// TestNewServiceOrchestrator_ValidInputs tests successful construction
func TestNewServiceOrchestrator_ValidInputs(t *testing.T) {
	deps := createTestServiceDependencies()

	orchestrator := NewServiceOrchestrator(deps)

	// Verify construction succeeded
	if orchestrator == nil {
		t.Fatal("NewServiceOrchestrator returned nil")
	}

	// Verify dependencies were stored
	if orchestrator.config != deps.Config {
		t.Error("Config was not stored correctly")
	}
	if orchestrator.registry != deps.Registry {
		t.Error("Registry was not stored correctly")
	}
	if orchestrator.fileWatcher != deps.FileWatcher {
		t.Error("FileWatcher was not stored correctly")
	}
	if orchestrator.scanner != deps.Scanner {
		t.Error("Scanner was not stored correctly")
	}
	if orchestrator.buildPipeline != deps.BuildPipeline {
		t.Error("BuildPipeline was not stored correctly")
	}
	if orchestrator.wsManager != deps.WSManager {
		t.Error("WSManager was not stored correctly")
	}

	// Verify initialization
	if orchestrator.ctx == nil {
		t.Error("Context was not initialized")
	}
	if orchestrator.cancel == nil {
		t.Error("Cancel function was not initialized")
	}
	if orchestrator.lastBuildErrors != nil {
		t.Error("LastBuildErrors should be nil initially")
	}

	// Clean shutdown
	orchestrator.Shutdown(context.Background())
}

// TestNewServiceOrchestrator_NilConfig tests panic on nil config
func TestNewServiceOrchestrator_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil config, but didn't panic")
		}
	}()

	deps := createTestServiceDependencies()
	deps.Config = nil
	NewServiceOrchestrator(deps)
}

// TestNewServiceOrchestrator_EmptyScanPaths tests panic on empty scan paths
func TestNewServiceOrchestrator_EmptyScanPaths(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty scan paths, but didn't panic")
		}
	}()

	deps := createTestServiceDependencies()
	deps.Config.Components.ScanPaths = []string{}
	NewServiceOrchestrator(deps)
}

// TestNewServiceOrchestrator_OptionalDependencies tests construction with optional dependencies
func TestNewServiceOrchestrator_OptionalDependencies(t *testing.T) {
	deps := createTestServiceDependencies()
	
	// Make some dependencies nil to test optional handling
	deps.Monitor = nil
	deps.Renderer = nil
	deps.WSManager = nil

	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	if orchestrator == nil {
		t.Fatal("NewServiceOrchestrator returned nil with optional dependencies nil")
	}

	// Verify nil dependencies are handled correctly
	if orchestrator.monitor != nil {
		t.Error("Monitor should be nil when not provided")
	}
	if orchestrator.renderer != nil {
		t.Error("Renderer should be nil when not provided")
	}
	if orchestrator.wsManager != nil {
		t.Error("WSManager should be nil when not provided")
	}
}

// TestServiceOrchestrator_Start tests service startup coordination
func TestServiceOrchestrator_Start(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orchestrator.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Verify services were started
	mockBuild := deps.BuildPipeline.(*MockBuildPipeline)
	if !mockBuild.started {
		t.Error("Build pipeline was not started")
	}

	mockWatcher := deps.FileWatcher.(*MockFileWatcher)
	if !mockWatcher.started {
		t.Error("File watcher was not started")
	}

	// Verify initial scan occurred
	mockScanner := deps.Scanner.(*MockComponentScanner)
	if len(mockScanner.scannedDirectories) == 0 {
		t.Error("Initial scan did not occur")
	}

	// Verify file watcher was configured
	if len(mockWatcher.filters) == 0 {
		t.Error("File watcher filters were not configured")
	}
	if len(mockWatcher.handlers) == 0 {
		t.Error("File watcher handlers were not configured")
	}
	if len(mockWatcher.watchPaths) == 0 {
		t.Error("File watcher paths were not configured")
	}
}

// TestServiceOrchestrator_HandleBuildResult tests build result processing
func TestServiceOrchestrator_HandleBuildResult(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	// Since WSManager is nil in our test deps, we'll test without WebSocket broadcasting

	// Test successful build result
	successResult := build.BuildResult{
		ParsedErrors: nil,
		Component:    &types.ComponentInfo{Name: "TestComponent"},
		Error:        nil,
		Duration:     100 * time.Millisecond,
		CacheHit:     false,
	}

	orchestrator.handleBuildResult(successResult)

	// Verify build state was updated
	buildErrors := orchestrator.GetLastBuildErrors()
	if buildErrors != nil {
		t.Error("Last build errors should be nil for successful build")
	}

	// Test failed build result
	failedResult := build.BuildResult{
		ParsedErrors: []*errors.ParsedError{
			{Message: "Test error", Line: 1, Column: 1},
		},
		Component: &types.ComponentInfo{Name: "TestComponent"},
		Error:     fmt.Errorf("build failed"),
		Duration:  200 * time.Millisecond,
		CacheHit:  false,
	}

	orchestrator.handleBuildResult(failedResult)

	// Verify build errors were stored
	buildErrors = orchestrator.GetLastBuildErrors()
	if len(buildErrors) != 1 {
		t.Errorf("Expected 1 build error, got %d", len(buildErrors))
	}
}

// TestServiceOrchestrator_GetBuildMetrics tests build metrics retrieval
func TestServiceOrchestrator_GetBuildMetrics(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	metrics := orchestrator.GetBuildMetrics()
	if metrics == nil {
		t.Error("GetBuildMetrics returned nil")
	}

	// Verify metrics interface methods work
	if metrics.GetBuildCount() == 0 {
		t.Error("Expected non-zero build count from mock")
	}
}

// TestServiceOrchestrator_GetConnectedWebSocketClients tests client count retrieval
func TestServiceOrchestrator_GetConnectedWebSocketClients(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	// Since our test dependencies use nil WSManager, expect 0 clients
	clientCount := orchestrator.GetConnectedWebSocketClients()
	if clientCount != 0 {
		t.Errorf("Expected 0 WebSocket clients with nil manager, got %d", clientCount)
	}
}

// TestServiceOrchestrator_IsHealthy tests health checking
func TestServiceOrchestrator_IsHealthy(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	// Should be healthy with essential services
	if !orchestrator.IsHealthy() {
		t.Error("Orchestrator should be healthy with all services")
	}

	// Test with missing essential services
	orchestrator.registry = nil
	if orchestrator.IsHealthy() {
		t.Error("Orchestrator should be unhealthy without registry")
	}

	orchestrator.scanner = nil
	if orchestrator.IsHealthy() {
		t.Error("Orchestrator should be unhealthy without scanner")
	}
}

// TestServiceOrchestrator_GetServiceStatus tests service status reporting
func TestServiceOrchestrator_GetServiceStatus(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	status := orchestrator.GetServiceStatus()

	// Verify all expected fields are present
	expectedFields := []string{
		"registry_available",
		"scanner_available", 
		"build_pipeline_available",
		"file_watcher_available",
		"websocket_manager_available",
		"renderer_available",
		"monitor_available",
		"component_count",
		"websocket_clients",
	}

	for _, field := range expectedFields {
		if _, exists := status[field]; !exists {
			t.Errorf("Status field %s is missing", field)
		}
	}

	// Verify specific values
	if status["registry_available"] != true {
		t.Error("Registry should be reported as available")
	}
	if status["monitor_available"] != false {
		t.Error("Monitor should be reported as unavailable (nil)")
	}
	if status["websocket_clients"] != 0 {
		t.Errorf("Expected 0 WebSocket clients in status (nil manager), got %v", status["websocket_clients"])
	}
}

// TestServiceOrchestrator_Shutdown tests graceful shutdown
func TestServiceOrchestrator_Shutdown(t *testing.T) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the orchestrator first
	orchestrator.Start(ctx)

	// Perform shutdown
	err := orchestrator.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify services were shut down
	mockBuild := deps.BuildPipeline.(*MockBuildPipeline)
	if !mockBuild.stopped {
		t.Error("Build pipeline was not stopped")
	}

	mockWatcher := deps.FileWatcher.(*MockFileWatcher)
	if !mockWatcher.stopped {
		t.Error("File watcher was not stopped")
	}

	// Test idempotent shutdown
	err = orchestrator.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown call failed: %v", err)
	}
}

// TestServiceOrchestrator_OpenBrowser tests browser opening functionality
func TestServiceOrchestrator_OpenBrowser(t *testing.T) {
	deps := createTestServiceDependencies()
	
	// Test with browser opening disabled
	deps.Config.Server.Open = false
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	// Should not attempt to open browser
	orchestrator.OpenBrowser("http://localhost:8080")

	// Test with browser opening enabled
	deps.Config.Server.Open = true
	orchestrator2 := NewServiceOrchestrator(deps)
	defer orchestrator2.Shutdown(context.Background())

	// This will attempt to open browser but should not fail in tests
	orchestrator2.OpenBrowser("http://localhost:8080")
}

// BenchmarkServiceOrchestrator_HandleBuildResult benchmarks build result processing
func BenchmarkServiceOrchestrator_HandleBuildResult(b *testing.B) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	buildResult := build.BuildResult{
		ParsedErrors: []*errors.ParsedError{
			{Message: "Benchmark error", Line: 1, Column: 1},
		},
		Component: &types.ComponentInfo{Name: "BenchmarkComponent"},
		Error:     fmt.Errorf("benchmark error"),
		Duration:  50 * time.Millisecond,
		CacheHit:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orchestrator.handleBuildResult(buildResult)
	}
}

// BenchmarkServiceOrchestrator_GetServiceStatus benchmarks status reporting
func BenchmarkServiceOrchestrator_GetServiceStatus(b *testing.B) {
	deps := createTestServiceDependencies()
	orchestrator := NewServiceOrchestrator(deps)
	defer orchestrator.Shutdown(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orchestrator.GetServiceStatus()
	}
}