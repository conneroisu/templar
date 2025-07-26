// Package performance provides comprehensive tests for statistical confidence calculations
// in performance regression detection.
//
// This test suite validates the accuracy of statistical methods including t-distribution,
// confidence intervals, multiple comparison corrections, and power analysis to ensure
// mathematically correct confidence levels in regression assessment.
package performance

import (
	"math"
	"os"
	"testing"
	"time"
)

// TestStatisticalValidator_BasicConfidenceCalculation tests basic statistical confidence
func TestStatisticalValidator_BasicConfidenceCalculation(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	tests := []struct {
		name              string
		currentValue      float64
		baseline          *PerformanceBaseline
		numComparisons    int
		expectSignificant bool
		minConfidence     float64
		description       string
	}{
		{
			name:         "clear regression with good sample size",
			currentValue: 2000.0,
			baseline: &PerformanceBaseline{
				BenchmarkName: "TestBenchmark",
				Samples:       []float64{1000, 1010, 990, 1005, 995, 1020, 980, 1015, 985, 1025},
				Mean:          1002.5,
				StdDev:        15.0,
				SampleCount:   10,
			},
			numComparisons:    1,
			expectSignificant: true,
			minConfidence:     0.95,
			description:       "Large difference with tight distribution should be highly significant",
		},
		{
			name:         "marginal change with large variance",
			currentValue: 1050.0,
			baseline: &PerformanceBaseline{
				BenchmarkName: "TestBenchmark",
				Samples:       []float64{900, 1200, 800, 1300, 700, 1400, 600, 1500, 1000, 1100},
				Mean:          1050.0,
				StdDev:        300.0,
				SampleCount:   10,
			},
			numComparisons:    1,
			expectSignificant: false,
			minConfidence:     0.0,
			description:       "Small difference with high variance should not be significant",
		},
		{
			name:         "small sample size t-test",
			currentValue: 150.0,
			baseline: &PerformanceBaseline{
				BenchmarkName: "TestBenchmark",
				Samples:       []float64{100, 105, 95},
				Mean:          100.0,
				StdDev:        5.0,
				SampleCount:   3,
			},
			numComparisons:    1,
			expectSignificant: true,
			minConfidence:     0.90,
			description:       "Small sample should use t-distribution with wider confidence intervals",
		},
		{
			name:         "multiple comparison correction",
			currentValue: 1015.0, // Very small effect size that should become non-significant
			baseline: &PerformanceBaseline{
				BenchmarkName: "TestBenchmark",
				Samples:       []float64{1000, 1010, 990, 1020, 980, 1005, 995, 1025, 975, 1030},
				Mean:          1003.5,
				StdDev:        20.0, // Large standard deviation relative to difference
				SampleCount:   10,
			},
			numComparisons:    100,   // Testing 100 benchmarks should reduce confidence significantly
			expectSignificant: false, // Should not be significant after Bonferroni correction
			minConfidence:     0.0,
			description:       "Multiple comparisons should reduce confidence via Bonferroni correction",
		},
		{
			name:         "zero variance baseline",
			currentValue: 1000.1,
			baseline: &PerformanceBaseline{
				BenchmarkName: "TestBenchmark",
				Samples:       []float64{1000, 1000, 1000, 1000, 1000},
				Mean:          1000.0,
				StdDev:        0.0,
				SampleCount:   5,
			},
			numComparisons:    1,
			expectSignificant: true,
			minConfidence:     0.99,
			description:       "Any difference from zero-variance baseline should be highly significant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.CalculateStatisticalConfidence(
				tt.currentValue,
				tt.baseline,
				tt.numComparisons,
			)

			isSignificant := validator.IsStatisticallySignificant(result)

			if isSignificant != tt.expectSignificant {
				t.Errorf("Expected significant=%v, got significant=%v. %s",
					tt.expectSignificant, isSignificant, tt.description)
				t.Errorf("Confidence: %.4f, P-value: %.4f, Test: %s",
					result.Confidence, result.PValue, result.TestType)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %.4f, got %.4f. %s",
					tt.minConfidence, result.Confidence, tt.description)
			}

			// Validate statistical result structure
			if result.SampleSize != tt.baseline.SampleCount {
				t.Errorf("Expected sample size %d, got %d",
					tt.baseline.SampleCount, result.SampleSize)
			}

			if result.DegreesOfFreedom != tt.baseline.SampleCount-1 {
				t.Errorf("Expected df %d, got %d",
					tt.baseline.SampleCount-1, result.DegreesOfFreedom)
			}
		})
	}
}

