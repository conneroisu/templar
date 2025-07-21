package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conneroisu/templar/internal/config"
)

func TestAuthMiddleware_Disabled(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled: false,
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK when auth disabled, got %d", w.Code)
	}
}

func TestAuthMiddleware_LocalhostBypass(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:         true,
		LocalhostBypass: true,
		RequireAuth:     true,
		Mode:            "basic",
		Username:        "admin",
		Password:        "secret",
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		expectCode int
	}{
		{
			name:       "localhost_ipv4",
			remoteAddr: "127.0.0.1:1234",
			expectCode: http.StatusOK,
		},
		{
			name:       "localhost_ipv6",
			remoteAddr: "[::1]:1234",
			expectCode: http.StatusOK,
		},
		{
			name:       "external_ip_no_auth",
			remoteAddr: "192.168.1.100:1234",
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_IPAllowlist(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:    true,
		AllowedIPs: []string{"192.168.1.100", "10.0.0.50"},
		Mode:       "none",
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		expectCode int
	}{
		{
			name:       "allowed_ip_1",
			remoteAddr: "192.168.1.100:1234",
			expectCode: http.StatusOK,
		},
		{
			name:       "allowed_ip_2",
			remoteAddr: "10.0.0.50:5678",
			expectCode: http.StatusOK,
		},
		{
			name:       "blocked_ip",
			remoteAddr: "192.168.1.200:1234",
			expectCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_BasicAuth(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:         true,
		Mode:            "basic",
		Username:        "admin",
		Password:        "secret123",
		RequireAuth:     true,
		LocalhostBypass: false,
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name       string
		username   string
		password   string
		expectCode int
	}{
		{
			name:       "valid_credentials",
			username:   "admin",
			password:   "secret123",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid_username",
			username:   "user",
			password:   "secret123",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "invalid_password",
			username:   "admin",
			password:   "wrong",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "empty_credentials",
			username:   "",
			password:   "",
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.168.1.100:1234"

			if tt.username != "" || tt.password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(tt.username + ":" + tt.password))
				req.Header.Set("Authorization", "Basic "+auth)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}

			if tt.expectCode == http.StatusUnauthorized {
				wwwAuth := w.Header().Get("WWW-Authenticate")
				if !containsString(wwwAuth, "Basic") {
					t.Errorf("Expected WWW-Authenticate header to contain 'Basic', got: %s", wwwAuth)
				}
			}
		})
	}
}

func TestAuthMiddleware_TokenAuth(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:         true,
		Mode:            "token",
		Token:           "super-secret-token-123",
		RequireAuth:     true,
		LocalhostBypass: false,
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name       string
		token      string
		method     string // "header" or "query"
		expectCode int
	}{
		{
			name:       "valid_token_header",
			token:      "super-secret-token-123",
			method:     "header",
			expectCode: http.StatusOK,
		},
		{
			name:       "valid_token_query",
			token:      "super-secret-token-123",
			method:     "query",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid_token_header",
			token:      "wrong-token",
			method:     "header",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "invalid_token_query",
			token:      "wrong-token",
			method:     "query",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "empty_token",
			token:      "",
			method:     "header",
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "query" {
				req = httptest.NewRequest(http.MethodGet, "/?token="+tt.token, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/", nil)
				if tt.token != "" {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}
			req.RemoteAddr = "192.168.1.100:1234"

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}

			if tt.expectCode == http.StatusUnauthorized {
				wwwAuth := w.Header().Get("WWW-Authenticate")
				if !containsString(wwwAuth, "Bearer") {
					t.Errorf("Expected WWW-Authenticate header to contain 'Bearer', got: %s", wwwAuth)
				}
			}
		})
	}
}

func TestAuthMiddleware_NoAuthRequired(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:         true,
		Mode:            "basic",
		Username:        "admin",
		Password:        "secret",
		RequireAuth:     false, // Key: no auth required
		LocalhostBypass: true,
	}

	middleware := AuthMiddleware(authConfig)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:1234" // External IP
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should pass without authentication because RequireAuth is false
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK when RequireAuth is false, got %d", w.Code)
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"localhost", true},
		{"[::1]", true},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"8.8.8.8", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isLocalhost(tt.ip)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	allowedIPs := []string{"192.168.1.100", "10.0.0.50", "::1"}

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.100", true},
		{"10.0.0.50", true},
		{"::1", true},
		{"[::1]", true},
		{"192.168.1.101", false},
		{"10.0.0.51", false},
		{"127.0.0.1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isIPAllowed(tt.ip, allowedIPs)
			if result != tt.expected {
				t.Errorf("isIPAllowed(%q, %v) = %v, want %v", tt.ip, allowedIPs, result, tt.expected)
			}
		})
	}
}

func TestAuthenticateToken(t *testing.T) {
	expectedToken := "test-token-123"

	tests := []struct {
		name          string
		authHeader    string
		queryToken    string
		expectedToken string
		expected      bool
	}{
		{
			name:          "valid_bearer_token",
			authHeader:    "Bearer test-token-123",
			expectedToken: expectedToken,
			expected:      true,
		},
		{
			name:          "valid_query_token",
			queryToken:    "test-token-123",
			expectedToken: expectedToken,
			expected:      true,
		},
		{
			name:          "invalid_bearer_token",
			authHeader:    "Bearer wrong-token",
			expectedToken: expectedToken,
			expected:      false,
		},
		{
			name:          "invalid_query_token",
			queryToken:    "wrong-token",
			expectedToken: expectedToken,
			expected:      false,
		},
		{
			name:          "malformed_header",
			authHeader:    "Basic dGVzdA==",
			expectedToken: expectedToken,
			expected:      false,
		},
		{
			name:          "empty_token",
			expectedToken: "",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?token="+tt.queryToken, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			result := authenticateToken(req, tt.expectedToken)
			if result != tt.expected {
				t.Errorf("authenticateToken() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuthenticateBasic(t *testing.T) {
	expectedUsername := "admin"
	expectedPassword := "secret"

	tests := []struct {
		name     string
		username string
		password string
		expected bool
	}{
		{
			name:     "valid_credentials",
			username: "admin",
			password: "secret",
			expected: true,
		},
		{
			name:     "invalid_username",
			username: "user",
			password: "secret",
			expected: false,
		},
		{
			name:     "invalid_password",
			username: "admin",
			password: "wrong",
			expected: false,
		},
		{
			name:     "empty_credentials",
			username: "",
			password: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.username != "" || tt.password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(tt.username + ":" + tt.password))
				req.Header.Set("Authorization", "Basic "+auth)
			}

			result := authenticateBasic(req, expectedUsername, expectedPassword)
			if result != tt.expected {
				t.Errorf("authenticateBasic() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function for string contains check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
