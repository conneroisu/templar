package scaffolding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComponentGenerator tests the ComponentGenerator functionality.
func TestComponentGenerator(t *testing.T) {
	t.Run("NewComponentGenerator", func(t *testing.T) {
		t.Run("creates generator with valid parameters", func(t *testing.T) {
			outputDir := "/tmp/test"
			packageName := "components"
			projectName := "test-project"
			author := "Test Author"

			generator := NewComponentGenerator(outputDir, packageName, projectName, author)

			assert.NotNil(t, generator)
			assert.Equal(t, outputDir, generator.outputDir)
			assert.Equal(t, packageName, generator.packageName)
			assert.Equal(t, projectName, generator.projectName)
			assert.Equal(t, author, generator.author)
			assert.NotNil(t, generator.templates)
			assert.Greater(t, len(generator.templates), 0, "Should have built-in templates")
		})

		t.Run("includes built-in templates", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			// Verify some expected built-in templates exist
			expectedTemplates := []string{"button", "card", "form", "layout", "navigation"}
			for _, templateName := range expectedTemplates {
				assert.Contains(t, generator.templates, templateName, "Should contain template: %s", templateName)
			}
		})
	})

	t.Run("Generate", func(t *testing.T) {
		tc := testutils.NewTestContext(t)
		fs := testutils.NewTestFileSystem(t, tc.TempDir())

		t.Run("generates component with default options", func(t *testing.T) {
			generator := NewComponentGenerator(tc.TempDir(), "components", "test-project", "Test Author")

			opts := GenerateOptions{
				Name:     "TestButton",
				Template: "button",
			}

			err := generator.Generate(opts)
			require.NoError(t, err)

			// Verify component file was created
			expectedFile := filepath.Join(tc.TempDir(), "testbutton.templ")
			fs.AssertFileExists("testbutton.templ")

			// Verify file content contains expected elements
			content, err := os.ReadFile(expectedFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "TestButton", "Should contain component name")
			assert.Contains(t, contentStr, "components", "Should contain package name")
		})

		t.Run("generates component with custom options", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			customOutputDir := filepath.Join(tc.TempDir(), "custom")
			opts := GenerateOptions{
				Name:        "CustomCard",
				Template:    "card",
				OutputDir:   customOutputDir,
				PackageName: "custom",
				ProjectName: "custom-project",
				Author:      "Custom Author",
				WithTests:   true,
				WithDocs:    true,
				WithStyles:  true,
				CustomProps: map[string]interface{}{
					"version": "1.0.0",
					"license": "MIT",
				},
			}

			err := generator.Generate(opts)
			require.NoError(t, err)

			// Verify main component file
			componentFile := filepath.Join(customOutputDir, "customcard.templ")
			assert.FileExists(t, componentFile)

			// Verify test file if template supports it
			testFile := filepath.Join(customOutputDir, "customcard_test.go")
			if generator.templates["card"].TestContent != "" {
				assert.FileExists(t, testFile)
			}

			// Verify docs file if template supports it
			docsFile := filepath.Join(customOutputDir, "docs", "customcard.md")
			if generator.templates["card"].DocContent != "" {
				assert.FileExists(t, docsFile)
			}

			// Verify styles file if template supports it
			stylesFile := filepath.Join(customOutputDir, "styles", "customcard.css")
			if generator.templates["card"].StylesCSS != "" {
				assert.FileExists(t, stylesFile)
			}
		})

		t.Run("uses generator defaults when options are empty", func(t *testing.T) {
			generator := NewComponentGenerator(tc.TempDir(), "mypackage", "myproject", "My Author")

			opts := GenerateOptions{
				Name:     "DefaultTest",
				Template: "button",
				// All other options empty to test defaults
			}

			err := generator.Generate(opts)
			require.NoError(t, err)

			// Verify file was created in generator's output dir
			expectedFile := filepath.Join(tc.TempDir(), "defaulttest.templ")
			assert.FileExists(t, expectedFile)

			// Verify content uses generator defaults
			content, err := os.ReadFile(expectedFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "mypackage", "Should use generator's package name")
		})

		t.Run("returns error for unknown template", func(t *testing.T) {
			generator := NewComponentGenerator(tc.TempDir(), "components", "test", "author")

			opts := GenerateOptions{
				Name:     "Test",
				Template: "nonexistent",
			}

			err := generator.Generate(opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "template 'nonexistent' not found")
		})

		t.Run("handles file creation errors gracefully", func(t *testing.T) {
			// Use a read-only directory
			readOnlyDir := filepath.Join(tc.TempDir(), "readonly")
			require.NoError(t, os.MkdirAll(readOnlyDir, 0444)) // read-only

			generator := NewComponentGenerator(readOnlyDir, "components", "test", "author")

			opts := GenerateOptions{
				Name:     "Test",
				Template: "button",
			}

			err := generator.Generate(opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create file")
		})

		t.Run("creates nested directories when needed", func(t *testing.T) {
			nestedDir := filepath.Join(tc.TempDir(), "deeply", "nested", "components")
			generator := NewComponentGenerator(nestedDir, "components", "test", "author")

			opts := GenerateOptions{
				Name:     "NestedTest",
				Template: "button",
			}

			err := generator.Generate(opts)
			require.NoError(t, err)

			expectedFile := filepath.Join(nestedDir, "nestedtest.templ")
			assert.FileExists(t, expectedFile)
		})
	})

	t.Run("ListTemplates", func(t *testing.T) {
		t.Run("returns all available templates", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			templates := generator.ListTemplates()

			assert.Greater(t, len(templates), 0, "Should return templates")

			// Verify template info structure
			for _, tmpl := range templates {
				assert.NotEmpty(t, tmpl.Name, "Template should have a name")
				assert.NotEmpty(t, tmpl.Description, "Template should have a description")
				assert.GreaterOrEqual(t, tmpl.Parameters, 0, "Template should have parameter count")
			}
		})

		t.Run("includes expected built-in templates", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			templates := generator.ListTemplates()
			templateNames := make([]string, len(templates))
			for i, tmpl := range templates {
				templateNames[i] = tmpl.Name
			}

			expectedTemplates := []string{"button", "card", "form", "layout"}
			for _, expected := range expectedTemplates {
				assert.Contains(t, templateNames, expected, "Should include template: %s", expected)
			}
		})
	})

	t.Run("GetTemplate", func(t *testing.T) {
		generator := NewComponentGenerator("", "", "", "")

		t.Run("returns existing template", func(t *testing.T) {
			tmpl, exists := generator.GetTemplate("button")

			assert.True(t, exists, "Button template should exist")
			assert.Equal(t, "button", tmpl.Name)
			assert.NotEmpty(t, tmpl.Description)
			assert.NotEmpty(t, tmpl.Content)
		})

		t.Run("returns false for non-existent template", func(t *testing.T) {
			_, exists := generator.GetTemplate("nonexistent")

			assert.False(t, exists, "Non-existent template should return false")
		})
	})

	t.Run("AddCustomTemplate", func(t *testing.T) {
		t.Run("adds and retrieves custom template", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			customTemplate := ComponentTemplate{
				Name:        "custom",
				Description: "Custom test template",
				Category:    "test",
				Parameters: []TemplateParameter{
					{Name: "name", Type: "string", Required: true},
				},
				Content: "package {{.PackageName}}\n\ntempl {{.ComponentName}}() {\n\t<div>Custom</div>\n}",
			}

			generator.AddCustomTemplate("custom", customTemplate)

			// Verify template was added
			tmpl, exists := generator.GetTemplate("custom")
			assert.True(t, exists)
			assert.Equal(t, customTemplate.Name, tmpl.Name)
			assert.Equal(t, customTemplate.Description, tmpl.Description)

			// Verify it appears in template list
			templates := generator.ListTemplates()
			templateNames := make([]string, len(templates))
			for i, t := range templates {
				templateNames[i] = t.Name
			}
			assert.Contains(t, templateNames, "custom")
		})

		t.Run("overwrites existing template", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			// Get original button template
			originalButton, _ := generator.GetTemplate("button")

			// Create modified template
			modifiedButton := originalButton
			modifiedButton.Description = "Modified button template"

			generator.AddCustomTemplate("button", modifiedButton)

			// Verify template was overwritten
			tmpl, exists := generator.GetTemplate("button")
			assert.True(t, exists)
			assert.Equal(t, "Modified button template", tmpl.Description)
		})
	})

	t.Run("generateFile", func(t *testing.T) {
		tc := testutils.NewTestContext(t)

		t.Run("generates file from template", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			templateContent := "package {{.PackageName}}\n\ntempl {{.ComponentName}}() {\n\t<div>Hello</div>\n}"
			ctx := TemplateContext{
				ComponentName: "TestComponent",
				PackageName:   "components",
			}

			filename := filepath.Join(tc.TempDir(), "test.templ")
			err := generator.generateFile(filename, templateContent, ctx)
			require.NoError(t, err)

			// Verify file content
			content, err := os.ReadFile(filename)
			require.NoError(t, err)

			expected := "package components\n\ntempl TestComponent() {\n\t<div>Hello</div>\n}"
			assert.Equal(t, expected, string(content))
		})

		t.Run("returns error for invalid template", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			invalidTemplate := "{{.InvalidTemplate"
			ctx := TemplateContext{}

			filename := filepath.Join(tc.TempDir(), "invalid.templ")
			err := generator.generateFile(filename, invalidTemplate, ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse template")
		})

		t.Run("returns error for invalid file path", func(t *testing.T) {
			generator := NewComponentGenerator("", "", "", "")

			templateContent := "{{.ComponentName}}"
			ctx := TemplateContext{ComponentName: "Test"}

			// Use invalid file path
			invalidPath := "/invalid/path/that/does/not/exist/file.templ"
			err := generator.generateFile(invalidPath, templateContent, ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create file")
		})
	})
}

