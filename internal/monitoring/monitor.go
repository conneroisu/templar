package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// Monitor provides comprehensive application monitoring.
type Monitor struct {
	metrics    *MetricsCollector
	health     *HealthMonitor
	appMetrics *ApplicationMetrics
	logger     logging.Logger
	config     MonitorConfig
	httpServer *http.Server
	stopChan   chan struct{}
	wg         sync.WaitGroup
	started    bool
	mutex      sync.RWMutex
}

// MonitorConfig contains monitoring configuration.
type MonitorConfig struct {
	// Metrics configuration
	MetricsEnabled    bool          `yaml:"metrics_enabled" json:"metrics_enabled"`
	MetricsOutputPath string        `yaml:"metrics_output_path" json:"metrics_output_path"`
	MetricsPrefix     string        `yaml:"metrics_prefix" json:"metrics_prefix"`
	MetricsInterval   time.Duration `yaml:"metrics_interval" json:"metrics_interval"`

	// Health check configuration
	HealthEnabled       bool          `yaml:"health_enabled" json:"health_enabled"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout"`

	// HTTP server configuration for exposing metrics/health
	HTTPEnabled bool   `yaml:"http_enabled" json:"http_enabled"`
	HTTPAddr    string `yaml:"http_addr" json:"http_addr"`
	HTTPPort    int    `yaml:"http_port" json:"http_port"`

	// Logging configuration
	LogLevel          string `yaml:"log_level" json:"log_level"`
	LogFormat         string `yaml:"log_format" json:"log_format"`
	LogOutputPath     string `yaml:"log_output_path" json:"log_output_path"`
	StructuredLogging bool   `yaml:"structured_logging" json:"structured_logging"`

	// Alerting configuration
	AlertingEnabled bool          `yaml:"alerting_enabled" json:"alerting_enabled"`
	AlertThresholds AlertConfig   `yaml:"alert_thresholds" json:"alert_thresholds"`
	AlertCooldown   time.Duration `yaml:"alert_cooldown" json:"alert_cooldown"`
}

// AlertConfig contains alerting thresholds.
type AlertConfig struct {
	ErrorRate           float64       `yaml:"error_rate" json:"error_rate"`
	ResponseTime        time.Duration `yaml:"response_time" json:"response_time"`
	MemoryUsage         int64         `yaml:"memory_usage" json:"memory_usage"`
	GoroutineCount      int           `yaml:"goroutine_count" json:"goroutine_count"`
	DiskUsage           float64       `yaml:"disk_usage" json:"disk_usage"`
	UnhealthyComponents int           `yaml:"unhealthy_components" json:"unhealthy_components"`
}

// DefaultMonitorConfig returns default monitoring configuration.
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		MetricsEnabled:      true,
		MetricsOutputPath:   "./logs/metrics.json",
		MetricsPrefix:       "templar",
		MetricsInterval:     30 * time.Second,
		HealthEnabled:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  10 * time.Second,
		HTTPEnabled:         true,
		HTTPAddr:            "localhost",
		HTTPPort:            8081,
		LogLevel:            "info",
		LogFormat:           "json",
		LogOutputPath:       "./logs/monitor.log",
		StructuredLogging:   true,
		AlertingEnabled:     false,
		AlertThresholds: AlertConfig{
			ErrorRate:           0.1, // 10%
			ResponseTime:        5 * time.Second,
			MemoryUsage:         1024 * 1024 * 1024, // 1GB
			GoroutineCount:      1000,
			DiskUsage:           0.9, // 90%
			UnhealthyComponents: 1,
		},
		AlertCooldown: 5 * time.Minute,
	}
}

// NewMonitor creates a new comprehensive monitor.
func NewMonitor(config MonitorConfig, logger logging.Logger) (*Monitor, error) {
	if logger == nil {
		// Create default logger if none provided
		logConfig := &logging.LoggerConfig{
			Level:     logging.LevelInfo,
			Format:    config.LogFormat,
			AddSource: true,
		}

		if config.LogOutputPath != "" {
			// Create file logger
			fileLogger, err := logging.NewFileLogger(logConfig, filepath.Dir(config.LogOutputPath))
			if err != nil {
				return nil, fmt.Errorf("failed to create file logger: %w", err)
			}
			logger = fileLogger
		} else {
			logger = logging.NewLogger(logConfig)
		}
	}

	// Create metrics collector
	var metricsCollector *MetricsCollector
	if config.MetricsEnabled {
		metricsCollector = NewMetricsCollector(config.MetricsPrefix, config.MetricsOutputPath)
		metricsCollector.flushPeriod = config.MetricsInterval
	}

	// Create health monitor
	var healthMonitor *HealthMonitor
	if config.HealthEnabled {
		healthMonitor = NewHealthMonitor(logger)
		healthMonitor.interval = config.HealthCheckInterval
		healthMonitor.timeout = config.HealthCheckTimeout
	}

	// Create application metrics
	var appMetrics *ApplicationMetrics
	if metricsCollector != nil {
		appMetrics = NewApplicationMetrics(metricsCollector)
		metricsCollector.RegisterCollector(appMetrics)
	}

	monitor := &Monitor{
		metrics:    metricsCollector,
		health:     healthMonitor,
		appMetrics: appMetrics,
		logger:     logger.WithComponent("monitor"),
		config:     config,
		stopChan:   make(chan struct{}),
	}

	// Set up HTTP server if enabled
	if config.HTTPEnabled {
		monitor.setupHTTPServer()
	}

	return monitor, nil
}

