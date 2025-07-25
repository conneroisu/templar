// Package cmd provides the command-line interface for templar.
//
// This package implements all CLI commands using the Cobra framework,
// providing a comprehensive set of tools for templ component development.
//
// # Available Commands
//
//   - init: Initialize a new templar project with optional templates
//   - serve: Start the development server with hot reload
//   - list: List all discovered components with metadata
//   - preview: Preview specific components with mock data
//   - watch: Watch for file changes and trigger rebuilds
//   - build: Build all components for production
//   - health: Check system health and dependencies
//
// # Command Examples
//
//	// Initialize a new project
//	templar init --template blog
//
//	// Start development server
//	templar serve --port 3000 --no-open
//
//	// List components with JSON output
//	templar list --format json --with-props
//
//	// Preview component with props
//	templar preview Card --props '{"title":"Test"}'
//
//	// Watch and rebuild on changes
//	templar watch --ignore "node_modules/**"
//
//	// Production build
//	templar build --production
//
//	// Health check
//	templar health --verbose
//
// # Security Considerations
//
// All commands implement security hardening:
//
//   - Input validation for all parameters
//   - Path traversal protection for file operations
//   - Command injection prevention in build operations
//   - Sanitization of user-provided component names
//
// # Configuration Integration
//
// Commands respect configuration from multiple sources in order of precedence:
//
//  1. Command-line flags (highest priority)
//  2. Environment variables (TEMPLAR_*)
//  3. Configuration file (.templar.yml)
//  4. Default values (lowest priority)
//
// # Error Handling
//
// All commands provide structured error reporting with:
//
//   - Clear error messages for common issues
//   - Detailed logging in debug mode
//   - Exit codes following Unix conventions
//   - Graceful handling of interrupts (Ctrl+C)
//
// For detailed usage of individual commands, see their respective documentation.
package cmd
