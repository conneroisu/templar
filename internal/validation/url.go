package validation

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURL validates URLs for browser auto-open functionality.
// This function prevents command injection attacks by strictly validating
// URL structure, schemes, and content before passing to system commands.
//
// Security features:
//   - Only allows http/https schemes to prevent protocol handler abuse
//   - Blocks shell metacharacters that could enable command injection
//   - Validates URL structure and hostname presence
//   - Rejects URLs with spaces or dangerous characters
//
// Returns an error if the URL is invalid or potentially dangerous.
func ValidateURL(rawURL string) error {
	// Parse and validate URL structure
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http/https schemes to prevent protocol handlers
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http/https allowed)", parsed.Scheme)
	}

	// Check for shell metacharacters that could enable command injection
	// Use more efficient single-pass validation for performance
	shellMetachars := ";|`$()<>\"'\\\n\r&"

	for _, char := range rawURL {
		if strings.ContainsRune(shellMetachars, char) {
			return fmt.Errorf("URL contains shell metacharacter %q (potential command injection)", char)
		}
	}

	// Additional safety: reject URLs with spaces (could indicate injection attempts)
	if strings.Contains(rawURL, " ") {
		return fmt.Errorf("URL contains spaces (possible command injection attempt)")
	}

	// Validate hostname isn't empty
	if parsed.Host == "" {
		return fmt.Errorf("URL must have a valid hostname")
	}

	// Check for path traversal patterns that could bypass browser security
	if strings.Contains(parsed.Path, "..") {
		return fmt.Errorf("URL contains path traversal sequence '..' (potential directory traversal)")
	}

	// Check for encoded path traversal attempts
	pathTraversalPatterns := []string{
		"%2e%2e",     // URL-encoded ".."
		"%2E%2E",     // URL-encoded ".." (uppercase)
		"%252e%252e", // Double-encoded ".."
		"%252E%252E", // Double-encoded ".." (uppercase)
		"....//",     // Variant traversal pattern
		"..%2f",      // Mixed encoding
		"..%2F",      // Mixed encoding (uppercase)
		"%c0%af",     // UTF-8 overlong encoding
	}

	for _, pattern := range pathTraversalPatterns {
		if strings.Contains(strings.ToLower(rawURL), pattern) {
			return fmt.Errorf("URL contains encoded path traversal pattern %q", pattern)
		}
	}

	return nil
}
