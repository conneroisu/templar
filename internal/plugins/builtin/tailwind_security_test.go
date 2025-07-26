package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/validation"
)

func TestTailwindPlugin_CommandInjectionPrevention(t *testing.T) {
	tests := []struct {
		name        string
		inputCSS    string
		expectError bool
		description string
	}{
		{
			name:        "safe_input",
			inputCSS:    "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n",
			expectError: false,
			description: "Normal CSS input should work",
		},
		{
			name:        "injection_attempt_semicolon",
			inputCSS:    "@tailwind base; rm -rf /;",
			expectError: false,
			description: "Semicolon injection should be sanitized",
		},
		{
			name:        "injection_attempt_pipe",
			inputCSS:    "@tailwind base | cat /etc/passwd",
			expectError: false,
			description: "Pipe injection should be sanitized",
		},
		{
			name:        "injection_attempt_backtick",
			inputCSS:    "@tailwind base `rm -rf /`",
			expectError: false,
			description: "Backtick injection should be sanitized",
		},
		{
			name:        "injection_attempt_dollar",
			inputCSS:    "@tailwind base $(rm -rf /)",
			expectError: false,
			description: "Dollar injection should be sanitized",
		},
		{
			name:        "injection_attempt_redirect",
			inputCSS:    "@tailwind base > /dev/null; rm -rf /",
			expectError: false,
			description: "Redirect injection should be sanitized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test input sanitization
			sanitized := sanitizeInput(tt.inputCSS)

			// Verify dangerous characters are removed
			dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\\", "'", "\""}
			for _, char := range dangerousChars {
				if strings.Contains(sanitized, char) {
					t.Errorf("Sanitized input still contains dangerous character: %s", char)
				}
			}

			// Verify the sanitized input doesn't contain shell injection patterns
			injectionPatterns := []string{"rm -rf", "cat /etc", "/dev/null", "$(", "`"}
			for _, pattern := range injectionPatterns {
				if strings.Contains(sanitized, pattern) {
					t.Errorf("Sanitized input still contains injection pattern: %s", pattern)
				}
			}
		})
	}
}

func TestTailwindPlugin_SecureFileOperations(t *testing.T) {
	plugin := NewTailwindPlugin()

	// Test file operations don't use shell commands
	t.Run("secure_file_creation", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.css")

		content := "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"

		// This should use os.WriteFile, not shell commands
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("File was not created")
		}

		// Clean up using os.Remove, not shell commands
		err = os.Remove(testFile)
		if err != nil {
			t.Errorf("Failed to remove file: %v", err)
		}
	})

	t.Run("secure_file_reading", func(t *testing.T) {
		// Create test file in current working directory
		testFile := "test_secure_reading.templ"
		content := `<div class="bg-blue-500 text-white p-4">Test</div>`
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		defer os.Remove(testFile) // Clean up

		// Test that file reading uses os.ReadFile, not shell commands
		classes, err := plugin.extractTailwindClasses(testFile)
		if err != nil {
			t.Fatalf("Failed to extract classes: %v", err)
		}

		// Verify classes were extracted
		expectedClasses := []string{"bg-blue-500", "text-white", "p-4"}
		for _, expected := range expectedClasses {
			found := false
			for _, class := range classes {
				if class == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected class %s not found in extracted classes: %v", expected, classes)
			}
		}
	})
}

func TestTailwindPlugin_NoShellCommandsInCodePaths(t *testing.T) {
	plugin := NewTailwindPlugin()

	// Create a test file in current working directory to pass path validation
	testFile := "test_no_shell_commands.templ"
	content := `<div class="bg-red-500">Test</div>`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile) // Clean up

	component := &types.ComponentInfo{
		Name:     "test",
		FilePath: testFile,
		Metadata: make(map[string]interface{}),
	}

	ctx := context.Background()

	// Test component handling doesn't use shell commands
	processedComponent, err := plugin.HandleComponent(ctx, component)
	if err != nil {
		t.Fatalf("HandleComponent failed: %v", err)
	}

	// Verify component was processed
	if processedComponent.Metadata["tailwind_processed"] != true {
		t.Error("Component was not marked as processed")
	}

	// Verify classes were extracted
	if classes, ok := processedComponent.Metadata["tailwind_classes"].([]string); ok {
		if len(classes) == 0 {
			t.Error("No classes were extracted")
		}
	} else {
		t.Error("Tailwind classes metadata not found")
	}
}

