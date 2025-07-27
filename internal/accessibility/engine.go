package accessibility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"golang.org/x/net/html"
)

// Rule IDs - accessibility rule constants.
const (
	RuleMissingAltText          = "missing-alt-text"
	RuleMissingFormLabel        = "missing-form-label"
	RuleMissingHeadingStructure = "missing-heading-structure"
	RuleLowContrast             = "low-contrast"
	RuleMissingButtonText       = "missing-button-text"
	RuleMissingLangAttribute    = "missing-lang-attribute"
	RuleMissingTitleElement     = "missing-title-element"
	RuleInvalidAriaAttribute    = "invalid-aria-attribute"
	RuleDuplicateID             = "duplicate-id"
	RuleMissingSkipLink         = "missing-skip-link"
)

// HTML attributes.
const (
	AttrAlt            = "alt"
	AttrClass          = "class"
	AttrAriaLabel      = "aria-label"
	AttrAriaLabelledBy = "aria-labelledby"
	AttrHref           = "href"
	AttrType           = "type"
	AttrTitle          = "title"
	AttrLang           = "lang"
	AttrFor            = "for"
	AttrID             = "id"
)

// HTML tags.
const (
	TagHTML     = "html"
	TagImg      = "img"
	TagButton   = "button"
	TagLabel    = "label"
	TagInput    = "input"
	TagTextarea = "textarea"
	TagSelect   = "select"
	TagA        = "a"
	TagH1       = "h1"
	TagH2       = "h2"
	TagH3       = "h3"
	TagH4       = "h4"
	TagH5       = "h5"
	TagH6       = "h6"
)

// Input types.
const (
	InputTypeButton   = "button"
	InputTypeSubmit   = "submit"
	InputTypeReset    = "reset"
	InputTypeCheckbox = "checkbox"
	InputTypeRadio    = "radio"
	InputTypeText     = "text"
)

// ARIA roles.
const (
	RoleButton   = "button"
	RoleLink     = "link"
	RoleCheckbox = "checkbox"
	RoleRadio    = "radio"
	RoleTextbox  = "textbox"
	RoleHeading  = "heading"
	RoleImg      = "img"
)

// WCAG tags.
const (
	TagWCAG2A   = "wcag2a"
	TagWCAG2AA  = "wcag2aa"
	TagWCAG2AAA = "wcag2aaa"
)

// DefaultAccessibilityEngine implements the AccessibilityEngine interface.
type DefaultAccessibilityEngine struct {
	config EngineConfig
	rules  map[string]AccessibilityRule
	logger logging.Logger
}

// NewDefaultAccessibilityEngine creates a new accessibility engine.
func NewDefaultAccessibilityEngine(logger logging.Logger) *DefaultAccessibilityEngine {
	return &DefaultAccessibilityEngine{
		rules:  make(map[string]AccessibilityRule),
		logger: logger.WithComponent("accessibility_engine"),
	}
}

// Initialize sets up the accessibility engine with configuration.
func (engine *DefaultAccessibilityEngine) Initialize(
	ctx context.Context,
	config EngineConfig,
) error {
	engine.config = config

	// Load default WCAG rules
	engine.loadDefaultRules()

	// Load custom rules if provided
	for _, customRule := range config.CustomRules {
		engine.rules[customRule.ID] = AccessibilityRule{
			ID:          customRule.ID,
			Description: customRule.Description,
			Impact:      string(customRule.Impact),
			Tags:        customRule.Tags,
			HelpURL:     "https://templar.dev/accessibility/rules/" + customRule.ID,
		}
	}

	engine.logger.Info(ctx, "Accessibility engine initialized",
		"total_rules", len(engine.rules),
		"custom_rules", len(config.CustomRules))

	return nil
}

