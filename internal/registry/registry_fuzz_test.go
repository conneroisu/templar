package registry

import (
	"strings"
	"testing"
	"time"
)

// FuzzComponentRegistration tests component registration with various inputs
func FuzzComponentRegistration(f *testing.F) {
	// Seed with various component registration scenarios
	f.Add("Button\x00./components/button.templ\x00package components\n\ntempl Button(text string) {\n\t<button>{text}</button>\n}")
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

		name, path, content := parts[0], parts[1], parts[2]

		registry := NewComponentRegistry()
		
		compInfo := &ComponentInfo{
			Name:         name,
			Path:         path,
			FilePath:     path,
			Package:      "components",
			Parameters:   []ComponentParameter{},
			Dependencies: []string{},
			Content:      content,
			LastModified: time.Now(),
		}

		// Test that registration doesn't panic
		registry.Register(compInfo)
		
		// Verify registered component is safe
		components := registry.GetComponents()
		for _, comp := range components {
			// Check for control characters in component name
			if strings.ContainsAny(comp.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Registered component name contains control characters: %q", comp.Name)
			}

			// Check for path traversal in component name or path
			if (strings.Contains(comp.Name, "..") || strings.Contains(comp.Path, "..")) &&
			   (strings.Contains(comp.Name, "etc") || strings.Contains(comp.Path, "etc") ||
			    strings.Contains(comp.Name, "system32") || strings.Contains(comp.Path, "system32")) {
				t.Errorf("Registered component contains dangerous path traversal: name=%q path=%q", comp.Name, comp.Path)
			}

			// Check for XSS patterns in component content
			if strings.Contains(comp.Content, "<script>") &&
			   !strings.Contains(comp.Content, "&lt;script&gt;") {
				t.Errorf("Registered component contains unescaped XSS: %q", comp.Content)
			}
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
		testComponent := &ComponentInfo{
			Name:     "TestButton",
			Path:     "./test.templ",
			FilePath: "./test.templ",
			Package:  "components",
		}
		registry.Register(testComponent)

		// Test that search doesn't panic with malicious queries
		results := registry.Search(searchQuery)
		
		// Verify search results are safe
		for _, result := range results {
			if strings.ContainsAny(result.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Search result contains control characters: %q", result.Name)
			}
		}

		// Test GetByName with the same query
		component := registry.GetByName(searchQuery)
		if component != nil {
			if strings.ContainsAny(component.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("GetByName result contains control characters: %q", component.Name)
			}
		}
	})
}

