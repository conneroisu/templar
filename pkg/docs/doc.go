// Package templar provides a rapid prototyping CLI tool for Go templ components.
//
// Templar is designed to streamline the development workflow for templ-based web applications
// by providing browser preview functionality, hot reload capabilities, and component management tools.
//
// # Key Features
//
//   - Component Discovery: Automatic scanning and discovery of .templ files in your project
//   - Live Preview: Browser-based preview server with hot reload capabilities
//   - Build Pipeline: Efficient compilation with caching and concurrent processing
//   - Development Server: HTTP server with WebSocket support for real-time updates
//   - File Watching: Intelligent file system monitoring with debouncing
//   - Security: Command injection prevention, input validation, and origin checking
//
// # Quick Start
//
//	// Initialize a new templar project
//	templar init
//
//	// Start the development server
//	templar serve
//
//	// List available components
//	templar list
//
//	// Preview a specific component
//	templar preview MyComponent
//
//	// Watch for changes and rebuild
//	templar watch
//
// # Architecture
//
// The templar package is organized into several core components:
//
//   - CLI Commands (cmd/): Cobra-based command interface
//   - Component Registry (internal/registry/): Central component management
//   - Build Pipeline (internal/build/): Multi-worker build system with caching
//   - Development Server (internal/server/): HTTP server with WebSocket support
//   - File Watcher (internal/watcher/): Real-time file system monitoring
//   - Configuration (internal/config/): Viper-based configuration management
//
// # Security
//
// Templar implements defense-in-depth security measures:
//
//   - Command injection prevention with strict allowlisting
//   - Path traversal protection with validation
//   - WebSocket origin validation and CSRF protection
//   - Input validation across all user interfaces
//   - Race condition protection with proper synchronization
//
// # Configuration
//
// Templar supports configuration through multiple sources:
//
//   - Configuration file (.templar.yml)
//   - Environment variables (TEMPLAR_*)
//   - Command-line flags
//
// Example configuration:
//
//	server:
//	  port: 8080
//	  host: localhost
//	  environment: development
//	  allowed_origins:
//	    - "https://app.example.com"
//	    - "https://dashboard.example.com"
//
//	components:
//	  scan_paths:
//	    - "./components"
//	    - "./views"
//	  exclude_patterns:
//	    - "*_test.templ"
//
//	build:
//	  command: "templ generate"
//	  cache_dir: ".templar/cache"
//
//	development:
//	  hot_reload: true
//	  error_overlay: true
//
// # Performance
//
// Templar is optimized for performance with:
//
//   - LRU caching with O(1) operations for build results
//   - Concurrent worker pools for parallel processing
//   - Metadata-based file hash caching to reduce I/O
//   - Efficient WebSocket broadcasting for live updates
//   - Debounced file watching to prevent excessive rebuilds
//
// # Testing
//
// The package includes comprehensive test coverage:
//
//   - Unit tests for individual components
//   - Integration tests for cross-component functionality
//   - Security tests for all hardening measures
//   - Performance benchmarks for critical paths
//   - End-to-end tests for complete workflows
//
// For more information, see the individual package documentation.
package docs
