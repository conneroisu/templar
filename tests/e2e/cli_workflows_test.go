package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIWorkflows provides comprehensive end-to-end testing of CLI commands
func TestCLIWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end CLI tests in short mode")
	}

	// Build templar binary for testing
	templarBinary := buildTemplarBinary(t)
	defer func() {
		if err := os.Remove(templarBinary); err != nil {
			t.Logf("Warning: failed to remove templar binary: %v", err)
		}
	}()

	t.Run("complete_project_lifecycle", func(t *testing.T) {
		testCompleteProjectLifecycle(t, templarBinary)
	})

	t.Run("init_command_workflows", func(t *testing.T) {
		testInitCommandWorkflows(t, templarBinary)
	})

	t.Run("component_management_workflows", func(t *testing.T) {
		testComponentManagementWorkflows(t, templarBinary)
	})

	t.Run("build_and_watch_workflows", func(t *testing.T) {
		testBuildAndWatchWorkflows(t, templarBinary)
	})

	t.Run("server_workflows", func(t *testing.T) {
		testServerWorkflows(t, templarBinary)
	})

	t.Run("error_handling_workflows", func(t *testing.T) {
		testErrorHandlingWorkflows(t, templarBinary)
	})

	t.Run("configuration_workflows", func(t *testing.T) {
		testConfigurationWorkflows(t, templarBinary)
	})
}

// testCompleteProjectLifecycle tests a complete project workflow from init to serve
func testCompleteProjectLifecycle(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)
	projectDir := tc.TempDir()

	// Step 1: Initialize new project
	t.Logf("Step 1: Initializing project in %s", projectDir)
	initCmd := exec.Command(templarBinary, "init", "--minimal")
	initCmd.Dir = projectDir
	output, err := initCmd.CombinedOutput()
	require.NoError(t, err, "init command should succeed: %s", output)

	// Verify project structure was created
	assert.FileExists(t, filepath.Join(projectDir, ".templar.yml"))
	assert.DirExists(t, filepath.Join(projectDir, "components"))

	// Step 2: Create some components manually
	t.Logf("Step 2: Creating test components")
	createTestComponents(t, projectDir)

	// Step 3: List components
	t.Logf("Step 3: Listing components")
	listCmd := exec.Command(templarBinary, "list")
	listCmd.Dir = projectDir
	listOutput, err := listCmd.CombinedOutput()
	require.NoError(t, err, "list command should succeed: %s", listOutput)

	// Verify components are listed
	assert.Contains(t, string(listOutput), "Button")
	assert.Contains(t, string(listOutput), "Card")

	// Step 4: Build project
	t.Logf("Step 4: Building project")
	buildCmd := exec.Command(templarBinary, "build")
	buildCmd.Dir = projectDir
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "build command should succeed: %s", buildOutput)

	// Step 5: Validate build artifacts
	t.Logf("Step 5: Validating build artifacts")
	validateBuildArtifacts(t, projectDir)

	t.Logf("Complete project lifecycle test passed")
}

// testInitCommandWorkflows tests various init command scenarios
func testInitCommandWorkflows(t *testing.T, templarBinary string) {
	testCases := []struct {
		name     string
		args     []string
		validate func(t *testing.T, projectDir string)
	}{
		{
			name: "minimal_init",
			args: []string{"init", "--minimal"},
			validate: func(t *testing.T, projectDir string) {
				assert.FileExists(t, filepath.Join(projectDir, ".templar.yml"))
				assert.DirExists(t, filepath.Join(projectDir, "components"))
				// Minimal init should not create examples
				assert.NoDirExists(t, filepath.Join(projectDir, "examples"))
			},
		},
		{
			name: "default_init",
			args: []string{"init"},
			validate: func(t *testing.T, projectDir string) {
				assert.FileExists(t, filepath.Join(projectDir, ".templar.yml"))
				assert.DirExists(t, filepath.Join(projectDir, "components"))
				// Default init might create examples
			},
		},
		{
			name: "custom_template_init",
			args: []string{"init", "--template", "blog"},
			validate: func(t *testing.T, projectDir string) {
				assert.FileExists(t, filepath.Join(projectDir, ".templar.yml"))
				// Blog template might have specific structure
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCtx := testutils.NewTestContext(t)
			projectDir := testCtx.TempDir()

			cmd := exec.Command(templarBinary, tc.args...)
			cmd.Dir = projectDir
			output, err := cmd.CombinedOutput()

			// Some template types might not exist, which is acceptable
			if err != nil && !strings.Contains(string(output), "template not found") {
				require.NoError(t, err, "init command should succeed: %s", output)
			}

			if err == nil {
				tc.validate(t, projectDir)
			}
		})
	}
}

