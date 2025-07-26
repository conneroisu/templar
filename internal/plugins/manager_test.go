package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
)

// TestEnhancedPluginManager tests the basic functionality of the enhanced plugin manager
func TestEnhancedPluginManager(t *testing.T) {
	ctx := context.Background()

	// Create test configuration
	cfg := &config.PluginsConfig{
		Enabled:        []string{"test-plugin"},
		Disabled:       []string{},
		DiscoveryPaths: []string{"./test-plugins"},
		Configurations: make(map[string]config.PluginConfigMap),
	}

	// Create mock logger
	logger := &MockLogger{}

	// Create error handler
	errorHandler := errors.NewErrorHandler(logger, nil)

	// Create registry
	reg := registry.NewComponentRegistry()

	// Create enhanced plugin manager
	epm := NewEnhancedPluginManager(cfg, logger, errorHandler, reg)

	// Create integrations
	buildAdapter := NewBuildPipelineAdapter()
	serverAdapter := NewServerAdapter()
	watcherAdapter := NewWatcherAdapter()

	epm.SetIntegrations(buildAdapter, serverAdapter, watcherAdapter)

	// Test plugin registration
	testPlugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		health: PluginHealth{
			Status:    HealthStatusHealthy,
			LastCheck: time.Now(),
		},
	}

	err := epm.SetBuiltinPlugins([]Plugin{testPlugin})
	if err != nil {
		t.Fatalf("Failed to register builtin plugins: %v", err)
	}

	// Test initialization
	err = epm.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize plugin manager: %v", err)
	}

	// Test getting plugin info
	pluginInfo := epm.GetPluginInfo()
	if len(pluginInfo) == 0 {
		t.Error("Expected at least one plugin to be discovered")
	}

	// Test getting loaded plugins
	loadedPlugins := epm.GetLoadedPlugins()
	if len(loadedPlugins) == 0 {
		t.Error("Expected at least one plugin to be loaded")
	}

	// Test plugin state
	state := epm.GetPluginState("test-plugin")
	if state == PluginStateUnknown {
		t.Error("Expected plugin state to be known")
	}

	// Test shutdown
	err = epm.Shutdown(ctx)
	if err != nil {
		t.Errorf("Failed to shutdown plugin manager: %v", err)
	}
}

// TestBuildPipelineAdapter tests the build pipeline adapter
func TestBuildPipelineAdapter(t *testing.T) {
	// Test build pipeline adapter
	buildAdapter := NewBuildPipelineAdapter()

	mockBuildPlugin := &MockBuildPlugin{
		MockPlugin: MockPlugin{
			name:    "mock-build",
			version: "1.0.0",
			health: PluginHealth{
				Status:    HealthStatusHealthy,
				LastCheck: time.Now(),
			},
		},
	}

	err := buildAdapter.RegisterPreBuildHook(mockBuildPlugin)
	if err != nil {
		t.Errorf("Failed to register pre-build hook: %v", err)
	}

	err = buildAdapter.RegisterPostBuildHook(mockBuildPlugin)
	if err != nil {
		t.Errorf("Failed to register post-build hook: %v", err)
	}

	// Test removal
	err = buildAdapter.RemovePlugin("mock-build")
	if err != nil {
		t.Errorf("Failed to remove plugin: %v", err)
	}
}

// Mock implementations for testing (using existing mocks)

type MockLogger struct{}

func (ml *MockLogger) Debug(ctx context.Context, msg string, fields ...interface{})            {}
func (ml *MockLogger) Info(ctx context.Context, msg string, fields ...interface{})             {}
func (ml *MockLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{})  {}
func (ml *MockLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {}
func (ml *MockLogger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {}

func (ml *MockLogger) With(
	fields ...interface{},
) logging.Logger {
	return ml
}

func (ml *MockLogger) WithComponent(
	component string,
) logging.Logger {
	return ml
}