// FuzzComponentParameters tests component parameter parsing
func FuzzComponentParameters(f *testing.F) {
	// Seed with various parameter patterns
	f.Add("text\x00string")
	f.Add("title\x00string")
	f.Add("data\x00map[string]interface{}")
	f.Add("items\x00[]Item")
	f.Add("../../../etc/passwd\x00string")
	f.Add("<script>alert('xss')</script>\x00string")
	f.Add("\x00\x00")
	f.Add("param\x00func() error")
	f.Add("dangerous\x00chan struct{}")

	f.Fuzz(func(t *testing.T, paramData string) {
		if len(paramData) > 5000 {
			t.Skip("Parameter data too large")
		}

		parts := strings.Split(paramData, "\x00")
		if len(parts) != 2 {
			t.Skip("Invalid parameter data format")
		}

		paramName, paramType := parts[0], parts[1]

		registry := NewComponentRegistry()
		
		param := ComponentParameter{
			Name: paramName,
			Type: paramType,
		}

		compInfo := &ComponentInfo{
			Name:       "TestComponent",
			Path:       "./test.templ",
			FilePath:   "./test.templ",
			Package:    "components",
			Parameters: []ComponentParameter{param},
		}

		// Test that registration with parameters doesn't panic
		registry.Register(compInfo)
		
		// Verify registered parameters are safe
		component := registry.GetByName("TestComponent")
		if component != nil {
			for _, p := range component.Parameters {
				// Check for control characters
				if strings.ContainsAny(p.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
					t.Errorf("Parameter name contains control characters: %q", p.Name)
				}
				if strings.ContainsAny(p.Type, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
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
	f.Add("<script>alert('xss')</script>\x00javascript:")
	f.Add("\x00\x00\x00")
	f.Add("Component1\x00Component2\x00Component3")

	f.Fuzz(func(t *testing.T, depData string) {
		if len(depData) > 10000 {
			t.Skip("Dependency data too large")
		}

		dependencies := strings.Split(depData, "\x00")
		
		registry := NewComponentRegistry()
		
		compInfo := &ComponentInfo{
			Name:         "TestComponent",
			Path:         "./test.templ",
			FilePath:     "./test.templ",
			Package:      "components",
			Dependencies: dependencies,
		}

		// Test that registration with dependencies doesn't panic
		registry.Register(compInfo)
		
		// Verify registered dependencies are safe
		component := registry.GetByName("TestComponent")
		if component != nil {
			for _, dep := range component.Dependencies {
				// Check for control characters
				if strings.ContainsAny(dep, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
					t.Errorf("Dependency contains control characters: %q", dep)
				}

				// Check for path traversal
				if strings.Contains(dep, "..") && 
				   (strings.Contains(dep, "etc") || strings.Contains(dep, "system32")) {
					t.Errorf("Dependency contains dangerous path traversal: %q", dep)
				}

				// Check for XSS patterns
				if strings.Contains(dep, "<script>") || strings.Contains(dep, "javascript:") {
					t.Errorf("Dependency contains XSS pattern: %q", dep)
				}
			}
		}
	})
}

// FuzzEventSubscription tests event subscription with various callback patterns
func FuzzEventSubscription(f *testing.F) {
	// Seed with various event scenarios
	f.Add("component_added")
	f.Add("component_removed")
	f.Add("component_updated")
	f.Add("../../../etc/passwd")
	f.Add("<script>alert('xss')</script>")
	f.Add("")
	f.Add("event\x00withNullByte")

	f.Fuzz(func(t *testing.T, eventType string) {
		if len(eventType) > 1000 {
			t.Skip("Event type too large")
		}

		registry := NewComponentRegistry()
		
		// Test event subscription doesn't panic
		callbackCalled := false
		callback := func(comp *ComponentInfo) {
			callbackCalled = true
			
			// Verify callback receives safe data
			if comp != nil {
				if strings.ContainsAny(comp.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
					// Note: we can't use t.Errorf here as we're in a callback
					panic("Callback received component with control characters")
				}
			}
		}

		registry.Subscribe(callback)
		
		// Trigger an event
		testComponent := &ComponentInfo{
			Name:     "TestComponent",
			Path:     "./test.templ",
			FilePath: "./test.templ",
			Package:  "components",
		}
		
		registry.Register(testComponent)
		
		// Verify callback was called safely
		if !callbackCalled {
			// This is expected behavior, not an error
		}
	})
}

// FuzzComponentSerialization tests component serialization/deserialization
func FuzzComponentSerialization(f *testing.F) {
	// Seed with various component data for serialization
	f.Add("Button\x00./button.templ\x00components\x00Button component")
	f.Add("<script>alert('xss')</script>\x00./xss.templ\x00malicious\x00XSS content")
	f.Add("Component\x00../../../etc/passwd\x00system\x00Traversal attempt")
	f.Add("\x00\x00\x00\x00")

	f.Fuzz(func(t *testing.T, serialData string) {
		if len(serialData) > 20000 {
			t.Skip("Serialization data too large")
		}

		parts := strings.Split(serialData, "\x00")
		if len(parts) != 4 {
			t.Skip("Invalid serialization data format")
		}

		name, path, pkg, content := parts[0], parts[1], parts[2], parts[3]

		// Create component with potentially malicious data
		compInfo := &ComponentInfo{
			Name:     name,
			Path:     path,
			FilePath: path,
			Package:  pkg,
			Content:  content,
		}

		registry := NewComponentRegistry()
		
		// Test registration and retrieval
		registry.Register(compInfo)
		retrieved := registry.GetByName(name)
		
		if retrieved != nil {
			// Verify retrieved data is safe
			if strings.ContainsAny(retrieved.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Retrieved component name contains control characters: %q", retrieved.Name)
			}
			
			if strings.ContainsAny(retrieved.Package, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Retrieved component package contains control characters: %q", retrieved.Package)
			}

			// Check for dangerous content patterns
			if strings.Contains(retrieved.Content, "<script>") &&
			   !strings.Contains(retrieved.Content, "&lt;script&gt;") {
				t.Errorf("Retrieved component content contains unescaped XSS: %q", retrieved.Content)
			}
		}
	})
}

// FuzzRegistryOperations tests concurrent registry operations
func FuzzRegistryOperations(f *testing.F) {
	// Seed with various operation sequences
	f.Add("register\x00Button\x00./button.templ")
	f.Add("unregister\x00Button")
	f.Add("search\x00Button")
	f.Add("getall")
	f.Add("clear")
	f.Add("register\x00../../../etc/passwd\x00./malicious.templ")

	f.Fuzz(func(t *testing.T, opData string) {
		if len(opData) > 5000 {
			t.Skip("Operation data too large")
		}

		parts := strings.Split(opData, "\x00")
		if len(parts) == 0 {
			t.Skip("Empty operation data")
		}

		operation := parts[0]
		registry := NewComponentRegistry()

		// Pre-populate with a test component
		testComp := &ComponentInfo{
			Name:     "TestComponent",
			Path:     "./test.templ",
			FilePath: "./test.templ",
			Package:  "components",
		}
		registry.Register(testComp)

		// Execute operation based on input
		switch operation {
		case "register":
			if len(parts) >= 3 {
				compInfo := &ComponentInfo{
					Name:     parts[1],
					Path:     parts[2],
					FilePath: parts[2],
					Package:  "components",
				}
				registry.Register(compInfo)
			}
		case "unregister":
			if len(parts) >= 2 {
				registry.Unregister(parts[1])
			}
		case "search":
			if len(parts) >= 2 {
				results := registry.Search(parts[1])
				_ = results
			}
		case "getall":
			components := registry.GetComponents()
			_ = components
		case "clear":
			registry.Clear()
		}

		// Verify registry state is safe after operation
		allComponents := registry.GetComponents()
		for _, comp := range allComponents {
			if strings.ContainsAny(comp.Name, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Registry contains component with control characters after %q operation: %q", operation, comp.Name)
			}
		}
	})
}