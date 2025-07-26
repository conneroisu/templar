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

func TestEnhancedWebInterface(t *testing.T) {
	// Setup
	reg := registry.NewComponentRegistry()
	component := &types.ComponentInfo{
		Name:    "TestButton",
		Package: "components",
		Parameters: []types.ParameterInfo{
			{Name: "text", Type: "string", Optional: false},
			{Name: "variant", Type: "string", Optional: true},
			{Name: "disabled", Type: "bool", Optional: true},
			{Name: "count", Type: "int", Optional: true},
		},
	}
	reg.Register(component)

	renderer := renderer.NewComponentRenderer(reg)
	server := &PreviewServer{
		registry: reg,
		renderer: renderer,
	}

	t.Run("Enhanced Index Page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/enhanced", nil)
		w := httptest.NewRecorder()

		server.handleEnhancedIndex(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "Enhanced Component Interface")
		assert.Contains(t, body, "Card View")
		assert.Contains(t, body, "prop-row")
		assert.Contains(t, body, "inline-editor")
	})

	t.Run("Component Editor Page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/editor/TestButton", nil)
		w := httptest.NewRecorder()

		server.handleComponentEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "TestButton Editor")
		assert.Contains(t, body, "prop-editor")
		assert.Contains(t, body, "component-preview")
		assert.Contains(t, body, "generatedCode")
		assert.Contains(t, body, "validation-status")
	})

	t.Run("Component Editor - Non-existent Component", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/editor/NonExistent", nil)
		w := httptest.NewRecorder()

		server.handleComponentEditor(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Component Editor - Invalid Component Name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/editor/../invalid", nil)
		w := httptest.NewRecorder()

		server.handleComponentEditor(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestInlineEditor(t *testing.T) {
	// Setup
	reg := registry.NewComponentRegistry()
	component := &types.ComponentInfo{
		Name:    "TestCard",
		Package: "ui",
		Parameters: []types.ParameterInfo{
			{Name: "title", Type: "string", Optional: false},
			{Name: "content", Type: "string", Optional: true},
			{Name: "visible", Type: "bool", Optional: true},
			{Name: "count", Type: "int", Optional: false},
		},
	}
	reg.Register(component)

	renderer := renderer.NewComponentRenderer(reg)
	server := &PreviewServer{
		registry: reg,
		renderer: renderer,
	}

	t.Run("Inline Render Action", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "TestCard",
			"props": map[string]interface{}{
				"title":   "Test Title",
				"content": "Test Content",
				"visible": true,
				"count":   5,
			},
			"action": "render",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "html")
		assert.Contains(t, response, "generated_code")

		html := response["html"].(string)
		assert.Contains(t, html, "TestCard")
		assert.Contains(t, html, "Test Title")

		code := response["generated_code"].(string)
		assert.Contains(t, code, "@TestCard(")
		assert.Contains(t, code, "Test Title")
	})

	t.Run("Inline Validate Action", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "TestCard",
			"props": map[string]interface{}{
				"title":   "Test Title",
				"visible": "invalid_bool", // Invalid type
				"count":   "not_a_number", // Invalid type
			},
			"action": "validate",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "valid")
		assert.Contains(t, response, "errors")

		valid := response["valid"].(bool)
		assert.False(t, valid)

		errors := response["errors"].([]interface{})
		assert.Greater(t, len(errors), 0)
	})

	t.Run("Inline Suggest Action", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "TestCard",
			"props": map[string]interface{}{
				"title": "Existing Title",
				// Missing other props
			},
			"action": "suggest",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "suggestions")

		suggestions := response["suggestions"].(map[string]interface{})
		// Should suggest missing props (content, visible, count)
		assert.Contains(t, suggestions, "content")
		assert.Contains(t, suggestions, "visible")
		assert.Contains(t, suggestions, "count")
		// Should not suggest already provided prop
		assert.NotContains(t, suggestions, "title")
	})

	t.Run("Invalid Request Method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/inline-editor", nil)
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/inline-editor",
			strings.NewReader("invalid json"),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Component Name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "../invalid",
			"props":          map[string]interface{}{},
			"action":         "render",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"].(string), "Invalid component name")
	})

	t.Run("Non-existent Component", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "NonExistent",
			"props":          map[string]interface{}{},
			"action":         "render",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"].(string), "not found")
	})

	t.Run("Invalid Action", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"component_name": "TestCard",
			"props":          map[string]interface{}{},
			"action":         "invalid_action",
		}

		reqBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/inline-editor", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleInlineEditor(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPropValidation(t *testing.T) {
	reg := registry.NewComponentRegistry()
	component := &types.ComponentInfo{
		Name: "ValidationTest",
		Parameters: []types.ParameterInfo{
			{Name: "required_string", Type: "string", Optional: false},
			{Name: "optional_int", Type: "int", Optional: true},
			{Name: "bool_field", Type: "bool", Optional: false},
		},
	}
	reg.Register(component)

	server := &PreviewServer{registry: reg}

	t.Run("Valid Props", func(t *testing.T) {
		props := map[string]interface{}{
			"required_string": "Valid String",
			"optional_int":    42,
			"bool_field":      true,
		}

		errors := server.validateComponentProps(component, props)
		assert.Empty(t, errors)
	})

	t.Run("Missing Required Props", func(t *testing.T) {
		props := map[string]interface{}{
			"optional_int": 42,
			// Missing required_string and bool_field
		}

		errors := server.validateComponentProps(component, props)
		assert.Len(t, errors, 2)

		// Check that both required fields are flagged
		requiredErrors := make(map[string]bool)
		for _, err := range errors {
			if err.Severity == "error" {
				requiredErrors[err.Property] = true
			}
		}

		assert.True(t, requiredErrors["required_string"])
		assert.True(t, requiredErrors["bool_field"])
	})

	t.Run("Type Mismatches", func(t *testing.T) {
		props := map[string]interface{}{
			"required_string": "Valid String",
			"bool_field":      true,
			"optional_int":    "not_a_number", // Wrong type
		}

		errors := server.validateComponentProps(component, props)
		assert.Greater(t, len(errors), 0)

		// Check for type mismatch errors
		typeErrors := make(map[string]bool)
		for _, err := range errors {
			if err.Severity == "error" && strings.Contains(err.Message, "Type mismatch") {
				typeErrors[err.Property] = true
			}
		}

		assert.True(t, typeErrors["optional_int"])
	})

	t.Run("Unknown Props", func(t *testing.T) {
		props := map[string]interface{}{
			"required_string": "Valid String",
			"bool_field":      true,
			"unknown_prop":    "Should be flagged", // Unknown prop
		}

		errors := server.validateComponentProps(component, props)
		assert.Greater(t, len(errors), 0)

		// Check for unknown prop warning
		unknownWarning := false
		for _, err := range errors {
			if err.Property == "unknown_prop" && err.Severity == "warning" {
				unknownWarning = true
				break
			}
		}

		assert.True(t, unknownWarning)
	})
}

