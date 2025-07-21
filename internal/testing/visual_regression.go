package testing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
)

// VisualRegressionTester handles visual regression testing for template output
type VisualRegressionTester struct {
	goldenDir  string
	updateMode bool
	renderer   *renderer.ComponentRenderer
	registry   *registry.ComponentRegistry
}

// TestCase represents a visual regression test case
type TestCase struct {
	Name        string                 `json:"name"`
	Component   string                 `json:"component"`
	Props       map[string]interface{} `json:"props"`
	GoldenFile  string                 `json:"golden_file"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags,omitempty"`
}

// RegressionResult contains the results of a visual regression test
type RegressionResult struct {
	TestCase     TestCase `json:"test_case"`
	Passed       bool     `json:"passed"`
	Expected     string   `json:"expected,omitempty"`
	Actual       string   `json:"actual,omitempty"`
	Diff         string   `json:"diff,omitempty"`
	Error        error    `json:"error,omitempty"`
	OutputHash   string   `json:"output_hash"`
	ExpectedHash string   `json:"expected_hash"`
}

// NewVisualRegressionTester creates a new visual regression tester
func NewVisualRegressionTester(goldenDir string, updateMode bool) *VisualRegressionTester {
	reg := registry.NewComponentRegistry()
	renderer := renderer.NewComponentRenderer(reg)

	return &VisualRegressionTester{
		goldenDir:  goldenDir,
		updateMode: updateMode,
		renderer:   renderer,
		registry:   reg,
	}
}

// RegisterComponents adds components to the registry for testing
func (vrt *VisualRegressionTester) RegisterComponents(components []*types.ComponentInfo) {
	for _, component := range components {
		vrt.registry.Register(component)
	}
}

// RunTest executes a single visual regression test
func (vrt *VisualRegressionTester) RunTest(t *testing.T, testCase TestCase) *RegressionResult {
	result := &RegressionResult{
		TestCase: testCase,
		Passed:   false,
	}

	// Render the component
	output, err := vrt.renderComponent(testCase.Component, testCase.Props)
	if err != nil {
		result.Error = fmt.Errorf("failed to render component %s: %w", testCase.Component, err)
		return result
	}

	result.Actual = string(output)
	result.OutputHash = vrt.hashContent(output)

	// Get golden file path
	goldenPath := filepath.Join(vrt.goldenDir, testCase.GoldenFile)

	if vrt.updateMode {
		// Update golden file
		if err := vrt.updateGoldenFile(goldenPath, output); err != nil {
			result.Error = fmt.Errorf("failed to update golden file %s: %w", goldenPath, err)
			return result
		}
		result.Passed = true
		return result
	}

	// Compare with golden file
	expected, err := vrt.readGoldenFile(goldenPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read golden file %s: %w", goldenPath, err)
		return result
	}

	result.Expected = string(expected)
	result.ExpectedHash = vrt.hashContent(expected)

	// Compare content
	if bytes.Equal(output, expected) {
		result.Passed = true
	} else {
		result.Diff = vrt.generateDiff(expected, output)
	}

	return result
}

// RunTestSuite executes multiple visual regression tests
func (vrt *VisualRegressionTester) RunTestSuite(t *testing.T, testCases []TestCase) []*RegressionResult {
	results := make([]*RegressionResult, 0, len(testCases))

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result := vrt.RunTest(t, testCase)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Test failed with error: %v", result.Error)
			} else if !result.Passed && !vrt.updateMode {
				t.Errorf("Visual regression detected for %s:\nExpected hash: %s\nActual hash: %s\nDiff:\n%s",
					testCase.Name, result.ExpectedHash, result.OutputHash, result.Diff)
			}
		})
	}

	return results
}

// renderComponent renders a component with given props
func (vrt *VisualRegressionTester) renderComponent(componentName string, props map[string]interface{}) ([]byte, error) {
	// This is a placeholder implementation
	// In a real implementation, this would render the actual component

	component, exists := vrt.registry.Get(componentName)
	if !exists {
		return nil, fmt.Errorf("component %s not found", componentName)
	}

	// Generate HTML based on component and props
	html := vrt.generateHTML(component, props)
	return []byte(html), nil
}

