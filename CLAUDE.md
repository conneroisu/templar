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

- **Unit tests**: Component-level testing with mocks and table-driven tests (`make test-unit`)
- **Integration tests**: Cross-component testing with real file system and WebSocket connections (`make test-integration`)
- **Security tests**: Comprehensive security hardening validation (`make test-security`)
- **Property-based tests**: Randomized testing with gopter framework (`make test-property`)
- **Fuzz tests**: Security-focused input validation (`make fuzz-short`, `make fuzz-security`)
- **Performance benchmarks**: Memory usage, concurrency, and throughput testing (`make test-bench`)
- **E2E tests**: Full workflow testing with temporary directories and live servers (`make test-e2e`)

### Security Test Coverage

The codebase includes comprehensive security testing covering:
- **Command injection prevention**: Strict allowlisting in build operations with edge case testing
- **Path traversal protection**: Unicode normalization, encoding schemes, and directory escape validation
- **WebSocket origin validation**: Scheme/host checking with CSRF protection and message size limits
- **Input validation**: Unicode attack prevention (homoglyphs, bidirectional text, zero-width chars)
- **Race condition prevention**: Mutex-protected concurrent access with property-based testing
- **Memory leak prevention**: Goroutine lifecycle management with resource limit enforcement

### Testing Commands Reference

```bash
# Quick testing workflow
make test                 # Standard test suite
make test-ci              # Comprehensive CI-like testing
make pre-commit          # Pre-commit validation (format, lint, race, security)

# Specialized testing
make test-property       # Property-based tests with gopter (thread safety, etc.)
make fuzz-short          # 30-second fuzz tests across all components
make fuzz-security       # 10-minute comprehensive security fuzzing
make test-race           # Race condition detection
make security-scan       # Vulnerability scanning with govulncheck

# Performance and analysis
make test-bench          # Performance benchmarks (30M+ ops/sec validation)
make test-coverage       # HTML coverage reports
make coverage-analysis   # Advanced coverage analysis
```

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

- **Cobra CLI structure**: Each command in separate file with comprehensive validation
- **Event-driven architecture**: Registry broadcasts changes, components subscribe
- **Worker pool pattern**: Build pipeline uses configurable worker pools with resource limits
- **LRU caching**: O(1) cache eviction with doubly-linked lists and memory mapping for large files
- **Object pooling**: Memory optimization with BuildResult, BuildTask, and buffer pools
- **Security-first design**: Defense-in-depth with input validation, allowlisting, and origin checking
- **Property-based testing**: Thread safety validation with gopter framework (100+ test cases)
- **Performance optimization**: 30M+ operations/second with concurrent processing

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

