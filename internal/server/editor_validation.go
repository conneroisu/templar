package server

import (
	"fmt"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// validateTemplContent validates templ file content for syntax errors
func (s *PreviewServer) validateTemplContent(content string) ([]EditorError, []EditorWarning) {
	var errors []EditorError
	var warnings []EditorWarning

	// Basic syntax validation
	syntaxErrors := s.validateTemplSyntax(content)
	errors = append(errors, syntaxErrors...)

	// Go syntax validation for templ functions
	goErrors := s.validateGoSyntax(content)
	errors = append(errors, goErrors...)

	// Template structure validation
	structErrors, structWarnings := s.validateTemplStructure(content)
	errors = append(errors, structErrors...)
	warnings = append(warnings, structWarnings...)

	// HTML validation within templates
	htmlWarnings := s.validateHTMLContent(content)
	warnings = append(warnings, htmlWarnings...)

	return errors, warnings
}

// validateTemplSyntax validates basic templ syntax
func (s *PreviewServer) validateTemplSyntax(content string) []EditorError {
	var errors []EditorError
	lines := strings.Split(content, "\n")

	// Regular expressions for templ syntax
	templFuncRegex := regexp.MustCompile(`^templ\s+(\w+)\s*\([^)]*\)\s*\{?`)
	templEndRegex := regexp.MustCompile(`^\s*\}\s*$`)

	inTemplFunc := false
	braceCount := 0
	templFuncStart := 0

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for templ function start
		if templFuncRegex.MatchString(trimmed) {
			if inTemplFunc {
				errors = append(errors, EditorError{
					Line:     lineNum,
					Column:   1,
					Message:  "Nested templ functions are not allowed",
					Severity: "error",
					Source:   "syntax",
				})
			}
			inTemplFunc = true
			templFuncStart = lineNum
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		// Track brace counting in templ functions
		if inTemplFunc {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Check for templ function end
			if braceCount == 0 && templEndRegex.MatchString(trimmed) {
				inTemplFunc = false
				continue
			}
		}

		// Validate templ expressions
		if inTemplFunc {
			templErrors := s.validateTemplExpressions(line, lineNum)
			errors = append(errors, templErrors...)
		}
	}

	// Check for unclosed templ function
	if inTemplFunc {
		errors = append(errors, EditorError{
			Line:     templFuncStart,
			Column:   1,
			Message:  "Unclosed templ function",
			Severity: "error",
			Source:   "syntax",
		})
	}

	return errors
}

// validateTemplExpressions validates expressions within templ functions
func (s *PreviewServer) validateTemplExpressions(line string, lineNum int) []EditorError {
	var errors []EditorError

	// Check for invalid templ expressions
	templExprRegex := regexp.MustCompile(`\{[^}]*\}`)
	matches := templExprRegex.FindAllString(line, -1)

	for _, match := range matches {
		expr := strings.Trim(match, "{}")

		// Skip empty expressions
		if strings.TrimSpace(expr) == "" {
			continue
		}

		// Validate Go expressions (basic validation)
		if !s.isValidGoExpression(expr) {
			column := strings.Index(line, match) + 1
			errors = append(errors, EditorError{
				Line:     lineNum,
				Column:   column,
				Message:  fmt.Sprintf("Invalid Go expression: %s", expr),
				Severity: "error",
				Source:   "syntax",
			})
		}
	}

	return errors
}

// isValidGoExpression performs basic Go expression validation
func (s *PreviewServer) isValidGoExpression(expr string) bool {
	// Simple heuristic validation
	expr = strings.TrimSpace(expr)

	// Empty expression
	if expr == "" {
		return false
	}

	// Try to parse as a Go expression
	_, err := parser.ParseExpr(expr)
	return err == nil
}

// validateGoSyntax validates Go syntax in templ content
func (s *PreviewServer) validateGoSyntax(content string) []EditorError {
	var errors []EditorError

	// Extract Go code from templ file
	goCode := s.extractGoCode(content)
	if goCode == "" {
		return errors
	}

	// Parse Go code
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "temp.go", goCode, parser.ParseComments)

	if err != nil {
		// Convert Go parser errors to editor errors
		errors = append(errors, EditorError{
			Line:     1,
			Column:   1,
			Message:  "Go syntax error: " + err.Error(),
			Severity: "error",
			Source:   "syntax",
		})
	}

	return errors
}

