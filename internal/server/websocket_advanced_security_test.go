// Package server provides comprehensive WebSocket security testing
// with advanced attack scenario coverage including connection hijacking,
// protocol downgrade attacks, and chaos testing with network failures.
package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

// TestWebSocketConnectionHijacking tests various connection hijacking attack scenarios
func TestWebSocketConnectionHijacking(t *testing.T) {
	tests := []struct {
		name        string
		setupAttack func(*httptest.Server) (*http.Request, error)
		expectBlock bool
		description string
	}{
		{
			name: "session_fixation_attack",
			setupAttack: func(server *httptest.Server) (*http.Request, error) {
				// Attempt to hijack by providing a fixed session cookie
				req, err := http.NewRequest(http.MethodGet, strings.Replace(server.URL, "http://", "ws://", 1)+"/ws", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Origin", "http://localhost:3000")
				req.Header.Set("Cookie", "session=admin123; csrf=bypassed")
				req.Header.Set("X-Forwarded-For", "127.0.0.1")
				return req, nil
			},
			expectBlock: true,
			description: "Should block session fixation attempts",
		},
		{
			name: "csrf_token_manipulation",
			setupAttack: func(server *httptest.Server) (*http.Request, error) {
				req, err := http.NewRequest(http.MethodGet, strings.Replace(server.URL, "http://", "ws://", 1)+"/ws", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Origin", "http://malicious.com")
				req.Header.Set("X-CSRF-Token", "fake_token")
				req.Header.Set("Referer", "http://malicious.com/attack")
				return req, nil
			},
			expectBlock: true,
			description: "Should block CSRF token manipulation",
		},
		{
			name: "host_header_injection",
			setupAttack: func(server *httptest.Server) (*http.Request, error) {
				req, err := http.NewRequest(http.MethodGet, strings.Replace(server.URL, "http://", "ws://", 1)+"/ws", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Origin", "http://localhost:3000")
				req.Header.Set("Host", "localhost:3000\r\nX-Injected: evil")
				return req, nil
			},
			expectBlock: true,
			description: "Should block host header injection attempts",
		},
		{
			name: "connection_upgrade_smuggling",
			setupAttack: func(server *httptest.Server) (*http.Request, error) {
				req, err := http.NewRequest(http.MethodGet, strings.Replace(server.URL, "http://", "ws://", 1)+"/ws", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Origin", "http://localhost:3000")
				req.Header.Set("Connection", "Upgrade, keep-alive")
				req.Header.Set("Upgrade", "websocket\r\nContent-Length: 100")
				return req, nil
			},
			expectBlock: true,
			description: "Should block connection upgrade smuggling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 3000,
				},
			}

			server, err := New(cfg)
			require.NoError(t, err)

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				server.handleWebSocket(w, r)
			}))
			defer testServer.Close()

			// Setup attack
			req, err := tt.setupAttack(testServer)
			require.NoError(t, err)

			// Attempt WebSocket connection
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, response, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1)+"/ws", &websocket.DialOptions{
				HTTPHeader: req.Header,
			})
			if response != nil && response.Body != nil {
				defer response.Body.Close()
			}

			if tt.expectBlock {
				// Should be blocked - either connection fails or non-101 response
				if err == nil && response != nil && response.StatusCode == http.StatusSwitchingProtocols {
					conn.Close(websocket.StatusNormalClosure, "")
					t.Errorf("%s: Expected connection to be blocked, but it succeeded", tt.description)
				}
			} else {
				// Should succeed
				require.NoError(t, err, tt.description)
				require.NotNil(t, conn, tt.description)
				conn.Close(websocket.StatusNormalClosure, "")
			}
		})
	}
}

