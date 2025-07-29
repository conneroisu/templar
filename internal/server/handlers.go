package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/version"
)

const indexHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Templar - Component Preview</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#007acc',
                        secondary: '#6c757d'
                    }
                }
            }
        }
    </script>
    <style>
        body { 
            font-family: system-ui, -apple-system, sans-serif; 
            margin: 0; 
            padding: 20px; 
            background: #f5f5f5; 
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            background: white; 
            padding: 20px; 
            border-radius: 8px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1); 
        }
        h1 { 
            color: #333; 
            border-bottom: 2px solid #007acc; 
            padding-bottom: 10px; 
        }
        .component-list { 
            display: grid; 
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); 
            gap: 20px; 
            margin-top: 20px; 
        }
        .component-card { 
            border: 1px solid #ddd; 
            border-radius: 6px; 
            padding: 15px; 
            background: #fafafa; 
        }
        .component-name { 
            font-weight: bold; 
            font-size: 16px; 
            color: #007acc; 
        }
        .component-path { 
            font-size: 12px; 
            color: #666; 
            margin-top: 5px; 
        }
        .component-params { 
            margin-top: 10px; 
            font-size: 12px; 
        }
        .status { 
            position: fixed; 
            top: 20px; 
            right: 20px; 
            padding: 10px 20px; 
            border-radius: 4px; 
            color: white; 
            font-weight: bold; 
            z-index: 1000; 
        }
        .status.connected { background: #28a745; }
        .status.disconnected { background: #dc3545; }
        .status.error { background: #ffc107; color: #333; }
        .component-card {
            transition: transform 0.2s ease-in-out, box-shadow 0.2s ease-in-out;
        }
        .component-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 20px rgba(0,0,0,0.1);
        }
        .fade-in {
            animation: fadeIn 0.5s ease-in;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
    </style>
</head>
<body class="bg-gray-50">
    <div class="container mx-auto max-w-6xl p-6">
        <div class="bg-white rounded-lg shadow-lg p-6">
            <h1 class="text-3xl font-bold text-gray-800 border-b-2 border-primary pb-4 mb-6">
                Templar Component Preview
            </h1>
            <div class="mb-6 flex gap-4">
                <a href="/enhanced" class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary-600 transition-colors font-medium">
                    🛠️ Enhanced Interface
                </a>
                <a href="/playground" class="bg-secondary-500 text-white px-4 py-2 rounded-lg hover:bg-secondary-600 transition-colors font-medium">
                    🎮 Component Playground
                </a>
                <a href="/editor" class="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors font-medium">
                    ✏️ Interactive Editor
                </a>
            </div>
            <div id="status" class="status disconnected fixed top-4 right-4 px-4 py-2 rounded-lg text-white font-semibold z-50">
                Disconnected
            </div>
            <div id="components" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-6">
                <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 animate-pulse">
                    <div class="text-blue-600 font-medium">Loading components...</div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        let ws;
        let reconnectInterval;
        
        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/ws');
            
            ws.onopen = function() {
                document.getElementById('status').className = 'status connected';
                document.getElementById('status').textContent = 'Connected';
                clearInterval(reconnectInterval);
                loadComponents();
            };
            
            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                handleMessage(message);
            };
            
            ws.onclose = function() {
                document.getElementById('status').className = 'status disconnected';
                document.getElementById('status').textContent = 'Disconnected';
                
                // Try to reconnect
                reconnectInterval = setInterval(connect, 2000);
            };
            
            ws.onerror = function(error) {
                document.getElementById('status').className = 'status error';
                document.getElementById('status').textContent = 'Error';
                console.error('WebSocket error:', error);
            };
        }
        
        function handleMessage(message) {
            switch(message.type) {
                case 'full_reload':
                    window.location.reload();
                    break;
                case 'component_update':
                    loadComponents();
                    break;
                case 'css_update':
                    updateCSS(message.content);
                    break;
            }
        }
        
        function loadComponents() {
            fetch('/components')
                .then(response => response.json())
                .then(components => {
                    const container = document.getElementById('components');
                    if (Object.keys(components).length === 0) {
                        container.innerHTML = 
                            '<div class="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">' +
                            '<div class="text-yellow-800 font-medium text-lg mb-2">No components found</div>' +
                            '<div class="text-yellow-600 text-sm">Create a .templ file to get started</div>' +
                            '<div class="text-yellow-500 text-xs mt-2">Watching: ' + window.location.origin + '</div>' +
                            '</div>';
                        return;
                    }
                    
                    container.innerHTML = '';
                    Object.values(components).forEach(component => {
                        const card = document.createElement('div');
                        card.className = 'component-card bg-white border border-gray-200 rounded-lg p-4 shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer fade-in';
                        
                        const params = component.parameters || [];
                        const paramsList = params.map(p => p.name + ': ' + p.type).join(', ');
                        
                        card.innerHTML = 
                            '<div class="component-name text-lg font-semibold text-primary mb-2">' + component.name + '</div>' +
                            '<div class="component-path text-sm text-gray-500 mb-3 truncate">' + component.filePath + '</div>' +
                            '<div class="component-params text-xs text-gray-600 bg-gray-50 rounded p-2">' +
                            '<span class="font-medium">Parameters:</span> ' + (paramsList || 'none') +
                            '</div>' +
                            '<div class="mt-3 text-xs text-gray-400">Package: ' + (component.package || 'unknown') + '</div>';
                        
                        container.appendChild(card);
                    });
                })
                .catch(error => {
                    console.error('Failed to load components:', error);
                    document.getElementById('components').innerHTML = 
                        '<div class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">' +
                        '<div class="font-medium">Error loading components</div>' +
                        '<div class="text-sm mt-1">Check the console for details</div>' +
                        '</div>';
                });
        }
        
        function updateCSS(content) {
            // Update CSS without full reload
            const style = document.createElement('style');
            style.textContent = content;
            document.head.appendChild(style);
        }
        
        // Initialize connection
        connect();
        
        // Save page state for preservation
        window.addEventListener('beforeunload', function() {
            window.__templarState = {
                scroll: { x: window.scrollX, y: window.scrollY }
            };
        });
    </script>
</body>
</html>`

func (s *PreviewServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(indexHTML)); err != nil {
		log.Printf("Failed to write index response: %v", err)
	}
}

func (s *PreviewServer) handleComponents(w http.ResponseWriter, r *http.Request) {
	components := s.registry.GetAll()

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(components); err != nil {
		log.Printf("Failed to encode components JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *PreviewServer) handleComponent(w http.ResponseWriter, r *http.Request) {
	// Extract component name from path
	path := strings.TrimPrefix(r.URL.Path, "/component/")
	componentName := strings.Split(path, "/")[0]

	// Validate component name to prevent path traversal and injection attacks
	if err := validateComponentName(componentName); err != nil {
		http.Error(w, "Invalid component name: "+err.Error(), http.StatusBadRequest)

		return
	}

	component, exists := s.registry.Get(componentName)
	if !exists {
		http.NotFound(w, r)

		return
	}

	// For now, just return component info
	// In a full implementation, this would render the component
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(component); err != nil {
		log.Printf("Failed to encode component JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *PreviewServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Handle static files
	// For now, just return 404
	http.NotFound(w, r)
}

// renderComponentPage renders an individual component page

func (s *PreviewServer) handleTargetFiles(w http.ResponseWriter, r *http.Request) {
	// When specific files are targeted, show a file selection interface
	if len(s.config.TargetFiles) == 1 {
		// Single file - try to find and render its first component
		s.handleSingleFile(w, r, s.config.TargetFiles[0])

		return
	}

	// Multiple files - show selection interface
	s.handleMultipleFiles(w, r)
}

func (s *PreviewServer) handleSingleFile(w http.ResponseWriter, r *http.Request, filename string) {
	// Check if scanner is available
	if s.scanner == nil {
		http.Error(w, "Scanner not initialized", http.StatusInternalServerError)

		return
	}

	// Scan the specific file to find components
	if err := s.scanner.ScanFile(filename); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error scanning file %s: %v", filename, err),
			http.StatusInternalServerError,
		)

		return
	}

	// Get all components from this file
	allComponents := s.registry.GetAll()
	var fileComponents []*types.ComponentInfo

	for _, component := range allComponents {
		if component.FilePath == filename {
			fileComponents = append(fileComponents, component)
		}
	}

	if len(fileComponents) == 0 {
		http.Error(w, "No components found in file "+filename, http.StatusNotFound)

		return
	}

	// If only one component, render it directly
	if len(fileComponents) == 1 {
		s.renderSingleComponent(w, r, fileComponents[0])

		return
	}

	// Multiple components - show selection
	s.renderComponentSelection(w, r, fileComponents, filename)
}

func (s *PreviewServer) handleMultipleFiles(w http.ResponseWriter, r *http.Request) {
	// Check if scanner is available
	if s.scanner == nil {
		http.Error(w, "Scanner not initialized", http.StatusInternalServerError)

		return
	}

	// Scan all target files
	for _, filename := range s.config.TargetFiles {
		if err := s.scanner.ScanFile(filename); err != nil {
			log.Printf("Error scanning file %s: %v", filename, err)
		}
	}

	// Show file selection interface
	s.renderFileSelection(w, r)
}

func (s *PreviewServer) handleRender(w http.ResponseWriter, r *http.Request) {
	// Extract component name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/render/")
	componentName := strings.Split(path, "/")[0]

	if componentName == "" {
		http.Error(w, "Component name required", http.StatusBadRequest)

		return
	}

	// Render the component
	html, err := s.renderer.RenderComponent(componentName)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error rendering component %s: %v", componentName, err),
			http.StatusInternalServerError,
		)

		return
	}

	// Get nonce from request context for CSP
	nonce := GetNonceFromContext(r.Context())

	// Wrap in layout with nonce support
	fullHTML := s.renderer.RenderComponentWithLayoutAndNonce(componentName, html, nonce)

	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(fullHTML)); err != nil {
		log.Printf("Failed to write component response: %v", err)
	}
}

func (s *PreviewServer) renderSingleComponent(
	w http.ResponseWriter,
	r *http.Request,
	component *types.ComponentInfo,
) {
	// Render the component directly
	html, err := s.renderer.RenderComponent(component.Name)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error rendering component %s: %v", component.Name, err),
			http.StatusInternalServerError,
		)

		return
	}

	// Get nonce from request context for CSP
	nonce := GetNonceFromContext(r.Context())

	// Wrap in layout with nonce support
	fullHTML := s.renderer.RenderComponentWithLayoutAndNonce(component.Name, html, nonce)

	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(fullHTML)); err != nil {
		log.Printf("Failed to write component response: %v", err)
	}
}

func (s *PreviewServer) renderComponentSelection(
	w http.ResponseWriter,
	r *http.Request,
	components []*types.ComponentInfo,
	filename string,
) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Select Component - %s</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 p-8">
    <div class="max-w-2xl mx-auto">
        <h1 class="text-2xl font-bold mb-6">Select Component from %s</h1>
        <div class="grid gap-4">`, filename, filename)

	for _, component := range components {
		html += fmt.Sprintf(`
            <a href="/render/%s" class="bg-white rounded-lg shadow p-4 hover:shadow-md transition-shadow">
                <h2 class="text-lg font-semibold text-blue-600">%s</h2>
                <p class="text-gray-600 text-sm mt-1">%d parameters</p>
            </a>`, component.Name, component.Name, len(component.Parameters))
	}

	html += `
        </div>
    </div>
</body>
</html>`

	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Failed to write component selection response: %v", err)
	}
}

