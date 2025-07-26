package plugins

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginBuildPipelineIntegration tests the complete integration between plugins and the build pipeline
func TestPluginBuildPipelineIntegration(t *testing.T) {
	t.Run("plugin lifecycle during build process", func(t *testing.T) {
		manager := NewPluginManager()

		// Create a build-aware plugin that tracks build lifecycle events
		buildPlugin := &MockBuildLifecyclePlugin{
			MockPlugin: MockPlugin{
				name:    "build-lifecycle-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			events: make([]string, 0),
		}

		config := PluginConfig{
			Name:    "build-lifecycle-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(buildPlugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Simulate build pipeline components
		components := []*types.ComponentInfo{
			{
				Name:     "Component1",
				FilePath: "/test/component1.templ",
				Package:  "test",
				Metadata: make(map[string]interface{}),
			},
			{
				Name:     "Component2",
				FilePath: "/test/component2.templ",
				Package:  "test",
				Metadata: make(map[string]interface{}),
			},
		}

		// Test pre-build hook
		if len(manager.buildPlugins) > 0 {
			err = manager.buildPlugins[0].PreBuild(ctx, components)
			assert.NoError(t, err)
			assert.Contains(t, buildPlugin.events, "PreBuild")
		}

		// Process components through plugin system (simulating build pipeline component processing)
		for _, component := range components {
			processedComponent, err := manager.ProcessComponent(ctx, component)
			assert.NoError(t, err)
			assert.NotNil(t, processedComponent)
		}

		// Test post-build hook with successful build
		buildResult := BuildResult{
			Success:         true,
			ComponentsBuilt: len(components),
			Output:          "Build completed successfully",
		}
		if len(manager.buildPlugins) > 0 {
			err = manager.buildPlugins[0].PostBuild(ctx, components, buildResult)
			assert.NoError(t, err)
			assert.Contains(t, buildPlugin.events, "PostBuild")
		}

		// Verify build lifecycle events were called in correct order
		expectedEvents := []string{"Initialize", "PreBuild", "PostBuild"}
		for _, expectedEvent := range expectedEvents {
			assert.Contains(
				t,
				buildPlugin.events,
				expectedEvent,
				"Event %s should have been called",
				expectedEvent,
			)
		}

		err = manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("plugin error handling during build failures", func(t *testing.T) {
		manager := NewPluginManager()

		// Create a plugin that handles build failures
		buildPlugin := &MockBuildLifecyclePlugin{
			MockPlugin: MockPlugin{
				name:    "build-failure-handler",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			events: make([]string, 0),
		}

		config := PluginConfig{
			Name:    "build-failure-handler",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(buildPlugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		components := []*types.ComponentInfo{
			{
				Name:     "FailingComponent",
				FilePath: "/test/failing.templ",
				Package:  "test",
			},
		}

		// Simulate build failure
		buildResult := BuildResult{
			Success:         false,
			Error:           "compilation failed",
			ComponentsBuilt: 0,
			Output:          "Build failed during compilation",
		}

		// Test that post-build hook handles failure gracefully
		if len(manager.buildPlugins) > 0 {
			err = manager.buildPlugins[0].PostBuild(ctx, components, buildResult)
			assert.NoError(t, err, "Plugin should handle build failures gracefully")
		}

		err = manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("concurrent build operations with plugins", func(t *testing.T) {
		manager := NewPluginManager()

		// Register multiple plugins that process components
		plugins := []*MockConcurrentBuildPlugin{
			{
				MockPlugin: MockPlugin{
					name:    "concurrent-plugin-1",
					version: "1.0.0",
					health:  PluginHealth{Status: HealthStatusHealthy},
				},
				processedComponents: make(map[string]int),
				mutex:               sync.RWMutex{},
			},
			{
				MockPlugin: MockPlugin{
					name:    "concurrent-plugin-2",
					version: "1.0.0",
					health:  PluginHealth{Status: HealthStatusHealthy},
				},
				processedComponents: make(map[string]int),
				mutex:               sync.RWMutex{},
			},
		}

		for i, plugin := range plugins {
			config := PluginConfig{
				Name:    fmt.Sprintf("concurrent-plugin-%d", i+1),
				Enabled: true,
				Config:  make(map[string]interface{}),
			}
			err := manager.RegisterPlugin(plugin, config)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// Simulate concurrent build operations
		const numComponents = 20
		const numConcurrentBuilds = 5

		var wg sync.WaitGroup
		errChan := make(chan error, numConcurrentBuilds*numComponents)

		for buildID := 0; buildID < numConcurrentBuilds; buildID++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for compID := 0; compID < numComponents; compID++ {
					component := &types.ComponentInfo{
						Name:     fmt.Sprintf("Build%d-Component%d", id, compID),
						FilePath: fmt.Sprintf("/test/build%d/comp%d.templ", id, compID),
						Package:  "test",
						Metadata: make(map[string]interface{}),
					}

					_, err := manager.ProcessComponent(ctx, component)
					if err != nil {
						errChan <- fmt.Errorf("build %d component %d: %w", id, compID, err)
					}
				}
			}(buildID)
		}

		wg.Wait()
		close(errChan)

		// Check for any errors
		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}
		assert.Empty(t, errors, "Concurrent build operations should not produce errors")

		// Verify that plugins processed components correctly
		for _, plugin := range plugins {
			plugin.mutex.RLock()
			totalProcessed := 0
			for _, count := range plugin.processedComponents {
				totalProcessed += count
			}
			plugin.mutex.RUnlock()

			assert.Equal(t, numConcurrentBuilds*numComponents, totalProcessed,
				"Plugin %s should have processed all components", plugin.name)
		}

		err := manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("plugin resource management during builds", func(t *testing.T) {
		manager := NewPluginManager()

		// Create a plugin that tracks resource usage
		resourcePlugin := &MockResourceTrackingPlugin{
			MockPlugin: MockPlugin{
				name:    "resource-tracker",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			allocatedMemory: 0,
			peakMemory:      0,
		}

		config := PluginConfig{
			Name:    "resource-tracker",
			Enabled: true,
			Config: map[string]interface{}{
				"max_memory": 1024 * 1024, // 1MB limit
			},
		}
		err := manager.RegisterPlugin(resourcePlugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Process many components to test resource management
		for i := 0; i < 100; i++ {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("Component%d", i),
				FilePath: fmt.Sprintf("/test/component%d.templ", i),
				Package:  "test",
				Metadata: make(map[string]interface{}),
			}

			_, err := manager.ProcessComponent(ctx, component)
			assert.NoError(t, err)
		}

		// Verify resource management
		assert.LessOrEqual(t, resourcePlugin.allocatedMemory, int64(1024*1024),
			"Plugin should not exceed memory limits")
		assert.Greater(t, resourcePlugin.peakMemory, int64(0),
			"Plugin should have allocated some memory")

		err = manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("plugin communication during build coordination", func(t *testing.T) {
		manager := NewPluginManager()

		// Create plugins that need to coordinate during builds
		coordinatorPlugin := &MockCoordinatorPlugin{
			MockPlugin: MockPlugin{
				name:    "build-coordinator",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			messages: make(chan string, 100),
		}

		workerPlugin := &MockWorkerPlugin{
			MockPlugin: MockPlugin{
				name:    "build-worker",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			coordinator: coordinatorPlugin,
		}

		// Register coordinator first
		config1 := PluginConfig{
			Name:    "build-coordinator",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(coordinatorPlugin, config1)
		require.NoError(t, err)

		// Register worker
		config2 := PluginConfig{
			Name:    "build-worker",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err = manager.RegisterPlugin(workerPlugin, config2)
		require.NoError(t, err)

		ctx := context.Background()

		// Process components that require coordination
		components := []*types.ComponentInfo{
			{Name: "SharedComponent1", FilePath: "/test/shared1.templ", Package: "test"},
			{Name: "SharedComponent2", FilePath: "/test/shared2.templ", Package: "test"},
		}

		for _, component := range components {
			_, err := manager.ProcessComponent(ctx, component)
			assert.NoError(t, err)
		}

		// Verify coordination occurred
		messageCount := len(coordinatorPlugin.messages)
		assert.Greater(t, messageCount, 0, "Plugins should have communicated during build")

		err = manager.Shutdown()
		assert.NoError(t, err)
	})
}

// TestPluginBuildPipelinePerformance tests performance aspects of plugin-build integration
func TestPluginBuildPipelinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("plugin overhead in build pipeline", func(t *testing.T) {
		manager := NewPluginManager()

		// Register a lightweight plugin
		plugin := &MockPerformancePlugin{
			MockPlugin: MockPlugin{
				name:    "performance-test-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			processingTimes: make([]time.Duration, 0),
		}

		config := PluginConfig{
			Name:    "performance-test-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(plugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Measure processing time with plugin
		start := time.Now()
		const numComponents = 1000

		for i := 0; i < numComponents; i++ {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("PerfComponent%d", i),
				FilePath: fmt.Sprintf("/test/perf%d.templ", i),
				Package:  "test",
			}

			_, err := manager.ProcessComponent(ctx, component)
			assert.NoError(t, err)
		}

		totalDuration := time.Since(start)
		avgPerComponent := totalDuration / numComponents

		// Performance assertions
		assert.Less(t, avgPerComponent, 1*time.Millisecond,
			"Average plugin processing time should be under 1ms per component")
		assert.Len(t, plugin.processingTimes, numComponents,
			"Plugin should have tracked processing times for all components")

		// Check for performance consistency
		if len(plugin.processingTimes) > 10 {
			var totalTime time.Duration
			for _, duration := range plugin.processingTimes {
				totalTime += duration
			}
			avgTime := totalTime / time.Duration(len(plugin.processingTimes))

			assert.Less(t, avgTime, 500*time.Microsecond,
				"Average plugin processing time should be under 500µs")
		}

		err = manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("plugin memory efficiency during builds", func(t *testing.T) {
		manager := NewPluginManager()

		// Create memory-efficient plugin
		plugin := &MockMemoryEfficientPlugin{
			MockPlugin: MockPlugin{
				name:    "memory-efficient-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			memoryUsage: make([]int64, 0),
		}

		config := PluginConfig{
			Name:    "memory-efficient-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(plugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Process components and monitor memory usage
		const numComponents = 500
		for i := 0; i < numComponents; i++ {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("MemComponent%d", i),
				FilePath: fmt.Sprintf("/test/mem%d.templ", i),
				Package:  "test",
				Metadata: make(map[string]interface{}),
			}

			_, err := manager.ProcessComponent(ctx, component)
			assert.NoError(t, err)

			// Check memory usage periodically
			if i%100 == 0 && len(plugin.memoryUsage) > 0 {
				currentMemory := plugin.memoryUsage[len(plugin.memoryUsage)-1]
				assert.Less(t, currentMemory, int64(10*1024*1024), // 10MB limit
					"Plugin memory usage should stay under 10MB")
			}
		}

		err = manager.Shutdown()
		assert.NoError(t, err)
	})
}

// TestPluginBuildPipelineErrorHandling tests error scenarios in plugin-build integration
func TestPluginBuildPipelineErrorHandling(t *testing.T) {
	t.Run("plugin failure during build process", func(t *testing.T) {
		manager := NewPluginManager()

		// Create a plugin that fails intermittently
		failingPlugin := &MockIntermittentFailurePlugin{
			MockPlugin: MockPlugin{
				name:    "intermittent-failure-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			failureRate: 0.3, // 30% failure rate
			callCount:   0,
		}

		config := PluginConfig{
			Name:    "intermittent-failure-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(failingPlugin, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Process components and expect some failures
		const numComponents = 20
		successCount := 0
		failureCount := 0

		for i := 0; i < numComponents; i++ {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("Component%d", i),
				FilePath: fmt.Sprintf("/test/component%d.templ", i),
				Package:  "test",
			}

			_, err := manager.ProcessComponent(ctx, component)
			if err != nil {
				failureCount++
			} else {
				successCount++
			}
		}

		// Verify that failures were handled gracefully
		assert.Greater(
			t,
			successCount,
			0,
			"Some components should have been processed successfully",
		)
		assert.Greater(
			t,
			failureCount,
			0,
			"Some failures should have occurred due to plugin issues",
		)
		assert.Equal(
			t,
			numComponents,
			successCount+failureCount,
			"All components should have been processed",
		)

		err = manager.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("build pipeline recovery from plugin errors", func(t *testing.T) {
		manager := NewPluginManager()

		// Register a good plugin and a bad plugin
		goodPlugin := &MockComponentPlugin{
			MockPlugin: MockPlugin{
				name:    "good-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusHealthy},
			},
			priority: 1,
		}

		badPlugin := &MockFailingPlugin{
			MockPlugin: MockPlugin{
				name:    "bad-plugin",
				version: "1.0.0",
				health:  PluginHealth{Status: HealthStatusUnhealthy},
			},
		}

		// Register good plugin first
		config1 := PluginConfig{
			Name:    "good-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		err := manager.RegisterPlugin(goodPlugin, config1)
		require.NoError(t, err)

		// Register bad plugin - this might fail during registration
		config2 := PluginConfig{
			Name:    "bad-plugin",
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
		_ = manager.RegisterPlugin(badPlugin, config2) // Ignore error

		ctx := context.Background()

		// Verify that build pipeline can still function with good plugin
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "/test/component.templ",
			Package:  "test",
			Metadata: make(map[string]interface{}),
		}

		processedComponent, err := manager.ProcessComponent(ctx, component)
		// Should succeed with good plugin even if bad plugin failed
		assert.NoError(t, err)
		assert.NotNil(t, processedComponent)

		err = manager.Shutdown()
		assert.NoError(t, err)
	})
}

// Mock plugin implementations for testing plugin-build pipeline integration

type MockBuildLifecyclePlugin struct {
	MockPlugin
	events []string
	mutex  sync.Mutex
}

func (m *MockBuildLifecyclePlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.events = append(m.events, "Initialize")
	m.initialized = true
	return nil
}

func (m *MockBuildLifecyclePlugin) PreBuild(
	ctx context.Context,
	components []*types.ComponentInfo,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.events = append(m.events, "PreBuild")
	return nil
}

func (m *MockBuildLifecyclePlugin) PostBuild(
	ctx context.Context,
	components []*types.ComponentInfo,
	result BuildResult,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.events = append(m.events, "PostBuild")
	return nil
}

func (m *MockBuildLifecyclePlugin) TransformBuildCommand(
	ctx context.Context,
	command []string,
) ([]string, error) {
	return command, nil
}

type MockConcurrentBuildPlugin struct {
	MockPlugin
	processedComponents map[string]int
	mutex               sync.RWMutex
}

func (m *MockConcurrentBuildPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockConcurrentBuildPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.processedComponents[component.Name]++
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["processed_by"] = m.name
	return component, nil
}

func (m *MockConcurrentBuildPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockConcurrentBuildPlugin) Priority() int                 { return 1 }

type MockResourceTrackingPlugin struct {
	MockPlugin
	allocatedMemory int64
	peakMemory      int64
	mutex           sync.Mutex
}

func (m *MockResourceTrackingPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockResourceTrackingPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Simulate memory allocation
	componentSize := int64(len(component.Name) + len(component.FilePath) + 1024)
	m.allocatedMemory += componentSize

	if m.allocatedMemory > m.peakMemory {
		m.peakMemory = m.allocatedMemory
	}

	// Simulate memory cleanup for older components
	if m.allocatedMemory > 512*1024 { // 512KB
		m.allocatedMemory = m.allocatedMemory / 2 // Simple cleanup simulation
	}

	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["memory_tracked"] = true
	return component, nil
}

func (m *MockResourceTrackingPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockResourceTrackingPlugin) Priority() int                 { return 1 }

type MockCoordinatorPlugin struct {
	MockPlugin
	messages chan string
}

func (m *MockCoordinatorPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockCoordinatorPlugin) SendMessage(message string) {
	select {
	case m.messages <- message:
	default:
		// Channel full, ignore
	}
}

func (m *MockCoordinatorPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	m.SendMessage(fmt.Sprintf("coordinator processing %s", component.Name))
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["coordinated"] = true
	return component, nil
}

func (m *MockCoordinatorPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockCoordinatorPlugin) Priority() int                 { return 1 }

type MockWorkerPlugin struct {
	MockPlugin
	coordinator *MockCoordinatorPlugin
}

func (m *MockWorkerPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockWorkerPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	if m.coordinator != nil {
		m.coordinator.SendMessage(fmt.Sprintf("worker processed %s", component.Name))
	}
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["worker_processed"] = true
	return component, nil
}

func (m *MockWorkerPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockWorkerPlugin) Priority() int                 { return 2 }

type MockPerformancePlugin struct {
	MockPlugin
	processingTimes []time.Duration
	mutex           sync.Mutex
}

func (m *MockPerformancePlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockPerformancePlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	start := time.Now()

	// Simulate minimal processing
	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["performance_tested"] = true

	duration := time.Since(start)

	m.mutex.Lock()
	m.processingTimes = append(m.processingTimes, duration)
	m.mutex.Unlock()

	return component, nil
}

func (m *MockPerformancePlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockPerformancePlugin) Priority() int                 { return 1 }

type MockMemoryEfficientPlugin struct {
	MockPlugin
	memoryUsage []int64
	mutex       sync.Mutex
}

func (m *MockMemoryEfficientPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockMemoryEfficientPlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Simulate memory-efficient processing
	currentMemory := int64(len(component.Name) + len(component.FilePath))
	m.memoryUsage = append(m.memoryUsage, currentMemory)

	// Keep only last 10 measurements to avoid unbounded growth
	if len(m.memoryUsage) > 10 {
		m.memoryUsage = m.memoryUsage[1:]
	}

	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["memory_efficient"] = true
	return component, nil
}

func (m *MockMemoryEfficientPlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockMemoryEfficientPlugin) Priority() int                 { return 1 }

type MockIntermittentFailurePlugin struct {
	MockPlugin
	failureRate float64
	callCount   int
	mutex       sync.Mutex
}

func (m *MockIntermittentFailurePlugin) Initialize(ctx context.Context, config PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *MockIntermittentFailurePlugin) HandleComponent(
	ctx context.Context,
	component *types.ComponentInfo,
) (*types.ComponentInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCount++

	// Fail based on call count and failure rate
	if float64(m.callCount%10)/10.0 < m.failureRate {
		return nil, fmt.Errorf("intermittent plugin failure for component %s", component.Name)
	}

	if component.Metadata == nil {
		component.Metadata = make(map[string]interface{})
	}
	component.Metadata["intermittent_processed"] = true
	return component, nil
}

func (m *MockIntermittentFailurePlugin) SupportedExtensions() []string { return []string{".templ"} }
func (m *MockIntermittentFailurePlugin) Priority() int                 { return 1 }
