package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateArgument_EdgeCases tests additional edge cases not covered in main validation tests
func TestValidateArgument_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		expectError bool
		errorType   string
	}{
		// Unicode and encoding edge cases
		{
			name:        "unicode null character",
			arg:         "file\x00.templ",
			expectError: false, // Should be allowed if no dangerous chars
		},
		{
			name:        "unicode control characters",
			arg:         "file\u0001\u0002.templ",
			expectError: false, // Control chars not explicitly blocked
		},
		{
			name:        "unicode homoglyph attack - cyrillic",
			arg:         "fіle.templ", // 'і' is cyrillic, looks like 'i'
			expectError: false,        // Unicode homoglyphs not blocked
		},
		{
			name:        "unicode right-to-left override",
			arg:         "file\u202e.templ",
			expectError: false, // RTL override not blocked
		},
		{
			name:        "unicode zero-width characters",
			arg:         "fi\u200ble.templ", // zero-width space
			expectError: false,              // Zero-width chars not blocked
		},

		// Path edge cases
		{
			name:        "extremely long path",
			arg:         strings.Repeat("a", 4096) + ".templ",
			expectError: false, // Long paths not explicitly blocked
		},
		{
			name:        "path with only dots",
			arg:         "....",
			expectError: true,
			errorType:   "path traversal", // Contains ".."
		},
		{
			name:        "path with mixed separators",
			arg:         "components\\windows\\style.templ",
			expectError: true, // Backslash is blocked by dangerous chars
			errorType:   "dangerous character",
		},
		{
			name:        "path with trailing dot",
			arg:         "component.templ.",
			expectError: false, // Trailing dots not blocked
		},
		{
			name:        "path with spaces and tabs",
			arg:         "component with spaces\t.templ",
			expectError: false, // Spaces and tabs not blocked
		},
		{
			name:        "path with newlines",
			arg:         "component\n.templ",
			expectError: false, // Newlines not explicitly blocked
		},

		// URL-encoded injection attempts
		{
			name:        "url encoded semicolon",
			arg:         "file%3Brm+-rf+/.templ",
			expectError: false, // URL encoding not decoded
		},
		{
			name:        "double url encoded",
			arg:         "file%253B.templ", // %253B = %3B = ;
			expectError: false,             // Double encoding not handled
		},
		{
			name:        "hex encoded characters",
			arg:         "file\x3B.templ", // \x3B = semicolon
			expectError: true,
			errorType:   "dangerous character",
		},

		// Case sensitivity edge cases
		{
			name:        "uppercase dangerous chars",
			arg:         "file.TEMPL",
			expectError: false, // No uppercase dangerous chars
		},

		// Empty and whitespace edge cases
		{
			name:        "empty string",
			arg:         "",
			expectError: false, // Empty string should be allowed
		},
		{
			name:        "only whitespace",
			arg:         "   ",
			expectError: false, // Whitespace not blocked
		},
		{
			name:        "whitespace with dangerous char",
			arg:         "  ;  ",
			expectError: true,
			errorType:   "dangerous character",
		},

		// Path traversal variations
		{
			name:        "encoded path traversal",
			arg:         "%2E%2E%2F", // ../
			expectError: false,       // Not decoded
		},
		{
			name:        "windows path traversal",
			arg:         "..\\..\\windows",
			expectError: true,
			errorType:   "dangerous character", // Backslash caught first
		},
		{
			name:        "mixed slash path traversal",
			arg:         "../.\\../etc",
			expectError: true,
			errorType:   "dangerous character", // Backslash caught first
		},

		// Boundary conditions for allowed paths
		{
			name:        "root tmp path",
			arg:         "/tmp",
			expectError: true, // Only /tmp/ subdirectories allowed
			errorType:   "absolute path",
		},
		{
			name:        "tmp with trailing slash",
			arg:         "/tmp/",
			expectError: false, // Should be allowed
		},
		{
			name:        "usr without local",
			arg:         "/usr/bin",
			expectError: false, // Should be allowed (starts with /usr/)
		},
		{
			name:        "proc filesystem",
			arg:         "/proc/self/environ",
			expectError: true,
			errorType:   "absolute path", // Not in allowed list
		},
		{
			name:        "dev filesystem",
			arg:         "/dev/null",
			expectError: true,
			errorType:   "absolute path", // Not in allowed list
		},

		// Special filenames
		{
			name:        "dot file",
			arg:         ".hidden",
			expectError: false, // Hidden files should be allowed
		},
		{
			name:        "double dot file",
			arg:         "..hidden",
			expectError: true,
			errorType:   "path traversal", // Contains ..
		},
		{
			name:        "filename with colon",
			arg:         "component:alt.templ",
			expectError: false, // Colons not blocked
		},

		// Injection via different quoting mechanisms
		{
			name:        "argument with equals",
			arg:         "VAR=value",
			expectError: false, // Equals not blocked
		},
		{
			name:        "argument with hash comment",
			arg:         "file.templ#comment",
			expectError: false, // Hash not blocked
		},
		{
			name:        "argument with tilde expansion",
			arg:         "~/file.templ",
			expectError: false, // Tilde not blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgument(tt.arg)

			if tt.expectError {
				require.Error(t, err, "Expected error for argument '%s'", tt.arg)
				if tt.errorType != "" {
					assert.Contains(
						t,
						strings.ToLower(err.Error()),
						tt.errorType,
						"Error should contain expected type: %s, got: %s",
						tt.errorType,
						err.Error(),
					)
				}
			} else {
				assert.NoError(t, err, "Expected no error for argument '%s', got: %v", tt.arg, err)
			}
		})
	}
}

