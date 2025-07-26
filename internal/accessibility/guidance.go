package accessibility

import (
	"fmt"
	"strings"
)

// AccessibilityGuide provides contextual accessibility guidance
type AccessibilityGuide struct {
	guidelines map[string][]GuidanceItem
}

// GuidanceItem represents a single piece of accessibility guidance
type GuidanceItem struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Examples    []GuidanceExample `json:"examples"`
	Resources   []Resource        `json:"resources"`
	WCAG        WCAG              `json:"wcag"`
	Severity    ViolationSeverity `json:"severity"`
	Priority    int               `json:"priority"`
	Tags        []string          `json:"tags"`
}

// GuidanceExample shows before/after code examples
type GuidanceExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	BadCode     string `json:"bad_code"`
	GoodCode    string `json:"good_code"`
	Explanation string `json:"explanation"`
}

// NewAccessibilityGuide creates a new accessibility guidance system
func NewAccessibilityGuide() *AccessibilityGuide {
	guide := &AccessibilityGuide{
		guidelines: make(map[string][]GuidanceItem),
	}

	guide.loadDefaultGuidelines()
	return guide
}

// GetGuidanceForRule returns guidance for a specific accessibility rule
func (guide *AccessibilityGuide) GetGuidanceForRule(rule string) []GuidanceItem {
	if items, exists := guide.guidelines[rule]; exists {
		return items
	}
	return []GuidanceItem{}
}

// GetGuidanceForComponent returns general accessibility guidance for component development
func (guide *AccessibilityGuide) GetGuidanceForComponent(componentName string) []GuidanceItem {
	componentType := strings.ToLower(componentName)

	var relevantGuidance []GuidanceItem

	// Add component-specific guidance based on component type
	switch {
	case strings.Contains(componentType, "button"):
		relevantGuidance = append(relevantGuidance, guide.getButtonGuidance()...)
	case strings.Contains(componentType, "form"):
		relevantGuidance = append(relevantGuidance, guide.getFormGuidance()...)
	case strings.Contains(componentType, "image") || strings.Contains(componentType, "img"):
		relevantGuidance = append(relevantGuidance, guide.getImageGuidance()...)
	case strings.Contains(componentType, "navigation") || strings.Contains(componentType, "nav"):
		relevantGuidance = append(relevantGuidance, guide.getNavigationGuidance()...)
	case strings.Contains(componentType, "modal") || strings.Contains(componentType, "dialog"):
		relevantGuidance = append(relevantGuidance, guide.getModalGuidance()...)
	case strings.Contains(componentType, "table"):
		relevantGuidance = append(relevantGuidance, guide.getTableGuidance()...)
	}

	// Add general guidance
	relevantGuidance = append(relevantGuidance, guide.getGeneralGuidance()...)

	return relevantGuidance
}

// GetAllGuidelines returns all available guidance organized by category
func (guide *AccessibilityGuide) GetAllGuidelines() map[string][]GuidanceItem {
	return guide.guidelines
}

