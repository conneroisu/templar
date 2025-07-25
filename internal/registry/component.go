// Package registry provides a central component registry with event-driven
// architecture for managing templ component metadata and change notifications.
//
// The registry maintains a thread-safe collection of discovered components,
// broadcasts change events to subscribers, and provides component lookup
// and enumeration capabilities. It supports real-time component management
// with automatic registration, updates, and removal, integrating with
// scanners for component discovery and servers for live reload functionality.
package registry

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// ComponentRegistry manages all discovered components with thread-safe operations
// and event-driven notifications.
//
// The registry provides:
// - Thread-safe component registration, lookup, and removal
// - Event broadcasting to subscribers for real-time updates
// - Dependency analysis and circular dependency detection
// - Security hardening through input sanitization
type ComponentRegistry struct {
	// components stores all registered component information indexed by component name
	components map[string]*types.ComponentInfo
	// mutex protects concurrent access to components and watchers
	mutex sync.RWMutex
	// watchers holds channels that receive component change events
	watchers []chan types.ComponentEvent
	// dependencyAnalyzer analyzes component dependencies and detects circular references
	dependencyAnalyzer *DependencyAnalyzer
}

// NewComponentRegistry creates a new component registry with dependency analysis enabled.
//
// The registry is initialized with:
// - Empty component storage
// - No active watchers
// - Dependency analyzer for automatic dependency resolution
func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{
		components: make(map[string]*types.ComponentInfo),
		watchers:   make([]chan types.ComponentEvent, 0),
	}

	// Initialize dependency analyzer
	registry.dependencyAnalyzer = NewDependencyAnalyzer(registry)

	return registry
}

// Register adds or updates a component in the registry with security sanitization.
//
// The method performs:
// 1. Input sanitization to prevent security vulnerabilities
// 2. Component registration or update based on existing state
// 3. Dependency analysis for the registered component
// 4. Event notification to all watchers
//
// The operation is thread-safe and non-blocking for event notifications.
func (r *ComponentRegistry) Register(component *types.ComponentInfo) {
	// Validate and sanitize component data
	component = r.sanitizeComponent(component)

	r.mutex.Lock()

	eventType := types.EventTypeAdded
	if _, exists := r.components[component.Name]; exists {
		eventType = types.EventTypeUpdated
	}

	r.components[component.Name] = component
	r.mutex.Unlock()

	// Analyze dependencies for the component
	if r.dependencyAnalyzer != nil {
		deps, err := r.dependencyAnalyzer.AnalyzeComponent(component)
		if err == nil {
			// Sanitize dependencies to prevent path traversal
			sanitizedDeps := make([]string, len(deps))
			for i, dep := range deps {
				sanitizedDeps[i] = sanitizeFilePath(dep)
			}

			r.mutex.Lock()
			component.Dependencies = sanitizedDeps
			r.mutex.Unlock()
		}
	}

	// Notify watchers
	r.mutex.RLock()
	event := types.ComponentEvent{
		Type:      eventType,
		Component: component,
		Timestamp: time.Now(),
	}

	for _, watcher := range r.watchers {
		select {
		case watcher <- event:
		default:
			// Skip if channel is full
		}
	}
	r.mutex.RUnlock()
}

// Get retrieves a component by name
func (r *ComponentRegistry) Get(name string) (*types.ComponentInfo, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	component, exists := r.components[name]
	return component, exists
}

// GetAll returns all registered components
func (r *ComponentRegistry) GetAll() []*types.ComponentInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*types.ComponentInfo, 0, len(r.components))
	for _, component := range r.components {
		result = append(result, component)
	}
	return result
}

// GetAllMap returns all registered components as a map
func (r *ComponentRegistry) GetAllMap() map[string]*types.ComponentInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*types.ComponentInfo)
	for name, component := range r.components {
		result[name] = component
	}
	return result
}

// Remove removes a component from the registry
func (r *ComponentRegistry) Remove(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	component, exists := r.components[name]
	if !exists {
		return
	}

	delete(r.components, name)

	// Notify watchers
	event := types.ComponentEvent{
		Type:      types.EventTypeRemoved,
		Component: component,
		Timestamp: time.Now(),
	}

	for _, watcher := range r.watchers {
		select {
		case watcher <- event:
		default:
			// Skip if channel is full
		}
	}
}

// Watch returns a channel that receives component events for real-time notifications.
//
// The returned channel receives events for:
// - Component registration (EventTypeAdded)
// - Component updates (EventTypeUpdated)
// - Component removal (EventTypeRemoved)
//
// The channel has a buffer of 100 events to prevent blocking. If the consumer
// cannot keep up, older events may be dropped. Callers should use UnWatch()
// to properly clean up the returned channel.
func (r *ComponentRegistry) Watch() <-chan types.ComponentEvent {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ch := make(chan types.ComponentEvent, 100)
	r.watchers = append(r.watchers, ch)
	return ch
}

