// Package mockdata provides advanced data composition for complex mock structures.
//
// This package implements sophisticated data composition capabilities that create
// nested and complex mock data structures from schema definitions. It supports:
// - Array composition with configurable length and element types
// - Object composition with nested properties and relationships
// - Template-based composition using faker expressions
// - Recursive depth limiting to prevent infinite structures
//
// The composer uses a context-driven approach to maintain generation state and
// provides deterministic output based on configuration seeds.
package mockdata

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// Type constants for mock data generation.
	typeString  = "string"
	typeInt     = "int"
	typeBool    = "bool"
	typeBoolean = "boolean"
	typeFloat64 = "float64"
)

// AdvancedDataComposer handles composition of complex nested mock data structures.
//
// Design philosophy: Balances flexibility with safety by supporting arbitrary
// nesting while preventing infinite recursion through depth limiting. Uses
// schema-driven composition to create realistic data structures that match
// real-world application patterns.
//
// Safety measures:
// - Maximum depth limiting prevents stack overflow in recursive structures
// - Context tracking enables debugging and prevents circular references
// - Graceful degradation returns placeholders when limits are reached.
type AdvancedDataComposer struct {
	maxDepth int // Maximum nesting depth to prevent infinite recursion
}

// NewAdvancedDataComposer creates a new data composer with reasonable defaults.
func NewAdvancedDataComposer() DataComposer {
	return &AdvancedDataComposer{
		maxDepth: 5, // Prevent infinite recursion
	}
}

// ComposeData creates complex nested structures based on schema definitions.
//
// Schema format:
//
//	{
//	  "type": "array|object|nested|value",
//	  "properties": {...},         // For objects
//	  "elementType": "string",    // For arrays
//	  "minLength": 1,             // For arrays
//	  "template": "user"          // For template-based generation
//	}
//
// Composition strategy:
// 1. Check depth limits for safety
// 2. Dispatch based on schema type
// 3. Fall back to object composition for flexibility.
func (c *AdvancedDataComposer) ComposeData(
	schema map[string]interface{},
	ctx *GenerationContext,
) (interface{}, error) {
	// Safety check: prevent infinite recursion
	if ctx.Depth >= c.maxDepth {
		return "max_depth_reached", nil
	}

	// Determine structure type and dispatch accordingly
	if structType, exists := schema["type"]; exists {
		switch structType {
		case "array":
			return c.composeArrayFromSchema(schema, ctx)
		case "object":
			return c.composeObjectFromSchema(schema, ctx)
		case "nested":
			return c.composeNestedFromSchema(schema, ctx)
		default:
			return c.composeValueFromSchema(schema, ctx)
		}
	}

	// Default to object composition for maximum flexibility
	return c.ComposeObject(schema, ctx)
}

// ComposeArray creates arrays with appropriate elements.
func (c *AdvancedDataComposer) ComposeArray(
	elementType string,
	length int,
	ctx *GenerationContext,
) ([]interface{}, error) {
	if ctx.Depth >= c.maxDepth {
		return []interface{}{}, nil
	}

	newCtx := c.createChildContext(ctx, "[]")
	result := make([]interface{}, length)

	for i := range length {
		element, err := c.composeElementByType(elementType, newCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to compose array element %d: %w", i, err)
		}
		result[i] = element
	}

	return result, nil
}

// ComposeObject creates objects with appropriate properties.
func (c *AdvancedDataComposer) ComposeObject(
	schema map[string]interface{},
	ctx *GenerationContext,
) (map[string]interface{}, error) {
	if ctx.Depth >= c.maxDepth {
		return map[string]interface{}{}, nil
	}

	newCtx := c.createChildContext(ctx, "object")
	result := make(map[string]interface{})

	for key, value := range schema {
		// Skip metadata keys
		if c.isMetadataKey(key) {
			continue
		}

		childCtx := c.createChildContext(newCtx, key)
		composedValue, err := c.composeValue(value, childCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to compose property %s: %w", key, err)
		}

		result[key] = composedValue
	}

	return result, nil
}

// composeArrayFromSchema creates an array based on schema definition.
func (c *AdvancedDataComposer) composeArrayFromSchema(
	schema map[string]interface{},
	ctx *GenerationContext,
) ([]interface{}, error) {
	// Extract array configuration
	elementType := c.getStringValue(schema, "elementType", "string")
	minLength := c.getIntValue(schema, "minLength", 1)
	maxLength := c.getIntValue(schema, "maxLength", 5)

	// Generate random length within bounds
	length := minLength
	if maxLength > minLength {
		length = minLength + int(ctx.Config.Seed%int64(maxLength-minLength+1))
	}

	return c.ComposeArray(elementType, length, ctx)
}

