package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/conneroisu/templar/internal/mockdata"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

func TestMocksCommandExists(t *testing.T) {
	// Test that the mocks command is properly registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "mocks" {
			found = true

			break
		}
	}

	if !found {
		t.Error("mocks command not found in root command")
	}
}

func TestMocksSubcommandsExist(t *testing.T) {
	tests := []string{
		"generate",
		"templates",
		"validate",
	}

	for _, expectedSubcommand := range tests {
		found := false
		for _, cmd := range mocksCmd.Commands() {
			if cmd.Name() == expectedSubcommand {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("mocks subcommand '%s' not found", expectedSubcommand)
		}
	}
}

func TestGenerateMockDataFilename(t *testing.T) {
	tests := []struct {
		componentName string
		format        string
		expected      string
	}{
		{"Button", "json", "button.json"},
		{"UserCard", "json", "usercard.json"},
		{"Button", "yaml", "button.yml"},
		{"Button", "typescript", "button.mock.ts"},
		{"Button", "unknown", "button.json"}, // defaults to JSON
	}

	for _, test := range tests {
		t.Run(test.componentName+"_"+test.format, func(t *testing.T) {
			result := generateMockDataFilename(test.componentName, test.format)
			if result != test.expected {
				t.Errorf("generateMockDataFilename(%s, %s) = %s, want %s",
					test.componentName, test.format, result, test.expected)
			}
		})
	}
}

func TestWriteMockDataJSON(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test data
	testData := map[string]interface{}{
		"name":   "John Doe",
		"email":  "john@example.com",
		"active": true,
	}

	// Write mock data
	outputPath := filepath.Join(tempDir, "test.json")
	err = writeMockDataJSON(outputPath, testData)
	if err != nil {
		t.Fatalf("writeMockDataJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal(content, &parsed)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check values
	if parsed["name"] != testData["name"] {
		t.Errorf("name mismatch: got %v, want %v", parsed["name"], testData["name"])
	}
	if parsed["email"] != testData["email"] {
		t.Errorf("email mismatch: got %v, want %v", parsed["email"], testData["email"])
	}
	if parsed["active"] != testData["active"] {
		t.Errorf("active mismatch: got %v, want %v", parsed["active"], testData["active"])
	}
}

func TestFilterTemplatesByTags(t *testing.T) {
	templates := []*mockdata.MockDataTemplate{
		{
			Name: "user",
			Tags: []string{"user", "person", "profile"},
		},
		{
			Name: "product",
			Tags: []string{"product", "commerce", "shopping"},
		},
		{
			Name: "article",
			Tags: []string{"article", "content", "blog"},
		},
	}

	tests := []struct {
		name          string
		filterTags    []string
		expectedNames []string
	}{
		{
			name:          "no filter",
			filterTags:    []string{},
			expectedNames: []string{"user", "product", "article"},
		},
		{
			name:          "filter by user",
			filterTags:    []string{"user"},
			expectedNames: []string{"user"},
		},
		{
			name:          "filter by commerce",
			filterTags:    []string{"commerce"},
			expectedNames: []string{"product"},
		},
		{
			name:          "filter by multiple tags",
			filterTags:    []string{"user", "article"},
			expectedNames: []string{"user", "article"},
		},
		{
			name:          "filter by non-existent tag",
			filterTags:    []string{"nonexistent"},
			expectedNames: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := filterTemplatesByTags(templates, test.filterTags)

			if len(result) != len(test.expectedNames) {
				t.Errorf("Expected %d templates, got %d", len(test.expectedNames), len(result))

				return
			}

			resultNames := make([]string, len(result))
			for i, template := range result {
				resultNames[i] = template.Name
			}

			for _, expectedName := range test.expectedNames {
				found := false
				for _, resultName := range resultNames {
					if resultName == expectedName {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("Expected template '%s' not found in results", expectedName)
				}
			}
		})
	}
}

func TestHasMatchingTags(t *testing.T) {
	tests := []struct {
		name         string
		templateTags []string
		filterTags   []string
		expected     bool
	}{
		{
			name:         "exact match",
			templateTags: []string{"user", "profile"},
			filterTags:   []string{"user"},
			expected:     true,
		},
		{
			name:         "partial match",
			templateTags: []string{"user-profile", "person"},
			filterTags:   []string{"user"},
			expected:     true,
		},
		{
			name:         "no match",
			templateTags: []string{"product", "commerce"},
			filterTags:   []string{"user"},
			expected:     false,
		},
		{
			name:         "multiple filters with match",
			templateTags: []string{"user", "profile"},
			filterTags:   []string{"user", "article"},
			expected:     true,
		},
		{
			name:         "case insensitive match",
			templateTags: []string{"User", "Profile"},
			filterTags:   []string{"user"},
			expected:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := hasMatchingTags(test.templateTags, test.filterTags)
			if result != test.expected {
				t.Errorf("hasMatchingTags(%v, %v) = %v, want %v",
					test.templateTags, test.filterTags, result, test.expected)
			}
		})
	}
}

func TestGenerateMockDataForComponent(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "templar-mock-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a mock component registry
	registry := registry.NewComponentRegistry()

	// Create a test component
	testComponent := &types.ComponentInfo{
		Name:     "TestButton",
		Package:  "components",
		FilePath: "/test/button.templ",
		Parameters: []types.ParameterInfo{
			{
				Name:     "text",
				Type:     "string",
				Optional: false,
			},
			{
				Name:     "disabled",
				Type:     "bool",
				Optional: true,
				Default:  false,
			},
		},
	}

	// Register the component
	registry.Register(testComponent)

	// Create mock generator
	config := mockdata.DefaultMockDataConfig()
	config.Seed = 12345 // For deterministic testing
	generator := mockdata.NewIntelligentMockGenerator(config)

	// Test generation
	err = generateMockDataForComponent(
		registry,
		generator,
		"TestButton",
		tempDir,
		"json",
		1,
		"",
		true,
		false,
	)

	if err != nil {
		t.Fatalf("generateMockDataForComponent failed: %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(tempDir, "testbutton.json")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Mock data file was not created")
	}

	// Verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read mock data file: %v", err)
	}

	var mockData map[string]interface{}
	err = json.Unmarshal(content, &mockData)
	if err != nil {
		t.Fatalf("Failed to parse mock data JSON: %v", err)
	}

	// Verify required fields exist
	if _, exists := mockData["text"]; !exists {
		t.Error("Required field 'text' missing from mock data")
	}

	if _, exists := mockData["disabled"]; !exists {
		t.Error("Field 'disabled' missing from mock data")
	}
}

func TestGenerateMockDataForComponentNotFound(t *testing.T) {
	// Create empty registry
	registry := registry.NewComponentRegistry()

	// Create mock generator
	generator := mockdata.NewIntelligentMockGenerator(mockdata.DefaultMockDataConfig())

	// Test with non-existent component
	err := generateMockDataForComponent(
		registry,
		generator,
		"NonExistentComponent",
		"/tmp",
		"json",
		1,
		"",
		true,
		false,
	)

	if err == nil {
		t.Error("Expected error for non-existent component, got nil")
	}

	expectedError := "component not found: NonExistentComponent"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMockDataFileOverwrite(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "templar-overwrite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create registry with test component
	registry := registry.NewComponentRegistry()
	testComponent := &types.ComponentInfo{
		Name:       "TestComponent",
		Parameters: []types.ParameterInfo{{Name: "test", Type: "string"}},
	}
	registry.Register(testComponent)

	generator := mockdata.NewIntelligentMockGenerator(mockdata.DefaultMockDataConfig())

	// Create existing file
	existingFile := filepath.Join(tempDir, "testcomponent.json")
	err = os.WriteFile(existingFile, []byte(`{"existing": "data"}`), 0600)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Test without overwrite (should not modify file)
	err = generateMockDataForComponent(
		registry,
		generator,
		"TestComponent",
		tempDir,
		"json",
		1,
		"",
		true,
		false, // overwrite = false
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// File should still contain original content
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != `{"existing": "data"}` {
		t.Error("File was modified when overwrite=false")
	}

	// Test with overwrite (should modify file)
	err = generateMockDataForComponent(
		registry,
		generator,
		"TestComponent",
		tempDir,
		"json",
		1,
		"",
		true,
		true, // overwrite = true
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// File should now contain new mock data
	content, err = os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) == `{"existing": "data"}` {
		t.Error("File was not modified when overwrite=true")
	}
}

// Benchmark tests.
func BenchmarkGenerateMockDataFilename(b *testing.B) {
	for range b.N {
		generateMockDataFilename("TestComponent", "json")
	}
}

func BenchmarkHasMatchingTags(b *testing.B) {
	templateTags := []string{"user", "profile", "person", "account"}
	filterTags := []string{"user", "admin"}

	for range b.N {
		hasMatchingTags(templateTags, filterTags)
	}
}
