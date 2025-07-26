// Package errors provides tests for error handling patterns.
package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceError(t *testing.T) {
	err := ServiceError("BUILD", "COMPILE", "compilation failed", fmt.Errorf("original error"))

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeInternal, err.Type)
	assert.Equal(t, "ERR_BUILD_COMPILE", err.Code)
	assert.Contains(t, err.Message, "BUILD service COMPILE failed")
	assert.Contains(t, err.Message, "compilation failed")
	assert.Equal(t, "BUILD", err.Component)
	assert.NotNil(t, err.Cause)
}

func TestInitError(t *testing.T) {
	originalErr := fmt.Errorf("directory not found")
	err := InitError("VALIDATE_DIR", "project directory validation failed", originalErr)

	assert.NotNil(t, err)
	assert.Equal(t, "ERR_INIT_VALIDATE_DIR", err.Code)
	assert.Contains(t, err.Message, "INIT service")
	assert.Equal(t, "INIT", err.Component)
	assert.Equal(t, originalErr, err.Cause)
}

func TestBuildServiceError(t *testing.T) {
	err := BuildServiceError("SCAN_COMPONENTS", "component scanning failed", nil)

	assert.NotNil(t, err)
	assert.Equal(t, "ERR_BUILD_SCAN_COMPONENTS", err.Code)
	assert.Contains(t, err.Message, "BUILD service")
	assert.Equal(t, "BUILD", err.Component)
}

func TestServeServiceError(t *testing.T) {
	err := ServeServiceError("START_SERVER", "server startup failed", fmt.Errorf("port in use"))

	assert.NotNil(t, err)
	assert.Equal(t, "ERR_SERVE_START_SERVER", err.Code)
	assert.Contains(t, err.Message, "SERVE service")
	assert.Equal(t, "SERVE", err.Component)
	assert.NotNil(t, err.Cause)
}

func TestDataError(t *testing.T) {
	err := DataError("READ", "user.json", "file access denied", fmt.Errorf("permission denied"))

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeIO, err.Type)
	assert.Equal(t, "ERR_DATA_READ", err.Code)
	assert.Contains(t, err.Message, "data READ failed for user.json")
	assert.NotNil(t, err.Cause)
}

func TestFileOperationError(t *testing.T) {
	err := FileOperationError("WRITE", "/tmp/test.txt", "disk full", fmt.Errorf("no space left"))

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeIO, err.Type)
	assert.Equal(t, "ERR_DATA_WRITE", err.Code)
	assert.Contains(t, err.Message, "file:/tmp/test.txt")
	assert.Equal(t, "/tmp/test.txt", err.Context["file_path"])
}

func TestConfigurationError(t *testing.T) {
	err := ConfigurationError("database.host", "invalid hostname format", "192.168.1")

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeConfig, err.Type)
	assert.Equal(t, "ERR_CONFIG_INVALID", err.Code)
	assert.Contains(t, err.Message, "database.host")
	assert.Equal(t, "database.host", err.Context["setting"])
	assert.Equal(t, "192.168.1", err.Context["value"])
}

func TestNetworkError(t *testing.T) {
	err := NetworkError(
		"CONNECT",
		"api.example.com:443",
		"connection timeout",
		fmt.Errorf("timeout"),
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "ERR_NETWORK_CONNECT", err.Code)
	assert.Contains(t, err.Message, "api.example.com:443")
	assert.True(t, err.Recoverable) // Network errors are recoverable
	assert.Equal(t, "api.example.com:443", err.Context["endpoint"])
}

func TestWebSocketError(t *testing.T) {
	err := WebSocketError(
		"SEND_MESSAGE",
		"client-123",
		"client disconnected",
		fmt.Errorf("broken pipe"),
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "ERR_NETWORK_WEBSOCKET_SEND_MESSAGE", err.Code)
	assert.Contains(t, err.Message, "client-123")
	assert.Equal(t, "client-123", err.Context["client_id"])
}

func TestServerError(t *testing.T) {
	err := ServerError("BIND_PORT", "failed to bind to port 8080", fmt.Errorf("address in use"))

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "ERR_NETWORK_SERVER_BIND_PORT", err.Code)
	assert.Contains(t, err.Message, "localhost")
}

func TestComponentError(t *testing.T) {
	err := ComponentError(
		"PARSE",
		"Button",
		"/components/button.templ",
		"syntax error",
		fmt.Errorf("unexpected token"),
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeBuild, err.Type)
	assert.Equal(t, "ERR_COMPONENT_PARSE", err.Code)
	assert.Contains(t, err.Message, "Button PARSE failed")
	assert.Equal(t, "Button", err.Component)
	assert.Equal(t, "/components/button.templ", err.FilePath)
	assert.Equal(t, "Button", err.Context["component"])
	assert.Equal(t, "PARSE", err.Context["operation"])
	assert.True(t, err.Recoverable)
}

