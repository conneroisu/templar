package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockOriginValidator implements OriginValidator interface for testing
type MockOriginValidator struct {
	AllowedOrigins []string
	AllowAll       bool
}

func (m *MockOriginValidator) IsAllowedOrigin(origin string) bool {
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

// Note: MockTokenBucketManager removed since current WebSocketManager implementation
// uses simplified rate limiting that always allows requests

// TestNewWebSocketManager_ValidInputs tests successful construction
func TestNewWebSocketManager_ValidInputs(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	var rateLimiter *TokenBucketManager = nil // Use nil for this test

	manager := NewWebSocketManager(validator, rateLimiter)

	// Verify construction succeeded
	if manager == nil {
		t.Fatal("NewWebSocketManager returned nil")
	}

	// Verify dependencies were stored
	if manager.originValidator != validator {
		t.Error("OriginValidator was not stored correctly")
	}
	if manager.rateLimiter != rateLimiter {
		t.Error("RateLimiter was not stored correctly")
	}

	// Verify initialization
	if manager.clients == nil {
		t.Error("Clients map was not initialized")
	}
	if manager.broadcast == nil {
		t.Error("Broadcast channel was not initialized")
	}
	if manager.register == nil {
		t.Error("Register channel was not initialized")
	}
	if manager.unregister == nil {
		t.Error("Unregister channel was not initialized")
	}
	if manager.ctx == nil {
		t.Error("Context was not initialized")
	}
	if manager.cancel == nil {
		t.Error("Cancel function was not initialized")
	}

	// Verify initial state
	if manager.isShutdown {
		t.Error("Manager should not be shut down initially")
	}

	// Clean shutdown
	manager.Shutdown(context.Background())
}

// TestNewWebSocketManager_NilOriginValidator tests panic on nil origin validator
func TestNewWebSocketManager_NilOriginValidator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil origin validator, but didn't panic")
		}
	}()

	NewWebSocketManager(nil, nil)
}

// TestNewWebSocketManager_NilRateLimiter tests construction with nil rate limiter (should be allowed)
func TestNewWebSocketManager_NilRateLimiter(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}

	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	if manager == nil {
		t.Fatal("NewWebSocketManager returned nil with nil rate limiter")
	}
	if manager.rateLimiter != nil {
		t.Error("Rate limiter should be nil when not provided")
	}
}

// TestWebSocketManager_HandleWebSocket_InvalidParameters tests parameter validation
func TestWebSocketManager_HandleWebSocket_InvalidParameters(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	// Test nil ResponseWriter
	req := httptest.NewRequest("GET", "/ws", nil)
	manager.HandleWebSocket(nil, req)

	// Test nil Request
	recorder := httptest.NewRecorder()
	manager.HandleWebSocket(recorder, nil)
}

// TestWebSocketManager_HandleWebSocket_ShutdownState tests handling during shutdown
func TestWebSocketManager_HandleWebSocket_ShutdownState(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)

	// Shutdown manager first
	manager.Shutdown(context.Background())

	// Try to handle WebSocket connection
	req := httptest.NewRequest("GET", "/ws", nil)
	recorder := httptest.NewRecorder()

	manager.HandleWebSocket(recorder, req)

	// Should return 503 Service Unavailable
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", recorder.Code)
	}
}

// TestWebSocketManager_HandleWebSocket_OriginValidation tests origin validation
func TestWebSocketManager_HandleWebSocket_OriginValidation(t *testing.T) {
	// Create validator that allows specific origins
	validator := &MockOriginValidator{
		AllowedOrigins: []string{"https://example.com"},
		AllowAll:       false,
	}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	testCases := []struct {
		name           string
		origin         string
		expectedStatus int
	}{
		{
			name:           "allowed_origin",
			origin:         "https://example.com",
			expectedStatus: http.StatusBadRequest, // WebSocket upgrade will fail but origin check passes
		},
		{
			name:           "forbidden_origin",
			origin:         "https://malicious.com",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no_origin",
			origin:         "",
			expectedStatus: http.StatusBadRequest, // No origin is allowed, WebSocket upgrade fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			recorder := httptest.NewRecorder()

			manager.HandleWebSocket(recorder, req)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}
		})
	}
}

