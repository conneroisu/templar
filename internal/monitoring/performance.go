package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// PerformanceMonitor tracks detailed performance metrics
type PerformanceMonitor struct {
	metrics *MetricsCollector
	logger  logging.Logger
	mutex   sync.RWMutex

	// Performance tracking
	operationMetrics map[string]*OperationMetrics
	resourceMetrics  *ResourceMetrics

	// Configuration
	trackingEnabled bool
	sampleRate      float64
	maxOperations   int
}

// OperationMetrics tracks performance for a specific operation
type OperationMetrics struct {
	Name        string
	Count       int64
	TotalTime   time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	LastTime    time.Duration
	ErrorCount  int64
	ActiveCount int64

	// Percentile tracking (simplified)
	durations   []time.Duration
	maxSamples  int
	lastCleanup time.Time
}

// ResourceMetrics tracks system resource usage
type ResourceMetrics struct {
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryPercent float64
	GCPauses      []time.Duration
	GoroutineHigh int

	// File descriptor usage (Linux/Unix)
	OpenFiles int
	MaxFiles  int

	// Network stats (if available)
	NetworkStats *NetworkStats

	lastUpdate time.Time
}

// NetworkStats tracks network-related metrics
type NetworkStats struct {
	ConnectionsActive int
	ConnectionsTotal  int64
	BytesSent         int64
	BytesReceived     int64
}

// PerformanceSnapshot represents a point-in-time performance snapshot
type PerformanceSnapshot struct {
	Timestamp  time.Time                   `json:"timestamp"`
	Operations map[string]OperationSummary `json:"operations"`
	Resources  ResourceSummary             `json:"resources"`
	SystemInfo SystemPerformanceInfo       `json:"system_info"`
}

// OperationSummary provides summary statistics for an operation
type OperationSummary struct {
	Name           string        `json:"name"`
	Count          int64         `json:"count"`
	ErrorRate      float64       `json:"error_rate"`
	AvgDuration    time.Duration `json:"avg_duration"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	P50Duration    time.Duration `json:"p50_duration"`
	P95Duration    time.Duration `json:"p95_duration"`
	P99Duration    time.Duration `json:"p99_duration"`
	ThroughputRPS  float64       `json:"throughput_rps"`
	ActiveRequests int64         `json:"active_requests"`
}

// ResourceSummary provides system resource summary
type ResourceSummary struct {
	CPUPercent      float64       `json:"cpu_percent"`
	MemoryUsage     uint64        `json:"memory_usage_bytes"`
	MemoryPercent   float64       `json:"memory_percent"`
	GCPauseAvg      time.Duration `json:"gc_pause_avg"`
	GCPauseMax      time.Duration `json:"gc_pause_max"`
	GoroutineCount  int           `json:"goroutine_count"`
	GoroutineHigh   int           `json:"goroutine_high"`
	OpenFiles       int           `json:"open_files"`
	FileDescPercent float64       `json:"file_desc_percent"`
}

// SystemPerformanceInfo provides detailed system performance information
type SystemPerformanceInfo struct {
	Uptime     time.Duration `json:"uptime"`
	GoVersion  string        `json:"go_version"`
	NumCPU     int           `json:"num_cpu"`
	GOOS       string        `json:"goos"`
	GOARCH     string        `json:"goarch"`
	CGOEnabled bool          `json:"cgo_enabled"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(metrics *MetricsCollector, logger logging.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:          metrics,
		logger:           logger.WithComponent("performance_monitor"),
		operationMetrics: make(map[string]*OperationMetrics),
		resourceMetrics:  &ResourceMetrics{},
		trackingEnabled:  true,
		sampleRate:       1.0, // Track all operations by default
		maxOperations:    1000,
	}
}

// TrackOperation tracks the performance of an operation
func (pm *PerformanceMonitor) TrackOperation(name string, fn func() error) error {
	if !pm.trackingEnabled {
		return fn()
	}

	start := time.Now()

	// Increment active count
	pm.incrementActive(name)
	defer pm.decrementActive(name)

	// Execute operation
	err := fn()

	// Record metrics
	duration := time.Since(start)
	pm.recordOperation(name, duration, err != nil)

	return err
}

// TrackOperationWithContext tracks operation with context
func (pm *PerformanceMonitor) TrackOperationWithContext(
	ctx context.Context,
	name string,
	fn func(context.Context) error,
) error {
	if !pm.trackingEnabled {
		return fn(ctx)
	}

	start := time.Now()

	pm.incrementActive(name)
	defer pm.decrementActive(name)

	err := fn(ctx)

	duration := time.Since(start)
	pm.recordOperation(name, duration, err != nil)

	// Also log to structured logger with context
	if err != nil {
		pm.logger.Error(ctx, err, "Operation failed",
			"operation", name,
			"duration", duration)
	} else {
		pm.logger.Debug(ctx, "Operation completed",
			"operation", name,
			"duration", duration)
	}

	return err
}

