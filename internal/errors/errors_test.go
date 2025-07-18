package errors

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorSeverityString(t *testing.T) {
	testCases := []struct {
		severity ErrorSeverity
		expected string
	}{
		{ErrorSeverityInfo, "info"},
		{ErrorSeverityWarning, "warning"},
		{ErrorSeverityError, "error"},
		{ErrorSeverityFatal, "fatal"},
		{ErrorSeverity(999), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.severity.String())
		})
	}
}

func TestBuildErrorError(t *testing.T) {
	err := BuildError{
		Component: "TestComponent",
		File:      "test.go",
		Line:      10,
		Column:    5,
		Message:   "syntax error",
		Severity:  ErrorSeverityError,
		Timestamp: time.Now(),
	}

	errorStr := err.Error()
	assert.Contains(t, errorStr, "test.go")
	assert.Contains(t, errorStr, "10")
	assert.Contains(t, errorStr, "5")
	assert.Contains(t, errorStr, "error")
	assert.Contains(t, errorStr, "syntax error")
}

func TestNewErrorCollector(t *testing.T) {
	collector := NewErrorCollector()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.errors)
	assert.Empty(t, collector.errors)
	assert.False(t, collector.HasErrors())
}

func TestErrorCollectorAdd(t *testing.T) {
	collector := NewErrorCollector()
	
	err := BuildError{
		Component: "TestComponent",
		File:      "test.go",
		Line:      10,
		Column:    5,
		Message:   "syntax error",
		Severity:  ErrorSeverityError,
	}
	
	before := time.Now()
	collector.Add(err)
	after := time.Now()
	
	assert.True(t, collector.HasErrors())
	assert.Len(t, collector.GetErrors(), 1)
	
	addedErr := collector.GetErrors()[0]
	assert.Equal(t, "TestComponent", addedErr.Component)
	assert.Equal(t, "test.go", addedErr.File)
	assert.Equal(t, 10, addedErr.Line)
	assert.Equal(t, 5, addedErr.Column)
	assert.Equal(t, "syntax error", addedErr.Message)
	assert.Equal(t, ErrorSeverityError, addedErr.Severity)
	
	// Check that timestamp was set
	assert.True(t, addedErr.Timestamp.After(before) || addedErr.Timestamp.Equal(before))
	assert.True(t, addedErr.Timestamp.Before(after) || addedErr.Timestamp.Equal(after))
}

func TestErrorCollectorGetErrors(t *testing.T) {
	collector := NewErrorCollector()
	
	err1 := BuildError{
		Component: "Component1",
		File:      "file1.go",
		Message:   "error 1",
		Severity:  ErrorSeverityError,
	}
	
	err2 := BuildError{
		Component: "Component2",
		File:      "file2.go",
		Message:   "error 2",
		Severity:  ErrorSeverityWarning,
	}
	
	collector.Add(err1)
	collector.Add(err2)
	
	errors := collector.GetErrors()
	assert.Len(t, errors, 2)
	assert.Equal(t, "error 1", errors[0].Message)
	assert.Equal(t, "error 2", errors[1].Message)
}

func TestErrorCollectorHasErrors(t *testing.T) {
	collector := NewErrorCollector()
	
	// No errors initially
	assert.False(t, collector.HasErrors())
	
	// Add an error
	err := BuildError{
		Message:  "test error",
		Severity: ErrorSeverityError,
	}
	collector.Add(err)
	
	// Should have errors now
	assert.True(t, collector.HasErrors())
	
	// Clear errors
	collector.Clear()
	
	// Should not have errors after clearing
	assert.False(t, collector.HasErrors())
}

func TestErrorCollectorClear(t *testing.T) {
	collector := NewErrorCollector()
	
	// Add some errors
	for i := 0; i < 3; i++ {
		err := BuildError{
			Message:  "test error",
			Severity: ErrorSeverityError,
		}
		collector.Add(err)
	}
	
	assert.True(t, collector.HasErrors())
	assert.Len(t, collector.GetErrors(), 3)
	
	// Clear errors
	collector.Clear()
	
	assert.False(t, collector.HasErrors())
	assert.Empty(t, collector.GetErrors())
}

