#!/bin/bash

# Parallel Testing Script for Templar
# Optimizes test execution through intelligent parallelization and resource management

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly REPORTS_DIR="${PROJECT_ROOT}/reports"
readonly TEST_CACHE_DIR="${PROJECT_ROOT}/.test-cache"

# Test execution parameters
readonly CPU_COUNT=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "4")
readonly PARALLEL_JOBS=$((CPU_COUNT > 8 ? 8 : CPU_COUNT))
readonly FUZZ_TIME="${FUZZ_TIME:-30s}"
readonly TEST_TIMEOUT="${TEST_TIMEOUT:-30m}"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Test categories for parallel execution
declare -A TEST_CATEGORIES=(
    ["unit"]="./... -short"
    ["integration"]="./tests/integration/... -timeout=${TEST_TIMEOUT}"
    ["property"]="./... -tags=property -timeout=15m"
    ["security"]="./cmd/... ./internal/server/... ./internal/config/... -tags=security -timeout=10m"
    ["e2e"]="./tests/e2e/... -timeout=45m"
    ["performance"]="./internal/performance/... -bench=. -benchmem -timeout=20m"
)

# Fuzz test targets for parallel execution
declare -A FUZZ_TARGETS=(
    ["scanner"]="./internal/scanner/ -fuzz=FuzzScanFile,FuzzParseTemplComponent"
    ["config"]="./internal/config/ -fuzz=FuzzLoadConfig,FuzzConfigValidation"
    ["server"]="./internal/server/ -fuzz=FuzzWebSocketOriginValidation,FuzzWebSocketMessage"
    ["validation"]="./internal/validation/ -fuzz=FuzzValidateURL,FuzzPathTraversal"
    ["build"]="./internal/build/ -fuzz=FuzzBuildPipelineInput,FuzzCompilerCommand"
    ["errors"]="./internal/errors/ -fuzz=FuzzErrorParser,FuzzHTMLErrorOverlay"
    ["registry"]="./internal/registry/ -fuzz=FuzzComponentRegistration,FuzzComponentSearch"
)

