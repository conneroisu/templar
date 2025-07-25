# Fuzzing Documentation

This document describes the comprehensive fuzzing test suite implemented for Templar, a rapid prototyping CLI tool for Go templ. The fuzzing tests are designed to validate security hardening, input validation, and robustness against malicious inputs.

## Overview

Fuzzing is an automated testing technique that provides invalid, unexpected, or random data as inputs to a program. Our fuzzing test suite focuses on:

- **Security validation**: Preventing command injection, XSS, path traversal, and other attacks
- **Input validation**: Testing edge cases and malformed inputs
- **Robustness**: Ensuring the system handles unexpected inputs gracefully
- **Memory safety**: Preventing buffer overflows and memory corruption

## Test Categories

### 1. Component Scanner Fuzzing (`internal/scanner/scanner_fuzz_test.go`)

Tests the component scanner with various malicious and malformed templ file contents.

**Key Test Functions:**
- `FuzzScanFile`: Tests file scanning with malformed templ content
- `FuzzParseTemplComponent`: Tests component declaration parsing
- `FuzzScanDirectory`: Tests directory scanning with various path inputs
- `FuzzExtractParameters`: Tests parameter extraction from component signatures
- `FuzzComponentFileContent`: Tests full file content parsing with malicious patterns

**Security Features Tested:**
- Control character filtering in component names and parameters
- Path traversal prevention in file paths
- XSS pattern detection in component content
- SQL injection pattern detection
- Command injection prevention

### 2. Configuration Fuzzing (`internal/config/config_fuzz_test.go`)

Tests configuration parsing and validation with various malformed and malicious configuration data.

**Key Test Functions:**
- `FuzzLoadConfig`: Tests YAML configuration loading with malformed inputs
- `FuzzConfigValidation`: Tests configuration structure validation
- `FuzzYAMLParsing`: Tests YAML parsing edge cases and potential attacks
- `FuzzEnvironmentVariables`: Tests environment variable parsing

**Security Features Tested:**
- YAML injection prevention
- Environment variable sanitization
- Port range validation (1-65535)
- Host validation and control character filtering
- Path validation for scan directories

### 3. WebSocket Fuzzing (`internal/server/websocket_fuzz_test.go`)

Tests WebSocket functionality with various malicious inputs and connection attempts.

**Key Test Functions:**
- `FuzzWebSocketOriginValidation`: Tests origin validation with malicious origins
- `FuzzWebSocketMessage`: Tests message handling with various payloads
- `FuzzWebSocketHeaders`: Tests header processing with malformed headers
- `FuzzWebSocketURL`: Tests URL handling with various patterns

**Security Features Tested:**
- Origin validation (only allows localhost and 127.0.0.1)
- Scheme validation (only http/https)
- Control character filtering in origins and messages
- Message size limits
- Protocol handler abuse prevention

### 4. Path Validation Fuzzing (`internal/validation/validation_fuzz_test.go`)

Tests URL and path validation with various attack patterns.

**Key Test Functions:**
- `FuzzValidateURL`: Tests URL validation with malicious inputs
- `FuzzURLParsing`: Tests URL parsing edge cases
- `FuzzPathTraversal`: Tests path traversal pattern detection
- `FuzzProtocolHandlers`: Tests protocol handler validation
- `FuzzCommandInjection`: Tests command injection pattern detection

