// Package validation provides security validation functions for preventing
// command injection, path traversal, and other security vulnerabilities.
package validation

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateArgument validates a command line argument to prevent injection attacks
func ValidateArgument(arg string) error {
	// Check argument length to prevent DoS attacks
	const maxArgLength = 4096 // Reasonable limit for command arguments
	if len(arg) > maxArgLength {
		return fmt.Errorf("argument too long: %d bytes (max %d)", len(arg), maxArgLength)
	}

	// Check for Unicode security issues
	if err := ValidateUnicodeString(arg); err != nil {
		return fmt.Errorf("unicode validation failed: %w", err)
	}

	// Check for shell metacharacters that could be used for command injection
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\\", "\"", "'", "~", "%"}
	for _, char := range dangerous {
		if strings.Contains(arg, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(arg, "..") {
		return fmt.Errorf("contains path traversal: %s", arg)
	}

	// Check for absolute paths (prefer relative paths for security)
	if filepath.IsAbs(arg) && !strings.HasPrefix(arg, "/usr/bin/") && !strings.HasPrefix(arg, "/bin/") {
		return fmt.Errorf("absolute path not allowed: %s", arg)
	}

	return nil
}

// ValidateCommand validates a command name against an allowlist
func ValidateCommand(command string, allowedCommands map[string]bool) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check if command is in allowlist
	if !allowedCommands[command] {
		return fmt.Errorf("command '%s' is not allowed", command)
	}

	// Additional security checks for the command itself
	if err := ValidateArgument(command); err != nil {
		return fmt.Errorf("invalid command '%s': %w", command, err)
	}

	return nil
}

// ValidatePath validates a file path to prevent path traversal attacks
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Prevent access to sensitive system directories
	restrictedPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/proc/",
		"/sys/",
		"/dev/",
		"/root/",
		"/boot/",
	}

	cleanPathLower := strings.ToLower(cleanPath)
	for _, restricted := range restrictedPaths {
		if strings.HasPrefix(cleanPathLower, restricted) {
			return fmt.Errorf("access to restricted path denied: %s", path)
		}
	}

	// Additional checks for dangerous characters and patterns in paths
	dangerousChars := []string{";", "&", "|", "$", "`", "<", ">", "~", "%"}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	return nil
}

// ValidateOrigin validates WebSocket origin for CSRF protection
func ValidateOrigin(origin string, allowedOrigins []string) error {
	if origin == "" {
		return fmt.Errorf("origin header is required")
	}

	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin format: %w", err)
	}

	// Only allow http/https schemes
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return fmt.Errorf("invalid origin scheme '%s': only http and https are allowed", originURL.Scheme)
	}

	// Check against allowed origins list
	for _, allowed := range allowedOrigins {
		if origin == allowed || originURL.Host == allowed {
			return nil
		}
	}

	return fmt.Errorf("origin '%s' is not in allowed origins list", origin)
}

// ValidateUserAgent validates user agent strings against a blocklist
func ValidateUserAgent(userAgent string, blockedAgents []string) error {
	if userAgent == "" {
		// Empty user agent is allowed
		return nil
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, blocked := range blockedAgents {
		if strings.Contains(userAgentLower, strings.ToLower(blocked)) {
			return fmt.Errorf("user agent '%s' is blocked", userAgent)
		}
	}

	return nil
}

// ValidateFileExtension validates file extensions against an allowlist
func ValidateFileExtension(filename string, allowedExtensions []string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fmt.Errorf("file must have an extension")
	}

	for _, allowed := range allowedExtensions {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}

	return fmt.Errorf("file extension '%s' is not allowed", ext)
}

// SanitizeInput removes or escapes potentially dangerous characters from user input
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except common whitespace
	var sanitized strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}

// ValidateUnicodeString validates Unicode strings against various security attacks
func ValidateUnicodeString(s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8 sequence")
	}

	for i, r := range s {
		// Check for bidirectional text manipulation characters
		if isBidiOverride(r) {
			return fmt.Errorf("contains bidirectional override character at position %d", i)
		}

		// Check for zero-width characters that could hide content
		if isZeroWidth(r) {
			return fmt.Errorf("contains zero-width character at position %d", i)
		}

		// Check for homoglyph attacks (non-ASCII that looks similar to ASCII)
		if isHomoglyphRisk(r) {
			return fmt.Errorf("contains potentially confusing non-ASCII character at position %d", i)
		}

		// Check for control characters - for command arguments, we should be strict
		if unicode.IsControl(r) {
			return fmt.Errorf("contains control character at position %d", i)
		}

		// Check for replacement character (indicates encoding issues)
		if r == '\uFFFD' {
			return fmt.Errorf("contains Unicode replacement character at position %d", i)
		}
	}

	return nil
}

// isBidiOverride checks for bidirectional text override characters
func isBidiOverride(r rune) bool {
	switch r {
	case '\u202D', // Left-to-Right Override (LRO)
		'\u202E', // Right-to-Left Override (RLO)
		'\u202C', // Pop Directional Formatting (PDF)
		'\u2066', // Left-to-Right Isolate (LRI)
		'\u2067', // Right-to-Left Isolate (RLI)
		'\u2068', // First Strong Isolate (FSI)
		'\u2069': // Pop Directional Isolate (PDI)
		return true
	}
	return false
}

// isZeroWidth checks for zero-width characters that could hide content
func isZeroWidth(r rune) bool {
	switch r {
	case '\u200B', // Zero Width Space (ZWSP)
		'\u200C', // Zero Width Non-Joiner (ZWNJ)
		'\u200D', // Zero Width Joiner (ZWJ)
		'\u2060', // Word Joiner (WJ)
		'\u2061', // Function Application
		'\u2062', // Invisible Times
		'\u2063', // Invisible Separator
		'\u2064', // Invisible Plus
		'\uFEFF': // Zero Width No-Break Space (BOM)
		return true
	}
	return false
}

// isHomoglyphRisk checks for characters that might be homoglyph attacks
func isHomoglyphRisk(r rune) bool {
	// For security, we restrict to ASCII printable characters for command arguments
	// This prevents homoglyph attacks where visually similar characters from
	// different scripts could be used to deceive users or bypass filters
	if r > 127 {
		return true // Any non-ASCII character is considered risky
	}
	return false
}
