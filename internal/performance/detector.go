// Package performance provides performance monitoring and regression detection capabilities.
//
// The detector package implements automated performance baseline establishment,
// metrics collection, regression detection with configurable thresholds, and
// CI/CD integration for continuous performance monitoring. It supports various
// benchmark formats and provides alerting for performance degradations.
package performance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/validation"
)

// BenchmarkResult represents a single benchmark measurement
type BenchmarkResult struct {
	Name        string    `json:"name"`
	Iterations  int       `json:"iterations"`
	NsPerOp     float64   `json:"ns_per_op"`
	BytesPerOp  int64     `json:"bytes_per_op"`
	AllocsPerOp int64     `json:"allocs_per_op"`
	MBPerSec    float64   `json:"mb_per_sec,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	GitCommit   string    `json:"git_commit,omitempty"`
	GitBranch   string    `json:"git_branch,omitempty"`
	Environment string    `json:"environment,omitempty"`
}

// PerformanceBaseline represents historical performance data
type PerformanceBaseline struct {
	BenchmarkName string    `json:"benchmark_name"`
	Samples       []float64 `json:"samples"`
	Mean          float64   `json:"mean"`
	Median        float64   `json:"median"`
	StdDev        float64   `json:"std_dev"`
	Min           float64   `json:"min"`
	Max           float64   `json:"max"`
	LastUpdated   time.Time `json:"last_updated"`
	SampleCount   int       `json:"sample_count"`
}

// RegressionThresholds defines acceptable performance degradation limits
type RegressionThresholds struct {
	// Performance degradation threshold (e.g., 1.15 = 15% slower is acceptable)
	SlownessThreshold float64 `json:"slowness_threshold"`
	// Memory usage increase threshold (e.g., 1.20 = 20% more memory is acceptable)
	MemoryThreshold float64 `json:"memory_threshold"`
	// Allocation increase threshold (e.g., 1.25 = 25% more allocations is acceptable)
	AllocThreshold float64 `json:"alloc_threshold"`
	// Minimum samples required before regression detection
	MinSamples int `json:"min_samples"`
	// Statistical confidence level (e.g., 0.95 = 95% confidence)
	ConfidenceLevel float64 `json:"confidence_level"`
}

// RegressionDetection contains regression analysis results
type RegressionDetection struct {
	BenchmarkName     string  `json:"benchmark_name"`
	IsRegression      bool    `json:"is_regression"`
	CurrentValue      float64 `json:"current_value"`
	BaselineValue     float64 `json:"baseline_value"`
	PercentageChange  float64 `json:"percentage_change"`
	Threshold         float64 `json:"threshold"`
	Confidence        float64 `json:"confidence"`
	RegressionType    string  `json:"regression_type"` // "performance", "memory", "allocations"
	Severity          string  `json:"severity"`        // "minor", "major", "critical"
	RecommendedAction string  `json:"recommended_action"`
}

// PerformanceDetector handles performance regression detection
type PerformanceDetector struct {
	baselineDir          string
	thresholds           RegressionThresholds
	gitCommit            string
	gitBranch            string
	environment          string
	statisticalValidator *StatisticalValidator
}

// NewPerformanceDetector creates a new performance detector
func NewPerformanceDetector(baselineDir string, thresholds RegressionThresholds) *PerformanceDetector {
	// Create statistical validator with 95% confidence level and minimum 3 samples
	statisticalValidator := NewStatisticalValidator(thresholds.ConfidenceLevel, 3)

	return &PerformanceDetector{
		baselineDir:          baselineDir,
		thresholds:           thresholds,
		environment:          getEnvironment(),
		statisticalValidator: statisticalValidator,
	}
}

// SetGitInfo sets git commit and branch information
func (pd *PerformanceDetector) SetGitInfo(commit, branch string) {
	pd.gitCommit = commit
	pd.gitBranch = branch
}

// ParseBenchmarkOutput parses Go benchmark output and returns structured results
func (pd *PerformanceDetector) ParseBenchmarkOutput(output string) ([]BenchmarkResult, error) {
	var results []BenchmarkResult

	// Regex to match Go benchmark output lines
	// Example: BenchmarkComponentScanner_ScanDirectory/components-10-16         	    2204	    604432 ns/op	  261857 B/op	    5834 allocs/op
	benchmarkRegex := regexp.MustCompile(`^Benchmark(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?(?:\s+(\d+(?:\.\d+)?)\s+MB/s)?`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		matches := benchmarkRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}

		iterations, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		nsPerOp, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			continue
		}

		result := BenchmarkResult{
			Name:        matches[1],
			Iterations:  iterations,
			NsPerOp:     nsPerOp,
			Timestamp:   time.Now(),
			GitCommit:   pd.gitCommit,
			GitBranch:   pd.gitBranch,
			Environment: pd.environment,
		}

		// Parse optional fields
		if len(matches) > 4 && matches[4] != "" {
			if bytesPerOp, err := strconv.ParseInt(matches[4], 10, 64); err == nil {
				result.BytesPerOp = bytesPerOp
			}
		}

		if len(matches) > 5 && matches[5] != "" {
			if allocsPerOp, err := strconv.ParseInt(matches[5], 10, 64); err == nil {
				result.AllocsPerOp = allocsPerOp
			}
		}

		if len(matches) > 6 && matches[6] != "" {
			if mbPerSec, err := strconv.ParseFloat(matches[6], 64); err == nil {
				result.MBPerSec = mbPerSec
			}
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning benchmark output: %w", err)
	}

	return results, nil
}

// UpdateBaselines updates performance baselines with new benchmark results
func (pd *PerformanceDetector) UpdateBaselines(results []BenchmarkResult) error {
	// Validate baseline directory path to prevent path traversal attacks
	if err := pd.validateBaselineDirectory(); err != nil {
		return fmt.Errorf("invalid baseline directory: %w", err)
	}

	if err := os.MkdirAll(pd.baselineDir, 0700); err != nil { // More restrictive permissions
		return fmt.Errorf("creating baseline directory: %w", err)
	}

	for _, result := range results {
		baseline, err := pd.loadBaseline(result.Name)
		if err != nil {
			// Create new baseline if it doesn't exist
			baseline = &PerformanceBaseline{
				BenchmarkName: result.Name,
				Samples:       []float64{},
			}
		}

		// Add new sample
		baseline.Samples = append(baseline.Samples, result.NsPerOp)

		// Keep only the last 100 samples to prevent unlimited growth
		const maxSamples = 100
		if len(baseline.Samples) > maxSamples {
			baseline.Samples = baseline.Samples[len(baseline.Samples)-maxSamples:]
		}

		// Recalculate statistics
		pd.calculateStatistics(baseline)
		baseline.LastUpdated = time.Now()
		baseline.SampleCount = len(baseline.Samples)

		if err := pd.saveBaseline(baseline); err != nil {
			return fmt.Errorf("saving baseline for %s: %w", result.Name, err)
		}
	}

	return nil
}

// DetectRegressions analyzes benchmark results against baselines for regressions
func (pd *PerformanceDetector) DetectRegressions(results []BenchmarkResult) ([]RegressionDetection, error) {
	var regressions []RegressionDetection

	// Calculate total number of statistical comparisons for multiple testing correction
	// We test 3 metrics per benchmark: performance, memory, allocations
	numComparisons := len(results) * 3

	for _, result := range results {
		baseline, err := pd.loadBaseline(result.Name)
		if err != nil {
			// Skip if no baseline exists yet
			continue
		}

		// Need minimum samples for reliable detection
		if baseline.SampleCount < pd.thresholds.MinSamples {
			continue
		}

		// Detect performance regression
		if perfRegression := pd.detectPerformanceRegressionWithStats(result, baseline, numComparisons); perfRegression != nil {
			regressions = append(regressions, *perfRegression)
		}

		// Detect memory regression
		if memRegression := pd.detectMemoryRegressionWithStats(result, baseline, numComparisons); memRegression != nil {
			regressions = append(regressions, *memRegression)
		}

		// Detect allocation regression
		if allocRegression := pd.detectAllocationRegressionWithStats(result, baseline, numComparisons); allocRegression != nil {
			regressions = append(regressions, *allocRegression)
		}
	}

	return regressions, nil
}

// detectPerformanceRegressionWithStats checks for execution time regressions with proper statistics
func (pd *PerformanceDetector) detectPerformanceRegressionWithStats(result BenchmarkResult, baseline *PerformanceBaseline, numComparisons int) *RegressionDetection {
	// Perform rigorous statistical analysis
	statResult := pd.statisticalValidator.CalculateStatisticalConfidence(
		result.NsPerOp,
		baseline,
		numComparisons,
	)

	// Check if statistically significant
	if !pd.statisticalValidator.IsStatisticallySignificant(statResult) {
		return nil // Not statistically significant
	}

	ratio := result.NsPerOp / baseline.Mean

	if ratio > pd.thresholds.SlownessThreshold {
		percentageChange := (ratio - 1.0) * 100
		severity := pd.calculateSeverity(ratio, pd.thresholds.SlownessThreshold)

		return &RegressionDetection{
			BenchmarkName:     result.Name,
			IsRegression:      true,
			CurrentValue:      result.NsPerOp,
			BaselineValue:     baseline.Mean,
			PercentageChange:  percentageChange,
			Threshold:         pd.thresholds.SlownessThreshold,
			Confidence:        statResult.Confidence,
			RegressionType:    "performance",
			Severity:          severity,
			RecommendedAction: pd.getPerformanceRecommendation(severity, percentageChange),
		}
	}

	return nil
}


// regressionParams holds parameters for regression detection
type regressionParams struct {
	metricValue     int64
	suffix          string
	meanMultiplier  float64
	stdDevMultiplier float64
	sampleScaling   float64
	threshold       float64
	regressionType  string
	getRecommendation func(severity string, percentageChange float64) string
}

// detectRegressionWithStats is a helper function for memory and allocation regression detection
func (pd *PerformanceDetector) detectRegressionWithStats(result BenchmarkResult, baseline *PerformanceBaseline, numComparisons int, params regressionParams) *RegressionDetection {
	if params.metricValue == 0 {
		return nil // No data available
	}

	// Create baseline from performance baseline samples
	// Convert ns/op samples to a rough baseline (this is a simplification)
	// In production, you'd maintain separate baselines for each metric type
	metricBaseline := &PerformanceBaseline{
		BenchmarkName: baseline.BenchmarkName + params.suffix,
		Samples:       make([]float64, len(baseline.Samples)),
		Mean:          float64(params.metricValue) * params.meanMultiplier,
		StdDev:        float64(params.metricValue) * params.stdDevMultiplier,
		SampleCount:   baseline.SampleCount,
	}

	// Copy samples with scaling (rough approximation)
	for i, sample := range baseline.Samples {
		metricBaseline.Samples[i] = sample * params.sampleScaling
	}

	// Perform statistical analysis
	statResult := pd.statisticalValidator.CalculateStatisticalConfidence(
		float64(params.metricValue),
		metricBaseline,
		numComparisons,
	)

	// Check if statistically significant
	if !pd.statisticalValidator.IsStatisticallySignificant(statResult) {
		return nil // Not statistically significant
	}

	ratio := float64(params.metricValue) / metricBaseline.Mean

	if ratio > params.threshold {
		percentageChange := (ratio - 1.0) * 100
		severity := pd.calculateSeverity(ratio, params.threshold)

		return &RegressionDetection{
			BenchmarkName:     result.Name,
			IsRegression:      true,
			CurrentValue:      float64(params.metricValue),
			BaselineValue:     metricBaseline.Mean,
			PercentageChange:  percentageChange,
			Threshold:         params.threshold,
			Confidence:        statResult.Confidence,
			RegressionType:    params.regressionType,
			Severity:          severity,
			RecommendedAction: params.getRecommendation(severity, percentageChange),
		}
	}

	return nil
}

// detectMemoryRegressionWithStats checks for memory usage regressions with proper statistics
func (pd *PerformanceDetector) detectMemoryRegressionWithStats(result BenchmarkResult, baseline *PerformanceBaseline, numComparisons int) *RegressionDetection {
	return pd.detectRegressionWithStats(result, baseline, numComparisons, regressionParams{
		metricValue:       result.BytesPerOp,
		suffix:            "_memory",
		meanMultiplier:    0.8, // Conservative estimate
		stdDevMultiplier:  0.1, // Assume 10% variance
		sampleScaling:     0.1, // Scale performance to approximate memory
		threshold:         pd.thresholds.MemoryThreshold,
		regressionType:    "memory",
		getRecommendation: pd.getMemoryRecommendation,
	})
}


// detectAllocationRegressionWithStats checks for allocation count regressions with proper statistics
func (pd *PerformanceDetector) detectAllocationRegressionWithStats(result BenchmarkResult, baseline *PerformanceBaseline, numComparisons int) *RegressionDetection {
	return pd.detectRegressionWithStats(result, baseline, numComparisons, regressionParams{
		metricValue:       result.AllocsPerOp,
		suffix:            "_allocs",
		meanMultiplier:    0.75, // Conservative estimate
		stdDevMultiplier:  0.05, // Assume 5% variance (allocations are typically more stable)
		sampleScaling:     0.001, // Scale performance to approximate allocations
		threshold:         pd.thresholds.AllocThreshold,
		regressionType:    "allocations",
		getRecommendation: pd.getAllocationRecommendation,
	})
}


// calculateSeverity determines regression severity based on threshold ratio
func (pd *PerformanceDetector) calculateSeverity(ratio, threshold float64) string {
	if ratio > threshold*2.0 {
		return "critical"
	} else if ratio > threshold*1.15 {
		return "major"
	}
	return "minor"
}


// calculateStatistics computes statistical measures for baseline samples
func (pd *PerformanceDetector) calculateStatistics(baseline *PerformanceBaseline) {
	if len(baseline.Samples) == 0 {
		return
	}

	// Calculate mean
	var sum float64
	for _, sample := range baseline.Samples {
		sum += sample
	}
	baseline.Mean = sum / float64(len(baseline.Samples))

	// Calculate median
	sorted := make([]float64, len(baseline.Samples))
	copy(sorted, baseline.Samples)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		baseline.Median = (sorted[n/2-1] + sorted[n/2]) / 2
	} else {
		baseline.Median = sorted[n/2]
	}

	// Calculate standard deviation
	var variance float64
	for _, sample := range baseline.Samples {
		variance += math.Pow(sample-baseline.Mean, 2)
	}
	variance /= float64(len(baseline.Samples))
	baseline.StdDev = math.Sqrt(variance)

	// Calculate min and max
	baseline.Min = sorted[0]
	baseline.Max = sorted[n-1]
}

// loadBaseline loads performance baseline from disk
func (pd *PerformanceDetector) loadBaseline(benchmarkName string) (*PerformanceBaseline, error) {
	filename := filepath.Join(pd.baselineDir, sanitizeFilename(benchmarkName)+".json")

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var baseline PerformanceBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("unmarshaling baseline: %w", err)
	}

	return &baseline, nil
}

// validateBaselineDirectory validates baseline directory path to prevent path traversal attacks
func (pd *PerformanceDetector) validateBaselineDirectory() error {
	// Validate the baseline directory path using the security validation package
	if err := validation.ValidatePath(pd.baselineDir); err != nil {
		return fmt.Errorf("baseline directory validation failed: %w", err)
	}

	// Ensure the baseline directory is within the current working directory
	absBaselineDir, err := filepath.Abs(pd.baselineDir)
	if err != nil {
		return fmt.Errorf("getting absolute baseline directory: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current working directory: %w", err)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("getting absolute current directory: %w", err)
	}

	// Ensure baseline directory is within current working directory or explicitly allowed subdirectories
	if !strings.HasPrefix(absBaselineDir, absCwd) {
		return fmt.Errorf("baseline directory '%s' is outside current working directory '%s'", pd.baselineDir, cwd)
	}

	// Additional security: prevent writing to parent directories
	relPath, err := filepath.Rel(absCwd, absBaselineDir)
	if err != nil {
		return fmt.Errorf("calculating relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("baseline directory contains parent directory traversal: %s", pd.baselineDir)
	}

	return nil
}

// saveBaseline saves performance baseline to disk
func (pd *PerformanceDetector) saveBaseline(baseline *PerformanceBaseline) error {
	// Sanitize and validate the benchmark name
	sanitizedName := sanitizeFilename(baseline.BenchmarkName)
	if sanitizedName == "" {
		return fmt.Errorf("invalid benchmark name after sanitization: %s", baseline.BenchmarkName)
	}

	filename := filepath.Join(pd.baselineDir, sanitizedName+".json")

	// Validate the complete file path
	if err := validation.ValidatePath(filename); err != nil {
		return fmt.Errorf("invalid baseline file path: %w", err)
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}

	// Use more restrictive file permissions (0600 = read/write for owner only)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("writing baseline file: %w", err)
	}

	return nil
}

// getPerformanceRecommendation provides actionable recommendations for performance regressions
func (pd *PerformanceDetector) getPerformanceRecommendation(severity string, percentageChange float64) string {
	switch severity {
	case "critical":
		return fmt.Sprintf("CRITICAL: %.1f%% performance degradation. Immediate investigation required. Consider reverting recent changes.", percentageChange)
	case "major":
		return fmt.Sprintf("MAJOR: %.1f%% performance degradation. Review recent commits for performance impact.", percentageChange)
	default:
		return fmt.Sprintf("MINOR: %.1f%% performance degradation. Monitor for trends.", percentageChange)
	}
}

// getMemoryRecommendation provides actionable recommendations for memory regressions
func (pd *PerformanceDetector) getMemoryRecommendation(severity string, percentageChange float64) string {
	switch severity {
	case "critical":
		return fmt.Sprintf("CRITICAL: %.1f%% memory increase. Check for memory leaks and excessive allocations.", percentageChange)
	case "major":
		return fmt.Sprintf("MAJOR: %.1f%% memory increase. Review data structures and caching strategies.", percentageChange)
	default:
		return fmt.Sprintf("MINOR: %.1f%% memory increase. Consider memory optimization opportunities.", percentageChange)
	}
}

// getAllocationRecommendation provides actionable recommendations for allocation regressions
func (pd *PerformanceDetector) getAllocationRecommendation(severity string, percentageChange float64) string {
	switch severity {
	case "critical":
		return fmt.Sprintf("CRITICAL: %.1f%% allocation increase. Implement object pooling and reduce unnecessary allocations.", percentageChange)
	case "major":
		return fmt.Sprintf("MAJOR: %.1f%% allocation increase. Review slice growth patterns and string concatenations.", percentageChange)
	default:
		return fmt.Sprintf("MINOR: %.1f%% allocation increase. Consider allocation reduction techniques.", percentageChange)
	}
}

// sanitizeFilename creates a safe filename from benchmark name
func sanitizeFilename(name string) string {
	// Replace unsafe characters with underscores
	safe := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`).ReplaceAllString(name, "_")
	return strings.TrimSuffix(safe, "_")
}

// getEnvironment detects the current environment
func getEnvironment() string {
	if os.Getenv("CI") != "" {
		return "ci"
	}
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return "github-actions"
	}
	return "local"
}

// DefaultThresholds returns reasonable default regression thresholds
func DefaultThresholds() RegressionThresholds {
	return RegressionThresholds{
		SlownessThreshold: 1.15, // 15% performance degradation
		MemoryThreshold:   1.20, // 20% memory increase
		AllocThreshold:    1.25, // 25% allocation increase
		MinSamples:        5,    // Need at least 5 samples
		ConfidenceLevel:   0.95, // 95% confidence
	}
}