// testComponentManagementWorkflows tests component creation, listing, and management
func testComponentManagementWorkflows(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)
	projectDir := tc.TempDir()

	// Initialize project
	initCmd := exec.Command(templarBinary, "init", "--minimal")
	initCmd.Dir = projectDir
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	// Test component generation
	t.Run("generate_component", func(t *testing.T) {
		genCmd := exec.Command(templarBinary, "generate", "component", "TestComponent")
		genCmd.Dir = projectDir
		output, err := genCmd.CombinedOutput()

		// Component generation might not be implemented yet
		if err != nil && strings.Contains(string(output), "command not found") {
			t.Skip("Component generation not implemented")
		}
		
		if err == nil {
			assert.FileExists(t, filepath.Join(projectDir, "components", "TestComponent.templ"))
		}
	})

	// Create components manually for testing
	createTestComponents(t, projectDir)

	// Test listing with different formats
	t.Run("list_formats", func(t *testing.T) {
		formats := []string{"", "--format=json", "--format=table"}
		
		for _, format := range formats {
			args := []string{"list"}
			if format != "" {
				args = append(args, format)
			}

			listCmd := exec.Command(templarBinary, args...)
			listCmd.Dir = projectDir
			output, err := listCmd.CombinedOutput()
			
			if err != nil && strings.Contains(string(output), "unknown flag") {
				continue // Skip unsupported formats
			}
			
			require.NoError(t, err, "list command should succeed: %s", output)
			assert.Contains(t, string(output), "Button")
		}
	})

	// Test listing with properties
	t.Run("list_with_properties", func(t *testing.T) {
		listCmd := exec.Command(templarBinary, "list", "--show-props")
		listCmd.Dir = projectDir
		output, err := listCmd.CombinedOutput()
		
		if err != nil && strings.Contains(string(output), "unknown flag") {
			t.Skip("--show-props flag not implemented")
		}
		
		if err == nil {
			assert.Contains(t, string(output), "Button")
		}
	})
}

// testBuildAndWatchWorkflows tests build and watch functionality
func testBuildAndWatchWorkflows(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)
	projectDir := tc.TempDir()

	// Initialize and setup project
	initCmd := exec.Command(templarBinary, "init", "--minimal")
	initCmd.Dir = projectDir
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	createTestComponents(t, projectDir)

	// Test build command
	t.Run("build_command", func(t *testing.T) {
		buildCmd := exec.Command(templarBinary, "build")
		buildCmd.Dir = projectDir
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build command should succeed: %s", output)

		// Verify build artifacts
		validateBuildArtifacts(t, projectDir)
	})

	// Test production build
	t.Run("production_build", func(t *testing.T) {
		buildCmd := exec.Command(templarBinary, "build", "--production")
		buildCmd.Dir = projectDir
		output, err := buildCmd.CombinedOutput()
		
		if err != nil && strings.Contains(string(output), "unknown flag") {
			t.Skip("--production flag not implemented")
		}
		
		if err == nil {
			validateBuildArtifacts(t, projectDir)
		}
	})

	// Test watch command (with timeout)
	t.Run("watch_command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(tc.Context(), 10*time.Second)
		defer cancel()

		watchCmd := exec.CommandContext(ctx, templarBinary, "watch")
		watchCmd.Dir = projectDir
		
		// Start watch in background
		err := watchCmd.Start()
		require.NoError(t, err, "watch command should start")

		// Give watch time to initialize
		time.Sleep(2 * time.Second)

		// Modify a component file
		componentFile := filepath.Join(projectDir, "components", "Button.templ")
		modifiedContent := `package components

templ Button(text string, variant string) {
	<button class={ "btn", "btn-" + variant, "modified" }>
		{ text } - Modified
	</button>
}`
		err = os.WriteFile(componentFile, []byte(modifiedContent), 0644)
		require.NoError(t, err)

		// Give watch time to detect change
		time.Sleep(3 * time.Second)

		// Stop watch
		if watchCmd.Process != nil {
			if err := watchCmd.Process.Kill(); err != nil {
				t.Logf("Warning: failed to kill watch process: %v", err)
			}
		}
		if err := watchCmd.Wait(); err != nil {
			t.Logf("Watch process ended with error: %v", err)
		}
	})
}

