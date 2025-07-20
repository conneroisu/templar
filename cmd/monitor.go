package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/performance"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Performance monitoring and optimization",
	Long: `Performance monitoring and optimization commands for Templar.

The monitor command provides tools for tracking system performance,
analyzing metrics, and applying optimization recommendations.`,
}

var monitorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start performance monitoring",
	Long: `Start the performance monitoring system.

This will begin collecting system metrics and generating optimization
recommendations based on performance thresholds.`,
	RunE: runMonitorStart,
}

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show performance monitoring status",
	Long: `Show the current status of performance monitoring.

This displays current metrics, recent recommendations, and system health.`,
	RunE: runMonitorStatus,
}

var monitorReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate performance report",
	Long: `Generate a comprehensive performance report.

This creates a detailed analysis of system performance over a specified
time period, including metrics summaries and recommendations.`,
	RunE: runMonitorReport,
}

var monitorRecommendationsCmd = &cobra.Command{
	Use:   "recommendations",
	Short: "Show performance recommendations",
	Long: `Show current performance optimization recommendations.

This lists active recommendations with their priorities and suggested actions.`,
	RunE: runMonitorRecommendations,
}

var monitorMetricsCmd = &cobra.Command{
	Use:   "metrics [metric-type]",
	Short: "Show performance metrics",
	Long: `Show performance metrics for a specific type or all types.

Available metric types:
  - build_time: Build operation duration
  - memory_usage: Memory consumption
  - cpu_usage: CPU utilization
  - goroutines: Number of active goroutines
  - file_watchers: File watcher activity
  - component_scan: Component scanning performance
  - server_latency: HTTP request latency
  - cache_hit_rate: Cache effectiveness
  - error_rate: Error frequency`,
	RunE: runMonitorMetrics,
}

// Command flags
var (
	monitorInterval      time.Duration
	monitorOutput        string
	monitorSince         string
	monitorFormat        string
	monitorAutoOptimize  bool
	monitorThreshold     float64
	monitorFollowMode    bool
)

func init() {
	rootCmd.AddCommand(monitorCmd)
	
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorReportCmd)
	monitorCmd.AddCommand(monitorRecommendationsCmd)
	monitorCmd.AddCommand(monitorMetricsCmd)

	// Start command flags
	monitorStartCmd.Flags().DurationVar(&monitorInterval, "interval", 30*time.Second, 
		"Monitoring interval")
	monitorStartCmd.Flags().BoolVar(&monitorAutoOptimize, "auto-optimize", true, 
		"Enable automatic optimization recommendations")
	monitorStartCmd.Flags().BoolVar(&monitorFollowMode, "follow", false, 
		"Follow mode - keep monitoring and displaying updates")

	// Report command flags
	monitorReportCmd.Flags().StringVar(&monitorSince, "since", "1h", 
		"Time period for report (e.g., 1h, 24h, 7d)")
	monitorReportCmd.Flags().StringVar(&monitorOutput, "output", "", 
		"Output file for report (default: stdout)")
	monitorReportCmd.Flags().StringVar(&monitorFormat, "format", "table", 
		"Output format: table, json, yaml")

	// Metrics command flags
	monitorMetricsCmd.Flags().StringVar(&monitorSince, "since", "1h", 
		"Time period for metrics")
	monitorMetricsCmd.Flags().StringVar(&monitorFormat, "format", "table", 
		"Output format: table, json, yaml")
	
	// Recommendations command flags
	monitorRecommendationsCmd.Flags().StringVar(&monitorFormat, "format", "table", 
		"Output format: table, json, yaml")
	monitorRecommendationsCmd.Flags().Float64Var(&monitorThreshold, "priority-threshold", 0, 
		"Minimum priority threshold for recommendations")
}

func runMonitorStart(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create performance monitor
	monitor := performance.NewPerformanceMonitor(monitorInterval)
	integration := performance.NewPerformanceIntegration(monitor)

	// Start monitoring
	monitor.Start()
	defer monitor.Stop()

	fmt.Printf("🔍 Performance monitoring started (interval: %v)\n", monitorInterval)
	
	if monitorAutoOptimize {
		fmt.Println("✨ Auto-optimization enabled")
	}

	if monitorFollowMode {
		return runFollowMode(monitor, integration)
	}

	// Just start and return
	fmt.Println("📊 Monitoring running in background. Use 'templar monitor status' to check.")
	return nil
}

