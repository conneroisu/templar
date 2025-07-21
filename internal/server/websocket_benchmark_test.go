package server

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

// BenchmarkWebSocketBroadcasting compares original vs optimized broadcasting performance
func BenchmarkWebSocketBroadcasting(b *testing.B) {
	clientCounts := []int{10, 50, 100, 500, 1000}

	for _, clientCount := range clientCounts {
		b.Run(fmt.Sprintf("Original_%d_clients", clientCount), func(b *testing.B) {
			benchmarkOriginalBroadcast(b, clientCount)
		})

		b.Run(fmt.Sprintf("Optimized_%d_clients", clientCount), func(b *testing.B) {
			benchmarkOptimizedBroadcast(b, clientCount)
		})
	}
}

// benchmarkOriginalBroadcast simulates the original broadcasting approach
func benchmarkOriginalBroadcast(b *testing.B, clientCount int) {
	// Simulate original approach with map iteration and slice allocations
	clients := make(map[*websocket.Conn]*Client)
	var clientsMutex sync.RWMutex

	// Create mock clients
	for i := 0; i < clientCount; i++ {
		client := &Client{
			send: make(chan []byte, 256),
		}
		conn := &websocket.Conn{} // Mock connection
		clients[conn] = client
	}

	message := []byte("test broadcast message for performance testing")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Original broadcasting logic (from websocket.go line 136-164)
		clientsMutex.RLock()
		var failedClients []*websocket.Conn // New allocation every broadcast!

		for conn, client := range clients {
			select {
			case client.send <- message:
			default:
				failedClients = append(failedClients, conn) // Slice growth allocations
			}
		}
		clientsMutex.RUnlock()

		// Simulate failed client cleanup
		if len(failedClients) > 0 {
			clientsMutex.Lock()
			for _, conn := range failedClients {
				if client, ok := clients[conn]; ok {
					delete(clients, conn)
					close(client.send)
				}
			}
			clientsMutex.Unlock()
		}
	}
}

// benchmarkOptimizedBroadcast tests the optimized broadcasting approach
func benchmarkOptimizedBroadcast(b *testing.B, clientCount int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewOptimizedWebSocketHub(ctx)
	defer hub.Shutdown()

	// Create optimized clients
	for i := 0; i < clientCount; i++ {
		client := &OptimizedClient{
			conn:         &websocket.Conn{}, // Mock connection
			server:       nil,               // Not needed for benchmark
			lastActivity: time.Now(),
			sendRing:     NewRingBuffer(DefaultRingBufferSize),
			priority:     ClientPriorityNormal,
			created:      time.Now(),
		}
		hub.clients.AddClient(client)
	}

	message := &BroadcastMessage{
		Data:      []byte("test broadcast message for performance testing"),
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Optimized broadcasting using pre-allocated structures
		hub.optimizedBroadcast(message)
	}
}

// BenchmarkRingBufferVsChannel compares ring buffer vs channel performance
func BenchmarkRingBufferVsChannel(b *testing.B) {
	message := []byte("test message for ring buffer vs channel comparison")

	b.Run("RingBuffer", func(b *testing.B) {
		ring := NewRingBuffer(1024)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			if !ring.Push(message) {
				// Buffer full, pop one message
				ring.Pop()
				ring.Push(message)
			}
		}
	})

	b.Run("Channel", func(b *testing.B) {
		ch := make(chan []byte, 1024)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			select {
			case ch <- message:
			default:
				// Channel full, drain one message
				<-ch
				ch <- message
			}
		}
	})
}

