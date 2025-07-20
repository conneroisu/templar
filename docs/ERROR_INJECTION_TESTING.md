# Error Injection and Resource Leak Detection Testing Guide

This document describes the comprehensive error injection and resource leak detection framework implemented for the Templar CLI project.

## Overview

The testing framework provides two main capabilities:

1. **Error Injection**: Controlled failure simulation for testing error handling and resilience
2. **Resource Leak Detection**: Automated detection of memory, goroutine, and file handle leaks

## Error Injection Framework

### Basic Usage

```go
package mypackage

import (
    "testing"
    testingpkg "github.com/conneroisu/templar/internal/testing"
)

func TestMyFunction_ErrorHandling(t *testing.T) {
    // Create error injector
    injector := testingpkg.NewErrorInjector()
    
    // Configure error injection
    injector.InjectError("database.connect", errors.New("connection failed"))
    
    // Test your function
    result, err := myFunction(injector)
    
    // Verify error handling
    if err == nil {
        t.Error("Expected function to handle database connection error")
    }
}
```

### Advanced Error Injection

#### Single-Use Injections
```go
// Inject error that triggers only once
injector.InjectErrorOnce("network.timeout", errors.New("timeout"))
```

#### Counted Injections
```go
// Inject error for first 3 attempts, then succeed
injector.InjectErrorCount("file.write", errors.New("disk full"), 3)
```

#### Delayed Injections
```go
// Inject error with 500ms delay to simulate slow failures
injector.InjectErrorWithDelay("api.call", errors.New("service unavailable"), 500*time.Millisecond)
```

#### Probabilistic Injections
```go
// Inject error with 20% probability
target := injector.InjectError("cache.miss", errors.New("cache unavailable"))
target.WithProbability(0.2)
```

### Error Scenarios

Pre-defined scenarios simulate complex failure patterns:

```go
func TestBuildPipeline_FailureScenarios(t *testing.T) {
    injector := testingpkg.NewErrorInjector()
    manager := testingpkg.NewScenarioManager(injector)
    
    // Use predefined build failure scenario
    scenario := testingpkg.CreateBuildFailureScenario()
    manager.RegisterScenario(scenario)
    
    // Execute scenario
    err := manager.ExecuteScenario("build_failure")
    if err != nil {
        t.Fatalf("Failed to execute scenario: %v", err)
    }
    
    // Test your code with injected failures
    // ...
    
    // Stop scenario when done
    manager.StopScenario("build_failure")
}
```

#### Available Predefined Scenarios

1. **Build Failure Scenario** (`CreateBuildFailureScenario()`)
   - File permission errors (30% probability, 3 attempts)
   - Command execution failures (2 attempts with delay)
   - Disk full errors (10% probability, 1 attempt)

2. **Network Failure Scenario** (`CreateNetworkFailureScenario()`)
   - WebSocket connection failures (20% probability, 5 attempts)
   - HTTP request timeouts (3 attempts with 500ms delay)
   - Service unavailable errors (10% probability, 10 attempts)

3. **Resource Exhaustion Scenario** (`CreateResourceExhaustionScenario()`)
   - Out of memory errors (5% probability, 2 attempts)
   - Disk full errors (2% probability, 1 attempt)
   - Too many goroutines (1% probability, 1 attempt)

### Custom Scenarios

```go
// Create custom scenario
customScenario := &testingpkg.ErrorScenario{
    Name:        "custom_failure",
    Description: "Custom failure pattern for my component",
    Steps: []testingpkg.ErrorStep{
        {
            Operation:   "my.operation",
            Error:       errors.New("custom error"),
            Count:       5,
            Probability: 0.3,
            Delay:       100 * time.Millisecond,
        },
    },
}

manager.RegisterScenario(customScenario)
```

## Resource Leak Detection

### Basic Usage

```go
func TestMyFunction_ResourceLeaks(t *testing.T) {
    // Create resource tracker
    tracker := testingpkg.NewResourceTracker("my_function_test")
    defer tracker.CheckLeaks(t) // Automatically check for leaks at test end
    
    // Run your test code
    for i := 0; i < 100; i++ {
        result := myFunction()
        _ = result
    }
    
    // CheckLeaks() will automatically verify:
    // - No goroutine leaks
    // - No file handle leaks  
    // - No excessive memory growth
    // - No object leaks
}
```

### Custom Resource Limits

```go
func TestResourceIntensiveFunction(t *testing.T) {
    tracker := testingpkg.NewResourceTracker("intensive_test")
    
    // Define custom limits for resource-intensive operations
    limits := testingpkg.ResourceLimits{
        MaxGoroutineIncrease: 10,           // Allow up to 10 new goroutines
        MaxFileIncrease:      5,            // Allow up to 5 new file handles
        MaxMemoryIncrease:    50*1024*1024, // Allow up to 50MB memory increase
        MaxObjectIncrease:    5000,         // Allow up to 5000 new objects
        TolerancePercent:     0.2,          // 20% tolerance for variations
    }
    
    defer tracker.CheckLeaksWithLimits(t, limits)
    
    // Run resource-intensive test
    resourceIntensiveOperation()
}
```

### Continuous Monitoring

```go
func TestLongRunningOperation(t *testing.T) {
    // Monitor resources every 100ms
    monitor := testingpkg.NewResourceMonitor("long_running_test", 100*time.Millisecond)
    monitor.Start()
    defer monitor.Stop()
    
    // Run long operation
    longRunningOperation()
    
    // Check final state
    tracker := monitor.GetTracker()
    tracker.CheckLeaks(t)
    
    // Generate detailed report
    report := tracker.GenerateReport()
    t.Logf("Resource usage report:\n%s", report)
}
```

