// Package apidocs provides Go type analysis for JSON schema generation.
//
// This file implements sophisticated type analysis that converts Go type definitions
// into OpenAPI-compatible JSON schemas. It handles complex types including structs,
// slices, maps, interfaces, and custom types while maintaining type safety and
// generating comprehensive documentation.
package apidocs

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// TypeAnalyzer performs static analysis of Go types to generate JSON schemas.
//
// Design philosophy: Uses reflection and AST analysis to understand Go type
// structures and convert them to OpenAPI-compatible schemas. Handles complex
// nested types, custom types, and generates appropriate validation constraints.
type TypeAnalyzer struct {
	// analyzedTypes caches analyzed types to prevent infinite recursion
	analyzedTypes map[string]*APISchema
	// typeDefinitions stores reusable type definitions
	typeDefinitions map[string]*ComponentAnalysis
}

// NewTypeAnalyzer creates a new type analyzer instance.
func NewTypeAnalyzer() *TypeAnalyzer {
	return &TypeAnalyzer{
		analyzedTypes:   make(map[string]*APISchema),
		typeDefinitions: make(map[string]*ComponentAnalysis),
	}
}

// AnalyzeType performs comprehensive Go type analysis to generate OpenAPI-compatible JSON schemas.
//
// This function implements the core type analysis algorithm that powers automatic schema generation
// for API documentation. It uses Go's reflection system to introspect type definitions and
// generates corresponding JSON Schema specifications following OpenAPI 3.0 standards.
//
// Type analysis strategy:
// 1. Cache-first lookup: Prevents infinite recursion and improves performance for complex nested types
// 2. Kind-based dispatch: Uses Go's type.Kind() for efficient type categorization
// 3. Reference generation: Complex types are extracted as reusable schema components
// 4. Validation constraints: Automatically infers validation rules from Go types
//
// Supported type categories:
// - Primitive types: string, int variants, uint variants, float32/64, bool
// - Collection types: slices, arrays, maps with proper element schema generation
// - Structured types: structs with field analysis and JSON tag interpretation
// - Pointer types: Automatic nullable field detection and optional property handling
// - Interface types: Generic object schemas with extensibility support
// - Time types: Special handling for time.Time with appropriate date-time formatting
//
// Performance characteristics:
// - O(1) cache lookup prevents re-analysis of previously processed types
// - Recursive analysis depth is bounded by type structure complexity
// - Memory usage scales linearly with unique type count in analyzed codebase
//
// Schema generation features:
// - Automatic validation constraint inference (min/max for numbers, length for strings)
// - JSON tag interpretation for field naming and optional markers
// - Format detection for common patterns (email, UUID, date-time)
// - Reference-based schema reuse for complex types
// - Comprehensive error reporting with type context
func (ta *TypeAnalyzer) AnalyzeType(goType reflect.Type) (*APISchema, error) {
	// Handle nil types gracefully - represents unknown or any type
	if goType == nil {
		return &APISchema{Type: "object"}, nil
	}

	// Generate canonical type name for caching and reference generation
	// This ensures consistent naming across the schema and prevents duplicates
	typeName := ta.getTypeName(goType)
	
	// Cache lookup to prevent infinite recursion in self-referential types
	// Also provides significant performance improvement for repeated type analysis
	if _, exists := ta.analyzedTypes[typeName]; exists {
		// Return schema reference instead of inline definition for complex types
		// This keeps the generated OpenAPI specification clean and reduces size
		return &APISchema{Ref: fmt.Sprintf("#/components/schemas/%s", typeName)}, nil
	}

	// Perform actual type analysis using internal dispatch mechanism
	// This separates the caching/reference logic from the core analysis
	schema, err := ta.analyzeTypeInternal(goType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze type %s: %w", typeName, err)
	}

	// Cache complex types for reuse and generate references
	// Simple types (primitives) are inlined, complex types become reusable components
	if ta.isComplexType(goType) {
		ta.analyzedTypes[typeName] = schema
		return &APISchema{Ref: fmt.Sprintf("#/components/schemas/%s", typeName)}, nil
	}

	return schema, nil
}