func (s *PreviewServer) renderFileSelection(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Select File - Templar</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 p-8">
    <div class="max-w-2xl mx-auto">
        <h1 class="text-2xl font-bold mb-6">Select File to Preview</h1>
        <div class="grid gap-4">`

	for _, filename := range s.config.TargetFiles {
		html += fmt.Sprintf(`
            <a href="/?file=%s" class="bg-white rounded-lg shadow p-4 hover:shadow-md transition-shadow">
                <h2 class="text-lg font-semibold text-blue-600">%s</h2>
                <p class="text-gray-600 text-sm mt-1">Templ file</p>
            </a>`, filename, filename)
	}

	html += `
        </div>
    </div>
</body>
</html>`

	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Failed to write file selection response: %v", err)
	}
}

// validateComponentName validates component name to prevent security issues.
func validateComponentName(name string) error {
	// Reject empty names
	if name == "" {
		return errors.New("empty component name")
	}

	// Clean the name
	cleanName := filepath.Clean(name)

	// Reject names containing path traversal patterns
	if strings.Contains(cleanName, "..") {
		return errors.New("path traversal attempt detected")
	}

	// Reject absolute paths
	if filepath.IsAbs(cleanName) {
		return errors.New("absolute path not allowed")
	}

	// Reject special characters that could be used in injection attacks (check first for security)
	dangerousChars := []string{
		"<",
		">",
		"\"",
		"'",
		"&",
		";",
		"|",
		"$",
		"`",
		"(",
		")",
		"{",
		"}",
		"[",
		"]",
		"\\",
	}
	for _, char := range dangerousChars {
		if strings.Contains(cleanName, char) {
			return fmt.Errorf("dangerous character not allowed: %s", char)
		}
	}

	// Reject names with path separators (should be simple component names)
	if strings.ContainsRune(cleanName, os.PathSeparator) {
		return errors.New("path separators not allowed in component name")
	}

	// Reject if name is too long (prevent buffer overflow attacks)
	if len(cleanName) > 100 {
		return errors.New("component name too long (max 100 characters)")
	}

	return nil
}

