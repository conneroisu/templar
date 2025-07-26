package performance

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// MetricType represents different types of performance metrics.
type MetricType string

const (
	MetricTypeBuildTime     MetricType = "build_time"
	MetricTypeMemoryUsage   MetricType = "memory_usage"
	MetricTypeCPUUsage      MetricType = "cpu_usage"
	MetricTypeGoroutines    MetricType = "goroutines"
	MetricTypeFileWatchers  MetricType = "file_watchers"
	MetricTypeComponentScan MetricType = "component_scan"
	MetricTypeServerLatency MetricType = "server_latency"
	MetricTypeCacheHitRate  MetricType = "cache_hit_rate"
	MetricTypeErrorRate     MetricType = "error_rate"
)

// Metric represents a single performance measurement.
type Metric struct {
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
	Threshold float64           `json:"threshold,omitempty"`
}

// MetricCollector collects and stores performance metrics.
type MetricCollector struct {
	metrics     []Metric
	maxMetrics  int
	mu          sync.RWMutex
	aggregates  map[MetricType]*MetricAggregate
	subscribers []chan<- Metric
}

// MetricAggregate stores aggregated metric data with efficient percentile calculation.
type MetricAggregate struct {
	Count          int64                 `json:"count"`
	Sum            float64               `json:"sum"`
	Min            float64               `json:"min"`
	Max            float64               `json:"max"`
	Avg            float64               `json:"avg"`
	P95            float64               `json:"p95"`
	P99            float64               `json:"p99"`
	percentileCalc *PercentileCalculator // Efficient O(log n) percentile calculation
	maxSize        int
}

// PerformanceMonitor monitors system performance and provides adaptive optimizations.
type PerformanceMonitor struct {
	collector         *MetricCollector
	lockFreeCollector *LockFreeMetricCollector // High-performance lock-free alternative
	ctx               context.Context
	cancel            context.CancelFunc
	interval          time.Duration
	thresholds        map[MetricType]float64
	adaptiveConfig    *AdaptiveConfig
	recommendations   chan Recommendation
	mu                sync.RWMutex
	useLockFree       bool // Enable/disable lock-free mode
}

// AdaptiveConfig contains configuration for adaptive optimization.
type AdaptiveConfig struct {
	EnableAutoOptimization bool                   `json:"enable_auto_optimization"`
	OptimizationRules      []OptimizationRule     `json:"optimization_rules"`
	ResourceLimits         ResourceLimits         `json:"resource_limits"`
	AlertThresholds        map[MetricType]float64 `json:"alert_thresholds"`
	SamplingRates          map[MetricType]float64 `json:"sampling_rates"`
}

// OptimizationRule defines when and how to optimize system performance.
type OptimizationRule struct {
	Name        string    `json:"name"`
	Condition   Condition `json:"condition"`
	Action      Action    `json:"action"`
	Priority    int       `json:"priority"`
	CooldownMin int       `json:"cooldown_minutes"`
}

// Condition defines when an optimization should be triggered.
type Condition struct {
	MetricType MetricType    `json:"metric_type"`
	Operator   string        `json:"operator"` // >, <, >=, <=, ==
	Threshold  float64       `json:"threshold"`
	Duration   time.Duration `json:"duration"` // How long condition must be true
}

