package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Critical    bool                   `json:"critical"`
}

// HealthChecker defines the interface for health check functions
type HealthChecker interface {
	Check(ctx context.Context) HealthCheck
	Name() string
	IsCritical() bool
}

// HealthCheckFunc is a function that implements HealthChecker
type HealthCheckFunc struct {
	name     string
	checkFn  func(ctx context.Context) HealthCheck
	critical bool
}

// Check executes the health check function
func (h *HealthCheckFunc) Check(ctx context.Context) HealthCheck {
	return h.checkFn(ctx)
}

// Name returns the health check name
func (h *HealthCheckFunc) Name() string {
	return h.name
}

// IsCritical returns whether this check is critical
func (h *HealthCheckFunc) IsCritical() bool {
	return h.critical
}

// NewHealthCheckFunc creates a new health check function
func NewHealthCheckFunc(
	name string,
	critical bool,
	checkFn func(ctx context.Context) HealthCheck,
) *HealthCheckFunc {
	return &HealthCheckFunc{
		name:     name,
		checkFn:  checkFn,
		critical: critical,
	}
}

// HealthMonitor manages and executes health checks
type HealthMonitor struct {
	checks   map[string]HealthChecker
	results  map[string]HealthCheck
	mutex    sync.RWMutex
	logger   logging.Logger
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status      HealthStatus           `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version,omitempty"`
	Uptime      time.Duration          `json:"uptime"`
	Checks      map[string]HealthCheck `json:"checks"`
	Summary     HealthSummary          `json:"summary"`
	SystemInfo  SystemInfo             `json:"system_info"`
	Environment string                 `json:"environment,omitempty"`
}

// HealthSummary provides a summary of health check results
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Degraded  int `json:"degraded"`
	Unknown   int `json:"unknown"`
	Critical  int `json:"critical"`
}

// SystemInfo provides system information
type SystemInfo struct {
	Hostname  string    `json:"hostname"`
	Platform  string    `json:"platform"`
	GoVersion string    `json:"go_version"`
	StartTime time.Time `json:"start_time"`
	PID       int       `json:"pid"`
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(logger logging.Logger) *HealthMonitor {
	return &HealthMonitor{
		checks:   make(map[string]HealthChecker),
		results:  make(map[string]HealthCheck),
		logger:   logger.WithComponent("health_monitor"),
		interval: 30 * time.Second,
		timeout:  10 * time.Second,
		stopChan: make(chan struct{}),
	}
}

// RegisterCheck registers a health check
func (hm *HealthMonitor) RegisterCheck(checker HealthChecker) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.checks[checker.Name()] = checker
	hm.logger.Info(context.Background(), "Registered health check",
		"name", checker.Name(),
		"critical", checker.IsCritical())
}

// UnregisterCheck removes a health check
func (hm *HealthMonitor) UnregisterCheck(name string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	delete(hm.checks, name)
	delete(hm.results, name)
	hm.logger.Info(context.Background(), "Unregistered health check", "name", name)
}

// Start begins periodic health checking
func (hm *HealthMonitor) Start() {
	hm.wg.Add(1)
	go hm.monitorLoop()
	hm.logger.Info(context.Background(), "Health monitor started", "interval", hm.interval)
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	// Only close if not already closed
	select {
	case <-hm.stopChan:
		// Already closed
	default:
		close(hm.stopChan)
	}
	hm.wg.Wait()
	hm.logger.Info(context.Background(), "Health monitor stopped")
}

// monitorLoop runs the health check monitoring loop
func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// Run initial health checks
	hm.runHealthChecks()

	for {
		select {
		case <-ticker.C:
			hm.runHealthChecks()
		case <-hm.stopChan:
			return
		}
	}
}

// runHealthChecks executes all registered health checks
func (hm *HealthMonitor) runHealthChecks() {
	hm.mutex.RLock()
	checks := make(map[string]HealthChecker)
	for name, checker := range hm.checks {
		checks[name] = checker
	}
	hm.mutex.RUnlock()

	var wg sync.WaitGroup
	resultsChan := make(chan HealthCheck, len(checks))

	// Run checks concurrently
	for _, checker := range checks {
		wg.Add(1)
		go func(checker HealthChecker) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
			defer cancel()

			start := time.Now()
			result := checker.Check(ctx)
			result.Duration = time.Since(start)
			result.LastChecked = time.Now()

			resultsChan <- result
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	hm.mutex.Lock()
	for result := range resultsChan {
		hm.results[result.Name] = result

		// Log unhealthy checks
		if result.Status != HealthStatusHealthy {
			hm.logger.Warn(context.Background(), nil, "Health check failed",
				"name", result.Name,
				"status", string(result.Status),
				"message", result.Message,
				"duration", result.Duration)
		}
	}
	hm.mutex.Unlock()
}

// GetHealth returns the current health status
func (hm *HealthMonitor) GetHealth() HealthResponse {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	checks := make(map[string]HealthCheck)
	for name, result := range hm.results {
		checks[name] = result
	}

	summary := hm.calculateSummary(checks)
	overallStatus := hm.calculateOverallStatus(checks)

	return HealthResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Uptime:      time.Since(startTime),
		Checks:      checks,
		Summary:     summary,
		SystemInfo:  getSystemInfo(),
		Environment: getEnvironment(),
	}
}

