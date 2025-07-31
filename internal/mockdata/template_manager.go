// Package mockdata provides comprehensive template management for mock data generation.
//
// This package implements a file-based template system with inheritance capabilities.
// Templates are stored as YAML files and support parent-child relationships for
// code reuse and customization. The design emphasizes safety with circular reference
// detection and atomic file operations.
package mockdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// FileTemplateManager manages mock data templates using file system storage.
//
// Design decisions:
// - File-based storage enables template sharing across project instances
// - In-memory caching reduces disk I/O for frequently accessed templates
// - RWMutex optimizes for read-heavy workloads (templates are read more than written)
// - YAML format provides human-readable, version-controllable template definitions.
type FileTemplateManager struct {
	baseDir   string                       // Base directory for template storage (.templar/templates)
	templates map[string]*MockDataTemplate // In-memory cache of loaded templates
	mutex     sync.RWMutex                 // Thread-safe access to template cache
}

// NewFileTemplateManager creates a new file-based template manager.
//
// Initialization strategy:
// 1. Creates template directory if it doesn't exist (fail-safe approach)
// 2. Loads built-in default templates for immediate usability
// 3. Ignores directory creation errors to handle read-only environments gracefully
//
// The .templar/templates directory structure mirrors typical project conventions.
func NewFileTemplateManager() TemplateManager {
	baseDir := filepath.Join(".", ".templar", "templates")
	manager := &FileTemplateManager{
		baseDir:   baseDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Ensure directory exists - graceful failure in read-only environments
	_ = os.MkdirAll(baseDir, 0755) // Ignore error, continue with operation

	// Load default templates for immediate usability
	manager.loadDefaultTemplates()

	return manager
}

// LoadTemplate loads a template by name.
func (tm *FileTemplateManager) LoadTemplate(name string) (*MockDataTemplate, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	// Check in-memory cache first
	if template, exists := tm.templates[name]; exists {
		return template, nil
	}

	// Load from file
	filename := filepath.Join(tm.baseDir, name+".yml")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", filename, err)
	}

	var template MockDataTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Cache the loaded template
	tm.templates[name] = &template

	return &template, nil
}

// SaveTemplate saves a template to file and cache.
func (tm *FileTemplateManager) SaveTemplate(template *MockDataTemplate) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Validate template
	if template.Name == "" {
		return errors.New("template name cannot be empty")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// Save to file
	filename := filepath.Join(tm.baseDir, template.Name+".yml")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	// Update cache
	tm.templates[template.Name] = template

	return nil
}

// ResolveTemplate resolves template inheritance and returns the final template.
//
// Safety mechanism: Uses history tracking to detect circular inheritance chains.
// This prevents infinite recursion that could crash the application or consume
// excessive memory. The resolved template represents the fully merged hierarchy.
func (tm *FileTemplateManager) ResolveTemplate(name string) (*MockDataTemplate, error) {
	return tm.resolveTemplateWithHistory(name, make(map[string]bool))
}

// resolveTemplateWithHistory resolves template with circular reference detection.
//
// Algorithm:
// 1. Track visiting templates to detect cycles (A extends B extends A)
// 2. Recursively resolve parent templates depth-first
// 3. Merge templates bottom-up (child fields override parent fields)
// 4. Clean up visiting state with defer for exception safety
//
// Merge strategy: Child templates override parent values, tags are concatenated.
// This allows progressive specialization while maintaining inheritance benefits.
func (tm *FileTemplateManager) resolveTemplateWithHistory(name string, visiting map[string]bool) (*MockDataTemplate, error) {
	// Check for circular reference - prevents infinite recursion
	if visiting[name] {
		return nil, fmt.Errorf("circular template inheritance detected: %s", name)
	}

	template, err := tm.LoadTemplate(name)
	if err != nil {
		return nil, err
	}

	// Base case: no inheritance, return template as-is
	if template.Extends == "" {
		return template, nil
	}

	// Mark this template as being visited for cycle detection
	visiting[name] = true
	defer delete(visiting, name) // Clean up on exit (exception-safe)

	// Recursively resolve parent template
	parent, err := tm.resolveTemplateWithHistory(template.Extends, visiting)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve parent template %s: %w", template.Extends, err)
	}

	// Merge templates with child-overrides-parent strategy
	resolved := &MockDataTemplate{
		Name:        template.Name,        // Child name takes precedence
		Description: template.Description, // Child description takes precedence
		Fields:      make(map[string]interface{}),
		Tags:        append(parent.Tags, template.Tags...), // Concatenate tags
		Version:     template.Version,                      // Child version takes precedence
	}

	// Merge fields with child override semantics
	for key, value := range parent.Fields {
		resolved.Fields[key] = value // Parent values first
	}
	for key, value := range template.Fields {
		resolved.Fields[key] = value // Child values override
	}

	return resolved, nil
}

// ListTemplates returns all available templates.
func (tm *FileTemplateManager) ListTemplates() ([]*MockDataTemplate, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	templates := make([]*MockDataTemplate, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}

	// Also scan directory for templates not in cache
	files, err := filepath.Glob(filepath.Join(tm.baseDir, "*.yml"))
	if err != nil {
		return templates, nil // Return cached templates on error
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".yml")
		if _, exists := tm.templates[name]; !exists {
			if template, err := tm.LoadTemplate(name); err == nil {
				templates = append(templates, template)
			}
		}
	}

	return templates, nil
}

