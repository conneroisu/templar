package performance

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCIIntegration_GenerateReport(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "json", false)

	results := []BenchmarkResult{
		{
			Name:        "TestBenchmark1",
			NsPerOp:     1000.0,
			BytesPerOp:  500,
			AllocsPerOp: 10,
			Timestamp:   time.Now(),
		},
		{
			Name:        "TestBenchmark2",
			NsPerOp:     2000.0,
			BytesPerOp:  1000,
			AllocsPerOp: 20,
			Timestamp:   time.Now(),
		},
	}

	regressions := []RegressionDetection{
		{
			BenchmarkName:     "TestBenchmark1",
			IsRegression:      true,
			CurrentValue:      1300.0,
			BaselineValue:     1000.0,
			PercentageChange:  30.0,
			RegressionType:    "performance",
			Severity:          "major",
			RecommendedAction: "Review recent commits",
		},
	}

	report := ci.GenerateReport(results, regressions)

	if report.Summary.TotalBenchmarks != 2 {
		t.Errorf("Expected 2 total benchmarks, got %d", report.Summary.TotalBenchmarks)
	}

	if report.Summary.RegressionsFound != 1 {
		t.Errorf("Expected 1 regression found, got %d", report.Summary.RegressionsFound)
	}

	if report.Summary.MajorRegressions != 1 {
		t.Errorf("Expected 1 major regression, got %d", report.Summary.MajorRegressions)
	}

	if len(report.Results) != 2 {
		t.Errorf("Expected 2 results in report, got %d", len(report.Results))
	}

	if len(report.Regressions) != 1 {
		t.Errorf("Expected 1 regression in report, got %d", len(report.Regressions))
	}

	// Health score should be reduced due to major regression
	expectedHealthScore := 100.0 - 15.0 // 100 - (1 major * 15 points)
	if report.Summary.OverallHealthScore != expectedHealthScore {
		t.Errorf("Expected health score %f, got %f", expectedHealthScore, report.Summary.OverallHealthScore)
	}
}

func TestCIIntegration_OutputJSON(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "json", false)

	report := PerformanceReport{
		Timestamp: time.Now(),
		Results: []BenchmarkResult{
			{Name: "Test", NsPerOp: 1000.0},
		},
		Regressions: []RegressionDetection{},
		Summary: ReportSummary{
			TotalBenchmarks:    1,
			RegressionsFound:   0,
			OverallHealthScore: 100.0,
		},
	}

	tempFile := t.TempDir() + "/report.json"
	err := ci.outputJSON(report, tempFile)
	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	// Verify file was created and contains valid JSON
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var parsedReport PerformanceReport
	if err := json.Unmarshal(data, &parsedReport); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsedReport.Summary.TotalBenchmarks != 1 {
		t.Errorf("Expected 1 total benchmark in parsed report, got %d", parsedReport.Summary.TotalBenchmarks)
	}
}

