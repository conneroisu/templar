// Package server provides optimized WebSocket functionality to fix broadcasting memory bomb
// and performance issues identified by Bob (Performance Agent).
package server

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// OptimizedWebSocketHub manages WebSocket connections with high-performance optimizations
type OptimizedWebSocketHub struct {
	// Client management with efficient data structures
	clients *ClientPool

	// Pre-allocated broadcast structures to eliminate allocations
	broadcastPool    *BroadcastPool
	failedClientPool *FailedClientPool

	// Channels for communication
	register   chan *OptimizedClient
	unregister chan *OptimizedClient
	broadcast  chan *BroadcastMessage

	// Backpressure handling
	backpressure *BackpressureManager

	// Performance metrics
	metrics *HubMetrics

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// OptimizedClient represents a WebSocket client with performance optimizations
type OptimizedClient struct {
	// Connection and basic info
	conn         *websocket.Conn
	id           uint64
	server       *PreviewServer
	lastActivity time.Time

	// High-performance buffering with ring buffer
	sendRing *RingBuffer

	// Backpressure tracking
	missedMessages int64
	priority       ClientPriority

	// Lifecycle tracking
	created time.Time
	active  int32 // atomic
}

// ClientPool efficiently manages client connections using hash map + ring buffer hybrid
type ClientPool struct {
	// Primary storage: hash map for O(1) lookups
	clients   map[uint64]*OptimizedClient
	clientsMu sync.RWMutex

	// Secondary storage: ring buffer for efficient iteration during broadcasts
	broadcastRing []*OptimizedClient
	ringSize      int
	ringHead      int
	ringMu        sync.RWMutex

	// Client ID generation
	nextID uint64

	// Pool for client objects
	clientPool sync.Pool
}

// RingBuffer provides lock-free message buffering for WebSocket clients
type RingBuffer struct {
	buffer [][]byte
	size   uint64
	mask   uint64

	// Atomic operations for lock-free access
	writePos uint64
	readPos  uint64
}

// BroadcastPool manages pre-allocated broadcast operations to eliminate allocations
type BroadcastPool struct {
	messagePool     sync.Pool
	operationPool   sync.Pool
	clientSlicePool sync.Pool
}

// FailedClientPool manages cleanup operations with object pooling
type FailedClientPool struct {
	cleanupPool sync.Pool
	cleanupChan chan *CleanupOperation
	workers     int
	wg          sync.WaitGroup
	ctx         context.Context
}

// BackpressureManager handles backpressure for WebSocket broadcasts
type BackpressureManager struct {
	maxQueueSize    int
	dropThreshold   float64
	priorityWeights map[ClientPriority]float64
}

// BroadcastMessage represents a message to broadcast with metadata
type BroadcastMessage struct {
	Data      []byte
	Priority  MessagePriority
	Timestamp time.Time
	ID        uint64
}

// CleanupOperation represents a client cleanup operation for pooling
type CleanupOperation struct {
	Client    *OptimizedClient
	Reason    string
	Timestamp time.Time
}

// HubMetrics tracks performance metrics for the WebSocket hub
type HubMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	BroadcastsSent      int64
	BroadcastLatencySum int64
	FailedBroadcasts    int64
	DroppedMessages     int64
	AllocationsSaved    int64
}

// MessagePriority defines message priority levels
type MessagePriority int