# Single test execution patterns
go test -v ./internal/build -run TestBuildWorker_ErrorHandling
go test -v ./cmd -run TestValidateArgument_EdgeCases
go test -v -tags=property ./internal/errors  # Property-based tests
```

### Performance Characteristics

The codebase achieves high performance through:
- **BuildPipeline**: 30M+ operations/second
- **Cache performance**: 100x improvement (4.7ms → 61µs with caching)
- **Worker pools**: 5.9M operations/second with proper resource management
- **Memory mapping**: Optimized file I/O for components >64KB
- **Object pooling**: Reduced memory allocations in hot paths

The development environment includes pprof and graphviz for performance analysis and profiling.

## Recent Development Context

### Test Coverage Status (2025-01-21)

The project has achieved **enterprise-grade reliability** with comprehensive test coverage:
- **83 test files** covering all critical components
- **7,000+ lines of test code** added for security hardening and performance validation
- **Property-based testing** implemented with gopter framework for thread safety
- **Security hardening** complete with Unicode attack prevention and injection protection
- **Performance optimization** validated with benchmarks achieving 30M+ ops/sec

### Key Test Files Added

Critical test coverage includes:
- `internal/build/buildworker_test.go` - BuildWorker error handling and cancellation (778 lines)
- `internal/build/pipeline_integration_test.go` - End-to-end pipeline testing (658 lines) 
- `internal/plugins/integration_test.go` - Plugin system integration and security (456 lines)
- `cmd/validation_edge_cases_test.go` - Unicode security and injection prevention (580 lines)
- `internal/errors/errors_property_test.go` - Property-based concurrent testing (369 lines)

All tests pass successfully with race detection and security validation enabled.

<!-- BACKLOG.MD GUIDELINES START -->
# Instructions for the usage of Backlog.md CLI Tool

## 1. Source of Truth

- Tasks live under **`backlog/tasks/`** (drafts under **`backlog/drafts/`**).
- Every implementation decision starts with reading the corresponding Markdown task file.
- Project documentation is in **`backlog/docs/`**.
- Project decisions are in **`backlog/decisions/`**.

## 2. Defining Tasks

### **Title**

Use a clear brief title that summarizes the task.

### **Description**: (The **"why"**)

Provide a concise summary of the task purpose and its goal. Do not add implementation details here. It
should explain the purpose and context of the task. Code snippets should be avoided.

### **Acceptance Criteria**: (The **"what"**)

List specific, measurable outcomes that define what means to reach the goal from the description. Use checkboxes (`- [ ]`) for tracking.
When defining `## Acceptance Criteria` for a task, focus on **outcomes, behaviors, and verifiable requirements** rather
than step-by-step implementation details.
Acceptance Criteria (AC) define *what* conditions must be met for the task to be considered complete.
They should be testable and confirm that the core purpose of the task is achieved.
**Key Principles for Good ACs:**

- **Outcome-Oriented:** Focus on the result, not the method.
- **Testable/Verifiable:** Each criterion should be something that can be objectively tested or verified.
- **Clear and Concise:** Unambiguous language.
- **Complete:** Collectively, ACs should cover the scope of the task.
- **User-Focused (where applicable):** Frame ACs from the perspective of the end-user or the system's external behavior.

    - *Good Example:* "- [ ] User can successfully log in with valid credentials."
    - *Good Example:* "- [ ] System processes 1000 requests per second without errors."
    - *Bad Example (Implementation Step):* "- [ ] Add a new function `handleLogin()` in `auth.ts`."

### Task file

Once a task is created it will be stored in `backlog/tasks/` directory as a Markdown file with the format
`task-<id> - <title>.md` (e.g. `task-42 - Add GraphQL resolver.md`).

### Additional task requirements

- Tasks must be **atomic** and **testable**. If a task is too large, break it down into smaller subtasks.
  Each task should represent a single unit of work that can be completed in a single PR.

- **Never** reference tasks that are to be done in the future or that are not yet created. You can only reference
  previous
  tasks (id < current task id).

- When creating multiple tasks, ensure they are **independent** and they do not depend on future tasks.   
  Example of wrong tasks splitting: task 1: "Add API endpoint for user data", task 2: "Define the user model and DB
  schema".  
  Example of correct tasks splitting: task 1: "Add system for handling API requests", task 2: "Add user model and DB
  schema", task 3: "Add API endpoint for user data".

## 3. Recommended Task Anatomy

```markdown
# task‑42 - Add GraphQL resolver

## Description (the why)

Short, imperative explanation of the goal of the task and why it is needed.

## Acceptance Criteria (the what)

- [ ] Resolver returns correct data for happy path
- [ ] Error response matches REST
- [ ] P95 latency ≤ 50 ms under 100 RPS

## Implementation Plan (the how)

1. Research existing GraphQL resolver patterns
2. Implement basic resolver with error handling
3. Add performance monitoring
4. Write unit and integration tests
5. Benchmark performance under load

## Implementation Notes (only added after working on the task)

- Approach taken
- Features implemented or modified
- Technical decisions and trade-offs
- Modified or added files
```

## 6. Implementing Tasks

Mandatory sections for every task:

- **Implementation Plan**: (The **"how"**) Outline the steps to achieve the task. Because the implementation details may
  change after the task is created, **the implementation notes must be added only after putting the task in progress**
  and before starting working on the task.
- **Implementation Notes**: Document your approach, decisions, challenges, and any deviations from the plan. This
  section is added after you are done working on the task. It should summarize what you did and why you did it. Keep it
  concise but informative.

