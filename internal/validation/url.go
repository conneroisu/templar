package validation

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURL validates URLs for browser auto-open functionality
// Prevents command injection via URL parameters
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

	// Check for dangerous characters that could enable command injection
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\"", "'", "\\", "\n", "\r"}
	for _, char := range dangerous {
		if strings.Contains(rawURL, char) {
			return fmt.Errorf("URL contains dangerous character: %s", char)
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

	return nil
}