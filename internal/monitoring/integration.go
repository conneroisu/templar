package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// MonitoringMiddleware provides HTTP middleware for request tracking
func MonitoringMiddleware(monitor *Monitor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Track the request
			defer func() {
				duration := time.Since(start)

				if monitor != nil && monitor.appMetrics != nil {
					monitor.appMetrics.ServerRequest(r.Method, r.URL.Path, wrapper.statusCode)

					// Track request duration
					monitor.metrics.Histogram(
						"http_request_duration_seconds",
						duration.Seconds(),
						map[string]string{
							"method": r.Method,
							"path":   r.URL.Path,
							"status": fmt.Sprintf("%d", wrapper.statusCode),
						},
					)
				}

				// Log the request
				if monitor != nil {
					logger := monitor.GetLogger().WithComponent("http_server")
					if wrapper.statusCode >= 400 {
						logger.Error(context.Background(), nil, "HTTP request failed",
							"method", r.Method,
							"path", r.URL.Path,
							"status", wrapper.statusCode,
							"duration", duration,
							"user_agent", r.Header.Get("User-Agent"),
							"remote_addr", r.RemoteAddr)
					} else {
						logger.Info(context.Background(), "HTTP request completed",
							"method", r.Method,
							"path", r.URL.Path,
							"status", wrapper.statusCode,
							"duration", duration)
					}
				}
			}()

			next.ServeHTTP(wrapper, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ComponentHealthChecker creates a health check for component operations
func ComponentHealthChecker(componentName string, checkFn func() error) HealthChecker {
	return NewHealthCheckFunc(
		fmt.Sprintf("component_%s", componentName),
		false,
		func(ctx context.Context) HealthCheck {
			start := time.Now()

			if err := checkFn(); err != nil {
				return HealthCheck{
					Name:        fmt.Sprintf("component_%s", componentName),
					Status:      HealthStatusUnhealthy,
					Message:     fmt.Sprintf("Component check failed: %v", err),
					LastChecked: time.Now(),
					Duration:    time.Since(start),
					Critical:    false,
					Metadata: map[string]interface{}{
						"component": componentName,
						"error":     err.Error(),
					},
				}
			}

			return HealthCheck{
				Name:        fmt.Sprintf("component_%s", componentName),
				Status:      HealthStatusHealthy,
				Message:     "Component is functioning correctly",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    false,
				Metadata: map[string]interface{}{
					"component": componentName,
				},
			}
		},
	)
}

// BuildPipelineHealthChecker creates a health check for build pipeline
func BuildPipelineHealthChecker(buildFn func() error) HealthChecker {
	return NewHealthCheckFunc("build_pipeline", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		if err := buildFn(); err != nil {
			return HealthCheck{
				Name:        "build_pipeline",
				Status:      HealthStatusUnhealthy,
				Message:     fmt.Sprintf("Build pipeline failed: %v", err),
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}

		return HealthCheck{
			Name:        "build_pipeline",
			Status:      HealthStatusHealthy,
			Message:     "Build pipeline is operational",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
		}
	})
}

// FileWatcherHealthChecker creates a health check for file watcher
func FileWatcherHealthChecker(isWatchingFn func() bool) HealthChecker {
	return NewHealthCheckFunc("file_watcher", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		if !isWatchingFn() {
			return HealthCheck{
				Name:        "file_watcher",
				Status:      HealthStatusUnhealthy,
				Message:     "File watcher is not active",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
			}
		}

		return HealthCheck{
			Name:        "file_watcher",
			Status:      HealthStatusHealthy,
			Message:     "File watcher is active and monitoring",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
		}
	})
}

// WebSocketHealthChecker creates a health check for WebSocket connections
func WebSocketHealthChecker(connectionCountFn func() int) HealthChecker {
	return NewHealthCheckFunc("websocket", false, func(ctx context.Context) HealthCheck {
		start := time.Now()
		connectionCount := connectionCountFn()

		status := HealthStatusHealthy
		message := fmt.Sprintf("WebSocket service operational with %d connections", connectionCount)

		// Consider it degraded if there are too many connections
		if connectionCount > 100 {
			status = HealthStatusDegraded
			message = fmt.Sprintf("High number of WebSocket connections: %d", connectionCount)
		}

		return HealthCheck{
			Name:        "websocket",
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    false,
			Metadata: map[string]interface{}{
				"connection_count": connectionCount,
			},
		}
	})
}

// LoggingIntegration provides integration between monitoring and logging
type LoggingIntegration struct {
	monitor *Monitor
	logger  logging.Logger
}

// NewLoggingIntegration creates a new logging integration
func NewLoggingIntegration(monitor *Monitor, logger logging.Logger) *LoggingIntegration {
	return &LoggingIntegration{
		monitor: monitor,
		logger:  logger,
	}
}

// LogWithMetrics logs a message and records corresponding metrics for monitoring
// and observability. This method provides integrated logging with automatic
// metric collection for error tracking and log entry counting.
//
// Metrics recorded:
// - "templar_errors_total" with labels category=component, component=operation
// - "templar_log_entries_total" with labels level=LogLevel, component=component
func (li *LoggingIntegration) LogWithMetrics(
	ctx context.Context,
	level logging.LogLevel,
	component, operation string,
	err error,
	message string,
	fields ...interface{},
) {
	// Record monitoring metrics if monitor is available
	if li.monitor != nil && li.monitor.appMetrics != nil {
		if err != nil {
			// Record error metric with category=component, component=operation labels
			// This allows tracking errors by component category and specific operation
			li.monitor.appMetrics.ErrorOccurred(component, operation)
		}

		// Track log entry count by level and component for observability
		li.monitor.metrics.Counter("log_entries_total", map[string]string{
			"level":     level.String(),
			"component": component,
		})
	}

	// Create logger with component context
	componentLogger := li.logger.WithComponent(component)

	// Log based on level
	switch level {
	case logging.LevelDebug:
		componentLogger.Debug(ctx, message, fields...)
	case logging.LevelInfo:
		componentLogger.Info(ctx, message, fields...)
	case logging.LevelWarn:
		componentLogger.Warn(ctx, err, message, fields...)
	case logging.LevelError:
		componentLogger.Error(ctx, err, message, fields...)
	case logging.LevelFatal:
		componentLogger.Fatal(ctx, err, message, fields...)
	}
}

// OperationTracker tracks operations with logging and metrics
type OperationTracker struct {
	monitor   *Monitor
	logger    logging.Logger
	component string
}

// NewOperationTracker creates a new operation tracker
func NewOperationTracker(
	monitor *Monitor,
	logger logging.Logger,
	component string,
) *OperationTracker {
	return &OperationTracker{
		monitor:   monitor,
		logger:    logger.WithComponent(component),
		component: component,
	}
}

// TrackOperation tracks an operation with logging and metrics
func (ot *OperationTracker) TrackOperation(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	start := time.Now()

	// Log operation start
	ot.logger.Info(ctx, "Starting operation", "operation", operation)

	// Track metrics if monitor is available
	var timer func()
	if ot.monitor != nil && ot.monitor.metrics != nil {
		timer = ot.monitor.metrics.TimerContext(
			ctx,
			fmt.Sprintf("%s_%s", ot.component, operation),
			map[string]string{
				"component": ot.component,
				"operation": operation,
			},
		)
	}

	// Execute operation
	err := fn(ctx)
	duration := time.Since(start)

	// Complete timer
	if timer != nil {
		timer()
	}

	// Log operation completion
	if err != nil {
		ot.logger.Error(ctx, err, "Operation failed",
			"operation", operation,
			"duration", duration)

		if ot.monitor != nil && ot.monitor.appMetrics != nil {
			ot.monitor.appMetrics.ErrorOccurred(ot.component, operation)
		}
	} else {
		ot.logger.Info(ctx, "Operation completed successfully",
			"operation", operation,
			"duration", duration)
	}

	// Track operation completion
	if ot.monitor != nil && ot.monitor.appMetrics != nil {
		success := err == nil
		ot.monitor.TrackComponentOperation(operation, ot.component, success)
	}

	return err
}

// BatchTracker tracks batch operations
type BatchTracker struct {
	operationTracker *OperationTracker
	batchSize        int
	processedCount   int
	errorCount       int
	start            time.Time
}

// NewBatchTracker creates a new batch tracker
func NewBatchTracker(
	monitor *Monitor,
	logger logging.Logger,
	component string,
	batchSize int,
) *BatchTracker {
	return &BatchTracker{
		operationTracker: NewOperationTracker(monitor, logger, component),
		batchSize:        batchSize,
		start:            time.Now(),
	}
}

// TrackItem tracks processing of a single item in the batch
func (bt *BatchTracker) TrackItem(ctx context.Context, itemName string, fn func() error) error {
	err := fn()
	bt.processedCount++

	if err != nil {
		bt.errorCount++
		bt.operationTracker.logger.Error(ctx, err, "Batch item processing failed",
			"item", itemName,
			"processed", bt.processedCount,
			"errors", bt.errorCount)
	}

	// Log progress periodically
	if bt.processedCount%10 == 0 {
		bt.operationTracker.logger.Info(ctx, "Batch processing progress",
			"processed", bt.processedCount,
			"total", bt.batchSize,
			"errors", bt.errorCount,
			"progress_percent", float64(bt.processedCount)/float64(bt.batchSize)*100)
	}

	return err
}

// Complete completes the batch processing and logs summary
func (bt *BatchTracker) Complete(ctx context.Context) {
	duration := time.Since(bt.start)
	successCount := bt.processedCount - bt.errorCount

	bt.operationTracker.logger.Info(ctx, "Batch processing completed",
		"total_processed", bt.processedCount,
		"successful", successCount,
		"errors", bt.errorCount,
		"duration", duration,
		"items_per_second", float64(bt.processedCount)/duration.Seconds())

	// Record batch metrics
	if bt.operationTracker.monitor != nil && bt.operationTracker.monitor.metrics != nil {
		bt.operationTracker.monitor.metrics.Histogram(
			"batch_processing_duration_seconds",
			duration.Seconds(),
			map[string]string{
				"component": bt.operationTracker.component,
			},
		)

		bt.operationTracker.monitor.metrics.Gauge(
			"batch_success_rate",
			float64(successCount)/float64(bt.processedCount),
			map[string]string{
				"component": bt.operationTracker.component,
			},
		)
	}
}

// MonitoringConfig provides configuration for monitoring integration
type MonitoringConfig struct {
	EnableHTTPMiddleware bool   `yaml:"enable_http_middleware" json:"enable_http_middleware"`
	EnableHealthChecks   bool   `yaml:"enable_health_checks" json:"enable_health_checks"`
	EnableMetrics        bool   `yaml:"enable_metrics" json:"enable_metrics"`
	LogLevel             string `yaml:"log_level" json:"log_level"`
}

// SetupMonitoring sets up monitoring for the entire application
func SetupMonitoring(config MonitoringConfig) (*Monitor, error) {
	// Create monitor with default config
	monitorConfig := DefaultMonitorConfig()

	// Override with provided config
	if !config.EnableMetrics {
		monitorConfig.MetricsEnabled = false
	}
	if !config.EnableHealthChecks {
		monitorConfig.HealthEnabled = false
	}

	// Create logger
	logLevel := logging.LevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = logging.LevelDebug
	case "warn":
		logLevel = logging.LevelWarn
	case "error":
		logLevel = logging.LevelError
	}

	loggerConfig := &logging.LoggerConfig{
		Level:     logLevel,
		Format:    "json",
		AddSource: true,
	}

	logger := logging.NewLogger(loggerConfig)

	// Create monitor
	monitor, err := NewMonitor(monitorConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	// Set as global monitor
	SetGlobalMonitor(monitor)

	return monitor, nil
}

// GetMiddleware returns HTTP middleware if monitoring is enabled
func GetMiddleware() func(http.Handler) http.Handler {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return MonitoringMiddleware(monitor)
}

// TrackOperation is a convenience function for tracking operations globally
func TrackOperation(
	ctx context.Context,
	component, operation string,
	fn func(ctx context.Context) error,
) error {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return fn(ctx)
	}

	tracker := NewOperationTracker(monitor, monitor.GetLogger(), component)
	return tracker.TrackOperation(ctx, operation, fn)
}

// LogError is a convenience function for logging errors with metrics
func LogError(
	ctx context.Context,
	component, operation string,
	err error,
	message string,
	fields ...interface{},
) {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return
	}

	integration := NewLoggingIntegration(monitor, monitor.GetLogger())
	integration.LogWithMetrics(
		ctx,
		logging.LevelError,
		component,
		operation,
		err,
		message,
		fields...)
}

// LogInfo is a convenience function for logging info with metrics
func LogInfo(
	ctx context.Context,
	component, operation string,
	message string,
	fields ...interface{},
) {
	monitor := GetGlobalMonitor()
	if monitor == nil {
		return
	}

	integration := NewLoggingIntegration(monitor, monitor.GetLogger())
	integration.LogWithMetrics(
		ctx,
		logging.LevelInfo,
		component,
		operation,
		nil,
		message,
		fields...)
}