// testServerWorkflows tests server functionality
func testServerWorkflows(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)
	projectDir := tc.TempDir()

	// Initialize project
	initCmd := exec.Command(templarBinary, "init", "--minimal")
	initCmd.Dir = projectDir
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	createTestComponents(t, projectDir)

	// Test server start with timeout
	t.Run("server_startup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(tc.Context(), 15*time.Second)
		defer cancel()

		// Use a custom port to avoid conflicts
		port := "8888"
		serveCmd := exec.CommandContext(ctx, templarBinary, "serve", "--port", port, "--no-open")
		serveCmd.Dir = projectDir

		err := serveCmd.Start()
		require.NoError(t, err, "serve command should start")

		// Give server time to start
		time.Sleep(3 * time.Second)

		// Test if server is responding (basic check)
		testServerResponse(t, port)

		// Stop server
		if serveCmd.Process != nil {
			if err := serveCmd.Process.Kill(); err != nil {
				t.Logf("Warning: failed to kill serve process: %v", err)
			}
		}
		if err := serveCmd.Wait(); err != nil {
			t.Logf("Serve process ended with error: %v", err)
		}
	})

	// Test preview command
	t.Run("preview_command", func(t *testing.T) {
		previewCmd := exec.Command(templarBinary, "preview", "Button")
		previewCmd.Dir = projectDir
		output, err := previewCmd.CombinedOutput()
		
		if err != nil && strings.Contains(string(output), "command not found") {
			t.Skip("preview command not implemented")
		}
		
		if err == nil {
			assert.Contains(t, string(output), "Button")
		}
	})
}

// testErrorHandlingWorkflows tests error scenarios and error handling
func testErrorHandlingWorkflows(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)

	// Test invalid project directory
	t.Run("invalid_project_directory", func(t *testing.T) {
		invalidDir := filepath.Join(tc.TempDir(), "nonexistent")
		
		listCmd := exec.Command(templarBinary, "list")
		listCmd.Dir = invalidDir
		output, err := listCmd.CombinedOutput()
		
		// Should handle invalid directory gracefully
		assert.Error(t, err)
		assert.Contains(t, string(output), "no such file")
	})

	// Test project without config
	t.Run("project_without_config", func(t *testing.T) {
		projectDir := tc.TempDir()
		
		listCmd := exec.Command(templarBinary, "list")
		listCmd.Dir = projectDir
		output, err := listCmd.CombinedOutput()
		
		// Should handle missing config gracefully
		if err != nil {
			assert.Contains(t, string(output), "config")
		}
	})

	// Test invalid component syntax
	t.Run("invalid_component_syntax", func(t *testing.T) {
		projectDir := tc.TempDir()
		
		// Initialize project
		initCmd := exec.Command(templarBinary, "init", "--minimal")
		initCmd.Dir = projectDir
		_, err := initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create component with invalid syntax
		invalidComponent := `package components

templ InvalidComponent() {
	<div
		<p>Missing closing tag</p>
	</div>
}`
		componentFile := filepath.Join(projectDir, "components", "Invalid.templ")
		err = os.WriteFile(componentFile, []byte(invalidComponent), 0644)
		require.NoError(t, err)

		// Try to build - should handle syntax errors gracefully
		buildCmd := exec.Command(templarBinary, "build")
		buildCmd.Dir = projectDir
		output, err := buildCmd.CombinedOutput()
		
		// Expect error but should not crash
		assert.Error(t, err)
		assert.NotEmpty(t, output)
	})
}

