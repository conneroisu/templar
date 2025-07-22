package errors

import (
	"fmt"
)

// ErrorPatterns defines standardized error creation patterns for consistent usage across the codebase.
// This file provides convenience functions and guidelines for creating properly structured errors.

// Pattern Guidelines:
// 1. Always use TemplarError for structured errors with context
// 2. Use appropriate error types (Validation, Security, IO, Network, Build, Config, Internal)
// 3. Include meaningful error codes for programmatic handling
// 4. Add component and file context when available
// 5. Wrap existing errors to preserve the error chain
// 6. Use consistent error messages and formatting

// Service Layer Error Patterns

// ServiceError creates a standardized service error with component context
func ServiceError(service, operation, message string, cause error) *TemplarError {
	code := fmt.Sprintf("ERR_%s_%s", service, operation)
	return WrapInternal(cause, code, fmt.Sprintf("%s service %s failed: %s", service, operation, message)).
		WithComponent(service)
}

// InitError creates initialization-related errors
func InitError(operation, message string, cause error) *TemplarError {
	return ServiceError("INIT", operation, message, cause)
}

// BuildServiceError creates build service errors
func BuildServiceError(operation, message string, cause error) *TemplarError {
	return ServiceError("BUILD", operation, message, cause)
}

// ServeServiceError creates serve service errors  
func ServeServiceError(operation, message string, cause error) *TemplarError {
	return ServiceError("SERVE", operation, message, cause)
}

// Repository/Data Layer Error Patterns

// DataError creates data layer errors with consistent formatting
func DataError(operation, resource, message string, cause error) *TemplarError {
	code := fmt.Sprintf("ERR_DATA_%s", operation)
	return WrapIO(cause, code, fmt.Sprintf("data %s failed for %s: %s", operation, resource, message))
}

// FileOperationError creates file operation errors
func FileOperationError(operation, filePath, message string, cause error) *TemplarError {
	return DataError(operation, fmt.Sprintf("file:%s", filePath), message, cause).
		WithContext("file_path", filePath)
}

// ConfigurationError creates configuration-related errors
func ConfigurationError(setting, message string, value interface{}) *TemplarError {
	return NewConfigError(
		"ERR_CONFIG_INVALID",
		fmt.Sprintf("invalid configuration for %s: %s", setting, message),
	).WithContext("setting", setting).WithContext("value", value)
}

// Network and Communication Error Patterns

// NetworkError creates network-related errors
func NetworkError(operation, endpoint, message string, cause error) *TemplarError {
	code := fmt.Sprintf("ERR_NETWORK_%s", operation)
	return &TemplarError{
		Type:        ErrorTypeNetwork,
		Code:        code,
		Message:     fmt.Sprintf("network %s failed for %s: %s", operation, endpoint, message),
		Cause:       cause,
		Context:     map[string]interface{}{"endpoint": endpoint},
		Recoverable: true, // Network errors are often temporary
	}
}

// WebSocketError creates WebSocket-related errors
func WebSocketError(operation, clientID, message string, cause error) *TemplarError {
	return NetworkError("WEBSOCKET_"+operation, clientID, message, cause).
		WithContext("client_id", clientID)
}

// ServerError creates server operation errors
func ServerError(operation, message string, cause error) *TemplarError {
	return NetworkError("SERVER_"+operation, "localhost", message, cause)
}

// Component and Build Error Patterns

// ComponentError creates component-related errors with full context
func ComponentError(operation, componentName, filePath, message string, cause error) *TemplarError {
	code := fmt.Sprintf("ERR_COMPONENT_%s", operation)
	return &TemplarError{
		Type:        ErrorTypeBuild,
		Code:        code,
		Message:     fmt.Sprintf("component %s %s failed: %s", componentName, operation, message),
		Cause:       cause,
		Component:   componentName,
		FilePath:    filePath,
		Context:     map[string]interface{}{"component": componentName, "operation": operation},
		Recoverable: true,
	}
}

// ScannerError creates scanner-related errors
func ScannerError(operation, path, message string, cause error) *TemplarError {
	return ComponentError("SCAN_"+operation, "scanner", path, message, cause)
}

// RegistryError creates registry operation errors
func RegistryError(operation, componentName, message string, cause error) *TemplarError {
	return ComponentError("REGISTRY_"+operation, componentName, "", message, cause)
}

// CLI and User Interface Error Patterns