// recordOperation records performance metrics for an operation
func (pm *PerformanceMonitor) recordOperation(name string, duration time.Duration, isError bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	op, exists := pm.operationMetrics[name]
	if !exists {
		op = &OperationMetrics{
			Name:        name,
			MinTime:     duration,
			MaxTime:     duration,
			durations:   make([]time.Duration, 0, 1000),
			maxSamples:  1000,
			lastCleanup: time.Now(),
		}
		pm.operationMetrics[name] = op
	}

	// Update metrics
	op.Count++
	op.TotalTime += duration
	op.LastTime = duration

	if duration < op.MinTime {
		op.MinTime = duration
	}
	if duration > op.MaxTime {
		op.MaxTime = duration
	}

	if isError {
		op.ErrorCount++
	}

	// Sample duration for percentile calculation
	if len(op.durations) < op.maxSamples {
		op.durations = append(op.durations, duration)
	} else {
		// Replace random sample to maintain distribution
		idx := int(time.Now().UnixNano()) % len(op.durations)
		op.durations[idx] = duration
	}

	// Cleanup old data periodically
	if time.Since(op.lastCleanup) > 5*time.Minute {
		pm.cleanupOperation(op)
	}

	// Record to metrics collector
	if pm.metrics != nil {
		labels := map[string]string{"operation": name}
		pm.metrics.Histogram("operation_duration_seconds", duration.Seconds(), labels)
		pm.metrics.Counter("operation_total", labels)

		if isError {
			errorLabels := map[string]string{"operation": name, "result": "error"}
			pm.metrics.Counter("operation_errors_total", errorLabels)
		}
	}
}

// incrementActive increments the active operation count
func (pm *PerformanceMonitor) incrementActive(name string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if op, exists := pm.operationMetrics[name]; exists {
		op.ActiveCount++
	}
}

// decrementActive decrements the active operation count
func (pm *PerformanceMonitor) decrementActive(name string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if op, exists := pm.operationMetrics[name]; exists {
		op.ActiveCount--
		if op.ActiveCount < 0 {
			op.ActiveCount = 0
		}
	}
}

// cleanupOperation cleans up old operation data
func (pm *PerformanceMonitor) cleanupOperation(op *OperationMetrics) {
	// Keep only recent samples for percentile calculation
	if len(op.durations) > 100 {
		// Keep the most recent 100 samples
		copy(op.durations[:100], op.durations[len(op.durations)-100:])
		op.durations = op.durations[:100]
	}

	op.lastCleanup = time.Now()
}

// UpdateResourceMetrics updates system resource metrics
func (pm *PerformanceMonitor) UpdateResourceMetrics() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Update memory metrics
	pm.resourceMetrics.MemoryUsage = m.Alloc

	// Update GC metrics
	if len(pm.resourceMetrics.GCPauses) > 10 {
		pm.resourceMetrics.GCPauses = pm.resourceMetrics.GCPauses[1:]
	}
	if m.NumGC > 0 {
		gcPause := time.Duration(m.PauseNs[(m.NumGC+255)%256])
		pm.resourceMetrics.GCPauses = append(pm.resourceMetrics.GCPauses, gcPause)
	}

	// Update goroutine count
	goroutines := runtime.NumGoroutine()
	if goroutines > pm.resourceMetrics.GoroutineHigh {
		pm.resourceMetrics.GoroutineHigh = goroutines
	}

	pm.resourceMetrics.lastUpdate = time.Now()

	// Record to metrics collector
	if pm.metrics != nil {
		pm.metrics.Gauge("memory_usage_bytes", float64(m.Alloc), nil)
		pm.metrics.Gauge("goroutine_count", float64(goroutines), nil)
		pm.metrics.Gauge("gc_pause_ns", float64(m.PauseNs[(m.NumGC+255)%256]), nil)
	}
}

// GetPerformanceSnapshot returns a current performance snapshot
func (pm *PerformanceMonitor) GetPerformanceSnapshot() PerformanceSnapshot {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	snapshot := PerformanceSnapshot{
		Timestamp:  time.Now(),
		Operations: make(map[string]OperationSummary),
		SystemInfo: pm.getSystemInfo(),
	}

	// Build operation summaries
	for name, op := range pm.operationMetrics {
		summary := pm.buildOperationSummary(op)
		snapshot.Operations[name] = summary
	}

	// Build resource summary
	snapshot.Resources = pm.buildResourceSummary()

	return snapshot
}

// buildOperationSummary builds summary statistics for an operation
func (pm *PerformanceMonitor) buildOperationSummary(op *OperationMetrics) OperationSummary {
	summary := OperationSummary{
		Name:           op.Name,
		Count:          op.Count,
		MinDuration:    op.MinTime,
		MaxDuration:    op.MaxTime,
		ActiveRequests: op.ActiveCount,
	}

	if op.Count > 0 {
		summary.AvgDuration = op.TotalTime / time.Duration(op.Count)
		summary.ErrorRate = float64(op.ErrorCount) / float64(op.Count)

		// Calculate throughput (operations per second over last period)
		if op.Count > 0 {
			// Simplified throughput calculation
			summary.ThroughputRPS = float64(op.Count) / time.Since(startTime).Seconds()
		}
	}

	// Calculate percentiles
	if len(op.durations) > 0 {
		sorted := make([]time.Duration, len(op.durations))
		copy(sorted, op.durations)

		// Simple sort for percentile calculation
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		summary.P50Duration = sorted[len(sorted)*50/100]
		if len(sorted) > 20 { // Only calculate P95/P99 if we have enough samples
			summary.P95Duration = sorted[len(sorted)*95/100]
			summary.P99Duration = sorted[len(sorted)*99/100]
		}
	}

	return summary
}

