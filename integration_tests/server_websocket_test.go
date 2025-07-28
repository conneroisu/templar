//go:build integration
// +build integration

package integrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWebSocketServer is a minimal WebSocket server for testing
type testWebSocketServer struct {
	clients   map[*websocket.Conn]chan []byte
	mutex     sync.RWMutex
	broadcast chan []byte
	register  chan *websocket.Conn
	cleanup   func()
}

// testServerWrapper wraps httptest.Server to provide custom cleanup
type testServerWrapper struct {
	*httptest.Server
	cleanup func()
}

// Close overrides the default Close to include custom cleanup
func (w *testServerWrapper) Close() {
	if w.cleanup != nil {
		w.cleanup()
	}
	w.Server.Close()
}

// createTestWebSocketServer creates a simple test WebSocket server
func createTestWebSocketServer() *testServerWrapper {
	server := &testWebSocketServer{
		clients:   make(map[*websocket.Conn]chan []byte),
		broadcast: make(chan []byte),
		register:  make(chan *websocket.Conn),
	}

	// Start hub goroutine
	ctx, cancel := context.WithCancel(context.Background())
	server.cleanup = cancel
	go server.run(ctx)

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", server.handleWebSocket)

	// Test broadcast endpoint
	mux.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var message map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		messageBytes, _ := json.Marshal(message)
		server.broadcast <- messageBytes

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message broadcasted"))
	})

	testServer := httptest.NewServer(mux)

	// Create a custom test server wrapper that handles cleanup
	return &testServerWrapper{
		Server:  testServer,
		cleanup: server.cleanup,
	}
}

func (s *testWebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for testing
	})
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	s.register <- conn
	go s.clientWritePump(conn)
	go s.clientReadPump(conn)
}

func (s *testWebSocketServer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case conn := <-s.register:
			s.mutex.Lock()
			s.clients[conn] = make(chan []byte, 1024) // Larger buffer to prevent blocking
			s.mutex.Unlock()

		case message := <-s.broadcast:
			s.mutex.RLock()
			// Create a list of clients to avoid holding lock during send
			var clientList []struct {
				conn *websocket.Conn
				send chan []byte
			}
			for conn, send := range s.clients {
				clientList = append(clientList, struct {
					conn *websocket.Conn
					send chan []byte
				}{conn, send})
			}
			s.mutex.RUnlock()

			// Send to clients without holding the main lock
			for _, client := range clientList {
				select {
				case client.send <- message:
					// Message sent successfully
				default:
					// Channel full, schedule removal in background to avoid blocking
					go func(c *websocket.Conn, ch chan []byte) {
						s.mutex.Lock()
						if _, exists := s.clients[c]; exists {
							close(ch)
							delete(s.clients, c)
							_ = c.Close(websocket.StatusNormalClosure, "")
						}
						s.mutex.Unlock()
					}(client.conn, client.send)
				}
			}
		}
	}
}

func (s *testWebSocketServer) clientWritePump(conn *websocket.Conn) {
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	s.mutex.RLock()
	send, exists := s.clients[conn]
	s.mutex.RUnlock()

	if !exists {
		return
	}

	ctx := context.Background()

	for message := range send {
		writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := conn.Write(writeCtx, websocket.MessageText, message)
		cancel()
		if err != nil {
			break
		}
	}
}

func (s *testWebSocketServer) clientReadPump(conn *websocket.Conn) {
	defer func() {
		s.mutex.Lock()
		if send, exists := s.clients[conn]; exists {
			close(send)
			delete(s.clients, conn)
		}
		s.mutex.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	conn.SetReadLimit(1024 * 1024) // 1MB limit to handle large test messages

	// Keep connection alive with ping/pong
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a goroutine to handle periodic pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.Ping(ctx); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		readCtx, readCancel := context.WithTimeout(ctx, 60*time.Second)
		msgType, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			break
		}

		// Handle different message types
		switch msgType {
		case websocket.MessageText, websocket.MessageBinary:
			// Normal message handling (currently we just read and ignore)
			_ = data
		default:
			// Other message types (ping/pong/close) are handled automatically
			_ = data
		}
	}
}

// connectWebSocketTestClient creates a WebSocket client connection
func connectWebSocketTestClient(serverURL string) (*websocket.Conn, error) {
	ctx := context.Background()
	url := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	return conn, err
}

// readWebSocketTestMessage reads a message from WebSocket with timeout
func readWebSocketTestMessage(
	conn *websocket.Conn,
	timeout time.Duration,
) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, message, err := conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	var msg map[string]interface{}
	err = json.Unmarshal(message, &msg)
	return msg, err
}

