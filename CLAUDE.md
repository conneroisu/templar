# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Templar is a rapid prototyping CLI tool for Go templ that provides browser preview functionality, hot reload capability, and streamlined development workflows. It's built as a Cobra-based CLI application with a comprehensive web server, component scanner, file watcher, and WebSocket-based live updates.

## Architecture Overview

### Core Components

- **CLI Commands (`cmd/`)**: Cobra-based commands (init, serve, list, build, watch, preview) that orchestrate the core functionality
- **Component Registry (`internal/registry/`)**: Central registry for component discovery, metadata management, and event broadcasting
- **Component Scanner (`internal/scanner/`)**: File system scanner that discovers and analyzes templ components, extracting metadata and dependencies
- **Build Pipeline (`internal/build/`)**: Multi-worker build system with LRU caching, goroutine lifecycle management, and error collection
- **Development Server (`internal/server/`)**: HTTP server with middleware, WebSocket support, and security-hardened origin validation
- **File Watcher (`internal/watcher/`)**: Real-time file system monitoring with debouncing and recursive directory watching
- **Configuration System (`internal/config/`)**: Viper-based configuration with validation and security checks

### Data Flow

1. **Component Discovery**: Scanner traverses directories finding `.templ` files, extracts metadata (parameters, dependencies)
2. **Registry Management**: Components registered with change events broadcast to subscribers
3. **Development Server**: HTTP handlers serve preview pages, WebSocket connections provide real-time updates
4. **File Watching**: Changes trigger re-scanning, building, and WebSocket notifications for live reload
5. **Build Pipeline**: Components processed through worker pools with caching and error handling

### Security Architecture

The codebase implements defense-in-depth security:
- **Command injection prevention** with strict allowlisting in build operations
- **Path traversal protection** with validation and current directory enforcement
- **WebSocket origin validation** with scheme/host checking and CSRF protection
- **Input validation** across all user-facing interfaces
- **Race condition protection** with proper mutex usage and goroutine lifecycle management

## Development Environment

### Nix Flake Development

The project uses Nix flakes for reproducible development environments. Enter the development shell:

```bash
# Enter development environment
nix develop

# Quick file editing shortcuts
dx  # Edit flake.nix
gx  # Edit go.mod
```

The Nix environment provides Go 1.24, development tools (air, golangci-lint, gopls), and hot reloading capabilities.

## Common Commands

### Essential Development Commands

```bash
# Basic development workflow
make dev-setup          # Set up development environment  
make serve               # Start development server (go run main.go serve)
make build               # Build the project
make test                # Run all tests
make fmt                 # Format code
make lint                # Run linter

# CLI command shortcuts  
make init                # Initialize project (go run main.go init)
make list                # List components (go run main.go list)
make watch               # Watch for changes (go run main.go watch)
make preview COMPONENT=Button  # Preview specific component
```

### Testing Commands

```bash
# Test categories
make test-unit           # Unit tests only (-short flag)
make test-integration    # Integration tests with file watching and WebSocket
make test-security       # Security tests for all packages with hardening
make test-e2e            # End-to-end tests (45m timeout)
make test-bench          # Performance benchmarks for all components

# Coverage and quality
make test-coverage       # Generate HTML coverage report
make test-race           # Race detection
make test-full           # Coverage + race detection
make test-ci             # Comprehensive CI-like test suite

# Security
make security-scan       # Vulnerability scanning with govulncheck
```

### Build and Docker Commands

```bash
# Building
make build-prod          # Static production binary
make generate            # Run go generate for templ files

# Docker
make docker-build        # Build Docker image
make docker-run          # Run container on port 8080
make docker-test         # Run tests in Docker environment
```

### CLI Usage Patterns

```bash
# Project initialization
templar init                     # Initialize in current directory
templar init --minimal           # Minimal setup without examples
templar init --template blog     # Use specific template

# Development server
templar serve                    # Start on default port (8080)
templar serve --port 3000        # Custom port
templar serve --no-open          # Don't auto-open browser

# Component management
templar list                     # List all components
templar list --format json      # JSON output
templar list --with-props        # Include component properties

# Component preview
templar preview Button           # Preview Button component
templar preview Card --props '{"title":"Test"}'  # With props
templar preview Card --mock ./mocks/card.json    # With mock data

# Build and watch
templar build                    # Build all components
templar build --production       # Production build
templar watch                    # Watch for changes and rebuild
```

