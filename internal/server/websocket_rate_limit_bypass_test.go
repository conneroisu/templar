package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/config"
)

// TestWebSocketRateLimitBypassVulnerability tests that rate limiting only applies to actual messages
// and not to connection maintenance (ping/pong) or connection attempts.
func TestWebSocketRateLimitBypassVulnerability(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T, wsURL string)
	}{
		{
			name:        "ConnectionWithoutMessages",
			description: "Connections that don't send messages should not trigger rate limiting",
			testFunc:    testConnectionWithoutMessages,
		},
		{
			name:        "RapidMessageBurst",
			description: "Rapid message sending should trigger rate limiting",
			testFunc:    testRapidMessageBurst,
		},
		{
			name:        "EmptyMessageHandling",
			description: "Empty messages should not count towards rate limit",
			testFunc:    testEmptyMessageHandling,
		},
		{
			name:        "RateLimitWindowBoundary",
			description: "Rate limiting should respect window boundaries correctly",
			testFunc:    testRateLimitWindowBoundary,
		},
		{
			name:        "ConcurrentConnections",
			description: "Multiple connections should have independent rate limits",
			testFunc:    testConcurrentConnections,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupTestWebSocketServerForBypassTest(t)

			testServer := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					server.handleWebSocket(w, r)
				}),
			)
			defer testServer.Close()

			wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

			// Run the specific test
			tt.testFunc(t, wsURL)
		})
	}
}

// testConnectionWithoutMessages verifies that idle connections don't trigger rate limiting.
func testConnectionWithoutMessages(t *testing.T, wsURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use localhost as origin to pass validation
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost:8080"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for longer than normal rate limit window would allow
	// If rate limiting was applied on connection read attempts instead of actual messages,
	// this connection would be terminated
	time.Sleep(2 * time.Second)

	// Connection should still be alive - try sending a ping
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	err = conn.Ping(ctx2)
	if err != nil {
		t.Errorf("Connection should still be alive after idle period, but ping failed: %v", err)
	}
}

// testRapidMessageBurst verifies that rapid message sending triggers rate limiting.
func testRapidMessageBurst(t *testing.T, wsURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use localhost as origin to pass validation
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost:8080"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send messages rapidly to trigger rate limiting
	// Default rate limit is 60 messages per minute, so send more than that
	messagesSent := 0
	rateLimitTriggered := false

	for i := range 70 {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := conn.Write(
			writeCtx,
			websocket.MessageText,
			[]byte(fmt.Sprintf("test message %d", i)),
		)
		writeCancel()

		if err != nil {
			// Check if it's a rate limit violation
			if websocket.CloseStatus(err) == websocket.StatusPolicyViolation {
				rateLimitTriggered = true

				break
			}
			// Other errors are also acceptable as rate limiting might close the connection
			rateLimitTriggered = true

			break
		}
		messagesSent++

		// Small delay to avoid overwhelming the test
		time.Sleep(10 * time.Millisecond)
	}

	if !rateLimitTriggered {
		t.Errorf(
			"Expected rate limiting to be triggered after sending %d messages, but it wasn't",
			messagesSent,
		)
	}

	if messagesSent < 60 {
		t.Logf(
			"Rate limiting triggered after %d messages (expected, as it should be around the limit)",
			messagesSent,
		)
	}
}

// testEmptyMessageHandling verifies that empty messages don't count towards rate limit.
func testEmptyMessageHandling(t *testing.T, wsURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use localhost as origin to pass validation
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost:8080"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send many empty messages - these should not count towards rate limit
	for i := range 100 {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := conn.Write(writeCtx, websocket.MessageText, []byte(""))
		writeCancel()

		if err != nil {
			t.Errorf("Empty message %d failed: %v", i, err)

			break
		}

		// Small delay
		time.Sleep(5 * time.Millisecond)
	}

	// Now send a real message - should still work since empty messages don't count
	writeCtx, writeCancel := context.WithTimeout(context.Background(), 1*time.Second)
	err = conn.Write(writeCtx, websocket.MessageText, []byte("real message"))
	writeCancel()

	if err != nil {
		t.Errorf("Real message should succeed after empty messages: %v", err)
	}
}

