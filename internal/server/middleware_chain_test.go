package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conneroisu/templar/internal/config"
)

// MockOriginValidatorForMiddleware provides mock origin validation for middleware tests
type MockOriginValidatorForMiddleware struct {
	AllowedOrigins []string
	AllowAll       bool
}

func (m *MockOriginValidatorForMiddleware) IsAllowedOrigin(origin string) bool {
	if m.AllowAll {
		return true
	}
	for _, allowed := range m.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// MockTokenBucketManagerForMiddleware provides mock rate limiting for middleware tests
type MockTokenBucketManagerForMiddleware struct {
	AllowRequests bool
}

func (m *MockTokenBucketManagerForMiddleware) Check(key string) RateLimitResult {
	return RateLimitResult{
		Allowed:   m.AllowRequests,
		Remaining: 100,
	}
}

func (m *MockTokenBucketManagerForMiddleware) Stop() {
	// Mock implementation
}

// createTestMiddlewareDependencies creates valid dependencies for testing
func createTestMiddlewareDependencies() MiddlewareDependencies {
	return MiddlewareDependencies{
		Config: &config.Config{
			Server: config.ServerConfig{
				Environment: "test",
				AllowedOrigins: []string{"https://example.com"},
			},
		},
		RateLimiter:     nil, // Use nil to avoid type issues - middleware handles nil gracefully
		Monitor:         nil, // Optional
		OriginValidator: &MockOriginValidatorForMiddleware{AllowAll: true},
	}
}

// TestNewMiddlewareChain_ValidInputs tests successful construction
func TestNewMiddlewareChain_ValidInputs(t *testing.T) {
	deps := createTestMiddlewareDependencies()

	chain := NewMiddlewareChain(deps)

	// Verify construction succeeded
	if chain == nil {
		t.Fatal("NewMiddlewareChain returned nil")
	}

	// Verify dependencies were stored
	if chain.config != deps.Config {
		t.Error("Config was not stored correctly")
	}
	// Note: rateLimiter may be created by middleware chain if rate limiting is enabled in config
	// so we don't check for exact equality here
	if chain.monitor != deps.Monitor {
		t.Error("Monitor was not stored correctly")
	}
	if chain.originValidator != deps.OriginValidator {
		t.Error("OriginValidator was not stored correctly")
	}

	// Verify middleware initialization
	if chain.middlewares == nil {
		t.Error("Middlewares slice was not initialized")
	}
	if len(chain.middlewares) == 0 {
		t.Error("Default middleware stack was not built")
	}

	// Should have at least logging, CORS, security, and auth middlewares
	expectedMinCount := 4
	if len(chain.middlewares) < expectedMinCount {
		t.Errorf("Expected at least %d middlewares, got %d", expectedMinCount, len(chain.middlewares))
	}
}

// TestNewMiddlewareChain_NilConfig tests panic on nil config
func TestNewMiddlewareChain_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil config, but didn't panic")
		}
	}()

	deps := createTestMiddlewareDependencies()
	deps.Config = nil
	NewMiddlewareChain(deps)
}

// TestNewMiddlewareChain_NilOriginValidator tests panic on nil origin validator
func TestNewMiddlewareChain_NilOriginValidator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil origin validator, but didn't panic")
		}
	}()

	deps := createTestMiddlewareDependencies()
	deps.OriginValidator = nil
	NewMiddlewareChain(deps)
}

// TestNewMiddlewareChain_EmptyEnvironment tests panic on empty environment
func TestNewMiddlewareChain_EmptyEnvironment(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty environment, but didn't panic")
		}
	}()

	deps := createTestMiddlewareDependencies()
	deps.Config.Server.Environment = ""
	NewMiddlewareChain(deps)
}