// handleAPIDocs serves the API documentation interface.
func (s *PreviewServer) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate and serve OpenAPI documentation
	s.serveAPIDocumentation(w, r)
}

// handleAPISpec serves the OpenAPI specification.
func (s *PreviewServer) handleAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine format from query parameter or Accept header
	format := r.URL.Query().Get("format")
	if format == "" {
		// Default to JSON
		format = "json"
	}

	switch format {
	case "json":
		s.serveOpenAPIJSON(w, r)
	case "yaml":
		s.serveOpenAPIYAML(w, r)
	default:
		http.Error(w, "Unsupported format. Use ?format=json or ?format=yaml", http.StatusBadRequest)
	}
}

// serveAPIDocumentation serves the interactive API documentation interface.
func (s *PreviewServer) serveAPIDocumentation(w http.ResponseWriter, r *http.Request) {
	// Get nonce from request context for CSP
	nonce := GetNonceFromContext(r.Context())

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Templar API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style nonce="%s">
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        
        *, *:before, *:after {
            box-sizing: inherit;
        }
        
        body {
            margin: 0;
            background: #fafafa;
        }
        
        .swagger-ui .topbar {
            background-color: #007acc;
        }
        
        .swagger-ui .topbar .download-url-wrapper .select-label {
            color: #fff;
        }
        
        .swagger-ui .info .title {
            color: #007acc;
        }
        
        .swagger-ui .btn.authorize {
            background-color: #007acc;
            border-color: #007acc;
        }
        
        .swagger-ui .btn.authorize:hover {
            background-color: #005a99;
            border-color: #005a99;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    
    <script nonce="%s" src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script nonce="%s" src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script nonce="%s">
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api/spec?format=json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                tryItOutEnabled: true,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                },
                onFailure: function(data) {
                    console.error('Failed to load Swagger UI:', data);
                }
            });
        };
    </script>
