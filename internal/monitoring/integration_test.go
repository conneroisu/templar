package monitoring

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitoringMiddleware(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	middleware := MonitoringMiddleware(monitor)

	t.Run("successful request", func(t *testing.T) {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "OK", recorder.Body.String())

		// Verify metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "templar_http_requests_total" {
				if metric.Labels["method"] == "GET" && metric.Labels["status"] == "200" {
					assert.Equal(t, 1.0, metric.Value)
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Should record HTTP request metric")
	})

	t.Run("error request", func(t *testing.T) {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error"))
		}))

		req := httptest.NewRequest("POST", "/error", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		// Verify error metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "templar_http_requests_total" {
				if metric.Labels["method"] == "POST" && metric.Labels["status"] == "500" {
					assert.Equal(t, 1.0, metric.Value)
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Should record HTTP error metric")
	})

	t.Run("request duration tracking", func(t *testing.T) {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/slow", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		// Verify duration metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "templar_http_request_duration_seconds_count" {
				assert.Equal(t, 1.0, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should record request duration metric")
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		wrapper := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}

		wrapper.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, wrapper.statusCode)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("default status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		wrapper := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}

		wrapper.Write([]byte("test"))
		assert.Equal(t, http.StatusOK, wrapper.statusCode)
	})
}

func TestComponentHealthChecker(t *testing.T) {
	t.Run("healthy component", func(t *testing.T) {
		checker := ComponentHealthChecker("test_component", func() error {
			return nil
		})

		assert.Equal(t, "component_test_component", checker.Name())
		assert.False(t, checker.IsCritical())

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Contains(t, result.Message, "functioning correctly")
		assert.Equal(t, "test_component", result.Metadata["component"])
	})

	t.Run("unhealthy component", func(t *testing.T) {
		testErr := errors.New("component failure")
		checker := ComponentHealthChecker("failing_component", func() error {
			return testErr
		})

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "Component check failed")
		assert.Equal(t, "failing_component", result.Metadata["component"])
		assert.Equal(t, testErr.Error(), result.Metadata["error"])
	})
}

func TestBuildPipelineHealthChecker(t *testing.T) {
	t.Run("healthy build pipeline", func(t *testing.T) {
		checker := BuildPipelineHealthChecker(func() error {
			return nil
		})

		assert.Equal(t, "build_pipeline", checker.Name())
		assert.True(t, checker.IsCritical())

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Contains(t, result.Message, "operational")
		assert.True(t, result.Critical)
	})

	t.Run("failed build pipeline", func(t *testing.T) {
		testErr := errors.New("build failed")
		checker := BuildPipelineHealthChecker(func() error {
			return testErr
		})

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "Build pipeline failed")
		assert.True(t, result.Critical)
		assert.Equal(t, testErr.Error(), result.Metadata["error"])
	})
}

func TestFileWatcherHealthChecker(t *testing.T) {
	t.Run("active file watcher", func(t *testing.T) {
		checker := FileWatcherHealthChecker(func() bool {
			return true
		})

		assert.Equal(t, "file_watcher", checker.Name())
		assert.True(t, checker.IsCritical())

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Contains(t, result.Message, "active and monitoring")
	})

	t.Run("inactive file watcher", func(t *testing.T) {
		checker := FileWatcherHealthChecker(func() bool {
			return false
		})

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "not active")
	})
}

func TestWebSocketHealthChecker(t *testing.T) {
	t.Run("normal connection count", func(t *testing.T) {
		checker := WebSocketHealthChecker(func() int {
			return 10
		})

		assert.Equal(t, "websocket", checker.Name())
		assert.False(t, checker.IsCritical())

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Contains(t, result.Message, "10 connections")
		assert.Equal(t, 10, result.Metadata["connection_count"])
	})

	t.Run("high connection count", func(t *testing.T) {
		checker := WebSocketHealthChecker(func() int {
			return 150
		})

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusDegraded, result.Status)
		assert.Contains(t, result.Message, "High number")
		assert.Equal(t, 150, result.Metadata["connection_count"])
	})
}

func TestLoggingIntegration(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	integration := NewLoggingIntegration(monitor, logger)

	t.Run("log with metrics - error", func(t *testing.T) {
		testErr := errors.New("test error")
		integration.LogWithMetrics(context.Background(), logging.LevelError, "test_component", "test_operation", testErr, "Test error message", "key", "value")

		// Verify error metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		errorFound := false
		logFound := false

		for _, metric := range metrics {
			if metric.Name == "templar_errors_total" && metric.Labels["category"] == "test_component" {
				errorFound = true
			}
			if metric.Name == "templar_log_entries_total" && metric.Labels["level"] == "ERROR" {
				logFound = true
			}
		}

		assert.True(t, errorFound, "Should record error metric")
		assert.True(t, logFound, "Should record log entry metric")
	})

	t.Run("log with metrics - info", func(t *testing.T) {
		integration.LogWithMetrics(context.Background(), logging.LevelInfo, "test_component", "test_operation", nil, "Test info message", "key", "value")

		// Verify log metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false

		for _, metric := range metrics {
			if metric.Name == "templar_log_entries_total" && metric.Labels["level"] == "INFO" {
				found = true
				break
			}
		}

		assert.True(t, found, "Should record log entry metric")
	})
}

