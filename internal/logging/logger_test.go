package logging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredError(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := NewStructuredError(ErrorCategoryValidation, "validate_input", "invalid parameter")

		assert.Equal(t, ErrorCategoryValidation, err.Category)
		assert.Equal(t, "validate_input", err.Operation)
		assert.Equal(t, "invalid parameter", err.Message)
		assert.Equal(t, "error", err.Severity)
		assert.False(t, err.Retryable)
		assert.NotNil(t, err.Context)
		assert.NotZero(t, err.Timestamp)
	})

	t.Run("error with cause", func(t *testing.T) {
		originalErr := errors.New("original error")
		structErr := NewStructuredError(ErrorCategorySystem, "operation", "failed").
			WithCause(originalErr)

		assert.Equal(t, originalErr, structErr.Cause)
		assert.Equal(t, originalErr, structErr.Unwrap())
		assert.Contains(t, structErr.Error(), "original error")
	})

	t.Run("error with context", func(t *testing.T) {
		err := NewStructuredError(ErrorCategoryFileSystem, "read_file", "permission denied").
			WithComponent("scanner").
			WithContext("file_path", "/test/path").
			WithContext("permissions", "0644").
			WithRetryable(true).
			WithSeverity("critical")

		assert.Equal(t, "scanner", err.Component)
		assert.Equal(t, "/test/path", err.Context["file_path"])
		assert.Equal(t, "0644", err.Context["permissions"])
		assert.True(t, err.Retryable)
		assert.Equal(t, "critical", err.Severity)
	})
}

func TestLogStructuredError(t *testing.T) {
	// Create a mock logger that captures log calls
	var capturedMessage string
	var capturedFields []interface{}

	mockLogger := &mockLogger{
		errorFunc: func(ctx context.Context, err error, msg string, fields ...interface{}) {
			capturedMessage = msg
			capturedFields = fields
		},
	}

	structErr := NewStructuredError(ErrorCategoryBuild, "compile_component", "syntax error").
		WithComponent("Button").
		WithContext("line", 42).
		WithRetryable(false)

	LogStructuredError(mockLogger, context.Background(), structErr)

	assert.Equal(t, "syntax error", capturedMessage)

	// Check that structured fields are included
	fieldsMap := fieldsToMap(capturedFields)
	assert.Equal(t, "build", fieldsMap["error_category"])
	assert.Equal(t, "compile_component", fieldsMap["operation"])
	assert.Equal(t, "Button", fieldsMap["component"])
	assert.Equal(t, 42, fieldsMap["line"])
	assert.Equal(t, false, fieldsMap["retryable"])
}

func TestResilientLogger(t *testing.T) {
	t.Run("successful logging on first attempt", func(t *testing.T) {
		mockLogger := &mockLogger{
			errorFunc: func(ctx context.Context, err error, msg string, fields ...interface{}) {
				// Success case - no panic
			},
		}

		resilientLogger := NewResilientLogger(mockLogger, 3, 10*time.Millisecond)
		resilientLogger.ErrorWithRetry(context.Background(), errors.New("test"), "test message")

		// Should succeed without retries
		assert.Equal(t, 1, mockLogger.errorCallCount)
	})

	t.Run("retry mechanism", func(t *testing.T) {
		attemptCount := 0
		mockLogger := &mockLogger{
			errorFunc: func(ctx context.Context, err error, msg string, fields ...interface{}) {
				attemptCount++
				if attemptCount < 3 {
					panic("simulated logging failure")
				}
				// Success on third attempt - no panic
			},
		}

		resilientLogger := NewResilientLogger(mockLogger, 2, 1*time.Millisecond)
		resilientLogger.ErrorWithRetry(context.Background(), errors.New("test"), "test message")

		// Should have tried 3 times before succeeding (initial attempt + 2 retries)
		assert.Equal(t, 3, attemptCount)
	})
}

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password field",
			input:    "user password: secret123",
			expected: "[REDACTED]",
		},
		{
			name:     "token field",
			input:    "auth token abc123",
			expected: "[REDACTED]",
		},
		{
			name:     "normal text",
			input:    "normal log message",
			expected: "normal log message",
		},
		{
			name:     "long text truncation",
			input:    string(make([]byte, 1500)),
			expected: string(make([]byte, 1000)) + "...[TRUNCATED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLog(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewFileLogger(t *testing.T) {
	t.Run("valid directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig()

		fileLogger, err := NewFileLogger(config, tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, fileLogger)

		err = fileLogger.Close()
		assert.NoError(t, err)
	})

	t.Run("invalid directory with path traversal", func(t *testing.T) {
		config := DefaultConfig()

		_, err := NewFileLogger(config, "../../../etc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("empty directory", func(t *testing.T) {
		config := DefaultConfig()

		_, err := NewFileLogger(config, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

// Mock logger for testing
type mockLogger struct {
	errorCallCount int
	errorFunc      func(ctx context.Context, err error, msg string, fields ...interface{})
}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...interface{})           {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...interface{})            {}
func (m *mockLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{}) {}
func (m *mockLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	m.errorCallCount++
	if m.errorFunc != nil {
		m.errorFunc(ctx, err, msg, fields...)
	}
}
func (m *mockLogger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {}

func (m *mockLogger) With(
	fields ...interface{},
) Logger {
	return m
}

func (m *mockLogger) WithComponent(
	component string,
) Logger {
	return m
}

// Helper function to convert fields slice to map
func fieldsToMap(fields []interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				result[key] = fields[i+1]
			}
		}
	}
	return result
}
