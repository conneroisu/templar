// Package mockdata provides tests for template management.
package mockdata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileTemplateManager_LoadTemplate(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Override base directory for testing
	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Test loading default template (should exist in memory)
	manager.loadDefaultTemplates()

	template, err := manager.LoadTemplate("default")
	require.NoError(t, err)
	assert.Equal(t, "default", template.Name)
	assert.NotEmpty(t, template.Description)
	assert.NotEmpty(t, template.Fields)

	// Test loading non-existent template
	_, err = manager.LoadTemplate("nonexistent")
	assert.Error(t, err)
}

func TestFileTemplateManager_SaveTemplate(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Test saving new template
	template := &MockDataTemplate{
		Name:        "test-template",
		Description: "Test template for unit testing",
		Fields: map[string]interface{}{
			"name":  "{{person.name}}",
			"email": "{{internet.email}}",
		},
		Tags:    []string{"test"},
		Version: "1.0.0",
	}

	err = manager.SaveTemplate(template)
	require.NoError(t, err)

	// Verify file was created
	filename := filepath.Join(tempDir, "test-template.yml")
	_, err = os.Stat(filename)
	assert.NoError(t, err)

	// Verify template is in cache
	cached, exists := manager.templates["test-template"]
	assert.True(t, exists)
	assert.Equal(t, template.Name, cached.Name)
	assert.Equal(t, template.Description, cached.Description)

	// Test saving template without name (should fail)
	invalidTemplate := &MockDataTemplate{
		Description: "Invalid template",
	}
	err = manager.SaveTemplate(invalidTemplate)
	assert.Error(t, err)
}

func TestFileTemplateManager_ResolveTemplate(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Create parent template
	parent := &MockDataTemplate{
		Name:        "parent",
		Description: "Parent template",
		Fields: map[string]interface{}{
			"name":  "{{person.name}}",
			"email": "{{internet.email}}",
			"age":   "{{number.age}}",
		},
		Tags:    []string{"base"},
		Version: "1.0.0",
	}
	err = manager.SaveTemplate(parent)
	require.NoError(t, err)

	// Create child template that extends parent
	child := &MockDataTemplate{
		Name:        "child",
		Description: "Child template",
		Extends:     "parent",
		Fields: map[string]interface{}{
			"email":    "{{internet.email}}", // Override parent
			"phone":    "{{phone.number}}",   // Add new field
			"isActive": "{{datatype.boolean}}",
		},
		Tags:    []string{"extended"},
		Version: "1.1.0",
	}
	err = manager.SaveTemplate(child)
	require.NoError(t, err)

	// Resolve child template
	resolved, err := manager.ResolveTemplate("child")
	require.NoError(t, err)

	// Should have parent fields plus child overrides/additions
	assert.Equal(t, "child", resolved.Name)
	assert.Equal(t, "Child template", resolved.Description)
	assert.Len(t, resolved.Fields, 5) // name, email, age from parent + phone, isActive from child

	// Check merged fields
	assert.Equal(t, "{{person.name}}", resolved.Fields["name"])          // From parent
	assert.Equal(t, "{{internet.email}}", resolved.Fields["email"])      // Overridden by child
	assert.Equal(t, "{{number.age}}", resolved.Fields["age"])            // From parent
	assert.Equal(t, "{{phone.number}}", resolved.Fields["phone"])        // From child
	assert.Equal(t, "{{datatype.boolean}}", resolved.Fields["isActive"]) // From child

	// Check merged tags
	assert.Contains(t, resolved.Tags, "base")
	assert.Contains(t, resolved.Tags, "extended")

	// Test resolving template without inheritance
	parentResolved, err := manager.ResolveTemplate("parent")
	require.NoError(t, err)
	assert.Equal(t, parent.Name, parentResolved.Name)
	assert.Equal(t, parent.Fields, parentResolved.Fields)

	// Test circular inheritance (should fail)
	circular := &MockDataTemplate{
		Name:    "circular",
		Extends: "circular",
		Fields:  map[string]interface{}{},
	}
	err = manager.SaveTemplate(circular)
	require.NoError(t, err)

	_, err = manager.ResolveTemplate("circular")
	assert.Error(t, err)
}

