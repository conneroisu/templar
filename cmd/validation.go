package cmd

import (
	"fmt"
	"strings"
)

// validateArgument validates individual command arguments for security
// This is a shared function used by both build.go and watch.go.
func validateArgument(arg string) error {
	// Reject arguments containing shell metacharacters
	dangerousChars := []string{
		";",
		"&",
		"|",
		"$",
		"`",
		"(",
		")",
		"{",
		"}",
		"[",
		"]",
		"<",
		">",
		"\"",
		"'",
		"\\",
	}
	for _, char := range dangerousChars {
		if strings.Contains(arg, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// Reject path traversal attempts
	if strings.Contains(arg, "..") {
		return errors.New("path traversal attempt detected")
	}

	// Additional validation for common patterns
	if strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, "/tmp/") &&
		!strings.HasPrefix(arg, "/usr/") {
		return fmt.Errorf("absolute path not allowed: %s", arg)
	}

	return nil
}

// validateCommand validates command names against an allowlist.
func validateCommand(command string, allowedCommands map[string]bool) error {
	if !allowedCommands[command] {
		return fmt.Errorf("command '%s' is not allowed", command)
	}

	return nil
}

// validateArguments validates a slice of arguments.
func validateArguments(args []string) error {
	for _, arg := range args {
		if err := validateArgument(arg); err != nil {
			return fmt.Errorf("invalid argument '%s': %w", arg, err)
		}
	}

	return nil
}

// validateBuildCommand validates the command and arguments to prevent command injection.
func validateBuildCommand(command string, args []string) error {
	// Allowlist of permitted commands
	allowedCommands := map[string]bool{
		"templ": true,
		"go":    true,
	}

	// Check if command is in allowlist
	if err := validateCommand(command, allowedCommands); err != nil {
		return fmt.Errorf("build command validation failed: %w", err)
	}

	// Validate arguments - prevent shell metacharacters and path traversal
	if err := validateArguments(args); err != nil {
		return fmt.Errorf("argument validation failed: %w", err)
	}

	return nil
}
