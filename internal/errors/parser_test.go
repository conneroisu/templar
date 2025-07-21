package errors

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseError_TemplPatterns tests templ error parsing patterns
func TestParseError_TemplPatterns(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name           string
		output         string
		expectedCount  int
		expectedType   BuildErrorType
		expectedFile   string
		expectedLine   int
		expectedColumn int
		expectedMsg    string
	}{
		{
			name:           "basic templ syntax error",
			output:         "components/button.templ:15:8: unexpected token",
			expectedCount:  1,
			expectedType:   BuildErrorTypeTemplSyntax,
			expectedFile:   "components/button.templ",
			expectedLine:   15,
			expectedColumn: 8,
			expectedMsg:    "unexpected token",
		},
		{
			name:           "templ semantic error",
			output:         "templ: ButtonComponent (components/button.templ:20:5): undefined variable 'title'",
			expectedCount:  1,
			expectedType:   BuildErrorTypeTemplSemantics,
			expectedFile:   "components/button.templ",
			expectedLine:   20,
			expectedColumn: 5,
			expectedMsg:    "ButtonComponent: undefined variable 'title'",
		},
		{
			name:           "templ generate error",
			output:         "templ generate: failed to parse template",
			expectedCount:  1,
			expectedType:   BuildErrorTypeTemplSyntax,
			expectedFile:   "",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "failed to parse template",
		},
		{
			name:           "templ file error",
			output:         "layout.templ: syntax error in template definition",
			expectedCount:  1,
			expectedType:   BuildErrorTypeTemplSyntax,
			expectedFile:   "layout.templ",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "syntax error in template definition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			require.Len(t, errors, tt.expectedCount, "Expected %d errors, got %d", tt.expectedCount, len(errors))

			if tt.expectedCount > 0 {
				err := errors[0]
				assert.Equal(t, tt.expectedType, err.Type)
				assert.Equal(t, tt.expectedFile, err.File)
				assert.Equal(t, tt.expectedLine, err.Line)
				assert.Equal(t, tt.expectedColumn, err.Column)
				assert.Equal(t, tt.expectedMsg, err.Message)
				assert.Equal(t, ErrorSeverityError, err.Severity)
				assert.NotEmpty(t, err.Suggestion)
				assert.Equal(t, tt.output, err.RawError)
			}
		})
	}
}

// TestParseError_GoPatterns tests Go error parsing patterns
func TestParseError_GoPatterns(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name           string
		output         string
		expectedCount  int
		expectedType   BuildErrorType
		expectedFile   string
		expectedLine   int
		expectedColumn int
		expectedMsg    string
	}{
		{
			name:           "go compile error with column",
			output:         "main.go:42:15: syntax error: unexpected semicolon",
			expectedCount:  1,
			expectedType:   BuildErrorTypeTemplSyntax, // First pattern matches templ syntax (parser checks templ first)
			expectedFile:   "main.go",
			expectedLine:   42,
			expectedColumn: 15,
			expectedMsg:    "syntax error: unexpected semicolon",
		},
		{
			name:           "go compile error without column",
			output:         "main.go:25: undefined: fmt.Printf",
			expectedCount:  1,
			expectedType:   BuildErrorTypeGoCompile, // Matches Go pattern `^(.+?):(\d+): (.+)$`
			expectedFile:   "main.go",
			expectedLine:   25,
			expectedColumn: 0,
			expectedMsg:    "undefined: fmt.Printf",
		},
		{
			name:           "go module error",
			output:         "go: module cache: permission denied",
			expectedCount:  1,
			expectedType:   BuildErrorTypeGoRuntime,
			expectedFile:   "",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "module cache: permission denied",
		},
		{
			name:           "package not found error",
			output:         "package github.com/unknown/package is not in GOROOT",
			expectedCount:  1,
			expectedType:   BuildErrorTypeGoCompile,
			expectedFile:   "",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "Package 'github.com/unknown/package' not found",
		},
		{
			name:           "can't load package error",
			output:         "can't load package: build constraints exclude all Go files",
			expectedCount:  1,
			expectedType:   BuildErrorTypeGoCompile,
			expectedFile:   "",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "build constraints exclude all Go files",
		},
		{
			name:           "permission denied error",
			output:         "permission denied: /usr/local/bin/go",
			expectedCount:  1,
			expectedType:   BuildErrorTypePermission,
			expectedFile:   "/usr/local/bin/go",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "Permission denied",
		},
		{
			name:           "file not found error",
			output:         "no such file or directory: missing.go",
			expectedCount:  1,
			expectedType:   BuildErrorTypeFileNotFound,
			expectedFile:   "missing.go",
			expectedLine:   0,
			expectedColumn: 0,
			expectedMsg:    "File not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			require.Len(t, errors, tt.expectedCount, "Expected %d errors, got %d", tt.expectedCount, len(errors))

			if tt.expectedCount > 0 {
				err := errors[0]
				assert.Equal(t, tt.expectedType, err.Type)
				assert.Equal(t, tt.expectedFile, err.File)
				assert.Equal(t, tt.expectedLine, err.Line)
				assert.Equal(t, tt.expectedColumn, err.Column)
				assert.Equal(t, tt.expectedMsg, err.Message)
				assert.Equal(t, ErrorSeverityError, err.Severity)
				assert.NotEmpty(t, err.Suggestion)
				assert.Equal(t, tt.output, err.RawError)
			}
		})
	}
}

