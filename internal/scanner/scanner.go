// Package scanner provides component discovery and analysis for templ templates.
//
// The scanner traverses file systems to find .templ files, parses them using
// Go's AST parser to extract component metadata including parameters, dependencies,
// and documentation. It integrates with the component registry to broadcast
// change events and supports recursive directory scanning with exclude patterns.
// The scanner maintains file hashes for change detection and provides both
// single-file and batch scanning capabilities.
package scanner

import (
	"crypto/md5"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// ComponentScanner discovers and parses templ components
type ComponentScanner struct {
	registry *registry.ComponentRegistry
	fileSet  *token.FileSet
}

// NewComponentScanner creates a new component scanner
func NewComponentScanner(registry *registry.ComponentRegistry) *ComponentScanner {
	return &ComponentScanner{
		registry: registry,
		fileSet:  token.NewFileSet(),
	}
}

// GetRegistry returns the component registry
func (s *ComponentScanner) GetRegistry() *registry.ComponentRegistry {
	return s.registry
}

// ScanDirectory scans a directory for templ components
func (s *ComponentScanner) ScanDirectory(dir string) error {
	// Validate directory path to prevent path traversal
	if _, err := s.validatePath(dir); err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	fmt.Printf("Scanning directory: %s\n", dir)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error walking path %s: %v\n", path, err)
			return err
		}

		if !strings.HasSuffix(path, ".templ") {
			return nil
		}

		// Validate each file path as we encounter it
		if _, err := s.validatePath(path); err != nil {
			fmt.Printf("Skipping invalid path %s: %v\n", path, err)
			return nil
		}

		fmt.Printf("Found templ file: %s\n", path)
		return s.scanFile(path)
	})
}

// ScanFile scans a single file for templ components
func (s *ComponentScanner) ScanFile(path string) error {
	return s.scanFile(path)
}

func (s *ComponentScanner) scanFile(path string) error {
	// Validate and clean the path to prevent directory traversal
	cleanPath, err := s.validatePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", cleanPath, err)
	}

	// Get file info
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("getting file info for %s: %w", cleanPath, err)
	}

	// Calculate hash
	hash := fmt.Sprintf("%x", md5.Sum(content))

	// Parse the file as Go code (templ generates Go)
	astFile, err := parser.ParseFile(s.fileSet, cleanPath, content, parser.ParseComments)
	if err != nil {
		// If it's a .templ file that can't be parsed as Go, try to extract components manually
		return s.parseTemplFile(cleanPath, content, hash, info.ModTime())
	}

	// Extract components from AST
	return s.extractFromAST(cleanPath, astFile, hash, info.ModTime())
}

func (s *ComponentScanner) parseTemplFile(path string, content []byte, hash string, modTime time.Time) error {
	lines := strings.Split(string(content), "\n")
	packageName := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract package name
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packageName = parts[1]
			}
		}

		// Extract templ component declarations
		if strings.HasPrefix(line, "templ ") {
			// Extract component name from templ declaration
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				if idx := strings.Index(name, "("); idx != -1 {
					name = name[:idx]
				}

				component := &registry.ComponentInfo{
					Name:         name,
					Package:      packageName,
					FilePath:     path,
					Parameters:   extractParameters(line),
					Imports:      []string{},
					LastMod:      modTime,
					Hash:         hash,
					Dependencies: []string{},
				}

				s.registry.Register(component)
			}
		}
	}

	return nil
}

func (s *ComponentScanner) extractFromAST(path string, astFile *ast.File, hash string, modTime time.Time) error {
	// Walk the AST to find function declarations that might be templ components
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil && node.Name.IsExported() {
				// Check if this might be a templ component
				if s.isTemplComponent(node) {
					component := &registry.ComponentInfo{
						Name:         node.Name.Name,
						Package:      astFile.Name.Name,
						FilePath:     path,
						Parameters:   s.extractParametersFromFunc(node),
						Imports:      s.extractImports(astFile),
						LastMod:      modTime,
						Hash:         hash,
						Dependencies: []string{},
					}

					s.registry.Register(component)
				}
			}
		}
		return true
	})

	return nil
}

func (s *ComponentScanner) isTemplComponent(fn *ast.FuncDecl) bool {
	// Check if the function returns a templ.Component
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	result := fn.Type.Results.List[0]
	if sel, ok := result.Type.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "templ" && sel.Sel.Name == "Component"
		}
	}

	return false
}

func (s *ComponentScanner) extractParametersFromFunc(fn *ast.FuncDecl) []registry.ParameterInfo {
	var params []registry.ParameterInfo

	if fn.Type.Params == nil {
		return params
	}

	for _, param := range fn.Type.Params.List {
		paramType := ""
		if param.Type != nil {
			paramType = s.typeToString(param.Type)
		}

		for _, name := range param.Names {
			params = append(params, registry.ParameterInfo{
				Name:     name.Name,
				Type:     paramType,
				Optional: false,
				Default:  nil,
			})
		}
	}

	return params
}

func (s *ComponentScanner) extractImports(astFile *ast.File) []string {
	var imports []string

	for _, imp := range astFile.Imports {
		if imp.Path != nil {
			imports = append(imports, imp.Path.Value)
		}
	}

	return imports
}

func (s *ComponentScanner) typeToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return s.typeToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + s.typeToString(e.X)
	case *ast.ArrayType:
		return "[]" + s.typeToString(e.Elt)
	default:
		return "unknown"
	}
}

func extractParameters(line string) []registry.ParameterInfo {
	// Simple parameter extraction from templ declaration
	// This is a basic implementation - real parser would be more robust
	if !strings.Contains(line, "(") {
		return []registry.ParameterInfo{}
	}

	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start == -1 || end == -1 || start >= end {
		return []registry.ParameterInfo{}
	}

	paramStr := line[start+1 : end]
	if strings.TrimSpace(paramStr) == "" {
		return []registry.ParameterInfo{}
	}

	// Basic parameter parsing - handle both "name type" and "name, name type" patterns
	parts := strings.Split(paramStr, ",")
	var params []registry.ParameterInfo

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by space to get name and type
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			// Handle "name type" format
			params = append(params, registry.ParameterInfo{
				Name:     fields[0],
				Type:     fields[1],
				Optional: false,
				Default:  nil,
			})
		} else if len(fields) == 1 {
			// Handle single parameter name (type might be from previous param)
			params = append(params, registry.ParameterInfo{
				Name:     fields[0],
				Type:     "string", // Default type
				Optional: false,
				Default:  nil,
			})
		}
	}

	return params
}

// validatePath validates and cleans a file path to prevent directory traversal
func (s *ComponentScanner) validatePath(path string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Get absolute path to normalize
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	// Ensure the path is within the current working directory or its subdirectories
	// This prevents directory traversal attacks
	if !strings.HasPrefix(absPath, cwd) {
		return "", fmt.Errorf("path %s is outside current working directory", path)
	}

	// Additional security check: reject paths with suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal: %s", path)
	}

	return cleanPath, nil
}
