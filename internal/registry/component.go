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
	"sync"
	"time"
)

// ComponentRegistry manages all discovered components
type ComponentRegistry struct {
	components         map[string]*ComponentInfo
	mutex              sync.RWMutex
	watchers           []chan ComponentEvent
	dependencyAnalyzer *DependencyAnalyzer
}

// ComponentInfo holds metadata about a templ component
type ComponentInfo struct {
	Name         string
	Package      string
	FilePath     string
	Parameters   []ParameterInfo
	Imports      []string
	LastMod      time.Time
	Hash         string
	Dependencies []string
	Metadata     map[string]interface{} // Plugin-specific metadata
}

// ParameterInfo describes a component parameter
type ParameterInfo struct {
	Name     string
	Type     string
	Optional bool
	Default  interface{}
}

// ComponentEvent represents a change in the component registry
type ComponentEvent struct {
	Type      EventType
	Component *ComponentInfo
	Timestamp time.Time
}

// EventType represents the type of component event
type EventType int

const (
	EventTypeAdded EventType = iota
	EventTypeUpdated
	EventTypeRemoved
)

// NewComponentRegistry creates a new component registry
func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{
		components: make(map[string]*ComponentInfo),
		watchers:   make([]chan ComponentEvent, 0),
	}

	// Initialize dependency analyzer
	registry.dependencyAnalyzer = NewDependencyAnalyzer(registry)

	return registry
}

// Register adds or updates a component in the registry
func (r *ComponentRegistry) Register(component *ComponentInfo) {
	r.mutex.Lock()

	eventType := EventTypeAdded
	if _, exists := r.components[component.Name]; exists {
		eventType = EventTypeUpdated
	}

	r.components[component.Name] = component
	r.mutex.Unlock()

	// Analyze dependencies for the component
	if r.dependencyAnalyzer != nil {
		deps, err := r.dependencyAnalyzer.AnalyzeComponent(component)
		if err == nil {
			r.mutex.Lock()
			component.Dependencies = deps
			r.mutex.Unlock()
		}
	}

	// Notify watchers
	r.mutex.RLock()
	event := ComponentEvent{
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
func (r *ComponentRegistry) Get(name string) (*ComponentInfo, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	component, exists := r.components[name]
	return component, exists
}

// GetAll returns all registered components
func (r *ComponentRegistry) GetAll() []*ComponentInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*ComponentInfo, 0, len(r.components))
	for _, component := range r.components {
		result = append(result, component)
	}
	return result
}

// GetAllMap returns all registered components as a map
func (r *ComponentRegistry) GetAllMap() map[string]*ComponentInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*ComponentInfo)
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
	event := ComponentEvent{
		Type:      EventTypeRemoved,
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

// Watch returns a channel that receives component events
func (r *ComponentRegistry) Watch() <-chan ComponentEvent {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ch := make(chan ComponentEvent, 100)
	r.watchers = append(r.watchers, ch)
	return ch
}

// UnWatch removes a watcher channel and closes it
func (r *ComponentRegistry) UnWatch(ch <-chan ComponentEvent) {
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

// DetectCircularDependencies detects circular dependencies using the dependency analyzer
func (r *ComponentRegistry) DetectCircularDependencies() [][]string {
	if r.dependencyAnalyzer == nil {
		return nil
	}
	return r.dependencyAnalyzer.DetectCircularDependencies()
}