// TestStatisticalValidator_TDistributionVsNormal tests t-distribution vs normal distribution usage
func TestStatisticalValidator_TDistributionVsNormal(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	// Small sample should use t-test
	smallSample := &PerformanceBaseline{
		BenchmarkName: "SmallSample",
		Samples:       []float64{100, 110, 90, 105, 95}, // n=5
		Mean:          100.0,
		StdDev:        10.0,
		SampleCount:   5,
	}

	smallResult := validator.CalculateStatisticalConfidence(150.0, smallSample, 1)
	if smallResult.TestType != "t-test" {
		t.Errorf("Expected t-test for small sample (n=%d), got %s",
			smallSample.SampleCount, smallResult.TestType)
	}

	// Large sample should use z-test
	largeSampleValues := make([]float64, 50)
	for i := 0; i < 50; i++ {
		largeSampleValues[i] = 100.0 + float64(i%10) // Values from 100-109
	}

	largeSample := &PerformanceBaseline{
		BenchmarkName: "LargeSample",
		Samples:       largeSampleValues,
		Mean:          104.5,
		StdDev:        3.0,
		SampleCount:   50,
	}

	largeResult := validator.CalculateStatisticalConfidence(150.0, largeSample, 1)
	if largeResult.TestType != "z-test" {
		t.Errorf("Expected z-test for large sample (n=%d), got %s",
			largeSample.SampleCount, largeResult.TestType)
	}

	// Small sample should have lower confidence for same effect size
	// (due to t-distribution having fatter tails)
	if smallResult.Confidence >= largeResult.Confidence {
		t.Errorf(
			"Expected small sample confidence (%.4f) < large sample confidence (%.4f) for same effect size",
			smallResult.Confidence,
			largeResult.Confidence,
		)
	}
}

// TestStatisticalValidator_MultipleComparisonCorrection tests Bonferroni correction
func TestStatisticalValidator_MultipleComparisonCorrection(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	baseline := &PerformanceBaseline{
		BenchmarkName: "TestBenchmark",
		Samples:       []float64{1000, 1010, 990, 1020, 980, 1030, 970, 1040, 960, 1050},
		Mean:          1000.0,
		StdDev:        30.0,
		SampleCount:   10,
	}

	currentValue := 1100.0 // 10% increase

	// Single comparison
	singleResult := validator.CalculateStatisticalConfidence(currentValue, baseline, 1)

	// Multiple comparisons (20 tests)
	multipleResult := validator.CalculateStatisticalConfidence(currentValue, baseline, 20)

	// Multiple comparison correction should reduce confidence
	if multipleResult.Confidence >= singleResult.Confidence {
		t.Errorf("Expected multiple comparison confidence (%.4f) < single comparison (%.4f)",
			multipleResult.Confidence, singleResult.Confidence)
	}

	// Test with extreme multiple comparisons
	extremeResult := validator.CalculateStatisticalConfidence(currentValue, baseline, 1000)

	if extremeResult.Confidence >= multipleResult.Confidence {
		t.Errorf(
			"Expected extreme multiple comparison confidence (%.4f) < moderate multiple (%.4f)",
			extremeResult.Confidence,
			multipleResult.Confidence,
		)
	}

	// Confidence should be bounded [0, 1]
	if extremeResult.Confidence < 0.0 || extremeResult.Confidence > 1.0 {
		t.Errorf("Confidence should be in [0,1], got %.4f", extremeResult.Confidence)
	}
}

// TestStatisticalValidator_ConfidenceIntervals tests confidence interval calculation
func TestStatisticalValidator_ConfidenceIntervals(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	baseline := &PerformanceBaseline{
		BenchmarkName: "TestBenchmark",
		Samples:       []float64{1000, 1020, 980, 1040, 960, 1060, 940, 1080, 920, 1100},
		Mean:          1000.0,
		StdDev:        50.0,
		SampleCount:   10,
	}

	result := validator.CalculateStatisticalConfidence(1200.0, baseline, 1)

	// Confidence interval should contain the mean difference
	meanDiff := 1200.0 - baseline.Mean // 200.0
	ci := result.ConfidenceInterval

	if ci.Lower > meanDiff || ci.Upper < meanDiff {
		t.Errorf("Confidence interval [%.2f, %.2f] should contain mean difference %.2f",
			ci.Lower, ci.Upper, meanDiff)
	}

	// Confidence interval should have the specified level
	if ci.Level != 0.95 {
		t.Errorf("Expected confidence level 0.95, got %.2f", ci.Level)
	}

	// Upper bound should be greater than lower bound
	if ci.Upper <= ci.Lower {
		t.Errorf("Upper bound (%.2f) should be > lower bound (%.2f)",
			ci.Upper, ci.Lower)
	}

	// For a positive mean difference, interval should generally be positive
	// (though it could cross zero in some cases)
	if ci.Upper < 0 {
		t.Errorf("For positive mean difference, upper bound should not be negative: %.2f",
			ci.Upper)
	}
}

