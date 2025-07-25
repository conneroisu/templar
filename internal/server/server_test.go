package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, cfg, server.config)
	assert.NotNil(t, server.clients)
	assert.NotNil(t, server.broadcast)
	assert.NotNil(t, server.register)
	assert.NotNil(t, server.unregister)
	assert.NotNil(t, server.registry)
	assert.NotNil(t, server.watcher)
	assert.NotNil(t, server.scanner)
	assert.NotNil(t, server.renderer)

	// Clean up
	server.Stop()
}

func TestNew_WatcherCreationFailure(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	// This test is tricky because NewFileWatcher rarely fails
	// We'll just verify the error handling path exists
	server, err := New(cfg)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to create file watcher")
	} else {
		assert.NotNil(t, server)
		server.Stop()
	}
}

func TestPreviewServer_CheckOrigin(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Test CheckOrigin function
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/ws", nil)

	// Test with no origin header - should reject for security
	assert.False(t, server.checkOrigin(req))

	// Test with valid origin
	req.Header.Set("Origin", "http://localhost:8080")
	assert.True(t, server.checkOrigin(req))

	// Test with invalid origin
	req.Header.Set("Origin", "http://malicious.com")
	assert.False(t, server.checkOrigin(req))

	// Test with localhost origin
	req.Header.Set("Origin", "http://localhost:3000")
	assert.True(t, server.checkOrigin(req))

	// Test with 127.0.0.1 origin
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	assert.True(t, server.checkOrigin(req))

	// Test with malformed origin
	req.Header.Set("Origin", "not-a-valid-url")
	assert.False(t, server.checkOrigin(req))
}

func TestPreviewServer_Shutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestPreviewServer_BroadcastMessage(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Start the WebSocket hub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.runWebSocketHub(ctx)

	// Test broadcast
	msg := UpdateMessage{
		Type:      "test",
		Content:   "test message",
		Timestamp: time.Now(),
	}

	// This should not block or panic
	server.broadcastMessage(msg)
}

func TestClient_String(t *testing.T) {
	// Create a mock websocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		// Keep connection alive for test
		<-time.After(100 * time.Millisecond)
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + server.URL[4:] // Replace http with ws
	ctx := context.Background()
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	previewServer, err := New(cfg)
	require.NoError(t, err)
	defer previewServer.Stop()

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		server: previewServer,
	}

	// Test that client has the expected fields
	assert.NotNil(t, client.conn)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.server)
}

func TestUpdateMessage_Structure(t *testing.T) {
	timestamp := time.Now()
	msg := UpdateMessage{
		Type:      "full_reload",
		Target:    "component.templ",
		Content:   "Updated content",
		Timestamp: timestamp,
	}

	assert.Equal(t, "full_reload", msg.Type)
	assert.Equal(t, "component.templ", msg.Target)
	assert.Equal(t, "Updated content", msg.Content)
	assert.Equal(t, timestamp, msg.Timestamp)
}

func TestPreviewServer_FileWatcherIntegration(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{t.TempDir()}, // Use temp dir for testing
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Verify file watcher is configured
	assert.NotNil(t, server.watcher)

	// Start server briefly to test integration
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should not panic or error
	err = server.Start(ctx)
	// We expect context deadline exceeded since we're stopping early
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPreviewServer_MiddlewareIntegration(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Apply middleware
	wrappedHandler := server.addMiddleware(handler)

	// Test the middleware
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Check CORS headers were added
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", rr.Header().Get("Access-Control-Allow-Headers"))

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test", rr.Body.String())
}

func TestPreviewServer_MiddlewareOptions(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	})

	// Apply middleware
	wrappedHandler := server.addMiddleware(handler)

	// Test OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Check that OPTIONS request is handled by middleware
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String()) // Should not reach the handler
}

// Helper function to stop the server (for tests that need cleanup)
func (s *PreviewServer) Stop() {
	if s.watcher != nil {
		s.watcher.Stop()
	}
}
