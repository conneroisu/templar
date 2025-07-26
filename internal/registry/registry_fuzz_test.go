package registry

import (
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// FuzzComponentRegistration tests component registration with various inputs
func FuzzComponentRegistration(f *testing.F) {
	// Seed with various component registration scenarios
	f.Add(
		"Button\x00./components/button.templ\x00package components\n\ntempl Button(text string) {\n\t<button>{text}</button>\n}",
	)
	f.Add("../../../etc/passwd\x00./malicious.templ\x00malicious content")
	f.Add("<script>alert('xss')</script>\x00./xss.templ\x00XSS content")
	f.Add("\x00\x00\x00")
	f.Add("Unicode🎯\x00./unicode💻.templ\x00Unicode content")
	f.Add("Component\x00\x00")
	f.Add("VeryLongComponentName" + strings.Repeat("A", 1000) + "\x00./long.templ\x00content")

	f.Fuzz(func(t *testing.T, regData string) {
		if len(regData) > 50000 {
			t.Skip("Registration data too large")
		}

		parts := strings.Split(regData, "\x00")
		if len(parts) != 3 {
			t.Skip("Invalid registration data format")
		}

		name, path := parts[0], parts[1]
		_ = parts[2] // content not used anymore

		registry := NewComponentRegistry()

		compInfo := &types.ComponentInfo{
			Name:         name,
			FilePath:     path,
			Package:      "components",
			Parameters:   []types.ParameterInfo{},
			Dependencies: []string{},
			LastMod:      time.Now(),
			Hash:         "test-hash",
			Imports:      []string{},
		}

		// Test that registration doesn't panic
		registry.Register(compInfo)

		// Verify registered component is safe
		components := registry.GetAll()
		for _, comp := range components {
			// Check for control characters in component name
			if strings.ContainsAny(
				comp.Name,
				"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
			) {
				t.Errorf("Registered component name contains control characters: %q", comp.Name)
			}

			// Check for path traversal in component name or path
			if (strings.Contains(comp.Name, "..") || strings.Contains(comp.FilePath, "..")) &&
				(strings.Contains(comp.Name, "etc") || strings.Contains(comp.FilePath, "etc") ||
					strings.Contains(comp.Name, "system32") || strings.Contains(comp.FilePath, "system32")) {
				t.Errorf(
					"Registered component contains dangerous path traversal: name=%q path=%q",
					comp.Name,
					comp.FilePath,
				)
			}

			// Component content validation removed since Content field doesn't exist
			// XSS protection is handled at the rendering level
		}
	})
}

// FuzzComponentSearch tests component search with various queries
func FuzzComponentSearch(f *testing.F) {
	// Seed with various search patterns
	f.Add("Button")
	f.Add("Card")
	f.Add("../../../etc/passwd")
	f.Add("<script>alert('xss')</script>")
	f.Add("*")
	f.Add("")
	f.Add("Component\x00WithNull")
	f.Add(strings.Repeat("A", 10000))

	f.Fuzz(func(t *testing.T, searchQuery string) {
		if len(searchQuery) > 10000 {
			t.Skip("Search query too large")
		}

		registry := NewComponentRegistry()

		// Register some test components first
		testComponent := &types.ComponentInfo{
			Name:       "TestButton",
			FilePath:   "./test.templ",
			Package:    "components",
			Parameters: []types.ParameterInfo{},
			Imports:    []string{},
			LastMod:    time.Now(),
			Hash:       "test-hash",
		}
		registry.Register(testComponent)

		// Test that Get doesn't panic with malicious queries
		component, exists := registry.Get(searchQuery)
		_ = exists

		// Verify Get result is safe
		if component != nil {
			if strings.ContainsAny(
				component.Name,
				"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
			) {
				t.Errorf("Get result contains control characters: %q", component.Name)
			}
		}

		// Test GetAll to ensure all components are safe
		allComponents := registry.GetAll()
		for _, comp := range allComponents {
			if strings.ContainsAny(
				comp.Name,
				"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
			) {
				t.Errorf("GetAll result contains control characters: %q", comp.Name)
			}
		}
	})
}

// FuzzComponentParameters tests component parameter parsing
func FuzzComponentParameters(f *testing.F) {
	// Seed with various parameter patterns
	f.Add("text\x00string")
	f.Add("count\x00int")
	f.Add("<script>\x00func()")
	f.Add("../../../etc/passwd\x00*")
	f.Add("param\x00\x00")
	f.Add(strings.Repeat("A", 1000) + "\x00" + strings.Repeat("B", 1000))

	f.Fuzz(func(t *testing.T, paramData string) {
		if len(paramData) > 10000 {
			t.Skip("Parameter data too large")
		}

		parts := strings.Split(paramData, "\x00")
		if len(parts) != 2 {
			t.Skip("Invalid parameter data format")
		}

		paramName, paramType := parts[0], parts[1]

		registry := NewComponentRegistry()

		param := types.ParameterInfo{
			Name: paramName,
			Type: paramType,
		}

		compInfo := &types.ComponentInfo{
			Name:       "TestComponent",
			FilePath:   "./test.templ",
			Package:    "components",
			Parameters: []types.ParameterInfo{param},
			Imports:    []string{},
			LastMod:    time.Now(),
			Hash:       "test-hash",
		}

		// Test that registration with parameters doesn't panic
		registry.Register(compInfo)

		// Verify registered parameters are safe
		component, exists := registry.Get("TestComponent")
		_ = exists
		if component != nil {
			for _, p := range component.Parameters {
				// Check for control characters
				if strings.ContainsAny(
					p.Name,
					"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
				) {
					t.Errorf("Parameter name contains control characters: %q", p.Name)
				}
				if strings.ContainsAny(
					p.Type,
					"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
				) {
					t.Errorf("Parameter type contains control characters: %q", p.Type)
				}

				// Check for dangerous patterns in parameter types
				if strings.Contains(p.Type, "func(") && strings.Contains(p.Type, "exec") {
					t.Errorf("Parameter type contains dangerous function signature: %q", p.Type)
				}
			}
		}
	})
}

// FuzzComponentDependencies tests component dependency handling
func FuzzComponentDependencies(f *testing.F) {
	// Seed with various dependency patterns
	f.Add("Button\x00Card\x00Form")
	f.Add("../../../etc/passwd\x00./malicious")
	f.Add("<script>\x00alert('xss')")
	f.Add("\x00\x00\x00")
	f.Add(strings.Repeat("A", 1000))

	f.Fuzz(func(t *testing.T, depData string) {
		if len(depData) > 50000 {
			t.Skip("Dependency data too large")
		}

		dependencies := strings.Split(depData, "\x00")

		registry := NewComponentRegistry()

		compInfo := &types.ComponentInfo{
			Name:         "TestComponent",
			FilePath:     "./test.templ",
			Package:      "components",
			Dependencies: dependencies,
			Parameters:   []types.ParameterInfo{},
			Imports:      []string{},
			LastMod:      time.Now(),
			Hash:         "test-hash",
		}

		// Test that registration with dependencies doesn't panic
		registry.Register(compInfo)

		// Verify registered dependencies are safe
		component, exists := registry.Get("TestComponent")
		_ = exists
		if component != nil {
			for _, dep := range component.Dependencies {
				// Check for control characters
				if strings.ContainsAny(
					dep,
					"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
				) {
					t.Errorf("Dependency contains control characters: %q", dep)
				}

				// Check for path traversal
				if strings.Contains(dep, "..") &&
					(strings.Contains(dep, "etc") || strings.Contains(dep, "system32")) {
					t.Errorf("Dependency contains dangerous path traversal: %q", dep)
				}
			}
		}
	})
}
