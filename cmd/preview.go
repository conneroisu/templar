package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/server"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview <component>",
	Short: "Preview a specific component in isolation",
	Long: `Preview a specific component in isolation with optional mock data.
This starts a lightweight server to preview just the specified component
with configurable properties and mock data.

Examples:
  templar preview Button                           # Preview Button component
  templar preview Button --props '{"text":"Click me"}' # Preview with props
  templar preview Card --mock ./mocks/card.json   # Preview with mock data
  templar preview Button --wrapper ./layout.templ # Preview with custom wrapper`,
	Args: cobra.ExactArgs(1),
	RunE: runPreview,
}

var (
	previewMock    string
	previewWrapper string
	previewProps   string
	previewPort    int
	previewNoOpen  bool
)

func init() {
	rootCmd.AddCommand(previewCmd)

	previewCmd.Flags().StringVarP(&previewMock, "mock", "m", "", "Mock data file or pattern")
	previewCmd.Flags().StringVarP(&previewWrapper, "wrapper", "w", "", "Wrapper template")
	previewCmd.Flags().StringVarP(&previewProps, "props", "p", "", "Component properties (JSON)")
	previewCmd.Flags().IntVar(&previewPort, "port", 8080, "Port to serve on")
	previewCmd.Flags().BoolVar(&previewNoOpen, "no-open", false, "Don't open browser automatically")
}

