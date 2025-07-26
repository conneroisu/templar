package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/monitoring"
)

// MiddlewareChain manages the HTTP middleware stack following the Chain of Responsibility pattern
// Following Single Responsibility Principle: manages middleware composition and execution only
//
// Design Principles:
// - Single Responsibility: Only manages middleware composition, ordering, and application
// - Chain of Responsibility: Middlewares can be added, removed, and reordered dynamically
// - Dependency Injection: All middleware dependencies injected at construction
// - Immutable Application: Apply() creates new handler chain without modifying original
// - Extensibility: Supports custom middleware injection for plugins and extensions
//
// Middleware Execution Order:
// - Middlewares execute in REVERSE order of addition (last added, first executed)
// - Outer middlewares wrap inner middlewares (onion model)
// - Request flows: Outer -> Middle -> Inner -> Handler
// - Response flows: Handler -> Inner -> Middle -> Outer
//
// Standard Middleware Stack (outer to inner):
// 1. Logging & Monitoring (request/response tracking)
// 2. CORS (cross-origin request handling)
// 3. Rate Limiting (request throttling)
// 4. Security (headers, validation)
// 5. Authentication (user verification)
//
// Invariants:
// - config must never be nil after construction
// - middlewares slice is never nil (can be empty)
// - middleware execution order is deterministic
// - Apply() is safe for concurrent access (read-only operation)
type MiddlewareChain struct {
	config          *config.Config             // Application configuration for middleware behavior
	rateLimiter     *RateLimiter               // Global rate limiter (optional)
	monitor         *monitoring.TemplarMonitor // Monitoring system (optional)
	originValidator OriginValidator            // Origin validation for CORS
	middlewares     []Middleware               // Ordered list of middleware functions
}

// Middleware represents a single middleware function
type Middleware func(http.Handler) http.Handler

// MiddlewareDependencies contains all dependencies needed for middleware construction
type MiddlewareDependencies struct {
	Config          *config.Config
	RateLimiter     *RateLimiter
	Monitor         *monitoring.TemplarMonitor
	OriginValidator OriginValidator
}

// NewMiddlewareChain creates a new middleware chain with dependency injection
//
// This constructor builds a complete middleware chain with the standard stack:
// - Validates all required dependencies
// - Initializes middleware storage
// - Builds the default security-first middleware stack
// - Ensures proper execution order for optimal security and performance
//
// The resulting middleware chain provides:
// - Request/response logging and monitoring
// - CORS handling for cross-origin requests
// - Rate limiting for DoS protection
// - Security headers and validation
// - Authentication and authorization
//
// Parameters:
// - deps: Struct containing all middleware dependencies
//
// Returns:
// - Fully configured MiddlewareChain ready for Apply()
//
// Panics:
// - If required dependencies are nil or invalid
func NewMiddlewareChain(deps MiddlewareDependencies) *MiddlewareChain {
	// Critical dependency validation - these are required for safe operation
	if deps.Config == nil {
		panic("MiddlewareChain: config cannot be nil")
	}
	if deps.OriginValidator == nil {
		panic("MiddlewareChain: originValidator cannot be nil (required for CORS security)")
	}

	// Validate configuration has required fields
	if deps.Config.Server.Environment == "" {
		panic("MiddlewareChain: config.Server.Environment cannot be empty")
	}

	// Initialize chain with validated dependencies
	chain := &MiddlewareChain{
		config:          deps.Config,              // Application configuration
		rateLimiter:     deps.RateLimiter,         // Optional global rate limiter
		monitor:         deps.Monitor,             // Optional monitoring system
		originValidator: deps.OriginValidator,     // Required origin validator
		middlewares:     make([]Middleware, 0, 8), // Pre-allocate for typical middleware count
	}

	// Build the default middleware stack with proper ordering
	// This must happen during construction to ensure consistent behavior
	chain.buildDefaultStack()

	// Post-construction invariant validation
	if len(chain.middlewares) == 0 {
		panic("MiddlewareChain: default stack build failed - no middlewares added")
	}
	if chain.middlewares == nil {
		panic("MiddlewareChain: middleware slice initialization failed")
	}

	return chain
}