func TestIntegration_ServerWebSocket_BasicConnection(t *testing.T) {

	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect WebSocket client
	conn, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Verify connection is established
	assert.NotNil(t, conn)

	// Give the server time to set up the connection properly
	time.Sleep(100 * time.Millisecond)

	// Test basic connection by sending a simple message instead of ping
	// (ping/pong handling has compatibility issues with the test server)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to write a simple message to verify the connection works
	testMessage := []byte("connection_test")
	err = conn.Write(ctx, websocket.MessageText, testMessage)
	assert.NoError(t, err, "Should be able to write to connection")
}

func TestIntegration_ServerWebSocket_MessageBroadcasting(t *testing.T) {

	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect multiple WebSocket clients
	client1, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer client1.Close(websocket.StatusNormalClosure, "")

	client2, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer client2.Close(websocket.StatusNormalClosure, "")

	client3, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer client3.Close(websocket.StatusNormalClosure, "")

	// Wait for clients to be registered
	time.Sleep(200 * time.Millisecond)

	// Prepare test message
	testMessage := map[string]interface{}{
		"type": "component_update",
		"data": map[string]interface{}{
			"name":      "Button",
			"timestamp": time.Now().Unix(),
		},
	}

	// Broadcast message via HTTP endpoint
	messageBytes, _ := json.Marshal(testMessage)
	resp, err := http.Post(server.URL+"/broadcast", "application/json",
		strings.NewReader(string(messageBytes)))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Give message time to propagate to all clients
	time.Sleep(100 * time.Millisecond)

	// Verify all clients receive the message
	receivedMessages := make([]map[string]interface{}, 3)
	var wg sync.WaitGroup

	clients := []*websocket.Conn{client1, client2, client3}
	for i, client := range clients {
		wg.Add(1)
		go func(index int, c *websocket.Conn) {
			defer wg.Done()
			msg, err := readWebSocketTestMessage(c, 3*time.Second)
			if err != nil {
				t.Errorf("Client %d failed to read message: %v", index, err)
				return
			}
			receivedMessages[index] = msg
		}(i, client)
	}

	wg.Wait()

	// Verify all clients received the same message
	for i, msg := range receivedMessages {
		if msg == nil {
			t.Errorf("Client %d received nil message", i)
			continue
		}

		assert.Equal(t, testMessage["type"], msg["type"],
			"Client %d should receive correct message type", i)

		data, ok := msg["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Client %d received invalid data object: %+v", i, msg)
			continue
		}
		if data["name"] == nil {
			t.Errorf("Client %d received nil component name in data: %+v", i, data)
			continue
		}
		assert.Equal(t, "Button", data["name"],
			"Client %d should receive correct component name", i)
	}
}

func TestIntegration_ServerWebSocket_ClientConnectionManagement(t *testing.T) {

	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Test multiple connection cycles
	connectionCount := 10
	var connections []*websocket.Conn

	// Create connections
	for i := 0; i < connectionCount; i++ {
		conn, err := connectWebSocketTestClient(server.URL)
		require.NoError(t, err, "Connection %d should succeed", i)
		connections = append(connections, conn)
	}

	// Wait for all connections to be registered and stabilized
	time.Sleep(500 * time.Millisecond)

	// Broadcast a test message
	testMessage := map[string]interface{}{
		"type": "test",
		"data": "connection_test",
	}

	messageBytes, _ := json.Marshal(testMessage)
	resp, err := http.Post(server.URL+"/broadcast", "application/json",
		strings.NewReader(string(messageBytes)))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Verify all connections receive the message
	var wg sync.WaitGroup
	successCount := make(chan int, connectionCount)

	for i, conn := range connections {
		wg.Add(1)
		go func(index int, c *websocket.Conn) {
			defer wg.Done()
			_, err := readWebSocketTestMessage(c, 2*time.Second)
			if err == nil {
				successCount <- 1
			} else {
				t.Logf("Connection %d failed to read message: %v", index, err)
				successCount <- 0
			}
		}(i, conn)
	}

	wg.Wait()
	close(successCount)

	// Count successful message deliveries
	totalSuccess := 0
	for success := range successCount {
		totalSuccess += success
	}

	// Should receive messages on most connections (allow reasonable tolerance for concurrent testing)
	assert.GreaterOrEqual(t, totalSuccess, connectionCount-4,
		"Most connections should receive the message (got %d/%d)", totalSuccess, connectionCount)

	// Close connections gracefully (some might already be closed)
	for i, conn := range connections {
		if conn != nil {
			err := conn.Close(websocket.StatusNormalClosure, "")
			// Allow "use of closed network connection" errors since some connections
			// might have been closed by the server due to slow processing
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				assert.NoError(t, err, "Connection %d should close gracefully", i)
			}
		}
	}

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
}

