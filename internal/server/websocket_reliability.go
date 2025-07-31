// Package server provides WebSocket reliability features including automatic
// reconnection, connection status tracking, and offline mode support.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// ConnectionStatus represents the current WebSocket connection state.
type ConnectionStatus int32

const (
	// StatusDisconnected indicates no active connection.
	StatusDisconnected ConnectionStatus = iota
	// StatusConnecting indicates attempting to establish connection.
	StatusConnecting
	// StatusConnected indicates active, healthy connection.
	StatusConnected
	// StatusReconnecting indicates attempting to reconnect after failure.
	StatusReconnecting
	// StatusOffline indicates offline mode (user-initiated or persistent failure).
	StatusOffline
)

// String returns a string representation of the connection status.
func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusReconnecting:
		return "reconnecting"
	case StatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// ReconnectionConfig defines reconnection behavior parameters.
type ReconnectionConfig struct {
	// InitialDelay is the initial delay before first reconnection attempt.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between reconnection attempts.
	MaxDelay time.Duration
	// BackoffMultiplier controls exponential backoff growth rate.
	BackoffMultiplier float64
	// MaxAttempts is the maximum number of reconnection attempts (0 = unlimited).
	MaxAttempts int
	// JitterFactor adds randomization to backoff delays (0.0-1.0).
	JitterFactor float64
	// HealthCheckInterval is how often to ping the connection.
	HealthCheckInterval time.Duration
	// ConnectionTimeout is the timeout for establishing connections.
	ConnectionTimeout time.Duration
}

// DefaultReconnectionConfig returns a sensible default configuration.
func DefaultReconnectionConfig() *ReconnectionConfig {
	return &ReconnectionConfig{
		InitialDelay:        1 * time.Second,
		MaxDelay:            30 * time.Second,
		BackoffMultiplier:   2.0,
		MaxAttempts:         0, // Unlimited
		JitterFactor:        0.1,
		HealthCheckInterval: 30 * time.Second,
		ConnectionTimeout:   10 * time.Second,
	}
}

// ConnectionMetrics tracks connection reliability statistics.
type ConnectionMetrics struct {
	// Connection attempt counters
	TotalAttempts   int64
	SuccessfulConns int64
	FailedAttempts  int64

	// Reconnection statistics
	ReconnectAttempts int64
	ReconnectSuccess  int64

	// Connection duration tracking
	TotalConnectedTime int64 // nanoseconds
	LongestConnection  int64 // nanoseconds
	AverageConnection  int64 // nanoseconds

	// Health check statistics
	PingSent     int64
	PongReceived int64
	TimeoutCount int64

	// Error tracking
	NetworkErrors    int64
	ProtocolErrors   int64
	UnexpectedErrors int64

	// Current connection info
	CurrentConnStart    int64 // Unix nano timestamp
	LastDisconnect      int64 // Unix nano timestamp
	ConsecutiveFailures int64
}

// StatusCallback is called when connection status changes.
type StatusCallback func(status ConnectionStatus, err error)

// MessageCallback is called when a message is received.
type MessageCallback func(messageType websocket.MessageType, data []byte)

// WebSocketReliabilityManager manages WebSocket connections with automatic
// reconnection, status tracking, and offline mode support.
type WebSocketReliabilityManager struct {
	// Configuration
	config *ReconnectionConfig
	url    string

	// Connection management
	conn      *websocket.Conn
	connMutex sync.RWMutex
	status    int32 // atomic ConnectionStatus

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Callback handlers
	statusCallbacks  []StatusCallback
	messageCallbacks []MessageCallback
	callbackMutex    sync.RWMutex

	// Metrics and monitoring
	metrics *ConnectionMetrics

	// Internal state
	// TODO: Implement periodic reconnect and health checking
	reconnectTicker *time.Ticker //nolint:unused
	healthTicker    *time.Ticker //nolint:unused
	currentDelay    time.Duration
	attemptCount    int64
	forceOffline    int32 // atomic bool

	// Message queuing for offline mode
	messageQueue [][]byte
	queueMutex   sync.Mutex
	maxQueueSize int
}

