# Makefile for Templar project

.PHONY: all test test-unit test-integration test-coverage test-verbose clean build run help

# Default target
all: test build

# Run all tests
test:
	go test ./...

# Run only unit tests (exclude integration tests)
test-unit:
	go test ./... -short

# Run only integration tests
test-integration:
	go test -run Integration

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

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# Build the project
build:
	go build -o templar .

# Run the project
run:
	go run main.go

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

# Show help
help:
	@echo "Available targets:"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  test-race       - Run tests with race detection"
	@echo "  test-full       - Run comprehensive tests with coverage and race detection"
	@echo "  clean           - Clean test artifacts"
	@echo "  build           - Build the project"
	@echo "  run             - Run the project"
	@echo "  deps            - Install dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  generate        - Generate code"
	@echo "  bench           - Run benchmarks"
	@echo "  help            - Show this help"

# Development shortcuts
dev-setup: deps generate fmt

# CI/CD target
ci: clean deps generate fmt test-full lint

# Pre-commit checks
pre-commit: fmt lint test-race