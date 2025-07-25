package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/validation"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Maximum number of concurrent WebSocket connections
	maxConnections = 100

	// Connection timeout for idle connections
	connectionTimeout = 5 * time.Minute

	// Rate limit: maximum messages per client per minute
	maxMessagesPerMinute = 60
)

func (s *PreviewServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate origin before accepting connection
	if !s.checkOrigin(r) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Check connection limit before accepting
	s.clientsMutex.RLock()
	currentConnections := len(s.clients)
	s.clientsMutex.RUnlock()

	if currentConnections >= maxConnections {
		http.Error(w, "Too Many Connections", http.StatusTooManyRequests)
		log.Printf("WebSocket connection rejected: limit exceeded (%d/%d)", currentConnections, maxConnections)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: false, // Always verify origin
	})
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:         conn,
		send:         make(chan []byte, 256),
		server:       s,
		lastActivity: time.Now(),
		rateLimiter:  NewSlidingWindowRateLimiter(maxMessagesPerMinute, time.Minute),
	}

	// MEMORY LEAK FIX: Register client first to ensure proper cleanup if goroutines fail
	s.register <- client

	// Start goroutines after successful registration
	go client.writePump()
	go client.readPump()
}

// checkOrigin validates the request origin for security
func (s *PreviewServer) checkOrigin(r *http.Request) bool {
	// Get the origin from the request
	origin := r.Header.Get("Origin")

	// Build allowed origins list
	expectedHost := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	allowedOrigins := []string{
		expectedHost,
		fmt.Sprintf("localhost:%d", s.config.Server.Port),
		fmt.Sprintf("127.0.0.1:%d", s.config.Server.Port),
		"localhost:3000", // Common dev server
		"127.0.0.1:3000", // Common dev server
	}

	// Use centralized validation
	err := validation.ValidateOrigin(origin, allowedOrigins)
	return err == nil
}

func (s *PreviewServer) runWebSocketHub(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-s.register:
			if client == nil || client.conn == nil {
				continue
			}
			s.clientsMutex.Lock()
			if s.clients != nil {
				s.clients[client.conn] = client
				clientCount := len(s.clients)
				s.clientsMutex.Unlock()
				log.Printf("Client connected, total: %d", clientCount)
			} else {
				s.clientsMutex.Unlock()
			}

		case conn := <-s.unregister:
			if conn == nil {
				continue
			}
			s.clientsMutex.Lock()
			if s.clients != nil {
				if client, ok := s.clients[conn]; ok {
					delete(s.clients, conn)
					close(client.send)
					conn.Close(websocket.StatusNormalClosure, "")
					log.Printf("Client disconnected, total: %d", len(s.clients))
				}
			}
			s.clientsMutex.Unlock()

		case message := <-s.broadcast:
			s.clientsMutex.RLock()
			var failedClients []*websocket.Conn
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

			// Clean up failed clients outside the read lock
			if len(failedClients) > 0 {
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
		}
	}
}

// readPump pumps messages from the websocket connection
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c.conn
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Set read limit
	c.conn.SetReadLimit(maxMessageSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		// Check for connection timeout
		if time.Since(c.lastActivity) > connectionTimeout {
			log.Printf("WebSocket connection timed out due to inactivity")
			break
		}

		// Set read timeout - DON'T check rate limit before reading
		readCtx, readCancel := context.WithTimeout(ctx, pongWait)
		_, message, err := c.conn.Read(readCtx)
		readCancel()

		if err != nil {
			// Check if it's a normal closure
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure &&
				websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// SECURITY FIX: Only check rate limiting AFTER successfully receiving a message
		// This prevents attackers from triggering rate limits without sending data
		if len(message) > 0 {
			if !c.rateLimiter.IsAllowed() {
				log.Printf("WebSocket rate limit exceeded for client (sliding window)")
				c.conn.Close(websocket.StatusPolicyViolation, "Rate limit exceeded")
				break
			}
		}

		// Update activity tracking
		c.lastActivity = time.Now()
	}
}

// writePump pumps messages to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()

	for {
		select {
		case message, ok := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				cancel()
				return
			}

			if err := c.conn.Write(writeCtx, websocket.MessageText, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				cancel()
				return
			}
			cancel()

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			if err := c.conn.Ping(pingCtx); err != nil {
				cancel()
				return
			}
			cancel()
		}
	}
}
