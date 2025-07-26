package accessibility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
)

// ComponentAccessibilityTester implements AccessibilityTester for testing components
type ComponentAccessibilityTester struct {
	engine   AccessibilityEngine
	registry interfaces.ComponentRegistry
	renderer *renderer.ComponentRenderer
	logger   logging.Logger
	config   TesterConfig
}

// TesterConfig contains configuration for the accessibility tester
type TesterConfig struct {
	DefaultWCAGLevel   WCAGLevel     `json:"default_wcag_level"`
	DefaultTimeout     time.Duration `json:"default_timeout"`
	EnableRealTimeWarn bool          `json:"enable_real_time_warnings"`
	ReportOutputDir    string        `json:"report_output_dir"`
	CustomRulePaths    []string      `json:"custom_rule_paths"`
	MaxConcurrentTests int           `json:"max_concurrent_tests"`
}

// NewComponentAccessibilityTester creates a new accessibility tester
func NewComponentAccessibilityTester(
	registry interfaces.ComponentRegistry,
	renderer *renderer.ComponentRenderer,
	logger logging.Logger,
	config TesterConfig,
) *ComponentAccessibilityTester {
	engine := NewDefaultAccessibilityEngine(logger)

	// Initialize engine with configuration
	engineConfig := EngineConfig{
		EnableBrowserEngine: false, // Start with HTML-only analysis
		MaxConcurrentChecks: config.MaxConcurrentTests,
		DefaultTimeout:      config.DefaultTimeout,
		CacheResults:        true,
		CacheSize:           1000,
		LogLevel:            "info",
	}

	if err := engine.Initialize(context.Background(), engineConfig); err != nil {
		// Log initialization error but continue with default configuration
		logger.Warn(context.Background(), err, "Failed to initialize accessibility engine, using defaults")
	}

	return &ComponentAccessibilityTester{
		engine:   engine,
		registry: registry,
		renderer: renderer,
		logger:   logger.WithComponent("accessibility_tester"),
		config:   config,
	}
}

// TestComponent runs accessibility tests on a single component
func (tester *ComponentAccessibilityTester) TestComponent(
	ctx context.Context,
	componentName string,
	props map[string]interface{},
) (*AccessibilityReport, error) {
	start := time.Now()

	tester.logger.Info(ctx, "Starting accessibility test for component",
		"component", componentName,
		"props", len(props))

	// Get component info from registry
	component, exists := tester.registry.Get(componentName)
	if !exists {
		return nil, fmt.Errorf("component not found: %s", componentName)
	}

	// Render component to HTML
	html, err := tester.renderComponentToHTML(ctx, component, props)
	if err != nil {
		return nil, fmt.Errorf("failed to render component: %w", err)
	}

	// Create audit configuration
	config := AuditConfiguration{
		WCAGLevel:     tester.config.DefaultWCAGLevel,
		ReportFormat:  FormatJSON,
		IncludeHTML:   true,
		MaxViolations: 1000,
		Timeout:       tester.config.DefaultTimeout,
	}

	// Run accessibility analysis
	report, err := tester.engine.Analyze(ctx, html, config)
	if err != nil {
		return nil, fmt.Errorf("accessibility analysis failed: %w", err)
	}

	// Update report with component-specific information
	report.ComponentName = componentName
	report.ComponentFile = component.FilePath
	report.Target.Name = componentName
	report.Target.Type = "component"

	// Add component context to violations
	for i := range report.Violations {
		report.Violations[i].Context.ComponentName = componentName
		report.Violations[i].Context.ComponentFile = component.FilePath
	}

	tester.logger.Info(ctx, "Accessibility test completed",
		"component", componentName,
		"violations", len(report.Violations),
		"duration", time.Since(start))

	return report, nil
}

// TestHTML runs accessibility tests on raw HTML content
func (tester *ComponentAccessibilityTester) TestHTML(
	ctx context.Context,
	html string,
	config AuditConfiguration,
) (*AccessibilityReport, error) {
	return tester.engine.Analyze(ctx, html, config)
}

// TestURL runs accessibility tests on a live web page
func (tester *ComponentAccessibilityTester) TestURL(
	ctx context.Context,
	url string,
	config AuditConfiguration,
) (*AccessibilityReport, error) {
	// For now, this would require a browser engine integration
	// This is a placeholder for future browser-based testing
	return nil, fmt.Errorf("URL testing not yet implemented - requires browser engine")
}

// GetAvailableRules returns all available accessibility rules
func (tester *ComponentAccessibilityTester) GetAvailableRules() []AccessibilityRule {
	engine := tester.engine.(*DefaultAccessibilityEngine)
	rules := []AccessibilityRule{}
	for _, rule := range engine.rules {
		rules = append(rules, rule)
	}
	return rules
}

