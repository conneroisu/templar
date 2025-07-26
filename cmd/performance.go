package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/conneroisu/templar/internal/performance"
	"github.com/spf13/cobra"
)

var (
	performancePackages   []string
	performanceFormat     string
	performanceOutput     string
	performanceBaseline   string
	performanceFailOn     bool
	performanceThresholds performance.RegressionThresholds
)

// performanceCmd represents the performance command
var performanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Performance monitoring and regression detection",
	Long: `Performance monitoring tools for detecting regressions and maintaining 
performance baselines. Includes benchmark execution, regression detection,
and CI/CD integration capabilities.`,
}

// performanceCheckCmd handles performance regression detection
var performanceCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Run performance check and detect regressions",
	Long: `Executes performance benchmarks, compares against baselines,
and detects performance regressions with configurable thresholds.`,
	RunE: runPerformanceCheck,
}

// performanceBaselineCmd manages performance baselines
var performanceBaselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Manage performance baselines",
	Long: `Commands for managing performance baselines including creation,
updates, and historical analysis.`,
}

// performanceBaselineCreateCmd creates new performance baselines
var performanceBaselineCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new performance baselines",
	Long: `Creates new performance baselines by running benchmarks and
establishing reference performance metrics.`,
	RunE: runPerformanceBaselineCreate,
}

// performanceBaselineListCmd lists existing baselines
var performanceBaselineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List performance baselines",
	Long: `Lists all existing performance baselines with statistics
and last update timestamps.`,
	RunE: runPerformanceBaselineList,
}

// performanceReportCmd generates performance reports
var performanceReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate performance reports",
	Long: `Generates comprehensive performance reports from historical
benchmark data and regression analysis.`,
	RunE: runPerformanceReport,
}

func init() {
	rootCmd.AddCommand(performanceCmd)

	// Add subcommands
	performanceCmd.AddCommand(performanceCheckCmd)
	performanceCmd.AddCommand(performanceBaselineCmd)
	performanceCmd.AddCommand(performanceReportCmd)

	performanceBaselineCmd.AddCommand(performanceBaselineCreateCmd)
	performanceBaselineCmd.AddCommand(performanceBaselineListCmd)

	// Global performance flags
	performanceCmd.PersistentFlags().
		StringSliceVar(&performancePackages, "packages", []string{"./..."}, "Go packages to benchmark")
	performanceCmd.PersistentFlags().
		StringVar(&performanceFormat, "format", "text", "Output format (text, json, junit, github)")
	performanceCmd.PersistentFlags().
		StringVar(&performanceOutput, "output", "", "Output file (defaults to stdout)")
	performanceCmd.PersistentFlags().
		StringVar(&performanceBaseline, "baseline-dir", ".performance-baselines", "Directory to store performance baselines")

	// Performance check specific flags
	performanceCheckCmd.Flags().
		BoolVar(&performanceFailOn, "fail-on-critical", false, "Fail CI on critical regressions")

	// Threshold configuration flags
	performanceCheckCmd.Flags().
		Float64Var(&performanceThresholds.SlownessThreshold, "slowness-threshold", 1.15, "Performance degradation threshold (e.g., 1.15 = 15% slower)")
	performanceCheckCmd.Flags().
		Float64Var(&performanceThresholds.MemoryThreshold, "memory-threshold", 1.20, "Memory usage increase threshold (e.g., 1.20 = 20% more memory)")
	performanceCheckCmd.Flags().
		Float64Var(&performanceThresholds.AllocThreshold, "alloc-threshold", 1.25, "Allocation increase threshold (e.g., 1.25 = 25% more allocations)")
	performanceCheckCmd.Flags().
		IntVar(&performanceThresholds.MinSamples, "min-samples", 5, "Minimum samples required for regression detection")
	performanceCheckCmd.Flags().
		Float64Var(&performanceThresholds.ConfidenceLevel, "confidence-level", 0.95, "Statistical confidence level (e.g., 0.95 = 95%)")

	// Set default thresholds
	performanceThresholds = performance.DefaultThresholds()
}

