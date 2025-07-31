// Package mockdata provides intelligent mock data generation with faker integration.
//
// This package implements context-aware mock data generation that analyzes parameter names
// and types to produce realistic test data. It uses semantic pattern matching to generate
// appropriate mock values (e.g., "email" parameters get valid email addresses, "age" gets
// reasonable age values). The design prioritizes safety through bounded resource usage,
// predictable execution paths, and comprehensive validation.
package mockdata

import (
	"context"
	"fmt"
	mathrand "math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/go-faker/faker/v4"

	"github.com/conneroisu/templar/internal/types"
)

// IntelligentMockGenerator provides advanced mock data generation with faker integration.
//
// Design rationale: Uses composition over inheritance to combine multiple generation
// strategies. The generator maintains its own RNG instance to ensure deterministic
// behavior when given the same seed, which is critical for reproducible testing.
// Pattern matching uses priority-based selection to resolve conflicts (e.g., "userAge"
// could match both "user" and "age" patterns - priority determines the winner).
type IntelligentMockGenerator struct {
	rng             *mathrand.Rand                // Deterministic random source for reproducible generation
	patterns        map[string]*PatternDefinition // Semantic pattern registry with O(1) lookup
	templateManager TemplateManager               // Template inheritance and composition system
	composer        DataComposer                  // Complex nested structure generation
	validator       MockDataValidator             // Generated data validation and safety checks
	cache           MockDataCache                 // Performance optimization with TTL support
	config          *MockDataConfig               // Generation behavior configuration
}

// PatternDefinition defines a semantic pattern for parameter recognition.
//
// Design rationale: Combines regex and keyword matching for flexibility while maintaining
// performance. Priority field resolves conflicts when multiple patterns match the same
// parameter name. Function pointers enable pattern-specific generation logic while
// keeping the core matching algorithm simple and fast.
type PatternDefinition struct {
	Name        string                                                           // Unique pattern identifier
	Regex       *regexp.Regexp                                                   // Compiled regex for parameter name matching
	Keywords    []string                                                         // Simple string contains matching (faster than regex)
	Generator   func(*IntelligentMockGenerator, types.ParameterInfo) interface{} // Pattern-specific generation function
	Priority    int                                                              // Conflict resolution priority (higher wins)
	Description string                                                           // Human-readable pattern description
}

// NewIntelligentMockGenerator creates a new intelligent mock generator.
//
// Design decisions:
// - Uses math/rand instead of crypto/rand for performance (mock data doesn't need cryptographic security)
// - Seeds faker library to ensure deterministic behavior across generator instances
// - Initializes all subsystems during construction to fail fast on configuration errors
// - Pattern registration happens during construction to amortize setup cost.
func NewIntelligentMockGenerator(config *MockDataConfig) *IntelligentMockGenerator {
	if config == nil {
		config = DefaultMockDataConfig()
	}

	// Use provided seed or generate one based on current time for reproducibility
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	// Create deterministic random source - critical for reproducible testing
	rng := mathrand.New(mathrand.NewSource(seed))

	// Set faker random source
	faker.SetRandomSource(rng)

	generator := &IntelligentMockGenerator{
		rng:             rng,
		patterns:        make(map[string]*PatternDefinition),
		templateManager: NewFileTemplateManager(),
		composer:        NewAdvancedDataComposer(),
		validator:       NewMockDataValidator(),
		cache:           NewMemoryMockDataCache(),
		config:          config,
	}

	// Register intelligent patterns
	generator.registerIntelligentPatterns()

	return generator
}

// DefaultMockDataConfig returns a default configuration.
func DefaultMockDataConfig() *MockDataConfig {
	return &MockDataConfig{
		DefaultTemplate: "default",
		Templates:       []*MockDataTemplate{},
		Patterns:        make(map[string]string),
		Seed:            0,
		Locale:          "en",
		Options: &GenerationOptions{
			UseRealisticData:  true,
			IncludeNullValues: false,
			StringLength:      &LengthRange{Min: 5, Max: 50},
			ArrayLength:       &LengthRange{Min: 1, Max: 5},
			CacheResults:      true,
			ValidationLevel:   "basic",
		},
	}
}