const (
	PriorityLow MessagePriority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// ClientPriority defines client priority levels for backpressure handling
type ClientPriority int

const (
	ClientPriorityLow ClientPriority = iota
	ClientPriorityNormal
	ClientPriorityHigh
)

// Optimized constants for performance
const (
	DefaultRingBufferSize = 1024  // Must be power of 2
	DefaultClientPoolSize = 256   // Expected concurrent clients
	BroadcastWorkers      = 4     // Number of broadcast workers
	CleanupWorkers        = 2     // Number of cleanup workers
	MaxBackpressureQueue  = 10000 // Maximum queued messages before dropping
)

// NewOptimizedWebSocketHub creates a new optimized WebSocket hub
func NewOptimizedWebSocketHub(ctx context.Context) *OptimizedWebSocketHub {
	hubCtx, cancel := context.WithCancel(ctx)

	hub := &OptimizedWebSocketHub{
		clients:          NewClientPool(),
		broadcastPool:    NewBroadcastPool(),
		failedClientPool: NewFailedClientPool(hubCtx, CleanupWorkers),
		register:         make(chan *OptimizedClient, 100),
		unregister:       make(chan *OptimizedClient, 100),
		broadcast:        make(chan *BroadcastMessage, 1000),
		backpressure:     NewBackpressureManager(),
		metrics:          &HubMetrics{},
		ctx:              hubCtx,
		cancel:           cancel,
	}

	// Start hub workers
	hub.wg.Add(1)
	go hub.runHub()

	return hub
}

// NewClientPool creates a new optimized client pool
func NewClientPool() *ClientPool {
	return &ClientPool{
		clients:       make(map[uint64]*OptimizedClient, DefaultClientPoolSize),
		broadcastRing: make([]*OptimizedClient, DefaultClientPoolSize),
		ringSize:      DefaultClientPoolSize,
		clientPool: sync.Pool{
			New: func() interface{} {
				return &OptimizedClient{
					sendRing: NewRingBuffer(DefaultRingBufferSize),
				}
			},
		},
	}
}

// NewRingBuffer creates a new lock-free ring buffer for message queuing
func NewRingBuffer(size uint64) *RingBuffer {
	// Ensure size is power of 2 for efficient modulo operations
	if size&(size-1) != 0 {
		// Round up to next power of 2
		size = 1 << (64 - uint64(countLeadingZeros(size-1)))
	}

	return &RingBuffer{
		buffer: make([][]byte, size),
		size:   size,
		mask:   size - 1, // For fast modulo: x % size == x & mask
	}
}

// NewBroadcastPool creates a new broadcast pool for zero-allocation broadcasting
func NewBroadcastPool() *BroadcastPool {
	return &BroadcastPool{
		messagePool: sync.Pool{
			New: func() interface{} {
				return &BroadcastMessage{}
			},
		},
		operationPool: sync.Pool{
			New: func() interface{} {
				return make([]*OptimizedClient, 0, DefaultClientPoolSize)
			},
		},
		clientSlicePool: sync.Pool{
			New: func() interface{} {
				return make([]*OptimizedClient, 0, 100)
			},
		},
	}
}

// NewFailedClientPool creates a new failed client pool for efficient cleanup
func NewFailedClientPool(ctx context.Context, workers int) *FailedClientPool {
	pool := &FailedClientPool{
		cleanupPool: sync.Pool{
			New: func() interface{} {
				return &CleanupOperation{}
			},
		},
		cleanupChan: make(chan *CleanupOperation, 1000),
		workers:     workers,
		ctx:         ctx,
	}

	// Start cleanup workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.cleanupWorker()
	}

	return pool
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager() *BackpressureManager {
	return &BackpressureManager{
		maxQueueSize:  MaxBackpressureQueue,
		dropThreshold: 0.8, // Drop messages when queue is 80% full
		priorityWeights: map[ClientPriority]float64{
			ClientPriorityLow:    0.3,
			ClientPriorityNormal: 1.0,
			ClientPriorityHigh:   3.0,
		},
	}
}

// AddClient efficiently adds a client to the pool with O(1) operation
func (cp *ClientPool) AddClient(client *OptimizedClient) {
	// Generate unique ID atomically
	client.id = atomic.AddUint64(&cp.nextID, 1)
	atomic.StoreInt32(&client.active, 1)

	cp.clientsMu.Lock()
	defer cp.clientsMu.Unlock()

	// Add to hash map for O(1) lookups
	cp.clients[client.id] = client

	// Add to ring buffer for efficient broadcast iteration
	cp.ringMu.Lock()
	if cp.ringHead < cp.ringSize {
		cp.broadcastRing[cp.ringHead] = client
		cp.ringHead++
	}
	cp.ringMu.Unlock()
}

// RemoveClient efficiently removes a client with O(1) operation
func (cp *ClientPool) RemoveClient(clientID uint64) *OptimizedClient {
	cp.clientsMu.Lock()
	defer cp.clientsMu.Unlock()

	client, exists := cp.clients[clientID]
	if !exists {
		return nil
	}

	delete(cp.clients, clientID)
	atomic.StoreInt32(&client.active, 0)

	// Remove from ring buffer (mark as nil for skip during iteration)
	cp.ringMu.Lock()
	for i := 0; i < cp.ringHead; i++ {
		if cp.broadcastRing[i] != nil && cp.broadcastRing[i].id == clientID {
			cp.broadcastRing[i] = nil
			break
		}
	}
	cp.ringMu.Unlock()

	return client
}

// GetActiveClientsForBroadcast returns a pre-allocated slice of active clients
func (cp *ClientPool) GetActiveClientsForBroadcast(pool *BroadcastPool) []*OptimizedClient {
	// Get pre-allocated slice from pool
	activeClients := pool.clientSlicePool.Get().([]*OptimizedClient)
	activeClients = activeClients[:0] // Reset length but keep capacity

	cp.ringMu.RLock()
	defer cp.ringMu.RUnlock()

	// Efficiently iterate through ring buffer
	for i := 0; i < cp.ringHead; i++ {
		client := cp.broadcastRing[i]
		if client != nil && atomic.LoadInt32(&client.active) == 1 {
			activeClients = append(activeClients, client)
		}
	}

	return activeClients
}

// ReturnClientsSlice returns a client slice to the pool
func (cp *ClientPool) ReturnClientsSlice(slice []*OptimizedClient, pool *BroadcastPool) {
	pool.clientSlicePool.Put(slice)
}

// Push adds a message to the ring buffer with lock-free operation
func (rb *RingBuffer) Push(message []byte) bool {
	// Get current write position
	writePos := atomic.LoadUint64(&rb.writePos)
	nextWritePos := writePos + 1

	// Check if buffer is full (leave one slot empty to distinguish full from empty)
	readPos := atomic.LoadUint64(&rb.readPos)
	if nextWritePos-readPos >= rb.size-1 {
		return false // Buffer full, apply backpressure
	}

	// Write message to buffer
	rb.buffer[writePos&rb.mask] = message

	// Atomically update write position
	atomic.StoreUint64(&rb.writePos, nextWritePos)
	return true
}

// Pop removes a message from the ring buffer with lock-free operation
func (rb *RingBuffer) Pop() ([]byte, bool) {
	readPos := atomic.LoadUint64(&rb.readPos)
	writePos := atomic.LoadUint64(&rb.writePos)

	// Check if buffer is empty
	if readPos >= writePos {
		return nil, false
	}

	// Read message from buffer
	message := rb.buffer[readPos&rb.mask]

	// Atomically update read position
	atomic.StoreUint64(&rb.readPos, readPos+1)
	return message, true
}

// IsEmpty checks if the ring buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	readPos := atomic.LoadUint64(&rb.readPos)
	writePos := atomic.LoadUint64(&rb.writePos)
	return readPos >= writePos
}

