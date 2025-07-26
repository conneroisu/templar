// Package errors provides build error parsing and HTML overlay generation
// for development-friendly error reporting.
//
// The error system parses compiler output from templ and Go build tools,
// extracts structured error information including file paths, line numbers,
// and error messages, and generates HTML error overlays for real-time
// debugging. It supports race-safe error collection and provides formatted
// error display with severity classification and context information.
package errors

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// BuildErrorType represents different types of build errors
type BuildErrorType int

const (
	BuildErrorTypeUnknown BuildErrorType = iota
	BuildErrorTypeTemplSyntax
	BuildErrorTypeTemplSemantics
	BuildErrorTypeGoCompile
	BuildErrorTypeGoRuntime
	BuildErrorTypeFileNotFound
	BuildErrorTypePermission
)

// ParsedError represents a parsed error with structured information
type ParsedError struct {
	Type       BuildErrorType `json:"type"`
	Severity   ErrorSeverity  `json:"severity"`
	Component  string         `json:"component"`
	File       string         `json:"file"`
	Line       int            `json:"line"`
	Column     int            `json:"column"`
	Message    string         `json:"message"`
	Suggestion string         `json:"suggestion,omitempty"`
	RawError   string         `json:"raw_error"`
	Context    []string       `json:"context,omitempty"`
}

// ErrorParser parses templ and Go errors into structured format
type ErrorParser struct {
	templPatterns []errorPattern
	goPatterns    []errorPattern
}

type errorPattern struct {
	regex       *regexp.Regexp
	errorType   BuildErrorType
	severity    ErrorSeverity
	suggestion  string
	parseFields func(matches []string) (file string, line int, column int, message string)
}

// NewErrorParser creates a new error parser
func NewErrorParser() *ErrorParser {
	return &ErrorParser{
		templPatterns: buildTemplPatterns(),
		goPatterns:    buildGoPatterns(),
	}
}

// ParseError parses a build error output into structured errors
func (ep *ErrorParser) ParseError(output string) []*ParsedError {
	var errors []*ParsedError

	lines := strings.Split(output, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try templ patterns first
		if err := ep.tryParseWithPatterns(line, ep.templPatterns); err != nil {
			// Add context lines
			err.Context = ep.getContextLines(lines, i, 2)
			errors = append(errors, err)
			continue
		}

		// Try Go patterns
		if err := ep.tryParseWithPatterns(line, ep.goPatterns); err != nil {
			// Add context lines
			err.Context = ep.getContextLines(lines, i, 2)
			errors = append(errors, err)
			continue
		}

		// If no pattern matches, create a generic error
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "failed") {
			errors = append(errors, &ParsedError{
				Type:     BuildErrorTypeUnknown,
				Severity: ErrorSeverityError,
				Message:  line,
				RawError: line,
				Context:  ep.getContextLines(lines, i, 1),
			})
		}
	}

	return errors
}

func (ep *ErrorParser) tryParseWithPatterns(line string, patterns []errorPattern) *ParsedError {
	for _, pattern := range patterns {
		matches := pattern.regex.FindStringSubmatch(line)
		if matches != nil {
			file, lineNum, column, message := pattern.parseFields(matches)

			return &ParsedError{
				Type:       pattern.errorType,
				Severity:   pattern.severity,
				File:       file,
				Line:       lineNum,
				Column:     column,
				Message:    message,
				Suggestion: pattern.suggestion,
				RawError:   line,
			}
		}
	}
	return nil
}

func (ep *ErrorParser) getContextLines(lines []string, index int, radius int) []string {
	start := max(0, index-radius)
	end := min(len(lines), index+radius+1)

	var context []string
	for i := start; i < end; i++ {
		prefix := "  "
		if i == index {
			prefix = "→ "
		}
		context = append(context, fmt.Sprintf("%s%s", prefix, lines[i]))
	}

	return context
}

