// Package websocket provides enhanced WebSocket functionality with performance optimizations
// for real-time communication in the Templar development server.
package websocket

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// WebSocketEnhancements provides performance improvements for WebSocket management
type WebSocketEnhancements struct {
	// Per-IP connection tracking for rate limiting
	ipConnections sync.Map // map[string]*IPConnectionTracker

	// Connection metrics
	totalConnections    int64
	rejectedConnections int64

	// Async cleanup
	cleanupQueue chan *websocket.Conn
	cleanupWg    sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc

	// Configuration
	maxConnectionsPerIP  int
	maxMessagesPerMinute int
	cleanupWorkers       int
}

// EnhancedClient extends the basic Client with performance tracking
type EnhancedClient struct {
	*Client
	ip           string
	id           string
	created      time.Time
	pingFailures int
}

// IPConnectionTracker tracks connections and rate limiting per IP
type IPConnectionTracker struct {
	connections  map[*websocket.Conn]struct{}
	count        int
	messageCount int
	rateWindow   time.Time
	mutex        sync.RWMutex
}

// NewWebSocketEnhancements creates enhanced WebSocket management
func NewWebSocketEnhancements() *WebSocketEnhancements {
	ctx, cancel := context.WithCancel(context.Background())

	enhancements := &WebSocketEnhancements{
		cleanupQueue:         make(chan *websocket.Conn, 100),
		maxConnectionsPerIP:  20,  // Per-IP connection limit
		maxMessagesPerMinute: 120, // Increased from 60
		cleanupWorkers:       4,   // Async cleanup workers
		ctx:                  ctx,
		cancel:               cancel,
	}

	// Start cleanup workers
	for i := 0; i < enhancements.cleanupWorkers; i++ {
		enhancements.cleanupWg.Add(1)
		go enhancements.cleanupWorker()
	}

	return enhancements
}

// Enhanced WebSocket handler that integrates with existing server
func (s *PreviewServer) handleWebSocketEnhanced(w http.ResponseWriter, r *http.Request) {
	// Initialize enhancements if not already done
	if s.enhancements == nil {
		s.enhancements = NewWebSocketEnhancements()
	}

	// Validate origin using centralized validation
	if !s.checkOrigin(r) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		atomic.AddInt64(&s.enhancements.rejectedConnections, 1)
		return
	}

	// Extract client IP efficiently
	clientIP := s.getClientIPEnhanced(r)

	// Check per-IP connection limit
	if !s.enhancements.checkIPLimit(clientIP) {
		http.Error(w, "Too Many Connections from IP", http.StatusTooManyRequests)
		atomic.AddInt64(&s.enhancements.rejectedConnections, 1)
		log.Printf("WebSocket connection rejected: IP limit exceeded for %s", clientIP)
		return
	}

	// Check global connection limit with existing logic
	s.clientsMutex.RLock()
	currentConnections := len(s.clients)
	s.clientsMutex.RUnlock()

	if currentConnections >= maxConnections {
		http.Error(w, "Too Many Connections", http.StatusTooManyRequests)
		atomic.AddInt64(&s.enhancements.rejectedConnections, 1)
		log.Printf("WebSocket connection rejected: global limit exceeded (%d/%d)", currentConnections, maxConnections)
		return
	}

	// Accept WebSocket connection with enhanced options
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: false,
		CompressionMode:    websocket.CompressionContextTakeover, // Enable compression for better performance
	})
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		atomic.AddInt64(&s.enhancements.rejectedConnections, 1)
		return
	}

	// Create enhanced client
	enhancedClient := &EnhancedClient{
		Client: &Client{
			conn:         conn,
			send:         make(chan []byte, 512), // Doubled buffer size for better performance
			server:       s,
			lastActivity: time.Now(),
			rateLimiter:  NewSlidingWindowRateLimiter(maxMessagesPerMinute, time.Minute),
		},
		ip:      clientIP,
		id:      fmt.Sprintf("%s_%d", clientIP, time.Now().UnixNano()),
		created: time.Now(),
	}

	// Track IP connection
	s.enhancements.trackIPConnection(clientIP, conn)

	// Start enhanced goroutines
	go s.readPumpEnhanced(enhancedClient)
	go s.writePumpEnhanced(enhancedClient)

	// Register with existing system
	s.register <- enhancedClient.Client

	atomic.AddInt64(&s.enhancements.totalConnections, 1)
	log.Printf("Enhanced WebSocket client connected: %s (total: %d)", enhancedClient.id, currentConnections+1)
}

