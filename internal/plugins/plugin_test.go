package plugins

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// MockPlugin is a test plugin implementation
type MockPlugin struct {
	name           string
	version        string
	initialized    bool
	shutdownCalled bool
	health         PluginHealth
}

func (mp *MockPlugin) Name() string        { return mp.name }
func (mp *MockPlugin) Version() string     { return mp.version }
func (mp *MockPlugin) Description() string { return "Mock plugin for testing" }
func (mp *MockPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	mp.initialized = true
	return nil
}
func (mp *MockPlugin) Shutdown(ctx context.Context) error { mp.shutdownCalled = true; return nil }
func (mp *MockPlugin) Health() PluginHealth               { return mp.health }

// MockComponentPlugin extends MockPlugin with component functionality
type MockComponentPlugin struct {
	MockPlugin
	priority int
}

func (mcp *MockComponentPlugin) HandleComponent(ctx context.Context, component *types.ComponentInfo) (*types.ComponentInfo, error) {
	// Add test metadata
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["processed_by"] = mcp.name
	return component, nil
}

func (mcp *MockComponentPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (mcp *MockComponentPlugin) Priority() int                 { return mcp.priority }

// MockBuildPlugin extends MockPlugin with build functionality
type MockBuildPlugin struct {
	MockPlugin
	preBuildCalled  bool
	postBuildCalled bool
}

func (mbp *MockBuildPlugin) PreBuild(ctx context.Context, components []*types.ComponentInfo) error {
	mbp.preBuildCalled = true
	return nil
}

func (mbp *MockBuildPlugin) PostBuild(ctx context.Context, components []*types.ComponentInfo, buildResult BuildResult) error {
	mbp.postBuildCalled = true
	return nil
}

func (mbp *MockBuildPlugin) TransformBuildCommand(ctx context.Context, command []string) ([]string, error) {
	// Add a test flag
	return append(command, "--test-flag"), nil
}

func TestPluginManager_RegisterPlugin(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		health:  PluginHealth{Status: HealthStatusHealthy},
	}

	config := PluginConfig{
		Name:    "test-plugin",
		Enabled: true,
		Config:  map[string]interface{}{"key": "value"},
	}

	err := pm.RegisterPlugin(plugin, config)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	if !plugin.initialized {
		t.Error("Plugin should have been initialized")
	}

	// Try to register the same plugin again
	err = pm.RegisterPlugin(plugin, config)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin")
	}
}

func TestPluginManager_UnregisterPlugin(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		health:  PluginHealth{Status: HealthStatusHealthy},
	}

	config := PluginConfig{
		Name:    "test-plugin",
		Enabled: true,
	}

	// Register plugin
	err := pm.RegisterPlugin(plugin, config)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Unregister plugin
	err = pm.UnregisterPlugin("test-plugin")
	if err != nil {
		t.Fatalf("Failed to unregister plugin: %v", err)
	}

	if !plugin.shutdownCalled {
		t.Error("Plugin shutdown should have been called")
	}

	// Try to get unregistered plugin
	_, err = pm.GetPlugin("test-plugin")
	if err == nil {
		t.Error("Expected error when getting unregistered plugin")
	}
}

