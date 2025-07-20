#!/bin/bash

# Property-Based Testing Script for Templar
# This script runs comprehensive property-based tests, fuzz tests, and visual regression tests

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
GOLDEN_DIR="${PROJECT_ROOT}/testdata/golden"
TEST_TIMEOUT="${TEST_TIMEOUT:-10m}"
FUZZ_TIME="${FUZZ_TIME:-30s}"
PROPERTY_TEST_RUNS="${PROPERTY_TEST_RUNS:-1000}"

# Create directories
mkdir -p "${COVERAGE_DIR}"
mkdir -p "${GOLDEN_DIR}"

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to run property-based tests
run_property_tests() {
    log "Running property-based tests..."
    
    # Set environment variables for property testing
    export GOPTER_RUNS="${PROPERTY_TEST_RUNS}"
    
    # Run property tests with specific build tag
    if go test -v -tags=property -timeout="${TEST_TIMEOUT}" ./internal/scanner ./internal/config 2>&1 | tee "${COVERAGE_DIR}/property-tests.log"; then
        success "Property-based tests completed successfully"
    else
        error "Property-based tests failed"
        return 1
    fi
}

# Function to run fuzz tests
run_fuzz_tests() {
    log "Running fuzz tests..."
    
    # Find all fuzz test functions
    local fuzz_funcs
    fuzz_funcs=$(grep -r "func Fuzz" --include="*_test.go" . | cut -d: -f3 | cut -d'(' -f1 | sort -u || true)
    
    if [ -z "$fuzz_funcs" ]; then
        warning "No fuzz tests found"
        return 0
    fi
    
    local failed=0
    
    # Run each fuzz test
    while IFS= read -r func; do
        if [ -n "$func" ]; then
            log "Running fuzz test: $func"
            
            # Extract package from grep result
            local pkg
            pkg=$(grep -r "$func" --include="*_test.go" . | head -1 | cut -d: -f1 | xargs dirname)
            
            if timeout "${TEST_TIMEOUT}" go test -fuzz="$func" -fuzztime="${FUZZ_TIME}" "$pkg" 2>&1 | tee -a "${COVERAGE_DIR}/fuzz-tests.log"; then
                success "Fuzz test $func completed"
            else
                error "Fuzz test $func failed"
                failed=$((failed + 1))
            fi
        fi
    done <<< "$fuzz_funcs"
    
    if [ $failed -gt 0 ]; then
        error "$failed fuzz tests failed"
        return 1
    else
        success "All fuzz tests completed successfully"
    fi
}

# Function to run visual regression tests
run_visual_tests() {
    log "Running visual regression tests..."
    
    # Check if we should update golden files
    local update_flag=""
    if [ "${UPDATE_GOLDEN:-}" = "true" ]; then
        update_flag="-args -update-golden=true"
        export UPDATE_GOLDEN=true
    fi
    
    # Run visual regression tests
    if go test -v -tags=visual -timeout="${TEST_TIMEOUT}" ./internal/testing $update_flag 2>&1 | tee "${COVERAGE_DIR}/visual-tests.log"; then
        success "Visual regression tests completed successfully"
    else
        error "Visual regression tests failed"
        return 1
    fi
}

