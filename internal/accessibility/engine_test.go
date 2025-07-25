package accessibility

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAccessibilityEngine_Initialize(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine:  false,
		MaxConcurrentChecks:  5,
		DefaultTimeout:       10 * time.Second,
		CacheResults:         true,
		CacheSize:           1000,
		LogLevel:            "info",
	}

	err := engine.Initialize(context.Background(), config)
	require.NoError(t, err)

	// Check that default rules were loaded
	assert.NotEmpty(t, engine.rules)
	assert.Contains(t, engine.rules, "missing-alt-text")
	assert.Contains(t, engine.rules, "missing-form-label")
}

func TestDefaultAccessibilityEngine_AnalyzeMissingAltText(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	htmlWithMissingAlt := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Page</title>
</head>
<body>
    <img src="test.jpg" />
    <img src="test2.jpg" alt="" />
    <img src="test3.jpg" alt="Proper alt text" />
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	report, err := engine.Analyze(context.Background(), htmlWithMissingAlt, auditConfig)
	require.NoError(t, err)

	// Should find 2 violations (missing alt and empty alt)
	violations := []AccessibilityViolation{}
	for _, violation := range report.Violations {
		if violation.Rule == "missing-alt-text" {
			violations = append(violations, violation)
		}
	}

	assert.Len(t, violations, 2, "Should find 2 missing alt text violations")

	// Check violation details
	violation := violations[0]
	assert.Equal(t, "missing-alt-text", violation.Rule)
	assert.Equal(t, SeverityError, violation.Severity)
	assert.Equal(t, ImpactCritical, violation.Impact)
	assert.Equal(t, "img", violation.Element)
	assert.NotEmpty(t, violation.Suggestions)
}

func TestDefaultAccessibilityEngine_AnalyzeMissingFormLabel(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	htmlWithMissingLabel := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Form</title>
</head>
<body>
    <form>
        <input type="text" name="unlabeled" />
        
        <label for="proper-input">Proper Label</label>
        <input type="text" id="proper-input" name="proper" />
        
        <input type="text" aria-label="Aria labeled input" name="aria-labeled" />
    </form>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	report, err := engine.Analyze(context.Background(), htmlWithMissingLabel, auditConfig)
	require.NoError(t, err)

	// Should find 1 violation (unlabeled input)
	violations := []AccessibilityViolation{}
	for _, violation := range report.Violations {
		if violation.Rule == "missing-form-label" {
			violations = append(violations, violation)
		}
	}

	assert.Len(t, violations, 1, "Should find 1 missing form label violation")
	
	violation := violations[0]
	assert.Equal(t, "missing-form-label", violation.Rule)
	assert.Equal(t, ImpactCritical, violation.Impact)
	assert.Contains(t, violation.Message, "Form control missing associated label")
}

func TestDefaultAccessibilityEngine_AnalyzeMissingButtonText(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	htmlWithMissingButtonText := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Buttons</title>
</head>
<body>
    <button></button>
    <button>Proper Button Text</button>
    <button aria-label="Close">×</button>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	report, err := engine.Analyze(context.Background(), htmlWithMissingButtonText, auditConfig)
	require.NoError(t, err)

	// Should find 1 violation (empty button)
	violations := []AccessibilityViolation{}
	for _, violation := range report.Violations {
		if violation.Rule == "missing-button-text" {
			violations = append(violations, violation)
		}
	}

	assert.Len(t, violations, 1, "Should find 1 missing button text violation")
}