// extractGoCode extracts Go code from templ content for parsing
func (s *PreviewServer) extractGoCode(content string) string {
	lines := strings.Split(content, "\n")
	var goLines []string
	inTemplFunc := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Include package declaration and imports
		if strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "import ") {
			goLines = append(goLines, line)
			continue
		}

		// Track templ function boundaries
		if strings.HasPrefix(trimmed, "templ ") {
			inTemplFunc = true
			// Convert templ function to Go function for parsing
			funcDecl := s.convertTemplFuncToGo(line)
			goLines = append(goLines, funcDecl)
			continue
		}

		// Skip content inside templ functions (HTML/template content)
		if inTemplFunc && strings.Contains(line, "}") &&
			strings.Count(line, "}") >= strings.Count(line, "{") {
			inTemplFunc = false
			goLines = append(goLines, "}")
			continue
		}

		// Include non-templ Go code
		if !inTemplFunc && !strings.Contains(line, "<") {
			goLines = append(goLines, line)
		}
	}

	return strings.Join(goLines, "\n")
}

// convertTemplFuncToGo converts templ function declaration to Go function
func (s *PreviewServer) convertTemplFuncToGo(line string) string {
	// Extract function name and parameters
	templFuncRegex := regexp.MustCompile(`templ\s+(\w+)\s*\(([^)]*)\)`)
	matches := templFuncRegex.FindStringSubmatch(line)

	if len(matches) < 3 {
		return "func tempFunc() {"
	}

	funcName := matches[1]
	params := matches[2]

	// Convert to Go function
	return fmt.Sprintf("func %s(%s) {", funcName, params)
}

// validateTemplStructure validates overall template structure
func (s *PreviewServer) validateTemplStructure(content string) ([]EditorError, []EditorWarning) {
	var errors []EditorError
	var warnings []EditorWarning

	lines := strings.Split(content, "\n")

	// Check for package declaration
	hasPackage := false
	templFuncCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "package ") {
			hasPackage = true
		}

		if strings.HasPrefix(trimmed, "templ ") {
			templFuncCount++
		}
	}

	// Validate package declaration
	if !hasPackage {
		warnings = append(warnings, EditorWarning{
			Line:    1,
			Column:  1,
			Message: "Missing package declaration",
			Code:    "missing-package",
		})
	}

	// Validate templ functions
	if templFuncCount == 0 {
		warnings = append(warnings, EditorWarning{
			Line:    1,
			Column:  1,
			Message: "No templ functions found",
			Code:    "no-templ-functions",
		})
	}

	return errors, warnings
}

// validateHTMLContent validates HTML content within templates
func (s *PreviewServer) validateHTMLContent(content string) []EditorWarning {
	var warnings []EditorWarning
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1

		// Check for unclosed HTML tags (basic check)
		htmlWarnings := s.validateHTMLTags(line, lineNum)
		warnings = append(warnings, htmlWarnings...)

		// Check for accessibility issues
		a11yWarnings := s.validateAccessibility(line, lineNum)
		warnings = append(warnings, a11yWarnings...)
	}

	return warnings
}

// validateHTMLTags validates HTML tags in a line
func (s *PreviewServer) validateHTMLTags(line string, lineNum int) []EditorWarning {
	var warnings []EditorWarning

	// Check for common HTML issues
	if strings.Contains(line, "<img") && !strings.Contains(line, "alt=") {
		warnings = append(warnings, EditorWarning{
			Line:    lineNum,
			Column:  strings.Index(line, "<img") + 1,
			Message: "Image missing alt attribute",
			Code:    "missing-alt",
		})
	}

	return warnings
}

// validateAccessibility validates accessibility concerns
func (s *PreviewServer) validateAccessibility(line string, lineNum int) []EditorWarning {
	var warnings []EditorWarning

	// Check for buttons without accessible text
	if strings.Contains(line, "<button") && !strings.Contains(line, "aria-label") &&
		!strings.Contains(line, ">") {
		warnings = append(warnings, EditorWarning{
			Line:    lineNum,
			Column:  strings.Index(line, "<button") + 1,
			Message: "Button may need accessible label",
			Code:    "button-accessibility",
		})
	}

	return warnings
}

