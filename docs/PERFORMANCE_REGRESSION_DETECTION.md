# Performance Regression Detection System

This document describes the comprehensive performance regression detection system implemented for Templar CLI.

## Overview

The performance regression detection system provides automated monitoring of benchmark performance, baseline establishment, regression detection with configurable thresholds, and CI/CD integration for continuous performance monitoring.

## Components

### 1. Performance Detector (`internal/performance/detector.go`)
- **Benchmark Parsing**: Parses Go benchmark output and extracts structured performance data
- **Baseline Management**: Maintains historical performance baselines with statistical analysis
- **Regression Detection**: Detects performance, memory, and allocation regressions with configurable thresholds
- **Statistical Analysis**: Calculates mean, median, standard deviation, and percentiles for baseline data

### 2. CI Integration (`internal/performance/ci.go`)
- **Automated Benchmarking**: Executes benchmarks and processes results
- **Report Generation**: Creates comprehensive reports in multiple formats (text, JSON, JUnit, GitHub Actions)
- **Threshold Configuration**: Configurable regression thresholds and confidence levels
- **Git Integration**: Tracks performance changes across commits and branches

### 3. CLI Commands (`cmd/performance.go`)
- **Performance Check**: `templar performance check` - Run benchmarks and detect regressions
- **Baseline Management**: `templar performance baseline create|list` - Manage performance baselines
- **Report Generation**: `templar performance report` - Generate comprehensive performance reports

### 4. GitHub Actions Integration (`.github/workflows/performance.yml`)
- **Automated CI/CD**: Runs performance checks on pull requests and main branch pushes
- **Baseline Caching**: Caches performance baselines across CI runs
- **PR Comments**: Automatically comments on PRs with performance analysis
- **Failure Handling**: Fails CI on critical performance regressions

## Configuration

### Default Regression Thresholds

```go
SlownessThreshold: 1.15  // 15% performance degradation
MemoryThreshold:   1.20  // 20% memory increase
AllocThreshold:    1.25  // 25% allocation increase
MinSamples:        5     // Minimum samples for detection
ConfidenceLevel:   0.95  // 95% statistical confidence
```

### Severity Levels

- **Critical**: >2x threshold (e.g., >30% performance degradation)
- **Major**: >1.15x threshold (e.g., >17.25% performance degradation)
- **Minor**: Above threshold but below major level

## Usage Examples

### Running Performance Checks

```bash
# Basic performance check
templar performance check

# Check specific packages with custom thresholds
templar performance check \
  --packages="./internal/scanner,./internal/build" \
  --slowness-threshold=1.10 \
  --memory-threshold=1.15 \
  --format=json

# Check with GitHub Actions format
templar performance check \
  --format=github \
  --fail-on-critical \
  --output=performance-report.txt
```

### Managing Baselines

```bash
# Create initial baselines
templar performance baseline create --packages="./internal/..."

# List existing baselines
templar performance baseline list

# Use custom baseline directory
templar performance baseline create --baseline-dir=".perf-baselines"
```

### Generating Reports

```bash
# Generate comprehensive performance report
templar performance report --format=json --output=perf-report.json

# Generate text report with git information
templar performance report --format=text
```

## CI/CD Integration

### GitHub Actions Setup

The system includes a GitHub Actions workflow that:

1. **Runs on Events**: Pull requests and pushes to main/dev branches
2. **Caches Baselines**: Maintains performance baselines across runs
3. **Detects Regressions**: Compares current performance against baselines
4. **Reports Results**: Comments on PRs with performance analysis
5. **Fails on Critical**: Fails CI builds for critical performance regressions

### Workflow Configuration

```yaml
- name: Run performance benchmarks and check for regressions
  run: |
    templar performance check \
      --packages="./internal/scanner,./internal/build,./internal/registry" \
      --format=github \
      --fail-on-critical \
      --baseline-dir=.performance-baselines \
      --slowness-threshold=1.15 \
      --memory-threshold=1.20 \
      --alloc-threshold=1.25
```

## Report Formats

### Text Format
Human-readable reports with:
- Performance summary with health score
- Detailed regression analysis
- Top performing benchmarks
- Actionable recommendations

### JSON Format
Structured data for programmatic processing:
- Complete benchmark results
- Regression detection details
- Statistical analysis
- Git and environment metadata

### JUnit Format
XML format for CI integration:
- Test case for each benchmark
- Failures for detected regressions
- Compatible with CI reporting tools

### GitHub Actions Format
Annotations for GitHub integration:
- Error/warning annotations for regressions
- PR comment formatting
- Workflow status indicators

## Performance Metrics

### Tracked Metrics
- **Execution Time**: Nanoseconds per operation
- **Memory Usage**: Bytes allocated per operation
- **Allocation Count**: Number of allocations per operation
- **Throughput**: Operations per second (when available)

### Baseline Statistics
- **Mean**: Average performance across samples
- **Median**: Middle value for robust central tendency
- **Standard Deviation**: Measure of performance variability
- **Percentiles**: P95 and P99 for outlier analysis
- **Min/Max**: Performance range bounds

## Security Considerations

- **Path Validation**: All file paths validated to prevent traversal attacks
- **Input Sanitization**: Benchmark names and parameters sanitized
- **Baseline Integrity**: Baseline files protected with proper permissions
- **Command Validation**: Only approved commands executed for benchmarking

## Integration Points

### Scanner Performance
- **File Scanning**: 85% improvement achieved through parallel scanning
- **Path Validation**: Security validation with minimal performance impact
- **Memory Management**: Bounded queues prevent memory leaks

### Build Pipeline Performance
- **Cache Performance**: 17-169x speedup through LRU cache optimization
- **Worker Pool Efficiency**: Optimal worker count based on CPU cores
- **Memory Optimization**: Object pooling for high-frequency operations

### WebSocket Performance
- **Connection Management**: Efficient client handling
- **Message Broadcasting**: Optimized slice pooling
- **Memory Leak Prevention**: Proper resource cleanup

## Troubleshooting

### Common Issues

1. **No Benchmarks Found**
   - Ensure packages contain benchmark functions
   - Check package paths are correct
   - Verify benchmark naming conventions (`BenchmarkXxx`)

2. **High Baseline Variance**
   - Run more benchmark iterations (`-count=5`)
   - Ensure consistent testing environment
   - Check for external system interference

3. **False Positives**
   - Adjust confidence thresholds
   - Increase minimum sample requirements
   - Review baseline calculation methods

### Performance Optimization

1. **Benchmark Duration**
   - Use targeted package selection
   - Implement benchmark timeouts
   - Consider parallel benchmark execution

2. **Baseline Storage**
   - Regular cleanup of old baselines
   - Compress baseline data for large projects
   - Implement baseline rotation policies

## Future Enhancements

### Planned Features
- **Trend Analysis**: Long-term performance trend visualization
- **Automated Optimization**: Suggest performance improvements
- **Custom Metrics**: Support for application-specific metrics
- **Performance Budgets**: Set and enforce performance budgets
- **Historical Comparison**: Compare performance across releases

### Integration Opportunities
- **Monitoring Integration**: Integration with Prometheus/Grafana
- **Alerting Systems**: Real-time performance alerts
- **Performance Dashboard**: Web-based performance monitoring
- **Load Testing**: Integration with load testing frameworks

## References

- [Go Benchmarking Documentation](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Statistical Analysis Methods](https://en.wikipedia.org/wiki/Statistical_process_control)
- [GitHub Actions CI/CD](https://docs.github.com/en/actions)
- [Performance Testing Best Practices](https://martinfowler.com/articles/performance-testing.html)