func TestDefaultAccessibilityEngine_AnalyzeHeadingStructure(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	// Test proper heading structure
	htmlWithProperHeadings := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Headings</title>
</head>
<body>
    <h1>Main Title</h1>
    <h2>Section Title</h2>
    <h3>Subsection Title</h3>
</body>
</html>`

	// Test improper heading structure (skipping h2)
	htmlWithImproperHeadings := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Headings</title>
</head>
<body>
    <h1>Main Title</h1>
    <h3>Skipped h2</h3>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	// Test proper structure - should pass
	report1, err := engine.Analyze(context.Background(), htmlWithProperHeadings, auditConfig)
	require.NoError(t, err)
	
	properHeadingViolations := []AccessibilityViolation{}
	for _, violation := range report1.Violations {
		if violation.Rule == "missing-heading-structure" {
			properHeadingViolations = append(properHeadingViolations, violation)
		}
	}
	assert.Len(t, properHeadingViolations, 0, "Proper heading structure should not have violations")

	// Test improper structure - should fail
	report2, err := engine.Analyze(context.Background(), htmlWithImproperHeadings, auditConfig)
	require.NoError(t, err)
	
	improperHeadingViolations := []AccessibilityViolation{}
	for _, violation := range report2.Violations {
		if violation.Rule == "missing-heading-structure" {
			improperHeadingViolations = append(improperHeadingViolations, violation)
		}
	}
	assert.Len(t, improperHeadingViolations, 1, "Improper heading structure should have violation")
}

func TestDefaultAccessibilityEngine_AnalyzeDuplicateIDs(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	htmlWithDuplicateIDs := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Duplicate IDs</title>
</head>
<body>
    <div id="unique-id">Unique element</div>
    <div id="duplicate-id">First duplicate</div>
    <div id="duplicate-id">Second duplicate</div>
    <div id="another-unique">Another unique element</div>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	report, err := engine.Analyze(context.Background(), htmlWithDuplicateIDs, auditConfig)
	require.NoError(t, err)

	// Should find 2 violations (both duplicate elements)
	violations := []AccessibilityViolation{}
	for _, violation := range report.Violations {
		if violation.Rule == "duplicate-id" {
			violations = append(violations, violation)
		}
	}

	assert.Len(t, violations, 2, "Should find 2 duplicate ID violations")
}

func TestDefaultAccessibilityEngine_AnalyzeMissingLangAttribute(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	htmlWithoutLang := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <p>This page is missing a lang attribute</p>
</body>
</html>`

	htmlWithLang := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Page</title>
</head>
<body>
    <p>This page has a proper lang attribute</p>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	// Test without lang - should fail
	report1, err := engine.Analyze(context.Background(), htmlWithoutLang, auditConfig)
	require.NoError(t, err)
	
	violations1 := []AccessibilityViolation{}
	for _, violation := range report1.Violations {
		if violation.Rule == "missing-lang-attribute" {
			violations1 = append(violations1, violation)
		}
	}
	assert.Len(t, violations1, 1, "Should find missing lang attribute violation")

	// Test with lang - should pass
	report2, err := engine.Analyze(context.Background(), htmlWithLang, auditConfig)
	require.NoError(t, err)
	
	violations2 := []AccessibilityViolation{}
	for _, violation := range report2.Violations {
		if violation.Rule == "missing-lang-attribute" {
			violations2 = append(violations2, violation)
		}
	}
	assert.Len(t, violations2, 0, "Should not find lang attribute violation when present")
}

func TestDefaultAccessibilityEngine_GetSuggestions(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	engine.Initialize(context.Background(), EngineConfig{})

	testCases := []struct {
		rule                string
		expectedSuggestions int
		expectedTypes       []SuggestionType
	}{
		{
			rule:                "missing-alt-text",
			expectedSuggestions: 1,
			expectedTypes:       []SuggestionType{SuggestionCodeChange},
		},
		{
			rule:                "missing-form-label",
			expectedSuggestions: 1,
			expectedTypes:       []SuggestionType{SuggestionCodeChange},
		},
		{
			rule:                "missing-button-text",
			expectedSuggestions: 1,
			expectedTypes:       []SuggestionType{SuggestionARIAAttribute},
		},
		{
			rule:                "low-contrast",
			expectedSuggestions: 1,
			expectedTypes:       []SuggestionType{SuggestionDesign},
		},
		{
			rule:                "unknown-rule",
			expectedSuggestions: 1,
			expectedTypes:       []SuggestionType{SuggestionContent},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.rule, func(t *testing.T) {
			violation := AccessibilityViolation{
				Rule: tc.rule,
				WCAG: WCAG{Level: WCAGLevelA, Criteria: Criteria1_1_1},
			}

			suggestions, err := engine.GetSuggestions(context.Background(), violation)
			require.NoError(t, err)

			assert.Len(t, suggestions, tc.expectedSuggestions)
			
			if len(suggestions) > 0 {
				assert.Contains(t, tc.expectedTypes, suggestions[0].Type)
				assert.NotEmpty(t, suggestions[0].Title)
				assert.NotEmpty(t, suggestions[0].Description)
			}
		})
	}
}

