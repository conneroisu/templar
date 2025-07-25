//go:build visual
// +build visual

package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/conneroisu/templar/internal/types"
)

// TestVisualRegressionSuite runs the comprehensive visual regression test suite
func TestVisualRegressionSuite(t *testing.T) {
	// Setup test environment with permanent golden directory
	goldenDir := filepath.Join(".", "golden")

	// Check if we're in update mode
	updateMode := os.Getenv("UPDATE_GOLDEN") == "true"

	// Create visual regression tester
	vrt := NewVisualRegressionTester(goldenDir, updateMode)

	// Register test components
	testComponents := []*types.ComponentInfo{
		{
			Name:     "Button",
			Package:  "components",
			FilePath: "button.templ",
			Parameters: []types.ParameterInfo{
				{Name: "text", Type: "string"},
				{Name: "variant", Type: "string"},
				{Name: "disabled", Type: "bool"},
			},
		},
		{
			Name:     "Card",
			Package:  "components",
			FilePath: "card.templ",
			Parameters: []types.ParameterInfo{
				{Name: "title", Type: "string"},
				{Name: "content", Type: "string"},
				{Name: "showBorder", Type: "bool"},
			},
		},
		{
			Name:     "Layout",
			Package:  "layouts",
			FilePath: "layout.templ",
			Parameters: []types.ParameterInfo{
				{Name: "title", Type: "string"},
			},
		},
	}

	vrt.RegisterComponents(testComponents)

	// Define test cases
	testCases := []TestCase{
		{
			Name:        "Button_Default",
			Component:   "Button",
			Props:       map[string]interface{}{"text": "Click me"},
			GoldenFile:  "button_default.html",
			Description: "Default button with basic text",
			Tags:        []string{"button", "basic"},
		},
		{
			Name:        "Button_Primary",
			Component:   "Button",
			Props:       map[string]interface{}{"text": "Primary Button", "variant": "primary"},
			GoldenFile:  "button_primary.html",
			Description: "Primary variant button",
			Tags:        []string{"button", "variant"},
		},
		{
			Name:        "Button_Disabled",
			Component:   "Button",
			Props:       map[string]interface{}{"text": "Disabled", "disabled": true},
			GoldenFile:  "button_disabled.html",
			Description: "Disabled button state",
			Tags:        []string{"button", "state"},
		},
		{
			Name:        "Card_Basic",
			Component:   "Card",
			Props:       map[string]interface{}{"title": "Test Card", "content": "This is test content"},
			GoldenFile:  "card_basic.html",
			Description: "Basic card with title and content",
			Tags:        []string{"card", "basic"},
		},
		{
			Name:        "Card_WithBorder",
			Component:   "Card",
			Props:       map[string]interface{}{"title": "Bordered Card", "content": "Content", "showBorder": true},
			GoldenFile:  "card_bordered.html",
			Description: "Card with border enabled",
			Tags:        []string{"card", "border"},
		},
		{
			Name:        "Layout_Default",
			Component:   "Layout",
			Props:       map[string]interface{}{"title": "Test Page"},
			GoldenFile:  "layout_default.html",
			Description: "Default page layout",
			Tags:        []string{"layout", "page"},
		},
	}

	// Run test suite
	results := vrt.RunTestSuite(t, testCases)

	// Generate and log report
	report := vrt.GenerateReport(results)
	t.Logf("Visual Regression Report:\n%s", report)

	// Verify all tests passed (unless in update mode)
	if !updateMode {
		for _, result := range results {
			if result.Error != nil {
				t.Errorf("Test %s failed with error: %v", result.TestCase.Name, result.Error)
			}
			if !result.Passed {
				t.Errorf("Visual regression detected for test %s", result.TestCase.Name)
			}
		}
	}
}

// TestVisualRegressionButtonVariants tests button component variants
func TestVisualRegressionButtonVariants(t *testing.T) {
	goldenDir := filepath.Join(".", "golden")
	updateMode := os.Getenv("UPDATE_GOLDEN") == "true"

	vrt := NewVisualRegressionTester(goldenDir, updateMode)

	// Register button component
	buttonComponent := &types.ComponentInfo{
		Name:     "Button",
		Package:  "components",
		FilePath: "button.templ",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string"},
			{Name: "variant", Type: "string"},
			{Name: "size", Type: "string"},
			{Name: "disabled", Type: "bool"},
		},
	}
	vrt.RegisterComponents([]*types.ComponentInfo{buttonComponent})

	// Test different button variants
	variants := []struct {
		name  string
		props map[string]interface{}
	}{
		{"default", map[string]interface{}{"text": "Default"}},
		{"primary", map[string]interface{}{"text": "Primary", "variant": "primary"}},
		{"secondary", map[string]interface{}{"text": "Secondary", "variant": "secondary"}},
		{"danger", map[string]interface{}{"text": "Danger", "variant": "danger"}},
		{"small", map[string]interface{}{"text": "Small", "size": "small"}},
		{"large", map[string]interface{}{"text": "Large", "size": "large"}},
		{"disabled", map[string]interface{}{"text": "Disabled", "disabled": true}},
	}

	testCases := make([]TestCase, len(variants))
	for i, variant := range variants {
		testCases[i] = TestCase{
			Name:        "Button_" + variant.name,
			Component:   "Button",
			Props:       variant.props,
			GoldenFile:  "button_" + variant.name + ".html",
			Description: "Button variant: " + variant.name,
			Tags:        []string{"button", "variant", variant.name},
		}
	}

	results := vrt.RunTestSuite(t, testCases)

	if !updateMode {
		for _, result := range results {
			if !result.Passed || result.Error != nil {
				t.Errorf("Button variant test failed: %s", result.TestCase.Name)
			}
		}
	}
}

