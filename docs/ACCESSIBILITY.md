# Accessibility Testing Framework

Templar includes a comprehensive accessibility testing framework that helps you build inclusive components from the start. The framework provides automated WCAG compliance checking, real-time warnings, and actionable guidance for fixing accessibility issues.

## Quick Start

### Running Accessibility Audits

```bash
# Audit all components
templar audit

# Audit specific component
templar audit Button

# Get accessibility guidance without running audit
templar audit --guidance-only Button

# Generate detailed HTML report
templar audit --output html --output-file accessibility-report.html

# Show only critical issues
templar audit --severity error

# Apply automatic fixes where possible
templar audit --auto-fix
```

### Getting Accessibility Guidance

```bash
# General accessibility guidelines
templar audit --guidance-only

# Component-specific guidance
templar audit --guidance-only Button
templar audit --guidance-only Form

# Include detailed guidance in audit results
templar audit Button --show-guidance
```

## Framework Overview

The accessibility framework consists of several key components:

### 1. Accessibility Engine
- WCAG 2.1/2.2 compliance checking
- Rule-based violation detection
- Automatic fix suggestions
- HTML analysis and DOM inspection

### 2. Real-time Monitoring
- WebSocket-based live warnings
- Preview integration
- Component-specific alerts
- Performance-optimized checking

### 3. CLI Integration
- Comprehensive audit command
- Contextual guidance system
- Multiple output formats
- Batch processing support

### 4. Reporting System
- Detailed WCAG compliance reports
- Severity-based categorization
- Actionable suggestions
- HTML, JSON, and Markdown output

## WCAG Compliance Levels

The framework supports all three WCAG conformance levels:

### Level A (Basic)
- Images have alternative text
- Form controls have labels
- Page has proper language declaration
- Content is keyboard accessible

### Level AA (Standard)
- Sufficient color contrast (4.5:1 for normal text)
- Text can be resized to 200%
- Content is presented in meaningful sequence
- Focus indicators are visible

### Level AAA (Enhanced)
- Enhanced color contrast (7:1 for normal text)
- No timing requirements
- Low or no background audio
- Context-sensitive help available

## Supported Accessibility Rules

### Images and Media
- **missing-alt-text**: Images must have alternative text
- **empty-alt-text**: Decorative images should have empty alt attributes
- **complex-images**: Complex images need detailed descriptions

### Forms
- **missing-form-label**: Form controls must have associated labels
- **invalid-form-markup**: Proper form structure and semantics
- **missing-fieldset**: Related form controls need grouping

### Interactive Elements
- **missing-button-text**: Buttons must have accessible names
- **invalid-link-text**: Links need descriptive text
- **missing-focus-indicator**: Interactive elements need visible focus

### Document Structure
- **missing-headings**: Pages need proper heading structure
- **invalid-heading-order**: Headings must be in logical sequence
- **missing-landmarks**: Pages need ARIA landmarks

### Color and Contrast
- **low-contrast**: Text must have sufficient contrast
- **color-only**: Information cannot rely on color alone
- **contrast-aa**: Enhanced contrast for Level AA compliance

### Keyboard and Focus
- **keyboard-trap**: No keyboard focus traps
- **missing-skip-links**: Pages need skip navigation
- **invalid-focus-order**: Logical focus sequence required

## Component-Specific Guidance

### Button Components
```templ
// ❌ Problematic
templ IconButton() {
    <button onclick="close()">×</button>
}

// ✅ Accessible
templ IconButton() {
    <button onclick="close()" aria-label="Close dialog">
        <span aria-hidden="true">×</span>
    </button>
}
```

### Form Components
```templ
// ❌ Problematic
templ FormField(placeholder string) {
    <input type="text" placeholder={placeholder} />
}

// ✅ Accessible
templ FormField(id, label, placeholder string, required bool) {
    <div class="form-field">
        <label for={id}>
            {label}
            if required {
                <span class="required" aria-label="required">*</span>
            }
        </label>
        <input 
            type="text" 
            id={id}
            placeholder={placeholder}
            required?={required}
            aria-describedby={id + "-help"}
        />
        <div id={id + "-help"} class="help-text">
            {children...}
        </div>
    </div>
}
```

### Image Components
```templ
// ❌ Problematic
templ ProductImage(src string) {
    <img src={src} />
}

// ✅ Accessible
templ ProductImage(src, alt, description string) {
    <figure>
        <img src={src} alt={alt} />
        if description != "" {
            <figcaption>{description}</figcaption>
        }
    </figure>
}
```