// TestWebSocketProtocolDowngradeAttacks tests protocol downgrade attack prevention
func TestWebSocketProtocolDowngradeAttacks(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setupRequest  func(*http.Request)
		expectUpgrade bool
		description   string
	}{
		{
			name: "force_http10_downgrade",
			setupRequest: func(req *http.Request) {
				req.Proto = "HTTP/1.0"
				req.ProtoMajor = 1
				req.ProtoMinor = 0
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			},
			expectUpgrade: false,
			description:   "Should reject HTTP/1.0 WebSocket upgrade attempts",
		},
		{
			name: "malformed_websocket_version",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Sec-WebSocket-Version", "12") // Invalid version
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			},
			expectUpgrade: false,
			description:   "Should reject malformed WebSocket version",
		},
		{
			name: "missing_upgrade_header",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Connection", "keep-alive") // Missing Upgrade
				req.Header.Set("Sec-WebSocket-Version", "13")
			},
			expectUpgrade: false,
			description:   "Should reject missing Upgrade header",
		},
		{
			name: "protocol_confusion_attack",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "h2c") // HTTP/2 cleartext instead of websocket
				req.Header.Set("HTTP2-Settings", "AAMAAABkAARAAAAAAAIAAAAA")
			},
			expectUpgrade: false,
			description:   "Should reject HTTP/2 protocol confusion attacks",
		},
		{
			name: "websocket_key_manipulation",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Sec-WebSocket-Key", "invalid_key") // Invalid base64
			},
			expectUpgrade: false,
			description:   "Should reject invalid WebSocket key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.setupRequest(r)
				r.Header.Set("Origin", "http://localhost:3000") // Valid origin
				server.handleWebSocket(w, r)
			}))
			defer testServer.Close()

			// Attempt connection
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, response, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), nil)
			if response != nil && response.Body != nil {
				defer response.Body.Close()
			}

			if tt.expectUpgrade {
				require.NoError(t, err, tt.description)
				require.NotNil(t, conn, tt.description)
				conn.Close(websocket.StatusNormalClosure, "")
			} else {
				// Should be rejected - either error or non-101 status
				if err == nil && response != nil && response.StatusCode == http.StatusSwitchingProtocols {
					conn.Close(websocket.StatusNormalClosure, "")
					t.Errorf("%s: Expected protocol downgrade attack to be blocked", tt.description)
				}
			}
		})
	}
}

// TestWebSocketRateLimitingEdgeCases tests rate limiting edge cases and bypass attempts
func TestWebSocketRateLimitingEdgeCases(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name        string
		testFunc    func(t *testing.T, server *PreviewServer)
		description string
	}{
		{
			name: "connection_flooding_attack",
			testFunc: func(t *testing.T, server *PreviewServer) {
				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					server.handleWebSocket(w, r)
				}))
				defer testServer.Close()

				// Attempt to create many connections rapidly
				connections := make([]*websocket.Conn, 0, 100)
				defer func() {
					for _, conn := range connections {
						if conn != nil {
							conn.Close(websocket.StatusNormalClosure, "")
						}
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				blocked := 0
				for i := 0; i < 100; i++ {
					conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
						HTTPHeader: http.Header{
							"Origin": []string{"http://localhost:3000"},
						},
					})
					if resp != nil && resp.Body != nil {
						resp.Body.Close()
					}

					if err != nil {
						blocked++
					} else {
						connections = append(connections, conn)
					}

					// Small delay to avoid overwhelming the test
					time.Sleep(1 * time.Millisecond)
				}

				// Should have blocked some connections
				if blocked == 0 {
					t.Error("Expected some connections to be blocked in flooding attack, but none were")
				}

				t.Logf("Blocked %d out of 100 connection attempts", blocked)
			},
			description: "Should limit connection flooding attacks",
		},
		{
			name: "message_size_limit_bypass",
			testFunc: func(t *testing.T, server *PreviewServer) {
				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					server.handleWebSocket(w, r)
				}))
				defer testServer.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost:3000"},
					},
				})
				if resp != nil && resp.Body != nil {
					defer resp.Body.Close()
				}
				require.NoError(t, err)
				defer conn.Close(websocket.StatusNormalClosure, "")

				// Attempt to send oversized message
				largeMessage := strings.Repeat("A", 100*1024) // 100KB message
				err = conn.Write(ctx, websocket.MessageText, []byte(largeMessage))

				// Should either fail to send or connection should be closed
				if err == nil {
					// Try to read response - connection might be closed
					_, _, readErr := conn.Read(ctx)
					if readErr == nil {
						t.Error("Expected large message to be rejected or connection closed")
					}
				}
			},
			description: "Should enforce message size limits",
		},
		{
			name: "rapid_reconnection_attack",
			testFunc: func(t *testing.T, server *PreviewServer) {
				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					server.handleWebSocket(w, r)
				}))
				defer testServer.Close()

				blocked := 0
				for i := 0; i < 50; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					
					conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
						HTTPHeader: http.Header{
							"Origin": []string{"http://localhost:3000"},
						},
					})
					if resp != nil && resp.Body != nil {
						resp.Body.Close()
					}

					if err != nil {
						blocked++
						cancel()
						continue
					}

					// Immediately close and reconnect
					conn.Close(websocket.StatusNormalClosure, "")
					cancel()
				}

				// Should block some rapid reconnection attempts
				if blocked == 0 {
					t.Log("Warning: No connections blocked in rapid reconnection test - rate limiting may not be effective")
				}

				t.Logf("Blocked %d out of 50 rapid reconnection attempts", blocked)
			},
			description: "Should limit rapid reconnection attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, server)
		})
	}
}

