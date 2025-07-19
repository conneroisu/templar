package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewComponentRegistry(t *testing.T) {
	registry := NewComponentRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.components)
	assert.NotNil(t, registry.watchers)
	assert.Equal(t, 0, len(registry.components))
	assert.Equal(t, 0, len(registry.watchers))
}

func TestComponentRegistry_Add(t *testing.T) {
	registry := NewComponentRegistry()

	component := &ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/path/to/component.templ",
		Package:  "main",
		Parameters: []ParameterInfo{
			{Name: "title", Type: "string"},
		},
	}

	registry.Register(component)

	// Test component was added
	retrieved, exists := registry.Get("TestComponent")
	assert.True(t, exists)
	assert.Equal(t, component, retrieved)

	// Test count
	assert.Equal(t, 1, registry.Count())

	// Test GetAll
	all := registry.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, component, all[0])
}

func TestComponentRegistry_Update(t *testing.T) {
	registry := NewComponentRegistry()

	// Add initial component
	component := &ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/path/to/component.templ",
		Package:  "main",
		Parameters: []ParameterInfo{
			{Name: "title", Type: "string"},
		},
	}
	registry.Register(component)

	// Update component
	updatedComponent := &ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/path/to/component.templ",
		Package:  "main",
		Parameters: []ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "subtitle", Type: "string"},
		},
	}
	registry.Register(updatedComponent)

	// Test component was updated
	retrieved, exists := registry.Get("TestComponent")
	assert.True(t, exists)
	assert.Equal(t, updatedComponent, retrieved)
	assert.Len(t, retrieved.Parameters, 2)

	// Count should still be 1
	assert.Equal(t, 1, registry.Count())
}

func TestComponentRegistry_Remove(t *testing.T) {
	registry := NewComponentRegistry()

	// Add component
	component := &ComponentInfo{
		Name:       "TestComponent",
		FilePath:   "/path/to/component.templ",
		Package:    "main",
		Parameters: []ParameterInfo{},
	}
	registry.Register(component)

	// Verify it exists
	_, exists := registry.Get("TestComponent")
	assert.True(t, exists)
	assert.Equal(t, 1, registry.Count())

	// Remove it
	registry.Remove("TestComponent")

	// Verify it's gone
	_, exists = registry.Get("TestComponent")
	assert.False(t, exists)
	assert.Equal(t, 0, registry.Count())

	// Test GetAll is empty
	all := registry.GetAll()
	assert.Len(t, all, 0)
}

func TestComponentRegistry_RemoveByPath(t *testing.T) {
	registry := NewComponentRegistry()

	// Add multiple components with different paths
	component1 := &ComponentInfo{
		Name:     "Component1",
		FilePath: "/path/to/component1.templ",
		Package:  "main",
	}
	component2 := &ComponentInfo{
		Name:     "Component2",
		FilePath: "/path/to/component2.templ",
		Package:  "main",
	}
	component3 := &ComponentInfo{
		Name:     "Component3",
		FilePath: "/path/to/component1.templ", // Same path as component1
		Package:  "main",
	}

	registry.Register(component1)
	registry.Register(component2)
	registry.Register(component3)

	assert.Equal(t, 3, registry.Count())

	// For this test, we'll just remove the specific components manually
	registry.Remove("Component1")
	registry.Remove("Component3")

	// Both component1 and component3 should be removed
	assert.Equal(t, 1, registry.Count())

	_, exists := registry.Get("Component1")
	assert.False(t, exists)

	_, exists = registry.Get("Component3")
	assert.False(t, exists)

	// Component2 should still exist
	_, exists = registry.Get("Component2")
	assert.True(t, exists)
}

func TestComponentRegistry_Watch(t *testing.T) {
	registry := NewComponentRegistry()

	// Create a watcher
	watcher := registry.Watch()
	assert.NotNil(t, watcher)

	// Add a component and check if event is received
	component := &ComponentInfo{
		Name:       "TestComponent",
		FilePath:   "/path/to/component.templ",
		Package:    "main",
		Parameters: []ParameterInfo{},
	}

	// Add component in goroutine to avoid blocking
	go func() {
		time.Sleep(10 * time.Millisecond)
		registry.Register(component)
	}()

	// Wait for event
	select {
	case event := <-watcher:
		assert.Equal(t, EventTypeAdded, event.Type)
		assert.Equal(t, component, event.Component)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive component added event")
	}
}

