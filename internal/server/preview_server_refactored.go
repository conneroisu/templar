package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/conneroisu/templar/internal/renderer"
)

// RefactoredPreviewServer coordinates all server components following Single Responsibility Principle
// This server acts as a composition root, orchestrating individual focused components
type RefactoredPreviewServer struct {
	// Configuration
	config *config.Config
	
	// Core components (each with single responsibility)
	httpRouter      *HTTPRouter
	wsManager       *WebSocketManager
	middlewareChain *MiddlewareChain
	orchestrator    *ServiceOrchestrator
	
	// Lifecycle management
	shutdownOnce sync.Once
	isShutdown   bool
	shutdownMu   sync.RWMutex
}

// NewRefactoredPreviewServer creates a new refactored server with full dependency injection
func NewRefactoredPreviewServer(
	cfg *config.Config,
	registry interfaces.ComponentRegistry,
	fileWatcher interfaces.FileWatcher,
	scanner interfaces.ComponentScanner,
	buildPipeline interfaces.BuildPipeline,
	monitor *monitoring.TemplarMonitor,
) (*RefactoredPreviewServer, error) {
	
	// Create renderer
	renderer := renderer.NewComponentRenderer(registry)
	
	// Create origin validator (implements OriginValidator interface)
	originValidator := &ServerOriginValidator{config: cfg}
	
	// Create rate limiter
	var rateLimiter *TokenBucketManager
	securityConfig := SecurityConfigFromAppConfig(cfg)
	if securityConfig.RateLimiting != nil && securityConfig.RateLimiting.Enabled {
		rateLimiter = NewRateLimiter(securityConfig.RateLimiting, nil)
	}
	
	// Create WebSocket manager
	wsManager := NewWebSocketManager(originValidator, rateLimiter)
	
	// Create middleware chain
	middlewareChain := NewMiddlewareChain(MiddlewareDependencies{
		Config:          cfg,
		RateLimiter:     rateLimiter,
		Monitor:         monitor,
		OriginValidator: originValidator,
	})
	
	// Create service orchestrator
	orchestrator := NewServiceOrchestrator(ServiceDependencies{
		Config:        cfg,
		Registry:      registry,
		FileWatcher:   fileWatcher,
		Scanner:       scanner,
		BuildPipeline: buildPipeline,
		Renderer:      renderer,
		Monitor:       monitor,
		WSManager:     wsManager,
	})
	
	// Create HTTP handlers adapter that implements HTTPHandlers interface
	handlerAdapter := &ServerHandlerAdapter{
		orchestrator: orchestrator,
		wsManager:    wsManager,
		registry:     registry,
		renderer:     renderer,
		config:       cfg,
	}
	
	// Create HTTP router
	httpRouter := NewHTTPRouter(cfg, handlerAdapter, middlewareChain)
	
	server := &RefactoredPreviewServer{
		config:          cfg,
		httpRouter:      httpRouter,
		wsManager:       wsManager,
		middlewareChain: middlewareChain,
		orchestrator:    orchestrator,
	}
	
	return server, nil
}

// Start starts all server components in coordinated fashion
func (s *RefactoredPreviewServer) Start(ctx context.Context) error {
	s.shutdownMu.RLock()
	if s.isShutdown {
		s.shutdownMu.RUnlock()
		return fmt.Errorf("server has been shut down")
	}
	s.shutdownMu.RUnlock()
	
	// Start service orchestrator (handles business logic coordination)
	if err := s.orchestrator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service orchestrator: %w", err)
	}
	
	// Open browser if configured
	if s.config.Server.Open {
		url := fmt.Sprintf("http://%s", s.httpRouter.GetAddr())
		s.orchestrator.OpenBrowser(url)
	}
	
	log.Printf("🚀 Templar server starting on %s", s.httpRouter.GetAddr())
	
	// Start HTTP router (this blocks until shutdown)
	return s.httpRouter.Start(ctx)
}

// Shutdown gracefully shuts down all server components
func (s *RefactoredPreviewServer) Shutdown(ctx context.Context) error {
	var shutdownErr error
	
	s.shutdownOnce.Do(func() {
		s.shutdownMu.Lock()
		s.isShutdown = true
		s.shutdownMu.Unlock()
		
		log.Printf("Shutting down refactored preview server...")
		
		// Shutdown components in reverse order of startup
		
		// 1. HTTP router
		if err := s.httpRouter.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP router: %v", err)
			shutdownErr = err
		}
		
		// 2. Service orchestrator
		if err := s.orchestrator.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down service orchestrator: %v", err)
			if shutdownErr == nil {
				shutdownErr = err
			}
		}
		
		// 3. WebSocket manager
		if err := s.wsManager.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down WebSocket manager: %v", err)
			if shutdownErr == nil {
				shutdownErr = err
			}
		}
		
		log.Printf("Refactored preview server shut down successfully")
	})
	
	return shutdownErr
}

// GetBuildMetrics returns build metrics through the orchestrator
func (s *RefactoredPreviewServer) GetBuildMetrics() interfaces.BuildMetrics {
	return s.orchestrator.GetBuildMetrics()
}

