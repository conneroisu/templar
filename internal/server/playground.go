package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PlaygroundRequest represents a request to the interactive playground
type PlaygroundRequest struct {
	ComponentName string                 `json:"component_name"`
	Props         map[string]interface{} `json:"props"`
	Theme         string                 `json:"theme,omitempty"`
	ViewportSize  ViewportSize           `json:"viewport_size,omitempty"`
	MockData      bool                   `json:"mock_data,omitempty"`
	GenerateCode  bool                   `json:"generate_code,omitempty"`
}

// PlaygroundResponse represents a response from the interactive playground
type PlaygroundResponse struct {
	HTML              string                 `json:"html"`
	GeneratedCode     string                 `json:"generated_code,omitempty"`
	AvailableProps    []PropDefinition       `json:"available_props"`
	CurrentProps      map[string]interface{} `json:"current_props"`
	MockDataSuggests  map[string]interface{} `json:"mock_data_suggestions"`
	ComponentMetadata *ComponentMetadata     `json:"metadata"`
	Error             string                 `json:"error,omitempty"`
}

// PropDefinition describes a component property
type PropDefinition struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	DefaultVal  interface{} `json:"default_value,omitempty"`
	Description string      `json:"description,omitempty"`
	Examples    []string    `json:"examples,omitempty"`
}

// ViewportSize represents viewport dimensions for responsive testing
type ViewportSize struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Name   string `json:"name,omitempty"`
}

// ComponentMetadata provides additional information about the component
type ComponentMetadata struct {
	Name           string   `json:"name"`
	Package        string   `json:"package"`
	FilePath       string   `json:"file_path"`
	Dependencies   []string `json:"dependencies"`
	LastModified   string   `json:"last_modified"`
	DocComments    string   `json:"doc_comments,omitempty"`
	RenderTime     string   `json:"render_time,omitempty"`
	ViewportPreset string   `json:"viewport_preset,omitempty"`
}

// handlePlaygroundRender handles interactive component playground rendering
func (s *PreviewServer) handlePlaygroundRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PlaygroundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate component name
	if err := validateComponentName(req.ComponentName); err != nil {
		response := PlaygroundResponse{Error: "Invalid component name: " + err.Error()}
		s.writeJSONResponse(w, response)
		return
	}

	// Get component from registry
	component, exists := s.registry.Get(req.ComponentName)
	if !exists {
		response := PlaygroundResponse{
			Error: fmt.Sprintf("Component '%s' not found", req.ComponentName),
		}
		s.writeJSONResponse(w, response)
		return
	}

	// Generate mock data if requested
	if req.MockData || len(req.Props) == 0 {
		req.Props = s.generateIntelligentMockData(component)
	}

	// Render component with custom renderer that supports prop injection
	html, err := s.renderComponentWithProps(req.ComponentName, req.Props)
	if err != nil {
		response := PlaygroundResponse{Error: "Render error: " + err.Error()}
		s.writeJSONResponse(w, response)
		return
	}

	// Wrap in playground layout
	html = s.wrapInPlaygroundLayout(req.ComponentName, html, req.Theme, req.ViewportSize)

	// Generate response
	response := PlaygroundResponse{
		HTML:              html,
		AvailableProps:    s.extractPropDefinitions(component),
		CurrentProps:      req.Props,
		MockDataSuggests:  s.generateMockDataSuggestions(component),
		ComponentMetadata: s.buildComponentMetadata(component),
	}

	// Generate code if requested
	if req.GenerateCode {
		response.GeneratedCode = s.generateComponentCode(req.ComponentName, req.Props)
	}

	s.writeJSONResponse(w, response)
}