func TestFileTemplateManager_ListTemplates(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Load default templates
	manager.loadDefaultTemplates()

	// Add some custom templates
	custom1 := &MockDataTemplate{
		Name:        "custom1",
		Description: "Custom template 1",
		Fields:      map[string]interface{}{"field1": "value1"},
	}
	custom2 := &MockDataTemplate{
		Name:        "custom2",
		Description: "Custom template 2",
		Fields:      map[string]interface{}{"field2": "value2"},
	}

	err = manager.SaveTemplate(custom1)
	require.NoError(t, err)
	err = manager.SaveTemplate(custom2)
	require.NoError(t, err)

	// List all templates
	templates, err := manager.ListTemplates()
	require.NoError(t, err)

	// Should have default templates plus custom ones
	assert.GreaterOrEqual(t, len(templates), 7) // 5 default + 2 custom

	// Check that our custom templates are in the list
	foundCustom1 := false
	foundCustom2 := false
	foundDefault := false

	for _, template := range templates {
		switch template.Name {
		case "custom1":
			foundCustom1 = true
			assert.Equal(t, "Custom template 1", template.Description)
		case "custom2":
			foundCustom2 = true
			assert.Equal(t, "Custom template 2", template.Description)
		case "default":
			foundDefault = true
		}
	}

	assert.True(t, foundCustom1, "custom1 template not found in list")
	assert.True(t, foundCustom2, "custom2 template not found in list")
	assert.True(t, foundDefault, "default template not found in list")
}

func TestFileTemplateManager_DeleteTemplate(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Create and save a template
	template := &MockDataTemplate{
		Name:        "to-delete",
		Description: "Template to be deleted",
		Fields:      map[string]interface{}{"field": "value"},
	}

	err = manager.SaveTemplate(template)
	require.NoError(t, err)

	// Verify template exists
	filename := filepath.Join(tempDir, "to-delete.yml")
	_, err = os.Stat(filename)
	assert.NoError(t, err)

	_, exists := manager.templates["to-delete"]
	assert.True(t, exists)

	// Delete the template
	err = manager.DeleteTemplate("to-delete")
	require.NoError(t, err)

	// Verify template is removed from file system
	_, err = os.Stat(filename)
	assert.True(t, os.IsNotExist(err))

	// Verify template is removed from cache
	_, exists = manager.templates["to-delete"]
	assert.False(t, exists)

	// Test deleting non-existent template (should not error)
	err = manager.DeleteTemplate("nonexistent")
	assert.NoError(t, err)
}

func TestFileTemplateManager_DefaultTemplates(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Load default templates
	manager.loadDefaultTemplates()

	// Check that default templates are loaded
	expectedDefaults := []string{"default", "user", "article", "product", "company", "event"}

	for _, name := range expectedDefaults {
		template, exists := manager.templates[name]
		assert.True(t, exists, "Default template %s not found", name)
		assert.Equal(t, name, template.Name)
		assert.NotEmpty(t, template.Description)
		assert.NotEmpty(t, template.Fields)
		assert.NotEmpty(t, template.Tags)
		assert.NotEmpty(t, template.Version)
	}

	// Verify user template has expected fields
	userTemplate := manager.templates["user"]
	expectedUserFields := []string{"firstName", "lastName", "email", "phone", "address", "city", "country"}
	for _, field := range expectedUserFields {
		assert.Contains(t, userTemplate.Fields, field, "User template missing field: %s", field)
	}

	// Verify article template has expected fields
	articleTemplate := manager.templates["article"]
	expectedArticleFields := []string{"title", "content", "author", "publishDate", "tags"}
	for _, field := range expectedArticleFields {
		assert.Contains(t, articleTemplate.Fields, field, "Article template missing field: %s", field)
	}
}

func TestFileTemplateManager_ConcurrentAccess(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Load default templates
	manager.loadDefaultTemplates()

	// Test concurrent read access
	const numGoroutines = 10
	const iterationsPerGoroutine = 100

	results := make(chan *MockDataTemplate, numGoroutines*iterationsPerGoroutine)
	errors := make(chan error, numGoroutines*iterationsPerGoroutine)

	for range numGoroutines {
		go func() {
			for range iterationsPerGoroutine {
				template, err := manager.LoadTemplate("default")
				if err != nil {
					errors <- err

					return
				}
				results <- template
			}
		}()
	}

	// Collect results
	for range numGoroutines * iterationsPerGoroutine {
		select {
		case template := <-results:
			assert.Equal(t, "default", template.Name)
		case err := <-errors:
			t.Fatalf("Concurrent read failed: %v", err)
		}
	}
}

func TestFileTemplateManager_InvalidTemplateHandling(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templar_template_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create invalid YAML file
	invalidFile := filepath.Join(tempDir, "invalid.yml")
	invalidYAML := `
name: "invalid"
description: "Invalid template"
fields:
  - this is not
  - valid yaml structure
  - for our template format
`
	err = os.WriteFile(invalidFile, []byte(invalidYAML), 0600)
	require.NoError(t, err)

	manager := &FileTemplateManager{
		baseDir:   tempDir,
		templates: make(map[string]*MockDataTemplate),
	}

	// Attempt to load invalid template
	_, err = manager.LoadTemplate("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse template")
}