// GetLastBuildErrors returns last build errors through the orchestrator
func (s *RefactoredPreviewServer) GetLastBuildErrors() []*errors.ParsedError {
	return s.orchestrator.GetLastBuildErrors()
}

// IsShutdown returns whether the server has been shut down
func (s *RefactoredPreviewServer) IsShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.isShutdown
}

// GetStatus returns comprehensive server status
func (s *RefactoredPreviewServer) GetStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	// Server status
	status["server_shutdown"] = s.IsShutdown()
	status["http_router_shutdown"] = s.httpRouter.IsShutdown()
	status["websocket_manager_shutdown"] = s.wsManager.IsShutdown()
	
	// Service status from orchestrator
	serviceStatus := s.orchestrator.GetServiceStatus()
	for key, value := range serviceStatus {
		status[key] = value
	}
	
	// Component metrics
	status["component_count"] = s.orchestrator.GetComponentCount()
	status["websocket_clients"] = s.orchestrator.GetConnectedWebSocketClients()
	
	return status
}

// ServerOriginValidator implements OriginValidator interface for the server
type ServerOriginValidator struct {
	config *config.Config
}

// IsAllowedOrigin checks if the origin is allowed for WebSocket connections
func (sov *ServerOriginValidator) IsAllowedOrigin(origin string) bool {
	if origin == "" {
		return true // Allow same-origin requests
	}
	
	// Development environment allows more origins
	if sov.config.Server.Environment == "development" {
		allowedOrigins := []string{
			fmt.Sprintf("http://localhost:%d", sov.config.Server.Port),
			fmt.Sprintf("http://127.0.0.1:%d", sov.config.Server.Port),
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
	}
	
	// Production: only allow same-origin
	expectedOrigin := fmt.Sprintf("http://%s:%d", sov.config.Server.Host, sov.config.Server.Port)
	return origin == expectedOrigin
}

// ServerHandlerAdapter adapts the server's handler methods to the HTTPHandlers interface
// This allows clean separation between HTTP routing and business logic
type ServerHandlerAdapter struct {
	orchestrator *ServiceOrchestrator
	wsManager    *WebSocketManager
	registry     interfaces.ComponentRegistry
	renderer     *renderer.ComponentRenderer
	config       *config.Config
}

// Implement all HTTPHandlers interface methods by delegating to existing handlers
// These methods will delegate to the existing handler implementations in handlers.go

func (sha *ServerHandlerAdapter) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sha.wsManager.HandleWebSocket(w, r)
}

func (sha *ServerHandlerAdapter) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleHealthCheck(w, r, sha.orchestrator)
}

func (sha *ServerHandlerAdapter) HandleComponents(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleComponentsList(w, r, sha.registry)
}

func (sha *ServerHandlerAdapter) HandleComponent(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleComponentDetail(w, r, sha.registry, sha.renderer)
}

func (sha *ServerHandlerAdapter) HandleRender(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleComponentRender(w, r, sha.registry, sha.renderer)
}

func (sha *ServerHandlerAdapter) HandleStatic(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleStaticFiles(w, r)
}

func (sha *ServerHandlerAdapter) HandlePlaygroundIndex(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handlePlaygroundIndexPage(w, r)
}

func (sha *ServerHandlerAdapter) HandlePlaygroundComponent(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handlePlaygroundComponentPage(w, r, sha.registry, sha.renderer)
}

func (sha *ServerHandlerAdapter) HandlePlaygroundRender(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handlePlaygroundRenderAPI(w, r, sha.registry, sha.renderer)
}

func (sha *ServerHandlerAdapter) HandleEnhancedIndex(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleEnhancedInterface(w, r, sha.registry)
}

func (sha *ServerHandlerAdapter) HandleEditorIndex(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleEditorInterface(w, r)
}

func (sha *ServerHandlerAdapter) HandleEditorAPI(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleEditorAPI(w, r)
}

func (sha *ServerHandlerAdapter) HandleFileAPI(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleFileAPI(w, r)
}

func (sha *ServerHandlerAdapter) HandleInlineEditor(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleInlineEditor(w, r)
}

func (sha *ServerHandlerAdapter) HandleBuildStatus(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleBuildStatus(w, r, sha.orchestrator)
}

func (sha *ServerHandlerAdapter) HandleBuildMetrics(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleBuildMetrics(w, r, sha.orchestrator)
}

func (sha *ServerHandlerAdapter) HandleBuildErrors(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleBuildErrors(w, r, sha.orchestrator)
}

func (sha *ServerHandlerAdapter) HandleBuildCache(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleBuildCache(w, r, sha.orchestrator)
}

func (sha *ServerHandlerAdapter) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleIndexPage(w, r, sha.registry)
}

func (sha *ServerHandlerAdapter) HandleTargetFiles(w http.ResponseWriter, r *http.Request) {
	// Delegate to existing handler - will need to be extracted from original server
	handleTargetFilesPage(w, r, sha.config, sha.registry, sha.renderer)
}

// TODO: The handler implementations above need to be extracted from the original server.go
// and placed into separate handler functions that take dependencies as parameters.
// This maintains clean separation of concerns and allows for proper unit testing.