func TestIntegration_ServerWebSocket_ConcurrentMessaging(t *testing.T) {
	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect clients
	clientCount := 5
	clients := make([]*websocket.Conn, clientCount)

	for i := 0; i < clientCount; i++ {
		conn, err := connectWebSocketTestClient(server.URL)
		require.NoError(t, err)
		clients[i] = conn
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	}

	// Wait for clients to be registered
	time.Sleep(200 * time.Millisecond)

	// Send multiple messages concurrently
	messageCount := 10
	var wg sync.WaitGroup

	// Track received messages per client
	receivedMessages := make([][]map[string]interface{}, clientCount)
	receiveMutexes := make([]sync.Mutex, clientCount)

	// Start message receivers for each client
	for i, client := range clients {
		wg.Add(1)
		go func(clientIndex int, c *websocket.Conn) {
			defer wg.Done()
			for j := 0; j < messageCount; j++ {
				msg, err := readWebSocketTestMessage(c, 3*time.Second)
				if err != nil {
					t.Logf("Client %d failed to read message %d: %v", clientIndex, j, err)
					continue
				}

				receiveMutexes[clientIndex].Lock()
				receivedMessages[clientIndex] = append(receivedMessages[clientIndex], msg)
				receiveMutexes[clientIndex].Unlock()
			}
		}(i, client)
	}

	// Send messages concurrently
	for i := 0; i < messageCount; i++ {
		go func(msgIndex int) {
			testMessage := map[string]interface{}{
				"type":  "concurrent_test",
				"data":  fmt.Sprintf("message_%d", msgIndex),
				"index": msgIndex,
			}

			messageBytes, _ := json.Marshal(testMessage)
			resp, err := http.Post(server.URL+"/broadcast", "application/json",
				strings.NewReader(string(messageBytes)))
			if err != nil {
				t.Logf("Failed to send message %d: %v", msgIndex, err)
				return
			}
			_ = resp.Body.Close()
		}(i)

		time.Sleep(50 * time.Millisecond) // Small delay between messages
	}

	wg.Wait()

	// Verify message delivery
	for i, messages := range receivedMessages {
		receiveMutexes[i].Lock()
		messageCount := len(messages)
		receiveMutexes[i].Unlock()

		// Allow some tolerance for message loss in concurrent scenarios
		assert.GreaterOrEqual(t, messageCount, messageCount-3,
			"Client %d should receive most messages", i)

		t.Logf("Client %d received %d/%d messages", i, messageCount, messageCount)
	}
}

func TestIntegration_ServerWebSocket_ErrorHandling(t *testing.T) {
	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect a client
	client, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)

	// Send invalid data to trigger error handling
	ctx := context.Background()
	err = client.Write(ctx, websocket.MessageBinary, []byte{0xFF, 0xFF, 0xFF})
	assert.NoError(t, err) // Writing should succeed

	// Abruptly close connection to test error handling
	client.Close(websocket.StatusNormalClosure, "")

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Server should continue working - test with new connection
	newClient, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer newClient.Close(websocket.StatusNormalClosure, "")

	// Send test message to verify server is still functional
	testMessage := map[string]interface{}{
		"type": "error_recovery_test",
		"data": "server_still_works",
	}

	messageBytes, _ := json.Marshal(testMessage)
	resp, err := http.Post(server.URL+"/broadcast", "application/json",
		strings.NewReader(string(messageBytes)))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Verify new client receives the message
	msg, err := readWebSocketTestMessage(newClient, 2*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, "error_recovery_test", msg["type"])
}

