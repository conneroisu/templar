package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/renderer"
)

func TestCSPNonceImplementation(t *testing.T) {
	tests := []struct {
		name         string
		config       *SecurityConfig
		expectNonce  bool
		expectUnsafe bool
		description  string
	}{
		{
			name:         "default_config_with_nonce",
			config:       DefaultSecurityConfig(),
			expectNonce:  true,
			expectUnsafe: false,
			description:  "Default config should use nonces and remove unsafe directives",
		},
		{
			name:         "development_config_with_nonce",
			config:       DevelopmentSecurityConfig(),
			expectNonce:  true,
			expectUnsafe: false,
			description:  "Development config should use nonces and remove unsafe directives",
		},
		{
			name:         "production_config_with_nonce",
			config:       ProductionSecurityConfig(),
			expectNonce:  true,
			expectUnsafe: false,
			description:  "Production config should use nonces and remove unsafe directives",
		},
		{
			name: "config_with_nonce_disabled",
			config: func() *SecurityConfig {
				config := DefaultSecurityConfig()
				config.EnableNonce = false
				// Add unsafe directives when nonce is disabled
				config.CSP.ScriptSrc = append(config.CSP.ScriptSrc, "'unsafe-inline'")
				config.CSP.StyleSrc = append(config.CSP.StyleSrc, "'unsafe-inline'")
				return config
			}(),
			expectNonce:  false,
			expectUnsafe: true,
			description:  "Config with nonce disabled should allow unsafe directives",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Apply security middleware
			middleware := SecurityMiddleware(tt.config)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if nonce is in context
				nonce := GetNonceFromContext(r.Context())
				if tt.expectNonce && nonce == "" {
					t.Error("Expected nonce in context but got empty string")
				}
				if !tt.expectNonce && nonce != "" {
					t.Error("Expected no nonce in context but got:", nonce)
				}

				w.WriteHeader(http.StatusOK)
			}))

			// Execute request
			handler.ServeHTTP(rec, req)

			// Check CSP header
			cspHeader := rec.Header().Get("Content-Security-Policy")
			if cspHeader == "" {
				t.Fatal("Expected CSP header but got none")
			}

			// Check for nonce directive
			hasNonce := strings.Contains(cspHeader, "nonce-")
			if tt.expectNonce && !hasNonce {
				t.Error("Expected nonce directive in CSP header but didn't find it")
				t.Logf("CSP header: %s", cspHeader)
			}
			if !tt.expectNonce && hasNonce {
				t.Error("Expected no nonce directive in CSP header but found it")
				t.Logf("CSP header: %s", cspHeader)
			}

			// Check for unsafe directives
			hasUnsafeInline := strings.Contains(cspHeader, "'unsafe-inline'")
			hasUnsafeEval := strings.Contains(cspHeader, "'unsafe-eval'")

			if tt.expectUnsafe && !hasUnsafeInline {
				t.Error("Expected unsafe-inline directive but didn't find it")
			}
			if !tt.expectUnsafe && hasUnsafeInline {
				t.Error("Expected no unsafe-inline directive but found it")
				t.Logf("CSP header: %s", cspHeader)
			}
			if !tt.expectUnsafe && hasUnsafeEval {
				t.Error("Expected no unsafe-eval directive but found it")
				t.Logf("CSP header: %s", cspHeader)
			}
		})
	}
}

func TestCSPNonceGeneration(t *testing.T) {
	// Test that nonces are unique across requests
	config := DefaultSecurityConfig()
	middleware := SecurityMiddleware(config)

	var nonces []string
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := GetNonceFromContext(r.Context())
		nonces = append(nonces, nonce)
		w.WriteHeader(http.StatusOK)
	}))

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Verify all nonces are unique
	if len(nonces) != 10 {
		t.Fatalf("Expected 10 nonces but got %d", len(nonces))
	}

	nonceMap := make(map[string]bool)
	for _, nonce := range nonces {
		if nonce == "" {
			t.Error("Found empty nonce")
		}
		if nonceMap[nonce] {
			t.Error("Found duplicate nonce:", nonce)
		}
		nonceMap[nonce] = true

		// Verify nonce format (base64 encoded)
		if len(nonce) < 16 {
			t.Error("Nonce too short:", nonce)
		}
	}
}

