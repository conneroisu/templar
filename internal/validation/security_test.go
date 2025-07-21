package validation

import (
	"testing"
)

func TestValidateArgument(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		{
			name:    "valid argument",
			arg:     "generate",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			arg:     "./components",
			wantErr: false,
		},
		{
			name:    "command injection semicolon",
			arg:     "generate; rm -rf /",
			wantErr: true,
		},
		{
			name:    "command injection pipe",
			arg:     "generate | cat /etc/passwd",
			wantErr: true,
		},
		{
			name:    "command injection backtick",
			arg:     "generate`whoami`",
			wantErr: true,
		},
		{
			name:    "path traversal",
			arg:     "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "absolute path not allowed",
			arg:     "/home/user/file",
			wantErr: true,
		},
		{
			name:    "allowed system binary path",
			arg:     "/usr/bin/templ",
			wantErr: false,
		},
		{
			name:    "dangerous shell characters",
			arg:     "file$(whoami).txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArgument(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateArgument() error = %v, wantErr %v", err, tt.wantErr)
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
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowed command templ",
			command: "templ",
			wantErr: false,
		},
		{
			name:    "allowed command go",
			command: "go",
			wantErr: false,
		},
		{
			name:    "disallowed command",
			command: "rm",
			wantErr: true,
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
		},
		{
			name:    "command with injection",
			command: "templ; rm -rf /",
			wantErr: true,
		},
		{
			name:    "command with dangerous chars",
			command: "templ`whoami`",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command, allowedCommands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid relative path",
			path:    "./components/button.templ",
			wantErr: false,
		},
		{
			name:    "valid filename",
			path:    "component.templ",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "path traversal with dots",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "access to /etc/passwd",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "access to /proc",
			path:    "/proc/version",
			wantErr: true,
		},
		{
			name:    "access to /sys",
			path:    "/sys/kernel",
			wantErr: true,
		},
		{
			name:    "path with dangerous characters",
			path:    "file; rm -rf /",
			wantErr: true,
		},
		{
			name:    "path with command substitution",
			path:    "file$(whoami).txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOrigin(t *testing.T) {
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"https://example.com",
	}

	tests := []struct {
		name    string
		origin  string
		wantErr bool
	}{
		{
			name:    "allowed localhost origin",
			origin:  "http://localhost:3000",
			wantErr: false,
		},
		{
			name:    "allowed 127.0.0.1 origin",
			origin:  "http://127.0.0.1:3000",
			wantErr: false,
		},
		{
			name:    "allowed https origin",
			origin:  "https://example.com",
			wantErr: false,
		},
		{
			name:    "empty origin",
			origin:  "",
			wantErr: true,
		},
		{
			name:    "disallowed origin",
			origin:  "http://malicious.com",
			wantErr: true,
		},
		{
			name:    "javascript protocol",
			origin:  "javascript:alert('xss')",
			wantErr: true,
		},
		{
			name:    "file protocol",
			origin:  "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "malformed origin",
			origin:  "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrigin(tt.origin, allowedOrigins)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrigin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserAgent(t *testing.T) {
	blockedAgents := []string{
		"bot",
		"crawler",
		"scanner",
	}

	tests := []struct {
		name      string
		userAgent string
		wantErr   bool
	}{
		{
			name:      "normal browser user agent",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			wantErr:   false,
		},
		{
			name:      "empty user agent",
			userAgent: "",
			wantErr:   false,
		},
		{
			name:      "blocked bot user agent",
			userAgent: "GoogleBot/1.0",
			wantErr:   true,
		},
		{
			name:      "blocked crawler user agent",
			userAgent: "WebCrawler/1.0",
			wantErr:   true,
		},
		{
			name:      "blocked scanner user agent",
			userAgent: "VulnScanner/2.0",
			wantErr:   true,
		},
		{
			name:      "case insensitive blocking",
			userAgent: "BOTNET/1.0",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserAgent(tt.userAgent, blockedAgents)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileExtension(t *testing.T) {
	allowedExtensions := []string{".templ", ".go", ".html", ".css", ".js"}

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "allowed templ file",
			filename: "component.templ",
			wantErr:  false,
		},
		{
			name:     "allowed go file",
			filename: "main.go",
			wantErr:  false,
		},
		{
			name:     "case insensitive extension",
			filename: "style.CSS",
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "no extension",
			filename: "filename",
			wantErr:  true,
		},
		{
			name:     "disallowed extension",
			filename: "script.sh",
			wantErr:  true,
		},
		{
			name:     "dangerous executable",
			filename: "malware.exe",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileExtension(tt.filename, allowedExtensions)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileExtension() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "text with control characters",
			input:    "Hello\x01\x02World",
			expected: "HelloWorld",
		},
		{
			name:     "preserve allowed whitespace",
			input:    "Hello\t\n\rWorld",
			expected: "Hello\t\n\rWorld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "mixed dangerous characters",
			input:    "Hello\x00\x01\x02\tWorld\n",
			expected: "Hello\tWorld\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// Security-focused edge case tests
func TestSecurityEdgeCases(t *testing.T) {
	t.Run("Path traversal variations", func(t *testing.T) {
		// Test various path traversal techniques
		dangerousPaths := []string{
			"..\\..\\..\\etc\\passwd",
			"....//....//etc//passwd",
		}

		for _, path := range dangerousPaths {
			err := ValidatePath(path)
			if err == nil {
				t.Errorf("ValidatePath should reject path traversal: %s", path)
			}
		}
	})

	t.Run("Command injection variations", func(t *testing.T) {
		// Test various command injection techniques
		dangerousArgs := []string{
			"generate&whoami",
			"generate|cat /etc/passwd",
			"generate$(id)",
			"generate`id`",
			"generate;ls -la",
		}

		for _, arg := range dangerousArgs {
			err := ValidateArgument(arg)
			if err == nil {
				t.Errorf("ValidateArgument should reject command injection: %s", arg)
			}
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkValidateArgument(b *testing.B) {
	arg := "generate"
	for i := 0; i < b.N; i++ {
		ValidateArgument(arg)
	}
}

func BenchmarkValidatePath(b *testing.B) {
	path := "./components/button.templ"
	for i := 0; i < b.N; i++ {
		ValidatePath(path)
	}
}

func BenchmarkSanitizeInput(b *testing.B) {
	input := "Hello World with some\x00null\x01bytes"
	for i := 0; i < b.N; i++ {
		SanitizeInput(input)
	}
}
