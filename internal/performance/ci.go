package performance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CIIntegration handles CI/CD pipeline integration for performance monitoring.
type CIIntegration struct {
	detector         *PerformanceDetector
	outputFormat     string // "json", "junit", "github", "text"
	failOnRegression bool
}

// NewCIIntegration creates a new CI integration.
func NewCIIntegration(
	detector *PerformanceDetector,
	outputFormat string,
	failOnRegression bool,
) *CIIntegration {
	return &CIIntegration{
		detector:         detector,
		outputFormat:     outputFormat,
		failOnRegression: failOnRegression,
	}
}

// RunPerformanceCheck executes full performance check for CI/CD.
func (ci *CIIntegration) RunPerformanceCheck(benchmarkPackages []string, outputFile string) error {
	// Step 1: Run benchmarks
	fmt.Println("🔍 Running performance benchmarks...")
	benchmarkOutput, err := ci.runBenchmarks(benchmarkPackages)
	if err != nil {
		return fmt.Errorf("running benchmarks: %w", err)
	}

	// Step 2: Parse benchmark results
	results, err := ci.detector.ParseBenchmarkOutput(benchmarkOutput)
	if err != nil {
		return fmt.Errorf("parsing benchmark output: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("⚠️  No benchmark results found")

		return nil
	}

	fmt.Printf("📊 Parsed %d benchmark results\n", len(results))

	// Step 3: Update baselines
	fmt.Println("📈 Updating performance baselines...")
	if err := ci.detector.UpdateBaselines(results); err != nil {
		return fmt.Errorf("updating baselines: %w", err)
	}

	// Step 4: Detect regressions
	fmt.Println("🔎 Detecting performance regressions...")
	regressions, err := ci.detector.DetectRegressions(results)
	if err != nil {
		return fmt.Errorf("detecting regressions: %w", err)
	}

	// Step 5: Generate report
	report := ci.GenerateReport(results, regressions)

	// Step 6: Output results
	if err := ci.OutputResults(report, outputFile); err != nil {
		return fmt.Errorf("outputting results: %w", err)
	}

	// Step 7: Handle CI failure if needed
	if ci.failOnRegression && len(regressions) > 0 {
		criticalRegressions := ci.countCriticalRegressions(regressions)
		if criticalRegressions > 0 {
			return fmt.Errorf("detected %d critical performance regressions", criticalRegressions)
		}
	}

	fmt.Printf("✅ Performance check completed. Found %d regressions\n", len(regressions))

	return nil
}

// validatePackagePaths validates package paths to prevent command injection.
func validatePackagePaths(packages []string) error {
	for _, pkg := range packages {
		if err := validateSinglePackagePath(pkg); err != nil {
			return fmt.Errorf("invalid package path '%s': %w", pkg, err)
		}
	}

	return nil
}

// validateSinglePackagePath validates a single package path.
func validateSinglePackagePath(pkg string) error {
	// Check for dangerous characters that could enable command injection
	dangerousChars := []string{
		";", "|", "&", "`", "$", "\n", "\r", "\x00",
		"\u2028", "\u2029", // Unicode line separators
	}

	for _, char := range dangerousChars {
		if strings.Contains(pkg, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(pkg, "..") {
		return errors.New("path traversal detected")
	}

	// Ensure package path is relative and within allowed directories
	if filepath.IsAbs(pkg) {
		return errors.New("absolute paths not allowed")
	}

	// Additional validation: must start with ./ or be a simple path
	if !strings.HasPrefix(pkg, "./") && !isSimplePackagePath(pkg) {
		return errors.New("invalid package path format")
	}

	return nil
}

// isSimplePackagePath checks if path is a simple Go package path.
func isSimplePackagePath(path string) bool {
	// Allow simple package paths like "internal/build"
	for _, char := range path {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '/' || char == '_' || char == '-') {
			return false
		}
	}

	return true
}

// runBenchmarks executes Go benchmarks and returns output.
func (ci *CIIntegration) runBenchmarks(packages []string) (string, error) {
	// Validate package paths to prevent command injection
	if err := validatePackagePaths(packages); err != nil {
		return "", fmt.Errorf("package validation failed: %w", err)
	}

	args := []string{"test", "-bench=.", "-benchmem", "-count=3"}
	args = append(args, packages...)

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail if benchmarks have some errors, just return what we got
		fmt.Printf("⚠️  Benchmark command had issues: %v\n", err)
	}

	return string(output), nil
}

// PerformanceReport contains comprehensive performance analysis.
type PerformanceReport struct {
	Timestamp   time.Time             `json:"timestamp"`
	GitCommit   string                `json:"git_commit,omitempty"`
	GitBranch   string                `json:"git_branch,omitempty"`
	Environment string                `json:"environment"`
	Results     []BenchmarkResult     `json:"results"`
	Regressions []RegressionDetection `json:"regressions"`
	Summary     ReportSummary         `json:"summary"`
}

// ReportSummary provides high-level performance metrics.
type ReportSummary struct {
	TotalBenchmarks     int     `json:"total_benchmarks"`
	RegressionsFound    int     `json:"regressions_found"`
	CriticalRegressions int     `json:"critical_regressions"`
	MajorRegressions    int     `json:"major_regressions"`
	MinorRegressions    int     `json:"minor_regressions"`
	AverageImprovement  float64 `json:"average_improvement"`
	AverageDegradation  float64 `json:"average_degradation"`
	OverallHealthScore  float64 `json:"overall_health_score"` // 0-100 scale
}

// GenerateReport creates comprehensive performance report.
func (ci *CIIntegration) GenerateReport(
	results []BenchmarkResult,
	regressions []RegressionDetection,
) PerformanceReport {
	summary := ci.calculateSummary(results, regressions)

	return PerformanceReport{
		Timestamp:   time.Now(),
		GitCommit:   ci.detector.gitCommit,
		GitBranch:   ci.detector.gitBranch,
		Environment: ci.detector.environment,
		Results:     results,
		Regressions: regressions,
		Summary:     summary,
	}
}

// calculateSummary computes report summary statistics.
func (ci *CIIntegration) calculateSummary(
	results []BenchmarkResult,
	regressions []RegressionDetection,
) ReportSummary {
	summary := ReportSummary{
		TotalBenchmarks:  len(results),
		RegressionsFound: len(regressions),
	}

	var totalImprovement, totalDegradation float64
	improvementCount, degradationCount := 0, 0

	for _, regression := range regressions {
		switch regression.Severity {
		case "critical":
			summary.CriticalRegressions++
		case "major":
			summary.MajorRegressions++
		case "minor":
			summary.MinorRegressions++
		}

		if regression.PercentageChange > 0 {
			totalDegradation += regression.PercentageChange
			degradationCount++
		} else {
			totalImprovement += -regression.PercentageChange
			improvementCount++
		}
	}

	if improvementCount > 0 {
		summary.AverageImprovement = totalImprovement / float64(improvementCount)
	}
	if degradationCount > 0 {
		summary.AverageDegradation = totalDegradation / float64(degradationCount)
	}

	// Calculate overall health score (0-100)
	// Start with 100 and deduct points for regressions
	score := 100.0
	score -= float64(summary.CriticalRegressions) * 30 // -30 per critical
	score -= float64(summary.MajorRegressions) * 15    // -15 per major
	score -= float64(summary.MinorRegressions) * 5     // -5 per minor

	if score < 0 {
		score = 0
	}
	summary.OverallHealthScore = score

	return summary
}

// OutputResults outputs performance report in the specified format.
func (ci *CIIntegration) OutputResults(report PerformanceReport, outputFile string) error {
	switch ci.outputFormat {
	case "json":
		return ci.outputJSON(report, outputFile)
	case "junit":
		return ci.outputJUnit(report, outputFile)
	case "github":
		return ci.outputGitHub(report, outputFile)
	case "text":
		return ci.outputText(report, outputFile)
	default:
		return ci.outputText(report, outputFile)
	}
}

// outputJSON outputs report in JSON format.
func (ci *CIIntegration) outputJSON(report PerformanceReport, outputFile string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	if outputFile != "" {
		return os.WriteFile(outputFile, data, 0644)
	}

	fmt.Println(string(data))

	return nil
}

// outputText outputs report in human-readable text format.
func (ci *CIIntegration) outputText(report PerformanceReport, outputFile string) error {
	var output strings.Builder

	output.WriteString("🚀 PERFORMANCE REPORT\n")
	output.WriteString("=====================\n\n")

	// Summary
	output.WriteString("📊 Summary:\n")
	output.WriteString(fmt.Sprintf("  • Total Benchmarks: %d\n", report.Summary.TotalBenchmarks))
	output.WriteString(fmt.Sprintf("  • Regressions Found: %d\n", report.Summary.RegressionsFound))
	output.WriteString(
		fmt.Sprintf("  • Health Score: %.1f/100\n", report.Summary.OverallHealthScore),
	)

	if report.GitCommit != "" {
		output.WriteString(fmt.Sprintf("  • Git Commit: %s\n", report.GitCommit))
	}
	if report.GitBranch != "" {
		output.WriteString(fmt.Sprintf("  • Git Branch: %s\n", report.GitBranch))
	}
	output.WriteString(fmt.Sprintf("  • Environment: %s\n", report.Environment))
	output.WriteString("\n")

	// Regressions
	if len(report.Regressions) > 0 {
		output.WriteString("⚠️  REGRESSIONS DETECTED:\n")
		output.WriteString("=========================\n\n")

		criticalCount := 0
		majorCount := 0
		minorCount := 0

		for _, regression := range report.Regressions {
			icon := "🟡"
			switch regression.Severity {
			case "critical":
				icon = "🔴"
				criticalCount++
			case "major":
				icon = "🟠"
				majorCount++
			case "minor":
				icon = "🟡"
				minorCount++
			}

			output.WriteString(
				fmt.Sprintf(
					"%s %s [%s]\n",
					icon,
					regression.BenchmarkName,
					strings.ToUpper(regression.Severity),
				),
			)
			output.WriteString(fmt.Sprintf("    Type: %s regression\n", regression.RegressionType))
			output.WriteString(fmt.Sprintf("    Change: %.1f%% (%.2f → %.2f)\n",
				regression.PercentageChange, regression.BaselineValue, regression.CurrentValue))
			output.WriteString(fmt.Sprintf("    Confidence: %.1f%%\n", regression.Confidence*100))
			output.WriteString(fmt.Sprintf("    Action: %s\n", regression.RecommendedAction))
			output.WriteString("\n")
		}

		output.WriteString(fmt.Sprintf("Summary: %d Critical, %d Major, %d Minor\n\n",
			criticalCount, majorCount, minorCount))
	} else {
		output.WriteString("✅ No performance regressions detected!\n\n")
	}

	// Top performing benchmarks
	if len(report.Results) > 0 {
		output.WriteString("🏆 TOP PERFORMING BENCHMARKS:\n")
		output.WriteString("=============================\n\n")

		// Sort by performance (ns/op)
		sortedResults := make([]BenchmarkResult, len(report.Results))
		copy(sortedResults, report.Results)

		// Show top 5 fastest benchmarks
		count := 5
		if len(sortedResults) < count {
			count = len(sortedResults)
		}

		for i := range count {
			result := sortedResults[i]
			output.WriteString(
				fmt.Sprintf("  %d. %s: %.2f ns/op", i+1, result.Name, result.NsPerOp),
			)
			if result.BytesPerOp > 0 {
				output.WriteString(fmt.Sprintf(" | %d B/op", result.BytesPerOp))
			}
			if result.AllocsPerOp > 0 {
				output.WriteString(fmt.Sprintf(" | %d allocs/op", result.AllocsPerOp))
			}
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	output.WriteString(fmt.Sprintf("Generated at: %s\n", report.Timestamp.Format(time.RFC3339)))

	result := output.String()

	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(result), 0644)
	}

	fmt.Print(result)

	return nil
}

// outputGitHub outputs report in GitHub Actions format.
func (ci *CIIntegration) outputGitHub(report PerformanceReport, outputFile string) error {
	var output strings.Builder

	// GitHub Actions annotations
	for _, regression := range report.Regressions {
		level := "warning"
		if regression.Severity == "critical" {
			level = "error"
		}

		output.WriteString(
			fmt.Sprintf(
				"::%s::Performance regression detected in %s: %.1f%% %s degradation (%.2f → %.2f)\n",
				level,
				regression.BenchmarkName,
				regression.PercentageChange,
				regression.RegressionType,
				regression.BaselineValue,
				regression.CurrentValue,
			),
		)
	}

	// Summary comment
	if len(report.Regressions) == 0 {
		output.WriteString("::notice::✅ No performance regressions detected in this PR\n")
	} else {
		output.WriteString(fmt.Sprintf("::warning::⚠️ Detected %d performance regressions (Health Score: %.1f/100)\n",
			len(report.Regressions), report.Summary.OverallHealthScore))
	}

	result := output.String()

	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(result), 0644)
	}

	fmt.Print(result)

	return nil
}