// TestMiddlewareChain_Apply_ValidHandler tests successful middleware application
func TestMiddlewareChain_Apply_ValidHandler(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Apply middleware chain
	wrappedHandler := chain.Apply(testHandler)

	// Verify wrapped handler is not nil
	if wrappedHandler == nil {
		t.Fatal("Apply returned nil handler")
	}

	// Test the wrapped handler
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	// Should complete successfully
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// TestMiddlewareChain_Apply_NilHandler tests panic on nil handler
func TestMiddlewareChain_Apply_NilHandler(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil handler, but didn't panic")
		}
	}()

	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)
	chain.Apply(nil)
}

// TestMiddlewareChain_AddMiddleware tests adding custom middleware
func TestMiddlewareChain_AddMiddleware(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	initialCount := chain.GetMiddlewareCount()

	// Add custom middleware
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test")
			next.ServeHTTP(w, r)
		})
	}

	chain.AddMiddleware(customMiddleware)

	// Verify middleware was added
	newCount := chain.GetMiddlewareCount()
	if newCount != initialCount+1 {
		t.Errorf("Expected middleware count to increase by 1, got %d -> %d", initialCount, newCount)
	}

	// Test the custom middleware is applied
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := chain.Apply(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	// Custom header should be present
	if recorder.Header().Get("X-Custom") != "test" {
		t.Error("Custom middleware was not applied correctly")
	}
}

// TestMiddlewareChain_Reset tests resetting middleware chain
func TestMiddlewareChain_Reset(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	// Add custom middleware
	customMiddleware := func(next http.Handler) http.Handler {
		return next
	}
	chain.AddMiddleware(customMiddleware)

	customCount := chain.GetMiddlewareCount()

	// Reset chain
	chain.Reset()

	// Should rebuild default stack
	resetCount := chain.GetMiddlewareCount()
	if resetCount == customCount {
		t.Error("Reset did not change middleware count")
	}
	if resetCount == 0 {
		t.Error("Reset should rebuild default stack, not clear all middlewares")
	}
}

// TestMiddlewareChain_Clone tests cloning middleware chain
func TestMiddlewareChain_Clone(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	original := NewMiddlewareChain(deps)

	// Add custom middleware to original
	customMiddleware := func(next http.Handler) http.Handler {
		return next
	}
	original.AddMiddleware(customMiddleware)

	clone := original.Clone()

	// Verify clone is not nil and not the same instance
	if clone == nil {
		t.Fatal("Clone returned nil")
	}
	if clone == original {
		t.Error("Clone returned same instance")
	}

	// Verify clone has same configuration
	if clone.config != original.config {
		t.Error("Clone config differs from original")
	}
	if clone.GetMiddlewareCount() != original.GetMiddlewareCount() {
		t.Error("Clone middleware count differs from original")
	}

	// Verify independence - modifying clone doesn't affect original
	originalCount := original.GetMiddlewareCount()
	clone.AddMiddleware(customMiddleware)
	if original.GetMiddlewareCount() != originalCount {
		t.Error("Modifying clone affected original")
	}
}