// Analyze performs accessibility analysis on HTML content.
func (engine *DefaultAccessibilityEngine) Analyze(
	ctx context.Context,
	htmlContent string,
	config AuditConfiguration,
) (*AccessibilityReport, error) {
	start := time.Now()

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Create report structure
	report := &AccessibilityReport{
		ID:            generateReportID(),
		Timestamp:     time.Now(),
		Target:        AccessibilityTarget{Type: "html_snippet", HTML: htmlContent},
		Configuration: config,
		Summary:       AccessibilitySummary{},
		Violations:    []AccessibilityViolation{},
		Passed:        []AccessibilityRule{},
		Inapplicable:  []AccessibilityRule{},
		Incomplete:    []AccessibilityIncomplete{},
	}

	// Convert HTML to our internal representation
	elements := engine.extractElements(doc)

	// Run accessibility checks
	violations := []AccessibilityViolation{}
	passedRules := []AccessibilityRule{}

	// Apply WCAG level filtering
	applicableRules := engine.getApplicableRules(
		config.WCAGLevel,
		config.Rules,
		config.ExcludeRules,
	)

	for _, rule := range applicableRules {
		ruleViolations, err := engine.checkRule(ctx, rule, elements, config)
		if err != nil {
			engine.logger.Warn(ctx, err, "Error checking accessibility rule", "rule", rule.ID)

			continue
		}

		if len(ruleViolations) > 0 {
			violations = append(violations, ruleViolations...)
		} else {
			passedRules = append(passedRules, rule)
		}
	}

	// Populate report
	report.Violations = violations
	report.Passed = passedRules
	report.Duration = time.Since(start)
	report.Summary = engine.generateSummary(violations, passedRules, applicableRules)

	if config.IncludeHTML {
		report.HTMLSnapshot = htmlContent
	}

	engine.logger.Info(ctx, "Accessibility analysis completed",
		"violations", len(violations),
		"passed_rules", len(passedRules),
		"duration", report.Duration)

	return report, nil
}

// GetSuggestions generates actionable suggestions for violations.
func (engine *DefaultAccessibilityEngine) GetSuggestions(
	ctx context.Context,
	violation AccessibilityViolation,
) ([]AccessibilitySuggestion, error) {
	suggestions := []AccessibilitySuggestion{}

	switch violation.Rule {
	case RuleMissingAltText:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionCodeChange,
			Title:       "Add alt attribute to image",
			Description: "Provide descriptive alternative text for the image content",
			Code:        `<img src="..." alt="Description of the image content" />`,
			Priority:    1,
			Resources: []Resource{
				{
					Title: "Alt text best practices",
					URL:   "https://www.w3.org/WAI/tutorials/images/",
					Type:  "documentation",
				},
			},
		})

	case RuleMissingFormLabel:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionCodeChange,
			Title:       "Associate label with form control",
			Description: "Form controls must have associated labels for screen readers",
			Code:        `<label for="input-id">Label text</label>\n<input id="input-id" type="text" />`,
			Priority:    1,
			Resources: []Resource{
				{
					Title: "Form labels",
					URL:   "https://www.w3.org/WAI/tutorials/forms/labels/",
					Type:  "documentation",
				},
			},
		})

	case RuleMissingHeadingStructure:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionStructural,
			Title:       "Fix heading hierarchy",
			Description: "Use proper heading sequence (h1, h2, h3) to create logical document structure",
			Code:        `<h1>Main title</h1>\n<h2>Section title</h2>\n<h3>Subsection title</h3>`,
			Priority:    2,
		})

	case RuleLowContrast:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionDesign,
			Title:       "Increase color contrast",
			Description: "Ensure text has sufficient contrast ratio (4.5:1 for normal text, 3:1 for large text)",
			Priority:    1,
			Resources: []Resource{
				{
					Title: "Color contrast checker",
					URL:   "https://webaim.org/resources/contrastchecker/",
					Type:  "tool",
				},
			},
		})

	case RuleMissingButtonText:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionARIAAttribute,
			Title:       "Add accessible name to button",
			Description: "Buttons need accessible names via text content, aria-label, or aria-labelledby",
			Code:        `<button aria-label="Close dialog">×</button>`,
			Priority:    1,
		})

	case RuleMissingLangAttribute:
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:        SuggestionCodeChange,
			Title:       "Add lang attribute to html element",
			Description: "Specify the primary language of the page",
			Code:        `<html lang="en">`,
			Priority:    2,
		})
	}

	// Add generic suggestions if no specific ones were found
	if len(suggestions) == 0 {
		suggestions = append(suggestions, AccessibilitySuggestion{
			Type:  SuggestionContent,
			Title: "Review accessibility guidelines",
			Description: fmt.Sprintf(
				"Review WCAG %s guidelines for rule: %s",
				violation.WCAG.Level,
				violation.Rule,
			),
			Priority: 3,
			Resources: []Resource{
				{
					Title: "WCAG Quick Reference",
					URL:   "https://www.w3.org/WAI/WCAG21/quickref/",
					Type:  "documentation",
				},
			},
		})
	}

	return suggestions, nil
}