// Action defines what optimization to perform.
type Action struct {
	Type       ActionType             `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ActionType represents different optimization actions.
type ActionType string

const (
	ActionScaleWorkers     ActionType = "scale_workers"
	ActionAdjustCacheSize  ActionType = "adjust_cache_size"
	ActionOptimizeGC       ActionType = "optimize_gc"
	ActionReducePolling    ActionType = "reduce_polling"
	ActionIncreasePolling  ActionType = "increase_polling"
	ActionClearCache       ActionType = "clear_cache"
	ActionRestartComponent ActionType = "restart_component"
)

// ResourceLimits defines system resource limits.
type ResourceLimits struct {
	MaxMemoryMB    int `json:"max_memory_mb"`
	MaxGoroutines  int `json:"max_goroutines"`
	MaxFileHandles int `json:"max_file_handles"`
	MaxWorkers     int `json:"max_workers"`
	MaxCacheSizeMB int `json:"max_cache_size_mb"`
}

// Recommendation represents a performance optimization recommendation.
type Recommendation struct {
	Type        string    `json:"type"`
	Priority    int       `json:"priority"`
	Description string    `json:"description"`
	Action      Action    `json:"action"`
	Impact      string    `json:"impact"`
	Confidence  float64   `json:"confidence"`
	Metrics     []Metric  `json:"metrics"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewMetricCollector creates a new metric collector.
func NewMetricCollector(maxMetrics int) *MetricCollector {
	return &MetricCollector{
		metrics:     make([]Metric, 0, maxMetrics),
		maxMetrics:  maxMetrics,
		aggregates:  make(map[MetricType]*MetricAggregate),
		subscribers: make([]chan<- Metric, 0),
	}
}

// Record records a new metric.
func (mc *MetricCollector) Record(metric Metric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Add timestamp if not set
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	// Add to metrics list (with rotation)
	if len(mc.metrics) >= mc.maxMetrics {
		// Remove oldest metric (ring buffer behavior)
		copy(mc.metrics, mc.metrics[1:])
		mc.metrics[len(mc.metrics)-1] = metric
	} else {
		mc.metrics = append(mc.metrics, metric)
	}

	// Update aggregates
	mc.updateAggregate(metric)

	// Notify subscribers
	for _, subscriber := range mc.subscribers {
		select {
		case subscriber <- metric:
		default:
			// Don't block if subscriber can't keep up
		}
	}
}

// Subscribe subscribes to metric updates.
func (mc *MetricCollector) Subscribe() <-chan Metric {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	ch := make(chan Metric, 100) // Buffered channel
	mc.subscribers = append(mc.subscribers, ch)

	return ch
}

// GetMetrics returns all metrics within the time range.
func (mc *MetricCollector) GetMetrics(metricType MetricType, since time.Time) []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var result []Metric
	for _, metric := range mc.metrics {
		if (metricType == "" || metric.Type == metricType) && metric.Timestamp.After(since) {
			result = append(result, metric)
		}
	}

	return result
}

// GetAggregate returns aggregated data for a metric type.
func (mc *MetricCollector) GetAggregate(metricType MetricType) *MetricAggregate {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if agg, exists := mc.aggregates[metricType]; exists {
		// Return a copy to avoid race conditions
		return &MetricAggregate{
			Count:          agg.Count,
			Sum:            agg.Sum,
			Min:            agg.Min,
			Max:            agg.Max,
			Avg:            agg.Avg,
			P95:            agg.P95,
			P99:            agg.P99,
			percentileCalc: nil, // Don't copy percentile calculator for performance/safety
			maxSize:        agg.maxSize,
		}
	}

	return nil
}

// updateAggregate updates aggregate statistics for a metric.
func (mc *MetricCollector) updateAggregate(metric Metric) {
	agg, exists := mc.aggregates[metric.Type]
	if !exists {
		agg = &MetricAggregate{
			Min:            metric.Value,
			Max:            metric.Value,
			percentileCalc: NewPercentileCalculator(1000), // Efficient percentile calculation
			maxSize:        1000,
		}
		mc.aggregates[metric.Type] = agg
	}

	// Update basic stats
	agg.Count++
	agg.Sum += metric.Value
	agg.Avg = agg.Sum / float64(agg.Count)

	if metric.Value < agg.Min {
		agg.Min = metric.Value
	}
	if metric.Value > agg.Max {
		agg.Max = metric.Value
	}

	// Update percentiles using efficient O(log n) skip list
	agg.percentileCalc.AddValue(metric.Value)
	agg.P95 = agg.percentileCalc.GetP95()
	agg.P99 = agg.percentileCalc.GetP99()
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor(interval time.Duration) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	collector := NewMetricCollector(10000)                 // Keep last 10k metrics
	lockFreeCollector := NewLockFreeMetricCollector(10000) // Lock-free alternative

	monitor := &PerformanceMonitor{
		collector:         collector,
		lockFreeCollector: lockFreeCollector,
		ctx:               ctx,
		cancel:            cancel,
		interval:          interval,
		thresholds:        getDefaultThresholds(),
		adaptiveConfig:    getDefaultAdaptiveConfig(),
		recommendations:   make(chan Recommendation, 100),
		useLockFree:       true, // Enable lock-free mode by default for better performance
	}

	return monitor
}

// Start starts the performance monitoring.
func (pm *PerformanceMonitor) Start() {
	go pm.collectSystemMetrics()
	go pm.analyzeMetrics()
}

