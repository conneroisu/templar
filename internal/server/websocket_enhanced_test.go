package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/config"
)

func TestWebSocketEnhancements_BasicFunctionality(t *testing.T) {
	enhancements := NewWebSocketEnhancements()
	defer enhancements.cancel()

	// Test IP limit checking
	testIP := "192.168.1.100"
	if !enhancements.checkIPLimit(testIP) {
		t.Error("Fresh IP should be allowed")
	}

	// Test configuration
	if enhancements.maxConnectionsPerIP != 20 {
		t.Errorf("Expected maxConnectionsPerIP=20, got %d", enhancements.maxConnectionsPerIP)
	}

	if enhancements.maxMessagesPerMinute != 120 {
		t.Errorf("Expected maxMessagesPerMinute=120, got %d", enhancements.maxMessagesPerMinute)
	}
}

func TestWebSocketEnhancements_IPTracking(t *testing.T) {
	enhancements := NewWebSocketEnhancements()
	defer enhancements.cancel()

	testIP := "192.168.1.100"

	// Create mock connections
	mockConns := make([]*websocket.Conn, 5)
	for i := range mockConns {
		// Use nil for mock connections in this test
		mockConns[i] = nil
		enhancements.trackIPConnection(testIP, mockConns[i])
	}

	// Check if IP limit is working
	for i := range enhancements.maxConnectionsPerIP - 5 {
		if !enhancements.checkIPLimit(testIP) {
			t.Errorf("IP should be allowed at connection %d", i+5)
		}
		enhancements.trackIPConnection(testIP, nil)
	}

	// Should now be at limit
	if enhancements.checkIPLimit(testIP) {
		t.Error("IP should be rejected after reaching limit")
	}

	// Test untracking
	for _, conn := range mockConns {
		enhancements.untrackIPConnection(testIP, conn)
	}

	// Should be allowed again
	if !enhancements.checkIPLimit(testIP) {
		t.Error("IP should be allowed after untracking connections")
	}
}

func TestEnhancedWebSocket_Integration(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Test server will assign port
		},
	}

	server := &PreviewServer{
		config:     cfg,
		clients:    make(map[*websocket.Conn]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *websocket.Conn),
	}

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocketEnhanced(w, r)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		t.Fatalf("Failed to connect to enhanced WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for connection to be processed
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := server.GetEnhancedMetrics()
	if metrics["current_connections"].(int) != 1 {
		t.Errorf("Expected 1 current connection, got %d", metrics["current_connections"])
	}

	if server.enhancements != nil {
		totalConns := metrics["total_connections"].(int64)
		if totalConns != 1 {
			t.Errorf("Expected 1 total connection, got %d", totalConns)
		}
	}
}

func TestEnhancedWebSocket_RateLimiting(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server := &PreviewServer{
		config:       cfg,
		clients:      make(map[*websocket.Conn]*Client),
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *websocket.Conn),
		enhancements: NewWebSocketEnhancements(),
	}
	defer server.enhancements.cancel()

	// Set low limit for testing
	server.enhancements.maxConnectionsPerIP = 2

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocketEnhanced(w, r)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect up to limit
	connections := make([]*websocket.Conn, 2)
	for i := range 2 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://localhost"},
			},
		})
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		cancel()

		if err != nil {
			t.Fatalf("Connection %d should succeed: %v", i, err)
		}
		connections[i] = conn
	}

	// Third connection should fail
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://localhost"},
		},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err == nil {
		t.Error("Third connection should fail due to IP limit")
	} else if resp != nil && resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected 429 status code, got %d", resp.StatusCode)
	}

	// Clean up
	for _, conn := range connections {
		if conn != nil {
			conn.Close(websocket.StatusNormalClosure, "")
		}
	}
}

func TestEnhancedWebSocket_Broadcasting(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server := &PreviewServer{
		config:       cfg,
		clients:      make(map[*websocket.Conn]*Client),
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *websocket.Conn),
		enhancements: NewWebSocketEnhancements(),
	}
	defer server.enhancements.cancel()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocketEnhanced(w, r)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients
	numClients := 3
	connections := make([]*websocket.Conn, numClients)

	for i := range numClients {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://localhost"},
			},
		})
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		cancel()

		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		connections[i] = conn
	}

	// Wait for connections to be registered
	time.Sleep(200 * time.Millisecond)

	// Test enhanced broadcasting
	testMessage := []byte("enhanced test broadcast")
	server.broadcastEnhanced(testMessage)

	// Wait for broadcast to complete
	time.Sleep(100 * time.Millisecond)

	// Verify metrics
	metrics := server.GetEnhancedMetrics()
	if metrics["current_connections"].(int) != numClients {
		t.Errorf("Expected %d connections, got %d", numClients, metrics["current_connections"])
	}

	// Clean up
	for _, conn := range connections {
		conn.Close(websocket.StatusNormalClosure, "")
	}
}

// Benchmark the enhanced WebSocket implementation.
func BenchmarkEnhancedWebSocket_100Clients(b *testing.B) {
	benchmarkEnhancedWebSocket(b, 100)
}

func BenchmarkEnhancedWebSocket_500Clients(b *testing.B) {
	benchmarkEnhancedWebSocket(b, 500)
}