// TestValidateCommand_EdgeCases tests edge cases for command validation
func TestValidateCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		allowedCommands map[string]bool
		expectError     bool
		errorType       string
	}{
		// Case sensitivity
		{
			name:    "uppercase command",
			command: "TEMPL",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed", // Case sensitive
		},
		{
			name:    "mixed case command",
			command: "Templ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed", // Case sensitive
		},

		// Empty and whitespace
		{
			name:    "empty command",
			command: "",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:    "whitespace command",
			command: "   ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:    "command with leading space",
			command: " templ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed", // Exact match required
		},
		{
			name:    "command with trailing space",
			command: "templ ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed", // Exact match required
		},

		// Unicode edge cases
		{
			name:    "command with unicode",
			command: "templ\u200b", // zero-width space
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:    "homoglyph attack",
			command: "tempІ", // cyrillic І instead of l
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},

		// Path-like commands
		{
			name:    "relative path command",
			command: "./templ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},
		{
			name:    "absolute path command",
			command: "/usr/bin/templ",
			allowedCommands: map[string]bool{
				"templ": true,
			},
			expectError: true,
			errorType:   "not allowed",
		},

		// Special characters in command name
		{
			name:    "command with dash",
			command: "templ-dev",
			allowedCommands: map[string]bool{
				"templ-dev": true,
			},
			expectError: false,
		},
		{
			name:    "command with underscore",
			command: "templ_dev",
			allowedCommands: map[string]bool{
				"templ_dev": true,
			},
			expectError: false,
		},
		{
			name:    "command with number",
			command: "templ2",
			allowedCommands: map[string]bool{
				"templ2": true,
			},
			expectError: false,
		},

		// Nil and empty allowlist edge cases
		{
			name:            "nil allowlist",
			command:         "templ",
			allowedCommands: nil,
			expectError:     true,
			errorType:       "not allowed",
		},
		{
			name:            "empty allowlist",
			command:         "templ",
			allowedCommands: map[string]bool{},
			expectError:     true,
			errorType:       "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.command, tt.allowedCommands)

			if tt.expectError {
				require.Error(t, err, "Expected error for command '%s'", tt.command)
				if tt.errorType != "" {
					assert.Contains(
						t,
						strings.ToLower(err.Error()),
						tt.errorType,
						"Error should contain expected type: %s, got: %s",
						tt.errorType,
						err.Error(),
					)
				}
			} else {
				assert.NoError(t, err, "Expected no error for command '%s', got: %v", tt.command, err)
			}
		})
	}
}

