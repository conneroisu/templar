package testing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
)

// VisualRegressionTester handles visual regression testing for template output
type VisualRegressionTester struct {
	goldenDir       string
	updateMode      bool
	renderer        *renderer.ComponentRenderer
	registry        *registry.ComponentRegistry
	screenshotDir   string
	enablePuppeteer bool
	serverPort      int
}

// TestCase represents a visual regression test case
type TestCase struct {
	Name        string                 `json:"name"`
	Component   string                 `json:"component"`
	Props       map[string]interface{} `json:"props"`
	GoldenFile  string                 `json:"golden_file"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags,omitempty"`
	Viewport    Viewport               `json:"viewport,omitempty"`
	WaitFor     string                 `json:"wait_for,omitempty"`
	Screenshot  bool                   `json:"screenshot,omitempty"`
}

// Viewport defines the browser viewport size for screenshot tests
type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RegressionResult contains the results of a visual regression test
type RegressionResult struct {
	TestCase       TestCase `json:"test_case"`
	Passed         bool     `json:"passed"`
	Expected       string   `json:"expected,omitempty"`
	Actual         string   `json:"actual,omitempty"`
	Diff           string   `json:"diff,omitempty"`
	Error          error    `json:"error,omitempty"`
	OutputHash     string   `json:"output_hash"`
	ExpectedHash   string   `json:"expected_hash"`
	ScreenshotPath string   `json:"screenshot_path,omitempty"`
	BaselinePath   string   `json:"baseline_path,omitempty"`
	DiffImagePath  string   `json:"diff_image_path,omitempty"`
	VisualDiff     bool     `json:"visual_diff"`
	PixelDiff      int      `json:"pixel_diff"`
	PercentDiff    float64  `json:"percent_diff"`
}

// NewVisualRegressionTester creates a new visual regression tester
func NewVisualRegressionTester(goldenDir string, updateMode bool) *VisualRegressionTester {
	reg := registry.NewComponentRegistry()
	renderer := renderer.NewComponentRenderer(reg)

	return &VisualRegressionTester{
		goldenDir:       goldenDir,
		updateMode:      updateMode,
		renderer:        renderer,
		registry:        reg,
		screenshotDir:   filepath.Join(goldenDir, "screenshots"),
		enablePuppeteer: checkPuppeteerAvailable(),
		serverPort:      8089, // Use different port to avoid conflicts
	}
}

// NewVisualRegressionTesterWithOptions creates a tester with custom options
func NewVisualRegressionTesterWithOptions(
	goldenDir string,
	updateMode bool,
	options VisualTestOptions,
) *VisualRegressionTester {
	vrt := NewVisualRegressionTester(goldenDir, updateMode)

	if options.ScreenshotDir != "" {
		vrt.screenshotDir = options.ScreenshotDir
	}
	if options.ServerPort > 0 {
		vrt.serverPort = options.ServerPort
	}
	vrt.enablePuppeteer = options.EnablePuppeteer

	return vrt
}

// VisualTestOptions provides configuration options for visual regression testing
type VisualTestOptions struct {
	ScreenshotDir   string
	EnablePuppeteer bool
	ServerPort      int
}

