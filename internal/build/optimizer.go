package build

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// AssetOptimizer handles post-build optimization of assets.
type AssetOptimizer struct {
	config *config.Config
}

// OptimizerOptions configures asset optimization.
type OptimizerOptions struct {
	Images      bool `json:"images"`
	CSS         bool `json:"css"`
	JavaScript  bool `json:"javascript"`
	Compression bool `json:"compression"`
}

// NewAssetOptimizer creates a new asset optimizer.
func NewAssetOptimizer(cfg *config.Config) *AssetOptimizer {
	return &AssetOptimizer{config: cfg}
}

// Optimize applies optimizations to assets in the specified directory.
func (o *AssetOptimizer) Optimize(
	ctx context.Context,
	assetsDir string,
	options OptimizerOptions,
) error {
	if options.Images {
		if err := o.optimizeImages(ctx, assetsDir); err != nil {
			return fmt.Errorf("image optimization failed: %w", err)
		}
	}

	if options.CSS {
		if err := o.optimizeCSS(ctx, assetsDir); err != nil {
			return fmt.Errorf("CSS optimization failed: %w", err)
		}
	}

	if options.JavaScript {
		if err := o.optimizeJavaScript(ctx, assetsDir); err != nil {
			return fmt.Errorf("JavaScript optimization failed: %w", err)
		}
	}

	if options.Compression {
		if err := o.compressAssets(ctx, assetsDir); err != nil {
			return fmt.Errorf("asset compression failed: %w", err)
		}
	}

	return nil
}

func (o *AssetOptimizer) optimizeImages(ctx context.Context, assetsDir string) error {
	// Placeholder for image optimization
	return nil
}

func (o *AssetOptimizer) optimizeCSS(ctx context.Context, assetsDir string) error {
	// Placeholder for CSS optimization
	return nil
}

func (o *AssetOptimizer) optimizeJavaScript(ctx context.Context, assetsDir string) error {
	// Placeholder for JavaScript optimization
	return nil
}

func (o *AssetOptimizer) compressAssets(ctx context.Context, assetsDir string) error {
	// Placeholder for asset compression
	return nil
}

// OptimizationLevel represents the severity/priority of an optimization recommendation.
type OptimizationLevel int

const (
	// OptimizationInfo provides informational recommendations.
	OptimizationInfo OptimizationLevel = iota
	// OptimizationLow suggests minor improvements.
	OptimizationLow
	// OptimizationMedium indicates moderate performance impact.
	OptimizationMedium
	// OptimizationHigh suggests significant improvements needed.
	OptimizationHigh
	// OptimizationCritical indicates urgent performance issues.
	OptimizationCritical
)

