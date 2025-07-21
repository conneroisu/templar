package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensiveMonitoringIntegration tests the complete monitoring system
func TestComprehensiveMonitoringIntegration(t *testing.T) {
	// Create temporary directory for test outputs
	tmpDir := t.TempDir()

	// Create comprehensive configuration
	config := &MonitoringConfiguration{
		Enabled: true,
		Logging: LoggingConfig{
			Level:      "debug",
			Format:     "json",
			Output:     "stdout",
			Structured: true,
		},
		Metrics: MetricsConfig{
			Enabled:       true,
			OutputPath:    tmpDir + "/metrics.json",
			FlushInterval: 100 * time.Millisecond,
			Prefix:        "test",
		},
		Health: HealthConfig{
			Enabled:       true,
			CheckInterval: 100 * time.Millisecond,
			CheckTimeout:  5 * time.Second,
		},
		Performance: PerformanceConfig{
			Enabled:        true,
			SampleRate:     1.0,
			UpdateInterval: 100 * time.Millisecond,
		},
		Alerting: AlertingConfig{
			Enabled:  true,
			Cooldown: 1 * time.Second,
		},
		HTTP: HTTPConfig{
			Enabled: true,
			Host:    "localhost",
			Port:    0, // Use random port for testing
		},
	}

	t.Run("full system integration", func(t *testing.T) {
		// Create logger
		logger := logging.NewLogger(logging.DefaultConfig())

		// Create monitor with configuration
		monitor, err := createMonitorFromConfig(config, logger)
		require.NoError(t, err)

		// Start monitoring
		err = monitor.Start()
		require.NoError(t, err)
		defer monitor.Stop()

		// Create and setup alerting
		alertManager := NewAlertManager(logger)
		testChannel := &TestChannel{alerts: make([]Alert, 0)}
		alertManager.AddChannel(testChannel)

		// Add test alert rule
		alertRule := &AlertRule{
			Name:      "test_errors",
			Component: "test",
			Metric:    "test_errors_total",
			Condition: "gt",
			Threshold: 3.0,
			Level:     AlertLevelWarning,
			Message:   "Too many test errors",
			Enabled:   true,
			Cooldown:  500 * time.Millisecond,
		}
		alertManager.AddRule(alertRule)

		// Create performance monitor
		perfMonitor := NewPerformanceMonitor(monitor.metrics, logger)
		perfMonitor.StartBackgroundUpdates(100 * time.Millisecond)

		// Create Templar-specific monitor
		templatorMonitor := &TemplarMonitor{
			Monitor:         monitor,
			scannerTracker:  NewOperationTracker(monitor, logger, "scanner"),
			buildTracker:    NewOperationTracker(monitor, logger, "build"),
			serverTracker:   NewOperationTracker(monitor, logger, "server"),
			watcherTracker:  NewOperationTracker(monitor, logger, "watcher"),
			rendererTracker: NewOperationTracker(monitor, logger, "renderer"),
			registryTracker: NewOperationTracker(monitor, logger, "registry"),
		}

		// Test component operations
		ctx := context.Background()

		// Test scanner operations
		for i := 0; i < 5; i++ {
			err := templatorMonitor.TrackScanOperation(ctx, "scan_components", func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				templatorMonitor.RecordComponentScanned("button", "Button")
				return nil
			})
			assert.NoError(t, err)
		}

		// Test build operations (with some failures)
		for i := 0; i < 7; i++ {
			componentName := "TestComponent"
			start := time.Now()

			err := templatorMonitor.TrackBuildOperation(ctx, "build_component", func(ctx context.Context) error {
				time.Sleep(15 * time.Millisecond)

				// Simulate some failures
				if i >= 5 {
					return assert.AnError
				}
				return nil
			})

			duration := time.Since(start)
			success := err == nil
			templatorMonitor.RecordComponentBuilt(componentName, success, duration)
		}

		// Test performance monitoring
		for i := 0; i < 10; i++ {
			err := perfMonitor.TrackOperation("test_operation", func() error {
				time.Sleep(5 * time.Millisecond)
				return nil
			})
			assert.NoError(t, err)
		}

		// Test file watcher events
		for i := 0; i < 3; i++ {
			templatorMonitor.RecordFileWatchEvent("modified", "./test.templ")
		}

		// Test WebSocket events
		templatorMonitor.RecordWebSocketEvent("client_connected", 5)
		templatorMonitor.RecordWebSocketEvent("client_disconnected", 4)

		// Test cache events
		templatorMonitor.RecordCacheEvent("get", true, "component:Button")
		templatorMonitor.RecordCacheEvent("get", false, "component:Modal")

		// Generate some error metrics to trigger alerts
		for i := 0; i < 5; i++ {
			monitor.metrics.Counter("test_errors_total", nil)
		}

		// Wait for metrics to be collected and processed
		time.Sleep(300 * time.Millisecond)

		// Gather metrics and evaluate alerts
		metrics := monitor.metrics.GatherMetrics()
		alertManager.EvaluateMetrics(ctx, metrics)

		// Verify metrics were collected
		assert.Greater(t, len(metrics), 0, "Should have collected metrics")

		// Check for specific metrics
		foundMetrics := make(map[string]bool)
		for _, metric := range metrics {
			foundMetrics[metric.Name] = true
		}

		assert.True(t, foundMetrics["test_components_scanned_total"], "Should have component scan metrics")
		assert.True(t, foundMetrics["test_components_built_total"], "Should have component build metrics")
		assert.True(t, foundMetrics["test_errors_total"], "Should have error metrics")

		// Verify alerts were triggered
		activeAlerts := alertManager.GetActiveAlerts()
		assert.Greater(t, len(activeAlerts), 0, "Should have triggered alerts")

		// Verify alert was sent to channel
		assert.Greater(t, len(testChannel.alerts), 0, "Should have sent alerts to channel")

		// Test performance snapshot
		snapshot := perfMonitor.GetPerformanceSnapshot()
		assert.NotEmpty(t, snapshot.Operations, "Should have operation metrics")
		assert.Contains(t, snapshot.Operations, "test_operation")

		// Test health checks
		healthMonitor := monitor.GetHealth()
		assert.NotNil(t, healthMonitor, "Should have health monitor")
		healthResponse := healthMonitor.GetHealth()
		assert.Greater(t, healthResponse.Summary.Total, 0, "Should have health checks")

		// Verify metrics file was created
		_, err = os.Stat(config.Metrics.OutputPath)
		assert.NoError(t, err, "Metrics file should be created")
	})

	t.Run("HTTP endpoints integration", func(t *testing.T) {
		// Create monitor with HTTP enabled
		logger := logging.NewLogger(logging.DefaultConfig())
		monitor, err := createMonitorFromConfig(config, logger)
		require.NoError(t, err)

		// Create alert manager
		alertManager := NewAlertManager(logger)

		// Create HTTP server for testing
		mux := http.NewServeMux()
		mux.Handle("/health", monitor.health.HTTPHandler())
		mux.Handle("/metrics", monitor.createMetricsHandler())
		mux.Handle("/info", monitor.createInfoHandler())
		mux.Handle("/alerts/", alertManager.HTTPHandler())

		server := httptest.NewServer(mux)
		defer server.Close()

		// Test health endpoint
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var healthResp HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)
		assert.NotEmpty(t, healthResp.Status)

		// Test metrics endpoint
		resp, err = http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test info endpoint
		resp, err = http.Get(server.URL + "/info")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test alerts endpoint
		resp, err = http.Get(server.URL + "/alerts")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("configuration loading and validation", func(t *testing.T) {
		// Test default configuration
		defaultConfig := DefaultMonitoringConfiguration()
		assert.True(t, defaultConfig.Enabled)
		assert.True(t, defaultConfig.Metrics.Enabled)
		assert.True(t, defaultConfig.Health.Enabled)

		// Test configuration validation
		err := validateConfiguration(defaultConfig)
		assert.NoError(t, err)

		// Test invalid configuration
		invalidConfig := DefaultMonitoringConfiguration()
		invalidConfig.Logging.Level = "invalid"
		err = validateConfiguration(invalidConfig)
		assert.Error(t, err)

		// Test configuration save/load
		configPath := tmpDir + "/test-config.yml"
		err = SaveConfiguration(defaultConfig, configPath)
		assert.NoError(t, err)

		loadedConfig, err := LoadConfiguration(configPath)
		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.Enabled, loadedConfig.Enabled)
	})

	t.Run("error handling and resilience", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())

		// Test with invalid output path
		invalidConfig := *config
		invalidConfig.Metrics.OutputPath = "/invalid/path/metrics.json"

		monitor, err := createMonitorFromConfig(&invalidConfig, logger)
		require.NoError(t, err) // Should not fail creation

		// Should handle start gracefully
		err = monitor.Start()
		// May succeed or fail depending on system, but shouldn't panic
		if err == nil {
			monitor.Stop()
		}

		// Test monitor with disabled components
		disabledConfig := *config
		disabledConfig.Metrics.Enabled = false
		disabledConfig.Health.Enabled = false

		monitor, err = createMonitorFromConfig(&disabledConfig, logger)
		require.NoError(t, err)

		err = monitor.Start()
		assert.NoError(t, err)
		monitor.Stop()

		// Test double start/stop
		monitor, err = createMonitorFromConfig(config, logger)
		require.NoError(t, err)

		err = monitor.Start()
		assert.NoError(t, err)

		err = monitor.Start() // Second start should return error
		assert.Error(t, err)

		err = monitor.Stop()
		assert.NoError(t, err)

		err = monitor.Stop() // Second stop should be safe
		assert.NoError(t, err)
	})

	t.Run("performance and scalability", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())

		// Test with high-throughput operations
		perfConfig := *config
		perfConfig.Performance.SampleRate = 0.1 // Sample only 10% for performance

		monitor, err := createMonitorFromConfig(&perfConfig, logger)
		require.NoError(t, err)

		err = monitor.Start()
		require.NoError(t, err)
		defer monitor.Stop()

		perfMonitor := NewPerformanceMonitor(monitor.metrics, logger)

		// Generate high load
		ctx := context.Background()
		start := time.Now()

		for i := 0; i < 1000; i++ {
			perfMonitor.TrackOperationWithContext(ctx, "high_load_test", func(ctx context.Context) error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
		}

		duration := time.Since(start)

		// Should complete in reasonable time
		assert.Less(t, duration, 5*time.Second, "High load test should complete quickly")

		// Check that some metrics were recorded
		snapshot := perfMonitor.GetPerformanceSnapshot()
		assert.Contains(t, snapshot.Operations, "high_load_test")
	})
}

