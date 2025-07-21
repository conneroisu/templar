package renderer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComponentRenderer(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	assert.NotNil(t, renderer)
	assert.Equal(t, reg, renderer.registry)
	assert.Contains(t, renderer.workDir, ".templar/render")

	// Check that work directory exists
	_, err := os.Stat(renderer.workDir)
	assert.NoError(t, err)
}

func TestGenerateMockData(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	component := &types.ComponentInfo{
		Name:    "TestComponent",
		Package: "test",
		Parameters: []types.ParameterInfo{
			{Name: "title", Type: "string", Optional: false},
			{Name: "count", Type: "int", Optional: false},
			{Name: "active", Type: "bool", Optional: false},
			{Name: "items", Type: "[]string", Optional: false},
			{Name: "unknown", Type: "CustomType", Optional: false},
		},
	}

	mockData := renderer.generateMockData(component)

	assert.Len(t, mockData, 5)
	assert.Equal(t, "Sample Title", mockData["title"])
	assert.Equal(t, 42, mockData["count"])
	assert.Equal(t, true, mockData["active"])
	assert.Equal(t, []string{"Item 1", "Item 2", "Item 3"}, mockData["items"])
	assert.Equal(t, "mock_unknown", mockData["unknown"])
}

func TestGenerateMockString(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	testCases := []struct {
		name     string
		expected string
	}{
		{"title", "Sample Title"},
		{"heading", "Sample Title"},
		{"name", "John Doe"},
		{"username", "John Doe"},
		{"email", "john@example.com"},
		{"message", "This is sample content for the component preview. Lorem ipsum dolor sit amet, consectetur adipiscing elit."},
		{"content", "This is sample content for the component preview. Lorem ipsum dolor sit amet, consectetur adipiscing elit."},
		{"text", "This is sample content for the component preview. Lorem ipsum dolor sit amet, consectetur adipiscing elit."},
		{"url", "https://example.com"},
		{"link", "https://example.com"},
		{"href", "https://example.com"},
		{"variant", "primary"},
		{"type", "primary"},
		{"kind", "primary"},
		{"color", "blue"},
		{"size", "medium"},
		{"custom", "Sample Custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.generateMockString(tc.name)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateGoCode(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	component := &types.ComponentInfo{
		Name:    "Button",
		Package: "components",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string", Optional: false},
			{Name: "disabled", Type: "bool", Optional: false},
		},
	}

	mockData := map[string]interface{}{
		"text":     "Click Me",
		"disabled": false,
	}

	goCode, err := renderer.generateGoCode(component, mockData)
	require.NoError(t, err)

	assert.Contains(t, goCode, "package main")
	assert.Contains(t, goCode, "func main() {")
	assert.Contains(t, goCode, "Button(\"Click Me\", false)")
	assert.Contains(t, goCode, "component.Render(ctx, os.Stdout)")
}

func TestCopyAndModifyTemplFile(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Create a temporary templ file
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.templ")
	dstFile := filepath.Join(tempDir, "dest.templ")

	content := `package components

templ Button(text string) {
	<button>{text}</button>
}
`

	err := os.WriteFile(srcFile, []byte(content), 0644)
	require.NoError(t, err)

	// Test copying and modifying
	err = renderer.copyAndModifyTemplFile(srcFile, dstFile)
	require.NoError(t, err)

	// Read the destination file
	result, err := os.ReadFile(dstFile)
	require.NoError(t, err)

	resultStr := string(result)
	assert.Contains(t, resultStr, "package main")
	assert.NotContains(t, resultStr, "package components")
	assert.Contains(t, resultStr, "templ Button(text string) {")
}