// TestValidateArguments_EdgeCases tests edge cases for multiple argument validation
func TestValidateArguments_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorType   string
	}{
		// Nil slice edge cases
		{
			name:        "nil arguments slice",
			args:        nil,
			expectError: false,
		},

		// Large argument lists
		{
			name:        "many valid arguments",
			args:        make([]string, 1000),
			expectError: false,
		},
		{
			name:        "many arguments with one invalid",
			args:        append(make([]string, 999), "invalid;"),
			expectError: true,
			errorType:   "dangerous character",
		},

		// Mixed valid and invalid
		{
			name:        "alternating valid invalid",
			args:        []string{"valid1", "invalid;", "valid2", "invalid|"},
			expectError: true,
			errorType:   "dangerous character", // Should catch first invalid
		},

		// Edge case arguments
		{
			name:        "arguments with unicode",
			args:        []string{"файл.templ"}, // Russian filename
			expectError: false,
		},
		{
			name:        "arguments with emoji",
			args:        []string{"🚀component.templ"},
			expectError: false,
		},

		// Performance edge cases
		{
			name:        "very long single argument",
			args:        []string{strings.Repeat("a", 10000) + ".templ"},
			expectError: false,
		},
		{
			name: "many small arguments",
			args: func() []string {
				args := make([]string, 10000)
				for i := range args {
					args[i] = "a.templ"
				}
				return args
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize slice with valid values if needed
			for i := range tt.args {
				if tt.args[i] == "" {
					tt.args[i] = "valid.templ"
				}
			}

			err := validateArguments(tt.args)

			if tt.expectError {
				require.Error(t, err, "Expected error for arguments")
				if tt.errorType != "" {
					assert.Contains(
						t,
						strings.ToLower(err.Error()),
						tt.errorType,
						"Error should contain expected type: %s, got: %s",
						tt.errorType,
						err.Error(),
					)
				}
			} else {
				assert.NoError(t, err, "Expected no error for arguments, got: %v", err)
			}
		})
	}
}

// TestUnicodeSecurityEdgeCases tests specific Unicode security edge cases
func TestUnicodeSecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		expectError bool
		description string
	}{
		{
			name:        "unicode normalization attack",
			arg:         "file\u0041\u0300.templ", // A + combining grave accent
			expectError: false,
			description: "Should handle Unicode normalization forms",
		},
		{
			name:        "mixed scripts",
			arg:         "fileНаме.templ", // mix of latin and cyrillic
			expectError: false,
			description: "Mixed scripts should be allowed",
		},
		{
			name:        "bidi override attack",
			arg:         "file\u202e/cte/moc\u202d.templ",
			expectError: false,
			description: "Bidirectional text override characters",
		},
		{
			name:        "invisible characters",
			arg:         "file\u2060\u180e.templ", // word joiner + mongolian vowel separator
			expectError: false,
			description: "Invisible Unicode characters",
		},
		{
			name:        "confusable characters",
			arg:         "fіΙе.templ", // i + Greek Iota + Cyrillic ie
			expectError: false,
			description: "Visually confusable characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s - %s", tt.description, tt.arg)

			// Show character breakdown for debugging
			for i, r := range tt.arg {
				t.Logf("  [%d] U+%04X (%c)", i, r, r)
			}

			err := validateArgument(tt.arg)

			if tt.expectError {
				assert.Error(t, err, "Expected error for Unicode edge case: %s", tt.name)
			} else {
				assert.NoError(t, err, "Expected no error for Unicode edge case: %s, got: %v", tt.name, err)
			}
		})
	}
}

// BenchmarkValidation_EdgeCases benchmarks validation performance with edge cases
func BenchmarkValidation_EdgeCases(b *testing.B) {
	b.Run("very_long_argument", func(b *testing.B) {
		arg := strings.Repeat("a", 10000) + ".templ"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validateArgument(arg)
		}
	})

	b.Run("unicode_argument", func(b *testing.B) {
		arg := "файл🚀НаМе.templ"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validateArgument(arg)
		}
	})

	b.Run("many_arguments", func(b *testing.B) {
		args := make([]string, 1000)
		for i := range args {
			args[i] = "component.templ"
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validateArguments(args)
		}
	})

	b.Run("command_validation", func(b *testing.B) {
		allowedCommands := map[string]bool{
			"templ": true,
			"go":    true,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validateCommand("templ", allowedCommands)
		}
	})
}