// GenerateForComponent generates intelligent mock data for all component parameters.
func (g *IntelligentMockGenerator) GenerateForComponent(
	component *types.ComponentInfo,
) map[string]interface{} {
	ctx := context.Background()
	result, err := g.GenerateWithContext(ctx, component, g.config.Options)
	if err != nil {
		// Fallback to basic generation on error
		return g.generateBasicMockData(component)
	}

	return result
}

// GenerateWithContext generates mock data with comprehensive context support.
func (g *IntelligentMockGenerator) GenerateWithContext(
	ctx context.Context,
	component *types.ComponentInfo,
	options *GenerationOptions,
) (map[string]interface{}, error) {
	if options == nil {
		options = g.config.Options
	}

	genCtx := &GenerationContext{
		ComponentInfo: component,
		Depth:         0,
		Path:          []string{},
		Config:        g.config,
		Cache:         make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
	}

	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		cacheKey := g.generateCacheKey(component.Name, param.Name, param.Type)

		// Check cache first
		if options.CacheResults {
			if cached, found := g.cache.Get(cacheKey); found {
				mockData[param.Name] = cached

				continue
			}
		}

		value := g.generateIntelligentParameter(param, genCtx)
		mockData[param.Name] = value

		// Cache the result
		if options.CacheResults {
			g.cache.Set(cacheKey, value, time.Hour)
		}
	}

	// Validate generated data
	if err := g.validator.ValidateData(mockData, component); err != nil {
		return nil, err
	}

	return mockData, nil
}

// generateIntelligentParameter generates mock data for a parameter using intelligent pattern matching.
//
// Design rationale: Implements a three-tier generation strategy for maximum flexibility:
// 1. Default values take precedence (respect user-defined defaults)
// 2. Semantic pattern matching uses priority-based selection (highest priority wins)
// 3. Type-based generation as fallback (ensures something is always generated)
//
// The priority system resolves conflicts when multiple patterns match the same parameter.
// For example, "userAge" matches both "user" (priority 90) and "age" (priority 95),
// so the age pattern wins and generates a realistic age value instead of a generic name.
func (g *IntelligentMockGenerator) generateIntelligentParameter(
	param types.ParameterInfo,
	ctx *GenerationContext,
) interface{} {
	// Use default value if available - respects user-defined parameter defaults
	if param.Default != nil {
		return param.Default
	}

	// Try intelligent pattern matching with priority-based conflict resolution
	bestMatch := ""
	highestPriority := -1

	for patternName, pattern := range g.patterns {
		if g.matchesPattern(param, pattern) {
			if pattern.Priority > highestPriority {
				highestPriority = pattern.Priority
				bestMatch = patternName
			}
		}
	}

	// Generate using best matching pattern - stores pattern metadata for debugging
	if bestMatch != "" {
		pattern := g.patterns[bestMatch]
		ctx.Metadata["matched_pattern"] = bestMatch // Enable pattern analysis in tests

		return pattern.Generator(g, param)
	}

	// Fallback to type-based generation with faker - ensures robustness
	return g.generateByTypeWithFaker(param.Type, param.Name)
}

