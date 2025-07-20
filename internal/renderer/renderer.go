// Package renderer provides component rendering and template processing
// capabilities for templ components.
//
// The renderer handles compilation of .templ files to Go code, manages
// template processing with mock data support, and provides HTML generation
// for component preview. It integrates with the build pipeline for secure
// command execution and supports both individual component rendering and
// batch processing with proper error handling and validation.
package renderer

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/conneroisu/templar/internal/registry"
)

// ComponentRenderer handles rendering of templ components
type ComponentRenderer struct {
	registry *registry.ComponentRegistry
	workDir  string
}

// NewComponentRenderer creates a new component renderer
func NewComponentRenderer(registry *registry.ComponentRegistry) *ComponentRenderer {
	workDir := ".templar/render"
	if err := os.MkdirAll(workDir, 0750); err != nil {
		log.Printf("Failed to create work directory %s: %v", workDir, err)
		// Continue with renderer creation - work directory creation can be retried later
	}

	return &ComponentRenderer{
		registry: registry,
		workDir:  workDir,
	}
}

// RenderComponent renders a specific component with mock data
func (r *ComponentRenderer) RenderComponent(componentName string) (string, error) {
	// Validate component name to prevent path traversal
	if err := r.validateComponentName(componentName); err != nil {
		return "", fmt.Errorf("invalid component name: %w", err)
	}

	component, exists := r.registry.Get(componentName)
	if !exists {
		return "", fmt.Errorf("component %s not found", componentName)
	}

	// Create a clean workspace for this component
	componentWorkDir := filepath.Join(r.workDir, componentName)

	// Validate the work directory path before operations
	if err := r.validateWorkDir(componentWorkDir); err != nil {
		return "", fmt.Errorf("invalid work directory: %w", err)
	}

	if err := os.RemoveAll(componentWorkDir); err != nil {
		log.Printf("Failed to remove existing component work directory %s: %v", componentWorkDir, err)
		// Continue - directory removal failure is not critical if we can still create the new one
	}
	if err := os.MkdirAll(componentWorkDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create component work directory %s: %w", componentWorkDir, err)
	}

	// Generate mock data for parameters
	mockData := r.generateMockData(component)

	// Create a Go file that renders the component
	goCode, err := r.generateGoCode(component, mockData)
	if err != nil {
		return "", fmt.Errorf("generating Go code: %w", err)
	}

	// Write the Go file
	goFile := filepath.Join(componentWorkDir, "main.go")
	if err := os.WriteFile(goFile, []byte(goCode), 0600); err != nil {
		return "", fmt.Errorf("writing Go file: %w", err)
	}

	// Copy and modify the templ file to use main package
	templFile := filepath.Join(componentWorkDir, filepath.Base(component.FilePath))
	if err := r.copyAndModifyTemplFile(component.FilePath, templFile); err != nil {
		return "", fmt.Errorf("copying templ file: %w", err)
	}

	// Run templ generate
	if err := r.runTemplGenerate(componentWorkDir); err != nil {
		return "", fmt.Errorf("running templ generate in %s: %w", componentWorkDir, err)
	}

	// Build and run the Go program
	html, err := r.buildAndRun(componentWorkDir)
	if err != nil {
		return "", fmt.Errorf("building and running component %s: %w", componentName, err)
	}

	return html, nil
}

// generateMockData creates mock data for component parameters
func (r *ComponentRenderer) generateMockData(component *registry.ComponentInfo) map[string]interface{} {
	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		switch param.Type {
		case "string":
			mockData[param.Name] = r.generateMockString(param.Name)
		case "int", "int64", "int32":
			mockData[param.Name] = 42
		case "bool":
			mockData[param.Name] = true
		case "[]string":
			mockData[param.Name] = []string{"Item 1", "Item 2", "Item 3"}
		default:
			mockData[param.Name] = fmt.Sprintf("mock_%s", param.Name)
		}
	}

	return mockData
}

// generateMockString generates realistic mock strings based on parameter name
func (r *ComponentRenderer) generateMockString(paramName string) string {
	switch strings.ToLower(paramName) {
	case "title", "heading":
		return "Sample Title"
	case "name", "username":
		return "John Doe"
	case "email":
		return "john@example.com"
	case "message", "content", "text":
		return "This is sample content for the component preview. Lorem ipsum dolor sit amet, consectetur adipiscing elit."
	case "url", "link", "href":
		return "https://example.com"
	case "variant", "type", "kind":
		return "primary"
	case "color":
		return "blue"
	case "size":
		return "medium"
	default:
		return fmt.Sprintf("Sample %s", strings.Title(paramName))
	}
}