// TestMiddlewareChain_CORSMiddleware tests CORS middleware functionality
func TestMiddlewareChain_CORSMiddleware(t *testing.T) {
	testCases := []struct {
		name               string
		environment        string
		origin             string
		allowedOrigins     []string
		expectedCORS       string
		expectCORSWildcard bool
	}{
		{
			name:           "development_no_origin",
			environment:    "development",
			origin:         "",
			expectedCORS:   "*",
			expectCORSWildcard: true,
		},
		{
			name:           "development_with_origin",
			environment:    "development",
			origin:         "https://example.com",
			expectedCORS:   "*", // Development allows wildcard
			expectCORSWildcard: true,
		},
		{
			name:           "production_allowed_origin",
			environment:    "production",
			origin:         "https://example.com",
			allowedOrigins: []string{"https://example.com"},
			expectedCORS:   "https://example.com",
			expectCORSWildcard: false,
		},
		{
			name:           "production_forbidden_origin",
			environment:    "production",
			origin:         "https://malicious.com",
			allowedOrigins: []string{"https://example.com"},
			expectedCORS:   "", // No CORS header for forbidden origins
			expectCORSWildcard: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := createTestMiddlewareDependencies()
			deps.Config.Server.Environment = tc.environment
			deps.OriginValidator = &MockOriginValidatorForMiddleware{
				AllowedOrigins: tc.allowedOrigins,
				AllowAll:       false,
			}

			// Create custom validator logic
			if len(tc.allowedOrigins) > 0 {
				for _, allowed := range tc.allowedOrigins {
					if tc.origin == allowed {
						deps.OriginValidator.(*MockOriginValidatorForMiddleware).AllowAll = true
						break
					}
				}
			}

			chain := NewMiddlewareChain(deps)

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(testHandler)
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			recorder := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(recorder, req)

			corsHeader := recorder.Header().Get("Access-Control-Allow-Origin")
			if tc.expectCORSWildcard && corsHeader != "*" {
				t.Errorf("Expected CORS wildcard '*', got '%s'", corsHeader)
			} else if !tc.expectCORSWildcard && tc.expectedCORS != "" && corsHeader != tc.expectedCORS {
				t.Errorf("Expected CORS '%s', got '%s'", tc.expectedCORS, corsHeader)
			} else if tc.expectedCORS == "" && corsHeader != "" {
				t.Errorf("Expected no CORS header, got '%s'", corsHeader)
			}
		})
	}
}

// TestMiddlewareChain_OptionsRequest tests CORS preflight handling
func TestMiddlewareChain_OptionsRequest(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not be reached for OPTIONS requests
		w.WriteHeader(http.StatusInternalServerError)
		t.Error("Handler should not be called for OPTIONS requests")
	})

	wrappedHandler := chain.Apply(testHandler)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	// OPTIONS requests should return 200 OK without calling the handler
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", recorder.Code)
	}

	// Should have CORS headers
	if recorder.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Missing CORS Allow-Methods header")
	}
	if recorder.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Missing CORS Allow-Headers header")
	}
}

// TestNewCustomMiddlewareChain tests custom middleware chain creation
func TestNewCustomMiddlewareChain(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	
	config := MiddlewareConfig{
		EnableLogging:    true,
		EnableCORS:       true,
		EnableRateLimit:  false,
		EnableSecurity:   false,
		EnableAuth:       false,
		EnableMonitoring: false,
	}

	chain := NewCustomMiddlewareChain(deps, config)

	// Verify construction succeeded
	if chain == nil {
		t.Fatal("NewCustomMiddlewareChain returned nil")
	}

	// Should have only enabled middlewares (logging + CORS)
	expectedCount := 2
	if chain.GetMiddlewareCount() != expectedCount {
		t.Errorf("Expected %d middlewares, got %d", expectedCount, chain.GetMiddlewareCount())
	}
}

// TestMiddlewareChain_WithMonitoring tests middleware chain with monitoring
func TestMiddlewareChain_WithMonitoring(t *testing.T) {
	deps := createTestMiddlewareDependencies()
	
	// Create mock monitor (would need actual implementation for real test)
	// For now, test without monitor
	deps.Monitor = nil

	chain := NewMiddlewareChain(deps)

	// Should work without monitor
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := chain.Apply(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// BenchmarkMiddlewareChain_Apply benchmarks middleware application
func BenchmarkMiddlewareChain_Apply(b *testing.B) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrappedHandler := chain.Apply(testHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, req)
	}
}

// BenchmarkMiddlewareChain_AddMiddleware benchmarks middleware addition
func BenchmarkMiddlewareChain_AddMiddleware(b *testing.B) {
	deps := createTestMiddlewareDependencies()
	chain := NewMiddlewareChain(deps)

	middleware := func(next http.Handler) http.Handler {
		return next
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset chain to avoid growing indefinitely
		if i%1000 == 0 {
			chain.Reset()
		}
		chain.AddMiddleware(middleware)
	}
}