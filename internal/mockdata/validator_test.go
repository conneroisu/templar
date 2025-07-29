// Package mockdata provides tests for mock data validation.
package mockdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/conneroisu/templar/internal/types"
)

func TestStandardMockDataValidator_ValidateData(t *testing.T) {
	validator := NewMockDataValidator()

	tests := []struct {
		name        string
		data        map[string]interface{}
		component   *types.ComponentInfo
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid data with all required fields",
			data: map[string]interface{}{
				"name":     "John Doe",
				"email":    "john@example.com",
				"age":      30,
				"isActive": true,
			},
			component: &types.ComponentInfo{
				Name: "UserProfile",
				Parameters: []types.ParameterInfo{
					{Name: "name", Type: "string", Optional: false},
					{Name: "email", Type: "string", Optional: false},
					{Name: "age", Type: "int", Optional: true},
					{Name: "isActive", Type: "bool", Optional: true},
				},
			},
			expectError: false,
		},
		{
			name: "missing required field",
			data: map[string]interface{}{
				"name": "John Doe",
				"age":  30,
			},
			component: &types.ComponentInfo{
				Name: "UserProfile",
				Parameters: []types.ParameterInfo{
					{Name: "name", Type: "string", Optional: false},
					{Name: "email", Type: "string", Optional: false},
					{Name: "age", Type: "int", Optional: true},
				},
			},
			expectError: true,
			errorMsg:    "required parameter 'email' is missing",
		},
		{
			name: "unexpected parameter",
			data: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"extra": "unexpected",
			},
			component: &types.ComponentInfo{
				Name: "UserProfile",
				Parameters: []types.ParameterInfo{
					{Name: "name", Type: "string", Optional: false},
					{Name: "email", Type: "string", Optional: false},
				},
			},
			expectError: true,
			errorMsg:    "unexpected parameter 'extra'",
		},
		{
			name: "invalid email format",
			data: map[string]interface{}{
				"email": "not-an-email",
			},
			component: &types.ComponentInfo{
				Name: "UserProfile",
				Parameters: []types.ParameterInfo{
					{Name: "email", Type: "string", Optional: false},
				},
			},
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name: "invalid type",
			data: map[string]interface{}{
				"age": "thirty", // Should be int
			},
			component: &types.ComponentInfo{
				Name: "UserProfile",
				Parameters: []types.ParameterInfo{
					{Name: "age", Type: "int", Optional: false},
				},
			},
			expectError: true,
			errorMsg:    "expected integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateData(tt.data, tt.component)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_ValidateTemplate(t *testing.T) {
	validator := NewMockDataValidator()

	tests := []struct {
		name        string
		template    *MockDataTemplate
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid template",
			template: &MockDataTemplate{
				Name:        "valid-template",
				Description: "A valid template for testing",
				Fields: map[string]interface{}{
					"name":  "{{person.name}}",
					"email": "{{internet.email}}",
				},
				Tags:    []string{"test"},
				Version: "1.0.0",
			},
			expectError: false,
		},
		{
			name: "missing name",
			template: &MockDataTemplate{
				Description: "Template without name",
				Fields:      map[string]interface{}{},
			},
			expectError: true,
			errorMsg:    "template name is required",
		},
		{
			name: "missing description",
			template: &MockDataTemplate{
				Name:   "no-description",
				Fields: map[string]interface{}{},
			},
			expectError: true,
			errorMsg:    "template description is required",
		},
		{
			name: "invalid name format",
			template: &MockDataTemplate{
				Name:        "invalid name with spaces!",
				Description: "Template with invalid name",
				Fields:      map[string]interface{}{},
			},
			expectError: true,
			errorMsg:    "template name must contain only alphanumeric characters",
		},
		{
			name: "circular inheritance",
			template: &MockDataTemplate{
				Name:        "circular",
				Description: "Circular template",
				Extends:     "circular",
				Fields:      map[string]interface{}{},
			},
			expectError: true,
			errorMsg:    "template cannot extend itself",
		},
		{
			name: "invalid version format",
			template: &MockDataTemplate{
				Name:        "bad-version",
				Description: "Template with bad version",
				Fields:      map[string]interface{}{},
				Version:     "not.a.version",
			},
			expectError: true,
			errorMsg:    "version must follow semantic versioning format",
		},
		{
			name: "invalid template expression",
			template: &MockDataTemplate{
				Name:        "bad-expression",
				Description: "Template with bad expression",
				Fields: map[string]interface{}{
					"field": "{{unknown.expression}}",
				},
			},
			expectError: true,
			errorMsg:    "unknown template expression",
		},
		{
			name: "unclosed template expression",
			template: &MockDataTemplate{
				Name:        "unclosed-expression",
				Description: "Template with unclosed expression",
				Fields: map[string]interface{}{
					"field": "{{person.name",
				},
			},
			expectError: true,
			errorMsg:    "unclosed template expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTemplate(tt.template)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_ValidateConfig(t *testing.T) {
	validator := NewMockDataValidator()

	tests := []struct {
		name        string
		config      *MockDataConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &MockDataConfig{
				DefaultTemplate: "default",
				Templates:       []*MockDataTemplate{},
				Patterns:        map[string]string{"email": "email pattern"},
				Seed:            12345,
				Locale:          "en",
				Options: &GenerationOptions{
					UseRealisticData:  true,
					IncludeNullValues: false,
					StringLength:      &LengthRange{Min: 5, Max: 50},
					ArrayLength:       &LengthRange{Min: 1, Max: 5},
					CacheResults:      true,
					ValidationLevel:   "basic",
				},
			},
			expectError: false,
		},
		{
			name: "invalid locale",
			config: &MockDataConfig{
				Locale: "invalid-locale-format",
				Options: &GenerationOptions{
					ValidationLevel: "basic",
				},
			},
			expectError: true,
			errorMsg:    "invalid locale format",
		},
		{
			name: "invalid validation level",
			config: &MockDataConfig{
				Locale: "en",
				Options: &GenerationOptions{
					ValidationLevel: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "validation level must be one of",
		},
		{
			name: "invalid string length range",
			config: &MockDataConfig{
				Options: &GenerationOptions{
					StringLength: &LengthRange{Min: 50, Max: 10}, // Max < Min
				},
			},
			expectError: true,
			errorMsg:    "string length maximum cannot be less than minimum",
		},
		{
			name: "invalid array length range",
			config: &MockDataConfig{
				Options: &GenerationOptions{
					ArrayLength: &LengthRange{Min: -1, Max: 10}, // Negative min
				},
			},
			expectError: true,
			errorMsg:    "array length minimum cannot be negative",
		},
		{
			name: "invalid date range",
			config: &MockDataConfig{
				Options: &GenerationOptions{
					DateRange: &DateRange{
						Start: &time.Time{},
						End:   &time.Time{},
					},
				},
			},
			expectError: false, // This should be OK since both are zero values
		},
		{
			name: "empty pattern",
			config: &MockDataConfig{
				Patterns: map[string]string{
					"email": "",
				},
			},
			expectError: true,
			errorMsg:    "pattern 'email' cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_SemanticValidation(t *testing.T) {
	validator := NewMockDataValidator().(*StandardMockDataValidator)

	tests := []struct {
		name        string
		value       interface{}
		paramName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid email",
			value:       "test@example.com",
			paramName:   "email",
			expectError: false,
		},
		{
			name:        "invalid email",
			value:       "not-an-email",
			paramName:   "userEmail",
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name:        "valid URL",
			value:       "https://example.com/page",
			paramName:   "profileUrl",
			expectError: false,
		},
		{
			name:        "invalid URL",
			value:       "not-a-url",
			paramName:   "websiteUrl",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "valid UUID",
			value:       "550e8400-e29b-41d4-a716-446655440000",
			paramName:   "userId",
			expectError: false,
		},
		{
			name:        "invalid UUID",
			value:       "not-a-uuid",
			paramName:   "uuid",
			expectError: true,
			errorMsg:    "invalid UUID format",
		},
		{
			name:        "valid phone",
			value:       "+1-555-123-4567",
			paramName:   "phoneNumber",
			expectError: false,
		},
		{
			name:        "invalid phone (too short)",
			value:       "123",
			paramName:   "phone",
			expectError: true,
			errorMsg:    "phone number should be between 10 and 20 characters",
		},
		{
			name:        "valid date",
			value:       "2024-01-15",
			paramName:   "createdDate",
			expectError: false,
		},
		{
			name:        "invalid date",
			value:       "not-a-date",
			paramName:   "birthDate",
			expectError: true,
			errorMsg:    "invalid date format",
		},
		{
			name:        "non-string value (should not validate semantically)",
			value:       123,
			paramName:   "email",
			expectError: false, // Semantic validation only applies to strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSemantic(tt.value, tt.paramName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_TypeValidation(t *testing.T) {
	validator := NewMockDataValidator().(*StandardMockDataValidator)

	tests := []struct {
		name         string
		value        interface{}
		expectedType string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid string",
			value:        "test string",
			expectedType: "string",
			expectError:  false,
		},
		{
			name:         "invalid string type",
			value:        123,
			expectedType: "string",
			expectError:  true,
			errorMsg:     "expected string",
		},
		{
			name:         "valid int",
			value:        42,
			expectedType: "int",
			expectError:  false,
		},
		{
			name:         "valid int32",
			value:        int32(42),
			expectedType: "int32",
			expectError:  false,
		},
		{
			name:         "invalid int type",
			value:        "not a number",
			expectedType: "int",
			expectError:  true,
			errorMsg:     "expected integer",
		},
		{
			name:         "valid float",
			value:        3.14,
			expectedType: "float64",
			expectError:  false,
		},
		{
			name:         "invalid float type",
			value:        "not a float",
			expectedType: "float64",
			expectError:  true,
			errorMsg:     "expected float",
		},
		{
			name:         "valid bool",
			value:        true,
			expectedType: "bool",
			expectError:  false,
		},
		{
			name:         "invalid bool type",
			value:        "true",
			expectedType: "bool",
			expectError:  true,
			errorMsg:     "expected boolean",
		},
		{
			name:         "valid time.Time",
			value:        time.Now(),
			expectedType: "time.Time",
			expectError:  false,
		},
		{
			name:         "valid time string",
			value:        "2024-01-15T10:30:00Z",
			expectedType: "time.Time",
			expectError:  false,
		},
		{
			name:         "valid slice",
			value:        []string{"a", "b", "c"},
			expectedType: "[]string",
			expectError:  false,
		},
		{
			name:         "invalid slice type",
			value:        "not a slice",
			expectedType: "[]string",
			expectError:  true,
			errorMsg:     "expected slice/array",
		},
		{
			name:         "valid map",
			value:        map[string]string{"key": "value"},
			expectedType: "map[string]string",
			expectError:  false,
		},
		{
			name:         "invalid map type",
			value:        []string{"not", "a", "map"},
			expectedType: "map[string]string",
			expectError:  true,
			errorMsg:     "expected map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateType(tt.value, tt.expectedType)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_TemplateExpressionValidation(t *testing.T) {
	validator := NewMockDataValidator().(*StandardMockDataValidator)

	tests := []struct {
		name        string
		expression  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid single expression",
			expression:  "{{person.name}}",
			expectError: false,
		},
		{
			name:        "valid multiple expressions",
			expression:  "Name: {{person.name}}, Email: {{internet.email}}",
			expectError: false,
		},
		{
			name:        "invalid expression",
			expression:  "{{unknown.expression}}",
			expectError: true,
			errorMsg:    "unknown template expression",
		},
		{
			name:        "unclosed expression",
			expression:  "{{person.name",
			expectError: true,
			errorMsg:    "unclosed template expression",
		},
		{
			name:        "empty expression",
			expression:  "{{}}",
			expectError: true,
			errorMsg:    "empty template expression",
		},
		{
			name:        "no template expressions",
			expression:  "plain text",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTemplateExpression(tt.expression)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardMockDataValidator_HelperMethods(t *testing.T) {
	validator := NewMockDataValidator().(*StandardMockDataValidator)

	// Test template name validation
	validNames := []string{"valid-name", "valid_name", "ValidName123"}
	for _, name := range validNames {
		assert.True(t, validator.isValidTemplateName(name), "Should be valid template name: %s", name)
	}

	invalidNames := []string{"invalid name", "invalid@name", "invalid.name"}
	for _, name := range invalidNames {
		assert.False(t, validator.isValidTemplateName(name), "Should be invalid template name: %s", name)
	}

	// Test version validation
	validVersions := []string{"1.0.0", "2.1.3", "1.0.0-beta"}
	for _, version := range validVersions {
		assert.True(t, validator.isValidVersion(version), "Should be valid version: %s", version)
	}

	invalidVersions := []string{"1.0", "1.0.0.0", "v1.0.0", "1.0.0-"}
	for _, version := range invalidVersions {
		assert.False(t, validator.isValidVersion(version), "Should be invalid version: %s", version)
	}

	// Test locale validation
	validLocales := []string{"en", "en-US", "fr", "de-DE"}
	for _, locale := range validLocales {
		assert.True(t, validator.isValidLocale(locale), "Should be valid locale: %s", locale)
	}

	invalidLocales := []string{"english", "en-us", "EN", "en-USA"}
	for _, locale := range invalidLocales {
		assert.False(t, validator.isValidLocale(locale), "Should be invalid locale: %s", locale)
	}

	// Test template expression validation
	validExpressions := []string{
		"person.name", "internet.email", "lorem.sentence", "number.int", "datatype.boolean",
	}
	for _, expr := range validExpressions {
		assert.True(t, validator.isValidTemplateExpression(expr), "Should be valid expression: %s", expr)
	}

	invalidExpressions := []string{
		"unknown.expression", "invalid", "person.unknownField",
	}
	for _, expr := range invalidExpressions {
		assert.False(t, validator.isValidTemplateExpression(expr), "Should be invalid expression: %s", expr)
	}
}

func TestStandardMockDataValidator_DateValidation(t *testing.T) {
	validator := NewMockDataValidator().(*StandardMockDataValidator)

	validDates := []string{
		"2024-01-15",
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00+07:00",
		"2024-01-15 10:30:00",
		"01/15/2024",
		"01-15-2024",
	}

	for _, date := range validDates {
		err := validator.validateDateString(date)
		assert.NoError(t, err, "Should be valid date: %s", date)
	}

	invalidDates := []string{
		"not-a-date",
		"2024-13-01", // Invalid month
		"2024/01/15", // Wrong separator
		"15-01-2024", // Wrong format
	}

	for _, date := range invalidDates {
		err := validator.validateDateString(date)
		assert.Error(t, err, "Should be invalid date: %s", date)
	}
}
