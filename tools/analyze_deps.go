package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	packages := []string{"internal/build", "internal/registry", "internal/server"}
	deps := make(map[string][]string)

	for _, pkg := range packages {
		imports := analyzePackage(pkg)
		deps[pkg] = imports
	}

	fmt.Println("=== DEPENDENCY ANALYSIS ===")
	for pkg, imports := range deps {
		fmt.Printf("\n%s imports:\n", pkg)
		for _, imp := range imports {
			if strings.Contains(imp, "templar/internal") {
				fmt.Printf("  -> %s\n", imp)
			}
		}
	}

	fmt.Println("\n=== CIRCULAR DEPENDENCY ANALYSIS ===")
	findCircularDeps(deps)
}

func analyzePackage(pkgPath string) []string {
	var imports []string

	err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if !contains(imports, importPath) {
				imports = append(imports, importPath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error analyzing package %s: %v", pkgPath, err)
	}

	return imports
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func findCircularDeps(deps map[string][]string) {
	// Check for direct circular dependencies
	for pkg1, imports1 := range deps {
		for _, imp1 := range imports1 {
			if !strings.Contains(imp1, "templar/internal") {
				continue
			}

			// Convert import path to local package name
			localPkg1 := strings.TrimPrefix(imp1, "github.com/conneroisu/templar/")

			for pkg2, imports2 := range deps {
				if pkg1 == pkg2 {
					continue
				}

				// Check if pkg1 imports pkg2 and pkg2 imports pkg1
				if localPkg1 == pkg2 {
					for _, imp2 := range imports2 {
						if !strings.Contains(imp2, "templar/internal") {
							continue
						}
						localPkg2 := strings.TrimPrefix(imp2, "github.com/conneroisu/templar/")
						if localPkg2 == pkg1 {
							fmt.Printf("CIRCULAR: %s <-> %s\n", pkg1, pkg2)
						}
					}
				}
			}
		}
	}

	// Check for indirect circular dependencies (A -> B -> C -> A)
	fmt.Println("\nChecking for indirect circular dependencies...")
	checkIndirectCircular(deps, "internal/build", []string{}, make(map[string]bool))
	checkIndirectCircular(deps, "internal/registry", []string{}, make(map[string]bool))
	checkIndirectCircular(deps, "internal/server", []string{}, make(map[string]bool))
}

func checkIndirectCircular(
	deps map[string][]string,
	currentPkg string,
	path []string,
	visited map[string]bool,
) {
	if visited[currentPkg] {
		if contains(path, currentPkg) {
			// Found a cycle
			cycleStart := -1
			for i, pkg := range path {
				if pkg == currentPkg {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(path[cycleStart:], currentPkg)
				fmt.Printf("INDIRECT CIRCULAR: %s\n", strings.Join(cycle, " -> "))
			}
		}
		return
	}

	visited[currentPkg] = true
	newPath := append(path, currentPkg)

	if imports, ok := deps[currentPkg]; ok {
		for _, imp := range imports {
			if strings.Contains(imp, "templar/internal") {
				localPkg := strings.TrimPrefix(imp, "github.com/conneroisu/templar/")
				if _, exists := deps[localPkg]; exists {
					checkIndirectCircular(deps, localPkg, newPath, visited)
				}
			}
		}
	}

	visited[currentPkg] = false
}
