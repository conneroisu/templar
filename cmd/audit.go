package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/accessibility"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	auditComponentName   string
	auditWCAGLevel       string
	auditOutputFormat    string
	auditOutputFile      string
	auditIncludeHTML     bool
	auditFixableOnly     bool
	auditSeverityFilter  string
	auditQuiet           bool
	auditVerbose         bool
	auditMaxViolations   int
	auditGenerateReport  bool
	auditShowSuggestions bool
	auditAutoFix         bool
	auditShowGuidance    bool
	auditGuidanceOnly    bool
)

// auditCmd represents the audit command.
var auditCmd = &cobra.Command{
	Use:   "audit [component-name]",
	Short: "Run accessibility audit on components",
	Long: `Run comprehensive accessibility audits on templ components to identify
WCAG compliance issues and get actionable suggestions for improvements.

The audit command can test individual components or all components in your project.
It provides detailed reports with severity levels, WCAG criteria mapping, and
specific suggestions for fixing accessibility issues.

Examples:
  # Audit all components
  templar audit

  # Audit specific component
  templar audit Button

  # Audit with specific WCAG level
  templar audit --wcag-level AA

  # Generate HTML report
  templar audit --output html --output-file report.html

  # Show only critical issues
  templar audit --severity error

  # Apply automatic fixes
  templar audit --auto-fix`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getComponentCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runAuditCommand,
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().
		StringVarP(&auditComponentName, "component", "c", "", "Specific component to audit (if not provided as argument)")
	auditCmd.Flags().
		StringVarP(&auditWCAGLevel, "wcag-level", "w", "AA", "WCAG compliance level to test against (A, AA, AAA)")
	auditCmd.Flags().
		StringVarP(&auditOutputFormat, "output", "o", "console", "Output format (console, json, html, markdown)")
	auditCmd.Flags().
		StringVarP(&auditOutputFile, "output-file", "f", "", "Output file path (stdout if not specified)")
	auditCmd.Flags().
		BoolVar(&auditIncludeHTML, "include-html", false, "Include HTML snapshot in report")
	auditCmd.Flags().
		BoolVar(&auditFixableOnly, "fixable-only", false, "Show only issues that can be automatically fixed")
	auditCmd.Flags().
		StringVarP(&auditSeverityFilter, "severity", "s", "", "Filter by severity level (error, warning, info)")
	auditCmd.Flags().BoolVarP(&auditQuiet, "quiet", "q", false, "Suppress non-error output")
	auditCmd.Flags().BoolVarP(&auditVerbose, "verbose", "v", false, "Enable verbose output")
	auditCmd.Flags().
		IntVarP(&auditMaxViolations, "max-violations", "m", 0, "Maximum number of violations to report (0 = unlimited)")
	auditCmd.Flags().
		BoolVar(&auditGenerateReport, "generate-report", false, "Generate detailed accessibility report")
	auditCmd.Flags().
		BoolVar(&auditShowSuggestions, "show-suggestions", true, "Include suggestions in output")
	auditCmd.Flags().
		BoolVar(&auditAutoFix, "auto-fix", false, "Attempt to automatically fix issues where possible")
	auditCmd.Flags().
		BoolVar(&auditShowGuidance, "show-guidance", false, "Include detailed accessibility guidance")
	auditCmd.Flags().
		BoolVar(&auditGuidanceOnly, "guidance-only", false, "Show only guidance without running audit")
}

func runAuditCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine component name from args or flag
	componentName := auditComponentName
	if len(args) > 0 {
		componentName = args[0]
	}

	// Handle guidance-only mode
	if auditGuidanceOnly {
		return showGuidanceOnly(componentName)
	}

	// Initialize logging
	loggerConfig := &logging.LoggerConfig{
		Level:     logging.LevelInfo,
		Format:    "text",
		Component: "audit",
		Output:    os.Stdout,
	}
	if auditQuiet {
		loggerConfig.Level = logging.LevelError
	} else if auditVerbose {
		loggerConfig.Level = logging.LevelDebug
	}
	logger := logging.NewLogger(loggerConfig)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry, cfg)

	// Scan components
	if !auditQuiet {
		logger.Info(ctx, "Scanning components...")
	}

	err = componentScanner.ScanDirectory(cfg.Components.ScanPaths[0])
	if err != nil {
		return fmt.Errorf("failed to scan components: %w", err)
	}

	// Create renderer
	componentRenderer := renderer.NewComponentRenderer(componentRegistry)

	// Initialize accessibility tester
	testerConfig := accessibility.TesterConfig{
		DefaultWCAGLevel:   parseWCAGLevel(auditWCAGLevel),
		DefaultTimeout:     30 * time.Second,
		EnableRealTimeWarn: false,
		MaxConcurrentTests: 1,
	}

	tester := accessibility.NewComponentAccessibilityTester(
		componentRegistry,
		componentRenderer,
		logger,
		testerConfig,
	)

	// Perform audit
	if componentName != "" {
		return runSingleComponentAudit(ctx, tester, componentName, logger)
	} else {
		return runAllComponentsAudit(ctx, tester, componentRegistry, logger)
	}
}

func runSingleComponentAudit(
	ctx context.Context,
	tester accessibility.AccessibilityTester,
	componentName string,
	logger logging.Logger,
) error {
	if !auditQuiet {
		logger.Info(ctx, "Running accessibility audit", "component", componentName)
	}

	// Run accessibility test
	report, err := tester.TestComponent(ctx, componentName, nil)
	if err != nil {
		return fmt.Errorf("accessibility audit failed for %s: %w", componentName, err)
	}

	// Apply filters
	report = applyReportFilters(report)

	// Apply auto-fixes if requested
	if auditAutoFix && len(report.Violations) > 0 {
		fixedCount, err := applyAutoFixes(ctx, tester, report)
		if err != nil {
			logger.Warn(ctx, err, "Failed to apply auto-fixes")
		} else if fixedCount > 0 {
			logger.Info(ctx, "Applied automatic fixes", "count", fixedCount)
		}
	}

	// Output results
	return outputAuditResults([]*accessibility.AccessibilityReport{report}, logger)
}

func runAllComponentsAudit(
	ctx context.Context,
	tester accessibility.AccessibilityTester,
	registry interfaces.ComponentRegistry,
	logger logging.Logger,
) error {
	components := registry.GetAll()

	if !auditQuiet {
		logger.Info(ctx, "Running accessibility audit on all components", "count", len(components))
	}

	reports := []*accessibility.AccessibilityReport{}
	totalViolations := 0
	totalAutoFixes := 0

	for i, component := range components {
		if auditVerbose {
			logger.Info(ctx, "Auditing component",
				"component", component.Name,
				"progress", fmt.Sprintf("%d/%d", i+1, len(components)))
		}

		// Run accessibility test
		report, err := tester.TestComponent(ctx, component.Name, nil)
		if err != nil {
			logger.Warn(ctx, err, "Failed to audit component", "component", component.Name)

			continue
		}

		// Apply filters
		report = applyReportFilters(report)

		// Apply auto-fixes if requested
		if auditAutoFix && len(report.Violations) > 0 {
			fixedCount, err := applyAutoFixes(ctx, tester, report)
			if err != nil {
				logger.Warn(ctx, err, "Failed to apply auto-fixes", "component", component.Name)
			} else {
				totalAutoFixes += fixedCount
			}
		}

		reports = append(reports, report)
		totalViolations += len(report.Violations)
	}

	if !auditQuiet {
		logger.Info(ctx, "Audit completed",
			"components", len(reports),
			"total_violations", totalViolations)

		if auditAutoFix && totalAutoFixes > 0 {
			logger.Info(ctx, "Applied automatic fixes", "total_fixes", totalAutoFixes)
		}
	}

	// Output results
	return outputAuditResults(reports, logger)
}

