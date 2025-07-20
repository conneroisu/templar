# Makefile for Templar project

.PHONY: all test test-unit test-integration test-security test-coverage test-verbose clean build run help docker test-bench test-e2e fuzz fuzz-short fuzz-long fuzz-security test-property test-visual test-advanced coverage-analysis

# Default target
all: test build

# Run all tests
test:
	go test ./...

# Run only unit tests (exclude integration and security tests)
test-unit:
	go test ./... -short

# Run only integration tests
test-integration:
	go test -v -tags=integration ./integration_tests/... -timeout=30m

# Run security tests
test-security:
	go test -v -tags=security ./cmd/... -run "TestSecurity" -timeout=10m
	go test -v -tags=security ./internal/server/... -run "TestSecurity" -timeout=10m
	go test -v -tags=security ./internal/config/... -run "TestSecurity" -timeout=10m

# Run E2E tests
test-e2e:
	go test -v -tags=integration ./integration_tests/... -run "TestE2E" -timeout=45m

# Run performance benchmarks
test-bench:
	go test -bench=BenchmarkComponentScanner -benchmem -benchtime=5s ./internal/scanner/...
	go test -bench=BenchmarkBuildPipeline -benchmem -benchtime=5s ./internal/build/...
	go test -bench=BenchmarkWebSocket -benchmem -benchtime=5s ./internal/server/...
	go test -bench=BenchmarkFileWatcher -benchmem -benchtime=5s ./internal/watcher/...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated in coverage.html"

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with race detection
test-race:
	go test -race ./...

# Run tests and generate coverage with race detection
test-full:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run comprehensive test suite (like CI)
test-ci: generate
	@echo "Running comprehensive test suite..."
	go test ./... -short -race -coverprofile=unit-coverage.out
	go test -v -tags=security ./cmd/... -run "TestSecurity" -coverprofile=security-coverage.out
	go test -v -tags=integration ./integration_tests/... -timeout=30m
	@echo "All tests completed successfully!"

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html *-coverage.out *-bench.txt
	go clean -testcache

# Build the project
build:
	go build -o templar .

# Build for production (static binary)
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
		-ldflags '-extldflags "-static" -s -w' \
		-o templar .

# Docker targets
docker-build:
	docker build -t templar:latest .

docker-run:
	docker run -p 8080:8080 templar:latest

docker-test:
	docker build -t templar:test --target builder .
	docker run --rm templar:test go test ./...

# Security scanning
security-scan:
	@command -v govulncheck >/dev/null 2>&1 || { echo "Installing govulncheck..."; go install golang.org/x/vuln/cmd/govulncheck@latest; }
	govulncheck ./...

# Run the project
run:
	go run main.go

# Advanced Testing Framework Targets

# Run property-based tests
test-property:
	@echo "🧪 Running property-based tests..."
	go test -v -tags=property ./... -timeout=10m

# Run comprehensive advanced testing suite
test-advanced:
	@echo "🚀 Running comprehensive advanced testing framework..."
	./scripts/advanced-testing.sh

# Run only baseline tests with advanced analysis
test-advanced-baseline:
	@echo "📊 Running baseline tests with advanced analysis..."
	./scripts/advanced-testing.sh --baseline

# Run property-based tests with advanced framework
test-advanced-property:
	@echo "🔬 Running property-based tests with advanced analysis..."
	./scripts/advanced-testing.sh --property

# Run mutation testing
test-mutation:
	@echo "🧬 Running mutation testing..."
	./scripts/advanced-testing.sh --mutation

# Run behavioral coverage analysis
test-behavioral:
	@echo "🧠 Running behavioral coverage analysis..."
	./scripts/advanced-testing.sh --behavioral

# Run advanced fuzz testing
test-advanced-fuzz:
	@echo "⚡ Running advanced fuzz testing..."
	./scripts/advanced-testing.sh --fuzz

# Generate comprehensive coverage analysis
coverage-advanced:
	@echo "📈 Generating advanced coverage analysis..."
	./scripts/advanced-testing.sh --behavioral
	@echo "Advanced coverage report available in reports/"

# Run visual regression tests  
test-visual:
	@echo "🎨 Running visual regression tests..."
	go test -v -tags=visual ./internal/testing -timeout=10m