// AutoFix attempts to automatically fix simple accessibility issues.
func (engine *DefaultAccessibilityEngine) AutoFix(
	ctx context.Context,
	htmlContent string,
	violations []AccessibilityViolation,
) (string, error) {
	fixed := htmlContent

	for _, violation := range violations {
		if !violation.CanAutoFix || violation.AutoFixCode == "" {
			continue
		}

		// Apply simple text-based fixes
		switch violation.Rule {
		case RuleMissingLangAttribute:
			if !strings.Contains(fixed, `lang="`) {
				fixed = strings.Replace(fixed, "<html>", `<html lang="en">`, 1)
			}

		case RuleMissingTitleElement:
			if !strings.Contains(fixed, "<title>") {
				headIndex := strings.Index(fixed, "</head>")
				if headIndex != -1 {
					fixed = fixed[:headIndex] + "    <title>Untitled Page</title>\n" + fixed[headIndex:]
				}
			}
		}
	}

	if fixed != htmlContent {
		engine.logger.Info(
			ctx,
			"Applied automatic accessibility fixes",
			"fixes_applied",
			len(violations),
		)
	}

	return fixed, nil
}

// Shutdown gracefully shuts down the accessibility engine.
func (engine *DefaultAccessibilityEngine) Shutdown(ctx context.Context) error {
	engine.logger.Info(ctx, "Accessibility engine shutdown")

	return nil
}

