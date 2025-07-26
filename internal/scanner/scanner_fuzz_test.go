package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// FuzzTemplateParser tests the template parser with various inputs
func FuzzTemplateParser(f *testing.F) {
	// Seed with known good templates
	f.Add(`package components

templ Button() {
	<button>Click</button>
}`)

	f.Add(`package components

templ Complex(items []Item) {
	for _, item := range items {
		<div>{ item.Name }</div>
	}
}`)

	f.Add(`package components

templ Card(title string, content string) {
	<div class="card">
		<h3>{ title }</h3>
		<p>{ content }</p>
	</div>
}`)

	f.Add(`package components

templ Layout() {
	<!DOCTYPE html>
	<html>
	<head><title>Test</title></head>
	<body>
		{ children... }
	</body>
	</html>
}`)

	// Seed with malformed templates that should not crash the parser
	f.Add(`templ Broken() { <div>unclosed`)
	f.Add(`templ Missing`)
	f.Add(`package components

templ Invalid(param) {
	<div>missing type</div>
}`)

	f.Add(`package components

templ 123Invalid() {
	<div>invalid name</div>
}`)

	f.Add(`package components

templ Valid() {
	<div>{ undefined_variable }</div>
}`)

	f.Fuzz(func(t *testing.T, template string) {
		// Limit input size to prevent resource exhaustion
		if len(template) > 10000 {
			t.Skip("Template too large")
		}

		// Parser should never panic, even on malformed input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked on input: %q\nPanic: %v", template, r)
			}
		}()

		registry := registry.NewComponentRegistry()
		scanner := NewComponentScanner(registry)

		// Parse template by creating temporary file - should not crash
		tempDir := "/tmp"
		tempFile := filepath.Join(tempDir, "fuzz_test.templ")
		os.WriteFile(tempFile, []byte(template), 0644)
		defer os.Remove(tempFile)

		err := scanner.ScanFile(tempFile)
		components := registry.GetAll()

		// Validate results are reasonable
		if err == nil && components != nil {
			for _, component := range components {
				// Component names should not be empty if parsing succeeded
				if component.Name == "" {
					t.Errorf("Parser returned component with empty name for input: %q", template)
				}

				// Package should be reasonable if set
				if component.Package != "" && strings.ContainsAny(component.Package, "/\\:;") {
					t.Errorf(
						"Parser returned invalid package name: %q for input: %q",
						component.Package,
						template,
					)
				}

				// Parameters should have names and types if present
				for _, param := range component.Parameters {
					if param.Name == "" {
						t.Errorf(
							"Parser returned parameter with empty name for input: %q",
							template,
						)
					}
					if param.Type == "" {
						t.Errorf(
							"Parser returned parameter with empty type for input: %q",
							template,
						)
					}
				}
			}
		}
	})
}

// FuzzDirectoryScanning tests directory scanning with various path inputs
func FuzzDirectoryScanning(f *testing.F) {
	// Seed with valid directory patterns
	f.Add("./components")
	f.Add("components")
	f.Add("./")
	f.Add(".")
	f.Add("internal/components")

	// Seed with potentially problematic paths
	f.Add("../../../etc/passwd")
	f.Add("/etc/passwd")
	f.Add("./components/../../../")
	f.Add("components;rm -rf /")
	f.Add("components|cat /etc/passwd")
	f.Add("components$(whoami)")
	f.Add("components`id`")
	f.Add("")
	f.Add("components\x00")
	f.Add("components\n")

	f.Fuzz(func(t *testing.T, dirPath string) {
		// Limit path length to prevent resource exhaustion
		if len(dirPath) > 1000 {
			t.Skip("Path too long")
		}

		// Scanner should never panic, even on malicious paths
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Scanner panicked on path: %q\nPanic: %v", dirPath, r)
			}
		}()

		registry := registry.NewComponentRegistry()
		scanner := NewComponentScanner(registry)

		// Scan directory - should not crash or execute commands
		err := scanner.ScanDirectory(dirPath)

		// If scanning succeeded, verify no suspicious behavior
		if err == nil {
			components := registry.GetAll()

			// Verify all components have reasonable file paths
			for _, component := range components {
				if strings.ContainsAny(component.FilePath, "\x00\n\r") {
					t.Errorf("Component has suspicious file path: %q from scanning: %q",
						component.FilePath, dirPath)
				}
			}
		}
	})
}

// FuzzComponentValidation tests component validation with various inputs
func FuzzComponentValidation(f *testing.F) {
	// Seed with valid component data
	f.Add("Button", "components", "button.templ", "text", "string")
	f.Add("Card", "components", "card.templ", "title", "string")
	f.Add("Layout", "layouts", "layout.templ", "", "")

	// Seed with potentially problematic component data
	f.Add("", "", "", "", "")
	f.Add("Button\x00", "components", "button.templ", "text", "string")
	f.Add("Button", "components\n", "button.templ", "text", "string")
	f.Add("Button", "components", "../../../etc/passwd", "text", "string")
	f.Add("Button", "components", "button.templ", "text\x00", "string")
	f.Add("Button", "components", "button.templ", "text", "string\n")

	f.Fuzz(func(t *testing.T, name, pkg, filePath, paramName, paramType string) {
		// Limit input sizes
		if len(name) > 100 || len(pkg) > 100 || len(filePath) > 500 ||
			len(paramName) > 50 || len(paramType) > 50 {
			t.Skip("Input too large")
		}

		// Component creation should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf(
					"Component creation panicked\nName: %q, Package: %q, FilePath: %q, ParamName: %q, ParamType: %q\nPanic: %v",
					name,
					pkg,
					filePath,
					paramName,
					paramType,
					r,
				)
			}
		}()

		// Create component info
		component := &types.ComponentInfo{
			Name:     name,
			Package:  pkg,
			FilePath: filePath,
		}

		// Add parameter if provided
		if paramName != "" && paramType != "" {
			component.Parameters = []types.ParameterInfo{
				{Name: paramName, Type: paramType},
			}
		}

		// Register component - should not crash
		reg := registry.NewComponentRegistry()
		reg.Register(component)

		// Verify component can be retrieved safely
		retrieved, exists := reg.Get(name)
		if exists {
			// Verify no suspicious data made it through
			if strings.ContainsAny(retrieved.FilePath, "\x00\n\r") {
				t.Errorf("Suspicious characters in file path: %q", retrieved.FilePath)
			}
			if strings.ContainsAny(retrieved.Package, "\x00\n\r") {
				t.Errorf("Suspicious characters in package: %q", retrieved.Package)
			}
		}
	})
}