func TestErrorCollectorGetErrorsByFile(t *testing.T) {
	collector := NewErrorCollector()
	
	err1 := BuildError{
		File:     "file1.go",
		Message:  "error in file1",
		Severity: ErrorSeverityError,
	}
	
	err2 := BuildError{
		File:     "file2.go",
		Message:  "error in file2",
		Severity: ErrorSeverityWarning,
	}
	
	err3 := BuildError{
		File:     "file1.go",
		Message:  "another error in file1",
		Severity: ErrorSeverityError,
	}
	
	collector.Add(err1)
	collector.Add(err2)
	collector.Add(err3)
	
	// Get errors for file1.go
	file1Errors := collector.GetErrorsByFile("file1.go")
	assert.Len(t, file1Errors, 2)
	assert.Equal(t, "error in file1", file1Errors[0].Message)
	assert.Equal(t, "another error in file1", file1Errors[1].Message)
	
	// Get errors for file2.go
	file2Errors := collector.GetErrorsByFile("file2.go")
	assert.Len(t, file2Errors, 1)
	assert.Equal(t, "error in file2", file2Errors[0].Message)
	
	// Get errors for non-existent file
	noErrors := collector.GetErrorsByFile("nonexistent.go")
	assert.Empty(t, noErrors)
}

func TestErrorCollectorGetErrorsByComponent(t *testing.T) {
	collector := NewErrorCollector()
	
	err1 := BuildError{
		Component: "Component1",
		Message:   "error in component1",
		Severity:  ErrorSeverityError,
	}
	
	err2 := BuildError{
		Component: "Component2",
		Message:   "error in component2",
		Severity:  ErrorSeverityWarning,
	}
	
	err3 := BuildError{
		Component: "Component1",
		Message:   "another error in component1",
		Severity:  ErrorSeverityError,
	}
	
	collector.Add(err1)
	collector.Add(err2)
	collector.Add(err3)
	
	// Get errors for Component1
	comp1Errors := collector.GetErrorsByComponent("Component1")
	assert.Len(t, comp1Errors, 2)
	assert.Equal(t, "error in component1", comp1Errors[0].Message)
	assert.Equal(t, "another error in component1", comp1Errors[1].Message)
	
	// Get errors for Component2
	comp2Errors := collector.GetErrorsByComponent("Component2")
	assert.Len(t, comp2Errors, 1)
	assert.Equal(t, "error in component2", comp2Errors[0].Message)
	
	// Get errors for non-existent component
	noErrors := collector.GetErrorsByComponent("NonExistentComponent")
	assert.Empty(t, noErrors)
}

func TestErrorCollectorErrorOverlayEmpty(t *testing.T) {
	collector := NewErrorCollector()
	
	// Should return empty string when no errors
	overlay := collector.ErrorOverlay()
	assert.Empty(t, overlay)
}

func TestErrorCollectorErrorOverlay(t *testing.T) {
	collector := NewErrorCollector()
	
	err1 := BuildError{
		Component: "TestComponent",
		File:      "test.go",
		Line:      10,
		Column:    5,
		Message:   "syntax error",
		Severity:  ErrorSeverityError,
		Timestamp: time.Now(),
	}
	
	err2 := BuildError{
		Component: "AnotherComponent",
		File:      "another.go",
		Line:      20,
		Column:    10,
		Message:   "warning message",
		Severity:  ErrorSeverityWarning,
		Timestamp: time.Now(),
	}
	
	collector.Add(err1)
	collector.Add(err2)
	
	overlay := collector.ErrorOverlay()
	
	// Check that overlay contains expected elements
	assert.Contains(t, overlay, "templar-error-overlay")
	assert.Contains(t, overlay, "Build Errors")
	assert.Contains(t, overlay, "syntax error")
	assert.Contains(t, overlay, "warning message")
	assert.Contains(t, overlay, "test.go")
	assert.Contains(t, overlay, "another.go")
	assert.Contains(t, overlay, "error")
	assert.Contains(t, overlay, "warning")
	assert.Contains(t, overlay, "10:5")
	assert.Contains(t, overlay, "20:10")
	
	// Check that it's valid HTML structure
	assert.Contains(t, overlay, "<div")
	assert.Contains(t, overlay, "</div>")
	assert.Contains(t, overlay, "Close")
}