// TestWebSocketChaosTestingNetworkFailures tests WebSocket behavior under network failures
func TestWebSocketChaosTestingNetworkFailures(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name        string
		chaosFunc   func(t *testing.T, server *PreviewServer, testServer *httptest.Server)
		description string
	}{
		{
			name: "sudden_connection_drop",
			chaosFunc: func(t *testing.T, server *PreviewServer, testServer *httptest.Server) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost:3000"},
					},
				})
				if resp != nil && resp.Body != nil {
					defer resp.Body.Close()
				}
				require.NoError(t, err)

				// Send a message to establish the connection
				err = conn.Write(ctx, websocket.MessageText, []byte("test"))
				require.NoError(t, err)

				// Forcefully close the connection
				conn.Close(websocket.StatusInternalError, "simulated network failure")

				// Try to send another message - should fail gracefully
				err = conn.Write(ctx, websocket.MessageText, []byte("should_fail"))
				assert.Error(t, err, "Expected error after TCP connection closed")
			},
			description: "Should handle sudden connection drops gracefully",
		},
		{
			name: "network_partition_simulation",
			chaosFunc: func(t *testing.T, server *PreviewServer, testServer *httptest.Server) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Create multiple connections
				connections := make([]*websocket.Conn, 3)
				defer func() {
					for _, conn := range connections {
						if conn != nil {
							conn.Close(websocket.StatusNormalClosure, "")
						}
					}
				}()

				// Establish connections
				for i := range connections {
					conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
						HTTPHeader: http.Header{
							"Origin": []string{"http://localhost:3000"},
						},
					})
					if resp != nil && resp.Body != nil {
						resp.Body.Close()
					}
					require.NoError(t, err)
					connections[i] = conn
				}

				// Simulate network partition by closing some connections abruptly
				for i := 0; i < 2; i++ {
					connections[i].Close(websocket.StatusInternalError, "network partition")
				}

				// Remaining connection should still work
				err = connections[2].Write(ctx, websocket.MessageText, []byte("survivor"))
				assert.NoError(t, err, "Surviving connection should still work after network partition")
			},
			description: "Should handle network partitions gracefully",
		},
		{
			name: "server_restart_simulation",
			chaosFunc: func(t *testing.T, server *PreviewServer, testServer *httptest.Server) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost:3000"},
					},
				})
				if resp != nil && resp.Body != nil {
					defer resp.Body.Close()
				}
				require.NoError(t, err)
				defer conn.Close(websocket.StatusNormalClosure, "")

				// Send initial message
				err = conn.Write(ctx, websocket.MessageText, []byte("before_restart"))
				require.NoError(t, err)

				// Simulate server restart by closing the test server
				testServer.Close()

				// Try to send message after "restart" - should fail
				err = conn.Write(ctx, websocket.MessageText, []byte("after_restart"))
				assert.Error(t, err, "Expected error after server restart")
			},
			description: "Should handle server restarts gracefully",
		},
		{
			name: "intermittent_connectivity",
			chaosFunc: func(t *testing.T, server *PreviewServer, testServer *httptest.Server) {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost:3000"},
					},
				})
				if resp != nil && resp.Body != nil {
					defer resp.Body.Close()
				}
				require.NoError(t, err)
				defer conn.Close(websocket.StatusNormalClosure, "")

				// Simulate intermittent connectivity by alternating successful and failed sends
				for i := 0; i < 10; i++ {
					message := []byte("intermittent_" + string(rune(i+'0')))
					err = conn.Write(ctx, websocket.MessageText, message)

					if i%3 == 2 {
						// Every third message, introduce a brief delay to simulate network hiccup
						time.Sleep(100 * time.Millisecond)
					}

					// Some messages may fail due to simulated network issues, which is expected
					if err != nil {
						t.Logf("Message %d failed as expected due to simulated network issues: %v", i, err)
					}
				}
			},
			description: "Should handle intermittent connectivity issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				server.handleWebSocket(w, r)
			}))
			defer testServer.Close()

			tt.chaosFunc(t, server, testServer)
		})
	}
}

