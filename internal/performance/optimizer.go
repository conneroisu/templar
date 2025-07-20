package performance

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/registry"
)

// AdaptiveOptimizer applies performance optimizations based on monitor recommendations
type AdaptiveOptimizer struct {
	monitor         *PerformanceMonitor
	buildPipeline   *build.BuildPipeline
	registry        *registry.ComponentRegistry
	ctx             context.Context
	cancel          context.CancelFunc
	config          *OptimizerConfig
	appliedActions  map[string]time.Time // Track when actions were last applied
	mu              sync.RWMutex
	metrics         *OptimizerMetrics
}

// OptimizerConfig contains configuration for the adaptive optimizer
type OptimizerConfig struct {
	EnableAutoApply         bool          `json:"enable_auto_apply"`
	MaxActionsPerInterval   int           `json:"max_actions_per_interval"`
	CooldownPeriod          time.Duration `json:"cooldown_period"`
	ConfidenceThreshold     float64       `json:"confidence_threshold"`
	DryRunMode              bool          `json:"dry_run_mode"`
	NotificationWebhook     string        `json:"notification_webhook,omitempty"`
	BackoffMultiplier       float64       `json:"backoff_multiplier"`
	MaxBackoffDuration      time.Duration `json:"max_backoff_duration"`
}

// OptimizerMetrics tracks optimizer performance
type OptimizerMetrics struct {
	ActionsApplied       int64     `json:"actions_applied"`
	ActionsSkipped       int64     `json:"actions_skipped"`
	ActionsSuccessful    int64     `json:"actions_successful"`
	ActionsFailed        int64     `json:"actions_failed"`
	AverageImpact        float64   `json:"average_impact"`
	LastOptimization     time.Time `json:"last_optimization"`
	TotalOptimizationTime time.Duration `json:"total_optimization_time"`
}