### Navigation Components
```templ
// ✅ Accessible Navigation
templ MainNavigation(currentPage string) {
    <nav role="navigation" aria-label="Main navigation">
        <a href="#main" class="skip-link">Skip to main content</a>
        <ul>
            <li>
                <a href="/" 
                   class={templ.KV("current", currentPage == "home")}
                   aria-current={templ.KV("page", currentPage == "home")}>
                    Home
                </a>
            </li>
            <li>
                <a href="/about"
                   class={templ.KV("current", currentPage == "about")}
                   aria-current={templ.KV("page", currentPage == "about")}>
                    About
                </a>
            </li>
        </ul>
    </nav>
}
```

## Real-time Accessibility Monitoring

### WebSocket Integration
The framework provides real-time accessibility warnings through WebSocket connections:

```javascript
// Connect to accessibility monitoring
const ws = new WebSocket('ws://localhost:8080/ws/accessibility');

ws.onmessage = function(event) {
    const update = JSON.parse(event.data);
    
    switch (update.type) {
        case 'warning':
            console.warn('Accessibility warning:', update.message);
            showAccessibilityWarning(update);
            break;
        case 'error':
            console.error('Accessibility error:', update.message);
            showAccessibilityError(update);
            break;
        case 'success':
            console.log('Accessibility check passed:', update.message);
            break;
    }
};
```

### Preview Integration
When using the development server, accessibility warnings appear automatically:

```bash
# Start server with accessibility monitoring
templar serve --accessibility

# Warnings appear in browser console and development overlay
```

## Configuration

### Project Configuration
Add accessibility settings to your `.templar.yml`:

```yaml
accessibility:
  enabled: true
  wcag_level: "AA"
  real_time_warnings: true
  severity_threshold: "warning"
  max_warnings_per_component: 10
  auto_fix_enabled: false
  
  # Custom rules
  custom_rules:
    - id: "custom-focus-indicator"
      name: "Custom Focus Indicator"
      description: "Ensure custom focus indicators meet brand guidelines"
      
  # Rule exclusions
  exclude_rules:
    - "color-only"  # If you have special exemptions
```

### Environment Variables
```bash
export TEMPLAR_ACCESSIBILITY_ENABLED=true
export TEMPLAR_ACCESSIBILITY_WCAG_LEVEL=AA
export TEMPLAR_ACCESSIBILITY_REAL_TIME_WARNINGS=true
```

## Command Reference

### Basic Commands
```bash
# Audit commands
templar audit                           # Audit all components
templar audit Button                    # Audit specific component
templar audit --wcag-level AAA         # Use specific WCAG level
templar audit --severity error         # Show only errors

# Guidance commands
templar audit --guidance-only           # Show general guidance
templar audit --guidance-only Button   # Component-specific guidance
templar audit --show-guidance          # Include guidance in audit

# Output formats
templar audit --output json            # JSON format
templar audit --output html            # HTML report
templar audit --output markdown        # Markdown format
templar audit --output-file report.html # Save to file

# Filtering and limiting
templar audit --fixable-only           # Only auto-fixable issues
templar audit --max-violations 10      # Limit number of violations
templar audit --quiet                  # Suppress output
templar audit --verbose               # Detailed output

# Auto-fixing
templar audit --auto-fix               # Apply automatic fixes
```

### Advanced Options
```bash
# Generate comprehensive report
templar audit \
  --wcag-level AA \
  --output html \
  --output-file accessibility-audit.html \
  --include-html \
  --show-suggestions \
  --show-guidance

# CI/CD integration
templar audit \
  --output json \
  --quiet \
  --severity error \
  --output-file audit-results.json

# Development workflow
templar audit \
  --show-guidance \
  --show-suggestions \
  --auto-fix \
  --verbose
```

## Testing Strategies

### Component Development
1. **Start with semantic HTML**: Use appropriate HTML elements
2. **Add accessibility attributes**: ARIA labels, roles, and properties
3. **Test with keyboard**: Ensure full keyboard functionality
4. **Verify with audit**: Run accessibility audit before committing
5. **Manual testing**: Test with screen readers when possible

### Continuous Integration
```bash
#!/bin/bash
# .github/workflows/accessibility.yml

# Run accessibility audit
templar audit --output json --quiet --output-file results.json

# Check for critical issues
CRITICAL_ISSUES=$(cat results.json | jq '.summary.critical_impact')

if [ "$CRITICAL_ISSUES" -gt 0 ]; then
    echo "❌ Critical accessibility issues found: $CRITICAL_ISSUES"
    exit 1
else
    echo "✅ No critical accessibility issues found"
fi
```

### Manual Testing Checklist
- [ ] Can navigate entire interface with keyboard only
- [ ] Screen reader announces all content meaningfully
- [ ] All interactive elements have visible focus indicators
- [ ] Text has sufficient contrast (use color picker tools)
- [ ] Images have appropriate alternative text
- [ ] Forms are properly labeled and described
- [ ] Error messages are clear and associated with controls
- [ ] Page structure is logical with proper headings

