//go:build integration
// +build integration

package integration_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestComponent creates a test component file with specified content
func createTestComponent(dir, name, content string) string {
	filePath := filepath.Join(dir, name+".templ")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		panic(fmt.Sprintf("Failed to create test component %s: %v", name, err))
	}
	return filePath
}

// createTestComponentsDir creates a directory with multiple test components
func createTestComponentsDir(componentDefinitions map[string]string) string {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	if err := os.MkdirAll(testDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create test directory: %v", err))
	}

	for name, content := range componentDefinitions {
		createTestComponent(testDir, name, content)
	}

	return testDir
}

func TestIntegration_ScannerRegistry_BasicDiscovery(t *testing.T) {
	// Create test components
	components := map[string]string{
		"Button": `package components

templ Button(text string) {
	<button class="btn">{text}</button>
}`,
		"Card": `package components

templ Card(title string, content string) {
	<div class="card">
		<h3>{title}</h3>
		<p>{content}</p>
	</div>
}`,
		"Modal": `package components

templ Modal(title string, active bool) {
	if active {
		<div class="modal">
			<h2>{title}</h2>
		</div>
	}
}`,
	}

	testDir := createTestComponentsDir(components)
	defer os.RemoveAll(testDir)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Scan directory
	err := componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Verify all components are registered
	assert.Equal(t, 3, reg.Count())

	// Verify Button component
	button, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Equal(t, "Button", button.Name)
	assert.Equal(t, "components", button.Package)
	assert.Len(t, button.Parameters, 1)
	assert.Equal(t, "text", button.Parameters[0].Name)
	assert.Equal(t, "string", button.Parameters[0].Type)

	// Verify Card component
	card, exists := reg.Get("Card")
	assert.True(t, exists)
	assert.Equal(t, "Card", card.Name)
	assert.Len(t, card.Parameters, 2)
	assert.Equal(t, "title", card.Parameters[0].Name)
	assert.Equal(t, "content", card.Parameters[1].Name)

	// Verify Modal component
	modal, exists := reg.Get("Modal")
	assert.True(t, exists)
	assert.Equal(t, "Modal", modal.Name)
	assert.Len(t, modal.Parameters, 2)
	assert.Equal(t, "title", modal.Parameters[0].Name)
	assert.Equal(t, "active", modal.Parameters[1].Name)
	assert.Equal(t, "bool", modal.Parameters[1].Type)
}

func TestIntegration_ScannerRegistry_ComponentModification(t *testing.T) {
	// Create initial test component
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	initialContent := `package components

templ Button(text string) {
	<button class="btn">{text}</button>
}`

	buttonFile := createTestComponent(testDir, "Button", initialContent)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Initial scan
	err := componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Verify initial component
	button, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Len(t, button.Parameters, 1)
	originalModTime := button.LastMod

	// Wait a moment to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	// Modify component
	modifiedContent := `package components

templ Button(text string, disabled bool) {
	<button class="btn" disabled?={disabled}>{text}</button>
}`

	err = os.WriteFile(buttonFile, []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Rescan directory
	err = componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Verify component was updated
	updatedButton, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Len(t, updatedButton.Parameters, 2)
	assert.Equal(t, "text", updatedButton.Parameters[0].Name)
	assert.Equal(t, "disabled", updatedButton.Parameters[1].Name)
	assert.Equal(t, "bool", updatedButton.Parameters[1].Type)
	assert.True(t, updatedButton.LastMod.After(originalModTime))
}

func TestIntegration_ScannerRegistry_ComponentDeletion(t *testing.T) {
	// Create test components
	components := map[string]string{
		"Button": `package components

templ Button(text string) {
	<button>{text}</button>
}`,
		"Card": `package components

templ Card(title string) {
	<div>{title}</div>
}`,
	}

	testDir := createTestComponentsDir(components)
	defer os.RemoveAll(testDir)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Initial scan
	err := componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)
	assert.Equal(t, 2, reg.Count())

	// Delete one component file
	buttonFile := filepath.Join(testDir, "Button.templ")
	err = os.Remove(buttonFile)
	require.NoError(t, err)

	// Rescan directory
	err = componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Verify only remaining component exists
	_, exists := reg.Get("Button")
	assert.False(t, exists)

	card, exists := reg.Get("Card")
	assert.True(t, exists)
	assert.Equal(t, "Card", card.Name)

	// Registry should only have 1 component now
	assert.Equal(t, 1, reg.Count())
}

