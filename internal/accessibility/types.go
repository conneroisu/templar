package accessibility

import (
	"context"
	"time"
)

// WCAG represents Web Content Accessibility Guidelines levels and criteria.
type WCAG struct {
	Level    WCAGLevel    `json:"level"`
	Criteria WCAGCriteria `json:"criteria"`
}

// WCAGLevel represents different WCAG compliance levels.
type WCAGLevel string

const (
	WCAGLevelA   WCAGLevel = "A"
	WCAGLevelAA  WCAGLevel = "AA"
	WCAGLevelAAA WCAGLevel = "AAA"
)

// WCAGCriteria represents the specific WCAG success criteria.
type WCAGCriteria string

const (
	// Level A criteria.
	Criteria1_1_1 WCAGCriteria = "1.1.1" // Non-text Content
	Criteria1_3_1 WCAGCriteria = "1.3.1" // Info and Relationships
	Criteria1_3_3 WCAGCriteria = "1.3.3" // Sensory Characteristics
	Criteria1_4_1 WCAGCriteria = "1.4.1" // Use of Color
	Criteria2_1_1 WCAGCriteria = "2.1.1" // Keyboard
	Criteria2_1_2 WCAGCriteria = "2.1.2" // No Keyboard Trap
	Criteria2_4_1 WCAGCriteria = "2.4.1" // Bypass Blocks
	Criteria2_4_2 WCAGCriteria = "2.4.2" // Page Titled
	Criteria3_1_1 WCAGCriteria = "3.1.1" // Language of Page
	Criteria3_2_1 WCAGCriteria = "3.2.1" // On Focus
	Criteria3_2_2 WCAGCriteria = "3.2.2" // On Input
	Criteria3_3_1 WCAGCriteria = "3.3.1" // Error Identification
	Criteria3_3_2 WCAGCriteria = "3.3.2" // Labels or Instructions
	Criteria4_1_1 WCAGCriteria = "4.1.1" // Parsing
	Criteria4_1_2 WCAGCriteria = "4.1.2" // Name, Role, Value

	// Level AA criteria.
	Criteria1_2_1 WCAGCriteria = "1.2.1" // Audio-only and Video-only (Prerecorded)
	Criteria1_2_2 WCAGCriteria = "1.2.2" // Captions (Prerecorded)
	Criteria1_2_3 WCAGCriteria = "1.2.3" // Audio Description or Media Alternative (Prerecorded)
	Criteria1_4_3 WCAGCriteria = "1.4.3" // Contrast (Minimum)
	Criteria1_4_4 WCAGCriteria = "1.4.4" // Resize text
	Criteria1_4_5 WCAGCriteria = "1.4.5" // Images of Text
	Criteria2_4_3 WCAGCriteria = "2.4.3" // Focus Order
	Criteria2_4_4 WCAGCriteria = "2.4.4" // Link Purpose (In Context)
	Criteria2_4_5 WCAGCriteria = "2.4.5" // Multiple Ways
	Criteria2_4_6 WCAGCriteria = "2.4.6" // Headings and Labels
	Criteria2_4_7 WCAGCriteria = "2.4.7" // Focus Visible
	Criteria3_1_2 WCAGCriteria = "3.1.2" // Language of Parts
	Criteria3_2_3 WCAGCriteria = "3.2.3" // Consistent Navigation
	Criteria3_2_4 WCAGCriteria = "3.2.4" // Consistent Identification
	Criteria3_3_3 WCAGCriteria = "3.3.3" // Error Suggestion
	Criteria3_3_4 WCAGCriteria = "3.3.4" // Error Prevention (Legal, Financial, Data)
)