// handlePlaygroundComponent serves the interactive playground UI
func (s *PreviewServer) handlePlaygroundComponent(w http.ResponseWriter, r *http.Request) {
	componentName := strings.TrimPrefix(r.URL.Path, "/playground/")
	if componentName == "" {
		s.handlePlaygroundIndex(w, r)
		return
	}

	// Validate component name
	if err := validateComponentName(componentName); err != nil {
		http.Error(w, "Invalid component name: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Check if component exists
	component, exists := s.registry.Get(componentName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	// Serve the playground interface
	html := s.generatePlaygroundHTML(component)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handlePlaygroundIndex serves the main playground page with component list
func (s *PreviewServer) handlePlaygroundIndex(w http.ResponseWriter, r *http.Request) {
	components := s.registry.GetAll()
	html := s.generatePlaygroundIndexHTML(components)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// renderComponentWithProps renders a component with custom props
func (s *PreviewServer) renderComponentWithProps(
	componentName string,
	props map[string]interface{},
) (string, error) {
	// Create a temporary renderer with custom mock data
	renderer := s.createCustomRenderer(props)
	return renderer.RenderComponent(componentName)
}

// generateIntelligentMockData creates contextually appropriate mock data
func (s *PreviewServer) generateIntelligentMockData(
	component *types.ComponentInfo,
) map[string]interface{} {
	mockData := make(map[string]interface{})

	for _, param := range component.Parameters {
		mockData[param.Name] = s.generateMockValueForType(param.Name, param.Type)
	}

	return mockData
}

// generateMockValueForType generates appropriate mock values based on parameter name and type
func (s *PreviewServer) generateMockValueForType(paramName, paramType string) interface{} {
	paramLower := strings.ToLower(paramName)

	switch paramType {
	case "string":
		return s.generateMockString(paramLower)
	case "int", "int64", "int32":
		return s.generateMockInt(paramLower)
	case "bool":
		return s.generateMockBool(paramLower)
	case "[]string":
		return s.generateMockStringSlice(paramLower)
	case "float64", "float32":
		return s.generateMockFloat(paramLower)
	default:
		// For complex types, return a JSON-serializable placeholder
		return fmt.Sprintf("mock_%s", paramName)
	}
}

// generateMockString creates contextually appropriate mock strings
func (s *PreviewServer) generateMockString(paramName string) string {
	contextualValues := map[string][]string{
		"title": {"Sample Title", "Welcome to Templar", "Interactive Component"},
		"name":  {"John Doe", "Alice Smith", "Bob Johnson"},
		"email": {"user@example.com", "alice@company.com", "developer@templar.dev"},
		"text": {
			"Sample text content",
			"Lorem ipsum dolor sit amet",
			"This is example text",
		},
		"label":       {"Click Me", "Submit", "Get Started"},
		"url":         {"https://example.com", "https://templar.dev", "#"},
		"href":        {"https://example.com", "https://templar.dev", "#"},
		"variant":     {"primary", "secondary", "danger", "success"},
		"color":       {"blue", "red", "green", "purple", "gray"},
		"size":        {"small", "medium", "large"},
		"type":        {"button", "submit", "reset"},
		"placeholder": {"Enter text here...", "Search...", "Type something..."},
		"description": {
			"A sample description for this component",
			"This explains what the component does",
		},
		"message": {
			"Hello from the playground!",
			"This is a test message",
			"Component rendered successfully",
		},
		"content": {"This is sample content that demonstrates the component functionality."},
	}

	if values, exists := contextualValues[paramName]; exists {
		return values[0] // Return first contextual value
	}

	// Fallback based on common patterns
	if strings.Contains(paramName, "id") {
		return "sample-id-123"
	}
	if strings.Contains(paramName, "class") {
		return "sample-class"
	}
	if strings.Contains(paramName, "src") {
		return "https://via.placeholder.com/300x200"
	}

	return fmt.Sprintf("Sample %s", cases.Title(language.English).String(paramName))
}

// generateMockInt creates appropriate integer values
func (s *PreviewServer) generateMockInt(paramName string) int {
	contextualValues := map[string]int{
		"width":    300,
		"height":   200,
		"count":    5,
		"max":      100,
		"min":      1,
		"limit":    10,
		"size":     42,
		"length":   250,
		"duration": 5000,
		"delay":    1000,
		"timeout":  3000,
		"port":     8080,
		"id":       123,
		"index":    0,
		"page":     1,
	}

	if value, exists := contextualValues[paramName]; exists {
		return value
	}

	return 42 // Universal answer
}

// generateMockBool creates appropriate boolean values
func (s *PreviewServer) generateMockBool(paramName string) bool {
	// Default to true for these commonly positive attributes
	positiveDefaults := []string{
		"enabled", "visible", "active", "open", "expanded",
		"selected", "checked", "valid", "success", "available",
	}

	for _, positive := range positiveDefaults {
		if strings.Contains(paramName, positive) {
			return true
		}
	}

	// Default to false for these commonly negative attributes
	negativeDefaults := []string{
		"disabled", "hidden", "collapsed", "closed", "error",
		"invalid", "loading", "readonly", "required",
	}

	for _, negative := range negativeDefaults {
		if strings.Contains(paramName, negative) {
			return false
		}
	}

	return true // Default optimistic
}

// generateMockStringSlice creates arrays of contextual strings
func (s *PreviewServer) generateMockStringSlice(paramName string) []string {
	contextualValues := map[string][]string{
		"tags":       {"React", "TypeScript", "CSS", "JavaScript"},
		"categories": {"Frontend", "Backend", "DevOps", "Design"},
		"items":      {"Item 1", "Item 2", "Item 3"},
		"options":    {"Option A", "Option B", "Option C"},
		"choices":    {"Choice 1", "Choice 2", "Choice 3"},
		"values":     {"Value 1", "Value 2", "Value 3"},
		"names":      {"Alice", "Bob", "Charlie"},
		"colors":     {"red", "green", "blue"},
		"sizes":      {"small", "medium", "large"},
		"types":      {"primary", "secondary", "tertiary"},
		"statuses":   {"active", "inactive", "pending"},
		"priorities": {"high", "medium", "low"},
	}

	if values, exists := contextualValues[paramName]; exists {
		return values
	}

	return []string{"Item 1", "Item 2", "Item 3"}
}

// generateMockFloat creates appropriate float values
func (s *PreviewServer) generateMockFloat(paramName string) float64 {
	contextualValues := map[string]float64{
		"scale":      1.0,
		"opacity":    0.8,
		"progress":   0.65,
		"percentage": 75.5,
		"ratio":      1.618,
		"price":      29.99,
		"rate":       4.5,
		"score":      8.7,
		"weight":     1.5,
	}

	if value, exists := contextualValues[paramName]; exists {
		return value
	}

	return 3.14159 // When in doubt, π
}

// extractPropDefinitions extracts property definitions from component
func (s *PreviewServer) extractPropDefinitions(component *types.ComponentInfo) []PropDefinition {
	props := make([]PropDefinition, 0, len(component.Parameters))

	for _, param := range component.Parameters {
		propDef := PropDefinition{
			Name:        param.Name,
			Type:        param.Type,
			Required:    !param.Optional,
			DefaultVal:  param.Default,
			Description: s.generatePropDescription(param.Name, param.Type),
			Examples:    s.generatePropExamples(param.Name, param.Type),
		}
		props = append(props, propDef)
	}

	return props
}

// generatePropDescription creates helpful descriptions for properties
func (s *PreviewServer) generatePropDescription(name, propType string) string {
	descriptions := map[string]string{
		"title":       "The main heading or title text",
		"text":        "The display text content",
		"label":       "The text label for the element",
		"placeholder": "Placeholder text shown when input is empty",
		"variant":     "Visual style variant (primary, secondary, etc.)",
		"size":        "Size of the component (small, medium, large)",
		"disabled":    "Whether the component is disabled",
		"loading":     "Whether the component is in loading state",
		"onClick":     "Click event handler function",
		"className":   "Additional CSS classes to apply",
		"children":    "Child elements to render inside",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}

	return fmt.Sprintf("The %s property of type %s", name, propType)
}

// generatePropExamples creates example values for properties
func (s *PreviewServer) generatePropExamples(name, propType string) []string {
	switch propType {
	case "string":
		return s.getStringExamples(name)
	case "bool":
		return []string{"true", "false"}
	case "int", "int64", "int32":
		return s.getIntExamples(name)
	case "[]string":
		return []string{`["item1", "item2", "item3"]`, `["tag1", "tag2"]`}
	default:
		return []string{fmt.Sprintf(`"example_%s"`, name)}
	}
}

// getStringExamples returns contextual string examples
func (s *PreviewServer) getStringExamples(name string) []string {
	examples := map[string][]string{
		"title":   {`"Welcome"`, `"Dashboard"`, `"Settings"`},
		"text":    {`"Click me"`, `"Get started"`, `"Learn more"`},
		"variant": {`"primary"`, `"secondary"`, `"danger"`},
		"size":    {`"small"`, `"medium"`, `"large"`},
		"color":   {`"blue"`, `"red"`, `"green"`},
	}

	if ex, exists := examples[name]; exists {
		return ex
	}

	return []string{fmt.Sprintf(`"Sample %s"`, name), fmt.Sprintf(`"Another %s"`, name)}
}

// getIntExamples returns contextual integer examples
func (s *PreviewServer) getIntExamples(name string) []string {
	examples := map[string][]string{
		"width":  {"300", "400", "500"},
		"height": {"200", "300", "400"},
		"count":  {"1", "5", "10"},
		"max":    {"10", "50", "100"},
		"min":    {"0", "1", "5"},
	}

	if ex, exists := examples[name]; exists {
		return ex
	}

	return []string{"1", "10", "42"}
}

// generateMockDataSuggestions creates suggestions for mock data
func (s *PreviewServer) generateMockDataSuggestions(
	component *types.ComponentInfo,
) map[string]interface{} {
	suggestions := make(map[string]interface{})

	for _, param := range component.Parameters {
		suggestions[param.Name] = s.generateMultipleMockValues(param.Name, param.Type)
	}

	return suggestions
}

// generateMultipleMockValues creates multiple example values
func (s *PreviewServer) generateMultipleMockValues(name, propType string) interface{} {
	switch propType {
	case "string":
		examples := s.getStringExamples(name)
		values := make([]string, len(examples))
		for i, ex := range examples {
			// Remove quotes from JSON string examples
			values[i] = strings.Trim(ex, `"`)
		}
		return values
	case "int", "int64", "int32":
		examples := s.getIntExamples(name)
		values := make([]int, len(examples))
		for i, ex := range examples {
			if val, err := strconv.Atoi(ex); err == nil {
				values[i] = val
			}
		}
		return values
	case "bool":
		return []bool{true, false}
	case "[]string":
		return [][]string{
			{"Item 1", "Item 2", "Item 3"},
			{"Option A", "Option B"},
			{"Red", "Green", "Blue"},
		}
	default:
		return []string{fmt.Sprintf("mock_%s_1", name), fmt.Sprintf("mock_%s_2", name)}
	}
}

// buildComponentMetadata creates metadata about the component
func (s *PreviewServer) buildComponentMetadata(component *types.ComponentInfo) *ComponentMetadata {
	return &ComponentMetadata{
		Name:         component.Name,
		Package:      component.Package,
		FilePath:     component.FilePath,
		Dependencies: component.Dependencies,
		LastModified: component.LastMod.Format("2006-01-02 15:04:05"),
		DocComments:  s.extractDocComments(component),
	}
}

// extractDocComments extracts documentation from component
func (s *PreviewServer) extractDocComments(component *types.ComponentInfo) string {
	// This would parse the actual file to extract doc comments
	// For now, return a placeholder
	return fmt.Sprintf("Component %s provides interactive UI functionality.", component.Name)
}

// generateComponentCode creates code showing current component usage
func (s *PreviewServer) generateComponentCode(
	componentName string,
	props map[string]interface{},
) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("@%s(", componentName))

	propStrings := make([]string, 0, len(props))
	for key, value := range props {
		propStr := s.formatPropForCode(key, value)
		propStrings = append(propStrings, propStr)
	}

	if len(propStrings) > 0 {
		code.WriteString(strings.Join(propStrings, ", "))
	}

	code.WriteString(")")

	return code.String()
}

// formatPropForCode formats a property value for code display
func (s *PreviewServer) formatPropForCode(key string, value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`%s: "%s"`, key, v)
	case bool:
		return fmt.Sprintf(`%s: %t`, key, v)
	case int, int64, int32:
		return fmt.Sprintf(`%s: %v`, key, v)
	case float64, float32:
		return fmt.Sprintf(`%s: %v`, key, v)
	case []string:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = fmt.Sprintf(`"%s"`, item)
		}
		return fmt.Sprintf(`%s: []string{%s}`, key, strings.Join(items, ", "))
	default:
		return fmt.Sprintf(`%s: %v`, key, v)
	}
}