// GetQuickStartGuide returns essential accessibility guidance for beginners
func (guide *AccessibilityGuide) GetQuickStartGuide() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Start with semantic HTML",
			Description: "Use appropriate HTML elements for their intended purpose",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title:       "Use semantic elements",
					BadCode:     `<div onclick="submit()">Submit</div>`,
					GoodCode:    `<button type="submit">Submit</button>`,
					Explanation: "Buttons provide keyboard support and screen reader context automatically",
				},
			},
			Tags: []string{"semantic", "html", "foundation"},
		},
		{
			Title:       "Provide text alternatives",
			Description: "All non-text content needs text alternatives",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title:       "Image alt text",
					BadCode:     `<img src="chart.png" />`,
					GoodCode:    `<img src="chart.png" alt="Sales increased 15% this quarter" />`,
					Explanation: "Alt text should describe the content and function, not just appearance",
				},
			},
			Tags: []string{"images", "alt-text", "wcag-a"},
		},
		{
			Title:       "Ensure keyboard accessibility",
			Description: "All interactive elements must be keyboard accessible",
			Priority:    2,
			Examples: []GuidanceExample{
				{
					Title:       "Focusable elements",
					BadCode:     `<div onclick="toggle()">Click me</div>`,
					GoodCode:    `<button onclick="toggle()" onkeydown="handleKey(event)">Click me</button>`,
					Explanation: "Use proper interactive elements or add tabindex and keyboard handlers",
				},
			},
			Tags: []string{"keyboard", "focus", "interaction"},
		},
		{
			Title:       "Use sufficient color contrast",
			Description: "Text must have adequate contrast against backgrounds",
			Priority:    2,
			Resources: []Resource{
				{
					Title: "Color Contrast Checker",
					URL:   "https://webaim.org/resources/contrastchecker/",
					Type:  "tool",
				},
			},
			Tags: []string{"color", "contrast", "wcag-aa"},
		},
		{
			Title:       "Label form controls",
			Description: "Every form input needs an associated label",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title:       "Form labels",
					BadCode:     `<input type="email" placeholder="Email" />`,
					GoodCode:    `<label for="email">Email Address</label>\n<input type="email" id="email" />`,
					Explanation: "Labels create programmatic relationships screen readers can understand",
				},
			},
			Tags: []string{"forms", "labels", "wcag-a"},
		},
	}
}

// GetBestPracticesGuide returns advanced accessibility best practices
func (guide *AccessibilityGuide) GetBestPracticesGuide() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Use ARIA landmarks",
			Description: "Structure your page with semantic landmarks",
			Priority:    3,
			Examples: []GuidanceExample{
				{
					Title:       "Page structure",
					BadCode:     `<div class="header">...</div>\n<div class="main">...</div>`,
					GoodCode:    `<header role="banner">...</header>\n<main role="main">...</main>`,
					Explanation: "Landmarks help screen reader users navigate efficiently",
				},
			},
			Tags: []string{"aria", "landmarks", "structure"},
		},
		{
			Title:       "Implement skip navigation",
			Description: "Provide skip links for keyboard users",
			Priority:    3,
			Examples: []GuidanceExample{
				{
					Title:       "Skip to content",
					GoodCode:    `<a href="#main" class="skip-link">Skip to main content</a>`,
					Explanation: "Skip links allow keyboard users to bypass repetitive navigation",
				},
			},
			Tags: []string{"navigation", "keyboard", "skip-links"},
		},
		{
			Title:       "Manage focus appropriately",
			Description: "Guide users through your interface logically",
			Priority:    3,
			Examples: []GuidanceExample{
				{
					Title:       "Modal focus management",
					BadCode:     `// Modal opens but focus stays on background`,
					GoodCode:    `// Move focus to modal, trap focus within modal, return focus on close`,
					Explanation: "Proper focus management is essential for keyboard and screen reader users",
				},
			},
			Tags: []string{"focus", "modals", "interaction"},
		},
	}
}