# Initialize directories
init_directories() {
    mkdir -p "${REPORTS_DIR}"
    mkdir -p "${TEST_CACHE_DIR}"
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Progress tracking
show_progress() {
    local current=$1
    local total=$2
    local category=$3
    local percent=$((current * 100 / total))
    printf "\r${BLUE}[%3d%%]${NC} Running %s tests (%d/%d)" "$percent" "$category" "$current" "$total"
}

# Test cache management
is_test_cached() {
    local test_key=$1
    local cache_file="${TEST_CACHE_DIR}/${test_key}.cache"
    local source_modified=$(find "${PROJECT_ROOT}" -name "*.go" -newer "$cache_file" 2>/dev/null | wc -l)
    
    [[ -f "$cache_file" && $source_modified -eq 0 ]]
}

cache_test_result() {
    local test_key=$1
    local result=$2
    local cache_file="${TEST_CACHE_DIR}/${test_key}.cache"
    
    echo "$result" > "$cache_file"
    touch "$cache_file"
}

# Smart test selection based on changed files
get_affected_packages() {
    if command -v git >/dev/null 2>&1; then
        local changed_files=$(git diff --name-only HEAD~1 2>/dev/null | grep '\.go$' || true)
        if [[ -n "$changed_files" ]]; then
            echo "$changed_files" | xargs -I {} dirname {} | sort -u | while read -r dir; do
                if [[ -d "$dir" ]]; then
                    echo "./$dir/..."
                fi
            done | tr '\n' ' '
        else
            echo "./..."
        fi
    else
        echo "./..."
    fi
}

# Resource monitoring
check_system_resources() {
    local available_memory
    local cpu_load
    
    # Check available memory (Linux/macOS compatible)
    if command -v free >/dev/null 2>&1; then
        available_memory=$(free -m | awk 'NR==2{printf "%.1f", $7/$2*100}')
    elif command -v vm_stat >/dev/null 2>&1; then
        available_memory="75.0"  # Estimate for macOS
    else
        available_memory="50.0"  # Conservative estimate
    fi
    
    # Check CPU load (Linux/macOS compatible)
    if command -v uptime >/dev/null 2>&1; then
        cpu_load=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
    else
        cpu_load="1.0"
    fi
    
    log_info "System resources: ${available_memory}% memory available, CPU load: ${cpu_load}"
    
    # Adjust parallelism based on resources
    if (( $(echo "$available_memory < 30" | bc -l 2>/dev/null || echo "0") )); then
        log_warning "Low memory detected, reducing parallel jobs"
        echo $((PARALLEL_JOBS / 2))
    elif (( $(echo "$cpu_load > $(nproc)" | bc -l 2>/dev/null || echo "0") )); then
        log_warning "High CPU load detected, reducing parallel jobs"
        echo $((PARALLEL_JOBS * 3 / 4))
    else
        echo "$PARALLEL_JOBS"
    fi
}

# Execute tests in parallel with resource management
run_parallel_tests() {
    local test_type=$1
    local max_jobs=$(check_system_resources)
    local pids=()
    local results=()
    local job_count=0
    
    log_info "Running $test_type tests with $max_jobs parallel jobs"
    
    case "$test_type" in
        "unit"|"integration"|"property"|"security"|"e2e"|"performance")
            local test_args=${TEST_CATEGORIES[$test_type]}
            run_test_category "$test_type" "$test_args" &
            pids+=($!)
            ;;
        "fuzz")
            for target in "${!FUZZ_TARGETS[@]}"; do
                if [[ $job_count -ge $max_jobs ]]; then
                    wait_for_job pids[@]
                    job_count=$((job_count - 1))
                fi
                
                run_fuzz_target "$target" "${FUZZ_TARGETS[$target]}" &
                pids+=($!)
                job_count=$((job_count + 1))
            done
            ;;
        "all")
            for category in "${!TEST_CATEGORIES[@]}"; do
                if [[ $job_count -ge $max_jobs ]]; then
                    wait_for_job pids[@]
                    job_count=$((job_count - 1))
                fi
                
                run_test_category "$category" "${TEST_CATEGORIES[$category]}" &
                pids+=($!)
                job_count=$((job_count + 1))
            done
            ;;
    esac
    
    # Wait for all jobs to complete
    local failed_jobs=0
    for pid in "${pids[@]}"; do
        if wait "$pid"; then
            results+=("success")
        else
            results+=("failed")
            failed_jobs=$((failed_jobs + 1))
        fi
    done
    
    return $failed_jobs
}

# Wait for the first job to complete
wait_for_job() {
    local -n pids_ref=$1
    local completed_pid
    
    if [[ ${#pids_ref[@]} -gt 0 ]]; then
        wait -n "${pids_ref[@]}"
        completed_pid=$!
        
        # Remove completed PID from array
        local new_pids=()
        for pid in "${pids_ref[@]}"; do
            if [[ $pid -ne $completed_pid ]]; then
                new_pids+=("$pid")
            fi
        done
        pids_ref=("${new_pids[@]}")
    fi
}

# Run a specific test category
run_test_category() {
    local category=$1
    local test_args=$2
    local cache_key="${category}_$(echo "$test_args" | md5sum | cut -d' ' -f1 2>/dev/null || echo "default")"
    local output_file="${REPORTS_DIR}/${category}_test_output.txt"
    
    if is_test_cached "$cache_key" && [[ "${FORCE_REBUILD:-false}" != "true" ]]; then
        log_info "Using cached results for $category tests"
        return 0
    fi
    
    log_info "Running $category tests..."
    
    local start_time=$(date +%s)
    local exit_code=0
    
    # Execute test with proper flags and output capture
    case "$category" in
        "unit")
            if go test $test_args -race -coverprofile="${REPORTS_DIR}/${category}_coverage.out" > "$output_file" 2>&1; then
                exit_code=0
            else
                exit_code=1
            fi
            ;;
        "performance")
            if go test $test_args -benchtime=5s > "$output_file" 2>&1; then
                exit_code=0
            else
                exit_code=1
            fi
            ;;
        "property")
            if go test $test_args -v > "$output_file" 2>&1; then
                exit_code=0
            else
                exit_code=1
            fi
            ;;
        *)
            if go test $test_args -v > "$output_file" 2>&1; then
                exit_code=0
            else
                exit_code=1
            fi
            ;;
    esac
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "$category tests completed in ${duration}s"
        cache_test_result "$cache_key" "success"
    else
        log_error "$category tests failed in ${duration}s"
        log_error "Output available in: $output_file"
        cache_test_result "$cache_key" "failed"
    fi
    
    return $exit_code
}