// TestVisualRegressionEdgeCases tests edge cases in visual regression
func TestVisualRegressionEdgeCases(t *testing.T) {
	goldenDir := filepath.Join(".", "golden")
	updateMode := os.Getenv("UPDATE_GOLDEN") == "true"

	vrt := NewVisualRegressionTester(goldenDir, updateMode)

	// Register test component
	testComponent := &types.ComponentInfo{
		Name:     "TestComponent",
		Package:  "test",
		FilePath: "test.templ",
		Parameters: []types.ParameterInfo{
			{Name: "content", Type: "string"},
		},
	}
	vrt.RegisterComponents([]*types.ComponentInfo{testComponent})

	// Test edge cases
	edgeCases := []TestCase{
		{
			Name:        "EmptyContent",
			Component:   "TestComponent",
			Props:       map[string]interface{}{"content": ""},
			GoldenFile:  "empty_content.html",
			Description: "Component with empty content",
		},
		{
			Name:        "SpecialCharacters",
			Component:   "TestComponent",
			Props:       map[string]interface{}{"content": "<>&\"'"},
			GoldenFile:  "special_chars.html",
			Description: "Component with special HTML characters",
		},
		{
			Name:        "UnicodeContent",
			Component:   "TestComponent",
			Props:       map[string]interface{}{"content": "🚀 Hello 世界"},
			GoldenFile:  "unicode_content.html",
			Description: "Component with unicode characters",
		},
		{
			Name:        "LongContent",
			Component:   "TestComponent",
			Props:       map[string]interface{}{"content": "This is a very long piece of content that might wrap across multiple lines and test how the component handles longer text inputs."},
			GoldenFile:  "long_content.html",
			Description: "Component with long content",
		},
	}

	results := vrt.RunTestSuite(t, edgeCases)

	if !updateMode {
		for _, result := range results {
			if !result.Passed || result.Error != nil {
				t.Errorf("Edge case test failed: %s", result.TestCase.Name)
			}
		}
	}
}

// TestVisualRegressionComponentNotFound tests behavior when component doesn't exist
func TestVisualRegressionComponentNotFound(t *testing.T) {
	goldenDir := filepath.Join(".", "golden")

	vrt := NewVisualRegressionTester(goldenDir, false)

	testCase := TestCase{
		Name:        "NonExistentComponent",
		Component:   "NonExistent",
		Props:       map[string]interface{}{},
		GoldenFile:  "nonexistent.html",
		Description: "Test with non-existent component",
	}

	result := vrt.RunTest(t, testCase)

	// Should fail with an error
	if result.Error == nil {
		t.Error("Expected error for non-existent component, but got none")
	}

	if result.Passed {
		t.Error("Test should not pass for non-existent component")
	}
}

// BenchmarkVisualRegressionPerformance benchmarks visual regression testing performance
func BenchmarkVisualRegressionPerformance(b *testing.B) {
	goldenDir := filepath.Join(".", "golden")

	vrt := NewVisualRegressionTester(goldenDir, true) // Update mode for benchmark

	// Register test component
	component := &types.ComponentInfo{
		Name:     "BenchComponent",
		Package:  "bench",
		FilePath: "bench.templ",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string"},
		},
	}
	vrt.RegisterComponents([]*types.ComponentInfo{component})

	testCase := TestCase{
		Name:        "BenchTest",
		Component:   "BenchComponent",
		Props:       map[string]interface{}{"text": "Benchmark content"},
		GoldenFile:  "bench.html",
		Description: "Benchmark test case",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := vrt.RunTest(&testing.T{}, testCase)
			if result.Error != nil {
				b.Errorf("Benchmark test failed: %v", result.Error)
			}
		}
	})
}

// TestHashingConsistency tests that content hashing is consistent
func TestHashingConsistency(t *testing.T) {
	vrt := NewVisualRegressionTester("", false)

	content := []byte("test content")

	// Hash the same content multiple times
	hash1 := vrt.hashContent(content)
	hash2 := vrt.hashContent(content)
	hash3 := vrt.hashContent(content)

	if hash1 != hash2 || hash2 != hash3 {
		t.Error("Hash function is not consistent")
	}

	// Different content should produce different hashes
	differentContent := []byte("different content")
	differentHash := vrt.hashContent(differentContent)

	if hash1 == differentHash {
		t.Error("Different content produced same hash")
	}
}