// AccessibilityViolation represents a single accessibility issue found during testing.
type AccessibilityViolation struct {
	ID          string                    `json:"id"`
	Rule        string                    `json:"rule"`
	Severity    ViolationSeverity         `json:"severity"`
	WCAG        WCAG                      `json:"wcag"`
	Element     string                    `json:"element"`
	Selector    string                    `json:"selector"`
	Message     string                    `json:"message"`
	Description string                    `json:"description"`
	HelpURL     string                    `json:"help_url"`
	Impact      ViolationImpact           `json:"impact"`
	Context     ViolationContext          `json:"context"`
	Suggestions []AccessibilitySuggestion `json:"suggestions"`
	CanAutoFix  bool                      `json:"can_auto_fix"`
	AutoFixCode string                    `json:"auto_fix_code,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
}

// ViolationSeverity represents the severity level of an accessibility violation.
type ViolationSeverity string

const (
	SeverityError   ViolationSeverity = "error"
	SeverityWarning ViolationSeverity = "warning"
	SeverityInfo    ViolationSeverity = "info"
)

// ViolationImpact represents the potential impact of an accessibility violation.
type ViolationImpact string

const (
	ImpactCritical ViolationImpact = "critical"
	ImpactSerious  ViolationImpact = "serious"
	ImpactModerate ViolationImpact = "moderate"
	ImpactMinor    ViolationImpact = "minor"
)

// ViolationContext provides contextual information about where the violation occurred.
type ViolationContext struct {
	ComponentName string                 `json:"component_name"`
	ComponentFile string                 `json:"component_file"`
	LineNumber    int                    `json:"line_number,omitempty"`
	ColumnNumber  int                    `json:"column_number,omitempty"`
	HTMLContext   string                 `json:"html_context"`
	ParentElement string                 `json:"parent_element,omitempty"`
	SiblingCount  int                    `json:"sibling_count,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AccessibilitySuggestion provides actionable suggestions for fixing accessibility issues.
type AccessibilitySuggestion struct {
	Type        SuggestionType `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Code        string         `json:"code,omitempty"`
	Resources   []Resource     `json:"resources,omitempty"`
	Priority    int            `json:"priority"` // 1 = highest, 5 = lowest
}

// SuggestionType categorizes different types of accessibility suggestions.
type SuggestionType string

const (
	SuggestionCodeChange    SuggestionType = "code_change"
	SuggestionARIAAttribute SuggestionType = "aria_attribute"
	SuggestionStructural    SuggestionType = "structural"
	SuggestionSemantic      SuggestionType = "semantic"
	SuggestionDesign        SuggestionType = "design"
	SuggestionContent       SuggestionType = "content"
)

// Resource provides additional learning resources for accessibility improvements.
type Resource struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Type        string `json:"type"` // "documentation", "example", "tool", "video"
	Description string `json:"description"`
}

// AccessibilityReport contains the complete results of an accessibility audit.
type AccessibilityReport struct {
	ID            string                    `json:"id"`
	Timestamp     time.Time                 `json:"timestamp"`
	ComponentName string                    `json:"component_name"`
	ComponentFile string                    `json:"component_file"`
	Target        AccessibilityTarget       `json:"target"`
	Configuration AuditConfiguration        `json:"configuration"`
	Summary       AccessibilitySummary      `json:"summary"`
	Violations    []AccessibilityViolation  `json:"violations"`
	Passed        []AccessibilityRule       `json:"passed"`
	Inapplicable  []AccessibilityRule       `json:"inapplicable"`
	Incomplete    []AccessibilityIncomplete `json:"incomplete"`
	Duration      time.Duration             `json:"duration"`
	HTMLSnapshot  string                    `json:"html_snapshot,omitempty"`
}

// AccessibilityTarget describes what was tested.
type AccessibilityTarget struct {
	Type     string `json:"type"` // "component", "page", "html_snippet"
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	HTML     string `json:"html,omitempty"`
	Selector string `json:"selector,omitempty"`
}

// AuditConfiguration contains settings for the accessibility audit.
type AuditConfiguration struct {
	WCAGLevel        WCAGLevel       `json:"wcag_level"`
	Rules            []string        `json:"rules,omitempty"` // Specific rules to run
	ExcludeRules     []string        `json:"exclude_rules,omitempty"`
	IncludeSelectors []string        `json:"include_selectors,omitempty"`
	ExcludeSelectors []string        `json:"exclude_selectors,omitempty"`
	Tags             []string        `json:"tags,omitempty"` // Rule tags to include
	ReportFormat     ReportFormat    `json:"report_format"`
	IncludeHTML      bool            `json:"include_html"`
	MaxViolations    int             `json:"max_violations"`
	Timeout          time.Duration   `json:"timeout"`
	BrowserSettings  BrowserSettings `json:"browser_settings,omitempty"`
}

// ReportFormat specifies the format for accessibility reports.
type ReportFormat string

const (
	FormatJSON     ReportFormat = "json"
	FormatHTML     ReportFormat = "html"
	FormatMarkdown ReportFormat = "markdown"
	FormatConsole  ReportFormat = "console"
)

// BrowserSettings configures browser-specific testing parameters.
type BrowserSettings struct {
	UserAgent    string `json:"user_agent,omitempty"`
	ViewportSize struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport_size"`
	ColorScheme string `json:"color_scheme,omitempty"` // "light", "dark", "no-preference"
}

// AccessibilitySummary provides high-level statistics about the accessibility audit.
type AccessibilitySummary struct {
	TotalRules   int `json:"total_rules"`
	PassedRules  int `json:"passed_rules"`
	FailedRules  int `json:"failed_rules"`
	SkippedRules int `json:"skipped_rules"`

	TotalViolations int `json:"total_violations"`
	ErrorViolations int `json:"error_violations"`
	WarnViolations  int `json:"warn_violations"`
	InfoViolations  int `json:"info_violations"`

	CriticalImpact int `json:"critical_impact"`
	SeriousImpact  int `json:"serious_impact"`
	ModerateImpact int `json:"moderate_impact"`
	MinorImpact    int `json:"minor_impact"`

	WCAGCompliance WCAGComplianceStatus `json:"wcag_compliance"`
	OverallScore   float64              `json:"overall_score"` // 0-100 accessibility score
}

// WCAGComplianceStatus represents compliance status for each WCAG level.
type WCAGComplianceStatus struct {
	LevelA   ComplianceLevel `json:"level_a"`
	LevelAA  ComplianceLevel `json:"level_aa"`
	LevelAAA ComplianceLevel `json:"level_aaa"`
}

// ComplianceLevel represents the compliance status for a specific WCAG level.
type ComplianceLevel struct {
	Status      ComplianceStatus `json:"status"`
	PassedCount int              `json:"passed_count"`
	FailedCount int              `json:"failed_count"`
	TotalCount  int              `json:"total_count"`
}

// ComplianceStatus represents the overall compliance status.
type ComplianceStatus string

const (
	StatusCompliant    ComplianceStatus = "compliant"
	StatusNonCompliant ComplianceStatus = "non_compliant"
	StatusPartial      ComplianceStatus = "partial"
	StatusUnknown      ComplianceStatus = "unknown"
)

// AccessibilityRule represents a specific accessibility rule that was checked.
type AccessibilityRule struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Tags        []string `json:"tags"`
	HelpURL     string   `json:"help_url"`
}