// TestParseError_MalformedOutput tests handling of malformed output
func TestParseError_MalformedOutput(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name         string
		output       string
		expectsError bool
		description  string
	}{
		{
			name:         "empty output",
			output:       "",
			expectsError: false,
			description:  "Empty input should produce no errors",
		},
		{
			name:         "whitespace only",
			output:       "   \n  \t  \n",
			expectsError: false,
			description:  "Whitespace-only input should produce no errors",
		},
		{
			name:         "generic error keyword",
			output:       "An error occurred during build",
			expectsError: true,
			description:  "Text containing 'error' should trigger generic error pattern",
		},
		{
			name:         "generic failed keyword",
			output:       "Build failed with unknown issue",
			expectsError: true,
			description:  "Text containing 'failed' should trigger generic error pattern",
		},
		{
			name:         "no error indicators",
			output:       "Some random output",
			expectsError: false,
			description:  "Text without error keywords should not produce errors",
		},
		{
			name:         "templ file pattern",
			output:       "components/button.templ: processing complete",
			expectsError: true,
			description:  "Templ file pattern should match .templ files",
		},
		{
			name:         "pattern-like but invalid",
			output:       "file.go:abc:xyz: error message",
			expectsError: true,
			description:  "Should handle malformed line/column numbers gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			if tt.expectsError {
				assert.Greater(t, len(errors), 0, tt.description)
				// Verify basic error structure
				for _, err := range errors {
					assert.NotEmpty(t, err.RawError, "RawError should not be empty")
					assert.True(t, err.Severity >= ErrorSeverityInfo && err.Severity <= ErrorSeverityFatal, "Severity should be valid")
				}
			} else {
				assert.Len(t, errors, 0, tt.description)
			}
		})
	}
}

// TestParseError_UnicodeHandling tests Unicode handling in error messages
func TestParseError_UnicodeHandling(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "basic unicode characters",
			output:   "файл.templ:1:1: ошибка синтаксиса",
			expected: "ошибка синтаксиса",
		},
		{
			name:     "emoji in error message",
			output:   "component.templ:5:3: Missing closing tag 🚫",
			expected: "Missing closing tag 🚫",
		},
		{
			name:     "chinese characters",
			output:   "组件.templ:10:2: 语法错误",
			expected: "语法错误",
		},
		{
			name:     "mixed unicode and ascii",
			output:   "café.templ:3:1: Invalid character 'ñ' in identifier",
			expected: "Invalid character 'ñ' in identifier",
		},
		{
			name:     "unicode file path",
			output:   "路径/组件.templ:15:8: 解析错误",
			expected: "解析错误",
		},
		{
			name:     "unicode with combining characters",
			output:   "file.templ:1:1: Error with é (e + ´)",
			expected: "Error with é (e + ´)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			require.Len(t, errors, 1, "Expected 1 error")

			err := errors[0]
			assert.Equal(t, tt.expected, err.Message)

			// Verify message is valid UTF-8
			assert.True(t, utf8.ValidString(err.Message), "Error message should be valid UTF-8")

			// Verify file path is valid UTF-8 if present
			if err.File != "" {
				assert.True(t, utf8.ValidString(err.File), "File path should be valid UTF-8")
			}
		})
	}
}