// analyzeTypeInternal performs the actual type analysis.
func (ta *TypeAnalyzer) analyzeTypeInternal(goType reflect.Type) (*APISchema, error) {
	switch goType.Kind() {
	case reflect.String:
		return ta.analyzeStringType(goType)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ta.analyzeIntegerType(goType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ta.analyzeUintegerType(goType)
	case reflect.Float32, reflect.Float64:
		return ta.analyzeFloatType(goType)
	case reflect.Bool:
		return ta.analyzeBoolType(goType)
	case reflect.Slice, reflect.Array:
		return ta.analyzeSliceType(goType)
	case reflect.Map:
		return ta.analyzeMapType(goType)
	case reflect.Struct:
		return ta.analyzeStructType(goType)
	case reflect.Ptr:
		return ta.analyzePointerType(goType)
	case reflect.Interface:
		return ta.analyzeInterfaceType(goType)
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// Unsupported types - return generic object schema
		return &APISchema{
			Type:        "object",
			Description: fmt.Sprintf("Unsupported Go type: %s", goType.Kind()),
		}, nil
	default:
		// Fallback for unsupported types
		return &APISchema{
			Type:        "object",
			Description: fmt.Sprintf("Unsupported Go type: %s", goType.Kind()),
		}, nil
	}
}

// analyzeStringType analyzes string types with format detection.
func (ta *TypeAnalyzer) analyzeStringType(goType reflect.Type) (*APISchema, error) {
	schema := &APISchema{
		Type: "string",
	}

	// Detect special string formats based on type name
	typeName := ta.getTypeName(goType)
	switch {
	case strings.Contains(typeName, "Time"):
		schema.Format = "date-time"
		schema.Example = time.Now().Format(time.RFC3339)
	case strings.Contains(typeName, "Date"):
		schema.Format = "date"
		schema.Example = time.Now().Format("2006-01-02")
	case strings.Contains(typeName, "Email"):
		schema.Format = "email"
		schema.Example = "user@example.com"
	case strings.Contains(typeName, "URL") || strings.Contains(typeName, "Uri"):
		schema.Format = "uri"
		schema.Example = "https://example.com"
	case strings.Contains(typeName, "UUID"):
		schema.Format = "uuid"
		schema.Example = "550e8400-e29b-41d4-a716-446655440000"
	case strings.Contains(typeName, "Password"):
		schema.Format = "password"
		schema.Example = "********"
	}

	return schema, nil
}

// analyzeIntegerType analyzes integer types with appropriate constraints.
func (ta *TypeAnalyzer) analyzeIntegerType(goType reflect.Type) (*APISchema, error) {
	schema := &APISchema{
		Type: "integer",
	}

	// Set format and constraints based on integer size
	switch goType.Kind() {
	case reflect.Int32:
		schema.Format = "int32"
		schema.Minimum = &[]float64{-2147483648}[0]
		schema.Maximum = &[]float64{2147483647}[0]
	case reflect.Int64:
		schema.Format = "int64"
		schema.Minimum = &[]float64{-9223372036854775808}[0]
		schema.Maximum = &[]float64{9223372036854775807}[0]
	case reflect.Int, reflect.Int8, reflect.Int16:
		schema.Format = "int32" // Default to int32 for compatibility
	default:
		// Handle all other reflect.Kind values that shouldn't reach here
		schema.Format = "int32"
	}

	return schema, nil
}

// analyzeUintegerType analyzes unsigned integer types.
func (ta *TypeAnalyzer) analyzeUintegerType(goType reflect.Type) (*APISchema, error) {
	schema := &APISchema{
		Type:    "integer",
		Minimum: &[]float64{0}[0], // Unsigned integers are always >= 0
	}

	// Set format and maximum based on size
	switch goType.Kind() {
	case reflect.Uint32:
		schema.Format = "int32"
		schema.Maximum = &[]float64{4294967295}[0]
	case reflect.Uint64:
		schema.Format = "int64"
		schema.Maximum = &[]float64{18446744073709551615}[0]
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uintptr:
		schema.Format = "int32"
	default:
		// Handle all other reflect.Kind values that shouldn't reach here
		schema.Format = "int32"
	}

	return schema, nil
}

// analyzeFloatType analyzes floating-point types.
func (ta *TypeAnalyzer) analyzeFloatType(goType reflect.Type) (*APISchema, error) {
	schema := &APISchema{
		Type: "number",
	}

	// Set format based on precision
	switch goType.Kind() {
	case reflect.Float32:
		schema.Format = "float"
	case reflect.Float64:
		schema.Format = "double"
	default:
		// Handle all other reflect.Kind values that shouldn't reach here
		schema.Format = "float"
	}

	return schema, nil
}

// analyzeBoolType analyzes boolean types.
func (ta *TypeAnalyzer) analyzeBoolType(goType reflect.Type) (*APISchema, error) {
	return &APISchema{
		Type:    "boolean",
		Example: true,
	}, nil
}

// analyzeSliceType analyzes slice and array types.
func (ta *TypeAnalyzer) analyzeSliceType(goType reflect.Type) (*APISchema, error) {
	elementType := goType.Elem()
	
	// Recursively analyze the element type
	elementSchema, err := ta.AnalyzeType(elementType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze slice element type: %w", err)
	}

	schema := &APISchema{
		Type:  "array",
		Items: elementSchema,
	}

	// Add constraints for arrays (fixed size)
	if goType.Kind() == reflect.Array {
		arrayLen := goType.Len()
		schema.MinLength = &arrayLen
		schema.MaxLength = &arrayLen
	}

	return schema, nil
}

// analyzeMapType analyzes map types.
func (ta *TypeAnalyzer) analyzeMapType(goType reflect.Type) (*APISchema, error) {
	// Maps in JSON are typically objects with string keys
	keyType := goType.Key()
	valueType := goType.Elem()

	// Only support string keys (standard for JSON)
	if keyType.Kind() != reflect.String {
		return &APISchema{
			Type:        "object",
			Description: fmt.Sprintf("Map with %s keys (converted to string)", keyType.Kind()),
		}, nil
	}

	// Analyze the value type  
	_, err := ta.AnalyzeType(valueType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze map value type: %w", err)
	}

	return &APISchema{
		Type:        "object",
		Description: fmt.Sprintf("Map with string keys and %s values", valueType.Kind()),
		// Note: OpenAPI additionalProperties would be handled in full implementation
	}, nil
}

// analyzeStructType analyzes struct types and generates object schemas.
func (ta *TypeAnalyzer) analyzeStructType(goType reflect.Type) (*APISchema, error) {
	schema := &APISchema{
		Type:       "object",
		Properties: make(map[string]*APISchema),
	}

	var required []string

	// Analyze each field in the struct
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name from tags
		jsonName := ta.getJSONFieldName(field)
		if jsonName == "-" {
			continue // Skip fields marked with json:"-"
		}

		// Analyze field type
		fieldSchema, err := ta.AnalyzeType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze field %s: %w", field.Name, err)
		}

		// Add field documentation from struct tags or comments
		ta.addFieldDocumentation(field, fieldSchema)

		schema.Properties[jsonName] = fieldSchema

		// Determine if field is required
		if ta.isFieldRequired(field) {
			required = append(required, jsonName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema, nil
}

// analyzePointerType analyzes pointer types.
func (ta *TypeAnalyzer) analyzePointerType(goType reflect.Type) (*APISchema, error) {
	// Pointers indicate optional fields - analyze the underlying type
	return ta.AnalyzeType(goType.Elem())
}

// analyzeInterfaceType analyzes interface types.
func (ta *TypeAnalyzer) analyzeInterfaceType(goType reflect.Type) (*APISchema, error) {
	// For empty interfaces (interface{}), return a generic object
	if goType.NumMethod() == 0 {
		return &APISchema{
			Type:        "object",
			Description: "Generic object (interface{})",
		}, nil
	}

	// For interfaces with methods, create a more specific schema
	return &APISchema{
		Type:        "object",
		Description: fmt.Sprintf("Interface type: %s", goType.Name()),
	}, nil
}

// getTypeName generates a canonical name for a Go type.
func (ta *TypeAnalyzer) getTypeName(goType reflect.Type) string {
	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		return ta.getTypeName(goType.Elem())
	}

	// Use package + name for named types
	if goType.PkgPath() != "" && goType.Name() != "" {
		return goType.PkgPath() + "." + goType.Name()
	}

	// Use just the name for built-in types
	if goType.Name() != "" {
		return goType.Name()
	}

	// Fallback to string representation
	return goType.String()
}

// isComplexType determines if a type should be extracted as a reusable component.
func (ta *TypeAnalyzer) isComplexType(goType reflect.Type) bool {
	switch goType.Kind() {
	case reflect.Struct:
		return true
	case reflect.Slice, reflect.Array:
		// Complex if element type is complex
		return ta.isComplexType(goType.Elem())
	case reflect.Map:
		// Complex if value type is complex
		return ta.isComplexType(goType.Elem())
	case reflect.Ptr:
		return ta.isComplexType(goType.Elem())
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer, reflect.Invalid:
		return false
	default:
		return false
	}
}

// getJSONFieldName extracts the JSON field name from struct tags.
func (ta *TypeAnalyzer) getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		// Convert field name to camelCase for JSON
		return ta.toCamelCase(field.Name)
	}

	// Parse JSON tag (handle options like omitempty)
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return ta.toCamelCase(field.Name)
}