// Enhanced read pump with better performance and monitoring
func (s *PreviewServer) readPumpEnhanced(client *EnhancedClient) {
	defer func() {
		// MEMORY LEAK FIX: Always untrack IP connection first
		if s.enhancements != nil {
			s.enhancements.untrackIPConnection(client.ip, client.conn)

			select {
			case s.enhancements.cleanupQueue <- client.conn:
			default:
				// Fallback to original cleanup
				s.unregister <- client.conn
			}
		} else {
			s.unregister <- client.conn
		}
		client.conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Set enhanced read limit
	client.conn.SetReadLimit(1024) // Doubled from 512

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get IP tracker for rate limiting
	var ipTracker *IPConnectionTracker
	if s.enhancements != nil {
		if trackerInterface, ok := s.enhancements.ipConnections.Load(client.ip); ok {
			ipTracker = trackerInterface.(*IPConnectionTracker)
		}
	}

	for {
		// Enhanced connection timeout check
		if time.Since(client.lastActivity) > connectionTimeout {
			log.Printf("Enhanced WebSocket connection %s timed out due to inactivity", client.id)
			break
		}

		// Enhanced read with context timeout - DON'T check rate limit before reading
		readCtx, readCancel := context.WithTimeout(ctx, pongWait)
		_, message, err := client.conn.Read(readCtx)
		readCancel()

		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure &&
				websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Printf("Enhanced WebSocket read error for client %s: %v", client.id, err)
			}
			break
		}

		// SECURITY FIX: Only apply rate limiting AFTER successfully receiving a message
		// This prevents attackers from triggering rate limits without sending data
		if len(message) > 0 {
			// Enhanced rate limiting
			// Use interface-based rate limiting instead of manual tracking

			// Check enhanced rate limits
			maxMessages := maxMessagesPerMinute
			if s.enhancements != nil {
				maxMessages = s.enhancements.maxMessagesPerMinute
			}

			if !client.rateLimiter.IsAllowed() {
				log.Printf("Enhanced WebSocket rate limit exceeded for client %s", client.id)
				client.conn.Close(websocket.StatusPolicyViolation, "Rate limit exceeded")
				break
			}

			// Check per-IP rate limiting if available
			if ipTracker != nil {
				ipTracker.mutex.RLock()
				ipRateExceeded := ipTracker.messageCount >= maxMessages
				ipTracker.mutex.RUnlock()

				if ipRateExceeded {
					log.Printf("Enhanced WebSocket IP rate limit exceeded for %s", client.ip)
					client.conn.Close(websocket.StatusPolicyViolation, "IP rate limit exceeded")
					break
				}

				// Increment per-IP message count for actual messages
				ipTracker.mutex.Lock()
				ipTracker.messageCount++
				// Note: lastMessage field not available in current IPConnectionTracker
				ipTracker.mutex.Unlock()
			}
		}

		// Update activity tracking
		now := time.Now()
		client.lastActivity = now
		// TODO: Re-implement message counting with new structure

		// Update IP tracker if available
		if ipTracker != nil {
			ipTracker.mutex.Lock()
			if now.Sub(ipTracker.rateWindow) >= time.Minute {
				ipTracker.messageCount = 0
				ipTracker.rateWindow = now
			}
			ipTracker.messageCount++
			ipTracker.mutex.Unlock()
		}
	}
}

// Enhanced write pump with better ping handling
func (s *PreviewServer) writePumpEnhanced(client *EnhancedClient) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		// MEMORY LEAK FIX: Always untrack IP connection first
		if s.enhancements != nil {
			s.enhancements.untrackIPConnection(client.ip, client.conn)

			select {
			case s.enhancements.cleanupQueue <- client.conn:
			default:
				s.unregister <- client.conn
			}
		} else {
			s.unregister <- client.conn
		}
		client.conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()

	for {
		select {
		case message, ok := <-client.send:
			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			if !ok {
				client.conn.Close(websocket.StatusNormalClosure, "")
				cancel()
				return
			}

			if err := client.conn.Write(writeCtx, websocket.MessageText, message); err != nil {
				log.Printf("Enhanced WebSocket write error for client %s: %v", client.id, err)
				cancel()
				return
			}
			cancel()

			client.lastActivity = time.Now()

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			if err := client.conn.Ping(pingCtx); err != nil {
				cancel()
				client.pingFailures++

				// Enhanced ping failure handling
				if client.pingFailures >= 3 {
					log.Printf("Enhanced WebSocket client %s failed ping test %d times, disconnecting",
						client.id, client.pingFailures)
					return
				}

				log.Printf("Enhanced WebSocket ping failed for client %s (failure %d/3): %v",
					client.id, client.pingFailures, err)
			} else {
				client.pingFailures = 0
				cancel()
			}
		}
	}
}

// Enhanced broadcast that works with existing system
func (s *PreviewServer) broadcastEnhanced(message []byte) {
	// Get current time for metrics
	start := time.Now()

	// Use existing broadcast logic but with optimizations
	s.clientsMutex.RLock()
	var failedClients []*websocket.Conn
	clientCount := len(s.clients)

	if s.clients != nil {
		for conn, client := range s.clients {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, mark for removal
				failedClients = append(failedClients, conn)
			}
		}
	}
	s.clientsMutex.RUnlock()

	// Enhanced async cleanup of failed clients
	if len(failedClients) > 0 && s.enhancements != nil {
		// Use async cleanup for better performance
		for _, conn := range failedClients {
			select {
			case s.enhancements.cleanupQueue <- conn:
			default:
				// Fallback to sync cleanup
				s.clientsMutex.Lock()
				if s.clients != nil {
					if client, ok := s.clients[conn]; ok {
						delete(s.clients, conn)
						close(client.send)
						conn.Close(websocket.StatusNormalClosure, "")
					}
				}
				s.clientsMutex.Unlock()
			}
		}
	} else if len(failedClients) > 0 {
		// Fallback to original cleanup
		s.clientsMutex.Lock()
		if s.clients != nil {
			for _, conn := range failedClients {
				if client, ok := s.clients[conn]; ok {
					delete(s.clients, conn)
					close(client.send)
					conn.Close(websocket.StatusNormalClosure, "")
				}
			}
		}
		s.clientsMutex.Unlock()
	}

	// Log performance metrics
	duration := time.Since(start)
	log.Printf("Enhanced broadcast to %d clients completed in %v (failed: %d)",
		clientCount, duration, len(failedClients))
}