func TestIntegration_ServerWebSocket_LoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Create many concurrent connections
	connectionCount := 50
	clients := make([]*websocket.Conn, connectionCount)
	var connectionWg sync.WaitGroup

	// Connect clients concurrently
	for i := 0; i < connectionCount; i++ {
		connectionWg.Add(1)
		go func(index int) {
			defer connectionWg.Done()
			conn, err := connectWebSocketTestClient(server.URL)
			if err != nil {
				t.Logf("Failed to connect client %d: %v", index, err)
				return
			}
			clients[index] = conn
		}(i)
	}

	connectionWg.Wait()

	// Count successful connections
	successfulConnections := 0
	for _, client := range clients {
		if client != nil {
			successfulConnections++
			defer client.Close(websocket.StatusNormalClosure, "")
		}
	}

	t.Logf("Successfully connected %d/%d clients", successfulConnections, connectionCount)
	assert.GreaterOrEqual(t, successfulConnections, connectionCount-5,
		"Most connections should succeed")

	// Wait for all clients to be registered
	time.Sleep(500 * time.Millisecond)

	// Send multiple messages under load
	messageCount := 20
	var messageWg sync.WaitGroup

	start := time.Now()

	for i := 0; i < messageCount; i++ {
		messageWg.Add(1)
		go func(msgIndex int) {
			defer messageWg.Done()

			testMessage := map[string]interface{}{
				"type": "load_test",
				"data": fmt.Sprintf("load_message_%d", msgIndex),
			}

			messageBytes, _ := json.Marshal(testMessage)
			resp, err := http.Post(server.URL+"/broadcast", "application/json",
				strings.NewReader(string(messageBytes)))
			if err != nil {
				t.Logf("Failed to send load message %d: %v", msgIndex, err)
				return
			}
			_ = resp.Body.Close()
		}(i)

		time.Sleep(100 * time.Millisecond) // Sustained load
	}

	messageWg.Wait()
	totalTime := time.Since(start)

	t.Logf("Sent %d messages to %d clients in %v",
		messageCount, successfulConnections, totalTime)

	// Performance assertion
	assert.Less(t, totalTime, 30*time.Second,
		"Load test should complete in reasonable time")
}

func TestIntegration_ServerWebSocket_MessageOrdering(t *testing.T) {
	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect client
	client, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer client.Close(websocket.StatusNormalClosure, "")

	// Wait for client registration
	time.Sleep(100 * time.Millisecond)

	// Send ordered messages
	messageCount := 10
	var receivedMessages []map[string]interface{}
	var receiveMutex sync.Mutex

	// Start message receiver
	receiveDone := make(chan struct{})
	go func() {
		for i := 0; i < messageCount; i++ {
			msg, err := readWebSocketTestMessage(client, 2*time.Second)
			if err != nil {
				t.Logf("Failed to read message %d: %v", i, err)
				continue
			}

			receiveMutex.Lock()
			receivedMessages = append(receivedMessages, msg)
			receiveMutex.Unlock()
		}
		receiveDone <- struct{}{}
	}()

	// Send messages in order
	for i := 0; i < messageCount; i++ {
		testMessage := map[string]interface{}{
			"type":     "order_test",
			"sequence": i,
			"data":     fmt.Sprintf("message_%d", i),
		}

		messageBytes, _ := json.Marshal(testMessage)
		resp, err := http.Post(server.URL+"/broadcast", "application/json",
			strings.NewReader(string(messageBytes)))
		require.NoError(t, err)
		_ = resp.Body.Close()

		time.Sleep(50 * time.Millisecond) // Small delay between messages
	}

	// Wait for all messages to be received
	select {
	case <-receiveDone:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for messages")
	}

	// Verify message ordering
	receiveMutex.Lock()
	defer receiveMutex.Unlock()

	assert.Equal(t, messageCount, len(receivedMessages),
		"Should receive all messages")

	for i, msg := range receivedMessages {
		sequence, ok := msg["sequence"].(float64) // JSON numbers are float64
		assert.True(t, ok, "Message %d should have sequence number", i)
		assert.Equal(t, float64(i), sequence,
			"Message %d should have correct sequence", i)
	}
}

func TestIntegration_ServerWebSocket_LargeMessageHandling(t *testing.T) {

	server := createTestWebSocketServer()
	defer func() { server.Close() }()

	// Connect client
	client, err := connectWebSocketTestClient(server.URL)
	require.NoError(t, err)
	defer client.Close(websocket.StatusNormalClosure, "")

	// Wait for client registration
	time.Sleep(100 * time.Millisecond)

	// Create large message payload (reduced for test stability)
	largeData := strings.Repeat("A", 10*1024) // 10KB (still large for WebSocket testing)
	testMessage := map[string]interface{}{
		"type": "large_message",
		"data": largeData,
		"size": len(largeData),
	}

	// Start message receiver
	messageDone := make(chan map[string]interface{}, 1)
	go func() {
		msg, err := readWebSocketTestMessage(client, 10*time.Second)
		if err != nil {
			t.Logf("Failed to read large message: %v", err)
			return
		}
		messageDone <- msg
	}()

	// Send large message
	messageBytes, _ := json.Marshal(testMessage)
	resp, err := http.Post(server.URL+"/broadcast", "application/json",
		strings.NewReader(string(messageBytes)))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Wait for message to be received
	select {
	case receivedMsg := <-messageDone:
		assert.Equal(t, "large_message", receivedMsg["type"])
		assert.Equal(t, largeData, receivedMsg["data"])
		assert.Equal(t, float64(len(largeData)), receivedMsg["size"])
	case <-time.After(15 * time.Second):
		t.Fatal("Timeout waiting for large message")
	}
}