func TestErrorOverlayDifferentSeverities(t *testing.T) {
	collector := NewErrorCollector()
	
	testCases := []struct {
		severity ErrorSeverity
		color    string
	}{
		{ErrorSeverityError, "#ff6b6b"},
		{ErrorSeverityWarning, "#feca57"},
		{ErrorSeverityInfo, "#48dbfb"},
		{ErrorSeverityFatal, "#ff6b6b"}, // Fatal uses same color as error
	}
	
	for _, tc := range testCases {
		collector.Clear()
		
		err := BuildError{
			Component: "TestComponent",
			File:      "test.go",
			Line:      1,
			Column:    1,
			Message:   "test message",
			Severity:  tc.severity,
			Timestamp: time.Now(),
		}
		
		collector.Add(err)
		overlay := collector.ErrorOverlay()
		
		// Check that the appropriate color is used
		assert.Contains(t, overlay, tc.color)
		assert.Contains(t, overlay, tc.severity.String())
	}
}

func TestParseTemplError(t *testing.T) {
	testCases := []struct {
		name      string
		output    []byte
		component string
		expected  int
	}{
		{
			name:      "Empty output",
			output:    []byte(""),
			component: "TestComponent",
			expected:  0,
		},
		{
			name:      "Error output",
			output:    []byte("compilation failed: syntax error"),
			component: "TestComponent",
			expected:  1,
		},
		{
			name:      "Multi-line output",
			output:    []byte("error: line 1\nwarning: line 2"),
			component: "TestComponent",
			expected:  1,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors := ParseTemplError(tc.output, tc.component)
			assert.Len(t, errors, tc.expected)
			
			if tc.expected > 0 {
				err := errors[0]
				assert.Equal(t, tc.component, err.Component)
				assert.Equal(t, "unknown", err.File)
				assert.Equal(t, 0, err.Line)
				assert.Equal(t, 0, err.Column)
				assert.Equal(t, string(tc.output), err.Message)
				assert.Equal(t, ErrorSeverityError, err.Severity)
				assert.False(t, err.Timestamp.IsZero())
			}
		})
	}
}

func TestParseTemplErrorSecurity(t *testing.T) {
	// Test with potentially malicious input
	maliciousInputs := [][]byte{
		[]byte("<script>alert('xss')</script>"),
		[]byte("'; DROP TABLE users; --"),
		[]byte("../../../etc/passwd"),
		[]byte(strings.Repeat("A", 10000)), // Large input
	}
	
	for i, input := range maliciousInputs {
		t.Run(fmt.Sprintf("malicious_input_%d", i), func(t *testing.T) {
			errors := ParseTemplError(input, "TestComponent")
			require.Len(t, errors, 1)
			
			// Should not panic and should safely contain the input
			err := errors[0]
			assert.Equal(t, string(input), err.Message)
			assert.Equal(t, "TestComponent", err.Component)
		})
	}
}

func TestBuildErrorFields(t *testing.T) {
	now := time.Now()
	err := BuildError{
		Component: "TestComponent",
		File:      "test.templ",
		Line:      42,
		Column:    15,
		Message:   "unexpected token",
		Severity:  ErrorSeverityFatal,
		Timestamp: now,
	}
	
	assert.Equal(t, "TestComponent", err.Component)
	assert.Equal(t, "test.templ", err.File)
	assert.Equal(t, 42, err.Line)
	assert.Equal(t, 15, err.Column)
	assert.Equal(t, "unexpected token", err.Message)
	assert.Equal(t, ErrorSeverityFatal, err.Severity)
	assert.Equal(t, now, err.Timestamp)
}

func TestErrorCollectorConcurrency(t *testing.T) {
	collector := NewErrorCollector()
	
	// Test concurrent access to collector
	// This is a basic test - in practice, you'd want to test with go race detector
	done := make(chan bool, 10)
	
	// Add errors concurrently
	for i := 0; i < 10; i++ {
		go func(i int) {
			err := BuildError{
				Component: fmt.Sprintf("Component%d", i),
				Message:   fmt.Sprintf("Error %d", i),
				Severity:  ErrorSeverityError,
			}
			collector.Add(err)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have all 10 errors
	assert.Equal(t, 10, len(collector.GetErrors()))
	assert.True(t, collector.HasErrors())
}