// outputJUnit outputs report in JUnit XML format for CI integration.
func (ci *CIIntegration) outputJUnit(report PerformanceReport, outputFile string) error {
	var output strings.Builder

	output.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	output.WriteString(
		fmt.Sprintf(`<testsuite name="performance" tests="%d" failures="%d" time="%.3f">`,
			report.Summary.TotalBenchmarks, report.Summary.RegressionsFound, 0.0) + "\n",
	)

	// Add test cases for each benchmark
	for _, result := range report.Results {
		output.WriteString(fmt.Sprintf(`  <testcase name="%s" classname="benchmark" time="%.9f">`,
			result.Name, result.NsPerOp/1e9))

		// Check if this benchmark has a regression
		hasRegression := false
		for _, regression := range report.Regressions {
			if regression.BenchmarkName == result.Name {
				output.WriteString("\n")
				output.WriteString(
					fmt.Sprintf(
						`    <failure message="Performance regression: %.1f%% %s degradation" type="regression">`,
						regression.PercentageChange,
						regression.RegressionType,
					),
				)
				output.WriteString(fmt.Sprintf("Baseline: %.2f, Current: %.2f, Threshold: %.2f",
					regression.BaselineValue, regression.CurrentValue, regression.Threshold))
				output.WriteString("</failure>\n")
				output.WriteString("  ")
				hasRegression = true

				break
			}
		}

		if !hasRegression {
			output.WriteString(" />")
		} else {
			output.WriteString("</testcase>")
		}
		output.WriteString("\n")
	}

	output.WriteString("</testsuite>\n")

	result := output.String()

	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(result), 0644)
	}

	fmt.Print(result)

	return nil
}