// loadDefaultRules loads the default WCAG accessibility rules.
func (engine *DefaultAccessibilityEngine) loadDefaultRules() {
	rules := []AccessibilityRule{
		{
			ID:          RuleMissingAltText,
			Description: "Images must have alternative text",
			Impact:      string(ImpactCritical),
			Tags:        []string{TagWCAG2A, "images"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/image-alt",
		},
		{
			ID:          RuleMissingFormLabel,
			Description: "Form elements must have labels",
			Impact:      string(ImpactCritical),
			Tags:        []string{TagWCAG2A, "forms"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/label",
		},
		{
			ID:          RuleMissingHeadingStructure,
			Description: "Headings must be in logical order",
			Impact:      string(ImpactSerious),
			Tags:        []string{TagWCAG2A, "headings"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/heading-order",
		},
		{
			ID:          RuleLowContrast,
			Description: "Text must have sufficient color contrast",
			Impact:      string(ImpactSerious),
			Tags:        []string{TagWCAG2AA, "color"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/color-contrast",
		},
		{
			ID:          RuleMissingButtonText,
			Description: "Buttons must have accessible names",
			Impact:      string(ImpactCritical),
			Tags:        []string{TagWCAG2A, "buttons"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/button-name",
		},
		{
			ID:          RuleMissingLangAttribute,
			Description: "HTML element must have a lang attribute",
			Impact:      string(ImpactSerious),
			Tags:        []string{TagWCAG2A, "language"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/html-has-lang",
		},
		{
			ID:          RuleMissingTitleElement,
			Description: "Documents must contain a title element",
			Impact:      string(ImpactSerious),
			Tags:        []string{TagWCAG2A, "document"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/document-title",
		},
		{
			ID:          RuleInvalidAriaAttribute,
			Description: "ARIA attributes must be valid",
			Impact:      string(ImpactCritical),
			Tags:        []string{TagWCAG2A, "aria"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/aria-valid-attr",
		},
		{
			ID:          RuleDuplicateID,
			Description: "IDs of active elements must be unique",
			Impact:      string(ImpactSerious),
			Tags:        []string{TagWCAG2A, "parsing"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/duplicate-id-active",
		},
		{
			ID:          RuleMissingSkipLink,
			Description: "Page should have skip navigation link",
			Impact:      string(ImpactModerate),
			Tags:        []string{TagWCAG2A, "navigation"},
			HelpURL:     "https://dequeuniversity.com/rules/axe/4.4/bypass",
		},
	}

	for _, rule := range rules {
		engine.rules[rule.ID] = rule
	}
}

// extractElements converts HTML nodes to our internal element representation.
func (engine *DefaultAccessibilityEngine) extractElements(node *html.Node) []HTMLElement {
	elements := []HTMLElement{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			elements = append(elements, &DefaultHTMLElement{Node: n})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)

	return elements
}

// getApplicableRules returns rules applicable for the given configuration.
func (engine *DefaultAccessibilityEngine) getApplicableRules(
	level WCAGLevel,
	includeRules, excludeRules []string,
) []AccessibilityRule {
	applicable := []AccessibilityRule{}

	for _, rule := range engine.rules {
		// Check if rule should be excluded
		if contains(excludeRules, rule.ID) {
			continue
		}

		// If specific rules are requested, only include those
		if len(includeRules) > 0 && !contains(includeRules, rule.ID) {
			continue
		}

		// Check WCAG level compatibility
		if engine.isRuleApplicableForLevel(rule, level) {
			applicable = append(applicable, rule)
		}
	}

	return applicable
}

// isRuleApplicableForLevel checks if a rule applies to the given WCAG level.
func (engine *DefaultAccessibilityEngine) isRuleApplicableForLevel(
	rule AccessibilityRule,
	level WCAGLevel,
) bool {
	switch level {
	case WCAGLevelA:
		return contains(rule.Tags, TagWCAG2A)
	case WCAGLevelAA:
		return contains(rule.Tags, TagWCAG2A) || contains(rule.Tags, TagWCAG2AA)
	case WCAGLevelAAA:
		return contains(rule.Tags, TagWCAG2A) || contains(rule.Tags, TagWCAG2AA) ||
			contains(rule.Tags, TagWCAG2AAA)
	default:
		return true
	}
}

// checkRule runs a specific accessibility rule against elements.
func (engine *DefaultAccessibilityEngine) checkRule(
	ctx context.Context,
	rule AccessibilityRule,
	elements []HTMLElement,
	config AuditConfiguration,
) ([]AccessibilityViolation, error) {
	violations := []AccessibilityViolation{}

	switch rule.ID {
	case RuleMissingAltText:
		for _, element := range elements {
			if element.TagName() == TagImg {
				if alt, hasAlt := element.GetAttribute(AttrAlt); !hasAlt || alt == "" {
					violations = append(
						violations,
						engine.createViolation(rule, element, "Image missing alt attribute"),
					)
				}
			}
		}

	case RuleMissingFormLabel:
		for _, element := range elements {
			if isFormControl(element.TagName()) {
				if !engine.hasAssociatedLabel(element, elements) {
					violations = append(
						violations,
						engine.createViolation(
							rule,
							element,
							"Form control missing associated label",
						),
					)
				}
			}
		}

	case RuleMissingHeadingStructure:
		headings := []HTMLElement{}
		for _, element := range elements {
			if isHeading(element.TagName()) {
				headings = append(headings, element)
			}
		}
		if !engine.hasLogicalHeadingOrder(headings) {
			if len(headings) > 0 {
				violations = append(
					violations,
					engine.createViolation(rule, headings[0], "Heading structure is not logical"),
				)
			}
		}

	case RuleMissingButtonText:
		for _, element := range elements {
			if element.TagName() == TagButton {
				if !engine.hasAccessibleName(element) {
					violations = append(
						violations,
						engine.createViolation(rule, element, "Button missing accessible name"),
					)
				}
			}
		}

	case RuleMissingLangAttribute:
		for _, element := range elements {
			if element.TagName() == TagHTML {
				if _, hasLang := element.GetAttribute(AttrLang); !hasLang {
					violations = append(
						violations,
						engine.createViolation(
							rule,
							element,
							"HTML element missing lang attribute",
						),
					)
				}
			}
		}

	case RuleDuplicateID:
		idMap := make(map[string][]HTMLElement)
		for _, element := range elements {
			if id, hasID := element.GetAttribute(AttrID); hasID && id != "" {
				idMap[id] = append(idMap[id], element)
			}
		}
		for id, elementsWithID := range idMap {
			if len(elementsWithID) > 1 {
				for _, element := range elementsWithID {
					violations = append(
						violations,
						engine.createViolation(rule, element, "Duplicate ID: "+id),
					)
				}
			}
		}
	}

	return violations, nil
}

// createViolation creates a new accessibility violation.
func (engine *DefaultAccessibilityEngine) createViolation(
	rule AccessibilityRule,
	element HTMLElement,
	message string,
) AccessibilityViolation {
	violation := AccessibilityViolation{
		ID:          generateViolationID(),
		Rule:        rule.ID,
		Severity:    engine.getSeverityFromImpact(rule.Impact),
		WCAG:        engine.getWCAGFromRule(rule),
		Element:     element.TagName(),
		Selector:    engine.generateSelector(element),
		Message:     message,
		Description: rule.Description,
		HelpURL:     rule.HelpURL,
		Impact:      ViolationImpact(rule.Impact),
		Context: ViolationContext{
			HTMLContext: element.GetOuterHTML(),
		},
		CreatedAt: time.Now(),
	}

	// Add suggestions
	suggestions, _ := engine.GetSuggestions(context.Background(), violation)
	violation.Suggestions = suggestions

	// Check if auto-fixable
	violation.CanAutoFix = engine.canAutoFix(rule.ID)

	return violation
}

// Helper functions.
func (engine *DefaultAccessibilityEngine) getSeverityFromImpact(impact string) ViolationSeverity {
	switch impact {
	case string(ImpactCritical):
		return SeverityError
	case string(ImpactSerious):
		return SeverityError
	case string(ImpactModerate):
		return SeverityWarning
	case string(ImpactMinor):
		return SeverityInfo
	default:
		return SeverityWarning
	}
}

func (engine *DefaultAccessibilityEngine) getWCAGFromRule(rule AccessibilityRule) WCAG {
	// Map rule to WCAG criteria based on tags
	if contains(rule.Tags, "images") {
		return WCAG{Level: WCAGLevelA, Criteria: Criteria1_1_1}
	}
	if contains(rule.Tags, "forms") {
		return WCAG{Level: WCAGLevelA, Criteria: Criteria3_3_2}
	}
	if contains(rule.Tags, "headings") {
		return WCAG{Level: WCAGLevelA, Criteria: Criteria1_3_1}
	}
	if contains(rule.Tags, "color") {
		return WCAG{Level: WCAGLevelAA, Criteria: Criteria1_4_3}
	}
	if contains(rule.Tags, "language") {
		return WCAG{Level: WCAGLevelA, Criteria: Criteria3_1_1}
	}

	return WCAG{Level: WCAGLevelA, Criteria: Criteria4_1_2}
}

func (engine *DefaultAccessibilityEngine) generateSelector(element HTMLElement) string {
	tagName := strings.ToLower(element.TagName())

	if id, hasID := element.GetAttribute(AttrID); hasID {
		return fmt.Sprintf("%s#%s", tagName, id)
	}

	if class, hasClass := element.GetAttribute(AttrClass); hasClass {
		classes := strings.Fields(class)
		if len(classes) > 0 {
			return fmt.Sprintf("%s.%s", tagName, strings.Join(classes, "."))
		}
	}

	return tagName
}

func (engine *DefaultAccessibilityEngine) canAutoFix(ruleID string) bool {
	autoFixableRules := []string{
		RuleMissingLangAttribute,
		RuleMissingTitleElement,
	}

	return contains(autoFixableRules, ruleID)
}

func (engine *DefaultAccessibilityEngine) hasAssociatedLabel(
	element HTMLElement,
	allElements []HTMLElement,
) bool {
	// Check for aria-label
	if _, hasAriaLabel := element.GetAttribute(AttrAriaLabel); hasAriaLabel {
		return true
	}

	// Check for aria-labelledby
	if _, hasAriaLabelledBy := element.GetAttribute(AttrAriaLabelledBy); hasAriaLabelledBy {
		return true
	}

	// Check for associated label element
	if id, hasID := element.GetAttribute(AttrID); hasID {
		for _, el := range allElements {
			if el.TagName() == TagLabel {
				if forAttr, hasFor := el.GetAttribute(AttrFor); hasFor && forAttr == id {
					return true
				}
			}
		}
	}

	// Check if wrapped in label
	parent := element.GetParent()

	return parent != nil && parent.TagName() == TagLabel
}

func (engine *DefaultAccessibilityEngine) hasLogicalHeadingOrder(headings []HTMLElement) bool {
	if len(headings) <= 1 {
		return true
	}

	levels := []int{}
	for _, heading := range headings {
		level := engine.getHeadingLevel(heading.TagName())
		levels = append(levels, level)
	}

	// Check if first heading is h1 and sequence is logical
	if levels[0] != 1 {
		return false
	}

	for i := 1; i < len(levels); i++ {
		if levels[i] > levels[i-1]+1 {
			return false
		}
	}

	return true
}

func (engine *DefaultAccessibilityEngine) getHeadingLevel(tagName string) int {
	switch tagName {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	default:
		return 0
	}
}

func (engine *DefaultAccessibilityEngine) hasAccessibleName(element HTMLElement) bool {
	// Check text content
	if strings.TrimSpace(element.GetTextContent()) != "" {
		return true
	}

	// Check aria-label
	if _, hasAriaLabel := element.GetAttribute(AttrAriaLabel); hasAriaLabel {
		return true
	}

	// Check aria-labelledby
	if _, hasAriaLabelledBy := element.GetAttribute(AttrAriaLabelledBy); hasAriaLabelledBy {
		return true
	}

	return false
}

func (engine *DefaultAccessibilityEngine) generateSummary(
	violations []AccessibilityViolation,
	passedRules []AccessibilityRule,
	totalRules []AccessibilityRule,
) AccessibilitySummary {
	summary := AccessibilitySummary{
		TotalRules:      len(totalRules),
		PassedRules:     len(passedRules),
		FailedRules:     len(totalRules) - len(passedRules),
		TotalViolations: len(violations),
	}

	// Count violations by severity and impact
	for _, violation := range violations {
		switch violation.Severity {
		case SeverityError:
			summary.ErrorViolations++
		case SeverityWarning:
			summary.WarnViolations++
		case SeverityInfo:
			summary.InfoViolations++
		}

		switch violation.Impact {
		case ImpactCritical:
			summary.CriticalImpact++
		case ImpactSerious:
			summary.SeriousImpact++
		case ImpactModerate:
			summary.ModerateImpact++
		case ImpactMinor:
			summary.MinorImpact++
		}
	}

	// Calculate overall score (100 - percentage of violations)
	if len(totalRules) > 0 {
		summary.OverallScore = float64(len(passedRules)) / float64(len(totalRules)) * 100
	}

	return summary
}

// Utility functions.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}

func isFormControl(tagName string) bool {
	formControls := []string{TagInput, TagTextarea, TagSelect, TagButton}

	return contains(formControls, tagName)
}

func isHeading(tagName string) bool {
	headings := []string{TagH1, TagH2, TagH3, TagH4, TagH5, TagH6}

	return contains(headings, tagName)
}

func generateReportID() string {
	return fmt.Sprintf("report_%d", time.Now().UnixNano())
}

func generateViolationID() string {
	return fmt.Sprintf("violation_%d", time.Now().UnixNano())
}

// DefaultHTMLElement implements HTMLElement interface for html.Node.
type DefaultHTMLElement struct {
	Node *html.Node
}

func (e *DefaultHTMLElement) TagName() string {
	return e.Node.Data
}

func (e *DefaultHTMLElement) GetAttribute(name string) (string, bool) {
	for _, attr := range e.Node.Attr {
		if attr.Key == name {
			return attr.Val, true
		}
	}

	return "", false
}

func (e *DefaultHTMLElement) GetTextContent() string {
	var text strings.Builder
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(e.Node)

	return text.String()
}

func (e *DefaultHTMLElement) GetInnerHTML() string {
	var result strings.Builder
	for c := e.Node.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&result, c); err != nil {
			// Log error but continue with partial content
			continue
		}
	}

	return result.String()
}

func (e *DefaultHTMLElement) GetOuterHTML() string {
	var result strings.Builder
	if err := html.Render(&result, e.Node); err != nil {
		// Return empty string if rendering fails
		return ""
	}

	return result.String()
}

func (e *DefaultHTMLElement) GetParent() HTMLElement {
	if e.Node.Parent != nil {
		return &DefaultHTMLElement{Node: e.Node.Parent}
	}

	return nil
}

func (e *DefaultHTMLElement) GetChildren() []HTMLElement {
	children := []HTMLElement{}
	for c := e.Node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			children = append(children, &DefaultHTMLElement{Node: c})
		}
	}

	return children
}

func (e *DefaultHTMLElement) QuerySelector(selector string) HTMLElement {
	// Simple selector implementation - would need more sophisticated parsing for full CSS selector support
	return nil
}

func (e *DefaultHTMLElement) QuerySelectorAll(selector string) []HTMLElement {
	return []HTMLElement{}
}

func (e *DefaultHTMLElement) HasClass(className string) bool {
	if class, hasClass := e.GetAttribute(AttrClass); hasClass {
		classes := strings.Fields(class)

		return contains(classes, className)
	}

	return false
}

func (e *DefaultHTMLElement) GetComputedStyle(property string) string {
	// Would require CSS parsing and computation - simplified for now
	return ""
}

func (e *DefaultHTMLElement) IsVisible() bool {
	// Simplified visibility check
	style, hasStyle := e.GetAttribute("style")
	if hasStyle {
		return !strings.Contains(style, "display:none") &&
			!strings.Contains(style, "visibility:hidden")
	}

	return true
}

func (e *DefaultHTMLElement) IsFocusable() bool {
	tagName := e.TagName()
	focusableTags := []string{TagInput, TagButton, TagSelect, TagTextarea, TagA}

	if contains(focusableTags, tagName) {
		return true
	}

	if _, hasTabIndex := e.GetAttribute("tabindex"); hasTabIndex {
		return true
	}

	return false
}

func (e *DefaultHTMLElement) GetAriaRole() string {
	if role, hasRole := e.GetAttribute("role"); hasRole {
		return role
	}

	// Return implicit role based on tag name
	tagName := e.TagName()
	switch tagName {
	case "button":
		return RoleButton
	case "a":
		if _, hasHref := e.GetAttribute(AttrHref); hasHref {
			return RoleLink
		}
	case "input":
		if inputType, hasType := e.GetAttribute(AttrType); hasType {
			switch inputType {
			case InputTypeButton, InputTypeSubmit, InputTypeReset:
				return RoleButton
			case InputTypeCheckbox:
				return RoleCheckbox
			case InputTypeRadio:
				return RoleRadio
			}
		}

		return RoleTextbox
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return RoleHeading
	case "img":
		return RoleImg
	}

	return ""
}

func (e *DefaultHTMLElement) GetAriaLabel() string {
	// Check aria-label first
	if ariaLabel, hasAriaLabel := e.GetAttribute(AttrAriaLabel); hasAriaLabel {
		return ariaLabel
	}

	// Check aria-labelledby
	if labelledBy, hasLabelledBy := e.GetAttribute(AttrAriaLabelledBy); hasLabelledBy {
		// In a real implementation, we'd find the referenced elements and get their text
		return labelledBy
	}

	// For images, use alt text
	if e.TagName() == TagImg {
		if alt, hasAlt := e.GetAttribute(AttrAlt); hasAlt {
			return alt
		}
	}

	// For form controls, check associated label
	// This is simplified - would need to traverse DOM to find actual labels

	// Fall back to text content
	return strings.TrimSpace(e.GetTextContent())
}

func (e *DefaultHTMLElement) GetAriaDescription() string {
	if ariaDescription, hasAriaDescription := e.GetAttribute("aria-describedby"); hasAriaDescription {
		// In a real implementation, we'd find the referenced elements and get their text
		return ariaDescription
	}

	// For images, use title attribute
	if e.TagName() == TagImg {
		if title, hasTitle := e.GetAttribute(AttrTitle); hasTitle {
			return title
		}
	}

	return ""
}
