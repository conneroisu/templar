// Comprehensive test suite to verify all implemented improvements
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/mockdata"
	"github.com/conneroisu/templar/internal/scaffolding"
	"github.com/conneroisu/templar/internal/types"
)

// TestSuite represents our comprehensive test suite
type TestSuite struct {
	results []TestResult
}

// TestResult holds the result of a test
type TestResult struct {
	Name        string
	Status      string // PASS, FAIL, SKIP
	Duration    time.Duration
	Error       error
	Description string
}

// Removed main function to avoid conflict - use as reference for testing

func (ts *TestSuite) addResult(name, description string, err error, duration time.Duration) {
	status := "PASS"
	if err != nil {
		status = "FAIL"
	}

	ts.results = append(ts.results, TestResult{
		Name:        name,
		Status:      status,
		Duration:    duration,
		Error:       err,
		Description: description,
	})
}

func (ts *TestSuite) runTest(name, description string, testFunc func() error) {
	start := time.Now()
	err := testFunc()
	duration := time.Since(start)

	ts.addResult(name, description, err, duration)

	status := "✅ PASS"
	if err != nil {
		status = "❌ FAIL"
	}

	fmt.Printf("%s %s (%v)\n", status, name, duration)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}
}

// Intelligent Mock Data Tests
func (ts *TestSuite) runIntelligentMockDataTests() {
	fmt.Println("\n🎭 Testing Intelligent Mock Data Generation")
	fmt.Println("------------------------------------------")

	ts.runTest("MockData_BasicGeneration", "Test basic mock data generation", func() error {
		generator := mockdata.NewMockGenerator()

		component := &types.ComponentInfo{
			Name: "TestComponent",
			Parameters: []types.ParameterInfo{
				{Name: "email", Type: "string"},
				{Name: "age", Type: "int"},
				{Name: "isActive", Type: "bool"},
			},
		}

		data := generator.GenerateForComponent(component)

		if len(data) != 3 {
			return fmt.Errorf("expected 3 parameters, got %d", len(data))
		}

		// Check email pattern
		if email, ok := data["email"].(string); !ok || !strings.Contains(email, "@") {
			return fmt.Errorf("email parameter not generated correctly: %v", data["email"])
		}

		// Check age is numeric
		if _, ok := data["age"].(int); !ok {
			return fmt.Errorf("age parameter not generated as int: %v", data["age"])
		}

		// Check boolean
		if _, ok := data["isActive"].(bool); !ok {
			return fmt.Errorf("isActive parameter not generated as bool: %v", data["isActive"])
		}

		return nil
	})

	ts.runTest("MockData_AdvancedGeneration", "Test advanced mock data patterns", func() error {
		generator := mockdata.NewAdvancedMockGenerator()

		component := &types.ComponentInfo{
			Name: "AdvancedComponent",
			Parameters: []types.ParameterInfo{
				{Name: "phoneNumber", Type: "string"},
				{Name: "companyName", Type: "string"},
				{Name: "percentage", Type: "string"},
			},
		}

		data := generator.GenerateForComponent(component)

		// Check phone number pattern
		if phone, ok := data["phoneNumber"].(string); !ok || !strings.HasPrefix(phone, "+1-") {
			return fmt.Errorf("phone number not generated correctly: %v", data["phoneNumber"])
		}

		// Check company name
		if company, ok := data["companyName"].(string); !ok || company == "" {
			return fmt.Errorf("company name not generated: %v", data["companyName"])
		}

		// Check percentage
		if percent, ok := data["percentage"].(string); !ok || !strings.HasSuffix(percent, "%") {
			return fmt.Errorf("percentage not generated correctly: %v", data["percentage"])
		}

		return nil
	})
}