func benchmarkEnhancedWebSocket(b *testing.B, numClients int) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server := &PreviewServer{
		config:       cfg,
		clients:      make(map[*websocket.Conn]*Client),
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *websocket.Conn),
		enhancements: NewWebSocketEnhancements(),
	}
	defer server.enhancements.cancel()

	// Increase limits for benchmarking
	server.enhancements.maxConnectionsPerIP = numClients + 100

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocketEnhanced(w, r)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		b.StopTimer()

		// Connect clients
		connections := make([]*websocket.Conn, numClients)
		var wg sync.WaitGroup

		start := time.Now()
		for j := range numClients {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost"},
					},
				})
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				if err != nil {
					b.Errorf("Failed to connect client %d: %v", index, err)

					return
				}
				connections[index] = conn
			}(j)
		}
		wg.Wait()
		connectionTime := time.Since(start)

		b.StartTimer()

		// Benchmark broadcasting
		broadcastStart := time.Now()
		for broadcast := range 10 {
			message := fmt.Sprintf("Enhanced broadcast %d", broadcast)
			server.broadcastEnhanced([]byte(message))
		}
		broadcastTime := time.Since(broadcastStart)

		b.StopTimer()

		// Clean up
		cleanup := time.Now()
		for _, conn := range connections {
			if conn != nil {
				conn.Close(websocket.StatusNormalClosure, "")
			}
		}
		cleanupTime := time.Since(cleanup)

		// Report detailed metrics
		b.ReportMetric(float64(connectionTime.Nanoseconds()), "connection_ns")
		b.ReportMetric(float64(broadcastTime.Nanoseconds()), "broadcast_ns")
		b.ReportMetric(float64(cleanupTime.Nanoseconds()), "cleanup_ns")

		metrics := server.GetEnhancedMetrics()
		b.ReportMetric(float64(metrics["current_connections"].(int)), "final_client_count")
	}
}

// Performance comparison between original and enhanced implementations.
func BenchmarkComparison_OriginalVsEnhanced(b *testing.B) {
	numClients := 100

	b.Run("Original", func(b *testing.B) {
		benchmarkOriginalWebSocket(b, numClients)
	})

	b.Run("Enhanced", func(b *testing.B) {
		benchmarkEnhancedWebSocket(b, numClients)
	})
}

func benchmarkOriginalWebSocket(b *testing.B, numClients int) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server := &PreviewServer{
		config:     cfg,
		clients:    make(map[*websocket.Conn]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *websocket.Conn),
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		b.StopTimer()

		// Connect clients
		connections := make([]*websocket.Conn, numClients)
		var wg sync.WaitGroup

		start := time.Now()
		for j := range numClients {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
					HTTPHeader: http.Header{
						"Origin": []string{"http://localhost"},
					},
				})
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				if err != nil {
					b.Errorf("Failed to connect client %d: %v", index, err)

					return
				}
				connections[index] = conn
			}(j)
		}
		wg.Wait()
		connectionTime := time.Since(start)

		b.StartTimer()

		// Benchmark original broadcasting via message struct
		broadcastStart := time.Now()
		for broadcast := range 10 {
			msg := UpdateMessage{
				Type:      "test",
				Content:   fmt.Sprintf("Original broadcast %d", broadcast),
				Timestamp: time.Now(),
			}
			server.broadcastMessage(msg)
		}
		broadcastTime := time.Since(broadcastStart)

		b.StopTimer()

		// Clean up
		cleanup := time.Now()
		for _, conn := range connections {
			if conn != nil {
				conn.Close(websocket.StatusNormalClosure, "")
			}
		}
		cleanupTime := time.Since(cleanup)

		b.ReportMetric(float64(connectionTime.Nanoseconds()), "connection_ns")
		b.ReportMetric(float64(broadcastTime.Nanoseconds()), "broadcast_ns")
		b.ReportMetric(float64(cleanupTime.Nanoseconds()), "cleanup_ns")
	}
}

func TestEnhancedWebSocket_Shutdown(t *testing.T) {
	server := &PreviewServer{
		enhancements: NewWebSocketEnhancements(),
	}

	// Test shutdown
	start := time.Now()
	server.ShutdownEnhancements()
	shutdownTime := time.Since(start)

	// Shutdown should be quick
	if shutdownTime > 2*time.Second {
		t.Errorf("Shutdown took too long: %v", shutdownTime)
	}
}

func TestEnhancedWebSocket_Metrics(t *testing.T) {
	server := &PreviewServer{
		clients:      make(map[*websocket.Conn]*Client),
		enhancements: NewWebSocketEnhancements(),
	}
	defer server.enhancements.cancel()

	metrics := server.GetEnhancedMetrics()

	// Check expected metrics fields
	expectedFields := []string{
		"total_connections",
		"rejected_connections",
		"max_connections_per_ip",
		"max_messages_per_minute",
		"cleanup_workers",
		"tracked_ips",
		"current_connections",
	}

	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Expected metric field '%s' not found", field)
		}
	}

	// Verify some values
	if metrics["max_connections_per_ip"].(int) != 20 {
		t.Errorf("Expected max_connections_per_ip=20, got %d", metrics["max_connections_per_ip"])
	}

	if metrics["cleanup_workers"].(int) != 4 {
		t.Errorf("Expected cleanup_workers=4, got %d", metrics["cleanup_workers"])
	}
}