**IMPORTANT**: Do not implement anything else that deviates from the **Acceptance Criteria**. If you need to
implement something that is not in the AC, update the AC first and then implement it or create a new task for it.

## 2. Typical Workflow

```bash
# 1 Identify work
backlog task list -s "To Do" --plain

# 2 Read details & documentation
backlog task 42 --plain
# Read also all documentation files in `backlog/docs/` directory.
# Read also all decision files in `backlog/decisions/` directory.

# 3 Start work: assign yourself & move column
backlog task edit 42 -a @{yourself} -s "In Progress"

# 4 Add implementation plan before starting
backlog task edit 42 --plan "1. Analyze current implementation\n2. Identify bottlenecks\n3. Refactor in phases"

# 5 Break work down if needed by creating subtasks or additional tasks
backlog task create "Refactor DB layer" -p 42 -a @{yourself} -d "Description" --ac "Tests pass,Performance improved"

# 6 Complete and mark Done
backlog task edit 42 -s Done --notes "Implemented GraphQL resolver with error handling and performance monitoring"
```

### 7. Final Steps Before Marking a Task as Done

Always ensure you have:

1. ✅ Marked all acceptance criteria as completed (change `- [ ]` to `- [x]`)
2. ✅ Added an `## Implementation Notes` section documenting your approach
3. ✅ Run all tests and linting checks
4. ✅ Updated relevant documentation

## 8. Definition of Done (DoD)

A task is **Done** only when **ALL** of the following are complete:

1. **Acceptance criteria** checklist in the task file is fully checked (all `- [ ]` changed to `- [x]`).
2. **Implementation plan** was followed or deviations were documented in Implementation Notes.
3. **Automated tests** (unit + integration) cover new logic.
4. **Static analysis**: linter & formatter succeed.
5. **Documentation**:
    - All relevant docs updated (any relevant README file, backlog/docs, backlog/decisions, etc.).
    - Task file **MUST** have an `## Implementation Notes` section added summarising:
        - Approach taken
        - Features implemented or modified
        - Technical decisions and trade-offs
        - Modified or added files
6. **Review**: self review code.
7. **Task hygiene**: status set to **Done** via CLI (`backlog task edit <id> -s Done`).
8. **No regressions**: performance, security and licence checks green.

⚠️ **IMPORTANT**: Never mark a task as Done without completing ALL items above.

## 9. Handy CLI Commands

| Purpose          | Command                                                                |
|------------------|------------------------------------------------------------------------|
| Create task      | `backlog task create "Add OAuth"`                                      |
| Create with desc | `backlog task create "Feature" -d "Enables users to use this feature"` |
| Create with AC   | `backlog task create "Feature" --ac "Must work,Must be tested"`        |
| Create with deps | `backlog task create "Feature" --dep task-1,task-2`                    |
| Create sub task  | `backlog task create -p 14 "Add Google auth"`                          |
| List tasks       | `backlog task list --plain`                                            |
| View detail      | `backlog task 7 --plain`                                               |
| Edit             | `backlog task edit 7 -a @{yourself} -l auth,backend`                   |
| Add plan         | `backlog task edit 7 --plan "Implementation approach"`                 |
| Add AC           | `backlog task edit 7 --ac "New criterion,Another one"`                 |
| Add deps         | `backlog task edit 7 --dep task-1,task-2`                              |
| Add notes        | `backlog task edit 7 --notes "We added this and that feature because"` |
| Mark as done     | `backlog task edit 7 -s "Done"`                                        |
| Archive          | `backlog task archive 7`                                               |
| Draft flow       | `backlog draft create "Spike GraphQL"` → `backlog draft promote 3.1`   |
| Demote to draft  | `backlog task demote <task-id>`                                        |

## 10. Tips for AI Agents

- **Always use `--plain` flag** when listing or viewing tasks for AI-friendly text output instead of using Backlog.md
  interactive UI.
- When users mention to create a task, they mean to create a task using Backlog.md CLI tool.

<!-- BACKLOG.MD GUIDELINES END -->
