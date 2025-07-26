package errors

import (
	"errors"
	"fmt"
)

// Wrap wraps an error with additional context, creating a TemplarError if the input is not already one
func Wrap(err error, errType ErrorType, code, message string) *TemplarError {
	if err == nil {
		return nil
	}

	// If it's already a TemplarError, preserve its properties but update the message
	var te *TemplarError
	if errors.As(err, &te) {
		return &TemplarError{
			Type:        errType,
			Code:        code,
			Message:     message,
			Cause:       te,
			Context:     te.Context,
			Component:   te.Component,
			FilePath:    te.FilePath,
			Line:        te.Line,
			Column:      te.Column,
			Recoverable: te.Recoverable,
		}
	}

	return &TemplarError{
		Type:        errType,
		Code:        code,
		Message:     message,
		Cause:       err,
		Recoverable: errType == ErrorTypeValidation || errType == ErrorTypeBuild,
	}
}

// WrapWithContext wraps an error with context information
func WrapWithContext(err error, errType ErrorType, code, message string, context map[string]interface{}) *TemplarError {
	templErr := Wrap(err, errType, code, message)
	if templErr != nil {
		templErr.Context = context
	}
	return templErr
}

// WrapBuild wraps an error as a build error with component context
func WrapBuild(err error, code, message, component string) *TemplarError {
	templErr := Wrap(err, ErrorTypeBuild, code, message)
	if templErr != nil {
		templErr.Component = component
	}
	return templErr
}

// WrapValidation wraps an error as a validation error
func WrapValidation(err error, code, message string) *TemplarError {
	return Wrap(err, ErrorTypeValidation, code, message)
}

// WrapSecurity wraps an error as a security error (non-recoverable)
func WrapSecurity(err error, code, message string) *TemplarError {
	templErr := Wrap(err, ErrorTypeSecurity, code, message)
	if templErr != nil {
		templErr.Recoverable = false
	}
	return templErr
}

// WrapIO wraps an error as an I/O error
func WrapIO(err error, code, message string) *TemplarError {
	templErr := Wrap(err, ErrorTypeIO, code, message)
	if templErr != nil {
		templErr.Recoverable = false
	}
	return templErr
}

// WrapConfig wraps an error as a configuration error
func WrapConfig(err error, code, message string) *TemplarError {
	templErr := Wrap(err, ErrorTypeConfig, code, message)
	if templErr != nil {
		templErr.Recoverable = false
	}
	return templErr
}

// WrapInternal wraps an error as an internal error
func WrapInternal(err error, code, message string) *TemplarError {
	templErr := Wrap(err, ErrorTypeInternal, code, message)
	if templErr != nil {
		templErr.Recoverable = false
	}
	return templErr
}

// EnhanceError adds debugging context to an existing error
func EnhanceError(err error, component, filePath string, line, column int) error {
	if err == nil {
		return nil
	}

	var te *TemplarError
	if errors.As(err, &te) {
		return te.WithComponent(component).WithLocation(filePath, line, column)
	}

	// Create a new TemplarError for non-TemplarError types
	return &TemplarError{
		Type:        ErrorTypeInternal,
		Code:        ErrCodeInternalError,
		Message:     err.Error(),
		Cause:       err,
		Component:   component,
		FilePath:    filePath,
		Line:        line,
		Column:      column,
		Recoverable: false,
	}
}

// FormatError formats an error for user display
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var te *TemplarError
	if errors.As(err, &te) {
		return te.Error()
	}

	return err.Error()
}

// FormatErrorWithSuggestions formats an error with suggestions for ValidationError types
func FormatErrorWithSuggestions(err error) string {
	if err == nil {
		return ""
	}

	var ve ValidationError
	if errors.As(err, &ve) {
		result := ve.Error()
		suggestions := ve.Suggestions()
		if len(suggestions) > 0 {
			result += "\n\nSuggestions:"
			for _, suggestion := range suggestions {
				result += fmt.Sprintf("\n  • %s", suggestion)
			}
		}
		return result
	}

	return FormatError(err)
}

// GetErrorContext extracts context information from a TemplarError
func GetErrorContext(err error) map[string]interface{} {
	var te *TemplarError
	if errors.As(err, &te) {
		context := make(map[string]interface{})
		if te.Context != nil {
			for k, v := range te.Context {
				context[k] = v
			}
		}
		if te.Component != "" {
			context["component"] = te.Component
		}
		if te.FilePath != "" {
			context["file"] = te.FilePath
			if te.Line > 0 {
				context["line"] = te.Line
				if te.Column > 0 {
					context["column"] = te.Column
				}
			}
		}
		context["type"] = string(te.Type)
		context["code"] = te.Code
		context["recoverable"] = te.Recoverable
		return context
	}

	return map[string]interface{}{
		"message": err.Error(),
		"type":    "unknown",
	}
}

// IsTemporaryError checks if an error is temporary and should be retried
func IsTemporaryError(err error) bool {
	var te *TemplarError
	if errors.As(err, &te) {
		// Build and validation errors are typically temporary
		return te.Type == ErrorTypeBuild || te.Type == ErrorTypeValidation || te.Type == ErrorTypeNetwork
	}
	return false
}

// IsFatalError checks if an error is fatal and should stop execution
func IsFatalError(err error) bool {
	var te *TemplarError
	if errors.As(err, &te) {
		return te.Type == ErrorTypeSecurity || te.Type == ErrorTypeInternal
	}
	return false
}

// ExtractCause extracts the root cause from a wrapped error
func ExtractCause(err error) error {
	for err != nil {
		var te *TemplarError
	if errors.As(err, &te) {
			if te.Cause == nil {
				return te
			}
			err = te.Cause
		} else {
			return err
		}
	}
	return nil
}

// CollectErrors helper for common error collection patterns
func CollectErrors(errs ...error) []error {
	var collected []error
	for _, err := range errs {
		if err != nil {
			collected = append(collected, err)
		}
	}
	return collected
}

// FirstError returns the first non-nil error from a list
func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// CombineErrors combines multiple errors into a single error with context
func CombineErrors(errs ...error) error {
	nonNilErrs := CollectErrors(errs...)
	if len(nonNilErrs) == 0 {
		return nil
	}
	if len(nonNilErrs) == 1 {
		return nonNilErrs[0]
	}

	var messages []string
	for _, err := range nonNilErrs {
		messages = append(messages, err.Error())
	}

	return &TemplarError{
		Type:    ErrorTypeInternal,
		Code:    "ERR_MULTIPLE_ERRORS",
		Message: fmt.Sprintf("multiple errors occurred: %d errors", len(nonNilErrs)),
		Context: map[string]interface{}{
			"error_count": len(nonNilErrs),
			"errors":      messages,
		},
		Recoverable: false,
	}
}
