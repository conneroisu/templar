# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go development project using Nix flakes for reproducible development environments.

## Common Commands

All commands should be run using `nix develop -c <command>` to ensure the proper shell environment is loaded.

### Development
- `nix develop -c dx` - Edit the flake.nix file
- `nix develop -c gx` - Edit the go.mod file
- `nix fmt` - Format code using treefmt (alejandra for Nix)

### Go Development
- `nix develop -c go build` - Build the Go project
- `nix develop -c go test ./...` - Run all tests
- `nix develop -c go run main.go` - Run the main application
- `nix develop -c air` - Hot reload development server
- `nix develop -c golangci-lint run` - Run Go linter
- `nix develop -c gopls` - Go language server

### Project Management
- `nix develop -c goreleaser` - Release management
- `nix develop -c cobra-cli` - CLI application scaffolding

## Development Environment

The Nix development shell provides:

### Go Tools (Go 1.24)
- `air` - Hot reload for Go applications
- `golangci-lint` - Go linter
- `gopls` - Go language server
- `revive` - Go linter
- `golines` - Go code formatter
- `golangci-lint-langserver` - Language server for golangci-lint
- `gomarkdoc` - Go documentation generator
- `gotests` - Go test generator
- `gotools` - Go tools
- `reftools` - Go refactoring tools
- `pprof` - Go profiler
- `graphviz` - Graph visualization (for pprof)
- `goreleaser` - Release automation
- `cobra-cli` - CLI application framework

### Nix Tools
- `alejandra` - Nix formatter
- `nixd` - Nix language server
- `statix` - Nix linter
- `deadnix` - Dead code elimination for Nix

### Environment Variables
- `REPO_ROOT` - Automatically set to the git repository root

## Development Workflow

1. Enter the development environment: `nix develop`
2. Edit Go code and the `go.mod` file as needed
3. Use `air` for hot reloading during development
4. Run tests with `go test ./...`
5. Format code with `nix fmt`
6. Lint code with `golangci-lint run`

## Notes

- The flake supports multiple systems: x86_64-linux, x86_64-darwin, aarch64-linux, aarch64-darwin
- All Go tools are built with Go 1.24 specifically
- The development environment is reproducible across different machines
- Use the provided scripts (`dx`, `gx`) for quick file editing