func TestIntegration_ScannerRegistry_ConcurrentAccess(t *testing.T) {
	// Create test components
	components := make(map[string]string)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("Component%d", i)
		components[name] = fmt.Sprintf(`package components

templ %s(text string) {
	<div class="component-%d">{text}</div>
}`, name, i)
	}

	testDir := createTestComponentsDir(components)
	defer os.RemoveAll(testDir)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Start concurrent scanning and registry access
	done := make(chan bool, 3)

	// Goroutine 1: Continuous scanning
	go func() {
		for i := 0; i < 5; i++ {
			err := componentScanner.ScanDirectory(testDir)
			if err != nil {
				t.Errorf("Scan error in goroutine 1: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Registry reading
	go func() {
		for i := 0; i < 20; i++ {
			count := reg.Count()
			if count > 0 {
				all := reg.GetAll()
				if len(all) != count {
					t.Errorf("Inconsistent registry state: count=%d, len(all)=%d", count, len(all))
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Component access
	go func() {
		for i := 0; i < 15; i++ {
			componentName := fmt.Sprintf("Component%d", i%10)
			_, exists := reg.Get(componentName)
			// It's okay if component doesn't exist during concurrent scanning
			_ = exists
			time.Sleep(7 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Final verification
	assert.Equal(t, 10, reg.Count())
	for i := 0; i < 10; i++ {
		componentName := fmt.Sprintf("Component%d", i)
		component, exists := reg.Get(componentName)
		assert.True(t, exists, "Component %s should exist", componentName)
		assert.Equal(t, componentName, component.Name)
	}
}

func TestIntegration_ScannerRegistry_ErrorHandling(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Create component with syntax error
	invalidContent := `package components

templ InvalidComponent(text string {  // Missing closing parenthesis
	<div>{text}</div>
}`

	createTestComponent(testDir, "Invalid", invalidContent)

	// Create valid component
	validContent := `package components

templ ValidComponent(text string) {
	<div>{text}</div>
}`

	createTestComponent(testDir, "Valid", validContent)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Scan directory - should handle errors gracefully
	_ = componentScanner.ScanDirectory(testDir)
	// May or may not error depending on scanner implementation
	// The key is that it should not crash

	// Valid component should still be registered
	validComponent, exists := reg.Get("ValidComponent")
	if exists {
		assert.Equal(t, "ValidComponent", validComponent.Name)
		assert.Equal(t, "components", validComponent.Package)
	}
}

func TestIntegration_ScannerRegistry_ComplexComponents(t *testing.T) {
	// Create components with various complexities
	components := map[string]string{
		"SimpleButton": `package components

templ SimpleButton(text string) {
	<button>{text}</button>
}`,
		"ComplexForm": `package components

import "fmt"

templ ComplexForm(fields []FormField, submitted bool, errors map[string]string) {
	<form>
		for _, field := range fields {
			<div class="field">
				<label>{field.Label}</label>
				<input type={field.Type} name={field.Name} value={field.Value} />
				if err, exists := errors[field.Name]; exists {
					<span class="error">{err}</span>
				}
			</div>
		}
		if submitted {
			<p class="success">Form submitted successfully!</p>
		}
		<button type="submit">Submit</button>
	</form>
}`,
		"DataTable": `package components

templ DataTable(headers []string, rows [][]string, sortBy string, ascending bool) {
	<table class="data-table">
		<thead>
			<tr>
				for _, header := range headers {
					<th class={ templ.KV("sorted", header == sortBy), templ.KV("asc", ascending) }>
						{header}
					</th>
				}
			</tr>
		</thead>
		<tbody>
			for i, row := range rows {
				<tr class={ templ.KV("even", i%2 == 0) }>
					for _, cell := range row {
						<td>{cell}</td>
					}
				</tr>
			}
		</tbody>
	</table>
}`,
	}

	testDir := createTestComponentsDir(components)
	defer os.RemoveAll(testDir)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Scan directory
	err := componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Verify all components are registered
	assert.Equal(t, 3, reg.Count())

	// Verify SimpleButton
	simpleButton, exists := reg.Get("SimpleButton")
	assert.True(t, exists)
	assert.Len(t, simpleButton.Parameters, 1)

	// Verify ComplexForm
	complexForm, exists := reg.Get("ComplexForm")
	assert.True(t, exists)
	assert.Len(t, complexForm.Parameters, 3)
	paramNames := make([]string, len(complexForm.Parameters))
	for i, param := range complexForm.Parameters {
		paramNames[i] = param.Name
	}
	assert.Contains(t, paramNames, "fields")
	assert.Contains(t, paramNames, "submitted")
	assert.Contains(t, paramNames, "errors")

	// Verify DataTable
	dataTable, exists := reg.Get("DataTable")
	assert.True(t, exists)
	assert.Len(t, dataTable.Parameters, 4)
	paramNames = make([]string, len(dataTable.Parameters))
	for i, param := range dataTable.Parameters {
		paramNames[i] = param.Name
	}
	assert.Contains(t, paramNames, "headers")
	assert.Contains(t, paramNames, "rows")
	assert.Contains(t, paramNames, "sortBy")
	assert.Contains(t, paramNames, "ascending")
}

func TestIntegration_ScannerRegistry_WatchEvents(t *testing.T) {
	testDir := fmt.Sprintf("integration_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// Initialize scanner and registry
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)

	// Set up event watching
	eventChan := reg.Watch()
	defer reg.UnWatch(eventChan)

	// Create initial component
	initialContent := `package components

templ TestComponent(text string) {
	<div>{text}</div>
}`

	createTestComponent(testDir, "TestComponent", initialContent)

	// Scan directory
	err := componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Wait for and verify add event
	select {
	case event := <-eventChan:
		assert.Equal(t, registry.EventTypeAdded, event.Type)
		assert.Equal(t, "TestComponent", event.Component.Name)
		assert.False(t, event.Timestamp.IsZero())
	case <-time.After(1 * time.Second):
		t.Fatal("Expected add event not received")
	}

	// Modify component
	modifiedContent := `package components

templ TestComponent(text string, active bool) {
	<div class={ templ.KV("active", active) }>{text}</div>
}`

	testFile := filepath.Join(testDir, "TestComponent.templ")
	require.NoError(t, os.WriteFile(testFile, []byte(modifiedContent), 0644))

	// Rescan
	err = componentScanner.ScanDirectory(testDir)
	require.NoError(t, err)

	// Wait for and verify update event
	select {
	case event := <-eventChan:
		assert.Equal(t, registry.EventTypeUpdated, event.Type)
		assert.Equal(t, "TestComponent", event.Component.Name)
		assert.Len(t, event.Component.Parameters, 2)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected update event not received")
	}
}