// AccessibilityIncomplete represents a rule that couldn't be fully evaluated.
type AccessibilityIncomplete struct {
	AccessibilityRule
	Reason string `json:"reason"`
}

// AccessibilityTester is the main interface for running accessibility tests.
type AccessibilityTester interface {
	// TestComponent runs accessibility tests on a single component
	TestComponent(
		ctx context.Context,
		componentName string,
		props map[string]interface{},
	) (*AccessibilityReport, error)

	// TestHTML runs accessibility tests on raw HTML content
	TestHTML(
		ctx context.Context,
		html string,
		config AuditConfiguration,
	) (*AccessibilityReport, error)

	// TestURL runs accessibility tests on a live web page
	TestURL(
		ctx context.Context,
		url string,
		config AuditConfiguration,
	) (*AccessibilityReport, error)

	// GetAvailableRules returns all available accessibility rules
	GetAvailableRules() []AccessibilityRule

	// GetRulesByWCAGLevel returns rules for a specific WCAG level
	GetRulesByWCAGLevel(level WCAGLevel) []AccessibilityRule
}

// AccessibilityEngine provides core accessibility testing functionality.
type AccessibilityEngine interface {
	// Initialize sets up the accessibility engine with configuration
	Initialize(ctx context.Context, config EngineConfig) error

	// Analyze performs accessibility analysis on HTML content
	Analyze(
		ctx context.Context,
		html string,
		config AuditConfiguration,
	) (*AccessibilityReport, error)

	// GetSuggestions generates actionable suggestions for violations
	GetSuggestions(
		ctx context.Context,
		violation AccessibilityViolation,
	) ([]AccessibilitySuggestion, error)

	// AutoFix attempts to automatically fix simple accessibility issues
	AutoFix(ctx context.Context, html string, violations []AccessibilityViolation) (string, error)

	// Shutdown gracefully shuts down the accessibility engine
	Shutdown(ctx context.Context) error
}

