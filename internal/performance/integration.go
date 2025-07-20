//go:build performance

package performance

import (
	"context"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/server"
)

// PerformanceIntegration provides integration points for the performance monitor
// with other Templar subsystems
type PerformanceIntegration struct {
	monitor *PerformanceMonitor
}

// NewPerformanceIntegration creates a new performance integration
func NewPerformanceIntegration(monitor *PerformanceMonitor) *PerformanceIntegration {
	return &PerformanceIntegration{
		monitor: monitor,
	}
}

// WrapBuildPipeline wraps a build pipeline to collect performance metrics
func (pi *PerformanceIntegration) WrapBuildPipeline(pipeline *build.BuildPipeline) *build.BuildPipeline {
	// Create a wrapper that measures build times
	originalBuild := pipeline.Build
	
	pipeline.Build = func(ctx context.Context, component string) error {
		start := time.Now()
		
		err := originalBuild(ctx, component)
		
		duration := time.Since(start)
		pi.monitor.Record(Metric{
			Type:  MetricTypeBuildTime,
			Value: float64(duration.Milliseconds()),
			Unit:  "ms",
			Labels: map[string]string{
				"component": component,
				"success":   boolToString(err == nil),
			},
		})
		
		if err != nil {
			pi.monitor.Record(Metric{
				Type:  MetricTypeErrorRate,
				Value: 1.0,
				Unit:  "count",
				Labels: map[string]string{
					"operation": "build",
					"component": component,
				},
			})
		}
		
		return err
	}
	
	return pipeline
}

// WrapComponentRegistry wraps a component registry to collect scan metrics
func (pi *PerformanceIntegration) WrapComponentRegistry(registry *registry.ComponentRegistry) *registry.ComponentRegistry {
	// Subscribe to registry events to collect component scan metrics
	// This would require modifying the registry to support event subscriptions
	// For now, we'll provide a method to manually record scan metrics
	return registry
}

// RecordComponentScan records metrics for component scanning operations
func (pi *PerformanceIntegration) RecordComponentScan(componentCount int, duration time.Duration, path string) {
	pi.monitor.Record(Metric{
		Type:  MetricTypeComponentScan,
		Value: float64(duration.Milliseconds()),
		Unit:  "ms",
		Labels: map[string]string{
			"path":            path,
			"component_count": string(rune(componentCount)),
		},
	})
	
	pi.monitor.Record(Metric{
		Type:  MetricTypeComponentScan,
		Value: float64(componentCount),
		Unit:  "count",
		Labels: map[string]string{
			"operation": "component_discovered",
			"path":      path,
		},
	})
}

// WrapServer wraps a server to collect request metrics
func (pi *PerformanceIntegration) WrapServer(srv *server.PreviewServer) *server.PreviewServer {
	// This would require modifying the server to support middleware
	// For now, we'll provide methods to manually record server metrics
	return srv
}

// RecordServerRequest records metrics for server requests
func (pi *PerformanceIntegration) RecordServerRequest(method, path string, duration time.Duration, statusCode int) {
	pi.monitor.Record(Metric{
		Type:  MetricTypeServerLatency,
		Value: float64(duration.Milliseconds()),
		Unit:  "ms",
		Labels: map[string]string{
			"method":      method,
			"path":        path,
			"status_code": string(rune(statusCode)),
		},
	})
	
	// Record error rate for non-2xx responses
	if statusCode >= 400 {
		pi.monitor.Record(Metric{
			Type:  MetricTypeErrorRate,
			Value: 1.0,
			Unit:  "count",
			Labels: map[string]string{
				"operation":   "http_request",
				"method":      method,
				"path":        path,
				"status_code": string(rune(statusCode)),
			},
		})
	}
}

// RecordCacheOperation records cache hit/miss metrics
func (pi *PerformanceIntegration) RecordCacheOperation(operation string, hit bool, duration time.Duration) {
	hitRate := 0.0
	if hit {
		hitRate = 1.0
	}
	
	pi.monitor.Record(Metric{
		Type:  MetricTypeCacheHitRate,
		Value: hitRate,
		Unit:  "ratio",
		Labels: map[string]string{
			"operation": operation,
			"result":    boolToString(hit),
		},
	})
	
	pi.monitor.Record(Metric{
		Type:  MetricTypeServerLatency, // Reuse for cache latency
		Value: float64(duration.Nanoseconds()),
		Unit:  "ns",
		Labels: map[string]string{
			"operation": "cache_" + operation,
			"result":    boolToString(hit),
		},
	})
}

// RecordFileWatcher records file watcher metrics
func (pi *PerformanceIntegration) RecordFileWatcher(watchedFiles int, eventCount int, processingDuration time.Duration) {
	pi.monitor.Record(Metric{
		Type:  MetricTypeFileWatchers,
		Value: float64(watchedFiles),
		Unit:  "count",
		Labels: map[string]string{
			"type": "watched_files",
		},
	})
	
	pi.monitor.Record(Metric{
		Type:  MetricTypeFileWatchers,
		Value: float64(eventCount),
		Unit:  "count",
		Labels: map[string]string{
			"type": "events_processed",
		},
	})
	
	if processingDuration > 0 {
		pi.monitor.Record(Metric{
			Type:  MetricTypeServerLatency, // Reuse for file processing latency
			Value: float64(processingDuration.Milliseconds()),
			Unit:  "ms",
			Labels: map[string]string{
				"operation": "file_event_processing",
			},
		})
	}
}