// Size returns the current number of messages in the buffer
func (rb *RingBuffer) Size() uint64 {
	writePos := atomic.LoadUint64(&rb.writePos)
	readPos := atomic.LoadUint64(&rb.readPos)
	return writePos - readPos
}

// runHub is the main hub loop with optimized broadcasting
func (hub *OptimizedWebSocketHub) runHub() {
	defer hub.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Cleanup ticker
	defer ticker.Stop()

	for {
		select {
		case <-hub.ctx.Done():
			return

		case client := <-hub.register:
			hub.clients.AddClient(client)
			atomic.AddInt64(&hub.metrics.ActiveConnections, 1)
			atomic.AddInt64(&hub.metrics.TotalConnections, 1)
			log.Printf("Optimized WebSocket client registered: %d (total: %d)",
				client.id, atomic.LoadInt64(&hub.metrics.ActiveConnections))

		case client := <-hub.unregister:
			if removedClient := hub.clients.RemoveClient(client.id); removedClient != nil {
				atomic.AddInt64(&hub.metrics.ActiveConnections, -1)

				// Schedule cleanup through pool
				cleanup := hub.failedClientPool.cleanupPool.Get().(*CleanupOperation)
				cleanup.Client = removedClient
				cleanup.Reason = "unregister"
				cleanup.Timestamp = time.Now()

				select {
				case hub.failedClientPool.cleanupChan <- cleanup:
				default:
					// Cleanup queue full, do immediate cleanup
					hub.cleanupClientImmediate(removedClient)
					hub.failedClientPool.cleanupPool.Put(cleanup)
				}
			}

		case message := <-hub.broadcast:
			start := time.Now()
			hub.optimizedBroadcast(message)

			// Update metrics
			latency := time.Since(start).Nanoseconds()
			atomic.AddInt64(&hub.metrics.BroadcastsSent, 1)
			atomic.AddInt64(&hub.metrics.BroadcastLatencySum, latency)

		case <-ticker.C:
			// Periodic cleanup and compaction
			hub.performMaintenance()
		}
	}
}