// BenchmarkClientPoolOperations tests client pool add/remove performance
func BenchmarkClientPoolOperations(b *testing.B) {
	pool := NewClientPool()

	b.Run("AddClient", func(b *testing.B) {
		clients := make([]*OptimizedClient, b.N)
		for i := 0; i < b.N; i++ {
			clients[i] = &OptimizedClient{
				conn:     &websocket.Conn{},
				sendRing: NewRingBuffer(DefaultRingBufferSize),
				created:  time.Now(),
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			pool.AddClient(clients[i])
		}
	})

	b.Run("RemoveClient", func(b *testing.B) {
		// Pre-populate pool
		clientIDs := make([]uint64, b.N)
		for i := 0; i < b.N; i++ {
			client := &OptimizedClient{
				conn:     &websocket.Conn{},
				sendRing: NewRingBuffer(DefaultRingBufferSize),
				created:  time.Now(),
			}
			pool.AddClient(client)
			clientIDs[i] = client.id
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			pool.RemoveClient(clientIDs[i])
		}
	})
}

// BenchmarkBroadcastPooling tests the efficiency of object pooling
func BenchmarkBroadcastPooling(b *testing.B) {
	broadcastPool := NewBroadcastPool()

	b.Run("WithPooling", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Get from pool
			slice := broadcastPool.clientSlicePool.Get().([]*OptimizedClient)
			slice = slice[:0]

			// Simulate usage
			for j := 0; j < 100; j++ {
				slice = append(slice, &OptimizedClient{})
			}

			// Return to pool
			broadcastPool.clientSlicePool.Put(slice)
		}
	})

	b.Run("WithoutPooling", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Allocate new slice every time
			slice := make([]*OptimizedClient, 0, 100)

			// Simulate usage
			for j := 0; j < 100; j++ {
				slice = append(slice, &OptimizedClient{})
			}

			// No cleanup (GC will handle it)
			_ = slice
		}
	})
}

// BenchmarkBackpressureHandling tests backpressure performance
func BenchmarkBackpressureHandling(b *testing.B) {
	backpressure := NewBackpressureManager()
	client := &OptimizedClient{
		sendRing: NewRingBuffer(64), // Small buffer for testing backpressure
		priority: ClientPriorityNormal,
	}

	// Fill buffer to trigger backpressure
	for i := 0; i < 50; i++ {
		client.sendRing.Push([]byte("filler message"))
	}

	message := &BroadcastMessage{
		Data:     []byte("test message"),
		Priority: PriorityNormal,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate shouldDropMessage logic
		queueSize := client.sendRing.Size()
		utilization := float64(queueSize) / float64(client.sendRing.size)

		if utilization < backpressure.dropThreshold {
			continue
		}

		if message.Priority >= PriorityHigh {
			continue
		}

		clientWeight := backpressure.priorityWeights[client.priority]
		if clientWeight >= 2.0 {
			continue
		}

		// Message would be dropped
	}
}

// BenchmarkMemoryUsage measures memory usage of different approaches
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("OriginalApproach", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Simulate original approach memory usage
		for i := 0; i < b.N; i++ {
			// Allocate slice for failed clients (original memory bomb)
			failedClients := make([]*websocket.Conn, 0, 100)
			for j := 0; j < 50; j++ {
				failedClients = append(failedClients, &websocket.Conn{})
			}

			// Simulate cleanup operations
			for _, conn := range failedClients {
				_ = conn // Prevent optimization
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})

	b.Run("OptimizedApproach", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Use optimized pooled approach
		pool := NewBroadcastPool()

		for i := 0; i < b.N; i++ {
			// Get pre-allocated slice from pool
			failedClients := pool.clientSlicePool.Get().([]*OptimizedClient)
			failedClients = failedClients[:0]

			// Simulate operations
			for j := 0; j < 50; j++ {
				failedClients = append(failedClients, &OptimizedClient{})
			}

			// Return to pool
			pool.clientSlicePool.Put(failedClients)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})
}

