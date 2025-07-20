package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"nhooyr.io/websocket"
)

// FuzzWebSocketOriginValidation tests origin validation with various malicious inputs
func FuzzWebSocketOriginValidation(f *testing.F) {
	// Seed with valid and invalid origins
	f.Add("http://localhost:8080")
	f.Add("https://localhost:8080")
	f.Add("http://127.0.0.1:8080")
	f.Add("javascript:alert('xss')")
	f.Add("data:text/html,<script>alert('xss')</script>")
	f.Add("file:///etc/passwd")
	f.Add("ftp://malicious.com")
	f.Add("http://malicious.com")
	f.Add("http://localhost:8080/../admin")
	f.Add("http://localhost:8080@malicious.com")
	f.Add("http://localhost\x00:8080")
	f.Add("")

	f.Fuzz(func(t *testing.T, origin string) {
		if len(origin) > 10000 {
			t.Skip("Origin too long")
		}

		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		server := &PreviewServer{
			config: cfg,
		}

		req := httptest.NewRequest("GET", "/ws", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}

		// Test that checkOrigin doesn't panic and correctly rejects malicious origins
		result := server.checkOrigin(req)

		// If origin validation passed, ensure it's actually safe
		if result {
			parsedOrigin, err := url.Parse(origin)
			if err != nil {
				t.Errorf("Origin validation passed for unparseable origin: %q", origin)
				return
			}

			// Ensure only http/https schemes are allowed
			if parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https" {
				t.Errorf("Origin validation passed for non-http(s) scheme: %q", origin)
			}

			// Ensure no control characters in host
			if strings.ContainsAny(parsedOrigin.Host, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
				t.Errorf("Origin validation passed for host with control characters: %q", origin)
			}

			// Ensure host is in allowed list
			allowedHosts := []string{
				"localhost:8080",
				"127.0.0.1:8080",
				"localhost:3000",
				"127.0.0.1:3000",
			}

			allowed := false
			for _, allowedHost := range allowedHosts {
				if parsedOrigin.Host == allowedHost {
					allowed = true
					break
				}
			}
			if !allowed {
				t.Errorf("Origin validation passed for non-allowed host: %q", parsedOrigin.Host)
			}
		}
	})
}

// FuzzWebSocketMessage tests WebSocket message handling with various payloads
func FuzzWebSocketMessage(f *testing.F) {
	// Seed with various message types and potentially dangerous content
	f.Add(`{"type":"reload"}`)
	f.Add(`{"type":"ping"}`)
	f.Add(`{"type":"malicious","payload":"<script>alert('xss')</script>"}`)
	f.Add(`{"type":"large","payload":"` + strings.Repeat("A", 1000) + `"}`)
	f.Add(`malformed json`)
	f.Add(`{"type":"command","payload":"rm -rf /"}`)
	f.Add(`null`)
	f.Add(`[]`)
	f.Add(`{"type":null}`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, message string) {
		if len(message) > maxMessageSize*2 {
			t.Skip("Message too large")
		}

		// Create a test WebSocket connection
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			// Test reading the fuzzed message
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, receivedMessage, err := conn.Read(ctx)
			_ = err // Expect many messages to cause errors

			// Ensure received message doesn't contain dangerous content that could be executed
			if receivedMessage != nil {
				msgStr := string(receivedMessage)
				if strings.Contains(msgStr, "<script>") ||
					strings.Contains(msgStr, "javascript:") ||
					strings.Contains(msgStr, "rm -rf") ||
					strings.ContainsAny(msgStr, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
					t.Errorf("Received dangerous message content: %q", msgStr)
				}
			}
		}))
		defer server.Close()

		// Convert HTTP URL to WebSocket URL
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Connect to the test server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Skip("Could not connect to test server")
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		// Send the fuzzed message
		err = conn.Write(ctx, websocket.MessageText, []byte(message))
		_ = err // Many messages will cause write errors, which is expected
	})
}

// FuzzWebSocketHeaders tests WebSocket upgrade with various header combinations
func FuzzWebSocketHeaders(f *testing.F) {
	// Seed with various header combinations
	f.Add("Upgrade\x00WebSocket\x00Connection\x00upgrade")
	f.Add("Origin\x00http://malicious.com")
	f.Add("Sec-WebSocket-Protocol\x00dangerous-protocol")
	f.Add("User-Agent\x00<script>alert('xss')</script>")
	f.Add("X-Forwarded-For\x00127.0.0.1, malicious.com")

	f.Fuzz(func(t *testing.T, headerData string) {
		if len(headerData) > 5000 {
			t.Skip("Header data too large")
		}

		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		server := &PreviewServer{
			config: cfg,
		}

		req := httptest.NewRequest("GET", "/ws", nil)

		// Parse header data and add to request
		headers := strings.Split(headerData, "\x00")
		for i := 0; i < len(headers)-1; i += 2 {
			if i+1 < len(headers) {
				key := headers[i]
				value := headers[i+1]

				// Skip headers with control characters that would break HTTP
				if strings.ContainsAny(key, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") ||
					strings.ContainsAny(value, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
					continue
				}

				req.Header.Set(key, value)
			}
		}

		// Test that WebSocket handling doesn't panic with malformed headers
		w := httptest.NewRecorder()

		// This should not panic regardless of header content
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("WebSocket handler panicked with headers: %v", r)
				}
			}()

			server.handleWebSocket(w, req)
		}()
	})
}

// FuzzWebSocketURL tests WebSocket endpoint with various URL patterns
func FuzzWebSocketURL(f *testing.F) {
	// Seed with various URL patterns and potential attacks
	f.Add("/ws")
	f.Add("/ws/../admin")
	f.Add("/ws?param=value")
	f.Add("/ws?param=<script>alert('xss')</script>")
	f.Add("/ws#fragment")
	f.Add("/ws%00admin")
	f.Add("/ws\x00admin")
	f.Add("/ws?../../../etc/passwd")

	f.Fuzz(func(t *testing.T, urlPath string) {
		if len(urlPath) > 2000 {
			t.Skip("URL path too long")
		}

		// Skip URLs with control characters that would break HTTP
		if strings.ContainsAny(urlPath, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
			t.Skip("URL contains control characters")
		}

		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		server := &PreviewServer{
			config: cfg,
		}

		req := httptest.NewRequest("GET", urlPath, nil)
		req.Header.Set("Origin", "http://localhost:8080")

		w := httptest.NewRecorder()

		// Test that URL handling doesn't cause security issues
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("WebSocket handler panicked with URL %q: %v", urlPath, r)
				}
			}()

			server.handleWebSocket(w, req)

			// If response contains error, ensure it doesn't leak sensitive information
			response := w.Body.String()
			if strings.Contains(response, "/etc/passwd") ||
				strings.Contains(response, "C:\\Windows") ||
				strings.Contains(response, "Administrator") {
				t.Errorf("Response contains sensitive path information: %q", response)
			}
		}()
	})
}
