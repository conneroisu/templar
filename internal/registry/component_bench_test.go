package registry

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkComponentRegistry_Register(b *testing.B) {
	registry := NewComponentRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		component := &ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "test",
			FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
			Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		registry.Register(component)
	}
}

func BenchmarkComponentRegistry_Get(b *testing.B) {
	registry := NewComponentRegistry()

	// Pre-populate with components
	for i := 0; i < 1000; i++ {
		component := &ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "test",
			FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
			Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		registry.Register(component)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		componentName := fmt.Sprintf("Component%d", i%1000)
		registry.Get(componentName)
	}
}

func BenchmarkComponentRegistry_List(b *testing.B) {
	registry := NewComponentRegistry()

	// Pre-populate with components
	for i := 0; i < 100; i++ {
		component := &ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "test",
			FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
			Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		registry.Register(component)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.GetAll()
	}
}

func BenchmarkComponentRegistry_Watch(b *testing.B) {
	registry := NewComponentRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := registry.Watch()
		registry.UnWatch(ch) // Clean up immediately
	}
}

func BenchmarkComponentRegistry_Concurrent(b *testing.B) {
	registry := NewComponentRegistry()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			component := &ComponentInfo{
				Name:         fmt.Sprintf("Component%d", i),
				Package:      "test",
				FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
				Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
				Imports:      []string{"context"},
				LastMod:      time.Now(),
				Hash:         fmt.Sprintf("hash%d", i),
				Dependencies: []string{},
			}
			registry.Register(component)

			// Alternate between register and get operations
			if i%2 == 0 {
				registry.Get(fmt.Sprintf("Component%d", i/2))
			}
			i++
		}
	})
}

func BenchmarkComponentRegistry_EventBroadcast(b *testing.B) {
	registry := NewComponentRegistry()

	// Create multiple subscribers
	subscribers := make([]<-chan ComponentEvent, 10)
	for i := range subscribers {
		subscribers[i] = registry.Watch()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		component := &ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "test",
			FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
			Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		registry.Register(component)
	}

	// Clean up channels using UnWatch
	for _, ch := range subscribers {
		registry.UnWatch(ch)
	}
}

func BenchmarkComponentInfo_Access(b *testing.B) {
	component := &ComponentInfo{
		Name:         "TestComponent",
		Package:      "test",
		FilePath:     "/path/to/component.templ",
		Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
		Imports:      []string{"context"},
		LastMod:      time.Now(),
		Hash:         "originalhash",
		Dependencies: []string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark accessing component properties
		_ = component.Name
		_ = component.Hash
		_ = len(component.Parameters)
	}
}

func BenchmarkComponentRegistry_Memory(b *testing.B) {
	b.ReportAllocs()

	registry := NewComponentRegistry()

	for i := 0; i < b.N; i++ {
		component := &ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "test",
			FilePath:     fmt.Sprintf("/path/to/component%d.templ", i),
			Parameters:   []ParameterInfo{{Name: "param", Type: "string", Optional: false}},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		registry.Register(component)
	}
}