// BenchmarkConcurrentBroadcasts tests performance under concurrent load
func BenchmarkConcurrentBroadcasts(b *testing.B) {
	const (
		clientCount          = 500
		goroutineCount       = 10
		messagesPerGoroutine = 100
	)

	b.Run("Original", func(b *testing.B) {
		clients := make(map[*websocket.Conn]*Client)
		var clientsMutex sync.RWMutex

		// Create clients
		for i := 0; i < clientCount; i++ {
			client := &Client{
				send: make(chan []byte, 256),
			}
			clients[&websocket.Conn{}] = client
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup

			for g := 0; g < goroutineCount; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for m := 0; m < messagesPerGoroutine; m++ {
						message := []byte(fmt.Sprintf("concurrent message %d", m))

						clientsMutex.RLock()
						var failedClients []*websocket.Conn

						for conn, client := range clients {
							select {
							case client.send <- message:
							default:
								failedClients = append(failedClients, conn)
							}
						}
						clientsMutex.RUnlock()

						// Cleanup failed clients
						if len(failedClients) > 0 {
							clientsMutex.Lock()
							for _, conn := range failedClients {
								delete(clients, conn)
							}
							clientsMutex.Unlock()
						}
					}
				}()
			}

			wg.Wait()
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		hub := NewOptimizedWebSocketHub(ctx)
		defer hub.Shutdown()

		// Create clients
		for i := 0; i < clientCount; i++ {
			client := &OptimizedClient{
				conn:     &websocket.Conn{},
				sendRing: NewRingBuffer(DefaultRingBufferSize),
				priority: ClientPriorityNormal,
				created:  time.Now(),
			}
			hub.clients.AddClient(client)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup

			for g := 0; g < goroutineCount; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for m := 0; m < messagesPerGoroutine; m++ {
						message := &BroadcastMessage{
							Data:      []byte(fmt.Sprintf("concurrent message %d", m)),
							Priority:  PriorityNormal,
							Timestamp: time.Now(),
						}

						hub.optimizedBroadcast(message)
					}
				}()
			}

			wg.Wait()
		}
	})
}

// TestOptimizedWebSocketMetrics validates that metrics are properly tracked
func TestOptimizedWebSocketMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewOptimizedWebSocketHub(ctx)
	defer hub.Shutdown()

	// Add some clients
	clientCount := 10
	for i := 0; i < clientCount; i++ {
		client := &OptimizedClient{
			conn:     &websocket.Conn{},
			sendRing: NewRingBuffer(DefaultRingBufferSize),
			priority: ClientPriorityNormal,
			created:  time.Now(),
		}
		hub.clients.AddClient(client)
	}

	// Perform some broadcasts
	for i := 0; i < 5; i++ {
		message := &BroadcastMessage{
			Data:      []byte(fmt.Sprintf("test message %d", i)),
			Priority:  PriorityNormal,
			Timestamp: time.Now(),
		}
		hub.optimizedBroadcast(message)
	}

	// Check metrics
	metrics := hub.GetOptimizedMetrics()

	if metrics["active_connections"].(int64) != int64(clientCount) {
		t.Errorf("Expected %d active connections, got %v", clientCount, metrics["active_connections"])
	}

	if metrics["client_pool_size"].(int) != clientCount {
		t.Errorf("Expected %d client pool size, got %v", clientCount, metrics["client_pool_size"])
	}

	if metrics["broadcasts_sent"].(int64) != 5 {
		t.Errorf("Expected 5 broadcasts sent, got %v", metrics["broadcasts_sent"])
	}

	t.Logf("Metrics: %+v", metrics)
}

// TestRingBufferOperations validates ring buffer functionality
func TestRingBufferOperations(t *testing.T) {
	ring := NewRingBuffer(8) // Small size for testing

	// Test basic push/pop
	message1 := []byte("message 1")
	if !ring.Push(message1) {
		t.Error("Failed to push to empty ring buffer")
	}

	if ring.IsEmpty() {
		t.Error("Ring buffer should not be empty after push")
	}

	popped, ok := ring.Pop()
	if !ok {
		t.Error("Failed to pop from non-empty ring buffer")
	}

	if string(popped) != string(message1) {
		t.Errorf("Expected %s, got %s", string(message1), string(popped))
	}

	// Test buffer full condition (size-1 capacity to distinguish full from empty)
	for i := 0; i < 7; i++ { // Can only fill 7 out of 8 slots
		if !ring.Push([]byte(fmt.Sprintf("message %d", i))) {
			t.Errorf("Failed to push message %d to ring buffer", i)
		}
	}

	// Buffer should be full now
	if ring.Push([]byte("overflow message")) {
		t.Error("Ring buffer should reject push when full")
	}

	// Test size calculation
	if ring.Size() != 7 {
		t.Errorf("Expected ring buffer size 7, got %d", ring.Size())
	}
}