// CLIError creates CLI command errors with user-friendly messages
func CLIError(command, message string, cause error) *TemplarError {
	code := fmt.Sprintf("ERR_CLI_%s", command)
	return &TemplarError{
		Type:        ErrorTypeValidation,
		Code:        code,
		Message:     fmt.Sprintf("command '%s' failed: %s", command, message),
		Cause:       cause,
		Component:   "cli",
		Recoverable: true,
	}
}

// FlagError creates CLI flag validation errors
func FlagError(flagName, message string, value interface{}) *TemplarError {
	return NewFieldValidationError(
		flagName,
		value,
		message,
		fmt.Sprintf("Check 'templar %s --help' for valid options", flagName),
	).ToTemplarError()
}

// ArgumentError creates CLI argument validation errors
func ArgumentError(argName, message string, value interface{}) *TemplarError {
	return NewFieldValidationError(
		argName,
		value,
		message,
		"Use 'templar --help' to see command usage",
	).ToTemplarError()
}

// Security and Validation Error Patterns

// SecurityViolation creates security violation errors (non-recoverable)
func SecurityViolation(operation, detail string, context map[string]interface{}) *TemplarError {
	code := fmt.Sprintf("ERR_SECURITY_%s", operation)
	return &TemplarError{
		Type:        ErrorTypeSecurity,
		Code:        code,
		Message:     fmt.Sprintf("security violation in %s: %s", operation, detail),
		Context:     context,
		Recoverable: false,
	}
}

// ValidationFailure creates validation errors with suggestions
func ValidationFailure(field, message string, value interface{}, suggestions ...string) *TemplarError {
	fieldErr := NewFieldValidationError(field, value, message, suggestions...)
	return fieldErr.ToTemplarError()
}

// PathValidationError creates path validation errors with security context
func PathValidationError(path, reason string) *TemplarError {
	if reason == "traversal" {
		return ErrPathTraversal(path)
	}
	return ErrInvalidPath(path).WithContext("reason", reason)
}

// Utility Functions for Error Enhancement

// WithLocationInfo adds file location information to any error
func WithLocationInfo(err error, filePath string, line, column int) error {
	return EnhanceError(err, "", filePath, line, column)
}

// WithComponentInfo adds component context to any error  
func WithComponentInfo(err error, componentName string) error {
	return EnhanceError(err, componentName, "", 0, 0)
}

// WithOperationContext adds operation context to any error
func WithOperationContext(err error, operation string, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	
	if te, ok := err.(*TemplarError); ok {
		if te.Context == nil {
			te.Context = make(map[string]interface{})
		}
		te.Context["operation"] = operation
		for k, v := range context {
			te.Context[k] = v
		}
		return te
	}
	
	// Wrap non-TemplarError with context
	return WrapWithContext(err, ErrorTypeInternal, ErrCodeInternalError, err.Error(), map[string]interface{}{
		"operation": operation,
	})
}

// Error Chain Utilities

// GetRootCause returns the deepest underlying error in the chain
func GetRootCause(err error) error {
	return ExtractCause(err)
}

// GetErrorChain returns all errors in the chain from outermost to innermost
func GetErrorChain(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		if te, ok := err.(*TemplarError); ok {
			err = te.Cause
		} else if wrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = wrapper.Unwrap()
		} else {
			break
		}
	}
	return chain
}

// HasErrorCode checks if any error in the chain has the specified code
func HasErrorCode(err error, code string) bool {
	chain := GetErrorChain(err)
	for _, e := range chain {
		if te, ok := e.(*TemplarError); ok && te.Code == code {
			return true
		}
	}
	return false
}

// HasErrorType checks if any error in the chain has the specified type
func HasErrorType(err error, errType ErrorType) bool {
	chain := GetErrorChain(err)
	for _, e := range chain {
		if te, ok := e.(*TemplarError); ok && te.Type == errType {
			return true
		}
	}
	return false
}

// Pattern Examples and Best Practices

// Example: Service Layer Error
// func (s *InitService) InitProject(opts InitOptions) error {
//     if err := s.validateProjectDirectory(opts.ProjectDir); err != nil {
//         return InitError("VALIDATE_DIR", "project directory validation failed", err)
//     }
//     return nil
// }

// Example: CLI Command Error  
// func runInit(cmd *cobra.Command, args []string) error {
//     if len(args) > 1 {
//         return ArgumentError("project_name", "too many arguments provided", args)
//     }
//     return nil
// }

// Example: Component Error with Location
// func (s *Scanner) parseComponent(filePath string) error {
//     if parseErr := templ.Parse(filePath); parseErr != nil {
//         return ComponentError("PARSE", "Button", filePath, "syntax error", parseErr).
//             WithLocation(filePath, 42, 10)
//     }
//     return nil
// }