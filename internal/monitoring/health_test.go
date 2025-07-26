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

func TestHealthCheckFunc(t *testing.T) {
	t.Run("create health check function", func(t *testing.T) {
		checkFn := NewHealthCheckFunc("test_check", true, func(ctx context.Context) HealthCheck {
			return HealthCheck{
				Name:        "test_check",
				Status:      HealthStatusHealthy,
				Message:     "All good",
				LastChecked: time.Now(),
				Critical:    true,
			}
		})

		assert.Equal(t, "test_check", checkFn.Name())
		assert.True(t, checkFn.IsCritical())

		result := checkFn.Check(context.Background())
		assert.Equal(t, "test_check", result.Name)
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Equal(t, "All good", result.Message)
		assert.True(t, result.Critical)
	})

	t.Run("health check with context timeout", func(t *testing.T) {
		checkFn := NewHealthCheckFunc("slow_check", false, func(ctx context.Context) HealthCheck {
			select {
			case <-time.After(100 * time.Millisecond):
				return HealthCheck{
					Name:   "slow_check",
					Status: HealthStatusHealthy,
				}
			case <-ctx.Done():
				return HealthCheck{
					Name:    "slow_check",
					Status:  HealthStatusUnhealthy,
					Message: "Timeout",
				}
			}
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		result := checkFn.Check(ctx)
		assert.Equal(t, HealthStatusUnhealthy, result.Status)
		assert.Equal(t, "Timeout", result.Message)
	})
}

func TestHealthMonitor(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())

	t.Run("create health monitor", func(t *testing.T) {
		monitor := NewHealthMonitor(logger)
		assert.NotNil(t, monitor)
		assert.Equal(t, 30*time.Second, monitor.interval)
		assert.Equal(t, 10*time.Second, monitor.timeout)
	})

	t.Run("register and unregister health checks", func(t *testing.T) {
		monitor := NewHealthMonitor(logger)

		checkFn := NewHealthCheckFunc("test_check", false, func(ctx context.Context) HealthCheck {
			return HealthCheck{
				Name:   "test_check",
				Status: HealthStatusHealthy,
			}
		})

		monitor.RegisterCheck(checkFn)
		assert.Contains(t, monitor.checks, "test_check")

		monitor.UnregisterCheck("test_check")
		assert.NotContains(t, monitor.checks, "test_check")
	})

	t.Run("run health checks manually", func(t *testing.T) {
		monitor := NewHealthMonitor(logger)

		// Register multiple checks
		healthyCheck := NewHealthCheckFunc(
			"healthy_check",
			false,
			func(ctx context.Context) HealthCheck {
				return HealthCheck{
					Name:        "healthy_check",
					Status:      HealthStatusHealthy,
					Message:     "All good",
					LastChecked: time.Now(),
				}
			},
		)

		unhealthyCheck := NewHealthCheckFunc(
			"unhealthy_check",
			true,
			func(ctx context.Context) HealthCheck {
				return HealthCheck{
					Name:        "unhealthy_check",
					Status:      HealthStatusUnhealthy,
					Message:     "Something is wrong",
					LastChecked: time.Now(),
					Critical:    true,
				}
			},
		)

		degradedCheck := NewHealthCheckFunc(
			"degraded_check",
			false,
			func(ctx context.Context) HealthCheck {
				return HealthCheck{
					Name:        "degraded_check",
					Status:      HealthStatusDegraded,
					Message:     "Performance degraded",
					LastChecked: time.Now(),
				}
			},
		)

		monitor.RegisterCheck(healthyCheck)
		monitor.RegisterCheck(unhealthyCheck)
		monitor.RegisterCheck(degradedCheck)

		monitor.runHealthChecks()

		health := monitor.GetHealth()
		assert.Equal(t, HealthStatusUnhealthy, health.Status) // Critical check failed
		assert.Equal(t, 3, health.Summary.Total)
		assert.Equal(t, 1, health.Summary.Healthy)
		assert.Equal(t, 1, health.Summary.Unhealthy)
		assert.Equal(t, 1, health.Summary.Degraded)
		assert.Equal(t, 1, health.Summary.Critical)

		assert.Contains(t, health.Checks, "healthy_check")
		assert.Contains(t, health.Checks, "unhealthy_check")
		assert.Contains(t, health.Checks, "degraded_check")
	})

	t.Run("start and stop monitor", func(t *testing.T) {
		monitor := NewHealthMonitor(logger)
		monitor.interval = 50 * time.Millisecond

		healthyCheck := NewHealthCheckFunc(
			"test_check",
			false,
			func(ctx context.Context) HealthCheck {
				return HealthCheck{
					Name:   "test_check",
					Status: HealthStatusHealthy,
				}
			},
		)

		monitor.RegisterCheck(healthyCheck)
		monitor.Start()

		// Wait for at least one check cycle
		time.Sleep(100 * time.Millisecond)

		health := monitor.GetHealth()
		assert.Contains(t, health.Checks, "test_check")
		assert.NotZero(t, health.Checks["test_check"].LastChecked)

		monitor.Stop()
	})
}

func TestHealthResponse(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	t.Run("health response structure", func(t *testing.T) {
		health := monitor.GetHealth()

		assert.NotEmpty(t, health.Status)
		assert.NotZero(t, health.Timestamp)
		assert.NotNil(t, health.Uptime)
		assert.NotNil(t, health.Checks)
		assert.NotNil(t, health.Summary)
		assert.NotNil(t, health.SystemInfo)

		// Verify system info
		assert.NotEmpty(t, health.SystemInfo.Platform)
		assert.NotEmpty(t, health.SystemInfo.GoVersion)
		assert.NotZero(t, health.SystemInfo.PID)
		assert.NotZero(t, health.SystemInfo.StartTime)
	})

	t.Run("health status calculation", func(t *testing.T) {
		testCases := []struct {
			name           string
			checks         []HealthCheck
			expectedStatus HealthStatus
		}{
			{
				name: "all healthy",
				checks: []HealthCheck{
					{Name: "check1", Status: HealthStatusHealthy, Critical: false},
					{Name: "check2", Status: HealthStatusHealthy, Critical: true},
				},
				expectedStatus: HealthStatusHealthy,
			},
			{
				name: "critical unhealthy",
				checks: []HealthCheck{
					{Name: "check1", Status: HealthStatusHealthy, Critical: false},
					{Name: "check2", Status: HealthStatusUnhealthy, Critical: true},
				},
				expectedStatus: HealthStatusUnhealthy,
			},
			{
				name: "non-critical unhealthy",
				checks: []HealthCheck{
					{Name: "check1", Status: HealthStatusUnhealthy, Critical: false},
					{Name: "check2", Status: HealthStatusHealthy, Critical: true},
				},
				expectedStatus: HealthStatusDegraded,
			},
			{
				name: "degraded check",
				checks: []HealthCheck{
					{Name: "check1", Status: HealthStatusHealthy, Critical: false},
					{Name: "check2", Status: HealthStatusDegraded, Critical: false},
				},
				expectedStatus: HealthStatusDegraded,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				checksMap := make(map[string]HealthCheck)
				for _, check := range tc.checks {
					checksMap[check.Name] = check
				}

				status := monitor.calculateOverallStatus(checksMap)
				assert.Equal(t, tc.expectedStatus, status)
			})
		}
	})
}