func buildTemplPatterns() []errorPattern {
	return []errorPattern{
		{
			regex:      regexp.MustCompile(`^(.+?):(\d+):(\d+): (.+)$`),
			errorType:  BuildErrorTypeTemplSyntax,
			severity:   ErrorSeverityError,
			suggestion: "Check the templ syntax at the specified location",
			parseFields: func(matches []string) (string, int, int, string) {
				file := matches[1]
				line, _ := strconv.Atoi(matches[2])
				column, _ := strconv.Atoi(matches[3])
				message := matches[4]
				return file, line, column, message
			},
		},
		{
			regex:      regexp.MustCompile(`^templ: (.+?) \((.+?):(\d+):(\d+)\): (.+)$`),
			errorType:  BuildErrorTypeTemplSemantics,
			severity:   ErrorSeverityError,
			suggestion: "Check the templ component definition and usage",
			parseFields: func(matches []string) (string, int, int, string) {
				file := matches[2]
				line, _ := strconv.Atoi(matches[3])
				column, _ := strconv.Atoi(matches[4])
				message := fmt.Sprintf("%s: %s", matches[1], matches[5])
				return file, line, column, message
			},
		},
		{
			regex:      regexp.MustCompile(`^templ generate: (.+)$`),
			errorType:  BuildErrorTypeTemplSyntax,
			severity:   ErrorSeverityError,
			suggestion: "Run 'templ generate' to see detailed error information",
			parseFields: func(matches []string) (string, int, int, string) {
				return "", 0, 0, matches[1]
			},
		},
		{
			regex:      regexp.MustCompile(`^(.+\.templ): (.+)$`),
			errorType:  BuildErrorTypeTemplSyntax,
			severity:   ErrorSeverityError,
			suggestion: "Check the templ file for syntax errors",
			parseFields: func(matches []string) (string, int, int, string) {
				return matches[1], 0, 0, matches[2]
			},
		},
	}
}

func buildGoPatterns() []errorPattern {
	return []errorPattern{
		{
			regex:      regexp.MustCompile(`^(.+?):(\d+):(\d+): (.+)$`),
			errorType:  BuildErrorTypeGoCompile,
			severity:   ErrorSeverityError,
			suggestion: "Check the Go syntax and imports",
			parseFields: func(matches []string) (string, int, int, string) {
				file := matches[1]
				line, _ := strconv.Atoi(matches[2])
				column, _ := strconv.Atoi(matches[3])
				message := matches[4]
				return file, line, column, message
			},
		},
		{
			regex:      regexp.MustCompile(`^(.+?):(\d+): (.+)$`),
			errorType:  BuildErrorTypeGoCompile,
			severity:   ErrorSeverityError,
			suggestion: "Check the Go syntax and types",
			parseFields: func(matches []string) (string, int, int, string) {
				file := matches[1]
				line, _ := strconv.Atoi(matches[2])
				message := matches[3]
				return file, line, 0, message
			},
		},
		{
			regex:      regexp.MustCompile(`^go: (.+)$`),
			errorType:  BuildErrorTypeGoRuntime,
			severity:   ErrorSeverityError,
			suggestion: "Check Go module dependencies and configuration",
			parseFields: func(matches []string) (string, int, int, string) {
				return "", 0, 0, matches[1]
			},
		},
		{
			regex:      regexp.MustCompile(`^package (.+) is not in GOROOT`),
			errorType:  BuildErrorTypeGoCompile,
			severity:   ErrorSeverityError,
			suggestion: "Check if the package is properly imported or installed",
			parseFields: func(matches []string) (string, int, int, string) {
				return "", 0, 0, fmt.Sprintf("Package '%s' not found", matches[1])
			},
		},
		{
			regex:      regexp.MustCompile(`^can't load package: (.+)$`),
			errorType:  BuildErrorTypeGoCompile,
			severity:   ErrorSeverityError,
			suggestion: "Check if the package exists and is properly configured",
			parseFields: func(matches []string) (string, int, int, string) {
				return "", 0, 0, matches[1]
			},
		},
		{
			regex:      regexp.MustCompile(`^permission denied: (.+)$`),
			errorType:  BuildErrorTypePermission,
			severity:   ErrorSeverityError,
			suggestion: "Check file permissions and ownership",
			parseFields: func(matches []string) (string, int, int, string) {
				return matches[1], 0, 0, "Permission denied"
			},
		},
		{
			regex:      regexp.MustCompile(`^no such file or directory: (.+)$`),
			errorType:  BuildErrorTypeFileNotFound,
			severity:   ErrorSeverityError,
			suggestion: "Check if the file exists and the path is correct",
			parseFields: func(matches []string) (string, int, int, string) {
				return matches[1], 0, 0, "File not found"
			},
		},
	}
}