# Run fuzz tests for a specific target
run_fuzz_target() {
    local target=$1
    local fuzz_args=$2
    local output_file="${REPORTS_DIR}/fuzz_${target}_output.txt"
    
    log_info "Running fuzz tests for $target..."
    
    local start_time=$(date +%s)
    local exit_code=0
    
    # Parse and run each fuzz test
    local package_path=$(echo "$fuzz_args" | cut -d' ' -f1)
    local fuzz_functions=$(echo "$fuzz_args" | cut -d' ' -f2- | tr ',' ' ')
    
    for fuzz_func in $fuzz_functions; do
        if ! go test "$package_path" -fuzz="$fuzz_func" -fuzztime="$FUZZ_TIME" >> "$output_file" 2>&1; then
            exit_code=1
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "Fuzz tests for $target completed in ${duration}s"
    else
        log_error "Fuzz tests for $target failed in ${duration}s"
        log_error "Output available in: $output_file"
    fi
    
    return $exit_code
}

# Generate comprehensive test report
generate_test_report() {
    local report_file="${REPORTS_DIR}/parallel_test_report.html"
    
    log_info "Generating comprehensive test report..."
    
    cat > "$report_file" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Templar Parallel Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 10px; border-radius: 5px; }
        .success { color: green; }
        .failed { color: red; }
        .warning { color: orange; }
        .section { margin: 20px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .code { background: #f8f8f8; padding: 10px; font-family: monospace; white-space: pre-wrap; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Templar Parallel Test Report</h1>
        <p>Generated: $(date)</p>
        <p>System: $(uname -a)</p>
        <p>Go Version: $(go version)</p>
    </div>
EOF

    # Add test results summary
    echo '    <div class="section">' >> "$report_file"
    echo '        <h2>Test Results Summary</h2>' >> "$report_file"
    echo '        <table>' >> "$report_file"
    echo '            <tr><th>Test Category</th><th>Status</th><th>Output File</th></tr>' >> "$report_file"
    
    for category in "${!TEST_CATEGORIES[@]}"; do
        local output_file="${REPORTS_DIR}/${category}_test_output.txt"
        local status="N/A"
        
        if [[ -f "$output_file" ]]; then
            if grep -q "PASS" "$output_file" && ! grep -q "FAIL" "$output_file"; then
                status='<span class="success">PASS</span>'
            else
                status='<span class="failed">FAIL</span>'
            fi
        fi
        
        echo "            <tr><td>$category</td><td>$status</td><td>$output_file</td></tr>" >> "$report_file"
    done
    
    echo '        </table>' >> "$report_file"
    echo '    </div>' >> "$report_file"

    # Add coverage information if available
    local coverage_files=$(find "${REPORTS_DIR}" -name "*_coverage.out" 2>/dev/null)
    if [[ -n "$coverage_files" ]]; then
        echo '    <div class="section">' >> "$report_file"
        echo '        <h2>Coverage Information</h2>' >> "$report_file"
        
        for coverage_file in $coverage_files; do
            local category=$(basename "$coverage_file" _coverage.out)
            local coverage_percent=$(go tool cover -func="$coverage_file" 2>/dev/null | tail -1 | awk '{print $3}' || echo "N/A")
            echo "        <p>$category: $coverage_percent</p>" >> "$report_file"
        done
        
        echo '    </div>' >> "$report_file"
    fi

    echo '</body></html>' >> "$report_file"
    
    log_success "Test report generated: $report_file"
}

# Performance optimization suggestions
suggest_optimizations() {
    log_info "Analyzing test performance and suggesting optimizations..."
    
    local slow_tests=$(find "${REPORTS_DIR}" -name "*_output.txt" -exec grep -l "slow" {} \; 2>/dev/null || true)
    local memory_intensive=$(find "${REPORTS_DIR}" -name "*_output.txt" -exec grep -l "memory" {} \; 2>/dev/null || true)
    
    if [[ -n "$slow_tests" ]]; then
        log_warning "Slow tests detected. Consider:"
        log_warning "  - Using test parallelization with t.Parallel()"
        log_warning "  - Reducing test data size"
        log_warning "  - Optimizing test setup/teardown"
    fi
    
    if [[ -n "$memory_intensive" ]]; then
        log_warning "Memory-intensive tests detected. Consider:"
        log_warning "  - Using object pooling in tests"
        log_warning "  - Reducing concurrent test execution"
        log_warning "  - Adding memory cleanup in test teardown"
    fi
}

# Clean up test artifacts
cleanup() {
    log_info "Cleaning up test artifacts..."
    
    # Clean old cache files (older than 1 day)
    find "${TEST_CACHE_DIR}" -name "*.cache" -mtime +1 -delete 2>/dev/null || true
    
    # Compress old log files
    find "${REPORTS_DIR}" -name "*.txt" -mtime +7 -exec gzip {} \; 2>/dev/null || true
    
    log_success "Cleanup completed"
}

# Main execution function
main() {
    local test_type="${1:-all}"
    local start_time=$(date +%s)
    
    log_info "Starting parallel test execution for: $test_type"
    log_info "Using $PARALLEL_JOBS parallel jobs on $CPU_COUNT CPU cores"
    
    init_directories
    
    # Execute tests
    local exit_code=0
    if ! run_parallel_tests "$test_type"; then
        exit_code=1
    fi
    
    # Generate reports
    generate_test_report
    suggest_optimizations
    
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "All parallel tests completed successfully in ${total_duration}s"
    else
        log_error "Some tests failed. Total execution time: ${total_duration}s"
    fi
    
    cleanup
    
    return $exit_code
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [TEST_TYPE]

TEST_TYPE options:
  unit         - Run unit tests only
  integration  - Run integration tests only
  property     - Run property-based tests only
  security     - Run security tests only
  e2e          - Run end-to-end tests only
  performance  - Run performance tests only
  fuzz         - Run fuzz tests only
  all          - Run all test categories (default)

Environment variables:
  FUZZ_TIME     - Duration for fuzz tests (default: 30s)
  TEST_TIMEOUT  - Timeout for individual tests (default: 30m)
  FORCE_REBUILD - Force rebuild ignoring cache (default: false)

Examples:
  $0 unit                    # Run only unit tests
  $0 all                     # Run all tests
  FUZZ_TIME=60s $0 fuzz      # Run fuzz tests for 60 seconds each
  FORCE_REBUILD=true $0 all  # Force full rebuild
EOF
}

# Command line argument handling
case "${1:-all}" in
    "unit"|"integration"|"property"|"security"|"e2e"|"performance"|"fuzz"|"all")
        main "$1"
        ;;
    "-h"|"--help"|"help")
        usage
        exit 0
        ;;
    *)
        log_error "Unknown test type: $1"
        usage
        exit 1
        ;;
esac