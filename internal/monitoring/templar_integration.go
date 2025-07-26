package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TemplarMonitor provides Templar-specific monitoring integration
type TemplarMonitor struct {
	*Monitor
	scannerTracker  *OperationTracker
	buildTracker    *OperationTracker
	serverTracker   *OperationTracker
	watcherTracker  *OperationTracker
	rendererTracker *OperationTracker
	registryTracker *OperationTracker
}

// NewTemplarMonitor creates a Templar-specific monitor with all integrations
func NewTemplarMonitor(configPath string) (*TemplarMonitor, error) {
	// Load configuration from file if provided
	config := DefaultMonitorConfig()
	if configPath != "" {
		// In a real implementation, you would load from YAML/JSON file
		config.LogOutputPath = "./logs/templar-monitor.log"
		config.MetricsOutputPath = "./logs/templar-metrics.json"
	}

	// Create base monitor
	monitor, err := NewMonitor(config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	logger := monitor.GetLogger()

	// Create component trackers
	templatorMonitor := &TemplarMonitor{
		Monitor:         monitor,
		scannerTracker:  NewOperationTracker(monitor, logger, "scanner"),
		buildTracker:    NewOperationTracker(monitor, logger, "build"),
		serverTracker:   NewOperationTracker(monitor, logger, "server"),
		watcherTracker:  NewOperationTracker(monitor, logger, "watcher"),
		rendererTracker: NewOperationTracker(monitor, logger, "renderer"),
		registryTracker: NewOperationTracker(monitor, logger, "registry"),
	}

	// Register Templar-specific health checks
	templatorMonitor.registerTemplarHealthChecks()

	return templatorMonitor, nil
}

// registerTemplarHealthChecks registers health checks specific to Templar components
func (tm *TemplarMonitor) registerTemplarHealthChecks() {
	// Component registry health check
	registryCheck := NewHealthCheckFunc(
		"component_registry",
		true,
		func(ctx context.Context) HealthCheck {
			start := time.Now()

			// Check if registry is accessible (mock implementation)
			registryPath := "./components"
			if _, err := os.Stat(registryPath); err != nil {
				return HealthCheck{
					Name:        "component_registry",
					Status:      HealthStatusUnhealthy,
					Message:     fmt.Sprintf("Component registry not accessible: %v", err),
					LastChecked: time.Now(),
					Duration:    time.Since(start),
					Critical:    true,
					Metadata: map[string]interface{}{
						"registry_path": registryPath,
						"error":         err.Error(),
					},
				}
			}

			return HealthCheck{
				Name:        "component_registry",
				Status:      HealthStatusHealthy,
				Message:     "Component registry is accessible",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
				Metadata: map[string]interface{}{
					"registry_path": registryPath,
				},
			}
		},
	)
	tm.RegisterHealthCheck(registryCheck)

	// Templ binary check
	templCheck := NewHealthCheckFunc("templ_binary", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		// Check if templ binary is available
		if _, err := os.Stat("templ"); err != nil {
			// Try to find in PATH
			if _, err := exec.LookPath("templ"); err != nil {
				return HealthCheck{
					Name:        "templ_binary",
					Status:      HealthStatusUnhealthy,
					Message:     "Templ binary not found in PATH or current directory",
					LastChecked: time.Now(),
					Duration:    time.Since(start),
					Critical:    true,
					Metadata: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}
		}

		return HealthCheck{
			Name:        "templ_binary",
			Status:      HealthStatusHealthy,
			Message:     "Templ binary is available",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
		}
	})
	tm.RegisterHealthCheck(templCheck)

	// Cache directory health check
	cacheCheck := NewHealthCheckFunc(
		"cache_directory",
		false,
		func(ctx context.Context) HealthCheck {
			start := time.Now()
			cacheDir := ".templar/cache"

			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return HealthCheck{
					Name:        "cache_directory",
					Status:      HealthStatusDegraded,
					Message:     fmt.Sprintf("Cannot create cache directory: %v", err),
					LastChecked: time.Now(),
					Duration:    time.Since(start),
					Critical:    false,
					Metadata: map[string]interface{}{
						"cache_dir": cacheDir,
						"error":     err.Error(),
					},
				}
			}

			return HealthCheck{
				Name:        "cache_directory",
				Status:      HealthStatusHealthy,
				Message:     "Cache directory is accessible",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    false,
				Metadata: map[string]interface{}{
					"cache_dir": cacheDir,
				},
			}
		},
	)
	tm.RegisterHealthCheck(cacheCheck)
}