// FormatError formats a parsed error for display
func (pe *ParsedError) FormatError() string {
	var builder strings.Builder

	// Error type and severity
	builder.WriteString(fmt.Sprintf("[%s] %s", pe.severityString(), pe.typeString()))

	// Location information
	if pe.File != "" {
		builder.WriteString(fmt.Sprintf(" in %s", pe.File))
		if pe.Line > 0 {
			builder.WriteString(fmt.Sprintf(":%d", pe.Line))
			if pe.Column > 0 {
				builder.WriteString(fmt.Sprintf(":%d", pe.Column))
			}
		}
	}

	builder.WriteString("\n")

	// Message
	builder.WriteString(fmt.Sprintf("  %s\n", pe.Message))

	// Suggestion
	if pe.Suggestion != "" {
		builder.WriteString(fmt.Sprintf("  💡 %s\n", pe.Suggestion))
	}

	// Context
	if len(pe.Context) > 0 {
		builder.WriteString("  Context:\n")
		for _, line := range pe.Context {
			builder.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}

	return builder.String()
}

// FormatErrorsForBrowser formats errors for browser display
func FormatErrorsForBrowser(errors []*ParsedError) string {
	if len(errors) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(`
<!DOCTYPE html>
<html>
<head>
    <title>Build Errors</title>
    <style>
        body { font-family: monospace; margin: 20px; background-color: #1e1e1e; color: #ffffff; }
        .error { margin: 20px 0; padding: 15px; border-left: 4px solid #ff4444; background-color: #2d2d2d; }
        .warning { border-left-color: #ffaa00; }
        .info { border-left-color: #4444ff; }
        .error-header { font-weight: bold; font-size: 1.1em; margin-bottom: 10px; }
        .error-location { color: #88ccff; font-size: 0.9em; }
        .error-message { margin: 10px 0; }
        .error-suggestion { color: #88ff88; font-style: italic; margin-top: 10px; }
        .error-context { margin-top: 10px; padding: 10px; background-color: #1a1a1a; border-radius: 4px; }
        .context-line { margin: 2px 0; }
        .context-current { color: #ff4444; font-weight: bold; }
    </style>
</head>
<body>
    <h1>Build Errors</h1>
`)

	for _, err := range errors {
		cssClass := "error"
		switch err.Severity {
		case ErrorSeverityWarning:
			cssClass = "warning"
		case ErrorSeverityInfo:
			cssClass = "info"
		}

		builder.WriteString(fmt.Sprintf(`    <div class="%s">`, cssClass))
		builder.WriteString(
			fmt.Sprintf(
				`        <div class="error-header">[%s] %s</div>`,
				err.severityString(),
				err.typeString(),
			),
		)

		if err.File != "" {
			builder.WriteString(fmt.Sprintf(`        <div class="error-location">%s`, err.File))
			if err.Line > 0 {
				builder.WriteString(fmt.Sprintf(`:%d`, err.Line))
				if err.Column > 0 {
					builder.WriteString(fmt.Sprintf(`:%d`, err.Column))
				}
			}
			builder.WriteString(`</div>`)
		}

		builder.WriteString(fmt.Sprintf(`        <div class="error-message">%s</div>`, err.Message))

		if err.Suggestion != "" {
			builder.WriteString(
				fmt.Sprintf(`        <div class="error-suggestion">💡 %s</div>`, err.Suggestion),
			)
		}

		if len(err.Context) > 0 {
			builder.WriteString(`        <div class="error-context">`)
			for _, line := range err.Context {
				class := "context-line"
				if strings.HasPrefix(line, "→ ") {
					class = "context-current"
				}
				builder.WriteString(
					fmt.Sprintf(`            <div class="%s">%s</div>`, class, line),
				)
			}
			builder.WriteString(`        </div>`)
		}

		builder.WriteString(`    </div>`)
	}

	builder.WriteString(`
</body>
</html>`)

	return builder.String()
}

func (pe *ParsedError) typeString() string {
	switch pe.Type {
	case BuildErrorTypeTemplSyntax:
		return "Templ Syntax"
	case BuildErrorTypeTemplSemantics:
		return "Templ Semantics"
	case BuildErrorTypeGoCompile:
		return "Go Compile"
	case BuildErrorTypeGoRuntime:
		return "Go Runtime"
	case BuildErrorTypeFileNotFound:
		return "File Not Found"
	case BuildErrorTypePermission:
		return "Permission"
	default:
		return "Unknown"
	}
}

func (pe *ParsedError) severityString() string {
	switch pe.Severity {
	case ErrorSeverityInfo:
		return "INFO"
	case ErrorSeverityWarning:
		return "WARN"
	case ErrorSeverityError:
		return "ERROR"
	case ErrorSeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