// createCustomRenderer creates a renderer with custom mock data
func (s *PreviewServer) createCustomRenderer(props map[string]interface{}) *CustomMockRenderer {
	return &CustomMockRenderer{
		baseRenderer: s.renderer,
		customProps:  props,
	}
}

// CustomMockRenderer extends the base renderer with custom prop injection
type CustomMockRenderer struct {
	baseRenderer *renderer.ComponentRenderer
	customProps  map[string]interface{}
}

// RenderComponent renders with injected props (mock implementation for playground)
func (cmr *CustomMockRenderer) RenderComponent(componentName string) (string, error) {
	// For the playground, we generate mock HTML based on the component name and props
	// This allows the playground to work without actual templ compilation
	return cmr.generateMockHTML(componentName, cmr.customProps)
}

// generateMockHTML creates mock HTML representation of a component
func (cmr *CustomMockRenderer) generateMockHTML(
	componentName string,
	props map[string]interface{},
) (string, error) {
	var html strings.Builder

	// Create a representative HTML structure based on component name
	html.WriteString(fmt.Sprintf(`<div class="component %s-component" data-component="%s">`,
		strings.ToLower(componentName), componentName))

	// Add component name as header
	html.WriteString(fmt.Sprintf(`<h3 class="component-title">%s</h3>`, componentName))

	// Render props as content
	if len(props) > 0 {
		html.WriteString(`<div class="component-props">`)

		for key, value := range props {
			html.WriteString(fmt.Sprintf(`<div class="prop-item">
				<strong class="prop-name">%s:</strong> 
				<span class="prop-value">%v</span>
			</div>`, key, value))
		}

		html.WriteString(`</div>`)
	}

	// Add mock content based on component type
	switch strings.ToLower(componentName) {
	case "button":
		text := "Click Me"
		if textProp, exists := props["text"]; exists {
			text = fmt.Sprintf("%v", textProp)
		}
		html.WriteString(fmt.Sprintf(`<button class="btn mock-button">%s</button>`, text))

	case "card":
		title := "Card Title"
		if titleProp, exists := props["title"]; exists {
			title = fmt.Sprintf("%v", titleProp)
		}
		content := "Sample card content"
		if contentProp, exists := props["content"]; exists {
			content = fmt.Sprintf("%v", contentProp)
		}
		html.WriteString(fmt.Sprintf(`
			<div class="card mock-card">
				<div class="card-header">%s</div>
				<div class="card-body">%s</div>
			</div>`, title, content))

	case "input":
		placeholder := "Enter text..."
		if placeholderProp, exists := props["placeholder"]; exists {
			placeholder = fmt.Sprintf("%v", placeholderProp)
		}
		html.WriteString(
			fmt.Sprintf(
				`<input type="text" class="form-input mock-input" placeholder="%s">`,
				placeholder,
			),
		)

	default:
		// Generic component representation
		html.WriteString(fmt.Sprintf(`
			<div class="mock-component-content">
				<p>This is a mock representation of the <code>%s</code> component.</p>
				<p>In a real implementation, this would render the actual templ component.</p>
			</div>`, componentName))
	}

	html.WriteString(`</div>`)

	// Add basic styling
	html.WriteString(`
		<style>
			.component { border: 2px dashed #e2e8f0; padding: 20px; margin: 10px; border-radius: 8px; }
			.component-title { color: #1e293b; margin: 0 0 15px 0; font-size: 1.2em; }
			.component-props { background: #f8fafc; padding: 15px; border-radius: 6px; margin: 15px 0; }
			.prop-item { margin: 5px 0; }
			.prop-name { color: #3b82f6; }
			.prop-value { background: #e0f2fe; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
			.mock-button { background: #3b82f6; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; }
			.mock-card { border: 1px solid #e2e8f0; border-radius: 6px; }
			.card-header { background: #f8fafc; padding: 12px; font-weight: bold; border-bottom: 1px solid #e2e8f0; }
			.card-body { padding: 15px; }
			.mock-input { border: 1px solid #d1d5db; padding: 8px; border-radius: 4px; width: 200px; }
			.mock-component-content { background: #fef3c7; padding: 15px; border-radius: 6px; color: #92400e; }
		</style>`)

	return html.String(), nil
}

// writeJSONResponse writes a JSON response
func (s *PreviewServer) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}