// composeObjectFromSchema creates an object based on schema definition.
func (c *AdvancedDataComposer) composeObjectFromSchema(
	schema map[string]interface{},
	ctx *GenerationContext,
) (map[string]interface{}, error) {
	// Extract properties schema
	if properties, exists := schema["properties"]; exists {
		if propertiesMap, ok := properties.(map[string]interface{}); ok {
			return c.ComposeObject(propertiesMap, ctx)
		}
	}

	// Fallback to treating entire schema as properties
	return c.ComposeObject(schema, ctx)
}

// composeNestedFromSchema creates nested structures based on schema.
func (c *AdvancedDataComposer) composeNestedFromSchema(
	schema map[string]interface{},
	ctx *GenerationContext,
) (interface{}, error) {
	// Handle nested object with specific structure
	if template, exists := schema["template"]; exists {
		if templateName, ok := template.(string); ok {
			return c.composeFromTemplate(templateName, ctx)
		}
	}

	// Handle custom nested structure
	if structure, exists := schema["structure"]; exists {
		return c.composeValue(structure, ctx)
	}

	return nil, errors.New("invalid nested schema definition")
}

// composeValueFromSchema creates a single value based on schema.
func (c *AdvancedDataComposer) composeValueFromSchema(
	schema map[string]interface{},
	ctx *GenerationContext,
) (interface{}, error) {
	valueType := c.getStringValue(schema, "type", "string")

	return c.composeElementByType(valueType, ctx)
}

// composeValue composes a value based on its type and context.
func (c *AdvancedDataComposer) composeValue(
	value interface{},
	ctx *GenerationContext,
) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		return c.ComposeData(v, ctx)
	case []interface{}:
		return c.composeSlice(v, ctx)
	case string:
		return c.interpolateTemplate(v, ctx)
	default:
		return value, nil
	}
}

// composeSlice handles slice composition.
func (c *AdvancedDataComposer) composeSlice(
	slice []interface{},
	ctx *GenerationContext,
) ([]interface{}, error) {
	result := make([]interface{}, len(slice))
	for i, item := range slice {
		childCtx := c.createChildContext(ctx, fmt.Sprintf("[%d]", i))
		composedItem, err := c.composeValue(item, childCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to compose slice item %d: %w", i, err)
		}
		result[i] = composedItem
	}

	return result, nil
}

// composeElementByType creates an element of a specific type.
func (c *AdvancedDataComposer) composeElementByType(
	elementType string,
	ctx *GenerationContext,
) (interface{}, error) {
	switch strings.ToLower(elementType) {
	case typeString:
		return c.generateString(ctx), nil
	case typeInt, "integer":
		return c.generateInt(ctx), nil
	case "float", "double":
		return c.generateFloat(ctx), nil
	case typeBool, typeBoolean:
		return c.generateBool(ctx), nil
	case "date", "datetime":
		return c.generateDate(ctx), nil
	case "uuid":
		return c.generateUUID(ctx), nil
	case "email":
		return c.generateEmail(ctx), nil
	case "name":
		return c.generateName(ctx), nil
	case "text":
		return c.generateText(ctx), nil
	case "url":
		return c.generateURL(ctx), nil
	default:
		// Try to interpret as custom type or template
		if strings.Contains(elementType, ".") {
			return c.interpolateTemplate("{{"+elementType+"}}", ctx)
		}

		return "mock_" + elementType, nil
	}
}

// composeFromTemplate creates data from a named template.
func (c *AdvancedDataComposer) composeFromTemplate(
	templateName string,
	ctx *GenerationContext,
) (interface{}, error) {
	// This would integrate with the template manager
	// For now, return a placeholder
	return map[string]interface{}{
		"template": templateName,
		"data":     "generated from " + templateName,
	}, nil
}

// interpolateTemplate handles template string interpolation.
func (c *AdvancedDataComposer) interpolateTemplate(
	template string,
	ctx *GenerationContext,
) (interface{}, error) {
	// Handle faker-style templates like {{person.firstName}}
	if strings.Contains(template, "{{") && strings.Contains(template, "}}") {
		return c.processFakerTemplate(template, ctx), nil
	}

	// Return as-is if not a template
	return template, nil
}

// processFakerTemplate processes faker-style template strings.
//
// Template format: {{namespace.function}} (e.g., {{person.name}})
//
// Implementation notes:
// - Handles single template expressions per string
// - Maps faker expressions to internal generators for consistency
// - Returns placeholder for unknown expressions to aid debugging
// - Uses deterministic generation based on context seed.
func (c *AdvancedDataComposer) processFakerTemplate(template string, ctx *GenerationContext) interface{} {
	// Extract template expression using simple string parsing
	start := strings.Index(template, "{{")
	end := strings.Index(template, "}}")
	if start == -1 || end == -1 || end <= start {
		return template // Not a valid template
	}

	expression := template[start+2 : end]
	expression = strings.TrimSpace(expression)

	// Map common faker expressions to internal generators
	// This ensures consistency with the intelligent generator patterns
	switch expression {
	case "person.name", "person.firstName":
		return c.generateName(ctx)
	case "person.lastName":
		return c.generateName(ctx) // Could be more specific in future
	case "internet.email":
		return c.generateEmail(ctx)
	case "internet.url":
		return c.generateURL(ctx)
	case "lorem.sentence":
		return c.generateText(ctx)
	case "lorem.paragraph":
		return c.generateLongText(ctx)
	case "number.int":
		return c.generateInt(ctx)
	case "number.float":
		return c.generateFloat(ctx)
	case "datatype.boolean":
		return c.generateBool(ctx)
	case "time.recent", "time.past", "time.future":
		return c.generateDate(ctx)
	default:
		// Unknown expression, return descriptive placeholder for debugging
		return fmt.Sprintf("[%s]", expression)
	}
}

