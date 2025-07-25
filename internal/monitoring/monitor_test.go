package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	t.Run("creates monitor with all components", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultMonitorConfig()
		config.MetricsOutputPath = tmpDir + "/metrics.json"
		config.LogOutputPath = tmpDir + "/monitor.log"

		logger := logging.NewLogger(logging.DefaultConfig())
		monitor, err := NewMonitor(config, logger)

		require.NoError(t, err)
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.metrics)
		assert.NotNil(t, monitor.health)
		assert.NotNil(t, monitor.appMetrics)
		assert.NotNil(t, monitor.logger)
	})

	t.Run("creates monitor with disabled components", func(t *testing.T) {
		config := DefaultMonitorConfig()
		config.MetricsEnabled = false
		config.HealthEnabled = false
		config.HTTPEnabled = false

		logger := logging.NewLogger(logging.DefaultConfig())
		monitor, err := NewMonitor(config, logger)

		require.NoError(t, err)
		assert.NotNil(t, monitor)
		assert.Nil(t, monitor.metrics)
		assert.Nil(t, monitor.health)
		assert.Nil(t, monitor.appMetrics)
	})
}

func TestMonitorStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultMonitorConfig()
	config.MetricsOutputPath = tmpDir + "/metrics.json"
	config.HTTPEnabled = false // Disable HTTP for simpler testing
	config.AlertingEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	t.Run("start and stop monitor", func(t *testing.T) {
		err := monitor.Start()
		assert.NoError(t, err)
		assert.True(t, monitor.started)

		err = monitor.Stop()
		assert.NoError(t, err)
		assert.False(t, monitor.started)
	})

	t.Run("cannot start already started monitor", func(t *testing.T) {
		err := monitor.Start()
		require.NoError(t, err)

		err = monitor.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		err = monitor.Stop()
		assert.NoError(t, err)
	})
}

func TestMonitorHTTPEndpoints(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = true
	config.HTTPPort = 0 // Use random port for testing

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	// Create test server
	monitor.setupHTTPServer()
	server := httptest.NewServer(monitor.httpServer.Handler)
	defer server.Close()

	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var health HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.NotEmpty(t, health.Status)
		assert.NotZero(t, health.Timestamp)
	})

	t.Run("liveness endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("readiness endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("metrics endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "timestamp")
		assert.Contains(t, response, "metrics")
	})

	t.Run("info endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/info")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var info map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)
		assert.Contains(t, info, "application")
		assert.Contains(t, info, "uptime")
		assert.Contains(t, info, "system_info")
	})
}