// TestTemplateContext tests template context functionality.
func TestTemplateContext(t *testing.T) {
	t.Run("TemplateContext creation", func(t *testing.T) {
		ctx := TemplateContext{
			ComponentName: "TestComponent",
			PackageName:   "components",
			Parameters:    []TemplateParameter{{Name: "text", Type: "string"}},
			Author:        "Test Author",
			Date:          "2023-01-01",
			ProjectName:   "test-project",
			CustomProps:   map[string]interface{}{"version": "1.0.0"},
		}

		assert.Equal(t, "TestComponent", ctx.ComponentName)
		assert.Equal(t, "components", ctx.PackageName)
		assert.Len(t, ctx.Parameters, 1)
		assert.Equal(t, "Test Author", ctx.Author)
		assert.Equal(t, "test-project", ctx.ProjectName)
		assert.Equal(t, "1.0.0", ctx.CustomProps["version"])
	})
}

// TestBuiltinTemplates tests built-in template functionality.
func TestBuiltinTemplates(t *testing.T) {
	t.Run("GetBuiltinTemplates", func(t *testing.T) {
		templates := GetBuiltinTemplates()

		assert.Greater(t, len(templates), 0, "Should have built-in templates")

		// Test some specific templates
		expectedTemplates := []string{"button", "card", "form", "layout"}
		for _, name := range expectedTemplates {
			tmpl, exists := templates[name]
			assert.True(t, exists, "Should have template: %s", name)
			assert.Equal(t, name, tmpl.Name)
			assert.NotEmpty(t, tmpl.Description)
			assert.NotEmpty(t, tmpl.Content)
		}
	})

	t.Run("individual template getters", func(t *testing.T) {
		// Test that individual template getters work
		buttonTemplate := getButtonTemplate()
		assert.Equal(t, "button", buttonTemplate.Name)
		assert.NotEmpty(t, buttonTemplate.Description)
		assert.NotEmpty(t, buttonTemplate.Content)
		assert.Greater(t, len(buttonTemplate.Parameters), 0, "Button should have parameters")

		// Verify button template has expected parameters
		paramNames := make([]string, len(buttonTemplate.Parameters))
		for i, param := range buttonTemplate.Parameters {
			paramNames[i] = param.Name
		}
		assert.Contains(t, paramNames, "text")
		assert.Contains(t, paramNames, "variant")
	})
}