// loadDefaultGuidelines loads the default accessibility guidelines
func (guide *AccessibilityGuide) loadDefaultGuidelines() {
	// Load guidelines for each rule
	guide.guidelines["missing-alt-text"] = []GuidanceItem{
		{
			Title:       "Images need alternative text",
			Description: "Provide descriptive alt text for all images that convey information",
			Priority:    1,
			WCAG:        WCAG{Level: WCAGLevelA, Criteria: Criteria1_1_1},
			Severity:    SeverityError,
			Examples: []GuidanceExample{
				{
					Title:       "Informative image",
					BadCode:     `<img src="chart.png" />`,
					GoodCode:    `<img src="chart.png" alt="Q3 sales increased 25% to $2.3M" />`,
					Explanation: "Alt text should describe the information the image conveys, not its appearance",
				},
				{
					Title:       "Decorative image",
					BadCode:     `<img src="decoration.png" alt="decoration" />`,
					GoodCode:    `<img src="decoration.png" alt="" role="presentation" />`,
					Explanation: "Decorative images should have empty alt text and presentation role",
				},
			},
			Resources: []Resource{
				{
					Title: "Alt Text Guide",
					URL:   "https://webaim.org/articles/images/",
					Type:  "documentation",
				},
			},
			Tags: []string{"images", "alt-text", "wcag-a"},
		},
	}

	guide.guidelines["missing-form-label"] = []GuidanceItem{
		{
			Title:       "Form controls need labels",
			Description: "Every form input must have an associated label that describes its purpose",
			Priority:    1,
			WCAG:        WCAG{Level: WCAGLevelA, Criteria: Criteria3_3_2},
			Severity:    SeverityError,
			Examples: []GuidanceExample{
				{
					Title:       "Explicit label association",
					BadCode:     `<input type="text" placeholder="Enter name" />`,
					GoodCode:    `<label for="name">Full Name</label>\n<input type="text" id="name" />`,
					Explanation: "Use for/id attributes to explicitly associate labels with form controls",
				},
				{
					Title:       "Implicit label association",
					GoodCode:    `<label>Full Name\n  <input type="text" />\n</label>`,
					Explanation: "Wrapping the input in a label also creates the association",
				},
			},
			Resources: []Resource{
				{
					Title: "Form Labels",
					URL:   "https://webaim.org/techniques/forms/controls",
					Type:  "documentation",
				},
			},
			Tags: []string{"forms", "labels", "wcag-a"},
		},
	}

	guide.guidelines["missing-button-text"] = []GuidanceItem{
		{
			Title:       "Buttons need accessible names",
			Description: "Every button must have text or an aria-label that describes its action",
			Priority:    1,
			WCAG:        WCAG{Level: WCAGLevelA, Criteria: Criteria4_1_2},
			Severity:    SeverityError,
			Examples: []GuidanceExample{
				{
					Title:       "Text content",
					BadCode:     `<button><img src="close.png" /></button>`,
					GoodCode:    `<button><img src="close.png" alt="" />Close</button>`,
					Explanation: "Include visible text that describes the button's action",
				},
				{
					Title:       "ARIA label",
					BadCode:     `<button>×</button>`,
					GoodCode:    `<button aria-label="Close dialog">×</button>`,
					Explanation: "Use aria-label when the visible text isn't descriptive enough",
				},
			},
			Tags: []string{"buttons", "aria-label", "wcag-a"},
		},
	}

	guide.guidelines["low-contrast"] = []GuidanceItem{
		{
			Title:       "Ensure sufficient color contrast",
			Description: "Text must have adequate contrast against its background for readability",
			Priority:    2,
			WCAG:        WCAG{Level: WCAGLevelAA, Criteria: Criteria1_4_3},
			Severity:    SeverityWarning,
			Examples: []GuidanceExample{
				{
					Title:       "Normal text contrast",
					BadCode:     `color: #999; background: #fff; /* 2.85:1 - too low */`,
					GoodCode:    `color: #666; background: #fff; /* 5.74:1 - sufficient */`,
					Explanation: "Normal text needs 4.5:1 contrast ratio minimum",
				},
				{
					Title:       "Large text contrast",
					BadCode:     `font-size: 24px; color: #999; background: #fff; /* 2.85:1 */`,
					GoodCode:    `font-size: 24px; color: #767676; background: #fff; /* 3.98:1 */`,
					Explanation: "Large text (18pt+ or 24px+) needs 3:1 contrast ratio minimum",
				},
			},
			Resources: []Resource{
				{
					Title: "Contrast Checker",
					URL:   "https://webaim.org/resources/contrastchecker/",
					Type:  "tool",
				},
			},
			Tags: []string{"color", "contrast", "wcag-aa"},
		},
	}
}

// Component-specific guidance methods
func (guide *AccessibilityGuide) getButtonGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Button accessibility checklist",
			Description: "Essential accessibility requirements for button components",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title: "Complete button example",
					GoodCode: `<button type="button" 
       aria-describedby="help-text"
       disabled={isLoading}>
  {isLoading ? 'Loading...' : 'Submit'}
</button>
<div id="help-text">Click to submit the form</div>`,
					Explanation: "Include clear text, appropriate type, and helpful descriptions",
				},
			},
			Tags: []string{"button", "component", "checklist"},
		},
	}
}