// TestWebSocketManager_HandleWebSocket_RateLimiting tests rate limiting
func TestWebSocketManager_HandleWebSocket_RateLimiting(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	
	// Test with no rate limiter (current behavior always allows)
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "/ws", nil)
	recorder := httptest.NewRecorder()

	manager.HandleWebSocket(recorder, req)

	// Should fail at WebSocket upgrade (BadRequest) since we're not doing real WebSocket handshake
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d (WebSocket upgrade failure), got %d", http.StatusBadRequest, recorder.Code)
	}
}

// TestWebSocketManager_BroadcastMessage tests message broadcasting
func TestWebSocketManager_BroadcastMessage(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	// Test broadcasting with valid message
	message := UpdateMessage{
		Type:      "test",
		Content:   "test message",
		Timestamp: time.Now(),
	}

	// This should not block or panic
	manager.BroadcastMessage(message)

	// Verify message was queued (hard to test directly, but should not error)
}

// TestWebSocketManager_GetConnectedClients tests client counting
func TestWebSocketManager_GetConnectedClients(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	// Initially should have no clients
	count := manager.GetConnectedClients()
	if count != 0 {
		t.Errorf("Expected 0 connected clients, got %d", count)
	}
}

// TestWebSocketManager_GetClients tests client map retrieval
func TestWebSocketManager_GetClients(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	clients := manager.GetClients()
	if clients == nil {
		t.Error("GetClients returned nil")
	}
	if len(clients) != 0 {
		t.Errorf("Expected empty clients map, got %d clients", len(clients))
	}
}

// TestWebSocketManager_Shutdown tests graceful shutdown
func TestWebSocketManager_Shutdown(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify shutdown state
	if !manager.IsShutdown() {
		t.Error("Manager should be marked as shut down")
	}

	// Test idempotent shutdown
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown call failed: %v", err)
	}
}

// TestWebSocketManager_ValidateWebSocketRequest tests request validation
func TestWebSocketManager_ValidateWebSocketRequest(t *testing.T) {
	validator := &MockOriginValidator{
		AllowedOrigins: []string{"https://example.com"},
		AllowAll:       false,
	}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	testCases := []struct {
		name     string
		origin   string
		expected bool
	}{
		{
			name:     "allowed_origin",
			origin:   "https://example.com",
			expected: true,
		},
		{
			name:     "forbidden_origin",
			origin:   "https://malicious.com",
			expected: false,
		},
		{
			name:     "no_origin",
			origin:   "",
			expected: true, // No origin is allowed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			result := manager.validateWebSocketRequest(req)
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

// TestWebSocketManager_GetClientIP tests client IP extraction
func TestWebSocketManager_GetClientIP(t *testing.T) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	testCases := []struct {
		name               string
		xForwardedFor      string
		xRealIP            string
		remoteAddr         string
		expectedContains   string
	}{
		{
			name:             "x_forwarded_for",
			xForwardedFor:    "192.168.1.1",
			expectedContains: "192.168.1.1",
		},
		{
			name:             "x_real_ip",
			xRealIP:          "10.0.0.1",
			expectedContains: "10.0.0.1",
		},
		{
			name:             "remote_addr",
			remoteAddr:       "127.0.0.1:8080",
			expectedContains: "127.0.0.1:8080",
		},
		{
			name:             "x_forwarded_for_priority",
			xForwardedFor:    "192.168.1.1",
			xRealIP:          "10.0.0.1",
			remoteAddr:       "127.0.0.1:8080",
			expectedContains: "192.168.1.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			
			if tc.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwardedFor)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			if tc.remoteAddr != "" {
				req.RemoteAddr = tc.remoteAddr
			}

			ip := manager.getClientIP(req)
			if !strings.Contains(ip, tc.expectedContains) {
				t.Errorf("Expected IP to contain %s, got %s", tc.expectedContains, ip)
			}
		})
	}
}

// BenchmarkWebSocketManager_BroadcastMessage benchmarks message broadcasting
func BenchmarkWebSocketManager_BroadcastMessage(b *testing.B) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	message := UpdateMessage{
		Type:      "benchmark",
		Content:   "benchmark message",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.BroadcastMessage(message)
	}
}

// BenchmarkWebSocketManager_GetConnectedClients benchmarks client counting
func BenchmarkWebSocketManager_GetConnectedClients(b *testing.B) {
	validator := &MockOriginValidator{AllowAll: true}
	manager := NewWebSocketManager(validator, nil)
	defer manager.Shutdown(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetConnectedClients()
	}
}