func runFollowMode(monitor *performance.PerformanceMonitor, integration *performance.PerformanceIntegration) error {
	fmt.Println("📈 Following performance metrics (Ctrl+C to stop)...")
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	recommendations := monitor.GetRecommendations()

	for {
		select {
		case <-ticker.C:
			displayCurrentMetrics(monitor)
		case rec := <-recommendations:
			displayRecommendation(rec)
			if monitorAutoOptimize && rec.Confidence > 0.8 {
				fmt.Printf("🤖 Auto-applying recommendation: %s\n", rec.Description)
				if err := integration.ApplyRecommendation(rec); err != nil {
					fmt.Printf("❌ Failed to apply recommendation: %v\n", err)
				} else {
					fmt.Printf("✅ Recommendation applied successfully\n")
				}
			}
		}
	}
}

func runMonitorStatus(cmd *cobra.Command, args []string) error {
	// For this demo, we'll create a temporary monitor to show status
	monitor := performance.NewPerformanceMonitor(time.Minute)
	
	fmt.Println("📊 Performance Monitor Status")
	fmt.Println("==============================")
	
	// Show recent metrics
	displayCurrentMetrics(monitor)
	
	return nil
}

func runMonitorReport(cmd *cobra.Command, args []string) error {
	duration, err := parseDuration(monitorSince)
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", monitorSince, err)
	}

	since := time.Now().Add(-duration)
	
	// Create monitor and integration
	monitor := performance.NewPerformanceMonitor(time.Minute)
	integration := performance.NewPerformanceIntegration(monitor)
	
	// Generate report
	report := integration.GetPerformanceReport(since)
	
	// Output report
	switch monitorFormat {
	case "json":
		return outputJSON(report, monitorOutput)
	case "yaml":
		return outputYAML(report, monitorOutput)
	default:
		return outputReportTable(report, monitorOutput)
	}
}

func runMonitorRecommendations(cmd *cobra.Command, args []string) error {
	monitor := performance.NewPerformanceMonitor(time.Minute)
	
	fmt.Println("💡 Performance Recommendations")
	fmt.Println("===============================")
	
	// Try to get recommendations (with timeout)
	timeout := time.After(1 * time.Second)
	recommendations := []performance.Recommendation{}
	
	for len(recommendations) < 10 {
		select {
		case rec := <-monitor.GetRecommendations():
			if rec.Priority >= int(monitorThreshold) {
				recommendations = append(recommendations, rec)
			}
		case <-timeout:
			goto display
		}
	}

display:
	if len(recommendations) == 0 {
		fmt.Println("✅ No active recommendations")
		return nil
	}

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	switch monitorFormat {
	case "json":
		return outputJSON(recommendations, monitorOutput)
	case "yaml":
		return outputYAML(recommendations, monitorOutput)
	default:
		return displayRecommendationsTable(recommendations)
	}
}

func runMonitorMetrics(cmd *cobra.Command, args []string) error {
	duration, err := parseDuration(monitorSince)
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", monitorSince, err)
	}

	since := time.Now().Add(-duration)
	monitor := performance.NewPerformanceMonitor(time.Minute)

	var metricType performance.MetricType
	if len(args) > 0 {
		metricType = performance.MetricType(args[0])
	}

	metrics := monitor.GetMetrics(metricType, since)
	
	switch monitorFormat {
	case "json":
		return outputJSON(metrics, monitorOutput)
	case "yaml":
		return outputYAML(metrics, monitorOutput)
	default:
		return displayMetricsTable(metrics, metricType)
	}
}

// Display functions
func displayCurrentMetrics(monitor *performance.PerformanceMonitor) {
	fmt.Printf("\r📈 [%s] ", time.Now().Format("15:04:05"))
	
	// Get recent aggregates
	memAgg := monitor.GetAggregate(performance.MetricTypeMemoryUsage)
	if memAgg != nil {
		fmt.Printf("Mem: %.1fMB ", memAgg.Avg/(1024*1024))
	}
	
	goroutineAgg := monitor.GetAggregate(performance.MetricTypeGoroutines)
	if goroutineAgg != nil {
		fmt.Printf("Goroutines: %.0f ", goroutineAgg.Avg)
	}
	
	buildAgg := monitor.GetAggregate(performance.MetricTypeBuildTime)
	if buildAgg != nil {
		fmt.Printf("Build: %.0fms ", buildAgg.Avg)
	}
	
	fmt.Print("                    ") // Clear any remaining characters
}

