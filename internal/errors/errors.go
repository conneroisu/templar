package errors

import (
	"fmt"
	"sync"
	"time"
)

// BuildError represents a build error
type BuildError struct {
	Component string
	File      string
	Line      int
	Column    int
	Message   string
	Severity  ErrorSeverity
	Timestamp time.Time
}

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	ErrorSeverityInfo ErrorSeverity = iota
	ErrorSeverityWarning
	ErrorSeverityError
	ErrorSeverityFatal
)

// String returns the string representation of the severity
func (s ErrorSeverity) String() string {
	switch s {
	case ErrorSeverityInfo:
		return "info"
	case ErrorSeverityWarning:
		return "warning"
	case ErrorSeverityError:
		return "error"
	case ErrorSeverityFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Error implements the error interface
func (be *BuildError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", be.File, be.Line, be.Column, be.Severity, be.Message)
}

// ErrorCollector collects and manages build errors and general errors
type ErrorCollector struct {
	buildErrors []BuildError
	errors      []error
	mutex       sync.RWMutex
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		buildErrors: make([]BuildError, 0),
		errors:      make([]error, 0),
	}
}

// Add adds a build error to the collector
func (ec *ErrorCollector) Add(err BuildError) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	err.Timestamp = time.Now()
	ec.buildErrors = append(ec.buildErrors, err)
}

// AddError adds a general error to the collector
func (ec *ErrorCollector) AddError(err error) {
	if err == nil {
		return
	}
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.errors = append(ec.errors, err)
}

// GetErrors returns all collected build errors
func (ec *ErrorCollector) GetErrors() []BuildError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	// Return a copy to avoid race conditions
	result := make([]BuildError, len(ec.buildErrors))
	copy(result, ec.buildErrors)
	return result
}

// GetAllErrors returns all collected errors (build and general)
func (ec *ErrorCollector) GetAllErrors() []error {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	allErrors := make([]error, 0, len(ec.buildErrors)+len(ec.errors))

	// Convert build errors to general errors
	for _, buildErr := range ec.buildErrors {
		allErrors = append(allErrors, &buildErr)
	}

	// Add general errors
	allErrors = append(allErrors, ec.errors...)

	return allErrors
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	return len(ec.buildErrors) > 0 || len(ec.errors) > 0
}

// Clear clears all errors
func (ec *ErrorCollector) Clear() {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.buildErrors = ec.buildErrors[:0]
	ec.errors = ec.errors[:0]
}

// GetErrorsByFile returns errors for a specific file
func (ec *ErrorCollector) GetErrorsByFile(file string) []BuildError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	var fileErrors []BuildError
	for _, err := range ec.buildErrors {
		if err.File == file {
			fileErrors = append(fileErrors, err)
		}
	}
	return fileErrors
}

// GetErrorsByComponent returns errors for a specific component
func (ec *ErrorCollector) GetErrorsByComponent(component string) []BuildError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	var componentErrors []BuildError
	for _, err := range ec.buildErrors {
		if err.Component == component {
			componentErrors = append(componentErrors, err)
		}
	}
	return componentErrors
}

// ErrorOverlay generates HTML for error overlay
func (ec *ErrorCollector) ErrorOverlay() string {
	if !ec.HasErrors() {
		return ""
	}

	html := `
<div id="templar-error-overlay" style="
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background: rgba(0, 0, 0, 0.8);
	color: white;
	font-family: 'Monaco', 'Menlo', monospace;
	font-size: 14px;
	z-index: 9999;
	padding: 20px;
	box-sizing: border-box;
	overflow: auto;
">
	<div style="max-width: 1000px; margin: 0 auto;">
		<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
			<h2 style="margin: 0; color: #ff6b6b;">Build Errors</h2>
			<button onclick="document.getElementById('templar-error-overlay').style.display='none'" 
					style="background: none; border: 1px solid #ccc; color: white; padding: 5px 10px; cursor: pointer;">
				Close
			</button>
		</div>
		<div>`

	ec.mutex.RLock()
	for _, err := range ec.buildErrors {
		severityColor := "#ff6b6b"
		switch err.Severity {
		case ErrorSeverityWarning:
			severityColor = "#feca57"
		case ErrorSeverityInfo:
			severityColor = "#48dbfb"
		}

		html += fmt.Sprintf(`
			<div style="
				background: #2d3748;
				padding: 15px;
				margin-bottom: 15px;
				border-radius: 4px;
				border-left: 4px solid %s;
			">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px;">
					<span style="color: %s; font-weight: bold;">%s</span>
					<span style="color: #a0aec0; font-size: 12px;">%s</span>
				</div>
				<div style="color: #e2e8f0; margin-bottom: 5px;">
					<strong>%s</strong>
				</div>
				<div style="color: #a0aec0; font-size: 12px;">
					%s:%d:%d
				</div>
			</div>
		`, severityColor, severityColor, err.Severity.String(), err.Timestamp.Format("15:04:05"), err.Message, err.File, err.Line, err.Column)
	}

	ec.mutex.RUnlock()

	html += `
		</div>
	</div>
</div>`

	return html
}

// ParseTemplError parses templ compiler error output
func ParseTemplError(output []byte, component string) []BuildError {
	var errors []BuildError

	// Simple error parsing - in a real implementation, this would be more sophisticated
	lines := string(output)
	if lines == "" {
		return errors
	}

	// Basic error parsing for demonstration
	// Real implementation would parse actual templ error format
	err := BuildError{
		Component: component,
		File:      "unknown",
		Line:      0,
		Column:    0,
		Message:   string(output),
		Severity:  ErrorSeverityError,
		Timestamp: time.Now(),
	}

	errors = append(errors, err)
	return errors
}
