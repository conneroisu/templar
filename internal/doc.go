// Package internal contains the core implementation packages for templar.
//
// This package follows Go's internal package convention, making these
// packages unavailable for import by external modules while providing
// all the core functionality for the templar CLI tool.
//
// # Package Organization
//
// The internal packages are organized by functional domain:
//
//   - build: Build pipeline with worker pools, caching, and metrics
//   - config: Configuration management with validation and security
//   - errors: Error collection, parsing, and HTML overlay generation
//   - registry: Component registry and event broadcasting system
//   - renderer: Component rendering and template processing
//   - scanner: File system scanning and metadata extraction
//   - server: HTTP server, WebSocket support, and middleware
//   - watcher: File system monitoring with debouncing
//
// # Design Principles
//
// All internal packages follow these design principles:
//
//   - Security by default with input validation and sanitization
//   - Concurrent safety with proper mutex usage and race protection
//   - Performance optimization with caching and efficient algorithms
//   - Testability with comprehensive unit and integration test coverage
//   - Observability with structured logging and metrics collection
//
// # Inter-Package Communication
//
// Packages communicate through well-defined interfaces:
//
//   - Registry acts as the central event hub for component changes
//   - Build pipeline consumes registry events and produces build results
//   - Server coordinates between all components and handles user requests
//   - Watcher monitors file system and triggers registry updates
//   - Scanner processes files and populates the registry
//
// # Security Considerations
//
// Security is implemented at multiple layers:
//
//   - Config package validates all configuration inputs
//   - Server package implements origin validation and CSRF protection
//   - Build package prevents command injection with strict allowlisting
//   - Scanner package validates file paths and prevents traversal attacks
//   - All packages sanitize user inputs and log security events
//
// # Performance Optimizations
//
// Key performance optimizations include:
//
//   - LRU caching in build pipeline for O(1) cache operations
//   - Metadata-based file hash caching to reduce I/O operations
//   - Concurrent worker pools for parallel processing
//   - Debounced file watching to prevent excessive rebuilds
//   - Efficient WebSocket broadcasting for real-time updates
//
// # Testing Strategy
//
// Each package includes comprehensive test coverage:
//
//   - Unit tests for individual functions and methods
//   - Integration tests for cross-package interactions
//   - Security tests for all hardening measures
//   - Performance benchmarks for critical code paths
//   - Race condition tests with Go's race detector
//
// For detailed documentation, see the individual package documentation.
package internal
