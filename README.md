# templar
rapid prototyping for templ components

## Development

### Running Tests

This project includes a comprehensive test suite covering unit tests, integration tests, and WebSocket functionality.

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run tests with race detection
make test-race

# Run comprehensive tests (coverage + race detection)
make test-full
```

### Test Coverage

The test suite includes:
- **Config package tests** - Configuration loading and validation
- **Registry package tests** - Component registration and event handling
- **Watcher package tests** - File system monitoring and change detection
- **Server package tests** - HTTP server, middleware, and routing
- **WebSocket tests** - Real-time communication and connection management
- **Integration tests** - Full system testing with file watching and WebSocket connections

### Building and Running

```bash
# Build the project
make build

# Run the project
make run

# Install dependencies
make deps

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint
```

### CLI Commands

Templar provides several commands for different development workflows:

```bash
# Initialize a new project
templar init [name]                     # Initialize in current directory or create new
templar init --minimal                  # Minimal setup without examples
templar init --example                  # Include example components
templar init --template blog            # Use specific template

# Development server
templar serve                           # Start development server
templar serve --port 3000               # Use different port
templar serve --no-open                 # Don't open browser

# Component management
templar list                            # List all components
templar list --format json             # Output as JSON
templar list --with-props               # Include component properties

# Preview components
templar preview Button                  # Preview Button component
templar preview Card --props '{"title":"Test"}' # Preview with props
templar preview Card --mock ./mocks/card.json   # Preview with mock data

# Build and watch
templar build                           # Build all components
templar build --production              # Production build
templar build --analyze                 # Generate build analysis
templar watch                           # Watch for changes and rebuild
```

### Development Workflow

```bash
# Set up development environment
make dev-setup

# Run pre-commit checks
make pre-commit

# Full CI checks
make ci

# CLI shortcuts
make init          # Initialize project
make serve         # Start development server
make list          # List components
make build-components # Build all components
make watch         # Watch for changes
make preview COMPONENT=Button # Preview specific component
```