// buildResourceSummary builds resource usage summary
func (pm *PerformanceMonitor) buildResourceSummary() ResourceSummary {
	summary := ResourceSummary{
		MemoryUsage:    pm.resourceMetrics.MemoryUsage,
		GoroutineCount: runtime.NumGoroutine(),
		GoroutineHigh:  pm.resourceMetrics.GoroutineHigh,
		OpenFiles:      pm.resourceMetrics.OpenFiles,
	}

	// Calculate GC pause statistics
	if len(pm.resourceMetrics.GCPauses) > 0 {
		var total time.Duration
		max := time.Duration(0)

		for _, pause := range pm.resourceMetrics.GCPauses {
			total += pause
			if pause > max {
				max = pause
			}
		}

		summary.GCPauseAvg = total / time.Duration(len(pm.resourceMetrics.GCPauses))
		summary.GCPauseMax = max
	}

	// Calculate file descriptor percentage
	if pm.resourceMetrics.MaxFiles > 0 {
		summary.FileDescPercent = float64(
			pm.resourceMetrics.OpenFiles,
		) / float64(
			pm.resourceMetrics.MaxFiles,
		) * 100
	}

	return summary
}

// getSystemInfo returns system performance information
func (pm *PerformanceMonitor) getSystemInfo() SystemPerformanceInfo {
	return SystemPerformanceInfo{
		Uptime:     time.Since(startTime),
		GoVersion:  runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		CGOEnabled: true, // This would need runtime detection
	}
}

// GetTopOperations returns the top N operations by various metrics
func (pm *PerformanceMonitor) GetTopOperations(n int, sortBy string) []OperationSummary {
	snapshot := pm.GetPerformanceSnapshot()

	operations := make([]OperationSummary, 0, len(snapshot.Operations))
	for _, op := range snapshot.Operations {
		operations = append(operations, op)
	}

	// Sort by specified metric
	for i := 0; i < len(operations); i++ {
		for j := i + 1; j < len(operations); j++ {
			var swap bool
			switch sortBy {
			case "count":
				swap = operations[i].Count < operations[j].Count
			case "error_rate":
				swap = operations[i].ErrorRate < operations[j].ErrorRate
			case "avg_duration":
				swap = operations[i].AvgDuration < operations[j].AvgDuration
			case "max_duration":
				swap = operations[i].MaxDuration < operations[j].MaxDuration
			case "throughput":
				swap = operations[i].ThroughputRPS < operations[j].ThroughputRPS
			default:
				swap = operations[i].Count < operations[j].Count
			}

			if swap {
				operations[i], operations[j] = operations[j], operations[i]
			}
		}
	}

	if n > len(operations) {
		n = len(operations)
	}

	return operations[:n]
}

// ResetMetrics resets all performance metrics
func (pm *PerformanceMonitor) ResetMetrics() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.operationMetrics = make(map[string]*OperationMetrics)
	pm.resourceMetrics = &ResourceMetrics{}

	pm.logger.Info(context.Background(), "Performance metrics reset")
}

// SetSampleRate sets the sampling rate for performance tracking
func (pm *PerformanceMonitor) SetSampleRate(rate float64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}

	pm.sampleRate = rate
	pm.logger.Info(context.Background(), "Performance sampling rate updated", "rate", rate)
}

// IsEnabled returns whether performance tracking is enabled
func (pm *PerformanceMonitor) IsEnabled() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.trackingEnabled
}

// SetEnabled enables or disables performance tracking
func (pm *PerformanceMonitor) SetEnabled(enabled bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.trackingEnabled = enabled
	pm.logger.Info(context.Background(), "Performance tracking updated", "enabled", enabled)
}

// StartBackgroundUpdates starts background resource metric updates
func (pm *PerformanceMonitor) StartBackgroundUpdates(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			pm.UpdateResourceMetrics()
		}
	}()

	pm.logger.Info(
		context.Background(),
		"Background performance updates started",
		"interval",
		interval,
	)
}

// PerformanceMiddleware creates HTTP middleware for performance tracking
func (pm *PerformanceMonitor) PerformanceMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			operationName := fmt.Sprintf("http_%s_%s", r.Method, r.URL.Path)

			err := pm.TrackOperationWithContext(
				r.Context(),
				operationName,
				func(ctx context.Context) error {
					next.ServeHTTP(w, r.WithContext(ctx))
					return nil
				},
			)

			if err != nil {
				pm.logger.Error(r.Context(), err, "HTTP request performance tracking failed")
			}
		})
	}
}
