package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// Router handles HTTP server lifecycle and route registration
// Following Single Responsibility Principle: manages HTTP routing concerns only
//
// Design Principles:
// - Single Responsibility: Only manages HTTP routing and server lifecycle
// - Dependency Injection: All handlers injected through Handlers interface
// - Thread Safety: All operations protected by serverMutex for concurrent access
// - Graceful Shutdown: Supports context-based cancellation and graceful termination
// - Extensibility: Supports custom route registration for plugins and extensions
//
// Invariants:
// - config must never be nil after construction
// - mux must never be nil after construction  
// - handlers must never be nil after construction
// - httpServer is nil only before Start() or after Shutdown()
// - isShutdown is write-protected by serverMutex
type Router struct {
	config     *config.Config    // Configuration for server binding and behavior
	httpServer *http.Server      // Underlying HTTP server instance (nil until Start())
	mux        *http.ServeMux    // Route multiplexer for handling HTTP requests
	
	// Server state management - all access must be protected by serverMutex
	serverMutex sync.RWMutex     // Protects httpServer and isShutdown fields
	isShutdown  bool             // Indicates if server has been shut down
	
	// Handler dependencies - injected via constructor to maintain testability
	handlers Handlers        // Interface providing all HTTP handler implementations
}

// Handlers interface defines all HTTP handler dependencies
// This allows for clean dependency injection and testability
type Handlers interface {
	// WebSocket handlers
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	
	// API handlers
	HandleHealth(w http.ResponseWriter, r *http.Request)
	HandleComponents(w http.ResponseWriter, r *http.Request)
	HandleComponent(w http.ResponseWriter, r *http.Request)
	HandleRender(w http.ResponseWriter, r *http.Request)
	HandleStatic(w http.ResponseWriter, r *http.Request)
	
	// Playground handlers
	HandlePlaygroundIndex(w http.ResponseWriter, r *http.Request)
	HandlePlaygroundComponent(w http.ResponseWriter, r *http.Request)
	HandlePlaygroundRender(w http.ResponseWriter, r *http.Request)
	
	// Enhanced interface handlers
	HandleEnhancedIndex(w http.ResponseWriter, r *http.Request)
	
	// Editor handlers
	HandleEditorIndex(w http.ResponseWriter, r *http.Request)
	HandleEditorAPI(w http.ResponseWriter, r *http.Request)
	HandleFileAPI(w http.ResponseWriter, r *http.Request)
	HandleInlineEditor(w http.ResponseWriter, r *http.Request)
	
	// Build API handlers
	HandleBuildStatus(w http.ResponseWriter, r *http.Request)
	HandleBuildMetrics(w http.ResponseWriter, r *http.Request)
	HandleBuildErrors(w http.ResponseWriter, r *http.Request)
	HandleBuildCache(w http.ResponseWriter, r *http.Request)
	
	// Index handlers
	HandleIndex(w http.ResponseWriter, r *http.Request)
	HandleTargetFiles(w http.ResponseWriter, r *http.Request)
}

// MiddlewareProvider interface for middleware chain injection
type MiddlewareProvider interface {
	Apply(handler http.Handler) http.Handler
}