// parseTemplParameters extracts component parameters from templ content
func (s *PreviewServer) parseTemplParameters(content string) ([]types.ParameterInfo, error) {
	var parameters []types.ParameterInfo

	// Find templ function declarations
	templFuncRegex := regexp.MustCompile(`templ\s+(\w+)\s*\(([^)]*)\)`)
	matches := templFuncRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		paramStr := strings.TrimSpace(match[2])
		if paramStr == "" {
			continue
		}

		// Parse parameters
		params := s.parseGoParameters(paramStr)
		parameters = append(parameters, params...)
	}

	return parameters, nil
}

// parseGoParameters parses Go function parameters
func (s *PreviewServer) parseGoParameters(paramStr string) []types.ParameterInfo {
	var parameters []types.ParameterInfo

	// Split parameters by comma (simple parsing)
	params := strings.Split(paramStr, ",")

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		// Parse parameter (name type format)
		parts := strings.Fields(param)
		if len(parts) >= 2 {
			name := parts[0]
			paramType := strings.Join(parts[1:], " ")

			parameters = append(parameters, types.ParameterInfo{
				Name:     name,
				Type:     paramType,
				Optional: strings.HasPrefix(paramType, "*"),
			})
		}
	}

	return parameters
}

// formatTemplContent formats templ content (basic formatting)
func (s *PreviewServer) formatTemplContent(content string) string {
	lines := strings.Split(content, "\n")
	var formatted []string
	indentLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}

		// Decrease indent for closing braces
		if strings.HasPrefix(trimmed, "}") {
			indentLevel--
		}

		// Apply indentation
		indent := strings.Repeat("\t", indentLevel)
		formatted = append(formatted, indent+trimmed)

		// Increase indent for opening braces
		if strings.HasSuffix(trimmed, "{") {
			indentLevel++
		}
	}

	return strings.Join(formatted, "\n")
}

// generateEditorSuggestions generates code suggestions for editor
func (s *PreviewServer) generateEditorSuggestions(content string) []EditorSuggestion {
	var suggestions []EditorSuggestion

	// HTML tag suggestions
	suggestions = append(suggestions, EditorSuggestion{
		Label:      "div",
		Kind:       "snippet",
		InsertText: "<div>\n\t$0\n</div>",
		Detail:     "HTML div element",
	})

	suggestions = append(suggestions, EditorSuggestion{
		Label:      "button",
		Kind:       "snippet",
		InsertText: "<button type=\"button\" onclick=\"{$1}\">\n\t$0\n</button>",
		Detail:     "HTML button element",
	})

	// Templ-specific suggestions
	suggestions = append(suggestions, EditorSuggestion{
		Label:      "templ",
		Kind:       "snippet",
		InsertText: "templ ${1:ComponentName}($2) {\n\t$0\n}",
		Detail:     "Templ component function",
	})

	suggestions = append(suggestions, EditorSuggestion{
		Label:      "if",
		Kind:       "snippet",
		InsertText: "if ${1:condition} {\n\t$0\n}",
		Detail:     "Conditional rendering",
	})

	suggestions = append(suggestions, EditorSuggestion{
		Label:      "for",
		Kind:       "snippet",
		InsertText: "for ${1:item} := range ${2:items} {\n\t$0\n}",
		Detail:     "Loop rendering",
	})

	return suggestions
}

// renderTemplContentWithProps renders template content with props for preview
func (s *PreviewServer) renderTemplContentWithProps(content string, props map[string]interface{}) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you would need to compile and execute the templ

	// For now, return the content wrapped in a preview container
	html := fmt.Sprintf(`
	<div class="templ-preview">
		<div class="preview-content">
			<!-- Templ content would be rendered here -->
			<pre><code>%s</code></pre>
		</div>
		<div class="preview-props">
			<h4>Props:</h4>
			<pre><code>%s</code></pre>
		</div>
	</div>
	`, content, s.formatPropsForDisplay(props))

	return html, nil
}

// formatPropsForDisplay formats props for display in preview
func (s *PreviewServer) formatPropsForDisplay(props map[string]interface{}) string {
	if len(props) == 0 {
		return "{}"
	}

	var lines []string
	lines = append(lines, "{")
	for key, value := range props {
		lines = append(lines, fmt.Sprintf("  %s: %v,", key, value))
	}
	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}