func TestScannerError(t *testing.T) {
	err := ScannerError(
		"DIRECTORY",
		"/components",
		"access denied",
		fmt.Errorf("permission denied"),
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeBuild, err.Type)
	assert.Equal(t, "ERR_COMPONENT_SCAN_DIRECTORY", err.Code)
	assert.Equal(t, "scanner", err.Component)
	assert.Equal(t, "/components", err.FilePath)
}

func TestRegistryError(t *testing.T) {
	err := RegistryError("REGISTER", "DuplicateComponent", "component already exists", nil)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeBuild, err.Type)
	assert.Equal(t, "ERR_COMPONENT_REGISTRY_REGISTER", err.Code)
	assert.Equal(t, "DuplicateComponent", err.Component)
}

func TestCLIError(t *testing.T) {
	err := CLIError("SERVE", "invalid port number", fmt.Errorf("not a number"))

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "ERR_CLI_SERVE", err.Code)
	assert.Contains(t, err.Message, "command 'SERVE' failed")
	assert.Equal(t, "cli", err.Component)
	assert.True(t, err.Recoverable)
}

func TestFlagError(t *testing.T) {
	err := FlagError("port", "must be between 1024-65535", 99)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "ERR_FIELD_PORT", err.Code)
	assert.Contains(t, err.Message, "must be between 1024-65535")
	assert.Equal(t, "port", err.Context["field"])
	assert.Equal(t, 99, err.Context["value"])
	assert.True(t, err.Recoverable)
}

func TestArgumentError(t *testing.T) {
	err := ArgumentError(
		"project_name",
		"too many arguments provided",
		[]string{"arg1", "arg2", "arg3"},
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "ERR_FIELD_PROJECT_NAME", err.Code)
	assert.Contains(t, err.Message, "too many arguments provided")
	assert.Equal(t, "project_name", err.Context["field"])
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, err.Context["value"])
	assert.True(t, err.Recoverable)
}

func TestSecurityViolation(t *testing.T) {
	context := map[string]interface{}{
		"attempted_path": "../../../etc/passwd",
		"source_ip":      "192.168.1.100",
	}
	err := SecurityViolation("PATH_TRAVERSAL", "attempted directory traversal", context)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeSecurity, err.Type)
	assert.Equal(t, "ERR_SECURITY_PATH_TRAVERSAL", err.Code)
	assert.Contains(t, err.Message, "directory traversal")
	assert.False(t, err.Recoverable) // Security errors are not recoverable
	assert.Equal(t, "../../../etc/passwd", err.Context["attempted_path"])
	assert.Equal(t, "192.168.1.100", err.Context["source_ip"])
}

func TestValidationFailure(t *testing.T) {
	err := ValidationFailure(
		"email",
		"invalid email format",
		"not-an-email",
		"Use format: user@domain.com",
	)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Contains(t, err.Code, "ERR_FIELD_EMAIL")
	assert.Contains(t, err.Message, "invalid email format")
	assert.Equal(t, "email", err.Context["field"])
	assert.Equal(t, "not-an-email", err.Context["value"])
}

func TestPathValidationError(t *testing.T) {
	t.Run("path_traversal", func(t *testing.T) {
		err := PathValidationError("../../../etc/passwd", "traversal")

		assert.NotNil(t, err)
		assert.Equal(t, ErrorTypeSecurity, err.Type)
		assert.Equal(t, ErrCodePathTraversal, err.Code)
		assert.Contains(t, err.Message, "../../../etc/passwd")
	})

	t.Run("invalid_path", func(t *testing.T) {
		err := PathValidationError("/invalid\x00path", "null_byte")

		assert.NotNil(t, err)
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeInvalidPath, err.Code)
		assert.Equal(t, "null_byte", err.Context["reason"])
	})
}

func TestWithLocationInfo(t *testing.T) {
	originalErr := fmt.Errorf("syntax error")
	enhancedErr := WithLocationInfo(originalErr, "/components/button.templ", 42, 10)

	var templErr *TemplarError
	ok := errors.As(enhancedErr, &templErr)
	assert.True(t, ok)
	assert.Equal(t, "/components/button.templ", templErr.FilePath)
	assert.Equal(t, 42, templErr.Line)
	assert.Equal(t, 10, templErr.Column)
}

func TestWithComponentInfo(t *testing.T) {
	originalErr := fmt.Errorf("compilation failed")
	enhancedErr := WithComponentInfo(originalErr, "HeaderComponent")

	var templErr *TemplarError
	ok := errors.As(enhancedErr, &templErr)
	assert.True(t, ok)
	assert.Equal(t, "HeaderComponent", templErr.Component)
}

func TestWithOperationContext(t *testing.T) {
	originalErr := fmt.Errorf("database error")
	context := map[string]interface{}{
		"query":    "SELECT * FROM users",
		"duration": "1.5s",
	}
	enhancedErr := WithOperationContext(originalErr, "DATABASE_QUERY", context)

	var templErr *TemplarError
	ok := errors.As(enhancedErr, &templErr)
	assert.True(t, ok)
	assert.Equal(t, "DATABASE_QUERY", templErr.Context["operation"])
	assert.Equal(t, "SELECT * FROM users", templErr.Context["query"])
	assert.Equal(t, "1.5s", templErr.Context["duration"])
}