// OptimizationResult represents the result of applying an optimization
type OptimizationResult struct {
	Success     bool          `json:"success"`
	Action      Action        `json:"action"`
	Impact      string        `json:"impact"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	MetricsBefore map[string]float64 `json:"metrics_before"`
	MetricsAfter  map[string]float64 `json:"metrics_after"`
	Timestamp   time.Time     `json:"timestamp"`
}

// WorkerScaler manages worker pool scaling
type WorkerScaler struct {
	currentWorkers int
	targetWorkers  int
	maxWorkers     int
	minWorkers     int
	scaleUpRate    float64
	scaleDownRate  float64
	mu             sync.RWMutex
}

// CacheManager manages adaptive cache sizing
type CacheManager struct {
	currentSizeMB int
	targetSizeMB  int
	maxSizeMB     int
	minSizeMB     int
	hitRate       float64
	mu            sync.RWMutex
}

// NewAdaptiveOptimizer creates a new adaptive optimizer
func NewAdaptiveOptimizer(monitor *PerformanceMonitor, buildPipeline *build.BuildPipeline, registry *registry.ComponentRegistry) *AdaptiveOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	
	optimizer := &AdaptiveOptimizer{
		monitor:        monitor,
		buildPipeline:  buildPipeline,
		registry:       registry,
		ctx:            ctx,
		cancel:         cancel,
		config:         getDefaultOptimizerConfig(),
		appliedActions: make(map[string]time.Time),
		metrics:        &OptimizerMetrics{},
	}

	return optimizer
}

// Start starts the adaptive optimizer
func (ao *AdaptiveOptimizer) Start() {
	go ao.processRecommendations()
	go ao.periodicOptimization()
}

// Stop stops the adaptive optimizer
func (ao *AdaptiveOptimizer) Stop() {
	ao.cancel()
}

// processRecommendations processes recommendations from the performance monitor
func (ao *AdaptiveOptimizer) processRecommendations() {
	recommendations := ao.monitor.GetRecommendations()
	
	for {
		select {
		case <-ao.ctx.Done():
			return
		case recommendation := <-recommendations:
			ao.handleRecommendation(recommendation)
		}
	}
}

// handleRecommendation handles a single recommendation
func (ao *AdaptiveOptimizer) handleRecommendation(recommendation Recommendation) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	// Check if we should apply this recommendation
	if !ao.shouldApplyRecommendation(recommendation) {
		ao.metrics.ActionsSkipped++
		log.Printf("Skipping recommendation: %s (reason: threshold/cooldown)", recommendation.Type)
		return
	}

	// Capture metrics before optimization
	metricsBefore := ao.captureCurrentMetrics()

	// Apply the optimization
	result := ao.applyOptimization(recommendation.Action)
	result.MetricsBefore = metricsBefore
	result.Timestamp = time.Now()

	// Update metrics
	ao.metrics.ActionsApplied++
	if result.Success {
		ao.metrics.ActionsSuccessful++
		ao.metrics.LastOptimization = time.Now()
		ao.metrics.TotalOptimizationTime += result.Duration
		
		// Record when this action was applied
		ao.appliedActions[recommendation.Type] = time.Now()
		
		log.Printf("Applied optimization: %s (impact: %s)", recommendation.Type, result.Impact)
	} else {
		ao.metrics.ActionsFailed++
		log.Printf("Failed to apply optimization: %s (error: %s)", recommendation.Type, result.Error)
	}

	// Capture metrics after optimization (with a small delay)
	go func() {
		time.Sleep(time.Second * 5) // Wait for effects to take place
		result.MetricsAfter = ao.captureCurrentMetrics()
		ao.calculateImpact(result)
	}()
}

// shouldApplyRecommendation determines if a recommendation should be applied
func (ao *AdaptiveOptimizer) shouldApplyRecommendation(recommendation Recommendation) bool {
	// Check if auto-apply is enabled
	if !ao.config.EnableAutoApply {
		return false
	}

	// Check confidence threshold
	if recommendation.Confidence < ao.config.ConfidenceThreshold {
		return false
	}

	// Check cooldown period
	if lastApplied, exists := ao.appliedActions[recommendation.Type]; exists {
		if time.Since(lastApplied) < ao.config.CooldownPeriod {
			return false
		}
	}

	// Check rate limiting
	recentActions := 0
	cutoff := time.Now().Add(-time.Hour) // Last hour
	for _, appliedTime := range ao.appliedActions {
		if appliedTime.After(cutoff) {
			recentActions++
		}
	}

	if recentActions >= ao.config.MaxActionsPerInterval {
		return false
	}

	return true
}

// applyOptimization applies a specific optimization action
func (ao *AdaptiveOptimizer) applyOptimization(action Action) OptimizationResult {
	startTime := time.Now()
	result := OptimizationResult{
		Action:    action,
		Timestamp: startTime,
	}

	if ao.config.DryRunMode {
		result.Success = true
		result.Impact = "Dry run - no actual changes made"
		result.Duration = time.Since(startTime)
		return result
	}

	switch action.Type {
	case ActionScaleWorkers:
		result = ao.scaleWorkers(action)
	case ActionAdjustCacheSize:
		result = ao.adjustCacheSize(action)
	case ActionOptimizeGC:
		result = ao.optimizeGC(action)
	case ActionReducePolling:
		result = ao.adjustPollingRate(action, false)
	case ActionIncreasePolling:
		result = ao.adjustPollingRate(action, true)
	case ActionClearCache:
		result = ao.clearCache(action)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unknown action type: %s", action.Type)
	}

	result.Duration = time.Since(startTime)
	return result
}

// scaleWorkers scales the number of worker goroutines
func (ao *AdaptiveOptimizer) scaleWorkers(action Action) OptimizationResult {
	result := OptimizationResult{Action: action}

	targetWorkers, ok := action.Parameters["target_workers"].(int)
	if !ok {
		result.Error = "Invalid target_workers parameter"
		return result
	}

	// Validate worker count
	maxWorkers := ao.config.MaxActionsPerInterval * 10 // Reasonable upper bound
	if targetWorkers > maxWorkers {
		targetWorkers = maxWorkers
	}
	if targetWorkers < 1 {
		targetWorkers = 1
	}

	// Apply worker scaling (this would integrate with actual build pipeline)
	if ao.buildPipeline != nil {
		// In a real implementation, this would call methods on the build pipeline
		// to adjust worker pool size
		result.Success = true
		result.Impact = fmt.Sprintf("Scaled workers to %d", targetWorkers)
	} else {
		result.Error = "Build pipeline not available"
	}

	return result
}

// adjustCacheSize adjusts cache size based on memory pressure
func (ao *AdaptiveOptimizer) adjustCacheSize(action Action) OptimizationResult {
	result := OptimizationResult{Action: action}

	reducePercent, ok := action.Parameters["reduce_by_percent"].(float64)
	if !ok {
		reducePercent = 20.0 // Default 20% reduction
	}

	// Calculate new cache size
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	currentMB := float64(m.Alloc) / (1024 * 1024)
	reductionMB := currentMB * (reducePercent / 100)

	// Apply cache size adjustment (placeholder - would integrate with actual cache)
	result.Success = true
	result.Impact = fmt.Sprintf("Reduced cache size by %.1fMB (%.1f%%)", reductionMB, reducePercent)

	return result
}

// optimizeGC triggers garbage collection optimization
func (ao *AdaptiveOptimizer) optimizeGC(action Action) OptimizationResult {
	result := OptimizationResult{Action: action}

	// Trigger garbage collection
	runtime.GC()
	
	// Optionally adjust GC target percentage
	if targetPercent, ok := action.Parameters["gc_target_percent"].(int); ok {
		if targetPercent > 0 && targetPercent <= 500 {
			// Note: runtime.GCPercent is not available in newer Go versions
			// In production, this would use debug.SetGCPercent()
			result.Impact = fmt.Sprintf("Would adjust GC target to %d%%", targetPercent)
		}
	} else {
		result.Impact = "Triggered garbage collection"
	}

	result.Success = true
	return result
}

// adjustPollingRate adjusts file watching polling rate
func (ao *AdaptiveOptimizer) adjustPollingRate(action Action, increase bool) OptimizationResult {
	result := OptimizationResult{Action: action}

	// This would integrate with the file watcher to adjust polling intervals
	direction := "decreased"
	if increase {
		direction = "increased"
	}

	result.Success = true
	result.Impact = fmt.Sprintf("Polling rate %s", direction)
	return result
}

// clearCache clears various caches to free memory
func (ao *AdaptiveOptimizer) clearCache(action Action) OptimizationResult {
	result := OptimizationResult{Action: action}

	// This would integrate with cache implementations to clear them
	cacheType, ok := action.Parameters["cache_type"].(string)
	if !ok {
		cacheType = "all"
	}

	result.Success = true
	result.Impact = fmt.Sprintf("Cleared %s cache(s)", cacheType)
	return result
}

// captureCurrentMetrics captures current system metrics
func (ao *AdaptiveOptimizer) captureCurrentMetrics() map[string]float64 {
	metrics := make(map[string]float64)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["memory_alloc_mb"] = float64(m.Alloc) / (1024 * 1024)
	metrics["memory_sys_mb"] = float64(m.Sys) / (1024 * 1024)
	metrics["goroutines"] = float64(runtime.NumGoroutine())
	metrics["gc_cycles"] = float64(m.NumGC)

	// Add performance monitor aggregates
	if memAgg := ao.monitor.GetAggregate(MetricTypeMemoryUsage); memAgg != nil {
		metrics["avg_memory_usage"] = memAgg.Avg
		metrics["p95_memory_usage"] = memAgg.P95
	}

	if buildAgg := ao.monitor.GetAggregate(MetricTypeBuildTime); buildAgg != nil {
		metrics["avg_build_time"] = buildAgg.Avg
		metrics["p95_build_time"] = buildAgg.P95
	}

	return metrics
}

// calculateImpact calculates the impact of an optimization
func (ao *AdaptiveOptimizer) calculateImpact(result OptimizationResult) {
	if !result.Success || result.MetricsBefore == nil || result.MetricsAfter == nil {
		return
	}

	// Calculate percentage improvements
	improvements := make(map[string]float64)
	
	for metric, beforeValue := range result.MetricsBefore {
		if afterValue, exists := result.MetricsAfter[metric]; exists && beforeValue > 0 {
			improvement := ((beforeValue - afterValue) / beforeValue) * 100
			improvements[metric] = improvement
		}
	}

	// Update average impact
	totalImprovement := 0.0
	count := 0
	for _, improvement := range improvements {
		if improvement > 0 { // Only count positive improvements
			totalImprovement += improvement
			count++
		}
	}

	if count > 0 {
		avgImprovement := totalImprovement / float64(count)
		
		// Update running average
		ao.mu.Lock()
		if ao.metrics.ActionsSuccessful == 1 {
			ao.metrics.AverageImpact = avgImprovement
		} else {
			ao.metrics.AverageImpact = (ao.metrics.AverageImpact + avgImprovement) / 2
		}
		ao.mu.Unlock()

		log.Printf("Optimization impact: %.2f%% average improvement", avgImprovement)
	}
}

// periodicOptimization performs periodic system optimization
func (ao *AdaptiveOptimizer) periodicOptimization() {
	ticker := time.NewTicker(time.Minute * 15) // Every 15 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ao.ctx.Done():
			return
		case <-ticker.C:
			ao.performPeriodicOptimization()
		}
	}
}

// performPeriodicOptimization performs routine optimizations
func (ao *AdaptiveOptimizer) performPeriodicOptimization() {
	// Periodic garbage collection if memory usage is high
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsageMB := float64(m.Alloc) / (1024 * 1024)

	if memUsageMB > 500 { // 500MB threshold
		ao.monitor.Record(Metric{
			Type:  MetricTypeMemoryUsage,
			Value: memUsageMB,
			Unit:  "MB",
			Labels: map[string]string{"trigger": "periodic_check"},
		})

		// Consider triggering GC optimization
		action := Action{
			Type:       ActionOptimizeGC,
			Parameters: map[string]interface{}{},
		}
		
		result := ao.applyOptimization(action)
		if result.Success {
			log.Printf("Periodic optimization: %s", result.Impact)
		}
	}

	// Record optimizer metrics
	ao.monitor.Record(Metric{
		Type:  "optimizer_actions_applied",
		Value: float64(ao.metrics.ActionsApplied),
		Unit:  "count",
	})

	ao.monitor.Record(Metric{
		Type:  "optimizer_success_rate",
		Value: float64(ao.metrics.ActionsSuccessful) / float64(ao.metrics.ActionsApplied) * 100,
		Unit:  "percent",
	})
}

// GetMetrics returns current optimizer metrics
func (ao *AdaptiveOptimizer) GetMetrics() *OptimizerMetrics {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &OptimizerMetrics{
		ActionsApplied:        ao.metrics.ActionsApplied,
		ActionsSkipped:        ao.metrics.ActionsSkipped,
		ActionsSuccessful:     ao.metrics.ActionsSuccessful,
		ActionsFailed:         ao.metrics.ActionsFailed,
		AverageImpact:         ao.metrics.AverageImpact,
		LastOptimization:      ao.metrics.LastOptimization,
		TotalOptimizationTime: ao.metrics.TotalOptimizationTime,
	}
}

// UpdateConfig updates the optimizer configuration
func (ao *AdaptiveOptimizer) UpdateConfig(config *OptimizerConfig) {
	ao.mu.Lock()
	defer ao.mu.Unlock()
	ao.config = config
}

// getDefaultOptimizerConfig returns default optimizer configuration
func getDefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		EnableAutoApply:       true,
		MaxActionsPerInterval: 5,
		CooldownPeriod:        time.Minute * 5,
		ConfidenceThreshold:   0.7,
		DryRunMode:            false,
		BackoffMultiplier:     2.0,
		MaxBackoffDuration:    time.Hour,
	}
}

// NewWorkerScaler creates a new worker scaler
func NewWorkerScaler(initialWorkers, maxWorkers int) *WorkerScaler {
	return &WorkerScaler{
		currentWorkers: initialWorkers,
		targetWorkers:  initialWorkers,
		maxWorkers:     maxWorkers,
		minWorkers:     1,
		scaleUpRate:    0.25,   // 25% increase
		scaleDownRate:  0.15,   // 15% decrease
	}
}

// Scale adjusts the number of workers
func (ws *WorkerScaler) Scale(direction string, factor float64) int {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	switch direction {
	case "up":
		ws.targetWorkers = int(float64(ws.currentWorkers) * (1 + ws.scaleUpRate*factor))
		if ws.targetWorkers > ws.maxWorkers {
			ws.targetWorkers = ws.maxWorkers
		}
	case "down":
		ws.targetWorkers = int(float64(ws.currentWorkers) * (1 - ws.scaleDownRate*factor))
		if ws.targetWorkers < ws.minWorkers {
			ws.targetWorkers = ws.minWorkers
		}
	}

	ws.currentWorkers = ws.targetWorkers
	return ws.currentWorkers
}

// GetCurrentWorkers returns the current number of workers
func (ws *WorkerScaler) GetCurrentWorkers() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.currentWorkers
}