// NewWebSocketReliabilityManager creates a new WebSocket reliability manager.
func NewWebSocketReliabilityManager(url string, config *ReconnectionConfig) *WebSocketReliabilityManager {
	if config == nil {
		config = DefaultReconnectionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketReliabilityManager{
		config:           config,
		url:              url,
		ctx:              ctx,
		cancel:           cancel,
		status:           int32(StatusDisconnected),
		metrics:          &ConnectionMetrics{},
		currentDelay:     config.InitialDelay,
		maxQueueSize:     100,
		statusCallbacks:  make([]StatusCallback, 0),
		messageCallbacks: make([]MessageCallback, 0),
		messageQueue:     make([][]byte, 0),
	}
}

// Start begins the WebSocket connection management.
func (wsrm *WebSocketReliabilityManager) Start() error {
	// Start the main connection management goroutine
	wsrm.wg.Add(1)
	go wsrm.connectionManager()

	// Start health checking goroutine
	wsrm.wg.Add(1)
	go wsrm.healthChecker()

	return nil
}

// Stop gracefully shuts down the WebSocket manager.
func (wsrm *WebSocketReliabilityManager) Stop() error {
	wsrm.cancel()
	wsrm.wg.Wait()

	wsrm.connMutex.Lock()
	if wsrm.conn != nil {
		_ = wsrm.conn.Close(websocket.StatusNormalClosure, "Shutting down")
		wsrm.conn = nil
	}
	wsrm.connMutex.Unlock()

	return nil
}

// Connect attempts to establish a WebSocket connection.
func (wsrm *WebSocketReliabilityManager) Connect() error {
	if atomic.LoadInt32(&wsrm.forceOffline) == 1 {
		return errors.New("manager is in offline mode")
	}

	wsrm.setStatus(StatusConnecting)

	ctx, cancel := context.WithTimeout(wsrm.ctx, wsrm.config.ConnectionTimeout)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsrm.url, nil)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		atomic.AddInt64(&wsrm.metrics.FailedAttempts, 1)
		atomic.AddInt64(&wsrm.metrics.ConsecutiveFailures, 1)
		wsrm.updateError(err)
		wsrm.setStatus(StatusDisconnected)

		return fmt.Errorf("failed to connect: %w", err)
	}

	wsrm.connMutex.Lock()
	wsrm.conn = conn
	wsrm.connMutex.Unlock()

	// Reset reconnection state on successful connection
	wsrm.currentDelay = wsrm.config.InitialDelay
	atomic.StoreInt64(&wsrm.attemptCount, 0)
	atomic.StoreInt64(&wsrm.metrics.ConsecutiveFailures, 0)

	// Update metrics
	atomic.AddInt64(&wsrm.metrics.TotalAttempts, 1)
	atomic.AddInt64(&wsrm.metrics.SuccessfulConns, 1)
	wsrm.metrics.CurrentConnStart = time.Now().UnixNano()

	wsrm.setStatus(StatusConnected)

	// Start read pump for this connection
	wsrm.wg.Add(1)
	go wsrm.readPump(conn)

	// Process any queued messages
	wsrm.processQueuedMessages()

	return nil
}

// Disconnect closes the current connection.
func (wsrm *WebSocketReliabilityManager) Disconnect() error {
	wsrm.connMutex.Lock()
	defer wsrm.connMutex.Unlock()

	if wsrm.conn != nil {
		err := wsrm.conn.Close(websocket.StatusNormalClosure, "User disconnect")
		wsrm.conn = nil
		wsrm.updateConnectionDuration()
		wsrm.setStatus(StatusDisconnected)

		return err
	}

	return nil
}

// SetOfflineMode enables or disables offline mode.
func (wsrm *WebSocketReliabilityManager) SetOfflineMode(offline bool) {
	if offline {
		atomic.StoreInt32(&wsrm.forceOffline, 1)
		_ = wsrm.Disconnect()
		wsrm.setStatus(StatusOffline)
	} else {
		atomic.StoreInt32(&wsrm.forceOffline, 0)
		wsrm.setStatus(StatusDisconnected)
		// Trigger reconnection attempt
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = wsrm.Connect()
		}()
	}
}

// SendMessage sends a message over the WebSocket connection.
func (wsrm *WebSocketReliabilityManager) SendMessage(messageType websocket.MessageType, data []byte) error {
	wsrm.connMutex.RLock()
	conn := wsrm.conn
	wsrm.connMutex.RUnlock()

	if conn == nil {
		// Queue message for when connection is restored
		wsrm.queueMessage(data)

		return errors.New("no active connection, message queued")
	}

	ctx, cancel := context.WithTimeout(wsrm.ctx, 10*time.Second)
	defer cancel()

	return conn.Write(ctx, messageType, data)
}