// buildDefaultStack constructs the standard middleware stack
func (mc *MiddlewareChain) buildDefaultStack() {
	// Order matters: middlewares are executed in reverse order (last added, first executed)

	// 1. Logging and monitoring (outermost - first to execute, last to complete)
	mc.AddMiddleware(mc.createLoggingMiddleware())

	// 2. CORS handling
	mc.AddMiddleware(mc.createCORSMiddleware())

	// 3. Monitoring middleware (if available)
	if mc.monitor != nil {
		monitoringMiddleware := mc.monitor.CreateTemplarMiddleware()
		mc.AddMiddleware(monitoringMiddleware)
	}

	// 4. Rate limiting middleware (if enabled)
	if mc.shouldEnableRateLimit() {
		if mc.rateLimiter == nil {
			mc.rateLimiter = mc.createRateLimiter()
		}
		mc.AddMiddleware(RateLimitMiddleware(mc.rateLimiter))
	}

	// 5. Security middleware
	securityConfig := SecurityConfigFromAppConfig(mc.config)
	mc.AddMiddleware(SecurityMiddleware(securityConfig))

	// 6. Authentication middleware (innermost - last to execute, first to complete)
	mc.AddMiddleware(AuthMiddleware(&mc.config.Server.Auth))
}

// AddMiddleware adds a middleware to the chain
func (mc *MiddlewareChain) AddMiddleware(middleware Middleware) {
	mc.middlewares = append(mc.middlewares, middleware)
}

// AddMiddlewareAt inserts a middleware at a specific position in the chain
func (mc *MiddlewareChain) AddMiddlewareAt(index int, middleware Middleware) {
	if index < 0 || index > len(mc.middlewares) {
		mc.middlewares = append(mc.middlewares, middleware)
		return
	}

	// Insert at specific position
	mc.middlewares = append(mc.middlewares[:index+1], mc.middlewares[index:]...)
	mc.middlewares[index] = middleware
}

// RemoveMiddleware removes a specific middleware by comparison
func (mc *MiddlewareChain) RemoveMiddleware(targetMiddleware Middleware) {
	for i, middleware := range mc.middlewares {
		// Simple address comparison - could be enhanced with interface-based matching
		if &middleware == &targetMiddleware {
			mc.middlewares = append(mc.middlewares[:i], mc.middlewares[i+1:]...)
			break
		}
	}
}

// Apply applies the entire middleware chain to a handler
//
// This method implements the Chain of Responsibility pattern by wrapping
// the provided handler with all registered middlewares in the correct order.
//
// Execution Flow:
// 1. Middlewares are applied in REVERSE order (onion model)
// 2. Last added middleware becomes the outermost wrapper
// 3. First added middleware becomes the innermost wrapper
// 4. Request flows through outer -> inner middlewares
// 5. Response flows through inner -> outer middlewares
//
// Example with middlewares [A, B, C] and handler H:
// - Chain order: A -> B -> C -> H
// - Execution: A(B(C(H)))
// - Request flow: A -> B -> C -> H
// - Response flow: H -> C -> B -> A
//
// Thread Safety:
// - Safe for concurrent access (read-only operation)
// - Does not modify the middleware chain state
// - Creates new handler chain for each invocation
//
// Parameters:
// - handler: The base HTTP handler to wrap with middlewares
//
// Returns:
// - New http.Handler with complete middleware chain applied
//
// Panics:
// - If handler is nil (programming error)
func (mc *MiddlewareChain) Apply(handler http.Handler) http.Handler {
	// Precondition validation
	if handler == nil {
		panic("MiddlewareChain.Apply: handler cannot be nil")
	}

	// Defensive copy check - ensure middlewares slice is valid
	if mc.middlewares == nil {
		panic("MiddlewareChain.Apply: middlewares slice is nil (initialization error)")
	}

	// Apply middlewares in reverse order to create onion-style wrapping
	// This ensures the first added middleware is closest to the handler
	// and the last added middleware is the outermost wrapper
	wrappedHandler := handler
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		middleware := mc.middlewares[i]

		// Validate each middleware before application
		if middleware == nil {
			panic(fmt.Sprintf("MiddlewareChain.Apply: middleware at index %d is nil", i))
		}

		// Apply middleware to create new wrapped handler
		wrappedHandler = middleware(wrappedHandler)

		// Validate middleware didn't return nil handler
		if wrappedHandler == nil {
			panic(
				fmt.Sprintf(
					"MiddlewareChain.Apply: middleware at index %d returned nil handler",
					i,
				),
			)
		}
	}

	// Return the fully wrapped handler
	return wrappedHandler
}

