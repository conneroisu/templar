package errors

import (
	"context"
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeSecurity   ErrorType = "security"
	ErrorTypeIO         ErrorType = "io"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeBuild      ErrorType = "build"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeInternal   ErrorType = "internal"
)

// TemplarError is a structured error type with context
type TemplarError struct {
	Type        ErrorType
	Code        string
	Message     string
	Cause       error
	Context     map[string]interface{}
	Component   string
	FilePath    string
	Line        int
	Column      int
	Recoverable bool
}

// Error implements the error interface
func (e *TemplarError) Error() string {
	var parts []string

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}

	if e.Component != "" {
		parts = append(parts, fmt.Sprintf("component:%s", e.Component))
	}

	if e.FilePath != "" {
		location := e.FilePath
		if e.Line > 0 {
			location += fmt.Sprintf(":%d", e.Line)
			if e.Column > 0 {
				location += fmt.Sprintf(":%d", e.Column)
			}
		}
		parts = append(parts, location)
	}

	parts = append(parts, e.Message)

	result := strings.Join(parts, " ")

	if e.Cause != nil {
		result += fmt.Sprintf(": %v", e.Cause)
	}

	return result
}

// Unwrap returns the underlying cause error
func (e *TemplarError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *TemplarError) Is(target error) bool {
	if t, ok := target.(*TemplarError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// WithContext adds context information to the error
func (e *TemplarError) WithContext(key string, value interface{}) *TemplarError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithLocation adds file location information
func (e *TemplarError) WithLocation(filePath string, line, column int) *TemplarError {
	e.FilePath = filePath
	e.Line = line
	e.Column = column
	return e
}

// WithComponent adds component context
func (e *TemplarError) WithComponent(component string) *TemplarError {
	e.Component = component
	return e
}

// Error creation functions

// NewValidationError creates a validation error
func NewValidationError(code, message string) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeValidation,
		Code:        code,
		Message:     message,
		Recoverable: true,
	}
}

// NewSecurityError creates a security error
func NewSecurityError(code, message string) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeSecurity,
		Code:        code,
		Message:     message,
		Recoverable: false,
	}
}

// NewBuildError creates a build error
func NewBuildError(code, message string, cause error) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeBuild,
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: true,
	}
}

// NewIOError creates an I/O error
func NewIOError(code, message string, cause error) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeIO,
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: false,
	}
}

// NewConfigError creates a configuration error
func NewConfigError(code, message string) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeConfig,
		Code:        code,
		Message:     message,
		Recoverable: false,
	}
}

// NewInternalError creates an internal error
func NewInternalError(code, message string, cause error) *TemplarError {
	return &TemplarError{
		Type:        ErrorTypeInternal,
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: false,
	}
}

// Error recovery and handling utilities

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	if te, ok := err.(*TemplarError); ok {
		return te.Recoverable
	}
	return false
}

// IsSecurityError checks if an error is security-related
func IsSecurityError(err error) bool {
	if te, ok := err.(*TemplarError); ok {
		return te.Type == ErrorTypeSecurity
	}
	return false
}

// IsBuildError checks if an error is build-related
func IsBuildError(err error) bool {
	if te, ok := err.(*TemplarError); ok {
		return te.Type == ErrorTypeBuild
	}
	return false
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger   Logger
	notifier Notifier
}

// Logger interface for error logging
type Logger interface {
	Error(ctx context.Context, err error, msg string, fields ...interface{})
	Warn(ctx context.Context, err error, msg string, fields ...interface{})
}

// Notifier interface for error notifications
type Notifier interface {
	NotifyError(ctx context.Context, err *TemplarError) error
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger, notifier Notifier) *ErrorHandler {
	return &ErrorHandler{
		logger:   logger,
		notifier: notifier,
	}
}

// Handle processes an error with appropriate logging and notifications
func (h *ErrorHandler) Handle(ctx context.Context, err error) {
	if err == nil {
		return
	}

	if te, ok := err.(*TemplarError); ok {
		h.handleTemplarError(ctx, te)
	} else {
		h.handleGenericError(ctx, err)
	}
}

func (h *ErrorHandler) handleTemplarError(ctx context.Context, err *TemplarError) {
	switch err.Type {
	case ErrorTypeSecurity:
		if h.logger != nil {
			h.logger.Error(ctx, err, "Security error occurred",
				"type", err.Type,
				"code", err.Code,
				"component", err.Component)
		}
		if h.notifier != nil {
			h.notifier.NotifyError(ctx, err)
		}
	case ErrorTypeBuild:
		if h.logger != nil {
			h.logger.Warn(ctx, err, "Build error occurred",
				"type", err.Type,
				"code", err.Code,
				"component", err.Component,
				"file", err.FilePath)
		}
	case ErrorTypeValidation:
		if h.logger != nil {
			h.logger.Warn(ctx, err, "Validation error occurred",
				"type", err.Type,
				"code", err.Code,
				"component", err.Component)
		}
	default:
		if h.logger != nil {
			h.logger.Error(ctx, err, "Error occurred",
				"type", err.Type,
				"code", err.Code,
				"component", err.Component)
		}
	}
}

func (h *ErrorHandler) handleGenericError(ctx context.Context, err error) {
	if h.logger != nil {
		h.logger.Error(ctx, err, "Unhandled error occurred")
	}
}

// Common error codes
const (
	ErrCodeInvalidPath       = "ERR_INVALID_PATH"
	ErrCodePathTraversal     = "ERR_PATH_TRAVERSAL"
	ErrCodeCommandInjection  = "ERR_COMMAND_INJECTION"
	ErrCodeInvalidOrigin     = "ERR_INVALID_ORIGIN"
	ErrCodeComponentNotFound = "ERR_COMPONENT_NOT_FOUND"
	ErrCodeBuildFailed       = "ERR_BUILD_FAILED"
	ErrCodeConfigInvalid     = "ERR_CONFIG_INVALID"
	ErrCodeFileNotFound      = "ERR_FILE_NOT_FOUND"
	ErrCodePermissionDenied  = "ERR_PERMISSION_DENIED"
	ErrCodeInternalError     = "ERR_INTERNAL"
)

// Helper functions for common errors

// ErrInvalidPath creates a path validation error
func ErrInvalidPath(path string) *TemplarError {
	return NewValidationError(ErrCodeInvalidPath, fmt.Sprintf("invalid path: %s", path))
}

// ErrPathTraversal creates a path traversal security error
func ErrPathTraversal(path string) *TemplarError {
	return NewSecurityError(ErrCodePathTraversal, fmt.Sprintf("path traversal attempt: %s", path))
}

// ErrCommandInjection creates a command injection security error
func ErrCommandInjection(command string) *TemplarError {
	return NewSecurityError(ErrCodeCommandInjection, fmt.Sprintf("command injection attempt: %s", command))
}

// ErrInvalidOrigin creates an invalid origin security error
func ErrInvalidOrigin(origin string) *TemplarError {
	return NewSecurityError(ErrCodeInvalidOrigin, fmt.Sprintf("invalid origin: %s", origin))
}

// ErrComponentNotFound creates a component not found error
func ErrComponentNotFound(name string) *TemplarError {
	return NewValidationError(ErrCodeComponentNotFound, fmt.Sprintf("component not found: %s", name))
}

// ErrBuildFailed creates a build failure error
func ErrBuildFailed(component string, cause error) *TemplarError {
	return NewBuildError(ErrCodeBuildFailed, fmt.Sprintf("build failed for component: %s", component), cause)
}