// GetStatus returns the current connection status.
func (wsrm *WebSocketReliabilityManager) GetStatus() ConnectionStatus {
	return ConnectionStatus(atomic.LoadInt32(&wsrm.status))
}

// GetMetrics returns a copy of current connection metrics.
func (wsrm *WebSocketReliabilityManager) GetMetrics() ConnectionMetrics {
	return ConnectionMetrics{
		TotalAttempts:       atomic.LoadInt64(&wsrm.metrics.TotalAttempts),
		SuccessfulConns:     atomic.LoadInt64(&wsrm.metrics.SuccessfulConns),
		FailedAttempts:      atomic.LoadInt64(&wsrm.metrics.FailedAttempts),
		ReconnectAttempts:   atomic.LoadInt64(&wsrm.metrics.ReconnectAttempts),
		ReconnectSuccess:    atomic.LoadInt64(&wsrm.metrics.ReconnectSuccess),
		TotalConnectedTime:  atomic.LoadInt64(&wsrm.metrics.TotalConnectedTime),
		LongestConnection:   atomic.LoadInt64(&wsrm.metrics.LongestConnection),
		AverageConnection:   atomic.LoadInt64(&wsrm.metrics.AverageConnection),
		PingSent:            atomic.LoadInt64(&wsrm.metrics.PingSent),
		PongReceived:        atomic.LoadInt64(&wsrm.metrics.PongReceived),
		TimeoutCount:        atomic.LoadInt64(&wsrm.metrics.TimeoutCount),
		NetworkErrors:       atomic.LoadInt64(&wsrm.metrics.NetworkErrors),
		ProtocolErrors:      atomic.LoadInt64(&wsrm.metrics.ProtocolErrors),
		UnexpectedErrors:    atomic.LoadInt64(&wsrm.metrics.UnexpectedErrors),
		CurrentConnStart:    atomic.LoadInt64(&wsrm.metrics.CurrentConnStart),
		LastDisconnect:      atomic.LoadInt64(&wsrm.metrics.LastDisconnect),
		ConsecutiveFailures: atomic.LoadInt64(&wsrm.metrics.ConsecutiveFailures),
	}
}

// AddStatusCallback adds a status change callback.
func (wsrm *WebSocketReliabilityManager) AddStatusCallback(callback StatusCallback) {
	wsrm.callbackMutex.Lock()
	defer wsrm.callbackMutex.Unlock()
	wsrm.statusCallbacks = append(wsrm.statusCallbacks, callback)
}

// AddMessageCallback adds a message received callback.
func (wsrm *WebSocketReliabilityManager) AddMessageCallback(callback MessageCallback) {
	wsrm.callbackMutex.Lock()
	defer wsrm.callbackMutex.Unlock()
	wsrm.messageCallbacks = append(wsrm.messageCallbacks, callback)
}

// connectionManager handles the main connection lifecycle and reconnection logic.
func (wsrm *WebSocketReliabilityManager) connectionManager() {
	defer wsrm.wg.Done()

	// Initial connection attempt
	if err := wsrm.Connect(); err != nil {
		log.Printf("Initial WebSocket connection failed: %v", err)
	}

	for {
		select {
		case <-wsrm.ctx.Done():
			return

		default:
			// Check if we need to reconnect
			status := wsrm.GetStatus()
			if status == StatusDisconnected && atomic.LoadInt32(&wsrm.forceOffline) == 0 {
				wsrm.attemptReconnection()
			}

			// Wait before next check
			time.Sleep(1 * time.Second)
		}
	}
}