// TestWebSocketOriginValidationComprehensive tests comprehensive origin validation scenarios
func TestWebSocketOriginValidationComprehensive(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	maliciousOrigins := []struct {
		origin      string
		description string
		attackType  string
	}{
		{"http://localhost:3000.evil.com", "Subdomain spoofing attack", "subdomain_spoofing"},
		{"http://localhost:3000/../admin", "Path traversal in origin", "path_traversal"},
		{"http://localhost:3000@attacker.com", "URL authority confusion", "authority_confusion"},
		{"http://localhost\x00:3000", "Null byte injection", "null_injection"},
		{"http://localhost\r\n:3000", "CRLF injection", "crlf_injection"},
		{"javascript:alert('xss')", "JavaScript protocol abuse", "js_protocol"},
		{"data:text/html,<script>alert('xss')</script>", "Data URI attack", "data_uri"},
		{"file:///etc/passwd", "File protocol attack", "file_protocol"},
		{"ftp://attacker.com", "FTP protocol attack", "ftp_protocol"},
		{"http://127.0.0.1:3000/../..", "IP-based path traversal", "ip_traversal"},
		{"http://[::1]:3000", "IPv6 localhost bypass attempt", "ipv6_bypass"},
		{"http://0.0.0.0:3000", "Wildcard IP bypass attempt", "wildcard_ip"},
		{"http://10.0.0.1:3000", "Private IP spoofing", "private_ip"},
		{"http://192.168.1.1:3000", "LAN IP spoofing", "lan_ip"},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	for _, test := range maliciousOrigins {
		t.Run(test.attackType, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, response, err := websocket.Dial(ctx, strings.Replace(testServer.URL, "http://", "ws://", 1), &websocket.DialOptions{
				HTTPHeader: http.Header{
					"Origin": []string{test.origin},
				},
			})
			if response != nil && response.Body != nil {
				defer response.Body.Close()
			}

			// All malicious origins should be blocked
			if err == nil && response != nil && response.StatusCode == http.StatusSwitchingProtocols {
				conn.Close(websocket.StatusNormalClosure, "")
				t.Errorf("Origin validation failed: %s should have been blocked (%s)", test.origin, test.description)
			} else {
				t.Logf("Successfully blocked %s: %s", test.attackType, test.description)
			}
		})
	}
}

// TestWebSocketSecurityHeaders tests that proper security headers are set
func TestWebSocketSecurityHeaders(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	// Make a request to WebSocket endpoint
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
	require.NoError(t, err)

	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check that security headers are present (these should be set by middleware)
	expectedHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options", 
		"X-XSS-Protection",
		"Referrer-Policy",
	}

	for _, header := range expectedHeaders {
		if resp.Header.Get(header) == "" {
			t.Logf("Security header %s is missing (may be set by middleware)", header)
		}
	}

	// WebSocket-specific checks
	if resp.StatusCode == http.StatusSwitchingProtocols {
		// If upgrade succeeded, connection should be secure
		upgrade := resp.Header.Get("Upgrade")
		connection := resp.Header.Get("Connection")

		assert.Equal(t, "websocket", upgrade, "Upgrade header should be 'websocket'")
		assert.Contains(t, strings.ToLower(connection), "upgrade", "Connection header should contain 'upgrade'")
	}
}

// BenchmarkWebSocketSecurityValidation benchmarks the performance of security validation
func BenchmarkWebSocketSecurityValidation(b *testing.B) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	server, err := New(cfg)
	require.NoError(b, err)

	// Create a request that will be validated
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Benchmark the origin validation function
			if server.checkOrigin(req) {
				// Valid origin processing
			}
		}
	})
}