func TestTypeCompatibility(t *testing.T) {
	server := &PreviewServer{}

	tests := []struct {
		name         string
		value        interface{}
		expectedType string
		compatible   bool
	}{
		{"string valid", "hello", "string", true},
		{"string invalid", 123, "string", false},
		{"int valid", 42, "int", true},
		{"int from float64 (JSON)", 42.0, "int", true},
		{"int invalid", "not_a_number", "int", false},
		{"bool valid", true, "bool", true},
		{"bool invalid", "true", "bool", false},
		{"float64 valid", 3.14, "float64", true},
		{"float64 from int", 42, "float64", true},
		{"float64 invalid", "not_a_float", "float64", false},
		{"string array valid", []interface{}{"a", "b"}, "[]string", true},
		{"string array invalid", []interface{}{"a", 123}, "[]string", false},
		{"string array not array", "not_array", "[]string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.isCompatibleType(tt.value, tt.expectedType)
			assert.Equal(t, tt.compatible, result)
		})
	}
}

func TestPropSuggestions(t *testing.T) {
	server := &PreviewServer{}
	component := &types.ComponentInfo{
		Name: "SuggestionTest",
		Parameters: []types.ParameterInfo{
			{Name: "title", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "visible", Type: "bool"},
			{Name: "tags", Type: "[]string"},
		},
	}

	t.Run("Suggest Missing Props", func(t *testing.T) {
		currentProps := map[string]interface{}{
			"title": "Existing Title",
			// Missing: count, visible, tags
		}

		suggestions := server.generatePropSuggestions(component, currentProps)

		// Should suggest missing props
		assert.Contains(t, suggestions, "count")
		assert.Contains(t, suggestions, "visible")
		assert.Contains(t, suggestions, "tags")

		// Should not suggest existing props
		assert.NotContains(t, suggestions, "title")
	})

	t.Run("No Suggestions When All Props Provided", func(t *testing.T) {
		currentProps := map[string]interface{}{
			"title":   "Title",
			"count":   42,
			"visible": true,
			"tags":    []string{"tag1", "tag2"},
		}

		suggestions := server.generatePropSuggestions(component, currentProps)

		assert.Empty(t, suggestions)
	})
}