# Function to generate coverage report
generate_coverage_report() {
    log "Generating comprehensive coverage report..."
    
    # Run tests with coverage
    go test -coverprofile="${COVERAGE_DIR}/coverage.out" -coverpkg=./... ./...
    
    # Generate HTML coverage report
    go tool cover -html="${COVERAGE_DIR}/coverage.out" -o "${COVERAGE_DIR}/coverage.html"
    
    # Generate coverage analysis
    if go run ./internal/testing/cmd/coverage-analyzer -project="${PROJECT_ROOT}" -output="${COVERAGE_DIR}/analysis.json"; then
        success "Coverage analysis generated"
    else
        warning "Coverage analysis failed (analyzer may not be available)"
    fi
    
    # Extract coverage percentage
    local coverage
    coverage=$(go tool cover -func="${COVERAGE_DIR}/coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
    
    log "Overall test coverage: ${coverage}%"
    
    # Set coverage threshold
    local threshold=${COVERAGE_THRESHOLD:-70}
    
    if (( $(echo "$coverage >= $threshold" | bc -l) )); then
        success "Coverage target achieved (${coverage}% >= ${threshold}%)"
    else
        warning "Coverage below threshold (${coverage}% < ${threshold}%)"
    fi
}

# Function to run benchmark tests
run_benchmarks() {
    log "Running benchmark tests..."
    
    # Run benchmarks
    if go test -bench=. -benchmem -run=^$ ./... 2>&1 | tee "${COVERAGE_DIR}/benchmarks.log"; then
        success "Benchmark tests completed"
    else
        warning "Some benchmark tests failed"
    fi
}

# Function to validate test quality
validate_test_quality() {
    log "Validating test quality..."
    
    local issues=0
    
    # Check for test files without assertions
    log "Checking for tests without assertions..."
    while IFS= read -r -d '' test_file; do
        if ! grep -q "assert\|expect\|Error\|Fatal" "$test_file"; then
            warning "Test file may lack assertions: $test_file"
            issues=$((issues + 1))
        fi
    done < <(find . -name "*_test.go" -print0)
    
    # Check for large test functions (potential code smell)
    log "Checking for large test functions..."
    while IFS= read -r line; do
        local file func_name line_count
        file=$(echo "$line" | cut -d: -f1)
        func_name=$(echo "$line" | cut -d: -f2)
        line_count=$(echo "$line" | cut -d: -f3)
        
        if [ "$line_count" -gt 50 ]; then
            warning "Large test function detected: $func_name in $file ($line_count lines)"
            issues=$((issues + 1))
        fi
    done < <(grep -r "^func Test" --include="*_test.go" . | while read -r match; do
        local file func_name start_line end_line
        file=$(echo "$match" | cut -d: -f1)
        func_name=$(echo "$match" | cut -d: -f3 | cut -d'(' -f1)
        start_line=$(echo "$match" | cut -d: -f2)
        
        # Find end of function (simple heuristic)
        end_line=$(awk -v start="$start_line" '
            NR >= start && /^}$/ { print NR; exit }
            NR >= start + 200 { print NR; exit }
        ' "$file")
        
        if [ -n "$end_line" ]; then
            echo "$file:$func_name:$((end_line - start_line))"
        fi
    done)
    
    if [ $issues -eq 0 ]; then
        success "Test quality validation passed"
    else
        warning "Test quality issues found: $issues"
    fi
}

# Function to generate final report
generate_final_report() {
    log "Generating final test report..."
    
    local report_file="${COVERAGE_DIR}/test-report.md"
    
    cat > "$report_file" << EOF
# Templar Advanced Testing Report

Generated on: $(date)

## Test Summary

### Property-Based Tests
$(if [ -f "${COVERAGE_DIR}/property-tests.log" ]; then
    grep -c "PASS\|FAIL" "${COVERAGE_DIR}/property-tests.log" || echo "No results"
else
    echo "Not run"
fi)

### Fuzz Tests  
$(if [ -f "${COVERAGE_DIR}/fuzz-tests.log" ]; then
    grep -c "PASS\|FAIL" "${COVERAGE_DIR}/fuzz-tests.log" || echo "No results"
else
    echo "Not run"
fi)

### Visual Regression Tests
$(if [ -f "${COVERAGE_DIR}/visual-tests.log" ]; then
    grep -c "PASS\|FAIL" "${COVERAGE_DIR}/visual-tests.log" || echo "No results"
else
    echo "Not run"
fi)

### Coverage
$(if [ -f "${COVERAGE_DIR}/coverage.out" ]; then
    go tool cover -func="${COVERAGE_DIR}/coverage.out" | grep total || echo "Coverage data unavailable"
else
    echo "Coverage not generated"
fi)

## Files Generated

- Coverage Report: [coverage.html](./coverage.html)
- Property Tests Log: [property-tests.log](./property-tests.log)  
- Fuzz Tests Log: [fuzz-tests.log](./fuzz-tests.log)
- Visual Tests Log: [visual-tests.log](./visual-tests.log)
- Benchmarks: [benchmarks.log](./benchmarks.log)

## Next Steps

1. Review failed tests in the log files
2. Address coverage gaps identified in the analysis
3. Add tests for uncovered critical functions
4. Consider adding more property-based tests for complex algorithms

EOF

    success "Test report generated: $report_file"
}

# Main execution
main() {
    log "Starting advanced testing suite for Templar"
    log "Project root: $PROJECT_ROOT"
    
    local failed=0
    
    # Validate environment
    if ! command -v go >/dev/null; then
        error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if bc is available for floating point comparison
    if ! command -v bc >/dev/null; then
        warning "bc not available, coverage comparison may not work"
    fi
    
    cd "$PROJECT_ROOT"
    
    # Run test suites
    if [ "${SKIP_PROPERTY:-}" != "true" ]; then
        run_property_tests || failed=$((failed + 1))
    fi
    
    if [ "${SKIP_FUZZ:-}" != "true" ]; then
        run_fuzz_tests || failed=$((failed + 1))
    fi
    
    if [ "${SKIP_VISUAL:-}" != "true" ]; then
        run_visual_tests || failed=$((failed + 1))
    fi
    
    if [ "${SKIP_BENCHMARKS:-}" != "true" ]; then
        run_benchmarks || failed=$((failed + 1))
    fi
    
    # Always generate coverage report
    generate_coverage_report
    
    # Validate test quality
    validate_test_quality
    
    # Generate final report
    generate_final_report
    
    # Summary
    if [ $failed -eq 0 ]; then
        success "All advanced tests completed successfully!"
        log "Results available in: $COVERAGE_DIR"
    else
        error "$failed test suites failed"
        log "Check logs in: $COVERAGE_DIR"
        exit 1
    fi
}

# Run main function
main "$@"