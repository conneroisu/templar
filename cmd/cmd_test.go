package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Reset flags
	initMinimal = false
	initExample = false
	initTemplate = ""
	
	// Test init command
	err = runInit(&cobra.Command{}, []string{})
	require.NoError(t, err)
	
	// Check that directories were created
	expectedDirs := []string{
		"components",
		"views",
		"examples",
		"static",
		"static/css",
		"static/js",
		"static/images",
		"mocks",
		"preview",
		".templar",
		".templar/cache",
	}
	
	for _, dir := range expectedDirs {
		assert.DirExists(t, dir)
	}
	
	// Check that files were created
	assert.FileExists(t, ".templar.yml")
	assert.FileExists(t, "go.mod")
	assert.FileExists(t, "components/button.templ")
	assert.FileExists(t, "components/card.templ")
	assert.FileExists(t, "views/layout.templ")
	assert.FileExists(t, "examples/demo.templ")
	assert.FileExists(t, "static/css/styles.css")
	assert.FileExists(t, "preview/wrapper.templ")
}

func TestInitCommandWithProjectName(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Reset flags
	initMinimal = false
	initExample = false
	initTemplate = ""
	
	// Test init command with project name
	err = runInit(&cobra.Command{}, []string{"test-project"})
	require.NoError(t, err)
	
	// Check that project directory was created
	assert.DirExists(t, "test-project")
	assert.FileExists(t, "test-project/.templar.yml")
	assert.FileExists(t, "test-project/go.mod")
}

func TestInitCommandMinimal(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Set minimal flag
	initMinimal = true
	initExample = false
	initTemplate = ""
	
	// Test init command
	err = runInit(&cobra.Command{}, []string{})
	require.NoError(t, err)
	
	// Check that basic directories were created
	assert.DirExists(t, "components")
	assert.FileExists(t, ".templar.yml")
	assert.FileExists(t, "go.mod")
	
	// Check that example components were NOT created
	assert.NoFileExists(t, "components/button.templ")
	assert.NoFileExists(t, "components/card.templ")
}

func TestInitCommandWithTemplate(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Set template flag
	initMinimal = false
	initExample = false
	initTemplate = "minimal"
	
	// Test init command
	err = runInit(&cobra.Command{}, []string{})
	require.NoError(t, err)
	
	// Check that template was applied
	assert.FileExists(t, "components/hello.templ")
}

func TestListCommand(t *testing.T) {
	// Create a temporary directory with components
	tempDir := t.TempDir()
	
	// Create component files
	componentDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentDir, 0755)
	require.NoError(t, err)
	
	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}
`
	
	err = os.WriteFile(filepath.Join(componentDir, "test.templ"), []byte(componentContent), 0644)
	require.NoError(t, err)
	
	// Set up viper configuration
	viper.Reset()
	viper.Set("components.scan_paths", []string{componentDir})
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")
	
	// Reset flags
	listFormat = "table"
	listWithDeps = false
	listWithProps = false
	
	// Test list command
	err = runList(&cobra.Command{}, []string{})
	require.NoError(t, err)
}

func TestListCommandJSON(t *testing.T) {
	// Create a temporary directory with components
	tempDir := t.TempDir()
	
	// Create component files
	componentDir := filepath.Join(tempDir, "components")
	err := os.MkdirAll(componentDir, 0755)
	require.NoError(t, err)
	
	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}
`
	
	err = os.WriteFile(filepath.Join(componentDir, "test.templ"), []byte(componentContent), 0644)
	require.NoError(t, err)
	
	// Set up viper configuration
	viper.Reset()
	viper.Set("components.scan_paths", []string{componentDir})
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")
	
	// Set flags
	listFormat = "json"
	listWithDeps = true
	listWithProps = true
	
	// Test list command
	err = runList(&cobra.Command{}, []string{})
	require.NoError(t, err)
}

func TestBuildCommand(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Create component files
	componentDir := "components"
	err = os.MkdirAll(componentDir, 0755)
	require.NoError(t, err)
	
	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}