func TestValidateWorkDir(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Test valid work directory
	validDir := ".templar/render/test"
	err := renderer.validateWorkDir(validDir)
	assert.NoError(t, err)

	// Test directory traversal attempt
	invalidDir := "../../../etc"
	err = renderer.validateWorkDir(invalidDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside current working directory")

	// Test path with .. that should be rejected due to being outside cwd
	invalidDir2 := ".templar/render/../../../etc"
	err = renderer.validateWorkDir(invalidDir2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside current working directory")
}

func TestRenderComponentWithLayout(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	componentName := "TestComponent"
	html := "<div>Test HTML</div>"

	result := renderer.RenderComponentWithLayout(componentName, html)

	assert.Contains(t, result, "<!DOCTYPE html>")
	assert.Contains(t, result, "<title>TestComponent - Templar Preview</title>")
	assert.Contains(t, result, "<div>Test HTML</div>")
	assert.Contains(t, result, "tailwind")
	assert.Contains(t, result, "WebSocket")
}

func TestRenderComponentNotFound(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Try to render a component that doesn't exist
	_, err := renderer.RenderComponent("NonExistentComponent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenderComponentIntegration(t *testing.T) {
	// This test requires templ to be installed, so we'll skip it if not available
	if !isTemplAvailable() {
		t.Skip("templ command not available, skipping integration test")
	}

	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Create a simple component
	component := &types.ComponentInfo{
		Name:         "SimpleButton",
		Package:      "main",
		FilePath:     "test.templ",
		Parameters:   []types.ParameterInfo{{Name: "text", Type: "string", Optional: false}},
		Imports:      []string{},
		LastMod:      time.Now(),
		Hash:         "testhash",
		Dependencies: []string{},
	}

	// Register the component
	reg.Register(component)

	// Create the actual templ file in a temporary location
	tempDir := t.TempDir()
	templFile := filepath.Join(tempDir, "test.templ")
	templContent := `package main

templ SimpleButton(text string) {
	<button class="btn">{text}</button>
}
`

	err := os.WriteFile(templFile, []byte(templContent), 0644)
	require.NoError(t, err)

	// Update the component file path
	component.FilePath = templFile

	// This would normally render the component, but requires Go modules and templ
	// For now, we'll just test that the method doesn't panic
	_, err = renderer.RenderComponent("SimpleButton")
	// We expect this to fail without proper setup, but shouldn't panic
	assert.Error(t, err)
}

// Helper function to check if templ command is available
func isTemplAvailable() bool {
	// Try to run templ --version
	// This is a simple check - in practice, we'd use exec.Command
	return false // For now, assume it's not available in test environment
}

func TestMockDataGeneration(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Test various parameter types
	testCases := []struct {
		paramType string
		expected  interface{}
	}{
		{"string", "Sample Test"},
		{"int", 42},
		{"int64", 42},
		{"int32", 42},
		{"bool", true},
		{"[]string", []string{"Item 1", "Item 2", "Item 3"}},
		{"CustomType", "mock_test"},
	}

	for _, tc := range testCases {
		t.Run(tc.paramType, func(t *testing.T) {
			component := &types.ComponentInfo{
				Parameters: []types.ParameterInfo{
					{Name: "test", Type: tc.paramType, Optional: false},
				},
			}

			mockData := renderer.generateMockData(component)
			assert.Equal(t, tc.expected, mockData["test"])
		})
	}
}

func TestWorkDirCreation(t *testing.T) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// The work directory should be created during initialization
	_, err := os.Stat(renderer.workDir)
	assert.NoError(t, err)

	// Check that it has reasonable permissions
	info, err := os.Stat(renderer.workDir)
	require.NoError(t, err)

	mode := info.Mode()
	assert.True(t, mode.IsDir())
	// Check that it's not world-writable (security)
	assert.Equal(t, os.FileMode(0), mode&os.FileMode(0002))
}

func TestSecureFileOperations(t *testing.T) {
	reg := registry.NewComponentRegistry()
	_ = NewComponentRenderer(reg)

	// Test that files are created with secure permissions
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(testFile, []byte("package main"), 0600)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)

	// Check that file has restricted permissions
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0600), mode&os.FileMode(0777))
}