// checkPuppeteerAvailable checks if Puppeteer or similar tools are available
func checkPuppeteerAvailable() bool {
	// Check for Puppeteer CLI or Chrome/Chromium in headless mode
	tools := []string{"puppeteer", "chrome", "chromium", "google-chrome"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			return true
		}
	}
	return false
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

	// Handle screenshot tests
	if testCase.Screenshot && vrt.enablePuppeteer {
		screenshotResult, err := vrt.runScreenshotTest(t, testCase, output)
		if err != nil {
			result.Error = fmt.Errorf("screenshot test failed: %w", err)
			return result
		}

		// Merge screenshot results
		result.ScreenshotPath = screenshotResult.ScreenshotPath
		result.BaselinePath = screenshotResult.BaselinePath
		result.DiffImagePath = screenshotResult.DiffImagePath
		result.VisualDiff = screenshotResult.VisualDiff
		result.PixelDiff = screenshotResult.PixelDiff
		result.PercentDiff = screenshotResult.PercentDiff

		if !screenshotResult.Passed {
			result.Passed = false
			return result
		}
	}

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
func (vrt *VisualRegressionTester) RunTestSuite(
	t *testing.T,
	testCases []TestCase,
) []*RegressionResult {
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
func (vrt *VisualRegressionTester) renderComponent(
	componentName string,
	props map[string]interface{},
) ([]byte, error) {
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
func (vrt *VisualRegressionTester) generateHTML(
	component *types.ComponentInfo,
	props map[string]interface{},
) string {
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

// runScreenshotTest performs visual screenshot testing
func (vrt *VisualRegressionTester) runScreenshotTest(
	t *testing.T,
	testCase TestCase,
	htmlContent []byte,
) (*RegressionResult, error) {
	result := &RegressionResult{
		TestCase: testCase,
		Passed:   false,
	}

	// Ensure screenshot directory exists
	if err := os.MkdirAll(vrt.screenshotDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Create a temporary HTML file for the component
	tempFile := filepath.Join(vrt.screenshotDir, fmt.Sprintf("%s_temp.html", testCase.Name))
	fullHTML := vrt.createFullHTMLPage(htmlContent, testCase)

	if err := ioutil.WriteFile(tempFile, []byte(fullHTML), 0644); err != nil {
		return result, fmt.Errorf("failed to write temporary HTML file: %w", err)
	}
	defer os.Remove(tempFile)

	// Screenshot paths
	screenshotPath := filepath.Join(vrt.screenshotDir, fmt.Sprintf("%s.png", testCase.Name))
	baselinePath := filepath.Join(
		vrt.screenshotDir,
		"baselines",
		fmt.Sprintf("%s.png", testCase.Name),
	)
	diffPath := filepath.Join(vrt.screenshotDir, "diffs", fmt.Sprintf("%s_diff.png", testCase.Name))

	// Take screenshot
	if err := vrt.takeScreenshot(tempFile, screenshotPath, testCase.Viewport); err != nil {
		return result, fmt.Errorf("failed to take screenshot: %w", err)
	}

	result.ScreenshotPath = screenshotPath

	if vrt.updateMode {
		// Update baseline
		if err := vrt.updateBaseline(screenshotPath, baselinePath); err != nil {
			return result, fmt.Errorf("failed to update baseline: %w", err)
		}
		result.Passed = true
		return result, nil
	}

	// Compare with baseline
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		return result, fmt.Errorf("baseline image not found: %s", baselinePath)
	}

	result.BaselinePath = baselinePath

	// Perform image comparison
	pixelDiff, percentDiff, err := vrt.compareImages(baselinePath, screenshotPath, diffPath)
	if err != nil {
		return result, fmt.Errorf("failed to compare images: %w", err)
	}

	result.PixelDiff = pixelDiff
	result.PercentDiff = percentDiff
	result.DiffImagePath = diffPath

	// Determine if test passed (allow small differences)
	threshold := 0.1 // 0.1% difference threshold
	if percentDiff <= threshold {
		result.Passed = true
	} else {
		result.VisualDiff = true
	}

	return result, nil
}

// createFullHTMLPage wraps component HTML in a full page template
func (vrt *VisualRegressionTester) createFullHTMLPage(
	componentHTML []byte,
	testCase TestCase,
) string {
	viewport := testCase.Viewport
	if viewport.Width == 0 {
		viewport.Width = 1280
	}
	if viewport.Height == 0 {
		viewport.Height = 720
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Visual Test: %s</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: white;
        }
        .test-container {
            width: %dpx;
            min-height: %dpx;
        }
    </style>
</head>
<body>
    <div class="test-container">
        %s
    </div>
    %s
</body>
</html>`, testCase.Name, viewport.Width-40, viewport.Height-40, string(componentHTML), vrt.getWaitForScript(testCase.WaitFor))

	return html
}

// getWaitForScript returns a script to wait for specific conditions
func (vrt *VisualRegressionTester) getWaitForScript(waitFor string) string {
	if waitFor == "" {
		return `<script>
			// Wait for fonts and images to load
			document.addEventListener('DOMContentLoaded', function() {
				if (document.fonts) {
					document.fonts.ready.then(function() {
						document.body.setAttribute('data-visual-test-ready', 'true');
					});
				} else {
					setTimeout(function() {
						document.body.setAttribute('data-visual-test-ready', 'true');
					}, 500);
				}
			});
		</script>`
	}

	return fmt.Sprintf(`<script>
		document.addEventListener('DOMContentLoaded', function() {
			function checkCondition() {
				if (%s) {
					document.body.setAttribute('data-visual-test-ready', 'true');
				} else {
					setTimeout(checkCondition, 100);
				}
			}
			checkCondition();
		});
	</script>`, waitFor)
}

// takeScreenshot captures a screenshot using headless Chrome/Chromium
func (vrt *VisualRegressionTester) takeScreenshot(
	htmlFile, outputPath string,
	viewport Viewport,
) error {
	if viewport.Width == 0 {
		viewport.Width = 1280
	}
	if viewport.Height == 0 {
		viewport.Height = 720
	}

	// Try different Chrome/Chromium binaries
	chromeBinaries := []string{"google-chrome", "chromium", "chrome"}
	var chromeCmd string

	for _, binary := range chromeBinaries {
		if _, err := exec.LookPath(binary); err == nil {
			chromeCmd = binary
			break
		}
	}

	if chromeCmd == "" {
		return fmt.Errorf("no Chrome/Chromium binary found")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Chrome arguments for headless screenshot
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-web-security",
		"--virtual-time-budget=5000",
		fmt.Sprintf("--window-size=%d,%d", viewport.Width, viewport.Height),
		fmt.Sprintf("--screenshot=%s", outputPath),
		fmt.Sprintf("file://%s", htmlFile),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, chromeCmd, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chrome screenshot failed: %w, output: %s", err, string(output))
	}

	// Verify screenshot was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("screenshot file was not created: %s", outputPath)
	}

	return nil
}

// updateBaseline copies the current screenshot as the new baseline
func (vrt *VisualRegressionTester) updateBaseline(screenshotPath, baselinePath string) error {
	// Ensure baseline directory exists
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		return err
	}

	src, err := os.Open(screenshotPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(baselinePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = dst.ReadFrom(src)
	return err
}

// compareImages compares two images and returns pixel difference and percentage
func (vrt *VisualRegressionTester) compareImages(
	baselinePath, screenshotPath, diffPath string,
) (int, float64, error) {
	// For now, use a simple file size comparison as a basic difference metric
	// In a production system, you would use an image comparison library like ImageMagick or a Go image library

	baselineInfo, err := os.Stat(baselinePath)
	if err != nil {
		return 0, 0, err
	}

	screenshotInfo, err := os.Stat(screenshotPath)
	if err != nil {
		return 0, 0, err
	}

	sizeDiff := int(screenshotInfo.Size() - baselineInfo.Size())
	if sizeDiff < 0 {
		sizeDiff = -sizeDiff
	}

	// Calculate percentage difference based on file size (rough approximation)
	percentDiff := float64(sizeDiff) / float64(baselineInfo.Size()) * 100

	// Create a simple diff marker file
	if percentDiff > 0.1 {
		diffInfo := fmt.Sprintf(
			"Baseline: %d bytes\nScreenshot: %d bytes\nDifference: %d bytes (%.2f%%)",
			baselineInfo.Size(),
			screenshotInfo.Size(),
			sizeDiff,
			percentDiff,
		)

		if err := os.MkdirAll(filepath.Dir(diffPath), 0755); err != nil {
			return sizeDiff, percentDiff, err
		}

		if err := ioutil.WriteFile(diffPath+".txt", []byte(diffInfo), 0644); err != nil {
			return sizeDiff, percentDiff, err
		}
	}

	return sizeDiff, percentDiff, nil
}

// StartTestServer starts a test server for component preview testing
func (vrt *VisualRegressionTester) StartTestServer(ctx context.Context) (*http.Server, error) {
	mux := http.NewServeMux()

	// Serve test components
	mux.HandleFunc("/component/", vrt.handleComponentPreview)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", vrt.serverPort),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Handle server error
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	return server, nil
}

// handleComponentPreview serves component previews for testing
func (vrt *VisualRegressionTester) handleComponentPreview(w http.ResponseWriter, r *http.Request) {
	// Extract component name and props from URL/query params
	// This is a simplified implementation
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<html><body><h1>Component Preview</h1></body></html>"))
}