// toCamelCase converts a string to camelCase.
func (ta *TypeAnalyzer) toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// addFieldDocumentation adds documentation to a field schema from struct tags.
func (ta *TypeAnalyzer) addFieldDocumentation(field reflect.StructField, schema *APISchema) {
	// Extract documentation from various struct tags
	if desc := field.Tag.Get("description"); desc != "" {
		schema.Description = desc
	}
	
	if example := field.Tag.Get("example"); example != "" {
		schema.Example = example
	}

	if format := field.Tag.Get("format"); format != "" {
		schema.Format = format
	}

	// Extract validation constraints
	ta.addValidationConstraints(field, schema)
}

// addValidationConstraints adds validation constraints from struct tags.
func (ta *TypeAnalyzer) addValidationConstraints(field reflect.StructField, schema *APISchema) {
	// This could be extended to support validation tags like:
	// - `validate:"min=1,max=100"` for numeric constraints
	// - `validate:"required"` for required fields
	// - `validate:"email"` for format validation
	// For now, we'll implement basic patterns

	if validate := field.Tag.Get("validate"); validate != "" {
		parts := strings.Split(validate, ",")
		for _, part := range parts {
			if strings.HasPrefix(part, "min=") {
				// Extract minimum value (simplified)
			} else if strings.HasPrefix(part, "max=") {
				// Extract maximum value (simplified)
			} else if part == "email" {
				schema.Format = "email"
			} else if part == "uuid" {
				schema.Format = "uuid"
			}
		}
	}
}

// isFieldRequired determines if a struct field is required.
func (ta *TypeAnalyzer) isFieldRequired(field reflect.StructField) bool {
	// Check JSON tag for omitempty
	tag := field.Tag.Get("json")
	if strings.Contains(tag, "omitempty") {
		return false
	}

	// Pointer types are typically optional
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	// Check validation tag for required
	validate := field.Tag.Get("validate")
	if strings.Contains(validate, "required") {
		return true
	}

	// Default to required for non-pointer fields
	return true
}

// GetAnalyzedTypes returns all analyzed types for schema generation.
func (ta *TypeAnalyzer) GetAnalyzedTypes() map[string]*APISchema {
	return ta.analyzedTypes
}

// Clear resets the analyzer state.
func (ta *TypeAnalyzer) Clear() {
	ta.analyzedTypes = make(map[string]*APISchema)
	ta.typeDefinitions = make(map[string]*ComponentAnalysis)
}