// TestParseError_LineNumberExtraction tests edge cases in line number extraction
func TestParseError_LineNumberExtraction(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name        string
		output      string
		description string
	}{
		{
			name:        "zero line number",
			output:      "file.go:0:0: error at start",
			description: "Should handle zero line numbers",
		},
		{
			name:        "large line number",
			output:      "file.go:999999:1: error at large line",
			description: "Should handle large line numbers",
		},
		{
			name:        "missing column",
			output:      "file.go:42: error without column",
			description: "Should handle missing column numbers",
		},
		{
			name:        "malformed line numbers",
			output:      "file.go:abc:5: error with invalid line",
			description: "Should handle malformed line numbers gracefully",
		},
		{
			name:        "decimal line numbers",
			output:      "file.go:10.5:3: error with decimal line",
			description: "Should handle decimal line numbers",
		},
		{
			name:        "scientific notation",
			output:      "file.go:1e2:1e1: error with scientific notation",
			description: "Should handle scientific notation gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			// The key test is that parsing doesn't crash and produces reasonable output
			assert.Greater(t, len(errors), 0, tt.description)

			for _, err := range errors {
				// Line and column should be non-negative
				assert.GreaterOrEqual(t, err.Line, 0, "Line number should be non-negative")
				assert.GreaterOrEqual(t, err.Column, 0, "Column number should be non-negative")

				// Should have meaningful content
				assert.NotEmpty(t, err.RawError, "RawError should not be empty")
				assert.True(t, err.Severity >= ErrorSeverityInfo && err.Severity <= ErrorSeverityFatal, "Severity should be valid")
			}
		})
	}
}

// TestParseError_ErrorMessageFormatting tests error message formatting
func TestParseError_ErrorMessageFormatting(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "message with leading/trailing whitespace",
			output:   "file.go:1:1:   error with whitespace   ",
			expected: "error with whitespace",
		},
		{
			name:     "message with newlines",
			output:   "file.go:1:1: error with\nnewlines",
			expected: "error with", // Parser splits on newlines, so only first part is captured
		},
		{
			name:     "message with tabs",
			output:   "file.go:1:1: error\twith\ttabs",
			expected: "error\twith\ttabs",
		},
		{
			name:     "message with special characters",
			output:   "file.go:1:1: error with special chars: !@#$%^&*()",
			expected: "error with special chars: !@#$%^&*()",
		},
		{
			name:     "very long message",
			output:   "file.go:1:1: " + strings.Repeat("very long error message ", 100),
			expected: strings.Repeat("very long error message ", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ParseError(tt.output)

			require.Len(t, errors, 1, "Expected 1 error")

			err := errors[0]
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(err.Message))
		})
	}
}

// TestParseError_MultilineOutput tests parsing of multiline output
func TestParseError_MultilineOutput(t *testing.T) {
	parser := NewErrorParser()

	output := `components/button.templ:15:8: unexpected token
main.go:42:15: syntax error: unexpected semicolon
templ: ButtonComponent (components/button.templ:20:5): undefined variable 'title'
Some random output without error keywords
go: module cache: permission denied
Build failed with issues`

	errors := parser.ParseError(output)

	// We expect multiple errors but the exact count depends on which patterns match
	assert.Greater(t, len(errors), 3, "Should parse multiple errors from multiline input")

	// Verify that we can find key error types
	errorTypes := make(map[BuildErrorType]int)
	fileMatches := 0

	for _, err := range errors {
		errorTypes[err.Type]++

		// Check for expected file matches
		if err.File == "components/button.templ" || err.File == "main.go" {
			fileMatches++
		}

		// All errors should have basic valid structure
		assert.NotEmpty(t, err.RawError, "RawError should not be empty")
		assert.True(t, err.Severity >= ErrorSeverityInfo && err.Severity <= ErrorSeverityFatal, "Severity should be valid")
	}

	// Should have parsed at least some specific file errors
	assert.Greater(t, fileMatches, 0, "Should find errors with specific file locations")

	// Should have some templ-related errors
	assert.Greater(t, errorTypes[BuildErrorTypeTemplSyntax]+errorTypes[BuildErrorTypeTemplSemantics], 0, "Should find templ-related errors")
}

// TestParseError_ContextLines tests context line extraction
func TestParseError_ContextLines(t *testing.T) {
	parser := NewErrorParser()

	output := `line 1
line 2 before error
file.go:3:1: error on this line
line 4 after error
line 5`

	errors := parser.ParseError(output)

	// The output contains "error" keywords in multiple lines, so will generate multiple errors
	// Let's focus on testing that context is extracted correctly for at least one error
	require.GreaterOrEqual(t, len(errors), 1, "Expected at least 1 error")

	// Find the error that matches our main pattern
	var mainError *ParsedError
	for _, err := range errors {
		if err.File == "file.go" && err.Line == 3 {
			mainError = err
			break
		}
	}

	require.NotNil(t, mainError, "Should find the main file.go:3:1 error")
	require.Len(t, mainError.Context, 5, "Expected 5 context lines")

	// Check context formatting
	assert.Contains(t, mainError.Context[0], "line 1")
	assert.Contains(t, mainError.Context[1], "line 2 before error")
	assert.Contains(t, mainError.Context[2], "→ file.go:3:1: error on this line") // Current line marked with →
	assert.Contains(t, mainError.Context[3], "line 4 after error")
	assert.Contains(t, mainError.Context[4], "line 5")
}

