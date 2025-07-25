package build

import (
	"sync"

	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/types"
)

// MockComponentRegistry is a test-only implementation of ComponentRegistry
// that doesn't create circular dependencies with the registry package.
type MockComponentRegistry struct {
	components map[string]*types.ComponentInfo
	watchers   map[<-chan types.ComponentEvent]chan types.ComponentEvent
	mutex      sync.RWMutex
}

// NewMockComponentRegistry creates a new mock registry for testing
func NewMockComponentRegistry() interfaces.ComponentRegistry {
	return &MockComponentRegistry{
		components: make(map[string]*types.ComponentInfo),
		watchers:   make(map[<-chan types.ComponentEvent]chan types.ComponentEvent),
	}
}

// Register adds or updates a component in the registry
func (m *MockComponentRegistry) Register(component *types.ComponentInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.components[component.Name] = component

	// Notify watchers
	event := types.ComponentEvent{
		Type:      types.EventTypeAdded,
		Component: component,
	}

	for _, ch := range m.watchers {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// Get retrieves a component by name
func (m *MockComponentRegistry) Get(name string) (*types.ComponentInfo, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	component, exists := m.components[name]
	return component, exists
}

// GetAll returns all registered components
func (m *MockComponentRegistry) GetAll() []*types.ComponentInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	components := make([]*types.ComponentInfo, 0, len(m.components))
	for _, component := range m.components {
		components = append(components, component)
	}

	return components
}

// Watch returns a channel for component change events
func (m *MockComponentRegistry) Watch() <-chan types.ComponentEvent {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ch := make(chan types.ComponentEvent, 10)
	m.watchers[ch] = ch

	return ch
}

// UnWatch removes a watcher and closes its channel
func (m *MockComponentRegistry) UnWatch(ch <-chan types.ComponentEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if writerCh, exists := m.watchers[ch]; exists {
		close(writerCh)
		delete(m.watchers, ch)
	}
}

// Count returns the number of registered components
func (m *MockComponentRegistry) Count() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.components)
}

// DetectCircularDependencies returns any circular dependency chains
func (m *MockComponentRegistry) DetectCircularDependencies() [][]string {
	// Mock implementation returns no circular dependencies
	return nil
}