// matchesPattern checks if a parameter matches a semantic pattern.
func (g *IntelligentMockGenerator) matchesPattern(
	param types.ParameterInfo,
	pattern *PatternDefinition,
) bool {
	paramName := strings.ToLower(param.Name)

	// Check regex pattern
	if pattern.Regex != nil && pattern.Regex.MatchString(paramName) {
		return true
	}

	// Check keywords
	for _, keyword := range pattern.Keywords {
		if strings.Contains(paramName, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// registerIntelligentPatterns registers comprehensive semantic patterns for parameter recognition.
//
// Design decisions:
// - Patterns are organized by domain (Personal, Geographic, Business, etc.) for maintainability
// - Each pattern uses both regex and keyword matching for maximum coverage
// - Priority values are carefully tuned to resolve common naming conflicts:
//   - Specific patterns (email, uuid) get highest priority (95-100)
//   - General patterns (name, description) get lower priority (70-90)
//   - This ensures "userEmail" matches "email" pattern, not "user" pattern
//
// - Generator functions use method pointers for type safety and performance.
func (g *IntelligentMockGenerator) registerIntelligentPatterns() {
	patterns := []*PatternDefinition{
		// Personal Information (High Priority)
		{
			Name:        "email",
			Regex:       regexp.MustCompile(`(?i)(email|mail|e-mail)`),
			Keywords:    []string{"email", "mail", "e-mail"},
			Generator:   (*IntelligentMockGenerator).generateRealisticEmail,
			Priority:    100,
			Description: "Email addresses",
		},
		{
			Name:        "first_name",
			Regex:       regexp.MustCompile(`(?i)(first_?name|fname|given_?name)`),
			Keywords:    []string{"firstname", "fname", "givenname"},
			Generator:   (*IntelligentMockGenerator).generateFirstName,
			Priority:    95,
			Description: "First names",
		},
		{
			Name:        "last_name",
			Regex:       regexp.MustCompile(`(?i)(last_?name|lname|sur_?name|family_?name)`),
			Keywords:    []string{"lastname", "lname", "surname", "familyname"},
			Generator:   (*IntelligentMockGenerator).generateLastName,
			Priority:    95,
			Description: "Last names",
		},
		{
			Name:        "full_name",
			Regex:       regexp.MustCompile(`(?i)(full_?name|name|person|author|user)`),
			Keywords:    []string{"name", "author", "person", "user"},
			Generator:   (*IntelligentMockGenerator).generateFullName,
			Priority:    90,
			Description: "Full names",
		},
		{
			Name:        "phone",
			Regex:       regexp.MustCompile(`(?i)(phone|tel|mobile|cell)`),
			Keywords:    []string{"phone", "tel", "mobile", "cell", "telephone"},
			Generator:   (*IntelligentMockGenerator).generatePhoneNumber,
			Priority:    90,
			Description: "Phone numbers",
		},

		// Geographic Information
		{
			Name:        "address",
			Regex:       regexp.MustCompile(`(?i)(address|street|addr)`),
			Keywords:    []string{"address", "street", "addr"},
			Generator:   (*IntelligentMockGenerator).generateAddress,
			Priority:    85,
			Description: "Street addresses",
		},
		{
			Name:        "city",
			Regex:       regexp.MustCompile(`(?i)(city|town)`),
			Keywords:    []string{"city", "town"},
			Generator:   (*IntelligentMockGenerator).generateCity,
			Priority:    85,
			Description: "City names",
		},
		{
			Name:        "country",
			Regex:       regexp.MustCompile(`(?i)(country|nation)`),
			Keywords:    []string{"country", "nation"},
			Generator:   (*IntelligentMockGenerator).generateCountry,
			Priority:    85,
			Description: "Country names",
		},

		// Business Information
		{
			Name:        "company",
			Regex:       regexp.MustCompile(`(?i)(company|corp|organization|org|business)`),
			Keywords:    []string{"company", "corp", "organization", "business"},
			Generator:   (*IntelligentMockGenerator).generateCompanyName,
			Priority:    80,
			Description: "Company names",
		},
		{
			Name:        "job_title",
			Regex:       regexp.MustCompile(`(?i)(job_?title|position|role|occupation)`),
			Keywords:    []string{"job", "title", "position", "role", "occupation"},
			Generator:   (*IntelligentMockGenerator).generateJobTitle,
			Priority:    80,
			Description: "Job titles",
		},

		// URLs and Media
		{
			Name:        "url",
			Regex:       regexp.MustCompile(`(?i)(url|link|href|website)`),
			Keywords:    []string{"url", "link", "href", "website"},
			Generator:   (*IntelligentMockGenerator).generateURL,
			Priority:    85,
			Description: "URLs and links",
		},
		{
			Name:        "image_url",
			Regex:       regexp.MustCompile(`(?i)(image|img|picture|photo|avatar)`),
			Keywords:    []string{"image", "img", "picture", "photo", "avatar"},
			Generator:   (*IntelligentMockGenerator).generateImageURL,
			Priority:    90,
			Description: "Image URLs",
		},

		// Content and Text
		{
			Name:        "title",
			Regex:       regexp.MustCompile(`(?i)(title|heading|header|subject)`),
			Keywords:    []string{"title", "heading", "header", "subject"},
			Generator:   (*IntelligentMockGenerator).generateTitle,
			Priority:    75,
			Description: "Titles and headings",
		},
		{
			Name:        "description",
			Regex:       regexp.MustCompile(`(?i)(description|desc|summary|about)`),
			Keywords:    []string{"description", "desc", "summary", "about"},
			Generator:   (*IntelligentMockGenerator).generateDescription,
			Priority:    75,
			Description: "Descriptions and summaries",
		},
		{
			Name:        "content",
			Regex:       regexp.MustCompile(`(?i)(content|text|body|message)`),
			Keywords:    []string{"content", "text", "body", "message"},
			Generator:   (*IntelligentMockGenerator).generateContent,
			Priority:    70,
			Description: "Text content",
		},

		// Identifiers and Keys
		{
			Name:        "uuid",
			Regex:       regexp.MustCompile(`(?i)(uuid|guid)`),
			Keywords:    []string{"uuid", "guid"},
			Generator:   (*IntelligentMockGenerator).generateUUID,
			Priority:    95,
			Description: "UUIDs and GUIDs",
		},
		{
			Name:        "id",
			Regex:       regexp.MustCompile(`(?i)(id|key|identifier)$`),
			Keywords:    []string{"id", "key", "identifier"},
			Generator:   (*IntelligentMockGenerator).generateID,
			Priority:    80,
			Description: "IDs and keys",
		},

		// Dates and Times
		{
			Name:        "date",
			Regex:       regexp.MustCompile(`(?i)(date|time|created|updated|modified|timestamp)`),
			Keywords:    []string{"date", "time", "created", "updated", "modified", "timestamp"},
			Generator:   (*IntelligentMockGenerator).generateDate,
			Priority:    80,
			Description: "Dates and timestamps",
		},
		{
			Name:        "birth_date",
			Regex:       regexp.MustCompile(`(?i)(birth|born|dob|birthday)`),
			Keywords:    []string{"birth", "born", "dob", "birthday"},
			Generator:   (*IntelligentMockGenerator).generateBirthDate,
			Priority:    85,
			Description: "Birth dates",
		},

		// Financial Information
		{
			Name:        "price",
			Regex:       regexp.MustCompile(`(?i)(price|cost|amount|fee|charge)`),
			Keywords:    []string{"price", "cost", "amount", "fee", "charge"},
			Generator:   (*IntelligentMockGenerator).generatePrice,
			Priority:    80,
			Description: "Prices and amounts",
		},
		{
			Name:        "currency",
			Regex:       regexp.MustCompile(`(?i)(currency|money|dollar|euro|pound)`),
			Keywords:    []string{"currency", "money", "dollar", "euro", "pound"},
			Generator:   (*IntelligentMockGenerator).generateCurrency,
			Priority:    80,
			Description: "Currency codes",
		},

		// Numeric Values
		{
			Name:        "age",
			Regex:       regexp.MustCompile(`(?i)(age|years_?old)`),
			Keywords:    []string{"age", "yearsold"},
			Generator:   (*IntelligentMockGenerator).generateAge,
			Priority:    95, // Higher priority than full_name pattern
			Description: "Age values",
		},
		{
			Name:        "count",
			Regex:       regexp.MustCompile(`(?i)(count|quantity|number|amount)`),
			Keywords:    []string{"count", "quantity", "number", "amount"},
			Generator:   (*IntelligentMockGenerator).generateCount,
			Priority:    70,
			Description: "Count and quantity values",
		},

		// Boolean Values
		{
			Name:        "status_flag",
			Regex:       regexp.MustCompile(`(?i)(active|enabled|visible|public|featured|selected|is_|has_)`),
			Keywords:    []string{"active", "enabled", "visible", "public", "featured", "selected"},
			Generator:   (*IntelligentMockGenerator).generateBoolean,
			Priority:    75,
			Description: "Status flags and boolean values",
		},

		// Colors and Design
		{
			Name:        "color",
			Regex:       regexp.MustCompile(`(?i)(color|colour|background|theme)`),
			Keywords:    []string{"color", "colour", "background", "theme"},
			Generator:   (*IntelligentMockGenerator).generateColor,
			Priority:    75,
			Description: "Colors and hex codes",
		},
	}

	for _, pattern := range patterns {
		g.patterns[pattern.Name] = pattern
	}
}

// Faker-based generators for realistic data
//
// Design rationale: These generators use the faker library for realistic data while maintaining
// deterministic behavior through the seeded RNG. Each generator focuses on a specific semantic
// pattern and leverages faker's domain-specific generation capabilities.

// generateRealisticEmail generates realistic email addresses using faker.
// Uses faker.Email() which produces properly formatted addresses with realistic domains.
func (g *IntelligentMockGenerator) generateRealisticEmail(_ types.ParameterInfo) interface{} {
	return faker.Email()
}

func (g *IntelligentMockGenerator) generateFirstName(_ types.ParameterInfo) interface{} {
	return faker.FirstName()
}

func (g *IntelligentMockGenerator) generateLastName(_ types.ParameterInfo) interface{} {
	return faker.LastName()
}

func (g *IntelligentMockGenerator) generateFullName(_ types.ParameterInfo) interface{} {
	return faker.Name()
}

func (g *IntelligentMockGenerator) generatePhoneNumber(_ types.ParameterInfo) interface{} {
	return faker.Phonenumber()
}

// generateAddress creates realistic street addresses using predefined components.
//
// Implementation note: Uses simple concatenation rather than faker.Address() because
// faker v4 has limited address generation options. This approach provides consistent
// formatting while maintaining realistic appearance for UI previews.
func (g *IntelligentMockGenerator) generateAddress(_ types.ParameterInfo) interface{} {
	// Predefined components for consistent, realistic-looking addresses
	streetNumbers := []string{"123", "456", "789", "101", "202"}
	streetNames := []string{"Main St", "Oak Ave", "Pine Rd", "Elm Dr", "Cedar Ln"}
	streetNum := streetNumbers[g.rng.Intn(len(streetNumbers))]
	streetName := streetNames[g.rng.Intn(len(streetNames))]

	return fmt.Sprintf("%s %s", streetNum, streetName)
}

func (g *IntelligentMockGenerator) generateCity(_ types.ParameterInfo) interface{} {
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia"}

	return cities[g.rng.Intn(len(cities))]
}

func (g *IntelligentMockGenerator) generateCountry(_ types.ParameterInfo) interface{} {
	countries := []string{"United States", "Canada", "United Kingdom", "Germany", "France", "Australia"}

	return countries[g.rng.Intn(len(countries))]
}

func (g *IntelligentMockGenerator) generateCompanyName(_ types.ParameterInfo) interface{} {
	// Faker v4 doesn't have Company directly, use name variation
	return faker.Name() + " Corp"
}

func (g *IntelligentMockGenerator) generateJobTitle(_ types.ParameterInfo) interface{} {
	// Generate a simple job title
	titles := []string{"Developer", "Designer", "Manager", "Analyst", "Engineer", "Consultant"}

	return titles[g.rng.Intn(len(titles))]
}

func (g *IntelligentMockGenerator) generateURL(_ types.ParameterInfo) interface{} {
	return faker.URL()
}

// generateImageURL creates URLs for placeholder images using Lorem Picsum service.
//
// Design decision: Uses picsum.photos for several reasons:
// 1. Provides actual images (not broken links) for realistic UI previews
// 2. Random dimensions (200-600px) simulate real-world image variety
// 3. HTTPS URLs ensure security in modern web applications
// 4. Service is reliable and commonly used in development.
func (g *IntelligentMockGenerator) generateImageURL(_ types.ParameterInfo) interface{} {
	width := 200 + g.rng.Intn(400)  // 200-600px width for realistic variety
	height := 200 + g.rng.Intn(400) // 200-600px height for realistic variety

	return fmt.Sprintf("https://picsum.photos/%d/%d", width, height)
}

func (g *IntelligentMockGenerator) generateTitle(_ types.ParameterInfo) interface{} {
	return faker.Sentence()
}

func (g *IntelligentMockGenerator) generateDescription(_ types.ParameterInfo) interface{} {
	return faker.Paragraph()
}

func (g *IntelligentMockGenerator) generateContent(_ types.ParameterInfo) interface{} {
	return faker.Paragraph()
}

func (g *IntelligentMockGenerator) generateUUID(_ types.ParameterInfo) interface{} {
	return faker.UUIDHyphenated()
}

func (g *IntelligentMockGenerator) generateID(_ types.ParameterInfo) interface{} {
	return faker.UUIDDigit()[:8] // Use first 8 chars of UUID
}

func (g *IntelligentMockGenerator) generateDate(_ types.ParameterInfo) interface{} {
	return faker.Date()
}

func (g *IntelligentMockGenerator) generateBirthDate(_ types.ParameterInfo) interface{} {
	return faker.Date()
}

func (g *IntelligentMockGenerator) generatePrice(_ types.ParameterInfo) interface{} {
	return faker.AmountWithCurrency()
}

func (g *IntelligentMockGenerator) generateCurrency(_ types.ParameterInfo) interface{} {
	return faker.Currency()
}

func (g *IntelligentMockGenerator) generateAge(_ types.ParameterInfo) interface{} {
	return 18 + g.rng.Intn(62) // Age between 18 and 80
}

func (g *IntelligentMockGenerator) generateCount(_ types.ParameterInfo) interface{} {
	return g.rng.Intn(100) + 1
}

func (g *IntelligentMockGenerator) generateBoolean(_ types.ParameterInfo) interface{} {
	return g.rng.Intn(2) == 1
}

func (g *IntelligentMockGenerator) generateColor(_ types.ParameterInfo) interface{} {
	colors := []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7"}

	return colors[g.rng.Intn(len(colors))]
}

// generateByTypeWithFaker generates mock data based on Go types using faker as fallback.
//
// Design rationale: This method serves as the final fallback when semantic pattern matching
// fails. It ensures that every parameter gets a sensible value regardless of naming.
// The type-based approach handles all Go primitive types plus common collections.
//
// Performance consideration: Type switching is more efficient than reflection for
// the common primitive types we handle here.
func (g *IntelligentMockGenerator) generateByTypeWithFaker(paramType, paramName string) interface{} {
	switch strings.ToLower(paramType) {
	case "string":
		return faker.Word()
	case "int", "int32", "int64":
		return g.rng.Intn(1000)
	case "uint", "uint32", "uint64":
		return g.rng.Intn(1000000)
	case "float32", "float64":
		return g.rng.Float64() * 100
	case "bool", "boolean":
		return g.rng.Intn(2) == 1
	case "time.time":
		return faker.Date()
	default:
		// Handle slice types
		if strings.HasPrefix(paramType, "[]") {
			elementType := strings.TrimPrefix(paramType, "[]")
			length := g.config.Options.ArrayLength.Min + g.rng.Intn(
				g.config.Options.ArrayLength.Max-g.config.Options.ArrayLength.Min+1,
			)
			result := make([]interface{}, length)
			for i := range length {
				result[i] = g.generateByTypeWithFaker(elementType, paramName)
			}

			return result
		}

		// Handle map types
		if strings.HasPrefix(paramType, "map[") {
			return map[string]interface{}{
				faker.Word(): faker.Sentence(),
				faker.Word(): faker.Sentence(),
			}
		}

		return faker.Sentence()
	}
}

// generateBasicMockData provides fallback mock data generation.
func (g *IntelligentMockGenerator) generateBasicMockData(
	component *types.ComponentInfo,
) map[string]interface{} {
	mockData := make(map[string]interface{})
	for _, param := range component.Parameters {
		mockData[param.Name] = g.generateByTypeWithFaker(param.Type, param.Name)
	}

	return mockData
}

// generateCacheKey creates a unique cache key for parameter data.
func (g *IntelligentMockGenerator) generateCacheKey(
	componentName, paramName, paramType string,
) string {
	return componentName + ":" + paramName + ":" + paramType
}

// Implement remaining interface methods.
func (g *IntelligentMockGenerator) GenerateForParameter(param types.ParameterInfo) interface{} {
	ctx := &GenerationContext{
		Depth:    0,
		Path:     []string{},
		Config:   g.config,
		Cache:    make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}

	return g.generateIntelligentParameter(param, ctx)
}

func (g *IntelligentMockGenerator) ValidateGenerated(
	data map[string]interface{},
	component *types.ComponentInfo,
) error {
	return g.validator.ValidateData(data, component)
}

func (g *IntelligentMockGenerator) GetSupportedPatterns() []string {
	patterns := make([]string, 0, len(g.patterns))
	for name := range g.patterns {
		patterns = append(patterns, name)
	}

	return patterns
}
