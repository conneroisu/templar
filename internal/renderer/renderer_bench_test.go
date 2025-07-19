package renderer

import (
	"fmt"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

func BenchmarkComponentRenderer_GenerateMockData(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	component := &registry.ComponentInfo{
		Name:    "TestComponent",
		Package: "test",
		Parameters: []registry.ParameterInfo{
			{Name: "title", Type: "string", Optional: false},
			{Name: "count", Type: "int", Optional: false},
			{Name: "active", Type: "bool", Optional: false},
			{Name: "items", Type: "[]string", Optional: false},
			{Name: "data", Type: "CustomType", Optional: false},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.generateMockData(component)
	}
}

func BenchmarkComponentRenderer_GenerateMockString(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	paramNames := []string{
		"title", "name", "email", "message", "url", "variant", "color", "size", "custom",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paramName := paramNames[i%len(paramNames)]
		renderer.generateMockString(paramName)
	}
}

func BenchmarkComponentRenderer_GenerateGoCode(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	component := &registry.ComponentInfo{
		Name:    "Button",
		Package: "components",
		Parameters: []registry.ParameterInfo{
			{Name: "text", Type: "string", Optional: false},
			{Name: "disabled", Type: "bool", Optional: false},
		},
	}

	mockData := map[string]interface{}{
		"text":     "Click Me",
		"disabled": false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.generateGoCode(component, mockData)
	}
}

func BenchmarkComponentRenderer_RenderComponentWithLayout(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	componentName := "TestComponent"
	html := "<div class='test-component'><h1>Hello World</h1><p>This is a test component</p></div>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.RenderComponentWithLayout(componentName, html)
	}
}

func BenchmarkComponentRenderer_ValidateWorkDir(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	validDirs := []string{
		".templar/render/test1",
		".templar/render/test2",
		".templar/render/component-a",
		".templar/render/component-b",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dir := validDirs[i%len(validDirs)]
		renderer.validateWorkDir(dir)
	}
}

func BenchmarkComponentRenderer_Memory(b *testing.B) {
	b.ReportAllocs()

	reg := registry.NewComponentRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer := NewComponentRenderer(reg)

		component := &registry.ComponentInfo{
			Name:    fmt.Sprintf("Component%d", i),
			Package: "test",
			Parameters: []registry.ParameterInfo{
				{Name: "param1", Type: "string", Optional: false},
				{Name: "param2", Type: "int", Optional: false},
			},
		}

		mockData := renderer.generateMockData(component)
		renderer.generateGoCode(component, mockData)
	}
}

func BenchmarkComponentRenderer_LargeComponent(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Create a component with many parameters
	var params []registry.ParameterInfo
	for i := 0; i < 20; i++ {
		params = append(params, registry.ParameterInfo{
			Name:     fmt.Sprintf("param%d", i),
			Type:     "string",
			Optional: false,
		})
	}

	component := &registry.ComponentInfo{
		Name:       "LargeComponent",
		Package:    "components",
		Parameters: params,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockData := renderer.generateMockData(component)
		renderer.generateGoCode(component, mockData)
	}
}

func BenchmarkComponentRenderer_ManyComponents(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	// Pre-register many components
	for i := 0; i < 100; i++ {
		component := &registry.ComponentInfo{
			Name:         fmt.Sprintf("Component%d", i),
			Package:      "components",
			FilePath:     fmt.Sprintf("/path/component%d.templ", i),
			Parameters:   []registry.ParameterInfo{{Name: "text", Type: "string", Optional: false}},
			Imports:      []string{},
			LastMod:      time.Now(),
			Hash:         fmt.Sprintf("hash%d", i),
			Dependencies: []string{},
		}
		reg.Register(component)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		componentName := fmt.Sprintf("Component%d", i%100)
		component, exists := reg.Get(componentName)
		if exists {
			mockData := renderer.generateMockData(component)
			renderer.generateGoCode(component, mockData)
		}
	}
}

func BenchmarkComponentRenderer_Concurrent(b *testing.B) {
	reg := registry.NewComponentRegistry()
	renderer := NewComponentRenderer(reg)

	component := &registry.ComponentInfo{
		Name:    "TestComponent",
		Package: "test",
		Parameters: []registry.ParameterInfo{
			{Name: "title", Type: "string", Optional: false},
			{Name: "count", Type: "int", Optional: false},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mockData := renderer.generateMockData(component)
			renderer.generateGoCode(component, mockData)
		}
	})
}