// createLoggingMiddleware creates the logging and request tracking middleware
func (mc *MiddlewareChain) createLoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Execute next handler
			next.ServeHTTP(w, r)

			duration := time.Since(start)

			// Track request in monitoring system
			if mc.monitor != nil {
				mc.monitor.RecordWebSocketEvent("http_request", 1)
			}

			// Log request
			log.Printf("%s %s %v", r.Method, r.URL.Path, duration)
		})
	}
}

// createCORSMiddleware creates the CORS handling middleware
func (mc *MiddlewareChain) createCORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS headers based on environment
			origin := r.Header.Get("Origin")

			if mc.originValidator.ValidateOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if mc.config.Server.Environment == "development" {
				// Only allow wildcard in development
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			// Production default: no CORS header (blocks cross-origin requests)

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Continue to next middleware
			next.ServeHTTP(w, r)
		})
	}
}

// shouldEnableRateLimit determines if rate limiting should be enabled
func (mc *MiddlewareChain) shouldEnableRateLimit() bool {
	securityConfig := SecurityConfigFromAppConfig(mc.config)
	rateLimitConfig := securityConfig.RateLimiting
	return rateLimitConfig != nil && rateLimitConfig.Enabled
}

// createRateLimiter creates a new rate limiter instance
func (mc *MiddlewareChain) createRateLimiter() *RateLimiter {
	// Create a basic rate limit config
	rateConfig := RateLimit{
		RequestsPerMinute: 60,
		BurstLimit:        10,
	}
	return NewRateLimiter(rateConfig)
}

// GetMiddlewareCount returns the number of middlewares in the chain
func (mc *MiddlewareChain) GetMiddlewareCount() int {
	return len(mc.middlewares)
}

// Reset clears all middlewares and rebuilds the default stack
func (mc *MiddlewareChain) Reset() {
	mc.middlewares = make([]Middleware, 0)
	mc.buildDefaultStack()
}

// Clone creates a copy of the middleware chain
func (mc *MiddlewareChain) Clone() *MiddlewareChain {
	clone := &MiddlewareChain{
		config:          mc.config,
		rateLimiter:     mc.rateLimiter,
		monitor:         mc.monitor,
		originValidator: mc.originValidator,
		middlewares:     make([]Middleware, len(mc.middlewares)),
	}

	copy(clone.middlewares, mc.middlewares)

	return clone
}

// MiddlewareConfig provides configuration for middleware components
type MiddlewareConfig struct {
	EnableLogging     bool
	EnableCORS        bool
	EnableRateLimit   bool
	EnableSecurity    bool
	EnableAuth        bool
	EnableMonitoring  bool
	CustomMiddlewares []Middleware
}

// NewCustomMiddlewareChain creates a middleware chain with custom configuration
func NewCustomMiddlewareChain(
	deps MiddlewareDependencies,
	config MiddlewareConfig,
) *MiddlewareChain {
	chain := &MiddlewareChain{
		config:          deps.Config,
		rateLimiter:     deps.RateLimiter,
		monitor:         deps.Monitor,
		originValidator: deps.OriginValidator,
		middlewares:     make([]Middleware, 0),
	}

	// Build stack based on configuration
	if config.EnableLogging {
		chain.AddMiddleware(chain.createLoggingMiddleware())
	}

	if config.EnableCORS {
		chain.AddMiddleware(chain.createCORSMiddleware())
	}

	if config.EnableMonitoring && chain.monitor != nil {
		chain.AddMiddleware(chain.monitor.CreateTemplarMiddleware())
	}

	if config.EnableRateLimit && chain.shouldEnableRateLimit() {
		if chain.rateLimiter == nil {
			chain.rateLimiter = chain.createRateLimiter()
		}
		chain.AddMiddleware(RateLimitMiddleware(chain.rateLimiter))
	}

	if config.EnableSecurity {
		securityConfig := SecurityConfigFromAppConfig(chain.config)
		chain.AddMiddleware(SecurityMiddleware(securityConfig))
	}

	if config.EnableAuth {
		chain.AddMiddleware(AuthMiddleware(&chain.config.Server.Auth))
	}

	// Add custom middlewares
	for _, middleware := range config.CustomMiddlewares {
		chain.AddMiddleware(middleware)
	}

	return chain
}

// DebugMiddlewares logs information about all middlewares in the chain (for debugging)
func (mc *MiddlewareChain) DebugMiddlewares() {
	log.Printf("Middleware chain contains %d middlewares:", len(mc.middlewares))
	for i := range mc.middlewares {
		log.Printf("  %d: Middleware at position %d", i, i)
	}
}