// attemptReconnection handles the reconnection logic with exponential backoff.
func (wsrm *WebSocketReliabilityManager) attemptReconnection() {
	// Check if we've exceeded max attempts
	if wsrm.config.MaxAttempts > 0 && atomic.LoadInt64(&wsrm.attemptCount) >= int64(wsrm.config.MaxAttempts) {
		wsrm.setStatus(StatusOffline)

		return
	}

	wsrm.setStatus(StatusReconnecting)
	atomic.AddInt64(&wsrm.attemptCount, 1)
	atomic.AddInt64(&wsrm.metrics.ReconnectAttempts, 1)

	// Calculate delay with jitter
	delay := wsrm.calculateBackoffDelay()

	select {
	case <-wsrm.ctx.Done():
		return
	case <-time.After(delay):
		// Attempt reconnection
		if err := wsrm.Connect(); err != nil {
			log.Printf("WebSocket reconnection attempt %d failed: %v",
				atomic.LoadInt64(&wsrm.attemptCount), err)

			// Increase delay for next attempt
			wsrm.currentDelay = time.Duration(float64(wsrm.currentDelay) * wsrm.config.BackoffMultiplier)
			if wsrm.currentDelay > wsrm.config.MaxDelay {
				wsrm.currentDelay = wsrm.config.MaxDelay
			}
		} else {
			log.Printf("WebSocket reconnection successful after %d attempts",
				atomic.LoadInt64(&wsrm.attemptCount))
			atomic.AddInt64(&wsrm.metrics.ReconnectSuccess, 1)
		}
	}
}

// calculateBackoffDelay calculates the next backoff delay with jitter.
func (wsrm *WebSocketReliabilityManager) calculateBackoffDelay() time.Duration {
	baseDelay := wsrm.currentDelay

	// Add jitter to prevent thundering herd
	if wsrm.config.JitterFactor > 0 {
		jitter := float64(baseDelay) * wsrm.config.JitterFactor * (2*rand.Float64() - 1)
		baseDelay = time.Duration(float64(baseDelay) + jitter)
	}

	if baseDelay < 0 {
		baseDelay = wsrm.config.InitialDelay
	}

	return baseDelay
}

// healthChecker periodically pings the connection to ensure it's healthy.
func (wsrm *WebSocketReliabilityManager) healthChecker() {
	defer wsrm.wg.Done()

	ticker := time.NewTicker(wsrm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wsrm.ctx.Done():
			return
		case <-ticker.C:
			if wsrm.GetStatus() == StatusConnected {
				wsrm.performHealthCheck()
			}
		}
	}
}

// performHealthCheck sends a ping and waits for pong response.
func (wsrm *WebSocketReliabilityManager) performHealthCheck() {
	wsrm.connMutex.RLock()
	conn := wsrm.conn
	wsrm.connMutex.RUnlock()

	if conn == nil {
		return
	}

	ctx, cancel := context.WithTimeout(wsrm.ctx, 5*time.Second)
	defer cancel()

	atomic.AddInt64(&wsrm.metrics.PingSent, 1)

	if err := conn.Ping(ctx); err != nil {
		log.Printf("WebSocket health check failed: %v", err)
		atomic.AddInt64(&wsrm.metrics.TimeoutCount, 1)

		// Connection appears unhealthy, trigger disconnect
		wsrm.handleConnectionError(err)
	} else {
		atomic.AddInt64(&wsrm.metrics.PongReceived, 1)
	}
}

// readPump handles reading messages from the WebSocket connection.
func (wsrm *WebSocketReliabilityManager) readPump(conn *websocket.Conn) {
	defer wsrm.wg.Done()
	defer wsrm.handleConnectionClosed()

	for {
		select {
		case <-wsrm.ctx.Done():
			return
		default:
			messageType, data, err := conn.Read(wsrm.ctx)
			if err != nil {
				if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
					log.Printf("WebSocket read error: %v", err)
					wsrm.handleConnectionError(err)
				}

				return
			}

			// Notify message callbacks
			wsrm.callbackMutex.RLock()
			callbacks := make([]MessageCallback, len(wsrm.messageCallbacks))
			copy(callbacks, wsrm.messageCallbacks)
			wsrm.callbackMutex.RUnlock()

			for _, callback := range callbacks {
				go callback(messageType, data)
			}
		}
	}
}

// handleConnectionError handles connection errors and triggers reconnection.
func (wsrm *WebSocketReliabilityManager) handleConnectionError(err error) {
	wsrm.updateError(err)
	wsrm.updateConnectionDuration()

	wsrm.connMutex.Lock()
	if wsrm.conn != nil {
		_ = wsrm.conn.Close(websocket.StatusAbnormalClosure, "Connection error")
		wsrm.conn = nil
	}
	wsrm.connMutex.Unlock()

	wsrm.setStatus(StatusDisconnected)
}

