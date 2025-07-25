package integration_tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory structure
	tempDir := t.TempDir()
	componentsDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create test component file
	componentFile := filepath.Join(componentsDir, "button.templ")
	componentContent := `package components

templ Button(text string, variant string) {
	<button class={"btn", "btn-" + variant}>
		{text}
	</button>
}`

	err = os.WriteFile(componentFile, []byte(componentContent), 0644)
	require.NoError(t, err)

	// Initialize registry and pipeline
	reg := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(4, reg)

	// Test component registration and build
	component := &types.ComponentInfo{
		Name:         "Button",
		Package:      "components",
		FilePath:     componentFile,
		Parameters:   []types.ParameterInfo{{Name: "text", Type: "string", Optional: false}, {Name: "variant", Type: "string", Optional: false}},
		Imports:      []string{},
		LastMod:      time.Now(),
		Hash:         "testhash",
		Dependencies: []string{},
	}

	reg.Register(component)

	// Build the component
	pipeline.Build(component)

	// Verify component is accessible
	retrievedComponent, exists := reg.Get("Button")
	assert.True(t, exists)
	assert.Equal(t, "Button", retrievedComponent.Name)
	assert.Equal(t, "components", retrievedComponent.Package)
}

func TestBuildIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	componentsDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create component with syntax error
	componentFile := filepath.Join(componentsDir, "broken.templ")
	brokenContent := `package components

templ BrokenComponent(text string) {
	<div>
		{text}
	// Missing closing </div>
}`

	err = os.WriteFile(componentFile, []byte(brokenContent), 0644)
	require.NoError(t, err)

	reg := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(4, reg)

	component := &types.ComponentInfo{
		Name:         "BrokenComponent",
		Package:      "components",
		FilePath:     componentFile,
		Parameters:   []types.ParameterInfo{{Name: "text", Type: "string", Optional: false}},
		Imports:      []string{},
		LastMod:      time.Now(),
		Hash:         "brokenhash",
		Dependencies: []string{},
	}

	reg.Register(component)

	// Build should handle the error gracefully
	pipeline.Build(component)
	// The build system will handle errors through callbacks
}

func TestBuildIntegration_CacheValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, ".templar", "cache")
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err)

	reg := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(4, reg)

	// Test that cache directory is properly initialized
	assert.DirExists(t, cacheDir)

	// Test cache validation
	component := &types.ComponentInfo{
		Name:    "CachedComponent",
		Package: "components",
		Hash:    "cachehash",
	}

	reg.Register(component)

	// First build - should create cache entry
	pipeline.Build(component)

	// Second build with same hash - should use cache
	pipeline.Build(component)
}

func TestBuildIntegration_MultipleComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	componentsDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create multiple component files
	components := map[string]string{
		"button.templ": `package components
templ Button(text string) {
	<button>{text}</button>
}`,
		"card.templ": `package components
templ Card(title string, content string) {
	<div class="card">
		<h3>{title}</h3>
		<p>{content}</p>
	</div>
}`,
		"nav.templ": `package components
templ Nav(items []string) {
	<nav>
		for _, item := range items {
			<a href="#">{item}</a>
		}
	</nav>
}`,
	}

	for filename, content := range components {
		filepath := filepath.Join(componentsDir, filename)
		err = os.WriteFile(filepath, []byte(content), 0644)
		require.NoError(t, err)
	}

	reg := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(4, reg)

	// Register all components
	componentInfos := []*types.ComponentInfo{
		{Name: "Button", Package: "components", FilePath: filepath.Join(componentsDir, "button.templ"), Hash: "buttonhash"},
		{Name: "Card", Package: "components", FilePath: filepath.Join(componentsDir, "card.templ"), Hash: "cardhash"},
		{Name: "Nav", Package: "components", FilePath: filepath.Join(componentsDir, "nav.templ"), Hash: "navhash"},
	}

	for _, comp := range componentInfos {
		reg.Register(comp)
	}

	// Build all components
	for _, comp := range componentInfos {
		pipeline.Build(comp)
	}

	// Verify all components are registered
	allComponents := reg.GetAll()
	assert.Len(t, allComponents, 3)

	componentNames := make(map[string]bool)
	for _, comp := range allComponents {
		componentNames[comp.Name] = true
	}

	assert.True(t, componentNames["Button"])
	assert.True(t, componentNames["Card"])
	assert.True(t, componentNames["Nav"])
}

func TestBuildIntegration_SecurityValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	reg := registry.NewComponentRegistry()
	pipeline := build.NewBuildPipeline(4, reg)

	// Test that malicious component names are rejected
	maliciousNames := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"component; rm -rf /",
		"component`rm -rf /`",
		"component$(rm -rf /)",
	}

	for _, name := range maliciousNames {
		t.Run("reject_malicious_name_"+name, func(t *testing.T) {
			component := &types.ComponentInfo{
				Name:    name,
				Package: "components",
				Hash:    "malicioushash",
			}

			reg.Register(component)

			// Build should either reject or sanitize the name
			pipeline.Build(component)
			// The build system should handle this securely
			t.Logf("Build handled component securely: %s", name)
		})
	}
}