func TestPluginManager_ComponentProcessing(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	// Register component plugins with different priorities
	plugin1 := &MockComponentPlugin{
		MockPlugin: MockPlugin{
			name:    "plugin-1",
			version: "1.0.0",
			health:  PluginHealth{Status: HealthStatusHealthy},
		},
		priority: 1,
	}

	plugin2 := &MockComponentPlugin{
		MockPlugin: MockPlugin{
			name:    "plugin-2",
			version: "1.0.0",
			health:  PluginHealth{Status: HealthStatusHealthy},
		},
		priority: 2,
	}

	config := PluginConfig{Enabled: true}

	err := pm.RegisterPlugin(plugin1, config)
	if err != nil {
		t.Fatalf("Failed to register plugin1: %v", err)
	}

	err = pm.RegisterPlugin(plugin2, config)
	if err != nil {
		t.Fatalf("Failed to register plugin2: %v", err)
	}

	// Test component processing
	component := &types.ComponentInfo{
		Name:     "TestComponent",
		Package:  "test",
		FilePath: "/test/component.templ",
	}

	processedComponent, err := pm.ProcessComponent(context.Background(), component)
	if err != nil {
		t.Fatalf("Failed to process component: %v", err)
	}

	if processedComponent.Metadata == nil {
		t.Fatal("Component metadata should not be nil")
	}

	// Since plugin1 has lower priority, it should execute first
	// but plugin2 should overwrite the metadata
	processedBy, ok := processedComponent.Metadata["processed_by"].(string)
	if !ok || processedBy != "plugin-2" {
		t.Errorf("Expected component to be processed by plugin-2, got %v", processedBy)
	}
}

func TestPluginManager_ListPlugins(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	plugin1 := &MockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
		health:  PluginHealth{Status: HealthStatusHealthy},
	}

	plugin2 := &MockPlugin{
		name:    "plugin-2",
		version: "2.0.0",
		health:  PluginHealth{Status: HealthStatusDegraded},
	}

	config1 := PluginConfig{Name: "plugin-1", Enabled: true}
	config2 := PluginConfig{Name: "plugin-2", Enabled: false}

	pm.RegisterPlugin(plugin1, config1)
	pm.RegisterPlugin(plugin2, config2)

	plugins := pm.ListPlugins()
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}

	// Check plugin info
	for _, pluginInfo := range plugins {
		switch pluginInfo.Name {
		case "plugin-1":
			if !pluginInfo.Enabled {
				t.Error("Plugin-1 should be enabled")
			}
			if pluginInfo.Version != "1.0.0" {
				t.Errorf("Expected version 1.0.0, got %s", pluginInfo.Version)
			}
		case "plugin-2":
			if pluginInfo.Enabled {
				t.Error("Plugin-2 should be disabled")
			}
			if pluginInfo.Version != "2.0.0" {
				t.Errorf("Expected version 2.0.0, got %s", pluginInfo.Version)
			}
		default:
			t.Errorf("Unexpected plugin: %s", pluginInfo.Name)
		}
	}
}

func TestPluginManager_HealthChecks(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	plugin := &MockPlugin{
		name:    "health-test-plugin",
		version: "1.0.0",
		health: PluginHealth{
			Status:    HealthStatusHealthy,
			LastCheck: time.Now(),
		},
	}

	config := PluginConfig{Enabled: true}
	pm.RegisterPlugin(plugin, config)

	// Start health checks with short interval
	pm.StartHealthChecks(100 * time.Millisecond)

	// Wait for at least one health check
	time.Sleep(200 * time.Millisecond)

	plugins := pm.ListPlugins()
	if len(plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(plugins))
	}

	// Health check should have updated the timestamp
	if plugins[0].Health.LastCheck.IsZero() {
		t.Error("Health check timestamp should not be zero")
	}
}