// Configuration Validation Tests
func (ts *TestSuite) runConfigurationValidationTests() {
	fmt.Println("\n⚙️  Testing Configuration Validation & Wizard")
	fmt.Println("--------------------------------------------")

	ts.runTest("Config_ValidationBasic", "Test basic configuration validation", func() error {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "localhost",
			},
			Components: config.ComponentsConfig{
				ScanPaths: []string{"./components"},
			},
			Build: config.BuildConfig{
				Command: "templ generate",
			},
		}

		validation := config.ValidateConfigWithDetails(cfg)
		if !validation.Valid {
			return fmt.Errorf("valid configuration failed validation: %v", validation.Errors)
		}

		return nil
	})

	ts.runTest("Config_ValidationInvalid", "Test validation catches invalid config", func() error {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 99999,                // Invalid port
				Host: "localhost;rm -rf /", // Dangerous host
			},
			Components: config.ComponentsConfig{
				ScanPaths: []string{}, // No scan paths
			},
		}

		validation := config.ValidateConfigWithDetails(cfg)
		if validation.Valid {
			return fmt.Errorf("invalid configuration passed validation")
		}

		if len(validation.Errors) == 0 {
			return fmt.Errorf("expected validation errors but got none")
		}

		return nil
	})

	ts.runTest("Config_WizardCreation", "Test configuration wizard creation", func() error {
		// Test that wizard can be created (basic instantiation test)
		wizard := config.NewConfigWizard()
		if wizard == nil {
			return fmt.Errorf("failed to create configuration wizard")
		}
		return nil
	})
}

// Component Scaffolding Tests
func (ts *TestSuite) runComponentScaffoldingTests() {
	fmt.Println("\n🏗️  Testing Component Scaffolding")
	fmt.Println("--------------------------------")

	ts.runTest("Scaffolding_TemplateList", "Test template listing", func() error {
		generator := scaffolding.NewComponentGenerator("./test", "components", "test", "")

		templates := generator.ListTemplates()
		if len(templates) == 0 {
			return fmt.Errorf("no templates found")
		}

		// Check for essential templates
		hasButton := false
		hasCard := false
		hasForm := false

		for _, tmpl := range templates {
			switch tmpl.Name {
			case "button":
				hasButton = true
			case "card":
				hasCard = true
			case "form":
				hasForm = true
			}
		}

		if !hasButton || !hasCard || !hasForm {
			return fmt.Errorf("missing essential templates: button=%v, card=%v, form=%v",
				hasButton, hasCard, hasForm)
		}

		return nil
	})

	ts.runTest("Scaffolding_ComponentGeneration", "Test component generation", func() error {
		tempDir := filepath.Join(os.TempDir(), "templar-test-scaffold")
		defer os.RemoveAll(tempDir)

		generator := scaffolding.NewComponentGenerator(tempDir, "components", "test", "tester")

		opts := scaffolding.GenerateOptions{
			Name:       "TestButton",
			Template:   "button",
			WithTests:  true,
			WithStyles: true,
		}

		if err := generator.Generate(opts); err != nil {
			return fmt.Errorf("failed to generate component: %w", err)
		}

		// Check generated files
		componentFile := filepath.Join(tempDir, "testbutton.templ")
		if _, err := os.Stat(componentFile); os.IsNotExist(err) {
			return fmt.Errorf("component file not generated: %s", componentFile)
		}

		testFile := filepath.Join(tempDir, "testbutton_test.go")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			return fmt.Errorf("test file not generated: %s", testFile)
		}

		stylesFile := filepath.Join(tempDir, "styles", "testbutton.css")
		if _, err := os.Stat(stylesFile); os.IsNotExist(err) {
			return fmt.Errorf("styles file not generated: %s", stylesFile)
		}

		return nil
	})

	ts.runTest("Scaffolding_ProjectScaffold", "Test project scaffold generation", func() error {
		tempDir := filepath.Join(os.TempDir(), "templar-test-project")
		defer os.RemoveAll(tempDir)

		generator := scaffolding.NewComponentGenerator("", "components", "test", "tester")

		if err := generator.CreateProjectScaffold(tempDir); err != nil {
			return fmt.Errorf("failed to create project scaffold: %w", err)
		}

		// Check essential directories
		dirs := []string{
			"components/ui",
			"components/layout",
			"components/forms",
			"views",
			"styles",
		}

		for _, dir := range dirs {
			fullPath := filepath.Join(tempDir, dir)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				return fmt.Errorf("directory not created: %s", dir)
			}
		}

		// Check main styles file
		mainStyles := filepath.Join(tempDir, "styles", "main.css")
		if _, err := os.Stat(mainStyles); os.IsNotExist(err) {
			return fmt.Errorf("main styles file not created")
		}

		return nil
	})
}