## Configuration System

### Configuration Files

- **`.templar.yml`**: Main configuration file (YAML format)
- **Environment variables**: Prefixed with `TEMPLAR_`
- **Command-line flags**: Override configuration values

### Key Configuration Sections

```yaml
server:
  port: 8080
  host: "localhost"
  open: true                    # Auto-open browser
  middleware: ["cors", "logging"]

components:
  scan_paths: ["./components", "./views", "./examples"]
  exclude_patterns: ["*_test.templ", "*.bak"]

build:
  command: "templ generate"
  watch: ["**/*.templ"]
  ignore: ["node_modules", ".git"]
  cache_dir: ".templar/cache"

development:
  hot_reload: true
  css_injection: true
  error_overlay: true

preview:
  mock_data: "auto"
  wrapper: "layout.templ"
  auto_props: true
```

## Testing Architecture

### Test Organization

- **Unit tests**: Component-level testing with mocks and table-driven tests
- **Integration tests**: Cross-component testing with real file system and WebSocket connections
- **Security tests**: Comprehensive security hardening validation
- **Performance benchmarks**: Memory usage, concurrency, and throughput testing
- **E2E tests**: Full workflow testing with temporary directories and live servers

### Security Test Coverage

Security tests validate:
- Command injection prevention in build operations
- Path traversal protection in file handlers
- WebSocket origin validation and CSRF protection
- Input validation across all interfaces
- Race condition prevention with proper synchronization
- Memory leak prevention with goroutine lifecycle management

## File Structure and Patterns

### Package Organization

```
cmd/                     # CLI commands (Cobra)
internal/
  build/                 # Build pipeline with worker pools and caching
  config/                # Configuration management with validation
  errors/                # Error collection and HTML overlay generation
  registry/              # Component registry and event system
  renderer/              # Component rendering and template processing
  scanner/               # File system scanning and metadata extraction
  server/                # HTTP server, WebSocket, and security
  watcher/               # File system watching with debouncing
components/              # Example components
examples/                # Generated template examples
```

### Development Patterns

- **Cobra CLI structure**: Each command in separate file with validation
- **Event-driven architecture**: Registry broadcasts changes, components subscribe
- **Worker pool pattern**: Build pipeline uses configurable worker pools
- **LRU caching**: O(1) cache eviction with doubly-linked lists
- **Security-first design**: Input validation, allowlisting, and origin checking
- **Table-driven tests**: Comprehensive test coverage with data-driven test cases

## CI/CD Pipeline

### GitHub Actions Workflows

- **9-phase CI pipeline**: Code quality, security, unit tests, performance, integration, build, E2E, security scanning, deployment readiness
- **Multi-platform testing**: Linux, Windows, macOS with Go 1.23 and 1.24
- **Performance regression detection**: Automated benchmark comparison
- **Security scanning**: Vulnerability detection with automated alerts
- **Docker integration**: Multi-stage builds with health checks

### Pre-commit Workflow

```bash
make pre-commit          # Format, lint, race detection, security tests
make ci                  # Full CI workflow locally
```

## WebSocket and Real-time Features

### WebSocket Security

- **Strict origin validation**: Only allowed origins (localhost:3000, 127.0.0.1:3000, server port)
- **Scheme validation**: HTTP/HTTPS only, rejects javascript:, file:, data: protocols
- **Connection lifecycle management**: Proper cleanup and goroutine management
- **Message size limits**: Protection against large message attacks

### Live Reload Architecture

1. File watcher detects changes in component files
2. Scanner re-analyzes changed components
3. Build pipeline processes updates with caching
4. WebSocket broadcasts change notifications
5. Browser receives updates and refreshes affected components

## Error Handling and Debugging

### Error Collection System

- **Structured error collection**: Component, file, line, column, severity
- **HTML error overlay**: Development-friendly error display
- **Build error parsing**: Integration with templ compiler error output
- **Race-safe error collection**: Mutex-protected error aggregation

### Debugging Tools

```bash
# Verbose testing and debugging
make test-verbose        # Detailed test output
go test -v ./internal/server -run TestWebSocket  # Specific test debugging
go test -race ./...      # Race condition detection
go test -bench=. -benchmem -cpuprofile=cpu.prof  # Performance profiling
```

The development environment includes pprof and graphviz for performance analysis and profiling.