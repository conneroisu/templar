package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/renderer"
)

// This file contains handler delegate functions that implement the actual HTTP handler logic
// These functions are called by the ServerHandlerAdapter to maintain separation of concerns

// handleHealthCheck handles health check requests
func handleHealthCheck(w http.ResponseWriter, r *http.Request, orchestrator *ServiceOrchestrator) {
	w.Header().Set("Content-Type", "application/json")
	
	status := orchestrator.GetServiceStatus()
	status["healthy"] = orchestrator.IsHealthy()
	
	response, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal health status", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleComponentsList handles requests for the components list
func handleComponentsList(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry) {
	w.Header().Set("Content-Type", "application/json")
	
	components := registry.GetAll()
	
	response, err := json.MarshalIndent(components, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal components", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleComponentDetail handles requests for individual component details
func handleComponentDetail(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry, renderer *renderer.ComponentRenderer) {
	// Extract component name from URL path
	path := r.URL.Path
	componentName := path[len("/component/"):]
	
	if componentName == "" {
		http.Error(w, "Component name required", http.StatusBadRequest)
		return
	}
	
	component, exists := registry.Get(componentName)
	if !exists {
		http.Error(w, "Component not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	response, err := json.MarshalIndent(component, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal component", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleComponentRender handles component rendering requests
func handleComponentRender(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry, renderer *renderer.ComponentRenderer) {
	// Extract component name from URL path
	path := r.URL.Path
	componentName := path[len("/render/"):]
	
	if componentName == "" {
		http.Error(w, "Component name required", http.StatusBadRequest)
		return
	}
	
	component, exists := registry.Get(componentName)
	if !exists {
		http.Error(w, "Component not found", http.StatusNotFound)
		return
	}
	
	// For now, return a placeholder response
	// TODO: Integrate with actual renderer implementation
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<h1>Rendered Component: %s</h1><p>File: %s</p>", 
		component.Name, component.FilePath)))
}

// handleStaticFiles handles static file requests
func handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Basic static file serving
	http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
}

// handlePlaygroundIndexPage handles playground index page
func handlePlaygroundIndexPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<html>
		<head><title>Component Playground</title></head>
		<body>
			<h1>Component Playground</h1>
			<p>Interactive component testing environment</p>
		</body>
		</html>
	`))
}

// handlePlaygroundComponentPage handles individual playground component pages
func handlePlaygroundComponentPage(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry, renderer *renderer.ComponentRenderer) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<html>
		<head><title>Playground Component</title></head>
		<body>
			<h1>Component Playground</h1>
			<p>Component testing interface</p>
		</body>
		</html>
	`))
}

// handlePlaygroundRenderAPI handles playground render API requests
func handlePlaygroundRenderAPI(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry, renderer *renderer.ComponentRenderer) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "rendered", "message": "Playground render complete"}`))
}

// handleEnhancedInterface handles enhanced web interface requests
func handleEnhancedInterface(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<html>
		<head><title>Enhanced Interface</title></head>
		<body>
			<h1>Enhanced Web Interface</h1>
			<p>Advanced component management interface</p>
		</body>
		</html>
	`))
}

// handleEditorInterface handles editor interface requests
func handleEditorInterface(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<html>
		<head><title>Component Editor</title></head>
		<body>
			<h1>Component Editor</h1>
			<p>Interactive component editing interface</p>
		</body>
		</html>
	`))
}

// handleEditorAPI handles editor API requests
func handleEditorAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Editor API ready"}`))
}

// handleFileAPI handles file management API requests
func handleFileAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "File API ready"}`))
}

// handleInlineEditor handles inline editor requests
func handleInlineEditor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Inline editor ready"}`))
}

// handleBuildStatus handles build status API requests
func handleBuildStatus(w http.ResponseWriter, r *http.Request, orchestrator *ServiceOrchestrator) {
	w.Header().Set("Content-Type", "application/json")
	
	buildErrors := orchestrator.GetLastBuildErrors()
	status := map[string]interface{}{
		"hasErrors": len(buildErrors) > 0,
		"errorCount": len(buildErrors),
		"healthy": len(buildErrors) == 0,
	}
	
	response, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal build status", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleBuildMetrics handles build metrics API requests
func handleBuildMetrics(w http.ResponseWriter, r *http.Request, orchestrator *ServiceOrchestrator) {
	w.Header().Set("Content-Type", "application/json")
	
	metrics := orchestrator.GetBuildMetrics()
	
	// Convert interface to map for JSON serialization
	metricsMap := map[string]interface{}{
		"build_count":      metrics.GetBuildCount(),
		"success_count":    metrics.GetSuccessCount(),
		"failure_count":    metrics.GetFailureCount(),
		"average_duration": metrics.GetAverageDuration(),
		"cache_hit_rate":   metrics.GetCacheHitRate(),
		"success_rate":     metrics.GetSuccessRate(),
	}
	
	response, err := json.MarshalIndent(metricsMap, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal build metrics", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleBuildErrors handles build errors API requests
func handleBuildErrors(w http.ResponseWriter, r *http.Request, orchestrator *ServiceOrchestrator) {
	w.Header().Set("Content-Type", "application/json")
	
	buildErrors := orchestrator.GetLastBuildErrors()
	
	response, err := json.MarshalIndent(buildErrors, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal build errors", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleBuildCache handles build cache API requests
func handleBuildCache(w http.ResponseWriter, r *http.Request, orchestrator *ServiceOrchestrator) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status": "ok",
		"message": "Build cache management ready",
	}
	
	response, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal cache status", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// handleIndexPage handles the main index page
func handleIndexPage(w http.ResponseWriter, r *http.Request, registry interfaces.ComponentRegistry) {
	w.Header().Set("Content-Type", "text/html")
	
	componentCount := registry.Count()
	
	html := fmt.Sprintf(`
		<html>
		<head><title>Templar - Component Preview Server</title></head>
		<body>
			<h1>Templar Component Preview Server</h1>
			<p>Server is running with %d components registered.</p>
			<h2>Available Endpoints:</h2>
			<ul>
				<li><a href="/components">Components List</a></li>
				<li><a href="/playground">Component Playground</a></li>
				<li><a href="/enhanced">Enhanced Interface</a></li>
				<li><a href="/editor">Component Editor</a></li>
				<li><a href="/health">Health Check</a></li>
			</ul>
		</body>
		</html>
	`, componentCount)
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleTargetFilesPage handles target files page
func handleTargetFilesPage(w http.ResponseWriter, r *http.Request, config *config.Config, registry interfaces.ComponentRegistry, renderer *renderer.ComponentRenderer) {
	w.Header().Set("Content-Type", "text/html")
	
	html := fmt.Sprintf(`
		<html>
		<head><title>Templar - Target Files</title></head>
		<body>
			<h1>Target Files Mode</h1>
			<p>Serving target files: %v</p>
			<p>Component count: %d</p>
		</body>
		</html>
	`, config.TargetFiles, registry.Count())
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}