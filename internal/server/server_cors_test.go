package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORSProduction(t *testing.T) {
	// Test production CORS policy
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:           "localhost",
			Port:           8080,
			Environment:    "production",
			AllowedOrigins: []string{"https://app.example.com", "https://dashboard.example.com"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		description    string
	}{
		{
			name:           "allowed production origin",
			origin:         "https://app.example.com",
			expectedOrigin: "https://app.example.com",
			description:    "Should allow whitelisted production origins",
		},
		{
			name:           "allowed dashboard origin",
			origin:         "https://dashboard.example.com",
			expectedOrigin: "https://dashboard.example.com",
			description:    "Should allow multiple whitelisted origins",
		},
		{
			name:           "malicious external origin",
			origin:         "https://evil.com",
			expectedOrigin: "",
			description:    "Should reject non-whitelisted origins",
		},
		{
			name:           "no origin header",
			origin:         "",
			expectedOrigin: "",
			description:    "Should handle missing origin gracefully",
		},
		{
			name:           "localhost in production",
			origin:         "http://localhost:3000",
			expectedOrigin: "",
			description:    "Should reject localhost in production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			w := httptest.NewRecorder()
			handler := server.addMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			)

			handler.ServeHTTP(w, req)

			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			assert.Equal(t, tt.expectedOrigin, corsHeader, tt.description)
		})
	}
}

func TestCORSDevelopment(t *testing.T) {
	// Test development CORS policy
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:           "localhost",
			Port:           8080,
			Environment:    "development",
			AllowedOrigins: []string{"https://app.example.com"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		description    string
	}{
		{
			name:           "allowed origin in dev",
			origin:         "https://app.example.com",
			expectedOrigin: "https://app.example.com",
			description:    "Should allow whitelisted origins in development",
		},
		{
			name:           "external origin in dev",
			origin:         "https://external.com",
			expectedOrigin: "*",
			description:    "Should fall back to wildcard for unknown origins in development",
		},
		{
			name:           "localhost in dev",
			origin:         "http://localhost:3000",
			expectedOrigin: "*",
			description:    "Should allow localhost via wildcard in development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			w := httptest.NewRecorder()
			handler := server.addMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			)

			handler.ServeHTTP(w, req)

			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			assert.Equal(t, tt.expectedOrigin, corsHeader, tt.description)
		})
	}
}

func TestCORSPreflightRequests(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:        "localhost",
			Port:        8080,
			Environment: "production",
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	handler := server.addMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS preflight requests")
	}))

	handler.ServeHTTP(w, req)

	// Verify preflight response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "GET, POST, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestIsAllowedOrigin(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AllowedOrigins: []string{
				"https://app.example.com",
				"https://dashboard.example.com",
				"http://localhost:3000",
			},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://app.example.com", true},
		{"https://dashboard.example.com", true},
		{"http://localhost:3000", true},
		{"https://evil.com", false},
		{"http://localhost:8080", false},
		{"", false},
		{"https://app.example.com.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result := server.isAllowedOrigin(tt.origin)
			assert.Equal(t, tt.expected, result)
		})
	}
}