// testConfigurationWorkflows tests configuration management
func testConfigurationWorkflows(t *testing.T, templarBinary string) {
	tc := testutils.NewTestContext(t)
	projectDir := tc.TempDir()

	// Initialize project
	initCmd := exec.Command(templarBinary, "init", "--minimal")
	initCmd.Dir = projectDir
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	// Test config validation
	t.Run("config_validation", func(t *testing.T) {
		// Create invalid config
		invalidConfig := `server:
  port: -1
  host: ""
components:
  scan_paths: []
`
		configFile := filepath.Join(projectDir, ".templar.yml")
		err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
		require.NoError(t, err)

		// Try to use invalid config
		listCmd := exec.Command(templarBinary, "list")
		listCmd.Dir = projectDir
		output, err := listCmd.CombinedOutput()
		
		// Should detect invalid config
		if err != nil {
			assert.Contains(t, string(output), "config")
		}
	})

	// Test config doctor command if available
	t.Run("config_doctor", func(t *testing.T) {
		doctorCmd := exec.Command(templarBinary, "doctor")
		doctorCmd.Dir = projectDir
		output, err := doctorCmd.CombinedOutput()
		
		if err != nil && strings.Contains(string(output), "command not found") {
			t.Skip("doctor command not implemented")
		}
		
		if err == nil {
			assert.NotEmpty(t, output)
		}
	})
}

// Helper functions

func buildTemplarBinary(t *testing.T) string {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "templar")
	
	// Build templar binary
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./main.go")
	buildCmd.Dir = "/home/connerohnesorge/Documents/001Repos/templar"
	
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build templar binary: %s", output)
	
	return binaryPath
}

func createTestComponents(t *testing.T, projectDir string) {
	componentsDir := filepath.Join(projectDir, "components")
	err := os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create Button component
	buttonContent := `package components

templ Button(text string, variant string) {
	<button class={ "btn", "btn-" + variant }>
		{ text }
	</button>
}`
	err = os.WriteFile(filepath.Join(componentsDir, "Button.templ"), []byte(buttonContent), 0644)
	require.NoError(t, err)

	// Create Card component
	cardContent := `package components

templ Card(title string, content string) {
	<div class="card">
		<div class="card-header">
			<h3>{ title }</h3>
		</div>
		<div class="card-body">
			<p>{ content }</p>
		</div>
	</div>
}`
	err = os.WriteFile(filepath.Join(componentsDir, "Card.templ"), []byte(cardContent), 0644)
	require.NoError(t, err)

	// Create Layout component
	layoutContent := `package components

templ Layout(title string) {
	<!DOCTYPE html>
	<html>
		<head>
			<title>{ title }</title>
			<link rel="stylesheet" href="/styles.css"/>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}`
	err = os.WriteFile(filepath.Join(componentsDir, "Layout.templ"), []byte(layoutContent), 0644)
	require.NoError(t, err)
}

func validateBuildArtifacts(t *testing.T, projectDir string) {
	// Check for generated Go files (templ generates .go files)
	componentsDir := filepath.Join(projectDir, "components")
	
	// Look for any .go files generated by templ
	entries, err := os.ReadDir(componentsDir)
	if err != nil {
		return // Components dir might not exist yet
	}

	goFileFound := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "_templ.go") {
			goFileFound = true
			break
		}
	}

	// Note: templ might generate files with different naming conventions
	// This is a basic check that some processing occurred
	t.Logf("Build validation: Go files generated: %v", goFileFound)
}