// TestTemplateValidation tests template validation.
func TestTemplateValidation(t *testing.T) {
	t.Run("template parameter validation", func(t *testing.T) {
		param := TemplateParameter{
			Name:         "text",
			Type:         "string",
			DefaultValue: "default",
			Description:  "Text parameter",
			Required:     true,
		}

		assert.Equal(t, "text", param.Name)
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "default", param.DefaultValue)
		assert.True(t, param.Required)
	})

	t.Run("template content validation", func(t *testing.T) {
		templates := GetBuiltinTemplates()

		for name, tmpl := range templates {
			t.Run(name, func(t *testing.T) {
				// Basic template structure validation
				assert.NotEmpty(t, tmpl.Name, "Template should have a name")
				assert.NotEmpty(t, tmpl.Description, "Template should have a description")
				assert.NotEmpty(t, tmpl.Content, "Template should have content")

				// Verify template content contains expected patterns
				assert.Contains(t, tmpl.Content, "{{.ComponentName}}", "Template should use ComponentName")
				assert.Contains(t, tmpl.Content, "{{.PackageName}}", "Template should use PackageName")

				// Verify parameters are valid
				for _, param := range tmpl.Parameters {
					assert.NotEmpty(t, param.Name, "Parameter should have a name")
					assert.NotEmpty(t, param.Type, "Parameter should have a type")
				}
			})
		}
	})
}