// handleConnectionClosed handles normal connection closure.
func (wsrm *WebSocketReliabilityManager) handleConnectionClosed() {
	wsrm.updateConnectionDuration()

	wsrm.connMutex.Lock()
	wsrm.conn = nil
	wsrm.connMutex.Unlock()

	if wsrm.GetStatus() != StatusOffline {
		wsrm.setStatus(StatusDisconnected)
	}
}

// setStatus updates the connection status and notifies callbacks.
func (wsrm *WebSocketReliabilityManager) setStatus(status ConnectionStatus) {
	oldStatus := ConnectionStatus(atomic.SwapInt32(&wsrm.status, int32(status)))

	if oldStatus != status {
		log.Printf("WebSocket status changed: %s -> %s", oldStatus, status)

		wsrm.callbackMutex.RLock()
		callbacks := make([]StatusCallback, len(wsrm.statusCallbacks))
		copy(callbacks, wsrm.statusCallbacks)
		wsrm.callbackMutex.RUnlock()

		for _, callback := range callbacks {
			go callback(status, nil)
		}
	}
}

// updateError categorizes and counts connection errors.
func (wsrm *WebSocketReliabilityManager) updateError(err error) {
	// Simple error categorization
	errStr := err.Error()
	if websocket.CloseStatus(err) != -1 {
		atomic.AddInt64(&wsrm.metrics.ProtocolErrors, 1)
	} else if containsAny(errStr, []string{"network", "connection", "timeout", "refused"}) {
		atomic.AddInt64(&wsrm.metrics.NetworkErrors, 1)
	} else {
		atomic.AddInt64(&wsrm.metrics.UnexpectedErrors, 1)
	}
}

// updateConnectionDuration updates connection duration metrics.
func (wsrm *WebSocketReliabilityManager) updateConnectionDuration() {
	startTime := atomic.LoadInt64(&wsrm.metrics.CurrentConnStart)
	if startTime == 0 {
		return
	}

	duration := time.Now().UnixNano() - startTime
	atomic.StoreInt64(&wsrm.metrics.LastDisconnect, time.Now().UnixNano())
	atomic.StoreInt64(&wsrm.metrics.CurrentConnStart, 0)

	// Update total connected time
	atomic.AddInt64(&wsrm.metrics.TotalConnectedTime, duration)

	// Update longest connection
	for {
		current := atomic.LoadInt64(&wsrm.metrics.LongestConnection)
		if duration <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&wsrm.metrics.LongestConnection, current, duration) {
			break
		}
	}

	// Update average connection time
	successfulConns := atomic.LoadInt64(&wsrm.metrics.SuccessfulConns)
	if successfulConns > 0 {
		totalTime := atomic.LoadInt64(&wsrm.metrics.TotalConnectedTime)
		atomic.StoreInt64(&wsrm.metrics.AverageConnection, totalTime/successfulConns)
	}
}

// queueMessage adds a message to the offline queue.
func (wsrm *WebSocketReliabilityManager) queueMessage(data []byte) {
	wsrm.queueMutex.Lock()
	defer wsrm.queueMutex.Unlock()

	// Prevent queue from growing too large
	if len(wsrm.messageQueue) >= wsrm.maxQueueSize {
		// Remove oldest message
		copy(wsrm.messageQueue, wsrm.messageQueue[1:])
		wsrm.messageQueue = wsrm.messageQueue[:len(wsrm.messageQueue)-1]
	}

	// Add new message
	messageCopy := make([]byte, len(data))
	copy(messageCopy, data)
	wsrm.messageQueue = append(wsrm.messageQueue, messageCopy)
}

// processQueuedMessages sends all queued messages when connection is restored.
func (wsrm *WebSocketReliabilityManager) processQueuedMessages() {
	wsrm.queueMutex.Lock()
	queue := make([][]byte, len(wsrm.messageQueue))
	copy(queue, wsrm.messageQueue)
	wsrm.messageQueue = wsrm.messageQueue[:0] // Clear queue
	wsrm.queueMutex.Unlock()

	for _, message := range queue {
		if err := wsrm.SendMessage(websocket.MessageText, message); err != nil {
			log.Printf("Failed to send queued message: %v", err)
			// Re-queue failed message
			wsrm.queueMessage(message)
		}
	}

	if len(queue) > 0 {
		log.Printf("Processed %d queued messages", len(queue))
	}
}

// containsAny checks if a string contains any of the provided substrings.
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}

	return false
}