func (guide *AccessibilityGuide) getFormGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Form accessibility pattern",
			Description: "Complete accessible form with validation and error handling",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title: "Accessible form field",
					GoodCode: `<div class="form-field">
  <label for="email">
    Email Address
    <span class="required" aria-label="required">*</span>
  </label>
  <input type="email" 
         id="email" 
         name="email"
         aria-describedby="email-help email-error"
         aria-invalid={hasError}
         required />
  <div id="email-help">We'll never share your email</div>
  {hasError && (
    <div id="email-error" role="alert">
      Please enter a valid email address
    </div>
  )}
</div>`,
					Explanation: "Include labels, help text, error messages, and ARIA attributes",
				},
			},
			Tags: []string{"form", "component", "validation"},
		},
	}
}

func (guide *AccessibilityGuide) getImageGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Image accessibility patterns",
			Description: "Different approaches for different types of images",
			Priority:    1,
			Examples: []GuidanceExample{
				{
					Title: "Complex image with description",
					GoodCode: `<figure>
  <img src="complex-chart.png" 
       alt="Monthly sales trends" 
       aria-describedby="chart-desc" />
  <figcaption id="chart-desc">
    Sales increased steadily from $100K in January to $250K in June,
    with the steepest growth in March and April.
  </figcaption>
</figure>`,
					Explanation: "For complex images, use alt + detailed description",
				},
			},
			Tags: []string{"image", "component", "complex-content"},
		},
	}
}

func (guide *AccessibilityGuide) getNavigationGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Navigation accessibility patterns",
			Description: "Create accessible navigation with proper landmarks and skip links",
			Priority:    2,
			Examples: []GuidanceExample{
				{
					Title: "Accessible navigation",
					GoodCode: `<nav role="navigation" aria-label="Main navigation">
  <a href="#main" class="skip-link">Skip to main content</a>
  <ul>
    <li><a href="/" aria-current="page">Home</a></li>
    <li><a href="/about">About</a></li>
    <li><a href="/contact">Contact</a></li>
  </ul>
</nav>`,
					Explanation: "Include skip links, proper roles, and current page indicators",
				},
			},
			Tags: []string{"navigation", "component", "landmarks"},
		},
	}
}

func (guide *AccessibilityGuide) getModalGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Modal dialog accessibility",
			Description: "Proper focus management and ARIA attributes for modals",
			Priority:    3,
			Examples: []GuidanceExample{
				{
					Title: "Accessible modal structure",
					GoodCode: `<div role="dialog" 
     aria-modal="true" 
     aria-labelledby="modal-title"
     aria-describedby="modal-desc">
  <h2 id="modal-title">Confirm Action</h2>
  <p id="modal-desc">Are you sure you want to delete this item?</p>
  <button onclick="confirmDelete()">Delete</button>
  <button onclick="closeModal()">Cancel</button>
</div>`,
					Explanation: "Use proper ARIA attributes and manage focus correctly",
				},
			},
			Tags: []string{"modal", "dialog", "focus-management"},
		},
	}
}

func (guide *AccessibilityGuide) getTableGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Data table accessibility",
			Description: "Proper table structure with headers and captions",
			Priority:    2,
			Examples: []GuidanceExample{
				{
					Title: "Accessible data table",
					GoodCode: `<table>
  <caption>Quarterly Sales Report</caption>
  <thead>
    <tr>
      <th scope="col">Quarter</th>
      <th scope="col">Sales</th>
      <th scope="col">Growth</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th scope="row">Q1</th>
      <td>$100K</td>
      <td>5%</td>
    </tr>
  </tbody>
</table>`,
					Explanation: "Use captions, proper header structure, and scope attributes",
				},
			},
			Tags: []string{"table", "data", "headers"},
		},
	}
}