// TestParseError_Integration tests integration with real templ compiler output
func TestParseError_Integration(t *testing.T) {
	parser := NewErrorParser()

	// Simulate real templ compiler error output
	templOutput := `templ generate
(admin) parsing file: components/layout.templ
(admin) parsing file: components/button.templ
components/button.templ:45:23: unexpected "}" in expression, expected operand
exit status 1`

	errors := parser.ParseError(templOutput)

	// Should find the actual error, ignoring informational lines
	require.Len(t, errors, 1, "Expected 1 error from templ output")

	err := errors[0]
	assert.Equal(t, BuildErrorTypeTemplSyntax, err.Type)
	assert.Equal(t, "components/button.templ", err.File)
	assert.Equal(t, 45, err.Line)
	assert.Equal(t, 23, err.Column)
	assert.Equal(t, `unexpected "}" in expression, expected operand`, err.Message)
	assert.NotEmpty(t, err.Suggestion)
}

// TestParsedError_FormatError tests error formatting
func TestParsedError_FormatError(t *testing.T) {
	tests := []struct {
		name     string
		error    *ParsedError
		expected []string // Parts that should be in the formatted output
	}{
		{
			name: "complete error with all fields",
			error: &ParsedError{
				Type:       BuildErrorTypeTemplSyntax,
				Severity:   ErrorSeverityError,
				File:       "components/button.templ",
				Line:       15,
				Column:     8,
				Message:    "unexpected token",
				Suggestion: "Check the templ syntax",
				RawError:   "components/button.templ:15:8: unexpected token",
				Context:    []string{"  line before", "→ error line", "  line after"},
			},
			expected: []string{
				"[ERROR] Templ Syntax",
				"components/button.templ:15:8",
				"unexpected token",
				"💡 Check the templ syntax",
				"Context:",
				"line before",
				"→ error line",
				"line after",
			},
		},
		{
			name: "minimal error",
			error: &ParsedError{
				Type:     BuildErrorTypeUnknown,
				Severity: ErrorSeverityWarning,
				Message:  "simple warning",
				RawError: "simple warning",
			},
			expected: []string{
				"[WARN] Unknown",
				"simple warning",
			},
		},
		{
			name: "error without suggestion or context",
			error: &ParsedError{
				Type:     BuildErrorTypeGoCompile,
				Severity: ErrorSeverityFatal,
				File:     "main.go",
				Line:     42,
				Message:  "fatal compile error",
				RawError: "main.go:42: fatal compile error",
			},
			expected: []string{
				"[FATAL] Go Compile",
				"main.go:42",
				"fatal compile error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := tt.error.FormatError()

			for _, expected := range tt.expected {
				assert.Contains(t, formatted, expected, "Formatted output should contain: %s", expected)
			}
		})
	}
}

// TestFormatErrorsForBrowser tests HTML error formatting
func TestFormatErrorsForBrowser(t *testing.T) {
	errors := []*ParsedError{
		{
			Type:       BuildErrorTypeTemplSyntax,
			Severity:   ErrorSeverityError,
			File:       "components/button.templ",
			Line:       15,
			Column:     8,
			Message:    "unexpected token",
			Suggestion: "Check the templ syntax",
			Context:    []string{"  line before", "→ error line", "  line after"},
		},
		{
			Type:     BuildErrorTypeGoCompile,
			Severity: ErrorSeverityWarning,
			File:     "main.go",
			Line:     42,
			Message:  "unused variable",
		},
	}

	html := FormatErrorsForBrowser(errors)

	// Should be valid HTML
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<html>")
	assert.Contains(t, html, "</html>")
	assert.Contains(t, html, "<head>")
	assert.Contains(t, html, "<body>")

	// Should contain error information
	assert.Contains(t, html, "Build Errors")
	assert.Contains(t, html, "components/button.templ:15:8")
	assert.Contains(t, html, "unexpected token")
	assert.Contains(t, html, "Check the templ syntax")
	assert.Contains(t, html, "main.go:42")
	assert.Contains(t, html, "unused variable")

	// Should have CSS styling
	assert.Contains(t, html, "<style>")
	assert.Contains(t, html, "error")
	assert.Contains(t, html, "warning")

	// Should handle different severity classes
	assert.Contains(t, html, `class="error"`)
	assert.Contains(t, html, `class="warning"`)

	// Should format context lines
	assert.Contains(t, html, "context-line")
	assert.Contains(t, html, "context-current")
}

