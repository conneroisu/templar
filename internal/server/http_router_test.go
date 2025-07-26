package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// MockHTTPHandlers implements HTTPHandlers interface for testing.
type MockHTTPHandlers struct {
	HandleWebSocketCalls  int
	HandleHealthCalls     int
	HandleComponentsCalls int
}

func (m *MockHTTPHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	m.HandleWebSocketCalls++
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	m.HandleHealthCalls++
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleComponents(w http.ResponseWriter, r *http.Request) {
	m.HandleComponentsCalls++
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleComponent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleRender(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleStatic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandlePlaygroundIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandlePlaygroundComponent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandlePlaygroundRender(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleEnhancedIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleEditorIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleEditorAPI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleFileAPI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleInlineEditor(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleBuildStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleBuildMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleBuildErrors(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleBuildCache(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *MockHTTPHandlers) HandleTargetFiles(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// MockMiddlewareProvider implements MiddlewareProvider interface for testing.
type MockMiddlewareProvider struct {
	ApplyCalls int
}

func (m *MockMiddlewareProvider) Apply(handler http.Handler) http.Handler {
	m.ApplyCalls++

	return handler
}

// createTestConfig creates a valid configuration for testing.
func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:        "localhost",
			Port:        8080,
			Environment: "test",
		},
	}
}

// TestNewHTTPRouter_ValidInputs tests successful construction.
func TestNewHTTPRouter_ValidInputs(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	// Verify construction succeeded
	if router == nil {
		t.Fatal("NewHTTPRouter returned nil")
	}

	// Verify dependencies were stored
	if router.config != cfg {
		t.Error("Config was not stored correctly")
	}
	if router.handlers != handlers {
		t.Error("Handlers were not stored correctly")
	}
	if router.mux == nil {
		t.Error("Mux was not initialized")
	}
	if router.httpServer == nil {
		t.Error("HTTP server was not initialized")
	}

	// Verify middleware was applied
	if middleware.ApplyCalls != 1 {
		t.Errorf("Expected middleware Apply to be called once, got %d calls", middleware.ApplyCalls)
	}

	// Verify server configuration
	expectedAddr := "localhost:8080"
	if router.httpServer.Addr != expectedAddr {
		t.Errorf("Expected server address %s, got %s", expectedAddr, router.httpServer.Addr)
	}
}

// TestNewHTTPRouter_NilConfig tests panic on nil config.
func TestNewHTTPRouter_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil config, but didn't panic")
		}
	}()

	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}
	NewHTTPRouter(nil, handlers, middleware)
}

// TestNewHTTPRouter_NilHandlers tests panic on nil handlers.
func TestNewHTTPRouter_NilHandlers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil handlers, but didn't panic")
		}
	}()

	cfg := createTestConfig()
	middleware := &MockMiddlewareProvider{}
	NewHTTPRouter(cfg, nil, middleware)
}

// TestNewHTTPRouter_NilMiddleware tests panic on nil middleware provider.
func TestNewHTTPRouter_NilMiddleware(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil middleware provider, but didn't panic")
		}
	}()

	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	NewHTTPRouter(cfg, handlers, nil)
}

// TestNewHTTPRouter_InvalidPort tests panic on invalid port.
func TestNewHTTPRouter_InvalidPort(t *testing.T) {
	testCases := []int{0, -1, 65536, 100000}

	for _, port := range testCases {
		t.Run(fmt.Sprintf("port_%d", port), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for invalid port %d, but didn't panic", port)
				}
			}()

			cfg := createTestConfig()
			cfg.Server.Port = port
			handlers := &MockHTTPHandlers{}
			middleware := &MockMiddlewareProvider{}
			NewHTTPRouter(cfg, handlers, middleware)
		})
	}
}

// TestNewHTTPRouter_EmptyHost tests panic on empty host.
func TestNewHTTPRouter_EmptyHost(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty host, but didn't panic")
		}
	}()

	cfg := createTestConfig()
	cfg.Server.Host = ""
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}
	NewHTTPRouter(cfg, handlers, middleware)
}

// TestHTTPRouter_RouteRegistration tests that routes are properly registered.
func TestHTTPRouter_RouteRegistration(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	// Test critical routes by making requests
	routes := []struct {
		path   string
		method string
	}{
		{"/health", "GET"},
		{"/components", "GET"},
		{"/component/test", "GET"},
		{"/playground", "GET"},
		{"/api/build/status", "GET"},
	}

	for _, route := range routes {
		t.Run(fmt.Sprintf("%s_%s", route.method, route.path), func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			recorder := httptest.NewRecorder()

			router.mux.ServeHTTP(recorder, req)

			if recorder.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not registered", route.method, route.path)
			}
		})
	}

	// Verify handlers were called
	if handlers.HandleHealthCalls == 0 {
		t.Error("Health handler was not called")
	}
	if handlers.HandleComponentsCalls == 0 {
		t.Error("Components handler was not called")
	}
}

// TestHTTPRouter_Shutdown tests graceful shutdown.
func TestHTTPRouter_Shutdown(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	// Test shutdown with valid context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := router.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify shutdown state
	if !router.IsShutdown() {
		t.Error("Router should be marked as shut down")
	}

	// Test idempotent shutdown
	err = router.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown call failed: %v", err)
	}
}

// TestHTTPRouter_Shutdown_NilContext tests error on nil context.
func TestHTTPRouter_Shutdown_NilContext(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	err := router.Shutdown(context.TODO())
	if err == nil {
		t.Error("Expected error for nil context, but got none")
	}
}

// TestHTTPRouter_GetAddr tests address retrieval.
func TestHTTPRouter_GetAddr(t *testing.T) {
	cfg := createTestConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 9090
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	addr := router.GetAddr()
	expected := "127.0.0.1:9090"
	if addr != expected {
		t.Errorf("Expected address %s, got %s", expected, addr)
	}
}

// TestHTTPRouter_Start_NilContext tests error on nil context.
func TestHTTPRouter_Start_NilContext(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	err := router.Start(context.TODO())
	if err == nil {
		t.Error("Expected error for nil context, but got none")
	}
}

// TestHTTPRouter_Start_AlreadyShutdown tests error when starting shut down router.
func TestHTTPRouter_Start_AlreadyShutdown(t *testing.T) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	// Shutdown router first
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	router.Shutdown(ctx)

	// Try to start shut down router
	startCtx, startCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer startCancel()

	err := router.Start(startCtx)
	if err == nil {
		t.Error("Expected error when starting shut down router, but got none")
	}
}

// TestHTTPRouter_TargetFiles tests target files routing.
func TestHTTPRouter_TargetFiles(t *testing.T) {
	cfg := createTestConfig()
	cfg.TargetFiles = []string{"test.templ"} // Enable target files mode
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	// Test root route should use HandleTargetFiles
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.mux.ServeHTTP(recorder, req)

	// Should not return 404 (route exists)
	if recorder.Code == http.StatusNotFound {
		t.Error("Root route not registered for target files mode")
	}
}

// BenchmarkHTTPRouter_Apply benchmarks middleware application.
func BenchmarkHTTPRouter_Apply(b *testing.B) {
	cfg := createTestConfig()
	handlers := &MockHTTPHandlers{}
	middleware := &MockMiddlewareProvider{}

	router := NewHTTPRouter(cfg, handlers, middleware)

	b.ResetTimer()
	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()
		router.mux.ServeHTTP(recorder, req)
	}
}