// TestStatisticalValidator_EffectSizeClassification tests Cohen's d effect size calculation
func TestStatisticalValidator_EffectSizeClassification(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	baseline := &PerformanceBaseline{
		BenchmarkName: "TestBenchmark",
		Samples:       []float64{1000, 1000, 1000, 1000, 1000},
		Mean:          1000.0,
		StdDev:        100.0, // Use consistent std dev for effect size calculation
		SampleCount:   5,
	}

	tests := []struct {
		name         string
		currentValue float64
		expectedSize string
		description  string
	}{
		{
			name:         "negligible effect",
			currentValue: 1010.0, // 0.1 Cohen's d
			expectedSize: "negligible",
			description:  "10ns difference with 100ns std dev should be negligible",
		},
		{
			name:         "small effect",
			currentValue: 1030.0, // 0.3 Cohen's d
			expectedSize: "small",
			description:  "30ns difference should be small effect",
		},
		{
			name:         "medium effect",
			currentValue: 1070.0, // 0.7 Cohen's d
			expectedSize: "medium",
			description:  "70ns difference should be medium effect",
		},
		{
			name:         "large effect",
			currentValue: 1090.0, // 0.9 Cohen's d
			expectedSize: "large",
			description:  "90ns difference should be large effect",
		},
		{
			name:         "very large effect",
			currentValue: 1150.0, // 1.5 Cohen's d
			expectedSize: "very_large",
			description:  "150ns difference should be very large effect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.CalculateStatisticalConfidence(tt.currentValue, baseline, 1)
			effectSize := validator.ClassifyEffectSize(result.EffectSize)

			if effectSize != tt.expectedSize {
				t.Errorf("Expected effect size '%s', got '%s'. Cohen's d = %.3f. %s",
					tt.expectedSize, effectSize, result.EffectSize, tt.description)
			}

			// Effect size should match Cohen's d calculation
			expectedCohenD := (tt.currentValue - baseline.Mean) / baseline.StdDev
			if math.Abs(result.EffectSize-expectedCohenD) > 0.001 {
				t.Errorf("Expected Cohen's d %.3f, got %.3f",
					expectedCohenD, result.EffectSize)
			}
		})
	}
}

// TestStatisticalValidator_EdgeCases tests statistical validator edge cases
func TestStatisticalValidator_EdgeCases(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	// Empty baseline
	emptyBaseline := &PerformanceBaseline{
		BenchmarkName: "Empty",
		Samples:       []float64{},
		SampleCount:   0,
	}

	emptyResult := validator.CalculateStatisticalConfidence(100.0, emptyBaseline, 1)
	if emptyResult.Confidence != 0.0 {
		t.Errorf("Expected 0.0 confidence for empty baseline, got %.4f", emptyResult.Confidence)
	}
	if emptyResult.TestType != "insufficient_data" {
		t.Errorf("Expected 'insufficient_data' test type, got '%s'", emptyResult.TestType)
	}

	// Single sample baseline
	singleBaseline := &PerformanceBaseline{
		BenchmarkName: "Single",
		Samples:       []float64{100.0},
		Mean:          100.0,
		StdDev:        0.0,
		SampleCount:   1,
	}

	singleResult := validator.CalculateStatisticalConfidence(200.0, singleBaseline, 1)
	if singleResult.Confidence != 0.5 {
		t.Errorf("Expected 0.5 confidence for single sample, got %.4f", singleResult.Confidence)
	}
	if singleResult.TestType != "single_sample" {
		t.Errorf("Expected 'single_sample' test type, got '%s'", singleResult.TestType)
	}

	// Zero variance baseline with same value
	zeroVarSame := &PerformanceBaseline{
		BenchmarkName: "ZeroVarSame",
		Samples:       []float64{100, 100, 100, 100},
		Mean:          100.0,
		StdDev:        0.0,
		SampleCount:   4,
	}

	sameResult := validator.CalculateStatisticalConfidence(100.0, zeroVarSame, 1)
	if sameResult.Confidence != 1.0 {
		t.Errorf(
			"Expected 1.0 confidence for identical value with zero variance, got %.4f",
			sameResult.Confidence,
		)
	}

	// Zero variance baseline with different value
	diffResult := validator.CalculateStatisticalConfidence(101.0, zeroVarSame, 1)
	if diffResult.Confidence < 0.99 {
		t.Errorf(
			"Expected high confidence (>=0.99) for different value with zero variance, got %.4f",
			diffResult.Confidence,
		)
	}
	if diffResult.TestType != "no_baseline_variance" {
		t.Errorf("Expected 'no_baseline_variance' test type, got '%s'", diffResult.TestType)
	}
}