// Component operation tracking methods

// TrackScanOperation tracks component scanning operations
func (tm *TemplarMonitor) TrackScanOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.scannerTracker.TrackOperation(ctx, operation, fn)
}

// TrackBuildOperation tracks build operations
func (tm *TemplarMonitor) TrackBuildOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.buildTracker.TrackOperation(ctx, operation, fn)
}

// TrackServerOperation tracks server operations
func (tm *TemplarMonitor) TrackServerOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.serverTracker.TrackOperation(ctx, operation, fn)
}

// TrackWatcherOperation tracks file watcher operations
func (tm *TemplarMonitor) TrackWatcherOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.watcherTracker.TrackOperation(ctx, operation, fn)
}

// TrackRendererOperation tracks renderer operations
func (tm *TemplarMonitor) TrackRendererOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.rendererTracker.TrackOperation(ctx, operation, fn)
}

// TrackRegistryOperation tracks registry operations
func (tm *TemplarMonitor) TrackRegistryOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	return tm.registryTracker.TrackOperation(ctx, operation, fn)
}

// Component-specific metrics

// RecordComponentScanned records a component scan event
func (tm *TemplarMonitor) RecordComponentScanned(componentType, componentName string) {
	if tm.appMetrics != nil {
		tm.appMetrics.ComponentScanned(componentType)
		tm.metrics.Counter("components_discovered_total", map[string]string{
			"type": componentType,
			"name": componentName,
		})
	}
}

// RecordComponentBuilt records a component build event
func (tm *TemplarMonitor) RecordComponentBuilt(
	componentName string,
	success bool,
	duration time.Duration,
) {
	if tm.appMetrics != nil {
		tm.appMetrics.ComponentBuilt(componentName, success)
		tm.appMetrics.BuildDuration(componentName, duration)

		status := "success"
		if !success {
			status = "failure"
		}

		tm.metrics.Histogram(
			"component_build_duration_seconds",
			duration.Seconds(),
			map[string]string{
				"component": componentName,
				"status":    status,
			},
		)
	}
}

// RecordFileWatchEvent records a file watch event
func (tm *TemplarMonitor) RecordFileWatchEvent(eventType, filePath string) {
	if tm.appMetrics != nil {
		tm.appMetrics.FileWatcherEvent(eventType)

		ext := filepath.Ext(filePath)
		tm.metrics.Counter("file_watch_events_total", map[string]string{
			"event_type": eventType,
			"file_ext":   ext,
		})
	}
}

// RecordWebSocketEvent records WebSocket events
func (tm *TemplarMonitor) RecordWebSocketEvent(eventType string, clientCount int) {
	if tm.appMetrics != nil {
		tm.appMetrics.WebSocketConnection(eventType)
		tm.metrics.Gauge("websocket_active_connections", float64(clientCount), nil)
	}
}

// RecordCacheEvent records cache hit/miss events
func (tm *TemplarMonitor) RecordCacheEvent(operation string, hit bool, itemKey string) {
	if tm.appMetrics != nil {
		tm.appMetrics.CacheOperation(operation, hit)

		result := "miss"
		if hit {
			result = "hit"
		}

		tm.metrics.Counter("cache_events_total", map[string]string{
			"operation": operation,
			"result":    result,
		})
	}
}

// HTTP Middleware factory for Templar server

