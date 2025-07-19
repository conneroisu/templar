package registry

// GetDependencyAnalyzer returns the dependency analyzer
func (r *ComponentRegistry) GetDependencyAnalyzer() *DependencyAnalyzer {
	return r.dependencyAnalyzer
}

// UpdateAllDependencies updates dependencies for all components
func (r *ComponentRegistry) UpdateAllDependencies() error {
	if r.dependencyAnalyzer == nil {
		return nil
	}
	return r.dependencyAnalyzer.UpdateAllDependencies()
}

// GetDependents returns components that depend on the given component
func (r *ComponentRegistry) GetDependents(componentName string) []*ComponentInfo {
	if r.dependencyAnalyzer == nil {
		return nil
	}
	return r.dependencyAnalyzer.GetDependents(componentName)
}

// GetDependencyGraph returns the full dependency graph
func (r *ComponentRegistry) GetDependencyGraph() map[string][]string {
	if r.dependencyAnalyzer == nil {
		return make(map[string][]string)
	}
	return r.dependencyAnalyzer.GetDependencyGraph()
}