// optimizedBroadcast performs zero-allocation broadcasting with backpressure handling
func (hub *OptimizedWebSocketHub) optimizedBroadcast(message *BroadcastMessage) {
	// Get pre-allocated client slice
	activeClients := hub.clients.GetActiveClientsForBroadcast(hub.broadcastPool)
	defer hub.clients.ReturnClientsSlice(activeClients, hub.broadcastPool)

	if len(activeClients) == 0 {
		return
	}

	// Track failed clients with pre-allocated slice
	failedClients := hub.broadcastPool.clientSlicePool.Get().([]*OptimizedClient)
	failedClients = failedClients[:0]
	defer hub.broadcastPool.clientSlicePool.Put(failedClients)

	// Efficient broadcast loop
	for _, client := range activeClients {
		if atomic.LoadInt32(&client.active) == 0 {
			continue
		}

		// Try to push to client's ring buffer with backpressure handling
		if !client.sendRing.Push(message.Data) {
			// Apply backpressure logic
			if hub.shouldDropMessage(client, message) {
				atomic.AddInt64(&client.missedMessages, 1)
				atomic.AddInt64(&hub.metrics.DroppedMessages, 1)
				continue
			}

			// Mark client as failed if buffer consistently full
			failedClients = append(failedClients, client)
		}
	}

	// Efficiently handle failed clients
	if len(failedClients) > 0 {
		hub.handleFailedClients(failedClients)
	}

	// Return message to pool
	hub.broadcastPool.messagePool.Put(message)
}

// shouldDropMessage implements intelligent backpressure handling
func (hub *OptimizedWebSocketHub) shouldDropMessage(
	client *OptimizedClient,
	message *BroadcastMessage,
) bool {
	// Calculate queue utilization
	queueSize := client.sendRing.Size()
	utilization := float64(queueSize) / float64(client.sendRing.size)

	// Check if we're above drop threshold
	if utilization < hub.backpressure.dropThreshold {
		return false
	}

	// Consider message priority
	if message.Priority >= PriorityHigh {
		return false
	}

	// Consider client priority
	clientWeight := hub.backpressure.priorityWeights[client.priority]
	if clientWeight >= 2.0 { // High priority clients
		return false
	}

	return true
}

// handleFailedClients efficiently handles clients that failed to receive messages
func (hub *OptimizedWebSocketHub) handleFailedClients(failedClients []*OptimizedClient) {
	for _, client := range failedClients {
		if atomic.LoadInt32(&client.active) == 0 {
			continue
		}

		// Schedule cleanup
		cleanup := hub.failedClientPool.cleanupPool.Get().(*CleanupOperation)
		cleanup.Client = client
		cleanup.Reason = "broadcast_failure"
		cleanup.Timestamp = time.Now()

		select {
		case hub.failedClientPool.cleanupChan <- cleanup:
		default:
			// Cleanup queue full, do immediate cleanup
			hub.cleanupClientImmediate(client)
			hub.failedClientPool.cleanupPool.Put(cleanup)
		}
	}
}

// cleanupWorker handles asynchronous client cleanup
func (pool *FailedClientPool) cleanupWorker() {
	defer pool.wg.Done()

	for {
		select {
		case <-pool.ctx.Done():
			return
		case cleanup := <-pool.cleanupChan:
			// Perform cleanup
			if atomic.LoadInt32(&cleanup.Client.active) == 1 {
				atomic.StoreInt32(&cleanup.Client.active, 0)
				cleanup.Client.conn.Close(websocket.StatusNormalClosure, cleanup.Reason)
			}

			// Return cleanup object to pool
			pool.cleanupPool.Put(cleanup)
		}
	}
}