// calculateSummary calculates health check summary
func (hm *HealthMonitor) calculateSummary(checks map[string]HealthCheck) HealthSummary {
	summary := HealthSummary{
		Total: len(checks),
	}

	for _, check := range checks {
		switch check.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusUnhealthy:
			summary.Unhealthy++
		case HealthStatusDegraded:
			summary.Degraded++
		case HealthStatusUnknown:
			summary.Unknown++
		}

		if check.Critical {
			summary.Critical++
		}
	}

	return summary
}

// calculateOverallStatus determines the overall health status
func (hm *HealthMonitor) calculateOverallStatus(checks map[string]HealthCheck) HealthStatus {
	// If any critical check is unhealthy, overall status is unhealthy
	for _, check := range checks {
		if check.Critical && check.Status == HealthStatusUnhealthy {
			return HealthStatusUnhealthy
		}
	}

	// If any check is degraded, overall status is degraded
	for _, check := range checks {
		if check.Status == HealthStatusDegraded {
			return HealthStatusDegraded
		}
	}

	// If any non-critical check is unhealthy, overall status is degraded
	for _, check := range checks {
		if !check.Critical && check.Status == HealthStatusUnhealthy {
			return HealthStatusDegraded
		}
	}

	// All checks are healthy
	return HealthStatusHealthy
}

// HTTPHandler returns an HTTP handler for health checks
func (hm *HealthMonitor) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hm.GetHealth()

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status based on health
		switch health.Status {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // 200 for degraded
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(health); err != nil {
			hm.logger.Error(context.Background(), err, "Failed to encode health response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// Predefined health checks

// FileSystemHealthChecker checks file system accessibility
func FileSystemHealthChecker(path string) HealthChecker {
	return NewHealthCheckFunc("filesystem", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		// Try to create and remove a temp file
		tempFile := fmt.Sprintf("%s/.health_check_%d", path, time.Now().UnixNano())

		if err := os.WriteFile(tempFile, []byte("health_check"), 0644); err != nil {
			return HealthCheck{
				Name:        "filesystem",
				Status:      HealthStatusUnhealthy,
				Message:     fmt.Sprintf("Cannot write to filesystem: %v", err),
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
			}
		}

		if err := os.Remove(tempFile); err != nil {
			return HealthCheck{
				Name:        "filesystem",
				Status:      HealthStatusDegraded,
				Message:     fmt.Sprintf("Cannot remove temp file: %v", err),
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
			}
		}

		return HealthCheck{
			Name:        "filesystem",
			Status:      HealthStatusHealthy,
			Message:     "Filesystem is accessible",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
		}
	})
}

// MemoryHealthChecker checks memory usage
func MemoryHealthChecker() HealthChecker {
	return NewHealthCheckFunc("memory", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		// Check if memory usage is concerning (>1GB heap)
		const maxHeapSize = 1 * 1024 * 1024 * 1024 // 1GB

		status := HealthStatusHealthy
		message := "Memory usage is normal"

		if mem.HeapAlloc > maxHeapSize {
			status = HealthStatusDegraded
			message = fmt.Sprintf("High memory usage: %d bytes", mem.HeapAlloc)
		}

		// Check for memory leak indicators (high GC frequency)
		if mem.NumGC > 1000 && mem.PauseNs[(mem.NumGC+255)%256] > 100*1000*1000 { // 100ms
			status = HealthStatusUnhealthy
			message = "Potential memory leak detected"
		}

		return HealthCheck{
			Name:        "memory",
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
			Metadata: map[string]interface{}{
				"heap_alloc":    mem.HeapAlloc,
				"heap_sys":      mem.HeapSys,
				"gc_runs":       mem.NumGC,
				"last_gc_pause": mem.PauseNs[(mem.NumGC+255)%256],
			},
		}
	})
}

// GoroutineHealthChecker checks for goroutine leaks
func GoroutineHealthChecker() HealthChecker {
	return NewHealthCheckFunc("goroutines", false, func(ctx context.Context) HealthCheck {
		start := time.Now()

		goroutines := runtime.NumGoroutine()

		status := HealthStatusHealthy
		message := "Goroutine count is normal"

		// Check for potential goroutine leaks
		if goroutines > 1000 {
			status = HealthStatusDegraded
			message = fmt.Sprintf("High goroutine count: %d", goroutines)
		}

		if goroutines > 10000 {
			status = HealthStatusUnhealthy
			message = fmt.Sprintf("Very high goroutine count: %d", goroutines)
		}

		return HealthCheck{
			Name:        "goroutines",
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    false,
			Metadata: map[string]interface{}{
				"count": goroutines,
			},
		}
	})
}

// Helper functions

var startTime = time.Now()

func getSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()

	return SystemInfo{
		Hostname:  hostname,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: runtime.Version(),
		StartTime: startTime,
		PID:       os.Getpid(),
	}
}

func getEnvironment() string {
	env := os.Getenv("TEMPLAR_ENV")
	if env == "" {
		env = "development"
	}
	return env
}
