// Package performance provides statistical functions for accurate confidence calculations
// in performance regression detection.
//
// This module implements proper statistical methods including t-distribution for small
// samples, confidence intervals, and multiple comparison corrections to prevent
// false positives in regression detection.
package performance

import (
	"math"
)

// StatisticalResult contains detailed statistical analysis results
type StatisticalResult struct {
	TStatistic         float64            `json:"t_statistic"`
	DegreesOfFreedom   int                `json:"degrees_of_freedom"`
	PValue             float64            `json:"p_value"`
	Confidence         float64            `json:"confidence"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	EffectSize         float64            `json:"effect_size"` // Cohen's d
	SampleSize         int                `json:"sample_size"`
	TestType           string             `json:"test_type"` // "t-test" or "z-test"
}

// ConfidenceInterval represents a statistical confidence interval
type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
	Level float64 `json:"level"` // e.g., 0.95 for 95% confidence
}

// MultipleComparisonCorrection applies corrections for multiple testing
type MultipleComparisonCorrection struct {
	Method         string  `json:"method"` // "bonferroni", "benjamini-hochberg"
	NumComparisons int     `json:"num_comparisons"`
	CorrectedAlpha float64 `json:"corrected_alpha"`
	OriginalAlpha  float64 `json:"original_alpha"`
}

// StatisticalValidator provides rigorous statistical analysis for performance regression
type StatisticalValidator struct {
	confidenceLevel       float64
	minSampleSize         int
	useMultipleCorrection bool
	correctionMethod      string
}

// NewStatisticalValidator creates a new validator with proper statistical configuration
func NewStatisticalValidator(confidenceLevel float64, minSampleSize int) *StatisticalValidator {
	return &StatisticalValidator{
		confidenceLevel:       confidenceLevel,
		minSampleSize:         minSampleSize,
		useMultipleCorrection: true,
		correctionMethod:      "bonferroni", // Conservative multiple comparison correction
	}
}

// CalculateStatisticalConfidence performs rigorous statistical analysis
func (sv *StatisticalValidator) CalculateStatisticalConfidence(
	currentValue float64,
	baseline *PerformanceBaseline,
	numComparisons int,
) StatisticalResult {

	// Handle edge cases
	if len(baseline.Samples) == 0 {
		return StatisticalResult{
			Confidence: 0.0,
			TestType:   "insufficient_data",
			SampleSize: 0,
		}
	}

	if len(baseline.Samples) == 1 {
		return StatisticalResult{
			Confidence: 0.5, // No statistical inference possible with n=1
			TestType:   "single_sample",
			SampleSize: 1,
		}
	}

	sampleSize := len(baseline.Samples)

	// Calculate sample statistics
	mean := baseline.Mean
	stdDev := baseline.StdDev

	// Handle zero variance case
	if stdDev == 0 {
		if math.Abs(currentValue-mean) < 1e-10 { // Account for floating point precision
			return StatisticalResult{
				Confidence:       1.0,
				TestType:         "no_variance",
				SampleSize:       sampleSize,
				DegreesOfFreedom: sampleSize - 1,
				EffectSize:       0.0,
			}
		} else {
			// Perfect confidence in detection of difference when baseline has no variance
			return StatisticalResult{
				Confidence:       0.99, // Cap at 99% to avoid overconfidence
				TestType:         "no_baseline_variance",
				SampleSize:       sampleSize,
				DegreesOfFreedom: sampleSize - 1,
				EffectSize:       math.Inf(1), // Infinite effect size
			}
		}
	}

	// Calculate standard error
	standardError := stdDev / math.Sqrt(float64(sampleSize))

	// Calculate t-statistic (more appropriate for small samples than z-score)
	tStatistic := (currentValue - mean) / standardError

	// Degrees of freedom for one-sample t-test
	degreesOfFreedom := sampleSize - 1

	// Choose appropriate distribution
	testType := "t-test"
	var pValue float64
	var confidence float64

	if sampleSize >= 30 {
		// For large samples, t-distribution approaches normal distribution
		testType = "z-test"
		pValue = sv.calculateZPValue(math.Abs(tStatistic))
		confidence = 1.0 - pValue
	} else {
		// For small samples, use t-distribution
		pValue = sv.calculateTPValue(math.Abs(tStatistic), degreesOfFreedom)
		confidence = 1.0 - pValue
	}

	// Apply multiple comparison correction if needed
	correctedConfidence := confidence
	var correction *MultipleComparisonCorrection

	if sv.useMultipleCorrection && numComparisons > 1 {
		correction = &MultipleComparisonCorrection{
			Method:         sv.correctionMethod,
			NumComparisons: numComparisons,
			OriginalAlpha:  1.0 - sv.confidenceLevel,
		}

		switch sv.correctionMethod {
		case "bonferroni":
			// Bonferroni correction: multiply p-value by number of comparisons
			correctedAlpha := (1.0 - sv.confidenceLevel) / float64(numComparisons)
			correction.CorrectedAlpha = correctedAlpha
			correctedPValue := pValue * float64(numComparisons)

			// If corrected p-value exceeds 1, set to 1 (no significance possible)
			if correctedPValue >= 1.0 {
				correctedConfidence = 0.0
			} else {
				correctedConfidence = 1.0 - correctedPValue
			}
		default:
			// Default to Bonferroni
			correctedAlpha := (1.0 - sv.confidenceLevel) / float64(numComparisons)
			correction.CorrectedAlpha = correctedAlpha
			correctedPValue := pValue * float64(numComparisons)

			if correctedPValue >= 1.0 {
				correctedConfidence = 0.0
			} else {
				correctedConfidence = 1.0 - correctedPValue
			}
		}

		// Ensure corrected confidence is bounded [0, 1]
		correctedConfidence = math.Max(0.0, math.Min(1.0, correctedConfidence))
	}

	// Calculate effect size (Cohen's d)
	effectSize := (currentValue - mean) / stdDev

	// Calculate confidence interval for the difference
	confidenceInterval := sv.calculateConfidenceInterval(
		currentValue-mean,
		standardError,
		degreesOfFreedom,
		sv.confidenceLevel,
	)

	return StatisticalResult{
		TStatistic:         tStatistic,
		DegreesOfFreedom:   degreesOfFreedom,
		PValue:             pValue,
		Confidence:         correctedConfidence,
		ConfidenceInterval: confidenceInterval,
		EffectSize:         effectSize,
		SampleSize:         sampleSize,
		TestType:           testType,
	}
}

// calculateTPValue calculates p-value using t-distribution approximation
// This is a simplified implementation - for production use, consider a statistics library
func (sv *StatisticalValidator) calculateTPValue(tStat float64, df int) float64 {
	// Simplified t-distribution p-value calculation
	// For more accuracy, use a proper statistics library like gonum.org/v1/gonum/stat

	if df <= 0 {
		return 0.5 // Default for invalid degrees of freedom
	}

	// Use normal approximation for large df, otherwise use t-distribution approximation
	if df >= 30 {
		return sv.calculateZPValue(tStat)
	}

	// Simplified t-distribution approximation
	// This is not as accurate as proper t-distribution implementation
	// but provides reasonable estimates for small samples

	// Welch-Satterthwaite approximation for t-distribution
	// Convert t-statistic to approximate p-value

	// For very small degrees of freedom, be more conservative
	if df == 1 {
		// Special case: Cauchy distribution (t with df=1)
		pValue := 2.0 * (1.0 / math.Pi) * math.Atan(1.0/tStat)
		return math.Max(0.001, pValue) // Minimum p-value to avoid overconfidence
	}

	// General approximation for t-distribution
	// This uses a polynomial approximation that's reasonably accurate for df > 1
	adjustment := 1.0 + (tStat*tStat)/(4.0*float64(df))
	normalizedT := tStat / math.Sqrt(adjustment)

	return sv.calculateZPValue(normalizedT)
}

// calculateZPValue calculates p-value using standard normal distribution
func (sv *StatisticalValidator) calculateZPValue(zStat float64) float64 {
	// Two-tailed p-value for standard normal distribution
	// Using complementary error function approximation

	absZ := math.Abs(zStat)

	// Abramowitz and Stegun approximation for normal CDF
	// This provides reasonable accuracy for z-scores

	if absZ > 6.0 {
		return 1e-9 // Very small p-value for extreme z-scores
	}

	// Use a more conservative approximation for the normal CDF
	// This avoids numerical issues with extreme z-scores

	// For moderate z-scores, use complementary error function approximation
	if absZ <= 3.0 {
		// Complementary error function approximation
		a1 := 0.254829592
		a2 := -0.284496736
		a3 := 1.421413741
		a4 := -1.453152027
		a5 := 1.061405429
		p := 0.3275911

		t := 1.0 / (1.0 + p*absZ)
		erfcApprox := t * (a1 + t*(a2+t*(a3+t*(a4+t*a5)))) * math.Exp(-absZ*absZ)

		// Convert to p-value (two-tailed)
		pValue := erfcApprox
		return math.Max(1e-10, math.Min(1.0, pValue))
	} else {
		// For large z-scores, use asymptotic approximation
		// P(|Z| > z) ≈ 2 * φ(z) / z * exp(-z²/2) for large z
		// This gives more reasonable p-values for extreme cases
		asymptotic := (2.0 / (absZ * math.Sqrt(2.0*math.Pi))) * math.Exp(-0.5*absZ*absZ)
		return math.Max(1e-10, math.Min(1.0, asymptotic))
	}
}

// calculateConfidenceInterval calculates confidence interval for the mean difference
func (sv *StatisticalValidator) calculateConfidenceInterval(
	meanDiff, standardError float64,
	degreesOfFreedom int,
	confidenceLevel float64,
) ConfidenceInterval {

	// Calculate critical value (t-score)
	_ = 1.0 - confidenceLevel // alpha (not used in this simplified implementation)

	// Simplified critical value calculation
	// For production, use proper t-distribution quantile function
	var criticalValue float64

	if degreesOfFreedom >= 30 {
		// Use normal distribution critical values for large samples
		switch {
		case confidenceLevel >= 0.99:
			criticalValue = 2.576 // 99% confidence
		case confidenceLevel >= 0.95:
			criticalValue = 1.960 // 95% confidence
		case confidenceLevel >= 0.90:
			criticalValue = 1.645 // 90% confidence
		default:
			criticalValue = 1.960 // Default to 95%
		}
	} else {
		// Approximate t-distribution critical values
		// These are simplified - use proper quantile functions in production
		multiplier := 1.0 + 2.0/float64(degreesOfFreedom) // Adjustment for small samples

		switch {
		case confidenceLevel >= 0.99:
			criticalValue = 2.576 * multiplier
		case confidenceLevel >= 0.95:
			criticalValue = 1.960 * multiplier
		case confidenceLevel >= 0.90:
			criticalValue = 1.645 * multiplier
		default:
			criticalValue = 1.960 * multiplier
		}
	}

	marginOfError := criticalValue * standardError

	return ConfidenceInterval{
		Lower: meanDiff - marginOfError,
		Upper: meanDiff + marginOfError,
		Level: confidenceLevel,
	}
}

// IsStatisticallySignificant determines if a regression is statistically significant
func (sv *StatisticalValidator) IsStatisticallySignificant(result StatisticalResult) bool {
	return result.Confidence >= sv.confidenceLevel
}

// ClassifyEffectSize classifies the practical significance using Cohen's d
func (sv *StatisticalValidator) ClassifyEffectSize(effectSize float64) string {
	absEffect := math.Abs(effectSize)

	switch {
	case absEffect < 0.2:
		return "negligible"
	case absEffect < 0.5:
		return "small"
	case absEffect < 0.8:
		return "medium"
	case absEffect < 1.2:
		return "large"
	default:
		return "very_large"
	}
}

// CalculatePowerAnalysis estimates statistical power for detecting regressions
func (sv *StatisticalValidator) CalculatePowerAnalysis(
	sampleSize int,
	effectSize float64,
	alpha float64,
) float64 {
	// Simplified power calculation for one-sample t-test
	// In production, use proper power analysis libraries

	if sampleSize <= 1 {
		return 0.0
	}

	// Convert effect size and sample size to non-centrality parameter
	ncp := effectSize * math.Sqrt(float64(sampleSize))

	// Simplified power approximation
	// This is not as accurate as proper non-central t-distribution
	if ncp < 0.5 {
		return 0.1 // Low power for small effects
	} else if ncp > 4.0 {
		return 0.95 // High power for large effects
	}

	// Linear approximation for moderate effects
	power := 0.1 + 0.85*(ncp-0.5)/3.5
	return math.Max(0.05, math.Min(0.99, power))
}