func TestDefaultAccessibilityEngine_AutoFix(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	engine.Initialize(context.Background(), EngineConfig{})

	originalHTML := `<html><head></head><body><p>Test content</p></body></html>`
	
	violations := []AccessibilityViolation{
		{
			Rule:        "missing-lang-attribute",
			CanAutoFix:  true,
			AutoFixCode: `lang="en"`,
		},
		{
			Rule:        "missing-title-element",
			CanAutoFix:  true,
			AutoFixCode: `<title>Untitled Page</title>`,
		},
		{
			Rule:        "missing-alt-text",
			CanAutoFix:  false, // Not auto-fixable
		},
	}

	fixedHTML, err := engine.AutoFix(context.Background(), originalHTML, violations)
	require.NoError(t, err)

	// Should have applied lang attribute
	assert.Contains(t, fixedHTML, `lang="en"`)
	
	// Original HTML should be different from fixed HTML
	assert.NotEqual(t, originalHTML, fixedHTML)
}

func TestDefaultAccessibilityEngine_WCAGLevelFiltering(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	// Get rules for different WCAG levels
	levelARules := engine.getApplicableRules(WCAGLevelA, nil, nil)
	levelAARules := engine.getApplicableRules(WCAGLevelAA, nil, nil)
	levelAAARules := engine.getApplicableRules(WCAGLevelAAA, nil, nil)

	// Level AA should include Level A rules
	assert.Greater(t, len(levelAARules), len(levelARules))
	
	// Level AAA should include Level A and AA rules
	assert.GreaterOrEqual(t, len(levelAAARules), len(levelAARules))
	assert.GreaterOrEqual(t, len(levelAAARules), len(levelAARules))

	// Check that Level A rules are included in Level AA
	levelAIDs := make(map[string]bool)
	for _, rule := range levelARules {
		levelAIDs[rule.ID] = true
	}

	for _, rule := range levelAARules {
		// Check if this rule has wcag2a tag (Level A rule)
		hasWCAG2A := false
		for _, tag := range rule.Tags {
			if tag == "wcag2a" {
				hasWCAG2A = true
				break
			}
		}
		if hasWCAG2A {
			assert.True(t, levelAIDs[rule.ID], "Level AA should include Level A rule: %s", rule.ID)
		}
	}
}

func TestDefaultAccessibilityEngine_ReportGeneration(t *testing.T) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	// Complex HTML with multiple issues
	complexHTML := `
<!DOCTYPE html>
<html>
<head>
</head>
<body>
    <img src="test.jpg" />
    <form>
        <input type="text" />
        <button></button>
    </form>
    <div id="duplicate"></div>
    <div id="duplicate"></div>
</body>
</html>`

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelAA,
		ReportFormat: FormatJSON,
		IncludeHTML:  true,
		Timeout:      10 * time.Second,
	}

	report, err := engine.Analyze(context.Background(), complexHTML, auditConfig)
	require.NoError(t, err)

	// Check report structure
	assert.NotEmpty(t, report.ID)
	assert.False(t, report.Timestamp.IsZero())
	assert.Equal(t, "html_snippet", report.Target.Type)
	assert.Equal(t, complexHTML, report.Target.HTML)
	assert.Equal(t, auditConfig, report.Configuration)

	// Should have multiple violations
	assert.Greater(t, len(report.Violations), 3)
	assert.Greater(t, report.Duration, time.Duration(0))

	// Check summary calculation
	assert.Equal(t, len(report.Violations), report.Summary.TotalViolations)
	assert.Greater(t, report.Summary.TotalRules, 0)
	assert.LessOrEqual(t, report.Summary.OverallScore, 100.0)
	assert.GreaterOrEqual(t, report.Summary.OverallScore, 0.0)

	// HTML should be included if requested
	if auditConfig.IncludeHTML {
		assert.Equal(t, complexHTML, report.HTMLSnapshot)
	}
}