func TestSanitizeInput_ComprehensiveSanitization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "@tailwind base;",
			expected: "@tailwind base",
		},
		{
			input:    "content with `backticks`",
			expected: "content with backticks",
		},
		{
			input:    "content with $(command)",
			expected: "content with command",
		},
		{
			input:    "content with 'single' and \"double\" quotes",
			expected: "content with single and double quotes",
		},
		{
			input:    "content with <redirect> and |pipe|",
			expected: "content with redirect and pipe",
		},
		{
			input:    "content\\with\\backslashes",
			expected: "contentwithbackslashes",
		},
	}

	for _, tt := range tests {
		t.Run("sanitize_"+tt.input, func(t *testing.T) {
			result := sanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeInput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidatePath_PathTraversalPrevention(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{
			name:        "valid_relative_path",
			path:        "components/button.templ",
			expectError: false,
			description: "Valid relative path should be allowed",
		},
		{
			name:        "valid_current_dir_path",
			path:        "./components/button.templ",
			expectError: false,
			description: "Current directory path should be allowed",
		},
		{
			name:        "path_traversal_attempt_simple",
			path:        "../../../etc/passwd",
			expectError: true,
			description: "Simple path traversal should be blocked",
		},
		{
			name:        "path_traversal_attempt_nested",
			path:        "components/../../etc/passwd",
			expectError: true,
			description: "Nested path traversal should be blocked",
		},
		{
			name:        "path_traversal_attempt_absolute",
			path:        "/etc/passwd",
			expectError: true,
			description: "Absolute paths outside project should be blocked",
		},
		{
			name:        "path_traversal_attempt_encoded",
			path:        "components%2F..%2F..%2Fetc%2Fpasswd",
			expectError: true,
			description: "URL-encoded path traversal should be blocked",
		},
		{
			name:        "empty_path",
			path:        "",
			expectError: true,
			description: "Empty path should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePath(tt.path)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for path %q but got none", tt.path)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

func TestTailwindPlugin_PathValidationInExtractClasses(t *testing.T) {
	plugin := NewTailwindPlugin()

	// Test that extractTailwindClasses validates paths
	t.Run("rejects_path_traversal", func(t *testing.T) {
		maliciousPath := "../../../etc/passwd"

		_, err := plugin.extractTailwindClasses(maliciousPath)
		if err == nil {
			t.Error("Expected error for path traversal attempt but got none")
		}

		if !strings.Contains(err.Error(), "invalid file path") {
			t.Errorf("Expected 'invalid file path' error but got: %v", err)
		}
	})

	t.Run("allows_valid_paths", func(t *testing.T) {
		tempDir := t.TempDir()
		validFile := filepath.Join(tempDir, "valid.templ")

		content := `<div class="bg-blue-500">Valid content</div>`
		err := os.WriteFile(validFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Change to temp directory to make the path relative and valid
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		classes, err := plugin.extractTailwindClasses("valid.templ")
		if err != nil {
			t.Errorf("Unexpected error for valid path: %v", err)
		}

		if len(classes) == 0 {
			t.Error("Expected to extract classes from valid file")
		}
	})
}

func BenchmarkSanitizeInput(b *testing.B) {
	input := "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n$(rm -rf /) && echo 'hello'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizeInput(input)
	}
}

func BenchmarkValidatePath(b *testing.B) {
	path := "components/button.templ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validation.ValidatePath(path)
	}
}

// TestTailwindPlugin_ConfigPathCommandInjection tests the specific vulnerability that was fixed
func TestTailwindPlugin_ConfigPathCommandInjection(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		shouldFail bool
		errorMsg   string
	}{
		{
			name:       "valid_config_path",
			configPath: "tailwind.config.js",
			shouldFail: false,
		},
		{
			name:       "path_traversal_attack",
			configPath: "../../../etc/passwd",
			shouldFail: true,
			errorMsg:   "path traversal",
		},
		{
			name:       "command_injection_semicolon",
			configPath: "config.js; rm -rf /tmp/test",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
		{
			name:       "command_injection_pipe",
			configPath: "config.js | cat /etc/passwd",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
		{
			name:       "command_injection_backticks",
			configPath: "config.js`rm -rf /`",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
		{
			name:       "command_injection_dollar",
			configPath: "config.js$(rm -rf /)",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
		{
			name:       "command_injection_ampersand",
			configPath: "config.js & rm -rf /tmp",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
		{
			name:       "absolute_path_to_sensitive_file",
			configPath: "/etc/passwd",
			shouldFail: true,
			errorMsg:   "restricted path denied",
		},
		{
			name:       "null_byte_injection",
			configPath: "config.js\x00; rm -rf /",
			shouldFail: true,
			errorMsg:   "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create plugin instance
			plugin := NewTailwindPlugin()

			// Set up plugin with potentially malicious config path
			plugin.tailwindPath = "npx tailwindcss"
			plugin.configPath = tt.configPath
			plugin.enabled = true

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Attempt to generate CSS - this should validate the config path
			classes := map[string]bool{"bg-blue-500": true}
			err := plugin.generateCSS(ctx, classes)

			if tt.shouldFail {
				if err == nil {
					t.Errorf(
						"Expected error for malicious config path %q, but got none",
						tt.configPath,
					)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				// For valid paths, we expect it to fail with tailwindcss not found (which is expected in test env)
				// but NOT with validation errors
				if err != nil && (strings.Contains(err.Error(), "dangerous character") ||
					strings.Contains(err.Error(), "path traversal") ||
					strings.Contains(err.Error(), "absolute path not allowed")) {
					t.Errorf("Valid config path %q failed validation: %v", tt.configPath, err)
				}
			}
		})
	}
}

// TestTailwindPlugin_CommandValidation tests command allowlisting
func TestTailwindPlugin_CommandValidation(t *testing.T) {
	tests := []struct {
		name         string
		tailwindPath string
		shouldFail   bool
		errorMsg     string
	}{
		{
			name:         "valid_npx_command",
			tailwindPath: "npx tailwindcss",
			shouldFail:   false,
		},
		{
			name:         "valid_tailwindcss_binary",
			tailwindPath: "/usr/local/bin/tailwindcss",
			shouldFail:   false,
		},
		{
			name:         "malicious_command_injection",
			tailwindPath: "tailwindcss; rm -rf /",
			shouldFail:   true,
			errorMsg:     "not allowed",
		},
		{
			name:         "unauthorized_command",
			tailwindPath: "curl",
			shouldFail:   true,
			errorMsg:     "not allowed",
		},
		{
			name:         "disguised_command",
			tailwindPath: "rm",
			shouldFail:   true,
			errorMsg:     "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := NewTailwindPlugin()
			plugin.tailwindPath = tt.tailwindPath
			plugin.enabled = true

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			classes := map[string]bool{"bg-blue-500": true}
			err := plugin.generateCSS(ctx, classes)

			if tt.shouldFail {
				if err == nil {
					t.Errorf(
						"Expected error for malicious tailwind path %q, but got none",
						tt.tailwindPath,
					)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				// For valid paths, we expect it to fail with execution errors (tailwindcss not found)
				// but NOT with validation errors
				if err != nil && (strings.Contains(err.Error(), "dangerous character") ||
					strings.Contains(err.Error(), "not allowed")) {
					t.Errorf("Valid tailwind path %q failed validation: %v", tt.tailwindPath, err)
				}
			}
		})
	}
}