func TestHealthHTTPHandler(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	// Register some checks
	healthyCheck := NewHealthCheckFunc(
		"healthy_check",
		false,
		func(ctx context.Context) HealthCheck {
			return HealthCheck{
				Name:   "healthy_check",
				Status: HealthStatusHealthy,
			}
		},
	)

	unhealthyCheck := NewHealthCheckFunc(
		"unhealthy_check",
		true,
		func(ctx context.Context) HealthCheck {
			return HealthCheck{
				Name:     "unhealthy_check",
				Status:   HealthStatusUnhealthy,
				Critical: true,
			}
		},
	)

	monitor.RegisterCheck(healthyCheck)
	monitor.runHealthChecks()

	t.Run("healthy response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		handler := monitor.HTTPHandler()
		handler(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var response HealthResponse
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, HealthStatusHealthy, response.Status)
	})

	t.Run("unhealthy response", func(t *testing.T) {
		monitor.RegisterCheck(unhealthyCheck)
		monitor.runHealthChecks()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		handler := monitor.HTTPHandler()
		handler(recorder, req)

		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)

		var response HealthResponse
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, HealthStatusUnhealthy, response.Status)
	})
}

func TestPredefinedHealthChecks(t *testing.T) {
	t.Run("filesystem health check - success", func(t *testing.T) {
		tmpDir := t.TempDir()
		checker := FileSystemHealthChecker(tmpDir)

		assert.Equal(t, "filesystem", checker.Name())
		assert.True(t, checker.IsCritical())

		result := checker.Check(context.Background())
		assert.Equal(t, HealthStatusHealthy, result.Status)
		assert.Contains(t, result.Message, "accessible")
	})

	t.Run("filesystem health check - read-only directory", func(t *testing.T) {
		// This test might not work on all systems
		checker := FileSystemHealthChecker("/proc")

		result := checker.Check(context.Background())
		// Should be unhealthy because /proc is typically read-only
		assert.Equal(t, HealthStatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "Cannot write")
	})

	t.Run("memory health check", func(t *testing.T) {
		checker := MemoryHealthChecker()

		assert.Equal(t, "memory", checker.Name())
		assert.True(t, checker.IsCritical())

		result := checker.Check(context.Background())
		// Memory should typically be healthy in tests
		assert.NotEqual(t, HealthStatusUnknown, result.Status)
		assert.Contains(t, result.Metadata, "heap_alloc")
		assert.Contains(t, result.Metadata, "gc_runs")
	})

	t.Run("goroutine health check", func(t *testing.T) {
		checker := GoroutineHealthChecker()

		assert.Equal(t, "goroutines", checker.Name())
		assert.False(t, checker.IsCritical())

		result := checker.Check(context.Background())
		// Goroutine count should typically be healthy in tests
		assert.NotEqual(t, HealthStatusUnknown, result.Status)
		assert.Contains(t, result.Metadata, "count")

		count := result.Metadata["count"].(int)
		assert.Greater(t, count, 0)
	})
}