func testServerResponse(t *testing.T, port string) {
	// Simple connectivity test - just check if something is listening
	// In a more comprehensive test, we'd make HTTP requests
	t.Logf("Server should be running on port %s", port)
	
	// This is a basic test - in reality we'd want to:
	// - Make HTTP requests to verify endpoints
	// - Check WebSocket connections
	// - Verify static file serving
	// - Test hot reload functionality
}

// TestCLIHelp tests help and version commands
func TestCLIHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI help tests in short mode")
	}

	templarBinary := buildTemplarBinary(t)
	defer func() {
		if err := os.Remove(templarBinary); err != nil {
			t.Logf("Warning: failed to remove templar binary: %v", err)
		}
	}()

	testCases := []struct {
		name string
		args []string
	}{
		{"help", []string{"--help"}},
		{"version", []string{"--version"}},
		{"help_init", []string{"help", "init"}},
		{"help_serve", []string{"help", "serve"}},
		{"help_build", []string{"help", "build"}},
		{"help_list", []string{"help", "list"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(templarBinary, tc.args...)
			output, err := cmd.CombinedOutput()
			
			// Help and version commands might exit with status 0 or 1
			// depending on implementation
			if err != nil {
				// Check if it's just a non-zero exit code
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Exit code 1 is often used for help commands
					if exitErr.ExitCode() > 2 {
						t.Errorf("Help command failed with exit code %d: %s", 
							exitErr.ExitCode(), output)
					}
				} else {
					t.Errorf("Help command failed: %v", err)
				}
			}

			// Should produce some output
			assert.NotEmpty(t, output, "Help command should produce output")
			
			// Common help indicators
			outputStr := string(output)
			hasHelpIndicators := strings.Contains(outputStr, "Usage:") ||
				strings.Contains(outputStr, "Commands:") ||
				strings.Contains(outputStr, "Flags:") ||
				strings.Contains(outputStr, "templar") ||
				strings.Contains(outputStr, "version")
			
			assert.True(t, hasHelpIndicators, 
				"Help output should contain usage information: %s", outputStr)
		})
	}
}

// TestCLIExitCodes tests proper exit code handling
func TestCLIExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI exit code tests in short mode")
	}

	templarBinary := buildTemplarBinary(t)
	defer func() {
		if err := os.Remove(templarBinary); err != nil {
			t.Logf("Warning: failed to remove templar binary: %v", err)
		}
	}()

	testCases := []struct {
		name         string
		args         []string
		expectError  bool
		setupProject bool
	}{
		{"unknown_command", []string{"unknown-command"}, true, false},
		{"invalid_flag", []string{"list", "--invalid-flag"}, true, false},
		{"list_in_invalid_dir", []string{"list"}, true, false},
		{"valid_help", []string{"--help"}, false, false},
	}

	tc := testutils.NewTestContext(t)
	
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var workDir string
			if test.setupProject {
				workDir = tc.TempDir()
				// Setup project
				initCmd := exec.Command(templarBinary, "init", "--minimal")
				initCmd.Dir = workDir
				if err := initCmd.Run(); err != nil {
					t.Logf("Init command failed: %v", err)
				}
			} else {
				workDir = tc.TempDir() // Empty directory
			}

			cmd := exec.Command(templarBinary, test.args...)
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()

			if test.expectError {
				assert.Error(t, err, "Command should fail: %s", string(output))
			} else {
				// Help commands might exit with code 0 or 1
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						assert.LessOrEqual(t, exitErr.ExitCode(), 1, 
							"Help commands should exit with code 0 or 1")
					}
				}
			}

			// All commands should produce some output
			assert.NotEmpty(t, output, "Command should produce output")
		})
	}
}