// String returns a string representation of the optimization level.
func (ol OptimizationLevel) String() string {
	switch ol {
	case OptimizationInfo:
		return "Info"
	case OptimizationLow:
		return "Low"
	case OptimizationMedium:
		return "Medium"
	case OptimizationHigh:
		return "High"
	case OptimizationCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// OptimizationCategory represents different categories of build optimizations.
type OptimizationCategory int

const (
	// CategoryPerformance focuses on build speed and efficiency.
	CategoryPerformance OptimizationCategory = iota
	// CategoryReliability focuses on build stability and error reduction.
	CategoryReliability
	// CategoryResource focuses on memory and CPU utilization.
	CategoryResource
	// CategoryCaching focuses on build caching effectiveness.
	CategoryCaching
	// CategoryConcurrency focuses on parallel processing optimization.
	CategoryConcurrency
	// CategoryConfiguration focuses on build configuration improvements.
	CategoryConfiguration
)

// String returns a string representation of the optimization category.
func (oc OptimizationCategory) String() string {
	switch oc {
	case CategoryPerformance:
		return "Performance"
	case CategoryReliability:
		return "Reliability"
	case CategoryResource:
		return "Resource"
	case CategoryCaching:
		return "Caching"
	case CategoryConcurrency:
		return "Concurrency"
	case CategoryConfiguration:
		return "Configuration"
	default:
		return "Unknown"
	}
}

// OptimizationRecommendation represents a specific recommendation for build improvement.
type OptimizationRecommendation struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Level       OptimizationLevel    `json:"level"`
	Category    OptimizationCategory `json:"category"`
	Impact      string               `json:"impact"`
	Effort      string               `json:"effort"`
	Actions     []string             `json:"actions"`
	Metrics     map[string]float64   `json:"metrics,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
}

// BuildOptimizer analyzes build performance and provides optimization recommendations.
type BuildOptimizer struct {
	// Configuration
	config           *config.Config
	progressReporter *BuildProgressReporter
	
	// Analysis state
	recommendations     []OptimizationRecommendation
	recommendationMutex sync.RWMutex
	
	// Analysis history
	analysisHistory     []AnalysisResult
	historyMutex        sync.Mutex
	maxHistorySize      int
	
	// Thresholds for analysis
	slowBuildThreshold      time.Duration
	highFailureRateThreshold float64
	lowCacheHitThreshold     float64
	highMemoryThreshold      int64
}

// AnalysisResult represents the result of a build performance analysis.
type AnalysisResult struct {
	Timestamp       time.Time                    `json:"timestamp"`
	TotalBuilds     int64                        `json:"total_builds"`
	SuccessRate     float64                      `json:"success_rate"`
	AverageBuildTime time.Duration               `json:"average_build_time"`
	CacheHitRate    float64                      `json:"cache_hit_rate"`
	Recommendations []OptimizationRecommendation `json:"recommendations"`
	Score           float64                      `json:"score"` // Overall performance score 0-100
}

// NewBuildOptimizer creates a new build optimizer.
func NewBuildOptimizer(cfg *config.Config, progressReporter *BuildProgressReporter) *BuildOptimizer {
	return &BuildOptimizer{
		config:                   cfg,
		progressReporter:         progressReporter,
		recommendations:          make([]OptimizationRecommendation, 0),
		analysisHistory:          make([]AnalysisResult, 0, 50),
		maxHistorySize:           50,
		slowBuildThreshold:       10 * time.Second,
		highFailureRateThreshold: 0.05, // 5%
		lowCacheHitThreshold:     0.70, // 70%
		highMemoryThreshold:      512 * 1024 * 1024, // 512MB
	}
}

// AnalyzeBuildPerformance performs comprehensive build performance analysis.
func (bo *BuildOptimizer) AnalyzeBuildPerformance() AnalysisResult {
	stats := bo.progressReporter.GetBuildStats()
	
	// Extract key metrics
	totalBuilds := int64(0)
	if val, ok := stats["total_builds"].(int64); ok {
		totalBuilds = val
	}
	
	successRate := float64(0)
	if val, ok := stats["success_rate"].(float64); ok {
		successRate = val
	}
	
	averageBuildTime := time.Duration(0)
	if val, ok := stats["average_build_time"].(string); ok {
		if parsed, err := time.ParseDuration(val); err == nil {
			averageBuildTime = parsed
		}
	}
	
	// Analyze different aspects of build performance
	recommendations := make([]OptimizationRecommendation, 0)
	
	// Performance analysis
	recommendations = append(recommendations, bo.analyzePerformance(stats)...)
	
	// Reliability analysis
	recommendations = append(recommendations, bo.analyzeReliability(stats)...)
	
	// Resource utilization analysis
	recommendations = append(recommendations, bo.analyzeResourceUtilization(stats)...)
	
	// Caching effectiveness analysis
	recommendations = append(recommendations, bo.analyzeCaching(stats)...)
	
	// Concurrency analysis
	recommendations = append(recommendations, bo.analyzeConcurrency(stats)...)
	
	// Configuration analysis
	recommendations = append(recommendations, bo.analyzeConfiguration(stats)...)
	
	// Calculate overall performance score
	score := bo.calculatePerformanceScore(stats, recommendations)
	
	result := AnalysisResult{
		Timestamp:       time.Now(),
		TotalBuilds:     totalBuilds,
		SuccessRate:     successRate,
		AverageBuildTime: averageBuildTime,
		Recommendations: recommendations,
		Score:           score,
	}
	
	// Update internal state
	bo.updateRecommendations(recommendations)
	bo.addToHistory(result)
	
	log.Printf("Build performance analysis completed: Score=%.1f, Recommendations=%d", 
		score, len(recommendations))
	
	return result
}

// analyzePerformance analyzes build speed and timing performance.
func (bo *BuildOptimizer) analyzePerformance(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// Check average build time
	if avgTimeStr, ok := stats["average_build_time"].(string); ok {
		if avgTime, err := time.ParseDuration(avgTimeStr); err == nil {
			if avgTime > bo.slowBuildThreshold {
				level := OptimizationMedium
				if avgTime > bo.slowBuildThreshold*2 {
					level = OptimizationHigh
				}
				if avgTime > bo.slowBuildThreshold*5 {
					level = OptimizationCritical
				}
				
				recommendations = append(recommendations, OptimizationRecommendation{
					ID:          "slow-build-time",
					Title:       "Slow Build Performance",
					Description: fmt.Sprintf("Average build time (%.2fs) exceeds threshold (%.2fs)", avgTime.Seconds(), bo.slowBuildThreshold.Seconds()),
					Level:       level,
					Category:    CategoryPerformance,
					Impact:      "High - Affects development velocity",
					Effort:      "Medium - Requires performance optimization",
					Actions: []string{
						"Profile build pipeline for bottlenecks",
						"Optimize component compilation",
						"Increase worker pool size",
						"Enable parallel processing",
						"Review component dependencies",
					},
					Metrics: map[string]float64{
						"current_avg_seconds": avgTime.Seconds(),
						"threshold_seconds":   bo.slowBuildThreshold.Seconds(),
					},
					CreatedAt: time.Now(),
				})
			}
		}
	}
	
	// Analyze build time percentiles
	if p95Str, ok := stats["build_time_p95"].(string); ok {
		if p95Time, err := time.ParseDuration(p95Str); err == nil {
			if p50Str, ok := stats["build_time_p50"].(string); ok {
				if p50Time, err := time.ParseDuration(p50Str); err == nil {
					// Check for high variance in build times
					variance := p95Time.Seconds() / p50Time.Seconds()
					if variance > 3.0 {
						recommendations = append(recommendations, OptimizationRecommendation{
							ID:          "high-build-variance",
							Title:       "Inconsistent Build Performance",
							Description: fmt.Sprintf("P95 build time (%.2fs) is %.1fx longer than P50 (%.2fs)", p95Time.Seconds(), variance, p50Time.Seconds()),
							Level:       OptimizationMedium,
							Category:    CategoryPerformance,
							Impact:      "Medium - Unpredictable build times",
							Effort:      "Medium - Requires performance profiling",
							Actions: []string{
								"Identify components causing high variance",
								"Optimize slow components",
								"Balance worker load distribution",
								"Monitor resource contention",
							},
							Metrics: map[string]float64{
								"p50_seconds": p50Time.Seconds(),
								"p95_seconds": p95Time.Seconds(),
								"variance":    variance,
							},
							CreatedAt: time.Now(),
						})
					}
				}
			}
		}
	}
	
	return recommendations
}

// analyzeReliability analyzes build success rates and error patterns.
func (bo *BuildOptimizer) analyzeReliability(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// Check failure rate
	if failureRate, ok := stats["failure_rate"].(float64); ok {
		if failureRate > bo.highFailureRateThreshold*100 {
			level := OptimizationMedium
			if failureRate > bo.highFailureRateThreshold*200 {
				level = OptimizationHigh
			}
			if failureRate > bo.highFailureRateThreshold*500 {
				level = OptimizationCritical
			}
			
			recommendations = append(recommendations, OptimizationRecommendation{
				ID:          "high-failure-rate",
				Title:       "High Build Failure Rate",
				Description: fmt.Sprintf("Build failure rate (%.1f%%) exceeds threshold (%.1f%%)", failureRate, bo.highFailureRateThreshold*100),
				Level:       level,
				Category:    CategoryReliability,
				Impact:      "High - Affects build reliability",
				Effort:      "High - Requires error analysis and fixes",
				Actions: []string{
					"Analyze common failure patterns",
					"Improve error handling in build pipeline",
					"Add retry mechanisms for transient failures",
					"Enhance input validation",
					"Monitor and fix dependency issues",
				},
				Metrics: map[string]float64{
					"current_failure_rate": failureRate,
					"threshold_rate":       bo.highFailureRateThreshold * 100,
				},
				CreatedAt: time.Now(),
			})
		}
	}
	
	return recommendations
}

// analyzeResourceUtilization analyzes memory and CPU usage patterns.
func (bo *BuildOptimizer) analyzeResourceUtilization(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// Analyze worker utilization
	if workerStats, ok := stats["worker_stats"].(map[string]interface{}); ok {
		activeWorkers := 0
		totalWorkers := 0
		
		for _, workerData := range workerStats {
			if worker, ok := workerData.(map[string]interface{}); ok {
				totalWorkers++
				if isActive, ok := worker["is_active"].(bool); ok && isActive {
					activeWorkers++
				}
			}
		}
		
		if totalWorkers > 0 {
			utilization := float64(activeWorkers) / float64(totalWorkers)
			
			// Low utilization
			if utilization < 0.3 {
				recommendations = append(recommendations, OptimizationRecommendation{
					ID:          "low-worker-utilization",
					Title:       "Low Worker Utilization",
					Description: fmt.Sprintf("Worker utilization (%.1f%%) suggests over-provisioning", utilization*100),
					Level:       OptimizationInfo,
					Category:    CategoryResource,
					Impact:      "Low - Resource waste",
					Effort:      "Low - Configuration adjustment",
					Actions: []string{
						"Reduce worker pool size",
						"Optimize worker allocation strategy",
						"Consider dynamic worker scaling",
					},
					Metrics: map[string]float64{
						"utilization_percent": utilization * 100,
						"active_workers":      float64(activeWorkers),
						"total_workers":       float64(totalWorkers),
					},
					CreatedAt: time.Now(),
				})
			}
			
			// High utilization
			if utilization > 0.9 {
				recommendations = append(recommendations, OptimizationRecommendation{
					ID:          "high-worker-utilization",
					Title:       "High Worker Utilization",
					Description: fmt.Sprintf("Worker utilization (%.1f%%) suggests resource constraints", utilization*100),
					Level:       OptimizationMedium,
					Category:    CategoryResource,
					Impact:      "Medium - Potential bottleneck",
					Effort:      "Medium - Resource scaling",
					Actions: []string{
						"Increase worker pool size",
						"Monitor system resource usage",
						"Consider adding more build workers",
						"Optimize worker task distribution",
					},
					Metrics: map[string]float64{
						"utilization_percent": utilization * 100,
						"active_workers":      float64(activeWorkers),
						"total_workers":       float64(totalWorkers),
					},
					CreatedAt: time.Now(),
				})
			}
		}
	}
	
	return recommendations
}

// analyzeCaching analyzes build caching effectiveness.
func (bo *BuildOptimizer) analyzeCaching(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// For now, we'll add a placeholder recommendation to improve caching
	// This would be enhanced with actual cache hit rate metrics
	recommendations = append(recommendations, OptimizationRecommendation{
		ID:          "improve-caching",
		Title:       "Enhance Build Caching",
		Description: "Build caching could be improved to reduce redundant compilation",
		Level:       OptimizationMedium,
		Category:    CategoryCaching,
		Impact:      "High - Reduces build times significantly",
		Effort:      "Medium - Requires cache implementation",
		Actions: []string{
			"Implement content-based caching",
			"Add cache warming strategies",
			"Monitor cache hit rates",
			"Optimize cache key generation",
			"Add cache size management",
		},
		CreatedAt: time.Now(),
	})
	
	return recommendations
}

// analyzeConcurrency analyzes parallel processing effectiveness.
func (bo *BuildOptimizer) analyzeConcurrency(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// Analyze queue depth patterns
	if currentDepth, ok := stats["current_queue_depth"].(int64); ok {
		if maxDepth, ok := stats["max_queue_depth"].(int64); ok {
			if maxDepth > 0 {
				queueUtilization := float64(currentDepth) / float64(maxDepth)
				
				if maxDepth > 100 {
					recommendations = append(recommendations, OptimizationRecommendation{
						ID:          "high-queue-depth",
						Title:       "High Build Queue Depth",
						Description: fmt.Sprintf("Maximum queue depth (%d) indicates potential throughput issues", maxDepth),
						Level:       OptimizationMedium,
						Category:    CategoryConcurrency,
						Impact:      "Medium - Affects build latency",
						Effort:      "Medium - Requires concurrency tuning",
						Actions: []string{
							"Increase number of build workers",
							"Optimize build pipeline parallelization",
							"Implement priority queuing",
							"Monitor worker efficiency",
						},
						Metrics: map[string]float64{
							"max_queue_depth":     float64(maxDepth),
							"current_queue_depth": float64(currentDepth),
							"queue_utilization":   queueUtilization * 100,
						},
						CreatedAt: time.Now(),
					})
				}
			}
		}
	}
	
	return recommendations
}

// analyzeConfiguration analyzes build configuration for optimizations.
func (bo *BuildOptimizer) analyzeConfiguration(stats map[string]interface{}) []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	
	// General configuration optimization recommendation
	recommendations = append(recommendations, OptimizationRecommendation{
		ID:          "optimize-configuration",
		Title:       "Build Configuration Optimization",
		Description: "Review build configuration for performance opportunities",
		Level:       OptimizationInfo,
		Category:    CategoryConfiguration,
		Impact:      "Variable - Depends on specific optimizations",
		Effort:      "Low - Configuration adjustments",
		Actions: []string{
			"Review worker pool configuration",
			"Optimize timeout settings",
			"Configure appropriate log levels",
			"Review file watching patterns",
			"Optimize scan path configuration",
		},
		CreatedAt: time.Now(),
	})
	
	return recommendations
}

// calculatePerformanceScore calculates an overall performance score (0-100).
func (bo *BuildOptimizer) calculatePerformanceScore(stats map[string]interface{}, recommendations []OptimizationRecommendation) float64 {
	score := 100.0
	
	// Deduct points based on recommendation severity
	for _, rec := range recommendations {
		switch rec.Level {
		case OptimizationCritical:
			score -= 25.0
		case OptimizationHigh:
			score -= 15.0
		case OptimizationMedium:
			score -= 8.0
		case OptimizationLow:
			score -= 3.0
		case OptimizationInfo:
			score -= 1.0
		}
	}
	
	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}
	
	return score
}

// updateRecommendations updates the current recommendations list.
func (bo *BuildOptimizer) updateRecommendations(recommendations []OptimizationRecommendation) {
	bo.recommendationMutex.Lock()
	defer bo.recommendationMutex.Unlock()
	
	bo.recommendations = recommendations
}

// addToHistory adds an analysis result to the history.
func (bo *BuildOptimizer) addToHistory(result AnalysisResult) {
	bo.historyMutex.Lock()
	defer bo.historyMutex.Unlock()
	
	bo.analysisHistory = append(bo.analysisHistory, result)
	
	// Keep history within limits
	if len(bo.analysisHistory) > bo.maxHistorySize {
		bo.analysisHistory = bo.analysisHistory[len(bo.analysisHistory)-bo.maxHistorySize:]
	}
}

// GetCurrentRecommendations returns the current optimization recommendations.
func (bo *BuildOptimizer) GetCurrentRecommendations() []OptimizationRecommendation {
	bo.recommendationMutex.RLock()
	defer bo.recommendationMutex.RUnlock()
	
	// Return a copy to prevent race conditions
	recommendations := make([]OptimizationRecommendation, len(bo.recommendations))
	copy(recommendations, bo.recommendations)
	
	return recommendations
}

// GetRecommendationsByLevel returns recommendations filtered by severity level.
func (bo *BuildOptimizer) GetRecommendationsByLevel(level OptimizationLevel) []OptimizationRecommendation {
	all := bo.GetCurrentRecommendations()
	filtered := make([]OptimizationRecommendation, 0)
	
	for _, rec := range all {
		if rec.Level == level {
			filtered = append(filtered, rec)
		}
	}
	
	return filtered
}

// GetRecommendationsByCategory returns recommendations filtered by category.
func (bo *BuildOptimizer) GetRecommendationsByCategory(category OptimizationCategory) []OptimizationRecommendation {
	all := bo.GetCurrentRecommendations()
	filtered := make([]OptimizationRecommendation, 0)
	
	for _, rec := range all {
		if rec.Category == category {
			filtered = append(filtered, rec)
		}
	}
	
	return filtered
}

// GetAnalysisHistory returns the build analysis history.
func (bo *BuildOptimizer) GetAnalysisHistory() []AnalysisResult {
	bo.historyMutex.Lock()
	defer bo.historyMutex.Unlock()
	
	// Return a copy to prevent race conditions
	history := make([]AnalysisResult, len(bo.analysisHistory))
	copy(history, bo.analysisHistory)
	
	return history
}

// GetOptimizationSummary returns a summary of current optimization status.
func (bo *BuildOptimizer) GetOptimizationSummary() map[string]interface{} {
	recommendations := bo.GetCurrentRecommendations()
	
	summary := make(map[string]interface{})
	summary["total_recommendations"] = len(recommendations)
	
	// Count by level
	levelCounts := make(map[string]int)
	for _, rec := range recommendations {
		levelCounts[rec.Level.String()]++
	}
	summary["by_level"] = levelCounts
	
	// Count by category
	categoryCounts := make(map[string]int)
	for _, rec := range recommendations {
		categoryCounts[rec.Category.String()]++
	}
	summary["by_category"] = categoryCounts
	
	// Get latest analysis score
	history := bo.GetAnalysisHistory()
	if len(history) > 0 {
		latest := history[len(history)-1]
		summary["performance_score"] = latest.Score
		summary["last_analysis"] = latest.Timestamp
	}
	
	return summary
}
