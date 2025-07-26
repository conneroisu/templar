package errors

import (
	"errors"
	"testing"
)

func TestTemplarError(t *testing.T) {
	tests := []struct {
		name     string
		err      *TemplarError
		expected string
	}{
		{
			name: "basic error",
			err: &TemplarError{
				Type:    ErrorTypeValidation,
				Code:    "TEST_ERROR",
				Message: "test message",
			},
			expected: "[TEST_ERROR] test message",
		},
		{
			name: "error with component",
			err: &TemplarError{
				Type:      ErrorTypeValidation,
				Code:      "TEST_ERROR",
				Message:   "test message",
				Component: "TestComponent",
			},
			expected: "[TEST_ERROR] component:TestComponent test message",
		},
		{
			name: "error with location",
			err: &TemplarError{
				Type:     ErrorTypeValidation,
				Code:     "TEST_ERROR",
				Message:  "test message",
				FilePath: "test.go",
				Line:     10,
				Column:   5,
			},
			expected: "[TEST_ERROR] test.go:10:5 test message",
		},
		{
			name: "error with cause",
			err: &TemplarError{
				Type:    ErrorTypeValidation,
				Code:    "TEST_ERROR",
				Message: "test message",
				Cause:   errors.New("underlying error"),
			},
			expected: "[TEST_ERROR] test message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("TemplarError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	fieldErr := NewFieldValidationError(
		"username",
		"invalid",
		"must be at least 3 characters",
		"Use a longer username",
		"Avoid special characters",
	)

	if fieldErr.Field() != "username" {
		t.Errorf("Field() = %v, want %v", fieldErr.Field(), "username")
	}

	if fieldErr.Value() != "invalid" {
		t.Errorf("Value() = %v, want %v", fieldErr.Value(), "invalid")
	}

	suggestions := fieldErr.Suggestions()
	if len(suggestions) != 2 {
		t.Errorf("Suggestions() length = %v, want %v", len(suggestions), 2)
	}

	expected := "validation error in field 'username': must be at least 3 characters"
	if fieldErr.Error() != expected {
		t.Errorf("Error() = %v, want %v", fieldErr.Error(), expected)
	}
}

func TestValidationErrorCollection(t *testing.T) {
	collection := &ValidationErrorCollection{}

	// Test empty collection
	if collection.HasErrors() {
		t.Error("HasErrors() should return false for empty collection")
	}

	// Add field error
	collection.AddField(
		"email",
		"invalid-email",
		"must be valid email",
		"Use format: user@domain.com",
	)

	if !collection.HasErrors() {
		t.Error("HasErrors() should return true after adding error")
	}

	if len(collection.Errors) != 1 {
		t.Errorf("Collection should have 1 error, got %d", len(collection.Errors))
	}

	// Convert to TemplarError
	templErr := collection.ToTemplarError()
	if templErr == nil {
		t.Fatal("ToTemplarError() should not return nil")
	}

	if templErr.Type != ErrorTypeValidation {
		t.Errorf("TemplarError type = %v, want %v", templErr.Type, ErrorTypeValidation)
	}

	if templErr.Code != ErrCodeValidationFailed {
		t.Errorf("TemplarError code = %v, want %v", templErr.Code, ErrCodeValidationFailed)
	}
}

func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("original error")

	// Test basic wrapping
	wrappedErr := Wrap(originalErr, ErrorTypeBuild, "BUILD_FAILED", "build operation failed")
	if wrappedErr == nil {
		t.Fatal("Wrap() should not return nil")
	}

	if wrappedErr.Type != ErrorTypeBuild {
		t.Errorf("Wrapped error type = %v, want %v", wrappedErr.Type, ErrorTypeBuild)
	}

	if wrappedErr.Cause != originalErr {
		t.Errorf("Wrapped error cause = %v, want %v", wrappedErr.Cause, originalErr)
	}

	// Test wrapping existing TemplarError
	existingTemplErr := &TemplarError{
		Type:      ErrorTypeValidation,
		Code:      "VALIDATION_ERROR",
		Message:   "validation failed",
		Component: "TestComponent",
	}

	rewrappedErr := Wrap(existingTemplErr, ErrorTypeBuild, "BUILD_FAILED", "build operation failed")
	if rewrappedErr.Component != "TestComponent" {
		t.Errorf("Rewrapped error should preserve component = %v", rewrappedErr.Component)
	}

	if rewrappedErr.Cause != existingTemplErr {
		t.Errorf("Rewrapped error cause should be original TemplarError")
	}
}

func TestSpecializedWrappers(t *testing.T) {
	originalErr := errors.New("test error")

	// Test build wrapper
	buildErr := WrapBuild(originalErr, "BUILD_FAILED", "build failed", "TestComponent")
	if buildErr.Type != ErrorTypeBuild {
		t.Errorf("WrapBuild type = %v, want %v", buildErr.Type, ErrorTypeBuild)
	}
	if buildErr.Component != "TestComponent" {
		t.Errorf("WrapBuild component = %v, want %v", buildErr.Component, "TestComponent")
	}

	// Test security wrapper
	securityErr := WrapSecurity(originalErr, "SECURITY_VIOLATION", "security error")
	if securityErr.Type != ErrorTypeSecurity {
		t.Errorf("WrapSecurity type = %v, want %v", securityErr.Type, ErrorTypeSecurity)
	}
	if securityErr.Recoverable {
		t.Error("WrapSecurity should create non-recoverable error")
	}

	// Test validation wrapper
	validationErr := WrapValidation(originalErr, "VALIDATION_FAILED", "validation error")
	if validationErr.Type != ErrorTypeValidation {
		t.Errorf("WrapValidation type = %v, want %v", validationErr.Type, ErrorTypeValidation)
	}
	if !validationErr.Recoverable {
		t.Error("WrapValidation should create recoverable error")
	}
}

func TestErrorEnhancement(t *testing.T) {
	originalErr := errors.New("original error")

	enhancedErr := EnhanceError(originalErr, "TestComponent", "test.go", 10, 5)
	if enhancedErr == nil {
		t.Fatal("EnhanceError should not return nil")
	}

	var templErr *TemplarError
	ok := errors.As(enhancedErr, &templErr)
	if !ok {
		t.Fatal("EnhanceError should return TemplarError")
	}

	if templErr.Component != "TestComponent" {
		t.Errorf("Enhanced error component = %v, want %v", templErr.Component, "TestComponent")
	}

	if templErr.FilePath != "test.go" {
		t.Errorf("Enhanced error file path = %v, want %v", templErr.FilePath, "test.go")
	}

	if templErr.Line != 10 {
		t.Errorf("Enhanced error line = %v, want %v", templErr.Line, 10)
	}

	if templErr.Column != 5 {
		t.Errorf("Enhanced error column = %v, want %v", templErr.Column, 5)
	}
}

func TestErrorUtilities(t *testing.T) {
	// Test IsRecoverable
	recoverableErr := &TemplarError{Type: ErrorTypeValidation, Recoverable: true}
	if !IsRecoverable(recoverableErr) {
		t.Error("IsRecoverable should return true for recoverable error")
	}

	nonRecoverableErr := &TemplarError{Type: ErrorTypeSecurity, Recoverable: false}
	if IsRecoverable(nonRecoverableErr) {
		t.Error("IsRecoverable should return false for non-recoverable error")
	}

	// Test IsSecurityError
	securityErr := &TemplarError{Type: ErrorTypeSecurity}
	if !IsSecurityError(securityErr) {
		t.Error("IsSecurityError should return true for security error")
	}

	buildErr := &TemplarError{Type: ErrorTypeBuild}
	if IsSecurityError(buildErr) {
		t.Error("IsSecurityError should return false for build error")
	}

	// Test IsBuildError
	if !IsBuildError(buildErr) {
		t.Error("IsBuildError should return true for build error")
	}

	if IsBuildError(securityErr) {
		t.Error("IsBuildError should return false for security error")
	}
}

func TestErrorCollection(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	var nilErr error

	// Test CollectErrors
	collected := CollectErrors(err1, nilErr, err2)
	if len(collected) != 2 {
		t.Errorf("CollectErrors should return 2 errors, got %d", len(collected))
	}

	// Test FirstError
	first := FirstError(nilErr, err1, err2)
	if first != err1 {
		t.Errorf("FirstError should return first non-nil error")
	}

	// Test CombineErrors
	combined := CombineErrors(err1, err2)
	if combined == nil {
		t.Fatal("CombineErrors should not return nil")
	}

	templErr, ok := combined.(*TemplarError)
	if !ok {
		t.Fatal("CombineErrors should return TemplarError")
	}

	if templErr.Type != ErrorTypeInternal {
		t.Errorf("Combined error type = %v, want %v", templErr.Type, ErrorTypeInternal)
	}
}

func TestErrorContext(t *testing.T) {
	err := &TemplarError{
		Type:      ErrorTypeBuild,
		Code:      "BUILD_FAILED",
		Message:   "build failed",
		Component: "TestComponent",
		FilePath:  "test.go",
		Line:      10,
		Column:    5,
		Context: map[string]interface{}{
			"custom": "value",
		},
	}

	context := GetErrorContext(err)

	expectedKeys := []string{
		"component",
		"file",
		"line",
		"column",
		"type",
		"code",
		"recoverable",
		"custom",
	}
	for _, key := range expectedKeys {
		if _, exists := context[key]; !exists {
			t.Errorf("Context should contain key %s", key)
		}
	}

	if context["component"] != "TestComponent" {
		t.Errorf("Context component = %v, want %v", context["component"], "TestComponent")
	}

	if context["type"] != string(ErrorTypeBuild) {
		t.Errorf("Context type = %v, want %v", context["type"], string(ErrorTypeBuild))
	}
}

func TestTemporaryAndFatalErrors(t *testing.T) {
	// Test temporary errors
	buildErr := &TemplarError{Type: ErrorTypeBuild}
	if !IsTemporaryError(buildErr) {
		t.Error("Build errors should be considered temporary")
	}

	validationErr := &TemplarError{Type: ErrorTypeValidation}
	if !IsTemporaryError(validationErr) {
		t.Error("Validation errors should be considered temporary")
	}

	// Test fatal errors
	securityErr := &TemplarError{Type: ErrorTypeSecurity}
	if !IsFatalError(securityErr) {
		t.Error("Security errors should be considered fatal")
	}

	internalErr := &TemplarError{Type: ErrorTypeInternal}
	if !IsFatalError(internalErr) {
		t.Error("Internal errors should be considered fatal")
	}

	// Non-fatal error
	if IsFatalError(buildErr) {
		t.Error("Build errors should not be considered fatal")
	}
}

func TestExtractCause(t *testing.T) {
	rootErr := errors.New("root cause")

	wrappedErr := &TemplarError{
		Type:    ErrorTypeBuild,
		Code:    "BUILD_FAILED",
		Message: "build failed",
		Cause:   rootErr,
	}

	doubleWrappedErr := &TemplarError{
		Type:    ErrorTypeInternal,
		Code:    "INTERNAL_ERROR",
		Message: "internal error",
		Cause:   wrappedErr,
	}

	extracted := ExtractCause(doubleWrappedErr)
	if extracted != rootErr {
		t.Errorf("ExtractCause should return root cause")
	}
}