// GetPerformanceReport generates a comprehensive performance report
func (pi *PerformanceIntegration) GetPerformanceReport(since time.Time) PerformanceReport {
	report := PerformanceReport{
		GeneratedAt: time.Now(),
		TimeRange: TimeRange{
			Start: since,
			End:   time.Now(),
		},
		Metrics: make(map[MetricType]MetricSummary),
	}
	
	// Collect metrics for each type
	metricTypes := []MetricType{
		MetricTypeBuildTime,
		MetricTypeMemoryUsage,
		MetricTypeCPUUsage,
		MetricTypeGoroutines,
		MetricTypeFileWatchers,
		MetricTypeComponentScan,
		MetricTypeServerLatency,
		MetricTypeCacheHitRate,
		MetricTypeErrorRate,
	}
	
	for _, metricType := range metricTypes {
		metrics := pi.monitor.GetMetrics(metricType, since)
		aggregate := pi.monitor.GetAggregate(metricType)
		
		summary := MetricSummary{
			Type:        metricType,
			Count:       len(metrics),
			RecentValue: 0,
		}
		
		if len(metrics) > 0 {
			summary.RecentValue = metrics[len(metrics)-1].Value
		}
		
		if aggregate != nil {
			summary.Average = aggregate.Avg
			summary.Min = aggregate.Min
			summary.Max = aggregate.Max
			summary.P95 = aggregate.P95
			summary.P99 = aggregate.P99
		}
		
		report.Metrics[metricType] = summary
	}
	
	// Get recent recommendations
	recommendations := []Recommendation{}
	timeout := time.After(100 * time.Millisecond)
	
	for {
		select {
		case rec := <-pi.monitor.GetRecommendations():
			if rec.CreatedAt.After(since) {
				recommendations = append(recommendations, rec)
			}
		case <-timeout:
			goto done
		}
	}
	
done:
	report.Recommendations = recommendations
	
	return report
}

// PerformanceReport represents a comprehensive performance report
type PerformanceReport struct {
	GeneratedAt     time.Time                  `json:"generated_at"`
	TimeRange       TimeRange                  `json:"time_range"`
	Metrics         map[MetricType]MetricSummary `json:"metrics"`
	Recommendations []Recommendation           `json:"recommendations"`
}

// TimeRange represents a time range for reporting
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricSummary provides a summary of metrics for a specific type
type MetricSummary struct {
	Type        MetricType `json:"type"`
	Count       int        `json:"count"`
	RecentValue float64    `json:"recent_value"`
	Average     float64    `json:"average"`
	Min         float64    `json:"min"`
	Max         float64    `json:"max"`
	P95         float64    `json:"p95"`
	P99         float64    `json:"p99"`
}

// ApplyRecommendation applies a performance recommendation
func (pi *PerformanceIntegration) ApplyRecommendation(recommendation Recommendation) error {
	switch recommendation.Action.Type {
	case ActionScaleWorkers:
		return pi.applyWorkerScaling(recommendation.Action.Parameters)
	case ActionAdjustCacheSize:
		return pi.applyCacheAdjustment(recommendation.Action.Parameters)
	case ActionOptimizeGC:
		return pi.applyGCOptimization(recommendation.Action.Parameters)
	case ActionReducePolling:
		return pi.applyPollingReduction(recommendation.Action.Parameters)
	case ActionIncreasePolling:
		return pi.applyPollingIncrease(recommendation.Action.Parameters)
	case ActionClearCache:
		return pi.applyCacheClear(recommendation.Action.Parameters)
	case ActionRestartComponent:
		return pi.applyComponentRestart(recommendation.Action.Parameters)
	default:
		return nil // Unknown action type, ignore
	}
}

// Helper functions for applying recommendations
func (pi *PerformanceIntegration) applyWorkerScaling(params map[string]interface{}) error {
	// This would integrate with the build pipeline to scale workers
	// Implementation depends on the build system architecture
	return nil
}

func (pi *PerformanceIntegration) applyCacheAdjustment(params map[string]interface{}) error {
	// This would integrate with cache systems to adjust size
	// Implementation depends on the cache architecture
	return nil
}

func (pi *PerformanceIntegration) applyGCOptimization(params map[string]interface{}) error {
	// This would adjust Go GC settings
	// Implementation would modify runtime settings
	return nil
}

func (pi *PerformanceIntegration) applyPollingReduction(params map[string]interface{}) error {
	// This would reduce file watcher polling frequency
	// Implementation depends on file watcher architecture
	return nil
}

func (pi *PerformanceIntegration) applyPollingIncrease(params map[string]interface{}) error {
	// This would increase file watcher polling frequency
	// Implementation depends on file watcher architecture
	return nil
}

func (pi *PerformanceIntegration) applyCacheClear(params map[string]interface{}) error {
	// This would clear various caches
	// Implementation depends on cache systems
	return nil
}

func (pi *PerformanceIntegration) applyComponentRestart(params map[string]interface{}) error {
	// This would restart specific components
	// Implementation depends on component architecture
	return nil
}

// Helper function to convert bool to string
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}