// Stop stops the performance monitoring.
func (pm *PerformanceMonitor) Stop() {
	pm.cancel()
}

// Record records a metric.
func (pm *PerformanceMonitor) Record(metric Metric) {
	if pm.useLockFree {
		pm.lockFreeCollector.Record(metric)
	} else {
		pm.collector.Record(metric)
	}
}

// GetRecommendations returns the recommendations channel.
func (pm *PerformanceMonitor) GetRecommendations() <-chan Recommendation {
	return pm.recommendations
}

// GetMetrics returns metrics for a specific type and time range.
func (pm *PerformanceMonitor) GetMetrics(metricType MetricType, since time.Time) []Metric {
	if pm.useLockFree {
		return pm.lockFreeCollector.GetMetrics(metricType, since)
	} else {
		return pm.collector.GetMetrics(metricType, since)
	}
}

// GetAggregate returns aggregated metrics.
func (pm *PerformanceMonitor) GetAggregate(metricType MetricType) *MetricAggregate {
	if pm.useLockFree {
		return pm.lockFreeCollector.GetAggregate(metricType)
	} else {
		return pm.collector.GetAggregate(metricType)
	}
}

// SetLockFree enables or disables lock-free mode.
func (pm *PerformanceMonitor) SetLockFree(enabled bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.useLockFree = enabled
}

// IsLockFreeEnabled returns whether lock-free mode is enabled.
func (pm *PerformanceMonitor) IsLockFreeEnabled() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.useLockFree
}

// collectSystemMetrics collects system-level performance metrics.
func (pm *PerformanceMonitor) collectSystemMetrics() {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectMemoryMetrics()
			pm.collectGoroutineMetrics()
			pm.collectCPUMetrics()
		}
	}
}

// collectMemoryMetrics collects memory usage metrics.
func (pm *PerformanceMonitor) collectMemoryMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Record various memory metrics
	pm.Record(Metric{
		Type:   MetricTypeMemoryUsage,
		Value:  float64(m.Alloc),
		Unit:   "bytes",
		Labels: map[string]string{"component": "heap_alloc"},
	})

	pm.Record(Metric{
		Type:   MetricTypeMemoryUsage,
		Value:  float64(m.Sys),
		Unit:   "bytes",
		Labels: map[string]string{"component": "sys_total"},
	})

	pm.Record(Metric{
		Type:   MetricTypeMemoryUsage,
		Value:  float64(m.NumGC),
		Unit:   "count",
		Labels: map[string]string{"component": "gc_cycles"},
	})
}

// collectGoroutineMetrics collects goroutine metrics.
func (pm *PerformanceMonitor) collectGoroutineMetrics() {
	numGoroutines := runtime.NumGoroutine()

	pm.Record(Metric{
		Type:  MetricTypeGoroutines,
		Value: float64(numGoroutines),
		Unit:  "count",
	})
}

// collectCPUMetrics collects CPU usage metrics (simplified).
func (pm *PerformanceMonitor) collectCPUMetrics() {
	numCPU := runtime.NumCPU()

	pm.Record(Metric{
		Type:   MetricTypeCPUUsage,
		Value:  float64(numCPU),
		Unit:   "cores",
		Labels: map[string]string{"component": "available_cores"},
	})
}

// analyzeMetrics analyzes collected metrics and generates recommendations.
func (pm *PerformanceMonitor) analyzeMetrics() {
	ticker := time.NewTicker(pm.interval * 5) // Analyze every 5 intervals
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.generateRecommendations()
		}
	}
}