// createMonitorFromConfig creates a monitor from configuration (helper function)
func createMonitorFromConfig(config *MonitoringConfiguration, logger logging.Logger) (*Monitor, error) {
	monitorConfig := MonitorConfig{
		MetricsEnabled:      config.Metrics.Enabled,
		MetricsOutputPath:   config.Metrics.OutputPath,
		MetricsPrefix:       config.Metrics.Prefix,
		MetricsInterval:     config.Metrics.FlushInterval,
		HealthEnabled:       config.Health.Enabled,
		HealthCheckInterval: config.Health.CheckInterval,
		HealthCheckTimeout:  config.Health.CheckTimeout,
		HTTPEnabled:         config.HTTP.Enabled,
		HTTPAddr:            config.HTTP.Host,
		HTTPPort:            config.HTTP.Port,
		LogLevel:            config.Logging.Level,
		LogFormat:           config.Logging.Format,
		LogOutputPath:       config.Logging.Output,
		StructuredLogging:   config.Logging.Structured,
		AlertingEnabled:     config.Alerting.Enabled,
		AlertCooldown:       config.Alerting.Cooldown,
	}

	return NewMonitor(monitorConfig, logger)
}

// Benchmark tests for performance validation
func BenchmarkComprehensiveMonitoring(b *testing.B) {
	logger := logging.NewLogger(logging.DefaultConfig())

	config := DefaultMonitorConfig()
	config.HTTPEnabled = false // Disable HTTP for benchmarking

	monitor, err := NewMonitor(config, logger)
	if err != nil {
		b.Fatal(err)
	}

	err = monitor.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer monitor.Stop()

	perfMonitor := NewPerformanceMonitor(monitor.metrics, logger)
	ctx := context.Background()

	b.ResetTimer()

	b.Run("operation_tracking", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			perfMonitor.TrackOperationWithContext(ctx, "bench_operation", func(ctx context.Context) error {
				return nil
			})
		}
	})

	b.Run("metrics_recording", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			monitor.metrics.Counter("bench_counter", nil)
			monitor.metrics.Gauge("bench_gauge", float64(i), nil)
			monitor.metrics.Histogram("bench_histogram", float64(i%100), nil)
		}
	})

	b.Run("health_checks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			monitor.health.runHealthChecks()
		}
	})
}