func TestMonitorOperationTracking(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultMonitorConfig()
	config.MetricsOutputPath = tmpDir + "/metrics.json"
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	t.Run("log successful operation", func(t *testing.T) {
		executed := false
		err := monitor.LogOperation("test_operation", func() error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("log failed operation", func(t *testing.T) {
		testErr := assert.AnError
		err := monitor.LogOperation("test_operation", func() error {
			return testErr
		})

		assert.Equal(t, testErr, err)
	})

	t.Run("track HTTP request", func(t *testing.T) {
		monitor.TrackHTTPRequest("GET", "/test", 200)
		// Verify metric was recorded (would require access to internal metrics)
	})

	t.Run("track WebSocket event", func(t *testing.T) {
		monitor.TrackWebSocketEvent("opened")
		// Verify metric was recorded
	})

	t.Run("track component operation", func(t *testing.T) {
		monitor.TrackComponentOperation("build", "TestComponent", true)
		monitor.TrackComponentOperation("scan", "TestComponent", true)
		// Verify metrics were recorded
	})
}

func TestMonitorMetricsFlush(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultMonitorConfig()
	config.MetricsOutputPath = tmpDir + "/metrics.json"
	config.MetricsInterval = 100 * time.Millisecond

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	err = monitor.Start()
	require.NoError(t, err)

	// Generate some metrics
	monitor.TrackHTTPRequest("GET", "/test", 200)
	monitor.TrackComponentOperation("build", "TestComponent", true)

	// Wait for metrics to be flushed
	time.Sleep(200 * time.Millisecond)

	err = monitor.Stop()
	require.NoError(t, err)

	// Check if metrics file was created
	_, err = os.Stat(config.MetricsOutputPath)
	assert.NoError(t, err, "Metrics file should be created")

	// Read and verify metrics file content
	data, err := os.ReadFile(config.MetricsOutputPath)
	require.NoError(t, err)

	var metricsData map[string]interface{}
	err = json.Unmarshal(data, &metricsData)
	require.NoError(t, err)

	assert.Contains(t, metricsData, "timestamp")
	assert.Contains(t, metricsData, "metrics")
	assert.Contains(t, metricsData, "system")
}

func TestMonitorHealthChecks(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HealthCheckInterval = 100 * time.Millisecond
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	// Register a custom health check
	customCheck := NewHealthCheckFunc("custom_check", false, func(ctx context.Context) HealthCheck {
		return HealthCheck{
			Name:        "custom_check",
			Status:      HealthStatusHealthy,
			Message:     "Custom check passed",
			LastChecked: time.Now(),
			Critical:    false,
		}
	})

	monitor.RegisterHealthCheck(customCheck)

	err = monitor.Start()
	require.NoError(t, err)

	// Wait for health checks to run
	time.Sleep(200 * time.Millisecond)

	health := monitor.GetHealth()
	assert.NotNil(t, health)

	healthResponse := health.GetHealth()
	assert.NotEmpty(t, healthResponse.Status)
	assert.Contains(t, healthResponse.Checks, "custom_check")
	assert.Contains(t, healthResponse.Checks, "filesystem")
	assert.Contains(t, healthResponse.Checks, "memory")
	assert.Contains(t, healthResponse.Checks, "goroutines")

	err = monitor.Stop()
	require.NoError(t, err)
}

func TestGlobalMonitor(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	t.Run("set and get global monitor", func(t *testing.T) {
		SetGlobalMonitor(monitor)
		retrieved := GetGlobalMonitor()
		assert.Equal(t, monitor, retrieved)
	})

	t.Run("nil global monitor", func(t *testing.T) {
		SetGlobalMonitor(nil)
		retrieved := GetGlobalMonitor()
		assert.Nil(t, retrieved)
	})
}

func TestDefaultMonitorConfig(t *testing.T) {
	config := DefaultMonitorConfig()

	assert.True(t, config.MetricsEnabled)
	assert.True(t, config.HealthEnabled)
	assert.True(t, config.HTTPEnabled)
	assert.Equal(t, "templar", config.MetricsPrefix)
	assert.Equal(t, 30*time.Second, config.MetricsInterval)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
	assert.Equal(t, 10*time.Second, config.HealthCheckTimeout)
	assert.Equal(t, "localhost", config.HTTPAddr)
	assert.Equal(t, 8081, config.HTTPPort)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "json", config.LogFormat)
	assert.False(t, config.AlertingEnabled)
}

func TestMonitorAlerting(t *testing.T) {
	t.Skip("Alerting tests require more complex setup")

	config := DefaultMonitorConfig()
	config.AlertingEnabled = true
	config.AlertThresholds.UnhealthyComponents = 1
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	// Register an unhealthy check
	unhealthyCheck := NewHealthCheckFunc("failing_check", true, func(ctx context.Context) HealthCheck {
		return HealthCheck{
			Name:        "failing_check",
			Status:      HealthStatusUnhealthy,
			Message:     "This check always fails",
			LastChecked: time.Now(),
			Critical:    true,
		}
	})

	monitor.RegisterHealthCheck(unhealthyCheck)

	err = monitor.Start()
	require.NoError(t, err)

	// Wait for alerts to trigger
	time.Sleep(2 * time.Minute)

	err = monitor.Stop()
	require.NoError(t, err)

	// This test would require capturing log output to verify alerts were triggered
}