// generateRecommendations analyzes metrics and generates optimization recommendations.
func (pm *PerformanceMonitor) generateRecommendations() {
	pm.mu.RLock()
	config := pm.adaptiveConfig
	pm.mu.RUnlock()

	if !config.EnableAutoOptimization {
		return
	}

	// Analyze memory usage
	if memAgg := pm.GetAggregate(MetricTypeMemoryUsage); memAgg != nil {
		memoryThreshold := float64(config.ResourceLimits.MaxMemoryMB) * 1024 * 1024 * 0.8
		if memAgg.Avg > memoryThreshold {
			recommendation := Recommendation{
				Type:        "memory_optimization",
				Priority:    1,
				Description: "High memory usage detected. Consider reducing cache size or scaling workers.",
				Action: Action{
					Type: ActionAdjustCacheSize,
					Parameters: map[string]interface{}{
						"reduce_by_percent": 20,
					},
				},
				Impact:     "Reduce memory usage by ~20%",
				Confidence: 0.85,
				CreatedAt:  time.Now(),
			}

			select {
			case pm.recommendations <- recommendation:
			default:
				// Channel full, skip recommendation
			}
		}
	}

	// Analyze goroutine count
	if goroutineAgg := pm.GetAggregate(MetricTypeGoroutines); goroutineAgg != nil {
		goroutineThreshold := float64(config.ResourceLimits.MaxGoroutines) * 0.9
		if goroutineAgg.Avg > goroutineThreshold {
			recommendation := Recommendation{
				Type:        "goroutine_optimization",
				Priority:    2,
				Description: "High goroutine count detected. Consider optimizing concurrent operations.",
				Action: Action{
					Type: ActionScaleWorkers,
					Parameters: map[string]interface{}{
						"target_workers": int(goroutineAgg.Avg * 0.8),
					},
				},
				Impact:     "Reduce goroutine count by ~20%",
				Confidence: 0.75,
				CreatedAt:  time.Now(),
			}

			select {
			case pm.recommendations <- recommendation:
			default:
			}
		}
	}

	// Analyze build times
	if buildAgg := pm.GetAggregate(MetricTypeBuildTime); buildAgg != nil {
		if buildAgg.P95 > 5000 { // 5 seconds
			recommendation := Recommendation{
				Type:        "build_optimization",
				Priority:    3,
				Description: "Slow build times detected. Consider optimizing build pipeline.",
				Action: Action{
					Type: ActionScaleWorkers,
					Parameters: map[string]interface{}{
						"increase_workers": true,
						"target_workers":   pm.calculateOptimalWorkers(),
					},
				},
				Impact:     "Reduce build times by ~30%",
				Confidence: 0.70,
				CreatedAt:  time.Now(),
			}

			select {
			case pm.recommendations <- recommendation:
			default:
			}
		}
	}
}

// calculateOptimalWorkers calculates the optimal number of workers based on current metrics.
func (pm *PerformanceMonitor) calculateOptimalWorkers() int {
	cpuCores := runtime.NumCPU()
	currentGoroutines := runtime.NumGoroutine()

	// Simple heuristic: optimal workers = CPU cores * 2, but don't exceed current goroutines
	optimal := cpuCores * 2
	if optimal > currentGoroutines {
		optimal = currentGoroutines
	}

	// Don't go below 1 or above resource limits
	if optimal < 1 {
		optimal = 1
	}
	if optimal > pm.adaptiveConfig.ResourceLimits.MaxWorkers {
		optimal = pm.adaptiveConfig.ResourceLimits.MaxWorkers
	}

	return optimal
}

// getDefaultThresholds returns default performance thresholds.
func getDefaultThresholds() map[MetricType]float64 {
	return map[MetricType]float64{
		MetricTypeBuildTime:     5000,              // 5 seconds
		MetricTypeMemoryUsage:   512 * 1024 * 1024, // 512MB
		MetricTypeGoroutines:    1000,              // 1000 goroutines
		MetricTypeServerLatency: 100,               // 100ms
		MetricTypeCacheHitRate:  0.8,               // 80%
		MetricTypeErrorRate:     0.05,              // 5%
	}
}

// getDefaultAdaptiveConfig returns default adaptive configuration.
func getDefaultAdaptiveConfig() *AdaptiveConfig {
	return &AdaptiveConfig{
		EnableAutoOptimization: true,
		ResourceLimits: ResourceLimits{
			MaxMemoryMB:    1024, // 1GB
			MaxGoroutines:  2000,
			MaxFileHandles: 1000,
			MaxWorkers:     runtime.NumCPU() * 4,
			MaxCacheSizeMB: 256, // 256MB
		},
		AlertThresholds: map[MetricType]float64{
			MetricTypeMemoryUsage: 800 * 1024 * 1024, // 800MB
			MetricTypeGoroutines:  1500,
			MetricTypeBuildTime:   10000, // 10 seconds
		},
		SamplingRates: map[MetricType]float64{
			MetricTypeMemoryUsage:   1.0, // Sample every metric
			MetricTypeGoroutines:    1.0,
			MetricTypeBuildTime:     1.0,
			MetricTypeServerLatency: 0.1, // Sample 10% of requests
		},
	}
}
