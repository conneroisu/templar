package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/validation"
	"github.com/conneroisu/templar/internal/version"
	"github.com/conneroisu/templar/internal/watcher"
	"nhooyr.io/websocket"
)

// Client represents a WebSocket client
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	server *PreviewServer
}

// PreviewServer serves components with live reload capability
type PreviewServer struct {
	config          *config.Config
	httpServer      *http.Server
	serverMutex     sync.RWMutex // Protects httpServer and server state
	clients         map[*websocket.Conn]*Client
	clientsMutex    sync.RWMutex
	broadcast       chan []byte
	register        chan *Client
	unregister      chan *websocket.Conn
	registry        *registry.ComponentRegistry
	watcher         *watcher.FileWatcher
	scanner         *scanner.ComponentScanner
	renderer        *renderer.ComponentRenderer
	buildPipeline   *build.BuildPipeline
	lastBuildErrors []*errors.ParsedError
	shutdownOnce    sync.Once
	isShutdown      bool
	shutdownMutex   sync.RWMutex
}

// UpdateMessage represents a message sent to the browser
type UpdateMessage struct {
	Type      string    `json:"type"`
	Target    string    `json:"target,omitempty"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new preview server
func New(cfg *config.Config) (*PreviewServer, error) {

	registry := registry.NewComponentRegistry()

	fileWatcher, err := watcher.NewFileWatcher(300 * time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	scanner := scanner.NewComponentScanner(registry)
	renderer := renderer.NewComponentRenderer(registry)

	// Create build pipeline
	buildPipeline := build.NewBuildPipeline(4, registry)

	return &PreviewServer{
		config:          cfg,
		clients:         make(map[*websocket.Conn]*Client),
		broadcast:       make(chan []byte),
		register:        make(chan *Client),
		unregister:      make(chan *websocket.Conn),
		registry:        registry,
		watcher:         fileWatcher,
		scanner:         scanner,
		renderer:        renderer,
		buildPipeline:   buildPipeline,
		lastBuildErrors: make([]*errors.ParsedError, 0),
	}, nil
}

// Start starts the preview server
func (s *PreviewServer) Start(ctx context.Context) error {
	// Set up file watcher
	s.setupFileWatcher(ctx)

	// Start build pipeline
	s.buildPipeline.Start(ctx)

	// Add build callback to handle errors and updates
	s.buildPipeline.AddCallback(s.handleBuildResult)

	// Initial scan
	if err := s.initialScan(); err != nil {
		log.Printf("Initial scan failed: %v", err)
	}

	// Start WebSocket hub
	go s.runWebSocketHub(ctx)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/components", s.handleComponents)
	mux.HandleFunc("/component/", s.handleComponent)
	mux.HandleFunc("/render/", s.handleRender)
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/api/build/status", s.handleBuildStatus)
	mux.HandleFunc("/api/build/metrics", s.handleBuildMetrics)
	mux.HandleFunc("/api/build/errors", s.handleBuildErrors)
	mux.HandleFunc("/api/build/cache", s.handleBuildCache)

	// Root handler depends on whether specific files are targeted
	if len(s.config.TargetFiles) > 0 {
		mux.HandleFunc("/", s.handleTargetFiles)
	} else {
		mux.HandleFunc("/", s.handleIndex)
	}

	// Add middleware
	handler := s.addMiddleware(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.serverMutex.Lock()
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	server := s.httpServer // Get local copy for safe access
	s.serverMutex.Unlock()

	// Open browser if configured
	if s.config.Server.Open {
		go s.openBrowser(fmt.Sprintf("http://%s", addr))
	}

	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *PreviewServer) setupFileWatcher(ctx context.Context) {
	// Add filters
	s.watcher.AddFilter(watcher.TemplFilter)
	s.watcher.AddFilter(watcher.GoFilter)
	s.watcher.AddFilter(watcher.NoTestFilter)
	s.watcher.AddFilter(watcher.NoVendorFilter)
	s.watcher.AddFilter(watcher.NoGitFilter)

	// Add handler
	s.watcher.AddHandler(s.handleFileChange)

	// Add watch paths
	for _, path := range s.config.Components.ScanPaths {
		if err := s.watcher.AddRecursive(path); err != nil {
			log.Printf("Failed to watch path %s: %v", path, err)
		}
	}

	// Start watching
	if err := s.watcher.Start(ctx); err != nil {
		log.Printf("Failed to start file watcher: %v", err)
	}
}

func (s *PreviewServer) initialScan() error {
	log.Printf("Starting initial scan with paths: %v", s.config.Components.ScanPaths)
	for _, path := range s.config.Components.ScanPaths {
		log.Printf("Scanning path: %s", path)
		if err := s.scanner.ScanDirectory(path); err != nil {
			log.Printf("Error scanning %s: %v", path, err)
			// Don't return error, just log and continue
			continue
		}
	}

	log.Printf("Found %d components", s.registry.Count())
	return nil
}

func (s *PreviewServer) handleFileChange(events []watcher.ChangeEvent) error {
	componentsToRebuild := make(map[string]*registry.ComponentInfo)

	for _, event := range events {
		log.Printf("File changed: %s (%s)", event.Path, event.Type)

		// Rescan the file
		if err := s.scanner.ScanFile(event.Path); err != nil {
			log.Printf("Failed to rescan file %s: %v", event.Path, err)
		}

		// Find components in the changed file
		components := s.registry.GetAll()
		for _, component := range components {
			if component.FilePath == event.Path {
				componentsToRebuild[component.Name] = component
			}
		}
	}

	// Queue components for rebuild
	for _, component := range componentsToRebuild {
		s.buildPipeline.BuildWithPriority(component)
	}

	// If no specific components to rebuild, do a full rebuild
	if len(componentsToRebuild) == 0 {
		s.triggerFullRebuild()
	}

	return nil
}

func (s *PreviewServer) openBrowser(url string) {
	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Validate URL for security before passing to system commands
	if err := validation.ValidateURL(url); err != nil {
		log.Printf("Browser open failed due to invalid URL: %v", err)
		return
	}

	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func (s *PreviewServer) addMiddleware(handler http.Handler) http.Handler {
	// Create security middleware
	securityConfig := SecurityConfigFromAppConfig(s.config)
	securityHandler := SecurityMiddleware(securityConfig)(handler)
	
	// Create rate limiting middleware
	rateLimitConfig := securityConfig.RateLimiting
	if rateLimitConfig != nil && rateLimitConfig.Enabled {
		rateLimiter := NewRateLimiter(rateLimitConfig, nil)
		rateLimitHandler := RateLimitMiddleware(rateLimiter)(securityHandler)
		securityHandler = rateLimitHandler
	}
	
	// Add CORS and logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers based on environment
		origin := r.Header.Get("Origin")
		if s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if s.config.Server.Environment == "development" {
			// Only allow wildcard in development
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		// Production default: no CORS header (blocks cross-origin requests)

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log requests
		start := time.Now()
		securityHandler.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// isAllowedOrigin checks if the origin is in the allowed origins list
func (s *PreviewServer) isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	// Check configured allowed origins
	for _, allowed := range s.config.Server.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}

func (s *PreviewServer) broadcastMessage(msg UpdateMessage) {
	// Marshal message to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		// Fallback to simple reload message
		s.broadcast <- []byte(`{"type":"full_reload"}`)
		return
	}

	s.broadcast <- jsonData
}

// handleBuildResult handles build results from the build pipeline
func (s *PreviewServer) handleBuildResult(result build.BuildResult) {
	if result.Error != nil {
		// Store build errors
		s.lastBuildErrors = result.ParsedErrors

		// Broadcast error message
		msg := UpdateMessage{
			Type:      "build_error",
			Content:   errors.FormatErrorsForBrowser(result.ParsedErrors),
			Timestamp: time.Now(),
		}
		s.broadcastMessage(msg)
	} else {
		// Clear previous errors
		s.lastBuildErrors = make([]*errors.ParsedError, 0)

		// Broadcast success message
		msg := UpdateMessage{
			Type:      "build_success",
			Target:    result.Component.Name,
			Timestamp: time.Now(),
		}
		s.broadcastMessage(msg)
	}
}

// triggerFullRebuild triggers a full rebuild of all components
func (s *PreviewServer) triggerFullRebuild() {
	components := s.registry.GetAll()
	for _, component := range components {
		s.buildPipeline.Build(component)
	}
}

// GetBuildMetrics returns the current build metrics
func (s *PreviewServer) GetBuildMetrics() build.BuildMetrics {
	return s.buildPipeline.GetMetrics()
}

// GetLastBuildErrors returns the last build errors
func (s *PreviewServer) GetLastBuildErrors() []*errors.ParsedError {
	return s.lastBuildErrors
}

// Shutdown gracefully shuts down the server and cleans up resources
func (s *PreviewServer) Shutdown(ctx context.Context) error {
	var shutdownErr error
	
	s.shutdownOnce.Do(func() {
		log.Println("Shutting down server...")

		// Mark as shutdown to prevent new operations
		s.shutdownMutex.Lock()
		s.isShutdown = true
		s.shutdownMutex.Unlock()

		// Stop build pipeline first
		if s.buildPipeline != nil {
			s.buildPipeline.Stop()
		}

		// Stop file watcher
		if s.watcher != nil {
			s.watcher.Stop()
		}

		// Close all WebSocket connections
		s.clientsMutex.Lock()
		for conn, client := range s.clients {
			close(client.send)
			conn.Close(websocket.StatusNormalClosure, "")
		}
		s.clients = make(map[*websocket.Conn]*Client)
		s.clientsMutex.Unlock()

		// Close channels safely
		select {
		case <-s.broadcast:
		default:
			close(s.broadcast)
		}
		
		select {
		case <-s.register:
		default:
			close(s.register)
		}
		
		select {
		case <-s.unregister:
		default:
			close(s.unregister)
		}

		// Shutdown HTTP server
		s.serverMutex.RLock()
		server := s.httpServer
		s.serverMutex.RUnlock()

		if server != nil {
			shutdownErr = server.Shutdown(ctx)
		}
	})

	return shutdownErr
}

// handleHealth returns the server health status for health checks
func (s *PreviewServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   version.GetShortVersion(),
		"build_info": version.GetBuildInfo(),
		"checks": map[string]interface{}{
			"server":   map[string]interface{}{"status": "healthy", "message": "HTTP server operational"},
			"registry": map[string]interface{}{"status": "healthy", "components": len(s.registry.GetAll())},
			"watcher":  map[string]interface{}{"status": "healthy", "message": "File watcher operational"},
			"build":    map[string]interface{}{"status": "healthy", "message": "Build pipeline operational"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// handleBuildStatus returns the current build status
func (s *PreviewServer) handleBuildStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.buildPipeline.GetMetrics()
	errors := s.GetLastBuildErrors()

	status := "healthy"
	if len(errors) > 0 {
		status = "error"
	}

	response := map[string]interface{}{
		"status":        status,
		"total_builds":  metrics.TotalBuilds,
		"failed_builds": metrics.FailedBuilds,
		"cache_hits":    metrics.CacheHits,
		"errors":        len(errors),
		"timestamp":     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBuildMetrics returns detailed build metrics
func (s *PreviewServer) handleBuildMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.buildPipeline.GetMetrics()
	cacheCount, cacheSize, cacheMaxSize := s.buildPipeline.GetCacheStats()

	response := map[string]interface{}{
		"build_metrics": map[string]interface{}{
			"total_builds":      metrics.TotalBuilds,
			"successful_builds": metrics.SuccessfulBuilds,
			"failed_builds":     metrics.FailedBuilds,
			"cache_hits":        metrics.CacheHits,
			"average_duration":  metrics.AverageDuration.String(),
			"total_duration":    metrics.TotalDuration.String(),
		},
		"cache_metrics": map[string]interface{}{
			"entries":    cacheCount,
			"size_bytes": cacheSize,
			"max_size":   cacheMaxSize,
			"hit_rate":   float64(metrics.CacheHits) / float64(metrics.TotalBuilds),
		},
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBuildErrors returns the last build errors
func (s *PreviewServer) handleBuildErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	errors := s.GetLastBuildErrors()

	response := map[string]interface{}{
		"errors":    errors,
		"count":     len(errors),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBuildCache manages the build cache
func (s *PreviewServer) handleBuildCache(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return cache statistics
		count, size, maxSize := s.buildPipeline.GetCacheStats()
		response := map[string]interface{}{
			"entries":    count,
			"size_bytes": size,
			"max_size":   maxSize,
			"timestamp":  time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Clear cache
		s.buildPipeline.ClearCache()

		response := map[string]interface{}{
			"message":   "Cache cleared successfully",
			"timestamp": time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
