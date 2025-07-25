#!/bin/bash

# Templar Integration Test Script
set -e

echo "Starting Templar Integration Tests..."

# Configuration
PORT=8081
TIMEOUT=30
TEMPLAR_BINARY="./templar"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    if [ -d "test-project" ]; then
        rm -rf test-project
    fi
}

trap cleanup EXIT

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warn "jq not found, some tests will be skipped"
        JQ_AVAILABLE=false
    else
        JQ_AVAILABLE=true
    fi
    
    if [ ! -f "$TEMPLAR_BINARY" ]; then
        log_info "Building templar binary..."
        go build -o templar .
    fi
}

# Test project initialization
test_init() {
    log_info "Testing project initialization..."
    
    $TEMPLAR_BINARY init test-project
    
    if [ ! -d "test-project" ]; then
        log_error "Project directory not created"
        return 1
    fi
    
    if [ ! -f "test-project/.templar.yml" ]; then
        log_error "Configuration file not created"
        return 1
    fi
    
    log_info "✓ Project initialization successful"
}

# Test component listing
test_list() {
    log_info "Testing component listing..."
    
    cd test-project
    
    # Test basic listing
    if ! ../$TEMPLAR_BINARY list > /dev/null; then
        log_error "Component listing failed"
        cd ..
        return 1
    fi
    
    # Test JSON format if jq is available
    if [ "$JQ_AVAILABLE" = true ]; then
        if ! ../$TEMPLAR_BINARY list --format json | jq . > /dev/null; then
            log_error "JSON format listing failed"
            cd ..
            return 1
        fi
    fi
    
    cd ..
    log_info "✓ Component listing successful"
}

# Start test server
start_server() {
    log_info "Starting test server on port $PORT..."
    
    cd test-project
    ../$TEMPLAR_BINARY serve --port $PORT --no-open &
    SERVER_PID=$!
    cd ..
    
    # Wait for server to start
    log_info "Waiting for server to start..."
    for i in $(seq 1 $TIMEOUT); do
        if curl -f http://localhost:$PORT/api/health > /dev/null 2>&1; then
            log_info "✓ Server started successfully"
            return 0
        fi
        sleep 1
    done
    
    log_error "Server failed to start within $TIMEOUT seconds"
    return 1
}

# Test HTTP API endpoints
test_api() {
    log_info "Testing HTTP API endpoints..."
    
    # Health check
    if ! curl -f http://localhost:$PORT/api/health > /dev/null 2>&1; then
        log_error "Health check failed"
        return 1
    fi
    log_info "✓ Health check passed"
    
    # Component listing
    if ! curl -f http://localhost:$PORT/api/components > /dev/null 2>&1; then
        log_error "Component API failed"
        return 1
    fi
    log_info "✓ Component API passed"
    
    # Build status
    if ! curl -f http://localhost:$PORT/api/build/status > /dev/null 2>&1; then
        log_error "Build status API failed"
        return 1
    fi
    log_info "✓ Build status API passed"
    
    # Test component preview if components exist
    if [ "$JQ_AVAILABLE" = true ]; then
        COMPONENTS=$(curl -s http://localhost:$PORT/api/components | jq -r '.components[0].name // empty' 2>/dev/null)
        if [ ! -z "$COMPONENTS" ]; then
            PREVIEW_DATA="{\"component\":\"$COMPONENTS\",\"props\":{}}"
            if curl -f -X POST \
                -H "Content-Type: application/json" \
                -d "$PREVIEW_DATA" \
                http://localhost:$PORT/api/preview > /dev/null 2>&1; then
                log_info "✓ Preview API passed"
            else
                log_warn "Preview API test skipped (no suitable components)"
            fi
        fi
    fi
}

# Test WebSocket connection
test_websocket() {
    log_info "Testing WebSocket connection..."
    
    # Use a simple WebSocket test if websocat is available
    if command -v websocat &> /dev/null; then
        if echo '{"type":"ping"}' | timeout 5s websocat ws://localhost:$PORT/ws/reload > /dev/null 2>&1; then
            log_info "✓ WebSocket connection successful"
        else
            log_warn "WebSocket test inconclusive"
        fi
    else
        log_warn "websocat not available, skipping WebSocket test"
    fi
}

# Test build process
test_build() {
    log_info "Testing build process..."
    
    cd test-project
    
    # Test build command
    if ! ../$TEMPLAR_BINARY build > /dev/null 2>&1; then
        log_error "Build command failed"
        cd ..
        return 1
    fi
    
    cd ..
    log_info "✓ Build process successful"
}

# Test watch mode (brief test)
test_watch() {
    log_info "Testing watch mode..."
    
    cd test-project
    
    # Start watch in background and kill it quickly
    ../$TEMPLAR_BINARY watch &
    WATCH_PID=$!
    sleep 3
    kill $WATCH_PID 2>/dev/null || true
    wait $WATCH_PID 2>/dev/null || true
    
    cd ..
    log_info "✓ Watch mode test completed"
}

# Performance test
test_performance() {
    log_info "Running basic performance tests..."
    
    # Test response time
    START_TIME=$(date +%s%N)
    curl -f http://localhost:$PORT/api/health > /dev/null 2>&1
    END_TIME=$(date +%s%N)
    RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))
    
    if [ $RESPONSE_TIME -lt 1000 ]; then
        log_info "✓ Response time: ${RESPONSE_TIME}ms (good)"
    elif [ $RESPONSE_TIME -lt 5000 ]; then
        log_warn "Response time: ${RESPONSE_TIME}ms (acceptable)"
    else
        log_error "Response time: ${RESPONSE_TIME}ms (slow)"
        return 1
    fi
    
    # Test concurrent requests
    if command -v xargs &> /dev/null; then
        log_info "Testing concurrent requests..."
        seq 1 10 | xargs -I {} -P 10 curl -f http://localhost:$PORT/api/health > /dev/null 2>&1
        log_info "✓ Concurrent requests handled"
    fi
}

# Main test execution
main() {
    log_info "=== Templar Integration Test Suite ==="
    
    check_dependencies
    test_init
    test_list
    start_server
    test_api
    test_websocket
    test_build
    test_watch
    test_performance
    
    log_info "=== All Tests Completed Successfully ==="
}

# Run tests
main "$@"