func displayRecommendation(rec performance.Recommendation) {
	fmt.Printf("\n💡 [Priority %d] %s\n", rec.Priority, rec.Description)
	fmt.Printf("   Impact: %s (Confidence: %.1f%%)\n", rec.Impact, rec.Confidence*100)
	fmt.Printf("   Action: %s\n", rec.Action.Type)
}

func displayRecommendationsTable(recommendations []performance.Recommendation) error {
	fmt.Printf("%-10s %-20s %-40s %-15s\n", "Priority", "Type", "Description", "Confidence")
	fmt.Println(strings.Repeat("-", 90))
	
	for _, rec := range recommendations {
		fmt.Printf("%-10d %-20s %-40s %-15.1f%%\n", 
			rec.Priority, 
			rec.Type, 
			truncateString(rec.Description, 40),
			rec.Confidence*100)
	}
	
	return nil
}

func displayMetricsTable(metrics []performance.Metric, metricType performance.MetricType) error {
	if len(metrics) == 0 {
		fmt.Printf("No metrics found for type: %s\n", metricType)
		return nil
	}

	fmt.Printf("Metrics for %s (showing last %d entries)\n", metricType, len(metrics))
	fmt.Printf("%-20s %-15s %-10s %-30s\n", "Timestamp", "Value", "Unit", "Labels")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, metric := range metrics {
		labels := ""
		for k, v := range metric.Labels {
			if labels != "" {
				labels += ", "
			}
			labels += fmt.Sprintf("%s=%s", k, v)
		}
		
		fmt.Printf("%-20s %-15.2f %-10s %-30s\n", 
			metric.Timestamp.Format("15:04:05"), 
			metric.Value, 
			metric.Unit,
			truncateString(labels, 30))
	}
	
	return nil
}

func outputReportTable(report performance.PerformanceReport, output string) error {
	content := fmt.Sprintf("Performance Report\n")
	content += fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339))
	content += fmt.Sprintf("Time Range: %s to %s\n\n", 
		report.TimeRange.Start.Format(time.RFC3339),
		report.TimeRange.End.Format(time.RFC3339))
	
	content += "Metrics Summary:\n"
	content += fmt.Sprintf("%-20s %-10s %-15s %-15s %-15s %-15s\n", 
		"Type", "Count", "Recent", "Average", "P95", "P99")
	content += strings.Repeat("-", 100) + "\n"
	
	for metricType, summary := range report.Metrics {
		content += fmt.Sprintf("%-20s %-10d %-15.2f %-15.2f %-15.2f %-15.2f\n",
			metricType, summary.Count, summary.RecentValue, 
			summary.Average, summary.P95, summary.P99)
	}
	
	content += fmt.Sprintf("\nRecommendations: %d active\n", len(report.Recommendations))
	
	if output != "" {
		return os.WriteFile(output, []byte(content), 0644)
	}
	
	fmt.Print(content)
	return nil
}

func outputJSON(data interface{}, output string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	if output != "" {
		return os.WriteFile(output, jsonData, 0644)
	}
	
	fmt.Println(string(jsonData))
	return nil
}

func outputYAML(data interface{}, output string) error {
	// For simplicity, we'll output JSON format for YAML
	// In a real implementation, you'd use a YAML library
	return outputJSON(data, output)
}

// Helper functions
func parseDuration(s string) (time.Duration, error) {
	// Simple duration parsing
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}
	
	unit := s[len(s)-1:]
	value := s[:len(s)-1]
	
	v, err := strconv.Atoi(value)
	if err != nil {
		return time.ParseDuration(s) // Fall back to Go's parser
	}
	
	switch unit {
	case "s":
		return time.Duration(v) * time.Second, nil
	case "m":
		return time.Duration(v) * time.Minute, nil
	case "h":
		return time.Duration(v) * time.Hour, nil
	case "d":
		return time.Duration(v) * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