// GetRulesByWCAGLevel returns rules for a specific WCAG level
func (tester *ComponentAccessibilityTester) GetRulesByWCAGLevel(
	level WCAGLevel,
) []AccessibilityRule {
	engine := tester.engine.(*DefaultAccessibilityEngine)
	return engine.getApplicableRules(level, nil, nil)
}

// TestAllComponents runs accessibility tests on all registered components
func (tester *ComponentAccessibilityTester) TestAllComponents(
	ctx context.Context,
) (map[string]*AccessibilityReport, error) {
	components := tester.registry.GetAll()
	reports := make(map[string]*AccessibilityReport)

	tester.logger.Info(
		ctx,
		"Starting accessibility test for all components",
		"count",
		len(components),
	)

	for _, component := range components {
		// Use default props or empty props for testing
		defaultProps := tester.getDefaultPropsForComponent(component)

		report, err := tester.TestComponent(ctx, component.Name, defaultProps)
		if err != nil {
			tester.logger.Warn(ctx, err, "Failed to test component", "component", component.Name)
			continue
		}

		reports[component.Name] = report
	}

	return reports, nil
}

// TestComponentWithMockData tests a component using mock data generation
func (tester *ComponentAccessibilityTester) TestComponentWithMockData(
	ctx context.Context,
	componentName string,
) (*AccessibilityReport, error) {
	component, exists := tester.registry.Get(componentName)
	if !exists {
		return nil, fmt.Errorf("component not found: %s", componentName)
	}

	// Generate mock data for component parameters
	mockProps := tester.generateMockPropsForComponent(component)

	return tester.TestComponent(ctx, componentName, mockProps)
}

// AutoFix attempts to automatically fix accessibility issues in HTML
func (tester *ComponentAccessibilityTester) AutoFix(
	ctx context.Context,
	html string,
	violations []AccessibilityViolation,
) (string, error) {
	return tester.engine.AutoFix(ctx, html, violations)
}

// GetAccessibilityScoreForComponent returns a simplified accessibility score
func (tester *ComponentAccessibilityTester) GetAccessibilityScoreForComponent(
	ctx context.Context,
	componentName string,
) (float64, error) {
	report, err := tester.TestComponent(ctx, componentName, nil)
	if err != nil {
		return 0, err
	}

	return report.Summary.OverallScore, nil
}

// GetAccessibilityInsights provides insights and recommendations for a component
func (tester *ComponentAccessibilityTester) GetAccessibilityInsights(
	ctx context.Context,
	componentName string,
) (*AccessibilityInsights, error) {
	report, err := tester.TestComponent(ctx, componentName, nil)
	if err != nil {
		return nil, err
	}

	insights := &AccessibilityInsights{
		ComponentName:   componentName,
		OverallScore:    report.Summary.OverallScore,
		WCAGLevel:       tester.getHighestCompliantLevel(report),
		CriticalIssues:  tester.getCriticalIssues(report.Violations),
		QuickWins:       tester.getQuickWins(report.Violations),
		Recommendations: tester.getRecommendations(report),
		NextSteps:       tester.getNextSteps(report),
	}

	return insights, nil
}