// Security Tests
func (ts *TestSuite) runSecurityTests() {
	fmt.Println("\n🔒 Testing Security Enhancements")
	fmt.Println("--------------------------------")

	ts.runTest("Security_ConfigValidation", "Test security validation", func() error {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "example.com; rm -rf /", // Command injection attempt
			},
			Build: config.BuildConfig{
				CacheDir: "../../../etc/passwd", // Path traversal attempt
			},
		}

		validation := config.ValidateConfigWithDetails(cfg)
		if validation.Valid {
			return fmt.Errorf("dangerous configuration passed validation")
		}

		// Should have security-related errors
		hasSecurityError := false
		for _, err := range validation.Errors {
			if strings.Contains(err.Message, "dangerous") ||
				strings.Contains(err.Message, "traversal") {
				hasSecurityError = true
				break
			}
		}

		if !hasSecurityError {
			return fmt.Errorf("expected security validation errors")
		}

		return nil
	})

	ts.runTest("Security_ComponentNameValidation", "Test component name validation", func() error {
		// Test invalid names
		invalidNames := []string{
			"",               // Empty
			"component-name", // Hyphen
			"123Component",   // Starts with number
			"component$",     // Special character
		}

		for _, name := range invalidNames {
			if err := scaffolding.ValidateComponentName(name); err == nil {
				return fmt.Errorf("invalid name '%s' passed validation", name)
			}
		}

		// Test valid names
		validNames := []string{
			"Component",
			"MyComponent",
			"Component123",
			"Component_Name",
		}

		for _, name := range validNames {
			if err := scaffolding.ValidateComponentName(name); err != nil {
				return fmt.Errorf("valid name '%s' failed validation: %v", name, err)
			}
		}

		return nil
	})
}

// Performance Tests
func (ts *TestSuite) runPerformanceTests() {
	fmt.Println("\n⚡ Testing Performance Optimizations")
	fmt.Println("-----------------------------------")

	ts.runTest("Performance_MockDataGeneration", "Test mock data generation performance", func() error {
		generator := mockdata.NewAdvancedMockGenerator()

		// Create large component with many parameters
		params := make([]types.ParameterInfo, 100)
		for i := 0; i < 100; i++ {
			params[i] = types.ParameterInfo{
				Name: fmt.Sprintf("param%d", i),
				Type: "string",
			}
		}

		component := &types.ComponentInfo{
			Name:       "LargeComponent",
			Parameters: params,
		}

		start := time.Now()
		data := generator.GenerateForComponent(component)
		duration := time.Since(start)

		if len(data) != 100 {
			return fmt.Errorf("expected 100 parameters, got %d", len(data))
		}

		// Should complete in reasonable time (< 100ms for 100 params)
		if duration > 100*time.Millisecond {
			return fmt.Errorf("mock data generation too slow: %v", duration)
		}

		return nil
	})

	ts.runTest("Performance_TemplateGeneration", "Test template generation performance", func() error {
		tempDir := filepath.Join(os.TempDir(), "templar-perf-test")
		defer os.RemoveAll(tempDir)

		generator := scaffolding.NewComponentGenerator(tempDir, "components", "test", "")

		start := time.Now()

		// Generate multiple components
		for i := 0; i < 10; i++ {
			opts := scaffolding.GenerateOptions{
				Name:     fmt.Sprintf("Component%d", i),
				Template: "button",
			}

			if err := generator.Generate(opts); err != nil {
				return fmt.Errorf("failed to generate component %d: %w", i, err)
			}
		}

		duration := time.Since(start)

		// Should complete in reasonable time (< 500ms for 10 components)
		if duration > 500*time.Millisecond {
			return fmt.Errorf("component generation too slow: %v", duration)
		}

		return nil
	})
}

// Print test results
func (ts *TestSuite) printResults() {
	fmt.Println("\n📊 Test Results Summary")
	fmt.Println("======================")

	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range ts.results {
		totalDuration += result.Duration
		if result.Status == "PASS" {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("Total Tests: %d\n", len(ts.results))
	fmt.Printf("✅ Passed: %d\n", passed)
	fmt.Printf("❌ Failed: %d\n", failed)
	fmt.Printf("⏱️  Total Duration: %v\n", totalDuration)
	fmt.Printf("📈 Success Rate: %.1f%%\n", float64(passed)/float64(len(ts.results))*100)

	if failed > 0 {
		fmt.Println("\n❌ Failed Tests:")
		for _, result := range ts.results {
			if result.Status == "FAIL" {
				fmt.Printf("   • %s: %v\n", result.Name, result.Error)
			}
		}
	}

	// Overall result
	if failed == 0 {
		fmt.Println("\n🎉 All tests passed! The implementation is working correctly.")
	} else {
		fmt.Printf("\n⚠️  %d tests failed. Please review the implementation.\n", failed)
	}
}