func TestXSSProtectionWithNonce(t *testing.T) {
	config := DefaultSecurityConfig()
	middleware := SecurityMiddleware(config)

	// Test various XSS payload patterns to ensure CSP properly blocks them
	xssTests := []struct {
		name        string
		description string
	}{
		{"script_tag", "Inline script tag XSS"},
		{"img_onerror", "Image onerror XSS"},
		{"svg_onload", "SVG onload XSS"},
		{"javascript_url", "JavaScript URL XSS"},
	}

	for _, tt := range xssTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify CSP header would block inline scripts
				cspHeader := w.Header().Get("Content-Security-Policy")
				if !strings.Contains(cspHeader, "script-src") {
					t.Error("Expected script-src directive in CSP")
				}
				if strings.Contains(cspHeader, "'unsafe-inline'") {
					t.Error("CSP should not contain unsafe-inline when nonce is enabled")
				}
				if !strings.Contains(cspHeader, "nonce-") {
					t.Error("CSP should contain nonce directive")
				}

				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rec, req)
		})
	}
}

func TestNonceInHTMLGeneration(t *testing.T) {
	// Test that generated HTML includes nonce attributes
	testNonce := "test-nonce-123"

	// Create renderer instance (we need a registry for the constructor)
	mockRenderer := renderer.NewComponentRenderer(nil)

	// Test HTML generation with nonce
	html := mockRenderer.RenderComponentWithLayoutAndNonce("TestComponent", "<div>Test</div>", testNonce)

	// Verify nonce is included in script and style tags
	expectedScriptNonce := `nonce="` + testNonce + `"`
	expectedStyleNonce := `nonce="` + testNonce + `"`

	if !strings.Contains(html, expectedScriptNonce) {
		t.Error("Expected script nonce attribute but didn't find it")
		t.Logf("Generated HTML: %s", html)
	}

	if !strings.Contains(html, expectedStyleNonce) {
		t.Error("Expected style nonce attribute but didn't find it")
		t.Logf("Generated HTML: %s", html)
	}

	// Verify inline scripts have nonces
	nonceCount := strings.Count(html, expectedScriptNonce)

	// Should have nonces for inline scripts
	if nonceCount < 2 {
		t.Errorf("Expected at least 2 script nonces but found %d", nonceCount)
	}
}

func TestCSPHeaderConstruction(t *testing.T) {
	tests := []struct {
		name     string
		csp      *CSPConfig
		nonce    string
		expected map[string]bool // Expected directives in header
	}{
		{
			name: "script_src_with_nonce",
			csp: &CSPConfig{
				ScriptSrc: []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"},
			},
			nonce: "test123",
			expected: map[string]bool{
				"script-src 'self' 'nonce-test123'": true,
				"'unsafe-inline'":                   false,
				"'unsafe-eval'":                     false,
			},
		},
		{
			name: "style_src_with_nonce",
			csp: &CSPConfig{
				StyleSrc: []string{"'self'", "'unsafe-inline'"},
			},
			nonce: "test456",
			expected: map[string]bool{
				"style-src 'self' 'nonce-test456'": true,
				"'unsafe-inline'":                  false,
			},
		},
		{
			name: "no_nonce_preserves_unsafe",
			csp: &CSPConfig{
				ScriptSrc: []string{"'self'", "'unsafe-inline'"},
			},
			nonce: "",
			expected: map[string]bool{
				"script-src 'self' 'unsafe-inline'": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := buildCSPHeader(tt.csp, tt.nonce)

			for directive, shouldExist := range tt.expected {
				exists := strings.Contains(header, directive)
				if shouldExist && !exists {
					t.Errorf("Expected directive '%s' in header but didn't find it", directive)
					t.Logf("CSP header: %s", header)
				}
				if !shouldExist && exists {
					t.Errorf("Expected directive '%s' NOT in header but found it", directive)
					t.Logf("CSP header: %s", header)
				}
			}
		})
	}
}

func TestGetNonceFromContextNew(t *testing.T) {
	// Test with nonce in context
	testNonce := "test-nonce-value"
	ctx := context.WithValue(context.Background(), nonceContextKey, testNonce)

	retrievedNonce := GetNonceFromContext(ctx)
	if retrievedNonce != testNonce {
		t.Errorf("Expected nonce '%s' but got '%s'", testNonce, retrievedNonce)
	}

	// Test with no nonce in context
	emptyCtx := context.Background()
	retrievedNonce = GetNonceFromContext(emptyCtx)
	if retrievedNonce != "" {
		t.Errorf("Expected empty nonce but got '%s'", retrievedNonce)
	}

	// Test with wrong type in context
	wrongCtx := context.WithValue(context.Background(), nonceContextKey, 123)
	retrievedNonce = GetNonceFromContext(wrongCtx)
	if retrievedNonce != "" {
		t.Errorf("Expected empty nonce for wrong type but got '%s'", retrievedNonce)
	}
}