// NewRouter creates a new HTTP router with dependency injection
// 
// This constructor follows the dependency injection pattern to ensure:
// - All dependencies are explicitly provided and validated
// - The router is fully initialized and ready for use
// - Route registration is centralized and deterministic
//
// Parameters:
// - config: Server configuration (host, port, target files)
// - handlers: Implementation of all HTTP handlers  
// - middlewareProvider: Middleware chain to apply to all routes
//
// Returns:
// - Fully initialized Router ready for Start()
//
// Panics:
// - If any required dependency is nil
// - If config contains invalid values
func NewRouter(
	config *config.Config,
	handlers Handlers,
	middlewareProvider MiddlewareProvider,
) *Router {
	// Critical assertions - these conditions must hold for safe operation
	if config == nil {
		panic("Router: config cannot be nil")
	}
	if handlers == nil {
		panic("Router: handlers cannot be nil") 
	}
	if middlewareProvider == nil {
		panic("Router: middlewareProvider cannot be nil")
	}
	
	// Validate configuration values
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		panic(fmt.Sprintf("Router: invalid port %d, must be 1-65535", config.Server.Port))
	}
	if config.Server.Host == "" {
		panic("Router: host cannot be empty")
	}
	
	// Initialize router with validated dependencies
	router := &Router{
		config:   config,        // Store configuration reference
		mux:      http.NewServeMux(), // Create new request multiplexer
		handlers: handlers,      // Store handler interface
		isShutdown: false,       // Router starts in active state
	}
	
	// Register all routes using centralized registration
	// This must happen before server creation to ensure all routes are available
	router.registerRoutes()
	
	// Create HTTP server with complete middleware chain applied
	// The middleware provider handles security, logging, CORS, rate limiting, etc.
	handler := middlewareProvider.Apply(router.mux)
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	
	// Thread-safe server initialization
	router.serverMutex.Lock()
	router.httpServer = &http.Server{
		Addr:    addr,          // Bind address from configuration
		Handler: handler,       // Handler with complete middleware stack
	}
	router.serverMutex.Unlock()
	
	// Post-construction invariant checks
	if router.mux == nil {
		panic("Router: mux initialization failed")
	}
	if router.httpServer == nil {
		panic("Router: httpServer initialization failed")
	}
	
	return router
}

// registerRoutes registers all HTTP routes with their handlers
// Centralized route registration following REST API conventions
func (r *Router) registerRoutes() {
	// WebSocket endpoint
	r.mux.HandleFunc("/ws", r.handlers.HandleWebSocket)
	
	// Health check endpoint
	r.mux.HandleFunc("/health", r.handlers.HandleHealth)
	
	// Component API endpoints
	r.mux.HandleFunc("/components", r.handlers.HandleComponents)
	r.mux.HandleFunc("/component/", r.handlers.HandleComponent)
	r.mux.HandleFunc("/render/", r.handlers.HandleRender)
	r.mux.HandleFunc("/static/", r.handlers.HandleStatic)
	
	// Playground endpoints
	r.mux.HandleFunc("/playground", r.handlers.HandlePlaygroundIndex)
	r.mux.HandleFunc("/playground/", r.handlers.HandlePlaygroundComponent)
	r.mux.HandleFunc("/api/playground/render", r.handlers.HandlePlaygroundRender)
	
	// Enhanced Web Interface endpoints
	r.mux.HandleFunc("/enhanced", r.handlers.HandleEnhancedIndex)
	
	// Interactive Editor endpoints
	r.mux.HandleFunc("/editor", r.handlers.HandleEditorIndex)
	r.mux.HandleFunc("/editor/", r.handlers.HandleEditorIndex)
	r.mux.HandleFunc("/api/editor", r.handlers.HandleEditorAPI)
	r.mux.HandleFunc("/api/files", r.handlers.HandleFileAPI)
	r.mux.HandleFunc("/api/inline-editor", r.handlers.HandleInlineEditor)
	
	// Build API endpoints
	r.mux.HandleFunc("/api/build/status", r.handlers.HandleBuildStatus)
	r.mux.HandleFunc("/api/build/metrics", r.handlers.HandleBuildMetrics)
	r.mux.HandleFunc("/api/build/errors", r.handlers.HandleBuildErrors)
	r.mux.HandleFunc("/api/build/cache", r.handlers.HandleBuildCache)
	
	// Root handler - depends on configuration
	if len(r.config.TargetFiles) > 0 {
		r.mux.HandleFunc("/", r.handlers.HandleTargetFiles)
	} else {
		r.mux.HandleFunc("/", r.handlers.HandleIndex)
	}
}

