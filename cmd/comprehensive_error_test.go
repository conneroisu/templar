package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitCommandErrorPaths tests error scenarios specific to the init command
func TestInitCommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, dir string)
		expectError bool
		errorContains string
	}{
		{
			name:        "init with path traversal attempt",
			args:        []string{"../../../invalid"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "invalid",
		},
		{
			name:        "init with command injection attempt",
			args:        []string{"project; rm -rf /"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "invalid",
		},
		{
			name:        "init with permission denied",
			args:        []string{"readonly-project"},
			setup:       createReadOnlyTestDir,
			cleanup:     cleanupReadOnlyTestDir,
			expectError: true,
			errorContains: "permission",
		},
		{
			name:        "init with existing non-empty directory",
			args:        []string{"existing-dir"},
			setup:       createNonEmptyTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "not empty",
		},
	}

	runCommandErrorTests(t, tests, initCmd)
}

// TestServeCommandErrorPaths tests error scenarios specific to the serve command
func TestServeCommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, dir string)
		expectError bool
		errorContains string
	}{
		{
			name:        "serve with invalid port range",
			args:        []string{"--port", "99999"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "port",
		},
		{
			name:        "serve with negative port",
			args:        []string{"--port", "-8080"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "port",
		},
		{
			name:        "serve with invalid host format",
			args:        []string{"--host", "invalid..host"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "host",
		},
		{
			name:        "serve with malformed config",
			args:        []string{"--config", "invalid-config.yml"},
			setup:       createInvalidConfigTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "config",
		},
	}

	runCommandErrorTests(t, tests, serveCmd)
}

// TestBuildCommandErrorPaths tests error scenarios specific to the build command
func TestBuildCommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, dir string)
		expectError bool
		errorContains string
	}{
		{
			name:        "build with missing templ binary",
			args:        []string{},
			setup:       createInvalidPathTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "templ",
		},
		{
			name:        "build with invalid syntax in templ file",
			args:        []string{},
			setup:       createInvalidTemplTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "build",
		},
		{
			name:        "build with permission denied on cache",
			args:        []string{},
			setup:       createReadOnlyCacheTestDir,
			cleanup:     cleanupReadOnlyTestDir,
			expectError: true,
			errorContains: "permission",
		},
	}

	runCommandErrorTests(t, tests, buildCmd)
}

// TestListCommandErrorPaths tests error scenarios specific to the list command
func TestListCommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, dir string)
		expectError bool
		errorContains string
	}{
		{
			name:        "list with no components directory",
			args:        []string{},
			setup:       createEmptyTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "components",
		},
		{
			name:        "list with permission denied on components",
			args:        []string{},
			setup:       createReadOnlyComponentsTestDir,
			cleanup:     cleanupReadOnlyTestDir,
			expectError: true,
			errorContains: "permission",
		},
	}

	runCommandErrorTests(t, tests, listCmd)
}

// TestPreviewCommandErrorPaths tests error scenarios specific to the preview command
func TestPreviewCommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, dir string)
		expectError bool
		errorContains string
	}{
		{
			name:        "preview with non-existent component",
			args:        []string{"NonExistentComponent"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "not found",
		},
		{
			name:        "preview with path traversal attempt",
			args:        []string{"../../../etc/passwd"},
			setup:       createTempTestDir,
			cleanup:     cleanupTestDir,
			expectError: true,
			errorContains: "invalid",
		},
	}

	runCommandErrorTests(t, tests, previewCmd)
}

// TestConfigValidationErrorPaths tests configuration validation error scenarios
func TestConfigValidationErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorContains string
	}{
		{
			name: "invalid YAML syntax",
			configYAML: `
server:
  port: 8080
invalid_yaml: [unclosed
`,
			expectError: true,
			errorContains: "yaml",
		},
		{
			name: "invalid port range",
			configYAML: `
server:
  port: 70000
`,
			expectError: true,
			errorContains: "port",
		},
		{
			name: "invalid host format",
			configYAML: `
server:
  host: "invalid..hostname"
`,
			expectError: true,
			errorContains: "host",
		},
		{
			name: "empty scan paths",
			configYAML: `
components:
  scan_paths: []
`,
			expectError: true,
			errorContains: "scan_paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			configPath := filepath.Join(tempDir, ".templar.yml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.configYAML), 0644))

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldDir) }()
			require.NoError(t, os.Chdir(tempDir))

			viper.Reset()

			// Test config validation through serve command
			serveCmd.SetArgs([]string{})
			err = serveCmd.Execute()

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" && err != nil {
					errorMsg := strings.ToLower(err.Error())
					assert.Contains(t, errorMsg, strings.ToLower(tt.errorContains),
						"Error should contain '%s' for test: %s. Got: %s", tt.errorContains, tt.name, err.Error())
				}
			} else {
				// May still fail for other reasons, just not config validation
				t.Logf("Command result: %v", err)
			}
		})
	}
}

// Helper functions for test setup

func createTempTestDir(t *testing.T) string {
	return t.TempDir()
}

func createEmptyTestDir(t *testing.T) string {
	return t.TempDir()
}

func createReadOnlyTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(tempDir, 0444))
	}
	return tempDir
}

func createNonEmptyTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	existingDir := filepath.Join(tempDir, "existing-dir")
	require.NoError(t, os.Mkdir(existingDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(existingDir, "file.txt"), []byte("content"), 0644))
	return tempDir
}

func createInvalidConfigTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	invalidConfig := `
server:
  port: invalid_port
  host: [invalid yaml syntax
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "invalid-config.yml"), []byte(invalidConfig), 0644))
	return tempDir
}

func createInvalidPathTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	t.Setenv("PATH", "") // Remove PATH to simulate missing templ binary
	return tempDir
}

func createInvalidTemplTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	componentsDir := filepath.Join(tempDir, "components")
	require.NoError(t, os.Mkdir(componentsDir, 0755))

	invalidTempl := `package components

templ InvalidComponent( {
	<div>Unclosed syntax
}
`
	require.NoError(t, os.WriteFile(filepath.Join(componentsDir, "invalid.templ"), []byte(invalidTempl), 0644))
	return tempDir
}

func createReadOnlyCacheTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, ".templar", "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(cacheDir, 0444))
	}
	return tempDir
}

func createReadOnlyComponentsTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	componentsDir := filepath.Join(tempDir, "components")
	require.NoError(t, os.Mkdir(componentsDir, 0755))
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(componentsDir, 0000))
	}
	return tempDir
}

func cleanupTestDir(t *testing.T, dir string) {
	// t.TempDir() handles cleanup automatically
}

func cleanupReadOnlyTestDir(t *testing.T, dir string) {
	if runtime.GOOS != "windows" {
		// Restore permissions before cleanup
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0755)
			}
			return nil
		})
	}
}

// runCommandErrorTests is a helper function to reduce code duplication
func runCommandErrorTests(t *testing.T, tests []struct {
	name          string
	args          []string
	setup         func(t *testing.T) string
	cleanup       func(t *testing.T, dir string)
	expectError   bool
	errorContains string
}, cmd interface{ SetArgs([]string); Execute() error }) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := tt.setup(t)
			defer tt.cleanup(t, tempDir)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldDir) }()
			require.NoError(t, os.Chdir(tempDir))

			viper.Reset()

			cmd.SetArgs(tt.args)
			err = cmd.Execute()

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" && err != nil {
					errorMsg := strings.ToLower(err.Error())
					assert.Contains(t, errorMsg, strings.ToLower(tt.errorContains),
						"Error should contain '%s' for test: %s. Got: %s", tt.errorContains, tt.name, err.Error())
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}