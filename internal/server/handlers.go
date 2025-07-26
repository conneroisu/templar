package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/types"
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
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(indexHTML)); err != nil {
		log.Printf("Failed to write index response: %v", err)
	}
}

func (s *PreviewServer) handleComponents(w http.ResponseWriter, r *http.Request) {
	components := s.registry.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(components)
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(component)
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

	w.Header().Set("Content-Type", "text/html")
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

	w.Header().Set("Content-Type", "text/html")
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

	w.Header().Set("Content-Type", "text/html")
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

	w.Header().Set("Content-Type", "text/html")
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