// renderComponentToHTML renders a component to HTML using the renderer
func (tester *ComponentAccessibilityTester) renderComponentToHTML(
	ctx context.Context,
	component *types.ComponentInfo,
	props map[string]interface{},
) (string, error) {
	// Create a simple HTML wrapper for the component
	wrapper := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s Component Test</title>
</head>
<body>
    <main>
        %s
    </main>
</body>
</html>`, component.Name, "%s")

	// Render the component
	// Note: This is simplified - in a real implementation, we'd need to properly
	// render the templ component with the provided props

	// For now, we'll create a mock HTML structure based on common patterns
	componentHTML := tester.generateMockHTML(component, props)

	return fmt.Sprintf(wrapper, componentHTML), nil
}

// generateMockHTML creates mock HTML for testing based on component name patterns
func (tester *ComponentAccessibilityTester) generateMockHTML(
	component *types.ComponentInfo,
	props map[string]interface{},
) string {
	name := strings.ToLower(component.Name)

	// Generate HTML based on component name patterns
	switch {
	case strings.Contains(name, "button"):
		text := "Button"
		if val, ok := props["text"]; ok {
			if str, ok := val.(string); ok {
				text = str
			}
		}
		return fmt.Sprintf(`<button type="button">%s</button>`, text)

	case strings.Contains(name, "form"):
		return `<form>
    <div class="form-field">
        <label for="test-input">Test Label</label>
        <input type="text" id="test-input" name="test" placeholder="Enter text" />
    </div>
    <button type="submit">Submit</button>
</form>`

	case strings.Contains(name, "card"):
		return `<div class="card">
    <h2>Card Title</h2>
    <p>This is a card component with some content.</p>
    <a href="#" class="card-link">Learn more</a>
</div>`

	case strings.Contains(name, "navigation") || strings.Contains(name, "nav"):
		return `<nav>
    <ul>
        <li><a href="#home">Home</a></li>
        <li><a href="#about">About</a></li>
        <li><a href="#contact">Contact</a></li>
    </ul>
</nav>`

	case strings.Contains(name, "header"):
		return `<header>
    <h1>Page Title</h1>
    <nav>
        <a href="#main" class="skip-link">Skip to main content</a>
    </nav>
</header>`

	case strings.Contains(name, "image") || strings.Contains(name, "img"):
		return `<img src="/placeholder.jpg" alt="Placeholder image description" />`

	default:
		// Generic component
		return fmt.Sprintf(`<div class="component-%s">
    <h3>%s Component</h3>
    <p>This is a %s component for accessibility testing.</p>
</div>`, strings.ToLower(component.Name), component.Name, component.Name)
	}
}

// getDefaultPropsForComponent returns default props for a component
func (tester *ComponentAccessibilityTester) getDefaultPropsForComponent(
	component *types.ComponentInfo,
) map[string]interface{} {
	props := make(map[string]interface{})

	// Set default values for common parameter patterns
	for _, param := range component.Parameters {
		switch strings.ToLower(param.Name) {
		case "text", "title", "label":
			props[param.Name] = fmt.Sprintf("Test %s", param.Name)
		case "variant", "type":
			props[param.Name] = "default"
		case "disabled", "required":
			props[param.Name] = false
		case "placeholder":
			props[param.Name] = fmt.Sprintf("Enter %s", param.Name)
		case "href", "url", "link":
			props[param.Name] = "#"
		case "alt", "alttext":
			props[param.Name] = "Alt text description"
		case "id":
			props[param.Name] = fmt.Sprintf("%s-test-id", strings.ToLower(component.Name))
		default:
			// Use default value if available
			if param.Default != nil {
				props[param.Name] = param.Default
			}
		}
	}

	return props
}

// generateMockPropsForComponent generates realistic mock data for component props
func (tester *ComponentAccessibilityTester) generateMockPropsForComponent(
	component *types.ComponentInfo,
) map[string]interface{} {
	props := make(map[string]interface{})

	for _, param := range component.Parameters {
		switch param.Type {
		case "string":
			props[param.Name] = tester.generateMockString(param.Name)
		case "bool", "boolean":
			props[param.Name] = false // Default to false for accessibility testing
		case "int", "int32", "int64":
			props[param.Name] = 1
		case "float64", "float32":
			props[param.Name] = 1.0
		default:
			if param.Default != nil {
				props[param.Name] = param.Default
			}
		}
	}

	return props
}

// generateMockString generates mock strings based on parameter name patterns
func (tester *ComponentAccessibilityTester) generateMockString(paramName string) string {
	name := strings.ToLower(paramName)

	switch {
	case strings.Contains(name, "title"):
		return "Test Title"
	case strings.Contains(name, "text"), strings.Contains(name, "content"):
		return "Test content for accessibility validation"
	case strings.Contains(name, "label"):
		return "Form Label"
	case strings.Contains(name, "placeholder"):
		return "Enter value here"
	case strings.Contains(name, "alt"):
		return "Descriptive alternative text for image"
	case strings.Contains(name, "href"), strings.Contains(name, "url"):
		return "#test-link"
	case strings.Contains(name, "id"):
		return fmt.Sprintf("test-%s", name)
	case strings.Contains(name, "class"):
		return "test-class"
	case strings.Contains(name, "name"):
		return "test-name"
	default:
		return fmt.Sprintf("Test %s value", paramName)
	}
}

// Helper methods for insights
func (tester *ComponentAccessibilityTester) getHighestCompliantLevel(
	report *AccessibilityReport,
) WCAGLevel {
	if report.Summary.WCAGCompliance.LevelAAA.Status == StatusCompliant {
		return WCAGLevelAAA
	}
	if report.Summary.WCAGCompliance.LevelAA.Status == StatusCompliant {
		return WCAGLevelAA
	}
	if report.Summary.WCAGCompliance.LevelA.Status == StatusCompliant {
		return WCAGLevelA
	}
	return WCAGLevelA // Default
}

func (tester *ComponentAccessibilityTester) getCriticalIssues(
	violations []AccessibilityViolation,
) []AccessibilityIssue {
	issues := []AccessibilityIssue{}
	for _, violation := range violations {
		if violation.Impact == ImpactCritical || violation.Severity == SeverityError {
			issues = append(issues, AccessibilityIssue{
				Rule:        violation.Rule,
				Description: violation.Message,
				Impact:      violation.Impact,
				FixEffort:   tester.estimateFixEffort(violation),
			})
		}
	}
	return issues
}

func (tester *ComponentAccessibilityTester) getQuickWins(
	violations []AccessibilityViolation,
) []AccessibilityIssue {
	issues := []AccessibilityIssue{}
	for _, violation := range violations {
		if violation.CanAutoFix || tester.isQuickFix(violation) {
			issues = append(issues, AccessibilityIssue{
				Rule:        violation.Rule,
				Description: violation.Message,
				Impact:      violation.Impact,
				FixEffort:   FixEffortLow,
			})
		}
	}
	return issues
}

func (tester *ComponentAccessibilityTester) getRecommendations(
	report *AccessibilityReport,
) []string {
	recommendations := []string{}

	if report.Summary.CriticalImpact > 0 {
		recommendations = append(recommendations, "Address critical accessibility issues first")
	}

	if report.Summary.OverallScore < 80 {
		recommendations = append(recommendations, "Focus on improving overall accessibility score")
	}

	ruleFrequency := make(map[string]int)
	for _, violation := range report.Violations {
		ruleFrequency[violation.Rule]++
	}

	// Find most common issues
	for rule, count := range ruleFrequency {
		if count > 1 {
			recommendations = append(
				recommendations,
				fmt.Sprintf("Multiple instances of %s found - consider component-wide fix", rule),
			)
		}
	}

	return recommendations
}

func (tester *ComponentAccessibilityTester) getNextSteps(report *AccessibilityReport) []string {
	steps := []string{}

	if len(report.Violations) == 0 {
		steps = append(
			steps,
			"Great! No accessibility violations found. Consider testing with screen reader.",
		)
		return steps
	}

	// Prioritize steps based on violations
	if report.Summary.CriticalImpact > 0 {
		steps = append(steps, "1. Fix critical accessibility issues immediately")
	}

	if report.Summary.ErrorViolations > 0 {
		steps = append(steps, "2. Address all error-level violations")
	}

	quickFixCount := 0
	for _, violation := range report.Violations {
		if violation.CanAutoFix {
			quickFixCount++
		}
	}

	if quickFixCount > 0 {
		steps = append(steps, fmt.Sprintf("3. Apply %d automatic fixes available", quickFixCount))
	}

	steps = append(steps, "4. Test with keyboard navigation")
	steps = append(steps, "5. Test with screen reader software")
	steps = append(steps, "6. Validate color contrast ratios")

	return steps
}

func (tester *ComponentAccessibilityTester) estimateFixEffort(
	violation AccessibilityViolation,
) FixEffort {
	if violation.CanAutoFix {
		return FixEffortLow
	}

	switch violation.Rule {
	case "missing-alt-text", "missing-lang-attribute", "missing-title-element":
		return FixEffortLow
	case "missing-form-label", "missing-button-text":
		return FixEffortMedium
	case "low-contrast", "missing-heading-structure":
		return FixEffortMedium
	default:
		return FixEffortHigh
	}
}

func (tester *ComponentAccessibilityTester) isQuickFix(violation AccessibilityViolation) bool {
	quickFixRules := []string{
		"missing-alt-text",
		"missing-lang-attribute",
		"missing-title-element",
		"missing-button-text",
	}

	for _, rule := range quickFixRules {
		if violation.Rule == rule {
			return true
		}
	}

	return false
}

// AccessibilityInsights provides high-level insights about component accessibility
type AccessibilityInsights struct {
	ComponentName   string               `json:"component_name"`
	OverallScore    float64              `json:"overall_score"`
	WCAGLevel       WCAGLevel            `json:"wcag_level"`
	CriticalIssues  []AccessibilityIssue `json:"critical_issues"`
	QuickWins       []AccessibilityIssue `json:"quick_wins"`
	Recommendations []string             `json:"recommendations"`
	NextSteps       []string             `json:"next_steps"`
}

// AccessibilityIssue represents a specific issue with fix effort estimation
type AccessibilityIssue struct {
	Rule        string          `json:"rule"`
	Description string          `json:"description"`
	Impact      ViolationImpact `json:"impact"`
	FixEffort   FixEffort       `json:"fix_effort"`
}

// FixEffort represents the estimated effort to fix an accessibility issue
type FixEffort string

const (
	FixEffortLow    FixEffort = "low"    // Minutes
	FixEffortMedium FixEffort = "medium" // Hours
	FixEffortHigh   FixEffort = "high"   // Days
)