// Generator methods for different data types
//
// Design philosophy: These generators create deterministic but varied mock data
// using the context seed. They prioritize readability in UI previews over
// cryptographic randomness. Each generator uses simple modulo arithmetic
// for reproducible output.

// generateString creates generic string values from a predefined set.
// Uses seed-based selection for deterministic but varied output.
func (c *AdvancedDataComposer) generateString(ctx *GenerationContext) string {
	words := []string{"sample", "demo", "test", "mock", "example"}
	idx := ctx.Config.Seed % int64(len(words))

	return words[idx]
}

func (c *AdvancedDataComposer) generateInt(ctx *GenerationContext) int {
	return int(ctx.Config.Seed%1000) + 1
}

func (c *AdvancedDataComposer) generateFloat(ctx *GenerationContext) float64 {
	return float64(ctx.Config.Seed%10000) / 100.0
}

func (c *AdvancedDataComposer) generateBool(ctx *GenerationContext) bool {
	return ctx.Config.Seed%2 == 0
}

func (c *AdvancedDataComposer) generateDate(ctx *GenerationContext) string {
	return "2024-01-15T10:30:00Z"
}

func (c *AdvancedDataComposer) generateUUID(ctx *GenerationContext) string {
	return fmt.Sprintf("uuid-%d", ctx.Config.Seed%100000)
}

func (c *AdvancedDataComposer) generateEmail(ctx *GenerationContext) string {
	return fmt.Sprintf("user%d@example.com", ctx.Config.Seed%1000)
}

func (c *AdvancedDataComposer) generateName(ctx *GenerationContext) string {
	names := []string{"John Doe", "Jane Smith", "Alex Johnson", "Taylor Brown"}
	idx := ctx.Config.Seed % int64(len(names))

	return names[idx]
}

func (c *AdvancedDataComposer) generateText(ctx *GenerationContext) string {
	texts := []string{
		"Lorem ipsum dolor sit amet.",
		"Consectetur adipiscing elit.",
		"Sed do eiusmod tempor incididunt.",
		"Ut labore et dolore magna aliqua.",
	}
	idx := ctx.Config.Seed % int64(len(texts))

	return texts[idx]
}

func (c *AdvancedDataComposer) generateLongText(ctx *GenerationContext) string {
	return "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris."
}

func (c *AdvancedDataComposer) generateURL(ctx *GenerationContext) string {
	return fmt.Sprintf("https://example.com/item/%d", ctx.Config.Seed%1000)
}

// Helper methods

// createChildContext creates a new generation context for nested composition.
//
// Context inheritance strategy:
// - Increments depth for recursion tracking
// - Extends path for debugging and error reporting
// - Preserves configuration and cache for consistency
// - Maintains metadata for pattern analysis.
func (c *AdvancedDataComposer) createChildContext(parent *GenerationContext, path string) *GenerationContext {
	// Build new path by appending to parent path
	newPath := make([]string, len(parent.Path)+1)
	copy(newPath, parent.Path)
	newPath[len(parent.Path)] = path

	return &GenerationContext{
		ComponentInfo: parent.ComponentInfo, // Component being generated
		ParentParam:   parent.ParentParam,   // Parent parameter context
		Depth:         parent.Depth + 1,     // Increment depth for safety
		Path:          newPath,              // Extended path for debugging
		Config:        parent.Config,        // Shared configuration
		Cache:         parent.Cache,         // Shared cache for performance
		Metadata:      parent.Metadata,      // Shared metadata for analysis
	}
}

func (c *AdvancedDataComposer) isMetadataKey(key string) bool {
	metadataKeys := []string{"type", "elementType", "minLength", "maxLength", "template", "structure", "properties"}
	for _, metaKey := range metadataKeys {
		if key == metaKey {
			return true
		}
	}

	return false
}

func (c *AdvancedDataComposer) getStringValue(schema map[string]interface{}, key, defaultValue string) string {
	if value, exists := schema[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}

	return defaultValue
}

func (c *AdvancedDataComposer) getIntValue(schema map[string]interface{}, key string, defaultValue int) int {
	if value, exists := schema[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Could parse string to int if needed
		}
	}

	return defaultValue
}