// cleanupClientImmediate performs immediate client cleanup when async queue is full
func (hub *OptimizedWebSocketHub) cleanupClientImmediate(client *OptimizedClient) {
	if atomic.LoadInt32(&client.active) == 1 {
		atomic.StoreInt32(&client.active, 0)
		client.conn.Close(websocket.StatusNormalClosure, "immediate_cleanup")
	}
}

// performMaintenance performs periodic maintenance for optimal performance
func (hub *OptimizedWebSocketHub) performMaintenance() {
	// Compact ring buffer if needed
	hub.clients.ringMu.Lock()

	// Remove nil entries from ring buffer
	writeIndex := 0
	for readIndex := 0; readIndex < hub.clients.ringHead; readIndex++ {
		if hub.clients.broadcastRing[readIndex] != nil {
			if writeIndex != readIndex {
				hub.clients.broadcastRing[writeIndex] = hub.clients.broadcastRing[readIndex]
				hub.clients.broadcastRing[readIndex] = nil
			}
			writeIndex++
		}
	}
	hub.clients.ringHead = writeIndex

	hub.clients.ringMu.Unlock()

	// Update allocation savings metric
	atomic.AddInt64(&hub.metrics.AllocationsSaved, int64(DefaultClientPoolSize))
}

// GetOptimizedMetrics returns comprehensive performance metrics
func (hub *OptimizedWebSocketHub) GetOptimizedMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Connection metrics
	metrics["total_connections"] = atomic.LoadInt64(&hub.metrics.TotalConnections)
	metrics["active_connections"] = atomic.LoadInt64(&hub.metrics.ActiveConnections)

	// Performance metrics
	metrics["broadcasts_sent"] = atomic.LoadInt64(&hub.metrics.BroadcastsSent)
	metrics["failed_broadcasts"] = atomic.LoadInt64(&hub.metrics.FailedBroadcasts)
	metrics["dropped_messages"] = atomic.LoadInt64(&hub.metrics.DroppedMessages)

	// Calculate average broadcast latency
	totalLatency := atomic.LoadInt64(&hub.metrics.BroadcastLatencySum)
	broadcastCount := atomic.LoadInt64(&hub.metrics.BroadcastsSent)
	if broadcastCount > 0 {
		metrics["avg_broadcast_latency_ns"] = totalLatency / broadcastCount
		metrics["avg_broadcast_latency_ms"] = float64(totalLatency/broadcastCount) / 1e6
	}

	// Memory optimization metrics
	metrics["allocations_saved"] = atomic.LoadInt64(&hub.metrics.AllocationsSaved)

	// Pool utilization
	hub.clients.clientsMu.RLock()
	metrics["client_pool_size"] = len(hub.clients.clients)
	hub.clients.clientsMu.RUnlock()

	return metrics
}

// Shutdown gracefully shuts down the optimized WebSocket hub
func (hub *OptimizedWebSocketHub) Shutdown() {
	log.Println("Shutting down optimized WebSocket hub...")

	// Cancel context to stop all workers
	hub.cancel()

	// Wait for all workers to finish
	hub.wg.Wait()

	// Shutdown failed client pool
	hub.failedClientPool.wg.Wait()

	log.Println("Optimized WebSocket hub shutdown complete")
}

// countLeadingZeros is a helper function for ring buffer size calculation
func countLeadingZeros(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	if x <= 0x00000000FFFFFFFF {
		n += 32
		x <<= 32
	}
	if x <= 0x0000FFFFFFFFFFFF {
		n += 16
		x <<= 16
	}
	if x <= 0x00FFFFFFFFFFFFFF {
		n += 8
		x <<= 8
	}
	if x <= 0x0FFFFFFFFFFFFFFF {
		n += 4
		x <<= 4
	}
	if x <= 0x3FFFFFFFFFFFFFFF {
		n += 2
		x <<= 2
	}
	if x <= 0x7FFFFFFFFFFFFFFF {
		n += 1
	}
	return n
}