// UnWatch removes a watcher channel and closes it
func (r *ComponentRegistry) UnWatch(ch <-chan types.ComponentEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, watcher := range r.watchers {
		if watcher == ch {
			close(watcher)
			r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
			break
		}
	}
}

// Count returns the number of registered components
func (r *ComponentRegistry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.components)
}

// sanitizeComponent sanitizes component data to prevent security issues
func (r *ComponentRegistry) sanitizeComponent(component *types.ComponentInfo) *types.ComponentInfo {
	if component == nil {
		return component
	}

	// Create a copy to avoid modifying the original
	sanitized := *component

	// Sanitize name - only allow alphanumeric and underscores
	sanitized.Name = sanitizeIdentifier(sanitized.Name)

	// Sanitize package name
	sanitized.Package = sanitizeIdentifier(sanitized.Package)

	// Sanitize file path - remove control characters
	sanitized.FilePath = sanitizeFilePath(sanitized.FilePath)

	// Sanitize parameters
	for i := range sanitized.Parameters {
		sanitized.Parameters[i].Name = sanitizeIdentifier(sanitized.Parameters[i].Name)
		sanitized.Parameters[i].Type = sanitizeIdentifier(sanitized.Parameters[i].Type)
	}

	// Sanitize dependencies to prevent path traversal
	if sanitized.Dependencies != nil {
		sanitizedDeps := make([]string, len(sanitized.Dependencies))
		for i, dep := range sanitized.Dependencies {
			sanitizedDeps[i] = sanitizeFilePath(dep)
		}
		sanitized.Dependencies = sanitizedDeps
	}

	return &sanitized
}

// sanitizeIdentifier removes dangerous characters from identifiers
func sanitizeIdentifier(identifier string) string {
	// Only allow alphanumeric characters, underscores, and dots (for package names)
	var cleaned []rune
	for _, r := range identifier {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			cleaned = append(cleaned, r)
		}
	}

	cleanedId := string(cleaned)

	// Additional security check for dangerous system identifiers
	dangerousPatterns := []string{"etc", "system32", "windows", "usr", "bin", "var", "tmp", "passwd", "shadow"}
	lowerCleaned := strings.ToLower(cleanedId)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCleaned, pattern) {
			// Replace with safe alternative
			cleanedId = "safe_component"
			break
		}
	}

	return cleanedId
}

// sanitizeFilePath removes control characters and prevents path traversal attacks
func sanitizeFilePath(path string) string {
	var cleaned []rune
	for _, r := range path {
		// Remove null bytes, newlines, carriage returns, and other control characters
		if r >= 32 && r != 127 { // Printable ASCII characters
			cleaned = append(cleaned, r)
		}
	}

	cleanedPath := string(cleaned)

	// Prevent path traversal attacks
	// Remove any directory traversal sequences
	cleanedPath = strings.ReplaceAll(cleanedPath, "../", "")
	cleanedPath = strings.ReplaceAll(cleanedPath, "..\\", "") // Windows paths
	cleanedPath = strings.ReplaceAll(cleanedPath, "..", "")   // Any remaining double dots

	// Use filepath.Clean to normalize the path and prevent other traversal techniques
	cleanedPath = filepath.Clean(cleanedPath)

	// Check for dangerous system paths before preserving absolute paths
	dangerousPatterns := []string{"etc", "system32", "windows", "usr", "bin", "var", "tmp"}
	lowerPath := strings.ToLower(cleanedPath)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerPath, pattern) {
			// Replace with safe alternative
			return "safe_component.templ"
		}
	}

	// For legitimate component paths, preserve the original structure
	// Only strip absolute path markers if they point to dangerous locations
	if strings.HasPrefix(cleanedPath, "/") &&
		(strings.Contains(lowerPath, "etc") || strings.Contains(lowerPath, "system") ||
			strings.Contains(lowerPath, "usr") || strings.Contains(lowerPath, "bin") ||
			strings.Contains(lowerPath, "var") || strings.Contains(lowerPath, "tmp")) {
		// Path contains dangerous system directories, return safe default
		return "safe_component.templ"
	}

	return cleanedPath
}

// DetectCircularDependencies detects circular dependencies using the dependency analyzer
func (r *ComponentRegistry) DetectCircularDependencies() [][]string {
	if r.dependencyAnalyzer == nil {
		return make([][]string, 0)
	}
	return r.dependencyAnalyzer.DetectCircularDependencies()
}