// EngineConfig contains configuration for the accessibility engine.
type EngineConfig struct {
	EnableBrowserEngine bool          `json:"enable_browser_engine"`
	BrowserPath         string        `json:"browser_path,omitempty"`
	CustomRules         []CustomRule  `json:"custom_rules,omitempty"`
	ExtensionPaths      []string      `json:"extension_paths,omitempty"`
	MaxConcurrentChecks int           `json:"max_concurrent_checks"`
	DefaultTimeout      time.Duration `json:"default_timeout"`
	CacheResults        bool          `json:"cache_results"`
	CacheSize           int           `json:"cache_size"`
	LogLevel            string        `json:"log_level"`
}

// CustomRule allows defining custom accessibility rules.
type CustomRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Impact      ViolationImpact   `json:"impact"`
	WCAG        WCAG              `json:"wcag"`
	Tags        []string          `json:"tags"`
	Selector    string            `json:"selector"`
	Check       RuleCheckFunction `json:"-"`
}

// RuleCheckFunction is a function that checks for accessibility violations.
type RuleCheckFunction func(ctx context.Context, element HTMLElement) ([]AccessibilityViolation, error)

// HTMLElement represents an HTML element for accessibility testing.
type HTMLElement interface {
	// TagName returns the tag name of the element
	TagName() string

	// GetAttribute returns the value of the specified attribute
	GetAttribute(name string) (string, bool)

	// GetTextContent returns the text content of the element
	GetTextContent() string

	// GetInnerHTML returns the inner HTML of the element
	GetInnerHTML() string

	// GetOuterHTML returns the outer HTML of the element
	GetOuterHTML() string

	// GetParent returns the parent element
	GetParent() HTMLElement

	// GetChildren returns all child elements
	GetChildren() []HTMLElement

	// QuerySelector returns the first matching child element
	QuerySelector(selector string) HTMLElement

	// QuerySelectorAll returns all matching child elements
	QuerySelectorAll(selector string) []HTMLElement

	// HasClass returns true if the element has the specified CSS class
	HasClass(className string) bool

	// GetComputedStyle returns the computed CSS value for a property
	GetComputedStyle(property string) string

	// IsVisible returns true if the element is visible on the page
	IsVisible() bool

	// IsFocusable returns true if the element can receive focus
	IsFocusable() bool

	// GetAriaRole returns the computed ARIA role
	GetAriaRole() string

	// GetAriaLabel returns the computed accessible name
	GetAriaLabel() string

	// GetAriaDescription returns the computed accessible description
	GetAriaDescription() string
}