func applyReportFilters(
	report *accessibility.AccessibilityReport,
) *accessibility.AccessibilityReport {
	filteredViolations := []accessibility.AccessibilityViolation{}

	for _, violation := range report.Violations {
		// Apply severity filter
		if auditSeverityFilter != "" {
			expectedSeverity := parseSeverity(auditSeverityFilter)
			if violation.Severity != expectedSeverity {
				continue
			}
		}

		// Apply fixable filter
		if auditFixableOnly && !violation.CanAutoFix {
			continue
		}

		filteredViolations = append(filteredViolations, violation)
	}

	// Apply max violations limit
	if auditMaxViolations > 0 && len(filteredViolations) > auditMaxViolations {
		filteredViolations = filteredViolations[:auditMaxViolations]
	}

	// Update report
	report.Violations = filteredViolations

	// Recalculate summary
	report.Summary = calculateAccessibilitySummary(filteredViolations, report.Passed)

	return report
}

func applyAutoFixes(
	ctx context.Context,
	tester accessibility.AccessibilityTester,
	report *accessibility.AccessibilityReport,
) (int, error) {
	if componentTester, ok := tester.(*accessibility.ComponentAccessibilityTester); ok {
		autoFixableViolations := []accessibility.AccessibilityViolation{}
		for _, violation := range report.Violations {
			if violation.CanAutoFix {
				autoFixableViolations = append(autoFixableViolations, violation)
			}
		}

		if len(autoFixableViolations) == 0 {
			return 0, nil
		}

		// Apply auto-fixes (this would need integration with file system)
		_, err := componentTester.AutoFix(ctx, report.HTMLSnapshot, autoFixableViolations)
		if err != nil {
			return 0, err
		}

		return len(autoFixableViolations), nil
	}

	return 0, errors.New("auto-fix not supported for this tester type")
}

func outputAuditResults(reports []*accessibility.AccessibilityReport, logger logging.Logger) error {
	switch auditOutputFormat {
	case "json":
		return outputJSON(reports)
	case "html":
		return outputHTML(reports)
	case "markdown":
		return outputMarkdown(reports)
	case "console":
		fallthrough
	default:
		return outputConsole(reports, logger)
	}
}