// checkIPLimit checks if an IP can accept new connections
func (enhancements *WebSocketEnhancements) checkIPLimit(ip string) bool {
	trackerInterface, _ := enhancements.ipConnections.LoadOrStore(ip, &IPConnectionTracker{
		connections: make(map[*websocket.Conn]struct{}),
		rateWindow:  time.Now(),
	})

	tracker := trackerInterface.(*IPConnectionTracker)
	tracker.mutex.RLock()
	count := tracker.count
	tracker.mutex.RUnlock()

	return count < enhancements.maxConnectionsPerIP
}

// trackIPConnection tracks a connection for an IP
func (enhancements *WebSocketEnhancements) trackIPConnection(ip string, conn *websocket.Conn) {
	trackerInterface, _ := enhancements.ipConnections.LoadOrStore(ip, &IPConnectionTracker{
		connections: make(map[*websocket.Conn]struct{}),
		rateWindow:  time.Now(),
	})

	tracker := trackerInterface.(*IPConnectionTracker)
	tracker.mutex.Lock()
	tracker.connections[conn] = struct{}{}
	tracker.count++
	tracker.mutex.Unlock()
}

// untrackIPConnection removes tracking for an IP connection
func (enhancements *WebSocketEnhancements) untrackIPConnection(ip string, conn *websocket.Conn) {
	if trackerInterface, ok := enhancements.ipConnections.Load(ip); ok {
		tracker := trackerInterface.(*IPConnectionTracker)
		tracker.mutex.Lock()
		delete(tracker.connections, conn)
		tracker.count--

		// Clean up empty tracker
		if tracker.count == 0 {
			tracker.mutex.Unlock()
			enhancements.ipConnections.Delete(ip)
		} else {
			tracker.mutex.Unlock()
		}
	}
}

// cleanupWorker handles asynchronous client cleanup
func (enhancements *WebSocketEnhancements) cleanupWorker() {
	defer enhancements.cleanupWg.Done()

	// MEMORY LEAK FIX: Add timeout to prevent indefinite blocking
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-enhancements.ctx.Done():
			return
		case <-timeout.C:
			// Prevent indefinite blocking - worker will exit after timeout
			return
		case conn := <-enhancements.cleanupQueue:
			// Reset timeout when we get work
			if !timeout.Stop() {
				<-timeout.C
			}
			timeout.Reset(30 * time.Second)

			// This would normally integrate with server's unregister channel
			// For now, just close the connection
			conn.Close(websocket.StatusNormalClosure, "")
		}
	}
}

// getClientIPEnhanced efficiently extracts client IP
func (s *PreviewServer) getClientIPEnhanced(r *http.Request) string {
	// Check X-Forwarded-For header first (most common in production)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if commaIdx := strings.Index(xff, ","); commaIdx > 0 {
			return strings.TrimSpace(xff[:commaIdx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr with efficient parsing
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// GetEnhancedMetrics returns enhanced WebSocket metrics
func (s *PreviewServer) GetEnhancedMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	if s.enhancements != nil {
		metrics["total_connections"] = atomic.LoadInt64(&s.enhancements.totalConnections)
		metrics["rejected_connections"] = atomic.LoadInt64(&s.enhancements.rejectedConnections)
		metrics["max_connections_per_ip"] = s.enhancements.maxConnectionsPerIP
		metrics["max_messages_per_minute"] = s.enhancements.maxMessagesPerMinute
		metrics["cleanup_workers"] = s.enhancements.cleanupWorkers

		// Count current IP connections
		ipCount := 0
		s.enhancements.ipConnections.Range(func(key, value interface{}) bool {
			ipCount++
			return true
		})
		metrics["tracked_ips"] = ipCount
	}

	// Add existing metrics
	s.clientsMutex.RLock()
	metrics["current_connections"] = len(s.clients)
	s.clientsMutex.RUnlock()

	return metrics
}

// ShutdownEnhancements gracefully shuts down enhanced WebSocket features
func (s *PreviewServer) ShutdownEnhancements() {
	if s.enhancements == nil {
		return
	}

	log.Println("Shutting down WebSocket enhancements...")

	// Cancel context to stop workers
	s.enhancements.cancel()

	// Wait for cleanup workers to finish
	s.enhancements.cleanupWg.Wait()

	// Close cleanup queue
	close(s.enhancements.cleanupQueue)

	log.Println("WebSocket enhancements shutdown complete")
}