func runPerformanceCheck(cmd *cobra.Command, args []string) error {
	// Create performance detector
	detector := performance.NewPerformanceDetector(performanceBaseline, performanceThresholds)

	// Set git information if available
	if commit, err := getGitCommit(); err == nil {
		if branch, err := getGitBranch(); err == nil {
			detector.SetGitInfo(commit, branch)
		}
	}

	// Create CI integration
	ci := performance.NewCIIntegration(detector, performanceFormat, performanceFailOn)

	// Run performance check
	return ci.RunPerformanceCheck(performancePackages, performanceOutput)
}

func runPerformanceBaselineCreate(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Creating performance baselines...")

	// Create performance detector
	detector := performance.NewPerformanceDetector(performanceBaseline, performanceThresholds)

	// Run benchmarks
	fmt.Println("📊 Running benchmarks to establish baselines...")
	benchmarkOutput, err := runBenchmarks(performancePackages)
	if err != nil {
		return fmt.Errorf("running benchmarks: %w", err)
	}

	// Parse results
	results, err := detector.ParseBenchmarkOutput(benchmarkOutput)
	if err != nil {
		return fmt.Errorf("parsing benchmark output: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no benchmark results found")
	}

	// Update baselines
	if err := detector.UpdateBaselines(results); err != nil {
		return fmt.Errorf("updating baselines: %w", err)
	}

	fmt.Printf("✅ Created baselines for %d benchmarks in %s\n", len(results), performanceBaseline)
	return nil
}

func runPerformanceBaselineList(cmd *cobra.Command, args []string) error {
	// List baseline files
	entries, err := os.ReadDir(performanceBaseline)
	if err != nil {
		return fmt.Errorf("reading baseline directory: %w", err)
	}

	fmt.Printf("📊 Performance Baselines in %s:\n\n", performanceBaseline)

	if len(entries) == 0 {
		fmt.Println("No baselines found. Run 'templar performance baseline create' to create them.")
		return nil
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		benchmarkName := strings.TrimSuffix(entry.Name(), ".json")
		fmt.Printf("  📈 %s\n", benchmarkName)
		fmt.Printf("      Last updated: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		fmt.Printf("      Size: %d bytes\n\n", info.Size())
	}

	return nil
}

func runPerformanceReport(cmd *cobra.Command, args []string) error {
	// Create performance detector
	detector := performance.NewPerformanceDetector(performanceBaseline, performanceThresholds)

	// Set git information if available
	if commit, err := getGitCommit(); err == nil {
		if branch, err := getGitBranch(); err == nil {
			detector.SetGitInfo(commit, branch)
		}
	}

	// Run fresh benchmarks for the report
	fmt.Println("🔍 Running benchmarks for performance report...")
	benchmarkOutput, err := runBenchmarks(performancePackages)
	if err != nil {
		return fmt.Errorf("running benchmarks: %w", err)
	}

	// Parse results
	results, err := detector.ParseBenchmarkOutput(benchmarkOutput)
	if err != nil {
		return fmt.Errorf("parsing benchmark output: %w", err)
	}

	// Detect regressions
	regressions, err := detector.DetectRegressions(results)
	if err != nil {
		return fmt.Errorf("detecting regressions: %w", err)
	}

	// Create CI integration for report generation
	ci := performance.NewCIIntegration(detector, performanceFormat, false)
	report := ci.GenerateReport(results, regressions)

	// Output report
	return ci.OutputResults(report, performanceOutput)
}

// Helper functions

func runBenchmarks(packages []string) (string, error) {
	args := []string{"test", "-bench=.", "-benchmem", "-count=3"}
	args = append(args, packages...)

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail completely if benchmarks have some issues
		fmt.Printf("⚠️  Benchmark command had issues: %v\n", err)
	}

	return string(output), nil
}

func getGitCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