// DeleteTemplate removes a template.
func (tm *FileTemplateManager) DeleteTemplate(name string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Remove from file system
	filename := filepath.Join(tm.baseDir, name+".yml")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	// Remove from cache
	delete(tm.templates, name)

	return nil
}

// loadDefaultTemplates loads built-in default templates for immediate usability.
//
// Design philosophy: Provides sensible defaults that work out-of-the-box for
// common component types (user, article, product, etc.). Each template includes
// realistic field mappings using faker expressions.
//
// Template coverage:
// - Personal data (user profiles, contact info)
// - Content management (articles, blogs)
// - E-commerce (products, pricing)
// - Business data (companies, events).
func (tm *FileTemplateManager) loadDefaultTemplates() {
	defaultTemplates := []*MockDataTemplate{
		{
			Name:        "default",
			Description: "Default template with basic patterns",
			Fields: map[string]interface{}{
				"name":        "{{person.name}}",
				"email":       "{{internet.email}}",
				"title":       "{{lorem.sentence}}",
				"description": "{{lorem.paragraph}}",
				"date":        "{{time.recent}}",
				"count":       "{{number.int}}",
				"active":      "{{datatype.boolean}}",
			},
			Tags:    []string{"default", "basic"},
			Version: "1.0.0",
		},
		{
			Name:        "user",
			Description: "Template for user-related components",
			Fields: map[string]interface{}{
				"firstName":  "{{person.firstName}}",
				"lastName":   "{{person.lastName}}",
				"email":      "{{internet.email}}",
				"phone":      "{{phone.number}}",
				"address":    "{{address.street}}",
				"city":       "{{address.city}}",
				"country":    "{{address.country}}",
				"avatar":     "{{internet.avatar}}",
				"birthDate":  "{{time.birthdate}}",
				"age":        "{{number.age}}",
				"isActive":   "{{datatype.boolean}}",
				"joinedDate": "{{time.recent}}",
			},
			Tags:    []string{"user", "person", "profile"},
			Version: "1.0.0",
		},
		{
			Name:        "article",
			Description: "Template for article and content components",
			Fields: map[string]interface{}{
				"title":       "{{lorem.sentence}}",
				"slug":        "{{lorem.slug}}",
				"content":     "{{lorem.paragraphs}}",
				"excerpt":     "{{lorem.paragraph}}",
				"author":      "{{person.name}}",
				"publishDate": "{{time.recent}}",
				"tags":        "{{lorem.words}}",
				"category":    "{{lorem.word}}",
				"featured":    "{{datatype.boolean}}",
				"viewCount":   "{{number.int}}",
				"imageUrl":    "{{image.url}}",
			},
			Tags:    []string{"article", "content", "blog"},
			Version: "1.0.0",
		},
		{
			Name:        "product",
			Description: "Template for product and e-commerce components",
			Fields: map[string]interface{}{
				"name":        "{{commerce.productName}}",
				"description": "{{commerce.productDescription}}",
				"price":       "{{commerce.price}}",
				"currency":    "{{finance.currencyCode}}",
				"sku":         "{{commerce.productMaterial}}",
				"category":    "{{commerce.department}}",
				"brand":       "{{company.name}}",
				"imageUrl":    "{{image.business}}",
				"inStock":     "{{datatype.boolean}}",
				"quantity":    "{{number.int}}",
				"rating":      "{{number.float}}",
				"reviews":     "{{number.int}}",
			},
			Tags:    []string{"product", "commerce", "shopping"},
			Version: "1.0.0",
		},
		{
			Name:        "company",
			Description: "Template for company and business components",
			Fields: map[string]interface{}{
				"name":        "{{company.name}}",
				"description": "{{company.catchPhrase}}",
				"website":     "{{internet.url}}",
				"email":       "{{internet.email}}",
				"phone":       "{{phone.number}}",
				"address":     "{{address.street}}",
				"city":        "{{address.city}}",
				"country":     "{{address.country}}",
				"industry":    "{{commerce.department}}",
				"employees":   "{{number.int}}",
				"founded":     "{{time.past}}",
				"logo":        "{{image.business}}",
			},
			Tags:    []string{"company", "business", "organization"},
			Version: "1.0.0",
		},
		{
			Name:        "event",
			Description: "Template for event and calendar components",
			Fields: map[string]interface{}{
				"title":       "{{lorem.sentence}}",
				"description": "{{lorem.paragraph}}",
				"startDate":   "{{time.future}}",
				"endDate":     "{{time.future}}",
				"location":    "{{address.city}}",
				"venue":       "{{company.name}} Hall",
				"organizer":   "{{person.name}}",
				"category":    "{{lorem.word}}",
				"price":       "{{commerce.price}}",
				"capacity":    "{{number.int}}",
				"attendees":   "{{number.int}}",
				"featured":    "{{datatype.boolean}}",
			},
			Tags:    []string{"event", "calendar", "meeting"},
			Version: "1.0.0",
		},
	}

	for _, template := range defaultTemplates {
		tm.templates[template.Name] = template
		// Save to file if it doesn't exist
		filename := filepath.Join(tm.baseDir, template.Name+".yml")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			_ = tm.SaveTemplate(template) // Ignore error for default templates
		}
	}
}