func TestHealthSummaryCalculation(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	checks := map[string]HealthCheck{
		"healthy1":   {Status: HealthStatusHealthy, Critical: false},
		"healthy2":   {Status: HealthStatusHealthy, Critical: true},
		"unhealthy1": {Status: HealthStatusUnhealthy, Critical: false},
		"unhealthy2": {Status: HealthStatusUnhealthy, Critical: true},
		"degraded1":  {Status: HealthStatusDegraded, Critical: false},
		"unknown1":   {Status: HealthStatusUnknown, Critical: false},
	}

	summary := monitor.calculateSummary(checks)

	assert.Equal(t, 6, summary.Total)
	assert.Equal(t, 2, summary.Healthy)
	assert.Equal(t, 2, summary.Unhealthy)
	assert.Equal(t, 1, summary.Degraded)
	assert.Equal(t, 1, summary.Unknown)
	assert.Equal(t, 2, summary.Critical) // healthy2 and unhealthy2 are both critical
}

func TestHealthMonitorConcurrency(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	// Add multiple checks that take some time
	for i := range 10 {
		checkName := fmt.Sprintf("check_%d", i)
		checker := NewHealthCheckFunc(checkName, false, func(ctx context.Context) HealthCheck {
			time.Sleep(10 * time.Millisecond) // Simulate work

			return HealthCheck{
				Name:   checkName,
				Status: HealthStatusHealthy,
			}
		})
		monitor.RegisterCheck(checker)
	}

	start := time.Now()
	monitor.runHealthChecks()
	duration := time.Since(start)

	// All checks should run concurrently, so total time should be much less than 10 * 10ms
	assert.Less(t, duration, 50*time.Millisecond, "Health checks should run concurrently")

	health := monitor.GetHealth()
	assert.Equal(t, 10, health.Summary.Total)
	assert.Equal(t, 10, health.Summary.Healthy)
}

func TestGetEnvironment(t *testing.T) {
	t.Run("default environment", func(t *testing.T) {
		// Clear the environment variable
		os.Unsetenv("TEMPLAR_ENV")
		env := getEnvironment()
		assert.Equal(t, "development", env)
	})

	t.Run("custom environment", func(t *testing.T) {
		os.Setenv("TEMPLAR_ENV", "production")
		defer os.Unsetenv("TEMPLAR_ENV")

		env := getEnvironment()
		assert.Equal(t, "production", env)
	})
}

func TestGetSystemInfo(t *testing.T) {
	info := getSystemInfo()

	assert.NotEmpty(t, info.Platform)
	assert.Contains(t, info.Platform, "/") // Should contain OS/ARCH
	assert.NotEmpty(t, info.GoVersion)
	assert.Greater(t, info.PID, 0)
	assert.NotZero(t, info.StartTime)
	// Hostname might be empty in some environments, so we don't assert it
}

// Helper functions and benchmarks

func BenchmarkHealthCheck(b *testing.B) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	checker := NewHealthCheckFunc("bench_check", false, func(ctx context.Context) HealthCheck {
		return HealthCheck{
			Name:   "bench_check",
			Status: HealthStatusHealthy,
		}
	})

	monitor.RegisterCheck(checker)

	b.ResetTimer()
	for range b.N {
		monitor.runHealthChecks()
	}
}

func BenchmarkHealthResponseGeneration(b *testing.B) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	// Add several checks
	for i := range 10 {
		checkName := fmt.Sprintf("check_%d", i)
		checker := NewHealthCheckFunc(checkName, false, func(ctx context.Context) HealthCheck {
			return HealthCheck{
				Name:   checkName,
				Status: HealthStatusHealthy,
			}
		})
		monitor.RegisterCheck(checker)
	}

	monitor.runHealthChecks()

	b.ResetTimer()
	for range b.N {
		_ = monitor.GetHealth()
	}
}

func TestHealthCheckDuration(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	monitor := NewHealthMonitor(logger)

	checker := NewHealthCheckFunc("duration_check", false, func(ctx context.Context) HealthCheck {
		time.Sleep(20 * time.Millisecond)

		return HealthCheck{
			Name:   "duration_check",
			Status: HealthStatusHealthy,
		}
	})

	monitor.RegisterCheck(checker)
	monitor.runHealthChecks()

	health := monitor.GetHealth()
	check := health.Checks["duration_check"]

	// Duration should be recorded and be roughly 20ms
	assert.Greater(t, check.Duration, 10*time.Millisecond)
	assert.Less(t, check.Duration, 50*time.Millisecond)
	assert.NotZero(t, check.LastChecked)
}