func runPreview(cmd *cobra.Command, args []string) error {
	componentName := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override server config for preview
	cfg.Server.Port = previewPort
	cfg.Server.Open = !previewNoOpen

	// Create component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Scan all configured paths
	fmt.Println("📁 Scanning for components...")
	for _, scanPath := range cfg.Components.ScanPaths {
		if err := componentScanner.ScanDirectory(scanPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", scanPath, err)
		}
	}

	// Find the requested component
	component, exists := componentRegistry.Get(componentName)
	if !exists {
		// Create enhanced error with suggestions
		ctx := &errors.SuggestionContext{
			Registry:       componentRegistry,
			ConfigPath:     ".templar.yml",
			ComponentsPath: cfg.Components.ScanPaths,
		}
		suggestions := errors.ComponentNotFoundError(componentName, ctx)
		enhancedErr := errors.NewEnhancedError(
			fmt.Sprintf("Component '%s' not found", componentName),
			fmt.Errorf("component not found"),
			suggestions,
		)
		return enhancedErr
	}

	fmt.Printf("🎭 Previewing component: %s\n", componentName)
	fmt.Printf("   File: %s\n", component.FilePath)
	fmt.Printf("   Package: %s\n", component.Package)

	// Parse component properties if provided
	var props map[string]interface{}
	if previewProps != "" {
		// Validate JSON size to prevent resource exhaustion attacks
		if len(previewProps) > 1024*1024 { // 1MB limit
			return fmt.Errorf("props JSON too large (max 1MB)")
		}

		if err := json.Unmarshal([]byte(previewProps), &props); err != nil {
			return fmt.Errorf("failed to parse props JSON: %w", err)
		}
	}

	// Load mock data if specified
	var mockData map[string]interface{}
	if previewMock != "" {
		mockData, err = loadMockData(previewMock)
		if err != nil {
			return fmt.Errorf("failed to load mock data: %w", err)
		}
	}

	// Generate mock data if not provided
	if mockData == nil && props == nil {
		mockData = generateMockData(component)
		fmt.Println("🎲 Generated mock data for component parameters")
	}

	// Create preview-specific server
	srv, err := createPreviewServer(cfg, component, props, mockData)
	if err != nil {
		return fmt.Errorf("failed to create preview server: %w", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("🚀 Starting preview server at http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// Handle graceful shutdown
	go func() {
		if err := srv.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	// Keep the server running
	select {
	case <-ctx.Done():
		return nil
	}
}

func loadMockData(mockFile string) (map[string]interface{}, error) {
	// Validate mock file path for security
	if err := validateMockFilePath(mockFile); err != nil {
		return nil, fmt.Errorf("invalid mock file path: %w", err)
	}

	data, err := os.ReadFile(mockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock file: %w", err)
	}

	var mockData map[string]interface{}
	if err := json.Unmarshal(data, &mockData); err != nil {
		return nil, fmt.Errorf("failed to parse mock data JSON: %w", err)
	}

	return mockData, nil
}

func generateMockData(component *registry.ComponentInfo) map[string]interface{} {
	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		mockData[param.Name] = generateMockValue(param.Type)
	}

	return mockData
}

func generateMockValue(paramType string) interface{} {
	switch strings.ToLower(paramType) {
	case "string":
		return "Mock Text"
	case "int", "int32", "int64":
		return 42
	case "float32", "float64":
		return 3.14
	case "bool":
		return true
	case "[]string":
		return []string{"Item 1", "Item 2", "Item 3"}
	case "[]int":
		return []int{1, 2, 3}
	default:
		if strings.HasPrefix(paramType, "[]") {
			return []interface{}{"Mock Item"}
		}
		return "Mock Value"
	}
}

func createPreviewServer(cfg *config.Config, component *registry.ComponentInfo, props map[string]interface{}, mockData map[string]interface{}) (*server.PreviewServer, error) {
	// Create a new registry with just the preview component
	previewRegistry := registry.NewComponentRegistry()
	previewRegistry.Register(component)

	// Create preview server
	srv, err := server.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Create custom renderer for preview
	previewRenderer := renderer.NewComponentRenderer(previewRegistry)

	// Generate preview HTML
	html, err := generatePreviewHTML(component, props, mockData, previewRenderer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview HTML: %w", err)
	}

	// Store preview HTML for serving
	// In a real implementation, this would be integrated with the server
	previewPath := filepath.Join(".templar", "preview.html")
	if err := os.MkdirAll(filepath.Dir(previewPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create preview directory: %w", err)
	}

	if err := os.WriteFile(previewPath, []byte(html), 0644); err != nil {
		return nil, fmt.Errorf("failed to write preview HTML: %w", err)
	}

	return srv, nil
}

func generatePreviewHTML(component *registry.ComponentInfo, props map[string]interface{}, mockData map[string]interface{}, renderer *renderer.ComponentRenderer) (string, error) {
	// Use provided props or generated mock data
	data := props
	if data == nil {
		data = mockData
	}

	// Generate component HTML
	componentHTML, err := renderer.RenderComponent(component.Name)
	if err != nil {
		return "", fmt.Errorf("failed to render component: %w", err)
	}

	// Create wrapper HTML
	wrapperHTML := generateWrapperHTML(component, data, componentHTML)

	return wrapperHTML, nil
}

func generateWrapperHTML(component *registry.ComponentInfo, data map[string]interface{}, componentHTML string) string {
	// Create a simple wrapper if custom wrapper is not specified
	if previewWrapper == "" {
		return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Preview: %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 2rem;
            background-color: #f8f9fa;
        }
        .preview-container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .preview-header {
            border-bottom: 1px solid #eee;
            padding-bottom: 1rem;
            margin-bottom: 2rem;
        }
        .preview-title {
            font-size: 1.5rem;
            font-weight: 600;
            color: #333;
            margin: 0 0 0.5rem 0;
        }
        .preview-info {
            font-size: 0.875rem;
            color: #666;
        }
        .preview-content {
            margin-top: 1rem;
        }
        .preview-props {
            margin-top: 2rem;
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 4px;
            border-left: 4px solid #007bff;
        }
        .preview-props h3 {
            margin: 0 0 0.5rem 0;
            font-size: 1rem;
            color: #333;
        }
        .preview-props pre {
            margin: 0;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.875rem;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="preview-container">
        <div class="preview-header">
            <h1 class="preview-title">%s</h1>
            <div class="preview-info">
                <strong>Package:</strong> %s<br>
                <strong>File:</strong> %s<br>
                <strong>Function:</strong> %s
            </div>
        </div>
        <div class="preview-content">
            %s
        </div>
        <div class="preview-props">
            <h3>Component Properties</h3>
            <pre>%s</pre>
        </div>
    </div>
    <script>
        // WebSocket connection for hot reload
        const ws = new WebSocket('ws://localhost:%d/ws');
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            if (message.type === 'full_reload') {
                location.reload();
            }
        };
    </script>
</body>
</html>`,
			component.Name,
			component.Name,
			component.Package,
			component.FilePath,
			component.Name,
			componentHTML,
			formatJSON(data),
			previewPort,
		)
	}

	// Use custom wrapper (placeholder for now)
	return fmt.Sprintf(`<!-- Custom wrapper would be loaded from %s -->
%s`, previewWrapper, componentHTML)
}

func formatJSON(data interface{}) string {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	return string(jsonData)
}

// validateMockFilePath validates mock file paths to prevent security vulnerabilities
func validateMockFilePath(mockFile string) error {
	if mockFile == "" {
		return fmt.Errorf("empty mock file path")
	}

	// Clean the path
	cleanPath := filepath.Clean(mockFile)

	// Reject path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal attempt detected: %s", mockFile)
	}

	// Only allow relative paths within the current directory and its subdirectories
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute paths not allowed: %s", mockFile)
	}

	// Reject dangerous characters that could be used for injection
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return fmt.Errorf("path contains dangerous character '%s': %s", char, mockFile)
		}
	}

	// Limit file extension to JSON for security
	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != ".json" {
		return fmt.Errorf("only JSON files are allowed for mock data: %s", mockFile)
	}

	return nil
}