func TestCIIntegration_OutputText(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "text", false)

	report := PerformanceReport{
		Timestamp:   time.Now(),
		GitCommit:   "abc123",
		GitBranch:   "main",
		Environment: "ci",
		Results: []BenchmarkResult{
			{Name: "FastBenchmark", NsPerOp: 100.0, BytesPerOp: 50, AllocsPerOp: 1},
			{Name: "SlowBenchmark", NsPerOp: 5000.0, BytesPerOp: 2000, AllocsPerOp: 50},
		},
		Regressions: []RegressionDetection{
			{
				BenchmarkName:     "SlowBenchmark",
				IsRegression:      true,
				CurrentValue:      5000.0,
				BaselineValue:     4000.0,
				PercentageChange:  25.0,
				RegressionType:    "performance",
				Severity:          "major",
				Confidence:        0.95,
				RecommendedAction: "Review recent commits for performance impact",
			},
		},
		Summary: ReportSummary{
			TotalBenchmarks:    2,
			RegressionsFound:   1,
			MajorRegressions:   1,
			OverallHealthScore: 85.0,
			AverageDegradation: 25.0,
		},
	}

	tempFile := t.TempDir() + "/report.txt"
	err := ci.outputText(report, tempFile)
	if err != nil {
		t.Fatalf("outputText failed: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	text := string(content)

	// Check for expected content
	expectedStrings := []string{
		"🚀 PERFORMANCE REPORT",
		"Total Benchmarks: 2",
		"Regressions Found: 1",
		"Health Score: 85.0/100",
		"Git Commit: abc123",
		"Git Branch: main",
		"Environment: ci",
		"⚠️  REGRESSIONS DETECTED",
		"🟠 SlowBenchmark [MAJOR]",
		"Type: performance regression",
		"Change: 25.0% (4000.00 → 5000.00)",
		"Confidence: 95.0%",
		"Action: Review recent commits",
		"Summary: 0 Critical, 1 Major, 0 Minor",
		"🏆 TOP PERFORMING BENCHMARKS",
		"FastBenchmark: 100.00 ns/op",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(text, expected) {
			t.Errorf("Expected to find '%s' in output, but it was missing", expected)
		}
	}
}

func TestCIIntegration_OutputGitHub(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "github", false)

	report := PerformanceReport{
		Regressions: []RegressionDetection{
			{
				BenchmarkName:    "CriticalBenchmark",
				IsRegression:     true,
				CurrentValue:     2000.0,
				BaselineValue:    1000.0,
				PercentageChange: 100.0,
				RegressionType:   "performance",
				Severity:         "critical",
			},
			{
				BenchmarkName:    "MajorBenchmark",
				IsRegression:     true,
				CurrentValue:     1500.0,
				BaselineValue:    1200.0,
				PercentageChange: 25.0,
				RegressionType:   "memory",
				Severity:         "major",
			},
		},
		Summary: ReportSummary{
			RegressionsFound:   2,
			OverallHealthScore: 55.0,
		},
	}

	tempFile := t.TempDir() + "/github-output.txt"
	err := ci.outputGitHub(report, tempFile)
	if err != nil {
		t.Fatalf("outputGitHub failed: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	text := string(content)

	// Check for GitHub Actions annotations
	expectedAnnotations := []string{
		"::error::Performance regression detected in CriticalBenchmark: 100.0% performance degradation",
		"::warning::Performance regression detected in MajorBenchmark: 25.0% memory degradation",
		"::warning::⚠️ Detected 2 performance regressions (Health Score: 55.0/100)",
	}

	for _, expected := range expectedAnnotations {
		if !strings.Contains(text, expected) {
			t.Errorf("Expected to find '%s' in GitHub output, but it was missing", expected)
		}
	}
}

func TestCIIntegration_OutputJUnit(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "junit", false)

	report := PerformanceReport{
		Results: []BenchmarkResult{
			{Name: "PassingBenchmark", NsPerOp: 100.0},
			{Name: "FailingBenchmark", NsPerOp: 2000.0},
		},
		Regressions: []RegressionDetection{
			{
				BenchmarkName:    "FailingBenchmark",
				IsRegression:     true,
				CurrentValue:     2000.0,
				BaselineValue:    1000.0,
				PercentageChange: 100.0,
				RegressionType:   "performance",
				Threshold:        1.15,
			},
		},
		Summary: ReportSummary{
			TotalBenchmarks:  2,
			RegressionsFound: 1,
		},
	}

	tempFile := t.TempDir() + "/junit.xml"
	err := ci.outputJUnit(report, tempFile)
	if err != nil {
		t.Fatalf("outputJUnit failed: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	text := string(content)

	// Check for JUnit XML structure
	expectedElements := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<testsuite name="performance" tests="2" failures="1"`,
		`<testcase name="PassingBenchmark" classname="benchmark"`,
		`<testcase name="FailingBenchmark" classname="benchmark"`,
		`<failure message="Performance regression: 100.0% performance degradation" type="regression">`,
		`Baseline: 1000.00, Current: 2000.00, Threshold: 1.15`,
		`</testsuite>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(text, expected) {
			t.Errorf("Expected to find '%s' in JUnit output, but it was missing", expected)
		}
	}
}

func TestCIIntegration_CalculateSummary(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "text", false)

	results := []BenchmarkResult{
		{Name: "Test1", NsPerOp: 100.0},
		{Name: "Test2", NsPerOp: 200.0},
		{Name: "Test3", NsPerOp: 300.0},
	}

	regressions := []RegressionDetection{
		{Severity: "critical", PercentageChange: 50.0},
		{Severity: "major", PercentageChange: 30.0},
		{Severity: "major", PercentageChange: 25.0},
		{Severity: "minor", PercentageChange: 10.0},
		{Severity: "minor", PercentageChange: -5.0}, // Improvement
	}

	summary := ci.calculateSummary(results, regressions)

	if summary.TotalBenchmarks != 3 {
		t.Errorf("Expected 3 total benchmarks, got %d", summary.TotalBenchmarks)
	}

	if summary.RegressionsFound != 5 {
		t.Errorf("Expected 5 regressions found, got %d", summary.RegressionsFound)
	}

	if summary.CriticalRegressions != 1 {
		t.Errorf("Expected 1 critical regression, got %d", summary.CriticalRegressions)
	}

	if summary.MajorRegressions != 2 {
		t.Errorf("Expected 2 major regressions, got %d", summary.MajorRegressions)
	}

	if summary.MinorRegressions != 2 {
		t.Errorf("Expected 2 minor regressions, got %d", summary.MinorRegressions)
	}

	// Health score calculation: 100 - (1*30 + 2*15 + 2*5) = 100 - 70 = 30
	expectedHealthScore := 30.0
	if summary.OverallHealthScore != expectedHealthScore {
		t.Errorf("Expected health score %f, got %f", expectedHealthScore, summary.OverallHealthScore)
	}

	// Average degradation: (50 + 30 + 25 + 10) / 4 = 28.75
	expectedAvgDegradation := 28.75
	if summary.AverageDegradation != expectedAvgDegradation {
		t.Errorf("Expected average degradation %f, got %f", expectedAvgDegradation, summary.AverageDegradation)
	}

	// Average improvement: 5.0 (only one improvement)
	expectedAvgImprovement := 5.0
	if summary.AverageImprovement != expectedAvgImprovement {
		t.Errorf("Expected average improvement %f, got %f", expectedAvgImprovement, summary.AverageImprovement)
	}
}

func TestCIIntegration_CountCriticalRegressions(t *testing.T) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "text", true)

	regressions := []RegressionDetection{
		{Severity: "critical"},
		{Severity: "major"},
		{Severity: "critical"},
		{Severity: "minor"},
		{Severity: "critical"},
	}

	count := ci.countCriticalRegressions(regressions)
	if count != 3 {
		t.Errorf("Expected 3 critical regressions, got %d", count)
	}
}

// Benchmark the CI integration components
func BenchmarkCIIntegration_GenerateReport(b *testing.B) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "json", false)

	results := make([]BenchmarkResult, 100)
	for i := 0; i < 100; i++ {
		results[i] = BenchmarkResult{
			Name:    "Benchmark" + string(rune(i)),
			NsPerOp: float64(1000 + i*10),
		}
	}

	regressions := make([]RegressionDetection, 10)
	for i := 0; i < 10; i++ {
		regressions[i] = RegressionDetection{
			BenchmarkName:    "Benchmark" + string(rune(i)),
			IsRegression:     true,
			PercentageChange: float64(10 + i*5),
			Severity:         "major",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ci.GenerateReport(results, regressions)
	}
}

func BenchmarkCIIntegration_OutputText(b *testing.B) {
	detector := NewPerformanceDetector("test", DefaultThresholds())
	ci := NewCIIntegration(detector, "text", false)

	report := PerformanceReport{
		Results:     make([]BenchmarkResult, 50),
		Regressions: make([]RegressionDetection, 5),
		Summary: ReportSummary{
			TotalBenchmarks:    50,
			RegressionsFound:   5,
			OverallHealthScore: 75.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ci.outputText(report, "")
	}
}
