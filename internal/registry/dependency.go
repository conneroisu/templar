package registry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// DependencyAnalyzer analyzes component dependencies
type DependencyAnalyzer struct {
	registry *ComponentRegistry
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(registry *ComponentRegistry) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		registry: registry,
	}
}

// AnalyzeComponent analyzes dependencies for a single component
func (da *DependencyAnalyzer) AnalyzeComponent(component *types.ComponentInfo) ([]string, error) {
	dependencies := make([]string, 0)

	// Read and parse the component file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, component.FilePath, nil, parser.ParseComments)
	if err != nil {
		return dependencies, fmt.Errorf("failed to parse file %s: %w", component.FilePath, err)
	}

	// Find templ component calls in the AST
	visitor := &dependencyVisitor{
		registry:     da.registry,
		dependencies: make(map[string]bool),
		currentComp:  component.Name,
	}

	ast.Walk(visitor, node)

	// Convert map to slice
	for dep := range visitor.dependencies {
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// AnalyzeComponentFromContent analyzes dependencies from raw content
func (da *DependencyAnalyzer) AnalyzeComponentFromContent(content, componentName string) []string {
	dependencies := make(map[string]bool)

	// Pattern to match component calls in templ syntax
	// Matches: @ComponentName(...) and ComponentName(...)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`@([A-Z][a-zA-Z0-9_]*)\s*\(`),        // @ComponentName(
		regexp.MustCompile(`\s([A-Z][a-zA-Z0-9_]*)\s*\(`),       // ComponentName(
		regexp.MustCompile(`{!\s*([A-Z][a-zA-Z0-9_]*)\s*\(`),    // {! ComponentName(
		regexp.MustCompile(`templ\s+([A-Z][a-zA-Z0-9_]*)\s*\(`), // templ ComponentName(
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Skip comment lines
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		for _, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 {
					dep := match[1]
					// Don't include self-references
					if dep != componentName {
						// Check if this component exists in registry
						if _, exists := da.registry.Get(dep); exists {
							dependencies[dep] = true
						}
					}
				}
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(dependencies))
	for dep := range dependencies {
		result = append(result, dep)
	}

	return result
}

// UpdateAllDependencies updates dependencies for all components
func (da *DependencyAnalyzer) UpdateAllDependencies() error {
	components := da.registry.GetAll()

	for _, component := range components {
		deps, err := da.AnalyzeComponent(component)
		if err != nil {
			// Log error but continue with other components
			continue
		}

		// Update component dependencies
		da.registry.mutex.Lock()
		if existing := da.registry.components[component.Name]; existing != nil {
			existing.Dependencies = deps
		}
		da.registry.mutex.Unlock()
	}

	return nil
}

// GetDependents returns components that depend on the given component
func (da *DependencyAnalyzer) GetDependents(componentName string) []*types.ComponentInfo {
	var dependents []*types.ComponentInfo

	da.registry.mutex.RLock()
	defer da.registry.mutex.RUnlock()

	for _, component := range da.registry.components {
		for _, dep := range component.Dependencies {
			if dep == componentName {
				dependents = append(dependents, component)
				break
			}
		}
	}

	return dependents
}

// GetDependencyGraph returns the full dependency graph
func (da *DependencyAnalyzer) GetDependencyGraph() map[string][]string {
	graph := make(map[string][]string)

	da.registry.mutex.RLock()
	defer da.registry.mutex.RUnlock()

	for name, component := range da.registry.components {
		graph[name] = make([]string, len(component.Dependencies))
		copy(graph[name], component.Dependencies)
	}

	return graph
}

// DetectCircularDependencies detects circular dependencies in the graph
func (da *DependencyAnalyzer) DetectCircularDependencies() [][]string {
	var cycles [][]string
	graph := da.GetDependencyGraph()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	for component := range graph {
		if !visited[component] {
			if cycle := da.detectCycleDFS(component, graph, visited, recStack, path); cycle != nil {
				cycles = append(cycles, cycle)
			}
		}
	}

	return cycles
}

// detectCycleDFS performs DFS to detect cycles
func (da *DependencyAnalyzer) detectCycleDFS(component string, graph map[string][]string, visited, recStack map[string]bool, path []string) []string {
	visited[component] = true
	recStack[component] = true
	path = append(path, component)

	for _, dep := range graph[component] {
		if !visited[dep] {
			if cycle := da.detectCycleDFS(dep, graph, visited, recStack, path); cycle != nil {
				return cycle
			}
		} else if recStack[dep] {
			// Found cycle - extract the cycle from path
			cycleStart := -1
			for i, p := range path {
				if p == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(path)-cycleStart+1)
				copy(cycle, path[cycleStart:])
				cycle[len(cycle)-1] = dep // Close the cycle
				return cycle
			}
		}
	}

	recStack[component] = false
	return nil
}

// dependencyVisitor implements ast.Visitor for dependency analysis
type dependencyVisitor struct {
	registry     *ComponentRegistry
	dependencies map[string]bool
	currentComp  string
}

// Visit visits AST nodes to find component calls
func (v *dependencyVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CallExpr:
		// Look for function calls that might be component calls
		if ident, ok := n.Fun.(*ast.Ident); ok {
			// Check if this looks like a component name (starts with uppercase)
			if len(ident.Name) > 0 && ident.Name[0] >= 'A' && ident.Name[0] <= 'Z' {
				// Don't include self-references
				if ident.Name != v.currentComp {
					// Check if this component exists in registry
					if _, exists := v.registry.Get(ident.Name); exists {
						v.dependencies[ident.Name] = true
					}
				}
			}
		}
	case *ast.SelectorExpr:
		// Handle package.ComponentName calls
		if len(n.Sel.Name) > 0 && n.Sel.Name[0] >= 'A' && n.Sel.Name[0] <= 'Z' {
			if n.Sel.Name != v.currentComp {
				if _, exists := v.registry.Get(n.Sel.Name); exists {
					v.dependencies[n.Sel.Name] = true
				}
			}
		}
	}

	return v
}