# Update visual regression golden files
test-visual-update:
	@echo "🎨 Updating visual regression golden files..."
	UPDATE_GOLDEN=true go test -v -tags=visual ./internal/testing -timeout=10m

# Run advanced tests for CI
test-advanced-ci:
	@echo "🤖 Running advanced tests for CI..."
	SKIP_VISUAL=true FUZZ_TIME=10s ./scripts/advanced-testing.sh

# Generate coverage analysis report
coverage-analysis:
	@echo "📊 Generating coverage analysis..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -coverpkg=./... ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

# Run only fuzz tests with custom duration
test-fuzz-custom:
	@echo "🔍 Running fuzz tests for $(FUZZ_TIME)..."
	FUZZ_TIME=$(or $(FUZZ_TIME),30s) ./scripts/run-property-tests.sh

# Performance testing with property-based approach
test-performance-property:
	@echo "⚡ Running performance property tests..."
	go test -v -tags=property -bench=Property ./internal/build ./internal/scanner -timeout=20m

# CLI command shortcuts
init:
	go run main.go init

serve:
	go run main.go serve

list:
	go run main.go list

build-components:
	go run main.go build

watch:
	go run main.go watch

preview:
	go run main.go preview $(COMPONENT)

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed. Please install it first."; exit 1; }
	golangci-lint run

# Generate code (if needed)
generate:
	go generate ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Fuzzing targets
fuzz: fuzz-short

# Run fuzzing tests for 30 seconds each
fuzz-short:
	@echo "Running short fuzzing tests (30s each)..."
	@echo "Fuzzing scanner..."
	@go test -fuzz=FuzzScanFile -fuzztime=30s ./internal/scanner/ || true
	@go test -fuzz=FuzzParseTemplComponent -fuzztime=30s ./internal/scanner/ || true
	@echo "Fuzzing configuration..."
	@go test -fuzz=FuzzLoadConfig -fuzztime=30s ./internal/config/ || true
	@go test -fuzz=FuzzConfigValidation -fuzztime=30s ./internal/config/ || true
	@echo "Fuzzing WebSocket..."
	@go test -fuzz=FuzzWebSocketOriginValidation -fuzztime=30s ./internal/server/ || true
	@go test -fuzz=FuzzWebSocketMessage -fuzztime=30s ./internal/server/ || true
	@echo "Fuzzing validation..."
	@go test -fuzz=FuzzValidateURL -fuzztime=30s ./internal/validation/ || true
	@go test -fuzz=FuzzPathTraversal -fuzztime=30s ./internal/validation/ || true
	@echo "Fuzzing build pipeline..."
	@go test -fuzz=FuzzBuildPipelineInput -fuzztime=30s ./internal/build/ || true
	@go test -fuzz=FuzzCompilerCommand -fuzztime=30s ./internal/build/ || true
	@echo "Fuzzing error handling..."
	@go test -fuzz=FuzzErrorParser -fuzztime=30s ./internal/errors/ || true
	@go test -fuzz=FuzzHTMLErrorOverlay -fuzztime=30s ./internal/errors/ || true
	@echo "Fuzzing registry..."
	@go test -fuzz=FuzzComponentRegistration -fuzztime=30s ./internal/registry/ || true
	@go test -fuzz=FuzzComponentSearch -fuzztime=30s ./internal/registry/ || true

# Run fuzzing tests for 5 minutes each
fuzz-long:
	@echo "Running long fuzzing tests (5m each)..."
	@echo "Fuzzing scanner..."
	@go test -fuzz=FuzzScanFile -fuzztime=300s ./internal/scanner/ || true
	@go test -fuzz=FuzzParseTemplComponent -fuzztime=300s ./internal/scanner/ || true
	@echo "Fuzzing configuration..."
	@go test -fuzz=FuzzLoadConfig -fuzztime=300s ./internal/config/ || true
	@go test -fuzz=FuzzConfigValidation -fuzztime=300s ./internal/config/ || true
	@echo "Fuzzing WebSocket..."
	@go test -fuzz=FuzzWebSocketOriginValidation -fuzztime=300s ./internal/server/ || true
	@go test -fuzz=FuzzWebSocketMessage -fuzztime=300s ./internal/server/ || true
	@echo "Fuzzing validation..."
	@go test -fuzz=FuzzValidateURL -fuzztime=300s ./internal/validation/ || true
	@go test -fuzz=FuzzPathTraversal -fuzztime=300s ./internal/validation/ || true
	@echo "Fuzzing build pipeline..."
	@go test -fuzz=FuzzBuildPipelineInput -fuzztime=300s ./internal/build/ || true
	@go test -fuzz=FuzzCompilerCommand -fuzztime=300s ./internal/build/ || true
	@echo "Fuzzing error handling..."
	@go test -fuzz=FuzzErrorParser -fuzztime=300s ./internal/errors/ || true
	@go test -fuzz=FuzzHTMLErrorOverlay -fuzztime=300s ./internal/errors/ || true
	@echo "Fuzzing registry..."
	@go test -fuzz=FuzzComponentRegistration -fuzztime=300s ./internal/registry/ || true
	@go test -fuzz=FuzzComponentSearch -fuzztime=300s ./internal/registry/ || true