// Start starts all monitoring components.
func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return errors.New("monitor already started")
	}

	m.logger.Info(context.Background(), "Starting monitor",
		"metrics_enabled", m.config.MetricsEnabled,
		"health_enabled", m.config.HealthEnabled,
		"http_enabled", m.config.HTTPEnabled)

	// Start metrics collector
	if m.metrics != nil {
		m.metrics.Start()
		m.logger.Info(context.Background(), "Metrics collector started")
	}

	// Start health monitor
	if m.health != nil {
		// Register default health checks
		m.registerDefaultHealthChecks()
		m.health.Start()
		m.logger.Info(context.Background(), "Health monitor started")
	}

	// Start HTTP server
	if m.httpServer != nil {
		m.wg.Add(1)
		go m.runHTTPServer()
		m.logger.Info(context.Background(), "HTTP server started",
			"addr", fmt.Sprintf("%s:%d", m.config.HTTPAddr, m.config.HTTPPort))
	}

	// Start alerting if enabled
	if m.config.AlertingEnabled {
		m.wg.Add(1)
		go m.runAlerting()
		m.logger.Info(context.Background(), "Alerting system started")
	}

	m.started = true
	m.logger.Info(context.Background(), "Monitor started successfully")

	return nil
}

// Stop stops all monitoring components.
func (m *Monitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Info(context.Background(), "Stopping monitor")

	// Stop components - only close if not already closed
	select {
	case <-m.stopChan:
		// Already closed
	default:
		close(m.stopChan)
	}

	// Stop metrics collector
	if m.metrics != nil {
		m.metrics.Stop()
		m.logger.Info(context.Background(), "Metrics collector stopped")
	}

	// Stop health monitor
	if m.health != nil {
		m.health.Stop()
		m.logger.Info(context.Background(), "Health monitor stopped")
	}

	// Stop HTTP server
	if m.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.httpServer.Shutdown(ctx); err != nil {
			m.logger.Error(context.Background(), err, "Failed to shutdown HTTP server")
		} else {
			m.logger.Info(context.Background(), "HTTP server stopped")
		}
	}

	// Wait for goroutines to finish
	m.wg.Wait()

	m.started = false
	m.logger.Info(context.Background(), "Monitor stopped successfully")

	return nil
}

// GetMetrics returns the metrics collector.
func (m *Monitor) GetMetrics() *ApplicationMetrics {
	return m.appMetrics
}

// GetHealth returns the health monitor.
func (m *Monitor) GetHealth() *HealthMonitor {
	return m.health
}

// GetLogger returns the logger.
func (m *Monitor) GetLogger() logging.Logger {
	return m.logger
}

// RegisterHealthCheck registers a custom health check.
func (m *Monitor) RegisterHealthCheck(checker HealthChecker) {
	if m.health != nil {
		m.health.RegisterCheck(checker)
	}
}

// setupHTTPServer configures the HTTP server for metrics and health endpoints.
func (m *Monitor) setupHTTPServer() {
	mux := http.NewServeMux()

	// Health endpoint
	if m.health != nil {
		mux.Handle("/health", m.health.HTTPHandler())
		mux.Handle("/health/live", m.createLivenessHandler())
		mux.Handle("/health/ready", m.createReadinessHandler())
	}

	// Metrics endpoint
	if m.metrics != nil {
		mux.Handle("/metrics", m.createMetricsHandler())
	}

	// Info endpoint
	mux.Handle("/info", m.createInfoHandler())

	m.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", m.config.HTTPAddr, m.config.HTTPPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

// runHTTPServer runs the HTTP server.
func (m *Monitor) runHTTPServer() {
	defer m.wg.Done()

	if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		m.logger.Error(context.Background(), err, "HTTP server error")
	}
}

// runAlerting runs the alerting system.
func (m *Monitor) runAlerting() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAlerts()
		case <-m.stopChan:
			return
		}
	}
}