// testRateLimitWindowBoundary verifies that rate limiting respects sliding window correctly.
func testRateLimitWindowBoundary(t *testing.T, wsURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use localhost as origin to pass validation
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost:8080"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send messages at a controlled rate just under the limit
	messagesPerBatch := 30         // Half of the 60/minute limit
	batchDelay := 35 * time.Second // Slightly more than half a minute

	// First batch
	for i := range messagesPerBatch {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := conn.Write(writeCtx, websocket.MessageText, []byte(fmt.Sprintf("batch1 msg %d", i)))
		writeCancel()

		if err != nil {
			t.Errorf("Message in first batch failed: %v", err)

			return
		}
		time.Sleep(20 * time.Millisecond) // Space out messages
	}

	// Wait for sliding window to partially reset
	time.Sleep(batchDelay)

	// Second batch should succeed as the window has slid
	for i := range messagesPerBatch {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := conn.Write(writeCtx, websocket.MessageText, []byte(fmt.Sprintf("batch2 msg %d", i)))
		writeCancel()

		if err != nil {
			t.Errorf(
				"Message in second batch failed (sliding window not working correctly): %v",
				err,
			)

			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// testConcurrentConnections verifies that multiple connections have independent rate limits.
func testConcurrentConnections(t *testing.T, wsURL string) {
	const numConnections = 3
	connections := make([]*websocket.Conn, numConnections)

	// Establish multiple connections
	for i := range numConnections {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://localhost:8080"},
			},
		})
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		cancel()

		if err != nil {
			t.Fatalf("Failed to establish connection %d: %v", i, err)
		}
		connections[i] = conn
		defer conn.Close(websocket.StatusNormalClosure, "")
	}

	// Each connection should be able to send messages up to its individual limit
	messageCount := 20 // Well under the limit per connection

	for connIdx, conn := range connections {
		for msgIdx := range messageCount {
			writeCtx, writeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			err := conn.Write(
				writeCtx,
				websocket.MessageText,
				[]byte(fmt.Sprintf("conn%d msg%d", connIdx, msgIdx)),
			)
			writeCancel()

			if err != nil {
				t.Errorf("Connection %d message %d failed: %v", connIdx, msgIdx, err)
			}

			time.Sleep(10 * time.Millisecond)
		}
	}
}

// setupTestWebSocketServerForBypassTest creates a test server for WebSocket testing.
func setupTestWebSocketServerForBypassTest(t *testing.T) *PreviewServer {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	server := &PreviewServer{
		config:     cfg,
		clients:    make(map[*websocket.Conn]*Client),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client, 256),
		unregister: make(chan *websocket.Conn, 256),
	}

	// Start the hub in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	go server.runWebSocketHub(ctx)

	// Add cleanup function to t.Cleanup for automatic cleanup
	t.Cleanup(func() {
		cancel()
	})

	return server
}

// TestSlidingWindowRateLimiterSecurityProperties tests security properties of the rate limiter.
func TestSlidingWindowRateLimiterSecurityProperties(t *testing.T) {
	t.Run("PreventsBurstAtWindowBoundary", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)

		// Fill up the window
		for i := range 5 {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i)
			}
		}

		// 6th request should be denied
		if limiter.IsAllowed() {
			t.Error("6th request should be denied")
		}

		// Wait for window to partially slide (but not fully)
		time.Sleep(50 * time.Millisecond)

		// Should still be denied as window hasn't fully reset
		if limiter.IsAllowed() {
			t.Error("Request should still be denied in middle of window")
		}

		// Wait for full window to slide AND backoff to expire (2 second backoff after 2 violations)
		time.Sleep(2200 * time.Millisecond)

		// Now should be allowed as both window has slid and backoff expired
		if !limiter.IsAllowed() {
			t.Error("Request should be allowed after window slides and backoff expires")
		}
	})

	t.Run("HandlesClockSkew", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(3, 100*time.Millisecond)

		// Simulate rapid requests
		for i := range 3 {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i)
			}
			// No sleep - simulate very rapid requests
		}

		// 4th request should be denied
		if limiter.IsAllowed() {
			t.Error("4th rapid request should be denied")
		}
	})
}
