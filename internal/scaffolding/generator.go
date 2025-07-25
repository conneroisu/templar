package scaffolding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// ComponentGenerator handles component scaffolding
type ComponentGenerator struct {
	templates   map[string]ComponentTemplate
	outputDir   string
	packageName string
	projectName string
	author      string
}

// GenerateOptions holds options for component generation
type GenerateOptions struct {
	Name        string
	Template    string
	OutputDir   string
	PackageName string
	ProjectName string
	Author      string
	WithTests   bool
	WithDocs    bool
	WithStyles  bool
	CustomProps map[string]interface{}
}

// NewComponentGenerator creates a new component generator
func NewComponentGenerator(outputDir, packageName, projectName, author string) *ComponentGenerator {
	return &ComponentGenerator{
		templates:   GetBuiltinTemplates(),
		outputDir:   outputDir,
		packageName: packageName,
		projectName: projectName,
		author:      author,
	}
}

// Generate creates a new component from a template
func (g *ComponentGenerator) Generate(opts GenerateOptions) error {
	// Set defaults
	if opts.OutputDir == "" {
		opts.OutputDir = g.outputDir
	}
	if opts.PackageName == "" {
		opts.PackageName = g.packageName
	}
	if opts.ProjectName == "" {
		opts.ProjectName = g.projectName
	}
	if opts.Author == "" {
		opts.Author = g.author
	}

	// Get template
	tmpl, exists := g.templates[opts.Template]
	if !exists {
		return fmt.Errorf("template '%s' not found", opts.Template)
	}

	// Create template context
	ctx := TemplateContext{
		ComponentName: capitalizeFirst(opts.Name),
		PackageName:   opts.PackageName,
		Parameters:    tmpl.Parameters,
		Author:        opts.Author,
		Date:          time.Now().Format("2006-01-02"),
		ProjectName:   opts.ProjectName,
		CustomProps:   opts.CustomProps,
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main component file
	componentFile := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.templ", strings.ToLower(opts.Name)))
	if err := g.generateFile(componentFile, tmpl.Content, ctx); err != nil {
		return fmt.Errorf("failed to generate component file: %w", err)
	}

	fmt.Printf("✅ Generated component: %s\n", componentFile)

	// Generate test file if requested
	if opts.WithTests && tmpl.TestContent != "" {
		testFile := filepath.Join(opts.OutputDir, fmt.Sprintf("%s_test.go", strings.ToLower(opts.Name)))
		if err := g.generateFile(testFile, tmpl.TestContent, ctx); err != nil {
			return fmt.Errorf("failed to generate test file: %w", err)
		}
		fmt.Printf("✅ Generated tests: %s\n", testFile)
	}

	// Generate styles file if requested
	if opts.WithStyles && tmpl.StylesCSS != "" {
		stylesDir := filepath.Join(opts.OutputDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			return fmt.Errorf("failed to create styles directory: %w", err)
		}

		stylesFile := filepath.Join(stylesDir, fmt.Sprintf("%s.css", strings.ToLower(opts.Name)))
		if err := os.WriteFile(stylesFile, []byte(tmpl.StylesCSS), 0644); err != nil {
			return fmt.Errorf("failed to generate styles file: %w", err)
		}
		fmt.Printf("✅ Generated styles: %s\n", stylesFile)
	}

	// Generate documentation if requested
	if opts.WithDocs && tmpl.DocContent != "" {
		docsDir := filepath.Join(opts.OutputDir, "docs")
		if err := os.MkdirAll(docsDir, 0755); err != nil {
			return fmt.Errorf("failed to create docs directory: %w", err)
		}

		docFile := filepath.Join(docsDir, fmt.Sprintf("%s.md", strings.ToLower(opts.Name)))
		if err := g.generateFile(docFile, tmpl.DocContent, ctx); err != nil {
			return fmt.Errorf("failed to generate documentation: %w", err)
		}
		fmt.Printf("✅ Generated docs: %s\n", docFile)
	}

	return nil
}

// ListTemplates returns available templates
func (g *ComponentGenerator) ListTemplates() []TemplateInfo {
	var templates []TemplateInfo
	for name, tmpl := range g.templates {
		templates = append(templates, TemplateInfo{
			Name:        name,
			Description: tmpl.Description,
			Category:    tmpl.Category,
			Parameters:  len(tmpl.Parameters),
		})
	}
	return templates
}

// GetTemplate returns a specific template
func (g *ComponentGenerator) GetTemplate(name string) (ComponentTemplate, bool) {
	tmpl, exists := g.templates[name]
	return tmpl, exists
}

// TemplateInfo holds basic template information
type TemplateInfo struct {
	Name        string
	Description string
	Category    string
	Parameters  int
}

// generateFile generates a file from a template
func (g *ComponentGenerator) generateFile(filename, content string, ctx TemplateContext) error {
	// Parse template
	tmpl, err := template.New("component").Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, ctx); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// AddCustomTemplate adds a custom template
func (g *ComponentGenerator) AddCustomTemplate(name string, tmpl ComponentTemplate) {
	g.templates[name] = tmpl
}

// ValidateComponentName checks if a component name is valid
func ValidateComponentName(name string) error {
	if name == "" {
		return fmt.Errorf("component name cannot be empty")
	}

	// Check for valid Go identifier
	if !isValidGoIdentifier(name) {
		return fmt.Errorf("component name must be a valid Go identifier")
	}

	// Check if name starts with uppercase (exported)
	if !isUppercase(name[0]) {
		return fmt.Errorf("component name must start with uppercase letter (exported)")
	}

	return nil
}

// GenerateComponentSet generates multiple related components
func (g *ComponentGenerator) GenerateComponentSet(setName string, components []string, outputDir string) error {
	fmt.Printf("🏗️  Generating component set: %s\n", setName)

	for _, component := range components {
		opts := GenerateOptions{
			Name:       component,
			Template:   strings.ToLower(component),
			OutputDir:  outputDir,
			WithTests:  true,
			WithStyles: true,
			WithDocs:   true,
		}

		if err := g.Generate(opts); err != nil {
			return fmt.Errorf("failed to generate component %s: %w", component, err)
		}
	}

	fmt.Printf("✅ Component set '%s' generated successfully\n", setName)
	return nil
}

// GetTemplatesByCategory returns templates grouped by category
func (g *ComponentGenerator) GetTemplatesByCategory() map[string][]TemplateInfo {
	categories := make(map[string][]TemplateInfo)

	for name, tmpl := range g.templates {
		info := TemplateInfo{
			Name:        name,
			Description: tmpl.Description,
			Category:    tmpl.Category,
			Parameters:  len(tmpl.Parameters),
		}
		categories[tmpl.Category] = append(categories[tmpl.Category], info)
	}

	return categories
}

// Helper functions

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func isValidGoIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// First character must be letter or underscore
	first := name[0]
	if !isLetter(first) && first != '_' {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for _, r := range name[1:] {
		if !isLetter(byte(r)) && !isDigit(byte(r)) && r != '_' {
			return false
		}
	}

	return true
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isUppercase(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

// CreateProjectScaffold creates a complete project structure with common components
func (g *ComponentGenerator) CreateProjectScaffold(projectDir string) error {
	fmt.Println("🏗️  Creating project scaffold...")

	// Create directory structure
	dirs := []string{
		"components/ui",
		"components/layout",
		"components/forms",
		"views",
		"styles",
		"docs/components",
		"examples",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate essential components
	essentialComponents := map[string]string{
		"Button":     "components/ui",
		"Card":       "components/ui",
		"Layout":     "components/layout",
		"Form":       "components/forms",
		"Navigation": "components/layout",
	}

	for component, dir := range essentialComponents {
		opts := GenerateOptions{
			Name:       component,
			Template:   strings.ToLower(component),
			OutputDir:  filepath.Join(projectDir, dir),
			WithTests:  true,
			WithStyles: true,
			WithDocs:   true,
		}

		if err := g.Generate(opts); err != nil {
			fmt.Printf("⚠️  Warning: failed to generate %s: %v\n", component, err)
		}
	}

	// Create main styles file
	mainStylesPath := filepath.Join(projectDir, "styles", "main.css")
	mainStyles := `/* Main styles for the project */
@import url('./components/');

/* Reset and base styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  line-height: 1.6;
  color: #1f2937;
  background-color: #ffffff;
}

/* Utility classes */
.container {
  max-width: 80rem;
  margin: 0 auto;
  padding: 0 1rem;
}

.text-center { text-align: center; }
.text-left { text-align: left; }
.text-right { text-align: right; }

.mt-1 { margin-top: 0.25rem; }
.mt-2 { margin-top: 0.5rem; }
.mt-4 { margin-top: 1rem; }
.mt-8 { margin-top: 2rem; }

.mb-1 { margin-bottom: 0.25rem; }
.mb-2 { margin-bottom: 0.5rem; }
.mb-4 { margin-bottom: 1rem; }
.mb-8 { margin-bottom: 2rem; }

.p-2 { padding: 0.5rem; }
.p-4 { padding: 1rem; }
.p-8 { padding: 2rem; }

.flex { display: flex; }
.flex-col { flex-direction: column; }
.items-center { align-items: center; }
.justify-center { justify-content: center; }
.justify-between { justify-content: space-between; }

.gap-2 { gap: 0.5rem; }
.gap-4 { gap: 1rem; }
.gap-8 { gap: 2rem; }
`

	if err := os.WriteFile(mainStylesPath, []byte(mainStyles), 0644); err != nil {
		return fmt.Errorf("failed to create main styles: %w", err)
	}

	fmt.Printf("✅ Project scaffold created in %s\n", projectDir)
	fmt.Println("📁 Created directories:")
	for _, dir := range dirs {
		fmt.Printf("   - %s\n", dir)
	}
	fmt.Println("🧩 Generated components:")
	for component := range essentialComponents {
		fmt.Printf("   - %s\n", component)
	}

	return nil
}
