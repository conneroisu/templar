package mockdata

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// MockGenerator generates intelligent mock data based on parameter names and types
type MockGenerator struct {
	rng *rand.Rand
}

// NewMockGenerator creates a new mock data generator
func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateForComponent generates mock data for all parameters of a component
func (g *MockGenerator) GenerateForComponent(component *types.ComponentInfo) map[string]interface{} {
	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		mockData[param.Name] = g.generateForParameter(param)
	}

	return mockData
}

// generateForParameter generates mock data for a single parameter
func (g *MockGenerator) generateForParameter(param types.ParameterInfo) interface{} {
	// Use default value if available
	if param.Default != nil {
		return param.Default
	}

	// Generate based on parameter name patterns
	mockValue := g.generateByNamePattern(param.Name, param.Type)
	if mockValue != nil {
		return mockValue
	}

	// Generate based on type
	return g.generateByType(param.Type)
}

// generateByNamePattern generates mock data based on parameter name patterns
func (g *MockGenerator) generateByNamePattern(name, paramType string) interface{} {
	nameLower := strings.ToLower(name)

	// Email patterns
	if containsAny(nameLower, []string{"email", "mail"}) {
		return g.generateEmail()
	}

	// Name patterns
	if containsAny(nameLower, []string{"name", "firstname", "lastname", "username", "author"}) {
		return g.generateName(nameLower)
	}

	// URL patterns
	if containsAny(nameLower, []string{"url", "link", "href", "src", "image", "avatar"}) {
		return g.generateURL(nameLower)
	}

	// Text content patterns
	if containsAny(nameLower, []string{"title", "heading", "header"}) {
		return g.generateTitle()
	}

	if containsAny(nameLower, []string{"description", "content", "text", "body", "message"}) {
		return g.generateText(nameLower)
	}

	// ID patterns
	if containsAny(nameLower, []string{"id", "key", "uuid"}) {
		return g.generateID()
	}

	// Date patterns
	if containsAny(nameLower, []string{"date", "time", "created", "updated", "modified"}) {
		return g.generateDate()
	}

	// Number patterns
	if containsAny(nameLower, []string{"age", "count", "number", "price", "amount", "quantity"}) {
		return g.generateNumber(nameLower)
	}

	// Boolean patterns
	if containsAny(nameLower, []string{"active", "enabled", "visible", "public", "featured", "selected"}) {
		return g.generateBoolean()
	}

	// Color patterns
	if containsAny(nameLower, []string{"color", "colour", "background", "theme"}) {
		return g.generateColor()
	}

	return nil
}

// generateByType generates mock data based on Go types
func (g *MockGenerator) generateByType(paramType string) interface{} {
	switch strings.ToLower(paramType) {
	case "string":
		return "Sample text"
	case "int", "int32", "int64":
		return g.rng.Intn(100) + 1
	case "float32", "float64":
		return g.rng.Float64() * 100
	case "bool", "boolean":
		return g.rng.Intn(2) == 1
	case "time.time":
		return time.Now().Format("2006-01-02T15:04:05Z07:00")
	default:
		// Handle slice types
		if strings.HasPrefix(paramType, "[]") {
			elementType := strings.TrimPrefix(paramType, "[]")
			return []interface{}{
				g.generateByType(elementType),
				g.generateByType(elementType),
			}
		}

		// Handle map types
		if strings.HasPrefix(paramType, "map[") {
			return map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}
		}

		return "Mock " + paramType
	}
}

// Specific generators for different data types
func (g *MockGenerator) generateEmail() string {
	domains := []string{"example.com", "test.org", "demo.net", "sample.io"}
	names := []string{"john", "jane", "alex", "taylor", "jordan", "casey"}

	name := names[g.rng.Intn(len(names))]
	domain := domains[g.rng.Intn(len(domains))]

	return fmt.Sprintf("%s@%s", name, domain)
}

func (g *MockGenerator) generateName(context string) string {
	firstNames := []string{"John", "Jane", "Alex", "Taylor", "Jordan", "Casey", "Morgan", "Riley"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}
	usernames := []string{"user123", "developer", "designer", "admin", "guest", "test_user"}

	if strings.Contains(context, "first") {
		return firstNames[g.rng.Intn(len(firstNames))]
	}

	if strings.Contains(context, "last") {
		return lastNames[g.rng.Intn(len(lastNames))]
	}

	if strings.Contains(context, "user") {
		return usernames[g.rng.Intn(len(usernames))]
	}

	if strings.Contains(context, "author") {
		return fmt.Sprintf("%s %s",
			firstNames[g.rng.Intn(len(firstNames))],
			lastNames[g.rng.Intn(len(lastNames))])
	}

	// Default full name
	return fmt.Sprintf("%s %s",
		firstNames[g.rng.Intn(len(firstNames))],
		lastNames[g.rng.Intn(len(lastNames))])
}

func (g *MockGenerator) generateURL(context string) string {
	if strings.Contains(context, "image") || strings.Contains(context, "avatar") {
		sizes := []string{"150x150", "200x200", "300x300", "400x400"}
		size := sizes[g.rng.Intn(len(sizes))]
		return fmt.Sprintf("https://via.placeholder.com/%s", size)
	}

	domains := []string{"example.com", "demo.org", "test.net", "sample.io"}
	paths := []string{"page", "article", "post", "item", "resource"}

	domain := domains[g.rng.Intn(len(domains))]
	path := paths[g.rng.Intn(len(paths))]
	id := g.rng.Intn(1000) + 1

	return fmt.Sprintf("https://%s/%s/%d", domain, path, id)
}

