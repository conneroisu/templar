# Developer Guide

This guide is for developers who want to contribute to Templar or understand its internal architecture.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Development Setup](#development-setup)
- [Code Organization](#code-organization)
- [Core Components](#core-components)
- [Security Architecture](#security-architecture)
- [Testing Strategy](#testing-strategy)
- [Performance Optimization](#performance-optimization)
- [Contributing Guidelines](#contributing-guidelines)

## Architecture Overview

Templar is built as a modular CLI application with the following high-level architecture:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Commands  │    │   Web Server    │    │   File Watcher  │
│   (Cobra)       │────│   (HTTP/WS)     │────│   (fsnotify)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         └──────────────│  Component      │──────────────┘
                        │  Registry       │
                        │  (Event Bus)    │
                        └─────────────────┘
                                 │
                ┌────────────────┼────────────────┐
                │                │                │
         ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
         │  Scanner    │  │  Builder    │  │  Renderer   │
         │  (Discovery)│  │  (Compile)  │  │  (Preview)  │
         └─────────────┘  └─────────────┘  └─────────────┘
```

### Key Design Principles

1. **Event-Driven Architecture**: Components communicate through events via the central registry
2. **Security-First**: Input validation, path traversal protection, and command injection prevention
3. **Performance Optimized**: Worker pools, LRU caching, and object pooling
4. **Modular Design**: Clear separation of concerns with well-defined interfaces
5. **Comprehensive Testing**: Unit, integration, security, and performance tests

## Development Setup

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **templ CLI** - `go install github.com/a-h/templ/cmd/templ@latest`
- **golangci-lint** - [Installation guide](https://golangci-lint.run/usage/install/)
- **Make** - For build automation

### Local Development

```bash
# Clone the repository
git clone https://github.com/conneroisu/templar.git
cd templar

# Install dependencies
go mod tidy

# Generate templ files
go generate ./...

# Build the project
make build

# Run tests
make test

# Start development with hot reload
make dev
```

### Nix Development Environment

For reproducible development environments:

```bash
# Enter Nix development shell
nix develop

# Available commands in Nix shell:
dx        # Edit flake.nix
gx        # Edit go.mod
build-go  # Build all Go packages
tests     # Run all tests
lint      # Run linting
format    # Format code
```

### IDE Setup

#### VS Code

Recommended extensions:
- **Go** - Rich Go language support
- **templ** - templ template language support
- **golangci-lint** - Linting integration
- **Test Explorer** - Visual test runner

#### Vim/Neovim

```lua
-- Add to your config
require('lspconfig').gopls.setup({
  settings = {
    gopls = {
      templateExtensions = {"templ"},
      analyses = {
        unusedparams = true,
      },
    },
  },
})
```

## Code Organization

### Directory Structure

```
templar/
├── cmd/                        # CLI commands (Cobra)
│   ├── init.go                # Project initialization
│   ├── serve.go               # Development server
│   ├── preview.go             # Component preview
│   ├── list.go                # Component listing
│   ├── build.go               # Build commands
│   ├── watch.go               # File watching
│   └── validation.go          # Security validation
├── internal/                   # Internal packages
│   ├── build/                 # Build pipeline
│   │   ├── pipeline.go        # Multi-worker build pipeline
│   │   ├── pools.go           # Object pooling for performance
│   │   ├── compiler.go        # templ compiler interface
│   │   └── cache.go           # LRU build cache
│   ├── config/                # Configuration management
│   │   ├── config.go          # Configuration loading/validation
│   │   ├── schema.go          # Configuration schema
│   │   └── defaults.go        # Default configuration values
│   ├── di/                    # Dependency injection
│   │   ├── container.go       # DI container implementation
│   │   ├── services.go        # Service definitions
│   │   └── lifecycle.go       # Service lifecycle management
│   ├── errors/                # Error handling
│   │   ├── errors.go          # Error collection and formatting
│   │   ├── suggestions.go     # Enhanced error messages
│   │   ├── parser.go          # Error parsing from compiler output
│   │   └── overlay.go         # HTML error overlay generation
│   ├── registry/              # Component registry
│   │   ├── registry.go        # Central component registry
│   │   ├── events.go          # Event system for component changes
│   │   ├── metadata.go        # Component metadata extraction
│   │   └── types.go           # Registry type definitions
│   ├── renderer/              # Component rendering
│   │   ├── renderer.go        # Component HTML rendering
│   │   ├── context.go         # Rendering context management
│   │   └── templates.go       # Template processing
│   ├── scanner/               # File system scanning
│   │   ├── scanner.go         # Component discovery and analysis
│   │   ├── parser.go          # templ file parsing
│   │   ├── metadata.go        # Metadata extraction from components
│   │   └── filters.go         # File filtering and exclusion
│   ├── server/                # HTTP server
│   │   ├── server.go          # HTTP server with middleware
│   │   ├── handlers.go        # Request handlers
│   │   ├── websocket.go       # WebSocket for live reload
│   │   ├── middleware.go      # HTTP middleware
│   │   └── security.go        # Security validation and CSRF protection
│   ├── testing/               # Testing framework
│   │   ├── error_injection.go # Error injection for testing
│   │   ├── resource_tracker.go # Resource leak detection
│   │   ├── scenarios.go       # Predefined test scenarios
│   │   └── helpers.go         # Test utilities
│   └── watcher/               # File system watching
│       ├── watcher.go         # File system event monitoring
│       ├── debouncer.go       # Event debouncing
│       └── patterns.go        # Watch pattern matching
├── examples/                   # Example components and projects
├── docs/                      # Documentation
├── scripts/                   # Build and deployment scripts
└── testdata/                  # Test data and fixtures
```

### Package Dependencies

```
cmd → internal/{config,server,registry,scanner,renderer,errors}
internal/server → internal/{registry,watcher,build,errors}
internal/build → internal/{registry,config,errors}
internal/scanner → internal/{registry,config}
internal/watcher → internal/{config,registry}
internal/registry → (no internal dependencies)
internal/config → (no internal dependencies)
internal/errors → internal/registry
```

## Core Components

### 1. Component Registry (`internal/registry/`)

The registry is the central hub that manages component metadata and events.

**Key Features:**
- Thread-safe component storage with RWMutex
- Event broadcasting for component changes
- Metadata caching and validation
- Subscription management for live updates

**Example Usage:**
```go
registry := registry.NewComponentRegistry()

// Register component
component := &registry.ComponentInfo{
    Name:     "Button",
    Package:  "components",
    FilePath: "button.templ",
}
registry.Register(component)

// Subscribe to changes
subscription := registry.Subscribe()
go func() {
    for event := range subscription {
        fmt.Printf("Component %s changed: %s\n", event.ComponentName, event.Type)
    }
}()
```

### 2. Build Pipeline (`internal/build/`)

Multi-worker build system with advanced optimizations.

**Key Features:**
- Configurable worker pools for parallel builds
- LRU caching with O(1) eviction
- Object pooling to reduce GC pressure
- Priority-based task scheduling
- Comprehensive error collection

**Architecture:**
```go
type BuildPipeline struct {
    queue       *BuildQueue
    workers     []*BuildWorker
    cache       *LRUCache
    objectPools *ObjectPools
    slicePools  *SlicePools
    workerPool  *WorkerPool
}
```

**Example Usage:**
```go
pipeline := build.NewBuildPipeline(4, registry) // 4 workers
pipeline.SetCompiler(compiler)
pipeline.Start(ctx)

// Queue build task
task := build.BuildTask{
    Component: component,
    Priority:  1,
}
pipeline.QueueBuild(task)
```

### 3. Development Server (`internal/server/`)

Production-ready HTTP server with WebSocket support.

**Key Features:**
- Security-hardened with origin validation
- WebSocket connections for live reload
- Middleware pipeline (CORS, logging, security)
- Graceful shutdown with context cancellation
- Static file serving with proper MIME types

**Security Measures:**
- Origin validation for WebSocket connections
- Path traversal protection
- Input validation and sanitization
- Rate limiting (configurable)
- CSRF protection for state-changing operations

**Example Usage:**
```go
server, err := server.New(config)
if err != nil {
    return err
}

// Start with graceful shutdown
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

err = server.Start(ctx)
```

### 4. File Watcher (`internal/watcher/`)

Real-time file system monitoring with debouncing.

**Key Features:**
- Recursive directory watching
- Event debouncing to prevent spam
- Pattern-based filtering (glob patterns)
- Cross-platform compatibility
- Resource-efficient with configurable limits

**Example Usage:**
```go
watcher, err := watcher.NewFileWatcher(time.Second) // 1s debounce
if err != nil {
    return err
}

watcher.AddPath("./components", "**/*.templ")
events := watcher.Events()

go func() {
    for event := range events {
        fmt.Printf("File changed: %s\n", event.Path)
    }
}()
```

### 5. Component Scanner (`internal/scanner/`)

Intelligent component discovery and metadata extraction.

**Key Features:**
- AST-based templ file parsing
- Parameter and dependency extraction
- Configurable scan paths and exclusions
- Performance optimized with worker pools
- Metadata caching for large projects

**Example Usage:**
```go
scanner := scanner.NewComponentScanner(registry)
err := scanner.ScanDirectory("./components")
if err != nil {
    return err
}

// Access discovered components
components := registry.GetComponents()
```

## Security Architecture

Templar implements defense-in-depth security with multiple layers:

### 1. Command Injection Prevention

**Location:** `cmd/validation.go`

```go
func validateCommand(command string) error {
    allowedCommands := []string{
        "templ", "generate", "go", "build", "run",
    }
    
    cmd := strings.Fields(command)[0]
    for _, allowed := range allowedCommands {
        if cmd == allowed {
            return nil
        }
    }
    return fmt.Errorf("command not allowed: %s", cmd)
}
```

### 2. Path Traversal Protection

**Location:** `cmd/validation.go`

```go
func validatePath(path string) error {
    cleanPath := filepath.Clean(path)
    
    // Reject path traversal attempts
    if strings.Contains(cleanPath, "..") {
        return fmt.Errorf("path traversal detected: %s", path)
    }
    
    // Only allow relative paths
    if filepath.IsAbs(cleanPath) {
        return fmt.Errorf("absolute paths not allowed: %s", path)
    }
    
    return nil
}
```

### 3. WebSocket Origin Validation

**Location:** `internal/server/websocket.go`

```go
func validateOrigin(origin string, allowedHosts []string) error {
    u, err := url.Parse(origin)
    if err != nil {
        return fmt.Errorf("invalid origin URL: %w", err)
    }
    
    // Check scheme
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("invalid origin scheme: %s", u.Scheme)
    }
    
    // Validate host
    for _, allowed := range allowedHosts {
        if u.Host == allowed {
            return nil
        }
    }
    
    return fmt.Errorf("origin not allowed: %s", origin)
}
```

### 4. Input Validation

All user inputs are validated:
- **JSON size limits** (1MB max for props)
- **File extension validation** (only .json for mocks)
- **URL validation** for redirects
- **Component name validation** (alphanumeric + underscore)

## Testing Strategy

Templar uses a comprehensive multi-layered testing approach:

### 1. Unit Tests

**Coverage:** Individual functions and methods
**Location:** `*_test.go` files alongside source code

```bash
# Run unit tests only
make test-unit

# With coverage
make test-coverage
```

### 2. Integration Tests

**Coverage:** Cross-component interactions
**Location:** `internal/server/integration_test.go`, etc.

```bash
# Run integration tests
make test-integration
```

### 3. Security Tests

**Coverage:** Security hardening validation
**Location:** `cmd/validation_test.go`, `internal/server/security_test.go`

```bash
# Run security tests
make test-security
```

### 4. Performance Benchmarks

**Coverage:** Critical path performance
**Location:** `*_bench_test.go` files

```bash
# Run benchmarks
make test-bench

# With memory profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof ./...
```

### 5. Error Injection Testing

**Coverage:** Failure resilience and error handling
**Location:** `internal/testing/`

```bash
# Run error injection tests
go test -tags=error_injection ./...
```

**Example Error Injection Test:**
```go
func TestBuildPipeline_ErrorRecovery(t *testing.T) {
    injector := testingpkg.NewErrorInjector()
    tracker := testingpkg.NewResourceTracker("test")
    defer tracker.CheckLeaks(t)
    
    // Inject file permission errors for first 2 attempts
    injector.InjectErrorCount("file.read", testingpkg.ErrPermissionDenied, 2)
    
    pipeline := build.NewBuildPipeline(1, registry)
    pipeline.SetCompiler(&MockCompilerWithInjection{injector})
    
    // Test that pipeline recovers after errors
    // ...
}
```

### 6. Resource Leak Detection

**Coverage:** Memory, goroutine, and file handle leaks

```go
func TestComponent_ResourceUsage(t *testing.T) {
    tracker := testingpkg.NewResourceTracker("component_test")
    defer tracker.CheckLeaks(t)
    
    // Test component operations
    for i := 0; i < 100; i++ {
        component := createComponent()
        component.Process()
    }
    
    // CheckLeaks() automatically verifies no resource leaks
}
```

## Performance Optimization

### 1. Object Pooling

**Location:** `internal/build/pools.go`

Reduces GC pressure by reusing objects:

```go
type ObjectPools struct {
    buildResults *sync.Pool
    buffers      *sync.Pool
    slices       *sync.Pool
}

func (p *ObjectPools) GetBuildResult() *BuildResult {
    result := p.buildResults.Get().(*BuildResult)
    result.Reset() // Clear previous state
    return result
}
```

### 2. LRU Caching

**Location:** `internal/build/cache.go`

O(1) cache operations with doubly-linked list:

```go
type LRUCache struct {
    capacity int
    items    map[string]*cacheNode
    head     *cacheNode
    tail     *cacheNode
    mu       sync.RWMutex
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if node, exists := c.items[key]; exists {
        c.moveToHead(node)
        return node.value, true
    }
    return nil, false
}
```

### 3. Worker Pools

**Location:** `internal/build/pipeline.go`

Configurable parallel processing:

```go
type BuildPipeline struct {
    workers    []*BuildWorker
    queue      chan BuildTask
    results    chan BuildResult
    workerPool *WorkerPool
}

func (bp *BuildPipeline) Start(ctx context.Context) {
    for i := 0; i < bp.numWorkers; i++ {
        go bp.worker(ctx, i)
    }
}
```

### Performance Benchmarks

Current performance targets:
- **Build time**: <100ms for typical component
- **Memory usage**: <10MB for 100 components
- **Startup time**: <500ms for development server
- **WebSocket latency**: <10ms for change notifications

## Contributing Guidelines

### Code Style

1. **Follow Go conventions**: Use `gofmt`, `goimports`, `golangci-lint`
2. **Write comprehensive tests**: Aim for >90% coverage
3. **Document public APIs**: All exported functions need docs
4. **Use meaningful names**: Prefer clarity over brevity
5. **Handle errors properly**: Never ignore errors

### Commit Standards

Use conventional commits:

```
feat: add component hot reloading
fix: resolve WebSocket connection issues  
docs: update configuration guide
test: add integration tests for scanner
perf: implement object pooling for builds
security: add path traversal validation
```

### Pull Request Process

1. **Fork the repository** and create a feature branch
2. **Write tests** for new functionality
3. **Update documentation** if needed
4. **Run the full test suite**: `make test-ci`
5. **Submit PR** with clear description

### Security Considerations

When contributing:

1. **Validate all inputs** from users
2. **Use allowlists** instead of blocklists
3. **Avoid command execution** with user input
4. **Sanitize file paths** to prevent traversal
5. **Test security measures** with dedicated tests

### Performance Guidelines

1. **Profile before optimizing**: Use `go test -bench` and `pprof`
2. **Optimize hot paths**: Focus on frequently called code
3. **Use object pooling** for high-allocation operations
4. **Benchmark new features**: Include performance tests
5. **Monitor resource usage**: Check for leaks

## Debugging and Profiling

### Enable Debug Logging

```bash
export TEMPLAR_LOG_LEVEL=debug
templar serve --verbose
```

### Memory Profiling

```bash
go test -memprofile=mem.prof -bench=BenchmarkBuildPipeline ./internal/build
go tool pprof mem.prof
```

### CPU Profiling

```bash
go test -cpuprofile=cpu.prof -bench=BenchmarkComponentScan ./internal/scanner
go tool pprof cpu.prof
```

### Race Detection

```bash
make test-race
# or
go test -race ./...
```

### Resource Leak Detection

```bash
# Using built-in resource tracker
go test -tags=leak_detection ./...

# Manual tracking in tests
func TestMyFunction(t *testing.T) {
    tracker := testingpkg.NewResourceTracker("my_test")
    defer tracker.CheckLeaks(t)
    
    // Your test code
}
```

---

For more specific topics, see:
- [Security Architecture](SECURITY.md)
- [Performance Optimization](PERFORMANCE.md)
- [Testing Guide](TESTING.md)
- [API Documentation](API.md)