// checkAlerts checks for alert conditions.
func (m *Monitor) checkAlerts() {
	if m.health == nil {
		return
	}

	health := m.health.GetHealth()

	// Check unhealthy components threshold
	if health.Summary.Unhealthy >= m.config.AlertThresholds.UnhealthyComponents {
		m.logger.Error(context.Background(), nil, "Alert: Too many unhealthy components",
			"unhealthy_count", health.Summary.Unhealthy,
			"threshold", m.config.AlertThresholds.UnhealthyComponents)
	}

	// Check memory usage from system info
	if memCheck, exists := health.Checks["memory"]; exists {
		if memoryUsage, ok := memCheck.Metadata["heap_alloc"].(uint64); ok {
			if int64(memoryUsage) > m.config.AlertThresholds.MemoryUsage {
				m.logger.Error(context.Background(), nil, "Alert: High memory usage",
					"current", memoryUsage,
					"threshold", m.config.AlertThresholds.MemoryUsage)
			}
		}
	}

	// Check goroutine count
	if goroutineCheck, exists := health.Checks["goroutines"]; exists {
		if goroutineCount, ok := goroutineCheck.Metadata["count"].(int); ok {
			if goroutineCount > m.config.AlertThresholds.GoroutineCount {
				m.logger.Error(context.Background(), nil, "Alert: High goroutine count",
					"current", goroutineCount,
					"threshold", m.config.AlertThresholds.GoroutineCount)
			}
		}
	}
}

// registerDefaultHealthChecks registers standard health checks.
func (m *Monitor) registerDefaultHealthChecks() {
	// Filesystem health check
	m.health.RegisterCheck(FileSystemHealthChecker("./"))

	// Memory health check
	m.health.RegisterCheck(MemoryHealthChecker())

	// Goroutine health check
	m.health.RegisterCheck(GoroutineHealthChecker())
}

// HTTP handlers

func (m *Monitor) createLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func (m *Monitor) createReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.health == nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

			return
		}

		health := m.health.GetHealth()
		if health.Status == HealthStatusHealthy || health.Status == HealthStatusDegraded {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Ready"))
		}
	}
}

func (m *Monitor) createMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.metrics == nil {
			http.Error(w, "Metrics not enabled", http.StatusNotFound)

			return
		}

		metrics := m.metrics.GatherMetrics()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"timestamp": time.Now(),
			"metrics":   metrics,
		}); err != nil {
			m.logger.Error(context.Background(), err, "Failed to encode metrics")
		}
	}
}

func (m *Monitor) createInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"application": "templar",
			"version":     "latest", // This could be injected at build time
			"uptime":      time.Since(startTime),
			"started_at":  startTime,
			"system_info": getSystemInfo(),
			"config": map[string]interface{}{
				"metrics_enabled": m.config.MetricsEnabled,
				"health_enabled":  m.config.HealthEnabled,
				"http_enabled":    m.config.HTTPEnabled,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(info); err != nil {
			m.logger.Error(context.Background(), err, "Failed to encode info")
		}
	}
}

// Convenience functions for integration with other components

// LogOperation logs an operation with metrics.
func (m *Monitor) LogOperation(operation string, fn func() error) error {
	if m.appMetrics == nil {
		return fn()
	}

	timer := m.metrics.Timer(operation, nil)
	defer timer()

	err := fn()
	if err != nil {
		m.appMetrics.ErrorOccurred("operation", operation)
	}

	return err
}

// LogOperationWithContext logs an operation with context and metrics.
func (m *Monitor) LogOperationWithContext(
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) error,
) error {
	start := time.Now()

	if m.appMetrics != nil {
		defer func() {
			duration := time.Since(start)
			m.appMetrics.BuildDuration(operation, duration)
		}()
	}

	err := fn(ctx)
	if err != nil && m.appMetrics != nil {
		m.appMetrics.ErrorOccurred("operation", operation)
	}

	return err
}

// TrackHTTPRequest tracks an HTTP request.
func (m *Monitor) TrackHTTPRequest(method, path string, statusCode int) {
	if m.appMetrics != nil {
		m.appMetrics.ServerRequest(method, path, statusCode)
	}
}

// TrackWebSocketEvent tracks a WebSocket event.
func (m *Monitor) TrackWebSocketEvent(action string) {
	if m.appMetrics != nil {
		m.appMetrics.WebSocketConnection(action)
	}
}

// TrackComponentOperation tracks component operations.
func (m *Monitor) TrackComponentOperation(operation, component string, success bool) {
	if m.appMetrics == nil {
		return
	}

	switch operation {
	case "scan":
		m.appMetrics.ComponentScanned(component)
	case "build":
		m.appMetrics.ComponentBuilt(component, success)
	}
}

// Global monitor instance for easy access.
var globalMonitor *Monitor
var globalMonitorMutex sync.RWMutex

// SetGlobalMonitor sets the global monitor instance.
func SetGlobalMonitor(monitor *Monitor) {
	globalMonitorMutex.Lock()
	defer globalMonitorMutex.Unlock()
	globalMonitor = monitor
}

// GetGlobalMonitor returns the global monitor instance.
func GetGlobalMonitor() *Monitor {
	globalMonitorMutex.RLock()
	defer globalMonitorMutex.RUnlock()

	return globalMonitor
}