# Run comprehensive security fuzzing (10 minutes each)
fuzz-security:
	@echo "Running comprehensive security fuzzing (10m each)..."
	@echo "Fuzzing scanner..."
	@go test -fuzz=FuzzScanFile -fuzztime=600s ./internal/scanner/ || true
	@go test -fuzz=FuzzParseTemplComponent -fuzztime=600s ./internal/scanner/ || true
	@echo "Fuzzing configuration..."
	@go test -fuzz=FuzzLoadConfig -fuzztime=600s ./internal/config/ || true
	@go test -fuzz=FuzzConfigValidation -fuzztime=600s ./internal/config/ || true
	@echo "Fuzzing WebSocket..."
	@go test -fuzz=FuzzWebSocketOriginValidation -fuzztime=600s ./internal/server/ || true
	@go test -fuzz=FuzzWebSocketMessage -fuzztime=600s ./internal/server/ || true
	@echo "Fuzzing validation..."
	@go test -fuzz=FuzzValidateURL -fuzztime=600s ./internal/validation/ || true
	@go test -fuzz=FuzzPathTraversal -fuzztime=600s ./internal/validation/ || true
	@echo "Fuzzing build pipeline..."
	@go test -fuzz=FuzzBuildPipelineInput -fuzztime=600s ./internal/build/ || true
	@go test -fuzz=FuzzCompilerCommand -fuzztime=600s ./internal/build/ || true
	@echo "Fuzzing error handling..."
	@go test -fuzz=FuzzErrorParser -fuzztime=600s ./internal/errors/ || true
	@go test -fuzz=FuzzHTMLErrorOverlay -fuzztime=600s ./internal/errors/ || true
	@echo "Fuzzing registry..."
	@go test -fuzz=FuzzComponentRegistration -fuzztime=600s ./internal/registry/ || true
	@go test -fuzz=FuzzComponentSearch -fuzztime=600s ./internal/registry/ || true

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Testing:"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-security   - Run security tests"
	@echo "  test-e2e        - Run end-to-end tests"
	@echo "  test-bench      - Run performance benchmarks"
	@echo "  test-ci         - Run comprehensive test suite (like CI)"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  test-race       - Run tests with race detection"
	@echo "  test-full       - Run comprehensive tests with coverage and race detection"
	@echo ""
	@echo "Fuzzing:"
	@echo "  fuzz            - Run fuzzing tests (default: short duration)"
	@echo "  fuzz-short      - Run fuzzing tests for 30 seconds each"
	@echo "  fuzz-long       - Run fuzzing tests for 5 minutes each"
	@echo "  fuzz-security   - Run comprehensive security fuzzing (10 minutes each)"
	@echo ""
	@echo "Building:"
	@echo "  build           - Build the project"
	@echo "  build-prod      - Build production binary (static)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  docker-test     - Run tests in Docker"
	@echo ""
	@echo "Security:"
	@echo "  security-scan   - Run vulnerability scanning"
	@echo ""
	@echo "Development:"
	@echo "  run             - Run the project"
	@echo "  deps            - Install dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  generate        - Generate code"
	@echo "  clean           - Clean test artifacts"
	@echo "  help            - Show this help"

# Development shortcuts
dev-setup: deps generate fmt

# CI/CD target
ci: clean deps generate fmt lint test-ci security-scan fuzz-short

# Pre-commit checks
pre-commit: fmt lint test-race test-security