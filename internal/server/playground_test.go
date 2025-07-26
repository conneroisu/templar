package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestPlaygroundRender(t *testing.T) {
	// Setup
	reg := registry.NewComponentRegistry()
	component := &types.ComponentInfo{
		Name:    "TestButton",
		Package: "components",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string", Optional: false},
			{Name: "variant", Type: "string", Optional: true},
			{Name: "disabled", Type: "bool", Optional: true},
		},
	}
	reg.Register(component)

	renderer := renderer.NewComponentRenderer(reg)
	server := &PreviewServer{
		registry: reg,
		renderer: renderer,
	}

	tests := []struct {
		name         string
		request      PlaygroundRequest
		expectError  bool
		validateFunc func(*testing.T, PlaygroundResponse)
	}{
		{
			name: "valid component with mock data",
			request: PlaygroundRequest{
				ComponentName: "TestButton",
				MockData:      true,
				GenerateCode:  true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, resp PlaygroundResponse) {
				assert.NotEmpty(t, resp.AvailableProps)
				assert.Equal(t, 3, len(resp.AvailableProps))
				assert.NotEmpty(t, resp.CurrentProps)
				assert.NotEmpty(t, resp.GeneratedCode)
				assert.NotNil(t, resp.ComponentMetadata)
			},
		},
		{
			name: "valid component with custom props",
			request: PlaygroundRequest{
				ComponentName: "TestButton",
				Props: map[string]interface{}{
					"text":     "Click Me",
					"variant":  "primary",
					"disabled": false,
				},
				GenerateCode: true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, resp PlaygroundResponse) {
				assert.Equal(t, "Click Me", resp.CurrentProps["text"])
				assert.Equal(t, "primary", resp.CurrentProps["variant"])
				assert.Equal(t, false, resp.CurrentProps["disabled"])
				assert.Contains(t, resp.GeneratedCode, "TestButton(")
				assert.Contains(t, resp.GeneratedCode, "Click Me")
			},
		},
		{
			name: "invalid component name",
			request: PlaygroundRequest{
				ComponentName: "NonExistentComponent",
				MockData:      true,
			},
			expectError: true,
			validateFunc: func(t *testing.T, resp PlaygroundResponse) {
				assert.NotEmpty(t, resp.Error)
				assert.Contains(t, resp.Error, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			reqBody, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/playground/render",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			server.handlePlaygroundRender(w, req)

			// Verify response
			var response PlaygroundResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)

			if tt.expectError {
				assert.NotEmpty(t, response.Error)
			} else {
				assert.Empty(t, response.Error)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, response)
			}
		})
	}
}

func TestMockDataGeneration(t *testing.T) {
	server := &PreviewServer{}

	component := &types.ComponentInfo{
		Name: "TestCard",
		Parameters: []types.ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "active", Type: "bool"},
			{Name: "tags", Type: "[]string"},
			{Name: "price", Type: "float64"},
		},
	}

	mockData := server.generateIntelligentMockData(component)

	// Verify mock data generation
	assert.Equal(t, 5, len(mockData))

	// Check string generation
	title, exists := mockData["title"]
	assert.True(t, exists)
	assert.IsType(t, "", title)
	assert.Contains(t, title.(string), "Title") // Should contain contextual content

	// Check integer generation
	count, exists := mockData["count"]
	assert.True(t, exists)
	assert.IsType(t, 0, count)

	// Check boolean generation
	active, exists := mockData["active"]
	assert.True(t, exists)
	assert.IsType(t, true, active)

	// Check slice generation
	tags, exists := mockData["tags"]
	assert.True(t, exists)
	assert.IsType(t, []string{}, tags)
	assert.Greater(t, len(tags.([]string)), 0)

	// Check float generation
	price, exists := mockData["price"]
	assert.True(t, exists)
	assert.IsType(t, 0.0, price)
}

func TestPropDefinitionExtraction(t *testing.T) {
	server := &PreviewServer{}

	component := &types.ComponentInfo{
		Name: "TestForm",
		Parameters: []types.ParameterInfo{
			{Name: "title", Type: "string", Optional: false},
			{Name: "placeholder", Type: "string", Optional: true},
			{Name: "maxLength", Type: "int", Optional: true},
			{Name: "required", Type: "bool", Optional: false},
		},
	}

	props := server.extractPropDefinitions(component)

	assert.Equal(t, 4, len(props))

	// Check required prop
	titleProp := findPropByName(props, "title")
	assert.NotNil(t, titleProp)
	assert.Equal(t, "string", titleProp.Type)
	assert.True(t, titleProp.Required)
	assert.NotEmpty(t, titleProp.Description)
	assert.NotEmpty(t, titleProp.Examples)

	// Check optional prop
	placeholderProp := findPropByName(props, "placeholder")
	assert.NotNil(t, placeholderProp)
	assert.Equal(t, "string", placeholderProp.Type)
	assert.False(t, placeholderProp.Required)

	// Check int prop
	maxLengthProp := findPropByName(props, "maxLength")
	assert.NotNil(t, maxLengthProp)
	assert.Equal(t, "int", maxLengthProp.Type)
	assert.NotEmpty(t, maxLengthProp.Examples)

	// Check bool prop
	requiredProp := findPropByName(props, "required")
	assert.NotNil(t, requiredProp)
	assert.Equal(t, "bool", requiredProp.Type)
	assert.Contains(t, requiredProp.Examples, "true")
	assert.Contains(t, requiredProp.Examples, "false")
}