func TestOperationTracker(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	tracker := NewOperationTracker(monitor, logger, "test_component")

	t.Run("successful operation", func(t *testing.T) {
		executed := false
		err := tracker.TrackOperation(context.Background(), "test_operation", func(ctx context.Context) error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)

		// Verify metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "templar_test_component_test_operation_duration_seconds_count" {
				assert.Equal(t, 1.0, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should record operation duration metric")
	})

	t.Run("failed operation", func(t *testing.T) {
		testErr := errors.New("operation failed")
		err := tracker.TrackOperation(context.Background(), "failing_operation", func(ctx context.Context) error {
			return testErr
		})

		assert.Equal(t, testErr, err)

		// Verify error metrics were recorded
		metrics := monitor.metrics.GatherMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "templar_errors_total" && metric.Labels["category"] == "test_component" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should record error metric")
	})
}

func TestBatchTracker(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	t.Run("successful batch processing", func(t *testing.T) {
		tracker := NewBatchTracker(monitor, logger, "test_component", 5)

		// Process items
		for i := 0; i < 5; i++ {
			err := tracker.TrackItem(context.Background(), fmt.Sprintf("item_%d", i), func() error {
				return nil
			})
			assert.NoError(t, err)
		}

		tracker.Complete(context.Background())

		assert.Equal(t, 5, tracker.processedCount)
		assert.Equal(t, 0, tracker.errorCount)
	})

	t.Run("batch processing with errors", func(t *testing.T) {
		tracker := NewBatchTracker(monitor, logger, "test_component", 3)

		// Process items with some errors
		err1 := tracker.TrackItem(context.Background(), "item_1", func() error {
			return nil
		})
		assert.NoError(t, err1)

		err2 := tracker.TrackItem(context.Background(), "item_2", func() error {
			return errors.New("processing failed")
		})
		assert.Error(t, err2)

		err3 := tracker.TrackItem(context.Background(), "item_3", func() error {
			return nil
		})
		assert.NoError(t, err3)

		tracker.Complete(context.Background())

		assert.Equal(t, 3, tracker.processedCount)
		assert.Equal(t, 1, tracker.errorCount)
	})
}

func TestSetupMonitoring(t *testing.T) {
	t.Run("default setup", func(t *testing.T) {
		config := MonitoringConfig{
			EnableHTTPMiddleware: true,
			EnableHealthChecks:   true,
			EnableMetrics:        true,
			LogLevel:             "info",
		}

		monitor, err := SetupMonitoring(config)
		require.NoError(t, err)
		assert.NotNil(t, monitor)

		// Verify global monitor is set
		globalMonitor := GetGlobalMonitor()
		assert.Equal(t, monitor, globalMonitor)

		// Clean up
		SetGlobalMonitor(nil)
	})

	t.Run("disabled features", func(t *testing.T) {
		config := MonitoringConfig{
			EnableHTTPMiddleware: false,
			EnableHealthChecks:   false,
			EnableMetrics:        false,
			LogLevel:             "error",
		}

		monitor, err := SetupMonitoring(config)
		require.NoError(t, err)
		assert.NotNil(t, monitor)

		// Clean up
		SetGlobalMonitor(nil)
	})
}

func TestGlobalFunctions(t *testing.T) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(t, err)

	SetGlobalMonitor(monitor)
	defer SetGlobalMonitor(nil)

	t.Run("get middleware", func(t *testing.T) {
		middleware := GetMiddleware()
		assert.NotNil(t, middleware)

		// Test with nil global monitor
		SetGlobalMonitor(nil)
		middleware = GetMiddleware()
		assert.NotNil(t, middleware) // Should return passthrough

		SetGlobalMonitor(monitor)
	})

	t.Run("track operation globally", func(t *testing.T) {
		executed := false
		err := TrackOperation(context.Background(), "global_component", "global_operation", func(ctx context.Context) error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("log error globally", func(t *testing.T) {
		testErr := errors.New("global error")
		LogError(context.Background(), "global_component", "global_operation", testErr, "Test error message", "key", "value")

		// Function should not panic
	})

	t.Run("log info globally", func(t *testing.T) {
		LogInfo(context.Background(), "global_component", "global_operation", "Test info message", "key", "value")

		// Function should not panic
	})

	t.Run("functions with nil global monitor", func(t *testing.T) {
		SetGlobalMonitor(nil)

		// These should not panic
		err := TrackOperation(context.Background(), "component", "operation", func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)

		LogError(context.Background(), "component", "operation", errors.New("test"), "message")
		LogInfo(context.Background(), "component", "operation", "message")

		SetGlobalMonitor(monitor)
	})
}

func BenchmarkMonitoringMiddleware(b *testing.B) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(b, err)

	middleware := MonitoringMiddleware(monitor)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}

func BenchmarkOperationTracking(b *testing.B) {
	config := DefaultMonitorConfig()
	config.HTTPEnabled = false

	logger := logging.NewLogger(logging.DefaultConfig())
	monitor, err := NewMonitor(config, logger)
	require.NoError(b, err)

	tracker := NewOperationTracker(monitor, logger, "benchmark_component")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tracker.TrackOperation(context.Background(), "benchmark_operation", func(ctx context.Context) error {
			return nil
		})
		require.NoError(b, err)
	}
}