func TestWithOperationContext_NilError(t *testing.T) {
	result := WithOperationContext(nil, "OPERATION", map[string]interface{}{})
	assert.Nil(t, result)
}

func TestGetRootCause(t *testing.T) {
	rootErr := fmt.Errorf("connection refused")
	wrappedErr := fmt.Errorf("database error: %w", rootErr)
	templErr := WrapBuild(wrappedErr, "ERR_DB_CONNECT", "database connection failed", "database")

	root := GetRootCause(templErr)
	// ExtractCause returns the deepest non-TemplarError, which is wrappedErr
	assert.Equal(t, wrappedErr, root)
}

func TestGetErrorChain(t *testing.T) {
	rootErr := fmt.Errorf("connection refused")
	wrappedErr := fmt.Errorf("database error: %w", rootErr)
	templErr := WrapBuild(wrappedErr, "ERR_DB_CONNECT", "database connection failed", "database")

	chain := GetErrorChain(templErr)
	assert.Len(t, chain, 3) // TemplarError, wrapped error, and root error
	assert.Equal(t, templErr, chain[0])
	// chain[1] would be wrappedErr, chain[2] would be rootErr
}

func TestHasErrorCode(t *testing.T) {
	err := NewBuildError("ERR_COMPILE", "compilation failed", fmt.Errorf("syntax error"))

	assert.True(t, HasErrorCode(err, "ERR_COMPILE"))
	assert.False(t, HasErrorCode(err, "ERR_NETWORK"))
}

func TestHasErrorType(t *testing.T) {
	err := NewSecurityError("ERR_INJECTION", "SQL injection attempt")

	assert.True(t, HasErrorType(err, ErrorTypeSecurity))
	assert.False(t, HasErrorType(err, ErrorTypeValidation))
}

func TestErrorChainingConsistency(t *testing.T) {
	// Test that all error creation functions create properly structured errors
	testCases := []struct {
		name string
		err  *TemplarError
	}{
		{"ServiceError", ServiceError("TEST", "OP", "message", fmt.Errorf("cause"))},
		{"InitError", InitError("OP", "message", fmt.Errorf("cause"))},
		{"BuildServiceError", BuildServiceError("OP", "message", fmt.Errorf("cause"))},
		{"ServeServiceError", ServeServiceError("OP", "message", fmt.Errorf("cause"))},
		{"NetworkError", NetworkError("OP", "endpoint", "message", fmt.Errorf("cause"))},
		{"ComponentError", ComponentError("OP", "comp", "path", "message", fmt.Errorf("cause"))},
		{"CLIError", CLIError("CMD", "message", fmt.Errorf("cause"))},
		{"SecurityViolation", SecurityViolation("OP", "detail", map[string]interface{}{})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, tc.err)
			assert.NotEmpty(t, tc.err.Type)
			assert.NotEmpty(t, tc.err.Code)
			assert.NotEmpty(t, tc.err.Message)

			// Test Error() method produces readable output
			errStr := tc.err.Error()
			assert.NotEmpty(t, errStr)
		})
	}
}

func TestErrorPatternConsistency(t *testing.T) {
	// Verify that similar operations produce consistent error patterns
	buildErr := BuildServiceError("COMPILE", "compilation failed", nil)
	serveErr := ServeServiceError("START", "startup failed", nil)

	// Both should have consistent structure
	assert.Equal(t, ErrorTypeInternal, buildErr.Type)
	assert.Equal(t, ErrorTypeInternal, serveErr.Type)

	assert.Contains(t, buildErr.Code, "ERR_BUILD_")
	assert.Contains(t, serveErr.Code, "ERR_SERVE_")

	assert.Equal(t, "BUILD", buildErr.Component)
	assert.Equal(t, "SERVE", serveErr.Component)
}

// Benchmark tests for error creation performance
func BenchmarkServiceError(b *testing.B) {
	cause := fmt.Errorf("original error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ServiceError("BUILD", "COMPILE", "compilation failed", cause)
	}
}

func BenchmarkSecurityViolation(b *testing.B) {
	context := map[string]interface{}{
		"ip":   "192.168.1.1",
		"path": "/admin/secret",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SecurityViolation("UNAUTHORIZED_ACCESS", "unauthorized access attempt", context)
	}
}

func BenchmarkErrorChainTraversal(b *testing.B) {
	rootErr := fmt.Errorf("connection refused")
	wrappedErr := fmt.Errorf("database error: %w", rootErr)
	templErr := WrapBuild(wrappedErr, "ERR_DB_CONNECT", "database connection failed", "database")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetErrorChain(templErr)
	}
}