## Browser Extension Integration

### axe-core Integration
The framework can integrate with the axe-core browser extension:

```javascript
// Custom axe configuration
const axeConfig = {
    rules: {
        'color-contrast': { enabled: true },
        'keyboard-navigation': { enabled: true },
        'aria-labels': { enabled: true }
    }
};

// Run axe audit
axe.run(document, axeConfig, (err, results) => {
    if (err) throw err;
    console.log('Accessibility results:', results);
});
```

### Lighthouse Integration
Use with Google Lighthouse for comprehensive auditing:

```bash
# Install Lighthouse CI
npm install -g @lhci/cli

# Run accessibility audit
lhci collect --url http://localhost:8080
lhci assert --config lighthouse.config.js
```

## Performance Considerations

### Optimization Settings
```yaml
accessibility:
  performance:
    check_interval: "5s"           # Real-time check frequency
    max_concurrent_checks: 5       # Parallel processing limit
    cache_results: true           # Cache audit results
    cache_size: 1000              # Maximum cached items
    timeout: "10s"                # Check timeout
```

### Resource Usage
- Accessibility checks run asynchronously to avoid blocking UI
- Results are cached to improve performance on repeated checks
- Real-time monitoring uses debouncing to reduce CPU usage
- WebSocket connections are managed efficiently

## Troubleshooting

### Common Issues

#### False Positives
```bash
# Exclude specific rules if needed
templar audit --exclude-rules "duplicate-id,color-only"

# Use custom configuration for edge cases
templar audit --config custom-accessibility.yml
```

#### Performance Issues
```bash
# Reduce check frequency
templar serve --accessibility-interval 10s

# Limit concurrent checks
templar audit --max-concurrent 2

# Disable real-time monitoring
templar serve --no-accessibility-monitoring
```

#### Integration Problems
```bash
# Test accessibility engine directly
templar audit --verbose --component Button

# Check WebSocket connection
curl -i -N -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Key: test" \
  http://localhost:8080/ws/accessibility
```

## Resources and Further Reading

### WCAG Guidelines
- [WCAG 2.1 Quick Reference](https://www.w3.org/WAI/WCAG21/quickref/)
- [WCAG 2.2 Understanding Document](https://www.w3.org/WAI/WCAG22/Understanding/)
- [How to Meet WCAG](https://www.w3.org/WAI/WCAG21/quickref/)

### Testing Tools
- [axe DevTools](https://www.deque.com/axe/devtools/)
- [WAVE Web Accessibility Evaluator](https://wave.webaim.org/)
- [Lighthouse Accessibility Audit](https://developers.google.com/web/tools/lighthouse)
- [Color Contrast Analyzers](https://www.tpgi.com/color-contrast-checker/)

### Screen Readers
- [NVDA (Windows)](https://www.nvaccess.org/)
- [JAWS (Windows)](https://www.freedomscientific.com/products/software/jaws/)
- [VoiceOver (macOS/iOS)](https://support.apple.com/guide/voiceover/)
- [TalkBack (Android)](https://support.google.com/accessibility/android/answer/6283677)

### Best Practice Guides
- [WebAIM Guidelines](https://webaim.org/)
- [A11y Project](https://www.a11yproject.com/)
- [MDN Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility)
- [Inclusive Components](https://inclusive-components.design/)

### Keyboard Testing
- [Keyboard Testing Guide](https://webaim.org/articles/keyboard/)
- [Focus Management](https://developers.google.com/web/fundamentals/accessibility/focus)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)

## Contributing

To contribute to the accessibility framework:

1. **Add new rules**: Create custom accessibility rules in `internal/accessibility/engine.go`
2. **Improve guidance**: Enhance guidance content in `internal/accessibility/guidance.go`  
3. **Extend testing**: Add comprehensive tests in `internal/accessibility/*_test.go`
4. **Update documentation**: Keep this documentation current with framework changes

### Adding Custom Rules

```go
// Example custom rule
func (engine *DefaultAccessibilityEngine) checkCustomFocusIndicator(ctx context.Context, elements []HTMLElement) ([]AccessibilityViolation, error) {
    violations := []AccessibilityViolation{}
    
    for _, element := range elements {
        if element.IsFocusable() {
            // Check for custom focus indicator requirements
            if !hasCustomFocusIndicator(element) {
                violation := engine.createViolation(
                    AccessibilityRule{
                        ID: "custom-focus-indicator",
                        Description: "Interactive elements must have visible focus indicators",
                        Impact: string(ImpactSerious),
                    },
                    element,
                    "Missing custom focus indicator",
                )
                violations = append(violations, violation)
            }
        }
    }
    
    return violations, nil
}
```

The accessibility framework is designed to grow with your needs and help you create truly inclusive user experiences.