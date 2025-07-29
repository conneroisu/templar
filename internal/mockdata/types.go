// Package mockdata defines types and interfaces for intelligent mock data generation.
//
// This package provides the foundational architecture for the mock data generation system.
// The design emphasizes extensibility through interfaces, type safety through structured
// data types, and performance through configuration-driven behavior.
//
// Key architectural principles:
// - Interface-driven design enables testing and extensibility
// - Structured configuration prevents runtime errors
// - Context passing enables stateful generation
// - Caching interfaces support performance optimization
package mockdata

import (
	"context"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// MockDataTemplate defines a template for generating mock data.
//
// Design rationale: Templates provide a way to define consistent mock data patterns
// that can be reused across components. The inheritance system (via Extends field)
// enables template hierarchies where specialized templates can build upon base
// templates, promoting code reuse and consistency.
//
// YAML/JSON serialization enables templates to be stored as configuration files,
// version controlled, and shared across development teams.
type MockDataTemplate struct {
	Name        string                 `yaml:"name" json:"name"`                           // Unique template identifier
	Description string                 `yaml:"description" json:"description"`             // Human-readable description
	Fields      map[string]interface{} `yaml:"fields" json:"fields"`                       // Field definitions with faker expressions
	Extends     string                 `yaml:"extends,omitempty" json:"extends,omitempty"` // Parent template for inheritance
	Tags        []string               `yaml:"tags,omitempty" json:"tags,omitempty"`       // Categorization tags for discovery
	Version     string                 `yaml:"version,omitempty" json:"version,omitempty"` // Semantic versioning for compatibility
}

// MockDataConfig configures mock data generation behavior.
//
// Configuration philosophy: Centralizes all generation behavior settings to enable
// consistent mock data across an entire project. The seed-based approach ensures
// reproducible generation for testing, while locale support enables internationalized
// mock data for global applications.
//
// Pattern customization allows projects to override default generation patterns
// for domain-specific requirements (e.g., custom ID formats, specialized naming).
type MockDataConfig struct {
	DefaultTemplate string              `yaml:"default_template" json:"default_template"` // Template to use when none specified
	Templates       []*MockDataTemplate `yaml:"templates" json:"templates"`               // Available templates for generation
	Patterns        map[string]string   `yaml:"patterns" json:"patterns"`                 // Custom pattern overrides
	Seed            int64               `yaml:"seed" json:"seed"`                         // Random seed for deterministic generation
	Locale          string              `yaml:"locale" json:"locale"`                     // Locale for internationalized data
	Options         *GenerationOptions  `yaml:"options" json:"options"`                   // Generation behavior options
}

// GenerationOptions controls various aspects of mock data generation.
//
// Design philosophy: Fine-grained control over generation behavior enables
// customization for different use cases:
// - Development: Realistic data with caching for performance
// - Testing: Deterministic data with validation for reliability
// - Demos: Attractive data with controlled ranges for presentation
//
// Range specifications prevent edge cases and ensure generated data fits
// UI constraints and business rules.
type GenerationOptions struct {
	UseRealisticData  bool         `yaml:"use_realistic_data" json:"use_realistic_data"`           // Enable faker integration vs simple placeholders
	IncludeNullValues bool         `yaml:"include_null_values" json:"include_null_values"`         // Allow null values for optional fields
	StringLength      *LengthRange `yaml:"string_length,omitempty" json:"string_length,omitempty"` // Control string length bounds
	ArrayLength       *LengthRange `yaml:"array_length,omitempty" json:"array_length,omitempty"`   // Control array size bounds
	DateRange         *DateRange   `yaml:"date_range,omitempty" json:"date_range,omitempty"`       // Control date generation bounds
	NumberRange       *NumberRange `yaml:"number_range,omitempty" json:"number_range,omitempty"`   // Control numeric value bounds
	CacheResults      bool         `yaml:"cache_results" json:"cache_results"`                     // Enable caching for performance
	ValidationLevel   string       `yaml:"validation_level" json:"validation_level"`               // Validation strictness level
}

// LengthRange defines min/max length constraints.
type LengthRange struct {
	Min int `yaml:"min" json:"min"`
	Max int `yaml:"max" json:"max"`
}

// DateRange defines date constraints for mock generation.
type DateRange struct {
	Start *time.Time `yaml:"start,omitempty" json:"start,omitempty"`
	End   *time.Time `yaml:"end,omitempty" json:"end,omitempty"`
}

// NumberRange defines numeric constraints.
type NumberRange struct {
	Min float64 `yaml:"min" json:"min"`
	Max float64 `yaml:"max" json:"max"`
}

// GenerationContext provides context for mock data generation.
//
// Context design: Carries generation state through the composition hierarchy,
// enabling context-aware decisions and preventing infinite recursion. The path
// tracking enables detailed error reporting and debugging.
//
// Performance considerations:
// - Cache enables memoization of expensive operations
// - Metadata tracks generation decisions for analysis and debugging
// - Depth tracking prevents stack overflow in recursive structures
type GenerationContext struct {
	ComponentInfo *types.ComponentInfo   // Component being generated for
	ParentParam   *types.ParameterInfo   // Parent parameter context (for nested generation)
	Depth         int                    // Current nesting depth (for recursion control)
	Path          []string               // Generation path for debugging and error reporting
	Config        *MockDataConfig        // Generation configuration
	Cache         map[string]interface{} // Per-generation cache for performance
	Metadata      map[string]interface{} // Generation metadata for analysis
}

// MockDataGenerator interface defines the contract for mock data generators.
//
// Interface design: Provides multiple generation methods to support different use cases:
// - Component-level generation for complete mock objects
// - Parameter-level generation for individual values
// - Context-aware generation for complex scenarios
// - Validation integration for quality assurance
type MockDataGenerator interface {
	// GenerateForComponent generates mock data for all parameters of a component
	GenerateForComponent(component *types.ComponentInfo) map[string]interface{}

	// GenerateForParameter generates mock data for a single parameter
	GenerateForParameter(param types.ParameterInfo) interface{}

	// GenerateWithContext generates mock data with additional context and options
	GenerateWithContext(ctx context.Context, component *types.ComponentInfo, options *GenerationOptions) (map[string]interface{}, error)

	// ValidateGenerated validates generated mock data against component requirements
	ValidateGenerated(data map[string]interface{}, component *types.ComponentInfo) error

	// GetSupportedPatterns returns semantic patterns this generator recognizes
	GetSupportedPatterns() []string
}

// TemplateManager manages mock data templates and inheritance.
type TemplateManager interface {
	// LoadTemplate loads a template by name
	LoadTemplate(name string) (*MockDataTemplate, error)

	// SaveTemplate saves a template
	SaveTemplate(template *MockDataTemplate) error

	// ResolveTemplate resolves template inheritance and returns final template
	ResolveTemplate(name string) (*MockDataTemplate, error)

	// ListTemplates returns all available templates
	ListTemplates() ([]*MockDataTemplate, error)

	// DeleteTemplate removes a template
	DeleteTemplate(name string) error
}

// PatternMatcher interface for intelligent pattern recognition.
type PatternMatcher interface {
	// MatchPattern determines if a parameter matches a semantic pattern
	MatchPattern(param types.ParameterInfo) (string, float64)

	// RegisterPattern registers a new pattern
	RegisterPattern(name string, matcher func(types.ParameterInfo) float64) error

	// GetPatterns returns all registered patterns
	GetPatterns() map[string]func(types.ParameterInfo) float64
}

// DataComposer handles composition of complex mock data structures.
type DataComposer interface {
	// ComposeData creates complex nested structures
	ComposeData(schema map[string]interface{}, ctx *GenerationContext) (interface{}, error)

	// ComposeArray creates arrays with appropriate elements
	ComposeArray(elementType string, length int, ctx *GenerationContext) ([]interface{}, error)

	// ComposeObject creates objects with appropriate properties
	ComposeObject(schema map[string]interface{}, ctx *GenerationContext) (map[string]interface{}, error)
}

// MockDataValidator validates generated mock data.
type MockDataValidator interface {
	// ValidateData validates mock data against component requirements
	ValidateData(data map[string]interface{}, component *types.ComponentInfo) error

	// ValidateTemplate validates template structure and content
	ValidateTemplate(template *MockDataTemplate) error

	// ValidateConfig validates mock data configuration
	ValidateConfig(config *MockDataConfig) error
}

// GenerationResult contains the result of mock data generation.
type GenerationResult struct {
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Errors    []error                `json:"errors,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
	Generated time.Time              `json:"generated"`
	Template  string                 `json:"template,omitempty"`
	Seed      int64                  `json:"seed"`
}

// MockDataCache interface for caching generated data.
type MockDataCache interface {
	// Get retrieves cached mock data
	Get(key string) (interface{}, bool)

	// Set stores mock data in cache
	Set(key string, value interface{}, ttl time.Duration)

	// Clear removes all cached data
	Clear()

	// Size returns current cache size
	Size() int
}