func (guide *AccessibilityGuide) getGeneralGuidance() []GuidanceItem {
	return []GuidanceItem{
		{
			Title:       "Test with keyboard navigation",
			Description: "Ensure your component works without a mouse",
			Priority:    4,
			Tags:        []string{"testing", "keyboard", "general"},
		},
		{
			Title:       "Test with screen reader",
			Description: "Verify the experience with assistive technology",
			Priority:    4,
			Resources: []Resource{
				{
					Title: "Screen Reader Testing",
					URL:   "https://webaim.org/articles/screenreader_testing/",
					Type:  "guide",
				},
			},
			Tags: []string{"testing", "screen-reader", "general"},
		},
		{
			Title:       "Consider reduced motion",
			Description: "Respect user preferences for reduced motion",
			Priority:    5,
			Examples: []GuidanceExample{
				{
					Title: "Respect motion preferences",
					GoodCode: `@media (prefers-reduced-motion: reduce) {
  .animated {
    animation: none;
    transition: none;
  }
}`,
					Explanation: "Honor the user's motion preferences for a better experience",
				},
			},
			Tags: []string{"motion", "preferences", "css"},
		},
	}
}

// GetGuidanceText returns guidance formatted as plain text for CLI display
func (guide *AccessibilityGuide) GetGuidanceText(rule string) string {
	items := guide.GetGuidanceForRule(rule)
	if len(items) == 0 {
		return fmt.Sprintf("No specific guidance available for rule: %s\n\nGeneral resources:\n"+
			"• WCAG Quick Reference: https://www.w3.org/WAI/WCAG21/quickref/\n"+
			"• WebAIM Guidelines: https://webaim.org/", rule)
	}

	var text strings.Builder

	for i, item := range items {
		if i > 0 {
			text.WriteString("\n" + strings.Repeat("-", 50) + "\n")
		}

		text.WriteString(fmt.Sprintf("📋 %s\n\n", item.Title))
		text.WriteString(fmt.Sprintf("%s\n\n", item.Description))

		if len(item.Examples) > 0 {
			text.WriteString("💡 Examples:\n")
			for _, example := range item.Examples {
				text.WriteString(fmt.Sprintf("\n• %s\n", example.Title))
				if example.BadCode != "" {
					text.WriteString(fmt.Sprintf("  ❌ Bad:\n    %s\n", example.BadCode))
				}
				if example.GoodCode != "" {
					text.WriteString(fmt.Sprintf("  ✅ Good:\n    %s\n", example.GoodCode))
				}
				if example.Explanation != "" {
					text.WriteString(fmt.Sprintf("  💬 %s\n", example.Explanation))
				}
			}
			text.WriteString("\n")
		}

		if len(item.Resources) > 0 {
			text.WriteString("📚 Resources:\n")
			for _, resource := range item.Resources {
				text.WriteString(fmt.Sprintf("• %s: %s\n", resource.Title, resource.URL))
			}
			text.WriteString("\n")
		}
	}

	return text.String()
}

// GetComponentGuidanceText returns component-specific guidance as text
func (guide *AccessibilityGuide) GetComponentGuidanceText(componentName string) string {
	items := guide.GetGuidanceForComponent(componentName)
	if len(items) == 0 {
		return fmt.Sprintf(
			"No specific guidance available for component: %s\n\nConsider these general principles:\n"+
				"• Use semantic HTML elements\n• Provide text alternatives\n"+
				"• Ensure keyboard accessibility\n• Test with screen readers",
			componentName,
		)
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("🎯 Accessibility Guidance for %s Component\n\n", componentName))

	// Group by priority and show most important first
	priorityGroups := make(map[int][]GuidanceItem)
	for _, item := range items {
		priorityGroups[item.Priority] = append(priorityGroups[item.Priority], item)
	}

	for priority := 1; priority <= 5; priority++ {
		if items, exists := priorityGroups[priority]; exists {
			for _, item := range items {
				text.WriteString(fmt.Sprintf("📌 %s\n", item.Title))
				if item.Description != "" {
					text.WriteString(fmt.Sprintf("%s\n\n", item.Description))
				}

				if len(item.Examples) > 0 && len(item.Examples[0].GoodCode) > 0 {
					text.WriteString(fmt.Sprintf("Example:\n%s\n\n", item.Examples[0].GoodCode))
				}
			}
		}
	}

	return text.String()
}
