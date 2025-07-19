package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTempProject(t *testing.T) {
	projectDir := CreateTempProject(t)

	// Verify standard directories exist
	expectedDirs := []string{
		"components",
		"examples",
		"static",
		".templar/cache",
		".templar/render",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(projectDir, dir)
		info, err := os.Stat(fullPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir(), "Expected %s to be a directory", dir)
	}
}

func TestCreateTestComponent(t *testing.T) {
	projectDir := CreateTempProject(t)
	componentsDir := filepath.Join(projectDir, "components")

	content := `package components
templ TestComponent(text string) {
	<div>{text}</div>
}`

	componentPath := CreateTestComponent(t, componentsDir, "TestComponent", content)

	// Verify file exists and has correct content
	assert.FileExists(t, componentPath)

	fileContent, err := os.ReadFile(componentPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(fileContent))
}

func TestCreateTestConfig(t *testing.T) {
	projectDir := CreateTempProject(t)
	cfg := CreateTestConfig(projectDir)

	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.False(t, cfg.Server.Open)

	assert.Contains(t, cfg.Components.ScanPaths, filepath.Join(projectDir, "components"))
	assert.Contains(t, cfg.Components.ScanPaths, filepath.Join(projectDir, "examples"))
	assert.Contains(t, cfg.Components.ExcludePatterns, "*_test.templ")

	assert.Equal(t, "echo 'test build'", cfg.Build.Command)
	assert.Contains(t, cfg.Build.Watch, "**/*.templ")
	assert.Equal(t, filepath.Join(projectDir, ".templar", "cache"), cfg.Build.CacheDir)

	assert.True(t, cfg.Development.HotReload)
	assert.True(t, cfg.Development.CSSInjection)
	assert.True(t, cfg.Development.ErrorOverlay)
}

func TestCreateTestRegistry(t *testing.T) {
	registry := CreateTestRegistry()

	components := registry.GetAll()
	assert.Len(t, components, 3)

	// Verify each test component exists
	componentNames := []string{"Button", "Card", "Nav"}
	for _, name := range componentNames {
		component, exists := registry.Get(name)
		assert.True(t, exists, "Component %s should exist", name)
		assert.Equal(t, name, component.Name)
		assert.Equal(t, "components", component.Package)
		assert.NotEmpty(t, component.Parameters)
		assert.NotEmpty(t, component.Hash)
	}
}

func TestStandardTemplContent(t *testing.T) {
	expectedComponents := []string{"Button", "Card", "Nav", "Layout"}

	for _, name := range expectedComponents {
		content, exists := StandardTemplContent[name]
		assert.True(t, exists, "Standard content for %s should exist", name)
		assert.NotEmpty(t, content, "Content for %s should not be empty", name)
		assert.Contains(t, content, "package components", "Content should contain package declaration")
		assert.Contains(t, content, "templ "+name, "Content should contain templ declaration")
	}
}

func TestSecurityTestCases(t *testing.T) {
	// Test that security test cases are comprehensive
	assert.NotEmpty(t, SecurityTestCases.PathTraversal)
	assert.NotEmpty(t, SecurityTestCases.CommandInjection)
	assert.NotEmpty(t, SecurityTestCases.ScriptInjection)
	assert.NotEmpty(t, SecurityTestCases.SQLInjection)

	// Verify we have comprehensive coverage
	assert.GreaterOrEqual(t, len(SecurityTestCases.PathTraversal), 6)
	assert.GreaterOrEqual(t, len(SecurityTestCases.CommandInjection), 6)
	assert.GreaterOrEqual(t, len(SecurityTestCases.ScriptInjection), 6)
	assert.GreaterOrEqual(t, len(SecurityTestCases.SQLInjection), 6)

	// Test some specific patterns
	assert.Contains(t, SecurityTestCases.PathTraversal, "../../../etc/passwd")
	assert.Contains(t, SecurityTestCases.CommandInjection, "component; rm -rf /")
	assert.Contains(t, SecurityTestCases.ScriptInjection, "<script>alert('xss')</script>")
	assert.Contains(t, SecurityTestCases.SQLInjection, "'; DROP TABLE users; --")
}

func TestCreateSecureTestEnvironment(t *testing.T) {
	projectDir, cfg := CreateSecureTestEnvironment(t)

	// Verify directory exists
	assert.DirExists(t, projectDir)

	// Verify configuration is set up
	assert.NotNil(t, cfg)
	assert.Equal(t, filepath.Join(projectDir, ".templar", "cache"), cfg.Build.CacheDir)

	// Verify cache directory has secure permissions
	AssertDirectoryPermissions(t, cfg.Build.CacheDir, 0700)
}

func TestAssertFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Test permission assertion
	AssertFilePermissions(t, testFile, 0644)

	// Test permission mismatch detection
	assert.Panics(t, func() {
		AssertFilePermissions(t, testFile, 0600)
	})
}

func TestAssertDirectoryPermissions(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "testdir")

	// Create directory with specific permissions
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// Test permission assertion
	AssertDirectoryPermissions(t, testDir, 0755)

	// Test permission mismatch detection
	assert.Panics(t, func() {
		AssertDirectoryPermissions(t, testDir, 0700)
	})
}

func TestWaitForFileChange(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create initial file
	err := os.WriteFile(testFile, []byte("initial"), 0644)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)
	originalModTime := info.ModTime()

	// Modify file in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(testFile, []byte("modified"), 0644)
	}()

	// Wait for file change
	WaitForFileChange(t, testFile, originalModTime, 200*time.Millisecond)

	// Verify file was actually modified
	newInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.True(t, newInfo.ModTime().After(originalModTime))

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "modified", string(content))
}

func TestWaitForFileChangeTimeout(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create file but don't modify it
	err := os.WriteFile(testFile, []byte("unchanged"), 0644)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)
	originalModTime := info.ModTime()

	// This should timeout and cause the test to fail
	assert.Panics(t, func() {
		WaitForFileChange(t, testFile, originalModTime, 50*time.Millisecond)
	})
}