**Security Features Tested:**
- Shell metacharacter detection (`;`, `&`, `|`, `` ` ``, `$`, etc.)
- Path traversal prevention (`../`, encoded variants)
- Protocol handler validation (only http/https allowed)
- Command injection prevention
- Encoded attack pattern detection

### 5. Build Pipeline Fuzzing (`internal/build/build_fuzz_test.go`)

Tests the build pipeline with various malicious component inputs and commands.

**Key Test Functions:**
- `FuzzBuildPipelineInput`: Tests build pipeline with malicious component content
- `FuzzCompilerCommand`: Tests command validation and execution
- `FuzzBuildCache`: Tests build cache with various key/value combinations
- `FuzzBuildTaskQueue`: Tests task queue with malicious component data
- `FuzzErrorParsing`: Tests error parsing with malicious compiler outputs
- `FuzzBuildMetrics`: Tests metrics collection with edge case inputs

**Security Features Tested:**
- Command allowlisting (only `templ` and `go` commands)
- Argument validation and shell metacharacter filtering
- Path traversal prevention in component paths
- XSS pattern detection in build outputs
- Cache key validation and control character filtering

### 6. Error Handling Fuzzing (`internal/errors/errors_fuzz_test.go`)

Tests error parsing and collection with various malicious error outputs.

**Key Test Functions:**
- `FuzzErrorParser`: Tests error parsing with malformed compiler outputs
- `FuzzErrorCollection`: Tests error collection and aggregation
- `FuzzHTMLErrorOverlay`: Tests HTML error overlay generation
- `FuzzErrorSeverityClassification`: Tests error severity classification
- `FuzzErrorTemplateRendering`: Tests template rendering with malicious inputs

**Security Features Tested:**
- HTML escaping in error overlays
- XSS prevention in error messages
- Path traversal prevention in error file paths
- Template injection prevention
- Control character filtering

### 7. Registry Fuzzing (`internal/registry/registry_fuzz_test.go`)

Tests component registry operations with various malicious component data.

**Key Test Functions:**
- `FuzzComponentRegistration`: Tests component registration with malicious data
- `FuzzComponentSearch`: Tests component search with malicious queries
- `FuzzComponentParameters`: Tests parameter parsing with malicious inputs
- `FuzzComponentDependencies`: Tests dependency handling
- `FuzzEventSubscription`: Tests event subscription safety
- `FuzzComponentSerialization`: Tests component serialization/deserialization

**Security Features Tested:**
- Component name validation and control character filtering
- Path traversal prevention in component paths
- XSS pattern detection in component content
- Parameter type validation
- Dependency validation and sanitization

## Running Fuzzing Tests

### Prerequisites

- Go 1.18 or later (for native fuzzing support)
- Sufficient disk space for fuzzing corpus generation
- Appropriate resource limits to prevent system overload

### Basic Fuzzing Commands

```bash
# Run all fuzzing tests for 30 seconds each
make fuzz

# Run specific fuzzing test
go test -fuzz=FuzzScanFile ./internal/scanner/

# Run with custom duration
go test -fuzz=FuzzWebSocketOriginValidation -fuzztime=60s ./internal/server/

# Run with minimum time
go test -fuzz=FuzzValidateURL -fuzzminimizetime=10s ./internal/validation/
```

### Advanced Fuzzing Options

```bash
# Generate and save interesting inputs
go test -fuzz=FuzzLoadConfig -fuzztime=300s ./internal/config/

# Run with custom worker count
go test -fuzz=FuzzBuildPipelineInput -parallel=8 ./internal/build/

# Run with verbose output
go test -fuzz=FuzzErrorParser -v ./internal/errors/
```

### Continuous Integration

The fuzzing tests are integrated into the CI pipeline:

```yaml
# .github/workflows/fuzz.yml
name: Fuzzing Tests
on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

jobs:
  fuzz:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        package:
          - ./internal/scanner/
          - ./internal/config/
          - ./internal/server/
          - ./internal/validation/
          - ./internal/build/
          - ./internal/errors/
          - ./internal/registry/
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      - name: Run fuzzing tests
        run: |
          go test -fuzz=. -fuzztime=30s ${{ matrix.package }}
```

## Security Patterns Tested

### 1. Command Injection Prevention

All fuzzing tests validate against shell metacharacters:
- `;` (command separator)
- `&` (background execution)
- `|` (pipe operator)
- `` ` `` (command substitution)
- `$` (variable expansion)
- `(`, `)` (subshell execution)
- `<`, `>` (redirection)
- `"`, `'` (quote characters)
- `\` (escape character)
- `\n`, `\r` (line terminators)

### 2. Path Traversal Prevention

Tests validate against various path traversal patterns:
- Basic: `../`, `..\\`
- URL-encoded: `%2e%2e/`, `%2E%2E/`
- Double-encoded: `%252e%252e/`
- Mixed encoding: `..%2f`, `..%2F`
- Unicode variants: `%c0%af`
- Alternative patterns: `....//`

### 3. Cross-Site Scripting (XSS) Prevention

Tests check for common XSS patterns:
- Script tags: `<script>`, `</script>`
- JavaScript URLs: `javascript:`
- Data URLs: `data:text/html`
- Event handlers: `onload=`, `onclick=`
- HTML entities and encoding bypasses

### 4. Control Character Filtering

All text inputs are validated against control characters:
- Null bytes: `\x00`
- Bell character: `\x07`
- Backspace: `\x08`
- Line feed: `\x0a`
- Carriage return: `\x0d`
- Other control characters: `\x01-\x1f`

### 5. Input Size Limits

Fuzzing tests include size limits to prevent resource exhaustion:
- Configuration files: 50KB limit
- Component content: 200KB limit
- URL paths: 10KB limit
- WebSocket messages: Message size limit enforced
- Error outputs: 50KB limit

## Best Practices

### 1. Fuzzing Duration

- **Development**: 30-60 seconds per test for quick validation
- **CI Pipeline**: 30 seconds per test to balance coverage and execution time
- **Security Testing**: 5-10 minutes per test for comprehensive coverage
- **Release Testing**: 30+ minutes per test for thorough validation

### 2. Seed Selection

Each fuzzing test includes carefully chosen seeds:
- Valid inputs that should pass validation
- Known attack patterns that should be rejected
- Edge cases and boundary conditions
- Previously discovered vulnerabilities

### 3. Parallel Execution

- Use `-parallel` flag to control worker count
- Recommended: 1-2 workers per CPU core
- Monitor system resources during fuzzing
- Adjust based on available memory and disk space

### 4. Corpus Management

- Fuzzing corpus is automatically generated and stored
- Review interesting inputs found by fuzzer
- Add significant findings to seed corpus
- Clean up corpus periodically to prevent bloat

## Interpreting Results

### Successful Fuzzing

```
fuzz: elapsed: 30s, gathering baseline coverage: 0/192 completed
fuzz: elapsed: 33s, execs: 25123 (833/sec), new interesting: 12 (total: 204)
fuzz: elapsed: 36s, execs: 47445 (1574/sec), new interesting: 18 (total: 222)
PASS
```

### Fuzzing Failure

```
fuzz: elapsed: 5s, execs: 12456 (2491/sec), new interesting: 3 (total: 45)
--- FAIL: FuzzValidateURL (5.23s)
    --- FAIL: FuzzValidateURL/seed#1 (0.00s)
        validation_fuzz_test.go:45: ValidateURL passed for dangerous protocol: "javascript:alert('xss')"
FAIL
```

## Maintenance

### Regular Updates

1. **Review and update seed corpus** based on new attack patterns
2. **Add new fuzzing tests** for new features and components
3. **Increase fuzzing duration** for release testing
4. **Monitor security advisories** and add relevant test cases

### Performance Monitoring

1. **Track fuzzing execution speed** and optimize slow tests
2. **Monitor memory usage** during fuzzing
3. **Adjust size limits** based on system capabilities
4. **Profile fuzzing performance** to identify bottlenecks

### Security Integration

1. **Integrate with security scanning tools** (e.g., gosec, govulncheck)
2. **Add fuzzing to security review process**
3. **Document security findings** and mitigations
4. **Share fuzzing corpus** with security team

## Conclusion

The comprehensive fuzzing test suite provides robust validation of Templar's security hardening measures. By testing all major input vectors with malicious and malformed data, we ensure the system remains secure against a wide range of attack patterns.

Regular execution of these tests, combined with proper monitoring and maintenance, helps maintain a strong security posture throughout the development lifecycle.