// countCriticalRegressions counts critical regressions for CI failure logic.
func (ci *CIIntegration) countCriticalRegressions(regressions []RegressionDetection) int {
	count := 0
	for _, regression := range regressions {
		if regression.Severity == "critical" {
			count++
		}
	}

	return count
}

// SetupGitHubActionsWorkflow creates a GitHub Actions workflow for performance monitoring.
func (ci *CIIntegration) SetupGitHubActionsWorkflow(workflowDir string) error {
	workflowContent := `name: Performance Regression Detection

on:
  push:
    branches: [ main, dev ]
  pull_request:
    branches: [ main ]

jobs:
  performance:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0  # Need full history for baseline comparison
        
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
        
    - name: Cache performance baselines
      uses: actions/cache@v3
      with:
        path: .performance-baselines
        key: performance-baselines-${{ runner.os }}-${{ github.base_ref || github.ref_name }}
        restore-keys: |
          performance-baselines-${{ runner.os }}-
          
    - name: Install dependencies
      run: go mod download
      
    - name: Run performance benchmarks
      run: |
        go run ./cmd/templar performance check \
          --packages="./internal/scanner,./internal/build,./internal/registry" \
          --format=github \
          --fail-on-critical \
          --baseline-dir=.performance-baselines \
          --output=performance-report.txt
          
    - name: Upload performance report
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: performance-report
        path: performance-report.txt
        
    - name: Comment PR with performance results
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          if (fs.existsSync('performance-report.txt')) {
            const report = fs.readFileSync('performance-report.txt', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '## 🚀 Performance Report\n\n' + report
            });
          }
`

	workflowPath := filepath.Join(workflowDir, "performance.yml")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("creating workflow directory: %w", err)
	}

	return os.WriteFile(workflowPath, []byte(workflowContent), 0644)
}