func (g *MockGenerator) generateTitle() string {
	adjectives := []string{"Amazing", "Incredible", "Fantastic", "Outstanding", "Remarkable", "Excellent"}
	nouns := []string{"Component", "Feature", "Design", "Experience", "Solution", "Product"}

	adj := adjectives[g.rng.Intn(len(adjectives))]
	noun := nouns[g.rng.Intn(len(nouns))]

	return fmt.Sprintf("%s %s", adj, noun)
}

func (g *MockGenerator) generateText(context string) string {
	if strings.Contains(context, "description") {
		descriptions := []string{
			"This is a comprehensive description that provides detailed information about the component and its functionality.",
			"A well-crafted component designed to enhance user experience with modern design principles.",
			"Discover the power of this innovative solution that streamlines your workflow and improves productivity.",
			"Experience seamless integration and exceptional performance with this cutting-edge component.",
		}
		return descriptions[g.rng.Intn(len(descriptions))]
	}

	if strings.Contains(context, "content") || strings.Contains(context, "body") {
		contents := []string{
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
			"Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
		}
		return contents[g.rng.Intn(len(contents))]
	}

	if strings.Contains(context, "message") {
		messages := []string{
			"Welcome to our platform!",
			"Thank you for your interest.",
			"We're excited to have you here.",
			"Get started with our amazing features.",
		}
		return messages[g.rng.Intn(len(messages))]
	}

	return "Sample text content"
}

func (g *MockGenerator) generateID() string {
	formats := []string{
		fmt.Sprintf("id_%d", g.rng.Intn(10000)),
		fmt.Sprintf("%08x", g.rng.Uint32()),
		fmt.Sprintf("item-%d", g.rng.Intn(1000)),
	}

	return formats[g.rng.Intn(len(formats))]
}

func (g *MockGenerator) generateDate() string {
	// Generate a date within the last year
	now := time.Now()
	randomDays := g.rng.Intn(365)
	randomDate := now.AddDate(0, 0, -randomDays)

	return randomDate.Format("2006-01-02")
}

func (g *MockGenerator) generateNumber(context string) interface{} {
	if strings.Contains(context, "age") {
		return g.rng.Intn(80) + 18 // Age between 18 and 98
	}

	if strings.Contains(context, "price") || strings.Contains(context, "amount") {
		return float64(g.rng.Intn(10000)) / 100 // Price up to $100.00
	}

	if strings.Contains(context, "count") || strings.Contains(context, "quantity") {
		return g.rng.Intn(100) + 1 // Count between 1 and 100
	}

	return g.rng.Intn(1000) + 1
}

func (g *MockGenerator) generateBoolean() bool {
	return g.rng.Intn(2) == 1
}

func (g *MockGenerator) generateColor() string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7",
		"#DDA0DD", "#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E9",
		"#F8C471", "#82E0AA", "#F1948A", "#AED6F1", "#A9DFBF",
	}

	return colors[g.rng.Intn(len(colors))]
}

// Helper function to check if a string contains any of the given substrings
func containsAny(str string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(str, substring) {
			return true
		}
	}
	return false
}

// AdvancedMockGenerator provides more sophisticated mock data generation
type AdvancedMockGenerator struct {
	*MockGenerator
	patterns map[string]*regexp.Regexp
}

// NewAdvancedMockGenerator creates a new advanced mock generator with regex patterns
func NewAdvancedMockGenerator() *AdvancedMockGenerator {
	patterns := map[string]*regexp.Regexp{
		"phone":      regexp.MustCompile(`(?i)(phone|tel|mobile|cell)`),
		"address":    regexp.MustCompile(`(?i)(address|street|city|zip|postal)`),
		"company":    regexp.MustCompile(`(?i)(company|organization|org|business)`),
		"password":   regexp.MustCompile(`(?i)(password|pass|secret|key)`),
		"percentage": regexp.MustCompile(`(?i)(percent|pct|rate|ratio)`),
		"currency":   regexp.MustCompile(`(?i)(currency|money|dollar|euro|pound)`),
	}

	return &AdvancedMockGenerator{
		MockGenerator: NewMockGenerator(),
		patterns:      patterns,
	}
}

// GenerateForComponent generates advanced mock data with pattern matching
func (g *AdvancedMockGenerator) GenerateForComponent(component *types.ComponentInfo) map[string]interface{} {
	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		mockData[param.Name] = g.generateAdvanced(param)
	}

	return mockData
}

func (g *AdvancedMockGenerator) generateAdvanced(param types.ParameterInfo) interface{} {
	// Try pattern matching first
	for pattern, regex := range g.patterns {
		if regex.MatchString(param.Name) {
			return g.generateByPattern(pattern)
		}
	}

	// Fall back to basic generation
	return g.generateForParameter(param)
}

func (g *AdvancedMockGenerator) generateByPattern(pattern string) interface{} {
	switch pattern {
	case "phone":
		return fmt.Sprintf("+1-%03d-%03d-%04d",
			g.rng.Intn(900)+100, g.rng.Intn(900)+100, g.rng.Intn(10000))
	case "address":
		streets := []string{"Main St", "Oak Ave", "Pine Rd", "Elm Dr", "Cedar Ln"}
		return fmt.Sprintf("%d %s", g.rng.Intn(9999)+1, streets[g.rng.Intn(len(streets))])
	case "company":
		companies := []string{"TechCorp", "InnovateLLC", "FutureSystems", "NextGenSolutions", "DigitalDynamics"}
		return companies[g.rng.Intn(len(companies))]
	case "password":
		return "••••••••" // Masked password
	case "percentage":
		return fmt.Sprintf("%.1f%%", g.rng.Float64()*100)
	case "currency":
		currencies := []string{"USD", "EUR", "GBP", "CAD", "AUD"}
		return currencies[g.rng.Intn(len(currencies))]
	default:
		return "Advanced mock data"
	}
}