func TestDefaultHTMLElement_Implementation(t *testing.T) {
	htmlContent := `<div id="test" class="example highlight" data-value="123">
		<span>Test content</span>
		<img src="test.jpg" alt="Test image" />
	</div>`

	doc, err := parseHTML(htmlContent)
	require.NoError(t, err)

	// Find the div element
	element := findFirstElementByTag(doc, "div")
	require.NotNil(t, element)

	htmlElement := &DefaultHTMLElement{Node: element}

	// Test basic properties
	assert.Equal(t, "div", htmlElement.TagName())

	// Test attributes
	id, hasId := htmlElement.GetAttribute("id")
	assert.True(t, hasId)
	assert.Equal(t, "test", id)

	class, hasClass := htmlElement.GetAttribute("class")
	assert.True(t, hasClass)
	assert.Equal(t, "example highlight", class)

	_, hasNonExistent := htmlElement.GetAttribute("non-existent")
	assert.False(t, hasNonExistent)

	// Test text content
	textContent := htmlElement.GetTextContent()
	assert.Contains(t, textContent, "Test content")

	// Test HTML content
	innerHTML := htmlElement.GetInnerHTML()
	assert.Contains(t, innerHTML, "<span>")
	assert.Contains(t, innerHTML, "<img")

	outerHTML := htmlElement.GetOuterHTML()
	assert.Contains(t, outerHTML, "<div")
	assert.Contains(t, outerHTML, `id="test"`)

	// Test CSS class checking
	assert.True(t, htmlElement.HasClass("example"))
	assert.True(t, htmlElement.HasClass("highlight"))
	assert.False(t, htmlElement.HasClass("non-existent"))

	// Test visibility (simplified check)
	assert.True(t, htmlElement.IsVisible())

	// Test children
	children := htmlElement.GetChildren()
	assert.Len(t, children, 2) // span and img

	// Test ARIA role detection
	role := htmlElement.GetAriaRole()
	assert.Equal(t, "", role) // div has no implicit role

	// Test accessible name
	accessibleName := htmlElement.GetAriaLabel()
	assert.Contains(t, accessibleName, "Test content")
}

// Helper functions for tests
func parseHTML(htmlContent string) (*html.Node, error) {
	return html.Parse(strings.NewReader(htmlContent))
}

func findFirstElementByTag(node *html.Node, tagName string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tagName {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := findFirstElementByTag(child, tagName); result != nil {
			return result
		}
	}

	return nil
}

// Benchmark tests
func BenchmarkAccessibilityEngine_Analyze(b *testing.B) {
	logger := logging.NewTestLogger()
	engine := NewDefaultAccessibilityEngine(logger)

	config := EngineConfig{
		EnableBrowserEngine: false,
		DefaultTimeout:      10 * time.Second,
	}
	engine.Initialize(context.Background(), config)

	complexHTML := generateComplexHTML(100) // Generate HTML with 100 elements

	auditConfig := AuditConfiguration{
		WCAGLevel:    WCAGLevelAA,
		ReportFormat: FormatJSON,
		IncludeHTML:  false,
		Timeout:      10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Analyze(context.Background(), complexHTML, auditConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func generateComplexHTML(elementCount int) string {
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html><html lang="en"><head><title>Test</title></head><body>`)

	for i := 0; i < elementCount; i++ {
		html.WriteString(fmt.Sprintf(`<div id="element-%d" class="test-class">`, i))
		
		if i%3 == 0 {
			html.WriteString(`<img src="test.jpg" alt="Test image" />`)
		}
		if i%4 == 0 {
			html.WriteString(`<input type="text" />`)
		}
		if i%5 == 0 {
			html.WriteString(`<button>Button</button>`)
		}
		
		html.WriteString(fmt.Sprintf(`Content %d</div>`, i))
	}

	html.WriteString(`</body></html>`)
	return html.String()
}