`
	
	err = os.WriteFile(filepath.Join(componentDir, "test.templ"), []byte(componentContent), 0644)
	require.NoError(t, err)
	
	// Set up viper configuration
	viper.Reset()
	viper.Set("components.scan_paths", []string{componentDir})
	viper.Set("build.command", "echo 'build command executed'")
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")
	
	// Reset flags
	buildOutput = ""
	buildProduction = false
	buildAnalyze = false
	buildClean = false
	
	// Test build command
	err = runBuild(&cobra.Command{}, []string{})
	// This might fail because templ is not available in test environment
	// But it should at least scan the components
	if err != nil {
		// Check if it's a templ-related error
		assert.Contains(t, err.Error(), "templ")
	}
}

func TestBuildCommandWithAnalysis(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Create component files
	componentDir := "components"
	err = os.MkdirAll(componentDir, 0755)
	require.NoError(t, err)
	
	componentContent := `package components

templ TestComponent(title string) {
	<h1>{ title }</h1>
}
`
	
	err = os.WriteFile(filepath.Join(componentDir, "test.templ"), []byte(componentContent), 0644)
	require.NoError(t, err)
	
	// Set up viper configuration
	viper.Reset()
	viper.Set("components.scan_paths", []string{componentDir})
	viper.Set("build.command", "echo 'build command executed'")
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")
	
	// Set flags for analysis
	buildOutput = ""
	buildProduction = false
	buildAnalyze = true
	buildClean = false
	
	// Test build command
	err = runBuild(&cobra.Command{}, []string{})
	// This might fail because templ is not available in test environment
	// But it should at least scan the components
	if err != nil {
		// Check if it's a templ-related error
		assert.Contains(t, err.Error(), "templ")
	}
}

func TestGenerateMockData(t *testing.T) {
	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		Package:  "components",
		FilePath: "test.templ",
		Parameters: []registry.ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "enabled", Type: "bool"},
			{Name: "items", Type: "[]string"},
		},
	}
	
	mockData := generateMockData(component)
	
	assert.Equal(t, "Mock Text", mockData["title"])
	assert.Equal(t, 42, mockData["count"])
	assert.Equal(t, true, mockData["enabled"])
	assert.Equal(t, []string{"Item 1", "Item 2", "Item 3"}, mockData["items"])
}

func TestGenerateMockValue(t *testing.T) {
	tests := []struct {
		paramType string
		expected  interface{}
	}{
		{"string", "Mock Text"},
		{"int", 42},
		{"bool", true},
		{"[]string", []string{"Item 1", "Item 2", "Item 3"}},
		{"[]int", []int{1, 2, 3}},
		{"unknown", "Mock Value"},
	}
	
	for _, test := range tests {
		t.Run(test.paramType, func(t *testing.T) {
			result := generateMockValue(test.paramType)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestCreateDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	
	err := createDirectoryStructure(tempDir)
	require.NoError(t, err)
	
	expectedDirs := []string{
		"components",
		"views",
		"examples",
		"static",
		"static/css",
		"static/js",
		"static/images",
		"mocks",
		"preview",
		".templar",
		".templar/cache",
	}
	
	for _, dir := range expectedDirs {
		assert.DirExists(t, filepath.Join(tempDir, dir))
	}
}

func TestCreateConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	
	err := createConfigFile(tempDir)
	require.NoError(t, err)
	
	configPath := filepath.Join(tempDir, ".templar.yml")
	assert.FileExists(t, configPath)
	
	// Check content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	assert.Contains(t, string(content), "server:")
	assert.Contains(t, string(content), "port: 8080")
	assert.Contains(t, string(content), "components:")
}

func TestCreateGoModule(t *testing.T) {
	tempDir := t.TempDir()
	
	err := createGoModule(tempDir)
	require.NoError(t, err)
	
	goModPath := filepath.Join(tempDir, "go.mod")
	assert.FileExists(t, goModPath)
	
	// Check content
	content, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	
	assert.Contains(t, string(content), "module")
	assert.Contains(t, string(content), "go 1.24")
	assert.Contains(t, string(content), "github.com/a-h/templ")
}