### Memory Pressure Testing

```go
func TestMemoryPressure(t *testing.T) {
    test := testingpkg.NewMemoryPressureTest("pressure_test")
    
    // Apply 100MB of memory pressure in 10MB chunks
    test.ApplyPressure(100, 10)
    
    // Run your code under memory pressure
    err := myMemoryIntensiveFunction()
    if err != nil {
        t.Errorf("Function failed under memory pressure: %v", err)
    }
    
    // Release pressure and verify memory recovery
    test.ReleasePressure()
    test.CheckMemoryRecovery(t)
}
```

## Integration with Build Pipeline

### Error Injection in Build Tests

```go
// +build error_injection

func TestBuildPipeline_ErrorInjection(t *testing.T) {
    injector := testingpkg.NewErrorInjector()
    tracker := testingpkg.NewResourceTracker("build_test")
    defer tracker.CheckLeaks(t)
    
    // Configure build failures
    injector.InjectErrorCount("file.read", testingpkg.ErrPermissionDenied, 3)
    
    // Create mock compiler with error injection
    mockCompiler := &MockCompilerWithInjection{
        injector: injector,
    }
    
    // Test build pipeline with failures
    pipeline := build.NewBuildPipeline(2, registry)
    pipeline.SetCompiler(mockCompiler)
    
    // Queue tasks and verify error handling
    // ...
}
```

### Resource Tracking for Build Operations

```go
func TestBuildPipeline_ResourceUsage(t *testing.T) {
    tracker := testingpkg.NewResourceTracker("build_resource_test")
    
    // Set limits appropriate for build operations
    limits := testingpkg.ResourceLimits{
        MaxGoroutineIncrease: 8,  // Build workers + management goroutines
        MaxFileIncrease:      20, // Template files + generated files
        MaxMemoryIncrease:    100*1024*1024, // 100MB for large builds
        MaxObjectIncrease:    10000,
        TolerancePercent:     0.15,
    }
    defer tracker.CheckLeaksWithLimits(t, limits)
    
    // Run build pipeline tests
    runBuildPipelineTests()
}
```

## Best Practices

### Error Injection

1. **Use Specific Operation Names**: Use descriptive, hierarchical names like `"database.connection.timeout"` rather than generic names like `"error"`

2. **Test Error Recovery**: Don't just test that errors occur, test that your code properly recovers from them

3. **Use Scenarios for Complex Testing**: For testing multiple failure modes, use scenario-based testing rather than individual injections

4. **Clean Up Injections**: Always clear or disable injections between tests to avoid interference

### Resource Leak Detection

1. **Use Appropriate Limits**: Set realistic limits based on what your code actually does. Don't use overly strict limits that cause false positives

2. **Account for Test Environment**: CI environments may have different resource patterns than local development

3. **Force Cleanup**: The framework automatically runs GC before checking, but you can add explicit cleanup in your tests

4. **Monitor Long-Running Tests**: Use continuous monitoring for tests that run for extended periods

### Integration Testing

1. **Combine Both Frameworks**: Use error injection and resource tracking together to ensure error handling doesn't cause leaks

2. **Test Realistic Scenarios**: Use the predefined scenarios as starting points, but customize them for your specific use cases

3. **Verify Recovery**: Always test that your system properly recovers after error injection scenarios end

## Running Error Injection Tests

Error injection tests are tagged with `error_injection` build tag to prevent them from running in normal test suites:

```bash
# Run normal tests (excluding error injection)
go test ./...

# Run error injection tests specifically
go test -tags=error_injection ./...

# Run specific error injection test
go test -tags=error_injection ./internal/build -run TestBuildPipeline_ErrorInjection

# Run with verbose output for detailed error information
go test -tags=error_injection -v ./...
```

## CI Integration

Add error injection tests to your CI pipeline:

```yaml
# .github/workflows/test.yml
- name: Error Injection Tests
  run: go test -tags=error_injection -v ./...

- name: Resource Leak Tests  
  run: go test -tags=leak_detection -v ./...

- name: Performance Regression Tests
  run: go test -bench=. -benchmem ./internal/testing
```

## Troubleshooting

### Common Issues

1. **False Positive Leak Detection**
   - Increase tolerance percentage
   - Check if test creates expected resources
   - Verify cleanup code is running

2. **Error Injection Not Working**
   - Verify operation name matches exactly
   - Check if error injection is enabled
   - Ensure ShouldFail() is called in the right place

3. **Flaky Tests**
   - Use deterministic error injection (avoid pure probability)
   - Add appropriate delays for async operations
   - Increase timeouts for resource cleanup

### Debugging

Enable detailed logging for troubleshooting:

```go
// Get detailed injection statistics
stats := injector.GetStats()
t.Logf("Injection stats: %+v", stats)

// Get detailed resource usage
usage := tracker.GetResourceUsage()
t.Logf("Resource usage: %+v", usage)

// Generate full report
report := tracker.GenerateReport()
t.Logf("Full report:\n%s", report)
```

## Performance Impact

The error injection and resource tracking frameworks are designed to have minimal performance impact:

- **Error Injection**: ~120ns per ShouldFail() call when no injection is configured
- **Resource Tracking**: ~1µs per TakeSample() call
- **Memory Overhead**: <1MB for typical test scenarios

Performance impact is negligible for unit and integration tests, making it safe to use in comprehensive test suites.