func TestCodeGeneration(t *testing.T) {
	server := &PreviewServer{}

	props := map[string]interface{}{
		"title":    "Test Title",
		"count":    42,
		"active":   true,
		"tags":     []string{"react", "typescript"},
		"price":    29.99,
		"callback": "handleClick",
	}

	code := server.generateComponentCode("MyComponent", props)

	// Verify code structure
	assert.Contains(t, code, "@MyComponent(")
	assert.Contains(t, code, ")")

	// Verify different prop types are formatted correctly
	assert.Contains(t, code, `title: "Test Title"`)
	assert.Contains(t, code, "count: 42")
	assert.Contains(t, code, "active: true")
	assert.Contains(t, code, `price: 29.99`)
	assert.Contains(t, code, `tags: []string{"react", "typescript"}`)
	assert.Contains(t, code, `callback: "handleClick"`)
}

func TestPlaygroundIndex(t *testing.T) {
	// Setup
	reg := registry.NewComponentRegistry()
	components := []*types.ComponentInfo{
		{
			Name:       "Button",
			Package:    "ui",
			Parameters: []types.ParameterInfo{{Name: "text", Type: "string"}},
		},
		{
			Name:    "Card",
			Package: "layout",
			Parameters: []types.ParameterInfo{
				{Name: "title", Type: "string"},
				{Name: "content", Type: "string"},
			},
		},
	}

	for _, comp := range components {
		reg.Register(comp)
	}

	server := &PreviewServer{registry: reg}

	req := httptest.NewRequest(http.MethodGet, "/playground", nil)
	w := httptest.NewRecorder()

	server.handlePlaygroundIndex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "Component Playground")
	assert.Contains(t, body, "Button")
	assert.Contains(t, body, "Card")
	assert.Contains(t, body, "ui")
	assert.Contains(t, body, "layout")
}

func TestPlaygroundComponent(t *testing.T) {
	// Setup
	reg := registry.NewComponentRegistry()
	component := &types.ComponentInfo{
		Name:    "TestButton",
		Package: "components",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string"},
		},
	}
	reg.Register(component)

	renderer := renderer.NewComponentRenderer(reg)
	server := &PreviewServer{
		registry: reg,
		renderer: renderer,
	}

	req := httptest.NewRequest(http.MethodGet, "/playground/TestButton", nil)
	w := httptest.NewRecorder()

	server.handlePlaygroundComponent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "TestButton Playground")
	assert.Contains(t, body, "prop-editor")
	assert.Contains(t, body, "component-container")
	assert.Contains(t, body, "Generated Code")
}

func TestPlaygroundComponentNotFound(t *testing.T) {
	server := &PreviewServer{registry: registry.NewComponentRegistry()}

	req := httptest.NewRequest(http.MethodGet, "/playground/NonExistent", nil)
	w := httptest.NewRecorder()

	server.handlePlaygroundComponent(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMockValueGeneration(t *testing.T) {
	server := &PreviewServer{}

	tests := []struct {
		name      string
		paramName string
		paramType string
		validate  func(interface{}) bool
	}{
		{
			name:      "string title",
			paramName: "title",
			paramType: "string",
			validate: func(v interface{}) bool {
				s, ok := v.(string)

				return ok && len(s) > 0 &&
					(strings.Contains(s, "Title") || strings.Contains(s, "Sample"))
			},
		},
		{
			name:      "int count",
			paramName: "count",
			paramType: "int",
			validate: func(v interface{}) bool {
				_, ok := v.(int)

				return ok
			},
		},
		{
			name:      "bool active",
			paramName: "active",
			paramType: "bool",
			validate: func(v interface{}) bool {
				val, ok := v.(bool)

				return ok && val == true // Should default to true for "active"
			},
		},
		{
			name:      "string slice tags",
			paramName: "tags",
			paramType: "[]string",
			validate: func(v interface{}) bool {
				slice, ok := v.([]string)

				return ok && len(slice) > 0
			},
		},
		{
			name:      "float price",
			paramName: "price",
			paramType: "float64",
			validate: func(v interface{}) bool {
				_, ok := v.(float64)

				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := server.generateMockValueForType(tt.paramName, tt.paramType)
			assert.True(
				t,
				tt.validate(value),
				"Generated value %v did not pass validation for %s:%s",
				value,
				tt.paramName,
				tt.paramType,
			)
		})
	}
}

// Helper function to find prop by name.
func findPropByName(props []PropDefinition, name string) *PropDefinition {
	for _, prop := range props {
		if prop.Name == name {
			return &prop
		}
	}

	return nil
}

func TestViewportSizeHandling(t *testing.T) {
	server := &PreviewServer{}

	// Test with default viewport
	html := server.wrapInPlaygroundLayout(
		"TestComponent",
		"<div>Test</div>",
		"light",
		ViewportSize{},
	)
	assert.Contains(t, html, "1200")    // Should default to 1200px width
	assert.Contains(t, html, "Desktop") // Should default to Desktop

	// Test with custom viewport
	customViewport := ViewportSize{Width: 375, Height: 667, Name: "Mobile"}
	html = server.wrapInPlaygroundLayout("TestComponent", "<div>Test</div>", "dark", customViewport)
	assert.Contains(t, html, "375")
	assert.Contains(t, html, "Mobile")
	assert.Contains(t, html, "theme-dark")
}

func TestIntelligentMockDataContextAwareness(t *testing.T) {
	server := &PreviewServer{}

	// Test email parameter
	emailValue := server.generateMockString("email")
	assert.Contains(t, emailValue, "@")
	assert.Contains(t, emailValue, ".")

	// Test URL parameter
	urlValue := server.generateMockString("url")
	assert.Contains(t, urlValue, "http")

	// Test title parameter
	titleValue := server.generateMockString("title")
	assert.Contains(t, titleValue, "Title")

	// Test numeric parameters
	widthValue := server.generateMockInt("width")
	assert.Greater(t, widthValue, 0)

	heightValue := server.generateMockInt("height")
	assert.Greater(t, heightValue, 0)
}