// generateGoCode creates Go code that renders the component
func (r *ComponentRenderer) generateGoCode(component *registry.ComponentInfo, mockData map[string]interface{}) (string, error) {
	tmplStr := `package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	component := {{.ComponentName}}({{range $i, $param := .Parameters}}{{if $i}}, {{end}}{{.MockValue}}{{end}})
	
	err := component.Render(ctx, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering component: %v\n", err)
		os.Exit(1)
	}
}
`

	tmpl, err := template.New("go").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	// Prepare template data
	templateData := struct {
		ComponentName string
		Parameters    []struct {
			Name      string
			MockValue string
		}
	}{
		ComponentName: component.Name,
	}

	for _, param := range component.Parameters {
		mockValue := mockData[param.Name]
		var mockValueStr string

		switch v := mockValue.(type) {
		case string:
			mockValueStr = fmt.Sprintf(`"%s"`, v)
		case int:
			mockValueStr = fmt.Sprintf("%d", v)
		case bool:
			mockValueStr = fmt.Sprintf("%t", v)
		case []string:
			mockValueStr = fmt.Sprintf(`[]string{%s}`, strings.Join(func() []string {
				var quoted []string
				for _, s := range v {
					quoted = append(quoted, fmt.Sprintf(`"%s"`, s))
				}
				return quoted
			}(), ", "))
		default:
			mockValueStr = fmt.Sprintf(`"%v"`, v)
		}

		templateData.Parameters = append(templateData.Parameters, struct {
			Name      string
			MockValue string
		}{
			Name:      param.Name,
			MockValue: mockValueStr,
		})
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// copyFile copies a file from src to dst
// copyAndModifyTemplFile copies a templ file and modifies it to use main package
func (r *ComponentRenderer) copyAndModifyTemplFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	content := string(input)
	lines := strings.Split(content, "\n")

	// Modify the package declaration to use main
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			lines[i] = "package main"
			break
		}
	}

	modifiedContent := strings.Join(lines, "\n")
	return os.WriteFile(dst, []byte(modifiedContent), 0600)
}

// runTemplGenerate runs templ generate in the work directory
func (r *ComponentRenderer) runTemplGenerate(workDir string) error {
	// Validate work directory path to prevent directory traversal
	if err := r.validateWorkDir(workDir); err != nil {
		return fmt.Errorf("invalid work directory: %w", err)
	}

	// Check if templ command is available
	if _, err := exec.LookPath("templ"); err != nil {
		return fmt.Errorf("templ command not found: %w. Please install it with: go install github.com/a-h/templ/cmd/templ@v0.3.819", err)
	}

	cmd := exec.Command("templ", "generate")
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("templ generate failed in directory %s: %w\nOutput: %s", workDir, err, output)
	}
	return nil
}

// buildAndRun builds and runs the Go program to generate HTML
func (r *ComponentRenderer) buildAndRun(workDir string) (string, error) {
	// Validate work directory path to prevent directory traversal
	if err := r.validateWorkDir(workDir); err != nil {
		return "", fmt.Errorf("invalid work directory: %w", err)
	}

	// Check if go command is available
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go command not found: %w", err)
	}

	// Initialize go module if it doesn't exist
	goModPath := filepath.Join(workDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "mod", "init", "templar-render")
		cmd.Dir = workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("go mod init failed in %s: %w\nOutput: %s", workDir, err, output)
		}

		// Add templ dependency
		cmd = exec.Command("go", "get", "github.com/a-h/templ")
		cmd.Dir = workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("go get templ failed in %s: %w\nOutput: %s", workDir, err, output)
		}
	}

	// Build and run
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go run failed in %s: %w\nOutput: %s", workDir, err, output)
	}

	return string(output), nil
}

// validateWorkDir validates the work directory path to prevent directory traversal
func (r *ComponentRenderer) validateWorkDir(workDir string) error {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(workDir)

	// Get absolute path to normalize
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Ensure the work directory is within the current working directory
	if !strings.HasPrefix(absPath, cwd) {
		return fmt.Errorf("work directory %s is outside current working directory", workDir)
	}

	// Additional security check: reject paths with suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("work directory contains directory traversal: %s", workDir)
	}

	return nil
}

// RenderComponentWithLayout wraps component HTML in a full page layout
func (r *ComponentRenderer) RenderComponentWithLayout(componentName string, html string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Templar Preview</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#007acc',
                        secondary: '#6c757d'
                    }
                }
            }
        }
    </script>
    <style>
        .btn { @apply px-4 py-2 rounded-md font-medium transition-colors; }
        .btn-primary { @apply bg-blue-600 text-white hover:bg-blue-700; }
        .btn-secondary { @apply bg-gray-200 text-gray-800 hover:bg-gray-300; }
        .card { @apply bg-white rounded-lg shadow-md p-6; }
        .card-header { @apply border-b border-gray-200 pb-4 mb-4; }
        .card-body { @apply text-gray-700; }
        .card-footer { @apply border-t border-gray-200 pt-4 mt-4 flex space-x-2; }
    </style>
</head>
<body class="bg-gray-50 p-8">
    <div class="max-w-4xl mx-auto">
        <div class="bg-white rounded-lg shadow-lg p-6 mb-6">
            <h1 class="text-2xl font-bold text-gray-800 mb-2">Preview: %s</h1>
            <p class="text-gray-600 text-sm">Live preview with Tailwind CSS styling</p>
        </div>
        
        <div class="bg-white rounded-lg shadow-lg p-6">
            %s
        </div>
    </div>
    
    <script>
        // WebSocket connection for live reload
        const ws = new WebSocket('ws://localhost:' + window.location.port + '/ws');
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            if (message.type === 'full_reload') {
                window.location.reload();
            }
        };
    </script>
</body>
</html>`, componentName, componentName, html)
}

// validateComponentName validates component name to prevent path traversal
func (r *ComponentRenderer) validateComponentName(name string) error {
	// Clean the name
	cleanName := filepath.Clean(name)

	// Reject names containing path traversal patterns
	if strings.Contains(cleanName, "..") {
		return fmt.Errorf("path traversal attempt detected: %s", name)
	}

	// Reject absolute paths
	if filepath.IsAbs(cleanName) {
		return fmt.Errorf("absolute path not allowed: %s", name)
	}

	// Reject names with path separators (should be simple component names)
	if strings.ContainsRune(cleanName, os.PathSeparator) {
		return fmt.Errorf("path separators not allowed in component name: %s", name)
	}

	// Reject empty names
	if cleanName == "" || cleanName == "." {
		return fmt.Errorf("empty or invalid component name: %s", name)
	}

	return nil
}