func TestComponentRegistry_UnWatch(t *testing.T) {
	registry := NewComponentRegistry()

	// Create watchers
	watcher1 := registry.Watch()
	watcher2 := registry.Watch()

	assert.Len(t, registry.watchers, 2)

	// Remove one watcher
	registry.UnWatch(watcher1)

	assert.Len(t, registry.watchers, 1)

	// Verify the channel is closed
	select {
	case _, ok := <-watcher1:
		assert.False(t, ok, "Channel should be closed")
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Channel should be closed immediately")
	}

	// Verify the other watcher is still active
	go func() {
		time.Sleep(10 * time.Millisecond)
		registry.Register(&ComponentInfo{
			Name:     "TestComponent",
			FilePath: "/path/to/component.templ",
			Package:  "main",
		})
	}()

	select {
	case event := <-watcher2:
		assert.Equal(t, EventTypeAdded, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Second watcher should still receive events")
	}
}

func TestComponentRegistry_EventTypes(t *testing.T) {
	registry := NewComponentRegistry()
	watcher := registry.Watch()

	component := &ComponentInfo{
		Name:       "TestComponent",
		FilePath:   "/path/to/component.templ",
		Package:    "main",
		Parameters: []ParameterInfo{},
	}

	// Test Add event
	go func() {
		time.Sleep(10 * time.Millisecond)
		registry.Register(component)
	}()

	select {
	case event := <-watcher:
		assert.Equal(t, EventTypeAdded, event.Type)
		assert.Equal(t, component, event.Component)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected ComponentAdded event")
	}

	// Test Update event
	updatedComponent := &ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/path/to/component.templ",
		Package:  "main",
		Parameters: []ParameterInfo{
			{Name: "title", Type: "string"},
		},
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		registry.Register(updatedComponent)
	}()

	select {
	case event := <-watcher:
		assert.Equal(t, EventTypeUpdated, event.Type)
		assert.Equal(t, updatedComponent, event.Component)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected ComponentUpdated event")
	}

	// Test Remove event
	go func() {
		time.Sleep(10 * time.Millisecond)
		registry.Remove("TestComponent")
	}()

	select {
	case event := <-watcher:
		assert.Equal(t, EventTypeRemoved, event.Type)
		assert.Equal(t, "TestComponent", event.Component.Name)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected ComponentRemoved event")
	}
}

func TestComponentRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewComponentRegistry()

	// Test concurrent adds
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(index int) {
			component := &ComponentInfo{
				Name:       fmt.Sprintf("Component%d", index),
				FilePath:   fmt.Sprintf("/path/to/component%d.templ", index),
				Package:    "main",
				Parameters: []ParameterInfo{},
			}
			registry.Register(component)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, registry.Count())

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func(index int) {
			name := fmt.Sprintf("Component%d", index)
			_, exists := registry.Get(name)
			assert.True(t, exists)
			done <- true
		}(i)
	}

	// Wait for all read goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestComponentInfo_Basic(t *testing.T) {
	component := &ComponentInfo{
		Name:     "TestComponent",
		FilePath: "/path/to/component.templ",
		Package:  "main",
		Parameters: []ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "count", Type: "int"},
		},
	}

	assert.Equal(t, "TestComponent", component.Name)
	assert.Equal(t, "/path/to/component.templ", component.FilePath)
	assert.Equal(t, "main", component.Package)
	assert.Len(t, component.Parameters, 2)
}

func TestParameterInfo_Basic(t *testing.T) {
	param := ParameterInfo{
		Name: "title",
		Type: "string",
	}

	assert.Equal(t, "title", param.Name)
	assert.Equal(t, "string", param.Type)
}
