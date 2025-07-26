package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// EnhancedWebInterface provides interactive component editing in the main web interface.
type EnhancedWebInterface struct {
	server *PreviewServer
}

// NewEnhancedWebInterface creates a new enhanced web interface.
func NewEnhancedWebInterface(server *PreviewServer) *EnhancedWebInterface {
	return &EnhancedWebInterface{
		server: server,
	}
}

// handleComponentEditor handles the enhanced component editor interface.
func (s *PreviewServer) handleComponentEditor(w http.ResponseWriter, r *http.Request) {
	// Extract component name from path
	path := strings.TrimPrefix(r.URL.Path, "/editor/")
	componentName := strings.Split(path, "/")[0]

	// Validate component name
	if err := validateComponentName(componentName); err != nil {
		http.Error(w, "Invalid component name: "+err.Error(), http.StatusBadRequest)

		return
	}

	// Get component from registry
	component, exists := s.registry.Get(componentName)
	if !exists {
		http.NotFound(w, r)

		return
	}

	// Serve the enhanced editor interface
	html := s.generateEnhancedEditorHTML(component)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleInlineEditor handles AJAX requests for inline prop editing.
func (s *PreviewServer) handleInlineEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var req struct {
		ComponentName string                 `json:"component_name"`
		Props         map[string]interface{} `json:"props"`
		Action        string                 `json:"action"` // "render", "validate", "suggest"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request: "+err.Error(), http.StatusBadRequest)

		return
	}

	// Validate component name
	if err := validateComponentName(req.ComponentName); err != nil {
		response := map[string]interface{}{"error": "Invalid component name: " + err.Error()}
		s.writeJSONResponse(w, response)

		return
	}

	// Get component from registry
	component, exists := s.registry.Get(req.ComponentName)
	if !exists {
		response := map[string]interface{}{
			"error": fmt.Sprintf("Component '%s' not found", req.ComponentName),
		}
		s.writeJSONResponse(w, response)

		return
	}

	switch req.Action {
	case "render":
		s.handleInlineRender(w, component, req.Props)
	case "validate":
		s.handleInlineValidate(w, component, req.Props)
	case "suggest":
		s.handleInlineSuggest(w, component, req.Props)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleInlineRender renders component with props for inline preview.
func (s *PreviewServer) handleInlineRender(
	w http.ResponseWriter,
	component *types.ComponentInfo,
	props map[string]interface{},
) {
	// Use the playground renderer for consistent behavior
	html, err := s.renderComponentWithProps(component.Name, props)
	if err != nil {
		response := map[string]interface{}{"error": "Render error: " + err.Error()}
		s.writeJSONResponse(w, response)

		return
	}

	response := map[string]interface{}{
		"html":           html,
		"generated_code": s.generateComponentCode(component.Name, props),
	}
	s.writeJSONResponse(w, response)
}

// handleInlineValidate validates props against component schema.
func (s *PreviewServer) handleInlineValidate(
	w http.ResponseWriter,
	component *types.ComponentInfo,
	props map[string]interface{},
) {
	validationErrors := s.validateComponentProps(component, props)

	response := map[string]interface{}{
		"valid":  len(validationErrors) == 0,
		"errors": validationErrors,
	}
	s.writeJSONResponse(w, response)
}

// handleInlineSuggest provides prop suggestions and autocompletion.
func (s *PreviewServer) handleInlineSuggest(
	w http.ResponseWriter,
	component *types.ComponentInfo,
	props map[string]interface{},
) {
	suggestions := s.generatePropSuggestions(component, props)

	response := map[string]interface{}{
		"suggestions": suggestions,
	}
	s.writeJSONResponse(w, response)
}

// validateComponentProps validates props against component parameter definitions.
func (s *PreviewServer) validateComponentProps(
	component *types.ComponentInfo,
	props map[string]interface{},
) []ValidationError {
	var errors []ValidationError

	// Check required parameters
	for _, param := range component.Parameters {
		if !param.Optional {
			if value, exists := props[param.Name]; !exists || value == nil {
				errors = append(errors, ValidationError{
					Property: param.Name,
					Expected: param.Type,
					Actual:   "missing",
					Message:  fmt.Sprintf("Required parameter '%s' is missing", param.Name),
					Severity: "error",
				})
			}
		}
	}

	// Check type compatibility
	for propName, propValue := range props {
		param := s.findParameterByName(component, propName)
		if param == nil {
			errors = append(errors, ValidationError{
				Property: propName,
				Expected: "unknown",
				Actual:   fmt.Sprintf("%T", propValue),
				Message:  fmt.Sprintf("Unknown parameter '%s'", propName),
				Severity: "warning",
			})

			continue
		}

		if !s.isCompatibleType(propValue, param.Type) {
			errors = append(errors, ValidationError{
				Property: propName,
				Expected: param.Type,
				Actual:   fmt.Sprintf("%T", propValue),
				Message: fmt.Sprintf(
					"Type mismatch for '%s': expected %s, got %T",
					propName,
					param.Type,
					propValue,
				),
				Severity: "error",
			})
		}
	}

	return errors
}

// findParameterByName finds a parameter by name in component definition.
func (s *PreviewServer) findParameterByName(
	component *types.ComponentInfo,
	name string,
) *types.ParameterInfo {
	for i, param := range component.Parameters {
		if param.Name == name {
			return &component.Parameters[i]
		}
	}

	return nil
}

// isCompatibleType checks if value type is compatible with expected type.
func (s *PreviewServer) isCompatibleType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)

		return ok
	case "int", "int32", "int64":
		switch value.(type) {
		case int, int32, int64, float64: // JSON numbers come as float64
			return true
		default:
			return false
		}
	case "bool":
		_, ok := value.(bool)

		return ok
	case "[]string":
		if slice, ok := value.([]interface{}); ok {
			for _, item := range slice {
				if _, ok := item.(string); !ok {
					return false
				}
			}

			return true
		}

		return false
	case "float64", "float32":
		switch value.(type) {
		case float64, float32, int, int32, int64:
			return true
		default:
			return false
		}
	default:
		// For complex types, allow any value for now
		return true
	}
}

// generatePropSuggestions generates contextual suggestions for props.
func (s *PreviewServer) generatePropSuggestions(
	component *types.ComponentInfo,
	currentProps map[string]interface{},
) map[string]interface{} {
	suggestions := make(map[string]interface{})

	for _, param := range component.Parameters {
		// Skip if already provided
		if _, exists := currentProps[param.Name]; exists {
			continue
		}

		// Generate contextual suggestions based on parameter name and type
		suggestions[param.Name] = s.generatePropExamples(param.Name, param.Type)
	}

	return suggestions
}

// handleEnhancedIndex serves the enhanced main interface.
func (s *PreviewServer) handleEnhancedIndex(w http.ResponseWriter, r *http.Request) {
	html := s.generateEnhancedIndexHTML()
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// ValidationError represents a prop validation error (already defined in playground.go).
type ValidationError struct {
	Property string `json:"property"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}