func TestBuildPlugin_Lifecycle(t *testing.T) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	buildPlugin := &MockBuildPlugin{
		MockPlugin: MockPlugin{
			name:    "build-test-plugin",
			version: "1.0.0",
			health:  PluginHealth{Status: HealthStatusHealthy},
		},
	}

	config := PluginConfig{Enabled: true}
	err := pm.RegisterPlugin(buildPlugin, config)
	if err != nil {
		t.Fatalf("Failed to register build plugin: %v", err)
	}

	// Test PreBuild
	components := []*types.ComponentInfo{
		{Name: "Component1", Package: "test"},
		{Name: "Component2", Package: "test"},
	}

	if len(pm.buildPlugins) != 1 {
		t.Fatalf("Expected 1 build plugin, got %d", len(pm.buildPlugins))
	}

	err = pm.buildPlugins[0].PreBuild(context.Background(), components)
	if err != nil {
		t.Fatalf("PreBuild failed: %v", err)
	}

	if !buildPlugin.preBuildCalled {
		t.Error("PreBuild should have been called")
	}

	// Test PostBuild
	buildResult := BuildResult{
		Success:         true,
		Duration:        time.Second,
		ComponentsBuilt: 2,
	}

	err = pm.buildPlugins[0].PostBuild(context.Background(), components, buildResult)
	if err != nil {
		t.Fatalf("PostBuild failed: %v", err)
	}

	if !buildPlugin.postBuildCalled {
		t.Error("PostBuild should have been called")
	}

	// Test TransformBuildCommand
	originalCommand := []string{"go", "build"}
	transformedCommand, err := pm.buildPlugins[0].TransformBuildCommand(context.Background(), originalCommand)
	if err != nil {
		t.Fatalf("TransformBuildCommand failed: %v", err)
	}

	if len(transformedCommand) != 3 || transformedCommand[2] != "--test-flag" {
		t.Errorf("Expected command to have test flag added, got %v", transformedCommand)
	}
}

func TestPluginHealth_Status(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected HealthStatus
	}{
		{"Healthy", HealthStatusHealthy, HealthStatusHealthy},
		{"Unhealthy", HealthStatusUnhealthy, HealthStatusUnhealthy},
		{"Degraded", HealthStatusDegraded, HealthStatusDegraded},
		{"Unknown", HealthStatusUnknown, HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := PluginHealth{
				Status:    tt.status,
				LastCheck: time.Now(),
			}

			if health.Status != tt.expected {
				t.Errorf("Expected status %s, got %s", tt.expected, health.Status)
			}
		})
	}
}

func TestPluginConfig_Validation(t *testing.T) {
	config := PluginConfig{
		Name:    "test-plugin",
		Enabled: true,
		Config: map[string]interface{}{
			"setting1": "value1",
			"setting2": 42,
		},
		Settings: PluginSettings{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			LogLevel:   "info",
			ResourceLimits: ResourceLimits{
				MaxMemoryMB:        100,
				MaxCPUPercent:      50.0,
				MaxGoroutines:      100,
				MaxFileDescriptors: 50,
			},
		},
	}

	if config.Name == "" {
		t.Error("Plugin name should not be empty")
	}

	if config.Settings.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}

	if config.Settings.ResourceLimits.MaxMemoryMB <= 0 {
		t.Error("Max memory should be positive")
	}
}

func BenchmarkPluginManager_ProcessComponent(b *testing.B) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	// Register multiple component plugins
	for i := 0; i < 5; i++ {
		plugin := &MockComponentPlugin{
			MockPlugin: MockPlugin{
				name:    fmt.Sprintf("plugin-%d", i),
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			priority: i,
		}

		config := PluginConfig{Enabled: true}
		pm.RegisterPlugin(plugin, config)
	}

	component := &types.ComponentInfo{
		Name:     "BenchmarkComponent",
		Package:  "test",
		FilePath: "/test/component.templ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pm.ProcessComponent(context.Background(), component)
		if err != nil {
			b.Fatalf("Failed to process component: %v", err)
		}
	}
}

func BenchmarkPluginManager_ListPlugins(b *testing.B) {
	pm := NewPluginManager()
	defer pm.Shutdown()

	// Register many plugins
	for i := 0; i < 100; i++ {
		plugin := &MockPlugin{
			name:    fmt.Sprintf("plugin-%d", i),
			version: "1.0.0",
			health:  PluginHealth{Status: HealthStatusHealthy},
		}

		config := PluginConfig{Enabled: true}
		pm.RegisterPlugin(plugin, config)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugins := pm.ListPlugins()
		if len(plugins) != 100 {
			b.Fatalf("Expected 100 plugins, got %d", len(plugins))
		}
	}
}