// TestRealWorldScenarios tests realistic usage patterns
func TestRealWorldScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logging.NewLogger(logging.DefaultConfig())

	// Create realistic configuration
	config := DefaultMonitoringConfiguration()
	config.Metrics.OutputPath = tmpDir + "/metrics.json"
	config.Metrics.FlushInterval = 100 * time.Millisecond

	monitor, err := createMonitorFromConfig(config, logger)
	require.NoError(t, err)

	err = monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	templatorMonitor := &TemplarMonitor{
		Monitor:         monitor,
		scannerTracker:  NewOperationTracker(monitor, logger, "scanner"),
		buildTracker:    NewOperationTracker(monitor, logger, "build"),
		serverTracker:   NewOperationTracker(monitor, logger, "server"),
		watcherTracker:  NewOperationTracker(monitor, logger, "watcher"),
		rendererTracker: NewOperationTracker(monitor, logger, "renderer"),
		registryTracker: NewOperationTracker(monitor, logger, "registry"),
	}

	t.Run("component development workflow", func(t *testing.T) {
		ctx := context.Background()

		// 1. Scan for components
		err := templatorMonitor.TrackScanOperation(ctx, "initial_scan", func(ctx context.Context) error {
			components := []string{"Button", "Card", "Modal", "Form", "Layout"}
			for _, comp := range components {
				templatorMonitor.RecordComponentScanned("template", comp)
			}
			return nil
		})
		assert.NoError(t, err)

		// 2. Build components
		components := []string{"Button", "Card", "Modal", "Form", "Layout"}
		for _, comp := range components {
			err := templatorMonitor.TrackBuildOperation(ctx, "build_component", func(ctx context.Context) error {
				// Simulate build time
				buildTime := time.Duration(20+len(comp)*2) * time.Millisecond
				time.Sleep(buildTime)

				// Simulate occasional build failure
				if comp == "Modal" {
					return assert.AnError
				}
				return nil
			})

			success := err == nil
			templatorMonitor.RecordComponentBuilt(comp, success, time.Duration(20+len(comp)*2)*time.Millisecond)
		}

		// 3. Start file watching
		err = templatorMonitor.TrackWatcherOperation(ctx, "start_watching", func(ctx context.Context) error {
			// Simulate file watcher startup
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		assert.NoError(t, err)

		// 4. Simulate file changes
		fileEvents := []string{"created", "modified", "deleted", "moved"}
		for i, event := range fileEvents {
			filePath := fmt.Sprintf("./components/component_%d.templ", i)
			templatorMonitor.RecordFileWatchEvent(event, filePath)
		}

		// 5. Simulate server operations
		for i := 0; i < 10; i++ {
			err := templatorMonitor.TrackServerOperation(ctx, "handle_request", func(ctx context.Context) error {
				// Simulate request processing
				time.Sleep(5 * time.Millisecond)
				return nil
			})
			assert.NoError(t, err)
		}

		// 6. WebSocket interactions
		templatorMonitor.RecordWebSocketEvent("client_connected", 1)
		templatorMonitor.RecordWebSocketEvent("reload_requested", 1)
		templatorMonitor.RecordWebSocketEvent("client_disconnected", 1)

		// Wait for metrics to be processed
		time.Sleep(200 * time.Millisecond)

		// Verify the workflow was properly tracked
		metrics := monitor.metrics.GatherMetrics()
		assert.Greater(t, len(metrics), 0)

		// Check health status
		health := monitor.GetHealth()
		assert.NotNil(t, health)
	})

	t.Run("error scenarios and recovery", func(t *testing.T) {
		ctx := context.Background()

		// Simulate various error conditions
		errorScenarios := []struct {
			component string
			operation string
			errorMsg  string
		}{
			{"scanner", "scan_directory", "permission denied"},
			{"build", "compile_template", "syntax error"},
			{"server", "handle_request", "connection refused"},
			{"watcher", "watch_file", "file not found"},
		}

		for _, scenario := range errorScenarios {
			err := MonitorComponentOperation(ctx, scenario.component, scenario.operation, func() error {
				return fmt.Errorf("%s", scenario.errorMsg)
			})
			assert.Error(t, err) // Should return the error

			// Log the error with monitoring
			LogComponentError(ctx, scenario.component, scenario.operation, err, map[string]interface{}{
				"scenario": "error_recovery_test",
			})
		}

		// Verify error metrics were recorded
		time.Sleep(100 * time.Millisecond)
		metrics := monitor.metrics.GatherMetrics()

		foundErrorMetrics := false
		for _, metric := range metrics {
			if metric.Name == "test_errors_total" {
				foundErrorMetrics = true
				break
			}
		}
		assert.True(t, foundErrorMetrics, "Should have recorded error metrics")
	})
}