</body>
</html>`, nonce, nonce, nonce, nonce)

	w.Header().Set(HeaderContentType, ContentTypeHTML)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Failed to write API documentation response: %v", err)
	}
}

// serveOpenAPIJSON serves the OpenAPI specification in JSON format.
func (s *PreviewServer) serveOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	spec, err := s.generateOpenAPISpec()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if err := json.NewEncoder(w).Encode(spec); err != nil {
		log.Printf("Failed to encode OpenAPI JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// serveOpenAPIYAML serves the OpenAPI specification in YAML format.
func (s *PreviewServer) serveOpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	spec, err := s.generateOpenAPISpec()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	// Convert to YAML (would need yaml package)
	// For now, return JSON with a note
	// TODO: Use the spec variable to generate actual YAML output
	_ = spec // Suppress unused variable warning
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.Write([]byte(`{"error": "YAML format not yet implemented. Use ?format=json"}`))
}

// generateOpenAPISpec generates the OpenAPI specification dynamically.
func (s *PreviewServer) generateOpenAPISpec() (interface{}, error) {
	// Determine API version from server configuration or version package
	apiVersion := s.getAPIVersion()
	
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "Templar API",
			"description": "REST API for the Templar component development server providing component management, rendering, and development tools.",
			"version":     apiVersion,
			"contact": map[string]interface{}{
				"name":  "Templar Team",
				"url":   "https://github.com/conneroisu/templar",
				"email": "support@templar.dev",
			},
			"license": map[string]interface{}{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
			"termsOfService": "https://github.com/conneroisu/templar/blob/main/LICENSE",
		},
		"servers": []map[string]interface{}{
			{
				"url":         fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port),
				"description": "Development server",
			},
		},
		"paths": s.generatePaths(),
		"components": map[string]interface{}{
			"schemas": s.generateSchemas(),
		},
		"tags": []map[string]interface{}{
			{"name": "system", "description": "System health and status endpoints"},
			{"name": "components", "description": "Component management and discovery"},
			{"name": "build", "description": "Build system monitoring and control"},
			{"name": "playground", "description": "Interactive component playground"},
			{"name": "editor", "description": "Component editing interface"},
			{"name": "websocket", "description": "Real-time communication"},
		},
	}

	return spec, nil
}

// generatePaths generates the OpenAPI paths specification.
func (s *PreviewServer) generatePaths() map[string]interface{} {
	return map[string]interface{}{
		"/health": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"system"},
				"summary":     "Health check endpoint",
				"description": "Returns the current health status of the server and its components",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Server is healthy",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/HealthStatus",
								},
							},
						},
					},
				},
			},
		},
		"/components": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"components"},
				"summary":     "List all components",
				"description": "Returns a list of all discovered templ components",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "List of components",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"$ref": "#/components/schemas/ComponentInfo",
									},
								},
							},
						},
					},
				},
			},
		},
		"/component/{name}": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"components"},
				"summary":     "Get component details",
				"description": "Returns detailed information about a specific component",
				"parameters": []map[string]interface{}{
					{
						"name":        "name",
						"in":          "path",
						"required":    true,
						"description": "The component name",
						"schema": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Component details",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ComponentInfo",
								},
							},
						},
					},
					"404": map[string]interface{}{
						"description": "Component not found",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/Error",
								},
							},
						},
					},
				},
			},
		},
		"/api/build/status": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"build"},
				"summary":     "Get build status",
				"description": "Returns the current build system status and metrics",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Build status information",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/BuildStatus",
								},
							},
						},
					},
				},
			},
		},
		"/ws": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"websocket"},
				"summary":     "WebSocket connection",
				"description": "Establishes a WebSocket connection for real-time updates",
				"responses": map[string]interface{}{
					"101": map[string]interface{}{
						"description": "WebSocket connection established",
					},
					"403": map[string]interface{}{
						"description": "Forbidden - invalid origin",
					},
				},
			},
		},
	}
}

// generateSchemas generates the OpenAPI schemas specification.
func (s *PreviewServer) generateSchemas() map[string]interface{} {
	return map[string]interface{}{
		"ComponentInfo": map[string]interface{}{
			"type":        "object",
			"description": "Information about a templ component",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Component name",
					"example":     "Button",
				},
				"filePath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the component file",
					"example":     "/components/button.templ",
				},
				"package": map[string]interface{}{
					"type":        "string",
					"description": "Go package name",
					"example":     "components",
				},
				"parameters": map[string]interface{}{
					"type":        "array",
					"description": "Component parameters",
					"items": map[string]interface{}{
						"$ref": "#/components/schemas/ParameterInfo",
					},
				},
			},
			"required": []string{"name", "filePath"},
		},
		"ParameterInfo": map[string]interface{}{
			"type":        "object",
			"description": "Information about a component parameter",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Parameter name",
					"example":     "text",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Parameter type",
					"example":     "string",
				},
			},
			"required": []string{"name", "type"},
		},
		"HealthStatus": map[string]interface{}{
			"type":        "object",
			"description": "Server health status",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Overall health status",
					"enum":        []string{"healthy", "unhealthy"},
					"example":     "healthy",
				},
				"timestamp": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Health check timestamp",
				},
				"version": map[string]interface{}{
					"type":        "string",
					"description": "Application version",
					"example":     "1.0.0",
				},
			},
			"required": []string{"status", "timestamp"},
		},
		"BuildStatus": map[string]interface{}{
			"type":        "object",
			"description": "Build system status",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Build status",
					"enum":        []string{"healthy", "error"},
					"example":     "healthy",
				},
				"total_builds": map[string]interface{}{
					"type":        "integer",
					"description": "Total number of builds",
					"minimum":     0,
				},
				"failed_builds": map[string]interface{}{
					"type":        "integer",
					"description": "Number of failed builds",
					"minimum":     0,
				},
			},
		},
		"Error": map[string]interface{}{
			"type":        "object",
			"description": "Error response",
			"properties": map[string]interface{}{
				"error": map[string]interface{}{
					"type":        "string",
					"description": "Error message",
					"example":     "Component not found",
				},
				"code": map[string]interface{}{
					"type":        "integer",
					"description": "Error code",
					"example":     404,
				},
			},
			"required": []string{"error"},
		},
	}
}

// getAPIVersion returns the current API version.
func (s *PreviewServer) getAPIVersion() string {
	// Use version package to get application version
	return version.GetVersion()
}

// handleAPIVersions serves available API versions.
func (s *PreviewServer) handleAPIVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	versions := map[string]interface{}{
		"versions": []map[string]interface{}{
			{
				"version":     "1.0.0",
				"status":      "stable",
				"released":    "2025-07-29",
				"deprecated":  false,
				"docs_url":    "/api/docs",
				"spec_url":    "/api/spec",
				"changelog":   "https://github.com/conneroisu/templar/releases/tag/v1.0.0",
			},
		},
		"current": "1.0.0",
		"latest":  "1.0.0",
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(versions); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