// TestStatisticalValidator_PowerAnalysis tests statistical power calculations
func TestStatisticalValidator_PowerAnalysis(t *testing.T) {
	validator := NewStatisticalValidator(0.95, 3)

	tests := []struct {
		name        string
		sampleSize  int
		effectSize  float64
		alpha       float64
		minPower    float64
		maxPower    float64
		description string
	}{
		{
			name:        "small sample small effect",
			sampleSize:  3,
			effectSize:  0.2,
			alpha:       0.05,
			minPower:    0.05,
			maxPower:    0.30,
			description: "Small sample with small effect should have low power",
		},
		{
			name:        "large sample large effect",
			sampleSize:  100,
			effectSize:  1.0,
			alpha:       0.05,
			minPower:    0.80,
			maxPower:    0.99,
			description: "Large sample with large effect should have high power",
		},
		{
			name:        "invalid sample size",
			sampleSize:  1,
			effectSize:  0.5,
			alpha:       0.05,
			minPower:    0.0,
			maxPower:    0.0,
			description: "Invalid sample size should return 0 power",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			power := validator.CalculatePowerAnalysis(tt.sampleSize, tt.effectSize, tt.alpha)

			if power < tt.minPower || power > tt.maxPower {
				t.Errorf("Expected power in [%.2f, %.2f], got %.4f. %s",
					tt.minPower, tt.maxPower, power, tt.description)
			}

			// Power should be bounded [0, 1]
			if power < 0.0 || power > 1.0 {
				t.Errorf("Power should be in [0,1], got %.4f", power)
			}
		})
	}
}

// TestStatisticalValidator_IntegrationWithDetector tests integration with performance detector
func TestStatisticalValidator_IntegrationWithDetector(t *testing.T) {
	// Create a detector with proper statistical validation
	detector := NewPerformanceDetector("test_stats_integration", DefaultThresholds())
	defer func() {
		_ = os.RemoveAll("test_stats_integration")
	}()

	// Create baseline data with known statistical properties
	baselineResults := []BenchmarkResult{
		{Name: "TestBenchmark", NsPerOp: 1000, Timestamp: time.Now()},
		{Name: "TestBenchmark", NsPerOp: 1010, Timestamp: time.Now()},
		{Name: "TestBenchmark", NsPerOp: 990, Timestamp: time.Now()},
		{Name: "TestBenchmark", NsPerOp: 1020, Timestamp: time.Now()},
		{Name: "TestBenchmark", NsPerOp: 980, Timestamp: time.Now()},
	}

	// Update baselines
	err := detector.UpdateBaselines(baselineResults)
	if err != nil {
		t.Fatalf("Failed to update baselines: %v", err)
	}

	// Test regression detection with statistical validation
	currentResults := []BenchmarkResult{
		{Name: "TestBenchmark", NsPerOp: 2000, Timestamp: time.Now()},  // Clear regression
		{Name: "TestBenchmark2", NsPerOp: 1000, Timestamp: time.Now()}, // No baseline yet
	}

	regressions, err := detector.DetectRegressions(currentResults)
	if err != nil {
		t.Fatalf("Failed to detect regressions: %v", err)
	}

	// Should detect the clear regression with high confidence
	if len(regressions) == 0 {
		t.Error("Expected to detect regression, but none found")
		return
	}

	regression := regressions[0]

	// Validate statistical properties
	if regression.Confidence < 0.90 {
		t.Errorf("Expected high confidence (>=0.90) for clear regression, got %.4f",
			regression.Confidence)
	}

	if regression.BenchmarkName != "TestBenchmark" {
		t.Errorf("Expected benchmark name 'TestBenchmark', got '%s'",
			regression.BenchmarkName)
	}

	if regression.RegressionType != "performance" {
		t.Errorf("Expected regression type 'performance', got '%s'",
			regression.RegressionType)
	}

	// Percentage change should be approximately 100% (1000 -> 2000)
	expectedChange := 100.0
	if math.Abs(regression.PercentageChange-expectedChange) > 10.0 {
		t.Errorf("Expected percentage change ~%.1f%%, got %.1f%%",
			expectedChange, regression.PercentageChange)
	}
}
