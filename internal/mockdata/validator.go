// Package mockdata provides comprehensive validation for mock data generation.
//
// This package implements multi-level validation to ensure generated mock data
// meets both structural and semantic requirements:
// - Structural validation: Type checking, required fields, parameter matching
// - Semantic validation: Format checking for emails, URLs, UUIDs, dates
// - Template validation: Expression syntax, inheritance rules, configuration
//
// The validation framework prioritizes safety and provides detailed error messages
// to help developers identify and fix mock data generation issues.
package mockdata

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// StandardMockDataValidator provides comprehensive validation for mock data.
//
// Design decisions:
// - Pre-compiled regex patterns for performance (avoid recompilation on each validation)
// - Comprehensive semantic validation based on parameter naming conventions
// - Layered validation approach: structural → type → semantic
// - Detailed error messages with context for debugging
type StandardMockDataValidator struct {
	emailRegex *regexp.Regexp // Pre-compiled email format validation
	urlRegex   *regexp.Regexp // Pre-compiled URL format validation
	uuidRegex  *regexp.Regexp // Pre-compiled UUID format validation
}

// NewMockDataValidator creates a new mock data validator.
func NewMockDataValidator() MockDataValidator {
	return &StandardMockDataValidator{
		emailRegex: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		urlRegex:   regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(/.*)?$`),
		uuidRegex:  regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
	}
}

// ValidateData validates mock data against component requirements.
//
// Validation strategy:
// 1. Check required parameters are present (structural completeness)
// 2. Validate each parameter value (type + semantic correctness)
// 3. Detect unexpected parameters (prevents typos and inconsistencies)
//
// Error accumulation: Collects all validation errors to provide comprehensive
// feedback rather than failing fast on the first error.
func (v *StandardMockDataValidator) ValidateData(
	data map[string]interface{},
	component *types.ComponentInfo,
) error {
	var errors []string

	// Check that all required parameters are present
	for _, param := range component.Parameters {
		if _, exists := data[param.Name]; !exists {
			if !param.Optional {
				errors = append(errors, fmt.Sprintf("required parameter '%s' is missing", param.Name))
			}
			continue
		}

		// Validate parameter value (type + semantic)
		if err := v.validateParameter(data[param.Name], param); err != nil {
			errors = append(errors, fmt.Sprintf("parameter '%s': %s", param.Name, err.Error()))
		}
	}

	// Check for unexpected parameters (prevents typos)
	for key := range data {
		found := false
		for _, param := range component.Parameters {
			if param.Name == key {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("unexpected parameter '%s'", key))
		}
	}

	// Return accumulated errors for comprehensive feedback
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidateTemplate validates template structure and content.
func (v *StandardMockDataValidator) ValidateTemplate(template *MockDataTemplate) error {
	var errors []string

	// Check required fields
	if template.Name == "" {
		errors = append(errors, "template name is required")
	}

	if template.Description == "" {
		errors = append(errors, "template description is required")
	}

	// Validate template name format
	if !v.isValidTemplateName(template.Name) {
		errors = append(errors, "template name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate inheritance
	if template.Extends != "" {
		if template.Extends == template.Name {
			errors = append(errors, "template cannot extend itself")
		}
	}

	// Validate field values
	for fieldName, fieldValue := range template.Fields {
		if err := v.validateTemplateField(fieldName, fieldValue); err != nil {
			errors = append(errors, fmt.Sprintf("field '%s': %s", fieldName, err.Error()))
		}
	}

	// Validate version format if provided
	if template.Version != "" {
		if !v.isValidVersion(template.Version) {
			errors = append(errors, "version must follow semantic versioning format (e.g., 1.0.0)")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("template validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidateConfig validates mock data configuration.
func (v *StandardMockDataValidator) ValidateConfig(config *MockDataConfig) error {
	var errors []string

	// Validate locale
	if config.Locale != "" && !v.isValidLocale(config.Locale) {
		errors = append(errors, "invalid locale format")
	}

	// Validate options
	if config.Options != nil {
		if err := v.validateGenerationOptions(config.Options); err != nil {
			errors = append(errors, fmt.Sprintf("options: %s", err.Error()))
		}
	}

	// Validate templates
	for _, template := range config.Templates {
		if err := v.ValidateTemplate(template); err != nil {
			errors = append(errors, fmt.Sprintf("template '%s': %s", template.Name, err.Error()))
		}
	}

	// Validate patterns
	for patternName, patternValue := range config.Patterns {
		if patternValue == "" {
			errors = append(errors, fmt.Sprintf("pattern '%s' cannot be empty", patternName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateParameter validates a parameter value against its definition.
func (v *StandardMockDataValidator) validateParameter(
	value interface{},
	param types.ParameterInfo,
) error {
	if value == nil {
		if !param.Optional {
			return fmt.Errorf("cannot be null")
		}
		return nil
	}

	// Type validation
	if err := v.validateType(value, param.Type); err != nil {
		return err
	}

	// Semantic validation based on parameter name
	if err := v.validateSemantic(value, param.Name); err != nil {
		return err
	}

	return nil
}

// validateType validates that a value matches the expected Go type.
func (v *StandardMockDataValidator) validateType(value interface{}, expectedType string) error {
	actualType := reflect.TypeOf(value)

	switch strings.ToLower(expectedType) {
	case "string":
		if actualType.Kind() != reflect.String {
			return fmt.Errorf("expected string, got %s", actualType.Kind())
		}
	case "int", "int32", "int64":
		switch actualType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Valid integer types
		case reflect.Invalid, reflect.Bool,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
			reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Pointer, reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
			return fmt.Errorf("expected integer, got %s", actualType.Kind())
		default:
			return fmt.Errorf("expected integer, got %s", actualType.Kind())
		}
	case "uint", "uint32", "uint64":
		switch actualType.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// Valid unsigned integer types
		case reflect.Invalid, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
			reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Pointer, reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
			return fmt.Errorf("expected unsigned integer, got %s", actualType.Kind())
		default:
			return fmt.Errorf("expected unsigned integer, got %s", actualType.Kind())
		}
	case "float32", "float64":
		switch actualType.Kind() {
		case reflect.Float32, reflect.Float64:
			// Valid float types
		case reflect.Invalid, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Complex64, reflect.Complex128,
			reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Pointer, reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
			return fmt.Errorf("expected float, got %s", actualType.Kind())
		default:
			return fmt.Errorf("expected float, got %s", actualType.Kind())
		}
	case "bool", "boolean":
		if actualType.Kind() != reflect.Bool {
			return fmt.Errorf("expected boolean, got %s", actualType.Kind())
		}
	case "time.time":
		// Check if it's a time.Time or string representation
		if actualType != reflect.TypeOf(time.Time{}) && actualType.Kind() != reflect.String {
			return fmt.Errorf("expected time.Time or string, got %s", actualType.Kind())
		}
	default:
		// Handle slice types
		if strings.HasPrefix(expectedType, "[]") {
			if actualType.Kind() != reflect.Slice && actualType.Kind() != reflect.Array {
				return fmt.Errorf("expected slice/array, got %s", actualType.Kind())
			}
		}
		// Handle map types
		if strings.HasPrefix(expectedType, "map[") {
			if actualType.Kind() != reflect.Map {
				return fmt.Errorf("expected map, got %s", actualType.Kind())
			}
		}
	}

	return nil
}

// validateSemantic validates that a value makes semantic sense for the parameter name.
//
// Semantic validation philosophy: Parameter names provide context about expected
// content format. For example, "email" parameters should contain valid email
// addresses, "url" parameters should be well-formed URLs.
//
// Design decisions:
// - Only validates string values (other types are structurally validated)
// - Uses parameter name analysis rather than expensive content inspection
// - Covers common web application data types (email, URL, UUID, phone, date)
func (v *StandardMockDataValidator) validateSemantic(value interface{}, paramName string) error {
	strValue, ok := value.(string)
	if !ok {
		return nil // Only validate string values semantically
	}

	paramLower := strings.ToLower(paramName)

	// Email validation
	if strings.Contains(paramLower, "email") || strings.Contains(paramLower, "mail") {
		if !v.emailRegex.MatchString(strValue) {
			return fmt.Errorf("invalid email format")
		}
	}

	// URL validation
	if strings.Contains(paramLower, "url") || strings.Contains(paramLower, "link") ||
		strings.Contains(paramLower, "href") {
		if !v.urlRegex.MatchString(strValue) {
			return fmt.Errorf("invalid URL format")
		}
	}

	// UUID validation
	if strings.Contains(paramLower, "uuid") || strings.Contains(paramLower, "guid") {
		if !v.uuidRegex.MatchString(strings.ToLower(strValue)) {
			return fmt.Errorf("invalid UUID format")
		}
	}

	// Phone number validation (basic)
	if strings.Contains(paramLower, "phone") || strings.Contains(paramLower, "tel") {
		if len(strValue) < 10 || len(strValue) > 20 {
			return fmt.Errorf("phone number should be between 10 and 20 characters")
		}
	}

	// Date validation (if it looks like a date parameter)
	if strings.Contains(paramLower, "date") || strings.Contains(paramLower, "time") {
		if err := v.validateDateString(strValue); err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
	}

	return nil
}

// validateTemplateField validates a template field value.
func (v *StandardMockDataValidator) validateTemplateField(fieldName string, fieldValue interface{}) error {
	if fieldValue == nil {
		return nil
	}

	switch value := fieldValue.(type) {
	case string:
		// Validate template expressions - check for any template syntax
		if strings.Contains(value, "{{") {
			return v.validateTemplateExpression(value)
		}
	case map[string]interface{}:
		// Recursive validation for nested objects
		for nestedName, nestedValue := range value {
			if err := v.validateTemplateField(nestedName, nestedValue); err != nil {
				return fmt.Errorf("nested field '%s': %w", nestedName, err)
			}
		}
	case []interface{}:
		// Validate array elements
		for i, item := range value {
			if err := v.validateTemplateField(fmt.Sprintf("%s[%d]", fieldName, i), item); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateTemplateExpression validates faker-style template expressions.
//
// Template expression format: {{namespace.function}} (e.g., {{person.name}})
//
// Parsing strategy:
// 1. Find all template expressions using {{ }} delimiters
// 2. Extract and validate each expression syntax
// 3. Check against whitelist of known faker expressions
//
// Safety measures:
// - Detects unclosed expressions to prevent template rendering errors
// - Validates against known expression patterns to catch typos
// - Handles multiple expressions in single template string
func (v *StandardMockDataValidator) validateTemplateExpression(expression string) error {
	// Extract all template expressions using delimiter scanning
	start := 0
	for {
		openIdx := strings.Index(expression[start:], "{{")
		if openIdx == -1 {
			break // No more expressions found
		}
		openIdx += start

		closeIdx := strings.Index(expression[openIdx:], "}}")
		if closeIdx == -1 {
			return fmt.Errorf("unclosed template expression")
		}
		closeIdx += openIdx

		// Extract expression content
		templateExpr := expression[openIdx+2 : closeIdx]
		templateExpr = strings.TrimSpace(templateExpr)

		if templateExpr == "" {
			return fmt.Errorf("empty template expression")
		}

		// Validate against known faker expressions
		if !v.isValidTemplateExpression(templateExpr) {
			return fmt.Errorf("unknown template expression: %s", templateExpr)
		}

		start = closeIdx + 2 // Continue after current expression
	}

	return nil
}

// validateGenerationOptions validates generation options.
func (v *StandardMockDataValidator) validateGenerationOptions(options *GenerationOptions) error {
	var errors []string

	// Validate string length range
	if options.StringLength != nil {
		if options.StringLength.Min < 0 {
			errors = append(errors, "string length minimum cannot be negative")
		}
		if options.StringLength.Max < options.StringLength.Min {
			errors = append(errors, "string length maximum cannot be less than minimum")
		}
	}

	// Validate array length range
	if options.ArrayLength != nil {
		if options.ArrayLength.Min < 0 {
			errors = append(errors, "array length minimum cannot be negative")
		}
		if options.ArrayLength.Max < options.ArrayLength.Min {
			errors = append(errors, "array length maximum cannot be less than minimum")
		}
	}

	// Validate date range
	if options.DateRange != nil {
		if options.DateRange.Start != nil && options.DateRange.End != nil {
			if options.DateRange.End.Before(*options.DateRange.Start) {
				errors = append(errors, "date range end cannot be before start")
			}
		}
	}

	// Validate number range
	if options.NumberRange != nil {
		if options.NumberRange.Max < options.NumberRange.Min {
			errors = append(errors, "number range maximum cannot be less than minimum")
		}
	}

	// Validate validation level
	if options.ValidationLevel != "" {
		validLevels := []string{"none", "basic", "strict", "comprehensive"}
		valid := false
		for _, level := range validLevels {
			if options.ValidationLevel == level {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, "validation level must be one of: none, basic, strict, comprehensive")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper validation methods
func (v *StandardMockDataValidator) isValidTemplateName(name string) bool {
	match, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return match
}

func (v *StandardMockDataValidator) isValidVersion(version string) bool {
	match, _ := regexp.MatchString(`^\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`, version)
	return match
}

func (v *StandardMockDataValidator) isValidLocale(locale string) bool {
	match, _ := regexp.MatchString(`^[a-z]{2}(-[A-Z]{2})?$`, locale)
	return match
}

func (v *StandardMockDataValidator) isValidTemplateExpression(expr string) bool {
	validExpressions := []string{
		"person.name", "person.firstName", "person.lastName", "person.jobTitle",
		"internet.email", "internet.url", "internet.avatar",
		"address.street", "address.city", "address.country",
		"company.name", "company.catchPhrase",
		"lorem.word", "lorem.sentence", "lorem.paragraph", "lorem.paragraphs",
		"number.int", "number.float", "number.age",
		"datatype.boolean",
		"time.recent", "time.past", "time.future", "time.birthdate",
		"phone.number",
		"commerce.productName", "commerce.price", "commerce.department",
		"finance.currencyCode",
		"image.url", "image.business", "image.avatar",
	}

	for _, valid := range validExpressions {
		if expr == valid {
			return true
		}
	}

	return false
}

func (v *StandardMockDataValidator) validateDateString(dateStr string) error {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"01/02/2006",
		"01-02-2006",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, dateStr); err == nil {
			return nil
		}
	}

	return fmt.Errorf("unrecognized date format")
}
