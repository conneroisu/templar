# Contributing to Templar

Thank you for your interest in contributing to Templar! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Making Contributions](#making-contributions)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Architecture Overview](#architecture-overview)
- [Security Considerations](#security-considerations)

## Code of Conduct

This project adheres to a code of conduct that we expect all participants to honor. Please be respectful and professional in all interactions.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Basic understanding of Go templ components
- Familiarity with CLI development (helpful but not required)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/templar.git
   cd templar
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/conneroisu/templar.git
   ```

## Development Environment

### Nix Development Environment (Recommended)

The project includes a Nix flake for reproducible development environments:

```bash
# Enter development environment
nix develop

# Quick file editing shortcuts
dx  # Edit flake.nix
gx  # Edit go.mod
```

The Nix environment provides Go 1.24, development tools (air, golangci-lint, gopls), and hot reloading capabilities.

### Manual Setup

If you prefer not to use Nix:

1. Install Go 1.21+
2. Install development dependencies:
   ```bash
   go mod download
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install github.com/air-verse/air@latest
   ```

### Essential Development Commands

```bash
# Basic development workflow
make dev-setup          # Set up development environment  
make serve               # Start development server
make build               # Build the project
make test                # Run all tests
make fmt                 # Format code
make lint                # Run linter

# Testing commands
make test-unit           # Unit tests only
make test-integration    # Integration tests
make test-security       # Security tests
make test-e2e            # End-to-end tests
make test-coverage       # Generate coverage report
make test-race           # Race detection

# Build and quality checks
make build-prod          # Static production binary
make security-scan       # Vulnerability scanning
make pre-commit          # Format, lint, race detection, security tests
```

## Making Contributions

### Types of Contributions

We welcome various types of contributions:

- **Bug fixes** - Fix issues and improve reliability
- **Features** - Add new functionality
- **Documentation** - Improve docs, examples, and guides
- **Performance** - Optimize speed and memory usage
- **Security** - Enhance security posture
- **Testing** - Add tests and improve coverage
- **Refactoring** - Improve code structure and maintainability

### Contribution Workflow

1. **Check existing issues** - Look for related issues or discussions
2. **Create an issue** - For significant changes, create an issue first to discuss
3. **Create a branch** - Use descriptive branch names:
   ```bash
   git checkout -b feature/component-scaffolding
   git checkout -b fix/websocket-memory-leak
   git checkout -b docs/contributing-guide
   ```
4. **Make changes** - Follow the code style guidelines
5. **Add tests** - Ensure your changes are well-tested
6. **Run quality checks** - `make pre-commit`
7. **Commit changes** - Use conventional commit format
8. **Push and create PR** - Open a pull request with a clear description

## Code Style Guidelines

### Go Code Standards

- **Formatting**: Use `gofmt` and `goimports` (run `make fmt`)
- **Linting**: Follow `golangci-lint` recommendations (run `make lint`)
- **Naming**: 
  - Use camelCase for variables and functions
  - Use PascalCase for exported types and functions
  - Use descriptive names (prefer `componentRegistry` over `cr`)
- **Error Handling**: 
  - Always handle errors explicitly
  - Use wrapped errors with context: `fmt.Errorf("failed to load config: %w", err)`
- **Documentation**:
  - Document all exported functions and types
  - Use complete sentences in comments
  - Provide examples for complex functions

### File Organization

- Keep files focused on a single responsibility
- Group related functionality into packages
- Use internal packages for implementation details
- Place tests alongside the code they test

### Architecture Principles

- **Security First**: All input validation, command injection prevention
- **Interface-Based Design**: Use interfaces for modularity and testability
- **Performance Conscious**: Consider memory allocation and concurrency
- **Error Resilience**: Graceful degradation and comprehensive error handling

## Testing

### Test Categories

- **Unit Tests** (`*_test.go`): Test individual functions and methods
- **Integration Tests** (`integration_tests/`): Test component interactions
- **Security Tests** (`*_security_test.go`): Test security hardening
- **Property Tests** (`*_property_test.go`): Property-based testing
- **Fuzz Tests** (`*_fuzz_test.go`): Fuzz testing for input validation
- **Benchmark Tests** (`*_bench_test.go`): Performance benchmarks

### Testing Guidelines

- **Test Coverage**: Aim for >80% coverage for new code
- **Test Structure**: Use table-driven tests where appropriate
- **Test Isolation**: Tests should be independent and repeatable
- **Mock Usage**: Use mocks to isolate units under test
- **Security Testing**: Include security test cases for security-sensitive code

### Running Tests

```bash
# Run specific test categories
make test-unit           # Fast unit tests
make test-integration    # Integration tests
make test-security       # Security-focused tests
make test-e2e            # End-to-end workflow tests

# Advanced testing
make test-race           # Race condition detection  
make test-coverage       # Coverage analysis
make test-bench          # Performance benchmarks
```

## Pull Request Process

### PR Requirements

1. **Code Quality**:
   - All tests pass (`make test`)
   - Linting passes (`make lint`)
   - No race conditions (`make test-race`)
   - Security tests pass (`make test-security`)

2. **Documentation**:
   - Update relevant documentation
   - Add or update tests
   - Update CHANGELOG.md for user-facing changes

3. **Commit Messages**:
   Use conventional commit format:
   ```
   feat(scanner): add parallel file processing for large codebases
   
   - Implement worker pool for concurrent file scanning
   - Add configurable concurrency limits
   - Improve performance by 3x for projects with >1000 files
   
   Fixes #123
   ```

### PR Description Template

```markdown
## Description
Brief description of changes and motivation.

## Changes Made
- [ ] Added new feature X
- [ ] Fixed bug in Y
- [ ] Updated documentation for Z

## Testing
- [ ] Added unit tests
- [ ] Added integration tests
- [ ] Manually tested the changes
- [ ] All existing tests pass

## Security Considerations
- [ ] No security implications
- [ ] Security review completed
- [ ] Added security tests

## Breaking Changes
- [ ] No breaking changes
- [ ] Breaking changes documented in CHANGELOG.md
```

### Review Process

1. **Automated Checks**: CI runs tests, linting, and security scans
2. **Code Review**: At least one maintainer reviews the code
3. **Testing**: Reviewers may test the changes locally
4. **Approval**: PR approved and merged by maintainer

## Issue Reporting

### Bug Reports

Use the bug report template and include:

- **Environment**: OS, Go version, Templar version
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Reproduction Steps**: Minimal steps to reproduce
- **Additional Context**: Error logs, screenshots, etc.

### Feature Requests

Use the feature request template and include:

- **Use Case**: Why is this needed?
- **Proposed Solution**: How should it work?
- **Alternatives**: Other solutions considered
- **Additional Context**: Examples, mockups, etc.

### Security Issues

**DO NOT** create public issues for security vulnerabilities. Instead:

1. Email security concerns to: [maintainer email if available]
2. Use GitHub's private vulnerability reporting if available
3. Provide detailed reproduction steps
4. Allow reasonable time for fixes before disclosure

## Architecture Overview

### Core Components

- **CLI Commands** (`cmd/`): Cobra-based command interface
- **Component Scanner** (`internal/scanner/`): Discovers and analyzes templ files
- **Build Pipeline** (`internal/build/`): Compiles components with caching
- **Development Server** (`internal/server/`): HTTP server with WebSocket support
- **Component Registry** (`internal/registry/`): Component metadata management
- **File Watcher** (`internal/watcher/`): Real-time file change detection
- **Configuration** (`internal/config/`): Application configuration management

### Key Interfaces

- `ComponentRegistry`: Component discovery and management
- `BuildPipeline`: Component compilation and caching
- `FileWatcher`: File system monitoring
- `ConfigManager`: Configuration handling

## Security Considerations

### Security-First Development

- **Input Validation**: Validate all user inputs
- **Command Injection Prevention**: Use allowlists for external commands
- **Path Traversal Protection**: Validate and sanitize file paths
- **WebSocket Security**: Implement origin validation and rate limiting
- **Memory Safety**: Prevent buffer overflows and memory leaks

### Security Testing

- Run security tests: `make test-security`
- Use fuzz testing: `make test-fuzz`
- Scan for vulnerabilities: `make security-scan`
- Review all user-facing interfaces for security implications

## Community and Support

- **GitHub Discussions**: For questions and community discussions
- **Issues**: For bug reports and feature requests
- **Pull Requests**: For code contributions
- **Documentation**: Keep docs up-to-date with changes

## Recognition

Contributors are recognized in:
- GitHub contributors list
- Release notes for significant contributions
- Special recognition for security improvements

---

Thank you for contributing to Templar! Your contributions help make Go templ development faster and more enjoyable for everyone.