// generateHTML generates HTML for a component (placeholder implementation)
func (vrt *VisualRegressionTester) generateHTML(component *types.ComponentInfo, props map[string]interface{}) string {
	var html strings.Builder

	html.WriteString(fmt.Sprintf("<!-- Component: %s -->\n", component.Name))
	html.WriteString(fmt.Sprintf("<div class=\"component %s\">\n", strings.ToLower(component.Name)))

	// Generate content based on component type and props
	switch component.Name {
	case "Button":
		text := "Click me"
		if textProp, ok := props["text"]; ok {
			if textStr, ok := textProp.(string); ok {
				text = textStr
			}
		}
		html.WriteString(fmt.Sprintf("  <button type=\"button\">%s</button>\n", text))

	case "Card":
		title := "Card Title"
		content := "Card content"
		if titleProp, ok := props["title"]; ok {
			if titleStr, ok := titleProp.(string); ok {
				title = titleStr
			}
		}
		if contentProp, ok := props["content"]; ok {
			if contentStr, ok := contentProp.(string); ok {
				content = contentStr
			}
		}
		html.WriteString(fmt.Sprintf("  <div class=\"card-header\">%s</div>\n", title))
		html.WriteString(fmt.Sprintf("  <div class=\"card-body\">%s</div>\n", content))

	case "Layout":
		html.WriteString("  <!DOCTYPE html>\n")
		html.WriteString("  <html>\n")
		html.WriteString("  <head><title>Layout</title></head>\n")
		html.WriteString("  <body>\n")
		html.WriteString("    <main>Content goes here</main>\n")
		html.WriteString("  </body>\n")
		html.WriteString("  </html>\n")

	default:
		html.WriteString(fmt.Sprintf("  <div>Unknown component: %s</div>\n", component.Name))
	}

	html.WriteString("</div>\n")
	html.WriteString(fmt.Sprintf("<!-- End Component: %s -->\n", component.Name))

	return html.String()
}

// hashContent generates a SHA256 hash of content for comparison
func (vrt *VisualRegressionTester) hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// updateGoldenFile writes content to a golden file
func (vrt *VisualRegressionTester) updateGoldenFile(path string, content []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(path, content, 0644)
}

// readGoldenFile reads content from a golden file
func (vrt *VisualRegressionTester) readGoldenFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

// generateDiff creates a simple diff between expected and actual content
func (vrt *VisualRegressionTester) generateDiff(expected, actual []byte) string {
	expectedLines := strings.Split(string(expected), "\n")
	actualLines := strings.Split(string(actual), "\n")

	var diff strings.Builder
	diff.WriteString("--- Expected\n")
	diff.WriteString("+++ Actual\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var expectedLine, actualLine string

		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			if expectedLine != "" {
				diff.WriteString(fmt.Sprintf("-%s\n", expectedLine))
			}
			if actualLine != "" {
				diff.WriteString(fmt.Sprintf("+%s\n", actualLine))
			}
		}
	}

	return diff.String()
}

// GenerateReport creates a detailed test report
func (vrt *VisualRegressionTester) GenerateReport(results []*RegressionResult) string {
	var report strings.Builder

	passed := 0
	failed := 0
	errors := 0

	report.WriteString("# Visual Regression Test Report\n\n")

	for _, result := range results {
		if result.Error != nil {
			errors++
		} else if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	report.WriteString(fmt.Sprintf("## Summary\n"))
	report.WriteString(fmt.Sprintf("- **Total Tests**: %d\n", len(results)))
	report.WriteString(fmt.Sprintf("- **Passed**: %d\n", passed))
	report.WriteString(fmt.Sprintf("- **Failed**: %d\n", failed))
	report.WriteString(fmt.Sprintf("- **Errors**: %d\n", errors))
	report.WriteString("\n")

	if failed > 0 || errors > 0 {
		report.WriteString("## Failed Tests\n\n")
		for _, result := range results {
			if !result.Passed || result.Error != nil {
				report.WriteString(fmt.Sprintf("### %s\n", result.TestCase.Name))
				report.WriteString(fmt.Sprintf("**Component**: %s\n", result.TestCase.Component))

				if result.Error != nil {
					report.WriteString(fmt.Sprintf("**Error**: %s\n", result.Error.Error()))
				} else {
					report.WriteString(fmt.Sprintf("**Expected Hash**: %s\n", result.ExpectedHash))
					report.WriteString(fmt.Sprintf("**Actual Hash**: %s\n", result.OutputHash))
					if result.Diff != "" {
						report.WriteString("**Diff**:\n```\n")
						report.WriteString(result.Diff)
						report.WriteString("\n```\n")
					}
				}
				report.WriteString("\n")
			}
		}
	}

	return report.String()
}

// CleanupGoldenFiles removes unused golden files
func (vrt *VisualRegressionTester) CleanupGoldenFiles(activeFiles []string) error {
	if !vrt.updateMode {
		return fmt.Errorf("cleanup only available in update mode")
	}

	activeFileSet := make(map[string]bool)
	for _, file := range activeFiles {
		activeFileSet[file] = true
	}

	return filepath.Walk(vrt.goldenDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(vrt.goldenDir, path)
			if err != nil {
				return err
			}

			if !activeFileSet[relPath] {
				fmt.Printf("Removing unused golden file: %s\n", relPath)
				return os.Remove(path)
			}
		}

		return nil
	})
}