// Start starts the HTTP server and blocks until context cancellation or server error
//
// This method implements graceful startup with the following guarantees:
// - Server starts listening on the configured address immediately
// - Context cancellation triggers graceful shutdown
// - Server errors are properly propagated to caller
// - Method is safe for concurrent access
//
// Parameters:
// - ctx: Context for cancellation and timeout control
//
// Returns:
// - nil if server shut down gracefully due to context cancellation
// - error if server failed to start or encountered runtime error
//
// Thread Safety:
// - Safe for concurrent calls (though multiple calls don't make sense)
// - Uses read lock for server access to avoid blocking shutdown
func (r *Router) Start(ctx context.Context) error {
	// Precondition checks
	if ctx == nil {
		return fmt.Errorf("Router.Start: context cannot be nil")
	}
	
	// Thread-safe access to server instance
	r.serverMutex.RLock()
	server := r.httpServer
	isShutdown := r.isShutdown
	r.serverMutex.RUnlock()
	
	// Assertion: server must be initialized by constructor
	if server == nil {
		return fmt.Errorf("Router.Start: server not initialized (call NewRouter first)")
	}
	
	// Cannot start an already shut down router
	if isShutdown {
		return fmt.Errorf("Router.Start: router has been shut down")
	}
	
	// Start server in separate goroutine to enable context-based cancellation
	// This pattern allows the server to run concurrently while monitoring for cancellation
	errChan := make(chan error, 1)
	go func() {
		// ListenAndServe blocks until server stops or encounters error
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// http.ErrServerClosed is expected during graceful shutdown
			errChan <- fmt.Errorf("Router: server error: %w", err)
		}
		// Normal shutdown (http.ErrServerClosed) doesn't send to errChan
	}()
	
	// Block until either:
	// 1. Context is cancelled (graceful shutdown requested)
	// 2. Server encounters an error (unexpected failure)
	select {
	case <-ctx.Done():
		// Context cancellation - initiate graceful shutdown
		// Use background context to avoid cancellation during shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return r.Shutdown(shutdownCtx)
		
	case err := <-errChan:
		// Server error - propagate to caller
		return err
	}
}

// Shutdown gracefully shuts down the HTTP server
//
// This method implements graceful shutdown with the following guarantees:
// - Ongoing requests are allowed to complete within context timeout
// - New connections are rejected immediately
// - Method is idempotent (safe to call multiple times)
// - Thread-safe for concurrent access
//
// Parameters:
// - ctx: Context for shutdown timeout control (recommended: 30s timeout)
//
// Returns:
// - nil if shutdown completed successfully or was already shut down
// - error if shutdown failed or context timeout exceeded
//
// Implementation Notes:
// - Uses write lock to ensure atomic state transition
// - Sets isShutdown flag to prevent future operations
// - Delegates actual shutdown to http.Server.Shutdown for proper connection draining
func (r *Router) Shutdown(ctx context.Context) error {
	// Precondition checks
	if ctx == nil {
		return fmt.Errorf("Router.Shutdown: context cannot be nil")
	}
	
	// Acquire write lock for atomic shutdown state transition
	r.serverMutex.Lock()
	defer r.serverMutex.Unlock()
	
	// Idempotent: if already shut down, return success immediately
	if r.isShutdown {
		return nil
	}
	
	// Mark as shut down to prevent future operations
	// This must be done before actual shutdown to prevent race conditions
	r.isShutdown = true
	
	// Perform actual graceful shutdown if server exists
	if r.httpServer != nil {
		// Delegate to http.Server.Shutdown for proper connection draining
		// This will:
		// - Stop accepting new connections
		// - Close idle connections  
		// - Wait for active connections to complete within context timeout
		if err := r.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("Router.Shutdown: server shutdown failed: %w", err)
		}
	}
	
	// Successful shutdown
	return nil
}

// GetAddr returns the server address
func (r *Router) GetAddr() string {
	r.serverMutex.RLock()
	defer r.serverMutex.RUnlock()
	
	if r.httpServer != nil {
		return r.httpServer.Addr
	}
	
	return fmt.Sprintf("%s:%d", r.config.Server.Host, r.config.Server.Port)
}

// IsShutdown returns whether the router has been shut down
func (r *Router) IsShutdown() bool {
	r.serverMutex.RLock()
	defer r.serverMutex.RUnlock()
	return r.isShutdown
}

// RegisterHealthCheck adds a health check route for monitoring
func (r *Router) RegisterHealthCheck(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, handler)
}

// RegisterCustomRoute allows adding custom routes for plugins or extensions
func (r *Router) RegisterCustomRoute(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}