// TestScaffoldingIntegration tests integration scenarios.
func TestScaffoldingIntegration(t *testing.T) {
	tc := testutils.NewTestContext(t)

	t.Run("full component generation workflow", func(t *testing.T) {
		// Create generator
		generator := NewComponentGenerator(tc.TempDir(), "components", "test-project", "Test Author")

		// Generate multiple components
		components := []struct {
			name     string
			template string
		}{
			{"AppButton", "button"},
			{"UserCard", "card"},
			{"ContactForm", "form"},
		}

		for _, comp := range components {
			opts := GenerateOptions{
				Name:      comp.name,
				Template:  comp.template,
				WithTests: true,
				WithDocs:  true,
			}

			err := generator.Generate(opts)
			require.NoError(t, err, "Should generate component: %s", comp.name)

			// Verify main file exists
			componentFile := filepath.Join(tc.TempDir(), strings.ToLower(comp.name)+".templ")
			assert.FileExists(t, componentFile)

			// Verify content is valid
			content, err := os.ReadFile(componentFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, comp.name)
			assert.Contains(t, contentStr, "components")
		}
	})

	t.Run("custom template workflow", func(t *testing.T) {
		generator := NewComponentGenerator(tc.TempDir(), "custom", "custom-project", "Custom Author")

		// Add custom template
		customTemplate := ComponentTemplate{
			Name:        "widget",
			Description: "Custom widget component",
			Category:    "custom",
			Parameters: []TemplateParameter{
				{Name: "title", Type: "string", Required: true},
				{Name: "size", Type: "string", DefaultValue: "medium", Required: false},
			},
			Content: `package {{.PackageName}}

templ {{.ComponentName}}(title string, size string) {
	<div class={ "widget", "widget--" + size }>
		<h2>{ title }</h2>
		<div class="widget__content">
			{ children... }
		</div>
	</div>
}`,
			StylesCSS: `.widget {
	border: 1px solid #ccc;
	border-radius: 4px;
	padding: 16px;
}

.widget--small { padding: 8px; }
.widget--large { padding: 24px; }`,
		}

		generator.AddCustomTemplate("widget", customTemplate)

		// Generate component using custom template
		opts := GenerateOptions{
			Name:       "MyWidget",
			Template:   "widget",
			WithStyles: true,
		}

		err := generator.Generate(opts)
		require.NoError(t, err)

		// Verify component was generated
		componentFile := filepath.Join(tc.TempDir(), "mywidget.templ")
		assert.FileExists(t, componentFile)

		// Verify styles were generated
		stylesFile := filepath.Join(tc.TempDir(), "styles", "mywidget.css")
		assert.FileExists(t, stylesFile)

		// Verify content
		content, err := os.ReadFile(componentFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "MyWidget")
		assert.Contains(t, string(content), "widget--")
	})
}

// BenchmarkComponentGeneration benchmarks component generation performance.
func BenchmarkComponentGeneration(b *testing.B) {
	generator := NewComponentGenerator("/tmp/bench", "components", "benchmark", "Benchmark")

	opts := GenerateOptions{
		Name:     "BenchmarkComponent",
		Template: "button",
	}

	b.ResetTimer()

	for i := range b.N {
		// Use unique names to avoid file conflicts
		opts.Name = fmt.Sprintf("BenchmarkComponent%d", i)
		err := generator.Generate(opts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

// BenchmarkTemplateExecution benchmarks template execution.
func BenchmarkTemplateExecution(b *testing.B) {
	generator := NewComponentGenerator("", "", "", "")
	templateContent := GetBuiltinTemplates()["button"].Content

	ctx := TemplateContext{
		ComponentName: "BenchButton",
		PackageName:   "components",
		Parameters:    []TemplateParameter{{Name: "text", Type: "string"}},
	}

	b.ResetTimer()

	for i := range b.N {
		filename := fmt.Sprintf("/tmp/bench_%d.templ", i)
		err := generator.generateFile(filename, templateContent, ctx)
		if err != nil {
			b.Fatalf("Template execution failed: %v", err)
		}
	}
}

// Helper functions for testing