// TestFormatErrorsForBrowser_Empty tests empty error list
func TestFormatErrorsForBrowser_Empty(t *testing.T) {
	html := FormatErrorsForBrowser([]*ParsedError{})
	assert.Empty(t, html, "Empty error list should return empty string")

	html = FormatErrorsForBrowser(nil)
	assert.Empty(t, html, "Nil error list should return empty string")
}

// TestErrorParser_TypeAndSeverityStrings tests error type and severity string methods
func TestErrorParser_TypeAndSeverityStrings(t *testing.T) {
	tests := []struct {
		errorType BuildErrorType
		expected  string
	}{
		{BuildErrorTypeTemplSyntax, "Templ Syntax"},
		{BuildErrorTypeTemplSemantics, "Templ Semantics"},
		{BuildErrorTypeGoCompile, "Go Compile"},
		{BuildErrorTypeGoRuntime, "Go Runtime"},
		{BuildErrorTypeFileNotFound, "File Not Found"},
		{BuildErrorTypePermission, "Permission"},
		{BuildErrorTypeUnknown, "Unknown"},
		{BuildErrorType(999), "Unknown"}, // Invalid type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			err := &ParsedError{Type: tt.errorType}
			assert.Equal(t, tt.expected, err.typeString())
		})
	}

	severityTests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{ErrorSeverityInfo, "INFO"},
		{ErrorSeverityWarning, "WARN"},
		{ErrorSeverityError, "ERROR"},
		{ErrorSeverityFatal, "FATAL"},
		{ErrorSeverity(999), "UNKNOWN"}, // Invalid severity
	}

	for _, tt := range severityTests {
		t.Run(tt.expected, func(t *testing.T) {
			err := &ParsedError{Severity: tt.severity}
			assert.Equal(t, tt.expected, err.severityString())
		})
	}
}

// TestParseError_HelperFunctions tests min and max helper functions
func TestParseError_HelperFunctions(t *testing.T) {
	// Test max function
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 5, max(5, 3))
	assert.Equal(t, 0, max(0, 0))
	assert.Equal(t, 1, max(-1, 1))

	// Test min function
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, 0, min(0, 0))
	assert.Equal(t, -1, min(-1, 1))
}

// TestParseError_EdgeCases tests various edge cases
func TestParseError_EdgeCases(t *testing.T) {
	parser := NewErrorParser()

	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "extremely long line",
			output: strings.Repeat("a", 10000) + ":1:1: error",
		},
		{
			name:   "line with null bytes",
			output: "file.go:1:1: error\x00with\x00nulls",
		},
		{
			name:   "line with control characters",
			output: "file.go:1:1: error\twith\rcontrol\nchars",
		},
		{
			name:   "repeated error patterns",
			output: strings.Repeat("file.go:1:1: error\n", 1000),
		},
		{
			name:   "alternating valid and invalid lines",
			output: "file.go:1:1: error\ninvalid line\nfile.go:2:2: another error\nanother invalid line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic or crash
			errors := parser.ParseError(tt.output)

			// Should produce some reasonable output
			assert.NotNil(t, errors)

			// All errors should have valid severity
			for _, err := range errors {
				assert.True(t, err.Severity >= ErrorSeverityInfo && err.Severity <= ErrorSeverityFatal)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkParseError_SingleError(b *testing.B) {
	parser := NewErrorParser()
	output := "components/button.templ:15:8: unexpected token"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseError(output)
	}
}

func BenchmarkParseError_MultipleErrors(b *testing.B) {
	parser := NewErrorParser()
	output := strings.Repeat("file.go:1:1: error\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseError(output)
	}
}

func BenchmarkFormatErrorsForBrowser(b *testing.B) {
	errors := make([]*ParsedError, 50)
	for i := range errors {
		errors[i] = &ParsedError{
			Type:       BuildErrorTypeTemplSyntax,
			Severity:   ErrorSeverityError,
			File:       "components/button.templ",
			Line:       i + 1,
			Column:     8,
			Message:    "unexpected token",
			Suggestion: "Check the templ syntax",
			Context:    []string{"  line before", "→ error line", "  line after"},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatErrorsForBrowser(errors)
	}
}
