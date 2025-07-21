package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNonceGeneration(t *testing.T) {
	// Test nonce generation
	nonce1, err := generateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	nonce2, err := generateNonce()
	if err != nil {
		t.Fatalf("Failed to generate second nonce: %v", err)
	}

	// Nonces should be different
	if nonce1 == nonce2 {
		t.Error("Generated nonces should be unique")
	}

	// Nonces should be base64 encoded (at least 16 characters for 12 byte input)
	if len(nonce1) < 16 {
		t.Errorf("Nonce too short: %d characters", len(nonce1))
	}

	// Should be valid base64
	if strings.Contains(nonce1, " ") || strings.Contains(nonce1, "\n") {
		t.Error("Nonce should not contain whitespace")
	}
}

func TestGetNonceFromContext(t *testing.T) {
	tests := []struct {
		name     string
		nonce    string
		expected string
	}{
		{
			name:     "valid_nonce",
			nonce:    "test-nonce-123",
			expected: "test-nonce-123",
		},
		{
			name:     "empty_nonce",
			nonce:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), nonceContextKey, tt.nonce)
			result := GetNonceFromContext(ctx)
			if result != tt.expected {
				t.Errorf("GetNonceFromContext() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test context without nonce
	t.Run("no_nonce_in_context", func(t *testing.T) {
		ctx := context.Background()
		result := GetNonceFromContext(ctx)
		if result != "" {
			t.Errorf("GetNonceFromContext() = %q, want empty string", result)
		}
	})
}

func TestCSPWithNonce(t *testing.T) {
	tests := []struct {
		name        string
		config      *CSPConfig
		nonce       string
		expectedCSP string
	}{
		{
			name: "script_src_with_nonce",
			config: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				ScriptSrc:  []string{"'self'"},
			},
			nonce:       "abc123",
			expectedCSP: "default-src 'self'; script-src 'self' 'nonce-abc123'",
		},
		{
			name: "style_src_with_nonce",
			config: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				StyleSrc:   []string{"'self'"},
			},
			nonce:       "xyz789",
			expectedCSP: "default-src 'self'; style-src 'self' 'nonce-xyz789'",
		},
		{
			name: "both_script_and_style_with_nonce",
			config: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				ScriptSrc:  []string{"'self'"},
				StyleSrc:   []string{"'self'"},
			},
			nonce:       "test456",
			expectedCSP: "default-src 'self'; script-src 'self' 'nonce-test456'; style-src 'self' 'nonce-test456'",
		},
		{
			name: "no_nonce_when_empty",
			config: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				ScriptSrc:  []string{"'self'"},
			},
			nonce:       "",
			expectedCSP: "default-src 'self'; script-src 'self'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCSPHeader(tt.config, tt.nonce)
			if result != tt.expectedCSP {
				t.Errorf("buildCSPHeader() = %q, want %q", result, tt.expectedCSP)
			}
		})
	}
}

func TestProductionCSPSecurity(t *testing.T) {
	config := ProductionSecurityConfig()

	// Production config should not have unsafe directives
	for _, src := range config.CSP.ScriptSrc {
		if src == "'unsafe-inline'" || src == "'unsafe-eval'" {
			t.Errorf("Production CSP should not contain unsafe directive: %s", src)
		}
	}

	for _, src := range config.CSP.StyleSrc {
		if src == "'unsafe-inline'" {
			t.Errorf("Production CSP should not contain unsafe directive: %s", src)
		}
	}

	// Should enable nonce for production
	if !config.EnableNonce {
		t.Error("Production config should enable nonce")
	}

	// Should have CSP violation reporting
	if config.CSP.ReportURI == "" {
		t.Error("Production config should have CSP violation reporting URI")
	}
}

func TestSecurityMiddlewareWithNonce(t *testing.T) {
	config := ProductionSecurityConfig()
	middleware := SecurityMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that nonce is available in context
		nonce := GetNonceFromContext(r.Context())
		if nonce == "" {
			t.Error("Nonce should be available in request context")
		}

		// Write nonce to response for testing
		w.Header().Set("X-Test-Nonce", nonce)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check that CSP header is set
	cspHeader := w.Header().Get("Content-Security-Policy")
	if cspHeader == "" {
		t.Error("CSP header should be set")
	}

	// Check that nonce is included in CSP
	if !strings.Contains(cspHeader, "'nonce-") {
		t.Error("CSP header should contain nonce")
	}

	// Check that nonce is in context
	testNonce := w.Header().Get("X-Test-Nonce")
	if testNonce == "" {
		t.Error("Nonce should be passed through context")
	}

	// Verify nonce in CSP matches context nonce
	expectedNonce := "'nonce-" + testNonce + "'"
	if !strings.Contains(cspHeader, expectedNonce) {
		t.Errorf("CSP should contain nonce %s, but got: %s", expectedNonce, cspHeader)
	}
}

func TestCSPViolationReporting(t *testing.T) {
	// Test that CSP violation reporting is properly configured
	config := ProductionSecurityConfig()

	if config.CSP.ReportURI != "/api/csp-violation-report" {
		t.Errorf("Expected CSP report URI to be '/api/csp-violation-report', got %s", config.CSP.ReportURI)
	}

	// Build CSP header and check it includes report-uri
	cspHeader := buildCSPHeader(config.CSP, "test-nonce")
	if !strings.Contains(cspHeader, "report-uri /api/csp-violation-report") {
		t.Error("CSP header should contain report-uri directive")
	}
}

func TestDevelopmentVsProductionCSP(t *testing.T) {
	devConfig := DevelopmentSecurityConfig()
	prodConfig := ProductionSecurityConfig()

	// Both development and production should use nonces when provided
	devCSP := buildCSPHeader(devConfig.CSP, "dev-nonce-123")
	if !strings.Contains(devCSP, "'nonce-dev-nonce-123'") {
		t.Error("Development CSP should contain nonce when provided")
	}

	// Development with nonce should not contain unsafe directives
	if strings.Contains(devCSP, "'unsafe-inline'") || strings.Contains(devCSP, "'unsafe-eval'") {
		t.Error("Development CSP should not contain unsafe directives when nonce is used")
	}

	// Production should not allow unsafe directives
	prodCSP := buildCSPHeader(prodConfig.CSP, "test-nonce")
	if strings.Contains(prodCSP, "'unsafe-inline'") || strings.Contains(prodCSP, "'unsafe-eval'") {
		t.Error("Production CSP should not contain unsafe directives")
	}

	// Production should use nonces
	if !strings.Contains(prodCSP, "'nonce-test-nonce'") {
		t.Error("Production CSP should contain nonce")
	}

	// Both configs should enable nonce by default
	if !devConfig.EnableNonce {
		t.Error("Development config should enable nonce")
	}
	if !prodConfig.EnableNonce {
		t.Error("Production config should enable nonce")
	}
}