func outputJSON(reports []*accessibility.AccessibilityReport) error {
	var output interface{} = reports
	if len(reports) == 1 {
		output = reports[0] // Single component audit returns single report
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return writeOutput(string(jsonData))
}

func outputHTML(reports []*accessibility.AccessibilityReport) error {
	html := generateHTMLReport(reports)

	return writeOutput(html)
}

func outputMarkdown(reports []*accessibility.AccessibilityReport) error {
	markdown := generateMarkdownReport(reports)

	return writeOutput(markdown)
}

func outputConsole(reports []*accessibility.AccessibilityReport, logger logging.Logger) error {
	ctx := context.Background()
	_ = ctx // TODO: Use context in console output if needed

	if len(reports) == 0 {
		fmt.Println("No components audited.")

		return nil
	}

	// Summary statistics
	totalComponents := len(reports)
	totalViolations := 0
	criticalViolations := 0
	componentsWithIssues := 0
	overallScoreSum := 0.0

	for _, report := range reports {
		totalViolations += len(report.Violations)
		overallScoreSum += report.Summary.OverallScore

		if len(report.Violations) > 0 {
			componentsWithIssues++
		}

		for _, violation := range report.Violations {
			if violation.Impact == accessibility.ImpactCritical {
				criticalViolations++
			}
		}
	}

	averageScore := overallScoreSum / float64(totalComponents)

	// Print summary
	fmt.Printf("\n🔍 Accessibility Audit Summary\n")
	fmt.Printf("═══════════════════════════════\n\n")
	fmt.Printf("Components audited:     %d\n", totalComponents)
	fmt.Printf("Components with issues: %d\n", componentsWithIssues)
	fmt.Printf("Total violations:       %d\n", totalViolations)
	fmt.Printf("Critical violations:    %d\n", criticalViolations)
	fmt.Printf("Average score:          %.1f/100\n", averageScore)

	// Overall status
	var status string
	var statusIcon string
	if criticalViolations > 0 {
		status = "CRITICAL ISSUES FOUND"
		statusIcon = "🚨"
	} else if totalViolations > 0 {
		status = "ISSUES FOUND"
		statusIcon = "⚠️"
	} else {
		status = "ALL CHECKS PASSED"
		statusIcon = "✅"
	}

	fmt.Printf("Status:                 %s %s\n\n", statusIcon, status)

	// Detailed component results
	if auditVerbose || len(reports) == 1 {
		for _, report := range reports {
			outputComponentDetails(report)
		}
	} else if totalViolations > 0 {
		// Show only components with issues
		fmt.Printf("Components with accessibility issues:\n")
		fmt.Printf("───────────────────────────────────\n\n")

		for _, report := range reports {
			if len(report.Violations) > 0 {
				outputComponentSummary(report)
			}
		}
	}

	// Show suggestions if enabled
	if auditShowSuggestions && totalViolations > 0 {
		fmt.Printf("\n💡 Top Suggestions\n")
		fmt.Printf("──────────────────\n\n")

		suggestions := aggregateSuggestions(reports)
		for i, suggestion := range suggestions {
			if i >= 5 { // Limit to top 5
				break
			}
			fmt.Printf("%d. %s\n", i+1, suggestion.Title)
			if suggestion.Description != "" {
				fmt.Printf("   %s\n", suggestion.Description)
			}
			fmt.Printf("\n")
		}
	}

	// Show detailed guidance if enabled and there are violations
	if totalViolations > 0 {
		allViolations := []accessibility.AccessibilityViolation{}
		for _, report := range reports {
			allViolations = append(allViolations, report.Violations...)
		}
		showGuidanceForViolations(allViolations)
	}

	return nil
}

func outputComponentDetails(report *accessibility.AccessibilityReport) {
	componentName := report.ComponentName
	if componentName == "" {
		componentName = "Unknown Component"
	}

	scoreColor := getScoreColor(report.Summary.OverallScore)

	fmt.Printf("📦 %s (Score: %s%.1f/100%s)\n",
		componentName, scoreColor, report.Summary.OverallScore, "\033[0m")
	fmt.Printf("   File: %s\n", report.ComponentFile)

	if len(report.Violations) == 0 {
		fmt.Printf("   ✅ No accessibility issues found\n\n")

		return
	}

	// Group violations by severity
	errorViolations := []accessibility.AccessibilityViolation{}
	warningViolations := []accessibility.AccessibilityViolation{}
	infoViolations := []accessibility.AccessibilityViolation{}

	for _, violation := range report.Violations {
		switch violation.Severity {
		case accessibility.SeverityError:
			errorViolations = append(errorViolations, violation)
		case accessibility.SeverityWarning:
			warningViolations = append(warningViolations, violation)
		case accessibility.SeverityInfo:
			infoViolations = append(infoViolations, violation)
		}
	}

	// Output violations by severity
	if len(errorViolations) > 0 {
		fmt.Printf("   🚨 Errors (%d):\n", len(errorViolations))
		for _, violation := range errorViolations {
			outputViolation(violation, "     ")
		}
	}

	if len(warningViolations) > 0 {
		fmt.Printf("   ⚠️  Warnings (%d):\n", len(warningViolations))
		for _, violation := range warningViolations {
			outputViolation(violation, "     ")
		}
	}

	if len(infoViolations) > 0 && auditVerbose {
		fmt.Printf("   ℹ️  Info (%d):\n", len(infoViolations))
		for _, violation := range infoViolations {
			outputViolation(violation, "     ")
		}
	}

	fmt.Printf("\n")
}

func outputComponentSummary(report *accessibility.AccessibilityReport) {
	componentName := report.ComponentName
	if componentName == "" {
		componentName = "Unknown Component"
	}

	errorCount := 0
	warningCount := 0
	criticalCount := 0

	for _, violation := range report.Violations {
		switch violation.Severity {
		case accessibility.SeverityError:
			errorCount++
		case accessibility.SeverityWarning:
			warningCount++
		}

		if violation.Impact == accessibility.ImpactCritical {
			criticalCount++
		}
	}

	scoreColor := getScoreColor(report.Summary.OverallScore)

	fmt.Printf(
		"📦 %s %s(%.1f/100)%s\n",
		componentName,
		scoreColor,
		report.Summary.OverallScore,
		"\033[0m",
	)

	if criticalCount > 0 {
		fmt.Printf("   🚨 %d critical issue(s)\n", criticalCount)
	}
	if errorCount > 0 {
		fmt.Printf("   ❌ %d error(s)\n", errorCount)
	}
	if warningCount > 0 {
		fmt.Printf("   ⚠️  %d warning(s)\n", warningCount)
	}

	fmt.Printf("\n")
}

func outputViolation(violation accessibility.AccessibilityViolation, indent string) {
	fmt.Printf("%s• %s\n", indent, violation.Message)
	fmt.Printf("%s  Rule: %s | WCAG: %s %s\n",
		indent, violation.Rule, violation.WCAG.Level, violation.WCAG.Criteria)

	if violation.Element != "" {
		fmt.Printf("%s  Element: <%s>\n", indent, violation.Element)
	}

	if auditShowSuggestions && len(violation.Suggestions) > 0 {
		fmt.Printf("%s  💡 %s\n", indent, violation.Suggestions[0].Title)
		if violation.Suggestions[0].Code != "" && auditVerbose {
			fmt.Printf("%s     Code: %s\n", indent, violation.Suggestions[0].Code)
		}
	}

	if violation.CanAutoFix {
		fmt.Printf("%s  🔧 Auto-fixable\n", indent)
	}

	fmt.Printf("\n")
}

func writeOutput(content string) error {
	if auditOutputFile != "" {
		// Ensure output directory exists
		dir := filepath.Dir(auditOutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write to file
		if err := os.WriteFile(auditOutputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Report written to: %s\n", auditOutputFile)

		return nil
	}

	// Write to stdout
	fmt.Print(content)

	return nil
}

// Helper functions.
func parseWCAGLevel(level string) accessibility.WCAGLevel {
	switch strings.ToUpper(level) {
	case "A":
		return accessibility.WCAGLevelA
	case "AA":
		return accessibility.WCAGLevelAA
	case "AAA":
		return accessibility.WCAGLevelAAA
	default:
		return accessibility.WCAGLevelAA
	}
}

func parseSeverity(severity string) accessibility.ViolationSeverity {
	switch strings.ToLower(severity) {
	case "error":
		return accessibility.SeverityError
	case "warning":
		return accessibility.SeverityWarning
	case "info":
		return accessibility.SeverityInfo
	default:
		return accessibility.SeverityWarning
	}
}

func getScoreColor(score float64) string {
	if score >= 90 {
		return "\033[32m" // Green
	} else if score >= 70 {
		return "\033[33m" // Yellow
	} else {
		return "\033[31m" // Red
	}
}

func aggregateSuggestions(
	reports []*accessibility.AccessibilityReport,
) []accessibility.AccessibilitySuggestion {
	suggestionMap := make(map[string]*accessibility.AccessibilitySuggestion)
	suggestionCounts := make(map[string]int)

	for _, report := range reports {
		for _, violation := range report.Violations {
			for _, suggestion := range violation.Suggestions {
				key := fmt.Sprintf("%s_%s", suggestion.Type, suggestion.Title)
				suggestionCounts[key]++

				if existing, exists := suggestionMap[key]; !exists ||
					suggestion.Priority < existing.Priority {
					suggestionCopy := suggestion
					suggestionMap[key] = &suggestionCopy
				}
			}
		}
	}

	// Convert to slice and sort by frequency and priority
	suggestions := []accessibility.AccessibilitySuggestion{}
	for key, suggestion := range suggestionMap {
		// Adjust priority based on frequency (more frequent = higher priority)
		suggestion.Priority -= suggestionCounts[key] // Lower number = higher priority
		suggestions = append(suggestions, *suggestion)
	}

	// Sort by priority
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority < suggestions[j].Priority
	})

	return suggestions
}

func calculateAccessibilitySummary(
	violations []accessibility.AccessibilityViolation,
	passed []accessibility.AccessibilityRule,
) accessibility.AccessibilitySummary {
	summary := accessibility.AccessibilitySummary{
		TotalRules:      len(passed) + len(violations),
		PassedRules:     len(passed),
		FailedRules:     len(violations),
		TotalViolations: len(violations),
	}

	for _, violation := range violations {
		switch violation.Severity {
		case accessibility.SeverityError:
			summary.ErrorViolations++
		case accessibility.SeverityWarning:
			summary.WarnViolations++
		case accessibility.SeverityInfo:
			summary.InfoViolations++
		}

		switch violation.Impact {
		case accessibility.ImpactCritical:
			summary.CriticalImpact++
		case accessibility.ImpactSerious:
			summary.SeriousImpact++
		case accessibility.ImpactModerate:
			summary.ModerateImpact++
		case accessibility.ImpactMinor:
			summary.MinorImpact++
		}
	}

	// Calculate overall score
	if summary.TotalRules > 0 {
		summary.OverallScore = float64(summary.PassedRules) / float64(summary.TotalRules) * 100
	}

	return summary
}

func generateHTMLReport(reports []*accessibility.AccessibilityReport) string {
	// This would generate a comprehensive HTML report
	// For brevity, returning a simplified version
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Accessibility Audit Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 8px; }
        .component { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }
        .violation { margin: 10px 0; padding: 10px; background: #fff3cd; border-left: 4px solid #ffc107; }
        .error { background: #f8d7da; border-left-color: #dc3545; }
        .success { background: #d4edda; border-left-color: #28a745; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Accessibility Audit Report</h1>
        <p>Generated on: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
    </div>`

	for _, report := range reports {
		html += fmt.Sprintf(`
    <div class="component">
        <h2>%s</h2>
        <p>Score: %.1f/100</p>
        <p>Violations: %d</p>
    </div>`, report.ComponentName, report.Summary.OverallScore, len(report.Violations))
	}

	html += `
</body>
</html>`

	return html
}

func generateMarkdownReport(reports []*accessibility.AccessibilityReport) string {
	md := fmt.Sprintf(
		"# Accessibility Audit Report\n\nGenerated on: %s\n\n",
		time.Now().Format("2006-01-02 15:04:05"),
	)

	for _, report := range reports {
		md += fmt.Sprintf("## %s\n\n", report.ComponentName)
		md += fmt.Sprintf("- **Score**: %.1f/100\n", report.Summary.OverallScore)
		md += fmt.Sprintf("- **Violations**: %d\n\n", len(report.Violations))

		if len(report.Violations) > 0 {
			md += "### Issues Found\n\n"
			for _, violation := range report.Violations {
				md += fmt.Sprintf("- **%s**: %s\n", violation.Rule, violation.Message)
			}
			md += "\n"
		}
	}

	return md
}

func getComponentCompletions(toComplete string) []string {
	// This would integrate with the component registry to provide completions
	// For now, returning empty slice
	return []string{}
}

// showGuidanceOnly displays accessibility guidance without running an audit.
func showGuidanceOnly(componentName string) error {
	guide := accessibility.NewAccessibilityGuide()

	if componentName != "" {
		// Show component-specific guidance
		fmt.Printf("🎯 Accessibility Guidance for %s Component\n", componentName)
		fmt.Printf("═══════════════════════════════════════════════\n\n")

		guidanceText := guide.GetComponentGuidanceText(componentName)
		fmt.Print(guidanceText)

		// Also show general guidance applicable to all components
		fmt.Printf("\n📋 General Accessibility Guidelines\n")
		fmt.Printf("──────────────────────────────────────────────\n\n")

		quickStart := guide.GetQuickStartGuide()
		for i, item := range quickStart {
			if i >= 3 { // Limit to top 3 for brevity
				break
			}
			fmt.Printf("%d. %s\n", i+1, item.Title)
			fmt.Printf("   %s\n\n", item.Description)
		}
	} else {
		// Show general accessibility guidance
		fmt.Printf("🌟 Accessibility Quick Start Guide\n")
		fmt.Printf("═══════════════════════════════════════\n\n")

		quickStart := guide.GetQuickStartGuide()
		for i, item := range quickStart {
			fmt.Printf("%d. %s\n", i+1, item.Title)
			fmt.Printf("   %s\n", item.Description)

			if len(item.Examples) > 0 {
				example := item.Examples[0]
				if example.BadCode != "" {
					fmt.Printf("   ❌ Avoid: %s\n", strings.ReplaceAll(example.BadCode, "\n", " "))
				}
				if example.GoodCode != "" {
					fmt.Printf("   ✅ Use: %s\n", strings.ReplaceAll(example.GoodCode, "\n", " "))
				}
			}
			fmt.Printf("\n")
		}

		fmt.Printf("💡 Advanced Guidelines\n")
		fmt.Printf("─────────────────────────\n\n")

		bestPractices := guide.GetBestPracticesGuide()
		for i, item := range bestPractices {
			if i >= 3 { // Limit for readability
				break
			}
			fmt.Printf("• %s\n", item.Title)
			fmt.Printf("  %s\n\n", item.Description)
		}

		fmt.Printf("📚 Additional Resources\n")
		fmt.Printf("──────────────────────────\n")
		fmt.Printf("• WCAG Quick Reference: https://www.w3.org/WAI/WCAG21/quickref/\n")
		fmt.Printf("• WebAIM Guidelines: https://webaim.org/\n")
		fmt.Printf("• A11y Project: https://www.a11yproject.com/\n")
		fmt.Printf("• MDN Accessibility: https://developer.mozilla.org/en-US/docs/Web/Accessibility\n\n")

		fmt.Printf("🔧 To audit your components, run:\n")
		fmt.Printf("   templar audit              # Audit all components\n")
		fmt.Printf("   templar audit Button       # Audit specific component\n")
		fmt.Printf("   templar audit --help        # See all options\n")
	}

	return nil
}

// showGuidanceForViolations displays guidance for specific accessibility violations.
func showGuidanceForViolations(violations []accessibility.AccessibilityViolation) {
	if !auditShowGuidance || len(violations) == 0 {
		return
	}

	fmt.Printf("\n🎓 Accessibility Guidance\n")
	fmt.Printf("═════════════════════════\n\n")

	guide := accessibility.NewAccessibilityGuide()

	// Group violations by rule to avoid duplicate guidance
	ruleMap := make(map[string]bool)
	uniqueRules := []string{}

	for _, violation := range violations {
		if !ruleMap[violation.Rule] {
			ruleMap[violation.Rule] = true
			uniqueRules = append(uniqueRules, violation.Rule)
		}
	}

	// Show guidance for each unique rule
	for i, rule := range uniqueRules {
		if i > 0 {
			fmt.Print("\n" + strings.Repeat("─", 60) + "\n\n")
		}

		guidanceText := guide.GetGuidanceText(rule)
		fmt.Print(guidanceText)
	}
}