// CreateTemplarMiddleware creates HTTP middleware with Templar-specific tracking
func (tm *TemplarMonitor) CreateTemplarMiddleware() func(http.Handler) http.Handler {
	baseMiddleware := MonitoringMiddleware(tm.Monitor)

	return func(next http.Handler) http.Handler {
		return baseMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Add request-specific tracking
			if tm.appMetrics != nil {
				// Track specific endpoint patterns
				switch r.URL.Path {
				case "/api/components":
					tm.metrics.Counter("api_component_requests_total", map[string]string{
						"method": r.Method,
					})
				case "/api/build":
					tm.metrics.Counter("api_build_requests_total", map[string]string{
						"method": r.Method,
					})
				case "/ws":
					tm.metrics.Counter("websocket_connection_attempts_total", nil)
				case "/preview":
					tm.metrics.Counter("preview_requests_total", map[string]string{
						"component": r.URL.Query().Get("component"),
					})
				}
			}

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		}))
	}
}

// Configuration and setup helpers

// SetupTemplarMonitoring sets up monitoring for a Templar application
func SetupTemplarMonitoring(configPath string) (*TemplarMonitor, error) {
	monitor, err := NewTemplarMonitor(configPath)
	if err != nil {
		return nil, err
	}

	// Start monitoring
	if err := monitor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Set as global monitor
	SetGlobalMonitor(monitor.Monitor)

	return monitor, nil
}

// TemplarConfig represents Templar-specific monitoring configuration
type TemplarConfig struct {
	MonitorConfig    `yaml:",inline"`
	ComponentPaths   []string `yaml:"component_paths" json:"component_paths"`
	CacheDirectory   string   `yaml:"cache_directory" json:"cache_directory"`
	BuildCommand     string   `yaml:"build_command" json:"build_command"`
	WatchPatterns    []string `yaml:"watch_patterns" json:"watch_patterns"`
	PreviewPort      int      `yaml:"preview_port" json:"preview_port"`
	WebSocketEnabled bool     `yaml:"websocket_enabled" json:"websocket_enabled"`
}

// DefaultTemplarConfig returns default Templar monitoring configuration
func DefaultTemplarConfig() TemplarConfig {
	return TemplarConfig{
		MonitorConfig: DefaultMonitorConfig(),
		ComponentPaths: []string{
			"./components",
			"./views",
			"./layouts",
		},
		CacheDirectory:   ".templar/cache",
		BuildCommand:     "templ generate",
		WatchPatterns:    []string{"**/*.templ", "**/*.go"},
		PreviewPort:      8080,
		WebSocketEnabled: true,
	}
}

// Startup and shutdown helpers

// GracefulShutdown handles graceful shutdown of Templar monitoring
func (tm *TemplarMonitor) GracefulShutdown(ctx context.Context) error {
	tm.GetLogger().Info(ctx, "Starting graceful shutdown of monitoring system")

	// Flush final metrics
	if tm.metrics != nil {
		if err := tm.metrics.FlushMetrics(); err != nil {
			tm.GetLogger().Error(ctx, err, "Failed to flush final metrics")
		}
	}

	// Stop monitoring
	if err := tm.Stop(); err != nil {
		return fmt.Errorf("failed to stop monitoring: %w", err)
	}

	tm.GetLogger().Info(ctx, "Monitoring system shutdown complete")
	return nil
}

// Utility functions for common monitoring patterns

// MonitorComponentOperation is a convenience function for monitoring component operations
func MonitorComponentOperation(
	ctx context.Context,
	component, operation string,
	fn func() error,
) error {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return fn()
	}

	// Simple operation tracking
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Log the operation
	if err != nil {
		monitor.logger.Error(ctx, err, "Component operation failed",
			"component", component,
			"operation", operation,
			"duration", duration)
	} else {
		monitor.logger.Info(ctx, "Component operation completed",
			"component", component,
			"operation", operation,
			"duration", duration)
	}

	return err
}

// LogComponentError logs component-specific errors with proper categorization
func LogComponentError(
	ctx context.Context,
	component, operation string,
	err error,
	details map[string]interface{},
) {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return
	}

	// Convert details to field pairs
	fields := make([]interface{}, 0, len(details)*2)
	for k, v := range details {
		fields = append(fields, k, v)
	}

	LogError(ctx, component, operation, err, err.Error(), fields...)

	// Track error in application metrics
	// TODO: Fix type assertion issues - simplified for now
}
